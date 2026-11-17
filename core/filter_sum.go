// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"math"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/schemas"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

// FilterSum describes a monetary sum at registered-message grain.
// NativeCurrency restricts the population; it never converts currencies.
// Where supports Record paths independent of the amount and currency reading.
// Minimum is a positive integer sample guard, composed using Kleene AND.
type FilterSum struct {
	Source         FilterCountSource `json:"source"`
	Period         FilterPeriod      `json:"period"`
	Field          FilterSumField    `json:"field"`
	NativeCurrency string            `json:"nativeCurrency"`
	Where          *Filter           `json:"where,omitzero"`
	Minimum        json.Value        `json:"minimum,omitzero"`
}

// UnmarshalJSON decodes a Sum subject without resolving its field reference.
func (sum *FilterSum) UnmarshalJSON(data []byte) error {
	if sum == nil {
		return errors.New("core: cannot unmarshal a monetary condition into nil")
	}
	if !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("monetary condition must be an object")
	}
	var decoded FilterSum
	for name, value := range json.Value(data).Properties() {
		if value.IsNull() && name != "where" {
			return errors.New("monetary fields cannot be null")
		}
		var err error
		switch name {
		case "source":
			err = json.Unmarshal(value, &decoded.Source)
		case "period":
			err = json.Unmarshal(value, &decoded.Period)
		case "field":
			err = json.Unmarshal(value, &decoded.Field)
		case "nativeCurrency":
			err = json.Unmarshal(value, &decoded.NativeCurrency)
		case "minimum":
			decoded.Minimum = bytes.Clone(value)
		case "where":
			if !value.IsNull() {
				var rule FilterRule
				rule, _, err = unmarshalFilterRule(json.NewDecoder(bytes.NewReader(value)),
					1, expectFilterGroup, occurrenceFilterRule)
				if err == nil {
					decoded.Where = rule.(*Filter)
				}
			}
		default:
			return errors.New("unknown monetary condition property")
		}
		if err != nil {
			return errors.New(err.Error())
		}
	}
	*sum = FilterSum(decoded)
	return nil
}

// FilterSumCondition compares a monetary sum with an exact signed JSON Number.
// The supported operators are the four ordered numeric comparisons.
type FilterSumCondition struct {
	Sum      FilterSum      `json:"sum"`
	Operator FilterOperator `json:"operator"`
	Values   []json.Value   `json:"values"`
}

// MarshalJSON returns the condition's JSON without semantic validation.
func (condition *FilterSumCondition) MarshalJSON() ([]byte, error) {
	if condition == nil {
		return nil, errors.New("filter sum condition cannot be nil")
	}
	type encoded FilterSumCondition
	data, err := json.Marshal((*encoded)(condition))
	if err != nil {
		return nil, errors.New(err.Error())
	}
	return data, nil
}

// UnmarshalJSON decodes a Sum condition using the common rule discriminator.
func (condition *FilterSumCondition) UnmarshalJSON(data []byte) error {
	if condition == nil {
		return errors.New("core: cannot unmarshal a Sum condition into nil")
	}
	rule, _, err := unmarshalFilterRule(json.NewDecoder(bytes.NewReader(data)), 1, expectAnyFilterRule, profileFilterRule)
	if err != nil {
		return errors.New(err.Error())
	}
	decoded, ok := rule.(*FilterSumCondition)
	if !ok {
		return errors.New("expected a sum condition")
	}
	*condition = *decoded
	return nil
}

func (*FilterSumCondition) filterRule() {}

// FilterSumField identifies a governed monetary reading and meaning revision.
type FilterSumField struct {
	Definition      string `json:"definition"`
	MeaningRevision string `json:"meaningRevision"`
}

// UnmarshalJSON decodes a field reference without resolving it.
func (field *FilterSumField) UnmarshalJSON(data []byte) error {
	if field == nil {
		return errors.New("core: cannot unmarshal a monetary field into nil")
	}
	if !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("monetary field must be an object")
	}
	for name, value := range json.Value(data).Properties() {
		switch name {
		case "definition", "meaningRevision":
			if !value.IsString() || len(value) > 260 {
				return errors.New("invalid monetary field reference")
			}
		default:
			return errors.New("unknown monetary field reference property")
		}
	}
	type encoded FilterSumField
	var decoded encoded
	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return errors.New(err.Error())
	}
	*field = FilterSumField(decoded)
	return nil
}

func prepareFilterSum(condition *FilterSumCondition, calendar *behavioral.Calendar) (behavioral.Rule, error) {

	if condition == nil {
		return behavioral.Rule{}, errors.New("Sum condition cannot be nil")
	}
	sum := condition.Sum
	scope, err := sum.Period.scope(sum.Source, calendar)
	var unavailable error
	if err != nil {
		if _, ok := errors.AsType[*behavioral.UnsupportedError](err); !ok {
			return behavioral.Rule{}, err
		}
		unavailable = err
	}
	field, err := behavioral.ResolveMonetaryField(sum.Field.Definition, sum.Field.MeaningRevision)
	if err != nil {
		return behavioral.Rule{}, err
	}
	if sum.Source.Type != "" && sum.Source.Type != "track" || sum.Source.Name != "Order Completed" {
		return behavioral.Rule{}, errors.New("monetary field is incompatible with the occurrence source")
	}
	if len(sum.NativeCurrency) != 3 {
		return behavioral.Rule{}, errors.New("nativeCurrency requires three uppercase letters")
	}
	for _, c := range sum.NativeCurrency {
		if c < 'A' || c > 'Z' {
			return behavioral.Rule{}, errors.New("nativeCurrency requires three uppercase letters")
		}
	}

	// Prepare the exact comparison operands.
	test, err := prepareMonetaryTest(condition.Operator, condition.Values)
	if err != nil {
		return behavioral.Rule{}, err
	}

	// Validate optional forms before reporting unavailable capabilities.
	minimum := filterCountInteger("0")
	if sum.Minimum != nil {
		minimum, err = parseFilterCountInteger(sum.Minimum)
		if err != nil {
			return behavioral.Rule{}, errors.New("sample minimum must be a positive JSON integer")
		}
		if minimum == "0" {
			return behavioral.Rule{}, errors.New("sample minimum must be positive")
		}
	}
	var where *state.Where
	if sum.Where != nil {
		source := behavioral.Scope{Type: sum.Source.Type, Event: sum.Source.Name, Grain: sum.Source.Grain}
		validation := filterValidation{maxDepth: 2, properties: schemas.Event.Properties(), source: &source}
		var err error
		where, err = prepareFilterGroup(sum.Where, &validation, 1)
		if err != nil {
			if _, unsupported := errors.AsType[*behavioral.UnsupportedError](err); !unsupported {
				return behavioral.Rule{}, err
			}
			unavailable = err
		}
	}
	err = field.ValidateWhere(where)
	if err != nil {
		return behavioral.Rule{}, err
	}
	if unavailable != nil {
		return behavioral.Rule{}, unavailable
	}
	m, ok := minimum.uint64()
	if !ok || m > math.MaxInt64 {
		return behavioral.Rule{}, &behavioral.UnsupportedError{
			Capability: "monetary sample minimum exceeds the execution integer range"}
	}
	if test.Operator == behavioral.Equal || test.Operator == behavioral.Different || test.Operator == behavioral.Between {
		return behavioral.Rule{}, &behavioral.UnsupportedError{Capability: "monetary equality or interval comparisons"}
	}

	return behavioral.Rule{Sum: &behavioral.Sum{Scope: scope, Field: field, NativeCurrency: sum.NativeCurrency,
		Where: where, Minimum: m, Test: test}}, nil
}
