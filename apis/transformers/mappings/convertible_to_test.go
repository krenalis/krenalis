//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"fmt"
	"testing"

	"github.com/meergo/meergo/types"
)

func TestConvertibleTo(t *testing.T) {
	// This test is not indented for testing the logic behind the conversion
	// matrix; instead, it tests the decoding of such matrix and the correct
	// alignment of properties inserted in it.
	type testCase struct {
		from, to types.Type
		expected bool
	}
	cases := []testCase{
		// Boolean.
		{types.Boolean(), types.Boolean(), true},
		{types.Boolean(), types.Int(32), false},
		{types.Boolean(), types.DateTime(), false},
		{types.Boolean(), types.JSON(), true},
		{types.Boolean(), types.Text(), true},
		{types.Boolean(), types.Array(types.Text()), true},
		{types.Boolean(), types.Array(types.Float(32)), false},
		{types.Boolean(), types.Object([]types.Property{{Name: "s", Type: types.Text()}}), false},
		{types.Boolean(), types.Map(types.Text()), false},
		// Int(8).
		{types.Int(8), types.Int(8), true},
		// Int(16).
		{types.Int(16), types.Int(16), true},
		// Int(24).
		{types.Int(24), types.Int(24), true},
		// Int(32).
		{types.Int(32), types.Boolean(), false},
		{types.Int(32), types.Int(32), true},
		{types.Int(32), types.Int(8), true},
		{types.Int(32), types.Uint(64), true},
		{types.Int(32), types.Float(64), true},
		{types.Int(32), types.JSON(), true},
		{types.Int(32), types.Year(), true},
		{types.Int(32), types.Text(), true},
		{types.Int(32), types.Array(types.Int(8)), true},
		{types.Int(32), types.Map(types.Int(8)), false},
		// Int(64).
		{types.Int(64), types.Int(64), true},
		// Uint(16).
		{types.Uint(16), types.Uint(16), true},
		// Uint(24).
		{types.Uint(24), types.Uint(24), true},
		// Uint(32).
		{types.Uint(32), types.Uint(32), true},
		// Uint(64).
		{types.Uint(64), types.Uint(64), true},
		{types.Uint(64), types.Year(), true},
		// Float(32).
		{types.Float(32), types.Float(32), true},
		// Float(64).
		{types.Float(64), types.Float(64), true},
		// DateTime.
		{types.DateTime(), types.DateTime(), true},
		// Date.
		{types.Date(), types.Date(), true},
		{types.Date(), types.Array(types.Date()), true},
		// Time.
		{types.Time(), types.Time(), true},
		// Year.
		{types.Year(), types.Boolean(), false},
		{types.Year(), types.Int(32), true},
		{types.Year(), types.Year(), true},
		{types.Year(), types.JSON(), true},
		{types.Year(), types.Array(types.Int(32)), true},
		// UUID.
		{types.UUID(), types.UUID(), true},
		// JSON.
		{types.JSON(), types.Boolean(), true},
		{types.JSON(), types.Int(64), true},
		{types.JSON(), types.Uint(32), true},
		{types.JSON(), types.Float(32), true},
		{types.JSON(), types.Decimal(10, 3), true},
		{types.JSON(), types.DateTime(), true},
		{types.JSON(), types.JSON(), true},
		{types.JSON(), types.Text(), true},
		{types.JSON(), types.UUID(), true},
		{types.JSON(), types.Array(types.Int(32)), true},
		{types.JSON(), types.Object([]types.Property{{Name: "s", Type: types.Text()}}), true},
		{types.JSON(), types.Map(types.Text()), true},
		// Inet.
		{types.Inet(), types.Inet(), true},
		// Inet.
		{types.Inet(), types.Inet(), true},
		// Array.
		{types.Array(types.Text()), types.Int(32), false},
		{types.Array(types.Text()), types.Uint(32), false},
		{types.Array(types.Int(32)), types.JSON(), true},
		{types.Array(types.Float(64)), types.Array(types.Float(64)), true},
		{types.Array(types.Int(32)), types.Array(types.Float(64)), true},
		{types.Array(types.DateTime()), types.Array(types.Int(32)), false},
		{types.Array(types.Text()), types.Object([]types.Property{{Name: "s", Type: types.Text()}}), false},
		{types.Array(types.Text()), types.Map(types.Text()), false},
		// Object.
		{types.Object([]types.Property{{Name: "x", Type: types.Text()}}), types.JSON(), true},
		{types.Object([]types.Property{{Name: "x", Type: types.Text()}}),
			types.Object([]types.Property{{Name: "x", Type: types.Text()}}), true},
		{types.Object([]types.Property{{Name: "x", Type: types.Boolean()}}),
			types.Object([]types.Property{{Name: "y", Type: types.Int(32)}}), false},
		{types.Object([]types.Property{{Name: "x", Type: types.Text()}, {Name: "y", Type: types.Text()}}),
			types.Object([]types.Property{{Name: "x", Type: types.Text()}}), true},
		{types.Object([]types.Property{{Name: "x", Type: types.Int(32)}}),
			types.Object([]types.Property{{Name: "x", Type: types.Text()}}), true},
		{types.Object([]types.Property{{Name: "x", Type: types.Year()}}),
			types.Object([]types.Property{{Name: "x", Type: types.Boolean()}}), false},
		// Map.
		{types.Map(types.Text()), types.JSON(), true},
		{types.Map(types.Text()), types.Int(32), false},
		{types.Map(types.Text()), types.Uint(32), false},
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
				t.Fatalf("expected ConvertibleTo(%s, %s) = %t, got %t", cas.from, cas.to, cas.expected, got)
			}
		})
	}
}
