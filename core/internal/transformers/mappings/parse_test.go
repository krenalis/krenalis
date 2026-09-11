// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

func Test_parseExpression(t *testing.T) {

	n := decimal.MustParse(`-6.803`)
	dt := types.Decimal(types.MaxDecimalPrecision, types.MaxDecimalScale)

	tests := []struct {
		src      string
		expected []part
		unparsed string
		err      error
	}{
		{`"Page View"`, []part{{value: `Page View`, typ: types.String(), end: 11}}, ``, nil},
		{` 'Page View' `, []part{{value: `Page View`, typ: types.String(), end: 13}}, ``, nil},
		{`51`, []part{{value: 51, typ: types.Int(32), end: 2}}, ``, nil},
		{`-6.803`, []part{{value: n, typ: dt, end: 6}}, ``, nil},
		{`true`, []part{{value: true, typ: types.Boolean(), end: 4}}, ``, nil},
		{`false`, []part{{value: false, typ: types.Boolean(), end: 5}}, ``, nil},
		{`null`, []part{{value: nil, typ: types.JSON(), end: 4}}, ``, nil},
		{`name`, []part{{path: path{elements: []string{`name`}, decorators: []decorators{0}}, end: 4}}, ``, nil},
		{`.name`, []part{{path: path{elements: []string{`name`}, decorators: []decorators{0}}, end: 5}}, ``, nil},
		{`context.os.version`, []part{{path: path{elements: []string{`context`, `os`, `version`}, decorators: []decorators{0, 0, 0}}, end: 18}}, ``, nil},
		{`.context.os.version`, []part{{path: path{elements: []string{`context`, `os`, `version`}, decorators: []decorators{0, 0, 0}}, end: 19}}, ``, nil},
		{`"Page " name`, []part{{value: `Page `, path: path{elements: []string{`name`}, decorators: []decorators{0}}, typ: types.String(), end: 12}}, ``, nil},
		{`"OS " context.os.name " (" context.os.version ")"`, []part{
			{value: `OS `, path: path{elements: []string{`context`, `os`, `name`}, decorators: []decorators{0, 0, 0}}, typ: types.String(), end: 22},
			{value: ` (`, path: path{elements: []string{`context`, `os`, `version`}, decorators: []decorators{0, 0, 0}}, typ: types.String(), start: 22, end: 46},
			{value: `)`, typ: types.String(), start: 46, end: 49},
		}, ``, nil},
		{`coalesce(event, 'Page ' true)`, []part{
			{path: path{elements: []string{`coalesce`}, decorators: []decorators{0}}, args: [][]part{
				{{path: path{elements: []string{`event`}, decorators: []decorators{0}}, start: 9, end: 14}},
				{{value: `Page true`, typ: types.String(), start: 16, end: 28}},
			}, end: 29},
		}, ``, nil},
		{`"" event`, []part{{value: ``, path: path{elements: []string{`event`}, decorators: []decorators{0}}, typ: types.String(), end: 8}}, ``, nil},
		{`coalesce(a)`, []part{{path: path{elements: []string{`coalesce`}, decorators: []decorators{0}}, args: [][]part{{{path: path{elements: []string{`a`}, decorators: []decorators{0}}, start: 9, end: 10}}}, end: 11}}, ``, nil},
		{`coalesce(a, 'b')`, []part{{path: path{elements: []string{`coalesce`}, decorators: []decorators{0}}, args: [][]part{{{path: path{elements: []string{`a`}, decorators: []decorators{0}}, start: 9, end: 10}}, {{value: `b`, typ: types.String(), start: 12, end: 15}}}, end: 16}}, ``, nil},
		{`coalesce(5, 'a', coalesce(b))`, []part{{path: path{elements: []string{`coalesce`}, decorators: []decorators{0}}, args: [][]part{
			{{value: 5, typ: types.Int(32), start: 9, end: 10}}, {{value: `a`, typ: types.String(), start: 12, end: 15}}, {{path: path{elements: []string{`coalesce`}, decorators: []decorators{0}}, args: [][]part{{{path: path{elements: []string{`b`}, decorators: []decorators{0}}, start: 26, end: 27}}}, start: 17, end: 28}},
		}, end: 29}}, ``, nil},
		{`coalesce("a" coalesce(b, 5) c)`, []part{{path: path{elements: []string{`coalesce`}, decorators: []decorators{0}}, args: [][]part{
			{{value: `a`, path: path{elements: []string{`coalesce`}, decorators: []decorators{0}}, args: [][]part{{{path: path{elements: []string{`b`}, decorators: []decorators{0}}, start: 22, end: 23}}, {{value: 5, typ: types.Int(32), start: 25, end: 26}}}, typ: types.String(), start: 9, end: 27}, {path: path{elements: []string{`c`}, decorators: []decorators{0}}, start: 27, end: 29}},
		}, end: 30}}, ``, nil},
		{`coalesce( coalesce ( x , 5 ) )`, []part{{path: path{elements: []string{`coalesce`}, decorators: []decorators{0}}, args: [][]part{
			{{path: path{elements: []string{`coalesce`}, decorators: []decorators{0}}, args: [][]part{{{path: path{elements: []string{`x`}, decorators: []decorators{0}}, start: 21, end: 23}}, {{value: 5, typ: types.Int(32), start: 25, end: 27}}}, start: 10, end: 28}},
		}, end: 30}}, ``, nil},
		{`  coalesce ( a ) `, []part{{path: path{elements: []string{`coalesce`}, decorators: []decorators{0}}, args: [][]part{{{path: path{elements: []string{`a`}, decorators: []decorators{0}}, start: 13, end: 15}}}, end: 16}}, ``, nil},
		{`coalesce( , )`, nil, ``, errors.New("expected argument, got ','")},
		{`coalesce(a, )`, nil, ``, errors.New("expected argument, got ')'")},
		{`coalesce( @`, nil, ``, errors.New("expected argument, got '@'")},
		{``, nil, ``, nil},
		{" \t\n \t", nil, ``, nil},
	}

	for _, test := range tests {
		got, src, err := parse(test.src, 0, len(test.src), 0)
		if err != nil {
			if test.err == nil {
				t.Fatalf("%q. unexpected error: %s", test.src, err)
			}
			if err.Error() != test.err.Error() {
				t.Fatalf("%q. expected error %q, got error %q", test.src, test.err.Error(), err.Error())
			}
			continue
		}
		if test.err != nil {
			t.Fatalf("%q. expected error %q, got no error", test.src, test.err)
		}
		if !reflect.DeepEqual(got, test.expected) {
			t.Fatalf("%q\nexpected %#v\ngot      %#v", test.src, test.expected, got)
		}
		if src != test.unparsed {
			t.Fatalf("%q. expected unparsed string %q, got %q", test.src, test.unparsed, src)
		}
	}

}

