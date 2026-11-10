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

// LastPreviousDifference compares Last minus Previous on compatible pairs.
// Core establishes one native unit and independent memberships. Identity and
// timestamps are determined; every order of a timestamp tie remains possible.
// Copies must agree on relevant qualification, timestamp, reading value and
// readability. Unreadable amounts are unbounded unknowns, never zeros.
// Threshold is an inclusive scale-four bound; AtLeast chooses >= instead of <=.
// ReadableDifferences none/all replaces Threshold, only for readable pairs.
type LastPreviousDifference struct {
	Source              OccurrenceSource
	Start, End          time.Time
	UnboundedStart      bool
	Qualification       RowQualification
	Value               DecimalReading
	Threshold           *decimal.Decimal
	AtLeast             bool
	ReadableDifferences string
}

func (*LastPreviousDifference) filterCondition() {}

func (pair *LastPreviousDifference) validate() error {
	if pair == nil {
		return fmt.Errorf("warehouses: nil prepared lastVersusPrevious")
	}
	err := pair.Value.Validate()
	if err != nil {
		return err
	}
	if pair.ReadableDifferences != "" {
		if pair.ReadableDifferences != "none" && pair.ReadableDifferences != "all" || pair.Threshold != nil || pair.AtLeast {
			return fmt.Errorf("warehouses: invalid normalized difference domain")
		}
		return nil
	}
	if pair.Threshold == nil {
		return fmt.Errorf("warehouses: missing normalized difference test")
	}
	_, scale := pair.Threshold.PrecisionScale()
	limit := decimal.MustParse("199999999999999.9998")
	if !types.IsDecimal(*pair.Threshold) || scale > 4 || pair.Threshold.Greater(limit) ||
		pair.Threshold.Less(limit.Neg()) {
		return fmt.Errorf("warehouses: invalid normalized difference threshold")
	}
	return nil
}

// OccurrencePairSelection describes the union of possible ordered pairs.
// Unique requires determined existence and exactly one ordered identity pair.
// Candidates counts message identities, not pairs, and is zero on shortage.
type OccurrencePairSelection struct {
	Candidates      int64
	AbsencePossible bool
	Unique          bool
}
