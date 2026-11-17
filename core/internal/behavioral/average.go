// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package behavioral

import (
	"fmt"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// Average shares Sum's monetary population and independent qualification.
// Unlike Sum it requires determined population and values, and Minimum >=1.
// The terminal compares the exact total to threshold times exact count.
type Average struct {
	Sum
}

// ValidateThreshold admits exact thresholds with at most four decimal places.
// It returns *UnsupportedError for finer thresholds, without rounding.
func (average *Average) ValidateThreshold() error {
	if average == nil {
		return fmt.Errorf("Average cannot be nil")
	}
	if !types.IsDecimal(average.Test.Value) {
		return fmt.Errorf("invalid Average threshold")
	}
	_, scale := average.Test.Value.PrecisionScale()
	if scale > 4 {
		return &UnsupportedError{Capability: "Average threshold finer than four decimal places"}
	}
	return nil
}

func (average *Average) prepare(at time.Time, source SourceBinding) (warehouses.FilterCondition, error) {

	// Reuse the monetary binding without Sum's threshold normalization.
	population := average.Sum
	population.Test = SumTest{Operator: GreaterOrEqual}
	prepared, err := population.prepare(at, source)
	if err != nil {
		return nil, err
	}
	sum, ok := prepared.(*warehouses.IndependentOccurrenceSum)
	if !ok || sum == nil {
		return nil, fmt.Errorf("core/internal/behavioral: unexpected prepared monetary population")
	}
	node := &warehouses.DeterminedOccurrenceAverage{Source: sum.Source, Start: sum.Start, End: sum.End,
		UnboundedStart: sum.UnboundedStart, Qualification: sum.Qualification, Value: sum.Value,
		Minimum: sum.Minimum, Threshold: average.Test.Value,
		AtLeast: average.Test.Operator == Greater || average.Test.Operator == GreaterOrEqual,
		Strict:  average.Test.Operator == Greater || average.Test.Operator == Less}

	// A determined average of Decimal(18,4) values is strictly inside +/-1e14.
	// This decides only its value test, not evidence adequacy or the guard.
	limit := decimal.MustParse("1e14")
	if !node.Threshold.Less(limit) || !node.Threshold.Greater(limit.Neg()) {
		result := node.AtLeast == node.Threshold.Less(decimal.Decimal{})
		node.DeterminedResult = &result
		node.Threshold = decimal.Decimal{}
	}

	return node, nil
}

func (average *Average) validate(at time.Time) error {
	if average == nil || average.Minimum == 0 {
		return fmt.Errorf("Average requires a positive sample minimum")
	}
	err := average.Sum.validate(at)
	if err != nil {
		return err
	}
	return average.ValidateThreshold()
}
