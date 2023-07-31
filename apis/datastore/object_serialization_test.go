//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package datastore

import (
	"testing"

	"chichi/connector/types"

	"github.com/google/go-cmp/cmp"
)

func TestColumnsToProperties(t *testing.T) {

	typ := types.Int()

	columns := []types.Property{
		{Name: "a", Type: typ},
		{Name: "b", Type: typ},
		{Name: "c_a", Type: typ},
		{Name: "c_b", Type: typ},
		{Name: "c_d", Type: typ},
		{Name: "dd_ee_f", Type: typ},
		{Name: "dd_ed_g", Type: typ},
		{Name: "e_f_g_h", Type: typ},
		{Name: "e_f_i", Type: typ},
		{Name: "f_g", Type: typ},
		{Name: "_f_h", Type: typ},
		{Name: "f_i", Type: typ},
		{Name: "f_j_k", Type: typ},
		{Name: "_g_a", Type: typ},
		{Name: "_g_c", Type: typ},
		{Name: "_h_a", Type: typ},
		{Name: "h_b", Type: typ},
		{Name: "i_a_b", Type: typ},
		{Name: "i_a_c", Type: typ},
		{Name: "i_b_c", Type: typ},
		{Name: "k", Type: typ},
		{Name: "k_", Type: typ},
		{Name: "k_a", Type: typ},
	}

	expected := []types.Property{
		{Name: "a", Type: typ},
		{Name: "b", Type: typ},
		{Name: "c", Type: types.Object([]types.Property{
			{Name: "a", Type: typ},
			{Name: "b", Type: typ},
			{Name: "d", Type: typ},
		}), Flat: true},
		{Name: "dd", Type: types.Object([]types.Property{
			{Name: "ee_f", Type: typ},
			{Name: "ed_g", Type: typ},
		}), Flat: true},
		{Name: "e_f", Type: types.Object([]types.Property{
			{Name: "g_h", Type: typ},
			{Name: "i", Type: typ},
		}), Flat: true},
		{Name: "f", Type: types.Object([]types.Property{
			{Name: "g", Type: typ},
			{Name: "i", Type: typ},
			{Name: "j_k", Type: typ},
		}), Flat: true},
		{Name: "h", Type: types.Object([]types.Property{
			{Name: "b", Type: typ},
		}), Flat: true},
		{Name: "i", Type: types.Object([]types.Property{
			{Name: "a", Type: types.Object([]types.Property{
				{Name: "b", Type: typ},
				{Name: "c", Type: typ},
			}), Flat: true},
			{Name: "b_c", Type: typ},
		}), Flat: true},
		{Name: "k", Type: typ},
		{Name: "k_", Type: typ},
		{Name: "k_a", Type: typ},
	}

	properties, err := ColumnsToProperties(columns)
	if err != nil {
		t.Fatal(err)
	}
	if len(properties) != len(expected) {
		t.Fatalf("expected %d properties, got %d", len(expected), len(properties))
	}
	for i, p := range properties {
		e := expected[i]
		if p.Name != e.Name {
			t.Fatalf("expected property name %q, got %q", e.Name, p.Name)
		}
		if !p.Type.EqualTo(e.Type) {
			t.Fatalf("type of property %q is not the expected one", e.Name)
		}
	}

}

func TestColumnsCommonPrefix(t *testing.T) {

	tests := []struct {
		names  []string
		prefix string
		n      int
	}{
		// with prefix.
		{[]string{"a_b", "a_c", "b_a"}, "a_", 2},
		{[]string{"a_b_c", "a_b", "c"}, "a_", 2},
		{[]string{"a_b", "a_b_c"}, "a_", 2},
		{[]string{"a_b_c", "a_b_d"}, "a_b_", 2},
		{[]string{"_a_b", "a_c"}, "a_", 2},
		{[]string{"a_b", "_a_c"}, "a_", 2},
		{[]string{"_a_b", "_a_c"}, "a_", 2},

		// without prefix.
		{[]string{"a"}, "", 0},
		{[]string{"a_"}, "", 0},
		{[]string{"a_b"}, "", 0},
		{[]string{"a", "b"}, "", 0},
		{[]string{"a", "b_a"}, "", 0},
		{[]string{"a_", "a"}, "", 0},
		{[]string{"a_", "a"}, "", 0},
		{[]string{"a", "a_"}, "", 0},
		{[]string{"a_b", "e_b"}, "", 0},
		{[]string{"a_b", "b", "a_c"}, "", 0},
		{[]string{"a_b", "b_a", "b_b"}, "", 0},
		{[]string{"a_b", "_b", "a_c"}, "", 0},
		{[]string{"a_", "a_b"}, "", 0},
		{[]string{"a_b", "a_"}, "", 0},
	}

	for _, test := range tests {
		columns := make([]types.Property, len(test.names))
		for i, name := range test.names {
			columns[i] = types.Property{Name: name}
		}
		prefix, n := columnsCommonPrefix(columns)
		if n != test.n {
			t.Fatalf("%#v: expecting n %d, got %d", test.names, test.n, n)
		}
		if prefix != test.prefix {
			t.Fatalf("%#v: expecting prefix %q, got %q", test.names, test.prefix, prefix)
		}
	}
}

