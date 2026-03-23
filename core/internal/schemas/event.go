// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package schemas

import (
	"github.com/krenalis/krenalis/tools/types"
)

// Event is the event schema.
var Event = types.Object([]types.Property{
	{Name: "mpid", Type: types.UUID(), ReadOptional: true, Description: "Meergo Profile ID"},
	{Name: "connectionId", Type: types.Int(32), Description: "Connection ID"},
	{Name: "anonymousId", Type: types.String(), Description: "Anonymous ID"},
	{Name: "channel", Type: types.String(), ReadOptional: true, Description: "Channel"},
	{Name: "category", Type: types.String(), ReadOptional: true, Description: "Category"},
	{
		Name: "context",
		Type: types.Object([]types.Property{
			{
				Name: "app",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.String(), ReadOptional: true, Description: "Name"},
					{Name: "version", Type: types.String(), ReadOptional: true, Description: "Version"},
					{Name: "build", Type: types.String(), ReadOptional: true, Description: "Build"},
					{Name: "namespace", Type: types.String(), ReadOptional: true, Description: "Namespace"},
				}),
				ReadOptional: true,
				Description:  "App",
			},
			{
				Name: "browser",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.String().WithValues("Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other"), ReadOptional: true, Description: "Name"},
					{Name: "other", Type: types.String(), ReadOptional: true, Description: "Other"},
					{Name: "version", Type: types.String(), ReadOptional: true, Description: "Version"},
				}),
				ReadOptional: true,
				Description:  "Browser",
			},
			{
				Name: "campaign",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.String(), ReadOptional: true, Description: "Name"},
					{Name: "source", Type: types.String(), ReadOptional: true, Description: "Source"},
					{Name: "medium", Type: types.String(), ReadOptional: true, Description: "Medium"},
					{Name: "term", Type: types.String(), ReadOptional: true, Description: "Term"},
					{Name: "content", Type: types.String(), ReadOptional: true, Description: "Content"},
				}),
				ReadOptional: true,
				Description:  "Campaign",
			},
			{
				Name: "device",
				Type: types.Object([]types.Property{
					{Name: "id", Type: types.String(), ReadOptional: true, Description: "Id"},
					{Name: "advertisingId", Type: types.String(), ReadOptional: true, Description: "Advertising ID"},
					{Name: "adTrackingEnabled", Type: types.Boolean(), ReadOptional: true, Description: "Ad tracking enabled"},
					{Name: "manufacturer", Type: types.String(), ReadOptional: true, Description: "Manufacturer"},
					{Name: "model", Type: types.String(), ReadOptional: true, Description: "Model"},
					{Name: "name", Type: types.String(), ReadOptional: true, Description: "Name"},
					{Name: "type", Type: types.String(), ReadOptional: true, Description: "Type"},
					{Name: "token", Type: types.String(), ReadOptional: true, Description: "Token"},
				}),
				ReadOptional: true,
				Description:  "Device",
			},
			{Name: "ip", Type: types.IP(), ReadOptional: true, Description: "IP"},
			{
				Name: "library",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.String(), ReadOptional: true, Description: "Name"},
					{Name: "version", Type: types.String(), ReadOptional: true, Description: "Version"},
				}),
				ReadOptional: true,
				Description:  "Library",
			},
			{Name: "locale", Type: types.String(), ReadOptional: true, Description: "Locale"},
			{
				Name: "location",
				Type: types.Object([]types.Property{
					{Name: "city", Type: types.String(), ReadOptional: true, Description: "City"},
					{Name: "country", Type: types.String(), ReadOptional: true, Description: "Country"},
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
					{Name: "carrier", Type: types.String(), ReadOptional: true, Description: "Carrier"},
					{Name: "cellular", Type: types.Boolean(), ReadOptional: true, Description: "Cellular"},
					{Name: "wifi", Type: types.Boolean(), ReadOptional: true, Description: "Wi-Fi"},
				}),
				ReadOptional: true,
				Description:  "Network",
			},
			{
				Name: "os",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.String().WithValues("Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other"), ReadOptional: true, Description: "Name"},
					{Name: "other", Type: types.String(), ReadOptional: true, Description: "Other"},
					{Name: "version", Type: types.String(), ReadOptional: true, Description: "Version"},
				}),
				ReadOptional: true,
				Description:  "OS",
			},
			{
				Name: "page",
				Type: types.Object([]types.Property{
					{Name: "path", Type: types.String(), ReadOptional: true, Description: "Path"},
					{Name: "referrer", Type: types.String(), ReadOptional: true, Description: "Referrer"},
					{Name: "search", Type: types.String(), ReadOptional: true, Description: "Search"},
					{Name: "title", Type: types.String(), ReadOptional: true, Description: "Title"},
					{Name: "url", Type: types.String(), ReadOptional: true, Description: "URL"},
				}),
				ReadOptional: true,
				Description:  "Page",
			},
			{
				Name: "referrer",
				Type: types.Object([]types.Property{
					{Name: "id", Type: types.String(), ReadOptional: true, Description: "ID"},
					{Name: "type", Type: types.String(), ReadOptional: true, Description: "Type"},
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
					{Name: "id", Type: types.Int(64), ReadOptional: true, Description: "ID"},
					{Name: "start", Type: types.Boolean(), ReadOptional: true, Description: "Start"},
				}),
				ReadOptional: true,
				Description:  "Session",
			},
			{Name: "timezone", Type: types.String(), ReadOptional: true, Description: "Timezone"},
			{Name: "userAgent", Type: types.String(), ReadOptional: true, Description: "User agent"},
		}),
		ReadOptional: true,
		Description:  "Context",
	},
	{Name: "event", Type: types.String(), ReadOptional: true, Description: "Event"},
	{Name: "groupId", Type: types.String(), ReadOptional: true, Description: "Group ID"},
	{Name: "messageId", Type: types.String(), Description: "Message ID"},
	{Name: "name", Type: types.String(), ReadOptional: true, Description: "Name"},
	{Name: "properties", Type: types.JSON(), ReadOptional: true, Description: "Properties"},
	{Name: "receivedAt", Type: types.DateTime(), Description: "Received at"},
	{Name: "sentAt", Type: types.DateTime(), Description: "Sent at"},
	{Name: "originalTimestamp", Type: types.DateTime(), Description: "Original timestamp"},
	{Name: "timestamp", Type: types.DateTime(), Description: "Timestamp"},
	{Name: "traits", Type: types.JSON(), Description: "Traits"},
	{Name: "type", Type: types.String().WithValues("alias", "identify", "group", "page", "screen", "track"), Description: "Type"},
	{Name: "previousId", Type: types.String(), ReadOptional: true, Description: "Previous ID"},
	{Name: "userId", Type: types.String(), ReadOptional: true, Description: "User ID"},
})
