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

var scheduleStartParameter = types.Property{
	Name:        "scheduleStart",
	Type:        types.Int(32),
	Placeholder: "15",
	Nullable:    true,
	Description: "The start time of the schedule in minutes. It indicates the minute, starting from 00:00, when the first scheduled execution of the day begins. Subsequent executions will occur based on the interval defined by the scheduler period. It is null if the scheduler is disabled.",
}

var schedulePeriodParameter = types.Property{
	Name:        "schedulePeriod",
	Type:        types.Text().WithValues("5m", "15m", "30m", "1h", "2h", "3h", "6h", "8h", "12h", "24h"),
	Placeholder: `"1h"`,
	Nullable:    true,
	Description: "The schedule period. It determines how often the import runs automatically. If it is null, the scheduler is disabled, and no automatic execution is executed.",
}

var setSchedulerPeriodParameter = types.Property{
	Name:           "schedulePeriod",
	Type:           types.Text().WithValues("5m", "15m", "30m", "1h", "2h", "3h", "6h", "8h", "12h", "24h"),
	CreateRequired: true,
	Placeholder:    `"1h"`,
	Nullable:       true,
	Description:    "The schedule period. It determines how often the execution runs automatically. If it is null, the scheduler will be disabled, and no automatic execution will be executed.",
}

func init() {

	filterParameter := types.Property{
		Name:        "filter",
		Type:        filterType,
		Nullable:    true,
		Placeholder: `{ "logical": "and", "conditions": [ { "property": "country", "operator": "is", "values": [ "US" ] } ] }`,
		Description: "The filter applied to the app users. If it's not null, only the users that match the filter will be included.\n\n" +
			"See the [filters documentation](/filters) for more details.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "source-app-actions-on-users",
		Name:        "Source app actions on users",
		Description: "A source app action is an action that extracts user data from an app and loads it into the workspace data warehouse for storage and analysis.",
		Endpoints: []*Endpoint{
			{
				Name:        "Create a source app action on users",
				Description: "Create a source app action on users.",
				Method:      POST,
				URL:         "/v0/actions",
				Parameters: []types.Property{
					{
						Name:           "name",
						Type:           types.Text().WithCharLen(60),
						CreateRequired: true,
						Placeholder:    `"HubSpot"`,
						Description:    "The action's name.",
					},
					{
						Name:           "connection",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "230527183",
						Description:    "The connection for which the action should be created. It should be a source app connection.",
					},
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicate if the action is enabled once created.",
					},
					filterParameter,
					{
						Name:           "inSchema",
						Type:           types.Parameter("schema"),
						CreateRequired: true,
						Placeholder:    `{...}`,
					},
					{
						Name:           "outSchema",
						Type:           types.Parameter("schema"),
						CreateRequired: true,
						Placeholder:    `{...}`,
					},
					{
						Name:           "transformation",
						Type:           types.Parameter("transformation"),
						CreateRequired: true,
						Placeholder:    `{...}`,
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Placeholder: "705981339",
							Description: "The ID of the action.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, ConnectionNotExist, "connection does not exist"},
					{422, ConnectorNotExist, "connector does not exist"},
					{422, UnsupportedLanguage, "transformation language is not supported"},
				},
			},
			{
				Name:        "Update a source app action on users",
				Description: "Update a source app action on users.",
				Method:      PUT,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source app action on users.",
					},
					{
						Name:           "name",
						Type:           types.Text().WithCharLen(60),
						CreateRequired: true,
						Placeholder:    `"HubSpot"`,
						Description:    "The action's name.",
					},
					{
						Name:        "enabled",
						Type:        types.Boolean(),
						Placeholder: "true",
						Description: "Indicates if the action is enable.",
					},
					filterParameter,
					{
						Name:        "inSchema",
						Type:        types.Parameter("schema"),
						Placeholder: `{...}`,
					},
					{
						Name:        "outSchema",
						Type:        types.Parameter("schema"),
						Placeholder: `{...}`,
					},
					{
						Name:        "transformation",
						Type:        types.Parameter("transformation"),
						Placeholder: `{...}`,
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
					{422, UnsupportedLanguage, "transformation language is not supported"},
				},
			},
			{
				Name:        "Set the status",
				Description: "Sets the status of a source app action.",
				Method:      PUT,
				URL:         "/v0/actions/:id/status",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the action.",
					},
					{
						Name:           "enabled",
						Type:           types.Boolean(),
						CreateRequired: true,
						Placeholder:    "true",
						Description:    "Indicates if the action is enabled..",
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
				},
			},
			{
				Name:        "Set schedule period",
				Description: "Sets the frequency at which a source app action imports users. Both the action and its connection must be enabled for the import to run as scheduled. If the period is null, the scheduler will be disabled.",
				Method:      PUT,
				URL:         "/v0/actions/:id/schedule",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source app action on users.",
					},
					setSchedulerPeriodParameter,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
				},
			},
			{
				Name:        "Get a source app action on users",
				Description: "Get a source app action on users.",
				Method:      GET,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						Placeholder:    "705981339",
						Description:    "The ID of the source app action on users.",
						CreateRequired: true,
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Placeholder: "705981339",
							Description: "The ID of the source action.",
						},
						{
							Name:        "connection",
							Type:        types.Int(32),
							Placeholder: "1371036433",
							Description: "The ID of the action's connection.",
						},
						{
							Name:        "name",
							Type:        types.Text().WithCharLen(60),
							Placeholder: `"HubSpot"`,
							Description: "The action's name.",
						},
						{
							Name:        "enabled",
							Type:        types.Boolean(),
							Placeholder: "true",
							Description: "Indicates if the action is enable.",
						},
						{
							Name:        "running",
							Type:        types.Boolean(),
							Placeholder: "false",
							Description: "Indicates if the action is running.",
						},
						scheduleStartParameter,
						schedulePeriodParameter,
						filterParameter,
						{
							Name:        "inSchema",
							Type:        types.Parameter("schema"),
							Placeholder: `{...}`,
						},
						{
							Name:        "outSchema",
							Type:        types.Parameter("schema"),
							Placeholder: `{...}`,
						},
						{
							Name:        "transformation",
							Type:        types.Parameter("transformation"),
							Placeholder: `{...}`,
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
				},
			},
			{
				Name: "Import users from an app",
				Description: "Starts a source app action execution to import its users into the workspace’s data warehouse, applying the action's filter and transformation.\n\n" +
					"It returns immediately without waiting for the execution to complete. To track the progress, call the [`/executions/:id`](/api/executions) endpoint using the returned execution ID.\n\n" +
					"Both the action and its connection must be enabled.",
				Method: POST,
				URL:    "/v0/actions/:id/exec",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source app action on users.",
					},
					{
						Name:           "reload",
						Type:           types.Boolean(),
						CreateRequired: false,
						Placeholder:    "false",
						Description: " Indicates whether the users should be re-imported from scratch. " +
							"If set to false or omitted, only new users and those modified since the last import are processed.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "id",
							Type:        types.Int(32),
							Placeholder: "609461413",
							Description: "The ID of the started execution.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
					{422, ConnectionDisabled, "connection is disabled"},
					{422, ActionDisabled, "action is disabled"},
					{422, ExecutionInProgress, "action is already in progress"},
					{422, InspectionMode, "data warehouse is in inspection mode"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
				},
			},
			{
				Name:        "Delete a source app action on users",
				Description: "Delete a source app action on users.",
				Method:      DELETE,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the source app action on users.",
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
				},
			},
		},
	})
}
