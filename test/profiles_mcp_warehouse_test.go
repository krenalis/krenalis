// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"os"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

// TestProfilesMCPPostgreSQL checks the bounded cross-warehouse MCP fixture.
func TestProfilesMCPPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("requires PostgreSQL, NATS and the server")
	}
	testProfilesMCPWarehouse(t, "PostgreSQL")
}

// TestProfilesMCPSnowflake requires the isolated internal-server overlay.
func TestProfilesMCPSnowflake(t *testing.T) {
	if testing.Short() || os.Getenv("KRENALIS_SKIP_SNOWFLAKE_TESTS") == "true" {
		t.Skip("remote Snowflake tests are disabled")
	}
	if !krenalistester.UsesInternalServer() {
		t.Skip("Snowflake MCP verification requires the internal-server overlay")
	}
	dw := warehouses.Registered("Snowflake").New(profileHTTPSettings{json.Value(`{}`)}, nil)
	verified := dw.(warehouses.ProfileFilterWarehouse).ProfileFilterVerified()
	err := dw.Close()
	if err != nil {
		t.Fatalf("expected unopened warehouse cleanup, got %v", err)
	}
	if !verified {
		t.Skip("Snowflake MCP verification requires the isolated gate overlay")
	}
	testProfilesMCPWarehouse(t, "Snowflake")
}

