// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package types

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"math"
	"net/netip"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

func Test_Decode(t *testing.T) {

	object := Object([]Property{
		{
			Name: "String",
			Type: String().WithMaxLength(10),
		},
		{
			Name:     "Text_values",
			Type:     String().WithValues("a", "b", "c"),
			Nullable: true,
		},
		{
			Name:     "Text_regexp",
			Type:     String().WithPattern(regexp.MustCompile(`oo$`)),
			Nullable: true,
		},
		{
			Name:     "Text_nil",
			Type:     String(),
			Nullable: true,
		},
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
			Type: Int(8).Unsigned(),
		},
		{
			Name: "Uint16",
			Type: Int(16).Unsigned(),
		},
		{
			Name: "Uint24",
			Type: Int(24).Unsigned(),
		},
		{
			Name: "Uint32",
			Type: Int(32).Unsigned(),
		},
		{
			Name: "Uint64",
			Type: Int(64).Unsigned(),
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
			Name: "IP",
			Type: IP(),
		},
		{
			Name: "Array",
			Type: Array(String()),
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
					Type:         Int(8).Unsigned(),
					ReadOptional: true,
				},
			}),
		},
		{
			Name: "Map",
			Type: Map(Int(32)),
		},
	})

	data := `{"String":"some text","Text_values":"c","Text_regexp":"foo","Text_nil":null,"Boolean":true,"Int8":-12,"Int16":8023,"Int24":-2880217,"Int32":1307298102,"Int64":"927041163082605","Uint8":12,"Uint16":8023,"Uint24":2880217,"Uint32":1307298102,"Uint64":"927041163082605","Float32":57.16038,"Float64":18372.36240184391,"Decimal":1752.064,"DateTime":"2023-10-17T09:34:25.836540129Z","Date":"2023-10-17","Time":"09:34:25.836540129","Year":2023,"UUID":"550E8400-E29B-41D4-A716-446655440000","JSON":{"foo": 5,"boo": true},"JSON_null":null,"IP":"192.158.1.38","Array":["foo","boo"],"Object":{"a":9,"b":null},"Map":{"a":1,"b":2,"c":3}}`
	expected := map[string]any{
		"String":      "some text",
		"Text_values": "c",
		"Text_regexp": "foo",
		"Text_nil":    nil,
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
		"IP":          "192.158.1.38",
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
			typ:  Object([]Property{{Name: "Array", Type: Array(Int(32))}}),
			data: `{"Array":[1,"two"]}`,
			err:  newErrInvalidValue(`does not have a valid value: "two"`, "Array[1]"),
		},
		{
			typ:  Object([]Property{{Name: "Array", Type: Array(Int(32)).WithMaxElements(3)}}),
			data: `{"Array":[1,2,3,4]}`,
			err:  newErrInvalidValue("contains more than 3 elements", "Array"),
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
			data: `{"UUID":"550e8400e29b41d4a716446655440000"}`,
			err:  newErrInvalidValue(`does not have a valid value: "550e8400e29b41d4a716446655440000"`, "UUID"),
		},
		{
			data: `{"String":"some long text"}`,
			err:  newErrInvalidValue(`is longer than 10 characters: "some long text"`, "String"),
		},
		{
			typ: Object([]Property{
				{Name: "String", Type: String().WithMaxBytes(3)},
			}),
			data: `{"String":"éé"}`,
			err:  newErrInvalidValue(`is longer than 3 bytes: "éé"`, "String"),
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

// Test_NormalizeIP checks IP address normalization and validation.
func Test_NormalizeIP(t *testing.T) {

	tests := []struct {
		name, value, want string
		valid             bool
	}{
		{"IPv4", "192.0.2.1", "192.0.2.1", true},
		{"IPv6", "2001:0db8:0000:0000:0000:ff00:0042:8329", "2001:db8::ff00:42:8329", true},
		{"zoned IPv6", "fe80::1ff:fe23:4567:890a%eth0", "fe80::1ff:fe23:4567:890a", true},
		{"IPv4-mapped IPv6", "::ffff:192.0.2.1", "192.0.2.1", true},
		{"prefix", "192.0.2.1/24", "", false},
		{"invalid", "not an IP address", "", false},
		{"empty", "", "", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, valid := NormalizeIP(test.value)
			if got != test.want || valid != test.valid {
				t.Fatalf("got (%q, %t), want (%q, %t)", got, valid, test.want, test.valid)
			}
		})
	}

	for _, test := range []struct {
		name  string
		value netip.Addr
		want  string
		valid bool
	}{
		{"IPv4 address", netip.MustParseAddr("192.0.2.1"), "192.0.2.1", true},
		{"IPv6 address", netip.MustParseAddr("2001:0db8:0000:0000:0000:ff00:0042:8329"), "2001:db8::ff00:42:8329", true},
		{"zoned IPv6 address", netip.MustParseAddr("fe80::1%eth0"), "fe80::1", true},
		{"IPv4-mapped IPv6 address", netip.MustParseAddr("::ffff:192.0.2.1"), "192.0.2.1", true},
		{"invalid address", netip.Addr{}, "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, valid := NormalizeIP(test.value)
			if got != test.want || valid != test.valid {
				t.Fatalf("got (%q, %t), want (%q, %t)", got, valid, test.want, test.valid)
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
			result: []byte(`{"String":"some text","Boolean":true,"Int8":-12,"Int16":8023,"Int24":-2880217,"Int32":1307298102,"Int64":"927041163082605","Uint8":12,"Uint16":8023,"Uint24":2880217,"Uint32":1307298102,"Uint64":"927041163082605","Float32":57.16038,"Float64":18372.36240184391,"Float64_NaN":"NaN","Float64_Positive_Infinity":"Infinity","Float64_Negative_Infinity":"-Infinity","Decimal":1752.064,"DateTime":"2023-10-17T09:34:25.836042841Z","Date":"2023-10-17","Time":"09:34:25.836042841","Year":2023,"UUID":"550e8400-e29b-41d4-a716-446655440000","JSON":{"foo":5,"boo":true},"JSON_null":null,"IP":"192.158.1.38","Array":["foo","boo"],"Object":{"a":9,"b":false},"Map":{"a":1,"b":2,"c":3}}`),
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
		for _, p := range t.Properties().All() {
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
		Name: "String",
		Type: String(),
	},
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
		Type: Int(8).Unsigned(),
	},
	{
		Name: "Uint16",
		Type: Int(16).Unsigned(),
	},
	{
		Name: "Uint24",
		Type: Int(24).Unsigned(),
	},
	{
		Name: "Uint32",
		Type: Int(32).Unsigned(),
	},
	{
		Name: "Uint64",
		Type: Int(64).Unsigned(),
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
		Name: "IP",
		Type: IP(),
	},
	{
		Name: "Array",
		Type: Array(String()),
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
	"String":                    "some text",
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
	"IP":                        "192.158.1.38",
	"Array":                     []any{"foo", "boo"},
	"Object":                    map[string]any{"a": 9, "b": false},
	"Map":                       map[string]any{"a": 1, "b": 2, "c": 3},
}

// TestDecodeArrayUnique checks the optional constraint after decoding and normalization.
func TestDecodeArrayUnique(t *testing.T) {

	tests := []struct {
		name      string
		element   Type
		source    string
		duplicate int
	}{
		{"empty", String(), `[]`, -1},
		{"singleton", String(), `["a"]`, -1},
		{"distinct", Int(32), `[1,2,3]`, -1},
		{"first duplicate", Int(32), `[1,2,2,1]`, 2},
		{"signed int64", Int(64), `["9223372036854775807","9223372036854775807"]`, 1},
		{"unsigned int64", Int(64).Unsigned(), `["18446744073709551615","18446744073709551615"]`, 1},
		{"strings", String(), `["a","b","a"]`, 2},
		{"string case", String(), `["Foo","foo"]`, -1},
		{"booleans", Boolean(), `[true,false,true]`, 2},
		{"decimal representations", Decimal(6, 2), `[1.50,15e-1]`, 1},
		{"decimal signed zero", Decimal(6, 2), `[0,-0.00]`, 1},
		{"decimal signs", Decimal(6, 2), `[-1.28,1.28]`, -1},
		{"decimal precision", Decimal(20, 0), `[9007199254740992,9007199254740993]`, -1},
		{"large decimal representations", Decimal(76, 2), `[1e25,10000000000000000000000000.00]`, 1},
		{"float32 normalization", Float(32), `[16777216,16777217]`, 1},
		{"float signed zero", Float(64), `[0,-0]`, 1},
		{"NaN", Float(64), `["NaN","NaN"]`, 1},
		{"NaN32", Float(32), `["NaN",0,"NaN"]`, 2},
		{"escaped NaN", Float(64), `["NaN","\u004eaN"]`, 1},
		{"one NaN", Float(64), `[0,"NaN","Infinity","-Infinity"]`, -1},
		{"infinity", Float(64), `["Infinity","Infinity"]`, 1},
		{"negative infinity", Float(64), `["-Infinity","-Infinity"]`, 1},
		{
			"datetime offsets", DateTime(),
			`["2026-09-08T10:00:00Z","2026-09-08T12:00:00+02:00"]`, 1,
		},
		{"dates", Date(), `["2026-09-08","2026-09-08"]`, 1},
		{"times", Time(), `["10:00:00.1","10:00:00.100"]`, 1},
		{"years", Year(), `[2025,2026,2025]`, 2},
		{
			"UUID case", UUID(),
			`["550e8400-e29b-41d4-a716-446655440000","550E8400-E29B-41D4-A716-446655440000"]`, 1,
		},
		{"IP normalization", IP(), `["2001:db8::1","2001:0db8:0:0:0:0:0:1"]`, 1},
	}

	for _, test := range tests {
		for _, unique := range []bool{false, true} {
			for _, nested := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/unique=%t/nested=%t", test.name, unique, nested), func(t *testing.T) {

					typ := Array(test.element)
					if unique {
						typ = typ.WithUnique()
					}
					source := test.source
					path := fmt.Sprintf("[%d]", test.duplicate)
					if nested {
						typ = Array(Object([]Property{{Name: "values", Type: typ}}))
						source = `[{"values":` + source + `}]`
						path = "[0].values" + path
					}

					got, err := Decode[[]any](strings.NewReader(source), typ)
					if err != nil {
						if !unique || test.duplicate < 0 {
							t.Fatal(err)
						}
						validation, ok := errors.AsType[*SchemaValidationError](err)
						if !ok {
							t.Fatalf("got %T: %v, want SchemaValidationError", err, err)
						}
						if validation.path != path || validation.msg != "duplicates an earlier array element" {
							t.Fatalf("got %v, want a duplicate at %s", err, path)
						}
						if got != nil {
							t.Fatalf("got partial value %#v with error", got)
						}
						return
					}
					if unique && test.duplicate >= 0 {
						t.Fatalf("got %#v, want a duplicate element error", got)
					}
					if nested {
						got = got[0].(map[string]any)["values"].([]any)
					}
					if want := json.Value(test.source).NumElement(); len(got) != want {
						t.Fatalf("got %d values, want %d", len(got), want)
					}
					if test.name == "one NaN" {
						if !math.IsNaN(got[1].(float64)) ||
							!math.IsInf(got[2].(float64), 1) || !math.IsInf(got[3].(float64), -1) {
							t.Fatalf("got %#v, want NaN and infinities with their original signs", got)
						}
					}

				})
			}
		}
	}

}

