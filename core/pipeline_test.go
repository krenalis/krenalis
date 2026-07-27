// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"testing"

	"github.com/krenalis/krenalis/tools/json"
)

// TestTargetMarshalJSON verifies that Target values encode to their public JSON
// representation.
func TestTargetMarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		target Target
		want   string
	}{
		{name: "none", target: TargetNone, want: "null"},
		{name: "event", target: TargetEvent, want: `"Event"`},
		{name: "user", target: TargetUser, want: `"User"`},
		{name: "group", target: TargetGroup, want: `"Group"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := json.Marshal(test.target)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if string(got) != test.want {
				t.Fatalf("expected %s, got %s", test.want, got)
			}
		})
	}
}

// TestTargetUnmarshalJSON verifies that public JSON target values decode to
// Target values.
func TestTargetUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		data string
		want Target
	}{
		{name: "none", data: "null", want: TargetNone},
		{name: "event", data: `"Event"`, want: TargetEvent},
		{name: "user", data: `"User"`, want: TargetUser},
		{name: "group", data: `"Group"`, want: TargetGroup},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got Target
			err := json.Unmarshal([]byte(test.data), &got)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}

// TestTargetUnmarshalJSONRejectsInvalidValues verifies that unsupported target
// values are rejected.
func TestTargetUnmarshalJSONRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "invalid string", data: `"None"`},
		{name: "number", data: "1"},
		{name: "object", data: `{}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got Target
			err := json.Unmarshal([]byte(test.data), &got)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
