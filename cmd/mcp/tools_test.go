// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mcp

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// TestToolsList tests the MCP server method "tools/list".
func TestToolsList(t *testing.T) {

	mcpServer := NewMCPServer(nil)
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
