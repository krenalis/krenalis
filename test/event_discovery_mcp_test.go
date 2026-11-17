// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
)

// TestEventDiscoveryMCP exercises real authentication, strict arguments and
// complete text/structured responses on an isolated PostgreSQL workspace.
func TestEventDiscoveryMCP(t *testing.T) {

	if testing.Short() {
		t.Skip("requires isolated PostgreSQL, NATS and the server")
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.SetProfileSchema(types.Object([]types.Property{{Name: "email", Type: types.String(), ReadOptional: true}}))
	k.Start()
	defer k.Stop()
	k.UpdateIdentityResolutionSettings(false, nil)
	var settings krenalistester.DBSettings
	err := json.Unmarshal(krenalistester.PostgresWarehouseSettings(), &settings)
	if err != nil {
		t.Fatalf("expected isolated warehouse settings, got %v", err)
	}
	pool, err := krenalistester.ConnectionPool(t.Context(), &settings)
	if err != nil {
		t.Fatalf("expected isolated warehouse connection, got %v", err)
	}
	defer pool.Close()
	_, err = pool.Exec(t.Context(), `INSERT INTO krenalis_events
(message_id,event,name,timestamp,received_at,sent_at,type,connection_id,anonymous_id,traits,properties)
VALUES
('d1','Custom Purchase',NULL,'2026-01-15','2026-01-15','2026-01-15','track','fixture','a','{}',
'{"amount":15,"nested":{"active":true},"arr":[{"hidden":1}],"literal.dot":3}'),
('d2','custom purchase',NULL,'2026-01-15','2026-01-15','2026-01-15','track','fixture','a','{}','{}'),
('d3',NULL,NULL,'2026-01-15','2026-01-15','2026-01-15','page','fixture','a','{}','{"unnamed":true}'),
('d4',NULL,'','2026-01-15','2026-01-15','2026-01-15','page','fixture','a','{}','{"empty":true}'),
('d5',NULL,'Pricing','2026-01-15','2026-01-15','2026-01-15','page','fixture','a','{}','null')`)
	if err != nil {
		t.Fatalf("expected custom event fixtures, got %v", err)
	}
	_, err = pool.Exec(t.Context(), `INSERT INTO krenalis_events
(message_id,event,timestamp,received_at,sent_at,type,connection_id,anonymous_id,traits,properties)
SELECT 'h'||i,repeat('<',1021)||lpad(i::text,3,'0'),'2026-01-15','2026-01-15','2026-01-15',
'track','fixture','a','{}','{}' FROM generate_series(1,96) AS i`)
	if err != nil {
		t.Fatalf("expected encoded source budget fixtures, got %v", err)
	}
	var key struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	k.Call("POST", "/v1/keys", nil, map[string]any{
		"name": "Event discovery", "workspace": k.WorkspaceID(), "type": "MCP",
	}, &key)
	token, session := key.Token, ""
	profileMCPCall(t, k.Addr(), token, &session, "initialize", map[string]any{
		"protocolVersion": "2024-11-05", "capabilities": map[string]any{},
		"clientInfo": map[string]any{"name": "Event discovery", "version": "1"},
	})
	profileMCPCall(t, k.Addr(), token, &session, "notifications/initialized", map[string]any{})
	tool := func(t *testing.T, name string, args any) profileMCPResponse {
		t.Helper()
		return profileMCPCall(t, k.Addr(), token, &session, "tools/call",
			map[string]any{"name": name, "arguments": args})
	}
	decode := func(t *testing.T, response profileMCPResponse, out any) {
		t.Helper()
		if response.Error != nil || response.Result.IsError || len(response.Result.Content) != 1 ||
			response.Result.Content[0].Type != "text" || len(response.Result.StructuredContent) == 0 {
			t.Fatalf("expected successful structured discovery, got %+v", response)
		}
		text := response.Result.Content[0].Text
		if len(text) > 128*1024 || len(response.Result.StructuredContent) > 128*1024 {
			t.Fatal("expected bounded discovery JSON, got oversized content")
		}
		var plain, structured any
		err := json.Unmarshal([]byte(text), &plain)
		if err != nil {
			t.Fatalf("expected discovery JSON text, got %v", err)
		}
		err = json.Unmarshal(response.Result.StructuredContent, &structured)
		if err != nil {
			t.Fatalf("expected structured discovery JSON, got %v", err)
		}
		if !reflect.DeepEqual(plain, structured) {
			t.Fatal("expected equivalent text and structured discovery, got different values")
		}
		err = json.Unmarshal([]byte(text), out)
		if err != nil {
			t.Fatalf("expected typed discovery, got %v", err)
		}
	}
	start, end := "2026-01-01T00:00:00Z", "2026-02-01T00:00:00Z"
	t.Run("custom source and literal search", func(t *testing.T) {
		var page core.EventSourceDiscoveryResult
		decode(t, tool(t, "list-observed-event-sources", map[string]any{
			"start": start, "end": end, "nameContains": "Custom", "limit": 500,
		}), &page)
		if len(page.Sources) != 1 || page.Sources[0].RecordedName == nil ||
			*page.Sources[0].RecordedName != "Custom Purchase" || page.Truncated || page.Limit != 500 {
			t.Fatalf("expected the custom source with literal case, got %+v", page)
		}
	})
	t.Run("nested fields and no array traversal", func(t *testing.T) {
		var fields core.EventFieldDiscoveryResult
		decode(t, tool(t, "inspect-event-fields", map[string]any{
			"start": start, "end": end,
			"source": map[string]any{"type": "track", "recordedName": "Custom Purchase"},
		}), &fields)
		if fields.Sample.Rows != 1 || fields.Sample.Limit != 200 || len(fields.Fields) != 5 ||
			fields.Roots["properties"].Kinds["object"] != 1 {
			t.Fatalf("expected five observed paths, got %+v", fields)
		}
		for _, field := range fields.Fields {
			if len(field.Path) == 2 && field.Path[1] == "literal.dot" && field.Property != "" {
				t.Fatalf("expected unrepresentable key kept only as segments, got %+v", field)
			}
		}
	})
	t.Run("encoded source budget", func(t *testing.T) {
		var page core.EventSourceDiscoveryResult
		decode(t, tool(t, "list-observed-event-sources", map[string]any{
			"start": start, "end": end, "nameContains": "<", "limit": 500,
		}), &page)
		if !page.Truncated || len(page.Sources) == 0 || len(page.Sources) >= 96 || page.OmittedLongNames != 0 {
			t.Fatalf("expected visible encoded-byte truncation, got %d sources, truncated=%t",
				len(page.Sources), page.Truncated)
		}
	})
	t.Run("null and empty page names", func(t *testing.T) {
		for _, name := range []any{nil, "", "Pricing"} {
			var fields core.EventFieldDiscoveryResult
			decode(t, tool(t, "inspect-event-fields", map[string]any{
				"start": start, "end": end, "source": map[string]any{"type": "page", "recordedName": name},
			}), &fields)
			if fields.Sample.Rows != 1 || name == "Pricing" && fields.Roots["properties"].Kinds["null"] != 1 {
				t.Fatalf("expected one exact page and separate JSON null, got %+v", fields)
			}
		}
	})
	t.Run("strict arguments", func(t *testing.T) {
		for _, test := range []struct {
			name string
			args string
		}{
			{"list-observed-event-sources", `{"extra":true}`},
			{"list-observed-event-sources", `{"limit":null}`},
			{"list-observed-event-sources", `{"limit":0}`},
			{"list-observed-event-sources", `{"limit":501}`},
			{"list-observed-event-sources", `{"limit":1.5}`},
			{"list-observed-event-sources", `{"type":null}`},
			{"list-observed-event-sources", `{"type":"identify"}`},
			{"list-observed-event-sources", `{"start":"2026-01-01T00:00:00Z"}`},
			{"list-observed-event-sources", `{"start":"2026-01-01T00:00:00.0000001Z","end":"2026-02-01T00:00:00Z"}`},
			{"list-observed-event-sources", `{"start":"2026-02-01T00:00:00Z","end":"2026-01-01T00:00:00Z"}`},
			{"inspect-event-fields", `{}`},
			{"inspect-event-fields", `{"source":null}`},
			{"inspect-event-fields", `{"source":{"type":"page"}}`},
			{"inspect-event-fields", `{"source":{"type":"page","name":""}}`},
			{"inspect-event-fields", `{"source":{"type":"page","recordedName":42}}`},
			{"inspect-event-fields", `{"source":{"type":"page","recordedName":null},"sampleLimit":0}`},
			{"inspect-event-fields", `{"source":{"type":"page","recordedName":null},"sampleLimit":1001}`},
			{"inspect-event-fields", `{"source":{"type":"page","recordedName":null,"recordedName":""}}`},
		} {
			response := tool(t, test.name, json.RawMessage(test.args))
			if response.Error != nil || !response.Result.IsError || len(response.Result.Content) != 1 ||
				!strings.HasPrefix(response.Result.Content[0].Text, "InvalidArguments:") {
				t.Fatalf("expected InvalidArguments for %s, got %+v", test.args, response)
			}
		}
	})
	t.Run("key type and revocation", func(t *testing.T) {
		apiToken := k.CreateWorkspaceRestrictedAPIKey("Not an MCP key")
		wrongType := "mcp_" + strings.TrimPrefix(apiToken, "api_")
		response := profileMCPCall(t, k.Addr(), wrongType, &session, "tools/call",
			map[string]any{"name": "list-observed-event-sources", "arguments": map[string]any{}})
		if response.Error == nil || response.Error.Message != "MCP key is invalid" || len(response.Result.Content) != 0 {
			t.Fatalf("expected wrong key type rejection, got %+v", response.Error)
		}
		k.Call("DELETE", "/v1/keys/"+key.ID, nil, nil, nil)
		for _, name := range []string{"list-observed-event-sources", "inspect-event-fields"} {
			response = tool(t, name, map[string]any{})
			if response.Error == nil || response.Error.Message != "MCP key is invalid" || len(response.Result.Content) != 0 {
				t.Fatalf("expected revoked key rejection on the same session, got %+v", response.Error)
			}
		}
	})

}

// TestEventDiscoveryMCPSnowflake checks the same two tools on a live test
// warehouse, without opening the production behavioral filter filter gate.
func TestEventDiscoveryMCPSnowflake(t *testing.T) {

	if testing.Short() || os.Getenv("KRENALIS_SKIP_SNOWFLAKE_TESTS") == "true" {
		t.Skip("remote Snowflake tests are disabled")
	}
	if !krenalistester.UsesInternalServer() {
		t.Skip("requires the isolated internal-server overlay")
	}
	k, _ := newProfileTransportFixture(t, "Snowflake")
	var key struct {
		Token string `json:"token"`
	}
	k.Call("POST", "/v1/keys", nil, map[string]any{
		"name": "Snowflake event discovery", "workspace": k.WorkspaceID(), "type": "MCP",
	}, &key)
	session := ""
	profileMCPCall(t, k.Addr(), key.Token, &session, "initialize", map[string]any{
		"protocolVersion": "2024-11-05", "capabilities": map[string]any{},
		"clientInfo": map[string]any{"name": "Snowflake event discovery", "version": "1"},
	})
	profileMCPCall(t, k.Addr(), key.Token, &session, "notifications/initialized", map[string]any{})
	read := func(t *testing.T, name string, args map[string]any, out any) {
		t.Helper()
		response := profileMCPCall(t, k.Addr(), key.Token, &session, "tools/call",
			map[string]any{"name": name, "arguments": args})
		if response.Error != nil || response.Result.IsError || len(response.Result.Content) != 1 {
			t.Fatalf("expected complete discovery result, got %+v", response)
		}
		var text, structured any
		err := json.Unmarshal([]byte(response.Result.Content[0].Text), &text)
		if err != nil {
			t.Fatalf("expected discovery JSON, got %v", err)
		}
		err = json.Unmarshal(response.Result.StructuredContent, &structured)
		if err != nil {
			t.Fatalf("expected structured discovery JSON, got %v", err)
		}
		if !reflect.DeepEqual(text, structured) {
			t.Fatal("expected equivalent text and structured content, got different values")
		}
		err = json.Unmarshal(response.Result.StructuredContent, out)
		if err != nil {
			t.Fatalf("expected typed discovery result, got %v", err)
		}
	}
	var page core.EventSourceDiscoveryResult
	read(t, "list-observed-event-sources", map[string]any{
		"start": "2026-01-01T00:00:00Z", "end": profilesHTTPAt, "type": "track", "nameContains": "Order Completed",
	}, &page)
	if page.Truncated || len(page.Sources) != 1 || page.Sources[0].RecordedName == nil ||
		*page.Sources[0].RecordedName != "Order Completed" {
		t.Fatalf("expected one stored Order Completed source, got %+v", page)
	}
	var fields core.EventFieldDiscoveryResult
	read(t, "inspect-event-fields", map[string]any{
		"start": "2026-01-01T00:00:00Z", "end": profilesHTTPAt,
		"source": map[string]any{"type": "track", "recordedName": "Order Completed"},
	}, &fields)
	if fields.Sample.Rows != 5 || fields.Sample.HasMoreRows || fields.Sample.FieldsTruncated ||
		fields.Roots["properties"].Kinds["object"] != 5 || len(fields.Fields) != 2 {
		t.Fatalf("expected all five stored messages and two paths, got %+v", fields)
	}
	for _, field := range fields.Fields {
		switch field.Property {
		case "properties.total":
			if field.Present != 5 || field.Kinds["number"] != 5 {
				t.Fatalf("expected five observed numeric totals, got %+v", field)
			}
		case "properties.currency":
			if field.Present != 4 || field.Kinds["string"] != 4 {
				t.Fatalf("expected four observed currencies, got %+v", field)
			}
		default:
			t.Fatalf("expected a fixture path, got %+v", field)
		}
	}

}
