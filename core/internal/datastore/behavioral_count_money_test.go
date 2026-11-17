// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
)

type countMoneyObservation struct {
	payload                  string
	currency, amount, record int
}

// TestBehavioralCountMoneyOraclePostgreSQL enumerates each occurrence's values
// before using the independent Count oracle, including Kleene sample guards.
func TestBehavioralCountMoneyOraclePostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	execBehavioralFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,timestamp timestamp,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	field, err := behavioral.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected monetary field, got %v", err)
	}
	var observations []countMoneyObservation
	for _, currency := range []struct {
		payload string
		value   int
	}{{`"currency":"EUR"`, 0}, {`"currency":"USD"`, 1}, {`"currency":"eur"`, 2},
		{"", 3}, {`"currency":null`, 3}, {`"currency":7`, 3}} {
		for _, amount := range []struct {
			payload string
			value   int
		}{{`"total":80`, 0}, {`"total":100`, 1}, {`"total":150`, 2},
			{`"total":99999999999999.9999`, 2}, {`"total":1e14`, 3}, {`"total":1e20`, 3},
			{`"total":100.00001`, 3}, {"", 3}, {`"total":null`, 3}, {`"total":"150"`, 3}, {`"total":false`, 3}} {
			for record := range 3 {
				properties := []string{}
				for _, property := range []string{currency.payload, amount.payload} {
					if property != "" {
						properties = append(properties, property)
					}
				}
				if record < 2 {
					properties = append(properties, fmt.Sprintf(`"flag":%v`, record == 1))
				}
				observations = append(observations, countMoneyObservation{"{" + strings.Join(properties, ",") + "}",
					currency.value, amount.value, record})
			}
		}
	}
	var profiles, events []string
	for i := range len(observations) + 1 {
		profiles = append(profiles, "('"+behavioralTestID(i+1)+"')")
		if i == len(observations) {
			continue // A profile without events must survive the population join.
		}
		for j, index := range []int{i, (i*37 + 11) % len(observations)} {
			row := fmt.Sprintf("('%s','%d/%d','track','Order Completed','2026-01-15','%s')",
				behavioralTestID(i+1), i, j, observations[index].payload)
			events = append(events, row, row) // Physical copies do not add occurrences.
		}
	}
	execBehavioralFixture(t, dw, "INSERT INTO profiles VALUES "+strings.Join(profiles, ","))
	execBehavioralFixture(t, dw, "INSERT INTO krenalis_events VALUES "+strings.Join(events, ","))

	for _, comparison := range []behavioral.Comparison{
		behavioral.Greater, behavioral.GreaterOrEqual, behavioral.Less, behavioral.LessOrEqual,
	} {
		for _, outerCurrency := range []string{"", "EUR", "USD"} {
			for _, mode := range []string{"money", "and", "or"} {
				t.Run(fmt.Sprintf("comparison=%d/currency=%s/%s", comparison, outerCurrency, mode), func(t *testing.T) {

					qualify := func(observation countMoneyObservation) int {
						seen := [2]bool{}
						for currency := range 3 {
							for amount := range 3 {
								for record := range 2 {
									if observation.currency < 3 && observation.currency != currency ||
										observation.amount < 3 && observation.amount != amount ||
										observation.record < 2 && observation.record != record {
										continue
									}
									var amountPass bool
									switch comparison {
									case behavioral.Greater:
										amountPass = amount == 2
									case behavioral.GreaterOrEqual:
										amountPass = amount >= 1
									case behavioral.Less:
										amountPass = amount == 0
									case behavioral.LessOrEqual:
										amountPass = amount <= 1
									default:
										t.Fatalf("expected ordered comparison, got %d", comparison)
									}
									pass := currency == 0 && amountPass
									switch mode {
									case "money":
										// No additional Record predicate.
									case "and":
										pass = pass && record == 1
									case "or":
										pass = pass || record == 1
									default:
										t.Fatalf("expected known where mode, got %q", mode)
									}
									if outerCurrency != "" {
										pass = pass && (outerCurrency == "EUR" && currency == 0 || outerCurrency == "USD" && currency == 1)
									}
									outcome := 0
									if pass {
										outcome = 1
									}
									seen[outcome] = true
								}
							}
						}
						if seen[0] && seen[1] {
							return 2
						}
						if seen[1] {
							return 1
						}
						return 0
					}

					where := &behavioral.OccurrencePredicate{Children: []behavioral.OccurrencePredicate{
						{Money: &behavioral.Money{Field: field, NativeCurrency: "EUR",
							Test: behavioral.SumTest{Operator: comparison, Value: decimal.MustParse("100")}}}}}
					if mode != "money" {
						where.Children = append(where.Children, behavioral.OccurrencePredicate{Record: &state.WhereCondition{
							Property: []string{"properties", "flag"}, Operator: state.OpIsTrue}})
						if mode == "or" {
							where.Operator = state.OpOr
						}
					}
					count := &behavioral.Count{Scope: behavioral.Scope{Event: "Order Completed", Grain: "registered-message",
						Start: at.AddDate(0, -1, 0), End: at}, Where: where}
					if outerCurrency != "" {
						count.Currency, err = behavioral.ResolveNativeCurrency("commerce.order_completed.currency", "1",
							outerCurrency, "Order Completed")
						if err != nil {
							t.Fatalf("expected resolved outer currency, got %v", err)
						}
					}
					for _, test := range []struct {
						terminal behavioral.CountTest
						minimum  uint64
					}{
						{behavioral.CountTest{Operator: behavioral.GreaterOrEqual, Value: 2}, 0},
						{behavioral.CountTest{Operator: behavioral.Equal}, 0},
						{behavioral.CountTest{Operator: behavioral.Between, Value: 1, Upper: 2}, 0},
						{behavioral.CountTest{Operator: behavioral.GreaterOrEqual, Value: 2}, 2},
						{behavioral.CountTest{Operator: behavioral.Different, Value: 2}, 2},
					} {
						count.Test, count.Minimum = test.terminal, test.minimum
						want := make([]behavioral.Truth, len(observations)+1)
						for i := range want {
							var states []int
							if i < len(observations) {
								states = []int{qualify(observations[i]), qualify(observations[(i*37+11)%len(observations)])}
							}
							want[i] = independentCountOracle(states, test.minimum, test.terminal)
						}
						result, err := store.QueryProfiles(t.Context(), &behavioral.Rule{Count: count},
							behavioral.Bindings{}, at,
							behavioral.Page{Limit: 1000})
						if err != nil {
							t.Fatalf("expected exact money-qualified Count, got %v", err)
						}
						assertBehavioralTruths(t, result, want)
					}

				})
			}
		}
	}

}

