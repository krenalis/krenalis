//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package mcp

import (
	"strings"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/types"
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
	for path, p := range types.WalkAll(core.EventSchema()) {
		if p.Type.Kind() == types.ObjectKind {
			continue
		}
		column := core.EventColumnByPath(path)
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
