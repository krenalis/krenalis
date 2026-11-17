// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mcp

import (
	"context"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var explainProfileFilterTool = server.ServerTool{
	Tool: mcp.NewTool("explain-profile-filter",
		mcp.WithDescription("Explain a supported profile filter for one KPID, including MATCH/NO_MATCH/UNKNOWN"+
			" for each rule, resolved periods, Count occurrence bounds and exact Sum bounds with field and currency."+
			" Average reports exact sum and count inputs instead of a rounded quotient."+
			" Experience reports candidate count, selection ambiguity and certain/possible/impossible absence."+
			" Latest N reports the candidate union and block shortage; uniqueness concerns the selected set."+
			" LastVersusPrevious reports candidate identities, shortage and whether the ordered pair is unique."+
			" Read profile-filter-doc first. Reuse query-profiles evaluation.at to anchor the periods;"+
			" this is a new evaluation, not a snapshot. It does not identify which Record operand caused NULL."+
			" Requires a valid workspace-scoped MCP key and uses the main warehouse connection. Its data access is"+
			" not restricted by former MCP credentials."+
			" Explanations are available on PostgreSQL and Snowflake."),
		mcp.WithString("kpid", mcp.Required(), mcp.Description("KPID of the profile to explain, as a UUID.")),
		mcp.WithString("filter", mcp.Required(), mcp.Description("The same core.Filter JSON text used in query-profiles.")),
		mcp.WithString("at", mcp.Description("Optional nonfuture RFC 3339 instant with microsecond precision.")),
		mcp.WithTitleAnnotation("Explain a profile filter"),
		mcp.WithReadOnlyHintAnnotation(true), mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false), mcp.WithOpenWorldHintAnnotation(false)),
	Handler: explainProfileFilter,
}

func explainProfileFilter(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	ws, err := workspaceFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	err = ws.ConsumeRateLimitCapacity(ctx, 1)
	if err != nil {
		return nil, err
	}
	// Decode the wire arguments without converting filter operands to float64.
	raw := req.Params.RawArguments
	if len(raw) == 0 || len(raw) > maxProfileQueryArgumentBytes || !json.Valid(raw) || !json.Value(raw).IsObject() {
		return profileQueryErrorResult(errors.BadRequest("arguments must be an object of at most 500 KiB"))
	}
	for name, value := range json.Value(raw).Properties() {
		switch name {
		case "kpid", "filter", "at":
			if !value.IsString() {
				return profileQueryErrorResult(errors.BadRequest("explanation arguments must be Strings"))
			}
		default:
			return profileQueryErrorResult(errors.BadRequest("unknown explain-profile-filter argument"))
		}
	}
	var args struct {
		KPID   string  `json:"kpid"`
		Filter string  `json:"filter"`
		At     *string `json:"at"`
	}
	err = json.Unmarshal(raw, &args)
	if err != nil {
		return profileQueryErrorResult(errors.BadRequest("invalid explanation arguments"))
	}
	var filter *core.Filter
	err = json.Unmarshal([]byte(args.Filter), &filter)
	if err != nil {
		return profileQueryErrorResult(errors.BadRequest("invalid filter: %s", err))
	}
	at := ""
	if args.At != nil {
		at = *args.At
		if at == "" {
			return profileQueryErrorResult(errors.BadRequest("at cannot be empty"))
		}
	}

	result, err := ws.ExplainProfileFilter(ctx, args.KPID, filter, at)
	if err != nil {
		if _, ok := errors.AsType[*errors.NotFoundError](err); ok {
			return mcp.NewToolResultError("ProfileNotFound: profile does not exist"), nil
		}
		return profileQueryErrorResult(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, errors.New("cmd/internal/mcp: cannot encode profile explanation")
	}
	if len(encoded) > maxProfileQueryResultBytes {
		return nil, errors.New("cmd/internal/mcp: profile explanation exceeds result limit")
	}

	return mcp.NewToolResultText(string(encoded)), nil
}
