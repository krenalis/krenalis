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
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

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
			Name:     "Text_values",
			Type:     types.Text().WithValues("a", "b", "c"),
			Nullable: true,
		},
		{
			Name:     "Text_regexp",
			Type:     types.Text().WithRegexp(regexp.MustCompile(`oo$`)),
			Nullable: true,
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
					Type: types.Int(32),
				},
				{
					Name: "b",
					Type: types.Boolean(),
				},
				{
					Name:     "c",
					Type:     types.Uint(8),
					Nullable: true,
				},
			}),
		},
		{
			Name: "Map",
			Type: types.Map(types.Int(32)),
		},
	})

	data := `{"Boolean":true,"Int8":-12,"Int16":8023,"Int24":-2880217,"Int32":1307298102,"Int64":"927041163082605","Uint8":12,"Uint16":8023,"Uint24":2880217,"Uint32":1307298102,"Uint64":"927041163082605","Float32":57.16038,"Float64":18372.36240184391,"Decimal":"1752.064","DateTime":"2023-10-17T09:34:25.836540129Z","Date":"2023-10-17","Time":"09:34:25.836540129","Year":2023,"UUID":"550e8400-e29b-41d4-a716-446655440000","JSON":"{\"foo\":5,\"boo\":true}","JSON_null":"null","JSON_nil":null,"Inet":"192.158.1.38","Text":"some text","Text_values":"c","Text_regexp":"foo","Text_nil":null,"Array":["foo","boo"],"Object":{"a":9,"b":false},"Map":{"a":1,"b":2,"c":3}}`
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
		"Decimal":     decimal.RequireFromString("1752.064"),
		"DateTime":    time.Date(2023, 10, 17, 9, 34, 25, 836540129, time.UTC),
		"Date":        time.Date(2023, 10, 17, 0, 0, 0, 0, time.UTC),
		"Time":        time.Date(1970, 01, 01, 9, 34, 25, 836540129, time.UTC),
		"Year":        2023,
		"UUID":        "550e8400-e29b-41d4-a716-446655440000",
		"JSON":        json.RawMessage(`{"foo":5,"boo":true}`),
		"JSON_null":   json.RawMessage(`null`),
		"JSON_nil":    nil,
		"Inet":        "192.158.1.38",
		"Text":        "some text",
		"Text_values": "c",
		"Text_regexp": "foo",
		"Text_nil":    nil,
		"Array":       []any{"foo", "boo"},
		"Object":      map[string]any{"a": 9, "b": false, "c": nil},
		"Map":         map[string]any{"a": 1, "b": 2, "c": 3},
	}

	tests := []struct {
		schema   types.Type
		data     string
		expected map[string]any
		err      error
	}{
		{
			data: ``,
			err:  ErrSyntaxInvalid,
		},
		{
			data:     data,
			expected: expected,
		},
		{
			data: data + ",",
			err:  ErrSyntaxInvalid,
		},
		{
			data: data + "," + data,
			err:  ErrSyntaxInvalid,
		},
		{
			data: `{"Boolean":[],}`,
			err:  ErrSyntaxInvalid,
		},
		{
			data: `5`,
			err:  newErrInvalidValue(`does not have a valid value: 5`, "data"),
		},
		{
			data: `{"Boolean":}`,
			err:  ErrSyntaxInvalid,
		},
		{
			data: `{"Boolean":true`,
			err:  ErrSyntaxInvalid,
		},
		{
			data: `[{"Boolean":true}]`,
			err:  newErrInvalidValue(`cannot be an array`, "data"),
		},
		{
			data: `{"Object":{"d":5}}`,
			err:  newErrPropertyNotExist("data.Object.d"),
		},
		{
			data: `{"Object":{"b":true}}`,
			err:  newErrMissingProperty("data.Object.a"),
		},
		{
			data: `{"Object":{"b":3}}`,
			err:  newErrInvalidValue(`does not have a valid value: 3`, "data.Object.b"),
		},
		{
			data: `{"Int8":21}`,
			err:  newErrInvalidValue(`is out of range [-20, 20]: 21`, "data.Int8"),
		},
		{
			data: `{"Int8":-25}`,
			err:  newErrInvalidValue(`is out of range [-20, 20]: -25`, "data.Int8"),
		},
		{
			data: `{"Boolean":"a \" \\ b"}`,
			err:  newErrInvalidValue(`does not have a valid value: "a \" \\ b"`, "data.Boolean"),
		},
		{
			data: `{"Boolean":null}`,
			err:  newErrInvalidValue(`cannot be null`, "data.Boolean"),
		},
		{
			data: `{"Date":"2023-02-30"}`,
			err:  newErrInvalidValue(`does not have a valid value: "2023-02-30"`, "data.Date"),
		},
		{
			data: `{"Text":"some long text"}`,
			err:  newErrInvalidValue(`is longer than 10 characters: "some long text"`, "data.Text"),
		},
		{
			data: `{"Text_values":"foo"}`,
			err:  newErrInvalidValue(`has an invalid value: "foo"; valid values are "a", "b", and "c"`, "data.Text_values"),
		},
		{
			data: `{"Text_regexp":"faa"}`,
			err:  newErrInvalidValue(`has an invalid value: "faa"; it does not match the property's regular expression`, "data.Text_regexp"),
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			b := strings.NewReader(test.data)
			testSchema := test.schema
			if !test.schema.Valid() {
				testSchema = schema
			}
			got, err := Unmarshal(b, "data", testSchema)
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
			if test.err != nil {
				t.Fatalf("Unmarshal: expected error %q, got no error", test.err)
			}
			if err := equalValues(schema, test.expected, got); err != nil {
				t.Fatalf("Unmarshal:\n\texpected value %#v\n\tgot value      %#v\n\terror:   %s", test.expected, got, err)
			}
		})
	}

}

