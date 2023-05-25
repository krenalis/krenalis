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
)

func TestColumnsToProperties(t *testing.T) {

	typ := types.Int()

	columns := []*Column{
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
		columns := make([]*Column, len(test.names))
		for i, name := range test.names {
			columns[i] = &Column{Name: name}
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
