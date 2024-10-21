//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package collector

import (
	"bytes"
	"context"
	stdjson "encoding/json"
	"log/slog"
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

	"github.com/google/uuid"
)

const MaxEventListeners = 100 // maximum number of event listeners.

// eventDateLayout is the layout used for dates in events.
const eventDateLayout = "2006-01-02T15:04:05.999Z"

var (
	ErrEventListenerNotFound = errors.New("event listener does not exist")
	ErrTooManyListeners      = errors.New("too many listeners")
)

// Observer represents an event observer.
type Observer struct {
	db *postgres.DB
	sync.RWMutex
	listeners struct {
		collected []*listener
		enriched  []*listener
	}
	statsMu sync.Mutex // for the stats field.
	stats   []statsEntry
}

// statsEntry is the element type of the observer.stats slice.
type statsEntry struct {
	source     int
	goodEvents int
	badEvents  int
}

// listener represents an event listener.
type listener struct {
	id            string
	sources       []int
	hasUserTraits bool
	onlyValid     bool
	filter        *state.Where
	sync.Mutex    // for the events and discarded fields
	events        []ObservedEvent
	times         []string
	discarded     int
}

// ObservedEvent represents an observed event.
type ObservedEvent struct {

	// Source, if not zero, it is the source mobile, server or website
	// connection for which the event was sent.
	Source int

	// Header is the message header. It is nil if a validation error occurred
	// processing the entire message.
	Header *events.Header

	// Data contains the data, encoded in JSON, of a single event in the message,
	// if header is not nil, or the data of the entire message, if Header is nil.
	Data []byte

	// Err, if not empty, is a validation error occurred processing the message.
	// It refers to a single event, if header is not nil, or to the entire message
	// if Header is nil.
	Err string
}

// newObserver returns a new observer.
func newObserver(db *postgres.DB) *Observer {
	observer := &Observer{db: db}
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for t := range ticker.C {
			err := observer.flushStats(t.UTC())
			if err != nil {
				slog.Error("cannot update event stats", "err", err)
			}
		}
	}()
	return observer
}

// AddCollectedListener adds a listener for collected events and returns its
// identifier. size specifies the maximum number of observed events to be
// returned by a subsequent call to the Events method and must be in range
// [1, 1000]. If sources is non-nil, only events originating from these sources
// will be observed. onlyValid determines whether only valid events should be
// observed.
//
// AddCollectedListener does not validate its arguments, so it is the caller's
// responsibility to pass valid arguments.
//
// It returns the ErrTooManyListeners error if there are already too many
// listeners.
func (observer *Observer) AddCollectedListener(size int, sources []int, onlyValid bool) (string, error) {
	id := uuid.New().String()
	listener := listener{
		id:        id,
		sources:   sources,
		onlyValid: onlyValid,
		events:    make([]ObservedEvent, 0, size),
		times:     make([]string, 0, size),
	}
	observer.Lock()
	defer observer.Unlock()
	if len(observer.listeners.collected) == MaxEventListeners {
		return "", ErrTooManyListeners
	}
	observer.listeners.collected = append(observer.listeners.collected, &listener)
	return id, nil
}

// AddEnrichedListener adds a listener for enriched events and returns its
// identifier. size specifies the maximum number of observed events to be
// returned by a subsequent call to the Events method. size must be in range
// [1, 1000]. If filter is non-nil, only events that satisfy the filter will be
// observed.
//
// AddEnrichedListener does not validate its arguments, so it is the caller's
// responsibility to pass valid arguments.
//
// It returns the ErrTooManyListeners error if there are already too many
// listeners.
func (observer *Observer) AddEnrichedListener(size int, sources []int, hasUserTraits bool, filter *state.Where) (string, error) {
	id := uuid.New().String()
	listener := listener{
		id:            id,
		sources:       sources,
		hasUserTraits: hasUserTraits,
		filter:        filter,
		events:        make([]ObservedEvent, 0, size),
		times:         make([]string, 0, size),
	}
	observer.Lock()
	defer observer.Unlock()
	if len(observer.listeners.enriched) == MaxEventListeners {
		return "", ErrTooManyListeners
	}
	observer.listeners.enriched = append(observer.listeners.enriched, &listener)
	return id, nil
}

// Events returns the observed events listen to by the specified listener and
// the number of discarded events. If the listener does not exist, it returns
// the ErrEventListenerNotFound error.
func (observer *Observer) Events(listenerID string) ([]ObservedEvent, int, error) {
	observer.RLock()
	defer observer.RUnlock()
	var l *listener
	for _, collected := range observer.listeners.collected {
		if collected.id == listenerID {
			l = collected
			break
		}
	}
	if l == nil {
		for _, enriched := range observer.listeners.enriched {
			if enriched.id == listenerID {
				l = enriched
				break
			}
		}
	}
	if l == nil {
		return nil, 0, ErrEventListenerNotFound
	}
	l.Lock()
	observedEvents := make([]ObservedEvent, len(l.events))
	var discarded int
	if len(l.events) > 0 {
		sort.Slice(l.events, func(i, j int) bool { return l.times[i] < l.times[j] })
		copy(observedEvents, l.events)
		discarded = l.discarded
		l.events = l.events[0:0]
		l.times = l.times[0:0]
		l.discarded = 0
	}
	l.Unlock()
	return observedEvents, discarded, nil
}

// RemoveListener removes the listener with identifier id.
func (observer *Observer) RemoveListener(id string) {
	var ok bool
	observer.Lock()
	observer.listeners.collected, ok = removeListener(observer.listeners.collected, id)
	if !ok {
		observer.listeners.enriched, _ = removeListener(observer.listeners.enriched, id)
	}
	observer.Unlock()
}

