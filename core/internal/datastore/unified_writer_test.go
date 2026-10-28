// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/krenalis/krenalis/connectors/dummy"
	_ "github.com/krenalis/krenalis/connectors/segment"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/initdb"
	"github.com/krenalis/krenalis/core/internal/schemas"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/streams"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/core/internal/util"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/kms"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type unifiedWriterMetrics struct{}

// FinalizePassed accepts the real writers' metric callbacks.
func (unifiedWriterMetrics) FinalizePassed(string, int) {}

// TestUnifiedWritersPostgreSQL reads evidence written by the real writers and
// published by a completed Identity Resolution, without concurrent mutations.
func TestUnifiedWritersPostgreSQL(t *testing.T) {

	if testing.Short() {
		t.Skip("requires PostgreSQL")
	}

	container, err := postgres.Run(t.Context(), testimages.PostgreSQL, postgres.WithDatabase("krenalis"),
		postgres.WithUsername("krenalis"), postgres.WithPassword("krenalis"), postgres.BasicWaitStrategies())
	if err != nil {
		t.Fatalf("expected PostgreSQL container, got %v", err)
	}
	t.Cleanup(func() {
		err := testcontainers.TerminateContainer(container)
		if err != nil {
			t.Errorf("expected container cleanup, got %v", err)
		}
	})
	host, err := container.Host(t.Context())
	if err != nil {
		t.Fatalf("expected container host, got %v", err)
	}
	port, err := container.MappedPort(t.Context(), "5432/tcp")
	if err != nil {
		t.Fatalf("expected container port, got %v", err)
	}
	database, err := db.Open(&db.Options{Host: host, Port: int(port.Num()), Username: "krenalis",
		Password: "krenalis", Database: "krenalis"})
	if err != nil {
		t.Fatalf("expected application database, got %v", err)
	}
	t.Cleanup(database.Close)
	manager, err := kms.New(t.Context(), "key:"+base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatalf("expected KMS, got %v", err)
	}
	err = initdb.InitIfEmpty(t.Context(), database, manager, false)
	if err != nil {
		t.Fatalf("expected application schema, got %v", err)
	}
	_, err = database.Exec(t.Context(), "CREATE SCHEMA unified_writers")
	if err != nil {
		t.Fatalf("expected warehouse schema, got %v", err)
	}
	settings, err := json.Marshal(map[string]any{"host": host, "port": port.Num(), "username": "krenalis",
		"password": "krenalis", "database": "krenalis", "schema": "unified_writers"})
	if err != nil {
		t.Fatalf("expected warehouse settings, got %v", err)
	}
	warehouse := warehouses.Registered("PostgreSQL").New(newTestSettingsLoader(settings), nil)
	t.Cleanup(func() {
		err := warehouse.Close()
		if err != nil {
			t.Errorf("expected warehouse cleanup, got %v", err)
		}
	})
	platform := fmt.Sprintf("UnifiedWriterTest%d", time.Now().UnixNano())
	warehouses.Register(warehouses.Platform{Name: platform},
		func(warehouses.SettingsLoader, warehouses.DialWith) warehouses.Warehouse { return warehouse })

	schema := types.Object([]types.Property{
		{Name: "fixture", Type: types.String(), ReadOptional: true},
		{Name: "plain", Type: types.String(), ReadOptional: true},
		{Name: "value", Type: types.JSON(), ReadOptional: true},
		{Name: "object", ReadOptional: true, Type: types.Object([]types.Property{
			{Name: "json", Type: types.JSON(), ReadOptional: true},
			{Name: "text", Type: types.String(), ReadOptional: true},
		})},
	})
	columns := util.PropertiesToColumns(schema.Properties())
	err = warehouse.Initialize(t.Context(), columns)
	if err != nil {
		t.Fatalf("expected production warehouse tables, got %v", err)
	}
	encodedSchema, err := schema.MarshalJSON()
	if err != nil {
		t.Fatalf("expected schema encoding, got %v", err)
	}

	const workspaceID = "111111111111"
	const batchConnection = "222222222222"
	const eventConnection = "333333333333"
	const batchPipeline = "444444444444"
	const eventPipeline = "555555555555"
	_, err = database.Exec(t.Context(), `INSERT INTO workspaces
(id,organization,name,warehouse_name,warehouse_mode,warehouse_settings,
 kms_encrypted_warehouse_settings_key,kms_encrypted_warehouse_mcp_settings_key,profile_schema,identifiers)
 SELECT $1,id,'Unified Writers',$2,'Normal','','','',$3,ARRAY[]::text[] FROM organizations`,
		workspaceID, platform, encodedSchema)
	if err != nil {
		t.Fatalf("expected workspace state, got %v", err)
	}
	_, err = database.Exec(t.Context(), `INSERT INTO connections
(id,workspace,connector,role,kms_encrypted_settings_key)
 VALUES ($1,$3,'dummy','Source',''),($2,$3,'segment','Source','')`,
		batchConnection, eventConnection, workspaceID)
	if err != nil {
		t.Fatalf("expected source connections, got %v", err)
	}
	for _, source := range []struct{ pipeline, connection string }{
		{batchPipeline, batchConnection}, {eventPipeline, eventConnection},
	} {
		_, err = database.Exec(t.Context(), `INSERT INTO pipelines
(id,connection,target,event_type,ordering_group,delivery_endpoint,transformation_language,
 matching_in,matching_out,update_on_duplicates,table_key,out_schema,transformation_out_paths)
 VALUES ($1,$2,'User','','','','JavaScript','','',false,'',$3,$4)`,
			source.pipeline, source.connection, encodedSchema, schema.Properties().Names())
		if err != nil {
			t.Fatalf("expected identity pipeline, got %v", err)
		}
	}
	_, err = database.Exec(t.Context(), `INSERT INTO pipelines_runs (id,pipeline,start_time,ping_time)
 VALUES ('666666666666',$1,'2026-01-15','2026-01-15')`, batchPipeline)
	if err != nil {
		t.Fatalf("expected active batch run, got %v", err)
	}

	st, err := state.New(t.Context(), database, manager, nil, false)
	if err != nil {
		t.Fatalf("expected application state, got %v", err)
	}
	t.Cleanup(func() { st.Close(context.Background()) })
	ds, err := New(st)
	if err != nil {
		t.Fatalf("expected datastore, got %v", err)
	}
	t.Cleanup(ds.Close)
	store, ok := ds.Store(workspaceID)
	if !ok {
		t.Fatal("expected workspace store, got none")
	}
	pipeline, ok := st.Pipeline(batchPipeline)
	if !ok {
		t.Fatal("expected batch pipeline, got none")
	}

	// Constructors, flattening, queues, COPY/merge and publication are real.
	// Closing writers provides a barrier only for this controlled fixture.
	batch, err := store.NewBatchIdentityWriter(pipeline, false, unifiedWriterMetrics{})
	if err != nil {
		t.Fatalf("expected batch identity writer, got %v", err)
	}
	eventIdentities := store.NewEventIdentityWriter(eventPipeline, unifiedWriterMetrics{})
	events := store.NewEventWriter(unifiedWriterMetrics{})
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	defer batch.Stop(ctx)
	defer func() {
		if !eventIdentities.closed.Load() {
			_ = eventIdentities.Close(ctx)
		}
		if !events.closed.Load() {
			_ = events.Close(ctx)
		}
	}()

	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	observed := at.AddDate(0, 0, -15)
	fixtures := []struct {
		name, fields                           string
		properties                             json.Value
		hasEvent                               bool
		valuePresent, valueNull, objectPresent bool
		rawAbsence, governedAbsence            unified.Truth
	}{
		{"no-event", "", nil, false, false, false, false, unified.Match, unified.Match},
		{"missing", "", nil, true, false, false, false, unified.Unknown, unified.Unknown},
		{"json-null", `,"value":null`, json.Value(`null`), true, true, true, false, unified.Unknown, unified.Unknown},
		{"currency-null", `,"value":{"currency":null}`, json.Value(`{"currency":null}`), true, true, false, false, unified.Match, unified.Unknown},
		{"eur", `,"plain":"known","value":{"currency":"EUR"}`, json.Value(`{"currency":"EUR"}`), true, true, false, false, unified.NoMatch, unified.NoMatch},
		{"empty-object", `,"plain":"","value":{},"object":{}`, json.Value(`{}`), true, true, false, false, unified.Unknown, unified.Unknown},
		{"number", `,"value":{"currency":9007199254740993},"object":{"text":""}`, json.Value(`{"currency":9007199254740993}`), true, true, false, true, unified.Match, unified.Unknown},
		{"nested-null", `,"value":{"currency":"USD"},"object":{"json":null}`, json.Value(`{"currency":"USD"}`), true, true, false, true, unified.Match, unified.Match},
		{"nested-present", `,"value":{"currency":"USD"},"object":{"json":{},"text":"x"}`, json.Value(`{"currency":"USD"}`), true, true, false, true, unified.Match, unified.Match},
	}

	written := map[string]bool{}
	for _, source := range []struct{ name, connection string }{{"batch", batchConnection}, {"event", eventConnection}} {
		for _, fixture := range fixtures {
			id := source.name + "/" + fixture.name
			written[id] = true
			attributes, err := types.Decode[map[string]any](strings.NewReader(`{"fixture":"`+id+`"`+fixture.fields+`}`), schema)
			if err != nil {
				t.Fatalf("expected validated identity input, got %v", err)
			}
			identity := Identity{ID: id, AnonymousID: "anonymous/" + id, UpdatedAt: observed, Attributes: attributes}
			if source.name == "batch" {
				err = batch.Write(ctx, identity)
			} else {
				err = eventIdentities.Write(ctx, identity, nil)
			}
			if err != nil {
				t.Fatalf("expected queued identity, got %v", err)
			}
			if !fixture.hasEvent {
				continue
			}
			attributes = map[string]any{"connectionId": source.connection, "anonymousId": "anonymous/" + id,
				"userId": id, "event": "Purchase", "messageId": "message/" + id,
				"type": "track", "traits": json.Value(`{}`), "timestamp": observed, "receivedAt": observed,
				"sentAt": observed, "originalTimestamp": observed}
			if fixture.properties != nil {
				attributes["properties"] = fixture.properties
			}
			encoded, err := types.Marshal(attributes, schemas.Event)
			if err != nil {
				t.Fatalf("expected valid event encoding, got %v", err)
			}
			attributes, err = types.Decode[map[string]any](strings.NewReader(string(encoded)), schemas.Event)
			if err != nil {
				t.Fatalf("expected validated event input, got %v", err)
			}
			event := streams.Event{Attributes: attributes, Destinations: []streams.Destination{{}}}
			err = events.Write(ctx, event, eventPipeline)
			if err != nil {
				t.Fatalf("expected queued event, got %v", err)
			}
		}
	}

	err = batch.Close(ctx)
	if err != nil {
		t.Fatalf("expected completed batch writes, got %v", err)
	}
	err = eventIdentities.Close(ctx)
	if err != nil {
		t.Fatalf("expected completed event identity writes, got %v", err)
	}
	err = events.Close(ctx)
	if err != nil {
		t.Fatalf("expected completed event writes, got %v", err)
	}

	err = store.ResolveIdentities(ctx, "fb43b3c4-dcc6-42ca-99e7-83679cbd5644")
	if err != nil {
		t.Fatalf("expected completed profile publication and attribution, got %v", err)
	}

	profiles, _, err := store.Profiles(ctx,
		Query{Properties: []string{"_kpid", "fixture"}, OrderBy: "fixture", Limit: 100})
	if err != nil {
		t.Fatalf("expected written profile listing, got %v", err)
	}
	if len(profiles) != 2*len(fixtures) {
		t.Fatalf("expected %d published profiles, got %d", 2*len(fixtures), len(profiles))
	}
	names := map[string]string{}
	for _, profile := range profiles {
		name := profile["fixture"].(string)
		if !written[name] {
			t.Fatalf("expected one profile per written identity, got %q", name)
		}
		delete(written, name)
		names[profile["_kpid"].(string)] = name
	}
	if len(names) != len(profiles) {
		t.Fatalf("expected %d distinct profile IDs, got %d", len(profiles), len(names))
	}

	bindings := unified.Bindings{Currencies: []unified.CurrencyBinding{{Definition: "NativeCurrency", Revision: "1",
		Path: []string{"properties", "currency"}, Evidence: unified.IndependentOccurrenceValues}}}
	count := unified.Rule{Count: &unified.Count{Scope: unified.Scope{Event: "Purchase", Grain: "registered-message",
		Start: at.AddDate(0, -1, 0), End: at}, Test: unified.CountTest{Operator: unified.Equal},
		Currency: &unified.NativeCurrency{Definition: "NativeCurrency", Revision: "1", Value: "EUR"}}}

	kinds := []string{"value present", "value null", "object present", "plain different", "exact JSON number",
		"raw absence", "governed absence", "mixed", "event once", "event properties null", "event exact number"}
	for _, kind := range kinds {
		t.Run(kind, func(t *testing.T) {

			rule := count
			switch kind {
			case "value present":
				rule = unified.Rule{Record: &state.WhereCondition{
					Property: []string{"value"}, Operator: state.OpExists}}
			case "value null":
				rule = unified.Rule{Record: &state.WhereCondition{
					Property: []string{"value"}, Operator: state.OpIsNull}}
			case "object present":
				rule = unified.Rule{Record: &state.WhereCondition{
					Property: []string{"object"}, Operator: state.OpExists}}
			case "plain different":
				rule = unified.Rule{Record: &state.WhereCondition{Property: []string{"plain"},
					Operator: state.OpIsNot, Values: []any{"known"}}}
			case "exact JSON number":
				rule = unified.Rule{Record: &state.WhereCondition{Property: []string{"value", "currency"},
					Operator: state.OpIs, Values: []any{json.Value(`9007199254740993`)}}}
			case "raw absence":
				c := *count.Count
				c.Currency = nil
				c.Where = &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{&state.WhereCondition{
					Property: []string{"properties", "currency"}, Operator: state.OpIs,
					Values: []any{json.Value(`"EUR"`)}}}}
				rule = unified.Rule{Count: &c}
			case "mixed":
				rule = unified.Rule{Operator: state.OpAnd, Children: []unified.Rule{count,
					{Record: &state.WhereCondition{Property: []string{"object"}, Operator: state.OpExists}}}}
			case "governed absence":
				// The governed absence retains the fixture Count.
			case "event once", "event properties null", "event exact number":
				c := *count.Count
				c.Currency = nil
				switch kind {
				case "event once":
					c.Test.Value = 1
				case "event properties null":
					c.Where = &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{&state.WhereCondition{
						Property: []string{"properties"}, Operator: state.OpIsNull}}}
				case "event exact number":
					c.Where = &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{&state.WhereCondition{
						Property: []string{"properties", "currency"}, Operator: state.OpIs,
						Values: []any{json.Value(`9007199254740993`)}}}}
				default:
					t.Fatalf("expected known event query kind, got %q", kind)
				}
				rule = unified.Rule{Count: &c}
			default:
				t.Fatalf("expected known query kind, got %q", kind)
			}

			result, err := store.UnifiedProfiles(ctx, &rule, bindings, at, unified.Page{Limit: 100})
			if err != nil {
				t.Fatalf("expected complete writer-backed Unified query, got %v", err)
			}
			if len(result.Profiles) != len(names) {
				t.Fatalf("expected %d profile outcomes, got %+v", len(names), result)
			}

			expected := map[string]unified.Truth{}
			var m, f, u int64
			for _, source := range []string{"batch", "event"} {
				for _, fixture := range fixtures {
					truth := unified.NoMatch
					switch kind {
					case "value present":
						if fixture.valuePresent {
							truth = unified.Match
						}
					case "value null":
						if fixture.valueNull {
							truth = unified.Match
						}
					case "object present":
						if fixture.objectPresent {
							truth = unified.Match
						}
					case "plain different":
						truth = unified.Unknown
						if fixture.name == "eur" {
							truth = unified.NoMatch
						}
						if fixture.name == "empty-object" {
							truth = unified.Match
						}
					case "exact JSON number":
						switch fixture.name {
						case "no-event", "missing", "json-null", "empty-object":
							truth = unified.Unknown
						case "number":
							truth = unified.Match
						default:
							// All other fixtures have a different present JSON value.
						}
					case "raw absence":
						truth = fixture.rawAbsence
					case "mixed":
						if fixture.objectPresent {
							truth = fixture.governedAbsence
						}
					case "governed absence":
						truth = fixture.governedAbsence
					case "event once":
						// Keep the expected history independent of the writer's hasEvent gate.
						if fixture.name != "no-event" {
							truth = unified.Match
						}
					case "event properties null":
						truth = unified.Match
						if fixture.name == "json-null" {
							truth = unified.NoMatch
						}
					case "event exact number":
						switch fixture.name {
						case "number":
							truth = unified.NoMatch
						case "missing", "json-null", "empty-object":
							truth = unified.Unknown
						default:
							// Empty histories and different present JSON values qualify no events.
							truth = unified.Match
						}
					default:
						t.Fatalf("expected known query kind, got %q", kind)
					}
					expected[source+"/"+fixture.name] = truth
					switch truth {
					case unified.Match:
						m++
					case unified.NoMatch:
						f++
					case unified.Unknown:
						u++
					default:
						t.Fatalf("expected known truth, got %v", truth)
					}
				}
			}

			for _, outcome := range result.Profiles {
				name, ok := names[outcome.ID]
				if !ok {
					t.Fatalf("expected a published profile ID, got %q", outcome.ID)
				}
				truth, ok := expected[name]
				if !ok {
					t.Fatalf("expected a fixture name, got %q", name)
				}
				if outcome.Truth != truth {
					t.Fatalf("expected %s truth %d, got %d", name, truth, outcome.Truth)
				}
			}
			if result.Match != m || result.NoMatch != f || result.Unknown != u {
				t.Fatalf("expected totals %d/%d/%d, got %+v", m, f, u, result)
			}

		})
	}

	rome, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		t.Fatalf("expected Rome timezone, got %v", err)
	}
	for _, season := range []struct {
		name, operation string
		at              time.Time
	}{
		{"spring", "fb43b3c4-dcc6-42ca-99e7-83679cbd5645", time.Date(2026, 3, 29, 12, 0, 0, 0, rome)},
		{"autumn", "fb43b3c4-dcc6-42ca-99e7-83679cbd5646", time.Date(2026, 10, 25, 12, 0, 0, 0, rome)},
	} {
		t.Run("elapsed "+season.name, func(t *testing.T) {

			// Use new recorded events and complete attribution before reading.
			// The already published identities supply the same controlled population.
			eventName := "Elapsed " + season.name
			writer := store.NewEventWriter(unifiedWriterMetrics{})
			defer func() {
				if !writer.closed.Load() {
					_ = writer.Close(ctx)
				}
			}()
			for i, offset := range []time.Duration{
				-48*time.Hour - time.Millisecond,
				-48 * time.Hour,
				-24*time.Hour - time.Millisecond,
				-24 * time.Hour,
				-time.Millisecond,
				0,
				time.Millisecond,
				-400 * time.Microsecond,
				-600 * time.Microsecond,
				-24*time.Hour - 400*time.Microsecond,
				-24*time.Hour - 600*time.Microsecond,
				-24 * time.Hour,
			} {
				connection, id, properties := batchConnection, "batch/eur", json.Value(`{"currency":"EUR"}`)
				if i == 11 {
					connection, id, properties = eventConnection, "event/missing", json.Value(`{"currency":null}`)
				}
				stamp := season.at.Add(offset)
				attributes := map[string]any{"connectionId": connection, "anonymousId": "anonymous/" + id,
					"userId": id, "event": eventName, "messageId": fmt.Sprintf("elapsed/%s/%d", season.name, i),
					"type": "track", "traits": json.Value(`{}`), "properties": properties,
					"timestamp": stamp, "receivedAt": stamp, "sentAt": stamp, "originalTimestamp": stamp}
				encoded, err := types.Marshal(attributes, schemas.Event)
				if err != nil {
					t.Fatalf("expected elapsed event encoding, got %v", err)
				}
				attributes, err = types.Decode[map[string]any](strings.NewReader(string(encoded)), schemas.Event)
				if err != nil {
					t.Fatalf("expected validated elapsed event, got %v", err)
				}
				event := streams.Event{Attributes: attributes, Destinations: []streams.Destination{{}}}
				err = writer.Write(ctx, event, eventPipeline)
				if err != nil {
					t.Fatalf("expected queued elapsed event, got %v", err)
				}
			}
			err = writer.Close(ctx)
			if err != nil {
				t.Fatalf("expected completed elapsed event writes, got %v", err)
			}
			err = store.ResolveIdentities(ctx, season.operation)
			if err != nil {
				t.Fatalf("expected completed elapsed event attribution, got %v", err)
			}
			published, _, err := store.Profiles(ctx,
				Query{Properties: []string{"_kpid", "fixture"}, OrderBy: "fixture", Limit: 100})
			if err != nil {
				t.Fatalf("expected published elapsed profile listing, got %v", err)
			}
			if len(published) != len(names) {
				t.Fatalf("expected %d published profiles, got %d", len(names), len(published))
			}
			profileNames := map[string]string{}
			for _, profile := range published {
				profileNames[profile["_kpid"].(string)] = profile["fixture"].(string)
			}

			for _, test := range []struct {
				name                    string
				period                  unified.ElapsedPeriod
				value                   uint64
				currency, combined      bool
				known, uncertain, other unified.Truth
			}{
				{"lookback count", unified.ElapsedPeriod{Older: 24 * time.Hour}, 4, false, false, unified.Match, unified.NoMatch, unified.NoMatch},
				{"lookback absence", unified.ElapsedPeriod{Older: 24 * time.Hour}, 0, true, false, unified.NoMatch, unified.Unknown, unified.Match},
				{"ago band count", unified.ElapsedPeriod{Older: 48 * time.Hour, Newer: 24 * time.Hour}, 3, false, false, unified.Match, unified.NoMatch, unified.NoMatch},
				{"ago band absence", unified.ElapsedPeriod{Older: 48 * time.Hour, Newer: 24 * time.Hour}, 0, true, false, unified.NoMatch, unified.Match, unified.Match},
				{"microsecond start", unified.ElapsedPeriod{Older: 24*time.Hour - time.Microsecond}, 2, false, false, unified.Match, unified.NoMatch, unified.NoMatch},
				{"microsecond end", unified.ElapsedPeriod{Older: 48 * time.Hour, Newer: 24*time.Hour - time.Microsecond}, 5, false, false, unified.Match, unified.NoMatch, unified.NoMatch},
				{"microsecond empty", unified.ElapsedPeriod{Older: time.Microsecond}, 0, false, false, unified.Match, unified.Match, unified.Match},
				{"one T for both periods", unified.ElapsedPeriod{Older: 24 * time.Hour}, 4, false, true, unified.Match, unified.NoMatch, unified.NoMatch},
			} {
				t.Run(test.name, func(t *testing.T) {

					c := unified.Count{Scope: unified.Scope{Event: eventName,
						Grain: "registered-message", Elapsed: &test.period},
						Test: unified.CountTest{Operator: unified.Equal, Value: test.value}}
					if test.currency {
						c.Currency = &unified.NativeCurrency{Definition: "NativeCurrency", Revision: "1", Value: "EUR"}
					}
					root := unified.Rule{Count: &c}
					if test.combined {
						previous := unified.Count{Scope: unified.Scope{Event: eventName, Grain: "registered-message",
							Elapsed: &unified.ElapsedPeriod{Older: 48 * time.Hour, Newer: 24 * time.Hour}},
							Test: unified.CountTest{Operator: unified.Equal, Value: 3}}
						root = unified.Rule{Operator: state.OpAnd, Children: []unified.Rule{root, {Count: &previous}}}
					}
					result, err := store.UnifiedProfiles(ctx, &root, bindings, season.at, unified.Page{Limit: 100})
					if err != nil {
						t.Fatalf("expected complete elapsed writer-backed query, got %v", err)
					}
					if len(result.Profiles) != len(profileNames) || !result.At.Equal(season.at) ||
						result.At.Location() != time.UTC {
						t.Fatalf("expected %d outcomes at %s in UTC, got %+v", len(profileNames), season.at, result)
					}

					var match, noMatch, unknown int64
					for _, outcome := range result.Profiles {
						name, ok := profileNames[outcome.ID]
						if !ok {
							t.Fatalf("expected published profile ID, got %q", outcome.ID)
						}
						expected := test.other
						switch name {
						case "batch/eur":
							expected = test.known
						case "event/missing":
							expected = test.uncertain
						default:
							// All other profiles have no events in this named repertoire.
						}
						if outcome.Truth != expected {
							t.Fatalf("expected %s truth %d, got %d", name, expected, outcome.Truth)
						}
						switch expected {
						case unified.Match:
							match++
						case unified.NoMatch:
							noMatch++
						case unified.Unknown:
							unknown++
						default:
							t.Fatalf("expected known truth, got %v", expected)
						}
					}
					if result.Match != match || result.NoMatch != noMatch || result.Unknown != unknown {
						t.Fatalf("expected elapsed totals %d/%d/%d, got %+v", match, noMatch, unknown, result)
					}
					if test.currency && test.period.Newer == 0 {
						empty, err := store.UnifiedProfiles(ctx, &root, bindings, season.at.UTC(),
							unified.Page{First: 100, Limit: 2, MatchingOnly: true})
						if err != nil {
							t.Fatalf("expected empty elapsed page with totals, got %v", err)
						}
						if len(empty.Profiles) != 0 || empty.Match != match || empty.NoMatch != noMatch ||
							empty.Unknown != unknown || !empty.At.Equal(result.At) {
							t.Fatalf("expected empty page and same ternary totals at T, got %+v", empty)
						}
					}

				})
			}

		})
	}

}
