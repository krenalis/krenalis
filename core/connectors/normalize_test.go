//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package connectors

import (
	"math"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/google/go-cmp/cmp"
)

type customJSONMarshaller []byte

func (m customJSONMarshaller) MarshalJSON() ([]byte, error) {
	return m, nil
}

func Test_normalize(t *testing.T) {

	aDateTime := time.Date(2023, 5, 3, 15, 47, 22, 769802537, time.UTC)
	aDate := time.Date(2023, 5, 3, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		t types.Type
		v any
		e any
		n bool
		l *state.TimeLayouts
	}{
		// boolean.
		{types.Boolean(), true, true, false, nil},
		// int(16).
		{types.Int(16), -6, -6, false, nil},
		{types.Int(16), -6.0, -6, false, nil},
		// int(32).
		{types.Int(32), -9261, -9261, false, nil},
		{types.Int(32), -9261.0, -9261, false, nil},
		// uint(8).
		{types.Uint(8), uint(3), uint(3), false, nil},
		{types.Uint(8), 3.0, uint(3), false, nil},
		// uint(32).
		{types.Uint(32), uint(47303), uint(47303), false, nil},
		{types.Uint(32), 47303.0, uint(47303), false, nil},
		// float(32).
		{types.Float(32), float64(float32(12.79)), float64(float32(12.79)), false, nil},
		{types.Float(32), math.NaN(), math.NaN(), false, nil},
		// float(64).
		{types.Float(64), 12.7902743017496882, 12.7902743017496882, false, nil},
		{types.Float(64), math.NaN(), math.NaN(), false, nil},
		// decimal.
		{types.Decimal(10, 3), "6.639e2", decimal.MustParse("663.9"), false, nil},
		{types.Decimal(8, 0), 793012, decimal.MustInt(793012), false, nil},
		{types.Decimal(5, 0), -14044, decimal.MustInt(-14044), false, nil},
		{types.Decimal(3, 2), "", nil, true, nil},
		// datetime.
		{types.DateTime(), aDateTime, aDateTime, false, nil},
		{types.DateTime(), strconv.FormatInt(aDateTime.Unix(), 10), time.Date(2023, 5, 3, 15, 47, 22, 0, time.UTC), false, &state.TimeLayouts{DateTime: "unix"}},
		{types.DateTime(), strconv.FormatInt(aDateTime.UnixMilli(), 10), time.Date(2023, 5, 3, 15, 47, 22, 769000000, time.UTC), false, &state.TimeLayouts{DateTime: "unixmilli"}},
		{types.DateTime(), strconv.FormatInt(aDateTime.UnixMicro(), 10), time.Date(2023, 5, 3, 15, 47, 22, 769802000, time.UTC), false, &state.TimeLayouts{DateTime: "unixmicro"}},
		{types.DateTime(), strconv.FormatInt(aDateTime.UnixNano(), 10), aDateTime, false, &state.TimeLayouts{DateTime: "unixnano"}},
		{types.DateTime(), "2023-05-03 15:47:22", time.Date(2023, 5, 3, 15, 47, 22, 0, time.UTC), false, &state.TimeLayouts{DateTime: time.DateTime}},
		{types.DateTime(), "2023-05-03", time.Date(2023, 5, 3, 0, 0, 0, 0, time.UTC), false, &state.TimeLayouts{DateTime: time.DateOnly}},
		{types.DateTime(), float64(aDateTime.Unix()), time.Date(2023, 5, 3, 15, 47, 22, 0, time.UTC), false, &state.TimeLayouts{DateTime: "unix"}},
		{types.DateTime(), float64(aDateTime.UnixMilli()), time.Date(2023, 5, 3, 15, 47, 22, 769000000, time.UTC), false, &state.TimeLayouts{DateTime: "unixmilli"}},
		{types.DateTime(), float64(aDateTime.UnixMicro()), time.Date(2023, 5, 3, 15, 47, 22, 769802000, time.UTC), false, &state.TimeLayouts{DateTime: "unixmicro"}},
		{types.DateTime(), float64(aDateTime.UnixNano()), time.Date(2023, 5, 3, 15, 47, 22, 769802496, time.UTC), false, &state.TimeLayouts{DateTime: "unixnano"}},
		{types.DateTime(), "", nil, true, &state.TimeLayouts{DateTime: "unixnano"}},
		// date.
		{types.Date(), aDate, aDate, false, nil},
		{types.Date(), "2023-05-03", aDate, false, &state.TimeLayouts{Date: time.DateOnly}},
		{types.Date(), "", nil, true, &state.TimeLayouts{Date: time.DateOnly}},
		// time.
		{types.Time(), time.Date(2023, 5, 3, 17, 34, 41, 804019312, time.UTC), time.Date(1970, 1, 1, 17, 34, 41, 804019312, time.UTC), false, nil},
		{types.Time(), "00:00:00", time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), false, &state.TimeLayouts{Time: "15:04:05.999999999"}},
		{types.Time(), "13:16:47.801", time.Date(1970, 1, 1, 13, 16, 47, 801000000, time.UTC), false, &state.TimeLayouts{Time: "15:04:05.999999999"}},
		{types.Time(), "23:59:59.999999999", time.Date(1970, 1, 1, 23, 59, 59, 999999999, time.UTC), false, &state.TimeLayouts{Time: "15:04:05.999999999"}},
		{types.Time(), "09:22:51.834", time.Date(1970, 1, 1, 9, 22, 51, 834000000, time.UTC), false, &state.TimeLayouts{Time: "15:04:05.000"}},
		{types.Time(), "09h 31m 13s", time.Date(1970, 1, 1, 9, 31, 13, 0, time.UTC), false, &state.TimeLayouts{Time: "15h 04m 05s"}},
		{types.Time(), "", nil, true, &state.TimeLayouts{Time: "15h 04m 05s"}},
		// year.
		{types.Year(), 2023, 2023, false, nil},
		{types.Year(), 2023.0, 2023, false, nil},
		// uuid.
		{types.UUID(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000", false, nil},
		{types.UUID(), "", nil, true, nil},
		// json.
		{types.JSON(), json.Value(`{"a": 5}`), json.Value(`{"a": 5}`), false, nil},
		{types.JSON(), []byte(`{"a": 5}`), json.Value(`{"a": 5}`), false, nil},
		{types.JSON(), `{"a": 5}`, json.Value(`{"a": 5}`), false, nil},
		{types.JSON(), customJSONMarshaller(`{"a": 5}`), json.Value(`{"a": 5}`), false, nil},
		{types.JSON(), json.Value(" \t503\n"), json.Value(" \t503\n"), false, nil},
		{types.JSON(), json.Value("302"), json.Value("302"), false, nil},
		{types.JSON(), []byte(`{ "a": 5 }`), json.Value(`{ "a": 5 }`), false, nil},
		{types.JSON(), "", nil, true, nil},
		// inet.
		{types.Inet(), "127.0.0.1", "127.0.0.1", false, nil},
		{types.Inet(), "2001:0db8:0000:0000:0000:ff00:0042:8329", "2001:db8::ff00:42:8329", false, nil},
		{types.Inet(), "", nil, true, nil},
		// text.
		{types.Text(), "foo", "foo", false, nil},
		{types.Text().WithValues("foo", "boo"), "boo", "boo", false, nil},
		{types.Text().WithRegexp(regexp.MustCompile(`oo$`)), "foo", "foo", false, nil},
		{types.Text().WithByteLen(3), "boo", "boo", false, nil},
		{types.Text().WithCharLen(3), "bòò", "bòò", false, nil},
		{types.Text().WithValues("foo", "boo"), "", nil, true, nil},
		// array.
		{types.Array(types.Int(32)), []any{1, 2}, []any{1, 2}, false, nil},
		{types.Array(types.Int(32)), []any{1.0, 2.0}, []any{1, 2}, false, nil},
		{types.Array(types.Array(types.Text())), []any{[]any{"foo"}, []any{"foo"}}, []any{[]any{"foo"}, []any{"foo"}}, false, nil},
		// object.
		{types.Object([]types.Property{{Name: "foo", Type: types.Text()}, {Name: "boo", Type: types.Int(32)}}), map[string]any{"foo": "alt", "boo": 3}, map[string]any{"foo": "alt", "boo": 3}, false, nil},
		{types.Object([]types.Property{{Name: "foo", Type: types.Inet(), Nullable: true}}), map[string]any{"foo": ""}, map[string]any{"foo": nil}, true, nil},
		// map.
		{types.Map(types.Text()), map[string]any{"foo": "boo"}, map[string]any{"foo": "boo"}, false, nil},
		{types.Map(types.Array(types.Boolean())), map[string]any{"foo": []any{true, false}}, map[string]any{"foo": []any{true, false}}, false, nil},
	}

	for _, test := range tests {
		got, err := normalize("k", test.t, test.v, test.n, test.l)
		if err != nil {
			t.Fatal(err)
		}
		expected := test.e
		if !cmp.Equal(got, expected) {
			if f, ok := expected.(float64); ok && math.IsNaN(f) {
				if f, ok := got.(float64); ok && math.IsNaN(f) {
					continue
				}
			}
			t.Fatalf("expected %#v, got %#v", expected, got)
		}
	}
}
