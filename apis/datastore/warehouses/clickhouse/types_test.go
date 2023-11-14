//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package clickhouse

import (
	"testing"

	"chichi/connector/types"
)

func TestTypes(t *testing.T) {

	tests := []struct {
		s        string
		t        types.Type
		nullable bool
	}{
		{`UInt8`, types.Uint(8), false},
		{`UInt16`, types.Uint(16), false},
		{`UInt32`, types.Uint(32), false},
		{`UInt64`, types.Uint(64), false},
		{`Int8`, types.Int(8), false},
		{`Int16`, types.Int(16), false},
		{`Int32`, types.Int(32), false},
		{`Int64`, types.Int(64), false},
		{`Float32`, types.Float(32), false},
		{`Float64`, types.Float(64), false},
		{`Decimal(12, 3)`, types.Decimal(12, 3), false},
		{`Bool`, types.Boolean(), false},
		{`String`, types.Text(), false},
		{`UUID`, types.UUID(), false},
		{`Date`, types.Date(), false},
		{`Date32`, types.Int(32), false},
		{`DateTime`, types.DateTime(), false},
		{`DateTime('Asia/Istanbul')`, types.DateTime(), false},
		{`DateTime64(0)`, types.DateTime(), false},
		{`DateTime64(9)`, types.DateTime(), false},
		{`DateTime64(3, 'Asia/Istanbul')`, types.DateTime(), false},
		{`Enum8('hello' = 1, 'world' = 2)`, types.Text().WithValues("hello", "world"), false},
		{`Enum8('a' = -10, 'b' = -8)`, types.Text().WithValues("a", "b"), false},
		{`Enum16('\b', '\f', '\r', '\n', '\t', '\0', '\a', '\v')`, types.Text().WithValues("\b", "\f", "\r", "\n", "\t", "\x00", "\a", "\v"), false},
		{`Enum16('0\b1\f2\r3\n4\t5\06\a7\v8')`, types.Text().WithValues("0\b1\f2\r3\n4\t5\x006\a7\v8"), false},
		{`Enum8('\e', '\\', '''', '\x3a', 'a\x7fb', '\x7F')`, types.Text().WithValues("e", "\\", "'", "\x3a", "a\x7fb", "\x7F"), false},
		{`LowCardinality(String)`, types.Text(), false},
		{`Array(String)`, types.Array(types.Text()), false},
		{`Array(DateTime64(9))`, types.Array(types.DateTime()), false},
		{`Array(Array(Enum8('hello' = 1, 'world' = 2)))`, types.Array(types.Array(types.Text().WithValues("hello", "world"))), false},
		{`Array(String)`, types.Array(types.Text()), false},
		{`JSON`, types.JSON(), false},
		{`IPv4`, types.Inet(), false},
		{`IPv6`, types.Inet(), false},
		{`FixedString(10)`, types.Text().WithByteLen(10), false},
		{`Map(String, Int32)`, types.Map(types.Int(32)), false},
		{`Map(String, Array(String))`, types.Map(types.Array(types.Text())), false},
		{`Nullable(Int8)`, types.Int(8), true},
		{`Nullable(Array(String))`, types.Array(types.Text()), true},
	}

	for _, test := range tests {
		gotType, gotNullable := columnType(test.s)
		if gotType.Valid() != test.t.Valid() {
			if test.t.Valid() {
				t.Errorf("%s: expecting a valid type, got an invalid type", test.s)
			}
			t.Errorf("%s: expecting an invalid type, got a valid type: %#v", test.s, gotType)
		}
		if !gotType.EqualTo(test.t) {
			t.Errorf("%s: unexpected type: %#v", test.s, gotType)
		}
		if gotNullable != test.nullable {
			t.Errorf("%s: expecting nullable = %t, got %t", test.s, test.nullable, gotNullable)
		}
	}
}

func TestUnsupportedTypes(t *testing.T) {

	tests := []string{
		// Unsupported types.
		`UInt128`, `UInt256`, `Int128`, `Int256`, `AggregateFunction(uniq, UInt64)`, `tuple(String,Int32)`,
		`Point`, `Ring`, `Polygon`, `MultiPolygon`, `SimpleAggregateFunction(sum, Double)`,
		`Array(Nullable(Int8))`,

		// Invalid types (note that some invalid types are intentionally parsed as valid.)
		``, ` `, "\x00", "\xFF", `a`, `Uint8(String)`, `Array`, `Array(`, `Array)`, `Array(String`, `Array (String)`,
		`Enum8('\')`, `Enum8(''')`, `Enum8('\x80')`, `Enum8('a\xac ')`, `Enum8('\xFF')`,
		`Nullable`,
	}

	for _, test := range tests {
		got, _ := columnType(test)
		if got.Valid() {
			t.Errorf("%s: expecting an invalid type, got a valid type: %#v", test, got)
		}
	}
}
