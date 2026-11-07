// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mcp

import (
	"context"
	_ "embed"
	"strings"
	"time"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	maxProfileQueryArgumentBytes = 500 * 1024
	maxProfileQueryResultBytes   = 128 * 1024
)

//go:embed profile_filter.md
var profileFilterDocumentation string

var profileFilterDocumentationTool = server.ServerTool{
	Tool: mcp.NewTool("profile-filter-doc",
		mcp.WithDescription("Read the supported Record/Count/Sum/Average filter syntax before using query-profiles."),
		mcp.WithTitleAnnotation("Profile filtering documentation"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	),
	Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ws, err := workspaceFromCtx(ctx)
		if err != nil {
			return nil, err
		}
		err = ws.ConsumeRateLimitCapacity(ctx, 1)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(profileFilterDocumentation), nil
	},
}

var queryProfilesTool = server.ServerTool{
	Tool: mcp.NewTool("query-profiles",
		mcp.WithDescription("Query profiles and segments using the same core.Filter JSON as GET /v1/profiles."+
			" Read profile-filter-doc first and use schema property paths, not warehouse column names."+
			" Returns matching profiles, schema, total, and Unified evaluation with at/match/noMatch/unknown."+
			" UNKNOWN profiles are not members and are not NO_MATCH; technical errors are not UNKNOWN."+
			" Reuse evaluation.at on later pages to anchor periods; this does not freeze the data."+
			" Configured MCP settings enable access, but execution uses the main Krenalis connection."+
			" The separate MCP user's table/column permissions do not restrict this tool or its event conditions."+
			" Snowflake uses the main virtual warehouse, not the MCP virtual warehouse; behavioral filters remain unavailable."+
			" Do not use SQL to emulate unsupported Unified capabilities."),
		mcp.WithString("filter", mcp.Description("core.Filter encoded as JSON text, not an object or SQL."+
			" Numbers inside the text retain exact digits. Omit to list all profiles; null is invalid.")),
		mcp.WithArray("properties", mcp.WithStringItems(),
			mcp.Description("Top-level profile properties to return; omission requests all. Request only needed data.")),
		mcp.WithString("order", mcp.Description("Scalar profile property for sorting; omission uses update time.")),
		mcp.WithBoolean("orderDesc", mcp.DefaultBool(false)),
		mcp.WithInteger("first", mcp.Min(0), mcp.Max(2147483647), mcp.DefaultNumber(0)),
		mcp.WithInteger("limit", mcp.Min(1), mcp.Max(1000), mcp.DefaultNumber(20)),
		mcp.WithString("at", mcp.Description("Optional nonfuture RFC3339 instant with microsecond precision."+
			" Only allowed with behavioral filters. Reuse evaluation.at when fetching subsequent pages.")),
		mcp.WithTitleAnnotation("Query unified profiles"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	),
	Handler: queryProfiles,
}

type profileQueryProfile struct {
	KPID       string     `json:"kpid"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	Attributes json.Value `json:"attributes"`
}

// profileQueryErrorResult exposes actionable request errors and hides failures.
func profileQueryErrorResult(err error) (*mcp.CallToolResult, error) {
	var message string
	if invalid, ok := errors.AsType[*errors.BadRequestError](err); ok {
		message = "InvalidArguments: " + invalid.Error()
	} else if invalid, ok := errors.AsType[*errors.UnprocessableError](err); ok {
		message = string(invalid.Code) + ": " + invalid.Error()
	} else {
		return nil, errors.New("cmd/internal/mcp: profile query failed")
	}
	if len(message) > 2048 {
		message = strings.ToValidUTF8(message[:2048], "") + "..."
	}
	return mcp.NewToolResultError(message), nil
}

func queryProfiles(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	ws, err := workspaceFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	err = ws.ConsumeRateLimitCapacity(ctx, 1)
	if err != nil {
		return nil, err
	}
	if !ws.HasWarehouseMCPSettings() {
		return mcp.NewToolResultError("MCPNotConfigured: configure MCP warehouse settings to enable profile queries"), nil
	}

	// Decode the wire arguments without converting filter operands to float64.
	raw := req.Params.RawArguments
	if len(raw) == 0 && req.Params.Arguments == nil {
		raw = []byte("{}")
	}
	if len(raw) == 0 || len(raw) > maxProfileQueryArgumentBytes || !json.Valid(raw) || !json.Value(raw).IsObject() {
		return profileQueryErrorResult(errors.BadRequest("arguments must be an object of at most 500 KiB"))
	}
	for name, value := range json.Value(raw).Properties() {
		switch name {
		case "properties", "filter", "order", "orderDesc", "first", "limit", "at":
			if value.IsNull() {
				return profileQueryErrorResult(errors.BadRequest("arguments cannot be null"))
			}
		default:
			return profileQueryErrorResult(errors.BadRequest("unknown query-profiles argument"))
		}
	}
	var args struct {
		Properties []string `json:"properties"`
		Filter     *string  `json:"filter"`
		Order      string   `json:"order"`
		OrderDesc  bool     `json:"orderDesc"`
		First      int      `json:"first"`
		Limit      int      `json:"limit"`
		At         *string  `json:"at"`
	}
	args.Limit = 20
	err = json.Unmarshal(raw, &args)
	if err != nil {
		return profileQueryErrorResult(errors.BadRequest("invalid query-profiles arguments"))
	}
	query := core.ProfileQuery{Properties: args.Properties, Order: args.Order, OrderDesc: args.OrderDesc,
		First: args.First, Limit: args.Limit}
	if args.Filter != nil {
		err = json.Unmarshal([]byte(*args.Filter), &query.Filter)
		if err != nil {
			return profileQueryErrorResult(errors.BadRequest("invalid filter: %s", err))
		}
		if query.Filter == nil {
			return profileQueryErrorResult(errors.BadRequest("filter cannot be null"))
		}
	}
	if args.At != nil {
		if *args.At == "" {
			return profileQueryErrorResult(errors.BadRequest("at cannot be empty"))
		}
		query.At = *args.At
	}

	result, err := ws.QueryProfiles(ctx, query)
	if err != nil {
		return profileQueryErrorResult(err)
	}
	if len(result.Profiles) > args.Limit {
		return nil, errors.New("cmd/internal/mcp: profile page exceeds requested limit")
	}

	var response struct {
		Profiles   []profileQueryProfile   `json:"profiles"`
		Schema     types.Type              `json:"schema"`
		Total      int                     `json:"total"`
		Evaluation *core.ProfileEvaluation `json:"evaluation,omitzero"`
	}
	response.Profiles = make([]profileQueryProfile, len(result.Profiles))
	response.Schema, response.Total, response.Evaluation = result.Schema, result.Total, result.Evaluation
	bytes := 0
	for i, profile := range result.Profiles {
		attributes, err := types.Marshal(profile.Attributes, result.Schema)
		if err != nil {
			return nil, errors.New("cmd/internal/mcp: cannot encode profile attributes")
		}
		if len(attributes) > maxProfileQueryResultBytes-bytes {
			return profileQueryErrorResult(errors.Unprocessable(core.ProfileResultTooLarge,
				"MCP profile result exceeds 128 KiB; reduce limit or request fewer properties"))
		}
		bytes += len(attributes)
		response.Profiles[i] = profileQueryProfile{profile.KPID, profile.UpdatedAt, attributes}
	}
	// Encode before publishing any content; an oversized page is never truncated.
	encoded, err := json.Marshal(response)
	if err != nil {
		return nil, errors.New("cmd/internal/mcp: cannot encode profile result")
	}
	if len(encoded) > maxProfileQueryResultBytes {
		return profileQueryErrorResult(errors.Unprocessable(core.ProfileResultTooLarge,
			"MCP profile result exceeds 128 KiB; reduce limit or request fewer properties"))
	}

	return mcp.NewToolResultText(string(encoded)), nil
}
