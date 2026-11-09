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
// KnownResult, when present, applies only to readable amounts; unreadable
// amounts still yield NULL for the amount test. It never fixes the currency.
type MoneyQualification struct {
	Currency    KnownTextEqual
	Amount      DecimalReading
	Threshold   decimal.Decimal
	AtLeast     bool
	KnownResult *bool
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
	limit := decimal.MustParse("1e14")
	_, scale := money.Threshold.PrecisionScale()
	if !types.IsDecimal(money.Threshold) || scale > 4 ||
		money.Threshold.Greater(limit) || money.Threshold.Less(limit.Neg()) {
		return fmt.Errorf("warehouses: invalid normalized monetary threshold")
	}
	if money.KnownResult != nil &&
		(!money.Threshold.Equal(limit) && !money.Threshold.Equal(limit.Neg()) ||
			*money.KnownResult != (money.AtLeast == money.Threshold.Less(decimal.Decimal{}))) {
		return fmt.Errorf("warehouses: invalid monetary known-value result")
	}
	return nil
}
