// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package unified implements the internal Count subset of Unified filtering.
// It has no public JSON or persistence representation.
package unified

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/warehouses"
)

// Comparison is a Count terminal, independent of the Record operator catalog.
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
	First        int
	Limit        int
	MatchingOnly bool
}

// Profile is one internal profile outcome.
type Profile = warehouses.ProfileOutcome

// Result is complete only when the operation returned no error.
type Result struct {
	Profiles []Profile
	Match    int64
	NoMatch  int64
	Unknown  int64
	At       time.Time
}

// Rule is either a Count or an outer AND/OR group. Root nil means AllProfiles.
// Validation is separate from canonicalization and never mutates state.Where.
type Rule struct {
	Count    *Count
	Operator state.WhereLogical
	Children []Rule
}

// Scope qualifies recorded events by their raw name at message_id grain.
// Start/End define a mature [Start,End) period. FullHistory is recognized but
// requires a governed history binding not implemented by this first vertical.
// All event timestamps are known and every event has a determined message_id
// and profile attribution in the coherent view supplied to this operation.
type Scope struct {
	Event       string
	Grain       string
	Start       time.Time
	End         time.Time
	FullHistory bool
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
	if err := validateTime(at); err != nil {
		return err
	}
	if root == nil {
		return nil
	}
	nodes := 0
	return validateRule(*root, at, 0, &nodes)
}

func validateRule(rule Rule, at time.Time, depth int, nodes *int) error {
	*nodes++
	if *nodes > 101 {
		return fmt.Errorf("Unified root exceeds 100 descendant nodes")
	}
	if rule.Count == nil {
		if depth >= 4 || (rule.Operator != state.OpAnd && rule.Operator != state.OpOr) || len(rule.Children) == 0 {
			return fmt.Errorf("invalid Unified boolean group")
		}
		for _, child := range rule.Children {
			err := validateRule(child, at, depth+1, nodes)
			if err != nil {
				return err
			}
		}
		return nil
	}
	if len(rule.Children) != 0 || rule.Operator != 0 {
		return fmt.Errorf("Count cannot contain outer rules")
	}
	c := rule.Count
	if c.Shared != nil {
		return &UnsupportedError{Capability: "shared governed qualification facts"}
	}
	if c.Scope.Grain != "registered-message" {
		return &UnsupportedError{Capability: "occurrence identity/grain other than registered-message"}
	}
	if len(c.Scope.Event) == 0 || len(c.Scope.Event) > 1024 ||
		!utf8.ValidString(c.Scope.Event) || strings.ContainsRune(c.Scope.Event, 0) {
		return fmt.Errorf("invalid recorded event selector")
	}
	if c.Scope.FullHistory {
		if !c.Scope.Start.IsZero() || !c.Scope.End.IsZero() {
			return fmt.Errorf("FullHistory cannot have explicit bounds")
		}
		return &UnsupportedError{Capability: "governed FullRelevantHistory binding"}
	} else {
		for _, bound := range []time.Time{c.Scope.Start, c.Scope.End} {
			err := validateTime(bound)
			if err != nil {
				return err
			}
		}
		if !c.Scope.Start.Before(c.Scope.End) {
			return fmt.Errorf("invalid absolute observation period")
		}
		if c.Scope.End.After(at) {
			return &UnsupportedError{Capability: "unmatured event windows"}
		}
	}
	if c.Minimum > math.MaxInt64 || c.Test.Value > math.MaxInt64 || c.Test.Upper > math.MaxInt64 {
		return &UnsupportedError{Capability: "Count bounds beyond signed 64-bit range"}
	}
	if c.Test.Operator > Between || (c.Test.Operator == Between && c.Test.Upper < c.Test.Value) {
		return fmt.Errorf("invalid Count terminal")
	}
	if c.Test.Operator == Greater && c.Test.Value == math.MaxInt64 {
		return &UnsupportedError{Capability: "Count threshold requiring values beyond signed 64-bit range"}
	}
	if c.Minimum > 0 && ((c.Test.Operator == Equal && c.Test.Value < c.Minimum) ||
		(c.Test.Operator == Less && c.Test.Value <= c.Minimum) ||
		(c.Test.Operator == LessOrEqual && c.Test.Value < c.Minimum) ||
		(c.Test.Operator == Between && c.Test.Upper < c.Minimum)) {
		return fmt.Errorf("Count sample minimum and terminal are statically incompatible")
	}
	if c.Currency != nil {
		p := c.Currency
		if c.Where != nil {
			return &UnsupportedError{Capability: "mixed governed and raw occurrence predicates"}
		}
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
		return validateWhere(c.Where, 1, &nodes)
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