// TestDecodeArrayUniqueErrors checks interaction with validation and JSON syntax errors.
func TestDecodeArrayUniqueErrors(t *testing.T) {

	tests := []struct {
		name, source string
		typ          Type
		message      string
		syntax       bool
	}{
		{"maximum count", `[1,2,3]`, Array(Int(32)).WithMaxElements(2).WithUnique(), "contains more than 2 elements", false},
		{"minimum count", `[1]`, Array(Int(32)).WithMinElements(2).WithUnique(), "contains less than 2 elements", false},
		{"real NaN", `["NaN"]`, Array(Float(64).Real()).WithUnique(), "is not a real", false},
		{"real infinity", `["Infinity"]`, Array(Float(32).Real()).WithUnique(), "is not a real", false},
		{"invalid third element", `[1,2,"bad"]`, Array(Int(32)).WithUnique(), `"[2]"`, false},
		{"malformed after duplicate", `[1,1,`, Array(Int(32)).WithUnique(), "", true},
		{"malformed after array", `[1,1] ?`, Array(Int(32)).WithUnique(), "", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Decode[[]any](strings.NewReader(test.source), test.typ)
			if err != nil {
				if test.syntax {
					if _, ok := errors.AsType[*json.SyntaxError](err); !ok {
						t.Fatalf("got %T: %v, want SyntaxError", err, err)
					}
					return
				}
				if _, ok := errors.AsType[*SchemaValidationError](err); !ok {
					t.Fatalf("got %T: %v, want SchemaValidationError", err, err)
				}
				if !strings.Contains(err.Error(), test.message) {
					t.Fatalf("got %q, want %q in the message", err, test.message)
				}
				return
			}
			t.Fatal("expected an error")
		})
	}

}

// TestDecodePhone checks canonical phone decoding at string leaves.
func TestDecodePhone(t *testing.T) {

	phone := String().AsPhone()
	tests := []struct {
		name  string
		typ   Type
		input string
		want  any
	}{
		{"canonical", phone, `"+390236618300"`, "+390236618300"},
		{"possible", phone, `"+12001230101"`, "+12001230101"},
		{"escaped canonical", phone, `"\u002b390236618300"`, "+390236618300"},
		{"formatted", phone, `"+39 02-36618 300"`, nil},
		{"local only", phone, `"+12530000"`, nil},
		{"double plus", phone, `"++390236618300"`, nil},
		{
			"array", Array(phone).WithMaxElements(2), `["+39 02-36618 300","+12001230101"]`,
			nil,
		},
		{
			"nested", Array(Map(phone)), `[{"home":"+39 02-36618 300"}]`,
			nil,
		},
		{
			"nested canonical", Array(Map(phone)), `[{"home":"+390236618300"}]`,
			[]any{map[string]any{"home": "+390236618300"}},
		},
		{"ordinary string", String(), `"+39 02-36618 300"`, "+39 02-36618 300"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Decode[any](strings.NewReader(test.input), test.typ)
			valid := test.want != nil
			if err != nil {
				if valid {
					t.Fatalf("expected no error, got %v", err)
				}
				if _, ok := errors.AsType[*SchemaValidationError](err); !ok {
					t.Fatalf("expected a SchemaValidationError, got %T", err)
				}
				return
			}
			if !valid || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("expected value %#v and valid=%t, got %#v and no error", test.want, valid, got)
			}
		})
	}

}

