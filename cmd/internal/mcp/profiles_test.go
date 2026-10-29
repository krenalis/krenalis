// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mcp

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestProfileQueryErrorResult(t *testing.T) {
	for _, test := range []struct {
		name    string
		err     error
		message string
	}{
		{"invalid", errors.BadRequest("invalid Count"), "InvalidArguments: invalid Count"},
		{"capability", errors.Unprocessable(core.FilterCapabilityNotImplemented, "not implemented"), "FilterCapabilityNotImplemented: not implemented"},
		{"backend", errors.Unprocessable(core.UnifiedFilteringUnavailable, "unavailable"), "UnifiedFilteringUnavailable: unavailable"},
		{"size", errors.Unprocessable(core.ProfileResultTooLarge, "reduce limit"), "ProfileResultTooLarge: reduce limit"},
		{"unavailable", errors.Unavailable("private warehouse detail"), ""},
		{"internal", errors.New("private warehouse detail"), ""},
		{"canceled", context.Canceled, ""},
		{"deadline", context.DeadlineExceeded, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := profileQueryErrorResult(test.err)
			if err != nil {
				if test.message != "" || result != nil || err.Error() != "cmd/internal/mcp: profile query failed" {
					t.Fatalf("expected opaque failure without result, got %+v, %v", result, err)
				}
				return
			}
			if test.message == "" || result == nil || !result.IsError || len(result.Content) != 1 {
				t.Fatalf("expected actionable tool error, got %+v, %v", result, err)
			}
			if text := mcp.GetTextFromContent(result.Content[0]); text != test.message {
				t.Fatalf("expected %q, got %q", test.message, text)
			}
		})
	}
}

func TestProfileQueryErrorResultBounded(t *testing.T) {
	result, err := profileQueryErrorResult(errors.BadRequest("%s", strings.Repeat("é", 5000)))
	if err != nil {
		t.Fatalf("expected bounded actionable error, got %v", err)
	}
	text := mcp.GetTextFromContent(result.Content[0])
	if len(text) > 2051 || !utf8.ValidString(text) || !strings.HasSuffix(text, "...") {
		t.Fatalf("expected at most 2051 valid UTF-8 bytes with ellipsis, got %d bytes", len(text))
	}
}

func TestQueryProfilesToolMetadata(t *testing.T) {

	mcpServer := NewMCPServer(nil)
	defer mcpServer.Close(t.Context())
	testServer := httptest.NewServer(mcpServer)
	defer testServer.Close()
	client, err := initMCPClient(t, testServer.Client(), testServer.URL)
	if err != nil {
		t.Fatalf("expected MCP client, got %v", err)
	}
	response, err := client.jsonRPCRequest("tools/list", nil)
	if err != nil {
		t.Fatalf("expected tool list, got %v", err)
	}
	defer response.Body.Close()
	var body struct {
		Result struct {
			Tools []mcp.Tool `json:"tools"`
		} `json:"result"`
	}
	err = json.NewDecoder(response.Body).Decode(&body)
	if err != nil {
		t.Fatalf("expected decoded tools, got %v", err)
	}

	found := map[string]bool{}
	for _, tool := range body.Result.Tools {
		found[tool.Name] = true
		if tool.Name != "query-profiles" {
			continue
		}
		filter, ok := tool.InputSchema.Properties["filter"].(map[string]any)
		if !ok || filter["type"] != "string" {
			t.Fatalf("expected lossless JSON text filter input, got %+v", filter)
		}
		if tool.Annotations.ReadOnlyHint == nil || !*tool.Annotations.ReadOnlyHint ||
			tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint {
			t.Fatalf("expected read-only non-destructive tool, got %+v", tool.Annotations)
		}
		for _, text := range []string{"main Krenalis connection", "table/column permissions", "main virtual warehouse"} {
			if !strings.Contains(tool.Description, text) {
				t.Fatalf("expected access description to contain %q, got %q", text, tool.Description)
			}
		}
	}
	for _, name := range []string{"query-profiles", "profile-filter-doc", "query-data-warehouse"} {
		if !found[name] {
			t.Fatalf("expected tool %s, got absent", name)
		}
	}
	for _, text := range []string{"query-profiles", "profile-filter-doc", "Never use SQL to emulate"} {
		if !strings.Contains(serverInstructions, text) {
			t.Fatalf("expected server instructions to contain %q, got %q", text, serverInstructions)
		}
	}

}
