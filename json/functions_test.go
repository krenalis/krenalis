//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package json

import (
	"bytes"
	"reflect"
	"testing"
)

func Test_Compaq(t *testing.T) {
	tests := []struct {
		data     string
		expected string
		err      error
	}{
		{`null`, `null`, nil},
		{"\n\t    { \"foo\": true, \"boo\" : [ 1,2, 3 ]}\n ", `{"foo":true,"boo":[1,2,3]}`, nil},
		{`["a","b","c"]`, `["a","b","c"]`, nil},
		{` ["a","b","c"`, ``, ErrInvalidJSON},
		{"\"\xFF\"", ``, ErrInvalidJSON},
		{"\"\\xFF\"", ``, ErrInvalidJSON},
	}
	for _, test := range tests {
		got, err := Compact([]byte(test.data))
		if !reflect.DeepEqual(test.err, err) {
			t.Fatalf("expected error %v (type %T), got %v (type %T)", test.err, test.err, err, err)
		}
		if test.expected != string(got) {
			t.Fatalf("expected `%s`, got `%s`", test.expected, string(got))
		}
	}
}

func Test_Indent(t *testing.T) {
	tests := []struct {
		data     string
		expected string
		err      error
	}{
		{`null`, `null`, nil},
		{" 56.23\t", `56.23`, nil},
		{"\n\t    { \"foo\": true, \"boo\" : [ 1,2, 3 ]}\n ", "{\n \t\"boo\": [\n \t\t1,\n \t\t2,\n \t\t3\n \t],\n \t\"foo\": true\n }", nil},
		{" [ \"a\", \"b\" ]", "[\n \t\"a\",\n \t\"b\"\n ]", nil},
		{"", "", ErrInvalidJSON},
		{"\"\xFF\"", "", ErrInvalidJSON},
		{"\"\\xFF\"", "", ErrInvalidJSON},
	}
	for _, test := range tests {
		got, err := Indent([]byte(test.data), " ", "\t")
		if !reflect.DeepEqual(test.err, err) {
			t.Fatalf("expected error %v (type %T), got %v (type %T)", test.err, test.err, err, err)
		}
		if test.expected != string(got) {
			t.Fatalf("unexpected value.\nexpected: %q\ngot:      %q\n", test.expected, got)
		}
	}
}

func Test_Quote(t *testing.T) {
	tests := []struct {
		s        string
		expected string
		err      error
	}{
		{``, `""`, nil},
		{`foo`, `"foo"`, nil},
		{"\x00", `"\u0000"`, nil},
		{`"foo boo"`, `"\"foo boo\""`, nil},
		{"\xFF", ``, ErrInvalidUTF8},
	}
	for _, test := range tests {
		got, err := Quote([]byte(test.s))
		if !reflect.DeepEqual(test.err, err) {
			t.Fatalf("expected error %v (type %T), got %v (type %T)", test.err, test.err, err, err)
		}
		if test.expected != string(got) {
			t.Fatalf("unexpected value.\nexpected: %q\ngot:      %q\n", test.expected, got)
		}
	}
}

func Test_StripZeroBytes(t *testing.T) {
	tests := []struct {
		s        string
		expected string
	}{
		{`5`, `5`},
		{`""`, `""`},
		{`"abc"`, `"abc"`},
		{`"hello\u0020world"`, `"hello\u0020world"`},
		{`"\u0000"`, `""`},
		{`"hello\u0000world"`, `"helloworld"`},
		{`"hello\\u0000world"`, `"hello\\u0000world"`},
		{`"hello\\\u0000world"`, `"hello\\world"`},
		{`"hello\\\\u0000world"`, `"hello\\\\u0000world"`},
		{`"hello\\\\\u0000world"`, `"hello\\\\world"`},
		{`"hello\n\u0000world"`, `"hello\nworld"`},
		{`"\u0000world"`, `"world"`},
		{`"hello\u0000"`, `"hello"`},
		{`"hello\u0000world \u0000 hello \\u0000 world"`, `"helloworld  hello \\u0000 world"`},
		{`{"hello":1,"hello\u0000":2}`, `{"hello":1,"hello":2}`},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := StripZeroBytes([]byte(test.s))
			if !bytes.Equal([]byte(test.expected), got) {
				t.Fatalf("expected %q, got %q", test.expected, string(got))
			}
		})
	}
}

func Test_TrimSpace(t *testing.T) {
	tests := []struct {
		data     string
		expected string
	}{
		{`""`, `""`},
		{` "" `, `""`},
		{`null`, `null`},
		{"\ntrue\n\r\t", `true`},
		{`{"foo": 5} `, `{"foo": 5}`},
		{`[1,2,3]`, `[1,2,3]`},
		{` [1,2,3]`, `[1,2,3]`},
	}
	for _, test := range tests {
		got := TrimSpace([]byte(test.data))
		if test.expected != string(got) {
			t.Fatalf("expected `%s`, got `%s`", test.expected, got)
		}
	}
}

func Test_Valid(t *testing.T) {
	tests := []struct {
		data     string
		expected bool
	}{
		{``, false},
		{`null`, true},
		{`{1,2}`, false},
		{"{\"\xFF\":5}", false},
		{`{"à":true}`, true},
		{`False`, false},
		{"\n { \"foo\": \"boo\" } ", true},
		{` { "foo": "boo" }, { }`, false},
	}
	for _, test := range tests {
		if got := Valid([]byte(test.data)); test.expected != got {
			t.Fatalf("expected %t, got %t", test.expected, got)
		}
	}
}

func Test_Validate(t *testing.T) {
	tests := []struct {
		data   string
		err    string
		offset int64
	}{
		{data: ``, err: "content is empty", offset: 0},
		{data: `   `, err: "content is empty", offset: 0},
		{data: `{"a":@}`, err: "invalid character '@' at start of value", offset: 5},
		{data: `[1,2,3]4`, err: "invalid token '4' after top-level value", offset: 7},
		{data: `true false`, err: "invalid token 'false' after top-level value", offset: 4},
		{data: "\"\xFF\"", err: "invalid UTF-8", offset: 1},
		{data: ` { "foo": "boo" } `},
	}
	for _, test := range tests {
		err := Validate([]byte(test.data))
		if err == nil {
			if test.err != "" {
				t.Fatalf("expected error %q, got no error", test.err)
			}
			continue
		}
		if test.err == "" {
			t.Fatalf("expected no error, got %q (%T)", err, err)
		}
		err2, ok := err.(*SyntaxError)
		if !ok {
			t.Fatalf("expected *SyntaxError error, got %q (%T)", err, err)
		}
		if test.err != err.Error() {
			t.Fatalf("expected error %q, got error %q", test.err, err)
		}
		if got := err2.ByteOffset(); test.offset != got {
			t.Fatalf("expected offset %d, got %d", test.offset, got)
		}
	}
}
