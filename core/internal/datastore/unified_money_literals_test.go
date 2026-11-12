// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
)

type moneyLiteralObservation struct {
	amount                        string
	currency, qualification, flag int
}

// TestUnifiedMoneyLiteralsPostgreSQL checks exact literal partitions in SQL.
// Unknown amounts may occupy any partition, even outside the readable grid.
// Selections enumerate membership subsets and permutations of timestamp ties.
func TestUnifiedMoneyLiteralsPostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	execUnifiedFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,timestamp timestamp,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	field, err := unified.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected monetary field, got %v", err)
	}
	var observations []moneyLiteralObservation
	var payloads []string
	for _, currency := range []struct {
		payload string
		value   int
	}{{`"EUR"`, 0}, {`"USD"`, 1}, {`"eur"`, 2}, {``, 3}, {`null`, 3}, {`7`, 3}} {
		for _, amount := range []string{"-99999999999999.9999", "-100.0001", "-0", "0", "50", "100", "100.0",
			"1E2", "100.0001", "200", "99999999999999.9999", "", "null", `"100"`, "100.00001",
			"1e14", "-1e14", "1e20", "-1e20", "false", "{}", "[]"} {
			for qualification := range 3 {
				flag := len(observations) / 3 % 3
				properties := []string{}
				if currency.payload != "" {
					properties = append(properties, `"currency":`+currency.payload)
				}
				if amount != "" {
					properties = append(properties, `"total":`+amount)
				}
				if qualification < 2 {
					properties = append(properties, `"q":"`+[]string{"no", "yes"}[qualification]+`"`)
				}
				if flag < 2 {
					properties = append(properties, fmt.Sprintf(`"flag":%v`, flag == 1))
				}
				observations = append(observations, moneyLiteralObservation{amount, currency.value, qualification, flag})
				payloads = append(payloads, "{"+strings.Join(properties, ",")+"}")
			}
		}
	}
	var profiles, events []string
	for i := range len(observations) + 1 {
		profiles = append(profiles, "('"+unifiedTestID(i+1)+"')")
		if i == len(observations) {
			continue
		}
		for j, index := range []int{i, (i*37 + 17) % len(observations)} {
			timestamp := index%3 + 1
			if j == 1 && i%2 == 0 {
				timestamp = i%3 + 1
			}
			row := fmt.Sprintf("('%s','%d/%d','track','Order Completed','2026-01-0%d','%s')",
				unifiedTestID(i+1), i, j, timestamp, payloads[index])
			events = append(events, row, row)
		}
	}
	execUnifiedFixture(t, dw, "INSERT INTO profiles VALUES "+strings.Join(profiles, ","))
	execUnifiedFixture(t, dw, "INSERT INTO krenalis_events VALUES "+strings.Join(events, ","))
	qualification := &state.WhereCondition{Property: []string{"properties", "q"}, Operator: state.OpIs,
		Values: []any{json.Value(`"yes"`)}}

	for _, literal := range []struct {
		operator     unified.Comparison
		value, upper string
	}{
		{unified.Equal, "100", ""}, {unified.Equal, "100.00001", ""},
		{unified.Equal, "1e14", ""}, {unified.Equal, "-1e20", ""},
		{unified.Equal, "-99999999999999.9999", ""}, {unified.Equal, "99999999999999.9999", ""},
		{unified.Different, "100", ""}, {unified.Different, "100.00001", ""},
		{unified.Different, "-1e14", ""}, {unified.Different, "1e20", ""},
		{unified.Between, "50", "100"}, {unified.Between, "100", "100"},
		{unified.Between, "99.99995", "100.00004"}, {unified.Between, "100.00001", "100.00009"},
		{unified.Between, "-1e20", "1e20"}, {unified.Between, "1e14", "1e20"},
		{unified.Between, "-1e20", "-1e14"}, {unified.Between, "-100.00009", "-50.00001"},
		{unified.Between, "-1e14", "-99999999999999.9999"},
		{unified.Between, "99999999999999.9999", "1e14"},
	} {
		t.Run(fmt.Sprintf("%d/%s/%s", literal.operator, literal.value, literal.upper), func(t *testing.T) {
			test := unified.SumTest{Operator: literal.operator, Value: decimal.MustParse(literal.value)}
			if literal.upper != "" {
				test.Upper = decimal.MustParse(literal.upper)
			}
			for _, currency := range []string{"", "EUR", "USD"} {
				for _, mode := range []string{"money", "or"} {
					var baseCurrency *unified.NativeCurrency
					if currency != "" {
						baseCurrency, err = unified.ResolveNativeCurrency("commerce.order_completed.currency", "1",
							currency, "Order Completed")
						if err != nil {
							t.Fatalf("expected governed base currency, got %v", err)
						}
					}
					body := unified.OccurrencePredicate{Money: &unified.Money{Field: field, NativeCurrency: "EUR", Test: test}}
					if mode == "or" {
						body = unified.OccurrencePredicate{Operator: state.OpOr, Children: []unified.OccurrencePredicate{
							body, {Record: &state.WhereCondition{Property: []string{"properties", "flag"}, Operator: state.OpIsTrue}},
						}}
					}
					for _, kind := range []string{"count", "absence", "first", "last", "nth"} {
						scope := unified.Scope{Event: "Order Completed", Grain: "registered-message",
							Start: at.AddDate(0, -1, 0), End: at}
						root := &unified.Rule{}
						terminal := unified.CountTest{Operator: unified.GreaterOrEqual, Value: 1}
						if kind == "count" || kind == "absence" {
							if kind == "absence" {
								terminal = unified.CountTest{Operator: unified.Equal}
							}
							root.Count = &unified.Count{Scope: scope, Currency: baseCurrency, Test: terminal,
								Where: &unified.OccurrencePredicate{Children: []unified.OccurrencePredicate{{Record: qualification}, body}}}
						} else {
							root.Experience = &unified.Experience{Scope: scope, Currency: baseCurrency, First: kind == "first",
								Where: &state.Where{Rules: []state.WhereRule{qualification}}, Verify: &body}
							if kind == "nth" {
								root.Experience.Nth = 2
							}
						}
						want := make([]unified.Truth, len(observations)+1)
						for i := range want {
							var flags []int
							var history []nthOracleOccurrence
							if i < len(observations) {
								for j, index := range []int{i, (i*37 + 17) % len(observations)} {
									count, base, verify := moneyLiteralOracle(observations[index], test, currency, mode)
									flags = append(flags, count)
									timestamp := index%3 + 1
									if j == 1 && i%2 == 0 {
										timestamp = i%3 + 1
									}
									if kind == "last" {
										timestamp = -timestamp
									}
									history = append(history, nthOracleOccurrence{timestamp, base, verify})
								}
							}
							if root.Count != nil {
								want[i] = independentCountOracle(flags, 0, terminal)
							} else {
								position := 1
								if kind == "nth" {
									position = 2
								}
								want[i], _ = enumerateNth(history, position)
							}
						}
						result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 1000})
						if err != nil {
							t.Fatalf("expected exact %s currency=%s mode=%s literal result, got %v", kind, currency, mode, err)
						}
						assertUnifiedTruths(t, result, want)
					}
				}
			}
		})
	}

}

