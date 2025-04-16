//go:build dev

//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/andybalholm/brotli"
	esbuild "github.com/evanw/esbuild/pkg/api"
)

// moduleRoot is the root directory of the Go module.
var moduleRoot string

// Path to the Shoelace icons within the "node_modules" directory.
const shoelaceIconsPath = "@shoelace-style/shoelace/dist/assets/icons"

func init() {
	// Set the moduleRoot global variable.
	dir, err := os.Getwd()
	if err != nil {
		panic("cannot get current working directory")
	}
	for {
		st, err := os.Stat(filepath.Join(dir, "go.mod"))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}
		if err == nil && st.Mode().IsRegular() {
			moduleRoot = dir
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("go.mod file not found in current directory or any parent directory")
		}
		dir = parent
	}
}

// assetsHandler implements a http.Handler to serve UI assets.
// It monitors the JavaScript and CSS files and creates a bundle for JavaScript
// and a separate bundle for CSS. It also serves the bundled files compressed
// with Brotli if the client supports it.
type assetsHandler struct {
	outDir  string
	fs      http.Handler
	watcher esbuild.BuildContext
}

func newAssetsHandler(_ fs.FS) (h *assetsHandler, err error) {

	// Create a temporary directory for the assets.
	outDir, err := os.MkdirTemp("", "meergo-assets-")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(outDir)
		}
	}()

	h = &assetsHandler{
		outDir: outDir,
		fs:     http.StripPrefix("/admin/src/", http.FileServer(http.Dir(outDir))),
	}

	entryPoint := filepath.Join(moduleRoot, "assets", "src", "index.jsx")

	// Watches the source files and rebuilds the assets when changes occur.
	var ctxErr *esbuild.ContextError
	h.watcher, ctxErr = esbuild.Context(esbuild.BuildOptions{
		Bundle:            true,
		EntryPoints:       []string{entryPoint},
		Format:            esbuild.FormatESModule,
		JSX:               esbuild.JSXAutomatic,
		JSXDev:            true,
		MinifyIdentifiers: false, // It must be false for the JSX dev setting to work
		MinifySyntax:      true,
		MinifyWhitespace:  true,
		Outdir:            outDir,
		Sourcemap:         esbuild.SourceMapLinked,
		Target:            esbuild.ES2018,
		TreeShaking:       esbuild.TreeShakingTrue,
		Write:             true,
	})
	if ctxErr != nil {
		var b strings.Builder
		for _, msg := range ctxErr.Errors {
			b.WriteString(fmt.Sprint(msg))
		}
		return nil, fmt.Errorf("cannot make esbuild contex: %s", b.String())
	}
	err = h.watcher.Watch(esbuild.WatchOptions{})
	if err != nil {
		return nil, fmt.Errorf("cannot watch assets: %s", err)
	}

	return h, nil
}

func (h *assetsHandler) Close() {
	h.watcher.Cancel()
	h.watcher.Dispose()
	_ = os.RemoveAll(h.outDir)
}

func (h *assetsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.Header.Get("Accept-Encoding"), "br") {
		w.Header().Set("Content-Encoding", "br")
		w.Header().Set("Vary", "Accept-Encoding")
		bw := brotli.NewWriter(w)
		defer bw.Close()
		w = brotliResponseWriter{bw, w}
	}
	if r.URL.Path == "/javascript-sdk/dist/meergo.min.js" {
		w.Header().Set("Content-Type", "text/javascript")
		http.ServeFile(w, r, filepath.Join(moduleRoot, "javascript-sdk", "dist", "meergo.min.js"))
		return
	}
	if r.URL.Path == "/javascript-sdk/dist/meergo.min.js.map" {
		w.Header().Set("Content-Type", "application/json")
		http.ServeFile(w, r, filepath.Join(moduleRoot, "javascript-sdk", "dist", "meergo.min.js.map"))
		return
	}
	if r.URL.Path == "/javascript-sdk/mywebsite/" || r.URL.Path == "/javascript-sdk/mywebsite/index.html" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, filepath.Join(moduleRoot, "javascript-sdk", "mywebsite", "index.html"))
		return
	}
	if strings.HasPrefix(r.URL.Path, "/admin/src/") {
		if icon, ok := strings.CutPrefix(r.URL.Path, "/admin/src/shoelace/dist/assets/icons/"); ok {
			w.Header().Set("Content-Type", "image/svg+xml")
			http.ServeFile(w, r, filepath.Join(moduleRoot, "assets/node_modules", shoelaceIconsPath, icon))
			return
		}
		h.fs.ServeHTTP(w, r)
		return
	}
	result := h.watcher.Rebuild()
	if result.Errors != nil {
		msg := "cannot generate assets when making vendor:"
		for _, err := range result.Errors {
			if len(result.Errors) == 1 {
				msg += " "
			} else {
				msg += "\n  - "
			}
			msg += err.Text
			if loc := err.Location; loc != nil {
				msg += fmt.Sprintf(" at %s %d:%d", loc.File, loc.Line, loc.Column)
			}
		}
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	if result.Warnings != nil {
		msg := "cannot generate assets when making vendor:"
		for _, err := range result.Warnings {
			if len(result.Warnings) == 1 {
				msg += " "
			} else {
				msg += "\n  - "
			}
			msg += err.Text
			if loc := err.Location; loc != nil {
				msg += fmt.Sprintf(" at %s %d:%d", loc.File, loc.Line, loc.Column)
			}
		}
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	index := filepath.Join(moduleRoot, "assets", "public", "index.html")
	http.ServeFile(w, r, index)
}

type brotliResponseWriter struct {
	brotliWriter *brotli.Writer
	http.ResponseWriter
}

func (b brotliResponseWriter) Write(data []byte) (int, error) {
	return b.brotliWriter.Write(data)
}
