// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/google/go-cmp/cmp"
)

func TestConvert(t *testing.T) {

	emptyEnum := types.String().WithValues("", "yes")
	tests := []struct {
		t1, t2   types.Type
		value    any
		expected any
		nullable bool
		layouts  *state.TimeLayouts
		err      error
	}{

		// string.
		{types.Int(32), types.String(), nil, nil, true, nil, nil},
		{types.String(), types.String(), "foo", "foo", true, nil, nil},
		{types.String(), types.String().WithValues("foo", "boo"), "", nil, true, nil, nil},
		{types.String(), types.String().WithValues("foo", "boo"), "boo", "boo", true, nil, nil},
		{types.String(), emptyEnum, "", "", true, nil, nil},
		{types.String(), emptyEnum, "", "", false, nil, nil},
		{emptyEnum, emptyEnum, "", "", true, nil, nil},
		{types.JSON(), emptyEnum, json.Value(`""`), "", true, nil, nil},
		{types.String(), emptyEnum, nil, nil, true, nil, nil},
		{types.JSON(), emptyEnum, json.Value("null"), nil, true, nil, nil},
		{types.String(), types.String().WithValues("yes"), "", nil, false, nil, errEnumConversion},
		{types.String(), types.String().WithPattern(regexp.MustCompile(`^bo+$`)), "", nil, true, nil, nil},
		{types.String(), types.String().WithPattern(regexp.MustCompile(`^bo+$`)), "boo", "boo", true, nil, nil},
		{types.Boolean(), types.String(), true, "true", true, nil, nil},
		{types.Int(32), types.String(), -603, "-603", true, nil, nil},
		{types.Float(64), types.String(), 7928301735.704827, "7.928301735704827e+09", true, nil, nil},
		{types.Float(32), types.String(), 3.14, "3.14", true, nil, nil},
		{types.Decimal(5, 2), types.String(), decimal.MustParse("120.79"), "120.79", true, nil, nil},
		{types.DateTime(), types.String(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), "2023-05-24T09:01:57.49361409Z", true, nil, nil},
		{types.DateTime(), types.String(), time.Date(2023, 5, 24, 9, 1, 57, 0, time.UTC), "2023-05-24T09:01:57Z", true, nil, nil},
		{types.Date(), types.String(), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), "2023-05-24", true, nil, nil},
		{types.Time(), types.String(), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), "09:01:57.49361409", true, nil, nil},
		{types.Time(), types.String(), time.Date(1970, 1, 1, 9, 1, 57, 0, time.UTC), "09:01:57", true, nil, nil},
		{types.Year(), types.String(), 1, "1", true, nil, nil},
		{types.Year(), types.String(), 2023, "2023", true, nil, nil},
		{types.UUID(), types.String(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000", true, nil, nil},
		{types.IP(), types.String(), "2001:db8::ff00:42:8329", "2001:db8::ff00:42:8329", true, nil, nil},
		{types.JSON(), types.String(), json.Value(`"foo"`), "foo", true, nil, nil},
		{types.JSON(), types.String(), json.Value("23.8013"), "23.8013", true, nil, nil},
		{types.JSON(), types.String(), json.Value("812"), "812", true, nil, nil},
		{types.JSON(), types.String(), json.Value("true"), "true", true, nil, nil},
		{types.JSON(), types.String(), json.Value("null"), nil, true, nil, nil},

		// boolean.
		{types.Boolean(), types.Boolean(), true, true, true, nil, nil},
		{types.Boolean(), types.Boolean(), false, false, true, nil, nil},
		{types.Int(8), types.Boolean(), 0, false, true, nil, nil},
		{types.Int(8), types.Boolean(), 1, true, true, nil, nil},
		{types.Int(8), types.Boolean(), -1, true, true, nil, nil},
		{types.Int(8).Unsigned(), types.Boolean(), uint(0), false, true, nil, nil},
		{types.Int(8).Unsigned(), types.Boolean(), uint(1), true, true, nil, nil},
		{types.String(), types.Boolean(), "false", false, true, nil, nil},
		{types.String(), types.Boolean(), "False", false, true, nil, nil},
		{types.String(), types.Boolean(), "FALSE", false, true, nil, nil},
		{types.String(), types.Boolean(), "no", false, true, nil, nil},
		{types.String(), types.Boolean(), "No", false, true, nil, nil},
		{types.String(), types.Boolean(), "NO", false, true, nil, nil},
		{types.String(), types.Boolean(), "true", true, true, nil, nil},
		{types.String(), types.Boolean(), "True", true, true, nil, nil},
		{types.String(), types.Boolean(), "TRUE", true, true, nil, nil},
		{types.String(), types.Boolean(), "yes", true, true, nil, nil},
		{types.String(), types.Boolean(), "Yes", true, true, nil, nil},
		{types.String(), types.Boolean(), "YES", true, true, nil, nil},
		{types.JSON(), types.Boolean(), json.Value("false"), false, true, nil, nil},
		{types.JSON(), types.Boolean(), json.Value("true"), true, true, nil, nil},

		// int.
		{types.Int(32), types.Int(32), 831, 831, true, nil, nil},
		{types.Int(32), types.Int(8), -123, -123, true, nil, nil},
		{types.Int(32), types.Int(16), 2571, 2571, true, nil, nil},
		{types.Int(32), types.Int(24), 670329, 670329, true, nil, nil},
		{types.Int(32), types.Int(64), math.MaxInt64, math.MaxInt64, true, nil, nil},
		{types.Int(8).Unsigned(), types.Int(32), uint(7), 7, true, nil, nil},
		{types.Int(16), types.Int(32), -29, -29, true, nil, nil},
		{types.Int(24).Unsigned(), types.Int(32), uint(89302), 89302, true, nil, nil},
		{types.Int(64), types.Int(32), math.MaxInt32, math.MaxInt32, true, nil, nil},
		{types.Float(64), types.Int(24), 10.0, 10, true, nil, nil},
		{types.Float(64), types.Int(8), 34.4, 34, true, nil, nil},
		{types.Float(64), types.Int(8), 34.5, 35, true, nil, nil},
		{types.Float(64), types.Int(8), -0.49, 0, true, nil, nil},
		{types.Float(32), types.Int(8), -0.5, -1, true, nil, nil},
		{types.Float(64), types.Int(64), minFloatConvertibleToInt64, -9223372036854775808, true, nil, nil},
		{types.Float(64), types.Int(64), maxFloatConvertibleToInt64, 9223372036854774784, true, nil, nil},
		{types.Decimal(5, 3), types.Int(32), decimal.MustInt(5), 5, true, nil, nil},
		{types.Decimal(5, 3), types.Int(8), decimal.MustParse("-12.0"), -12, true, nil, nil},
		{types.Decimal(60, 0), types.Int(64), minIntDecimal, math.MinInt64, true, nil, nil},
		{types.Decimal(60, 0), types.Int(64), maxIntDecimal, math.MaxInt64, true, nil, nil},
		{types.Year(), types.Int(16), 2020, 2020, true, nil, nil},
		{types.String(), types.Int(32), "502842", 502842, true, nil, nil},
		{types.String(), types.Int(32), "", nil, true, nil, nil},
		{types.JSON(), types.Int(32), json.Value("12.0"), 12, true, nil, nil},
		{types.JSON(), types.Int(32), json.Value("-15"), -15, true, nil, nil},
		{types.Boolean(), types.Int(8), false, 0, true, nil, nil},
		{types.Boolean(), types.Int(8), true, 1, true, nil, nil},

		// unsigned int.
		{types.Int(32), types.Int(32).Unsigned(), 831, uint(831), true, nil, nil},
		{types.Int(32), types.Int(8).Unsigned(), 218, uint(218), true, nil, nil},
		{types.Int(32), types.Int(16).Unsigned(), 2571, uint(2571), true, nil, nil},
		{types.Int(32), types.Int(24).Unsigned(), 670329, uint(670329), true, nil, nil},
		{types.Int(32).Unsigned(), types.Int(64).Unsigned(), uint(math.MaxUint32), uint(math.MaxUint32), true, nil, nil},
		{types.Int(8).Unsigned(), types.Int(32).Unsigned(), uint(7), uint(7), true, nil, nil},
		{types.Int(16).Unsigned(), types.Int(32).Unsigned(), uint(29), uint(29), true, nil, nil},
		{types.Int(24).Unsigned(), types.Int(32).Unsigned(), uint(89302), uint(89302), true, nil, nil},
		{types.Float(64), types.Int(64).Unsigned(), maxFloatConvertibleToUint64, uint(18446744073709549568), true, nil, nil},
		{types.Decimal(60, 0), types.Int(64).Unsigned(), maxUintDecimal, uint(math.MaxUint64), true, nil, nil},
		{types.Year(), types.Int(16).Unsigned(), 2020, uint(2020), true, nil, nil},
		{types.String(), types.Int(32).Unsigned(), "502842", uint(502842), true, nil, nil},
		{types.JSON(), types.Int(32).Unsigned(), json.Value("15"), uint(15), true, nil, nil},
		{types.Boolean(), types.Int(8).Unsigned(), false, uint(0), true, nil, nil},
		{types.Boolean(), types.Int(8).Unsigned(), true, uint(1), true, nil, nil},

		// float.
		{types.Float(64), types.Float(64), 701.502830285, 701.502830285, true, nil, nil},
		{types.Float(64), types.Float(32), 3.918347105316932e+10, float64(float32(3.918347e+10)), true, nil, nil},
		{types.Float(32), types.Float(32), float64(float32(6316.0513)), float64(float32(6316.0513)), true, nil, nil},
		{types.Float(32), types.Float(64), float64(float32(-32.04262)), -32.04262161254883, true, nil, nil},
		{types.Int(32), types.Float(64), 5617072831, 5.617072831e+09, true, nil, nil},
		{types.Int(8).Unsigned(), types.Float(32), uint(256), float64(float32(256)), true, nil, nil},
		{types.Decimal(20, 10), types.Float(64), decimal.MustParse("767.5018382257"), 767.5018382257, true, nil, nil},
		{types.String(), types.Float(64), "767.5018382257", 767.5018382257, true, nil, nil},
		{types.JSON(), types.Float(64), json.Value("767.5018382257"), 767.5018382257, true, nil, nil},

		// decimal.
		{types.Int(32), types.Decimal(13, 3), math.MaxInt32, decimal.MustInt(math.MaxInt32), true, nil, nil},
		{types.Int(32), types.Decimal(10, 0), math.MinInt32, decimal.MustInt(math.MinInt32), true, nil, nil},
		{types.Int(8).Unsigned(), types.Decimal(3, 0), uint(math.MaxUint8), decimal.MustInt(math.MaxUint8), true, nil, nil},
		{types.Float(64), types.Decimal(16, 5), 3.918347105316932e+10, decimal.MustParse("39183471053.16932"), true, nil, nil},
		{types.Float(32), types.Decimal(15, 11), float64(float32(6316.0513)), decimal.MustParse("6316.05126953125"), true, nil, nil},
		{types.String(), types.Decimal(20, 10), "1048294.202936601", decimal.MustParse("1048294.202936601"), true, nil, nil},
		{types.JSON(), types.Decimal(20, 10), json.Value("1048294.202936601"), decimal.MustParse("1048294.202936601"), true, nil, nil},

		// datetime.
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), true, nil, nil},
		{types.Date(), types.DateTime(), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, nil, nil},
		{types.String(), types.DateTime(), "2023-05-24T09:01:57.49361409Z", time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), true, nil, nil},
		{types.String(), types.DateTime(), "2023-05-24T09:01:57-07:00", time.Date(2023, 5, 24, 16, 1, 57, 0, time.UTC), true, nil, nil},
		{types.JSON(), types.DateTime(), json.Value(`"2023-05-24T09:01:57-07:00"`), time.Date(2023, 5, 24, 16, 1, 57, 0, time.UTC), true, nil, nil},
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), int64(1684918917), true, &state.TimeLayouts{DateTime: "unix"}, nil},
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), int64(1684918917493), true, &state.TimeLayouts{DateTime: "unixmilli"}, nil},
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), int64(1684918917493614), true, &state.TimeLayouts{DateTime: "unixmicro"}, nil},
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), int64(1684918917493614090), true, &state.TimeLayouts{DateTime: "unixnano"}, nil},
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), "Wednesday, 24-May-23 09:01:57 UTC", true, &state.TimeLayouts{DateTime: time.RFC850}, nil},
		{types.String(), types.DateTime(), "2023-05-24T09:01:57.49361409Z", "Wed, 24 May 2023 09:01:57 +0000", true, &state.TimeLayouts{DateTime: time.RFC1123Z}, nil},
		{types.DateTime(), types.DateTime(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), true, nil, nil},

		// date.
		{types.Date(), types.Date(), time.Date(2023, 24, 5, 0, 0, 0, 0, time.UTC), time.Date(2023, 24, 5, 0, 0, 0, 0, time.UTC), true, nil, nil},
		{types.DateTime(), types.Date(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, nil, nil},
		{types.String(), types.Date(), "2023-05-24", time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, nil, nil},
		{types.JSON(), types.Date(), json.Value(`"2023-05-24"`), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, nil, nil},
		{types.Date(), types.Date(), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), "2023-05-24", true, &state.TimeLayouts{Date: time.DateOnly}, nil},
		{types.String(), types.Date(), "2023-05-24", "05/24/2023", true, &state.TimeLayouts{Date: "01/02/2006"}, nil},
		{types.Date(), types.Date(), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), true, nil, nil},

		// time.
		{types.Boolean(), types.Time(), true, nil, true, nil, errInvalidConversion},
		{types.Int(32), types.Time(), 42, nil, true, nil, errInvalidConversion},
		{types.Float(64), types.Time(), 1.5, nil, true, nil, errInvalidConversion},
		{types.Decimal(6, 2), types.Time(), decimal.MustParse("1.5"), nil, true, nil, errInvalidConversion},
		{types.Year(), types.Time(), 2026, nil, true, nil, errInvalidConversion},
		{types.Time(), types.Time(), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, nil, nil},
		{types.DateTime(), types.Time(), time.Date(2023, 5, 24, 9, 1, 57, 493614090, time.UTC), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, nil, nil},
		{types.String(), types.Time(), "09:01:57.49361409Z", time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, nil, nil},
		{types.String(), types.Time(), "09:01:57", time.Date(1970, 1, 1, 9, 1, 57, 0, time.UTC), true, nil, nil},
		{types.JSON(), types.Time(), json.Value(`"09:01:57"`), time.Date(1970, 1, 1, 9, 1, 57, 0, time.UTC), true, nil, nil},
		{types.JSON(), types.Time(), json.Value(`"09:01:57.49361409Z"`), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, nil, nil},
		{types.Time(), types.Time(), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), "09:01:57.493614", true, &state.TimeLayouts{Time: "15:04:05.999999"}, nil},
		{types.Time(), types.Time(), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), time.Date(1970, 1, 1, 9, 1, 57, 493614090, time.UTC), true, nil, nil},

		// year.
		{types.Year(), types.Year(), 2023, 2023, true, nil, nil},
		{types.Int(16), types.Year(), 1, 1, true, nil, nil},
		{types.Int(64).Unsigned(), types.Year(), uint(9999), 9999, true, nil, nil},
		{types.String(), types.Year(), "2023", 2023, true, nil, nil},
		{types.String(), types.Year(), "1", 1, true, nil, nil},
		{types.JSON(), types.Year(), json.Value("1.0"), 1, true, nil, nil},
		{types.JSON(), types.Year(), json.Value("2023.0"), 2023, true, nil, nil},
		{types.JSON(), types.Year(), json.Value("2023"), 2023, true, nil, nil},

		// uuid.
		{types.UUID(), types.UUID(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000", true, nil, nil},
		{types.String(), types.UUID(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000", true, nil, nil},
		{types.JSON(), types.UUID(), json.Value(`"123e4567-e89b-12d3-a456-426614174000"`), "123e4567-e89b-12d3-a456-426614174000", true, nil, nil},

		// json.
		{types.Int(32), types.JSON(), nil, nil, true, nil, nil},
		{types.Int(32), types.JSON(), nil, json.Value(`null`), false, nil, nil},
		{types.JSON(), types.JSON(), json.Value(`{"foo":5}`), json.Value(`{"foo":5}`), true, nil, nil},
		{types.JSON(), types.JSON(), json.Value("null"), json.Value(`null`), true, nil, nil},
		{types.String(), types.JSON(), "", json.Value(`""`), false, nil, nil},
		{types.JSON(), types.JSON(), json.Value("true"), json.Value("true"), true, nil, nil},
		{types.JSON(), types.JSON(), json.Value(`"foo"`), json.Value(`"foo"`), true, nil, nil},
		{types.JSON(), types.JSON(), json.Value("3.14"), json.Value("3.14"), true, nil, nil},
		{types.JSON(), types.JSON(), json.Value("7204812694472.9355460893"), json.Value("7204812694472.9355460893"), true, nil, nil},
		{types.JSON(), types.JSON(), json.Value(`{"foo":"boo"}`), json.Value(`{"foo":"boo"}`), true, nil, nil},
		{types.JSON(), types.JSON(), json.Value(`[1,2,3]`), json.Value(`[1,2,3]`), true, nil, nil},

		// ip.
		{types.IP(), types.IP(), "2001:db8::ff00:42:8329", "2001:db8::ff00:42:8329", true, nil, nil},
		{types.String(), types.IP(), "2001:0db8:0000:0000:0000:ff00:0042:8329", "2001:db8::ff00:42:8329", true, nil, nil},
		{types.JSON(), types.IP(), json.Value(`"2001:0db8:0000:0000:0000:ff00:0042:8329"`), "2001:db8::ff00:42:8329", true, nil, nil},

		// array.
		{types.Array(types.Int(32)), types.Array(types.Int(32)), []any{1, 2, 3}, []any{1, 2, 3}, true, nil, nil},
		{types.Array(types.Int(32)), types.Array(types.Int(8)), []any{1, 2, 3}, []any{1, 2, 3}, true, nil, nil},
		{types.Array(types.String()), types.Array(types.String()), []any{"123e4567-e89b-12d3-a456-426614174000"}, []any{"123e4567-e89b-12d3-a456-426614174000"}, true, nil, nil},
		{types.JSON(), types.Array(types.Int(32)), json.Value("[1.0,2.0,3.0]"), []any{1, 2, 3}, true, nil, nil},
		{types.JSON(), types.Array(types.Int(32)), json.Value("[1,2,3]"), []any{1, 2, 3}, true, nil, nil},
		{types.JSON(), types.Array(types.Int(32)), json.Value("6.0"), []any{6}, true, nil, nil},
		{types.JSON(), types.Array(types.Boolean()), json.Value("true"), []any{true}, true, nil, nil},
		{types.JSON(), types.Array(types.String()), json.Value(`"foo"`), []any{"foo"}, true, nil, nil},
		{types.JSON(), types.Array(types.Float(64)), json.Value(`15.07`), []any{15.07}, true, nil, nil},
		{types.String(), types.Array(types.Float(64)), "foo", nil, false, nil, errInvalidConversion},

		// object.
		{
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}, {Name: "boo", Type: types.String(), Nullable: true}}),
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}, {Name: "boo", Type: types.String(), Nullable: true}}),
			map[string]any{"foo": 5, "boo": nil},
			map[string]any{"foo": 5, "boo": nil},
			true,
			nil,
			nil,
		},
		{
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}, {Name: "boo", Type: types.String()}}),
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32), CreateRequired: true}, {Name: "boo", Type: types.String()}}),
			map[string]any{"foo": 5},
			map[string]any{"foo": 5},
			true,
			nil,
			nil,
		},
		{
			types.JSON(),
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}, {Name: "boo", Type: types.String(), Nullable: true}}),
			json.Value(`{"foo":5.0,"boo":null}`),
			map[string]any{"foo": 5, "boo": nil},
			true,
			nil,
			nil,
		},
		{
			types.JSON(),
			types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}, {Name: "boo", Type: types.String(), Nullable: true}}),
			json.Value(`{"foo":5,"boo":null}`),
			map[string]any{"foo": 5, "boo": nil},
			true,
			nil,
			nil,
		},
		{types.JSON(), types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}}), json.Value(`{"@":7,"foo":8}`), map[string]any{"foo": 8}, true, nil, nil},
		{types.Map(types.Boolean()), types.Object([]types.Property{{Name: "x", Type: types.String()}, {Name: "y", Type: types.Boolean()}}), map[string]any{"x": true, "y": false}, map[string]any{"x": "true", "y": false}, true, nil, nil},
		{types.Int(32), types.Object([]types.Property{{Name: "x", Type: types.String()}}), 56, nil, false, nil, errInvalidConversion},

		// map.
		{types.Map(types.Boolean()), types.Map(types.Boolean()), map[string]any{"a": true, "b": false}, map[string]any{"a": true, "b": false}, true, nil, nil},
		{types.Map(types.Int(16)), types.Map(types.Float(32)), map[string]any{"a": 4032, "b": -721}, map[string]any{"a": float64(float32(4032)), "b": float64(float32(-721))}, true, nil, nil},
		{types.JSON(), types.Map(types.Float(32)), json.Value(`{"a":4032,"b":-721}`), map[string]any{"a": float64(float32(4032)), "b": float64(float32(-721))}, true, nil, nil},
		{types.JSON(), types.Map(types.Float(32)), json.Value(`{"a":4032,"b":-721}`), map[string]any{"a": float64(float32(4032)), "b": float64(float32(-721))}, true, nil, nil},
		{types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}}), types.Map(types.String()), map[string]any{"foo": 572}, map[string]any{"foo": "572"}, true, nil, nil},
		{types.String(), types.Map(types.String()), "boo", nil, false, nil, errInvalidConversion},
	}

	for _, test := range tests {
		got, err := convert(test.value, test.t1, test.t2, test.nullable, false, test.layouts, Create)
		if err != nil {
			if test.err != nil {
				if test.err != err {
					t.Fatalf("converting %s<%v> to type %s, expected error %q, got %q", test.t1, test.value, test.t2, test.err, err)
				}
				continue
			}
			t.Fatalf("converting %s<%v> to type %s, extected no error, got error %q", test.t1, test.value, test.t2, err)
		}
		if test.err != nil {
			t.Fatalf("converting %s<%v> to type %s, expected error %q, got no error", test.t1, test.value, test.t2, test.err)
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

// TestConvertStringToDate checks parsing of various date formats.
func TestConvertStringToDate(t *testing.T) {
	tests := []struct {
		in  string
		t   time.Time
		err error
	}{
		{"2023-05-24", time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), nil},
		{"05/24/2023", time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), nil},
		{"05.24.2023", time.Date(2023, 5, 24, 0, 0, 0, 0, time.UTC), nil},
		{"44927", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), nil},
		{"44927.50", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), nil},
		{"44927.5000", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), nil},
		{"59.99999999999999999999", time.Date(1900, 2, 28, 0, 0, 0, 0, time.UTC), nil},
		{"61.0", time.Date(1900, 3, 1, 0, 0, 0, 0, time.UTC), nil},
		{"60.0", time.Time{}, errParseConversion},
		{"60.5", time.Time{}, errParseConversion},
		{"60.99999999999999999999", time.Time{}, errParseConversion},
		{"2958465.99", time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC), nil},
		{"2958466.00", time.Time{}, errYearRangeConversion},
		{"999999999999999999999", time.Time{}, errYearRangeConversion},
		{"2023-13-01", time.Time{}, errParseConversion},
		{"2023/05-24", time.Time{}, errParseConversion},
		{"23/05-24", time.Time{}, errParseConversion},
		{"0000-01-01", time.Time{}, errYearRangeConversion},
		{"2000-02-30", time.Time{}, errParseConversion},
		{"abc", time.Time{}, errParseConversion},
	}
	for _, tt := range tests {
		got, err := convertStringToDate(tt.in)
		if tt.err != err {
			t.Fatalf("%s: expected error %v, got %v", tt.in, tt.err, err)
		}
		if !tt.t.Equal(got) {
			t.Fatalf("%s: expected %v, got %v", tt.in, tt.t, got)
		}
	}
}

