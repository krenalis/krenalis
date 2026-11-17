// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
	"github.com/krenalis/krenalis/warehouses/postgresql"
)

func TestExplainBehavioralProfilePreparedValidation(t *testing.T) {
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(&postgresql.PostgreSQL{}))
	// An invalid physical column models a binding defect, not a warehouse outage.
	store.columnByProperty.user = map[string]warehouses.Column{
		"country": {Name: "invalid.column", Type: types.String(), Nullable: true},
	}
	rule := behavioral.Rule{Record: &state.WhereCondition{Property: []string{"country"}, Operator: state.OpExists}}
	result, err := store.ExplainProfileFilter(t.Context(), &rule, behavioral.Bindings{},
		time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), behavioralTestID(1))
	if err != nil {
		if _, unavailable := errors.AsType[*WarehouseError](err); unavailable || result != nil ||
			!strings.Contains(err.Error(), "invalid prepared Record qualification") {
			t.Fatalf("expected internal prepared-filter failure without result, got %+v, %v", result, err)
		}
		return
	}
	t.Fatal("expected internal prepared-filter failure, got success")
}
