// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/warehouses"
)

// UnifiedProfiles executes the internal Count vertical against one coherent
// view of registered events and Profiles. This method assumes the publisher
// and event attribution supply that view; it does not implement that guarantee.
// It returns *unified.UnsupportedError for unimplemented capabilities, mode
// errors when the operation is unavailable, and *UnavailableError on execution
// failure. No partial result is returned on failure.
func (store *Store) UnifiedProfiles(ctx context.Context, root *unified.Rule, bindings unified.Bindings, at time.Time, page unified.Page) (*unified.Result, error) {
	store.mustBeOpen()
	ctx, done, err := store.mc.StartOperation(ctx, normalMode|inspectionMode)
	if err != nil {
		return nil, err
	}
	defer done()
	err = unified.Validate(root, at, page)
	if err != nil {
		return nil, err
	}
	backend, ok := store.warehouse().(warehouses.CompiledQueryWarehouse)
	if !ok || !backend.CompiledQueryDialect().CountVerified {
		return nil, &unified.UnsupportedError{Capability: "Unified Count on this warehouse has not been verified"}
	}
	query, err := unified.Compile(root, bindings, at, page, backend.CompiledQueryDialect(), func(where *state.Where) (warehouses.Expr, error) {
		return convertWhere(where, eventColumnByProperty)
	})
	if err != nil {
		return nil, err
	}
	rows, err := backend.QueryCompiled(ctx, query)
	if err != nil {
		return nil, unavailableError(err)
	}
	result, err := unified.Read(ctx, rows, at, page)
	if err != nil {
		return nil, unavailableError(err)
	}
	return result, nil
}
