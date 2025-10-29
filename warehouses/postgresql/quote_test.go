// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"strings"
	"testing"
)

// Test_quoteIdent verifies that identifiers containing quotes are quoted
// properly for PostgreSQL.
func Test_quoteIdent(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"simple", "\"simple\""},
		{"a\"b", "\"a\"\"b\""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quoteIdent(tt.name)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

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
