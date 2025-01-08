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

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "users",
		Name: "Users",
		Description: "Users are the users associated with a [workspace](workspaces), imported from the various sources, that are stored inside the [warehouse](warehouse), and that can be queried or exported to data destinations.\n\n" +
			"User identities represent the users as they are imported from the various sources, while the actual users are the users resolved – and possibly merged – by [identity resolution](identity-resolution).",
		Endpoints: []*Endpoint{
			{
				Name: "List all users",
				Description: "This endpoint retrieves users stored in the workspace's data warehouse, up to a maximum number of users defined by `limit`. You must specify which properties to include. " +
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
						Description: "The property by which to sort the users to be returned. It can be any property from the user schema with an sortable type, meaning it cannot be of type JSON, Array, Object, or Map.",
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
					{422, WarehouseError, "error occurred with the data warehouse"},
				},
			},
			{
				Name:        "Retrive the traits of a user ",
				Description: "Returns the traits of a user given its identifier.",
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
					{422, WarehouseError, "error occurred with the data warehouse"},
				},
			},
			{
				Name:        "Retrive the identities of a user",
				Description: "Returns the identities of a user given its identifier.",
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
						Description: "The maximum number of identities to return. It must be between 0 and 1000, with a default value of 1000.",
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
					{422, WarehouseError, "error occurred with the data warehouse"},
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
						Description: "The set of renamed and deleted paths. If a property path has been renamed in the new schema, `rePaths` maps the old path to the new one. " +
							"If a property path has been removed, `rePaths` specifies the old path as the key with null as the value.",
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, AlterSchemaInProgress, "alter schema operation is already in progress"},
					{422, ConnectionNotExist, "primary source does not exist"},
					{422, IdentityResolutionInProgress, "identity resolution is currently in progress"},
					{422, InspectionMode, "data warehouse is in inspection mode"},
					{422, InvalidSchemaUpdate, "cannot update the schema as specified"},
					{422, WarehouseError, "error occurred with the data warehouse"},
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
						Description: "The set of renamed and deleted paths. If a property path has been renamed in the new schema, `rePaths` maps the old path to the new one. " +
							"If a property path has been removed, `rePaths` specifies the old path as the key with null as the value.",
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
					{422, WarehouseError, "error occurred with the data warehouse"},
				},
			},
		},
	})

}
