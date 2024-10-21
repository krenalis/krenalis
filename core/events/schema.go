//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package events

import "github.com/meergo/meergo/types"

// Schema is the schema of an enriched event.
var Schema = types.Object([]types.Property{
	{Name: "anonymousId", Type: types.Text()},
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
					{Name: "width", Type: types.Int(32)},
					{Name: "height", Type: types.Int(32)},
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
			{Name: "traits", Type: types.JSON(), ReadOptional: true},
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
	{Name: "source", Type: types.Int(32)},
	{Name: "timestamp", Type: types.DateTime()},
	{Name: "traits", Type: types.JSON(), ReadOptional: true},
	{Name: "type", Type: types.Text().WithValues("alias", "identify", "group", "page", "screen", "track")},
	{Name: "userId", Type: types.Text(), Nullable: true},
})
