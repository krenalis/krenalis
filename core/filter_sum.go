// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"math"

	"github.com/krenalis/krenalis/core/internal/schemas"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
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
		return errors.New("core: cannot unmarshal Sum into nil")
	}
	if !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("sum must be an object")
	}
	var decoded FilterSum
	for name, value := range json.Value(data).Properties() {
		if value.IsNull() && name != "where" {
			return errors.New("Sum fields cannot be null")
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
				rule, _, err = unmarshalFilterRule(json.NewDecoder(bytes.NewReader(value)), 1, expectFilterGroup, true)
				if err == nil {
					decoded.Where = rule.(*Filter)
				}
			}
		default:
			return errors.New("unknown sum property")
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
	rule, _, err := unmarshalFilterRule(json.NewDecoder(bytes.NewReader(data)), 1, expectAnyFilterRule, false)
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
		return errors.New("core: cannot unmarshal a Sum field into nil")
	}
	if !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("Sum field must be an object")
	}
	for name, value := range json.Value(data).Properties() {
		switch name {
		case "definition", "meaningRevision":
			if !value.IsString() || len(value) > 260 {
				return errors.New("invalid Sum field reference")
			}
		default:
			return errors.New("unknown Sum field reference property")
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

func prepareFilterSum(condition *FilterSumCondition, calendar *unified.Calendar) (unified.Rule, error) {

	if condition == nil {
		return unified.Rule{}, errors.New("Sum condition cannot be nil")
	}
	sum := condition.Sum
	scope, err := sum.Period.scope(sum.Source, calendar)
	var unavailable error
	if err != nil {
		if _, ok := errors.AsType[*unified.UnsupportedError](err); !ok {
			return unified.Rule{}, err
		}
		unavailable = err
	}
	field, err := unified.ResolveMonetaryField(sum.Field.Definition, sum.Field.MeaningRevision)
	if err != nil {
		return unified.Rule{}, err
	}
	if sum.Source.Name != "Order Completed" {
		return unified.Rule{}, errors.New("monetary field is incompatible with the occurrence source")
	}
	if len(sum.NativeCurrency) != 3 {
		return unified.Rule{}, errors.New("nativeCurrency requires three uppercase letters")
	}
	for _, c := range sum.NativeCurrency {
		if c < 'A' || c > 'Z' {
			return unified.Rule{}, errors.New("nativeCurrency requires three uppercase letters")
		}
	}

	// Prepare the exact comparison operands.
	operator, ok := map[FilterOperator]unified.Comparison{
		OpIs: unified.Equal, OpIsNot: unified.Different, OpIsLessThan: unified.Less,
		OpIsLessThanOrEqualTo: unified.LessOrEqual, OpIsGreaterThan: unified.Greater,
		OpIsGreaterThanOrEqualTo: unified.GreaterOrEqual, OpIsBetween: unified.Between,
	}[condition.Operator]
	if !ok {
		return unified.Rule{}, errors.New("invalid Sum comparison")
	}
	arity := 1
	if operator == unified.Between {
		arity = 2
	}
	if len(condition.Values) != arity {
		return unified.Rule{}, errors.New("invalid Sum operand count")
	}
	values := make([]decimal.Decimal, arity)
	for i, value := range condition.Values {
		// Bound lexical work before parsing arbitrary precision decimal literals.
		if len(value) > 4096 {
			return unified.Rule{}, errors.New("Sum literal exceeds the input length limit")
		}
		decoded, err := state.DecodeFilterOperand(value, types.Decimal(18, 4), state.OpIs)
		if err != nil {
			return unified.Rule{}, err
		}
		values[i] = decoded.(decimal.Decimal)
	}
	if arity == 2 && values[1].Less(values[0]) {
		return unified.Rule{}, errors.New("invalid Sum interval")
	}

	// Validate optional forms before reporting unavailable capabilities.
	minimum := filterCountInteger("0")
	if sum.Minimum != nil {
		minimum, err = parseFilterCountInteger(sum.Minimum)
		if err != nil {
			return unified.Rule{}, err
		}
		if minimum == "0" {
			return unified.Rule{}, errors.New("Sum minimum must be positive")
		}
	}
	var where *state.Where
	if sum.Where != nil {
		validation := filterValidation{maxDepth: 2, properties: schemas.Event.Properties()}
		var err error
		where, err = prepareFilterGroup(sum.Where, &validation, 1)
		if err != nil {
			return unified.Rule{}, err
		}
	}
	err = field.ValidateWhere(where)
	if err != nil {
		return unified.Rule{}, err
	}
	if unavailable != nil {
		return unified.Rule{}, unavailable
	}
	m, ok := minimum.uint64()
	if !ok || m > math.MaxInt64 {
		return unified.Rule{}, &unified.UnsupportedError{Capability: "Sum sample minimum exceeds the execution integer range"}
	}
	if operator == unified.Equal || operator == unified.Different || operator == unified.Between {
		return unified.Rule{}, &unified.UnsupportedError{Capability: "Sum equality or interval reachability"}
	}

	return unified.Rule{Sum: &unified.Sum{Scope: scope, Field: field, NativeCurrency: sum.NativeCurrency,
		Where: where, Minimum: m, Test: unified.SumTest{Operator: operator, Value: values[0]}}}, nil
}
