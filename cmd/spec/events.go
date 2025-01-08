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

func init() {

	eventProperties := []types.Property{
		{Name: "anonymousId", Type: types.Text(), Placeholder: `"3e93e10e-5ca0-4a8c-bef6-cf9197b37729"`},
		{Name: "category", Type: types.Text(), ReadOptional: true},
		{
			Name: "context",
			Type: types.Object([]types.Property{
				{
					Name: "app",
					Type: types.Object([]types.Property{
						{Name: "name", Type: types.Text()},
						{Name: "version", Type: types.Text()},
						{Name: "build", Type: types.Text()},
						{Name: "namespace", Type: types.Text()},
					}),
					ReadOptional: true,
				},
				{
					Name: "browser",
					Type: types.Object([]types.Property{
						{Name: "name", Type: types.Text().WithValues("None", "Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other")},
						{Name: "other", Type: types.Text()},
						{Name: "version", Type: types.Text()},
					}),
					ReadOptional: true,
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
					ReadOptional: true,
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
					ReadOptional: true,
				},
				{Name: "ip", Type: types.Inet(), ReadOptional: true},
				{
					Name: "library",
					Type: types.Object([]types.Property{
						{Name: "name", Type: types.Text()},
						{Name: "version", Type: types.Text()},
					}),
					ReadOptional: true,
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
					ReadOptional: true,
				},
				{
					Name: "network",
					Type: types.Object([]types.Property{
						{Name: "bluetooth", Type: types.Boolean()},
						{Name: "carrier", Type: types.Text()},
						{Name: "cellular", Type: types.Boolean()},
						{Name: "wifi", Type: types.Boolean()},
					}),
					ReadOptional: true,
				},
				{
					Name: "os",
					Type: types.Object([]types.Property{
						{Name: "name", Type: types.Text().WithValues("None", "Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other")},
						{Name: "version", Type: types.Text()},
					}),
					ReadOptional: true,
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
					ReadOptional: true,
				},
				{
					Name: "referrer",
					Type: types.Object([]types.Property{
						{Name: "id", Type: types.Text()},
						{Name: "type", Type: types.Text()},
					}),
					ReadOptional: true,
				},
				{
					Name: "screen",
					Type: types.Object([]types.Property{
						{Name: "width", Type: types.Int(16)},
						{Name: "height", Type: types.Int(16)},
						{Name: "density", Type: types.Decimal(3, 2)},
					}),
					ReadOptional: true,
				},
				{
					Name: "session",
					Type: types.Object([]types.Property{
						{Name: "id", Type: types.Int(64)},
						{Name: "start", Type: types.Boolean(), ReadOptional: true},
					}),
					ReadOptional: true,
				},
				{Name: "timezone", Type: types.Text(), ReadOptional: true},
				{Name: "userAgent", Type: types.Text(), ReadOptional: true},
			}),
		},
		{Name: "event", Type: types.Text(), ReadOptional: true},
		{Name: "groupId", Type: types.Text(), ReadOptional: true},
		{Name: "messageId", Type: types.Text()},
		{Name: "name", Type: types.Text(), ReadOptional: true},
		{Name: "properties", Type: types.JSON(), ReadOptional: true},
		{Name: "receivedAt", Type: types.DateTime()},
		{Name: "sentAt", Type: types.DateTime()},
		{Name: "originalTimestamp", Type: types.DateTime()},
		{Name: "timestamp", Type: types.DateTime()},
		{Name: "traits", Type: types.JSON(), ReadOptional: true},
		{Name: "type", Type: types.Text().WithValues("alias", "identify", "group", "page", "screen", "track")},
		{Name: "userId", Type: types.Text(), Nullable: true},
	}
	eventsParameter := types.Object(append([]types.Property{
		{Name: "id", Type: types.UUID(), Placeholder: `"b1d868f3-43f6-4965-bbd2-85ca8dd609b3"`},
		{Name: "user", Type: types.UUID(), ReadOptional: true, Placeholder: `"9102d2a1-0714-4c13-bafd-8a38bc3d0cff"`},
		{Name: "connection", Type: types.Int(32), Placeholder: "1371036433"},
	}, eventProperties...))
	observedEventsParameter := types.Object(eventProperties)
	idParameter := types.Property{
		Name:           "id",
		Type:           types.Int(32),
		Placeholder:    "902647263",
		CreateRequired: true,
		Description:    "The ID of the event listener.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "events",
		Name:        "Events",
		Description: "Events are the events associated with a [workspace](workspaces), imported from the various sources, that are stored inside the [warehouse](warehouse).",
		Endpoints: []*Endpoint{
			{
				Name: "List all events",
				Description: "This endpoint retrieves events stored in the workspace's data warehouse, up to a maximum number of events defined by `limit`. You must specify which properties to include. " +
					"If a filter is provided, only events that match the filter criteria will be returned.",
				Method: POST,
				URL:    "/v0/events",
				Parameters: []types.Property{
					{
						Name:           "properties",
						Type:           types.Array(types.Text()),
						CreateRequired: true,
						Placeholder:    `[ "user", "type", "event" ]`,
						Description:    "The event properties to return.",
					},
					{
						Name:        "filter",
						Type:        filterType,
						Nullable:    true,
						Description: "The filter applied to the events. If it's not null, only the events that match the filter will be returned.",
					},
					{
						Name:        "order",
						Type:        types.Text().WithValues("id", "user", "connection", "anonymousId", "category", "event", "groupId", "messageId", "name", "receivedAt", "sentAt", "originalTimestamp", "timestamp", "type", "userId"),
						Placeholder: `"..."`,
						Description: "The name of the property by which to sort the events to be returned.",
					},
					{
						Name:        "orderDesc",
						Type:        types.Boolean(),
						Placeholder: `false`,
						Description: "The descending sorting order in which to return the events: if true, the events are sorted in descending order; otherwise, they are sorted in ascending order.",
					},
					{
						Name:        "first",
						Type:        types.Int(32),
						Placeholder: `0`,
						Description: "The number of events to skip before starting to return results. The default value is 0.",
					},
					{
						Name:           "limit",
						Type:           types.Int(32).WithIntRange(1, 1000),
						CreateRequired: true,
						Placeholder:    `1000`,
						Description:    "The maximum number of events to return. The value must be within the range [1, 1000].",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "events",
							Type:        eventsParameter,
							Placeholder: "...",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{422, MaintenanceMode, "data warehouse is in maintenance mode"},
					{422, WarehouseError, "error occurred with the data warehouse"},
				},
			},
			{
				Name:        "Get the event schema",
				Description: "Return the event schema. The event schema is the same for all workspaces.",
				Method:      GET,
				URL:         "/v0/events/schema",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "schema",
							Placeholder: "{ ... }",
							Type:        types.Parameter("Schema"),
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
				},
			},
			{
				Name:        "Create an event listener",
				Description: "Creates an event listener to the workspace that listens to events and returns its identifier.",
				Method:      POST,
				URL:         "/v0/events/listeners",
				Parameters: []types.Property{
					{
						Name:        "size",
						Type:        types.Int(32),
						Placeholder: `10`,
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
				Name:        "Retrive observed events",
				Description: "Returns the events captured by the specified listener along with the count of discarded events.",
				Method:      GET,
				URL:         "/v0/events/listeners/:id",
				Parameters: []types.Property{
					idParameter,
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "events",
							Type:        observedEventsParameter,
							Placeholder: "902647263",
							Description: "The observed events.",
						},
						{
							Name:        "discarded",
							Type:        types.Int(32),
							Placeholder: "572",
							Description: "The number of discarded events.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "event listener does not exist"},
				},
			},
			{
				Name:        "Delete an event listener",
				Description: "Deletes an event listener. It does nothing if the event lister does not exist.",
				Method:      DELETE,
				URL:         "/v0/events/listeners/:id",
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