// TestDecodeSemantics checks semantic constraints and nested containers.
func TestDecodeSemantics(t *testing.T) {

	country := String().AsCountry(ISO3166Alpha2)
	tests := []struct {
		name     string
		semantic Type
		value    string
		valid    bool
	}{
		{"current country", country, "IT", true},
		{"alpha-3 country", String().AsCountry(ISO3166Alpha3), "ITA", true},
		{"former alpha-3 country", String().AsCountry(ISO3166Alpha3), "ANT", true},
		{"unknown alpha-3 country", String().AsCountry(ISO3166Alpha3), "ZZZ", false},
		{"reserved alpha-3 country", String().AsCountry(ISO3166Alpha3), "EUR", false},
		{"empty alpha-3 country", String().AsCountry(ISO3166Alpha3), "", false},
		{"short alpha-3 country", String().AsCountry(ISO3166Alpha3), "IT", false},
		{"lowercase alpha-3 country", String().AsCountry(ISO3166Alpha3), "ita", false},
		{"mixed case alpha-3 country", String().AsCountry(ISO3166Alpha3), "Ita", false},
		{"numeric alpha-3 country", String().AsCountry(ISO3166Alpha3), "380", false},
		{"non-ASCII alpha-3 country", String().AsCountry(ISO3166Alpha3), "éA", false},
		{"long alpha-3 country", String().AsCountry(ISO3166Alpha3), "ITAL", false},
		{"former country", country, "AN", true},
		{"unknown country", country, "ZZ", false},
		{"reserved country", country, "UK", false},
		{"lowercase country", country, "it", false},
		{"empty country", country, "", false},
		{"short country", country, "I", false},
		{"long country", country, "ITA", false},
		{"non-ASCII country", country, "é", false},
		{"canonical phone", String().AsPhone(), "+390236618300", true},
		{"formatted phone", String().AsPhone(), "+39 02-36618 300", false},
		{"structurally possible phone", String().AsPhone(), "+12001230101", true},
		{"local-only phone", String().AsPhone(), "+12530000", false},
		{"double plus phone", String().AsPhone(), "++390236618300", false},
		{"long phone", String().AsPhone(), "+1234567890123456", false},
		{"empty phone", String().AsPhone(), "", false},
		{"phone without format restriction", String().AsPhone(), "a (b)", false},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			for _, shape := range []string{"string", "array", "map", "nested", "object"} {

				t.Run(shape, func(t *testing.T) {

					typ := test.semantic
					data := strconv.Quote(test.value)
					switch shape {
					case "array":
						typ = Array(typ)
						data = "[" + data + "]"
					case "map":
						typ = Map(typ)
						data = "{\"home\":" + data + "}"
					case "nested":
						typ = Array(Map(Array(typ)))
						data = "[{\"home\":[" + data + "]}]"
					case "object":
						typ = Object([]Property{{Name: "inner", Type: typ}})
						data = "{\"inner\":" + data + "}"
					}
					schema := Object([]Property{{Name: "value", Type: typ}})
					input := strings.NewReader("{\"value\":" + data + "}")
					_, err := Decode[map[string]any](input, schema)
					if err != nil {
						if test.valid {
							t.Fatalf("expected no error, got %v", err)
						}
						if _, ok := errors.AsType[*SchemaValidationError](err); !ok {
							t.Fatalf("expected SchemaValidationError, got %T", err)
						}
						return
					}
					if !test.valid {
						t.Fatal("expected a SchemaValidationError, got nil")
					}

				})

			}

		})

	}

}

// TestSemanticConstraintsRoundTrip checks that semantics do not introduce
// string constraints.
func TestSemanticConstraintsRoundTrip(t *testing.T) {

	typesToTest := []Type{
		String().AsCountry(ISO3166Alpha2),
		String().AsCountry(ISO3166Alpha3),
		String().AsPhone(),
	}
	for _, typ := range typesToTest {
		data, err := typ.MarshalJSON()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if strings.Contains(string(data), "maxBytes") || strings.Contains(string(data), "maxLength") {
			t.Fatalf("expected no serialized string constraints, got %s", data)
		}
		var got Type
		err = got.UnmarshalJSON(data)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if n, ok := got.MaxBytes(); ok || n != 0 {
			t.Fatalf("expected no maxBytes constraint, got %d and %t", n, ok)
		}
		if n, ok := got.MaxLength(); ok || n != 0 {
			t.Fatalf("expected no maxLength constraint, got %d and %t", n, ok)
		}
	}

}

// BenchmarkDecodePhones measures materialization of a batch of canonical phone
// strings.
func BenchmarkDecodePhones(b *testing.B) {

	input := "[" + strings.Repeat(`"+390236618300",`, 511) + `"+390236618300"]`
	typ := Array(String().AsPhone())
	b.ReportAllocs()
	b.SetBytes(int64(len(input)))
	for b.Loop() {
		_, err := Decode[[]any](strings.NewReader(input), typ)
		if err != nil {
			b.Fatalf("expected no error, got %v", err)
		}
	}

}

// BenchmarkNormalizePhone measures representative normalization and rejection
// paths.
func BenchmarkNormalizePhone(b *testing.B) {
	tests := []struct {
		name  string
		input string
	}{
		{"canonical", "+390236618300"},
		{"formatted", "+39 02-36618 300"},
		{"invalid", "++390236618300"},
	}
	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				NormalizePhone(test.input)
			}
		})
	}
}

// BenchmarkNormalizePhoneInRegion measures representative normalization paths
// using a parsing region.
func BenchmarkNormalizePhoneInRegion(b *testing.B) {
	tests := []struct {
		name   string
		input  string
		region string
	}{
		{"national", "02-36618 300", "IT"},
		{"international", "+390236618300", "IT"},
		{"invalid", "2530000", "US"},
	}

	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				NormalizePhoneInRegion(test.input, test.region)
			}
		})
	}
}

// FuzzNormalizePhone checks canonical output and idempotence for arbitrary
// input.
func FuzzNormalizePhone(f *testing.F) {
	for _, s := range []string{
		"",
		"+",
		"+390236618300",
		"+39 02-36618 300",
		"+393401234567",
		"+３９０２３６６１８３００",
		"0236618300",
		"+12530000",
		"+12001230101",
		"+270000000",
		"+390236618300 x42",
		" +390236618300",
		"++390236618300",
		"tel:+390236618300",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		canonical, ok := NormalizePhone(s)
		if !ok {
			if canonical != "" {
				t.Fatalf("expected an empty result after failed normalization, got %q", canonical)
			}
			return
		}
		if len(canonical) < 2 ||
			len(canonical) > 16 ||
			canonical[0] != '+' ||
			canonical[1] < '1' ||
			canonical[1] > '9' {
			t.Fatalf("expected canonical E.164 output, got %q", canonical)
		}
		for _, r := range canonical[1:] {
			if r < '0' || r > '9' {
				t.Fatalf("expected ASCII digits after '+', got %q", canonical)
			}
		}
		if again, ok := NormalizePhone(canonical); !ok || again != canonical {
			t.Fatalf(
				"expected repeated normalization of %q to return %q and true, got %q and %t",
				s, canonical, again, ok,
			)
		}
	})
}

// TestIsPhone checks whether phone values are already in canonical E.164 form.
func TestIsPhone(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"canonical", "+390236618300", true},
		{"structurally possible", "+12001230101", true},
		{"formatted", "+39 02-36618 300", false},
		{"national", "0236618300", false},
		{"local-only", "+12530000", false},
		{"invalid", "not a phone", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsPhone(test.input); got != test.want {
				t.Fatalf("expected IsPhone(%q) to return %t, got %t", test.input, test.want, got)
			}
		})
	}
}

// TestNormalizePhone checks international phone normalization and whole-input
// validation.
func TestNormalizePhone(t *testing.T) {

	tests := []struct {
		input string
		want  string
	}{
		{"+390236618300", "+390236618300"},
		{"+39 02-36618 300", "+390236618300"},
		{"+39 (02) 36618.300", "+390236618300"},
		{"+39\u00a002\u201136618300", "+390236618300"},
		{"+３９０２３６６１８３００", "+390236618300"},
		{"+٣٩٠٢٣٦٦١٨٣٠٠", "+390236618300"},
		{"+393401234567", "+393401234567"},
		{"+1 (650) 253-0000", "+16502530000"},
		{"+80012345678", "+80012345678"},
		{"", ""},
		{"+", ""},
		{"++390236618300", ""},
		{"+ +390236618300", ""},
		{"(+39)0236618300", ""},
		{"3902+36618300", ""},
		{"＋390236618300", ""},
		{"+39＋0236618300", ""},
		{"00390236618300", ""},
		{"011390236618300", ""},
		{"0236618300", ""},
		{"390236618300", ""},
		{" +390236618300", ""},
		{"+390236618300 ", ""},
		{"\u00a0+390236618300", ""},
		{"+390236618300\u200b", ""},
		{"Call +390236618300", ""},
		{"+390236618300 please", ""},
		{"tel:+390236618300", ""},
		{"tel:+390236618300;isub=42", ""},
		{"tel:0236618300;phone-context=+39;ext=42", ""},
		{"+1-800-FLOWERS", ""},
		{"+390236618300 x42", ""},
		{"+390236618300 ext. 42", ""},
		{"+390236618300#42", ""},
		{"+390236618300;42", ""},
		{"+390236618300,42", ""},
		{"+390236618300/+16502530000", ""},
		{"+390236618300 6502530000", ""},
		{"+39\t0236618300", ""},
		{"+390236618300\n", ""},
		{"+390236618300\xff", ""},
		{"+12001230101", "+12001230101"}, // structurally possible even if not classified as valid
		{"+12530000", ""},                // possible only locally: missing an area code
		{"+270000000", ""},               // canonical output must be stable when parsed again
		{"+3902", ""},
		{"+9990236618300", ""},
		{"+49301234567890123", ""}, // numbering-plan lengths must still fit E.164
		{"+" + strings.Repeat("1", 250), ""},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()
			got, ok := NormalizePhone(test.input)
			wantOK := test.want != ""
			if got != test.want || ok != wantOK {
				t.Fatalf("expected NormalizePhone(%q) to return %q and %t, got %q and %t", test.input, test.want, wantOK, got, ok)
			}
		})
	}

}

