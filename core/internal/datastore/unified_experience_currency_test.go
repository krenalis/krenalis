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

// TestUnifiedExperienceCurrencyOraclePostgreSQL completes each shared reading.
func TestUnifiedExperienceCurrencyOraclePostgreSQL(t *testing.T) {
	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	execUnifiedFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,timestamp timestamp,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	currency, err := unified.ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR", "Order Completed")
	if err != nil {
		t.Fatalf("expected resolved currency, got %v", err)
	}
	field, err := unified.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected resolved amount, got %v", err)
	}
	for _, test := range []struct{ first, usd, orTime, tie bool }{
		{false, false, false, false}, {true, false, false, false},
		{false, true, false, false}, {true, true, false, false},
		{false, false, false, true}, {true, false, false, true},
		{false, false, true, false}, {true, false, true, false},
		{false, true, true, false}, {true, true, true, false},
	} {
		t.Run(fmt.Sprintf("first=%v/usd=%v/or=%v/tie=%v", test.first, test.usd, test.orTime, test.tie), func(t *testing.T) {
			execUnifiedFixture(t, dw, "TRUNCATE profiles,krenalis_events")
			var profiles, events []string
			want := make([]unified.Truth, 729)
			times := [2]int{10, 20}
			if test.tie {
				times[0] = 20
			}
			for code := range want {
				id := unifiedTestID(code + 1)
				profiles = append(profiles, "('"+id+"')")
				n := code
				var states [2][3]int
				for i := range 2 {
					states[i] = [3]int{n % 3, n / 3 % 3, n / 9 % 3}
					n /= 27
					payload := []string{}
					if states[i][0] < 2 {
						payload = append(payload, `"currency":"`+[]string{"EUR", "USD"}[states[i][0]]+`"`)
					}
					if states[i][1] < 2 {
						payload = append(payload, `"q":"`+[]string{"no", "yes"}[states[i][1]]+`"`)
					}
					payload = append(payload, `"total":`+[]string{"80", "150", "null"}[states[i][2]])
					events = append(events, fmt.Sprintf("('%s','%d/%d','track','Order Completed','2026-01-%d','{%s}')",
						id, code, i, times[i], strings.Join(payload, ",")))
				}
				seen := [2]bool{}
				// Currency has three observational classes: EUR, USD, and another String.
				// Qualification and amount completions are independent of that one value.
				for currencies := range 9 {
					actual := [2]int{currencies % 3, currencies / 3}
					for qmask := range 4 {
						for amask := range 4 {
							valid := true
							selected := -1
							var present [2]bool
							for i, s := range states {
								q, a := qmask&(1<<i) != 0, amask&(1<<i) != 0
								if s[0] < 2 && actual[i] != s[0] || s[1] < 2 && q != (s[1] == 1) || s[2] < 2 && a != (s[2] == 1) {
									valid = false
								}
								present[i] = actual[i] == 0 && q
								if present[i] && (selected < 0 || test.first && times[i] < selected || !test.first && times[i] > selected) {
									selected = times[i]
								}
							}
							if !valid {
								continue
							}
							if selected < 0 {
								seen[0] = true
							}
							target := 0
							if test.usd {
								target = 1
							}
							for i := range 2 {
								if !present[i] || times[i] != selected {
									continue
								}
								truth := actual[i] == target && amask&(1<<i) != 0
								if test.orTime {
									truth = truth || times[i] >= 15
								}
								if truth {
									seen[1] = true
								} else {
									seen[0] = true
								}
							}
						}
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
			native := "EUR"
			if test.usd {
				native = "USD"
			}
			verify := &unified.ExperienceVerification{Money: &unified.Money{Field: field, NativeCurrency: native,
				Test: unified.SumTest{Operator: unified.Greater, Value: decimal.MustParse("100")}}}
			if test.orTime {
				period := unified.Scope{Event: "Order Completed", Grain: "registered-message",
					Start: at.AddDate(0, 0, -17), End: at}
				verify = &unified.ExperienceVerification{Operator: state.OpOr, Children: []unified.ExperienceVerification{
					*verify, {OccurredIn: &period}}}
			}
			experience := &unified.Experience{First: test.first, Currency: currency,
				Scope: unified.Scope{Event: "Order Completed", Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at},
				Where: &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"properties", "q"}, Operator: state.OpIs,
						Values: []any{json.Value(`"yes"`)}}}},
				Verify: verify}
			result, err := store.UnifiedProfiles(t.Context(), &unified.Rule{Experience: experience},
				unified.Bindings{}, at, unified.Page{Limit: 1000})
			if err != nil {
				t.Fatalf("expected exact conditional selection, got %v", err)
			}
			assertUnifiedTruths(t, result, want)
		})
	}
}

