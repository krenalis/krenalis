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
	"log/slog"
	"net/http"
	"os"
	"path"
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

// assetsHandler implements a http.Handler to serve admin assets.
// It monitors the JavaScript and CSS files and creates a bundle for JavaScript
// and a separate bundle for CSS. It also serves the bundled files compressed
// with Brotli if the client supports it.
type assetsHandler struct {
	outDir   string
	fs       http.Handler
	watchers struct {
		index  esbuild.BuildContext
		monaco struct {
			main         esbuild.BuildContext
			tsWorker     esbuild.BuildContext
			editorWorker esbuild.BuildContext
		}
	}
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

	// Build the admin.
	entryPoint := filepath.Join(moduleRoot, "assets", "src", "index.jsx")
	external := []string{"monaco-editor"}
	h.watchers.index, err = watchAndBuild(entryPoint, outDir, external)
	if err != nil {
		return nil, fmt.Errorf("cannot bundle admin: %w", err)
	}

	// Build Monaco editor and its workers.
	entryPoint = filepath.Join(moduleRoot, "assets", "node_modules", "monaco-editor", "esm", "vs", "editor", "editor.main.js")
	monacoOutDir := filepath.Join(outDir, "monaco")
	external = []string{
		"vs/language/json/json.worker.js",
		"vs/language/css/css.worker.js",
		"vs/language/html/html.worker.js",
		"vs/language/typescript/ts.worker.js",
		"vs/editor/editor.worker.js",
	}
	h.watchers.monaco.main, err = watchAndBuild(entryPoint, monacoOutDir, external)
	if err != nil {
		return nil, fmt.Errorf("cannot bundle Monaco editor: %w", err)
	}
	h.watchers.monaco.tsWorker, err = watchAndBuild(filepath.Join(moduleRoot, "assets", "node_modules", "monaco-editor", "esm", "vs", "language", "typescript", "ts.worker.js"), monacoOutDir, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot bundle Monaco's `ts.worker.js` worker: %w", err)
	}
	h.watchers.monaco.editorWorker, err = watchAndBuild(filepath.Join(moduleRoot, "assets", "node_modules", "monaco-editor", "esm", "vs", "editor", "editor.worker.js"), monacoOutDir, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot bundle Monaco's `editor.worker.js` worker: %w", err)
	}

	return h, nil
}

func (h *assetsHandler) Close() {
	h.watchers.index.Cancel()
	h.watchers.index.Dispose()
	h.watchers.monaco.main.Cancel()
	h.watchers.monaco.main.Dispose()
	h.watchers.monaco.tsWorker.Cancel()
	h.watchers.monaco.tsWorker.Dispose()
	h.watchers.monaco.editorWorker.Cancel()
	h.watchers.monaco.editorWorker.Dispose()
	_ = os.RemoveAll(h.outDir)
}

var contentType = map[string]string{
	".css": "text/css",
	".js":  "application/x-javascript",
	".map": "application/json",
	".ttf": "font/ttf",
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
	if strings.HasPrefix(r.URL.Path, "/admin/src/") {
		// Serve Shoelace icons.
		if icon, ok := strings.CutPrefix(r.URL.Path, "/admin/src/shoelace/dist/assets/icons/"); ok {
			w.Header().Set("Content-Type", "image/svg+xml")
			http.ServeFile(w, r, filepath.Join(moduleRoot, "assets/node_modules", shoelaceIconsPath, icon))
			return
		}
		// Serve Monaco editor.
		if after, ok := strings.CutPrefix(r.URL.Path, "/admin/src/monaco/"); ok {
			switch after {
			case
				"vs/editor/codicon-37A3DWZT.ttf",
				"vs/editor/editor.main.css",
				"vs/editor/editor.main.css.map",
				"vs/editor/editor.main.js",
				"vs/editor/editor.main.js.map",
				"vs/editor/editor.worker.js",
				"vs/editor/editor.worker.js.map",
				"vs/language/typescript/ts.worker.js",
				"vs/language/typescript/ts.worker.js.map":
			default:
				http.NotFound(w, r)
				return
			}
			name := path.Base(after)
			if ct, ok := contentType[path.Ext(name)]; ok {
				w.Header().Add("Content-Type", ct)
			}
			data, err := os.ReadFile(filepath.Join(h.outDir, "monaco", name))
			if err != nil {
				slog.Error("cannot read Monaco asset file", "path", name, "err", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			_, _ = w.Write(data)
			return
		}
		h.fs.ServeHTTP(w, r)
		return
	}
	watchers := []esbuild.BuildContext{
		h.watchers.index,
		h.watchers.monaco.main,
		h.watchers.monaco.tsWorker,
		h.watchers.monaco.editorWorker,
	}
	for _, watcher := range watchers {
		result := watcher.Rebuild()
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
	}
	// Set Content-Security-Policy header.
	// The hash refers to the inline import map in the index.html file.
	w.Header().Set("Content-Security-Policy", "script-src 'self' 'sha256-Eggv/sxfau5R7DCyaPw6OwmcQOlrO9oX7GhT3aozILU='")
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

// watchAndBuild watches source files and bundles them when a change occurs.
func watchAndBuild(entryPoint, outDir string, external []string) (esbuild.BuildContext, error) {
	var ctxErr *esbuild.ContextError
	buildContext, ctxErr := esbuild.Context(esbuild.BuildOptions{
		Bundle:            true,
		EntryPoints:       []string{entryPoint},
		Format:            esbuild.FormatESModule,
		JSX:               esbuild.JSXAutomatic,
		JSXDev:            true,
		LegalComments:     esbuild.LegalCommentsEndOfFile,
		MinifyIdentifiers: false, // It must be false for the JSX dev setting to work
		MinifySyntax:      true,
		MinifyWhitespace:  true,
		Outdir:            outDir,
		Sourcemap:         esbuild.SourceMapLinked,
		Target:            esbuild.ES2018,
		TreeShaking:       esbuild.TreeShakingTrue,
		Loader: map[string]esbuild.Loader{
			".ttf": esbuild.LoaderFile,
		},
		External: external,
		Write:    true,
	})
	if ctxErr != nil {
		var b strings.Builder
		for _, msg := range ctxErr.Errors {
			b.WriteString(fmt.Sprint(msg))
		}
		return nil, fmt.Errorf("cannot make esbuild contex: %s", b.String())
	}
	err := buildContext.Watch(esbuild.WatchOptions{})
	if err != nil {
		return nil, fmt.Errorf("cannot watch assets: %s", err)
	}
	return buildContext, nil
}
