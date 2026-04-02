// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mcp

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

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
			Tools []any
		}
	}
	err = json.NewDecoder(resp.Body).Decode(&got)
	if err != nil {
		t.Fatal(err)
	}

	const expectedToolCount = 7
	if len(got.Result.Tools) != expectedToolCount {
		t.Fatalf("expected %d tools, got %d", expectedToolCount, len(got.Result.Tools))
	}
	t.Logf("the MCP server correctly returned %d tools", len(got.Result.Tools))

}

func TestQueryDataWarehouseToolMetadata(t *testing.T) {
	mcpServer := NewMCPServer(nil)
	defer mcpServer.Close(context.Background())

	testServer := httptest.NewServer(mcpServer)
	defer testServer.Close()

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
			Tools []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				InputSchema struct {
					Properties map[string]struct {
						Description string `json:"description"`
					} `json:"properties"`
				} `json:"inputSchema"`
				Annotations struct {
					ReadOnlyHint    *bool `json:"readOnlyHint"`
					DestructiveHint *bool `json:"destructiveHint"`
					IdempotentHint  *bool `json:"idempotentHint"`
					OpenWorldHint   *bool `json:"openWorldHint"`
				} `json:"annotations"`
			} `json:"tools"`
		} `json:"result"`
	}
	err = json.NewDecoder(resp.Body).Decode(&got)
	if err != nil {
		t.Fatal(err)
	}

	for _, tool := range got.Result.Tools {
		if tool.Name == "warehouse-information" {
			if tool.Description != warehouseInformationToolDescription {
				t.Fatalf("warehouse-information description = %q, want %q", tool.Description, warehouseInformationToolDescription)
			}
			if tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint {
				t.Fatalf("warehouse-information openWorldHint = %v, want false", tool.Annotations.OpenWorldHint)
			}
			continue
		}
		if tool.Name != "query-data-warehouse" {
			continue
		}
		if tool.Description != queryDataWarehouseToolDescription {
			t.Fatalf("query-data-warehouse description = %q, want %q", tool.Description, queryDataWarehouseToolDescription)
		}
		queryArg, ok := tool.InputSchema.Properties["query"]
		if !ok {
			t.Fatal("query-data-warehouse tool is missing the query argument")
		}
		if queryArg.Description != queryDataWarehouseQueryDescription {
			t.Fatalf("query-data-warehouse query description = %q, want %q", queryArg.Description, queryDataWarehouseQueryDescription)
		}
		if tool.Annotations.ReadOnlyHint == nil || !*tool.Annotations.ReadOnlyHint {
			t.Fatalf("query-data-warehouse readOnlyHint = %v, want true", tool.Annotations.ReadOnlyHint)
		}
		if tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint {
			t.Fatalf("query-data-warehouse destructiveHint = %v, want false", tool.Annotations.DestructiveHint)
		}
		if tool.Annotations.IdempotentHint == nil || !*tool.Annotations.IdempotentHint {
			t.Fatalf("query-data-warehouse idempotentHint = %v, want true", tool.Annotations.IdempotentHint)
		}
		if tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint {
			t.Fatalf("query-data-warehouse openWorldHint = %v, want false", tool.Annotations.OpenWorldHint)
		}
		return
	}

	t.Fatal("query-data-warehouse tool not found")
}
