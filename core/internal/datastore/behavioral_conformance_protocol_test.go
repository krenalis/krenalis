// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// testBehavioralConformanceProtocol checks projection, ties in page ordering
// and mixed explanation layouts, then proves pure Record needs no event table.
func testBehavioralConformanceProtocol(t *testing.T, store *Store, dw warehouses.Warehouse, table warehouses.Table, name string) {

	t.Helper()
	execBehavioralFixture(t, dw, "TRUNCATE TABLE profiles; TRUNCATE TABLE krenalis_events")
	switch name {
	case "PostgreSQL":
		execBehavioralFixture(t, dw, "ALTER TABLE profiles ADD COLUMN country text, "+
			"ADD COLUMN label text, ADD COLUMN priority bigint")
	case "Snowflake":
		execBehavioralFixture(t, dw, "ALTER TABLE profiles ADD COLUMN country VARCHAR, "+
			"label VARCHAR, priority BIGINT")
	default:
		t.Fatalf("expected supported protocol warehouse, got %q", name)
	}
	schema := types.Object([]types.Property{
		{Name: "country", Type: types.String(), ReadOptional: true},
		{Name: "label", Type: types.String()},
		{Name: "priority", Type: types.Int(64)},
	})
	store.columnByProperty.user = profileColumnByProperty(schema)
	profiles := warehouses.Table{Name: "profiles", Keys: []string{"_kpid"}, Columns: []warehouses.Column{
		{Name: "_kpid", Type: types.UUID()}, {Name: "country", Type: types.String(), Nullable: true},
		{Name: "label", Type: types.String()}, {Name: "priority", Type: types.Int(64)},
	}}
	at := time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.UTC)
	var profileRows, events [][]any
	for p := range 9 {
		var country any
		if p < 6 {
			country = []string{"IT", "FR"}[p/3]
		}
		profileRows = append(profileRows, []any{behavioralTestID(p + 1), country, fmt.Sprintf("client-%d", p+1), 1})
		if p%3 != 0 {
			payload := `{"total":100}`
			if p%3 == 1 {
				payload = `{"total":100,"currency":"EUR"}`
			}
			events = append(events, []any{behavioralTestID(p + 1), fmt.Sprintf("protocol/%d", p), "track",
				"Order Completed", nil, at.Add(-time.Hour), at, at, json.Value(payload)})
		}
	}
	err := dw.Merge(t.Context(), profiles, profileRows, nil)
	if err != nil {
		t.Fatalf("expected profile projection fixtures, got %v", err)
	}
	err = dw.Merge(t.Context(), table, events, nil)
	if err != nil {
		t.Fatalf("expected protocol event fixtures, got %v", err)
	}
	field, err := behavioral.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected protocol amount field, got %v", err)
	}
	currency, err := behavioral.ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR",
		"Order Completed")
	if err != nil {
		t.Fatalf("expected protocol currency field, got %v", err)
	}
	scope := behavioral.Scope{Event: "Order Completed", Grain: "registered-message",
		Start: at.Add(-24 * time.Hour), End: at}
	record := behavioral.Rule{Record: &state.WhereCondition{Property: []string{"country"}, Operator: state.OpIs,
		Values: []any{"IT"}}}
	sum := behavioral.Sum{Scope: scope, Field: field, NativeCurrency: "EUR",
		Test: behavioral.SumTest{Operator: behavioral.GreaterOrEqual, Value: decimal.MustParse("30")}}
	average := sum
	average.Minimum = 1
	money := &behavioral.OccurrencePredicate{Money: &behavioral.Money{Field: field, NativeCurrency: "EUR", Test: sum.Test}}
	children := []behavioral.Rule{record,
		{Count: &behavioral.Count{Scope: scope, Currency: currency, Test: behavioral.CountTest{Operator: behavioral.Equal}}},
		{Sum: &sum}, {Average: &behavioral.Average{Sum: average}},
		{Experience: &behavioral.Experience{Scope: scope, Currency: currency, Verify: money}},
		{LatestN: &behavioral.LatestN{Scope: scope, Currency: currency, Size: 1, Predicate: money}},
		{LastVersusPrevious: &behavioral.LastVersusPrevious{Scope: scope, Currency: currency, Field: field,
			Check: behavioral.LastPreviousCheck{Kind: behavioral.LastPreviousDifference,
				Test: behavioral.SumTest{Operator: behavioral.GreaterOrEqual}}}},
	}
	const m, f, u = behavioral.Match, behavioral.NoMatch, behavioral.Unknown
	leaves := [][]behavioral.Truth{
		{m, m, m, f, f, f, u, u, u}, {m, f, u, m, f, u, m, f, u}, {f, m, u, f, m, u, f, m, u},
		{f, m, u, f, m, u, f, m, u}, {f, m, u, f, m, u, f, m, u}, {f, m, u, f, m, u, f, m, u},
		{f, f, f, f, f, f, f, f, f},
	}
	evidence := [][5]bool{{}, {true, false, false, false, false}, {false, true, false, false, false},
		{false, false, true, false, false}, {false, false, false, true, false}, {false, false, false, true, false},
		{false, false, false, false, true}}
	for length := 2; length <= len(children); length++ {
		t.Run(fmt.Sprintf("mixed-%d", length), func(t *testing.T) {

			root := &behavioral.Rule{Operator: state.OpOr, Children: children[:length]}
			want := make([]behavioral.Truth, 9)
			for p := range want {
				want[p] = f
				for _, truth := range leaves[:length] {
					if truth[p] == m {
						want[p] = m
						break
					}
					if truth[p] == u {
						want[p] = u
					}
				}
			}
			result, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{Limit: 20})
			if err != nil {
				t.Fatalf("expected mixed protocol evaluation, got %v", err)
			}
			assertBehavioralTruths(t, result, want)
			for p := range want {
				explained, err := store.ExplainProfileFilter(t.Context(), root, behavioral.Bindings{}, at,
					behavioralTestID(p+1))
				if err != nil {
					t.Fatalf("expected mixed protocol explanation, got %v", err)
				}
				if explained == nil || len(explained.Nodes) != length+1 ||
					explained.Nodes[0].Truth != want[p] || !explained.At.Equal(at) {
					t.Fatalf("expected %d nodes with root %v at %s, got %+v", length+1, want[p], at, explained)
				}
				for i, node := range explained.Nodes {
					expectedTruth := want[p]
					expectedFields := [5]bool{}
					if i > 0 {
						expectedTruth = leaves[i-1][p]
						expectedFields = evidence[i-1]
					}
					actualFields := [5]bool{node.Count != nil, node.Sum != nil, node.Average != nil,
						node.Experience != nil, node.Pair != nil}
					if node.Truth != expectedTruth || actualFields != expectedFields {
						t.Fatalf("expected node %d truth %v and fields %v, got %+v",
							i, expectedTruth, expectedFields, node)
					}
				}
			}

			// All sort values tie. The profile identity must order every page
			// deterministically, in both requested directions of the sort key.
			var ids []string
			for p, truth := range want {
				if truth == m {
					ids = append(ids, behavioralTestID(p+1))
				}
			}
			for _, desc := range []bool{false, true} {
				for first := 0; first <= len(ids)+2; first += 2 {
					page := behavioral.Page{Properties: []string{"country", "label", "priority"},
						OrderBy: "priority", OrderDesc: desc, First: first, Limit: 2, MatchingOnly: true}
					actual, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, page)
					if err != nil {
						t.Fatalf("expected ordered projection page, got %v", err)
					}
					lo := min(first, len(ids))
					expected := ids[lo:min(lo+2, len(ids))]
					if actual.Match != result.Match || actual.NoMatch != result.NoMatch ||
						actual.Unknown != result.Unknown || !actual.At.Equal(at) ||
						len(actual.Profiles) != len(expected) {
						t.Fatalf("expected page %v with unchanged totals and T, got %+v", expected, actual)
					}
					for i, profile := range actual.Profiles {
						if profile.ID != expected[i] || profile.Truth != behavioral.Match {
							t.Fatalf("expected matching profile %s, got %+v", expected[i], profile)
						}
						p := 0
						for behavioralTestID(p+1) != profile.ID {
							p++
						}
						attributes := map[string]any{"label": fmt.Sprintf("client-%d", p+1), "priority": 1}
						if p < 6 {
							attributes["country"] = []string{"IT", "FR"}[p/3]
						}
						if !reflect.DeepEqual(profile.Attributes, attributes) {
							t.Fatalf("expected projected attributes %v, got %v", attributes, profile.Attributes)
						}
					}
				}
			}

		})
	}

	t.Run("Record without events", func(t *testing.T) {

		execBehavioralFixture(t, dw, "DROP TABLE krenalis_events")
		result, err := store.QueryProfiles(t.Context(), &record, behavioral.Bindings{}, at, behavioral.Page{Limit: 20})
		if err != nil {
			t.Fatalf("expected pure Record evaluation without events, got %v", err)
		}
		assertBehavioralTruths(t, result, leaves[0])
		explained, err := store.ExplainProfileFilter(t.Context(), &record, behavioral.Bindings{}, at,
			behavioralTestID(9))
		if err != nil {
			t.Fatalf("expected pure Record explanation without events, got %v", err)
		}
		if explained == nil || len(explained.Nodes) != 1 || explained.Nodes[0].Truth != u ||
			explained.Nodes[0].Count != nil || explained.Nodes[0].Sum != nil {
			t.Fatalf("expected pure Record UNKNOWN without occurrence metadata, got %+v", explained)
		}
		missing, err := store.ExplainProfileFilter(t.Context(), &record, behavioral.Bindings{}, at,
			behavioralTestID(99))
		if err != nil {
			t.Fatalf("expected missing-profile lookup, got %v", err)
		}
		if missing != nil {
			t.Fatalf("expected no explanation for missing profile, got %+v", missing)
		}
		execBehavioralFixture(t, dw, "TRUNCATE TABLE profiles")
		empty, err := store.QueryProfiles(t.Context(), &record, behavioral.Bindings{}, at, behavioral.Page{Limit: 20})
		if err != nil {
			t.Fatalf("expected empty profile population, got %v", err)
		}
		assertBehavioralTruths(t, empty, []behavioral.Truth{})

	})

}
