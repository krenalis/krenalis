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
		ID:   "action-types",
		Name: "Action types",
		Description: "Action types are the types of actions that can be created for a specific connection. They differ based on the target they operate on:\n" +
			"* `Events` for actions that receive or send events\n" +
			"* `Users` for actions that import or export users\n" +
			"* `Groups` for actions that import or export groups\n\n" +
			"If the connection is a destination, the actions on events may also vary depending on the type of event.",
		Endpoints: []*Endpoint{
			{
				Name:        "List action types for a connection",
				Description: "Retrieves the action types for a connection.",
				Method:      GET,
				URL:         "/v0/connections/:id/action-types",
				Parameters: []types.Property{
					{
						Name:           ":id",
						Type:           types.Int(32),
						CreateRequired: true,
						Description:    "The ID of the connection for which the action types are returned.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name: "actionTypes",
							Type: types.Array(types.Object([]types.Property{
								{
									Name:        "name",
									Type:        types.Text(),
									Placeholder: `"Export contacts"`,
									Description: "The name of the action type.",
								},
								{
									Name:        "description",
									Type:        types.Text(),
									Placeholder: `"Export the users  as contacts to Mailchimp"`,
									Description: "The description of the action type.",
								},
								{
									Name:        "target",
									Type:        types.Text().WithValues("Events", "Users", "Groups"),
									Placeholder: `"Users"`,
									Description: "The target of the action.",
								},
								{
									Name:        "eventType",
									Type:        types.Text(),
									Placeholder: `null`,
									Nullable:    true,
									Description: "The event type of the action. It is null if target is not `\"Users\"` or `\"Groups\"`.",
								},
							})),
							Placeholder: "...",
							Description: "The action types of the connection.",
						},
					},
				},
			},
		},
	})

}
