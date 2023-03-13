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

// Keep in sync with the SQL in files "apis/warehouses/*/events.sql"

var _, Schema = types.ObjectOf([]types.Property{
	{Name: "source", Type: types.Int()},
	{Name: "anonymous_id", Type: types.UUID()},
	{Name: "user_id", Type: types.Text()},
	{Name: "date", Type: types.Date(time.DateOnly)},
	{Name: "timestamp", Type: types.DateTime(time.StampMilli)},
	{Name: "sent_at", Type: types.DateTime(time.StampMilli)},
	{Name: "received_at", Type: types.DateTime(time.StampMilli)},
	{Name: "ip", Type: types.Inet()},
	{Name: "os_name", Type: types.Text().WithEnum([]string{"Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other"})},
	{Name: "os_version", Type: types.Text()},
	{Name: "user_agent", Type: types.Text()},
	{Name: "screen_density", Type: types.UInt16()},
	{Name: "screen_width", Type: types.UInt16()},
	{Name: "screen_height", Type: types.UInt16()},
	{Name: "browser_name", Type: types.Text().WithEnum([]string{"Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other"})},
	{Name: "browser_other", Type: types.Text()},
	{Name: "browser_version", Type: types.Text()},
	{Name: "location_city", Type: types.Text()},
	{Name: "location_country_code", Type: types.Text()},
	{Name: "location_country_name", Type: types.Text()},
	{Name: "location_latitude", Type: types.Float()},
	{Name: "location_longitude", Type: types.Float()},
	{Name: "device_type", Type: types.Text().WithEnum([]string{"desktop", "tablet", "mobile"})},
	{Name: "event", Type: types.Text()},
	{Name: "language", Type: types.Text(types.Chars(2))},
	{Name: "page_path", Type: types.Text()},
	{Name: "page_referrer", Type: types.Text()},
	{Name: "page_title", Type: types.Text()},
	{Name: "page_url", Type: types.Text()},
	{Name: "page_search", Type: types.Text()},
	{Name: "utm_source", Type: types.Text()},
	{Name: "utm_medium", Type: types.Text()},
	{Name: "utm_campaign", Type: types.Text()},
	{Name: "utm_term", Type: types.Text()},
	{Name: "utm_content", Type: types.Text()},
	{Name: "target", Type: types.Text()},
	{Name: "text", Type: types.Text()},
	{Name: "properties", Type: types.JSON()},
})
