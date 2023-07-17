//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package events

import (
	"context"
	"sync"
	"time"

	"chichi/apis/mappings"
	"chichi/apis/state"
	"chichi/connector"
)

const pipeSize = 100

const (
	flushQueueTimeout = 1 * time.Second // interval to flushEvents the warehouseQueue
	geoLite2Path      = "GeoLite2-City.mmdb"
	maxEventSize      = 32 * 1024
	maxEventsQueueLen = 10000
)

// eventDateLayout is the layout used for dates in events.
const eventDateLayout = "2006-01-02T15:04:05.999Z"

type processedEvent struct {
	*collectedEvent
	action      *state.Action
	destination int
	eventType   string
	endpoint    int
	mappedEvent map[string]any
	inEvent     connector.Event
	err         error
}

// Processor processes events received from source streams and sent them to
// their data warehouses.
type Processor struct {
	sync.Mutex // for the streams field.
	ctx        context.Context
	state      *eventsState
	events     struct {
		in  <-chan *collectedEvent
		out chan *processedEvent
	}
	eventLog *eventsLog
}

// newProcessor returns a new processor.
func newProcessor(ctx context.Context, st *eventsState, eventLog *eventsLog, events <-chan *collectedEvent) (*Processor, error) {

	processor := Processor{
		ctx:      ctx,
		state:    st,
		eventLog: eventLog,
	}
	processor.events.in = events
	processor.events.out = make(chan *processedEvent, pipeSize)

	// Starts the workers.
	for i := 0; i < 10; i++ {
		go func() {
			for {
				select {
				case event := <-events:
					for _, action := range st.Actions() {
						if action.Target != state.EventsTarget {
							continue
						}
						// Convert the collectedEvent to a map of properties.
						mapEvent, err := collectedEventToMap(event)
						if err != nil {
							eventLog.TransformationFailed(event.id, action.ID, err)
							continue
						}
						// Check if the filter applies.
						ok, err := mappings.ActionFilterApplies(action.Filter, mapEvent)
						if err != nil {
							eventLog.TransformationFailed(event.id, action.ID, err)
							continue
						}
						if !ok {
							continue
						}
						var mappedEvent map[string]any
						// If the action's input schema is valid (which means
						// that there is a mapping or a transformation defined),
						// apply the mapping or the transformation.
						if action.InSchema.Valid() {
							mapping, err := mappings.New(action.InSchema, action.OutSchema, action.Mapping, action.Transformation, false)
							if err != nil {
								eventLog.TransformationFailed(event.id, action.ID, err)
								continue
							}
							mappedEvent, err = mapping.Apply(ctx, mapEvent)
							if err != nil {
								eventLog.TransformationFailed(event.id, action.ID, err)
								continue
							}
						}
						ev := &processedEvent{
							collectedEvent: event,
							action:         action,
							destination:    action.Connection().ID,
							eventType:      action.EventType,
							mappedEvent:    mappedEvent,
							// TODO(Gianluca): since the endpoints have been
							// removed from the action, we do not have
							// information about the endpoint. We should
							// review/refactor this.
							//
							// See the issue https://github.com/open2b/chichi/issues/194.
							//
							endpoint: 0,
							inEvent:  eventToConnectorEvent(event),
						}
						processor.events.out <- ev
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	return &processor, nil
}

// Events returns the processed events channel.
func (processor *Processor) Events() <-chan *processedEvent {
	return processor.events.out
}

// eventToConnectorEvent returns the connector.Event corresponding to event.
func eventToConnectorEvent(event *collectedEvent) connector.Event {
	// Keep in sync with the schema in "apis/events/schema.go".
	e := connector.Event{}
	e.MessageID = event.MessageID
	e.AnonymousID = event.AnonymousID
	e.UserID = event.UserID
	e.Timestamp = event.timestamp
	e.SentAt = event.sentAt
	e.ReceivedAt = event.receivedAt
	e.IP = event.ip
	e.Network.Cellular = event.Context.Network.Cellular
	e.Network.WiFi = event.Context.Network.WiFi
	e.Network.Bluetooth = event.Context.Network.Bluetooth
	e.Network.Carrier = event.Context.Network.Carrier
	e.OS.Name = event.Context.OS.Name
	e.OS.Version = event.Context.OS.Version
	e.App.Name = event.Context.App.Name
	e.App.Version = event.Context.App.Version
	e.App.Build = event.Context.App.Build
	e.App.Namespace = event.Context.App.Namespace
	e.UserAgent = event.userAgent
	e.Screen.Density = event.screen.density
	e.Screen.Width = event.screen.width
	e.Screen.Height = event.screen.height
	e.Browser.Name = event.browser.name
	e.Browser.Other = event.browser.other
	e.Browser.Version = event.browser.version
	e.Location.City = event.Context.Location.City
	e.Location.Country = event.Context.Location.Country
	e.Location.Region = event.Context.Location.Region
	e.Location.Latitude = event.Context.Location.Latitude
	e.Location.Longitude = event.Context.Location.Longitude
	e.Location.Speed = event.Context.Location.Speed
	e.Event = event.Event
	e.Name = event.Name
	e.Locale = event.Context.Locale
	e.Page.URL = event.page.url
	e.Page.Path = event.page.path
	e.Page.Search = event.page.search
	e.Page.Hash = event.page.hash
	e.Page.Title = event.page.title
	e.Page.Referrer = event.page.referrer
	e.Referrer.Type = event.Context.Referrer.Type
	e.Referrer.Name = event.Context.Referrer.Name
	e.Referrer.URL = event.Context.Referrer.URL
	e.Referrer.Link = event.Context.Referrer.Link
	e.Campaign.Name = event.Context.Campaign.Name
	e.Campaign.Source = event.Context.Campaign.Source
	e.Campaign.Medium = event.Context.Campaign.Medium
	e.Campaign.Term = event.Context.Campaign.Term
	e.Campaign.Content = event.Context.Campaign.Content
	e.Library.Name = event.Context.Library.Name
	e.Library.Version = event.Context.Library.Version
	return e
}

// collectedEventToMap returns a map built from the properties of the given
// collectedEvent.
func collectedEventToMap(event *collectedEvent) (map[string]any, error) {

	// Keep in sync with the schema in "apis/events/schema.go".

	// TODO(Gianluca): define datetime layout and parse/convert the values.
	mapEvent := map[string]any{
		"source":       event.source,
		"message_id":   event.MessageID,
		"anonymous_id": event.AnonymousID,
		"user_id":      event.UserID,
		"date":         event.date,
		"timestamp":    event.timestamp,
		"sent_at":      event.sentAt,
		"received_at":  event.receivedAt,
		"ip":           event.ip,
		"network": map[string]any{
			"cellular":  event.Context.Network.Cellular,
			"wifi":      event.Context.Network.WiFi,
			"bluetooth": event.Context.Network.Bluetooth,
			"carrier":   event.Context.Network.Carrier,
		},
		"os": map[string]any{
			"name":    event.Context.OS.Name,
			"version": event.Context.OS.Version,
		},
		"app": map[string]any{
			"name":      event.Context.App.Name,
			"version":   event.Context.App.Version,
			"build":     event.Context.App.Build,
			"namespace": event.Context.App.Namespace,
		},
		"screen": map[string]any{
			"density": event.screen.density,
			"width":   event.screen.width,
			"height":  event.screen.height,
		},
		"user_agent": event.userAgent,
		"browser": map[string]any{
			"name":    event.browser.name,
			"other":   event.browser.other,
			"version": event.browser.version,
		},
		"device": map[string]any{
			"id":             event.Context.Device.Type,
			"name":           event.Context.Device.Name,
			"manufacturer":   event.Context.Device.Manufacturer,
			"model":          event.Context.Device.Model,
			"type":           event.Context.Device.Type,
			"version":        event.Context.Device.Version,
			"advertising_id": event.Context.Device.AdvertisingID,
			"token":          event.Context.Device.Token,
		},
		"location": map[string]any{
			"city":      event.Context.Location.City,
			"country":   event.Context.Location.Country,
			"region":    event.Context.Location.Region,
			"latitude":  event.Context.Location.Latitude,
			"longitude": event.Context.Location.Longitude,
			"speed":     event.Context.Location.Speed,
		},
		"event":    event.Event,
		"name":     event.Name,
		"locale":   event.Context.Locale,
		"timezone": event.Context.Timezone,
		"page": map[string]any{
			"url":      event.page.url,
			"path":     event.page.path,
			"search":   event.page.search,
			"hash":     event.page.hash,
			"title":    event.page.title,
			"referrer": event.page.referrer,
		},
		"referrer": map[string]any{
			"type": event.Context.Referrer.Type,
			"name": event.Context.Referrer.Name,
			"url":  event.Context.Referrer.URL,
			"link": event.Context.Referrer.Link,
		},
		"campaign": map[string]any{
			"name":    event.Context.Campaign.Name,
			"source":  event.Context.Campaign.Source,
			"medium":  event.Context.Campaign.Medium,
			"term":    event.Context.Campaign.Term,
			"content": event.Context.Campaign.Content,
		},
		"library": map[string]any{
			"name":    event.Context.Library.Name,
			"version": event.Context.Library.Version,
		},
		"properties": event.Properties,
	}

	return mapEvent, nil
}
