//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package events

import (
	"context"
	"log"
	"math/rand"
	"strings"
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
	"location_city",
	"location_country",
	"location_region",
	"location_latitude",
	"location_longitude",
	"location_speed",
	"event",
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
		for _, e := range events {
			err = batch.Append(
				e.source,
				e.MessageID,
				e.AnonymousID,
				e.UserID,
				e.date,
				e.timestamp,
				e.sentAt,
				e.receivedAt,
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
				e.Context.Location.City,
				e.Context.Location.Country,
				e.Context.Location.Region,
				e.Context.Location.Latitude,
				e.Context.Location.Longitude,
				e.Context.Location.Speed,
				e.Event,
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
				e.properties,
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

// isDeviceType reports whether t is a device type.
func isDeviceType(t string) bool {
	switch t {
	case "Mobile", "Tablet", "Desktop":
		return true
	}
	return false
}

// abbreviate abbreviates s to almost n runes. If s is longer than n runes,
// the abbreviated string terminates with "...".
func abbreviate(s string, n int) string {
	const spaces = " \n\r\t\f" // https://infra.spec.whatwg.org/#ascii-whitespace
	s = strings.TrimRight(s, spaces)
	if len(s) <= n {
		return s
	}
	if n < 3 {
		return ""
	}
	p := 0
	n2 := 0
	for i := range s {
		switch p {
		case n - 2:
			n2 = i
		case n:
			break
		}
		p++
	}
	if p < n {
		return s
	}
	if p = strings.LastIndexAny(s[:n2], spaces); p > 0 {
		s = strings.TrimRight(s[:p], spaces)
	} else {
		s = ""
	}
	if l := len(s) - 1; l >= 0 && (s[l] == '.' || s[l] == ',') {
		s = s[:l]
	}
	return s + "..."
}

// asciiEqualFold is strings.EqualFold, ASCII only. It reports whether s and t
// are equal, ASCII-case-insensitively.
//
// This function is the ascii.EqualFold internal function of the http package
// of the Go standard library, copyright Go Authors.
func asciiEqualFold(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if lower(s[i]) != lower(t[i]) {
			return false
		}
	}
	return true
}

// lower returns the ASCII lowercase version of b.
//
// This function is the ascii.lower internal function of the http package of
// the Go standard library, copyright Go Authors.
func lower(b byte) byte {
	if 'A' <= b && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}
