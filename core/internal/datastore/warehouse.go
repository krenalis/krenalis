// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"fmt"

	"github.com/krenalis/krenalis/tools/errors"
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
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be:
//   - *warehouses.SettingsError if the settings are invalid
//   - *warehouses.OperationError if the operation failed and the failure was
//     recorded; subsequent calls with the same opID return that failure again
func (dw *warehouse) AlterProfileSchema(ctx context.Context, opID string, columns []warehouses.Column, operations []warehouses.AlterOperation) error {
	err := dw.inner.AlterProfileSchema(ctx, opID, columns, operations)
	return wrapWarehouseError(err)
}

// CanInitialize implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be:
//   - *warehouses.SettingsError if the settings are invalid
//   - *warehouses.NonInitializableError if the warehouse cannot be initialized
func (dw *warehouse) CanInitialize(ctx context.Context) error {
	err := dw.inner.CanInitialize(ctx)
	return wrapWarehouseError(err)
}

// CheckReadOnlyAccess implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be:
//   - *warehouses.SettingsError if the settings are invalid
//   - *warehouses.SettingsNotReadOnly if access is not read-only
func (dw *warehouse) CheckReadOnlyAccess(ctx context.Context) error {
	err := dw.inner.CheckReadOnlyAccess(ctx)
	return wrapWarehouseError(err)
}

// Close implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError.
func (dw *warehouse) Close() error {
	err := dw.inner.Close()
	return wrapWarehouseError(err)
}

// ColumnTypeDescription implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError.
func (dw *warehouse) ColumnTypeDescription(t types.Type) (string, error) {
	description, err := dw.inner.ColumnTypeDescription(t)
	return description, wrapWarehouseError(err)
}

// CountIdentities implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be *warehouses.SettingsError if the settings are invalid.
func (dw *warehouse) CountIdentities(ctx context.Context, pipelines []string) (*warehouses.IdentityCounts, error) {
	counts, err := dw.inner.CountIdentities(ctx, pipelines)
	return counts, wrapWarehouseError(err)
}

// Counts implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be *warehouses.SettingsError if the settings are invalid.
func (dw *warehouse) Counts(ctx context.Context, table string, conditions []warehouses.Expr) ([]int, error) {
	counts, err := dw.inner.Counts(ctx, table, conditions)
	return counts, wrapWarehouseError(err)
}

// Delete implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be *warehouses.SettingsError if the settings are invalid.
func (dw *warehouse) Delete(ctx context.Context, table string, where warehouses.Expr) error {
	err := dw.inner.Delete(ctx, table, where)
	return wrapWarehouseError(err)
}

// DeleteSimulatedSystemRecords deletes a system's records with writer fencing.
// It returns a *WarehouseError if the operation is unsupported or fails.
func (dw *warehouse) DeleteSimulatedSystemRecords(ctx context.Context, system string, generation int64) error {
	inner, ok := dw.inner.(simulatedSystemWarehouse)
	if !ok {
		return wrapWarehouseError(errors.New("warehouse does not support simulated system writer fencing"))
	}
	err := inner.DeleteSimulatedSystemRecords(ctx, system, generation)
	return wrapWarehouseError(err)
}

// FenceSimulatedSystem establishes a simulated system's writer authority.
// It returns a *WarehouseError if the operation is unsupported or fails.
func (dw *warehouse) FenceSimulatedSystem(ctx context.Context, system string, generation int64) error {
	inner, ok := dw.inner.(simulatedSystemWarehouse)
	if !ok {
		return wrapWarehouseError(errors.New("warehouse does not support simulated system writer fencing"))
	}
	err := inner.FenceSimulatedSystem(ctx, system, generation)
	return wrapWarehouseError(err)
}

// Initialize implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be *warehouses.SettingsError if the settings are invalid.
func (dw *warehouse) Initialize(ctx context.Context, environment warehouses.Environment, profileColumns []warehouses.Column) error {
	err := dw.inner.Initialize(ctx, environment, profileColumns)
	return wrapWarehouseError(err)
}

// Merge implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be *warehouses.SettingsError if the settings are invalid.
func (dw *warehouse) Merge(ctx context.Context, table warehouses.Table, rows [][]any, deleted []any) error {
	err := dw.inner.Merge(ctx, table, rows, deleted)
	return wrapWarehouseError(err)
}

// MergeIdentities implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be *warehouses.SettingsError if the settings are invalid.
func (dw *warehouse) MergeIdentities(ctx context.Context, columns []warehouses.Column, rows []map[string]any) error {
	err := dw.inner.MergeIdentities(ctx, columns, rows)
	return wrapWarehouseError(err)
}

