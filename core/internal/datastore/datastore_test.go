// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"testing"

	"github.com/meergo/meergo/tools/types"
)

func Test_CheckConflictingProperties(t *testing.T) {
	tests := []struct {
		io     string
		schema types.Type
		err    string
	}{
		{
			io: "profile",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Text()},
			}),
		},
		{
			io: "profile",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Text()},
				{Name: "x_a", Type: types.Text()},
				{Name: "x_b", Type: types.Text()},
			}),
		},
		{
			io: "profile",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
				})},
				{Name: "x_a", Type: types.Text()},
				{Name: "x_b", Type: types.Text()},
			}),
			err: `two profile pipeline schema properties would have the same column name "x_a" in the data warehouse, case-insensitively`,
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
			err: `two input pipeline schema properties would have the same column name "x_y_a" in the data warehouse, case-insensitively`,
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
			err: `two output pipeline schema properties would have the same column name "x_a" in the data warehouse, case-insensitively`,
		},
		{
			io: "profile",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "A", Type: types.Text()},
				})},
				{Name: "x_a", Type: types.Text()},
				{Name: "x_b", Type: types.Text()},
			}),
			err: `two profile pipeline schema properties would have the same column name "x_a" in the data warehouse, case-insensitively`,
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
