// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import "testing"

func TestTransformationEqual(t *testing.T) {
	fn1 := &TransformationFunction{ID: "f1", Version: "v1"}
	fn1copy := &TransformationFunction{ID: "f1", Version: "v1"}
	fn2 := &TransformationFunction{ID: "f2", Version: "v1"}

	tests := []struct {
		name  string
		t1    Transformation
		t2    Transformation
		equal bool
	}{
		{
			name:  "nil functions equal mapping",
			t1:    Transformation{Mapping: map[string]string{"a": "b"}, InPaths: []string{"b"}, OutPaths: []string{"a"}},
			t2:    Transformation{Mapping: map[string]string{"a": "b"}, InPaths: []string{"b"}, OutPaths: []string{"a"}},
			equal: true,
		},
		{
			name:  "mapping differs when functions nil",
			t1:    Transformation{Mapping: map[string]string{"a": "b"}},
			t2:    Transformation{Mapping: map[string]string{"a": "c"}},
			equal: false,
		},
		{
			name:  "functions equals",
			t1:    Transformation{Function: fn1},
			t2:    Transformation{Function: fn1copy},
			equal: true,
		},
		{
			name:  "functions differ",
			t1:    Transformation{Function: fn1},
			t2:    Transformation{Function: fn2},
			equal: false,
		},
		{
			name:  "in paths differ",
			t1:    Transformation{Function: fn1, InPaths: []string{"a"}},
			t2:    Transformation{Function: fn1copy, InPaths: []string{"b"}},
			equal: false,
		},
		{
			name:  "out paths differ",
			t1:    Transformation{Function: fn1, OutPaths: []string{"a"}},
			t2:    Transformation{Function: fn1copy, OutPaths: []string{"b"}},
			equal: false,
		},
		{
			name:  "everything equal",
			t1:    Transformation{Function: fn1, InPaths: []string{"in"}, OutPaths: []string{"out"}},
			t2:    Transformation{Function: fn1copy, InPaths: []string{"in"}, OutPaths: []string{"out"}},
			equal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.t1.Equal(tt.t2) != tt.equal {
				t.Fatalf("expected %v", tt.equal)
			}
			if tt.t2.Equal(tt.t1) != tt.equal {
				t.Fatalf("expected symmetry %v", tt.equal)
			}
		})
	}
}
