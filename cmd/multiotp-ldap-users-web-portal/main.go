package main

import (
	"crypto/tls"
	"database/sql"
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
}

func main() {
	var (
		workDir         string = vafswork.GetExePath()
		logsPathDefault string = workDir + "/logs" + "_" + appName
		tlsCertDefault  string = workDir + "/tls" + "/" + "cert.pem"
		tlsKeyDefault   string = workDir + "/tls" + "/" + "key.pem"
	)

	// checking Domain data OS env
	userDomainFQDN, ok := os.LookupEnv("USER_DOM_FQDN")
	if !ok {
		fmt.Println("failed to find USER_DOM_FQDN env var")
		os.Exit(1)
	}
	userDomainBaseDN, ok := os.LookupEnv("USER_DOM_BASE")
	if !ok {
		fmt.Println("failed to find USER_DOM_BASE env var")
		os.Exit(1)
	}
	qrDomainFQDN, ok := os.LookupEnv("QR_DOM_FQDN")
	if !ok {
		fmt.Println("failed to find QR_DOM_FQDN env var")
		os.Exit(1)
	}
	qrDomainBaseDN, ok := os.LookupEnv("QR_DOM_BASE")
	if !ok {
		fmt.Println("failed to find QR_DOM_BASE env var")
		os.Exit(1)
	}
	qrDomainBindUser, ok := os.LookupEnv("QR_DOM_BIND_USER")
	if !ok {
		fmt.Println("failed to find QR_DOM_BIND_USER env var")
		os.Exit(1)
	}
	qrDomainBindUserPass, ok := os.LookupEnv("QR_DOM_BIND_USER_PASS")
	if !ok {
		fmt.Println("failed to find QR_DOM_BIND_USER_PASS env var")
		os.Exit(1)
	}

	// checking OS env exists for OTP_DB_USR & OTP_DB_PASS
	dbUsr := os.Getenv("OTP_DB_USR")
	dbPass := os.Getenv("OTP_DB_PASS")
	if len(dbUsr) == 0 || len(dbPass) == 0 {
		fmt.Println("OTP_DB_USR and/or OTP_DB_PASS not found in OS env! exiting")
		os.Exit(1)
	}

	// setting flags
	logsDir := flag.String("log-dir", logsPathDefault, "set custom log dir")
	logsToKeep := flag.Int("keep-logs", 7, "set number of logs to keep after rotation")
	addr := flag.String("addr", ":3000", "HTTP server address, ex. ':3000' for localhost:3000")
	tlsCert := flag.String("tls-cert", tlsCertDefault, "full path to tls Cert file")
	tlsKey := flag.String("tls-key", tlsKeyDefault, "full path to tls Key file")
	dbName := flag.String("db", "otpportal", "MySQL db name")
	multiOTPBinPath := flag.String("m", "c:/MultiOTP/windows/multiotp.exe", "Full path to MulitOTP binary")
	// ldapBaseDN := flag.String("b", "dc=example,dc=com", "Base DN for LDAP Domain")

	flag.Usage = func() {
		fmt.Println("MultiOTP Web Portal for LDAP Users")
		fmt.Println("Version = x.x.x")
		// fmt.Println("Usage: <app> [-opt] ...")
		fmt.Println("Flags:")
		flag.PrintDefaults()
	}
	flag.Parse()

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
	dsn := fmt.Sprintf("%s:%s@/%s?parseTime=true", dbUsr, dbPass, *dbName)
	// try to open db
	db, err := openDB(dsn)
	if err != nil {
		logger.Error(err.Error())
	}
	defer db.Close()

	// Initialize a new template cache...
	templateCache, err := newTemplateCache()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// Init session manager
	sessionManager := scs.New()
	sessionManager.Store = mysqlstore.New(db)
	sessionManager.Lifetime = 12 * time.Hour
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
	}

	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
	}

	srv := &http.Server{
		Addr:         *addr,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
		TLSConfig:    tlsConfig,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
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
