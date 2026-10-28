// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import (
	"context"
	"fmt"
	"time"
)

// MaxProfilePageBytes bounds variable-size values in a Unified profile page.
const MaxProfilePageBytes = 1 << 20

// Filter is a backend-independent filter prepared by core. It has no wire or
// persistence format. A nil Root matches every profile in the requested
// population.
// The execution subset contains profile Record, independent Count and AND/OR.
type Filter struct {
	Root FilterCondition
}

// FilterCondition produces one closed ternary outcome per profile.
type FilterCondition interface {
	filterCondition()
}

// FilterGroup combines closed outcomes using strong Kleene AND/OR.
type FilterGroup struct {
	Operator LogicalOperator
	Children []FilterCondition
}

func (*FilterGroup) filterCondition() {}

// RecordFilter evaluates one prepared Record condition on the population row.
// Its columns belong to ProfileOutcomeRequest.Population, not an event source.
// Expr may include the binder's physical expansion of that one condition.
type RecordFilter struct {
	Expr Expr
}

func (*RecordFilter) filterCondition() {}

// IndependentOccurrenceCount counts determined identities with independent,
// row-local qualification uncertainty in the mature period [Start,End).
// Core must establish these preconditions before constructing the node.
// For valid source data every integer between the certain and possible counts
// is reachable. Accept is the normalized set of counts satisfying the terminal
// and its sample guard. Empty Accept accepts no count.
//
// One identity denotes one recorded event. Copies must agree on attribution,
// period membership and qualification within the period. Relevant conflicts
// fail the operation; this node does not model uncertain business identity.
// Unattributed occurrences do not contribute to a profile's history.
type IndependentOccurrenceCount struct {
	Source        OccurrenceSource
	Start, End    time.Time
	Qualification RowQualification
	Accept        []CountRange
}

func (*IndependentOccurrenceCount) filterCondition() {}

// OccurrenceSource binds the fields needed to identify and attribute events.
// Identity and Timestamp must be determined; Profile may be absent for events
// not yet attributed. All qualification columns belong to this table.
type OccurrenceSource struct {
	Table     string
	Profile   Column
	Identity  Column
	Timestamp Column
}

// Validate checks the known identity and timestamp source contract.
func (s OccurrenceSource) Validate() error {
	if !IsValidIdentifier(s.Table) || !IsValidIdentifier(s.Profile.Name) || !s.Profile.Type.IsUUID() ||
		!IsValidIdentifier(s.Identity.Name) || !s.Identity.Type.IsString() || s.Identity.Nullable ||
		!IsValidIdentifier(s.Timestamp.Name) || !s.Timestamp.Type.IsDateTime() || s.Timestamp.Nullable {
		return fmt.Errorf("warehouses: invalid occurrence source")
	}
	return nil
}

// CountRange is inclusive. Upper nil means unbounded. Accepted ranges must be
// sorted, disjoint and non-adjacent, with at most two ranges. Runtime counters
// are signed 64-bit; overflow must fail the operation, not change membership.
type CountRange struct {
	Lower int64
	Upper *int64
}

// RowQualification describes a row-local ternary predicate. It cannot represent
// a shared governed fact. Record NULL retains its existing meaning and does
// not establish a latent governed value or cross-occurrence correlation.
type RowQualification interface {
	rowQualification()
}

// RecordQualification reuses prepared Record operations with literal operands.
// It does not admit cross-subject reads through Column operands.
type RecordQualification struct {
	Expr Expr
}

func (*RecordQualification) rowQualification() {}

// KnownTextEqual compares a known JSON String verbatim with a literal. Missing,
// JSON null and non-String values produce NULL. Core resolves its business
// meaning before constructing it; no normalization or domain inference occurs.
type KnownTextEqual struct {
	Column Column
	Keys   []string
	Value  string
}

func (*KnownTextEqual) rowQualification() {}

// QualificationGroup combines local qualifications using strong Kleene logic.
type QualificationGroup struct {
	Operator LogicalOperator
	Children []RowQualification
}

func (*QualificationGroup) rowQualification() {}

// ProfileSource is the requested profile population and its UUID key.
// The source must provide one logical row for each profile in a coherent view.
type ProfileSource struct {
	Table string
	Key   Column
}

// Validate checks the profile population's table and determined UUID key.
func (s ProfileSource) Validate() error {
	if !IsValidIdentifier(s.Table) || !IsValidIdentifier(s.Key.Name) || !s.Key.Type.IsUUID() || s.Key.Nullable {
		return fmt.Errorf("warehouses: invalid profile population")
	}
	return nil
}

// ProfileOutcomeRequest selects an internal outcome page, independent of the
// filter. MatchingOnly selects segment membership; all three totals always
// describe the whole population. Columns projects flat population fields;
// OrderBy admits one scalar key, NULLS LAST, then UUID ascending. Without a
// key the order is UUID ascending. Neither ordering nor page freezes later reads.
type ProfileOutcomeRequest struct {
	Population   ProfileSource
	Columns      []Column
	OrderBy      []Column
	OrderDesc    bool
	First        int
	Limit        int
	MatchingOnly bool
}

// ProfileOutcome contains one profile's semantic result and its projected
// values in request-column order.
type ProfileOutcome struct {
	ID     string
	Truth  FilterTruth
	Values []any
}

// ProfileOutcomes is complete only when the operation returns no error.
type ProfileOutcomes struct {
	Profiles []ProfileOutcome
	Match    int64
	NoMatch  int64
	Unknown  int64
}

// FilterTruth is the result of a successful semantic evaluation.
type FilterTruth uint8

const (
	FilterNoMatch FilterTruth = iota
	FilterMatch
	FilterUnknown
)

// ProfileFilterWarehouse provides the internal prepared-filter execution path.
// A new driver can implement it without changing core. A backend remains gated
// until its real-database conformance tests have passed.
type ProfileFilterWarehouse interface {
	ProfileFilterVerified() bool
	ProfileOutcomes(context.Context, Filter, ProfileOutcomeRequest) (*ProfileOutcomes, error)
}
