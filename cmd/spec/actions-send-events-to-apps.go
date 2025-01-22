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
					{
						Name:           "eventType",
						Type:           types.Text(),
						CreateRequired: true,
						Placeholder:    `"send_add_to_cart"`,
						Description: "The action's event type.\n\n" +
							" It must be one of the event types supported by the connection of the action.",
					},
					filterParameter,
					{
						Name:        "outSchema",
						Type:        types.Parameter("schema"),
						Nullable:    true,
						Placeholder: `{...}`,
					},
					{
						Name:        "transformation",
						Type:        types.Parameter("transformation"),
						Nullable:    true,
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
					{
						Name:        "outSchema",
						Type:        types.Parameter("schema"),
						Nullable:    true,
						Placeholder: `{...}`,
					},
					{
						Name:        "transformation",
						Type:        types.Parameter("transformation"),
						Nullable:    true,
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
							Placeholder: `"Klavyio"`,
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
							Nullable:    true,
							Placeholder: `{...}`,
						},
						{
							Name:        "outSchema",
							Type:        types.Parameter("schema"),
							Nullable:    true,
							Placeholder: `{...}`,
						},
						{
							Name:        "transformation",
							Type:        types.Parameter("transformation"),
							Nullable:    true,
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
