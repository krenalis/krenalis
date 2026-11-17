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

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

type pairOracleOccurrence struct {
	timestamp  int
	membership int // 0 excluded, 1 certain, 2 possible
	amount     string
}

// TestBehavioralLastPreviousCopiesPostgreSQL checks faults and profile
// isolation.
func TestBehavioralLastPreviousCopiesPostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	execBehavioralFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,timestamp timestamp,properties jsonb);
INSERT INTO profiles VALUES ('00000000-0000-0000-0000-000000000001'),('00000000-0000-0000-0000-000000000002')`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	field, err := behavioral.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected field binding, got %v", err)
	}
	currency, err := behavioral.ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR",
		"Order Completed")
	if err != nil {
		t.Fatalf("expected currency binding, got %v", err)
	}
	pair := &behavioral.LastVersusPrevious{Scope: behavioral.Scope{Event: "Order Completed", Grain: "registered-message",
		Start: at.AddDate(0, -1, 0), End: at}, Currency: currency, Field: field,
		Check: behavioral.LastPreviousCheck{Kind: behavioral.LastPreviousDifference,
			Test: behavioral.SumTest{Operator: behavioral.LessOrEqual, Value: decimal.MustParse("-30")}}}
	root := &behavioral.Rule{LastVersusPrevious: pair}
	for _, test := range []struct {
		name, first, second, firstTime, secondTime, secondProfile string
		fault                                                     bool
	}{
		{"same", `{"currency":"EUR","total":70}`, `{"currency":"EUR","total":70.0000}`, "2026-01-16", "2026-01-16", "1", false},
		{"different known amounts", `{"currency":"EUR","total":70}`, `{"currency":"EUR","total":60}`, "2026-01-16", "2026-01-16", "1", true},
		{"known and unknown", `{"currency":"EUR","total":70}`, `{"currency":"EUR"}`, "2026-01-16", "2026-01-16", "1", true},
		{"same unreadable", `{"currency":"EUR","total":null}`, `{"currency":"EUR","total":"70"}`, "2026-01-16", "2026-01-16", "1", false},
		{"qualification", `{"currency":"EUR","total":70}`, `{"total":70}`, "2026-01-16", "2026-01-16", "1", true},
		{"timestamp", `{"currency":"EUR","total":70}`, `{"currency":"EUR","total":70}`, "2026-01-16", "2026-01-16 00:00:00.000001", "1", true},
		{"possible timestamp", `{"total":70}`, `{"total":70}`, "2026-01-16", "2026-01-17", "1", true},
		{"period", `{"currency":"EUR","total":70}`, `{"currency":"EUR","total":70}`, "2026-01-16", "2026-02-01", "1", true},
		{"profile", `{"currency":"EUR","total":70}`, `{"currency":"EUR","total":70}`, "2026-01-16", "2026-01-16", "2", true},
		{"null and nonnull profile", `{"currency":"EUR","total":70}`, `{"currency":"EUR","total":70}`, "2026-01-16", "2026-01-16", "NULL", true},
		{"excluded", `{"currency":"USD","total":70}`, `{"currency":"GBP","total":60}`, "2026-01-16", "2026-01-17", "1", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			execBehavioralFixture(t, dw, "TRUNCATE krenalis_events")
			profile := "NULL"
			if test.secondProfile != "NULL" {
				profile = "'" + behavioralTestID(1) + "'"
				if test.secondProfile == "2" {
					profile = "'" + behavioralTestID(2) + "'"
				}
			}
			execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','old','track','Order Completed','2026-01-15','{"currency":"EUR","total":100}'),
('%s','copy','track','Order Completed','%s','%s'),
(%s,'copy','track','Order Completed','%s','%s')`,
				behavioralTestID(1), behavioralTestID(1), test.firstTime, test.first, profile, test.secondTime,
				test.second))
			result, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
			if err != nil {
				if !test.fault || result != nil || !strings.Contains(err.Error(), "occurrence contract") {
					t.Fatalf("expected source fault=%v without partial result, got %+v, %v", test.fault, result, err)
				}
			}
			if err == nil && test.fault {
				t.Fatal("expected source fault, got success")
			}
			_, err = store.ExplainProfileFilter(t.Context(), root, behavioral.Bindings{}, at, behavioralTestID(1))
			if (err != nil) != test.fault {
				t.Fatalf("expected target profile source fault=%v, got %v", test.fault, err)
			}
			if test.secondProfile != "2" {
				explained, err := store.ExplainProfileFilter(t.Context(), root, behavioral.Bindings{}, at,
					behavioralTestID(2))
				if err != nil {
					t.Fatalf("expected unrelated profile isolation, got %v", err)
				}
				if explained.Nodes[0].Truth != behavioral.NoMatch {
					t.Fatalf("expected unrelated profile NO_MATCH, got %+v", explained.Nodes)
				}
			}
		})
	}

	// Raw qualifications close to the same ternary state for all copies.
	// They remain independent of the monetary reading, never inferred from it.
	t.Run("independent where", func(t *testing.T) {
		execBehavioralFixture(t, dw, `TRUNCATE krenalis_events;
INSERT INTO krenalis_events VALUES
('00000000-0000-0000-0000-000000000001','a','track','Order Completed','2026-01-15',
'{"currency":"EUR","total":100,"q":"yes"}'),
('00000000-0000-0000-0000-000000000001','b','track','Order Completed','2026-01-16','{"currency":"EUR","total":50}'),
('00000000-0000-0000-0000-000000000002','c','track','Order Completed','2026-01-15',
'{"currency":"EUR","total":100,"q":"yes"}'),
('00000000-0000-0000-0000-000000000002','d','track','Order Completed','2026-01-16',
'{"currency":"EUR","total":50,"q":"no"}')`)
		pair.Where = &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
			&state.WhereCondition{Property: []string{"properties", "q"}, Operator: state.OpIs,
				Values: []any{json.Value(`"yes"`)}},
		}}
		result, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
		if err != nil {
			t.Fatalf("expected independent where, got %v", err)
		}
		assertBehavioralTruths(t, result, []behavioral.Truth{behavioral.Unknown, behavioral.NoMatch})
	})

}

