// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package unified implements the internal Record, Count and Sum Unified subset.
// It has no public JSON or persistence representation.
package unified

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/warehouses"
)

// Comparison identifies numeric terminals independently of Record operators.
type Comparison uint8

const (
	Equal Comparison = iota
	Different
	Less
	LessOrEqual
	Greater
	GreaterOrEqual
	Between
)

// Count is Direct(Count(scope), literal test). Where contains prepared Record
// operands, not an alternative public filter format.
type Count struct {
	Scope    Scope
	Where    *state.Where
	Currency *NativeCurrency
	Test     CountTest
	Minimum  uint64

	// Shared records a governed dependency this first implementation cannot
	// resolve. Such a rule is unsupported, not invalid under specification 1.2.
	Shared *SharedBoolean
}

// CountTest compares Count to a nonnegative integer or an inclusive interval.
type CountTest struct {
	Operator Comparison
	Value    uint64
	Upper    uint64
}

// Bindings supplies already resolved, immutable semantic definitions for this
// operation. It is not a registry or public/persisted filter document.
type Bindings struct {
	Currencies []CurrencyBinding
}

// CurrencyBinding maps one NativeCurrency meaning revision to its local field
// and evidence contract. These facts belong to binding, not a filter payload.
type CurrencyBinding struct {
	Definition string
	Revision   string
	Path       []string
	Evidence   EvidenceModel
}

// EvidenceModel is a resolved source contract, not a user unknown policy.
type EvidenceModel uint8

const (
	// IndependentOccurrenceValues guarantees an unconstrained unknown value
	// per occurrence, with both qualifying and non-qualifying completions.
	IndependentOccurrenceValues EvidenceModel = iota + 1
)

// ElapsedPeriod is the internal period [T-Older,T-Newer). Newer zero represents
// a lookback; a positive Newer excludes that recent duration. Both offsets are
// elapsed durations, independent of calendar days and daylight-saving changes.
// It has no public or persisted representation.
type ElapsedPeriod struct {
	Older time.Duration
	Newer time.Duration
}

// NativeCurrency binds a resolved occurrence-local currency definition.
// Missing, JSON null and non-text values are unknown business currency.
// The definition must guarantee independent unknown values per occurrence.
type NativeCurrency struct {
	Definition string
	Revision   string
	Value      string
}

// Page selects an internal outcome page. MatchingOnly requests segment
// membership; otherwise every profile is listed with its ternary outcome.
// Neither choice changes the filter or its totals.
type Page struct {
	Properties   []string
	OrderBy      string
	OrderDesc    bool
	First        int
	Limit        int
	MatchingOnly bool
}

// Profile is one internal profile outcome.
type Profile struct {
	ID         string
	Truth      Truth
	Attributes map[string]any
}

// Result is complete only when the operation returned no error.
type Result struct {
	Profiles []Profile
	Match    int64
	NoMatch  int64
	Unknown  int64
	At       time.Time
}

// Rule is a profile Record condition, Count, Sum or AND/OR group.
// Root nil means AllProfiles. Record contains schema-prepared operands.
// Validation is separate from canonicalization and never mutates state.Where.
type Rule struct {
	Record   *state.WhereCondition
	Count    *Count
	Sum      *Sum
	Operator state.WhereLogical
	Children []Rule
}

// Scope qualifies recorded track events by their raw name at message_id grain.
// Start/End define a mature [Start,End) period; Elapsed and Calendar resolve
// against the operation's single T. History selects all queryable recorded
// events before T without an asserted lower bound.
// All event timestamps are known and every event has a determined message_id
// and profile attribution in the coherent view supplied to this operation.
type Scope struct {
	Event    string
	Grain    string
	Start    time.Time
	End      time.Time
	Elapsed  *ElapsedPeriod
	Calendar *CalendarPeriod
	History  *History
}

