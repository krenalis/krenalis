// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/test/syntheticworldtest"
)

// TestSimulatedSystemExternalPhotoGeneration checks the external runtime path.
func TestSimulatedSystemExternalPhotoGeneration(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	assets := syntheticworldtest.Assets(t)
	var calls atomic.Int32
	handler := syntheticworldtest.Handler(t, assets, "/assets")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		handler.ServeHTTP(w, r)
	}))
	defer server.Close()
	k := krenalistester.NewKrenalisInstance(t)
	k.SetExternalAssetsURLs([]string{server.URL + "/assets"})
	testSimulatedSystemGeneration(t, k, server.URL+"/assets/fixture/256/")
	// The shared scenario restarts three times and creates several systems per process.
	if calls.Load() != 16 {
		t.Fatalf("expected four JSON requests per process across three restarts, got %d", calls.Load())
	}

}
