//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package transformers

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/meergo/meergo/core/decimal"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

var schema = types.Object([]types.Property{
	{
		Name:         "Text",
		Type:         types.Text(),
		ReadOptional: true,
	},
	{
		Name:         "Boolean",
		Type:         types.Boolean(),
		ReadOptional: true,
	},
	{
		Name:         "Int8",
		Type:         types.Int(8),
		ReadOptional: true,
	},
	{
		Name:         "Int16",
		Type:         types.Int(16),
		ReadOptional: true,
	},
	{
		Name:         "Int24",
		Type:         types.Int(24),
		ReadOptional: true,
	},
	{
		Name:         "Int32",
		Type:         types.Int(32),
		ReadOptional: true,
	},
	{
		Name:         "Int64",
		Type:         types.Int(64),
		ReadOptional: true,
	},
	{
		Name:         "Uint8",
		Type:         types.Uint(8),
		ReadOptional: true,
	},
	{
		Name:         "Uint16",
		Type:         types.Uint(16),
		ReadOptional: true,
	},
	{
		Name:         "Uint24",
		Type:         types.Uint(24),
		ReadOptional: true,
	},
	{
		Name:         "Uint32",
		Type:         types.Uint(32),
		ReadOptional: true,
	},
	{
		Name:         "Uint64",
		Type:         types.Uint(64),
		ReadOptional: true,
	},
	{
		Name:         "Float32",
		Type:         types.Float(32),
		ReadOptional: true,
	},
	{
		Name:         "Float64",
		Type:         types.Float(64),
		ReadOptional: true,
	},
	{
		Name:         "Decimal",
		Type:         types.Decimal(10, 3),
		ReadOptional: true,
	},
	{
		Name:         "DateTime",
		Type:         types.DateTime(),
		ReadOptional: true,
	},
	{
		Name:         "Date",
		Type:         types.Date(),
		ReadOptional: true,
	},
	{
		Name:         "Time",
		Type:         types.Time(),
		ReadOptional: true,
	},
	{
		Name:         "Year",
		Type:         types.Year(),
		ReadOptional: true,
	},
	{
		Name:         "UUID",
		Type:         types.UUID(),
		ReadOptional: true,
	},
	{
		Name:         "JSON",
		Type:         types.JSON(),
		ReadOptional: true,
	},
	{
		Name:         "JSON_null",
		Type:         types.JSON(),
		ReadOptional: true,
	},
	{
		Name:         "Inet",
		Type:         types.Inet(),
		ReadOptional: true,
	},
	{
		Name:         "Array",
		Type:         types.Array(types.Text()),
		ReadOptional: true,
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
		ReadOptional: true,
	},
	{
		Name:         "Map",
		Type:         types.Map(types.Int(32)),
		ReadOptional: true,
	},
	{
		Name:         "MapArray",
		Type:         types.Map(types.Array(types.Text())),
		ReadOptional: true,
	},
})

var records = []Record{{Properties: map[string]any{
	"Text":      "some text",
	"Boolean":   true,
	"Int8":      -12,
	"Int16":     8023,
	"Int24":     -2880217,
	"Int32":     1307298102,
	"Int64":     927041163082605,
	"Uint8":     uint(12),
	"Uint16":    uint(8023),
	"Uint24":    uint(2880217),
	"Uint32":    uint(1307298102),
	"Uint64":    uint(927041163082605),
	"Float32":   float64(float32(57.16038)),
	"Float64":   18372.36240184391,
	"Decimal":   decimal.MustParse("1752.064"),
	"DateTime":  time.Date(2023, 10, 17, 9, 34, 25, 836042841, time.UTC),
	"Date":      time.Date(2023, 10, 17, 0, 0, 0, 0, time.UTC),
	"Time":      time.Date(1970, 01, 01, 9, 34, 25, 836042841, time.UTC),
	"Year":      2023,
	"UUID":      "550e8400-e29b-41d4-a716-446655440000",
	"JSON":      json.Value(`{"foo":true,"boo":[5,8]}`),
	"JSON_null": json.Value(`null`),
	"Inet":      "192.158.1.38",
	"Array":     []any{"foo", "boo"},
	"Object":    map[string]any{"a": 9, "b": false},
	"Map":       map[string]any{},
},
}}

