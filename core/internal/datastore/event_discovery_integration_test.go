// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/snowflakedb/gosnowflake/v2"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestEventDiscoveryPostgreSQL exercises bounded discovery on disposable
// PostgreSQL.
func TestEventDiscoveryPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("requires disposable PostgreSQL")
	}
	testEventDiscoveryWarehouse(t, "PostgreSQL")
}

// TestEventDiscoverySnowflake repeats the fixtures on a live test database.
func TestEventDiscoverySnowflake(t *testing.T) {
	if testing.Short() {
		t.Skip("requires explicitly configured Snowflake test environment")
	}
	testEventDiscoveryWarehouse(t, "Snowflake")
}

func testEventDiscoveryWarehouse(t *testing.T, platform string) {

	t.Helper()
	warehouse := newJSONFilterTestWarehouse(t, platform)
	tag := "krenalis-discovery/" + rand.Text()
	observer := newBehavioralInterruptionObserver(t, warehouse, platform, tag, "cancellation")
	q := func(name string) string { return `"` + name + `"` }
	jsonType, timestampType, textType := "jsonb", "timestamp(6)", "VARCHAR"
	if platform == "Snowflake" {
		q = func(name string) string { return `"` + strings.ToUpper(name) + `"` }
		jsonType, timestampType, textType = "VARIANT", "TIMESTAMP_NTZ(6)", "VARCHAR COLLATE 'en-ci'"
	}
	execBehavioralFixtureStatement(t, warehouse, fmt.Sprintf(`CREATE TABLE %s (
%s %s PRIMARY KEY,%s %s,%s %s,%s %s,%s %s,%s %s,%s %s)`,
		q("krenalis_events"), q("message_id"), "VARCHAR", q("timestamp"), timestampType,
		q("type"), textType, q("event"), textType, q("name"), textType,
		q("traits"), jsonType, q("properties"), jsonType))
	table := warehouses.Table{Name: "krenalis_events", Keys: []string{"message_id"}, Columns: []warehouses.Column{
		{Name: "message_id", Type: types.String()}, {Name: "timestamp", Type: types.DateTime()},
		{Name: "type", Type: types.String()}, {Name: "event", Type: types.String(), Nullable: true},
		{Name: "name", Type: types.String(), Nullable: true}, {Name: "traits", Type: types.JSON(), Nullable: true},
		{Name: "properties", Type: types.JSON(), Nullable: true},
	}}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	window := warehouses.EventDiscoveryWindow{Start: start, End: start.Add(24 * time.Hour)}
	var rows [][]any
	add := func(typ string, name any, at time.Time, traits, properties any) {
		var event, pageName any
		if typ == "track" {
			event = name
		} else {
			pageName = name
		}
		rows = append(rows, []any{fmt.Sprintf("e%04d", len(rows)), at, typ, event, pageName, traits, properties})
	}
	for _, source := range []struct{ name, properties string }{
		{"Purchase", `{"upper":true}`},
		{"purchase", `{"lower":true}`},
		{"é", `{"composed":true}`},
		{"e\u0301", `{"decomposed":true}`},
		{"x%_\\y", `{}`},
		{strings.Repeat("L", 1025), `{}`},
	} {
		add("track", source.name, start, nil, json.Value(source.properties))
	}
	for _, name := range []any{nil, "", "Pricing"} {
		properties := json.Value(`{}`)
		if name == nil {
			properties = json.Value(`{"unnamed":true}`)
		} else if name == "" {
			properties = json.Value(`{"empty":true}`)
		}
		add("page", name, start, nil, properties)
	}
	add("screen", "Home", start, nil, json.Value(`{}`))
	add("identify", "Ignored", start, nil, nil)
	add("TRACK", "IgnoredCaseType", start, nil, nil)
	add("track", "Before", start.Add(-time.Microsecond), nil, nil)
	add("track", "AtEnd", window.End, nil, nil)
	add("track", "LastMicrosecond", window.End.Add(-time.Microsecond), nil, nil)
	for range 220 {
		add("track", "Frequent", start.Add(time.Hour), nil, json.Value(`{}`))
	}
	add("track", "Rare", start, nil, json.Value(`{"nested":{"x":1},"arr":[{"hidden":1}]}`))
	for _, raw := range []string{"", "null", "{}", "1", `"x"`, "true", "[]", `{"a":{"b":1},"literal.dot":false}`} {
		var value any
		if raw != "" {
			value = json.Value(raw)
		}
		add("track", "Root Matrix", start, value, value)
	}
	readable := json.Value(`"` + strings.Repeat("é", 32767) + `"`)
	oversized := json.Value(`"` + strings.Repeat("é", 32767) + `x"`)
	for i := range 65 {
		add("track", "Budget", start.Add(time.Duration(i)*time.Second), readable, readable)
	}
	add("track", "Budget", start.Add(time.Hour), oversized, oversized)
	err := warehouse.Merge(t.Context(), table, rows, nil)
	if err != nil {
		t.Fatalf("expected event discovery fixtures, got %v", err)
	}
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(warehouse))
	list := func(t *testing.T, typ, contains string, limit int) EventSourceDiscoveryResult {
		t.Helper()
		page, err := store.ObservedEventSources(t.Context(), warehouses.EventSourceDiscoveryRequest{
			Window: window, Type: typ, NameContains: contains, Limit: limit,
		})
		if err != nil {
			t.Fatalf("expected source page, got %v", err)
		}
		return page
	}
	inspect := func(t *testing.T, typ string, name *string, limit int) EventFieldDiscoveryResult {
		t.Helper()
		result, err := store.ObservedEventFields(t.Context(), warehouses.EventFieldSampleRequest{
			Window: window, Source: warehouses.ObservedEventSource{Type: typ, RecordedName: name}, Limit: limit,
		})
		if err != nil {
			t.Fatalf("expected observed fields, got %v", err)
		}
		return result
	}

	t.Run("literal sources and time borders", func(t *testing.T) {
		page := list(t, "", "", 500)
		seen := map[string]bool{}
		for _, source := range page.Sources {
			key := source.Type + "/NULL"
			if source.RecordedName != nil {
				key = source.Type + "/" + *source.RecordedName
			}
			seen[key] = true
		}
		for _, key := range []string{"track/Purchase", "track/purchase", "track/é", "track/e\u0301",
			"page/NULL", "page/", "page/Pricing", "screen/Home", "track/LastMicrosecond", "track/Rare"} {
			if !seen[key] {
				t.Fatalf("expected literal source %q, got %+v", key, page.Sources)
			}
		}
		if seen["track/Before"] || seen["track/AtEnd"] || seen["identify/NULL"] ||
			page.OmittedLongNames != 1 || !page.Truncated {
			t.Fatalf("expected half-open window and visible long-name omission, got %+v", page)
		}
		if frequent := list(t, "track", "Frequent", 1); len(frequent.Sources) != 1 || frequent.Truncated {
			t.Fatalf("expected one distinct frequent source, got %+v", frequent)
		}
		page = list(t, "track", "%_\\", 500)
		if page.Truncated || len(page.Sources) != 1 || *page.Sources[0].RecordedName != "x%_\\y" {
			t.Fatalf("expected literal wildcard characters, got %+v", page)
		}
		page = list(t, "track", "Purchase", 500)
		if len(page.Sources) != 1 || *page.Sources[0].RecordedName != "Purchase" {
			t.Fatalf("expected case-sensitive name search, got %+v", page)
		}
		for _, source := range []struct{ name, property string }{
			{"Purchase", "properties.upper"},
			{"purchase", "properties.lower"},
			{"é", "properties.composed"},
			{"e\u0301", "properties.decomposed"},
		} {
			result := inspect(t, "track", &source.name, 200)
			if result.Sample.Rows != 1 || len(result.Fields) != 1 || result.Fields[0].Property != source.property {
				t.Fatalf("expected one literal source %q with %s, got %+v", source.name, source.property, result)
			}
		}
		if page := list(t, "page", "", 2); len(page.Sources) != 2 || !page.Truncated {
			t.Fatalf("expected two-source prefix and more sources, got %+v", page)
		}
	})

	t.Run("exact unnamed and empty page", func(t *testing.T) {
		for _, name := range []*string{nil, new("")} {
			result := inspect(t, "page", name, 1000)
			property := "properties.unnamed"
			if name != nil {
				property = "properties.empty"
			}
			if result.Sample.Rows != 1 || len(result.Fields) != 1 || result.Fields[0].Property != property {
				t.Fatalf("expected one exact unnamed/empty page with %s, got %+v", property, result)
			}
		}
	})

	t.Run("rare source and arrays", func(t *testing.T) {
		result := inspect(t, "track", new("Rare"), 200)
		if result.Sample.Rows != 1 || len(result.Fields) != 3 || result.Sample.FieldsTruncated {
			t.Fatalf("expected rare source independent of frequent rows, got %+v", result)
		}
		if result := inspect(t, "track", new("Does Not Exist"), 200); result.Sample.Rows != 0 {
			t.Fatalf("expected empty sample, got %+v", result.Sample)
		}
	})

	t.Run("SQL null JSON null and all kinds", func(t *testing.T) {
		result := inspect(t, "track", new("Root Matrix"), 1000)
		for _, name := range []string{"traits", "properties"} {
			root := result.Roots[name]
			if root.Present != 7 || root.Uninspected != 0 || root.Kinds["object"] != 2 || len(root.Kinds) != 6 {
				t.Fatalf("expected separate null/absence and six JSON kinds, got %+v", root)
			}
		}
	})

	t.Run("individual and cumulative byte limits", func(t *testing.T) {
		result := inspect(t, "track", new("Budget"), 1000)
		if result.Sample.Rows != 65 || !result.Sample.HasMoreRows || result.Sample.FieldsTruncated {
			t.Fatalf("expected one unread row plus 64 readable rows, got %+v", result.Sample)
		}
		for _, root := range result.Roots {
			if root.Present != 65 || root.Uninspected != 1 || root.Kinds["string"] != 64 {
				t.Fatalf("expected exact per-root byte boundary, got %+v", root)
			}
		}
		result = inspect(t, "track", new("Budget"), 1)
		if result.Sample.Rows != 1 || !result.Sample.HasMoreRows || result.Roots["properties"].Uninspected != 1 {
			t.Fatalf("expected oversized first row retained and control JSON absent, got %+v", result)
		}
	})

	t.Run("cancel active distinct and recover", func(t *testing.T) {

		switch platform {
		case "PostgreSQL":
			execBehavioralFixtureStatement(t, warehouse, `CREATE FUNCTION event_discovery_delay(id text, seconds double precision)
RETURNS boolean LANGUAGE plpgsql VOLATILE AS $$ BEGIN PERFORM pg_sleep(seconds); RETURN true; END $$`)
		case "Snowflake":
			execBehavioralFixtureStatement(t, warehouse, `CREATE FUNCTION event_discovery_delay(id VARCHAR, seconds FLOAT)
RETURNS BOOLEAN LANGUAGE JAVASCRIPT VOLATILE AS $$
var until = Date.now() + SECONDS * 1000; while (Date.now() < until) {} return true; $$`)
		default:
			t.Fatalf("expected supported warehouse, got %s", platform)
		}
		execBehavioralFixtureStatement(t, warehouse, `SELECT event_discovery_delay('warm',0.001)`)
		execBehavioralFixtureStatement(t, warehouse, `ALTER TABLE krenalis_events RENAME TO event_discovery_base`)
		execBehavioralFixtureStatement(t, warehouse, `CREATE VIEW krenalis_events AS
SELECT * FROM event_discovery_base WHERE event_discovery_delay(message_id,30)`)
		request := warehouses.EventSourceDiscoveryRequest{Window: window, Type: "track", NameContains: "Frequent", Limit: 1}
		ctx, cancel := context.WithCancel(gosnowflake.WithQueryTag(t.Context(), tag))
		defer cancel()
		type outcome struct {
			page EventSourceDiscoveryResult
			err  error
		}
		done := make(chan outcome, 1)
		finished := make(chan struct{})
		t.Cleanup(func() {
			cancel()
			select {
			case <-finished:
			case <-time.After(60 * time.Second):
				t.Error("expected discovery worker cleanup, got timeout")
			}
		})
		started := time.Now()
		go func() {
			defer close(finished)
			page, err := store.ObservedEventSources(ctx, request)
			done <- outcome{page, err}
		}()
		observer.wait(t, true)
		select {
		case <-done:
			t.Fatal("expected DISTINCT still running before cancellation, got completed operation")
		default:
			// Cancel only after the observer sees active execution.
		}
		cancel()
		var result outcome
		select {
		case result = <-done:
		case <-time.After(60 * time.Second):
			t.Fatal("expected completed cancelled discovery, got client timeout")
		}
		if result.err != nil {

			failure, ok := errors.AsType[*WarehouseError](result.err)
			if !ok || !errors.Is(failure.Err, context.Canceled) || result.page.Sources != nil {
				t.Fatalf("expected cancellation without partial sources, got %v, %+v", result.err, result.page)
			}
			terminal := observer.wait(t, false)
			if platform == "PostgreSQL" && time.Since(started) >= 20*time.Second ||
				platform == "Snowflake" && terminal.code != 604 {
				t.Fatalf("expected server cancellation before natural completion, got code=%d after %v",
					terminal.code, time.Since(started))
			}
			t.Logf("discovery cancellation server_code=%d total_elapsed_ms=%d", terminal.code,
				time.Since(started).Milliseconds())

			execBehavioralFixtureStatement(t, warehouse, `CREATE OR REPLACE VIEW krenalis_events AS
SELECT * FROM event_discovery_base`)
			page, err := store.ObservedEventSources(t.Context(), request)
			if err != nil {
				t.Fatalf("expected healthy follow-up on the same warehouse, got %v", err)
			}
			if len(page.Sources) != 1 || page.Truncated || page.Sources[0].RecordedName == nil ||
				*page.Sources[0].RecordedName != "Frequent" {
				t.Fatalf("expected one complete follow-up source, got %+v", page)
			}

			return
		}
		t.Fatal("expected cancelled DISTINCT failure, got success")

	})

}
