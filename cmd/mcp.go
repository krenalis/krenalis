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
	"fmt"
	"net/http"
	"strings"

	"github.com/meergo/meergo/core/datastore"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/types"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// userSchemaInfoForMCPClient takes the user schema, a function that maps a
// types.Type to the corresponding type on the currently connected warehouse,
// and returns a data structure containing information about the user schema
// that can be serialized to JSON and sent to the MCP client.
func userSchemaInfoForMCPClient(userSchema types.Type, columnTypeDescription func(types.Type) (string, error)) []any {
	var info []any
	for path, p := range types.WalkAll(userSchema) {
		if p.Type.Kind() == types.ObjectKind {
			continue
		}
		colTypDescription, _ := columnTypeDescription(p.Type)
		info = append(info, map[string]any{
			"userSchemaProperty": map[string]any{
				"path": path,
				"type": p.Type,
			},
			"userViewColumn": map[string]any{
				"name":     strings.ReplaceAll(path, ".", "_"),
				"type":     colTypDescription,
				"nullable": true,
			},
		})
	}
	// Add information about the "__id__" and "__last_change_time__".
	info = append(info, map[string]any{
		"userViewColumn": map[string]any{
			"name":        "__id__",
			"type":        "uuid",
			"nullable":    true,
			"description": "ID that uniquely identifies the user. It doesn't have a corresponding property in the user schema. It's used to reference the 'events.user' column.",
		},
	})
	info = append(info, map[string]any{
		"userViewColumn": map[string]any{
			"name":        "__last_change_time__",
			"type":        "timestamp without time zone",
			"nullable":    true,
			"description": "ID of the user's last update. It doesn't have a corresponding property in the user schema.",
		},
	})
	return info
}

// eventSchemaInfoForMCPClient holds a data structure containing information
// about the event schema that can be serialized to JSON and sent to the MCP
// client.
var eventSchemaInfoForMCPClient []any

func init() {
	// Initialize eventSchemaInfoForMCPClient.
	for path, p := range types.WalkAll(events.Schema) {
		if p.Type.Kind() == types.ObjectKind {
			continue
		}
		column := datastore.EventColumnByPath(path)
		info := map[string]any{
			"eventSchemaProperty": map[string]any{
				"path": path,
				"type": p.Type,
			},
			"eventTableColumn": map[string]any{
				"name":     column.Name,
				"type":     column.Type,
				"nullable": column.Nullable,
			},
		}
		if path == "user" {
			info["eventTableColumn"].(map[string]any)["description"] = "If present, indicates the user (in the users view) associated with this event."
		}
		eventSchemaInfoForMCPClient = append(eventSchemaInfoForMCPClient, info)
	}
}

