//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	_core "github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/errors"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var tools = []server.ServerTool{

	// Tool that exposes information about the warehouse.
	{
		Tool: mcp.NewTool("warehouse-information",
			mcp.WithDescription("Return information about the data warehouse connected to the workspace"),
			mcp.WithTitleAnnotation("Information about the warehouse"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ws, err := workspaceFromCtx(ctx)
			if err != nil {
				return nil, err
			}
			typ, _, _ := ws.Warehouse()
			information := fmt.Sprintf("Connected to the workspace there is a %s data warehouse", typ)
			return mcp.NewToolResultText(string(information)), nil
		},
	},

	// Tool that queries the data warehouse.
	{
		Tool: mcp.NewTool("query-data-warehouse",
			mcp.WithDescription("Run a query on the data warehouse connected to the workspace (to retrieve events, users, or other relevant data) and returns the results for analysis."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Query to execute on the workspace's data warehouse to retrieve data")),
			mcp.WithTitleAnnotation("Query the data warehouse of the workspace"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ws, err := workspaceFromCtx(ctx)
			if err != nil {
				return nil, err
			}
			query, err := req.RequireString("query")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
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
		},
	},

	// Tool that exposes the user schema.
	{
		Tool: mcp.NewTool("user-schema",
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
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ws, err := workspaceFromCtx(ctx)
			if err != nil {
				return nil, err
			}
			schemaInfo := userSchemaInfoForMCPClient(ws.UserSchema, ws.ColumnTypeDescription)
			encoded, err := json.Marshal(schemaInfo)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(encoded)), nil
		},
	},

	// Tool that exposes the event schema.
	{
		Tool: mcp.NewTool("event-schema",
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
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			encoded, err := json.Marshal(eventSchemaInfoForMCPClient)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(encoded)), nil
		},
	},

	// Tool that exposes information about the user identities.
	{
		Tool: mcp.NewTool("user-identities-doc",
			mcp.WithDescription(
				"Return information about the user identities in Meergo.",
			),
			mcp.WithTitleAnnotation("Documentation about user identities"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(strings.Join([]string{
				"The '_user_identities' table contains user identities before they are unified through Identity Resolution and made available in the 'users' view.",
				"The '_user_identities.__connection__' column references the ID (integer) of the connection from which the user identity was imported.",
				"If there's no match between the contents of '_user_identities' and the 'users' view, it might be because the Identity Resolution process hasn't been run recently.",
			}, " ")), nil
		},
	},

	// Tool that exposes the Identity Resolution executions.
	{
		Tool: mcp.NewTool("identity-resolution-executions",
			mcp.WithDescription(
				"Return information about Identity Resolution executions."+
					" Regardless of the language, use the English term Identity Resolution without translating it, as it is a key concept in the software.",
			),
			mcp.WithTitleAnnotation("Information about Identity Resolution execution."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ws, err := workspaceFromCtx(ctx)
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
		},
	},

	// Tool that exposes information about the workspace connections.
	{
		Tool: mcp.NewTool("connections",
			mcp.WithDescription(
				"Return information about the workspace connections."+
					" A workspace can have zero, one or multiple connections."+
					" A connection with role 'source', depending on its type and the external service it's connected to, can import users and events into the data warehouse, and send events to a destination connection."+
					" A connection with role 'destination', depending on its type and the external service it's connected to, can export users read from the data warehouse, and send events received from a source connection to an app."+
					" Once events are imported into the data warehouse by a source connection, they can no longer be re-read or forwarded via a destination connection."+
					" A connection performs its operations (importing, sending, and exporting data) through 'actions'."+
					" Each connection can have zero, one, or multiple 'actions'."+
					" App connections interface with external applications outside Meergo."+
					" Database connections interface with external databases outside Meergo."+
					" File connections work in conjunction with file storage connections to interact with files for reading and writing data."+
					" SDK connections receive data (events and users) from SDKs, browsers, and server-side applications"+
					" Regardless of the language, use the English terms Connection, Source, Destination and Action without translating them, as they are key concepts in the software.",
			),
			mcp.WithTitleAnnotation("Information about workspace connections."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ws, err := workspaceFromCtx(ctx)
			if err != nil {
				return nil, err
			}
			var info []any
			for _, c := range ws.Connections() {
				info = append(info, map[string]any{
					"id":            c.ID,
					"name":          c.Name,
					"connector":     c.Connector,
					"connectorType": c.ConnectorType,
					"role":          c.Role,
					"actionsCount":  c.ActionsCount,
				})
			}
			encoded, err := json.Marshal(info)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(encoded)), nil
		},
	},
}

// workspaceFromCtx retrieves the Meergo workspace from the information provided
// within the context.
//
// If the workspace cannot be retrieved for some reason, this function returns
// an error explaining the problem.
func workspaceFromCtx(ctx context.Context) (*_core.Workspace, error) {
	mcpToken, err := mcpTokenFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	core, err := meergoCoreFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	organizationID, workspaceID, found := core.AccessKey(mcpToken, _core.AccessKeyTypeMCP)
	if !found {
		return nil, errors.New("invalid MCP key")
	}
	org, err := core.Organization(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if workspaceID == 0 {
		return nil, errors.New("the MCP key must be restricted to a workspace")
	}
	ws, err := org.Workspace(workspaceID)
	if err != nil {
		return nil, err
	}
	return ws, nil
}
