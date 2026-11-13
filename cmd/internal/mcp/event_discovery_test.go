// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mcp

import (
	"reflect"
	"strings"
	"testing"
)

// TestEventDiscoveryToolMetadata checks registration and the selector schema.
func TestEventDiscoveryToolMetadata(t *testing.T) {
	found := map[string]bool{}
	for _, tool := range tools {
		if tool.Tool.Name != "list-observed-event-sources" && tool.Tool.Name != "inspect-event-fields" {
			continue
		}
		found[tool.Tool.Name] = true
		annotations := tool.Tool.Annotations
		if annotations.ReadOnlyHint == nil || !*annotations.ReadOnlyHint ||
			annotations.DestructiveHint == nil || *annotations.DestructiveHint ||
			annotations.OpenWorldHint == nil || *annotations.OpenWorldHint {
			t.Fatalf("expected bounded read-only discovery metadata, got %+v", annotations)
		}
		if !strings.Contains(serverInstructions, tool.Tool.Name) {
			t.Fatalf("expected instructions about %s, got absent", tool.Tool.Name)
		}
		if tool.Tool.Name == "inspect-event-fields" {
			source, ok := tool.Tool.InputSchema.Properties["source"].(map[string]any)
			if !ok || !reflect.DeepEqual(source["required"], []string{"type", "recordedName"}) ||
				source["additionalProperties"] != false {
				t.Fatalf("expected a closed source with explicit nullable name, got %+v", source)
			}
		}
	}
	if len(found) != 2 {
		t.Fatalf("expected two registered discovery tools, got %v", found)
	}
}
