// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"testing"

	"github.com/meergo/meergo/core/internal/util"
	"github.com/meergo/meergo/core/types"
)

func Test_IsMetaProperty(t *testing.T) {
	tests := []struct {
		p        types.Property
		expected bool
	}{
		{types.Property{}, false}, // invalid property, shouldn't happen.
		{types.Property{Name: "a", Type: types.Int(32)}, false},
		{types.Property{Name: "_", Type: types.Int(32)}, true},
		{types.Property{Name: "hello", Type: types.Int(32)}, false},
		{types.Property{Name: "_hello", Type: types.Int(32)}, true},
		{types.Property{Name: "__hello", Type: types.Int(32)}, true},
		{types.Property{Name: "__", Type: types.Int(32)}, true},
		{types.Property{Name: "____", Type: types.Int(32)}, true},
		{types.Property{Name: "__hello__", Type: types.Int(32)}, true},
		{types.Property{Name: "__h__", Type: types.Int(32)}, true},
		{types.Property{Name: "__hey_test__", Type: types.Int(32)}, true},
	}
	for _, test := range tests {
		got := IsMetaProperty(test.p.Name)
		if test.expected != got {
			t.Errorf("%#v: expected %t, got %t", test.p, test.expected, got)
		}
	}
}

var typ = types.Int(32)

var schema = types.Object([]types.Property{
	{Name: "__id__", Type: typ},
	{Name: "a", Type: typ},
	{Name: "b", Type: typ},
	{Name: "c", Type: types.Object([]types.Property{
		{Name: "a", Type: typ},
		{Name: "b", Type: typ},
		{Name: "d", Type: typ},
	})},
	{Name: "dd", Type: types.Object([]types.Property{
		{Name: "ee_f", Type: typ},
		{Name: "ed_g", Type: typ},
	})},
	{Name: "e_f", Type: types.Object([]types.Property{
		{Name: "g_h", Type: typ},
		{Name: "i", Type: typ},
	})},
	{Name: "f", Type: types.Object([]types.Property{
		{Name: "g", Type: typ},
		{Name: "h_", Type: typ},
		{Name: "i", Type: typ},
		{Name: "j_k", Type: typ},
	})},
	{Name: "g", Type: types.Object([]types.Property{
		{Name: "a_", Type: typ},
		{Name: "c_", Type: typ},
	})},
	{Name: "h", Type: types.Object([]types.Property{
		{Name: "a_", Type: typ},
		{Name: "b", Type: typ},
	})},
	{Name: "i", Type: types.Object([]types.Property{
		{Name: "a", Type: types.Object([]types.Property{
			{Name: "b", Type: typ},
			{Name: "c", Type: typ},
		})},
		{Name: "b_c", Type: typ},
	})},
	{Name: "k", Type: typ},
	{Name: "k_", Type: typ},
	{Name: "k_a", Type: typ},
})

func Test_propertiesToColumns(t *testing.T) {

	columns := []types.Property{
		{Name: "__id__", Type: typ},
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

	got := util.PropertiesToColumns(schema.Properties())
	if len(got) != len(columns) {
		t.Fatalf("expected %d columns, got %d", len(columns), len(got))
	}
	for i, c := range got {
		e := columns[i]
		if c.Name != e.Name {
			t.Fatalf("expected column name %q, got %q", e.Name, c.Name)
		}
		if !types.Equal(c.Type, e.Type) {
			t.Fatalf("type of column %q is not the expected one", e.Name)
		}
	}

}
