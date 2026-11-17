// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"time"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// ErrProfileProjectionTooWide reports a behavioral filter projection over its
// budget.
var ErrProfileProjectionTooWide = errors.New("profile projection exceeds the column limit")

// ErrProfileResultTooLarge reports a behavioral filter result page over its
// size budget.
var ErrProfileResultTooLarge = errors.New("profile result exceeds the size limit")

// ErrBehavioralFilteringUnavailable reports a workspace backend without
// verified support.
var ErrBehavioralFilteringUnavailable = errors.New("behavioral filtering is unavailable on this warehouse")

// QueryProfiles executes the internal Record/Count path against a coherent view
// of registered events and Profiles. This method assumes the publisher and
// event attribution supply that view; it does not implement that guarantee. It
// returns *behavioral.UnsupportedError for unimplemented capabilities, mode
// errors when the operation is unavailable (including
// ErrBehavioralFilteringUnavailable), and *WarehouseError on execution failure.
// Prepared-filter limits are checked before backend execution.
// ErrProfileProjectionTooWide and ErrProfileResultTooLarge distinguish
// request-dependent page limits. No partial result is returned on failure.
func (store *Store) QueryProfiles(ctx context.Context, root *behavioral.Rule, bindings behavioral.Bindings, at time.Time, page behavioral.Page) (*behavioral.Result, error) {
	store.mustBeOpen()
	ctx, done, err := store.mc.StartOperation(ctx, normalMode|inspectionMode)
	if err != nil {
		return nil, err
	}
	defer done()
	err = behavioral.Validate(root, at, page)
	if err != nil {
		return nil, err
	}
	backend, ok := store.warehouse().(warehouses.ProfileFilterWarehouse)
	if !ok || !backend.ProfileFilterVerified() {
		return nil, ErrBehavioralFilteringUnavailable
	}
	columns := store.profileColumnByProperty()
	source := profileFilterSourceBinding(columns)
	filter, err := behavioral.Prepare(root, bindings, at, source)
	if err != nil {
		return nil, err
	}
	request := warehouses.ProfileOutcomeRequest{
		Population: source.Profiles,
		First:      page.First, Limit: page.Limit, MatchingOnly: page.MatchingOnly,
	}
	projection, unflat := columnsFromProperties(page.Properties, columns, true)
	request.Columns = projection
	if page.OrderBy != "" {
		column, ok := columns[page.OrderBy]
		if !ok {
			return nil, errors.New("core/internal/datastore: invalid behavioral filter ordering property")
		}
		request.OrderBy = []warehouses.Column{column}
		request.OrderDesc = page.OrderDesc
	}
	err = warehouses.ValidateProfileFilter(filter, request)
	if err != nil {
		if errors.Is(err, warehouses.ErrProfileProjectionTooWide) {
			return nil, ErrProfileProjectionTooWide
		}
		return nil, err
	}
	result, err := backend.ProfileOutcomes(ctx, filter, request)
	if err != nil {
		if errors.Is(err, warehouses.ErrProfileResultTooLarge) {
			return nil, ErrProfileResultTooLarge
		}
		return nil, err
	}
	profiles := make([]behavioral.Profile, len(result.Profiles))
	for i, profile := range result.Profiles {
		profiles[i] = behavioral.Profile{ID: profile.ID, Truth: profile.Truth}
		if len(projection) > 0 {
			profiles[i].Attributes = unflat(profile.Values)
		}
	}
	return &behavioral.Result{Profiles: profiles, Match: result.Match, NoMatch: result.NoMatch,
		Unknown: result.Unknown, At: at.UTC()}, nil
}

// profileFilterSourceBinding binds readers against one profile-column snapshot.
func profileFilterSourceBinding(columns map[string]warehouses.Column) behavioral.SourceBinding {
	return behavioral.SourceBinding{
		Profiles: warehouses.ProfileSource{Table: "profiles", Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}},
		ProfileRecord: func(where *state.Where) (warehouses.Expr, error) {
			return convertWhere(where, columns)
		},
		Occurrences: warehouses.OccurrenceSource{Table: "krenalis_events", Profile: eventColumnByProperty["kpid"],
			Identity: eventColumnByProperty["messageId"], Timestamp: eventColumnByProperty["timestamp"]},
		Properties: eventColumnByProperty["properties"],
		EventType:  eventColumnByProperty["type"],
		Event:      eventColumnByProperty["event"],
		Name:       eventColumnByProperty["name"],
		Record:     func(where *state.Where) (warehouses.Expr, error) { return convertWhere(where, eventColumnByProperty) },
	}
}
