// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
)

// TestUnifiedAveragePostgreSQL runs the Go path on coherent monetary data.
func TestUnifiedAveragePostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	execUnifiedFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
INSERT INTO profiles SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid FROM generate_series(1,4) n;
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,
 timestamp timestamp(3) NOT NULL,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	field, err := unified.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected monetary fixture, got %v", err)
	}
	average := unified.Average{Sum: unified.Sum{
		Scope: unified.Scope{Event: "Order Completed", Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at},
		Field: field, NativeCurrency: "EUR", Minimum: 2,
		Test: unified.SumTest{Operator: unified.GreaterOrEqual, Value: decimal.MustParse("100")}}}

	for _, test := range []struct {
		name        string
		data        []string
		truth       unified.Truth
		sum         string
		count       int64
		certain, hi int64
	}{
		{"empty", nil, unified.NoMatch, "0", 0, 0, 0},
		{"one large insufficient", []string{`{"total":500,"currency":"EUR"}`}, unified.NoMatch, "500", 1, 1, 1},
		{"two known", []string{`{"total":120,"currency":"EUR"}`, `{"total":100,"currency":"EUR"}`}, unified.Match, "220", 2, 2, 2},
		{"below", []string{`{"total":99,"currency":"EUR"}`, `{"total":99,"currency":"EUR"}`}, unified.NoMatch, "198", 2, 2, 2},
		{"negative", []string{`{"total":-120,"currency":"EUR"}`, `{"total":-100,"currency":"EUR"}`}, unified.NoMatch, "-220", 2, 2, 2},
		{"missing amount", []string{`{"total":120,"currency":"EUR"}`, `{"currency":"EUR"}`}, unified.Unknown, "", 2, 2, 2},
		{"string amount", []string{`{"total":120,"currency":"EUR"}`, `{"total":"100","currency":"EUR"}`}, unified.Unknown, "", 2, 2, 2},
		{"null amount", []string{`{"total":120,"currency":"EUR"}`, `{"total":null,"currency":"EUR"}`}, unified.Unknown, "", 2, 2, 2},
		{"too fine amount", []string{`{"total":120,"currency":"EUR"}`, `{"total":100.00001,"currency":"EUR"}`}, unified.Unknown, "", 2, 2, 2},
		{"out of range amount", []string{`{"total":120,"currency":"EUR"}`, `{"total":1e14,"currency":"EUR"}`}, unified.Unknown, "", 2, 2, 2},
		{"unknown currency", []string{`{"total":120,"currency":"EUR"}`, `{"total":100}`}, unified.Unknown, "", -1, 1, 2},
		{"uncertain population above minimum", []string{`{"total":120,"currency":"EUR"}`, `{"total":100,"currency":"EUR"}`, `{"total":200}`}, unified.Unknown, "", -1, 2, 3},
		{"null currency", []string{`{"total":120,"currency":"EUR"}`, `{"total":100,"currency":null}`}, unified.Unknown, "", -1, 1, 2},
		{"incompatible currency", []string{`{"total":120,"currency":"EUR"}`, `{"total":100,"currency":100}`}, unified.Unknown, "", -1, 1, 2},
		{"foreign unknown amount", []string{`{"total":120,"currency":"EUR"}`, `{"currency":"USD"}`}, unified.NoMatch, "120", 1, 1, 1},
		{"verbatim currency", []string{`{"total":120,"currency":"EUR"}`, `{"total":100,"currency":"eur"}`}, unified.NoMatch, "120", 1, 1, 1},
		{"exact beyond float", []string{`{"total":90071992547409.1234,"currency":"EUR"}`, `{"total":0.0001,"currency":"EUR"}`}, unified.Match, "90071992547409.1235", 2, 2, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			execUnifiedFixture(t, dw, "TRUNCATE krenalis_events")
			for i, payload := range test.data {
				execUnifiedFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','a%d','track','Order Completed','2026-01-15','%s')`, unifiedTestID(1), i, payload))
			}
			execUnifiedFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','b1','track','Order Completed','2026-01-15','{"total":100,"currency":"EUR"}'),
('%s','b2','track','Order Completed','2026-01-15','{"total":100,"currency":"EUR"}'),
('%s','wrong-source','track','Refund','2026-01-15','{"total":10000,"currency":"EUR"}'),
('%s','at-T','track','Order Completed','2026-02-01','{"total":10000,"currency":"EUR"}')`,
				unifiedTestID(2), unifiedTestID(2), unifiedTestID(3), unifiedTestID(3)))
			root := &unified.Rule{Average: &average}
			result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
			if err != nil {
				t.Fatalf("expected complete Average evaluation, got %v", err)
			}
			assertUnifiedTruths(t, result, []unified.Truth{test.truth, unified.Match, unified.NoMatch, unified.NoMatch})
			explained, err := store.ExplainUnifiedProfile(t.Context(), root, unified.Bindings{}, at, unifiedTestID(1))
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
			empty, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at,
				unified.Page{First: 99, Limit: 1, MatchingOnly: true})
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
			truth      unified.Truth
			conflict   bool
		}{
			{"one identity", `('same',120,'EUR','PROMO'),('same',120,'EUR','PROMO')`, unified.NoMatch, false},
			{"two identities", `('a',120,'EUR','PROMO'),('b',100,'EUR','PROMO')`, unified.Match, false},
			{"amount conflict", `('same',120,'EUR','PROMO'),('same',100,'EUR','PROMO')`, 0, true},
			{"qualification conflict", `('same',120,'EUR','PROMO'),('same',120,'EUR','OTHER')`, 0, true},
			{"unknown qualification", `('a',120,'EUR','PROMO'),('b',100,'EUR','PROMO'),('c',200,'EUR',NULL)`, unified.Unknown, false},
			{"false qualification unknown reading", `('a',120,'EUR','PROMO'),('b',100,'EUR','PROMO'),('c',NULL,NULL,'OTHER')`, unified.Match, false},
		} {
			t.Run(test.name, func(t *testing.T) {
				execUnifiedFixture(t, dw, "TRUNCATE krenalis_events")
				execUnifiedFixture(t, dw, `INSERT INTO krenalis_events
 SELECT '00000000-0000-0000-0000-000000000001',id,'track','Order Completed','2026-01-15',
 jsonb_strip_nulls(jsonb_build_object('total',total,'currency',currency,'coupon',coupon))
 FROM (VALUES `+test.rows+`) v(id,total,currency,coupon)`)
				qualified := average
				qualified.Where = &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"properties", "coupon"},
						Operator: state.OpIs, Values: []any{json.Value(`"PROMO"`)}},
				}}
				root := &unified.Rule{Average: &qualified}
				result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
				if err != nil {
					if !test.conflict || result != nil {
						t.Fatalf("expected conflict=%v without partial result, got %+v, %v", test.conflict, result, err)
					}
				}
				if err == nil {
					if test.conflict {
						t.Fatal("expected source conflict, got success")
					}
					assertUnifiedTruths(t, result, []unified.Truth{test.truth, unified.NoMatch, unified.NoMatch, unified.NoMatch})
				}
				explained, err := store.ExplainUnifiedProfile(t.Context(), root, unified.Bindings{}, at, unifiedTestID(1))
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
		for _, amounts := range [][2]string{{"0.0001", "0"}, {"0.0001", "0.0001"},
			{"-0.0001", "0"}, {"90071992547409.1234", "90071992547409.1233"}} {
			execUnifiedFixture(t, dw, "TRUNCATE krenalis_events")
			for i, amount := range amounts {
				execUnifiedFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','a%d','track','Order Completed','2026-01-15','{"total":%s,"currency":"EUR"}')`, unifiedTestID(1), i, amount))
			}
			for _, threshold := range []string{"0", "0.0001", "-0.0001", "90071992547409.1234",
				"1e14", "-1e14", "1e75", "-1e75"} {
				for _, operator := range []unified.Comparison{unified.Less, unified.LessOrEqual,
					unified.Greater, unified.GreaterOrEqual} {
					average.Test = unified.SumTest{Operator: operator, Value: decimal.MustParse(threshold)}
					left, _ := new(big.Rat).SetString(amounts[0])
					right, _ := new(big.Rat).SetString(amounts[1])
					test, _ := new(big.Rat).SetString(threshold)
					cmp := new(big.Rat).Add(left, right).Cmp(new(big.Rat).Mul(test, big.NewRat(2, 1)))
					want := operator == unified.Less && cmp < 0 || operator == unified.LessOrEqual && cmp <= 0 ||
						operator == unified.Greater && cmp > 0 || operator == unified.GreaterOrEqual && cmp >= 0
					result, err := store.UnifiedProfiles(t.Context(), &unified.Rule{Average: &average},
						unified.Bindings{}, at, unified.Page{Limit: 10})
					if err != nil {
						t.Fatalf("expected exact comparison for %v with %v %s, got %v", amounts, operator, threshold, err)
					}
					truth := unified.NoMatch
					if want {
						truth = unified.Match
					}
					assertUnifiedTruths(t, result, []unified.Truth{truth, unified.NoMatch, unified.NoMatch, unified.NoMatch})
				}
			}
		}
	})

}
