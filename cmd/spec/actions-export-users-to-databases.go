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
	filterParameter := types.Property{
		Name:        "filter",
		Type:        filterType,
		Nullable:    true,
		Placeholder: `{ "logical": "and", "conditions": [ { "property": "country", "operator": "is", "values": [ "US" ] } ] }`,
		Description: "The filter applied to the users in the database table. If it's not null, only the users that match the filter will be included.\n\n" +
			"See the [filters documentation](/filters) for more details.",
	}
	tableNameParameter := types.Property{
		Name:           "tableName",
		Type:           types.Text().WithCharLen(1024),
		CreateRequired: true,
		Placeholder:    `"customers"`,
		Description:    "The name of the table where the users will be exported.",
	}
	tableKeyParameter := types.Property{
		Name:           "tableKey",
		Type:           types.Text(),
		CreateRequired: true,
		Placeholder:    `"email"`,
		Description: "The name of the table column that contains a value identifying a user within the table. This column must be included in the output schema.\n\n" +
			"Typically, this is the column used as the table's primary key. However, it can also be a column with a unique constraint, or one that is guaranteed to contain only unique values.\n\n" +
			"If a row with the same value in this column already exists, it will be updated; otherwise, a new row will be created for the exported user.\n\n" +
			"The type of this column must match one of the following Meergo types: `Int`, `Uint`, `UUID`, or `Text`.",
	}
	outSchemaParameter := types.Property{
		Name:           "outSchema",
		Type:           types.Parameter("schema"),
		Placeholder:    `{...}`,
		CreateRequired: true,
		Description: "The schema for the table key property and the output properties within the transformation.\n\n" +
			"When exporting users to a database, this should be a subset of the database table schema.",
	}
	transformationParameter := types.Property{
		Name:        "transformation",
		Type:        types.Parameter("transformation"),
		Placeholder: `{...}`,
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "actions-export-users-to-databases",
		Name: "Export users to databases",
		Description: "This type of action exports user data from the workspace's data warehouse to a database table. " +
			"It operates on a destination database connection.",
		Endpoints: []*Endpoint{
			{
				Name:        "Create action",
				Description: "Create a destination action that exports users to a database.",
				Method:      POST,
				URL:         "/v0/actions",
				Parameters: []types.Property{
					nameParameter,
					{
						Name:           "connection",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "230527183",
						Description:    "The ID of the destination database connection to which the users will be written.",
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
					tableNameParameter,
					tableKeyParameter,
					{
						Name:           "inSchema",
						Type:           types.Parameter("schema"),
						Placeholder:    `{...}`,
						CreateRequired: true,
						Description: "The schema of the properties used in the filter and the input properties within the transformation.\n\n" +
							"It should be a subset of the workspace's user schema.",
					},
					outSchemaParameter,
					transformationParameter,
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
				Description: "Update a destination action that exports users to a database.",
				Method:      PUT,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the destination database action that exports users.",
					},
					nameParameter,
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicates if the action is enabled. Use the [Set status](/api/actions#set-status) endpoint to change only the action's status.",
					},
					tableNameParameter,
					tableKeyParameter,
					outSchemaParameter,
					transformationParameter,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
					{422, UnsupportedLanguage, "transformation language is not supported"},
				},
			},
			{
				Name:        "Get action",
				Description: "Get a destination action that exports users to a database.",
				Method:      GET,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the destination database action that exports users.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Placeholder: "705981339",
							Description: "The ID of the destination database action that exports users.",
						},
						nameParameter,
						{
							Name:        "connector",
							Type:        types.Text(),
							Placeholder: `"PostgreSQL"`,
							Description: "The name of the connection's connector.",
						},
						{
							Name:        "connectorType",
							Type:        types.Text().WithValues("App", "Database", "FileStorage", "Mobile", "Server", "Website"),
							Placeholder: `"Database"`,
							Description: "The type of the connection's connector. It is always `\"Database\"` when the action exports users to a database.",
						},
						{
							Name:        "connection",
							Type:        types.Int(32),
							Placeholder: "1371036433",
							Description: "The ID of the connection to which the users will be exported. It is a destination database.",
						},
						{
							Name:        "connectionRole",
							Type:        types.Text().WithValues("Source", "Destination"),
							Placeholder: `"Destination"`,
							Description: "The role of the action's connection. It is always `\"Destination\"` when the action exports users to a database.",
						},
						{
							Name:        "target",
							Type:        types.Text().WithValues("Users", "Events"),
							Placeholder: `"Users"`,
							Description: "The entity on which the action operates. It is always `\"Users\"` when the action exports users to a database.",
						},
						{
							Name:        "enabled",
							Type:        types.Boolean(),
							Placeholder: "true",
							Description: "Indicates if the action is enabled.",
						},
						filterParameter,
						{
							Name:           "tableName",
							Type:           types.Text().WithCharLen(1024),
							CreateRequired: true,
							Placeholder:    `"customers"`,
							Description:    "The name of the table where the users are exported.",
						},
						{
							Name:           "tableKey",
							Type:           types.Text(),
							CreateRequired: true,
							Placeholder:    `"email"`,
							Description: "The name of the table column that contains a value identifying a user within the table.\n\n" +
								"Typically, this is the column used as the table's primary key. However, it can also be a column with a unique constraint, or one that is guaranteed to contain only unique values.\n\n" +
								"If a row with the same value in this column already exists, it will be updated; otherwise, a new row will be created for the exported user.\n\n" +
								"The type of this column match one of the following Meergo types: `Int`, `Uint`, `UUID`, or `Text`.",
						},
						{
							Name:        "inSchema",
							Type:        types.Parameter("schema"),
							Placeholder: `{...}`,
							Description: "The schema of the properties used in the filter and the input properties within the transformation.",
						},
						{
							Name:        "outSchema",
							Type:        types.Parameter("schema"),
							Placeholder: `{...}`,
							Description: "The schema of the table key property and the output properties within the transformation.",
						},
						{
							Name:        "transformation",
							Type:        types.Parameter("transformation"),
							Placeholder: `{...}`,
						},
						runningParameter,
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
