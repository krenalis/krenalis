//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package mcp

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// TestPromptsList tests the MCP server method "prompts/list".
func TestPromptsList(t *testing.T) {

	mcpServer := NewMCPServer(nil)
	testServer := httptest.NewServer(mcpServer)

	mcpClient, err := initMCPClient(t, testServer.Client(), testServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := mcpClient.jsonRPCRequest("prompts/list", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var got struct {
		Result struct {
			Prompts []any
		}
	}
	err = json.NewDecoder(resp.Body).Decode(&got)
	if err != nil {
		t.Fatal(err)
	}

	const expectedPromptCount = 5
	if len(got.Result.Prompts) != expectedPromptCount {
		t.Fatalf("expected %d prompts, got %d", expectedPromptCount, len(got.Result.Prompts))
	}
	t.Logf("the MCP server correctly returned %d prompts", len(got.Result.Prompts))

}
