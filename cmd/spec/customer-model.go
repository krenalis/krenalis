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

	const rePathsDescription = "Specifies renamed properties and additional information that cannot be expressed through the `schema` parameter alone.\n" +
		"\n" +
		"In particular:\n" +
		"\n" +
		"- If a property in `schema` has been renamed, the new path must be added as a key in `rePaths` and the old path as the associated value. Otherwise, instead of performing a rename operation, a new property with the new path would be created, and the property with the old path would be deleted.\n" +
		"- If a property in `schema` has been added with the same path of an already existent one which should be removed, then the path of the new property must be added as a key in `rePaths` and `null` as the associated value. Otherwise, instead of creating a new property and deleting the old one, it would be interpreted as a rename operation."

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "customer-model",
		Name:        "Customer model",
		Description: "",
		Endpoints: []*Endpoint{
			{
				Name:        "Get schema",
				Description: "Returns the user schema of the workspace.",
				Method:      GET,
				URL:         "/v1/users/schema",
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
				Name:        "Update schema",
				Description: "Updates the user schema of the workspace.",
				Method:      PUT,
				URL:         "/v1/users/schema",
				Parameters: []types.Property{
					{
						Name:           "schema",
						Type:           types.Parameter("Schema"),
						CreateRequired: true,
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
					{422, ConnectionNotExist, "primary source does not exist"},
					{422, InspectionMode, "data warehouse is in inspection mode"},
					{422, InvalidSchemaUpdate, "cannot update the schema as specified"},
					{422, OperationAlreadyExecuting, "another operation is already executing"},
				},
			},
			{
				Name: "Preview schema update",
				Description: "Returns the SQL queries that would be executed on the warehouse to update the user schema.\n\n" +
					"It does not make any changes to the schema or execute any queries on the warehouse.",
				Method: PUT,
				URL:    "/v1/users/schema/preview",
				Parameters: []types.Property{
					{
						Name:           "schema",
						Type:           types.Parameter("Schema"),
						CreateRequired: true,
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
			{
				Name: "Get information about latest user schema update",
				Description: "Returns information about the latest user schema update.\n\n" +
					"Depending on the returned values:\n" +
					"- If neither `startTime` nor `endTime` are returned, it means that no user schema update has never been performed for the workspace.\n" +
					"- If only `startTime` is returned, it means that the workspace is currently running a user schema update.\n" +
					"- If both `startTime` and `endTime` are returned, it means that a user schema update has been performed and there are no user schema updates currently running.",
				Method: GET,
				URL:    "/v1/users/schema/latest-update",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "startTime",
							Type:        types.DateTime(),
							Placeholder: `"2025-01-12T09:37:22"`,
							Nullable:    true,
							Description: "Start timestamp (UTC) of the latest user schema update, either running or completed.\n\n" +
								"If null, no user schema update has never been started for the workspace.",
						},
						{
							Name:        "endTime",
							Type:        types.DateTime(),
							Placeholder: `"2025-01-12T09:37:25"`,
							Nullable:    true,
							Description: "End timestamp (UTC) for the latest user schema update.\n\n" +
								"If null, it means that the user schema update is still in progress, or that no schema update has never been performed for the workspace.",
						},
						{
							Name:        "error",
							Type:        types.Text(),
							Placeholder: "null",
							Nullable:    true,
							Description: "A possible error in the execution of the latest update of the user schema.\n\n" +
								"If null, it means that no update of the user schema has never been executed, or that one is in progress, or that the last one executed completed without errors.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
		},
	})

}
