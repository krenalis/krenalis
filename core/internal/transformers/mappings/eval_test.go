// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestErrorHelpers checks formatting of conversion error messages.
func TestErrorHelpers(t *testing.T) {
	bErr := errBooleanConversion("and", "x", "foo", types.String())
	if bErr.Error() != "«x» (type string) does not represent a boolean when passed to the «and» function" {
		t.Fatalf("unexpected boolean error: %v", bErr)
	}
	jb := json.Value("true")
	bErr = errBooleanConversion("and", "x", jb, types.JSON())
	if bErr.Error() != "«x», of type JSON true, cannot be passed as boolean to the «and» function" {
		t.Fatalf("unexpected boolean json error: %v", bErr)
	}
	iErr := errInt32Conversion("fn", "x", 5.5, types.Float(64))
	if iErr.Error() != "«x», with a value of 5.5, cannot be passed as a 32-bit int to the «fn» function" {
		t.Fatalf("unexpected int error: %v", iErr)
	}
	ji := json.Value("\"foo\"")
	iErr = errInt32Conversion("fn", "x", ji, types.JSON())
	if iErr.Error() != "«x», of type JSON string, cannot be passed as an int to the «fn» function" {
		t.Fatalf("unexpected int json error: %v", iErr)
	}
	tErr := errStringConversion("up", "x", json.Value("[1]"))
	if tErr.Error() != "«x» (a JSON array) cannot be converted to a string value to be passed to the «up» function" {
		t.Fatalf("unexpected string error: %v", tErr)
	}
}

// Test_appendAsString ensures values are appended correctly as strings.
func Test_appendAsString(t *testing.T) {
	t0 := time.Date(2023, 1, 2, 3, 4, 5, 0, time.UTC)
	tests := []struct {
		v   any
		typ types.Type
		out string
		err error
	}{
		{nil, types.String(), "start", nil},
		{"foo", types.String(), "startfoo", nil},
		{true, types.Boolean(), "starttrue", nil},
		{false, types.Boolean(), "startfalse", nil},
		{int(3), types.Int(32), "start3", nil},
		{uint(4), types.Int(16).Unsigned(), "start4", nil},
		{1.5, types.Float(64), "start1.5", nil},
		{decimal.MustParse("2.7"), types.Decimal(2, 1), "start2.7", nil},
		{t0, types.DateTime(), "start2023-01-02T03:04:05Z", nil},
		{t0, types.Date(), "start2023-01-02", nil},
		{t0, types.Time(), "start03:04:05", nil},
		{json.Value("\"bar\""), types.JSON(), "startbar", nil},
		{json.Value("123"), types.JSON(), "start123", nil},
		{json.Value("true"), types.JSON(), "starttrue", nil},
		{json.Value("null"), types.JSON(), "startnull", nil},
		{json.Value("[1,2]"), types.JSON(), "start", errInvalidConversion},
	}
	for _, tt := range tests {
		buf, err := appendAsString([]byte("start"), tt.v, tt.typ)
		if err != tt.err {
			t.Fatalf("%v: expected err %v, got %v", tt.v, tt.err, err)
		}
		if string(buf) != tt.out {
			t.Fatalf("%v: expected %q, got %q", tt.v, tt.out, string(buf))
		}
	}
}
func Test_digitCountInt(t *testing.T) {

	tests := []struct {
		n        int64
		expected int
	}{
		{0, 1},
		{10, 2},
		{7940200381, 10},
		{-1, 2},
		{-2817482, 8},
		{9223372036854775807, 19},
		{-9223372036854775807, 20},
		{-9223372036854775808, 20},
	}

	for _, test := range tests {
		got := digitCountInt(test.n)
		if test.expected != got {
			t.Fatalf("%d: expected %d, got %d", test.n, test.expected, got)
		}
	}

}

func Test_digitCountUint(t *testing.T) {

	tests := []struct {
		n        uint64
		expected int
	}{
		{0, 1},
		{10, 2},
		{63471038, 8},
		{18446744073709551615, 20},
	}

	for _, test := range tests {
		got := digitCountUint(test.n)
		if test.expected != got {
			t.Fatalf("%d: expected %d, got %d", test.n, test.expected, got)
		}
	}

}

func Test_eval(t *testing.T) {

	attributes := map[string]any{
		"a": 165,
		"b": map[string]any{
			"c": "foo",
			"e": 1024,
		},
		"d": nil,
	}
	n := decimal.MustInt(5)
	dt := types.Decimal(types.MaxDecimalPrecision, types.MaxDecimalScale)

	tests := []struct {
		expr          []part
		expectedValue any
		expectedType  types.Type
		err           error
	}{
		{[]part{{value: ``, typ: types.String()}}, "", types.String(), nil},
		{[]part{{value: `a`, typ: types.String()}}, "a", types.String(), nil},
		{[]part{{value: n, typ: dt}}, n, dt, nil},
		{[]part{{path: path{elements: []string{"a"}, decorators: []decorators{0}}, typ: types.Int(32)}}, 165, types.Int(32), nil},
		{[]part{{path: path{elements: []string{"b", "c"}, decorators: []decorators{0, 0}}, typ: types.String()}}, "foo", types.String(), nil},
		{[]part{{path: path{elements: []string{"b", "e"}, decorators: []decorators{0, 0}}, typ: types.Int(32)}}, 1024, types.Int(32), nil},
		{[]part{{value: `a`, path: path{elements: []string{"a"}, decorators: []decorators{0}}, typ: types.Int(32)}}, "a165", types.String(), nil},
		{[]part{{path: path{elements: []string{"coalesce"}, decorators: []decorators{0}}, args: [][]part{
			{{path: path{elements: []string{"a"}, decorators: []decorators{0}}, typ: types.Int(32)}, {value: " boo", typ: types.String()}},
			{{value: "foo", typ: types.String()}},
		}}}, "165 boo", types.String(), nil},
		{[]part{{path: path{elements: []string{"coalesce"}, decorators: []decorators{0}}, args: [][]part{
			{{path: path{elements: []string{"d"}, decorators: []decorators{0}}, typ: types.String()}},
			{{path: path{elements: []string{"a"}, decorators: []decorators{0}}, typ: types.Int(32)}, {value: " boo", typ: types.String()}},
		}}}, "165 boo", types.String(), nil},
		{[]part{{value: ``, typ: types.String()}, {path: path{elements: []string{"a"}, decorators: []decorators{0}}, typ: types.Int(32)}}, "165", types.String(), nil},
		{[]part{{path: path{elements: []string{"x"}, decorators: []decorators{0}}, typ: types.Boolean()}}, nil, types.Type{}, nil},
		{[]part{{path: path{elements: []string{"b", "x"}, decorators: []decorators{0, 0}}, typ: types.Boolean()}}, nil, types.Boolean(), nil},
	}

	for i, test := range tests {
		got, typ, err := eval(test.expr, "", attributes)
		if err != nil {
			if test.err == nil {
				t.Fatalf("%d. unexpected error: %s", i+1, err)
			}
			if err.Error() != test.err.Error() {
				t.Fatalf("%d. expected error %q, got error %q", i+1, test.err.Error(), err.Error())
			}
			continue
		}
		if test.err != nil {
			t.Fatalf("%d. expected error %q, got no error", i+1, test.err)
		}
		if !reflect.DeepEqual(got, test.expectedValue) {
			t.Fatalf("%d. unexpected value\nexpected %#v\ngot      %#v", i+1, test.expectedValue, got)
		}
		if !types.Equal(typ, test.expectedType) {
			if typ.Valid() {
				t.Fatalf("%d. expected type %s, got %s", i+1, test.expectedType, typ)
			}
			t.Fatalf("%d. expected type %s, got invalid type", i+1, test.expectedType)
		}
	}

}

