// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
)

type profileMCPResponse struct {
	Result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StructuredContent json.RawMessage `json:"structuredContent"`
		IsError           bool            `json:"isError"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// TestProfilesMCP uses the real MCP transport, core and isolated PostgreSQL.
func TestProfilesMCP(t *testing.T) {

	if testing.Short() {
		t.Skip("requires PostgreSQL, NATS and the server")
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.SetProfileSchema(types.Object([]types.Property{
		{Name: "email", Type: types.String(), ReadOptional: true},
		{Name: "country", Type: types.String(), ReadOptional: true},
		{Name: "account", Type: types.Int(64), ReadOptional: true},
		{Name: "bio", Type: types.String(), ReadOptional: true},
	}))
	k.Start()
	defer k.Stop()
	k.UpdateIdentityResolutionSettings(false, nil)
	var settings krenalistester.DBSettings
	err := json.Unmarshal(krenalistester.PostgresWarehouseSettings(), &settings)
	if err != nil {
		t.Fatalf("expected warehouse settings, got %v", err)
	}
	pool, err := krenalistester.ConnectionPool(t.Context(), &settings)
	if err != nil {
		t.Fatalf("expected isolated warehouse, got %v", err)
	}
	defer pool.Close()

	// Publish one coherent view, with the asynchronous writer and IR inactive.
	// The MCP role can discover countries but cannot read profile email/account
	// or event properties. query-profiles must still use the main connection.
	_, err = pool.Exec(t.Context(), `INSERT INTO krenalis_profiles_0
 (_kpid,_updated_at,email,country,account,bio,_anonymous_count,_recognized_count)
 SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid,'2026-02-01 12:00:00.123456',
 'profile-'||n||'@example.com',CASE WHEN n IN (1,3) THEN 'IT' ELSE 'FR' END,
 CASE WHEN n=1 THEN 9007199254740993 ELSE n END,'short',0,1 FROM generate_series(1,4) n;
INSERT INTO krenalis_events
 (kpid,message_id,event,timestamp,received_at,sent_at,type,connection_id,anonymous_id,traits,properties)
 SELECT _kpid,'e'||n,'Purchase','2026-01-15','2026-02-01','2026-02-01','track','fixture','a'||n,'{}',
 CASE n WHEN 2 THEN '{"currency":"EUR"}' WHEN 3 THEN '{"currency":null,"amount":9007199254740993}'
 ELSE '{"amount":"9007199254740993"}' END::jsonb
 FROM (SELECT _kpid,right(_kpid::text,1)::int n FROM profiles) p WHERE n>1;
INSERT INTO krenalis_events
 (kpid,message_id,event,timestamp,received_at,sent_at,type,connection_id,anonymous_id,traits)
 SELECT _kpid,'non-track/'||event_type||'/'||_kpid,NULL,'2026-01-15','2026-02-01','2026-02-01',
 event_type::event_type,'fixture','a','{}'
 FROM profiles CROSS JOIN (VALUES ('identify'),('page'),('screen'),('group'),('alias')) types(event_type);
CREATE ROLE mcp_profile_reader LOGIN PASSWORD 'mcp-profile-fixture';
GRANT USAGE ON SCHEMA public TO mcp_profile_reader;
GRANT SELECT ON krenalis_profile_schema_versions TO mcp_profile_reader;
GRANT SELECT (_kpid,country) ON profiles TO mcp_profile_reader;`)
	if err != nil {
		t.Fatalf("expected coherent fixture and restricted MCP role, got %v", err)
	}
	var key struct {
		Token string `json:"token"`
	}
	k.Call("POST", "/v1/keys", nil, map[string]any{"name": "Profile MCP test",
		"workspace": k.WorkspaceID(), "type": "MCP"}, &key)
	token := key.Token
	session := ""
	call := func(t *testing.T, method string, params any) profileMCPResponse {
		t.Helper()
		return profileMCPCall(t, k.Addr(), token, &session, method, params)
	}
	call(t, "initialize", map[string]any{"protocolVersion": "2024-11-05", "capabilities": map[string]any{},
		"clientInfo": map[string]any{"name": "Profile MCP test", "version": "1"}})
	call(t, "notifications/initialized", map[string]any{})
	tool := func(t *testing.T, name string, args any) profileMCPResponse {
		t.Helper()
		return call(t, "tools/call", map[string]any{"name": name, "arguments": args})
	}
	assertToolError := func(t *testing.T, result profileMCPResponse, code string) {
		t.Helper()
		if result.Error != nil || !result.Result.IsError || len(result.Result.Content) != 1 ||
			!strings.HasPrefix(result.Result.Content[0].Text, code+":") {
			t.Fatalf("expected actionable %s error without data, got %+v", code, result)
		}
	}
	assertResult := func(t *testing.T, result profileMCPResponse) string {
		t.Helper()
		if result.Error != nil || result.Result.IsError || len(result.Result.Content) != 1 ||
			result.Result.Content[0].Type != "text" {
			t.Fatalf("expected successful complete text result, got %+v", result)
		}
		return result.Result.Content[0].Text
	}
	assertToolError(t, tool(t, "query-profiles", map[string]any{}), "MCPNotConfigured")

	// Metadata discovery needs only the scoped MCP key, not warehouse MCP settings.
	catalog := assertResult(t, tool(t, "list-event-definitions", map[string]any{"query": "ORDER COMPLETED"}))
	var discovery struct {
		Definitions []struct {
			ID    string `json:"id"`
			Event string `json:"event"`
		} `json:"definitions"`
	}
	err = json.Unmarshal([]byte(catalog), &discovery)
	if err != nil {
		t.Fatalf("expected catalog JSON, got %v", err)
	}
	orderEvent := ""
	orderID := ""
	for _, definition := range discovery.Definitions {
		if definition.ID == "commerce.order_completed" {
			orderEvent, orderID = definition.Event, definition.ID
		}
	}
	if orderEvent != "Order Completed" {
		t.Fatalf("expected Order Completed discovery, got %s", catalog)
	}
	definitionResult := tool(t, "get-event-definition", map[string]any{"id": orderID})
	detail := assertResult(t, definitionResult)
	if len(definitionResult.Result.StructuredContent) == 0 {
		t.Fatal("expected structured catalog content, got absent")
	}
	var textObject, structuredObject map[string]any
	err = json.Unmarshal([]byte(detail), &textObject)
	if err != nil {
		t.Fatalf("expected catalog text JSON, got %v", err)
	}
	err = json.Unmarshal(definitionResult.Result.StructuredContent, &structuredObject)
	if err != nil {
		t.Fatalf("expected structured catalog JSON, got %v", err)
	}
	if !reflect.DeepEqual(textObject, structuredObject) {
		t.Fatal("expected equivalent text and structured content, got different JSON values")
	}
	if !strings.Contains(detail, `"semantics":"money"`) || !strings.Contains(detail, `"name":"products"`) {
		t.Fatalf("expected native money and product schemas, got %s", detail)
	}
	for _, test := range []struct {
		name string
		tool string
		args any
		code string
	}{
		{"missing ID", "get-event-definition", map[string]any{}, "InvalidArguments"},
		{"unknown ID", "get-event-definition", map[string]any{"id": "missing"}, "EventDefinitionNotExist"},
		{"numeric ID", "get-event-definition", map[string]any{"id": 42}, "InvalidArguments"},
		{"null ID", "get-event-definition", map[string]any{"id": nil}, "InvalidArguments"},
		{"unknown argument", "list-event-definitions", map[string]any{"id": "x"}, "InvalidArguments"},
		{"null query", "list-event-definitions", map[string]any{"query": nil}, "InvalidArguments"},
		{"long query", "list-event-definitions", map[string]any{"query": strings.Repeat("x", 257)}, "InvalidArguments"},
		{"oversize arguments", "list-event-definitions", map[string]any{"query": strings.Repeat("x", 2048)}, "InvalidArguments"},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertToolError(t, tool(t, test.tool, test.args), test.code)
		})
	}
	emptyCatalog := assertResult(t, tool(t, "list-event-definitions", map[string]any{"query": "no-such-definition"}))
	if !strings.Contains(emptyCatalog, `"definitions":[]`) {
		t.Fatalf("expected an empty definitions array, got %s", emptyCatalog)
	}
	guide := assertResult(t, tool(t, "profile-filter-doc", map[string]any{}))
	if !strings.Contains(guide, "table, column") || !strings.Contains(guide, "main virtual warehouse") {
		t.Fatalf("expected explicit access and Snowflake documentation, got absent")
	}
	mcpSettings := settings
	mcpSettings.Username, mcpSettings.Password = "mcp_profile_reader", "mcp-profile-fixture"
	k.Call("PUT", "/v1/warehouse", nil, map[string]any{"mode": "Normal", "settings": settings,
		"mcpSettings": mcpSettings}, nil)

	// Use the discovered technical name, not the catalog ID, in an actual Count.
	_, err = pool.Exec(t.Context(), `INSERT INTO krenalis_events
 (kpid,message_id,event,timestamp,received_at,sent_at,type,connection_id,anonymous_id,traits,properties)
 SELECT _kpid,'catalog-'||n,'Order Completed','2026-01-15','2026-02-01','2026-02-01',
 'track','fixture','a','{}','{}' FROM profiles CROSS JOIN generate_series(1,2) n
 WHERE email='profile-1@example.com'`)
	if err != nil {
		t.Fatalf("expected recorded order fixtures, got %v", err)
	}
	orderFilter, err := json.Marshal(map[string]any{"operator": "and", "rules": []any{map[string]any{
		"count": map[string]any{"source": map[string]any{"kind": "recordedEvents",
			"name": orderEvent, "grain": "registered-message"},
			"period": map[string]any{"kind": "elapsedLookback", "duration": map[string]any{"value": 30, "unit": "day"}}},
		"operator": "is greater than or equal to", "values": []int{2},
	}}})
	if err != nil {
		t.Fatalf("expected a filter using the discovered event, got %v", err)
	}
	orders := assertResult(t, tool(t, "query-profiles", map[string]any{"filter": string(orderFilter),
		"properties": []string{"email"}, "at": "2026-02-01T00:00:00Z"}))
	if !strings.Contains(orders, `"total":1`) || !strings.Contains(orders, `"email":"profile-1@example.com"`) ||
		!strings.Contains(orders, `"match":1,"noMatch":3,"unknown":0`) {
		t.Fatalf("expected one matching profile from two recorded orders, got %s", orders)
	}
	defaults := assertResult(t, tool(t, "query-profiles", map[string]any{}))
	if !strings.Contains(defaults, `"total":4`) || strings.Contains(defaults, `"evaluation":`) {
		t.Fatalf("expected default Record listing without evaluation, got %s", defaults)
	}
	invalidToken := "mcp_" + strings.Repeat("1", len(token)-4)
	rejected := profileMCPCall(t, k.Addr(), invalidToken, &session, "tools/call",
		map[string]any{"name": "query-profiles", "arguments": map[string]any{}})
	if rejected.Error == nil || len(rejected.Result.Content) != 0 {
		t.Fatalf("expected invalid-key rejection without data, got %+v", rejected)
	}
	for _, name := range []string{"list-event-definitions", "get-event-definition"} {
		rejected := profileMCPCall(t, k.Addr(), invalidToken, &session, "tools/call",
			map[string]any{"name": name, "arguments": map[string]any{}})
		if rejected.Error == nil || len(rejected.Result.Content) != 0 {
			t.Fatalf("expected catalog authentication failure without data, got %+v", rejected)
		}
	}

	const count = `{"operator":"and","rules":[{"count":{"source":{"kind":"recordedEvents",` +
		`"name":"Purchase","grain":"registered-message"},"period":{"kind":"elapsedLookback",` +
		`"duration":{"value":31,"unit":"day"}},"where":{"operator":"and","rules":[` +
		`{"property":"properties.currency","operator":"is","values":["EUR"]}]}},"operator":"is","values":[0]}]}`
	const at = "2026-02-01T00:00:00Z"
	calendarCount := strings.Replace(count,
		`{"kind":"elapsedLookback","duration":{"value":31,"unit":"day"}}`,
		`{"kind":"lastCompleteCalendarPeriods","unit":"month","count":1}`, 1)
	calendarCount = strings.TrimSuffix(calendarCount, `}`) +
		`,"defaultCalendar":{"definition":"gregorian-monday/Europe/Rome","meaningRevision":"1"}}`
	const record = `{"operator":"and","rules":[{"property":"country","operator":"is","values":["IT"]}]}`
	recordCalendar := strings.TrimSuffix(record, `}`) +
		`,"defaultCalendar":{"definition":"gregorian-monday/Europe/Rome","meaningRevision":"1"}}`
	var recordResult string
	for _, test := range []struct {
		name   string
		filter string
		at     string
		first  int
	}{
		{"Record", record, "", 0},
		{"Record calendar", recordCalendar, "", 0},
		{"exact Int64", `{"operator":"and","rules":[{"property":"account","operator":"is","values":[9007199254740993]}]}`, "", 0},
		{"Count first page", count, at, 0},
		{"Count next page", count, at, 1},
		{"mixed Record and Count", strings.Replace(count, `"rules":[{"count":`, `"rules":[{"property":"country","operator":"is","values":["FR"]},{"count":`, 1), at, 0},
		{"Count empty page", count, at, 99},
		{"Count calendar month", calendarCount, at, 0},
		{"Count calendar empty page", calendarCount, at, 99},
		{"exact JSON number", strings.Replace(count, `"property":"properties.currency","operator":"is","values":["EUR"]`, `"property":"properties.amount","operator":"is","values":[9007199254740993]`, 1), at, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			args := map[string]any{"filter": test.filter, "properties": []string{"email", "account"},
				"order": "email", "limit": 1, "first": test.first}
			q := url.Values{"filter": {test.filter}, "properties": {"email", "account"},
				"order": {"email"}, "limit": {"1"}, "first": {strconv.Itoa(test.first)}}
			if test.at != "" {
				args["at"] = test.at
				q.Set("at", test.at)
			}
			var expected json.RawMessage
			k.Call("GET", "/v1/profiles?"+q.Encode(), nil, nil, &expected)
			actual := assertResult(t, tool(t, "query-profiles", args))
			if actual != string(expected) {
				t.Fatalf("expected HTTP-identical profile result %s, got %s", expected, actual)
			}
			if test.name == "Record" {
				recordResult = actual
			}
			if test.name == "Record calendar" && (actual != recordResult || strings.Contains(actual, `"evaluation":`)) {
				t.Fatalf("expected byte-identical Record response %s without evaluation, got %s", recordResult, actual)
			}
			if test.name == "exact Int64" && !strings.Contains(actual, `"account":"9007199254740993"`) {
				t.Fatalf("expected exact Int64 profile operand and result, got %s", actual)
			}
			if strings.HasPrefix(test.name, "Count") &&
				!strings.Contains(actual, `"match":2,"noMatch":1,"unknown":1`) {
				t.Fatalf("expected ternary totals 2/1/1, got %s", actual)
			}
			if strings.HasSuffix(test.name, "empty page") && !strings.HasPrefix(actual, `{"profiles":[]`) {
				t.Fatalf("expected empty array with retained totals, got %s", actual)
			}
		})
	}

	// The SQL tool still uses the restricted connection, unlike query-profiles.
	assertResult(t, tool(t, "query-data-warehouse", map[string]any{"query": "SELECT country FROM profiles"}))
	denied := tool(t, "query-data-warehouse", map[string]any{"query": "SELECT email FROM profiles"})
	if denied.Error == nil || len(denied.Result.Content) != 0 {
		t.Fatalf("expected SQL permission failure without data, got %+v", denied)
	}

	for _, test := range []struct {
		name string
		args any
		code string
	}{
		{"filter object", map[string]any{"filter": map[string]any{"operator": "and", "rules": []any{}}}, "InvalidArguments"},
		{"filter null", map[string]any{"filter": nil}, "InvalidArguments"},
		{"null text", map[string]any{"filter": "null"}, "InvalidArguments"},
		{"empty filter", map[string]any{"filter": ""}, "InvalidArguments"},
		{"unknown argument", map[string]any{"query": "SELECT 1"}, "InvalidArguments"},
		{"workspace override", map[string]any{"workspace": "another-workspace"}, "InvalidArguments"},
		{"unknown property", map[string]any{"properties": []string{"notPresent"}}, "PropertyNotExist"},
		{"null limit", map[string]any{"limit": nil}, "InvalidArguments"},
		{"fractional limit", map[string]any{"limit": 1.5}, "InvalidArguments"},
		{"zero limit", map[string]any{"limit": 0}, "InvalidArguments"},
		{"excess limit", map[string]any{"limit": 1001}, "InvalidArguments"},
		{"negative offset", map[string]any{"first": -1}, "InvalidArguments"},
		{"excess offset", map[string]any{"first": int64(2147483648)}, "InvalidArguments"},
		{"empty at", map[string]any{"filter": count, "at": ""}, "InvalidArguments"},
		{"nanosecond at", map[string]any{"filter": count, "at": "2026-02-01T00:00:00.000000001Z"}, "InvalidArguments"},
		{"at without Count", map[string]any{"at": at}, "InvalidArguments"},
		{"invalid filter", map[string]any{"filter": strings.Replace(count, `"values":[0]`, `"values":[-1]`, 1)}, "InvalidArguments"},
		{"invalid Record calendar", map[string]any{"filter": strings.Replace(recordCalendar, `"meaningRevision":"1"`, `"meaningRevision":"2"`, 1)}, "InvalidArguments"},
		{"inverted exact interval", map[string]any{"filter": strings.Replace(count, `"operator":"is","values":[0]`, `"operator":"is between","values":[9007199254740993,9007199254740992]`, 1)}, "InvalidArguments"},
		{"unsupported range", map[string]any{"filter": strings.Replace(count, `"values":[0]`, `"values":[9223372036854775808]`, 1)}, "FilterCapabilityNotImplemented"},
		{"long natural", map[string]any{"filter": strings.Replace(count, `"values":[0]`, `"values":[`+strings.Repeat("9", 10000)+`]`, 1)}, "FilterCapabilityNotImplemented"},
		{"oversize arguments", map[string]any{"filter": strings.Repeat(" ", 500*1024)}, "InvalidArguments"},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertToolError(t, tool(t, "query-profiles", test.args), test.code)
		})
	}

	// Exceed the smaller MCP budget, with no truncation or Record/SQL fallback.
	_, err = pool.Exec(t.Context(), "UPDATE krenalis_profiles_0 SET bio=repeat('x',140000)")
	if err != nil {
		t.Fatalf("expected large profile fixture, got %v", err)
	}
	for _, filter := range []string{"", count} {
		args := map[string]any{"properties": []string{"email", "bio"}, "limit": 1}
		if filter != "" {
			args["filter"], args["at"] = filter, at
		}
		assertToolError(t, tool(t, "query-profiles", args), "ProfileResultTooLarge")
		args["properties"] = []string{"email"}
		assertResult(t, tool(t, "query-profiles", args))
	}

	// Removing the configuration revokes subsequent admissions. Metadata remains
	// accessible, and SQL retains its own missing-configuration behavior.
	k.Call("PUT", "/v1/warehouse", nil, map[string]any{"mode": "Normal", "settings": settings}, nil)
	assertToolError(t, tool(t, "query-profiles", map[string]any{"filter": count}), "MCPNotConfigured")
	assertResult(t, tool(t, "profile-filter-doc", map[string]any{}))
	k.Call("PUT", "/v1/warehouse", nil, map[string]any{"mode": "Normal", "settings": settings,
		"mcpSettings": mcpSettings}, nil)

	// A failed query is an opaque operation failure, never UNKNOWN or partial data.
	_, err = pool.Exec(t.Context(), "ALTER TABLE krenalis_events RENAME TO hidden_event_fixture")
	if err != nil {
		t.Fatalf("expected failing source fixture, got %v", err)
	}
	failed := tool(t, "query-profiles", map[string]any{"filter": count, "properties": []string{"email"}, "at": at})
	if failed.Error == nil || len(failed.Result.Content) != 0 ||
		failed.Error.Message != "cmd/internal/mcp: profile query failed" {
		t.Fatalf("expected opaque operation failure without data, got %+v", failed)
	}
	_, err = pool.Exec(t.Context(), "ALTER VIEW profiles RENAME TO hidden_profile_fixture")
	if err != nil {
		t.Fatalf("expected failing Record fixture, got %v", err)
	}
	failed = tool(t, "query-profiles", map[string]any{"properties": []string{"email"}})
	if failed.Error == nil || len(failed.Result.Content) != 0 ||
		failed.Error.Message != "cmd/internal/mcp: profile query failed" {
		t.Fatalf("expected opaque Record failure without warehouse details, got %+v", failed)
	}

}

// profileMCPCall drives the actual server transport with its scoped MCP token.
func profileMCPCall(t *testing.T, address, token string, session *string, method string, params any) profileMCPResponse {

	t.Helper()
	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": "profiles-test", "method": method,
		"params": params})
	if err != nil {
		t.Fatalf("expected JSON-RPC request, got %v", err)
	}
	request, err := http.NewRequestWithContext(t.Context(), "POST", "http://"+address+"/mcp", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("expected MCP request, got %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	if *session != "" {
		request.Header.Set("Mcp-Session-Id", *session)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("expected MCP response, got %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", response.StatusCode)
	}
	if value := response.Header.Get("Mcp-Session-Id"); value != "" {
		*session = value
	}

	var result profileMCPResponse
	err = json.NewDecoder(io.LimitReader(response.Body, 1024*1024)).Decode(&result)
	if err != nil {
		t.Fatalf("expected JSON-RPC response, got %v", err)
	}

	return result
}
