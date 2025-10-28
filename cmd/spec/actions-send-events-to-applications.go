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
		Prefilled:      `"Mixpanel"`,
		Description:    "The action's name.",
	}
	filterParameter := types.Property{
		Name:      "filter",
		Type:      filterType,
		Nullable:  true,
		Prefilled: `{ "logical": "and", "conditions": [ { "property": "type", "operator": "is", "values": [ "track" ] } ] }`,
		Description: "The filter applied to the events. If it's not null, only the events that match the filter will be sent to the destination.\n\n" +
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
				Description:    "The transformation mapping. Each key represents a property path in the event type schema, and its corresponding value is an expression. This expression can reference property paths from the event schema.",
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
						Description:    "The property paths that will be passed to the function. If the function does not depend on the event to produce its response, specify an empty array.",
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
				Description:    "The transformation function. A JavaScript or Python function that, given an event, returns the values needed to send the event to the API.",
			},
		}),
		Prefilled:      `...`,
		Nullable:       true,
		CreateRequired: true,
		UpdateRequired: true,
		Description: "This mapping or function is responsible for transforming events into the values required for sending the event to the API.\n\n" +
			"If the event type's schema requires a specific property, you should provide a transformation that returns a value for this property.\n" +
			"If a mapping or function is provided (not null), only one of them should be specified. The other must either be absent or set to null.",
	}
	outSchemaParameter := types.Property{
		Name:           "outSchema",
		Type:           types.Parameter("schema"),
		Prefilled:      `{...}`,
		UpdateRequired: true,
		Nullable:       true,
		Description: "The schema for the output properties of the transformation. It is required and must not be null if a transformation is present.\n\n" +
			"It should be a subset of the schema of the passed event type.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "actions/send-events-to-applications",
		Name: "Send events to applications",
		Description: "This type of action sends the received events to application APIs.\n\n" +
			"It operates on a destination **API connection** that supports events.",
		Endpoints: []*Endpoint{
			{
				Name:        "Create action",
				Description: "Create a destination action that sends events to an API.",
				Method:      POST,
				URL:         "/v1/actions",
				Parameters: []types.Property{
					nameParameter,
					{
						Name:           "connection",
						Type:           types.Int(32),
						CreateRequired: true,
						Prefilled:      "753166510",
						Description:    "The ID of the connection to which the events will be sent. It must be a destination API that supports events.",
					},
					{
						Name:           "target",
						Type:           types.Text().WithValues("Event"),
						CreateRequired: true,
						Prefilled:      `"Event"`,
						Description:    "The entity on which the action operates, which must be `\"Event\"` in order to create an action that sends events.",
					},
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Prefilled:   "true",
						Description: "Indicates if the action is enabled once created.",
					},
					{
						Name:           "eventType",
						Type:           types.Text().WithCharLen(100),
						CreateRequired: true,
						Prefilled:      `"send_add_to_cart"`,
						Description: "The action's event type.\n\n" +
							"This should be the ID of one of the event types supported by the connection, which can be retrieved with the [`/connections/:id`](connections#get-connection) method.",
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
					{422, EventTypeNotExist, "connection does not have event type"},
					{422, ConnectorNotExist, "connector does not exist"},
					{422, UnsupportedLanguage, "transformation language is not supported"},
				},
			},
			{
				Name:        "Update action",
				Description: "Update a destination action that sends events to an API.",
				Method:      PUT,
				URL:         "/v1/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Prefilled:      "705981339",
						Description:    "The ID of the destination API action.",
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
					{422, UnsupportedLanguage, "transformation language is not supported"},
				},
			},
			{
				Name:        "Get action",
				Description: "Get a destination action that sends events to an API.",
				Method:      GET,
				URL:         "/v1/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Prefilled:      "705981339",
						Description:    "The ID of the destination API event action.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Prefilled:   "705981339",
							Description: "The ID of the destination API event action.",
						},
						nameParameter,
						{
							Name:        "connector",
							Type:        types.Text(),
							Prefilled:   `"klaviyo"`,
							Description: "The code of the connection's connector.",
						},
						{
							Name:        "connectorType",
							Type:        types.Text().WithValues("API", "Database", "FileStorage", "MessageBroker", "SDK", "Webhook"),
							Prefilled:   `"API"`,
							Description: "The type of the connection's connector. It is always `\"API\"` when the action sends events to an API.",
						},
						{
							Name:        "connection",
							Type:        types.Int(32),
							Prefilled:   "1371036433",
							Description: "The ID of the connection to which the events will be sent. It is a destination API that supports events.",
						},
						{
							Name:        "connectionRole",
							Type:        types.Text().WithValues("Source", "Destination"),
							Prefilled:   `"Destination"`,
							Description: "The role of the action's connection. It is always `\"Destination\"` when the action sends events to an API.",
						},
						{
							Name:        "target",
							Type:        types.Text().WithValues("User", "Event"),
							Prefilled:   `"Event"`,
							Description: "The entity on which the action operates. It is always `\"Event\"` when the action sends events to an API.",
						},
						{
							Name:        "enabled",
							Type:        types.Boolean(),
							Prefilled:   "true",
							Description: "Indicates if the action is enabled.",
						},
						{
							Name:           "eventType",
							Type:           types.Text().WithCharLen(100),
							CreateRequired: true,
							Prefilled:      `send_add_to_cart`,
							Description:    "The action's event type.",
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
									Description: "The transformation mapping. Each key represents a property path in the event type schema, and its corresponding value is an expression. This expression can reference property paths from the event schema.",
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
											Description: "The property paths that will be passed to the function. If empty, the function does not rely on the event to generate the response.",
										},
										{
											Name:        "outPaths",
											Type:        types.Array(types.Text()),
											Prefilled:   `[ "first_name", "last_name" ]`,
											Description: "The paths of the properties that may be returned by the function. It contains at least one property path.",
										},
									}),
									Nullable:    true,
									Description: "The transformation function. A JavaScript or Python function that, given an event, returns the values needed to send the event to the API.",
								},
							}),
							Prefilled: `...`,
							Nullable:  true,
							Description: "This mapping or function is responsible for transforming events into the values required for sending them to the API.\n\n" +
								"If there is no mapping, it is null. Otherwise one of either a mapping or a function is present, but not both. The one that is not present is null.",
						},
						{
							Name:      "inSchema",
							Type:      types.Parameter("schema"),
							Prefilled: `{...}`,
							Description: "The schema for the properties used in the filter and the input properties in the transformation.\n\n" +
								"When sending events to apps, this is the event schema.",
						},
						{
							Name:        "outSchema",
							Type:        types.Parameter("schema"),
							Nullable:    true,
							Prefilled:   `{...}`,
							Description: "The schema for the output properties of the transformation. It is null if no transformation is present.",
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
