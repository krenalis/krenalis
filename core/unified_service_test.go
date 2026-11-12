// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/initdb"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/util"
	"github.com/krenalis/krenalis/test/snowflaketester"
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

const (
	unifiedServiceAt     = "2026-11-02T12:00:00.123456Z"
	unifiedServiceStart  = "2026-10-01T00:00:00Z"
	unifiedServiceSource = `{"kind":"recordedEvents","name":"Order Completed","grain":"registered-message"}`
	unifiedServicePeriod = `{"kind":"absoluteHalfOpen","start":"` + unifiedServiceStart +
		`","end":"` + unifiedServiceAt + `"}`
	unifiedServiceField = `{"definition":"commerce.order_completed.total","meaningRevision":"1"}`
	unifiedServiceBase  = `{"source":` + unifiedServiceSource + `,"period":` + unifiedServicePeriod +
		`,"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"},"nativeCurrency":"EUR"}`
	unifiedServiceMoney = `{"money":{"field":` + unifiedServiceField +
		`,"nativeCurrency":"EUR"},"operator":"is greater than or equal to","values":[100.00001]}`
	unifiedServiceMoneyExplanation = `{"path":[0],` +
		`"field":{"definition":"commerce.order_completed.total","meaningRevision":"1"},"nativeCurrency":"EUR",` +
		`"requested":{"operator":"is greater than or equal to","values":[100.00001]},` +
		`"normalized":{"operator":"is greater than or equal to","values":[100.0001]}}`
	unifiedServiceBounds = `"start":"` + unifiedServiceStart + `","end":"` + unifiedServiceAt + `"`
)

// TestUnifiedServicePostgreSQL checks public values against the shared oracle.
func TestUnifiedServicePostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("requires PostgreSQL")
	}
	testUnifiedService(t, "PostgreSQL")
}

// TestUnifiedServiceSnowflake requires an isolated gate-overlay build.
func TestUnifiedServiceSnowflake(t *testing.T) {
	if testing.Short() || os.Getenv("KRENALIS_SKIP_SNOWFLAKE_TESTS") == "true" {
		t.Skip("remote Snowflake tests are disabled")
	}
	dw := warehouses.Registered("Snowflake").New(unifiedTestSettings{json.Value(`{}`)}, nil)
	verified := dw.(warehouses.ProfileFilterWarehouse).ProfileFilterVerified()
	err := dw.Close()
	if err != nil {
		t.Fatalf("expected unopened warehouse cleanup, got %v", err)
	}
	if !verified {
		t.Skip("Snowflake verification requires an isolated test-overlay build")
	}
	testUnifiedService(t, "Snowflake")
}

// assertUnifiedServiceJSON compares exact public numbers, times and metadata.
func assertUnifiedServiceJSON(t *testing.T, actual any, expected string) {
	t.Helper()
	encoded, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("expected public JSON encoding, got %v", err)
	}
	want, err := json.Compact(json.Value(expected))
	if err != nil {
		t.Fatalf("expected valid handwritten JSON oracle, got %v", err)
	}
	if string(encoded) != string(want) {
		t.Fatalf("expected JSON %s, got %s", want, encoded)
	}
}