// TestUnifiedExperienceCurrencyPostgreSQL checks readable business examples.
func TestUnifiedExperienceCurrencyPostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	execUnifiedFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,timestamp timestamp,properties jsonb);
INSERT INTO profiles VALUES ('`+unifiedTestID(1)+`'),('`+unifiedTestID(2)+`')`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	currency, err := unified.ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR", "Order Completed")
	if err != nil {
		t.Fatalf("expected resolved currency, got %v", err)
	}
	field, err := unified.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected resolved amount, got %v", err)
	}
	money := &unified.Money{Field: field, NativeCurrency: "EUR",
		Test: unified.SumTest{Operator: unified.Greater, Value: decimal.MustParse("100")}}
	experience := &unified.Experience{Currency: currency,
		Scope:  unified.Scope{Event: "Order Completed", Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at},
		Verify: &unified.ExperienceVerification{Money: money}}
	root := &unified.Rule{Experience: experience}
	for _, first := range []bool{false, true} {
		experience.First = first
		for _, test := range []struct {
			name, older, next string
			want              unified.Truth
		}{
			{"excluded USD", `{"total":150,"currency":"EUR"}`, `{"currency":"USD"}`, unified.Match},
			{"shared currency", `{"total":150,"currency":"EUR"}`, `{"total":150}`, unified.Match},
			{"possible insufficient", `{"total":150,"currency":"EUR"}`, `{"total":80}`, unified.Unknown},
			{"possible absence", ``, `{"total":150}`, unified.Unknown},
			{"unreadable amount", `{"total":150,"currency":"EUR"}`, `{"total":"150"}`, unified.Unknown},
			{"no events", ``, ``, unified.NoMatch},
		} {
			t.Run(fmt.Sprintf("first=%v/%s", first, test.name), func(t *testing.T) {
				execUnifiedFixture(t, dw, "TRUNCATE krenalis_events")
				for i, payload := range []string{test.older, test.next} {
					if payload == "" {
						continue
					}
					day := 10 + i*10
					if first {
						day = 30 - day
					}
					execUnifiedFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','event-%d','track','Order Completed','2026-01-%d','%s')`,
						unifiedTestID(1), i, day, payload))
				}
				result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
				if err != nil {
					t.Fatalf("expected governed selection, got %v", err)
				}
				assertUnifiedTruths(t, result, []unified.Truth{test.want, unified.NoMatch})
			})
		}
	}

	t.Run("different currency", func(t *testing.T) {
		money.NativeCurrency = "USD"
		defer func() {
			money.NativeCurrency = "EUR"
			experience.Verify = &unified.ExperienceVerification{Money: money}
		}()
		for _, test := range []struct {
			payload string
			day     int
			orTime  bool
			want    unified.Truth
		}{
			{`{"total":150}`, 10, false, unified.NoMatch},
			{`{"total":"broken"}`, 10, false, unified.NoMatch},
			{`{"total":"broken"}`, 10, true, unified.NoMatch},
			{`{"total":"broken"}`, 20, true, unified.Unknown},
			{`{"total":"broken","currency":"EUR"}`, 20, true, unified.Match},
		} {
			execUnifiedFixture(t, dw, "TRUNCATE krenalis_events")
			execUnifiedFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','different','track','Order Completed','2026-01-%d','%s')`, unifiedTestID(1), test.day, test.payload))
			experience.Verify = &unified.ExperienceVerification{Money: money}
			if test.orTime {
				period := unified.Scope{Event: "Order Completed", Grain: "registered-message",
					Start: at.AddDate(0, 0, -17), End: at}
				experience.Verify = &unified.ExperienceVerification{Operator: state.OpOr,
					Children: []unified.ExperienceVerification{*experience.Verify, {OccurredIn: &period}}}
			}
			result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
			if err != nil {
				t.Fatalf("expected conditional FALSE composed with time, got %v", err)
			}
			assertUnifiedTruths(t, result, []unified.Truth{test.want, unified.NoMatch})
		}
	})

	t.Run("copies", func(t *testing.T) {
		for _, test := range []struct {
			left, right string
			fault       bool
			want        unified.Truth
		}{
			{`"EUR"`, `"EUR"`, false, unified.Match},
			{`"USD"`, `"GBP"`, false, unified.NoMatch},
			{`null`, `null`, false, unified.Unknown},
			{`"EUR"`, `null`, true, 0},
			{`"EUR"`, `"USD"`, true, 0},
		} {
			execUnifiedFixture(t, dw, `TRUNCATE krenalis_events`)
			for _, cur := range []string{test.left, test.right} {
				execUnifiedFixture(t, dw, `INSERT INTO krenalis_events VALUES
('`+unifiedTestID(1)+`','copy','track','Order Completed','2026-01-15','{"total":150,"currency":`+cur+`}')`)
			}
			result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
			if err != nil {
				if !test.fault || result != nil {
					t.Fatalf("expected source fault=%v without partial results, got %+v, %v", test.fault, result, err)
				}
			}
			if err == nil {
				if test.fault {
					t.Fatal("expected source fault, got success")
				}
				assertUnifiedTruths(t, result, []unified.Truth{test.want, unified.NoMatch})
			}
			explanation, err := store.ExplainUnifiedProfile(t.Context(), root, unified.Bindings{}, at, unifiedTestID(2))
			if err != nil {
				t.Fatalf("expected unrelated explanation, got %v", err)
			}
			if explanation.Nodes[0].Truth != unified.NoMatch {
				t.Fatalf("expected unrelated NO_MATCH, got %+v", explanation)
			}
		}
	})

}
