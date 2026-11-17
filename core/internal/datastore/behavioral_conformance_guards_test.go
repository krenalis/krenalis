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
	"github.com/krenalis/krenalis/warehouses"
)

// testBehavioralConformanceGuards batches the discriminating Sum/Where/Minimum
// and Average fixtures, with exact metadata for every profile.
func testBehavioralConformanceGuards(t *testing.T, store *Store, dw warehouses.Warehouse, table warehouses.Table) {

	t.Helper()
	execBehavioralFixture(t, dw, "TRUNCATE TABLE profiles; TRUNCATE TABLE krenalis_events")
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	fixtures := []struct {
		name              string
		payloads          []string
		certain, possible int64
		lower, upper      string
	}{
		{"empty", nil, 0, 0, "0", "0"},
		{"known", []string{`{"total":50,"currency":"EUR","coupon":"PROMO"}`}, 1, 1, "50", "50"},
		{"positive joint impossibility", []string{`{"total":50,"currency":"EUR","coupon":"PROMO"}`, `{"total":60,"coupon":"PROMO"}`, `{"total":60,"coupon":"PROMO"}`}, 1, 3, "50", "170"},
		{"negative joint impossibility", []string{`{"total":50,"currency":"EUR","coupon":"PROMO"}`, `{"total":-60,"coupon":"PROMO"}`, `{"total":-60,"coupon":"PROMO"}`}, 1, 3, "-70", "50"},
		{"unknown amount counts", []string{`{"currency":"EUR","coupon":"PROMO"}`}, 1, 1, "", ""},
		{"known pair", []string{`{"total":20,"currency":"EUR","coupon":"PROMO"}`, `{"total":40,"currency":"EUR","coupon":"PROMO"}`}, 2, 2, "60", "60"},
		{"unknown where positive", []string{`{"total":35,"currency":"EUR","coupon":"PROMO"}`, `{"total":20,"currency":"EUR"}`}, 1, 2, "35", "55"},
		{"unknown where negative", []string{`{"total":35,"currency":"EUR","coupon":"PROMO"}`, `{"total":-20,"currency":"EUR"}`}, 1, 2, "15", "35"},
		{"foreign unreadable excluded", []string{`{"currency":"USD","coupon":"PROMO"}`}, 0, 0, "0", "0"},
		{"false where excludes unknown", []string{`{"coupon":"OTHER"}`}, 0, 0, "0", "0"},
		{"raw null where", []string{`{"total":100,"currency":"EUR","coupon":null}`}, 0, 0, "0", "0"},
		{"distinct equal values", []string{`{"total":20,"currency":"EUR","coupon":"PROMO"}`, `{"total":20,"currency":"EUR","coupon":"PROMO"}`}, 2, 2, "40", "40"},
		{"exact sum beyond float", []string{`{"total":90071992547409.1234,"currency":"EUR","coupon":"PROMO"}`, `{"total":0.0001,"currency":"EUR","coupon":"PROMO"}`}, 2, 2, "90071992547409.1235", "90071992547409.1235"},
		{"too fine amount", []string{`{"total":100.00001,"currency":"EUR","coupon":"PROMO"}`, `{"total":20,"currency":"EUR","coupon":"PROMO"}`}, 2, 2, "", ""},
		{"out of domain amount", []string{`{"total":1e14,"currency":"EUR","coupon":"PROMO"}`, `{"total":20,"currency":"EUR","coupon":"PROMO"}`}, 2, 2, "", ""},
		{"string amount", []string{`{"total":"100","currency":"EUR","coupon":"PROMO"}`, `{"total":20,"currency":"EUR","coupon":"PROMO"}`}, 2, 2, "", ""},
		{"unknown currency and where", []string{`{}`}, 0, 1, "", ""},
	}
	var profiles []string
	var rows [][]any
	for p, fixture := range fixtures {
		profiles = append(profiles, "('"+behavioralTestID(p+1)+"')")
		for i, payload := range fixture.payloads {
			rows = append(rows, []any{behavioralTestID(p + 1), fmt.Sprintf("guard/%d/%d", p, i), "track",
				"Order Completed", nil, at.Add(-time.Hour), at, at, json.Value(payload)})
		}
	}
	execBehavioralFixture(t, dw, "INSERT INTO profiles VALUES "+strings.Join(profiles, ","))
	err := dw.Merge(t.Context(), table, rows, nil)
	if err != nil {
		t.Fatalf("expected real monetary guard writes, got %v", err)
	}
	// Physical copies must not satisfy a sample minimum or change a total.
	execBehavioralFixture(t, dw, "INSERT INTO krenalis_events SELECT * FROM krenalis_events")
	field, err := behavioral.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected monetary field, got %v", err)
	}
	const m, f, u = behavioral.Match, behavioral.NoMatch, behavioral.Unknown
	for _, test := range []struct {
		name      string
		average   bool
		operator  behavioral.Comparison
		threshold string
		minimum   uint64
		want      []behavioral.Truth
	}{
		{"sum at least 30", false, behavioral.GreaterOrEqual, "30", 0, []behavioral.Truth{f, m, m, u, u, m, m, u, f, f, f, m, m, u, u, u, u}},
		{"sum at most 100 minimum 3", false, behavioral.LessOrEqual, "100", 3, []behavioral.Truth{f, f, u, u, f, f, f, f, f, f, f, f, f, f, f, f, f}},
		{"sum nonnegative minimum 3", false, behavioral.GreaterOrEqual, "0", 3, []behavioral.Truth{f, f, u, u, f, f, f, f, f, f, f, f, f, f, f, f, f}},
		{"sum nonnegative minimum 1", false, behavioral.GreaterOrEqual, "0", 1, []behavioral.Truth{f, m, m, u, u, m, m, m, f, f, f, m, m, u, u, u, u}},
		{"sum at most 30 minimum 1", false, behavioral.LessOrEqual, "30", 1, []behavioral.Truth{f, f, f, u, u, f, f, u, f, f, f, f, f, u, u, u, u}},
		{"average at least 30 minimum 2", true, behavioral.GreaterOrEqual, "30", 2, []behavioral.Truth{f, f, u, u, f, m, u, u, f, f, f, f, m, u, u, u, f}},
		{"average below 30 minimum 2", true, behavioral.Less, "30", 2, []behavioral.Truth{f, f, u, u, f, f, u, u, f, f, f, m, f, u, u, u, f}},
		{"average threshold outside domain", true, behavioral.GreaterOrEqual, "1e14", 1, []behavioral.Truth{f, f, u, u, u, f, u, u, f, f, f, f, f, u, u, u, u}},
	} {
		t.Run(test.name, func(t *testing.T) {

			sum := behavioral.Sum{Scope: behavioral.Scope{Event: "Order Completed", Grain: "registered-message",
				Start: at.Add(-24 * time.Hour), End: at}, Field: field, NativeCurrency: "EUR", Minimum: test.minimum,
				Where: &state.Where{Rules: []state.WhereRule{&state.WhereCondition{
					Property: []string{"properties", "coupon"}, Operator: state.OpIs,
					Values: []any{json.Value(`"PROMO"`)}}}},
				Test: behavioral.SumTest{Operator: test.operator, Value: decimal.MustParse(test.threshold)}}
			root := &behavioral.Rule{Sum: &sum}
			if test.average {
				root = &behavioral.Rule{Average: &behavioral.Average{Sum: sum}}
			}
			result, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{Limit: 100})
			if err != nil {
				t.Fatalf("expected complete independent guard evaluation, got %v", err)
			}
			assertBehavioralTruths(t, result, test.want)
			for p, fixture := range fixtures {
				t.Run(fixture.name, func(t *testing.T) {

					explained, err := store.ExplainProfileFilter(t.Context(), root, behavioral.Bindings{}, at,
						behavioralTestID(p+1))
					if err != nil {
						t.Fatalf("expected monetary evidence, got %v", err)
					}
					if explained == nil || len(explained.Nodes) != 1 ||
						explained.Nodes[0].Truth != test.want[p] || !explained.At.Equal(at) {
						t.Fatalf("expected exact explanation truth %v at %s, got %+v", test.want[p], at, explained)
					}
					node := explained.Nodes[0]
					if test.average {
						values := node.Average
						if values == nil || node.Sum != nil || node.Count != nil ||
							values.Certain != fixture.certain || values.Possible != fixture.possible {
							t.Fatalf("expected Average population [%d,%d], got %+v",
								fixture.certain, fixture.possible, node)
						}
						determined := fixture.certain == fixture.possible
						knownSum := determined && fixture.lower != ""
						if (values.Count != nil) != determined || (values.Sum != nil) != knownSum {
							t.Fatalf("expected determined count=%t and sum=%t, got %+v", determined, knownSum, values)
						}
						if values.Count != nil && *values.Count != fixture.certain ||
							values.Sum != nil && !values.Sum.Equal(decimal.MustParse(fixture.lower)) {
							t.Fatalf("expected exact Average inputs count=%d sum=%s, got %+v",
								fixture.certain, fixture.lower, values)
						}
						return
					}
					bounds := node.Sum
					if bounds == nil || node.Average != nil || node.Count != nil ||
						bounds.Certain != fixture.certain || bounds.Possible != fixture.possible ||
						(bounds.Lower == nil) != (fixture.lower == "") ||
						(bounds.Upper == nil) != (fixture.upper == "") {
						t.Fatalf("expected Sum count [%d,%d] and bounds [%s,%s], got %+v",
							fixture.certain, fixture.possible, fixture.lower, fixture.upper, node)
					}
					if bounds.Lower != nil && !bounds.Lower.Equal(decimal.MustParse(fixture.lower)) ||
						bounds.Upper != nil && !bounds.Upper.Equal(decimal.MustParse(fixture.upper)) {
						t.Fatalf("expected exact Sum bounds [%s,%s], got %+v", fixture.lower, fixture.upper, bounds)
					}

				})
			}

		})
	}

}
