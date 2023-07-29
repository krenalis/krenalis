//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package events

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"sync"
	"time"

	"chichi/apis/warehouses"
	"chichi/connector/types"
)

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
		{Name: "browser_name", Type: types.Text().WithEnum([]string{"Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other"})},
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
		{Name: "os_name", Type: types.Text().WithEnum([]string{"Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other"})},
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
		{Name: "type", Type: types.Text().WithEnum([]string{"alias", "identify", "group", "page", "screen", "track"})},
		{Name: "user_id", Type: types.Text()},
	},
	PrimaryKeys: []string{"message_id"},
}

type warehouse struct {
	sync.Mutex
	state  *eventsState
	queues map[int]*warehouseQueue
}

func newWarehouses(st *eventsState) *warehouse {
	return &warehouse{state: st, queues: make(map[int]*warehouseQueue)}
}

// Add adds events to the data warehouses.
func (dw *warehouse) Add(events []*collectedEvent) {

	source, ok := dw.state.Source(events[0].header.source)
	if !ok {
		return
	}
	workspaceID := source.Workspace().ID

	// Add the events to the warehouseQueue of the workspace.
	dw.Lock()
	queue, ok := dw.queues[workspaceID]
	if !ok {
		queue = newQueue(dw.state, workspaceID)
		dw.queues[workspaceID] = queue
	}
	dw.Unlock()
	queue.add(events)

}

// warehouseQueue is the queue of events that are waiting to be written to the
// data warehouse of a workspace.
type warehouseQueue struct {
	sync.Mutex // for the events field.
	state      *eventsState
	workspace  int
	events     [][]any
}

// newQueue returns a new warehouseQueue that flushed events into the given data
// warehouse.
func newQueue(state *eventsState, workspace int) *warehouseQueue {
	q := &warehouseQueue{state: state, workspace: workspace}
	// Start a goroutine that flushes the events every flushQueueTimeout seconds.
	go func() {
		ticker := time.NewTicker(flushQueueTimeout)
		for {
			select {
			case <-ticker.C:
				q.Lock()
				toFlush := q.events
				q.events = nil
				q.Unlock()
				go q.flush(toFlush)
			}
		}
	}()
	return q
}

// add adds events to the queue.
func (q *warehouseQueue) add(events []*collectedEvent) {

	var traits bytes.Buffer
	traitsEnc := json.NewEncoder(&traits)
	traitsEnc.SetEscapeHTML(false)
	var properties bytes.Buffer
	propertiesEnc := json.NewEncoder(&properties)
	propertiesEnc.SetEscapeHTML(false)

	rows := make([][]any, len(events))

	for i, e := range events {

		traits.Reset()

		var err error
		if *e.Type == "identify" || *e.Type == "group" {
			err = traitsEnc.Encode(e.Traits)
		} else {
			err = traitsEnc.Encode(e.Context.Traits)
		}
		if err != nil {
			log.Printf("[error] cannot marshal event: %s", err)
			continue
		}
		properties.Reset()
		err = propertiesEnc.Encode(e.Properties)
		if err != nil {
			log.Printf("[error] cannot marshal event: %s", err)
			continue
		}
		groupId := e.GroupId
		if *e.Type != "group" {
			groupId = e.Context.GroupId
		}

		rows[i] = []any{
			0,
			e.AnonymousId,
			e.Category,

			// app.
			e.Context.App.Name,
			e.Context.App.Version,
			e.Context.App.Build,
			e.Context.App.Namespace,

			// browser.
			e.Context.browser.Name,
			e.Context.browser.Other,
			e.Context.browser.Version,

			// campaign.
			e.Context.Campaign.Name,
			e.Context.Campaign.Source,
			e.Context.Campaign.Medium,
			e.Context.Campaign.Term,
			e.Context.Campaign.Content,

			// device.
			e.Context.Device.Id,
			e.Context.Device.AdvertisingId,
			e.Context.Device.AdTrackingEnabled,
			e.Context.Device.Manufacturer,
			e.Context.Device.Model,
			e.Context.Device.Name,
			e.Context.Device.Type,
			e.Context.Device.Token,

			e.Context.IP,

			// library.
			e.Context.Library.Name,
			e.Context.Library.Version,

			e.Context.Locale,

			// location.
			e.Context.Location.City,
			e.Context.Location.Country,
			e.Context.Location.Latitude,
			e.Context.Location.Longitude,
			e.Context.Location.Speed,

			// network.
			e.Context.Network.Bluetooth,
			e.Context.Network.Carrier,
			e.Context.Network.Cellular,
			e.Context.Network.WiFi,

			// os.
			e.Context.OS.Name,
			e.Context.OS.Version,

			// page.
			e.Context.Page.Path,
			e.Context.Page.Referrer,
			e.Context.Page.Search,
			e.Context.Page.Title,
			e.Context.Page.URL,

			// referrer.
			e.Context.Referrer.Id,
			e.Context.Referrer.Type,

			// screen.
			int16(e.Context.Screen.Width),
			int16(e.Context.Screen.Height),
			int16(e.Context.Screen.Density),

			// session.
			e.Context.SessionId,
			e.Context.SessionStart,

			e.Context.Timezone,
			e.Context.UserAgent,

			e.Event,
			groupId,
			e.MessageId,
			e.Name,
			properties.Bytes(),
			e.receivedAt,
			e.sentAt,
			e.source,
			e.timestamp,
			traits.Bytes(),
			*e.Type,
			e.UserId,
		}
	}

	q.Lock()
	q.events = append(q.events, rows...)
	var toFlush [][]any
	if len(q.events) == maxEventsQueueLen {
		toFlush = q.events
		q.events = nil
	}
	q.Unlock()
	if toFlush != nil {
		go q.flush(toFlush)
	}

}

// flush flushes a batch of events to the data warehouse.
func (q *warehouseQueue) flush(rows [][]any) {
	if len(rows) == 0 {
		return
	}
	log.Printf("[info] flushing %d events", len(rows))
RETRY:
	for {
		wh, ok := q.state.Warehouse(q.workspace)
		if !ok {
			return
		}
		err := wh.Merge(context.Background(), eventsMergeTable, rows, nil)
		if err != nil {
			log.Printf("[error] cannot log events: %s", err)
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
			continue RETRY
		}
		break
	}
}
