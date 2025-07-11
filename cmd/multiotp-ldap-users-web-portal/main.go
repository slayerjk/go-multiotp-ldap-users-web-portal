package main

import (
	"crypto/tls"
	"database/sql"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	"github.com/slayerjk/go-vafswork"

	_ "github.com/go-sql-driver/mysql"

	dataembed "github.com/slayerjk/go-multiotp-ldap-users-web-portal/data"
)

const appName = "OTP-Portal"

type application struct {
	logger          *slog.Logger
	templateCache   map[string]*template.Template
	formDecoder     *form.Decoder
	sessionManager  *scs.SessionManager
	multiOTPBinPath *string
	// domain data
	userDomainFQDN       string
	userDomainBaseDN     string
	qrDomainFQDN         string
	qrDomainBaseDN       string
	qrDomainBindUser     string
	qrDomainBindUserPass string
	mfaUrl               string
	mfaTriggerUser       string
	mfaTriggerUserPass   string
	lang                 *string
	secondFactorOn       *bool
}

type AppData struct {
	UserDomainFQDN       string `json:"userDomainFQDN"`
	UserDomainBaseDN     string `json:"userDomainBaseDN"`
	QrDomainFQDN         string `json:"qrDomainFQDN"`
	QrDomainBaseDN       string `json:"qrDomainBaseDN"`
	QrDomainBindUser     string `json:"qrDomainBindUser"`
	QrDomainBindUserPass string `json:"qrDomainBindUserPass"`
	DbUser               string `json:"dbUser"`
	DbPass               string `json:"dbPass"`
	MfaUrl               string `json:"mfaUrl"`
	MfaTriggerUser       string `json:"mfaTriggerUser"`
	MfaTriggerUserPass   string `json:"mfaTriggerUserPass"`
}