// Test_MarshalAppend checks that Marshal appends to the provided buffer.
func Test_MarshalAppend(t *testing.T) {
	schema := types.Object([]types.Property{{Name: "a", Type: types.Int(32)}})
	record := []Record{{Properties: map[string]any{"a": 5}}}
	b := []byte("pre")
	got, err := Marshal(b, schema, record, state.JavaScript, false)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !bytes.HasPrefix(got, b) {
		t.Fatalf("expected prefix %q, got %q", string(b), string(got))
	}
	wantSuffix := "[{a:5}]"
	if !bytes.HasSuffix(got, []byte(wantSuffix)) {
		t.Fatalf("expected suffix %q, got %q", wantSuffix, string(got))
	}
}

// Test_MarshalErrors verifies that Marshal returns errors for invalid input.
func Test_MarshalErrors(t *testing.T) {
	schema := types.Object([]types.Property{{Name: "a", Type: types.Int(32)}})
	record := []Record{{Properties: map[string]any{"a": 1}}}

	t.Run("invalid language", func(t *testing.T) {
		_, err := Marshal(nil, schema, record, state.Language(42), false)
		if err == nil || err.Error() != "core/transformers: language is not valid" {
			t.Fatalf("expected language error, got %v", err)
		}
	})

	t.Run("schema not object", func(t *testing.T) {
		_, err := Marshal(nil, types.Text(), record, state.JavaScript, false)
		if err == nil || err.Error() != "core/transformers: schema is not an object" {
			t.Fatalf("expected schema error, got %v", err)
		}
	})
}

var mapValue = map[string]any{"Map": map[string]any{"a": 1, "b": 2, "c": 3}}
var mapArrayValue = map[string]any{"MapArray": map[string]any{"x": []any{"boo", "foo"}, "y": []any{}}}

