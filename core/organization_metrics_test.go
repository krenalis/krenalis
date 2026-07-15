// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"testing"
)

// TestValidatePipelineMetricsSelectionRequiresOneGroup verifies that metrics
// requests must specify exactly one grouping dimension.
func TestValidatePipelineMetricsSelectionRequiresOneGroup(t *testing.T) {
	tests := []struct {
		name      string
		selection MetricSelection
	}{
		{
			name:      "missing group",
			selection: MetricSelection{},
		},
		{
			name: "multiple groups",
			selection: MetricSelection{
				Connections: []string{"7QaT3mN7KxP5"},
				Pipelines:   []string{"8QaT3mN7KxP5"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateMetricsSelection(test.selection)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

// TestValidatePipelineMetricsSelectionAllowsWorkspaceGroup verifies that workspace
// grouping is valid when workspaces are provided as the grouping parameter.
func TestValidatePipelineMetricsSelectionAllowsWorkspaceGroup(t *testing.T) {
	err := validateMetricsSelection(MetricSelection{
		Workspaces: []string{"9RbU4nP8LyQ6"},
	})
	if err != nil {
		t.Fatalf("expected workspace group to be valid, got %v", err)
	}
}
