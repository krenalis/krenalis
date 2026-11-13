// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestAveragePreparation checks exact thresholds before the warehouse boundary.
func TestAveragePreparation(t *testing.T) {
	field, err := ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected field fixture, got %v", err)
	}
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	for _, operator := range []Comparison{Less, LessOrEqual, Greater, GreaterOrEqual} {
		for _, literal := range []string{"100.0001", "-100.0001", "1e14", "-1e14", "1e75", "-1e75"} {
			t.Run(literal+"/"+string(rune('0'+operator)), func(t *testing.T) {
				average := &Average{Sum: Sum{Scope: Scope{Event: "Order Completed", Grain: "registered-message",
					Start: at.Add(-time.Hour), End: at}, Field: field, NativeCurrency: "EUR", Minimum: 2,
					Test: SumTest{Operator: operator, Value: decimal.MustParse(literal)}}}
				source := SourceBinding{Occurrences: warehouses.OccurrenceSource{Table: "events",
					Profile:   warehouses.Column{Name: "kpid", Type: types.UUID()},
					Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
					Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}},
					EventType:  warehouses.Column{Name: "type", Type: types.String()},
					Event:      warehouses.Column{Name: "event", Type: types.String()},
					Name:       warehouses.Column{Name: "name", Type: types.String(), Nullable: true},
					Properties: warehouses.Column{Name: "properties", Type: types.JSON()},
					Record: func(*state.Where) (warehouses.Expr, error) {
						return warehouses.NewBaseExpr(warehouses.Column{Name: "type", Type: types.String()},
							warehouses.OpIs, "track"), nil
					}}
				prepared, err := Prepare(&Rule{Average: average}, Bindings{}, at, source)
				if err != nil {
					t.Fatalf("expected prepared Average, got %v", err)
				}
				node, ok := prepared.Root.(*warehouses.DeterminedOccurrenceAverage)
				if !ok || node.Minimum != 2 || node.AtLeast != (operator == Greater || operator == GreaterOrEqual) ||
					node.Strict != (operator == Less || operator == Greater) {
					t.Fatalf("expected exact direction, strictness and minimum, got %+v", node)
				}
				if literal == "100.0001" || literal == "-100.0001" {
					if node.DeterminedResult != nil || !node.Threshold.Equal(average.Test.Value) {
						t.Fatalf("expected unchanged exact threshold %s, got %+v", literal, node)
					}
				} else {
					want := node.AtLeast == average.Test.Value.Less(decimal.Decimal{})
					if node.DeterminedResult == nil || *node.DeterminedResult != want ||
						!node.Threshold.Equal(decimal.Decimal{}) {
						t.Fatalf("expected predecided test %v without large product, got %+v", want, node)
					}
				}
				if !average.Test.Value.Equal(decimal.MustParse(literal)) {
					t.Fatalf("expected unchanged input %s, got %s", literal, average.Test.Value)
				}
			})
		}
	}
}

// TestAverageValidation preserves invalidity before unavailable capabilities.
func TestAverageValidation(t *testing.T) {
	field, err := ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected field fixture, got %v", err)
	}
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name        string
		modify      func(*Average)
		unsupported bool
	}{
		{"missing minimum", func(a *Average) { a.Minimum = 0 }, false},
		{"non-track", func(a *Average) { a.Scope.Type = "page" }, false},
		{"unknown field", func(a *Average) { a.Field = nil }, false},
		{"finer threshold", func(a *Average) { a.Test.Value = decimal.MustParse("0.00001") }, true},
		{"finer but invalid currency", func(a *Average) { a.Test.Value = decimal.MustParse("0.00001"); a.NativeCurrency = "eur" }, false},
		{"threshold beyond global precision", func(a *Average) { a.Test.Value = decimal.MustParse("1e76") }, false},
		{"equality", func(a *Average) { a.Test.Operator = Equal }, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			average := &Average{Sum: Sum{Scope: Scope{Event: "Order Completed", Grain: "registered-message",
				Start: at.Add(-time.Hour), End: at}, Field: field, NativeCurrency: "EUR", Minimum: 2,
				Test: SumTest{Operator: GreaterOrEqual, Value: decimal.MustParse("100")}}}
			test.modify(average)
			err := Validate(&Rule{Average: average}, at, Page{Limit: 10})
			if err != nil {
				if _, unsupported := errors.AsType[*UnsupportedError](err); unsupported != test.unsupported {
					t.Fatalf("expected unsupported=%v, got %v", test.unsupported, err)
				}
				return
			}
			t.Fatal("expected Average rejection, got success")
		})
	}
}
