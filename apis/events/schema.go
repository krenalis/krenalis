//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package events

import (
	"time"

	"chichi/apis/types"
)

// Schema is the schema of an event.
//
// Keep in sync with:
//
//   - the SQL in files "apis/warehouses/*/events.sql"
//   - the function "applyActionMapping" in "apis/events/processor.go".
//   - the fields of the type "Event" in "connector/apps.go" (and the functions
//     that read/write it, as "collectedEventToMap").
var Schema = types.Object([]types.Property{
	{Name: "source", Type: types.Int()},
	{Name: "event", Type: types.Text()},
	{Name: "message_id", Type: types.Text()},
	{Name: "anonymous_id", Type: types.UUID()},
	{Name: "user_id", Type: types.Text()},
	{Name: "date", Type: types.Date(time.DateOnly)},
	{Name: "timestamp", Type: types.DateTime(time.StampMilli)},
	{Name: "sent_at", Type: types.DateTime(time.StampMilli)},
	{Name: "received_at", Type: types.DateTime(time.StampMilli)},
	{Name: "ip", Type: types.Inet()},
	{
		Name: "network",
		Type: types.Object([]types.Property{
			{Name: "cellular", Type: types.Boolean()},
			{Name: "wifi", Type: types.Boolean()},
			{Name: "bluetooth", Type: types.Boolean()},
			{Name: "carrier", Type: types.Text()},
		}),
		Flat: true,
	},
	{
		Name: "os",
		Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text().WithEnum([]string{"Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other"})},
			{Name: "version", Type: types.Text()},
		}),
		Flat: true,
	},
	{
		Name: "app",
		Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text()},
			{Name: "version", Type: types.Text()},
			{Name: "build", Type: types.Text()},
			{Name: "namespace", Type: types.Text()},
		}),
		Flat: true,
	},
	{
		Name: "screen",
		Type: types.Object([]types.Property{
			{Name: "density", Type: types.UInt16()},
			{Name: "width", Type: types.UInt16()},
			{Name: "height", Type: types.UInt16()},
		}),
		Flat: true,
	},
	{Name: "user_agent", Type: types.Text()},
	{
		Name: "browser",
		Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text().WithEnum([]string{"Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other"})},
			{Name: "other", Type: types.Text()},
			{Name: "version", Type: types.Text()},
		}),
		Flat: true,
	},
	{
		Name: "device",
		Type: types.Object([]types.Property{
			{Name: "id", Type: types.Text()},
			{Name: "name", Type: types.Text()},
			{Name: "manufacturer", Type: types.Text()},
			{Name: "model", Type: types.Text()},
			{Name: "type", Type: types.Text()},
			{Name: "version", Type: types.Text()},
			{Name: "advertising_id", Type: types.Text()},
		}),
		Flat: true,
	},
	{
		Name: "location",
		Type: types.Object([]types.Property{
			{Name: "city", Type: types.Text()},
			{Name: "country", Type: types.Text()},
			{Name: "region", Type: types.Text()},
			{Name: "latitude", Type: types.Float()},
			{Name: "longitude", Type: types.Float()},
			{Name: "speed", Type: types.Float()},
		}),
		Flat: true,
	},
	{Name: "device_type", Type: types.Text().WithEnum([]string{"desktop", "tablet", "mobile"})},
	{Name: "locale", Type: types.Text(types.Chars(5))},
	{Name: "timezone", Type: types.Text()},
	{
		Name: "page",
		Type: types.Object([]types.Property{
			{Name: "url", Type: types.Text()},
			{Name: "path", Type: types.Text()},
			{Name: "search", Type: types.Text()},
			{Name: "title", Type: types.Text()},
			{Name: "hash", Type: types.Text()},
			{Name: "referrer", Type: types.Text()},
		}),
		Flat: true,
	},
	{
		Name: "referrer",
		Type: types.Object([]types.Property{
			{Name: "type", Type: types.Text()},
			{Name: "name", Type: types.Text()},
			{Name: "url", Type: types.Text()},
			{Name: "link", Type: types.Text()},
		}),
		Flat: true,
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
		Flat: true,
	},
	{
		Name: "library",
		Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text()},
			{Name: "version", Type: types.Text()},
		}),
		Flat: true,
	},
	{Name: "properties", Type: types.JSON()},
})
