//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"encoding/json"
	"math"
	"regexp"
	"testing"
	"time"

	"chichi/connector/types"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"
)

func TestConvert(t *testing.T) {

	tests := []struct {
		t1, t2 types.Type
		v      any
		e      any
	}{
		// nil.
		{types.JSON(), types.Text(), json.RawMessage("null"), nil},
		{types.Text(), types.Int(), "", nil},
		{types.Text(), types.Text().WithEnum([]string{"foo", "boo"}), "", nil},
		{types.Text(), types.Text().WithRegexp(regexp.MustCompile(`^bo+$`)), "", nil},

		// Boolean.
		{types.Boolean(), types.Boolean(), true, true},
		{types.Boolean(), types.Boolean(), false, false},
		{types.Text(), types.Boolean(), "false", false},
		{types.Text(), types.Boolean(), "False", false},
		{types.Text(), types.Boolean(), "FALSE", false},
		{types.Text(), types.Boolean(), "no", false},
		{types.Text(), types.Boolean(), "No", false},
		{types.Text(), types.Boolean(), "NO", false},
		{types.Text(), types.Boolean(), "true", true},
		{types.Text(), types.Boolean(), "True", true},
		{types.Text(), types.Boolean(), "TRUE", true},
		{types.Text(), types.Boolean(), "yes", true},
		{types.Text(), types.Boolean(), "Yes", true},
		{types.Text(), types.Boolean(), "YES", true},
		{types.JSON(), types.Boolean(), false, false},
		{types.JSON(), types.Boolean(), true, true},
		{types.JSON(), types.Boolean(), json.RawMessage("false"), false},
		{types.JSON(), types.Boolean(), json.RawMessage("true"), true},

		// Int, Int8, Int16, Int24, and Int64.
		{types.Int(), types.Int(), 831, 831},
		{types.Int(), types.Int8(), -123, -123},
		{types.Int(), types.Int16(), 2571, 2571},
		{types.Int(), types.Int24(), 670329, 670329},
		{types.Int(), types.Int64(), math.MaxInt64, math.MaxInt64},
		{types.UInt8(), types.Int(), uint(7), 7},
		{types.Int16(), types.Int(), -29, -29},
		{types.UInt24(), types.Int(), uint(89302), 89302},
		{types.Int64(), types.Int(), math.MaxInt32, math.MaxInt32},
		{types.Float(), types.Int24(), 10.0, 10},
		{types.Float(), types.Int8(), 34.4, 34},
		{types.Float(), types.Int8(), 34.5, 35},
		{types.Float(), types.Int8(), -0.49, 0},
		{types.Float32(), types.Int8(), -0.5, -1},
		{types.Float(), types.Int64(), minFloatConvertibleToInt64, -9223372036854775808},
		{types.Float(), types.Int64(), maxFloatConvertibleToInt64, 9223372036854774784},
		{types.Decimal(5, 3), types.Int(), decimal.RequireFromString("5"), 5},
		{types.Decimal(5, 3), types.Int8(), decimal.RequireFromString("-12.0"), -12},
		{types.Decimal(60, 0), types.Int64(), minIntDecimal, math.MinInt64},
		{types.Decimal(60, 0), types.Int64(), maxIntDecimal, math.MaxInt64},
		{types.Year(), types.Int16(), 2020, 2020},
		{types.Text(), types.Int(), "502842", 502842},
		{types.JSON(), types.Int(), 12.627, 13},
		{types.JSON(), types.Int(), json.Number("-15"), -15},
		{types.JSON(), types.Int(), json.RawMessage("-2"), -2},

		// UInt, UInt8, UInt16, UInt24 and UInt64.
		{types.Int(), types.UInt(), 831, uint(831)},
		{types.Int(), types.UInt8(), 218, uint(218)},
		{types.Int(), types.UInt16(), 2571, uint(2571)},
		{types.Int(), types.UInt24(), 670329, uint(670329)},
		{types.UInt(), types.UInt64(), uint(math.MaxUint32), uint(math.MaxUint32)},
		{types.UInt8(), types.UInt(), uint(7), uint(7)},
		{types.UInt16(), types.UInt(), uint(29), uint(29)},
		{types.UInt24(), types.UInt(), uint(89302), uint(89302)},
		{types.Float(), types.UInt64(), maxFloatConvertibleToUInt64, uint(18446744073709549568)},
		{types.Decimal(60, 0), types.UInt64(), maxUIntDecimal, uint(math.MaxUint64)},
		{types.Year(), types.UInt16(), 2020, uint(2020)},
		{types.Text(), types.UInt(), "502842", uint(502842)},
		{types.JSON(), types.UInt64(), maxFloatConvertibleToUInt64, uint(18446744073709549568)},
		{types.JSON(), types.UInt(), json.Number("15"), uint(15)},
		{types.JSON(), types.UInt(), json.RawMessage("2"), uint(2)},

		// Float and Float32.
		{types.Float(), types.Float(), 701.502830285, 701.502830285},
		{types.Float(), types.Float32(), 3.918347105316932e+10, float64(float32(3.918347e+10))},
		{types.Float32(), types.Float32(), float64(float32(6316.0513)), float64(float32(6316.0513))},
		{types.Float32(), types.Float(), float64(float32(-32.04262)), -32.04262161254883},
		{types.Int(), types.Float(), 5617072831, 5.617072831e+09},
		{types.UInt8(), types.Float32(), uint(256), float64(float32(256))},
		{types.Decimal(20, 10), types.Float(), decimal.RequireFromString("767.5018382257"), 767.5018382257},
		{types.Text(), types.Float(), "767.5018382257", 767.5018382257},
		{types.JSON(), types.Float(), 767.5018382257, 767.5018382257},
		{types.JSON(), types.Float(), json.Number("767.5018382257"), 767.5018382257},
		{types.JSON(), types.Float(), json.RawMessage("767.5018382257"), 767.5018382257},

		// Decimal.
		{types.Int(), types.Decimal(10, 3), math.MaxInt32, decimal.NewFromInt(math.MaxInt32)},
		{types.Int(), types.Decimal(10, 0), math.MinInt32, decimal.NewFromInt(math.MinInt32)},
		{types.UInt8(), types.Decimal(3, 0), uint(math.MaxUint8), decimal.NewFromInt(math.MaxUint8)},
		{types.Float(), types.Decimal(3, 0), 3.918347105316932e+10, decimal.RequireFromString("39183471053.16932")},
		{types.Float32(), types.Decimal(3, 0), float64(float32(6316.0513)), decimal.RequireFromString("6316.05126953125")},
		{types.Text(), types.Decimal(20, 10), "1048294.202936601", decimal.RequireFromString("1048294.202936601")},
		{types.JSON(), types.Decimal(20, 10), 1048294.202936601, decimal.RequireFromString("1048294.202936601")},
		{types.JSON(), types.Decimal(20, 10), json.Number("1048294.202936601"), decimal.RequireFromString("1048294.202936601")},
		{types.JSON(), types.Decimal(20, 10), json.RawMessage("1048294.202936601"), decimal.RequireFromString("1048294.202936601")},

		// DateTime.
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC)},
		{types.Date(), types.DateTime(), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC)},
		{types.Text(), types.DateTime(), "2023-05-24T09:01:57.49361409Z", time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC)},
		{types.Text(), types.DateTime(), "2023-05-24T09:01:57-07:00", time.Date(2023, 5, 24, 16, 1, 57, 0, time.UTC)},
		{types.JSON(), types.DateTime(), "2023-05-24T09:01:57-07:00", time.Date(2023, 5, 24, 16, 1, 57, 0, time.UTC)},
		{types.JSON(), types.DateTime(), json.RawMessage(`"2023-05-24T09:01:57-07:00"`), time.Date(2023, 5, 24, 16, 1, 57, 0, time.UTC)},

		// Date.
		{types.Date(), types.Date(), time.Date(2023, 24, 5, 0, 0, 0, 0, time.UTC), time.Date(2023, 24, 5, 0, 0, 0, 0, time.UTC)},
		{types.DateTime(), types.Date(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC)},
		{types.Text(), types.Date(), "2023-05-24", time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC)},
		{types.JSON(), types.Date(), "2023-05-24", time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC)},
		{types.JSON(), types.Date(), json.RawMessage(`"2023-05-24"`), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC)},

		// Time.
		{types.Time(), types.Time(), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC)},
		{types.DateTime(), types.Time(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC)},
		{types.Text(), types.Time(), "09:01:57.49361409Z", time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC)},
		{types.Text(), types.Time(), "09:01:57", time.Date(1970, 1, 1, 9, 1, 57, 0, time.UTC)},
		{types.JSON(), types.Time(), "09:01:57.49361409Z", time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC)},
		{types.JSON(), types.Time(), "09:01:57", time.Date(1970, 1, 1, 9, 1, 57, 0, time.UTC)},
		{types.JSON(), types.Time(), json.RawMessage(`"09:01:57.49361409Z"`), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC)},

		// Year.
		{types.Year(), types.Year(), 2023, 2023},
		{types.Int16(), types.Year(), 1, 1},
		{types.UInt64(), types.Year(), uint(9999), 9999},
		{types.Text(), types.Year(), "2023", 2023},
		{types.Text(), types.Year(), "1", 1},
		{types.JSON(), types.Year(), 1.0, 1},
		{types.JSON(), types.Year(), 2023.0, 2023},
		{types.JSON(), types.Year(), json.Number("2023"), 2023},
		{types.JSON(), types.Year(), json.RawMessage("2023"), 2023},

		// UUID.
		{types.UUID(), types.UUID(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000"},
		{types.Text(), types.UUID(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000"},
		{types.JSON(), types.UUID(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000"},
		{types.JSON(), types.UUID(), json.RawMessage(`"123e4567-e89b-12d3-a456-426614174000"`), "123e4567-e89b-12d3-a456-426614174000"},

		// JSON.
		{types.JSON(), types.JSON(), json.RawMessage(`{"foo":5}`), json.RawMessage(`{"foo":5}`)},
		{types.JSON(), types.JSON(), json.RawMessage(`null`), json.RawMessage(`null`)},
		{types.JSON(), types.JSON(), true, json.RawMessage(`true`)},
		{types.JSON(), types.JSON(), "foo", json.RawMessage(`"foo"`)},
		{types.JSON(), types.JSON(), 3.14, json.RawMessage(`3.14`)},
		{types.JSON(), types.JSON(), json.Number("7204812694472.9355460893"), json.RawMessage(`7204812694472.9355460893`)},
		{types.JSON(), types.JSON(), map[string]any{"foo": "boo"}, json.RawMessage(`{"foo":"boo"}`)},
		{types.JSON(), types.JSON(), []any{1, 2, 3}, json.RawMessage(`[1,2,3]`)},

		// Inet.
		{types.Inet(), types.Inet(), "2001:db8::ff00:42:8329", "2001:db8::ff00:42:8329"},
		{types.Text(), types.Inet(), "2001:0db8:0000:0000:0000:ff00:0042:8329", "2001:db8::ff00:42:8329"},
		{types.JSON(), types.Inet(), "2001:0db8:0000:0000:0000:ff00:0042:8329", "2001:db8::ff00:42:8329"},
		{types.JSON(), types.Inet(), json.RawMessage(`"2001:0db8:0000:0000:0000:ff00:0042:8329"`), "2001:db8::ff00:42:8329"},

		// Text.
		{types.Text(), types.Text(), "foo", "foo"},
		{types.Text(), types.Text().WithEnum([]string{"foo", "boo"}), "boo", "boo"},
		{types.Text(), types.Text().WithRegexp(regexp.MustCompile(`^bo+$`)), "boo", "boo"},
		{types.Boolean(), types.Text(), true, "true"},
		{types.Int(), types.Text(), -603, "-603"},
		{types.Float(), types.Text(), 7928301735.704827, "7.928301735704827e+09"},
		{types.Float32(), types.Text(), 3.14, "3.14"},
		{types.Decimal(5, 2), types.Text(), decimal.RequireFromString("120.79"), "120.79"},
		{types.DateTime(), types.Text(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), "2023-05-24T09:01:57.49361409Z"},
		{types.DateTime(), types.Text(), time.Date(2023, 5, 24, 9, 1, 57, 0, time.UTC), "2023-05-24T09:01:57Z"},
		{types.Date(), types.Text(), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), "2023-05-24"},
		{types.Time(), types.Text(), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), "09:01:57.49361409"},
		{types.Time(), types.Text(), time.Date(1970, 1, 1, 9, 1, 57, 0, time.UTC), "09:01:57"},
		{types.Year(), types.Text(), 1, "1"},
		{types.Year(), types.Text(), 2023, "2023"},
		{types.UUID(), types.Text(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000"},
		{types.Inet(), types.Text(), "2001:db8::ff00:42:8329", "2001:db8::ff00:42:8329"},
		{types.JSON(), types.Text(), "foo", "foo"},
		{types.JSON(), types.Text(), 23.8013, "23.8013"},
		{types.JSON(), types.Text(), json.Number("812"), "812"},
		{types.JSON(), types.Text(), true, "true"},
		{types.JSON(), types.Text(), json.RawMessage("null"), nil},

		// Array.
		{types.Array(types.Int()), types.Array(types.Int()), []any{1, 2, 3}, []any{1, 2, 3}},
		{types.Array(types.Int()), types.Array(types.Int8()), []any{1, 2, 3}, []any{1, 2, 3}},
		{types.JSON(), types.Array(types.Int()), []any{1.0, 2.0, 3.0}, []any{1, 2, 3}},
		{types.JSON(), types.Array(types.Int()), []any{json.Number("1"), json.Number("2"), json.Number("3")}, []any{1, 2, 3}},
		{types.JSON(), types.Array(types.Int()), json.RawMessage(`[1,2,3]`), []any{1, 2, 3}},

		// Object.
		{
			types.Object([]types.Property{{Name: "foo", Type: types.Int()}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			types.Object([]types.Property{{Name: "foo", Type: types.Int()}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			map[string]any{"foo": 5, "boo": nil},
			map[string]any{"foo": 5, "boo": nil},
		},
		{
			types.JSON(),
			types.Object([]types.Property{{Name: "foo", Type: types.Int()}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			map[string]any{"foo": 5.0, "boo": nil},
			map[string]any{"foo": 5, "boo": nil},
		},
		{
			types.JSON(),
			types.Object([]types.Property{{Name: "foo", Type: types.Int()}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			map[string]any{"foo": json.Number("5"), "boo": nil},
			map[string]any{"foo": 5, "boo": nil},
		},
		{
			types.JSON(),
			types.Object([]types.Property{{Name: "foo", Type: types.Int()}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			json.RawMessage(`{"foo":5,"boo":null}`),
			map[string]any{"foo": 5, "boo": nil},
		},

		// Map.
		{types.Map(types.Boolean()), types.Map(types.Boolean()), map[string]any{"a": true, "b": false}, map[string]any{"a": true, "b": false}},
		{types.Map(types.Int16()), types.Map(types.Float32()), map[string]any{"a": 4032, "b": -721}, map[string]any{"a": float64(float32(4032)), "b": float64(float32(-721))}},
		{types.JSON(), types.Map(types.Float32()), map[string]any{"a": 4032.0, "b": -721.0}, map[string]any{"a": float64(float32(4032)), "b": float64(float32(-721))}},
		{types.JSON(), types.Map(types.Float32()), map[string]any{"a": json.Number("4032"), "b": json.Number("-721")}, map[string]any{"a": float64(float32(4032)), "b": float64(float32(-721))}},
		{types.JSON(), types.Map(types.Float32()), json.RawMessage(`{"a":4032,"b":-721}`), map[string]any{"a": float64(float32(4032)), "b": float64(float32(-721))}},
	}

	for _, test := range tests {
		got, err := convert(test.v, test.t1, test.t2, true)
		if err != nil {
			t.Fatalf("cannot convert %s<%v> to type %s", test.t1, test.v, test.t2)
		}
		expected := test.e
		if !cmp.Equal(got, expected) {
			if f, ok := expected.(float64); ok && math.IsNaN(f) {
				if f, ok := got.(float64); ok && math.IsNaN(f) {
					continue
				}
			}
			t.Fatalf("expected %T(%v), got %T(%v)", expected, expected, got, got)
		}
	}
}
