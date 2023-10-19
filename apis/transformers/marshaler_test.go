//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package transformers

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"chichi/connector/types"

	"github.com/shopspring/decimal"
)

var schema = types.Object([]types.Property{
	{
		Name: "Boolean",
		Type: types.Boolean(),
	},
	{
		Name: "Int",
		Type: types.Int(),
	},
	{
		Name: "Int8",
		Type: types.Int8(),
	},
	{
		Name: "Int16",
		Type: types.Int16(),
	},
	{
		Name: "Int24",
		Type: types.Int24(),
	},
	{
		Name: "Int64",
		Type: types.Int64(),
	},
	{
		Name: "UInt",
		Type: types.UInt(),
	},
	{
		Name: "UInt8",
		Type: types.UInt8(),
	},
	{
		Name: "UInt16",
		Type: types.UInt16(),
	},
	{
		Name: "UInt24",
		Type: types.UInt24(),
	},
	{
		Name: "UInt64",
		Type: types.UInt64(),
	},
	{
		Name: "Float",
		Type: types.Float(),
	},
	{
		Name: "Float32",
		Type: types.Float32(),
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
		Name: "JSON_nil",
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
				Type: types.Int(),
			},
			{
				Name: "b",
				Type: types.Boolean(),
			},
		}),
	},
	{
		Name: "Map",
		Type: types.Map(types.Int()),
	},
})

var values = []map[string]any{
	{
		"Boolean":         true,
		"Int":             1307298102,
		"Int8":            -12,
		"Int16":           8023,
		"Int24":           -2880217,
		"Int64":           927041163082605,
		"UInt":            uint(1307298102),
		"UInt8":           uint(12),
		"UInt16":          uint(8023),
		"UInt24":          uint(2880217),
		"UInt64":          uint(927041163082605),
		"Float":           18372.36240184391,
		"Float32":         57.16038,
		"Decimal":         decimal.RequireFromString("1752.064"),
		"DateTime":        time.Date(2023, 10, 17, 9, 34, 25, 836042841, time.UTC),
		"Date":            time.Date(2023, 10, 17, 0, 0, 0, 0, time.UTC),
		"Time":            time.Date(1970, 01, 01, 9, 34, 25, 836042841, time.UTC),
		"Year":            2023,
		"UUID":            "550e8400-e29b-41d4-a716-446655440000",
		"JSON_RawMessage": json.RawMessage(`{"foo":5,"boo":true}`),
		"JSON_bool":       true,
		"JSON_string":     `foo & boo \u`,
		"JSON_float64":    23.871,
		"JSON_Number":     json.Number("85802.7305"),
		"JSON_slice":      []any{"foo", 3, true},
		"JSON_map":        map[string]any{"a": 1, "b": 2},
		"JSON_nil":        nil,
		"Inet":            "192.158.1.38",
		"Text":            "some text",
		"Array":           []any{"foo", "boo"},
		"Object":          map[string]any{"a": 9, "b": false},
		"Map":             map[string]any{"a": 1, "b": 2, "c": 3},
	},
}

func Test_MarshalJavaScript(t *testing.T) {
	tests := []struct {
		name   string
		schema types.Type
		values []map[string]any
		result []byte
	}{
		{
			name:   "Types",
			schema: schema,
			values: values,
			result: []byte(`[{Boolean:true,Int:1307298102,Int8:-12,Int16:8023,Int24:-2880217,Int64:927041163082605n,UInt:1307298102,UInt8:12,UInt16:8023,UInt24:2880217,UInt64:927041163082605n,Float:18372.36240184391,Float32:57.16038,Decimal:'1752.064',DateTime:new Date(1697535265836),Date:new Date(1697500800000),Time:new Date(34465836),Year:2023,UUID:'550e8400-e29b-41d4-a716-446655440000',JSON_RawMessage:'{\"foo\":5,\"boo\":true}',JSON_bool:'true',JSON_string:'\"foo \u0026 boo \\\\u\"',JSON_float64:'23.871',JSON_Number:'85802.7305',JSON_slice:'[\"foo\",3,true]',JSON_map:'{\"a\":1,\"b\":2}',JSON_nil:'null',Inet:'192.158.1.38',Text:'some text',Array:['foo','boo'],Object:{a:9,b:false},Map:{'a':1,'b':2,'c':3}}]`),
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
				{"a": `&`},
				{"a": `<`},
				{"a": "\u2028"},
				{"a": "\u2029"},
			},
			result: []byte(`[{a:''},{a:'\u0027'},{a:'\"'},{a:'\u0026'},{a:'\u003c'},{a:'\u2028'},{a:'\u2029'}]`),
		},
	}
	var b []byte
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := MarshalJavaScript(b, test.schema, test.values)
			if !bytes.Equal(test.result, got) {
				t.Fatalf("MarshalJavaScript: expected %s, got %s", string(test.result), string(got))
			}
		})
	}
}

func Test_MarshalPython(t *testing.T) {
	tests := []struct {
		name   string
		schema types.Type
		values []map[string]any
		result []byte
	}{
		{
			name:   "Types",
			schema: schema,
			values: values,
			result: []byte(`[{'Boolean':True,'Int':1307298102,'Int8':-12,'Int16':8023,'Int24':-2880217,'Int64':927041163082605,'UInt':1307298102,'UInt8':12,'UInt16':8023,'UInt24':2880217,'UInt64':927041163082605,'Float':18372.36240184391,'Float32':57.16038,'Decimal':Decimal('1752.064'),'DateTime':datetime(2023,10,17,9,34,25,836042),'Date':date(2023,10,17),'Time':time(9,34,25,836042),'Year':2023,'UUID':UUID('550e8400-e29b-41d4-a716-446655440000'),'JSON_RawMessage':'{\"foo\":5,\"boo\":true}','JSON_bool':'true','JSON_string':'\"foo \x26 boo \\\\u\"','JSON_float64':'23.871','JSON_Number':'85802.7305','JSON_slice':'[\"foo\",3,true]','JSON_map':'{\"a\":1,\"b\":2}','JSON_nil':'null','Inet':'192.158.1.38','Text':'some text','Array':['foo','boo'],'Object':{'a':9,'b':False},'Map':{'a':1,'b':2,'c':3}}]`),
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
				{"a": `&`},
				{"a": `<`},
				{"a": "\u2028"},
				{"a": "\u2029"},
			},
			result: []byte(`[{'a':''},{'a':'\x27'},{'a':'\"'},{'a':'\x26'},{'a':'\x3c'},{'a':'\u2028'},{'a':'\u2029'}]`),
		},
	}
	var b []byte
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := MarshalPython(b, test.schema, test.values)
			if !bytes.Equal(test.result, got) {
				t.Fatalf("MarshalJavaScript: expected %s, got %s", string(test.result), string(got))
			}
		})
	}
}