func Test_substring(t *testing.T) {

	tests := []struct {
		s        string
		start    int
		length   int
		expected string
	}{
		{"", 1, 0, ""},
		{"", 5, 3, ""},
		{"a", 1, 1, "a"},
		{"a", 1, -1, "a"},
		{"a", 2, -1, ""},
		{"hello world", 1, 0, ""},
		{"hello world", 1, -1, "hello world"},
		{"hello world", 1, 5, "hello"},
		{"hello world", 7, 5, "world"},
		{"hello world", 2, 3, "ell"},
		{"hello world", 1, 20, "hello world"},
		{"hello world", 1, 1, "h"},
		{"hello world", 11, 1, "d"},
		{"hello world", 11, 20, "d"},
		{"hello world", 12, 5, ""},
		{"hello world", 3, -1, "llo world"},
		{"hello world", 3, 9, "llo world"},
		{"hello world", 12, -1, ""},
		{"hello world", 50, -1, ""},
		{"The café is ready for lunch at the résidence", 8, 1, "é"},
		{"The café is ready for lunch at the résidence", 8, 30, "é is ready for lunch at the ré"},
		{"The café is ready for lunch at the résidence", 8, 31, "é is ready for lunch at the rés"},
		{"日本の文化はとても興味深いです。", 1, 5, "日本の文化"},
		{"日本の文化はとても興味深いです。", 6, 7, "はとても興味深"},
		{"日本の文化はとても興味深いです。", 10, -1, "興味深いです。"},
	}

	for _, test := range tests {
		got := substring(test.s, test.start, test.length)
		if test.expected != got {
			t.Fatalf("expected %q, got %q", test.expected, got)
		}
	}

}

func Test_valueOf(t *testing.T) {

	attributes := map[string]any{
		"a": 5, // int
		"b": map[string]any{ // object
			"c": "foo", // string
			"d": map[string]any{ // map(array(int))
				"e":  []any{1},
				".e": []any{2},
				"e]": []any{3},
			},
		},
		"f": nil, // string
		"g": json.Value("12.53"),
		"h": json.Value(`{"i":true,"i?":5,"?i?":"boo","[i":"foo","i]":"zoo"}`),
		"l": json.Value(`{"name":"Bob","email":"bob@axample.com"}`),
	}

	tests := []struct {
		path     path
		expected any
		err      error
	}{
		{path{elements: []string{"a"}, decorators: []decorators{0}}, 5, nil},
		{path{elements: []string{"a"}, decorators: []decorators{indexing}}, 5, nil},
		{path{elements: []string{"b", "c"}, decorators: []decorators{0, 0}}, "foo", nil},
		{path{elements: []string{"b", "c"}, decorators: []decorators{0, indexing}}, "foo", nil},
		{path{elements: []string{"b", "x"}, decorators: []decorators{0, 0}}, nil, nil},
		{path{elements: []string{"b", "d", "e"}, decorators: []decorators{0, 0, 0}}, []any{1}, nil},
		{path{elements: []string{"b", "d", ".e"}, decorators: []decorators{0, 0, indexing}}, []any{2}, nil},
		{path{elements: []string{"b", "d", "e]"}, decorators: []decorators{0, 0, indexing}}, []any{3}, nil},
		{path{elements: []string{"b", "d", "x"}, decorators: []decorators{0, 0, 0}}, nil, nil},
		{path{elements: []string{"f"}, decorators: []decorators{0}}, nil, nil},
		{path{elements: []string{"g"}, decorators: []decorators{0}}, json.Value("12.53"), nil},
		{path{elements: []string{"g", "x"}, decorators: []decorators{0, 0}}, nil, TransformationError{msg: `invalid g.x: g is not JSON object, it is number`}},
		{path{elements: []string{"g", "x"}, decorators: []decorators{0, indexing}}, nil, TransformationError{msg: `invalid g["x"]: g is not JSON object, it is number`}},
		{path{elements: []string{"h", "i"}, decorators: []decorators{0, 0}}, json.Value("true"), nil},
		{path{elements: []string{"h", "i"}, decorators: []decorators{0, optional}}, json.Value("true"), nil},
		{path{elements: []string{"h", "i?"}, decorators: []decorators{0, indexing | optional}}, json.Value("5"), nil},
		{path{elements: []string{"h", "?i?"}, decorators: []decorators{0, indexing}}, json.Value(`"boo"`), nil},
		{path{elements: []string{"h", "?i?"}, decorators: []decorators{0, indexing | optional}}, json.Value(`"boo"`), nil},
		{path{elements: []string{"h", "[i"}, decorators: []decorators{0, indexing}}, json.Value(`"foo"`), nil},
		{path{elements: []string{"h", "[i"}, decorators: []decorators{0, indexing | optional}}, json.Value(`"foo"`), nil},
		{path{elements: []string{"h", "i]"}, decorators: []decorators{0, indexing}}, json.Value(`"zoo"`), nil},
		{path{elements: []string{"h", "i]"}, decorators: []decorators{0, indexing | optional}}, json.Value(`"zoo"`), nil},
		{path{elements: []string{"h", "i", "x"}, decorators: []decorators{0, 0, 0}}, nil, TransformationError{msg: `invalid h.i.x: h.i is not JSON object, it is true`}},
		{path{elements: []string{"h", "i", "x"}, decorators: []decorators{0, 0, optional}}, nil, nil},
		{path{elements: []string{"h", "x"}, decorators: []decorators{0, 0}}, nil, nil},
		{path{elements: []string{"h", "i", "x"}, decorators: []decorators{0, 0, indexing | optional}}, nil, nil},
		{path{elements: []string{"x"}, decorators: []decorators{0}}, nil, nil},
		{path{elements: []string{"l", "email"}, decorators: []decorators{0, 0}}, json.Value(`"bob@axample.com"`), nil},
		{path{elements: []string{"l", "name"}, decorators: []decorators{0, 0}}, json.Value(`"Bob"`), nil},
	}

	for _, test := range tests {
		got, err := valueOf(test.path, attributes)
		if !reflect.DeepEqual(test.err, err) {
			t.Fatalf("%s. expected error %v (type %T), got error %v (%T)", test.path, test.err, test.err, err, err)
		}
		if !reflect.DeepEqual(got, test.expected) {
			t.Fatalf("%s. unexpected value\nexpected %v (type %T)\ngot      %v (type %T)", test.path, test.expected, test.expected, got, got)
		}

	}

}

// TestOrConversionArgumentError checks the failing argument's diagnostic and
// short-circuit evaluation.
func TestOrConversionArgumentError(t *testing.T) {

	schema := types.Object([]types.Property{{Name: "value", Type: types.String()}})
	outSchema := types.Object([]types.Property{{Name: "out", Type: types.Boolean()}})
	tests := []struct {
		source   string
		argument string
	}{
		{"or(value, false)", "value"},
		{"or(false, value)", "value"},
		{"or(false, false, lower(value))", "lower(value)"},
		{"or(true, value)", ""},
	}

	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {

			mapping, err := New(map[string]string{"out": test.source}, schema, outSchema, false, nil)
			if err != nil {
				t.Fatal(err)
			}

			got, err := mapping.Transform(map[string]any{"value": "not-a-boolean"}, None)
			if err != nil {
				if test.argument == "" {
					t.Fatal(err)
				}
				if _, ok := errors.AsType[TransformationError](err); !ok {
					t.Fatalf("got %T, want TransformationError: %v", err, err)
				}
				want := fmt.Sprintf("«%s» (type string) does not represent a boolean "+
					"when passed to the «or» function while mapping to «out»", test.argument)
				if err.Error() != want {
					t.Fatalf("got %q, want %q", err, want)
				}
				return
			}
			if test.argument != "" {
				t.Fatal("expected a boolean conversion error")
			}
			if got["out"] != true {
				t.Fatalf("got %#v, want out=true", got)
			}

		})
	}

}

