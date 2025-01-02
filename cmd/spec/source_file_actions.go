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

	nameParameter := types.Property{
		Name:           "name",
		Type:           types.Text().WithCharLen(60),
		CreateRequired: true,
		Placeholder:    `"Newsletter Subscribers"`,
		Description:    "The action's name.",
	}

	filterParameter := types.Property{
		Name:        "filter",
		Type:        filterType,
		Nullable:    true,
		Placeholder: `{ "logical": "and", "conditions": [ { "property": "country", "operator": "is", "values": [ "US" ] } ] }`,
		Description: "The filter applied to the users in the file. If it's not null, only the users that match the filter will be included.\n\n" +
			"See the [filters documentation](/filters) for more details.",
	}

	formatParameter := types.Property{
		Name:           "format",
		Type:           types.Text().WithValues("CVS", "Excel", "Parquet", "JSON"),
		CreateRequired: true,
		Placeholder:    `"Excel"`,
		Description:    "The file format. It correspond to the name of a file connector.",
	}

	pathParameter := types.Property{
		Name:           "path",
		Type:           types.Text().WithCharLen(1024),
		CreateRequired: true,
		Placeholder:    `"subscribers.xlsx"`,
		Description:    "The file path relative to the root path defined in the action's connection. Refer to the file storage connector documentation for details on the specific format.",
	}

	sheetParameter := types.Property{
		Name:        "sheet",
		Type:        types.Text(),
		Placeholder: `"Sheet1"`,
		Description: "The sheet name. It can only be used with the Excel format, where it is required.\n\n" +
			"When provided, it must have a length between 1 and 31 characters, not start or end with a single quote `'`, and cannot contain any of the following characters: `*`, `/`, `:`, `?`, `[`, `\\`, and `]`.",
	}

	compressionParameter := types.Property{
		Name:        "compression",
		Type:        types.Text().WithValues("", "Zip", "Gzip", "Snappy"),
		Placeholder: `"Gzip"`,
		Description: "The method used to compress the file, if applicable.\n\n" +
			"**Note:** An Excel file is inherently compressed, so no compression method needs to be specified unless the file has been further compressed.",
	}

	formatSettingsParameter := types.Property{
		Name:        "formatSettings",
		Type:        types.Parameter("Settings"),
		Nullable:    true,
		Placeholder: `{ "HasColumnNames": true }`,
		Description: "The settings for the file format. Refer to the documentation for the [connector](/connectors/) related to the file format to understand the available settings and their corresponding values.\n\n" +
			"If the file format does not require any settings, the `formatSettings` field may be omitted or set to null.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "source-file-actions",
		Name:        "Source file actions",
		Description: "A source file action is an action that extracts user data from a file and loads it into the workspace data warehouse for further processing and analysis.",
		Endpoints: []*Endpoint{
			{
				Name:        "Create a source file action",
				Description: "Create a source file action.",
				Method:      POST,
				URL:         "/v0/actions",
				Parameters: []types.Property{
					nameParameter,
					{
						Name:           "connection",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "230527183",
						Description:    "The connection for which the action should be created. It should be a **source file storage** connection.",
					},
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicate if the action is enabled once created.",
					},
					filterParameter,
					{
						Name:           "inSchema",
						Type:           types.Parameter("schema"),
						CreateRequired: true,
						Placeholder:    `{...}`,
					},
					{
						Name:           "outSchema",
						Type:           types.Parameter("schema"),
						CreateRequired: true,
						Placeholder:    `{...}`,
					},
					{
						Name:           "transformation",
						Type:           types.Parameter("transformation"),
						CreateRequired: true,
						Placeholder:    `{...}`,
					},
					formatParameter,
					pathParameter,
					sheetParameter,
					compressionParameter,
					formatSettingsParameter,
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Placeholder: "285017124",
							Description: "The ID of the action.",
						},
					},
				},
				Errors: []Error{
					{422, ConnectionNotExist, "connection does not exist"},
					{422, FormatNotExist, "format does not exist"},
					{422, InvalidSettings, "format settings are not valid"},
					{422, UnsupportedLanguage, "transformation language is not supported"},
				},
			},
			{
				Name:        "Update a source file action",
				Description: "Update a source file action.",
				Method:      PUT,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						UpdateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source file action.",
					},
					nameParameter,
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicates if the action is enable.",
					},
					{
						Name:           "inSchema",
						Type:           types.Parameter("schema"),
						CreateRequired: true,
						Placeholder:    `{...}`,
					},
					{
						Name:           "outSchema",
						Type:           types.Parameter("schema"),
						CreateRequired: true,
						Placeholder:    `{...}`,
					},
					filterParameter,
					{
						Name:           "transformation",
						Type:           types.Parameter("transformation"),
						CreateRequired: true,
						Placeholder:    `{...}`,
					},
					pathParameter,
					sheetParameter,
					compressionParameter,
				},
				Errors: []Error{
					{404, NotFound, "action does not exist"},
					{422, FormatNotExist, "format does not exist"},
					{422, InvalidSettings, "format settings are not valid"},
					{422, UnsupportedLanguage, "transformation language is not supported"},
				},
			},
			{
				Name:        "Set the schedule period",
				Description: "Sets the frequency at which a source file action imports users. Both the action and its connection must be active for the import to run as scheduled.",
				Method:      PUT,
				URL:         "/v0/actions/:id/schedule",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						UpdateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source file action on users.",
					},
					{
						Name:           "schedulePeriod",
						Type:           types.Int(32),
						UpdateRequired: true,
						Placeholder:    "60",
						Description: "The schedule period in minutes.\n\n" +
							"Possible values: `5`, `15`, `30`, `60`, `120`, `180`, `360`, `480`, `720`, `1440`.",
					},
				},
				Errors: []Error{
					{404, NotFound, "action does not exist"},
				},
			},
			{
				Name:        "Get a source file action",
				Description: "Get a source file action.",
				Method:      GET,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						Placeholder:    "705981339",
						Description:    "The ID of the source file action.",
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
						nameParameter,
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
							Name:        "scheduleStart",
							Type:        types.Int(32),
							Placeholder: "10",
							Description: "The schedule start",
						},
						{
							Name:        "schedulePeriod",
							Type:        types.Int(32),
							Placeholder: "60",
							Description: "The schedule period. It can be 5, 15, 30, 60, 120, 180, 360, 480, 720, and 1440.",
						},
						{
							Name:           "inSchema",
							Type:           types.Parameter("schema"),
							CreateRequired: true,
							Placeholder:    `{...}`,
						},
						{
							Name:           "outSchema",
							Type:           types.Parameter("schema"),
							CreateRequired: true,
							Placeholder:    `{...}`,
						},
						filterParameter,
						{
							Name:           "transformation",
							Type:           types.Parameter("transformation"),
							CreateRequired: true,
							Placeholder:    `{...}`,
						},
						formatParameter,
						pathParameter,
						sheetParameter,
						compressionParameter,
					},
				},
				Errors: []Error{
					{404, NotFound, "action does not exist"},
				},
			},
			{
				Name: "Import users from a file",
				Description: "Starts a source file action execution to import users from the action's file into the workspace’s data warehouse, applying the action's transformation.\n\n" +
					"It returns immediately without waiting for the execution to complete. To track the progress, call the [`/executions/:id`](/api/executions) endpoint using the returned execution ID.",
				Method: POST,
				URL:    "/v0/actions/:id/exec",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						UpdateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source file action.",
					},
					{
						Name:           "reload",
						Type:           types.Boolean(),
						UpdateRequired: false,
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
					{404, NotFound, "action does not exist"},
					{422, ConnectionDisabled, "connection is disabled"},
					{422, ExecutionInProgress, "action is already in progress"},
					{422, InspectionMode, "data warehouse is in inspection mode"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
				},
			},

			{
				Name:        "Delete a source file action",
				Description: "Delete a source file action.",
				Method:      DELETE,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						UpdateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source file action.",
					},
				},
				Errors: []Error{
					{404, NotFound, "action does not exist"},
				},
			},
		},
	})
}
