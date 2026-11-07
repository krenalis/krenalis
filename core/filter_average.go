// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"

	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

// FilterAverage shares the monetary subject fields and strict codec of Sum.
// Its condition requires Minimum; omission is invalid, as is zero.
// NativeCurrency restricts the population without converting currencies.
type FilterAverage = FilterSum

// FilterAverageCondition compares a monetary average without division.
// It supports the four ordered comparisons and exact thresholds at scale <=4.
type FilterAverageCondition struct {
	Average  FilterAverage  `json:"average"`
	Operator FilterOperator `json:"operator"`
	Values   []json.Value   `json:"values"`
}

// MarshalJSON returns the condition's JSON without semantic validation.
func (condition *FilterAverageCondition) MarshalJSON() ([]byte, error) {
	if condition == nil {
		return nil, errors.New("filter average condition cannot be nil")
	}
	type encoded FilterAverageCondition
	data, err := json.Marshal((*encoded)(condition))
	if err != nil {
		return nil, errors.New(err.Error())
	}
	return data, nil
}

// UnmarshalJSON decodes an Average using the common rule discriminator.
func (condition *FilterAverageCondition) UnmarshalJSON(data []byte) error {
	if condition == nil {
		return errors.New("core: cannot unmarshal an Average condition into nil")
	}
	rule, _, err := unmarshalFilterRule(json.NewDecoder(bytes.NewReader(data)), 1, expectAnyFilterRule, false)
	if err != nil {
		return errors.New(err.Error())
	}
	decoded, ok := rule.(*FilterAverageCondition)
	if !ok {
		return errors.New("expected an average condition")
	}
	*condition = *decoded
	return nil
}

func (*FilterAverageCondition) filterRule() {}

func prepareFilterAverage(condition *FilterAverageCondition, calendar *unified.Calendar) (unified.Rule, error) {
	if condition == nil {
		return unified.Rule{}, errors.New("Average condition cannot be nil")
	}
	if condition.Average.Minimum == nil {
		return unified.Rule{}, errors.New("Average requires a positive sample minimum")
	}
	rule, err := prepareFilterSum(&FilterSumCondition{Sum: condition.Average,
		Operator: condition.Operator, Values: condition.Values}, calendar)
	if err != nil {
		return unified.Rule{}, err
	}
	average := &unified.Average{Sum: *rule.Sum}
	err = average.ValidateThreshold()
	if err != nil {
		return unified.Rule{}, err
	}
	return unified.Rule{Average: average}, nil
}
