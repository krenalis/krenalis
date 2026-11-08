// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"math"
	"slices"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// profileQueryWarehouse returns an empty result set and records the last query
// and the number of Query calls.
type profileQueryWarehouse struct {
	warehouses.Warehouse
	query warehouses.RowQuery
	calls int
}

// Query records the query and returns an empty result set.
func (w *profileQueryWarehouse) Query(_ context.Context, query warehouses.RowQuery, _ bool) (warehouses.Rows, int, error) {
	w.calls++
	w.query = query
	return &profileQueryRows{}, 0, nil
}

// profileQueryRows represents an empty result set.
type profileQueryRows struct{}

func (r *profileQueryRows) Close() error { return nil }

func (r *profileQueryRows) Err() error { return nil }

func (r *profileQueryRows) Next() bool { return false }

func (r *profileQueryRows) Scan(dest ...any) error {
	return errors.New("core/internal/datastore: no current row")
}

// TestCountIdentitiesEmptyPipelines verifies that an empty pipeline list
// returns initialized empty maps without accessing the warehouse.
func TestCountIdentitiesEmptyPipelines(t *testing.T) {

	store := &Store{mc: newModeCoordinator(state.Normal)}
	counts, err := store.CountIdentities(t.Context(), []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if counts == nil {
		t.Fatal("expected non-nil identity counts, got nil")
	}
	if counts.Anonymous == nil || counts.Recognized == nil || counts.WithoutProfile == nil {
		t.Fatalf("expected initialized identity count maps, got %#v", counts)
	}
	if len(counts.Anonymous) != 0 || len(counts.Recognized) != 0 || len(counts.WithoutProfile) != 0 {
		t.Fatalf("expected three empty identity count maps, got %#v", counts)
	}

}

// TestProfileQueryArguments verifies that invalid arguments are rejected and
// valid boundary values pass validation before starting a warehouse operation.
func TestProfileQueryArguments(t *testing.T) {
	schema := types.Object([]types.Property{{Name: "email", Type: types.String()}})
	for _, tc := range []struct {
		name   string
		query  Query
		schema types.Type
		valid  bool
	}{
		{name: "missing schema", query: Query{Limit: 1}},
		{name: "non-object schema", query: Query{Limit: 1}, schema: types.String()},
		{name: "negative offset", query: Query{First: -1, Limit: 1}, schema: schema},
		{name: "excess offset", query: Query{First: 2147483648, Limit: 1}, schema: schema},
		{name: "missing limit", schema: schema},
		{name: "negative limit", query: Query{Limit: -1}, schema: schema},
		{name: "minimum bounds", query: Query{First: 0, Limit: 1}, schema: schema, valid: true},
		{name: "maximum offset", query: Query{First: math.MaxInt32, Limit: 1}, schema: schema, valid: true},
		{name: "large limit", query: Query{Limit: math.MaxInt}, schema: schema, valid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &Store{}
			if tc.valid {
				store.mc = newModeCoordinator(state.Maintenance)
			}
			query := tc.query
			query.Properties = []string{"email"}
			_, _, err := store.Profiles(t.Context(), query, tc.schema)
			if err != nil {
				if tc.valid && !errors.Is(err, ErrMaintenanceMode) {
					t.Fatalf("expected %v, got %v", ErrMaintenanceMode, err)
				}
				return
			}
			t.Fatal("expected an error, got nil")
		})
	}
}

// TestQueryOrdering checks that properties are mapped to the correct columns
// and that queries preserve the requested sort order and direction.
func TestQueryOrdering(t *testing.T) {
	for _, tc := range []struct {
		name      string
		orderBy   []string
		orderDesc bool
		want      []string
		wantDesc  bool
		wantError string
	}{
		{name: "multiple ascending", orderBy: []string{"contact.email", "name"}, want: []string{"contact_email", "full_name"}},
		{name: "multiple descending", orderBy: []string{"contact.email", "name"}, orderDesc: true, want: []string{"contact_email", "full_name"}, wantDesc: true},
		{name: "single property", orderBy: []string{"contact.email"}, want: []string{"contact_email"}},
		{name: "unordered ignores direction", orderDesc: true},
		{name: "unknown secondary property", orderBy: []string{"contact.email", "unknown"}, wantError: "property path unknown does not exist"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			warehouse := &profileQueryWarehouse{}
			store := &Store{}
			store.wh.Store(warehouses.Warehouse(warehouse))
			_, _, err := store.query(t.Context(), Query{
				table: "profiles", Properties: []string{"name"}, OrderBy: tc.orderBy, OrderDesc: tc.orderDesc, Limit: 1,
			}, map[string]warehouses.Column{
				"contact.email": {Name: "contact_email", Type: types.String()},
				"name":          {Name: "full_name", Type: types.String()},
			}, true)
			if err != nil {
				if tc.wantError == "" {
					t.Fatalf("expected no error, got %v", err)
				}
				if err.Error() != tc.wantError {
					t.Fatalf("expected error %q, got %v", tc.wantError, err)
				}
				if warehouse.calls != 0 {
					t.Fatalf("expected no warehouse queries, got %d", warehouse.calls)
				}
				return
			}
			if tc.wantError != "" {
				t.Fatalf("expected error %q, got nil", tc.wantError)
			}
			if warehouse.calls != 1 {
				t.Fatalf("expected one warehouse query, got %d", warehouse.calls)
			}
			query := warehouse.query
			if query.Table != "profiles" {
				t.Fatalf("expected warehouse query for profiles, got %q", query.Table)
			}
			var got []string
			for _, column := range query.OrderBy {
				got = append(got, column.Name)
			}
			if !slices.Equal(got, tc.want) || query.OrderDesc != tc.wantDesc {
				t.Fatalf("expected order %v descending %t, got %v descending %t",
					tc.want, tc.wantDesc, got, query.OrderDesc)
			}
		})
	}
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