func Test_parseNumber(t *testing.T) {

	tests := []struct {
		src      string
		expected string
		unparsed string
		err      error
	}{
		{`0`, `0`, ``, nil},
		{`682`, `682`, ``, nil},
		{`-4992`, `-4992`, ``, nil},
		{`0.`, `0`, ``, nil},
		{`.0`, `0`, ``, nil},
		{`.00`, `0`, ``, nil},
		{`-.0`, `0`, ``, nil},
		{`.652`, `0.652`, ``, nil},
		{`0.1`, `0.1`, ``, nil},
		{`9.0134`, `9.0134`, ``, nil},
		{`0e0`, `0`, ``, nil},
		{`551e3`, `551e3`, ``, nil},
		{`7e-2`, `7e-2`, ``, nil},
		{`13.5E012`, `13.5e12`, ``, nil},
		{`819.6520e3`, `819.652e3`, ``, nil},
		{`-7.0284710e-3`, `-7.028471e-3`, ``, nil},
		{`0 a`, `0`, ` a`, nil},
		{"207.35\t a", `207.35`, "\t a", nil},
		{"1\n\ta", `1`, "\n\ta", nil},
		{`0.02"a"`, `0.02`, `"a"`, nil},
		{`-3'a'`, `-3`, `'a'`, nil},
		{`5, `, `5`, `, `, nil},
		{`-3)`, `-3`, `)`, nil},
		{`3e`, ``, `3e`, errInvalidNumber},
		{`2.7name`, ``, `2.7name`, errInvalidNumber},
		{`1.x`, ``, `1.x`, errInvalidNumber},
		{`1e a`, ``, `1e a`, errInvalidNumber},
		{`1_000`, ``, `1_000`, errInvalidNumber},
		{`0x123`, ``, `0x123`, errInvalidNumber},
	}

	for _, test := range tests {
		got, src, err := parseNumber(test.src)
		if err != nil {
			if test.err == nil {
				t.Fatalf("%q. unexpected error: %s", test.src, err)
			}
			if err.Error() != test.err.Error() {
				t.Fatalf("%q. expected error %q, got error %q", test.src, test.err.Error(), err.Error())
			}
		} else if test.err != nil {
			t.Fatalf("%q. expected error %q, got no error", test.src, test.err)
		} else {
			if got.Cmp(decimal.MustParse(test.expected)) != 0 {
				t.Fatalf("%q. expected number %s, got %s", test.src, test.expected, got)
			}
		}
		if src != test.unparsed {
			t.Fatalf("%q. expected unparsed string %q, got %q", test.src, test.unparsed, src)
		}
	}

}

