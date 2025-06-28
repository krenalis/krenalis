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
		Description: "The filter applied to the app users. If it's not null, only the app users that match the filter will be included, otherwise all users will be included.\n\n" +
			"See the [filters documentation](/filters) for more details.",
	}
	incrementalParameter := types.Property{
		Name:        "incremental",
		Type:        types.Boolean(),
		Placeholder: `true`,
		Description: "Determines whether users are imported incrementally:\n" +
			"* `true`: imports only users who were created or updated since the last import.\n" +
			"* `false`: imports all users, regardless of previous imports.",
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
				Description:    "The transformation mapping. A key represents a property path in the user schema, and its corresponding value is an expression. This expression can reference property paths from the source schema of the app.",
			},
			{
				Name: "function",
				Type: types.Object([]types.Property{
					{
						Name:           "source",
						Type:           types.Text().WithCharLen(50_000),
						Placeholder:    `"const transform = (user) => { ... }"`,
						CreateRequired: true,
						Description:    "The source code of the JavaScript or Python function.",
					},
					{
						Name:           "language",
						Type:           types.Text().WithValues("JavaScript", "Python"),
						Placeholder:    `"JavaScript"`,
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
				Description:    "The transformation function. A JavaScript or Python function that given an app user, returns a user identity.",
			},
		}),
		Placeholder:    `...`,
		CreateRequired: true,
		UpdateRequired: true,
		Description: "The mapping or function responsible for transforming app users into user identities linked to the action. " +
			"Once the identity resolution process is complete, the user identities associated with all actions are merged into unified users.\n\n" +
			"One of either a mapping or a function must be provided, but not both. The one that is not provided can be either missing or set to null.",
	}
	inSchemaParameter := types.Property{
		Name:           "inSchema",
		Type:           types.Parameter("schema"),
		CreateRequired: true,
		Placeholder:    `{...}`,
		Description: "The schema for the properties used in the filter, as well as the input properties for the transformation.\n\n" +
			"When importing users from apps, this should be a subset of the app's destination schema.",
	}
	outSchemaParameter := types.Property{
		Name:           "outSchema",
		Type:           types.Parameter("schema"),
		CreateRequired: true,
		Placeholder:    `{...}`,
		Description: "The schema for the output properties of the transformation.\n\n" +
			"When importing users from apps, this should be a subset of the user schema.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "actions-import-users-from-apps",
		Name: "Import users from apps",
		Description: "This type of action imports user data from an application into the workspace's data warehouse. " +
			"It operates on a source app connection that supports users.",
		Endpoints: []*Endpoint{
			{
				Name:        "Create action",
				Description: "Create a source action to import users from an app.",
				Method:      POST,
				URL:         "/v1/actions",
				Parameters: []types.Property{
					nameParameter,
					{
						Name:           "connection",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "230527183",
						Description:    "The ID of the connection from which to read the users. It must be a source app that can import users.",
					},
					{
						Name:           "target",
						Type:           types.Text().WithValues("User"),
						CreateRequired: true,
						Placeholder:    `"User"`,
						Description:    "The entity on which the action operates, which must be `\"User\"` in order to create an action that imports users.",
					},
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicates if the action is enabled once created.",
					},
					filterParameter,
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
				Description: "Update a source action that imports users from an app.",
				Method:      PUT,
				URL:         "/v1/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source app action to update.",
					},
					nameParameter,
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicates if the action is enabled. Use the [Set status](actions#set-status) endpoint to change only the action's status.",
					},
					filterParameter,
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
				Description: "Get a source action that imports users from an app.",
				Method:      GET,
				URL:         "/v1/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source app action that imports users.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Placeholder: "705981339",
							Description: "The ID of the source app action that imports users.",
						},
						nameParameter,
						{
							Name:        "connector",
							Type:        types.Text(),
							Placeholder: `"HubSpot"`,
							Description: "The name of the connection's connector.",
						},
						{
							Name:        "connectorType",
							Type:        types.Text().WithValues("App", "Database", "FileStorage", "SDK"),
							Placeholder: `"App"`,
							Description: "The type of the connection's connector. It is always `\"App\"` when the action imports users from an app.",
						},
						{
							Name:        "connection",
							Type:        types.Int(32),
							Placeholder: "1371036433",
							Description: "The ID of the connection from which users are read. It is a source app that supports users.",
						},
						{
							Name:        "connectionRole",
							Type:        types.Text().WithValues("Source", "Destination"),
							Placeholder: `"Source"`,
							Description: "The role of the action's connection. It is always `\"Source\"` when the action imports users from an app.",
						},
						{
							Name:        "target",
							Type:        types.Text().WithValues("User", "Event"),
							Placeholder: `"User"`,
							Description: "The entity on which the action operates. It is always `\"User\"` when the action imports users from an app.",
						},
						{
							Name:        "enabled",
							Type:        types.Boolean(),
							Placeholder: "true",
							Description: "Indicates if the action is enabled.",
						},
						filterParameter,
						{
							Name:        "incremental",
							Type:        types.Boolean(),
							Placeholder: `true`,
							Description: "Indicates whether users are imported incrementally:\n" +
								"* `true`: imports only users who were created or updated since the last import.\n" +
								"* `false`: imports all users, regardless of previous imports.",
						},
						{
							Name: "transformation",
							Type: types.Object([]types.Property{
								{
									Name:        "mapping",
									Type:        types.Map(types.Text()),
									Placeholder: `{ "first_name": "firstName" }`,
									Nullable:    true,
									Description: "The transformation mapping. A key represents a property path in the user schema, and its corresponding value is an expression. This expression can reference property paths from the source schema of the app.",
								},
								{
									Name: "function",
									Type: types.Object([]types.Property{
										{
											Name:        "source",
											Type:        types.Text().WithCharLen(50_000),
											Placeholder: `"const transform = (user) => { ... }"`,
											Description: "The source code of the JavaScript or Python function.",
										},
										{
											Name:        "language",
											Type:        types.Text().WithValues("JavaScript", "Python"),
											Placeholder: `"JavaScript"`,
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
									Description: "The transformation function. A JavaScript or Python function that given an app user, returns a user identity.",
								},
							}),
							Placeholder: `...`,
							Description: "The mapping or function responsible for transforming app users into user identities linked to the action. " +
								"Once identity resolution is completed, the user identities associated to all actions are merged into unified users.\n\n" +
								"One of either a mapping or a function is present, but not both. The one that is not present is null.",
						},
						{
							Name:        "inSchema",
							Type:        types.Parameter("schema"),
							Placeholder: `{...}`,
							Description: "The schema for the properties used in the filter, as well as the input properties for the transformation.",
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