// newUnifiedServiceWorkspace uses production DDL and real state/datastore, but
// the facade's Core and warehouse factory are fixtures, not Core.Open or HTTP.
func newUnifiedServiceWorkspace(t *testing.T, name string) (*Workspace, warehouses.Warehouse) {

	t.Helper()
	container, err := postgres.Run(t.Context(), testimages.PostgreSQL, postgres.WithDatabase("krenalis"),
		postgres.WithUsername("krenalis"), postgres.WithPassword("krenalis"), postgres.BasicWaitStrategies())
	if err != nil {
		t.Fatalf("expected application PostgreSQL container, got %v", err)
	}
	t.Cleanup(func() {
		err := testcontainers.TerminateContainer(container)
		if err != nil {
			t.Errorf("expected application container cleanup, got %v", err)
		}
	})
	host, err := container.Host(t.Context())
	if err != nil {
		t.Fatalf("expected application database host, got %v", err)
	}
	port, err := container.MappedPort(t.Context(), "5432/tcp")
	if err != nil {
		t.Fatalf("expected application database port, got %v", err)
	}
	database, err := db.Open(&db.Options{Host: host, Port: int(port.Num()), Username: "krenalis",
		Password: "krenalis", Database: "krenalis"})
	if err != nil {
		t.Fatalf("expected application database, got %v", err)
	}
	t.Cleanup(database.Close)
	manager, err := kms.New(t.Context(), "key:"+base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatalf("expected test KMS, got %v", err)
	}
	err = initdb.InitIfEmpty(t.Context(), database, manager, false)
	if err != nil {
		t.Fatalf("expected application schema, got %v", err)
	}

	platform := fmt.Sprintf("UnifiedServiceTest%d", unifiedWarehouseNumber.Add(1))
	var dw warehouses.Warehouse
	switch name {
	case "PostgreSQL":
		_, err = database.Exec(t.Context(), "CREATE SCHEMA unified_service")
		if err != nil {
			t.Fatalf("expected isolated warehouse schema, got %v", err)
		}
		settings, err := json.Marshal(map[string]any{"host": host, "port": port.Num(), "username": "krenalis",
			"password": "krenalis", "database": "krenalis", "schema": "unified_service"})
		if err != nil {
			t.Fatalf("expected warehouse settings encoding, got %v", err)
		}
		warehouse := warehouses.Registered(name).New(unifiedTestSettings{settings}, nil).(*postgresql.PostgreSQL)
		warehouses.Register(warehouses.Platform{Name: platform},
			func(warehouses.SettingsLoader, warehouses.DialWith) *postgresql.PostgreSQL { return warehouse })
		dw = warehouse
	case "Snowflake":
		environment, err := snowflaketester.CreateTestEnvironment()
		if err != nil {
			t.Fatalf("expected isolated Snowflake environment, got %v", err)
		}
		t.Cleanup(func() {
			err := environment.Teardown()
			if err != nil {
				t.Errorf("expected Snowflake environment cleanup, got %v", err)
			}
		})
		settings := json.Value(environment.Settings().JSON())
		warehouse := warehouses.Registered(name).New(unifiedTestSettings{settings}, nil).(*snowflake.Snowflake)
		warehouses.Register(warehouses.Platform{Name: platform},
			func(warehouses.SettingsLoader, warehouses.DialWith) *snowflake.Snowflake { return warehouse })
		dw = warehouse
	default:
		t.Fatalf("expected supported service warehouse, got %s", name)
	}
	t.Cleanup(func() {
		err := dw.Close()
		if err != nil {
			t.Errorf("expected service warehouse cleanup, got %v", err)
		}
	})

	schema := types.Object([]types.Property{
		{Name: "label", Type: types.String(), ReadOptional: true},
		{Name: "amount", Type: types.Decimal(38, 10), ReadOptional: true},
		{Name: "document", Type: types.JSON(), ReadOptional: true},
		{Name: "settings", ReadOptional: true, Type: types.Object([]types.Property{
			{Name: "value", Type: types.String(), ReadOptional: true},
		})},
		{Name: "tags", Type: types.Array(types.String()), ReadOptional: true},
		{Name: "bio", Type: types.String(), ReadOptional: true},
	})
	columns := util.PropertiesToColumns(schema.Properties())
	err = dw.Initialize(t.Context(), columns)
	if err != nil {
		t.Fatalf("expected production warehouse initialization, got %v", err)
	}
	encoded, err := schema.MarshalJSON()
	if err != nil {
		t.Fatalf("expected profile schema encoding, got %v", err)
	}
	_, err = database.Exec(t.Context(), `INSERT INTO workspaces
 (id,organization,name,warehouse_name,warehouse_mode,warehouse_settings,
 kms_encrypted_warehouse_settings_key,kms_encrypted_warehouse_mcp_settings_key,profile_schema,identifiers)
 SELECT '111111111111',id,'Unified service',$1,'Normal','','','',$2,ARRAY[]::text[] FROM organizations`,
		platform, encoded)
	if err != nil {
		t.Fatalf("expected workspace state fixture, got %v", err)
	}
	st, err := state.New(t.Context(), database, manager, nil, false)
	if err != nil {
		t.Fatalf("expected application state, got %v", err)
	}
	t.Cleanup(func() { st.Close(context.Background()) })
	// New/newStore only install listeners and build the warehouse and column
	// snapshot. No pipeline or background publisher runs in this fixture.
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

	return &Workspace{core: &Core{}, store: store, workspace: ws}, dw
}

