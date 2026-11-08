// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import "time"

// SelectedOccurrence selects First or Last before verifying all possible winners.
// First chooses the earliest occurrence; its zero value chooses Last.
// Qualifications are independent local Record outcomes. An unknown qualifier
// makes that identity optional; it does not declare a latent governed fact.
// Copies must agree on attribution and period membership. Identities not
// certainly excluded also require equal timestamps, qualification and relevant
// verification. Identity and timestamps are determined, with no ID tie-break.
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
