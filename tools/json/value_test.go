// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package json

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"testing"

	"github.com/krenalis/krenalis/tools/decimal"
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
		if n, err := Value(`102`).Decimal(3, 0); err != nil {
			t.Fatalf("expected 102, got error %q", err)
		} else if !n.Equal(decimal.MustParse("102")) {
			t.Fatalf("expected 102, got %q", n)
		}
		if n, err := Value("\t\r\n102  ").Decimal(3, 0); err != nil {
			t.Fatalf("expected 102, got error %q", err)
		} else if !n.Equal(decimal.MustParse("102")) {
			t.Fatalf("expected 102, got %q", n)
		}
		if n, err := Value(`-37.0281e3`).Decimal(6, 1); err != nil {
			t.Fatalf("expected -37028.1, got error %q", err)
		} else if !n.Equal(decimal.MustParse("-37028.1")) {
			t.Fatalf("expected -37028.1, got %q", n)
		}
		if n, err := Value(`true`).Decimal(1, 0); err == nil {
			t.Fatalf("expected error, got %q and no error", n)
		}
		if n, err := Value(`12.8401`).Decimal(3, 2); err == nil {
			t.Fatalf("expected error, got %q and no error", n)
		} else if err != ErrRange {
			t.Fatalf("expected error ErrRange, got error %#v", n)
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
		if n, err := Value(`1.8e55`).Float(32); err == nil {
			t.Fatalf("expected error, got %f and no error", n)
		} else if err != ErrRange {
			t.Fatalf("expected ErrRange error, got error %#v", err)
		}
		if n, err := Value(`1.8e529`).Float(64); err == nil {
			t.Fatalf("expected error, got %f and no error", n)
		} else if err != ErrRange {
			t.Fatalf("expected error ErrRanger, got error %#v", err)
		}
	})

	t.Run("Format", func(t *testing.T) {
		tests := []struct {
			format   string
			data     string
			expected string
		}{
			// null.
			{`%s`, `null`, `null`},
			{`%q`, `null`, `"null"`},
			{`%v`, `null`, `[110 117 108 108]`},
			{`%#v`, `null`, `json.Value{0x6e, 0x75, 0x6c, 0x6c}`},
			{`%T`, `null`, `json.Value`},
			// bool.
			{`%s`, `true`, `true`},
			{`%q`, `true`, `"true"`},
			{`%v`, `true`, `[116 114 117 101]`},
			{`%#v`, `true`, `json.Value{0x74, 0x72, 0x75, 0x65}`},
			{`%T`, `true`, `json.Value`},
			// integer.
			{`%s`, `5`, `5`},
			{`%q`, `5`, `"5"`},
			{`%v`, `5`, `[53]`},
			{`%#v`, `5`, `json.Value{0x35}`},
			{`%T`, `5`, `json.Value`},
			// string.
			{`%s`, `"foo"`, `"foo"`},
			{`%q`, `"foo"`, `"\"foo\""`},
			{`%v`, `"foo"`, `[34 102 111 111 34]`},
			{`%#v`, `"foo"`, `json.Value{0x22, 0x66, 0x6f, 0x6f, 0x22}`},
			{`%T`, `"foo"`, `json.Value`},
			// object.
			{`%s`, `{"boo":5}`, `{"boo":5}`},
			{`%q`, `{"boo":5}`, `"{\"boo\":5}"`},
			{`%v`, `{"boo":5}`, `[123 34 98 111 111 34 58 53 125]`},
			{`%#v`, `{"boo":5}`, `json.Value{0x7b, 0x22, 0x62, 0x6f, 0x6f, 0x22, 0x3a, 0x35, 0x7d}`},
			{`%T`, `{"boo":5}`, `json.Value`},
			// array.
			{`%s`, `[1,2,3]`, `[1,2,3]`},
			{`%q`, `[1,2,3]`, `"[1,2,3]"`},
			{`%v`, `[1,2,3]`, `[91 49 44 50 44 51 93]`},
			{`%#v`, `[1,2,3]`, `json.Value{0x5b, 0x31, 0x2c, 0x32, 0x2c, 0x33, 0x5d}`},
			{`%T`, `[1,2,3]`, `json.Value`},
		}
		for _, test := range tests {
			got := fmt.Sprintf(test.format, Value(test.data))
			if test.expected != got {
				t.Fatalf("expected `%s`, got `%s`", test.expected, got)
			}
		}
	})

	t.Run("Get", func(t *testing.T) {
		tests := []struct {
			data  string
			path  []string
			value Value
			ok    bool
		}{
			{`null`, []string{"a"}, nil, false},
			{`{"a": 1, "b": 2}`, []string{"a"}, Value("1"), true},
			{`{"a": 1, "b": 2}`, []string{"b"}, Value("2"), true},
			{`{"a": 1, "b": 2}`, []string{"b"}, Value("2"), true},
			{`{"a": 1, "b": 2}`, []string{"c"}, nil, false},
			{`{"a": 1, "b": {"c": true}}`, []string{"b", "c"}, Value(`true`), true},
			{`{"a": 1, "b": {"c": true}}`, []string{"a", "c"}, nil, false},
			{`{"a": 1, "b": {"c": {"d": null, "e": "foo", "f": [1, 2]}}}`, []string{"b", "c"}, Value(`{"d": null, "e": "foo", "f": [1, 2]}`), true},
			{`{"a": 1, "b": {"c": {"d": null, "e": "foo", "f": [1, 2]}}}`, []string{"b", "c", "e"}, Value(`"foo"`), true},
			{`{"a": 1, "b": {"c": {"d": null, "e": "foo", "f": [1, 2]}}}`, []string{"b", "c", "d"}, Value(`null`), true},
			{`{"a": 1, "b": {"c": {"d": null, "e": "foo", "f": [1, 2]}}}`, []string{"b", "c", "f"}, Value(`[1, 2]`), true},
		}
		for _, test := range tests {
			got, ok := Value(test.data).Get(test.path)
			if test.ok != ok {
				t.Fatalf("expected %t, got %t", test.ok, ok)
			}
			if !reflect.DeepEqual(test.value, got) {
				t.Fatalf("expected `%s`, got `%s`", test.value, string(got))
			}
		}
	})

	t.Run("Int", func(t *testing.T) {
		if n, err := Value(`-55`).Int(); err != nil {
			t.Fatalf("expected -55, got error %q", err)
		} else if n != -55 {
			t.Fatalf("expected -55, got %d", n)
		}
		if n, err := Value(strconv.FormatInt(math.MaxInt64, 10)).Int(); err != nil {
			t.Fatalf("expected %d, got error %q", math.MaxInt64, err)
		} else if n != math.MaxInt64 {
			t.Fatalf("expected %d, got %d", math.MaxInt64, n)
		}
		if n, err := Value(`-3.0`).Int(); err != nil {
			t.Fatalf("expected -3, got error %q", err)
		} else if n != -3 {
			t.Fatalf("expected -3, got %d", n)
		}
		if n, err := Value(`3.45`).Int(); err == nil {
			t.Fatalf("expected error, got %d and no error", n)
		}
		if n, err := Value(`74068198354071205726051295`).Uint(); err == nil {
			t.Fatalf("expected error, got %d and no error", n)
		} else if err != ErrRange {
			t.Fatalf("expected error ErrRange, got error %#v", err)
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

	t.Run("IsEmpty", func(t *testing.T) {
		if !Value(` ""`).IsEmpty() {
			t.Fatal("expected true, got false")
		}
		if !Value(` [ ]`).IsEmpty() {
			t.Fatal("expected true, got false")
		}
		if !Value(` { }`).IsEmpty() {
			t.Fatal("expected true, got false")
		}
		if Value(`"a"`).Bool() {
			t.Fatal("expected false, got true")
		}
		if Value(`[ 1, 2, 3 ]`).Bool() {
			t.Fatal("expected false, got true")
		}
		if Value(` { "a": true }`).Bool() {
			t.Fatal("expected false, got true")
		}
		if Value(`null`).Bool() {
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
		tests := []struct {
			data  string
			path  []string
			value Value
			err   error
		}{
			{`null`, []string{"a"}, nil, NotExistError{Kind: Null}},
			{`{"a": 1, "b": 2}`, []string{"a"}, Value("1"), nil},
			{`{"a": 1, "b":2}`, []string{"b"}, Value("2"), nil},
			{`{"a": 1, "b"  :2}`, []string{"b"}, Value("2"), nil},
			{`{"a": 1, "b": 2}`, []string{"c"}, nil, NotExistError{Kind: Object}},
			{"{\"a\": 1, \"b\": {\"c\"\t :\n\ttrue}}", []string{"b", "c"}, Value(`true`), nil},
			{`{"a": 1, "b": {"c": true}}`, []string{"a", "c"}, nil, NotExistError{Index: 1, Kind: Number}},
			{`{"a": 1, "b": {"c": {"d": null, "e": "foo", "f": [1, 2]}}}`, []string{"b", "c"}, Value(`{"d": null, "e": "foo", "f": [1, 2]}`), nil},
			{`{"a": 1, "b": {"c": {"d": null, "e": "foo", "f": [1, 2]}}}`, []string{"b", "c", "e"}, Value(`"foo"`), nil},
			{`{"a": 1, "b": {"c": {"d": null, "e": "foo", "f": [1, 2]}}}`, []string{"b", "c", "d"}, Value(`null`), nil},
			{`{"a": 1, "b": {"c": {"d": null, "e": "foo", "f": [1, 2]}}}`, []string{"b", "c", "f"}, Value(`[1, 2]`), nil},
		}
		for _, test := range tests {
			got, err := Value(test.data).Lookup(test.path)
			if !reflect.DeepEqual(test.err, err) {
				t.Fatalf("expected error '%#v', got error '%#v'", test.err, err)
			}
			if !reflect.DeepEqual(test.value, got) {
				t.Fatalf("expected `%s`, got `%s`", test.value, string(got))
			}
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

	t.Run("NumElement", func(t *testing.T) {
		if n := Value(`[ ] `).NumElement(); n != 0 {
			t.Fatalf("expected 0, got %d", n)
		}
		if n := Value(`[1, 6,0,23 ]`).NumElement(); n != 4 {
			t.Fatalf("expected 4, got %d", n)
		}
	})

	t.Run("NumProperty", func(t *testing.T) {
		if n := Value(` { } `).NumProperty(); n != 0 {
			t.Fatalf("expected 0, got %d", n)
		}
		if n := Value(`{ "a": 5, "b": [8, 6, 1], "c" : true} `).NumProperty(); n != 3 {
			t.Fatalf("expected 3, got %d", n)
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
			t.Fatalf("expected 816, got %d", n)
		}
		if n, err := Value(strconv.FormatUint(math.MaxUint64, 10)).Uint(); err != nil {
			t.Fatalf("expected %d, got error %q", uint64(math.MaxUint64), err)
		} else if n != math.MaxUint64 {
			t.Fatalf("expected %d, got %d", uint64(math.MaxUint64), n)
		}
		if n, err := Value(`3.0`).Uint(); err != nil {
			t.Fatalf("expected 3, got error %q", err)
		} else if n != 3 {
			t.Fatalf("expected 3, got %d", n)
		}
		if n, err := Value(`8.2`).Uint(); err == nil {
			t.Fatalf("expected error, got %d and no error", n)
		}
		if n, err := Value(`74068198354071205726051295`).Uint(); err == nil {
			t.Fatalf("expected error, got %d and no error", n)
		} else if err != ErrRange {
			t.Fatalf("expected error ErrRange, got error %#v", err)
		}
	})

	t.Run("UnmarshalJSON nil", func(t *testing.T) {
		expected := errors.New("UnmarshalJSON on nil pointer")
		var v *Value
		err := v.UnmarshalJSON([]byte("{}"))
		if !reflect.DeepEqual(err, expected) {
			t.Fatalf("expected error %v (type %T), got error %v (type %T)", expected, expected, err, err)
		}
		if v != nil {
			t.Fatalf("expected unchanged v, got `%s`", string(*v))
		}
	})

	t.Run("UnmarshalJSON", func(t *testing.T) {

		tests := []struct {
			value    string
			data     string
			expected string
			err      error
		}{
			{"{}", `[]`, ``, errors.New("UnmarshalJSON on non-empty value")},
			{"", `""`, `""`, nil},
			{"", ` "" `, `""`, nil},
			{"", `"foo"`, `"foo"`, nil},
			{"", `"foo`, ``, ErrInvalidJSON},
			{"", `{"foo": true, "boo": 5 } `, `{"foo": true, "boo": 5 }`, nil},
			{"", "\"\xFF\"", ``, ErrInvalidJSON},
		}
		for _, test := range tests {
			v := Value(test.value)
			err := v.UnmarshalJSON([]byte(test.data))
			if !reflect.DeepEqual(err, test.err) {
				t.Fatalf("expected error %v (type %T), got error %v (type %T)", test.err, test.err, err, err)
			}
			if test.err != nil {
				if string(v) != test.value {
					t.Fatalf("expected unchanged v, got `%s`", string(v))
				}
				continue
			}
			if got := string(v); test.expected != got {
				t.Fatalf("expected `%s`, got `%s`", test.expected, got)
			}
		}
	})

}

func TestAlloc(t *testing.T) {

	t.Run("Format", func(t *testing.T) {
		value := Value(`{"id":1,"name":"Alice","email":"alice@example.com","age":30,"registered":"2024-01-15","isActive":true}`)
		a := testing.AllocsPerRun(10, func() {
			_ = fmt.Sprintf("%#v", value)
		})
		if a != 2 {
			t.Fatalf("expected 2 allocations, got %.0f", a)
		}
	})

	t.Run("Lookup", func(t *testing.T) {
		value := Value(`{"id":5,"name":"Alice","email":"alice@example.com","age":30,"address":{"street":"123 Main St","city":"Wonderland","zip":"12345"}}`)
		path := []string{"address", "city"}
		a := testing.AllocsPerRun(1000, func() { _, _ = value.Lookup(path) })
		if a != 0 {
			t.Fatalf("expected 1 allocations, got %.0f", a)
		}
	})

	t.Run("Elements", func(t *testing.T) {
		value := Value(`[0,1,2,3,4,5,6,7,8,9]`)
		a := testing.AllocsPerRun(1000, func() {
			for _, _ = range value.Elements() {
			}
		})
		if a != 7 {
			t.Fatalf("expected 8 allocations, got %.0f", a)
		}
	})

	t.Run("Properties", func(t *testing.T) {
		value := Value(`{"id":1,"name":"Alice","email":"alice@example.com","age":30,"registered":"2024-01-15","isActive":true}`)
		a := testing.AllocsPerRun(1000, func() {
			for _, _ = range value.Properties() {
			}
		})
		if a != 13 {
			t.Fatalf("expected 14 allocations, got %.0f", a)
		}
	})

}

func Benchmark_Elements(b *testing.B) {
	value := Value(`[0,1,2,3,4,5,6,7,8,9]`)
	for i := 0; i < b.N; i++ {
		for _, _ = range value.Elements() {
		}
	}
}

func Benchmark_Lookup(b *testing.B) {
	value := Value(`{"id":5,"name":"Alice","email":"alice@example.com","age":30,"address":{"street":"123 Main St","city":"Wonderland","zip":"12345"}}`)
	path := []string{"address", "city"}
	for i := 0; i < b.N; i++ {
		_, _ = value.Lookup(path)
	}
}

func Benchmark_Properties(b *testing.B) {
	value := Value(`{"id":1,"name":"Alice","email":"alice@example.com","age":30,"registered":"2024-01-15","isActive":true}`)
	for i := 0; i < b.N; i++ {
		for _, _ = range value.Properties() {
		}
	}
}