// TestValidPhoneInput checks the input grammar before phone parsing.
func TestValidPhoneInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"at byte limit", "+" + strings.Repeat("1", 249), true},
		{"over byte limit", "+" + strings.Repeat("1", 250), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validPhoneInput(test.input); got != test.valid {
				t.Fatalf("expected validPhoneInput(%q) to return %t, got %t", test.input, test.valid, got)
			}
		})
	}
}

// TestNormalizePhoneInRegion checks parsing with a region without restricting
// the number's country.
func TestNormalizePhoneInRegion(t *testing.T) {

	tests := []struct {
		input  string
		region string
		want   string
	}{
		{"02-36618 300", "IT", "+390236618300"},
		{"340 1234567", "IT", "+393401234567"},
		{"0039 02-36618 300", "IT", "+390236618300"},
		{"0039 02-36618 300", "US", ""},
		{"011 39 02-36618 300", "US", "+390236618300"},
		{"(650) 253-0000", "US", "+16502530000"},
		{"200 123-0101", "US", "+12001230101"},
		{"2530000", "US", ""},
		{"020 7946 0018", "GB", "+442079460018"},
		{"20 7946 0018", "GB", "+442079460018"},
		{"+390236618300", "US", "+390236618300"},
		{"+390236618300", "XK", "+390236618300"},
		{"+80012345678", "IT", "+80012345678"},
		{"+390236618300", "it", ""},
		{"+390236618300", "ZZ", ""},
		{"+390236618300", "001", ""},
		{"+390236618300", "", ""},
		{"0236618300", "", ""},
		{"++390236618300", "IT", ""},
		{"(650) 253-0000 ext. 42", "US", ""},
		{"tel:0236618300;phone-context=+39", "IT", ""},
	}

	for _, test := range tests {
		t.Run(test.region+"/"+test.input, func(t *testing.T) {
			got, ok := NormalizePhoneInRegion(test.input, test.region)
			wantOK := test.want != ""
			if got != test.want || ok != wantOK {
				t.Fatalf("expected %q and %t, got %q and %t", test.want, wantOK, got, ok)
			}
			if ok {
				if canonical, valid := NormalizePhone(got); !valid || canonical != got {
					t.Fatalf("expected independently canonical output %q, got %q and valid=%t", got, canonical, valid)
				}
			}
		})
	}

}

// FuzzNormalizePhoneInRegion checks normalization with a region and canonical
// output stability.
func FuzzNormalizePhoneInRegion(f *testing.F) {
	for _, s := range []string{
		"02-36618 300",
		"340 1234567",
		"0039 02-36618 300",
		"+33 6 12 34 56 78",
		"2530000",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		canonical, ok := NormalizePhoneInRegion(s, "IT")
		if !ok {
			if canonical != "" {
				t.Fatalf("expected an empty result after failed normalization, got %q", canonical)
			}
			return
		}
		if again, valid := NormalizePhone(canonical); !valid || again != canonical {
			t.Fatalf(
				"expected regional result %q to normalize to itself and true, got %q and %t for input %q",
				canonical, again, valid, s,
			)
		}
	})
}

// Test_NormalizeUUID checks that UUIDs are canonicalized and invalid or
// non-standard forms are rejected.
func Test_NormalizeUUID(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		id, ok := NormalizeUUID("F47AC10B-58CC-4372-A567-0E02B2C3D479")
		if !ok || id != "f47ac10b-58cc-4372-a567-0e02b2c3d479" {
			t.Fatalf("unexpected result %q %t", id, ok)
		}
	})
	t.Run("invalid", func(t *testing.T) {
		if id, ok := NormalizeUUID("invalid"); ok || id != "" {
			t.Fatalf("expected failure, got %q %t", id, ok)
		}
	})
	t.Run("malformed", func(t *testing.T) {
		if id, ok := NormalizeUUID("F47AC10B-58CC-4372-A567-0E02B2C3D47Z"); ok || id != "" {
			t.Fatalf("expected failure, got %q %t", id, ok)
		}
	})
	t.Run("non-standard", func(t *testing.T) {
		for _, s := range []string{
			"F47AC10B58CC4372A5670E02B2C3D479",
			"{F47AC10B-58CC-4372-A567-0E02B2C3D479}",
			"urn:uuid:F47AC10B-58CC-4372-A567-0E02B2C3D479",
		} {
			if id, ok := NormalizeUUID(s); ok || id != "" {
				t.Fatalf("expected failure for %q, got %q %t", s, id, ok)
			}
		}
	})
}

// requireMarshalValidateError verifies that MarshalValidate failed without
// returning partial JSON, and that the validation error has the expected kind
// and path. If contains is not empty, the error message must contain it too.
func requireMarshalValidateError(
	t *testing.T,
	got json.Value,
	err error,
	kind schemaValidationKind,
	path string,
	contains string,
) {
	t.Helper()

	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if got != nil {
		t.Fatalf("expected nil JSON on error, got %q", got)
	}

	validation, ok := errors.AsType[*SchemaValidationError](err)
	if !ok {
		t.Fatalf("expected *SchemaValidationError, got %T: %v", err, err)
	}
	if validation.kind != kind {
		t.Fatalf("expected validation kind %v, got %v", kind, validation.kind)
	}
	if validation.path != path {
		t.Fatalf("expected validation path %q, got %q", path, validation.path)
	}
	if contains != "" && !strings.Contains(validation.msg, contains) {
		t.Fatalf(
			"expected validation message to contain %q, got %q",
			contains,
			validation.msg,
		)
	}
}

// requireSameValidationLocation verifies that MarshalValidate and Decode report
// the same validation kind at the same path. Their messages are allowed to
// differ because they validate different input representations.
func requireSameValidationLocation(t *testing.T, gotErr, decodeErr error) {
	t.Helper()

	got, ok := errors.AsType[*SchemaValidationError](gotErr)
	if !ok {
		t.Fatalf(
			"MarshalValidate returned %T: %v, want *SchemaValidationError",
			gotErr,
			gotErr,
		)
	}

	want, ok := errors.AsType[*SchemaValidationError](decodeErr)
	if !ok {
		t.Fatalf(
			"Decode returned %T: %v, want *SchemaValidationError",
			decodeErr,
			decodeErr,
		)
	}

	if got.kind != want.kind || got.path != want.path {
		t.Fatalf(
			"validation location differs: MarshalValidate kind=%v path=%q; Decode kind=%v path=%q",
			got.kind,
			got.path,
			want.kind,
			want.path,
		)
	}
}

// TestMarshalValidate checks that valid canonical Go values produce the same
// JSON as Marshal. It also verifies that decoding and marshaling the result
// again does not change the JSON.
func TestMarshalValidate(t *testing.T) {
	got, err := MarshalValidate(value, schema)
	if err != nil {
		t.Fatalf("MarshalValidate: unexpected error: %v", err)
	}

	want, err := Marshal(value, schema)
	if err != nil {
		t.Fatalf("Marshal: unexpected error: %v", err)
	}

	if !bytes.Equal(got, want) {
		t.Fatalf(
			"MarshalValidate differs from Marshal:\nwant: %s\ngot:  %s",
			want,
			got,
		)
	}

	decoded, err := Decode[map[string]any](bytes.NewReader(got), schema)
	if err != nil {
		t.Fatalf(
			"Decode(MarshalValidate(...)): unexpected error: %v",
			err,
		)
	}

	encodedAgain, err := Marshal(decoded, schema)
	if err != nil {
		t.Fatalf("Marshal(decoded): unexpected error: %v", err)
	}

	if !bytes.Equal(got, encodedAgain) {
		t.Fatalf(
			"JSON changed after MarshalValidate -> Decode -> Marshal:\nfirst:  %s\nsecond: %s",
			got,
			encodedAgain,
		)
	}
}

