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
		{Name: "gid", Type: types.Int()},
		{Name: "anonymous_id", Type: types.Text()},
		{Name: "category", Type: types.Text()},
		{Name: "app_name", Type: types.Text()},
		{Name: "app_version", Type: types.Text()},
		{Name: "app_build", Type: types.Text()},
		{Name: "app_namespace", Type: types.Text()},
		{Name: "browser_name", Type: types.Text().WithValues("Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other")},
		{Name: "browser_other", Type: types.Text()},
		{Name: "browser_version", Type: types.Text()},
		{Name: "campaign_name", Type: types.Text()},
		{Name: "campaign_source", Type: types.Text()},
		{Name: "campaign_medium", Type: types.Text()},
		{Name: "campaign_term", Type: types.Text()},
		{Name: "campaign_content", Type: types.Text()},
		{Name: "device_id", Type: types.Text()},
		{Name: "device_advertising_id", Type: types.Text()},
		{Name: "device_ad_tracking_enabled", Type: types.Boolean()},
		{Name: "device_manufacturer", Type: types.Text()},
		{Name: "device_model", Type: types.Text()},
		{Name: "device_name", Type: types.Text()},
		{Name: "device_type", Type: types.Text()},
		{Name: "device_token", Type: types.Text()},
		{Name: "ip", Type: types.Inet()},
		{Name: "library_name", Type: types.Text()},
		{Name: "library_version", Type: types.Text()},
		{Name: "locale", Type: types.Text()},
		{Name: "location_city", Type: types.Text()},
		{Name: "location_country", Type: types.Text()},
		{Name: "location_latitude", Type: types.Float()},
		{Name: "location_longitude", Type: types.Float()},
		{Name: "location_speed", Type: types.Float()},
		{Name: "network_bluetooth", Type: types.Boolean()},
		{Name: "network_carrier", Type: types.Text()},
		{Name: "network_cellular", Type: types.Boolean()},
		{Name: "network_wifi", Type: types.Boolean()},
		{Name: "os_name", Type: types.Text().WithValues("Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other")},
		{Name: "os_version", Type: types.Text()},
		{Name: "page_path", Type: types.Text()},
		{Name: "page_referrer", Type: types.Text()},
		{Name: "page_search", Type: types.Text()},
		{Name: "page_title", Type: types.Text()},
		{Name: "page_url", Type: types.Text()},
		{Name: "referrer_id", Type: types.Text()},
		{Name: "referrer_type", Type: types.Text()},
		{Name: "screen_width", Type: types.Int()},
		{Name: "screen_height", Type: types.Int()},
		{Name: "screen_density", Type: types.Float()},
		{Name: "session_id", Type: types.Int64()},
		{Name: "session_start", Type: types.Boolean()},
		{Name: "timezone", Type: types.Text()},
		{Name: "user_agent", Type: types.Text()},
		{Name: "event", Type: types.Text()},
		{Name: "group_id", Type: types.Text()},
		{Name: "message_id", Type: types.Text()},
		{Name: "name", Type: types.Text()},
		{Name: "properties", Type: types.JSON()},
		{Name: "received_at", Type: types.DateTime()},
		{Name: "sent_at", Type: types.DateTime()},
		{Name: "source", Type: types.Int()},
		{Name: "timestamp", Type: types.DateTime()},
		{Name: "traits", Type: types.JSON()},
		{Name: "type", Type: types.Text().WithValues("alias", "identify", "group", "page", "screen", "track")},
		{Name: "user_id", Type: types.Text()},
	},
	PrimaryKeys: []types.Property{
		{Name: "message_id", Type: types.Text()},
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
