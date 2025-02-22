//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package types

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
)

func Test_Decode(t *testing.T) {

	object := Object([]Property{
		{
			Name: "Boolean",
			Type: Boolean(),
		},
		{
			Name: "Int8",
			Type: Int(8).WithIntRange(-20, 20),
		},
		{
			Name: "Int16",
			Type: Int(16),
		},
		{
			Name: "Int24",
			Type: Int(24),
		},
		{
			Name: "Int32",
			Type: Int(32),
		},
		{
			Name: "Int64",
			Type: Int(64),
		},
		{
			Name: "Uint8",
			Type: Uint(8),
		},
		{
			Name: "Uint16",
			Type: Uint(16),
		},
		{
			Name: "Uint24",
			Type: Uint(24),
		},
		{
			Name: "Uint32",
			Type: Uint(32),
		},
		{
			Name: "Uint64",
			Type: Uint(64),
		},
		{
			Name: "Float32",
			Type: Float(32),
		},
		{
			Name: "Float64",
			Type: Float(64),
		},
		{
			Name: "Decimal",
			Type: Decimal(10, 3),
		},
		{
			Name: "DateTime",
			Type: DateTime(),
		},
		{
			Name: "Date",
			Type: Date(),
		},
		{
			Name: "Time",
			Type: Time(),
		},
		{
			Name: "Year",
			Type: Year(),
		},
		{
			Name: "UUID",
			Type: UUID(),
		},
		{
			Name: "JSON",
			Type: JSON(),
		},
		{
			Name: "JSON_null",
			Type: JSON(),
		},
		{
			Name: "Inet",
			Type: Inet(),
		},
		{
			Name: "Text",
			Type: Text().WithCharLen(10),
		},
		{
			Name:     "Text_values",
			Type:     Text().WithValues("a", "b", "c"),
			Nullable: true,
		},
		{
			Name:     "Text_regexp",
			Type:     Text().WithRegexp(regexp.MustCompile(`oo$`)),
			Nullable: true,
		},
		{
			Name:     "Text_nil",
			Type:     Text(),
			Nullable: true,
		},
		{
			Name: "Array",
			Type: Array(Text()),
		},
		{
			Name: "Object",
			Type: Object([]Property{
				{
					Name: "a",
					Type: Int(32),
				},
				{
					Name:     "b",
					Type:     Boolean(),
					Nullable: true,
				},
				{
					Name:         "c",
					Type:         Uint(8),
					ReadOptional: true,
				},
			}),
		},
		{
			Name: "Map",
			Type: Map(Int(32)),
		},
	})

	data := `{"Boolean":true,"Int8":-12,"Int16":8023,"Int24":-2880217,"Int32":1307298102,"Int64":"927041163082605","Uint8":12,"Uint16":8023,"Uint24":2880217,"Uint32":1307298102,"Uint64":"927041163082605","Float32":57.16038,"Float64":18372.36240184391,"Decimal":1752.064,"DateTime":"2023-10-17T09:34:25.836540129Z","Date":"2023-10-17","Time":"09:34:25.836540129","Year":2023,"UUID":"550e8400-e29b-41d4-a716-446655440000","JSON":{"foo": 5,"boo": true},"JSON_null":null,"Inet":"192.158.1.38","Text":"some text","Text_values":"c","Text_regexp":"foo","Text_nil":null,"Array":["foo","boo"],"Object":{"a":9,"b":null},"Map":{"a":1,"b":2,"c":3}}`
	expected := map[string]any{
		"Boolean":     true,
		"Int8":        -12,
		"Int16":       8023,
		"Int24":       -2880217,
		"Int32":       1307298102,
		"Int64":       927041163082605,
		"Uint8":       uint(12),
		"Uint16":      uint(8023),
		"Uint24":      uint(2880217),
		"Uint32":      uint(1307298102),
		"Uint64":      uint(927041163082605),
		"Float32":     float64(float32(57.16038)),
		"Float64":     18372.36240184391,
		"Decimal":     decimal.MustParse("1752.064"),
		"DateTime":    time.Date(2023, 10, 17, 9, 34, 25, 836540129, time.UTC),
		"Date":        time.Date(2023, 10, 17, 0, 0, 0, 0, time.UTC),
		"Time":        time.Date(1970, 01, 01, 9, 34, 25, 836540129, time.UTC),
		"Year":        2023,
		"UUID":        "550e8400-e29b-41d4-a716-446655440000",
		"JSON":        json.Value(`{"foo": 5,"boo": true}`),
		"JSON_null":   json.Value(`null`),
		"JSON_nil":    json.Value(`null`),
		"Inet":        "192.158.1.38",
		"Text":        "some text",
		"Text_values": "c",
		"Text_regexp": "foo",
		"Text_nil":    nil,
		"Array":       []any{"foo", "boo"},
		"Object":      map[string]any{"a": 9, "b": nil},
		"Map":         map[string]any{"a": 1, "b": 2, "c": 3},
	}

	tests := []struct {
		typ      Type
		data     string
		expected map[string]any
		err      error
	}{
		{
			data: ``,
			err:  json.NewSyntaxError(io.EOF, 0),
		},
		{
			data:     data,
			expected: expected,
		},
		{
			data: data + ",",
			err:  json.NewSyntaxError(errors.New("invalid character ',' at start of value"), 0),
		},
		{
			data: data + "," + data,
			err:  json.NewSyntaxError(errors.New("invalid character ',' at start of value"), 0),
		},
		{
			data: `{"Boolean":[],}`,
			err:  json.NewSyntaxError(errors.New("invalid character ',' at start of value"), 0),
		},
		{
			data: `5`,
			err:  newErrInvalidValue("does not have a valid value: 5", ""),
		},
		{
			data: `{"Boolean":}`,
			err:  json.NewSyntaxError(errors.New("invalid character '}' at start of value"), 0),
		},
		{
			data: `{"Boolean":true`,
			err:  json.NewSyntaxError(errors.New("unexpected EOF"), 0),
		},
		{
			data: `{"Object":{"a.b":true}}`,
			err:  json.NewSyntaxError(errors.New("property name is not valid"), 0),
		},
		{
			data: `[{"Boolean":true}]`,
			err:  newErrInvalidValue("cannot be an array", ""),
		},
		{
			data: `{"Object":{"d":5}}`,
			err:  newErrPropertyNotExist("Object.d"),
		},
		{
			data: `{"Object":{"b":true}}`,
			err:  newErrMissingProperty("Object.a"),
		},
		{
			data: `{"Object":{"b":3}}`,
			err:  newErrInvalidValue(`does not have a valid value: 3`, "Object.b"),
		},
		{
			data: `{"Int8":21}`,
			err:  newErrInvalidValue(`is out of range [-20, 20]: 21`, "Int8"),
		},
		{
			data: `{"Int8":-25}`,
			err:  newErrInvalidValue(`is out of range [-20, 20]: -25`, "Int8"),
		},
		{
			data: `{"Boolean":"a \" \\ b"}`,
			err:  newErrInvalidValue(`does not have a valid value: "a \" \\ b"`, "Boolean"),
		},
		{
			data: `{"Boolean":null}`,
			err:  newErrInvalidValue(`cannot be null`, "Boolean"),
		},
		{
			data: `{"Date":"2023-02-30"}`,
			err:  newErrInvalidValue(`does not have a valid value: "2023-02-30"`, "Date"),
		},
		{
			data: `{"Text":"some long text"}`,
			err:  newErrInvalidValue(`is longer than 10 characters: "some long text"`, "Text"),
		},
		{
			data: `{"Text_values":"foo"}`,
			err:  newErrInvalidValue(`has an invalid value: "foo"; valid values are "a", "b", and "c"`, "Text_values"),
		},
		{
			data: `{"Text_regexp":"faa"}`,
			err:  newErrInvalidValue(`has an invalid value: "faa"; it does not match the property's regular expression`, "Text_regexp"),
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			b := strings.NewReader(test.data)
			testType := test.typ
			if !test.typ.Valid() {
				testType = object
			}
			got, err := Decode[map[string]any](b, testType)
			if err != nil {
				if test.err == nil {
					t.Fatalf("Decode: expected no error, got error %s", err)
				}
				if reflect.TypeOf(test.err) != reflect.TypeOf(err) || test.err != nil && test.err.Error() != err.Error() {
					t.Fatalf("Decode: expected error '%v' (type %T), got error '%v' (type %T)", test.err, test.err, err, err)
				}
				if got != nil {
					t.Fatalf("Decode: expected nil, got %#v", got)
				}
				return
			}
			if test.err != nil {
				t.Fatalf("Decode: expected error %q, got no error", test.err)
			}
			if err := equalValues(object, test.expected, got); err != nil {
				t.Fatalf("Decode:\n\texpected value %#v\n\tgot value      %#v\n\terror:   %s", test.expected, got, err)
			}
		})
	}

}

