// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/test/snowflaketester"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
	_ "github.com/krenalis/krenalis/warehouses/snowflake"
)

const (
	profilesHTTPAt     = "2026-11-02T12:00:00.123456Z"
	profilesHTTPSource = `{"kind":"recordedEvents","name":"Order Completed","grain":"registered-message"}`
	profilesHTTPPeriod = `{"kind":"absoluteHalfOpen","start":"2026-10-01T00:00:00Z","end":"` + profilesHTTPAt + `"}`
	profilesHTTPField  = `{"definition":"commerce.order_completed.total","meaningRevision":"1"}`
	profilesHTTPMoney  = `{"money":{"field":` + profilesHTTPField +
		`,"nativeCurrency":"EUR"},"operator":"is greater than or equal to","values":[100.00001]}`
	profilesHTTPCount = `{"operator":"and","rules":[{"count":{"source":` + profilesHTTPSource +
		`,"period":` + profilesHTTPPeriod + `,"where":{"operator":"and","rules":[` + profilesHTTPMoney +
		`]}},"operator":"is greater than or equal to","values":[1]}]}`
	profilesHTTPAll = `{"operator":"and","rules":[{"count":{"source":` + profilesHTTPSource +
		`,"period":` + profilesHTTPPeriod + `},"operator":"is greater than or equal to","values":[0]}]}`
	profilesHTTPSum = `{"operator":"and","rules":[{"sum":{"source":` + profilesHTTPSource +
		`,"period":` + profilesHTTPPeriod + `,"field":` + profilesHTTPField +
		`,"nativeCurrency":"EUR","minimum":2},"operator":"is greater than or equal to","values":[170.00001]}]}`
	profilesHTTPLabelSchema = `{"kind":"object","properties":[` +
		`{"name":"label","type":{"kind":"string"},"readOptional":true}]}`
	profilesHTTPRecord = `{"operator":"and","rules":[` +
		`{"property":"label","operator":"does not exist"}]}`
	profilesHTTPUpdated = `"updatedAt":"2026-10-01T12:00:00.123456Z"`
	profilesHTTPBounds  = `"start":"2026-10-01T00:00:00Z","end":"` + profilesHTTPAt + `"`
)

type profileHTTPClient struct {
	client  *http.Client
	address string
	token   string
}

func (c profileHTTPClient) get(t *testing.T, path string, status int) json.Value {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+c.address+path, nil)
	if err != nil {
		t.Fatalf("expected profile HTTP request, got %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	response, err := c.client.Do(request)
	if err != nil {
		t.Fatalf("expected profile HTTP response, got %v", err)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 2<<20+1))
	closeErr := response.Body.Close()
	if err != nil {
		t.Fatalf("expected complete HTTP body, got %v", err)
	}
	if closeErr != nil {
		t.Fatalf("expected HTTP body cleanup, got %v", closeErr)
	}
	if len(data) > 2<<20 {
		t.Fatalf("expected bounded HTTP body, got %d bytes", len(data))
	}
	media, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("expected JSON Content-Type, got %v", err)
	}
	if response.StatusCode != status || media != "application/json" {
		t.Fatalf("expected HTTP %d JSON, got %d %s: %.512s", status, response.StatusCode, media, data)
	}
	compact, err := json.Compact(data)
	if err != nil {
		t.Fatalf("expected JSON response body, got %v", err)
	}
	return json.Value(compact)
}

type profileHTTPSettings struct {
	data json.Value
}

// Load decodes settings for the fixture writer, not the server's warehouse.
func (s profileHTTPSettings) Load(ctx context.Context, value any) error {
	return json.Unmarshal(s.data, value)
}

// TestProfilesHTTPPostgreSQL checks the HTTP contract with a disposable server.
func TestProfilesHTTPPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("requires PostgreSQL, NATS and the server")
	}
	testProfilesHTTP(t, "PostgreSQL")
}

// TestProfilesHTTPSnowflake requires the isolated internal-server overlay.
func TestProfilesHTTPSnowflake(t *testing.T) {
	if testing.Short() || os.Getenv("KRENALIS_SKIP_SNOWFLAKE_TESTS") == "true" {
		t.Skip("remote Snowflake tests are disabled")
	}
	if !krenalistester.UsesInternalServer() {
		t.Skip("Snowflake HTTP verification requires the internal-server overlay")
	}
	dw := warehouses.Registered("Snowflake").New(profileHTTPSettings{json.Value(`{}`)}, nil)
	verified := dw.(warehouses.ProfileFilterWarehouse).ProfileFilterVerified()
	err := dw.Close()
	if err != nil {
		t.Fatalf("expected unopened warehouse cleanup, got %v", err)
	}
	if !verified {
		t.Skip("Snowflake HTTP verification requires the isolated gate overlay")
	}
	testProfilesHTTP(t, "Snowflake")
}