// TestArrayArgumentTransformationError preserves errors from evaluating an
// array argument.
func TestArrayArgumentTransformationError(t *testing.T) {

	inSchema := types.Object([]types.Property{{Name: "value", Type: types.String()}})
	outSchema := types.Object([]types.Property{{Name: "out", Type: types.Array(types.JSON())}})
	mapping, err := New(map[string]string{"out": "array(json_parse(value))"}, inSchema, outSchema, false, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = mapping.Transform(map[string]any{"value": "abc"}, None)
	if err != nil {
		if _, ok := errors.AsType[TransformationError](err); !ok {
			t.Fatalf("got %T (%v), want TransformationError", err, err)
		}
		want := "«value» cannot be parsed by «json_parse» because it is not valid JSON while mapping to «out»"
		if err.Error() != want {
			t.Fatalf("got %q, want %q", err, want)
		}
		return
	}
	t.Fatal("expected an argument evaluation error")

}

// TestArrayElementConversions checks that literals, properties, and calls use
// the destination element type.
func TestArrayElementConversions(t *testing.T) {

	day := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	clock := time.Date(1970, 1, 1, 12, 34, 56, 0, time.UTC)
	tests := []struct {
		name    string
		source  types.Type
		element types.Type
		value   any
		literal string
		want    any
	}{
		{"string to int", types.String(), types.Int(32), "42", "'42'", 42},
		{"string to boolean", types.String(), types.Boolean(), "true", "'true'", true},
		{"int8 to boolean", types.Int(8), types.Boolean(), 1, "", true},
		{"boolean to int8", types.Boolean(), types.Int(8), true, "true", 1},
		{"string to float32", types.String(), types.Float(32), "0.1", "'0.1'", float64(float32(0.1))},
		{
			name: "int64", source: types.Int(64), element: types.Int(64),
			value: int(9223372036854775807), literal: "9223372036854775807", want: int(9223372036854775807),
		},
		{"date", types.Date(), types.Date(), day, "'2026-09-08'", day},
		{"time", types.Time(), types.Time(), clock, "'12:34:56'", clock},
		{
			name: "nested array", source: types.Array(types.String()), element: types.Array(types.Int(32)),
			value: []any{"42", "7"}, literal: "array('42', '7')", want: []any{42, 7},
		},
		{
			name: "object", source: types.Object([]types.Property{{Name: "x", Type: types.String()}}),
			element: types.Object([]types.Property{{Name: "x", Type: types.Int(32)}}),
			value:   map[string]any{"x": "42"}, want: map[string]any{"x": 42},
		},
		{
			name: "map", source: types.Map(types.String()), element: types.Map(types.Int(32)),
			value: map[string]any{"x": "42"}, want: map[string]any{"x": 42},
		},
	}
	for _, test := range tests {

		expressions := []string{"array(value)", "array(if(true, value, value))"}
		if test.source.Kind() == types.StringKind {
			expressions = append(expressions, "array(substring(value, 1))")
		}
		if test.literal != "" {
			expressions = append(expressions, "array("+test.literal+")")
		}
		for _, expression := range expressions {

			t.Run(test.name+"/"+expression, func(t *testing.T) {

				inSchema := types.Object([]types.Property{{Name: "value", Type: test.source}})
				outSchema := types.Object([]types.Property{{Name: "out", Type: types.Array(test.element)}})
				mapping, err := New(map[string]string{"out": expression}, inSchema, outSchema, false, nil)
				if err != nil {
					t.Fatal(err)
				}
				got, err := mapping.Transform(map[string]any{"value": test.value}, None)
				if err != nil {
					t.Fatal(err)
				}
				want := map[string]any{"out": []any{test.want}}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("got %#v, want %#v", got, want)
				}

			})

		}

	}

}

// TestArrayElementRequiredProperties checks that final conversion validates
// required fields in constructed arrays.
func TestArrayElementRequiredProperties(t *testing.T) {

	source := types.Object([]types.Property{{Name: "x", Type: types.String(), ReadOptional: true}})
	element := types.Object([]types.Property{
		{Name: "x", Type: types.String(), CreateRequired: true, UpdateRequired: true},
	})
	tests := []struct {
		name    string
		source  types.Type
		element types.Type
		purpose Purpose
		value   any
		wantErr bool
	}{
		{"missing on create", source, element, Create, map[string]any{}, true},
		{"missing on update", source, element, Update, map[string]any{}, true},
		{"no required validation", source, element, None, map[string]any{}, false},
		{"present on create", source, element, Create, map[string]any{"x": "ok"}, false},
		{
			name: "inside map", source: types.Map(source), element: types.Map(element),
			purpose: Create, value: map[string]any{"key": map[string]any{}}, wantErr: true,
		},
		{
			name: "inside array", source: types.Array(source), element: types.Array(element),
			purpose: Update, value: []any{map[string]any{}}, wantErr: true,
		},
	}
	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			inSchema := types.Object([]types.Property{{Name: "value", Type: test.source}})
			outSchema := types.Object([]types.Property{{Name: "out", Type: types.Array(test.element)}})
			mapping, err := New(map[string]string{"out": "array(value)"}, inSchema, outSchema, false, nil)
			if err != nil {
				t.Fatal(err)
			}
			got, err := mapping.Transform(map[string]any{"value": test.value}, test.purpose)
			if err != nil {
				if !test.wantErr {
					t.Fatal(err)
				}
				if _, ok := err.(ValidationError); !ok {
					t.Fatalf("got %T, want ValidationError", err)
				}
				return
			}
			if test.wantErr {
				t.Fatal("expected a missing required property error")
			}
			want := map[string]any{"out": []any{test.value}}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %#v, want %#v", got, want)
			}

		})

	}

}

// TestArrayElementTimeLayouts checks final formatting of time values inside
// constructed containers.
func TestArrayElementTimeLayouts(t *testing.T) {

	moment := time.Date(2026, 9, 8, 12, 34, 56, 0, time.UTC)
	object := types.Object([]types.Property{{Name: "at", Type: types.DateTime()}})
	layouts := &state.TimeLayouts{DateTime: "unix", Date: "02/01/2006", Time: "15:04"}
	tests := []struct {
		name string
		typ  types.Type
		in   any
		out  any
	}{
		{"datetime", types.DateTime(), moment, moment.Unix()},
		{"date", types.Date(), time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC), "08/09/2026"},
		{"time", types.Time(), time.Date(1970, 1, 1, 12, 34, 56, 0, time.UTC), "12:34"},
		{"array", types.Array(types.DateTime()), []any{moment}, []any{moment.Unix()}},
		{"object", object, map[string]any{"at": moment}, map[string]any{"at": moment.Unix()}},
		{"map", types.Map(types.DateTime()), map[string]any{"at": moment}, map[string]any{"at": moment.Unix()}},
	}
	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			inSchema := types.Object([]types.Property{{Name: "value", Type: test.typ}})
			outSchema := types.Object([]types.Property{{Name: "out", Type: types.Array(test.typ)}})
			mapping, err := New(map[string]string{"out": "array(value)"}, inSchema, outSchema, false, layouts)
			if err != nil {
				t.Fatal(err)
			}
			got, err := mapping.Transform(map[string]any{"value": test.in}, None)
			if err != nil {
				t.Fatal(err)
			}
			want := map[string]any{"out": []any{test.out}}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %#v, want %#v", got, want)
			}

		})

	}

}

