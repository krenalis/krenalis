// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import (
	"fmt"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/types"
)

// MoneyQualification combines currency equality and an exact amount test using
// Kleene AND. Amount and currency are read separately from the same row.
// Unreadable amounts yield NULL for the amount test.
// Count and SelectedOccurrence condition Currency on a required base reading.
// Inclusive bounds have scale at most four; Negate is allowed only for a point.
// ReadableAmounts none/all replaces bounds, preserving unreadable amounts.
type MoneyQualification struct {
	Currency        KnownTextEqual
	Amount          DecimalReading
	Lower, Upper    *decimal.Decimal
	Negate          bool
	ReadableAmounts string
}

func (*MoneyQualification) rowQualification() {}

func (money *MoneyQualification) validate() error {
	if money == nil {
		return fmt.Errorf("warehouses: nil monetary qualification")
	}
	err := money.Amount.Validate()
	if err != nil {
		return err
	}
	if money.ReadableAmounts != "" {
		if money.ReadableAmounts != "none" && money.ReadableAmounts != "all" ||
			money.Lower != nil || money.Upper != nil || money.Negate {
			return fmt.Errorf("warehouses: invalid normalized monetary domain")
		}
		return nil
	}
	if money.Lower == nil && money.Upper == nil {
		return fmt.Errorf("warehouses: missing normalized monetary test")
	}
	limit := decimal.MustParse("1e14")
	for _, bound := range []*decimal.Decimal{money.Lower, money.Upper} {
		if bound == nil {
			continue
		}
		_, scale := bound.PrecisionScale()
		if !types.IsDecimal(*bound) || scale > 4 || bound.Greater(limit) || bound.Less(limit.Neg()) {
			return fmt.Errorf("warehouses: invalid normalized monetary bound")
		}
	}
	if money.Lower != nil && money.Upper != nil && money.Lower.Greater(*money.Upper) {
		return fmt.Errorf("warehouses: inverted normalized monetary bounds")
	}
	if money.Negate && (money.Lower == nil || money.Upper == nil || !money.Lower.Equal(*money.Upper)) {
		return fmt.Errorf("warehouses: monetary negation requires a point")
	}
	return nil
}