func testProfilesMCPWarehouse(t *testing.T, platform string) {

	t.Helper()
	k, _ := newProfileTransportFixture(t, platform)
	var key struct {
		Token string `json:"token"`
	}
	k.Call("POST", "/v1/keys", nil, map[string]any{"name": "MCP transport fixture",
		"workspace": k.WorkspaceID(), "type": "MCP"}, &key)
	session := ""
	response := profileMCPCall(t, k.Addr(), key.Token, &session, "initialize", map[string]any{
		"protocolVersion": "2024-11-05", "capabilities": map[string]any{},
		"clientInfo": map[string]any{"name": "MCP transport fixture", "version": "1"},
	})
	if response.Error != nil {
		t.Fatalf("expected initialized MCP session, got %+v", response.Error)
	}
	profileMCPCall(t, k.Addr(), key.Token, &session, "notifications/initialized", map[string]any{})
	tool := func(t *testing.T, name string, args any) string {
		t.Helper()
		response := profileMCPCall(t, k.Addr(), key.Token, &session, "tools/call",
			map[string]any{"name": name, "arguments": args})
		if response.Error != nil || response.Result.IsError || len(response.Result.Content) != 1 ||
			response.Result.Content[0].Type != "text" {
			t.Fatalf("expected complete successful MCP result, got %+v", response)
		}
		return response.Result.Content[0].Text
	}

	t.Run("catalog", func(t *testing.T) {
		catalog := tool(t, "list-event-definitions", map[string]any{"query": "Order Completed"})
		if !strings.Contains(catalog, `"id":"commerce.order_completed"`) ||
			!strings.Contains(catalog, `"event":"Order Completed"`) {
			t.Fatalf("expected catalog ID and technical event name, got %s", catalog)
		}
		definition := tool(t, "get-event-definition", map[string]any{"id": "commerce.order_completed"})
		if !strings.Contains(definition, `"semantics":"money"`) {
			t.Fatalf("expected governed monetary field documentation, got %s", definition)
		}
		guide := tool(t, "profile-filter-doc", map[string]any{})
		if !strings.Contains(guide, "main virtual warehouse") || !strings.Contains(guide, "MCP key") {
			t.Fatal("expected main connection and key access contract, got absent")
		}
	})

	for _, test := range []struct {
		name, filter, profile, page, explanation string
	}{
		{"Count", profilesHTTPCount, "00000000-0000-0000-0000-000000000002", `{"profiles":[{"kpid":"00000000-0000-0000-0000-000000000001",` + profilesHTTPUpdated + `,"attributes":{"label":"same"}}],"schema":` + profilesHTTPLabelSchema + `,"total":1,"evaluation":{"at":"` + profilesHTTPAt + `","match":1,"noMatch":2,"unknown":1}}`, `{"kpid":"00000000-0000-0000-0000-000000000002","at":"` + profilesHTTPAt + `","rule":{"kind":"and","result":"UNKNOWN","rules":[{"kind":"count","result":"UNKNOWN","count":{` + profilesHTTPBounds + `,"certain":0,"uncertain":1,"money":[{"path":[0],"field":` + profilesHTTPField + `,"nativeCurrency":"EUR","requested":{"operator":"is greater than or equal to","values":[100.00001]},"normalized":{"operator":"is greater than or equal to","values":[100.0001]}}]}}]}}`},
		{"Sum", profilesHTTPSum, "00000000-0000-0000-0000-000000000001", `{"profiles":[{"kpid":"00000000-0000-0000-0000-000000000001",` + profilesHTTPUpdated + `,"attributes":{"label":"same"}}],"schema":` + profilesHTTPLabelSchema + `,"total":1,"evaluation":{"at":"` + profilesHTTPAt + `","match":1,"noMatch":3,"unknown":0}}`, `{"kpid":"00000000-0000-0000-0000-000000000001","at":"` + profilesHTTPAt + `","rule":{"kind":"and","result":"MATCH","rules":[{"kind":"sum","result":"MATCH","sum":{` + profilesHTTPBounds + `,"field":` + profilesHTTPField + `,"nativeCurrency":"EUR","bounds":{"lower":170.1253,"upper":170.1253},"certain":2,"uncertain":0}}]}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			page := tool(t, "query-profiles", map[string]any{"filter": test.filter,
				"at": profilesHTTPAt, "properties": []string{"label"}, "limit": 10})
			if page != test.page {
				t.Fatalf("expected MCP page JSON %s, got %s", test.page, page)
			}
			explanation := tool(t, "explain-profile-filter", map[string]any{
				"kpid": test.profile, "filter": test.filter, "at": profilesHTTPAt,
			})
			if explanation != test.explanation {
				t.Fatalf("expected MCP explanation JSON %s, got %s", test.explanation, explanation)
			}
		})
	}

	t.Run("projection", func(t *testing.T) {
		actual := tool(t, "query-profiles", map[string]any{"filter": profilesHTTPAll, "at": profilesHTTPAt,
			"properties": []string{"label", "amount", "document", "settings", "tags"},
			"order":      "label", "first": 3, "limit": 1})
		expected := `{"profiles":[{"kpid":"00000000-0000-0000-0000-000000000003",` + profilesHTTPUpdated +
			`,"attributes":{"amount":9007199254740993.0000000001,"document":{"v":null},` +
			`"settings":{"value":"nested"},"tags":["","x"]}}],"schema":{"kind":"object","properties":[` +
			`{"name":"label","type":{"kind":"string"},"readOptional":true},` +
			`{"name":"amount","type":{"kind":"decimal","precision":38,"scale":10},"readOptional":true},` +
			`{"name":"document","type":{"kind":"json"},"readOptional":true},` +
			`{"name":"settings","type":{"kind":"object","properties":[` +
			`{"name":"value","type":{"kind":"string"},"readOptional":true}]},"readOptional":true},` +
			`{"name":"tags","type":{"kind":"array","elementType":{"kind":"string"}},"readOptional":true}]},` +
			`"total":4,"evaluation":{"at":"` + profilesHTTPAt + `","match":4,"noMatch":0,"unknown":0}}`
		if actual != expected {
			t.Fatalf("expected MCP projection JSON %s, got %s", expected, actual)
		}
	})

	t.Run("Record", func(t *testing.T) {
		actual := tool(t, "query-profiles", map[string]any{"filter": profilesHTTPRecord,
			"properties": []string{"label", "amount", "document"}, "limit": 1})
		expected := `{"profiles":[{"kpid":"00000000-0000-0000-0000-000000000003",` + profilesHTTPUpdated +
			`,"attributes":{"amount":9007199254740993.0000000001,"document":{"v":null}}}],` +
			`"schema":{"kind":"object","properties":[` +
			`{"name":"label","type":{"kind":"string"},"readOptional":true},` +
			`{"name":"amount","type":{"kind":"decimal","precision":38,"scale":10},"readOptional":true},` +
			`{"name":"document","type":{"kind":"json"},"readOptional":true}]},"total":1}`
		if actual != expected {
			t.Fatalf("expected MCP Record JSON %s, got %s", expected, actual)
		}
	})

}
