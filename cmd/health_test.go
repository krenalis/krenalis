// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"net/http/httptest"
	"testing"
)

func Test_serveLivez(t *testing.T) {
	tests := []struct {
		method string
		status int
		body   string
	}{
		{"GET", 200, "ok"},
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
