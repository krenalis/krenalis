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

// TestPromptsList tests the MCP server method "prompts/list".
func TestPromptsList(t *testing.T) {

	mcpServer := NewMCPServer(nil)
	defer mcpServer.Close(context.Background())

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