// TestArrayElementValidation checks error types and messages for both array
// elements and final conversion.
func TestArrayElementValidation(t *testing.T) {

	pattern := types.String().WithPattern(regexp.MustCompile("^a+$"))
	tests := []struct {
		name    string
		source  types.Type
		element types.Type
		value   any
		message string
	}{
		{
			name: "invalid integer", source: types.String(), element: types.Int(32), value: "abc",
			message: "«value» is not convertible to the «int(32)» type",
		},
		{
			name: "out of range", source: types.String(), element: types.Int(8), value: "128",
			message: "number «value» is greater than 127",
		},
		{
			name: "below minimum", source: types.String(), element: types.Int(8), value: "-129",
			message: "number «value» is less than -128",
		},
		{
			name: "overflow", source: types.String(), element: types.Int(64), value: "9223372036854775808",
			message: "number «value» is not a «int(64)» value",
		},
		{
			name: "decimal float32 overflow", source: types.Decimal(40, 0), element: types.Float(32),
			value: decimal.MustParse("1e39"), message: "number «value» is not a «float(32)» value",
		},
		{
			name: "negative decimal float32 overflow", source: types.Decimal(40, 0), element: types.Float(32),
			value: decimal.MustParse("-1e39"), message: "number «value» is not a «float(32)» value",
		},
		{
			name: "real float32 overflow", source: types.Float(64).Real(), element: types.Float(32).Real(),
			value: math.MaxFloat64, message: "number «value» is not a «float(32)» value",
		},
		{
			name: "null element", source: types.String(), element: types.String(),
			message: "«value» is not convertible to the «string» type",
		},
		{
			name: "enum", source: types.String(), element: types.String().WithValues("yes", "no"), value: "maybe",
			message: "«value» is not one of the allowed values",
		},
		{
			name: "pattern", source: types.String(), element: pattern,
			value: "abc", message: "«value» does not match «/^a+$/»",
		},
		{
			name: "max bytes", source: types.String(), element: types.String().WithMaxBytes(2), value: "abc",
			message: "«value» exceeds the 2-byte limit",
		},
		{
			name: "max length", source: types.String(), element: types.String().WithMaxLength(2), value: "abc",
			message: "«value» exceeds the 2-char limit",
		},
		{
			name: "date", source: types.String(), element: types.Date(), value: "abc",
			message: "«value» is not parsable as a date in ISO 8601 format",
		},
		{
			name: "year", source: types.Int(32), element: types.Year(), value: 10000,
			message: "year of «value» is not in range [1,9999]",
		},
		{
			name: "nested array", source: types.Array(types.String()), element: types.Array(types.Int(32)),
			value: []any{"abc"}, message: "«value» is not convertible to the «array(int(32))» type",
		},
		{
			name: "object", source: types.Object([]types.Property{{Name: "x", Type: types.String()}}),
			element: types.Object([]types.Property{{Name: "x", Type: types.Int(32)}}),
			value:   map[string]any{"x": "abc"}, message: "«value» is not convertible to the «object» type",
		},
		{
			name: "map", source: types.Map(types.String()), element: types.Map(types.Int(32)),
			value: map[string]any{"x": "abc"}, message: "«value» is not convertible to the «map(int(32))» type",
		},
		{
			name: "nested pattern", source: types.Object([]types.Property{{Name: "x", Type: types.String()}}),
			element: types.Object([]types.Property{{Name: "x", Type: pattern}}),
			value:   map[string]any{"x": "abc"}, message: "«value» is not convertible to the «object» type",
		},
		{
			name: "nested max bytes", source: types.Map(types.String()),
			element: types.Map(types.String().WithMaxBytes(2)), value: map[string]any{"x": "abc"},
			message: "«value» is not convertible to the «map(string)» type",
		},
		{
			name: "nested max length", source: types.Array(types.String()),
			element: types.Array(types.String().WithMaxLength(2)), value: []any{"abc"},
			message: "«value» is not convertible to the «array(string)» type",
		},
		{
			name: "nested minimum", source: types.Map(types.String()), element: types.Map(types.Int(8)),
			value: map[string]any{"x": "-129"}, message: "«value» is not convertible to the «map(int(8))» type",
		},
		{
			name: "nested maximum", source: types.Map(types.String()), element: types.Map(types.Int(8)),
			value: map[string]any{"x": "128"}, message: "«value» is not convertible to the «map(int(8))» type",
		},
		{
			name: "nested date", source: types.Map(types.String()), element: types.Map(types.Date()),
			value: map[string]any{"x": "abc"}, message: "«value» is not convertible to the «map(date)» type",
		},
	}
	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			inSchema := types.Object([]types.Property{{Name: "value", Type: test.source, Nullable: true}})
			destinations := []struct {
				expression string
				typ        types.Type
			}{
				{"value", test.element},
				{"array(value)", types.Array(test.element)},
				{"array(array(value))", types.Array(types.Array(test.element))},
			}
			if test.value == nil {
				// A null output property may be omitted; a null string array element is invalid.
				destinations = destinations[1:]
			}
			for _, destination := range destinations {

				t.Run(destination.expression, func(t *testing.T) {

					outSchema := types.Object([]types.Property{{Name: "out", Type: destination.typ}})
					expressions := map[string]string{"out": destination.expression}
					mapping, err := New(expressions, inSchema, outSchema, false, nil)
					if err != nil {
						t.Fatal(err)
					}
					_, err = mapping.Transform(map[string]any{"value": test.value}, None)
					if err != nil {
						if _, ok := errors.AsType[ValidationError](err); !ok {
							t.Fatalf("got %T (%v), want ValidationError", err, err)
						}
						want := test.message + " while mapping to «out»"
						if err.Error() != want {
							t.Fatalf("got %q, want %q", err, want)
						}
						return
					}
					t.Fatal("expected a conversion error")

				})

			}

		})

	}

}

// TestArrayDecimalScale checks exact decimal representation inside constructed
// arrays and their nested containers.
func TestArrayDecimalScale(t *testing.T) {

	source := types.Decimal(6, 3)
	target := types.Decimal(6, 2)
	containers := []struct {
		name           string
		source, target types.Type
		wrap           func(any) any
	}{
		{"decimal", source, target, func(v any) any { return v }},
		{"array", types.Array(source), types.Array(target), func(v any) any { return []any{v} }},
		{"map", types.Map(source), types.Map(target), func(v any) any { return map[string]any{"x": v} }},
		{
			"object",
			types.Object([]types.Property{{Name: "x", Type: source}}),
			types.Object([]types.Property{{Name: "x", Type: target}}),
			func(v any) any { return map[string]any{"x": v} },
		},
	}
	values := []struct {
		text    string
		wantErr bool
	}{
		{"1.234", true},
		{"-1.234", true},
		{"1.23", false},
		{"1.230", false},
		{"123e-2", false},
		{"0.000", false},
	}
	expressions := []string{"array(value)", "array(if(true, value, value))", "array(coalesce(value, value))"}
	for _, container := range containers {

		for _, value := range values {

			for _, expression := range expressions {

				t.Run(container.name+"/"+value.text+"/"+expression, func(t *testing.T) {

					inSchema := types.Object([]types.Property{{Name: "value", Type: container.source}})
					outSchema := types.Object([]types.Property{{Name: "out", Type: types.Array(container.target)}})
					mapping, err := New(map[string]string{"out": expression}, inSchema, outSchema, false, nil)
					if err != nil {
						t.Fatal(err)
					}
					input := container.wrap(decimal.MustParse(value.text))
					got, err := mapping.Transform(map[string]any{"value": input}, None)
					if err != nil {
						if !value.wantErr {
							t.Fatal(err)
						}
						if _, ok := errors.AsType[ValidationError](err); !ok {
							t.Fatalf("got %T (%v), want ValidationError", err, err)
						}
						return
					}
					if value.wantErr {
						t.Fatalf("got %#v, want a decimal scale validation error", got)
					}
					want := map[string]any{"out": []any{input}}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("got %#v, want %#v", got, want)
					}

				})

			}

		}

	}

}

