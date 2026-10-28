// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/initdb"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/kms"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
	"github.com/krenalis/krenalis/warehouses/postgresql"

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
CREATE TABLE unified_v.profiles (_kpid uuid PRIMARY KEY,country text,profile_id text,event text);
CREATE TABLE unified_v.krenalis_events (kpid uuid,message_id text PRIMARY KEY,
 event text,timestamp timestamp(3) NOT NULL,properties jsonb);
INSERT INTO unified_v.profiles (_kpid)
 SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid FROM generate_series(1,4) n;
UPDATE unified_v.profiles SET event='Refund',profile_id='customer-'||right(_kpid::text,1),
 country=CASE right(_kpid::text,1) WHEN '1' THEN 'IT' WHEN '2' THEN 'FR' WHEN '4' THEN 'IT' END;
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
		{Name: "email", Type: types.String(), ReadOptional: true},
		{Name: "country", Type: types.String(), ReadOptional: true},
		{Name: "profile_id", Type: types.String(), ReadOptional: true},
		{Name: "event", Type: types.String(), ReadOptional: true},
	}).MarshalJSON()
	if err != nil {
		t.Fatalf("expected profile schema encoding, got %v", err)
	}
	_, err = database.Exec(t.Context(), `INSERT INTO workspaces
 (id,organization,name,warehouse_name,warehouse_mode,warehouse_settings,
 kms_encrypted_warehouse_settings_key,kms_encrypted_warehouse_mcp_settings_key,profile_schema,identifiers)
 SELECT '111111111111',id,'Unified Count',$1,'Normal','','','',$2,ARRAY[]::text[] FROM organizations`, platform, schema)
	if err != nil {
		t.Fatalf("expected workspace state fixture, got %v", err)
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
	var filter Filter
	err = json.Unmarshal([]byte(`{"operator":"and","rules":[`+
		`{"property":"properties.currency","operator":"is","values":["EUR"]}]}`), &filter)
	if err != nil {
		t.Fatalf("expected existing public Record filter decoding, got %v", err)
	}
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	count := unifiedRule{Count: &unifiedCount{
		Scope: unified.Scope{Event: "Purchase", Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at},
		Where: &filter, Test: unified.CountTest{Operator: unified.Equal},
	}}
	result, err := workspace.unifiedProfiles(t.Context(), unifiedQuery{
		Root: &count, At: at, Page: unified.Page{Limit: 10},
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

	for _, period := range []unified.ElapsedPeriod{
		{Older: 31 * 24 * time.Hour},
		{Older: 31 * 24 * time.Hour, Newer: 14 * 24 * time.Hour},
	} {
		t.Run(fmt.Sprintf("elapsed %s to %s", period.Older, period.Newer), func(t *testing.T) {
			c := *count.Count
			c.Scope = unified.Scope{Event: "Purchase", Grain: "registered-message", Elapsed: &period}
			root := unifiedRule{Count: &c}
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
				Root: &unifiedRule{Operator: test.op, Rules: []unifiedRule{{Profile: profile.Rules[0]}, count}},
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
	deep := unifiedRule{Operator: OpAnd, Rules: []unifiedRule{{Profile: profile.Rules[0]}, count}}
	for range 3 {
		deep = unifiedRule{Operator: OpAnd, Rules: []unifiedRule{deep}}
	}
	wide := unifiedRule{Operator: OpAnd, Rules: make([]unifiedRule, 100)}
	for i := range 99 {
		wide.Rules[i] = unifiedRule{Profile: profile.Rules[0]}
	}
	wide.Rules[99] = count
	for _, test := range []struct {
		name string
		root unifiedRule
	}{
		{"four mixed levels", deep},
		{"100 mixed descendants", wide},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := workspace.unifiedProfiles(t.Context(), unifiedQuery{
				Root: &test.root, At: at, Page: unified.Page{Limit: 10},
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
		Root: &unifiedRule{Profile: profile}, At: at, Page: unified.Page{Limit: 10, MatchingOnly: true},
	})
	if err != nil {
		t.Fatalf("expected pure Record service query, got %v", err)
	}
	if len(result.Profiles) != 2 || result.Match != 2 || result.NoMatch != 1 || result.Unknown != 1 {
		t.Fatalf("expected pure Record totals 2/1/1 and two members, got %+v", result)
	}

}
