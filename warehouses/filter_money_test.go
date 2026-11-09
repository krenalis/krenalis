// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/types"
)

// TestValidateMoneyQualification checks the prepared reading and threshold.
func TestValidateMoneyQualification(t *testing.T) {
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name   string
		change func(*MoneyQualification)
	}{
		{"amount column", func(m *MoneyQualification) { m.Amount.Column.Type = types.String() }},
		{"amount type", func(m *MoneyQualification) { m.Amount.Type = types.Decimal(19, 4) }},
		{"amount key", func(m *MoneyQualification) { m.Amount.Keys = []string{"\x00"} }},
		{"currency column", func(m *MoneyQualification) { m.Currency.Column.Type = types.String() }},
		{"currency key", func(m *MoneyQualification) { m.Currency.Keys = []string{"\x00"} }},
		{"threshold scale", func(m *MoneyQualification) { m.Threshold = decimal.MustParse("0.00001") }},
		{"threshold bound", func(m *MoneyQualification) { m.Threshold = decimal.MustParse("1e15") }},
	} {
		t.Run(test.name, func(t *testing.T) {
			properties := Column{Name: "properties", Type: types.JSON()}
			money := &MoneyQualification{
				Currency:  KnownTextEqual{Column: properties, Keys: []string{"currency"}, Value: "EUR"},
				Amount:    DecimalReading{Column: properties, Keys: []string{"total"}, Type: types.Decimal(18, 4)},
				Threshold: decimal.MustParse("100"), AtLeast: true,
			}
			test.change(money)
			node := &SelectedOccurrence{Source: OccurrenceSource{Table: "events",
				Profile: Column{Name: "kpid", Type: types.UUID()}, Identity: Column{Name: "id", Type: types.String()},
				Timestamp: Column{Name: "timestamp", Type: types.DateTime()}},
				Start: end.AddDate(0, -1, 0), End: end, Verify: money}
			request := ProfileOutcomeRequest{Population: ProfileSource{Table: "profiles",
				Key: Column{Name: "id", Type: types.UUID()}}, Limit: 1}
			err := ValidateProfileFilter(Filter{Root: node}, request)
			if err == nil {
				t.Fatal("expected invalid monetary qualification, got success")
			}
		})
	}
}
