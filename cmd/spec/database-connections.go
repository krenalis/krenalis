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

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "database-connections",
		Name:        "Database connections",
		Description: "These endpoints are specific to database connections.",
		Endpoints: []*Endpoint{
			{
				Name:        "Retrieve query schema",
				Description: "It executes the provided query on a source database connection and returns the result schema along with the first rows.",
				Method:      POST,
				URL:         "/v1/connections/:id/query",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						Placeholder:    "1371036433",
						CreateRequired: true,
						Description:    "The ID of the source database connection on which the query will be executed.",
					},
					{
						Name:           "query",
						Type:           types.Text(),
						Placeholder:    `"SELECT name, email FROM users"`,
						CreateRequired: true,
						Description: "The query to execute to read the result schema along with sample rows for reference. " +
							"It must contain the \"limit\" placeholder and cannot be longer than 16,777,215 characters.",
					},
					{
						Name:        "limit",
						Type:        types.Int(32).WithIntRange(0, 100),
						Placeholder: `50`,
						Description: "The maximum number of rows to return along with the schema. The maximum is 100; the default is 0, meaning no rows are returned.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "schema",
							Type:        types.Parameter("schema"),
							Nullable:    true,
							Placeholder: `{ ... }`,
							Description: "The schema of the query. It will be null if there are no supported columns.",
						},
						{
							Name:        "rows",
							Type:        types.Array(types.Map(types.JSON())),
							Placeholder: `[ { ... } ]`,
							Description: "The rows.",
						},
						{
							Name:        "issues",
							Type:        types.Array(types.Text()),
							Nullable:    true,
							Placeholder: `[ "Column \"mac_addr\" cannot be imported because its type \"MACADDR\" is not supported" ]`,
							Description: "The issues encountered while reading the table, such as unsupported columns, which did not prevent processing. " +
								"If it is not null, it contains at least one issue.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
				},
			},
			{
				Name:        "Retrieve table schema",
				Description: "Returns the schema of a specified database table of a destination database connection.",
				Method:      GET,
				URL:         "/v1/connections/:id/tables/:name",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						Placeholder:    "1371036433",
						CreateRequired: true,
						Description:    "The database destination connection where the table is located.",
					},
					{
						Name:           "name",
						Type:           types.Text(),
						Placeholder:    `users`,
						CreateRequired: true,
						Description:    "The name of the table.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "schema",
							Type:        types.Parameter("schema"),
							Nullable:    true,
							Placeholder: `{ ... }`,
							Description: "The schema of the table. It will be null if there are no supported columns.",
						},
						{
							Name:        "issues",
							Type:        types.Array(types.Text()),
							Nullable:    true,
							Placeholder: `[ "Column \"mac_addr\" cannot be imported because its type \"MACADDR\" is not supported" ]`,
							Description: "The issues encountered while reading the table, such as unsupported columns, which did not prevent processing. " +
								"If it is not null, it contains at least one issue.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
				},
			},
		},
	})

}
