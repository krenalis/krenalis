//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package warehouses

import (
	"testing"

	"chichi/connector/types"

	"github.com/google/go-cmp/cmp"
)

var typ = types.Int(32)

var properties = []types.Property{
	{Name: "a", Type: typ},
	{Name: "b", Type: typ},
	{Name: "c", Type: types.Object([]types.Property{
		{Name: "a", Type: typ},
		{Name: "b", Type: typ},
		{Name: "d", Type: typ},
	})},
	{Name: "dd", Type: types.Object([]types.Property{
		{Name: "eeF", Type: typ},
		{Name: "edG", Type: typ},
	})},
	{Name: "eF", Type: types.Object([]types.Property{
		{Name: "gH", Type: typ},
		{Name: "i", Type: typ},
	})},
	{Name: "f", Type: types.Object([]types.Property{
		{Name: "g", Type: typ},
		{Name: "i", Type: typ},
		{Name: "jK", Type: typ},
	})},
	{Name: "h", Type: types.Object([]types.Property{
		{Name: "b", Type: typ},
	})},
	{Name: "i", Type: types.Object([]types.Property{
		{Name: "a", Type: types.Object([]types.Property{
			{Name: "b", Type: typ},
			{Name: "c", Type: typ},
		})},
		{Name: "bC", Type: typ},
	})},
	{Name: "k", Type: typ},
	{Name: "kA", Type: typ},
}

func TestColumnsToProperties(t *testing.T) {

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
		{Name: "f_h_", Type: typ},
		{Name: "f_i", Type: typ},
		{Name: "f_j_k", Type: typ},
		{Name: "g_a_", Type: typ},
		{Name: "g_c_", Type: typ},
		{Name: "h_a_", Type: typ},
		{Name: "h_b", Type: typ},
		{Name: "i_a_b", Type: typ},
		{Name: "i_a_c", Type: typ},
		{Name: "i_b_c", Type: typ},
		{Name: "k", Type: typ},
		{Name: "k_", Type: typ},
		{Name: "k_a", Type: typ},
	}

	got, err := ColumnsToProperties(columns)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(properties) {
		t.Fatalf("expected %d properties, got %d", len(properties), len(got))
	}
	for i, p := range got {
		e := properties[i]
		if p.Name != e.Name {
			t.Fatalf("expected property name %q, got %q", e.Name, p.Name)
		}
		if !p.Type.EqualTo(e.Type) {
			t.Fatalf("type of property %q is not the expected one", e.Name)
		}
	}

}

func TestPropertiesToColumns(t *testing.T) {

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
		{Name: "f_i", Type: typ},
		{Name: "f_j_k", Type: typ},
		{Name: "h_b", Type: typ},
		{Name: "i_a_b", Type: typ},
		{Name: "i_a_c", Type: typ},
		{Name: "i_b_c", Type: typ},
		{Name: "k", Type: typ},
		{Name: "k_a", Type: typ},
	}

	got := PropertiesToColumns(properties)
	if len(got) != len(columns) {
		t.Fatalf("expected %d columns, got %d", len(columns), len(got))
	}
	for i, c := range got {
		e := columns[i]
		if c.Name != e.Name {
			t.Fatalf("expected column name %q, got %q", e.Name, c.Name)
		}
		if !c.Type.EqualTo(e.Type) {
			t.Fatalf("type of column %q is not the expected one", e.Name)
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
		{[]string{"a_b_", "a_c"}, "a_", 2},
		{[]string{"a_b", "a_c_"}, "a_", 2},
		{[]string{"a_b_", "a_c_"}, "a_", 2},
		{[]string{"_a_b", "_a_c", "_b"}, "_a_", 2},

		// without prefix.
		{[]string{"a"}, "", 0},
		{[]string{"_a"}, "", 0},
		{[]string{"a_b"}, "", 0},
		{[]string{"a", "b"}, "", 0},
		{[]string{"a", "b_a"}, "", 0},
		{[]string{"_a", "a"}, "", 0},
		{[]string{"a_", "a"}, "", 0},
		{[]string{"a", "_a"}, "", 0},
		{[]string{"a_b", "e_b"}, "", 0},
		{[]string{"a_b", "b", "a_c"}, "", 0},
		{[]string{"a_b", "b_a", "b_b"}, "", 0},
		{[]string{"a_b", "b_", "a_c"}, "", 0},
		{[]string{"_a", "a_b"}, "", 0},
		{[]string{"a_b", "_a"}, "", 0},
		{[]string{"a", "a_", "a_b"}, "", 0},
		{[]string{"_a", "_b", "_c"}, "", 0},
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

func Test_PropertyPathToColumn(t *testing.T) {
	tests := []struct {
		path string
		col  types.Property
		err  string
	}{
		{path: "a", col: types.Property{Name: "a", Type: types.Int(32)}},
		{path: "b.c", col: types.Property{Name: "b_c", Type: types.Text()}},
		{path: "b.i.j", err: "path refers to a non-object type"},
		{path: "VIA.z", col: types.Property{Name: "_v_i_a_z", Type: types.Float(32)}},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			col, err := PropertyPathToColumn(testSchema, test.path)
			// Check the error.
			if err != nil && test.err == "" {
				t.Fatalf("unexpected error %q", err)
			}
			if err == nil && test.err != "" {
				t.Fatalf("expected error %q, got no errors", test.err)
			}
			var errStr string
			if err != nil {
				errStr = err.Error()
			}
			if errStr != test.err {
				t.Fatalf("expected error %q, got error %q", test.err, errStr)
			}
			// Check the column.
			if col.Name != test.col.Name {
				t.Fatalf("expected column name %q, got %q", test.col.Name, col.Name)
			}
			if !col.Type.EqualTo(test.col.Type) {
				t.Fatalf("expected column type %s, got %s", test.col.Type, col.Type)
			}
		})
	}
}

