//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package schemas

import (
	"github.com/meergo/meergo/types"
)

// Event is the event schema.
var Event = types.Object([]types.Property{
	{Name: "id", Type: types.UUID()},
	// The "user" field may be set only by the Identity Resolution for events
	// stored in the data warehouse.
	// For all other events, it never has a value.
	// For consistency, it is included in all event schemas to avoid having to
	// differentiate between schemas.
	{Name: "user", Type: types.UUID(), ReadOptional: true},
	{Name: "connection", Type: types.Int(32)},
	{Name: "anonymousId", Type: types.Text()},
	{Name: "channel", Type: types.Text(), ReadOptional: true},
	{Name: "category", Type: types.Text(), ReadOptional: true},
	{
		Name: "context",
		Type: types.Object([]types.Property{
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
					{Name: "name", Type: types.Text().WithValues("Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other"), ReadOptional: true},
					{Name: "other", Type: types.Text(), ReadOptional: true},
					{Name: "version", Type: types.Text(), ReadOptional: true},
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
			},
			{
				Name: "os",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.Text().WithValues("Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other"), ReadOptional: true},
					{Name: "other", Type: types.Text(), ReadOptional: true},
					{Name: "version", Type: types.Text(), ReadOptional: true},
				}),
				ReadOptional: true,
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
					{Name: "width", Type: types.Int(16), ReadOptional: true},
					{Name: "height", Type: types.Int(16), ReadOptional: true},
					{Name: "density", Type: types.Decimal(3, 2), ReadOptional: true},
				}),
				ReadOptional: true,
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
		}),
		ReadOptional: true,
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
	{Name: "traits", Type: types.JSON()},
	{Name: "type", Type: types.Text().WithValues("alias", "identify", "group", "page", "screen", "track")},
	{Name: "previousId", Type: types.Text(), ReadOptional: true},
	{Name: "userId", Type: types.Text(), ReadOptional: true},
})