// TestIsSimpleFloat validates detection of simple floating point strings.
func TestIsSimpleFloat(t *testing.T) {
	tests := []struct {
		s  string
		ok bool
	}{
		{"1.2", true},
		{"123", true},
		{"12", false},
		{"1.2.3", false},
		{".5", false},
		{"5.", false},
		{"1a2", false},
		{"12.34", true},
	}
	for _, tt := range tests {
		if got := isSimpleFloat(tt.s); got != tt.ok {
			t.Fatalf("%s: expected %t, got %t", tt.s, tt.ok, got)
		}
	}
}

// TestParseUint verifies integer parsing with edge cases.
func TestParseUint(t *testing.T) {
	tests := []struct {
		in string
		n  int
	}{
		{"0", 0},
		{"0010", 10},
		{"123", 123},
		{"9223372036854775807", 9223372036854775807},
		{"9223372036854775808", -1},
		{"1a2", -1},
	}
	for _, tt := range tests {
		if got := parseUint(tt.in); got != tt.n {
			t.Fatalf("%s: expected %d, got %d", tt.in, tt.n, got)
		}
	}
}

// TestRejectNonTemporalTimeLiterals checks that unsupported literals are
// rejected like properties.
func TestRejectNonTemporalTimeLiterals(t *testing.T) {

	schema := types.Object([]types.Property{{Name: "value", Type: types.Boolean()}})

	for _, source := range []string{"true", "false", "42", "-42", "1.5", "value"} {
		t.Run(source, func(t *testing.T) {
			_, _, err := Compile(source, schema, types.Time())
			if err != nil {
				return
			}
			t.Fatal("expected compilation to reject conversion to time")
		})
	}

}

