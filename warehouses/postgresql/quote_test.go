//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package postgresql

import (
	"strings"
	"testing"
)

func Test_quoteString(t *testing.T) {
	tests := []struct {
		s        string
		expected string
	}{
		{"", "''"},
		{"'", "''''"},      // one single quote
		{"\"", "'\"'"},     // one double quote
		{"''", "''''''"},   // two single quotes
		{"\"\"", "'\"\"'"}, // two double quotes
		{"\x00", "''"},
		{"hello", "'hello'"},
		{"_+\tè+^", "'_+\tè+^'"},
		{"paul's car", "'paul''s car'"},
		{"hello world", "'hello world'"},
		{"hello\x00world", "'helloworld'"},
		{"\x00\x00\x00\x00", "''"},
		{"\x00'\x00a''\x00''", "'''a'''''''''"},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			var got strings.Builder
			quoteString(&got, test.s)
			if test.expected != got.String() {
				t.Fatalf("expected %q, got %q", test.expected, got.String())
			}
		})
	}
}
