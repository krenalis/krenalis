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
	"github.com/meergo/meergo/core/types"
)

// eventPostContextType is a types.Type representing the context of an event
// passed to the API methods that POST events, i.e. ingestion.
var eventPostContextType = types.Object([]types.Property{
	{
		Name: "app",
		Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text(), Description: "The application name."},
			{Name: "version", Type: types.Text(), Description: "The application version."},
			{Name: "build", Type: types.Text(), Description: "The application build identifier."},
			{Name: "namespace", Type: types.Text(), Description: "The application namespace or internal identifier."},
		}),
		Description: "The application that sent the event.",
	},
	{
		Name: "browser",
		Type: types.Object([]types.Property{
			{
				Name: "name",
				Type: types.Text(),
				Description: "The name of the browser from which the event originated.\n\n" +
					"Meergo tries to normalize this field and store it with one of those names: `\"Chrome\"`, `\"Safari\"`, `\"Edge\"`, `\"Firefox\"`, `\"Samsung Internet\"` or `\"Opera\"`.\n\n" +
					"Otherwise, if the passed browser name cannot be normalized, it is set to `\"Other\"` and the passed name is stored — as is — into `context.browser.other`.",
			},
			{
				Name:        "version",
				Type:        types.Text(),
				Description: "The version of the browser from which the event originated.",
			},
		}),
		Description: "The browser from which the event originates.\n\n" +
			"If not explicitly provided, Meergo will attempt to infer it from the `context.userAgent` property, if available.",
	},
	{
		Name: "campaign",
		Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text(), Description: "The campaign name."},
			{Name: "source", Type: types.Text(), Description: "The campaign source."},
			{Name: "medium", Type: types.Text(), Description: "The campaign medium (e.g. email, social)."},
			{Name: "term", Type: types.Text(), Description: "The campaign keyword or term."},
			{Name: "content", Type: types.Text(), Description: "The campaign content or variation."},
		}),
		Description: "The campaign that originated the event.",
	},
	{
		Name: "device",
		Type: types.Object([]types.Property{
			{Name: "id", Type: types.Text(), Description: "The device identifier."},
			{Name: "advertisingId", Type: types.Text(), Description: "The advertising identifier."},
			{Name: "adTrackingEnabled", Type: types.Boolean(), Description: "Indicates whether ad tracking is enabled."},
			{Name: "manufacturer", Type: types.Text(), Description: "The device manufacturer."},
			{Name: "model", Type: types.Text(), Description: "The device model."},
			{Name: "name", Type: types.Text(), Description: "The device name."},
			{Name: "type", Type: types.Text(), Description: "The device type (e.g., mobile, desktop)."},
			{Name: "token", Type: types.Text(), Description: "The unique device token."},
		}),
		Description: "The device from which the event originated.",
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
			{Name: "name", Type: types.Text(), Description: "The name of the analytics library."},
			{Name: "version", Type: types.Text(), Description: "The version of the analytics library."},
		}),
		Description: "The analytics library used to send the event.",
	},
	{Name: "locale", Type: types.Text(), Description: "The user's language and regional settings, such as `\"en-US\"`."},
	{
		Name: "location",
		Type: types.Object([]types.Property{
			{Name: "city", Type: types.Text(), Description: "The city."},
			{Name: "country", Type: types.Text(), Description: "The country."},
			{Name: "latitude", Type: types.Float(64), Description: "The latitude."},
			{Name: "longitude", Type: types.Float(64), Description: "The longitude."},
			{Name: "speed", Type: types.Float(64), Description: "The speed at which the device is moving, in meters per second."},
		}),
		Description: "The device location from which the event originated. If not explicitly provided, Meergo attempts to determine it automatically based on the event's IP.",
	},
	{
		Name: "network",
		Type: types.Object([]types.Property{
			{Name: "bluetooth", Type: types.Boolean(), Description: "Indicates whether Bluetooth is active."},
			{Name: "carrier", Type: types.Text(), Description: "The mobile network carrier name."},
			{Name: "cellular", Type: types.Boolean(), Description: "Indicates whether a cellular connection is active."},
			{Name: "wifi", Type: types.Boolean(), Description: "Indicates whether Wi-Fi is active."},
		}),
		Description: "The network to which the device originating the event is connected.",
	},
	{
		Name: "os",
		Type: types.Object([]types.Property{
			{
				Name: "name",
				Type: types.Text(),
				Description: "The name of the OS.\n\n" +
					"Meergo tries to normalize this field and store it with one of those names: `\"Android\"`, `\"Windows\"`, `\"macOS\"`, `\"iOS\"`, `\"Linux\"` or `\"Chrome OS\"`." +
					"Otherwise, if the passed OS name cannot be normalized, it is set to `\"Other\"` and the passed name is stored — as is — into `context.os.other`.",
			},
			{
				Name:        "version",
				Type:        types.Text(),
				Description: "The version of the OS.",
			},
		}),
		Description: "The OS of the device or browser from which the event originated.\n\n" +
			"If not explicitly provided, Meergo will attempt to infer it from the `context.userAgent` property, if available.",
	},
	{
		Name: "page",
		Type: types.Object([]types.Property{
			{Name: "path", Type: types.Text(), Description: "The page path."},
			{Name: "referrer", Type: types.Text(), Description: "The referring URL."},
			{Name: "search", Type: types.Text(), Description: "The URL search parameters."},
			{Name: "title", Type: types.Text(), Description: "The page title."},
			{Name: "url", Type: types.Text(), Description: "The full page URL."},
		}),
		Description: "The page where the event originated.",
	},
	{
		Name: "referrer",
		Type: types.Object([]types.Property{
			{Name: "id", Type: types.Text(), Description: "The identifier of the referring source (e.g., campaign or partner ID)."},
			{Name: "type", Type: types.Text(), Description: "The source category of the referrer (e.g., search, social, email)."},
		}),
		Description: "The URL of the page or source that referred the user to the current page.",
	},
	{
		Name: "screen",
		Type: types.Object([]types.Property{
			{
				Name:        "width",
				Type:        types.Int(16),
				Description: "The screen width. Must be in range [1, 32767].",
			},
			{
				Name:        "height",
				Type:        types.Int(16),
				Description: "The screen height. Must be in range [1, 32767].",
			},
			{
				Name:        "density",
				Type:        types.Decimal(3, 2),
				Description: "The screen density. Must be a positive number.",
			},
		}),
		Description: "The screen of the app where the event was originated.",
	},
	{
		Name: "session",
		Type: types.Object([]types.Property{
			{Name: "id", Type: types.Int(64), Description: "The session identifier."},
			{Name: "start", Type: types.Boolean(), Description: "Indicates whether the event starts a session."},
		}),
		Description: "The current user session",
	},
	{Name: "timezone", Type: types.Text(), Description: "The user's timezone as a tzdata string (e.g., America/New_York)."},
	{
		Name:        "userAgent",
		Type:        types.Text(),
		Description: "The device identifier from which the event originates.",
	},
})

