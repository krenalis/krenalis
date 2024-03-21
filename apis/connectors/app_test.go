//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package connectors

import (
	"regexp"
	"testing"

	"chichi/types"

	"github.com/shopspring/decimal"
)

// object returns an Object type with a property named "p" with type t.
func object(t types.Type) types.Type {
	return types.Object([]types.Property{{Name: "p", Type: t}})
}

func Test_verifySchemaCompatibilityForSendEvents(t *testing.T) {

	tests := []struct {
		name string
		t1   types.Type
		t2   types.Type
		err  string
	}{

		{"", types.Type{}, types.Type{}, ""},
		{"", types.Type{}, object(types.Text()), ""},
		{"", types.Type{}, types.Object([]types.Property{
			{Name: "p", Type: types.Text(), Required: true},
		}), `there is a new required property "p"`},
		{"", object(types.Text()), types.Type{}, `property "p" is no longer present`},

		{"", object(types.Boolean()), object(types.Boolean()), ""},
		{"", object(types.Int(32)), object(types.Int(32)), ""},
		{"", object(types.Int(16)), object(types.Int(32)), ""},
		{"", object(types.Int(16).WithIntRange(0, 10)), object(types.Int(16).WithIntRange(0, 20)), ""},
		{"", object(types.Uint(32)), object(types.Uint(32)), ""},
		{"", object(types.Uint(32)), object(types.Uint(32)), ""},
		{"", object(types.Uint(16)), object(types.Uint(64)), ""},
		{"", object(types.Uint(16).WithUintRange(100, 200)), object(types.Uint(64).WithUintRange(50, 250)), ""},
		{"", object(types.Float(32)), object(types.Float(64)), ""},
		{"", object(types.Float(32).WithFloatRange(-5, 5)), object(types.Float(32).WithFloatRange(-10, 10)), ""},
		{"", object(types.Decimal(10, 3)), object(types.Decimal(11, 4)), ""},
		{"", object(types.Decimal(10, 3)), object(types.Decimal(12, 3)), ""},
		{"", object(types.DateTime()), object(types.DateTime()), ""},
		{"", object(types.Date()), object(types.Date()), ""},
		{"", object(types.Time()), object(types.Time()), ""},
		{"", object(types.Year()), object(types.Year()), ""},
		{"", object(types.UUID()), object(types.UUID()), ""},
		{"", object(types.JSON()), object(types.JSON()), ""},
		{"", object(types.Inet()), object(types.Inet()), ""},
		{"", object(types.Text()), object(types.Text()), ""},
		{"", object(types.Text().WithValues("foo")), object(types.Text().WithValues("boo", "foo")), ""},
		{"", object(types.Text().WithRegexp(regexp.MustCompile(`^\d+`))), object(types.Text().WithRegexp(regexp.MustCompile(`^\d+`))), ""},
		{"", object(types.Text().WithByteLen(10)), object(types.Text()), ""},
		{"", object(types.Text().WithByteLen(10)), object(types.Text().WithByteLen(10)), ""},
		{"", object(types.Text().WithByteLen(10)), object(types.Text().WithByteLen(20)), ""},
		{"", object(types.Text().WithCharLen(10)), object(types.Text().WithCharLen(20)), ""},
		{"", object(types.Array(object(types.Text()))), object(types.Array(object(types.Text()))), ""},
		{"", object(types.Array(types.Text().WithByteLen(10))), object(types.Array(types.Text())), ""},
		{"", types.Object([]types.Property{
			{Name: "a", Type: types.Int(32)},
			{Name: "b", Type: types.Text().WithCharLen(10)},
			{Name: "c", Type: types.Object([]types.Property{
				{Name: "x", Type: types.Int(64).WithIntRange(0, 100), Required: true},
				{Name: "y", Type: types.Array(types.Year()), Required: true}}),
			},
		}), types.Object([]types.Property{
			{Name: "a", Type: types.Int(64)},
			{Name: "b", Type: types.Text().WithCharLen(15)},
			{Name: "c", Type: types.Object([]types.Property{
				{Name: "x", Type: types.Int(64).WithIntRange(0, 100), Required: true},
				{Name: "y", Type: types.Array(types.Year())},
				{Name: "z", Type: types.Decimal(12, 2)}}),
			},
		}), ""},
		{"", object(types.Map(types.Text().WithByteLen(10))), object(types.Map(types.Text().WithByteLen(20))), ""},

		{"", object(types.Int(64)), object(types.Int(32)), `type of the "p" property is changed; minimum value is changed from -9223372036854775808 to -2147483648`},
		{"", object(types.Int(16).WithIntRange(0, 10)), object(types.Int(16).WithIntRange(0, 5)), `type of the "p" property is changed; maximum value is changed from 10 to 5`},
		{"", object(types.Uint(16).WithUintRange(0, 200)), object(types.Uint(64).WithUintRange(50, 200)), `type of the "p" property is changed; minimum value is changed from 0 to 50`},
		{"", object(types.Float(64)), object(types.Float(32)), `type of the "p" property is changed; bit size is changed from 64 to 32`},
		{"", object(types.Float(32).WithFloatRange(-10, 10)), object(types.Float(32).WithFloatRange(-5, 5)), `type of the "p" property is changed; minimum value is changed from -10 to -5`},
		{"", object(types.Uint(16)), object(types.Uint(8)), `type of the "p" property is changed; maximum value is changed from 65535 to 255`},
		{"", object(types.Decimal(10, 3)), object(types.Decimal(10, 2)), `type of property "p" has changed from Decimal(10,3) to Decimal(10,2)`},
		{"", object(types.Decimal(5, 2)), object(types.Decimal(4, 2)), `type of property "p" has changed from Decimal(5,2) to Decimal(4,2)`},
		{"", object(types.Decimal(5, 2).WithDecimalRange(decimal.NewFromFloat(1), decimal.NewFromFloat(3.5))), object(types.Decimal(5, 2).WithDecimalRange(decimal.NewFromFloat(1), decimal.NewFromFloat(3.4))), `type of property "p" has changed; maximum value is changed from 3.5 to 3.4`},
		{"", object(types.Date()), object(types.DateTime()), `type of the "p" property has changed from Date to DateTime`},
		{"", object(types.Text()), object(types.Text().WithByteLen(10)), `type of property "p" has changed; it is now restricted in byte length`},
		{"", object(types.Text().WithByteLen(100)), object(types.Text().WithByteLen(10)), `type of property "p" has changed; maximum length in bytes has changed from 100 to 10`},
		{"", object(types.Text()), object(types.Text().WithCharLen(50)), `type of property "p" has changed; it is now restricted in character length`},
		{"", object(types.Text().WithCharLen(15)), object(types.Text().WithCharLen(5)), `type of property "p" has changed; maximum length in characters has changed from 15 to 5`},
		{"", object(types.Text().WithValues("boo", "foo")), object(types.Text().WithValues("foo")), `type of property "p" has changed; value "boo" is no longer allowed`},
		{"", object(types.Text().WithRegexp(regexp.MustCompile(`^\d+`))), object(types.Text().WithRegexp(regexp.MustCompile(`^\s+`))), `type of property "p" has changed; it validates against a different regular expression`},
		{"", object(types.Array(types.Text())), object(types.Array(types.JSON())), `type of the "p[]" property has changed from Text to JSON`},
		{"", types.Object([]types.Property{
			{Name: "a", Type: types.Int(32)},
			{Name: "b", Type: types.Text().WithCharLen(10)},
		}), types.Object([]types.Property{
			{Name: "a", Type: types.Int(64)},
		}), `property "b" is no longer present`},
		{"", types.Object([]types.Property{
			{Name: "c", Type: types.Object([]types.Property{
				{Name: "x", Type: types.Int(64).WithIntRange(0, 100)},
				{Name: "y", Type: types.Array(types.Year())}}),
			},
		}), types.Object([]types.Property{
			{Name: "c", Type: types.Object([]types.Property{
				{Name: "x", Type: types.Int(64).WithIntRange(0, 100)}}),
			},
		}), `property "c.y" is no longer present`},
		{"", types.Object([]types.Property{
			{Name: "c", Type: types.Object([]types.Property{
				{Name: "x", Type: types.Int(64).WithIntRange(0, 100)},
				{Name: "y", Type: types.Array(types.Year())},
				{Name: "z", Type: types.Decimal(12, 2)}}),
			},
		}), types.Object([]types.Property{
			{Name: "c", Type: types.Object([]types.Property{
				{Name: "x", Type: types.Int(64).WithIntRange(0, 100)},
				{Name: "y", Type: types.Array(types.Year()), Required: true},
				{Name: "z", Type: types.Decimal(12, 2)}}),
			},
		}), `property "c.y" has become required`},
		{"", types.Object([]types.Property{
			{Name: "c", Type: types.Object([]types.Property{
				{Name: "x", Type: types.Int(64).WithIntRange(0, 100), Required: true},
				{Name: "y", Type: types.Array(types.Year())}}),
			},
		}), types.Object([]types.Property{
			{Name: "c", Type: types.Object([]types.Property{
				{Name: "x", Type: types.Int(64).WithIntRange(0, 100), Required: true},
				{Name: "y", Type: types.Array(types.Year())},
				{Name: "z", Type: types.Decimal(12, 2), Required: true}}),
			},
		}), `there is a new required property "c.z"`},
	}

	for _, test := range tests {
		err := verifySchemaCompatibilityForSendEvents(test.t1, test.t2)
		if err != nil {
			if test.err == "" {
				t.Fatalf("expected no error, got error %q", err)
			}
			if test.err != err.Error() {
				t.Fatalf("expected error %q, got error %q", test.err, err)
			}
			continue
		}
		if test.err != "" {
			t.Fatalf("expected error %q, got no error", test.err)
		}
	}

}
