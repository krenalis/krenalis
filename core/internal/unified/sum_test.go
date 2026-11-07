// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"math/big"
	"slices"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestMonetaryFieldPaths keeps reading and qualification admission in sync.
func TestMonetaryFieldPaths(t *testing.T) {

	field, err := ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected resolved monetary field, got %v", err)
	}
	// Change the test binding to detect a separate hard-coded path blacklist.
	field.amountPath = []string{"properties", "amount"}
	field.currencyPath = []string{"properties", "unit"}
	for _, test := range []struct {
		path        []string
		unsupported bool
	}{
		{[]string{"properties"}, true},
		{[]string{"properties", "amount"}, true},
		{[]string{"properties", "unit", "code"}, true},
		{[]string{"properties", "total"}, false},
		{[]string{"properties", "currency"}, false},
	} {
		where := &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
			&state.WhereCondition{Property: test.path, Operator: state.OpExists},
		}}
		err := field.ValidateWhere(where)
		if err != nil {
			if _, unsupported := errors.AsType[*UnsupportedError](err); !test.unsupported || !unsupported {
				t.Fatalf("expected unsupported=%v for %v, got %v", test.unsupported, test.path, err)
			}
			continue
		}
		if test.unsupported {
			t.Fatalf("expected unavailable path %v, got success", test.path)
		}
	}

	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	sum := Sum{Scope: Scope{Event: "Order Completed", Grain: "registered-message",
		Start: at.AddDate(0, -1, 0), End: at}, Field: field, NativeCurrency: "EUR",
		Test: SumTest{Operator: GreaterOrEqual, Value: decimal.MustParse("30")}}
	source := SourceBinding{Occurrences: warehouses.OccurrenceSource{Table: "events",
		Profile:   warehouses.Column{Name: "kpid", Type: types.UUID()},
		Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
		Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}},
		Properties: warehouses.Column{Name: "payload", Type: types.JSON()},
		Record: func(*state.Where) (warehouses.Expr, error) {
			return warehouses.NewBaseExpr(warehouses.Column{Name: "event", Type: types.String()},
				warehouses.OpIs, "Order Completed"), nil
		}}
	prepared, err := sum.prepare(at, source)
	if err != nil {
		t.Fatalf("expected prepared monetary reading, got %v", err)
	}
	reading := prepared.(*warehouses.IndependentOccurrenceSum)
	currency := reading.Qualification.(*warehouses.QualificationGroup).Children[1].(*warehouses.KnownTextEqual)
	if !slices.Equal(reading.Value.Keys, []string{"amount"}) ||
		!slices.Equal(currency.Keys, []string{"unit"}) || reading.Value.Column.Name != "payload" ||
		currency.Column.Name != "payload" || !types.Equal(reading.Value.Type, field.amountType) {
		t.Fatalf("expected paths and type from the resolved reading, got %+v and %+v", reading.Value, currency)
	}
	reading.Value.Keys[0], currency.Keys[0] = "changed", "changed"
	if field.amountPath[1] != "amount" || field.currencyPath[1] != "unit" {
		t.Fatalf("expected immutable reading paths, got %v and %v", field.amountPath, field.currencyPath)
	}

}

// TestSumThreshold compares normalized tests against exact rational arithmetic.
func TestSumThreshold(t *testing.T) {
	for _, literal := range []string{"0", "-0", "1e-37", "-1e-37", "30.00001", "-30.00001",
		"30.00000", "-30", "1e75", "-1e75", "99999999999999.9999", "1000.0000", "5000.0000",
		"-2000.0000", "999.9999", "1000.0001", "1e33", "-1e33"} {
		for _, op := range []Comparison{Less, LessOrEqual, Greater, GreaterOrEqual} {
			test := SumTest{Operator: op, Value: decimal.MustParse(literal)}
			threshold, atLeast, err := test.threshold()
			if err != nil {
				t.Fatalf("expected normalized threshold, got %v", err)
			}
			precision, scale := threshold.PrecisionScale()
			if precision > 38 || scale > 4 {
				t.Fatalf("expected exact backend-representable threshold, got %s", threshold)
			}
			for _, value := range []string{"-100000000000000000000000000000000", "-30.0001", "-30", "-0.0001", "0",
				"0.0001", "29.9999", "30", "30.0001", "999.9999", "1000", "1500", "5000",
				"-2000", "100000000000000000000000000000000"} {
				a, _ := new(big.Rat).SetString(value)
				b, _ := new(big.Rat).SetString(literal)
				comparison := a.Cmp(b)
				want := false
				switch op {
				case Less:
					want = comparison < 0
				case LessOrEqual:
					want = comparison <= 0
				case Greater:
					want = comparison > 0
				case GreaterOrEqual:
					want = comparison >= 0
				default:
					t.Fatalf("expected ordered comparison, got %v", op)
				}
				n := decimal.MustParse(value)
				got := n.LessEqual(threshold)
				if atLeast {
					got = n.GreaterEqual(threshold)
				}
				if got != want {
					t.Fatalf("expected %s op%d %s = %v, got %v at normalized %s", value, op, literal, want, got, threshold)
				}
			}
		}
	}
}
