// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"bytes"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/errors"
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
	assertToolError(t, tool(t, "explain-profile-filter", map[string]any{}), "MCPNotConfigured")

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
	historyCount := strings.Replace(count,
		`{"kind":"elapsedLookback","duration":{"value":31,"unit":"day"}}`,
		`{"kind":"fullRelevantHistory","history":{"definition":"registered-queryable-events","meaningRevision":"1"}}`, 1)
	for i, expected := range []string{"MATCH", "NO_MATCH", "MATCH", "UNKNOWN"} {
		t.Run("explanation "+strconv.Itoa(i+1), func(t *testing.T) {
			profile := "00000000-0000-0000-0000-00000000000" + strconv.Itoa(i+1)
			var httpResult json.RawMessage
			k.Call("GET", "/v1/profiles/"+profile+"/filter-explanation?filter="+url.QueryEscape(count)+
				"&at="+url.QueryEscape(at), nil, nil, &httpResult)
			actual := assertResult(t, tool(t, "explain-profile-filter",
				map[string]any{"kpid": profile, "filter": count, "at": at}))
			if actual != string(httpResult) {
				t.Fatalf("expected HTTP-identical explanation %s, got %s", httpResult, actual)
			}
			var explanation struct {
				KPID string `json:"kpid"`
				Rule struct {
					Result string `json:"result"`
					Rules  []struct {
						Result string `json:"result"`
						Count  struct {
							Certain, Uncertain int
							Start, End         string
						} `json:"count"`
					} `json:"rules"`
				} `json:"rule"`
			}
			err := json.Unmarshal([]byte(actual), &explanation)
			if err != nil {
				t.Fatalf("expected decoded explanation, got %v", err)
			}
			if explanation.KPID != profile || explanation.Rule.Result != expected || len(explanation.Rule.Rules) != 1 {
				t.Fatalf("expected explained %s for profile %s, got %s", expected, profile, actual)
			}
			leaf := explanation.Rule.Rules[0]
			lo, uncertain := 0, 0
			if i == 1 {
				lo = 1
			}
			if i == 3 {
				uncertain = 1
			}
			if leaf.Result != expected || leaf.Count.Certain != lo || leaf.Count.Uncertain != uncertain ||
				leaf.Count.Start != "2026-01-01T00:00:00Z" || leaf.Count.End != at {
				t.Fatalf("expected Count %d certain and %d uncertain in January, got %s", lo, uncertain, actual)
			}
			if strings.Contains(actual, `"history":`) || strings.Contains(actual, `"unboundedStart":`) {
				t.Fatalf("expected unchanged bounded explanation without history metadata, got %s", actual)
			}
		})
	}
	for _, test := range []struct {
		name, code string
		args       any
	}{
		{"missing profile", "ProfileNotFound", map[string]any{"kpid": "00000000-0000-0000-0000-000000000099", "filter": count, "at": at}},
		{"missing filter", "InvalidArguments", map[string]any{"kpid": "00000000-0000-0000-0000-000000000001"}},
		{"invalid KPID", "InvalidArguments", map[string]any{"kpid": "bad", "filter": count}},
		{"filter object", "InvalidArguments", map[string]any{"kpid": "00000000-0000-0000-0000-000000000001", "filter": map[string]any{}}},
		{"filter null", "InvalidArguments", map[string]any{"kpid": "00000000-0000-0000-0000-000000000001", "filter": "null"}},
		{"null at", "InvalidArguments", map[string]any{"kpid": "00000000-0000-0000-0000-000000000001", "filter": count, "at": nil}},
		{"empty at", "InvalidArguments", map[string]any{"kpid": "00000000-0000-0000-0000-000000000001", "filter": count, "at": ""}},
		{"unknown argument", "InvalidArguments", map[string]any{"kpid": "00000000-0000-0000-0000-000000000001", "filter": count, "sql": "SELECT 1"}},
	} {
		t.Run("explanation "+test.name, func(t *testing.T) {
			assertToolError(t, tool(t, "explain-profile-filter", test.args), test.code)
		})
	}
	t.Run("history explanation", func(t *testing.T) {
		profile := "00000000-0000-0000-0000-000000000004"
		var expected json.RawMessage
		k.Call("GET", "/v1/profiles/"+profile+"/filter-explanation?filter="+url.QueryEscape(historyCount)+
			"&at="+url.QueryEscape(at), nil, nil, &expected)
		actual := assertResult(t, tool(t, "explain-profile-filter",
			map[string]any{"kpid": profile, "filter": historyCount, "at": at}))
		if actual != string(expected) || !strings.Contains(actual, `"unboundedStart":true`) ||
			strings.Contains(actual, `"start":`) || !strings.Contains(actual, `"end":"`+at+`"`) ||
			!strings.Contains(actual, `"result":"UNKNOWN"`) || !strings.Contains(actual, `"uncertain":1`) ||
			!strings.Contains(actual, `"history":{"definition":"registered-queryable-events","meaningRevision":"1"}`) {
			t.Fatalf("expected HTTP-identical UNKNOWN history explanation without a lower date, got %s", actual)
		}
	})
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
		{"Count recorded history", historyCount, at, 0},
		{"Count recorded history empty page", historyCount, at, 99},
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

	t.Run("page and screen Count", func(t *testing.T) {

		_, err := pool.Exec(t.Context(), `INSERT INTO krenalis_events
(kpid,message_id,type,name,timestamp,received_at,sent_at,connection_id,anonymous_id,traits,properties,
 context_page_path)
 VALUES
 ('00000000-0000-0000-0000-000000000001','visit-mcp/1','page','Pricing','2026-01-15',
 '2026-02-01','2026-02-01','fixture','a','{}','{}','/pricing'),
 ('00000000-0000-0000-0000-000000000002','visit-mcp/2','page','pricing','2026-01-15',
 '2026-02-01','2026-02-01','fixture','a','{}','{}','/pricing'),
 ('00000000-0000-0000-0000-000000000002','visit-mcp/empty','page','','2026-01-15',
 '2026-02-01','2026-02-01','fixture','a','{}','{}','/pricing/'),
 ('00000000-0000-0000-0000-000000000004','visit-mcp/4','screen','Pricing','2026-01-15',
 '2026-02-01','2026-02-01','fixture','a','{}','{}',NULL)`)
		if err != nil {
			t.Fatalf("expected page/screen MCP fixtures, got %v", err)
		}
		defer func() {
			_, err := pool.Exec(t.Context(), "DELETE FROM krenalis_events WHERE message_id LIKE 'visit-mcp/%'")
			if err != nil {
				t.Errorf("expected page/screen fixture cleanup, got %v", err)
			}
		}()

		const pages = `{"operator":"and","rules":[{"count":{"source":{"kind":"recordedEvents",` +
			`"type":"page","grain":"registered-message"},"period":{"kind":"elapsedLookback",` +
			`"duration":{"value":31,"unit":"day"}}},"operator":"is greater than or equal to","values":[1]}]}`
		named := strings.Replace(pages, `"type":"page"`, `"type":"page","name":"Pricing"`, 1)
		named = strings.Replace(named, `"operator":"is greater than or equal to","values":[1]`,
			`"operator":"is","values":[0]`, 1)
		raw := strings.Replace(pages, `"duration":{"value":31,"unit":"day"}}`,
			`"duration":{"value":31,"unit":"day"}},"where":{"operator":"and","rules":[`+
				`{"property":"name","operator":"is","values":["Pricing"]}]}`, 1)
		raw = strings.Replace(raw, `"operator":"is greater than or equal to","values":[1]`,
			`"operator":"is","values":[0]`, 1)
		path := strings.Replace(pages, `"duration":{"value":31,"unit":"day"}}`,
			`"duration":{"value":31,"unit":"day"}},"where":{"operator":"and","rules":[`+
				`{"property":"context.page.path","operator":"is","values":["/pricing"]}]}`, 1)
		screens := strings.Replace(pages, `"type":"page"`, `"type":"screen"`, 1)
		namedScreen := strings.Replace(screens, `"type":"screen"`, `"type":"screen","name":"Pricing"`, 1)
		for _, test := range []struct{ name, filter, totals string }{
			{"all pages", pages, `"match":4,"noMatch":0,"unknown":0`},
			{"named page absence", named, `"match":3,"noMatch":1,"unknown":0`},
			{"raw name absence", raw, `"match":0,"noMatch":1,"unknown":3`},
			{"path qualification", path, `"match":2,"noMatch":0,"unknown":2`},
			{"all screens", screens, `"match":4,"noMatch":0,"unknown":0`},
			{"named screen", namedScreen, `"match":1,"noMatch":3,"unknown":0`},
		} {
			t.Run(test.name, func(t *testing.T) {
				for _, first := range []int{0, 99} {
					var expected json.RawMessage
					k.Call("GET", "/v1/profiles?filter="+url.QueryEscape(test.filter)+"&at="+
						url.QueryEscape(at)+"&first="+strconv.Itoa(first), nil, nil, &expected)
					actual := assertResult(t, tool(t, "query-profiles", map[string]any{
						"filter": test.filter, "at": at, "first": first,
					}))
					if actual != string(expected) || !strings.Contains(actual, test.totals) ||
						first == 99 && !strings.HasPrefix(actual, `{"profiles":[]`) {
						t.Fatalf("expected identical HTTP/MCP result with %s, got %s", test.totals, actual)
					}
				}
			})
		}

		for _, test := range []struct{ name, filter, result, counts string }{
			{"named source", named, "MATCH", `"certain":0,"uncertain":0`},
			{"raw name where", raw, "UNKNOWN", `"certain":0,"uncertain":1`},
		} {
			t.Run("explain "+test.name, func(t *testing.T) {
				profile := "00000000-0000-0000-0000-000000000002"
				var expected json.RawMessage
				k.Call("GET", "/v1/profiles/"+profile+"/filter-explanation?filter="+url.QueryEscape(test.filter)+
					"&at="+url.QueryEscape(at), nil, nil, &expected)
				actual := assertResult(t, tool(t, "explain-profile-filter", map[string]any{
					"kpid": profile, "filter": test.filter, "at": at,
				}))
				if actual != string(expected) || !strings.Contains(actual, `"result":"`+test.result+`"`) ||
					!strings.Contains(actual, test.counts) {
					t.Fatalf("expected HTTP-identical %s with %s, got %s", test.result, test.counts, actual)
				}
			})
		}

	})

	t.Run("Count currency", func(t *testing.T) {

		_, err := pool.Exec(t.Context(), `UPDATE krenalis_events SET properties='{"currency":"USD"}'
WHERE event='Order Completed';
INSERT INTO krenalis_events
(kpid,message_id,event,timestamp,received_at,sent_at,type,connection_id,anonymous_id,traits,properties) VALUES
('00000000-0000-0000-0000-000000000002','native-mcp/2','Order Completed','2026-01-15',
'2026-02-01','2026-02-01','track','fixture','a','{}','{"currency":"EUR","coupon":"PROMO"}'),
('00000000-0000-0000-0000-000000000003','native-mcp/3','Order Completed','2026-01-15',
'2026-02-01','2026-02-01','track','fixture','a','{}','{"currency":null,"coupon":"PROMO"}'),
('00000000-0000-0000-0000-000000000004','native-mcp/4','Order Completed','2026-01-15',
'2026-02-01','2026-02-01','track','fixture','a','{}','{"coupon":"OTHER"}')`)
		if err != nil {
			t.Fatalf("expected governed currency fixture, got %v", err)
		}
		defer func() {
			_, err := pool.Exec(t.Context(), `DELETE FROM krenalis_events WHERE message_id LIKE 'native-mcp/%';
UPDATE krenalis_events SET properties='{}' WHERE event='Order Completed'`)
			if err != nil {
				t.Errorf("expected restored catalog fixture, got %v", err)
			}
		}()

		const currencyFields = `"currencyField":{"definition":"commerce.order_completed.currency",` +
			`"meaningRevision":"1"},"nativeCurrency":"EUR"`
		const native = `{"operator":"and","rules":[{"count":{"source":{"kind":"recordedEvents",` +
			`"name":"Order Completed","grain":"registered-message"},"period":{"kind":"elapsedLookback",` +
			`"duration":{"value":30,"unit":"day"}},` + currencyFields + `},"operator":"is","values":[0]}]}`
		raw := strings.Replace(native, currencyFields, `"where":{"operator":"and","rules":[`+
			`{"property":"properties.currency","operator":"is","values":["EUR"]}]}`, 1)
		qualified := strings.Replace(native, `"nativeCurrency":"EUR"`,
			`"nativeCurrency":"EUR","where":{"operator":"and","rules":[`+
				`{"property":"properties.coupon","operator":"is","values":["PROMO"]}]}`, 1)
		minimum := strings.Replace(native, `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","minimum":2`, 1)
		minimum = strings.Replace(minimum, `"operator":"is","values":[0]`,
			`"operator":"is greater than or equal to","values":[1]`, 1)
		for _, test := range []struct{ name, filter, totals string }{
			{"governed", native, `"match":1,"noMatch":1,"unknown":2`},
			{"raw", raw, `"match":2,"noMatch":1,"unknown":1`},
			{"independent where", qualified, `"match":2,"noMatch":1,"unknown":1`},
			{"unreachable minimum", minimum, `"match":0,"noMatch":4,"unknown":0`},
		} {
			t.Run(test.name, func(t *testing.T) {
				for _, first := range []int{0, 99} {
					var expected json.RawMessage
					path := "/v1/profiles?filter=" + url.QueryEscape(test.filter) +
						"&at=" + url.QueryEscape(at) + "&first=" + strconv.Itoa(first)
					k.Call("GET", path, nil, nil, &expected)
					actual := assertResult(t, tool(t, "query-profiles", map[string]any{
						"filter": test.filter, "at": at, "first": first,
					}))
					if actual != string(expected) || !strings.Contains(actual, test.totals) ||
						first == 99 && !strings.HasPrefix(actual, `{"profiles":[]`) {
						t.Fatalf("expected identical HTTP/MCP result with %s, got %s", test.totals, actual)
					}
				}
			})
		}

		for _, test := range []struct{ name, filter, profile, truth, counts string }{
			{"governed null", native, "3", "UNKNOWN", `"certain":0,"uncertain":1`},
			{"governed missing", native, "4", "UNKNOWN", `"certain":0,"uncertain":1`},
			{"raw null", raw, "3", "MATCH", `"certain":0,"uncertain":0`},
			{"raw missing", raw, "4", "UNKNOWN", `"certain":0,"uncertain":1`},
			{"false where", qualified, "4", "MATCH", `"certain":0,"uncertain":0`},
		} {
			t.Run("explain "+test.name, func(t *testing.T) {
				profile := "00000000-0000-0000-0000-00000000000" + test.profile
				var expected json.RawMessage
				path := "/v1/profiles/" + profile + "/filter-explanation?filter=" +
					url.QueryEscape(test.filter) + "&at=" + url.QueryEscape(at)
				k.Call("GET", path, nil, nil, &expected)
				actual := assertResult(t, tool(t, "explain-profile-filter", map[string]any{
					"kpid": profile, "filter": test.filter, "at": at,
				}))
				if actual != string(expected) || !strings.Contains(actual, `"result":"`+test.truth+`"`) ||
					!strings.Contains(actual, test.counts) {
					t.Fatalf("expected HTTP-identical %s with %s, got %s", test.truth, test.counts, actual)
				}
				if strings.HasPrefix(test.name, "raw") {
					if strings.Contains(actual, `"currencyField":`) || strings.Contains(actual, `"nativeCurrency":`) {
						t.Fatalf("expected unchanged raw explanation without currency metadata, got %s", actual)
					}
					return
				}
				if !strings.Contains(actual, currencyFields) {
					t.Fatalf("expected resolved currency reference and literal, got %s", actual)
				}
			})
		}

		for _, test := range []struct{ name, filter, code string }{
			{"unresolved field", strings.Replace(native, `commerce.order_completed.currency`, `unknown`, 1), "InvalidArguments"},
			{"unresolved revision", strings.Replace(native, `"meaningRevision":"1"`, `"meaningRevision":"2"`, 1), "InvalidArguments"},
			{"missing field", strings.Replace(native, currencyFields, `"nativeCurrency":"EUR"`, 1), "InvalidArguments"},
			{"null currency", strings.Replace(native, `"nativeCurrency":"EUR"`, `"nativeCurrency":null`, 1), "InvalidArguments"},
			{"different source", strings.Replace(native, `Order Completed`, `Refund`, 1), "InvalidArguments"},
			{"page with same name", strings.Replace(native, `"kind":"recordedEvents"`, `"kind":"recordedEvents","type":"page"`, 1), "InvalidArguments"},
			{"screen with same name", strings.Replace(native, `"kind":"recordedEvents"`, `"kind":"recordedEvents","type":"screen"`, 1), "InvalidArguments"},
			{"dependent where", strings.Replace(qualified, `properties.coupon`, `properties.currency`, 1), "FilterCapabilityNotImplemented"},
		} {
			t.Run(test.name, func(t *testing.T) {
				assertToolError(t, tool(t, "query-profiles", map[string]any{
					"filter": test.filter, "at": at,
				}), test.code)
				assertToolError(t, tool(t, "explain-profile-filter", map[string]any{
					"kpid": "00000000-0000-0000-0000-000000000003", "filter": test.filter, "at": at,
				}), test.code)
			})
		}

	})

	t.Run("Experience", func(t *testing.T) {

		_, err := pool.Exec(t.Context(), `INSERT INTO krenalis_events
(kpid,message_id,event,timestamp,received_at,sent_at,type,connection_id,anonymous_id,traits,properties)
SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid,'experience/'||id,
'Order Completed',stamp::timestamp,'2026-02-01','2026-02-01','track','fixture','a','{}',payload::jsonb
FROM (VALUES
(1,'historical','2025-12-15','{"experience_fixture":true,"q":"yes","v":"yes"}'),
(1,'old','2026-01-01','{"experience_fixture":true,"q":"yes","v":"no"}'),
(1,'new','2026-01-02','{"experience_fixture":true,"q":"yes","v":"yes"}'),
(2,'tie-a','2026-01-10','{"experience_fixture":true,"q":"yes","v":"yes"}'),
(2,'tie-b','2026-01-10','{"experience_fixture":true,"q":"yes","v":"no"}'),
(3,'possible','2026-01-15','{"experience_fixture":true,"v":"yes"}')
) v(n,id,stamp,payload)`)
		if err != nil {
			t.Fatalf("expected coherent experience fixtures, got %v", err)
		}
		defer func() {
			_, err := pool.Exec(t.Context(), "DELETE FROM krenalis_events WHERE message_id LIKE 'experience/%'")
			if err != nil {
				t.Errorf("expected experience fixture cleanup, got %v", err)
			}
		}()

		const experience = `{"operator":"and","rules":[{"experience":{"base":{` +
			`"source":{"kind":"recordedEvents","name":"Order Completed","grain":"registered-message"},` +
			`"period":{"kind":"elapsedLookback","duration":{"value":31,"unit":"day"}},` +
			`"where":{"operator":"and","rules":[{"property":"properties.experience_fixture","operator":"exists"},` +
			`{"property":"properties.q","operator":"is","values":["yes"]}]}},"select":{"kind":"last"},` +
			`"verify":{"operator":"and","rules":[{"property":"properties.v","operator":"is","values":["yes"]}]}}}]}`
		temporal := strings.Replace(experience,
			`{"property":"properties.v","operator":"is","values":["yes"]}`,
			`{"occurredIn":{"kind":"elapsedLookback","duration":{"value":31,"unit":"day"}}}`, 1)
		firstInPeriod := strings.Replace(temporal, `"kind":"last"`, `"kind":"first"`, 1)
		firstInHistory := strings.Replace(firstInPeriod,
			`"period":{"kind":"elapsedLookback","duration":{"value":31,"unit":"day"}}`,
			`"period":{"kind":"fullRelevantHistory",`+
				`"history":{"definition":"registered-queryable-events","meaningRevision":"1"}}`, 1)
		for _, test := range []struct{ name, filter, totals string }{
			{"first recorded recent", firstInHistory, `"match":1,"noMatch":2,"unknown":1`},
			{"first within recent base", firstInPeriod, `"match":2,"noMatch":1,"unknown":1`},
			{"last", experience, `"match":1,"noMatch":1,"unknown":2`},
			{"first", strings.Replace(experience, `"kind":"last"`, `"kind":"first"`, 1), `"match":0,"noMatch":2,"unknown":2`},
			{"outer OR", strings.Replace(strings.Replace(experience, `"operator":"and"`, `"operator":"or"`, 1), `"rules":[{"experience"`, `"rules":[{"property":"country","operator":"is","values":["FR"]},{"experience"`, 1), `"match":3,"noMatch":0,"unknown":1`},
			{"outer AND", strings.Replace(experience, `"rules":[{"experience"`, `"rules":[{"property":"country","operator":"is","values":["IT"]},{"experience"`, 1), `"match":1,"noMatch":2,"unknown":1`},
		} {
			t.Run(test.name, func(t *testing.T) {
				for _, first := range []int{0, 99} {
					var expected json.RawMessage
					k.Call("GET", "/v1/profiles?filter="+url.QueryEscape(test.filter)+"&at="+
						url.QueryEscape(at)+"&first="+strconv.Itoa(first), nil, nil, &expected)
					actual := assertResult(t, tool(t, "query-profiles", map[string]any{
						"filter": test.filter, "at": at, "first": first}))
					if actual != string(expected) || !strings.Contains(actual, test.totals) ||
						first == 99 && !strings.HasPrefix(actual, `{"profiles":[]`) {
						t.Fatalf("expected identical complete HTTP/MCP experience with %s, got %s", test.totals, actual)
					}
				}
			})
		}

		for _, test := range []struct{ id, kind, absence, result string }{
			{"00000000-0000-0000-0000-000000000001", "unique", "impossible", "MATCH"},
			{"00000000-0000-0000-0000-000000000002", "ambiguous", "impossible", "UNKNOWN"},
			{"00000000-0000-0000-0000-000000000003", "ambiguous", "possible", "UNKNOWN"},
			{"00000000-0000-0000-0000-000000000004", "absent", "certain", "NO_MATCH"},
		} {
			t.Run("explain "+test.id, func(t *testing.T) {
				var expected json.RawMessage
				k.Call("GET", "/v1/profiles/"+test.id+"/filter-explanation?filter="+url.QueryEscape(experience)+
					"&at="+url.QueryEscape(at), nil, nil, &expected)
				actual := assertResult(t, tool(t, "explain-profile-filter", map[string]any{
					"kpid": test.id, "filter": experience, "at": at}))
				if actual != string(expected) || !strings.Contains(actual, `"kind":"experience"`) ||
					!strings.Contains(actual, `"selection":{"kind":"`+test.kind+`","absence":"`+test.absence+`"`) ||
					!strings.Contains(actual, `"result":"`+test.result+`"`) {
					t.Fatalf("expected %s selection, %s absence and %s, got %s", test.kind, test.absence, test.result, actual)
				}
			})
		}

		for _, test := range []struct{ filter, result string }{
			{firstInHistory, "NO_MATCH"}, {firstInPeriod, "MATCH"},
		} {
			t.Run("time explanation "+test.result, func(t *testing.T) {
				const id = "00000000-0000-0000-0000-000000000001"
				var expected json.RawMessage
				k.Call("GET", "/v1/profiles/"+id+"/filter-explanation?filter="+url.QueryEscape(test.filter)+
					"&at="+url.QueryEscape(at), nil, nil, &expected)
				actual := assertResult(t, tool(t, "explain-profile-filter", map[string]any{
					"kpid": id, "filter": test.filter, "at": at}))
				if actual != string(expected) || !strings.Contains(actual, `"result":"`+test.result+`"`) ||
					!strings.Contains(actual, `"occurredIn":[{"path":[0],"start":"2026-01-01T00:00:00Z","end":"`+at+`"}]`) {
					t.Fatalf("expected identical complete temporal explanations with %s, got %s", test.result, actual)
				}
			})
		}

		for _, test := range []struct{ name, filter, code string }{
			{"time history unavailable", strings.Replace(firstInHistory,
				`{"occurredIn":{"kind":"elapsedLookback","duration":{"value":31,"unit":"day"}}}`,
				`{"occurredIn":{"kind":"fullRelevantHistory","history":{"definition":"registered-queryable-events","meaningRevision":"1"}}}`, 1), "FilterCapabilityNotImplemented"},
			{"nth unavailable", strings.Replace(experience, `"kind":"last"`, `"kind":"nth","position":2`, 1), "FilterCapabilityNotImplemented"},
			{"verify required", strings.Replace(experience, `,"verify":{"operator":"and","rules":[{"property":"properties.v","operator":"is","values":["yes"]}]}`, ``, 1), "InvalidArguments"},
			{"invalid verify", strings.Replace(experience, `properties.v`, `unknown`, 1), "PropertyNotExist"},
		} {
			t.Run(test.name, func(t *testing.T) {
				assertToolError(t, tool(t, "query-profiles", map[string]any{"filter": test.filter, "at": at}), test.code)
				assertToolError(t, tool(t, "explain-profile-filter", map[string]any{
					"kpid": "00000000-0000-0000-0000-000000000001", "filter": test.filter, "at": at}), test.code)
			})
		}

	})

	t.Run("ExperienceMoney", func(t *testing.T) {

		_, err := pool.Exec(t.Context(), `INSERT INTO krenalis_events
(kpid,message_id,event,timestamp,received_at,sent_at,type,connection_id,anonymous_id,traits,properties)
SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid,'money/'||n||'/'||id,
'Order Completed',stamp::timestamp,'2026-02-01','2026-02-01','track','fixture','a','{}',payload::jsonb
FROM (VALUES
(1,'old','2026-01-01','{"money_fixture":true,"total":150,"currency":"EUR"}'),
(1,'new','2026-01-15','{"money_fixture":true,"total":120,"currency":"EUR"}'),
(2,'old','2026-01-01','{"money_fixture":true,"total":150,"currency":"EUR"}'),
(2,'new','2026-01-20','{"money_fixture":true,"total":120,"currency":"USD"}'),
(3,'unknown','2026-01-15','{"money_fixture":true,"total":120}')
) v(n,id,stamp,payload)`)
		if err != nil {
			t.Fatalf("expected coherent monetary experience fixtures, got %v", err)
		}
		defer func() {
			_, err := pool.Exec(t.Context(), "DELETE FROM krenalis_events WHERE message_id LIKE 'money/%'")
			if err != nil {
				t.Errorf("expected monetary experience cleanup, got %v", err)
			}
		}()

		const money = `{"money":{"field":{"definition":"commerce.order_completed.total",` +
			`"meaningRevision":"1"},"nativeCurrency":"EUR"},"operator":"is greater than","values":[100]}`
		const filter = `{"operator":"and","rules":[{"experience":{"base":{` +
			`"source":{"kind":"recordedEvents","name":"Order Completed","grain":"registered-message"},` +
			`"period":{"kind":"fullRelevantHistory","history":{"definition":"registered-queryable-events",` +
			`"meaningRevision":"1"}},"where":{"operator":"and","rules":[` +
			`{"property":"properties.money_fixture","operator":"exists"}]}},"select":{"kind":"last"},` +
			`"verify":{"operator":"and","rules":[{"occurredIn":{"kind":"elapsedLookback",` +
			`"duration":{"value":30,"unit":"day"}}},` + money + `]}}}]}`
		for _, first := range []int{0, 99} {
			var expected json.RawMessage
			k.Call("GET", "/v1/profiles?filter="+url.QueryEscape(filter)+"&at="+url.QueryEscape(at)+
				"&first="+strconv.Itoa(first), nil, nil, &expected)
			actual := assertResult(t, tool(t, "query-profiles", map[string]any{
				"filter": filter, "at": at, "first": first}))
			if actual != string(expected) || !strings.Contains(actual, `"match":1,"noMatch":2,"unknown":1`) ||
				first == 99 && !strings.HasPrefix(actual, `{"profiles":[]`) {
				t.Fatalf("expected identical complete I028 page/totals, got %s", actual)
			}
		}
		for _, test := range []struct{ id, result string }{
			{"00000000-0000-0000-0000-000000000001", "MATCH"},
			{"00000000-0000-0000-0000-000000000002", "NO_MATCH"},
			{"00000000-0000-0000-0000-000000000003", "UNKNOWN"},
		} {
			var expected json.RawMessage
			k.Call("GET", "/v1/profiles/"+test.id+"/filter-explanation?filter="+url.QueryEscape(filter)+
				"&at="+url.QueryEscape(at), nil, nil, &expected)
			actual := assertResult(t, tool(t, "explain-profile-filter", map[string]any{
				"kpid": test.id, "filter": filter, "at": at}))
			if actual != string(expected) || !strings.Contains(actual, `"result":"`+test.result+`"`) ||
				!strings.Contains(actual, `"money":[{"path":[1],"field":{"definition":"commerce.order_completed.total",`+
					`"meaningRevision":"1"},"nativeCurrency":"EUR","requested":{"operator":"is greater than",`) ||
				!strings.Contains(actual, `"normalized":{"operator":"is greater than or equal to","values":[100.0001]}`) {
				t.Fatalf("expected identical I028 explanation with requested and normalized tests, got %s", actual)
			}
		}
		for _, test := range []struct{ name, input, code string }{
			{"multiple money", strings.Replace(filter, money, money+`,`+money, 1), "FilterCapabilityNotImplemented"},
			{"overlap verify", strings.Replace(filter, money, money+`,{"property":"properties.currency","operator":"exists"}`, 1), "FilterCapabilityNotImplemented"},
			{"overlap base", strings.Replace(filter, `{"property":"properties.money_fixture","operator":"exists"}`, `{"property":"properties.currency","operator":"is","values":["EUR"]}`, 1), "FilterCapabilityNotImplemented"},
			{"money in base", strings.Replace(filter, `{"property":"properties.money_fixture","operator":"exists"}`, money, 1), "FilterCapabilityNotImplemented"},
			{"equality", strings.Replace(filter, `"operator":"is greater than"`, `"operator":"is"`, 1), "FilterCapabilityNotImplemented"},
			{"wrong source", strings.Replace(filter, `"name":"Order Completed"`, `"name":"Other"`, 1), "InvalidArguments"},
		} {
			t.Run(test.name, func(t *testing.T) {
				assertToolError(t, tool(t, "query-profiles", map[string]any{"filter": test.input, "at": at}), test.code)
				assertToolError(t, tool(t, "explain-profile-filter", map[string]any{
					"kpid": "00000000-0000-0000-0000-000000000001", "filter": test.input, "at": at}), test.code)
			})
		}

	})

	t.Run("Average", func(t *testing.T) {

		_, err := pool.Exec(t.Context(), `INSERT INTO krenalis_events
 (kpid,message_id,event,timestamp,received_at,sent_at,type,connection_id,anonymous_id,traits,properties)
 SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid,'average/'||id,'Order Completed',
 '2026-01-15','2026-02-01','2026-02-01','track','fixture','a','{}',payload::jsonb
 FROM (VALUES
 (1,'a','{"total":120,"currency":"EUR","coupon":"PROMO","average_fixture":true}'),
 (1,'b','{"total":100,"currency":"EUR","coupon":"PROMO","average_fixture":true}'),
 (1,'excluded','{"coupon":"OTHER","average_fixture":true}'),
 (1,'foreign','{"currency":"USD","coupon":"PROMO","average_fixture":true}'),
 (2,'c','{"total":100,"currency":"EUR","coupon":"PROMO","average_fixture":true}'),
 (2,'d','{"total":99.9999,"currency":"EUR","coupon":"PROMO","average_fixture":true}'),
 (3,'e','{"total":120,"currency":"EUR","coupon":"PROMO","average_fixture":true}'),
 (3,'f','{"currency":"EUR","coupon":"PROMO","average_fixture":true}'),
 (4,'g','{"total":100,"coupon":"PROMO","average_fixture":true}')
 ) v(n,id,payload)`)
		if err != nil {
			t.Fatalf("expected coherent Average fixtures, got %v", err)
		}
		defer func() {
			_, err := pool.Exec(t.Context(), "DELETE FROM krenalis_events WHERE message_id LIKE 'average/%'")
			if err != nil {
				t.Errorf("expected Average fixture cleanup, got %v", err)
			}
		}()

		const average = `{"operator":"and","rules":[{"average":{"source":{"kind":"recordedEvents",` +
			`"name":"Order Completed","grain":"registered-message"},"period":{"kind":"elapsedLookback",` +
			`"duration":{"value":31,"unit":"day"}},"field":{"definition":"commerce.order_completed.total",` +
			`"meaningRevision":"1"},"nativeCurrency":"EUR","minimum":2,` +
			`"where":{"operator":"and","rules":[{"property":"properties.average_fixture","operator":"exists"},` +
			`{"property":"properties.coupon","operator":"is","values":["PROMO"]}]}},` +
			`"operator":"is greater than or equal to","values":[100]}]}`
		for _, test := range []struct{ name, filter, totals string }{
			{"exact average", average, `"match":1,"noMatch":2,"unknown":1`},
			{"fractional average", strings.Replace(average, `"values":[100]`, `"values":[99.9999]`, 1), `"match":2,"noMatch":1,"unknown":1`},
			{"strict boundary", strings.Replace(strings.Replace(average, `is greater than or equal to`, `is greater than`, 1), `"values":[100]`, `"values":[110]`, 1), `"match":0,"noMatch":3,"unknown":1`},
			{"minimum false", strings.Replace(average, `"minimum":2`, `"minimum":3`, 1), `"match":0,"noMatch":4,"unknown":0`},
			{"large threshold", strings.Replace(average, `"values":[100]`, `"values":[1e75]`, 1), `"match":0,"noMatch":3,"unknown":1`},
			{"outer OR", strings.Replace(strings.Replace(average, `"operator":"and"`, `"operator":"or"`, 1), `"rules":[{"average"`, `"rules":[{"property":"country","operator":"is","values":["FR"]},{"average"`, 1), `"match":3,"noMatch":0,"unknown":1`},
		} {
			t.Run(test.name, func(t *testing.T) {
				for _, first := range []int{0, 99} {
					var expected json.RawMessage
					k.Call("GET", "/v1/profiles?filter="+url.QueryEscape(test.filter)+"&at="+
						url.QueryEscape(at)+"&first="+strconv.Itoa(first), nil, nil, &expected)
					actual := assertResult(t, tool(t, "query-profiles", map[string]any{
						"filter": test.filter, "at": at, "first": first}))
					if actual != string(expected) || !strings.Contains(actual, test.totals) ||
						first == 99 && !strings.HasPrefix(actual, `{"profiles":[]`) {
						t.Fatalf("expected identical complete HTTP/MCP Average with %s, got %s", test.totals, actual)
					}
				}
			})
		}
		for _, test := range []struct{ id, sum, count, counts, result string }{
			{"00000000-0000-0000-0000-000000000001", "220", "2", `"certain":2,"uncertain":0`, "MATCH"},
			{"00000000-0000-0000-0000-000000000002", "199.9999", "2", `"certain":2,"uncertain":0`, "NO_MATCH"},
			{"00000000-0000-0000-0000-000000000003", "null", "2", `"certain":2,"uncertain":0`, "UNKNOWN"},
			{"00000000-0000-0000-0000-000000000004", "null", "null", `"certain":0,"uncertain":1`, "NO_MATCH"},
		} {
			t.Run("explain "+test.id, func(t *testing.T) {
				var expected json.RawMessage
				k.Call("GET", "/v1/profiles/"+test.id+"/filter-explanation?filter="+url.QueryEscape(average)+
					"&at="+url.QueryEscape(at), nil, nil, &expected)
				actual := assertResult(t, tool(t, "explain-profile-filter", map[string]any{
					"kpid": test.id, "filter": average, "at": at}))
				var decoded struct {
					Rule struct {
						Rules []struct {
							Average struct {
								Sum, Count json.RawMessage
							}
						}
					}
				}
				err := json.Unmarshal([]byte(actual), &decoded)
				if err != nil {
					t.Fatalf("expected exact JSON inputs decoded, got %v", err)
				}
				if len(decoded.Rule.Rules) != 1 {
					t.Fatalf("expected one Average explanation, got %s", actual)
				}
				values := decoded.Rule.Rules[0].Average
				matches := string(values.Sum) == test.sum
				if test.sum != "null" {
					value, ok := new(big.Rat).SetString(string(values.Sum))
					want, _ := new(big.Rat).SetString(test.sum)
					matches = ok && value.Cmp(want) == 0
				}
				if actual != string(expected) || !strings.Contains(actual, `"kind":"average"`) ||
					!matches || string(values.Count) != test.count || !strings.Contains(actual, test.counts) ||
					!strings.Contains(actual, `"result":"`+test.result+`"`) {
					t.Fatalf("expected exact Average inputs sum=%s count=%s, got %s", test.sum, test.count, actual)
				}
			})
		}
		for _, test := range []struct {
			name, filter, code, message string
			status                      int
		}{
			{"omitted minimum", strings.Replace(average, `,"minimum":2`, ``, 1), "InvalidArguments", "Average requires a positive sample minimum", 400},
			{"zero minimum", strings.Replace(average, `"minimum":2`, `"minimum":0`, 1), "InvalidArguments", "sample minimum must be positive", 400},
			{"negative minimum", strings.Replace(average, `"minimum":2`, `"minimum":-1`, 1), "InvalidArguments", "sample minimum must be a positive JSON integer", 400},
			{"oversize minimum", strings.Replace(average, `"minimum":2`, `"minimum":9223372036854775808`, 1), "FilterCapabilityNotImplemented", "monetary sample minimum exceeds the execution integer range", 422},
			{"null minimum", strings.Replace(average, `"minimum":2`, `"minimum":null`, 1), "InvalidArguments", "monetary fields cannot be null", 400},
			{"finer threshold", strings.Replace(average, `"values":[100]`, `"values":[0.00001]`, 1), "FilterCapabilityNotImplemented", "Average threshold finer", 422},
			{"excessive precision", strings.Replace(average, `"values":[100]`, `"values":[1e76]`, 1), "InvalidArguments", "", 400},
			{"non-track source", strings.Replace(average, `"kind":"recordedEvents"`, `"kind":"recordedEvents","type":"page"`, 1), "InvalidArguments", "monetary field is incompatible", 400},
			{"dependent where", strings.Replace(average, `properties.coupon`, `properties.total`, 1), "FilterCapabilityNotImplemented", "monetary qualification dependent", 422},
			{"equality", strings.Replace(average, `is greater than or equal to`, `is`, 1), "FilterCapabilityNotImplemented", "monetary equality or interval comparisons", 422},
		} {
			t.Run(test.name, func(t *testing.T) {
				query := tool(t, "query-profiles", map[string]any{"filter": test.filter, "at": at})
				assertToolError(t, query, test.code)
				explanation := tool(t, "explain-profile-filter", map[string]any{
					"kpid": "00000000-0000-0000-0000-000000000001", "filter": test.filter, "at": at})
				assertToolError(t, explanation, test.code)
				for _, result := range []profileMCPResponse{query, explanation} {
					message := result.Result.Content[0].Text
					if !strings.Contains(message, test.message) || strings.Contains(message, "Sum") {
						t.Fatalf("expected neutral diagnostic %q with code %s, got %s", test.message, test.code, message)
					}
				}
				var body json.RawMessage
				err := k.TryCall("GET", "/v1/profiles?filter="+url.QueryEscape(test.filter)+"&at="+
					url.QueryEscape(at), nil, nil, &body)
				if err != nil {
					status, ok := errors.AsType[*krenalistester.StatusCodeError](err)
					message := test.message
					if test.name == "null minimum" {
						message = "invalid filter"
					}
					if !ok || status.Response.Code != test.status ||
						!strings.Contains(status.Response.Text, message) || strings.Contains(status.Response.Text, "Sum") {
						t.Fatalf("expected HTTP %d, got %v", test.status, err)
					}
					return
				}
				t.Fatalf("expected HTTP %d without a result, got %s", test.status, body)
			})
		}

		t.Run("nested Average error location", func(t *testing.T) {
			rule := strings.TrimSuffix(strings.TrimPrefix(average, `{"operator":"and","rules":[`), `]}`)
			invalid := strings.Replace(rule, `"minimum":2`, `"minimum":null`, 1)
			filter := `{"operator":"and","rules":[` + rule + "," + rule +
				`,{"operator":"or","rules":[` + rule + "," + invalid + `]}]}`
			query := tool(t, "query-profiles", map[string]any{"filter": filter, "at": at})
			assertToolError(t, query, "InvalidArguments")
			explanation := tool(t, "explain-profile-filter", map[string]any{
				"kpid": "00000000-0000-0000-0000-000000000001", "filter": filter, "at": at})
			assertToolError(t, explanation, "InvalidArguments")
			for _, result := range []profileMCPResponse{query, explanation} {
				message := result.Result.Content[0].Text
				if !strings.Contains(message, `within "/rules/2/rules/1/average": monetary fields cannot be null`) ||
					strings.Contains(message, "Sum") {
					t.Fatalf("expected nested Average path and neutral InvalidArguments diagnostic, got %s", message)
				}
			}
		})

	})

	t.Run("Sum", func(t *testing.T) {

		_, err := pool.Exec(t.Context(), `UPDATE krenalis_events SET properties='{"total":20,"currency":"EUR"}'
WHERE event='Order Completed';
INSERT INTO krenalis_events
(kpid,message_id,event,timestamp,received_at,sent_at,type,connection_id,anonymous_id,traits,properties) VALUES
('00000000-0000-0000-0000-000000000002','sum-two','Order Completed','2026-01-15',
'2026-02-01','2026-02-01','track','fixture','a','{}','{"total":20,"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000002','sum-usd','Order Completed','2026-01-15',
'2026-02-01','2026-02-01','track','fixture','a','{}','{"currency":"USD"}'),
('00000000-0000-0000-0000-000000000003','sum-three','Order Completed','2026-01-15',
'2026-02-01','2026-02-01','track','fixture','a','{}','{"total":20,"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000003','sum-unknown','Order Completed','2026-01-15',
'2026-02-01','2026-02-01','track','fixture','a','{}','{"total":20}')`)
		if err != nil {
			t.Fatalf("expected coherent monetary fixtures, got %v", err)
		}

		const sum = `{"operator":"and","rules":[{"sum":{"source":{"kind":"recordedEvents",` +
			`"name":"Order Completed","grain":"registered-message"},"period":{"kind":"elapsedLookback",` +
			`"duration":{"value":31,"unit":"day"}},"field":{"definition":"commerce.order_completed.total",` +
			`"meaningRevision":"1"},"nativeCurrency":"EUR"},"operator":"is greater than or equal to","values":[30]}]}`
		for _, first := range []int{0, 99} {
			var expected json.RawMessage
			path := "/v1/profiles?filter=" + url.QueryEscape(sum) +
				"&at=" + url.QueryEscape(at) + "&first=" + strconv.Itoa(first)
			k.Call("GET", path, nil, nil, &expected)
			actual := assertResult(t, tool(t, "query-profiles", map[string]any{"filter": sum, "at": at, "first": first}))
			if actual != string(expected) || !strings.Contains(actual, `"match":1,"noMatch":2,"unknown":1`) {
				t.Fatalf("expected identical HTTP/MCP Sum listing and totals 1/2/1, got %s", actual)
			}
			if first == 99 && !strings.HasPrefix(actual, `{"profiles":[]`) {
				t.Fatalf("expected empty Sum page, got %s", actual)
			}
		}
		var expected json.RawMessage
		profile := "00000000-0000-0000-0000-000000000003"
		path := "/v1/profiles/" + profile + "/filter-explanation?filter=" +
			url.QueryEscape(sum) + "&at=" + url.QueryEscape(at)
		k.Call("GET", path, nil, nil, &expected)
		actual := assertResult(t, tool(t, "explain-profile-filter", map[string]any{"kpid": profile, "filter": sum, "at": at}))
		if actual != string(expected) || !strings.Contains(actual, `"result":"UNKNOWN"`) ||
			!strings.Contains(actual, `"definition":"commerce.order_completed.total"`) ||
			!strings.Contains(actual, `"nativeCurrency":"EUR"`) {
			t.Fatalf("expected measured Sum explanation with reference and currency, got %s", actual)
		}
		var explanation struct {
			Rule struct {
				Rules []struct {
					Sum struct {
						Bounds    struct{ Lower, Upper json.Number } `json:"bounds"`
						Certain   int                                `json:"certain"`
						Uncertain int                                `json:"uncertain"`
					} `json:"sum"`
				} `json:"rules"`
			} `json:"rule"`
		}
		err = json.Unmarshal([]byte(actual), &explanation)
		if err != nil {
			t.Fatalf("expected exact bounds JSON, got %v", err)
		}
		if len(explanation.Rule.Rules) != 1 {
			t.Fatalf("expected one Sum leaf, got %s", actual)
		}
		lower, lok := new(big.Rat).SetString(string(explanation.Rule.Rules[0].Sum.Bounds.Lower))
		upper, uok := new(big.Rat).SetString(string(explanation.Rule.Rules[0].Sum.Bounds.Upper))
		if !lok || !uok || lower.Cmp(big.NewRat(20, 1)) != 0 || upper.Cmp(big.NewRat(40, 1)) != 0 {
			t.Fatalf("expected exact bounds [20,40], got %s", actual)
		}
		if explanation.Rule.Rules[0].Sum.Certain != 1 || explanation.Rule.Rules[0].Sum.Uncertain != 1 {
			t.Fatalf("expected Sum count 1 certain and 1 uncertain without minimum, got %s", actual)
		}
		for _, test := range []struct{ minimum, threshold, totals string }{
			{"2", "30", `"match":1,"noMatch":2,"unknown":1`},
			{"2", "0", `"match":1,"noMatch":2,"unknown":1`},
			{"3", "30", `"match":0,"noMatch":4,"unknown":0`},
		} {
			t.Run("minimum "+test.minimum+" threshold "+test.threshold, func(t *testing.T) {
				filter := strings.Replace(sum, `"nativeCurrency":"EUR"`,
					`"nativeCurrency":"EUR","minimum":`+test.minimum, 1)
				filter = strings.Replace(filter, `[30]`, `[`+test.threshold+`]`, 1)
				for _, first := range []int{0, 99} {
					var expected json.RawMessage
					path := "/v1/profiles?filter=" + url.QueryEscape(filter) +
						"&at=" + url.QueryEscape(at) + "&first=" + strconv.Itoa(first)
					k.Call("GET", path, nil, nil, &expected)
					actual := assertResult(t, tool(t, "query-profiles",
						map[string]any{"filter": filter, "at": at, "first": first}))
					if actual != string(expected) || !strings.Contains(actual, test.totals) {
						t.Fatalf("expected identical guarded HTTP/MCP totals %s, got %s", test.totals, actual)
					}
				}
			})
		}
		for _, test := range []struct{ id, counts string }{
			{"00000000-0000-0000-0000-000000000003", `"certain":1,"uncertain":1`},
			{"00000000-0000-0000-0000-000000000004", `"certain":0,"uncertain":0`},
		} {
			for _, minimum := range []string{"", `,"minimum":3`} {
				filter := strings.Replace(sum, `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR"`+minimum, 1)
				var expected json.RawMessage
				path := "/v1/profiles/" + test.id + "/filter-explanation?filter=" +
					url.QueryEscape(filter) + "&at=" + url.QueryEscape(at)
				k.Call("GET", path, nil, nil, &expected)
				actual := assertResult(t, tool(t, "explain-profile-filter",
					map[string]any{"kpid": test.id, "filter": filter, "at": at}))
				if actual != string(expected) || !strings.Contains(actual, test.counts) ||
					minimum != "" && !strings.Contains(actual, `"result":"NO_MATCH"`) {
					t.Fatalf("expected explicit qualified counts %s with minimum %s, got %s", test.counts, minimum, actual)
				}
			}
		}
		for _, test := range []struct{ name, filter, code string }{
			{"equality", strings.Replace(sum, `is greater than or equal to`, `is`, 1), "FilterCapabilityNotImplemented"},
			{"page with same name", strings.Replace(sum, `"kind":"recordedEvents"`, `"kind":"recordedEvents","type":"page"`, 1), "InvalidArguments"},
			{"screen with same name", strings.Replace(sum, `"kind":"recordedEvents"`, `"kind":"recordedEvents","type":"screen"`, 1), "InvalidArguments"},
			{"missing currency", strings.Replace(sum, `,"nativeCurrency":"EUR"`, ``, 1), "InvalidArguments"},
			{"incompatible source", strings.Replace(sum, `"name":"Order Completed"`, `"name":"Refund"`, 1), "InvalidArguments"},
			{"zero minimum", strings.Replace(sum, `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","minimum":0`, 1), "InvalidArguments"},
			{"large minimum", strings.Replace(sum, `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","minimum":9223372036854775808`, 1), "FilterCapabilityNotImplemented"},
		} {
			t.Run(test.name, func(t *testing.T) {
				assertToolError(t, tool(t, "query-profiles", map[string]any{"filter": test.filter, "at": at}), test.code)
			})
		}
		negative := strings.Replace(sum, `[30]`, `[-0.00001]`, 1)
		actual = assertResult(t, tool(t, "query-profiles", map[string]any{"filter": negative, "at": at}))
		if !strings.Contains(actual, `"match":4,"noMatch":0,"unknown":0`) {
			t.Fatalf("expected negative threshold with empty sum zero, got %s", actual)
		}

		_, err = pool.Exec(t.Context(), `UPDATE krenalis_events
SET properties='{"total":150,"currency":"EUR"}' WHERE message_id='sum-two'`)
		if err != nil {
			t.Fatalf("expected large threshold fixture, got %v", err)
		}
		for _, literal := range []string{"1000", "1000.0000", "1e75"} {
			t.Run(literal, func(t *testing.T) {
				filter := strings.Replace(sum, `[30]`, `[`+literal+`]`, 1)
				var expected json.RawMessage
				path := "/v1/profiles?filter=" + url.QueryEscape(filter) + "&at=" + url.QueryEscape(at)
				k.Call("GET", path, nil, nil, &expected)
				actual := assertResult(t, tool(t, "query-profiles", map[string]any{"filter": filter, "at": at}))
				if actual != string(expected) || !strings.Contains(actual, `"match":0,"noMatch":4,"unknown":0`) {
					t.Fatalf("expected exact large threshold with no members in HTTP and MCP, got %s", actual)
				}
			})
		}

		_, err = pool.Exec(t.Context(), `UPDATE krenalis_events
SET properties=jsonb_set(properties,'{coupon}','"PROMO"')
WHERE event='Order Completed' AND kpid='00000000-0000-0000-0000-000000000001';
UPDATE krenalis_events SET properties=jsonb_set(properties,'{coupon}','"OTHER"') WHERE message_id='sum-two';
UPDATE krenalis_events SET properties=jsonb_set(properties,'{coupon}','"PROMO"') WHERE message_id='sum-three'`)
		if err != nil {
			t.Fatalf("expected independent coupon fixtures, got %v", err)
		}
		qualified := strings.Replace(sum, `"nativeCurrency":"EUR"`,
			`"nativeCurrency":"EUR","where":{"operator":"and","rules":[`+
				`{"property":"properties.coupon","operator":"is","values":["PROMO"]}]}`, 1)
		for _, first := range []int{0, 99} {
			var expected json.RawMessage
			path := "/v1/profiles?filter=" + url.QueryEscape(qualified) +
				"&at=" + url.QueryEscape(at) + "&first=" + strconv.Itoa(first)
			k.Call("GET", path, nil, nil, &expected)
			actual := assertResult(t, tool(t, "query-profiles",
				map[string]any{"filter": qualified, "at": at, "first": first}))
			if actual != string(expected) || !strings.Contains(actual, `"match":1,"noMatch":2,"unknown":1`) ||
				first == 0 && !strings.Contains(actual, `"profile-1@example.com"`) ||
				first == 99 && !strings.HasPrefix(actual, `{"profiles":[]`) {
				t.Fatalf("expected only the PROMO member with complete totals, got %s", actual)
			}
		}
		profile = "00000000-0000-0000-0000-000000000002"
		path = "/v1/profiles/" + profile + "/filter-explanation?filter=" +
			url.QueryEscape(qualified) + "&at=" + url.QueryEscape(at)
		k.Call("GET", path, nil, nil, &expected)
		actual = assertResult(t, tool(t, "explain-profile-filter",
			map[string]any{"kpid": profile, "filter": qualified, "at": at}))
		if actual != string(expected) || !strings.Contains(actual, `"result":"NO_MATCH"`) {
			t.Fatalf("expected matching HTTP/MCP explanation of the coupon exclusion, got %s", actual)
		}
		err = json.Unmarshal([]byte(actual), &explanation)
		if err != nil {
			t.Fatalf("expected qualified bounds JSON, got %v", err)
		}
		if len(explanation.Rule.Rules) != 1 {
			t.Fatalf("expected one qualified Sum, got %s", actual)
		}
		lower, lok = new(big.Rat).SetString(string(explanation.Rule.Rules[0].Sum.Bounds.Lower))
		upper, uok = new(big.Rat).SetString(string(explanation.Rule.Rules[0].Sum.Bounds.Upper))
		if !lok || !uok || lower.Sign() != 0 || upper.Sign() != 0 {
			t.Fatalf("expected excluded coupon orders to contribute [0,0], got %s", actual)
		}
		for _, property := range []string{"properties", "properties.total.x", "properties.currency"} {
			filter := strings.Replace(qualified, `properties.coupon`, property, 1)
			assertToolError(t, tool(t, "query-profiles", map[string]any{"filter": filter, "at": at}),
				"FilterCapabilityNotImplemented")
			assertToolError(t, tool(t, "explain-profile-filter",
				map[string]any{"kpid": profile, "filter": filter, "at": at}), "FilterCapabilityNotImplemented")
		}

	})

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
		{"unresolved history", map[string]any{"filter": strings.Replace(historyCount, `"meaningRevision":"1"`, `"meaningRevision":"2"`, 1)}, "InvalidArguments"},
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
	assertToolError(t, tool(t, "explain-profile-filter", map[string]any{"filter": count}), "MCPNotConfigured")
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
	failed = tool(t, "explain-profile-filter", map[string]any{
		"kpid": "00000000-0000-0000-0000-000000000001", "filter": count, "at": at})
	if failed.Error == nil || len(failed.Result.Content) != 0 ||
		failed.Error.Message != "cmd/internal/mcp: profile query failed" {
		t.Fatalf("expected opaque explanation failure without data, got %+v", failed)
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
