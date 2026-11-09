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

// Experience selects the first or last recorded occurrence before verifying it.
// Where contains local Record predicates; Verify also admits time and money.
// Unknown local qualifications are independent across determined identities.
type Experience struct {
	First  bool
	Scope  Scope
	Where  *state.Where
	Verify *ExperienceVerification
}

func (experience *Experience) prepare(at time.Time, source SourceBinding) (warehouses.FilterCondition, error) {
	// Reuse the recorded-source binding, period and local qualification of Count.
	population, err := prepareRule(Rule{Count: &Count{Scope: experience.Scope, Where: experience.Where,
		Test: CountTest{Operator: GreaterOrEqual, Value: 1}}}, Bindings{}, at, source)
	if err != nil {
		return nil, err
	}
	count, ok := population.(*warehouses.IndependentOccurrenceCount)
	if !ok {
		return nil, fmt.Errorf("core/internal/unified: invalid experience population binding")
	}
	verify, err := experience.Verify.prepare(at, source)
	if err != nil {
		return nil, err
	}
	return &warehouses.SelectedOccurrence{First: experience.First, Source: count.Source,
		Start: count.Start, End: count.End,
		UnboundedStart: count.UnboundedStart, Qualification: count.Qualification,
		Verify: verify}, nil
}

func (experience *Experience) validate(at time.Time) error {
	if experience == nil || experience.Verify == nil {
		return fmt.Errorf("experience requires verify")
	}
	err := experience.Scope.validateSource()
	if err != nil {
		return err
	}
	if experience.Where != nil {
		nodes := 0
		err := validateWhere(experience.Where, 1, &nodes)
		if err != nil {
			return err
		}
	}
	nodes := 0
	err = experience.Verify.validate(at, experience.Scope, 1, &nodes)
	var unavailable error
	if err != nil {
		if _, unsupported := errors.AsType[*UnsupportedError](err); !unsupported {
			return err
		}
		unavailable = err
	}
	_, _, err = experience.Scope.bounds(at)
	if err != nil {
		if _, unsupported := errors.AsType[*UnsupportedError](err); !unsupported {
			return err
		}
		unavailable = err
	}
	err = experience.Verify.validateDependencies(experience.Where)
	if err != nil {
		return err
	}
	return unavailable
}
