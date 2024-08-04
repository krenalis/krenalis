//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/meergo/meergo/types"

	"github.com/shopspring/decimal"
)

func Test_eval(t *testing.T) {

	properties := map[string]any{
		"a": 165,
		"b": map[string]any{
			"c": "foo",
			"e": 1024,
		},
		"d": nil,
	}
	n := decimal.NewFromInt(5)
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
		{[]part{{path: path{"a"}, typ: types.Int(32)}}, 165, types.Int(32), nil},
		{[]part{{path: path{"b", "c"}, typ: types.Text()}}, "foo", types.Text(), nil},
		{[]part{{path: path{"b", "e"}, typ: types.Int(32)}}, 1024, types.Int(32), nil},
		{[]part{{value: `a`, path: path{"a"}, typ: types.Int(32)}}, "a165", types.Text(), nil},
		{[]part{{path: path{"coalesce"}, args: [][]part{
			{{path: path{"a"}, typ: types.Int(32)}, {value: " boo", typ: types.Text()}},
			{{value: "foo", typ: types.Text()}},
		}}}, "165 boo", types.Text(), nil},
		{[]part{{path: path{"coalesce"}, args: [][]part{
			{{path: path{"d"}, typ: types.Text()}},
			{{path: path{"a"}, typ: types.Int(32)}, {value: " boo", typ: types.Text()}},
		}}}, "165 boo", types.Text(), nil},
		{[]part{{value: ``, typ: types.Text()}, {path: path{"a"}, typ: types.Int(32)}}, "165", types.Text(), nil},
		{[]part{{path: path{"x"}, typ: types.Boolean()}}, nil, types.Type{}, nil},
		{[]part{{path: path{"b", "x"}, typ: types.Boolean()}}, nil, types.Boolean(), nil},
	}

	for i, test := range tests {
		got, typ, err := eval(test.expr, properties, nil, None)
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

func Test_valueOf(t *testing.T) {

	properties := map[string]any{
		"a": 5, // Int
		"b": map[string]any{ // Object
			"c": "foo", // Text
			"d": map[string]any{ // Map(Array(Int))
				"e":  []any{1},
				":e": []any{2},
				"e]": []any{3},
			},
		},
		"f": nil,                  // Text
		"g": json.Number("12.53"), // JSON
		"h": map[string]any{ // JSON
			"i":   true,
			"i?":  5,
			":i?": "boo",
			"[i":  "foo",
			"i]":  "zoo",
		},
		"l": json.RawMessage(`{"name":"Bob","email":"bob@axample.com"}`),
	}

	tests := []struct {
		path     path
		expected any
		err      error
	}{
		{path{"a"}, 5, nil},
		{path{"[a]"}, 5, nil},
		{path{"b", "c"}, "foo", nil},
		{path{"b", "[c]"}, "foo", nil},
		{path{"b", "x"}, nil, nil},
		{path{"b", "d", ":e"}, []any{1}, nil},
		{path{"b", "d", ":[:e]"}, []any{2}, nil},
		{path{"b", "d", ":[e]]"}, []any{3}, nil},
		{path{"b", "d", ":x"}, nil, nil},
		{path{"f"}, nil, nil},
		{path{"g"}, json.Number("12.53"), nil},
		{path{"g", ":x"}, nil, errors.New(`invalid g.x: g is not a JSON object, it is a JSON number`)},
		{path{"g", ":[x]"}, nil, errors.New(`invalid g["x"]: g is not a JSON object, it is a JSON number`)},
		{path{"h", ":i"}, true, nil},
		{path{"h", ":i?"}, true, nil},
		{path{"h", ":[i?]?"}, 5, nil},
		{path{"h", ":[:i?]"}, "boo", nil},
		{path{"h", ":[:i?]?"}, "boo", nil},
		{path{"h", ":[[i]"}, "foo", nil},
		{path{"h", ":[[i]?"}, "foo", nil},
		{path{"h", ":[i]]"}, "zoo", nil},
		{path{"h", ":[i]]?"}, "zoo", nil},
		{path{"h", ":i", ":x"}, nil, errors.New(`invalid h.i.x: h.i is not a JSON object, it is a JSON boolean`)},
		{path{"h", ":i", ":x?"}, nil, nil},
		{path{"h", ":x"}, nil, nil},
		{path{"h", ":i", ":[x]?"}, nil, nil},
		{path{"x"}, nil, nil},
		{path{"l", "email"}, "bob@axample.com", nil},
		{path{"l", "name"}, "Bob", nil},
	}

	for _, test := range tests {
		got, err := valueOf(test.path, properties)
		if err != nil {
			if test.err == nil {
				t.Fatalf("%s. unexpected error: %s", stringifyPath(test.path), err)
			}
			if err.Error() != test.err.Error() {
				t.Fatalf("%s. expected error %q, got error %q", stringifyPath(test.path), test.err.Error(), err.Error())
			}
			continue
		}
		if test.err != nil {
			t.Fatalf("%s. expected error %q, got no error", stringifyPath(test.path), test.err)
		}
		if !reflect.DeepEqual(got, test.expected) {
			t.Fatalf("%s. unexpected value\nexpected %v (type %T)\ngot      %v (type %T)", stringifyPath(test.path), test.expected, test.expected, got, got)
		}

	}

}
