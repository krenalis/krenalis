// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/initdb"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/kms"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
	"github.com/krenalis/krenalis/warehouses/postgresql"
	"github.com/krenalis/krenalis/warehouses/snowflake"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var unifiedWarehouseNumber atomic.Uint64

type unifiedTestSettings struct {
	json json.Value
}

// Load decodes the settings of the disposable warehouse.
func (s unifiedTestSettings) Load(ctx context.Context, value any) error {
	return json.Unmarshal(s.json, value)
}

// TestUnifiedWorkspacePostgreSQL exercises the internal service with real
// state.
func TestUnifiedWorkspacePostgreSQL(t *testing.T) {

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

	// The application state and warehouse live in separate schemas of this
	// disposable database. All evidence is published before the service call.
	_, err = database.Exec(t.Context(), `CREATE SCHEMA unified_v;
CREATE TABLE unified_v.profiles (_kpid uuid PRIMARY KEY,country text,profile_id text,event text,
 _updated_at timestamp(6) NOT NULL DEFAULT '2026-01-31 12:00:00.123456',document jsonb,amount numeric(38,10),
 settings_value text,tags text[]);
CREATE TABLE unified_v.krenalis_events (kpid uuid,message_id text PRIMARY KEY,
 event text,timestamp timestamp(3) NOT NULL,properties jsonb,type text NOT NULL DEFAULT 'track');
INSERT INTO unified_v.profiles (_kpid)
 SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid FROM generate_series(1,4) n;
UPDATE unified_v.profiles SET event='Refund',profile_id='customer-'||right(_kpid::text,1),
 country=CASE right(_kpid::text,1) WHEN '1' THEN 'IT' WHEN '2' THEN 'FR' WHEN '4' THEN 'IT' END;
UPDATE unified_v.profiles SET document='{"currency":null}',amount=9007199254740993.0000000001,
 settings_value='nested',tags=ARRAY['first','second'];
INSERT INTO unified_v.krenalis_events VALUES
('00000000-0000-0000-0000-000000000002','e2','Purchase','2026-01-15','{"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000003','e3','Purchase','2026-01-15','{"currency":null}'),
('00000000-0000-0000-0000-000000000004','e4','Purchase','2026-01-15','{}');`)
	if err != nil {
		t.Fatalf("expected coherent warehouse fixtures, got %v", err)
	}
	settings, err := json.Marshal(map[string]any{"host": host, "port": port.Num(), "username": "krenalis",
		"password": "krenalis", "database": "krenalis", "schema": "unified_v"})
	if err != nil {
		t.Fatalf("expected settings encoding, got %v", err)
	}
	warehouse := warehouses.Registered("PostgreSQL").New(unifiedTestSettings{settings}, nil).(*postgresql.PostgreSQL)
	platform := fmt.Sprintf("UnifiedCountTest%d", unifiedWarehouseNumber.Add(1))
	warehouses.Register(warehouses.Platform{Name: platform}, func(warehouses.SettingsLoader, warehouses.DialWith) *postgresql.PostgreSQL { return warehouse })
	schema, err := types.Object([]types.Property{

		{Name: "country", Type: types.String(), ReadOptional: true},
		{Name: "profile_id", Type: types.String(), ReadOptional: true},
		{Name: "event", Type: types.String(), ReadOptional: true},
		{Name: "document", Type: types.JSON(), ReadOptional: true},
		{Name: "amount", Type: types.Decimal(38, 10), ReadOptional: true},
		{Name: "settings", ReadOptional: true, Type: types.Object([]types.Property{
			{Name: "value", Type: types.String(), ReadOptional: true},
		})},
		{Name: "tags", Type: types.Array(types.String()), ReadOptional: true},
	}).MarshalJSON()
	if err != nil {
		t.Fatalf("expected profile schema encoding, got %v", err)
	}
	_, err = database.Exec(t.Context(), `INSERT INTO workspaces
 (id,organization,name,warehouse_name,warehouse_mode,warehouse_settings,
 kms_encrypted_warehouse_settings_key,profile_schema,identifiers)
 SELECT '111111111111',id,'Unified Count',$1,'Normal','','',$2,ARRAY[]::text[] FROM organizations`, platform, schema)
	if err != nil {
		t.Fatalf("expected workspace state fixture, got %v", err)
	}
	unverified := warehouses.Registered("Snowflake").New(unifiedTestSettings{json.Value(`{}`)}, nil).(*snowflake.Snowflake)
	unverifiedPlatform := platform + "Unverified"
	warehouses.Register(warehouses.Platform{Name: unverifiedPlatform},
		func(warehouses.SettingsLoader, warehouses.DialWith) *snowflake.Snowflake { return unverified })
	_, err = database.Exec(t.Context(), `INSERT INTO workspaces
 (id,organization,name,warehouse_name,warehouse_mode,warehouse_settings,
 kms_encrypted_warehouse_settings_key,profile_schema,identifiers)
 SELECT '222222222222',id,'Unverified Count',$1,'Normal','','',$2,ARRAY[]::text[]
 FROM organizations`, unverifiedPlatform, schema)
	if err != nil {
		t.Fatalf("expected unverified backend fixture, got %v", err)
	}
	st, err := state.New(t.Context(), database, manager, nil, false)
	if err != nil {
		t.Fatalf("expected application state, got %v", err)
	}
	t.Cleanup(func() { st.Close(context.Background()) })
	ds, err := datastore.New(st)
	if err != nil {
		t.Fatalf("expected datastore, got %v", err)
	}
	t.Cleanup(ds.Close)
	store, ok := ds.Store("111111111111")
	if !ok {
		t.Fatal("expected workspace store, got none")
	}
	ws, ok := st.Workspace("111111111111")
	if !ok {
		t.Fatal("expected workspace state, got none")
	}
	workspace := &Workspace{core: &Core{}, store: store, workspace: ws}

	t.Run("Record-only default calendar", func(t *testing.T) {

		filter := &Filter{Operator: OpAnd, Rules: []FilterRule{
			&FilterCondition{Property: "country", Operator: OpIs, Values: filterValues("IT")}}}
		query := ProfileQuery{Filter: filter, Properties: []string{"profile_id"}, Order: "profile_id", Limit: 10}
		expected, err := workspace.QueryProfiles(t.Context(), query)
		if err != nil {
			t.Fatalf("expected Record query, got %v", err)
		}
		filter.DefaultCalendar = &FilterCalendar{Definition: "gregorian-monday/Europe/Rome", MeaningRevision: "1"}
		before, err := json.Marshal(filter)
		if err != nil {
			t.Fatalf("expected calendar metadata encoding, got %v", err)
		}

		result, err := workspace.QueryProfiles(t.Context(), query)
		if err != nil {
			t.Fatalf("expected Record query retaining defaultCalendar, got %v", err)
		}
		if expected.Total != 2 || expected.Evaluation != nil || !reflect.DeepEqual(result, expected) {
			t.Fatalf("expected unchanged Record result %+v without evaluation, got %+v", expected, result)
		}
		after, err := json.Marshal(filter)
		if err != nil {
			t.Fatalf("expected retained calendar metadata encoding, got %v", err)
		}
		if string(before) != string(after) {
			t.Fatalf("expected unchanged filter %s, got %s", before, after)
		}

	})

	var document Filter
	err = json.Unmarshal([]byte(`{"operator":"and","rules":[{"count":{`+
		`"source":{"kind":"recordedEvents","name":"Purchase","grain":"registered-message"},`+
		`"period":{"kind":"elapsedLookback","duration":{"value":31,"unit":"day"}},`+
		`"where":{"operator":"and","rules":[`+
		`{"property":"properties.currency","operator":"is","values":["EUR"]}]}},`+
		`"operator":"is","values":[0]}]}`), &document)
	if err != nil {
		t.Fatalf("expected public Count document decoding, got %v", err)
	}
	encodedDocument, err := json.Marshal(&document)
	if err != nil {
		t.Fatalf("expected public Count document encoding, got %v", err)
	}
	var storedDocument []byte
	err = database.QueryRow(t.Context(), "SELECT $1::jsonb", string(encodedDocument)).Scan(&storedDocument)
	if err != nil {
		t.Fatalf("expected JSONB document reloading, got %v", err)
	}
	err = json.Unmarshal(storedDocument, &document)
	if err != nil {
		t.Fatalf("expected reloaded public Count document, got %v", err)
	}
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	count := document.Rules[0].(*FilterCountCondition)
	countRoot := &document

	t.Run("filter explanations", func(t *testing.T) {

		record := &Filter{Operator: OpAnd, Rules: []FilterRule{
			&FilterCondition{Property: "country", Operator: OpIs, Values: filterValues("IT")}},
			DefaultCalendar: &FilterCalendar{Definition: "gregorian-monday/Europe/Rome", MeaningRevision: "1"}}
		for i, expected := range []string{"MATCH", "NO_MATCH", "UNKNOWN", "MATCH"} {
			result, err := workspace.ExplainProfileFilter(t.Context(),
				fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1), record, at.Format(time.RFC3339))
			if err != nil {
				t.Fatalf("expected explained Record result, got %v", err)
			}
			if result.Rule.Result != expected || len(result.Rule.Rules) != 1 ||
				result.Rule.Rules[0].Kind != "record" || result.Rule.Rules[0].Count != nil {
				t.Fatalf("expected Record %s without count details, got %+v", expected, result)
			}
		}

		for i, expected := range []struct {
			truth              string
			certain, uncertain int64
		}{{"MATCH", 0, 0}, {"NO_MATCH", 1, 0}, {"MATCH", 0, 0}, {"UNKNOWN", 0, 1}} {
			profile := fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1)
			result, err := workspace.ExplainProfileFilter(t.Context(), profile, countRoot, at.Format(time.RFC3339))
			if err != nil {
				t.Fatalf("expected complete explanation for profile %d, got %v", i+1, err)
			}
			if result.KPID != profile || !result.At.Equal(at) || result.Rule.Result != expected.truth ||
				result.Rule.Kind != "and" || len(result.Rule.Rules) != 1 {
				t.Fatalf("expected %s at %s for profile %s, got %+v", expected.truth, at, profile, result)
			}
			leaf := result.Rule.Rules[0]
			if leaf.Kind != "count" || leaf.Result != expected.truth || leaf.Count == nil ||
				leaf.Count.Certain != expected.certain || leaf.Count.Uncertain != expected.uncertain ||
				!leaf.Count.Start.Equal(at.AddDate(0, 0, -31)) || !leaf.Count.End.Equal(at) ||
				leaf.Count.History != nil || leaf.Count.UnboundedStart {
				t.Fatalf("expected Count %d certain and %d uncertain with exact period, got %+v",
					expected.certain, expected.uncertain, leaf)
			}
			data, err := json.Marshal(leaf.Count)
			if err != nil {
				t.Fatalf("expected bounded Count encoding, got %v", err)
			}
			expectedJSON := fmt.Sprintf(`{"start":"2026-01-01T00:00:00Z","end":"2026-02-01T00:00:00Z",`+
				`"certain":%d,"uncertain":%d}`, expected.certain, expected.uncertain)
			if string(data) != expectedJSON {
				t.Fatalf("expected unchanged bounded Count JSON %s, got %s", expectedJSON, data)
			}
		}

		_, err := database.Exec(t.Context(), `INSERT INTO unified_v.krenalis_events VALUES
('00000000-0000-0000-0000-000000000004','explain-certain','Explain','2026-01-15','{"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000004','explain-uncertain','Explain','2026-01-15','{}'),
('00000000-0000-0000-0000-000000000004','explain-outside','Explain','2026-02-02','{"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000002','explain-other','Explain','2026-01-15','{"currency":"EUR"}')`)
		if err != nil {
			t.Fatalf("expected mixed evidence fixtures, got %v", err)
		}
		mixed := *count
		mixed.Count.Source.Name = "Explain"
		mixed.Operator, mixed.Values = OpIsGreaterThanOrEqualTo, filterValues(2)
		result, err := workspace.ExplainProfileFilter(t.Context(), "00000000-0000-0000-0000-000000000004",
			&Filter{Operator: OpOr, Rules: []FilterRule{
				&Filter{Operator: OpAnd, Rules: []FilterRule{&mixed}},
				&FilterCondition{Property: "country", Operator: OpIs, Values: filterValues("IT")}}}, at.Format(time.RFC3339))
		if err != nil {
			t.Fatalf("expected explained mixed evidence and nested composition, got %v", err)
		}
		child := result.Rule.Rules[0]
		if result.Rule.Kind != "or" || result.Rule.Result != "MATCH" || child.Kind != "and" ||
			child.Result != "UNKNOWN" || child.Rules[0].Count.Certain != 1 || child.Rules[0].Count.Uncertain != 1 ||
			result.Rule.Rules[1].Kind != "record" || result.Rule.Rules[1].Result != "MATCH" {
			t.Fatalf("expected MATCH OR with UNKNOWN [1,2] Count, got %+v", result)
		}

		root := &Filter{Operator: OpAnd, Rules: []FilterRule{count}}
		for range 99 {
			root.Rules = append(root.Rules, &FilterCondition{Property: "country", Operator: OpExists})
		}
		result, err = workspace.ExplainProfileFilter(t.Context(), "00000000-0000-0000-0000-000000000004",
			root, at.Format(time.RFC3339))
		if err != nil {
			t.Fatalf("expected explanation of all 101 nodes, got %v", err)
		}
		if len(result.Rule.Rules) != 100 || result.Rule.Result != "UNKNOWN" {
			t.Fatalf("expected 100 children and UNKNOWN, got %+v", result)
		}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		result, err = workspace.ExplainProfileFilter(ctx, "00000000-0000-0000-0000-000000000004",
			countRoot, at.Format(time.RFC3339))
		if err != nil {
			if result != nil {
				t.Fatalf("expected cancellation without explanation, got %+v", result)
			}
		}
		if err == nil {
			t.Fatal("expected cancellation error, got success")
		}

		result, err = workspace.ExplainProfileFilter(t.Context(), "00000000-0000-0000-0000-000000000099",
			countRoot, at.Format(time.RFC3339))
		if err != nil {
			if _, ok := errors.AsType[*errors.NotFoundError](err); !ok || result != nil {
				t.Fatalf("expected missing profile without explanation, got %+v, %v", result, err)
			}
			return
		}
		t.Fatal("expected profile not found, got success")

	})

	t.Run("unverified backend API error", func(t *testing.T) {
		store, ok := ds.Store("222222222222")
		if !ok {
			t.Fatal("expected unverified workspace store, got none")
		}
		ws, ok := st.Workspace("222222222222")
		if !ok {
			t.Fatal("expected unverified workspace state, got none")
		}
		unverified := &Workspace{core: &Core{}, store: store, workspace: ws}
		result, err := unverified.QueryProfiles(t.Context(), ProfileQuery{Filter: countRoot,
			At: at.Format(time.RFC3339Nano), Limit: 10})
		if err != nil {
			failure, ok := errors.AsType[*errors.UnprocessableError](err)
			if !ok || failure.Code != UnifiedFilteringUnavailable {
				t.Fatalf("expected Unprocessable UnifiedFilteringUnavailable, got %T: %v", err, err)
			}
		}
		if err == nil || result != nil {
			t.Fatalf("expected unavailable backend without semantic result, got %+v, %v", result, err)
		}
		explained, err := unverified.ExplainProfileFilter(t.Context(), "00000000-0000-0000-0000-000000000001",
			countRoot, at.Format(time.RFC3339))
		if err != nil {
			failure, ok := errors.AsType[*errors.UnprocessableError](err)
			if !ok || failure.Code != UnifiedFilteringUnavailable || explained != nil {
				t.Fatalf("expected unavailable explanation backend, got %+v, %v", explained, err)
			}
			return
		}
		t.Fatal("expected unavailable explanation backend, got success")
	})
	result, err := workspace.unifiedProfiles(t.Context(), unifiedQuery{
		Root: countRoot, At: at, Page: unified.Page{Limit: 10},
	})
	if err != nil {
		t.Fatalf("expected complete service/datastore/warehouse query, got %v", err)
	}
	want := []unified.Truth{unified.Match, unified.NoMatch, unified.Match, unified.Unknown}
	if len(result.Profiles) != len(want) || result.Match != 2 || result.NoMatch != 1 || result.Unknown != 1 {
		t.Fatalf("expected four outcomes with totals 2/1/1, got %+v", result)
	}
	for i, profile := range result.Profiles {
		if profile.Truth != want[i] {
			t.Fatalf("expected profile %d truth %d, got %+v", i, want[i], profile)
		}
	}

	start := at.AddDate(0, -1, 0)
	for _, period := range []FilterPeriod{
		{Kind: "absoluteHalfOpen", Start: &start, End: &at},
		{Kind: "elapsedLookback", Duration: &FilterDuration{Value: json.Value(`31`), Unit: "day"}},
		{Kind: "elapsedAgoBand", Older: &FilterDuration{Value: json.Value(`31`), Unit: "day"},
			Newer: &FilterDuration{Value: json.Value(`14`), Unit: "day"}},
	} {
		t.Run(period.Kind, func(t *testing.T) {
			c := *count
			c.Count.Period = period
			root := Filter{Operator: OpAnd, Rules: []FilterRule{&c}}
			result, err := workspace.unifiedProfiles(t.Context(), unifiedQuery{
				Root: &root, At: at, Page: unified.Page{Limit: 10},
			})
			if err != nil {
				t.Fatalf("expected elapsed service/datastore/warehouse query, got %v", err)
			}
			if len(result.Profiles) != len(want) || result.Match != 2 || result.NoMatch != 1 || result.Unknown != 1 ||
				!result.At.Equal(at) {
				t.Fatalf("expected four elapsed outcomes with totals 2/1/1 at %s, got %+v", at, result)
			}
			for i, profile := range result.Profiles {
				if profile.Truth != want[i] {
					t.Fatalf("expected elapsed profile %d truth %d, got %+v", i, want[i], profile)
				}
			}
		})
	}

	t.Run("full recorded history", func(t *testing.T) {

		_, err := database.Exec(t.Context(), `INSERT INTO unified_v.krenalis_events VALUES
('00000000-0000-0000-0000-000000000001','history-oldest','History','0001-01-01','{"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000001','history-old','History','1900-01-01','{"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000002','history-before','History','2026-01-31 23:59:59.999','{"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000002','history-at','History','2026-02-01','{"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000003','history-null','History','1800-01-01','{"currency":null}'),
('00000000-0000-0000-0000-000000000004','history-after','History','2026-02-02','{}')`)
		if err != nil {
			t.Fatalf("expected recorded history fixtures, got %v", err)
		}
		filter := &Filter{Operator: OpAnd, Rules: []FilterRule{&FilterCountCondition{
			Count: FilterCount{Source: FilterCountSource{Kind: "recordedEvents", Name: "History", Grain: "registered-message"},
				Period: FilterPeriod{Kind: "fullRelevantHistory", History: &FilterHistory{
					Definition: "registered-queryable-events", MeaningRevision: "1"}}},
			Operator: OpIsGreaterThanOrEqualTo, Values: filterValues(1)}}}
		before, err := json.Marshal(filter)
		if err != nil {
			t.Fatalf("expected history filter encoding, got %v", err)
		}
		var stored []byte
		err = database.QueryRow(t.Context(), "SELECT $1::jsonb", string(before)).Scan(&stored)
		if err != nil {
			t.Fatalf("expected history document JSONB round-trip, got %v", err)
		}
		var restored Filter
		err = json.Unmarshal(stored, &restored)
		if err != nil {
			t.Fatalf("expected restored history reference, got %v", err)
		}
		query := ProfileQuery{Filter: &restored, At: at.Format(time.RFC3339), Properties: []string{"profile_id"},
			Order: "profile_id", Limit: 10}
		result, err := workspace.QueryProfiles(t.Context(), query)
		if err != nil {
			t.Fatalf("expected full-history query, got %v", err)
		}
		if result.Total != 3 || len(result.Profiles) != 3 || result.Evaluation.Match != 3 ||
			result.Evaluation.NoMatch != 1 || result.Evaluation.Unknown != 0 || !result.Evaluation.At.Equal(at) {
			t.Fatalf("expected three matches and one nonmatch on recorded history, got %+v", result)
		}
		for i, expected := range []int64{2, 1, 1, 0} {
			explained, err := workspace.ExplainProfileFilter(t.Context(),
				fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1), &restored, query.At)
			if err != nil {
				t.Fatalf("expected measured history explanation, got %v", err)
			}
			count := explained.Rule.Rules[0].Count
			if count == nil || !count.UnboundedStart || !count.Start.IsZero() || !count.End.Equal(at) ||
				count.Certain != expected || count.Uncertain != 0 || count.History == nil ||
				count.History.Definition != "registered-queryable-events" || count.History.MeaningRevision != "1" {
				t.Fatalf("expected unbounded history before T with %d occurrences, got %+v", expected, count)
			}
			data, err := json.Marshal(explained)
			if err != nil {
				t.Fatalf("expected history explanation encoding, got %v", err)
			}
			if strings.Contains(string(data), `"start":`) || !strings.Contains(string(data), `"unboundedStart":true`) {
				t.Fatalf("expected explicit unbounded history without a fictional start, got %s", data)
			}
		}
		query.First = 99
		empty, err := workspace.QueryProfiles(t.Context(), query)
		if err != nil {
			t.Fatalf("expected empty history page, got %v", err)
		}
		if len(empty.Profiles) != 0 || empty.Total != result.Total || *empty.Evaluation != *result.Evaluation {
			t.Fatalf("expected unchanged history totals on empty page, got %+v", empty)
		}

		// A prior occurrence and no recent occurrence is an AND of two Counts.
		recent := *restored.Rules[0].(*FilterCountCondition)
		recent.Count.Period = FilterPeriod{Kind: "elapsedLookback",
			Duration: &FilterDuration{Value: json.Value(`30`), Unit: "day"}}
		recent.Operator, recent.Values = OpIs, filterValues(0)
		restored.Rules = append(restored.Rules, &recent)
		query.First = 0
		result, err = workspace.QueryProfiles(t.Context(), query)
		if err != nil {
			t.Fatalf("expected history and recent-window composition, got %v", err)
		}
		if result.Total != 2 || len(result.Profiles) != 2 ||
			result.Profiles[0].Attributes["profile_id"] != "customer-1" ||
			result.Profiles[1].Attributes["profile_id"] != "customer-3" {
			t.Fatalf("expected two historical-only profiles, got %+v", result)
		}
		mixed := &Filter{Operator: OpOr, Rules: []FilterRule{
			&FilterCondition{Property: "profile_id", Operator: OpIs, Values: filterValues("unmatched")}, &restored}}
		explained, err := workspace.ExplainProfileFilter(t.Context(), result.Profiles[0].KPID, mixed, query.At)
		if err != nil {
			t.Fatalf("expected nested Record and Count explanation, got %v", err)
		}
		children := explained.Rule.Rules[1].Rules
		if explained.Rule.Result != "MATCH" || explained.Rule.Rules[0].Kind != "record" ||
			children[0].Count.History == nil || children[0].Count.History.Definition != "registered-queryable-events" ||
			children[0].Count.History.MeaningRevision != "1" || children[1].Count.History != nil ||
			children[1].Count.UnboundedStart || children[1].Count.Start.IsZero() {
			t.Fatalf("expected history reference only on the corresponding Count, got %+v", children)
		}
		after, err := json.Marshal(filter)
		if err != nil {
			t.Fatalf("expected history encoding after execution, got %v", err)
		}
		if string(before) != string(after) {
			t.Fatalf("expected unchanged original document %s, got %s", before, after)
		}

	})

	t.Run("page and screen Count", func(t *testing.T) {

		_, err := database.Exec(t.Context(), `ALTER TABLE unified_v.krenalis_events
 ADD COLUMN name text, ADD COLUMN context_page_path text, ADD COLUMN context_page_url text;
INSERT INTO unified_v.krenalis_events
 (kpid,message_id,type,name,context_page_path,context_page_url,event,timestamp,properties)
 SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid,
 'visit/'||n||'/'||id,typ,name,path,'https://example.test'||path,event,'2026-01-15','{}'
 FROM (VALUES
 (1,'named','page','Pricing','/pricing',NULL),
 (1,'empty','page','','/pricing',NULL),
 (2,'null','page',NULL,'/pricing','Pricing'),
 (2,'lower','page','pricing','/pricing',NULL),
 (2,'empty','page','','/pricing',NULL),
 (2,'slash','page','Other','/pricing/',NULL),
 (3,'missing-path','page','Pricing',NULL,NULL),
 (3,'catalog','page','Catalog','/catalog',NULL),
 (3,'track','track','Pricing','/pricing','Pricing'),
 (4,'named','screen','Pricing',NULL,NULL),
 (4,'null','screen',NULL,NULL,NULL),
 (4,'empty','screen','',NULL,NULL)
 ) v(n,id,typ,name,path,event);
INSERT INTO unified_v.krenalis_events (kpid,message_id,type,timestamp,context_page_path,context_page_url)
 VALUES ('00000000-0000-0000-0000-000000000001','visit/omitted','page','2026-01-15',
 '/pricing','https://example.test/pricing');
INSERT INTO unified_v.krenalis_events (kpid,message_id,type,name,timestamp,context_page_path)
 VALUES ('00000000-0000-0000-0000-000000000002','visit/end','page','Pricing','2026-02-01','/pricing')`)
		if err != nil {
			t.Fatalf("expected page/screen fixtures, got %v", err)
		}
		defer func() {
			_, err := database.Exec(t.Context(), `DELETE FROM unified_v.krenalis_events WHERE message_id LIKE 'visit/%';
ALTER TABLE unified_v.krenalis_events DROP COLUMN name, DROP COLUMN context_page_path, DROP COLUMN context_page_url`)
			if err != nil {
				t.Errorf("expected page/screen fixture cleanup, got %v", err)
			}
		}()
		for _, test := range []struct {
			name, typ, event, property, value string
			operator                          FilterOperator
			threshold                         int
			truths                            [4]string
			bounds                            [4][2]int64
		}{
			{"named pages", "page", "Pricing", "", "", OpIs, 0, [4]string{"NO_MATCH", "MATCH", "NO_MATCH", "MATCH"}, [4][2]int64{{1, 0}, {0, 0}, {1, 0}, {0, 0}}},
			{"all pages", "page", "", "", "", OpIsGreaterThanOrEqualTo, 3, [4]string{"MATCH", "MATCH", "NO_MATCH", "NO_MATCH"}, [4][2]int64{{3, 0}, {4, 0}, {2, 0}, {0, 0}}},
			{"raw name where", "page", "", "name", "Pricing", OpIs, 0, [4]string{"NO_MATCH", "UNKNOWN", "NO_MATCH", "MATCH"}, [4][2]int64{{1, 1}, {0, 1}, {1, 0}, {0, 0}}},
			{"path frequency", "page", "", "context.page.path", "/pricing", OpIsGreaterThanOrEqualTo, 3, [4]string{"MATCH", "MATCH", "NO_MATCH", "NO_MATCH"}, [4][2]int64{{3, 0}, {3, 0}, {0, 1}, {0, 0}}},
			{"path absence", "page", "", "context.page.path", "/pricing", OpIs, 0, [4]string{"NO_MATCH", "NO_MATCH", "UNKNOWN", "MATCH"}, [4][2]int64{{3, 0}, {3, 0}, {0, 1}, {0, 0}}},
			{"path case", "page", "", "context.page.path", "/PRICING", OpIs, 0, [4]string{"MATCH", "MATCH", "UNKNOWN", "MATCH"}, [4][2]int64{{0, 0}, {0, 0}, {0, 1}, {0, 0}}},
			{"path slash", "page", "", "context.page.path", "/pricing/", OpIsGreaterThanOrEqualTo, 1, [4]string{"NO_MATCH", "MATCH", "UNKNOWN", "NO_MATCH"}, [4][2]int64{{0, 0}, {1, 0}, {0, 1}, {0, 0}}},
			{"literal URL", "page", "", "context.page.url", "https://example.test/pricing", OpIsGreaterThanOrEqualTo, 1, [4]string{"MATCH", "MATCH", "UNKNOWN", "NO_MATCH"}, [4][2]int64{{3, 0}, {3, 0}, {0, 1}, {0, 0}}},
			{"named screens", "screen", "Pricing", "", "", OpIsGreaterThanOrEqualTo, 1, [4]string{"NO_MATCH", "NO_MATCH", "NO_MATCH", "MATCH"}, [4][2]int64{{0, 0}, {0, 0}, {0, 0}, {1, 0}}},
			{"all screens", "screen", "", "", "", OpIs, 3, [4]string{"NO_MATCH", "NO_MATCH", "NO_MATCH", "MATCH"}, [4][2]int64{{0, 0}, {0, 0}, {0, 0}, {3, 0}}},
			{"legacy track", "", "Pricing", "", "", OpIsGreaterThanOrEqualTo, 1, [4]string{"NO_MATCH", "NO_MATCH", "MATCH", "NO_MATCH"}, [4][2]int64{{0, 0}, {0, 0}, {1, 0}, {0, 0}}},
			{"explicit track", "track", "Pricing", "", "", OpIsGreaterThanOrEqualTo, 1, [4]string{"NO_MATCH", "NO_MATCH", "MATCH", "NO_MATCH"}, [4][2]int64{{0, 0}, {0, 0}, {1, 0}, {0, 0}}},
		} {
			t.Run(test.name, func(t *testing.T) {
				condition := &FilterCountCondition{Count: FilterCount{
					Source: FilterCountSource{Kind: "recordedEvents", Type: test.typ,
						Name: test.event, Grain: "registered-message"},
					Period: FilterPeriod{Kind: "elapsedLookback",
						Duration: &FilterDuration{Value: json.Value(`31`), Unit: "day"}}},
					Operator: test.operator, Values: filterValues(test.threshold)}
				if test.property != "" {
					condition.Count.Where = &Filter{Operator: OpAnd, Rules: []FilterRule{
						&FilterCondition{Property: test.property, Operator: OpIs, Values: filterValues(test.value)}}}
				}
				filter := &Filter{Operator: OpAnd, Rules: []FilterRule{condition}}
				before, err := json.Marshal(filter)
				if err != nil {
					t.Fatalf("expected page/screen filter encoding, got %v", err)
				}
				query := ProfileQuery{Filter: filter, At: at.Format(time.RFC3339),
					Properties: []string{"profile_id"}, Order: "profile_id", Limit: 10}
				page, err := workspace.QueryProfiles(t.Context(), query)
				if err != nil {
					t.Fatalf("expected public page/screen Count, got %v", err)
				}
				totals := [3]int64{}
				expectedProfiles := []string{}
				for i, truth := range test.truths {
					profile := fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1)
					explained, err := workspace.ExplainProfileFilter(t.Context(), profile, filter, query.At)
					if err != nil {
						t.Fatalf("expected page/screen explanation, got %v", err)
					}
					if explained.Rule.Result != truth || len(explained.Rule.Rules) != 1 {
						t.Fatalf("expected %s for profile %s, got %+v", truth, profile, explained)
					}
					count := explained.Rule.Rules[0].Count
					if count == nil || count.Certain != test.bounds[i][0] || count.Uncertain != test.bounds[i][1] {
						t.Fatalf("expected occurrence bounds %v, got %+v", test.bounds[i], count)
					}
					switch truth {
					case "MATCH":
						totals[0]++
						expectedProfiles = append(expectedProfiles, profile)
					case "NO_MATCH":
						totals[1]++
					case "UNKNOWN":
						totals[2]++
					default:
						t.Fatalf("expected ternary fixture result, got %s", truth)
					}
				}
				if page.Total != len(expectedProfiles) || len(page.Profiles) != len(expectedProfiles) ||
					page.Evaluation == nil || page.Evaluation.Match != totals[0] ||
					page.Evaluation.NoMatch != totals[1] || page.Evaluation.Unknown != totals[2] {
					t.Fatalf("expected totals %v and profiles %v, got %+v", totals, expectedProfiles, page)
				}
				for i, profile := range page.Profiles {
					if profile.KPID != expectedProfiles[i] {
						t.Fatalf("expected profile %s at %d, got %+v", expectedProfiles[i], i, profile)
					}
				}
				query.First = 99
				empty, err := workspace.QueryProfiles(t.Context(), query)
				if err != nil {
					t.Fatalf("expected empty page with complete totals, got %v", err)
				}
				if len(empty.Profiles) != 0 || empty.Total != page.Total || *empty.Evaluation != *page.Evaluation {
					t.Fatalf("expected unchanged totals on empty page, got %+v", empty)
				}
				after, err := json.Marshal(filter)
				if err != nil {
					t.Fatalf("expected filter encoding after execution, got %v", err)
				}
				if string(before) != string(after) {
					t.Fatalf("expected unchanged source document %s, got %s", before, after)
				}
			})
		}

		condition := &FilterCountCondition{Count: FilterCount{
			Source: FilterCountSource{Kind: "recordedEvents", Type: "page",
				Name: "Pricing", Grain: "registered-message"}},
			Operator: OpIs, Values: filterValues(0)}
		filter := &Filter{Operator: OpAnd, Rules: []FilterRule{condition},
			DefaultCalendar: &FilterCalendar{Definition: "gregorian-monday/UTC", MeaningRevision: "1"}}
		for _, period := range []FilterPeriod{
			{Kind: "lastCompleteCalendarPeriods", Unit: "month", Count: json.Value(`1`)},
			{Kind: "fullRelevantHistory", History: &FilterHistory{Definition: "registered-queryable-events", MeaningRevision: "1"}},
		} {
			condition.Count.Period = period
			page, err := workspace.QueryProfiles(t.Context(), ProfileQuery{Filter: filter,
				At: at.Format(time.RFC3339), Limit: 10})
			if err != nil {
				t.Fatalf("expected page Count with period %s, got %v", period.Kind, err)
			}
			if page.Evaluation.Match != 2 || page.Evaluation.NoMatch != 2 || page.Evaluation.Unknown != 0 {
				t.Fatalf("expected named page totals 2/2/0 with %s, got %+v", period.Kind, page)
			}
		}
		condition.Count.Source.Name = ""
		condition.Count.Where = &Filter{Operator: OpAnd, Rules: []FilterRule{
			&FilterCondition{Property: "context.page.path", Operator: OpIs, Values: filterValues("/pricing")}}}
		condition.Count.Minimum = json.Value(`1`)
		condition.Operator, condition.Values = OpIsNot, filterValues(1)
		explained, err := workspace.ExplainProfileFilter(t.Context(), "00000000-0000-0000-0000-000000000003",
			filter, at.Format(time.RFC3339))
		if err != nil {
			t.Fatalf("expected page sample guard explanation, got %v", err)
		}
		count := explained.Rule.Rules[0].Count
		if explained.Rule.Result != "UNKNOWN" || count == nil || count.Certain != 0 || count.Uncertain != 1 {
			t.Fatalf("expected separate Kleene guard on page count [0,1], got %+v", explained)
		}

	})

	t.Run("public Count currency", func(t *testing.T) {

		_, err := database.Exec(t.Context(), `INSERT INTO unified_v.krenalis_events VALUES
('00000000-0000-0000-0000-000000000002','native-count/2','Order Completed','2026-01-15',
'{"currency":"EUR","coupon":"PROMO","total":20}'),
('00000000-0000-0000-0000-000000000003','native-count/3','Order Completed','2026-01-15',
'{"currency":null,"coupon":"PROMO"}'),
('00000000-0000-0000-0000-000000000004','native-count/4','Order Completed','2026-01-15',
'{"coupon":"OTHER"}')`)
		if err != nil {
			t.Fatalf("expected governed currency fixtures, got %v", err)
		}
		defer func() {
			_, err := database.Exec(t.Context(), "DELETE FROM unified_v.krenalis_events WHERE message_id LIKE 'native-count/%'")
			if err != nil {
				t.Errorf("expected currency fixture cleanup, got %v", err)
			}
		}()
		var stored []byte
		err = database.QueryRow(t.Context(), "SELECT $1::jsonb",
			`{"operator":"and","rules":[`+currencyCountConditionJSON+`]}`).Scan(&stored)
		if err != nil {
			t.Fatalf("expected currency reference JSONB round-trip, got %v", err)
		}
		var filter Filter
		err = json.Unmarshal(stored, &filter)
		if err != nil {
			t.Fatalf("expected decoded reference after JSONB, got %v", err)
		}
		condition := filter.Rules[0].(*FilterCountCondition)
		for _, test := range []struct {
			name, payload, truth string
			certain, uncertain   int64
			period               FilterPeriod
		}{
			{"JSON null", `{"currency":null}`, "UNKNOWN", 0, 1, condition.Count.Period},
			{"missing", `{}`, "UNKNOWN", 0, 1, condition.Count.Period},
			{"Number", `{"currency":100}`, "UNKNOWN", 0, 1, condition.Count.Period},
			{"Boolean", `{"currency":true}`, "UNKNOWN", 0, 1, condition.Count.Period},
			{"Object", `{"currency":{}}`, "UNKNOWN", 0, 1, condition.Count.Period},
			{"Array", `{"currency":[]}`, "UNKNOWN", 0, 1, condition.Count.Period},
			{"lowercase", `{"currency":"eur"}`, "MATCH", 0, 0, condition.Count.Period},
			{"empty String", `{"currency":""}`, "MATCH", 0, 0, condition.Count.Period},
			{"USD", `{"currency":"USD"}`, "MATCH", 0, 0, condition.Count.Period},
			{"EUR", `{"currency":"EUR"}`, "NO_MATCH", 1, 0, condition.Count.Period},
			{"calendar", `{"currency":null}`, "UNKNOWN", 0, 1, FilterPeriod{Kind: "lastCompleteCalendarPeriods", Unit: "month", Count: json.Value(`1`)}},
			{"recorded history", `{"currency":null}`, "UNKNOWN", 0, 1, FilterPeriod{Kind: "fullRelevantHistory", History: &FilterHistory{Definition: "registered-queryable-events", MeaningRevision: "1"}}},
		} {
			t.Run(test.name, func(t *testing.T) {
				_, err := database.Exec(t.Context(), "UPDATE unified_v.krenalis_events SET properties=$1::jsonb "+
					"WHERE message_id='native-count/3'", test.payload)
				if err != nil {
					t.Fatalf("expected currency fixture update, got %v", err)
				}
				condition.Count.Period = test.period
				filter.DefaultCalendar = &FilterCalendar{Definition: "gregorian-monday/UTC", MeaningRevision: "1"}
				query := ProfileQuery{Filter: &filter, At: at.Format(time.RFC3339),
					Properties: []string{"profile_id"}, Order: "profile_id", Limit: 10}
				page, err := workspace.QueryProfiles(t.Context(), query)
				if err != nil {
					t.Fatalf("expected public governed Count listing, got %v", err)
				}
				truths := []string{"MATCH", "NO_MATCH", test.truth, "UNKNOWN"}
				totals := [3]int64{}
				for i, truth := range truths {
					profile := fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1)
					explained, err := workspace.ExplainProfileFilter(t.Context(), profile, &filter, query.At)
					if err != nil {
						t.Fatalf("expected public currency explanation, got %v", err)
					}
					if explained.Rule.Result != truth || len(explained.Rule.Rules) != 1 {
						t.Fatalf("expected profile %s to be %s, got %+v", profile, truth, explained)
					}
					bounds := explained.Rule.Rules[0].Count
					if bounds == nil || bounds.CurrencyField == nil || bounds.NativeCurrency != "EUR" ||
						bounds.CurrencyField.Definition != "commerce.order_completed.currency" ||
						bounds.CurrencyField.MeaningRevision != "1" {
						t.Fatalf("expected measured currency reference, got %+v", bounds)
					}
					if i == 2 && (bounds.Certain != test.certain || bounds.Uncertain != test.uncertain) {
						t.Fatalf("expected currency count %d/%d, got %+v", test.certain, test.uncertain, bounds)
					}
					switch truth {
					case "MATCH":
						totals[0]++
					case "NO_MATCH":
						totals[1]++
					case "UNKNOWN":
						totals[2]++
					default:
						t.Fatalf("expected a ternary fixture result, got %s", truth)
					}
				}
				if page.Evaluation == nil || page.Total != int(totals[0]) || len(page.Profiles) != page.Total ||
					page.Evaluation.Match != totals[0] || page.Evaluation.NoMatch != totals[1] ||
					page.Evaluation.Unknown != totals[2] {
					t.Fatalf("expected currency totals %v, got %+v", totals, page)
				}
				query.First = 99
				empty, err := workspace.QueryProfiles(t.Context(), query)
				if err != nil {
					t.Fatalf("expected empty currency page, got %v", err)
				}
				if len(empty.Profiles) != 0 || empty.Total != page.Total || *empty.Evaluation != *page.Evaluation {
					t.Fatalf("expected unchanged totals on empty page, got %+v", empty)
				}
			})
		}

		raw := *condition
		raw.Count.CurrencyField, raw.Count.NativeCurrency = nil, nil
		raw.Count.Where = &Filter{Operator: OpAnd, Rules: []FilterRule{
			&FilterCondition{Property: "properties.currency", Operator: OpIs, Values: filterValues("EUR")}}}
		rawFilter := &Filter{Operator: OpAnd, Rules: []FilterRule{&raw}}
		rawPage, err := workspace.QueryProfiles(t.Context(), ProfileQuery{Filter: rawFilter,
			At: at.Format(time.RFC3339), Limit: 10})
		if err != nil {
			t.Fatalf("expected distinct raw currency query, got %v", err)
		}
		if rawPage.Evaluation.Match != 2 || rawPage.Evaluation.NoMatch != 1 || rawPage.Evaluation.Unknown != 1 {
			t.Fatalf("expected raw JSON null FALSE and missing UNKNOWN, got %+v", rawPage)
		}

		// A false raw qualification excludes an unknown currency. A missing raw
		// coupon remains uncertain; it does not mean that no coupon was used.
		_, err = database.Exec(t.Context(), `UPDATE unified_v.krenalis_events
SET properties='{"currency":null,"coupon":"OTHER"}' WHERE message_id='native-count/3'`)
		if err != nil {
			t.Fatalf("expected independent qualification fixture, got %v", err)
		}
		condition.Count.Where = &Filter{Operator: OpAnd, Rules: []FilterRule{
			&FilterCondition{Property: "properties.coupon", Operator: OpIs, Values: filterValues("PROMO")}}}
		query := ProfileQuery{Filter: &filter, At: at.Format(time.RFC3339), Limit: 10}
		page, err := workspace.QueryProfiles(t.Context(), query)
		if err != nil {
			t.Fatalf("expected mixed independent qualification, got %v", err)
		}
		if page.Evaluation.Match != 3 || page.Evaluation.NoMatch != 1 || page.Evaluation.Unknown != 0 {
			t.Fatalf("expected false where to exclude unknown currencies, got %+v", page)
		}
		_, err = database.Exec(t.Context(), `INSERT INTO unified_v.krenalis_events VALUES
('00000000-0000-0000-0000-000000000002','native-count/optional-1','Order Completed','2026-01-15','{}'),
('00000000-0000-0000-0000-000000000002','native-count/optional-2','Order Completed','2026-01-15','{}')`)
		if err != nil {
			t.Fatalf("expected independent optional currency occurrences, got %v", err)
		}
		condition.Count.Where = nil
		condition.Count.Minimum = json.Value(`3`)
		condition.Operator, condition.Values = OpIsNot, filterValues(3)
		explained, err := workspace.ExplainProfileFilter(t.Context(), "00000000-0000-0000-0000-000000000002",
			&filter, query.At)
		if err != nil {
			t.Fatalf("expected currency sample guard explanation, got %v", err)
		}
		bounds := explained.Rule.Rules[0].Count
		if explained.Rule.Result != "UNKNOWN" || bounds == nil || bounds.Certain != 1 || bounds.Uncertain != 2 {
			t.Fatalf("expected separate Kleene guard over [1,3], got %+v", explained)
		}

		unverifiedStore, ok := ds.Store("222222222222")
		if !ok {
			t.Fatal("expected unverified store, got none")
		}
		unverifiedState, ok := st.Workspace("222222222222")
		if !ok {
			t.Fatal("expected unverified state, got none")
		}
		unverified := &Workspace{core: &Core{}, store: unverifiedStore, workspace: unverifiedState}
		page, err = unverified.QueryProfiles(t.Context(), query)
		if err != nil {
			failure, ok := errors.AsType[*errors.UnprocessableError](err)
			if !ok || failure.Code != UnifiedFilteringUnavailable || page != nil {
				t.Fatalf("expected currency request blocked on unverified backend, got %+v, %v", page, err)
			}
		}
		if err == nil {
			t.Fatal("expected unavailable backend, got success")
		}

	})

	t.Run("public Sum", func(t *testing.T) {

		_, err := database.Exec(t.Context(), `INSERT INTO unified_v.krenalis_events VALUES
('00000000-0000-0000-0000-000000000001','sum1','Order Completed','2026-01-15','{"total":20,"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000001','sum2','Order Completed','2026-01-16','{"total":15,"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000002','sum3','Order Completed','2026-01-15','{"total":20,"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000002','sum4','Order Completed','2026-01-15','{"currency":"USD"}'),
('00000000-0000-0000-0000-000000000003','sum5','Order Completed','2026-01-15','{"total":20,"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000003','sum6','Order Completed','2026-01-15','{"total":20}')`)
		if err != nil {
			t.Fatalf("expected coherent monetary fixtures, got %v", err)
		}

		var filter Filter
		err = json.Unmarshal([]byte(`{"operator":"and","rules":[`+sumConditionJSON+`]}`), &filter)
		if err != nil {
			t.Fatalf("expected public Sum decoding, got %v", err)
		}
		condition := filter.Rules[0].(*FilterSumCondition)
		calendar := &FilterCalendar{Definition: "gregorian-monday/UTC", MeaningRevision: "1"}
		for _, period := range []FilterPeriod{
			{Kind: "elapsedLookback", Duration: &FilterDuration{Value: json.Value(`31`), Unit: "day"}},
			{Kind: "lastCompleteCalendarPeriods", Unit: "month", Count: json.Value(`1`)},
			{Kind: "fullRelevantHistory", History: &FilterHistory{Definition: "registered-queryable-events", MeaningRevision: "1"}},
		} {
			condition.Sum.Period = period
			filter.DefaultCalendar = calendar
			query := ProfileQuery{Filter: &filter, At: at.Format(time.RFC3339), Properties: []string{"profile_id"}, Limit: 10}
			result, err := workspace.QueryProfiles(t.Context(), query)
			if err != nil {
				t.Fatalf("expected public Sum execution, got %v", err)
			}
			if result.Total != 1 || len(result.Profiles) != 1 || result.Evaluation.Match != 1 ||
				result.Evaluation.NoMatch != 2 || result.Evaluation.Unknown != 1 ||
				result.Profiles[0].Attributes["profile_id"] != "customer-1" {
				t.Fatalf("expected one Sum member and totals 1/2/1, got %+v", result)
			}
			for i, expected := range []struct {
				truth, lower, upper string
				certain, uncertain  int64
			}{
				{"MATCH", "35", "35", 2, 0},
				{"NO_MATCH", "20", "20", 1, 0},
				{"UNKNOWN", "20", "40", 1, 1},
				{"NO_MATCH", "0", "0", 0, 0},
			} {
				profile := fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1)
				explained, err := workspace.ExplainProfileFilter(t.Context(), profile, &filter, query.At)
				if err != nil {
					t.Fatalf("expected public Sum explanation, got %v", err)
				}
				value := explained.Rule.Rules[0].Sum
				if explained.Rule.Result != expected.truth || value == nil || value.NativeCurrency != "EUR" ||
					value.Field.Definition != "commerce.order_completed.total" || value.Field.MeaningRevision != "1" ||
					value.Bounds.Lower == nil || value.Bounds.Upper == nil ||
					!value.Bounds.Lower.Equal(decimal.MustParse(expected.lower)) ||
					!value.Bounds.Upper.Equal(decimal.MustParse(expected.upper)) || !value.End.Equal(at) {
					t.Fatalf("expected %s Sum bounds [%s,%s] and resolved metadata, got %+v",
						expected.truth, expected.lower, expected.upper, value)
				}
				if value.Certain != expected.certain || value.Uncertain != expected.uncertain {
					t.Fatalf("expected %d certain and %d uncertain Sum occurrences, got %+v",
						expected.certain, expected.uncertain, value)
				}
				if period.Kind == "fullRelevantHistory" && (!value.UnboundedStart || !value.Start.IsZero() ||
					value.History == nil || value.History.Definition != "registered-queryable-events" ||
					value.History.MeaningRevision != "1") {
					t.Fatalf("expected full history metadata, got %+v", value)
				}
			}
		}

		// The same explanation transports both numeric Sum and integer Count bounds.
		filter.DefaultCalendar = nil
		mixed := &Filter{Operator: OpOr, DefaultCalendar: calendar,
			Rules: []FilterRule{&filter,
				&FilterCondition{Property: "country", Operator: OpIs, Values: filterValues("IT")}, count}}
		profile := "00000000-0000-0000-0000-000000000003"
		explained, err := workspace.ExplainProfileFilter(t.Context(), profile, mixed, at.Format(time.RFC3339))
		if err != nil {
			t.Fatalf("expected mixed Record/Count/Sum explanation, got %v", err)
		}
		if explained.Rule.Result != "MATCH" || explained.Rule.Rules[0].Rules[0].Sum == nil ||
			explained.Rule.Rules[1].Kind != "record" || explained.Rule.Rules[2].Count == nil {
			t.Fatalf("expected measured mixed-tree outcomes, got %+v", explained)
		}
		unverifiedStore, ok := ds.Store("222222222222")
		if !ok {
			t.Fatal("expected unverified store, got none")
		}
		unverifiedState, ok := st.Workspace("222222222222")
		if !ok {
			t.Fatal("expected unverified workspace, got none")
		}
		unverified := &Workspace{core: &Core{}, store: unverifiedStore, workspace: unverifiedState}
		query := ProfileQuery{Filter: mixed, At: at.Format(time.RFC3339), Limit: 1}
		unavailable, err := unverified.QueryProfiles(t.Context(), query)
		if err != nil {
			failure, ok := errors.AsType[*errors.UnprocessableError](err)
			if !ok || failure.Code != UnifiedFilteringUnavailable || unavailable != nil {
				t.Fatalf("expected gated Sum backend without result, got %+v, %v", unavailable, err)
			}
		}
		if err == nil {
			t.Fatal("expected unavailable Sum backend, got success")
		}

	})

	t.Run("complete calendar periods", func(t *testing.T) {

		_, err := database.Exec(t.Context(), `INSERT INTO unified_v.krenalis_events VALUES
('00000000-0000-0000-0000-000000000001','calendar-start','Calendar','2026-09-30 22:00:00','{}'),
('00000000-0000-0000-0000-000000000002','calendar-before-end','Calendar','2026-10-31 22:59:59.999','{}'),
('00000000-0000-0000-0000-000000000003','calendar-end','Calendar','2026-10-31 23:00:00','{}'),
('00000000-0000-0000-0000-000000000004','calendar-utc-end','Calendar','2026-11-01 00:00:00','{}'),
('00000000-0000-0000-0000-000000000004','calendar-before-start','Calendar','2026-09-30 21:59:59.999','{}')`)
		if err != nil {
			t.Fatalf("expected calendar boundary fixtures, got %v", err)
		}

		for _, test := range []struct {
			zone, unit string
			count      json.Value
			ids        []string
		}{
			{"Europe/Rome", "month", json.Value(`1`), []string{"1", "2"}},
			{"UTC", "month", json.Value(`1`), []string{"2", "3"}},
			{"Europe/Rome", "week", json.Value(`1`), []string{"2", "3", "4"}},
			{"Europe/Rome", "day", json.Value(`1`), []string{"3", "4"}},
			{"Europe/Rome", "month", json.Value(`22`), []string{"1", "2", "4"}},
		} {
			t.Run(test.zone+"/"+test.unit+"/"+string(test.count), func(t *testing.T) {

				filter := Filter{Operator: OpAnd,
					DefaultCalendar: &FilterCalendar{Definition: "gregorian-monday/" + test.zone, MeaningRevision: "1"},
					Rules: []FilterRule{&FilterCountCondition{Count: FilterCount{
						Source: FilterCountSource{Kind: "recordedEvents", Name: "Calendar", Grain: "registered-message"},
						Period: FilterPeriod{Kind: "lastCompleteCalendarPeriods", Unit: test.unit, Count: test.count},
					}, Operator: OpIsGreaterThanOrEqualTo, Values: filterValues(1)}}}
				before, err := json.Marshal(&filter)
				if err != nil {
					t.Fatalf("expected calendar JSON encoding, got %v", err)
				}
				query := ProfileQuery{Filter: &filter, At: "2026-11-02T12:00:00Z", Properties: []string{"profile_id"},
					Order: "profile_id", Limit: 10}
				result, err := workspace.QueryProfiles(t.Context(), query)
				if err != nil {
					t.Fatalf("expected calendar query through core and warehouse, got %v", err)
				}
				if result.Total != len(test.ids) || len(result.Profiles) != len(test.ids) ||
					result.Evaluation.Match != int64(len(test.ids)) || result.Evaluation.NoMatch != int64(4-len(test.ids)) ||
					result.Evaluation.Unknown != 0 || result.Evaluation.At.Format(time.RFC3339) != query.At {
					t.Fatalf("expected %d calendar matches and complete totals, got %+v", len(test.ids), result)
				}
				for i, id := range test.ids {
					if result.Profiles[i].Attributes["profile_id"] != "customer-"+id {
						t.Fatalf("expected calendar member customer-%s, got %+v", id, result.Profiles[i])
					}
				}
				if test.zone == "Europe/Rome" && test.unit == "month" && string(test.count) == "1" {
					explained, err := workspace.ExplainProfileFilter(t.Context(), result.Profiles[0].KPID, &filter, query.At)
					if err != nil {
						t.Fatalf("expected calendar explanation, got %v", err)
					}
					period := explained.Rule.Rules[0].Count
					if period == nil || period.Start.Format(time.RFC3339) != "2026-09-30T22:00:00Z" ||
						period.End.Format(time.RFC3339) != "2026-10-31T23:00:00Z" || period.Certain != 1 ||
						period.Uncertain != 0 || explained.Rule.Result != "MATCH" {
						t.Fatalf("expected resolved Rome October period and one certain occurrence, got %+v", period)
					}
				}

				query.First = 99
				empty, err := workspace.QueryProfiles(t.Context(), query)
				if err != nil {
					t.Fatalf("expected empty calendar page, got %v", err)
				}
				if len(empty.Profiles) != 0 || empty.Total != result.Total || *empty.Evaluation != *result.Evaluation {
					t.Fatalf("expected empty page with unchanged calendar totals, got %+v", empty)
				}
				after, err := json.Marshal(&filter)
				if err != nil {
					t.Fatalf("expected calendar encoding after execution, got %v", err)
				}
				if string(before) != string(after) {
					t.Fatalf("expected preserved relative filter %s, got %s", before, after)
				}

			})
		}

	})

	for _, test := range []struct {
		op      FilterOperator
		values  []json.Value
		minimum json.Value
		want    []unified.Truth
	}{
		{OpIs, filterValues(0), nil, []unified.Truth{unified.Match, unified.NoMatch, unified.Match, unified.Unknown}},
		{OpIsNot, filterValues(0), nil, []unified.Truth{unified.NoMatch, unified.Match, unified.NoMatch, unified.Unknown}},
		{OpIsLessThan, filterValues(1), nil, []unified.Truth{unified.Match, unified.NoMatch, unified.Match, unified.Unknown}},
		{OpIsLessThanOrEqualTo, filterValues(0), nil, []unified.Truth{unified.Match, unified.NoMatch, unified.Match, unified.Unknown}},
		{OpIsGreaterThan, filterValues(0), nil, []unified.Truth{unified.NoMatch, unified.Match, unified.NoMatch, unified.Unknown}},
		{OpIsGreaterThanOrEqualTo, filterValues(1), nil, []unified.Truth{unified.NoMatch, unified.Match, unified.NoMatch, unified.Unknown}},
		{OpIsBetween, filterValues(0, 1), nil, []unified.Truth{unified.Match, unified.Match, unified.Match, unified.Match}},
		{OpIsGreaterThanOrEqualTo, filterValues(0), json.Value(`1`), []unified.Truth{unified.NoMatch, unified.Match, unified.NoMatch, unified.Unknown}},
		{OpIsNot, filterValues(1), json.Value(`1`), []unified.Truth{unified.NoMatch, unified.NoMatch, unified.NoMatch, unified.Unknown}},
	} {
		t.Run(fmt.Sprintf("public %s minimum %s", test.op, test.minimum), func(t *testing.T) {
			c := *count
			c.Operator, c.Values, c.Count.Minimum = test.op, test.values, test.minimum
			input := &Filter{Operator: OpAnd, Rules: []FilterRule{&c}}
			encoded, err := json.Marshal(input)
			if err != nil {
				t.Fatalf("expected public Count encoding, got %v", err)
			}
			var restored Filter
			err = json.Unmarshal(encoded, &restored)
			if err != nil {
				t.Fatalf("expected restored Count document, got %v", err)
			}
			result, err := workspace.unifiedProfiles(t.Context(), unifiedQuery{
				Root: &restored, At: at, Page: unified.Page{Limit: 10},
			})
			if err != nil {
				t.Fatalf("expected complete public Count execution, got %v", err)
			}
			if len(result.Profiles) != len(test.want) {
				t.Fatalf("expected %d profile outcomes, got %+v", len(test.want), result)
			}
			for i, profile := range result.Profiles {
				if profile.Truth != test.want[i] {
					t.Fatalf("expected public Count profile %d truth %d, got %+v", i, test.want[i], profile)
				}
			}
		})
	}

	profile := &Filter{Operator: OpAnd, Rules: []FilterRule{
		&FilterCondition{Property: "country", Operator: OpIs, Values: filterValues("IT")},
	}}
	for _, test := range []struct {
		op   FilterLogical
		want []unified.Truth
	}{
		{OpAnd, []unified.Truth{unified.Match, unified.NoMatch, unified.Unknown, unified.Unknown}},
		{OpOr, []unified.Truth{unified.Match, unified.NoMatch, unified.Match, unified.Match}},
	} {
		t.Run(string(test.op), func(t *testing.T) {
			result, err := workspace.unifiedProfiles(t.Context(), unifiedQuery{
				Root: &Filter{Operator: test.op, Rules: []FilterRule{profile.Rules[0], count}},
				At:   at, Page: unified.Page{Limit: 10},
			})
			if err != nil {
				t.Fatalf("expected composed service query, got %v", err)
			}
			if len(result.Profiles) != len(test.want) {
				t.Fatalf("expected four profiles, got %+v", result)
			}
			for i, outcome := range result.Profiles {
				if outcome.Truth != test.want[i] {
					t.Fatalf("expected profile %d truth %d, got %+v", i, test.want[i], outcome)
				}
			}
		})
	}
	deep := &Filter{Operator: OpAnd, Rules: []FilterRule{profile.Rules[0], count}}
	for range 3 {
		deep = &Filter{Operator: OpAnd, Rules: []FilterRule{deep}}
	}
	wide := &Filter{Operator: OpAnd, Rules: make([]FilterRule, 100)}
	for i := range 99 {
		wide.Rules[i] = profile.Rules[0]
	}
	wide.Rules[99] = count
	for _, test := range []struct {
		name string
		root *Filter
	}{
		{"four mixed levels", deep},
		{"100 mixed descendants", wide},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := workspace.unifiedProfiles(t.Context(), unifiedQuery{
				Root: test.root, At: at, Page: unified.Page{Limit: 10},
			})
			if err != nil {
				t.Fatalf("expected complete mixed query at the structural limit, got %v", err)
			}
			expected := []unified.Truth{unified.Match, unified.NoMatch, unified.Unknown, unified.Unknown}
			if len(result.Profiles) != len(expected) || result.Match != 1 || result.NoMatch != 1 || result.Unknown != 2 {
				t.Fatalf("expected four profiles and totals 1/1/2, got %+v", result)
			}
			for i, outcome := range result.Profiles {
				if outcome.Truth != expected[i] {
					t.Fatalf("expected profile %d truth %d, got %+v", i, expected[i], outcome)
				}
			}
		})
	}
	result, err = workspace.unifiedProfiles(t.Context(), unifiedQuery{
		Root: profile, At: at, Page: unified.Page{Limit: 10, MatchingOnly: true},
	})
	if err != nil {
		t.Fatalf("expected pure Record service query, got %v", err)
	}
	if len(result.Profiles) != 2 || result.Match != 2 || result.NoMatch != 1 || result.Unknown != 1 {
		t.Fatalf("expected pure Record totals 2/1/1 and two members, got %+v", result)
	}

	// The public service entry projects attributes and orders the membership
	// page in the same SQL evaluation as all three totals.
	t.Run("projected QueryProfiles", func(t *testing.T) {

		query := ProfileQuery{Filter: &document, Order: "country", Limit: 10, At: at.Format(time.RFC3339Nano)}
		result, err := workspace.QueryProfiles(t.Context(), query)
		if err != nil {
			t.Fatalf("expected projected service result, got %v", err)
		}
		if result.Total != 2 || result.Evaluation == nil || result.Evaluation.Match != 2 ||
			result.Evaluation.NoMatch != 1 || result.Evaluation.Unknown != 1 || !result.Evaluation.At.Equal(at) {
			t.Fatalf("expected exact totals 2/1/1 at T, got %+v", result)
		}
		if len(result.Profiles) != 2 || result.Profiles[0].KPID != "00000000-0000-0000-0000-000000000001" ||
			result.Profiles[1].KPID != "00000000-0000-0000-0000-000000000003" {
			t.Fatalf("expected IT profile then NULL-country profile, got %+v", result.Profiles)
		}
		for _, profile := range result.Profiles {
			encoded, err := types.Marshal(profile.Attributes, result.Schema)
			if err != nil {
				t.Fatalf("expected typed projected attributes, got %v", err)
			}
			if !strings.Contains(string(encoded), `"amount":9007199254740993.0000000001`) ||
				!strings.Contains(string(encoded), `"settings":{"value":"nested"}`) ||
				!strings.Contains(string(encoded), `"document":{"currency":null}`) ||
				!reflect.DeepEqual(profile.Attributes["tags"], []any{"first", "second"}) ||
				profile.UpdatedAt.Format(time.RFC3339Nano) != "2026-01-31T12:00:00.123456Z" {
				t.Fatalf("expected exact decimal, JSON, nested, array and timestamp values, got %s / %s",
					encoded, profile.UpdatedAt)
			}
		}

		nulls := query
		nulls.OrderDesc = true
		nullPage, err := workspace.QueryProfiles(t.Context(), nulls)
		if err != nil {
			t.Fatalf("expected descending nullable order, got %v", err)
		}
		if len(nullPage.Profiles) != 2 || nullPage.Profiles[0].KPID != result.Profiles[0].KPID {
			t.Fatalf("expected NULLS LAST also in descending order, got %+v", nullPage)
		}
		for _, desc := range []bool{false, true} {
			ordered := query
			ordered.Properties, ordered.Order, ordered.OrderDesc = []string{"event"}, "profile_id", desc
			got, err := workspace.QueryProfiles(t.Context(), ordered)
			if err != nil {
				t.Fatalf("expected scalar ordering outside the projection, got %v", err)
			}
			wantFirst := "00000000-0000-0000-0000-000000000001"
			if desc {
				wantFirst = "00000000-0000-0000-0000-000000000003"
			}
			if len(got.Profiles) != 2 || got.Profiles[0].KPID != wantFirst || len(got.Profiles[0].Attributes) != 1 {
				t.Fatalf("expected requested scalar order desc=%t, got %+v", desc, got)
			}
		}

		// Equal update timestamps need a UUID tie-break in either direction.
		for _, desc := range []bool{false, true} {
			for i, id := range []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000003"} {
				page := query
				page.Order, page.OrderDesc, page.First, page.Limit = "", desc, i, 1
				page.Properties = []string{"profile_id", "event"}
				got, err := workspace.QueryProfiles(t.Context(), page)
				if err != nil {
					t.Fatalf("expected ordered projected page, got %v", err)
				}
				if len(got.Profiles) != 1 || got.Profiles[0].KPID != id || len(got.Profiles[0].Attributes) != 2 {
					t.Fatalf("expected stable UUID %s and requested projection, got %+v", id, got)
				}
			}
		}
		query.First = 99
		empty, err := workspace.QueryProfiles(t.Context(), query)
		if err != nil {
			t.Fatalf("expected empty page with totals, got %v", err)
		}
		if len(empty.Profiles) != 0 || empty.Total != 2 || *empty.Evaluation != *result.Evaluation {
			t.Fatalf("expected empty page and unchanged evaluation totals, got %+v", empty)
		}
		query.First, query.At = 0, ""
		current, err := workspace.QueryProfiles(t.Context(), query)
		if err != nil {
			t.Fatalf("expected automatic T, got %v", err)
		}
		if current.Evaluation.At.After(time.Now()) || current.Evaluation.At.Nanosecond()%1000 != 0 {
			t.Fatalf("expected fixed nonfuture microsecond T, got %s", current.Evaluation.At)
		}

		for _, filter := range []*Filter{nil, profile} {
			legacy, legacySchema, total, err := workspace.Profiles(t.Context(), []string{"country"},
				filter, "country", false, 0, 10)
			if err != nil {
				t.Fatalf("expected legacy Record result, got %v", err)
			}
			got, err := workspace.QueryProfiles(t.Context(), ProfileQuery{Properties: []string{"country"},
				Filter: filter, Order: "country", Limit: 10})
			if err != nil {
				t.Fatalf("expected delegated Record query, got %v", err)
			}
			if got.Evaluation != nil || got.Total != total || !types.Equal(got.Schema, legacySchema) ||
				!reflect.DeepEqual(got.Profiles, legacy) {
				t.Fatalf("expected identical legacy Record result without metadata, got %+v", got)
			}
		}

		// A payload exceeding the page budget fails before the public result is
		// assembled. It cannot become missing data or a partial success.
		_, err = database.Exec(t.Context(), `UPDATE unified_v.profiles SET document=to_jsonb(repeat('x',1048577))`)
		if err != nil {
			t.Fatalf("expected oversized fixture publication, got %v", err)
		}
		failed, err := workspace.QueryProfiles(t.Context(), query)
		if err != nil {
			failure, ok := errors.AsType[*errors.UnprocessableError](err)
			if !ok || failure.Code != ProfileResultTooLarge {
				t.Fatalf("expected Unprocessable ProfileResultTooLarge, got %T: %v", err, err)
			}
		}
		if err == nil || failed != nil {
			t.Fatalf("expected complete failure without partial page, got %+v, %v", failed, err)
		}

	})

}