func assertProfilesHTTPJSON(t *testing.T, actual json.Value, expected string) {
	t.Helper()
	if string(actual) != expected {
		t.Fatalf("expected HTTP JSON %s, got %s", expected, actual)
	}
}

func newProfileTransportFixture(t *testing.T, platform string) (*krenalistester.Krenalis, warehouses.Warehouse) {

	t.Helper()
	var settings json.Value
	if platform == "Snowflake" {
		environment, err := snowflaketester.CreateTestEnvironment()
		if err != nil {
			t.Fatalf("expected isolated profile Snowflake environment, got %v", err)
		}
		t.Cleanup(func() {
			err := environment.Teardown()
			if err != nil {
				t.Errorf("expected profile Snowflake environment cleanup, got %v", err)
			}
		})
		settings = json.Value(environment.Settings().JSON())
	}
	k := krenalistester.NewKrenalisInstance(t)
	if settings != nil {
		k.SetWarehouse(platform, []byte(settings))
	}
	k.SetProfileSchema(types.Object([]types.Property{
		{Name: "label", Type: types.String(), ReadOptional: true},
		{Name: "amount", Type: types.Decimal(38, 10), ReadOptional: true},
		{Name: "document", Type: types.JSON(), ReadOptional: true},
		{Name: "settings", ReadOptional: true, Type: types.Object([]types.Property{
			{Name: "value", Type: types.String(), ReadOptional: true},
		})},
		{Name: "tags", Type: types.Array(types.String()), ReadOptional: true},
		{Name: "bio", Type: types.String(), ReadOptional: true},
	}))
	k.Start()
	t.Cleanup(k.Stop)
	k.UpdateIdentityResolutionSettings(false, nil)
	if settings == nil {
		settings = krenalistester.PostgresWarehouseSettings()
	}
	dw := warehouses.Registered(platform).New(profileHTTPSettings{settings}, nil)
	t.Cleanup(func() {
		err := dw.Close()
		if err != nil {
			t.Errorf("expected profile fixture writer cleanup, got %v", err)
		}
	})

	updated := time.Date(2026, 10, 1, 12, 0, 0, 123456000, time.UTC)
	profiles := warehouses.Table{Name: "krenalis_profiles_0", Keys: []string{"_kpid"}, Columns: []warehouses.Column{
		{Name: "_kpid", Type: types.UUID()}, {Name: "_updated_at", Type: types.DateTime()},
		{Name: "_anonymous_count", Type: types.Int(64)}, {Name: "_recognized_count", Type: types.Int(64)},
		{Name: "label", Type: types.String(), Nullable: true}, {Name: "amount", Type: types.Decimal(38, 10)},
		{Name: "document", Type: types.JSON()}, {Name: "settings_value", Type: types.String()},
		{Name: "tags", Type: types.Array(types.String())},
	}}
	var data [][]any
	for i, label := range []any{"same", "same", nil, "z"} {
		data = append(data, []any{fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1), updated, 0, 0, label,
			decimal.MustParse("9007199254740993.0000000001"), json.Value(`{"v":null}`), "nested", []any{"", "x"}})
	}
	err := dw.Merge(t.Context(), profiles, data, nil)
	if err != nil {
		t.Fatalf("expected profile fixtures written, got %v", err)
	}
	events := warehouses.Table{Name: "krenalis_events", Keys: []string{"message_id"}, Columns: []warehouses.Column{
		{Name: "kpid", Type: types.UUID()}, {Name: "message_id", Type: types.String()},
		{Name: "connection_id", Type: types.String()}, {Name: "anonymous_id", Type: types.String()},
		{Name: "received_at", Type: types.DateTime()}, {Name: "sent_at", Type: types.DateTime()},
		{Name: "timestamp", Type: types.DateTime()}, {Name: "traits", Type: types.JSON()},
		{Name: "type", Type: types.String().WithValues("alias", "identify", "group", "page", "screen", "track")},
		{Name: "event", Type: types.String()}, {Name: "properties", Type: types.JSON()},
	}}
	data = nil
	for i, event := range []struct {
		profile int
		payload string
	}{
		{1, `{"total":100.125,"currency":"EUR"}`},
		{1, `{"total":70.0003,"currency":"EUR"}`},
		{2, `{"total":20,"currency":"EUR"}`},
		{2, `{"total":150}`},
		{4, `{"total":1000,"currency":"USD"}`},
	} {
		data = append(data, []any{fmt.Sprintf("00000000-0000-0000-0000-%012d", event.profile),
			fmt.Sprintf("http-%d", i), "fixture", "anonymous", updated, updated, updated.AddDate(0, 0, i+1),
			json.Value(`{}`), "track", "Order Completed", json.Value(event.payload)})
	}
	err = dw.Merge(t.Context(), events, data, nil)
	if err != nil {
		t.Fatalf("expected event fixtures written, got %v", err)
	}

	return k, dw
}

