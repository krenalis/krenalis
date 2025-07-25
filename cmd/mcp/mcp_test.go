//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// This file provides a MCP client that can be used for testing the MCP server.

// Here are some useful resources:
//
// - https://modelcontextprotocol.io/specification/2025-06-18/basic
// - https://www.jsonrpc.org/specification

type testMCPClient struct {
	t         *testing.T
	client    *http.Client
	serverURL string
	sessionID string
}

// initMCPClient initializes and return an MCP client that can be used when
// testing this package.
func initMCPClient(t *testing.T, client *http.Client, serverURL string) (*testMCPClient, error) {

	mcpClient := &testMCPClient{
		t:         t,
		client:    client,
		serverURL: serverURL,
		// The sessionID is set later.
	}

	// «The client MUST initiate this phase by sending an initialize request.»
	//
	// Source: https://modelcontextprotocol.io/specification/2024-11-05/basic/lifecycle#lifecycle-phases.
	resp, err := mcpClient.jsonRPCRequest("initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"roots":    map[string]any{},
			"sampling": map[string]any{},
		},
		"clientInfo": map[string]any{
			"name":    "MeergoTestClient",
			"version": "1.0.0",
		},
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// «A server using the Streamable HTTP transport MAY assign a session ID at
	// initialization time, by including it in an Mcp-Session-Id header on the
	// HTTP response containing the InitializeResult.»
	//
	// Source: https://modelcontextprotocol.io/specification/2025-06-18/basic/transports#session-management.
	if sessionID, ok := resp.Header["Mcp-Session-Id"]; ok && len(sessionID) > 0 {
		mcpClient.sessionID = sessionID[0]
	} else {
		mcpClient.sessionID = uuid.New().String()
	}

	// «After successful initialization, the client MUST send an initialized notification to indicate it is ready to begin normal operations.»
	//
	// Source:https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle#lifecycle-phases
	resp, err = mcpClient.jsonRPCRequest("notifications/initialized", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return mcpClient, nil
}

func (mcp *testMCPClient) jsonRPCRequest(method string, params map[string]any) (*http.Response, error) {

	if params == nil {
		params = map[string]any{}
	}

	// Prepare the JSON-RPC body.
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      mcp.sessionID,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return nil, err
	}

	// Prepare and send the POST request.
	mcp.t.Logf("POST to %s, JSON-RPC method: %q", mcp.serverURL, method)
	req, err := http.NewRequest("POST", mcp.serverURL, io.NopCloser(bytes.NewReader(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer C8MdB29AVjo5DMF6dG7tcyR41faJYDkx6lxDL1Djqzo") // just a random key.
	if mcp.sessionID != "" {
		req.Header.Add("Mcp-Session-Id", mcp.sessionID)
	}
	resp, err := mcp.client.Do(req)
	if err != nil {
		return nil, err
	}

	// Check the status code.
	if resp.StatusCode == 200 {
		mcp.t.Logf("server responded with HTTP status code %d", resp.StatusCode)
	} else {
		mcp.t.Fatalf("server responded with unexpected HTTP status code %d", resp.StatusCode)
	}

	return resp, nil
}
