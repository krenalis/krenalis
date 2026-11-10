// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"fmt"

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// WarehouseError represents an error with the data warehouse.
type WarehouseError struct {
	Err error
}

// Error returns the warehouse error message.
func (err *WarehouseError) Error() string {
	return fmt.Sprintf("data warehouse: %s", err.Err)
}

// Unwrap returns the underlying warehouse error.
func (err *WarehouseError) Unwrap() error {
	return err.Err
}

// warehouse wraps a warehouses.Warehouse and its rows, returning their errors
// as *WarehouseError values. Argument validation is delegated to the warehouse.
type warehouse struct {
	inner warehouses.Warehouse
}

// newWarehouse creates a warehouse that wraps errors in *WarehouseError.
// It panics if the platform is not registered.
func newWarehouse(platform string, settings warehouses.SettingsLoader, dialWith warehouses.DialWith) warehouses.Warehouse {
	return &warehouse{inner: warehouses.Registered(platform).New(settings, dialWith)}
}

// AlterProfileSchema implements [warehouses.Warehouse].
func (dw *warehouse) AlterProfileSchema(ctx context.Context, opID string, columns []warehouses.Column, operations []warehouses.AlterOperation) error {
	err := dw.inner.AlterProfileSchema(ctx, opID, columns, operations)
	return wrapWarehouseError(err)
}

// CanInitialize implements [warehouses.Warehouse].
func (dw *warehouse) CanInitialize(ctx context.Context) error {
	err := dw.inner.CanInitialize(ctx)
	return wrapWarehouseError(err)
}

// CheckReadOnlyAccess implements [warehouses.Warehouse].
func (dw *warehouse) CheckReadOnlyAccess(ctx context.Context) error {
	err := dw.inner.CheckReadOnlyAccess(ctx)
	return wrapWarehouseError(err)
}

// Close implements [warehouses.Warehouse].
func (dw *warehouse) Close() error {
	err := dw.inner.Close()
	return wrapWarehouseError(err)
}

// ColumnTypeDescription implements [warehouses.Warehouse].
func (dw *warehouse) ColumnTypeDescription(t types.Type) (string, error) {
	description, err := dw.inner.ColumnTypeDescription(t)
	return description, wrapWarehouseError(err)
}

// CountIdentities implements [warehouses.Warehouse].
func (dw *warehouse) CountIdentities(ctx context.Context, pipelines []string) (*warehouses.IdentityCounts, error) {
	counts, err := dw.inner.CountIdentities(ctx, pipelines)
	return counts, wrapWarehouseError(err)
}

// Counts implements [warehouses.Warehouse].
func (dw *warehouse) Counts(ctx context.Context, table string, conditions []warehouses.Expr) ([]int, error) {
	counts, err := dw.inner.Counts(ctx, table, conditions)
	return counts, wrapWarehouseError(err)
}

// Delete implements [warehouses.Warehouse].
func (dw *warehouse) Delete(ctx context.Context, table string, where warehouses.Expr) error {
	err := dw.inner.Delete(ctx, table, where)
	return wrapWarehouseError(err)
}

// Initialize implements [warehouses.Warehouse].
func (dw *warehouse) Initialize(ctx context.Context, profileColumns []warehouses.Column) error {
	err := dw.inner.Initialize(ctx, profileColumns)
	return wrapWarehouseError(err)
}

// Merge implements [warehouses.Warehouse].
func (dw *warehouse) Merge(ctx context.Context, table warehouses.Table, rows [][]any, deleted []any) error {
	err := dw.inner.Merge(ctx, table, rows, deleted)
	return wrapWarehouseError(err)
}

// MergeIdentities implements [warehouses.Warehouse].
func (dw *warehouse) MergeIdentities(ctx context.Context, columns []warehouses.Column, rows []map[string]any) error {
	err := dw.inner.MergeIdentities(ctx, columns, rows)
	return wrapWarehouseError(err)
}

// PreviewAlterProfileSchema implements [warehouses.Warehouse].
func (dw *warehouse) PreviewAlterProfileSchema(ctx context.Context, columns []warehouses.Column, operations []warehouses.AlterOperation) ([]string, error) {
	queries, err := dw.inner.PreviewAlterProfileSchema(ctx, columns, operations)
	return queries, wrapWarehouseError(err)
}

// Query implements [warehouses.Warehouse].
func (dw *warehouse) Query(ctx context.Context, query warehouses.RowQuery, withTotal bool) (warehouses.Rows, int, error) {
	rows, total, err := dw.inner.Query(ctx, query, withTotal)
	if rows != nil {
		rows = &warehouseRows{inner: rows}
	}
	return rows, total, wrapWarehouseError(err)
}

// QueryReadOnly implements [warehouses.Warehouse].
func (dw *warehouse) QueryReadOnly(ctx context.Context, query string) (warehouses.Rows, int, error) {
	rows, columnCount, err := dw.inner.QueryReadOnly(ctx, query)
	if rows != nil {
		rows = &warehouseRows{inner: rows}
	}
	return rows, columnCount, wrapWarehouseError(err)
}

// Repair implements [warehouses.Warehouse].
func (dw *warehouse) Repair(ctx context.Context, profileColumns []warehouses.Column) error {
	err := dw.inner.Repair(ctx, profileColumns)
	return wrapWarehouseError(err)
}

// ResolveIdentities implements [warehouses.Warehouse].
func (dw *warehouse) ResolveIdentities(ctx context.Context, opID string, identifiers, profileColumns []warehouses.Column, profilePrimarySources map[string]string) (*warehouses.IdentityResolutionCounts, error) {
	counts, err := dw.inner.ResolveIdentities(ctx, opID, identifiers, profileColumns, profilePrimarySources)
	return counts, wrapWarehouseError(err)
}

// Truncate implements [warehouses.Warehouse].
func (dw *warehouse) Truncate(ctx context.Context, table string) error {
	err := dw.inner.Truncate(ctx, table)
	return wrapWarehouseError(err)
}

// UnsetIdentityColumns implements [warehouses.Warehouse].
func (dw *warehouse) UnsetIdentityColumns(ctx context.Context, pipeline string, columns []warehouses.Column) error {
	err := dw.inner.UnsetIdentityColumns(ctx, pipeline, columns)
	return wrapWarehouseError(err)
}

// ValidateSettings implements [warehouses.Warehouse].
func (dw *warehouse) ValidateSettings(ctx context.Context) (json.Value, error) {
	settings, err := dw.inner.ValidateSettings(ctx)
	return settings, wrapWarehouseError(err)
}

// warehouseRows wraps errors returned while reading or closing warehouse rows.
type warehouseRows struct {
	inner warehouses.Rows
}

// Close implements [warehouses.Rows].
func (r *warehouseRows) Close() error {
	err := r.inner.Close()
	return wrapWarehouseError(err)
}

// Err implements [warehouses.Rows].
func (r *warehouseRows) Err() error {
	err := r.inner.Err()
	return wrapWarehouseError(err)
}

// Next implements [warehouses.Rows].
func (r *warehouseRows) Next() bool {
	return r.inner.Next()
}

// Scan implements [warehouses.Rows].
func (r *warehouseRows) Scan(dest ...any) error {
	err := r.inner.Scan(dest...)
	return wrapWarehouseError(err)
}

// wrapWarehouseError wraps err in *WarehouseError if not nil.
func wrapWarehouseError(err error) error {
	if err != nil {
		return &WarehouseError{Err: err}
	}
	return nil
}
