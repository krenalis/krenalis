// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import "time"

// SelectedOccurrence selects First or Last before verifying possible winners.
// First chooses the earliest occurrence; its zero value chooses Last.
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
	Source         OccurrenceSource
	Start, End     time.Time
	UnboundedStart bool
	Qualification  RowQualification
	Verify         RowQualification
}

func (*SelectedOccurrence) filterCondition() {}

// OccurrenceSelection describes the possible winners of a selection.
// AbsencePossible includes certain absence, represented by zero Candidates.
// One candidate is unambiguous only when absence is impossible.
type OccurrenceSelection struct {
	Candidates      int64
	AbsencePossible bool
}
