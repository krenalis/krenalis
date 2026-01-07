// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package util

import (
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"
)

// Test_ParseTime ensures ParseTime correctly handles valid and invalid inputs.
func Test_ParseTime(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Time
		ok    bool
	}{
		{
			name:  "basic",
			input: "00:00:00",
			want:  time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			ok:    true,
		},
		{
			name:  "fractional",
			input: "12:34:56.789",
			want:  time.Date(1970, 1, 1, 12, 34, 56, 789000000, time.UTC),
			ok:    true,
		},
		{
			name:  "nine digits",
			input: "23:59:59.123456789",
			want:  time.Date(1970, 1, 1, 23, 59, 59, 123456789, time.UTC),
			ok:    true,
		},
		{
			name:  "trailing characters",
			input: "03:04:05.1extra",
			want:  time.Date(1970, 1, 1, 3, 4, 5, 100000000, time.UTC),
			ok:    true,
		},
		{
			name:  "ten digits",
			input: "03:04:05.1234567890",
			want:  time.Date(1970, 1, 1, 3, 4, 5, 123456789, time.UTC),
			ok:    true,
		},
		{
			name:  "short input",
			input: "12:34",
			ok:    false,
		},
		{
			name:  "invalid hour",
			input: "24:00:00",
			ok:    false,
		},
		{
			name:  "invalid minute",
			input: "12:60:00",
			ok:    false,
		},
		{
			name:  "invalid second",
			input: "12:00:60",
			ok:    false,
		},
		{
			name:  "no fractional digits",
			input: "12:34:56.",
			ok:    false,
		},
		{
			name:  "non digit fractional",
			input: "12:34:56.a",
			ok:    false,
		},
		{
			name:  "nondigits",
			input: "aa:bb:cc",
			ok:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseTime(tt.input)
			if ok != tt.ok {
				t.Fatalf("expected ok=%t, got %t", tt.ok, ok)
			}
			if tt.ok && !got.Equal(tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}

	// Also test byte slice input.
	t.Run("bytes", func(t *testing.T) {
		want := time.Date(1970, 1, 1, 3, 4, 5, 100000000, time.UTC)
		got, ok := ParseTime([]byte("03:04:05.100"))
		if !ok {
			t.Fatal("ParseTime returned false for valid bytes")
		}
		if !got.Equal(want) {
			t.Fatalf("expected %v, got %v", want, got)
		}
	})
}

// Test_PropertiesToColumns verifies that nested object properties are flattened
// into columns and that nullability is preserved.
func Test_PropertiesToColumns(t *testing.T) {
	int32Type := types.Int(32)

	cases := []struct {
		name string
		typ  types.Type
		want []warehouses.Column
	}{
		{
			name: "flat",
			typ: types.Object([]types.Property{
				{Name: "id", Type: int32Type},
				{Name: "name", Type: types.String(), Nullable: true},
			}),
			want: []warehouses.Column{
				{Name: "id", Type: int32Type},
				{Name: "name", Type: types.String(), Nullable: true},
			},
		},
		{
			name: "nested",
			typ: types.Object([]types.Property{
				{Name: "info", Type: types.Object([]types.Property{
					{Name: "age", Type: int32Type},
					{Name: "email", Type: types.String(), Nullable: true},
				})},
				{Name: "title", Type: types.String()},
			}),
			want: []warehouses.Column{
				{Name: "info_age", Type: int32Type},
				{Name: "info_email", Type: types.String(), Nullable: true},
				{Name: "title", Type: types.String()},
			},
		},
		{
			name: "multi level with underscores",
			typ: types.Object([]types.Property{
				{Name: "outer_", Type: types.Object([]types.Property{
					{Name: "inner_value", Type: types.Boolean()},
				})},
				{Name: "x", Type: types.Int(8), Nullable: true},
			}),
			want: []warehouses.Column{
				{Name: "outer__inner_value", Type: types.Boolean()},
				{Name: "x", Type: types.Int(8), Nullable: true},
			},
		},
		{
			name: "deep nesting",
			typ: types.Object([]types.Property{
				{Name: "a", Type: types.Object([]types.Property{
					{Name: "b", Type: types.Object([]types.Property{
						{Name: "c", Type: types.Int(16)},
					})},
				})},
			}),
			want: []warehouses.Column{{Name: "a_b_c", Type: types.Int(16)}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := PropertiesToColumns(tc.typ.Properties())
			if len(got) != len(tc.want) {
				t.Fatalf("expected %d columns, got %d", len(tc.want), len(got))
			}
			for i, g := range got {
				w := tc.want[i]
				if g.Name != w.Name {
					t.Fatalf("column %d: expected name %q, got %q", i, w.Name, g.Name)
				}
				if !types.Equal(g.Type, w.Type) {
					t.Fatalf("column %d: unexpected type", i)
				}
				if g.Nullable != w.Nullable {
					t.Fatalf("column %d: expected nullable %t, got %t", i, w.Nullable, g.Nullable)
				}
			}
		})
	}
}

// Test_ValidateStringField ensures ValidateStringField correctly validates a
// field.
func Test_ValidateStringField(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		maxLen  int
		wantErr bool
		errSub  string
	}{
		{
			name:    "Empty string",
			input:   "",
			maxLen:  5,
			wantErr: true,
			errSub:  "is empty",
		},
		{
			name:    "Invalid UTF-8",
			input:   string([]byte{0xff, 0xfe, 0xfd}),
			maxLen:  5,
			wantErr: true,
			errSub:  "invalid UTF-8",
		},
		{
			name:    "Contains NUL byte",
			input:   "foo\x00bar",
			maxLen:  10,
			wantErr: true,
			errSub:  "contains the NUL byte",
		},
		{
			name:    "Too many runes",
			input:   "abcdef",
			maxLen:  3,
			wantErr: true,
			errSub:  "longer than",
		},
		{
			name:    "Valid short string",
			input:   "abc世𠜎",
			maxLen:  10,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStringField("field", tt.input, tt.maxLen)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errSub) {
					t.Errorf("wrong error message: got %q, want it to contain %q", err.Error(), tt.errSub)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
			}
		})
	}
}
