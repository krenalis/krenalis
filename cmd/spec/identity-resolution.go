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
		Description: "Identities are the users obtained from sources like [apps](app-connections), [databases](database-connections), [files](file-storage-connections), or events. " +
			"These identities are filtered and transformed according to a defined action. " +
			"Once identity resolution is complete, the collected identities from all [connections](connections) are unified to form the workspace [users](users).",
		Endpoints: []*Endpoint{
			{
				Name:        "Start identity resolution",
				Description: "Starts identity resolution that resolves the identities of the workspace.",
				Method:      POST,
				URL:         "/v1/identity-resolution/start",
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, InspectionMode, "data warehouse is in inspection mode"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
				},
			},
			{
				Name:        "Retrieve latest identity resolution info",
				Description: "Returns the start and end times of the latest identity resolution.",
				Method:      GET,
				URL:         "/v1/identity-resolution/latest",
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
				URL:    "/v1/identity-resolution/settings",
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
						Description: "The identifiers of the identity resolution, ordered from the highest precedence to the lowest precedence.\n\n" +
							"An identifier must be a property path that refers to a property of the user schema of the workspace. " +
							"The referred property must have type `int`, `uint`, `uuid`, `inet`, `text`, or `decimal` with zero scale. Identifiers cannot be repeated.\n\n" +
							"Not specifying any identifier means performing identity resolution without comparing any identifiers.",
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
				URL:    "/v1/identity-resolution/settings",
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
							Description: "The identifiers of the identity resolution, ordered from the highest precedence to the lowest precedence.\n\n" +
								"If no identifiers are returned, it means that the identity resolution is performed without comparing any identifiers.",
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
