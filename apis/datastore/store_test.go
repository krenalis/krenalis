//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package datastore_test

import (
	"testing"

	"chichi/apis/datastore"
	"chichi/types"
)

func Test_CanBeIdentifier(t *testing.T) {
	tests := []struct {
		t        types.Type
		expected bool
	}{
		{types.Boolean(), false},
		{types.Int(16), true},
		{types.Int(32), true},
		{types.Int(64), true},
		{types.Uint(8), true},
		{types.Uint(32), true},
		{types.Float(32), false},
		{types.Float(64), false},
		{types.Decimal(10, 0), true},
		{types.Decimal(10, 3), false},
		{types.Decimal(3, 3), false},
		{types.DateTime(), false},
		{types.Date(), false},
		{types.Time(), false},
		{types.Year(), false},
		{types.UUID(), true},
		{types.Inet(), true},
		{types.Text(), true},
		{types.Array(types.Text()), false},
		{types.Array(types.Float(32)), false},
		{types.Array(types.Decimal(10, 0)), false},
		{types.Array(types.Array(types.Text())), false},
		{types.Object([]types.Property{{Name: "a", Type: types.Text()}}), false},
		{types.Map(types.Text()), false},
	}
	for _, test := range tests {
		got := datastore.CanBeIdentifier(test.t)
		if got != test.expected {
			t.Errorf("type %v: expected %t, got %t", test.t, test.expected, got)
		}
	}
}
