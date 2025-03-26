package main

import (
	"html/template"
	"net/http"

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

	// TODO: Ldap password validation
	// check LDAP password via validator(?)

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
		return
	}

	// trying to authenicate user via LDAP
	userDisplayName, err := app.ldapAuth(form.Login, form.Password)
	if err != nil {
		app.serverError(w, r, err)
	}

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

	// get samaAcountName for QR domain of corresponding user

	// get totpURL
	totpURL, err := multiotp.GetMultiOTPTokenURL(accName, *app.multiOTPBinPath)
	if err != nil {
		// app.serverError(w, r, fmt.Errorf("failed to get totpURL:\n\t%v", err))
		app.logger.Warn("failed to find totpURL", "user", accName)
		// render view.tmpl with empty QR
		app.render(w, r, http.StatusOK, "view.tmpl", data)
		return
	}

	// get QR svg content(between <svg> tags)
	qr, err := qrwork.GenerateTOTPSvgQrHTML(totpURL)
	if err != nil {
		// app.serverError(w, r, fmt.Errorf("failed to get qr for %s:\n\t%v", accName, err))
		app.logger.Warn("failed to generate QR", "user", accName)
		// render view.tmpl with empty QR
		app.render(w, r, http.StatusOK, "view.tmpl", data)
		return
	}

	// save string qr as HTML code
	data.QR = template.HTML(qr)

	// app.render(w, r, http.StatusOK, "create.tmpl", data)
	app.render(w, r, http.StatusOK, "view.tmpl", data)
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

	// Add a flash message to the session to confirm to the user that they've been
	// logged out.
	app.sessionManager.Put(r.Context(), "flash", "You've been logged out successfully!")

	// Redirect the user to the application home page.
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
