// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import "time"

// SelectedOccurrence selects First, Last or Nth before verification.
// Nth selects position 1 to 10 from the start of the base, without First.
// Otherwise First chooses the earliest occurrence; the zero value chooses Last.
// Qualifications are occurrence-local outcomes, independent across identities.
// An unknown qualifier makes that identity optional. NULL Record outcomes alone
// do not declare a shared governed fact.
// Copies must agree on attribution and period membership. Identities not
// certainly excluded also require equal timestamps, qualification and relevant
// verification. Identity and timestamps are determined, with no ID tie-break.
// Verification is conditional on selection: a direct KnownTextEqual conjunct
// of an AND Qualification fixes the same typed reading in a MoneyQualification,
// even when its stored value is unknown. Other qualification forms do not.
type SelectedOccurrence struct {
	First          bool
	Nth            int
	Source         OccurrenceSource
	Start, End     time.Time
	UnboundedStart bool
	Qualification  RowQualification
	Verify         RowQualification
}

func (*SelectedOccurrence) filterCondition() {}

// OccurrenceSelection describes the possible winners of a selection.
// AbsencePossible includes shortages for Nth and certain absence, represented
// by zero Candidates.
// One candidate is unambiguous only when absence is impossible.
type OccurrenceSelection struct {
	Candidates      int64
	AbsencePossible bool
}
