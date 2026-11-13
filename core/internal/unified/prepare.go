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
// EventType, Event and Name bind the native recorded-source identity columns.
type SourceBinding struct {
	Profiles      warehouses.ProfileSource
	ProfileRecord func(*state.Where) (warehouses.Expr, error)
	Occurrences   warehouses.OccurrenceSource
	Properties    warehouses.Column
	EventType     warehouses.Column
	Event         warehouses.Column
	Name          warehouses.Column
	Record        func(*state.Where) (warehouses.Expr, error)
}

// selection binds the recorded source identity without inheriting collation.
func (source SourceBinding) selection(scope Scope) (warehouses.RowQualification, error) {

	if !warehouses.IsValidIdentifier(source.EventType.Name) || !source.EventType.Type.IsString() {
		return nil, fmt.Errorf("core/internal/unified: invalid resolved event type column")
	}
	typ := scope.eventType()
	var children []warehouses.RowQualification
	if scope.Event != "" {
		column := source.Event
		if typ != "track" {
			column = source.Name
		}
		if !warehouses.IsValidIdentifier(column.Name) || !column.Type.IsString() {
			return nil, fmt.Errorf("core/internal/unified: invalid resolved event name column")
		}
		if typ != "track" {
			children = append(children, &warehouses.RecordQualification{
				Expr: warehouses.NewBaseExpr(column, warehouses.OpExists)})
		}
		children = append(children, &warehouses.ExactColumnText{Column: column, Value: scope.Event})
	}
	children = append(children, &warehouses.ExactColumnText{Column: source.EventType, Value: typ})

	return &warehouses.QualificationGroup{Operator: warehouses.OpAnd, Children: children}, nil
}

// Prepare specializes Record, Count, Sum, Average, First/Last/Nth, LatestN
// and LastVersusPrevious into the supported warehouse filter subset.
// It preserves every main-tree node and child order one to one.
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

func acceptedCountRanges(test CountTest) []warehouses.CountRange {
	value := int64(test.Value)
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
	return candidates
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
	if rule.Sum != nil {
		return rule.Sum.prepare(at, source)
	}
	if rule.LastVersusPrevious != nil {
		return rule.LastVersusPrevious.prepare(at, source)
	}
	if rule.LatestN != nil {
		return rule.LatestN.prepare(at, source)
	}
	if rule.Experience != nil {
		return rule.Experience.prepare(at, source)
	}
	if rule.Average != nil {
		return rule.Average.prepare(at, source)
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
	selector, err := source.selection(c.Scope)
	if err != nil {
		return nil, err
	}
	qualifiers := []warehouses.RowQualification{selector}
	if c.Where != nil {
		var qualification warehouses.RowQualification
		if where, recordOnly := c.Where.recordWhere(); recordOnly {
			expr, err := source.Record(where)
			if err != nil {
				return nil, err
			}
			qualification = &warehouses.RecordQualification{Expr: expr}
		} else {
			qualification, err = c.Where.prepare(at, source)
			if err != nil {
				return nil, err
			}
		}
		qualifiers = append(qualifiers, qualification)
	}
	if c.Currency != nil {
		var binding *CurrencyBinding
		if field := c.Currency.field; field != nil {
			binding = &CurrencyBinding{Definition: field.definition, Revision: field.revision,
				Path: field.path, Evidence: IndependentOccurrenceValues}
		} else {
			for i, b := range bindings.Currencies {
				if b.Definition == c.Currency.Definition && b.Revision == c.Currency.Revision {
					binding = &bindings.Currencies[i]
					break
				}
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
		Source: source.Occurrences, Start: start, End: end, UnboundedStart: c.Scope.History != nil,
		Qualification: &warehouses.QualificationGroup{Operator: warehouses.OpAnd, Children: qualifiers},
		Accept:        acceptedCountRanges(c.Test), Minimum: int64(c.Minimum),
	}, nil
}
