//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"math"
	"regexp"
	"testing"
	"time"

	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/google/go-cmp/cmp"
)

func TestConvert(t *testing.T) {

	tests := []struct {
		t1, t2   types.Type
		value    any
		expected any
		nullable bool
		layouts  *state.TimeLayouts
	}{

		// boolean.
		{types.Boolean(), types.Boolean(), true, true, true, nil},
		{types.Boolean(), types.Boolean(), false, false, true, nil},
		{types.Int(8), types.Boolean(), 0, false, true, nil},
		{types.Int(8), types.Boolean(), 1, true, true, nil},
		{types.Int(8), types.Boolean(), -1, true, true, nil},
		{types.Uint(8), types.Boolean(), uint(0), false, true, nil},
		{types.Uint(8), types.Boolean(), uint(1), true, true, nil},
		{types.Text(), types.Boolean(), "false", false, true, nil},
		{types.Text(), types.Boolean(), "False", false, true, nil},
		{types.Text(), types.Boolean(), "FALSE", false, true, nil},
		{types.Text(), types.Boolean(), "no", false, true, nil},
		{types.Text(), types.Boolean(), "No", false, true, nil},
		{types.Text(), types.Boolean(), "NO", false, true, nil},
		{types.Text(), types.Boolean(), "true", true, true, nil},
		{types.Text(), types.Boolean(), "True", true, true, nil},
		{types.Text(), types.Boolean(), "TRUE", true, true, nil},
		{types.Text(), types.Boolean(), "yes", true, true, nil},
		{types.Text(), types.Boolean(), "Yes", true, true, nil},
		{types.Text(), types.Boolean(), "YES", true, true, nil},
		{types.JSON(), types.Boolean(), json.Value("false"), false, true, nil},
		{types.JSON(), types.Boolean(), json.Value("true"), true, true, nil},

		// int.
		{types.Int(32), types.Int(32), 831, 831, true, nil},
		{types.Int(32), types.Int(8), -123, -123, true, nil},
		{types.Int(32), types.Int(16), 2571, 2571, true, nil},
		{types.Int(32), types.Int(24), 670329, 670329, true, nil},
		{types.Int(32), types.Int(64), math.MaxInt64, math.MaxInt64, true, nil},
		{types.Uint(8), types.Int(32), uint(7), 7, true, nil},
		{types.Int(16), types.Int(32), -29, -29, true, nil},
		{types.Uint(24), types.Int(32), uint(89302), 89302, true, nil},
		{types.Int(64), types.Int(32), math.MaxInt32, math.MaxInt32, true, nil},
		{types.Float(64), types.Int(24), 10.0, 10, true, nil},
		{types.Float(64), types.Int(8), 34.4, 34, true, nil},
		{types.Float(64), types.Int(8), 34.5, 35, true, nil},
		{types.Float(64), types.Int(8), -0.49, 0, true, nil},
		{types.Float(32), types.Int(8), -0.5, -1, true, nil},
		{types.Float(64), types.Int(64), minFloatConvertibleToInt64, -9223372036854775808, true, nil},
		{types.Float(64), types.Int(64), maxFloatConvertibleToInt64, 9223372036854774784, true, nil},
		{types.Decimal(5, 3), types.Int(32), decimal.MustInt(5), 5, true, nil},
		{types.Decimal(5, 3), types.Int(8), decimal.MustParse("-12.0"), -12, true, nil},
		{types.Decimal(60, 0), types.Int(64), minIntDecimal, math.MinInt64, true, nil},
		{types.Decimal(60, 0), types.Int(64), maxIntDecimal, math.MaxInt64, true, nil},
		{types.Year(), types.Int(16), 2020, 2020, true, nil},
		{types.Text(), types.Int(32), "502842", 502842, true, nil},
		{types.Text(), types.Int(32), "", nil, true, nil},
		{types.JSON(), types.Int(32), json.Value("12.0"), 12, true, nil},
		{types.JSON(), types.Int(32), json.Value("-15"), -15, true, nil},
		{types.Boolean(), types.Int(8), false, 0, true, nil},
		{types.Boolean(), types.Int(8), true, 1, true, nil},

		// uint.
		{types.Int(32), types.Uint(32), 831, uint(831), true, nil},
		{types.Int(32), types.Uint(8), 218, uint(218), true, nil},
		{types.Int(32), types.Uint(16), 2571, uint(2571), true, nil},
		{types.Int(32), types.Uint(24), 670329, uint(670329), true, nil},
		{types.Uint(32), types.Uint(64), uint(math.MaxUint32), uint(math.MaxUint32), true, nil},
		{types.Uint(8), types.Uint(32), uint(7), uint(7), true, nil},
		{types.Uint(16), types.Uint(32), uint(29), uint(29), true, nil},
		{types.Uint(24), types.Uint(32), uint(89302), uint(89302), true, nil},
		{types.Float(64), types.Uint(64), maxFloatConvertibleToUint64, uint(18446744073709549568), true, nil},
		{types.Decimal(60, 0), types.Uint(64), maxUintDecimal, uint(math.MaxUint64), true, nil},
		{types.Year(), types.Uint(16), 2020, uint(2020), true, nil},
		{types.Text(), types.Uint(32), "502842", uint(502842), true, nil},
		{types.JSON(), types.Uint(32), json.Value("15"), uint(15), true, nil},
		{types.Boolean(), types.Uint(8), false, uint(0), true, nil},
		{types.Boolean(), types.Uint(8), true, uint(1), true, nil},

		// float.
		{types.Float(64), types.Float(64), 701.502830285, 701.502830285, true, nil},
		{types.Float(64), types.Float(32), 3.918347105316932e+10, float64(float32(3.918347e+10)), true, nil},
		{types.Float(32), types.Float(32), float64(float32(6316.0513)), float64(float32(6316.0513)), true, nil},
		{types.Float(32), types.Float(64), float64(float32(-32.04262)), -32.04262161254883, true, nil},
		{types.Int(32), types.Float(64), 5617072831, 5.617072831e+09, true, nil},
		{types.Uint(8), types.Float(32), uint(256), float64(float32(256)), true, nil},
		{types.Decimal(20, 10), types.Float(64), decimal.MustParse("767.5018382257"), 767.5018382257, true, nil},
		{types.Text(), types.Float(64), "767.5018382257", 767.5018382257, true, nil},
		{types.JSON(), types.Float(64), json.Value("767.5018382257"), 767.5018382257, true, nil},

		// decimal.
		{types.Int(32), types.Decimal(13, 3), math.MaxInt32, decimal.MustInt(math.MaxInt32), true, nil},
		{types.Int(32), types.Decimal(10, 0), math.MinInt32, decimal.MustInt(math.MinInt32), true, nil},
		{types.Uint(8), types.Decimal(3, 0), uint(math.MaxUint8), decimal.MustInt(math.MaxUint8), true, nil},
		{types.Float(64), types.Decimal(16, 5), 3.918347105316932e+10, decimal.MustParse("39183471053.16932"), true, nil},
		{types.Float(32), types.Decimal(15, 11), float64(float32(6316.0513)), decimal.MustParse("6316.05126953125"), true, nil},
		{types.Text(), types.Decimal(20, 10), "1048294.202936601", decimal.MustParse("1048294.202936601"), true, nil},
		{types.JSON(), types.Decimal(20, 10), json.Value("1048294.202936601"), decimal.MustParse("1048294.202936601"), true, nil},

		// datetime.
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), true, nil},
		{types.Date(), types.DateTime(), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, nil},
		{types.Text(), types.DateTime(), "2023-05-24T09:01:57.49361409Z", time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), true, nil},
		{types.Text(), types.DateTime(), "2023-05-24T09:01:57-07:00", time.Date(2023, 5, 24, 16, 1, 57, 0, time.UTC), true, nil},
		{types.JSON(), types.DateTime(), json.Value(`"2023-05-24T09:01:57-07:00"`), time.Date(2023, 5, 24, 16, 1, 57, 0, time.UTC), true, nil},
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), int64(1684918917), true, &state.TimeLayouts{DateTime: "unix"}},
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), int64(1684918917493), true, &state.TimeLayouts{DateTime: "unixmilli"}},
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), int64(1684918917493614), true, &state.TimeLayouts{DateTime: "unixmicro"}},
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), int64(1684918917493614090), true, &state.TimeLayouts{DateTime: "unixnano"}},
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), "Wednesday, 24-May-23 09:01:57 UTC", true, &state.TimeLayouts{DateTime: time.RFC850}},
		{types.Text(), types.DateTime(), "2023-05-24T09:01:57.49361409Z", "Wed, 24 May 2023 09:01:57 +0000", true, &state.TimeLayouts{DateTime: time.RFC1123Z}},
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), true, nil},

		// date.
		{types.Date(), types.Date(), time.Date(2023, 24, 5, 0, 0, 0, 0, time.UTC), time.Date(2023, 24, 5, 0, 0, 0, 0, time.UTC), true, nil},
		{types.DateTime(), types.Date(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, nil},
		{types.Text(), types.Date(), "2023-05-24", time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, nil},
		{types.JSON(), types.Date(), json.Value(`"2023-05-24"`), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, nil},
		{types.Date(), types.Date(), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), "2023-05-24", true, &state.TimeLayouts{Date: time.DateOnly}},
		{types.Text(), types.Date(), "2023-05-24", "05/24/2023", true, &state.TimeLayouts{Date: "01/02/2006"}},
		{types.Date(), types.Date(), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, nil},

		// time.
		{types.Time(), types.Time(), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, nil},
		{types.DateTime(), types.Time(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, nil},
		{types.Text(), types.Time(), "09:01:57.49361409Z", time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, nil},
		{types.Text(), types.Time(), "09:01:57", time.Date(1970, 1, 1, 9, 1, 57, 0, time.UTC), true, nil},
		{types.JSON(), types.Time(), json.Value(`"09:01:57"`), time.Date(1970, 1, 1, 9, 1, 57, 0, time.UTC), true, nil},
		{types.JSON(), types.Time(), json.Value(`"09:01:57.49361409Z"`), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, nil},
		{types.Time(), types.Time(), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), "09:01:57.493614", true, &state.TimeLayouts{Time: "15:04:05.999999"}},
		{types.Time(), types.Time(), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, nil},

		// year.
		{types.Year(), types.Year(), 2023, 2023, true, nil},
		{types.Int(16), types.Year(), 1, 1, true, nil},
		{types.Uint(64), types.Year(), uint(9999), 9999, true, nil},
		{types.Text(), types.Year(), "2023", 2023, true, nil},
		{types.Text(), types.Year(), "1", 1, true, nil},
		{types.JSON(), types.Year(), json.Value("1.0"), 1, true, nil},
		{types.JSON(), types.Year(), json.Value("2023.0"), 2023, true, nil},
		{types.JSON(), types.Year(), json.Value("2023"), 2023, true, nil},

		// uuid.
		{types.UUID(), types.UUID(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000", true, nil},
		{types.Text(), types.UUID(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000", true, nil},
		{types.JSON(), types.UUID(), json.Value(`"123e4567-e89b-12d3-a456-426614174000"`), "123e4567-e89b-12d3-a456-426614174000", true, nil},

		// json.
		{types.Int(32), types.JSON(), nil, nil, true, nil},
		{types.Int(32), types.JSON(), nil, json.Value(`null`), false, nil},
		{types.JSON(), types.JSON(), json.Value(`{"foo":5}`), json.Value(`{"foo":5}`), true, nil},
		{types.JSON(), types.JSON(), json.Value("null"), json.Value(`null`), true, nil},
		{types.Text(), types.JSON(), "", json.Value(`""`), false, nil},
		{types.JSON(), types.JSON(), json.Value("true"), json.Value("true"), true, nil},
		{types.JSON(), types.JSON(), json.Value(`"foo"`), json.Value(`"foo"`), true, nil},
		{types.JSON(), types.JSON(), json.Value("3.14"), json.Value("3.14"), true, nil},
		{types.JSON(), types.JSON(), json.Value("7204812694472.9355460893"), json.Value("7204812694472.9355460893"), true, nil},
		{types.JSON(), types.JSON(), json.Value(`{"foo":"boo"}`), json.Value(`{"foo":"boo"}`), true, nil},
		{types.JSON(), types.JSON(), json.Value(`[1,2,3]`), json.Value(`[1,2,3]`), true, nil},

		// inet.
		{types.Inet(), types.Inet(), "2001:db8::ff00:42:8329", "2001:db8::ff00:42:8329", true, nil},
		{types.Text(), types.Inet(), "2001:0db8:0000:0000:0000:ff00:0042:8329", "2001:db8::ff00:42:8329", true, nil},
		{types.JSON(), types.Inet(), json.Value(`"2001:0db8:0000:0000:0000:ff00:0042:8329"`), "2001:db8::ff00:42:8329", true, nil},

		// text.
		{types.Int(32), types.Text(), nil, nil, true, nil},
		{types.Text(), types.Text(), "foo", "foo", true, nil},
		{types.Text(), types.Text().WithValues("foo", "boo"), "", nil, true, nil},
		{types.Text(), types.Text().WithValues("foo", "boo"), "boo", "boo", true, nil},
		{types.Text(), types.Text().WithRegexp(regexp.MustCompile(`^bo+$`)), "", nil, true, nil},
		{types.Text(), types.Text().WithRegexp(regexp.MustCompile(`^bo+$`)), "boo", "boo", true, nil},
		{types.Boolean(), types.Text(), true, "true", true, nil},
		{types.Int(32), types.Text(), -603, "-603", true, nil},
		{types.Float(64), types.Text(), 7928301735.704827, "7.928301735704827e+09", true, nil},
		{types.Float(32), types.Text(), 3.14, "3.14", true, nil},
		{types.Decimal(5, 2), types.Text(), decimal.MustParse("120.79"), "120.79", true, nil},
		{types.DateTime(), types.Text(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), "2023-05-24T09:01:57.49361409Z", true, nil},
		{types.DateTime(), types.Text(), time.Date(2023, 5, 24, 9, 1, 57, 0, time.UTC), "2023-05-24T09:01:57Z", true, nil},
		{types.Date(), types.Text(), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), "2023-05-24", true, nil},
		{types.Time(), types.Text(), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), "09:01:57.49361409", true, nil},
		{types.Time(), types.Text(), time.Date(1970, 1, 1, 9, 1, 57, 0, time.UTC), "09:01:57", true, nil},
		{types.Year(), types.Text(), 1, "1", true, nil},
		{types.Year(), types.Text(), 2023, "2023", true, nil},
		{types.UUID(), types.Text(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000", true, nil},
		{types.Inet(), types.Text(), "2001:db8::ff00:42:8329", "2001:db8::ff00:42:8329", true, nil},
		{types.JSON(), types.Text(), json.Value(`"foo"`), "foo", true, nil},
		{types.JSON(), types.Text(), json.Value("23.8013"), "23.8013", true, nil},
		{types.JSON(), types.Text(), json.Value("812"), "812", true, nil},
		{types.JSON(), types.Text(), json.Value("true"), "true", true, nil},
		{types.JSON(), types.Text(), json.Value("null"), nil, true, nil},

		// array.
		{types.Array(types.Int(32)), types.Array(types.Int(32)), []any{1, 2, 3}, []any{1, 2, 3}, true, nil},
		{types.Array(types.Int(32)), types.Array(types.Int(8)), []any{1, 2, 3}, []any{1, 2, 3}, true, nil},
		{types.Array(types.Text()), types.Array(types.Text()), []any{"123e4567-e89b-12d3-a456-426614174000"}, []any{"123e4567-e89b-12d3-a456-426614174000"}, true, nil},
		{types.JSON(), types.Array(types.Int(32)), json.Value("[1.0,2.0,3.0]"), []any{1, 2, 3}, true, nil},
		{types.JSON(), types.Array(types.Int(32)), json.Value("[1,2,3]"), []any{1, 2, 3}, true, nil},
		{types.JSON(), types.Array(types.Int(32)), json.Value("6.0"), []any{6}, true, nil},
		{types.JSON(), types.Array(types.Boolean()), json.Value("true"), []any{true}, true, nil},
		{types.JSON(), types.Array(types.Text()), json.Value(`"foo"`), []any{"foo"}, true, nil},
		{types.JSON(), types.Array(types.Float(64)), json.Value(`15.07`), []any{15.07}, true, nil},

		// object.
		{
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			map[string]any{"foo": 5, "boo": nil},
			map[string]any{"foo": 5, "boo": nil},
			true,
			nil,
		},
		{
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}, {Name: "boo", Type: types.Text()}}),
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32), CreateRequired: true}, {Name: "boo", Type: types.Text()}}),
			map[string]any{"foo": 5},
			map[string]any{"foo": 5},
			true,
			nil,
		},
		{
			types.JSON(),
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			json.Value(`{"foo":5.0,"boo":null}`),
			map[string]any{"foo": 5, "boo": nil},
			true,
			nil,
		},
		{
			types.JSON(),
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			json.Value(`{"foo":5,"boo":null}`),
			map[string]any{"foo": 5, "boo": nil},
			true,
			nil,
		},
		{
			types.JSON(),
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}}),
			json.Value(`{"@":7,"foo":8}`),
			map[string]any{"foo": 8},
			true,
			nil,
		},
		{
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}}),
			types.Map(types.Text()),
			map[string]any{"foo": 572},
			map[string]any{"foo": "572"},
			true,
			nil,
		},

		// map.
		{types.Map(types.Boolean()), types.Map(types.Boolean()), map[string]any{"a": true, "b": false}, map[string]any{"a": true, "b": false}, true, nil},
		{types.Map(types.Int(16)), types.Map(types.Float(32)), map[string]any{"a": 4032, "b": -721}, map[string]any{"a": float64(float32(4032)), "b": float64(float32(-721))}, true, nil},
		{types.JSON(), types.Map(types.Float(32)), json.Value(`{"a":4032,"b":-721}`), map[string]any{"a": float64(float32(4032)), "b": float64(float32(-721))}, true, nil},
		{types.JSON(), types.Map(types.Float(32)), json.Value(`{"a":4032,"b":-721}`), map[string]any{"a": float64(float32(4032)), "b": float64(float32(-721))}, true, nil},
		{types.Map(types.Boolean()), types.Object([]types.Property{{Name: "x", Type: types.Text()}, {Name: "y", Type: types.Boolean()}}), map[string]any{"x": true, "y": false}, map[string]any{"x": "true", "y": false}, true, nil},
	}

	for _, test := range tests {
		got, err := convert(test.value, test.t1, test.t2, test.nullable, false, test.layouts, Create)
		if err != nil {
			t.Fatalf("cannot convert %s<%v> to type %s", test.t1, test.value, test.t2)
		}
		if !cmp.Equal(test.expected, got) {
			if f, ok := test.expected.(float64); ok && math.IsNaN(f) {
				if f, ok := got.(float64); ok && math.IsNaN(f) {
					continue
				}
			}
			t.Fatalf("expected %T(%v), got %T(%v)", test.expected, test.expected, got, got)
		}
	}

}
