//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package json

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"math"
	"testing"
	"time"

	"chichi/connector/types"

	"github.com/shopspring/decimal"
)

//go:embed test_data/expected_false_args.json
var expectedTrue []byte

//go:embed test_data/expected_true_args.json
var expectedFalse []byte

func TestEncoder(t *testing.T) {

	var typ = types.Object([]types.Property{
		{Name: "boolean_null", Type: types.Boolean()},
		{Name: "boolean", Type: types.Boolean()},
		{Name: "int", Type: types.Int()},
		{Name: "int8", Type: types.Int8()},
		{Name: "int16", Type: types.Int16()},
		{Name: "int24", Type: types.Int24()},
		{Name: "int64", Type: types.Int64()},
		{Name: "uint", Type: types.UInt()},
		{Name: "uint8", Type: types.UInt8()},
		{Name: "uint16", Type: types.UInt16()},
		{Name: "uint24", Type: types.UInt24()},
		{Name: "uint64", Type: types.UInt64()},
		{Name: "float", Type: types.Float()},
		{Name: "float32", Type: types.Float32()},
		{Name: "float_nan", Type: types.Float()},
		{Name: "float_inf", Type: types.Float()},
		{Name: "float_neg_inf", Type: types.Float()},
		{Name: "decimal", Type: types.Decimal(10, 3)},
		{Name: "datetime", Type: types.DateTime()},
		{Name: "date", Type: types.Date()},
		{Name: "time", Type: types.Time()},
		{Name: "year", Type: types.Year()},
		{Name: "uuid", Type: types.UUID()},
		{Name: "json", Type: types.JSON()},
		{Name: "inet", Type: types.Inet()},
		{Name: "text", Type: types.Text()},
		{Name: "array_int", Type: types.Array(types.Int())},
		{Name: "array_map", Type: types.Array(types.Map(types.Int()))},
		{Name: "object", Type: types.Object([]types.Property{
			{Name: "a", Type: types.Float()},
			{Name: "b", Type: types.Object([]types.Property{
				{Name: "b1", Type: types.Time()},
				{Name: "b2", Type: types.UInt()},
			})},
		})},
		{Name: "map", Type: types.Map(types.Boolean())},
	})

	v := map[string]any{
		"boolean_null":  nil,
		"boolean":       true,
		"int":           2067926348,
		"int8":          -34,
		"int16":         91578,
		"int24":         -3083617,
		"int64":         1740762658369,
		"uint":          uint(2067926348),
		"uint8":         uint(34),
		"uint16":        uint(91578),
		"uint24":        uint(3083617),
		"uint64":        uint(1740762658369),
		"float":         3.14159,
		"float32":       3.14,
		"float_nan":     math.NaN(),
		"float_inf":     math.Inf(1),
		"float_neg_inf": math.Inf(-1),
		"decimal":       decimal.RequireFromString("70418339.602755193"),
		"datetime":      time.Date(2023, 05, 20, 12, 37, 22, 792021695, time.UTC),
		"date":          time.Date(2023, 05, 20, 0, 0, 0, 0, time.UTC),
		"time":          time.Date(1970, 1, 1, 12, 37, 22, 792021695, time.UTC),
		"year":          2023,
		"uuid":          "123e4567-e89b-12d3-a456-426614174000",
		"json":          json.RawMessage(`{"foo":"boo","values":[1,2,3]}`),
		"inet":          "2001:db8:85a3::8a2e:370:7334",
		"text":          "abc \x00 \b\t\n\v\f\r \x18 \"&'<>\\ é 日 🌍 \u2028 \u2029",
		"array_int":     []any{846, 106728, -23719},
		"array_map":     []any{map[string]any{"a": 1, "b": 2}},
		"object": map[string]any{
			"a": 12.78,
			"b": map[string]any{
				"b1": time.Date(1970, 1, 1, 17, 22, 48, 0, time.UTC),
				"b2": uint(34083),
			},
		},
		"map": map[string]any{"boo": true, "foo": false},
	}

	enc := newEncoder(false, false, false)
	got := enc.Append(nil, typ, v)
	if !bytes.Equal(got, expectedTrue) {
		t.Fatal("unexpected test false result")
	}

	enc = newEncoder(true, true, true)
	got = enc.Append(nil, typ, v)
	if !bytes.Equal(got, expectedFalse) {
		t.Fatal("unexpected test true result")
	}

}
