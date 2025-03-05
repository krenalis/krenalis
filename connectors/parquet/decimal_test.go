//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package parquet

import (
	"fmt"
	"testing"

	"github.com/meergo/meergo/decimal"
)

func Test_decimalToInt64(t *testing.T) {
	tests := []struct {
		d          decimal.Decimal
		scale      int
		expected   int64
		expectedOk bool
	}{
		{d: decimal.New(1, 0), scale: 0, expected: 1, expectedOk: true},
		{d: decimal.New(10, 0), scale: 0, expected: 10, expectedOk: true},
		{d: decimal.New(1, 0), scale: 3, expected: 1_000, expectedOk: true},
		{d: decimal.New(31415, 4), scale: 1, expected: 0, expectedOk: false},
		{d: decimal.New(31415, 4), scale: 3, expected: 0, expectedOk: false},
		{d: decimal.New(31415, 4), scale: 4, expected: 31415, expectedOk: true},
		{d: decimal.New(31415, 4), scale: 5, expected: 314150, expectedOk: true},
		{d: decimal.New(100, 1), scale: 1, expected: 100, expectedOk: true},
		{d: decimal.New(100, 1), scale: 2, expected: 1_000, expectedOk: true},

		// Negative numbers.
		{d: decimal.New(-1, 0), scale: 0, expected: -1, expectedOk: true},
		{d: decimal.New(-10, 0), scale: 0, expected: -10, expectedOk: true},
		{d: decimal.New(-1, 0), scale: 3, expected: -1_000, expectedOk: true},
		{d: decimal.New(-31415, 4), scale: 4, expected: -31415, expectedOk: true},
		{d: decimal.New(-31415, 4), scale: 5, expected: -314150, expectedOk: true},
		{d: decimal.New(-100, 1), scale: 1, expected: -100, expectedOk: true},
		{d: decimal.New(-100, 1), scale: 2, expected: -1_000, expectedOk: true},
	}
	for _, test := range tests {
		t.Run(fmt.Sprint(test.d, test.scale), func(t *testing.T) {
			got, gotOk := decimalToInt64(test.d, test.scale)
			if gotOk != test.expectedOk {
				t.Fatalf("expected ok = %t, got %t", test.expectedOk, gotOk)
			}
			if got != test.expected {
				t.Fatalf("expected %v, got %v", test.expected, got)
			}
		})
	}
}
