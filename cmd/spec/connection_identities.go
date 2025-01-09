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
		ID:   "connection-identities",
		Name: "Connection identities",
		Description: "Users from sources such as apps, databases, files, or events are filtered and transformed based on defined actions into user identities within the workspace.\n\n" +
			"Then, identity resolution combines all user identities from different source connections into a single user profile.",
		Endpoints: []*Endpoint{
			{
				Name:        "Retrieve user identities",
				Description: "Retrieves, from the workspace's data warehouse, the user identities imported by a connection.",
				Method:      POST,
				URL:         "/v0/connections/:id/identities",
				Parameters: []types.Property{
					{
						Name:           "connection",
						Type:           types.Int(32),
						Placeholder:    "1371036433",
						UpdateRequired: true,
						Description:    "The source connection from which to retrieve the user identities.",
					},
					{
						Name:        "first",
						Type:        types.Int(32),
						Placeholder: `0`,
						Description: "The number of user identities to skip before starting to return results. The default value is 0.",
					},
					{
						Name:           "limit",
						Type:           types.Int(32).WithIntRange(1, 1000),
						CreateRequired: true,
						Placeholder:    `1000`,
						Description:    "The maximum number of user identities to return. The value must be within the range [1, 1000].",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "identities",
							Type:        types.Array(types.Map(types.JSON())),
							Placeholder: `[ { ... } ]`,
							Description: "The connection's user identities.",
						},
						{
							Name:        "count",
							Type:        types.Int(32),
							Placeholder: `23`,
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
		},
	})

}
