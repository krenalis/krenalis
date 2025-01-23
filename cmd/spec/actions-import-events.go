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
		Placeholder:    `"Site example.com"`,
		Description:    "The action's name.",
	}
	filterParameter := types.Property{
		Name:        "filter",
		Type:        filterType,
		Nullable:    true,
		Placeholder: `{ "logical": "and", "conditions": [ { "property": "type", "operator": "is", "values": [ "track" ] } ] }`,
		Description: "The filter applied to the events. If it's not null, only the events that match the filter will be imported.\n\n" +
			"See the [filters documentation](/filters) for more details.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "actions-import-events",
		Name: "Import events",
		Description: "This type of action imports events into the workspace’s data warehouse. " +
			"It operates on a website, mobile, or server connection.",
		Endpoints: []*Endpoint{
			{
				Name:        "Create action",
				Description: "Create a source action that imports events into the data warehouse.",
				Method:      POST,
				URL:         "/v0/actions",
				Parameters: []types.Property{
					nameParameter,
					{
						Name:           "connection",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "230527183",
						Description:    "The ID of the connection from which the events are received. It must be a website, mobile, or server.",
					},
					{
						Name:           "target",
						Type:           types.Text().WithValues("Events"),
						CreateRequired: true,
						Placeholder:    `"Events"`,
						Description:    "The entity on which the action operates, which must be `\"Events\"` in order to create an action that imports events.",
					},
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicate if the action is enabled once created.",
					},
					filterParameter,
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
				},
			},
			{
				Name:        "Update action",
				Description: "Update a source action that imports events into the data warehouse.",
				Method:      PUT,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source action on event.",
					},
					nameParameter,
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicates if the action is enabled. Use the [Set status](/api/actions#set-status) endpoint to change only the action's status.",
					},
					filterParameter,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
				},
			},
			{
				Name:        "Get action",
				Description: "Get a source action that imports events into the data warehouse.",
				Method:      GET,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source action on event.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Placeholder: "705981339",
							Description: "The ID of the source action on event.",
						},
						nameParameter,
						{
							Name:        "connector",
							Type:        types.Text(),
							Placeholder: `"Android"`,
							Description: "The name of the connection's connector.",
						},
						{
							Name:        "connectorType",
							Type:        types.Text().WithValues("App", "Database", "FileStorage", "Mobile", "Server", "Website"),
							Placeholder: `"Website"`,
							Description: "The type of the connection's connector. It is always `\"Mobile\"`, `\"Server\"`, or `\"Website\"` when the action imports events.",
						},
						{
							Name:        "connection",
							Type:        types.Int(32),
							Placeholder: "1371036433",
							Description: "The ID of the connection from which the events are received. It is a source website, mobile, or server.",
						},
						{
							Name:        "connectionRole",
							Type:        types.Text().WithValues("Source", "Destination"),
							Placeholder: `"Source"`,
							Description: "The role of the action's connection. It is always `\"Source\"` when the action imports events.",
						},
						{
							Name:        "target",
							Type:        types.Text().WithValues("Users", "Events"),
							Placeholder: `"Events"`,
							Description: "The entity on which the action operates. It is always `\"Events\"` when the action imports events.",
						},
						{
							Name:        "enabled",
							Type:        types.Boolean(),
							Placeholder: "true",
							Description: "Indicates if the action is enabled.",
						},
						filterParameter,
						{
							Name:        "inSchema",
							Type:        types.Parameter("schema"),
							Placeholder: `{...}`,
							Description: "The input schema. When importing events, it is the event schema.",
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
