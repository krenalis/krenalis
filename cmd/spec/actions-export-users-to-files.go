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
		Description:    "The file path relative to the root path defined in the file storage. Refer to the file storage documentation for details on the specific format.",
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
		Description: "The format used to compress the file. If not provided or empty, the file will remain uncompressed.",
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
		ID:   "actions-export-users-to-files",
		Name: "Export users to files",
		Description: "This type of action exports user data from the workspace’s data warehouse to a newly created file. " +
			"It operates on a destination file store connection.",
		Endpoints: []*Endpoint{
			{
				Name:        "Create action",
				Description: "Create a destination action that exports users to a file.",
				Method:      POST,
				URL:         "/v0/actions",
				Parameters: []types.Property{
					nameParameter,
					{
						Name:           "connection",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "230527183",
						Description:    "The ID of the connection to which the file will be written. It must be a destination file storage.",
					},
					{
						Name:           "target",
						Type:           types.Text().WithValues("Users"),
						CreateRequired: true,
						Placeholder:    `"Users"`,
						Description:    "The entity on which the action operates, which must be `\"Users\"` in order to create an action that exports users.",
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
				Description: "Update a destination action that exports users to a file.",
				Method:      PUT,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the destination file action.",
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
				Description: "Get a destination action that exports users to a file.",
				Method:      GET,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the destination file action.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Placeholder: "705981339",
							Description: "The ID of the destination file action.",
						},
						nameParameter,
						{
							Name:        "connector",
							Type:        types.Text(),
							Placeholder: `"S3"`,
							Description: "The name of the connection's connector.",
						},
						{
							Name:        "connectorType",
							Type:        types.Text().WithValues("App", "Database", "FileStorage", "Mobile", "Server", "Website"),
							Placeholder: `"FileStorage"`,
							Description: "The type of the connection's connector. It is always `\"FileStorage\"` when the action exports users to a file.",
						},
						{
							Name:        "connection",
							Type:        types.Int(32),
							Placeholder: "1371036433",
							Description: "The ID of the connection to which the file is written. It is a destination file storage.",
						},
						{
							Name:        "connectionRole",
							Type:        types.Text().WithValues("Source", "Destination"),
							Placeholder: `"Destination"`,
							Description: "The role of the action's connection. It is always `\"Destination\"` when the action exports users to a file.",
						},
						{
							Name:        "target",
							Type:        types.Text().WithValues("Users", "Events"),
							Placeholder: `"Users"`,
							Description: "The entity on which the action operates. It is always `\"Users\"` for an action that exports users.",
						},
						{
							Name:        "enabled",
							Type:        types.Boolean(),
							Placeholder: "true",
							Description: "Indicates if the action is enabled.",
						},
						formatParameter,
						pathParameter,
						{
							Name:           "sheet",
							Type:           types.Text(),
							Placeholder:    `"Sheet1"`,
							UpdateRequired: true,
							Description:    "The name of the sheet. It is empty if the format is not Excel.",
						},
						{
							Name:        "compression",
							Type:        types.Text().WithValues("", "Zip", "Gzip", "Snappy"),
							Placeholder: `"Gzip"`,
							Description: "The format used to compress the file. If empty, the file will remain uncompressed.",
						},
						filterParameter,
						{
							Name:           "inSchema",
							Type:           types.Parameter("schema"),
							CreateRequired: true,
							Placeholder:    `{...}`,
						},
						{
							Name:        "running",
							Type:        types.Boolean(),
							Placeholder: "false",
							Description: "Indicates if the action is running.",
						},
						scheduleStartParameter,
						exportSchedulePeriodParameter,
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
