// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"fmt"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
)

// TestLastPreviousNormalization preserves the exact inclusive delta domain.
func TestLastPreviousNormalization(t *testing.T) {
	for _, test := range []struct {
		operator           Comparison
		value, bound, mode string
		atLeast            bool
	}{
		{GreaterOrEqual, "100.00001", "100.0001", "", true},
		{Greater, "100.00000", "100.0001", "", true},
		{LessOrEqual, "-100.00001", "-100.0001", "", false},
		{Less, "-100.00000", "-100.0001", "", false},
		{Greater, "-100.00001", "-100", "", true},
		{Less, "100.00001", "100", "", false},
		{GreaterOrEqual, "199999999999999.9998", "199999999999999.9998", "", true},
		{Greater, "199999999999999.9998", "", "none", false},
		{LessOrEqual, "-199999999999999.9998", "-199999999999999.9998", "", false},
		{Less, "-199999999999999.9998", "", "none", false},
		{GreaterOrEqual, "-199999999999999.9998", "", "all", false},
		{LessOrEqual, "199999999999999.9998", "", "all", false},
		{GreaterOrEqual, "199999999999999.99981", "", "none", false},
		{LessOrEqual, "-199999999999999.99981", "", "none", false},
		{GreaterOrEqual, "1e75", "", "none", false},
		{LessOrEqual, "1e75", "", "all", false},
		{GreaterOrEqual, "-1e75", "", "all", false},
		{LessOrEqual, "-1e75", "", "none", false},
	} {
		t.Run(fmt.Sprintf("operator=%d/value=%s", test.operator, test.value), func(t *testing.T) {
			check := LastPreviousCheck{Kind: LastPreviousDifference,
				Test: SumTest{Operator: test.operator, Value: decimal.MustParse(test.value)}}
			actual, err := check.NormalizedTest()
			if err != nil {
				t.Fatalf("expected exact normalization, got %v", err)
			}
			if actual.ReadableDifferences != test.mode || (actual.Comparison == nil) != (test.mode != "") {
				t.Fatalf("expected readable domain %q, got %+v", test.mode, actual)
			}
			if test.mode == "" {
				atLeast := actual.Comparison.Operator == GreaterOrEqual
				if atLeast != test.atLeast || !actual.Comparison.Value.Equal(decimal.MustParse(test.bound)) {
					t.Fatalf("expected inclusive bound %s, atLeast=%v, got %+v", test.bound, test.atLeast, actual)
				}
			}
		})
	}
}

// TestLastPreviousValidation checks models constructed without the JSON codec.
func TestLastPreviousValidation(t *testing.T) {
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	field, err := ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected field binding, got %v", err)
	}
	currency, err := ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR", "Order Completed")
	if err != nil {
		t.Fatalf("expected currency binding, got %v", err)
	}
	valid := LastVersusPrevious{Scope: Scope{Event: "Order Completed", Grain: "registered-message",
		Start: at.AddDate(0, -1, 0), End: at}, Currency: currency, Field: field,
		Check: LastPreviousCheck{Kind: LastPreviousDifference, Test: SumTest{Operator: LessOrEqual}}}
	for _, test := range []struct {
		name        string
		change      func(*LastVersusPrevious)
		unavailable bool
	}{
		{"missing currency", func(p *LastVersusPrevious) { p.Currency = nil }, true},
		{"same", func(p *LastVersusPrevious) { p.Check = LastPreviousCheck{Kind: LastPreviousSameValue} }, true},
		{"same with operand", func(p *LastVersusPrevious) {
			p.Check = LastPreviousCheck{Kind: LastPreviousSameValue, Test: SumTest{Value: decimal.MustParse("1")}}
		}, false},
		{"missing kind", func(p *LastVersusPrevious) { p.Check.Kind = 0 }, false},
		{"bad operator", func(p *LastVersusPrevious) { p.Check.Test.Operator = 99 }, false},
		{"inverted", func(p *LastVersusPrevious) { p.Check.Test = SumTest{Operator: Between, Value: decimal.MustParse("1")} }, false},
		{"missing field", func(p *LastVersusPrevious) { p.Field = nil }, false},
		{"wrong source", func(p *LastVersusPrevious) { p.Scope.Type = "page" }, false},
		{"overlap", func(p *LastVersusPrevious) {
			p.Where = &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
				&state.WhereCondition{Property: []string{"properties", "total"}, Operator: state.OpIs,
					Values: []any{nil}},
			}}
		}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			pair := valid
			test.change(&pair)
			err := Validate(&Rule{LastVersusPrevious: &pair}, at, Page{Limit: 1})
			if err != nil {
				_, unavailable := errors.AsType[*UnsupportedError](err)
				if unavailable != test.unavailable {
					t.Fatalf("expected unavailable=%v, got %v", test.unavailable, err)
				}
				return
			}
			t.Fatal("expected invalid or unsupported pair, got success")
		})
	}
}
