// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/krenalis/krenalis/core"
)

// TestSplitQueryParameters verifies comma-separated query parameter splitting.
func TestSplitQueryParameters(t *testing.T) {
	tests := []struct {
		name      string
		in        []string
		want      []string
		sameSlice bool
	}{
		{
			name: "nil slice",
			in:   nil,
			want: nil,
		},
		{
			name: "empty slice",
			in:   []string{},
			want: nil,
		},
		{
			name:      "single value without comma",
			in:        []string{"foo"},
			want:      []string{"foo"},
			sameSlice: true,
		},
		{
			name:      "multiple values without commas",
			in:        []string{"a", "b", "c"},
			want:      []string{"a", "b", "c"},
			sameSlice: true,
		},
		{
			name: "single value with commas",
			in:   []string{"a,b,c"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "mixed plain and comma-separated values",
			in:   []string{"foo", "bar,baz"},
			want: []string{"foo", "bar", "baz"},
		},
		{
			name: "values with spaces around commas",
			in:   []string{" x , y , z "},
			want: []string{"x", "y", "z"},
		},
		{
			name: "values with mixed alphanumeric segments",
			in:   []string{"a1,b2,c3"},
			want: []string{"a1", "b2", "c3"},
		},
		{
			name: "values with empty segments",
			in:   []string{",foo,,bar,"},
			want: []string{"foo", "bar"},
		},
		{
			name: "only empty or whitespace segments",
			in:   []string{", , ,", "\t", "\n", " ", ","},
			want: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := splitQueryParameters(test.in)
			if !slices.Equal(test.want, got) {
				t.Errorf("%v: expected %v, got %v", test.in, test.want, got)
			}
		})
	}
}

// TestParsePipelineMetricsSelectionConnectionTarget verifies connection and
// target query parameters for pipeline metrics requests.
func TestParsePipelineMetricsSelectionConnectionTarget(t *testing.T) {

	const connection = "7QaT3mN7KxP5"

	tests := []struct {
		name                   string
		query                  string
		wantConnections        []string
		wantConnectionsPresent bool
		wantTarget             core.Target
		wantErr                bool
	}{
		{
			name:                   "connections and user target",
			query:                  "?connections=" + connection + "&target=User",
			wantConnections:        []string{connection},
			wantConnectionsPresent: true,
			wantTarget:             core.TargetUser,
		},
		{
			name:                   "connections and event target",
			query:                  "?connections=" + connection + "&target=Event",
			wantConnections:        []string{connection},
			wantConnectionsPresent: true,
			wantTarget:             core.TargetEvent,
		},
		{
			name:                   "connections without target",
			query:                  "?connections=" + connection,
			wantConnections:        []string{connection},
			wantConnectionsPresent: true,
			wantTarget:             core.TargetNone,
		},
		{
			name:       "target without grouping",
			query:      "?target=Event",
			wantTarget: core.TargetEvent,
		},
		{
			name:                   "empty connections parameter",
			query:                  "?connections=",
			wantConnections:        []string{},
			wantConnectionsPresent: true,
			wantTarget:             core.TargetNone,
		},
		{
			name:    "invalid target",
			query:   "?target=Group",
			wantErr: true,
		},
		{
			name:    "empty target",
			query:   "?target=",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/pipelines/metrics/minutes/15"+test.query, nil)
			selection, err := parseMetricsSelection(req)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if test.wantConnectionsPresent && selection.Connections == nil {
				t.Fatal("expected non-nil connections, got nil")
			}
			if !slices.Equal(selection.Connections, test.wantConnections) {
				t.Fatalf("expected connections %v, got %v", test.wantConnections, selection.Connections)
			}
			if selection.Target != test.wantTarget {
				t.Fatalf("expected target %q, got %q", test.wantTarget, selection.Target)
			}
		})
	}

}

// TestParsePipelineMetricsSelectionWorkspaces verifies that workspace grouping
// is parsed from the query parameters without an implicit workspace scope.
func TestParsePipelineMetricsSelectionWorkspaces(t *testing.T) {

	const workspace = "9RbU4nP8LyQ6"

	req := httptest.NewRequest("GET", "/pipelines/metrics/minutes/15?workspaces="+workspace, nil)
	selection, err := parseMetricsSelection(req)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(selection.Workspaces, []string{workspace}) {
		t.Fatalf("expected workspaces %v, got %v", []string{workspace}, selection.Workspaces)
	}

}
