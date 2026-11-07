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

// TestUnifiedSumMinimumPostgreSQL checks independent guard and sum outcomes.
func TestUnifiedSumMinimumPostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	// Publish a coherent view; multiply physical rows without adding identities.
	execUnifiedFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
INSERT INTO profiles SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid FROM generate_series(1,3) n;
CREATE TABLE event_fixture (kpid uuid,message_id text,type text,event text,
 timestamp timestamp(3) NOT NULL,properties jsonb);
CREATE VIEW krenalis_events AS SELECT e.* FROM event_fixture e CROSS JOIN (VALUES (1),(2),(3)) copies(n)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	field, err := unified.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected monetary field, got %v", err)
	}
	sum := unified.Sum{Scope: unified.Scope{Event: "Order Completed", Grain: "registered-message",
		Start: at.AddDate(0, -1, 0), End: at}, Field: field, NativeCurrency: "EUR"}

	for _, test := range []struct {
		name, threshold string
		op              unified.Comparison
		minimum         uint64
		data            []string
		where           bool
		certain, unsure int64
		lower, upper    string
		want            unified.Truth
	}{
		{"positive joint impossibility", "100", unified.LessOrEqual, 3, []string{`{"total":50,"currency":"EUR"}`, `{"total":60}`, `{"total":60}`}, false, 1, 2, "50", "170", unified.Unknown},
		{"negative joint impossibility", "0", unified.GreaterOrEqual, 3, []string{`{"total":50,"currency":"EUR"}`, `{"total":-60}`, `{"total":-60}`}, false, 1, 2, "-70", "50", unified.Unknown},
		{"unknown amount below minimum", "0", unified.GreaterOrEqual, 2, []string{`{"currency":"EUR"}`}, false, 1, 0, "", "", unified.NoMatch},
		{"unknown amount counts", "0", unified.GreaterOrEqual, 1, []string{`{"currency":"EUR"}`}, false, 1, 0, "", "", unified.Unknown},
		{"uncertain unknown amount", "0", unified.GreaterOrEqual, 1, []string{`{}`}, false, 0, 1, "", "", unified.Unknown},
		{"certain false sum dominates", "30", unified.GreaterOrEqual, 1, []string{`{"total":20}`}, false, 0, 1, "0", "20", unified.NoMatch},
		{"true sum uncertain minimum", "0", unified.GreaterOrEqual, 1, []string{`{"total":20}`}, false, 0, 1, "0", "20", unified.Unknown},
		{"certain minimum and sum", "30", unified.GreaterOrEqual, 1, []string{`{"total":50,"currency":"EUR"}`, `{"total":20}`}, false, 1, 1, "50", "70", unified.Match},
		{"empty", "0", unified.GreaterOrEqual, 1, nil, false, 0, 0, "0", "0", unified.NoMatch},
		{"foreign unknown excluded", "0", unified.GreaterOrEqual, 2, []string{`{"total":20,"currency":"EUR"}`, `{"currency":"USD"}`}, false, 1, 0, "20", "20", unified.NoMatch},
		{"where false unknown excluded", "0", unified.GreaterOrEqual, 2, []string{`{"total":20,"currency":"EUR","coupon":"PROMO"}`, `{"currency":"EUR","coupon":"OTHER"}`}, true, 1, 0, "20", "20", unified.NoMatch},
		{"where raw NULL", "0", unified.GreaterOrEqual, 1, []string{`{"total":20,"currency":"EUR"}`}, true, 0, 1, "0", "20", unified.Unknown},
		{"where false currency unknown", "0", unified.GreaterOrEqual, 1, []string{`{"total":100,"coupon":"OTHER"}`}, true, 0, 0, "0", "0", unified.NoMatch},
		{"zero amount counts", "0", unified.GreaterOrEqual, 1, []string{`{"total":0,"currency":"EUR"}`}, false, 1, 0, "0", "0", unified.Match},
		{"negative amount counts", "-10", unified.LessOrEqual, 1, []string{`{"total":-20,"currency":"EUR"}`}, false, 1, 0, "-20", "-20", unified.Match},
		{"distinct equal payloads", "40", unified.GreaterOrEqual, 2, []string{`{"total":20,"currency":"EUR"}`, `{"total":20,"currency":"EUR"}`}, false, 2, 0, "40", "40", unified.Match},
		{"int64 minimum", "0", unified.GreaterOrEqual, 9223372036854775807, []string{`{"currency":"EUR"}`}, false, 1, 0, "", "", unified.NoMatch},
	} {
		t.Run(test.name, func(t *testing.T) {

			execUnifiedFixture(t, dw, "TRUNCATE event_fixture")
			for i, data := range test.data {
				execUnifiedFixture(t, dw, fmt.Sprintf(`INSERT INTO event_fixture VALUES
('%s','e%d','track','Order Completed','2026-01-15','%s')`, unifiedTestID(1), i, data))
			}
			for i := range 3 {
				execUnifiedFixture(t, dw, fmt.Sprintf(`INSERT INTO event_fixture VALUES
('%s','other%d','track','Order Completed','2026-01-15','{"total":40,"currency":"EUR","coupon":"PROMO"}')`,
					unifiedTestID(2), i))
			}

			sum.Minimum, sum.Where = test.minimum, nil
			if test.where {
				sum.Where = &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"properties", "coupon"}, Operator: state.OpIs,
						Values: []any{json.Value(`"PROMO"`)}},
				}}
			}
			sum.Test = unified.SumTest{Operator: test.op, Value: decimal.MustParse(test.threshold)}
			rule := unified.Rule{Sum: &sum}
			result, err := store.UnifiedProfiles(t.Context(), &rule, unified.Bindings{}, at, unified.Page{Limit: 10})
			if err != nil {
				t.Fatalf("expected complete guarded Sum, got %v", err)
			}
			other := unified.Match
			if test.op == unified.LessOrEqual || test.minimum > 3 {
				other = unified.NoMatch
			}
			assertUnifiedTruths(t, result, []unified.Truth{test.want, other, unified.NoMatch})

			explained, err := store.ExplainUnifiedProfile(t.Context(), &rule, unified.Bindings{}, at, unifiedTestID(1))
			if err != nil {
				t.Fatalf("expected guarded Sum explanation, got %v", err)
			}
			if explained == nil || len(explained.Nodes) != 1 || explained.Nodes[0].Truth != test.want ||
				explained.Nodes[0].Sum == nil || explained.Nodes[0].Count != nil {
				t.Fatalf("expected Sum explanation with outcome %v, got %+v", test.want, explained)
			}
			bounds := explained.Nodes[0].Sum
			if bounds.Certain != test.certain || bounds.Possible != test.certain+test.unsure {
				t.Fatalf("expected count %d certain and %d uncertain, got %+v", test.certain, test.unsure, bounds)
			}
			if (bounds.Lower == nil) != (test.lower == "") || (bounds.Upper == nil) != (test.upper == "") {
				t.Fatalf("expected bounds [%s,%s], got %+v", test.lower, test.upper, bounds)
			}
			if bounds.Lower != nil && !bounds.Lower.Equal(decimal.MustParse(test.lower)) ||
				bounds.Upper != nil && !bounds.Upper.Equal(decimal.MustParse(test.upper)) {
				t.Fatalf("expected exact bounds [%s,%s], got %+v", test.lower, test.upper, bounds)
			}

			empty, err := store.UnifiedProfiles(t.Context(), &rule, unified.Bindings{}, at,
				unified.Page{First: 99, Limit: 1, MatchingOnly: true})
			if err != nil {
				t.Fatalf("expected empty guarded page, got %v", err)
			}
			if len(empty.Profiles) != 0 || empty.Match != result.Match || empty.NoMatch != result.NoMatch ||
				empty.Unknown != result.Unknown {
				t.Fatalf("expected empty page with complete totals, got %+v", empty)
			}

		})
	}

	t.Run("mixed Count and Sum guards", func(t *testing.T) {

		execUnifiedFixture(t, dw, "TRUNCATE event_fixture")
		for i, data := range []string{`{"total":50,"currency":"EUR"}`, `{"total":60}`, `{"total":60}`} {
			execUnifiedFixture(t, dw, fmt.Sprintf(`INSERT INTO event_fixture VALUES
('%s','mixed%d','track','Order Completed','2026-01-15','%s')`, unifiedTestID(1), i, data))
		}
		sum.Minimum, sum.Where = 3, nil
		sum.Test = unified.SumTest{Operator: unified.LessOrEqual, Value: decimal.MustParse("100")}
		count := unified.Count{Scope: sum.Scope, Minimum: 3,
			Test: unified.CountTest{Operator: unified.Different, Value: 3},
			Where: &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
				&state.WhereCondition{Property: []string{"properties", "currency"}, Operator: state.OpIs,
					Values: []any{json.Value(`"EUR"`)}},
			}}}
		rule := unified.Rule{Operator: state.OpAnd, Children: []unified.Rule{{Count: &count}, {Sum: &sum}}}
		result, err := store.UnifiedProfiles(t.Context(), &rule, unified.Bindings{}, at, unified.Page{Limit: 10})
		if err != nil {
			t.Fatalf("expected mixed guarded query, got %v", err)
		}
		assertUnifiedTruths(t, result, []unified.Truth{unified.Unknown, unified.NoMatch, unified.NoMatch})

		explained, err := store.ExplainUnifiedProfile(t.Context(), &rule, unified.Bindings{}, at, unifiedTestID(1))
		if err != nil {
			t.Fatalf("expected mixed explanation, got %v", err)
		}
		if explained == nil || len(explained.Nodes) != 3 || explained.Nodes[0].Truth != unified.Unknown ||
			explained.Nodes[1].Count == nil || explained.Nodes[1].Sum != nil ||
			explained.Nodes[2].Sum == nil || explained.Nodes[2].Count != nil ||
			explained.Nodes[1].Count.Certain != 1 || explained.Nodes[1].Count.Possible != 3 ||
			explained.Nodes[2].Sum.Certain != 1 || explained.Nodes[2].Sum.Possible != 3 {
			t.Fatalf("expected mixed Count/Sum protocol with count [1,3], got %+v", explained)
		}

	})

	t.Run("false minimum preserves faults", func(t *testing.T) {

		execUnifiedFixture(t, dw, "TRUNCATE event_fixture")
		for _, amount := range []int{20, 30} {
			execUnifiedFixture(t, dw, fmt.Sprintf(`INSERT INTO event_fixture VALUES
('%s','same','track','Order Completed','2026-01-15','{"total":%d,"currency":"EUR"}')`, unifiedTestID(1), amount))
		}

		sum.Minimum, sum.Where = 100, nil
		sum.Test = unified.SumTest{Operator: unified.GreaterOrEqual, Value: decimal.MustParse("30")}
		rule := unified.Rule{Sum: &sum}
		result, err := store.UnifiedProfiles(t.Context(), &rule, unified.Bindings{}, at, unified.Page{Limit: 10})
		if err != nil {
			if result != nil || !strings.Contains(err.Error(), "occurrence source violates") {
				t.Fatalf("expected source failure without partial result, got %+v, %v", result, err)
			}
		}
		if err == nil {
			t.Fatal("expected source failure despite false minimum, got success")
		}

		explained, err := store.ExplainUnifiedProfile(t.Context(), &rule, unified.Bindings{}, at, unifiedTestID(1))
		if err != nil {
			if explained != nil || !strings.Contains(err.Error(), "occurrence source violates") {
				t.Fatalf("expected source failure without explanation, got %+v, %v", explained, err)
			}
			return
		}
		t.Fatal("expected explanation source failure despite false minimum, got success")

	})

}
