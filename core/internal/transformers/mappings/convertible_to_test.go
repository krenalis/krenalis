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

	"github.com/meergo/meergo/core/types"
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
		// boolean.
		{types.Boolean(), types.Boolean(), true},
		{types.Boolean(), types.Int(32), false},
		{types.Boolean(), types.DateTime(), false},
		{types.Boolean(), types.JSON(), true},
		{types.Boolean(), types.Text(), true},
		{types.Boolean(), types.Array(types.Boolean()), false},
		{types.Boolean(), types.Object([]types.Property{{Name: "s", Type: types.Text()}}), false},
		{types.Boolean(), types.Map(types.Text()), false},
		// int(8).
		{types.Int(8), types.Int(8), true},
		// int(16).
		{types.Int(16), types.Int(16), true},
		// int(24).
		{types.Int(24), types.Int(24), true},
		// int(32).
		{types.Int(32), types.Boolean(), false},
		{types.Int(32), types.Int(32), true},
		{types.Int(32), types.Int(8), true},
		{types.Int(32), types.Uint(64), true},
		{types.Int(32), types.Float(64), true},
		{types.Int(32), types.JSON(), true},
		{types.Int(32), types.Year(), true},
		{types.Int(32), types.Text(), true},
		{types.Int(32), types.Array(types.Int(32)), false},
		{types.Int(32), types.Map(types.Int(8)), false},
		// int(64).
		{types.Int(64), types.Int(64), true},
		// uint(16).
		{types.Uint(16), types.Uint(16), true},
		// uint(24).
		{types.Uint(24), types.Uint(24), true},
		// uint(32).
		{types.Uint(32), types.Uint(32), true},
		// uint(64).
		{types.Uint(64), types.Uint(64), true},
		{types.Uint(64), types.Year(), true},
		// float(32).
		{types.Float(32), types.Float(32), true},
		// float(64).
		{types.Float(64), types.Float(64), true},
		// datetime.
		{types.DateTime(), types.DateTime(), true},
		// date.
		{types.Date(), types.Date(), true},
		{types.Date(), types.Array(types.Date()), false},
		// time.
		{types.Time(), types.Time(), true},
		// year.
		{types.Year(), types.Boolean(), false},
		{types.Year(), types.Int(32), true},
		{types.Year(), types.Year(), true},
		{types.Year(), types.JSON(), true},
		{types.Year(), types.Array(types.Year()), false},
		// uuid.
		{types.UUID(), types.UUID(), true},
		// json.
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
		// inet.
		{types.Inet(), types.Inet(), true},
		// inet.
		{types.Inet(), types.Inet(), true},
		// array.
		{types.Array(types.Text()), types.Int(32), false},
		{types.Array(types.Text()), types.Uint(32), false},
		{types.Array(types.Int(32)), types.JSON(), true},
		{types.Array(types.Float(64)), types.Array(types.Float(64)), true},
		{types.Array(types.Int(32)), types.Array(types.Float(64)), true},
		{types.Array(types.DateTime()), types.Array(types.Int(32)), false},
		{types.Array(types.Text()), types.Object([]types.Property{{Name: "s", Type: types.Text()}}), false},
		{types.Array(types.Text()), types.Map(types.Text()), false},
		// object.
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
		{types.Object([]types.Property{{Name: "x", Type: types.Year()}}),
			types.Map(types.Year()), true},
		{types.Object([]types.Property{{Name: "x", Type: types.Year()}}),
			types.Map(types.Boolean()), false},
		// map.
		{types.Map(types.Text()), types.JSON(), true},
		{types.Map(types.Text()), types.Int(32), false},
		{types.Map(types.Text()), types.Uint(32), false},
		{types.Map(types.Text()), types.JSON(), true},
		{types.Map(types.Text()), types.Array(types.Text()), false},
		{types.Map(types.Text()), types.Object([]types.Property{{Name: "s", Type: types.Text()}}), true},
		{types.Map(types.Boolean()), types.Object([]types.Property{{Name: "s", Type: types.Int(32)}}), false},
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
