// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package spec

import (
	"github.com/meergo/meergo/core/types"
)

func init() {

	modeParameter := types.Property{
		Name:           "mode",
		Type:           types.Text().WithValues("Normal", "Inspection", "Maintenance"),
		Prefilled:      `"Normal"`,
		CreateRequired: true,
		Description:    "The mode of the data warehouse.",
	}
	cancelIncompatibleOperationsParameter := types.Property{
		Name:        "cancelIncompatibleOperations",
		Type:        types.Boolean(),
		Description: "Indicates whether operations currently running on the warehouse that are incompatible with the passed `mode` must be canceled.",
	}
	settingsParameter := types.Property{
		Name:           "settings",
		Type:           types.Parameter("Warehouse"),
		Prefilled:      "{...}",
		CreateRequired: true,
		Description:    "The settings of the data warehouse.",
	}
	getMCPSettingsParameter := types.Property{
		Name:      "mcpSettings",
		Type:      types.Parameter("Warehouse"),
		Prefilled: "{...}",
		Nullable:  true,
		Description: "The read-only settings of the data warehouse that are used for accessing it from the MCP (Model Context Protocol) server.\n\n" +
			"When `null`, it means that the MCP server settings aren't configured, so the MCP tools cannot be used for this workspace.",
	}
	postMCPSettingsParameter := types.Property{
		Name:      "mcpSettings",
		Type:      types.Parameter("Warehouse"),
		Prefilled: "{...}",
		Nullable:  true,
		Description: "The settings of the data warehouse that are used for accessing it from the MCP (Model Context Protocol) server.\n\n" +
			"When provided, these settings must refer to a read-only access to the data warehouse; this is a security requirement to prevent the MCP client from performing destructive operations on the warehouse data by mistake.\n\n" +
			"If `null` is passed, the MCP server settings aren't configured, preventing the use of MCP tools for this workspace.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "warehouse",
		Name:        "Warehouse",
		Description: "A warehouse, which is associated with a [workspace](workspaces), stores [user](users) and [event](events) data. This data can be accessed or queried for export to external destinations.",
		Endpoints: []*Endpoint{
			{
				Name:        "Get warehouse",
				Description: "Returns the name and the settings of the current workspace's warehouse.",
				Method:      GET,
				URL:         "/v1/warehouse",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "name",
							Type:        types.Text().WithValues("Snowflake", "PostgreSQL"),
							Prefilled:   `"Snowflake"`,
							Description: "The name of the data warehouse.",
						},
						settingsParameter,
						getMCPSettingsParameter,
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
			{
				Name:        "Update warehouse",
				Description: "Updates the warehouse of the current workspace.",
				Method:      PUT,
				URL:         "/v1/warehouse",
				Parameters: []types.Property{
					settingsParameter,
					postMCPSettingsParameter,
					modeParameter,
					cancelIncompatibleOperationsParameter,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, DifferentWarehouse, "data warehouse is a different data warehouse"},
					{422, InvalidWarehouseSettings, "data warehouse settings are not valid"},
					{422, NotReadOnlyMCPSettings, "warehouse MCP settings are not read-only"},
				},
			},
			{
				Name: "Test warehouse update",
				Description: "Tests the update of the warehouse of the current workspace.\n\n" +
					"If the settings are incorrect or the warehouse can't be accessed with the given settings, an error will be returned. " +
					"If no error occurs, the settings are valid.",
				Method: PUT,
				URL:    "/v1/warehouse/test",
				Parameters: []types.Property{
					settingsParameter,
					postMCPSettingsParameter,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, DifferentWarehouse, "data warehouse is a different data warehouse"},
					{422, InvalidWarehouseSettings, "data warehouse settings are not valid"},
					{422, NotReadOnlyMCPSettings, "warehouse MCP settings are not read-only"},
				},
			},
			{
				Name:        "Update warehouse mode",
				Description: "Updates the mode of the current workspace's data warehouse.",
				Method:      PUT,
				URL:         "/v1/warehouse/mode",
				Parameters: []types.Property{
					modeParameter,
					cancelIncompatibleOperationsParameter,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
			{
				Name: "Repair warehouse",
				Description: "Repairs the current workspace's warehouse.\n\n" +
					"This endpoint can be called when neither identity resolution nor alter schema operations are running on the data warehouse.",
				Method: POST,
				URL:    "/v1/warehouse/repair",
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
			{
				Name:        "List warehouse types",
				Description: "Returns a list of warehouse types that can be used for a workspace warehouse.",
				Method:      GET,
				URL:         "/v1/warehouse/types",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name: "types",
							Type: types.Array(types.Object([]types.Property{
								{
									Name:        "name",
									Type:        types.Text(),
									Prefilled:   `"Snowflake"`,
									Description: "The name of the warehouse type",
								},
							})),
							Prefilled: "...",
						},
					},
				},
			},
		},
	})

}