// TestMarshalValidateCanonicalRepresentation verifies that MarshalValidate
// accepts only the canonical Go values used for export. Values that import
// normalization could convert must be rejected instead of normalized here.
func TestMarshalValidateCanonicalRepresentation(t *testing.T) {
	optionalObject := Object([]Property{{
		Name:         "a",
		Type:         String(),
		ReadOptional: true,
	}})

	tests := []struct {
		name  string
		typ   Type
		value any
		valid bool
	}{
		// String values.
		{"string", String(), "hello", true},
		{"string as bytes", String(), []byte("hello"), false},

		// Boolean values.
		{"boolean", Boolean(), true, true},
		{"boolean as string", Boolean(), "true", false},

		// Signed integers.
		{"signed int", Int(32), int(12), true},
		{"signed int as int8", Int(32), int8(12), false},
		{"signed int as int16", Int(32), int16(12), false},
		{"signed int as int32", Int(32), int32(12), false},
		{"signed int as int64", Int(32), int64(12), false},
		{"signed int as uint", Int(32), uint(12), false},
		{"signed int as float64", Int(32), float64(12), false},
		{"signed int as string", Int(32), "12", false},

		// Unsigned integers.
		{"unsigned int", Int(32).Unsigned(), uint(12), true},
		{"unsigned int as int", Int(32).Unsigned(), int(12), false},
		{"unsigned int as uint8", Int(32).Unsigned(), uint8(12), false},
		{"unsigned int as uint32", Int(32).Unsigned(), uint32(12), false},
		{"unsigned int as string", Int(32).Unsigned(), "12", false},

		// Floating-point values.
		{"float64", Float(64), float64(1.25), true},
		{"float32 as float32", Float(32), float32(1.25), false},
		{"float as int", Float(64), int(1), false},
		{
			"canonical float32 carried by float64",
			Float(32),
			float64(float32(57.16038)),
			true,
		},
		{
			"non-canonical float32 carried by float64",
			Float(32),
			float64(57.16038),
			false,
		},
		{"float NaN", Float(64), math.NaN(), true},
		{"float positive infinity", Float(64), math.Inf(1), true},
		{"float negative infinity", Float(64), math.Inf(-1), true},

		// Decimal values.
		{
			"decimal",
			Decimal(10, 3),
			decimal.MustParse("1752.064"),
			true,
		},
		{"decimal as string", Decimal(10, 3), "1752.064", false},
		{
			"decimal as float64",
			Decimal(10, 3),
			float64(1752.064),
			false,
		},

		// Datetime values.
		{
			"datetime",
			DateTime(),
			time.Date(
				2026, 9, 20,
				12, 30, 45, 123456789,
				time.UTC,
			),
			true,
		},
		{
			"datetime with zero-offset non-UTC location",
			DateTime(),
			time.Date(
				2026, 9, 20,
				12, 30, 45, 0,
				time.FixedZone("UTC", 0),
			),
			false,
		},
		{
			"datetime with non-zero offset",
			DateTime(),
			time.Date(
				2026, 9, 20,
				12, 30, 45, 0,
				time.FixedZone("CEST", 2*60*60),
			),
			false,
		},
		{
			"datetime as string",
			DateTime(),
			"2026-09-20T12:30:45Z",
			false,
		},

		// Date values.
		{
			"date",
			Date(),
			time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC),
			true,
		},
		{
			"date with time component",
			Date(),
			time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC),
			false,
		},
		{
			"date with zero-offset non-UTC location",
			Date(),
			time.Date(
				2026, 9, 20,
				0, 0, 0, 0,
				time.FixedZone("UTC", 0),
			),
			false,
		},
		{"date as string", Date(), "2026-09-20", false},

		// Time values.
		{
			"time",
			Time(),
			time.Date(
				1970, 1, 1,
				12, 30, 45, 123456789,
				time.UTC,
			),
			true,
		},
		{
			"time with wrong date",
			Time(),
			time.Date(2026, 1, 1, 12, 30, 45, 0, time.UTC),
			false,
		},
		{
			"time with zero-offset non-UTC location",
			Time(),
			time.Date(
				1970, 1, 1,
				12, 30, 45, 0,
				time.FixedZone("UTC", 0),
			),
			false,
		},
		{"time as string", Time(), "12:30:45", false},

		// Year values.
		{"year", Year(), int(2026), true},
		{"year as int32", Year(), int32(2026), false},
		{"year as string", Year(), "2026", false},

		// UUID values.
		{
			"uuid",
			UUID(),
			"550e8400-e29b-41d4-a716-446655440000",
			true,
		},
		{
			"uuid uppercase",
			UUID(),
			"550E8400-E29B-41D4-A716-446655440000",
			false,
		},
		{
			"uuid as bytes",
			UUID(),
			[]byte("550e8400-e29b-41d4-a716-446655440000"),
			false,
		},

		// JSON values.
		{"JSON object", JSON(), json.Value(`{"a":1}`), true},
		{"JSON array", JSON(), json.Value(`[1,2,3]`), true},
		{"JSON string", JSON(), json.Value(`"hello"`), true},
		{"JSON number", JSON(), json.Value(`12.5`), true},
		{"JSON boolean", JSON(), json.Value(`true`), true},
		{"JSON null", JSON(), json.Value(`null`), true},
		{"JSON typed nil", JSON(), json.Value(nil), false},
		{"JSON as bytes", JSON(), []byte(`{"a":1}`), false},
		{"JSON as Go string", JSON(), `{"a":1}`, false},

		// IP values.
		{"IPv4", IP(), "192.0.2.1", true},
		{"canonical IPv6", IP(), "2001:db8::1", true},
		{
			"expanded IPv6",
			IP(),
			"2001:0db8:0000:0000:0000:0000:0000:0001",
			false,
		},
		{
			"IPv4-mapped IPv6",
			IP(),
			"::ffff:192.0.2.1",
			false,
		},
		{
			"IP as netip.Addr",
			IP(),
			netip.MustParseAddr("192.0.2.1"),
			false,
		},

		// Arrays.
		{
			"array",
			Array(String()),
			[]any{"a", "b"},
			true,
		},
		{
			"array as []string",
			Array(String()),
			[]string{"a", "b"},
			false,
		},
		{
			"array typed nil",
			Array(String()),
			[]any(nil),
			false,
		},

		// Objects.
		{
			"object",
			Object([]Property{{Name: "a", Type: Int(32)}}),
			map[string]any{"a": int(1)},
			true,
		},
		{
			"object typed nil",
			optionalObject,
			map[string]any(nil),
			false,
		},
		{
			"object as struct",
			optionalObject,
			struct{ A string }{A: "a"},
			false,
		},

		// Maps.
		{
			"map",
			Map(Int(32)),
			map[string]any{"a": int(1)},
			true,
		},
		{
			"map as map[string]int",
			Map(Int(32)),
			map[string]int{"a": 1},
			false,
		},
		{
			"map typed nil",
			Map(Int(32)),
			map[string]any(nil),
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := MarshalValidate(test.value, test.typ)

			if !test.valid {
				requireMarshalValidateError(
					t,
					got,
					err,
					invalidValue,
					"",
					"",
				)
				return
			}

			if err != nil {
				t.Fatalf(
					"expected no error, got %T: %v",
					err,
					err,
				)
			}
			if !json.Valid(got) {
				t.Fatalf(
					"MarshalValidate returned invalid JSON: %q",
					got,
				)
			}

			want, err := Marshal(test.value, test.typ)
			if err != nil {
				t.Fatalf(
					"Marshal: unexpected error: %v",
					err,
				)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf(
					"MarshalValidate differs from Marshal: want %s, got %s",
					want,
					got,
				)
			}
		})
	}
}

