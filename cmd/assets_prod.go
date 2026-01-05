//go:build !dev

// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	_ "embed"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/andybalholm/brotli"
)

const devMode = false

// assetsHandler implements an http.Handler to serve the Admin assets. It serves
// bundle files that are embedded in the executable, compressed with Brotli if
// the client supports it.
type assetsHandler struct {
	fsys fs.FS
}

func newAssetsHandler(fsys fs.FS) (*assetsHandler, error) {
	return &assetsHandler{fsys}, nil
}

func (h *assetsHandler) Close() {}

var contentType = map[string]string{
	".css": "text/css",
	".js":  "application/x-javascript",
	".map": "application/json",
	".ttf": "font/ttf",
}

func (h *assetsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var asset string
	if after, ok := strings.CutPrefix(r.URL.Path, "/admin/src/"); ok {
		switch after {
		case
			"index.css",
			"index.css.map",
			"index.js",
			"index.js.map",
			"monaco/vs/editor/codicon-37A3DWZT.ttf",
			"monaco/vs/editor/editor.main.css",
			"monaco/vs/editor/editor.main.css.map",
			"monaco/vs/editor/editor.main.js",
			"monaco/vs/editor/editor.main.js.map",
			"monaco/vs/editor/editor.worker.js",
			"monaco/vs/editor/editor.worker.js.map",
			"monaco/vs/language/typescript/ts.worker.js",
			"monaco/vs/language/typescript/ts.worker.js.map":
			if ct, ok := contentType[path.Ext(after)]; ok {
				w.Header().Add("Content-Type", ct)
			}
		case "codicon-37A3DWZT.ttf":
			// See issue https://github.com/meergo/meergo/issues/1497
			after = "monaco/vs/editor/codicon-37A3DWZT.ttf"
			w.Header().Add("Content-Type", "font/ttf")
		default:
			if !strings.HasSuffix(after, ".svg") {
				http.NotFound(w, r)
				return
			}
			w.Header().Add("Content-Type", "image/svg+xml")
		}
		asset = after + ".br"
	} else {
		// Set Content-Security-Policy header.
		// The hash refers to the inline import map in the index.html file.
		w.Header().Set("Content-Security-Policy", "script-src 'self' 'sha256-Eggv/sxfau5R7DCyaPw6OwmcQOlrO9oX7GhT3aozILU='")
		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		asset = "index.html.br"
	}
	fi, err := h.fsys.Open(asset)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer fi.Close()
	w.Header().Set("Vary", "Accept-Encoding")
	if strings.Contains(r.Header.Get("Accept-Encoding"), "br") {
		w.Header().Set("Content-Encoding", "br")
		_, _ = io.Copy(w, fi)
	} else {
		_, _ = io.Copy(w, brotli.NewReader(fi))
	}
}
