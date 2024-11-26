//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package datastore

import (
	"context"
	"fmt"
	"time"

	"github.com/meergo/meergo"
)

// WarehouseError represents an error with the data warehouse.
type WarehouseError struct {
	Err error
}

func (err *WarehouseError) Error() string {
	return fmt.Sprintf("data warehouse: %s", err.Err)
}

// wrapWarehouseError wraps err in a *WarehouseError, if it is a generic error.
func wrapWarehouseError(err error) error {
	switch err.(type) {
	case
		nil,
		*meergo.WarehouseNonInitializableError,
		*meergo.WarehouseSettingsError:
		return err
	}
	return &WarehouseError{Err: err}
}

// registeredWarehouse returns a warehouse instance registered under the given
// name, initialized with the provided settings, and wrapped in a warehouse
// type.
//
// It panics if a warehouse with the given name does not exist.
// It returns a *meergo.WarehouseSettingsError if the settings are invalid.
func registeredWarehouse(name string, settings []byte) (warehouse, error) {
	inner, err := meergo.RegisteredWarehouse(name).New(&meergo.WarehouseConfig{Settings: settings})
	if err != nil {
		return warehouse{}, err
	}
	return warehouse{inner}, nil
}

// warehouse wraps a meergo.Warehouse, returning any error from its methods
// wrapped in a WarehouseError.
type warehouse struct {
	inner meergo.Warehouse
}

func (dw warehouse) AlterUserColumns(ctx context.Context, columns []meergo.Column, operations []meergo.AlterOperation) error {
	return wrapWarehouseError(dw.inner.AlterUserColumns(ctx, columns, operations))
}

func (dw warehouse) AlterUserColumnsQueries(ctx context.Context, columns []meergo.Column, operations []meergo.AlterOperation) ([]string, error) {
	queries, err := dw.inner.AlterUserColumnsQueries(ctx, columns, operations)
	err = wrapWarehouseError(err)
	return queries, err
}

func (dw warehouse) CanInitialize(ctx context.Context) error {
	return wrapWarehouseError(dw.inner.CanInitialize(ctx))
}

func (dw warehouse) Close() error {
	return wrapWarehouseError(dw.inner.Close())
}

func (dw warehouse) Delete(ctx context.Context, table string, where meergo.Expr) error {
	return wrapWarehouseError(dw.inner.Delete(ctx, table, where))
}

func (dw warehouse) Initialize(ctx context.Context, userColumns []meergo.Column) error {
	return wrapWarehouseError(dw.inner.Initialize(ctx, userColumns))
}

func (dw warehouse) LastIdentityResolution(ctx context.Context) (*time.Time, *time.Time, error) {
	startTime, endTime, err := dw.inner.LastIdentityResolution(ctx)
	err = wrapWarehouseError(err)
	return startTime, endTime, err
}

func (dw warehouse) Merge(ctx context.Context, table meergo.Table, rows [][]any, deleted []any) error {
	return wrapWarehouseError(dw.inner.Merge(ctx, table, rows, deleted))
}

func (dw warehouse) MergeIdentities(ctx context.Context, columns []meergo.Column, rows []map[string]any) error {
	return wrapWarehouseError(dw.inner.MergeIdentities(ctx, columns, rows))
}

func (dw warehouse) Query(ctx context.Context, query meergo.RowQuery, withCount bool) (meergo.Rows, int, error) {
	rows, count, err := dw.inner.Query(ctx, query, withCount)
	err = wrapWarehouseError(err)
	return rows, count, err
}

func (dw warehouse) ResolveIdentities(ctx context.Context, identifiers, userColumns []meergo.Column, userPrimarySources map[string]int) error {
	return wrapWarehouseError(dw.inner.ResolveIdentities(ctx, identifiers, userColumns, userPrimarySources))
}

func (dw warehouse) Repair(ctx context.Context, userColumns []meergo.Column) error {
	return wrapWarehouseError(dw.inner.Repair(ctx, userColumns))
}

func (dw warehouse) Settings() []byte {
	return dw.inner.Settings()
}

func (dw warehouse) Truncate(ctx context.Context, table string) error {
	return wrapWarehouseError(dw.inner.Truncate(ctx, table))
}