// TestMarshalValidateConstraints verifies the constraints carried by Type once
// the input is already using the canonical Go representation.
func TestMarshalValidateConstraints(t *testing.T) {
	tests := []struct {
		name     string
		typ      Type
		value    any
		valid    bool
		path     string
		contains string
	}{
		// Allowed string values.
		{
			"string allowed value",
			String().WithValues("a", "b", "c"),
			"b",
			true,
			"",
			"",
		},
		{
			"string unsupported value",
			String().WithValues("a", "b", "c"),
			"d",
			false,
			"",
			"invalid value",
		},
		{
			"empty allowed value",
			String().WithValues("", "a"),
			"",
			true,
			"",
			"",
		},

		// String patterns.
		{
			"string regexp match",
			String().WithPattern(regexp.MustCompile(`oo$`)),
			"foo",
			true,
			"",
			"",
		},
		{
			"string regexp mismatch",
			String().WithPattern(regexp.MustCompile(`oo$`)),
			"faa",
			false,
			"",
			"regular expression",
		},

		// String length limits.
		{
			"string max characters boundary",
			String().WithMaxLength(2),
			"éé",
			true,
			"",
			"",
		},
		{
			"string too many characters",
			String().WithMaxLength(1),
			"éé",
			false,
			"",
			"longer than",
		},
		{
			"string max bytes boundary",
			String().WithMaxBytes(4),
			"éé",
			true,
			"",
			"",
		},
		{
			"string too many bytes",
			String().WithMaxBytes(3),
			"éé",
			false,
			"",
			"longer than",
		},
		{
			"invalid UTF-8 string",
			String(),
			string([]byte{0xff}),
			false,
			"",
			"UTF-8",
		},

		// Signed integer ranges.
		{
			"signed int minimum",
			Int(8).WithIntRange(-20, 20),
			int(-20),
			true,
			"",
			"",
		},
		{
			"signed int maximum",
			Int(8).WithIntRange(-20, 20),
			int(20),
			true,
			"",
			"",
		},
		{
			"signed int below minimum",
			Int(8).WithIntRange(-20, 20),
			int(-21),
			false,
			"",
			"out of range",
		},
		{
			"signed int above maximum",
			Int(8).WithIntRange(-20, 20),
			int(21),
			false,
			"",
			"out of range",
		},

		// Unsigned integer ranges.
		{
			"unsigned int minimum",
			Int(8).Unsigned().WithUnsignedRange(10, 20),
			uint(10),
			true,
			"",
			"",
		},
		{
			"unsigned int maximum",
			Int(8).Unsigned().WithUnsignedRange(10, 20),
			uint(20),
			true,
			"",
			"",
		},
		{
			"unsigned int below minimum",
			Int(8).Unsigned().WithUnsignedRange(10, 20),
			uint(9),
			false,
			"",
			"out of range",
		},
		{
			"unsigned int above maximum",
			Int(8).Unsigned().WithUnsignedRange(10, 20),
			uint(21),
			false,
			"",
			"out of range",
		},

		// Float ranges and real-only values.
		{
			"float minimum",
			Float(64).WithFloatRange(-20.5, 8),
			float64(-20.5),
			true,
			"",
			"",
		},
		{
			"float maximum",
			Float(64).WithFloatRange(-20.5, 8),
			float64(8),
			true,
			"",
			"",
		},
		{
			"float below minimum",
			Float(64).WithFloatRange(-20.5, 8),
			float64(-20.5001),
			false,
			"",
			"out of range",
		},
		{
			"float above maximum",
			Float(64).WithFloatRange(-20.5, 8),
			float64(8.0001),
			false,
			"",
			"out of range",
		},
		{
			"real finite",
			Float(64).Real(),
			float64(1.5),
			true,
			"",
			"",
		},
		{
			"real NaN",
			Float(64).Real(),
			math.NaN(),
			false,
			"",
			"not a real",
		},
		{
			"real positive infinity",
			Float(64).Real(),
			math.Inf(1),
			false,
			"",
			"not a real",
		},
		{
			"real negative infinity",
			Float(64).Real(),
			math.Inf(-1),
			false,
			"",
			"not a real",
		},

		// Decimal precision, scale and ranges.
		{
			"decimal valid scale",
			Decimal(5, 2),
			decimal.MustParse("123.45"),
			true,
			"",
			"",
		},
		{
			"decimal unrepresentable scale",
			Decimal(5, 2),
			decimal.MustParse("1.234"),
			false,
			"",
			"",
		},
		{
			"decimal precision overflow",
			Decimal(5, 2),
			decimal.MustParse("1000.00"),
			false,
			"",
			"out of range",
		},
		{
			"decimal custom minimum",
			Decimal(5, 2).WithDecimalRange(
				decimal.MustParse("-10.50"),
				decimal.MustParse("8.25"),
			),
			decimal.MustParse("-10.50"),
			true,
			"",
			"",
		},
		{
			"decimal custom maximum",
			Decimal(5, 2).WithDecimalRange(
				decimal.MustParse("-10.50"),
				decimal.MustParse("8.25"),
			),
			decimal.MustParse("8.25"),
			true,
			"",
			"",
		},
		{
			"decimal outside custom range",
			Decimal(5, 2).WithDecimalRange(
				decimal.MustParse("-10.50"),
				decimal.MustParse("8.25"),
			),
			decimal.MustParse("8.26"),
			false,
			"",
			"out of range",
		},

		// Year boundaries.
		{
			"minimum year",
			Year(),
			int(MinYear),
			true,
			"",
			"",
		},
		{
			"maximum year",
			Year(),
			int(MaxYear),
			true,
			"",
			"",
		},
		{
			"year below minimum",
			Year(),
			int(0),
			false,
			"",
			"out of range",
		},
		{
			"year above maximum",
			Year(),
			int(10000),
			false,
			"",
			"out of range",
		},

		// Datetime and date year boundaries.
		{
			"minimum datetime year",
			DateTime(),
			time.Date(
				MinYear, 1, 1,
				0, 0, 0, 0,
				time.UTC,
			),
			true,
			"",
			"",
		},
		{
			"maximum datetime year",
			DateTime(),
			time.Date(
				MaxYear, 12, 31,
				23, 59, 59, 999999999,
				time.UTC,
			),
			true,
			"",
			"",
		},
		{
			"datetime year zero",
			DateTime(),
			time.Date(
				0, 1, 1,
				0, 0, 0, 0,
				time.UTC,
			),
			false,
			"",
			"year",
		},
		{
			"datetime year 10000",
			DateTime(),
			time.Date(
				10000, 1, 1,
				0, 0, 0, 0,
				time.UTC,
			),
			false,
			"",
			"year",
		},
		{
			"minimum date year",
			Date(),
			time.Date(
				MinYear, 1, 1,
				0, 0, 0, 0,
				time.UTC,
			),
			true,
			"",
			"",
		},
		{
			"maximum date year",
			Date(),
			time.Date(
				MaxYear, 12, 31,
				0, 0, 0, 0,
				time.UTC,
			),
			true,
			"",
			"",
		},

		// String semantics currently checked by Decode.
		{
			"country alpha-2",
			String().AsCountry(ISO3166Alpha2),
			"IT",
			true,
			"",
			"",
		},
		{
			"country lowercase",
			String().AsCountry(ISO3166Alpha2),
			"it",
			false,
			"",
			"country code",
		},
		{
			"country unknown",
			String().AsCountry(ISO3166Alpha2),
			"ZZ",
			false,
			"",
			"country code",
		},
		{
			"country alpha-3",
			String().AsCountry(ISO3166Alpha3),
			"ITA",
			true,
			"",
			"",
		},
		{
			"phone canonical",
			String().AsPhone(),
			"+390236618300",
			true,
			"",
			"",
		},
		{
			"phone structurally possible",
			String().AsPhone(),
			"+12001230101",
			true,
			"",
			"",
		},
		{
			"phone formatted but non-canonical",
			String().AsPhone(),
			"+39 02-36618 300",
			false,
			"",
			"canonical phone",
		},
		{
			"phone local-only",
			String().AsPhone(),
			"+12530000",
			false,
			"",
			"canonical phone",
		},

		// JSON syntax.
		{
			"valid arbitrary JSON",
			JSON(),
			json.Value(`{"a":[1,true,null]}`),
			true,
			"",
			"",
		},
		{
			"invalid JSON",
			JSON(),
			json.Value(`{"a":`),
			false,
			"",
			"valid JSON",
		},

		// Array size constraints.
		{
			"array minimum",
			Array(Int(32)).
				WithMinElements(2).
				WithMaxElements(3),
			[]any{int(1), int(2)},
			true,
			"",
			"",
		},
		{
			"array maximum",
			Array(Int(32)).
				WithMinElements(2).
				WithMaxElements(3),
			[]any{int(1), int(2), int(3)},
			true,
			"",
			"",
		},
		{
			"array below minimum",
			Array(Int(32)).
				WithMinElements(2).
				WithMaxElements(3),
			[]any{int(1)},
			false,
			"",
			"less than 2 elements",
		},
		{
			"array above maximum",
			Array(Int(32)).
				WithMinElements(2).
				WithMaxElements(3),
			[]any{int(1), int(2), int(3), int(4)},
			false,
			"",
			"more than 3 elements",
		},

		// Array uniqueness.
		{
			"unique array",
			Array(Int(32)).WithUnique(),
			[]any{int(1), int(2), int(3)},
			true,
			"",
			"",
		},
		{
			"duplicate array",
			Array(Int(32)).WithUnique(),
			[]any{int(1), int(2), int(2)},
			false,
			"[2]",
			"duplicates",
		},
		{
			"duplicate NaN",
			Array(Float(64)).WithUnique(),
			[]any{math.NaN(), math.NaN()},
			false,
			"[1]",
			"duplicates",
		},

		// Map keys.
		{
			"valid map key",
			Map(Int(32)),
			map[string]any{"a": int(1)},
			true,
			"",
			"",
		},
		{
			"invalid UTF-8 map key",
			Map(Int(32)),
			map[string]any{
				string([]byte{0xff}): int(1),
			},
			false,
			"",
			"UTF-8",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := MarshalValidate(test.value, test.typ)

			if !test.valid {
				requireMarshalValidateError(
					t,
					got,
					err,
					invalidValue,
					test.path,
					test.contains,
				)
				return
			}

			if err != nil {
				t.Fatalf(
					"expected no error, got %T: %v",
					err,
					err,
				)
			}
			if !json.Valid(got) {
				t.Fatalf(
					"MarshalValidate returned invalid JSON: %q",
					got,
				)
			}

			want, err := Marshal(test.value, test.typ)
			if err != nil {
				t.Fatalf(
					"Marshal: unexpected error: %v",
					err,
				)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf(
					"MarshalValidate differs from Marshal: want %s, got %s",
					want,
					got,
				)
			}
		})
	}
}

