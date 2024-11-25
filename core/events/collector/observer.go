//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package collector

import (
	"math/rand/v2"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/core/filters"
	"github.com/meergo/meergo/core/postgres"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
)

const MaxEventListeners = 100 // maximum number of event listeners.

var (
	ErrEventListenerNotFound = errors.New("event listener does not exist")
	ErrTooManyListeners      = errors.New("too many listeners")
)

// Observer represents an event observer.
type Observer struct {
	db *postgres.DB
	sync.RWMutex
	listeners []*listener
}

// listener represents an event listener.
type listener struct {
	id         string
	filter     *state.Where
	sync.Mutex // for the events and discarded fields
	events     []json.Value
	times      []time.Time
	discarded  int
}

// newObserver returns a new observer.
func newObserver(db *postgres.DB) *Observer {
	return &Observer{db: db}
}

// AddListener adds a listener for events and returns its identifier. size
// specifies the maximum number of observed events to be returned by a
// subsequent call to the Events method. size must be in range [1, 1000]. If
// filter is non-nil, only events that satisfy the filter will be observed.
//
// AddListener does not validate its arguments, so it is the caller's
// responsibility to pass valid arguments.
//
// It returns the ErrTooManyListeners error if there are already too many
// listeners.
func (observer *Observer) AddListener(size int, filter *state.Where) (string, error) {
	id := uuid.New().String()
	listener := listener{
		id:     id,
		filter: filter,
		events: make([]json.Value, 0, size),
		times:  make([]time.Time, 0, size),
	}
	observer.Lock()
	defer observer.Unlock()
	if len(observer.listeners) == MaxEventListeners {
		return "", ErrTooManyListeners
	}
	observer.listeners = append(observer.listeners, &listener)
	return id, nil
}

// Events returns the observed events listen to by the specified listener and
// the number of discarded events. If the listener does not exist, it returns
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
	var discarded int
	if len(listener.events) > 0 {
		sort.Slice(listener.events, func(i, j int) bool { return listener.times[i].Before(listener.times[j]) })
		copy(observedEvents, listener.events)
		discarded = listener.discarded
		listener.events = listener.events[0:0]
		listener.times = listener.times[0:0]
		listener.discarded = 0
	}
	listener.Unlock()
	return observedEvents, discarded, nil
}

// RemoveListener removes the listener with identifier id.
func (observer *Observer) RemoveListener(id string) {
	observer.Lock()
	for i, listener := range observer.listeners {
		if listener.id == id {
			observer.listeners = slices.Delete(observer.listeners, i, i)
			break
		}
	}
	observer.Unlock()
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
	for _, listener := range observer.listeners {
		if !filters.Applies(listener.filter, event) {
			continue
		}
		listener.Lock()
		var p int
		if len(listener.events) == cap(listener.events) {
			listener.discarded++
			p = rand.IntN(len(listener.events) + listener.discarded)
			if p >= cap(listener.events) {
				listener.Unlock()
				continue
			}
		}
		if properties == nil {
			properties, _ = types.Marshal(event, events.Schema)
			receivedAt = event["receivedAt"].(time.Time)
		}
		if listener.discarded == 0 {
			listener.events = append(listener.events, properties)
			listener.times = append(listener.times, receivedAt)
		} else {
			listener.events[p] = properties
			listener.times[p] = receivedAt
		}
		listener.Unlock()
	}

}