// TestArrayExcelSerialDates checks serial date conversion without overflowing
// durations or the supported year range.
func TestArrayExcelSerialDates(t *testing.T) {

	tests := []struct {
		serial  string
		want    time.Time
		message string
	}{
		{"000", time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC), ""},
		{"059", time.Date(1900, 2, 28, 0, 0, 0, 0, time.UTC), ""},
		{"061", time.Date(1900, 3, 1, 0, 0, 0, 0, time.UTC), ""},
		{"59.99999999999999999999", time.Date(1900, 2, 28, 0, 0, 0, 0, time.UTC), ""},
		{"61.0", time.Date(1900, 3, 1, 0, 0, 0, 0, time.UTC), ""},
		{"44927.5", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ""},
		{"44927.50", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ""},
		{"44927.500", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ""},
		{"44927.5000", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ""},
		{"00044927", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ""},
		{"0000044927", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ""},
		{"109575", time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC), ""},
		{"109575.0", time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC), ""},
		{"109575.000", time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC), ""},
		{"109575.75", time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC), ""},
		{"2958465", time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC), ""},
		{"2958465.5", time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC), ""},
		{"2958465.99", time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC), ""},
		{"2958465.999999999999999", time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC), ""},
		{"2958466", time.Time{}, "year of «value» is not in range [1,9999]"},
		{"2958466.5", time.Time{}, "year of «value» is not in range [1,9999]"},
		{"2958466.00", time.Time{}, "year of «value» is not in range [1,9999]"},
		{"999999999999999999999", time.Time{}, "year of «value» is not in range [1,9999]"},
		{"060", time.Time{}, "«value» is not parsable as a date in ISO 8601 format"},
		{"60.0", time.Time{}, "«value» is not parsable as a date in ISO 8601 format"},
		{"60.01", time.Time{}, "«value» is not parsable as a date in ISO 8601 format"},
		{"60.5", time.Time{}, "«value» is not parsable as a date in ISO 8601 format"},
		{"60.999", time.Time{}, "«value» is not parsable as a date in ISO 8601 format"},
		{"60.00001", time.Time{}, "«value» is not parsable as a date in ISO 8601 format"},
		{"60.99999999999999999999", time.Time{}, "«value» is not parsable as a date in ISO 8601 format"},
	}
	for _, test := range tests {

		for _, purpose := range []Purpose{None, Create, Update} {

			t.Run(fmt.Sprintf("%s/purpose=%v", test.serial, purpose), func(t *testing.T) {

				inSchema := types.Object([]types.Property{{Name: "value", Type: types.String()}})
				outSchema := types.Object([]types.Property{{Name: "out", Type: types.Array(types.Date())}})
				mapping, err := New(map[string]string{"out": "array(value)"}, inSchema, outSchema, false, nil)
				if err != nil {
					t.Fatal(err)
				}
				got, err := mapping.Transform(map[string]any{"value": test.serial}, purpose)
				if err != nil {
					if test.message == "" {
						t.Fatal(err)
					}
					if _, ok := errors.AsType[ValidationError](err); !ok {
						t.Fatalf("got %T (%v), want ValidationError", err, err)
					}
					if want := test.message + " while mapping to «out»"; err.Error() != want {
						t.Fatalf("got %q, want %q", err, want)
					}
					return
				}
				if test.message != "" {
					t.Fatalf("got %#v, want a date validation error", got)
				}
				want := map[string]any{"out": []any{test.want}}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("got %#v, want %#v", got, want)
				}

			})

		}

	}

}

// TestArrayFloat32Precision checks rounding numeric elements directly to the
// destination precision.
func TestArrayFloat32Precision(t *testing.T) {

	tests := []struct {
		name   string
		source types.Type
		value  any
		want   float64
	}{
		{"signed", types.Int(32), int(16777217), 16777216},
		{"unsigned", types.Int(32).Unsigned(), uint(16777217), 16777216},
		{"negative", types.Int(32), int(-16777217), -16777216},
		// These values lie just above a float32 midpoint but round to that midpoint as float64.
		{"signed int64", types.Int(64), int(18014399583223809), 18014400656965632},
		{"unsigned int64", types.Int(64).Unsigned(), uint(18014399583223809), 18014400656965632},
		{"decimal integer", types.Decimal(17, 0), decimal.MustParse("18014399583223809"), 18014400656965632},
		{"negative decimal integer", types.Decimal(17, 0), decimal.MustParse("-18014399583223809"), -18014400656965632},
		{
			"decimal fraction", types.Decimal(31, 30),
			decimal.MustParse("1.000000059604644775390625000001"), 1.00000011920928955078125,
		},
		{
			"negative decimal fraction", types.Decimal(31, 30),
			decimal.MustParse("-1.000000059604644775390625000001"), -1.00000011920928955078125,
		},
		{
			"maximum decimal", types.Decimal(39, 0),
			decimal.MustParse("340282346638528859811704183484516925440"), math.MaxFloat32,
		},
	}
	for _, test := range tests {

		for _, purpose := range []Purpose{None, Create, Update} {

			t.Run(fmt.Sprintf("%s/purpose=%v", test.name, purpose), func(t *testing.T) {

				inSchema := types.Object([]types.Property{{Name: "value", Type: test.source}})
				outSchema := types.Object([]types.Property{{Name: "out", Type: types.Array(types.Float(32))}})
				mapping, err := New(map[string]string{"out": "array(value)"}, inSchema, outSchema, false, nil)
				if err != nil {
					t.Fatal(err)
				}
				got, err := mapping.Transform(map[string]any{"value": test.value}, purpose)
				if err != nil {
					t.Fatal(err)
				}
				want := map[string]any{"out": []any{test.want}}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("got %#v, want %#v", got, want)
				}

			})

		}

	}

}

// TestArrayMapNullValidation checks map value nullability independently of the
// transformation purpose.
func TestArrayMapNullValidation(t *testing.T) {

	source := types.Object([]types.Property{{Name: "x", Type: types.String(), Nullable: true, ReadOptional: true}})
	tests := []struct {
		name    string
		element types.Type
		value   map[string]any
		want    map[string]any
		wantErr bool
	}{
		{"null string", types.String(), map[string]any{"x": nil}, nil, true},
		{"empty integer", types.Int(32), map[string]any{"x": ""}, nil, true},
		{"JSON null", types.JSON(), map[string]any{"x": nil}, map[string]any{"x": json.Value("null")}, false},
		{"missing field", types.String(), map[string]any{}, map[string]any{}, false},
		{"non-null field", types.String(), map[string]any{"x": "ok"}, map[string]any{"x": "ok"}, false},
	}
	for _, test := range tests {

		for _, purpose := range []Purpose{None, Create, Update} {

			t.Run(fmt.Sprintf("%s/purpose=%v", test.name, purpose), func(t *testing.T) {

				inSchema := types.Object([]types.Property{{Name: "value", Type: source}})
				outSchema := types.Object([]types.Property{{Name: "out", Type: types.Array(types.Map(test.element))}})
				mapping, err := New(map[string]string{"out": "array(value)"}, inSchema, outSchema, false, nil)
				if err != nil {
					t.Fatal(err)
				}
				got, err := mapping.Transform(map[string]any{"value": test.value}, purpose)
				if err != nil {
					if !test.wantErr {
						t.Fatal(err)
					}
					if _, ok := errors.AsType[ValidationError](err); !ok {
						t.Fatalf("got %T (%v), want ValidationError", err, err)
					}
					return
				}
				if test.wantErr {
					t.Fatalf("got %#v, want a map value validation error", got)
				}
				want := map[string]any{"out": []any{test.want}}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("got %#v, want %#v", got, want)
				}

			})

		}

	}

}

