//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"chichi/apis"
	"chichi/apis/errors"
	"chichi/telemetry"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/gorilla/securecookie"
)

// sessionMaxAge contains the max age property for the session cookie (6 hours).
const sessionMaxAge = 6 * 60 * 60

type admin struct {
	apis         *apis.APIs
	secureCookie *securecookie.SecureCookie // secureCookie contains keys to encrypt/decrypt/remove the session cookie.
}

func New(apis *apis.APIs, sessionKey []byte) *admin {
	a := &admin{
		apis: apis,
	}
	if len(sessionKey) != 64 {
		panic("sessionKey is not 64 bytes long")
	}
	hashKey, blockKey := sessionKey[:32], sessionKey[32:]
	a.secureCookie = securecookie.New(hashKey, blockKey)
	a.secureCookie.MaxAge(sessionMaxAge)
	return a
}

func (admin *admin) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx, s := telemetry.TraceSpan(r.Context(), "admin.ServeHTTP", "urlPath", r.URL.Path)
	defer s.End()

	telemetry.IncrementCounter(ctx, "admin.ServeHTTP", 1)

	rpath := r.URL.Path[6:]

	if rpath == "/logout" && r.Method == "POST" {
		admin.logout(w, r)
		return
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

	// check the session cookie.
	var isLoggedIn bool
	session := admin.getSession(r)
	if session == nil {
		isLoggedIn = false
	} else {
		organization, err := admin.apis.Organization(ctx, 1)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			slog.Error("cannot retrieve organization", "err", err)
			return
		}
		_, err = organization.Member(ctx, session.Member)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			slog.Error("cannot retrieve member", "err", err)
			return
		}
		isLoggedIn = true
		_ = admin.storeSession(session, w)
	}

	if strings.HasPrefix(rpath, "/src/") {
		admin.serveWithESBuild(ctx, w, r)
		return
	}

	if !isLoggedIn {
		http.Redirect(w, r, "/admin/", http.StatusTemporaryRedirect)
	}

	http.ServeFile(w, r, "./admin/public/index.html")

}

func (admin *admin) serveWithESBuild(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, span := telemetry.TraceSpan(ctx, "admin.serveWithESBuild", "path", r.URL.Path)
	defer span.End()
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
		MinifyIdentifiers: false,               // TODO: review in production.
		MinifySyntax:      false,               // TODO: review in production.
		MinifyWhitespace:  false,               // TODO: review in production.
		JSXDev:            true,                // TODO: review in production.
		Sourcemap:         api.SourceMapLinked, // TODO: review in production.
		Outdir:            "out",
		Target:            api.ES2018,
		TreeShaking:       api.TreeShakingTrue,
		Write:             false,
	})

	// Handle errors and warnings.
	if result.Errors != nil {
		errorMessages := &strings.Builder{}
		for _, msg := range result.Errors {
			slog.Error("ESBuild error", "msg", msg)
			errorMessages.WriteString(fmt.Sprint(msg))
		}
		slog.Error("errors while executing ESbuild, cannot serve URL", "url", r.URL.Path)
		http.Error(w, errorMessages.String(), http.StatusInternalServerError)
		return
	}
	if result.Warnings != nil {
		for _, msg := range result.Warnings {
			slog.Warn("ESBuild warning", "msg", msg)
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
			case ".map":
				w.Header().Add("Content-Type", "application/json")
			default:
				http.Error(w, "Bad Request: cannot determine Content-Type for this file type", http.StatusBadRequest)
				return
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
	organization, err := admin.apis.Organization(r.Context(), 1)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		slog.Error("cannot read organization", "err", err)
		return
	}
	memberID, err := organization.AuthenticateMember(r.Context(), loginData.Email, loginData.Password)
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
		slog.Error("cannot log member", "err", err)
		return
	}

	err = admin.addSession(organization.ID, w, r)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		slog.Error("cannot add session", "err", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc.Encode([]any{memberID, nil})
}

// logout logs the user out from the admin.
func (admin *admin) logout(res http.ResponseWriter, req *http.Request) {
	// Remove the session cookie (settings its MaxAge property to -1).
	err := admin.removeSession(res, req)
	if err != nil {
		http.Error(res, "Internal Server Error", http.StatusInternalServerError)
		slog.Error("cannot log out member", "err", err)
		return
	}
}
