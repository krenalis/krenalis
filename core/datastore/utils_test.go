//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package datastore

import (
	"testing"

	"github.com/meergo/meergo/core/util"
	"github.com/meergo/meergo/types"
)

var typ = types.Int(32)

var properties = []types.Property{
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
}

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

	got := util.PropertiesToColumns(types.Object(properties))
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