// TestArrayUnique checks uniqueness after conversion for constructed, native,
// and JSON arrays.
func TestArrayUnique(t *testing.T) {

	tests := []struct {
		name       string
		expression string
		source     types.Type
		element    types.Type
		value      any
		wantErr    bool
	}{
		{name: "empty", expression: "array()", element: types.Int(32)},
		{name: "singleton", expression: "array(1)", element: types.Int(32)},
		{name: "distinct", expression: "array(1, 2)", element: types.Int(32)},
		{name: "adjacent duplicates", expression: "array(1, 1)", element: types.Int(32), wantErr: true},
		{name: "separated duplicates", expression: "array(1, 2, 1)", element: types.Int(32), wantErr: true},
		{
			name: "native distinct", expression: "value", source: types.Array(types.Int(32)),
			element: types.Int(32), value: []any{1, 2},
		},
		{
			name: "native duplicates", expression: "value", source: types.Array(types.Int(32)),
			element: types.Int(32), value: []any{1, 2, 1}, wantErr: true,
		},
		{
			name: "JSON distinct", expression: "value", source: types.JSON(),
			element: types.Int(32), value: json.Value("[1,2]"),
		},
		{
			name: "JSON duplicates", expression: "value", source: types.JSON(),
			element: types.Int(32), value: json.Value("[1,2,1]"), wantErr: true,
		},
		{
			name: "distinct decimals", expression: "array(1.2, 1.3)", element: types.Decimal(4, 2),
		},
		{
			name: "equal decimals with different representations", expression: "value",
			source: types.Array(types.Decimal(4, 2)), element: types.Decimal(4, 2),
			value: []any{decimal.MustParse("1.20"), decimal.MustParse("1.2")}, wantErr: true,
		},
		{
			name: "JSON distinct decimals", expression: "value", source: types.JSON(),
			element: types.Decimal(4, 2), value: json.Value("[1.2,1.3]"),
		},
		{
			name: "JSON equal decimals", expression: "value", source: types.JSON(),
			element: types.Decimal(4, 2), value: json.Value("[1.20,1.2]"), wantErr: true,
		},
		{
			name: "unique source stays distinct", expression: "value",
			source: types.Array(types.Float(64)).WithUnique(), element: types.Int(32), value: []any{1.2, 2.3},
		},
		{
			name: "unique source gains duplicates", expression: "value",
			source: types.Array(types.Float(64)).WithUnique(), element: types.Int(32), value: []any{1.2, 1.4},
			wantErr: true,
		},
	}
	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			var inSchema types.Type
			if test.source.Valid() {
				inSchema = types.Object([]types.Property{{Name: "value", Type: test.source}})
			}
			outSchema := types.Object([]types.Property{{Name: "out", Type: types.Array(test.element).WithUnique()}})
			mapping, err := New(map[string]string{"out": test.expression}, inSchema, outSchema, false, nil)
			if err != nil {
				t.Fatal(err)
			}
			got, err := mapping.Transform(map[string]any{"value": test.value}, None)
			if err != nil {
				if !test.wantErr {
					t.Fatal(err)
				}
				if _, ok := errors.AsType[ValidationError](err); !ok {
					t.Fatalf("got %T (%v), want ValidationError", err, err)
				}
				return
			}
			if test.wantErr {
				t.Fatalf("got %#v, want a duplicate element error", got)
			}

		})

	}

}

