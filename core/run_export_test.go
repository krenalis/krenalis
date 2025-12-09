// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/meergo/meergo/core/internal/connections"
	"github.com/meergo/meergo/tools/types"
)

func TestConvertToExternal(t *testing.T) {
	tests := []struct {
		v        any
		in, ex   types.Type
		expected any
		err      error
	}{
		{v: -23, in: types.Int(32), ex: types.Int(64), expected: -23},
		{v: uint(905), in: types.Int(32).Unsigned(), ex: types.Int(16).Unsigned(), expected: uint(905)},
		{v: "bob", in: types.String(), ex: types.String(), expected: "bob"},
		{v: "b04da085-b620-4b56-9cda-308c0377f02a", in: types.UUID(), ex: types.UUID(), expected: "b04da085-b620-4b56-9cda-308c0377f02a"},
		{v: 331, in: types.Int(32), ex: types.Int(32).Unsigned(), expected: uint(331)},
		{v: uint(229401), in: types.Int(32).Unsigned(), ex: types.Int(64), expected: 229401},
		{v: -638, in: types.Int(32), ex: types.String(), expected: "-638"},
		{v: uint(93894), in: types.Int(32).Unsigned(), ex: types.String(), expected: "93894"},
		{v: "-3372960", in: types.String(), ex: types.Int(32), expected: -3372960},
		{v: "22", in: types.String(), ex: types.Int(8).Unsigned(), expected: uint(22)},
		{v: "5B358B26-7FF1-4126-8283-661A6CE656CF", in: types.String(), ex: types.UUID(), expected: "5b358b26-7ff1-4126-8283-661a6ce656cf"},
		{v: "6eb95d08-f97d-4753-82a3-b0aa3ce21001", in: types.UUID(), ex: types.String(), expected: "6eb95d08-f97d-4753-82a3-b0aa3ce21001"},
		{v: "boo", in: types.String(), ex: types.String().WithValues("foo", "boo"), expected: "boo"},
		{v: "boo", in: types.String(), ex: types.String().WithMaxLength(3), expected: "boo"},
		{v: "bòò", in: types.String(), ex: types.String().WithMaxBytes(5), expected: "bòò"},
		{v: "boo", in: types.String(), ex: types.String().WithPattern(regexp.MustCompile(`^b..$`)), expected: "boo"},
		{v: 331, in: types.Int(16), ex: types.Int(8), err: errMatchingPropertyConversion("in", "ex")},
		{v: -57, in: types.Int(8), ex: types.Int(32).Unsigned(), err: errMatchingPropertyConversion("in", "ex")},
		{v: "bob", in: types.String(), ex: types.String().WithMaxLength(2), err: errMatchingPropertyConversion("in", "ex")},
		{v: "bòb", in: types.String(), ex: types.String().WithMaxBytes(3), err: errMatchingPropertyConversion("in", "ex")},
		{v: "boo", in: types.String(), ex: types.String().WithPattern(regexp.MustCompile(`^f..$`)), err: errMatchingPropertyConversion("in", "ex")},
		{v: "bob", in: types.String(), ex: types.String().WithValues("foo", "boo"), err: errMatchingPropertyConversion("in", "ex")},
		{v: "ABCDEF", in: types.String(), ex: types.UUID(), err: errMatchingPropertyConversion("in", "ex")},
		{v: "034e3414-6daa-40a9-a1ca-28c7d26f8014", in: types.UUID(), ex: types.String().WithMaxBytes(10), err: errMatchingPropertyConversion("in", "ex")},
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

func TestGetAttribute(t *testing.T) {
	cases := []struct {
		name       string
		attributes map[string]any
		path       string
		expected   any
		ok         bool
	}{
		{
			name: "flat path",
			attributes: map[string]any{
				"email": "user@example.com",
			},
			path:     "email",
			expected: "user@example.com",
			ok:       true,
		},
		{
			name: "nested path",
			attributes: map[string]any{
				"profile": map[string]any{
					"address": map[string]any{
						"city": "Rome",
					},
				},
			},
			path:     "profile.address.city",
			expected: "Rome",
			ok:       true,
		},
		{
			name: "nested path",
			attributes: map[string]any{
				"address": map[string]any{
					"country": "IT",
				},
			},
			path: "address.city",
			ok:   false,
		},
	}
	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			got, ok := getAttribute(test.attributes, test.path)
			if got != test.expected {
				t.Fatalf("expected %v, got %v", test.expected, got)
			}
			if ok != test.ok {
				t.Fatalf("expected ok %t, got ok %t", test.ok, ok)
			}
		})
	}
}

