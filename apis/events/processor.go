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

	"chichi/apis/mappings"
	"chichi/apis/state"
	"chichi/connector"
)

const pipeSize = 100

const (
	geoLite2Path = "GeoLite2-City.mmdb"
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
	close    struct {
		ctx       context.Context
		cancelCtx context.CancelFunc
	}
}

// newProcessor returns a new processor.
func newProcessor(st *eventsState, eventLog *eventsLog, events <-chan *collectedEvent) (*Processor, error) {

	processor := Processor{
		state:    st,
		eventLog: eventLog,
	}
	processor.events.in = events
	processor.events.out = make(chan *processedEvent, pipeSize)
	processor.close.ctx, processor.close.cancelCtx = context.WithCancel(context.Background())

	ctx := processor.close.ctx

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

// Close closes the processor.
func (processor *Processor) Close() {
	processor.close.cancelCtx()
}

// Events returns the processed events channel.
func (processor *Processor) Events() <-chan *processedEvent {
	return processor.events.out
}

// eventToConnectorEvent returns the connector.Event corresponding to event.
func eventToConnectorEvent(event *collectedEvent) connector.Event {
	// Keep in sync with the connector.Event type.
	groupId := event.GroupId
	if event.GroupId == "" {
		groupId = event.Context.GroupId
	}
	e := connector.Event{}
	e.AnonymousId = event.AnonymousId
	e.Category = event.Category
	e.Context.Active = event.Context.Active
	e.Context.App.Name = event.Context.App.Name
	e.Context.App.Version = event.Context.App.Version
	e.Context.App.Build = event.Context.App.Build
	e.Context.App.Namespace = event.Context.App.Namespace
	e.Context.Campaign.Name = event.Context.Campaign.Name
	e.Context.Campaign.Source = event.Context.Campaign.Source
	e.Context.Campaign.Medium = event.Context.Campaign.Medium
	e.Context.Campaign.Term = event.Context.Campaign.Term
	e.Context.Campaign.Content = event.Context.Campaign.Content
	e.Context.Device.Id = event.Context.Device.Id
	e.Context.Device.AdvertisingId = event.Context.Device.AdvertisingId
	e.Context.Device.AdTrackingEnabled = event.Context.Device.AdTrackingEnabled
	e.Context.Device.Manufacturer = event.Context.Device.Manufacturer
	e.Context.Device.Model = event.Context.Device.Model
	e.Context.Device.Name = event.Context.Device.Name
	e.Context.Device.Type = event.Context.Device.Type
	e.Context.Device.Token = event.Context.Device.Token
	e.Context.Device.Token = event.Context.Device.Token
	e.Context.IP = event.Context.IP
	e.Context.Library.Name = event.Context.Library.Name
	e.Context.Library.Version = event.Context.Library.Version
	e.Context.Locale = event.Context.Locale
	e.Context.Location.City = event.Context.Location.City
	e.Context.Location.Country = event.Context.Location.Country
	e.Context.Location.Latitude = event.Context.Location.Latitude
	e.Context.Location.Longitude = event.Context.Location.Longitude
	e.Context.Location.Speed = event.Context.Location.Speed
	e.Context.Network.Bluetooth = event.Context.Network.Bluetooth
	e.Context.Network.Carrier = event.Context.Network.Carrier
	e.Context.Network.Cellular = event.Context.Network.Cellular
	e.Context.Network.WiFi = event.Context.Network.WiFi
	e.Context.OS.Name = event.Context.OS.Name
	e.Context.OS.Version = event.Context.OS.Version
	e.Context.Page.Path = event.Context.Page.Path
	e.Context.Page.Referrer = event.Context.Page.Referrer
	e.Context.Page.Search = event.Context.Page.Search
	e.Context.Page.Title = event.Context.Page.Path
	e.Context.Page.URL = event.Context.Page.URL
	e.Context.Referrer.Id = event.Context.Referrer.Id
	e.Context.Referrer.Type = event.Context.Referrer.Type
	e.Context.Screen.Width = event.Context.Screen.Width
	e.Context.Screen.Height = event.Context.Screen.Height
	e.Context.Screen.Density = event.Context.Screen.Density
	e.Context.SessionId = event.Context.SessionId
	e.Context.SessionStart = event.Context.SessionStart
	e.Context.Timezone = event.Context.Timezone
	e.Context.UserAgent = event.Context.UserAgent
	e.Event = event.Event
	e.GroupId = groupId
	e.MessageId = event.MessageId
	e.Name = event.Name
	e.ReceivedAt = event.receivedAt
	e.SentAt = event.sentAt
	e.Timestamp = event.timestamp
	e.Type = *event.Type
	e.UserId = event.UserId
	return e
}

// collectedEventToMap returns a map built from the properties of the given
// collectedEvent.
func collectedEventToMap(event *collectedEvent) (map[string]any, error) {

	// Keep in sync with the schema in "apis/events/schema.go".

	// TODO(Gianluca): define datetime layout and parse/convert the values.
	mapEvent := map[string]any{
		"anonymousId": event.AnonymousId,
		"category":    event.Category,
		"context": map[string]any{
			"active": event.Context.Active,
			"app": map[string]any{
				"name":      event.Context.App.Name,
				"version":   event.Context.App.Version,
				"build":     event.Context.App.Build,
				"namespace": event.Context.App.Namespace,
			},
			"browser": map[string]any{
				"name":    event.Context.browser.Name,
				"other":   event.Context.browser.Other,
				"version": event.Context.browser.Version,
			},
			"campaign": map[string]any{
				"name":    event.Context.Campaign.Name,
				"source":  event.Context.Campaign.Source,
				"medium":  event.Context.Campaign.Medium,
				"term":    event.Context.Campaign.Term,
				"content": event.Context.Campaign.Content,
			},
			"device": map[string]any{
				"id":                event.Context.Device.Type,
				"advertisingId":     event.Context.Device.AdvertisingId,
				"adTrackingEnabled": event.Context.Device.AdTrackingEnabled,
				"manufacturer":      event.Context.Device.Manufacturer,
				"model":             event.Context.Device.Model,
				"name":              event.Context.Device.Name,
				"type":              event.Context.Device.Type,
				"token":             event.Context.Device.Token,
			},
			"ip": event.Context.IP,
			"library": map[string]any{
				"name":    event.Context.Library.Name,
				"version": event.Context.Library.Version,
			},
			"locale": event.Context.Locale,
			"location": map[string]any{
				"city":      event.Context.Location.City,
				"country":   event.Context.Location.Country,
				"latitude":  event.Context.Location.Latitude,
				"longitude": event.Context.Location.Longitude,
				"speed":     event.Context.Location.Speed,
			},
			"network": map[string]any{
				"bluetooth": event.Context.Network.Bluetooth,
				"carrier":   event.Context.Network.Carrier,
				"cellular":  event.Context.Network.Cellular,
				"wifi":      event.Context.Network.WiFi,
			},
			"os": map[string]any{
				"name":    event.Context.OS.Name,
				"version": event.Context.OS.Version,
			},
			"page": map[string]any{
				"path":     event.Context.Page.Path,
				"referrer": event.Context.Page.Referrer,
				"search":   event.Context.Page.Search,
				"title":    event.Context.Page.Title,
				"url":      event.Context.Page.URL,
			},
			"referrer": map[string]any{
				"id":   event.Context.Referrer.Id,
				"type": event.Context.Referrer.Type,
			},
			"screen": map[string]any{
				"width":   event.Context.Screen.Width,
				"height":  event.Context.Screen.Height,
				"density": event.Context.Screen.Density,
			},
			"sessionId":    event.Context.SessionId,
			"sessionStart": event.Context.SessionStart,
			"groupId":      event.Context.GroupId,
			"timezone":     event.Context.Timezone,
			"traits":       event.Context.Traits,
			"userAgent":    event.Context.UserAgent,
		},
		"event":      event.Event,
		"groupId":    event.GroupId,
		"messageId":  event.MessageId,
		"name":       event.Name,
		"properties": event.Properties,
		"receivedAt": event.receivedAt,
		"sendAt":     event.sentAt,
		"source":     event.source,
		"timestamp":  event.timestamp,
		"traits":     event.Traits,
		"type":       event.Type,
		"userId":     event.UserId,
		"version":    event.version,
	}

	return mapEvent, nil
}
