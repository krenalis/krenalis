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
		Placeholder:    `"HubSpot"`,
		Description:    "The action's name.",
	}
	filterParameter := types.Property{
		Name:        "filter",
		Type:        filterType,
		Nullable:    true,
		Placeholder: `{ "logical": "and", "conditions": [ { "property": "country", "operator": "is", "values": [ "US" ] } ] }`,
		Description: "The filter applied to the app users. If it's not null, only the users that match the filter will be included.\n\n" +
			"See the [filters documentation](/filters) for more details.",
	}
	exportModeParameter := types.Property{
		Name:           "exportMode",
		Type:           types.Text(),
		CreateRequired: true,
		Placeholder:    `"CreateOnly"`,
		Description: "The mode in which users are exported:\n\n" +
			"* `\"CreateOnly\"`: Only new users are created in the app. No existing users are modified.\n" +
			"* `\"UpdateOnly\"`: Only existing users are updated in the app. No new users are created.\n" +
			"* `\"CreateOrUpdate\"`: If a user already exists in the app, they are updated; otherwise, they are created as a new user.",
	}
	matchingParameter := types.Property{
		Name: "matching",
		Type: types.Object([]types.Property{
			{
				Name:           "in",
				Type:           types.Text(),
				CreateRequired: true,
				Placeholder:    `"email"`,
				Description:    "The matching input property. It cannot be empty.\n\nIt represents the name of the property in the user schema. Its definition must also be included in the action's input schema.",
			},
			{
				Name:           "out",
				Type:           types.Text(),
				CreateRequired: true,
				Placeholder:    `"email"`,
				Description:    "The matching output property. It cannot be empty.\n\nIt represents the name of the property in the app's user schema. Its definition must also be included in the action's output schema.",
			},
		}),
		CreateRequired: true,
		Description: "The properties used to identify the match between a user in the workspace and a user in the app. " +
			"These properties are required to determine which users should be updated and which should be created as new in the app.",
	}
	exportOnDuplicatesParameter := types.Property{
		Name:        "exportOnDuplicates",
		Type:        types.Boolean(),
		Placeholder: `true`,
		Description: "Determines whether a user should be exported even if there are multiple matching users in the app.\n\n" +
			"If set to true, the export will proceed regardless of duplicates, otherwise the user will not be exported, and an error will be logged.",
	}
	transformationParameter := types.Property{
		Name: "transformation",
		Type: types.Object([]types.Property{
			{
				Name:           "mapping",
				Type:           types.Map(types.Text()),
				Placeholder:    `{ "firstName": "first_name" }`,
				UpdateRequired: true,
				Nullable:       true,
				Description:    "The transformation mapping. A key represents a path of a property in the destination schema of the app, and its corresponding value is an expression. This expression can reference property paths from the user schema.",
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
						Placeholder:    `[ "email", "first_name", "last_name" ]`,
						CreateRequired: true,
						Description:    "The paths of the properties that will be passed to the function. At least one path must be present.",
					},
					{
						Name:           "outPaths",
						Type:           types.Array(types.Text()),
						Placeholder:    `[ "emailAddress", "firstName", "lastName" ]`,
						CreateRequired: true,
						Description:    "The paths of the properties that may be returned by the function. At least one path must be present.",
					},
				}),
				Placeholder:    `null`,
				UpdateRequired: true,
				Nullable:       true,
				Description:    "The transformation function. A JavaScript or Python function that given a workspace's user, returns an app user.",
			},
		}),
		Placeholder:    `...`,
		CreateRequired: true,
		Description: "The mapping or function responsible for transforming unified users into app users.\n\n" +
			"One of either a mapping or a function must be provided, but not both. The one that is not provided can be either missing or set to null.",
	}
	inSchemaParameter := types.Property{
		Name:           "inSchema",
		Type:           types.Parameter("schema"),
		Placeholder:    `{...}`,
		CreateRequired: true,
		Description: "The schema of the input matching property, the properties used in the filter, and the input properties within the transformation.\n\n" +
			"When exporting users to apps, it should be a subset of user schema.",
	}
	outSchemaParameter := types.Property{
		Name:           "outSchema",
		Type:           types.Parameter("schema"),
		Placeholder:    `{...}`,
		CreateRequired: true,
		Description: "The schema of the output matching property and the output properties within the transformation.\n\n" +
			"When exporting users to apps, it should be a subset of app's destination schema, also including the schema of the output matching property if it is not already present.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "actions-export-users-to-apps",
		Name: "Export users to apps",
		Description: "This type of action exports user data from the workspace's data warehouse to an application. " +
			"It operates on a destination app connection that supports users.",
		Endpoints: []*Endpoint{
			{
				Name:        "Create action",
				Description: "Create a destination action that exports users to an app.",
				Method:      POST,
				URL:         "/v0/actions",
				Parameters: []types.Property{
					nameParameter,
					{
						Name:           "connection",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "230527183",
						Description:    "The ID of the connection to which the users will be written. It must be a destination app connection that exports users.",
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
					filterParameter,
					exportModeParameter,
					matchingParameter,
					exportOnDuplicatesParameter,
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
				Description: "Update a destination action that exports users to an app.",
				Method:      PUT,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the destination app action that exports users.",
					},
					nameParameter,
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicates if the action is enabled. Use the [Set status](/api/actions#set-status) endpoint to change only the action's status.",
					},
					filterParameter,
					exportModeParameter,
					matchingParameter,
					exportOnDuplicatesParameter,
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
				Description: "Get a destination action that exports users to an app.",
				Method:      GET,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the destination app action that exports users.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Placeholder: "705981339",
							Description: "The ID of the destination app action that exports users.",
						},
						nameParameter,
						{
							Name:        "connector",
							Type:        types.Text(),
							Placeholder: `"Mailchimp"`,
							Description: "The name of the connection's connector.",
						},
						{
							Name:        "connectorType",
							Type:        types.Text().WithValues("App", "Database", "FileStorage", "Mobile", "Server", "Website"),
							Placeholder: `"App"`,
							Description: "The type of the connection's connector. It is always `\"App\"` when the action exports users to an app.",
						},
						{
							Name:        "connection",
							Type:        types.Int(32),
							Placeholder: "1371036433",
							Description: "The ID of the connection to which the users will be exported. It is a destination app.",
						},
						{
							Name:        "connectionRole",
							Type:        types.Text().WithValues("Source", "Destination"),
							Placeholder: `"Destination"`,
							Description: "The role of the action's connection. It is always `\"Destination\"` when the action exports users to an app.",
						},
						{
							Name:        "target",
							Type:        types.Text().WithValues("Users", "Events"),
							Placeholder: `"Users"`,
							Description: "The entity on which the action operates. It is always `\"Users\"` when the action exports users to an app.",
						},
						{
							Name:        "enabled",
							Type:        types.Boolean(),
							Placeholder: "true",
							Description: "Indicates if the action is enabled.",
						},
						filterParameter,
						exportModeParameter,
						{
							Name: "matching",
							Type: types.Object([]types.Property{
								{
									Name:           "in",
									Type:           types.Text(),
									CreateRequired: true,
									Placeholder:    `"email"`,
									Description:    "The matching input property.\n\nIt represents the name of the property in the user schema. Its definition is included in the action's input schema.",
								},
								{
									Name:           "out",
									Type:           types.Text(),
									CreateRequired: true,
									Placeholder:    `"email"`,
									Description:    "The matching output property.\n\nIt represents the name of the property in the app's user schema. Its definition is included in the action's output schema.",
								},
							}),
							CreateRequired: true,
							Description: "The properties used to identify the match between a user in the workspace and a user in the app. " +
								"These properties determine which users should be updated and which should be created as new in the app.",
						},
						{
							Name:        "exportOnDuplicates",
							Type:        types.Boolean(),
							Placeholder: `true`,
							Description: "Determines whether a user should be exported even if there are multiple matching users in the app.\n\n" +
								"If true, the export will proceed regardless of duplicates. If false, the user will not be exported, and an error will be logged.",
						},
						{
							Name: "transformation",
							Type: types.Object([]types.Property{
								{
									Name:        "mapping",
									Type:        types.Map(types.Text()),
									Placeholder: `{ "firstName": "first_name" }`,
									Nullable:    true,
									Description: "The transformation mapping. A key represents a path of a property in the destination schema of the app, and its corresponding value is an expression. This expression can reference property paths from the user schema.",
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
											Placeholder: `[ "email", "first_name", "last_name" ]`,
											Description: "The paths of the properties that will be passed to the function. It contains at least one property path.",
										},
										{
											Name:        "outPaths",
											Type:        types.Array(types.Text()),
											Placeholder: `[ "emailAddress", "firstName", "lastName" ]`,
											Description: "The paths of the properties that may be returned by the function. It contains at least one property path.",
										},
									}),
									Placeholder: `null`,
									Nullable:    true,
									Description: "The transformation function. A JavaScript or Python function that given a workspace's user, returns an app user.",
								},
							}),
							Placeholder: `...`,
							Description: "The mapping or function responsible for transforming unified users into app users.\n\n" +
								"One of either a mapping or a function is present, but not both. The one that is not present is null.",
						},
						{
							Name:        "inSchema",
							Type:        types.Parameter("schema"),
							Placeholder: `{...}`,
							Description: "The schema of the input matching property, the properties used in the filter, and the input properties within the transformation.",
						},
						{
							Name:        "outSchema",
							Type:        types.Parameter("schema"),
							Placeholder: `{...}`,
							Description: "The schema of the output matching property and the output properties within the transformation.",
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