// MergeSimulatedSystemRecords writes a fenced batch of simulated records.
// It returns a *WarehouseError if the operation is unsupported or fails.
func (dw *warehouse) MergeSimulatedSystemRecords(ctx context.Context, system string, generation int64, table warehouses.Table, rows [][]any) error {
	inner, ok := dw.inner.(simulatedSystemWarehouse)
	if !ok {
		return wrapWarehouseError(errors.New("warehouse does not support simulated system writer fencing"))
	}
	err := inner.MergeSimulatedSystemRecords(ctx, system, generation, table, rows)
	return wrapWarehouseError(err)
}

// PreviewAlterProfileSchema implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be *warehouses.SettingsError if the settings are invalid.
func (dw *warehouse) PreviewAlterProfileSchema(ctx context.Context, columns []warehouses.Column, operations []warehouses.AlterOperation) ([]string, error) {
	queries, err := dw.inner.PreviewAlterProfileSchema(ctx, columns, operations)
	return queries, wrapWarehouseError(err)
}

// Query implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be *warehouses.SettingsError if the settings are invalid.
func (dw *warehouse) Query(ctx context.Context, query warehouses.RowQuery, withTotal bool) (warehouses.Rows, int, error) {
	rows, total, err := dw.inner.Query(ctx, query, withTotal)
	if rows != nil {
		rows = &warehouseRows{inner: rows}
	}
	return rows, total, wrapWarehouseError(err)
}

// QueryReadOnly implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be:
//   - *warehouses.SettingsError if the settings are invalid
//   - *warehouses.RejectedReadOnlyQueryError if the query is rejected
func (dw *warehouse) QueryReadOnly(ctx context.Context, query string) (warehouses.Rows, int, error) {
	rows, columnCount, err := dw.inner.QueryReadOnly(ctx, query)
	if rows != nil {
		rows = &warehouseRows{inner: rows}
	}
	return rows, columnCount, wrapWarehouseError(err)
}

// Repair implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be *warehouses.SettingsError if the settings are invalid.
func (dw *warehouse) Repair(ctx context.Context, environment warehouses.Environment, profileColumns []warehouses.Column) error {
	err := dw.inner.Repair(ctx, environment, profileColumns)
	return wrapWarehouseError(err)
}

// ResolveIdentities implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be:
//   - *warehouses.SettingsError if the settings are invalid
//   - *warehouses.OperationError if the operation failed and the failure was
//     recorded; subsequent calls with the same opID return that failure again
func (dw *warehouse) ResolveIdentities(ctx context.Context, opID string, identifiers, profileColumns []warehouses.Column, profilePrimarySources map[string]string) (*warehouses.IdentityResolutionCounts, error) {
	counts, err := dw.inner.ResolveIdentities(ctx, opID, identifiers, profileColumns, profilePrimarySources)
	return counts, wrapWarehouseError(err)
}

// Truncate implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be *warehouses.SettingsError if the settings are invalid.
func (dw *warehouse) Truncate(ctx context.Context, table string) error {
	err := dw.inner.Truncate(ctx, table)
	return wrapWarehouseError(err)
}

// UnsetIdentityColumns implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be *warehouses.SettingsError if the settings are invalid.
func (dw *warehouse) UnsetIdentityColumns(ctx context.Context, pipeline string, columns []warehouses.Column) error {
	err := dw.inner.UnsetIdentityColumns(ctx, pipeline, columns)
	return wrapWarehouseError(err)
}

// ValidateSettings implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be *warehouses.SettingsError if the settings are invalid.
func (dw *warehouse) ValidateSettings(ctx context.Context) (json.Value, error) {
	settings, err := dw.inner.ValidateSettings(ctx)
	return settings, wrapWarehouseError(err)
}

// warehouseRows wraps errors returned while reading or closing warehouse rows.
type warehouseRows struct {
	inner warehouses.Rows
}

// Close implements [warehouses.Rows].
//
// If a data warehouse error occurs, it returns a *WarehouseError.
func (r *warehouseRows) Close() error {
	err := r.inner.Close()
	return wrapWarehouseError(err)
}

// Err implements [warehouses.Rows].
//
// If a data warehouse error occurs, it returns a *WarehouseError.
func (r *warehouseRows) Err() error {
	err := r.inner.Err()
	return wrapWarehouseError(err)
}

// Next implements [warehouses.Rows].
func (r *warehouseRows) Next() bool {
	return r.inner.Next()
}

// Scan implements [warehouses.Rows].
//
// If a data warehouse error occurs, it returns a *WarehouseError.
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
