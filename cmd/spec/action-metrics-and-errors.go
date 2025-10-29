// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package spec

import (
	"github.com/meergo/meergo/core/types"
)

func init() {

	actionsParameter := types.Property{
		Name:           "actions",
		Prefilled:      "705981339,1360924687",
		Type:           types.Array(types.Int(32)),
		CreateRequired: true,
		Description: "The IDs of the actions for which metrics should be returned. At least one action must be provided. The request does not fail if an action does not exist within the workspace.\n\n" +
			"The actions can be specified in the query string in this way:\n\n`actions=705981339&actions=1360924687`",
	}
	responseParameters := []types.Property{
		{
			Name:        "start",
			Type:        types.DateTime(),
			Prefilled:   `"2025-01-02T09:00:00"`,
			Description: "The starting date in the `YYYY-MM-DDTHH:mm:ss` ISO 8601 format.",
		},
		{
			Name:        "end",
			Type:        types.DateTime(),
			Prefilled:   `"2025-01-02T18:00:00"`,
			Description: "The ending date in the `YYYY-MM-DDTHH:mm:ss` ISO 8601 format.",
		},
		{
			Name:        "passed",
			Type:        types.Array(types.Array(types.Int(32))),
			Prefilled:   `[ [ 6029, 6029, 5974, 5974, 5974, 5974 ] ]`,
			Description: "The number of users or events that successfully passed each step on each hour within the start and end dates.",
		},
		{
			Name:        "failed",
			Type:        types.Array(types.Array(types.Int(32))),
			Prefilled:   `[ [ 0, 0, 55, 0, 0, 0 ] ]`,
			Description: "The number of users or events that failed at each step on each hour within the start and end dates.",
		},
	}
	stepType := types.Text().WithValues("Receive", "InputValidation", "Filter", "Transformation", "OutputValidation", "Finalize")

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "actions/metrics-and-errors",
		Name:        "Metrics and errors",
		Description: "Metrics and errors related to the actions, including both import/export processes and the sending or receiving of events within the pipelines.",
		Endpoints: []*Endpoint{
			{
				Name:        "Get metrics per dates",
				Description: "Retrieves metrics for actions aggregated by day for a time interval between specified start and end dates.",
				Method:      GET,
				URL:         "/v1/actions/metrics/dates/:start/:end",
				Parameters: []types.Property{
					{
						Name:           "start",
						Type:           types.Date(),
						CreateRequired: true,
						Description: "The starting date in ISO 8601 format. It must be earlier than the end date.\n\n" +
							"Additionally, the date must be no earlier than 1970-01-01T00:00:00 and no later than the 2262-04-11T23:47:00.",
					},
					{
						Name:           "end",
						Type:           types.Date(),
						CreateRequired: true,
						Description: "The ending date in ISO 8601 format. It must be later than the start date.\n\n" +
							"Additionally, the date must be no earlier than 1970-01-01T00:00:00 and no later than the 2262-04-11T23:47:00.",
					},
					actionsParameter,
				},
				Response: &Response{
					Parameters: responseParameters,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
			{
				Name:        "Get metrics per day",
				Description: "Retrieves metrics for actions for a specified number of days up to the current time.",
				Method:      GET,
				URL:         "/v1/actions/metrics/days/:days",
				Parameters: []types.Property{
					{
						Name:           "days",
						Type:           types.Int(32).WithIntRange(1, 30),
						CreateRequired: true,
						Description:    "The number of days, ranging from 1 to 30. By default, it is 30.",
					},
					actionsParameter,
				},
				Response: &Response{
					Parameters: responseParameters,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
			{
				Name:        "Get metrics per hour",
				Description: "Retrieves metrics for actions for a specified number of hours up to the current time.",
				Method:      GET,
				URL:         "/v1/actions/metrics/hours/:hours",
				Parameters: []types.Property{
					{
						Name:           "hours",
						Type:           types.Int(32).WithIntRange(1, 48),
						CreateRequired: true,
						Description:    "The number of hours, ranging from 1 to 48. By default, it is 48.",
					},
					actionsParameter,
				},
				Response: &Response{
					Parameters: responseParameters,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
			{
				Name:        "Get metrics per minute",
				Description: "Retrieves metrics for actions for a specified number of minutes up to the current time.",
				Method:      GET,
				URL:         "/v1/actions/metrics/minutes/:minutes",
				Parameters: []types.Property{
					{
						Name:           "minutes",
						Type:           types.Int(32).WithIntRange(1, 60),
						CreateRequired: true,
						Description:    "The number of minutes, ranging from 1 to 60. By default, it is 60.",
					},
					actionsParameter,
				},
				Response: &Response{
					Parameters: responseParameters,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
			{
				Name:        "Get errors",
				Description: "Retrieves errors for actions for a time interval between specified start and end dates.",
				Method:      GET,
				URL:         "/v1/actions/errors/:start/:end",
				Parameters: []types.Property{
					{
						Name:           "start",
						Type:           types.DateTime(),
						CreateRequired: true,
						Description: "The starting date in ISO 8601 format. It must be earlier than the end date.\n\n" +
							"Additionally, the date must be no earlier than 1970-01-01T00:00:00 and no later than the 2262-04-11T23:47:00.",
					},
					{
						Name:           "end",
						Type:           types.DateTime(),
						CreateRequired: true,
						Description: "The ending date in ISO 8601 format. It must be later than the start date.\n\n" +
							"Additionally, the date must be no earlier than 1970-01-01T00:00:00 and no later than the 2262-04-11T23:47:00.",
					},
					{
						Name:           "actions",
						Prefilled:      "705981339,1360924687",
						Type:           types.Array(types.Int(32)),
						CreateRequired: true,
						Description: "The IDs of the actions for which errors should be returned. At least one action must be provided. The request does not fail if an action does not exist within the workspace.\n\n" +
							"The actions can be specified in the query string in this way:\n\n`actions=705981339&actions=1360924687`",
					},
					{
						Name:        "step",
						Type:        stepType,
						Description: "The pipeline step for which errors should be returned. If no step is specified, errors will be returned for all steps.",
					},
					{
						Name:        "first",
						Type:        types.Int(32).WithIntRange(0, 9999),
						Description: "The number of errors to skip before starting to return errors. It must be between 0 and 9999, with a default value of 0.",
					},
					{
						Name:        "limit",
						Type:        types.Int(32).WithIntRange(1, 100),
						Description: "The maximum number of errors to return. It must be between 1 and 100, with a default value of 100.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name: "errors",
							Type: types.Array(types.Object([]types.Property{
								{
									Name:        "action",
									Type:        types.Int(32),
									Description: "The action in which the error occurred.",
								},
								{
									Name:        "step",
									Type:        stepType,
									Description: "The step where the error occurred.",
								},
								{
									Name:        "count",
									Type:        types.Int(32),
									Description: "The number of occurrences of the error within the specified period.",
								},
								{
									Name:        "message",
									Type:        types.Text(),
									Description: "The error message.",
								},
								{
									Name:        "lastOccurred",
									Type:        types.DateTime(),
									Description: "The date of the last occurrence of the error within the specified period.",
								},
							})),
							Description: "The errors that occurred during the specified period for the provided actions.",
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
