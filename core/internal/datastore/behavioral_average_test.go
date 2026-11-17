// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
)

// TestBehavioralAveragePostgreSQL runs the Go path on coherent monetary data.
func TestBehavioralAveragePostgreSQL(t *testing.T) {

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
		t.Fatalf("expected monetary fixture, got %v", err)
	}
	average := behavioral.Average{Sum: behavioral.Sum{
		Scope: behavioral.Scope{Event: "Order Completed", Grain: "registered-message", Start: at.AddDate(0, -1, 0),
			End: at},
		Field: field, NativeCurrency: "EUR", Minimum: 2,
		Test: behavioral.SumTest{Operator: behavioral.GreaterOrEqual, Value: decimal.MustParse("100")}}}

	for _, test := range []struct {
		name        string
		data        []string
		truth       behavioral.Truth
		sum         string
		count       int64
		certain, hi int64
	}{
		{"empty", nil, behavioral.NoMatch, "0", 0, 0, 0},
		{"one large insufficient", []string{`{"total":500,"currency":"EUR"}`}, behavioral.NoMatch, "500", 1, 1, 1},
		{"uncertain population below minimum", []string{`{"total":100}`}, behavioral.NoMatch, "", -1, 0, 1},
		{"two known", []string{`{"total":120,"currency":"EUR"}`, `{"total":100,"currency":"EUR"}`}, behavioral.Match, "220", 2, 2, 2},
		{"below", []string{`{"total":99,"currency":"EUR"}`, `{"total":99,"currency":"EUR"}`}, behavioral.NoMatch, "198", 2, 2, 2},
		{"negative", []string{`{"total":-120,"currency":"EUR"}`, `{"total":-100,"currency":"EUR"}`}, behavioral.NoMatch, "-220", 2, 2, 2},
		{"missing amount", []string{`{"total":120,"currency":"EUR"}`, `{"currency":"EUR"}`}, behavioral.Unknown, "", 2, 2, 2},
		{"string amount", []string{`{"total":120,"currency":"EUR"}`, `{"total":"100","currency":"EUR"}`}, behavioral.Unknown, "", 2, 2, 2},
		{"null amount", []string{`{"total":120,"currency":"EUR"}`, `{"total":null,"currency":"EUR"}`}, behavioral.Unknown, "", 2, 2, 2},
		{"too fine amount", []string{`{"total":120,"currency":"EUR"}`, `{"total":100.00001,"currency":"EUR"}`}, behavioral.Unknown, "", 2, 2, 2},
		{"out of range amount", []string{`{"total":120,"currency":"EUR"}`, `{"total":1e14,"currency":"EUR"}`}, behavioral.Unknown, "", 2, 2, 2},
		{"unknown currency", []string{`{"total":120,"currency":"EUR"}`, `{"total":100}`}, behavioral.Unknown, "", -1, 1, 2},
		{"uncertain population above minimum", []string{`{"total":120,"currency":"EUR"}`, `{"total":100,"currency":"EUR"}`, `{"total":200}`}, behavioral.Unknown, "", -1, 2, 3},
		{"null currency", []string{`{"total":120,"currency":"EUR"}`, `{"total":100,"currency":null}`}, behavioral.Unknown, "", -1, 1, 2},
		{"incompatible currency", []string{`{"total":120,"currency":"EUR"}`, `{"total":100,"currency":100}`}, behavioral.Unknown, "", -1, 1, 2},
		{"foreign unknown amount", []string{`{"total":120,"currency":"EUR"}`, `{"currency":"USD"}`}, behavioral.NoMatch, "120", 1, 1, 1},
		{"verbatim currency", []string{`{"total":120,"currency":"EUR"}`, `{"total":100,"currency":"eur"}`}, behavioral.NoMatch, "120", 1, 1, 1},
		{"exact beyond float", []string{`{"total":90071992547409.1234,"currency":"EUR"}`, `{"total":0.0001,"currency":"EUR"}`}, behavioral.Match, "90071992547409.1235", 2, 2, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			execBehavioralFixture(t, dw, "TRUNCATE krenalis_events")
			for i, payload := range test.data {
				execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','a%d','track','Order Completed','2026-01-15','%s')`, behavioralTestID(1), i, payload))
			}
			execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','b1','track','Order Completed','2026-01-15','{"total":100,"currency":"EUR"}'),
('%s','b2','track','Order Completed','2026-01-15','{"total":100,"currency":"EUR"}'),
('%s','wrong-source','track','Refund','2026-01-15','{"total":10000,"currency":"EUR"}'),
('%s','at-T','track','Order Completed','2026-02-01','{"total":10000,"currency":"EUR"}')`,
				behavioralTestID(2), behavioralTestID(2), behavioralTestID(3), behavioralTestID(3)))
			root := &behavioral.Rule{Average: &average}
			result, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
			if err != nil {
				t.Fatalf("expected complete Average evaluation, got %v", err)
			}
			assertBehavioralTruths(t, result,
				[]behavioral.Truth{test.truth, behavioral.Match, behavioral.NoMatch, behavioral.NoMatch})
			explained, err := store.ExplainProfileFilter(t.Context(), root, behavioral.Bindings{}, at,
				behavioralTestID(1))
			if err != nil {
				t.Fatalf("expected exact Average inputs, got %v", err)
			}
			values := explained.Nodes[0].Average
			if values == nil || explained.Nodes[0].Sum != nil || values.Certain != test.certain ||
				values.Possible != test.hi || (values.Count == nil) != (test.count < 0) ||
				(values.Sum == nil) != (test.sum == "") {
				t.Fatalf("expected exact inputs sum=%s count=%d and bounds [%d,%d], got %+v",
					test.sum, test.count, test.certain, test.hi, values)
			}
			if values.Count != nil && *values.Count != test.count ||
				values.Sum != nil && !values.Sum.Equal(decimal.MustParse(test.sum)) {
				t.Fatalf("expected exact sum=%s count=%d, got %+v", test.sum, test.count, values)
			}
			empty, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at,
				behavioral.Page{First: 99, Limit: 1, MatchingOnly: true})
			if err != nil {
				t.Fatalf("expected empty page and complete totals, got %v", err)
			}
			if len(empty.Profiles) != 0 || empty.Match != result.Match || empty.NoMatch != result.NoMatch ||
				empty.Unknown != result.Unknown {
				t.Fatalf("expected unchanged totals on empty page, got %+v", empty)
			}
		})
	}

	t.Run("copies and qualification", func(t *testing.T) {
		for _, test := range []struct {
			name, rows string
			truth      behavioral.Truth
			conflict   bool
		}{
			{"one identity", `('same',120,'EUR','PROMO'),('same',120,'EUR','PROMO')`, behavioral.NoMatch, false},
			{"two identities", `('a',120,'EUR','PROMO'),('b',100,'EUR','PROMO')`, behavioral.Match, false},
			{"amount conflict", `('same',120,'EUR','PROMO'),('same',100,'EUR','PROMO')`, 0, true},
			{"qualification conflict", `('same',120,'EUR','PROMO'),('same',120,'EUR','OTHER')`, 0, true},
			{"unknown qualification", `('a',120,'EUR','PROMO'),('b',100,'EUR','PROMO'),('c',200,'EUR',NULL)`, behavioral.Unknown, false},
			{"false qualification unknown reading", `('a',120,'EUR','PROMO'),('b',100,'EUR','PROMO'),('c',NULL,NULL,'OTHER')`, behavioral.Match, false},
		} {
			t.Run(test.name, func(t *testing.T) {
				execBehavioralFixture(t, dw, "TRUNCATE krenalis_events")
				execBehavioralFixture(t, dw, `INSERT INTO krenalis_events
 SELECT '00000000-0000-0000-0000-000000000001',id,'track','Order Completed','2026-01-15',
 jsonb_strip_nulls(jsonb_build_object('total',total,'currency',currency,'coupon',coupon))
 FROM (VALUES `+test.rows+`) v(id,total,currency,coupon)`)
				qualified := average
				qualified.Where = &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"properties", "coupon"},
						Operator: state.OpIs, Values: []any{json.Value(`"PROMO"`)}},
				}}
				root := &behavioral.Rule{Average: &qualified}
				result, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{
					Limit: 10})
				if err != nil {
					if !test.conflict || result != nil {
						t.Fatalf("expected conflict=%v without partial result, got %+v, %v", test.conflict, result, err)
					}
				}
				if err == nil {
					if test.conflict {
						t.Fatal("expected source conflict, got success")
					}
					assertBehavioralTruths(t, result,
						[]behavioral.Truth{test.truth, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch})
				}
				explained, err := store.ExplainProfileFilter(t.Context(), root, behavioral.Bindings{}, at,
					behavioralTestID(1))
				if err != nil {
					if !test.conflict || explained != nil {
						t.Fatalf("expected conflict=%v without partial explanation, got %+v, %v", test.conflict, explained, err)
					}
					return
				}
				if test.conflict || explained.Nodes[0].Average == nil || explained.Nodes[0].Truth != test.truth {
					t.Fatalf("expected qualified Average %v with conflict=%v, got %+v", test.truth, test.conflict, explained)
				}
			})
		}
	})

	t.Run("exact comparisons", func(t *testing.T) {
		for _, amounts := range [][]string{{"0.0001", "0"}, {"0.0001", "0.0001"},
			{"-0.0001", "0"}, {"90071992547409.1234", "90071992547409.1233"}, {"100", "100", "101"}} {
			execBehavioralFixture(t, dw, "TRUNCATE krenalis_events")
			for i, amount := range amounts {
				execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','a%d','track','Order Completed','2026-01-15','{"total":%s,"currency":"EUR"}')`, behavioralTestID(1), i, amount))
			}
			for _, threshold := range []string{"0", "0.0001", "-0.0001", "90071992547409.1234",
				"1e14", "-1e14", "1e75", "-1e75", "100.3333", "100.3334"} {
				for _, operator := range []behavioral.Comparison{behavioral.Less, behavioral.LessOrEqual,
					behavioral.Greater, behavioral.GreaterOrEqual} {
					average.Test = behavioral.SumTest{Operator: operator, Value: decimal.MustParse(threshold)}
					total := new(big.Rat)
					for _, amount := range amounts {
						value, _ := new(big.Rat).SetString(amount)
						total.Add(total, value)
					}
					test, _ := new(big.Rat).SetString(threshold)
					cmp := total.Cmp(new(big.Rat).Mul(test, big.NewRat(int64(len(amounts)), 1)))
					want := operator == behavioral.Less && cmp < 0 || operator == behavioral.LessOrEqual && cmp <= 0 ||
						operator == behavioral.Greater && cmp > 0 || operator == behavioral.GreaterOrEqual && cmp >= 0
					result, err := store.QueryProfiles(t.Context(), &behavioral.Rule{Average: &average},
						behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
					if err != nil {
						t.Fatalf("expected exact comparison for %v with %v %s, got %v", amounts, operator, threshold, err)
					}
					truth := behavioral.NoMatch
					if want {
						truth = behavioral.Match
					}
					assertBehavioralTruths(t, result,
						[]behavioral.Truth{truth, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch})
				}
			}
		}
	})

}
