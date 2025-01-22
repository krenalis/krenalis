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
		Description:    "The file path relative to the root path defined in the file storage connection. Refer to the file storage connector documentation for details on the specific format.",
	}

	sheetParameter := types.Property{
		Name:           "sheet",
		Type:           types.Text(),
		Placeholder:    `"Sheet1"`,
		UpdateRequired: true,
		Description: "The sheet name. It can only be used with the Excel format, where it is required.\n\n" +
			"When provided, it must have a length between 1 and 31 characters, not start or end with a single quote `'`, and cannot contain any of the following characters: `*`, `/`, `:`, `?`, `[`, `\\`, and `]`.",
	}

	compressionParameter := types.Property{
		Name:        "compression",
		Type:        types.Text().WithValues("", "Zip", "Gzip", "Snappy"),
		Placeholder: `"Gzip"`,
		Description: "The compression format of the file. It is an empty if the file is not compressed.\n\n" +
			"Note that an Excel file is inherently compressed, so no compression format needs to be specified unless the file has been further compressed.",
	}

	formatSettingsParameter := types.Property{
		Name:        "formatSettings",
		Type:        types.Parameter("Settings"),
		Nullable:    true,
		Placeholder: `{ "HasColumnNames": true }`,
		Description: "The settings for the file format. Refer to the documentation for the [file connector](/connectors/) related to the file format to understand the available settings and their corresponding values.\n\n" +
			"If the file format does not require any settings, the `formatSettings` field may be omitted or set to null.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "actions-import-users-from-files",
		Name: "Import users from files",
		Description: "This type of action imports user data from a file into the workspace's data warehouse. " +
			"It operates on a source file storage connection.",
		Endpoints: []*Endpoint{
			{
				Name:        "Create action",
				Description: "Create a source action that imports users from a file.",
				Method:      POST,
				URL:         "/v0/actions",
				Parameters: []types.Property{
					nameParameter,
					{
						Name:           "connection",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "230527183",
						Description:    "The ID of the connection from which to read the file. It must be a source file storage.",
					},
					{
						Name:           "target",
						Type:           types.Text().WithValues("Users"),
						CreateRequired: true,
						Placeholder:    `"Users"`,
						Description:    "The entity on which the action operates, which must be `\"Users\"` in order to create an action that imports users.",
					},
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicate if the action is enabled once created.",
					},
					formatParameter,
					pathParameter,
					sheetParameter,
					compressionParameter,
					formatSettingsParameter,
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
					{404, NotFound, "workspace does not exist"},
					{422, ConnectionNotExist, "connection does not exist"},
					{422, FormatNotExist, "format does not exist"},
					{422, InvalidSettings, "format settings are not valid"},
					{422, UnsupportedLanguage, "transformation language is not supported"},
				},
			},
			{
				Name:        "Update action",
				Description: "Update a source action that imports users from a file.",
				Method:      PUT,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source file action.",
					},
					nameParameter,
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicates if the action is enabled. Use the [Set status](/api/actions#set-status) endpoint to change only the action's status.",
					},
					pathParameter,
					sheetParameter,
					compressionParameter,
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
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
					{422, FormatNotExist, "format does not exist"},
					{422, InvalidSettings, "format settings are not valid"},
					{422, UnsupportedLanguage, "transformation language is not supported"},
				},
			},
			{
				Name:        "Get action",
				Description: "Get a source action that imports users from a file.",
				Method:      GET,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source file action.",
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
						nameParameter,
						{
							Name:        "connector",
							Type:        types.Text(),
							Placeholder: `"SFTP"`,
							Description: "The name of the connection's connector.",
						},
						{
							Name:        "connectorType",
							Type:        types.Text().WithValues("App", "Database", "FileStorage", "Mobile", "Server", "Website"),
							Placeholder: `"FileStorage"`,
							Description: "The type of the connection's connector. It is always `\"FileStorage\"` when the action imports users from a file.",
						},
						{
							Name:        "connection",
							Type:        types.Int(32),
							Placeholder: "1371036433",
							Description: "The ID of the connection from which the file is read. It is a source file storage.",
						},
						{
							Name:        "connectionRole",
							Type:        types.Text().WithValues("Source", "Destination"),
							Placeholder: `"Source"`,
							Description: "The role of the action's connection. It is always `\"Source\"` when the action imports users from a file.",
						},
						{
							Name:        "target",
							Type:        types.Text().WithValues("Users", "Events"),
							Placeholder: `"Users"`,
							Description: "The entity on which the action operates. It is always `\"Users\"` when the action imports users from a file.",
						},
						{
							Name:        "enabled",
							Type:        types.Boolean(),
							Placeholder: "true",
							Description: "Indicates if the action is enabled.",
						},
						{
							Name:        "running",
							Type:        types.Boolean(),
							Placeholder: "false",
							Description: "Indicates if the action is running.",
						},
						scheduleStartParameter,
						importSchedulePeriodParameter,
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
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
				},
			},
		},
	})
}
