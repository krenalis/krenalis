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

	filterParameter := types.Property{
		Name:        "filter",
		Type:        filterType,
		Nullable:    true,
		Placeholder: `{ "logical": "and", "conditions": [ { "property": "type", "operator": "is", "values": [ "track" ] } ] }`,
		Description: "The filter applied to the events. If it's not null, only the events that match the filter will be sent to the destination.\n\n" +
			"See the [filters documentation](/filters) for more details.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "destination-actions-on-events",
		Name:        "Destination actions on events",
		Description: "A destination action on events is an action that streams events received from a source connection to an app in real time.",
		Endpoints: []*Endpoint{
			{
				Name:        "Create a destination action on events",
				Description: "Create a new destination action on events.",
				Method:      POST,
				URL:         "/v0/actions",
				Parameters: []types.Property{
					{
						Name:           "eventType",
						Type:           types.Text(),
						CreateRequired: true,
						Placeholder:    `"send_add_to_cart"`,
						Description: "The action's event type.\n\n" +
							" It must be one of the event types supported by the connection of the action.",
					},
					{
						Name:           "name",
						Type:           types.Text().WithCharLen(60),
						CreateRequired: true,
						Placeholder:    `"Mixpanel"`,
						Description:    "The action's name.",
					},
					{
						Name:           "connection",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "753166510",
						Description:    "The connection for which the action should be created. It should be a destination app connection that support events.",
					},
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicate if the action is enabled once created.",
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
				Name:        "Update a destination action on events",
				Description: "Update a destination action on events.",
				Method:      PUT,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the destination action on event.",
					},
					{
						Name:           "name",
						Type:           types.Text().WithCharLen(60),
						CreateRequired: true,
						Placeholder:    `"Mixpanel"`,
						Description:    "The action's name.",
					},
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicates if the action is enable.",
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
				Name:        "Get a destination action on events",
				Description: "Get a destination action on events.",
				Method:      GET,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						Placeholder:    "705981339",
						Description:    "The ID of the destination action on event.",
						CreateRequired: true,
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Placeholder: "705981339",
							Description: "The ID of the destination action.",
						},
						{
							Name:        "connection",
							Type:        types.Int(32),
							Placeholder: "1371036433",
							Description: "The ID of the action's connection.",
						},
						{
							Name:        "name",
							Type:        types.Text().WithCharLen(60),
							Placeholder: `"Mixpanel"`,
							Description: "The action's name.",
						},
						{
							Name:        "enabled",
							Type:        types.Boolean(),
							Placeholder: "true",
							Description: "Indicates if the action is enable.",
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
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
				},
			},
			{
				Name:        "Delete a destination action on events",
				Description: "Delete a destination action on events.",
				Method:      DELETE,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the destination action on event.",
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
