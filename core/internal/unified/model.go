// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package unified implements Record/Count/Sum/Average and selected experiences.
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

// Count is Direct(Count(scope), literal test). Where contains local Record
// and money predicates, not an alternative public filter format.
type Count struct {
	Scope    Scope
	Where    *OccurrencePredicate
	Currency *NativeCurrency
	Test     CountTest
	Minimum  uint64

	// Shared records a governed dependency this first implementation cannot
	// resolve. Such a rule is unsupported, not invalid under specification 1.2.
	Shared *SharedBoolean
}

// ValidateWhere checks the local body and its dependencies on Count currency.
// It returns *UnsupportedError for valid bodies outside the admitted subset.
func (c *Count) ValidateWhere() error {
	if c == nil {
		return fmt.Errorf("Count cannot be nil")
	}
	err := c.Scope.validateSource()
	if err != nil {
		return err
	}
	if c.Currency != nil {
		if c.Scope.eventType() != "track" {
			return fmt.Errorf("currency field is incompatible with the occurrence source")
		}
		err := c.Currency.validate(c.Scope.Event)
		if err != nil {
			return err
		}
	}
	if c.Where == nil {
		return nil
	}
	err = c.Where.validateCount(c.Scope)
	var unavailable error
	if err != nil {
		if _, unsupported := errors.AsType[*UnsupportedError](err); !unsupported {
			return err
		}
		unavailable = err
	}
	err = c.Where.validateDependencies(nil, c.Currency, countWherePredicate)
	if err != nil {
		return err
	}
	return unavailable
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

// Rule is a Record, Count, Sum, Average, Experience, LatestN or AND/OR group.
// Root nil means AllProfiles. Record contains schema-prepared operands.
// Validation is separate from canonicalization and never mutates state.Where.
type Rule struct {
	Record     *state.WhereCondition
	Count      *Count
	Sum        *Sum
	Average    *Average
	Experience *Experience
	LatestN    *LatestN
	Operator   state.WhereLogical
	Children   []Rule
}

// Scope qualifies recorded track/page/screen messages at message_id grain.
// Type defaults to track. Event is the event/name selector; an empty selector
// is allowed only for page/screen. Named page/screen exclude unnamed messages.
// Start/End define a mature [Start,End) period; Elapsed and Calendar resolve
// against the operation's single T. History selects all queryable recorded
// events before T without an asserted lower bound.
// All event timestamps are known and every event has a determined message_id
// and profile attribution in the coherent view supplied to this operation.
type Scope struct {
	Event    string
	Type     string
	Grain    string
	Start    time.Time
	End      time.Time
	Elapsed  *ElapsedPeriod
	Calendar *CalendarPeriod
	History  *History
}

// RecordedScope validates an event selector and resolves its default type.
func RecordedScope(typ, name, grain string) (Scope, error) {
	scope := Scope{Type: typ, Event: name, Grain: grain}
	err := scope.validateSource()
	if err != nil {
		return Scope{}, err
	}
	scope.Type = scope.eventType()
	return scope, nil
}

// Bounds validates and resolves the period against at without source binding.
// It returns *UnsupportedError when the valid period exceeds this subset.
func (s Scope) Bounds(at time.Time) (time.Time, time.Time, error) {
	err := validateTime(at)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return s.bounds(at)
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

func (s Scope) eventType() string {
	if s.Type == "" {
		return "track"
	}
	return s.Type
}

func (s Scope) validateSource() error {
	if s.Grain != "registered-message" {
		return fmt.Errorf("invalid recorded event grain")
	}
	typ := s.eventType()
	if typ != "track" && typ != "page" && typ != "screen" {
		return fmt.Errorf("recorded event type must be track, page or screen")
	}
	if typ == "track" && s.Event == "" || len(s.Event) > 1024 ||
		!utf8.ValidString(s.Event) || strings.ContainsRune(s.Event, 0) {
		return fmt.Errorf("invalid recorded event selector")
	}
	return nil
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
		if rule.Count != nil || rule.Sum != nil || rule.Average != nil || rule.Experience != nil || rule.LatestN != nil ||
			rule.Operator != 0 || len(rule.Children) != 0 {
			return fmt.Errorf("profile Record condition cannot contain other rules")
		}
		c := rule.Record
		if len(c.Property) == 0 || c.Operator < state.OpIs || c.Operator > state.OpDoesNotExist {
			return fmt.Errorf("invalid prepared profile Record condition")
		}
		return nil
	}
	if rule.LatestN != nil {
		if rule.Count != nil || rule.Sum != nil || rule.Average != nil || rule.Experience != nil ||
			rule.Operator != 0 || len(rule.Children) != 0 {
			return fmt.Errorf("latestN cannot contain other rules")
		}
		return rule.LatestN.validate(at)
	}
	if rule.Experience != nil {
		if rule.Count != nil || rule.Sum != nil || rule.Average != nil || rule.Operator != 0 || len(rule.Children) != 0 {
			return fmt.Errorf("experience cannot contain other rules")
		}
		return rule.Experience.validate(at)
	}
	if rule.Average != nil {
		if rule.Count != nil || rule.Sum != nil || rule.Operator != 0 || len(rule.Children) != 0 {
			return fmt.Errorf("Average cannot contain other rules")
		}
		return rule.Average.validate(at)
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
	err := c.Scope.validateSource()
	if err != nil {
		return err
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
	whereErr := c.ValidateWhere()
	if whereErr != nil {
		if _, unsupported := errors.AsType[*UnsupportedError](whereErr); !unsupported {
			return whereErr
		}
	}
	_, _, err = c.Scope.bounds(at)
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
	return whereErr
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
