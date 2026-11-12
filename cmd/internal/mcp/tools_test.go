// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mcp

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
)

// TestRetiredSQLTool rejects calls to the removed SQL tool.
func TestRetiredSQLTool(t *testing.T) {
	mcpServer := NewMCPServer(nil)
	defer mcpServer.Close(t.Context())
	testServer := httptest.NewServer(mcpServer)
	defer testServer.Close()
	client, err := initMCPClient(t, testServer.Client(), testServer.URL)
	if err != nil {
		t.Fatalf("expected MCP client, got %v", err)
	}
	response, err := client.jsonRPCRequest("tools/call", map[string]any{
		"name": "query-data-warehouse", "arguments": map[string]any{"query": "SELECT 1"},
	})
	if err != nil {
		t.Fatalf("expected MCP response, got %v", err)
	}
	defer response.Body.Close()
	var body struct {
		Error *struct {
			Code int `json:"code"`
		} `json:"error"`
		Result json.Value `json:"result"`
	}
	err = json.NewDecoder(response.Body).Decode(&body)
	if err != nil {
		t.Fatalf("expected decoded MCP error, got %v", err)
	}
	if body.Error == nil || body.Error.Code != -32602 || len(body.Result) != 0 {
		t.Fatalf("expected unknown-tool error without result, got %+v", body)
	}
}

// TestToolsList tests the MCP server method "tools/list".
func TestToolsList(t *testing.T) {

	mcpServer := NewMCPServer(nil)
	defer mcpServer.Close(context.Background())

	testServer := httptest.NewServer(mcpServer)

	mcpClient, err := initMCPClient(t, testServer.Client(), testServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := mcpClient.jsonRPCRequest("tools/list", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var got struct {
		Result struct {
			Tools []any `json:"tools"`
		} `json:"result"`
	}
	err = json.NewDecoder(resp.Body).Decode(&got)
	if err != nil {
		t.Fatal(err)
	}

	const expectedToolCount = 11
	if len(got.Result.Tools) != expectedToolCount {
		t.Fatalf("expected %d tools, got %d", expectedToolCount, len(got.Result.Tools))
	}
	t.Logf("the MCP server correctly returned %d tools", len(got.Result.Tools))

}

// TestWarehouseInformationToolMetadata checks the remaining warehouse metadata.
func TestWarehouseInformationToolMetadata(t *testing.T) {
	mcpServer := NewMCPServer(nil)
	defer mcpServer.Close(t.Context())
	testServer := httptest.NewServer(mcpServer)
	defer testServer.Close()
	client, err := initMCPClient(t, testServer.Client(), testServer.URL)
	if err != nil {
		t.Fatalf("expected MCP client, got %v", err)
	}
	response, err := client.jsonRPCRequest("tools/list", nil)
	if err != nil {
		t.Fatalf("expected tool list, got %v", err)
	}
	defer response.Body.Close()
	var body struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Annotations struct {
					OpenWorldHint *bool `json:"openWorldHint"`
				} `json:"annotations"`
			} `json:"tools"`
		} `json:"result"`
	}
	err = json.NewDecoder(response.Body).Decode(&body)
	if err != nil {
		t.Fatalf("expected decoded tool list, got %v", err)
	}
	for _, tool := range body.Result.Tools {
		if tool.Name != "warehouse-information" {
			continue
		}
		if tool.Description != warehouseInformationToolDescription ||
			tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint {
			t.Fatalf("expected local warehouse metadata description, got %+v", tool)
		}
		return
	}
	t.Fatal("expected warehouse-information tool, got absent")
}