// TestArrayObjectNullValidation checks null properties against the destination
// type and nullability.
func TestArrayObjectNullValidation(t *testing.T) {

	source := types.Object([]types.Property{{Name: "x", Type: types.String(), Nullable: true, ReadOptional: true}})
	tests := []struct {
		name     string
		typ      types.Type
		nullable bool
		want     any
		wantErr  bool
	}{
		{"JSON null", types.JSON(), false, json.Value("null"), false},
		{"nullable string", types.String(), true, nil, false},
		{"non-nullable string", types.String(), false, nil, true},
	}
	for _, test := range tests {

		for _, purpose := range []Purpose{None, Create, Update} {

			for _, inPlace := range []bool{false, true} {

				t.Run(fmt.Sprintf("%s/purpose=%v/inPlace=%v", test.name, purpose, inPlace), func(t *testing.T) {

					target := types.Object([]types.Property{{Name: "x", Type: test.typ, Nullable: test.nullable}})
					inSchema := types.Object([]types.Property{{Name: "value", Type: source}})
					outSchema := types.Object([]types.Property{{Name: "out", Type: types.Array(target)}})
					mapping, err := New(map[string]string{"out": "array(value)"}, inSchema, outSchema, inPlace, nil)
					if err != nil {
						t.Fatal(err)
					}
					got, err := mapping.Transform(map[string]any{"value": map[string]any{"x": nil}}, purpose)
					if err != nil {
						if !test.wantErr {
							t.Fatal(err)
						}
						if _, ok := errors.AsType[ValidationError](err); !ok {
							t.Fatalf("got %T (%v), want ValidationError", err, err)
						}
						return
					}
					if test.wantErr {
						t.Fatalf("got %#v, want an object property validation error", got)
					}
					want := map[string]any{"out": []any{map[string]any{"x": test.want}}}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("got %#v, want %#v", got, want)
					}

				})

			}

		}

	}

}

// TestArrayRealFloats checks the real constraint on numeric results from
// strings and floats.
func TestArrayRealFloats(t *testing.T) {

	tests := []struct {
		name   string
		source types.Type
		value  any
		want   float64
	}{
		{"string NaN", types.String(), "NaN", math.NaN()},
		{"string infinity", types.String(), "+Inf", math.Inf(1)},
		{"string negative infinity", types.String(), "-Infinity", math.Inf(-1)},
		{"string finite", types.String(), "1.5", 1.5},
		{"float NaN", types.Float(64), math.NaN(), math.NaN()},
		{"float infinity", types.Float(64), math.Inf(1), math.Inf(1)},
		{"float negative infinity", types.Float(64), math.Inf(-1), math.Inf(-1)},
		{"float finite", types.Float(64), 1.5, 1.5},
	}
	targets := []types.Type{types.Float(32), types.Float(32).Real(), types.Float(64), types.Float(64).Real()}
	for _, test := range tests {

		for _, target := range targets {

			for _, purpose := range []Purpose{None, Create, Update} {

				t.Run(fmt.Sprintf("%s/%s/purpose=%v", test.name, target, purpose), func(t *testing.T) {

					inSchema := types.Object([]types.Property{{Name: "value", Type: test.source}})
					outSchema := types.Object([]types.Property{{Name: "out", Type: types.Array(target)}})
					mapping, err := New(map[string]string{"out": "array(value)"}, inSchema, outSchema, false, nil)
					if err != nil {
						t.Fatal(err)
					}
					wantErr := target.IsReal() && (math.IsNaN(test.want) || math.IsInf(test.want, 0))
					got, err := mapping.Transform(map[string]any{"value": test.value}, purpose)
					if err != nil {
						if !wantErr {
							t.Fatal(err)
						}
						if _, ok := errors.AsType[ValidationError](err); !ok {
							t.Fatalf("got %T (%v), want ValidationError", err, err)
						}
						return
					}
					if wantErr {
						t.Fatalf("got %#v, want a real number validation error", got)
					}
					value := got["out"].([]any)[0].(float64)
					if value != test.want && !(math.IsNaN(value) && math.IsNaN(test.want)) {
						t.Fatalf("got %v, want %v", value, test.want)
					}

				})

			}

		}

	}

}

// TestArraySharedContainers checks time formatting of containers reused within
// and between expressions.
func TestArraySharedContainers(t *testing.T) {

	moment := time.Date(2026, 9, 8, 12, 34, 56, 0, time.UTC)
	object := types.Object([]types.Property{{Name: "at", Type: types.DateTime()}})
	containers := []struct {
		name string
		typ  types.Type
		wrap func(any) any
	}{
		{"object", object, func(v any) any { return map[string]any{"at": v} }},
		{"map", types.Map(types.DateTime()), func(v any) any { return map[string]any{"at": v} }},
		{"array", types.Array(types.DateTime()), func(v any) any { return []any{v} }},
		{
			"nested", types.Array(types.Map(object)),
			func(v any) any { return []any{map[string]any{"key": map[string]any{"at": v}}} },
		},
	}
	formats := []struct {
		name    string
		layouts *state.TimeLayouts
		want    any
	}{
		{"native", nil, moment},
		{"timestamp", &state.TimeLayouts{DateTime: "unix"}, moment.Unix()},
		{"text", &state.TimeLayouts{DateTime: time.RFC3339}, "2026-09-08T12:34:56Z"},
	}
	for _, container := range containers {

		for _, format := range formats {

			for _, inPlace := range []bool{false, true} {

				for _, purpose := range []Purpose{None, Create, Update} {

					name := fmt.Sprintf("%s/%s/inPlace=%v/purpose=%v", container.name, format.name, inPlace, purpose)
					t.Run(name, func(t *testing.T) {

						inSchema := types.Object([]types.Property{{Name: "value", Type: container.typ}})
						outSchema := types.Object([]types.Property{
							{Name: "first", Type: container.typ},
							{Name: "out", Type: types.Array(container.typ)},
						})
						expressions := map[string]string{"first": "value", "out": "array(value, value)"}
						mapping, err := New(expressions, inSchema, outSchema, inPlace, format.layouts)
						if err != nil {
							t.Fatal(err)
						}
						input := map[string]any{"value": container.wrap(moment)}
						got, err := mapping.Transform(input, purpose)
						if err != nil {
							t.Fatal(err)
						}
						want := map[string]any{
							"first": container.wrap(format.want),
							"out":   []any{container.wrap(format.want), container.wrap(format.want)},
						}
						if !reflect.DeepEqual(got, want) {
							t.Fatalf("got %#v, want %#v", got, want)
						}
						original := map[string]any{"value": container.wrap(moment)}
						if !reflect.DeepEqual(input, original) {
							t.Fatalf("input changed to %#v, want %#v", input, original)
						}

					})

				}

			}

		}

	}

}

// TestArrayShortDates checks date parsing with literal calendar years and
// rejects malformed short dates.
func TestArrayShortDates(t *testing.T) {

	day := time.Date(26, 9, 8, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		value   string
		want    time.Time
		wantErr bool
	}{
		{"09/08/26", day, false},
		{"09.08.26", day, false},
		{"26-09-08", day, false},
		{"01-02-03", time.Date(1, 2, 3, 0, 0, 0, 0, time.UTC), false},
		{"02/29/24", time.Date(24, 2, 29, 0, 0, 0, 0, time.UTC), false},
		{"02/29/23", time.Time{}, true},
		{"09/08/00", time.Time{}, true},
		{"ab/cd/ef", time.Time{}, true},
		{"09.08/26", time.Time{}, true},
	}
	for _, test := range tests {

		t.Run(test.value, func(t *testing.T) {

			inSchema := types.Object([]types.Property{{Name: "value", Type: types.String()}})
			outSchema := types.Object([]types.Property{{Name: "out", Type: types.Array(types.Date())}})
			mapping, err := New(map[string]string{"out": "array(value)"}, inSchema, outSchema, false, nil)
			if err != nil {
				t.Fatal(err)
			}
			got, err := mapping.Transform(map[string]any{"value": test.value}, None)
			if err != nil {
				if !test.wantErr {
					t.Fatal(err)
				}
				if _, ok := errors.AsType[ValidationError](err); !ok {
					t.Fatalf("got %T (%v), want ValidationError", err, err)
				}
				return
			}
			if test.wantErr {
				t.Fatalf("got %#v, want a date validation error", got)
			}
			want := map[string]any{"out": []any{test.want}}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %#v, want %#v", got, want)
			}

		})

	}

}

