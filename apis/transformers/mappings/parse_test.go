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

	"github.com/meergo/meergo/types"

	"github.com/shopspring/decimal"
)

func Test_parseExpression(t *testing.T) {

	n := decimal.RequireFromString(`-6.803`)
	dt := types.Decimal(types.MaxDecimalPrecision, types.MaxDecimalScale)

	tests := []struct {
		src      string
		expected []part
		unparsed string
		err      error
	}{
		{`"Page View"`, []part{{value: `Page View`, typ: types.Text()}}, ``, nil},
		{` 'Page View' `, []part{{value: `Page View`, typ: types.Text()}}, ``, nil},
		{`51`, []part{{value: 51, typ: types.Int(32)}}, ``, nil},
		{`-6.803`, []part{{value: n, typ: dt}}, ``, nil},
		{`true`, []part{{value: true, typ: types.Boolean()}}, ``, nil},
		{`false`, []part{{value: false, typ: types.Boolean()}}, ``, nil},
		{`null`, []part{{value: nil, typ: types.JSON()}}, ``, nil},
		{`name`, []part{{path: []string{`name`}}}, ``, nil},
		{`.name`, []part{{path: path{`name`}}}, ``, nil},
		{`context.os.version`, []part{{path: path{`context`, `os`, `version`}}}, ``, nil},
		{`.context.os.version`, []part{{path: path{`context`, `os`, `version`}}}, ``, nil},
		{`"Page " name`, []part{{value: `Page `, path: path{`name`}, typ: types.Text()}}, ``, nil},
		{`"OS " context.os.name " (" context.os.version ")"`, []part{
			{value: `OS `, path: path{`context`, `os`, `name`}, typ: types.Text()},
			{value: ` (`, path: path{`context`, `os`, `version`}, typ: types.Text()},
			{value: `)`, typ: types.Text()},
		}, ``, nil},
		{`coalesce(event, 'Page ' true)`, []part{
			{path: path{`coalesce`}, args: [][]part{
				{{path: path{`event`}}},
				{{value: `Page true`, typ: types.Text()}},
			}},
		}, ``, nil},
		{`"" event`, []part{{value: ``, path: path{"event"}, typ: types.Text()}}, ``, nil},
		{`coalesce(a)`, []part{{path: path{`coalesce`}, args: [][]part{{{path: path{`a`}}}}}}, ``, nil},
		{`coalesce(a, 'b')`, []part{{path: path{`coalesce`}, args: [][]part{{{path: path{`a`}}}, {{value: `b`, typ: types.Text()}}}}}, ``, nil},
		{`coalesce(5, 'a', coalesce(b))`, []part{{path: path{`coalesce`}, args: [][]part{
			{{value: 5, typ: types.Int(32)}}, {{value: `a`, typ: types.Text()}}, {{path: path{`coalesce`}, args: [][]part{{{path: path{`b`}}}}}},
		}}}, ``, nil},
		{`coalesce("a" coalesce(b, 5) c)`, []part{{path: path{`coalesce`}, args: [][]part{
			{{value: `a`, path: path{`coalesce`}, args: [][]part{{{path: path{`b`}}}, {{value: 5, typ: types.Int(32)}}}, typ: types.Text()}, {path: path{`c`}}},
		}}}, ``, nil},
		{`coalesce( coalesce ( x , 5 ) )`, []part{{path: path{`coalesce`}, args: [][]part{
			{{path: path{`coalesce`}, args: [][]part{{{path: path{`x`}}}, {{value: 5, typ: types.Int(32)}}}}},
		}}}, ``, nil},
		{`coalesce( , )`, nil, ``, errors.New("unexpected ',', expecting argument")},
		{`coalesce(a, )`, nil, ``, errors.New("unexpected ), expecting argument")},
		{`coalesce( @`, nil, ``, errors.New("unexpected '@', expecting argument")},
	}

	for _, test := range tests {
		got, src, err := parse(test.src)
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
		{`000`, `0`, ``, nil},
		{`682`, `682`, ``, nil},
		{`-4992`, `-4992`, ``, nil},
		{`0554`, `554`, ``, nil},
		{`0.`, `0`, ``, nil},
		{`00.`, `0`, ``, nil},
		{`.0`, `0`, ``, nil},
		{`.00`, `0`, ``, nil},
		{`-.0`, `0`, ``, nil},
		{`.652`, `0.652`, ``, nil},
		{`0.1`, `0.1`, ``, nil},
		{`00.14`, `00.14`, ``, nil},
		{`9.0134`, `9.0134`, ``, nil},
		{`0622.9350`, `622.935`, ``, nil},
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
		{`3e`, ``, ``, errInvalidNumber},
		{`2.7name`, ``, ``, errInvalidNumber},
		{`1.x`, ``, ``, errInvalidNumber},
		{`1e a`, ``, ``, errInvalidNumber},
		{`1_000`, ``, ``, errInvalidNumber},
		{`0x123`, ``, ``, errInvalidNumber},
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
			continue
		}
		if test.err != nil {
			t.Fatalf("%q. expected error %q, got no error", test.src, test.err)
		}
		if got.Cmp(decimal.RequireFromString(test.expected)) != 0 {
			t.Fatalf("%q. expected number %s, got %s", test.src, test.expected, got)
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
		{`_`, path{`_`}, ``, nil},
		{`a`, path{`a`}, ``, nil},
		{`foo`, path{`foo`}, ``, nil},
		{`_foo`, path{`_foo`}, ``, nil},
		{`foo53`, path{`foo53`}, ``, nil},
		{`_8`, path{`_8`}, ``, nil},
		{`foo.boo`, path{`foo`, `boo`}, ``, nil},
		{`foo.boo foo`, path{`foo`, `boo`}, ` foo`, nil},
		{`_._`, path{`_`, `_`}, ``, nil},
		{`a$`, path{`a`}, `$`, nil},
		{`a["k"]`, path{`a`, `[k]`}, ``, nil},
		{`a["k"].b`, path{`a`, `[k]`, `b`}, ``, nil},
		{`a["x.y"].b`, path{`a`, `[x.y]`, `b`}, ``, nil},
		{`a["[x"]`, path{`a`, `[[x]`}, ``, nil},
		{`a["x]"]`, path{`a`, `[x]]`}, ``, nil},
		{`a["[x]"]`, path{`a`, `[[x]]`}, ``, nil},
		{`a["x?"]`, path{`a`, `[x?]`}, ``, nil},
		{`a["[x?"]`, path{`a`, `[[x?]`}, ``, nil},
		{`a["x]?"]`, path{`a`, `[x]?]`}, ``, nil},
		{`a[":x"]`, path{`a`, `[:x]`}, ``, nil},
		{`a[":x?"]`, path{`a`, `[:x?]`}, ``, nil},
		{`a[ "k"]['j' ]`, path{`a`, `[k]`, `[j]`}, ``, nil},
		{`a['k']["j"].b`, path{`a`, `[k]`, `[j]`, `b`}, ``, nil},
		{`a.b["k"]`, path{`a`, `b`, `[k]`}, ``, nil},
		{`a.b?`, path{`a`, `b?`}, ``, nil},
		{`a.b?.c`, path{`a`, `b?`, `c`}, ``, nil},
		{`a['b']?`, path{`a`, `[b]?`}, ``, nil},
		{`a['b']?.c`, path{`a`, `[b]?`, `c`}, ``, nil},
		{`a['b']?`, path{`a`, `[b]?`}, ``, nil},
		{`a['?']?`, path{`a`, `[?]?`}, ``, nil},
		{`a['x?']?`, path{`a`, `[x?]?`}, ``, nil},
		{`a.`, nil, ``, errUnterminatedPath},
		{`a.b.`, nil, ``, errUnterminatedPath},
		{`a..`, nil, ``, errUnexpectedPeriod},
		{`a.b..`, nil, ``, errUnexpectedPeriod},
		{`a.["k"]`, nil, ``, errUnexpectedPeriod},
		{`a[k]`, nil, ``, errNoStringMapKey},
		{`a["k]`, nil, ``, errNoTerminatedString},
		{`a["k"`, nil, ``, errUnterminatedPath},
		{`a['k')`, nil, ``, errUnterminatedPath},
		{`a[]`, nil, ``, errNoStringMapKey},
		{`a[  ]`, nil, ``, errNoStringMapKey},
		{`a.?`, nil, ``, errUnexpectedPeriod},
		{`a[?`, nil, ``, errNoStringMapKey},
	}

	for _, test := range tests {
		got, src, err := parsePath(test.src)
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
			t.Fatalf("%s. expected error %q, got no error", test.src, test.err)
		}
		if len(got) != len(test.expected) {
			t.Fatalf("%s. expected path length %d, got %d", test.src, len(test.expected), len(got))
		}
		for i, expected := range test.expected {
			if got[i] != expected {
				t.Fatalf("%s. expected path component %q, got %q", test.src, expected, got[i])
			}
		}
		if src != test.unparsed {
			t.Fatalf("%s. expected unparsed string %q, got %q", test.src, test.unparsed, src)
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
