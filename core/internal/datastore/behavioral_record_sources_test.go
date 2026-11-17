// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestBehavioralRecordSourcesPostgreSQL checks literal sources under ICU
// collation.
func TestBehavioralRecordSourcesPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("requires disposable PostgreSQL")
	}
	testBehavioralRecordSources(t, "PostgreSQL")
}

// TestBehavioralRecordSourcesSnowflake checks inherited schema collation.
func TestBehavioralRecordSourcesSnowflake(t *testing.T) {
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
	testBehavioralRecordSources(t, "Snowflake")
}

func quoteBehavioralRecordIdentifier(platform, name string) string {
	if platform == "Snowflake" {
		name = strings.ToUpper(name)
	}
	return `"` + name + `"`
}

func testBehavioralRecordSources(t *testing.T, platform string) {

	t.Helper()
	dw := newJSONFilterTestWarehouse(t, platform)
	if platform == "Snowflake" {
		rows, _, err := dw.(interface {
			RawQuery(context.Context, string) (warehouses.Rows, int, error)
		}).RawQuery(t.Context(), "SELECT CURRENT_SCHEMA()")
		if err != nil {
			t.Fatalf("expected isolated schema lookup, got %v", err)
		}
		var schema string
		if !rows.Next() {
			_ = rows.Close()
			t.Fatal("expected isolated schema row, got none")
		}
		err = rows.Scan(&schema)
		if err != nil {
			_ = rows.Close()
			t.Fatalf("expected isolated schema name, got %v", err)
		}
		extra := rows.Next()
		err = rows.Err()
		closeErr := rows.Close()
		if err != nil {
			t.Fatalf("expected complete schema lookup, got %v", err)
		}
		if closeErr != nil {
			t.Fatalf("expected schema lookup close, got %v", closeErr)
		}
		if extra || !warehouses.IsValidIdentifier(schema) || !strings.HasPrefix(schema, "KRENALIS_TEST_SCHEMA_") {
			t.Fatal("expected one generated test schema, got invalid lookup")
		}
		execBehavioralFixture(t, dw, `ALTER SCHEMA "`+schema+`" SET DEFAULT_DDL_COLLATION='en-ci'`)
	}
	err := dw.Initialize(t.Context(), nil)
	if err != nil {
		t.Fatalf("expected real warehouse initialization, got %v", err)
	}
	if platform == "PostgreSQL" {
		execBehavioralFixture(t, dw, `CREATE COLLATION krenalis_ci
 (provider=icu,locale='und-u-ks-level2',deterministic=false);
DROP VIEW events;
ALTER TABLE krenalis_events ALTER COLUMN event TYPE varchar COLLATE krenalis_ci;
ALTER TABLE krenalis_events ALTER COLUMN name TYPE varchar COLLATE krenalis_ci;
CREATE VIEW events AS SELECT * FROM krenalis_events`)
	}
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	fixture := &behavioralRecordFixture{store: store, warehouse: dw,
		at: time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.UTC)}
	execBehavioralFixture(t, dw, "INSERT INTO "+quoteBehavioralRecordIdentifier(platform, "krenalis_profiles_0")+
		" ("+quoteBehavioralRecordIdentifier(platform, "_kpid")+","+
		quoteBehavioralRecordIdentifier(platform, "_anonymous_count")+","+
		quoteBehavioralRecordIdentifier(platform, "_recognized_count")+","+
		quoteBehavioralRecordIdentifier(platform, "_updated_at")+") VALUES ('"+behavioralTestID(1)+
		"',0,1,'2026-02-01'),('"+behavioralTestID(2)+"',0,1,'2026-02-01')")
	table := warehouses.Table{Name: "krenalis_events", Keys: []string{"message_id"}, Columns: []warehouses.Column{
		{Name: "kpid", Type: types.UUID()}, {Name: "message_id", Type: types.String()},
		{Name: "connection_id", Type: types.String()}, {Name: "anonymous_id", Type: types.String()},
		{Name: "type", Type: types.String()}, {Name: "event", Type: types.String(), Nullable: true},
		{Name: "name", Type: types.String(), Nullable: true}, {Name: "timestamp", Type: types.DateTime()},
		{Name: "received_at", Type: types.DateTime()}, {Name: "sent_at", Type: types.DateTime()},
		{Name: "traits", Type: types.JSON()}, {Name: "properties", Type: types.JSON(), Nullable: true},
	}}
	var rows [][]any
	for i, event := range []struct {
		id, typ string
		event   any
		name    any
		amount  string
		profile int
	}{
		{"purchase-exact", "track", "Purchase", nil, "0", 1},
		{"purchase-case", "track", "purchase", nil, "0", 1},
		{"order-variant-first", "track", "order completed", nil, "1000", 1},
		{"order-exact-first", "track", "Order Completed", nil, "50", 1},
		{"order-exact-last", "track", "Order Completed", nil, "100", 1},
		{"order-variant-last", "track", "order completed", nil, "500", 1},
		{"order-type-case", "TRACK", "Order Completed", nil, "900", 1},
		{"page-exact", "page", nil, "Pricing", "0", 1},
		{"page-case", "page", nil, "pricing", "0", 1},
		{"page-null", "page", nil, nil, "0", 1},
		{"page-empty", "page", nil, "", "0", 1},
		{"screen-exact", "screen", nil, "Home", "0", 1},
		{"screen-case", "screen", nil, "home", "0", 1},
		{"screen-null", "screen", nil, nil, "0", 1},
		{"screen-empty", "screen", nil, "", "0", 1},
		{"unicode-composed", "track", "é", nil, "0", 1},
		{"unicode-decomposed", "track", "e\u0301", nil, "0", 1},
		{"track-null", "track", nil, nil, "200", 2},
	} {
		if platform == "PostgreSQL" && event.typ == "TRACK" {
			// The managed PostgreSQL enum cannot store an upper-case track.
			event.typ = "identify"
		}
		rows = append(rows, []any{behavioralTestID(event.profile), event.id, "fixture", "fixture", event.typ,
			event.event, event.name, fixture.at.Add(-time.Hour).Add(time.Duration(i) * time.Second), fixture.at,
			fixture.at, json.Value(`{}`), json.Value(`{"currency":"EUR","total":` + event.amount + `}`)})
	}
	err = dw.Merge(t.Context(), table, rows, nil)
	if err != nil {
		t.Fatalf("expected real source writes, got %v", err)
	}

	for _, test := range []struct {
		typ, event string
		count      uint64
		other      behavioral.Truth
	}{
		{"track", "Purchase", 1, behavioral.Unknown},
		{"track", "Order Completed", 2, behavioral.NoMatch}, // One possible message cannot reach two.
		{"track", "é", 1, behavioral.Unknown},
		{"page", "Pricing", 1, behavioral.NoMatch},
		{"screen", "Home", 1, behavioral.NoMatch},
		{"page", "", 4, behavioral.NoMatch},
		{"screen", "", 4, behavioral.NoMatch},
		{"page", "Missing", 0, behavioral.Match},
	} {
		t.Run("identity/"+test.typ+"/"+test.event, func(t *testing.T) {
			count := &behavioral.Count{Scope: behavioral.Scope{Type: test.typ, Event: test.event, Grain: "registered-message",
				Start: fixture.at.Add(-24 * time.Hour), End: fixture.at},
				Test: behavioral.CountTest{Operator: behavioral.Equal, Value: test.count}}
			root := &behavioral.Rule{Count: count}
			assertBehavioralTruths(t, fixture.query(t, root, behavioral.Page{Limit: 10}),
				[]behavioral.Truth{behavioral.Match, test.other})
			explained := fixture.explain(t, root, 1)
			if len(explained.Nodes) != 1 || explained.Nodes[0].Count == nil ||
				*explained.Nodes[0].Count != (warehouses.OccurrenceCountBounds{
					Certain: int64(test.count), Possible: int64(test.count)}) {
				t.Fatalf("expected literal count %d, got %+v", test.count, explained)
			}
			if test.typ == "page" && test.event == "Pricing" {
				count.Where = &behavioral.OccurrencePredicate{Record: &state.WhereCondition{Property: []string{"name"},
					Operator: state.OpIs, Values: []any{"pricing"}}}
				assertBehavioralTruths(t, fixture.query(t, root, behavioral.Page{Limit: 10}),
					[]behavioral.Truth{behavioral.Match, behavioral.NoMatch})
			}
		})
	}

	field, err := behavioral.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected monetary field, got %v", err)
	}
	currency, err := behavioral.ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR",
		"Order Completed")
	if err != nil {
		t.Fatalf("expected native currency, got %v", err)
	}
	scope := behavioral.Scope{Event: "Order Completed", Grain: "registered-message",
		Start: fixture.at.Add(-24 * time.Hour), End: fixture.at}
	for _, test := range []struct {
		name  string
		rule  behavioral.Rule
		other behavioral.Truth
	}{
		{"sum", behavioral.Rule{Sum: &behavioral.Sum{Scope: scope, Field: field, NativeCurrency: "EUR",
			Test: behavioral.SumTest{Operator: behavioral.LessOrEqual, Value: decimal.MustParse("150")}}}, behavioral.Unknown},
		{"average", behavioral.Rule{Average: &behavioral.Average{Sum: behavioral.Sum{Scope: scope, Field: field, NativeCurrency: "EUR",
			Minimum: 1, Test: behavioral.SumTest{Operator: behavioral.LessOrEqual, Value: decimal.MustParse("75")}}}}, behavioral.Unknown},
		{"first", behavioral.Rule{Experience: &behavioral.Experience{Scope: scope, First: true,
			Verify: &behavioral.OccurrencePredicate{Record: &state.WhereCondition{Property: []string{"messageId"},
				Operator: state.OpIs, Values: []any{"order-exact-first"}}}}}, behavioral.NoMatch},
		{"last", behavioral.Rule{Experience: &behavioral.Experience{Scope: scope,
			Verify: &behavioral.OccurrencePredicate{Record: &state.WhereCondition{Property: []string{"messageId"},
				Operator: state.OpIs, Values: []any{"order-exact-last"}}}}}, behavioral.NoMatch},
		{"nth", behavioral.Rule{Experience: &behavioral.Experience{Scope: scope, Nth: 2,
			Verify: &behavioral.OccurrencePredicate{Record: &state.WhereCondition{Property: []string{"messageId"},
				Operator: state.OpIs, Values: []any{"order-exact-last"}}}}}, behavioral.NoMatch},
		{"latestN", behavioral.Rule{LatestN: &behavioral.LatestN{Scope: scope, Size: 2,
			Predicate: &behavioral.OccurrencePredicate{Record: &state.WhereCondition{Property: []string{"messageId"},
				Operator: state.OpStartsWith, Values: []any{"order-exact-"}}}}}, behavioral.NoMatch},
		{"lastVersusPrevious", behavioral.Rule{LastVersusPrevious: &behavioral.LastVersusPrevious{Scope: scope, Field: field,
			Currency: currency, Check: behavioral.LastPreviousCheck{Kind: behavioral.LastPreviousDifference,
				Test: behavioral.SumTest{Operator: behavioral.LessOrEqual, Value: decimal.MustParse("50")}}}}, behavioral.NoMatch},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertBehavioralTruths(t, fixture.query(t, &test.rule, behavioral.Page{Limit: 10}),
				[]behavioral.Truth{behavioral.Match, test.other})
			explained := fixture.explain(t, &test.rule, 1)
			if len(explained.Nodes) != 1 || explained.Nodes[0].Truth != behavioral.Match {
				t.Fatalf("expected literal source explanation, got %+v", explained)
			}
		})
	}
	t.Logf("Source budget: query=%d explanation=%d", fixture.queries, fixture.explains)

}
