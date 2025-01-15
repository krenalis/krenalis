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
		Description:    "Indicates if the connection is a source or a destination.",
	}
	strategyParameter := types.Property{
		Name:           "strategy",
		Type:           types.Text().WithValues("AB-C", "ABC", "A-B-C", "AC-B"),
		Placeholder:    `"AB-C"`,
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
		Description: `The mode for sending events. It is required and can only be provided with destination app connections that support it. ` +
			`In this case, it must be one of the sending modes supported by the app.`,
	}
	websiteHostParameter := types.Property{
		Name:        "websiteHost",
		Type:        types.Text(),
		Nullable:    true,
		Placeholder: "www.example.com",
		Description: "The host of the website. It is used for documentation purposes only, with no functional impact.",
	}
	linkedConnectionsParameter := types.Property{
		Name:        "linkedConnections",
		Type:        types.Array(types.Int(32)),
		Nullable:    true,
		Placeholder: "null",
		Description: "The connections (IDs) to which to send or from which to receive events.\n\n" +
			"For source mobile, server, or website connections, linked connections are the app connections to which the received events are sent. " +
			"On the other hand, for destination app connections, linked connections are the source mobile, website, and server connections from which events are received.\n\n" +
			"If it is null, there are no linked connections, or the connection is not one of the mentioned types.",
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
			Name:        "type",
			Type:        types.Text().WithValues("App", "Database", "FileStorage", "Mobile", "Server", "Stream", "Website"),
			Placeholder: "Website",
			Description: "The type of the connection's connector.",
		},
		roleParameter,
		{
			Name:        "connector",
			Type:        types.Text(),
			Placeholder: `"WebSite"`,
			Description: "The name of the connection's [connector](/connectors/).",
		},
		{
			Name:        "strategy",
			Type:        types.Text().WithValues("AB-C", "ABC", "A-B-C", "AC-B"),
			Placeholder: `"AB-C"`,
			Nullable:    true,
			Description: "The [strategy](/identity-resolution/anonymous-users-strategies) for anonymous users. " +
				"It is `null` if the connection is not a mobile or website source connection.",
		},
		{
			Name:        "sendingMode",
			Type:        types.Text().WithValues("Cloud", "Device", "Combined"),
			Placeholder: `"Cloud"`,
			Nullable:    true,
			Description: "The mode for sending events. It is null if the connection is not a destination app connection that supports sending mode.",
		},
		{
			Name:        "websiteHost",
			Type:        types.Text(),
			Nullable:    true,
			Placeholder: `"www.example.com"`,
			Description: "The host of the website. It is null if the connection is not a website connection. It is used for documentation purposes only, with no functional impact.",
		},
		linkedConnectionsParameter,
		{
			Name:        "hasSettings",
			Type:        types.Boolean(),
			Placeholder: `true`,
			Description: "It indicates if the connection has settings.",
		},
		{
			Name:        "actionsCount",
			Type:        types.Int(32),
			Placeholder: `3`,
			Description: "The total number of actions of the connection.",
		},
		{
			Name:        "health",
			Type:        types.Text().WithValues("Healthy", "NoRecentData", "RecentError"),
			Placeholder: `"Healthy"`,
			Description: "The connection's health.",
		},
	}

	getReturnsConnection := append(listReturnsConnection, types.Property{
		Name: "eventTypes",
		Type: types.Array(types.Object([]types.Property{
			{
				Name:        "id",
				Type:        types.Text(),
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
		Nullable:    true,
		Description: "The event types of the connection.\n\nIt has a null value if the connection is not a destination connection of type app.",
	})

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "connections",
		Name:        "Connections",
		Description: "A connection enables Meergo to retrieve customer and event data from an external source location or send them to an external destination location.",
		Endpoints: []*Endpoint{
			{
				Name: "Create a connection",
				Description: "Creates a new connection.\n\n" +
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
				Name:        "Update a connection",
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
				Name:        "Get a connection",
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
				Name:        "Delete a connection",
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
