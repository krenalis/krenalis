// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/andybalholm/brotli"
)

// TestAssetsHandler_Index verifies that the production assets handler correctly
// serves the index page and handles Brotli compression when the client requests
// it.
func TestAssetsHandler_Index(t *testing.T) {

	html := "<html>hello</html>"
	fsys := newMapFS(map[string]string{"index.html.br": html})

	a, err := newAdmin(fsys)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	req := httptest.NewRequest("GET", "/admin", nil)
	rr := httptest.NewRecorder()
	a.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Header().Get("Content-Encoding") != "" {
		t.Fatalf("unexpected content encoding: %s", rr.Header().Get("Content-Encoding"))
	}
	if got := rr.Body.String(); got != html {
		t.Fatalf("expected %q, got %q", html, got)
	}

	req = httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Accept-Encoding", "br")
	rr = httptest.NewRecorder()
	a.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Header().Get("Content-Encoding") != "br" {
		t.Fatalf("missing brotli encoding")
	}
	if !bytes.Equal(rr.Body.Bytes(), brotliCompress([]byte(html))) {
		t.Fatalf("unexpected compressed body")
	}

}

// TestAssetsHandler_JS checks that JavaScript assets are served with the proper
// content type and without compression when the client doesn't request Brotli.
func TestAssetsHandler_JS(t *testing.T) {

	js := "console.log('x')"
	fsys := newMapFS(map[string]string{"index.js.br": js})
	a, err := newAdmin(fsys)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	req := httptest.NewRequest("GET", "/admin/src/index.js", nil)
	rr := httptest.NewRecorder()
	a.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/x-javascript" {
		t.Fatalf("unexpected content type: %s", ct)
	}
	if rr.Header().Get("Content-Encoding") != "" {
		t.Fatalf("unexpected content encoding: %s", rr.Header().Get("Content-Encoding"))
	}
	if rr.Body.String() != js {
		t.Fatalf("unexpected body: %q", rr.Body.String())
	}

}

func brotliCompress(data []byte) []byte {
	var b bytes.Buffer
	w := brotli.NewWriter(&b)
	w.Write(data)
	w.Close()
	return b.Bytes()
}

// newMapFS creates an fstest.MapFS from a map of filenames to contents.
func newMapFS(files map[string]string) fs.FS {
	m := make(fstest.MapFS)
	for name, content := range files {
		m[name] = &fstest.MapFile{Data: brotliCompress([]byte(content))}
	}
	return m
}
