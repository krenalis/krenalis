// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

// testUnifiedConformanceCountMoney compares bounded monetary predicates with
// independent completions, including Count's external currency restriction.
func testUnifiedConformanceCountMoney(t *testing.T, store *Store, dw warehouses.Warehouse, table warehouses.Table) {

	t.Helper()
	execUnifiedFixture(t, dw, "TRUNCATE TABLE profiles; TRUNCATE TABLE krenalis_events")
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	field, err := unified.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected monetary field, got %v", err)
	}
	// This bounded sample uses the same independent completion oracle as the
	// exhaustive PostgreSQL test. Currency and amount states are fixture data.
	observations := []moneyLiteralObservation{
		{"100", 0, 1, 0}, {"50", 0, 1, 0}, {"150", 0, 1, 1},
		{"150", 1, 1, 1}, {"150", 2, 1, 0}, {"150", 3, 1, 0},
		{"80", 3, 1, 0}, {"", 0, 1, 0}, {"null", 0, 1, 2},
		{`"100"`, 0, 1, 1}, {"100.00001", 0, 1, 0}, {"1e14", 0, 1, 0},
		{"1E2", 0, 1, 0}, {"-100", 0, 1, 0}, {"100", 0, 0, 1},
		{"100", 0, 2, 2}, {"150", 1, 2, 1}, {"150", 3, 1, 1},
		{"false", 0, 1, 0},
	}
	var profiles []string
	var rows [][]any
	for p := range len(observations) + 1 {
		profiles = append(profiles, "('"+unifiedTestID(p+1)+"')")
		if p == len(observations) {
			continue // Preserve one profile without any event.
		}
		for i, index := range []int{p, (p + 5) % len(observations)} {
			observation := observations[index]
			var properties []string
			if observation.currency < 3 {
				properties = append(properties, `"currency":"`+[]string{"EUR", "USD", "eur"}[observation.currency]+`"`)
			} else if index%2 == 0 {
				properties = append(properties, `"currency":null`)
			}
			if observation.amount != "" {
				properties = append(properties, `"total":`+observation.amount)
			}
			if observation.qualification < 2 {
				properties = append(properties, `"q":"`+[]string{"no", "yes"}[observation.qualification]+`"`)
			}
			if observation.flag < 2 {
				properties = append(properties, fmt.Sprintf(`"flag":%t`, observation.flag == 1))
			}
			rows = append(rows, []any{unifiedTestID(p + 1), fmt.Sprintf("money/%d/%d", p, i), "track",
				"Order Completed", nil, at.Add(-time.Hour), at, at,
				json.Value("{" + strings.Join(properties, ",") + "}")})
		}
	}
	execUnifiedFixture(t, dw, "INSERT INTO profiles VALUES "+strings.Join(profiles, ","))
	err = dw.Merge(t.Context(), table, rows, nil)
	if err != nil {
		t.Fatalf("expected recorded Count money fixtures, got %v", err)
	}
	execUnifiedFixture(t, dw, "INSERT INTO krenalis_events SELECT * FROM krenalis_events")

	for _, literal := range []struct {
		name         string
		operator     unified.Comparison
		lower, upper string
	}{
		{"is", unified.Equal, "100", "0"},
		{"is not", unified.Different, "100", "0"},
		{"between", unified.Between, "50", "100"},
		{"unreadable equality", unified.Equal, "100.00001", "0"},
		{"unreadable inequality", unified.Different, "100.00001", "0"},
		{"empty interval", unified.Between, "100.00001", "100.00009"},
		{"whole readable domain", unified.Between, "-1e20", "1e20"},
		{"greater", unified.Greater, "100", "0"},
		{"at least", unified.GreaterOrEqual, "100", "0"},
		{"less", unified.Less, "100", "0"},
		{"at most", unified.LessOrEqual, "100", "0"},
		{"above readable domain", unified.Greater, "1e20", "0"},
		{"below readable domain", unified.GreaterOrEqual, "-1e20", "0"},
	} {
		for _, currency := range []string{"", "EUR", "USD"} {
			for _, mode := range []string{"money", "or"} {
				t.Run(literal.name+"/"+currency+"/"+mode, func(t *testing.T) {

					test := unified.SumTest{Operator: literal.operator, Value: decimal.MustParse(literal.lower),
						Upper: decimal.MustParse(literal.upper)}
					body := unified.OccurrencePredicate{Money: &unified.Money{Field: field,
						NativeCurrency: "EUR", Test: test}}
					if mode == "or" {
						body = unified.OccurrencePredicate{Operator: state.OpOr,
							Children: []unified.OccurrencePredicate{body, {Record: &state.WhereCondition{
								Property: []string{"properties", "flag"}, Operator: state.OpIsTrue}}}}
					}
					count := &unified.Count{Scope: unified.Scope{Event: "Order Completed", Grain: "registered-message",
						Start: at.Add(-24 * time.Hour), End: at}, Where: &unified.OccurrencePredicate{
						Children: []unified.OccurrencePredicate{{Record: &state.WhereCondition{
							Property: []string{"properties", "q"}, Operator: state.OpIs,
							Values: []any{json.Value(`"yes"`)}}}, body}}}
					if currency != "" {
						count.Currency, err = unified.ResolveNativeCurrency("commerce.order_completed.currency",
							"1", currency, "Order Completed")
						if err != nil {
							t.Fatalf("expected Count currency binding, got %v", err)
						}
					}
					states := make([][]int, len(observations)+1)
					for p := range observations {
						for _, index := range []int{p, (p + 5) % len(observations)} {
							qualified, _, _ := moneyLiteralOracle(observations[index], test, currency, mode)
							states[p] = append(states[p], qualified)
						}
					}
					for _, terminal := range []struct {
						test    unified.CountTest
						minimum uint64
					}{
						{unified.CountTest{Operator: unified.GreaterOrEqual, Value: 1}, 0},
						{unified.CountTest{Operator: unified.Equal}, 0},
						{unified.CountTest{Operator: unified.Different, Value: 2}, 2},
					} {
						count.Test, count.Minimum = terminal.test, terminal.minimum
						want := make([]unified.Truth, len(states))
						for p, values := range states {
							want[p] = independentCountOracle(values, terminal.minimum, terminal.test)
						}
						root := &unified.Rule{Count: count}
						result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at,
							unified.Page{Limit: 100})
						if err != nil {
							t.Fatalf("expected exact money-qualified Count, got %v", err)
						}
						assertUnifiedTruths(t, result, want)
						if !result.At.Equal(at) {
							t.Fatalf("expected evaluation at %s, got %s", at, result.At)
						}
						// Check every profile's metadata for one whole fixture,
						// without multiplying remote explanation queries.
						if literal.name == "is" && currency == "USD" && mode == "or" && terminal.minimum == 2 {
							for p, values := range states {
								var certain, possible int64
								for _, value := range values {
									if value == 1 {
										certain++
									}
									if value != 0 {
										possible++
									}
								}
								explained, err := store.ExplainUnifiedProfile(t.Context(), root,
									unified.Bindings{}, at, unifiedTestID(p+1))
								if err != nil {
									t.Fatalf("expected Count explanation, got %v", err)
								}
								if explained == nil || len(explained.Nodes) != 1 ||
									explained.Nodes[0].Truth != want[p] || explained.Nodes[0].Count == nil ||
									*explained.Nodes[0].Count != (warehouses.OccurrenceCountBounds{
										Certain: certain, Possible: possible}) {
									t.Fatalf("expected Count truth %v and bounds [%d,%d], got %+v",
										want[p], certain, possible, explained)
								}
							}
						}
					}

				})
			}
		}
	}

}
