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

	getReturnsParameters := []types.Property{
		{
			Name:        "name",
			Type:        types.Text(),
			Placeholder: `"HubSpot"`,
			Description: "The connector's name.",
		},
		{
			Name:        "sourceDescription",
			Type:        types.Text(),
			Placeholder: `"import contacts as users and companies as groups from HubSpot"`,
			Description: `A brief description of the connector when it is used as a source. It complete the sentence "Add an action to ...".`,
		},
		{
			Name:        "destinationDescription",
			Type:        types.Text(),
			Placeholder: `"export users as contacts and groups as companies to HubSpot"`,
			Description: `A brief description of the connector when it is used as a destination. It should complete the sentence "Add an action to ...".`,
		},
		{
			Name:        "termForUsers",
			Type:        types.Text(),
			Placeholder: `"contacts"`,
			Description: `The term used by an app to indicate the users. For example "clients", "customers" or "users". ` +
				"It will be empty if the connector is not an app or if the app does not handle users.",
		},
		{
			Name:        "termForGroups",
			Type:        types.Text(),
			Placeholder: `"companies"`,
			Description: `The term used by an app to indicate the groups. For example "organizations", "teams", or "groups".` +
				"\n\nIt will be empty if the connector is not an app or if the app does not handle groups.",
		},
		{
			Name:        "type",
			Type:        types.Text().WithValues("App", "Database", "File", "FileStorage", "Mobile", "Server", "Stream", "Website"),
			Placeholder: `"App"`,
			Description: "The type of connector.",
		},
		{
			Name:        "targets",
			Type:        types.Array(types.Text().WithValues("Events", "Users", "Groups")),
			Placeholder: `[ "Users", "Groups" ]`,
			Description: "The targets supported by the app connector. It includes one or more of the following: `\"Events\"`, `\"Users\"`, and `\"Groups\"`.\n\n" +
				"It will be empty if the connector is not an app.",
		},
		{
			Name:        "sendingMode",
			Type:        types.Text().WithValues("Cloud", "Device", "Combined"),
			Nullable:    true,
			Placeholder: `null`,
			Description: "The mode used by app connectors to dispatch the events to the app, if the app supports events. It is empty is the connector is not an app or it does not handle events.",
		},
		{
			Name:        "hasSheets",
			Type:        types.Boolean(),
			Placeholder: `false`,
			Description: "It indicates, for file connectors, if it supports sheets. It is false if the connector is not a file connector or does not support sheets.",
		},
		{
			Name:        "hasSettings",
			Type:        types.Boolean(),
			Placeholder: `true`,
			Description: "It indicates if the connector has settings.",
		},
		{
			Name:        "identityIDLabel",
			Type:        types.Text(),
			Placeholder: `"HubSpot ID"`,
			Description: "The descriptive name of the identifier used by the app to identify a user. For example \"ID\", \"User ID\", or \"HubSpot ID\".\n\nIt is empty if the connector is not an app.",
		},
		{
			Name:        "icon",
			Type:        types.Text(),
			Placeholder: `"<svg icon>"`,
			Description: "The icon in SVG format representing the connector, minimized for embedding in an HTML page.\n\nIt is empty if the connector does not have an icon.",
		},
		{
			Name:        "fileExtension",
			Type:        types.Text(),
			Placeholder: `""`,
			Description: "The main extension of the file type that the connector reads and writes. It is used as a placeholder in the input field, where the user specifies the file name to read or write.\n\n" +
				"It is empty if the connector is not a file connector.",
		},
		{
			Name:        "sampleQuery",
			Type:        types.Text(),
			Placeholder: `""`,
			Description: "The sample query displayed in the query editor when creating a new database source action.\n\nIt is empty if the connector is not a database connector.",
		},
		{
			Name:        "webhooksPer",
			Type:        types.Text().WithValues("None", "Account", "Connection", "Connector"),
			Placeholder: `"None"`,
			Description: "\"Indicates, for app connectors supporting webhooks, whether webhooks are per account, connection, or connector.\n\n" +
				"It is `\"None\"` if the connector is not an app or does not support webhooks.",
		},
		{
			Name:        "oAuth",
			Type:        types.Boolean(),
			Placeholder: `true`,
			Description: "Indicates, for app connections, whether it supports authentication with OAuth 2.0. It is false if the connector is not an app or does not support OAuth.",
		},
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "connectors",
		Name:        "Connectors",
		Description: "Connectors allows to instantiate [connections](connections) to interface Meergo with external data.",
		Endpoints: []*Endpoint{
			{
				Name:        "List all connectors",
				Description: "Returns the connectors, sorted by name.",
				Method:      GET,
				URL:         "/v0/connectors",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "connectors",
							Placeholder: "...",
							Type:        types.Array(types.Object(getReturnsParameters)),
						},
					},
				},
			},
			{
				Name:        "Get a connector",
				Description: "Get a connector by name.",
				Method:      GET,
				URL:         "/v0/connectors/:name",
				Parameters: []types.Property{
					{
						Name:        "name",
						Type:        types.Text(),
						Placeholder: `"HubSpot"`,
						Description: "The connector's name.",
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
				Name:        "Get OAuth consent URL",
				Description: "Gets the URL for an app connector that directs to the consent page of the app's OAuth 2.0 provider.",
				Method:      GET,
				URL:         "/v0/connectors/:name/oauth",
				Parameters: []types.Property{
					{
						Name:           "name",
						Type:           types.Text(),
						CreateRequired: true,
						Placeholder:    `"HubSpot"`,
						Description:    "The connector's name. It must be an app connector that supports authorization with OAuth.",
					},
					{
						Name:           "role",
						Type:           types.Text().WithValues("Source", "Destination"),
						CreateRequired: true,
						Placeholder:    `"Source"`,
						Description:    "The role for which to request authorization.",
					},
					{
						Name:           "redirecturi",
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
							Description: "The OAuth consent URL that directs to the consent page of the app's OAuth 2.0 provider. This page requests explicit permissions for the scopes required by the role.",
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