// BenchmarkArrayUnique measures validation of distinct integer arrays of
// increasing size.
func BenchmarkArrayUnique(b *testing.B) {
	for _, size := range []int{99, 100, 1000, 10000} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {

			values := make([]any, size)
			for i := range values {
				values[i] = i
			}
			array := types.Array(types.Int(32))
			schema := types.Object([]types.Property{{Name: "value", Type: array}})
			outSchema := types.Object([]types.Property{{Name: "out", Type: array.WithUnique()}})
			mapping, err := New(map[string]string{"out": "value"}, schema, outSchema, false, nil)
			if err != nil {
				b.Fatal(err)
			}
			attributes := map[string]any{"value": values}
			b.ReportAllocs()

			for b.Loop() {
				_, err := mapping.Transform(attributes, None)
				if err != nil {
					b.Fatal(err)
				}
			}

		})
	}
}

// TestArrayUniqueTimeLayouts checks duplicates introduced by formatting native
// and JSON temporal values.
func TestArrayUniqueTimeLayouts(t *testing.T) {

	instant := time.Date(2026, 9, 8, 10, 0, 0, 125000000, time.UTC)
	date := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	clock := time.Date(1970, 1, 1, 10, 0, 0, 125000000, time.UTC)
	tests := []struct {
		name      string
		element   types.Type
		values    []any
		encoded   json.Value
		layouts   state.TimeLayouts
		want      []any
		duplicate bool
	}{
		{
			"datetime unix collision", types.DateTime(), []any{instant, instant.Add(time.Nanosecond)},
			json.Value(`["2026-09-08T10:00:00.125Z","2026-09-08T10:00:00.125000001Z"]`),
			state.TimeLayouts{DateTime: "unix"}, nil, true,
		},
		{
			"datetime text collision", types.DateTime(), []any{instant, instant.Add(time.Nanosecond)},
			json.Value(`["2026-09-08T10:00:00.125Z","2026-09-08T10:00:00.125000001Z"]`),
			state.TimeLayouts{DateTime: time.RFC3339}, nil, true,
		},
		{
			"date text collision", types.Date(), []any{date, date.AddDate(0, 0, 1)},
			json.Value(`["2026-09-08","2026-09-09"]`), state.TimeLayouts{Date: "2006-01"}, nil, true,
		},
		{
			"time text collision", types.Time(), []any{clock, clock.Add(time.Nanosecond)},
			json.Value(`["10:00:00.125","10:00:00.125000001"]`), state.TimeLayouts{Time: time.TimeOnly}, nil, true,
		},
		{
			"datetime unix distinct", types.DateTime(), []any{instant, instant.Add(time.Second)},
			json.Value(`["2026-09-08T10:00:00.125Z","2026-09-08T10:00:01.125Z"]`),
			state.TimeLayouts{DateTime: "unix"}, []any{instant.Unix(), instant.Unix() + 1}, false,
		},
		{
			"time text distinct", types.Time(), []any{clock, clock.Add(time.Second)},
			json.Value(`["10:00:00.125","10:00:01.125"]`), state.TimeLayouts{Time: time.TimeOnly},
			[]any{"10:00:00", "10:00:01"}, false,
		},
	}

	for _, test := range tests {
		for _, fromJSON := range []bool{false, true} {
			for _, purpose := range []Purpose{None, Create, Update} {
				for _, inPlace := range []bool{false, true} {
					name := fmt.Sprintf("%s/JSON=%t/purpose=%d/inPlace=%t", test.name, fromJSON, purpose, inPlace)
					t.Run(name, func(t *testing.T) {

						array := types.Array(test.element).WithUnique()
						source := array
						var value any = test.values
						if fromJSON {
							source, value = types.JSON(), test.encoded
						}
						inSchema := types.Object([]types.Property{{Name: "value", Type: source}})
						outSchema := types.Object([]types.Property{{Name: "out", Type: array}})
						mapping, err := New(map[string]string{"out": "value"}, inSchema, outSchema, inPlace, &test.layouts)
						if err != nil {
							t.Fatal(err)
						}

						got, err := mapping.Transform(map[string]any{"value": value}, purpose)
						if err != nil {
							if !test.duplicate {
								t.Fatal(err)
							}
							if _, ok := errors.AsType[ValidationError](err); !ok {
								t.Fatalf("got %T (%v), want ValidationError", err, err)
							}
							return
						}
						if test.duplicate {
							t.Fatalf("got %#v, want a duplicate element error", got)
						}
						if !reflect.DeepEqual(got["out"], test.want) {
							t.Fatalf("got %#v, want %#v", got["out"], test.want)
						}

					})
				}
			}
		}
	}

}