func main() {
	var (
		workDir              string = vafswork.GetExePath()
		logsPathDefault      string = workDir + "/logs" + "_" + appName
		tlsCertDefault       string = workDir + "/tls" + "/" + "cert.pem"
		tlsKeyDefault        string = workDir + "/tls" + "/" + "key.pem"
		userDomainFQDN       string
		userDomainBaseDN     string
		qrDomainFQDN         string
		qrDomainBaseDN       string
		qrDomainBindUser     string
		qrDomainBindUserPass string
		dbUser               string
		dbPass               string
		appData              AppData
		mfaUrl               string
		mfaTriggerUser       string
		mfaTriggerUserPass   string
	)

	// setting flags
	logsDir := flag.String("log-dir", logsPathDefault, "set custom log dir")
	logsToKeep := flag.Int("keep-logs", 30, "set number of logs to keep after rotation")
	addr := flag.String("addr", ":3000", "HTTP server address, ex. ':3000' for localhost:3000")
	tlsCert := flag.String("tls-cert", tlsCertDefault, "full path to tls Cert file")
	tlsKey := flag.String("tls-key", tlsKeyDefault, "full path to tls Key file")
	dbName := flag.String("db", "otpportal", "MySQL db name")
	multiOTPBinPath := flag.String("m", "c:/MultiOTP/windows/multiotp.exe", "Full path to MulitOTP binary")
	lang := flag.String("lang", "ru", "Set pages languages('ru'/'en' only)")
	dataFileOn := flag.Bool("df", false, "Use embed dataFile(placed in 'data/data.json') instead of ENV vars")
	secondFactorOn := flag.Bool("2fa", false, "Use (PrivacyIdea API) provider for second factor auth")

	flag.Usage = func() {
		fmt.Println("MultiOTP Web Portal for LDAP Users")
		fmt.Println("Version = 0.3.4")
		// fmt.Println("Usage: <app> [-opt] ...")
		fmt.Println("Flags:")
		flag.PrintDefaults()
	}
	flag.Parse()

	// dataFile flag is OFF(default)
	if !*dataFileOn {
		// checking Domain data OS env
		userDomainFQDN = os.Getenv("USER_DOM_FQDN")
		if len(userDomainFQDN) == 0 {
			fmt.Println("USER_DOM_FQDN not found or empty in OS env! exiting")
			os.Exit(1)
		}

		userDomainBaseDN = os.Getenv("USER_DOM_BASE")
		if len(userDomainBaseDN) == 0 {
			fmt.Println("USER_DOM_BASE not found or empty in OS env! exiting")
			os.Exit(1)
		}

		qrDomainFQDN = os.Getenv("QR_DOM_FQDN")
		if len(qrDomainFQDN) == 0 {
			fmt.Println("QR_DOM_FQDN not found or empty in OS env! exiting")
			os.Exit(1)
		}

		qrDomainBaseDN = os.Getenv("QR_DOM_BASE")
		if len(qrDomainBaseDN) == 0 {
			fmt.Println("QR_DOM_BASE not found or empty in OS env! exiting")
			os.Exit(1)
		}

		qrDomainBindUser = os.Getenv("QR_DOM_BIND_USER")
		if len(qrDomainBindUser) == 0 {
			fmt.Println("QR_DOM_BIND_USER not found or empty in OS env! exiting")
			os.Exit(1)
		}

		qrDomainBindUserPass = os.Getenv("QR_DOM_BIND_USER_PASS")
		if len(qrDomainBindUserPass) == 0 {
			fmt.Println("QR_DOM_BIND_USER_PASS not found or empty in OS env! exiting")
			os.Exit(1)
		}

		// checking OS env exists for OTP_DB_USR & OTP_DB_PASS
		dbUser = os.Getenv("OTP_DB_USR")
		dbPass = os.Getenv("OTP_DB_PASS")
		if len(dbUser) == 0 || len(dbPass) == 0 {
			fmt.Println("OTP_DB_USR and/or OTP_DB_PASS not found in OS env! exiting")
			os.Exit(1)
		}

		// checking mfa ENV vars
		if *secondFactorOn {
			mfaUrl = os.Getenv("2FA_URL")
			mfaTriggerUser = os.Getenv("2FA_TRIGGER_USER")
			mfaTriggerUserPass = os.Getenv("2FA_TRIGGER_PASS")

			if len(mfaUrl) == 0 || len(mfaTriggerUser) == 0 || len(mfaTriggerUserPass) == 0 {
				fmt.Println("one or serveral mfa ENV vars doesn't exist or empty! exiting")
				os.Exit(1)
			}
		}

		// if dataFile flag is ON use embed data.json
	} else {
		err := json.Unmarshal(dataembed.DataFileBytes, &appData)
		if err != nil {
			fmt.Printf("can't process data file:\n\t%v", err)
			os.Exit(1)
		}

		userDomainFQDN = appData.UserDomainFQDN
		userDomainBaseDN = appData.UserDomainBaseDN
		qrDomainFQDN = appData.QrDomainFQDN
		qrDomainBaseDN = appData.QrDomainBaseDN
		qrDomainBindUser = appData.QrDomainBindUser
		qrDomainBindUserPass = appData.QrDomainBindUserPass
		dbUser = appData.DbUser
		dbPass = appData.DbPass

		// checking mfa ENV vars
		if *secondFactorOn {
			mfaUrl = appData.MfaUrl
			mfaTriggerUser = appData.MfaTriggerUser
			mfaTriggerUserPass = appData.MfaTriggerUserPass

			if len(mfaUrl) == 0 || len(mfaTriggerUser) == 0 || len(mfaTriggerUserPass) == 0 {
				fmt.Println("one or serveral mfa data not found in data file or empty! exiting")
				os.Exit(1)
			}
		}
	}

	// create logs dir
	if err := os.MkdirAll(*logsDir, os.ModePerm); err != nil {
		fmt.Fprintf(os.Stdout, "failed to create log dir %s:\n\t%v", *logsDir, err)
		os.Exit(1)
	}

	// set current date
	dateNow := time.Now().Format("02.01.2006")

	// create log file
	logFilePath := fmt.Sprintf("%s/%s_%s.log", *logsDir, appName, dateNow)
	// open logFile in Append mode
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Fprintf(os.Stdout, "failed to open created log file %s:\n\t%v", logFilePath, err)
		os.Exit(1)
	}
	defer logFile.Close()

	// set slog.Logger
	logger := slog.New(slog.NewTextHandler(logFile, nil))

	// setting dsn using OTP_DB_USR, OTP_DB_PASS and dbName
	dsn := fmt.Sprintf("%s:%s@/%s?parseTime=true", dbUser, dbPass, *dbName)
	// try to open db
	db, err := openDB(dsn)
	if err != nil {
		logger.Error(err.Error())
	}
	defer db.Close()

	// Initialize a new template cache...
	templateCache, err := newTemplateCache(*lang)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// Init session manager
	sessionManager := scs.New()
	sessionManager.Store = mysqlstore.New(db)
	sessionManager.Lifetime = 30 * time.Minute
	sessionManager.Cookie.Secure = true

	// Initialize a decoder instance...
	formDecoder := form.NewDecoder()

	// define app
	app := &application{
		logger:               logger,
		templateCache:        templateCache,
		formDecoder:          formDecoder,
		sessionManager:       sessionManager,
		multiOTPBinPath:      multiOTPBinPath,
		userDomainFQDN:       userDomainFQDN,
		userDomainBaseDN:     userDomainBaseDN,
		qrDomainFQDN:         qrDomainFQDN,
		qrDomainBaseDN:       qrDomainBaseDN,
		qrDomainBindUser:     qrDomainBindUser,
		qrDomainBindUserPass: qrDomainBindUserPass,
		mfaUrl:               mfaUrl,
		mfaTriggerUser:       mfaTriggerUser,
		mfaTriggerUserPass:   mfaTriggerUserPass,
		secondFactorOn:       secondFactorOn,
		lang:                 lang,
	}

	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
	}

	srv := &http.Server{
		Addr:         *addr,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
		TLSConfig:    tlsConfig,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		Handler:      app.routes(),
	}

	// starting program notification
	logger.Info("Program started", "appName", appName)

	// rotate log first
	logger.Info("Log rotation first", "logsDir", *logsDir, "logs to keep", *logsToKeep)
	if err := vafswork.RotateFilesByMtime(*logsDir, *logsToKeep); err != nil {
		fmt.Fprintf(os.Stdout, "failed to rotate logs:\n\t%v", err)
	}

	// starting http srv info
	logger.Info("starting server", slog.Any("addr", *addr))

	// starting HTTP server
	err = srv.ListenAndServeTLS(*tlsCert, *tlsKey)
	logger.Error(err.Error())
	os.Exit(1)

}

// The openDB() function wraps sql.Open() and returns a sql.DB connection pool
// for a given DSN.
func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