// TestConcatenationBoolean checks boolean values in prefixes, suffixes, and
// selected expressions.
func TestConcatenationBoolean(t *testing.T) {

	schema := types.Object([]types.Property{{Name: "flag", Type: types.Boolean(), Nullable: true}})
	tests := []struct {
		expression string
		value      any
		want       string
	}{
		{"flag '!'", true, "true!"},
		{"flag '!'", false, "false!"},
		{"flag '!'", nil, "!"},
		{"'!' flag", true, "!true"},
		{"'[' flag ']'", false, "[false]"},
		{"flag flag", true, "truetrue"},
		{"if(true, flag, null) '!'", true, "true!"},
		{"coalesce(flag, null) '!'", false, "false!"},
		{"not(flag) '!'", false, "true!"},
	}
	for _, test := range tests {

		t.Run(test.expression+"/"+test.want, func(t *testing.T) {

			expr, _, err := Compile(test.expression, schema, types.String())
			if err != nil {
				t.Fatal(err)
			}
			got, typ, err := expr.Eval(map[string]any{"flag": test.value})
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want || !types.Equal(typ, types.String()) {
				t.Fatalf("got %#v (%s), want %q (string)", got, typ, test.want)
			}

		})

	}

}

// TestMapTemporalStrings uses the textual representation of each temporal type.
func TestMapTemporalStrings(t *testing.T) {

	tests := []struct {
		typ   types.Type
		input string
	}{
		{types.Date(), `"2026-09-08"`},
		{types.Time(), `"12:34:56.123456789"`},
		{types.DateTime(), `"2026-09-08T12:34:56.123456789Z"`},
	}

	for _, test := range tests {

		for _, sorted := range []bool{false, true} {

			t.Run(fmt.Sprintf("%s/sorted=%t", test.typ, sorted), func(t *testing.T) {

				previous := encodeSorted
				encodeSorted = sorted
				t.Cleanup(func() { encodeSorted = previous })
				inSchema := types.Object([]types.Property{{Name: "value", Type: test.typ}})
				attributes, err := types.Decode[map[string]any](strings.NewReader(`{"value":`+test.input+`}`), inSchema)
				if err != nil {
					t.Fatal(err)
				}
				outSchema := types.Object([]types.Property{{Name: "out", Type: types.Map(types.String())}})
				mapping, err := New(map[string]string{"out": "map('x', value)"}, inSchema, outSchema, false, nil)
				if err != nil {
					t.Fatal(err)
				}
				out, err := mapping.Transform(attributes, None)
				if err != nil {
					t.Fatal(err)
				}

				got := out["out"].(map[string]any)["x"]
				if want := json.Value(test.input).String(); got != want {
					t.Fatalf("got %q, want %q", got, want)
				}

			})

		}

	}

}

// TestLenSelectedValues checks that selecting a value preserves the length of
// its typed representation.
func TestLenSelectedValues(t *testing.T) {

	tests := []struct {
		name  string
		typ   types.Type
		value any
		want  int
	}{
		{"date", types.Date(), time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC), 10},
		{"datetime", types.DateTime(), time.Date(2026, 9, 8, 12, 34, 56, 0, time.UTC), 20},
		{"datetime millis", types.DateTime(), time.Date(2026, 9, 8, 12, 34, 56, 125000000, time.UTC), 24},
		{"datetime one nano", types.DateTime(), time.Date(2026, 9, 8, 12, 34, 56, 1, time.UTC), 30},
		{"datetime nanos", types.DateTime(), time.Date(2026, 9, 8, 12, 34, 56, 123456789, time.UTC), 30},
		{"time", types.Time(), time.Date(1970, 1, 1, 12, 34, 56, 125000000, time.UTC), 12},
		{"float32", types.Float(32), float64(float32(0.1)), 3},
		{"float64", types.Float(64), float64(float32(0.1)), 19},
	}
	expressions := []string{
		"len(value)",
		"len(value '')",
		"len(if(true, value, null))",
		"len(if(false, null, value))",
		"len(coalesce(null, value))",
		"len(coalesce(if(false, value), value))",
	}
	for _, test := range tests {
		for _, expression := range expressions {

			t.Run(test.name+"/"+expression, func(t *testing.T) {

				schema := types.Object([]types.Property{{Name: "value", Type: test.typ}})
				expr, _, err := Compile(expression, schema, types.Int(32))
				if err != nil {
					t.Fatal(err)
				}
				got, typ, err := expr.Eval(map[string]any{"value": test.value})
				if err != nil {
					t.Fatal(err)
				}
				if got != test.want || !types.Equal(typ, types.Int(32)) {
					t.Fatalf("got %v (%s), want %d (int(32))", got, typ, test.want)
				}

			})

		}
	}

}

// TestLenIntegerBoundaries checks exact digit counts near decimal powers and
// integer limits.
func TestLenIntegerBoundaries(t *testing.T) {

	tests := []struct {
		typ   types.Type
		value any
		want  int
	}{
		{types.Int(64), int(0), 1},
		{types.Int(64), int(999999999999999), 15},
		{types.Int(64), int(99999999999999999), 17},
		{types.Int(64), int(999999999999999999), 18},
		{types.Int(64), int(1000000000000000000), 19},
		{types.Int(64), int(1000000000000000001), 19},
		{types.Int(64), int(-999999999999999999), 19},
		{types.Int(64), int(9223372036854775807), 19},
		{types.Int(64), int(-9223372036854775808), 20},
		{types.Int(64).Unsigned(), uint(0), 1},
		{types.Int(64).Unsigned(), uint(999999999999999999), 18},
		{types.Int(64).Unsigned(), uint(9999999999999999999), 19},
		{types.Int(64).Unsigned(), uint(10000000000000000000), 20},
		{types.Int(64).Unsigned(), uint(18446744073709551615), 20},
	}
	for _, test := range tests {

		t.Run(fmt.Sprintf("%T/%v", test.value, test.value), func(t *testing.T) {

			schema := types.Object([]types.Property{{Name: "value", Type: test.typ}})
			expr, _, err := Compile("len(value)", schema, types.Int(32))
			if err != nil {
				t.Fatal(err)
			}
			got, typ, err := expr.Eval(map[string]any{"value": test.value})
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want || !types.Equal(typ, types.Int(32)) {
				t.Fatalf("got %v (%s), want %d (int(32))", got, typ, test.want)
			}

		})

	}

}

// TestMapEncodingFailure checks that failed JSON encoding stops map evaluation
// with a transformation error.
func TestMapEncodingFailure(t *testing.T) {

	for _, sorted := range []bool{false, true} {

		for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {

			for _, element := range []types.Type{types.JSON(), types.String()} {

				t.Run(fmt.Sprintf("sorted=%v/%v/%s", sorted, value, element), func(t *testing.T) {

					previous := encodeSorted
					encodeSorted = sorted
					t.Cleanup(func() { encodeSorted = previous })
					inSchema := types.Object([]types.Property{{Name: "value", Type: types.Float(64)}})
					outSchema := types.Object([]types.Property{{Name: "out", Type: types.Map(element)}})
					mapping, err := New(map[string]string{"out": "map('x', value)"}, inSchema, outSchema, false, nil)
					if err != nil {
						t.Fatal(err)
					}
					got, err := mapping.Transform(map[string]any{"value": value}, None)
					if err != nil {
						if _, ok := errors.AsType[TransformationError](err); !ok {
							t.Fatalf("got %T (%v), want TransformationError", err, err)
						}
						if !strings.HasSuffix(err.Error(), " while mapping to «out»") {
							t.Fatalf("error does not identify the output: %v", err)
						}
						return
					}
					t.Fatalf("got %#v, want a JSON encoding transformation error", got)

				})

			}

		}

	}

}

