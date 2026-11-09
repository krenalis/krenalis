// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"fmt"
	"slices"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// Money verifies native currency and an exact amount on the selected message.
// Field owns both reading paths; Test retains the requested comparison.
type Money struct {
	Field          *MonetaryField
	NativeCurrency string
	Test           SumTest
}

// NormalizedTest returns an inclusive test on readable Decimal(18,4) amounts.
// Bounds are clipped to +/-1e14, outside every readable amount. Unknown amounts
// remain unknown even when the comparison is constant for readable values.
// It returns *UnsupportedError for equality and interval comparisons.
func (money *Money) NormalizedTest() (SumTest, error) {
	return money.normalizedTest()
}

// Validate checks the reading, source, currency and requested comparison.
// It returns *UnsupportedError for valid comparisons outside this subset.
func (money *Money) Validate(scope Scope) error {
	if money == nil || money.Field == nil || money.Field.definition == "" {
		return fmt.Errorf("unresolved monetary field")
	}
	if scope.eventType() != "track" || scope.Event != "Order Completed" || scope.Grain != "registered-message" {
		return fmt.Errorf("monetary field is incompatible with the occurrence source")
	}
	if len(money.NativeCurrency) != 3 {
		return fmt.Errorf("nativeCurrency requires three uppercase letters")
	}
	for _, c := range money.NativeCurrency {
		if c < 'A' || c > 'Z' {
			return fmt.Errorf("nativeCurrency requires three uppercase letters")
		}
	}
	_, err := money.normalizedTest()
	return err
}

func (money *Money) normalizedTest() (SumTest, error) {
	if money == nil || money.Test.Operator > Between || !types.IsDecimal(money.Test.Value) ||
		!types.IsDecimal(money.Test.Upper) || money.Test.Operator == Between && money.Test.Upper.Less(money.Test.Value) {
		return SumTest{}, fmt.Errorf("invalid monetary terminal")
	}
	if money.Test.Operator == Equal || money.Test.Operator == Different || money.Test.Operator == Between {
		return SumTest{}, &UnsupportedError{Capability: "monetary equality or interval comparisons"}
	}
	value, atLeast, err := money.Test.threshold()
	if err != nil {
		return SumTest{}, err
	}
	limit := decimal.MustParse("1e14")
	if value.Greater(limit) {
		value = limit
	} else if value.Less(limit.Neg()) {
		value = limit.Neg()
	}
	operator := LessOrEqual
	if atLeast {
		operator = GreaterOrEqual
	}
	return SumTest{Operator: operator, Value: value}, nil
}

func (money *Money) prepare(source SourceBinding) (warehouses.RowQualification, error) {
	if !warehouses.IsValidIdentifier(source.Properties.Name) || !source.Properties.Type.IsJSON() {
		return nil, fmt.Errorf("core/internal/unified: invalid resolved monetary column")
	}
	test, err := money.normalizedTest()
	if err != nil {
		return nil, err
	}
	for _, path := range [][]string{money.Field.amountPath, money.Field.currencyPath} {
		if len(path) < 2 || path[0] != "properties" {
			return nil, fmt.Errorf("core/internal/unified: invalid resolved monetary path")
		}
	}
	return &warehouses.MoneyQualification{
		Currency: warehouses.KnownTextEqual{Column: source.Properties,
			Keys: slices.Clone(money.Field.currencyPath[1:]), Value: money.NativeCurrency},
		Amount: warehouses.DecimalReading{Column: source.Properties,
			Keys: slices.Clone(money.Field.amountPath[1:]), Type: money.Field.amountType},
		Threshold: test.Value, AtLeast: test.Operator == GreaterOrEqual,
	}, nil
}
