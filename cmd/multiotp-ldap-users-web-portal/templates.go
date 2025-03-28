package main

import (
	"html/template"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/slayerjk/go-multiotp-ldap-users-web-portal/ui"
)

// Define a templateData type to act as the holding structure for dynamic data
type templateData struct {
	CurrentYear     int
	Form            any
	Flash           string
	IsAuthenticated bool
	CSRFToken       string
	QR              template.HTML // must be <svg> code chunc to insert in template
	Username        string
}

// Create a humanDate function which returns a human date
func humanDate(t time.Time) string {
	return t.Format("02 Jan 2006 at 15:04")
}

// Init template function to pass a date
var functions = template.FuncMap{
	"humanDate": humanDate,
}

func newTemplateCache(lang string) (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	// setting different pathes for languages
	langPrefix := "html/ru"
	if lang == "en" {
		langPrefix = "html/en"
	}

	pages, err := fs.Glob(ui.Files, langPrefix+"/pages/*.tmpl")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		// Create a slice containing the filepath patterns for the templates we
		// want to parse.
		patterns := []string{
			langPrefix + "/base.tmpl",
			langPrefix + "/partials/*.tmpl",
			page,
		}

		// Use ParseFS() instead of ParseFiles() to parse the template files
		// from the ui.Files embedded filesystem.
		ts, err := template.New(name).Funcs(functions).ParseFS(ui.Files, patterns...)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}
