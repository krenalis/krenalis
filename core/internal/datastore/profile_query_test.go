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

// profileQueryRows supplies a fixed number of rows, tracking reads and closure.
type profileQueryRows struct {
	remaining int
	scanned   int
	closed    bool
}

// Close records closure of the result set.
func (r *profileQueryRows) Close() error {
	r.closed = true
	return nil
}

// Err reports successful iteration.
func (r *profileQueryRows) Err() error { return nil }

// Next advances through the configured rows.
func (r *profileQueryRows) Next() bool {
	r.remaining--
	return r.remaining >= 0
}

// Scan supplies a fixed column value.
func (r *profileQueryRows) Scan(dest ...any) error {
	r.scanned++
	dest[0] = "email@example.com"
	return nil
}

// profileQueryWarehouse returns configured rows and count and records the query.
type profileQueryWarehouse struct {
	warehouses.Warehouse
	rows  *profileQueryRows
	total int
	query *warehouses.RowQuery
}

// Query returns the configured observation.
func (w profileQueryWarehouse) Query(_ context.Context, query warehouses.RowQuery, _ bool) (warehouses.Rows, int, error) {
	if w.query != nil {
		*w.query = query
	}
	return w.rows, w.total, nil
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
		{name: "excess limit", query: Query{Limit: 1001}, schema: schema},
		{name: "minimum bounds", query: Query{First: 0, Limit: 1}, schema: schema, valid: true},
		{name: "maximum bounds", query: Query{First: math.MaxInt32, Limit: 1000}, schema: schema, valid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &Store{}
			if tc.valid {
				store.mc = newModeCoordinator(state.Maintenance)
			}
			_, _, _, err := store.Profiles(t.Context(), tc.query, tc.schema)
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

// TestProfileQueryCount verifies that the estimated total and returned rows
// remain independent observations.
func TestProfileQueryCount(t *testing.T) {
	rows := &profileQueryRows{remaining: 2}
	store := &Store{}
	store.wh.Store(warehouses.Warehouse(profileQueryWarehouse{rows: rows, total: 0}))
	records, total, err := store.query(t.Context(), Query{
		table: "profiles", Properties: []string{"email"}, Limit: 2,
	}, map[string]warehouses.Column{"email": {Name: "email", Type: types.String()}}, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !rows.closed || rows.scanned != 2 {
		t.Fatalf("expected closed true and scanned 2, got closed %t and scanned %d", rows.closed, rows.scanned)
	}
	if total != 0 || len(records) != 2 {
		t.Fatalf("expected total 0 and rows 2, got total %d and rows %d", total, len(records))
	}
}

// TestProfileQueryOrdering checks the ordering sent to the data warehouse.
func TestProfileQueryOrdering(t *testing.T) {
	for _, tc := range []struct {
		name      string
		table     string
		orderBy   []string
		orderDesc bool
		want      []string
		wantError bool
	}{
		{name: "profiles ascending", table: "profiles", orderBy: []string{"email", "_kpid"}, want: []string{"email", "_kpid"}},
		{name: "profiles descending", table: "profiles", orderBy: []string{"email", "_kpid"}, orderDesc: true, want: []string{"email", "_kpid"}},
		{name: "profiles ordered by email", table: "profiles", orderBy: []string{"email"}, want: []string{"email"}},
		{name: "profiles ordered by kpid", table: "profiles", orderBy: []string{"_kpid"}, want: []string{"_kpid"}},
		{name: "profiles unordered", table: "profiles"},
		{name: "other table", table: "events", orderBy: []string{"email"}, want: []string{"email"}},
		{name: "unknown secondary property", table: "profiles", orderBy: []string{"email", "unknown"}, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var query warehouses.RowQuery
			store := &Store{}
			store.wh.Store(warehouses.Warehouse(profileQueryWarehouse{rows: &profileQueryRows{}, query: &query}))
			_, _, err := store.query(t.Context(), Query{
				table: tc.table, Properties: []string{"email"}, OrderBy: tc.orderBy, OrderDesc: tc.orderDesc, Limit: 1,
			}, map[string]warehouses.Column{
				"email": {Name: "email", Type: types.String()},
				"_kpid": {Name: "_kpid", Type: types.UUID()},
			}, true)
			if err != nil {
				if !tc.wantError {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if tc.wantError {
				t.Fatal("expected an error, got nil")
			}
			var got []string
			for _, column := range query.OrderBy {
				got = append(got, column.Name)
			}
			if !slices.Equal(got, tc.want) || query.OrderDesc != tc.orderDesc {
				t.Fatalf("expected order %v descending %t, got %v descending %t",
					tc.want, tc.orderDesc, got, query.OrderDesc)
			}
		})
	}
}