// bounds validates the period and resolves it against validated T.
// A zero start denotes an unbounded period only with a resolved History.
func (s Scope) bounds(at time.Time) (time.Time, time.Time, error) {

	start, end := s.Start, s.End
	var unavailable error
	if s.Calendar != nil {
		if s.Elapsed != nil || s.History != nil || !start.IsZero() || !end.IsZero() {
			return time.Time{}, time.Time{}, fmt.Errorf("calendar period cannot have other temporal forms")
		}
		var err error
		start, end, err = s.Calendar.bounds(at)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	} else if s.Elapsed != nil {
		if s.History != nil || !start.IsZero() || !end.IsZero() {
			return time.Time{}, time.Time{}, fmt.Errorf("elapsed period cannot have explicit bounds or history")
		}
		p := s.Elapsed
		if p.Older <= 0 || p.Newer < 0 || p.Newer >= p.Older {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid elapsed observation period")
		}
		if p.Older%time.Microsecond != 0 || p.Newer%time.Microsecond != 0 {
			unavailable = &UnsupportedError{
				Capability: "elapsed duration precision greater than one microsecond"}
		}
		start, end = at.Add(-p.Older), at.Add(-p.Newer)
	} else if s.History != nil {
		if !start.IsZero() || !end.IsZero() {
			return time.Time{}, time.Time{}, fmt.Errorf("full relevant history cannot have explicit bounds")
		}
		if s.History.definition == "" {
			return time.Time{}, time.Time{}, fmt.Errorf("unresolved recorded history")
		}
		err := validateTime(at)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		return time.Time{}, at.UTC(), nil
	}

	for _, bound := range []time.Time{start, end} {
		err := validateTime(bound)
		if err != nil {
			if s.Elapsed != nil {
				return time.Time{}, time.Time{}, &UnsupportedError{
					Capability: "elapsed period outside supported timestamp range"}
			}
			if s.Calendar != nil {
				return time.Time{}, time.Time{}, &UnsupportedError{
					Capability: "calendar period outside supported timestamp range"}
			}
			if _, unsupported := errors.AsType[*UnsupportedError](err); !unsupported {
				return time.Time{}, time.Time{}, err
			}
			unavailable = err
		}
	}
	if !start.Before(end) {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid absolute observation period")
	}
	if unavailable != nil {
		return time.Time{}, time.Time{}, unavailable
	}
	if end.After(at) {
		return time.Time{}, time.Time{}, &UnsupportedError{Capability: "unmatured event windows"}
	}

	return start.UTC(), end.UTC(), nil
}

// SharedBoolean identifies a single governed fact used by multiple occurrence
// qualifiers. EqualTo distinguishes g from its typed false comparison.
type SharedBoolean struct {
	Fact    string
	Field   []string
	EqualTo bool
}

// Truth is the result of successful semantic evaluation.
type Truth = warehouses.FilterTruth

const (
	NoMatch = warehouses.FilterNoMatch
	Match   = warehouses.FilterMatch
	Unknown = warehouses.FilterUnknown
)

// UnsupportedError reports a valid-language capability outside this vertical.
// It must never be converted to a semantic UNKNOWN or an invalid-filter claim.
type UnsupportedError struct {
	Capability string
}

// Error describes the unavailable capability.
func (e *UnsupportedError) Error() string {
	return "unified capability not implemented: " + e.Capability
}

// Validate checks the implemented canonical subset and its structural limits.
// Record operands must also pass the existing schema-aware preparer and binder.
func Validate(root *Rule, at time.Time, page Page) error {
	if page.Limit < 1 || page.Limit > 1000 || page.First < 0 || int64(page.First) > math.MaxInt32 {
		return fmt.Errorf("invalid internal Unified page")
	}
	return validateFilter(root, at)
}

// validateFilter checks semantic shape independently of the result request.
func validateFilter(root *Rule, at time.Time) error {
	timeErr := validateTime(at)
	if timeErr != nil {
		if _, unsupported := errors.AsType[*UnsupportedError](timeErr); !unsupported {
			return timeErr
		}
	}
	if root == nil {
		return timeErr
	}
	nodes := 0
	err := validateRule(*root, at, 0, &nodes)
	if err != nil {
		if _, unsupported := errors.AsType[*UnsupportedError](err); !unsupported {
			return err
		}
	}
	if timeErr != nil {
		return timeErr
	}
	return err
}

