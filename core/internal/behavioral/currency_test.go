// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package behavioral

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestNativeCurrencyReading keeps Count and Sum on the same governed reader.
func TestNativeCurrencyReading(t *testing.T) {

	currency, err := ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR", "Order Completed")
	if err != nil {
		t.Fatalf("expected resolved currency, got %v", err)
	}
	field, err := ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected monetary field, got %v", err)
	}
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	scope := Scope{Event: "Order Completed", Grain: "registered-message", Start: at.Add(-time.Hour), End: at}
	source := SourceBinding{Occurrences: warehouses.OccurrenceSource{Table: "events",
		Profile:   warehouses.Column{Name: "kpid", Type: types.UUID()},
		Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
		Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}},
		EventType:  warehouses.Column{Name: "type", Type: types.String()},
		Event:      warehouses.Column{Name: "event", Type: types.String()},
		Name:       warehouses.Column{Name: "name", Type: types.String(), Nullable: true},
		Properties: warehouses.Column{Name: "payload", Type: types.JSON()},
		Record: func(*state.Where) (warehouses.Expr, error) {
			return warehouses.NewBaseExpr(warehouses.Column{Name: "event", Type: types.String()},
				warehouses.OpIs, "Order Completed"), nil
		}}
	root := &Rule{Count: &Count{Scope: scope, Currency: currency}}
	count, err := Prepare(root, Bindings{}, at, source)
	if err != nil {
		t.Fatalf("expected prepared bound currency without caller bindings, got %v", err)
	}
	sum, err := (&Sum{Scope: scope, Field: field, NativeCurrency: "EUR",
		Test: SumTest{Operator: GreaterOrEqual, Value: decimal.MustParse("30")}}).prepare(at, source)
	if err != nil {
		t.Fatalf("expected prepared monetary Sum, got %v", err)
	}
	countReading := count.Root.(*warehouses.IndependentOccurrenceCount).
		Qualification.(*warehouses.QualificationGroup).Children[1].(*warehouses.KnownTextEqual)
	sumReading := sum.(*warehouses.IndependentOccurrenceSum).
		Qualification.(*warehouses.QualificationGroup).Children[1].(*warehouses.KnownTextEqual)
	if !reflect.DeepEqual(countReading, sumReading) || countReading.Column.Name != "payload" ||
		!slices.Equal(countReading.Keys, []string{"currency"}) || countReading.Value != "EUR" {
		t.Fatalf("expected identical bound readers, got %+v and %+v", countReading, sumReading)
	}
	countReading.Keys[0] = "mutated"
	if currency.field.path[1] != "currency" {
		t.Fatalf("expected immutable semantic path, got %v", currency.field.path)
	}

	// The resolved revision cannot be overridden by a caller-supplied binding.
	override := Bindings{Currencies: []CurrencyBinding{{Definition: currency.Definition,
		Revision: currency.Revision, Path: []string{"properties", "wrong"}, Evidence: IndependentOccurrenceValues}}}
	count, err = Prepare(root, override, at, source)
	if err != nil {
		t.Fatalf("expected immutable resolved currency reading, got %v", err)
	}
	reading := count.Root.(*warehouses.IndependentOccurrenceCount).
		Qualification.(*warehouses.QualificationGroup).Children[1].(*warehouses.KnownTextEqual)
	if !slices.Equal(reading.Keys, []string{"currency"}) {
		t.Fatalf("expected reference-owned currency path, got %v", reading.Keys)
	}

}

// TestNativeCurrencyPaths derives admission and reading from the same path.
func TestNativeCurrencyPaths(t *testing.T) {
	currency, err := ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR", "Order Completed")
	if err != nil {
		t.Fatalf("expected resolved currency, got %v", err)
	}
	currency.field.path = []string{"properties", "unit"}
	for _, test := range []struct {
		path        []string
		unsupported bool
	}{
		{[]string{"properties"}, true},
		{[]string{"properties", "unit"}, true},
		{[]string{"properties", "unit", "code"}, true},
		{[]string{"properties", "currency"}, false},
		{[]string{"properties", "unit_code"}, false},
		{[]string{"properties", "Unit"}, false},
		{[]string{"properties", "total"}, false},
	} {
		where := &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
			&state.Where{Operator: state.OpOr, Rules: []state.WhereRule{
				&state.WhereCondition{Property: test.path, Operator: state.OpExists},
			}},
		}}
		err := currency.ValidateWhere(where)
		if err != nil {
			if _, unsupported := errors.AsType[*UnsupportedError](err); !unsupported || !test.unsupported {
				t.Fatalf("expected unsupported=%v for %v, got %v", test.unsupported, test.path, err)
			}
			continue
		}
		if test.unsupported {
			t.Fatalf("expected dependent path %v rejected, got success", test.path)
		}
	}
}

