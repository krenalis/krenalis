// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"slices"
	"testing"

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

// Scan supplies a value without allocating a payload for surplus rows.
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
// warehouse operation.
func TestProfileQueryArguments(t *testing.T) {
	schema := types.Object([]types.Property{{Name: "email", Type: types.String()}})
	for _, tc := range []struct {
		name   string
		query  Query
		schema types.Type
	}{
		{name: "missing schema", query: Query{Limit: 1}},
		{name: "non-object schema", query: Query{Limit: 1}, schema: types.String()},
		{name: "negative offset", query: Query{First: -1, Limit: 1}, schema: schema},
		{name: "excess offset", query: Query{First: 2147483648, Limit: 1}, schema: schema},
		{name: "missing limit", schema: schema},
		{name: "excess limit", query: Query{Limit: 1001}, schema: schema},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := (&Store{}).Profiles(t.Context(), tc.query, tc.schema)
			if err != nil {
				return
			}
			t.Fatal("expected an invalid argument error, got nil")
		})
	}
}

// TestProfileQueryBounds verifies safety limits without enforcing count/row
// agreement.
func TestProfileQueryBounds(t *testing.T) {
	for _, tc := range []struct {
		name      string
		total     int
		rows      int
		scanned   int
		wantError bool
	}{
		{name: "negative count", total: -1, rows: 1, wantError: true},
		{name: "excess rows", total: 3, rows: 3, scanned: 2, wantError: true},
		{name: "independent observations", total: 0, rows: 2, scanned: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows := &profileQueryRows{remaining: tc.rows}
			store := &Store{}
			store.wh.Store(warehouses.Warehouse(profileQueryWarehouse{rows: rows, total: tc.total}))
			records, total, err := store.query(t.Context(), Query{
				table: "profiles", Properties: []string{"email"}, Limit: 2,
			}, map[string]warehouses.Column{"email": {Name: "email", Type: types.String()}}, true)
			if !rows.closed || rows.scanned != tc.scanned {
				t.Fatalf("expected closed true and scanned %d, got closed %t and scanned %d",
					tc.scanned, rows.closed, rows.scanned)
			}
			if err != nil {
				if !tc.wantError {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if tc.wantError {
				t.Fatal("expected an error, got nil")
			}
			if total != tc.total || len(records) != tc.rows {
				t.Fatalf("expected total %d and rows %d, got total %d and rows %d",
					tc.total, tc.rows, total, len(records))
			}
		})
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
