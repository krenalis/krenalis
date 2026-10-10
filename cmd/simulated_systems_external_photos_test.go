// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
)

// TestSimulatedSystemExternalPhotoGeneration checks the external runtime path.
func TestSimulatedSystemExternalPhotoGeneration(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	fixture, err := os.ReadFile("../tools/fakedata/testdata/external-photos-v1.json")
	if err != nil {
		t.Fatalf("expected photo fixture, got %v", err)
	}
	model := simulatedSystemModelFixture(t)
	var catalogCalls, modelCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/assets/synthetic-world/it-2026a/model.json" {
			modelCalls.Add(1)
			_, _ = w.Write(model)
			return
		}
		if r.URL.Path != "/assets/synthetic-world/photos/v1/catalog.json" {
			t.Errorf("expected catalog or model request only, got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		catalogCalls.Add(1)
		_, _ = w.Write(fixture)
	}))
	defer server.Close()
	k := krenalistester.NewKrenalisInstance(t)
	k.SetExternalAssetsURLs([]string{server.URL + "/assets"})
	testSimulatedSystemGeneration(t, k, server.URL+"/assets/synthetic-world/photos/v1/256/")
	// The shared scenario restarts twice and creates several systems per process.
	if catalogCalls.Load() != 3 {
		t.Fatalf("expected one catalog fetch per process across two restarts, got %d", catalogCalls.Load())
	}
	if modelCalls.Load() != 3 {
		t.Fatalf("expected one shared model fetch per process across two restarts, got %d", modelCalls.Load())
	}

}
