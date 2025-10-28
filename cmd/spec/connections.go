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

	idParameter := types.Property{
		Name:           "id",
		Type:           types.Int(32),
		CreateRequired: true,
		Prefilled:      "1371036433",
		Description:    "The ID of the connection.",
	}
	nameParameter := types.Property{
		Name:           "name",
		Type:           types.Text().WithCharLen(60),
		CreateRequired: true,
		Prefilled:      `"Example.com Website"`,
		Description:    "The connection's name.",
	}
	roleParameter := types.Property{
		Name:           "role",
		Type:           types.Text().WithValues("Source", "Destination"),
		CreateRequired: true,
		Prefilled:      `"Source"`,
		Description:    "Indicates if the connection is a data source or a data destination.",
	}
	strategyParameter := types.Property{
		Name:           "strategy",
		Type:           types.Text().WithValues("Conversion", "Fusion", "Isolation", "Preservation"),
		Prefilled:      `"Conversion"`,
		UpdateRequired: true,
		Nullable:       true,
		Description: `The [strategy](/identity-resolution/anonymous-users-strategies) for anonymous users. ` +
			`It is required and can only be provided for source SDK connections that use them.`,
	}
	sendingModeParameter := types.Property{
		Name:           "sendingMode",
		Type:           types.Text().WithValues("Client", "Server", "ClientAndServer"),
		Prefilled:      `"Server"`,
		UpdateRequired: true,
		Nullable:       true,
		Description: "The mode for sending events. It can be one of the sending modes supported by the API.\n\n" +
			"It is required and can only be provided with destination API connections that support it.",
	}
	linkedConnectionsParameter := types.Property{
		Name:      "linkedConnections",
		Type:      types.Array(types.Int(32)),
		Nullable:  true,
		Prefilled: "null",
		Description: "The IDs of the connections to which events are sent or from which events are received.\n\n" +
			"For source connections (SDK and webhook), the linked connections are the destination API connections to which received events are forwarded. " +
			"Conversely, for destination API connections, the linked connections are the source SDK and webhook connections from which events are received.\n\n" +
			"If this field is null, it means there are no linked connections or the connection type does not match any of the types mentioned above.",
	}
	settingsParameter := types.Property{
		Name:      "settings",
		Type:      types.Parameter("Settings"),
		Nullable:  true,
		Prefilled: "null",
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
			Prefilled:   `"javascript"`,
			Description: "The code of the connection's [connector](/connectors/).",
		},
		{
			Name:        "connectorType",
			Type:        types.Text().WithValues("API", "Database", "FileStorage", "MessageBroker", "SDK", "Webhook"),
			Prefilled:   `"SDK"`,
			Description: "The type of the connection's connector.",
		},
		roleParameter,
		{
			Name:      "strategy",
			Type:      types.Text().WithValues("Conversion", "Fusion", "Isolation", "Preservation"),
			Prefilled: `"Conversion"`,
			Nullable:  true,
			Description: "The [strategy](/identity-resolution/anonymous-users-strategies) for anonymous users. " +
				"It is `null` if the connection is not an SDK source connection, or if the connection's connector does not support strategies.",
		},
		{
			Name:        "sendingMode",
			Type:        types.Text().WithValues("Client", "Server", "ClientAndServer"),
			Prefilled:   `"Server"`,
			Nullable:    true,
			Description: "The mode for sending events. It is null if the connection is not a destination API that supports sending mode.",
		},
		linkedConnectionsParameter,
		{
			Name:        "actionsCount",
			Type:        types.Int(32),
			Prefilled:   `3`,
			Description: "The total number of actions of the connection.",
		},
		/* See issue https://github.com/meergo/meergo/issues/1255.
		{
			Name:        "health",
			Type:        types.Text().WithValues("Healthy", "NoRecentData", "RecentError"),
			Prefilled: `"Healthy"`,
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
				{
					Name:        "filter",
					Type:        types.Text(),
					Description: "The recommended default filter for actions that send events to APIs using this event type.",
				},
			})),
			Nullable: true,
			Description: "The type of events that can be sent to a destination API connection.\n\n" +
				"It has a null value if the connection is not a destination connection of type API or if it does not support events.\n\n" +
				"Once you have retrieved an event type and its ID, you can obtain its schema through the method [`/connections/:id/schemas/event/:type`](api/connections/apps#get-event-schema).",
		},
	)

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "connections",
		Name: "Connections",
		Description: "Connections serve as a channel between workspaces and external sources or destinations, such as websites, applications, databases, file storage, and files.\n\n" +
			"[Actions](actions) then allow you to perform operations on these connections.",
		Endpoints: []*Endpoint{
			{
				Name: "Create connection",
				Description: "Creates a connection.\n\n" +
					"For connectors that require authorization, follow these steps to create the connection:\n\n" +
					"1. [Get the auth URL](connections#get-auth-url) and redirect the user to it.\n" +
					"2. Once the user grants permission, [retrieve the auth token](connections#retrieve-auth-token).\n" +
					"3. Create the connection by passing the auth token as the `authToken` argument.",
				Method: POST,
				URL:    "/v1/connections",
				Parameters: []types.Property{
					nameParameter,
					roleParameter,
					{
						Name:           "connector",
						Type:           types.Text(),
						CreateRequired: true,
						Prefilled:      `"javascript"`,
						Description: "The code of the [connector](/connectors/) for which to create the connection. " +
							"It can be an API, database, file storage, SDK, or webhook connector, " +
							"but cannot be a file connector or a message broker connector.\n\nMessage broker connectors will be available soon.",
					},
					strategyParameter,
					sendingModeParameter,
					linkedConnectionsParameter,
					settingsParameter,
					{
						Name:           "authToken",
						Type:           types.Text(),
						Prefilled:      `"eyJz93a...k4F5sdtW"`,
						UpdateRequired: true,
						Description:    "The authorization token. Is required if the connector requires authorization.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Prefilled:   "1371036433",
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
				Description: "Gets the URL for an API connector that directs to the authorization page of the API.\n\n",
				Method:      GET,
				URL:         "/v1/connections/auth-url",
				Parameters: []types.Property{
					{
						Name:           "connector",
						Type:           types.Text(),
						CreateRequired: true,
						Prefilled:      `hubspot`,
						Description:    "The connector's code. It must be an API connector that requires authorization.",
					},
					{
						Name:           "role",
						Type:           types.Text().WithValues("Source", "Destination"),
						CreateRequired: true,
						Prefilled:      `Source`,
						Description:    "The role for which to request authorization.",
					},
					{
						Name:           "redirectURI",
						Type:           types.Text(),
						CreateRequired: true,
						Prefilled:      `https://example.com/oauth`,
						Description:    "The URL to which redirect after granting permissions.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "url",
							Type:        types.Text(),
							Prefilled:   `"https://app-eu1.hubspot.com/oauth/authorize"`,
							Description: "The authorization URL that directs to the consent page of the API. This page requests explicit permissions for the role.",
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
				URL:         "/v1/connections/auth-token",
				Parameters: []types.Property{
					{
						Name:           "connector",
						Type:           types.Text(),
						CreateRequired: true,
						Prefilled:      `HubSpot`,
						Description:    "The name of the connector for which the connection will be created.",
					},
					{
						Name:           "redirectURI",
						Type:           types.Text(),
						CreateRequired: true,
						Prefilled:      `https://example.com/oauth`,
						Description:    "The URI where the user will be redirected after authorization.",
					},
					{
						Name:           "oauthCode",
						Type:           types.Text(),
						CreateRequired: true,
						Prefilled:      `8aa112345`,
						Description:    "The authorization code to complete the authorization process.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "token",
							Type:        types.Text(),
							Prefilled:   `"jK64q6Vu0DMoqpXm0M+/6qbZagaVipMsRZ"`,
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
				URL:         "/v1/connections/:id",
				Parameters: []types.Property{
					idParameter,
					nameParameter,
					strategyParameter,
					sendingModeParameter,
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
				URL:         "/v1/connections",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:      "connections",
							Prefilled: "...",
							Type:      types.Array(types.Object(listReturnsConnection)),
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
				URL:         "/v1/connections/:id",
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
				Name: "Retrieve user identities",
				Description: "Retrieves user identities from the workspace's data warehouse, exactly as they were imported from a connection, before being unified with identities from other connections.\n\n" +
					"Identities are sorted by last change time, in descending order, so the most recently changed identities are returned first.",
				Method: GET,
				URL:    "/v1/connections/:id/identities",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						Prefilled:      "1371036433",
						CreateRequired: true,
						Description:    "The ID of the connection from which the user identities were imported. It must be a source connection.",
					},
					{
						Name:        "first",
						Type:        types.Int(32),
						Prefilled:   `0`,
						Description: "The number of user identities to skip before starting to return results. The default value is 0.",
					},
					{
						Name:           "limit",
						Type:           types.Int(32).WithIntRange(1, 1000),
						CreateRequired: true,
						Prefilled:      `1000`,
						Description:    "The maximum number of user identities to return. The value must be within the range [1, 1000].",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "identities",
							Type:        types.Array(types.Map(types.JSON())),
							Prefilled:   `[ { ... } ]`,
							Description: "The connection's user identities.",
						},
						{
							Name:        "total",
							Type:        types.Int(32),
							Prefilled:   `23`,
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
				URL:         "/v1/connections/:id",
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
