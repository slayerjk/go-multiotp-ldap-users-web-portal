package main

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/slayerjk/go-multiotp-ldap-users-web-portal/internal/multiotp"
	"github.com/slayerjk/go-multiotp-ldap-users-web-portal/internal/qrwork"
	"github.com/slayerjk/go-multiotp-ldap-users-web-portal/internal/validator"
	ldapwork "github.com/slayerjk/go-valdapwork"
)

// Home handler
func (app *application) home(w http.ResponseWriter, r *http.Request) {
	// data := app.newTemplateData(r)
	// app.render(w, r, http.StatusOK, "home.tmpl", data)

	http.Redirect(w, r, "/qr/view", http.StatusSeeOther)
}

// Create a new userLoginForm struct.
type userLoginForm struct {
	// Email               string `form:"email"`
	Login               string `form:"login"`
	Password            string `form:"password"`
	OTP                 string `form:"otp"`
	validator.Validator `form:"-"`
}

// Update the handler so it displays the login page.
func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userLoginForm{}
	app.render(w, r, http.StatusOK, "login.tmpl", data)
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	// Decode the form data into the userLoginForm struct.
	var (
		form          userLoginForm
		blankFieldErr string
		validLoginErr string
		validOTPErr   string
		ldapAuthErr   string
		otpAuthErr    string
	)

	// decode form
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// localization set
	if *app.lang == "ru" {
		blankFieldErr = "Это поле не может быть пустым"
		validLoginErr = "Логин не валидный"
		validOTPErr = "OTP не валидный"
		ldapAuthErr = "Не верный логин или пароль"
		otpAuthErr = "Не верный OTP"
	} else {
		blankFieldErr = "This field cannot be blank"
		validLoginErr = "This field must be a valid login"
		validOTPErr = "OTP is not valid"
		ldapAuthErr = "Wrong login or password"
		otpAuthErr = "Wrong OTP"
	}

	// login validation
	form.CheckField(validator.NotBlank(form.Login), "login", blankFieldErr)
	form.CheckField(validator.Matches(form.Login, validator.LoginRX), "login", validLoginErr)

	// password validation
	form.CheckField(validator.NotBlank(form.Password), "password", blankFieldErr)

	// OTP field validation
	if *app.secondFactorOn {
		form.CheckField(validator.NotBlank(form.OTP), "otp", blankFieldErr)
		form.CheckField(validator.ValidOTP(form.OTP), "otp", validOTPErr)
	}

	// check errors of form
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
		return
	}

	// making LDAP connection with TLS
	ldapConn, err := ldapwork.StartTLSConnWoVerification(app.userDomainFQDN)
	if err != nil {
		app.logger.Error("failed to make LDAP TLS connection", slog.Any("error", err))
		data := app.newTemplateData(r)
		app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
		return
	}

	// trying to Bind(authenticate via LDAP)
	app.logger.Info("making LDAP BIND", "user", form.Login)
	bindUser := form.Login + "@" + app.userDomainFQDN
	err = ldapwork.LdapBind(ldapConn, bindUser, form.Password)
	// ldapConn, err := app.ldapConnectBind(form.Login, form.Password, app.userDomainFQDN)
	if err != nil {
		form.CheckField(false, "login", ldapAuthErr)
		app.logger.Warn("failed to do LDAP bind", "user", form.Login, slog.Any("error", err))
	}
	defer ldapConn.Close()

	// check errors of form
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
		return
	}

	// OTP auth, if enabled
	app.logger.Info("making PrivacyIdea validate check of given user's OTP", "user", form.Login)
	if *app.secondFactorOn {
		_, err := mfaAuth(app.mfaTriggerUser, app.mfaTriggerUserPass, app.mfaUrl, app.userDomainFQDN, form.Login, form.OTP)
		if err != nil {
			form.CheckField(false, "otp", otpAuthErr)
			app.logger.Warn("failed to do make OTP Auth", slog.Any("error", err))
		}
	}

	// check errors of form
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
		return
	}

	// save displayName for
	filter := fmt.Sprintf("(&(objectClass=user)(samaccountname=%s))", form.Login)
	userDisplayName, err := ldapwork.GetAttr(ldapConn, filter, form.Login, app.userDomainBaseDN, "displayName")
	if err != nil {
		app.logger.Warn("failed to do get displayName attr", "user", form.Login, slog.Any("error", err))
	}
	defer ldapConn.Close()

	// TODO: add PrivacyIdea validate check

	// renew session token
	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// add auth id to session
	app.sessionManager.Put(r.Context(), "authenticatedUserID", 1)

	// add account name & AD displayName attr to the session
	app.sessionManager.Put(r.Context(), "accName", form.Login)
	app.sessionManager.Put(r.Context(), "displayName", userDisplayName)

	// redirect user to qr view page(view.tmpl)
	http.Redirect(w, r, "/qr/view", http.StatusSeeOther)
}

