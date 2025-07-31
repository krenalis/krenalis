//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package spec

import (
	"github.com/meergo/meergo/types"
)

func init() {

	idParameter := types.Property{
		Name:           "id",
		Type:           types.Int(32),
		CreateRequired: true,
		Placeholder:    "1371036433",
		Description:    "The ID of the workspace.",
	}
	nameParameter := types.Property{
		Name:           "name",
		Type:           types.Text().WithCharLen(100),
		CreateRequired: true,
		Placeholder:    `"Customers"`,
		Description:    "The workspace's name.",
	}
	userSchemaParameter := types.Property{
		Name:           "userSchema",
		Type:           types.Parameter("schema"),
		CreateRequired: true,
		Description:    "The user schema, defining the structure and properties of a user in the customer model of the workspace.",
	}
	userPrimarySources := types.Property{
		Name:        "userPrimarySources",
		Type:        types.Map(types.Int(32)),
		Placeholder: "{...}",
		Description: "The primary source for each user schema property that has one, where the key is the property name and the value is the connection identifier.\n\n" +
			"This source defines where the definitive value for the property is read from, preventing other sources from overwriting it once it is set.",
	}
	resolveIdentitiesOnBatchImport := types.Property{
		Name:        "resolveIdentitiesOnBatchImport",
		Type:        types.Boolean(),
		Placeholder: "false",
		Description: "Indicates whether identity resolution should be performed immediately after a batch import.\n\n" +
			"If set to true, Meergo will automatically resolve identities as soon as the import is finished.\n" +
			"If set to false, identity resolution will not happen automatically and will need to be triggered separately.",
	}
	identifiers := types.Property{
		Name:        "identifiers",
		Type:        types.Array(types.Text()),
		Placeholder: `[ "customerId", "email" ]`,
		Description: "The user schema properties used to determine if two users are the same. The identity resolution checks these identities in order.\n" +
			"Two users are considered the same if they have the same value for the first identity, or if neither user has a value for the first identity, the identity resolution moves to the next one, and so on.\n\n" +
			"If no identifiers are specified, once identity resolution is performed, all users will be considered different from each other.",
	}
	warehouseTypeParameter := types.Property{
		Name:        "type",
		Type:        types.Text().WithValues("Snowflake", "PostgreSQL"),
		Placeholder: `"Snowflake"`,
		Description: "The data warehouse type.",
	}
	warehouseModeParameter := types.Property{
		Name:        "mode",
		Type:        types.Text().WithValues("Normal", "Inspection", "Maintenance"),
		Placeholder: `"Normal"`,
		Description: "The mode of the data warehouse.",
	}
	warehouseSettingsParameter := types.Property{
		Name:        "settings",
		Type:        types.Parameter("Warehouse"),
		Placeholder: "{...}",
		Description: "The settings of the data warehouse.",
	}
	warehouseMCPSettingsParameter := types.Property{
		Name:        "mcpSettings",
		Type:        types.Parameter("Warehouse"),
		Placeholder: "{...}",
		Nullable:    true,
		Description: "The settings of the data warehouse that are used for accessing it from the MCP (Model Context Protocol) server.\n\n" +
			"When provided, these settings must refer to a read-only access to the data warehouse; this is a security requirement to prevent the MCP client from performing destructive operations on the warehouse data by mistake.\n\n" +
			"If `null` is passed, the MCP server settings aren't configured, preventing the use of MCP tools for this workspace.",
	}
	uiPreferencesParameter := types.Property{
		Name: "uiPreferences",
		Type: types.Object([]types.Property{
			{
				Name: "userProfile",
				Type: types.Object([]types.Property{
					{
						Name:        "image",
						Type:        types.Text().WithCharLen(100),
						Placeholder: `"customer_image"`,
						Description: "The path of the property in the user schema that represents the user's image, in base64 format.",
					},
					{
						Name:        "firstName",
						Type:        types.Text().WithCharLen(100),
						Placeholder: `"first_name"`,
						Description: "The path of the property in the user schema to display as the first name in the profile header.",
					},
					{
						Name:        "lastName",
						Type:        types.Text().WithCharLen(100),
						Placeholder: `"last_name"`,
						Description: "The path of the property in the user schema for the user's last name.",
					},
					{
						Name:        "extra",
						Type:        types.Text().WithCharLen(100),
						Placeholder: `"email"`,
						Description: "The path of the property in the user schema for additional user information. For example, the user schema property that represents the email.",
					},
				}),
			},
		}),
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "workspaces",
		Name:        "Workspaces",
		Description: "Workspaces enable Meergo to retrieve customer and event data from an external source location or send them to an external destination location.",
		Endpoints: []*Endpoint{
			{
				Name:        "Create workspace",
				Description: "Creates a new workspace.",
				Method:      POST,
				URL:         "/v1/workspaces",
				Parameters: []types.Property{
					nameParameter,
					userSchemaParameter,
					{
						Name: "warehouse",
						Type: types.Object([]types.Property{
							warehouseTypeParameter,
							warehouseModeParameter,
							warehouseSettingsParameter,
							warehouseMCPSettingsParameter,
						}),
						CreateRequired: true,
						Description: "The data warehouse of the workspace, serving as the central repository for the workspace's user data and collected events.\n\n" +
							"As part of the workspace provisioning process, the specified database is initialized with the schema, including all necessary tables, views, and stored procedures. " +
							"The target database must be empty prior to initialization.",
					},
					uiPreferencesParameter,
				},
				Response: &Response{
					Parameters: []types.Property{
						idParameter,
					},
				},
				Errors: []Error{
					{422, WarehouseTypeNotExist, "warehouse type does not exist"},
					{422, InvalidWarehouseSettings, "data warehouse settings are not valid"},
					{422, WarehouseNonInitializable, "data warehouse cannot be initialized"},
					{422, NotReadOnlyMCPSettings, "warehouse MCP settings are not read-only"},
				},
			},
			{
				Name:        "Test workspace creation",
				Description: "Checks the process of creating a new workspace. It performs all the validations required for workspace creation, including verifying that the data warehouse can be initialized, without making any changes to the data warehouse or actually creating the workspace.",
				Method:      POST,
				URL:         "/v1/workspaces/test",
				Parameters: []types.Property{
					nameParameter,
					userSchemaParameter,
					{
						Name: "warehouse",
						Type: types.Object([]types.Property{
							warehouseTypeParameter,
							warehouseModeParameter,
							warehouseSettingsParameter,
							warehouseMCPSettingsParameter,
						}),
						CreateRequired: true,
						Description: "The data warehouse of the workspace, serving as the central repository for the workspace's user data and collected events.\n\n" +
							"Since this method only tests the creation of a workspace, the data warehouse is not modified. " +
							"The target database must be empty to make the check succeed.",
					},
					uiPreferencesParameter,
				},
				Errors: []Error{
					{422, WarehouseTypeNotExist, "warehouse type does not exist"},
					{422, InvalidWarehouseSettings, "data warehouse settings are not valid"},
					{422, WarehouseNonInitializable, "data warehouse cannot be initialized"},
					{422, NotReadOnlyMCPSettings, "warehouse MCP settings are not read-only"},
				},
			},
			{
				Name:        "Update workspace",
				Description: "Updates the current workspace.",
				Method:      PUT,
				URL:         "/v1/workspaces/current",
				Parameters: []types.Property{
					nameParameter,
					uiPreferencesParameter,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
			{
				Name:        "List all workspaces",
				Description: "Lists all the workspaces of the organization.",
				Method:      GET,
				URL:         "/v1/workspaces",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name: "workspaces",
							Type: types.Array(types.Object([]types.Property{
								idParameter,
								nameParameter,
								userSchemaParameter,
								userPrimarySources,
								resolveIdentitiesOnBatchImport,
								identifiers,
								warehouseModeParameter,
								uiPreferencesParameter,
							})),
							Placeholder: "...",
							Description: "The workspaces of the organization.",
						},
					},
				},
			},
			{
				Name:        "Get workspace",
				Description: "Returns the current workspace.",
				Method:      GET,
				URL:         "/v1/workspaces/current",
				Response: &Response{
					Parameters: []types.Property{
						idParameter,
						nameParameter,
						userSchemaParameter,
						userPrimarySources,
						resolveIdentitiesOnBatchImport,
						identifiers,
						warehouseModeParameter,
						uiPreferencesParameter,
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
			{
				Name:        "Delete the workspace",
				Description: "Deletes the current workspace.",
				Method:      DELETE,
				URL:         "/v1/workspaces/current",
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
		},
	})

}
