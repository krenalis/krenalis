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

	idParameter := types.Property{
		Name:           "id",
		Type:           types.Int(32),
		CreateRequired: true,
		Placeholder:    "1371036433",
		Description:    "The ID of the connection.",
	}
	nameParameter := types.Property{
		Name:           "name",
		Type:           types.Text().WithCharLen(60),
		CreateRequired: true,
		Placeholder:    `"Example.com Website"`,
		Description:    "The connection's name.",
	}
	roleParameter := types.Property{
		Name:           "role",
		Type:           types.Text().WithValues("Source", "Destination"),
		CreateRequired: true,
		Placeholder:    `"Source"`,
		Description:    "Indicates if the connection is a data source or a data destination.",
	}
	strategyParameter := types.Property{
		Name:           "strategy",
		Type:           types.Text().WithValues("Conversion", "Fusion", "Isolation", "Preservation"),
		Placeholder:    `"Conversion"`,
		UpdateRequired: true,
		Nullable:       true,
		Description: `The [strategy](/identity-resolution/anonymous-users-strategies) for anonymous users. ` +
			`It is required and can only be provided for source mobile and website connections.`,
	}
	sendingModeParameter := types.Property{
		Name:           "sendingMode",
		Type:           types.Text().WithValues("Cloud", "Device", "Combined"),
		Placeholder:    `"Cloud"`,
		UpdateRequired: true,
		Nullable:       true,
		Description: "The mode for sending events. It can be one of the sending modes supported by the app.\n\n" +
			"It is required and can only be provided with destination app connections that supports it.",
	}
	websiteHostParameter := types.Property{
		Name:           "websiteHost",
		Type:           types.Text(),
		UpdateRequired: true,
		Nullable:       true,
		Placeholder:    "www.example.com",
		Description: "The host of the website. It is used for documentation purposes only, with no functional impact.\n\n" +
			"It is required and can only be provided with website connections.",
	}
	linkedConnectionsParameter := types.Property{
		Name:        "linkedConnections",
		Type:        types.Array(types.Int(32)),
		Nullable:    true,
		Placeholder: "null",
		Description: "The IDs of of the connections to which events are sent or from which events are received.\n\n" +
			"For source connections (website, mobile, server), the linked connections are the destination app connections where the received events are forwarded. " +
			"On the other hand, for destination app connections, the linked connections are the source website, mobile, or server connections from which events are received.\n\n" +
			"If this field is null, it means there are no linked connections, or the connection type does not match any of the types mentioned above.",
	}
	settingsParameter := types.Property{
		Name:        "settings",
		Type:        types.Parameter("Settings"),
		Nullable:    true,
		Placeholder: "null",
		Description: "The specific settings of the connection, which vary based on the connector specified in the `connector` field. " +
			"Please refer to the documentation for the [connector](/connectors/) to understand the available settings and their corresponding values.\n\n" +
			"If the connector does not require any settings, the `settings` field may be omitted or set to null.",
	}
	listReturnsConnection := []types.Property{
		idParameter,
		nameParameter,
		{
			Name:        "connector",
			Type:        types.Text(),
			Placeholder: `"WebSite"`,
			Description: "The name of the connection's [connector](/connectors/).",
		},
		{
			Name:        "connectorType",
			Type:        types.Text().WithValues("App", "Database", "FileStorage", "Mobile", "Server", "Stream", "Website"),
			Placeholder: `"Website"`,
			Description: "The type of the connection's connector.",
		},
		roleParameter,
		{
			Name:        "strategy",
			Type:        types.Text().WithValues("Conversion", "Fusion", "Isolation", "Preservation"),
			Placeholder: `"Conversion"`,
			Nullable:    true,
			Description: "The [strategy](/identity-resolution/anonymous-users-strategies) for anonymous users. " +
				"It is `null` if the connection is not a mobile or website source connection.",
		},
		{
			Name:        "sendingMode",
			Type:        types.Text().WithValues("Cloud", "Device", "Combined"),
			Placeholder: `"Cloud"`,
			Nullable:    true,
			Description: "The mode for sending events. It is null if the connection is not a destination app that supports sending mode.",
		},
		{
			Name:        "websiteHost",
			Type:        types.Text(),
			Nullable:    true,
			Placeholder: `"www.example.com"`,
			Description: "The host of the website. It is used for documentation purposes only, with no functional impact.\n\n" +
				"It is null if the connection is not a website.",
		},
		linkedConnectionsParameter,
		{
			Name:        "actionsCount",
			Type:        types.Int(32),
			Placeholder: `3`,
			Description: "The total number of actions of the connection.",
		},
		/* See issue https://github.com/meergo/meergo/issues/1255.
		{
			Name:        "health",
			Type:        types.Text().WithValues("Healthy", "NoRecentData", "RecentError"),
			Placeholder: `"Healthy"`,
			Description: "The connection's health.",
		},
		*/}
	getReturnsConnection := append(listReturnsConnection,
		types.Property{
			Name:        "actions",
			Type:        types.Array(types.Parameter("action")),
			Description: "The actions of the connection.",
		},
		types.Property{
			Name: "eventTypes",
			Type: types.Array(types.Object([]types.Property{
				{
					Name:        "id",
					Type:        types.Text().WithCharLen(100),
					Description: "The event type ID, which uniquely identifies it in the context of the connection.",
				},
				{
					Name:        "name",
					Type:        types.Text(),
					Description: "The name of the event type. Compared to the ID, the name is human-readable and suitable for display to a user, for example in an interface.",
				},
				{
					Name:        "description",
					Type:        types.Text(),
					Description: "The description of the event type.",
				},
			})),
			Nullable: true,
			Description: "The type of events that can be sent to a destination app connection.\n\n" +
				"It has a null value if the connection is not a destination connection of type app or if it does not support events.\n\n" +
				"Once you have retrieved an event type and its ID, you can obtain its schema through the method [`/connections/:id/schemas/event/:type`](connection-app#get-event-type-schema).",
		},
	)

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "connections",
		Name: "Connections",
		Description: "Connections serve as a channel between workspaces and external sources or destinations, such as applications, databases, file storage, websites, mobile apps, and servers.\n\n" +
			"[Actions](/api/actions) then allow you to perform operations on these connections.",
		Endpoints: []*Endpoint{
			{
				Name: "Create connection",
				Description: "Creates a connection.\n\n" +
					"For connectors that require authorization, follow these steps to create the connection:\n\n" +
					"1. [Get the auth URL](/api/connections#get-auth-url) and redirect the user to it.\n" +
					"2. Once the user grants permission, [retrieve the auth token](/api/connections#retrieve-auth-token).\n" +
					"3. Create the connection by passing the auth token as the `authToken` argument.",
				Method: POST,
				URL:    "/v0/connections",
				Parameters: []types.Property{
					nameParameter,
					roleParameter,
					{
						Name:           "connector",
						Type:           types.Text(),
						CreateRequired: true,
						Placeholder:    `"WebSite"`,
						Description: "The name of the [connector](/connectors/) for which to create the connection. " +
							"It can be an app, database, file storage, mobile, server, stream, or website connector, " +
							"but cannot be a file connector.",
					},
					strategyParameter,
					sendingModeParameter,
					websiteHostParameter,
					linkedConnectionsParameter,
					settingsParameter,
					{
						Name:           "authToken",
						Type:           types.Text(),
						Placeholder:    `"eyJz93a...k4F5sdtW"`,
						UpdateRequired: true,
						Description:    "The authorization token. Is required if the connector requires authorization.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Placeholder: "1371036433",
							Description: "The ID of the connection.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, ConnectorNotExist, "connector does not exist"},
					{422, LinkedConnectionNotExist, "linked connection does not exist"},
					{422, InvalidSettings, "settings are not valid"},
				},
			},
			{
				Name:        "Get auth URL",
				Description: "Gets the URL for an app connector that directs to the authorization page of the app.\n\n",
				Method:      GET,
				URL:         "/v0/connections/auth-url",
				Parameters: []types.Property{
					{
						Name:           "connector",
						Type:           types.Text(),
						CreateRequired: true,
						Placeholder:    `"HubSpot"`,
						Description:    "The connector's name. It must be an app connector that requires authorization.",
					},
					{
						Name:           "role",
						Type:           types.Text().WithValues("Source", "Destination"),
						CreateRequired: true,
						Placeholder:    `"Source"`,
						Description:    "The role for which to request authorization.",
					},
					{
						Name:           "redirectURI",
						Type:           types.Text(),
						CreateRequired: true,
						Placeholder:    `"https://example.com/oauth"`,
						Description:    "The URL to which redirect after granting permissions.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "url",
							Type:        types.Text(),
							Placeholder: `"https://app.hubspot.com/oauth/authorize"`,
							Description: "The authorization URL that directs to the consent page of the app. This page requests explicit permissions for the role.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "connector does not exist"},
				},
			},
			{
				Name:        "Retrieve auth token",
				Description: "Retrieves an authorization token that can be used to create a connection.",
				Method:      GET,
				URL:         "/v0/connections/auth-token",
				Parameters: []types.Property{
					{
						Name:           "connector",
						Type:           types.Text(),
						CreateRequired: true,
						Description:    "The name of the connector for which the connection will be created.",
					},
					{
						Name:           "redirectURI",
						Type:           types.Text(),
						CreateRequired: true,
						Description:    "The URI where the user will be redirected after authorization.",
					},
					{
						Name:           "oauthCode",
						Type:           types.Text(),
						CreateRequired: true,
						Description:    "The authorization code to complete the authorization process.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "token",
							Type:        types.Text(),
							Placeholder: `"jK64q6Vu0DMoqpXm0M+/6qbZagaVipMsRZ"`,
							Description: "The OAuth token that can be used to create a connection on the specified connector.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, ConnectorNotExist, "connector does not exist"},
				},
			},
			{
				Name:        "Update connection",
				Description: "Updates a connection.",
				Method:      PUT,
				URL:         "/v0/connections/:id",
				Parameters: []types.Property{
					idParameter,
					nameParameter,
					strategyParameter,
					sendingModeParameter,
					websiteHostParameter,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
				},
			},
			{
				Name:        "List all connections",
				Description: "Returns the workspace's connections, sorted by name.",
				Method:      GET,
				URL:         "/v0/connections",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "connections",
							Placeholder: "...",
							Type:        types.Array(types.Object(listReturnsConnection)),
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
			{
				Name:        "Get connection",
				Description: "Gets a connection.",
				Method:      GET,
				URL:         "/v0/connections/:id",
				Parameters: []types.Property{
					idParameter,
				},
				Response: &Response{
					Parameters: getReturnsConnection,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
				},
			},
			{
				Name:        "Retrieve user identities",
				Description: "Retrieves user identities from the workspace's data warehouse, exactly as they were imported from a connection, before being unified with identities from other connections.",
				Method:      GET,
				URL:         "/v0/connections/:id/identities",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						Placeholder:    "1371036433",
						CreateRequired: true,
						Description:    "The ID of the connection from which the user identities were imported. It must be a source connection.",
					},
					{
						Name:        "first",
						Type:        types.Int(32),
						Placeholder: `0`,
						Description: "The number of user identities to skip before starting to return results. The default value is 0.",
					},
					{
						Name:           "limit",
						Type:           types.Int(32).WithIntRange(1, 1000),
						CreateRequired: true,
						Placeholder:    `1000`,
						Description:    "The maximum number of user identities to return. The value must be within the range [1, 1000].",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "identities",
							Type:        types.Array(types.Map(types.JSON())),
							Placeholder: `[ { ... } ]`,
							Description: "The connection's user identities.",
						},
						{
							Name:        "total",
							Type:        types.Int(32),
							Placeholder: `23`,
							Description: "The estimated total number of user identities in the connection.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
				},
			},
			{
				Name:        "Delete connection",
				Description: "Deletes a connection.",
				Method:      DELETE,
				URL:         "/v0/connections/:id",
				Parameters: []types.Property{
					idParameter,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
				},
			},
		},
	})

}
