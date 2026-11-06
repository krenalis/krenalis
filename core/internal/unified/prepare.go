// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"fmt"
	"math"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/warehouses"
)

// SourceBinding contains resolved source metadata and existing Record binders.
// It contains no SQL, dialect callback or backend name. Properties is the
// bound JSON column used by the currently supported NativeCurrency revisions.
// ProfileRecord binds only columns of Profiles; Record binds Occurrences.
type SourceBinding struct {
	Profiles      warehouses.ProfileSource
	ProfileRecord func(*state.Where) (warehouses.Expr, error)
	Occurrences   warehouses.OccurrenceSource
	Properties    warehouses.Column
	Record        func(*state.Where) (warehouses.Expr, error)
}

// Prepare resolves and specializes the Record and Count subset for execution.
// It returns *UnsupportedError for valid-language forms outside this subset.
// The supplied bindings already determine source meaning and evidence model;
// NativeCurrency retains its current verbatim known-JSON-String reading.
func Prepare(root *Rule, bindings Bindings, at time.Time, source SourceBinding) (warehouses.Filter, error) {
	err := validateFilter(root, at)
	if err != nil {
		return warehouses.Filter{}, err
	}
	if len(bindings.Currencies) > 100 {
		return warehouses.Filter{}, fmt.Errorf("invalid currency binding count")
	}
	for i, b := range bindings.Currencies {
		if b.Definition == "" || b.Revision == "" || len(b.Definition) > 256 || len(b.Revision) > 256 ||
			len(b.Path) < 2 || len(b.Path) > 32 || b.Path[0] != "properties" {
			return warehouses.Filter{}, fmt.Errorf("invalid resolved currency binding")
		}
		for _, other := range bindings.Currencies[:i] {
			if b.Definition == other.Definition && b.Revision == other.Revision {
				return warehouses.Filter{}, fmt.Errorf("duplicate currency binding")
			}
		}
	}
	if root == nil {
		return warehouses.Filter{}, nil
	}
	node, err := prepareRule(*root, bindings, at, source)
	if err != nil {
		return warehouses.Filter{}, err
	}
	return warehouses.Filter{Root: node}, nil
}

func acceptedCountRanges(test CountTest, minimum uint64) []warehouses.CountRange {
	lower, value := int64(minimum), int64(test.Value)
	upper := func(n int64) *int64 { return &n }
	var candidates []warehouses.CountRange
	switch test.Operator {
	case Equal:
		candidates = []warehouses.CountRange{{Lower: value, Upper: upper(value)}}
	case Different:
		if value > 0 {
			candidates = append(candidates, warehouses.CountRange{Upper: upper(value - 1)})
		}
		if value < math.MaxInt64 {
			candidates = append(candidates, warehouses.CountRange{Lower: value + 1})
		}
	case Less:
		if value > 0 {
			candidates = []warehouses.CountRange{{Upper: upper(value - 1)}}
		}
	case LessOrEqual:
		candidates = []warehouses.CountRange{{Upper: upper(value)}}
	case Greater:
		candidates = []warehouses.CountRange{{Lower: value + 1}}
	case GreaterOrEqual:
		candidates = []warehouses.CountRange{{Lower: value}}
	case Between:
		candidates = []warehouses.CountRange{{Lower: value, Upper: upper(int64(test.Upper))}}
	default:
		panic(fmt.Sprintf("core/internal/unified: unexpected Count comparison %d", test.Operator))
	}
	var accepted []warehouses.CountRange
	for _, r := range candidates {
		r.Lower = max(r.Lower, lower)
		if r.Upper == nil || r.Lower <= *r.Upper {
			accepted = append(accepted, r)
		}
	}
	return accepted
}

func prepareRule(rule Rule, bindings Bindings, at time.Time, source SourceBinding) (warehouses.FilterCondition, error) {
	if rule.Record != nil {
		if source.ProfileRecord == nil {
			return nil, fmt.Errorf("core/internal/unified: missing profile Record binder")
		}
		err := source.Profiles.Validate()
		if err != nil {
			return nil, err
		}
		expr, err := source.ProfileRecord(&state.Where{Operator: state.OpAnd,
			Rules: []state.WhereRule{rule.Record}})
		if err != nil {
			return nil, err
		}
		return &warehouses.RecordFilter{Expr: expr}, nil
	}
	if rule.Count == nil {
		var operator warehouses.LogicalOperator
		switch rule.Operator {
		case state.OpAnd:
			operator = warehouses.OpAnd
		case state.OpOr:
			operator = warehouses.OpOr
		default:
			return nil, fmt.Errorf("core/internal/unified: unexpected group operator %d", rule.Operator)
		}
		group := &warehouses.FilterGroup{Operator: operator, Children: make([]warehouses.FilterCondition, len(rule.Children))}
		var unavailable error
		for i, child := range rule.Children {
			prepared, err := prepareRule(child, bindings, at, source)
			if err != nil {
				if _, unsupported := errors.AsType[*UnsupportedError](err); !unsupported {
					return nil, err
				}
				unavailable = err
				continue
			}
			group.Children[i] = prepared
		}
		if unavailable != nil {
			return nil, unavailable
		}
		return group, nil
	}
	if source.Record == nil {
		return nil, fmt.Errorf("core/internal/unified: missing occurrence Record binder")
	}
	err := source.Occurrences.Validate()
	if err != nil {
		return nil, err
	}
	c := rule.Count
	selector, err := source.Record(&state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
		&state.WhereCondition{Property: []string{"event"}, Operator: state.OpIs, Values: []any{c.Scope.Event}},
		&state.WhereCondition{Property: []string{"type"}, Operator: state.OpIs, Values: []any{"track"}},
	}})
	if err != nil {
		return nil, err
	}
	qualifiers := []warehouses.RowQualification{&warehouses.RecordQualification{Expr: selector}}
	if c.Where != nil {
		expr, err := source.Record(c.Where)
		if err != nil {
			return nil, err
		}
		qualifiers = append(qualifiers, &warehouses.RecordQualification{Expr: expr})
	}
	if c.Currency != nil {
		var binding *CurrencyBinding
		for i, b := range bindings.Currencies {
			if b.Definition == c.Currency.Definition && b.Revision == c.Currency.Revision {
				binding = &bindings.Currencies[i]
				break
			}
		}
		if binding == nil {
			return nil, fmt.Errorf("unresolved NativeCurrency meaning revision")
		}
		if binding.Evidence != IndependentOccurrenceValues {
			return nil, &UnsupportedError{Capability: "governed evidence without independent occurrence values"}
		}
		if !warehouses.IsValidIdentifier(source.Properties.Name) || !source.Properties.Type.IsJSON() {
			return nil, fmt.Errorf("invalid resolved NativeCurrency column")
		}
		qualifiers = append(qualifiers, &warehouses.KnownTextEqual{Column: source.Properties,
			Keys: append([]string(nil), binding.Path[1:]...), Value: c.Currency.Value})
	}
	start, end, err := c.Scope.bounds(at)
	if err != nil {
		return nil, err
	}
	return &warehouses.IndependentOccurrenceCount{
		Source: source.Occurrences, Start: start, End: end,
		Qualification: &warehouses.QualificationGroup{Operator: warehouses.OpAnd, Children: qualifiers},
		Accept:        acceptedCountRanges(c.Test, c.Minimum),
	}, nil
}
