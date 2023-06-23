//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mapexp

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
		t1, t2     types.Type
		v          any
		e          any
		nullable   bool
		formatTime bool
	}{

		// Boolean.
		{types.Boolean(), types.Boolean(), true, true, true, false},
		{types.Boolean(), types.Boolean(), false, false, true, false},
		{types.Int8(), types.Boolean(), 0, false, true, false},
		{types.Int8(), types.Boolean(), 1, true, true, false},
		{types.Int8(), types.Boolean(), -1, true, true, false},
		{types.UInt8(), types.Boolean(), uint(0), false, true, false},
		{types.UInt8(), types.Boolean(), uint(1), true, true, false},
		{types.Text(), types.Boolean(), "false", false, true, false},
		{types.Text(), types.Boolean(), "False", false, true, false},
		{types.Text(), types.Boolean(), "FALSE", false, true, false},
		{types.Text(), types.Boolean(), "no", false, true, false},
		{types.Text(), types.Boolean(), "No", false, true, false},
		{types.Text(), types.Boolean(), "NO", false, true, false},
		{types.Text(), types.Boolean(), "true", true, true, false},
		{types.Text(), types.Boolean(), "True", true, true, false},
		{types.Text(), types.Boolean(), "TRUE", true, true, false},
		{types.Text(), types.Boolean(), "yes", true, true, false},
		{types.Text(), types.Boolean(), "Yes", true, true, false},
		{types.Text(), types.Boolean(), "YES", true, true, false},
		{types.JSON(), types.Boolean(), false, false, true, false},
		{types.JSON(), types.Boolean(), true, true, true, false},
		{types.JSON(), types.Boolean(), json.RawMessage("false"), false, true, false},
		{types.JSON(), types.Boolean(), json.RawMessage("true"), true, true, false},

		// Int, Int8, Int16, Int24, and Int64.
		{types.Int(), types.Int(), 831, 831, true, false},
		{types.Int(), types.Int8(), -123, -123, true, false},
		{types.Int(), types.Int16(), 2571, 2571, true, false},
		{types.Int(), types.Int24(), 670329, 670329, true, false},
		{types.Int(), types.Int64(), math.MaxInt64, math.MaxInt64, true, false},
		{types.UInt8(), types.Int(), uint(7), 7, true, false},
		{types.Int16(), types.Int(), -29, -29, true, false},
		{types.UInt24(), types.Int(), uint(89302), 89302, true, false},
		{types.Int64(), types.Int(), math.MaxInt32, math.MaxInt32, true, false},
		{types.Float(), types.Int24(), 10.0, 10, true, false},
		{types.Float(), types.Int8(), 34.4, 34, true, false},
		{types.Float(), types.Int8(), 34.5, 35, true, false},
		{types.Float(), types.Int8(), -0.49, 0, true, false},
		{types.Float32(), types.Int8(), -0.5, -1, true, false},
		{types.Float(), types.Int64(), minFloatConvertibleToInt64, -9223372036854775808, true, false},
		{types.Float(), types.Int64(), maxFloatConvertibleToInt64, 9223372036854774784, true, false},
		{types.Decimal(5, 3), types.Int(), decimal.RequireFromString("5"), 5, true, false},
		{types.Decimal(5, 3), types.Int8(), decimal.RequireFromString("-12.0"), -12, true, false},
		{types.Decimal(60, 0), types.Int64(), minIntDecimal, math.MinInt64, true, false},
		{types.Decimal(60, 0), types.Int64(), maxIntDecimal, math.MaxInt64, true, false},
		{types.Year(), types.Int16(), 2020, 2020, true, false},
		{types.Text(), types.Int(), "502842", 502842, true, false},
		{types.Text(), types.Int(), "", nil, true, false},
		{types.JSON(), types.Int(), 12.627, 13, true, false},
		{types.JSON(), types.Int(), json.Number("-15"), -15, true, false},
		{types.JSON(), types.Int(), json.RawMessage("-2"), -2, true, false},
		{types.Boolean(), types.Int8(), false, 0, true, false},
		{types.Boolean(), types.Int8(), true, 1, true, false},

		// UInt, UInt8, UInt16, UInt24 and UInt64.
		{types.Int(), types.UInt(), 831, uint(831), true, false},
		{types.Int(), types.UInt8(), 218, uint(218), true, false},
		{types.Int(), types.UInt16(), 2571, uint(2571), true, false},
		{types.Int(), types.UInt24(), 670329, uint(670329), true, false},
		{types.UInt(), types.UInt64(), uint(math.MaxUint32), uint(math.MaxUint32), true, false},
		{types.UInt8(), types.UInt(), uint(7), uint(7), true, false},
		{types.UInt16(), types.UInt(), uint(29), uint(29), true, false},
		{types.UInt24(), types.UInt(), uint(89302), uint(89302), true, false},
		{types.Float(), types.UInt64(), maxFloatConvertibleToUInt64, uint(18446744073709549568), true, false},
		{types.Decimal(60, 0), types.UInt64(), maxUIntDecimal, uint(math.MaxUint64), true, false},
		{types.Year(), types.UInt16(), 2020, uint(2020), true, false},
		{types.Text(), types.UInt(), "502842", uint(502842), true, false},
		{types.JSON(), types.UInt64(), maxFloatConvertibleToUInt64, uint(18446744073709549568), true, false},
		{types.JSON(), types.UInt(), json.Number("15"), uint(15), true, false},
		{types.JSON(), types.UInt(), json.RawMessage("2"), uint(2), true, false},
		{types.Boolean(), types.UInt8(), false, uint(0), true, false},
		{types.Boolean(), types.UInt8(), true, uint(1), true, false},

		// Float and Float32.
		{types.Float(), types.Float(), 701.502830285, 701.502830285, true, false},
		{types.Float(), types.Float32(), 3.918347105316932e+10, float64(float32(3.918347e+10)), true, false},
		{types.Float32(), types.Float32(), float64(float32(6316.0513)), float64(float32(6316.0513)), true, false},
		{types.Float32(), types.Float(), float64(float32(-32.04262)), -32.04262161254883, true, false},
		{types.Int(), types.Float(), 5617072831, 5.617072831e+09, true, false},
		{types.UInt8(), types.Float32(), uint(256), float64(float32(256)), true, false},
		{types.Decimal(20, 10), types.Float(), decimal.RequireFromString("767.5018382257"), 767.5018382257, true, false},
		{types.Text(), types.Float(), "767.5018382257", 767.5018382257, true, false},
		{types.JSON(), types.Float(), 767.5018382257, 767.5018382257, true, false},
		{types.JSON(), types.Float(), json.Number("767.5018382257"), 767.5018382257, true, false},
		{types.JSON(), types.Float(), json.RawMessage("767.5018382257"), 767.5018382257, true, false},

		// Decimal.
		{types.Int(), types.Decimal(10, 3), math.MaxInt32, decimal.NewFromInt(math.MaxInt32), true, false},
		{types.Int(), types.Decimal(10, 0), math.MinInt32, decimal.NewFromInt(math.MinInt32), true, false},
		{types.UInt8(), types.Decimal(3, 0), uint(math.MaxUint8), decimal.NewFromInt(math.MaxUint8), true, false},
		{types.Float(), types.Decimal(3, 0), 3.918347105316932e+10, decimal.RequireFromString("39183471053.16932"), true, false},
		{types.Float32(), types.Decimal(3, 0), float64(float32(6316.0513)), decimal.RequireFromString("6316.05126953125"), true, false},
		{types.Text(), types.Decimal(20, 10), "1048294.202936601", decimal.RequireFromString("1048294.202936601"), true, false},
		{types.JSON(), types.Decimal(20, 10), 1048294.202936601, decimal.RequireFromString("1048294.202936601"), true, false},
		{types.JSON(), types.Decimal(20, 10), json.Number("1048294.202936601"), decimal.RequireFromString("1048294.202936601"), true, false},
		{types.JSON(), types.Decimal(20, 10), json.RawMessage("1048294.202936601"), decimal.RequireFromString("1048294.202936601"), true, false},

		// DateTime.
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), true, false},
		{types.Date(), types.DateTime(), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, false},
		{types.Text(), types.DateTime(), "2023-05-24T09:01:57.49361409Z", time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), true, false},
		{types.Text(), types.DateTime(), "2023-05-24T09:01:57-07:00", time.Date(2023, 5, 24, 16, 1, 57, 0, time.UTC), true, false},
		{types.JSON(), types.DateTime(), "2023-05-24T09:01:57-07:00", time.Date(2023, 5, 24, 16, 1, 57, 0, time.UTC), true, false},
		{types.JSON(), types.DateTime(), json.RawMessage(`"2023-05-24T09:01:57-07:00"`), time.Date(2023, 5, 24, 16, 1, 57, 0, time.UTC), true, false},
		{types.DateTime(), types.DateTime().WithLayout(types.Seconds), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), int64(1684918917), true, true},
		{types.DateTime(), types.DateTime().WithLayout(types.Milliseconds), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), int64(1684918917493), true, true},
		{types.DateTime(), types.DateTime().WithLayout(types.Microseconds), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), int64(1684918917493614), true, true},
		{types.DateTime(), types.DateTime().WithLayout(types.Nanoseconds), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), int64(1684918917493614090), true, true},
		{types.DateTime(), types.DateTime().WithLayout(time.RFC850), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), "Wednesday, 24-May-23 09:01:57 UTC", true, true},
		{types.Text(), types.DateTime().WithLayout(time.RFC1123Z), "2023-05-24T09:01:57.49361409Z", "Wed, 24 May 2023 09:01:57 +0000", true, true},
		{types.DateTime(), types.DateTime().WithLayout(time.RFC850), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), true, false},

		// Date.
		{types.Date(), types.Date(), time.Date(2023, 24, 5, 0, 0, 0, 0, time.UTC), time.Date(2023, 24, 5, 0, 0, 0, 0, time.UTC), true, false},
		{types.DateTime(), types.Date(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, false},
		{types.Text(), types.Date(), "2023-05-24", time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, false},
		{types.JSON(), types.Date(), "2023-05-24", time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, false},
		{types.JSON(), types.Date(), json.RawMessage(`"2023-05-24"`), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, false},
		{types.Date(), types.Date().WithLayout(time.DateOnly), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), "2023-05-24", true, true},
		{types.Text(), types.Date().WithLayout("01/02/2006"), "2023-05-24", "05/24/2023", true, true},
		{types.Date(), types.Date().WithLayout(time.RFC850), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, false},

		// Time.
		{types.Time(), types.Time(), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, false},
		{types.DateTime(), types.Time(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, false},
		{types.Text(), types.Time(), "09:01:57.49361409Z", time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, false},
		{types.Text(), types.Time(), "09:01:57", time.Date(1970, 1, 1, 9, 1, 57, 0, time.UTC), true, false},
		{types.JSON(), types.Time(), "09:01:57.49361409Z", time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, false},
		{types.JSON(), types.Time(), "09:01:57", time.Date(1970, 1, 1, 9, 1, 57, 0, time.UTC), true, false},
		{types.JSON(), types.Time(), json.RawMessage(`"09:01:57.49361409Z"`), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, false},
		{types.Time(), types.Time().WithLayout("15:04:05.999999"), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), "09:01:57.493614", true, true},
		{types.Time(), types.Time().WithLayout("15:04:05.999999"), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, false},

		// Year.
		{types.Year(), types.Year(), 2023, 2023, true, false},
		{types.Int16(), types.Year(), 1, 1, true, false},
		{types.UInt64(), types.Year(), uint(9999), 9999, true, false},
		{types.Text(), types.Year(), "2023", 2023, true, false},
		{types.Text(), types.Year(), "1", 1, true, false},
		{types.JSON(), types.Year(), 1.0, 1, true, false},
		{types.JSON(), types.Year(), 2023.0, 2023, true, false},
		{types.JSON(), types.Year(), json.Number("2023"), 2023, true, false},
		{types.JSON(), types.Year(), json.RawMessage("2023"), 2023, true, false},

		// UUID.
		{types.UUID(), types.UUID(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000", true, false},
		{types.Text(), types.UUID(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000", true, false},
		{types.JSON(), types.UUID(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000", true, false},
		{types.JSON(), types.UUID(), json.RawMessage(`"123e4567-e89b-12d3-a456-426614174000"`), "123e4567-e89b-12d3-a456-426614174000", true, false},

		// JSON.
		{types.Int(), types.JSON(), nil, nil, true, false},
		{types.Int(), types.JSON(), nil, json.RawMessage(`null`), false, false},
		{types.JSON(), types.JSON(), json.RawMessage(`{"foo":5}`), json.RawMessage(`{"foo":5}`), true, false},
		{types.JSON(), types.JSON(), json.RawMessage(`{"foo":5}`), json.RawMessage(`{"foo":5}`), true, false},
		{types.JSON(), types.JSON(), json.RawMessage(`null`), json.RawMessage(`null`), true, false},
		{types.Text(), types.JSON(), "", json.RawMessage("null"), false, false},
		{types.JSON(), types.JSON(), true, json.RawMessage(`true`), true, false},
		{types.JSON(), types.JSON(), "foo", json.RawMessage(`"foo"`), true, false},
		{types.JSON(), types.JSON(), 3.14, json.RawMessage(`3.14`), true, false},
		{types.JSON(), types.JSON(), json.Number("7204812694472.9355460893"), json.RawMessage(`7204812694472.9355460893`), true, false},
		{types.JSON(), types.JSON(), map[string]any{"foo": "boo"}, json.RawMessage(`{"foo":"boo"}`), true, false},
		{types.JSON(), types.JSON(), []any{1, 2, 3}, json.RawMessage(`[1,2,3]`), true, false},

		// Inet.
		{types.Inet(), types.Inet(), "2001:db8::ff00:42:8329", "2001:db8::ff00:42:8329", true, false},
		{types.Text(), types.Inet(), "2001:0db8:0000:0000:0000:ff00:0042:8329", "2001:db8::ff00:42:8329", true, false},
		{types.JSON(), types.Inet(), "2001:0db8:0000:0000:0000:ff00:0042:8329", "2001:db8::ff00:42:8329", true, false},
		{types.JSON(), types.Inet(), json.RawMessage(`"2001:0db8:0000:0000:0000:ff00:0042:8329"`), "2001:db8::ff00:42:8329", true, false},

		// Text.
		{types.Int(), types.Text(), nil, nil, true, false},
		{types.Int(), types.Text(), nil, "", false, false},
		{types.Text(), types.Text(), "foo", "foo", true, false},
		{types.Text(), types.Text().WithEnum([]string{"foo", "boo"}), "", nil, true, false},
		{types.Text(), types.Text().WithEnum([]string{"foo", "boo"}), "boo", "boo", true, false},
		{types.Text(), types.Text().WithRegexp(regexp.MustCompile(`^bo+$`)), "", nil, true, false},
		{types.Text(), types.Text().WithRegexp(regexp.MustCompile(`^bo+$`)), "boo", "boo", true, false},
		{types.Boolean(), types.Text(), true, "true", true, false},
		{types.Int(), types.Text(), -603, "-603", true, false},
		{types.Float(), types.Text(), 7928301735.704827, "7.928301735704827e+09", true, false},
		{types.Float32(), types.Text(), 3.14, "3.14", true, false},
		{types.Decimal(5, 2), types.Text(), decimal.RequireFromString("120.79"), "120.79", true, false},
		{types.DateTime(), types.Text(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), "2023-05-24T09:01:57.49361409Z", true, false},
		{types.DateTime(), types.Text(), time.Date(2023, 5, 24, 9, 1, 57, 0, time.UTC), "2023-05-24T09:01:57Z", true, false},
		{types.Date(), types.Text(), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), "2023-05-24", true, false},
		{types.Time(), types.Text(), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), "09:01:57.49361409", true, false},
		{types.Time(), types.Text(), time.Date(1970, 1, 1, 9, 1, 57, 0, time.UTC), "09:01:57", true, false},
		{types.Year(), types.Text(), 1, "1", true, false},
		{types.Year(), types.Text(), 2023, "2023", true, false},
		{types.UUID(), types.Text(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000", true, false},
		{types.Inet(), types.Text(), "2001:db8::ff00:42:8329", "2001:db8::ff00:42:8329", true, false},
		{types.JSON(), types.Text(), "foo", "foo", true, false},
		{types.JSON(), types.Text(), 23.8013, "23.8013", true, false},
		{types.JSON(), types.Text(), json.Number("812"), "812", true, false},
		{types.JSON(), types.Text(), true, "true", true, false},
		{types.JSON(), types.Text(), json.RawMessage("null"), nil, true, false},
		{types.JSON(), types.Text(), json.RawMessage("null"), "", false, false},

		// Array.
		{types.Array(types.Int()), types.Array(types.Int()), []any{1, 2, 3}, []any{1, 2, 3}, true, false},
		{types.Array(types.Int()), types.Array(types.Int8()), []any{1, 2, 3}, []any{1, 2, 3}, true, false},
		{types.JSON(), types.Array(types.Int()), []any{1.0, 2.0, 3.0}, []any{1, 2, 3}, true, false},
		{types.JSON(), types.Array(types.Int()), []any{json.Number("1"), json.Number("2"), json.Number("3")}, []any{1, 2, 3}, true, false},
		{types.JSON(), types.Array(types.Int()), json.RawMessage(`[1,2,3]`), []any{1, 2, 3}, true, false},

		// Object.
		{
			types.Object([]types.Property{{Name: "foo", Type: types.Int()}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			types.Object([]types.Property{{Name: "foo", Type: types.Int()}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			map[string]any{"foo": 5, "boo": nil},
			map[string]any{"foo": 5, "boo": nil},
			true,
			false,
		},
		{
			types.JSON(),
			types.Object([]types.Property{{Name: "foo", Type: types.Int()}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			map[string]any{"foo": 5.0, "boo": nil},
			map[string]any{"foo": 5, "boo": nil},
			true,
			false,
		},
		{
			types.JSON(),
			types.Object([]types.Property{{Name: "foo", Type: types.Int()}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			map[string]any{"foo": json.Number("5"), "boo": nil},
			map[string]any{"foo": 5, "boo": nil},
			true,
			false,
		},
		{
			types.JSON(),
			types.Object([]types.Property{{Name: "foo", Type: types.Int()}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			json.RawMessage(`{"foo":5,"boo":null}`),
			map[string]any{"foo": 5, "boo": nil},
			true,
			false,
		},

		// Map.
		{types.Map(types.Boolean()), types.Map(types.Boolean()), map[string]any{"a": true, "b": false}, map[string]any{"a": true, "b": false}, true, false},
		{types.Map(types.Int16()), types.Map(types.Float32()), map[string]any{"a": 4032, "b": -721}, map[string]any{"a": float64(float32(4032)), "b": float64(float32(-721))}, true, false},
		{types.JSON(), types.Map(types.Float32()), map[string]any{"a": 4032.0, "b": -721.0}, map[string]any{"a": float64(float32(4032)), "b": float64(float32(-721))}, true, false},
		{types.JSON(), types.Map(types.Float32()), map[string]any{"a": json.Number("4032"), "b": json.Number("-721")}, map[string]any{"a": float64(float32(4032)), "b": float64(float32(-721))}, true, false},
		{types.JSON(), types.Map(types.Float32()), json.RawMessage(`{"a":4032,"b":-721}`), map[string]any{"a": float64(float32(4032)), "b": float64(float32(-721))}, true, false},
	}

	for _, test := range tests {
		got, err := convert(test.v, test.t1, test.t2, test.nullable, test.formatTime)
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