// eventGetContextType is a types.Type representing the context of an event
// returned by the API methods that GET events.
var eventGetContextType = types.Object([]types.Property{
	{
		Name: "app",
		Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text(), ReadOptional: true, Description: "The application name."},
			{Name: "version", Type: types.Text(), ReadOptional: true, Description: "The application version."},
			{Name: "build", Type: types.Text(), ReadOptional: true, Description: "The application build identifier."},
			{Name: "namespace", Type: types.Text(), ReadOptional: true, Description: "The application namespace or internal identifier."},
		}),
		ReadOptional: true,
		Description:  "The application that sent the event.",
	},
	{
		Name: "browser",
		Type: types.Object([]types.Property{
			{
				Name:         "name",
				Type:         types.Text().WithValues("Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other"),
				ReadOptional: true,
				Description: "The name of the browser from which the event originated.\n\n" +
					"If the value is `\"Other\"`, then the field `other` is populated with the browser name.",
			},
			{
				Name:         "other",
				Type:         types.Text(),
				ReadOptional: true,
				Description: "The name of the browser in case it is not one of those recognized by Meergo.\n\n" +
					"This field is present only when `name` is `\"Other\"`.",
			},
			{
				Name:         "version",
				Type:         types.Text(),
				ReadOptional: true,
				Description:  "The version of the browser from which the event originated.",
			},
		}),
		ReadOptional: true,
		Description:  "The browser from which the event originated.",
	},
	{
		Name: "campaign",
		Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text(), ReadOptional: true, Description: "The campaign name."},
			{Name: "source", Type: types.Text(), ReadOptional: true, Description: "The campaign source."},
			{Name: "medium", Type: types.Text(), ReadOptional: true, Description: "The campaign medium (e.g. \"email\", \"social\")."},
			{Name: "term", Type: types.Text(), ReadOptional: true, Description: "The campaign keyword or term."},
			{Name: "content", Type: types.Text(), ReadOptional: true, Description: "The campaign content or variation."},
		}),
		ReadOptional: true,
		Description:  "The campaign that originated the event.",
	},
	{
		Name: "device",
		Type: types.Object([]types.Property{
			{Name: "id", Type: types.Text(), ReadOptional: true, Description: "The device identifier."},
			{Name: "advertisingId", Type: types.Text(), ReadOptional: true, Description: "The advertising identifier."},
			{Name: "adTrackingEnabled", Type: types.Boolean(), ReadOptional: true, Description: "Indicates whether ad tracking is enabled."},
			{Name: "manufacturer", Type: types.Text(), ReadOptional: true, Description: "The device manufacturer."},
			{Name: "model", Type: types.Text(), ReadOptional: true, Description: "The device model."},
			{Name: "name", Type: types.Text(), ReadOptional: true, Description: "The device name."},
			{Name: "type", Type: types.Text(), ReadOptional: true, Description: "The device type (e.g., mobile, desktop)."},
			{Name: "token", Type: types.Text(), ReadOptional: true, Description: "The unique device token."},
		}),
		ReadOptional: true,
		Description:  "The device from which the event originated.\n\nFor iOS, note that model identifiers may differ from marketed names (e.g., `\"iPhone16,2\"` for iPhone 15 Pro Max).",
	},
	{Name: "ip", Type: types.Inet(), ReadOptional: true, Description: "The IP address associated with the event."},
	{
		Name: "library",
		Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text(), ReadOptional: true, Description: "The name of the analytics library."},
			{Name: "version", Type: types.Text(), ReadOptional: true, Description: "The version of the analytics library."},
		}),
		ReadOptional: true,
		Description:  "The analytics library used to send the event.",
	},
	{Name: "locale", Type: types.Text(), ReadOptional: true},
	{
		Name: "location",
		Type: types.Object([]types.Property{
			{Name: "city", Type: types.Text(), ReadOptional: true, Description: "The city."},
			{Name: "country", Type: types.Text(), ReadOptional: true, Description: "The country."},
			{Name: "latitude", Type: types.Float(64), ReadOptional: true, Description: "The latitude."},
			{Name: "longitude", Type: types.Float(64), ReadOptional: true, Description: "The longitude."},
			{Name: "speed", Type: types.Float(64), ReadOptional: true, Description: "The speed at which the device is moving, in meters per second."},
		}),
		ReadOptional: true,
		Description:  "Device location from which the event was originated.",
	},
	{
		Name: "network",
		Type: types.Object([]types.Property{
			{Name: "bluetooth", Type: types.Boolean(), ReadOptional: true, Description: "Indicates whether Bluetooth is active."},
			{Name: "carrier", Type: types.Text(), ReadOptional: true, Description: "The mobile network carrier name."},
			{Name: "cellular", Type: types.Boolean(), ReadOptional: true, Description: "Indicates whether a cellular connection is active."},
			{Name: "wifi", Type: types.Boolean(), ReadOptional: true, Description: "Indicates whether Wi-Fi is active."},
		}),
		ReadOptional: true,
		Description:  "The network to which the device originating the event was connected.",
	},
	{
		Name: "os",
		Type: types.Object([]types.Property{
			{
				Name:         "name",
				Type:         types.Text().WithValues("Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other"),
				ReadOptional: true,
				Description: "The name of the OS from which the event was originated.\n\n" +
					"If the value is `\"Other\"`, then the field `other` is populated with the OS name.",
			},
			{
				Name:         "other",
				Type:         types.Text(),
				ReadOptional: true,
				Description:  "The name of the OS in case it is not one of those recognized by Meergo.\n\nThis field is present only when `name` is `\"Other\"`.",
			},
			{
				Name:         "version",
				Type:         types.Text(),
				ReadOptional: true,
				Description:  "The version of the OS.",
			},
		}),
		ReadOptional: true,
		Description:  "The OS of the device or browser from which the event was originated.",
	},
	{
		Name: "page",
		Type: types.Object([]types.Property{
			{Name: "path", Type: types.Text(), ReadOptional: true, Description: "The page path."},
			{Name: "referrer", Type: types.Text(), ReadOptional: true, Description: "The referring URL."},
			{Name: "search", Type: types.Text(), ReadOptional: true, Description: "The URL search parameters."},
			{Name: "title", Type: types.Text(), ReadOptional: true, Description: "The page title."},
			{Name: "url", Type: types.Text(), ReadOptional: true, Description: "The full page URL."},
		}),
		ReadOptional: true,
		Description:  "The page where the event originated.",
	},
	{
		Name: "referrer",
		Type: types.Object([]types.Property{
			{Name: "id", Type: types.Text(), ReadOptional: true, Description: "The identifier of the referring source (e.g., campaign or partner ID)."},
			{Name: "type", Type: types.Text(), ReadOptional: true, Description: "The source category of the referrer (e.g., search, social, email)."},
		}),
		ReadOptional: true,
		Description:  "The URL of the page or source that referred the user to the page where the event originated.",
	},
	{
		Name: "screen",
		Type: types.Object([]types.Property{
			{
				Name:         "width",
				Type:         types.Int(16),
				ReadOptional: true,
				Description:  "The screen width.",
			},
			{
				Name:         "height",
				Type:         types.Int(16),
				ReadOptional: true,
				Description:  "The screen height.",
			},
			{
				Name:         "density",
				Type:         types.Decimal(3, 2),
				ReadOptional: true,
				Description:  "The screen density. It is a positive number.",
			},
		}),
		ReadOptional: true,
		Description:  "The screen of the app where the event was originated.",
	},
	{
		Name: "session",
		Type: types.Object([]types.Property{
			{Name: "id", Type: types.Int(64), ReadOptional: true, Description: "The session identifier."},
			{Name: "start", Type: types.Boolean(), ReadOptional: true, Description: "Indicates whether the event started a session."},
		}),
		ReadOptional: true,
		Description:  "The user session when the event was generated.",
	},
	{Name: "timezone", Type: types.Text(), ReadOptional: true, Description: "The user's timezone as a tzdata string (e.g., `\"America/New_York\"`)."},
	{Name: "userAgent", Type: types.Text(), ReadOptional: true, Description: "The device identifier from which the event originated."},
})

