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
	"log"
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

// statsKey is the key in the observer.stats slice.
type statsKey struct {
	source int
	server int
	stream int
}

// statsKey is the element type of the observer.stats slice.
type statsEntry struct {
	key        statsKey
	goodEvents int
	badEvents  int
}

// listener represents an event listener.
type listener struct {
	id         string
	source     int
	server     int
	stream     int
	sync.Mutex // for the events and discarded fields
	events     []json.RawMessage
	times      []string
	discarded  int
}

// ProcessedEvent represents a processed event.
type ProcessedEvent struct {

	// Source, if not zero, it is the mobile or website collector on which
	// the event was generated.
	Source int

	// Server, if not zero, is the server collector that sent the message.
	Server int

	// Stream is the stream from which the event was received.
	Stream int

	// Header is the message header. It is nil if a validation error occurred
	// processing the entire message.
	Header *collectedHeader

	// Data contains the data, encoded in JSON, of a single event in the message,
	// if header is not nil, or the data of the entire message, if header is nil.
	Data []byte

	// Err, if not empty, is a validation error occurred processing the message.
	// It refers to a single event, if header is not nil, or to the entire message
	// if header is nil.
	Err string
}

// newObserver returns a new observer.
func newObserver(db *postgres.DB) *Observer {
	observer := &Observer{db: db}
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case t := <-ticker.C:
				err := observer.flushStats(t.UTC())
				if err != nil {
					log.Fatalf("[error] cannot update event stats: %s", err)
				}
			}
		}
	}()
	return observer
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
		query := "INSERT INTO connections_stats_events AS s (hour, source, server, stream, good_events, bad_events)\n" +
			"VALUES ($1, NULLIF($2, 0), NULLIF($3, 0), NULLIF($4, 0), $5, $6)\n" +
			"\tON CONFLICT (hour, source, server, stream) DO UPDATE SET good_events = s.good_events + EXCLUDED.good_events," +
			" bad_events = s.bad_events + EXCLUDED.bad_events"
		stmt, err := tx.Prepare(ctx, query)
		if err != nil {
			return err
		}
		hour := hoursFromEpoc(t)
		for _, s := range stats {
			_, err = stmt.Exec(ctx, hour, s.key.source, s.key.server, s.key.stream, s.goodEvents, s.badEvents)
			if err != nil {
				_ = stmt.Close()
				return err
			}
		}
		return stmt.Close()
	})

	return err
}

// AddEvent adds an event to the observed events. source, if not-zero is the
// connection, mobile or website, where the event occurred. If the event was
// sent by a server, server is its connection identifier, otherwise server is
// zero. If a message or event is invalid, err contains the error.
func (observer *Observer) AddEvent(source, server, stream int, event *collectedEvent, err error) {

	observer.RLock()
	defer observer.RUnlock()

	// Update statistics.
	var found bool
	key := statsKey{source, server, stream}
	observer.statsMu.Lock()
	for i, s := range observer.stats {
		if s.key == key {
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
		entry := statsEntry{key: key}
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
	var rawEvent json.RawMessage
	var receivedAt string
	for _, listener := range observer.listeners {
		if listener.source > 0 && listener.source != source {
			continue
		}
		if listener.server > 0 && listener.server != server {
			continue
		}
		if listener.stream > 0 && listener.stream != stream {
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
		if rawEvent == nil {
			var b bytes.Buffer
			enc := json.NewEncoder(&b)
			enc.SetEscapeHTML(false)
			_ = enc.Encode(event)
			data := b.Bytes()
			var errStr string
			if err != nil {
				errStr = err.Error()
			}
			b.Reset()
			_ = enc.Encode(ProcessedEvent{
				Source: source,
				Server: server,
				Stream: stream,
				Header: event.header,
				Data:   data,
				Err:    errStr,
			})
			rawEvent = b.Bytes()
		}
		receivedAt = event.header.ReceivedAt.Format(eventDateLayout)
		if listener.discarded == 0 {
			listener.events = append(listener.events, rawEvent)
			listener.times = append(listener.times, receivedAt)
		} else {
			listener.events[p] = rawEvent
			listener.times[p] = receivedAt
		}
		listener.Unlock()
	}

}

// Events returns the events listen to by the specified listener and the number
// of discarded events. If the listener does not exist, it returns the
// ErrEventListenerNotFound error.
func (observer *Observer) Events(listener string) ([]json.RawMessage, int, error) {
	observer.RLock()
	for _, l := range observer.listeners {
		if l.id != listener {
			continue
		}
		observer.RUnlock()
		l.Lock()
		var events = make([]json.RawMessage, len(l.events))
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

// AddListener adds a processed event listener. It returns the
// ErrTooManyListeners error if there are already too many listeners.
func (observer *Observer) AddListener(size, source, server, stream int) (string, error) {
	id := uuid.New().String()
	listener := listener{
		id:     id,
		source: source,
		server: server,
		stream: stream,
		events: make([]json.RawMessage, 0, size),
		times:  make([]string, 0, size),
	}
	observer.Lock()
	defer observer.Unlock()
	if len(observer.listeners) == MaxEventListeners {
		return "", ErrTooManyListeners
	}
	observer.listeners = append(observer.listeners, &listener)
	return id, nil
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
	return
}

// hoursFromEpoc returns the hours since January 1, 1970 UTC until time t.
// t must be a UTC time.
func hoursFromEpoc(t time.Time) int {
	epoc := int(t.Unix())
	return epoc / (60 * 60)
}
