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

	// home (for all)
	mux.Handle("GET /{$}", dynamic.ThenFunc(app.home))

	// login page (for all)
	mux.Handle("GET /user/login", dynamic.ThenFunc(app.userLogin))
	mux.Handle("POST /user/login", dynamic.ThenFunc(app.userLoginPost))

	// protected pages, only for autenticated users
	protected := dynamic.Append(app.requireAuthentication)

	// logout (for authenticated user)
	mux.Handle("POST /user/logout", protected.ThenFunc(app.userLogoutPost))

	// view QR (for authenticated user)
	mux.Handle("GET /qr/view", protected.ThenFunc(app.qrView))

	// reissue QR (for authenticated user)
	mux.Handle("GET /qr/reissue", protected.ThenFunc(app.qrReissue))

	// for all pages
	standard := alice.New(app.recoverPanic, app.logRequest, commonHeaders)

	return standard.Then(mux)
}