var testSchema = types.Object([]types.Property{
	{Name: "A", Type: types.Text()},
	{Name: "a", Type: types.Int(32)},
	{Name: "b", Type: types.Object([]types.Property{
		{Name: "c", Type: types.Text()},
		{Name: "d", Type: types.Object([]types.Property{
			{Name: "e", Type: types.Boolean()},
			{Name: "f", Type: types.Text()},
		})},
		{Name: "g", Type: types.Array(types.Float(64))},
		{Name: "h", Type: types.Array(types.Array(types.Text()))},
		{Name: "i", Type: types.Array(types.Object([]types.Property{
			{Name: "j", Type: types.Uint(32)},
			{Name: "k", Type: types.Object([]types.Property{
				{Name: "l", Type: types.Text()},
				{Name: "m", Type: types.Object([]types.Property{
					{Name: "n", Type: types.Text()},
					{Name: "o", Type: types.Text()},
				}),
				}})},
		}))},
		{Name: "p", Type: types.Map(types.Text())},
		{Name: "q", Type: types.Map(types.Object([]types.Property{
			{Name: "r", Type: types.Text()},
			{Name: "s", Type: types.Object([]types.Property{
				{Name: "t", Type: types.Int(32)},
				{Name: "u", Type: types.Int(32)},
			})},
		}))},
	})},
	{Name: "VIA", Type: types.Object([]types.Property{
		{Name: "z", Type: types.Float(32)},
	})},
})

func TestSerializeRow(t *testing.T) {

	row := map[string]any{
		"A": "boo",
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
		"VIA": map[string]any{
			"z": 3.14,
		},
	}

	expected := map[string]any{
		"_a":    "boo",
		"a":     56,
		"b_c":   "foo",
		"b_d_e": true,
		"b_d_f": "boo",
		"b_g":   []any{1.22, -5.96},
		"b_h":   []any{[]any{"foo", "boo"}, []any{"faa", "baa"}},
		"b_i": []any{
			map[string]any{
				"j":     uint(84103),
				"k_l":   "foo",
				"k_m_n": "foo",
				"k_m_o": "boo",
			},
		},
		"b_p": map[string]any{
			"boo": "foo",
			"foo": "boo",
		},
		"b_q": map[string]any{
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
		"_v_i_a_z": 3.14,
	}

	SerializeRow(row, testSchema)
	if !cmp.Equal(row, expected) {
		t.Fatalf("unexpected row")
	}

}

func TestPropertyNameToColumnName(t *testing.T) {
	tests := []struct {
		p, c string
	}{
		{"a", "a"},
		{"ab", "ab"},
		{"aB", "a_b"},
		{"abC", "ab_c"},
		{"a5", "a5"},
		{"aBc", "a_bc"},
		{"aBC", "a_b_c"},
		{"aBcd", "a_bcd"},
		{"aBcD", "a_bc_d"},
		{"aB5d6", "a_b5d6"},
		{"aB5D6", "a_b5_d6"},
		{"A", "_a"},
		{"Ab", "_ab"},
		{"AB", "_a_b"},
		{"AbC", "_ab_c"},
	}
	for _, test := range tests {
		t.Run(test.p, func(t *testing.T) {
			c := PropertyNameToColumnName(test.p)
			if test.c != c {
				t.Fatalf("expected column %q, got %q", test.c, c)
			}
		})
	}
}

func TestColumnNameToPropertyName(t *testing.T) {
	tests := []struct {
		c, p string
	}{
		{"a", "a"},
		{"ab", "ab"},
		{"a_b", "aB"},
		{"ab_c", "abC"},
		{"a5", "a5"},
		{"a_bc", "aBc"},
		{"a_b_c", "aBC"},
		{"a_bcd", "aBcd"},
		{"a_bc_d", "aBcD"},
		{"a_b5d6", "aB5d6"},
		{"a_b5_d6", "aB5D6"},
		{"_a", "A"},
		{"_ab", "Ab"},
		{"_a_b", "AB"},
		{"_ab_c", "AbC"},
	}
	for _, test := range tests {
		t.Run(test.c, func(t *testing.T) {
			p := ColumnNameToPropertyName(test.c)
			if test.p != p {
				t.Fatalf("expected property %q, got %q", test.p, p)
			}
		})
	}
}
