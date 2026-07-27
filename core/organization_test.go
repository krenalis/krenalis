// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestPipelineMetricsPerDateRejectsTooManyEntryDays verifies that requests
// exceeding the maximum product between the number of entries in the selection
// and the number of days in the date range are rejected.
func TestPipelineMetricsPerDateRejectsTooManyEntryDays(t *testing.T) {
	organization := Organization{core: &Core{}}
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		days    int
		entries int
	}{
		{name: "61 days and 1,000 entries", days: 61, entries: 1000},
		{name: "121 days and 500 entries", days: 121, entries: 500},
		{name: "20,001 days and 3 entries", days: 20001, entries: 3},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := organization.PipelineMetricsPerDate(context.Background(), start, start.AddDate(0, 0, test.days), "", MetricSelection{
				Workspaces: validMetricIDs(test.entries),
				Target:     TargetEvent,
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "requested metrics exceed the maximum") {
				t.Fatalf("expected entry-day limit error, got %v", err)
			}
		})
	}
}

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
			_, err := validateMetricsSelection(test.selection)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

// TestValidatePipelineMetricsSelectionAllowsWorkspaceGroup verifies that workspace
// grouping is valid when a target is provided.
func TestValidatePipelineMetricsSelectionAllowsWorkspaceGroup(t *testing.T) {
	entries, err := validateMetricsSelection(MetricSelection{
		Workspaces: []string{"9RbU4nP8LyQ6"},
		Target:     TargetEvent,
	})
	if err != nil {
		t.Fatalf("expected workspace group to be valid, got %v", err)
	}
	if entries != 1 {
		t.Fatalf("expected 1 selection entry, got %d", entries)
	}
}

// TestValidatePipelineMetricsSelectionTargetRequirements verifies that target is
// optional for pipeline selections and required for workspace and connection
// selections.
func TestValidatePipelineMetricsSelectionTargetRequirements(t *testing.T) {
	tests := []struct {
		name      string
		selection MetricSelection
		wantErr   bool
	}{
		{
			name: "pipelines without target",
			selection: MetricSelection{
				Pipelines: []string{"9RbU4nP8LyQ6"},
			},
		},
		{
			name: "workspaces without target",
			selection: MetricSelection{
				Workspaces: []string{"9RbU4nP8LyQ6"},
			},
			wantErr: true,
		},
		{
			name: "connections without target",
			selection: MetricSelection{
				Connections: []string{"9RbU4nP8LyQ6"},
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entries, err := validateMetricsSelection(test.selection)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "target is required") {
					t.Fatalf("expected missing target error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected selection to be valid, got %v", err)
			}
			if entries != 1 {
				t.Fatalf("expected 1 selection entry, got %d", entries)
			}
		})
	}
}

// TestValidatePipelineMetricsSelectionAllowsMaximumGroupSize verifies that the
// selected grouping dimension may contain up to 1,000 identifiers.
func TestValidatePipelineMetricsSelectionAllowsMaximumGroupSize(t *testing.T) {
	tests := []struct {
		name      string
		selection MetricSelection
	}{
		{
			name: "workspaces",
			selection: MetricSelection{
				Workspaces: validMetricIDs(1000),
				Target:     TargetEvent,
			},
		},
		{
			name: "connections",
			selection: MetricSelection{
				Connections: validMetricIDs(1000),
				Target:      TargetEvent,
			},
		},
		{
			name: "pipelines",
			selection: MetricSelection{
				Pipelines: validMetricIDs(1000),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entries, err := validateMetricsSelection(test.selection)
			if err != nil {
				t.Fatalf("expected selection to be valid, got %v", err)
			}
			if entries != 1000 {
				t.Fatalf("expected 1,000 selection entries, got %d", entries)
			}
		})
	}
}

// TestValidatePipelineMetricsSelectionRejectsOversizedGroup verifies that the
// selected grouping dimension cannot contain more than 1,000 identifiers.
func TestValidatePipelineMetricsSelectionRejectsOversizedGroup(t *testing.T) {
	tests := []struct {
		name      string
		selection MetricSelection
	}{
		{
			name: "workspaces",
			selection: MetricSelection{
				Workspaces: validMetricIDs(1001),
			},
		},
		{
			name: "connections",
			selection: MetricSelection{
				Connections: validMetricIDs(1001),
			},
		},
		{
			name: "pipelines",
			selection: MetricSelection{
				Pipelines: validMetricIDs(1001),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validateMetricsSelection(test.selection)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "must contain at most 1,000 entries") {
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
			_, err := validateMetricsSelection(test.selection)
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
