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

// newWarehouse wraps errors and preserves the provider's optional capabilities.
// It panics if the platform is not registered.
func newWarehouse(platform string, settings warehouses.SettingsLoader, dialWith warehouses.DialWith) warehouses.Warehouse {
	return wrapWarehouse(warehouses.Registered(platform).New(settings, dialWith))
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

// DiscoverEventSources implements [warehouses.EventDiscoveryWarehouse].
// Warehouse execution errors are returned as *WarehouseError.
func (dw *warehouse) DiscoverEventSources(ctx context.Context, request warehouses.EventSourceDiscoveryRequest) (warehouses.EventSourceDiscoveryPage, error) {
	backend, ok := dw.inner.(warehouses.EventDiscoveryWarehouse)
	if !ok {
		return warehouses.EventSourceDiscoveryPage{}, errors.New("core/internal/datastore: event discovery backend missing")
	}
	page, err := backend.DiscoverEventSources(ctx, request)
	return page, wrapWarehouseError(err)
}

// ExplainProfileFilter forwards the provider's per-profile explanation.
// Warehouse execution errors are returned as *WarehouseError.
func (dw *warehouse) ExplainProfileFilter(ctx context.Context, filter warehouses.Filter, source warehouses.ProfileSource, profile string) ([]warehouses.FilterNodeOutcome, error) {
	backend, ok := dw.inner.(warehouses.ProfileFilterExplanationWarehouse)
	if !ok {
		return nil, errors.New("core/internal/datastore: profile explanation backend missing")
	}
	outcomes, err := backend.ExplainProfileFilter(ctx, filter, source, profile)
	return outcomes, wrapWarehouseError(err)
}

// Initialize implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be *warehouses.SettingsError if the settings are invalid.
func (dw *warehouse) Initialize(ctx context.Context, profileColumns []warehouses.Column) error {
	err := dw.inner.Initialize(ctx, profileColumns)
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

// PreviewAlterProfileSchema implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be *warehouses.SettingsError if the settings are invalid.
func (dw *warehouse) PreviewAlterProfileSchema(ctx context.Context, columns []warehouses.Column, operations []warehouses.AlterOperation) ([]string, error) {
	queries, err := dw.inner.PreviewAlterProfileSchema(ctx, columns, operations)
	return queries, wrapWarehouseError(err)
}

// ProfileFilterVerified implements [warehouses.ProfileFilterWarehouse].
func (dw *warehouse) ProfileFilterVerified() bool {
	backend, ok := dw.inner.(warehouses.ProfileFilterWarehouse)
	return ok && backend.ProfileFilterVerified()
}

// ProfileOutcomes implements [warehouses.ProfileFilterWarehouse].
// Warehouse execution errors are returned as *WarehouseError.
func (dw *warehouse) ProfileOutcomes(ctx context.Context, filter warehouses.Filter, request warehouses.ProfileOutcomeRequest) (*warehouses.ProfileOutcomes, error) {
	backend, ok := dw.inner.(warehouses.ProfileFilterWarehouse)
	if !ok {
		return nil, errors.New("core/internal/datastore: profile filter backend missing")
	}
	outcomes, err := backend.ProfileOutcomes(ctx, filter, request)
	return outcomes, wrapWarehouseError(err)
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

// Repair implements [warehouses.Warehouse].
//
// If a data warehouse error occurs, it returns a *WarehouseError wrapping the
// cause, which may be *warehouses.SettingsError if the settings are invalid.
func (dw *warehouse) Repair(ctx context.Context, profileColumns []warehouses.Column) error {
	err := dw.inner.Repair(ctx, profileColumns)
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

// SampleEventFields implements [warehouses.EventDiscoveryWarehouse].
// Warehouse execution errors are returned as *WarehouseError.
func (dw *warehouse) SampleEventFields(ctx context.Context, request warehouses.EventFieldSampleRequest) (warehouses.EventFieldSample, error) {
	backend, ok := dw.inner.(warehouses.EventDiscoveryWarehouse)
	if !ok {
		return warehouses.EventFieldSample{}, errors.New("core/internal/datastore: event discovery backend missing")
	}
	sample, err := backend.SampleEventFields(ctx, request)
	return sample, wrapWarehouseError(err)
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

// wrapWarehouse preserves optional capabilities and wraps errors.
func wrapWarehouse(inner warehouses.Warehouse) warehouses.Warehouse {
	dw := &warehouse{inner: inner}
	_, profiles := dw.inner.(warehouses.ProfileFilterWarehouse)
	_, explanations := dw.inner.(warehouses.ProfileFilterExplanationWarehouse)
	_, discovery := dw.inner.(warehouses.EventDiscoveryWarehouse)
	switch {
	case explanations && discovery:
		return struct {
			warehouses.Warehouse
			warehouses.ProfileFilterExplanationWarehouse
			warehouses.EventDiscoveryWarehouse
		}{dw, dw, dw}
	case explanations:
		return struct {
			warehouses.Warehouse
			warehouses.ProfileFilterExplanationWarehouse
		}{dw, dw}
	case profiles && discovery:
		return struct {
			warehouses.Warehouse
			warehouses.ProfileFilterWarehouse
			warehouses.EventDiscoveryWarehouse
		}{dw, dw, dw}
	case profiles:
		return struct {
			warehouses.Warehouse
			warehouses.ProfileFilterWarehouse
		}{dw, dw}
	case discovery:
		return struct {
			warehouses.Warehouse
			warehouses.EventDiscoveryWarehouse
		}{dw, dw}
	default:
		return struct{ warehouses.Warehouse }{dw}
	}
}

// wrapWarehouseError wraps err in *WarehouseError if not nil.
func wrapWarehouseError(err error) error {
	if err != nil {
		return &WarehouseError{Err: err}
	}
	return nil
}
