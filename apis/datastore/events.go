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

	"github.com/meergo/meergo/apis/datastore/warehouses"
	"github.com/meergo/meergo/types"
)

const flushEventsQueueTimeout = 1 * time.Second // interval to flush queued Events the data warehouse

var eventsMergeTable = warehouses.MergeTable{
	Name:    "events",
	Columns: eventsColumnsForMerge,
	Keys: []warehouses.Column{
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

// eventsColumnsForMerge holds the events columns for the merge of events.
//
// Note that this list does not contain the "user" column, which is not written
// during the merge.
var eventsColumnsForMerge = []warehouses.Column{
	{Name: "anonymous_id", Type: types.Text()},
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
	{Name: "context_device_advertising_id", Type: types.Text()},
	{Name: "context_device_ad_tracking_enabled", Type: types.Boolean()},
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
	{Name: "context_user_agent", Type: types.Text()},
	{Name: "event", Type: types.Text()},
	{Name: "group_id", Type: types.Text()},
	{Name: "message_id", Type: types.Text()},
	{Name: "name", Type: types.Text()},
	{Name: "properties", Type: types.JSON()},
	{Name: "received_at", Type: types.DateTime()},
	{Name: "sent_at", Type: types.DateTime()},
	{Name: "source", Type: types.Int(32)},
	{Name: "timestamp", Type: types.DateTime()},
	{Name: "traits", Type: types.JSON()},
	{Name: "type", Type: types.Text().WithValues("alias", "identify", "group", "page", "screen", "track")},
	{Name: "user_id", Type: types.Text()},
}

// eventColumnByProperty maps each event property to the corresponding column.
var eventColumnByProperty = map[string]warehouses.Column{
	"user":                             {Name: "user", Type: types.UUID()},
	"anonymousId":                      {Name: "anonymous_id", Type: types.Text()},
	"category":                         {Name: "category", Type: types.Text()},
	"context.app.name":                 {Name: "context_app_name", Type: types.Text()},
	"context.app.version":              {Name: "context_app_version", Type: types.Text()},
	"context.app.build":                {Name: "context_app_build", Type: types.Text()},
	"context.app.namespace":            {Name: "context_app_namespace", Type: types.Text()},
	"context.browser.name":             {Name: "context_browser_name", Type: types.Text().WithValues("None", "Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other")},
	"context.browser.other":            {Name: "context_browser_other", Type: types.Text()},
	"context.browser.version":          {Name: "context_browser_version", Type: types.Text()},
	"context.campaign.name":            {Name: "context_campaign_name", Type: types.Text()},
	"context.campaign.source":          {Name: "context_campaign_source", Type: types.Text()},
	"context.campaign.medium":          {Name: "context_campaign_medium", Type: types.Text()},
	"context.campaign.term":            {Name: "context_campaign_term", Type: types.Text()},
	"context.campaign.content":         {Name: "context_campaign_content", Type: types.Text()},
	"context.device.id":                {Name: "context_device_id", Type: types.Text()},
	"context.device.advertisingId":     {Name: "context_device_advertising_id", Type: types.Text()},
	"context.device.adTrackingEnabled": {Name: "context_device_ad_tracking_enabled", Type: types.Boolean()},
	"context.device.manufacturer":      {Name: "context_device_manufacturer", Type: types.Text()},
	"context.device.model":             {Name: "context_device_model", Type: types.Text()},
	"context.device.name":              {Name: "context_device_name", Type: types.Text()},
	"context.device.type":              {Name: "context_device_type", Type: types.Text()},
	"context.device.token":             {Name: "context_device_token", Type: types.Text()},
	"context.ip":                       {Name: "context_ip", Type: types.Inet()},
	"context.library.name":             {Name: "context_library_name", Type: types.Text()},
	"context.library.version":          {Name: "context_library_version", Type: types.Text()},
	"context.locale":                   {Name: "context_locale", Type: types.Text()},
	"context.location.city":            {Name: "context_location_city", Type: types.Text()},
	"context.location.country":         {Name: "context_location_country", Type: types.Text()},
	"context.location.latitude":        {Name: "context_location_latitude", Type: types.Float(64)},
	"context.location.longitude":       {Name: "context_location_longitude", Type: types.Float(64)},
	"context.location.speed":           {Name: "context_location_speed", Type: types.Float(64)},
	"context.network.bluetooth":        {Name: "context_network_bluetooth", Type: types.Boolean()},
	"context.network.carrier":          {Name: "context_network_carrier", Type: types.Text()},
	"context.network.cellular":         {Name: "context_network_cellular", Type: types.Boolean()},
	"context.network.wifi":             {Name: "context_network_wifi", Type: types.Boolean()},
	"context.os.name":                  {Name: "context_os_name", Type: types.Text().WithValues("None", "Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other")},
	"context.os.version":               {Name: "context_os_version", Type: types.Text()},
	"context.page.path":                {Name: "context_page_path", Type: types.Text()},
	"context.page.referrer":            {Name: "context_page_referrer", Type: types.Text()},
	"context.page.search":              {Name: "context_page_search", Type: types.Text()},
	"context.page.title":               {Name: "context_page_title", Type: types.Text()},
	"context.page.url":                 {Name: "context_page_url", Type: types.Text()},
	"context.referrer.id":              {Name: "context_referrer_id", Type: types.Text()},
	"context.referrer.type":            {Name: "context_referrer_type", Type: types.Text()},
	"context.screen.width":             {Name: "context_screen_width", Type: types.Int(32)},
	"context.screen.height":            {Name: "context_screen_height", Type: types.Int(32)},
	"context.screen.density":           {Name: "context_screen_density", Type: types.Decimal(3, 2)},
	"context.session.id":               {Name: "context_session_id", Type: types.Int(64)},
	"context.session.start":            {Name: "context_session_start", Type: types.Boolean()},
	"context.timezone":                 {Name: "context_timezone", Type: types.Text()},
	"context.userAgent":                {Name: "context_user_agent", Type: types.Text()},
	"event":                            {Name: "event", Type: types.Text()},
	"groupId":                          {Name: "group_id", Type: types.Text()},
	"messageId":                        {Name: "message_id", Type: types.Text()},
	"name":                             {Name: "name", Type: types.Text()},
	"properties":                       {Name: "properties", Type: types.JSON()},
	"receivedAt":                       {Name: "received_at", Type: types.DateTime()},
	"sentAt":                           {Name: "sent_at", Type: types.DateTime()},
	"source":                           {Name: "source", Type: types.Int(32)},
	"timestamp":                        {Name: "timestamp", Type: types.DateTime()},
	"traits":                           {Name: "traits", Type: types.JSON()},
	"type":                             {Name: "type", Type: types.Text().WithValues("alias", "identify", "group", "page", "screen", "track")},
	"userId":                           {Name: "user_id", Type: types.Text()},
}