// newMCPServer returns a new MCP server.
func newMCPServer(apisServer *apisServer) *mcpServer {

	// MCP server.
	m := server.NewMCPServer("Meergo MCP server", "0.0.0", server.WithRecovery())

	// Prompt for getting a description of the user schema.
	m.AddPrompt(
		mcp.NewPrompt("describe-user-schema", mcp.WithPromptDescription("Get a better understanding of the user schema")),
		func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			pr := mcp.NewGetPromptResult("Get a better understanding of the user schema", []mcp.PromptMessage{
				mcp.NewPromptMessage(
					mcp.RoleUser,
					mcp.NewTextContent(
						"Retrieve the current user schema associated with the workspace and provide a high-level description of it,"+
							" indicating which parts of it could be improved."+
							" Also explain the relationship between the properties of the user schema and the columns of the corresponding 'users' view on the data warehouse.",
					),
				),
			})
			return pr, nil
		})

	// Prompt for getting a description of the event schema.
	m.AddPrompt(
		mcp.NewPrompt("describe-event-schema", mcp.WithPromptDescription("Get a better understanding of the event schema")),
		func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			pr := mcp.NewGetPromptResult("Get a better understanding of the event schema", []mcp.PromptMessage{
				mcp.NewPromptMessage(
					mcp.RoleUser,
					mcp.NewTextContent("Retrieve the event schema and provide a high-level description of it."+
						" Also explain the relationship between the properties of the event schema and the columns of the corresponding 'events' table on the data warehouse.",
					),
				),
			})
			return pr, nil
		})

	// Tool that exposes information about the warehouse.
	warehouseInformation := mcp.NewTool("warehouse-information",
		// See https://modelcontextprotocol.io/docs/concepts/tools#tool-definition-structure.
		mcp.WithDescription("Return information about the data warehouse connected to the workspace"),
		mcp.WithTitleAnnotation("Information about the warehouse"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	m.AddTool(warehouseInformation, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		typ, _ := ws.Warehouse()
		information := fmt.Sprintf("Connected to the workspace there is a %s data warehouse", typ)
		return mcp.NewToolResultText(string(information)), nil
	})

	// Tool that exposes the user schema.
	userSchema := mcp.NewTool("user-schema",
		// See https://modelcontextprotocol.io/docs/concepts/tools#tool-definition-structure.
		mcp.WithDescription(
			"Return the user schema (with details of all its properties and the corresponding data warehouse columns) related to the Meergo workspace."+
				" Information is returned about the properties of the user schema in Meergo (with their types),"+
				" and about the corresponding column of the 'users' view in the data warehouse (along with its column type), where the user information is actually stored."+
				" All user schema properties in Meergo are always nullable, as any of them can be omitted."+
				" Unlike the event schema, which is fixed for each workspace, the user schema can be modified and thus change over time.",
		),
		mcp.WithTitleAnnotation("User schema of the workspace"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
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
		schemaInfo := userSchemaInfoForMCPClient(ws.UserSchema, ws.ColumnTypeDescription)
		encoded, err := json.Marshal(schemaInfo)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(encoded)), nil
	})

	// Tool that exposes the Identity Resolution executions.
	irExecutions := mcp.NewTool("identity-resolution-executions",
		// See https://modelcontextprotocol.io/docs/concepts/tools#tool-definition-structure.
		mcp.WithDescription(
			"Return information about Identity Resolution executions.",
		),
		mcp.WithTitleAnnotation("Information about Identity Resolution execution."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
	)
	m.AddTool(irExecutions, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		startTime, endTime, err := ws.LatestIdentityResolution(ctx)
		if err != nil {
			return nil, err
		}
		var info string
		switch {
		case startTime == nil && endTime == nil:
			info = "Identity Resolution has never been performed on the workspace"
		case startTime != nil && endTime != nil:
			info = fmt.Sprintf("The Identity Resolution procedure has been started on the workspace at %v and ended at %v", startTime, endTime)
		case startTime != nil && endTime == nil:
			info = fmt.Sprintf("The Identity Resolution procedure has been started on the workspace at %v and it's still running", startTime)
		}
		return mcp.NewToolResultText(info), nil
	})

	// Tool that exposes information about the user identities.
	userIdentitiesDoc := mcp.NewTool("user-identities-doc",
		// See https://modelcontextprotocol.io/docs/concepts/tools#tool-definition-structure.
		mcp.WithDescription(
			"Return information about the user identities in Meergo.",
		),
		mcp.WithTitleAnnotation("Documentation about user identities"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	m.AddTool(userIdentitiesDoc, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(strings.Join([]string{
			"The '_user_identities' table contains user identities before they are unified through Identity Resolution and made available in the 'users' view.",
			"The '_user_identities.__connection__' column contains the connection from which the user identity was imported.",
			"If there's no match between the contents of '_user_identities' and the 'users' view, it might be because the Identity Resolution process hasn't been run recently.",
		}, " ")), nil
	})

	// Tool that exposes the event schema.
	eventSchema := mcp.NewTool("event-schema",
		// See https://modelcontextprotocol.io/docs/concepts/tools#tool-definition-structure.
		mcp.WithDescription(
			"Return the event schema (with details of all its properties and the corresponding data warehouse columns)."+
				" Information is returned about the properties of the event schema in Meergo (with their types),"+
				" and about the corresponding column of the 'events' table in the data warehouse (along with its column type), where the user information is actually stored."+
				" Unlike the workspace user schema, which can be modified, the event schema is the same for every workspace and is never modified.",
		),
		mcp.WithTitleAnnotation("Event schema"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	m.AddTool(eventSchema, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		encoded, err := json.Marshal(eventSchemaInfoForMCPClient)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(encoded)), nil
	})

	// Tool that queries the data warehouse.
	queryDataWarehouse := mcp.NewTool("query-data-warehouse",
		mcp.WithDescription("Run a query on the data warehouse connected to the workspace (to retrieve events, users, or other relevant data) and returns the results for analysis."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Query to execute on the workspace's data warehouse to retrieve data")),
		mcp.WithTitleAnnotation("Query the data warehouse of the workspace"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
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