func Test_parsePath(t *testing.T) {

	tests := []struct {
		src      string
		expected path
		unparsed string
		err      error
	}{
		{`_`, path{[]string{`_`}, []decorators{0}}, ``, nil},
		{`a`, path{[]string{`a`}, []decorators{0}}, ``, nil},
		{`foo`, path{[]string{`foo`}, []decorators{0}}, ``, nil},
		{`_foo`, path{[]string{`_foo`}, []decorators{0}}, ``, nil},
		{`foo53`, path{[]string{`foo53`}, []decorators{0}}, ``, nil},
		{`_8`, path{[]string{`_8`}, []decorators{0}}, ``, nil},
		{`foo.boo`, path{[]string{`foo`, `boo`}, []decorators{0, 0}}, ``, nil},
		{`foo.boo foo`, path{[]string{`foo`, `boo`}, []decorators{0, 0}}, ` foo`, nil},
		{`_._`, path{[]string{`_`, `_`}, []decorators{0, 0}}, ``, nil},
		{`a$`, path{[]string{`a`}, []decorators{0}}, `$`, nil},
		{`a["k"]`, path{[]string{`a`, `k`}, []decorators{0, indexing}}, ``, nil},
		{`a["k"].b`, path{[]string{`a`, `k`, `b`}, []decorators{0, indexing, 0}}, ``, nil},
		{`a["x.y"].b`, path{[]string{`a`, `x.y`, `b`}, []decorators{0, indexing, 0}}, ``, nil},
		{`a["[x"]`, path{[]string{`a`, `[x`}, []decorators{0, indexing}}, ``, nil},
		{`a["x]"]`, path{[]string{`a`, `x]`}, []decorators{0, indexing}}, ``, nil},
		{`a["[x]"]`, path{[]string{`a`, `[x]`}, []decorators{0, indexing}}, ``, nil},
		{`a["x?"]`, path{[]string{`a`, `x?`}, []decorators{0, indexing}}, ``, nil},
		{`a["[x?"]`, path{[]string{`a`, `[x?`}, []decorators{0, indexing}}, ``, nil},
		{`a["x]?"]`, path{[]string{`a`, `x]?`}, []decorators{0, indexing}}, ``, nil},
		{`a[":x"]`, path{[]string{`a`, `:x`}, []decorators{0, indexing}}, ``, nil},
		{`a[":x?"]`, path{[]string{`a`, `:x?`}, []decorators{0, indexing}}, ``, nil},
		{`a[ "k"]['j' ]`, path{[]string{`a`, `k`, `j`}, []decorators{0, indexing, indexing}}, ``, nil},
		{`a['k']["j"].b`, path{[]string{`a`, `k`, `j`, `b`}, []decorators{0, indexing, indexing, 0}}, ``, nil},
		{`a.b["k"]`, path{[]string{`a`, `b`, `k`}, []decorators{0, 0, indexing}}, ``, nil},
		{`a.b?`, path{[]string{`a`, `b`}, []decorators{0, optional}}, ``, nil},
		{`a.b?.c`, path{[]string{`a`, `b`, `c`}, []decorators{0, optional, 0}}, ``, nil},
		{`a['b']?`, path{[]string{`a`, `b`}, []decorators{0, indexing | optional}}, ``, nil},
		{`a['b']?.c`, path{[]string{`a`, `b`, `c`}, []decorators{0, indexing | optional, 0}}, ``, nil},
		{`a['?']?`, path{[]string{`a`, `?`}, []decorators{0, indexing | optional}}, ``, nil},
		{`a['x?']?`, path{[]string{`a`, `x?`}, []decorators{0, indexing | optional}}, ``, nil},
		{`a.`, path{}, ``, errUnterminatedPath},
		{`a.b.`, path{}, ``, errUnterminatedPath},
		{`a..`, path{}, ``, errUnexpectedPeriod},
		{`a.b..`, path{}, ``, errUnexpectedPeriod},
		{`a.["k"]`, path{}, ``, errUnexpectedPeriod},
		{`a[k]`, path{}, ``, errNoStringMapKey},
		{`a["k]`, path{}, ``, errNoTerminatedString},
		{`a["k"`, path{}, ``, errUnterminatedPath},
		{`a['k')`, path{}, ``, errUnterminatedPath},
		{`a[]`, path{}, ``, errNoStringMapKey},
		{`a[  ]`, path{}, ``, errNoStringMapKey},
		{`a.?`, path{}, ``, errUnexpectedPeriod},
		{`a[?`, path{}, ``, errNoStringMapKey},
	}

	for _, test := range tests {
		got, src, err := parsePath(test.src)
		if err != nil {
			if test.err == nil {
				t.Fatalf("%q: unexpected error: %s", test.src, err)
			}
			if err.Error() != test.err.Error() {
				t.Fatalf("%q: expected error %q, got error %q", test.src, test.err.Error(), err.Error())
			}
			continue
		}
		if test.err != nil {
			t.Fatalf("%q: expected error %q, got no error", test.src, test.err)
		}
		if len(test.expected.elements) != len(got.elements) {
			t.Fatalf("%q: expected elements length %d, got %d", test.src, len(test.expected.elements), len(got.elements))
		}
		if len(test.expected.decorators) != len(got.decorators) {
			t.Fatalf("%q: expected decorators length %d, got %d", test.src, len(test.expected.decorators), len(got.decorators))
		}
		for i, expected := range test.expected.elements {
			if expected != got.elements[i] {
				t.Fatalf("%q[%d]: expected element %q, got %q", test.src, i, expected, got.elements[i])
			}
		}
		for i, expected := range test.expected.decorators {
			if expected != got.decorators[i] {
				t.Fatalf("%q[%d]: expected decorator %b, got %b", test.src, i, expected, got.decorators[i])
			}
		}
		if src != test.unparsed {
			t.Fatalf("%q: expected unparsed string %q, got %q", test.src, test.unparsed, src)
		}
	}

}