// eventGetProperties is a types.Type representing the properties of an event
// returned by the API methods that GET events.
var eventGetProperties = []types.Property{
	{
		Name:        "anonymousId",
		Type:        types.Text(),
		Prefilled:   `"3e93e10e-5ca0-4a8c-bef6-cf9197b37729"`,
		Description: "A unique identifier assigned to a user before authentication, used to track anonymous actions across sessions.",
	},
	{
		Name:         "channel",
		Type:         types.Text(),
		ReadOptional: true,
		Description:  "The source channel through which the event was received (e.g., web, mobile, server).",
	},
	{
		Name:         "category",
		Type:         types.Text(),
		ReadOptional: true,
		Description:  "It is used to group related pages or screens for analysis and reporting.",
	},
	{
		Name:         "context",
		Type:         eventGetContextType,
		ReadOptional: true,
		Description:  "Information about the environment where the event occurred. If there's no information in the context, this field is not returned.",
	},
	{
		Name:         "event",
		Type:         types.Text(),
		ReadOptional: true,
		Description:  "Identifies a track-type event, for example with a value like `Product Purchased`. For any other event type, it is never returned.",
	},
	{
		Name:         "groupId",
		Type:         types.Text(),
		ReadOptional: true,
		Description:  "The group ID related to the event. Returned only for group-type event, and absent for all other event types.",
	},
	{
		Name:        "messageId",
		Type:        types.Text(),
		Description: "The ID that uniquely identifies the event.",
	},
	{
		Name:         "name",
		Type:         types.Text(),
		ReadOptional: true,
		Description:  "The title of the viewed page or screen in page and screen events. Not returned for other event types.",
	},
	{
		Name:         "properties",
		Type:         types.JSON(),
		ReadOptional: true,
		Description:  "A key–value pairs describing contextual details about the event (e.g., product_id, revenue, rating). Distinct from `traits`, which describe user attributes.",
	},
	{
		Name:        "receivedAt",
		Type:        types.DateTime(),
		Description: "An UTC timestamp indicating when the event was received. Set by Meergo.",
	},
	{
		Name:        "sentAt",
		Type:        types.DateTime(),
		Description: "An UTC timestamp indicating when the event was sent by the client. Not reliable for analysis due to potential client clock drift; use `timestamp` for analytics.",
	},
	{
		Name:        "originalTimestamp",
		Type:        types.DateTime(),
		Description: "An UTC timestamp indicating when the event occurred on the client, in ISO 8601 format. Not recommended for analysis due to possible clock drift; use `timestamp` instead.",
	},
	{
		Name:        "timestamp",
		Type:        types.DateTime(),
		Description: "An UTC timestamp indicating when the event occurred, in ISO 8601 format. Suitable for analysis.",
	},
	{
		Name:        "traits",
		Type:        types.JSON(),
		Description: "A key–value pairs containing user information (e.g., name, email, plan).\n\nThis field is always returned, regardless of the event type.\n\nIf there are no traits, an empty JSON object is returned.",
	},
	{
		Name:        "type",
		Type:        types.Text().WithValues("alias", "identify", "group", "page", "screen", "track"),
		Description: "The event type: one of \"page\", \"screen\", \"track\", \"identify\", \"group\", or \"alias\".",
	},
	{
		Name:         "previousId",
		Type:         types.Text(),
		ReadOptional: true,
		Description:  "The user's previous identifier.",
	},
	{
		Name:         "userId",
		Type:         types.Text(),
		ReadOptional: true,
		Description:  "The unique identifier assigned to a user after authentication. If absent, the user is anonymous.",
	},
}

