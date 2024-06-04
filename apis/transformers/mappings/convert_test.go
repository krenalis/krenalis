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

	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/types"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"
)

func TestConvert(t *testing.T) {

	tests := []struct {
		t1, t2   types.Type
		v        any
		e        any
		nullable bool
		layouts  *state.TimeLayouts
	}{

		// Boolean.
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
		{types.JSON(), types.Boolean(), false, false, true, nil},
		{types.JSON(), types.Boolean(), true, true, true, nil},
		{types.JSON(), types.Boolean(), json.RawMessage("false"), false, true, nil},
		{types.JSON(), types.Boolean(), json.RawMessage("true"), true, true, nil},

		// Int8, Int16, Int24, Int32, and Int64.
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
		{types.Decimal(5, 3), types.Int(32), decimal.RequireFromString("5"), 5, true, nil},
		{types.Decimal(5, 3), types.Int(8), decimal.RequireFromString("-12.0"), -12, true, nil},
		{types.Decimal(60, 0), types.Int(64), minIntDecimal, math.MinInt64, true, nil},
		{types.Decimal(60, 0), types.Int(64), maxIntDecimal, math.MaxInt64, true, nil},
		{types.Year(), types.Int(16), 2020, 2020, true, nil},
		{types.Text(), types.Int(32), "502842", 502842, true, nil},
		{types.Text(), types.Int(32), "", nil, true, nil},
		{types.JSON(), types.Int(32), 12.627, 13, true, nil},
		{types.JSON(), types.Int(32), json.Number("-15"), -15, true, nil},
		{types.JSON(), types.Int(32), json.RawMessage("-2"), -2, true, nil},
		{types.Boolean(), types.Int(8), false, 0, true, nil},
		{types.Boolean(), types.Int(8), true, 1, true, nil},

		// Uint8, Uint16, Uint24, Uint32, and Uint64.
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
		{types.JSON(), types.Uint(64), maxFloatConvertibleToUint64, uint(18446744073709549568), true, nil},
		{types.JSON(), types.Uint(32), json.Number("15"), uint(15), true, nil},
		{types.JSON(), types.Uint(32), json.RawMessage("2"), uint(2), true, nil},
		{types.Boolean(), types.Uint(8), false, uint(0), true, nil},
		{types.Boolean(), types.Uint(8), true, uint(1), true, nil},

		// Float32 and Float64.
		{types.Float(64), types.Float(64), 701.502830285, 701.502830285, true, nil},
		{types.Float(64), types.Float(32), 3.918347105316932e+10, float64(float32(3.918347e+10)), true, nil},
		{types.Float(32), types.Float(32), float64(float32(6316.0513)), float64(float32(6316.0513)), true, nil},
		{types.Float(32), types.Float(64), float64(float32(-32.04262)), -32.04262161254883, true, nil},
		{types.Int(32), types.Float(64), 5617072831, 5.617072831e+09, true, nil},
		{types.Uint(8), types.Float(32), uint(256), float64(float32(256)), true, nil},
		{types.Decimal(20, 10), types.Float(64), decimal.RequireFromString("767.5018382257"), 767.5018382257, true, nil},
		{types.Text(), types.Float(64), "767.5018382257", 767.5018382257, true, nil},
		{types.JSON(), types.Float(64), 767.5018382257, 767.5018382257, true, nil},
		{types.JSON(), types.Float(64), json.Number("767.5018382257"), 767.5018382257, true, nil},
		{types.JSON(), types.Float(64), json.RawMessage("767.5018382257"), 767.5018382257, true, nil},

		// Decimal.
		{types.Int(32), types.Decimal(10, 3), math.MaxInt32, decimal.NewFromInt(math.MaxInt32), true, nil},
		{types.Int(32), types.Decimal(10, 0), math.MinInt32, decimal.NewFromInt(math.MinInt32), true, nil},
		{types.Uint(8), types.Decimal(3, 0), uint(math.MaxUint8), decimal.NewFromInt(math.MaxUint8), true, nil},
		{types.Float(64), types.Decimal(3, 0), 3.918347105316932e+10, decimal.RequireFromString("39183471053.16932"), true, nil},
		{types.Float(32), types.Decimal(3, 0), float64(float32(6316.0513)), decimal.RequireFromString("6316.05126953125"), true, nil},
		{types.Text(), types.Decimal(20, 10), "1048294.202936601", decimal.RequireFromString("1048294.202936601"), true, nil},
		{types.JSON(), types.Decimal(20, 10), 1048294.202936601, decimal.RequireFromString("1048294.202936601"), true, nil},
		{types.JSON(), types.Decimal(20, 10), json.Number("1048294.202936601"), decimal.RequireFromString("1048294.202936601"), true, nil},
		{types.JSON(), types.Decimal(20, 10), json.RawMessage("1048294.202936601"), decimal.RequireFromString("1048294.202936601"), true, nil},

		// DateTime.
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), true, nil},
		{types.Date(), types.DateTime(), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, nil},
		{types.Text(), types.DateTime(), "2023-05-24T09:01:57.49361409Z", time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), true, nil},
		{types.Text(), types.DateTime(), "2023-05-24T09:01:57-07:00", time.Date(2023, 5, 24, 16, 1, 57, 0, time.UTC), true, nil},
		{types.JSON(), types.DateTime(), "2023-05-24T09:01:57-07:00", time.Date(2023, 5, 24, 16, 1, 57, 0, time.UTC), true, nil},
		{types.JSON(), types.DateTime(), json.RawMessage(`"2023-05-24T09:01:57-07:00"`), time.Date(2023, 5, 24, 16, 1, 57, 0, time.UTC), true, nil},
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), int64(1684918917), true, &state.TimeLayouts{DateTime: "unix"}},
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), int64(1684918917493), true, &state.TimeLayouts{DateTime: "unixmilli"}},
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), int64(1684918917493614), true, &state.TimeLayouts{DateTime: "unixmicro"}},
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), int64(1684918917493614090), true, &state.TimeLayouts{DateTime: "unixnano"}},
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), "Wednesday, 24-May-23 09:01:57 UTC", true, &state.TimeLayouts{DateTime: time.RFC850}},
		{types.Text(), types.DateTime(), "2023-05-24T09:01:57.49361409Z", "Wed, 24 May 2023 09:01:57 +0000", true, &state.TimeLayouts{DateTime: time.RFC1123Z}},
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), true, nil},

		// Date.
		{types.Date(), types.Date(), time.Date(2023, 24, 5, 0, 0, 0, 0, time.UTC), time.Date(2023, 24, 5, 0, 0, 0, 0, time.UTC), true, nil},
		{types.DateTime(), types.Date(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, nil},
		{types.Text(), types.Date(), "2023-05-24", time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, nil},
		{types.JSON(), types.Date(), "2023-05-24", time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, nil},
		{types.JSON(), types.Date(), json.RawMessage(`"2023-05-24"`), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, nil},
		{types.Date(), types.Date(), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), "2023-05-24", true, &state.TimeLayouts{Date: time.DateOnly}},
		{types.Text(), types.Date(), "2023-05-24", "05/24/2023", true, &state.TimeLayouts{Date: "01/02/2006"}},
		{types.Date(), types.Date(), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, nil},

		// Time.
		{types.Time(), types.Time(), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, nil},
		{types.DateTime(), types.Time(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, nil},
		{types.Text(), types.Time(), "09:01:57.49361409Z", time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, nil},
		{types.Text(), types.Time(), "09:01:57", time.Date(1970, 1, 1, 9, 1, 57, 0, time.UTC), true, nil},
		{types.JSON(), types.Time(), "09:01:57.49361409Z", time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, nil},
		{types.JSON(), types.Time(), "09:01:57", time.Date(1970, 1, 1, 9, 1, 57, 0, time.UTC), true, nil},
		{types.JSON(), types.Time(), json.RawMessage(`"09:01:57.49361409Z"`), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, nil},
		{types.Time(), types.Time(), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), "09:01:57.493614", true, &state.TimeLayouts{Time: "15:04:05.999999"}},
		{types.Time(), types.Time(), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, nil},

		// Year.
		{types.Year(), types.Year(), 2023, 2023, true, nil},
		{types.Int(16), types.Year(), 1, 1, true, nil},
		{types.Uint(64), types.Year(), uint(9999), 9999, true, nil},
		{types.Text(), types.Year(), "2023", 2023, true, nil},
		{types.Text(), types.Year(), "1", 1, true, nil},
		{types.JSON(), types.Year(), 1.0, 1, true, nil},
		{types.JSON(), types.Year(), 2023.0, 2023, true, nil},
		{types.JSON(), types.Year(), json.Number("2023"), 2023, true, nil},
		{types.JSON(), types.Year(), json.RawMessage("2023"), 2023, true, nil},

		// UUID.
		{types.UUID(), types.UUID(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000", true, nil},
		{types.Text(), types.UUID(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000", true, nil},
		{types.JSON(), types.UUID(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000", true, nil},
		{types.JSON(), types.UUID(), json.RawMessage(`"123e4567-e89b-12d3-a456-426614174000"`), "123e4567-e89b-12d3-a456-426614174000", true, nil},

		// JSON.
		{types.Int(32), types.JSON(), nil, nil, true, nil},
		{types.Int(32), types.JSON(), nil, json.RawMessage(`null`), false, nil},
		{types.JSON(), types.JSON(), json.RawMessage(`{"foo":5}`), json.RawMessage(`{"foo":5}`), true, nil},
		{types.JSON(), types.JSON(), json.RawMessage(`{"foo":5}`), json.RawMessage(`{"foo":5}`), true, nil},
		{types.JSON(), types.JSON(), json.RawMessage(`null`), json.RawMessage(`null`), true, nil},
		{types.Text(), types.JSON(), "", json.RawMessage("null"), false, nil},
		{types.JSON(), types.JSON(), true, json.RawMessage(`true`), true, nil},
		{types.JSON(), types.JSON(), "foo", json.RawMessage(`"foo"`), true, nil},
		{types.JSON(), types.JSON(), 3.14, json.RawMessage(`3.14`), true, nil},
		{types.JSON(), types.JSON(), json.Number("7204812694472.9355460893"), json.RawMessage(`7204812694472.9355460893`), true, nil},
		{types.JSON(), types.JSON(), map[string]any{"foo": "boo"}, json.RawMessage(`{"foo":"boo"}`), true, nil},
		{types.JSON(), types.JSON(), []any{1, 2, 3}, json.RawMessage(`[1,2,3]`), true, nil},

		// Inet.
		{types.Inet(), types.Inet(), "2001:db8::ff00:42:8329", "2001:db8::ff00:42:8329", true, nil},
		{types.Text(), types.Inet(), "2001:0db8:0000:0000:0000:ff00:0042:8329", "2001:db8::ff00:42:8329", true, nil},
		{types.JSON(), types.Inet(), "2001:0db8:0000:0000:0000:ff00:0042:8329", "2001:db8::ff00:42:8329", true, nil},
		{types.JSON(), types.Inet(), json.RawMessage(`"2001:0db8:0000:0000:0000:ff00:0042:8329"`), "2001:db8::ff00:42:8329", true, nil},

		// Text.
		{types.Int(32), types.Text(), nil, nil, true, nil},
		{types.Int(32), types.Text(), nil, "", false, nil},
		{types.Text(), types.Text(), "foo", "foo", true, nil},
		{types.Text(), types.Text().WithValues("foo", "boo"), "", nil, true, nil},
		{types.Text(), types.Text().WithValues("foo", "boo"), "boo", "boo", true, nil},
		{types.Text(), types.Text().WithRegexp(regexp.MustCompile(`^bo+$`)), "", nil, true, nil},
		{types.Text(), types.Text().WithRegexp(regexp.MustCompile(`^bo+$`)), "boo", "boo", true, nil},
		{types.Boolean(), types.Text(), true, "true", true, nil},
		{types.Int(32), types.Text(), -603, "-603", true, nil},
		{types.Float(64), types.Text(), 7928301735.704827, "7.928301735704827e+09", true, nil},
		{types.Float(32), types.Text(), 3.14, "3.14", true, nil},
		{types.Decimal(5, 2), types.Text(), decimal.RequireFromString("120.79"), "120.79", true, nil},
		{types.DateTime(), types.Text(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), "2023-05-24T09:01:57.49361409Z", true, nil},
		{types.DateTime(), types.Text(), time.Date(2023, 5, 24, 9, 1, 57, 0, time.UTC), "2023-05-24T09:01:57Z", true, nil},
		{types.Date(), types.Text(), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), "2023-05-24", true, nil},
		{types.Time(), types.Text(), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), "09:01:57.49361409", true, nil},
		{types.Time(), types.Text(), time.Date(1970, 1, 1, 9, 1, 57, 0, time.UTC), "09:01:57", true, nil},
		{types.Year(), types.Text(), 1, "1", true, nil},
		{types.Year(), types.Text(), 2023, "2023", true, nil},
		{types.UUID(), types.Text(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000", true, nil},
		{types.Inet(), types.Text(), "2001:db8::ff00:42:8329", "2001:db8::ff00:42:8329", true, nil},
		{types.JSON(), types.Text(), "foo", "foo", true, nil},
		{types.JSON(), types.Text(), 23.8013, "23.8013", true, nil},
		{types.JSON(), types.Text(), json.Number("812"), "812", true, nil},
		{types.JSON(), types.Text(), true, "true", true, nil},
		{types.JSON(), types.Text(), json.RawMessage("null"), nil, true, nil},
		{types.JSON(), types.Text(), json.RawMessage("null"), "", false, nil},

		// Array.
		{types.Array(types.Int(32)), types.Array(types.Int(32)), []any{1, 2, 3}, []any{1, 2, 3}, true, nil},
		{types.Array(types.Int(32)), types.Array(types.Int(8)), []any{1, 2, 3}, []any{1, 2, 3}, true, nil},
		{types.Int(32), types.Array(types.Int(8)), 5, []any{5}, true, nil},
		{types.Boolean(), types.Array(types.Boolean()).WithMinItems(1), false, []any{false}, true, nil},
		{types.Text(), types.Array(types.UUID()), "123e4567-e89b-12d3-a456-426614174000", []any{"123e4567-e89b-12d3-a456-426614174000"}, true, nil},
		{types.JSON(), types.Array(types.Int(32)), []any{1.0, 2.0, 3.0}, []any{1, 2, 3}, true, nil},
		{types.JSON(), types.Array(types.Int(32)), []any{json.Number("1"), json.Number("2"), json.Number("3")}, []any{1, 2, 3}, true, nil},
		{types.JSON(), types.Array(types.Int(32)), json.RawMessage(`[1,2,3]`), []any{1, 2, 3}, true, nil},
		{types.JSON(), types.Array(types.Int(32)), 6.0, []any{6}, true, nil},
		{types.JSON(), types.Array(types.Boolean()), true, []any{true}, true, nil},
		{types.JSON(), types.Array(types.Text()), "foo", []any{"foo"}, true, nil},
		{types.JSON(), types.Array(types.Float(64)), json.RawMessage(`15.07`), []any{15.07}, true, nil},

		// Object.
		{
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			map[string]any{"foo": 5, "boo": nil},
			map[string]any{"foo": 5, "boo": nil},
			true,
			nil,
		},
		{
			types.JSON(),
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			map[string]any{"foo": 5.0, "boo": nil},
			map[string]any{"foo": 5, "boo": nil},
			true,
			nil,
		},
		{
			types.JSON(),
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			map[string]any{"foo": json.Number("5"), "boo": nil},
			map[string]any{"foo": 5, "boo": nil},
			true,
			nil,
		},
		{
			types.JSON(),
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}, {Name: "boo", Type: types.Text(), Nullable: true}}),
			json.RawMessage(`{"foo":5,"boo":null}`),
			map[string]any{"foo": 5, "boo": nil},
			true,
			nil,
		},

		// Map.
		{types.Map(types.Boolean()), types.Map(types.Boolean()), map[string]any{"a": true, "b": false}, map[string]any{"a": true, "b": false}, true, nil},
		{types.Map(types.Int(16)), types.Map(types.Float(32)), map[string]any{"a": 4032, "b": -721}, map[string]any{"a": float64(float32(4032)), "b": float64(float32(-721))}, true, nil},
		{types.JSON(), types.Map(types.Float(32)), map[string]any{"a": 4032.0, "b": -721.0}, map[string]any{"a": float64(float32(4032)), "b": float64(float32(-721))}, true, nil},
		{types.JSON(), types.Map(types.Float(32)), map[string]any{"a": json.Number("4032"), "b": json.Number("-721")}, map[string]any{"a": float64(float32(4032)), "b": float64(float32(-721))}, true, nil},
		{types.JSON(), types.Map(types.Float(32)), json.RawMessage(`{"a":4032,"b":-721}`), map[string]any{"a": float64(float32(4032)), "b": float64(float32(-721))}, true, nil},
	}

	for _, test := range tests {
		got, err := convert(test.v, test.t1, test.t2, test.nullable, test.layouts)
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
