//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package datastore

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"chichi/apis/datastore/warehouses"
	"chichi/connector/types"
)

const flushEventsQueueTimeout = 1 * time.Second // interval to flush queued Events the data warehouse

var eventsMergeTable = warehouses.MergeTable{
	Name: "events",
	Columns: []types.Property{
		{Name: "gid", Type: types.Int(32)},
		{Name: "anonymousId", Type: types.Text()},
		{Name: "category", Type: types.Text()},
		{Name: "context_app_name", Type: types.Text()},
		{Name: "context_app_version", Type: types.Text()},
		{Name: "context_app_build", Type: types.Text()},
		{Name: "context_app_namespace", Type: types.Text()},
		{Name: "context_browser_name", Type: types.Text().WithValues("None", "Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other")},
		{Name: "context_browser_other", Type: types.Text()},
		{Name: "context_browser_version", Type: types.Text()},
		{Name: "context_campaign_name", Type: types.Text()},
		{Name: "context_campaign_source", Type: types.Text()},
		{Name: "context_campaign_medium", Type: types.Text()},
		{Name: "context_campaign_term", Type: types.Text()},
		{Name: "context_campaign_content", Type: types.Text()},
		{Name: "context_device_id", Type: types.Text()},
		{Name: "context_device_advertisingId", Type: types.Text()},
		{Name: "context_device_adTrackingEnabled", Type: types.Boolean()},
		{Name: "context_device_manufacturer", Type: types.Text()},
		{Name: "context_device_model", Type: types.Text()},
		{Name: "context_device_name", Type: types.Text()},
		{Name: "context_device_type", Type: types.Text()},
		{Name: "context_device_token", Type: types.Text()},
		{Name: "context_ip", Type: types.Inet()},
		{Name: "context_library_name", Type: types.Text()},
		{Name: "context_library_version", Type: types.Text()},
		{Name: "context_locale", Type: types.Text()},
		{Name: "context_location_city", Type: types.Text()},
		{Name: "context_location_country", Type: types.Text()},
		{Name: "context_location_latitude", Type: types.Float(64)},
		{Name: "context_location_longitude", Type: types.Float(64)},
		{Name: "context_location_speed", Type: types.Float(64)},
		{Name: "context_network_bluetooth", Type: types.Boolean()},
		{Name: "context_network_carrier", Type: types.Text()},
		{Name: "context_network_cellular", Type: types.Boolean()},
		{Name: "context_network_wifi", Type: types.Boolean()},
		{Name: "context_os_name", Type: types.Text().WithValues("None", "Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other")},
		{Name: "context_os_version", Type: types.Text()},
		{Name: "context_page_path", Type: types.Text()},
		{Name: "context_page_referrer", Type: types.Text()},
		{Name: "context_page_search", Type: types.Text()},
		{Name: "context_page_title", Type: types.Text()},
		{Name: "context_page_url", Type: types.Text()},
		{Name: "context_referrer_id", Type: types.Text()},
		{Name: "context_referrer_type", Type: types.Text()},
		{Name: "context_screen_width", Type: types.Int(32)},
		{Name: "context_screen_height", Type: types.Int(32)},
		{Name: "context_screen_density", Type: types.Decimal(3, 2)},
		{Name: "context_session_id", Type: types.Int(64)},
		{Name: "context_session_start", Type: types.Boolean()},
		{Name: "context_timezone", Type: types.Text()},
		{Name: "context_userAgent", Type: types.Text()},
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
	},
	PrimaryKeys: []types.Property{
		{Name: "messageId", Type: types.Text()},
	},
}

// flushEvents flushes a batch of events to the data warehouse.
func (store *Store) flushEvents(events [][]any) {
	slog.Info("flush events", "count", len(events))
	for {
		err := store.warehouse.Merge(context.Background(), eventsMergeTable, events, nil)
		if err != nil {
			slog.Error("cannot flush the event queue", "workspace", store.workspace, "err", err)
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
			continue
		}
		break
	}
}
