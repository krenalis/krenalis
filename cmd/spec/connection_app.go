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
		ID:          "connection-app",
		Name:        "Connection app",
		Description: "A connection enables Meergo to retrieve customer and event data from an external source location or send them to an external destination location.",
		Endpoints: []*Endpoint{
			{
				Name:        "List the app users",
				Description: "Returns the users of an app.",
				Method:      POST,
				URL:         "/v0/connections/:id/users",
				Parameters: []types.Property{
					{
						Name:           "schema",
						Type:           types.Parameter("Schema"),
						Placeholder:    `{ ... }`,
						UpdateRequired: true,
						Description:    "The schema that the returned users must satisfy.",
					},
					{
						Name:        "cursor",
						Type:        types.Text(),
						Placeholder: `"..."`,
						Description: "The cursor used to fetch the next set of users. If empty, it returns the first set of users.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "users",
							Type:        types.Array(types.Map(types.JSON())),
							Placeholder: `{ { "id": "30cae655-4f86-4696-ae2c-2c9105b282ba" } }`,
							Description: "The app users from the provided cursor.",
						},
						{
							Name:        "cursor",
							Type:        types.Text(),
							Placeholder: `""`,
							Description: "The cursor to be passed to retrieve the next set of users. If there are no more users, the cursor will be empty.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "connection does not exist"},
					{422, SchemaNotAligned, "schema is not aligned with the app's source schema"},
				},
			},
			{
				Name: "Preview send event",
				Description: "Returns a preview of an event as it would be sent to the connection's app.\n\n" +
					"The connection must be a destination app connection, and it is expected to have the provided event type.",
				Method: POST,
				URL:    "/v0/connections/:id/preview-send-event",
				Parameters: []types.Property{
					{
						Name:           "evenType",
						Type:           types.Text(),
						Placeholder:    `"addToCart"`,
						UpdateRequired: true,
						Description:    "The event type.",
					},
					{
						Name:           "event",
						Type:           types.JSON(),
						Placeholder:    `{...}`,
						UpdateRequired: true,
						Description:    "The event that would be sent. It must conform to the schema of the event type.",
					},
					{
						Name:        "transformation",
						Type:        types.Parameter("dataTransformation"),
						Placeholder: `{...}`,
						Description: "The transformation.",
					},
					{
						Name:        "outSchema",
						Type:        types.Parameter("Schema"),
						Placeholder: `{...}`,
						Description: "The output schema.",
					},
				},
				Errors: []Error{
					{404, NotFound, "connection does not exist"},
					{422, EventTypeNotExist, "connection does not have the event type"},
					{422, SchemaNotAligned, "output schema is not compatible with the event type's schema"},
					{422, TransformationFailed, "transformation failed"},
					{422, UnsupportedLanguage, "transformation language is not supported"},
				},
			},
			{
				Name:        "Get the user schemas",
				Description: "Returns the source and destination user schema of an app connection.",
				Method:      GET,
				URL:         "/v0/connections/:id/schemas/user",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						Placeholder:    "1371036433",
						UpdateRequired: true,
						Description:    "The ID of the app connection.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "sourceSchema",
							Type:        types.Parameter("Schema"),
							Placeholder: `{ ... }`,
							Description: "The source user schema of the app connection.",
						},
						{
							Name:        "destinationSchema",
							Type:        types.Parameter("Schema"),
							Placeholder: `{ ... }`,
							Description: "The destination user schema of the app connection.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "connection does not exist"},
				},
			},
			{
				Name:        "Get the group schemas",
				Description: "Returns the source and destination group schema of an app connection.",
				Method:      GET,
				URL:         "/v0/connections/:id/schemas/group",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						Placeholder:    "1371036433",
						UpdateRequired: true,
						Description:    "The ID of the app connection.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "sourceSchema",
							Type:        types.Parameter("Schema"),
							Placeholder: `{ ... }`,
							Description: "The source group schema of the app connection.",
						},
						{
							Name:        "destinationSchema",
							Type:        types.Parameter("Schema"),
							Placeholder: `{ ... }`,
							Description: "The destination group schema of the app connection.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "connection does not exist"},
				},
			},
			{
				Name:        "Get an event type schema",
				Description: "Returns the schema of an event type within a destination app connection.",
				Method:      GET,
				URL:         "/v0/connections/:id/schemas/event/:type",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						Placeholder:    "1371036433",
						UpdateRequired: true,
						Description:    "The ID of the destination app connection.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "schema",
							Type:        types.Parameter("Schema"),
							Placeholder: `{ ... }`,
							Description: "The schema of the event type.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "connection does not exist"},
				},
			},
		},
	})

}
