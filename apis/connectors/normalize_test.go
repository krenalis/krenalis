//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package connectors

import (
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/types"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"
)

func Test_normalize(t *testing.T) {

	aDateTime := time.Date(2023, 5, 3, 15, 47, 22, 769802537, time.UTC)
	aDate := time.Date(2023, 5, 3, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		t types.Type
		v any
		e any
		l *state.TimeLayouts
	}{
		// Boolean.
		{types.Boolean(), true, true, nil},
		// Int(16).
		{types.Int(16), -6, -6, nil},
		{types.Int(16), -6.0, -6, nil},
		{types.Int(16), json.Number("-6"), -6, nil},
		{types.Int(16), json.Number("-6.0"), -6, nil},
		// Int(32).
		{types.Int(32), -9261, -9261, nil},
		{types.Int(32), -9261.0, -9261, nil},
		{types.Int(32), json.Number("-9261"), -9261, nil},
		{types.Int(32), json.Number("-9261.0"), -9261, nil},
		// Uint(8).
		{types.Uint(8), uint(3), uint(3), nil},
		{types.Uint(8), 3.0, uint(3), nil},
		{types.Uint(8), json.Number("3"), uint(3), nil},
		{types.Uint(8), json.Number("3.0"), uint(3), nil},
		// Uint(32).
		{types.Uint(32), uint(47303), uint(47303), nil},
		{types.Uint(32), 47303.0, uint(47303), nil},
		{types.Uint(32), json.Number("47303"), uint(47303), nil},
		{types.Uint(32), json.Number("47303.0"), uint(47303), nil},
		// Float(32).
		{types.Float(32), float64(float32(12.79)), float64(float32(12.79)), nil},
		{types.Float(32), json.Number("12"), float64(float32(12.0)), nil},
		{types.Float(32), json.Number("12.79"), float64(float32(12.79)), nil},
		{types.Float(32), math.NaN(), math.NaN(), nil},
		// Float(64).
		{types.Float(64), 12.7902743017496882, 12.7902743017496882, nil},
		{types.Float(64), json.Number("12"), 12.0, nil},
		{types.Float(64), json.Number("12.79027430174968829"), 12.79027430174968829, nil},
		{types.Float(64), math.NaN(), math.NaN(), nil},
		// Decimal.
		{types.Decimal(10, 3), decimal.NewFromFloat(6.639), decimal.NewFromFloat(6.639), nil},
		{types.Decimal(10, 3), "6.639", decimal.NewFromFloat(6.639), nil},
		{types.Decimal(10, 3), 6.639, decimal.NewFromFloat(6.639), nil},
		{types.Decimal(10, 3), json.Number("6.639"), decimal.NewFromFloat(6.639), nil},
		// DateTime.
		{types.DateTime(), aDateTime, aDateTime, nil},
		{types.DateTime(), strconv.FormatInt(aDateTime.Unix(), 10), time.Date(2023, 5, 3, 15, 47, 22, 0, time.UTC), &state.TimeLayouts{DateTime: "unix"}},
		{types.DateTime(), strconv.FormatInt(aDateTime.UnixMilli(), 10), time.Date(2023, 5, 3, 15, 47, 22, 769000000, time.UTC), &state.TimeLayouts{DateTime: "unixmilli"}},
		{types.DateTime(), strconv.FormatInt(aDateTime.UnixMicro(), 10), time.Date(2023, 5, 3, 15, 47, 22, 769802000, time.UTC), &state.TimeLayouts{DateTime: "unixmicro"}},
		{types.DateTime(), strconv.FormatInt(aDateTime.UnixNano(), 10), aDateTime, &state.TimeLayouts{DateTime: "unixnano"}},
		{types.DateTime(), "2023-05-03 15:47:22", time.Date(2023, 5, 3, 15, 47, 22, 0, time.UTC), &state.TimeLayouts{DateTime: time.DateTime}},
		{types.DateTime(), "2023-05-03", time.Date(2023, 5, 3, 0, 0, 0, 0, time.UTC), &state.TimeLayouts{DateTime: time.DateOnly}},
		{types.DateTime(), float64(aDateTime.Unix()), time.Date(2023, 5, 3, 15, 47, 22, 0, time.UTC), &state.TimeLayouts{DateTime: "unix"}},
		{types.DateTime(), float64(aDateTime.UnixMilli()), time.Date(2023, 5, 3, 15, 47, 22, 769000000, time.UTC), &state.TimeLayouts{DateTime: "unixmilli"}},
		{types.DateTime(), float64(aDateTime.UnixMicro()), time.Date(2023, 5, 3, 15, 47, 22, 769802000, time.UTC), &state.TimeLayouts{DateTime: "unixmicro"}},
		{types.DateTime(), float64(aDateTime.UnixNano()), time.Date(2023, 5, 3, 15, 47, 22, 769802496, time.UTC), &state.TimeLayouts{DateTime: "unixnano"}},
		{types.DateTime(), json.Number(strconv.FormatInt(aDateTime.Unix(), 10)), time.Date(2023, 5, 3, 15, 47, 22, 0, time.UTC), &state.TimeLayouts{DateTime: "unix"}},
		// Date.
		{types.Date(), aDate, aDate, nil},
		{types.Date(), "2023-05-03", aDate, &state.TimeLayouts{Date: time.DateOnly}},
		{types.Date(), "Wed, 03 May 2023", aDate, &state.TimeLayouts{Date: "Mon, 02 Jan 2006"}},
		// Time.
		{types.Time(), time.Date(2023, 5, 3, 17, 34, 41, 804019312, time.UTC), time.Date(1970, 1, 1, 17, 34, 41, 804019312, time.UTC), nil},
		{types.Time(), "00:00:00", time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), &state.TimeLayouts{Time: "15:04:05.999999999"}},
		{types.Time(), "13:16:47.801", time.Date(1970, 1, 1, 13, 16, 47, 801000000, time.UTC), &state.TimeLayouts{Time: "15:04:05.999999999"}},
		{types.Time(), "23:59:59.999999999", time.Date(1970, 1, 1, 23, 59, 59, 999999999, time.UTC), &state.TimeLayouts{Time: "15:04:05.999999999"}},
		{types.Time(), "09:22:51.834", time.Date(1970, 1, 1, 9, 22, 51, 834000000, time.UTC), &state.TimeLayouts{Time: "15:04:05.000"}},
		{types.Time(), "09h 31m 13s", time.Date(1970, 1, 1, 9, 31, 13, 0, time.UTC), &state.TimeLayouts{Time: "15h 04m 05s"}},
		// Year.
		{types.Year(), 2023, 2023, nil},
		{types.Year(), 2023.0, 2023, nil},
		{types.Year(), json.Number("2023"), 2023, nil},
		// UUID.
		{types.UUID(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000", nil},
		// JSON.
		{types.JSON(), json.RawMessage(`{"a":5}`), json.RawMessage(`{"a":5}`), nil},
		{types.JSON(), json.RawMessage(" \t503\n"), json.RawMessage(`503`), nil},
		{types.JSON(), json.Number("302"), json.Number("302"), nil},
		{types.JSON(), `{"a":5}`, `{"a":5}`, nil},
		{types.JSON(), map[string]any{"a": 5.0}, map[string]any{"a": 5.0}, nil},
		// Inet.
		{types.Inet(), "127.0.0.1", "127.0.0.1", nil},
		{types.Inet(), "2001:0db8:0000:0000:0000:ff00:0042:8329", "2001:db8::ff00:42:8329", nil},
		// Text.
		{types.Text(), "foo", "foo", nil},
		{types.Text().WithValues("foo", "boo"), "boo", "boo", nil},
		{types.Text().WithRegexp(regexp.MustCompile(`oo$`)), "foo", "foo", nil},
		{types.Text().WithByteLen(3), "boo", "boo", nil},
		{types.Text().WithCharLen(3), "bòò", "bòò", nil},
		// Array.
		{types.Array(types.Int(32)), []any{1, 2}, []any{1, 2}, nil},
		{types.Array(types.Int(32)), []any{1.0, 2.0}, []any{1, 2}, nil},
		{types.Array(types.Int(32)), []any{json.Number("1.0"), json.Number("2.0")}, []any{1, 2}, nil},
		{types.Array(types.Array(types.Text())), []any{[]any{"foo"}, []any{"foo"}}, []any{[]any{"foo"}, []any{"foo"}}, nil},
		// Object.
		{types.Object([]types.Property{{Name: "foo", Type: types.Text()}, {Name: "boo", Type: types.Int(32)}}), map[string]any{"foo": "alt", "boo": 3}, map[string]any{"foo": "alt", "boo": 3}, nil},
		// Map.
		{types.Map(types.Text()), map[string]any{"foo": "boo"}, map[string]any{"foo": "boo"}, nil},
		{types.Map(types.Array(types.Boolean())), map[string]any{"foo": []any{true, false}}, map[string]any{"foo": []any{true, false}}, nil},
	}

	for _, test := range tests {
		got, err := normalize("k", test.t, test.v, false, test.l)
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

func Test_validJSON(t *testing.T) {

	tests := []struct {
		src   any
		valid bool
	}{
		{nil, false},
		{"", true},
		{"boo", true},
		{"\xff", false},
		{true, true},
		{12, false},
		{23.985, true},
		{[]any{1.0, "a", json.RawMessage(`{"foo":[1,2,3]}`)}, true},
		{map[string]any{"": "", "a": 5.4}, true},
		{math.NaN(), false},
		{math.Inf(-1), false},
		{json.Number(""), false},
		{json.Number("515"), true},
		{json.Number("-34.96"), true},
		{json.Number("+34.96"), false},
		{json.RawMessage(""), false},
		{json.RawMessage("+"), false},
		{json.RawMessage("5"), true},
		{json.RawMessage(" 5 "), true},
		{json.RawMessage(`"\xff"`), false},
		{json.RawMessage(`{"foo":[1,2,3]}`), true},
		{json.RawMessage("{\n\t\"foo\": [1,2,3]\n}"), true},
	}

	for _, test := range tests {
		valid := validJSON(test.src)
		if valid != test.valid {
			if test.valid {
				t.Fatalf("%#v: expected a valid JSON, got an invalid one", test.src)
			}
			t.Fatalf("%#v: expected a invalid JSON, got a valid one", test.src)
		}
	}

}
