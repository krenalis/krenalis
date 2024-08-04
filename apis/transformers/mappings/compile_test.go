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
	"time"

	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/types"

	"github.com/shopspring/decimal"
)

func Test_Compile(t *testing.T) {

	d := time.Date(2023, 7, 8, 0, 0, 0, 0, time.UTC)
	dt := time.Date(2023, 7, 8, 13, 38, 23, 0, time.UTC)

	schema := types.Object([]types.Property{
		{Name: "manufacturer", Type: types.Text()},
		{Name: "model", Type: types.Text()},
		{Name: "engine", Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text()},
			{Name: "power", Type: types.Uint(32)},
			{Name: "afterburner", Type: types.Text(), Nullable: true},
		})},
		{Name: "cx", Type: types.Decimal(4, 3)},
		{Name: "passengers", Type: types.Int(32)},
		{Name: "revision_dates", Type: types.Array(types.DateTime())},
		{Name: "map", Type: types.Map(types.Int(32))},
		{Name: "deep", Type: types.Map(types.Map(types.Object([]types.Property{
			{Name: "p", Type: types.Map(types.Int(32))},
		})))},
		{Name: "other", Type: types.Int(32), CreateRequired: true, Nullable: true},
		{Name: "properties", Type: types.JSON(), Nullable: true},
		{Name: "jsonNull", Type: types.JSON()},
		{Name: "jsonNil", Type: types.JSON(), Nullable: true},
	})

	cx := decimal.NewFromFloat(0.142)

	tests := []struct {
		expr          string
		dt            types.Type
		purpose       Purpose
		nullable      bool
		layouts       *state.TimeLayouts
		compileErr    error
		evalErr       error
		expectedValue any
	}{
		{expr: "null", dt: types.Int(32), nullable: true, expectedValue: nil},
		{expr: "null", dt: types.Int(32), nullable: false, compileErr: errors.New("cannot convert null to Int(32)")},
		{expr: "engine.afterburner", dt: types.Text(), nullable: true, expectedValue: nil},
		{expr: "engine.afterburner", dt: types.Text(), nullable: false, expectedValue: Void},
		{expr: "properties.no_key", dt: types.Int(32), nullable: true, expectedValue: Void},
		{expr: "properties.no_key", dt: types.Int(32), nullable: false, expectedValue: Void},
		{expr: "jsonNull", dt: types.Int(32), nullable: false, expectedValue: Void},
		{expr: "jsonNull", dt: types.Int(32), nullable: true, expectedValue: nil},
		{expr: "jsonNil", dt: types.Int(32), nullable: false, expectedValue: Void},
		{expr: "jsonNil", dt: types.Int(32), nullable: true, expectedValue: nil},
		{expr: "null", dt: types.JSON(), nullable: true, expectedValue: nil},
		{expr: "null", dt: types.JSON(), nullable: false, expectedValue: json.RawMessage("null")},
		{expr: "engine.afterburner", dt: types.JSON(), nullable: true, expectedValue: nil},
		{expr: "engine.afterburner", dt: types.JSON(), nullable: false, expectedValue: json.RawMessage("null")},
		{expr: "properties.no_key", dt: types.JSON(), nullable: true, expectedValue: Void},
		{expr: "properties.no_key", dt: types.JSON(), nullable: false, expectedValue: Void},
		{expr: "jsonNull", dt: types.JSON(), nullable: false, expectedValue: json.RawMessage("null")},
		{expr: "jsonNull", dt: types.JSON(), nullable: true, expectedValue: json.RawMessage("null")},
		{expr: "jsonNil", dt: types.JSON(), nullable: false, expectedValue: json.RawMessage("null")},
		{expr: "jsonNil", dt: types.JSON(), nullable: true, expectedValue: nil},

		{expr: "' '   '  '", dt: types.Text(), expectedValue: "   "},
		{expr: "''", dt: types.JSON(), expectedValue: json.RawMessage("null")},
		{expr: "'100'", dt: types.JSON(), expectedValue: json.RawMessage(`"100"`)},
		{expr: "'42'", dt: types.Int(32), expectedValue: 42},
		{expr: "'42'", dt: types.Text(), expectedValue: "42"},
		{expr: "'Afterburner: ' coalesce('yes', 'no afterburner')", dt: types.Text(), expectedValue: "Afterburner: yes"},
		{expr: "'Afterburner: ' coalesce(engine.afterburner, 'no afterburner')", dt: types.Text(), expectedValue: "Afterburner: no afterburner"},
		{expr: "'Afterburner: ' coalesce(engine.afterburner, engine.afterburner, 'no afterburner')", dt: types.Text(), expectedValue: "Afterburner: no afterburner"},
		{expr: "42", dt: types.Int(32), expectedValue: 42},
		{expr: "42", dt: types.JSON(), expectedValue: json.RawMessage("42")},
		{expr: "42", dt: types.Text(), expectedValue: "42"},
		{expr: "cx cx", dt: types.Text(), expectedValue: "0.1420.142"},
		{expr: "engine.name", dt: types.Text(), expectedValue: "TurboX"},
		{expr: "engine.power ' x ' 1.36", dt: types.Text(), expectedValue: "700 x 1.36"},
		{expr: "engine['power']", dt: types.Float(64), expectedValue: 700.0},
		{expr: "engine", dt: types.JSON(), expectedValue: json.RawMessage(`{"afterburner":null,"name":"TurboX","power":700}`)},
		{expr: "manufacturer ' ' model", dt: types.Text(), expectedValue: "MyPlaneCompany SuperFast"},
		{expr: "manufacturer", dt: types.JSON(), expectedValue: json.RawMessage("\"MyPlaneCompany\"")},
		{expr: "manufacturer", dt: types.Text(), expectedValue: "MyPlaneCompany"},
		{expr: "revision_dates", dt: types.Array(types.Date()), expectedValue: []any{d}},
		{expr: "map", dt: types.Map(types.Int(32)), expectedValue: map[string]any{"x": 1, "y": 2}},
		{expr: `""`, dt: types.JSON(), expectedValue: json.RawMessage("null")},
		{expr: "other", dt: types.Int(32), nullable: true, expectedValue: nil},
		{expr: "other", dt: types.Int(32), expectedValue: Void},

		{expr: "map['x']", dt: types.Int(32), expectedValue: 1},
		{expr: "map.x", dt: types.Int(32), expectedValue: 1},
		{expr: "map['not-exist']", dt: types.Int(32), expectedValue: Void},
		{expr: "deep['a']", dt: types.JSON(), expectedValue: json.RawMessage(`{"b":{"p":{"x":1,"y":2}}}`)},
		{expr: "deep['a']['b']", dt: types.JSON(), expectedValue: json.RawMessage(`{"p":{"x":1,"y":2}}`)},
		{expr: "deep['a']['b'].p", dt: types.JSON(), expectedValue: json.RawMessage(`{"x":1,"y":2}`)},
		{expr: "deep.a.b.p", dt: types.JSON(), expectedValue: json.RawMessage(`{"x":1,"y":2}`)},
		{expr: "deep['a']['not-exist'].p", dt: types.JSON(), expectedValue: Void},
		{expr: "deep['a']['b'].p['x']", dt: types.Int(32), expectedValue: 1},

		{expr: "properties", dt: types.JSON(), expectedValue: json.RawMessage(`{":":7,":x":8,"?":4,"[x":1,"[x]":3,"[x]?":6,"a":1,"b":{"c":[1,2]},"x?":5,"x]":2}`)},
		{expr: "properties.a", dt: types.Int(32), expectedValue: 1},
		{expr: "properties.a.x", dt: types.Int(32), evalErr: errors.New(`invalid properties.a.x: properties.a is not a JSON object, it is a JSON number`)},
		{expr: "properties.a.x?", dt: types.Int(32), expectedValue: Void},
		{expr: "properties.b.c", dt: types.Array(types.Int(32)), expectedValue: []any{1, 2}},
		{expr: "properties.b['c']", dt: types.Array(types.Int(32)), expectedValue: []any{1, 2}},
		{expr: "properties.b.x", dt: types.Array(types.Int(32)), expectedValue: Void},
		{expr: `properties["[x"]`, dt: types.Float(64), expectedValue: 1.0},
		{expr: `properties["x]"]`, dt: types.Float(64), expectedValue: 2.0},
		{expr: `properties["[x]"]`, dt: types.Float(64), expectedValue: 3.0},
		{expr: `properties["?"]`, dt: types.Float(64), expectedValue: 4.0},
		{expr: `properties["x?"]`, dt: types.Float(64), expectedValue: 5.0},
		{expr: `properties["[x]?"]`, dt: types.Float(64), expectedValue: 6.0},
		{expr: `properties[":"]`, dt: types.Float(64), expectedValue: 7.0},
		{expr: `properties[":x"]`, dt: types.Float(64), expectedValue: 8.0},

		// Compile errors.
		{expr: "!true", dt: types.Boolean(), compileErr: errors.New("unexpected character '!'")},
		{expr: "'Engine power: ' (coalesce engine.power, 0)", dt: types.Text(), compileErr: errors.New("unexpected character '('")},
		{expr: "'Engine power: ' coalesce engine.power", dt: types.Text(), compileErr: errors.New(`property "coalesce" does not exist`)},
		{expr: "'Engine power: engine.power", dt: types.Text(), compileErr: errors.New("string is not terminated")},
		{expr: "1 + 2", dt: types.Int(32), compileErr: errors.New("unexpected character '+'")},
		{expr: "engine.name?", dt: types.Text(), compileErr: errors.New("invalid engine.name?: operator '?' can be used only with JSON")},
		{expr: "engine['a name']", dt: types.Text(), compileErr: errors.New(`invalid engine["a name"]: "a name" is not a valid property name`)},
		{expr: "engine.power * 1.36", dt: types.Text(), compileErr: errors.New("unexpected character '*'")},
		{expr: "len('hello')", dt: types.Uint(32), compileErr: errors.New(`function "len" does not exist`)},
		{expr: "not true", dt: types.Boolean(), compileErr: errors.New(`property "not" does not exist`)},
		{expr: "passenger", dt: types.Text(), compileErr: errors.New(`property "passenger" does not exist`)},
		{expr: "true && false", dt: types.Boolean(), compileErr: errors.New(`unexpected character '&'`)},
		{expr: "1,000", dt: types.Int(32), compileErr: errors.New(`unexpected character ','`)},
		{expr: "true", dt: types.Int(32), compileErr: errors.New("cannot convert true (type Boolean) to Int(32)")},
		{expr: "engine", dt: types.Object([]types.Property{{Name: "notfound", Type: types.Text()}}), compileErr: errors.New("cannot convert expression (type Object) to Object")},
		{expr: "engine", dt: types.Object([]types.Property{{Name: "power", Type: types.Boolean()}}), compileErr: errors.New("cannot convert expression (type Object) to Object")},
		{expr: "revision_dates", dt: types.Array(types.Boolean()), compileErr: errors.New("cannot convert expression (type Array) to Array")},
		{expr: "map", dt: types.Map(types.Date()), compileErr: errors.New("cannot convert expression (type Map) to Map")},
		{expr: "map.x?", dt: types.Int(32), compileErr: errors.New("invalid map.x?: operator '?' can be used only with JSON")},
		{expr: "engine.pover", dt: types.Int(32), compileErr: errors.New(`invalid engine.pover: property "pover" does not exist`)},
		{expr: "engin.power", dt: types.Int(32), compileErr: errors.New(`property "engin" does not exist`)},
		{expr: "manufacturer.power", dt: types.Int(32), compileErr: errors.New(`invalid manufacturer.power: manufacturer (type Text) cannot have properties or keys`)},

		// Eval errors.
		{expr: "manufacturer", dt: types.Int(32), evalErr: errors.New(`cannot convert "MyPlaneCompany" (type Text) to type Int(32)`)},

		// and.
		{expr: "and(true, true)", dt: types.Boolean(), expectedValue: true},
		{expr: "and(true, false)", dt: types.Boolean(), expectedValue: false},
		{expr: "and(false, true)", dt: types.Boolean(), expectedValue: false},
		{expr: "and(false, false)", dt: types.Boolean(), expectedValue: false},
		{expr: "and(and(true, true), true)", dt: types.Boolean(), expectedValue: true},
		{expr: "and(true, and(true, true))", dt: types.Boolean(), expectedValue: true},
		{expr: "and(true, and(true, false))", dt: types.Boolean(), expectedValue: false},
		{expr: "and(true)", dt: types.Boolean(), compileErr: errors.New("'and' function requires at least two argument")},
		{expr: "and(1, true)", dt: types.Boolean(), compileErr: errors.New("cannot convert 1 (type Int(32)) to Boolean")},
		{expr: "and(true, true)", dt: types.Int(32), compileErr: errors.New("cannot convert expression (type Boolean) to Int(32)")},

		// array.
		{expr: "array()", dt: types.Array(types.JSON()), nullable: true, expectedValue: []any{}},
		{expr: "array(1)", dt: types.Array(types.Int(32)), nullable: false, expectedValue: []any{1}},
		{expr: "array(1, \"a\", false)", dt: types.Array(types.JSON()), nullable: true, expectedValue: []any{json.RawMessage(`1`), json.RawMessage(`"a"`), json.RawMessage(`false`)}},
		{expr: "array(array(1,2,3))", dt: types.Array(types.Array(types.Int(32))), nullable: true, expectedValue: []any{[]any{1, 2, 3}}},
		{expr: "array(null)", dt: types.Int(32), compileErr: errors.New("cannot convert expression (type Array) to Int(32)")},

		// coalesce.
		{expr: "coalesce(1, 2)", dt: types.Int(32), nullable: true, expectedValue: 1},
		{expr: "coalesce(1, null)", dt: types.Int(32), nullable: true, expectedValue: 1},
		{expr: "coalesce(null, 2)", dt: types.Int(32), nullable: true, expectedValue: 2},
		{expr: "0 coalesce(null, 2)", dt: types.Text(), nullable: true, expectedValue: "02"},
		{expr: "coalesce(null, coalesce(null, 3))", dt: types.Int(32), nullable: true, expectedValue: 3},
		{expr: "coalesce(other, 2)", dt: types.Int(32), nullable: true, expectedValue: 2},
		{expr: "coalesce(coalesce(other, null), coalesce(other, 2))", dt: types.Int(32), nullable: true, expectedValue: 2},
		{expr: "coalesce()", dt: types.Int(32), compileErr: errors.New("'coalesce' function requires at least one argument")},
		{expr: "coalesce(null)", dt: types.Int(32), compileErr: errors.New("cannot convert null to Int(32)")},
		{expr: "coalesce(1, coalesce(2, null))", dt: types.Int(32), nullable: false, compileErr: errors.New("cannot convert null to Int(32)")},
		{expr: "coalesce(1, 2)", dt: types.Boolean(), nullable: true, compileErr: errors.New("cannot convert 1 (type Int(32)) to Boolean")},
		{expr: "coalesce(coalesce(other, null), coalesce(other, 2))", dt: types.Int(32), compileErr: errors.New(`cannot convert null to Int(32)`)},

		// eq.
		{expr: "eq(1, 1)", dt: types.Boolean(), expectedValue: true},
		{expr: "eq(1, 2)", dt: types.Boolean(), expectedValue: false},
		{expr: "eq(1, '1')", dt: types.Boolean(), expectedValue: true},
		{expr: "eq('1', 1)", dt: types.Boolean(), expectedValue: true},
		{expr: "eq(1)", dt: types.Boolean(), compileErr: errors.New("'eq' function requires two arguments")},
		{expr: "eq(1, 1)", dt: types.Int(32), compileErr: errors.New("cannot convert expression (type Boolean) to Int(32)")},

		// if.
		{expr: "if(true, 1)", dt: types.Int(32), expectedValue: 1},
		{expr: "if(false, 1)", dt: types.Int(32), expectedValue: Void},
		{expr: "if(true, 1, 2)", dt: types.Int(32), expectedValue: 1},
		{expr: "if(false, 1, 2)", dt: types.Int(32), expectedValue: 2},
		{expr: "if(eq(3, 5), 1, 2)", dt: types.Int(32), expectedValue: 2},
		{expr: "if(false)", dt: types.Int(32), compileErr: errors.New("'if' function requires either two or three arguments")},
		{expr: "if(1, 2)", dt: types.Int(32), compileErr: errors.New("cannot convert 1 (type Int(32)) to Boolean")},
		{expr: "if(false, null)", dt: types.Int(32), compileErr: errors.New("cannot convert null to Int(32)")},
		{expr: "if(false, 2)", dt: types.Boolean(), compileErr: errors.New("cannot convert 2 (type Int(32)) to Boolean")},
	}

	for _, test := range tests {
		t.Run(test.expr, func(t *testing.T) {

			properties := map[string]any{
				"manufacturer": "MyPlaneCompany",
				"model":        "SuperFast",
				"engine": map[string]any{
					"name":        "TurboX",
					"power":       uint(700),
					"afterburner": nil,
				},
				"cx":             cx,
				"passengers":     250,
				"revision_dates": []any{dt},
				"map": map[string]any{
					"x": 1,
					"y": 2,
				},
				"deep": map[string]any{
					"a": map[string]any{
						"b": map[string]any{
							"p": map[string]any{
								"x": 1,
								"y": 2,
							},
						},
					},
				},
				"other": nil,
				"properties": map[string]any{
					"a": 1.0,
					"b": map[string]any{
						"c": []any{1.0, 2.0},
					},
					"[x":   1.0,
					"x]":   2.0,
					"[x]":  3.0,
					"?":    4.0,
					"x?":   5.0,
					"[x]?": 6.0,
					":":    7.0,
					":x":   8.0,
				},
				"jsonNull": json.RawMessage("null"),
				"jsonNil":  nil,
			}

			// Test Compile.
			expr, err := Compile(test.expr, schema, test.dt, test.nullable, test.layouts)
			if err != nil {
				if test.compileErr == nil {
					t.Fatalf("unexpected compile error %q (type %T)", err, err)
				}
				if !reflect.DeepEqual(test.compileErr, err) {
					t.Fatalf("expected compile error %q (type %T), got %q (type %T)", test.compileErr, test.compileErr, err, err)
				}
				return
			}
			if test.compileErr != nil {
				t.Fatalf("expecting compile error %q (type %T), got no errors", test.compileErr, test.compileErr)
			}

			// Test Eval.
			gotValue, err := expr.Eval(properties, test.purpose)
			if test.evalErr != nil {
				if err == nil {
					t.Fatalf("expected eval error %q, got no errors", test.evalErr)
				}
				if test.evalErr.Error() != err.Error() {
					t.Fatalf("expected eval error %q, got %q", test.evalErr.Error(), err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected eval error: %s", err)
			}
			if !reflect.DeepEqual(test.expectedValue, gotValue) {
				if j, ok := gotValue.(json.RawMessage); ok {
					t.Fatalf("expected value %#v, got %#v (which represents the string %q)", test.expectedValue, gotValue, string(j))
				}
				t.Fatalf("expected value %#v, got %#v", test.expectedValue, gotValue)
			}

		})
	}

}

func TestInvalidSchema(t *testing.T) {

	tests := []struct {
		expr string
		dt   types.Type
	}{
		{expr: "''", dt: types.Text()},
		{expr: "5", dt: types.Int(32)},
		{expr: "eq(2, 2)", dt: types.Boolean()},
	}

	for _, test := range tests {
		t.Run(test.expr, func(t *testing.T) {
			_, err := Compile(test.expr, types.Type{}, test.dt, false, nil)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
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
		{Name: "d", Type: types.Map(types.Object([]types.Property{
			{Name: "e", Type: types.Text()},
		}))},
		{Name: "e", Type: types.JSON()},
		{Name: "f", Type: types.Map(types.JSON())},
	})

	tests := []struct {
		src      string
		expected []string
	}{
		{`"a"`, nil},
		{`a`, []string{"a"}},
		{`a.b.c`, []string{"a.b.c"}},
		{`b c`, []string{"b", "c"}},
		{`d['foo']`, []string{"d"}},
		{`d.boo.e`, []string{"d.e"}},
		{`e`, []string{"e"}},
		{`e.foo`, []string{"e"}},
		{`e.foo?.boo`, []string{"e"}},
		{`f.foo`, []string{"f"}},
		{`f.foo.boo`, []string{"f"}},
		{`coalesce("a", 5)`, nil},
		{`coalesce(a.b.c, 5) a.b.c b`, []string{"a.b.c", "b"}},
		{`coalesce(a.b.c, coalesce(b)) a.b.c b`, []string{"a.b.c", "b"}},
	}

	for _, test := range tests {
		expression, err := Compile(test.src, schema, types.JSON(), true, nil)
		if err != nil {
			t.Fatalf("%q. unexpected compilation error: %s", test.src, err)
		}
		got := expression.Properties()
		if !reflect.DeepEqual(got, test.expected) {
			t.Fatalf("%q. unexpected paths\nexpected %v\ngot      %v", test.src, test.expected, got)
		}
	}

}
