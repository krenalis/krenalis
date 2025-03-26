//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/google/go-cmp/cmp"
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

	cx := decimal.MustParse("0.142")

	tests := []struct {
		expr       string
		dt         types.Type
		purpose    Purpose
		nullable   bool
		layouts    *state.TimeLayouts
		compileErr error
		evalErr    error
		convertErr error
		expected   any
	}{
		{expr: "null", dt: types.Int(32), nullable: true, expected: nil},
		{expr: "null", dt: types.Int(32), nullable: false, expected: nil},
		{expr: "engine.afterburner", dt: types.Text(), nullable: true, expected: nil},
		{expr: "engine.afterburner", dt: types.Text(), nullable: false, expected: nil},
		{expr: "properties.no_key", dt: types.Int(32), nullable: true, expected: nil},
		{expr: "properties.no_key", dt: types.Int(32), nullable: false, expected: nil},
		{expr: "jsonNull", dt: types.Int(32), nullable: false, expected: nil},
		{expr: "jsonNull", dt: types.Int(32), nullable: true, expected: nil},
		{expr: "jsonNil", dt: types.Int(32), nullable: false, expected: nil},
		{expr: "jsonNil", dt: types.Int(32), nullable: true, expected: nil},
		{expr: "null", dt: types.JSON(), nullable: true, expected: nil},
		{expr: "null", dt: types.JSON(), nullable: false, expected: nil},
		{expr: "engine.afterburner", dt: types.JSON(), nullable: true, expected: nil},
		{expr: "engine.afterburner", dt: types.JSON(), nullable: false, expected: nil},
		{expr: "properties.no_key", dt: types.JSON(), nullable: true, expected: nil},
		{expr: "properties.no_key", dt: types.JSON(), nullable: false, expected: nil},
		{expr: "jsonNull", dt: types.JSON(), nullable: false, expected: json.Value("null")},
		{expr: "jsonNull", dt: types.JSON(), nullable: true, expected: json.Value("null")},
		{expr: "jsonNil", dt: types.JSON(), nullable: false, expected: nil},
		{expr: "jsonNil", dt: types.JSON(), nullable: true, expected: nil},

		{expr: "' '   '  '", dt: types.Text(), expected: "   "},
		{expr: "''", dt: types.JSON(), expected: json.Value(`""`)},
		{expr: "'100'", dt: types.JSON(), expected: json.Value(`"100"`)},
		{expr: "'42'", dt: types.Int(32), expected: 42},
		{expr: "'42'", dt: types.Text(), expected: "42"},
		{expr: "'Afterburner: ' coalesce('yes', 'no afterburner')", dt: types.Text(), expected: "Afterburner: yes"},
		{expr: "'Afterburner: ' coalesce(engine.afterburner, 'no afterburner')", dt: types.Text(), expected: "Afterburner: no afterburner"},
		{expr: "'Afterburner: ' coalesce(engine.afterburner, engine.afterburner, 'no afterburner')", dt: types.Text(), expected: "Afterburner: no afterburner"},
		{expr: "42", dt: types.Int(32), expected: 42},
		{expr: "42", dt: types.JSON(), expected: json.Value("42")},
		{expr: "42", dt: types.Text(), expected: "42"},
		{expr: "cx cx", dt: types.Text(), expected: "0.1420.142"},
		{expr: "engine.name", dt: types.Text(), expected: "TurboX"},
		{expr: "engine.power ' x ' 1.36", dt: types.Text(), expected: "7000 x 1.36"},
		{expr: "engine['power']", dt: types.Float(64), expected: 7000.0},
		{expr: "engine", dt: types.JSON(), expected: json.Value(`{"afterburner":null,"name":"TurboX","power":7000}`)},
		{expr: "manufacturer ' ' model", dt: types.Text(), expected: "MyPlaneCompany SuperFast"},
		{expr: "manufacturer", dt: types.JSON(), expected: json.Value(`"MyPlaneCompany"`)},
		{expr: "manufacturer", dt: types.Text(), expected: "MyPlaneCompany"},
		{expr: "revision_dates", dt: types.Array(types.Date()), expected: []any{d}},
		{expr: "map", dt: types.Map(types.Int(32)), expected: map[string]any{"x": 1, "y": 2}},
		{expr: `""`, dt: types.JSON(), expected: json.Value(`""`)},
		{expr: "other", dt: types.Int(32), nullable: true, expected: nil},
		{expr: "other", dt: types.Int(32), expected: nil},

		{expr: "map['x']", dt: types.Int(32), expected: 1},
		{expr: "map.x", dt: types.Int(32), expected: 1},
		{expr: "map['not-exist']", dt: types.Int(32), expected: nil},
		{expr: "deep['a']", dt: types.JSON(), expected: json.Value(`{"b":{"p":{"x":1,"y":2}}}`)},
		{expr: "deep['a']['b']", dt: types.JSON(), expected: json.Value(`{"p":{"x":1,"y":2}}`)},
		{expr: "deep['a']['b'].p", dt: types.JSON(), expected: json.Value(`{"x":1,"y":2}`)},
		{expr: "deep.a.b.p", dt: types.JSON(), expected: json.Value(`{"x":1,"y":2}`)},
		{expr: "deep['a']['not-exist'].p", dt: types.JSON(), expected: nil},
		{expr: "deep['a']['b'].p['x']", dt: types.Int(32), expected: 1},

		{expr: "properties", dt: types.JSON(), expected: json.Value(`{"a":1.0,"b":{"c":[1.0,2.0]},"[x":1.0,"x]":2.0,"[x]":3.0,"?":4.0,"x?":5.0,"[x]?":6.0,":":7.0,":x":8.0}`)},
		{expr: "properties.a", dt: types.Int(32), expected: 1},
		{expr: "properties.a.x", dt: types.Int(32), evalErr: errors.New(`invalid properties.a.x: properties.a is not JSON object, it is number`)},
		{expr: "properties.a.x?", dt: types.Int(32), expected: nil},
		{expr: "properties.b.c", dt: types.Array(types.Int(32)), expected: []any{1, 2}},
		{expr: "properties.b['c']", dt: types.Array(types.Int(32)), expected: []any{1, 2}},
		{expr: "properties.b.x", dt: types.Array(types.Int(32)), expected: nil},
		{expr: `properties["[x"]`, dt: types.Float(64), expected: 1.0},
		{expr: `properties["x]"]`, dt: types.Float(64), expected: 2.0},
		{expr: `properties["[x]"]`, dt: types.Float(64), expected: 3.0},
		{expr: `properties["?"]`, dt: types.Float(64), expected: 4.0},
		{expr: `properties["x?"]`, dt: types.Float(64), expected: 5.0},
		{expr: `properties["[x]?"]`, dt: types.Float(64), expected: 6.0},
		{expr: `properties[":"]`, dt: types.Float(64), expected: 7.0},
		{expr: `properties[":x"]`, dt: types.Float(64), expected: 8.0},

		// Compile errors.
		{expr: "!true", dt: types.Boolean(), compileErr: errors.New("unexpected character '!'")},
		{expr: "'Engine power: ' (coalesce engine.power, 0)", dt: types.Text(), compileErr: errors.New("unexpected character '('")},
		{expr: "'Engine power: ' coalesce engine.power", dt: types.Text(), compileErr: errors.New(`property "coalesce" does not exist`)},
		{expr: "'Engine power: engine.power", dt: types.Text(), compileErr: errors.New("string is not terminated")},
		{expr: "1 + 2", dt: types.Int(32), compileErr: errors.New("unexpected character '+'")},
		{expr: "engine.name?", dt: types.Text(), compileErr: errors.New("invalid engine.name?: operator '?' can be used only with json")},
		{expr: "engine['a name']", dt: types.Text(), compileErr: errors.New(`invalid engine["a name"]: "a name" is not a valid property name`)},
		{expr: "engine.power * 1.36", dt: types.Text(), compileErr: errors.New("unexpected character '*'")},
		{expr: "foo('hello')", dt: types.Uint(32), compileErr: errors.New(`function "foo" does not exist`)},
		{expr: "not true", dt: types.Boolean(), compileErr: errors.New(`property "not" does not exist`)},
		{expr: "passenger", dt: types.Text(), compileErr: errors.New(`property "passenger" does not exist`)},
		{expr: "true && false", dt: types.Boolean(), compileErr: errors.New(`unexpected character '&'`)},
		{expr: "1,000", dt: types.Int(32), compileErr: errors.New(`unexpected character ','`)},
		{expr: "true", dt: types.Int(32), compileErr: errors.New("cannot convert true (type boolean) to int(32)")},
		{expr: "engine", dt: types.Object([]types.Property{{Name: "notfound", Type: types.Text()}}), compileErr: errors.New("cannot convert expression (type object) to object")},
		{expr: "engine", dt: types.Object([]types.Property{{Name: "power", Type: types.Boolean()}}), compileErr: errors.New("cannot convert expression (type object) to object")},
		{expr: "revision_dates", dt: types.Array(types.Boolean()), compileErr: errors.New("cannot convert expression (type array) to array")},
		{expr: "map", dt: types.Map(types.Date()), compileErr: errors.New("cannot convert expression (type map) to map")},
		{expr: "map.x?", dt: types.Int(32), compileErr: errors.New("invalid map.x?: operator '?' can be used only with json")},
		{expr: "engine.pover", dt: types.Int(32), compileErr: errors.New(`invalid engine.pover: property "pover" does not exist`)},
		{expr: "engin.power", dt: types.Int(32), compileErr: errors.New(`property "engin" does not exist`)},
		{expr: "manufacturer.power", dt: types.Int(32), compileErr: errors.New(`invalid manufacturer.power: manufacturer (type text) cannot have properties or keys`)},

		// Eval errors.
		{expr: "manufacturer", dt: types.Int(32), convertErr: errors.New(`cannot convert`)},

		// and.
		{expr: "and(true, true)", dt: types.Boolean(), expected: true},
		{expr: "and(true, false)", dt: types.Boolean(), expected: false},
		{expr: "and(false, true)", dt: types.Boolean(), expected: false},
		{expr: "and(false, false)", dt: types.Boolean(), expected: false},
		{expr: "and(null, false)", dt: types.Boolean(), expected: false},
		{expr: "and(true, null)", dt: types.Boolean(), expected: nil},
		{expr: "and(and(true, true), true)", dt: types.Boolean(), expected: true},
		{expr: "and(true, and(true, true))", dt: types.Boolean(), expected: true},
		{expr: "and(true, and(true, false))", dt: types.Boolean(), expected: false},
		{expr: "and(true)", dt: types.Boolean(), compileErr: errors.New("'and' function requires at least two argument")},
		{expr: "and(1, true)", dt: types.Boolean(), compileErr: errors.New("cannot convert 1 (type int(32)) to boolean")},
		{expr: "and(true, true)", dt: types.Int(32), compileErr: errors.New("cannot convert expression (type boolean) to int(32)")},
		{expr: "and()", dt: types.Int(32), compileErr: errors.New("'and' function requires at least two argument")},
		{expr: "and('true', 'true')", dt: types.Boolean(), expected: true},

		// array.
		{expr: "array()", dt: types.Array(types.JSON()), nullable: true, expected: []any{}},
		{expr: "array(1)", dt: types.Array(types.Int(32)), nullable: false, expected: []any{1}},
		{expr: "array(1, \"a\", false)", dt: types.Array(types.JSON()), nullable: true, expected: []any{json.Value("1"), json.Value(`"a"`), json.Value("false")}},
		{expr: "array(array(1,2,3))", dt: types.Array(types.Array(types.Int(32))), nullable: true, expected: []any{[]any{1, 2, 3}}},
		{expr: "array(null)", dt: types.Array(types.JSON()), expected: []any{json.Value("null")}},

		// coalesce.
		{expr: "coalesce(1, 2)", dt: types.Int(32), nullable: true, expected: 1},
		{expr: "coalesce(1, null)", dt: types.Int(32), nullable: true, expected: 1},
		{expr: "coalesce(null, 2)", dt: types.Int(32), nullable: true, expected: 2},
		{expr: "coalesce(null, null)", dt: types.Int(32), nullable: false, expected: nil},
		{expr: "0 coalesce(null, 2)", dt: types.Text(), nullable: true, expected: "02"},
		{expr: "coalesce(null, coalesce(null, 3))", dt: types.Int(32), nullable: true, expected: 3},
		{expr: "coalesce(other, 2)", dt: types.Int(32), nullable: true, expected: 2},
		{expr: "coalesce(coalesce(other, null), coalesce(other, 2))", dt: types.Int(32), nullable: true, expected: 2},
		{expr: "coalesce()", dt: types.Int(32), compileErr: errors.New("'coalesce' function requires at least one argument")},
		{expr: "coalesce(null)", dt: types.Int(32), expected: nil},
		{expr: "coalesce(1, coalesce(2, null))", dt: types.Int(32), nullable: false, expected: 1},
		{expr: "coalesce(1, 2)", dt: types.Boolean(), nullable: true, compileErr: errors.New("cannot convert 1 (type int(32)) to boolean")},
		{expr: "coalesce(coalesce(other, null), coalesce(other, 2))", dt: types.Int(32), expected: 2},
		{expr: "coalesce()", dt: types.Int(32), compileErr: errors.New("'coalesce' function requires at least one argument")},

		// eq.
		{expr: "eq(1, 1)", dt: types.Boolean(), expected: true},
		{expr: "eq(1, 2)", dt: types.Boolean(), expected: false},
		{expr: "eq(1, '1')", dt: types.Boolean(), expected: true},
		{expr: "eq('1', 1)", dt: types.Boolean(), expected: true},
		{expr: "eq(1, null)", dt: types.Boolean(), nullable: true, expected: nil},
		{expr: "eq(null, 2)", dt: types.Boolean(), nullable: true, expected: nil},
		{expr: "eq(null, null)", dt: types.Boolean(), nullable: true, expected: nil},
		{expr: "eq(1)", dt: types.Boolean(), compileErr: errors.New("'eq' function requires two arguments")},
		{expr: "eq(1, 1)", dt: types.Int(32), compileErr: errors.New("cannot convert expression (type boolean) to int(32)")},
		{expr: "eq()", dt: types.Int(32), compileErr: errors.New("'eq' function requires two arguments")},
		{expr: "eq(1, 2, 3)", dt: types.Int(32), compileErr: errors.New("'eq' function requires two arguments")},

		// if.
		{expr: "if(true, 1)", dt: types.Int(32), expected: 1},
		{expr: "if(false, 1)", dt: types.Int(32), expected: nil},
		{expr: "if(true, 1, 2)", dt: types.Int(32), expected: 1},
		{expr: "if(false, 1, 2)", dt: types.Int(32), expected: 2},
		{expr: "if(null, 1, 2)", dt: types.Int(32), expected: 2},
		{expr: "if(true, 1)", dt: types.Int(32), expected: 1},
		{expr: "if(false, 1)", dt: types.Int(32), expected: nil},
		{expr: "if(null, 1)", dt: types.Int(32), expected: nil},
		{expr: "if(eq(3, 5), 1, 2)", dt: types.Int(32), expected: 2},
		{expr: "if(false)", dt: types.Int(32), compileErr: errors.New("'if' function requires either two or three arguments")},
		{expr: "if(1, 2)", dt: types.Int(32), compileErr: errors.New("cannot convert 1 (type int(32)) to boolean")},
		{expr: "if(false, null)", dt: types.Int(32), expected: nil},
		{expr: "if(true, 1, null)", dt: types.Int(32), expected: 1},
		{expr: "if(false, 2)", dt: types.Boolean(), compileErr: errors.New("cannot convert 2 (type int(32)) to boolean")},
		{expr: "if()", dt: types.Boolean(), compileErr: errors.New("'if' function requires either two or three arguments")},
		{expr: "if(1, 2, 3, 4)", dt: types.Boolean(), compileErr: errors.New("'if' function requires either two or three arguments")},
		{expr: "if('true', 1, 2)", dt: types.Int(32), expected: 1},

		// initcap.
		{expr: "initcap('new york')", dt: types.Text(), expected: "New York"},
		{expr: "initcap(' new york ')", dt: types.Text(), expected: " New York "},
		{expr: "initcap('neW YORK')", dt: types.Text(), expected: "NeW YORK"},
		{expr: "initcap(null)", dt: types.Text(), expected: nil},
		{expr: "initcap()", dt: types.Text(), compileErr: errors.New("'initcap' function requires a single argument")},
		{expr: "initcap('a', 5)", dt: types.Text(), compileErr: errors.New("'initcap' function requires a single argument")},
		{expr: "initcap(true)", dt: types.Text(), expected: "True"},

		// json_parse.
		{expr: "json_parse('')", dt: types.JSON(), evalErr: errors.New("«''» cannot be parsed by «json_parse» because it is not valid JSON")},
		{expr: `json_parse('"foo"')`, dt: types.JSON(), expected: json.Value(`"foo"`)},
		{expr: `json_parse('[1, 2, 3]')`, dt: types.JSON(), expected: json.Value(`[1, 2, 3]`)},
		{expr: `json_parse(' {"a": 5, "b": true }')`, dt: types.JSON(), expected: json.Value(` {"a": 5, "b": true }`)},
		{expr: `json_parse('true')`, dt: types.JSON(), expected: json.Value(`true`)},
		{expr: `json_parse('null')`, dt: types.JSON(), expected: json.Value(`null`)},
		{expr: `json_parse('foo boo')`, dt: types.JSON(), evalErr: errors.New("«'foo boo'» cannot be parsed by «json_parse» because it is not valid JSON")},
		{expr: `json_parse(null)`, dt: types.JSON(), expected: nil},
		{expr: `json_parse(false)`, dt: types.JSON(), expected: json.Value(`false`)},
		{expr: `json_parse(json_parse('"\\"a\\""'))`, dt: types.JSON(), expected: json.Value(`"a"`)},

		// len.
		{expr: "len('')", dt: types.Int(32), expected: 0},
		{expr: "len('Hello World')", dt: types.Int(32), expected: 11},
		{expr: "len(array(1, 2, 3))", dt: types.Int(32), expected: 3},
		{expr: "len(map)", dt: types.Int(32), expected: 2},
		{expr: "len(engine)", dt: types.Int(32), expected: 3},
		{expr: "len(null)", dt: types.Int(32), expected: 0},
		{expr: "len(engine.power)", dt: types.Int(32), expected: 4},
		{expr: "len(passengers)", dt: types.Int(32), expected: 3},
		{expr: "len(deep)", dt: types.Int(32), expected: 1},
		{expr: "len(cx)", dt: types.Int(32), expected: 5},
		{expr: "len(other)", dt: types.Int(32), expected: 0},
		{expr: "len(properties)", dt: types.Int(32), expected: 10},
		{expr: "len(jsonNull)", dt: types.Int(32), expected: 0},
		{expr: "len(jsonNil)", dt: types.Int(32), expected: 0},

		// lower.
		{expr: "lower('')", dt: types.Text(), expected: ""},
		{expr: "lower('New York')", dt: types.Text(), expected: "new york"},
		{expr: "lower('new york')", dt: types.Text(), expected: "new york"},
		{expr: "lower()", dt: types.Text(), compileErr: errors.New("'lower' function requires a single argument")},
		{expr: "lower('New', 'York')", dt: types.Text(), compileErr: errors.New("'lower' function requires a single argument")},
		{expr: "lower(false)", dt: types.Text(), expected: "false"},

		// ltrim.
		{expr: "ltrim('')", dt: types.Text(), expected: ""},
		{expr: "ltrim('New York')", dt: types.Text(), expected: "New York"},
		{expr: "ltrim(' New York\t\n')", dt: types.Text(), expected: "New York\t\n"},
		{expr: "ltrim('\t\n \nNew \n York ')", dt: types.Text(), expected: "New \n York "},
		{expr: "ltrim()", dt: types.Text(), compileErr: errors.New("'ltrim' function requires a single argument")},
		{expr: "ltrim(' New', 'York ')", dt: types.Text(), compileErr: errors.New("'ltrim' function requires a single argument")},
		{expr: "ltrim(null)", dt: types.Text(), expected: nil},
		{expr: "ltrim(12.67)", dt: types.Text(), expected: "12.67"},

		// ne.
		{expr: "ne(1, 2)", dt: types.Boolean(), expected: true},
		{expr: "ne(1, 1)", dt: types.Boolean(), expected: false},
		{expr: "ne(1, '1')", dt: types.Boolean(), expected: false},
		{expr: "ne('1', 1)", dt: types.Boolean(), expected: false},
		{expr: "ne('2', 1)", dt: types.Boolean(), expected: true},
		{expr: "ne(1, null)", dt: types.Boolean(), nullable: true, expected: nil},
		{expr: "ne(null, 2)", dt: types.Boolean(), nullable: true, expected: nil},
		{expr: "ne(null, null)", dt: types.Boolean(), nullable: true, expected: nil},
		{expr: "ne(1)", dt: types.Boolean(), compileErr: errors.New("'ne' function requires two arguments")},
		{expr: "ne(1, 2)", dt: types.Int(32), compileErr: errors.New("cannot convert expression (type boolean) to int(32)")},
		{expr: "ne()", dt: types.Int(32), compileErr: errors.New("'ne' function requires two arguments")},
		{expr: "ne(1, 2, 3)", dt: types.Int(32), compileErr: errors.New("'ne' function requires two arguments")},

		// not.
		{expr: "not(true)", dt: types.Boolean(), expected: false},
		{expr: "not(false)", dt: types.Boolean(), expected: true},
		{expr: "not(null)", dt: types.Boolean(), expected: nil},
		{expr: "not(true, false)", dt: types.Boolean(), compileErr: errors.New("'not' function requires a single argument")},
		{expr: "not()", dt: types.Boolean(), compileErr: errors.New("'not' function requires a single argument")},
		{expr: "not('false')", dt: types.Boolean(), expected: true},

		// rtrim.
		{expr: "rtrim('')", dt: types.Text(), expected: ""},
		{expr: "rtrim('New York')", dt: types.Text(), expected: "New York"},
		{expr: "rtrim(' New York\t\n')", dt: types.Text(), expected: " New York"},
		{expr: "rtrim(' New \n York\t\n \n')", dt: types.Text(), expected: " New \n York"},
		{expr: "rtrim()", dt: types.Text(), compileErr: errors.New("'rtrim' function requires a single argument")},
		{expr: "rtrim(' New', 'York ')", dt: types.Text(), compileErr: errors.New("'rtrim' function requires a single argument")},
		{expr: "rtrim(null)", dt: types.Text(), expected: nil},

		// substring.
		{expr: "substring('Hello World', 7, 5)", dt: types.Text(), expected: "World"},
		{expr: "substring('Hello World', -5, 5)", dt: types.Text(), expected: "Hello"},
		{expr: "substring('Hello World', 0)", dt: types.Text(), expected: "Hello World"},
		{expr: "substring(null, 3, 12)", dt: types.Text(), expected: nil},
		{expr: "substring('Hello World', null, 12)", dt: types.Text(), expected: nil},
		{expr: "substring('Hello World', 3, null)", dt: types.Text(), expected: nil},
		{expr: "substring('Hello World', null)", dt: types.Text(), expected: nil},
		{expr: "substring('Hello World', 3, -2)", dt: types.Text(), evalErr: errors.New("substring: negative substring length is not allowed")},
		{expr: "substring(250, 2, 2)", dt: types.Int(32), expected: 50},
		{expr: "substring('Hello World', 'a', 2)", dt: types.Int(32), compileErr: errors.New("cannot convert a (type text) to int(32)")},
		{expr: "substring('Hello World', 0, 'b')", dt: types.Int(32), compileErr: errors.New("cannot convert b (type text) to int(32)")},
		{expr: "substring()", dt: types.Int(32), compileErr: errors.New("'substring' function requires two or three arguments")},
		{expr: "substring('a', 3, 6, 8)", dt: types.Int(32), compileErr: errors.New("'substring' function requires two or three arguments")},

		// or.
		{expr: "or(true, true)", dt: types.Boolean(), expected: true},
		{expr: "or(true, false)", dt: types.Boolean(), expected: true},
		{expr: "or(false, true)", dt: types.Boolean(), expected: true},
		{expr: "or(false, false)", dt: types.Boolean(), expected: false},
		{expr: "or(null, false)", dt: types.Boolean(), expected: nil},
		{expr: "or(true, null)", dt: types.Boolean(), expected: true},
		{expr: "or(or(false, true), true)", dt: types.Boolean(), expected: true},
		{expr: "or(true, or(true, false))", dt: types.Boolean(), expected: true},
		{expr: "or(false, or(false, false))", dt: types.Boolean(), expected: false},
		{expr: "or(false)", dt: types.Boolean(), compileErr: errors.New("'or' function requires at least two argument")},
		{expr: "or(1, true)", dt: types.Boolean(), compileErr: errors.New("cannot convert 1 (type int(32)) to boolean")},
		{expr: "or(true, false)", dt: types.Int(32), compileErr: errors.New("cannot convert expression (type boolean) to int(32)")},
		{expr: "or()", dt: types.Int(32), compileErr: errors.New("'or' function requires at least two argument")},
		{expr: "or(1)", dt: types.Int(32), compileErr: errors.New("'or' function requires at least two argument")},
		{expr: "or('false', 'true')", dt: types.Boolean(), expected: true},

		// trim.
		{expr: "trim('')", dt: types.Text(), expected: ""},
		{expr: "trim('New York')", dt: types.Text(), expected: "New York"},
		{expr: "trim(' New York\t\n')", dt: types.Text(), expected: "New York"},
		{expr: "trim('\t\n \r\nNew \n York  ')", dt: types.Text(), expected: "New \n York"},
		{expr: "trim()", dt: types.Text(), compileErr: errors.New("'trim' function requires a single argument")},
		{expr: "trim(' New', 'York ')", dt: types.Text(), compileErr: errors.New("'trim' function requires a single argument")},
		{expr: "trim(null)", dt: types.Text(), expected: nil},
		{expr: "trim(55)", dt: types.Text(), expected: "55"},

		// upper.
		{expr: "upper('')", dt: types.Text(), expected: ""},
		{expr: "upper('New York')", dt: types.Text(), expected: "NEW YORK"},
		{expr: "upper('NEW YORK')", dt: types.Text(), expected: "NEW YORK"},
		{expr: "upper()", dt: types.Text(), compileErr: errors.New("'upper' function requires a single argument")},
		{expr: "upper('New', 'York')", dt: types.Text(), compileErr: errors.New("'upper' function requires a single argument")},
		{expr: "upper(false)", dt: types.Text(), expected: "FALSE"},
	}

	encodeSorted = true
	defer func() {
		encodeSorted = false
	}()

	for _, test := range tests {
		t.Run(test.expr, func(t *testing.T) {

			properties := map[string]any{
				"manufacturer": "MyPlaneCompany",
				"model":        "SuperFast",
				"engine": map[string]any{
					"name":        "TurboX",
					"power":       uint(7000),
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
				"other":      nil,
				"properties": json.Value(`{"a":1.0,"b":{"c":[1.0,2.0]},"[x":1.0,"x]":2.0,"[x]":3.0,"?":4.0,"x?":5.0,"[x]?":6.0,":":7.0,":x":8.0}`),
				"jsonNull":   json.Value("null"),
				"jsonNil":    nil,
				"ip":         " 127.0.0.1  ",
				"uuid":       "\n2e3e96c4-2bc8-41d5-b492-bd6745190691 ",
			}

			// Test Compile.
			expr, _, err := Compile(test.expr, schema, test.dt)
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
			v, vt, err := expr.Eval(properties)
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
			if v != nil {
				v, err = convert(v, vt, test.dt, true, false, test.layouts, test.purpose)
			}
			if test.convertErr != nil {
				if err == nil {
					t.Fatalf("expected convert error %q, got no errors", test.convertErr)
				}
				if test.convertErr.Error() != err.Error() {
					t.Fatalf("expected convert error %q, got %q", test.convertErr.Error(), err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected convert error: %s", err)
			}
			if !cmp.Equal(test.expected, v) {
				if j, ok := v.(json.Value); ok {
					t.Fatalf("expected value %#v, got %#v (which represents the string %q)", test.expected, v, string(j))
				}
				t.Fatalf("unexpected value:\n\texpected: %#v\n\tgot:      %#v\n\n%s\n", test.expected, v, cmp.Diff(test.expected, v))
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
			_, _, err := Compile(test.expr, types.Type{}, test.dt)
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
		_, got, err := Compile(test.src, schema, types.JSON())
		if err != nil {
			t.Fatalf("%q. unexpected compilation error: %s", test.src, err)
		}
		if !reflect.DeepEqual(got, test.expected) {
			t.Fatalf("%q. unexpected paths\nexpected %v\ngot      %v", test.src, test.expected, got)
		}
	}

}
