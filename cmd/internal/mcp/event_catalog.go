// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mcp

import (
	"context"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const eventCatalogNotice = "Catalog definitions are not evidence of recorded events or historical schema conformity." +
	" Read profile-filter-doc and event-schema before building filters." +
	" Use the exact track event name in Count source.name, never the catalog ID or display name." +
	" For page/screen Count use source.type and an optional recorded source.name," +
	" never the generic catalog ID or label." +
	" Identity and legacy merge definitions do not imply Count support." +
	" Native catalog schemas document meaning; traits/properties filters still use raw JSON semantics." +
	" Do not infer governed bindings, array traversal, sums or other unsupported capabilities from a schema."

var listEventDefinitionsTool = server.ServerTool{
	Tool: mcp.NewTool("list-event-definitions",
		mcp.WithDescription("Discover catalog events by ID, technical name and business description."+
			" Optional query is a literal case-insensitive substring, not semantic search."+
			" Omit it to compare all definitions; use get-event-definition for property schemas."+
			" Catalog access requires an authenticated workspace MCP key but no MCP warehouse settings."),
		mcp.WithString("query", mcp.Description("Optional text, at most 256 UTF-8 bytes.")),
		mcp.WithTitleAnnotation("Discover catalog events"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	),
	Handler: eventCatalog,
}

var getEventDefinitionTool = server.ServerTool{
	Tool: mcp.NewTool("get-event-definition",
		mcp.WithDescription("Read an event definition and its native traits/properties schemas, including descriptions,"+
			" units, nullable fields, nested objects and array element types. "+eventCatalogNotice),
		mcp.WithString("id", mcp.Required(), mcp.Description("Exact catalog ID returned by list-event-definitions.")),
		mcp.WithTitleAnnotation("Read an event definition"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	),
	Handler: eventCatalog,
}

func eventCatalog(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	ws, err := workspaceFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	err = ws.ConsumeRateLimitCapacity(ctx, 1)
	if err != nil {
		return nil, err
	}
	raw := req.Params.RawArguments
	if len(raw) == 0 && req.Params.Arguments == nil {
		raw = []byte("{}")
	}
	if len(raw) == 0 || len(raw) > 2048 || !json.Valid(raw) || !json.Value(raw).IsObject() {
		return mcp.NewToolResultError("InvalidArguments: arguments must be an object of at most 2 KiB"), nil
	}
	var args struct {
		ID    string `json:"id"`
		Query string `json:"query"`
	}
	for name, value := range json.Value(raw).Properties() {
		if !value.IsString() || req.Params.Name == "list-event-definitions" && name != "query" ||
			req.Params.Name == "get-event-definition" && name != "id" {
			return mcp.NewToolResultError("InvalidArguments: unexpected argument or argument type"), nil
		}
	}
	err = json.Unmarshal(raw, &args)
	if err != nil {
		return mcp.NewToolResultError("InvalidArguments: invalid catalog arguments"), nil
	}

	var result any
	switch req.Params.Name {
	case "list-event-definitions":
		definitions, err := ws.EventDefinitions(args.Query)
		if err != nil {
			if _, ok := errors.AsType[*errors.BadRequestError](err); ok {
				return mcp.NewToolResultError("InvalidArguments: invalid catalog query"), nil
			}
			return nil, errors.New("cmd/internal/mcp: event catalog failed")
		}
		summaries := make([]map[string]string, len(definitions))
		for i, definition := range definitions {
			summaries[i] = map[string]string{"id": definition.ID, "type": definition.Type,
				"event": definition.Event, "name": definition.Name, "displayName": definition.DisplayName,
				"description": definition.Description, "trigger": definition.Trigger}
		}
		result = map[string]any{"definitions": summaries, "notice": eventCatalogNotice}
	case "get-event-definition":
		definition, err := ws.EventDefinition(args.ID)
		if err != nil {
			if _, ok := errors.AsType[*errors.BadRequestError](err); ok {
				return mcp.NewToolResultError("InvalidArguments: invalid event definition ID"), nil
			}
			if invalid, ok := errors.AsType[*errors.UnprocessableError](err); ok {
				return mcp.NewToolResultError(string(invalid.Code) + ": event definition does not exist"), nil
			}
			return nil, errors.New("cmd/internal/mcp: event catalog failed")
		}
		result = map[string]any{"definition": definition, "notice": eventCatalogNotice}
	default:
		return nil, errors.New("cmd/internal/mcp: unexpected event catalog tool")
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, errors.New("cmd/internal/mcp: cannot encode event catalog")
	}
	if len(encoded) > 128*1024 {
		return mcp.NewToolResultError("EventCatalogResultTooLarge: narrow the catalog query"), nil
	}

	return mcp.NewToolResultStructured(result, string(encoded)), nil
}