func Test_UnmarshalSlice(t *testing.T) {

	schema := types.Object([]types.Property{
		{
			Name: "foo",
			Type: types.Boolean(),
		},
		{
			Name:     "boo",
			Type:     types.Int(32),
			Nullable: true,
		},
	})

	tests := []struct {
		data     string
		expected []map[string]any
		err      error
	}{
		{
			data: ``,
			err:  ErrSyntaxInvalid,
		},
		{
			data: `{}`,
			err:  ErrSyntaxInvalid,
		},
		{
			data:     `[]`,
			expected: []map[string]any{},
		},
		{
			data: `[5]`,
			err:  newErrInvalidValue(`does not have a valid value: 5`, "data[0]"),
		},
		{
			data: `[{"foo":}]`,
			err:  ErrSyntaxInvalid,
		},
		{
			data: `[{"foo":true`,
			err:  ErrSyntaxInvalid,
		},
		{
			data:     `[{"foo":true}]`,
			expected: []map[string]any{{"foo": true, "boo": nil}},
		},
		{
			data: `[{"foo":true}] ,`,
			err:  ErrSyntaxInvalid,
		},
		{
			data: `[{"foo":true}],[{"foo":false}]`,
			err:  ErrSyntaxInvalid,
		},
		{
			data:     `[{"foo":true},{"foo": false,"boo":547}]`,
			expected: []map[string]any{{"foo": true, "boo": nil}, {"foo": false, "boo": 547}},
		},
		{
			data: `[{"ops":5},{"boo":547}]`,
			err:  newErrPropertyNotExist("data[0].ops"),
		},
		{
			data: `[{"foo":3},{"boo":547}]`,
			err:  newErrInvalidValue(`does not have a valid value: 3`, "data[0].foo"),
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			b := strings.NewReader(test.data)
			got, err := UnmarshalSlice(b, "data", schema)
			if err != nil {
				if test.err == nil {
					t.Fatalf("UnmarshalSlice: expected no error, got error %s", err)
				}
				if !reflect.DeepEqual(test.err, err) {
					t.Fatalf("UnmarshalSlice: expected error %q, got error %q", test.err, err)
				}
				if got != nil {
					t.Fatalf("UnmarshalSlice: expected nil, got %#v", got)
				}
				return
			}
			if test.err != nil {
				t.Fatalf("UnmarshalSlice: expected error %q, got no error", test.err)
			}
			if len(test.expected) != len(got) {
				t.Fatalf("UnmarshalSlice: expected %d values, got %d", len(test.expected), len(got))
			}
			for i, value := range test.expected {
				if err := equalValues(schema, value, got[i]); err != nil {
					t.Fatalf("UnmarshalSlice data[%d]:\n\texpected value %#v\n\tgot value      %#v\n\terror:   %s", i, value, got[i], err)
				}
			}
		})
	}

}

// equalValues reports whether v1 and v2 are equal according to the type t.
// v1 is supposed to conform to type t, and v2 is checked for equality with v1.
func equalValues(t types.Type, v1, v2 any) error {
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
		t1 := v1.(time.Time)
		if !t1.Equal(t2) {
			return fmt.Errorf("expected value %s, got %s", v1, t2)
		}
		return nil
	case types.JSONKind:
		j2, ok := v2.(json.RawMessage)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		j1 := v1.(json.RawMessage)
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
			err := equalValues(t.Elem(), e1, a2[i])
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
			err := equalValues(p.Type, s1, s2)
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
	case types.MapKind:
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
