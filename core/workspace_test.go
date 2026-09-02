// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"testing"

	"github.com/krenalis/krenalis/tools/types"
)

func Test_suitableAsIdentifier(t *testing.T) {
	tests := []struct {
		t        types.Type
		expected bool
	}{
		{types.String(), true},
		{types.Boolean(), false},
		{types.Int(16), true},
		{types.Int(32), true},
		{types.Int(64), true},
		{types.Int(8).Unsigned(), true},
		{types.Int(32).Unsigned(), true},
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
		{types.IP(), true},
		{types.Array(types.String()), false},
		{types.Array(types.Float(32)), false},
		{types.Array(types.Decimal(10, 0)), false},
		{types.Array(types.Array(types.String())), false},
		{types.Object([]types.Property{{Name: "a", Type: types.String()}}), false},
		{types.Map(types.String()), false},
	}
	for _, test := range tests {
		got := suitableAsIdentifier(test.t)
		if got != test.expected {
			t.Errorf("type %v: expected %t, got %t", test.t, test.expected, got)
		}
	}
}
