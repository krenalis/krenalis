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
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"chichi/apis/state"
	"chichi/connector/types"

	"github.com/shopspring/decimal"
	"golang.org/x/exp/maps"
)

func Test_Unmarshal(t *testing.T) {

	schema := types.Object([]types.Property{
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
			Type: types.Int8().WithIntRange(-20, 20),
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
					Name:     "a",
					Type:     types.Int(),
					Required: true,
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

	results := []Result{
		{
			Value: map[string]any{
				"Boolean":   true,
				"Int":       1307298102,
				"Int8":      -12,
				"Int16":     8023,
				"Int24":     -2880217,
				"Int64":     927041163082605,
				"UInt":      uint(1307298102),
				"UInt8":     uint(12),
				"UInt16":    uint(8023),
				"UInt24":    uint(2880217),
				"UInt64":    uint(927041163082605),
				"Float":     18372.36240184391,
				"Float32":   57.16038,
				"Decimal":   decimal.RequireFromString("1752.064"),
				"DateTime":  time.Date(2023, 10, 17, 9, 34, 25, 836540129, time.UTC),
				"Date":      time.Date(2023, 10, 17, 0, 0, 0, 0, time.UTC),
				"Time":      time.Date(1970, 01, 01, 9, 34, 25, 836540129, time.UTC),
				"Year":      2023,
				"UUID":      "550e8400-e29b-41d4-a716-446655440000",
				"JSON":      json.RawMessage(`{"foo":5,"boo":true}`),
				"JSON_null": json.RawMessage(`null`),
				"JSON_nil":  nil,
				"Inet":      "192.158.1.38",
				"Text":      "some text",
				"Text_nil":  nil,
				"Array":     []any{"foo", "boo"},
				"Object":    map[string]any{"a": 9, "b": false},
				"Map":       map[string]any{"a": 1, "b": 2, "c": 3},
			},
		},
	}

	jsTerms := javaScriptDecoderOptions.terms
	pyTerms := pythonDecoderOptions.terms

	tests := []struct {
		language     state.Language
		timeTruncate time.Duration
		data         string
		results      []Result
		err          error
	}{
		{
			language:     state.JavaScript,
			timeTruncate: time.Millisecond,
			data:         `[{"value":{"Boolean":true,"Int":1307298102,"Int8":-12,"Int16":8023,"Int24":-2880217,"Int64":"927041163082605","UInt":1307298102,"UInt8":12,"UInt16":8023,"UInt24":2880217,"UInt64":"927041163082605","Float":18372.36240184391,"Float32":57.16038,"Decimal":"1752.064","DateTime":"2023-10-17T09:34:25.836Z","Date":"2023-10-17T00:00:00.000Z","Time":"1970-01-01T09:34:25.836Z","Year":2023,"UUID":"550e8400-e29b-41d4-a716-446655440000","JSON":"{\"foo\":5,\"boo\":true}","JSON_null":"null","JSON_nil":null,"Inet":"192.158.1.38","Text":"some text","Text_nil":null,"Array":["foo","boo"],"Object":{"a":9,"b":false},"Map":{"a":1,"b":2,"c":3}}}]`,
			results:      results,
		},
		{
			language:     state.Python,
			timeTruncate: time.Microsecond,
			data:         `[{"value":{"Boolean":true,"Int":1307298102,"Int8":-12,"Int16":8023,"Int24":-2880217,"Int64":927041163082605,"UInt":1307298102,"UInt8":12,"UInt16":8023,"UInt24":2880217,"UInt64":927041163082605,"Float":18372.36240184391,"Float32":57.16038,"Decimal":"1752.064","DateTime":"2023-10-17 09:34:25.83654","Date":"2023-10-17","Time":"09:34:25.83654","Year":2023,"UUID":"550e8400-e29b-41d4-a716-446655440000","JSON":"{\"foo\":5,\"boo\":true}","JSON_null":"null","JSON_nil":null,"Inet":"192.158.1.38","Text":"some text","Text_nil":null,"Array":["foo","boo"],"Object":{"a":9,"b":false},"Map":{"a":1,"b":2,"c":3}}}]`,
			results:      results,
		},
		{
			language: state.JavaScript,
			data:     ``,
			err:      ErrSyntaxInvalid,
		},
		{
			language: state.Python,
			data:     `[]`,
			results:  []Result{},
		},
		{
			language: state.JavaScript,
			data:     `[5]`,
			err:      ErrSyntaxInvalid,
		},
		{
			language: state.JavaScript,
			data:     `[{"value":{"Boolean":}}]`,
			err:      ErrSyntaxInvalid,
		},
		{
			language: state.JavaScript,
			data:     `[{"value":{"Boolean":true`,
			err:      ErrSyntaxInvalid,
		},
		{
			language: state.Python,
			data:     `[{"value":{"Object":{"c":5}}}]`,
			results:  []Result{{Error: newErrPropertyNotExist("Object.c", pyTerms)}},
		},
		{
			language: state.Python,
			data:     `[{"value":{"Object":{"b":true}}}]`,
			results:  []Result{{Error: newErrMissingProperty("Object.a", pyTerms)}},
		},
		{
			language: state.JavaScript,
			data:     `[{"value":{"Object":{"b":3}}}]`,
			results:  []Result{{Error: newErrInvalidValue(`does not have a valid value: 3`, "Object.b", jsTerms)}},
		},
		{
			language: state.JavaScript,
			data:     `[{"value":{"Int8":21}}]`,
			results:  []Result{{Error: newErrInvalidValue(`is out of range [-20, 20]: 21`, "Int8", jsTerms)}},
		},
		{
			language: state.JavaScript,
			data:     `[{"value":{"Int8":-25}}]`,
			results:  []Result{{Error: newErrInvalidValue(`is out of range [-20, 20]: -25`, "Int8", jsTerms)}},
		},
		{
			language: state.Python,
			data:     `[{"value":{"Boolean":"a \" \\ b"}}]`,
			results:  []Result{{Error: newErrInvalidValue(`does not have a valid value: "a \" \\ b"`, "Boolean", pyTerms)}},
		},
		{
			language: state.Python,
			data:     `[{"value":{"Boolean":null}}]`,
			results:  []Result{{Error: newErrInvalidValue(`cannot be None`, "Boolean", pyTerms)}},
		},
		{
			language: state.Python,
			data:     `[{"value":{"Date":"2023-02-30"}}]`,
			results:  []Result{{Error: newErrInvalidValue(`does not have a valid value: "2023-02-30"`, "Date", pyTerms)}},
		},
		{
			language: state.Python,
			data:     `[{"value":{"Text":"some long text"}}]`,
			results:  []Result{{Error: newErrInvalidValue(`is longer than 10 characters: "some long text"`, "Text", pyTerms)}},
		},
		{
			language: state.Python,
			data:     `[{"value":{"Text_values":"c"}}]`,
			results:  []Result{{Value: map[string]any{"Text_values": "c"}}},
		},
		{
			language: state.Python,
			data:     `[{"value":{"Text_values":"foo"}}]`,
			results:  []Result{{Error: newErrInvalidValue(`has an invalid value: "foo"; valid values are "a", "b", and "c"`, "Text_values", pyTerms)}},
		},
		{
			language: state.Python,
			data:     `[{"value":{"Text_regexp":"foo"}}]`,
			results:  []Result{{Value: map[string]any{"Text_regexp": "foo"}}},
		},
		{
			language: state.Python,
			data:     `[{"value":{"Text_regexp":"faa"}}]`,
			results:  []Result{{Error: newErrInvalidValue(fmt.Sprintf(`has an invalid value: "faa"; it does not match the property's regular expression`), "Text_regexp", pyTerms)}},
		},
		{
			language: state.Python,
			data:     `[{"value":{}},{"value":{}}]`,
			results:  []Result{{Value: map[string]any{}}, {Value: map[string]any{}}},
		},
		{
			language: state.JavaScript,
			data:     `[{"value":{"Boolean":true}},{"value":{"Int":547}}]`,
			results:  []Result{{Value: map[string]any{"Boolean": true}}, {Value: map[string]any{"Int": 547}}},
		},
		{
			language: state.JavaScript,
			data:     `[{"value":{"foo":"boo"}},{"value":{"Int":547}}]`,
			results:  []Result{{Error: newErrPropertyNotExist("foo", jsTerms)}, {Value: map[string]any{"Int": 547}}},
		},
		{
			language: state.Python,
			data:     `[{"value":{"Object":{}}},{"value":{"Int":547}}]`,
			results:  []Result{{Error: newErrMissingProperty("Object.a", pyTerms)}, {Value: map[string]any{"Int": 547}}},
		},
		{
			language: state.JavaScript,
			data:     `[{"value":{"Boolean":3}},{"value":{"Int":547}}]`,
			results:  []Result{{Error: newErrInvalidValue(`does not have a valid value: 3`, "Boolean", jsTerms)}, {Value: map[string]any{"Int": 547}}},
		},
		{
			language: state.Python,
			data:     `[{"value":{"Boolean":3}},{"value":{"Object":{}}}]`,
			results:  []Result{{Error: newErrInvalidValue(`does not have a valid value: 3`, "Boolean", pyTerms)}, {Error: newErrMissingProperty("Object.a", pyTerms)}},
		},
	}

	for _, test := range tests {
		t.Run(test.language.String(), func(t *testing.T) {
			b := strings.NewReader(test.data)
			got, err := Unmarshal(b, schema, test.language)
			if err != nil {
				if test.err == nil {
					t.Fatalf("Unmarshal: expected no error, got error %s", err)
				}
				if !reflect.DeepEqual(test.err, err) {
					t.Fatalf("Unmarshal: expected error %q, got error %q", test.err, err)
				}
				if got != nil {
					t.Fatalf("Unmarshal: expected nil, got %#v", got)
				}
				return
			}
			if len(test.results) != len(got) {
				t.Fatalf("Unmarshal: expected %d results, got %d", len(test.results), len(got))
			}
			for i, result := range test.results {
				if result.Value == nil {
					if got[i].Value != nil {
						t.Fatalf("Unmarshal:\n\texpected nil value\n\tgot value %#v", got[i].Value)
					}
					if got[i].Error == nil {
						t.Fatalf("Unmarshal:\n\texpected error\n\tgot nil")
					}
					if !reflect.DeepEqual(result.Error, got[i].Error) {
						t.Fatalf("Unmarshal:\n\texpected error %#v\n\tgot error %#v", result.Error, got[i].Error)
					}
					continue
				}
				if got[i].Error != nil {
					t.Fatalf("Unmarshal:\n\texpected no error\n\tgot error %#v", got[i].Error)
				}
				if got[i].Value == nil {
					t.Fatalf("Unmarshal:\n\texpected value\n\tgot no value")
				}
				if err := equalValues(schema, test.timeTruncate, result.Value, got[i].Value); err != nil {
					t.Fatalf("Unmarshal:\n\texpected value %#v\n\tgot value      %#v\n\terror:   %s", result.Value, got[i].Value, err)
				}
			}
		})
	}

}

// equalValues reports whether v1 and v2 are equal according to the type t.
// v1 is supposed to conform to type t, and v2 is checked for equality with v1.
// If t is a DateTime, Date, or Time type, v1 is truncated to a multiple of
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
	switch t.PhysicalType() {
	case types.PtFloat32:
		f2, ok := v2.(float64)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		f1 := v1.(float64)
		if float32(f1) != float32(f2) {
			return fmt.Errorf("expected value %f, got %f", float32(f1), float32(f2))
		}
		return nil
	case types.PtDecimal:
		d2, ok := v2.(decimal.Decimal)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		d1 := v1.(decimal.Decimal)
		if d1.Cmp(d2) != 0 {
			return fmt.Errorf("expected value %s, got %s", v1, d2)
		}
		return nil
	case types.PtDateTime, types.PtDate, types.PtTime:
		t2, ok := v2.(time.Time)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		t1 := v1.(time.Time).Truncate(timeTruncate)
		if !t1.Equal(t2) {
			return fmt.Errorf("expected value %s, got %s", v1, t2)
		}
		return nil
	case types.PtJSON:
		j2, ok := v2.(json.RawMessage)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		j1 := v1.(json.RawMessage)
		if !bytes.Equal(j1, j2) {
			return fmt.Errorf("expected value %q (%T), got %q (%T)", string(j1), v1, string(j2), v2)
		}
		return nil
	case types.PtArray:
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
	case types.PtObject:
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
			keys := maps.Keys(unexpected)
			slices.Sort(keys)
			return fmt.Errorf("unexpected property %q", keys[0])
		}
		return nil
	case types.PtMap:
		m1 := v1.(map[string]any)
		m2, ok := v2.(map[string]any)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		names := maps.Keys(m2)
		slices.Sort(names)
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
