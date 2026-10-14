// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"io"
	"net/http"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
)

// TestHealth tests the /livez and /readyz endpoints.
func TestHealth(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	tests := []struct {
		path string
		body string
	}{
		{"/livez", "ok"},
		{"/readyz", "ok"},
		{"/readyz?verbose", "[+]postgres ok\n[+]nats ok\n[+]shutdown ok\nreadyz check passed"},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), "GET", "http://"+k.Addr()+test.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
			}
			if string(body) != test.body {
				t.Fatalf("expected body %q, got %q", test.body, body)
			}
		})
	}

}
