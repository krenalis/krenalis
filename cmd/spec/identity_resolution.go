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
		ID:   "identity-resolution",
		Name: "Identity resolution",
		Description: "Identities are the users obtained from sources like [apps](connection-app), [databases](connection-database), [files](connection-files), or events. " +
			"These identities are filtered and transformed according to a defined action. " +
			"Once identity resolution is complete, the collected identities from all [connections](connections) are unified to form the workspace [users](users).",
		Endpoints: []*Endpoint{
			{
				Name:        "Start identity resolution",
				Description: "Starts identity resolution that resolves the identities of the workspace.",
				Method:      POST,
				URL:         "/v0/identity-resolution/start",
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, InspectionMode, "data warehouse is in inspection mode"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
				},
			},
			{
				Name:        "Retrieve last identity resolution info",
				Description: "Returns the start and end times of the last identity resolution.",
				Method:      GET,
				URL:         "/v0/identity-resolution/latest",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "startTime",
							Type:        types.DateTime(),
							Placeholder: `"2025-01-12T09:37:22"`,
							Nullable:    true,
							Description: "The UTC start time of the latest identity resolution. It is null if no identity resolution has been executed yet.",
						},
						{
							Name:        "endTime",
							Type:        types.DateTime(),
							Placeholder: `"2025-01-12T09:42:51"`,
							Nullable:    true,
							Description: "The UTC end time of the latest identity resolution. It is null if an identity resolution is still in progress or if none has been executed yet.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
				},
			},
			{
				Name:   "Update identity resolution settings",
				Method: PUT,
				URL:    "/v0/identity-resolution/settings",
				Parameters: []types.Property{
					{
						Name:        "runOnBatchImport",
						Type:        types.Boolean(),
						Placeholder: `true`,
						Description: "Indicates if identity resolution is automatically run when a batch import is completed." +
							" By default is false.",
					},
					{
						Name:        "identifiers",
						Type:        types.Array(types.Text()),
						Nullable:    true,
						Placeholder: `[ "customer_id", "email" ]`,
						Description: "The identifiers in the specified order. An identifier must be a property in the user schema " +
							"with a type of `Int`, `Uint`, `UUID`, `Inet`, `Text`, or `Decimal` with zero scale. Identifiers cannot be repeated.",
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, PropertyNotExist, "property does not exist in the user schema"},
					{422, TypeNotAllowed, "a property has a type which is not allowed for identifiers"},
				},
			},
			{
				Name:   "Get identity resolution settings",
				Method: GET,
				URL:    "/v0/identity-resolution/settings",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "runOnBatchImport",
							Type:        types.Boolean(),
							Placeholder: `true`,
							Description: "Indicates if identity resolution is automatically run when a batch import is completed." +
								" By default is false.",
						},
						{
							Name:        "identifiers",
							Type:        types.Array(types.Text()),
							Placeholder: `[ "customer_id", "email" ]`,
							Description: "The identifiers in the specified order.",
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
