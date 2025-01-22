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
	identityPropertyParameter := types.Property{
		Name:           "identityProperty",
		Type:           types.Text().WithCharLen(1024),
		CreateRequired: true,
		Placeholder:    `"email"`,
		Description: "The column that uniquely identifies each user in the database. It serves as the single, unique identifier for each user record, ensuring that each user can be distinctly referenced.\n\n" +
			"Only columns with types corresponding to the following Meergo types can be used as an identity: `Int`, `Uint`, `UUID`, `JSON`, and `Text`.",
	}
	lastChangeTimeProperty := types.Property{
		Name:        "lastChangeTimeProperty",
		Type:        types.Text().WithCharLen(1024),
		Placeholder: `"updated_at"`,
		Description: "The column that stores the date when a user record was last updated. It tracks the most recent modification made to the user’s data, helping to identify when changes occurred.\n\n" +
			"The value of this column is used for incremental imports, where only records that have been modified since the last import need to be processed.\n\n" +
			"Only columns with types corresponding to the following Meergo types can be used as the last change time: `Date`, `DateTime`, `JSON`, and `Text`.",
	}
	lastChangeTimeFormat := types.Property{
		Name:           "lastChangeTimeFormat",
		Type:           types.Text().WithCharLen(64),
		UpdateRequired: true,
		Placeholder:    `"ISO8601"`,
		Description: "The format of the value in the last change time column. It can be set to `\"ISO8601\"` if the column value follows the ISO 8601 format. " +
			"Otherwise, it should follow a format accepted by the [Python strftime function](https://docs.python.org/3/library/datetime.html#strftime-strptime-behavior).\n\n" +
			"This field is only required if the `lastChangeTimeProperty` is provided, is not empty, and has a type `JSON` or `Text`.",
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
				URL:         "/v0/actions",
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
					identityPropertyParameter,
					lastChangeTimeProperty,
					lastChangeTimeFormat,
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
				Name:        "Update action",
				Description: "Update a source action to import users from database.",
				Method:      PUT,
				URL:         "/v0/actions/:id",
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
					identityPropertyParameter,
					lastChangeTimeProperty,
					lastChangeTimeFormat,
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
				Name:        "Get action",
				Description: "Get a source action that imports users from a database.",
				Method:      GET,
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
							Name:        "identityProperty",
							Type:        types.Text(),
							Placeholder: `"email"`,
							Description: "The column that uniquely identifies each user in the database.",
						},
						{
							Name:        "lastChangeTimeProperty",
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
							Description: "The format of the value in the last change time column. It is null if no such column exists or if the corresponding Meergo type is `Date` or `DateTime`.\n\n" +
								"It is `\"ISO8601\"` if the column value follows the ISO 8601 format. " +
								"Otherwise, it follows the format accepted by the [Python strftime function](https://docs.python.org/3/library/datetime.html#strftime-strptime-behavior).",
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
		},
	})
}
