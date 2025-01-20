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
		Placeholder:    `"HubSpot"`,
		Description:    "The action's name.",
	}

	filterParameter := types.Property{
		Name:        "filter",
		Type:        filterType,
		Nullable:    true,
		Placeholder: `{ "logical": "and", "conditions": [ { "property": "country", "operator": "is", "values": [ "US" ] } ] }`,
		Description: "The filter applied to the app users. If it's not null, only the users that match the filter will be included.\n\n" +
			"See the [filters documentation](/filters) for more details.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "actions-export-users-to-apps",
		Name: "Export users to apps",
		Description: "This type of action exports user data from the workspace's data warehouse to an application. " +
			"It operates on a destination app connection that supports users.",
		Endpoints: []*Endpoint{
			{
				Name:        "Create action",
				Description: "Create a destination action that exports users to an app.",
				Method:      POST,
				URL:         "/v0/actions",
				Parameters: []types.Property{
					nameParameter,
					{
						Name:           "connection",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "230527183",
						Description:    "The ID of the connection to which the users will be written. It must be a destination app connection that exports users.",
					},
					{
						Name:           "target",
						Type:           types.Text().WithValues("Users"),
						CreateRequired: true,
						Placeholder:    `"Users"`,
						Description:    "The entity on which the action operates, which must be `\"Users\"` in order to create an action that exports users.",
					},
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicate if the action is enabled once created.",
					},
					filterParameter,
					{
						Name:           "exportMode",
						Type:           types.Text(),
						CreateRequired: true,
						Placeholder:    `"CreateOnly"`,
						Description: "La modalità con cui gi utenti sono esportati:\n\n" +
							"* `CreateOnly`: vengono esclusivamente creati degli utenti nell'app. Nessun cliente esistente viene modificato.\n" +
							"* `UpdateOnly`: vengono esclusivamente aggiornati gli utenti esistenti nell'app. Nessun nuovo cliente viene creato.\n" +
							"* `CreateOrUpdate`: se un utente è già presente nell'app, viene aggiornato, altrimenti viene creato come nuovo.",
					},
					{
						Name:           "matching",
						Type:           types.Text().WithValues("CreateOnly", "UpdateOnly", "CreateOrUpdate"),
						CreateRequired: true,
						Placeholder:    `"CreateOnly"`,
					},
					{
						Name:           "inSchema",
						Type:           types.Parameter("schema"),
						CreateRequired: true,
						Placeholder:    `{...}`,
					},
					{
						Name:           "outSchema",
						Type:           types.Parameter("schema"),
						CreateRequired: true,
						Placeholder:    `{...}`,
					},
					{
						Name:           "transformation",
						Type:           types.Parameter("transformation"),
						CreateRequired: true,
						Placeholder:    `{...}`,
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
					{422, ConnectorNotExist, "connector does not exist"},
					{422, UnsupportedLanguage, "transformation language is not supported"},
				},
			},
			{
				Name:        "Update action",
				Description: "Update a destination action that exports users to an app.",
				Method:      PUT,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the destination app action that exports users.",
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
						Name:        "inSchema",
						Type:        types.Parameter("schema"),
						Placeholder: `{...}`,
					},
					{
						Name:        "outSchema",
						Type:        types.Parameter("schema"),
						Placeholder: `{...}`,
					},
					{
						Name:        "transformation",
						Type:        types.Parameter("transformation"),
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
				Description: "Get a destination action that exports users to an app.",
				Method:      GET,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the destination app action that exports users.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Placeholder: "705981339",
							Description: "The ID of the destination app action that exports users.",
						},
						nameParameter,
						{
							Name:        "connection",
							Type:        types.Int(32),
							Placeholder: "1371036433",
							Description: "The ID of the connection to which the users will be exported. It is a destination app.",
						},
						{
							Name:        "target",
							Type:        types.Text().WithValues("Users"),
							Placeholder: `"Users"`,
							Description: "The entity on which the action operates. It is always `\"Users\"` for an action that exports users.",
						},
						{
							Name:        "enabled",
							Type:        types.Boolean(),
							Placeholder: "true",
							Description: "Indicates if the action is enabled.",
						},
						{
							Name:        "running",
							Type:        types.Boolean(),
							Placeholder: "false",
							Description: "Indicates if the action is running.",
						},
						scheduleStartParameter,
						exportSchedulePeriodParameter,
						filterParameter,
						{
							Name:        "inSchema",
							Type:        types.Parameter("schema"),
							Placeholder: `{...}`,
						},
						{
							Name:        "outSchema",
							Type:        types.Parameter("schema"),
							Placeholder: `{...}`,
						},
						{
							Name:        "transformation",
							Type:        types.Parameter("transformation"),
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