func testProfilesHTTP(t *testing.T, platform string) {

	t.Helper()
	k, dw := newProfileTransportFixture(t, platform)
	c := profileHTTPClient{client: &http.Client{Timeout: time.Minute}, address: k.Addr(),
		token: k.CreateWorkspaceRestrictedAPIKey("HTTP profile fixture")}
	t.Cleanup(c.client.CloseIdleConnections)

	for _, test := range []struct {
		name, filter string
		expected     string
	}{
		{"Count", profilesHTTPCount, `{"profiles":[{"kpid":"00000000-0000-0000-0000-000000000001",` + profilesHTTPUpdated + `,"attributes":{"label":"same"}}],"schema":` + profilesHTTPLabelSchema + `,"total":1,"evaluation":{"at":"` + profilesHTTPAt + `","match":1,"noMatch":2,"unknown":1}}`},
		{"Sum", profilesHTTPSum, `{"profiles":[{"kpid":"00000000-0000-0000-0000-000000000001",` + profilesHTTPUpdated + `,"attributes":{"label":"same"}}],"schema":` + profilesHTTPLabelSchema + `,"total":1,"evaluation":{"at":"` + profilesHTTPAt + `","match":1,"noMatch":3,"unknown":0}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			query := url.Values{"filter": {test.filter}, "at": {profilesHTTPAt}, "properties": {"label"}, "limit": {"10"}}
			actual := c.get(t, "/v1/profiles?"+query.Encode(), http.StatusOK)
			assertProfilesHTTPJSON(t, actual, test.expected)
		})
	}
	t.Run("explanations", func(t *testing.T) {
		for _, test := range []struct {
			profile, filter, expected string
		}{
			{"00000000-0000-0000-0000-000000000002", profilesHTTPCount, `{"kpid":"00000000-0000-0000-0000-000000000002","at":"` + profilesHTTPAt + `","rule":{"kind":"and","result":"UNKNOWN","rules":[{"kind":"count","result":"UNKNOWN","count":{` + profilesHTTPBounds + `,"certain":0,"uncertain":1,"money":[{"path":[0],"field":` + profilesHTTPField + `,"nativeCurrency":"EUR","requested":{"operator":"is greater than or equal to","values":[100.00001]},"normalized":{"operator":"is greater than or equal to","values":[100.0001]}}]}}]}}`},
			{"00000000-0000-0000-0000-000000000001", profilesHTTPSum, `{"kpid":"00000000-0000-0000-0000-000000000001","at":"` + profilesHTTPAt + `","rule":{"kind":"and","result":"MATCH","rules":[{"kind":"sum","result":"MATCH","sum":{` + profilesHTTPBounds + `,"field":` + profilesHTTPField + `,"nativeCurrency":"EUR","bounds":{"lower":170.1253,"upper":170.1253},"certain":2,"uncertain":0}}]}}`},
		} {
			query := url.Values{"filter": {test.filter}, "at": {profilesHTTPAt}}
			actual := c.get(t, "/v1/profiles/"+test.profile+"/filter-explanation?"+query.Encode(), http.StatusOK)
			assertProfilesHTTPJSON(t, actual, test.expected)
		}
	})

	testProfilesHTTPPages(t, c)
	testProfilesHTTPErrors(t, c, dw)

}

func testProfilesHTTPErrors(t *testing.T, c profileHTTPClient, dw warehouses.Warehouse) {

	t.Helper()
	t.Run("missing profile", func(t *testing.T) {
		query := url.Values{"filter": {profilesHTTPCount}, "at": {profilesHTTPAt}}
		body := c.get(t, "/v1/profiles/00000000-0000-0000-0000-000000000099/filter-explanation?"+
			query.Encode(), http.StatusNotFound)
		assertProfilesHTTPJSON(t, body, `{"error":{"code":"NotFound","message":"profile does not exist"}}`)
	})

	t.Run("page budget", func(t *testing.T) {
		table := warehouses.Table{Name: "krenalis_profiles_0", Keys: []string{"_kpid"}, Columns: []warehouses.Column{
			{Name: "_kpid", Type: types.UUID()}, {Name: "bio", Type: types.String()},
		}}
		err := dw.Merge(t.Context(), table,
			[][]any{{"00000000-0000-0000-0000-000000000004", strings.Repeat("x", warehouses.MaxProfilePageBytes+1)}}, nil)
		if err != nil {
			t.Fatalf("expected oversized HTTP profile fixture, got %v", err)
		}
		query := url.Values{"filter": {profilesHTTPAll}, "at": {profilesHTTPAt}, "properties": {"bio"}, "limit": {"10"}}
		body := c.get(t, "/v1/profiles?"+query.Encode(), http.StatusUnprocessableEntity)
		assertProfilesHTTPJSON(t, body,
			`{"error":{"code":"ProfileResultTooLarge",`+
				`"message":"profile page exceeds the supported size; reduce limit or request fewer properties"}}`)
	})

}

func testProfilesHTTPPages(t *testing.T, c profileHTTPClient) {

	t.Helper()
	t.Run("pages", func(t *testing.T) {
		for _, test := range []struct {
			first, desc, expected string
		}{
			{"1", "false", `[{"kpid":"00000000-0000-0000-0000-000000000002",` + profilesHTTPUpdated + `,"attributes":{"label":"same"}},{"kpid":"00000000-0000-0000-0000-000000000004",` + profilesHTTPUpdated + `,"attributes":{"label":"z"}}]`},
			{"0", "true", `[{"kpid":"00000000-0000-0000-0000-000000000004",` + profilesHTTPUpdated + `,"attributes":{"label":"z"}},{"kpid":"00000000-0000-0000-0000-000000000001",` + profilesHTTPUpdated + `,"attributes":{"label":"same"}}]`},
			{"99", "false", `[]`},
		} {
			query := url.Values{"filter": {profilesHTTPAll}, "at": {profilesHTTPAt}, "properties": {"label"},
				"order": {"label"}, "first": {test.first}, "limit": {"2"}, "orderDesc": {test.desc}}
			body := c.get(t, "/v1/profiles?"+query.Encode(), http.StatusOK)
			assertProfilesHTTPJSON(t, body, `{"profiles":`+test.expected+`,"schema":`+profilesHTTPLabelSchema+
				`,"total":4,"evaluation":{"at":"`+profilesHTTPAt+`","match":4,"noMatch":0,"unknown":0}}`)
		}
	})

	t.Run("projection", func(t *testing.T) {
		query := url.Values{"filter": {profilesHTTPAll}, "at": {profilesHTTPAt},
			"properties": {"label", "amount", "document", "settings", "tags"},
			"order":      {"label"}, "first": {"3"}, "limit": {"1"}}
		body := c.get(t, "/v1/profiles?"+query.Encode(), http.StatusOK)
		assertProfilesHTTPJSON(t, body,
			`{"profiles":[{"kpid":"00000000-0000-0000-0000-000000000003",`+profilesHTTPUpdated+
				`,"attributes":{"amount":9007199254740993.0000000001,"document":{"v":null},`+
				`"settings":{"value":"nested"},"tags":["","x"]}}],"schema":{"kind":"object","properties":[`+
				`{"name":"label","type":{"kind":"string"},"readOptional":true},`+
				`{"name":"amount","type":{"kind":"decimal","precision":38,"scale":10},"readOptional":true},`+
				`{"name":"document","type":{"kind":"json"},"readOptional":true},`+
				`{"name":"settings","type":{"kind":"object","properties":[`+
				`{"name":"value","type":{"kind":"string"},"readOptional":true}]},"readOptional":true},`+
				`{"name":"tags","type":{"kind":"array","elementType":{"kind":"string"}},"readOptional":true}]},`+
				`"total":4,"evaluation":{"at":"`+profilesHTTPAt+`","match":4,"noMatch":0,"unknown":0}}`)
	})

	t.Run("Record", func(t *testing.T) {
		query := url.Values{"filter": {profilesHTTPRecord},
			"properties": {"label", "amount", "document"}, "limit": {"1"}}
		body := c.get(t, "/v1/profiles?"+query.Encode(), http.StatusOK)
		assertProfilesHTTPJSON(t, body,
			`{"profiles":[{"kpid":"00000000-0000-0000-0000-000000000003",`+profilesHTTPUpdated+
				`,"attributes":{"amount":9007199254740993.0000000001,"document":{"v":null}}}],`+
				`"schema":{"kind":"object","properties":[`+
				`{"name":"label","type":{"kind":"string"},"readOptional":true},`+
				`{"name":"amount","type":{"kind":"decimal","precision":38,"scale":10},"readOptional":true},`+
				`{"name":"document","type":{"kind":"json"},"readOptional":true}]},"total":1}`)
	})

}
