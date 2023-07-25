//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package admin

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"chichi/apis"
	"chichi/apis/errors"

	"github.com/evanw/esbuild/pkg/api"
)

type admin struct {
	apis *apis.APIs
}

func New(apis *apis.APIs) *admin {
	return &admin{apis: apis}
}

func (admin *admin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rpath := r.URL.Path[6:]

	// check the session cookie.
	var isLoggedIn bool
	cookie, err := r.Cookie("session")
	if err == nil {
		isLoggedIn = true
	}

	var accountID int
	var workspace *apis.Workspace
	if isLoggedIn {
		// get the account id
		accountID, _ = strconv.Atoi(cookie.Value)

		// instantiate the account API
		account, err := admin.apis.Account(accountID)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// get the workspace
		workspace, err = account.Workspace(1) // TODO(marco)
		if err != nil {
			http.NotFound(w, r)
			return
		}

	}

	// handle requests to login page.
	if rpath == "/" {
		if r.Method == "POST" {
			admin.login(w, r)
			return
		}
		http.ServeFile(w, r, "./admin/public/index.html")
		return
	}

	if strings.HasPrefix(rpath, "/src/") {
		admin.serveWithESBuild(w, r)
		return
	}

	if !isLoggedIn {
		http.Redirect(w, r, "/admin/", http.StatusTemporaryRedirect)
	}

	// TODO(Gianluca): use the 'workspace' variable or just remove it.
	_ = workspace

	http.ServeFile(w, r, "./admin/public/index.html")

}

func (admin *admin) serveWithESBuild(w http.ResponseWriter, r *http.Request) {
	file, err := filepath.Abs("admin/src/index.jsx")
	if err != nil {
		panic(err)
	}
	result := api.Build(api.BuildOptions{
		Bundle:            true,
		EntryPoints:       []string{file},
		Format:            api.FormatESModule,
		JSX:               api.JSXAutomatic,
		LegalComments:     api.LegalCommentsEndOfFile,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		MinifyWhitespace:  true,
		Outdir:            "out",
		Target:            api.ES2018,
		TreeShaking:       api.TreeShakingTrue,
		Write:             false,
	})

	// Handle errors and warnings.
	if result.Errors != nil {
		errorMessages := &strings.Builder{}
		for _, msg := range result.Errors {
			log.Printf("[error] ESBuild error: %v", msg)
			errorMessages.WriteString(fmt.Sprint(msg))
		}
		log.Printf("[error] errors while executing ESbuild, cannot serve %q", r.URL.Path)
		http.Error(w, errorMessages.String(), http.StatusInternalServerError)
		return
	}
	if result.Warnings != nil {
		for _, msg := range result.Warnings {
			log.Printf("[warning] ESBuild warning: %v", msg)
		}
	}

	base := path.Base(r.URL.Path)
	for _, out := range result.OutputFiles {
		if strings.HasSuffix(out.Path, base) {
			switch filepath.Ext(base) {
			case ".js":
				w.Header().Add("Content-Type", "text/javascript")
			case ".css":
				w.Header().Add("Content-Type", "text/css")
			default:
				log.Printf("[error] cannot determine Content-Type for %q", base)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			w.Write(out.Contents)
			return
		}
	}
	http.NotFound(w, r)
}

func (admin *admin) login(w http.ResponseWriter, r *http.Request) {
	loginData := struct {
		Email    string
		Password string
	}{}
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&loginData)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	enc := json.NewEncoder(w)
	accountID, err := admin.apis.AuthenticateAccount(loginData.Email, loginData.Password)
	if err != nil {
		if err, ok := err.(*errors.UnprocessableError); ok && err.Code == apis.AuthenticationFailed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			enc.Encode([]any{0, "AuthenticationFailed"})
			return
		}
		if err, ok := err.(errors.ResponseWriterTo); ok {
			_ = err.WriteTo(w)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("[error] cannot log account: %s", err)
		return
	}

	http.SetCookie(w, &http.Cookie{Name: "session", Value: strconv.Itoa(accountID), Path: "/"})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc.Encode([]any{accountID, nil})
}
