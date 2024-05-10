//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package apis

import (
	"testing"

	"github.com/open2b/chichi/types"
)

func Test_checkConflictingProperties(t *testing.T) {
	tests := []struct {
		schema types.Type
		err    string
	}{
		{
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Text()},
			}),
		},
		{
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Text()},
				{Name: "x_a", Type: types.Text()},
				{Name: "x_b", Type: types.Text()},
			}),
		},
		{
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
				})},
				{Name: "x_a", Type: types.Text()},
				{Name: "x_b", Type: types.Text()},
			}),
			err: `two or more properties cannot have the same representation as column "x_a"`,
		},
		{
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
			err: `two or more properties cannot have the same representation as column "x_y_a"`,
		},
		{
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
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
				})},
				{Name: "x_a", Type: types.Text()},
				{Name: "x_b", Type: types.Text()},
			}),
			err: `two or more properties cannot have the same representation as column "x_a"`,
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			gotErr := checkConflictingProperties(test.schema)
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
