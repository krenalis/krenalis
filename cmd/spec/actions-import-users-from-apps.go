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
		Description: "The filter applied to the app users. If it's not null, only the app users that match the filter will be included, otherwise all users will be included.\n\n" +
			"See the [filters documentation](/filters) for more details.",
	}
	inSchemaParameter := types.Property{
		Name:           "inSchema",
		Type:           types.Parameter("schema"),
		CreateRequired: true,
		Placeholder:    `{...}`,
		Description: "The schema for the properties used in the filter, as well as the input properties for the transformation.\n\n" +
			"When importing users from apps, this should be a subset of the app’s destination schema.",
	}
	outSchemaParameter := types.Property{
		Name:           "outSchema",
		Type:           types.Parameter("schema"),
		CreateRequired: true,
		Placeholder:    `{...}`,
		Description: "The schema for the output properties of the transformation.\n\n" +
			"When importing users from apps, this should be a subset of the workspace's user schema.",
	}
	transformationParameter := types.Property{
		Name:           "transformation",
		Type:           types.Parameter("transformation"),
		CreateRequired: true,
		Placeholder:    `{...}`,
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "actions-import-users-from-apps",
		Name: "Import users from apps",
		Description: "This type of action imports user data from an application into the workspace's data warehouse. " +
			"It operates on a source app connection that supports users.",
		Endpoints: []*Endpoint{
			{
				Name:        "Create action",
				Description: "Create a source action to import users from an app.",
				Method:      POST,
				URL:         "/v0/actions",
				Parameters: []types.Property{
					nameParameter,
					{
						Name:           "connection",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "230527183",
						Description:    "The ID of the connection from which to read the users. It must be a source app that can imports users.",
					},
					{
						Name:           "target",
						Type:           types.Text().WithValues("Users"),
						CreateRequired: true,
						Placeholder:    `"Users"`,
						Description:    "The entity on which the action operates, which must be `\"Users\"` in order to create an action that imports users.",
					},
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicate if the action is enabled once created.",
					},
					filterParameter,
					inSchemaParameter,
					outSchemaParameter,
					transformationParameter,
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
				Description: "Update a source action that imports users from an app.",
				Method:      PUT,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source app action to update.",
					},
					nameParameter,
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicates if the action is enabled. Use the [Set status](/api/actions#set-status) endpoint to change only the action's status.",
					},
					filterParameter,
					inSchemaParameter,
					outSchemaParameter,
					transformationParameter,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
					{422, UnsupportedLanguage, "transformation language is not supported"},
				},
			},
			{
				Name:        "Get action",
				Description: "Get a source action that imports users from an app.",
				Method:      GET,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source app action that imports users.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Placeholder: "705981339",
							Description: "The ID of the source app action that imports users.",
						},
						nameParameter,
						{
							Name:        "connector",
							Type:        types.Text(),
							Placeholder: `"HubSpot"`,
							Description: "The name of the connection's connector.",
						},
						{
							Name:        "connectorType",
							Type:        types.Text().WithValues("App", "Database", "FileStorage", "Mobile", "Server", "Website"),
							Placeholder: `"App"`,
							Description: "The type of the connection's connector. It is always `\"App\"` when the action imports users from an app.",
						},
						{
							Name:        "connection",
							Type:        types.Int(32),
							Placeholder: "1371036433",
							Description: "The ID of the connection from which users are read. It is a source app that supports users.",
						},
						{
							Name:        "connectionRole",
							Type:        types.Text().WithValues("Source", "Destination"),
							Placeholder: `"Source"`,
							Description: "The role of the action's connection. It is always `\"Source\"` when the action imports users from an app.",
						},
						{
							Name:        "target",
							Type:        types.Text().WithValues("Users", "Events"),
							Placeholder: `"Users"`,
							Description: "The entity on which the action operates. It is always `\"Users\"` when the action imports users from an app.",
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
							Description: "The schema for the properties used in the filter, as well as the input properties for the transformation.",
						},
						{
							Name:        "outSchema",
							Type:        types.Parameter("schema"),
							Placeholder: `{...}`,
							Description: "The schema for the output properties of the transformation.",
						},
						{
							Name:        "transformation",
							Type:        types.Parameter("transformation"),
							Placeholder: `{...}`,
						},
						runningParameter,
						scheduleStartParameter,
						importSchedulePeriodParameter,
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
