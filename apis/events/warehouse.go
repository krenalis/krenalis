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
)

type warehouses struct {
	sync.Mutex
	state  *eventsState
	queues map[int]*warehouseQueue
}

func newWarehouses(st *eventsState) *warehouses {
	return &warehouses{state: st, queues: make(map[int]*warehouseQueue)}
}

// Add adds events to the data warehouses.
func (dw *warehouses) Add(events []*collectedEvent) {

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
	events     []*collectedEvent
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
	q.Lock()
	q.events = append(q.events, events...)
	var toFlush []*collectedEvent
	if len(q.events) == maxEventsQueueLen {
		toFlush = q.events
		q.events = nil
	}
	q.Unlock()
	if toFlush != nil {
		go q.flush(toFlush)
	}
}

var batchEventsColumns = []string{
	"source",
	"message_id",
	"anonymous_id",
	"user_id",
	"date",
	"timestamp",
	"sent_at",
	"received_at",
	"session_id",
	"session_start",
	"ip",
	"network_cellular",
	"network_wifi",
	"network_bluetooth",
	"network_carrier",
	"os_name",
	"os_version",
	"app_name",
	"app_version",
	"app_build",
	"app_namespace",
	"screen_density",
	"screen_width",
	"screen_height",
	"user_agent",
	"browser_name",
	"browser_other",
	"browser_version",
	"device_id",
	"device_name",
	"device_manufacturer",
	"device_model",
	"device_type",
	"device_version",
	"device_advertising_id",
	"device_token",
	"location_city",
	"location_country",
	"location_region",
	"location_latitude",
	"location_longitude",
	"location_speed",
	"event",
	"name",
	"locale",
	"timezone",
	"page_url",
	"page_path",
	"page_search",
	"page_hash",
	"page_title",
	"page_referrer",
	"referrer_type",
	"referrer_name",
	"referrer_url",
	"referrer_link",
	"campaign_name",
	"campaign_source",
	"campaign_medium",
	"campaign_term",
	"campaign_content",
	"library_name",
	"library_version",
	"properties",
}

// flush flushes a batch of events to the data warehouse.
func (q *warehouseQueue) flush(events []*collectedEvent) {
	if len(events) == 0 {
		return
	}
	log.Printf("[info] flushing %d events", len(events))
RETRY:
	for {
		wh, ok := q.state.Warehouse(q.workspace)
		if !ok {
			return
		}
		batch, err := wh.PrepareBatch(context.Background(), "events", batchEventsColumns)
		if err != nil {
			log.Printf("[error] cannot log events: %s", err)
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
			continue
		}
		var properties bytes.Buffer
		enc := json.NewEncoder(&properties)
		enc.SetEscapeHTML(false)
		for _, e := range events {
			properties.Reset()
			err = enc.Encode(e.Properties)
			if err != nil {
				log.Printf("[error] cannot marshal event: %s", err)
				continue
			}
			err = batch.Append(
				e.source,
				e.MessageID,
				e.AnonymousID,
				e.UserID,
				e.date,
				e.timestamp,
				e.sentAt,
				e.receivedAt,
				e.Context.SessionId,
				e.Context.SessionStart,
				e.ip,
				e.Context.Network.Cellular,
				e.Context.Network.WiFi,
				e.Context.Network.Bluetooth,
				e.Context.Network.Carrier,
				e.Context.OS.Name,
				e.Context.OS.Version,
				e.Context.App.Name,
				e.Context.App.Version,
				e.Context.App.Build,
				e.Context.App.Namespace,
				e.screen.density,
				e.screen.width,
				e.screen.height,
				e.userAgent,
				e.browser.name,
				e.browser.other,
				e.browser.version,
				e.Context.Device.ID,
				e.Context.Device.Name,
				e.Context.Device.Manufacturer,
				e.Context.Device.Model,
				e.Context.Device.Type,
				e.Context.Device.Version,
				e.Context.Device.AdvertisingID,
				e.Context.Device.Token,
				e.Context.Location.City,
				e.Context.Location.Country,
				e.Context.Location.Region,
				e.Context.Location.Latitude,
				e.Context.Location.Longitude,
				e.Context.Location.Speed,
				e.Event,
				e.Name,
				e.Context.Locale,
				e.Context.Timezone,
				e.page.url,
				e.page.path,
				e.page.search,
				e.page.hash,
				e.page.title,
				e.page.referrer,
				e.Context.Referrer.Type,
				e.Context.Referrer.Name,
				e.Context.Referrer.URL,
				e.Context.Referrer.Link,
				e.Context.Campaign.Name,
				e.Context.Campaign.Source,
				e.Context.Campaign.Medium,
				e.Context.Campaign.Term,
				e.Context.Campaign.Content,
				e.Context.Library.Name,
				e.Context.Library.Version,
				properties.Bytes(),
			)
			if err != nil {
				log.Printf("[error] cannot log events: %s", err)
				time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
				continue RETRY
			}
		}
		err = batch.Send()
		if err != nil {
			log.Printf("[error] cannot log events: %s", err)
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
			continue
		}
		break
	}
}
