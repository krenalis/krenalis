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

var runningParameter = types.Property{
	Name:        "running",
	Type:        types.Boolean(),
	Placeholder: "false",
	Description: "Indicates if the action is running.",
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

func init() {

	executionParameters := []types.Property{
		{
			Name:        "id",
			Type:        types.Int(32),
			Placeholder: "609461413",
			Description: "The ID of the execution.",
		},
		{
			Name:        "action",
			Type:        types.Int(32),
			Placeholder: "705981339",
			Description: "The ID of the executed action.",
		},
		{
			Name:        "startTime",
			Type:        types.DateTime(),
			Placeholder: `"2024/11/27T18:22:47.937Z"`,
			Description: "The start time in ISO 8601 format.",
		},
		{
			Name:        "endTime",
			Type:        types.DateTime(),
			Nullable:    true,
			Placeholder: `"2024/11/27T18:49:07.150Z"`,
			Description: "The end time in ISO 8601 format. It is null if the execution has not yet finished.",
		},
		{
			Name:        "passed",
			Type:        types.Int(32),
			Placeholder: "22947",
			Description: "The number of passed users or events.",
		},
		{
			Name:        "failed",
			Type:        types.Int(32),
			Placeholder: "172",
			Description: "The number of failed users or events.",
		},
		{
			Name:        "error",
			Type:        types.Text(),
			Placeholder: `""`,
			Description: "An error occurred during execution, causing it to stop prematurely. It is empty if the execution has not yet finished or if no error occurred.",
		},
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "actions",
		Name: "Actions",
		Description: "Actions represent the operations that can be performed on [connections](/api/connections), " +
			"such as importing and exporting users or storing and sending events.\n\n" +
			"This section documents the endpoints common to various types of actions. " +
			"For creating, updating, and retrieving an action, refer to the specific sections for each type of action.",
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
						Description:    "The ID of the action that imports or exports users.",
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
				Description: "Sets the frequency at which an action imports or exports users. The action must be enabled for the execution to run as scheduled. If the period is null, the scheduler will be disabled.",
				Method:      PUT,
				URL:         "/v0/actions/:id/schedule",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the action that imports or exports users.",
					},
					{
						Name:           "period",
						Type:           types.Text().WithValues("5m", "15m", "30m", "1h", "2h", "3h", "6h", "8h", "12h", "24h"),
						CreateRequired: true,
						Placeholder:    `"1h"`,
						Nullable:       true,
						Description:    "The schedule period. It determines how often the execution runs automatically. If it is null, the scheduler will be disabled, and no automatic execution will be executed.",
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "action does not exist"},
				},
			},
			{
				Name: "Execute action",
				Description: "Starts an action execution to import its users into the data warehouse or export the user in the data warehouse to the app, applying the action's filter and transformation.\n\n" +
					"It returns immediately without waiting for the execution to complete. To track the progress, call the [Get execution](#get-execution) endpoint using the returned execution ID.\n\n" +
					"The action must be enabled.",
				Method: POST,
				URL:    "/v0/actions/:id/exec",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "705981339",
						Description:    "The ID of the action that imports or exports users.",
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
				Name: "List all executions",
				Description: "Returns all action executions.\n\n" +
					"Actions executions are automatically triggered by the scheduler or can be started by calling the specific endpoint for the action.",
				Method: GET,
				URL:    "/v0/actions/executions",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "executions",
							Type:        types.Array(types.Object(executionParameters)),
							Description: "The action executions.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
			{
				Name: "Get execution",
				Description: "Returns an action execution.\n\n" +
					"Actions executions are automatically triggered by the scheduler or can be started by calling the specific endpoint for the action.",
				Method: GET,
				URL:    "/v0/actions/executions/:id",
				Response: &Response{
					Parameters: executionParameters,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
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
