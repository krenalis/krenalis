//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mapexp

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"chichi/connector/types"

	"github.com/shopspring/decimal"
)

func TestEval(t *testing.T) {

	values := map[string]any{
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
		{[]part{{text: ``}}, "", types.Text(), nil},
		{[]part{{text: `a`}}, "a", types.Text(), nil},
		{[]part{{value: n, typ: dt}}, n, dt, nil},
		{[]part{{path: types.Path{"a"}, typ: types.Int()}}, 165, types.Int(), nil},
		{[]part{{path: types.Path{"b", "c"}, typ: types.Text()}}, "foo", types.Text(), nil},
		{[]part{{path: types.Path{"b", "e"}, typ: types.Int()}}, 1024, types.Int(), nil},
		{[]part{{text: `a`, path: types.Path{"a"}, typ: types.Int()}}, "a165", types.Text(), nil},
		{[]part{{path: types.Path{"coalesce"}, args: [][]part{
			{{path: types.Path{"a"}, typ: types.Int()}, {text: " boo"}},
			{{text: "foo"}},
		}}}, "165 boo", types.Text(), nil},
		{[]part{{path: types.Path{"coalesce"}, args: [][]part{
			{{path: types.Path{"d"}, typ: types.Text()}},
			{{path: types.Path{"a"}, typ: types.Int()}, {text: " boo"}},
		}}}, "165 boo", types.Text(), nil},
		{[]part{{text: ``}, {path: types.Path{"a"}, typ: types.Int()}}, "165", types.Text(), nil},
	}

	for i, test := range tests {
		got, typ, err := eval(test.expr, values)
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
		if !typ.EqualTo(test.expectedType) {
			if typ.Valid() {
				t.Fatalf("%d. expected type %s, got %s", i+1, test.expectedType, typ)
			}
			t.Fatalf("%d. expected type %s, got invalid type", i+1, test.expectedType)
		}
	}

}

