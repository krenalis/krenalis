// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"strings"
	"testing"
)

// Test_quoteIdent verifies that identifiers containing quotes are quoted
// properly for Snowflake.
func Test_quoteIdent(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"simple", "\"SIMPLE\""},
		{"a\"b", "\"A\"\"B\""},
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
		{"", `''`},
		{"'", `'\''`},    // one single quote
		{"\"", `'"'`},    // one double quote
		{"''", `'\'\''`}, // two single quotes
		{"\"\"", `'""'`}, // two double quotes
		{"\x00", `'\0'`},
		{"hello", `'hello'`},
		{"_+\tè+^", `'_+\tè+^'`},
		{"paul's car", `'paul\'s car'`},
		{"hello world", `'hello world'`},
		{"hello\x00world", `'hello\0world'`},
		{"\x00\x00\x00\x00", `'\0\0\0\0'`},
		{"\x00'\x00a''\x00''", `'\0\'\0a\'\'\0\'\''`},
		{"\b\f\n\r", `'\b\f\n\r'`},
		{"a\\b", `'a\\b'`},
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

// Test_quoteStringForDynamicSQL verifies quoting for nested SQL strings.
func Test_quoteStringForDynamicSQL(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			name: "empty",
			want: "''''",
		},
		{
			name: "base58 identifier",
			s:    "6NpT4zB8QaR2",
			want: "''6NpT4zB8QaR2''",
		},
		{
			name: "single quote",
			s:    "paul's car",
			want: "''paul\\''s car''",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got strings.Builder
			quoteStringForDynamicSQL(&got, test.s)
			if got.String() != test.want {
				t.Fatalf("expected %q, got %q", test.want, got.String())
			}
		})
	}
}
