//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"reflect"
	"testing"

	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

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

	properties := map[string]any{
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
		{[]part{{value: ``, typ: types.Text()}}, "", types.Text(), nil},
		{[]part{{value: `a`, typ: types.Text()}}, "a", types.Text(), nil},
		{[]part{{value: n, typ: dt}}, n, dt, nil},
		{[]part{{path: path{elements: []string{"a"}, decorators: []decorators{0}}, typ: types.Int(32)}}, 165, types.Int(32), nil},
		{[]part{{path: path{elements: []string{"b", "c"}, decorators: []decorators{0, 0}}, typ: types.Text()}}, "foo", types.Text(), nil},
		{[]part{{path: path{elements: []string{"b", "e"}, decorators: []decorators{0, 0}}, typ: types.Int(32)}}, 1024, types.Int(32), nil},
		{[]part{{value: `a`, path: path{elements: []string{"a"}, decorators: []decorators{0}}, typ: types.Int(32)}}, "a165", types.Text(), nil},
		{[]part{{path: path{elements: []string{"coalesce"}, decorators: []decorators{0}}, args: [][]part{
			{{path: path{elements: []string{"a"}, decorators: []decorators{0}}, typ: types.Int(32)}, {value: " boo", typ: types.Text()}},
			{{value: "foo", typ: types.Text()}},
		}}}, "165 boo", types.Text(), nil},
		{[]part{{path: path{elements: []string{"coalesce"}, decorators: []decorators{0}}, args: [][]part{
			{{path: path{elements: []string{"d"}, decorators: []decorators{0}}, typ: types.Text()}},
			{{path: path{elements: []string{"a"}, decorators: []decorators{0}}, typ: types.Int(32)}, {value: " boo", typ: types.Text()}},
		}}}, "165 boo", types.Text(), nil},
		{[]part{{value: ``, typ: types.Text()}, {path: path{elements: []string{"a"}, decorators: []decorators{0}}, typ: types.Int(32)}}, "165", types.Text(), nil},
		{[]part{{path: path{elements: []string{"x"}, decorators: []decorators{0}}, typ: types.Boolean()}}, nil, types.Type{}, nil},
		{[]part{{path: path{elements: []string{"b", "x"}, decorators: []decorators{0, 0}}, typ: types.Boolean()}}, nil, types.Boolean(), nil},
	}

	for i, test := range tests {
		got, typ, err := eval(test.expr, properties)
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

	properties := map[string]any{
		"a": 5, // int
		"b": map[string]any{ // object
			"c": "foo", // text
			"d": map[string]any{ // map(array(int))
				"e":  []any{1},
				".e": []any{2},
				"e]": []any{3},
			},
		},
		"f": nil, // text
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
		got, err := valueOf(test.path, properties)
		if !reflect.DeepEqual(test.err, err) {
			t.Fatalf("%s. expected error %v (type %T), got error %v (%T)", test.path, test.err, test.err, err, err)
		}
		if !reflect.DeepEqual(got, test.expected) {
			t.Fatalf("%s. unexpected value\nexpected %v (type %T)\ngot      %v (type %T)", test.path, test.expected, test.expected, got, got)
		}

	}

}
