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

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "source-database-actions",
		Name:        "Source database actions",
		Description: "A source database action is an action that extracts user data from a database and loads it into the workspace data warehouse for further processing and analysis.",
		Endpoints: []*Endpoint{
			{
				Name:        "Create a source database action",
				Description: "Create a source database action.",
				Method:      POST,
				URL:         "/v0/actions",
				Parameters: []types.Property{
					{
						Name:           "name",
						Type:           types.Text().WithCharLen(60),
						CreateRequired: true,
						Placeholder:    `"PostgreSQL"`,
						Description:    "The action's name.",
					},
					{
						Name:           "connection",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "230527183",
						Description:    "The connection for which the action should be created. It should be a source database connection.",
					},
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicate if the action is enabled once created.",
					},
					{
						Name:           "query",
						Type:           types.Text(),
						CreateRequired: true,
						Placeholder:    `"SELECT name, email, updated_at FROM customers LIMIT ${limit}"`,
					},
					{
						Name:           "identityProperty",
						Type:           types.Text(),
						CreateRequired: true,
						Placeholder:    `"email"`,
					},
					{
						Name:        "lastChangeTimeProperty",
						Type:        types.Text(),
						Placeholder: `"updated_at"`,
					},
					{
						Name:        "lastChangeTimeFormat",
						Type:        types.Text(),
						Placeholder: `"ISO8601"`,
					},
					{
						Name:        "inSchema",
						Type:        types.Parameter("schema"),
						Placeholder: `{...}`,
					},
					{
						Name:        "outSchema",
						Type:        types.Parameter("schema"),
						Placeholder: `{...}`,
					},
					{
						Name:        "transformation",
						Type:        types.Parameter("transformation"),
						Placeholder: `{...}`,
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Placeholder: "705981339",
							Description: "The ID of the action.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, ConnectionNotExist, "connection does not exist"},
					{422, ConnectorNotExist, "connector does not exist"},
					{422, UnsupportedLanguage, "transformation language is not supported"},
				},
			},
			{
				Name:        "Update a source database action",
				Description: "Update a source database action.",
				Method:      PUT,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source database action.",
					},
					{
						Name:           "name",
						Type:           types.Text().WithCharLen(60),
						CreateRequired: true,
						Placeholder:    `"HubSpot"`,
						Description:    "The action's name.",
					},
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicates if the action is enable.",
					},
					{
						Name:           "query",
						Type:           types.Text(),
						CreateRequired: true,
						Placeholder:    `"SELECT name, email, updated_at FROM customers"`,
					},
					{
						Name:           "identityProperty",
						Type:           types.Text(),
						CreateRequired: true,
						Placeholder:    `"email"`,
					},
					{
						Name:        "lastChangeTimeProperty",
						Type:        types.Text(),
						Placeholder: `"updated_at"`,
					},
					{
						Name:        "lastChangeTimeFormat",
						Type:        types.Text(),
						Placeholder: `"ISO8601"`,
					},
					{
						Name:        "inSchema",
						Type:        types.Parameter("schema"),
						Placeholder: `{...}`,
					},
					{
						Name:        "outSchema",
						Type:        types.Parameter("schema"),
						Placeholder: `{...}`,
					},
					{
						Name:        "transformation",
						Type:        types.Parameter("transformation"),
						Placeholder: `{...}`,
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
					{422, UnsupportedLanguage, "transformation language is not supported"},
				},
			},
			{
				Name:        "Set the schedule period",
				Description: "Sets the frequency at which a source database action imports users. Both the action and its connection must be active for the import to run as scheduled.",
				Method:      PUT,
				URL:         "/v0/actions/:id/schedule",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source database action on users.",
					},
					{
						Name:           "schedulePeriod",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "60",
						Description: "The schedule period in minutes.\n\n" +
							"Possible values: `5`, `15`, `30`, `60`, `120`, `180`, `360`, `480`, `720`, `1440`.",
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
				},
			},
			{
				Name:        "Get a source database action",
				Description: "Get a source database action.",
				Method:      GET,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						Placeholder:    "705981339",
						Description:    "The ID of the source database action.",
						CreateRequired: true,
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Placeholder: "705981339",
							Description: "The ID of the source action.",
						},
						{
							Name:        "connection",
							Type:        types.Int(32),
							Placeholder: "1371036433",
							Description: "The ID of the action's connection.",
						},
						{
							Name:        "name",
							Type:        types.Text().WithCharLen(60),
							Placeholder: `"HubSpot"`,
							Description: "The action's name.",
						},
						{
							Name:        "enabled",
							Type:        types.Boolean(),
							Placeholder: "true",
							Description: "Indicates if the action is enable.",
						},
						{
							Name:        "running",
							Type:        types.Boolean(),
							Placeholder: "false",
							Description: "Indicates if the action is running.",
						},
						{
							Name:        "query",
							Type:        types.Text(),
							Placeholder: `"SELECT name, email, updated_at FROM customers"`,
						},
						{
							Name:        "identityProperty",
							Type:        types.Text(),
							Placeholder: `"email"`,
						},
						{
							Name:        "lastChangeTimeProperty",
							Type:        types.Text(),
							Placeholder: `"updated_at"`,
						},
						{
							Name:        "lastChangeTimeFormat",
							Type:        types.Text(),
							Placeholder: `"ISO8601"`,
						},
						{
							Name:        "inSchema",
							Type:        types.Parameter("schema"),
							Placeholder: `{...}`,
						},
						{
							Name:        "outSchema",
							Type:        types.Parameter("schema"),
							Placeholder: `{...}`,
						},
						{
							Name:        "transformation",
							Type:        types.Parameter("transformation"),
							Placeholder: `{...}`,
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
				},
			},
			{
				Name: "Import users from a database",
				Description: "Starts a source database action execution to import users from the action's database into the workspace’s data warehouse, applying the action's transformation.\n\n" +
					"It returns immediately without waiting for the execution to complete. To track the progress, call the [`/executions/:id`](/api/executions) endpoint using the returned execution ID.",
				Method: POST,
				URL:    "/v0/actions/:id/exec",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source database action.",
					},
					{
						Name:           "reload",
						Type:           types.Boolean(),
						CreateRequired: false,
						Placeholder:    "false",
						Description: " Indicates whether the users should be re-imported from scratch. " +
							"If set to false or omitted, only new users and those modified since the last import are processed.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Placeholder: "609461413",
							Description: "The ID of the started execution.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
					{422, ConnectionDisabled, "connection is disabled"},
					{422, ExecutionInProgress, "action is already in progress"},
					{422, InspectionMode, "data warehouse is in inspection mode"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
				},
			},

			{
				Name:        "Delete a source database action",
				Description: "Delete a source database action.",
				Method:      DELETE,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source database action.",
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
				},
			},
		},
	})
}
