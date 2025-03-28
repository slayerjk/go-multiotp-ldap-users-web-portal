package main

import (
	"html/template"
	"log/slog"
	"net/http"

	"github.com/slayerjk/go-multiotp-ldap-users-web-portal/internal/ldapwork"
	"github.com/slayerjk/go-multiotp-ldap-users-web-portal/internal/multiotp"
	"github.com/slayerjk/go-multiotp-ldap-users-web-portal/internal/qrwork"
	"github.com/slayerjk/go-multiotp-ldap-users-web-portal/internal/validator"
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
	var form userLoginForm

	// decode form
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// login validation
	form.CheckField(validator.NotBlank(form.Login), "login", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Login, validator.LoginRX), "login", "This field must be a valid login")
	// TODO: Ldap login validation

	// password validation
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")

	// check errors of form
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
		return
	}

	// making LDAP connection
	ldapConn, err := ldapwork.MakeLdapConnection(app.userDomainFQDN)
	if err != nil {
		app.logger.Error("failed to make LDAP connection", slog.Any("error", err))
		data := app.newTemplateData(r)
		app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
		return
	}

	// making TLS over LDAP connection
	err = ldapwork.StartTLSConnWoVerification(ldapConn)
	if err != nil {
		app.logger.Error("failed to make LDAP TLS connection", slog.Any("error", err))
		data := app.newTemplateData(r)
		app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
		return
	}

	// trying to Bind(authenticate via LDAP)
	bindUser := form.Login + "@" + app.userDomainFQDN
	err = ldapwork.LdapBind(ldapConn, bindUser, form.Password)
	// ldapConn, err := app.ldapConnectBind(form.Login, form.Password, app.userDomainFQDN)
	if err != nil {
		form.CheckField(false, "login", "Wrong LDAP login or password")
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

	// save displayName for context
	userDisplayName, err := app.ldapGetDisplayname(ldapConn, form.Login, app.userDomainBaseDN)
	if err != nil {
		app.logger.Warn("failed to do get displayName attr", "user", form.Login, slog.Any("error", err))
	}
	ldapConn.Close()

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

	// making LDAP connection
	ldapConn, err := ldapwork.MakeLdapConnection(app.qrDomainFQDN)
	if err != nil {
		app.logger.Error("failed to make QR LDAP connection", slog.Any("error", err))
		app.render(w, r, http.StatusOK, "view.tmpl", data)
		return
	}
	defer ldapConn.Close()

	// making TLS over LDAP connection
	err = ldapwork.StartTLSConnWoVerification(ldapConn)
	if err != nil {
		ldapConn.Close()
		app.logger.Error("failed to make QR LDAP TLS connection", slog.Any("error", err))
		app.render(w, r, http.StatusOK, "view.tmpl", data)
		return
	}

	// trying to Bind(authenticate via QR LDAP)
	bindUser := app.qrDomainBindUser + "@" + app.qrDomainFQDN
	err = ldapwork.LdapBind(ldapConn, bindUser, app.qrDomainBindUserPass)
	if err != nil {
		ldapConn.Close()
		app.logger.Warn("failed to do QR LDAP bind", "user", bindUser, slog.Any("error", err))
		app.render(w, r, http.StatusOK, "view.tmpl", data)
		return
	}

	// save displayName for context
	userSama, err := app.ldapMatchSamaAccName(ldapConn, accName, app.qrDomainBaseDN)
	ldapConn.Close()
	if err != nil {
		app.logger.Warn("failed to do get samaAccountName attr", "user", accName, slog.Any("error", err))
		app.render(w, r, http.StatusOK, "view.tmpl", data)
		return
	}

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
