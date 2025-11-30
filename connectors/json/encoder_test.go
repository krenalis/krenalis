// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package json

import (
	"bytes"
	_ "embed"
	"math"
	"testing"
	"time"

	"github.com/meergo/meergo/tools/decimal"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
)

//go:embed test_data/expected_false_args.json
var expectedFalse []byte

//go:embed test_data/expected_true_args.json
var expectedTrue []byte

func TestEncoder(t *testing.T) {

	var typ = types.Object([]types.Property{
		{Name: "no_value_1", Type: types.Boolean()},
		{Name: "boolean_null", Type: types.Boolean()},
		{Name: "boolean", Type: types.Boolean()},
		{Name: "int8", Type: types.Int(8)},
		{Name: "int16", Type: types.Int(16)},
		{Name: "int24", Type: types.Int(24)},
		{Name: "int32", Type: types.Int(32)},
		{Name: "int64", Type: types.Int(64)},
		{Name: "uint8", Type: types.Uint(8)},
		{Name: "uint16", Type: types.Uint(16)},
		{Name: "uint24", Type: types.Uint(24)},
		{Name: "uint32", Type: types.Uint(32)},
		{Name: "no_value_2", Type: types.Boolean()},
		{Name: "uint64", Type: types.Uint(64)},
		{Name: "float32", Type: types.Float(32)},
		{Name: "float64", Type: types.Float(64)},
		{Name: "float_nan", Type: types.Float(64)},
		{Name: "float_inf", Type: types.Float(64)},
		{Name: "float_neg_inf", Type: types.Float(64)},
		{Name: "decimal", Type: types.Decimal(10, 3)},
		{Name: "datetime", Type: types.DateTime()},
		{Name: "date", Type: types.Date()},
		{Name: "time", Type: types.Time()},
		{Name: "year", Type: types.Year()},
		{Name: "uuid", Type: types.UUID()},
		{Name: "json", Type: types.JSON()},
		{Name: "inet", Type: types.Inet()},
		{Name: "string", Type: types.String()},
		{Name: "array_int", Type: types.Array(types.Int(32))},
		{Name: "array_map", Type: types.Array(types.Map(types.Int(32)))},
		{Name: "object", Type: types.Object([]types.Property{
			{Name: "a", Type: types.Float(64)},
			{Name: "b", Type: types.Object([]types.Property{
				{Name: "b1", Type: types.Time()},
				{Name: "b2", Type: types.Uint(32)},
			})},
		})},
		{Name: "map", Type: types.Map(types.Boolean())},
		{Name: "no_value_3", Type: types.Boolean()},
	})

	v := map[string]any{
		"boolean_null":  nil,
		"boolean":       true,
		"int8":          -34,
		"int16":         91578,
		"int24":         -3083617,
		"int32":         2067926348,
		"int64":         1740762658369,
		"uint8":         uint(34),
		"uint16":        uint(91578),
		"uint24":        uint(3083617),
		"uint32":        uint(2067926348),
		"uint64":        uint(1740762658369),
		"float32":       3.14,
		"float64":       3.14159,
		"float_nan":     math.NaN(),
		"float_inf":     math.Inf(1),
		"float_neg_inf": math.Inf(-1),
		"decimal":       decimal.MustParse("70418339.602755193"),
		"datetime":      time.Date(2023, 05, 20, 12, 37, 22, 792021695, time.UTC),
		"date":          time.Date(2023, 05, 20, 0, 0, 0, 0, time.UTC),
		"time":          time.Date(1970, 1, 1, 12, 37, 22, 792021695, time.UTC),
		"year":          2023,
		"uuid":          "123e4567-e89b-12d3-a456-426614174000",
		"json":          json.Value(`{"foo":"boo","values":[1,2,3]}`),
		"inet":          "2001:db8:85a3::8a2e:370:7334",
		"string":        "abc \x00 \b\t\n\v\f\r \x18 \"&'<>\\ é 日 🌍 \u2028 \u2029",
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
	if !bytes.Equal(got, expectedFalse) {
		t.Fatalf("unexpected newEncoder(false, false, false) result:\ngot     : %s\nexpected: %s\n", got, expectedFalse)
	}

	enc = newEncoder(true, true, true)
	got = enc.Append(nil, typ, v)
	if !bytes.Equal(got, expectedTrue) {
		t.Fatalf("unexpected newEncoder(true, true, true) result:\ngot     : %s\nexpected: %s\n", got, expectedTrue)
	}

}
