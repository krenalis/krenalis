// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
)

// TestSetPipelineStatus tests the endpoint that sets the status of a pipeline.
func TestSetPipelineStatus(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	jsSrc := k.CreateJavaScriptSource("JavaScript (source)", nil)
	pipeline := k.CreatePipeline(jsSrc, "Event", krenalistester.PipelineToSet{
		Name:    "Store events",
		Enabled: true,
	})
	statusPath := fmt.Sprintf("/v1/pipelines/%s/status", pipeline)

	// Test that the call fails if the "enabled" field is missing or null.
	for _, body := range []map[string]any{{}, {"enabled": nil}} {
		err := k.TryCall("PUT", statusPath, nil, body, nil)
		if err != nil {
			statusErr, ok := err.(*krenalistester.StatusCodeError)
			if !ok {
				t.Fatalf("expected *StatusCodeError, got %T: %v", err, err)
			}
			if statusErr.Response.Code != http.StatusBadRequest {
				t.Fatalf("expected HTTP status %d, got %d: %s", http.StatusBadRequest, statusErr.Response.Code, statusErr.Response.Text)
			}
			continue
		}
		t.Fatalf("expected an error for body %v, got nil", body)
	}

	// Test that the call disables the pipeline if "enabled" is false.
	k.Call("PUT", statusPath, nil, map[string]any{"enabled": false}, nil)
	var p struct {
		Enabled bool `json:"enabled"`
	}
	k.Call("GET", fmt.Sprintf("/v1/pipelines/%s", pipeline), nil, nil, &p)
	if p.Enabled {
		t.Fatal("expected the pipeline to be disabled, got enabled")
	}

}
