// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"github.com/meergo/meergo/core/internal/schemas"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"
)

// EventColumnByPath returns the warehouses.Column corresponding to the property
// of the events schema with the specified path.
// propertyPath must always refer to an existing property in the event schema.
func EventColumnByPath(propertyPath string) warehouses.Column {
	return eventColumnByProperty[propertyPath]
}

// eventColumnNameFromPropertyPath maps a property path in the event schema
// to the corresponding column name in the events table.
//
// Note: The "originalTimestamp" property exists in the schema but does not have
// a corresponding column in the table.
var eventColumnNameFromPropertyPath = map[string]string{
	"mpid":                             "mpid",
	"connectionId":                     "connection_id",
	"anonymousId":                      "anonymous_id",
	"channel":                          "channel",
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
	"context.os.other":                 "context_os_other",
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
	"timestamp":                        "timestamp",
	"traits":                           "traits",
	"type":                             "type",
	"previousId":                       "previous_id",
	"userId":                           "user_id",
}

// eventColumnByProperty maps each event property to the corresponding column.
var eventColumnByProperty map[string]warehouses.Column

// eventsColumnsForMerge holds the events columns for the merge of events.
// It does not contain the "user" column, which is not written during the merge.
var eventsColumnsForMerge []warehouses.Column

// eventsMergeTable is the table used to merge the events.
var eventsMergeTable warehouses.Table

// init initializes eventColumnByProperty, eventsColumnsForMerge, and
// eventsMergeTable.
func init() {

	eventColumnByProperty = make(map[string]warehouses.Column, len(eventColumnNameFromPropertyPath))
	eventsColumnsForMerge = make([]warehouses.Column, len(eventColumnNameFromPropertyPath)-1)

	i := 0
	for path, p := range schemas.Event.Properties().WalkAll() {
		if p.Type.Kind() == types.ObjectKind || path == "originalTimestamp" {
			continue
		}
		name := eventColumnNameFromPropertyPath[path]
		c := warehouses.Column{
			Name: name,
			Type: p.Type,
			// In the database, nullable properties are those marked as read
			// optional in the event schema, meaning they may or may not be
			// present when the event is written and then retrieved. The others,
			// which are not read optional and thus always present, are NOT NULL
			// in the database.
			Nullable: p.ReadOptional,
		}
		eventColumnByProperty[path] = c
		if path != "mpid" {
			eventsColumnsForMerge[i] = c
			i++
		}
	}

	eventsMergeTable = warehouses.Table{
		Name:    "meergo_events",
		Columns: eventsColumnsForMerge,
		Keys:    []string{"message_id"},
	}

}
