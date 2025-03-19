package main

import (
	"net/http"

	"github.com/justinas/alice"
	"github.com/slayerjk/go-multiotp-ldap-users-web-portal/ui"
)

// The routes() method returns a servemux containing our application routes.
func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.FileServerFS(ui.Files))

	// for dynamic pages, see all
	dynamic := alice.New(app.sessionManager.LoadAndSave, noSurf, app.authenticate)

	mux.Handle("GET /{$}", dynamic.ThenFunc(app.home))
	mux.Handle("GET /user/login", dynamic.ThenFunc(app.userLogin))
	mux.Handle("POST /user/login", dynamic.ThenFunc(app.userLoginPost))
	// mux.Handle("GET /qrview", dynamic.ThenFunc(app.qrView))

	// protected pages, only for autenticated users
	protected := dynamic.Append(app.requireAuthentication)

	mux.Handle("POST /user/logout", protected.ThenFunc(app.userLogoutPost))
	mux.Handle("GET /qrview", protected.ThenFunc(app.qrView))

	// for all pages
	standard := alice.New(app.recoverPanic, app.logRequest, commonHeaders)

	return standard.Then(mux)
}
