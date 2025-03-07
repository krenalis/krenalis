//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package decimal

import (
	"math"
	"testing"
)

func Fuzz_Binary(f *testing.F) {
	tests := []struct {
		b         []byte
		precision int
		scale     int
	}{
		{[]byte{0x00}, 10, 2},
		{[]byte{0x01, 0x02, 0x03}, 10, 2},
		{[]byte{0xFF}, 10, 2},                  // Negative number in two’s complement
		{[]byte{}, 10, 2},                      // Empty input
		{[]byte{0x7F, 0xFF, 0xFF, 0xFF}, 5, 2}, // Out-of-range case
	}
	for _, tc := range tests {
		f.Add(tc.b, tc.precision, tc.scale)
	}
	f.Fuzz(func(t *testing.T, b []byte, precision, scale int) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Binary caused a panic with input: b=%v, precision=%d, scale=%d", b, precision, scale)
			}
		}()
		_, _ = Binary(b, precision, scale)
	})
}

func Fuzz_Float64(f *testing.F) {
	tests := []struct {
		f         float64
		precision int
		scale     int
	}{
		{123.456, 10, 2},
		{-987.654, 10, 3},
		{0.0001, 10, 5},
		{math.NaN(), 10, 2},   // NaN case
		{math.Inf(1), 10, 2},  // +Inf case
		{math.Inf(-1), 10, 2}, // -Inf case
		{1.23456789, 5, 2},    // Out-of-range case
	}
	for _, tc := range tests {
		f.Add(tc.f, tc.precision, tc.scale)
	}
	f.Fuzz(func(t *testing.T, f float64, precision, scale int) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Float64 caused a panic with input: f=%f, precision=%d, scale=%d", f, precision, scale)
			}
		}()
		_, _ = Float64(f, precision, scale)
	})
}

func Fuzz_Int(f *testing.F) {
	tests := []struct {
		i         int
		precision int
		scale     int
	}{
		{123, 10, 2},
		{-987, 10, 3},
		{0, 10, 5},
		{math.MaxInt64, 10, 2}, // Max int case
		{math.MinInt64, 10, 2}, // Min int case
		{12345, 4, 2},          // Out-of-range case
	}
	for _, tc := range tests {
		f.Add(tc.i, tc.precision, tc.scale)
	}
	f.Fuzz(func(t *testing.T, i int, precision, scale int) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Int caused a panic with input: i=%d, precision=%d, scale=%d", i, precision, scale)
			}
		}()
		_, _ = Int(i, precision, scale)
	})
}

func Fuzz_Parse(f *testing.F) {
	tests := []struct {
		n         string
		precision int
		scale     int
	}{
		{"123", 10, 2},
		{"123.45", 10, 2},
		{".5", 10, 2},
		{"-123.456", 10, 3},
		{"+123e+4", 10, 2},
		{"abc", 10, 2},    // Non-numeric case
		{"123.456", 2, 1}, // Out-of-range case
		{"", 10, 2},       // Empty string
		{"1e1000", 10, 2}, // Exponent out of scale
	}
	for _, tc := range tests {
		f.Add(tc.n, tc.precision, tc.scale)
	}
	f.Fuzz(func(t *testing.T, n string, precision, scale int) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Parse caused a panic with input: n=%q, precision=%d, scale=%d", n, precision, scale)
			}
		}()
		_, _ = Parse(n, precision, scale)
	})
}
