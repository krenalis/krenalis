// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"fmt"
	"math"
	"net"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/tools/decimal"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"

	"github.com/google/go-cmp/cmp"
)

type customJSONMarshaller []byte

func (m customJSONMarshaller) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte(`null`), nil
	}
	return m, nil
}

type failingJSONMarshaller struct{}

func (f failingJSONMarshaller) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("marshal error")
}

type nilJSONMarshaller struct{}

func (f nilJSONMarshaller) MarshalJSON() ([]byte, error) {
	return nil, nil
}

func Test_normalize(t *testing.T) {

	aDateTime := time.Date(2023, 5, 3, 15, 47, 22, 769802537, time.UTC)
	aDate := time.Date(2023, 5, 3, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		typ      types.Type
		value    any
		expected any
		null     bool
		layout   *state.TimeLayouts
	}{
		// string.
		{types.String(), "foo", "foo", false, nil},
		{types.String().WithValues("foo", "boo"), "boo", "boo", false, nil},
		{types.String().WithPattern(regexp.MustCompile(`oo$`)), "foo", "foo", false, nil},
		{types.String().WithMaxByteLength(3), "boo", "boo", false, nil},
		{types.String().WithMaxLength(3), "bòò", "bòò", false, nil},
		{types.String().WithValues("foo", "boo"), "", nil, true, nil},
		{types.String(), []byte(nil), nil, true, nil},
		// boolean.
		{types.Boolean(), true, true, false, nil},
		// int(16).
		{types.Int(16), -6, -6, false, nil},
		{types.Int(16), -6.0, -6, false, nil},
		{types.Int(16), []byte(nil), nil, true, nil},
		// int(32).
		{types.Int(32), -9261, -9261, false, nil},
		{types.Int(32), -9261.0, -9261, false, nil},
		{types.Int(32), []byte(nil), nil, true, nil},
		{types.Int(32), []byte("-57"), -57, true, nil},
		// uint(8).
		{types.Uint(8), uint(3), uint(3), false, nil},
		{types.Uint(8), 3.0, uint(3), false, nil},
		{types.Uint(8), []byte(nil), nil, true, nil},
		// uint(32).
		{types.Uint(32), uint(47303), uint(47303), false, nil},
		{types.Uint(32), 47303.0, uint(47303), false, nil},
		{types.Uint(32), []byte(nil), nil, true, nil},
		// float(32).
		{types.Float(32), float64(float32(12.79)), float64(float32(12.79)), false, nil},
		{types.Float(32), math.NaN(), math.NaN(), false, nil},
		{types.Float(32), []byte(nil), nil, true, nil},
		// float(64).
		{types.Float(64), 12.7902743017496882, 12.7902743017496882, false, nil},
		{types.Float(64), math.NaN(), math.NaN(), false, nil},
		{types.Float(64), []byte(nil), nil, true, nil},
		// decimal.
		{types.Decimal(10, 3), "6.639e2", decimal.MustParse("663.9"), false, nil},
		{types.Decimal(8, 0), 793012, decimal.MustInt(793012), false, nil},
		{types.Decimal(5, 0), -14044, decimal.MustInt(-14044), false, nil},
		{types.Decimal(18, 14), 23.94758746151403, decimal.MustParse("23.94758746151403"), false, nil},
		{types.Decimal(6, 3), float32(23.94758746151403), decimal.MustParse("23.948"), false, nil},
		{types.Decimal(3, 2), "", nil, true, nil},
		{types.Decimal(3, 2), decimal.MustInt(0), decimal.MustInt(0), false, nil},
		{types.Decimal(3, 2), decimal.MustParse("3.14"), decimal.MustParse("3.14"), false, nil},
		{types.Decimal(3, 2), decimal.MustParse("3.14"), decimal.MustParse("3.14"), true, nil},
		{types.Decimal(3, 2), []byte(nil), nil, true, nil},
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
		{types.Time(), []byte(nil), nil, true, &state.TimeLayouts{Time: "15h 04m 05s"}},
		// year.
		{types.Year(), 2023, 2023, false, nil},
		{types.Year(), 2023.0, 2023, false, nil},
		{types.Year(), []byte(nil), nil, true, nil},
		// uuid.
		{types.UUID(), "123e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000", false, nil},
		{types.UUID(), "", nil, true, nil},
		{types.UUID(), []byte{211, 124, 89, 213, 136, 127, 68, 248, 143, 250, 126, 36, 49, 79, 71, 62}, "d37c59d5-887f-44f8-8ffa-7e24314f473e", false, nil},
		{types.UUID(), []byte{211, 124, 89, 213, 136, 127, 68, 248, 143, 250, 126, 36, 49, 79, 71, 62}, "d37c59d5-887f-44f8-8ffa-7e24314f473e", true, nil},
		{types.UUID(), []byte(nil), nil, true, nil},
		// json.
		{types.JSON(), json.Value(`{"a": 5}`), json.Value(`{"a": 5}`), false, nil},
		{types.JSON(), []byte(`{"a": 5}`), json.Value(`{"a": 5}`), false, nil},
		{types.JSON(), `{"a": 5}`, json.Value(`{"a": 5}`), false, nil},
		{types.JSON(), customJSONMarshaller(`{"a": 5}`), json.Value(`{"a": 5}`), false, nil},
		{types.JSON(), json.Value(" \t503\n"), json.Value(" \t503\n"), false, nil},
		{types.JSON(), json.Value("302"), json.Value("302"), false, nil},
		{types.JSON(), []byte(`{ "a": 5 }`), json.Value(`{ "a": 5 }`), false, nil},
		{types.JSON(), "", nil, true, nil},
		{types.JSON(), customJSONMarshaller(nil), json.Value(`null`), false, nil},
		{types.JSON(), customJSONMarshaller(nil), json.Value(`null`), true, nil},
		// inet.
		{types.Inet(), "127.0.0.1", "127.0.0.1", false, nil},
		{types.Inet(), "192.168.1.10/24", "192.168.1.10", false, nil},
		{types.Inet(), "2001:0db8:0000:0000:0000:ff00:0042:8329", "2001:db8::ff00:42:8329", false, nil},
		{types.Inet(), "2001:0db8:85a3::8a2e:0370:7334/64", "2001:db8:85a3::8a2e:370:7334", false, nil},
		{types.Inet(), "fe80::1ff:fe23:4567:890a%eth0", "fe80::1ff:fe23:4567:890a", false, nil},
		{types.Inet(), "::ffff:192.168.1.10", "::ffff:192.168.1.10", false, nil},
		{types.Inet(), "", nil, true, nil},
		{types.Inet(), net.ParseIP("127.0.0.1"), "127.0.0.1", false, nil},
		{types.Inet(), net.ParseIP("192.168.1.1"), "192.168.1.1", false, nil},
		{types.Inet(), net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334"), "2001:db8:85a3::8a2e:370:7334", false, nil},
		{types.Inet(), net.ParseIP("::ffff:192.168.1.10"), "192.168.1.10", false, nil},
		{types.Inet(), netip.MustParseAddr("127.0.0.1"), "127.0.0.1", false, nil},
		{types.Inet(), netip.MustParseAddr("2001:0db8:0000:0000:0000:ff00:0042:8329"), "2001:db8::ff00:42:8329", false, nil},
		{types.Inet(), netip.MustParseAddr("fe80::1ff:fe23:4567:890a%eth0"), "fe80::1ff:fe23:4567:890a", false, nil},
		{types.Inet(), netip.MustParseAddr("::ffff:192.168.1.10"), "::ffff:192.168.1.10", false, nil},
		{types.Inet(), net.IP(nil), nil, true, nil},
		// array.
		{types.Array(types.Int(32)), []any{1, 2}, []any{1, 2}, false, nil},
		{types.Array(types.Int(32)), []any{1.0, 2.0}, []any{1, 2}, false, nil},
		{types.Array(types.Array(types.String())), []any{[]any{"foo"}, []any{"foo"}}, []any{[]any{"foo"}, []any{"foo"}}, false, nil},
		{types.Array(types.Int(32)), []any(nil), nil, true, nil},
		{types.Array(types.Int(32)), []int(nil), nil, true, nil},
		{types.Array(types.Int(32)), []string(nil), nil, true, nil},
		{types.Array(types.Int(32)), []decimal.Decimal(nil), nil, true, nil},
		{types.Array(types.Int(32)), []map[string][]map[string][]int(nil), nil, true, nil},
		// object.
		{types.Object([]types.Property{{Name: "foo", Type: types.String()}, {Name: "boo", Type: types.Int(32)}}), map[string]any{"foo": "alt", "boo": 3}, map[string]any{"foo": "alt", "boo": 3}, false, nil},
		{types.Object([]types.Property{{Name: "foo", Type: types.Inet(), Nullable: true}}), map[string]any{"foo": ""}, map[string]any{"foo": nil}, true, nil},
		{types.Object([]types.Property{{Name: "foo", Type: types.String()}, {Name: "boo", Type: types.Int(32)}}), map[string]any(nil), nil, true, nil},
		{types.Object([]types.Property{{Name: "foo", Type: types.String()}, {Name: "boo", Type: types.Int(32), ReadOptional: true}}), map[string]any{"foo": "alt", "spurious": 5}, map[string]any{"foo": "alt"}, false, nil},
		// map.
		{types.Map(types.String()), map[string]any{"foo": "boo"}, map[string]any{"foo": "boo"}, false, nil},
		{types.Map(types.Array(types.Boolean())), map[string]any{"foo": []any{true, false}}, map[string]any{"foo": []any{true, false}}, false, nil},
		{types.Map(types.String()), map[string]any(nil), nil, true, nil},
		{types.Map(types.String()), map[string]string(nil), nil, true, nil},
	}

	for _, test := range tests {
		t.Run(fmt.Sprint(test.typ), func(t *testing.T) {
			got, err := normalize("k", test.typ, test.value, test.null, test.layout)
			if err != nil {
				t.Fatal(err)
			}
			expected := test.expected
			if !cmp.Equal(got, expected) {
				if f, ok := expected.(float64); ok && math.IsNaN(f) {
					if f, ok := got.(float64); ok && math.IsNaN(f) {
						return
					}
				}
				t.Fatalf("expected %#v, got %#v", expected, got)
			}
		})
	}
}

func Test_normalize_errors(t *testing.T) {
	timeLayout := &state.TimeLayouts{Time: "15:04"}

	tests := []struct {
		name         string
		typ          types.Type
		value        any
		nullable     bool
		layout       *state.TimeLayouts
		wantContains string
	}{
		{name: "nilNotNullable", typ: types.String(), value: nil, wantContains: "has value null but it is not nullable"},
		{name: "textInvalidType", typ: types.String(), value: 5, wantContains: "has type int"},
		{name: "textInvalidUTF8", typ: types.String(), value: string([]byte{0xff}), wantContains: "does not contain valid UTF-8 characters"},
		{name: "textRegexpMismatch", typ: types.String().WithPattern(regexp.MustCompile(`^foo$`)), value: "bar", wantContains: "contains an unsupported value"},
		{name: "textUnsupportedValue", typ: types.String().WithValues("foo", "bar"), value: "baz", wantContains: "contains an unsupported value"},
		{name: "textTooLong", typ: types.String().WithMaxByteLength(1), value: "toolong", wantContains: "has a value longer than 1 bytes"},
		{name: "textTooManyChars", typ: types.String().WithMaxLength(2), value: "bòò", wantContains: "has a value longer than 2 characters"},
		{name: "booleanWrongString", typ: types.Boolean(), value: "maybe", wantContains: "string value but it is not 'true' or 'false'"},
		{name: "booleanInvalidType", typ: types.Boolean(), value: 1, wantContains: "has type int that is not allowed for type boolean"},
		{name: "intFractionalFloat", typ: types.Int(32), value: 1.5, wantContains: "float64 value that cannot represent an int(32) value"},
		{name: "intOutOfRange", typ: types.Int(8), value: 200, wantContains: "has value which is not in the range"},
		{name: "intStringParseError", typ: types.Int(32), value: "abc", wantContains: "string value that does not represent an int value"},
		{name: "intBytesParseError", typ: types.Int(32), value: []byte("abc"), wantContains: "has a []byte value that cannot represent an int value"},
		{name: "intInvalidType", typ: types.Int(32), value: true, wantContains: "has type bool that is not allowed for type int(32)"},
		{name: "intDecimalTooLarge", typ: types.Int(32), value: decimal.MustParse("1e20"), wantContains: "has a decimal.decimal value that cannot represent an int value"},
		{name: "uintNegativeInt", typ: types.Uint(8), value: -3, wantContains: "has a negative int value that cannot represent an uint(8) value"},
		{name: "uintAboveRange", typ: types.Uint(8), value: uint16(math.MaxUint16), wantContains: "has value which is not in the range [0, 255]"},
		{name: "uintStringParseError", typ: types.Uint(16), value: "bad", wantContains: "has a string value that cannot represent an uint value"},
		{name: "uintBytesParseError", typ: types.Uint(16), value: []byte("bad"), wantContains: "has a []byte value that cannot represent an uint value"},
		{name: "uintNegativeFloat", typ: types.Uint(16), value: -2.5, wantContains: "has a float64 value that cannot represent an uint(16) value"},
		{name: "uintDecimalNegative", typ: types.Uint(16), value: decimal.MustInt(-1), wantContains: "has a decimal.decimal value that cannot represent an uint value"},
		{name: "uintInvalidType", typ: types.Uint(16), value: true, wantContains: "has type bool that is not allowed for type uint(16)"},
		{name: "floatIntNotRepresentable", typ: types.Float(32), value: 1 << 26, wantContains: "has an int value that cannot represent a float(32) value"},
		{name: "float64TooPreciseFor32", typ: types.Float(32), value: 1e40, wantContains: "has a float64 value that cannot represent a float(32) value"},
		{name: "floatDecimalNotRepresentable", typ: types.Float(64), value: decimal.MustParse("1e500"), wantContains: "has a decimal.Decimal value that cannot represent a float(64) value"},
		{name: "floatStringParseError", typ: types.Float(64), value: "abc", wantContains: "has a string value that cannot represent a float(64) value"},
		{name: "floatRange", typ: types.Float(32).WithFloatRange(0, 1), value: 2.0, wantContains: "has a value 2.000000 that is not in the range [0.000000, 1.000000]"},
		{name: "floatInvalidType", typ: types.Float(32), value: true, wantContains: "has type bool that is not allowed for type float(32)"},
		{name: "floatRealNaN", typ: types.Float(64).AsReal(), value: math.NaN(), wantContains: "has a value of NaN, which is not allowed"},
		{name: "decimalOutOfRange", typ: types.Decimal(3, 2), value: "100", wantContains: "cannot represent a decimal(3,2) value"},
		{name: "decimalInvalidType", typ: types.Decimal(3, 2), value: true, wantContains: "has type bool that is not allowed for type decimal(3,2)"},
		{name: "decimalIntTooLarge", typ: types.Decimal(3, 2), value: 123, wantContains: "cannot represent a decimal(3,2) value"},
		{name: "decimalUintTooLarge", typ: types.Decimal(3, 0), value: uint(1000), wantContains: "cannot represent a decimal(3,0) value"},
		{name: "decimalFloatOutOfRange", typ: types.Decimal(4, 2), value: 123.45, wantContains: "cannot represent a decimal(4,2) value"},
		{name: "decimalInvalidString", typ: types.Decimal(3, 2), value: "bad", wantContains: "syntax error"},
		{name: "dateTimeInvalidString", typ: types.DateTime(), value: "bad-date", layout: &state.TimeLayouts{}, wantContains: "has a string value that cannot represent a datetime value"},
		{name: "dateTimeYearOutOfRange", typ: types.DateTime(), value: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC), layout: &state.TimeLayouts{}, wantContains: "has date and time with a year not in range"},
		{name: "dateTimeInvalidType", typ: types.DateTime(), value: 5, layout: &state.TimeLayouts{}, wantContains: "has type int that is not allowed for type datetime"},
		{name: "dateTimeFloatInvalidLayout", typ: types.DateTime(), value: float64(10), layout: &state.TimeLayouts{DateTime: "bad"}, wantContains: "has a float64 value that cannot represent a datetime value"},
		{name: "dateTimeUnixStringParse", typ: types.DateTime(), value: "abc", layout: &state.TimeLayouts{DateTime: "unix"}, wantContains: "has a string value that cannot represent a datetime value"},
		{name: "dateInvalidString", typ: types.Date(), value: "not-a-date", layout: &state.TimeLayouts{Date: time.DateOnly}, wantContains: "has a string value that cannot represent a date value"},
		{name: "dateYearOutOfRange", typ: types.Date(), value: time.Date(0, 6, 1, 0, 0, 0, 0, time.UTC), layout: &state.TimeLayouts{}, wantContains: "has date with a year not in range"},
		{name: "dateInvalidType", typ: types.Date(), value: 5, layout: &state.TimeLayouts{}, wantContains: "has type int that is not allowed for type date"},
		{name: "timeInvalidString", typ: types.Time(), value: "25:61", layout: timeLayout, wantContains: "has a string value that does not represent a time value"},
		{name: "timeInvalidType", typ: types.Time(), value: 12, layout: &state.TimeLayouts{}, wantContains: "has type int that is not allowed for type time"},
		{name: "timeInvalidBytes", typ: types.Time(), value: []byte("bad"), layout: &state.TimeLayouts{}, wantContains: "has a []byte value that cannot represent a time value"},
		{name: "yearFractional", typ: types.Year(), value: 2024.5, wantContains: "has a float64 value that cannot represent a year value"},
		{name: "yearOutOfRange", typ: types.Year(), value: 0, wantContains: "has value which is not in the range [1, 9999]"},
		{name: "yearStringParse", typ: types.Year(), value: "bad", wantContains: "has a string value that cannot represent a year value"},
		{name: "yearInvalidType", typ: types.Year(), value: true, wantContains: "has type bool that is not allowed for type year"},
		{name: "uuidInvalidString", typ: types.UUID(), value: "not-a-uuid", wantContains: "has a string value that cannot represent a uuid value"},
		{name: "uuidInvalidBytes", typ: types.UUID(), value: []byte{1, 2, 3}, wantContains: "has a []byte value that cannot represent a uuid value"},
		{name: "uuidInvalidType", typ: types.UUID(), value: 5, wantContains: "has type int that is not allowed for type uuid"},
		{name: "jsonInvalid", typ: types.JSON(), value: []byte("not json"), wantContains: "is not valid JSON"},
		{name: "jsonMarshalError", typ: types.JSON(), value: failingJSONMarshaller{}, wantContains: "MarshalJSON returned an error"},
		{name: "jsonMarshalNil", typ: types.JSON(), value: nilJSONMarshaller{}, wantContains: "MarshalJSON returned nil"},
		{name: "jsonInvalidType", typ: types.JSON(), value: true, wantContains: "has type bool that is not allowed for type json"},
		{name: "inetInvalidString", typ: types.Inet(), value: "999.999.1.1", wantContains: "has a string value that cannot represent a valid inet value"},
		{name: "inetInvalidIP", typ: types.Inet(), value: net.IP{}, wantContains: "has a net.IP value that cannot represent a valid inet value"},
		{name: "inetInvalidType", typ: types.Inet(), value: 5, wantContains: "has type int that is not allowed for type inet"},
		{name: "arrayInvalidType", typ: types.Array(types.Int(32)), value: 5, wantContains: "has type int that is not allowed"},
		{name: "arrayElementError", typ: types.Array(types.Int(16)), value: []any{1, "bad"}, wantContains: "has a string value that does not represent an int value"},
		{name: "arrayTooFewElements", typ: types.Array(types.String()).WithMinElements(2), value: []any{"ok"}, wantContains: "is an array with 1 elements"},
		{name: "arrayStringNotJSON", typ: types.Array(types.JSON()), value: "notjson", wantContains: "has a string value but does not contain a JSON array"},
		{name: "arrayStringInvalidJSON", typ: types.Array(types.JSON()), value: "[bad", wantContains: "has a string value but is not valid JSON"},
		{name: "arrayStringTooManyElements", typ: types.Array(types.JSON()).WithMaxElements(1), value: "[1,2]", wantContains: "is an array with more than 1 elements"},
		{name: "arrayStringTooFewElements", typ: types.Array(types.JSON()).WithMinElements(2), value: "[1]", wantContains: "is an array with less than 2 elements"},
		{name: "arrayUniqueDuplicated", typ: types.Array(types.Int(32)).WithUnique(), value: []any{1, 1}, wantContains: "contains the duplicated value 1"},
		{name: "objectMissingRequired", typ: types.Object([]types.Property{{Name: "foo", Type: types.String()}}), value: map[string]any{}, wantContains: "property 'k.foo' does not have a value, but the property is not optional for reading"},
		{name: "objectPropertyError", typ: types.Object([]types.Property{{Name: "foo", Type: types.Int(32)}}), value: map[string]any{"foo": "bad"}, wantContains: "property 'k.foo' has a string value that does not represent an int value"},
		{name: "objectInvalidType", typ: types.Object([]types.Property{{Name: "foo", Type: types.String()}}), value: 5, wantContains: "has type int that is not allowed for type object"},
		{name: "mapStringNotJSONObject", typ: types.Map(types.String()), value: "not json", wantContains: "has a string value but does not contain a JSON object"},
		{name: "mapStringInvalidJSON", typ: types.Map(types.JSON()), value: "{bad", wantContains: "has a string value but is not valid JSON"},
		{name: "mapValueTypeError", typ: types.Map(types.Int(32)), value: map[string]any{"ok": 1, "bad": "nope"}, wantContains: "property 'k[\"bad\"]' has a string value that does not represent an int value"},
		{name: "mapInvalidType", typ: types.Map(types.String()), value: 5, wantContains: "has type int that is not allowed for type map"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalize("k", tt.typ, tt.value, tt.nullable, tt.layout)
			if err == nil {
				t.Fatalf("expected error, got value %#v", got)
			}
			if tt.wantContains != "" && !strings.Contains(err.Error(), tt.wantContains) {
				t.Fatalf("expected error containing %q, got %q", tt.wantContains, err)
			}
		})
	}
}
