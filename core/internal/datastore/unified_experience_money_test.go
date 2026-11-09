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
)

// TestUnifiedExperienceMoneyOraclePostgreSQL enumerates possible selections.
// Its three body cases include 80/unknown currency (false), 120/EUR (true) and
// 120/unknown currency (unknown), without consulting normalization or SQL.
func TestUnifiedExperienceMoneyOraclePostgreSQL(t *testing.T) {
	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	execUnifiedFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,timestamp timestamp,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	field, err := unified.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected monetary field resolved, got %v", err)
	}
	for _, test := range []struct {
		first bool
		times [3]int
	}{
		{false, [3]int{1, 2, 3}}, {false, [3]int{1, 2, 2}},
		{false, [3]int{1, 1, 1}}, {false, [3]int{3, 2, 1}},
		{true, [3]int{1, 2, 3}}, {true, [3]int{1, 2, 2}},
		{true, [3]int{1, 1, 1}}, {true, [3]int{3, 2, 1}},
	} {
		t.Run(fmt.Sprintf("first=%v/times=%v", test.first, test.times), func(t *testing.T) {
			execUnifiedFixture(t, dw, "TRUNCATE profiles,krenalis_events")
			var profiles, events []string
			want := make([]unified.Truth, 729)
			for code := range want {
				id := unifiedTestID(code + 1)
				profiles = append(profiles, "('"+id+"')")
				n := code
				var qualifications, bodies [3]int
				for i := range 3 {
					qualifications[i], bodies[i] = n%3, n/3%3
					n /= 9
					payload := []string{`"total":120`}
					if bodies[i] == 0 {
						payload[0] = `"total":80`
					} else if bodies[i] == 1 {
						payload = append(payload, `"currency":"EUR"`)
					}
					if qualifications[i] < 2 {
						payload = append(payload, `"q":"`+[]string{"no", "yes"}[qualifications[i]]+`"`)
					}
					events = append(events, fmt.Sprintf("('%s','%d/%d','track','Order Completed','2026-01-0%d','{%s}')",
						id, code, i, test.times[i], strings.Join(payload, ",")))
				}
				seen := [2]bool{}
				for mask := range 8 {
					valid, selected := true, 0
					for i, q := range qualifications {
						present := mask&(1<<i) != 0
						if q < 2 && present != (q == 1) {
							valid = false
						}
						if present && (selected == 0 || test.first && test.times[i] < selected ||
							!test.first && test.times[i] > selected) {
							selected = test.times[i]
						}
					}
					if !valid {
						continue
					}
					if selected == 0 {
						seen[0] = true
					}
					for i, body := range bodies {
						if mask&(1<<i) == 0 || test.times[i] != selected {
							continue
						}
						seen[0] = seen[0] || body != 1
						seen[1] = seen[1] || body != 0
					}
				}
				want[code] = unified.Unknown
				if !seen[0] {
					want[code] = unified.Match
				} else if !seen[1] {
					want[code] = unified.NoMatch
				}
			}
			execUnifiedFixture(t, dw, "INSERT INTO profiles VALUES "+strings.Join(profiles, ","))
			execUnifiedFixture(t, dw, "INSERT INTO krenalis_events VALUES "+strings.Join(events, ","))
			experience := &unified.Experience{First: test.first,
				Scope: unified.Scope{Event: "Order Completed", Grain: "registered-message",
					Start: at.AddDate(0, -1, 0), End: at},
				Where: &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"properties", "q"}, Operator: state.OpIs,
						Values: []any{json.Value(`"yes"`)}},
				}}, Verify: &unified.OccurrencePredicate{Money: &unified.Money{
					Field: field, NativeCurrency: "EUR", Test: unified.SumTest{Operator: unified.GreaterOrEqual,
						Value: decimal.MustParse("100")}}}}
			result, err := store.UnifiedProfiles(t.Context(), &unified.Rule{Experience: experience},
				unified.Bindings{}, at, unified.Page{Limit: 1000})
			if err != nil {
				t.Fatalf("expected independent completion oracle result, got %v", err)
			}
			assertUnifiedTruths(t, result, want)
		})
	}
}

