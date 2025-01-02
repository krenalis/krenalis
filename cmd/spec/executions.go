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
		ID:          "executions",
		Name:        "Executions",
		Description: "The executions refer to action executions. User actions are automatically triggered by the scheduler or can be started by calling the specific endpoint for the action to execute.",
		Endpoints: []*Endpoint{
			{
				Name:        "List all action executions",
				Description: "Returns all action executions.",
				Method:      GET,
				URL:         "/v0/executions",
				Response: &Response{
					Parameters: []types.Property{
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
					},
				},
			},
		},
	})

}