// TestArrayUniqueValues checks semantic duplicates without changing the
// representation of accepted values.
func TestArrayUniqueValues(t *testing.T) {

	instant := time.Date(2026, 9, 8, 10, 0, 0, 125000000, time.UTC)
	date := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	clock := time.Date(1970, 1, 1, 10, 0, 0, 125000000, time.UTC)
	utc := time.FixedZone("UTC", 0)
	offset := time.FixedZone("UTC+02", 2*60*60)
	now := time.Now()
	tests := []struct {
		name      string
		element   types.Type
		values    []any
		duplicate bool
	}{
		{"datetime location", types.DateTime(), []any{instant, instant.In(utc)}, true},
		{"datetime offset", types.DateTime(), []any{instant, instant.In(offset)}, true},
		{"datetime monotonic", types.DateTime(), []any{now, now.Round(0)}, true},
		{"datetime distinct", types.DateTime(), []any{instant, instant.In(utc).Add(time.Nanosecond)}, false},
		{"date location", types.Date(), []any{date, date.In(utc)}, true},
		{"date components", types.Date(), []any{date, time.Date(2026, 9, 8, 0, 0, 0, 0, offset)}, true},
		{"date distinct", types.Date(), []any{date, date.AddDate(0, 0, 1)}, false},
		{"time location", types.Time(), []any{clock, clock.In(utc)}, true},
		{"time reference date", types.Time(), []any{clock, instant}, true},
		{"time distinct", types.Time(), []any{clock, clock.In(utc).Add(time.Nanosecond)}, false},
		{
			"decimal representation", types.Decimal(6, 2),
			[]any{decimal.New(15, 1), decimal.MustParse("15e-1")}, true,
		},
		{
			"negative decimal representation", types.Decimal(6, 2),
			[]any{decimal.New(-15, 1), decimal.MustParse("-1.50")}, true,
		},
		{"decimal zero", types.Decimal(6, 2), []any{decimal.Decimal{}, decimal.MustParse("-0.00")}, true},
		{
			"decimal opposite signs", types.Decimal(6, 2),
			[]any{decimal.MustParse("1.28"), decimal.MustParse("-1.28")}, false,
		},
		{
			"large distinct decimals", types.Decimal(20, 0),
			[]any{decimal.MustParse("9007199254740992"), decimal.MustParse("9007199254740993")}, false,
		},
		{"float signed zero", types.Float(64), []any{0.0, math.Copysign(0, -1)}, true},
		{"float infinity", types.Float(64), []any{math.Inf(1), math.Inf(1)}, true},
		{"float opposite infinities", types.Float(64), []any{math.Inf(-1), math.Inf(1)}, false},
		{"float NaN", types.Float(64), []any{math.NaN(), math.NaN()}, true},
		{
			"float NaN representations", types.Float(64),
			[]any{math.Float64frombits(0x7ff8000000000001), math.Float64frombits(0xfff0000000000001)}, true,
		},
		{"float32 NaN", types.Float(32), []any{math.NaN(), math.NaN()}, true},
		{"single NaN", types.Float(64), []any{math.NaN(), math.Inf(1)}, false},
		{"float distinct", types.Float(64), []any{1.0, math.Nextafter(1, 2)}, false},
		{"string case", types.String(), []any{"Foo", "foo"}, false},
	}

	for _, test := range tests {
		for _, size := range []int{2, 99, 100, 101} {

			// Separate the pair with distinct values so duplicates are encountered at the end.
			input := make([]any, size)
			input[0], input[size-1] = test.values[0], test.values[1]
			for i := 1; i < size-1; i++ {
				switch test.element.Kind() {
				case types.DateTimeKind:
					input[i] = time.Date(2000, 1, 1, 0, 0, i, 0, time.UTC)
				case types.DateKind:
					input[i] = time.Date(2000, 1, i, 0, 0, 0, 0, time.UTC)
				case types.TimeKind:
					input[i] = time.Date(1970, 1, 1, 0, 0, i, 0, time.UTC)
				case types.DecimalKind:
					input[i] = decimal.MustParse(fmt.Sprint(1000 + i))
				case types.FloatKind:
					input[i] = float64(1000 + i)
				case types.StringKind:
					input[i] = fmt.Sprintf("padding-%d", i)
				}
			}
			expressions := []string{"value"}
			if size == 2 {
				expressions = append(expressions, "array(a, b)")
			}

			for _, expression := range expressions {
				for _, purpose := range []Purpose{None, Create, Update} {
					for _, inPlace := range []bool{false, true} {
						name := fmt.Sprintf("%s/size=%d/%s/purpose=%d/inPlace=%t", test.name, size, expression, purpose, inPlace)
						t.Run(name, func(t *testing.T) {

							array := types.Array(test.element)
							inSchema := types.Object([]types.Property{
								{Name: "value", Type: array}, {Name: "a", Type: test.element}, {Name: "b", Type: test.element},
							})
							outSchema := types.Object([]types.Property{{Name: "out", Type: array.WithUnique()}})
							mapping, err := New(map[string]string{"out": expression}, inSchema, outSchema, inPlace, nil)
							if err != nil {
								t.Fatal(err)
							}
							attributes := map[string]any{"value": input, "a": test.values[0], "b": test.values[1]}

							got, err := mapping.Transform(attributes, purpose)
							if err != nil {
								if !test.duplicate {
									t.Fatal(err)
								}
								if _, ok := errors.AsType[ValidationError](err); !ok {
									t.Fatalf("got %T (%v), want ValidationError", err, err)
								}
								return
							}
							if test.duplicate {
								t.Fatalf("got %#v, want a duplicate element error", got)
							}
							values := got["out"].([]any)
							if len(values) != len(input) {
								t.Fatalf("got %d values, want %d", len(values), len(input))
							}
							for i, value := range values {
								if f, ok := value.(float64); ok && math.IsNaN(f) {
									if expected, ok := input[i].(float64); ok && math.IsNaN(expected) {
										continue
									}
								}
								if !reflect.DeepEqual(value, input[i]) {
									t.Fatalf("element %d: got %#v, want %#v", i, value, input[i])
								}
							}

						})
					}
				}
			}

		}
	}

}

