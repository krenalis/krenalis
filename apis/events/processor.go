//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package events

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"chichi/apis/state"
	"chichi/apis/transformations"
	"chichi/apis/types"
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

// emptyProperties represents an empty event properties.
var emptyProperties = []byte("{}")

type processedEvent struct {
	*collectedEvent
	action      *state.Action
	destination int
	actionType  int
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
						if !actionFilterApplies(action.Filter, event) {
							continue
						}
						mappedEvent, err := applyActionMapping(action, action.ActionType, event)
						if err != nil {
							eventLog.TransformationFailed(event.id, action.ID, err)
							continue
						}
						ev := &processedEvent{
							collectedEvent: event,
							action:         action,
							destination:    action.Connection().ID,
							actionType:     action.ActionType.ID,
							mappedEvent:    mappedEvent,
							endpoint:       action.Endpoint,
							inEvent:        eventToConnectorEvent(event),
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

// actionFilterApplies reports whether the action filter applies to the event.
func actionFilterApplies(filter state.ActionFilter, event *collectedEvent) bool {
	if len(filter.Conditions) == 0 {
		return true // no conditions, so this filter always applies.
	}
	for _, cond := range filter.Conditions {
		var value string
		switch cond.Property {
		case "AnonymousID":
			value = event.AnonymousID
		case "Event":
			value = event.Event
		case "UserID":
			value = event.UserID
		}
		var conditionApplies bool
		switch cond.Operator {
		case "is":
			conditionApplies = value == cond.Value
		case "is not":
			conditionApplies = value != cond.Value
		}
		if conditionApplies && filter.Logical == "any" {
			return true
		}
		if !conditionApplies && filter.Logical == "all" {
			return false
		}
	}
	if filter.Logical == "any" {
		return false // none of the conditions applied.
	}
	// All the conditions applied.
	return true
}

// eventToConnectorEvent returns the connector.Event corresponding to event.
func eventToConnectorEvent(event *collectedEvent) connector.Event {
	// Keep in sync with the schema in "apis/events/schema.go".
	e := connector.Event{}
	e.Source = event.source
	e.MessageID = event.MessageID
	e.AnonymousID = event.AnonymousID
	e.UserID = event.UserID
	e.Date = event.date
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
	e.Properties = event.properties
	return e
}

// applyActionMapping applies the action mapping (or transformation) to the
// event, returning the mapped event. actionType holds information about the
// action type relative to the action.
func applyActionMapping(action *state.Action, actionType *state.ActionType, event *collectedEvent) (map[string]any, error) {

	// Convert the input event to a map.
	// Keep in sync with the schema in "apis/events/schema.go".

	var properties map[string]any
	err := json.Unmarshal([]byte(event.properties), &properties)
	if err != nil {
		return nil, err
	}
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
		"properties": properties,
	}

	var mappedEvent map[string]any

	// Map using properties mapping.
	if action.Mapping != nil {

		mappedEvent = map[string]any{}
		for out, in := range action.Mapping {
			inputPropPath := strings.Split(in, ".")
			value, ok := readPropertyFrom(mapEvent, inputPropPath)
			if !ok {
				continue
			}
			// TODO(Gianluca): handle conversions of values here, when the type
			// checking rules will be defined.
			outputPropPath := strings.Split(out, ".")
			writePropertyTo(mappedEvent, outputPropPath, value)
		}

	} else if action.Transformation != nil {

		// Map using the transformation function.
		t := action.Transformation

		// Prepare the event for the transformation.
		inPropsNames := t.In.PropertiesNames()
		inEvent := make(map[string]any, len(inPropsNames))
		for _, name := range inPropsNames {
			value, ok := mapEvent[name]
			if !ok {
				continue
			}
			inEvent[name] = value
		}

		// Validate the input properties according to the input schema.
		err = validateProps(inEvent, t.In)
		if err != nil {
			return nil, fmt.Errorf("input schema validation failed: %s", err)
		}

		// Run the Python transformation function.
		pool := transformations.NewPool()
		ctx := context.Background()
		mappedEvent, err = pool.Run2(ctx, t.PythonSource, inEvent)
		if err != nil {
			return nil, fmt.Errorf("error while calling the transformation function of the action: %s", err)
		}

		// Validate the properties returned by Python according to the output schema.
		err = validateProps(mappedEvent, t.Out)
		if err != nil {
			return nil, fmt.Errorf("output schema validation failed: %s", err)
		}

	}

	// If the action type schema is defined, then validate the mapped properties
	// with it.
	if actionType.Schema.Valid() {
		err = validateProps(mappedEvent, actionType.Schema)
		if err != nil {
			return nil, fmt.Errorf("mapped properties validation failed: %s", err)
		}
	}

	return mappedEvent, nil

}

// validateProps validate the given properties using schema, returning error if
// the validation fails.
func validateProps(props map[string]any, schema types.Type) error {
	data, err := json.Marshal(props)
	if err != nil {
		return err
	}
	_, err = types.Decode(bytes.NewReader(data), schema)
	return err
}

// readPropertyFrom reads the property with the given path from m, returning its
// value (if found, otherwise nil) and a boolean indicating if the property path
// corresponds to a value in m or not.
func readPropertyFrom(m map[string]any, propPath []string) (any, bool) {
	name := propPath[0]
	v, ok := m[name]
	if !ok {
		return nil, false
	}
	if len(propPath) == 1 {
		return v, ok
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	return readPropertyFrom(obj, propPath[1:])
}

// writePropertyTo writes the property value v into m at the given property
// path.
// m cannot be nil.
func writePropertyTo(m map[string]any, propPath []string, v any) {
	name := propPath[0]
	if len(propPath) == 1 {
		m[name] = v
		return
	}
	_, ok := m[name]
	if !ok {
		m[name] = map[string]any{}
	}
	obj, ok := m[name].(map[string]any)
	if !ok {
		return
	}
	writePropertyTo(obj, propPath[1:], v)
}
