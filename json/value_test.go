//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package json

import (
	"bytes"
	"math"
	"reflect"
	"strconv"
	"testing"

	"github.com/shopspring/decimal"
)

func Test_Value(t *testing.T) {

	t.Run("Bytes", func(t *testing.T) {
		tests := []struct {
			data     string
			expected []byte
		}{
			{`""`, []byte(``)},
			{`"a\"b"`, []byte(`a"b`)},
			{`null`, []byte(`null`)},
			{`true`, []byte(`true`)},
			{`false`, []byte(`false`)},
			{`5`, []byte(`5`)},
			{`{"b":3}`, nil},
			{`[1,2,3]`, nil},
		}
		for _, test := range tests {
			if b := Value(test.data).Bytes(); !bytes.Equal(test.expected, b) {
				t.Fatalf("expected %#v, got %#v", test.expected, b)
			}
		}
	})

	t.Run("Bool", func(t *testing.T) {
		if !Value(`true`).Bool() {
			t.Fatal("expected true, got false")
		}
		if !Value(` true`).Bool() {
			t.Fatal("expected true, got false")
		}
		if Value(`false`).Bool() {
			t.Fatal("expected false, got true")
		}
		if Value(`5`).Bool() {
			t.Fatal("expected false, got true")
		}
	})

	t.Run("Decimal", func(t *testing.T) {
		if n, err := Value(`102`).Decimal(); err != nil {
			t.Fatalf("expected 102, got error %q", err)
		} else if !n.Equal(decimal.RequireFromString("102")) {
			t.Fatalf("expected 102, got %q", n)
		}
		if n, err := Value("\t\r\n102  ").Decimal(); err != nil {
			t.Fatalf("expected 102, got error %q", err)
		} else if !n.Equal(decimal.RequireFromString("102")) {
			t.Fatalf("expected 102, got %q", n)
		}
		if n, err := Value(`-37.0281e3`).Decimal(); err != nil {
			t.Fatalf("expected -37028.1, got error %q", err)
		} else if !n.Equal(decimal.RequireFromString("-37028.1")) {
			t.Fatalf("expected -37028.1, got %q", n)
		}
		//if n, err := Value("1e2147483647").Decimal(); err == nil {
		//	t.Fatalf("expected error, got Decimal %q", n)
		//}
		if n, err := Value(`true`).Decimal(); err == nil {
			t.Fatalf("expected error, got %q and no error", n)
		}
	})

	t.Run("Elements", func(t *testing.T) {
		count := 0
		arr := []Value{Value(`5`), Value(`"a"`), Value(`true`), Value(`"b"`)}
		for i, got := range Value(`[5, "a", true, "b"]`).Elements() {
			if i != count {
				t.Fatalf("expected index %d, got %d", count, i)
			}
			if i == len(arr) {
				t.Fatalf("expected %d elements, got %d", len(arr), i)
			}
			if !bytes.Equal(arr[i], got) {
				t.Fatalf("expected value %q, got %q", string(arr[i]), string(got))
			}
			count++
		}
		if count < len(arr) {
			t.Fatalf("expected %d iterations, done %d", len(arr), count)
		}
	})

	t.Run("Float", func(t *testing.T) {
		if n, err := Value(`2.55`).Float(32); err != nil {
			t.Fatalf("expected 2.55, got error %q", err)
		} else if n != float64(float32(2.55)) {
			t.Fatalf("expected 2.55, got %f", n)
		}
		if n, err := Value("\n 2.55\t").Float(32); err != nil {
			t.Fatalf("expected 2.55, got error %q", err)
		} else if n != float64(float32(2.55)) {
			t.Fatalf("expected 2.55, got %f", n)
		}
		if n, err := Value(`80371.75592013886`).Float(64); err != nil {
			t.Fatalf("expected 80371.75592013886, got error %q", err)
		} else if n != 80371.75592013886 {
			t.Fatalf("expected 80371.75592013886, got %f", n)
		}
		if n, err := Value(`{}`).Float(64); err == nil {
			t.Fatalf("expected error, got %f and no error", n)
		}
	})

	t.Run("Int", func(t *testing.T) {
		if n, err := Value(`-55`).Int(); err != nil {
			t.Fatalf("expected -55, got error %q", err)
		} else if n != -55 {
			t.Fatalf("expected -55, got %q", n)
		}
		if n, err := Value(strconv.FormatInt(math.MaxInt64, 10)).Int(); err != nil {
			t.Fatalf("expected %d, got error %q", math.MaxInt64, err)
		} else if n != math.MaxInt64 {
			t.Fatalf("expected %d, got %q", math.MaxInt64, n)
		}
		if n, err := Value(`-3.0`).Int(); err != nil {
			t.Fatalf("expected -3, got error %q", err)
		} else if n != -3 {
			t.Fatalf("expected -3, got %q", n)
		}
		if n, err := Value(`3.45`).Int(); err == nil {
			t.Fatalf("expected error, got %d and no error", n)
		}
	})

	t.Run("IsArray", func(t *testing.T) {
		if !Value(`["a",5]`).IsArray() {
			t.Fatal("expected true, got false")
		}
		if Value(`{"a":5}`).IsArray() {
			t.Fatal("expected false, got true")
		}
	})

	t.Run("IsFalse", func(t *testing.T) {
		if !Value(`false`).IsFalse() {
			t.Fatal("expected true, got false")
		}
		if Value(`{}`).IsFalse() {
			t.Fatal("expected false, got true")
		}
	})

	t.Run("IsNull", func(t *testing.T) {
		if !Value(`null`).IsNull() {
			t.Fatal("expected true, got false")
		}
		if Value(`true`).IsNull() {
			t.Fatal("expected false, got true")
		}
	})

	t.Run("IsNumber", func(t *testing.T) {
		if !Value(`537`).IsNumber() {
			t.Fatal("expected true, got false")
		}
		if !Value(`-2.67e5`).IsNumber() {
			t.Fatal("expected true, got false")
		}
		if !Value(`0`).IsNumber() {
			t.Fatal("expected true, got false")
		}
		if Value(`[]`).IsNumber() {
			t.Fatal("expected false, got true")
		}
	})

	t.Run("IsObject", func(t *testing.T) {
		if !Value(`{"a":5}`).IsObject() {
			t.Fatal("expected true, got false")
		}
		if Value(`["a",5]`).IsObject() {
			t.Fatal("expected false, got true")
		}
	})

	t.Run("IsString", func(t *testing.T) {
		if !Value(`"abc"`).IsString() {
			t.Fatal("expected true, got false")
		}
		if Value(`true`).IsString() {
			t.Fatal("expected false, got true")
		}
	})

	t.Run("IsTrue", func(t *testing.T) {
		if !Value(`true`).IsTrue() {
			t.Fatal("expected true, got false")
		}
		if Value(`5`).IsTrue() {
			t.Fatal("expected false, got true")
		}
	})

	t.Run("Kind", func(t *testing.T) {
		tests := []struct {
			data string
			kind Kind
		}{
			{`null`, Null},
			{`true`, True},
			{`false`, False},
			{`"abc"`, String},
			{`3.45`, Number},
			{`-12`, Number},
			{`0.5e7`, Number},
			{`{"a":1}`, Object},
			{`{"a":true}`, Object},
			{`["a","b"]`, Array},
			{` null`, Null},
			{"\ntrue", True},
			{`   false`, False},
			{"\t\t\"abc\"", String},
			{` 3.45`, Number},
			{"\r\n \t-12", Number},
			{` 0.5e7`, Number},
			{` { "a": 1 }`, Object},
			{"\n[\"a\",\"b\"]", Array},
		}
		for _, test := range tests {
			if got := Value(test.data).Kind(); test.kind != got {
				t.Fatalf("expected %s, got %s", test.kind, got)
			}
		}
	})

	t.Run("Lookup", func(t *testing.T) {
		var v Value
		var rec any
		func() {
			defer func() {
				rec = recover()
			}()
			v, _ = Value(`null`).Lookup("a")
		}()
		if rec == nil {
			t.Fatalf("expected panic, got value %q", string(v))
		}
		if v, ok := Value(`{"b":7`).Lookup("a"); v != nil || ok {
			t.Fatalf("expected (nil, false), got (%q, %t)", string(v), ok)
		}
		if v, ok := Value(`{"":7`).Lookup(""); !bytes.Equal(v, []byte(`7`)) || !ok {
			t.Fatalf("expected (\"7\", true), got (%q, %t)", string(v), ok)
		}
		if v, ok := Value(`{"a":1,"a":2}`).Lookup("a"); !bytes.Equal(v, []byte(`1`)) || !ok {
			t.Fatalf("expected (\"1\", true), got (%q, %t)", string(v), ok)
		}
		if v, ok := Value(`{"a":1,"b":{"c":false}}`).Lookup("c"); v != nil || ok {
			t.Fatalf("expected (nil, false), got (%q, %t)", string(v), ok)
		}
		if v, ok := Value(`{"a":1,"b":{"c":true}}`).Lookup("b.c"); !bytes.Equal(v, []byte(`true`)) || !ok {
			t.Fatalf("expected (\"true\", true), got (%q, %t)", string(v), ok)
		}
	})

	t.Run("MarshalJSON", func(t *testing.T) {
		if b, err := Value(`null`).MarshalJSON(); err != nil {
			t.Fatalf("expected \"null\", got error %q", err)
		} else if !bytes.Equal(b, []byte(`null`)) {
			t.Fatalf("expected \"null\", got %q", string(b))
		}
		if b, err := Value(`[1,2,3]`).MarshalJSON(); err != nil {
			t.Fatalf("expected \"[1,2,3]\", got error %q", err)
		} else if !bytes.Equal(b, []byte(`[1,2,3]`)) {
			t.Fatalf("expected \"[1,2,3]\", got %q", string(b))
		}
	})

	t.Run("Properties", func(t *testing.T) {
		properties := []struct {
			K string
			V Value
		}{
			{`a`, Value(`5`)},
			{``, Value(`true`)},
			{`"b"`, Value(`{"c":5}`)},
			{`d`, Value(`null`)},
		}
		i := 0
		for gotK, gotV := range Value(`{"a":5,"":true,"\"b\"":{"c":5},"d":null`).Properties() {
			if i == len(properties) {
				t.Fatalf("expected %d keys, got %d", len(properties), i)
			}
			kv := properties[i]
			if kv.K != gotK {
				t.Fatalf("expected key %q, got %q", kv.K, gotK)
			}
			if !bytes.Equal(kv.V, gotV) {
				t.Fatalf("expected value %q, got %q", string(kv.V), string(gotV))
			}
			i++
		}
		if i < len(properties) {
			t.Fatalf("expected %d iterations, done %d", len(properties), i)
		}
	})

	t.Run("String", func(t *testing.T) {
		tests := []struct {
			data     string
			expected string
		}{
			{`""`, ``},
			{` "" `, ``},
			{`"a\"b"`, `a"b`},
			{`null`, `null`},
			{`true`, `true`},
			{`false`, `false`},
			{`5`, `5`},
			{"\n5", `5`},
			{`{"b":3}`, ``},
			{`[1,2,3]`, ``},
		}
		for _, test := range tests {
			if s := Value(test.data).String(); test.expected != s {
				t.Fatalf("expected %q, got %q", test.expected, s)
			}
		}
	})

	t.Run("Uint", func(t *testing.T) {
		if n, err := Value(`816`).Uint(); err != nil {
			t.Fatalf("expected 816, got error %q", err)
		} else if n != 816 {
			t.Fatalf("expected 816, got %q", n)
		}
		if n, err := Value(strconv.FormatUint(math.MaxUint64, 10)).Uint(); err != nil {
			t.Fatalf("expected %d, got error %q", uint64(math.MaxUint64), err)
		} else if n != math.MaxUint64 {
			t.Fatalf("expected %d, got %q", uint64(math.MaxUint64), n)
		}
		if n, err := Value(`3.0`).Uint(); err != nil {
			t.Fatalf("expected 3, got error %q", err)
		} else if n != 3 {
			t.Fatalf("expected 3, got %q", n)
		}
		if n, err := Value(`8.2`).Uint(); err == nil {
			t.Fatalf("expected error, got %d and no error", n)
		}
	})

	t.Run("Compact", func(t *testing.T) {
		tests := []struct {
			data     string
			expected string
			err      error
		}{
			{`null`, `null`, nil},
			{"\n\t    { \"foo\": true, \"boo\" : [ 1,2, 3 ]}\n ", `{"foo":true,"boo":[1,2,3]}`, nil},
			{`["a","b","c"]`, `["a","b","c"]`, nil},
			{` ["a","b","c"`, ``, ErrInvalidJSON},
			{" [\"\xFF\"] ", ``, ErrInvalidJSON},
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
	})

	t.Run("StripZeroBytes", func(t *testing.T) {
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
	})

	t.Run("TrimSpace", func(t *testing.T) {
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
	})

	t.Run("Valid", func(t *testing.T) {
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
	})

	t.Run("UnmarshalJSON", func(t *testing.T) {
		tests := []struct {
			data     string
			expected string
			err      error
		}{
			{`""`, `""`, nil},
			{` "" `, `""`, nil},
			{`"foo"`, `"foo"`, nil},
			{`"foo`, ``, ErrInvalidJSON},
			{`{"foo": true, "boo": 5 } `, `{"foo": true, "boo": 5 }`, nil},
			{"\"\xFF\"", ``, ErrInvalidJSON},
		}
		for _, test := range tests {
			v := Value("true")
			err := v.UnmarshalJSON([]byte(test.data))
			if !reflect.DeepEqual(err, test.err) {
				t.Fatalf("expected error %v (type %T), got error %v (type %T)", test.err, test.err, err, err)
			}
			if test.err != nil {
				if !bytes.Equal(v, []byte("true")) {
					t.Fatalf("expected unchanged v, got `%s`", string(v))
				}
				continue
			}
			if got := string(v); test.expected != got {
				t.Fatalf("expected `%s`, got `%s`", test.expected, got)
			}
		}
	})

	t.Run("Valid", func(t *testing.T) {
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
	})

}
