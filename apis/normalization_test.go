//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"chichi/connector"
	"chichi/connector/types"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"
)

func TestNormalizeAppPropertyValue(t *testing.T) {

	aDateTime := connector.MustDateTime(time.Date(2023, 5, 3, 15, 47, 22, 769802537, time.UTC))
	aDate := connector.MustDate(time.Date(2023, 5, 3, 0, 0, 0, 0, time.UTC))
	aTime, _ := connector.ParseTime("17:36:19.4016849239")

	tests := []struct {
		t types.Type
		v any
		e any
	}{
		// Boolean.
		{types.Boolean(), true, true},
		// Int.
		{types.Int(), -9261, -9261},
		{types.Int(), -9261.0, -9261},
		{types.Int(), json.Number("-9261"), -9261},
		{types.Int(), json.Number("-9261.0"), -9261},
		// Int16.
		{types.Int8(), -6, -6},
		{types.Int8(), -6.0, -6},
		{types.Int8(), json.Number("-6"), -6},
		{types.Int8(), json.Number("-6.0"), -6},
		// UInt.
		{types.UInt(), uint(47303), uint(47303)},
		{types.UInt(), 47303.0, uint(47303)},
		{types.UInt(), json.Number("47303"), uint(47303)},
		{types.UInt(), json.Number("47303.0"), uint(47303)},
		// UInt8.
		{types.UInt8(), uint(3), uint(3)},
		{types.UInt8(), 3.0, uint(3)},
		{types.UInt8(), json.Number("3"), uint(3)},
		{types.UInt8(), json.Number("3.0"), uint(3)},
		// Float.
		{types.Float(), 12.7902743017496882, 12.7902743017496882},
		{types.Float(), json.Number("12"), 12.0},
		{types.Float(), json.Number("12.79027430174968829"), 12.79027430174968829},
		// Float32.
		{types.Float32(), 12.79, 12.79},
		{types.Float32(), json.Number("12"), 12.0},
		{types.Float32(), json.Number("12.79"), 12.79},
		// Decimal.
		{types.Decimal(10, 3), decimal.NewFromFloat(6.639), decimal.NewFromFloat(6.639)},
		{types.Decimal(10, 3), "6.639", decimal.NewFromFloat(6.639)},
		{types.Decimal(10, 3), 6.639, decimal.NewFromFloat(6.639)},
		{types.Decimal(10, 3), json.Number("6.639"), decimal.NewFromFloat(6.639)},
		// DateTime.
		{types.DateTime(), aDateTime, aDateTime},
		{types.DateTime().WithLayout("s"), strconv.FormatInt(aDateTime.Unix(), 10), connector.MustDateTime(time.Date(2023, 5, 3, 15, 47, 22, 0, time.UTC))},
		{types.DateTime().WithLayout("ms"), strconv.FormatInt(aDateTime.UnixMilli(), 10), connector.MustDateTime(time.Date(2023, 5, 3, 15, 47, 22, 769000000, time.UTC))},
		{types.DateTime().WithLayout("us"), strconv.FormatInt(aDateTime.UnixMicro(), 10), connector.MustDateTime(time.Date(2023, 5, 3, 15, 47, 22, 769802000, time.UTC))},
		{types.DateTime().WithLayout("ns"), strconv.FormatInt(aDateTime.UnixNano(), 10), aDateTime},
		{types.DateTime().WithLayout(time.DateTime), "2023-05-03 15:47:22", connector.MustDateTime(time.Date(2023, 5, 3, 15, 47, 22, 0, time.UTC))},
		{types.DateTime().WithLayout(time.DateOnly), "2023-05-03", connector.MustDateTime(time.Date(2023, 5, 3, 0, 0, 0, 0, time.UTC))},
		{types.DateTime().WithLayout("s"), float64(aDateTime.Unix()), connector.MustDateTime(time.Date(2023, 5, 3, 15, 47, 22, 0, time.UTC))},
		{types.DateTime().WithLayout("ms"), float64(aDateTime.UnixMilli()), connector.MustDateTime(time.Date(2023, 5, 3, 15, 47, 22, 769000000, time.UTC))},
		{types.DateTime().WithLayout("us"), float64(aDateTime.UnixMicro()), connector.MustDateTime(time.Date(2023, 5, 3, 15, 47, 22, 769802000, time.UTC))},
		{types.DateTime().WithLayout("ns"), float64(aDateTime.UnixNano()), connector.MustDateTime(time.Date(2023, 5, 3, 15, 47, 22, 769802496, time.UTC))},
		{types.DateTime().WithLayout("s"), json.Number(strconv.FormatInt(aDateTime.Unix(), 10)), connector.MustDateTime(time.Date(2023, 5, 3, 15, 47, 22, 0, time.UTC))},
		// Date.
		{types.Date(), aDate, aDate},
		{types.Date(), "2023-05-03", aDate},
		{types.Date().WithLayout(time.DateOnly), "2023-05-03", aDate},
		{types.Date().WithLayout("Mon, 02 Jan 2006"), "Wed, 03 May 2023", aDate},
		// Time.
		{types.Time(), aTime, aTime},
		{types.Time().WithLayout("s"), float64(48192), connector.Time(48192 * time.Second)},
		{types.Time().WithLayout("ms"), float64(48192517), connector.Time(48192517 * time.Millisecond)},
		{types.Time().WithLayout("us"), float64(48192517065), connector.Time(48192517065 * time.Microsecond)},
		{types.Time().WithLayout("ns"), float64(48192517065128), connector.Time(48192517065128)},
		{types.Time(), "09:43:22.305129745", connector.Time(35002305129745)},
		{types.Time().WithLayout("15"), "10", connector.Time(10 * time.Hour)},
		{types.Time().WithLayout("s"), json.Number("72204"), connector.Time(72204 * time.Second)},
		// Year.
		{types.Year(), 2023, 2023},
		{types.Year(), 2023.0, 2023},
		{types.Year(), json.Number("2023"), 2023},
		// UUID.
		{types.UUID(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000"},
		// JSON.
		{types.JSON(), json.RawMessage(`{"a":5}`), json.RawMessage(`{"a":5}`)},
		{types.JSON(), `{"a":5}`, json.RawMessage(`{"a":5}`)},
		// Inet.
		{types.Inet(), "127.0.0.1", "127.0.0.1"},
		{types.Inet(), "2001:0db8:0000:0000:0000:ff00:0042:8329", "2001:db8::ff00:42:8329"},
		// Text.
		{types.Text(), "foo", "foo"},
		// Array.
		{types.Array(types.Int()), []any{1, 2}, []any{1, 2}},
		{types.Array(types.Int()), []any{1.0, 2.3}, []any{1, 2}},
		{types.Array(types.Int()), []any{json.Number("1.0"), json.Number("2.3")}, []any{1, 2}},
		{types.Array(types.Array(types.Text())), []any{[]any{"foo"}, []any{"foo"}}, []any{[]any{"foo"}, []any{"foo"}}},
		// Object.
		{types.Object([]types.Property{{Name: "foo", Type: types.Text()}, {Name: "boo", Type: types.Int()}}), map[string]any{"foo": "alt", "boo": 3}, map[string]any{"foo": "alt", "boo": 3}},
		// Map.
		{types.Map(types.Text()), map[string]any{"foo": "boo"}, map[string]any{"foo": "boo"}},
		{types.Map(types.Array(types.Boolean())), map[string]any{"foo": []any{true, false}}, map[string]any{"foo": []any{true, false}}},
	}

	for _, test := range tests {
		got, err := normalizeAppPropertyValue("k", false, test.t, test.v)
		if err != nil {
			t.Fatal(err)
		}
		expected := test.e
		if !cmp.Equal(got, expected) {
			t.Fatalf("expected %#v, got %#v", expected, got)
		}
	}
}
