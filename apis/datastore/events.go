//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package datastore

import (
	"github.com/meergo/meergo/apis/datastore/warehouses"
	"github.com/meergo/meergo/types"
)

// EventSchema is the schema for events as stored in the data warehouse.
var EventSchema = types.Object([]types.Property{
	{Name: "user", Type: types.UUID(), Nullable: true},
	{Name: "anonymousId", Type: types.Text()},
	{Name: "category", Type: types.Text()},
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
			},
			{
				Name: "browser",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.Text().WithValues("None", "Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other")},
					{Name: "other", Type: types.Text()},
					{Name: "version", Type: types.Text()},
				}),
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
			},
			{Name: "ip", Type: types.Inet()},
			{
				Name: "library",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.Text()},
					{Name: "version", Type: types.Text()},
				}),
			},
			{Name: "locale", Type: types.Text()},
			{
				Name: "location",
				Type: types.Object([]types.Property{
					{Name: "city", Type: types.Text()},
					{Name: "country", Type: types.Text()},
					{Name: "latitude", Type: types.Float(64)},
					{Name: "longitude", Type: types.Float(64)},
					{Name: "speed", Type: types.Float(64)},
				}),
			},
			{
				Name: "network",
				Type: types.Object([]types.Property{
					{Name: "bluetooth", Type: types.Boolean()},
					{Name: "carrier", Type: types.Text()},
					{Name: "cellular", Type: types.Boolean()},
					{Name: "wifi", Type: types.Boolean()},
				}),
			},
			{
				Name: "os",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.Text().WithValues("None", "Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other")},
					{Name: "version", Type: types.Text()},
				}),
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
			},
			{
				Name: "screen",
				Type: types.Object([]types.Property{
					{Name: "width", Type: types.Int(32)},
					{Name: "height", Type: types.Int(32)},
					{Name: "density", Type: types.Decimal(3, 2)},
				}),
			},
			{
				Name: "session",
				Type: types.Object([]types.Property{
					{Name: "id", Type: types.Int(64)},
					{Name: "start", Type: types.Boolean()},
				}),
			},
			{Name: "timezone", Type: types.Text()},
			{Name: "userAgent", Type: types.Text()},
		}),
	},
	{Name: "event", Type: types.Text()},
	{Name: "groupId", Type: types.Text()},
	{Name: "messageId", Type: types.Text()},
	{Name: "name", Type: types.Text()},
	{Name: "properties", Type: types.JSON()},
	{Name: "receivedAt", Type: types.DateTime()},
	{Name: "sentAt", Type: types.DateTime()},
	{Name: "source", Type: types.Int(32)},
	{Name: "timestamp", Type: types.DateTime()},
	{Name: "traits", Type: types.JSON()},
	{Name: "type", Type: types.Text().WithValues("alias", "identify", "group", "page", "screen", "track")},
	{Name: "userId", Type: types.Text()},
})

// eventColumnNameFromPropertyPath maps a property path to the corresponding
// column name in the events table.
var eventColumnNameFromPropertyPath = map[string]string{
	"user":                             "user",
	"anonymousId":                      "anonymous_id",
	"category":                         "category",
	"context.app.name":                 "context_app_name",
	"context.app.version":              "context_app_version",
	"context.app.build":                "context_app_build",
	"context.app.namespace":            "context_app_namespace",
	"context.browser.name":             "context_browser_name",
	"context.browser.other":            "context_browser_other",
	"context.browser.version":          "context_browser_version",
	"context.campaign.name":            "context_campaign_name",
	"context.campaign.source":          "context_campaign_source",
	"context.campaign.medium":          "context_campaign_medium",
	"context.campaign.term":            "context_campaign_term",
	"context.campaign.content":         "context_campaign_content",
	"context.device.id":                "context_device_id",
	"context.device.advertisingId":     "context_device_advertising_id",
	"context.device.adTrackingEnabled": "context_device_ad_tracking_enabled",
	"context.device.manufacturer":      "context_device_manufacturer",
	"context.device.model":             "context_device_model",
	"context.device.name":              "context_device_name",
	"context.device.type":              "context_device_type",
	"context.device.token":             "context_device_token",
	"context.ip":                       "context_ip",
	"context.library.name":             "context_library_name",
	"context.library.version":          "context_library_version",
	"context.locale":                   "context_locale",
	"context.location.city":            "context_location_city",
	"context.location.country":         "context_location_country",
	"context.location.latitude":        "context_location_latitude",
	"context.location.longitude":       "context_location_longitude",
	"context.location.speed":           "context_location_speed",
	"context.network.bluetooth":        "context_network_bluetooth",
	"context.network.carrier":          "context_network_carrier",
	"context.network.cellular":         "context_network_cellular",
	"context.network.wifi":             "context_network_wifi",
	"context.os.name":                  "context_os_name",
	"context.os.version":               "context_os_version",
	"context.page.path":                "context_page_path",
	"context.page.referrer":            "context_page_referrer",
	"context.page.search":              "context_page_search",
	"context.page.title":               "context_page_title",
	"context.page.url":                 "context_page_url",
	"context.referrer.id":              "context_referrer_id",
	"context.referrer.type":            "context_referrer_type",
	"context.screen.width":             "context_screen_width",
	"context.screen.height":            "context_screen_height",
	"context.screen.density":           "context_screen_density",
	"context.session.id":               "context_session_id",
	"context.session.start":            "context_session_start",
	"context.timezone":                 "context_timezone",
	"context.userAgent":                "context_user_agent",
	"event":                            "event",
	"groupId":                          "group_id",
	"messageId":                        "message_id",
	"name":                             "name",
	"properties":                       "properties",
	"receivedAt":                       "received_at",
	"sentAt":                           "sent_at",
	"source":                           "source",
	"timestamp":                        "timestamp",
	"traits":                           "traits",
	"type":                             "type",
	"userId":                           "user_id",
}

// eventColumnByProperty maps each event property to the corresponding column.
var eventColumnByProperty map[string]warehouses.Column

// eventsColumnsForMerge holds the events columns for the merge of events.
// It does not contain the "user" column, which is not written during the merge.
var eventsColumnsForMerge []warehouses.Column

// eventsMergeTable is the merge table used to merge the events.
var eventsMergeTable warehouses.MergeTable

// init initializes eventColumnByProperty, eventsColumnsForMerge, and
// eventsMergeTable.
func init() {

	eventColumnByProperty = make(map[string]warehouses.Column, len(eventColumnNameFromPropertyPath))
	eventsColumnsForMerge = make([]warehouses.Column, len(eventColumnNameFromPropertyPath)-1)

	i := 0
	for path, p := range types.Walk(EventSchema) {
		if p.Type.Kind() == types.ObjectKind {
			continue
		}
		name := eventColumnNameFromPropertyPath[path]
		c := warehouses.Column{Name: name, Type: p.Type, Nullable: p.Nullable}
		eventColumnByProperty[path] = c
		if path != "user" {
			eventsColumnsForMerge[i] = c
			i++
		}
	}

	eventsMergeTable = warehouses.MergeTable{
		Name:    "events",
		Columns: eventsColumnsForMerge,
		Keys:    []string{"source", "message_id"},
	}

}
