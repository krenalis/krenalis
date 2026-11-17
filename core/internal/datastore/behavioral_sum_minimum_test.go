// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
)

// TestBehavioralSumMinimumPostgreSQL checks independent guard and sum outcomes.
func TestBehavioralSumMinimumPostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	// Publish a coherent view; multiply physical rows without adding identities.
	execBehavioralFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
INSERT INTO profiles SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid FROM generate_series(1,3) n;
CREATE TABLE event_fixture (kpid uuid,message_id text,type text,event text,
 timestamp timestamp(3) NOT NULL,properties jsonb);
CREATE VIEW krenalis_events AS SELECT e.* FROM event_fixture e CROSS JOIN (VALUES (1),(2),(3)) copies(n)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	field, err := behavioral.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected monetary field, got %v", err)
	}
	sum := behavioral.Sum{Scope: behavioral.Scope{Event: "Order Completed", Grain: "registered-message",
		Start: at.AddDate(0, -1, 0), End: at}, Field: field, NativeCurrency: "EUR"}

	for _, test := range []struct {
		name, threshold string
		op              behavioral.Comparison
		minimum         uint64
		data            []string
		where           bool
		certain, unsure int64
		lower, upper    string
		want            behavioral.Truth
	}{
		{"positive joint impossibility", "100", behavioral.LessOrEqual, 3, []string{`{"total":50,"currency":"EUR"}`, `{"total":60}`, `{"total":60}`}, false, 1, 2, "50", "170", behavioral.Unknown},
		{"negative joint impossibility", "0", behavioral.GreaterOrEqual, 3, []string{`{"total":50,"currency":"EUR"}`, `{"total":-60}`, `{"total":-60}`}, false, 1, 2, "-70", "50", behavioral.Unknown},
		{"unknown amount below minimum", "0", behavioral.GreaterOrEqual, 2, []string{`{"currency":"EUR"}`}, false, 1, 0, "", "", behavioral.NoMatch},
		{"unknown amount counts", "0", behavioral.GreaterOrEqual, 1, []string{`{"currency":"EUR"}`}, false, 1, 0, "", "", behavioral.Unknown},
		{"uncertain unknown amount", "0", behavioral.GreaterOrEqual, 1, []string{`{}`}, false, 0, 1, "", "", behavioral.Unknown},
		{"certain false sum dominates", "30", behavioral.GreaterOrEqual, 1, []string{`{"total":20}`}, false, 0, 1, "0", "20", behavioral.NoMatch},
		{"true sum uncertain minimum", "0", behavioral.GreaterOrEqual, 1, []string{`{"total":20}`}, false, 0, 1, "0", "20", behavioral.Unknown},
		{"certain minimum and sum", "30", behavioral.GreaterOrEqual, 1, []string{`{"total":50,"currency":"EUR"}`, `{"total":20}`}, false, 1, 1, "50", "70", behavioral.Match},
		{"empty", "0", behavioral.GreaterOrEqual, 1, nil, false, 0, 0, "0", "0", behavioral.NoMatch},
		{"foreign unknown excluded", "0", behavioral.GreaterOrEqual, 2, []string{`{"total":20,"currency":"EUR"}`, `{"currency":"USD"}`}, false, 1, 0, "20", "20", behavioral.NoMatch},
		{"where false unknown excluded", "0", behavioral.GreaterOrEqual, 2, []string{`{"total":20,"currency":"EUR","coupon":"PROMO"}`, `{"currency":"EUR","coupon":"OTHER"}`}, true, 1, 0, "20", "20", behavioral.NoMatch},
		{"where raw NULL", "0", behavioral.GreaterOrEqual, 1, []string{`{"total":20,"currency":"EUR"}`}, true, 0, 1, "0", "20", behavioral.Unknown},
		{"where false currency unknown", "0", behavioral.GreaterOrEqual, 1, []string{`{"total":100,"coupon":"OTHER"}`}, true, 0, 0, "0", "0", behavioral.NoMatch},
		{"zero amount counts", "0", behavioral.GreaterOrEqual, 1, []string{`{"total":0,"currency":"EUR"}`}, false, 1, 0, "0", "0", behavioral.Match},
		{"negative amount counts", "-10", behavioral.LessOrEqual, 1, []string{`{"total":-20,"currency":"EUR"}`}, false, 1, 0, "-20", "-20", behavioral.Match},
		{"distinct equal payloads", "40", behavioral.GreaterOrEqual, 2, []string{`{"total":20,"currency":"EUR"}`, `{"total":20,"currency":"EUR"}`}, false, 2, 0, "40", "40", behavioral.Match},
		{"int64 minimum", "0", behavioral.GreaterOrEqual, 9223372036854775807, []string{`{"currency":"EUR"}`}, false, 1, 0, "", "", behavioral.NoMatch},
	} {
		t.Run(test.name, func(t *testing.T) {

			execBehavioralFixture(t, dw, "TRUNCATE event_fixture")
			for i, data := range test.data {
				execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO event_fixture VALUES
('%s','e%d','track','Order Completed','2026-01-15','%s')`, behavioralTestID(1), i, data))
			}
			for i := range 3 {
				execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO event_fixture VALUES
('%s','other%d','track','Order Completed','2026-01-15','{"total":40,"currency":"EUR","coupon":"PROMO"}')`,
					behavioralTestID(2), i))
			}

			sum.Minimum, sum.Where = test.minimum, nil
			if test.where {
				sum.Where = &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"properties", "coupon"}, Operator: state.OpIs,
						Values: []any{json.Value(`"PROMO"`)}},
				}}
			}
			sum.Test = behavioral.SumTest{Operator: test.op, Value: decimal.MustParse(test.threshold)}
			rule := behavioral.Rule{Sum: &sum}
			result, err := store.QueryProfiles(t.Context(), &rule, behavioral.Bindings{}, at, behavioral.Page{
				Limit: 10})
			if err != nil {
				t.Fatalf("expected complete guarded Sum, got %v", err)
			}
			other := behavioral.Match
			if test.op == behavioral.LessOrEqual || test.minimum > 3 {
				other = behavioral.NoMatch
			}
			assertBehavioralTruths(t, result, []behavioral.Truth{test.want, other, behavioral.NoMatch})

			explained, err := store.ExplainProfileFilter(t.Context(), &rule, behavioral.Bindings{}, at,
				behavioralTestID(1))
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

			empty, err := store.QueryProfiles(t.Context(), &rule, behavioral.Bindings{}, at,
				behavioral.Page{First: 99, Limit: 1, MatchingOnly: true})
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

		execBehavioralFixture(t, dw, "TRUNCATE event_fixture")
		for i, data := range []string{`{"total":50,"currency":"EUR"}`, `{"total":60}`, `{"total":60}`} {
			execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO event_fixture VALUES
('%s','mixed%d','track','Order Completed','2026-01-15','%s')`, behavioralTestID(1), i, data))
		}
		sum.Minimum, sum.Where = 3, nil
		sum.Test = behavioral.SumTest{Operator: behavioral.LessOrEqual, Value: decimal.MustParse("100")}
		count := behavioral.Count{Scope: sum.Scope, Minimum: 3,
			Test: behavioral.CountTest{Operator: behavioral.Different, Value: 3},
			Where: recordCountWhere(t, &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
				&state.WhereCondition{Property: []string{"properties", "currency"}, Operator: state.OpIs,
					Values: []any{json.Value(`"EUR"`)}},
			}})}
		rule := behavioral.Rule{Operator: state.OpAnd, Children: []behavioral.Rule{{Count: &count}, {Sum: &sum}}}
		result, err := store.QueryProfiles(t.Context(), &rule, behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
		if err != nil {
			t.Fatalf("expected mixed guarded query, got %v", err)
		}
		assertBehavioralTruths(t, result,
			[]behavioral.Truth{behavioral.Unknown, behavioral.NoMatch, behavioral.NoMatch})

		explained, err := store.ExplainProfileFilter(t.Context(), &rule, behavioral.Bindings{}, at, behavioralTestID(1))
		if err != nil {
			t.Fatalf("expected mixed explanation, got %v", err)
		}
		if explained == nil || len(explained.Nodes) != 3 || explained.Nodes[0].Truth != behavioral.Unknown ||
			explained.Nodes[1].Count == nil || explained.Nodes[1].Sum != nil ||
			explained.Nodes[2].Sum == nil || explained.Nodes[2].Count != nil ||
			explained.Nodes[1].Count.Certain != 1 || explained.Nodes[1].Count.Possible != 3 ||
			explained.Nodes[2].Sum.Certain != 1 || explained.Nodes[2].Sum.Possible != 3 {
			t.Fatalf("expected mixed Count/Sum protocol with count [1,3], got %+v", explained)
		}

	})

	t.Run("false minimum preserves faults", func(t *testing.T) {

		execBehavioralFixture(t, dw, "TRUNCATE event_fixture")
		for _, amount := range []int{20, 30} {
			execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO event_fixture VALUES
('%s','same','track','Order Completed','2026-01-15','{"total":%d,"currency":"EUR"}')`, behavioralTestID(1), amount))
		}

		sum.Minimum, sum.Where = 100, nil
		sum.Test = behavioral.SumTest{Operator: behavioral.GreaterOrEqual, Value: decimal.MustParse("30")}
		rule := behavioral.Rule{Sum: &sum}
		result, err := store.QueryProfiles(t.Context(), &rule, behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
		if err != nil {
			if result != nil || !strings.Contains(err.Error(), "occurrence source violates") {
				t.Fatalf("expected source failure without partial result, got %+v, %v", result, err)
			}
		}
		if err == nil {
			t.Fatal("expected source failure despite false minimum, got success")
		}

		explained, err := store.ExplainProfileFilter(t.Context(), &rule, behavioral.Bindings{}, at, behavioralTestID(1))
		if err != nil {
			if explained != nil || !strings.Contains(err.Error(), "occurrence source violates") {
				t.Fatalf("expected source failure without explanation, got %+v, %v", explained, err)
			}
			return
		}
		t.Fatal("expected explanation source failure despite false minimum, got success")

	})

}
