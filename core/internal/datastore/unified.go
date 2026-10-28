// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/types"
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
	backend, ok := store.warehouse().(warehouses.ProfileFilterWarehouse)
	if !ok || !backend.ProfileFilterVerified() {
		return nil, &unified.UnsupportedError{Capability: "Unified Count on this warehouse has not been verified"}
	}
	source := unified.SourceBinding{
		Occurrences: warehouses.OccurrenceSource{Table: "krenalis_events", Profile: eventColumnByProperty["kpid"],
			Identity: eventColumnByProperty["messageId"], Timestamp: eventColumnByProperty["timestamp"]},
		Properties: eventColumnByProperty["properties"],
		Record:     func(where *state.Where) (warehouses.Expr, error) { return convertWhere(where, eventColumnByProperty) },
	}
	filter, err := unified.Prepare(root, bindings, at, source)
	if err != nil {
		return nil, err
	}
	result, err := backend.ProfileOutcomes(ctx, filter, warehouses.ProfileOutcomeRequest{
		Population: warehouses.ProfileSource{Table: "profiles", Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}},
		First:      page.First, Limit: page.Limit, MatchingOnly: page.MatchingOnly,
	})
	if err != nil {
		return nil, unavailableError(err)
	}
	return &unified.Result{Profiles: result.Profiles, Match: result.Match, NoMatch: result.NoMatch,
		Unknown: result.Unknown, At: at.UTC()}, nil
}