func Test_Marshal(t *testing.T) {
	tests := []struct {
		name   string
		schema Type
		value  map[string]any
		result []byte
	}{
		{
			name:   "Types",
			schema: schema,
			value:  value,
			result: []byte(`{"Boolean":true,"Int8":-12,"Int16":8023,"Int24":-2880217,"Int32":1307298102,"Int64":"927041163082605","Uint8":12,"Uint16":8023,"Uint24":2880217,"Uint32":1307298102,"Uint64":"927041163082605","Float32":57.16038,"Float64":18372.36240184391,"Float64_NaN":"NaN","Float64_Positive_Infinity":"Infinity","Float64_Negative_Infinity":"-Infinity","Decimal":1752.064,"DateTime":"2023-10-17T09:34:25.836042841Z","Date":"2023-10-17","Time":"09:34:25.836042841","Year":2023,"UUID":"550e8400-e29b-41d4-a716-446655440000","JSON":{"foo":5,"boo":true},"JSON_null":null,"Inet":"192.158.1.38","Text":"some text","Array":["foo","boo"],"Object":{"a":9,"b":false},"Map":{"a":1,"b":2,"c":3}}`),
		},
		{
			name:   "Empty",
			schema: schema,
			value:  map[string]any{},
			result: []byte(`{}`),
		},
		{
			name: "JSON nil",
			schema: Object([]Property{
				{
					Name:     "a",
					Type:     JSON(),
					Nullable: true,
				},
			}),
			value:  map[string]any{"a": nil},
			result: []byte(`{"a":null}`),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Marshal(test.value, test.schema)
			if err != nil {
				t.Fatalf("MarshalBySchema: unexpected error: %s", err)
			}
			if !bytes.Equal(test.result, got) {
				t.Fatalf("MarshalBySchema: expected %s, got %s", string(test.result), string(got))
			}
		})
	}
}

