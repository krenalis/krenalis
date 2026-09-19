// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// negativeProfileCountWarehouse returns an invalid profile count.
type negativeProfileCountWarehouse struct {
	warehouses.Warehouse
}

// Count returns a negative row count.
func (negativeProfileCountWarehouse) Count(context.Context, string, []warehouses.Join, warehouses.Expr) (int, error) {
	return -1, nil
}

// TestProfileCountRejectsNegativeWarehouseValue verifies that profile counts are validated at the warehouse boundary.
func TestProfileCountRejectsNegativeWarehouseValue(t *testing.T) {

	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(warehouses.Warehouse(negativeProfileCountWarehouse{}))
	total, err := store.ProfileCount(t.Context(), nil, types.Type{})
	if err != nil {

		if total != 0 {
			t.Fatalf("expected a zero count with the error, got %d", total)
		}
		unavailableErr, ok := errors.AsType[*UnavailableError](err)
		if !ok {
			t.Fatalf("expected an *UnavailableError error, got %T", err)
		}
		if unavailableErr.Err.Error() != "profile count is negative" {
			t.Fatalf("unexpected underlying error: %s", unavailableErr.Err)
		}

		return
	}

	t.Fatal("expected an error")

}

func Test_CheckConflictingProperties(t *testing.T) {
	tests := []struct {
		io     string
		schema types.Type
		err    string
	}{
		{
			io: "profile",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.String()},
			}),
		},
		{
			io: "profile",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.String()},
				{Name: "x_a", Type: types.String()},
				{Name: "x_b", Type: types.String()},
			}),
		},
		{
			io: "profile",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String()},
				})},
				{Name: "x_a", Type: types.String()},
				{Name: "x_b", Type: types.String()},
			}),
			err: `two profile pipeline schema properties would have the same column name "x_a" in the data warehouse, case-insensitively`,
		},
		{
			io: "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "y", Type: types.Object([]types.Property{
						{Name: "a", Type: types.String()},
					})},
					{Name: "y_a", Type: types.String()},
				})},
				{Name: "x_a", Type: types.String()},
				{Name: "x_b", Type: types.String()},
			}),
			err: `two input pipeline schema properties would have the same column name "x_y_a" in the data warehouse, case-insensitively`,
		},
		{
			io: "output",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Object([]types.Property{
						{Name: "a", Type: types.String()},
					})},
				})},
				{Name: "x_a", Type: types.String()},
				{Name: "x_b", Type: types.String()},
			}),
		},
		{
			io: "output",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String()},
				})},
				{Name: "x_a", Type: types.String()},
				{Name: "x_b", Type: types.String()},
			}),
			err: `two output pipeline schema properties would have the same column name "x_a" in the data warehouse, case-insensitively`,
		},
		{
			io: "profile",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "A", Type: types.String()},
				})},
				{Name: "x_a", Type: types.String()},
				{Name: "x_b", Type: types.String()},
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
