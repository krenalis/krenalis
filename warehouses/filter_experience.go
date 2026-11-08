// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import "time"

// LastOccurrence selects before verifying, preserving all compatible winners.
// Qualifications are independent local Record outcomes. An unknown qualifier
// makes that identity optional; it does not declare a latent governed fact.
// Copies must agree on attribution and period membership. Identities not
// certainly excluded also require equal timestamps, qualification and relevant
// verification. Identity and timestamps are determined, with no ID tie-break.
type LastOccurrence struct {
	Source         OccurrenceSource
	Start, End     time.Time
	UnboundedStart bool
	Qualification  RowQualification
	Verify         RowQualification
}

func (*LastOccurrence) filterCondition() {}

// OccurrenceSelection describes the possible winners of a last selection.
// AbsencePossible includes certain absence, represented by zero Candidates.
// One candidate is unambiguous only when absence is impossible.
type OccurrenceSelection struct {
	Candidates      int64
	AbsencePossible bool
}
