// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestBehavioralRecordEnumPostgreSQL checks substring predicates on managed
// enums.
func TestBehavioralRecordEnumPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("requires disposable PostgreSQL")
	}
	testBehavioralRecordEnum(t, "PostgreSQL")
}

// TestBehavioralRecordEnumSnowflake checks the corresponding managed text
// columns.
func TestBehavioralRecordEnumSnowflake(t *testing.T) {
	if testing.Short() {
		t.Skip("requires explicitly configured Snowflake")
	}
	dw := warehouses.Registered("Snowflake").New(newTestSettingsLoader(json.Value(`{}`)), nil)
	verified := dw.(warehouses.ProfileFilterWarehouse).ProfileFilterVerified()
	err := dw.Close()
	if err != nil {
		t.Fatalf("expected unopened warehouse cleanup, got %v", err)
	}
	if !verified {
		t.Fatal("expected verified Snowflake provider, got closed gate")
	}
	testBehavioralRecordEnum(t, "Snowflake")
}

func testBehavioralRecordEnum(t *testing.T, platform string) {

	t.Helper()
	fixture := newBehavioralRecordFixture(t, platform, nil)
	profiles := warehouses.Table{Name: "krenalis_profiles_0", Keys: []string{"_kpid"}, Columns: []warehouses.Column{
		{Name: "_kpid", Type: types.UUID()}, {Name: "_anonymous_count", Type: types.Int(32)},
		{Name: "_recognized_count", Type: types.Int(32)}, {Name: "_updated_at", Type: types.DateTime()},
	}}
	err := fixture.warehouse.Merge(t.Context(), profiles, [][]any{
		{behavioralTestID(1), 0, 1, fixture.at},
		{behavioralTestID(2), 0, 1, fixture.at},
		{behavioralTestID(3), 0, 1, fixture.at},
	}, nil)
	if err != nil {
		t.Fatalf("expected profile writes, got %v", err)
	}
	events := warehouses.Table{Name: "krenalis_events", Keys: []string{"message_id"}}
	for _, property := range []string{"kpid", "messageId", "connectionId", "anonymousId", "type", "event",
		"timestamp", "receivedAt", "sentAt", "traits", "properties", "context.browser.name", "context.os.name"} {
		events.Columns = append(events.Columns, eventColumnByProperty[property])
	}
	var rows [][]any
	for i, values := range [][2]any{{"Chrome", "Linux"}, {"Safari", "Windows"}, {nil, nil}} {
		rows = append(rows,
			[]any{behavioralTestID(i + 1), behavioralTestID(i + 1), "fixture", "fixture", "track", "Purchase",
				fixture.at.Add(-time.Hour), fixture.at, fixture.at,
				json.Value(`{}`), json.Value(`{}`), values[0], values[1]})
	}
	err = fixture.warehouse.Merge(t.Context(), events, rows, nil)
	if err != nil {
		t.Fatalf("expected managed event writes, got %v", err)
	}

	m, f, u := behavioral.Match, behavioral.NoMatch, behavioral.Unknown
	for _, test := range []struct {
		property []string
		value    string
		want     []behavioral.Truth
	}{
		{[]string{"type"}, "track", []behavioral.Truth{m, m, m}},
		{[]string{"type"}, "identify", []behavioral.Truth{f, f, f}},
		{[]string{"context", "browser", "name"}, "Chrome", []behavioral.Truth{m, f, u}},
		{[]string{"context", "os", "name"}, "Linux", []behavioral.Truth{m, f, u}},
	} {
		for _, op := range []state.WhereOperator{state.OpContains, state.OpDoesNotContain} {
			t.Run(strings.Join(test.property, ".")+"/"+test.value+"/"+op.String(), func(t *testing.T) {

				want := slices.Clone(test.want)
				if op == state.OpDoesNotContain {
					for i, truth := range want {
						if truth == m {
							want[i] = f
						} else if truth == f {
							want[i] = m
						}
					}
				}
				condition := &state.WhereCondition{Property: test.property, Operator: op, Values: []any{test.value}}
				root := &behavioral.Rule{Count: &behavioral.Count{
					Scope: behavioral.Scope{Event: "Purchase", Grain: "registered-message",
						Start: fixture.at.Add(-24 * time.Hour), End: fixture.at},
					Where: &behavioral.OccurrencePredicate{Record: condition},
					Test:  behavioral.CountTest{Operator: behavioral.GreaterOrEqual, Value: 1},
				}}
				assertBehavioralTruths(t, fixture.query(t, root, behavioral.Page{Limit: 10}), want)
				var expectedIDs []string
				for i, truth := range want {
					bounds := warehouses.OccurrenceCountBounds{}
					if truth == m {
						bounds.Certain = 1
						expectedIDs = append(expectedIDs, behavioralTestID(i+1))
					}
					if truth != f {
						bounds.Possible = 1
					}
					explained := fixture.explain(t, root, i+1)
					if len(explained.Nodes) != 1 || explained.Nodes[0].Truth != truth ||
						explained.Nodes[0].Count == nil || *explained.Nodes[0].Count != bounds {
						t.Fatalf("expected profile %d truth %v and bounds %+v, got %+v", i+1, truth, bounds, explained)
					}
				}

				legacy, err := fixture.store.Events(t.Context(), Query{Properties: []string{"kpid"},
					Where:   &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{condition}},
					OrderBy: "kpid", Limit: 10})
				if err != nil {
					t.Fatalf("expected legacy event query, got %v", err)
				}
				if len(legacy) != len(expectedIDs) {
					t.Fatalf("expected legacy profile IDs %v, got %v", expectedIDs, legacy)
				}
				for i, event := range legacy {
					if event["kpid"] != expectedIDs[i] {
						t.Fatalf("expected legacy profile ID %s, got %v", expectedIDs[i], event)
					}
				}

			})
		}
	}

}
