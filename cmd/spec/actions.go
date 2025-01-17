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
	Description: "The start time of the schedule in minutes, counting from 00:00. It specifies the minute when the first scheduled execution of the day begins. " +
		"Subsequent executions occur based on the interval defined by the scheduler period. If the scheduler is disabled, this value is null.",
}

var importSchedulePeriodParameter = types.Property{
	Name:        "schedulePeriod",
	Type:        types.Text().WithValues("5m", "15m", "30m", "1h", "2h", "3h", "6h", "8h", "12h", "24h"),
	Placeholder: `"1h"`,
	Nullable:    true,
	Description: "The schedule period, which determines how often the import runs automatically. If it is null, the scheduler is disabled, and no automatic executions will occur.\n\n" +
		"To change the schedule period, use the [Set schedule period](/api/actions#set-schedule-period) endpoint.",
}

var exportSchedulePeriodParameter = types.Property{
	Name:        "schedulePeriod",
	Type:        types.Text().WithValues("5m", "15m", "30m", "1h", "2h", "3h", "6h", "8h", "12h", "24h"),
	Placeholder: `"1h"`,
	Nullable:    true,
	Description: "The schedule period, which determines how often the export runs automatically. If it is null, the scheduler is disabled, and no automatic executions will occur.\n\n" +
		"To change the schedule period, use the [Set schedule period](/api/actions#set-schedule-period) endpoint.",
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

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "actions",
		Name:        "Actions",
		Description: "",
		Endpoints: []*Endpoint{
			{
				Name:        "Set status",
				Description: "Sets the status of an action.",
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
						Description:    "Indicates if the action is enabled.",
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
				},
			},
			{
				Name:        "Set schedule period",
				Description: "Sets the frequency at which an action imports users. The action must be enabled for the import to run as scheduled. If the period is null, the scheduler will be disabled.",
				Method:      PUT,
				URL:         "/v0/actions/:id/schedule",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the action on users.",
					},
					setSchedulerPeriodParameter,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
				},
			},
			{
				Name: "Execute action",
				Description: "Starts an action execution to import its users into the data warehouse or export the user in the data warehouse to the app, applying the action's filter and transformation.\n\n" +
					"It returns immediately without waiting for the execution to complete. To track the progress, call the [`/executions/:id`](/api/executions) endpoint using the returned execution ID.\n\n" +
					"The action must be enabled.",
				Method: POST,
				URL:    "/v0/actions/:id/exec",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the action on users.",
					},
					{
						Name:           "reload",
						Type:           types.Boolean(),
						CreateRequired: false,
						Placeholder:    "false",
						Description: " Indicates whether the users should be re-imported or re-exported from scratch. " +
							"If set to false or omitted, only new users and those modified since the last execution are processed.",
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
					{422, ActionDisabled, "action is disabled"},
					{422, ExecutionInProgress, "action is already in progress"},
					{422, InspectionMode, "data warehouse is in inspection mode"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
				},
			},
			{
				Name:        "Delete action",
				Description: "Delete an action.",
				Method:      DELETE,
				URL:         "/v0/actions/:id",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the action to delete.",
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
