//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mapexp

import (
	"fmt"
	"testing"

	"chichi/connector/types"
)

func TestConvertibleTo(t *testing.T) {
	// This test is not indented for testing the logic behind the conversion
	// matrix; instead, it tests the decoding of such matrix and the correct
	// alignment of values inserted in it.
	type testCase struct {
		from, to types.PhysicalType
		expected bool
	}
	cases := []testCase{
		// Boolean.
		{types.PtBoolean, types.PtBoolean, true},
		{types.PtBoolean, types.PtInt, false},
		{types.PtBoolean, types.PtDateTime, false},
		{types.PtBoolean, types.PtText, true},
		{types.PtBoolean, types.PtArray, false},
		{types.PtBoolean, types.PtObject, false},
		{types.PtBoolean, types.PtMap, false},
		// Array.
		{types.PtArray, types.PtInt, false},
		{types.PtArray, types.PtUInt, false},
		{types.PtArray, types.PtArray, true},
		{types.PtArray, types.PtObject, false},
		{types.PtArray, types.PtMap, false},
		// Map.
		{types.PtMap, types.PtInt, false},
		{types.PtMap, types.PtUInt, false},
		{types.PtMap, types.PtArray, false},
		{types.PtMap, types.PtObject, false},
		{types.PtMap, types.PtMap, true},
	}
	// JSON can be converted to every type.
	for pt := types.PtBoolean; pt <= types.PtMap; pt++ {
		cases = append(cases, testCase{from: types.PtJSON, to: pt, expected: true})
	}
	// Every type can be converted to JSON.
	for pt := types.PtBoolean; pt <= types.PtMap; pt++ {
		cases = append(cases, testCase{from: pt, to: types.PtJSON, expected: true})
	}
	// Every int/uint type (except for Uint64) can be converted to Year.
	for pt := types.PtInt; pt <= types.PtUInt24; pt++ {
		cases = append(cases, testCase{from: pt, to: types.PtYear, expected: true})
	}
	// Every type can be converted to itself (this tests the matrix's main
	// diagonal).
	for pt := types.PtBoolean; pt <= types.PtMap; pt++ {
		cases = append(cases, testCase{from: pt, to: pt, expected: true})
	}
	for _, cas := range cases {
		t.Run(fmt.Sprintf("%s to %s", cas.from, cas.to), func(t *testing.T) {
			got := convertibleTo(cas.from, cas.to)
			if cas.expected != got {
				t.Fatalf("expecting ConvertibleTo(%s, %s) = %t, got %t", cas.from, cas.to, cas.expected, got)
			}
		})
	}
}