// TestBehavioralLastPreviousOraclePostgreSQL compares SQL with actual
// compatible pairs from subsets and tie orders, and exact rational comparisons.
func TestBehavioralLastPreviousOraclePostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	execBehavioralFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,timestamp timestamp,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	field, err := behavioral.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected field binding, got %v", err)
	}
	currency, err := behavioral.ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR",
		"Order Completed")
	if err != nil {
		t.Fatalf("expected currency binding, got %v", err)
	}
	pair := &behavioral.LastVersusPrevious{Scope: behavioral.Scope{Event: "Order Completed", Grain: "registered-message",
		Start: at.AddDate(0, -1, 0), End: at}, Currency: currency, Field: field}
	root := &behavioral.Rule{LastVersusPrevious: pair}
	amounts := [][5]string{
		{"100", "80", "60", "40", "20"},
		{"null", "100", `"80"`, "-10", "100.00001"},
		{"99999999999999.9999", "-99999999999999.9999", "-0", "1E2", "1e20"},
	}
	var profiles, events []string
	var histories [][]pairOracleOccurrence
	for ti, times := range [][5]int{{1, 2, 3, 4, 5}, {3, 1, 2, 2, 3}, {1, 1, 1, 1, 1}} {
		for code := 0; code < 243; code++ {
			id, remaining := len(histories)+1, code
			profiles = append(profiles, "('"+behavioralTestID(id)+"')")
			var history []pairOracleOccurrence
			for i, timestamp := range times {
				membership := remaining % 3
				remaining /= 3
				amount := amounts[(code+ti)%len(amounts)][i]
				history = append(history, pairOracleOccurrence{timestamp, membership, amount})
				payload := `"currency":"EUR",`
				if membership == 0 {
					payload = `"currency":"USD",`
				} else if membership == 2 {
					payload = []string{"", `"currency":null,`, `"currency":42,`}[i%3]
				}
				if amount != "null" || i%2 == 0 {
					payload += `"total":` + amount + `,`
				}
				payload += `"fixture":true`
				event := fmt.Sprintf("('%s','oracle/%d/%d','track','Order Completed','2026-01-0%d','{%s}')",
					behavioralTestID(id), id, i, timestamp, payload)
				// Deliberate physical duplication must preserve message grain.
				events = append(events, event, event)
			}
			histories = append(histories, history)
		}
	}
	// Full readable-domain extrema, shortage, optional earlier events and an
	// unreadable event hidden by two certain later messages need explicit cases.
	for _, history := range [][]pairOracleOccurrence{
		{{1, 1, "-99999999999999.9999"}, {2, 1, "99999999999999.9999"}},
		{{1, 1, "99999999999999.9999"}, {2, 1, "-99999999999999.9999"}},
		nil, {{1, 2, "100"}},
		{{1, 1, "null"}, {2, 1, "100"}, {3, 1, "50"}},
		{{1, 2, "1e20"}, {2, 1, "100"}, {3, 1, "50"}},
	} {
		id := len(histories) + 1
		profiles = append(profiles, "('"+behavioralTestID(id)+"')")
		for i, event := range history {
			currency := `"EUR"`
			if event.membership == 2 {
				currency = "null"
			}
			events = append(events, fmt.Sprintf("('%s','edge/%d/%d','track','Order Completed','2026-01-0%d',"+
				"'{\"currency\":%s,\"total\":%s}')",
				behavioralTestID(id), id, i, event.timestamp, currency, event.amount))
		}
		histories = append(histories, history)
	}
	execBehavioralFixture(t, dw, "INSERT INTO profiles VALUES "+strings.Join(profiles, ","))
	execBehavioralFixture(t, dw, "INSERT INTO krenalis_events VALUES "+strings.Join(events, ","))

	operators := []behavioral.Comparison{behavioral.Less, behavioral.LessOrEqual, behavioral.Greater, behavioral.GreaterOrEqual}
	for _, operator := range operators {
		for _, threshold := range []string{"-30", "0", "0.00001", "199999999999999.9998", "1e20", "-1e20"} {
			t.Run(fmt.Sprintf("operator=%d/threshold=%s", operator, threshold), func(t *testing.T) {
				pair.Check = behavioral.LastPreviousCheck{Kind: behavioral.LastPreviousDifference,
					Test: behavioral.SumTest{Operator: operator, Value: decimal.MustParse(threshold)}}
				want := make([]behavioral.Truth, len(histories))
				for i, history := range histories {
					want[i], _ = enumeratePairTest(history, operator, threshold)
				}
				result, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{
					Limit: 1000})
				if err != nil {
					t.Fatalf("expected exact pair comparison, got %v", err)
				}
				assertBehavioralTruths(t, result, want)
				if operator == behavioral.LessOrEqual && threshold == "-30" {
					for i, history := range histories {
						_, selection := enumeratePairTest(history, operator, threshold)
						explanation, err := store.ExplainProfileFilter(t.Context(), root, behavioral.Bindings{},
							at, behavioralTestID(i+1))
						if err != nil {
							t.Fatalf("expected selection for every profile, got %v", err)
						}
						if explanation.Nodes[0].Pair == nil || *explanation.Nodes[0].Pair != selection ||
							explanation.Nodes[0].Truth != want[i] {
							t.Fatalf("expected profile %d truth %v pair %+v, got %+v", i+1, want[i], selection, explanation.Nodes)
						}
					}
				}
			})
		}
	}

}

