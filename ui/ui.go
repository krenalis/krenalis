//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package ui

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/open2b/chichi/apis"
	"github.com/open2b/chichi/telemetry"

	"github.com/evanw/esbuild/pkg/api"
)

type ui struct {
	apis *apis.APIs
}

func New(apis *apis.APIs) *ui {
	a := &ui{
		apis: apis,
	}
	return a
}

func (ui *ui) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx, s := telemetry.TraceSpan(r.Context(), "ui.ServeHTTP", "urlPath", r.URL.Path)
	defer s.End()

	telemetry.IncrementCounter(ctx, "ui.ServeHTTP", 1)

	if strings.HasPrefix(r.URL.Path[len("/ui"):], "/src/") {
		ui.serveWithESBuild(ctx, w, r)
		return
	}

	http.ServeFile(w, r, "./ui/public/index.html")

}

func (ui *ui) serveWithESBuild(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, span := telemetry.TraceSpan(ctx, "ui.serveWithESBuild", "path", r.URL.Path)
	defer span.End()
	file, err := filepath.Abs("ui/src/index.jsx")
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
