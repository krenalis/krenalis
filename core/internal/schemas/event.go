// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package schemas

import (
	"github.com/krenalis/krenalis/tools/types"
)

// Event is the event schema.
var Event = types.Object([]types.Property{
	{Name: "kpid", Type: types.UUID(), ReadOptional: true, DisplayName: "Krenalis Profile ID"},
	{Name: "connectionId", Type: types.String(), DisplayName: "Connection ID"},
	{Name: "anonymousId", Type: types.String(), DisplayName: "Anonymous ID"},
	{Name: "channel", Type: types.String(), ReadOptional: true, DisplayName: "Channel"},
	{Name: "category", Type: types.String(), ReadOptional: true, DisplayName: "Category"},
	{
		Name: "context",
		Type: types.Object([]types.Property{
			{
				Name: "app",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.String(), ReadOptional: true, DisplayName: "Name"},
					{Name: "version", Type: types.String(), ReadOptional: true, DisplayName: "Version"},
					{Name: "build", Type: types.String(), ReadOptional: true, DisplayName: "Build"},
					{Name: "namespace", Type: types.String(), ReadOptional: true, DisplayName: "Namespace"},
				}),
				ReadOptional: true,
				DisplayName:  "App",
			},
			{
				Name: "browser",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.String().WithValues("Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other"), ReadOptional: true, DisplayName: "Name"},
					{Name: "other", Type: types.String(), ReadOptional: true, DisplayName: "Other"},
					{Name: "version", Type: types.String(), ReadOptional: true, DisplayName: "Version"},
				}),
				ReadOptional: true,
				DisplayName:  "Browser",
			},
			{
				Name: "campaign",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.String(), ReadOptional: true, DisplayName: "Name"},
					{Name: "source", Type: types.String(), ReadOptional: true, DisplayName: "Source"},
					{Name: "medium", Type: types.String(), ReadOptional: true, DisplayName: "Medium"},
					{Name: "term", Type: types.String(), ReadOptional: true, DisplayName: "Term"},
					{Name: "content", Type: types.String(), ReadOptional: true, DisplayName: "Content"},
				}),
				ReadOptional: true,
				DisplayName:  "Campaign",
			},
			{
				Name:         "consents",
				Type:         types.Map(types.Boolean()),
				ReadOptional: true,
				DisplayName:  "Consents",
			},
			{
				Name: "device",
				Type: types.Object([]types.Property{
					{Name: "id", Type: types.String(), ReadOptional: true, DisplayName: "Id"},
					{Name: "advertisingId", Type: types.String(), ReadOptional: true, DisplayName: "Advertising ID"},
					{Name: "adTrackingEnabled", Type: types.Boolean(), ReadOptional: true, DisplayName: "Ad tracking enabled"},
					{Name: "manufacturer", Type: types.String(), ReadOptional: true, DisplayName: "Manufacturer"},
					{Name: "model", Type: types.String(), ReadOptional: true, DisplayName: "Model"},
					{Name: "name", Type: types.String(), ReadOptional: true, DisplayName: "Name"},
					{Name: "type", Type: types.String(), ReadOptional: true, DisplayName: "Type"},
					{Name: "token", Type: types.String(), ReadOptional: true, DisplayName: "Token"},
				}),
				ReadOptional: true,
				DisplayName:  "Device",
			},
			{Name: "ip", Type: types.IP(), ReadOptional: true, DisplayName: "IP"},
			{
				Name: "library",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.String(), ReadOptional: true, DisplayName: "Name"},
					{Name: "version", Type: types.String(), ReadOptional: true, DisplayName: "Version"},
				}),
				ReadOptional: true,
				DisplayName:  "Library",
			},
			{Name: "locale", Type: types.String(), ReadOptional: true, DisplayName: "Locale"},
			{
				Name: "location",
				Type: types.Object([]types.Property{
					{Name: "city", Type: types.String(), ReadOptional: true, DisplayName: "City"},
					{Name: "country", Type: types.String(), ReadOptional: true, DisplayName: "Country"},
					{Name: "latitude", Type: types.Float(64), ReadOptional: true, DisplayName: "Latitude"},
					{Name: "longitude", Type: types.Float(64), ReadOptional: true, DisplayName: "Longitude"},
					{Name: "speed", Type: types.Float(64), ReadOptional: true, DisplayName: "Speed"},
				}),
				ReadOptional: true,
				DisplayName:  "Location",
			},
			{
				Name: "network",
				Type: types.Object([]types.Property{
					{Name: "bluetooth", Type: types.Boolean(), ReadOptional: true, DisplayName: "Bluetooth"},
					{Name: "carrier", Type: types.String(), ReadOptional: true, DisplayName: "Carrier"},
					{Name: "cellular", Type: types.Boolean(), ReadOptional: true, DisplayName: "Cellular"},
					{Name: "wifi", Type: types.Boolean(), ReadOptional: true, DisplayName: "Wi-Fi"},
				}),
				ReadOptional: true,
				DisplayName:  "Network",
			},
			{
				Name: "os",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.String().WithValues("Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other"), ReadOptional: true, DisplayName: "Name"},
					{Name: "other", Type: types.String(), ReadOptional: true, DisplayName: "Other"},
					{Name: "version", Type: types.String(), ReadOptional: true, DisplayName: "Version"},
				}),
				ReadOptional: true,
				DisplayName:  "OS",
			},
			{
				Name: "page",
				Type: types.Object([]types.Property{
					{Name: "path", Type: types.String(), ReadOptional: true, DisplayName: "Path"},
					{Name: "referrer", Type: types.String(), ReadOptional: true, DisplayName: "Referrer"},
					{Name: "search", Type: types.String(), ReadOptional: true, DisplayName: "Search"},
					{Name: "title", Type: types.String(), ReadOptional: true, DisplayName: "Title"},
					{Name: "url", Type: types.String(), ReadOptional: true, DisplayName: "URL"},
				}),
				ReadOptional: true,
				DisplayName:  "Page",
			},
			{
				Name: "referrer",
				Type: types.Object([]types.Property{
					{Name: "id", Type: types.String(), ReadOptional: true, DisplayName: "ID"},
					{Name: "type", Type: types.String(), ReadOptional: true, DisplayName: "Type"},
				}),
				ReadOptional: true,
				DisplayName:  "Referrer",
			},
			{
				Name: "screen",
				Type: types.Object([]types.Property{
					{Name: "width", Type: types.Int(16), ReadOptional: true, DisplayName: "Width"},
					{Name: "height", Type: types.Int(16), ReadOptional: true, DisplayName: "Height"},
					{Name: "density", Type: types.Decimal(3, 2), ReadOptional: true, DisplayName: "Density"},
				}),
				ReadOptional: true,
				DisplayName:  "Screen",
			},
			{
				Name: "session",
				Type: types.Object([]types.Property{
					{Name: "id", Type: types.Int(64), ReadOptional: true, DisplayName: "ID"},
					{Name: "start", Type: types.Boolean(), ReadOptional: true, DisplayName: "Start"},
				}),
				ReadOptional: true,
				DisplayName:  "Session",
			},
			{Name: "timezone", Type: types.String(), ReadOptional: true, DisplayName: "Timezone"},
			{Name: "userAgent", Type: types.String(), ReadOptional: true, DisplayName: "User agent"},
		}),
		ReadOptional: true,
		DisplayName:  "Context",
	},
	{Name: "event", Type: types.String(), ReadOptional: true, DisplayName: "Event"},
	{Name: "groupId", Type: types.String(), ReadOptional: true, DisplayName: "Group ID"},
	{Name: "messageId", Type: types.String(), DisplayName: "Message ID"},
	{Name: "name", Type: types.String(), ReadOptional: true, DisplayName: "Name"},
	{Name: "properties", Type: types.JSON(), ReadOptional: true, DisplayName: "Properties"},
	{Name: "receivedAt", Type: types.DateTime(), DisplayName: "Received at"},
	{Name: "sentAt", Type: types.DateTime(), DisplayName: "Sent at"},
	{Name: "originalTimestamp", Type: types.DateTime(), DisplayName: "Original timestamp"},
	{Name: "timestamp", Type: types.DateTime(), DisplayName: "Timestamp"},
	{Name: "traits", Type: types.JSON(), DisplayName: "Traits"},
	{Name: "type", Type: types.String().WithValues("alias", "identify", "group", "page", "screen", "track"), DisplayName: "Type"},
	{Name: "previousId", Type: types.String(), ReadOptional: true, DisplayName: "Previous ID"},
	{Name: "userId", Type: types.String(), ReadOptional: true, DisplayName: "User ID"},
})
