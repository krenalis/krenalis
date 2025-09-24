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
		ID:          "app-connections",
		Name:        "App connections",
		Description: "These endpoints are specific to app connections.",
		Endpoints: []*Endpoint{
			{
				Name: "Retrieve app users",
				Description: "Retrieves users directly from the app. These are the users as they appear in the app.\n\n" +
					"For users that have already been imported into the workspace, refer to the [Users](users) endpoints.",
				Method: GET,
				URL:    "/v1/connections/:id/users",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						Prefilled:      "1371036433",
						CreateRequired: true,
						Description:    "The ID of the app connection from which to read users.",
					},
					{
						Name:           "schema",
						Type:           types.Parameter("schema"),
						Prefilled:      `{...}`,
						CreateRequired: true,
						Description:    "The schema that the returned users must satisfy",
					},
					{
						Name:        "filter",
						Type:        filterType,
						Description: "The filter applied to the users. If not empty, only users that match the filter will be returned.",
					},
					{
						Name:        "cursor",
						Type:        types.Text(),
						Prefilled:   `...`,
						Description: "The cursor used to fetch the next set of users. If empty, it returns the first set of users.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "users",
							Type:        types.Array(types.Map(types.JSON())),
							Prefilled:   `[ { "id": "30cae655-4f86-4696-ae2c-2c9105b282ba" } ]`,
							Description: "The app users from the provided cursor.",
						},
						{
							Name:        "cursor",
							Type:        types.Text(),
							Prefilled:   `""`,
							Description: "The cursor to be passed to retrieve the next set of users. If there are no more users, the cursor will be empty.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
					{422, SchemaNotAligned, "schema is not aligned with the app's source schema"},
				},
			},
			{
				Name:        "Preview send event",
				Description: "Returns a preview of an event as it would be sent to an app, but no event is actually sent.",
				Method:      POST,
				URL:         "/v1/connections/:id/preview-send-event",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						Prefilled:      "1371036433",
						CreateRequired: true,
						Description:    "The ID of the destination app connection through which the event would be sent.",
					},
					{
						Name:           "type",
						Type:           types.Text(),
						Prefilled:      `"addToCart"`,
						CreateRequired: true,
						Description:    "The ID of the event type to be sent. It must be one of the event types supported by the app connection.",
					},
					{
						Name:           "event",
						Type:           types.JSON(),
						Prefilled:      `{...}`,
						CreateRequired: true,
						Description: "The event (as it would be received from an SDK connection) that is sent to the app. " +
							"It must adhere to the [event schema](events#get-event-schema).",
					},
					{
						Name:        "transformation",
						Type:        types.Parameter("dataTransformation"),
						Prefilled:   `{...}`,
						Description: "The transformation.",
					},
					{
						Name:        "outSchema",
						Type:        types.Parameter("schema"),
						Prefilled:   `{...}`,
						Description: "The output schema.",
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
					{422, EventTypeNotExist, "connection does not have the event type"},
					{422, InvalidEvent, "event is invalid"},
					{422, SchemaNotAligned, "output schema is not compatible with the event type's schema"},
					{422, TransformationFailed, "transformation failed"},
					{422, UnsupportedLanguage, "transformation language is not supported"},
				},
			},
			{
				Name:        "Get user schemas",
				Description: "Returns the source and destination user schema of a connection. The connection must be an app connection that supports users.",
				Method:      GET,
				URL:         "/v1/connections/:id/schemas/user",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						Prefilled:      "1371036433",
						CreateRequired: true,
						Description:    "The ID of the app connection. It must support users.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name: "schemas",
							Type: types.Object([]types.Property{
								{
									Name:        "source",
									Type:        types.Parameter("schema"),
									Prefilled:   `{ ... }`,
									Description: "The source schema.",
								},
								{
									Name:        "destination",
									Type:        types.Parameter("schema"),
									Prefilled:   `{ ... }`,
									Description: "The destination schema. It is null for source connections.",
								},
							}),
							Prefilled:   `...`,
							Description: "The user schemas of the app connection.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
				},
			},
			{
				Name:        "Get event schema",
				Description: "Returns the schema for a specified event type in a connection. The connection must be a destination app connection that supports events.",
				Method:      GET,
				URL:         "/v1/connections/:id/schemas/event",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						Prefilled:      "1371036433",
						CreateRequired: true,
						Description:    "The ID of the destination app connection. It must support events.",
					},
					{
						Name:           "type",
						Type:           types.Text(),
						Prefilled:      `page_view`,
						CreateRequired: true,
						Description:    "The ID of the event type.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "schema",
							Type:        types.Parameter("schema"),
							Prefilled:   `{ ... }`,
							Description: "The schema of the event type. It is null if the event type does not have a schema.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
					{404, NotFound, "event type does not exist"},
				},
			},
		},
	})

}
