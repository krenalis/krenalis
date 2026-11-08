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

// profileQueryRows supplies an empty result set.
type profileQueryRows struct{}

// Close closes the result set.
func (r *profileQueryRows) Close() error { return nil }

// Err reports successful iteration.
func (r *profileQueryRows) Err() error { return nil }

// Next reports that there are no rows.
func (r *profileQueryRows) Next() bool { return false }

// Scan reports that there is no current row.
func (r *profileQueryRows) Scan(dest ...any) error {
	return errors.New("core/internal/datastore: no current row")
}

// profileQueryWarehouse returns an empty result set and records the query.
type profileQueryWarehouse struct {
	warehouses.Warehouse
	query *warehouses.RowQuery
}

// Query records the query and returns an empty result set.
func (w profileQueryWarehouse) Query(_ context.Context, query warehouses.RowQuery, _ bool) (warehouses.Rows, int, error) {
	if w.query != nil {
		*w.query = query
	}
	return &profileQueryRows{}, 0, nil
}

// TestProfileQueryArguments verifies argument rejection before starting a
// warehouse operation and acceptance of valid boundary values.
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
			_, _, err := store.Profiles(t.Context(), tc.query, tc.schema)
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

// TestQueryOrdering checks property-to-column mapping, ordering precedence,
// and direction in queries sent to the data warehouse.
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
		{name: "unordered", orderDesc: true},
		{name: "unknown secondary property", orderBy: []string{"contact.email", "unknown"}, wantError: "property path unknown does not exist"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var query warehouses.RowQuery
			store := &Store{}
			store.wh.Store(warehouses.Warehouse(profileQueryWarehouse{query: &query}))
			_, _, err := store.query(t.Context(), Query{
				table: "profiles", Properties: []string{"name"}, OrderBy: tc.orderBy, OrderDesc: tc.orderDesc, Limit: 1,
			}, map[string]warehouses.Column{
				"contact.email": {Name: "contact_email", Type: types.String()},
				"name":          {Name: "full_name", Type: types.String()},
			}, true)
			if err != nil {
				if err.Error() != tc.wantError {
					t.Fatalf("expected error %q, got %v", tc.wantError, err)
				}
				if query.Table != "" {
					t.Fatal("expected no warehouse query for an unknown property, got a query")
				}
				return
			}
			if tc.wantError != "" {
				t.Fatalf("expected error %q, got nil", tc.wantError)
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