func testUnifiedService(t *testing.T, name string) {

	t.Helper()
	workspace, dw := newUnifiedServiceWorkspace(t, name)
	updated := time.Date(2026, 10, 1, 12, 0, 0, 123456000, time.UTC)
	profiles := warehouses.Table{Name: "krenalis_profiles_0", Keys: []string{"_kpid"}, Columns: []warehouses.Column{
		{Name: "_kpid", Type: types.UUID()}, {Name: "_updated_at", Type: types.DateTime()},
		{Name: "_anonymous_count", Type: types.Int(64)}, {Name: "_recognized_count", Type: types.Int(64)},
		{Name: "label", Type: types.String(), Nullable: true}, {Name: "amount", Type: types.Decimal(38, 10)},
		{Name: "document", Type: types.JSON()}, {Name: "settings_value", Type: types.String()},
		{Name: "tags", Type: types.Array(types.String())},
	}}
	var data [][]any
	for i, label := range []any{"same", "same", nil, "z", nil, "a"} {
		data = append(data, []any{fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1), updated, 0, 0, label,
			decimal.MustParse("9007199254740993.0000000001"), json.Value(`{"v":null}`), "nested", []any{"", "x"}})
	}
	err := dw.Merge(t.Context(), profiles, data, nil)
	if err != nil {
		t.Fatalf("expected real profile writes, got %v", err)
	}
	events := warehouses.Table{Name: "krenalis_events", Keys: []string{"message_id"}}
	for _, path := range []string{"kpid", "messageId", "connectionId", "anonymousId", "receivedAt", "sentAt",
		"timestamp", "traits", "type", "event", "name", "context.page.path", "properties"} {
		events.Columns = append(events.Columns, datastore.EventColumnByPath(path))
	}
	data = nil
	for i, event := range []struct {
		profile int
		stamp   string
		typ     string
		payload string
	}{
		{1, "2026-10-10T10:00:00Z", "track", `{"total":100.125,"currency":"EUR"}`},
		{1, "2026-10-20T10:00:00Z", "track", `{"total":70.0003,"currency":"EUR"}`},
		{2, "2026-10-10T10:00:00Z", "track", `{"total":20,"currency":"EUR"}`},
		{2, "2026-10-20T10:00:00Z", "track", `{"total":150}`},
		{4, "2026-10-21T10:00:00Z", "track", `{"total":200,"currency":"EUR"}`},
		{4, "2026-10-21T10:00:00Z", "track", `{"total":50,"currency":"EUR"}`},
		{5, "2026-10-20T10:00:00Z", "track", `{"total":1000,"currency":"USD"}`},
		{6, "2026-10-10T10:00:00Z", "track", `{"total":-30.0001,"currency":"EUR"}`},
		{6, "2026-10-20T10:00:00Z", "track", `{"total":"unreadable","currency":"EUR"}`},
		{1, "2026-09-30T21:59:59.999Z", "page", `{}`},
		{2, "2026-09-30T22:00:00Z", "page", `{}`},
		{4, "2026-10-31T23:00:00Z", "page", `{}`},
		{6, "2026-10-31T22:59:59.999Z", "page", `{}`},
	} {
		stamp, err := time.Parse(time.RFC3339Nano, event.stamp)
		if err != nil {
			t.Fatalf("expected fixture timestamp, got %v", err)
		}
		var eventName, pageName, path any
		if event.typ == "track" {
			eventName = "Order Completed"
		} else {
			pageName, path = "Pricing", "/pricing"
		}
		data = append(data, []any{fmt.Sprintf("00000000-0000-0000-0000-%012d", event.profile),
			fmt.Sprintf("service-%d", i), "test-connection", "test-anonymous", updated, updated, stamp,
			json.Value(`{}`), event.typ, eventName, pageName, path, json.Value(event.payload)})
	}
	err = dw.Merge(t.Context(), events, data, nil)
	if err != nil {
		t.Fatalf("expected real recorded event writes, got %v", err)
	}

	count := `{"count":{"source":` + unifiedServiceSource + `,"period":` + unifiedServicePeriod +
		`,"where":{"operator":"and","rules":[` + unifiedServiceMoney + `]}},` +
		`"operator":"is greater than or equal to","values":[1]}`
	for _, test := range []struct {
		name         string
		body         string
		truths       []string
		explanations map[int]string
	}{
		{"Count money", count, []string{"MATCH", "UNKNOWN", "NO_MATCH", "MATCH", "NO_MATCH", "UNKNOWN"}, map[int]string{
			1: `{"kind":"count","result":"MATCH","count":{` + unifiedServiceBounds + `,"certain":1,"uncertain":0,"money":[` + unifiedServiceMoneyExplanation + `]}}`,
			2: `{"kind":"count","result":"UNKNOWN","count":{` + unifiedServiceBounds + `,"certain":0,"uncertain":1,"money":[` + unifiedServiceMoneyExplanation + `]}}`,
			3: `{"kind":"count","result":"NO_MATCH","count":{` + unifiedServiceBounds + `,"certain":0,"uncertain":0,"money":[` + unifiedServiceMoneyExplanation + `]}}`,
			4: `{"kind":"count","result":"MATCH","count":{` + unifiedServiceBounds + `,"certain":1,"uncertain":0,"money":[` + unifiedServiceMoneyExplanation + `]}}`,
			5: `{"kind":"count","result":"NO_MATCH","count":{` + unifiedServiceBounds + `,"certain":0,"uncertain":0,"money":[` + unifiedServiceMoneyExplanation + `]}}`,
			6: `{"kind":"count","result":"UNKNOWN","count":{` + unifiedServiceBounds + `,"certain":0,"uncertain":1,"money":[` + unifiedServiceMoneyExplanation + `]}}`,
		}},
		{"Sum", `{"sum":{"source":` + unifiedServiceSource + `,"period":` + unifiedServicePeriod + `,"field":` + unifiedServiceField + `,"nativeCurrency":"EUR","minimum":2},"operator":"is greater than or equal to","values":[170]}`,
			[]string{"MATCH", "UNKNOWN", "NO_MATCH", "MATCH", "NO_MATCH", "UNKNOWN"}, map[int]string{
				1: `{"kind":"sum","result":"MATCH","sum":{` + unifiedServiceBounds + `,"field":` + unifiedServiceField + `,"nativeCurrency":"EUR","bounds":{"lower":170.1253,"upper":170.1253},"certain":2,"uncertain":0}}`,
				6: `{"kind":"sum","result":"UNKNOWN","sum":{` + unifiedServiceBounds + `,"field":` + unifiedServiceField + `,"nativeCurrency":"EUR","bounds":{"lower":null,"upper":null},"certain":2,"uncertain":0}}`,
			}},
		{"Average", `{"average":{"source":` + unifiedServiceSource + `,"period":` + unifiedServicePeriod + `,"field":` + unifiedServiceField + `,"nativeCurrency":"EUR","minimum":2},"operator":"is greater than or equal to","values":[85]}`,
			[]string{"MATCH", "UNKNOWN", "NO_MATCH", "MATCH", "NO_MATCH", "UNKNOWN"}, map[int]string{
				1: `{"kind":"average","result":"MATCH","average":{` + unifiedServiceBounds + `,"field":` + unifiedServiceField + `,"nativeCurrency":"EUR","sum":170.1253,"count":2,"certain":2,"uncertain":0}}`,
				2: `{"kind":"average","result":"UNKNOWN","average":{` + unifiedServiceBounds + `,"field":` + unifiedServiceField + `,"nativeCurrency":"EUR","sum":null,"count":null,"certain":1,"uncertain":1}}`,
			}},
		{"Nth with occurredIn", `{"experience":{"base":` + unifiedServiceBase + `,"select":{"kind":"nth","position":2},"verify":{"operator":"and","rules":[{"occurredIn":{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}}},` + unifiedServiceMoney + `]}}}`,
			[]string{"NO_MATCH", "UNKNOWN", "NO_MATCH", "UNKNOWN", "NO_MATCH", "UNKNOWN"}, map[int]string{
				2: `{"kind":"experience","result":"UNKNOWN","experience":{` + unifiedServiceBounds + `,"select":{"kind":"nth","position":2},"selection":{"kind":"ambiguous","absence":"possible","candidates":1},"occurredIn":[{"path":[0],"start":"2026-10-03T12:00:00.123456Z","end":"` + unifiedServiceAt + `"}],"money":[` + strings.Replace(unifiedServiceMoneyExplanation, `"path":[0]`, `"path":[1]`, 1) + `],"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"},"nativeCurrency":"EUR"}}`,
				3: `{"kind":"experience","result":"NO_MATCH","experience":{` + unifiedServiceBounds + `,"select":{"kind":"nth","position":2},"selection":{"kind":"absent","absence":"certain","candidates":0},"occurredIn":[{"path":[0],"start":"2026-10-03T12:00:00.123456Z","end":"` + unifiedServiceAt + `"}],"money":[` + strings.Replace(unifiedServiceMoneyExplanation, `"path":[0]`, `"path":[1]`, 1) + `],"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"},"nativeCurrency":"EUR"}}`,
				4: `{"kind":"experience","result":"UNKNOWN","experience":{` + unifiedServiceBounds + `,"select":{"kind":"nth","position":2},"selection":{"kind":"ambiguous","absence":"impossible","candidates":2},"occurredIn":[{"path":[0],"start":"2026-10-03T12:00:00.123456Z","end":"` + unifiedServiceAt + `"}],"money":[` + strings.Replace(unifiedServiceMoneyExplanation, `"path":[0]`, `"path":[1]`, 1) + `],"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"},"nativeCurrency":"EUR"}}`,
			}},
		{"Latest N", `{"latestN":{"base":` + unifiedServiceBase + `,"size":2,"predicate":{"operator":"and","rules":[` + unifiedServiceMoney + `]},"quantifier":{"kind":"some"}}}`,
			[]string{"MATCH", "UNKNOWN", "NO_MATCH", "MATCH", "NO_MATCH", "UNKNOWN"}, map[int]string{
				4: `{"kind":"latestN","result":"MATCH","latestN":{` + unifiedServiceBounds + `,"size":2,"quantifier":{"kind":"some"},"selection":{"kind":"unique","absence":"impossible","candidates":2},"money":[` + unifiedServiceMoneyExplanation + `],"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"},"nativeCurrency":"EUR"}}`,
			}},
		{"Last versus previous", `{"lastVersusPrevious":{"base":` + unifiedServiceBase + `,"field":` + unifiedServiceField + `,"check":{"kind":"difference","operator":"is less than or equal to","values":[-30]}}}`,
			[]string{"MATCH", "NO_MATCH", "NO_MATCH", "UNKNOWN", "NO_MATCH", "UNKNOWN"}, map[int]string{
				4: `{"kind":"lastVersusPrevious","result":"UNKNOWN","lastVersusPrevious":{` + unifiedServiceBounds + `,"field":` + unifiedServiceField + `,"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"},"nativeCurrency":"EUR","check":{"kind":"difference","requested":{"operator":"is less than or equal to","values":[-3E+1]},"normalized":{"operator":"is less than or equal to","values":[-3E+1]}},"selection":{"kind":"ambiguous","absence":"impossible","candidates":2}}}`,
			}},
	} {
		t.Run(test.name, func(t *testing.T) {

			var filter Filter
			err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+test.body+`]}`), &filter)
			if err != nil {
				t.Fatalf("expected public filter decoding, got %v", err)
			}
			result, err := workspace.QueryProfiles(snowflaketester.ServiceContext(t.Context(), snowflaketester.ServiceQueryTag),
				ProfileQuery{Filter: &filter, At: unifiedServiceAt, Properties: []string{"label"}, Limit: 10})
			if err != nil {
				t.Fatalf("expected complete public query, got %v", err)
			}

			var ids []string
			var counts [3]int64
			for i, truth := range test.truths {
				switch truth {
				case "MATCH":
					counts[0]++
					ids = append(ids, fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1))
				case "NO_MATCH":
					counts[1]++
				case "UNKNOWN":
					counts[2]++
				default:
					t.Fatalf("expected ternary oracle, got %s", truth)
				}
			}
			var got []string
			for _, profile := range result.Profiles {
				got = append(got, profile.KPID)
			}
			if !reflect.DeepEqual(got, ids) || result.Total != len(ids) || result.Evaluation == nil {
				t.Fatalf("expected matching IDs %v and total %d, got %+v", ids, len(ids), result)
			}
			assertUnifiedServiceJSON(t, result.Evaluation, fmt.Sprintf(
				`{"at":"%s","match":%d,"noMatch":%d,"unknown":%d}`, unifiedServiceAt, counts[0], counts[1], counts[2]))

			for profile := 1; profile <= 6; profile++ {
				leaf, ok := test.explanations[profile]
				if !ok {
					continue
				}
				id := fmt.Sprintf("00000000-0000-0000-0000-%012d", profile)
				explanation, err := workspace.ExplainProfileFilter(
					snowflaketester.ServiceContext(t.Context(), snowflaketester.ServiceExplanationTag), id, &filter, unifiedServiceAt)
				if err != nil {
					t.Fatalf("expected public explanation for %s, got %v", id, err)
				}
				assertUnifiedServiceJSON(t, explanation, `{"kpid":"`+id+`","at":"`+unifiedServiceAt+
					`","rule":{"kind":"and","result":"`+test.truths[profile-1]+`","rules":[`+leaf+`]}}`)
			}

		})
	}

	testUnifiedServicePages(t, workspace)
	testUnifiedServiceRecordCalendar(t, workspace)
	testUnifiedServiceLargePage(t, workspace, dw)

}

