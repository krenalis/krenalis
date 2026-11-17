// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"fmt"
	"time"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/schemas"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

// behavioralFilterQuery uses the public filter document with an internal result
// request. T and pagination belong to the operation, not to the saved filter.
type behavioralFilterQuery struct {
	Root *Filter
	At   time.Time
	Page behavioral.Page
}

func (this *Workspace) queryProfilesWithBehavioralFilter(ctx context.Context, query behavioralFilterQuery) (*behavioral.Result, error) {
	this.core.mustBeOpen()
	var root *behavioral.Rule
	if query.Root != nil {
		nodes := 0
		prepared, err := prepareBehavioralRule(query.Root, this.workspace.ProfileSchema, 0, &nodes)
		if err != nil {
			return nil, err
		}
		root = &prepared
	}
	return this.store.QueryProfiles(ctx, root, behavioral.Bindings{}, query.At, query.Page)
}

// prepareBehavioralCount reuses the schema-aware occurrence preparer. With
// *behavioral.UnsupportedError it returns Scope and Test in a partial Count
// without qualification, so callers can validate remaining parameters. That
// partial rule must not be executed.
func prepareBehavioralCount(scope behavioral.Scope, filter *Filter, test behavioral.CountTest) (behavioral.Rule, error) {
	var where *behavioral.OccurrencePredicate
	if filter != nil {
		validation := filterValidation{maxDepth: 2, properties: schemas.Event.Properties(), source: &scope}
		var err error
		where, err = prepareOccurrencePredicate(filter, &validation, 1,
			FilterCountSource{Type: scope.Type, Name: scope.Event, Grain: scope.Grain}, nil, occurrenceFilterRule)
		if err != nil {
			if _, unsupported := errors.AsType[*behavioral.UnsupportedError](err); !unsupported {
				return behavioral.Rule{}, err
			}
			return behavioral.Rule{Count: &behavioral.Count{Scope: scope, Test: test}}, err
		}
	}
	return behavioral.Rule{Count: &behavioral.Count{Scope: scope, Where: where, Test: test}}, nil
}

func prepareBehavioralRule(rule FilterRule, schema types.Type, depth int, nodes *int) (behavioral.Rule, error) {
	var calendar *behavioral.Calendar
	if filter, ok := rule.(*Filter); ok && filter != nil && depth == 0 && filter.DefaultCalendar != nil {
		var err error
		calendar, err = behavioral.ResolveCalendar(filter.DefaultCalendar.Definition,
			filter.DefaultCalendar.MeaningRevision)
		if err != nil {
			return behavioral.Rule{}, err
		}
	}
	return prepareBehavioralRuleInCalendar(rule, schema, depth, nodes, calendar)
}

func prepareBehavioralRuleInCalendar(rule FilterRule, schema types.Type, depth int, nodes *int, calendar *behavioral.Calendar) (behavioral.Rule, error) {
	*nodes++
	if *nodes > 101 {
		return behavioral.Rule{}, fmt.Errorf("behavioral filter exceeds 100 descendant nodes")
	}
	switch rule := rule.(type) {
	case *Filter:
		if rule == nil || depth >= 4 || (rule.Operator != OpAnd && rule.Operator != OpOr) ||
			len(rule.Rules) == 0 || len(rule.Rules) > 100 {
			return behavioral.Rule{}, fmt.Errorf("invalid behavioral filter group")
		}
		if depth > 0 && rule.DefaultCalendar != nil {
			return behavioral.Rule{}, fmt.Errorf("defaultCalendar is only allowed on the root group")
		}
		result := behavioral.Rule{Operator: convertLogicalToWhere(rule.Operator),
			Children: make([]behavioral.Rule, len(rule.Rules))}
		var unavailable error
		for i, child := range rule.Rules {
			prepared, err := prepareBehavioralRuleInCalendar(child, schema, depth+1, nodes, calendar)
			if err != nil {
				if _, unsupported := errors.AsType[*behavioral.UnsupportedError](err); !unsupported {
					return behavioral.Rule{}, err
				}
				unavailable = err
				continue
			}
			result.Children[i] = prepared
		}
		if unavailable != nil {
			return behavioral.Rule{}, unavailable
		}
		return result, nil
	case *FilterCondition:
		where, _, err := prepareFilter(&Filter{Operator: OpAnd, Rules: []FilterRule{rule}}, schema)
		if err != nil {
			return behavioral.Rule{}, err
		}
		return behavioral.Rule{Record: where.Rules[0].(*state.WhereCondition)}, nil
	case *FilterCountCondition:
		return prepareFilterCount(rule, calendar)
	case *FilterSumCondition:
		return prepareFilterSum(rule, calendar)
	case *FilterAverageCondition:
		return prepareFilterAverage(rule, calendar)
	case *FilterMoneyCondition:
		return behavioral.Rule{}, errors.New("money requires an occurrence predicate")
	case *FilterOccurredInCondition:
		return behavioral.Rule{}, errors.New("occurredIn requires experience verify")
	case *FilterExperienceCondition:
		return prepareFilterExperience(rule, calendar)
	case *FilterLastVersusPreviousCondition:
		return prepareFilterLastVersusPrevious(rule, calendar)
	case *FilterLatestNCondition:
		return prepareFilterLatestN(rule, calendar)
	default:
		return behavioral.Rule{}, fmt.Errorf("unsupported behavioral filter rule %T", rule)
	}
}
