//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package spec

import (
	"fmt"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/types"
)

// eventPostContextType is a types.Type representing the context of an event
// passed to the API methods that POST events, i.e. ingestion.
var eventPostContextType = types.Object([]types.Property{
	{
		Name: "app",
		Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text()},
			{Name: "version", Type: types.Text()},
			{Name: "build", Type: types.Text()},
			{Name: "namespace", Type: types.Text()},
		}),
	},
	{
		Name: "browser",
		Type: types.Object([]types.Property{
			{
				Name: "name",
				Type: types.Text(),
				Description: "Name of the browser from which the event originated.\n\n" +
					"Meergo tries to normalize this field and store it with one of those names: `\"Chrome\"`, `\"Safari\"`, `\"Edge\"`, `\"Firefox\"`, `\"Samsung Internet\"` or `\"Opera\"`. Otherwise, if the passed browser name cannot be normalized, it is set to `\"Other\"` and the passed name is stored — as is — into `context.browser.other`.",
			},
			{
				Name:        "version",
				Type:        types.Text(),
				Description: "Version of the browser from which the event originated.",
			},
		}),
		Description: "Browser from which the event originates.\n\n" +
			"If not explicitly provided, Meergo will attempt to infer it from the User Agent.\n\n" +
			"To intentionally leave the event browser unspecified (e.g., for server-generated events where the browser is not applicable), do **not** set a value for `context.browser` and set `context.userAgent` to `\"N/A\"`.",
	},
	{
		Name: "campaign",
		Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text()},
			{Name: "source", Type: types.Text()},
			{Name: "medium", Type: types.Text()},
			{Name: "term", Type: types.Text()},
			{Name: "content", Type: types.Text()},
		}),
	},
	{
		Name: "device",
		Type: types.Object([]types.Property{
			{Name: "id", Type: types.Text()},
			{Name: "advertisingId", Type: types.Text()},
			{Name: "adTrackingEnabled", Type: types.Boolean()},
			{Name: "manufacturer", Type: types.Text()},
			{Name: "model", Type: types.Text()},
			{Name: "name", Type: types.Text()},
			{Name: "type", Type: types.Text()},
			{Name: "token", Type: types.Text()},
		}),
	},
	{
		Name: "ip",
		Type: types.Inet(),
		Description: "IP is the IP address associated with the event.\n\n" +
			"If `context.ip` is explicitly set, its value will be used. Otherwise, the IP will be inferred from the HTTP request of the event.\n\n" +
			"To deliberately avoid associating any IP with the event, set `context.ip` to `0.0.0.0`.\n" +
			"This can be useful for server-side events where the originating IP is irrelevant or where no client IP can be meaningfully assigned (e.g. background jobs, internal system events).",
	},
	{
		Name: "library",
		Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text()},
			{Name: "version", Type: types.Text()},
		}),
	},
	{Name: "locale", Type: types.Text(), ReadOptional: true},
	{
		Name: "location",
		Type: types.Object([]types.Property{
			{Name: "city", Type: types.Text()},
			{Name: "country", Type: types.Text()},
			{Name: "latitude", Type: types.Float(64)},
			{Name: "longitude", Type: types.Float(64)},
			{Name: "speed", Type: types.Float(64)},
		}),
		Description: "Device location from which the event originated. If not explicitly provided with this field, Meergo attempts to determine it automatically based on the event's IP.",
	},
	{
		Name: "network",
		Type: types.Object([]types.Property{
			{Name: "bluetooth", Type: types.Boolean()},
			{Name: "carrier", Type: types.Text()},
			{Name: "cellular", Type: types.Boolean()},
			{Name: "wifi", Type: types.Boolean()},
		}),
		Description: "Network to which the device originating the event is connected.",
	},
	{
		Name: "os",
		Type: types.Object([]types.Property{
			{
				Name: "name",
				Type: types.Text(),
				Description: "Name of the OS from which the event originated.\n\n" +
					"Meergo tries to normalize this field and store it with one of those names: `\"macOS\"`, `\"Android\"`, `\"Windows\"`, `\"iOS\"`, `\"Linux\"` or `\"Chrome OS\"`. Otherwise, if the passed OS name cannot be normalized, it is set to `\"Other\"` and the passed name is stored — as is — into `context.os.other`.",
			},
			{
				Name:        "version",
				Type:        types.Text(),
				Description: "Version of the OS from which the event originated.",
			},
		}),
		Description: "OS of the device from which the event originated.\n\n" +
			"If not explicitly provided, Meergo will attempt to infer it from the User Agent.\n\n" +
			"To intentionally leave the event OS unspecified (e.g., for server-generated events where the OS is irrelevant), do **not** set a value for `context.os` and set `context.userAgent` to `\"N/A\"`.",
	},
	{
		Name: "page",
		Type: types.Object([]types.Property{
			{Name: "path", Type: types.Text()},
			{Name: "referrer", Type: types.Text()},
			{Name: "search", Type: types.Text()},
			{Name: "title", Type: types.Text()},
			{Name: "url", Type: types.Text()},
		}),
	},
	{
		Name: "referrer",
		Type: types.Object([]types.Property{
			{Name: "id", Type: types.Text()},
			{Name: "type", Type: types.Text()},
		}),
	},
	{
		Name: "screen",
		Type: types.Object([]types.Property{
			{
				Name:        "width",
				Type:        types.Int(16),
				Description: "Screen width. Must be in range [1, 32767].",
			},
			{
				Name:        "height",
				Type:        types.Int(16),
				Description: "Screen height. Must be in range [1, 32767].",
			},
			{
				Name:        "density",
				Type:        types.Decimal(3, 2),
				Description: "Screen density. Must be a positive number.",
			},
		}),
		Description: "Screen of the device where the event originated.",
	},
	{
		Name: "session",
		Type: types.Object([]types.Property{
			{Name: "id", Type: types.Int(64)},
			{Name: "start", Type: types.Boolean(), ReadOptional: true},
		}),
	},
	{Name: "timezone", Type: types.Text(), ReadOptional: true},
	{
		Name: "userAgent",
		Type: types.Text(),
		Description: "User Agent is the device identifier from which the event originates.\n\n" +
			"If not provided, Meergo derives it from the event's HTTP request.\n\n" +
			"To explicitly assign no User Agent to the event (useful in cases where events are generated by backend systems or scheduled jobs), set this field to `\"N/A\"` (not applicable).",
	},
})

