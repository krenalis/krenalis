// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"encoding/json"
	"testing"
	"time"
)

// TestEmptyPipelineMetricsHasEmptyMetricsList verifies that empty metrics
// responses encode the metrics list as [] instead of null.
func TestEmptyPipelineMetricsHasEmptyMetricsList(t *testing.T) {
	start := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	metrics := emptyPipelineMetrics(start, end)
	if metrics.Metrics == nil {
		t.Fatal("expected non-nil metrics list, got nil")
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) {
		t.Fatalf("expected valid JSON, got %s", data)
	}
	if string(data) == `{"start":"2026-07-13T12:00:00Z","end":"2026-07-13T13:00:00Z","metrics":null}` {
		t.Fatal("expected metrics list to encode as [], got null")
	}
	if string(data) != `{"start":"2026-07-13T12:00:00Z","end":"2026-07-13T13:00:00Z","metrics":[]}` {
		t.Fatalf("expected JSON with empty metrics list, got %s", data)
	}
}

// TestValidatePipelineMetricsScopeRequiresOneGroup verifies that metrics
// requests must specify exactly one grouping dimension.
func TestValidatePipelineMetricsScopeRequiresOneGroup(t *testing.T) {
	tests := []struct {
		name  string
		scope PipelineMetricsScope
	}{
		{
			name:  "missing group",
			scope: PipelineMetricsScope{},
		},
		{
			name: "multiple groups",
			scope: PipelineMetricsScope{
				Connections: []string{"7QaT3mN7KxP5"},
				Pipelines:   []string{"8QaT3mN7KxP5"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePipelineMetricsScope(test.scope)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

// TestValidatePipelineMetricsScopeAllowsWorkspaceGroup verifies that workspace
// grouping is valid when workspaces are provided as the grouping parameter.
func TestValidatePipelineMetricsScopeAllowsWorkspaceGroup(t *testing.T) {
	err := validatePipelineMetricsScope(PipelineMetricsScope{
		Workspaces: []string{"9RbU4nP8LyQ6"},
	})
	if err != nil {
		t.Fatalf("expected workspace group to be valid, got %v", err)
	}
}
