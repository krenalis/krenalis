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

	"github.com/meergo/meergo/core/decimal"
	"github.com/meergo/meergo/types"
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
		{`"Page View"`, []part{{value: `Page View`, typ: types.Text(), end: 11}}, ``, nil},
		{` 'Page View' `, []part{{value: `Page View`, typ: types.Text(), end: 13}}, ``, nil},
		{`51`, []part{{value: 51, typ: types.Int(32), end: 2}}, ``, nil},
		{`-6.803`, []part{{value: n, typ: dt, end: 6}}, ``, nil},
		{`true`, []part{{value: true, typ: types.Boolean(), end: 4}}, ``, nil},
		{`false`, []part{{value: false, typ: types.Boolean(), end: 5}}, ``, nil},
		{`null`, []part{{value: nil, typ: types.JSON(), end: 4}}, ``, nil},
		{`name`, []part{{path: path{elements: []string{`name`}, decorators: []decorators{0}}, end: 4}}, ``, nil},
		{`.name`, []part{{path: path{elements: []string{`name`}, decorators: []decorators{0}}, end: 5}}, ``, nil},
		{`context.os.version`, []part{{path: path{elements: []string{`context`, `os`, `version`}, decorators: []decorators{0, 0, 0}}, end: 18}}, ``, nil},
		{`.context.os.version`, []part{{path: path{elements: []string{`context`, `os`, `version`}, decorators: []decorators{0, 0, 0}}, end: 19}}, ``, nil},
		{`"Page " name`, []part{{value: `Page `, path: path{elements: []string{`name`}, decorators: []decorators{0}}, typ: types.Text(), end: 12}}, ``, nil},
		{`"OS " context.os.name " (" context.os.version ")"`, []part{
			{value: `OS `, path: path{elements: []string{`context`, `os`, `name`}, decorators: []decorators{0, 0, 0}}, typ: types.Text(), end: 22},
			{value: ` (`, path: path{elements: []string{`context`, `os`, `version`}, decorators: []decorators{0, 0, 0}}, typ: types.Text(), start: 22, end: 46},
			{value: `)`, typ: types.Text(), start: 46, end: 49},
		}, ``, nil},
		{`coalesce(event, 'Page ' true)`, []part{
			{path: path{elements: []string{`coalesce`}, decorators: []decorators{0}}, args: [][]part{
				{{path: path{elements: []string{`event`}, decorators: []decorators{0}}, start: 9, end: 14}},
				{{value: `Page true`, typ: types.Text(), start: 16, end: 28}},
			}, end: 29},
		}, ``, nil},
		{`"" event`, []part{{value: ``, path: path{elements: []string{`event`}, decorators: []decorators{0}}, typ: types.Text(), end: 8}}, ``, nil},
		{`coalesce(a)`, []part{{path: path{elements: []string{`coalesce`}, decorators: []decorators{0}}, args: [][]part{{{path: path{elements: []string{`a`}, decorators: []decorators{0}}, start: 9, end: 10}}}, end: 11}}, ``, nil},
		{`coalesce(a, 'b')`, []part{{path: path{elements: []string{`coalesce`}, decorators: []decorators{0}}, args: [][]part{{{path: path{elements: []string{`a`}, decorators: []decorators{0}}, start: 9, end: 10}}, {{value: `b`, typ: types.Text(), start: 12, end: 15}}}, end: 16}}, ``, nil},
		{`coalesce(5, 'a', coalesce(b))`, []part{{path: path{elements: []string{`coalesce`}, decorators: []decorators{0}}, args: [][]part{
			{{value: 5, typ: types.Int(32), start: 9, end: 10}}, {{value: `a`, typ: types.Text(), start: 12, end: 15}}, {{path: path{elements: []string{`coalesce`}, decorators: []decorators{0}}, args: [][]part{{{path: path{elements: []string{`b`}, decorators: []decorators{0}}, start: 26, end: 27}}}, start: 17, end: 28}},
		}, end: 29}}, ``, nil},
		{`coalesce("a" coalesce(b, 5) c)`, []part{{path: path{elements: []string{`coalesce`}, decorators: []decorators{0}}, args: [][]part{
			{{value: `a`, path: path{elements: []string{`coalesce`}, decorators: []decorators{0}}, args: [][]part{{{path: path{elements: []string{`b`}, decorators: []decorators{0}}, start: 22, end: 23}}, {{value: 5, typ: types.Int(32), start: 25, end: 26}}}, typ: types.Text(), start: 9, end: 27}, {path: path{elements: []string{`c`}, decorators: []decorators{0}}, start: 27, end: 29}},
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
		got, src, err := parse(test.src, 0, len(test.src))
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
