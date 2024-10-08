//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package postgresql

import (
	"reflect"
	"testing"

	"github.com/meergo/meergo/apis/datastore/warehouses"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/google/go-cmp/cmp"
)

func Test_rowEncoder(t *testing.T) {
	tests := []struct {
		columns  []warehouses.Column
		rows     [][]any
		expected [][]any
		ok       bool
	}{
		{
			columns:  []warehouses.Column{{Name: "c", Type: types.Boolean()}},
			rows:     [][]any{{true}, {false}},
			expected: [][]any{{true}, {false}},
			ok:       false,
		},
		{
			columns:  []warehouses.Column{{Name: "c", Type: types.Text()}},
			rows:     [][]any{{"boo"}, {"\x00foo"}},
			expected: [][]any{{"boo"}, {"foo"}},
			ok:       true,
		},
		{
			columns:  []warehouses.Column{{Name: "c", Type: types.JSON()}},
			rows:     [][]any{{json.Value(`"boo"`)}, {json.Value(`"\u0000foo"`)}},
			expected: [][]any{{json.Value(`"boo"`)}, {json.Value(`"foo"`)}},
			ok:       true,
		},
		{
			columns:  []warehouses.Column{{Name: "c", Type: types.Array(types.Text())}},
			rows:     [][]any{{[]any{"boo", "foo"}}, {[]any{"\x00foo", "boo", "\x00"}}},
			expected: [][]any{{[]any{"boo", "foo"}}, {[]any{"foo", "boo", ""}}},
			ok:       true,
		},
		{
			columns:  []warehouses.Column{{Name: "c", Type: types.Map(types.Int(32))}},
			rows:     [][]any{{map[string]any{"boo": 5}}, {map[string]any{"'boo\x00'": 7, "hello \x00world": 2}}},
			expected: [][]any{{json.Value(`{"boo":5}`)}, {json.Value(`{"'boo'":7,"hello world":2}`)}},
			ok:       true,
		},
		{
			columns:  []warehouses.Column{{Name: "c", Type: types.Map(types.JSON())}},
			rows:     [][]any{{map[string]any{"boo": json.Value(`{"a":5}`)}}, {map[string]any{"'boo\x00'": json.Value(`{"b":"\u0000foo\\u0000"}`)}}},
			expected: [][]any{{json.Value(`{"boo":"{\"a\":5}"}`)}, {json.Value(`{"'boo'":"{\"b\":\"\\u0000foo\\\\u0000\"}"}`)}},
			ok:       true,
		},
		{
			columns:  []warehouses.Column{{Name: "a", Type: types.Text()}, {Name: "b", Type: types.Float(32)}, {Name: "c", Type: types.Map(types.Text())}},
			rows:     [][]any{{"\x00boo", 1.234, map[string]any{"boo": ""}}, {"\x00", -73.55, map[string]any{"boo": "\x00foo", "hello\x00 world": "\x00"}}},
			expected: [][]any{{"boo", 1.234, json.Value(`{"boo":""}`)}, {"", -73.55, json.Value(`{"boo":"foo","hello world":""}`)}},
			ok:       true,
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			enc, ok := newRowEncoder(test.columns)
			if ok != test.ok {
				t.Fatalf("expected ok %t, got %t", test.ok, ok)
			}
			if !ok {
				return
			}
			for i, row := range test.rows {
				err := enc.encode(row)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(test.expected[i], row) {
					t.Fatalf("unexpected row:\n\texpected: %#v\n\tgot:      %#v\n\n%s\n", test.expected[i], row, cmp.Diff(test.expected[i], row))
				}
			}
		})
	}
}

func Test_stripZeroBytes(t *testing.T) {
	tests := []struct {
		s        string
		expected string
	}{
		{"", ""},
		{"\x00", ""},
		{"\x00\x00", ""},
		{"a", "a"},
		{"hello world", "hello world"},
		{"\x00hello\x00\x00 world\x00\x00", "hello world"},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := stripZeroBytes(test.s)
			if test.expected != got {
				t.Fatalf("expected %q, got %q", test.expected, got)
			}
		})
	}
}
