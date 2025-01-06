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

	modeParameter := types.Property{
		Name:           "mode",
		Type:           types.Text().WithValues("Normal", "Inspection", "Maintenance"),
		Placeholder:    `"Normal"`,
		UpdateRequired: true,
		Description:    "The mode of the data warehouse.",
	}
	cancelIncompatibleOperationsParameter := types.Property{
		Name: "cancelIncompatibleOperations",
		Type: types.Boolean(),
	}
	settingsParameter := types.Property{
		Name:           "settings",
		Type:           types.Parameter("Warehouse"),
		Placeholder:    "{...}",
		UpdateRequired: true,
		Description:    "The settings of the data warehouse.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "warehouse",
		Name:        "Warehouse",
		Description: "A workspace enables Meergo to retrieve customer and event data from an external source location or send them to an external destination location.",
		Endpoints: []*Endpoint{
			{
				Name:        "Get the warehouse",
				Description: "Returns the name and the settings of the current workspace's warehouse.",
				Method:      GET,
				URL:         "/v0/warehouse",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "name",
							Type:        types.Text().WithValues("Snowflake", "PostgreSQL"),
							Placeholder: `"Snowflake"`,
							Description: "The name of the data warehouse.",
						},
						settingsParameter,
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
			{
				Name:        "Update the warehouse",
				Description: "Updates the warehouse of the current workspace.",
				Method:      PUT,
				URL:         "/v0/warehouse",
				Parameters: []types.Property{
					settingsParameter,
					modeParameter,
					cancelIncompatibleOperationsParameter,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, DifferentWarehouse, "data warehouse is a different data warehouse"},
					{422, InvalidWarehouseSettings, "data warehouse settings are not valid"},
					{422, WarehouseError, "error occurred with the data warehouse"},
				},
			},
			{
				Name: "Test the warehouse update",
				Description: "Tests the update of the warehouse of the current workspace.\n\n" +
					"If the settings are incorrect or the warehouse can’t be accessed with the given settings, an error will be returned. " +
					"If no error occurs, the settings are valid.",
				Method: PUT,
				URL:    "/v0/warehouse/test",
				Parameters: []types.Property{
					settingsParameter,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, DifferentWarehouse, "data warehouse is a different data warehouse"},
					{422, InvalidWarehouseSettings, "data warehouse settings are not valid"},
					{422, WarehouseError, "error occurred with the data warehouse"},
				},
			},
			{
				Name:        "Update warehouse mode",
				Description: "Updates the mode of the current workspace's data warehouse.",
				Method:      PUT,
				URL:         "/v0/warehouse/mode",
				Parameters: []types.Property{
					modeParameter,
					cancelIncompatibleOperationsParameter,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
			{
				Name:        "Repair the warehouse",
				Description: "Repairs the current workspace's warehouse.",
				Method:      POST,
				URL:         "/v0/warehouse/repair",
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, WarehouseError, "error occurred with the data warehouse"},
				},
			},
		},
	})

}
