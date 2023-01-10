//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package clickhouse

import (
	"testing"

	"chichi/apis/types"
)

func TestTypes(t *testing.T) {

	tests := []struct {
		s string
		t types.Type
	}{
		{`UInt8`, types.UInt8()},
		{`UInt16`, types.UInt16()},
		{`UInt32`, types.UInt()},
		{`UInt64`, types.UInt64()},
		{`Int8`, types.Int8()},
		{`Int16`, types.Int16()},
		{`Int32`, types.Int()},
		{`Int64`, types.Int64()},
		{`Float32`, types.Float32()},
		{`Float64`, types.Float()},
		{`Decimal(12, 3)`, types.Decimal(12, 3)},
		{`Boolean`, types.Boolean()},
		{`String`, types.Text()},
		{`UUID`, types.UUID()},
		{`Date`, types.Date("2006-01-02")},
		{`Date32`, types.Int()},
		{`DateTime`, types.DateTime("2006-01-02 15:04:05")},
		{`DateTime('Asia/Istanbul')`, types.DateTime("2006-01-02 15:04:05")},
		{`DateTime64(0)`, types.DateTime("2006-01-02 15:04:05")},
		{`DateTime64(9)`, types.DateTime("2006-01-02 15:04:05.999999999")},
		{`DateTime64(3, 'Asia/Istanbul')`, types.DateTime("2006-01-02 15:04:05.999")},
		{`Enum8('hello' = 1, 'world' = 2)`, types.Text().WithEnum([]string{"hello", "world"})},
		{`Enum8('a' = -10, 'b' = -8)`, types.Text().WithEnum([]string{"a", "b"})},
		{`Enum16('\b', '\f', '\r', '\n', '\t', '\0', '\a', '\v')`, types.Text().WithEnum([]string{"\b", "\f", "\r", "\n", "\t", "\x00", "\a", "\v"})},
		{`Enum16('0\b1\f2\r3\n4\t5\06\a7\v8')`, types.Text().WithEnum([]string{"0\b1\f2\r3\n4\t5\x006\a7\v8"})},
		{`Enum8('\e', '\\', '''', '\x3a', 'a\x7fb', '\x7F')`, types.Text().WithEnum([]string{"e", "\\", "'", "\x3a", "a\x7fb", "\x7F"})},
		{`LowCardinality(String)`, types.Text()},
		{`Array(String)`, types.Array(types.Text())},
		{`Array(DateTime64(9))`, types.Array(types.DateTime("2006-01-02 15:04:05.999999999"))},
		{`Array(Array(Enum8('hello' = 1, 'world' = 2)))`, types.Array(types.Array(types.Text().WithEnum([]string{"hello", "world"})))},
		{`Array(Nullable(String))`, types.Array(types.Text().WithNull())},
		{`JSON`, types.JSON()},
		{`Nullable(String)`, types.Text().WithNull()},
		{`Nullable(DateTime('Asia/Istanbul'))`, types.DateTime("2006-01-02 15:04:05").WithNull()},
		{`IPv4`, types.UInt()},
		{`IPv6`, types.Text(types.Bytes(16))},
		{`FixedString(10)`, types.Text(types.Bytes(10))},
		{`Map(String, Int32)`, types.Map(types.Int())},
		{`Map(String, Array(String))`, types.Map(types.Array(types.Text()))},
	}

	for _, test := range tests {
		got := columnType(test.s)
		if got.Valid() != test.t.Valid() {
			if test.t.Valid() {
				t.Errorf("%s: expecting a valid type, got an invalid type", test.s)
			}
			t.Errorf("%s: expecting an invalid type, got a valid type: %#v", test.s, got)
		}
		if !got.EqualTo(test.t) {
			t.Errorf("%s: unexpected type: %#v", test.s, got)
		}
	}
}

func TestUnsupportedTypes(t *testing.T) {

	tests := []string{
		// Unsupported types.
		`UInt128`, `UInt256`, `Int128`, `Int256`, `AggregateFunction(uniq, UInt64)`, `tuple(String,Int32)`,
		`Point`, `Ring`, `Polygon`, `MultiPolygon`, `SimpleAggregateFunction(sum, Double)`,

		// Invalid types (note that some invalid types are intentionally parsed as valid.)
		``, ` `, "\x00", "\xFF", `a`, `Uint8(String)`, `Array`, `Array(`, `Array)`, `Array(String`, `Array (String)`,
		`Enum8('\')`, `Enum8(''')`, `Enum8('\x80')`, `Enum8('a\xac ')`, `Enum8('\xFF')`,
	}

	for _, test := range tests {
		got := columnType(test)
		if got.Valid() {
			t.Errorf("%s: expecting an invalid type, got a valid type: %#v", test, got)
		}
	}
}
