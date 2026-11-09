// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"fmt"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
)

// TestMoneyLiteralNormalization preserves exact points and inclusive intervals.
func TestMoneyLiteralNormalization(t *testing.T) {
	for _, test := range []struct {
		operator                  Comparison
		value, upper, lower, high string
		readable                  string
	}{
		{Equal, "100.00000", "", "100", "", ""},
		{Different, "1E2", "", "100", "", ""},
		{Equal, "-0", "", "0", "", ""},
		{Equal, "99999999999999.9999", "", "99999999999999.9999", "", ""},
		{Different, "-99999999999999.9999", "", "-99999999999999.9999", "", ""},
		{Equal, "100.00001", "", "", "", "none"},
		{Different, "100.00001", "", "", "", "all"},
		{Equal, "-100.00001", "", "", "", "none"},
		{Different, "-100.00001", "", "", "", "all"},
		{Equal, "1e14", "", "", "", "none"},
		{Different, "-1e14", "", "", "", "all"},
		{Equal, "1e20", "", "", "", "none"},
		{Different, "-1e20", "", "", "", "all"},
		{Between, "50.00001", "100.00009", "50.0001", "100", ""},
		{Between, "99.99995", "100.00004", "100", "100", ""},
		{Between, "100", "100", "100", "100", ""},
		{Between, "-100.00009", "-50.00001", "-100", "-50.0001", ""},
		{Between, "100.00001", "100.00009", "", "", "none"},
		{Between, "-100.00009", "-100.00001", "", "", "none"},
		{Between, "1e14", "1e20", "", "", "none"},
		{Between, "-1e20", "-1e14", "", "", "none"},
		{Between, "-1e20", "1e20", "", "", "all"},
		{Between, "-99999999999999.9999", "99999999999999.9999", "", "", "all"},
		{Between, "-1e20", "0", "-99999999999999.9999", "0", ""},
		{Between, "0", "1e20", "0", "99999999999999.9999", ""},
		{Between, "-1e14", "-99999999999999.9999", "-99999999999999.9999", "-99999999999999.9999", ""},
		{Between, "99999999999999.9999", "1e14", "99999999999999.9999", "99999999999999.9999", ""},
	} {
		t.Run(fmt.Sprintf("%d/%s/%s", test.operator, test.value, test.upper), func(t *testing.T) {
			money := &Money{Test: SumTest{Operator: test.operator, Value: decimal.MustParse(test.value)}}
			if test.upper != "" {
				money.Test.Upper = decimal.MustParse(test.upper)
			}
			normalized, err := money.NormalizedTest()
			if err != nil {
				t.Fatalf("expected exact literal normalization, got %v", err)
			}
			if normalized.ReadableAmounts != test.readable || (normalized.Comparison == nil) != (test.readable != "") {
				t.Fatalf("expected readable domain %q, got %+v", test.readable, normalized)
			}
			if comparison := normalized.Comparison; comparison != nil {
				if comparison.Operator != test.operator || !comparison.Value.Equal(decimal.MustParse(test.lower)) ||
					test.high != "" && !comparison.Upper.Equal(decimal.MustParse(test.high)) {
					t.Fatalf("expected operator %d with [%s,%s], got %+v", test.operator, test.lower, test.high, comparison)
				}
			}
		})
	}
}

