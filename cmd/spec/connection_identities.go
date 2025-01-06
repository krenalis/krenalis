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
		Description: "Identities are the users obtained from sources like apps, databases, files, or events. " +
			"These identities are filtered and transformed according to a defined action. " +
			"Once identity resolution is complete, the collected identities from all connections are unified to form the workspace users.",
		Endpoints: []*Endpoint{
			{
				Name:        "List the connection identities",
				Description: "Retrieves the identities associated with a connection.",
				Method:      POST,
				URL:         "/v0/connections/:id/identities",
				Parameters: []types.Property{
					{
						Name:           "connection",
						Type:           types.Int(32),
						Placeholder:    "1371036433",
						UpdateRequired: true,
						Description:    "The source connection from which to retrive the identities.",
					},
					{
						Name:        "first",
						Type:        types.Int(32),
						Placeholder: `0`,
						Description: "The number of identities to skip before starting to return results. The default value is 0.",
					},
					{
						Name:           "limit",
						Type:           types.Int(32).WithIntRange(1, 1000),
						CreateRequired: true,
						Placeholder:    `1000`,
						Description:    "The maximum number of identities to return. The value must be within the range [1, 1000].",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "identities",
							Type:        types.Array(types.Map(types.JSON())),
							Placeholder: `[ { ... } ]`,
							Description: "The connection's identities.",
						},
						{
							Name:        "count",
							Type:        types.Int(32),
							Placeholder: `23`,
							Description: "The estimated total number of identities in the connection.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
					{422, WarehouseError, "error occurred with the data warehouse"},
				},
			},
		},
	})

}
