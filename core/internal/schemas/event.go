//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package schemas

import (
	"github.com/meergo/meergo/core/types"
)

// Event is the event schema.
var Event = types.Object([]types.Property{
	// The "user" field may be set only by the Identity Resolution for events
	// stored in the data warehouse.
	// For all other events, it never has a value.
	// For consistency, it is included in all event schemas to avoid having to
	// differentiate between schemas.
	{Name: "user", Type: types.UUID(), ReadOptional: true, Description: "User"},
	{Name: "connectionId", Type: types.Int(32), Description: "Connection Id"},
	{Name: "anonymousId", Type: types.Text(), Description: "Anonymous Id"},
	{Name: "channel", Type: types.Text(), ReadOptional: true, Description: "Channel"},
	{Name: "category", Type: types.Text(), ReadOptional: true, Description: "Category"},
	{
		Name: "context",
		Type: types.Object([]types.Property{
			{
				Name: "app",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.Text(), ReadOptional: true, Description: "Name"},
					{Name: "version", Type: types.Text(), ReadOptional: true, Description: "Version"},
					{Name: "build", Type: types.Text(), ReadOptional: true, Description: "Build"},
					{Name: "namespace", Type: types.Text(), ReadOptional: true, Description: "Namespace"},
				}),
				ReadOptional: true,
				Description:  "App",
			},
			{
				Name: "browser",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.Text().WithValues("Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other"), ReadOptional: true, Description: "Name"},
					{Name: "other", Type: types.Text(), ReadOptional: true, Description: "Other"},
					{Name: "version", Type: types.Text(), ReadOptional: true, Description: "Version"},
				}),
				ReadOptional: true,
				Description:  "Browser",
			},
			{
				Name: "campaign",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.Text(), ReadOptional: true, Description: "Name"},
					{Name: "source", Type: types.Text(), ReadOptional: true, Description: "Source"},
					{Name: "medium", Type: types.Text(), ReadOptional: true, Description: "Medium"},
					{Name: "term", Type: types.Text(), ReadOptional: true, Description: "Term"},
					{Name: "content", Type: types.Text(), ReadOptional: true, Description: "Content"},
				}),
				ReadOptional: true,
				Description:  "Campaign",
			},
			{
				Name: "device",
				Type: types.Object([]types.Property{
					{Name: "id", Type: types.Text(), ReadOptional: true, Description: "Id"},
					{Name: "advertisingId", Type: types.Text(), ReadOptional: true, Description: "Advertising Id"},
					{Name: "adTrackingEnabled", Type: types.Boolean(), ReadOptional: true, Description: "Ad tracking enabled"},
					{Name: "manufacturer", Type: types.Text(), ReadOptional: true, Description: "Manufacturer"},
					{Name: "model", Type: types.Text(), ReadOptional: true, Description: "Model"},
					{Name: "name", Type: types.Text(), ReadOptional: true, Description: "Name"},
					{Name: "type", Type: types.Text(), ReadOptional: true, Description: "Type"},
					{Name: "token", Type: types.Text(), ReadOptional: true, Description: "Token"},
				}),
				ReadOptional: true,
				Description:  "Device",
			},
			{Name: "ip", Type: types.Inet(), ReadOptional: true, Description: "IP"},
			{
				Name: "library",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.Text(), ReadOptional: true, Description: "Name"},
					{Name: "version", Type: types.Text(), ReadOptional: true, Description: "Version"},
				}),
				ReadOptional: true,
				Description:  "Library",
			},
			{Name: "locale", Type: types.Text(), ReadOptional: true, Description: "Locale"},
			{
				Name: "location",
				Type: types.Object([]types.Property{
					{Name: "city", Type: types.Text(), ReadOptional: true, Description: "City"},
					{Name: "country", Type: types.Text(), ReadOptional: true, Description: "Country"},
					{Name: "latitude", Type: types.Float(64), ReadOptional: true, Description: "Latitude"},
					{Name: "longitude", Type: types.Float(64), ReadOptional: true, Description: "Longitude"},
					{Name: "speed", Type: types.Float(64), ReadOptional: true, Description: "Speed"},
				}),
				ReadOptional: true,
				Description:  "Location",
			},
			{
				Name: "network",
				Type: types.Object([]types.Property{
					{Name: "bluetooth", Type: types.Boolean(), ReadOptional: true, Description: "Bluetooth"},
					{Name: "carrier", Type: types.Text(), ReadOptional: true, Description: "Carrier"},
					{Name: "cellular", Type: types.Boolean(), ReadOptional: true, Description: "Cellular"},
					{Name: "wifi", Type: types.Boolean(), ReadOptional: true, Description: "Wi-Fi"},
				}),
				ReadOptional: true,
				Description:  "Network",
			},
			{
				Name: "os",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.Text().WithValues("Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other"), ReadOptional: true, Description: "Name"},
					{Name: "other", Type: types.Text(), ReadOptional: true, Description: "Other"},
					{Name: "version", Type: types.Text(), ReadOptional: true, Description: "Version"},
				}),
				ReadOptional: true,
				Description:  "OS",
			},
			{
				Name: "page",
				Type: types.Object([]types.Property{
					{Name: "path", Type: types.Text(), ReadOptional: true, Description: "Path"},
					{Name: "referrer", Type: types.Text(), ReadOptional: true, Description: "Referrer"},
					{Name: "search", Type: types.Text(), ReadOptional: true, Description: "Search"},
					{Name: "title", Type: types.Text(), ReadOptional: true, Description: "Title"},
					{Name: "url", Type: types.Text(), ReadOptional: true, Description: "URL"},
				}),
				ReadOptional: true,
				Description:  "Page",
			},
			{
				Name: "referrer",
				Type: types.Object([]types.Property{
					{Name: "id", Type: types.Text(), ReadOptional: true, Description: "Id"},
					{Name: "type", Type: types.Text(), ReadOptional: true, Description: "Type"},
				}),
				ReadOptional: true,
				Description:  "Referrer",
			},
			{
				Name: "screen",
				Type: types.Object([]types.Property{
					{Name: "width", Type: types.Int(16), ReadOptional: true, Description: "Width"},
					{Name: "height", Type: types.Int(16), ReadOptional: true, Description: "Height"},
					{Name: "density", Type: types.Decimal(3, 2), ReadOptional: true, Description: "Density"},
				}),
				ReadOptional: true,
				Description:  "Screen",
			},
			{
				Name: "session",
				Type: types.Object([]types.Property{
					{Name: "id", Type: types.Int(64), ReadOptional: true, Description: "Id"},
					{Name: "start", Type: types.Boolean(), ReadOptional: true, Description: "Start"},
				}),
				ReadOptional: true,
				Description:  "Session",
			},
			{Name: "timezone", Type: types.Text(), ReadOptional: true, Description: "Timezone"},
			{Name: "userAgent", Type: types.Text(), ReadOptional: true, Description: "User agent"},
		}),
		ReadOptional: true,
		Description:  "Context",
	},
	{Name: "event", Type: types.Text(), ReadOptional: true, Description: "Event"},
	{Name: "groupId", Type: types.Text(), ReadOptional: true, Description: "Group Id"},
	{Name: "messageId", Type: types.Text(), Description: "Message Id"},
	{Name: "name", Type: types.Text(), ReadOptional: true, Description: "Name"},
	{Name: "properties", Type: types.JSON(), ReadOptional: true, Description: "Properties"},
	{Name: "receivedAt", Type: types.DateTime(), Description: "Received at"},
	{Name: "sentAt", Type: types.DateTime(), Description: "Sent at"},
	{Name: "originalTimestamp", Type: types.DateTime(), Description: "Original timestamp"},
	{Name: "timestamp", Type: types.DateTime(), Description: "Timestamp"},
	{Name: "traits", Type: types.JSON(), Description: "Traits"},
	{Name: "type", Type: types.Text().WithValues("alias", "identify", "group", "page", "screen", "track"), Description: "Type"},
	{Name: "previousId", Type: types.Text(), ReadOptional: true, Description: "Previous Id"},
	{Name: "userId", Type: types.Text(), ReadOptional: true, Description: "User Id"},
})