// eventGetContextType is a types.Type representing the context of an event
// returned by the API methods that GET events.
var eventGetContextType = types.Object([]types.Property{
	{
		Name: "app",
		Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text(), ReadOptional: true},
			{Name: "version", Type: types.Text(), ReadOptional: true},
			{Name: "build", Type: types.Text(), ReadOptional: true},
			{Name: "namespace", Type: types.Text(), ReadOptional: true},
		}),
		ReadOptional: true,
	},
	{
		Name: "browser",
		Type: types.Object([]types.Property{
			{
				Name:         "name",
				Type:         types.Text().WithValues("Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other"),
				ReadOptional: true,
				Description: "Name of the browser from which the event originated.\n\n" +
					"If the value is `\"Other\"`, then the field `other` is populated with the browser name.",
			},
			{
				Name:         "other",
				Type:         types.Text(),
				ReadOptional: true,
				Description:  "Name of the browser in case it is not one of those recognized by Meergo.\n\nThis field is present only when `name` is `\"Other\"`.",
			},
			{
				Name:         "version",
				Type:         types.Text(),
				ReadOptional: true,
				Description:  "Version of the browser from which the event originated.",
			},
		}),
		ReadOptional: true,
	},
	{
		Name: "campaign",
		Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text(), ReadOptional: true},
			{Name: "source", Type: types.Text(), ReadOptional: true},
			{Name: "medium", Type: types.Text(), ReadOptional: true},
			{Name: "term", Type: types.Text(), ReadOptional: true},
			{Name: "content", Type: types.Text(), ReadOptional: true},
		}),
		ReadOptional: true,
	},
	{
		Name: "device",
		Type: types.Object([]types.Property{
			{Name: "id", Type: types.Text(), ReadOptional: true},
			{Name: "advertisingId", Type: types.Text(), ReadOptional: true},
			{Name: "adTrackingEnabled", Type: types.Boolean(), ReadOptional: true},
			{Name: "manufacturer", Type: types.Text(), ReadOptional: true},
			{Name: "model", Type: types.Text(), ReadOptional: true},
			{Name: "name", Type: types.Text(), ReadOptional: true},
			{Name: "type", Type: types.Text(), ReadOptional: true},
			{Name: "token", Type: types.Text(), ReadOptional: true},
		}),
		ReadOptional: true,
	},
	{Name: "ip", Type: types.Inet(), ReadOptional: true},
	{
		Name: "library",
		Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text(), ReadOptional: true},
			{Name: "version", Type: types.Text(), ReadOptional: true},
		}),
		ReadOptional: true,
	},
	{Name: "locale", Type: types.Text(), ReadOptional: true},
	{
		Name: "location",
		Type: types.Object([]types.Property{
			{Name: "city", Type: types.Text(), ReadOptional: true},
			{Name: "country", Type: types.Text(), ReadOptional: true},
			{Name: "latitude", Type: types.Float(64), ReadOptional: true},
			{Name: "longitude", Type: types.Float(64), ReadOptional: true},
			{Name: "speed", Type: types.Float(64), ReadOptional: true},
		}),
		ReadOptional: true,
		Description:  "Device location from which the event originated.",
	},
	{
		Name: "network",
		Type: types.Object([]types.Property{
			{Name: "bluetooth", Type: types.Boolean(), ReadOptional: true},
			{Name: "carrier", Type: types.Text(), ReadOptional: true},
			{Name: "cellular", Type: types.Boolean(), ReadOptional: true},
			{Name: "wifi", Type: types.Boolean(), ReadOptional: true},
		}),
		ReadOptional: true,
		Description:  "Network to which the device originating the event was connected.",
	},
	{
		Name: "os",
		Type: types.Object([]types.Property{
			{
				Name:         "name",
				Type:         types.Text().WithValues("Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other"),
				ReadOptional: true,
				Description: "Name of the OS from which the event originated.\n\n" +
					"If the value is `\"Other\"`, then the field `other` is populated with the OS name.",
			},
			{
				Name:         "other",
				Type:         types.Text(),
				ReadOptional: true,
				Description:  "Name of the OS in case it is not one of those recognized by Meergo.\n\nThis field is present only when `name` is `\"Other\"`.",
			},
			{Name: "version", Type: types.Text(), ReadOptional: true},
		}),
		ReadOptional: true,
		Description:  "OS of the device from which the event originated.",
	},
	{
		Name: "page",
		Type: types.Object([]types.Property{
			{Name: "path", Type: types.Text(), ReadOptional: true},
			{Name: "referrer", Type: types.Text(), ReadOptional: true},
			{Name: "search", Type: types.Text(), ReadOptional: true},
			{Name: "title", Type: types.Text(), ReadOptional: true},
			{Name: "url", Type: types.Text(), ReadOptional: true},
		}),
		ReadOptional: true,
	},
	{
		Name: "referrer",
		Type: types.Object([]types.Property{
			{Name: "id", Type: types.Text(), ReadOptional: true},
			{Name: "type", Type: types.Text(), ReadOptional: true},
		}),
		ReadOptional: true,
	},
	{
		Name: "screen",
		Type: types.Object([]types.Property{
			{
				Name:         "width",
				Type:         types.Int(16),
				ReadOptional: true,
				Description:  "Screen width.",
			},
			{
				Name:         "height",
				Type:         types.Int(16),
				ReadOptional: true,
				Description:  "Screen height.",
			},
			{
				Name:         "density",
				Type:         types.Decimal(3, 2),
				ReadOptional: true,
				Description:  "Screen density.",
			},
		}),
		ReadOptional: true,
		Description:  "Screen of the device where the event originated.",
	},
	{
		Name: "session",
		Type: types.Object([]types.Property{
			{Name: "id", Type: types.Int(64), ReadOptional: true},
			{Name: "start", Type: types.Boolean(), ReadOptional: true},
		}),
		ReadOptional: true,
	},
	{Name: "timezone", Type: types.Text(), ReadOptional: true},
	{Name: "userAgent", Type: types.Text(), ReadOptional: true},
})

