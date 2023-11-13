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
	"chichi/apis/transformers"
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
	inEvent     *connector.Event
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
func newProcessor(st *eventsState, eventLog *eventsLog, transformer transformers.Transformer, events <-chan *collectedEvent) (*Processor, error) {

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
						// Convert the collectedEvent to a map of properties.
						mapEvent := event.MapEvent()
						// Check if the filter applies.
						ok, err := mappings.FilterApplies(action.Filter, mapEvent)
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
							mapping, err := mappings.New(action.InSchema, action.OutSchema, action.Mapping, action.Transformation, action.ID, transformer, nil)
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
							inEvent:  event.ConnectorEvent(),
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