func TestNewPathPlaceholderReplacer(t *testing.T) {
	now := time.Date(2035, 10, 30, 16, 33, 25, 0, time.UTC)
	tests := []struct {
		path     string
		expected string
		err      string
	}{

		// Valid.
		{path: "/files/users/${ today }.csv", expected: "/files/users/2035-10-30.csv"},
		{path: "/files/users/${ now }.csv", expected: "/files/users/2035-10-30-16-33-25.csv"},
		{path: "/files/users/${ unix }.csv", expected: "/files/users/2077374805.csv"},
		{path: "/files/users/${ UNIX }.csv", expected: "/files/users/2077374805.csv"},
		{path: "/files/users/${ Today }.csv", expected: "/files/users/2035-10-30.csv"},
		{path: "/files/users/${   Now }.csv", expected: "/files/users/2035-10-30-16-33-25.csv"},

		// Errors.
		{path: "${ today }} ${ yesterday }", err: "placeholder \"yesterday\" does not exist"},
		{path: "${ today }} ${ YESTERDAY }", err: "placeholder \"YESTERDAY\" does not exist"},
		{path: "${ invalid1 }} ${ invalid2 }", err: "placeholder \"invalid1\" does not exist"},
		{path: "/files/users/${ yesterday }.csv", err: "placeholder \"yesterday\" does not exist"},
	}
	replacer := newPathPlaceholderReplacer(now)
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			// This test here tests the "newPathPlaceholderReplacer" function,
			// and assumes that connections.ReplacePlaceholders is correct and
			// already tested elsewhere.
			got, gotErr := connections.ReplacePlaceholders(test.path, replacer)
			var gotErrStr string
			if gotErr != nil {
				gotErrStr = gotErr.Error()
			}
			if test.err != gotErrStr {
				t.Fatalf("expected error %q, got %q", test.err, gotErrStr)
			}
			if test.expected != got {
				t.Fatalf("expected %q, got %q", test.expected, got)
			}
		})
	}
}

func TestSetAttribute(t *testing.T) {

	t.Run("email", func(t *testing.T) {
		attributes := map[string]any{}
		setAttribute(attributes, "email", "user@example.com")
		got, ok := attributes["email"]
		if !ok {
			t.Fatal("expected top-level property to be set")
		}
		if got != "user@example.com" {
			t.Fatalf("expected %v, got %v", "user@example.com", got)
		}
	})

	t.Run("profile.address.city", func(t *testing.T) {
		attributes := map[string]any{
			"profile": map[string]any{
				"address": map[string]any{
					"street": "Via Veneto 143",
				},
			},
		}
		setAttribute(attributes, "profile.address.city", "Rome")
		profile, ok := attributes["profile"].(map[string]any)
		if !ok {
			t.Fatal("expected 'profile' map to exist")
		}
		address, ok := profile["address"].(map[string]any)
		if !ok {
			t.Fatal("expected 'address' map to exist")
		}
		got, ok := address["city"]
		if !ok {
			t.Fatal("expected 'city' property to be set")
		}
		if got != "Rome" {
			t.Fatalf("expected %v, got %v", "Rome", got)
		}
	})

	t.Run("profile.name", func(t *testing.T) {
		attributes := map[string]any{
			"profile": map[string]any{
				"address": map[string]any{
					"street": "Via Veneto 143",
				},
			},
		}
		setAttribute(attributes, "profile.name", "Marcello")
		profile, ok := attributes["profile"].(map[string]any)
		if !ok {
			t.Fatal("expected 'profile' map to exist")
		}
		got, ok := profile["name"]
		if !ok {
			t.Fatal("expected 'name' property to be set")
		}
		if got != "Marcello" {
			t.Fatalf("expected %v, got %v", "Marcello", got)
		}
	})

	t.Run("games.tetris.score", func(t *testing.T) {
		attributes := map[string]any{}
		setAttribute(attributes, "games.tetris.score", 704)
		games, ok := attributes["games"].(map[string]any)
		if !ok {
			t.Fatal("expected 'games' map to exist")
		}
		tetris, ok := games["tetris"].(map[string]any)
		if !ok {
			t.Fatal("expected 'name' map to exist")
		}
		got, ok := tetris["score"]
		if !ok {
			t.Fatal("expected 'score' property to be set")
		}
		if got != 704 {
			t.Fatalf("expected %v, got %v", "Marcello", got)
		}
	})

}
