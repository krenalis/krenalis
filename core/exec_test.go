// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/meergo/meergo/tools/types"
)

func Test_convertToExternal(t *testing.T) {
	tests := []struct {
		v        any
		in, ex   types.Type
		expected any
		err      error
	}{
		{v: -23, in: types.Int(32), ex: types.Int(64), expected: -23},
		{v: uint(905), in: types.Uint(32), ex: types.Uint(16), expected: uint(905)},
		{v: "bob", in: types.String(), ex: types.String(), expected: "bob"},
		{v: "b04da085-b620-4b56-9cda-308c0377f02a", in: types.UUID(), ex: types.UUID(), expected: "b04da085-b620-4b56-9cda-308c0377f02a"},
		{v: 331, in: types.Int(32), ex: types.Uint(32), expected: uint(331)},
		{v: uint(229401), in: types.Uint(32), ex: types.Int(64), expected: 229401},
		{v: -638, in: types.Int(32), ex: types.String(), expected: "-638"},
		{v: uint(93894), in: types.Uint(32), ex: types.String(), expected: "93894"},
		{v: "-3372960", in: types.String(), ex: types.Int(32), expected: -3372960},
		{v: "22", in: types.String(), ex: types.Uint(8), expected: uint(22)},
		{v: "5B358B26-7FF1-4126-8283-661A6CE656CF", in: types.String(), ex: types.UUID(), expected: "5b358b26-7ff1-4126-8283-661a6ce656cf"},
		{v: "6eb95d08-f97d-4753-82a3-b0aa3ce21001", in: types.UUID(), ex: types.String(), expected: "6eb95d08-f97d-4753-82a3-b0aa3ce21001"},
		{v: "boo", in: types.String(), ex: types.String().WithValues("foo", "boo"), expected: "boo"},
		{v: "boo", in: types.String(), ex: types.String().WithCharLen(3), expected: "boo"},
		{v: "bòò", in: types.String(), ex: types.String().WithByteLen(5), expected: "bòò"},
		{v: "boo", in: types.String(), ex: types.String().WithRegexp(regexp.MustCompile(`^b..$`)), expected: "boo"},
		{v: 331, in: types.Int(16), ex: types.Int(8), err: errMatchingPropertyConversion("in", "ex")},
		{v: -57, in: types.Int(8), ex: types.Uint(32), err: errMatchingPropertyConversion("in", "ex")},
		{v: "bob", in: types.String(), ex: types.String().WithCharLen(2), err: errMatchingPropertyConversion("in", "ex")},
		{v: "bòb", in: types.String(), ex: types.String().WithByteLen(3), err: errMatchingPropertyConversion("in", "ex")},
		{v: "boo", in: types.String(), ex: types.String().WithRegexp(regexp.MustCompile(`^f..$`)), err: errMatchingPropertyConversion("in", "ex")},
		{v: "bob", in: types.String(), ex: types.String().WithValues("foo", "boo"), err: errMatchingPropertyConversion("in", "ex")},
		{v: "ABCDEF", in: types.String(), ex: types.UUID(), err: errMatchingPropertyConversion("in", "ex")},
		{v: "034e3414-6daa-40a9-a1ca-28c7d26f8014", in: types.UUID(), ex: types.String().WithByteLen(10), err: errMatchingPropertyConversion("in", "ex")},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got, err := convertToExternal(test.v, test.in, test.ex, "in", "ex")
			if !reflect.DeepEqual(test.err, err) {
				t.Fatalf("expected error %#v, got %#v", test.err, err)
			}
			if test.expected != got {
				t.Fatalf("expected %#v, got %#v", test.expected, got)
			}
		})
	}
}

func Test_readPropertyFrom(t *testing.T) {
	cases := []struct {
		m          map[string]any
		prop       string
		expectedV  any
		expectedOk bool
	}{
		{
			m:          map[string]any{},
			prop:       "email",
			expectedV:  nil,
			expectedOk: false,
		},
		{
			m:          map[string]any{"email": "hello@example.com"},
			prop:       "email",
			expectedV:  "hello@example.com",
			expectedOk: true,
		},
		{
			m:          map[string]any{"traits": map[string]any{"email": "world@example.com"}},
			prop:       "traits.email",
			expectedV:  "world@example.com",
			expectedOk: true,
		},
		{
			m:          map[string]any{"traits": nil},
			prop:       "traits.email",
			expectedV:  nil,
			expectedOk: false,
		},
		{
			m:          map[string]any{"traits": 42},
			prop:       "traits.email",
			expectedV:  nil,
			expectedOk: false,
		},
		{
			m:          map[string]any{"traits": map[string]any{"email": "world@example.com"}},
			prop:       "traits.name",
			expectedV:  nil,
			expectedOk: false,
		},
	}
	for _, cas := range cases {
		t.Run("", func(t *testing.T) {
			gotV, gotOk := readPropertyFrom(cas.m, cas.prop)
			if !reflect.DeepEqual(gotV, cas.expectedV) {
				t.Fatalf("expected %#v, got %#v", cas.expectedV, gotV)
			}
			if gotOk != cas.expectedOk {
				t.Fatalf("expected ok = %t, got %t", cas.expectedOk, gotOk)
			}
		})
	}
}

// readPropertyFrom reads the property with the given path from m, returning its
// value (if found, otherwise nil) and a boolean indicating if the property path
// corresponds to a value in m or not.
func readPropertyFrom(m map[string]any, path string) (any, bool) {
	var name string
	for {
		name, path, _ = strings.Cut(path, ".")
		v, ok := m[name]
		if !ok {
			return nil, false
		}
		if path == "" {
			return v, true
		}
		m, ok = v.(map[string]any)
		if !ok {
			return nil, false
		}
	}
}