// TestUnifiedExperienceMoneyPostgreSQL evaluates the governed matrix in Go.
func TestUnifiedExperienceMoneyPostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	execUnifiedFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,timestamp timestamp,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	field, err := unified.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected monetary field resolved, got %v", err)
	}
	money := &unified.Money{Field: field, NativeCurrency: "EUR",
		Test: unified.SumTest{Operator: unified.GreaterOrEqual, Value: decimal.MustParse("100")}}
	experience := &unified.Experience{Scope: unified.Scope{Event: "Order Completed", Grain: "registered-message",
		Start: at.AddDate(0, -1, 0), End: at}, Verify: &unified.OccurrencePredicate{Money: money}}
	root := &unified.Rule{Experience: experience}
	var profiles, events []string
	var want []unified.Truth
	for _, currency := range []struct {
		value string
		state int
	}{
		{`"EUR"`, 1}, {`"USD"`, 0}, {`"eur"`, 0}, {`""`, 0},
		{``, 2}, {`null`, 2}, {`100`, 2}, {`true`, 2}, {`{}`, 2}, {`[]`, 2},
	} {
		for _, amount := range []struct {
			value string
			state int
		}{
			{`80`, 0}, {`120`, 1}, {`100`, 1}, {`120.00000`, 1}, {`-10`, 0},
			{`99999999999999.9999`, 1}, {`-99999999999999.9999`, 0},
			{``, 2}, {`null`, 2}, {`"120"`, 2}, {`100.00001`, 2}, {`1e20`, 2}, {`{}`, 2}, {`true`, 2},
		} {
			n := len(profiles) + 1
			id := unifiedTestID(n)
			payload := []string{}
			if currency.value != "" {
				payload = append(payload, `"currency":`+currency.value)
			}
			if amount.value != "" {
				payload = append(payload, `"total":`+amount.value)
			}
			profiles = append(profiles, "('"+id+"')")
			events = append(events, fmt.Sprintf("('%s','matrix-%d','track','Order Completed','2026-01-15','{%s}')",
				id, n, strings.Join(payload, ",")))
			truth := unified.Unknown
			if currency.state == 0 || amount.state == 0 {
				truth = unified.NoMatch
			} else if currency.state == 1 && amount.state == 1 {
				truth = unified.Match
			}
			want = append(want, truth)
		}
	}
	profiles = append(profiles, "('"+unifiedTestID(len(profiles)+1)+"')")
	want = append(want, unified.NoMatch)
	execUnifiedFixture(t, dw, "INSERT INTO profiles VALUES "+strings.Join(profiles, ","))
	execUnifiedFixture(t, dw, "INSERT INTO krenalis_events VALUES "+strings.Join(events, ","))
	for _, first := range []bool{false, true} {
		experience.First = first
		result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 1000})
		if err != nil {
			t.Fatalf("expected complete monetary matrix, got %v", err)
		}
		assertUnifiedTruths(t, result, want)
	}

	t.Run("Kleene composition", func(t *testing.T) {
		for _, test := range []struct {
			operator state.WhereLogical
			event    string
			truth    unified.Truth
		}{
			{state.OpOr, "Order Completed", unified.Match},
			{state.OpAnd, "Other", unified.NoMatch},
		} {
			experience.Verify = &unified.OccurrencePredicate{Operator: test.operator,
				Children: []unified.OccurrencePredicate{
					{Money: money},
					{Record: &state.WhereCondition{Property: []string{"event"}, Operator: state.OpIs,
						Values: []any{test.event}}},
				}}
			result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 1000})
			if err != nil {
				t.Fatalf("expected composed monetary verification, got %v", err)
			}
			truths := make([]unified.Truth, len(want))
			for i := range len(truths) - 1 {
				truths[i] = test.truth
			}
			truths[len(truths)-1] = unified.NoMatch
			assertUnifiedTruths(t, result, truths)
		}
		experience.Verify = &unified.OccurrencePredicate{Money: money}
	})

	t.Run("selection before currency", func(t *testing.T) {
		execUnifiedFixture(t, dw, `TRUNCATE profiles,krenalis_events;
INSERT INTO profiles SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid FROM generate_series(1,4) n;
INSERT INTO krenalis_events VALUES
('`+unifiedTestID(1)+`','old','track','Order Completed','2026-01-01','{"total":150,"currency":"EUR"}'),
('`+unifiedTestID(1)+`','new','track','Order Completed','2026-01-20','{"total":120,"currency":"USD"}'),
('`+unifiedTestID(2)+`','certain','track','Order Completed','2026-01-01','{"total":120,"currency":"EUR"}'),
('`+unifiedTestID(2)+`','possible','track','Order Completed','2026-01-20','{"total":120}'),
('`+unifiedTestID(3)+`','tie-eur','track','Order Completed','2026-01-10','{"total":120,"currency":"EUR"}'),
('`+unifiedTestID(3)+`','tie-usd','track','Order Completed','2026-01-10','{"total":120,"currency":"USD"}')`)
		for _, first := range []bool{false, true} {
			experience.First = first
			result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
			if err != nil {
				t.Fatalf("expected selection before monetary verification, got %v", err)
			}
			truths := []unified.Truth{unified.NoMatch, unified.Unknown, unified.Unknown, unified.NoMatch}
			if first {
				truths[0], truths[1] = unified.Match, unified.Match
			}
			assertUnifiedTruths(t, result, truths)
		}
	})

	t.Run("thresholds and unknown amounts", func(t *testing.T) {
		execUnifiedFixture(t, dw, `TRUNCATE profiles,krenalis_events;
INSERT INTO profiles SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid FROM generate_series(1,5) n;
INSERT INTO krenalis_events VALUES
('`+unifiedTestID(1)+`','a','track','Order Completed','2026-01-10','{"currency":"EUR","total":100}'),
('`+unifiedTestID(2)+`','b','track','Order Completed','2026-01-10','{"currency":"EUR","total":100.0001}'),
('`+unifiedTestID(3)+`','c','track','Order Completed','2026-01-10','{"currency":"EUR","total":1e20}'),
('`+unifiedTestID(4)+`','d','track','Order Completed','2026-01-10','{"total":80}'),
('`+unifiedTestID(5)+`','e','track','Order Completed','2026-01-10','{"currency":"USD"}')`)
		for _, test := range []struct {
			operator unified.Comparison
			value    string
			truths   []unified.Truth
		}{
			{unified.Greater, "100", []unified.Truth{unified.NoMatch, unified.Match, unified.Unknown, unified.NoMatch, unified.NoMatch}},
			{unified.GreaterOrEqual, "100.00001", []unified.Truth{unified.NoMatch, unified.Match, unified.Unknown, unified.NoMatch, unified.NoMatch}},
			{unified.Less, "100.00001", []unified.Truth{unified.Match, unified.NoMatch, unified.Unknown, unified.Unknown, unified.NoMatch}},
			{unified.GreaterOrEqual, "1e75", []unified.Truth{unified.NoMatch, unified.NoMatch, unified.Unknown, unified.NoMatch, unified.NoMatch}},
			{unified.LessOrEqual, "1e75", []unified.Truth{unified.Match, unified.Match, unified.Unknown, unified.Unknown, unified.NoMatch}},
			{unified.LessOrEqual, "-1e75", []unified.Truth{unified.NoMatch, unified.NoMatch, unified.Unknown, unified.NoMatch, unified.NoMatch}},
			{unified.GreaterOrEqual, "-1e75", []unified.Truth{unified.Match, unified.Match, unified.Unknown, unified.Unknown, unified.NoMatch}},
		} {
			money.Test = unified.SumTest{Operator: test.operator, Value: decimal.MustParse(test.value)}
			result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
			if err != nil {
				t.Fatalf("expected exact threshold %s, got %v", test.value, err)
			}
			assertUnifiedTruths(t, result, test.truths)
		}
	})

	t.Run("copy faults and scoped explanation", func(t *testing.T) {
		execUnifiedFixture(t, dw, `TRUNCATE profiles,krenalis_events;
INSERT INTO profiles VALUES ('`+unifiedTestID(1)+`'),('`+unifiedTestID(2)+`')`)
		money.Test = unified.SumTest{Operator: unified.GreaterOrEqual, Value: decimal.MustParse("100")}
		for _, test := range []struct {
			other string
			fault bool
		}{
			{`{"total":120,"currency":"EUR"}`, false},
			{`{"total":80,"currency":"EUR"}`, true},
			{`{"total":null,"currency":"EUR"}`, true},
		} {
			execUnifiedFixture(t, dw, `TRUNCATE krenalis_events;
INSERT INTO krenalis_events VALUES
('`+unifiedTestID(1)+`','same','track','Order Completed','2026-01-15','{"total":120,"currency":"EUR"}'),
('`+unifiedTestID(1)+`','same','track','Order Completed','2026-01-15','`+test.other+`')`)
			result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
			if err != nil {
				if !test.fault || result != nil || !strings.Contains(err.Error(), "occurrence source violates") {
					t.Fatalf("expected fault=%v without partial results, got %+v, %v", test.fault, result, err)
				}
			}
			if err == nil {
				if test.fault {
					t.Fatal("expected copy fault, got success")
				}
				assertUnifiedTruths(t, result, []unified.Truth{unified.Match, unified.NoMatch})
				explained, err := store.ExplainUnifiedProfile(t.Context(), root, unified.Bindings{}, at, unifiedTestID(1))
				if err != nil {
					t.Fatalf("expected deduplicated explanation, got %v", err)
				}
				if explained.Nodes[0].Experience.Candidates != 1 {
					t.Fatalf("expected one logical candidate, got %+v", explained)
				}
			}
			explained, err := store.ExplainUnifiedProfile(t.Context(), root, unified.Bindings{}, at, unifiedTestID(2))
			if err != nil {
				t.Fatalf("expected unrelated-profile explanation, got %v", err)
			}
			if explained.Nodes[0].Truth != unified.NoMatch {
				t.Fatalf("expected absent selection for unrelated profile, got %+v", explained)
			}
		}
	})

}
