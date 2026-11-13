// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mcp

import (
	"context"
	jsonv1 "encoding/json"
	"fmt"
	"time"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const eventDiscoveryNotice = "Observed metadata describes stored rows in the returned window, not business meaning" +
	" or a guaranteed schema. Read profile-filter-doc and event-schema before constructing filters." +
	" recordedName is not a Count source; empty or null does not mean all page/screen messages." +
	" Field types do not enable unsupported capabilities. Values are not returned."

var listObservedEventSourcesTool = server.ServerTool{
	Tool: mcp.NewTool("list-observed-event-sources",
		mcp.WithDescription("List distinct recorded track/page/screen names, including custom events outside the catalog."+
			" Uses the event timestamp in [start,end); omit both bounds for the last 30 elapsed days."+
			" Results are a literal ordered prefix, not ranked by recency or frequency. Check truncated."+
			" nameContains is a case-sensitive literal substring; %, _ and backslash are not wildcards."),
		mcp.WithString("type", mcp.Enum("", "track", "page", "screen"),
			mcp.Description("Omit or use an empty string for all three recorded types.")),
		mcp.WithString("nameContains", mcp.Description("Optional literal substring, at most 256 UTF-8 bytes.")),
		mcp.WithString("start", mcp.Description("RFC3339 start, inclusive; supply together with end.")),
		mcp.WithString("end", mcp.Description("RFC3339 end, exclusive and not future; maximum span 365 elapsed days.")),
		mcp.WithInteger("limit", mcp.Min(1), mcp.Max(500),
			mcp.Description("Omit for 100 sources; byte limits may shorten it.")),
		mcp.WithTitleAnnotation("Discover recorded event sources"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	),
	Handler: eventDiscovery,
}

var inspectEventFieldsTool = server.ServerTool{
	Tool: mcp.NewTool("inspect-event-fields",
		mcp.WithDescription("Inspect traits/properties paths, JSON kinds and presence in a bounded recent stored-row sample."+
			" Supply the exact type and recordedName, including explicit null or empty string when intended."+
			" Separate roots.uninspected from missing fields; fieldsTruncated concerns paths only."+
			" Stops at arrays and returns no payload values or identifiers. "+eventDiscoveryNotice),
		mcp.WithObject("source", mcp.Required(), mcp.AdditionalProperties(false),
			mcp.Properties(map[string]any{
				"type":         map[string]any{"type": "string", "enum": []string{"track", "page", "screen"}},
				"recordedName": map[string]any{"type": []string{"string", "null"}},
			}), func(schema map[string]any) { schema["required"] = []string{"type", "recordedName"} }),
		mcp.WithString("start", mcp.Description("RFC3339 start, inclusive; supply together with end.")),
		mcp.WithString("end", mcp.Description("RFC3339 end, exclusive and not future; maximum span 365 elapsed days.")),
		mcp.WithInteger("sampleLimit", mcp.Min(1), mcp.Max(1000),
			mcp.Description("Omit for 200 stored rows; byte limits can shorten the sample.")),
		mcp.WithTitleAnnotation("Inspect recorded event fields"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	),
	Handler: eventDiscovery,
}

func eventDiscovery(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {

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
	if len(raw) == 0 || len(raw) > 8*1024 || !json.Valid(raw) || !json.Value(raw).IsObject() {
		return mcp.NewToolResultError("InvalidArguments: arguments must be an object of at most 8 KiB"), nil
	}
	var args struct {
		Source       *core.ObservedEventSource `json:"source"`
		Type         string                    `json:"type"`
		NameContains string                    `json:"nameContains"`
		Start        *string                   `json:"start"`
		End          *string                   `json:"end"`
		Limit        *int                      `json:"limit"`
		SampleLimit  *int                      `json:"sampleLimit"`
	}
	for name, value := range json.Value(raw).Properties() {
		valid := false
		switch name {
		case "start", "end":
			valid = value.IsString()
		case "type", "nameContains":
			valid = req.Params.Name == "list-observed-event-sources" && value.IsString()
		case "limit":
			valid = req.Params.Name == "list-observed-event-sources" && value.IsNumber()
		case "sampleLimit":
			valid = req.Params.Name == "inspect-event-fields" && value.IsNumber()
		case "source":
			valid = req.Params.Name == "inspect-event-fields" && value.IsObject()
			if valid {
				hasType, hasName := false, false
				for member, field := range value.Properties() {
					switch member {
					case "type":
						hasType = field.IsString()
					case "recordedName":
						hasName = field.IsString() || field.IsNull()
					default:
						valid = false
					}
				}
				valid = valid && hasType && hasName
			}
		default:
			// Closed tool arguments reject every unknown member.
		}
		if !valid {
			return mcp.NewToolResultError("InvalidArguments: unexpected argument or argument type"), nil
		}
	}
	err = json.Unmarshal(raw, &args)
	if err != nil {
		return mcp.NewToolResultError("InvalidArguments: invalid event discovery arguments"), nil
	}
	var window *core.EventDiscoveryWindow
	if args.Start != nil || args.End != nil {
		if args.Start == nil || args.End == nil {
			return mcp.NewToolResultError("InvalidArguments: supply start and end together"), nil
		}
		start, err := time.Parse(time.RFC3339Nano, *args.Start)
		if err != nil {
			return mcp.NewToolResultError("InvalidArguments: start must be RFC3339"), nil
		}
		end, err := time.Parse(time.RFC3339Nano, *args.End)
		if err != nil {
			return mcp.NewToolResultError("InvalidArguments: end must be RFC3339"), nil
		}
		window = &core.EventDiscoveryWindow{Start: start, End: end}
	}

	result := map[string]any{"notice": eventDiscoveryNotice}
	switch req.Params.Name {
	case "list-observed-event-sources":
		var page core.EventSourceDiscoveryResult
		page, err = ws.ObservedEventSources(ctx, core.EventSourceDiscoveryQuery{
			Window: window, Type: args.Type, NameContains: args.NameContains, Limit: args.Limit})
		result["window"], result["sources"], result["limit"] = page.Window, page.Sources, page.Limit
		result["truncated"], result["omittedLongNames"] = page.Truncated, page.OmittedLongNames
	case "inspect-event-fields":
		if args.Source == nil {
			return mcp.NewToolResultError("InvalidArguments: source is required"), nil
		}
		var fields core.EventFieldDiscoveryResult
		fields, err = ws.ObservedEventFields(ctx, core.EventFieldDiscoveryQuery{
			Window: window, Source: *args.Source, SampleLimit: args.SampleLimit})
		result["source"], result["window"], result["sample"] = fields.Source, fields.Window, fields.Sample
		result["roots"], result["fields"] = fields.Roots, fields.Fields
	default:
		return nil, fmt.Errorf("cmd/internal/mcp: unexpected event discovery tool %.64q", req.Params.Name)
	}
	if err != nil {
		if _, ok := errors.AsType[*errors.BadRequestError](err); ok {
			return mcp.NewToolResultError("InvalidArguments: " + err.Error()), nil
		}
		if invalid, ok := errors.AsType[*errors.UnprocessableError](err); ok {
			return mcp.NewToolResultError(string(invalid.Code) + ": " + invalid.Error()), nil
		}
		return nil, errors.New("cmd/internal/mcp: event discovery failed")
	}
	encoded, err := jsonv1.Marshal(result)
	if err != nil {
		return nil, errors.New("cmd/internal/mcp: cannot encode event discovery")
	}
	if len(encoded) > 128*1024 {
		return nil, errors.New("cmd/internal/mcp: event discovery response exceeds its budget")
	}

	return mcp.NewToolResultStructured(result, string(encoded)), nil
}