// TestNativeCurrencyValidation protects internal admission of invalid inputs.
func TestNativeCurrencyValidation(t *testing.T) {
	for _, test := range []struct{ definition, revision, value, event string }{
		{"NativeCurrency", "1", "EUR", "Order Completed"},
		{"commerce.order_completed.currency", "2", "EUR", "Order Completed"},
		{"commerce.order_completed.currency", "1", "eur", "Order Completed"},
		{"commerce.order_completed.currency", "1", "EUR", "Refund"},
	} {
		_, err := ResolveNativeCurrency(test.definition, test.revision, test.value, test.event)
		if err != nil {
			if _, unsupported := errors.AsType[*UnsupportedError](err); unsupported {
				t.Fatalf("expected invalid currency reference, got %v", err)
			}
			continue
		}
		t.Fatalf("expected invalid currency reference %+v, got success", test)
	}
	for _, change := range []func(*Count){
		func(c *Count) { c.Scope.Event = "Refund" },
		func(c *Count) { c.Currency.Revision = "2" },
		func(c *Count) { c.Currency.Value = "eur" },
		func(c *Count) { c.Scope.End = c.Scope.Start },
	} {
		currency, err := ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR", "Order Completed")
		if err != nil {
			t.Fatalf("expected resolved test currency, got %v", err)
		}
		at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
		count := &Count{Scope: Scope{Event: "Order Completed", Grain: "registered-message",
			Start: at.Add(-time.Hour), End: at}, Currency: currency,
			Where: recordCountWhere(t, &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
				&state.WhereCondition{Property: []string{"properties", "currency"}, Operator: state.OpExists},
			}})}
		change(count)
		err = Validate(&Rule{Count: count}, at, Page{Limit: 1})
		if err != nil {
			if _, unsupported := errors.AsType[*UnsupportedError](err); unsupported {
				t.Fatalf("expected invalidity before dependent qualification, got %v", err)
			}
			continue
		}
		t.Fatal("expected invalid bound currency Count, got success")
	}
}

// TestNativeCurrencyWhereAdmission checks both internal entry points, including
// valid predicates with unavailable qualification dependencies.
func TestNativeCurrencyWhereAdmission(t *testing.T) {
	for _, test := range []struct {
		name                  string
		path                  []string
		resolved, unsupported bool
	}{
		{"resolved currency path", []string{"properties", "currency"}, true, true},
		{"resolved parent path", []string{"properties"}, true, true},
		{"resolved descendant path", []string{"properties", "currency", "code"}, true, true},
		{"resolved independent path", []string{"properties", "coupon"}, true, false},
		{"caller binding with where", []string{"properties", "coupon"}, false, true},
		{"caller binding without where", nil, false, false},
	} {
		for _, entry := range []string{"Validate", "Prepare"} {
			t.Run(test.name+"/"+entry, func(t *testing.T) {
				at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
				currency := &NativeCurrency{Definition: "NativeCurrency", Revision: "1", Value: "EUR"}
				bindings := Bindings{Currencies: []CurrencyBinding{{Definition: currency.Definition,
					Revision: currency.Revision, Path: []string{"properties", "currency"},
					Evidence: IndependentOccurrenceValues}}}
				if test.resolved {
					var err error
					currency, err = ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR", "Order Completed")
					if err != nil {
						t.Fatalf("expected resolved currency fixture, got %v", err)
					}
					bindings = Bindings{}
				}
				count := &Count{Scope: Scope{Event: "Order Completed", Grain: "registered-message",
					Start: at.Add(-time.Hour), End: at}, Currency: currency}
				if test.path != nil {
					count.Where = recordCountWhere(t, &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
						&state.WhereCondition{Property: test.path, Operator: state.OpExists},
					}})
				}
				source := SourceBinding{Occurrences: warehouses.OccurrenceSource{Table: "events",
					Profile:   warehouses.Column{Name: "kpid", Type: types.UUID()},
					Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
					Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}},
					EventType:  warehouses.Column{Name: "type", Type: types.String()},
					Event:      warehouses.Column{Name: "event", Type: types.String()},
					Name:       warehouses.Column{Name: "name", Type: types.String(), Nullable: true},
					Properties: warehouses.Column{Name: "properties", Type: types.JSON()},
					Record: func(*state.Where) (warehouses.Expr, error) {
						return warehouses.NewBaseExpr(warehouses.Column{Name: "event", Type: types.String()},
							warehouses.OpIs, "Order Completed"), nil
					}}
				root := &Rule{Count: count}
				var prepared warehouses.Filter
				var err error
				switch entry {
				case "Validate":
					err = Validate(root, at, Page{Limit: 1})
				case "Prepare":
					prepared, err = Prepare(root, bindings, at, source)
				default:
					t.Fatalf("expected known internal entry point, got %s", entry)
				}
				if err != nil {
					if _, unsupported := errors.AsType[*UnsupportedError](err); !test.unsupported || !unsupported {
						t.Fatalf("expected unsupported=%v, got %v", test.unsupported, err)
					}
					if prepared.Root != nil {
						t.Fatalf("expected no partially prepared filter, got %+v", prepared)
					}
					return
				}
				if test.unsupported {
					t.Fatal("expected unavailable currency qualification, got success")
				}
				if entry == "Prepare" && prepared.Root == nil {
					t.Fatal("expected valid fixture to compile completely, got an empty filter")
				}
			})
		}
	}
}