// TestMarshalValidateNullability covers nil values, typed nils, nullable
// properties, and the difference between a nil json.Value and JSON null.
func TestMarshalValidateNullability(t *testing.T) {
	optionalObject := Object([]Property{{
		Name:         "a",
		Type:         String(),
		ReadOptional: true,
	}})

	tests := []struct {
		name     string
		typ      Type
		nullable bool
		value    any
		valid    bool
	}{
		{
			"nullable string nil",
			String(),
			true,
			nil,
			true,
		},
		{
			"non-nullable string nil",
			String(),
			false,
			nil,
			false,
		},

		{
			"nullable JSON nil",
			JSON(),
			true,
			nil,
			true,
		},
		{
			"nullable JSON typed nil",
			JSON(),
			true,
			json.Value(nil),
			true,
		},
		{
			"non-nullable JSON nil",
			JSON(),
			false,
			nil,
			false,
		},
		{
			"non-nullable JSON typed nil",
			JSON(),
			false,
			json.Value(nil),
			false,
		},
		{
			"non-nullable JSON null",
			JSON(),
			false,
			json.Value(`null`),
			true,
		},

		{
			"nullable array typed nil",
			Array(String()),
			true,
			[]any(nil),
			true,
		},
		{
			"non-nullable array typed nil",
			Array(String()),
			false,
			[]any(nil),
			false,
		},

		{
			"nullable object typed nil",
			optionalObject,
			true,
			map[string]any(nil),
			true,
		},
		{
			"non-nullable object typed nil",
			optionalObject,
			false,
			map[string]any(nil),
			false,
		},

		{
			"nullable map typed nil",
			Map(String()),
			true,
			map[string]any(nil),
			true,
		},
		{
			"non-nullable map typed nil",
			Map(String()),
			false,
			map[string]any(nil),
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			typ := Object([]Property{{
				Name:     "value",
				Type:     test.typ,
				Nullable: test.nullable,
			}})

			got, err := MarshalValidate(
				map[string]any{"value": test.value},
				typ,
			)

			if test.valid {
				if err != nil {
					t.Fatalf(
						"expected no error, got %T: %v",
						err,
						err,
					)
				}
				if string(got) != `{"value":null}` {
					t.Fatalf(
						`expected {"value":null}, got %s`,
						got,
					)
				}
				return
			}

			requireMarshalValidateError(
				t,
				got,
				err,
				invalidValue,
				"value",
				"cannot be null",
			)
		})
	}

	t.Run("top-level nil", func(t *testing.T) {
		got, err := MarshalValidate(nil, String())

		requireMarshalValidateError(
			t,
			got,
			err,
			invalidValue,
			"",
			"cannot be null",
		)
	})
}

// TestMarshalValidateObjectsAndPaths covers object membership, required and
// optional properties, nil elements in containers, and nested error paths.
func TestMarshalValidateObjectsAndPaths(t *testing.T) {
	t.Run("unknown property", func(t *testing.T) {
		typ := Object([]Property{{
			Name: "a",
			Type: Int(32),
		}})

		got, err := MarshalValidate(
			map[string]any{
				"a":     int(1),
				"extra": int(2),
			},
			typ,
		)

		requireMarshalValidateError(
			t,
			got,
			err,
			propertyNotExist,
			"extra",
			"",
		)
	})

	t.Run("missing required property", func(t *testing.T) {
		typ := Object([]Property{{
			Name: "a",
			Type: Int(32),
		}})

		got, err := MarshalValidate(
			map[string]any{},
			typ,
		)

		requireMarshalValidateError(
			t,
			got,
			err,
			missingProperty,
			"a",
			"",
		)
	})

	t.Run("missing optional property", func(t *testing.T) {
		typ := Object([]Property{{
			Name:         "a",
			Type:         Int(32),
			ReadOptional: true,
		}})

		got, err := MarshalValidate(
			map[string]any{},
			typ,
		)
		if err != nil {
			t.Fatalf(
				"expected no error, got %v",
				err,
			)
		}
		if string(got) != `{}` {
			t.Fatalf(
				"expected {}, got %s",
				got,
			)
		}
	})

	t.Run("nested invalid value path", func(t *testing.T) {
		typ := Object([]Property{{
			Name: "accounts",
			Type: Array(Object([]Property{{
				Name: "age",
				Type: Int(8),
			}})),
		}})

		input := map[string]any{
			"accounts": []any{
				map[string]any{
					"age": int(20),
				},
				map[string]any{
					"age": int(300),
				},
			},
		}

		got, err := MarshalValidate(input, typ)

		requireMarshalValidateError(
			t,
			got,
			err,
			invalidValue,
			"accounts[1].age",
			"out of range",
		)
	})

	t.Run("nested missing property path", func(t *testing.T) {
		typ := Object([]Property{{
			Name: "profile",
			Type: Object([]Property{{
				Name: "age",
				Type: Int(8),
			}}),
		}})

		got, err := MarshalValidate(
			map[string]any{
				"profile": map[string]any{},
			},
			typ,
		)

		requireMarshalValidateError(
			t,
			got,
			err,
			missingProperty,
			"profile.age",
			"",
		)
	})

	t.Run("nested unknown property path", func(t *testing.T) {
		typ := Object([]Property{{
			Name: "profile",
			Type: Object([]Property{{
				Name: "age",
				Type: Int(8),
			}}),
		}})

		got, err := MarshalValidate(
			map[string]any{
				"profile": map[string]any{
					"age":   int(20),
					"extra": true,
				},
			},
			typ,
		)

		requireMarshalValidateError(
			t,
			got,
			err,
			propertyNotExist,
			"profile.extra",
			"",
		)
	})

	t.Run("array element type path", func(t *testing.T) {
		got, err := MarshalValidate(
			[]any{"a", int(1)},
			Array(String()),
		)

		requireMarshalValidateError(
			t,
			got,
			err,
			invalidValue,
			"[1]",
			"",
		)
	})

	t.Run("array nil element", func(t *testing.T) {
		got, err := MarshalValidate(
			[]any{"a", nil},
			Array(String()),
		)

		requireMarshalValidateError(
			t,
			got,
			err,
			invalidValue,
			"[1]",
			"cannot be null",
		)
	})

	t.Run("JSON null array element", func(t *testing.T) {
		got, err := MarshalValidate(
			[]any{
				json.Value(`null`),
			},
			Array(JSON()),
		)
		if err != nil {
			t.Fatalf(
				"expected no error, got %v",
				err,
			)
		}
		if string(got) != `[null]` {
			t.Fatalf(
				"expected [null], got %s",
				got,
			)
		}
	})

	t.Run("nil JSON array element", func(t *testing.T) {
		got, err := MarshalValidate(
			[]any{nil},
			Array(JSON()),
		)

		requireMarshalValidateError(
			t,
			got,
			err,
			invalidValue,
			"[0]",
			"cannot be null",
		)
	})

	t.Run("map nil element", func(t *testing.T) {
		got, err := MarshalValidate(
			map[string]any{
				"a": nil,
			},
			Map(String()),
		)

		requireMarshalValidateError(
			t,
			got,
			err,
			invalidValue,
			"",
			"cannot be null",
		)
	})

	t.Run("map element keeps Decode path semantics", func(t *testing.T) {
		typ := Object([]Property{{
			Name: "labels",
			Type: Map(Int(32)),
		}})

		got, err := MarshalValidate(
			map[string]any{
				"labels": map[string]any{
					"a": "1",
				},
			},
			typ,
		)

		requireMarshalValidateError(
			t,
			got,
			err,
			invalidValue,
			"labels",
			"",
		)
	})
}