func validateRule(rule Rule, at time.Time, depth int, nodes *int) error {
	*nodes++
	if *nodes > 101 {
		return fmt.Errorf("Unified root exceeds 100 descendant nodes")
	}
	if rule.Record != nil {
		if rule.Count != nil || rule.Sum != nil || rule.Operator != 0 || len(rule.Children) != 0 {
			return fmt.Errorf("profile Record condition cannot contain other rules")
		}
		c := rule.Record
		if len(c.Property) == 0 || c.Operator < state.OpIs || c.Operator > state.OpDoesNotExist {
			return fmt.Errorf("invalid prepared profile Record condition")
		}
		return nil
	}
	if rule.Sum != nil {
		if rule.Count != nil || rule.Operator != 0 || len(rule.Children) != 0 {
			return fmt.Errorf("Sum cannot contain other rules")
		}
		return rule.Sum.validate(at)
	}
	if rule.Count == nil {
		if depth >= 4 || (rule.Operator != state.OpAnd && rule.Operator != state.OpOr) || len(rule.Children) == 0 {
			return fmt.Errorf("invalid Unified boolean group")
		}
		var unavailable error
		for _, child := range rule.Children {
			err := validateRule(child, at, depth+1, nodes)
			if err != nil {
				if _, unsupported := errors.AsType[*UnsupportedError](err); !unsupported {
					return err
				}
				unavailable = err
			}
		}
		return unavailable
	}
	if len(rule.Children) != 0 || rule.Operator != 0 {
		return fmt.Errorf("Count cannot contain outer rules")
	}
	c := rule.Count
	if c.Scope.Grain != "registered-message" {
		return fmt.Errorf("invalid recorded event grain")
	}
	if len(c.Scope.Event) == 0 || len(c.Scope.Event) > 1024 ||
		!utf8.ValidString(c.Scope.Event) || strings.ContainsRune(c.Scope.Event, 0) {
		return fmt.Errorf("invalid recorded event selector")
	}
	if c.Test.Operator > Between || (c.Test.Operator == Between && c.Test.Upper < c.Test.Value) {
		return fmt.Errorf("invalid Count terminal")
	}
	if c.Minimum > 0 && ((c.Test.Operator == Equal && c.Test.Value < c.Minimum) ||
		(c.Test.Operator == Less && c.Test.Value <= c.Minimum) ||
		(c.Test.Operator == LessOrEqual && c.Test.Value < c.Minimum) ||
		(c.Test.Operator == Between && c.Test.Upper < c.Minimum)) {
		return fmt.Errorf("Count sample minimum and terminal are statically incompatible")
	}
	if c.Currency != nil {
		p := c.Currency
		if p.Definition == "" || p.Revision == "" || len(p.Definition) > 256 || len(p.Revision) > 256 ||
			len(p.Value) != 3 {
			return fmt.Errorf("invalid resolved NativeCurrency predicate")
		}
		for _, c := range p.Value {
			if c < 'A' || c > 'Z' {
				return fmt.Errorf("invalid currency literal")
			}
		}
	}
	if c.Where != nil {
		nodes := 0
		err := validateWhere(c.Where, 1, &nodes)
		if err != nil {
			return err
		}
	}
	_, _, err := c.Scope.bounds(at)
	if err != nil {
		return err
	}
	if c.Shared != nil {
		return &UnsupportedError{Capability: "shared governed qualification facts"}
	}
	if c.Minimum > math.MaxInt64 || c.Test.Value > math.MaxInt64 || c.Test.Upper > math.MaxInt64 {
		return &UnsupportedError{Capability: "Count bounds beyond signed 64-bit range"}
	}
	if c.Test.Operator == Greater && c.Test.Value == math.MaxInt64 {
		return &UnsupportedError{Capability: "Count threshold requiring values beyond signed 64-bit range"}
	}
	if c.Currency != nil && c.Where != nil {
		return &UnsupportedError{Capability: "mixed governed and raw occurrence predicates"}
	}
	return nil
}

func validateTime(t time.Time) error {
	t = t.UTC()
	if t.IsZero() || t.Year() < 1 || t.Year() > 9999 {
		return fmt.Errorf("invalid Unified timestamp")
	}
	if t.Nanosecond()%1000 != 0 {
		return &UnsupportedError{Capability: "timestamp precision greater than one microsecond"}
	}
	return nil
}

func validateWhere(where *state.Where, depth int, nodes *int) error {
	if where == nil || depth > 2 || len(where.Rules) == 0 ||
		(where.Operator != state.OpAnd && where.Operator != state.OpOr) {
		return fmt.Errorf("invalid occurrence Record group")
	}
	for _, rule := range where.Rules {
		*nodes++
		if *nodes > 100 {
			return fmt.Errorf("occurrence Record node budget exceeded")
		}
		switch rule := rule.(type) {
		case *state.Where:
			err := validateWhere(rule, depth+1, nodes)
			if err != nil {
				return err
			}
		case *state.WhereCondition:
			if rule == nil || len(rule.Property) == 0 || rule.Operator < state.OpIs || rule.Operator > state.OpDoesNotExist {
				return fmt.Errorf("invalid prepared Record condition")
			}
		default:
			return fmt.Errorf("invalid prepared Record node")
		}
	}
	return nil
}
