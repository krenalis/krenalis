// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// ErrUnifiedUnavailable reports a workspace backend without verified support.
var ErrUnifiedUnavailable = errors.New("Unified filtering is unavailable on this warehouse")

// UnifiedProfiles executes the internal Record/Count path against a coherent
// view of registered events and Profiles. This method assumes the publisher
// and event attribution supply that view; it does not implement that guarantee.
// It returns *unified.UnsupportedError for unimplemented capabilities, mode
// errors when the operation is unavailable (including ErrUnifiedUnavailable),
// and *UnavailableError on execution failure. Prepared-filter limits are
// checked before backend execution.
// No partial result is returned on failure.
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
		return nil, ErrUnifiedUnavailable
	}
	columns := store.profileColumnByProperty()
	source := unified.SourceBinding{
		Profiles: warehouses.ProfileSource{Table: "profiles", Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}},
		ProfileRecord: func(where *state.Where) (warehouses.Expr, error) {
			return convertWhere(where, columns)
		},
		Occurrences: warehouses.OccurrenceSource{Table: "krenalis_events", Profile: eventColumnByProperty["kpid"],
			Identity: eventColumnByProperty["messageId"], Timestamp: eventColumnByProperty["timestamp"]},
		Properties: eventColumnByProperty["properties"],
		Record:     func(where *state.Where) (warehouses.Expr, error) { return convertWhere(where, eventColumnByProperty) },
	}
	filter, err := unified.Prepare(root, bindings, at, source)
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
			return nil, errors.New("core/internal/datastore: invalid Unified ordering property")
		}
		request.OrderBy = []warehouses.Column{column}
		request.OrderDesc = page.OrderDesc
	}
	err = warehouses.ValidateProfileFilter(filter, request)
	if err != nil {
		return nil, err
	}
	result, err := backend.ProfileOutcomes(ctx, filter, request)
	if err != nil {
		return nil, unavailableError(err)
	}
	profiles := make([]unified.Profile, len(result.Profiles))
	for i, profile := range result.Profiles {
		profiles[i] = unified.Profile{ID: profile.ID, Truth: profile.Truth}
		if len(projection) > 0 {
			profiles[i].Attributes = unflat(profile.Values)
		}
	}
	return &unified.Result{Profiles: profiles, Match: result.Match, NoMatch: result.NoMatch,
		Unknown: result.Unknown, At: at.UTC()}, nil
}
