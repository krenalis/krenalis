// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"testing"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/warehouses"
)

type errorWarehouse struct {
	warehouses.Warehouse
	rows warehouses.Rows
	err  error
}

// AlterProfileSchema implements [warehouses.Warehouse].
func (w *errorWarehouse) AlterProfileSchema(ctx context.Context, opID string, columns []warehouses.Column, operations []warehouses.AlterOperation) error {
	return w.err
}

// Query implements [warehouses.Warehouse].
func (w *errorWarehouse) Query(ctx context.Context, query warehouses.RowQuery, withTotal bool) (warehouses.Rows, int, error) {
	return w.rows, 2, w.err
}

// QueryReadOnly implements [warehouses.Warehouse].
func (w *errorWarehouse) QueryReadOnly(ctx context.Context, query string) (warehouses.Rows, int, error) {
	return w.rows, 2, w.err
}

type errorRows struct {
	values   []any
	next     bool
	closed   bool
	scanErr  error
	rowsErr  error
	closeErr error
}

// Close implements [warehouses.Rows].
func (r *errorRows) Close() error {
	r.closed = true
	return r.closeErr
}

// Err implements [warehouses.Rows].
func (r *errorRows) Err() error {
	return r.rowsErr
}

// Next implements [warehouses.Rows].
func (r *errorRows) Next() bool {
	next := r.next
	r.next = false
	return next
}

// Scan implements [warehouses.Rows].
func (r *errorRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	copy(dest, r.values)
	return nil
}

// TestWarehouseErrors verifies that wrapping adds exactly one *WarehouseError
// and preserves the original cause, including *warehouses.OperationError.
func TestWarehouseErrors(t *testing.T) {
	tests := []struct {
		name  string
		cause error
	}{
		{name: "success"},
		{name: "generic", cause: errors.New("warehouse failed")},
		{name: "settings", cause: warehouses.SettingsErrorf("invalid settings")},
		{name: "operation", cause: warehouses.NewOperationError(errors.New("operation failed"))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dw := &warehouse{inner: &errorWarehouse{err: test.cause}}
			err := dw.AlterProfileSchema(t.Context(), "operation", nil, nil)
			assertWarehouseError(t, err, test.cause)
			if err != nil {
				if want, ok := test.cause.(*warehouses.OperationError); ok {
					got, ok := errors.AsType[*warehouses.OperationError](err)
					if !ok || got != want {
						t.Fatalf("expected original OperationError, got %v", got)
					}
				}
			}
		})
	}
}

// TestWarehouseRows verifies that both query methods preserve row values and
// counts and wrap errors from queries, scanning, iteration, and closing.
func TestWarehouseRows(t *testing.T) {
	for _, method := range []string{"Query", "QueryReadOnly"} {
		t.Run(method, func(t *testing.T) {

			innerRows := &errorRows{values: []any{"value", 7}, next: true}
			inner := &errorWarehouse{rows: innerRows}
			dw := &warehouse{inner: inner}
			query := func() (warehouses.Rows, int, error) {
				if method == "QueryReadOnly" {
					return dw.QueryReadOnly(t.Context(), "SELECT 'value', 7")
				}
				return dw.Query(t.Context(), warehouses.RowQuery{}, true)
			}
			rows, count, err := query()
			if err != nil {
				t.Fatalf("expected successful query, got %v", err)
			}
			if count != 2 {
				t.Fatalf("expected count 2, got %d", count)
			}
			if rows == nil || !rows.Next() {
				t.Fatal("expected a row, got none")
			}
			values := make([]any, 2)
			err = rows.Scan(values...)
			if err != nil {
				t.Fatalf("expected successful scan, got %v", err)
			}
			if values[0] != "value" || values[1] != 7 {
				t.Fatalf("expected [value 7], got %v", values)
			}
			if rows.Next() {
				t.Fatal("expected end of rows, got another row")
			}
			assertWarehouseError(t, rows.Err(), nil)

			innerRows.rowsErr = errors.New("iteration failed")
			assertWarehouseError(t, rows.Err(), innerRows.rowsErr)
			innerRows.closeErr = errors.New("close failed")
			err = rows.Close()
			assertWarehouseError(t, err, innerRows.closeErr)
			if !innerRows.closed {
				t.Fatal("expected closed rows, got open rows")
			}

			scanRows := &errorRows{next: true, scanErr: errors.New("scan failed")}
			inner.rows = scanRows
			rows, _, err = query()
			if err != nil {
				t.Fatalf("expected successful query, got %v", err)
			}
			if rows == nil || !rows.Next() {
				t.Fatal("expected a row, got none")
			}
			defer rows.Close()
			err = rows.Scan(values...)
			assertWarehouseError(t, err, scanRows.scanErr)

			inner.rows = nil
			inner.err = &warehouses.RejectedReadOnlyQueryError{Msg: "query rejected"}
			rows, _, err = query()
			assertWarehouseError(t, err, inner.err)
			if rows != nil {
				t.Fatalf("expected nil rows on query failure, got %T", rows)
			}

		})
	}
}

// assertWarehouseError checks that err is nil if cause is nil, and otherwise is
// a *WarehouseError wrapping cause directly.
func assertWarehouseError(t *testing.T, err, cause error) {
	t.Helper()
	if err != nil {
		if cause == nil {
			t.Fatalf("expected no error, got %v", err)
		}
		wrapped, ok := err.(*WarehouseError)
		if !ok || wrapped == nil {
			t.Fatalf("expected non-nil *WarehouseError, got %#v", err)
		}
		if wrapped.Err != cause {
			t.Fatalf("expected original cause %v, got %v", cause, wrapped.Err)
		}
		unwrapped := errors.Unwrap(err)
		if unwrapped != cause {
			t.Fatalf("expected Unwrap() to return %v, got %v", cause, unwrapped)
		}
		if !errors.Is(err, cause) {
			t.Fatal("expected errors.Is(err, cause) to return true, got false")
		}
		return
	}
	if cause != nil {
		t.Fatalf("expected error wrapping %v, got nil", cause)
	}
}