// TestMarshalValidateMatchesDecodeValidation verifies that MarshalValidate and
// Decode point to the same schema error when they are validating the same
// logical value. Non-canonical Go representations are covered separately,
// because Decode is allowed to normalize input while MarshalValidate is not.
func TestMarshalValidateMatchesDecodeValidation(t *testing.T) {
	tests := []struct {
		name   string
		typ    Type
		value  any
		source string
	}{
		{
			"string values",
			String().WithValues("a", "b"),
			"c",
			`"c"`,
		},
		{
			"string pattern",
			String().WithPattern(regexp.MustCompile(`oo$`)),
			"bar",
			`"bar"`,
		},
		{
			"string max length",
			String().WithMaxLength(1),
			"ab",
			`"ab"`,
		},
		{
			"country",
			String().AsCountry(ISO3166Alpha2),
			"ZZ",
			`"ZZ"`,
		},
		{
			"phone",
			String().AsPhone(),
			"+39 02-36618 300",
			`"+39 02-36618 300"`,
		},
		{
			"signed range",
			Int(8).WithIntRange(-20, 20),
			int(21),
			`21`,
		},
		{
			"unsigned range",
			Int(8).Unsigned().WithUnsignedRange(10, 20),
			uint(9),
			`9`,
		},
		{
			"float range",
			Float(64).WithFloatRange(-20.5, 8),
			float64(8.1),
			`8.1`,
		},
		{
			"real NaN",
			Float(64).Real(),
			math.NaN(),
			`"NaN"`,
		},
		{
			"decimal range",
			Decimal(5, 2).WithDecimalRange(
				decimal.MustParse("-10.50"),
				decimal.MustParse("8.25"),
			),
			decimal.MustParse("8.26"),
			`8.26`,
		},
		{
			"year",
			Year(),
			int(10000),
			`10000`,
		},
		{
			"array minimum",
			Array(Int(32)).WithMinElements(2),
			[]any{int(1)},
			`[1]`,
		},
		{
			"array maximum",
			Array(Int(32)).WithMaxElements(2),
			[]any{int(1), int(2), int(3)},
			`[1,2,3]`,
		},
		{
			"array unique",
			Array(Int(32)).WithUnique(),
			[]any{int(1), int(2), int(2)},
			`[1,2,2]`,
		},
		{
			"nested invalid value",
			Object([]Property{{
				Name: "accounts",
				Type: Array(Object([]Property{{
					Name: "age",
					Type: Int(8),
				}})),
			}}),
			map[string]any{
				"accounts": []any{
					map[string]any{
						"age": int(300),
					},
				},
			},
			`{"accounts":[{"age":300}]}`,
		},
		{
			"unknown property",
			Object([]Property{{
				Name: "a",
				Type: Int(32),
			}}),
			map[string]any{
				"a":     int(1),
				"extra": int(2),
			},
			`{"a":1,"extra":2}`,
		},
		{
			"missing property",
			Object([]Property{{
				Name: "a",
				Type: Int(32),
			}}),
			map[string]any{},
			`{}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, gotErr := MarshalValidate(
				test.value,
				test.typ,
			)
			if gotErr == nil {
				t.Fatalf(
					"MarshalValidate returned %s, want validation error",
					got,
				)
			}
			if got != nil {
				t.Fatalf(
					"MarshalValidate returned partial JSON %q with error",
					got,
				)
			}

			_, decodeErr := Decode[any](
				strings.NewReader(test.source),
				test.typ,
			)
			if decodeErr == nil {
				t.Fatal(
					"Decode returned no error for reference invalid value",
				)
			}

			requireSameValidationLocation(
				t,
				gotErr,
				decodeErr,
			)
		})
	}
}

// TestMarshalValidateDoesNotModifyInput verifies that validation leaves the
// caller's values untouched, including JSON formatting and unknown object
// properties.
func TestMarshalValidateDoesNotModifyInput(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		typ := Object([]Property{
			{
				Name: "JSON",
				Type: JSON(),
			},
			{
				Name: "Array",
				Type: Array(Object([]Property{{
					Name: "name",
					Type: String(),
				}})),
			},
			{
				Name: "Map",
				Type: Map(Int(32)),
			},
		})

		input := map[string]any{
			"JSON": json.Value(
				`{ "a": 1, "b": [ true, null ] }`,
			),
			"Array": []any{
				map[string]any{
					"name": "Alice",
				},
			},
			"Map": map[string]any{
				"b": int(2),
				"a": int(1),
			},
		}

		want := map[string]any{
			"JSON": json.Value(
				`{ "a": 1, "b": [ true, null ] }`,
			),
			"Array": []any{
				map[string]any{
					"name": "Alice",
				},
			},
			"Map": map[string]any{
				"b": int(2),
				"a": int(1),
			},
		}

		_, err := MarshalValidate(input, typ)
		if err != nil {
			t.Fatalf(
				"expected no error, got %v",
				err,
			)
		}

		if !reflect.DeepEqual(input, want) {
			t.Fatalf(
				"MarshalValidate modified valid input:\nwant: %#v\ngot:  %#v",
				want,
				input,
			)
		}
	})

	t.Run("invalid object keeps unknown property", func(t *testing.T) {
		typ := Object([]Property{{
			Name: "known",
			Type: Int(32),
		}})

		input := map[string]any{
			"known": int(1),
			"extra": int(2),
		}
		want := maps.Clone(input)

		got, err := MarshalValidate(
			input,
			typ,
		)

		requireMarshalValidateError(
			t,
			got,
			err,
			propertyNotExist,
			"extra",
			"",
		)

		if !reflect.DeepEqual(input, want) {
			t.Fatalf(
				"MarshalValidate modified invalid input:\nwant: %#v\ngot:  %#v",
				want,
				input,
			)
		}
	})
}

// TestMarshalValidateSchemaErrors verifies that MarshalValidate enforces the
// same schema preconditions as Marshal.
func TestMarshalValidateSchemaErrors(t *testing.T) {
	tests := []struct {
		name string
		typ  Type
		want string
	}{
		{
			"invalid type",
			Type{},
			"json: schema is the invalid type",
		},
		{
			"generic type",
			Parameter("T"),
			"json: schema is a generic type",
		},
		{
			"nested generic type",
			Array(Parameter("T")),
			"json: schema is a generic type",
		},
		{
			"object containing generic type",
			Object([]Property{{
				Name: "value",
				Type: Parameter("T"),
			}}),
			"json: schema is a generic type",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := MarshalValidate(
				nil,
				test.typ,
			)

			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if got != nil {
				t.Fatalf(
					"expected nil JSON on error, got %q",
					got,
				)
			}
			if err.Error() != test.want {
				t.Fatalf(
					"expected error %q, got %q",
					test.want,
					err,
				)
			}
		})
	}
}