// eventGetProperties is a types.Type representing the properties of an event
// returned by the API methods that GET events.
var eventGetProperties = []types.Property{
	{Name: "anonymousId", Type: types.Text(), Prefilled: `"3e93e10e-5ca0-4a8c-bef6-cf9197b37729"`},
	{
		Name:         "channel",
		Type:         types.Text(),
		ReadOptional: true,
		Description:  "The channel from which the event originates.",
	},
	{Name: "category", Type: types.Text(), ReadOptional: true},
	{
		Name:         "context",
		Type:         eventGetContextType,
		ReadOptional: true,
		Description:  "Additional contextual information for the event. If there's no information in the context, this field is not returned.",
	},
	{
		Name:         "event",
		Type:         types.Text(),
		ReadOptional: true,
		Description:  "Field that identifies a track-type event, for example with a value like `Product Purchased`. For any other event type, it is never returned.",
	},
	{
		Name:         "groupId",
		Type:         types.Text(),
		ReadOptional: true,
		Description:  "Group ID related to the event. Returned only for group-type event, and absent for all other event types.",
	},
	{
		Name:        "messageId",
		Type:        types.Text(),
		Description: "ID that uniquely identifies the event for a given connection. Automatically generated by Meergo when the event is received.\n\nUnlike `id`, which uniquely identifies the event across all connections, this identifies the event uniquely only for a specific connection.",
	},
	{
		Name:         "name",
		Type:         types.Text(),
		ReadOptional: true,
		Description:  "It is returned only for screen and page events.",
	},
	{
		Name:         "properties",
		Type:         types.JSON(),
		ReadOptional: true,
		Description:  "A JSON Object representing the properties associated with the event. Returned only for page, screen, and track events.",
	},
	{
		Name:        "receivedAt",
		Type:        types.DateTime(),
		Description: "Event reception timestamp. Automatically set by Meergo when the event is received.",
	},
	{
		Name:        "sentAt",
		Type:        types.DateTime(),
		Description: "Event send timestamp. If not provided with the event, one is automatically generated by Meergo upon event receipt.",
	},
	{
		Name:        "originalTimestamp",
		Type:        types.DateTime(),
		Description: "Original timestamp associated to the event.",
	},
	{Name: "timestamp", Type: types.DateTime()},
	{
		Name:        "traits",
		Type:        types.JSON(),
		Description: "Traits associated with the event in the form of a JSON object.\n\nThis field is always returned, regardless of the event type.\n\nIf there are no traits, an empty JSON object is returned.",
	},
	{Name: "type", Type: types.Text().WithValues("alias", "identify", "group", "page", "screen", "track")},
	{
		Name:         "previousId",
		Type:         types.Text(),
		ReadOptional: true,
		Description:  "Previous user ID associated with the event. Defined only for alias-type events.\n\nThis field is unused by the Meergo model, and exists solely for compatibility with SDKs that send it.",
	},
	{
		Name:         "userId",
		Type:         types.Text(),
		ReadOptional: true,
		Description:  "ID of the user who generated the event, if available; otherwise, this field is not returned. Not to be confused with user, which refers to the internal Meergo user ID associated with the event by the Identity Resolution.",
	},
}

