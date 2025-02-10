//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package transformers

import (
	"bytes"
	"fmt"
	"maps"
	"math"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

func Test_Unmarshal(t *testing.T) {

	schema := types.Object([]types.Property{
		{
			Name: "Boolean",
			Type: types.Boolean(),
		},
		{
			Name: "Int8",
			Type: types.Int(8).WithIntRange(-20, 20),
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
			Name: "Float64_Infinity",
			Type: types.Float(64),
		},
		{
			Name: "Float64_NegInfinity",
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
			Name: "JSON",
			Type: types.JSON(),
		},
		{
			Name: "JSON_null",
			Type: types.JSON(),
		},
		{
			Name:     "JSON_nil",
			Type:     types.JSON(),
			Nullable: true,
		},
		{
			Name: "Inet",
			Type: types.Inet(),
		},
		{
			Name: "Text",
			Type: types.Text().WithCharLen(10),
		},
		{
			Name: "Text_values",
			Type: types.Text().WithValues("a", "b", "c"),
		},
		{
			Name: "Text_regexp",
			Type: types.Text().WithRegexp(regexp.MustCompile(`oo$`)),
		},
		{
			Name:     "Text_nil",
			Type:     types.Text(),
			Nullable: true,
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
					Type: types.Boolean(),
				},
				{
					Name:           "b",
					Type:           types.Int(32),
					CreateRequired: true,
				},
				{
					Name:           "c",
					Type:           types.JSON(),
					UpdateRequired: true,
				},
				{
					Name:           "d",
					Type:           types.DateTime(),
					UpdateRequired: true,
					CreateRequired: true,
				},
			}),
		},
		{
			Name: "Map",
			Type: types.Map(types.Int(32)),
		},
	})

	records := []Record{
		{
			Properties: map[string]any{
				"Boolean":             true,
				"Int8":                -12,
				"Int16":               8023,
				"Int24":               -2880217,
				"Int32":               1307298102,
				"Int64":               927041163082605,
				"Uint8":               uint(12),
				"Uint16":              uint(8023),
				"Uint24":              uint(2880217),
				"Uint32":              uint(1307298102),
				"Uint64":              uint(927041163082605),
				"Float32":             float64(float32(57.16038)),
				"Float64":             18372.36240184391,
				"Float64_NaN":         math.NaN(),
				"Float64_Infinity":    math.Inf(1),
				"Float64_NegInfinity": math.Inf(-1),
				"Decimal":             decimal.MustParse("1752.064"),
				"DateTime":            time.Date(2023, 10, 17, 9, 34, 25, 836540129, time.UTC),
				"Date":                time.Date(2023, 10, 17, 0, 0, 0, 0, time.UTC),
				"Time":                time.Date(1970, 01, 01, 9, 34, 25, 836540129, time.UTC),
				"Year":                2023,
				"UUID":                "550e8400-e29b-41d4-a716-446655440000",
				"JSON":                json.Value(`{"foo": 5,"boo": true}`),
				"JSON_null":           json.Value(`null`),
				"Inet":                "192.158.1.38",
				"Text":                "some text",
				"Text_nil":            nil,
				"Array":               []any{"foo", "boo"},
				"Object":              map[string]any{"a": false, "b": 9},
				"Map":                 map[string]any{"a": 1, "b": 2, "c": 3},
			},
		},
	}

	jsTerms := javaScriptDecoderOptions.terms
	pyTerms := pythonDecoderOptions.terms

	tests := []struct {
		language     state.Language
		schema       types.Type
		preserveJSON bool
		timeTruncate time.Duration
		data         string
		records      []Record
		expected     []Record
		err          error
	}{
		{
			language:     state.JavaScript,
			schema:       schema,
			preserveJSON: false,
			timeTruncate: time.Millisecond,
			data:         `{"records":[{"value":{"Boolean":true,"Int8":-12,"Int16":8023,"Int24":-2880217,"Int32":1307298102,"Int64":"927041163082605","Uint8":12,"Uint16":8023,"Uint24":2880217,"Uint32":1307298102,"Uint64":"927041163082605","Float32":57.16038,"Float64":18372.36240184391,"Float64_NaN":"NaN","Float64_Infinity":"Infinity","Float64_NegInfinity":"-Infinity","Decimal":"1752.064","DateTime":"2023-10-17T09:34:25.836Z","Date":"2023-10-17T00:00:00.000Z","Time":"1970-01-01T09:34:25.836Z","Year":2023,"UUID":"550e8400-e29b-41d4-a716-446655440000","JSON":{"foo": 5,"boo": true},"JSON_null":null,"Inet":"192.158.1.38","Text":"some text","Text_nil":null,"Array":["foo","boo"],"Object":{"a":false,"b":9},"Map":{"a":1,"b":2,"c":3}}}]}`,
			records:      records,
		},
		{
			language:     state.JavaScript,
			schema:       schema,
			preserveJSON: true,
			timeTruncate: time.Millisecond,
			data:         `{"records":[{"value":{"Boolean":true,"Int8":-12,"Int16":8023,"Int24":-2880217,"Int32":1307298102,"Int64":"927041163082605","Uint8":12,"Uint16":8023,"Uint24":2880217,"Uint32":1307298102,"Uint64":"927041163082605","Float32":57.16038,"Float64":18372.36240184391,"Float64_NaN":"NaN","Float64_Infinity":"Infinity","Float64_NegInfinity":"-Infinity","Decimal":"1752.064","DateTime":"2023-10-17T09:34:25.836Z","Date":"2023-10-17T00:00:00.000Z","Time":"1970-01-01T09:34:25.836Z","Year":2023,"UUID":"550e8400-e29b-41d4-a716-446655440000","JSON":"{\"foo\": 5,\"boo\": true}","JSON_null":"null","Inet":"192.158.1.38","Text":"some text","Text_nil":null,"Array":["foo","boo"],"Object":{"a":false,"b":9},"Map":{"a":1,"b":2,"c":3}}}]}`,
			records:      records,
		},
		{
			language:     state.Python,
			schema:       schema,
			preserveJSON: false,
			timeTruncate: time.Microsecond,
			data:         `{"records":[{"value":{"Boolean":true,"Int8":-12,"Int16":8023,"Int24":-2880217,"Int32":1307298102,"Int64":927041163082605,"Uint8":12,"Uint16":8023,"Uint24":2880217,"Uint32":1307298102,"Uint64":927041163082605,"Float32":57.16038,"Float64":18372.36240184391,"Float64_NaN":"NaN","Float64_Infinity":"Infinity","Float64_NegInfinity":"-Infinity","Decimal":"1752.064","DateTime":"2023-10-17 09:34:25.83654","Date":"2023-10-17","Time":"09:34:25.83654","Year":2023,"UUID":"550e8400-e29b-41d4-a716-446655440000","JSON":{"foo": 5,"boo": true},"JSON_null":null,"Inet":"192.158.1.38","Text":"some text","Text_nil":null,"Array":["foo","boo"],"Object":{"a":false,"b":9},"Map":{"a":1,"b":2,"c":3}}}]}`,
			records:      records,
		},
		{
			language:     state.Python,
			schema:       schema,
			preserveJSON: true,
			timeTruncate: time.Microsecond,
			data:         `{"records":[{"value":{"Boolean":true,"Int8":-12,"Int16":8023,"Int24":-2880217,"Int32":1307298102,"Int64":927041163082605,"Uint8":12,"Uint16":8023,"Uint24":2880217,"Uint32":1307298102,"Uint64":927041163082605,"Float32":57.16038,"Float64":18372.36240184391,"Float64_NaN":"NaN","Float64_Infinity":"Infinity","Float64_NegInfinity":"-Infinity","Decimal":"1752.064","DateTime":"2023-10-17 09:34:25.83654","Date":"2023-10-17","Time":"09:34:25.83654","Year":2023,"UUID":"550e8400-e29b-41d4-a716-446655440000","JSON":"{\"foo\": 5,\"boo\": true}","JSON_null":"null","Inet":"192.158.1.38","Text":"some text","Text_nil":null,"Array":["foo","boo"],"Object":{"a":false,"b":9},"Map":{"a":1,"b":2,"c":3}}}]}`,
			records:      records,
		},
		{
			language:     state.JavaScript,
			schema:       schema,
			preserveJSON: true,
			data:         `{"records":[{"value":{"JSON_nil":null}}]}`,
			records:      []Record{{Properties: map[string]any{"JSON_nil": nil}}},
		},
		{
			language:     state.Python,
			schema:       schema,
			preserveJSON: true,
			data:         `{"records":[{"value":{"JSON_nil":null}}]}`,
			records:      []Record{{Properties: map[string]any{"JSON_nil": nil}}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     ``,
			err:      errSyntaxInvalid,
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{}`,
			err:      errSyntaxInvalid,
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `[{"value":"boo"}]`,
			err:      errSyntaxInvalid,
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[]}`,
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[],}`,
			err:      errSyntaxInvalid,
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[],"records":[]}`,
			err:      errSyntaxInvalid,
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[5]}`,
			err:      errSyntaxInvalid,
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"Boolean":}}]}`,
			records:  make([]Record, 1),
			err:      errSyntaxInvalid,
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"Boolean":true`,
			records:  make([]Record, 1),
			err:      errSyntaxInvalid,
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"error":true}`,
			err:      errSyntaxInvalid,
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"error":"syntax error","records":[{"value":4}]}`,
			err:      errSyntaxInvalid,
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Object":{"e":5}}}]}`,
			records:  []Record{{Err: newErrPropertyNotExist("Object.e", pyTerms)}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Object":{"a":true}}}]}`,
			records:  []Record{{Purpose: Create, Err: newErrMissingProperty("Object.b", pyTerms)}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Object":{"a":true}}}]}`,
			records:  []Record{{Purpose: Update, Err: newErrMissingProperty("Object.c", pyTerms)}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"Object":{"b":false}}}]}`,
			records:  []Record{{Err: newErrInvalidValue(`does not have a valid value: false`, "Object.b", jsTerms)}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"Int8":21}}]}`,
			records:  []Record{{Err: newErrInvalidValue(`is out of range [-20, 20]: 21`, "Int8", jsTerms)}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"Int8":-25}}]}`,
			records:  []Record{{Err: newErrInvalidValue(`is out of range [-20, 20]: -25`, "Int8", jsTerms)}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Boolean":"a \" \\ b"}}]}`,
			records:  []Record{{Err: newErrInvalidValue(`does not have a valid value: "a \" \\ b"`, "Boolean", pyTerms)}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Boolean":null}}]}`,
			records:  []Record{{Err: newErrInvalidValue(`cannot be None`, "Boolean", pyTerms)}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Date":"2023-02-30"}}]}`,
			records:  []Record{{Err: newErrInvalidValue(`does not have a valid value: "2023-02-30"`, "Date", pyTerms)}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Text":"some long text"}}]}`,
			records:  []Record{{Err: newErrInvalidValue(`is longer than 10 characters: "some long text"`, "Text", pyTerms)}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Text_values":"c"}}]}`,
			records:  []Record{{Properties: map[string]any{"Text_values": "c"}}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Text_values":"foo"}}]}`,
			records:  []Record{{Err: newErrInvalidValue(`has an invalid value: "foo"; valid values are "a", "b", and "c"`, "Text_values", pyTerms)}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Text_regexp":"foo"}}]}`,
			records:  []Record{{Properties: map[string]any{"Text_regexp": "foo"}}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Text_regexp":"faa"}}]}`,
			records:  []Record{{Err: newErrInvalidValue(`has an invalid value: "faa"; it does not match the property's regular expression`, "Text_regexp", pyTerms)}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{}},{"value":{}}]}`,
			records:  []Record{{Properties: map[string]any{}}, {Properties: map[string]any{}}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"Boolean":true}},{"value":{"Int32":547}}]}`,
			records:  []Record{{Properties: map[string]any{"Boolean": true}}, {Properties: map[string]any{"Int32": 547}}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"foo":"boo"}},{"value":{"Int32":547}}]}`,
			records:  []Record{{Err: newErrPropertyNotExist("foo", jsTerms)}, {Properties: map[string]any{"Int32": 547}}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Object":{}}},{"value":{"Int32":547}}]}`,
			records:  []Record{{Purpose: Update, Err: newErrMissingProperty("Object.c", pyTerms)}, {Properties: map[string]any{"Int32": 547}}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"Boolean":3}},{"value":{"Int32":547}}]}`,
			records:  []Record{{Err: newErrInvalidValue(`does not have a valid value: 3`, "Boolean", jsTerms)}, {Properties: map[string]any{"Int32": 547}}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Boolean":3}},{"value":{"Object":{}}}]}`,
			records:  []Record{{Err: newErrInvalidValue(`does not have a valid value: 3`, "Boolean", pyTerms)}, {Purpose: Create, Err: newErrMissingProperty("Object.b", pyTerms)}},
		},
		{
			language: state.JavaScript,
			schema:   types.Type{},
			data:     `{"records":[{"value":{}},{"value":{"foo":5}}]}`,
			records:  []Record{{Properties: map[string]any{}}, {Err: newErrPropertyNotExist("foo", jsTerms)}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"a.b.c":5}}]}`,
			records:  []Record{{Err: newErrPropertyNotExist("a.b.c", jsTerms)}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"error":"unexpected token ')'"}`,
			err:      FunctionExecutionError("unexpected token ')'"),
		},
	}

	for _, test := range tests {
		t.Run(test.language.String(), func(t *testing.T) {
			b := strings.NewReader(test.data)
			records := make([]Record, len(test.records))
			for i, record := range test.records {
				records[i].Purpose = record.Purpose
			}
			err := Unmarshal(b, records, test.schema, test.language, test.preserveJSON)
			if err != nil {
				if test.err == nil {
					t.Fatalf("Unmarshal: expected no error, got error %s", err)
				}
				if !reflect.DeepEqual(test.err, err) {
					t.Fatalf("Unmarshal: expected error %q, got error %q", test.err, err)
				}
				return
			}
			if test.err != nil {
				t.Fatalf("Unmarshal: expected error %q, got no error", test.err)
			}
			for i, want := range test.records {
				got := records[i]
				if got.Err != nil {
					if want.Err == nil {
						t.Fatalf("Unmarshal:\n\texpected no error\n\tgot error %q", got.Err.Error())
					}
					if !reflect.DeepEqual(want.Err, got.Err) {
						t.Fatalf("Unmarshal:\n\texpected error %q\n\tgot error %q", want.Err.Error(), got.Err.Error())
					}
					continue
				}
				if want.Err != nil {
					t.Fatalf("Unmarshal:\n\texpected error %q\n\tgot no error", want.Err)
				}
				if got.Properties == nil {
					t.Fatalf("Unmarshal:\n\texpected properties\n\tgot no properties")
				}
				if err := equalValues(schema, test.timeTruncate, want.Properties, got.Properties); err != nil {
					t.Fatalf("Unmarshal:\n\texpected properties %#v\n\tgot properties      %#v\n\terror:   %s", want.Properties, got.Properties, err)
				}
			}
		})
	}

}

