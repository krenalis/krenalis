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
		UpdateRequired: true,
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
	enabledParameter := types.Property{
		Name:        "enabled",
		Type:        types.Boolean(),
		Placeholder: "true",
		Description: "Indicate if the connection is enabled.",
	}
	strategyParameter := types.Property{
		Name:        "strategy",
		Type:        types.Text().WithValues("AB-C", "ABC", "A-B-C", "AC-B"),
		Placeholder: `"AB-C"`,
		Nullable:    true,
		Description: `The [strategy](/identity-resolution/anonymous-users-strategies) for anonymous users. ` +
			`It is required and can only be provided for source mobile and website connections.`,
	}
	sendingModeParameter := types.Property{
		Name:        "sendingMode",
		Type:        types.Text().WithValues("Cloud", "Device", "Combined"),
		Placeholder: `"Cloud"`,
		Nullable:    true,
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

	getReturnsParameters := []types.Property{
		idParameter,
		nameParameter,
		{
			Name:        "type",
			Type:        types.Text().WithValues("App", "Database", "FileStorage", "Mobile", "Server", "Stream", "Website"),
			Placeholder: "Website",
			Description: "The type of the connection's connector.",
		},
		roleParameter,
		enabledParameter,
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
			Description: "The connection's healthy.",
		},
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "connections",
		Name:        "Connections",
		Description: "A connection enables Meergo to retrieve customer and event data from an external source location or send them to an external destination location.",
		Endpoints: []*Endpoint{
			{
				Name:        "Create a connection",
				Description: "Create a new connection.",
				Method:      POST,
				URL:         "/v0/connections",
				Parameters: []types.Property{
					nameParameter,
					roleParameter,
					enabledParameter,
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
					{404, NotFound, ConnectorNotExist},
					{422, LinkedConnectionNotExist, "linked connection does not exist"},
					{422, InvalidSettings, "settings are not valid"},
				},
			},
			{
				Name:        "Get an OAuth token",
				Description: "Retrieves an OAuth token that can be used to create a connection that requires OAuth authentication.",
				Method:      GET,
				URL:         "/v0/connections/oauth",
				Parameters: []types.Property{
					{
						Name:           "oauthCode",
						Type:           types.Text(),
						CreateRequired: true,
						Description:    "The OAuth authorization code to complete the authentication process.",
					},
					{
						Name:           "redirectURI",
						Type:           types.Text(),
						CreateRequired: true,
						Description:    "The URI where the user will be redirected after authorization.",
					},
					{
						Name:           "connector",
						Type:           types.Text(),
						CreateRequired: true,
						Description:    "The name of the connector for which the connection will be created.",
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
				Description: "Update a connection.",
				Method:      PUT,
				URL:         "/v0/connections/:id",
				Parameters: []types.Property{
					idParameter,
					nameParameter,
					enabledParameter,
					strategyParameter,
					sendingModeParameter,
					websiteHostParameter,
				},
				Errors: []Error{
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
							Type:        types.Array(types.Object(getReturnsParameters)),
						},
					},
				},
			},
			{
				Name:        "Get a connection",
				Description: "Get a connection.",
				Method:      GET,
				URL:         "/v0/connections/:id",
				Parameters: []types.Property{
					idParameter,
				},
				Response: &Response{
					Parameters: getReturnsParameters,
				},
				Errors: []Error{
					{404, NotFound, "connection does not exist"},
				},
			},
			{
				Name:        "Delete a connection",
				Description: "Delete a connection.",
				Method:      DELETE,
				URL:         "/v0/connections/:id",
				Parameters: []types.Property{
					idParameter,
				},
				Errors: []Error{
					{404, NotFound, "connection does not exist"},
				},
			},
		},
	})

}
