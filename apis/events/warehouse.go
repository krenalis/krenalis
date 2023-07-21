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

	"gid",
	"anonymous_id",
	"category",

	// app.
	"app_name",
	"app_version",
	"app_build",
	"app_namespace",

	// browser.
	"browser_name",
	"browser_other",
	"browser_version",

	// campaign.
	"campaign_name",
	"campaign_source",
	"campaign_medium",
	"campaign_term",
	"campaign_content",

	// device.
	"device_id",
	"device_advertising_id",
	"device_ad_tracking_enabled",
	"device_manufacturer",
	"device_model",
	"device_name",
	"device_type",
	"device_token",

	"ip",

	// library.
	"library_name",
	"library_version",

	"locale",

	// location.
	"location_city",
	"location_country",
	"location_latitude",
	"location_longitude",
	"location_speed",

	// network.
	"network_bluetooth",
	"network_carrier",
	"network_cellular",
	"network_wifi",

	// os.
	"os_name",
	"os_version",

	// page.
	"page_path",
	"page_referrer",
	"page_search",
	"page_title",
	"page_url",

	// referrer.
	"referrer_id",
	"referrer_type",

	// screen.
	"screen_width",
	"screen_height",
	"screen_density",

	// session.
	"session_id",
	"session_start",

	"timezone",
	"user_agent",

	"event",
	"group_id",
	"message_id",
	"name",
	"properties",
	"received_at",
	"sent_at",
	"source",
	"timestamp",
	"traits",
	"type",
	"user_id",
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
		var traits bytes.Buffer
		traitsEnc := json.NewEncoder(&traits)
		traitsEnc.SetEscapeHTML(false)
		var properties bytes.Buffer
		propertiesEnc := json.NewEncoder(&properties)
		propertiesEnc.SetEscapeHTML(false)
		for _, e := range events {
			traits.Reset()
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
			err = batch.Append(

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
				e.SentAt,
				e.source,
				e.Timestamp,
				traits.Bytes(),
				*e.Type,
				e.UserId,
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