// moneyLiteralOracle enumerates the Boolean partitions of one exact predicate.
// It never rounds a literal or restricts unknown values to the readable domain.
func moneyLiteralOracle(observation moneyLiteralObservation, test unified.SumTest, currency, mode string) (int, int, int) {

	lower, _ := new(big.Rat).SetString(test.Value.String())
	upper, _ := new(big.Rat).SetString(test.Upper.String())
	amount, readable := new(big.Rat).SetString(observation.amount)
	maximum, _ := new(big.Rat).SetString("99999999999999.9999")
	if readable {
		scaled := new(big.Rat).Mul(amount, big.NewRat(10000, 1))
		readable = scaled.IsInt() && amount.Cmp(maximum) <= 0 && amount.Cmp(new(big.Rat).Neg(maximum)) >= 0
	}
	amounts := []*big.Rat{amount}
	if !readable {
		amounts = []*big.Rat{lower, new(big.Rat).Sub(lower, big.NewRat(1, 1)), new(big.Rat).Add(lower, big.NewRat(1, 1))}
		if test.Operator == unified.Between {
			amounts = append(amounts, upper, new(big.Rat).Add(upper, big.NewRat(1, 1)))
		}
	}

	seen := [3][2]bool{}
	for c := range 3 {
		for q := range 2 {
			for flag := range 2 {
				if observation.currency < 3 && observation.currency != c ||
					observation.qualification < 2 && observation.qualification != q ||
					observation.flag < 2 && observation.flag != flag {
					continue
				}
				for _, value := range amounts {
					pass := value.Cmp(lower) == 0
					switch test.Operator {
					case unified.Equal:
						// Exact numerical equality, independent of JSON spelling.
					case unified.Different:
						pass = !pass
					case unified.Between:
						pass = value.Cmp(lower) >= 0 && value.Cmp(upper) <= 0
					case unified.Greater:
						pass = value.Cmp(lower) > 0
					case unified.GreaterOrEqual:
						pass = value.Cmp(lower) >= 0
					case unified.Less:
						pass = value.Cmp(lower) < 0
					case unified.LessOrEqual:
						pass = value.Cmp(lower) <= 0
					default:
						panic(fmt.Sprintf("core/internal/datastore: unexpected literal oracle operator %d", test.Operator))
					}
					verify := c == 0 && pass
					if mode == "or" {
						verify = verify || flag == 1
					}
					base := q == 1 && (currency == "" || currency == "EUR" && c == 0 || currency == "USD" && c == 1)
					for i, truth := range []bool{base && verify, base, verify} {
						if i == 2 && !base {
							continue
						}
						index := 0
						if truth {
							index = 1
						}
						seen[i][index] = true
					}
				}
			}
		}
	}

	states := [3]int{}
	for i, outcomes := range seen {
		if outcomes[0] && outcomes[1] {
			states[i] = 2
		} else if outcomes[1] {
			states[i] = 1
		}
	}

	return states[0], states[1], states[2]
}