func Test_parsePredeclaredIdentifier(t *testing.T) {

	tests := []struct {
		src           string
		expectedValue any
		expectedType  types.Type
		unparsed      string
	}{
		{`true`, true, types.Boolean(), ``},
		{`false`, false, types.Boolean(), ``},
		{`null`, nil, types.JSON(), ``},
		{`null.`, nil, types.JSON(), `.`},
		{`true a.b`, true, types.Boolean(), ` a.b`},
		{`false"foo"`, false, types.Boolean(), `"foo"`},
		{`null()`, nil, types.JSON(), `()`},
		{`truevalue`, nil, types.Type{}, `truevalue`},
	}

	for _, test := range tests {
		got, typ, src := parsePredeclaredIdentifier(test.src)
		if got != test.expectedValue {
			t.Fatalf("%q. expected value %#v, got %#v", test.src, test.expectedValue, got)
		}
		if !types.Equal(typ, test.expectedType) {
			if typ.Valid() {
				t.Fatalf("%q. expected type %s, got %s", test.src, test.expectedType, typ)
			}
			t.Fatalf("%q. expected type %s, got invalid type", test.src, test.expectedType)
		}
		if src != test.unparsed {
			t.Fatalf("%q. expected unparsed string %q, got %q", test.src, test.unparsed, src)
		}
	}

}