func TestSerializeRow(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "a", Type: types.Int()},
		{Name: "b", Type: types.Object([]types.Property{
			{Name: "c", Type: types.Text()},
			{Name: "d", Type: types.Object([]types.Property{
				{Name: "e", Type: types.Boolean()},
				{Name: "f", Type: types.Text()},
			})},
			{Name: "g", Type: types.Array(types.Float())},
			{Name: "h", Type: types.Array(types.Array(types.Text()))},
			{Name: "i", Type: types.Array(types.Object([]types.Property{
				{Name: "j", Type: types.UInt()},
				{Name: "k", Flat: true, Type: types.Object([]types.Property{
					{Name: "l", Type: types.Text()},
					{Name: "m", Flat: true, Type: types.Object([]types.Property{
						{Name: "n", Type: types.Text()},
						{Name: "o", Type: types.Text()},
					}),
					}})},
			}))},
			{Name: "p", Type: types.Map(types.Text())},
			{Name: "q", Type: types.Map(types.Object([]types.Property{
				{Name: "r", Type: types.Text()},
				{Name: "s", Flat: true, Type: types.Object([]types.Property{
					{Name: "t", Type: types.Int()},
					{Name: "u", Type: types.Int()},
				})},
			}))},
		})},
		{Name: "v", Flat: true, Type: types.Object([]types.Property{
			{Name: "z", Type: types.Float32()},
		})},
	})

	row := map[string]any{
		"a": 56,
		"b": map[string]any{
			"c": "foo",
			"d": map[string]any{
				"e": true,
				"f": "boo",
			},
			"g": []any{1.22, -5.96},
			"h": []any{[]any{"foo", "boo"}, []any{"faa", "baa"}},
			"i": []any{map[string]any{
				"j": uint(84103),
				"k": map[string]any{
					"l": "foo",
					"m": map[string]any{
						"n": "foo",
						"o": "boo",
					},
				},
			}},
			"p": map[string]any{"foo": "boo", "boo": "foo"},
			"q": map[string]any{
				"foo": map[string]any{
					"r": "foo",
					"s": map[string]any{
						"t": 5,
						"u": 3,
					},
				},
				"boo": map[string]any{
					"r": "boo",
					"s": map[string]any{
						"t": 3,
						"u": -2,
					},
				},
			},
		},
		"v": map[string]any{
			"z": 3.14,
		},
	}

	expected := map[string]any{
		"a": 56,
		"b": map[string]any{
			"c": "foo",
			"d": map[string]any{
				"e": true,
				"f": "boo",
			},
			"g": []any{1.22, -5.96},
			"h": []any{[]any{"foo", "boo"}, []any{"faa", "baa"}},
			"i": []any{
				map[string]any{
					"j":     uint(84103),
					"k_l":   "foo",
					"k_m_n": "foo",
					"k_m_o": "boo",
				},
			},
			"p": map[string]any{
				"boo": "foo",
				"foo": "boo",
			},
			"q": map[string]any{
				"boo": map[string]any{
					"r":   "boo",
					"s_t": 3,
					"s_u": -2,
				},
				"foo": map[string]any{
					"r":   "foo",
					"s_t": 5,
					"s_u": 3,
				},
			},
		},
		"v_z": 3.14,
	}

	SerializeRow(row, schema)
	if !cmp.Equal(row, expected) {
		t.Fatalf("unexpected row")
	}

}