// TestArrayWithoutUnique checks that mappings preserve duplicates unless the
// output schema forbids them.
func TestArrayWithoutUnique(t *testing.T) {
	for _, expression := range []string{"value", "array(a, a)"} {
		for _, size := range []int{2, 100} {
			t.Run(fmt.Sprintf("%s/%d", expression, size), func(t *testing.T) {

				values := make([]any, size)
				for i := range values {
					values[i] = math.NaN()
				}
				array := types.Array(types.Float(64))
				schema := types.Object([]types.Property{
					{Name: "value", Type: array}, {Name: "a", Type: types.Float(64)},
				})
				outSchema := types.Object([]types.Property{{Name: "out", Type: array}})
				mapping, err := New(map[string]string{"out": expression}, schema, outSchema, false, nil)
				if err != nil {
					t.Fatal(err)
				}

				got, err := mapping.Transform(map[string]any{"value": values, "a": math.NaN()}, None)
				if err != nil {
					t.Fatal(err)
				}
				wantLength := size
				if expression != "value" {
					wantLength = 2
				}
				out := got["out"].([]any)
				if len(out) != wantLength {
					t.Fatalf("got %d elements, want %d", len(out), wantLength)
				}
				for i, value := range out {
					if !math.IsNaN(value.(float64)) {
						t.Fatalf("element %d: got %v, want NaN", i, value)
					}
				}

			})
		}
	}
}

// TestEnumEmptyStringMapping preserves allowed empty strings in optional,
// nullable, and required output properties.
func TestEnumEmptyStringMapping(t *testing.T) {

	enum := types.String().WithValues("", "yes")
	outSchema := types.Object([]types.Property{
		{Name: "optional", Type: enum},
		{Name: "nullable", Type: enum, Nullable: true},
		{Name: "required", Type: enum, CreateRequired: true, UpdateRequired: true},
		{Name: "nullableRequired", Type: enum, Nullable: true, CreateRequired: true, UpdateRequired: true},
	})
	tests := []struct {
		name, source string
		typ          types.Type
		value        any
	}{
		{"literal", "''", types.String(), ""},
		{"string", "value", types.String(), ""},
		{"enum", "value", enum, ""},
		{"JSON", "value", types.JSON(), json.Value(`""`)},
		{"selector", "coalesce(value)", enum, ""},
	}

	for _, test := range tests {
		for _, purpose := range []Purpose{None, Create, Update} {
			t.Run(fmt.Sprintf("%s/purpose=%d", test.name, purpose), func(t *testing.T) {

				inSchema := types.Object([]types.Property{{Name: "value", Type: test.typ}})
				expressions := map[string]string{
					"optional": test.source, "nullable": test.source,
					"required": test.source, "nullableRequired": test.source,
				}
				mapping, err := New(expressions, inSchema, outSchema, false, nil)
				if err != nil {
					t.Fatal(err)
				}

				got, err := mapping.Transform(map[string]any{"value": test.value}, purpose)
				if err != nil {
					t.Fatal(err)
				}

				for name := range expressions {
					value, present := got[name]
					if !present || value != "" {
						t.Fatalf("got %#v, want %s to contain an empty string", got, name)
					}
				}

			})
		}
	}

}

