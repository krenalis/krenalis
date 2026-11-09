// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"fmt"
	"time"

	"github.com/krenalis/krenalis/core/internal/schemas"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

// unifiedQuery uses the public filter document with an internal result request.
// T and pagination belong to the operation, not to the saved filter.
type unifiedQuery struct {
	Root *Filter
	At   time.Time
	Page unified.Page
}

func (this *Workspace) unifiedProfiles(ctx context.Context, query unifiedQuery) (*unified.Result, error) {
	this.core.mustBeOpen()
	var root *unified.Rule
	if query.Root != nil {
		nodes := 0
		prepared, err := prepareUnifiedRule(query.Root, this.workspace.ProfileSchema, 0, &nodes)
		if err != nil {
			return nil, err
		}
		root = &prepared
	}
	return this.store.UnifiedProfiles(ctx, root, unified.Bindings{}, query.At, query.Page)
}

// prepareUnifiedCount reuses the schema-aware occurrence preparer.
// With *unified.UnsupportedError it returns Scope and Test in a partial Count
// without qualification, so callers can validate remaining parameters.
// That partial rule must not be executed.
func prepareUnifiedCount(scope unified.Scope, filter *Filter, test unified.CountTest) (unified.Rule, error) {
	var where *unified.OccurrencePredicate
	if filter != nil {
		validation := filterValidation{maxDepth: 2, properties: schemas.Event.Properties(), source: &scope}
		var err error
		where, err = prepareOccurrencePredicate(filter, &validation, 1,
			FilterCountSource{Type: scope.Type, Name: scope.Event, Grain: scope.Grain}, nil, occurrenceFilterRule)
		if err != nil {
			if _, unsupported := errors.AsType[*unified.UnsupportedError](err); !unsupported {
				return unified.Rule{}, err
			}
			return unified.Rule{Count: &unified.Count{Scope: scope, Test: test}}, err
		}
	}
	return unified.Rule{Count: &unified.Count{Scope: scope, Where: where, Test: test}}, nil
}

func prepareUnifiedRule(rule FilterRule, schema types.Type, depth int, nodes *int) (unified.Rule, error) {
	var calendar *unified.Calendar
	if filter, ok := rule.(*Filter); ok && filter != nil && depth == 0 && filter.DefaultCalendar != nil {
		var err error
		calendar, err = unified.ResolveCalendar(filter.DefaultCalendar.Definition, filter.DefaultCalendar.MeaningRevision)
		if err != nil {
			return unified.Rule{}, err
		}
	}
	return prepareUnifiedRuleInCalendar(rule, schema, depth, nodes, calendar)
}

func prepareUnifiedRuleInCalendar(rule FilterRule, schema types.Type, depth int, nodes *int, calendar *unified.Calendar) (unified.Rule, error) {
	*nodes++
	if *nodes > 101 {
		return unified.Rule{}, fmt.Errorf("Unified filter exceeds 100 descendant nodes")
	}
	switch rule := rule.(type) {
	case *Filter:
		if rule == nil || depth >= 4 || (rule.Operator != OpAnd && rule.Operator != OpOr) ||
			len(rule.Rules) == 0 || len(rule.Rules) > 100 {
			return unified.Rule{}, fmt.Errorf("invalid Unified filter group")
		}
		if depth > 0 && rule.DefaultCalendar != nil {
			return unified.Rule{}, fmt.Errorf("defaultCalendar is only allowed on the root group")
		}
		result := unified.Rule{Operator: convertLogicalToWhere(rule.Operator),
			Children: make([]unified.Rule, len(rule.Rules))}
		var unavailable error
		for i, child := range rule.Rules {
			prepared, err := prepareUnifiedRuleInCalendar(child, schema, depth+1, nodes, calendar)
			if err != nil {
				if _, unsupported := errors.AsType[*unified.UnsupportedError](err); !unsupported {
					return unified.Rule{}, err
				}
				unavailable = err
				continue
			}
			result.Children[i] = prepared
		}
		if unavailable != nil {
			return unified.Rule{}, unavailable
		}
		return result, nil
	case *FilterCondition:
		where, _, err := prepareFilter(&Filter{Operator: OpAnd, Rules: []FilterRule{rule}}, schema)
		if err != nil {
			return unified.Rule{}, err
		}
		return unified.Rule{Record: where.Rules[0].(*state.WhereCondition)}, nil
	case *FilterCountCondition:
		return prepareFilterCount(rule, calendar)
	case *FilterSumCondition:
		return prepareFilterSum(rule, calendar)
	case *FilterAverageCondition:
		return prepareFilterAverage(rule, calendar)
	case *FilterMoneyCondition:
		return unified.Rule{}, errors.New("money requires an occurrence predicate")
	case *FilterOccurredInCondition:
		return unified.Rule{}, errors.New("occurredIn requires experience verify")
	case *FilterExperienceCondition:
		return prepareFilterExperience(rule, calendar)
	case *FilterLatestNCondition:
		return prepareFilterLatestN(rule, calendar)
	default:
		return unified.Rule{}, fmt.Errorf("unsupported Unified filter rule %T", rule)
	}
}
