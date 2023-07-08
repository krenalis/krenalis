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
		from, to types.Type
		expected bool
	}
	cases := []testCase{
		// Boolean.
		{types.Boolean(), types.Boolean(), true},
		{types.Boolean(), types.Int(), false},
		{types.Boolean(), types.DateTime(), false},
		{types.Boolean(), types.JSON(), true},
		{types.Boolean(), types.Text(), true},
		{types.Boolean(), types.Array(types.Text()), false},
		{types.Boolean(), types.Object([]types.Property{{Name: "s", Type: types.Text()}}), false},
		{types.Boolean(), types.Map(types.Text()), false},
		// Int.
		{types.Int(), types.Boolean(), false},
		{types.Int(), types.Int(), true},
		{types.Int(), types.Int8(), true},
		{types.Int(), types.UInt64(), true},
		{types.Int(), types.Float(), true},
		{types.Int(), types.JSON(), true},
		{types.Int(), types.Year(), true},
		{types.Int(), types.Text(), true},
		{types.Int(), types.Array(types.Int8()), false},
		// Int8.
		{types.Int8(), types.Int8(), true},
		// Int16.
		{types.Int16(), types.Int16(), true},
		// Int24.
		{types.Int24(), types.Int24(), true},
		// Int64.
		{types.Int64(), types.Int64(), true},
		// UInt.
		{types.UInt(), types.UInt(), true},
		// UInt16.
		{types.UInt16(), types.UInt16(), true},
		// UInt24.
		{types.UInt24(), types.UInt24(), true},
		// UInt64.
		{types.UInt64(), types.UInt64(), true},
		{types.UInt64(), types.Year(), true},
		// Float.
		{types.Float(), types.Float(), true},
		// Float32.
		{types.Float32(), types.Float32(), true},
		// DateTime.
		{types.DateTime(), types.DateTime(), true},
		// Date.
		{types.Date(), types.Date(), true},
		// Time.
		{types.Time(), types.Time(), true},
		// Year.
		{types.Year(), types.Boolean(), false},
		{types.Year(), types.Int(), true},
		{types.Year(), types.Year(), true},
		{types.Year(), types.JSON(), true},
		{types.Year(), types.Array(types.Int()), false},
		// UUID.
		{types.UUID(), types.UUID(), true},
		// JSON.
		{types.JSON(), types.Boolean(), true},
		{types.JSON(), types.Int64(), true},
		{types.JSON(), types.UInt(), true},
		{types.JSON(), types.Float32(), true},
		{types.JSON(), types.Decimal(10, 3), true},
		{types.JSON(), types.DateTime(), true},
		{types.JSON(), types.JSON(), true},
		{types.JSON(), types.Text(), true},
		{types.JSON(), types.UUID(), true},
		{types.JSON(), types.Array(types.Int()), true},
		{types.JSON(), types.Object([]types.Property{{Name: "s", Type: types.Text()}}), true},
		{types.JSON(), types.Map(types.Text()), true},
		// Inet.
		{types.Inet(), types.Inet(), true},
		// Inet.
		{types.Inet(), types.Inet(), true},
		// Array.
		{types.Array(types.Text()), types.Int(), false},
		{types.Array(types.Text()), types.UInt(), false},
		{types.Array(types.Int()), types.JSON(), true},
		{types.Array(types.Float()), types.Array(types.Float()), true},
		{types.Array(types.Int()), types.Array(types.Float()), true},
		{types.Array(types.DateTime()), types.Array(types.Int()), false},
		{types.Array(types.Text()), types.Object([]types.Property{{Name: "s", Type: types.Text()}}), false},
		{types.Array(types.Text()), types.Map(types.Text()), false},
		// Object.
		{types.Object([]types.Property{{Name: "x", Type: types.Text()}}), types.JSON(), true},
		{types.Object([]types.Property{{Name: "x", Type: types.Text()}}),
			types.Object([]types.Property{{Name: "x", Type: types.Text()}}), true},
		{types.Object([]types.Property{{Name: "x", Type: types.Boolean()}}),
			types.Object([]types.Property{{Name: "y", Type: types.Int()}}), false},
		{types.Object([]types.Property{{Name: "x", Type: types.Text()}, {Name: "y", Type: types.Text()}}),
			types.Object([]types.Property{{Name: "x", Type: types.Text()}}), true},
		{types.Object([]types.Property{{Name: "x", Type: types.Int()}}),
			types.Object([]types.Property{{Name: "x", Type: types.Text()}}), true},
		{types.Object([]types.Property{{Name: "x", Type: types.Year()}}),
			types.Object([]types.Property{{Name: "x", Type: types.Boolean()}}), false},
		// Map.
		{types.Map(types.Text()), types.JSON(), true},
		{types.Map(types.Text()), types.Int(), false},
		{types.Map(types.Text()), types.UInt(), false},
		{types.Map(types.Text()), types.JSON(), true},
		{types.Map(types.Text()), types.Array(types.Text()), false},
		{types.Map(types.Text()), types.Object([]types.Property{{Name: "s", Type: types.Text()}}), false},
		{types.Map(types.Text()), types.Map(types.Text()), true},
		{types.Map(types.Year()), types.Map(types.Boolean()), false},
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