func Test_MarshalJavaScript(t *testing.T) {
	tests := []struct {
		name         string
		schema       types.Type
		preserveJSON bool
		records      []Record
		result       []byte
		results      [][]byte
		err          error
	}{
		{
			name:         "Types",
			schema:       schema,
			preserveJSON: false,
			records:      records,
			results: [][]byte{
				[]byte(`[{Text:'some text',Boolean:true,Int8:-12,Int16:8023,Int24:-2880217,Int32:1307298102,Int64:927041163082605n,Uint8:12,Uint16:8023,Uint24:2880217,Uint32:1307298102,Uint64:927041163082605n,Float32:57.16038,Float64:18372.36240184391,Decimal:'1752.064',DateTime:new Date(1697535265836),Date:new Date(1697500800000),Time:new Date(34465836),Year:2023,UUID:'550e8400-e29b-41d4-a716-446655440000',JSON:{'foo':true,'boo':[5,8]},JSON_null:null,Inet:'192.158.1.38',Array:['foo','boo'],Object:{a:9,b:false},Map:{}}]`),
				[]byte(`[{Text:'some text',Boolean:true,Int8:-12,Int16:8023,Int24:-2880217,Int32:1307298102,Int64:927041163082605n,Uint8:12,Uint16:8023,Uint24:2880217,Uint32:1307298102,Uint64:927041163082605n,Float32:57.16038,Float64:18372.36240184391,Decimal:'1752.064',DateTime:new Date(1697535265836),Date:new Date(1697500800000),Time:new Date(34465836),Year:2023,UUID:'550e8400-e29b-41d4-a716-446655440000',JSON:{'boo':[5,3],'foo':true},JSON_null:null,Inet:'192.158.1.38',Array:['foo','boo'],Object:{a:9,b:false},Map:{}}]`),
				[]byte(`[{Text:'some text',Boolean:true,Int8:-12,Int16:8023,Int24:-2880217,Int32:1307298102,Int64:927041163082605n,Uint8:12,Uint16:8023,Uint24:2880217,Uint32:1307298102,Uint64:927041163082605n,Float32:57.16038,Float64:18372.36240184391,Decimal:'1752.064',DateTime:new Date(1697535265836),Date:new Date(1697500800000),Time:new Date(34465836),Year:2023,UUID:'550e8400-e29b-41d4-a716-446655440000',JSON:{'foo':true,'boo':[5,8]},JSON_null:null,Inet:'192.158.1.38',Array:['foo','boo'],Object:{a:9,b:false},Map:{}}]`),
				[]byte(`[{Text:'some text',Boolean:true,Int8:-12,Int16:8023,Int24:-2880217,Int32:1307298102,Int64:927041163082605n,Uint8:12,Uint16:8023,Uint24:2880217,Uint32:1307298102,Uint64:927041163082605n,Float32:57.16038,Float64:18372.36240184391,Decimal:'1752.064',DateTime:new Date(1697535265836),Date:new Date(1697500800000),Time:new Date(34465836),Year:2023,UUID:'550e8400-e29b-41d4-a716-446655440000',JSON:{'boo':[5,8],'foo':true},JSON_null:null,Inet:'192.158.1.38',Array:['foo','boo'],Object:{a:9,b:false},Map:{}}]`),
			},
		},
		{
			name:         "Types",
			schema:       schema,
			preserveJSON: true,
			records:      records,
			result:       []byte(`[{Text:'some text',Boolean:true,Int8:-12,Int16:8023,Int24:-2880217,Int32:1307298102,Int64:927041163082605n,Uint8:12,Uint16:8023,Uint24:2880217,Uint32:1307298102,Uint64:927041163082605n,Float32:57.16038,Float64:18372.36240184391,Decimal:'1752.064',DateTime:new Date(1697535265836),Date:new Date(1697500800000),Time:new Date(34465836),Year:2023,UUID:'550e8400-e29b-41d4-a716-446655440000',JSON:'{\"foo\":true,\"boo\":[5,8]}',JSON_null:'null',Inet:'192.158.1.38',Array:['foo','boo'],Object:{a:9,b:false},Map:{}}]`),
		},
		{
			name:    "Map",
			schema:  schema,
			records: []Record{{Properties: mapValue}},
			results: [][]byte{
				[]byte(`[{Map:{'a':1,'b':2,'c':3}}]`),
				[]byte(`[{Map:{'a':1,'c':3,'b':1}}]`),
				[]byte(`[{Map:{'b':2,'a':1,'c':3}}]`),
				[]byte(`[{Map:{'b':2,'c':3,'a':1}}]`),
				[]byte(`[{Map:{'c':3,'a':1,'b':2}}]`),
				[]byte(`[{Map:{'c':3,'b':2,'a':1}]]`),
			},
		},
		{
			name:    "MapArray",
			schema:  schema,
			records: []Record{{Properties: mapArrayValue}},
			results: [][]byte{
				[]byte(`[{MapArray:{'x':['boo','foo'],'y':[]}}]`),
				[]byte(`[{MapArray:{'y':[],'x':['boo','foo']}}]`),
			},
		},
		{
			name: "Empty values",
			records: []Record{
				{Properties: map[string]any{}},
				{Properties: map[string]any{}},
				{Properties: map[string]any{}},
			},
			result: []byte(`[{},{},{}]`),
		},
		{
			name:   "Invalid schema",
			schema: types.Type{},
			records: []Record{
				{Properties: map[string]any{"foo": 4}},
				{Properties: map[string]any{}},
				{Properties: map[string]any{"boo": true}},
			},
			result: []byte(`[{},{},{}]`),
		},
		{
			name: "Text encoding",
			schema: types.Object([]types.Property{
				{
					Name: "a",
					Type: types.Text(),
				},
			}),
			records: []Record{
				{Properties: map[string]any{"a": ``}},
				{Properties: map[string]any{"a": `'`}},
				{Properties: map[string]any{"a": `"`}},
				{Properties: map[string]any{"a": `&`}},
				{Properties: map[string]any{"a": `<`}},
				{Properties: map[string]any{"a": "\u2028"}},
				{Properties: map[string]any{"a": "\u2029"}},
			},
			result: []byte(`[{a:''},{a:'\u0027'},{a:'\"'},{a:'\u0026'},{a:'\u003c'},{a:'\u2028'},{a:'\u2029'}]`),
		},
		{
			name: "Nullable property",
			schema: types.Object([]types.Property{
				{Name: "a", Type: types.JSON(), Nullable: true},
				{Name: "b", Type: types.Text(), Nullable: true},
			}),
			records: []Record{
				{Properties: map[string]any{"a": nil, "b": nil}},
			},
			result: []byte(`[{a:null,b:null}]`),
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
			records: []Record{
				{Properties: map[string]any{}},
				{Properties: map[string]any{"a": "foo"}},
				{Properties: map[string]any{"a": "foo", "b": nil}},
				{Properties: map[string]any{"a": "foo", "b": map[string]any{"y": 45}}},
				{Properties: map[string]any{"a": "foo", "b": map[string]any{"x": 12, "y": 45}}},
			},
			result: []byte(`[{},{a:'foo'},{a:'foo',b:null},{a:'foo',b:{y:45}},{a:'foo',b:{x:12,y:45}}]`),
		},
		{
			name: "JSON null - not preserve",
			schema: types.Object([]types.Property{
				{Name: "a", Type: types.JSON(), Nullable: false},
				{Name: "b", Type: types.JSON(), Nullable: true},
				{Name: "c", Type: types.JSON(), Nullable: true},
			}),
			preserveJSON: false,
			records: []Record{{Properties: map[string]any{
				"a": json.Value("null"),
				"b": nil,
				"c": json.Value("null"),
			}}},
			result: []byte(`[{a:null,b:null,c:null}]`),
		},
		{
			name: "JSON null - preserve",
			schema: types.Object([]types.Property{
				{Name: "a", Type: types.JSON(), Nullable: false},
				{Name: "b", Type: types.JSON(), Nullable: true},
				{Name: "c", Type: types.JSON(), Nullable: true},
			}),
			preserveJSON: true,
			records: []Record{{Properties: map[string]any{
				"a": json.Value("null"),
				"b": nil,
				"c": json.Value("null"),
			}}},
			result: []byte(`[{a:'null',b:null,c:'null'}]`),
		},
		{
			name: "Spurious properties",
			schema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
			}),
			records: []Record{{Properties: map[string]any{
				"a": "foo",
				"b": "boo",
				"c": 24,
			}}},
			result: []byte(`[{a:'foo'}]`),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Marshal(nil, test.schema, test.records, state.JavaScript, test.preserveJSON)
			if err != nil {
				if test.err == nil {
					t.Fatalf("Marshal JavaScript: expected no error, got error %s", err)
				}
				if !reflect.DeepEqual(test.err, err) {
					t.Fatalf("Marshal JavaScript: expected error %q, got error %q", test.err, err)
				}
				if got != nil {
					t.Fatalf("Marshal JavaScript: expected nil, got %#v", got)
				}
				return
			}
			if test.err != nil {
				t.Fatalf("Marshal JavaScript: expected error %q, got no error", test.err)
			}
			if test.result != nil {
				if !bytes.Equal(test.result, got) {
					t.Fatalf("Marshal JavaScript: expected %s, got %s", string(test.result), string(got))
				}
				return
			}
			for _, result := range test.results {
				if bytes.Equal(result, got) {
					return
				}
			}
			t.Fatalf("Marshal JavaScript: expected %s (ignoring key order), got %s", string(test.results[0]), string(got))
		})
	}
}

