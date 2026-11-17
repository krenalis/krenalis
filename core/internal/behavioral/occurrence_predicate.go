// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package behavioral

import (
	"fmt"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/warehouses"
)

// OccurrencePredicate is a closed body owned by one occurrence.
// A node is a Record, timestamp, money predicate or an AND/OR group.
// Timestamp predicates are admitted only in selected-occurrence verification.
// It cannot introduce another scope or facts shared across occurrences.
type OccurrencePredicate struct {
	Record     *state.WhereCondition
	OccurredIn *Scope
	Money      *Money
	Operator   state.WhereLogical
	Children   []OccurrencePredicate
}

// RecordPredicate preserves the shape and operands of a prepared Record group.
func RecordPredicate(where *state.Where) (*OccurrencePredicate, error) {
	if where == nil {
		return nil, nil
	}
	nodes := 0
	err := validateWhere(where, 1, &nodes)
	if err != nil {
		return nil, err
	}
	return recordPredicate(where), nil
}

func (v *OccurrencePredicate) prepare(at time.Time, source SourceBinding) (warehouses.RowQualification, error) {
	if v.Record != nil {
		expr, err := source.Record(&state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{v.Record}})
		if err != nil {
			return nil, err
		}
		return &warehouses.RecordQualification{Expr: expr}, nil
	}
	if v.Money != nil {
		return v.Money.prepare(source)
	}
	if v.OccurredIn != nil {
		start, end, err := v.OccurredIn.bounds(at)
		if err != nil {
			return nil, err
		}
		// Use the same exact datetime comparisons as Record, after resolving T.
		expr, err := source.Record(&state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
			&state.WhereCondition{Property: []string{"timestamp"}, Operator: state.OpIsOnOrAfter, Values: []any{start}},
			&state.WhereCondition{Property: []string{"timestamp"}, Operator: state.OpIsBefore, Values: []any{end}},
		}})
		if err != nil {
			return nil, err
		}
		return &warehouses.RecordQualification{Expr: expr}, nil
	}
	group := &warehouses.QualificationGroup{Operator: warehouses.LogicalOperator(v.Operator),
		Children: make([]warehouses.RowQualification, len(v.Children))}
	for i := range v.Children {
		child, err := v.Children[i].prepare(at, source)
		if err != nil {
			return nil, err
		}
		group.Children[i] = child
	}
	return group, nil
}

// recordWhere reconstructs the original group when the body is entirely Record.
// The caller has validated the body; no groups are flattened or reordered.
func (v *OccurrencePredicate) recordWhere() (*state.Where, bool) {
	if v.Money != nil || v.OccurredIn != nil {
		return nil, false
	}
	if v.Record != nil {
		return &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{v.Record}}, true
	}
	where := &state.Where{Operator: v.Operator, Rules: make([]state.WhereRule, len(v.Children))}
	for i := range v.Children {
		child := &v.Children[i]
		if child.Record != nil {
			where.Rules[i] = child.Record
			continue
		}
		group, ok := child.recordWhere()
		if !ok {
			return nil, false
		}
		where.Rules[i] = group
	}
	return where, true
}

func (v *OccurrencePredicate) validate(at time.Time, scope Scope, depth int, nodes *int, context predicateContext) error {
	if v == nil {
		return fmt.Errorf("experience requires verify")
	}
	variants := 0
	for _, present := range []bool{v.Record != nil, v.OccurredIn != nil, v.Money != nil, v.Children != nil} {
		if present {
			variants++
		}
	}
	if variants != 1 {
		if context == countWherePredicate {
			return fmt.Errorf("invalid occurrence qualification")
		}
		return fmt.Errorf("invalid selected-occurrence verification")
	}
	if v.Record != nil {
		local := 0
		return validateWhere(&state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{v.Record}}, 1, &local)
	}
	if v.Money != nil {
		return v.Money.Validate(scope)
	}
	if v.OccurredIn != nil {
		if context == countWherePredicate {
			return fmt.Errorf("occurredIn requires experience verify")
		}
		_, _, err := v.OccurredIn.bounds(at)
		if err != nil {
			return err
		}
		if v.OccurredIn.History != nil {
			return &UnsupportedError{Capability: "occurredIn full relevant history"}
		}
		return nil
	}
	if depth > 2 || len(v.Children) == 0 || (v.Operator != state.OpAnd && v.Operator != state.OpOr) {
		if context == countWherePredicate {
			return fmt.Errorf("invalid occurrence qualification group")
		}
		return fmt.Errorf("invalid experience verification group")
	}
	var unavailable error
	for i := range v.Children {
		*nodes++
		if *nodes > 100 {
			if context == countWherePredicate {
				return fmt.Errorf("occurrence qualification exceeds 100 descendant nodes")
			}
			return fmt.Errorf("experience verification exceeds 100 descendant nodes")
		}
		err := v.Children[i].validate(at, scope, depth+1, nodes, context)
		if err != nil {
			if _, unsupported := errors.AsType[*UnsupportedError](err); !unsupported {
				return err
			}
			unavailable = err
		}
	}
	return unavailable
}

func (v *OccurrencePredicate) validateCount(scope Scope) error {
	nodes := 0
	return v.validate(time.Time{}, scope, 1, &nodes, countWherePredicate)
}

// validateDependencies runs only after body validation, including its budgets.
func (v *OccurrencePredicate) validateDependencies(where *state.Where, currency *NativeCurrency, context predicateContext) error {
	stack := []*OccurrencePredicate{v}
	var field *MonetaryField
	count := 0
	var records []state.WhereRule
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current.Money != nil {
			count++
			field = current.Money.Field
		}
		if current.Record != nil {
			records = append(records, current.Record)
		}
		for i := range current.Children {
			stack = append(stack, &current.Children[i])
		}
	}
	if count > 1 {
		if context == countWherePredicate {
			return &UnsupportedError{Capability: "multiple monetary predicates in Count where"}
		}
		return &UnsupportedError{Capability: "multiple monetary predicates in experience verify"}
	}
	if currency != nil && len(records) > 0 {
		err := currency.ValidateWhere(&state.Where{Operator: state.OpAnd, Rules: records})
		if err != nil {
			return err
		}
	}
	if field == nil {
		return nil
	}
	err := field.ValidateWhere(where)
	if err != nil {
		return err
	}
	if len(records) > 0 {
		return field.ValidateWhere(&state.Where{Operator: state.OpAnd, Rules: records})
	}
	return nil
}

func (v *OccurrencePredicate) validateVerify(at time.Time, scope Scope) error {
	nodes := 0
	return v.validate(at, scope, 1, &nodes, selectedOccurrencePredicate)
}

type predicateContext uint8

const (
	countWherePredicate predicateContext = iota
	selectedOccurrencePredicate
)

func recordPredicate(where *state.Where) *OccurrencePredicate {
	result := &OccurrencePredicate{Operator: where.Operator, Children: make([]OccurrencePredicate, len(where.Rules))}
	for i, rule := range where.Rules {
		switch rule := rule.(type) {
		case *state.WhereCondition:
			result.Children[i].Record = rule
		case *state.Where:
			result.Children[i] = *recordPredicate(rule)
		default:
			panic(fmt.Sprintf("core/internal/behavioral: unexpected prepared Record rule %T", rule))
		}
	}
	return result
}
