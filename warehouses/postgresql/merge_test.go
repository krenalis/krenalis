//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b

package postgresql

import (
	"reflect"
	"testing"
)

// Test_copyForDeleteFrom feeds the helper used by DELETE FROM into various key
// combinations and compares the generated rows.
func Test_copyForDeleteFrom(t *testing.T) {
	tests := []struct {
		name     string
		numKeys  int
		keys     []any
		expected [][]any
	}{
		{
			name:    "two_keys_two_rows",
			numKeys: 2,
			keys:    []any{1, "a", 2, "b"},
			expected: [][]any{
				{1, "a", true},
				{2, "b", true},
			},
		},
		{
			name:    "single_key_three_rows",
			numKeys: 1,
			keys:    []any{"x", "y", "z"},
			expected: [][]any{
				{"x", true},
				{"y", true},
				{"z", true},
			},
		},
		{
			name:    "three_keys_two_rows",
			numKeys: 3,
			keys:    []any{1, 2, 3, 4, 5, 6},
			expected: [][]any{
				{1, 2, 3, true},
				{4, 5, 6, true},
			},
		},
		{
			name:     "no_keys",
			numKeys:  2,
			keys:     []any{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCopyForDeleteFrom(tt.numKeys, tt.keys)
			var rows [][]any
			for c.Next() {
				row, err := c.Values()
				if err != nil {
					t.Fatal(err)
				}
				cp := make([]any, len(row))
				copy(cp, row)
				rows = append(rows, cp)
			}
			if err := c.Err(); err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if !reflect.DeepEqual(rows, tt.expected) {
				t.Fatalf("expected %v, got %v", tt.expected, rows)
			}
		})
	}
}
