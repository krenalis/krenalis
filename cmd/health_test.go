// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	corePkg "github.com/krenalis/krenalis/core"
)

type fakeHealthChecker []corePkg.HealthCheck

func (f fakeHealthChecker) Health(context.Context) []corePkg.HealthCheck {
	return f
}

// The recorder does not discard the body of HEAD responses as the HTTP server
// does, so for HEAD the expected body is the one written by the handlers.
func Test_serveLivez(t *testing.T) {
	tests := []struct {
		method string
		status int
		body   string
	}{
		{"GET", 200, "ok"},
		{"HEAD", 200, "ok"},
		{"POST", 405, "Method Not Allowed\n"},
	}
	for _, test := range tests {
		t.Run(test.method, func(t *testing.T) {
			w := httptest.NewRecorder()
			serveLivez(w, httptest.NewRequest(test.method, "/livez", nil))
			if w.Code != test.status {
				t.Fatalf("expected status %d, got %d", test.status, w.Code)
			}
			if w.Body.String() != test.body {
				t.Fatalf("expected body %q, got %q", test.body, w.Body.String())
			}
		})
	}
}

func Test_readyzHandler(t *testing.T) {
	ok := fakeHealthChecker{{Name: "postgres", OK: true}, {Name: "nats", OK: true}}
	tests := []struct {
		name       string
		checker    fakeHealthChecker
		isShutdown bool
		method     string
		target     string
		status     int
		body       string
	}{
		{"ok", ok, false, "GET", "/readyz", 200, "ok"},
		{"verbose", ok, false, "GET", "/readyz?verbose", 200,
			"[+]postgres ok\n[+]nats ok\n[+]shutdown ok\nreadyz check passed"},
		{"head", ok, false, "HEAD", "/readyz", 200, "ok"},
		{"postgres down", fakeHealthChecker{{Name: "postgres"}, {Name: "nats", OK: true}}, false, "GET", "/readyz", 503,
			"[-]postgres failed\n[+]nats ok\n[+]shutdown ok\nreadyz check failed"},
		{"nats down", fakeHealthChecker{{Name: "postgres", OK: true}, {Name: "nats"}}, false, "GET", "/readyz", 503,
			"[+]postgres ok\n[-]nats failed\n[+]shutdown ok\nreadyz check failed"},
		{"shutting down", ok, true, "GET", "/readyz", 503,
			"[+]postgres ok\n[+]nats ok\n[-]shutdown failed\nreadyz check failed"},
		{"post", ok, false, "POST", "/readyz", 405, "Method Not Allowed\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var isShuttingDown atomic.Bool
			isShuttingDown.Store(test.isShutdown)
			handler := newReadyzHandler(test.checker, &isShuttingDown)
			w := httptest.NewRecorder()
			handler(w, httptest.NewRequest(test.method, test.target, nil))
			if w.Code != test.status {
				t.Fatalf("expected status %d, got %d", test.status, w.Code)
			}
			if w.Body.String() != test.body {
				t.Fatalf("expected body %q, got %q", test.body, w.Body.String())
			}
			if test.status != http.StatusMethodNotAllowed && w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("expected Cache-Control %q, got %q", "no-store", w.Header().Get("Cache-Control"))
			}
		})
	}
}