func init() {

	eventsGetParameter := types.Array(types.Object(append([]types.Property{
		{
			Name:        "id",
			Type:        types.UUID(),
			Prefilled:   `"b1d868f3-43f6-4965-bbd2-85ca8dd609b3"`,
			Description: "ID that uniquely identifies the event. Automatically generated by Meergo when the event is received.\n\nUnlike `messageId`, which uniquely identifies the event within the context of a specific connection, this identifies the event across all connections.",
		},
		{
			Name:         "user",
			Type:         types.UUID(),
			ReadOptional: true,
			Prefilled:    `"9102d2a1-0714-4c13-bafd-8a38bc3d0cff"`,
			Description:  "The Meergo user associated with the event, if any; otherwise, this field is absent.\n\n`user` is set for each event by the Identity Resolution process, and its value may change over time depending on how users are unified and associated with events.",
		},
		{
			Name:        "connection",
			Type:        types.Int(32),
			Prefilled:   "1371036433",
			Description: "The connection from which the event originates. Automatically set by Meergo when the event is received.",
		},
	}, eventGetProperties...)))
	observedEventsParameter := types.Array(types.Object(
		append([]types.Property{
			{
				Name:         "user",
				Type:         types.UUID(),
				ReadOptional: true,
				Description:  "User associated with the event.\n\nPlease note that, currently, this value is set by the Identity Resolution on the events on the data warehouse, so this field is never returned for observed events.",
			},
		}, eventGetProperties...),
	))
	idParameter := types.Property{
		Name:           "id",
		Type:           types.Int(32),
		Prefilled:      "902647263",
		CreateRequired: true,
		Description:    "The ID of the event listener.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "events",
		Name: "Events",
		Description: "Events are customer behavioral events (such as page views, clicks, or purchases) received from websites, mobile apps, and servers. " +
			"They can be stored in the workspace's data warehouse and forwarded to apps. You can also import customer data for identity resolution and unification.\n\n" +
			"These endpoints allow you to ingest events, retrieve events from the data warehouse, get the event schema, and manage event listeners.\n\n" +
			"You can also use one of the available [SDKs to send events](/developers/), instead of interacting with these API endpoints directly.",
		Endpoints: []*Endpoint{
			{
				Name: "Ingest events",
				Description: "This endpoint allows you to ingest a batch of events.\n" +
					"### Authentication\n" +
					"It supports authentication with both an [API key](authentication) and an [event write key](authentication#event-write-keys):\n" +
					"* For a website or mobile app, you must exclusively use an event write key, as it only provides access to event ingestion endpoints.\n" +
					"* For a server application, using an event write key is recommended if you don't need access to other endpoints.\n" +
					"### Alternative payloads\n" +
					"Alternatively, when using an event write key, you can also pass the array of events directly in the request body.\n\n" +
					"If there is a single event to ingest, you can pass the event object directly, similar to the [Ingest event](events#ingest-event) endpoint, " +
					"but specifying the event type in the request body.",
				Method: POST,
				URL:    "/v1/events",
				Parameters: []types.Property{
					{
						Name:           "connection",
						Type:           types.Int(32),
						Prefilled:      "1371036433",
						UpdateRequired: true,
						Description: "The ID of the connection to which the events refer. It can only be a source SDK connection.\n\n" +
							"It is required only if the call is authenticated using an API key. " +
							"If authentication is done with an event write key, it is not needed, as the connection is that of the key.",
					},
					{
						Name: "batch",
						Type: types.Array(types.Object([]types.Property{
							{
								Name:           "anonymousId",
								Type:           types.Text(),
								Prefilled:      `"3e93e10e-5ca0-4a8c-bef6-cf9197b37729"`,
								UpdateRequired: true,
								Nullable:       true,
								Description:    "Either `anonymousId` or `userId` must be provided, and neither can be null or empty.",
							},
							{
								Name:        "channel",
								Type:        types.Text(),
								Description: "The channel from which the event originates.",
							},
							{
								Name:        "category",
								Type:        types.Text(),
								Nullable:    true,
								Description: "It is allowed only for page events.",
							},
							{
								Name:        "context",
								Type:        eventPostContextType,
								Description: "Additional contextual information for the event.",
							},
							{
								Name:           "event",
								Type:           types.Text(),
								UpdateRequired: true,
								Description:    "It is required only for track events. For all other types of events, it is not permitted.",
							},
							{
								Name:           "groupId",
								Type:           types.Text(),
								UpdateRequired: true,
								Nullable:       true,
								Description:    "It is required only for group events. For all other types of events, it is not permitted.",
							},
							{
								Name:        "messageId",
								Type:        types.Text(),
								Nullable:    true,
								Description: "If it is missing or null, a generated UUID will be used as its value.",
							},
							{
								Name:        "name",
								Type:        types.Text(),
								Description: "It is allowed only for page and screen events.",
							},
							{
								Name:        "properties",
								Type:        types.JSON(),
								Description: "It is allowed only for page, screen, and track events.",
							},
							{
								Name: "receivedAt",
								Type: types.DateTime(),
							},
							{
								Name: "sentAt",
								Type: types.DateTime(),
							},
							{
								Name:        "originalTimestamp",
								Type:        types.DateTime(),
								Nullable:    true,
								Description: "It must follow the ISO 8601 timestamp format. When not provided, it is automatically set by Meergo.",
							},
							{
								Name:           "timestamp",
								Type:           types.DateTime(),
								UpdateRequired: true,
								Nullable:       true,
								Description:    "It must follow the ISO 8601 timestamp format. It is required and cannot be null if `originalTimestamp` is present and not null.",
							},
							{
								Name:        "traits",
								Type:        types.JSON(),
								Description: "It is allowed only for identify and group events. You can use `context.traits` for other types of events.",
							},
							{
								Name:           "type",
								Type:           types.Text().WithValues("alias", "identify", "group", "page", "screen", "track"),
								CreateRequired: true,
							},
							{
								Name:           "previousId",
								Type:           types.Text(),
								UpdateRequired: true,
								Description:    "It is required only for alias events. For all other types of events, it is not permitted.",
							},
							{
								Name:           "userId",
								Type:           types.Text(),
								UpdateRequired: true,
								Nullable:       true,
								Description:    "It is required and cannot be null or empty for identify and alias events. For other event types, either `anonymousId` or `userId` must be provided, and neither can be null or empty.",
							},
						})),
						CreateRequired: true,
						Description:    "The events to ingest.",
					},
					{
						Name:        "context",
						Type:        eventPostContextType,
						Description: "The global context. If present, the context for each event is merged with this global context.",
					},
					{
						Name:        "sentAt",
						Type:        types.DateTime(),
						Description: "The date on which the request was sent. The year must be in the range 1 to 9999. The sentAt value of each event, if present, overwrites this value.",
					},
					{
						Name:        "writeKey",
						Type:        types.Text(),
						Description: "The event write key of the connection.",
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
				},
			},
			{
				Name: "Ingest event",
				Description: "Ingests a single event.\n\n This endpoint supports authentication only with an [event write key](authentication#event-write-keys). " +
					"To ingest events with an API key, use the [Ingest events](events#ingest-events) endpoint, which supports both authentication methods.",
				Method:       POST,
				WriteKeyAuth: true,
				URL:          "/v1/events/:type",
				Parameters: []types.Property{
					{
						Name:           "type",
						Type:           types.Text().WithValues("alias", "identify", "group", "page", "screen", "track"),
						CreateRequired: true,
						Description:    "The type of the event.",
					},
					{
						Name:           "anonymousId",
						Type:           types.Text(),
						Prefilled:      `"3e93e10e-5ca0-4a8c-bef6-cf9197b37729"`,
						UpdateRequired: true,
						Nullable:       true,
						Description:    "Either `anonymousId` or `userId` must be provided. They cannot both be null or empty.",
					},
					{
						Name:        "channel",
						Type:        types.Text(),
						Description: "The channel from which the event originates.",
					},
					{
						Name:        "category",
						Type:        types.Text(),
						Nullable:    true,
						Description: "It is allowed only for page events.",
					},
					{
						Name:        "context",
						Type:        eventPostContextType,
						Description: "Additional contextual information for the event.",
					},
					{
						Name:           "event",
						Type:           types.Text(),
						UpdateRequired: true,
						Description:    "It is required only for track events. For all other types of events, it is not permitted.",
					},
					{
						Name:           "groupId",
						Type:           types.Text(),
						UpdateRequired: true,
						Nullable:       true,
						Description:    "It is required only for group events. For all other types of events, it is not permitted.",
					},
					{
						Name:        "messageId",
						Type:        types.Text(),
						Nullable:    true,
						Description: "If it is missing or null, a generated UUID will be used as its value.",
					},
					{
						Name:        "name",
						Type:        types.Text(),
						Description: "It is allowed only for page and screen events.",
					},
					{
						Name:        "properties",
						Type:        types.JSON(),
						Description: "It is allowed only for page, screen, and track events.",
					},
					{
						Name: "receivedAt",
						Type: types.DateTime(),
					},
					{
						Name: "sentAt",
						Type: types.DateTime(),
					},
					{
						Name:        "originalTimestamp",
						Type:        types.DateTime(),
						Nullable:    true,
						Description: "It must follow the ISO 8601 timestamp format. When not provided, it is automatically set by Meergo.",
					},
					{
						Name:           "timestamp",
						Type:           types.DateTime(),
						UpdateRequired: true,
						Nullable:       true,
						Description:    "It must follow the ISO 8601 timestamp format. It is required and cannot be null if `originalTimestamp` is present and not null.",
					},
					{
						Name:        "traits",
						Type:        types.JSON(),
						Description: "It is allowed only for identify and group events. You can use `context.traits` for other types of events.",
					},
					{
						Name:           "previousId",
						Type:           types.Text(),
						UpdateRequired: true,
						Description:    "It is required only for alias events. For all other types of events, it is not permitted.",
					},
					{
						Name:           "userId",
						Type:           types.Text(),
						UpdateRequired: true,
						Nullable:       true,
						Description:    "It is required and cannot be null or empty for identify and alias events. For other event types, either `anonymousId` or `userId` must be provided, and they cannot both be null or empty.",
					},
				},
			},
			{
				Name: "Retrieve all events",
				Description: "Retrieves events stored in the workspace's data warehouse, up to a maximum number of events defined by `limit`. You must specify which properties to include. " +
					"If a filter is provided, only events that match the filter criteria will be returned.",
				Method: GET,
				URL:    "/v1/events",
				Parameters: []types.Property{
					{
						Name:           "properties",
						Type:           types.Array(types.Text()),
						CreateRequired: true,
						Prefilled:      `connection,anonymousId`,
						Description: "The event properties to return. " +
							"The properties can be specified in the query string in this way:\n" +
							"```\nproperties=user&properties=connection&properties=anonymousId\n```",
					},
					{
						Name: "filter",
						Type: filterType,
						Description: "The filter applied to the events. Only the events that match the filter will be returned.\n\n" +
							"It must be encoded in JSON, then escaped for the context of the query string. So, for example, the JSON-encoded filter:\n\n" +
							"`" + `{"logical":"and","conditions":[{"property":"user","operator":"is","values":["960ae86c-fc6e-438a-ae03-838fa6c94946"]}]}` + "`\n\n" +
							"must then be escaped and passed in the query string as:\n\n" +
							"`filter=%7B%22logical%22%3A%22and%22%2C%22conditions\n%22%3A%5B%7B%22property%22%3A%22user%22%2C%22\n" +
							"operator%22%3A%22is%22%2C%22values%22%3A%5B%22\n960ae86c-fc6e-438a-ae03-838fa6c94946%22%5D%7D%5D%7D`",
					},
					{
						Name:        "order",
						Type:        types.Text().WithValues("id", "user", "connection", "anonymousId", "channel", "category", "event", "groupId", "messageId", "name", "receivedAt", "sentAt", "originalTimestamp", "timestamp", "type", "userId"),
						Prefilled:   `id`,
						Description: "The name of the property by which to sort the events to be returned.\n\nIf not provided, the events are sorted by `id`.",
					},
					{
						Name:        "orderDesc",
						Type:        types.Boolean(),
						Prefilled:   `false`,
						Description: "Indicates if the returned events are sorted in descending order; if not `true`, they are sorted in ascending order.",
					},
					{
						Name:        "first",
						Type:        types.Int(32),
						Prefilled:   `0`,
						Description: "The number of events to skip before starting to return results. The default value is 0.",
					},
					{
						Name:           "limit",
						Type:           types.Int(32).WithIntRange(1, 1000),
						CreateRequired: true,
						Prefilled:      `1000`,
						Description:    "The maximum number of events to return. The value must be within the range [1, 1000].",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:      "events",
							Type:      eventsGetParameter,
							Prefilled: "...",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
				},
			},
			{
				Name:        "Get event schema",
				Description: "Return the event schema. The event schema is the same for all workspaces.",
				Method:      GET,
				URL:         "/v1/events/schema",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:      "schema",
							Prefilled: "{ ... }",
							Type:      types.Parameter("schema"),
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
			{
				Name:        "Create event listener",
				Description: "Creates an event listener for the workspace that listens to events and returns its identifier.",
				Method:      POST,
				URL:         "/v1/events/listeners",
				Parameters: []types.Property{
					{
						Name:        "size",
						Type:        types.Int(32),
						Prefilled:   `10`,
						Nullable:    true,
						Description: "The maximum number of observed events to return. It must be between 1 and 1000. If not specified or set to null, the default is 10.",
					},
					{
						Name:        "filter",
						Type:        filterType,
						Nullable:    true,
						Description: "The filter applied to the events. If not null, only events that match the filter will be included.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						idParameter,
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, TooManyListeners, fmt.Sprintf("there are already %d listeners", core.MaxEventListeners)},
				},
			},
			{
				Name:        "List observed events",
				Description: "Returns the events captured by the specified listener along with the count of omitted events.",
				Method:      GET,
				URL:         "/v1/events/listeners/:id",
				Parameters: []types.Property{
					idParameter,
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "events",
							Type:        observedEventsParameter,
							Prefilled:   "...",
							Description: "The observed events.",
						},
						{
							Name:        "omitted",
							Type:        types.Int(32),
							Prefilled:   "572",
							Description: "The number of omitted events.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "event listener does not exist"},
				},
			},
			{
				Name:        "Delete event listener",
				Description: "Deletes an event listener. It does nothing if the event listener does not exist.",
				Method:      DELETE,
				URL:         "/v1/events/listeners/:id",
				Parameters: []types.Property{
					idParameter,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
		},
	})

}