// TestNullAncestor checks that a null container never redirects path lookup to
// a containing object.
func TestNullAncestor(t *testing.T) {

	object := types.Object([]types.Property{{Name: "name", Type: types.String()}})
	nested := types.Object([]types.Property{
		{Name: "child", Type: object, Nullable: true}, {Name: "name", Type: types.String()},
	})
	tests := []struct {
		name, path string
		parentType types.Type
		parent     any
		want       any
	}{
		{"object", "parent.name", object, nil, nil},
		{"map", "parent['name']", types.Map(types.String()), nil, nil},
		{"typed nil", "parent.name", object, map[string]any(nil), nil},
		{"nested", "parent.child.name", nested, map[string]any{"child": nil, "name": "PARENT"}, nil},
		{"populated", "parent.name", object, map[string]any{"name": "nested"}, "nested"},
	}
	for _, test := range tests {

		for _, form := range []string{"path", "lower", "concatenation"} {

			t.Run(test.name+"/"+form, func(t *testing.T) {

				source, want := test.path, test.want
				switch form {
				case "lower":
					source = "lower(" + source + ")"
				case "concatenation":
					source += " '!'"
					want = "!"
					if test.want != nil {
						want = test.want.(string) + "!"
					}
				}
				inSchema := types.Object([]types.Property{
					{Name: "parent", Type: test.parentType, Nullable: true}, {Name: "name", Type: types.Int(32)},
				})
				outSchema := types.Object([]types.Property{{Name: "out", Type: types.String(), Nullable: true}})
				mapping, err := New(map[string]string{"out": source}, inSchema, outSchema, false, nil)
				if err != nil {
					t.Fatal(err)
				}
				got, err := mapping.Transform(map[string]any{"parent": test.parent, "name": 123}, None)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, map[string]any{"out": want}) {
					t.Fatalf("got %#v, want out=%#v", got, want)
				}

			})

		}

	}

}

// TestSubstringArguments checks dynamic integer arguments and errors
// identifying the invalid argument.
func TestSubstringArguments(t *testing.T) {

	tests := []struct {
		name       string
		expression string
		typ        types.Type
		position   any
		length     any
		want       string
		message    string
	}{
		{
			name: "invalid start with two arguments", expression: "substring('abc', position)",
			typ: types.String(), position: "bad",
			message: "«position», of type string, cannot be passed as int to the «substring» function",
		},
		{
			name: "invalid start with three arguments", expression: "substring('abc', position, length)",
			typ: types.String(), position: "bad", length: "1",
			message: "«position», of type string, cannot be passed as int to the «substring» function",
		},
		{
			name: "invalid length", expression: "substring('abc', position, length)",
			typ: types.String(), position: "1", length: "bad",
			message: "«length», of type string, cannot be passed as int to the «substring» function",
		},
		{
			name: "unsigned start", expression: "substring('abc', position)",
			typ: types.Int(32).Unsigned(), position: uint(2), want: "bc",
		},
		{
			name: "unsigned length", expression: "substring('abc', 1, length)",
			typ: types.Int(32).Unsigned(), length: uint(2), want: "ab",
		},
		{
			name: "unsigned start and length", expression: "substring('aé🙂z', position, length)",
			typ: types.Int(16).Unsigned(), position: uint(2), length: uint(2), want: "é🙂",
		},
		{
			name: "zero unsigned start", expression: "substring('abc', position)",
			typ: types.Int(8).Unsigned(), position: uint(0), want: "abc",
		},
		{
			name: "start beyond string", expression: "substring('abc', position)",
			typ: types.Int(32).Unsigned(), position: uint(2147483647), want: "",
		},
		{
			name: "start overflow", expression: "substring('abc', position)",
			typ: types.Int(32).Unsigned(), position: uint(2147483648),
			message: "«position», with a value of 2147483648, " +
				"cannot be passed as a 32-bit int to the «substring» function",
		},
		{
			name: "length overflow", expression: "substring('abc', 1, length)",
			typ: types.Int(32).Unsigned(), length: uint(2147483648),
			message: "«length», with a value of 2147483648, " +
				"cannot be passed as a 32-bit int to the «substring» function",
		},
	}
	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			inSchema := types.Object([]types.Property{
				{Name: "position", Type: test.typ, Nullable: true},
				{Name: "length", Type: test.typ, Nullable: true},
			})
			outSchema := types.Object([]types.Property{{Name: "out", Type: types.String()}})
			mapping, err := New(map[string]string{"out": test.expression}, inSchema, outSchema, false, nil)
			if err != nil {
				t.Fatal(err)
			}
			got, err := mapping.Transform(map[string]any{"position": test.position, "length": test.length}, None)
			if err != nil {
				if test.message == "" {
					t.Fatal(err)
				}
				if _, ok := errors.AsType[TransformationError](err); !ok {
					t.Fatalf("got %T (%v), want TransformationError", err, err)
				}
				want := test.message + " while mapping to «out»"
				if err.Error() != want {
					t.Fatalf("got %q, want %q", err, want)
				}
				return
			}
			if test.message != "" {
				t.Fatalf("got %#v, want an argument conversion error", got)
			}
			want := map[string]any{"out": test.want}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %#v, want %#v", got, want)
			}

		})

	}

}

// TestSubstringEmptyString checks the empty result for all valid start and
// length combinations.
func TestSubstringEmptyString(t *testing.T) {

	expressions := []string{
		"substring('', 1)",
		"substring('', 1, 1)",
		"substring('', 0, 1)",
		"substring('', -1, 1)",
		"substring('', 1, 0)",
		"substring('', 2, 1)",
	}
	for _, expression := range expressions {

		t.Run(expression, func(t *testing.T) {

			expr, _, err := Compile(expression, types.Type{}, types.String())
			if err != nil {
				t.Fatal(err)
			}
			got, typ, err := expr.Eval(nil)
			if err != nil {
				t.Fatal(err)
			}
			if got != "" || !types.Equal(typ, types.String()) {
				t.Fatalf("got %#v (%s), want an empty string", got, typ)
			}

		})

	}

}

// TestSubstringUTF8Boundary checks byte widths, including malformed input
// outside the UTF-8 precondition.
func TestSubstringUTF8Boundary(t *testing.T) {

	tests := []struct {
		name          string
		value         string
		start, length int
		want          string
	}{
		{"valid multibyte rune", "é🙂x", 2, 1, "🙂"},
		{"valid replacement rune", "\uFFFDx", 1, 1, "\uFFFD"},
		{"valid replacement at end", "é\uFFFD", 2, 10, "\uFFFD"},
		{"invalid byte", "\xff", 1, 1, "\xff"},
		{"invalid byte before ASCII", "\xffab", 1, 1, "\xff"},
		{"invalid byte before multibyte rune", "\xffé", 1, 1, "\xff"},
		{"invalid byte at end", "é\xff", 2, 1, "\xff"},
		{"invalid byte before replacement rune", "\xff\uFFFD", 1, 1, "\xff"},
		{"replacement rune before invalid byte", "\uFFFD\xff", 1, 1, "\uFFFD"},
		{"continuation byte", "\x80", 1, 1, "\x80"},
		{"truncated rune", "\xe2\x82", 1, 1, "\xe2"},
		{"truncated rune to end", "\xe2\x82", 1, 10, "\xe2\x82"},
		{"overlong encoding", "\xc0\xaf", 1, 1, "\xc0"},
		{"surrogate encoding", "\xed\xa0\x80", 1, 1, "\xed"},
		{"out of range rune", "\xf4\x90\x80\x80", 1, 1, "\xf4"},
	}
	inSchema := types.Object([]types.Property{
		{Name: "value", Type: types.String()},
		{Name: "start", Type: types.Int(32)},
		{Name: "length", Type: types.Int(32)},
	})
	outSchema := types.Object([]types.Property{{Name: "out", Type: types.String()}})
	mapping, err := New(map[string]string{"out": "substring(value, start, length)"}, inSchema, outSchema, false, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			attributes := map[string]any{"value": test.value, "start": test.start, "length": test.length}
			out, err := mapping.Transform(attributes, None)
			if err != nil {
				t.Fatal(err)
			}
			if got := out["out"]; got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}

}
