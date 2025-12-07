// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"slices"
	"testing"
)

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