// TestBehavioralLastPreviousPostgreSQL exercises the defining counterexamples.
func TestBehavioralLastPreviousPostgreSQL(t *testing.T) {
	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	execBehavioralFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,timestamp timestamp,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	field, err := behavioral.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected field binding, got %v", err)
	}
	currency, err := behavioral.ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR",
		"Order Completed")
	if err != nil {
		t.Fatalf("expected currency binding, got %v", err)
	}
	pair := &behavioral.LastVersusPrevious{Scope: behavioral.Scope{Event: "Order Completed", Grain: "registered-message",
		Start: at.AddDate(0, -1, 0), End: at}, Currency: currency, Field: field,
		Check: behavioral.LastPreviousCheck{Kind: behavioral.LastPreviousDifference,
			Test: behavioral.SumTest{Operator: behavioral.LessOrEqual, Value: decimal.MustParse("-30")}}}
	root := &behavioral.Rule{LastVersusPrevious: pair}
	for _, test := range []struct {
		name    string
		history []pairOracleOccurrence
		want    behavioral.Truth
	}{
		{"decrease", []pairOracleOccurrence{{1, 1, "100"}, {2, 1, "70"}}, behavioral.Match},
		{"incompatible marginal pair", []pairOracleOccurrence{{3, 2, "60"}, {2, 1, "80"}, {1, 1, "100"}}, behavioral.NoMatch},
		{"tie", []pairOracleOccurrence{{1, 1, "100"}, {1, 1, "50"}}, behavioral.Unknown},
		{"possible absence", []pairOracleOccurrence{{1, 1, "100"}, {2, 2, "50"}}, behavioral.Unknown},
		{"certain shortage", []pairOracleOccurrence{{1, 1, "100"}}, behavioral.NoMatch},
		{"unknown", []pairOracleOccurrence{{1, 1, "100"}, {2, 1, "null"}}, behavioral.Unknown},
		{"old irrelevant unknown", []pairOracleOccurrence{{1, 1, "null"}, {2, 1, "100"}, {3, 1, "50"}}, behavioral.Match},
		{"optional possible pair cannot pass", []pairOracleOccurrence{{1, 2, "100"}, {2, 2, "100"}}, behavioral.NoMatch},
		{"equal amounts tied", []pairOracleOccurrence{{1, 1, "100"}, {1, 1, "100.0"}}, behavioral.NoMatch},
		{"empty", nil, behavioral.NoMatch},
	} {
		t.Run(test.name, func(t *testing.T) {
			execBehavioralFixture(t, dw, "TRUNCATE profiles,krenalis_events")
			execBehavioralFixture(t, dw, "INSERT INTO profiles VALUES ('"+behavioralTestID(1)+"')")
			var events []string
			for i, event := range test.history {
				currency := `"EUR"`
				if event.membership == 2 {
					currency = "null"
				}
				events = append(events, fmt.Sprintf("('%s','%d','track','Order Completed','2026-01-0%d',"+
					"'{\"currency\":%s,\"total\":%s}')",
					behavioralTestID(1), i, event.timestamp, currency, event.amount))
			}
			if len(events) > 0 {
				execBehavioralFixture(t, dw, "INSERT INTO krenalis_events VALUES "+strings.Join(events, ","))
			}
			result, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{Limit: 1000})
			if err != nil {
				t.Fatalf("expected exact pair query, got %v", err)
			}
			assertBehavioralTruths(t, result, []behavioral.Truth{test.want})
			truth, selection := enumeratePairTest(test.history, behavioral.LessOrEqual, "-30")
			if truth != test.want {
				t.Fatalf("expected independent oracle %v, got %v", test.want, truth)
			}
			explanation, err := store.ExplainProfileFilter(t.Context(), root, behavioral.Bindings{}, at,
				behavioralTestID(1))
			if err != nil {
				t.Fatalf("expected exact pair explanation, got %v", err)
			}
			if explanation.Nodes[0].Truth != truth || explanation.Nodes[0].Pair == nil ||
				*explanation.Nodes[0].Pair != selection {
				t.Fatalf("expected truth %v and pair %+v, got %+v", truth, selection, explanation.Nodes)
			}
		})
	}
}

