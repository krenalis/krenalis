// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/types"
)

// IndependentOccurrenceSum sums independent, row-local monetary contributions.
// Qualification is closed before reading the value. A certainly excluded row
// contributes zero even if its value is unknown. Optional known contributions
// have reachable extrema min(0,value), max(0,value). Relevant unknown values
// have no sign or finite domain attestation, making both bounds unbounded.
// Copies of an identity must agree on attribution, period, qualification and
// the relevant value reading. Conflicts and overflow fail the whole operation.
// Threshold is a core-normalized inclusive test on the Decimal(18,4) grid:
// at least Threshold when AtLeast, otherwise at most Threshold.
type IndependentOccurrenceSum struct {
	Source         OccurrenceSource
	Start, End     time.Time
	UnboundedStart bool
	Qualification  RowQualification
	Value          DecimalReading
	Threshold      decimal.Decimal
	AtLeast        bool
}

func (*IndependentOccurrenceSum) filterCondition() {}

func (sum *IndependentOccurrenceSum) validate() error {
	if sum == nil {
		return fmt.Errorf("warehouses: nil prepared Sum")
	}
	err := sum.Source.Validate()
	if err != nil {
		return err
	}
	bounds := []time.Time{sum.End}
	if sum.UnboundedStart {
		if !sum.Start.IsZero() {
			return fmt.Errorf("warehouses: unbounded prepared period cannot have a start")
		}
	} else {
		bounds = append(bounds, sum.Start)
	}
	for _, bound := range bounds {
		utc := bound.UTC()
		if utc.IsZero() || utc.Year() < 1 || utc.Year() > 9999 || utc.Nanosecond()%1000 != 0 {
			return fmt.Errorf("warehouses: invalid prepared timestamp")
		}
	}
	if !sum.UnboundedStart && !sum.Start.Before(sum.End) {
		return fmt.Errorf("warehouses: invalid prepared period")
	}
	_, scale := sum.Threshold.PrecisionScale()
	limit := decimal.MustParse("1e33")
	if scale > 4 || sum.Threshold.Greater(limit) || sum.Threshold.Less(limit.Neg()) {
		return fmt.Errorf("warehouses: invalid normalized Sum threshold")
	}
	return sum.Value.Validate()
}

// DecimalReading reads a JSON Number exactly, without coercion or rounding.
// Missing, null, other kinds and values outside Type are unknown readings.
type DecimalReading struct {
	Column Column
	Keys   []string
	Type   types.Type
}

// Validate checks the reading admitted by this execution subset.
func (reading DecimalReading) Validate() error {
	if !IsValidIdentifier(reading.Column.Name) || !reading.Column.Type.IsJSON() || len(reading.Keys) > 32 ||
		!reading.Type.IsDecimal() || reading.Type.Precision() != 18 || reading.Type.Scale() != 4 {
		return fmt.Errorf("warehouses: invalid exact decimal reading")
	}
	minimum, maximum := reading.Type.DecimalRange()
	expectedMin, expectedMax := decimal.Range(18, 4)
	if !minimum.Equal(expectedMin) || !maximum.Equal(expectedMax) {
		return fmt.Errorf("warehouses: unsupported exact decimal range")
	}
	for _, key := range reading.Keys {
		if len(key) > 1024 || !utf8.ValidString(key) || strings.ContainsRune(key, 0) {
			return fmt.Errorf("warehouses: invalid exact decimal key")
		}
	}
	return nil
}

// OccurrenceSumBounds contains tight finite extrema for independent occurrence
// qualification outcomes, or nil for no finite bound. Record groups keep Kleene
// logic without solving raw-field correlations. Intermediate values need not
// be reachable. These are not partial sums.
type OccurrenceSumBounds struct {
	Lower, Upper *decimal.Decimal
}
