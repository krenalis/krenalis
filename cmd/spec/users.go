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

	eventsParameter := types.Array(types.Object(append([]types.Property{
		{Name: "user", Type: types.UUID(), ReadOptional: true, Prefilled: `"9102d2a1-0714-4c13-bafd-8a38bc3d0cff"`},
		{Name: "connectionId", Type: types.Int(32), Prefilled: "1371036433"},
	}, eventGetProperties...)))

	identityType := types.Object([]types.Property{
		{
			Name:        "connection",
			Type:        types.Int(32),
			Description: "The ID of the connection through which the identity was read.",
		},
		{
			Name:        "action",
			Type:        types.Int(32),
			Description: "The ID of the action through which the identity was imported.",
		},
		{
			Name: "id",
			Type: types.Text(),
			Description: "The unique identifier that represents the identity in the source from which it was retrieved. This field is empty for anonymous users. Specifically:\n" +
				"* For SDK and webhook connections, it corresponds to the User ID of the event. This field is empty for identities imported from anonymous events.\n" +
				"* For an API connection, it is the value used to identify the user in the API. For example, in HubSpot's API, it corresponds to the HubSpot ID.\n" +
				"* For a database or file storage connection, it is the value in the column designated as the identity.",
		},
		{
			Name:        "anonymousIds",
			Type:        types.Array(types.Text()),
			Nullable:    true,
			Description: "The anonymous IDs of the identity. It is null for identities not imported from events.",
		},
		{
			Name:        "lastChangeTime",
			Type:        types.DateTime(),
			Description: "The identity's most recent change time.",
		},
	})

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "users",
		Name:        "Users",
		Description: "Users are those imported from external sources and events, unified through identity resolution, and stored in the workspace's data warehouse.",
		Endpoints: []*Endpoint{
			{
				Name: "Retrieve all users",
				Description: "Retrieves users stored in the workspace's data warehouse, up to a maximum number of users defined by `limit`. You must specify which properties to include. " +
					"If a filter is provided, only users that match the filter criteria will be returned.",
				Method: GET,
				URL:    "/v1/users",
				Parameters: []types.Property{
					{
						Name:           "properties",
						Type:           types.Array(types.Text()),
						CreateRequired: true,
						Prefilled:      `first_name,last_name`,
						Description: "The names of the properties to return. At least one property must be included.\n\n" +
							"The properties can be specified in query string in this way:\n" +
							"```\nproperties=first_name&properties=last_name&properties=email\n```",
					},
					{
						Name: "filter",
						Type: filterType,
						Description: "The filter applied to the users. Only the users that match the filter will be returned.\n\n" +
							"It must be encoded in JSON, then escaped for the context of the query string. So, for example, the JSON-encoded filter:\n\n" +
							"`" + `{"logical":"and","conditions":[{"property":"email","operator":"is","values":["my.friend@example.com"]}]}` + "`\n\n" +
							"must then be escaped and passed in the query string as:\n\n" +
							"`filter=%7B%22logical%22%3A%22and%22%2C%22conditions\n%22%3A%5B%7B%22property%22%3A%22email%22%2C%22\n" +
							"operator%22%3A%22is%22%2C%22values%22%3A%5B%22\nmy.friend%40example.com%22%5D%7D%5D%7D`",
					},
					{
						Name:        "order",
						Type:        types.Text(),
						Prefilled:   `email`,
						Description: "The name of the property by which to sort the users to be returned. It can be any property from the user schema with a sortable type, meaning it cannot be of type `json`, `array`, `object`, or `map`.\n\nIf not provided, the users are ordered by their last change time.",
					},
					{
						Name:        "orderDesc",
						Type:        types.Boolean(),
						Prefilled:   `false`,
						Description: "Indicates if the returned users are sorted in descending order; if not `true`, they are sorted in ascending order.",
					},
					{
						Name:        "first",
						Type:        types.Int(32),
						Prefilled:   `0`,
						Description: "The number of users to skip before starting to return results. The default value is 0.",
					},
					{
						Name:        "limit",
						Type:        types.Int(32).WithIntRange(1, 1000),
						Prefilled:   `1000`,
						Description: "The maximum number of users to return. It can be any value between 1 and 1,000. If not provided, the default value is 100.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name: "users",
							Type: types.Array(types.Object([]types.Property{
								{
									Name:        "id",
									Type:        types.UUID(),
									Prefilled:   `"02bc2281-f801-4f59-9c56-b96ff81df84f"`,
									Description: "The ID of the user.",
								},
								{
									Name:      "sourcesLastUpdate",
									Type:      types.DateTime(),
									Prefilled: `"2015-01-21T08:51:32.137139Z"`,
									Description: "The date and time when the user's data was last updated on the sources. It corresponds to the most recent last update of its identities.\n\n" +
										"Its value is independent of which properties were requested.",
								},
								{
									Name:        "traits",
									Type:        types.Parameter("Traits"),
									Prefilled:   `{ "name": "John Walker", "email": "walker@example.com" }`,
									Description: "The traits of the user. Only the properties explicitly requested are included.",
								},
							})),
							Prefilled: "...",
						},
						{
							Name:        "schema",
							Type:        types.Parameter("schema"),
							Prefilled:   `{...}`,
							Description: "The schema of the returned traits. It corresponds to the user schema but includes only the properties that were explicitly requested.",
						},
						{
							Name:        "total",
							Type:        types.Int(32),
							Prefilled:   `803154`,
							Description: "An estimate of the total number of users, without considering `first` and `limit`.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
					{422, PropertyNotExist, "property does not exist in the user schema"},
					{422, OrderNotExist, "order does not exist in schema"},
					{422, OrderTypeNotSortable, "cannot sort by non-sortable type"},
				},
			},
			{
				Name:        "Retrieve user traits",
				Description: "Retrieves, from the workspace's data warehouse, the traits of a user given its ID.",
				Method:      GET,
				URL:         "/v1/users/:id/traits",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.UUID(),
						Prefilled:      `02bc2281-f801-4f59-9c56-b96ff81df84f`,
						CreateRequired: true,
						Description:    "The ID of the user.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "traits",
							Type:        types.Parameter("Traits"),
							Prefilled:   `{ "name": "John Walker", "email": "walker@example.com" }`,
							Description: "The traits of the user.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
					{404, NotFound, "user does not exist"},
				},
			},
			{
				Name: "Retrieve user events",
				Description: "Retrieves the most recent events for a user given their identifier. The events are read from the workspace's data warehouse and are listed in descending order, starting with the most recent ones.\n\n" +
					"This endpoint provides a streamlined, user-focused alternative to the [Retrieve all events](events#retrieve-all-events) endpoint.",
				Method: GET,
				URL:    "/v1/users/:id/events",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.UUID(),
						Prefilled:      `02bc2281-f801-4f59-9c56-b96ff81df84f`,
						CreateRequired: true,
						Description:    "The ID of the user.",
					},
					{
						Name:           "properties",
						Type:           types.Array(types.Text()),
						Prefilled:      `timestamp,event`,
						CreateRequired: true,
						Description: "The names of the event properties to return. At least one property must be included.\n\n" +
							"The properties can be specified in the query string in two ways:\n" +
							"* `properties=timestamp,event`\n* `properties=timestamp&properties=event`",
					},
					{
						Name:        "limit",
						Type:        types.Int(32).WithIntRange(1, 1000),
						Description: "The maximum number of events to return. It can be any value between 1 and 1,000. If not provided, the default value is 100.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "events",
							Type:        eventsParameter,
							Prefilled:   `...`,
							Description: "The most recent events of the user. An empty array is returned if no events are available or if the specified user does not exist.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
				},
			},
			{
				Name: "Retrieve user identities",
				Description: "Retrieves, from the workspace's data warehouse, the identities of a user given its identifier.\n\n" +
					"Identities are sorted by last change time, in descending order, so the most recently changed identities are returned first.\n\n" +
					"If the user does not exist, no identities are returned.",
				Method: GET,
				URL:    "/v1/users/:id/identities",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.UUID(),
						Prefilled:      `86de98fe-8f26-49ac-87dc-8a14997a97d9`,
						CreateRequired: true,
						Description:    "The ID of the user.",
					},
					{
						Name:        "first",
						Type:        types.Int(32),
						Description: "The number of identities to skip before starting to return results. The default value is 0.",
					},
					{
						Name:        "limit",
						Type:        types.Int(32).WithIntRange(1, 1000),
						Description: "The maximum number of identities to return. It can be any value between 1 and 1,000. If not provided, the default value is 100.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "identities",
							Type:        types.Array(identityType),
							Prefilled:   `{ ... }`,
							Description: "The user's identities. It is empty if the user has no identities.",
						},
						{
							Name:        "total",
							Type:        types.Int(32),
							Prefilled:   `12`,
							Description: "The total number of identities.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
				},
			},
		},
	})

}
