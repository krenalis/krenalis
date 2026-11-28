// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/krenalistester"
)

// TestLivez checks that /livez stays available during cleanup until Stop.
func TestLivez(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	t.Cleanup(k.Stop)

	t.Cleanup(func() {
		client := &http.Client{Timeout: 5 * time.Second}
		res, err := client.Get("http://" + k.Addr() + "/livez")
		if err != nil {
			t.Fatalf("expected /livez response during cleanup, got %v", err)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("expected readable /livez response, got %v", err)
		}
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
		}
		if string(body) != "ok" {
			t.Fatalf("expected body %q, got %q", "ok", body)
		}
	})

}
