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
	"log/slog"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"chichi/apis/errors"
	"chichi/apis/postgres"

	"github.com/google/uuid"
)

const MaxEventListeners = 100 // maximum number of event listeners.

var (
	ErrEventListenerNotFound = errors.New("event listener does not exist")
	ErrTooManyListeners      = errors.New("too many listeners")
)

// Observer represents the event observer.
type Observer struct {
	db *postgres.DB
	sync.RWMutex
	listeners []*listener
	statsMu   sync.Mutex // for the stats field.
	stats     []statsEntry
}

// statsEntry is the element type of the observer.stats slice.
type statsEntry struct {
	source     int
	goodEvents int
	badEvents  int
}

// listener represents an event listener.
type listener struct {
	id         string
	source     int
	onlyValid  bool
	sync.Mutex // for the events and discarded fields
	events     []ObservedEvent
	times      []string
	discarded  int
}

// ObservedEvent represents an observed event.
type ObservedEvent struct {

	// Source, if not zero, it is the source mobile, server or website
	// connection for which the event was sent.
	Source int

	// Header is the message header. It is nil if a validation error occurred
	// processing the entire message.
	Header *EventHeader

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

// addEvent adds an event to the observed events. source, if not-zero is the
// source mobile, server or website connection for which the event was sent. If
// a message or event is invalid, err contains the error.
func (observer *Observer) addEvent(source int, event *collectedEvent, err error) {

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
	if len(observer.listeners) == 0 {
		return
	}
	var ev ObservedEvent
	var receivedAt string
	for _, listener := range observer.listeners {
		if listener.source > 0 && listener.source != source {
			continue
		}
		if listener.onlyValid && err != nil {
			continue
		}
		listener.Lock()
		var p int
		if len(listener.events) == cap(listener.events) {
			listener.discarded++
			p = rand.Intn(len(listener.events) + listener.discarded)
			if p >= cap(listener.events) {
				listener.Unlock()
				continue
			}
		}
		if ev.Data == nil {
			var b bytes.Buffer
			enc := json.NewEncoder(&b)
			enc.SetEscapeHTML(false)
			_ = enc.Encode(event)
			data := b.Bytes()
			var errStr string
			if err != nil {
				errStr = err.Error()
			}
			ev = ObservedEvent{
				Source: source,
				Header: event.header,
				Data:   data,
				Err:    errStr,
			}
		}
		receivedAt = event.header.ReceivedAt.Format(eventDateLayout)
		if listener.discarded == 0 {
			listener.events = append(listener.events, ev)
			listener.times = append(listener.times, receivedAt)
		} else {
			listener.events[p] = ev
			listener.times[p] = receivedAt
		}
		listener.Unlock()
	}

}

// AddListener adds an event listener and returns its identifier. size specifies
// the maximum number of observed events to be returned by a subsequent call to
// the Events method. If source is non-zero, only events originating from this
// source will be observed. onlyValid determines whether only valid events
// should be observed.
//
// It returns the ErrTooManyListeners error if there are already too many
// listeners.
func (observer *Observer) AddListener(size, source int, onlyValid bool) (string, error) {
	if size < 1 {
		return "", fmt.Errorf("size %d is not valid", size)
	}
	if source < 0 || source > math.MaxInt32 {
		return "", fmt.Errorf("source %d is not valid", source)
	}
	id := uuid.New().String()
	listener := listener{
		id:        id,
		source:    source,
		onlyValid: onlyValid,
		events:    make([]ObservedEvent, 0, size),
		times:     make([]string, 0, size),
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
func (observer *Observer) Events(listener string) ([]ObservedEvent, int, error) {
	observer.RLock()
	for _, l := range observer.listeners {
		if l.id != listener {
			continue
		}
		observer.RUnlock()
		l.Lock()
		var events = make([]ObservedEvent, len(l.events))
		var discarded int
		if len(l.events) > 0 {
			sort.Slice(l.events, func(i, j int) bool { return l.times[i] < l.times[j] })
			copy(events, l.events)
			discarded = l.discarded
			l.events = l.events[0:0]
			l.times = l.times[0:0]
			l.discarded = 0
		}
		l.Unlock()
		return events, discarded, nil
	}
	observer.RUnlock()
	return nil, 0, ErrEventListenerNotFound
}

// RemoveListener removes the listener with identifier id.
func (observer *Observer) RemoveListener(id string) {
	observer.Lock()
	p := -1
	for i, listener := range observer.listeners {
		if listener.id == id {
			p = i
		}
	}
	if p != -1 {
		observer.listeners = append(observer.listeners[:p], observer.listeners[p+1:]...)
	}
	observer.Unlock()
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

// hoursFromEpoch returns the hours since January 1, 1970 UTC until time t.
// t must be a UTC time.
func hoursFromEpoch(t time.Time) int {
	epoch := int(t.Unix())
	return epoch / (60 * 60)
}