// equalValues reports whether v1 and v2 are equal according to the type t.
// v1 is supposed to conform to type t, and v2 is checked for equality with v1.
// If t is a datetime, date, or time type, v1 is truncated to a multiple of
// timeTruncate.
func equalValues(t types.Type, timeTruncate time.Duration, v1, v2 any) error {
	if v1 == nil {
		if v2 != nil {
			return fmt.Errorf("expected nil, got %#v (%T)", v2, v2)
		}
		return nil
	} else if v2 == nil {
		return fmt.Errorf("expected %#v (%T), got nil", v1, v1)
	}
	switch t.Kind() {
	case types.FloatKind:
		f2, ok := v2.(float64)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		f1 := v1.(float64)
		switch {
		case math.IsNaN(f1):
			if !math.IsNaN(f2) {
				if t.BitSize() == 32 {
					return fmt.Errorf("expected value NaN, got %f", float32(f2))
				}
				return fmt.Errorf("expected value NaN, got %f", f2)
			}
		case math.IsInf(f1, 1):
			if !math.IsInf(f2, 1) {
				if t.BitSize() == 32 {
					return fmt.Errorf("expected value +Inf, got %f", float32(f2))
				}
				return fmt.Errorf("expected value +Inf, got %f", f2)
			}
		case math.IsInf(f1, -1):
			if !math.IsInf(f2, -1) {
				if t.BitSize() == 32 {
					return fmt.Errorf("expected value -Inf, got %f", float32(f2))
				}
				return fmt.Errorf("expected value -Inf, got %f", f2)
			}
		case t.BitSize() == 32:
			if float32(f1) != float32(f2) {
				return fmt.Errorf("expected value %f, got %f", float32(f1), float32(f2))
			}
		default:
			if f1 != f2 {
				return fmt.Errorf("expected value %f, got %f", f1, f2)
			}
		}
		return nil
	case types.DecimalKind:
		d2, ok := v2.(decimal.Decimal)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		d1 := v1.(decimal.Decimal)
		if d1.Cmp(d2) != 0 {
			return fmt.Errorf("expected value %s, got %s", v1, d2)
		}
		return nil
	case types.DateTimeKind, types.DateKind, types.TimeKind:
		t2, ok := v2.(time.Time)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		t1 := v1.(time.Time).Truncate(timeTruncate)
		if !t1.Equal(t2) {
			return fmt.Errorf("expected value %s, got %s", v1, t2)
		}
		return nil
	case types.JSONKind:
		j2, ok := v2.(json.Value)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		j1 := v1.(json.Value)
		if !bytes.Equal(j1, j2) {
			return fmt.Errorf("expected value %q (%T), got %q (%T)", string(j1), v1, string(j2), v2)
		}
		return nil
	case types.ArrayKind:
		a1 := v1.([]any)
		a2, ok := v2.([]any)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		for i, e1 := range a1 {
			err := equalValues(t.Elem(), timeTruncate, e1, a2[i])
			if err != nil {
				return err
			}
		}
		return nil
	case types.ObjectKind:
		o1 := v1.(map[string]any)
		o2, ok := v2.(map[string]any)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		unexpected := maps.Clone(o2)
		for _, p := range t.Properties() {
			s1, ok := o1[p.Name]
			if !ok {
				_, ok := o2[p.Name]
				if ok {
					return fmt.Errorf("not expected property %s, got property", p.Name)
				}
				continue
			}
			s2, ok := o2[p.Name]
			if !ok {
				return fmt.Errorf("expected property %s, got no property", p.Name)
			}
			err := equalValues(p.Type, timeTruncate, s1, s2)
			if err != nil {
				return err
			}
			delete(unexpected, p.Name)
		}
		if len(unexpected) > 0 {
			keys := slices.Sorted(maps.Keys(unexpected))
			return fmt.Errorf("unexpected property %q", keys[0])
		}
		return nil
	case types.MapKind:
		m1 := v1.(map[string]any)
		m2, ok := v2.(map[string]any)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		names := slices.Sorted(maps.Keys(m2))
		if len(m1) != len(m2) {
			for _, name := range names {
				if _, ok := m1[name]; !ok {
					return fmt.Errorf("unexpected property %q", name)
				}
			}
		}
		for _, name := range names {
			e1, ok := m1[name]
			if !ok {
				return fmt.Errorf("unexpected property %q", name)
			}
			e2 := m2[name]
			err := equalValues(t.Elem(), timeTruncate, e1, e2)
			if err != nil {
				return err
			}
		}
		return nil
	}
	if v1 != v2 {
		return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
	}
	return nil
}
