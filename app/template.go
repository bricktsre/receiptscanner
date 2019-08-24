package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/bricktsre/receiptscanner"
)

func parseTemplate(filename string) *appTemplate {
	tmpl := template.Must(template.ParseFiles("templates/base.html"))

	path := filepath.Join("templates", filename)
	b, err := ioutil.ReadFile(path)
	if err != nil {
		panic(fmt.Errorf("could not read template: %v", err))
	}
	template.Must(tmpl.New("body").Parse(string(b)))

	return &appTemplate{tmpl.Lookup("base.html")}
}

type appTemplate struct {
	t *template.Template
}

func (tmpl *appTemplate) Execute(w http.ResponseWriter, r *http.Request, data interface{}) *appError {
	d := struct {
		Data        interface{}
		AuthEnabled bool
		Profile     *Profile
		LoginURL    string
		LogoutURL   string
	}{
		Data:        data,
		AuthEnabled: receiptscanner.OAuthConfig != nil,
		LoginURL:    "/login?redirect=" + r.URL.RequestURI(),
		LogoutURL:   "/logout?redirect=" + r.URL.RequestURI(),
	}

	if d.AuthEnabled {
		// Ignore any errors.
		d.Profile = profileFromSession(r)
	}

	if err := tmpl.t.Execute(w, d); err != nil {
		return appErrorf(err, "could not write template: %v", err)
	}
	return nil
}