func Test_parseString(t *testing.T) {

	tests := []struct {
		src      string
		expected string
		unparsed string
		err      error
	}{
		{`"`, ``, ``, errNoTerminatedString},
		{`'`, ``, ``, errNoTerminatedString},
		{`""`, ``, ``, nil},
		{`''`, ``, ``, nil},
		{`"a"`, `a`, ``, nil},
		{`'a'`, `a`, ``, nil},
		{`"hello world"`, `hello world`, ``, nil},
		{`"hello world`, ``, ``, errNoTerminatedString},
		{`"\a \b \f \n \r \t \v \\ \' \""`, "\a \b \f \n \r \t \v \\ ' \"", ``, nil},
		{`"\a`, ``, ``, errNoTerminatedString},
		{"\"\\t \x00\"", ``, ``, errZeroByteInString},
		{"\"\x00\"", ``, ``, errZeroByteInString},
		{`"\u0000"`, ``, ``, errZeroByteInString},
		{`"\u123`, ``, ``, errNoTerminatedString},
		{`"\u1234`, ``, ``, errNoTerminatedString},
		{`"\U00000000"`, ``, ``, errZeroByteInString},
		{`"\U1234567`, ``, ``, errNoTerminatedString},
		{`"hello" foo "word"`, `hello`, ` foo "word"`, nil},
		{`'hello' foo 'word'`, `hello`, ` foo 'word'`, nil},
		{`"à" ò`, `à`, ` ò`, nil},
	}

	for _, test := range tests {
		got, src, err := parseString(test.src)
		if err != nil {
			if test.err == nil {
				t.Fatalf("%q. unexpected error: %s", test.src, err)
			}
			if err.Error() != test.err.Error() {
				t.Fatalf("%q. expected error %q, got error %q", test.src, test.err.Error(), err.Error())
			}
			continue
		}
		if test.err != nil {
			t.Fatalf("%q. expected error %q, got no error", test.src, test.err)
		}
		if got != test.expected {
			t.Fatalf("%q. expected string %q, got %q", test.src, test.expected, got)
		}
		if src != test.unparsed {
			t.Fatalf("%q. expected unparsed string %q, got %q", test.src, test.unparsed, src)
		}
	}

}

// TestIntegerLiteralTypes checks the inferred types before contextual
// conversion can change them.
func TestIntegerLiteralTypes(t *testing.T) {

	tests := []struct {
		source string
		want   types.Type
	}{
		{"0", types.Int(32)},
		{"2147483647", types.Int(32)},
		{"-2147483648", types.Int(32)},
		{"2147483648", types.Int(64)},
		{"-2147483649", types.Int(64)},
		{"9223372036854775807", types.Int(64)},
		{"-9223372036854775808", types.Int(64)},
		{"9223372036854775808", types.Decimal(types.MaxDecimalPrecision, types.MaxDecimalScale)},
		{"-9223372036854775809", types.Decimal(types.MaxDecimalPrecision, types.MaxDecimalScale)},
		{"1.5", types.Decimal(types.MaxDecimalPrecision, types.MaxDecimalScale)},
	}

	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {

			parts, rest, err := parse(test.source, 0, len(test.source), 0)
			if err != nil {
				t.Fatal(err)
			}

			if len(parts) != 1 || rest != "" {
				t.Fatalf("got %d parts and remainder %q, want one complete literal", len(parts), rest)
			}
			if fmt.Sprint(parts[0].value) != test.source || !types.Equal(parts[0].typ, test.want) {
				t.Fatalf("got %v (%s), want %s (%s)", parts[0].value, parts[0].typ, test.source, test.want)
			}

		})
	}

}

