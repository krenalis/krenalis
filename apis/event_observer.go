//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"bytes"
	"encoding/json"
	"log"
	"math/rand"
	"sort"
	"sync"
	"time"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/state"

	"github.com/google/uuid"
)

const (
	maxEventListeners   = 100  // maximum number of event listeners.
	maxEventsListenedTo = 1000 // maximum number of processed events listened to.
)

var (
	ServerNotExist   errors.Code = "ServerNotExist"
	SourceNotExist   errors.Code = "SourceNotExist"
	TooManyListeners errors.Code = "TooManyListeners"
)

type EventListeners struct {
	db        *postgres.DB
	processor *eventProcessor
	workspace *state.Workspace
}

// Add adds an event listener that listen to processed events
//
//   - occurred on the mobile or website connection source, if source is not
//     zero,
//   - sent by the server connection server, if server is not zero,
//   - received from the stream connection stream, if stream is not zero,
//
// and returns its identifier. size is the maximum number of events to return
// for each call to the Events method, it must be in [1,1000].
//
// If the source, server, or stream does not exist, it returns an
// errors.UnprocessableError error with code SourceNotExist, ServerNotExist,
// and StreamNotExist respectively.
// If there are already too many listeners, it returns an
// errors.UnprocessableError error with code TooManyListeners.
func (this *EventListeners) Add(size, source, server, stream int) (string, error) {
	if size < 1 || size > maxEventsListenedTo {
		return "", errors.BadRequest("size %d is not valid", size)
	}
	if source < 0 || source > maxInt32 {
		return "", errors.BadRequest("source identifier %d is not valid", source)
	}
	if server < 0 || server > maxInt32 {
		return "", errors.BadRequest("server identifier %d is not valid", server)
	}
	if stream < 0 || stream > maxInt32 {
		return "", errors.BadRequest("stream identifier %d is not valid", stream)
	}
	if source > 0 || server > 0 || stream > 0 {
		var sourceExist, serverExist, streamExist bool
		err := this.db.QueryScan("SELECT id, type , role FROM connections\n"+
			"WHERE id IN ($1, $2, $3) AND workspace = $4", source, server, stream, this.workspace.ID,
			func(rows *postgres.Rows) error {
				var id int
				var typ ConnectorType
				var role ConnectionRole
				for rows.Next() {
					if err := rows.Scan(&id, &typ, &role); err != nil {
						return err
					}
					switch id {
					case source:
						if typ != MobileType && typ != WebsiteType {
							return errors.BadRequest("connection %d is not a mobile or website", source)
						}
						sourceExist = true
					case server:
						if typ != ServerType {
							return errors.BadRequest("connection %d is not a server", server)
						}
						serverExist = true
					case stream:
						if typ != EventStreamType {
							return errors.BadRequest("connection %d is not a stream", stream)
						}
						streamExist = true
					}
					if role != SourceRole {
						return errors.BadRequest("connection %d is not a source", id)
					}
				}
				return nil
			})
		if err != nil {
			return "", err
		}
		if source > 0 && !sourceExist {
			return "", errors.Unprocessable(SourceNotExist, "source %d does not exist", source)
		}
		if server > 0 && !serverExist {
			return "", errors.Unprocessable(ServerNotExist, "server %d does not exist", server)
		}
		if stream > 0 && !streamExist {
			return "", errors.Unprocessable(StreamNotExist, "stream %d does not exist", stream)
		}
	}
	return this.processor.observer.AddListener(size, source, server, stream)
}

// Events returns the events listen to and the number of discarded events by
// the listener with identifier id.
//
// If the listener does not exist, it returns an errors.NotFoundError error.
func (this *EventListeners) Events(id string) ([]json.RawMessage, int, error) {
	return this.processor.observer.Events(id)
}

// Remove removes the event listener with identifier id. If the listener does
// not exist, it does nothing.
func (this *EventListeners) Remove(id string) {
	this.processor.observer.RemoveListener(id)
}

// eventObserver represents the event observer.
type eventObserver struct {
	db *postgres.DB
	sync.RWMutex
	listeners []*eventListener
	statsMu   sync.Mutex // for the stats field.
	stats     []statsEntry
}

// statsKey is the key in the eventObserver.stats slice.
type statsKey struct {
	source int
	server int
	stream int
}