func Test_MarshalPython(t *testing.T) {
	tests := []struct {
		name         string
		schema       types.Type
		preserveJSON bool
		records      []Record
		result       []byte
		results      [][]byte
		err          error
	}{
		{
			name:         "Types - not preserve JSON",
			schema:       schema,
			preserveJSON: false,
			records:      records,
			results: [][]byte{
				[]byte(`[{'Text':'some text','Boolean':True,'Int8':-12,'Int16':8023,'Int24':-2880217,'Int32':1307298102,'Int64':927041163082605,'Uint8':12,'Uint16':8023,'Uint24':2880217,'Uint32':1307298102,'Uint64':927041163082605,'Float32':57.16038,'Float64':18372.36240184391,'Decimal':Decimal('1752.064'),'DateTime':datetime(2023,10,17,9,34,25,836042),'Date':date(2023,10,17),'Time':time(9,34,25,836042),'Year':2023,'UUID':UUID('550e8400-e29b-41d4-a716-446655440000'),'JSON':{'foo':True,'boo':[5,8]},'JSON_null':None,'Inet':'192.158.1.38','Array':['foo','boo'],'Object':{'a':9,'b':False},'Map':{}}]`),
				[]byte(`[{'Text':'some text','Boolean':True,'Int8':-12,'Int16':8023,'Int24':-2880217,'Int32':1307298102,'Int64':927041163082605,'Uint8':12,'Uint16':8023,'Uint24':2880217,'Uint32':1307298102,'Uint64':927041163082605,'Float32':57.16038,'Float64':18372.36240184391,'Decimal':Decimal('1752.064'),'DateTime':datetime(2023,10,17,9,34,25,836042),'Date':date(2023,10,17),'Time':time(9,34,25,836042),'Year':2023,'UUID':UUID('550e8400-e29b-41d4-a716-446655440000'),'JSON':{'boo':[5,8],'foo':True},'JSON_null':None,'Inet':'192.158.1.38','Array':['foo','boo'],'Object':{'a':9,'b':False},'Map':{}}]`),
				[]byte(`[{'Text':'some text','Boolean':True,'Int8':-12,'Int16':8023,'Int24':-2880217,'Int32':1307298102,'Int64':927041163082605,'Uint8':12,'Uint16':8023,'Uint24':2880217,'Uint32':1307298102,'Uint64':927041163082605,'Float32':57.16038,'Float64':18372.36240184391,'Decimal':Decimal('1752.064'),'DateTime':datetime(2023,10,17,9,34,25,836042),'Date':date(2023,10,17),'Time':time(9,34,25,836042),'Year':2023,'UUID':UUID('550e8400-e29b-41d4-a716-446655440000'),'JSON':{'foo':True,'boo':[5,8]},'JSON_null':None,'Inet':'192.158.1.38','Array':['foo','boo'],'Object':{'a':9,'b':False},'Map':{}}]`),
				[]byte(`[{'Text':'some text','Boolean':True,'Int8':-12,'Int16':8023,'Int24':-2880217,'Int32':1307298102,'Int64':927041163082605,'Uint8':12,'Uint16':8023,'Uint24':2880217,'Uint32':1307298102,'Uint64':927041163082605,'Float32':57.16038,'Float64':18372.36240184391,'Decimal':Decimal('1752.064'),'DateTime':datetime(2023,10,17,9,34,25,836042),'Date':date(2023,10,17),'Time':time(9,34,25,836042),'Year':2023,'UUID':UUID('550e8400-e29b-41d4-a716-446655440000'),'JSON':{'boo':[5,8],'foo':True},'JSON_null':None,'Inet':'192.158.1.38','Array':['foo','boo'],'Object':{'a':9,'b':False},'Map':{}}]`),
			},
		},
		{
			name:         "Types - preserve JSON",
			schema:       schema,
			preserveJSON: true,
			records:      records,
			result:       []byte(`[{'Text':'some text','Boolean':True,'Int8':-12,'Int16':8023,'Int24':-2880217,'Int32':1307298102,'Int64':927041163082605,'Uint8':12,'Uint16':8023,'Uint24':2880217,'Uint32':1307298102,'Uint64':927041163082605,'Float32':57.16038,'Float64':18372.36240184391,'Decimal':Decimal('1752.064'),'DateTime':datetime(2023,10,17,9,34,25,836042),'Date':date(2023,10,17),'Time':time(9,34,25,836042),'Year':2023,'UUID':UUID('550e8400-e29b-41d4-a716-446655440000'),'JSON':'{\"foo\":true,\"boo\":[5,8]}','JSON_null':'null','Inet':'192.158.1.38','Array':['foo','boo'],'Object':{'a':9,'b':False},'Map':{}}]`),
		},
		{
			name:    "Map",
			schema:  schema,
			records: []Record{{Properties: mapValue}},
			results: [][]byte{
				[]byte(`[{'Map':{'a':1,'b':2,'c':3}}]`),
				[]byte(`[{'Map':{'a':1,'c':3,'b':1}}]`),
				[]byte(`[{'Map':{'b':2,'a':1,'c':3}}]`),
				[]byte(`[{'Map':{'b':2,'c':3,'a':1}}]`),
				[]byte(`[{'Map':{'c':3,'a':1,'b':2}}]`),
				[]byte(`[{'Map':{'c':3,'b':2,'a':1}]]`),
			},
		},
		{
			name:    "MapArray",
			schema:  schema,
			records: []Record{{Properties: mapArrayValue}},
			results: [][]byte{
				[]byte(`[{'MapArray':{'x':['boo','foo'],'y':[]}}]`),
				[]byte(`[{'MapArray':{'y':[],'x':['boo','foo']}}]`),
			},
		},
		{
			name: "Empty values",
			records: []Record{
				{Properties: map[string]any{}},
				{Properties: map[string]any{}},
				{Properties: map[string]any{}},
			},
			result: []byte(`[{},{},{}]`),
		},
		{
			name:   "Invalid schema",
			schema: types.Type{},
			records: []Record{
				{Properties: map[string]any{"foo": 4}},
				{Properties: map[string]any{}},
				{Properties: map[string]any{"boo": true}},
			},
			result: []byte(`[{},{},{}]`),
		},
		{
			name: "Text encoding",
			schema: types.Object([]types.Property{
				{
					Name: "a",
					Type: types.Text(),
				},
			}),
			records: []Record{
				{Properties: map[string]any{"a": ``}},
				{Properties: map[string]any{"a": `'`}},
				{Properties: map[string]any{"a": `"`}},
				{Properties: map[string]any{"a": `&`}},
				{Properties: map[string]any{"a": `<`}},
				{Properties: map[string]any{"a": "\u2028"}},
				{Properties: map[string]any{"a": "\u2029"}},
			},
			result: []byte(`[{'a':''},{'a':'\x27'},{'a':'\"'},{'a':'\x26'},{'a':'\x3c'},{'a':'\u2028'},{'a':'\u2029'}]`),
		},
		{
			name: "Nullable property",
			schema: types.Object([]types.Property{
				{Name: "a", Type: types.JSON(), Nullable: true},
				{Name: "b", Type: types.Text(), Nullable: true},
			}),
			records: []Record{
				{Properties: map[string]any{"a": nil, "b": nil}},
			},
			result: []byte(`[{'a':None,'b':None}]`),
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
			records: []Record{
				{Properties: map[string]any{}},
				{Properties: map[string]any{"a": "foo"}},
				{Properties: map[string]any{"a": "foo", "b": nil}},
				{Properties: map[string]any{"a": "foo", "b": map[string]any{"y": 45}}},
				{Properties: map[string]any{"a": "foo", "b": map[string]any{"x": 12, "y": 45}}},
			},
			result: []byte(`[{},{'a':'foo'},{'a':'foo','b':None},{'a':'foo','b':{'y':45}},{'a':'foo','b':{'x':12,'y':45}}]`),
		},
		{
			name: "JSON null - not preserve",
			schema: types.Object([]types.Property{
				{Name: "a", Type: types.JSON(), Nullable: false},
				{Name: "b", Type: types.JSON(), Nullable: true},
				{Name: "c", Type: types.JSON(), Nullable: true},
			}),
			preserveJSON: false,
			records: []Record{{Properties: map[string]any{
				"a": json.Value("null"),
				"b": nil,
				"c": json.Value("null"),
			}}},
			result: []byte(`[{'a':None,'b':None,'c':None}]`),
		},
		{
			name: "JSON null - preserve",
			schema: types.Object([]types.Property{
				{Name: "a", Type: types.JSON(), Nullable: false},
				{Name: "b", Type: types.JSON(), Nullable: true},
				{Name: "c", Type: types.JSON(), Nullable: true},
			}),
			preserveJSON: true,
			records: []Record{{Properties: map[string]any{
				"a": json.Value("null"),
				"b": nil,
				"c": json.Value("null"),
			}}},
			result: []byte(`[{'a':'null','b':None,'c':'null'}]`),
		},
		{
			name: "Spurious properties",
			schema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
			}),
			records: []Record{{Properties: map[string]any{
				"a": "foo",
				"b": "boo",
				"c": 24,
			}}},
			result: []byte(`[{'a':'foo'}]`),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Marshal(nil, test.schema, test.records, state.Python, test.preserveJSON)
			if err != nil {
				if test.err == nil {
					t.Fatalf("Marshal Python: expected no error, got error %s", err)
				}
				if !reflect.DeepEqual(test.err, err) {
					t.Fatalf("Marshal Python: expected error %q, got error %q", test.err, err)
				}
				if got != nil {
					t.Fatalf("Marshal Python: expected nil, got %#v", got)
				}
				return
			}
			if test.err != nil {
				t.Fatalf("Marshal Python: expected error %q, got no error", test.err)
			}
			if test.result != nil {
				if !bytes.Equal(test.result, got) {
					t.Fatalf("Marshal Python: expected %s, got %s", string(test.result), string(got))
				}
				return
			}
			for _, result := range test.results {
				if bytes.Equal(result, got) {
					return
				}
			}
			t.Fatalf("Marshal Python: expected %s (ignoring key order), got %s", string(test.results[0]), string(got))
		})
	}
}
