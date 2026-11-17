// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// FilterMoney identifies one message's native monetary reading.
// Field determines both amount and currency paths; no conversion is performed.
type FilterMoney struct {
	Field          FilterSumField `json:"field"`
	NativeCurrency string         `json:"nativeCurrency"`
}

// UnmarshalJSON decodes the closed monetary subject without resolving it.
func (money *FilterMoney) UnmarshalJSON(data []byte) error {
	if money == nil {
		return errors.New("core: cannot unmarshal money into nil")
	}
	if !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("money must be an object")
	}
	var decoded FilterMoney
	for name, value := range json.Value(data).Properties() {
		if value.IsNull() {
			return errors.New("money fields cannot be null")
		}
		var err error
		switch name {
		case "field":
			err = json.Unmarshal(value, &decoded.Field)
		case "nativeCurrency":
			err = json.Unmarshal(value, &decoded.NativeCurrency)
		default:
			return errors.New("unknown money property")
		}
		if err != nil {
			return errors.New(err.Error())
		}
	}
	*money = decoded
	return nil
}

// FilterMoneyCondition verifies native currency and one amount.
// It executes in Count.where and experience.verify, using exact comparisons.
type FilterMoneyCondition struct {
	Money    FilterMoney    `json:"money"`
	Operator FilterOperator `json:"operator"`
	Values   []json.Value   `json:"values"`
}

// MarshalJSON returns the condition without semantic validation.
func (condition *FilterMoneyCondition) MarshalJSON() ([]byte, error) {
	if condition == nil {
		return nil, errors.New("filter money condition cannot be nil")
	}
	type encoded FilterMoneyCondition
	data, err := json.Marshal((*encoded)(condition))
	if err != nil {
		return nil, errors.New(err.Error())
	}
	return data, nil
}

// UnmarshalJSON decodes a standalone monetary condition.
func (condition *FilterMoneyCondition) UnmarshalJSON(data []byte) error {
	if condition == nil {
		return errors.New("core: cannot unmarshal a money condition into nil")
	}
	rule, _, err := unmarshalFilterRule(json.NewDecoder(bytes.NewReader(data)), 1, expectAnyFilterRule, selectedFilterRule)
	if err != nil {
		return errors.New(err.Error())
	}
	decoded, ok := rule.(*FilterMoneyCondition)
	if !ok {
		return errors.New("expected a money condition")
	}
	*condition = *decoded
	return nil
}

func (*FilterMoneyCondition) filterRule() {}

func prepareFilterMoney(condition *FilterMoneyCondition, source behavioral.Scope) (*behavioral.Money, error) {
	if condition == nil {
		return nil, errors.New("filter money condition cannot be nil")
	}
	field, err := behavioral.ResolveMonetaryField(condition.Money.Field.Definition, condition.Money.Field.MeaningRevision)
	if err != nil {
		return nil, err
	}
	test, err := prepareMonetaryTest(condition.Operator, condition.Values)
	if err != nil {
		return nil, err
	}
	money := &behavioral.Money{Field: field, NativeCurrency: condition.Money.NativeCurrency, Test: test}
	err = money.Validate(source)
	if err != nil {
		if _, unsupported := errors.AsType[*behavioral.UnsupportedError](err); !unsupported {
			return nil, err
		}
	}
	return money, nil
}

// prepareMonetaryTest reads a literal test with the shared monetary limits.
func prepareMonetaryTest(operator FilterOperator, operands []json.Value) (behavioral.SumTest, error) {
	comparison, ok := map[FilterOperator]behavioral.Comparison{
		OpIs: behavioral.Equal, OpIsNot: behavioral.Different, OpIsLessThan: behavioral.Less,
		OpIsLessThanOrEqualTo: behavioral.LessOrEqual, OpIsGreaterThan: behavioral.Greater,
		OpIsGreaterThanOrEqualTo: behavioral.GreaterOrEqual, OpIsBetween: behavioral.Between,
	}[operator]
	if !ok {
		return behavioral.SumTest{}, errors.New("invalid monetary comparison")
	}
	arity := 1
	if comparison == behavioral.Between {
		arity = 2
	}
	if len(operands) != arity {
		return behavioral.SumTest{}, errors.New("invalid monetary operand count")
	}
	var values [2]decimal.Decimal
	for i, value := range operands {
		if len(value) > 4096 {
			return behavioral.SumTest{}, errors.New("monetary literal exceeds the input length limit")
		}
		decoded, err := state.DecodeFilterOperand(value, types.Decimal(18, 4), state.OpIs)
		if err != nil {
			return behavioral.SumTest{}, err
		}
		values[i] = decoded.(decimal.Decimal)
	}
	if arity == 2 && values[1].Less(values[0]) {
		return behavioral.SumTest{}, errors.New("invalid monetary interval")
	}
	return behavioral.SumTest{Operator: comparison, Value: values[0], Upper: values[1]}, nil
}
