// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"fmt"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/tools/errors"
)

// prepareOccurrencePredicate preserves a closed occurrence-local body.
// Unsupported leaves do not hide invalid siblings or invalid nested groups.
func prepareOccurrencePredicate(rule FilterRule, validation *filterValidation, depth int, source FilterCountSource, calendar *behavioral.Calendar, context filterRuleContext) (*behavioral.OccurrencePredicate, error) {
	switch rule := rule.(type) {
	case *Filter:
		if rule == nil || rule.DefaultCalendar != nil || depth > validation.maxDepth || len(rule.Rules) == 0 ||
			(rule.Operator != OpAnd && rule.Operator != OpOr) {
			if context == occurrenceFilterRule {
				return nil, errors.New("invalid occurrence qualification group")
			}
			return nil, errors.New("invalid experience verification group")
		}
		if len(rule.Rules) > maxFilterRuleCount-validation.ruleCount {
			if context == occurrenceFilterRule {
				return nil, errors.New("occurrence qualification exceeds 100 descendant nodes")
			}
			return nil, errors.New("experience verification exceeds 100 descendant nodes")
		}
		result := &behavioral.OccurrencePredicate{Operator: convertLogicalToWhere(rule.Operator),
			Children: make([]behavioral.OccurrencePredicate, len(rule.Rules))}
		var unavailable error
		for i, child := range rule.Rules {
			validation.ruleCount++
			prepared, err := prepareOccurrencePredicate(child, validation, depth+1, source, calendar, context)
			if err != nil {
				if _, unsupported := errors.AsType[*behavioral.UnsupportedError](err); !unsupported {
					return nil, err
				}
				unavailable = err
				continue
			}
			result.Children[i] = *prepared
		}
		if unavailable != nil {
			return nil, unavailable
		}
		return result, nil
	case *FilterCondition:
		if rule == nil {
			return nil, errors.New("filter condition cannot be nil")
		}
		condition, _, err := prepareFilterCondition(rule, validation)
		if err != nil {
			return nil, err
		}
		return &behavioral.OccurrencePredicate{Record: condition}, nil
	case *FilterMoneyCondition:
		scope, err := behavioral.RecordedScope(source.Type, source.Name, source.Grain)
		if err != nil {
			return nil, err
		}
		money, err := prepareFilterMoney(rule, scope)
		if err != nil {
			return nil, err
		}
		return &behavioral.OccurrencePredicate{Money: money}, nil
	case *FilterOccurredInCondition:
		if context != selectedFilterRule {
			return nil, errors.New("occurredIn requires experience verify")
		}
		if rule == nil {
			return nil, errors.New("filter occurredIn condition cannot be nil")
		}
		period, err := rule.OccurredIn.scope(source, calendar)
		if err != nil {
			return nil, err
		}
		return &behavioral.OccurrencePredicate{OccurredIn: &period}, nil
	default:
		if context == occurrenceFilterRule {
			return nil, fmt.Errorf("rule %T is not allowed in Count where", rule)
		}
		return nil, fmt.Errorf("rule %T is not allowed in experience verify", rule)
	}
}