// addCollectedEvent adds a collected event to the observed events. source, if
// non-zero, indicates the origin (mobile, server, or website connection) for
// which the event was sent. If the event or message is invalid, err contains
// the error.
func (observer *Observer) addCollectedEvent(source int, event *collectedEvent, err error) {

	observer.RLock()
	defer observer.RUnlock()

	// Update statistics.
	var found bool
	observer.statsMu.Lock()
	for i, s := range observer.stats {
		if s.source == source {
			if err == nil {
				observer.stats[i].goodEvents++
			} else {
				observer.stats[i].badEvents++
			}
			found = true
			break
		}
	}
	if !found {
		entry := statsEntry{source: source}
		if err == nil {
			entry.goodEvents = 1
		} else {
			entry.badEvents = 1
		}
		observer.stats = append(observer.stats, entry)
	}
	observer.statsMu.Unlock()

	// Update listened events.
	if len(observer.listeners.collected) == 0 {
		return
	}
	var oe ObservedEvent
	var receivedAt string
	for _, listener := range observer.listeners.collected {
		if !slices.Contains(listener.sources, source) {
			continue
		}
		if listener.onlyValid && err != nil {
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
		if oe.Data == nil {
			var b bytes.Buffer
			enc := stdjson.NewEncoder(&b)
			enc.SetEscapeHTML(false)
			_ = enc.Encode(event)
			data := b.Bytes()
			var errStr string
			if err != nil {
				errStr = err.Error()
			}
			oe = ObservedEvent{
				Source: source,
				Header: event.header,
				Data:   data,
				Err:    errStr,
			}
		}
		receivedAt = event.header.ReceivedAt.Format(eventDateLayout)
		if listener.discarded == 0 {
			listener.events = append(listener.events, oe)
			listener.times = append(listener.times, receivedAt)
		} else {
			listener.events[p] = oe
			listener.times[p] = receivedAt
		}
		listener.Unlock()
	}

}

// addCollectedEvent adds an enriched event to the observed events. source, if
// non-zero, indicates the origin (mobile, server, or website connection) for
// which the event was sent.
func (observer *Observer) addEnrichedEvent(source int, event *events.Event) {

	observer.RLock()
	defer observer.RUnlock()

	if len(observer.listeners.enriched) == 0 {
		return
	}
	var properties map[string]any
	var oe ObservedEvent
	var receivedAt string
	for _, listener := range observer.listeners.enriched {
		if !slices.Contains(listener.sources, source) {
			continue
		}
		if listener.hasUserTraits {
			if t := *event.Type; t != "identify" && event.Context.Traits == nil {
				continue
			}
		}
		if listener.filter != nil {
			if properties == nil {
				properties = event.AsProperties()
			}
			if !filters.Applies(listener.filter, properties) {
				continue
			}
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
		if oe.Data == nil {
			if properties == nil {
				properties = event.AsProperties()
			}
			data, _ := json.MarshalBySchema(properties, events.Schema)
			oe = ObservedEvent{
				Source: source,
				Header: event.Header,
				Data:   data,
			}
		}
		receivedAt = event.Header.ReceivedAt.Format(eventDateLayout)
		if listener.discarded == 0 {
			listener.events = append(listener.events, oe)
			listener.times = append(listener.times, receivedAt)
		} else {
			listener.events[p] = oe
			listener.times[p] = receivedAt
		}
		listener.Unlock()
	}

}

func (observer *Observer) flushStats(t time.Time) error {

	observer.statsMu.Lock()
	if len(observer.stats) == 0 {
		observer.statsMu.Unlock()
		return nil
	}
	stats := make([]statsEntry, len(observer.stats))
	copy(stats, observer.stats)
	observer.stats = observer.stats[0:0]
	observer.statsMu.Unlock()

	ctx := context.Background()

	err := observer.db.Transaction(ctx, func(tx *postgres.Tx) error {
		query := "INSERT INTO connections_stats_events AS s (hour, source, good_events, bad_events)\n" +
			"VALUES ($1, NULLIF($2, 0), $3, $4)\n" +
			"\tON CONFLICT (hour, source) DO UPDATE SET good_events = s.good_events + EXCLUDED.good_events," +
			" bad_events = s.bad_events + EXCLUDED.bad_events"
		stmt, err := tx.Prepare(ctx, query)
		if err != nil {
			return err
		}
		hour := hoursFromEpoch(t)
		for _, s := range stats {
			_, err = stmt.Exec(ctx, hour, s.source, s.goodEvents, s.badEvents)
			if err != nil {
				_ = stmt.Close()
				return err
			}
		}
		return stmt.Close()
	})

	return err
}

// hasEnrichedListener checks if there is a listener for enriched events
// received from the specified source connection.
func (observer *Observer) hasEnrichedListener(source int) bool {
	observer.RLock()
	defer observer.RUnlock()
	for _, listener := range observer.listeners.enriched {
		if slices.Contains(listener.sources, source) {
			return true
		}
	}
	return false
}

// hoursFromEpoch returns the hours since January 1, 1970 UTC until time t.
// t must be a UTC time.
func hoursFromEpoch(t time.Time) int {
	epoch := int(t.Unix())
	return epoch / (60 * 60)
}

// removeListener removes from listeners the listener with the identifier id and
// returns the slice with the listener removed if it was found. The boolean
// return value indicates whether the listener has been removed.
func removeListener(listeners []*listener, id string) ([]*listener, bool) {
	i := slices.IndexFunc(listeners, func(l *listener) bool {
		return l.id == id
	})
	if i < 0 {
		return listeners, false
	}
	return slices.Delete(listeners, i, i), true
}
