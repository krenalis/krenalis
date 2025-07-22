//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/events"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// newMCPServer returns a new MCP server.
func newMCPServer(apisServer *apisServer) *mcpServer {

	// MCP server.
	m := server.NewMCPServer("Meergo MCP server", "0.0.0", server.WithRecovery())

	// Tool that exposes the user schema.
	userSchema := mcp.NewTool("user-schema",
		mcp.WithDescription("Returns the user schema (with details of all its properties) related to the Meergo workspace."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	m.AddTool(userSchema, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		apiToken, err := apiTokenFromContext(ctx)
		if err != nil {
			return nil, err
		}
		organizationID, workspaceID, found := apisServer.core.APIKey(apiToken)
		if !found {
			return nil, errors.New("invalid API key")
		}
		org, err := apisServer.core.Organization(ctx, organizationID)
		if err != nil {
			return nil, err
		}
		if workspaceID == 0 {
			return nil, errors.New("the API key must be restricted to a workspace")
		}
		ws, err := org.Workspace(workspaceID)
		if err != nil {
			return nil, err
		}
		encoded, err := json.Marshal(ws.UserSchema)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(encoded)), nil
	})

	// Tool that exposes the event schema.
	eventSchema := mcp.NewTool("event-schema",
		mcp.WithDescription("Returns the event schema (with details of all properties) of Meergo."+
			" The event schema does not change over time but is fixed, encoded within Meergo."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	m.AddTool(eventSchema, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		encoded, err := json.Marshal(events.Schema)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(encoded)), nil
	})

	// Tool that queries the data warehouse.
	queryDataWarehouse := mcp.NewTool("query-data-warehouse",
		mcp.WithDescription("Runs a query on the data warehouse connected to the workspace (to retrieve events, users, or other relevant data) and returns the results for analysis."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Query to execute on the workspace's data warehouse to retrieve data")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	m.AddTool(queryDataWarehouse, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		apiToken, err := apiTokenFromContext(ctx)
		if err != nil {
			return nil, err
		}
		organizationID, workspaceID, found := apisServer.core.APIKey(apiToken)
		if !found {
			return nil, errors.New("invalid API key")
		}
		org, err := apisServer.core.Organization(ctx, organizationID)
		if err != nil {
			return nil, err
		}
		if workspaceID == 0 {
			return nil, errors.New("the API key must be restricted to a workspace")
		}
		query, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ws, err := org.Workspace(workspaceID)
		if err != nil {
			return nil, err
		}
		rows, err := ws.RawQueryWarehouse(ctx, query)
		if err != nil {
			return nil, err
		}
		encoded, err := json.Marshal(rows)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(encoded)), nil
	})

	return &mcpServer{
		server: server.NewStreamableHTTPServer(m),
	}
}

type mcpServer struct {
	server *server.StreamableHTTPServer
}

// ServeHTTP serves the HTTP requests from the MCP clients.
func (mcp *mcpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Retrieve the Meergo API token from the request, otherwise return a Bad
	// Request error to the MCP client.
	apiToken, err := apiTokenFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Inject the API token into the request's context, that will be then passed
	// to the MCP tool code.
	ctxWithToken := context.WithValue(r.Context(), mcpContextKey("api-token"), apiToken)
	r = r.Clone(ctxWithToken)

	mcp.server.ServeHTTP(w, r)
}

type mcpContextKey string

// apiTokenFromContext extracts the API token from the given context, returning
// error if it is not found.
func apiTokenFromContext(ctx context.Context) (string, error) {
	apiToken, ok := ctx.Value(mcpContextKey("api-token")).(string)
	if !ok {
		return "", errors.New("API token not found in context")
	}
	return apiToken, nil
}

// apiTokenFromRequest reads the Meergo API token from the request's
// 'Authorization' header.
func apiTokenFromRequest(r *http.Request) (string, error) {
	auth, ok := r.Header["Authorization"]
	if !ok {
		return "", errors.New("no Authorization header found in request")
	}
	token, found := strings.CutPrefix(auth[0], "Bearer ")
	if !found || token == "" {
		return "", errors.BadRequest("Authorization header is invalid. It should be in the format 'Authorization: Bearer <YOUR_API_KEY>'.")
	}
	return token, nil
}
