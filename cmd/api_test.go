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

// TestParsePipelineMetricsScopeConnectionTarget verifies connection and target
// query parameters for pipeline metrics requests.
func TestParsePipelineMetricsScopeConnectionTarget(t *testing.T) {
	const validConnection = "7QaT3mN7KxP5"
	const validPipeline = "8QaT3mN7KxP5"
	const validWorkspace = "9RbU4nP8LyQ6"

	tests := []struct {
		name            string
		query           string
		wantConnections []string
		wantTarget      *core.Target
		wantErr         bool
	}{
		{
			name:            "connections and user target",
			query:           "?connections=" + validConnection + "&target=User",
			wantConnections: []string{validConnection},
			wantTarget:      new(core.TargetUser),
		},
		{
			name:            "connections and event target",
			query:           "?connections=" + validConnection + "&target=Event",
			wantConnections: []string{validConnection},
			wantTarget:      new(core.TargetEvent),
		},
		{
			name:    "target without grouping",
			query:   "?target=Event",
			wantErr: true,
		},
		{
			name:    "missing grouping",
			query:   "",
			wantErr: true,
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
		{
			name:    "connections with pipelines",
			query:   "?connections=" + validConnection + "&pipelines=" + validPipeline,
			wantErr: true,
		},
		{
			name:    "connections with workspaces",
			query:   "?connections=" + validConnection + "&workspaces=" + validWorkspace,
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/pipelines/metrics/minutes/15"+test.query, nil)
			scope, err := parsePipelineMetricsScope(req, nil)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(scope.Connections, test.wantConnections) {
				t.Fatalf("expected connections %v, got %v", test.wantConnections, scope.Connections)
			}
			if test.wantTarget == nil {
				if scope.Target != nil {
					t.Fatalf("expected nil target, got %s", scope.Target.String())
				}
				return
			}
			if scope.Target == nil || *scope.Target != *test.wantTarget {
				t.Fatalf("expected target %s, got %v", test.wantTarget.String(), scope.Target)
			}
		})
	}
}

// TestParsePipelineMetricsScopeWorkspaceScopedWorkspaces verifies that
// workspace grouping is still parsed when the request is already scoped to a
// workspace.
func TestParsePipelineMetricsScopeWorkspaceScopedWorkspaces(t *testing.T) {
	const validWorkspace = "9RbU4nP8LyQ6"

	req := httptest.NewRequest("GET", "/pipelines/metrics/minutes/15?workspaces="+validWorkspace, nil)
	scope, err := parsePipelineMetricsScope(req, &core.Workspace{ID: validWorkspace})
	if err != nil {
		t.Fatal(err)
	}
	if scope.Workspace != validWorkspace {
		t.Fatalf("expected workspace scope %q, got %q", validWorkspace, scope.Workspace)
	}
	if !slices.Equal(scope.Workspaces, []string{validWorkspace}) {
		t.Fatalf("expected workspaces %v, got %v", []string{validWorkspace}, scope.Workspaces)
	}
}

// targetPtr returns a pointer to v.
//
//go:fix inline
func targetPtr(v core.Target) *core.Target {
	return new(v)
}
