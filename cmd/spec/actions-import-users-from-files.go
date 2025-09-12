//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package spec

import (
	"github.com/meergo/meergo/core/types"
)

func init() {

	nameParameter := types.Property{
		Name:           "name",
		Type:           types.Text().WithCharLen(60),
		CreateRequired: true,
		Prefilled:      `"Newsletter Subscribers"`,
		Description:    "The action's name.",
	}
	filterParameter := types.Property{
		Name:      "filter",
		Type:      filterType,
		Nullable:  true,
		Prefilled: `{ "logical": "and", "conditions": [ { "property": "country", "operator": "is", "values": [ "US" ] } ] }`,
		Description: "The filter applied to the users in the file. If it's not null, only the users that match the filter will be included.\n\n" +
			"See the [filters documentation](/filters) for more details.",
	}
	formatParameter := types.Property{
		Name:           "format",
		Type:           types.Text().WithValues("CSV", "Excel", "Parquet", "JSON"),
		CreateRequired: true,
		Prefilled:      `"Excel"`,
		Description:    "The file format. It corresponds to the name of a file connector.",
	}
	pathParameter := types.Property{
		Name:           "path",
		Type:           types.Text().WithCharLen(1024),
		CreateRequired: true,
		Prefilled:      `"subscribers.xlsx"`,
		Description:    "The file path relative to the root path defined in the file storage connection. Refer to the file storage connector documentation for details on the specific format.",
	}
	sheetParameter := types.Property{
		Name:           "sheet",
		Type:           types.Text(),
		Prefilled:      `"Sheet1"`,
		UpdateRequired: true,
		Description: "The sheet name. It can only be used with the Excel format, where it is required.\n\n" +
			"When provided, it must have a length between 1 and 31 characters, not start or end with a single quote `'`, and cannot contain any of the following characters: `*`, `/`, `:`, `?`, `[`, `\\`, and `]`.",
	}
	compressionParameter := types.Property{
		Name:      "compression",
		Type:      types.Text().WithValues("", "Zip", "Gzip", "Snappy"),
		Prefilled: `"Gzip"`,
		Description: "The compression format of the file. It is empty if the file is not compressed.\n\n" +
			"Note that an Excel file is inherently compressed, so no compression format needs to be specified unless the file has been further compressed.",
	}
	formatSettingsParameter := types.Property{
		Name:      "formatSettings",
		Type:      types.Parameter("Settings"),
		Nullable:  true,
		Prefilled: `{ "HasColumnNames": true }`,
		Description: "The settings for the file format. Refer to the documentation for the [file connector](/connectors/) related to the file format to understand the available settings and their corresponding values.\n\n" +
			"If the file format does not require any settings, the `formatSettings` field must be either omitted or set to null.",
	}
	identityColumnParameter := types.Property{
		Name:           "identityColumn",
		Type:           types.Text().WithCharLen(1024),
		CreateRequired: true,
		Prefilled:      `"email"`,
		Description: "The column that uniquely identifies each user in the file. It serves as the single, unique identifier for each user record, ensuring that each user can be distinctly referenced.\n\n" +
			"Only columns with types corresponding to the following Meergo types can be used as an identity: `int`, `uint`, `uuid`, `json`, and `text`.",
	}
	lastChangeTimeParameter := types.Property{
		Name:      "lastChangeTimeColumn",
		Type:      types.Text().WithCharLen(1024),
		Prefilled: `"updated_at"`,
		Description: "The column that stores the date when a user record was last updated. It tracks the most recent modification made to the user's data, helping to identify when changes occurred.\n\n" +
			"The value of this column is used for incremental imports, where only records that have been modified since the last import need to be processed.\n\n" +
			"Only columns with types corresponding to the following Meergo types can be used as the last change time: `datetime`, `date`, `json`, and `text`.",
	}
	lastChangeTimeFormatParameter := types.Property{
		Name:           "lastChangeTimeFormat",
		Type:           types.Text().WithCharLen(64),
		UpdateRequired: true,
		Prefilled:      `"ISO8601"`,
		Description: "The format of the value in the last change time column. It can be set to `\"ISO8601\"` if the column value follows the ISO 8601 format. If `format` is `\"Excel\"`, it can also be set to `\"Excel\"`. " +
			"Otherwise, it should follow a format accepted by the [Python strftime function](https://docs.python.org/3/library/datetime.html#strftime-strptime-behavior).\n\n" +
			"This field is only required if the `lastChangeTimeColumn` is provided, is not empty, and has a type `json` or `text`.",
	}
	incrementalParameter := types.Property{
		Name:      "incremental",
		Type:      types.Boolean(),
		Prefilled: `true`,
		Description: "Determines whether users are imported incrementally:\n" +
			"* `true`: are imported only users whose last change time is equal to or later than the last imported user's change time.\n" +
			"* `false`: all users are imported again, regardless of their last change time. `false` is the default value.\n\n" +
			"If set to `true`, a column for the last change time must be specified (i.e., `lastChangeTimeColumn` is not null).",
	}
	transformationParameter := types.Property{
		Name: "transformation",
		Type: types.Object([]types.Property{
			{
				Name:           "mapping",
				Type:           types.Map(types.Text()),
				Prefilled:      `{ "first_name": "firstName" }`,
				UpdateRequired: true,
				Nullable:       true,
				Description:    "The transformation mapping. A key represents a property path in the user schema, and its corresponding value is an expression. This expression can reference columns of the file.",
			},
			{
				Name: "function",
				Type: types.Object([]types.Property{
					{
						Name:           "source",
						Type:           types.Text().WithCharLen(50_000),
						Prefilled:      `"const transform = (user) => { ... }"`,
						CreateRequired: true,
						Description:    "The source code of the JavaScript or Python function.",
					},
					{
						Name:           "language",
						Type:           types.Text().WithValues("JavaScript", "Python"),
						Prefilled:      `"JavaScript"`,
						CreateRequired: true,
						Description:    "The language of the function.",
					},
					{
						Name:        "preserveJSON",
						Type:        types.Boolean(),
						Prefilled:   "false",
						Description: "Specifies whether JSON values are passed to and returned from the function as strings, keeping their original format without any encoding or decoding.",
					},
					{
						Name:           "inPaths",
						Type:           types.Array(types.Text()),
						Prefilled:      `[ "email", "firstName", "lastName" ]`,
						CreateRequired: true,
						Description:    "The paths of the properties that will be passed to the function. At least one path must be present.",
					},
					{
						Name:           "outPaths",
						Type:           types.Array(types.Text()),
						CreateRequired: true,
						Prefilled:      `[ "email_address", "first_name", "last_name" ]`,
						Description:    "The paths of the properties that may be returned by the function. At least one path must be present.",
					},
				}),
				UpdateRequired: true,
				Nullable:       true,
				Description:    "The transformation function. A JavaScript or Python function that given a user in the file, returns a user identity.",
			},
		}),
		Prefilled:      `...`,
		CreateRequired: true,
		UpdateRequired: true,
		Description: "The mapping or function responsible for transforming file users into user identities linked to the action. " +
			"Once the identity resolution process is complete, the user identities associated with all actions are merged into unified users.\n\n" +
			"One of either a mapping or a function must be provided, but not both. The one that is not provided can be either missing or set to null.",
	}
	inSchemaParameter := types.Property{
		Name:           "inSchema",
		Type:           types.Parameter("schema"),
		Prefilled:      `{...}`,
		CreateRequired: true,
		Description: "The schema for the properties used in the filter, the identity column, the last change time column, and the input properties for the transformation.\n\n" +
			"When importing users from files, this should be a subset of the file schema.",
	}
	outSchemaParameter := types.Property{
		Name:           "outSchema",
		Type:           types.Parameter("schema"),
		Prefilled:      `{...}`,
		CreateRequired: true,
		Description: "The schema for the output properties of the transformation.\n\n" +
			"When importing users from files, this should be a subset of the user schema.",
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
				URL:         "/v1/actions",
				Parameters: []types.Property{
					nameParameter,
					{
						Name:           "connection",
						Type:           types.Int(32),
						CreateRequired: true,
						Prefilled:      "230527183",
						Description:    "The ID of the connection from which to read the users. It must be a source file storage.",
					},
					{
						Name:           "target",
						Type:           types.Text().WithValues("User"),
						CreateRequired: true,
						Prefilled:      `"User"`,
						Description:    "The entity on which the action operates, which must be `\"User\"` in order to create an action that imports users.",
					},
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Prefilled:   "true",
						Description: "Indicates if the action is enabled once created.",
					},
					formatParameter,
					pathParameter,
					sheetParameter,
					compressionParameter,
					formatSettingsParameter,
					filterParameter,
					identityColumnParameter,
					lastChangeTimeParameter,
					lastChangeTimeFormatParameter,
					incrementalParameter,
					transformationParameter,
					inSchemaParameter,
					outSchemaParameter,
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Prefilled:   "285017124",
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
				URL:         "/v1/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Prefilled:      "705981339",
						Description:    "The ID of the source file action.",
					},
					nameParameter,
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Prefilled:   "true",
						Description: "Indicates if the action is enabled. Use the [Set status](actions#set-status) endpoint to change only the action's status.",
					},
					pathParameter,
					sheetParameter,
					compressionParameter,
					filterParameter,
					identityColumnParameter,
					lastChangeTimeParameter,
					lastChangeTimeFormatParameter,
					incrementalParameter,
					transformationParameter,
					inSchemaParameter,
					outSchemaParameter,
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
				URL:         "/v1/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Prefilled:      "705981339",
						Description:    "The ID of the source file action.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Prefilled:   "705981339",
							Description: "The ID of the source action.",
						},
						nameParameter,
						{
							Name:        "connector",
							Type:        types.Text(),
							Prefilled:   `"SFTP"`,
							Description: "The name of the connection's connector.",
						},
						{
							Name:        "connectorType",
							Type:        types.Text().WithValues("App", "Database", "FileStorage", "SDK"),
							Prefilled:   `"FileStorage"`,
							Description: "The type of the connection's connector. It is always `\"FileStorage\"` when the action imports users from a file.",
						},
						{
							Name:        "connection",
							Type:        types.Int(32),
							Prefilled:   "1371036433",
							Description: "The ID of the connection from which the file is read. It is a source file storage.",
						},
						{
							Name:        "connectionRole",
							Type:        types.Text().WithValues("Source", "Destination"),
							Prefilled:   `"Source"`,
							Description: "The role of the action's connection. It is always `\"Source\"` when the action imports users from a file.",
						},
						{
							Name:        "target",
							Type:        types.Text().WithValues("User", "Event"),
							Prefilled:   `"User"`,
							Description: "The entity on which the action operates. It is always `\"User\"` when the action imports users from a file.",
						},
						{
							Name:        "enabled",
							Type:        types.Boolean(),
							Prefilled:   "true",
							Description: "Indicates if the action is enabled.",
						},
						formatParameter,
						pathParameter,
						{
							Name:        "sheet",
							Type:        types.Text(),
							Nullable:    true,
							Prefilled:   `"Sheet1"`,
							Description: "The name of the sheet. It is empty if the format is not Excel.",
						},
						compressionParameter,
						{
							Name:        "identityColumn",
							Type:        types.Text(),
							Prefilled:   `"email"`,
							Description: "The column that uniquely identifies each user in the file.",
						},
						{
							Name:        "lastChangeTimeColumn",
							Type:        types.Text(),
							Nullable:    true,
							Prefilled:   `"updated_at"`,
							Description: "The column that stores the timestamp of the last update to a user record. It is null if no such column exists.",
						},
						{
							Name:      "lastChangeTimeFormat",
							Type:      types.Text(),
							Nullable:  true,
							Prefilled: `"ISO8601"`,
							Description: "The format of the value in the last change time column. It is null if no such column exists or if the corresponding Meergo type is `datetime` or `date`.\n\n" +
								"It is `\"ISO8601\"` if the column value follows the ISO 8601 format. " +
								"It is `\"Excel\"` if the format is `\"Excel\"` and the column value follows the Excel format. " +
								"Otherwise, it follows the format accepted by the [Python strftime function](https://docs.python.org/3/library/datetime.html#strftime-strptime-behavior).",
						},
						{
							Name:      "incremental",
							Type:      types.Boolean(),
							Prefilled: `true`,
							Description: "Indicates whether users are imported incrementally:\n" +
								"* `true`: are imported only users whose last change time is equal to or later than the last imported user's change time.\n" +
								"* `false`: all users are imported again, regardless of their last change time.",
						},
						{
							Name: "transformation",
							Type: types.Object([]types.Property{
								{
									Name:        "mapping",
									Type:        types.Map(types.Text()),
									Prefilled:   `{ "first_name": "firstName" }`,
									Nullable:    true,
									Description: "The transformation mapping. A key represents a property path in the user schema, and its corresponding value is an expression. This expression can reference columns of the file.",
								},
								{
									Name: "function",
									Type: types.Object([]types.Property{
										{
											Name:        "source",
											Type:        types.Text().WithCharLen(50_000),
											Prefilled:   `"const transform = (user) => { ... }"`,
											Description: "The source code of the JavaScript or Python function.",
										},
										{
											Name:        "language",
											Type:        types.Text().WithValues("JavaScript", "Python"),
											Prefilled:   `"JavaScript"`,
											Description: "The language of the function.",
										},
										{
											Name:        "preserveJSON",
											Type:        types.Boolean(),
											Prefilled:   "false",
											Description: "Specifies whether JSON values are passed to and returned from the function as strings, keeping their original format without any encoding or decoding.",
										},
										{
											Name:        "inPaths",
											Type:        types.Array(types.Text()),
											Prefilled:   `[ "email", "firstName", "lastName" ]`,
											Description: "The paths of the properties that will be passed to the function. At least one path must be present.",
										},
										{
											Name:        "outPaths",
											Type:        types.Array(types.Text()),
											Prefilled:   `[ "email_address", "first_name", "last_name" ]`,
											Description: "The paths of the properties that may be returned by the function. At least one path must be present.",
										},
									}),
									Nullable:    true,
									Description: "The transformation function. A JavaScript or Python function that given a user in the file, returns a user identity.",
								},
							}),
							Prefilled: `...`,
							Description: "The mapping or function responsible for transforming file users into user identities linked to the action. " +
								"Once the identity resolution process is complete, the user identities associated with all actions are merged into unified users.\n\n" +
								"One of either a mapping or a function is present, but not both. The one that is not present is null.",
						},
						{
							Name:        "inSchema",
							Type:        types.Parameter("schema"),
							Prefilled:   `{...}`,
							Description: "The schema for the properties used in the filter, the identity column, the last change time column, and the input properties for the transformation.",
						},
						{
							Name:        "outSchema",
							Type:        types.Parameter("schema"),
							Prefilled:   `{...}`,
							Description: "The schema for the output properties of the transformation.",
						},
						runningParameter,
						scheduleStartParameter,
						importSchedulePeriodParameter,
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