// TestNullLiteralConcatenation checks that null contributes no characters,
// while a lone null remains nil.
func TestNullLiteralConcatenation(t *testing.T) {

	tests := []struct {
		source string
		want   any
	}{
		{`null`, nil},
		{`null 'x'`, "x"},
		{`'x' null`, "x"},
		{`null true`, "true"},
		{`null false`, "false"},
		{`null 42`, "42"},
		{`null 1.5`, "1.5"},
		{`null null`, ""},
		{`null null 'x'`, "x"},
		{`null 'x' null 'y'`, "xy"},
		{`null lower('X') 'y'`, "xy"},
	}

	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {

			expr, _, err := Compile(test.source, types.Type{}, types.String())
			if err != nil {
				t.Fatal(err)
			}

			got, typ, err := expr.Eval(nil)
			if err != nil {
				t.Fatal(err)
			}

			if got != test.want || !types.Equal(typ, types.String()) {
				t.Fatalf("got %#v (%s), want %#v (string)", got, typ, test.want)
			}

		})
	}

}

// TestCompileFunctionDepth checks the nesting boundary through compilation and
// evaluation.
func TestCompileFunctionDepth(t *testing.T) {
	for _, depth := range []int{0, 1, 49, 50, 51} {
		t.Run(fmt.Sprint(depth), func(t *testing.T) {

			source := strings.Repeat("not(", depth) + "value" + strings.Repeat(")", depth)
			schema := types.Object([]types.Property{{Name: "value", Type: types.Boolean()}})
			expr, _, err := Compile(source, schema, types.Boolean())
			if err != nil {
				if depth <= 50 {
					t.Fatal(err)
				}
				if want := "function calls cannot be nested more than 50 levels"; err.Error() != want {
					t.Fatalf("got %q, want %q", err, want)
				}
				return
			}
			if depth > 50 {
				t.Fatal("expected compilation to reject more than 50 nested calls")
			}

			got, typ, err := expr.Eval(map[string]any{"value": true})
			if err != nil {
				t.Fatal(err)
			}
			if want := depth%2 == 0; got != want || !types.Equal(typ, types.Boolean()) {
				t.Fatalf("got %v (%s), want %v (boolean)", got, typ, want)
			}

		})
	}
}

// TestCompileFunctionDepthShapes checks that only enclosing calls contribute to
// nesting depth.
func TestCompileFunctionDepthShapes(t *testing.T) {

	tests := []struct {
		name, source string
		typ          types.Type
		want         any
		tooDeep      bool
	}{
		{
			"zero-argument call at level 50", strings.Repeat("coalesce(", 49) + "array()" + strings.Repeat(")", 49),
			types.Array(types.JSON()), []any{}, false,
		},
		{
			"zero-argument call at level 51", strings.Repeat("coalesce(", 50) + "array()" + strings.Repeat(")", 50),
			types.Array(types.JSON()), nil, true,
		},
		{
			"sibling calls at level 50",
			strings.Repeat("coalesce(", 49) + "lower('X') upper('x')" + strings.Repeat(")", 49),
			types.String(), "xX", false,
		},
		{"many sibling calls", strings.Repeat("lower('X') ", 60), types.String(), strings.Repeat("x", 60), false},
		{"many arguments", "coalesce(" + strings.Repeat("null, ", 60) + "'x')", types.String(), "x", false},
		{
			"function syntax in literal", strings.Repeat("lower(", 50) + "'not((('" + strings.Repeat(")", 50),
			types.String(), "not(((", false,
		},
		{
			"unselected branch at level 50",
			"if(false, " + strings.Repeat("lower(", 49) + "'X'" + strings.Repeat(")", 49) + ", 'ok')",
			types.String(), "ok", false,
		},
		{
			"unselected branch at level 51",
			"if(false, " + strings.Repeat("lower(", 50) + "'X'" + strings.Repeat(")", 50) + ", 'ok')",
			types.String(), nil, true,
		},
		{
			"reject before parsing deeper arguments",
			strings.Repeat("coalesce(", 51) + "@" + strings.Repeat(")", 51), types.String(), nil, true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			expr, _, err := Compile(test.source, types.Type{}, test.typ)
			if err != nil {
				if !test.tooDeep {
					t.Fatal(err)
				}
				if want := "function calls cannot be nested more than 50 levels"; err.Error() != want {
					t.Fatalf("got %q, want %q", err, want)
				}
				return
			}
			if test.tooDeep {
				t.Fatal("expected compilation to reject more than 50 nested calls")
			}

			got, typ, err := expr.Eval(nil)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, test.want) || !types.Equal(typ, test.typ) {
				t.Fatalf("got %#v (%s), want %#v (%s)", got, typ, test.want, test.typ)
			}

		})
	}

}

