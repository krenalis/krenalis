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

// TestValidateMoneyDomains checks the closed normalized amount-test forms.
func TestValidateMoneyDomains(t *testing.T) {
	lower, upper := decimal.MustParse("50"), decimal.MustParse("100")
	for _, test := range []struct {
		name   string
		change func(*MoneyQualification)
		valid  bool
	}{
		{"lower", func(m *MoneyQualification) {}, true},
		{"upper", func(m *MoneyQualification) { m.Lower, m.Upper = nil, &upper }, true},
		{"point", func(m *MoneyQualification) { m.Upper = &lower }, true},
		{"different", func(m *MoneyQualification) { m.Upper, m.Negate = &lower, true }, true},
		{"interval", func(m *MoneyQualification) { m.Upper = &upper }, true},
		{"none", func(m *MoneyQualification) { m.Lower, m.ReadableAmounts = nil, "none" }, true},
		{"all", func(m *MoneyQualification) { m.Lower, m.ReadableAmounts = nil, "all" }, true},
		{"missing", func(m *MoneyQualification) { m.Lower = nil }, false},
		{"mixed", func(m *MoneyQualification) { m.ReadableAmounts = "none" }, false},
		{"unknown domain", func(m *MoneyQualification) { m.Lower, m.ReadableAmounts = nil, "some" }, false},
		{"negated domain", func(m *MoneyQualification) { m.Lower, m.ReadableAmounts, m.Negate = nil, "all", true }, false},
		{"negated lower", func(m *MoneyQualification) { m.Negate = true }, false},
		{"negated interval", func(m *MoneyQualification) { m.Upper, m.Negate = &upper, true }, false},
		{"inverted", func(m *MoneyQualification) { m.Lower, m.Upper = &upper, &lower }, false},
		{"upper scale", func(m *MoneyQualification) { v := decimal.MustParse("100.00001"); m.Upper = &v }, false},
		{"upper limit", func(m *MoneyQualification) { v := decimal.MustParse("1e15"); m.Upper = &v }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			properties := Column{Name: "properties", Type: types.JSON()}
			money := &MoneyQualification{
				Currency: KnownTextEqual{Column: properties, Keys: []string{"currency"}, Value: "EUR"},
				Amount:   DecimalReading{Column: properties, Keys: []string{"total"}, Type: types.Decimal(18, 4)},
				Lower:    &lower,
			}
			test.change(money)
			err := money.validate()
			if err != nil {
				if test.valid {
					t.Fatalf("expected normalized domain accepted, got %v", err)
				}
				return
			}
			if !test.valid {
				t.Fatal("expected invalid normalized domain rejected, got success")
			}
		})
	}
}

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
		{"threshold scale", func(m *MoneyQualification) { value := decimal.MustParse("0.00001"); m.Lower = &value }},
		{"threshold bound", func(m *MoneyQualification) { value := decimal.MustParse("1e15"); m.Lower = &value }},
	} {
		t.Run(test.name, func(t *testing.T) {
			properties := Column{Name: "properties", Type: types.JSON()}
			threshold := decimal.MustParse("100")
			money := &MoneyQualification{
				Currency: KnownTextEqual{Column: properties, Keys: []string{"currency"}, Value: "EUR"},
				Amount:   DecimalReading{Column: properties, Keys: []string{"total"}, Type: types.Decimal(18, 4)},
				Lower:    &threshold,
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
