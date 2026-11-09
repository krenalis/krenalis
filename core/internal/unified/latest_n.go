// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"fmt"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/warehouses"
)

// LatestN applies All to the compatible complete blocks of Size occurrences.
// Identity and timestamps are determined; optional membership and local values
// are independent across identities. Ties admit every order, without ID ranks.
// Predicate admits Record and one money leaf; it cannot read occurredIn.
type LatestN struct {
	Size      int
	Scope     Scope
	Where     *state.Where
	Currency  *NativeCurrency
	Predicate *OccurrencePredicate
}

func (latest *LatestN) prepare(at time.Time, source SourceBinding) (warehouses.FilterCondition, error) {
	selection, err := (&Experience{Scope: latest.Scope, Where: latest.Where,
		Currency: latest.Currency, Verify: latest.Predicate}).prepare(at, source)
	if err != nil {
		return nil, err
	}
	base, ok := selection.(*warehouses.SelectedOccurrence)
	if !ok {
		return nil, fmt.Errorf("core/internal/unified: invalid latestN population binding")
	}
	return &warehouses.LatestOccurrences{Size: latest.Size, Source: base.Source, Start: base.Start,
		End: base.End, UnboundedStart: base.UnboundedStart, Qualification: base.Qualification,
		Predicate: base.Verify}, nil
}

func (latest *LatestN) validate(at time.Time) error {
	if latest == nil || latest.Size < 1 || latest.Size > 10 || latest.Predicate == nil {
		return fmt.Errorf("latestN requires size 1 to 10 and a predicate")
	}
	// Share the complete base, dependency and body checks of a single selection.
	err := (&Experience{Scope: latest.Scope, Where: latest.Where, Currency: latest.Currency,
		Verify: latest.Predicate}).validate(at)
	var unavailable error
	if err != nil {
		if _, unsupported := errors.AsType[*UnsupportedError](err); !unsupported {
			return err
		}
		unavailable = err
	}
	// The uniform block predicate is narrower: occurredIn is not admitted.
	err = latest.Predicate.validateCount(latest.Scope)
	if err != nil {
		return err
	}
	return unavailable
}
