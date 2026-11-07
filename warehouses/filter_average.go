// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import (
	"fmt"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/types"
)

// DeterminedOccurrenceAverage compares an exact total with Threshold*count.
// The value test is unknown unless the nonempty population and all its values
// are determined. The positive Minimum guard composes with it in Kleene AND.
// DeterminedResult predecides only the value test for an out-of-domain literal;
// evidence adequacy and the sample guard still apply. Threshold is then zero.
type DeterminedOccurrenceAverage struct {
	Source           OccurrenceSource
	Start, End       time.Time
	UnboundedStart   bool
	Qualification    RowQualification
	Value            DecimalReading
	Threshold        decimal.Decimal
	AtLeast, Strict  bool
	DeterminedResult *bool
	Minimum          int64
}

func (*DeterminedOccurrenceAverage) filterCondition() {}

func (average *DeterminedOccurrenceAverage) validate() error {
	if average == nil || average.Minimum < 1 {
		return fmt.Errorf("warehouses: invalid prepared Average sample minimum")
	}
	population := IndependentOccurrenceSum{Source: average.Source, Start: average.Start, End: average.End,
		UnboundedStart: average.UnboundedStart, Qualification: average.Qualification,
		Value: average.Value, Threshold: average.Threshold, Minimum: average.Minimum}
	err := population.validate()
	if err != nil {
		return err
	}
	limit := decimal.MustParse("1e14")
	if !types.IsDecimal(average.Threshold) || !average.Threshold.Less(limit) || !average.Threshold.Greater(limit.Neg()) ||
		average.DeterminedResult != nil && !average.Threshold.Equal(decimal.Decimal{}) {
		return fmt.Errorf("warehouses: invalid prepared Average threshold")
	}
	return nil
}

// OccurrenceAverageValues reports exact inputs instead of a rounded quotient.
// Count is nil for an uncertain population. Sum is nil unless population and
// amounts are determined. An empty population has exact Sum=0 and Count=0.
// Certain and Possible retain the qualified occurrence count bounds.
type OccurrenceAverageValues struct {
	Sum               *decimal.Decimal
	Count             *int64
	Certain, Possible int64
}
