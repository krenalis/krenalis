// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

// testUnifiedConformanceSources checks each reader's relevant copy conflicts
// and the isolation of a target explanation from other profiles' faults.
func testUnifiedConformanceSources(t *testing.T, store *Store, dw warehouses.Warehouse, table warehouses.Table) {

	t.Helper()
	execUnifiedFixture(t, dw, "TRUNCATE TABLE profiles; TRUNCATE TABLE krenalis_events")
	execUnifiedFixture(t, dw, "INSERT INTO profiles VALUES ('"+unifiedTestID(1)+"'),('"+unifiedTestID(2)+"')")
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	scope := unified.Scope{Event: "Order Completed", Grain: "registered-message",
		Start: at.Add(-24 * time.Hour), End: at}
	field, err := unified.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected monetary source field, got %v", err)
	}
	currency, err := unified.ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR", scope.Event)
	if err != nil {
		t.Fatalf("expected currency source field, got %v", err)
	}
	where := &state.Where{Rules: []state.WhereRule{&state.WhereCondition{
		Property: []string{"properties", "coupon"}, Operator: state.OpIs, Values: []any{json.Value(`"PROMO"`)}}}}
	money := &unified.OccurrencePredicate{Money: &unified.Money{Field: field, NativeCurrency: "EUR",
		Test: unified.SumTest{Operator: unified.GreaterOrEqual, Value: decimal.MustParse("100")}}}
	const (
		count uint8 = 1 << iota
		sum
		average
		selected
		latest
		pair
		all = count | sum | average | selected | latest | pair
	)
	rules := []struct {
		name string
		bit  uint8
		rule unified.Rule
	}{
		{"count", count, unified.Rule{Count: &unified.Count{Scope: scope, Currency: currency,
			Where: recordCountWhere(t, where), Test: unified.CountTest{Operator: unified.GreaterOrEqual, Value: 1}}}},
		// An impossible minimum must never hide a fault in the source.
		{"sum", sum, unified.Rule{Sum: &unified.Sum{Scope: scope, Field: field, NativeCurrency: "EUR",
			Where: where, Minimum: 100, Test: unified.SumTest{Operator: unified.GreaterOrEqual}}}},
		{"average", average, unified.Rule{Average: &unified.Average{Sum: unified.Sum{Scope: scope, Field: field,
			NativeCurrency: "EUR", Where: where, Minimum: 1, Test: unified.SumTest{Operator: unified.GreaterOrEqual}}}}},
		{"last", selected, unified.Rule{Experience: &unified.Experience{Scope: scope,
			Currency: currency, Where: where, Verify: money}}},
		{"latest", latest, unified.Rule{LatestN: &unified.LatestN{Scope: scope, Size: 1,
			Currency: currency, Where: where, Predicate: money}}},
		{"pair", pair, unified.Rule{LastVersusPrevious: &unified.LastVersusPrevious{Scope: scope,
			Field: field, Currency: currency, Where: where, Check: unified.LastPreviousCheck{
				Kind: unified.LastPreviousDifference, Test: unified.SumTest{Operator: unified.GreaterOrEqual}}}}},
	}
	type event struct {
		profile   int
		id        string
		timestamp time.Time
		payload   string
	}
	row := func(profile int, id string, timestamp time.Time, payload string) event {
		return event{profile, id, timestamp, payload}
	}
	base := at.Add(-time.Hour)
	known := `{"total":150,"currency":"EUR","coupon":"PROMO"}`
	unknown := `{"total":150,"coupon":"PROMO"}`
	foreign := `{"total":150,"currency":"USD","coupon":"PROMO"}`
	const m, f, u = unified.Match, unified.NoMatch, unified.Unknown
	for _, test := range []struct {
		name         string
		rows         []event
		faults       uint8
		otherProfile bool
		countTruth   unified.Truth
	}{
		{"same copies", []event{row(1, "same", base, known), row(1, "same", base, known)}, 0, false, m},
		{"distinct identities", []event{row(1, "a", base, known), row(1, "b", base, known)}, 0, false, m},
		{"qualification true unknown", []event{row(1, "same", base, known), row(1, "same", base, unknown)}, all, false, 0},
		{"qualification true false", []event{row(1, "same", base, known), row(1, "same", base, foreign)}, all, false, 0},
		{"qualification unknown false", []event{row(1, "same", base, unknown), row(1, "same", base, foreign)}, all, false, 0},
		{"where true unknown", []event{row(1, "same", base, known), row(1, "same", base, `{"total":150,"currency":"EUR"}`)}, all, false, 0},
		{"equivalent unknown copies", []event{row(1, "same", base, unknown), row(1, "same", base, `{"total":150,"currency":null,"coupon":"PROMO"}`)}, 0, false, u},
		{"different profiles", []event{row(1, "same", base, known), row(2, "same", base, known)}, all, false, 0},
		{"assigned and unassigned", []event{row(1, "same", base, known), row(0, "same", base, known)}, all, false, 0},
		{"unassigned only", []event{row(0, "same", base, known), row(0, "same", base, unknown)}, 0, false, f},
		{"period membership", []event{row(1, "same", base, known), row(1, "same", at.Add(-48*time.Hour), known)}, all, false, 0},
		{"exclusive end", []event{row(1, "same", base, known), row(1, "same", at, known)}, all, false, 0},
		{"outside conflicts irrelevant", []event{row(1, "same", at, known), row(2, "same", at.Add(time.Hour), unknown)}, 0, false, f},
		{"inside exact timestamps", []event{row(1, "same", base, known), row(1, "same", base.Add(time.Microsecond), known)}, selected | latest | pair, false, m},
		{"different amounts same verify", []event{row(1, "same", base, known), row(1, "same", base, `{"total":160,"currency":"EUR","coupon":"PROMO"}`)}, sum | average | pair, false, m},
		{"different verify", []event{row(1, "same", base, known), row(1, "same", base, `{"total":50,"currency":"EUR","coupon":"PROMO"}`)}, sum | average | selected | latest | pair, false, m},
		{"different readability", []event{row(1, "same", base, known), row(1, "same", base, `{"total":"150","currency":"EUR","coupon":"PROMO"}`)}, sum | average | selected | latest | pair, false, m},
		{"equivalent unreadable values", []event{row(1, "same", base, `{"total":"150","currency":"EUR","coupon":"PROMO"}`), row(1, "same", base, `{"total":null,"currency":"EUR","coupon":"PROMO"}`)}, 0, false, m},
		{"excluded timestamps and readings", []event{row(1, "same", base, foreign), row(1, "same", base.Add(time.Microsecond), `{"currency":"GBP","coupon":"PROMO"}`)}, 0, false, f},
		{"false where excludes unknown", []event{row(1, "same", base, `{"coupon":"OTHER"}`), row(1, "same", base.Add(time.Microsecond), `{"total":150,"coupon":"OTHER"}`)}, 0, false, f},
		{"missing identity", []event{row(1, "", base, known)}, all, false, 0},
		{"missing timestamp", []event{row(1, "same", time.Time{}, known)}, all, false, 0},
		{"fault in other profile", []event{row(1, "target", base, known), row(2, "bad", base, known), row(2, "bad", base, foreign)}, all, true, m},
	} {
		t.Run(test.name, func(t *testing.T) {

			execUnifiedFixture(t, dw, "TRUNCATE TABLE krenalis_events")
			var rows [][]any
			for _, event := range test.rows {
				var profile, identity, timestamp any
				if event.profile != 0 {
					profile = unifiedTestID(event.profile)
				}
				if event.id != "" {
					identity = event.id
				}
				if !event.timestamp.IsZero() {
					timestamp = event.timestamp
				}
				rows = append(rows, []any{profile, identity, "track", "Order Completed", nil,
					timestamp, at, at, json.Value(event.payload)})
			}
			// The target is empty and has no identity constraint. Merge must
			// retain all physical rows, including deliberately conflicting keys.
			err := dw.Merge(t.Context(), table, rows, nil)
			if err != nil {
				t.Fatalf("expected source fixture writes, got %v", err)
			}
			counts, err := dw.Counts(t.Context(), table.Name, nil)
			if err != nil {
				t.Fatalf("expected physical source row count, got %v", err)
			}
			if len(counts) != 1 || counts[0] != len(rows) {
				t.Fatalf("expected %d physical rows without deduplication by the fixture, got %v", len(rows), counts)
			}
			applicable := rules
			if test.name == "missing timestamp" {
				// Reduction sources require known timestamps. Their behavior
				// outside that contract is not a new semantic expectation.
				// Selected readers already have an explicit runtime guard.
				applicable = rules[3:]
			}
			for _, kind := range applicable {
				t.Run(kind.name, func(t *testing.T) {

					fault := test.faults&kind.bit != 0
					result, err := store.UnifiedProfiles(t.Context(), &kind.rule, unified.Bindings{}, at,
						unified.Page{Limit: 10})
					if err != nil {
						if !fault || result != nil || !strings.Contains(err.Error(), "occurrence source violates") {
							t.Fatalf("expected source fault=%t without partial result, got %+v, %v", fault, result, err)
						}
						if _, ok := errors.AsType[*UnavailableError](err); !ok {
							t.Fatalf("expected opaque execution failure, got %T", err)
						}
					}
					if err == nil {
						if fault || result == nil || len(result.Profiles) != 2 {
							t.Fatalf("expected valid complete population with fault=%t, got %+v", fault, result)
						}
						if kind.bit == count && result.Profiles[0].Truth != test.countTruth {
							t.Fatalf("expected Count truth %v, got %v", test.countTruth, result.Profiles[0].Truth)
						}
						if test.name == "same copies" || test.name == "distinct identities" {
							expected := m
							if kind.bit == sum || kind.bit == pair && test.name == "same copies" {
								expected = f
							}
							if result.Profiles[0].Truth != expected {
								t.Fatalf("expected source fixture truth %v, got %v", expected, result.Profiles[0].Truth)
							}
						}
					}
					explained, err := store.ExplainUnifiedProfile(t.Context(), &kind.rule, unified.Bindings{}, at,
						unifiedTestID(1))
					targetFault := fault && !test.otherProfile
					if err != nil {
						if !targetFault || explained != nil ||
							!strings.Contains(err.Error(), "occurrence source violates") {
							t.Fatalf("expected target fault=%t without partial explanation, got %+v, %v",
								targetFault, explained, err)
						}
						return
					}
					if targetFault || explained == nil || len(explained.Nodes) != 1 {
						t.Fatalf("expected complete target explanation with fault=%t, got %+v", targetFault, explained)
					}
					if kind.bit == count && explained.Nodes[0].Truth != test.countTruth {
						t.Fatalf("expected Count explanation %v, got %v", test.countTruth, explained.Nodes[0].Truth)
					}
					if result != nil && explained.Nodes[0].Truth != result.Profiles[0].Truth {
						t.Fatalf("expected explanation truth %v, got %v",
							result.Profiles[0].Truth, explained.Nodes[0].Truth)
					}

				})
			}

		})
	}

}
