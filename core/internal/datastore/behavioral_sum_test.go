// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
)

// TestBehavioralSumPostgreSQL checks monetary bounds on coherent fixtures.
func TestBehavioralSumPostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	execBehavioralFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
INSERT INTO profiles SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid FROM generate_series(1,4) n;
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,
 timestamp timestamp(3) NOT NULL,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	field, err := behavioral.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected monetary field, got %v", err)
	}
	sum := behavioral.Sum{Scope: behavioral.Scope{Event: "Order Completed", Grain: "registered-message",
		Start: at.AddDate(0, -1, 0), End: at}, Field: field, NativeCurrency: "EUR",
		Test: behavioral.SumTest{Operator: behavioral.GreaterOrEqual, Value: decimal.MustParse("30")}}
	for _, test := range []struct {
		name         string
		data         []string
		lower, upper string
		want         behavioral.Truth
	}{
		{"empty", nil, "0", "0", behavioral.NoMatch},
		{"two exact amounts", []string{`{"total":20,"currency":"EUR"}`, `{"total":15,"currency":"EUR"}`}, "35", "35", behavioral.Match},
		{"foreign currency", []string{`{"total":20,"currency":"EUR"}`, `{"total":20,"currency":"USD"}`}, "20", "20", behavioral.NoMatch},
		{"foreign unknown amount", []string{`{"total":20,"currency":"EUR"}`, `{"currency":"USD"}`}, "20", "20", behavioral.NoMatch},
		{"unknown currency", []string{`{"total":20,"currency":"EUR"}`, `{"total":20}`}, "20", "40", behavioral.Unknown},
		{"decisive lower", []string{`{"total":35,"currency":"EUR"}`, `{"total":20}`}, "35", "55", behavioral.Match},
		{"negative optional", []string{`{"total":35,"currency":"EUR"}`, `{"total":-20}`}, "15", "35", behavioral.Unknown},
		{"negative certain", []string{`{"total":-20,"currency":"EUR"}`}, "-20", "-20", behavioral.NoMatch},
		{"missing amount", []string{`{"total":35,"currency":"EUR"}`, `{"currency":"EUR"}`}, "", "", behavioral.Unknown},
		{"missing currency and amount", []string{`{"total":35,"currency":"EUR"}`, `{}`}, "", "", behavioral.Unknown},
		{"missing currency null amount", []string{`{"total":35,"currency":"EUR"}`, `{"total":null}`}, "", "", behavioral.Unknown},
		{"null properties", []string{`{"total":35,"currency":"EUR"}`, `null`}, "", "", behavioral.Unknown},
		{"null amount", []string{`{"total":null,"currency":"EUR"}`}, "", "", behavioral.Unknown},
		{"string amount", []string{`{"total":"35","currency":"EUR"}`}, "", "", behavioral.Unknown},
		{"Boolean amount", []string{`{"total":true,"currency":"EUR"}`}, "", "", behavioral.Unknown},
		{"object amount", []string{`{"total":{},"currency":"EUR"}`}, "", "", behavioral.Unknown},
		{"array amount", []string{`{"total":[],"currency":"EUR"}`}, "", "", behavioral.Unknown},
		{"fractional amount", []string{`{"total":20.00001,"currency":"EUR"}`}, "", "", behavioral.Unknown},
		{"range overflow reading", []string{`{"total":100000000000000,"currency":"EUR"}`}, "", "", behavioral.Unknown},
		{"trailing zeroes", []string{`{"total":30.00000,"currency":"EUR"}`}, "30", "30", behavioral.Match},
		{"exact fraction", []string{`{"total":29.9999,"currency":"EUR"}`, `{"total":0.0001,"currency":"EUR"}`}, "30", "30", behavioral.Match},
		{"precision beyond float", []string{`{"total":90071992547409.1234,"currency":"EUR"}`, `{"total":0.0001,"currency":"EUR"}`}, "90071992547409.1235", "90071992547409.1235", behavioral.Match},
		{"null currency", []string{`{"total":20,"currency":null}`}, "0", "20", behavioral.NoMatch},
		{"number currency", []string{`{"total":40,"currency":100}`}, "0", "40", behavioral.Unknown},
		{"verbatim currency", []string{`{"total":40,"currency":"eur"}`}, "0", "0", behavioral.NoMatch},
	} {
		t.Run(test.name, func(t *testing.T) {
			execBehavioralFixture(t, dw, "TRUNCATE krenalis_events")
			for i, data := range test.data {
				execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','e%d','track','Order Completed','2026-01-15','%s')`, behavioralTestID(1), i, data))
			}
			// A second profile proves that reduction and joins retain the profile key.
			execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','other','track','Order Completed','2026-01-15','{"total":40,"currency":"EUR"}')`, behavioralTestID(2)))
			rule := behavioral.Rule{Count: nil, Sum: &sum}
			result, err := store.QueryProfiles(t.Context(), &rule, behavioral.Bindings{}, at, behavioral.Page{
				Limit: 10})
			if err != nil {
				t.Fatalf("expected complete Sum evaluation, got %v", err)
			}
			assertBehavioralTruths(t, result,
				[]behavioral.Truth{test.want, behavioral.Match, behavioral.NoMatch, behavioral.NoMatch})
			explained, err := store.ExplainProfileFilter(t.Context(), &rule, behavioral.Bindings{}, at,
				behavioralTestID(1))
			if err != nil {
				t.Fatalf("expected complete Sum explanation, got %v", err)
			}
			if explained == nil || len(explained.Nodes) != 1 || explained.Nodes[0].Truth != test.want ||
				explained.Nodes[0].Sum == nil || explained.Nodes[0].Count != nil {
				t.Fatalf("expected Sum bounds with outcome %v, got %+v", test.want, explained)
			}
			bounds := explained.Nodes[0].Sum
			if (bounds.Lower == nil) != (test.lower == "") || (bounds.Upper == nil) != (test.upper == "") {
				t.Fatalf("expected bounds [%s,%s], got %+v", test.lower, test.upper, bounds)
			}
			if bounds.Lower != nil && !bounds.Lower.Equal(decimal.MustParse(test.lower)) ||
				bounds.Upper != nil && !bounds.Upper.Equal(decimal.MustParse(test.upper)) {
				t.Fatalf("expected exact bounds [%s,%s], got %+v", test.lower, test.upper, bounds)
			}
			empty, err := store.QueryProfiles(t.Context(), &rule, behavioral.Bindings{}, at,
				behavioral.Page{First: 99, Limit: 1, MatchingOnly: true})
			if err != nil {
				t.Fatalf("expected empty membership page, got %v", err)
			}
			if len(empty.Profiles) != 0 || empty.Match != result.Match || empty.NoMatch != result.NoMatch ||
				empty.Unknown != result.Unknown {
				t.Fatalf("expected empty page with complete totals, got %+v", empty)
			}
		})
	}

	t.Run("copies", func(t *testing.T) {
		for _, test := range []struct {
			name, first, second, lower string
			conflict                   bool
		}{
			{"equal", `{"total":20,"currency":"EUR"}`, `{"total":20.0000,"currency":"EUR"}`, "20", false},
			{"different", `{"total":20,"currency":"EUR"}`, `{"total":30,"currency":"EUR"}`, "", true},
			{"known missing", `{"total":20,"currency":"EUR"}`, `{"currency":"EUR"}`, "", true},
			{"both unknown", `{"total":null,"currency":"EUR"}`, `{"currency":"EUR"}`, "", false},
			{"excluded different amounts", `{"total":20,"currency":"USD"}`, `{"total":30,"currency":"USD"}`, "0", false},
			{"excluded missing amount", `{"total":20,"currency":"USD"}`, `{"currency":"USD"}`, "0", false},
		} {
			t.Run(test.name, func(t *testing.T) {
				execBehavioralFixture(t, dw, "TRUNCATE krenalis_events")
				for _, data := range []string{test.first, test.second} {
					execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','same','track','Order Completed','2026-01-15','%s')`, behavioralTestID(1), data))
				}
				rule := behavioral.Rule{Sum: &sum}
				result, err := store.QueryProfiles(t.Context(), &rule, behavioral.Bindings{}, at, behavioral.Page{
					Limit: 10})
				if err != nil {
					if !test.conflict || result != nil {
						t.Fatalf("expected source conflict=%v without result, got %+v, %v", test.conflict, result, err)
					}
				}
				if err == nil && test.conflict {
					t.Fatal("expected source contract error, got success")
				}
				explained, err := store.ExplainProfileFilter(t.Context(), &rule, behavioral.Bindings{}, at,
					behavioralTestID(1))
				if err != nil {
					if !test.conflict || explained != nil {
						t.Fatalf("expected source conflict=%v without explanation, got %+v, %v", test.conflict, explained, err)
					}
					return
				}
				if test.conflict || explained == nil || explained.Nodes[0].Sum == nil {
					t.Fatalf("expected consistent deduplicated Sum, got %+v", explained)
				}
				bounds := explained.Nodes[0].Sum
				if (bounds.Lower == nil) != (test.lower == "") || (bounds.Upper == nil) != (test.lower == "") {
					t.Fatalf("expected deduplicated bounds %s, got %+v", test.lower, bounds)
				}
				if test.lower != "" && (!bounds.Lower.Equal(decimal.MustParse(test.lower)) ||
					!bounds.Upper.Equal(decimal.MustParse(test.lower))) {
					t.Fatalf("expected one contribution of %s, got %+v", test.lower, bounds)
				}
			})
		}
	})

	for _, test := range []struct{ name, rows string }{
		{"unrelated missing identity", `('00000000-0000-0000-0000-000000000002',NULL,'track','Order Completed','2026-01-15','{"total":20,"currency":"EUR"}')`},
		{"unrelated amount conflict", `('00000000-0000-0000-0000-000000000002','other','track','Order Completed','2026-01-15','{"total":20,"currency":"EUR"}'),('00000000-0000-0000-0000-000000000002','other','track','Order Completed','2026-01-15','{"total":30,"currency":"EUR"}')`},
		{"unrelated attribution conflict", `('00000000-0000-0000-0000-000000000002','other','track','Order Completed','2026-01-15','{"total":20,"currency":"EUR"}'),(NULL,'other','track','Order Completed','2026-01-15','{"total":20,"currency":"EUR"}')`},
	} {
		t.Run(test.name, func(t *testing.T) {
			execBehavioralFixture(t, dw, "TRUNCATE krenalis_events")
			execBehavioralFixture(t, dw, `INSERT INTO krenalis_events VALUES
('00000000-0000-0000-0000-000000000001','target','track','Order Completed','2026-01-15',
'{"total":40,"currency":"EUR"}'),`+test.rows)
			rule := behavioral.Rule{Sum: &sum}
			explained, err := store.ExplainProfileFilter(t.Context(), &rule, behavioral.Bindings{}, at,
				behavioralTestID(1))
			if err != nil {
				t.Fatalf("expected unaffected Sum explanation, got %v", err)
			}
			if explained == nil || len(explained.Nodes) != 1 || explained.Nodes[0].Truth != behavioral.Match ||
				explained.Nodes[0].Sum == nil || explained.Nodes[0].Sum.Lower == nil ||
				explained.Nodes[0].Sum.Upper == nil || !explained.Nodes[0].Sum.Lower.Equal(decimal.MustParse("40")) ||
				!explained.Nodes[0].Sum.Upper.Equal(decimal.MustParse("40")) {
				t.Fatalf("expected target MATCH with exact Sum [40,40], got %+v", explained)
			}
			explained, err = store.ExplainProfileFilter(t.Context(), &rule, behavioral.Bindings{}, at,
				behavioralTestID(2))
			if err != nil {
				if explained != nil || !strings.Contains(err.Error(), "occurrence source violates") {
					t.Fatalf("expected source-contract failure without explanation, got %+v, %v", explained, err)
				}
			}
			if err == nil {
				t.Fatal("expected affected profile source-contract failure, got success")
			}
			result, err := store.QueryProfiles(t.Context(), &rule, behavioral.Bindings{}, at, behavioral.Page{
				Limit: 10})
			if err != nil {
				if result != nil || !strings.Contains(err.Error(), "occurrence source violates") {
					t.Fatalf("expected source-contract failure without partial result, got %+v, %v", result, err)
				}
				return
			}
			t.Fatal("expected population source-contract failure, got success")
		})
	}

	for _, test := range []struct {
		name, literal string
		operator      behavioral.Comparison
		large         bool
		want          []behavioral.Truth
	}{
		{"at least 1000", "1000", behavioral.GreaterOrEqual, false, []behavioral.Truth{behavioral.NoMatch, behavioral.NoMatch, behavioral.Match, behavioral.Match}},
		{"above 999.9999", "999.9999", behavioral.Greater, false, []behavioral.Truth{behavioral.NoMatch, behavioral.NoMatch, behavioral.Match, behavioral.Match}},
		{"at most 1000", "1000", behavioral.LessOrEqual, false, []behavioral.Truth{behavioral.Match, behavioral.Match, behavioral.Match, behavioral.NoMatch}},
		{"below 1000.0001", "1000.0001", behavioral.Less, false, []behavioral.Truth{behavioral.Match, behavioral.Match, behavioral.Match, behavioral.NoMatch}},
		{"at least 5000", "5000", behavioral.GreaterOrEqual, false, []behavioral.Truth{behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch}},
		{"positive clipped lower", "1e75", behavioral.GreaterOrEqual, true, []behavioral.Truth{behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch, behavioral.Unknown}},
		{"positive clipped upper", "1e75", behavioral.LessOrEqual, true, []behavioral.Truth{behavioral.Match, behavioral.Match, behavioral.Match, behavioral.Unknown}},
		{"negative clipped lower", "-1e75", behavioral.GreaterOrEqual, true, []behavioral.Truth{behavioral.Match, behavioral.Match, behavioral.Match, behavioral.Unknown}},
		{"negative clipped upper", "-1e75", behavioral.Less, true, []behavioral.Truth{behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch, behavioral.Unknown}},
	} {
		t.Run(test.name, func(t *testing.T) {
			execBehavioralFixture(t, dw, "TRUNCATE krenalis_events")
			amounts := [][]string{{"150"}, {"999.9999"}, {"1000"}, {"1500"}}
			totals := []string{"150", "999.9999", "1000", "1500"}
			if test.large {
				amounts = [][]string{
					{"99999999999999.9999", "99999999999999.9999", "99999999999999.9999"},
					{"-99999999999999.9999", "-99999999999999.9999", "-99999999999999.9999"}, nil, {"null"},
				}
				totals = []string{"299999999999999.9997", "-299999999999999.9997", "0", ""}
			}
			for i, values := range amounts {
				for j, value := range values {
					execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','threshold-%d-%d','track','Order Completed','2026-01-15','{"total":%s,"currency":"EUR"}')`,
						behavioralTestID(i+1), i, j, value))
				}
			}
			thresholdSum := sum
			thresholdSum.Test = behavioral.SumTest{Operator: test.operator, Value: decimal.MustParse(test.literal)}
			rule := behavioral.Rule{Sum: &thresholdSum}
			result, err := store.QueryProfiles(t.Context(), &rule, behavioral.Bindings{}, at, behavioral.Page{
				Limit: 10})
			if err != nil {
				t.Fatalf("expected exact Sum threshold evaluation, got %v", err)
			}
			assertBehavioralTruths(t, result, test.want)
			for i, want := range test.want {
				explained, err := store.ExplainProfileFilter(t.Context(), &rule, behavioral.Bindings{}, at,
					behavioralTestID(i+1))
				if err != nil {
					t.Fatalf("expected exact Sum threshold explanation, got %v", err)
				}
				if explained == nil || len(explained.Nodes) != 1 || explained.Nodes[0].Truth != want ||
					explained.Nodes[0].Sum == nil {
					t.Fatalf("expected explanation outcome %v, got %+v", want, explained)
				}
				bounds := explained.Nodes[0].Sum
				if (bounds.Lower == nil) != (totals[i] == "") || (bounds.Upper == nil) != (totals[i] == "") {
					t.Fatalf("expected threshold bounds %s, got %+v", totals[i], bounds)
				}
				if totals[i] != "" && (!bounds.Lower.Equal(decimal.MustParse(totals[i])) ||
					!bounds.Upper.Equal(decimal.MustParse(totals[i]))) {
					t.Fatalf("expected exact bounds %s, got %+v", totals[i], bounds)
				}
			}
		})
	}

	// Independent oracle enumerates actual discrete sums; bounds are not assumed.
	execBehavioralFixture(t, dw, "TRUNCATE krenalis_events,profiles")
	for i := range 27 {
		execBehavioralFixture(t, dw, fmt.Sprintf("INSERT INTO profiles VALUES ('%s')", behavioralTestID(i+1)))
		for j, amount := range []int{-2, 1, 3} {
			flag := i
			for range j {
				flag /= 3
			}
			currency := []string{`,"currency":"USD"`, `,"currency":"EUR"`, ``}[flag%3]
			execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','o%d/%d','track','Order Completed','2026-01-15','{"total":%d%s}')`, behavioralTestID(i+1), i, j, amount,
				currency))
		}
	}
	for _, op := range []behavioral.Comparison{behavioral.Less, behavioral.LessOrEqual, behavioral.Greater, behavioral.GreaterOrEqual} {
		for _, literal := range []string{"-3.00001", "-3", "0", "0.00001", "1.00001", "4", "1e-37", "1e75", "-1e75"} {
			sum.Test = behavioral.SumTest{Operator: op, Value: decimal.MustParse(literal)}
			rule := behavioral.Rule{Sum: &sum}
			result, err := store.QueryProfiles(t.Context(), &rule, behavioral.Bindings{}, at, behavioral.Page{
				Limit: 100})
			if err != nil {
				t.Fatalf("expected oracle query op%d %s, got %v", op, literal, err)
			}
			want := make([]behavioral.Truth, 27)
			threshold, _ := new(big.Rat).SetString(literal)
			for i := range 27 {
				possible := []int{0}
				certain, uncertain := int64(0), int64(0)
				flags := i
				for _, amount := range []int{-2, 1, 3} {
					flag := flags % 3
					flags /= 3
					if flag == 1 {
						certain++
					}
					if flag == 2 {
						uncertain++
					}
					next := []int{}
					for _, total := range possible {
						if flag != 1 {
							next = append(next, total)
						}
						if flag != 0 {
							next = append(next, total+amount)
						}
					}
					possible = next
				}
				yes, no := false, false
				for _, total := range possible {
					comparison := new(big.Rat).SetInt64(int64(total)).Cmp(threshold)
					match := false
					switch op {
					case behavioral.Less:
						match = comparison < 0
					case behavioral.LessOrEqual:
						match = comparison <= 0
					case behavioral.Greater:
						match = comparison > 0
					case behavioral.GreaterOrEqual:
						match = comparison >= 0
					default:
						t.Fatalf("expected ordered test, got %d", op)
					}
					yes, no = yes || match, no || !match
				}
				want[i] = behavioral.NoMatch
				if yes {
					want[i] = behavioral.Match
				}
				if yes && no {
					want[i] = behavioral.Unknown
				}
				if op == behavioral.Less && literal == "0" {
					lo, hi := possible[0], possible[0]
					for _, total := range possible {
						lo, hi = min(lo, total), max(hi, total)
					}
					explained, err := store.ExplainProfileFilter(t.Context(), &rule, behavioral.Bindings{}, at,
						behavioralTestID(i+1))
					if err != nil {
						t.Fatalf("expected oracle extrema explanation, got %v", err)
					}
					bounds := explained.Nodes[0].Sum
					if bounds == nil || bounds.Lower == nil || bounds.Upper == nil ||
						!bounds.Lower.Equal(decimal.MustInt(lo)) || !bounds.Upper.Equal(decimal.MustInt(hi)) ||
						bounds.Certain != certain || bounds.Possible != certain+uncertain {
						t.Fatalf("expected reachable extrema [%d,%d] from %v, got %+v", lo, hi, possible, bounds)
					}
				}
			}
			assertBehavioralTruths(t, result, want)
			for _, minimum := range []uint64{1, 2, 3, 4} {
				sum.Minimum = minimum
				guarded := make([]behavioral.Truth, len(want))
				for i, truth := range want {
					flags := []int{i % 3, i / 3 % 3, i / 9 % 3}
					guard := independentCountOracle(flags, minimum,
						behavioral.CountTest{Operator: behavioral.GreaterOrEqual})
					guarded[i] = behavioral.Match
					if guard == behavioral.Unknown || truth == behavioral.Unknown {
						guarded[i] = behavioral.Unknown
					}
					if guard == behavioral.NoMatch || truth == behavioral.NoMatch {
						guarded[i] = behavioral.NoMatch
					}
				}
				result, err := store.QueryProfiles(t.Context(), &rule, behavioral.Bindings{}, at, behavioral.Page{
					Limit: 100})
				if err != nil {
					t.Fatalf("expected guarded oracle query minimum %d op%d %s, got %v", minimum, op, literal, err)
				}
				assertBehavioralTruths(t, result, guarded)
			}
			sum.Minimum = 0
		}
	}

	t.Run("time and cancellation", func(t *testing.T) {
		execBehavioralFixture(t, dw, "TRUNCATE krenalis_events,profiles")
		execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO profiles VALUES ('%s');
INSERT INTO krenalis_events VALUES
('%s','before','track','Order Completed','2025-12-31 23:59:59.999','{"total":-1000,"currency":"EUR"}'),
('%s','start','track','Order Completed','2026-01-01','{"total":20,"currency":"EUR"}'),
('%s','end-before','track','Order Completed','2026-01-31 23:59:59.999','{"total":10,"currency":"EUR"}'),
('%s','end','track','Order Completed','2026-02-01','{"total":1000,"currency":"EUR"}'),
('%s','future','track','Order Completed','2026-02-02','{"total":1000,"currency":"EUR"}')`,
			behavioralTestID(1), behavioralTestID(1), behavioralTestID(1), behavioralTestID(1),
			behavioralTestID(1), behavioralTestID(1)))
		sum.Test = behavioral.SumTest{Operator: behavioral.GreaterOrEqual, Value: decimal.MustParse("30")}
		rule := behavioral.Rule{Sum: &sum}
		result, err := store.QueryProfiles(t.Context(), &rule, behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
		if err != nil {
			t.Fatalf("expected half-open Sum period, got %v", err)
		}
		assertBehavioralTruths(t, result, []behavioral.Truth{behavioral.Match})
		sum.Scope.Start = sum.Scope.Start.Add(time.Microsecond)
		result, err = store.QueryProfiles(t.Context(), &rule, behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
		if err != nil {
			t.Fatalf("expected microsecond lower bound, got %v", err)
		}
		assertBehavioralTruths(t, result, []behavioral.Truth{behavioral.NoMatch})
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		result, err = store.QueryProfiles(ctx, &rule, behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
		if err != nil {
			if result != nil {
				t.Fatalf("expected cancellation without partial result, got %+v", result)
			}
		}
		if err == nil {
			t.Fatal("expected cancellation failure, got success")
		}
		explained, err := store.ExplainProfileFilter(ctx, &rule, behavioral.Bindings{}, at, behavioralTestID(1))
		if err != nil {
			if explained != nil {
				t.Fatalf("expected cancellation without explanation, got %+v", explained)
			}
		}
		if err == nil {
			t.Fatal("expected canceled explanation failure, got success")
		}
		execBehavioralFixture(t, dw, "TRUNCATE profiles")
		result, err = store.QueryProfiles(t.Context(), &rule, behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
		if err != nil {
			t.Fatalf("expected empty profile population, got %v", err)
		}
		if len(result.Profiles) != 0 || result.Match != 0 || result.NoMatch != 0 || result.Unknown != 0 {
			t.Fatalf("expected empty Sum population with zero totals, got %+v", result)
		}
	})

}

// TestBehavioralSumWherePostgreSQL closes independent Record qualifications.
func TestBehavioralSumWherePostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	execBehavioralFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
INSERT INTO profiles SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid FROM generate_series(1,4) n;
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,
 timestamp timestamp(3) NOT NULL,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	field, err := behavioral.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected monetary field, got %v", err)
	}
	sum := behavioral.Sum{Scope: behavioral.Scope{Event: "Order Completed", Grain: "registered-message",
		Start: at.AddDate(0, -1, 0), End: at}, Field: field, NativeCurrency: "EUR",
		Where: &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
			&state.WhereCondition{Property: []string{"properties", "coupon"}, Operator: state.OpIs,
				Values: []any{json.Value(`"PROMO"`)}},
		}}, Test: behavioral.SumTest{Operator: behavioral.GreaterOrEqual, Value: decimal.MustParse("30")}}
	for _, test := range []struct {
		name         string
		data         []string
		lower, upper string
		want         behavioral.Truth
	}{
		{"true", []string{`{"total":40,"currency":"EUR","coupon":"PROMO"}`}, "40", "40", behavioral.Match},
		{"false", []string{`{"total":40,"currency":"EUR","coupon":"OTHER"}`}, "0", "0", behavioral.NoMatch},
		{"unknown positive", []string{`{"total":40,"currency":"EUR"}`}, "0", "40", behavioral.Unknown},
		{"unknown negative", []string{`{"total":-40,"currency":"EUR"}`}, "-40", "0", behavioral.NoMatch},
		{"decisive lower", []string{`{"total":35,"currency":"EUR","coupon":"PROMO"}`, `{"total":20,"currency":"EUR"}`}, "35", "55", behavioral.Match},
		{"optional negative", []string{`{"total":35,"currency":"EUR","coupon":"PROMO"}`, `{"total":-20,"currency":"EUR"}`}, "15", "35", behavioral.Unknown},
		{"unknown amount and where", []string{`{"total":35,"currency":"EUR","coupon":"PROMO"}`, `{"currency":"EUR"}`}, "", "", behavioral.Unknown},
		{"false unknown amount", []string{`{"currency":"EUR","coupon":"OTHER"}`}, "0", "0", behavioral.NoMatch},
		{"unknown currency and where", []string{`{"total":40}`}, "0", "40", behavioral.Unknown},
		{"false unknown currency", []string{`{"total":40,"coupon":"OTHER"}`}, "0", "0", behavioral.NoMatch},
		{"false all unknown", []string{`{"coupon":"OTHER"}`}, "0", "0", behavioral.NoMatch},
		{"null coupon", []string{`{"total":40,"currency":"EUR","coupon":null}`}, "0", "0", behavioral.NoMatch},
		{"number coupon", []string{`{"total":40,"currency":"EUR","coupon":100}`}, "0", "0", behavioral.NoMatch},
		{"foreign currency", []string{`{"total":40,"currency":"USD","coupon":"PROMO"}`}, "0", "0", behavioral.NoMatch},
	} {
		t.Run(test.name, func(t *testing.T) {
			execBehavioralFixture(t, dw, "TRUNCATE krenalis_events")
			for i, data := range test.data {
				execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','e%d','track','Order Completed','2026-01-15','%s')`, behavioralTestID(1), i, data))
			}
			execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','other','track','Order Completed','2026-01-15','{"total":40,"currency":"EUR","coupon":"PROMO"}')`,
				behavioralTestID(2)))
			rule := behavioral.Rule{Sum: &sum}
			result, err := store.QueryProfiles(t.Context(), &rule, behavioral.Bindings{}, at, behavioral.Page{
				Limit: 10})
			if err != nil {
				t.Fatalf("expected qualified Sum evaluation, got %v", err)
			}
			assertBehavioralTruths(t, result,
				[]behavioral.Truth{test.want, behavioral.Match, behavioral.NoMatch, behavioral.NoMatch})
			explained, err := store.ExplainProfileFilter(t.Context(), &rule, behavioral.Bindings{}, at,
				behavioralTestID(1))
			if err != nil {
				t.Fatalf("expected qualified Sum explanation, got %v", err)
			}
			if explained == nil || len(explained.Nodes) != 1 || explained.Nodes[0].Sum == nil ||
				explained.Nodes[0].Truth != test.want {
				t.Fatalf("expected qualified Sum outcome %v, got %+v", test.want, explained)
			}
			bounds := explained.Nodes[0].Sum
			if (bounds.Lower == nil) != (test.lower == "") || (bounds.Upper == nil) != (test.upper == "") {
				t.Fatalf("expected bounds [%s,%s], got %+v", test.lower, test.upper, bounds)
			}
			if bounds.Lower != nil && !bounds.Lower.Equal(decimal.MustParse(test.lower)) ||
				bounds.Upper != nil && !bounds.Upper.Equal(decimal.MustParse(test.upper)) {
				t.Fatalf("expected reachable bounds [%s,%s], got %+v", test.lower, test.upper, bounds)
			}
		})
	}

	// Reachability is relative to Record/Kleene outcomes, not raw completions.
	for _, test := range []struct {
		name     string
		operator state.WhereLogical
	}{
		{"and", state.OpAnd},
		{"or", state.OpOr},
	} {
		t.Run("correlated Record leaves/"+test.name, func(t *testing.T) {
			execBehavioralFixture(t, dw, "TRUNCATE krenalis_events")
			execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','correlated','track','Order Completed','2026-01-15','{"total":40,"currency":"EUR"}')`, behavioralTestID(1)))
			qualified := sum
			qualified.Where = &state.Where{Operator: test.operator, Rules: []state.WhereRule{
				&state.WhereCondition{Property: []string{"properties", "coupon"}, Operator: state.OpIs,
					Values: []any{json.Value(`"PROMO"`)}},
				&state.WhereCondition{Property: []string{"properties", "coupon"}, Operator: state.OpIsNot,
					Values: []any{json.Value(`"PROMO"`)}},
			}}
			rule := behavioral.Rule{Sum: &qualified}
			result, err := store.QueryProfiles(t.Context(), &rule, behavioral.Bindings{}, at, behavioral.Page{
				Limit: 10})
			if err != nil {
				t.Fatalf("expected Record/Kleene Sum evaluation, got %v", err)
			}
			want := []behavioral.Truth{behavioral.Unknown, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch}
			assertBehavioralTruths(t, result, want)
			explained, err := store.ExplainProfileFilter(t.Context(), &rule, behavioral.Bindings{}, at,
				behavioralTestID(1))
			if err != nil {
				t.Fatalf("expected Record/Kleene Sum explanation, got %v", err)
			}
			if explained == nil || len(explained.Nodes) != 1 || explained.Nodes[0].Truth != behavioral.Unknown ||
				explained.Nodes[0].Sum == nil || explained.Nodes[0].Sum.Lower == nil ||
				explained.Nodes[0].Sum.Upper == nil || !explained.Nodes[0].Sum.Lower.Equal(decimal.MustInt(0)) ||
				!explained.Nodes[0].Sum.Upper.Equal(decimal.MustInt(40)) {
				t.Fatalf("expected UNKNOWN with qualification bounds [0,40], got %+v", explained)
			}
		})
	}

	for _, test := range []struct {
		name, first, second, lower, upper string
		conflict, distinct                bool
	}{
		{"true false conflict", `{"total":40,"currency":"EUR","coupon":"PROMO"}`, `{"total":40,"currency":"EUR","coupon":"OTHER"}`, "", "", true, false},
		{"true unknown conflict", `{"total":40,"currency":"EUR","coupon":"PROMO"}`, `{"total":40,"currency":"EUR"}`, "", "", true, false},
		{"false unknown conflict", `{"total":40,"currency":"EUR","coupon":"OTHER"}`, `{"total":40,"currency":"EUR"}`, "", "", true, false},
		{"consistent unknown", `{"total":40,"currency":"EUR"}`, `{"total":40,"currency":"EUR"}`, "0", "40", false, false},
		{"excluded amounts", `{"total":40,"currency":"EUR","coupon":"OTHER"}`, `{"currency":"EUR","coupon":"OTHER"}`, "0", "0", false, false},
		{"distinct messages", `{"total":40,"currency":"EUR","coupon":"PROMO"}`, `{"total":40,"currency":"EUR","coupon":"PROMO"}`, "80", "80", false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			execBehavioralFixture(t, dw, "TRUNCATE krenalis_events")
			execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','target','track','Order Completed','2026-01-15','{"total":40,"currency":"EUR","coupon":"PROMO"}')`,
				behavioralTestID(1)))
			for i, data := range []string{test.first, test.second} {
				id := "same"
				if test.distinct {
					id = fmt.Sprintf("distinct%d", i)
				}
				execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','%s','track','Order Completed','2026-01-15','%s')`, behavioralTestID(2), id, data))
			}
			rule := behavioral.Rule{Sum: &sum}
			target, err := store.ExplainProfileFilter(t.Context(), &rule, behavioral.Bindings{}, at,
				behavioralTestID(1))
			if err != nil {
				t.Fatalf("expected unaffected target explanation, got %v", err)
			}
			if target == nil || len(target.Nodes) != 1 || target.Nodes[0].Truth != behavioral.Match ||
				target.Nodes[0].Sum == nil || target.Nodes[0].Sum.Lower == nil || target.Nodes[0].Sum.Upper == nil ||
				!target.Nodes[0].Sum.Lower.Equal(decimal.MustParse("40")) ||
				!target.Nodes[0].Sum.Upper.Equal(decimal.MustParse("40")) {
				t.Fatalf("expected unaffected target bounds [40,40], got %+v", target)
			}
			result, err := store.QueryProfiles(t.Context(), &rule, behavioral.Bindings{}, at, behavioral.Page{
				Limit: 10})
			if err != nil {
				if !test.conflict || result != nil || !strings.Contains(err.Error(), "occurrence source violates") {
					t.Fatalf("expected source conflict=%v without partial result, got %+v, %v", test.conflict, result, err)
				}
			}
			if err == nil && test.conflict {
				t.Fatal("expected qualification source-contract failure, got success")
			}
			explained, err := store.ExplainProfileFilter(t.Context(), &rule, behavioral.Bindings{}, at,
				behavioralTestID(2))
			if err != nil {
				if !test.conflict || explained != nil || !strings.Contains(err.Error(), "occurrence source violates") {
					t.Fatalf("expected source conflict=%v without explanation, got %+v, %v", test.conflict, explained, err)
				}
				return
			}
			if test.conflict || explained == nil || len(explained.Nodes) != 1 || explained.Nodes[0].Sum == nil {
				t.Fatalf("expected qualified deduplicated Sum, got %+v", explained)
			}
			bounds := explained.Nodes[0].Sum
			if bounds.Lower == nil || bounds.Upper == nil || !bounds.Lower.Equal(decimal.MustParse(test.lower)) ||
				!bounds.Upper.Equal(decimal.MustParse(test.upper)) {
				t.Fatalf("expected copy bounds [%s,%s], got %+v", test.lower, test.upper, bounds)
			}
		})
	}

}