// TestMappingFunctionDepth checks that mapping construction also enforces the
// nesting limit.
func TestMappingFunctionDepth(t *testing.T) {
	for _, depth := range []int{50, 51} {
		t.Run(fmt.Sprint(depth), func(t *testing.T) {

			source := strings.Repeat("not(", depth) + "true" + strings.Repeat(")", depth)
			outSchema := types.Object([]types.Property{{Name: "out", Type: types.Boolean()}})
			mapping, err := New(map[string]string{"out": source}, types.Type{}, outSchema, false, nil)
			if err != nil {
				if depth <= 50 {
					t.Fatal(err)
				}
				if want := "function calls cannot be nested more than 50 levels"; err.Error() != want {
					t.Fatalf("got %q, want %q", err, want)
				}
				return
			}
			if depth > 50 {
				t.Fatal("expected mapping construction to reject more than 50 nested calls")
			}

			got, err := mapping.Transform(nil, None)
			if err != nil {
				t.Fatal(err)
			}
			if got["out"] != true {
				t.Fatalf("got %#v, want out=true", got)
			}

		})
	}
}

// TestCompileLeadingDecimalPoint checks numeric literals and disambiguated
// property paths through Compile.
func TestCompileLeadingDecimalPoint(t *testing.T) {

	tests := []struct {
		source string
		want   float64
	}{
		{".5", 0.5}, {".25", 0.25}, {".5e2", 50}, {".5e-2", 0.005},
		{".0", 0}, {"0.5", 0.5}, {"-.5", -0.5}, {".value", 7}, {".true", 9},
	}
	schema := types.Object([]types.Property{
		{Name: "value", Type: types.Float(64)}, {Name: "true", Type: types.Float(64)},
	})
	for _, test := range tests {

		t.Run(test.source, func(t *testing.T) {

			expr, _, err := Compile(test.source, schema, types.Float(64))
			if err != nil {
				t.Fatal(err)
			}
			got, typ, err := expr.Eval(map[string]any{"value": 7.0, "true": 9.0})
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want || !types.Equal(typ, types.Float(64)) {
				t.Fatalf("got %v (%s), want %v (float(64))", got, typ, test.want)
			}

		})

	}

}

