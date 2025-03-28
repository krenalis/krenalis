//go:build !dev

//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	_ "embed"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
)

// assetsHandler implements a http.Handler to serve UI assets. It serves bundle
// files that are embedded in the executable, compressed with Brotli if the
// client supports it.
type assetsHandler struct {
	fsys fs.FS
}

func newAssetsHandler(fsys fs.FS) (*assetsHandler, error) {
	return &assetsHandler{fsys}, nil
}

func (h *assetsHandler) Close() {}

func (h *assetsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var asset string
	if path, ok := strings.CutPrefix(r.URL.Path, "/ui/src/"); ok {
		switch path {
		case "index.js":
			w.Header().Add("Content-Type", "text/javascript")
		case "index.js.map":
			w.Header().Add("Content-Type", "application/json")
		case "index.css":
			w.Header().Add("Content-Type", "text/css")
		case "index.css.map":
			w.Header().Add("Content-Type", "application/json")
		default:
			if !strings.HasSuffix(path, ".svg") {
				http.NotFound(w, r)
				return
			}
			w.Header().Add("Content-Type", "image/svg+xml")
		}
		asset = path + ".br"
	} else {
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