// TestBehavioralCountMoneyPostgreSQL checks currency implications and copies.
func TestBehavioralCountMoneyPostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	execBehavioralFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
INSERT INTO profiles VALUES ('`+behavioralTestID(1)+`'),('`+behavioralTestID(2)+`');
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,timestamp timestamp,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	field, err := behavioral.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected monetary field, got %v", err)
	}
	currency, err := behavioral.ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR",
		"Order Completed")
	if err != nil {
		t.Fatalf("expected governed currency, got %v", err)
	}
	money := &behavioral.Money{Field: field, NativeCurrency: "EUR",
		Test: behavioral.SumTest{Operator: behavioral.GreaterOrEqual, Value: decimal.MustParse("100")}}
	where := &behavioral.OccurrencePredicate{Children: []behavioral.OccurrencePredicate{{Money: money}}}
	count := &behavioral.Count{Scope: behavioral.Scope{Event: "Order Completed", Grain: "registered-message",
		Start: at.AddDate(0, -1, 0), End: at}, Where: where, Test: behavioral.CountTest{Operator: behavioral.Equal}}
	root := &behavioral.Rule{Count: count}
	for _, test := range []struct {
		name, first, next string
		fault             bool
		truth             behavioral.Truth
	}{
		{"identical", `{"currency":"EUR","total":150}`, `{"currency":"EUR","total":150}`, false, behavioral.NoMatch},
		{"different raw same outcome", `{"currency":"EUR","total":150}`, `{"currency":"EUR","total":200}`, false, behavioral.NoMatch},
		{"qualification conflict", `{"currency":"EUR","total":150}`, `{"currency":"EUR","total":80}`, true, 0},
		{"certain unknown conflict", `{"currency":"EUR","total":150}`, `{"currency":"EUR"}`, true, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			execBehavioralFixture(t, dw, "TRUNCATE krenalis_events")
			for _, payload := range []string{test.first, test.next} {
				execBehavioralFixture(t, dw, "INSERT INTO krenalis_events VALUES ('"+behavioralTestID(1)+
					"','same','track','Order Completed','2026-01-15','"+payload+"')")
			}
			result, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
			if err != nil {
				if !test.fault || result != nil || !strings.Contains(err.Error(), "occurrence source violates") {
					t.Fatalf("expected source fault=%v without partial result, got %+v, %v", test.fault, result, err)
				}
			}
			if err == nil {
				if test.fault {
					t.Fatal("expected copy conflict, got success")
				}
				assertBehavioralTruths(t, result, []behavioral.Truth{test.truth, behavioral.Match})
			}
			other, err := store.ExplainProfileFilter(t.Context(), root, behavioral.Bindings{}, at, behavioralTestID(2))
			if err != nil {
				t.Fatalf("expected unrelated explanation independent of source fault, got %v", err)
			}
			if other.Nodes[0].Truth != behavioral.Match {
				t.Fatalf("expected MATCH for empty history, got %+v", other.Nodes)
			}
		})
	}

	t.Run("currency conditioning also precedes copy checks", func(t *testing.T) {
		execBehavioralFixture(t, dw, `TRUNCATE krenalis_events;
INSERT INTO krenalis_events VALUES
('`+behavioralTestID(1)+`','same','track','Order Completed','2026-01-15','{"currency":"GBP","total":150,"flag":false}'),
('`+behavioralTestID(1)+`','same','track','Order Completed','2026-01-15','{"total":150,"flag":false}')`)
		count.Currency, money.NativeCurrency = currency, "USD"
		where.Operator = state.OpOr
		where.Children = append(where.Children, behavioral.OccurrencePredicate{Record: &state.WhereCondition{
			Property: []string{"properties", "flag"}, Operator: state.OpIsTrue}})
		result, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
		if err != nil {
			t.Fatalf("expected both copies certainly excluded, got %v", err)
		}
		assertBehavioralTruths(t, result, []behavioral.Truth{behavioral.Match, behavioral.Match})
		explained, err := store.ExplainProfileFilter(t.Context(), root, behavioral.Bindings{}, at, behavioralTestID(1))
		if err != nil {
			t.Fatalf("expected no spurious copy fault, got %v", err)
		}
		if explained.Nodes[0].Count.Certain != 0 || explained.Nodes[0].Count.Possible != 0 {
			t.Fatalf("expected exact zero Count bounds, got %+v", explained.Nodes[0].Count)
		}
	})

	t.Run("equivalence only for necessary monetary requirement", func(t *testing.T) {
		execBehavioralFixture(t, dw, `TRUNCATE krenalis_events;
INSERT INTO krenalis_events VALUES
('`+behavioralTestID(1)+`','eur','track','Order Completed','2026-01-15','{"total":150,"flag":true}'),
('`+behavioralTestID(2)+
			`','promo','track','Order Completed','2026-01-15','{"currency":"USD","total":150,"flag":true}')`)
		money.NativeCurrency = "EUR"
		for _, mode := range []string{"money", "and", "or"} {
			where.Operator, where.Children = state.OpAnd, []behavioral.OccurrencePredicate{{Money: money}}
			if mode != "money" {
				where.Children = append(where.Children, behavioral.OccurrencePredicate{Record: &state.WhereCondition{
					Property: []string{"properties", "flag"}, Operator: state.OpIsTrue}})
				if mode == "or" {
					where.Operator = state.OpOr
				}
			}
			count.Currency = nil
			without, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{
				Limit: 10})
			if err != nil {
				t.Fatalf("expected where without outer currency, got %v", err)
			}
			count.Currency = currency
			with, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
			if err != nil {
				t.Fatalf("expected where with outer currency, got %v", err)
			}
			if mode == "or" {
				assertBehavioralTruths(t, without, []behavioral.Truth{behavioral.NoMatch, behavioral.NoMatch})
				assertBehavioralTruths(t, with, []behavioral.Truth{behavioral.Unknown, behavioral.Match})
				continue
			}
			if !reflect.DeepEqual(without, with) {
				t.Fatal("expected equivalent necessary monetary qualification, got different results")
			}
		}
	})

}
