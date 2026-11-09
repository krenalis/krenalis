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

// Money verifies native currency and an exact amount on one occurrence.
// Field owns both reading paths; Test retains the requested comparison.
type Money struct {
	Field          *MonetaryField
	NativeCurrency string
	Test           SumTest
}

// NormalizedTest returns an exact test on readable Decimal(18,4) amounts.
// Ordered bounds retain clipping to +/-1e14, outside every readable amount.
// Points and intervals may yield none/all; unreadable amounts remain unknown.
func (money *Money) NormalizedTest() (NormalizedMoneyTest, error) {
	return money.normalizedTest()
}

// Validate checks the reading, source, currency and requested comparison.
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

func (money *Money) normalizedTest() (NormalizedMoneyTest, error) {
	if money == nil || money.Test.Operator > Between || !types.IsDecimal(money.Test.Value) ||
		!types.IsDecimal(money.Test.Upper) || money.Test.Operator == Between && money.Test.Upper.Less(money.Test.Value) {
		return NormalizedMoneyTest{}, fmt.Errorf("invalid monetary terminal")
	}
	if money.Test.Operator == Equal || money.Test.Operator == Different {
		_, scale := money.Test.Value.PrecisionScale()
		limit := decimal.MustParse("1e14")
		if scale > 4 || !money.Test.Value.Less(limit) || !money.Test.Value.Greater(limit.Neg()) {
			readable := "none"
			if money.Test.Operator == Different {
				readable = "all"
			}
			return NormalizedMoneyTest{ReadableAmounts: readable}, nil
		}
		test := money.Test
		return NormalizedMoneyTest{Comparison: &test}, nil
	}
	if money.Test.Operator == Between {
		lower, _, err := (SumTest{Operator: GreaterOrEqual, Value: money.Test.Value}).threshold()
		if err != nil {
			return NormalizedMoneyTest{}, err
		}
		upper, _, err := (SumTest{Operator: LessOrEqual, Value: money.Test.Upper}).threshold()
		if err != nil {
			return NormalizedMoneyTest{}, err
		}
		maximum := decimal.MustParse("99999999999999.9999")
		minimum := maximum.Neg()
		if lower.Greater(upper) || lower.Greater(maximum) || upper.Less(minimum) {
			return NormalizedMoneyTest{ReadableAmounts: "none"}, nil
		}
		if !lower.Greater(minimum) && !upper.Less(maximum) {
			return NormalizedMoneyTest{ReadableAmounts: "all"}, nil
		}
		if lower.Less(minimum) {
			lower = minimum
		}
		if upper.Greater(maximum) {
			upper = maximum
		}
		return NormalizedMoneyTest{Comparison: &SumTest{Operator: Between, Value: lower, Upper: upper}}, nil
	}
	value, atLeast, err := money.Test.threshold()
	if err != nil {
		return NormalizedMoneyTest{}, err
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
	return NormalizedMoneyTest{Comparison: &SumTest{Operator: operator, Value: value}}, nil
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
	qualification := &warehouses.MoneyQualification{
		Currency: warehouses.KnownTextEqual{Column: source.Properties,
			Keys: slices.Clone(money.Field.currencyPath[1:]), Value: money.NativeCurrency},
		Amount: warehouses.DecimalReading{Column: source.Properties,
			Keys: slices.Clone(money.Field.amountPath[1:]), Type: money.Field.amountType},
		ReadableAmounts: test.ReadableAmounts,
	}
	if comparison := test.Comparison; comparison != nil {
		switch comparison.Operator {
		case GreaterOrEqual:
			qualification.Lower = &comparison.Value
		case LessOrEqual:
			qualification.Upper = &comparison.Value
		case Equal, Different:
			qualification.Lower, qualification.Upper = &comparison.Value, &comparison.Value
			qualification.Negate = comparison.Operator == Different
		case Between:
			qualification.Lower, qualification.Upper = &comparison.Value, &comparison.Upper
		default:
			return nil, fmt.Errorf("core/internal/unified: unexpected normalized monetary operator %d", comparison.Operator)
		}
	}
	return qualification, nil
}

// NormalizedMoneyTest contains a comparison or a readable-amount domain.
// ReadableAmounts is none or all only when Comparison is nil. Neither domain
// determines an unreadable amount or removes the separate currency requirement.
type NormalizedMoneyTest struct {
	Comparison      *SumTest
	ReadableAmounts string
}