func testUnifiedServiceLargePage(t *testing.T, workspace *Workspace, dw warehouses.Warehouse) {
	t.Helper()
	t.Run("page budget", func(t *testing.T) {
		table := warehouses.Table{Name: "krenalis_profiles_0", Keys: []string{"_kpid"}, Columns: []warehouses.Column{
			{Name: "_kpid", Type: types.UUID()}, {Name: "bio", Type: types.String()},
		}}
		err := dw.Merge(t.Context(), table,
			[][]any{{"00000000-0000-0000-0000-000000000006", strings.Repeat("x", warehouses.MaxProfilePageBytes+1)}}, nil)
		if err != nil {
			t.Fatalf("expected oversized profile fixture, got %v", err)
		}
		var filter Filter
		err = json.Unmarshal([]byte(`{"operator":"and","rules":[{"count":{"source":`+unifiedServiceSource+
			`,"period":`+unifiedServicePeriod+`},"operator":"is greater than or equal to","values":[0]}]}`), &filter)
		if err != nil {
			t.Fatalf("expected all-profile filter, got %v", err)
		}
		result, err := workspace.QueryProfiles(snowflaketester.ServiceContext(t.Context(), snowflaketester.ServiceQueryTag),
			ProfileQuery{Filter: &filter, At: unifiedServiceAt, Properties: []string{"bio"}, Limit: 10})
		if err != nil {
			failure, ok := errors.AsType[*errors.UnprocessableError](err)
			if !ok || failure.Code != ProfileResultTooLarge || result != nil {
				t.Fatalf("expected ProfileResultTooLarge without a partial page, got %+v, %v", result, err)
			}
			return
		}
		t.Fatalf("expected oversized page rejection, got %+v", result)
	})
}