// TestJSONTemporalValues preserves temporal types when values pass through JSON
// or map.
func TestJSONTemporalValues(t *testing.T) {

	tests := []struct {
		name  string
		typ   types.Type
		input string
	}{
		{"date", types.Date(), `"2026-09-08"`},
		{"minimum date", types.Date(), `"0001-01-01"`},
		{"maximum date", types.Date(), `"9999-12-31"`},
		{"time", types.Time(), `"12:34:56"`},
		{"fractional time", types.Time(), `"12:34:56.123456789"`},
		{"datetime", types.DateTime(), `"2026-09-08T12:34:56.123456789Z"`},
	}
	options := []struct {
		asJSON, sorted, inPlace bool
		purpose                 Purpose
	}{
		{false, false, false, None},
		{false, true, true, Create},
		{false, false, true, Update},
		{true, false, false, None},
		{true, true, true, Create},
		{true, false, true, Update},
	}

	for _, test := range tests {

		for _, shape := range []string{"scalar", "array", "object", "map", "nested"} {

			st, input := test.typ, test.input
			switch shape {
			case "array":
				st, input = types.Array(st), "["+input+"]"
			case "object":
				st = types.Object([]types.Property{
					{Name: "at", Type: st},
					{Name: "null", Type: st, Nullable: true},
					{Name: "missing", Type: st, ReadOptional: true},
				})
				input = `{"at":` + input + `,"null":null}`
			case "map":
				st, input = types.Map(st), `{"at":`+input+`}`
			case "nested":
				st = types.Array(types.Map(types.Object([]types.Property{{Name: "at", Type: st}})))
				input = `[{"x":{"at":` + input + `}}]`
			}
			inSchema := types.Object([]types.Property{{Name: "value", Type: st}})
			expressions := []string{
				"value", "array(value)", "map('x', value)",
				"map('x', if(true, value, value))", "map('x', coalesce(value, value))",
				"map('x', array(value))", "map('x', map('y', value))",
			}

			for _, expression := range expressions {

				outType, expected := st, input
				switch expression {
				case "array(value)":
					outType, expected = types.Array(st), "["+input+"]"
				case "map('x', array(value))":
					outType, expected = types.Map(types.Array(st)), `{"x":[`+input+`]}`
				case "map('x', map('y', value))":
					outType, expected = types.Map(types.Map(st)), `{"x":{"y":`+input+`}}`
				case "value":
				default:
					outType, expected = types.Map(st), `{"x":`+input+`}`
				}

				for _, option := range options {

					name := fmt.Sprintf("%s/%s/%s/json=%t/sorted=%t/inPlace=%t/purpose=%d",
						test.name, shape, expression, option.asJSON, option.sorted, option.inPlace, option.purpose)
					t.Run(name, func(t *testing.T) {

						previous := encodeSorted
						encodeSorted = option.sorted
						t.Cleanup(func() { encodeSorted = previous })
						attributes, err := types.Decode[map[string]any](strings.NewReader(`{"value":`+input+`}`), inSchema)
						if err != nil {
							t.Fatal(err)
						}
						want, err := types.Decode[any](strings.NewReader(expected), outType)
						if err != nil {
							t.Fatal(err)
						}

						dt := outType
						var layouts *state.TimeLayouts
						if option.asJSON {
							dt = types.JSON()
							layouts = &state.TimeLayouts{DateTime: "unixnano", Date: "02/01/2006", Time: "15:04"}
						}
						outSchema := types.Object([]types.Property{{Name: "out", Type: dt}})
						mapping, err := New(map[string]string{"out": expression}, inSchema, outSchema, option.inPlace, layouts)
						if err != nil {
							t.Fatal(err)
						}
						out, err := mapping.Transform(attributes, option.purpose)
						if err != nil {
							t.Fatal(err)
						}

						got := out["out"]
						if option.asJSON {
							got, err = types.Decode[any](bytes.NewReader(got.(json.Value)), outType)
							if err != nil {
								t.Fatalf("invalid temporal JSON %s: %v", out["out"], err)
							}
						}
						if !reflect.DeepEqual(got, want) {
							t.Fatalf("got %#v, want %#v", got, want)
						}
						unchanged, err := types.Decode[map[string]any](strings.NewReader(`{"value":`+input+`}`), inSchema)
						if err != nil {
							t.Fatal(err)
						}
						if !reflect.DeepEqual(attributes, unchanged) {
							t.Fatalf("source was modified: %#v", attributes)
						}

					})

				}

			}

		}

	}

}

// TestJSONValueRepresentations checks temporal text and preserves JSON
// representations of other types.
func TestJSONValueRepresentations(t *testing.T) {

	day := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	clock := time.Date(1970, 1, 1, 12, 34, 56, 125000000, time.UTC)
	tests := []struct {
		name  string
		typ   types.Type
		value any
		want  string
	}{
		{"date", types.Date(), day, `"2026-09-08"`},
		{"time", types.Time(), clock, `"12:34:56.125"`},
		{"datetime", types.DateTime(), clock, `"1970-01-01T12:34:56.125Z"`},
		{"int64", types.Int(64), int(math.MaxInt64), "9223372036854775807"},
		{"uint64", types.Int(64).Unsigned(), uint(math.MaxUint64), "18446744073709551615"},
		{"decimal", types.Decimal(20, 3), decimal.MustParse("9007199254740993.125"), "9007199254740993.125"},
		{"float32", types.Float(32), float64(float32(0.1)), "0.10000000149011612"},
		{"json", types.JSON(), json.Value(`{"z":1,"a":"1970-01-01T12:34:56Z"}`), `{"z":1,"a":"1970-01-01T12:34:56Z"}`},
		{"empty array", types.Array(types.Date()), []any{}, "[]"},
		{"empty map", types.Map(types.Time()), map[string]any{}, "{}"},
		{
			"empty object", types.Object([]types.Property{{Name: "at", Type: types.Date(), ReadOptional: true}}),
			map[string]any{}, "{}",
		},
	}

	for _, test := range tests {

		for _, sorted := range []bool{false, true} {

			for _, expression := range []string{"value", "map('x', value)"} {

				t.Run(fmt.Sprintf("%s/sorted=%t/%s", test.name, sorted, expression), func(t *testing.T) {

					previous := encodeSorted
					encodeSorted = sorted
					t.Cleanup(func() { encodeSorted = previous })
					inSchema := types.Object([]types.Property{{Name: "value", Type: test.typ}})
					outSchema := types.Object([]types.Property{{Name: "out", Type: types.JSON()}})
					mapping, err := New(map[string]string{"out": expression}, inSchema, outSchema, false, nil)
					if err != nil {
						t.Fatal(err)
					}
					out, err := mapping.Transform(map[string]any{"value": test.value}, None)
					if err != nil {
						t.Fatal(err)
					}

					want := test.want
					if expression != "value" {
						want = `{"x":` + want + `}`
					}
					if got := string(out["out"].(json.Value)); got != want {
						t.Fatalf("got %s, want %s", got, want)
					}

				})

			}

		}

	}

}

