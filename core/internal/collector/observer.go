// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package collector

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/krenalis/krenalis/core/internal/events"
	"github.com/krenalis/krenalis/core/internal/filters"
	"github.com/krenalis/krenalis/core/internal/schemas"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/google/uuid"
)

const MaxEventListeners = 100 // maximum number of event listeners.

var (
	ErrEventListenerNotFound = errors.New("event listener does not exist")
	ErrTooManyListeners      = fmt.Errorf("there are already %d listeners", MaxEventListeners)
)

// Observers maps each workspace to its observer.
type Observers struct {
	m sync.Map
}

// Delete deletes the observer for the provided workspace.
func (o *Observers) Delete(workspace int) {
	o.m.LoadAndDelete(workspace)
}

// Load returns the observer for the given workspace identifier, or nil if none
// exists.
// The ok result indicates whether the workspace was found.
func (o *Observers) Load(workspace int) (observer *Observer, ok bool) {
	v, ok := o.m.Load(workspace)
	if !ok {
		return nil, false
	}
	return v.(*Observer), true
}

// Store sets the observer for a workspace.
func (o *Observers) Store(workspace int, observer *Observer) {
	o.m.Store(workspace, observer)
}

// Observer represents an event observer.
type Observer struct {
	sync.RWMutex
	listeners []*listener
}

// listener represents an event listener.
type listener struct {
	id          string
	connections []int
	filter      *state.Where
	sync.Mutex  // for the events and omitted fields
	events      []json.Value
	times       []time.Time
	omitted     int
}

// newObserver returns a new observer.
func newObserver() *Observer {
	return &Observer{}
}

// CreateListener creates a listener for events and returns its identifier.
//
// If connections is not nil, only events received from these connections will
// be returned.
//
// size specifies the maximum number of observed events to be returned by a
// subsequent call to the Events method. size must be in range [1, 1000]. If
// filter is non-nil, only events that satisfy the filter will be observed.
//
// CreateListener does not validate its arguments, so it is the caller's
// responsibility to pass valid arguments.
//
// It returns the ErrTooManyListeners error if there are already too many
// listeners.
func (observer *Observer) CreateListener(connections []int, size int, filter *state.Where) (string, error) {
	id := uuid.New().String()
	listener := listener{
		id:          id,
		connections: connections,
		filter:      filter,
		events:      make([]json.Value, 0, size),
		times:       make([]time.Time, 0, size),
	}
	observer.Lock()
	defer observer.Unlock()
	if len(observer.listeners) == MaxEventListeners {
		return "", ErrTooManyListeners
	}
	observer.listeners = append(observer.listeners, &listener)
	return id, nil
}

// DeleteListener deletes the listener with identifier id.
func (observer *Observer) DeleteListener(id string) {
	observer.Lock()
	for i, listener := range observer.listeners {
		if listener.id == id {
			observer.listeners = slices.Delete(observer.listeners, i, i+1)
			break
		}
	}
	observer.Unlock()
}

// Events returns the observed events listen to by the specified listener and
// the number of omitted events. If the listener does not exist, it returns
// the ErrEventListenerNotFound error.
func (observer *Observer) Events(listenerID string) ([]json.Value, int, error) {
	observer.RLock()
	defer observer.RUnlock()
	var listener *listener
	for _, l := range observer.listeners {
		if l.id == listenerID {
			listener = l
			break
		}
	}
	if listener == nil {
		return nil, 0, ErrEventListenerNotFound
	}
	listener.Lock()
	observedEvents := make([]json.Value, len(listener.events))
	var omitted int
	if len(listener.events) > 0 {
		sort.Slice(listener.events, func(i, j int) bool { return listener.times[i].Before(listener.times[j]) })
		copy(observedEvents, listener.events)
		omitted = listener.omitted
		listener.events = listener.events[0:0]
		listener.times = listener.times[0:0]
		listener.omitted = 0
	}
	listener.Unlock()
	return observedEvents, omitted, nil
}

// addEvent adds an event to the observed events.
func (observer *Observer) addEvent(event events.Event) {

	observer.RLock()
	defer observer.RUnlock()

	// Update the events.
	if len(observer.listeners) == 0 {
		return
	}
	var properties json.Value
	var receivedAt time.Time
	connectionId := event["connectionId"].(int)
	for _, listener := range observer.listeners {
		if listener.connections != nil && !slices.Contains(listener.connections, connectionId) {
			continue
		}
		if !filters.Applies(listener.filter, event) {
			continue
		}
		listener.Lock()
		var p int
		if len(listener.events) == cap(listener.events) {
			listener.omitted++
			p = rand.IntN(len(listener.events) + listener.omitted)
			if p >= cap(listener.events) {
				listener.Unlock()
				continue
			}
		}
		if properties == nil {
			properties, _ = types.Marshal(event, schemas.Event)
			receivedAt = event["receivedAt"].(time.Time)
		}
		if listener.omitted == 0 {
			listener.events = append(listener.events, properties)
			listener.times = append(listener.times, receivedAt)
		} else {
			listener.events[p] = properties
			listener.times[p] = receivedAt
		}
		listener.Unlock()
	}

}
