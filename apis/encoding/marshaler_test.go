//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package encoding

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/meergo/meergo/types"

	"github.com/shopspring/decimal"
)

var schema = types.Object([]types.Property{
	{
		Name: "Boolean",
		Type: types.Boolean(),
	},
	{
		Name: "Int8",
		Type: types.Int(8),
	},
	{
		Name: "Int16",
		Type: types.Int(16),
	},
	{
		Name: "Int24",
		Type: types.Int(24),
	},
	{
		Name: "Int32",
		Type: types.Int(32),
	},
	{
		Name: "Int64",
		Type: types.Int(64),
	},
	{
		Name: "Uint8",
		Type: types.Uint(8),
	},
	{
		Name: "Uint16",
		Type: types.Uint(16),
	},
	{
		Name: "Uint24",
		Type: types.Uint(24),
	},
	{
		Name: "Uint32",
		Type: types.Uint(32),
	},
	{
		Name: "Uint64",
		Type: types.Uint(64),
	},
	{
		Name: "Float32",
		Type: types.Float(32),
	},
	{
		Name: "Float64",
		Type: types.Float(64),
	},
	{
		Name: "Float64_NaN",
		Type: types.Float(64),
	},
	{
		Name: "Float64_Positive_Infinity",
		Type: types.Float(64),
	},
	{
		Name: "Float64_Negative_Infinity",
		Type: types.Float(64),
	},
	{
		Name: "Decimal",
		Type: types.Decimal(10, 3),
	},
	{
		Name: "DateTime",
		Type: types.DateTime(),
	},
	{
		Name: "Date",
		Type: types.Date(),
	},
	{
		Name: "Time",
		Type: types.Time(),
	},
	{
		Name: "Year",
		Type: types.Year(),
	},
	{
		Name: "UUID",
		Type: types.UUID(),
	},
	{
		Name: "JSON_RawMessage",
		Type: types.JSON(),
	},
	{
		Name: "JSON_bool",
		Type: types.JSON(),
	},
	{
		Name: "JSON_string",
		Type: types.JSON(),
	},
	{
		Name: "JSON_float64",
		Type: types.JSON(),
	},
	{
		Name: "JSON_Number",
		Type: types.JSON(),
	},
	{
		Name: "JSON_slice",
		Type: types.JSON(),
	},
	{
		Name: "JSON_map",
		Type: types.JSON(),
	},
	{
		Name: "JSON_null",
		Type: types.JSON(),
	},
	{
		Name: "Inet",
		Type: types.Inet(),
	},
	{
		Name: "Text",
		Type: types.Text(),
	},
	{
		Name: "Array",
		Type: types.Array(types.Text()),
	},
	{
		Name: "Object",
		Type: types.Object([]types.Property{
			{
				Name: "a",
				Type: types.Int(32),
			},
			{
				Name: "b",
				Type: types.Boolean(),
			},
		}),
	},
	{
		Name: "Map",
		Type: types.Map(types.Int(32)),
	},
})

var value = map[string]any{
	"Boolean":                   true,
	"Int8":                      -12,
	"Int16":                     8023,
	"Int24":                     -2880217,
	"Int32":                     1307298102,
	"Int64":                     927041163082605,
	"Uint8":                     uint(12),
	"Uint16":                    uint(8023),
	"Uint24":                    uint(2880217),
	"Uint32":                    uint(1307298102),
	"Uint64":                    uint(927041163082605),
	"Float32":                   float64(float32(57.16038)),
	"Float64":                   18372.36240184391,
	"Float64_NaN":               math.NaN(),
	"Float64_Positive_Infinity": math.Inf(1),
	"Float64_Negative_Infinity": math.Inf(-1),
	"Decimal":                   decimal.RequireFromString("1752.064"),
	"DateTime":                  time.Date(2023, 10, 17, 9, 34, 25, 836042841, time.UTC),
	"Date":                      time.Date(2023, 10, 17, 0, 0, 0, 0, time.UTC),
	"Time":                      time.Date(1970, 01, 01, 9, 34, 25, 836042841, time.UTC),
	"Year":                      2023,
	"UUID":                      "550e8400-e29b-41d4-a716-446655440000",
	"JSON_RawMessage":           json.RawMessage(`{"foo":5,"boo":true}`),
	"JSON_bool":                 true,
	"JSON_string":               `foo & boo \u`,
	"JSON_float64":              23.871,
	"JSON_Number":               json.Number("85802.7305"),
	"JSON_slice":                []any{"foo", 3, true},
	"JSON_map":                  map[string]any{"a": 1, "b": 2},
	"JSON_null":                 json.RawMessage(`null`),
	"Inet":                      "192.158.1.38",
	"Text":                      "some text",
	"Array":                     []any{"foo", "boo"},
	"Object":                    map[string]any{"a": 9, "b": false},
	"Map":                       map[string]any{"a": 1, "b": 2, "c": 3},
}