// equalValues reports whether v1 and v2 are equal according to the type t.
// v1 is supposed to conform to type t, and v2 is checked for equality with v1.
func equalValues(t Type, v1, v2 any) error {
	if v1 == nil {
		if v2 != nil {
			return fmt.Errorf("expected nil, got %#v (%T)", v2, v2)
		}
		return nil
	} else if v2 == nil {
		return fmt.Errorf("expected %#v (%T), got nil", v1, v1)
	}
	switch t.Kind() {
	case FloatKind:
		if t.BitSize() == 32 {
			f2, ok := v2.(float64)
			if !ok {
				return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
			}
			f1 := v1.(float64)
			if float32(f1) != float32(f2) {
				return fmt.Errorf("expected value %f, got %f", float32(f1), float32(f2))
			}
			return nil
		}
	case DecimalKind:
		d2, ok := v2.(decimal.Decimal)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		d1 := v1.(decimal.Decimal)
		if d1.Cmp(d2) != 0 {
			return fmt.Errorf("expected value %s, got %s", v1, d2)
		}
		return nil
	case DateTimeKind, DateKind, TimeKind:
		t2, ok := v2.(time.Time)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		t1 := v1.(time.Time)
		if !t1.Equal(t2) {
			return fmt.Errorf("expected value %s, got %s", v1, t2)
		}
		return nil
	case JSONKind:
		j2, ok := v2.(json.Value)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		j1 := v1.(json.Value)
		if !bytes.Equal(j1, j2) {
			return fmt.Errorf("expected value %q (%T), got %q (%T)", string(j1), v1, string(j2), v2)
		}
		return nil
	case ArrayKind:
		a1 := v1.([]any)
		a2, ok := v2.([]any)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		for i, e1 := range a1 {
			err := equalValues(t.Elem(), e1, a2[i])
			if err != nil {
				return err
			}
		}
		return nil
	case ObjectKind:
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
			err := equalValues(p.Type, s1, s2)
			if err != nil {
				return err
			}
			delete(unexpected, p.Name)
		}
		if len(unexpected) > 0 {
			property := ""
			for name := range unexpected {
				if property < name {
					property = name
				}
			}
			return fmt.Errorf("unexpected property %q", property)
		}
		return nil
	case MapKind:
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
			err := equalValues(t.Elem(), e1, e2)
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

var schema = Object([]Property{
	{
		Name: "Boolean",
		Type: Boolean(),
	},
	{
		Name: "Int8",
		Type: Int(8),
	},
	{
		Name: "Int16",
		Type: Int(16),
	},
	{
		Name: "Int24",
		Type: Int(24),
	},
	{
		Name: "Int32",
		Type: Int(32),
	},
	{
		Name: "Int64",
		Type: Int(64),
	},
	{
		Name: "Uint8",
		Type: Uint(8),
	},
	{
		Name: "Uint16",
		Type: Uint(16),
	},
	{
		Name: "Uint24",
		Type: Uint(24),
	},
	{
		Name: "Uint32",
		Type: Uint(32),
	},
	{
		Name: "Uint64",
		Type: Uint(64),
	},
	{
		Name: "Float32",
		Type: Float(32),
	},
	{
		Name: "Float64",
		Type: Float(64),
	},
	{
		Name: "Float64_NaN",
		Type: Float(64),
	},
	{
		Name: "Float64_Positive_Infinity",
		Type: Float(64),
	},
	{
		Name: "Float64_Negative_Infinity",
		Type: Float(64),
	},
	{
		Name: "Decimal",
		Type: Decimal(10, 3),
	},
	{
		Name: "DateTime",
		Type: DateTime(),
	},
	{
		Name: "Date",
		Type: Date(),
	},
	{
		Name: "Time",
		Type: Time(),
	},
	{
		Name: "Year",
		Type: Year(),
	},
	{
		Name: "UUID",
		Type: UUID(),
	},
	{
		Name: "JSON",
		Type: JSON(),
	},
	{
		Name: "JSON_null",
		Type: JSON(),
	},
	{
		Name: "Inet",
		Type: Inet(),
	},
	{
		Name: "Text",
		Type: Text(),
	},
	{
		Name: "Array",
		Type: Array(Text()),
	},
	{
		Name: "Object",
		Type: Object([]Property{
			{
				Name: "a",
				Type: Int(32),
			},
			{
				Name: "b",
				Type: Boolean(),
			},
		}),
	},
	{
		Name: "Map",
		Type: Map(Int(32)),
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
	"Decimal":                   decimal.MustParse("1752.064"),
	"DateTime":                  time.Date(2023, 10, 17, 9, 34, 25, 836042841, time.UTC),
	"Date":                      time.Date(2023, 10, 17, 0, 0, 0, 0, time.UTC),
	"Time":                      time.Date(1970, 01, 01, 9, 34, 25, 836042841, time.UTC),
	"Year":                      2023,
	"UUID":                      "550e8400-e29b-41d4-a716-446655440000",
	"JSON":                      json.Value(`{"foo":5,"boo":true}`),
	"JSON_null":                 json.Value(`null`),
	"Inet":                      "192.158.1.38",
	"Text":                      "some text",
	"Array":                     []any{"foo", "boo"},
	"Object":                    map[string]any{"a": 9, "b": false},
	"Map":                       map[string]any{"a": 1, "b": 2, "c": 3},
}
