//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package db

import (
	"testing"
	"time"
)

// Test_Quote checks that Quote correctly formats different Go values for use
// in SQL queries.
func Test_Quote(t *testing.T) {

	t1 := time.Date(2024, 6, 1, 2, 3, 4, 567890000, time.UTC)
	t2 := time.Date(2024, 6, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name string
		v    any
		exp  string
	}{
		// Nil should become the SQL NULL literal
		{"nil", nil, "NULL"},
		// TRUE and FALSE are returned for booleans
		{"bool true", true, "TRUE"},
		{"bool false", false, "FALSE"},
		// Signed integers are formatted as decimal strings
		{"int", 42, "42"},
		{"int negative", -7, "-7"},
		{"int64", int64(99), "99"},
		{"int32", int32(-12), "-12"},
		// Unsigned integers use decimal formatting
		{"uint", uint(3), "3"},
		{"uint64", uint64(42), "42"},
		{"uint32", uint32(15), "15"},
		// Floats keep their decimal representation
		{"float32", float32(2.5), "2.5"},
		{"float64", 1.125, "1.125"},
		// Simple strings are quoted
		{"string", "foo", "'foo'"},
		// Single quotes inside strings are escaped by doubling them
		{"string with quote", "O'Reilly", "'O''Reilly'"},
		// Potential SQL injection payload is escaped properly
		{"sql injection", "a'; DROP TABLE t; --", "'a''; DROP TABLE t; --'"},
		// time.Time values are formatted with microsecond precision
		{"time", t1, "'2024-06-01 02:03:04.56789'"},
		// Slice of strings are joined with commas and quoted
		{"string slice", []string{"a", "b"}, "('a','b')"},
		{"string slice one", []string{"x"}, "('x')"},
		// Slice containing a quote is escaped
		{"string slice with quote", []string{"O'Reilly", "foo"}, "('O''Reilly','foo')"},
		// Slices of integers use decimal numbers
		{"int slice", []int{1, 2}, "(1,2)"},
		{"int slice one", []int{5}, "(5)"},
		// int64 slice formatting
		{"int64 slice", []int64{1, -2}, "(1,-2)"},
		// time slice is formatted using DateTime layout
		{"time slice", []time.Time{t1, t2}, "('2024-06-01 02:03:04','2024-06-02 03:04:05')"},
		{"time slice one", []time.Time{t2}, "('2024-06-02 03:04:05')"},
		// Mixed slice of supported types
		{"any slice", []any{nil, "a", 3, true, t2}, "(NULL,'a',3,TRUE,'2024-06-02 03:04:05')"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Quote(tc.v)
			if got != tc.exp {
				t.Fatalf("expected %s, got %s", tc.exp, got)
			}
		})
	}

	t.Run("unsupported type", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
		}()
		Quote(struct{}{})
	})

	t.Run("unsupported nested type", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
		}()
		Quote([]any{1, struct{}{}})
	})

}