func TestExpressions(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "manufacturer", Type: types.Text()},
		{Name: "model", Type: types.Text()},
		{Name: "engine", Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text()},
			{Name: "power", Type: types.UInt()},
			{Name: "afterburner", Type: types.Text(), Nullable: true},
		})},
		{Name: "cx", Type: types.Decimal(4, 3)},
		{Name: "passengers", Type: types.Int()},
	})

	cx := decimal.NewFromFloat(0.142)
	values := map[string]any{
		"manufacturer": "MyPlaneCompany",
		"model":        "SuperFast",
		"engine": map[string]any{
			"name":        "TurboX",
			"power":       uint(700),
			"afterburner": nil,
		},
		"cx":         cx,
		"passengers": 250,
	}

	tests := []struct {
		expr          string
		dt            types.Type
		compileErr    error
		evalErr       error
		expectedValue any
	}{
		{expr: "' '   '  '", dt: types.Text(), expectedValue: "   "},
		{expr: "''", dt: types.JSON(), expectedValue: json.RawMessage("null")},
		{expr: "'100'", dt: types.JSON(), expectedValue: json.RawMessage(`"100"`)},
		{expr: "'42'", dt: types.Int(), expectedValue: 42},
		{expr: "'42'", dt: types.Text(), expectedValue: "42"},
		{expr: "'Afterburner: ' coalesce('yes', 'no afterburner')", dt: types.Text(), expectedValue: "Afterburner: yes"},
		{expr: "'Afterburner: ' coalesce(engine.afterburner, 'no afterburner')", dt: types.Text(), expectedValue: "Afterburner: no afterburner"},
		{expr: "'Afterburner: ' coalesce(engine.afterburner, engine.afterburner, 'no afterburner')", dt: types.Text(), expectedValue: "Afterburner: no afterburner"},
		{expr: "42", dt: types.Int(), expectedValue: 42},
		{expr: "42", dt: types.JSON(), expectedValue: json.RawMessage("42")},
		{expr: "42", dt: types.Text(), expectedValue: "42"},
		{expr: "coalesce()", dt: types.JSON(), expectedValue: json.RawMessage("null")},
		{expr: "cx cx", dt: types.Text(), expectedValue: "0.1420.142"},
		{expr: "engine.name", dt: types.Text(), expectedValue: "TurboX"},
		{expr: "engine.power ' x ' 1.36", dt: types.Text(), expectedValue: "700 x 1.36"},
		{expr: "engine.power", dt: types.Float(), expectedValue: 700.0},
		{expr: "engine", dt: types.JSON(), expectedValue: json.RawMessage(`{"afterburner":null,"name":"TurboX","power":700}`)},
		{expr: "manufacturer ' ' model", dt: types.Text(), expectedValue: "MyPlaneCompany SuperFast"},
		{expr: "manufacturer", dt: types.JSON(), expectedValue: json.RawMessage("\"MyPlaneCompany\"")},
		{expr: "manufacturer", dt: types.Text(), expectedValue: "MyPlaneCompany"},
		{expr: `""`, dt: types.JSON(), expectedValue: json.RawMessage("null")},

		// Compile errors.
		{expr: "!true", dt: types.Boolean(), compileErr: errors.New("unexpected character '!'")},
		{expr: "'Engine power: ' (coalesce engine.power, 0)", dt: types.Text(), compileErr: errors.New("unexpected character '('")},
		{expr: "'Engine power: ' coalesce engine.power, 0", dt: types.Text(), compileErr: errors.New(`property path "coalesce" does not exist`)},
		{expr: "'Engine power: engine.power", dt: types.Text(), compileErr: errors.New("string is not terminated")},
		{expr: "1 + 2", dt: types.Int(), compileErr: errors.New("unexpected character '+'")},
		{expr: "engine.power * 1.36", dt: types.Text(), compileErr: errors.New("unexpected character '*'")},
		{expr: "len('hello')", dt: types.UInt(), compileErr: errors.New(`function "len" does not exist`)},
		{expr: "not true", dt: types.Boolean(), compileErr: errors.New(`property path "not" does not exist`)},
		{expr: "passenger", dt: types.Text(), compileErr: errors.New(`property path "passenger" does not exist`)},
		{expr: "true && false", dt: types.Boolean(), compileErr: errors.New(`unexpected character '&'`)},
		{expr: "1,000", dt: types.Int(), compileErr: errors.New(`unexpected character ','`)},

		// Eval errors.
		{expr: "manufacturer", dt: types.Int(), evalErr: errors.New(`cannot convert "MyPlaneCompany" (type Text) to type Int`)},
	}

	for _, test := range tests {
		t.Run(test.expr, func(t *testing.T) {

			// Test Compile.
			expr, err := Compile(test.expr, schema, test.dt, false)
			if test.compileErr != nil {
				if err == nil {
					t.Fatalf("expecting compile error %s, got no errors", test.compileErr)
				}
				if test.compileErr.Error() != err.Error() {
					t.Fatalf("expecting compile error %q, got %q", test.compileErr.Error(), err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected compile error: %s", err)
			}

			// Test Eval.
			gotValue, err := expr.Eval(values, false)
			if test.evalErr != nil {
				if err == nil {
					t.Fatalf("expecting eval error %s, got no errors", test.evalErr)
				}
				if test.evalErr.Error() != err.Error() {
					t.Fatalf("expecting eval error %q, got %q", test.evalErr.Error(), err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected eval error: %s", err)
			}
			if !reflect.DeepEqual(test.expectedValue, gotValue) {
				if j, ok := gotValue.(json.RawMessage); ok {
					t.Fatalf("expecting value %#v, got %#v (which represents the string %q)", test.expectedValue, gotValue, string(j))
				}
				t.Fatalf("expecting value %#v, got %#v", test.expectedValue, gotValue)
			}

		})
	}

}

func TestPropertyPaths(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "a", Type: types.Object([]types.Property{
			{Name: "b", Type: types.Object([]types.Property{
				{Name: "c", Type: types.Text()},
			})},
		})},
		{Name: "b", Type: types.Text()},
		{Name: "c", Type: types.Text()},
	})

	tests := []struct {
		src      string
		expected []types.Path
	}{
		{`"a"`, nil},
		{`a`, []types.Path{{"a"}}},
		{`a.b.c`, []types.Path{{"a", "b", "c"}}},
		{`a b c`, []types.Path{{"a"}, {"b"}, {"c"}}},
		{`coalesce("a", 5)`, nil},
		{`coalesce(a.b.c, 5) a.b.c b`, []types.Path{{"a", "b", "c"}, {"b"}}},
		{`coalesce(a.b.c, coalesce(b)) a.b.c b`, []types.Path{{"a", "b", "c"}, {"b"}}},
	}

	for _, test := range tests {
		expression, err := Compile(test.src, schema, types.JSON(), false)
		if err != nil {
			t.Fatalf("%q. unexpected compilation error: %s", test.src, err)
		}
		got := expression.Properties()
		if !reflect.DeepEqual(got, test.expected) {
			t.Fatalf("%q. unexpected paths\nexpected %v\ngot      %v", test.src, test.expected, got)
		}
	}

}

func TestValueOf(t *testing.T) {

	values := map[string]any{
		"a": 5,
		"b": map[string]any{
			"c": "foo",
			"d": map[string]any{
				"e": []any{1, 2, 3},
			},
		},
	}

	tests := []struct {
		path     types.Path
		expected any
		err      error
	}{
		{types.Path{"a"}, 5, nil},
		{types.Path{"b", "c"}, "foo", nil},
		{types.Path{"b", "d", "e"}, []any{1, 2, 3}, nil},
		{types.Path{"f"}, nil, errors.New("cannot find value for property path \"f\"")},
		{types.Path{"b", "g"}, nil, errors.New("cannot find value for property path \"b.g\"")},
		{types.Path{"a", "f"}, nil, errors.New("cannot find value for property path \"a.f\" (\"a\" has type int)")},
	}

	for _, test := range tests {
		got, err := valueOf(test.path, values)
		if err != nil {
			if test.err == nil {
				t.Fatalf("%q. unexpected error: %s", test.path, err)
			}
			if err.Error() != test.err.Error() {
				t.Fatalf("%q. expected error %q, got error %q", test.path, test.err.Error(), err.Error())
			}
			continue
		}
		if test.err != nil {
			t.Fatalf("%q. expected error %q, got no error", test.path, test.err)
		}
		if !reflect.DeepEqual(got, test.expected) {
			t.Fatalf("%q. unexpected value\nexpected %v (type %T)\ngot      %v (type %T)", test.path, test.expected, test.expected, got, got)
		}

	}

}