// TestCompileStringEscapes rejects unsupported escapes while preserving
// literal backslashes and supported escapes.
func TestCompileStringEscapes(t *testing.T) {

	tests := []struct {
		content, want, unknown string
	}{
		{`\q`, "", `\q`},
		{`prefix\q`, "", `\q`},
		{`\n\q`, "", `\q`},
		{`\u0041\q`, "", `\q`},
		{`\x41`, "", `\x`},
		{`\0`, "", `\0`},
		{`\é`, "", `\é`},
		{`\\q`, `\q`, ""},
		{`\'\"`, `'"`, ""},
		{`\a\b\f\n\r\t\v`, "\a\b\f\n\r\t\v", ""},
		{`\u0041\U0001F600`, "A😀", ""},
	}

	for _, quote := range []string{"'", `"`} {
		for _, test := range tests {

			source := quote + test.content + quote

			t.Run(fmt.Sprintf("%q", source), func(t *testing.T) {

				expr, _, err := Compile(source, types.Type{}, types.String())
				if err != nil {
					if test.unknown == "" {
						t.Fatal(err)
					}
					want := fmt.Sprintf("unknown escape sequence %q", test.unknown)
					if err.Error() != want {
						t.Fatalf("got %q, want %q", err, want)
					}
					return
				}
				if test.unknown != "" {
					t.Fatalf("accepted unsupported escape %q", test.unknown)
				}

				got, typ, err := expr.Eval(nil)
				if err != nil {
					t.Fatal(err)
				}
				if got != test.want || !types.Equal(typ, types.String()) {
					t.Fatalf("got %#v (%s), want %q (string)", got, typ, test.want)
				}

			})

		}
	}

}

// TestCompileStringNUL rejects NUL bytes regardless of their position relative
// to escapes.
func TestCompileStringNUL(t *testing.T) {

	contents := []string{
		"\x00x", "\x00x\\n", "\x00x\\u0041", "\x00x\\U00000041",
		"x\\n\x00", "x\\u0041\x00", "\\u0000", "\\U00000000",
		"x\\'\x00\\n", "x\\\"\x00\\n",
		"x\\'\\\x00", "x\\\"\\\x00",
	}
	for _, quote := range []string{"'", `"`} {

		for _, content := range contents {

			source := quote + content + quote
			t.Run(fmt.Sprintf("%q", source), func(t *testing.T) {

				for _, parser := range []string{"parseString", "Compile"} {

					t.Run(parser, func(t *testing.T) {

						var err error
						if parser == "parseString" {
							_, _, err = parseString(source)
						} else {
							_, _, err = Compile(source, types.Type{}, types.String())
						}

						if err != nil {
							if !errors.Is(err, errZeroByteInString) {
								t.Fatalf("got %v, want %v", err, errZeroByteInString)
							}
							return
						}
						t.Fatal("accepted a NUL byte")

					})

				}

			})

		}

	}

}

// TestCompileUnicodeEscapes checks escape boundaries and rejects incomplete or
// invalid code points without panic.
func TestCompileUnicodeEscapes(t *testing.T) {

	tests := []struct {
		source  string
		want    string
		invalid bool
	}{
		{`'\u0041'`, "A", false},
		{`'\u0041BC'`, "ABC", false},
		{`'x\u0041'`, "xA", false},
		{`'\u0041\u0042'`, "AB", false},
		{`"\u0041B"`, "AB", false},
		{`'\U0001F600x'`, "😀x", false},
		{`'\U0010FFFF'`, "\U0010FFFF", false},
		{`'\u0041' 'B'`, "AB", false},
		{`'A\nB'`, "A\nB", false},
		{`'\'\u0041`, "", true},
		{`'\'\U00000041`, "", true},
		{`'\u00`, "", true},
		{`'\u00x1'`, "", true},
		{`'\u0000'`, "", true},
		{`'\uD800'`, "", true},
		{`'\uDFFF'`, "", true},
		{`'\U00110000'`, "", true},
		{`'\U80000000'`, "", true},
		{`'\UFFFFFFFFx'`, "", true},
	}
	for _, test := range tests {

		t.Run(test.source, func(t *testing.T) {

			expr, _, err := Compile(test.source, types.Type{}, types.String())
			if err != nil {
				if !test.invalid {
					t.Fatal(err)
				}
				return
			}
			if test.invalid {
				t.Fatal("expected a compilation error")
			}
			got, typ, err := expr.Eval(nil)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want || !types.Equal(typ, types.String()) {
				t.Fatalf("got %q (%s), want %q (string)", got, typ, test.want)
			}

		})

	}

}
