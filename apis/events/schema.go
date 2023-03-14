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

var Schema = types.Object([]types.Property{
	{Name: "source", Type: types.Int()},
	{Name: "anonymous_id", Type: types.UUID()},
	{Name: "user_id", Type: types.Text()},
	{Name: "date", Type: types.Date(time.DateOnly)},
	{Name: "timestamp", Type: types.DateTime(time.StampMilli)},
	{Name: "sent_at", Type: types.DateTime(time.StampMilli)},
	{Name: "received_at", Type: types.DateTime(time.StampMilli)},
	{Name: "ip", Type: types.Inet()},
	{
		Name: "os",
		Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text().WithEnum([]string{"Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other"})},
			{Name: "version", Type: types.Text()},
		}),
	},
	{Name: "user_agent", Type: types.Text()},
	{
		Name: "screen",
		Type: types.Object([]types.Property{
			{Name: "density", Type: types.UInt16()},
			{Name: "width", Type: types.UInt16()},
			{Name: "height", Type: types.UInt16()},
		}),
	},
	{
		Name: "browser",
		Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text().WithEnum([]string{"Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other"})},
			{Name: "other", Type: types.Text()},
			{Name: "version", Type: types.Text()},
		}),
	},
	{
		Name: "location",
		Type: types.Object([]types.Property{
			{Name: "city", Type: types.Text()},
			{Name: "country", Type: types.Object([]types.Property{
				{Name: "code", Type: types.Text()},
				{Name: "name", Type: types.Text()},
			})},
			{Name: "latitude", Type: types.Float()},
			{Name: "longitude", Type: types.Float()},
		}),
	},
	{Name: "device_type", Type: types.Text().WithEnum([]string{"desktop", "tablet", "mobile"})},
	{Name: "event", Type: types.Text()},
	{Name: "language", Type: types.Text(types.Chars(2))},
	{
		Name: "page",
		Type: types.Object([]types.Property{
			{Name: "path", Type: types.Text()},
			{Name: "referrer", Type: types.Text()},
			{Name: "title", Type: types.Text()},
			{Name: "url", Type: types.Text()},
			{Name: "search", Type: types.Text()},
		}),
	},
	{
		Name: "utm",
		Type: types.Object([]types.Property{
			{Name: "source", Type: types.Text()},
			{Name: "medium", Type: types.Text()},
			{Name: "campaign", Type: types.Text()},
			{Name: "term", Type: types.Text()},
			{Name: "content", Type: types.Text()},
		}),
	},
	{Name: "target", Type: types.Text()},
	{Name: "text", Type: types.Text()},
	{Name: "properties", Type: types.JSON()},
})
