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

	getReturnsParameters := []types.Property{
		{
			Name:        "code",
			Type:        types.Text(),
			Prefilled:   `"hubspot"`,
			Description: "The connector's code. It contains only `[a-z0-9-]`.",
		},
		{
			Name:        "label",
			Type:        types.Text(),
			Prefilled:   `"HubSpot"`,
			Description: "The connector's label.",
		},
		{
			Name:        "type",
			Type:        types.Text().WithValues("App", "Database", "File", "FileStorage", "SDK", "Stream"),
			Prefilled:   `"App"`,
			Description: "The type of connector.",
		},
		{
			Name:        "categories",
			Type:        types.Array(types.Text()),
			Prefilled:   `[ "CRM" ]`,
			Description: "The categories of the connector. There is always at least one.",
		},
		{
			Name: "asSource",
			Type: types.Object([]types.Property{
				{
					Name:        "targets",
					Type:        types.Array(types.Text().WithValues("Event", "User")),
					Prefilled:   `[ "User" ]`,
					Description: "The targets supported by the connector when it is used as a source. It includes one or more of the following: `\"Event\"` and `\"User\"`",
				},
				{
					Name:        "hasSettings",
					Type:        types.Boolean(),
					Prefilled:   `true`,
					Description: "It indicates if the connector has settings when it is used as a source.",
				},
				{
					Name:        "sampleQuery",
					Type:        types.Text(),
					Prefilled:   `""`,
					Description: "The sample query displayed in the query editor when creating a new database source action.\n\nIt is empty if the connector is not a database connector.",
				},
				// TODO(marco): implement webhooks
				//{
				//	Name:        "webhooksPer",
				//	Type:        types.Text().WithValues("None", "Account", "Connection", "Connector"),
				//	Prefilled: `"None"`,
				//	Description: "Indicates, for app connectors supporting webhooks, whether webhooks are per account, connection, or connector.\n\n" +
				//		"It is `\"None\"` if the connector is not an app or does not support webhooks.",
				//},
				{
					Name:        "summary",
					Type:        types.Text(),
					Prefilled:   `"Import contacts as users and companies as groups from HubSpot"`,
					Description: `A brief description of the connector when it is used as a source.`,
				},
			}),
			Prefilled:   `...`,
			Nullable:    true,
			Description: `The characteristics of the connector when it is used as a data source. This will be null if the connector cannot function as a data source.`,
		},
		{
			Name: "asDestination",
			Type: types.Object([]types.Property{
				{
					Name:        "targets",
					Type:        types.Array(types.Text().WithValues("Event", "User")),
					Prefilled:   `[ "User" ]`,
					Description: "The targets supported by the connector when it is used as a destination. It includes one or more of the following: `\"Event\"`, and `\"User\"`",
				},
				{
					Name:        "hasSettings",
					Type:        types.Boolean(),
					Prefilled:   `true`,
					Description: "It indicates if the connector has settings when it is used as a destination.",
				},
				{
					Name:        "sendingMode",
					Type:        types.Text().WithValues("Client", "Server", "ClientAndServer"),
					Nullable:    true,
					Prefilled:   `null`,
					Description: "The mode used by app connectors to send the events to the app, if the app supports events. It is empty if the connector is not an app or it does not handle events.",
				},
				{
					Name:        "summary",
					Type:        types.Text(),
					Prefilled:   `"Export users as contacts and groups as companies to HubSpot"`,
					Description: `A brief description of the connector when it is used as a destination.`,
				},
			}),
			Prefilled:   `...`,
			Nullable:    true,
			Description: `The characteristics of the connector when it is used as a data destination. This will be null if the connector cannot function as a data destination.`,
		},
		{
			Name: "terms",
			Type: types.Object([]types.Property{
				{
					Name:      "user",
					Type:      types.Text(),
					Prefilled: `"contact"`,
					Description: `The term used by an app to indicate a single user. For example "client", "customer" or "user".` +
						"\n\nIt will be empty if the connector is not an app or if the app does not handle users.",
				},
				{
					Name:      "users",
					Type:      types.Text(),
					Prefilled: `"contacts"`,
					Description: `The term used by an app to indicate the users. For example "clients", "customers" or "users".` +
						"\n\nIt will be empty if the connector is not an app or if the app does not handle users.",
				},
				{
					Name:      "group",
					Type:      types.Text(),
					Prefilled: `"company"`,
					Description: `The term used by an app to indicate a single group. For example "organization", "team", or "group".` +
						"\n\nIt will be empty if the connector is not an app or if the app does not handle groups.",
				},
				{
					Name:      "groups",
					Type:      types.Text(),
					Prefilled: `"companies"`,
					Description: `The term used by an app to indicate the groups. For example "organizations", "teams", or "groups".` +
						"\n\nIt will be empty if the connector is not an app or if the app does not handle groups.",
				},
			}),
			Description: "Specific terms by which the connector refers to various entities, such as users and groups.",
		},
		{
			Name:        "hasSheets",
			Type:        types.Boolean(),
			Prefilled:   `false`,
			Description: "It indicates, for file connectors, if it supports sheets. It is false if the connector is not a file connector or does not support sheets.",
		},
		{
			Name:        "identityIDLabel",
			Type:        types.Text(),
			Prefilled:   `"HubSpot ID"`,
			Description: "The descriptive name of the identifier used by the app to identify a user. For example \"ID\", \"User ID\", or \"HubSpot ID\".\n\nIt is empty if the connector is not an app.",
		},
		{
			Name:      "fileExtension",
			Type:      types.Text(),
			Prefilled: `""`,
			Description: "The main extension of the file type that the connector reads and writes. It is used as a placeholder in the input field, where the user specifies the file name to read or write.\n\n" +
				"It is empty if the connector is not a file connector.",
		},
		{
			Name:        "requiresAuth",
			Type:        types.Boolean(),
			Prefilled:   `true`,
			Description: "Indicates whether an authorization is required to create a connection for this connector. It is false if the connector is not an app or does not require authorization.",
		},
		{
			Name:        "authConfigured",
			Type:        types.Boolean(),
			Prefilled:   `true`,
			Description: "Indicates whether the required OAuth credentials (client ID and client secret) have been provided for this connector. It is true only if the connector requires authorization and the necessary environment variables are set.",
		},
		{
			Name:        "icon",
			Type:        types.Text(),
			Prefilled:   `"<svg icon>"`,
			Description: "The icon in SVG format representing the connector, minimized for embedding in an HTML page.\n\nIt is empty if the connector does not have an icon.",
		},
		{
			Name:        "strategies",
			Type:        types.Boolean(),
			Prefilled:   `true`,
			Description: "Indicates whether the connector requires a strategy to be configured for user management.\n\nIf true, the strategy is mandatory and must be provided when creating connections for this connector, otherwise, if false, the strategy is not allowed for such connections.",
		},
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "connectors",
		Name:        "Connectors",
		Description: "Connectors allow you to instantiate [connections](connections) to interface Meergo with external data.",
		Endpoints: []*Endpoint{
			{
				Name:        "List all connectors",
				Description: "Returns the connectors, sorted by code.",
				Method:      GET,
				URL:         "/v1/connectors",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:      "connectors",
							Prefilled: "...",
							Type:      types.Array(types.Object(getReturnsParameters)),
						},
					},
				},
			},
			{
				Name:        "Get connector",
				Description: "Get a connector.",
				Method:      GET,
				URL:         "/v1/connectors/:code",
				Parameters: []types.Property{
					{
						Name:           "code",
						Type:           types.Text(),
						Prefilled:      `hubspot`,
						CreateRequired: true,
						Description:    "The connector's code.",
					},
				},
				Response: &Response{
					Parameters: getReturnsParameters,
				},
				Errors: []Error{
					{404, NotFound, "connector does not exist"},
				},
			},
			{
				Name:        "Get connector documentation",
				Description: "Get the documentation of a connector.",
				Method:      GET,
				URL:         "/v1/connectors/:code/documentation",
				Parameters: []types.Property{
					{
						Name:           "code",
						Type:           types.Text(),
						Prefilled:      `hubspot`,
						CreateRequired: true,
						Description:    "The connector's code.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name: "source",
							Type: types.Object([]types.Property{
								{
									Name:        "summary",
									Type:        types.Text(),
									Prefilled:   `"Export users as contacts and groups as companies to HubSpot"`,
									Description: `A brief description of the connector.`,
								},
								{
									Name:        "overview",
									Type:        types.Text(),
									Prefilled:   `"# HubSpot\nThe HubSpot data source allows..."`,
									Description: `A Markdown-formatted overview of the connector.`,
								},
							}),
							Prefilled:   "...",
							Description: "The documentation of the connector when it is used as a source.",
						},
						{
							Name: "destination",
							Type: types.Object([]types.Property{
								{
									Name:        "summary",
									Type:        types.Text(),
									Prefilled:   `"Export users as contacts and groups as companies to HubSpot"`,
									Description: `A brief description of the connector.`,
								},
								{
									Name:        "overview",
									Type:        types.Text(),
									Prefilled:   `"# HubSpot\nThe HubSpot data destination allows..."`,
									Description: `A Markdown-formatted overview of the connector.`,
								},
							}),
							Prefilled:   "...",
							Description: "The documentation of the connector when it is used as a destination.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "connector does not exist"},
				},
			},
		},
	})

}
