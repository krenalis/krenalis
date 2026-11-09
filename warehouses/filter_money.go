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
// SelectedOccurrence can condition Currency on a required base reading.
type MoneyQualification struct {
	Currency  KnownTextEqual
	Amount    DecimalReading
	Threshold decimal.Decimal
	AtLeast   bool
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
	return nil
}
