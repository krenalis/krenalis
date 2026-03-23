// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mcp

import (
	"strings"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/types"
)

// profileSchemaInfoForMCPClient takes the profile schema, a function that maps
// a types.Type to the corresponding type on the currently connected warehouse,
// and returns a data structure containing information about the profile schema
// that can be serialized to JSON and sent to the MCP client.
func profileSchemaInfoForMCPClient(profileSchema types.Type, columnTypeDescription func(types.Type) (string, error)) []any {
	var info []any
	for path, p := range profileSchema.Properties().WalkAll() {
		if p.Type.Kind() == types.ObjectKind {
			continue
		}
		colTypDescription, _ := columnTypeDescription(p.Type)
		info = append(info, map[string]any{
			"profileSchemaProperty": map[string]any{
				"path": path,
				"type": p.Type,
			},
			"profileViewColumn": map[string]any{
				"name":     strings.ReplaceAll(path, ".", "_"),
				"type":     colTypDescription,
				"nullable": true,
			},
		})
	}
	// Add information about the "_mpid" and "_updated_at".
	info = append(info, map[string]any{
		"profileViewColumn": map[string]any{
			"name":        "_mpid",
			"type":        "uuid",
			"nullable":    true,
			"description": "The MPID (Krenalis Profile ID) uniquely identifies an unified profile within Krenalis. It doesn't have a corresponding property in the profile schema. It's used to reference the 'events.mpid' column.",
		},
	})
	info = append(info, map[string]any{
		"profileViewColumn": map[string]any{
			"name":        "_updated_at",
			"type":        "timestamp without time zone",
			"nullable":    true,
			"description": "Timestamp of the profile's last update. It doesn't have a corresponding property in the profile schema.",
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
	for path, p := range core.EventSchema().Properties().WalkAll() {
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
		if path == "profile" {
			info["eventTableColumn"].(map[string]any)["description"] = "If present, indicates the profile (in the profiles view) associated with this event."
		}
		eventSchemaInfoForMCPClient = append(eventSchemaInfoForMCPClient, info)
	}
}