func init() {

	eventsGetParameter := types.Array(types.Object(append([]types.Property{
		{
			Name:         "user",
			Type:         types.UUID(),
			ReadOptional: true,
			Prefilled:    `"9102d2a1-0714-4c13-bafd-8a38bc3d0cff"`,
			Description:  "The Meergo user associated with the event, if any; otherwise, this field is absent.\n\n`user` is set for each event by the Identity Resolution process, and its value may change over time depending on how users are unified and associated with events.",
		},
		{
			Name:        "connectionId",
			Type:        types.Int(32),
			Prefilled:   "1371036433",
			Description: "The ID of the source connection from which the event originates. Automatically set by Meergo when the event is received.",
		},
	}, eventGetProperties...)))
	observedEventsParameter := types.Array(types.Object(
		append([]types.Property{
			{
				Name:         "user",
				Type:         types.UUID(),
				ReadOptional: true,
				Description:  "The user associated with the event.\n\nPlease note that, currently, this value is set by the Identity Resolution on the events on the data warehouse, so this field is never returned for observed events.",
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
			"You can also use one of the available [SDKs to send events](/integrations/sources), instead of interacting with these API endpoints directly.",
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
						Name:           "connectionId",
						Type:           types.Int(32),
						Prefilled:      "1371036433",
						UpdateRequired: true,
						Description: "The ID of the source connection to which the events refer. It can only be a source SDK connection.\n\n" +
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
								Description: "A unique identifier assigned to a user before authentication, used to track anonymous actions across sessions.\n\n" +
									"Either `anonymousId` or `userId` must be provided, and neither can be null or empty.",
							},
							{
								Name:        "channel",
								Type:        types.Text(),
								Description: "The source channel through which the event was received (e.g., web, mobile, server).",
							},
							{
								Name:           "category",
								Type:           types.Text(),
								UpdateRequired: true,
								Nullable:       true,
								Description:    "It is used to group related pages or screens for analysis and reporting. It is allowed only for page events.",
							},
							{
								Name:        "context",
								Type:        eventPostContextType,
								Description: "Information about the environment where the event occurred.",
							},
							{
								Name:           "event",
								Type:           types.Text(),
								UpdateRequired: true,
								Description: "The name of the user action in a track event. Examples: Product Viewed, Order Completed.\n\n" +
									"It is required only for track events. For all other types of events, it is not permitted.",
							},
							{
								Name:           "groupId",
								Type:           types.Text(),
								UpdateRequired: true,
								Nullable:       true,
								Description: "Identifier of the group (e.g., account, company, organization) associated with the user.\n\n" +
									"It is required only for group events. For all other types of events, it is not permitted.",
							},
							{
								Name:        "messageId",
								Type:        types.Text(),
								Nullable:    true,
								Description: "The ID that uniquely identifies the event. If it is missing or null, a generated identifier will be used as its value.",
							},
							{
								Name: "name",
								Type: types.Text(),
								Description: "The title of the viewed page or screen in page and screen events.\n\n" +
									"It is allowed only for page and screen events.",
							},
							{
								Name: "properties",
								Type: types.JSON(),
								Description: "A key–value pairs describing contextual details about the event (e.g., product_id, revenue, rating). Distinct from `traits`, which describe user attributes.\n\n" +
									"It is allowed only for page, screen, and track events.",
							},
							{
								Name:        "sentAt",
								Type:        types.DateTime(),
								Description: "An UTC timestamp indicating when the event was sent, in ISO 8601 format. The year must be in the range 1 to 9999.",
							},
							{
								Name:     "originalTimestamp",
								Type:     types.DateTime(),
								Nullable: true,
								Description: "An UTC timestamp indicating when the event occurred on the client, in ISO 8601 format. The year must be in the range 1 to 9999.\n\n" +
									"Pass this property if the event occurred in the past.",
							},
							{
								Name:           "timestamp",
								Type:           types.DateTime(),
								UpdateRequired: true,
								Nullable:       true,
								Description: "An UTC timestamp indicating when the event occurred, in ISO 8601 format. The year must be in the range 1 to 9999.\n\n" +
									"It will be adjusted by Meergo to account for clock drift.\n\nIt is required if the property `originalTimestamp` is present.",
							},
							{
								Name:           "traits",
								Type:           types.JSON(),
								UpdateRequired: true,
								Description: "A key–value pairs containing user information (e.g., name, email, plan).\n\n" +
									"It is allowed only for identify and group events. You can use `context.traits` for other types of events.",
							},
							{
								Name:           "type",
								Type:           types.Text().WithValues("alias", "identify", "group", "page", "screen", "track"),
								CreateRequired: true,
								Description:    "The event type.",
							},
							{
								Name:           "previousId",
								Type:           types.Text(),
								UpdateRequired: true,
								Description: "The user's previous identifier. Not used by Meergo.\n\n" +
									"It is required only for alias events. For all other types of events, it is not permitted.",
							},
							{
								Name:           "userId",
								Type:           types.Text(),
								UpdateRequired: true,
								Nullable:       true,
								Description: "The unique identifier assigned to a user after authentication. If absent, the user is anonymous.\n\n" +
									"It is required and cannot be null or empty for identify and alias events. For other event types, either `anonymousId` or `userId` must be provided, and neither can be null or empty.",
							},
						})),
						CreateRequired: true,
						Description:    "The events to ingest.",
					},
					{
						Name:        "context",
						Type:        eventPostContextType,
						Prefilled:   `{...}`,
						Description: "This object defines shared context data applied to every event in the batch. If a property appears in both the global context and an event's own context, the event-specific value takes precedence.",
					},
					{
						Name:        "sentAt",
						Type:        types.DateTime(),
						Description: "An UTC timestamp indicating when the event was sent, in ISO 8601 format. The year must be in the range 1 to 9999. The sentAt value of each event, if present, overwrites this value.",
					},
					{
						Name:        "writeKey",
						Type:        types.Text(),
						Description: "The event write key of the connection, which can be used for authentication as an alternative to the **Authorization** header.",
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
						Description: "A unique identifier assigned to a user before authentication, used to track anonymous actions across sessions.\n\n" +
							"Either `anonymousId` or `userId` must be provided, and neither can be null or empty.",
					},
					{
						Name:        "channel",
						Type:        types.Text(),
						Description: "The source channel through which the event was received (e.g., web, mobile, server).",
					},
					{
						Name:           "category",
						Type:           types.Text(),
						UpdateRequired: true,
						Nullable:       true,
						Description:    "It is used to group related pages or screens for analysis and reporting. It is allowed only for page events.",
					},
					{
						Name:        "context",
						Type:        eventPostContextType,
						Description: "Information about the environment where the event occurred.",
					},
					{
						Name:           "event",
						Type:           types.Text(),
						UpdateRequired: true,
						Description: "The name of the user action in a track event. Examples: Product Viewed, Order Completed.\n\n" +
							"It is required only for track events. For all other types of events, it is not permitted.",
					},
					{
						Name:           "groupId",
						Type:           types.Text(),
						UpdateRequired: true,
						Nullable:       true,
						Description: "Identifier of the group (e.g., account, company, organization) associated with the user.\n\n" +
							"It is required only for group events. For all other types of events, it is not permitted.",
					},
					{
						Name:        "messageId",
						Type:        types.Text(),
						Nullable:    true,
						Description: "The ID that uniquely identifies the event. If it is missing or null, a generated identifier will be used as its value.",
					},
					{
						Name: "name",
						Type: types.Text(),
						Description: "The title of the viewed page or screen in page and screen events.\n\n" +
							"It is allowed only for page and screen events.",
					},
					{
						Name: "properties",
						Type: types.JSON(),
						Description: "A key–value pairs describing contextual details about the event (e.g., product_id, revenue, rating). Distinct from `traits`, which describe user attributes.\n\n" +
							"It is allowed only for page, screen, and track events.",
					},
					{
						Name:        "sentAt",
						Type:        types.DateTime(),
						Description: "An UTC timestamp indicating when the event was sent. The year must be in the range 1 to 9999.",
					},
					{
						Name:     "originalTimestamp",
						Type:     types.DateTime(),
						Nullable: true,
						Description: "An UTC timestamp indicating when the event occurred on the client, in ISO 8601 format. The year must be in the range 1 to 9999.\n\n" +
							"Pass this property if the event occurred in the past.",
					},
					{
						Name:           "timestamp",
						Type:           types.DateTime(),
						UpdateRequired: true,
						Nullable:       true,
						Description: "An UTC timestamp indicating when the event occurred, in ISO 8601 format. The year must be in the range 1 to 9999.\n\n" +
							"It will be adjusted by Meergo to account for clock drift.\n\nIt is required if the property `originalTimestamp` is present.",
					},
					{
						Name: "traits",
						Type: types.JSON(),
						Description: "A key–value pairs containing user information (e.g., name, email, plan).\n\n" +
							"It is allowed only for identify and group events. You can use `context.traits` for other types of events.",
					},
					{
						Name:           "previousId",
						Type:           types.Text(),
						UpdateRequired: true,
						Description: "The user's previous identifier. Not used by Meergo.\n\n" +
							"It is required only for alias events. For all other types of events, it is not permitted.",
					},
					{
						Name:           "userId",
						Type:           types.Text(),
						UpdateRequired: true,
						Nullable:       true,
						Description: "The unique identifier assigned to a user after authentication. If absent, the user is anonymous.\n\n" +
							"It is required and cannot be null or empty for identify and alias events. For other event types, either `anonymousId` or `userId` must be provided, and neither can be null or empty.",
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
						Prefilled:      `connectionId,anonymousId`,
						Description: "The event properties to return. " +
							"The properties can be specified in the query string in this way:\n" +
							"```\nproperties=user&properties=connectionId&properties=anonymousId\n```",
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
						Type:        types.Text().WithValues("user", "connectionId", "anonymousId", "channel", "category", "event", "groupId", "messageId", "name", "receivedAt", "sentAt", "originalTimestamp", "timestamp", "type", "userId"),
						Prefilled:   `id`,
						Description: "The name of the property by which to sort the events to be returned.\n\nIf not provided, the events are sorted by `messageId`.",
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
						Name:        "connectionId",
						Type:        types.Int(32),
						Prefilled:   `1371036433`,
						Nullable:    true,
						Description: "The ID of the source connection that received the events, or of the destination connection where the events are sent. The connection must support events.",
					},
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
					{422, ConnectionNotExist, "connection does not exist"},
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