func testUnifiedServicePages(t *testing.T, workspace *Workspace) {
	t.Helper()
	t.Run("public projection and pages", func(t *testing.T) {

		var filter Filter
		err := json.Unmarshal([]byte(`{"operator":"and","rules":[{"count":{"source":`+unifiedServiceSource+
			`,"period":`+unifiedServicePeriod+`},"operator":"is greater than or equal to","values":[0]}]}`), &filter)
		if err != nil {
			t.Fatalf("expected all-profile filter, got %v", err)
		}
		for _, descending := range []bool{false, true} {
			want := []int{6, 1, 2, 4, 3, 5}
			if descending {
				want = []int{4, 1, 2, 6, 3, 5}
			}
			for first := 0; first <= 6; first += 2 {
				result, err := workspace.QueryProfiles(snowflaketester.ServiceContext(t.Context(), snowflaketester.ServiceQueryTag),
					ProfileQuery{Filter: &filter, At: unifiedServiceAt, Properties: []string{"label"},
						Order: "label", OrderDesc: descending, First: first, Limit: 2})
				if err != nil {
					t.Fatalf("expected page %d descending=%t, got %v", first, descending, err)
				}
				if result.Total != 6 || len(result.Profiles) != min(2, 6-first) {
					t.Fatalf("expected page size %d and total 6, got %+v", min(2, 6-first), result)
				}
				assertUnifiedServiceJSON(t, result.Evaluation,
					`{"at":"`+unifiedServiceAt+`","match":6,"noMatch":0,"unknown":0}`)
				for i, profile := range result.Profiles {
					id := fmt.Sprintf("00000000-0000-0000-0000-%012d", want[first+i])
					if profile.KPID != id || profile.UpdatedAt.Format(time.RFC3339Nano) != "2026-10-01T12:00:00.123456Z" {
						t.Fatalf("expected ordered ID %s and microsecond update time, got %+v", id, profile)
					}
				}
			}
		}

		result, err := workspace.QueryProfiles(snowflaketester.ServiceContext(t.Context(), snowflaketester.ServiceQueryTag),
			ProfileQuery{Filter: &filter, At: unifiedServiceAt,
				Properties: []string{"label", "amount", "document", "settings", "tags"},
				Order:      "label", First: 4, Limit: 1})
		if err != nil {
			t.Fatalf("expected projected public page, got %v", err)
		}
		if len(result.Profiles) != 1 {
			t.Fatalf("expected one projected profile, got %+v", result)
		}
		attributes, err := types.Marshal(result.Profiles[0].Attributes, result.Schema)
		if err != nil {
			t.Fatalf("expected schema-directed public attributes, got %v", err)
		}
		// These are the pieces encoded by the HTTP handler, without invoking it.
		page := struct {
			Profiles   []json.Value       `json:"profiles"`
			Schema     types.Type         `json:"schema"`
			Total      int                `json:"total"`
			Evaluation *ProfileEvaluation `json:"evaluation"`
		}{Profiles: []json.Value{json.Value(`{"kpid":"` + result.Profiles[0].KPID + `","updatedAt":"` +
			result.Profiles[0].UpdatedAt.Format(time.RFC3339Nano) + `","attributes":` + string(attributes) + `}`)},
			Schema: result.Schema, Total: result.Total, Evaluation: result.Evaluation}
		assertUnifiedServiceJSON(t, page,
			`{"profiles":[{"kpid":"00000000-0000-0000-0000-000000000003","updatedAt":"2026-10-01T12:00:00.123456Z",`+
				`"attributes":{"amount":9007199254740993.0000000001,"document":{"v":null},`+
				`"settings":{"value":"nested"},"tags":["","x"]}}],"schema":{"kind":"object","properties":[`+
				`{"name":"label","type":{"kind":"string"},"readOptional":true},`+
				`{"name":"amount","type":{"kind":"decimal","precision":38,"scale":10},"readOptional":true},`+
				`{"name":"document","type":{"kind":"json"},"readOptional":true},`+
				`{"name":"settings","type":{"kind":"object","properties":[`+
				`{"name":"value","type":{"kind":"string"},"readOptional":true}]},"readOptional":true},`+
				`{"name":"tags","type":{"kind":"array","elementType":{"kind":"string"}},"readOptional":true}]},`+
				`"total":6,"evaluation":{"at":"2026-11-02T12:00:00.123456Z","match":6,"noMatch":0,"unknown":0}}`)

		missing, err := workspace.ExplainProfileFilter(
			snowflaketester.ServiceContext(t.Context(), snowflaketester.ServiceExplanationTag),
			"00000000-0000-0000-0000-000000000099", &filter, unifiedServiceAt)
		if err != nil {
			if _, ok := errors.AsType[*errors.NotFoundError](err); !ok || missing != nil {
				t.Fatalf("expected NotFound without a missing profile explanation, got %+v, %v", missing, err)
			}
			return
		}
		t.Fatalf("expected missing profile rejection, got %+v", missing)

	})
}