// QR view page for authenticated users
func (app *application) qrView(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	// get accName from session
	displayName := app.sessionManager.GetString(r.Context(), "displayName")
	accName := app.sessionManager.GetString(r.Context(), "accName")
	// use accName if displayName is empty
	data.Username = displayName
	if len(displayName) == 0 {
		data.Username = accName
	}

	// making TLS over LDAP connection
	ldapConn, err := ldapwork.StartTLSConnWoVerification(app.qrDomainFQDN)
	if err != nil {
		app.logger.Error("failed to make QR LDAP TLS connection", slog.Any("error", err))
		app.render(w, r, http.StatusOK, "view.tmpl", data)
		return
	}
	defer ldapConn.Close()

	// trying to Bind(authenticate via QR LDAP)
	bindUser := app.qrDomainBindUser + "@" + app.qrDomainFQDN
	err = ldapwork.LdapBind(ldapConn, bindUser, app.qrDomainBindUserPass)
	if err != nil {
		app.logger.Warn("failed to do QR LDAP bind", "user", bindUser, slog.Any("error", err))
		app.render(w, r, http.StatusOK, "view.tmpl", data)
		return
	}
	defer ldapConn.Close()

	// save sAMAccountName for context
	filter := fmt.Sprintf("(&(objectClass=user)(samaccountname=*%s))", accName)
	userSama, err := ldapwork.GetAttr(ldapConn, filter, accName, app.qrDomainBaseDN, "sAMAccountName")
	if err != nil {
		app.logger.Warn("failed to do get samaAccountName attr", "user", accName, slog.Any("error", err))
		app.render(w, r, http.StatusOK, "view.tmpl", data)
		return
	}
	defer ldapConn.Close()

	// Put context of SamaAccount name(to use for reissueQR)
	app.sessionManager.Put(r.Context(), "QrAcc", userSama)

	// get totpURL
	totpURL, err := multiotp.GetMultiOTPTokenURL(userSama, *app.multiOTPBinPath)
	if err != nil {
		// app.serverError(w, r, fmt.Errorf("failed to get totpURL:\n\t%v", err))
		app.logger.Warn("failed to find totpURL", "user", userSama, slog.Any("error", err))
		// render view.tmpl with empty QR
		app.render(w, r, http.StatusOK, "view.tmpl", data)
		return
	}

	// get QR svg content(between <svg> tags)
	qr, err := qrwork.GenerateTOTPSvgQrHTML(totpURL)
	if err != nil {
		// app.serverError(w, r, fmt.Errorf("failed to get qr for %s:\n\t%v", accName, err))
		app.logger.Warn("failed to generate QR", "user", userSama, slog.Any("error", err))
		// render view.tmpl with empty QR
		app.render(w, r, http.StatusOK, "view.tmpl", data)
		return
	}

	// save string qr as HTML code
	data.QR = template.HTML(qr)

	// app.render(w, r, http.StatusOK, "create.tmpl", data)
	app.render(w, r, http.StatusOK, "view.tmpl", data)
}

// Reissue QR and redirect ot qrView for authenticated users
func (app *application) qrReissue(w http.ResponseWriter, r *http.Request) {
	// get accName from session
	qrAcc := app.sessionManager.GetString(r.Context(), "QrAcc")
	if len(qrAcc) == 0 {
		app.logger.Error("failed to reissue QR, Empty QrAcc")
		app.sessionManager.Put(r.Context(), "flash", "Ваш QR НЕ перевыпущен!")
		if *app.lang == "en" {
			app.sessionManager.Put(r.Context(), "flash", "Your QR hasn't been reissued!")
		}
		http.Redirect(w, r, "/qr/view", http.StatusSeeOther)
	}

	// make reissue of user(del->resync)
	err := multiotp.ReissueMultiOTPQR(*app.multiOTPBinPath, qrAcc)
	if err != nil {
		app.logger.Error("failed to reissue QR", "acc", qrAcc, slog.Any("error", err))
	}

	app.sessionManager.Put(r.Context(), "flash", "Ваш QR перевыпущен!")
	if *app.lang == "en" {
		app.sessionManager.Put(r.Context(), "flash", "Your QR has been reissued!")
	}

	http.Redirect(w, r, "/qr/view", http.StatusSeeOther)
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	// Use the RenewToken() method on the current session to change the session
	// ID again.
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// Remove the authenticatedUserID from the session data so that the user is
	// 'logged out'.
	app.sessionManager.Remove(r.Context(), "authenticatedUserID")

	// remove accName & displayName from the session
	app.sessionManager.Remove(r.Context(), "accName")
	app.sessionManager.Remove(r.Context(), "displayName")
	app.sessionManager.Remove(r.Context(), "QrAcc")

	// Add a flash message to the session to confirm to the user that they've been
	// logged out.
	app.sessionManager.Put(r.Context(), "flash", "Вы успешно вышли!")
	if *app.lang == "en" {
		app.sessionManager.Put(r.Context(), "flash", "You've been logged out successfully!")
	}

	// Redirect the user to the application home page.
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
