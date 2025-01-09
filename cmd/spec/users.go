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

	eventsParameter := types.Array(types.Object(append([]types.Property{
		{Name: "id", Type: types.UUID(), Placeholder: `"b1d868f3-43f6-4965-bbd2-85ca8dd609b3"`},
		{Name: "user", Type: types.UUID(), ReadOptional: true, Placeholder: `"9102d2a1-0714-4c13-bafd-8a38bc3d0cff"`},
		{Name: "connection", Type: types.Int(32), Placeholder: "1371036433"},
	}, eventProperties...)))

	identityType := types.Object([]types.Property{
		{
			Name:        "connection",
			Type:        types.Int(32),
			Description: "The ID of the connection through which the identity was observed.",
		},
		{
			Name:        "action",
			Type:        types.Int(32),
			Description: "The ID of the action through which the identity was observed.",
		},
		{
			Name:        "id",
			Type:        types.Text(),
			Description: "The ID of the identity. It is empty for identities imported from anonymous events.",
		},
		{
			Name:        "anonymousIds",
			Type:        types.Array(types.Text()),
			Nullable:    true,
			Description: "The anonymousIds of the identity. It is null for identities not imported from events.",
		},
		{
			Name:        "lastChangeTime",
			Type:        types.DateTime(),
			Description: "The identity’s most recent change time.",
		},
	})

	const rePathsDescription = "Specifies renamed properties and additional information that cannot be expressed through the `schema` parameter alone.\n" +
		"\n" +
		"In particular:\n" +
		"\n" +
		"- If a property in `schema` has been renamed, the new path must be added as a key in `rePaths` and the old path as the associated value. Otherwise, instead of performing a rename operation, a new property with the new path would be created, and the property with the old path would be deleted.\n" +
		"- If a property in `schema` has been added with the same path of an already existent one which should be removed, then the path of the new property must be added as a key in `rePaths` and `null` as the associated value. Otherwise, instead of creating a new property and deleting the old one, it would be interpreted as a rename operation."

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "users",
		Name: "Users",
		Description: "Users are the users associated with a [workspace](workspaces), imported from the various sources, that are stored inside the [warehouse](warehouse), and that can be queried or exported to data destinations.\n\n" +
			"User identities represent the users as they are imported from the various sources, while the actual users are the users resolved – and possibly merged – by [identity resolution](identity-resolution).",
		Endpoints: []*Endpoint{
			{
				Name: "Retrieve all users",
				Description: "Retrieves users stored in the workspace's data warehouse, up to a maximum number of users defined by `limit`. You must specify which properties to include. " +
					"If a filter is provided, only users that match the filter criteria will be returned.",
				Method: POST,
				URL:    "/v0/users",
				Parameters: []types.Property{
					{
						Name:           "properties",
						Type:           types.Array(types.Text()),
						CreateRequired: true,
						Placeholder:    `[ "email", "last_name" ]`,
						Description:    "The user properties to return.",
					},
					{
						Name:        "filter",
						Type:        filterType,
						Nullable:    true,
						Description: "The filter applied to the users. If it's not null, only the users that match the filter will be returned.",
					},
					{
						Name:        "order",
						Type:        types.Text(),
						Placeholder: `"email"`,
						Description: "The name of the property by which to sort the users to be returned. It can be any property from the user schema with an sortable type, meaning it cannot be of type JSON, Array, Object, or Map.",
					},
					{
						Name:        "orderDesc",
						Type:        types.Boolean(),
						Placeholder: `false`,
						Description: "The descending sorting order in which to return the users: if true, the users are sorted in descending order; otherwise, they are sorted in ascending order.",
					},
					{
						Name:        "first",
						Type:        types.Int(32),
						Placeholder: `0`,
						Description: "The number of users to skip before starting to return results. The default value is 0.",
					},
					{
						Name:           "limit",
						Type:           types.Int(32).WithIntRange(1, 1000),
						CreateRequired: true,
						Placeholder:    `1000`,
						Description:    "The maximum number of users to return. The value must be within the range [1, 1000].",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "users",
							Type:        types.Array(types.Object([]types.Property{{Name: "id", Type: types.Text()}})),
							Placeholder: "[ { \"id\": 123 } ]",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
					{422, OrderNotExist, "order does not exist in schema"},
					{422, OrderTypeNotSortable, "cannot sort by non-sortable type"},
					{422, PropertyNotExist, "property does not exist in the user schema"},
				},
			},
			{
				Name:        "Retrieve user traits",
				Description: "Retrieves, from the workspace's data warehouse, the traits of a user given its identifier.",
				Method:      GET,
				URL:         "/v0/users/:id/traits",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.UUID(),
						Placeholder:    `"86de98fe-8f26-49ac-87dc-8a14997a97d9"`,
						UpdateRequired: true,
						Description:    "The ID of the user.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "traits",
							Type:        types.Map(types.JSON()),
							Placeholder: `{ ... }`,
							Description: "The traits of the user, following the user schema.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "user does not exist"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
				},
			},
			{
				Name: "Retrieve user events",
				Description: "Retrieves the most recent events for a user given their identifier. The events are read from the workspace's data warehouse and are listed in descending order, starting with the most recent ones.\n\n" +
					"This endpoint provides a streamlined, user-focused alternative to the [/events](/api/events#list-all-events) endpoint.\n" +
					"While the [/events](/api/events#list-all-events) endpoint offers advanced filtering and sorting options, this endpoint is designed for simple access to a single user’s event history.",
				Method: GET,
				URL:    "/v0/users/:id/events",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.UUID(),
						Placeholder:    `"86de98fe-8f26-49ac-87dc-8a14997a97d9"`,
						UpdateRequired: true,
						Description:    "The ID of the user.",
					},
					{
						Name:           "properties",
						Type:           types.Array(types.Text()),
						UpdateRequired: true,
						Description: "The names of the properties to return. At least one property must be provided.\n\n" +
							"The properties can be specified in the query string in two ways:\n" +
							"* `properties=timestamp,event`\n* `properties=timestamp&properties=event`",
					},
					{
						Name:        "limit",
						Type:        types.Int(32),
						Description: "The maximum number of events to return. If provided, it must be a value between 1 and 1000. If not specified, the default value is 100.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "events",
							Type:        eventsParameter,
							Placeholder: `...`,
							Description: "The most recent events of the user. An empty array is returned if no events are available or if the specified user does not exist.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
				},
			},
			{
				Name:        "Retrieve user identities",
				Description: "Retrieves, from the workspace's data warehouse, the identities of a user given its identifier.",
				Method:      GET,
				URL:         "/v0/users/:id/identities",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.UUID(),
						Placeholder:    `"86de98fe-8f26-49ac-87dc-8a14997a97d9"`,
						UpdateRequired: true,
						Description:    "The ID of the user.",
					},
					{
						Name:        "first",
						Type:        types.Int(32),
						Description: "The number of identities to skip before starting to return results. The default value is 0.",
					},
					{
						Name:        "limit",
						Type:        types.Int(32),
						Description: "The maximum number of identities to return. If provided, it must be a value between 1 and 1000. If not specified, the default value is 1000.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "identities",
							Type:        types.Array(identityType),
							Placeholder: `{ ... }`,
							Description: "The user’s identities, containing at least one identity.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "user does not exist"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
				},
			},
			{
				Name:        "Get the user schema",
				Description: "Returns the user schema of the workspace.",
				Method:      GET,
				URL:         "/v0/users/schema",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "schema",
							Type:        types.Parameter("Schema"),
							Placeholder: "...",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
			{
				Name:        "Update the user schema",
				Description: "Updates the user schema of the workspace.",
				Method:      PUT,
				URL:         "/v0/users/schema",
				Parameters: []types.Property{
					{
						Name:           "schema",
						Type:           types.Parameter("Schema"),
						UpdateRequired: true,
						Description:    "The new user schema. It must include at least one property.",
					},
					{
						Name:        "primarySources",
						Type:        types.Map(types.Int(32)),
						Placeholder: `{ "email": 1371036433 }`,
						Description: "The primary source for each schema property that has one, where the key is the property name and the value is the connection identifier.\n\n" +
							"This source defines where the definitive value for the property is read from, preventing other sources from overwriting it once it is set.\n\n" +
							"If no primary source is provided, the new schema will have no primary sources defined.",
					},
					{
						Name:        "rePaths",
						Type:        types.Map(types.Text()),
						Placeholder: `{ "city": "address.city", "street3": null }`,
						Description: rePathsDescription,
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, AlterSchemaInProgress, "alter schema operation is already in progress"},
					{422, ConnectionNotExist, "primary source does not exist"},
					{422, IdentityResolutionInProgress, "identity resolution is currently in progress"},
					{422, InspectionMode, "data warehouse is in inspection mode"},
					{422, InvalidSchemaUpdate, "cannot update the schema as specified"},
				},
			},
			{
				Name: "Preview a user schema update",
				Description: "Returns the SQL queries that would be executed on the warehouse to update the user schema.\n\n" +
					"It does not make any changes to the schema or execute any queries on the warehouse.",
				Method: PUT,
				URL:    "/v0/users/schema/preview",
				Parameters: []types.Property{
					{
						Name:           "schema",
						Type:           types.Parameter("Schema"),
						UpdateRequired: true,
						Description:    "The new user schema. It must include at least one property.",
					},
					{
						Name:        "rePaths",
						Type:        types.Map(types.Text()),
						Placeholder: `{ "city": "address.city", "street3": null }`,
						Description: rePathsDescription,
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "queries",
							Type:        types.Array(types.Text()),
							Placeholder: `[ "ALTER TABLE ..." ]`,
							Description: "The SQL queries that would be executed on the warehouse to modify the user schema.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, InvalidSchemaUpdate, "cannot update the schema as specified"},
				},
			},
		},
	})

}
