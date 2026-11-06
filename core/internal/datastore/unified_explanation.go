// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"time"

	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// ExplainUnifiedProfile evaluates every node for one determined profile.
// It returns nil for a missing profile, ErrUnifiedUnavailable for an unverified
// backend, *unified.UnsupportedError for unavailable language capabilities,
// mode errors when unavailable, and *UnavailableError on execution failure.
// It assumes a coherent source view, as does UnifiedProfiles.
func (store *Store) ExplainUnifiedProfile(ctx context.Context, root *unified.Rule, bindings unified.Bindings, at time.Time, profile string) (*unified.Explanation, error) {
	store.mustBeOpen()
	ctx, done, err := store.mc.StartOperation(ctx, normalMode|inspectionMode)
	if err != nil {
		return nil, err
	}
	defer done()
	if id, ok := types.NormalizeUUID(profile); !ok || id != profile {
		return nil, errors.New("core/internal/datastore: invalid explanation profile")
	}
	err = unified.Validate(root, at, unified.Page{Limit: 1})
	if err != nil {
		return nil, err
	}
	backend, ok := store.warehouse().(warehouses.ProfileFilterExplanationWarehouse)
	if !ok || !backend.ProfileFilterVerified() {
		return nil, ErrUnifiedUnavailable
	}
	source := unifiedSourceBinding(store.profileColumnByProperty())
	filter, err := unified.Prepare(root, bindings, at, source)
	if err != nil {
		return nil, err
	}
	err = warehouses.ValidateProfileFilter(filter, warehouses.ProfileOutcomeRequest{Population: source.Profiles, Limit: 1})
	if err != nil {
		return nil, err
	}
	nodes, err := backend.ExplainProfileFilter(ctx, filter, source.Profiles, profile)
	if err != nil {
		return nil, unavailableError(err)
	}
	if len(nodes) == 0 {
		return nil, nil
	}
	return &unified.Explanation{Filter: filter, Nodes: nodes, At: at.UTC()}, nil
}
