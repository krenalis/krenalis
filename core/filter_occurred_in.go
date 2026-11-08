// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"fmt"

	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

// FilterOccurredInCondition tests the selected occurrence's timestamp.
// It is allowed only inside an experience verify, without operator or values.
type FilterOccurredInCondition struct {
	OccurredIn FilterPeriod `json:"occurredIn"`
}

// MarshalJSON returns the condition without resolving its period against T.
func (condition *FilterOccurredInCondition) MarshalJSON() ([]byte, error) {
	if condition == nil {
		return nil, errors.New("filter occurredIn condition cannot be nil")
	}
	type encoded FilterOccurredInCondition
	data, err := json.Marshal((*encoded)(condition))
	if err != nil {
		return nil, errors.New(err.Error())
	}
	return data, nil
}

// UnmarshalJSON decodes a standalone selected-occurrence time condition.
func (condition *FilterOccurredInCondition) UnmarshalJSON(data []byte) error {
	if condition == nil {
		return errors.New("core: cannot unmarshal an occurredIn condition into nil")
	}
	rule, _, err := unmarshalFilterRule(json.NewDecoder(bytes.NewReader(data)), 1, expectAnyFilterRule, selectedFilterRule)
	if err != nil {
		return errors.New(err.Error())
	}
	decoded, ok := rule.(*FilterOccurredInCondition)
	if !ok {
		return errors.New("expected an occurredIn condition")
	}
	*condition = *decoded
	return nil
}

func (*FilterOccurredInCondition) filterRule() {}

// prepareExperienceVerification preserves the closed selected-owner body.
// Unsupported leaves do not hide invalid siblings or invalid nested groups.
func prepareExperienceVerification(rule FilterRule, validation *filterValidation, depth int, source FilterCountSource, calendar *unified.Calendar) (*unified.ExperienceVerification, error) {
	switch rule := rule.(type) {
	case *Filter:
		if rule == nil || rule.DefaultCalendar != nil || depth > validation.maxDepth || len(rule.Rules) == 0 ||
			(rule.Operator != OpAnd && rule.Operator != OpOr) {
			return nil, errors.New("invalid experience verification group")
		}
		if len(rule.Rules) > maxFilterRuleCount-validation.ruleCount {
			return nil, errors.New("experience verification exceeds 100 descendant nodes")
		}
		result := &unified.ExperienceVerification{Operator: convertLogicalToWhere(rule.Operator),
			Children: make([]unified.ExperienceVerification, len(rule.Rules))}
		var unavailable error
		for i, child := range rule.Rules {
			validation.ruleCount++
			prepared, err := prepareExperienceVerification(child, validation, depth+1, source, calendar)
			if err != nil {
				if _, unsupported := errors.AsType[*unified.UnsupportedError](err); !unsupported {
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
		return &unified.ExperienceVerification{Record: condition}, nil
	case *FilterOccurredInCondition:
		if rule == nil {
			return nil, errors.New("filter occurredIn condition cannot be nil")
		}
		period, err := rule.OccurredIn.scope(source, calendar)
		if err != nil {
			return nil, err
		}
		return &unified.ExperienceVerification{OccurredIn: &period}, nil
	default:
		return nil, fmt.Errorf("rule %T is not allowed in experience verify", rule)
	}
}