func testUnifiedServiceRecordCalendar(t *testing.T, workspace *Workspace) {

	t.Helper()
	t.Run("Record", func(t *testing.T) {
		filter := &Filter{Operator: OpAnd, Rules: []FilterRule{
			&FilterCondition{Property: "label", Operator: OpIs, Values: filterValues("same")}}}
		result, err := workspace.QueryProfiles(snowflaketester.ServiceContext(t.Context(), snowflaketester.ServiceRecordTag),
			ProfileQuery{Filter: filter, Properties: []string{"label"}, Limit: 10})
		if err != nil {
			t.Fatalf("expected legacy Record page, got %v", err)
		}
		if result.Evaluation != nil || result.Total != 2 || len(result.Profiles) != 2 {
			t.Fatalf("expected two Record profiles without evaluation, got %+v", result)
		}
		explanation, err := workspace.ExplainProfileFilter(
			snowflaketester.ServiceContext(t.Context(), snowflaketester.ServiceExplanationTag),
			"00000000-0000-0000-0000-000000000003", filter, unifiedServiceAt)
		if err != nil {
			t.Fatalf("expected Record explanation, got %v", err)
		}
		assertUnifiedServiceJSON(t, explanation,
			`{"kpid":"00000000-0000-0000-0000-000000000003","at":"2026-11-02T12:00:00.123456Z",`+
				`"rule":{"kind":"and","result":"UNKNOWN","rules":[{"kind":"record","result":"UNKNOWN"}]}}`)
	})

	t.Run("calendar boundaries", func(t *testing.T) {
		var filter Filter
		err := json.Unmarshal([]byte(
			`{"defaultCalendar":{"definition":"gregorian-monday/Europe/Rome","meaningRevision":"1"},`+
				`"operator":"and","rules":[{"count":{"source":{"kind":"recordedEvents","type":"page",`+
				`"name":"Pricing","grain":"registered-message"},`+
				`"period":{"kind":"lastCompleteCalendarPeriods","unit":"month","count":1},`+
				`"where":{"operator":"and","rules":[{"property":"context.page.path","operator":"is",`+
				`"values":["/pricing"]}]}},"operator":"is greater than or equal to","values":[1]}]}`), &filter)
		if err != nil {
			t.Fatalf("expected public calendar filter, got %v", err)
		}
		result, err := workspace.QueryProfiles(snowflaketester.ServiceContext(t.Context(), snowflaketester.ServiceQueryTag),
			ProfileQuery{Filter: &filter, At: unifiedServiceAt, Properties: []string{"label"}, Limit: 10})
		if err != nil {
			t.Fatalf("expected calendar query, got %v", err)
		}
		if result.Total != 2 || len(result.Profiles) != 2 ||
			result.Profiles[0].KPID != "00000000-0000-0000-0000-000000000002" ||
			result.Profiles[1].KPID != "00000000-0000-0000-0000-000000000006" {
			t.Fatalf("expected exact half-open calendar membership for profiles 2 and 6, got %+v", result)
		}
		assertUnifiedServiceJSON(t, result.Evaluation,
			`{"at":"2026-11-02T12:00:00.123456Z","match":2,"noMatch":4,"unknown":0}`)
		explanation, err := workspace.ExplainProfileFilter(
			snowflaketester.ServiceContext(t.Context(), snowflaketester.ServiceExplanationTag),
			"00000000-0000-0000-0000-000000000002", &filter, unifiedServiceAt)
		if err != nil {
			t.Fatalf("expected resolved calendar explanation, got %v", err)
		}
		assertUnifiedServiceJSON(t, explanation,
			`{"kpid":"00000000-0000-0000-0000-000000000002","at":"2026-11-02T12:00:00.123456Z",`+
				`"rule":{"kind":"and","result":"MATCH","rules":[{"kind":"count","result":"MATCH",`+
				`"count":{"start":"2026-09-30T22:00:00Z","end":"2026-10-31T23:00:00Z","certain":1,"uncertain":0}}]}}`)
	})

}