// enumeratePairTest enumerates subsets and every chronological permutation.
// Unknown amounts admit witnesses on both sides of a finite ordered threshold.
// No cumulative-count formula or production normalization is used.
func enumeratePairTest(history []pairOracleOccurrence, operator behavioral.Comparison, threshold string) (behavioral.Truth, warehouses.OccurrencePairSelection) {
	pairs := map[[2]int]bool{}
	absent := false
	for mask := 0; mask < 1<<len(history); mask++ {
		valid := true
		var included []int
		for i, event := range history {
			present := mask&(1<<i) != 0
			if present && event.membership == 0 || !present && event.membership == 1 {
				valid = false
			}
			if present {
				included = append(included, i)
			}
		}
		if !valid {
			continue
		}
		if len(included) < 2 {
			absent = true
			continue
		}
		var permute func(int)
		permute = func(position int) {
			if position == len(included) {
				pairs[[2]int{included[len(included)-1], included[len(included)-2]}] = true
				return
			}
			for i := position; i < len(included); i++ {
				included[i], included[position] = included[position], included[i]
				if position == 0 || history[included[position-1]].timestamp <= history[included[position]].timestamp {
					permute(position + 1)
				}
				included[i], included[position] = included[position], included[i]
			}
		}
		permute(0)
	}
	identities := map[int]bool{}
	success, failure := false, absent
	t, ok := new(big.Rat).SetString(threshold)
	if !ok {
		panic("core/internal/datastore: invalid oracle threshold")
	}
	for pair := range pairs {
		identities[pair[0]], identities[pair[1]] = true, true
		x, xok := new(big.Rat).SetString(history[pair[0]].amount)
		y, yok := new(big.Rat).SetString(history[pair[1]].amount)
		limit := big.NewRat(999999999999999999, 10000)
		readable := func(value *big.Rat, ok bool) bool {
			return ok && new(big.Rat).Mul(value, big.NewRat(10000, 1)).IsInt() &&
				new(big.Rat).Abs(value).Cmp(limit) <= 0
		}
		deltas := []*big.Rat{}
		if readable(x, xok) && readable(y, yok) {
			deltas = append(deltas, new(big.Rat).Sub(x, y))
		} else {
			// One free amount can realize t-1 and t+1 regardless of the
			// other amount. For x unknown, x=y+t±1; for y, y=x-t±1.
			// These witnesses suffice for all four finite ordered tests.
			deltas = append(deltas, new(big.Rat).Sub(t, big.NewRat(1, 1)), new(big.Rat).Add(t, big.NewRat(1, 1)))
		}
		for _, delta := range deltas {
			comparison, passes := delta.Cmp(t), false
			switch operator {
			case behavioral.GreaterOrEqual:
				passes = comparison >= 0
			case behavioral.Greater:
				passes = comparison > 0
			case behavioral.LessOrEqual:
				passes = comparison <= 0
			case behavioral.Less:
				passes = comparison < 0
			default:
				panic("core/internal/datastore: invalid oracle operator")
			}
			success, failure = success || passes, failure || !passes
		}
	}
	truth := behavioral.Unknown
	if !success {
		truth = behavioral.NoMatch
	} else if !failure {
		truth = behavioral.Match
	}
	return truth, warehouses.OccurrencePairSelection{Candidates: int64(len(identities)),
		AbsencePossible: absent, Unique: !absent && len(pairs) == 1}
}
