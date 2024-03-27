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

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/apis/transformers"
	"github.com/open2b/chichi/types"
)

const pipeSize = 100

const (
	geoLite2Path = "GeoLite2-City.mmdb"
)

// eventDateLayout is the layout used for dates in events.
const eventDateLayout = "2006-01-02T15:04:05.999Z"

type processedEvent struct {
	*collectedEvent
	action       *state.Action
	destination  int
	eventType    string
	endpoint     int
	values       map[string]any
	valuesSchema types.Type
	inEvent      *chichi.Event
	err          error
}

// Processor processes events received from source streams and sent them to
// their data warehouses.
type Processor struct {
	sync.Mutex // for the streams field.
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
func newProcessor(st *eventsState, eventLog *eventsLog, transformer transformers.Function, events <-chan *collectedEvent) (*Processor, error) {

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
					source, ok := st.state.Connection(event.source)
					if !ok {
						continue
					}
					for _, action := range st.Actions(source) {
						// Convert the collectedEvent to a map of properties.
						eventAsMap := event.ToMap()
						// Check if the filter applies.
						ok, err := filterApplies(action.Filter, eventAsMap)
						if err != nil {
							eventLog.TransformationFailed(event.id, action.ID, err)
							continue
						}
						if !ok {
							continue
						}
						var values map[string]any
						if tr := action.Transformation; tr.Mapping != nil || tr.Function != nil {
							transformer, err := transformers.New(action.InSchema, action.OutSchema, tr, action.ID, transformer, nil)
							if err != nil {
								eventLog.TransformationFailed(event.id, action.ID, err)
								continue
							}
							values, err = transformer.Transform(ctx, eventAsMap)
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
							values:         values,
							valuesSchema:   action.OutSchema,
							// TODO(Gianluca): since the endpoints have been
							// removed from the action, we do not have
							// information about the endpoint. We should
							// review/refactor this.
							//
							// See the issue https://github.com/open2b/chichi/issues/194.
							//
							endpoint: 0,
							inEvent:  event.ToConnectorEvent(),
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
