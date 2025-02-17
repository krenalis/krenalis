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

// UnavailableError represents an error with the data warehouse.
type UnavailableError struct {
	Err error
}

func (err *UnavailableError) Error() string {
	return fmt.Sprintf("data warehouse: %s", err.Err)
}

// unavailableError wraps err in a *UnavailableError, if it is a generic error.
func unavailableError(err error) error {
	switch err.(type) {
	case
		nil,
		*meergo.WarehouseNonInitializableError,
		*meergo.WarehouseSettingsError:
		return err
	}
	return &UnavailableError{Err: err}
}

// getWarehouseInstance returns a warehouse instance for the warehouse type with
// the given name, initialized with the provided settings, and wrapped in a
// warehouse type.
//
// It panics if a warehouse driver with the given name does not exist.
// It returns a *meergo.WarehouseSettingsError if the settings are invalid.
func getWarehouseInstance(name string, settings []byte) (warehouse, error) {
	inner, err := meergo.RegisteredWarehouseDriver(name).New(&meergo.WarehouseConfig{Settings: settings})
	if err != nil {
		return warehouse{}, err
	}
	return warehouse{inner}, nil
}

// warehouse wraps a meergo.Warehouse, returning any error from its methods
// wrapped in a UnavailableError.
type warehouse struct {
	inner meergo.Warehouse
}

func (dw warehouse) AlterUserColumns(ctx context.Context, columns []meergo.Column, operations []meergo.AlterOperation) error {
	return unavailableError(dw.inner.AlterUserColumns(ctx, columns, operations))
}

func (dw warehouse) AlterUserColumnsQueries(ctx context.Context, columns []meergo.Column, operations []meergo.AlterOperation) ([]string, error) {
	queries, err := dw.inner.AlterUserColumnsQueries(ctx, columns, operations)
	err = unavailableError(err)
	return queries, err
}

func (dw warehouse) CanInitialize(ctx context.Context) error {
	return unavailableError(dw.inner.CanInitialize(ctx))
}

func (dw warehouse) Close() error {
	return unavailableError(dw.inner.Close())
}

func (dw warehouse) Delete(ctx context.Context, table string, where meergo.Expr) error {
	return unavailableError(dw.inner.Delete(ctx, table, where))
}

func (dw warehouse) Initialize(ctx context.Context, userColumns []meergo.Column) error {
	return unavailableError(dw.inner.Initialize(ctx, userColumns))
}

func (dw warehouse) LatestIdentityResolution(ctx context.Context) (*time.Time, *time.Time, error) {
	startTime, endTime, err := dw.inner.LatestIdentityResolution(ctx)
	err = unavailableError(err)
	return startTime, endTime, err
}

func (dw warehouse) Merge(ctx context.Context, table meergo.Table, rows [][]any, deleted []any) error {
	return unavailableError(dw.inner.Merge(ctx, table, rows, deleted))
}

func (dw warehouse) MergeIdentities(ctx context.Context, columns []meergo.Column, rows []map[string]any) error {
	return unavailableError(dw.inner.MergeIdentities(ctx, columns, rows))
}

func (dw warehouse) Query(ctx context.Context, query meergo.RowQuery, withTotal bool) (meergo.Rows, int, error) {
	rows, total, err := dw.inner.Query(ctx, query, withTotal)
	err = unavailableError(err)
	return rows, total, err
}

func (dw warehouse) ResolveIdentities(ctx context.Context, identifiers, userColumns []meergo.Column, userPrimarySources map[string]int) error {
	return unavailableError(dw.inner.ResolveIdentities(ctx, identifiers, userColumns, userPrimarySources))
}

func (dw warehouse) Repair(ctx context.Context, userColumns []meergo.Column) error {
	return unavailableError(dw.inner.Repair(ctx, userColumns))
}

func (dw warehouse) Settings() []byte {
	return dw.inner.Settings()
}

func (dw warehouse) Truncate(ctx context.Context, table string) error {
	return unavailableError(dw.inner.Truncate(ctx, table))
}

func (dw warehouse) UnsetIdentityColumns(ctx context.Context, action int, columns []meergo.Column) error {
	return unavailableError(dw.inner.UnsetIdentityColumns(ctx, action, columns))
}