// TestInPlaceConversions checks shared containers and schema projections with
// one or several outputs.
func TestInPlaceConversions(t *testing.T) {

	stringObject := types.Object([]types.Property{{Name: "x", Type: types.String()}})
	intObject := types.Object([]types.Property{{Name: "x", Type: types.Int(32)}})
	projection := types.Object([]types.Property{{Name: "keep", Type: types.String()}})
	tests := []struct {
		name           string
		source, target types.Type
		value          func() any
		want           any
	}{
		{
			"object", stringObject, intObject,
			func() any { return map[string]any{"x": "42"} }, map[string]any{"x": 42},
		},
		{
			"map", types.Map(types.String()), types.Map(types.Int(32)),
			func() any { return map[string]any{"x": "42"} }, map[string]any{"x": 42},
		},
		{
			"array", types.Array(types.String()), types.Array(types.Int(32)),
			func() any { return []any{"42"} }, []any{42},
		},
		{
			"shared array elements", types.Array(stringObject), types.Array(intObject),
			func() any {
				value := map[string]any{"x": "42"}
				return []any{value, value}
			},
			[]any{map[string]any{"x": 42}, map[string]any{"x": 42}},
		},
		{
			"object projection",
			types.Object([]types.Property{{Name: "keep", Type: types.String()}, {Name: "extra", Type: types.String()}}),
			projection, func() any { return map[string]any{"keep": "x", "extra": "y"} }, map[string]any{"keep": "x"},
		},
		{
			"map projection", types.Map(types.String()), projection,
			func() any { return map[string]any{"keep": "x", "extra": "y"} }, map[string]any{"keep": "x"},
		},
	}
	for _, test := range tests {

		for _, purpose := range []Purpose{None, Create, Update} {

			for _, inPlace := range []bool{false, true} {

				for _, convertedPath := range []string{"a", "b", "out"} {

					t.Run(fmt.Sprintf("%s/%s/purpose=%v/inPlace=%v", test.name, convertedPath, purpose, inPlace),
						func(t *testing.T) {

							expressions := map[string]string{convertedPath: "value"}
							properties := []types.Property{{Name: convertedPath, Type: test.target}}
							want := map[string]any{convertedPath: test.want}
							if convertedPath != "out" {
								originalPath := "a"
								if convertedPath == "a" {
									originalPath = "b"
								}
								expressions[originalPath] = "value"
								properties = append(properties, types.Property{Name: originalPath, Type: test.source})
								want[originalPath] = test.value()
							}
							inSchema := types.Object([]types.Property{{Name: "value", Type: test.source}})
							mapping, err := New(expressions, inSchema, types.Object(properties), inPlace, nil)
							if err != nil {
								t.Fatal(err)
							}
							got, err := mapping.Transform(map[string]any{"value": test.value()}, purpose)
							if err != nil {
								t.Fatal(err)
							}
							if !reflect.DeepEqual(got, want) {
								t.Fatalf("got %#v, want %#v", got, want)
							}

						})

				}

			}

		}

	}

}

// TestMappingUnixNanoRange checks timestamp bounds through scalar and container
// mappings.
func TestMappingUnixNanoRange(t *testing.T) {

	tests := []struct {
		name    string
		value   time.Time
		want    int64
		invalid bool
	}{
		{"minimum", time.Date(1677, 9, 21, 0, 12, 43, 145224192, time.UTC), math.MinInt64, false},
		{"below minimum", time.Date(1677, 9, 21, 0, 12, 43, 145224191, time.UTC), 0, true},
		{"above minimum", time.Date(1677, 9, 21, 0, 12, 43, 145224193, time.UTC), math.MinInt64 + 1, false},
		{"maximum", time.Date(2262, 4, 11, 23, 47, 16, 854775807, time.UTC), math.MaxInt64, false},
		{"below maximum", time.Date(2262, 4, 11, 23, 47, 16, 854775806, time.UTC), math.MaxInt64 - 1, false},
		{"above maximum", time.Date(2262, 4, 11, 23, 47, 16, 854775808, time.UTC), 0, true},
		{"late 2262", time.Date(2262, 12, 31, 0, 0, 0, 0, time.UTC), 0, true},
		{"year one", time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC), 0, true},
		{"year 9999", time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC), 0, true},
		{"epoch", time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), 0, false},
	}
	for _, test := range tests {

		for _, source := range []string{"datetime", "string", "json", "array", "object", "map"} {

			st, dt := types.DateTime(), types.DateTime()
			var value any = test.value
			var want any = test.want
			switch source {
			case "string":
				st = types.String()
				value = test.value.In(time.FixedZone("west", -3600)).Format(time.RFC3339Nano)
			case "json":
				st = types.JSON()
				value = json.Value(`"` + test.value.Format(time.RFC3339Nano) + `"`)
			case "array":
				st, dt = types.Array(st), types.Array(dt)
				value, want = []any{value}, []any{want}
			case "object":
				st = types.Object([]types.Property{{Name: "at", Type: st}})
				dt = st
				value, want = map[string]any{"at": value}, map[string]any{"at": want}
			case "map":
				st, dt = types.Map(st), types.Map(dt)
				value, want = map[string]any{"at": value}, map[string]any{"at": want}
			}

			for _, expression := range []string{"value", "array(value)"} {

				outType, expected := dt, want
				if expression == "array(value)" {
					outType, expected = types.Array(dt), []any{want}
				}

				for _, purpose := range []Purpose{None, Create, Update} {

					for _, inPlace := range []bool{false, true} {

						name := fmt.Sprintf("%s/%s/%s/purpose=%d/inPlace=%t",
							test.name, source, expression, purpose, inPlace)
						t.Run(name, func(t *testing.T) {

							inSchema := types.Object([]types.Property{{Name: "value", Type: st}})
							outSchema := types.Object([]types.Property{{Name: "out", Type: outType}})
							layouts := &state.TimeLayouts{DateTime: "unixnano"}
							mapping, err := New(map[string]string{"out": expression}, inSchema, outSchema, inPlace, layouts)
							if err != nil {
								t.Fatal(err)
							}

							got, err := mapping.Transform(map[string]any{"value": value}, purpose)
							if err != nil {
								if !test.invalid {
									t.Fatal(err)
								}
								if _, ok := errors.AsType[ValidationError](err); !ok {
									t.Fatalf("got %T (%v), want ValidationError", err, err)
								}
								message := fmt.Sprintf("«%s» is outside the range supported by "+
									"the «unixnano» datetime format while mapping to «out»", expression)
								if err.Error() != message {
									t.Fatalf("got %q, want %q", err, message)
								}
								return
							}
							if test.invalid {
								t.Fatalf("got %#v, want ValidationError for an unrepresentable timestamp", got)
							}
							if !reflect.DeepEqual(got, map[string]any{"out": expected}) {
								t.Fatalf("got %#v, want out=%#v", got, expected)
							}

						})

					}

				}

			}

		}

	}

}

// TestUnixLayoutsDateRange checks that coarser units retain the full datetime
// range.
func TestUnixLayoutsDateRange(t *testing.T) {

	values := []time.Time{
		time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC),
	}
	for _, value := range values {

		for _, layout := range []string{"unix", "unixmilli", "unixmicro"} {

			t.Run(value.Format(time.RFC3339)+"/"+layout, func(t *testing.T) {

				got, err := convert(value, types.DateTime(), types.DateTime(), false, false,
					&state.TimeLayouts{DateTime: layout}, None)
				if err != nil {
					t.Fatal(err)
				}

				var restored time.Time
				switch layout {
				case "unix":
					restored = time.Unix(got.(int64), 0)
				case "unixmilli":
					restored = time.UnixMilli(got.(int64))
				case "unixmicro":
					restored = time.UnixMicro(got.(int64))
				}
				if !restored.Equal(value) {
					t.Fatalf("got %s, want %s", restored, value)
				}

			})

		}

	}

}
