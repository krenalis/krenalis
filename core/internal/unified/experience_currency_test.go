// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
)

// TestExperienceCurrencyValidation checks canonical callers independently.
func TestExperienceCurrencyValidation(t *testing.T) {
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name        string
		change      func(*Experience)
		unsupported bool
	}{
		{"unresolved", func(e *Experience) { e.Currency = &NativeCurrency{} }, false},
		{"page", func(e *Experience) { e.Scope.Type = "page" }, false},
		{"source", func(e *Experience) { e.Scope.Event = "Other" }, false},
		{"currency", func(e *Experience) { e.Currency.Value = "eur" }, false},
		{"revision", func(e *Experience) { e.Currency.Revision = "2" }, false},
		{"base overlap", func(e *Experience) {
			e.Where = &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
				&state.WhereCondition{Property: []string{"properties", "currency"}, Operator: state.OpIsNull}}}
		}, true},
		{"verify overlap", func(e *Experience) { e.Verify.Record.Property = []string{"properties", "currency"} }, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			currency, err := ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR", "Order Completed")
			if err != nil {
				t.Fatalf("expected resolved currency, got %v", err)
			}
			experience := &Experience{Scope: Scope{Event: "Order Completed", Grain: "registered-message",
				Start: at.Add(-time.Hour), End: at}, Currency: currency,
				Verify: &OccurrencePredicate{Record: &state.WhereCondition{Property: []string{"event"},
					Operator: state.OpIsNull}}}
			test.change(experience)
			err = Validate(&Rule{Experience: experience}, at, Page{Limit: 1})
			if err != nil {
				_, unsupported := errors.AsType[*UnsupportedError](err)
				if unsupported != test.unsupported {
					t.Fatalf("expected unsupported=%v, got %v", test.unsupported, err)
				}
				return
			}
			t.Fatal("expected canonical validation error, got success")
		})
	}
}
