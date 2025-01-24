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
		Placeholder:    `"Mixpanel"`,
		Description:    "The action's name.",
	}
	filterParameter := types.Property{
		Name:        "filter",
		Type:        filterType,
		Nullable:    true,
		Placeholder: `{ "logical": "and", "conditions": [ { "property": "type", "operator": "is", "values": [ "track" ] } ] }`,
		Description: "The filter applied to the events. If it's not null, only the events that match the filter will be sent to the destination.\n\n" +
			"See the [filters documentation](/filters) for more details.",
	}
	outSchemaParameter := types.Property{
		Name:           "outSchema",
		Type:           types.Parameter("schema"),
		Placeholder:    `{...}`,
		UpdateRequired: true,
		Nullable:       true,
		Description: "The schema for the output properties of the transformation. It is required and must not be null if a transformation is present.\n\n" +
			"It should be a subset of the schema of the passed event type.",
	}
	transformationParameter := types.Property{
		Name: "transformation",
		Type: types.Object([]types.Property{
			{
				Name:        "mapping",
				Type:        types.Map(types.Text()),
				Placeholder: `{ "first_name": "firstName" }`,
				Nullable:    true,
				Description: "The transformation mapping. Each key represents a property path in the event type schema, and its corresponding value is an expression. This expression can reference property paths from the event schema.",
			},
			{
				Name: "function",
				Type: types.Object([]types.Property{
					{
						Name:        "source",
						Type:        types.Text().WithCharLen(50_000),
						Placeholder: `const transform = (event) => { ... }`,
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
						Placeholder: `[ "traits.firstName", "traits.lastName" ]`,
						Description: "The property paths that will be passed to the function. This can be empty if the function does not rely on the event to generate the response.",
					},
					{
						Name:        "outPaths",
						Type:        types.Array(types.Text()),
						Placeholder: `[ "first_name", "last_name" ]`,
						Description: "The paths of the properties that may be returned by the function. At least one path must be present.",
					},
				}),
				Placeholder: `null`,
				Nullable:    true,
				Description: "The transformation function. A JavaScript or Python function that, given an event, returns the values needed to send the event to the app.",
			},
		}),
		Placeholder: `...`,
		Nullable:    true,
		Description: "This mapping or function is responsible for transforming unified users into the necessary values for sending the event to the app.\n\n" +
			"If the event type's schema requires a specific property, you should provide a transformation that returns a value for this property.\n" +
			"If a mapping or function is provided (not null), only one of them should be specified. The other must either be absent or set to null.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "actions-send-events-to-apps",
		Name: "Send events to apps",
		Description: "This type of action sends the received events to applications. " +
			"It operates on a destination app connection that supports events.",
		Endpoints: []*Endpoint{
			{
				Name:        "Create action",
				Description: "Create a destination action that send events to an app.",
				Method:      POST,
				URL:         "/v0/actions",
				Parameters: []types.Property{
					nameParameter,
					{
						Name:           "connection",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "753166510",
						Description:    "The ID of the connection to which the events will be sent. It must be a source app that supports events.",
					},
					{
						Name:           "target",
						Type:           types.Text().WithValues("Events"),
						CreateRequired: true,
						Placeholder:    `"Events"`,
						Description:    "The entity on which the action operates, which must be `\"Events\"` in order to create an action that sends events.",
					},
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicate if the action is enabled once created.",
					},
					filterParameter,
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
					{422, EventTypeNotExists, "connection does not have event type"},
					{422, ConnectorNotExist, "connector does not exist"},
					{422, UnsupportedLanguage, "transformation language is not supported"},
				},
			},
			{
				Name:        "Update action",
				Description: "Update a destination action that send events to an app.",
				Method:      PUT,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the destination app action on event.",
					},
					nameParameter,
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicates if the action is enabled. Use the [Set status](/api/actions#set-status) endpoint to change only the action's status.",
					},
					filterParameter,
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
				Description: "Get a destination action that send events to an app.",
				Method:      GET,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the destination app action on event.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Placeholder: "705981339",
							Description: "The ID of the destination app action on event.",
						},
						nameParameter,
						{
							Name:        "connector",
							Type:        types.Text(),
							Placeholder: `"Klaviyo"`,
							Description: "The name of the connection's connector.",
						},
						{
							Name:        "connectorType",
							Type:        types.Text().WithValues("App", "Database", "FileStorage", "Mobile", "Server", "Website"),
							Placeholder: `"Website"`,
							Description: "The type of the connection's connector. It is always `\"Mobile\"`, `\"Server\"`, or `\"Website\"` when the action sends events to an app.",
						},
						{
							Name:        "connection",
							Type:        types.Int(32),
							Placeholder: "1371036433",
							Description: "The ID of the connection to which the events will be sent. It is a destination app that supports events.",
						},
						{
							Name:        "connectionRole",
							Type:        types.Text().WithValues("Source", "Destination"),
							Placeholder: `"Destination"`,
							Description: "The role of the action's connection. It is always `\"Destination\"` when the action sends events to an app.",
						},
						{
							Name:        "target",
							Type:        types.Text().WithValues("Users", "Events"),
							Placeholder: `"Events"`,
							Description: "The entity on which the action operates. It is always `\"Events\"` when the action sends events to an app.",
						},
						{
							Name:        "enabled",
							Type:        types.Boolean(),
							Placeholder: "true",
							Description: "Indicates if the action is enabled.",
						},
						{
							Name:        "eventType",
							Type:        types.Text(),
							Placeholder: `"send_add_to_cart"`,
							Description: "The action's event type.",
						},
						filterParameter,
						{
							Name:        "inSchema",
							Type:        types.Parameter("schema"),
							Placeholder: `{...}`,
							Description: "The schema for the properties used in the filter and the input properties in the transformation.\n\n" +
								"When sending events to apps, this is the event schema.",
						},
						{
							Name:        "outSchema",
							Type:        types.Parameter("schema"),
							Nullable:    true,
							Placeholder: `{...}`,
							Description: "The schema for the output properties of the transformation. It is null if no transformation is present.",
						},
						{
							Name: "transformation",
							Type: types.Object([]types.Property{
								{
									Name:        "mapping",
									Type:        types.Map(types.Text()),
									Placeholder: `{ "first_name": "firstName" }`,
									Nullable:    true,
									Description: "The transformation mapping. Each key represents a property path in the event type schema, and its corresponding value is an expression. This expression can reference property paths from the event schema.",
								},
								{
									Name: "function",
									Type: types.Object([]types.Property{
										{
											Name:        "source",
											Type:        types.Text().WithCharLen(50_000),
											Placeholder: `const transform = (event) => { ... }`,
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
											Placeholder: `[ "traits.firstName", "traits.lastName" ]`,
											Description: "The property paths that will be passed to the function. If empty, the function does not rely on the event to generate the response.",
										},
										{
											Name:        "outPaths",
											Type:        types.Array(types.Text()),
											Placeholder: `[ "first_name", "last_name" ]`,
											Description: "The paths of the properties that may be returned by the function. It contains at least one property path.",
										},
									}),
									Placeholder: `null`,
									Nullable:    true,
									Description: "The transformation function. A JavaScript or Python function that, given an event, returns the values needed to send the event to the app.",
								},
							}),
							Placeholder: `...`,
							Nullable:    true,
							Description: "This mapping or function is responsible for transforming unified users into the necessary values for sending the event to the app.\n\n" +
								"If there is no mapping, it is null. Otherwise one of either a mapping or a function is present, but not both. The one that is not present is null.",
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
