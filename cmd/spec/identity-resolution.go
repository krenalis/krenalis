//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package spec

import (
	"github.com/meergo/meergo/core/types"
)

func init() {

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "identity-resolution",
		Name: "Identity resolution",
		Description: "Identities are the users obtained from sources like [application APIs](api/connections/applications), [databases](api/connections/databases), [files](api/connections/file-storages), or events. " +
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
					{422, OperationAlreadyExecuting, "another operation is already executing"},
				},
			},
			{
				Name: "Get information about latest identity resolution",
				Description: "Returns information about the latest identity resolution.\n\n" +
					"Depending on the returned values:\n" +
					"- If neither `startTime` nor `endTime` are returned, it means that no identity resolutions have ever been performed for the workspace.\n" +
					"- If only `startTime` is returned, it means that the workspace is currently running an identity resolution.\n" +
					"- If both `startTime` and `endTime` are returned, it means that an identity resolution has been performed and there are no identity resolutions currently running.",
				Method: GET,
				URL:    "/v1/identity-resolution/latest",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:      "startTime",
							Type:      types.DateTime(),
							Prefilled: `"2025-01-12T09:37:22"`,
							Nullable:  true,
							Description: "Start timestamp (UTC) of the latest identity resolution, either running or completed.\n\n" +
								"If null, no identity resolutions have ever been executed for the workspace.",
						},
						{
							Name:      "endTime",
							Type:      types.DateTime(),
							Prefilled: `"2025-01-12T09:42:51"`,
							Nullable:  true,
							Description: "End timestamp (UTC) for the latest identity resolution.\n\n" +
								"If null, it means that the identity resolution is still in progress, or that no identity resolution has ever been executed on the workspace.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
			{
				Name:   "Update identity resolution settings",
				Method: PUT,
				URL:    "/v1/identity-resolution/settings",
				Parameters: []types.Property{
					{
						Name:      "runOnBatchImport",
						Type:      types.Boolean(),
						Prefilled: `true`,
						Description: "Indicates if identity resolution is automatically run when a batch import is completed." +
							" The default is false.",
					},
					{
						Name:      "identifiers",
						Type:      types.Array(types.Text()),
						Nullable:  true,
						Prefilled: `[ "customer_id", "email" ]`,
						Description: "The identifiers of the identity resolution, ordered from the highest precedence to the lowest precedence.\n\n" +
							"An identifier must be a property path that refers to a property of the user schema of the workspace. " +
							"The referred property must have type `int`, `uint`, `uuid`, `inet`, `text`, or `decimal` with zero scale. Identifiers cannot be repeated.\n\n" +
							"Not specifying any identifier means performing identity resolution without comparing any identifiers.",
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, AlterSchemaInExecution, "alter schema is in execution so the identifiers cannot be updated"},
					{422, IdentityResolutionInExecution, "identity resolution is in execution so the identifiers cannot be updated"},
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
							Name:      "runOnBatchImport",
							Type:      types.Boolean(),
							Prefilled: `true`,
							Description: "Indicates if identity resolution is automatically run when a batch import is completed." +
								" The default is false.",
						},
						{
							Name:      "identifiers",
							Type:      types.Array(types.Text()),
							Prefilled: `[ "customer_id", "email" ]`,
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