// TestMoneyNormalization compares exact tests, including signed grid borders.
func TestMoneyNormalization(t *testing.T) {

	for _, operator := range []Comparison{Less, LessOrEqual, Greater, GreaterOrEqual} {
		for threshold := -31; threshold <= 31; threshold++ {
			value := decimal.MustParse(fmt.Sprintf("%de-5", threshold))
			money := &Money{Test: SumTest{Operator: operator, Value: value}}
			normalized, err := money.NormalizedTest()
			if err != nil {
				t.Fatalf("expected normalized exact threshold, got %v", err)
			}
			for coefficient := -4; coefficient <= 4; coefficient++ {
				amount := decimal.MustParse(fmt.Sprintf("%de-4", coefficient))
				want := false
				switch operator {
				case Less:
					want = amount.Less(value)
				case LessOrEqual:
					want = !amount.Greater(value)
				case Greater:
					want = amount.Greater(value)
				case GreaterOrEqual:
					want = !amount.Less(value)
				default:
					t.Fatalf("expected an ordered comparison, got %d", operator)
				}
				got := !amount.Greater(normalized.Comparison.Value)
				if normalized.Comparison.Operator == GreaterOrEqual {
					got = !amount.Less(normalized.Comparison.Value)
				}
				if got != want {
					t.Fatalf("expected %s operator=%d %s to give %v, got %+v", amount, operator, value, want, normalized)
				}
			}
		}
	}

	for _, test := range []struct{ input, expected string }{
		{"1e75", "1e14"}, {"-1e75", "-1e14"}, {"1e-37", "0.0001"}, {"-1e-37", "0"},
	} {
		money := &Money{Test: SumTest{Operator: GreaterOrEqual, Value: decimal.MustParse(test.input)}}
		normalized, err := money.NormalizedTest()
		if err != nil {
			t.Fatalf("expected bounded normalization, got %v", err)
		}
		if !normalized.Comparison.Value.Equal(decimal.MustParse(test.expected)) {
			t.Fatalf("expected threshold %s, got %s", test.expected, normalized.Comparison.Value)
		}
	}

}

// TestMoneyValidation checks internal callers as well as public preparation.
func TestMoneyValidation(t *testing.T) {

	field, err := ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected monetary reference resolved, got %v", err)
	}
	scope := Scope{Event: "Order Completed", Grain: "registered-message"}
	for _, test := range []struct {
		name   string
		change func(*Money, *Scope)
		valid  bool
	}{
		{"nil field", func(m *Money, _ *Scope) { m.Field = nil }, false},
		{"source", func(_ *Money, s *Scope) { s.Event = "Other" }, false},
		{"screen", func(_ *Money, s *Scope) { s.Type = "screen" }, false},
		{"currency", func(m *Money, _ *Scope) { m.NativeCurrency = "eur" }, false},
		{"operator", func(m *Money, _ *Scope) { m.Test.Operator = 255 }, false},
		{"global domain", func(m *Money, _ *Scope) { m.Test.Value = decimal.MustParse("1e76") }, false},
		{"equality", func(m *Money, _ *Scope) { m.Test.Operator = Equal }, true},
		{"different", func(m *Money, _ *Scope) { m.Test.Operator = Different }, true},
		{"interval", func(m *Money, _ *Scope) { m.Test.Operator = Between; m.Test.Upper = decimal.MustParse("200") }, true},
		{"inverted interval", func(m *Money, _ *Scope) { m.Test.Operator = Between }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			money := &Money{Field: field, NativeCurrency: "EUR",
				Test: SumTest{Operator: GreaterOrEqual, Value: decimal.MustParse("100")}}
			source := scope
			test.change(money, &source)
			err := money.Validate(source)
			if err != nil {
				_, unsupported := errors.AsType[*UnsupportedError](err)
				if test.valid || unsupported {
					t.Fatalf("expected valid=%v, got %v", test.valid, err)
				}
				return
			}
			if !test.valid {
				t.Fatal("expected validation failure, got success")
			}
		})
	}

	// A repeated leaf remains unavailable even under OR and must not be split.
	money := &Money{Field: field, NativeCurrency: "EUR",
		Test: SumTest{Operator: GreaterOrEqual, Value: decimal.MustParse("100")}}
	body := &OccurrencePredicate{Operator: state.OpOr,
		Children: []OccurrencePredicate{{Money: money}, {Money: money}}}
	err = body.validateDependencies(nil, nil, selectedOccurrencePredicate)
	if err != nil {
		if _, unsupported := errors.AsType[*UnsupportedError](err); !unsupported {
			t.Fatalf("expected unavailable repeated monetary reading, got %v", err)
		}
		return
	}
	t.Fatal("expected unavailable repeated monetary reading, got success")

}
