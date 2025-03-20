package main

import (
	"fmt"
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

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
		return
	}

	// renew session token
	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// add auth id to session
	app.sessionManager.Put(r.Context(), "authenticatedUserID", 1)

	// add account name to the session
	app.sessionManager.Put(r.Context(), "accName", form.Login)

	// redirect user to qr view page(view.tmpl)
	http.Redirect(w, r, "/qr/view", http.StatusSeeOther)
}

func (app *application) qrView(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	// get accName from session
	accName := app.sessionManager.GetString(r.Context(), "accName")
	if len(accName) == 0 {
		app.serverError(w, r, fmt.Errorf("no accName found"))
		return
	}
	// fmt.Println(accName)

	// get QR svg code for user
	// qrFile, err := os.Open("TEST.svg")
	// if err != nil {
	// 	app.logger.Error("failed to open svg")
	// 	return
	// }
	// defer qrFile.Close()

	// qr, err := io.ReadAll(qrFile)
	// if err != nil {
	// 	app.logger.Error("failed to read svg")
	// 	return
	// }

	// data.QR = qr
	// data.QR = "HELLO!"

	// get totpURL
	totpURL, err := multiotp.GetMultiOTPTokenURL(accName, *app.multiOTPBinPath)
	if err != nil {
		app.serverError(w, r, fmt.Errorf("failed to get totpURL:\n\t%v", err))
		return
	}

	// get QR svg content(between <svg> tags)
	qr, err := qrwork.GenerateTOTPSvgQrHTML(totpURL)
	if err != nil {
		app.serverError(w, r, fmt.Errorf("failed to get qr for %s:\n\t%v", accName, err))
	}
	data.QR = qr

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

	// remove accName from the session
	app.sessionManager.Remove(r.Context(), "accName")

	// Add a flash message to the session to confirm to the user that they've been
	// logged out.
	app.sessionManager.Put(r.Context(), "flash", "You've been logged out successfully!")

	// Redirect the user to the application home page.
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