// statsKey is the element type of the eventObserver.stats slice.
type statsEntry struct {
	key        statsKey
	goodEvents int
	badEvents  int
}

// eventListener represents an event listener.
type eventListener struct {
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
	Header *MessageHeader

	// Data contains the data, encoded in JSON, of a single event in the message,
	// if header is not nil, or the data of the entire message, if header is nil.
	Data []byte

	// Err, if not empty, is a validation error occurred processing the message.
	// It refers to a single event, if header is not nil, or to the entire message
	// if header is nil.
	Err string
}

// newEventObserver returns a new event observer.
func newEventObserver(db *postgres.DB) *eventObserver {
	observer := &eventObserver{db: db}
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

func (observer *eventObserver) flushStats(t time.Time) error {

	observer.statsMu.Lock()
	if len(observer.stats) == 0 {
		observer.statsMu.Unlock()
		return nil
	}
	stats := make([]statsEntry, len(observer.stats))
	copy(stats, observer.stats)
	observer.stats = observer.stats[0:0]
	observer.statsMu.Unlock()

	query := "INSERT INTO connections_stats_events AS s (hour, source, server, stream, good_events, bad_events)\n" +
		"VALUES ($1, $2, NULLIF($3, 0), NULLIF($4, 0), $5, $6)\n" +
		"\tON CONFLICT (hour, source, server, stream) DO UPDATE SET good_events = s.good_events + EXCLUDED.good_events," +
		" bad_events = s.bad_events + EXCLUDED.bad_events"
	stmt, err := observer.db.Prepare(query)
	if err != nil {
		return err
	}
	hour := hoursFromEpoc(t)
	for _, s := range stats {
		_, err = stmt.Exec(hour, s.key.source, s.key.server, s.key.stream, s.goodEvents, s.badEvents)
		if err != nil {
			_ = stmt.Close()
			return err
		}
	}
	err = stmt.Close()

	return err
}

// AddEvent adds a processed message or event to the observed events. source,
// if not-zero is the connection, mobile or website, where the event occurred.
// If the event was sent by a server, server is its connection identifier,
// otherwise server is zero. header contains the HTTP message headers and can
// be nil if an error occurred processing the message headers. data is the
// entire message data if headers is nil, otherwise is the event data. If a
// message or event is invalid, err contains the error.
func (observer *eventObserver) AddEvent(source, server, stream int, header *MessageHeader, data []byte, err error) {

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
	var event json.RawMessage
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
		if event == nil {
			if header == nil {
				receivedAt = time.Now().UTC().Format(eventDateLayout)
			} else {
				receivedAt = header.ReceivedAt
			}
			var b bytes.Buffer
			enc := json.NewEncoder(&b)
			enc.SetEscapeHTML(false)
			var errStr string
			if err != nil {
				errStr = err.Error()
			}
			_ = enc.Encode(ProcessedEvent{
				Source: source,
				Server: server,
				Stream: stream,
				Header: header,
				Data:   data,
				Err:    errStr,
			})
			event = b.Bytes()
		}
		if listener.discarded == 0 {
			listener.events = append(listener.events, event)
			listener.times = append(listener.times, receivedAt)
		} else {
			listener.events[p] = event
			listener.times[p] = receivedAt
		}
		listener.Unlock()
	}

}

// Events returns the events listen to by the specified listener and the number
// of discarded events. If the listener does not exist, it returns the
// ErrEventListenerNotFound error.
func (observer *eventObserver) Events(listener string) ([]json.RawMessage, int, error) {
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
	return nil, 0, errors.NotFound("event listener %q does not exist", listener)
}

// AddListener adds a processed event listener.
// See the (*EventListeners).Add documentation for details.
func (observer *eventObserver) AddListener(size, source, server, stream int) (string, error) {
	id := uuid.New().String()
	listener := eventListener{
		id:     id,
		source: source,
		server: server,
		stream: stream,
		events: make([]json.RawMessage, 0, size),
		times:  make([]string, 0, size),
	}
	observer.Lock()
	defer observer.Unlock()
	if len(observer.listeners) == maxEventListeners {
		return "", errors.Unprocessable(TooManyListeners, "there are already %d listeners", maxEventListeners)
	}
	observer.listeners = append(observer.listeners, &listener)
	return id, nil
}

// RemoveListener removes the listener with identifier id.
func (observer *eventObserver) RemoveListener(id string) {
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
