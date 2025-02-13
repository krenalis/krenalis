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
		Placeholder:    `"PostgreSQL"`,
		Description:    "The action's name.",
	}
	queryParameter := types.Property{
		Name:           "query",
		Type:           types.Text().WithCharLen(1000),
		CreateRequired: true,
		Placeholder:    `"SELECT name, email, updated_at FROM customers WHERE ${last_change_time}"`,
		Description: "The SELECT query executed on the database to retrieve the users to import.\n\n" +
			"The column names returned must be valid property names. They must:\n" +
			"- Start with `A`...`Z`, `a`...`z`, or `_`,\n" +
			"- Contain only `A`...`Z`, `a`...`z`, `0`...`9`, or `_`.\n\n" +
			"Note that you can use a column alias if necessary (e.g., `SELECT 1 AS v FROM users`).\n\n" +
			"The query should include the `last_change_time` placeholder as a condition in the WHERE clause.",
	}
	identityColumnParameter := types.Property{
		Name:           "identityColumn",
		Type:           types.Text().WithCharLen(1024),
		CreateRequired: true,
		Placeholder:    `"email"`,
		Description: "The column that uniquely identifies each user in the database. It serves as the single, unique identifier for each user record, ensuring that each user can be distinctly referenced.\n\n" +
			"Only columns with types corresponding to the following Meergo types can be used as an identity: `int`, `uint`, `uuid`, `json`, and `text`.",
	}
	lastChangeTimeParameter := types.Property{
		Name:        "lastChangeTimeColumn",
		Type:        types.Text().WithCharLen(1024),
		Placeholder: `"updated_at"`,
		Description: "The column that stores the date when a user record was last updated. It tracks the most recent modification made to the user’s data, helping to identify when changes occurred.\n\n" +
			"The value of this column is used for incremental imports, where only records that have been modified since the last import need to be processed.\n\n" +
			"Only columns with types corresponding to the following Meergo types can be used as the last change time: `datetime`, `date`, `json`, and `text`.",
	}
	lastChangeTimeFormatParameter := types.Property{
		Name:           "lastChangeTimeFormat",
		Type:           types.Text().WithCharLen(64),
		UpdateRequired: true,
		Placeholder:    `"ISO8601"`,
		Description: "The format of the value in the last change time column. It can be set to `\"ISO8601\"` if the column value follows the ISO 8601 format. " +
			"Otherwise, it should follow a format accepted by the [Python strftime function](https://docs.python.org/3/library/datetime.html#strftime-strptime-behavior).\n\n" +
			"This field is only required if the `lastChangeTimeColumn` is provided, is not empty, and has a type `json` or `text`.",
	}
	incrementalParameter := types.Property{
		Name:        "incremental",
		Type:        types.Boolean(),
		Placeholder: `true`,
		Description: "Determines whether users are imported incrementally:\n" +
			"* `true`: are imported only users whose last change time is equal to or later than the last imported user's change time.\n" +
			"* `false`: all users are imported again, regardless of their last change time. `false` is the default value.\n\n" +
			"If set to `true`, a column for the last change time must be specified (i.e., `lastChangeTimeColumn` is not null). " +
			"Additionally, if the last change time column type corresponds to the Meergo `json` or `text` types, the values in the column must be sortable.",
	}
	transformationParameter := types.Property{
		Name: "transformation",
		Type: types.Object([]types.Property{
			{
				Name:           "mapping",
				Type:           types.Map(types.Text()),
				Placeholder:    `{ "first_name": "firstName" }`,
				UpdateRequired: true,
				Nullable:       true,
				Description:    "The transformation mapping. A key represents a property path in the user schema, and its corresponding value is an expression. This expression can reference columns of the database table.",
			},
			{
				Name: "function",
				Type: types.Object([]types.Property{
					{
						Name:           "source",
						Type:           types.Text().WithCharLen(50_000),
						Placeholder:    `const transform = (user) => { ... }`,
						CreateRequired: true,
						Description:    "The source code of the JavaScript or Python function.",
					},
					{
						Name:           "language",
						Type:           types.Text().WithValues("JavaScript", "Python"),
						Placeholder:    "JavaScript",
						CreateRequired: true,
						Description:    "The language of the function.",
					},
					{
						Name:        "preserveJSON",
						Type:        types.Boolean(),
						Placeholder: "false",
						Description: "Specifies whether JSON values are passed to and returned from the function as strings, keeping their original format without any encoding or decoding.",
					},
					{
						Name:           "inPaths",
						Type:           types.Array(types.Text()),
						Placeholder:    `[ "email", "firstName", "lastName" ]`,
						CreateRequired: true,
						Description:    "The paths of the properties that will be passed to the function. At least one path must be present.",
					},
					{
						Name:           "outPaths",
						Type:           types.Array(types.Text()),
						Placeholder:    `[ "email_address", "first_name", "last_name" ]`,
						CreateRequired: true,
						Description:    "The paths of the properties that may be returned by the function. At least one path must be present.",
					},
				}),
				UpdateRequired: true,
				Nullable:       true,
				Description:    "The transformation function. A JavaScript or Python function that given a user in the database, returns a user identity.",
			},
		}),
		Placeholder:    `...`,
		CreateRequired: true,
		Description: "The mapping or function responsible for transforming database users into user identities linked to the action. " +
			"Once the identity resolution process is complete, the user identities associated with all actions are merged into unified users.\n\n" +
			"One of either a mapping or a function must be provided, but not both. The one that is not provided can be either missing or set to null.",
	}
	inSchemaParameter := types.Property{
		Name:           "inSchema",
		Type:           types.Parameter("schema"),
		CreateRequired: true,
		Placeholder:    `{...}`,
		Description: "The schema for the identity column, the last change time column, and the input properties for the transformation.\n\n" +
			"When importing users from databases, this should be a subset of the query schema.",
	}
	outSchemaParameter := types.Property{
		Name:           "outSchema",
		Type:           types.Parameter("schema"),
		CreateRequired: true,
		Placeholder:    `{...}`,
		Description: "The schema for the output properties of the transformation.\n\n" +
			"When importing users from databases, this should be a subset of the user schema.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "actions-import-users-from-databases",
		Name: "Import users from databases",
		Description: "This type of action imports user data from a database into the workspace's data warehouse. " +
			"It operates on a source database connection.",
		Endpoints: []*Endpoint{
			{
				Name:        "Create action",
				Description: "Create a source action to import users from a database.",
				Method:      POST,
				URL:         "/v1/actions",
				Parameters: []types.Property{
					nameParameter,
					{
						Name:           "connection",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "230527183",
						Description:    "The ID of the connection from which to read the users. It must be a source database.",
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
					queryParameter,
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
				Name:        "Update action",
				Description: "Update a source action to import users from database.",
				Method:      PUT,
				URL:         "/v1/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source database action to update.",
					},
					nameParameter,
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicates if the action is enabled. Use the [Set status](/api/actions#set-status) endpoint to change only the action's status.",
					},
					queryParameter,
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
					{422, UnsupportedLanguage, "transformation language is not supported"},
				},
			},
			{
				Name:        "Get action",
				Description: "Get a source action that imports users from a database.",
				Method:      GET,
				URL:         "/v1/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source database action.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Placeholder: "705981339",
							Description: "The ID of the source database action.",
						},
						nameParameter,
						{
							Name:        "connector",
							Type:        types.Text(),
							Placeholder: `"MySQL"`,
							Description: "The name of the connection's connector.",
						},
						{
							Name:        "connectorType",
							Type:        types.Text().WithValues("App", "Database", "FileStorage", "Mobile", "Server", "Website"),
							Placeholder: `"Database"`,
							Description: "The type of the connection's connector. It is always `\"Database\"` when the action imports users from a database.",
						},
						{
							Name:        "connection",
							Type:        types.Int(32),
							Placeholder: "1371036433",
							Description: "The ID of the connection from which users are read. It is a source database.",
						},
						{
							Name:        "connectionRole",
							Type:        types.Text().WithValues("Source", "Destination"),
							Placeholder: `"Source"`,
							Description: "The role of the action's connection. It is always `\"Source\"` when the action imports users from a database.",
						},
						{
							Name:        "target",
							Type:        types.Text().WithValues("Users", "Events"),
							Placeholder: `"Users"`,
							Description: "The entity on which the action operates. It is always `\"Users\"` when the action imports users from a database.",
						},
						{
							Name:        "enabled",
							Type:        types.Boolean(),
							Placeholder: "true",
							Description: "Indicates if the action is enabled.",
						},
						{
							Name:        "query",
							Type:        types.Text().WithCharLen(1000),
							Placeholder: `"SELECT name, email, updated_at FROM customers WHERE ${last_change_time}"`,
							Description: "The SELECT query executed on the database to retrieve the users to import.",
						},
						{
							Name:        "identityColumn",
							Type:        types.Text(),
							Placeholder: `"email"`,
							Description: "The column that uniquely identifies each user in the database.",
						},
						{
							Name:        "lastChangeTimeColumn",
							Type:        types.Text(),
							Nullable:    true,
							Placeholder: `"updated_at"`,
							Description: "The column that stores the timestamp of the last update to a user record. It is null if no such column exists.",
						},
						{
							Name:        "lastChangeTimeFormat",
							Type:        types.Text(),
							Nullable:    true,
							Placeholder: `"ISO8601"`,
							Description: "The format of the value in the last change time column. It is null if no such column exists or if the corresponding Meergo type is `datetime` or `date`.\n\n" +
								"It is `\"ISO8601\"` if the column value follows the ISO 8601 format. " +
								"Otherwise, it follows the format accepted by the [Python strftime function](https://docs.python.org/3/library/datetime.html#strftime-strptime-behavior).",
						},
						{
							Name:        "incremental",
							Type:        types.Boolean(),
							Placeholder: `true`,
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
									Placeholder: `{ "first_name": "firstName" }`,
									Nullable:    true,
									Description: "The transformation mapping. A key represents a property path in the user schema, and its corresponding value is an expression. This expression can reference columns of the database table.",
								},
								{
									Name: "function",
									Type: types.Object([]types.Property{
										{
											Name:        "source",
											Type:        types.Text().WithCharLen(50_000),
											Placeholder: `const transform = (user) => { ... }`,
											Description: "The source code of the JavaScript or Python function.",
										},
										{
											Name:        "language",
											Type:        types.Text().WithValues("JavaScript", "Python"),
											Placeholder: "JavaScript",
											Description: "The language of the function.",
										},
										{
											Name:        "preserveJSON",
											Type:        types.Boolean(),
											Placeholder: "false",
											Description: "Specifies whether JSON values are passed to and returned from the function as strings, keeping their original format without any encoding or decoding.",
										},
										{
											Name:        "inPaths",
											Type:        types.Array(types.Text()),
											Placeholder: `[ "email", "firstName", "lastName" ]`,
											Description: "The paths of the properties that will be passed to the function. It contains at least one property path.",
										},
										{
											Name:        "outPaths",
											Type:        types.Array(types.Text()),
											Placeholder: `[ "email_address", "first_name", "last_name" ]`,
											Description: "The paths of the properties that may be returned by the function. It contains at least one property path.",
										},
									}),
									Nullable:    true,
									Description: "The transformation function. A JavaScript or Python function that given a user in the database, returns a user identity.",
								},
							}),
							Placeholder: `...`,
							Description: "The mapping or function responsible for transforming database users into user identities linked to the action. " +
								"Once identity resolution is completed, the user identities associated to all actions are merged into unified users.\n\n" +
								"One of either a mapping or a function is present, but not both. The one that is not present is null.",
						},
						{
							Name:        "inSchema",
							Type:        types.Parameter("schema"),
							Placeholder: `{...}`,
							Description: "The schema for the identity column, the last change time column, and the input properties for the transformation.",
						},
						{
							Name:        "outSchema",
							Type:        types.Parameter("schema"),
							Placeholder: `{...}`,
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
