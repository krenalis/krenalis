// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mcp

import (
	"strings"
	"testing"
)

func TestEventCatalogToolMetadata(t *testing.T) {
	found := map[string]bool{}
	for _, tool := range tools {
		if tool.Tool.Name != "list-event-definitions" && tool.Tool.Name != "get-event-definition" {
			continue
		}
		found[tool.Tool.Name] = true
		annotations := tool.Tool.Annotations
		if annotations.ReadOnlyHint == nil || !*annotations.ReadOnlyHint ||
			annotations.DestructiveHint == nil || *annotations.DestructiveHint ||
			annotations.IdempotentHint == nil || !*annotations.IdempotentHint ||
			annotations.OpenWorldHint == nil || *annotations.OpenWorldHint {
			t.Fatalf("expected read-only deterministic catalog metadata, got %+v", annotations)
		}
	}
	for _, name := range []string{"list-event-definitions", "get-event-definition"} {
		if !found[name] {
			t.Fatalf("expected registered tool %s, got absent", name)
		}
		if !strings.Contains(serverInstructions, name) {
			t.Fatalf("expected instructions about %s, got absent", name)
		}
	}
}
