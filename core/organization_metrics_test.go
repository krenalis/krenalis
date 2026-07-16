// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"strings"
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
			name: "multiple empty groups",
			selection: MetricSelection{
				Workspaces:  []string{},
				Connections: []string{},
				Pipelines:   []string{},
			},
		},
		{
			name: "multiple groups",
			selection: MetricSelection{
				Workspaces:  []string{"9QaT3mN7KxP5"},
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

// TestValidatePipelineMetricsSelectionAllowsMaximumGroupSize verifies that the
// selected grouping dimension may contain up to 100 identifiers.
func TestValidatePipelineMetricsSelectionAllowsMaximumGroupSize(t *testing.T) {
	tests := []struct {
		name      string
		selection MetricSelection
	}{
		{
			name: "workspaces",
			selection: MetricSelection{
				Workspaces: validMetricIDs(100),
			},
		},
		{
			name: "connections",
			selection: MetricSelection{
				Connections: validMetricIDs(100),
			},
		},
		{
			name: "pipelines",
			selection: MetricSelection{
				Pipelines: validMetricIDs(100),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateMetricsSelection(test.selection)
			if err != nil {
				t.Fatalf("expected selection to be valid, got %v", err)
			}
		})
	}
}

// TestValidatePipelineMetricsSelectionRejectsOversizedGroup verifies that the
// selected grouping dimension cannot contain more than 100 identifiers.
func TestValidatePipelineMetricsSelectionRejectsOversizedGroup(t *testing.T) {
	tests := []struct {
		name      string
		selection MetricSelection
	}{
		{
			name: "workspaces",
			selection: MetricSelection{
				Workspaces: validMetricIDs(101),
			},
		},
		{
			name: "connections",
			selection: MetricSelection{
				Connections: validMetricIDs(101),
			},
		},
		{
			name: "pipelines",
			selection: MetricSelection{
				Pipelines: validMetricIDs(101),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateMetricsSelection(test.selection)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "must contain at most 100 entries") {
				t.Fatalf("expected maximum group size error, got %v", err)
			}
		})
	}
}

// TestValidatePipelineMetricsSelectionRejectsEmptyGroup verifies that the
// selected grouping dimension must contain at least one identifier.
func TestValidatePipelineMetricsSelectionRejectsEmptyGroup(t *testing.T) {
	tests := []struct {
		name      string
		selection MetricSelection
	}{
		{
			name: "workspaces",
			selection: MetricSelection{
				Workspaces: []string{},
			},
		},
		{
			name: "connections",
			selection: MetricSelection{
				Connections: []string{},
			},
		},
		{
			name: "pipelines",
			selection: MetricSelection{
				Pipelines: []string{},
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

func validMetricIDs(count int) []string {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	ids := make([]string, count)
	for i := range ids {
		ids[i] = "9RbU4nP8Ly" + string(alphabet[i/len(alphabet)]) + string(alphabet[i%len(alphabet)])
	}
	return ids
}
