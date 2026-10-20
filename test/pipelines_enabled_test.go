// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
)

// TestPipelinesEnabledOptional verifies that the "enabled" field can be omitted
// when creating or updating a pipeline, and that omitting it disables the
// pipeline.
func TestPipelinesEnabledOptional(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	javaScriptID := k.CreateJavaScriptSource("JavaScript (source)", nil)

	isEnabled := func(id string) bool {
		var pipeline struct {
			Enabled bool `json:"enabled"`
		}
		k.Call("GET", "/v1/pipelines/"+id, nil, nil, &pipeline)
		return pipeline.Enabled
	}

	// Create a pipeline without the "enabled" field.
	var response struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/pipelines", nil, map[string]any{
		"connection": javaScriptID,
		"target":     "Event",
		"name":       "JavaScript events",
	}, &response)
	if isEnabled(response.ID) {
		t.Fatal("expected created pipeline to be disabled, got enabled")
	}

	// Update an enabled pipeline without the "enabled" field.
	id := response.ID
	k.Call("PUT", "/v1/pipelines/"+id+"/status", nil, map[string]any{"enabled": true}, nil)
	if !isEnabled(id) {
		t.Fatal("expected pipeline to be enabled, got disabled")
	}
	k.Call("PUT", "/v1/pipelines/"+id, nil, map[string]any{
		"name": "JavaScript events",
	}, nil)
	if isEnabled(id) {
		t.Fatal("expected updated pipeline to be disabled, got enabled")
	}

}