func Test_Marshal(t *testing.T) {
	tests := []struct {
		name   string
		schema types.Type
		value  map[string]any
		result []byte
	}{
		{
			name:   "Types",
			schema: schema,
			value:  value,
			result: []byte(`{"Boolean":true,"Int8":-12,"Int16":8023,"Int24":-2880217,"Int32":1307298102,"Int64":"927041163082605","Uint8":12,"Uint16":8023,"Uint24":2880217,"Uint32":1307298102,"Uint64":"927041163082605","Float32":57.16038,"Float64":18372.36240184391,"Float64_NaN":"NaN","Float64_Positive_Infinity":"Infinity","Float64_Negative_Infinity":"-Infinity","Decimal":"1752.064","DateTime":"2023-10-17T09:34:25.836042841Z","Date":"2023-10-17","Time":"09:34:25.836042841","Year":2023,"UUID":"550e8400-e29b-41d4-a716-446655440000","JSON_RawMessage":"{\"foo\":5,\"boo\":true}","JSON_bool":"true","JSON_string":"\"foo & boo \\\\u\"","JSON_float64":"23.871","JSON_Number":"85802.7305","JSON_slice":"[\"foo\",3,true]","JSON_map":"{\"a\":1,\"b\":2}","JSON_null":"null","Inet":"192.158.1.38","Text":"some text","Array":["foo","boo"],"Object":{"a":9,"b":false},"Map":{"a":1,"b":2,"c":3}}`),
		},
		{
			name:   "Empty",
			schema: schema,
			value:  map[string]any{},
			result: []byte(`{}`),
		},
		{
			name: "JSON nil",
			schema: types.Object([]types.Property{
				{
					Name:     "a",
					Type:     types.JSON(),
					Nullable: true,
				},
			}),
			value:  map[string]any{"a": nil},
			result: []byte(`{"a":null}`),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Marshal(test.schema, test.value)
			if err != nil {
				t.Fatalf("Marshal: unexpected error: %s", err)
			}
			if !bytes.Equal(test.result, got) {
				t.Fatalf("Marshal: expected %s, got %s", string(test.result), string(got))
			}
		})
	}
}

func Test_MarshalSlice(t *testing.T) {
	tests := []struct {
		name   string
		schema types.Type
		values []map[string]any
		result []byte
	}{
		{
			name: "Slice",
			schema: types.Object([]types.Property{
				{
					Name: "a",
					Type: types.Boolean(),
				},
				{
					Name: "b",
					Type: types.Int(8),
				},
			}),
			values: []map[string]any{{"a": true, "b": 3}, {"a": false, "b": 1}, {"a": true, "b": 14}},
			result: []byte(`[{"a":true,"b":3},{"a":false,"b":1},{"a":true,"b":14}]`),
		},
		{
			name: "Text encoding",
			schema: types.Object([]types.Property{
				{
					Name: "a",
					Type: types.Text(),
				},
			}),
			values: []map[string]any{
				{"a": ``},
				{"a": `'`},
				{"a": `"`},
			},
			result: []byte(`[{"a":""},{"a":"'"},{"a":"\""}]`),
		},
		{
			name: "ReadOptional property",
			schema: types.Object([]types.Property{
				{
					Name: "a",
					Type: types.Text(),
				},
				{
					Name:         "b",
					Type:         types.Boolean(),
					ReadOptional: true,
				},
			}),
			values: []map[string]any{
				{"a": "foo", "b": true},
				{"a": "foo"},
			},
			result: []byte(`[{"a":"foo","b":true},{"a":"foo"}]`),
		},
		{
			name: "Mix",
			schema: types.Object([]types.Property{
				{
					Name: "a",
					Type: types.Text(),
				},
				{
					Name: "b",
					Type: types.Object([]types.Property{
						{Name: "x", Type: types.Int(32), ReadOptional: true},
						{Name: "y", Type: types.Int(32)},
					}),
					ReadOptional: true,
					Nullable:     true,
				},
			}),
			values: []map[string]any{
				{},
				{"a": "foo"},
				{"a": "foo", "b": nil},
				{"a": "foo", "b": map[string]any{"y": 45}},
				{"a": "foo", "b": map[string]any{"x": 12, "y": 45}},
			},
			result: []byte(`[{},{"a":"foo"},{"a":"foo","b":null},{"a":"foo","b":{"y":45}},{"a":"foo","b":{"x":12,"y":45}}]`),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := MarshalSlice(test.schema, test.values)
			if err != nil {
				t.Fatalf("MarshalSlice: unexpected error: %s", err)
			}
			if !bytes.Equal(test.result, got) {
				t.Fatalf("MarshalSlice: expected %s, got %s", string(test.result), string(got))
			}
		})
	}
}
