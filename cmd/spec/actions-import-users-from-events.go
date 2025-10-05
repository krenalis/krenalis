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
		Prefilled:      `"Site example.com"`,
		Description:    "The action's name.",
	}
	filterParameter := types.Property{
		Name:      "filter",
		Type:      filterType,
		Nullable:  true,
		Prefilled: `{ "logical": "and", "conditions": [ { "property": "type", "operator": "is", "values": [ "track" ] } ] }`,
		Description: "The filter applied to the events. If it's not null, only the users of events that match the filter will be imported.\n\n" +
			"See the [filters documentation](/filters) for more details.",
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
				Description:    "The transformation mapping. A key represents a property path in the user schema, and its corresponding value is an expression. This expression can reference property paths from the event schema.",
			},
			{
				Name: "function",
				Type: types.Object([]types.Property{
					{
						Name:           "source",
						Type:           types.Text().WithCharLen(50_000),
						Prefilled:      `"const transform = (event) => { ... }"`,
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
						Prefilled:      `[ "traits.firstName", "traits.lastName" ]`,
						CreateRequired: true,
						Description:    "The paths of the properties that will be passed to the function. At least one path must be present.",
					},
					{
						Name:           "outPaths",
						Type:           types.Array(types.Text()),
						Prefilled:      `[ "first_name", "last_name" ]`,
						CreateRequired: true,
						Description:    "The paths of the properties that may be returned by the function. At least one path must be present.",
					},
				}),
				UpdateRequired: true,
				Nullable:       true,
				Description:    "The transformation function. A JavaScript or Python function that given an event, returns a user identity.",
			},
		}),
		Prefilled: `...`,
		Nullable:  true,
		Description: "The mapping or function responsible for transforming incoming events into user identities linked to the action. " +
			"Once the identity resolution process is complete, the user identities associated with all actions are merged into unified users.\n\n" +
			"* If no transformation is provided (or if it's set to null), the imported user identity will not have any properties.\n" +
			"* If a mapping or function is provided, only one of them should be specified. The other must either be absent or set to null.\n\n" +
			"In any case, the imported user identity will always include an Anonymous ID and, if available, a User ID.",
	}
	outSchemaParameter := types.Property{
		Name:           "outSchema",
		Type:           types.Parameter("schema"),
		Prefilled:      `{...}`,
		UpdateRequired: true,
		Nullable:       true,
		Description: "The schema for the output properties of the transformation. It is required if a transformation is present.\n\n" +
			"When importing users from events, this should be a subset of the user schema.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "actions/import-users-from-events",
		Name:        "Import users from events",
		Description: "This type of action imports user data into the workspace's data warehouse from events received from websites, mobile apps, and servers.",
		Endpoints: []*Endpoint{
			{
				Name:        "Create action",
				Description: "Create a source action that imports users from events.",
				Method:      POST,
				URL:         "/v1/actions",
				Parameters: []types.Property{
					nameParameter,
					{
						Name:           "connection",
						Type:           types.Int(32),
						CreateRequired: true,
						Prefilled:      "230527183",
						Description:    "The ID of the connection from which the events are received. It must be a source SDK connection.",
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
					filterParameter,
					transformationParameter,
					outSchemaParameter,
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Prefilled:   "705981339",
							Description: "The ID of the action.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, ConnectionNotExist, "connection does not exist"},
					{422, ConnectorNotExist, "connector does not exist"},
				},
			},
			{
				Name:        "Update action",
				Description: "Update a source action that imports users from events.",
				Method:      PUT,
				URL:         "/v1/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Prefilled:      "705981339",
						Description:    "The ID of the source SDK action.",
					},
					nameParameter,
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Prefilled:   "true",
						Description: "Indicates if the action is enabled. Use the [Set status](actions#set-status) endpoint to change only the action's status.",
					},
					filterParameter,
					transformationParameter,
					outSchemaParameter,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
				},
			},
			{
				Name:        "Get action",
				Description: "Get a source action that imports users from events.",
				Method:      GET,
				URL:         "/v1/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Prefilled:      "705981339",
						Description:    "The ID of the source SDK action.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Prefilled:   "705981339",
							Description: "The ID of the source SDK action.",
						},
						nameParameter,
						{
							Name:        "connector",
							Type:        types.Text(),
							Prefilled:   `"java"`,
							Description: "The code of the connection's connector.",
						},
						{
							Name:        "connectorType",
							Type:        types.Text().WithValues("App", "Database", "FileStorage", "SDK"),
							Prefilled:   `"SDK"`,
							Description: "The type of the connection's connector. It is always `\"SDK\"` when the action imports users from events.",
						},
						{
							Name:        "connection",
							Type:        types.Int(32),
							Prefilled:   "1371036433",
							Description: "The ID of the connection from which the events are received. It is a source SDK connection.",
						},
						{
							Name:        "connectionRole",
							Type:        types.Text().WithValues("Source", "Destination"),
							Prefilled:   `"Source"`,
							Description: "The role of the action's connection. It is always `\"Source\"` when the action imports users from events.",
						},
						{
							Name:        "target",
							Type:        types.Text().WithValues("User", "Event"),
							Prefilled:   `"User"`,
							Description: "The entity on which the action operates. It is always `\"User\"` when the action imports users from events.",
						},
						{
							Name:        "enabled",
							Type:        types.Boolean(),
							Prefilled:   "true",
							Description: "Indicates if the action is enabled.",
						},
						filterParameter,
						{
							Name: "transformation",
							Type: types.Object([]types.Property{
								{
									Name:        "mapping",
									Type:        types.Map(types.Text()),
									Prefilled:   `{ "first_name": "firstName" }`,
									Nullable:    true,
									Description: "The transformation mapping. A key represents a property path in the user schema, and its corresponding value is an expression. This expression can reference property paths from the event schema.",
								},
								{
									Name: "function",
									Type: types.Object([]types.Property{
										{
											Name:        "source",
											Type:        types.Text().WithCharLen(50_000),
											Prefilled:   `"const transform = (event) => { ... }"`,
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
											Prefilled:   `[ "traits.firstName", "traits.lastName" ]`,
											Description: "The paths of the properties that will be passed to the function. It contains at least one property path.",
										},
										{
											Name:        "outPaths",
											Type:        types.Array(types.Text()),
											Prefilled:   `[ "first_name", "last_name" ]`,
											Description: "The paths of the properties that may be returned by the function. It contains at least one property path.",
										},
									}),
									Nullable:    true,
									Description: "The transformation function. A JavaScript or Python function that given an event, returns a user identity.",
								},
							}),
							Prefilled: `...`,
							Nullable:  true,
							Description: "The mapping or function responsible for transforming incoming events into user identities linked to the action. " +
								"Once the identity resolution process is complete, the user identities associated with all actions are merged into unified users.\n\n" +
								"* If no transformation is present, it is null.\n" +
								"* If a transformation is present, either a mapping or a function will be present, but not both. The one that is not present it is null.",
						},
						{
							Name:        "inSchema",
							Type:        types.Parameter("schema"),
							Prefilled:   `{...}`,
							Description: "The schema for the properties used in the filter and any input properties for the transformation.",
						},
						{
							Name:        "outSchema",
							Type:        types.Parameter("schema"),
							Prefilled:   `{...}`,
							Nullable:    true,
							Description: "The schema for the output properties of the transformation. If no transformation is present, it is null.",
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
