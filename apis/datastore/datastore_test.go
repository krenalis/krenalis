//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package datastore

import (
	"testing"

	"github.com/meergo/meergo/types"
)

func Test_CheckConflictingProperties(t *testing.T) {
	tests := []struct {
		io     string
		schema types.Type
		err    string
	}{
		{
			io: "users",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Text()},
			}),
		},
		{
			io: "users",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Text()},
				{Name: "x_a", Type: types.Text()},
				{Name: "x_b", Type: types.Text()},
			}),
		},
		{
			io: "users",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
				})},
				{Name: "x_a", Type: types.Text()},
				{Name: "x_b", Type: types.Text()},
			}),
			err: `two properties in the users schema would have the same column name "x_a" in the data warehouse`,
		},
		{
			io: "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "y", Type: types.Object([]types.Property{
						{Name: "a", Type: types.Text()},
					})},
					{Name: "y_a", Type: types.Text()},
				})},
				{Name: "x_a", Type: types.Text()},
				{Name: "x_b", Type: types.Text()},
			}),
			err: `two properties in the input schema would have the same column name "x_y_a" in the data warehouse`,
		},
		{
			io: "output",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Object([]types.Property{
						{Name: "a", Type: types.Text()},
					})},
				})},
				{Name: "x_a", Type: types.Text()},
				{Name: "x_b", Type: types.Text()},
			}),
		},
		{
			io: "output",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
				})},
				{Name: "x_a", Type: types.Text()},
				{Name: "x_b", Type: types.Text()},
			}),
			err: `two properties in the output schema would have the same column name "x_a" in the data warehouse`,
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			gotErr := CheckConflictingProperties(test.io, test.schema)
			var gotErrStr string
			if gotErr != nil {
				gotErrStr = gotErr.Error()
			}
			if gotErrStr != test.err {
				t.Fatalf("expected error %q, got %q", test.err, gotErrStr)
			}
		})
	}
}
