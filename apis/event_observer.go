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
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"chichi/pkg/open2b/sql"
	"github.com/google/uuid"
)

const (
	maxEventListeners   = 100  // maximum number of event listeners.
	maxEventsListenedTo = 1000 // maximum number of processed events listened to.
)

var (
	ErrEventListenerNotFound = errors.New("event listener does not exist")
	ErrTooManyEventListeners = errors.New("too many event listeners")
)

type EventListeners struct {
	*WorkspaceAPI
}

// Add adds an event listener that listen to processed events
//
//   - occurred on the mobile or website connection source, if source is not
//     zero,
//   - sent by the server connection server, if server is not zero,
//   - received from the stream connection stream, if stream is not zero,
//
// and returns its identifier.
//
// Returns a ConnectionNotFoundError error with type WebsiteType if the source
// does not exist, with type ServerType if the server does not exist and with
// type EventStreamType if the stream does not exist. if there are too many
// event listeners, it returns a ErrTooManyEventListeners error.
func (this *EventListeners) Add(source, server, stream int) (string, error) {
	if source < 0 || source > maxInt32 {
		return "", errors.New("invalid source identifier")
	}
	if server < 0 || server > maxInt32 {
		return "", errors.New("invalid server identifier")
	}
	if stream < 0 || stream > maxInt32 {
		return "", errors.New("invalid stream identifier")
	}
	if source > 0 || server > 0 || stream > 0 {
		var sourceExist, serverExist, streamExist bool
		err := this.myDB.QueryScan("SELECT `id`, CAST(`s`.`type` AS UNSIGNED), CAST(`s`.`role` AS UNSIGNED), \n"+
			"FROM `connections`\n"+
			"WHERE `id` IN (?, ?, ?) AND `workspace` = ?", source, server, stream, this.workspace,
			func(rows *sql.Rows) error {
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
							return fmt.Errorf("connector %d is not a mobile or website connector", source)
						}
						sourceExist = true
					case server:
						if typ != ServerType {
							return fmt.Errorf("connector %d is not a server connector", server)
						}
						serverExist = true
					case stream:
						if typ != EventStreamType {
							return fmt.Errorf("connector %d is not an event stream connector", server)
						}
						streamExist = true
					}
					if role != SourceRole {
						return fmt.Errorf("connector %d is not a source connector", id)
					}
				}
				return nil
			})
		if err != nil {
			return "", err
		}
		if source > 0 && !sourceExist {
			return "", ConnectionNotFoundError{WebsiteType}
		}
		if server > 0 && !serverExist {
			return "", ConnectionNotFoundError{ServerType}
		}
		if stream > 0 && !streamExist {
			return "", ConnectionNotFoundError{EventStreamType}
		}
	}
	return this.api.apis.eventProcessor.observer.AddListener(source, server, stream)
}

// Events returns the events listen to by the specified listener and the number
// of discarded events. If the listener does not exist, it returns the
// ErrEventListenerNotFound error.
func (this *EventListeners) Events(id string) ([]json.RawMessage, int, error) {
	return this.api.apis.eventProcessor.observer.Events(id)
}

// Remove removes the event listener with identifier id. If the listener does
// not exist, it does nothing.
func (this *EventListeners) Remove(id string) {
	this.api.apis.eventProcessor.observer.RemoveListener(id)
}

// eventObserver represents the event observer.
type eventObserver struct {
	sync.RWMutex
	listeners []*eventListener
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
func newEventObserver() *eventObserver {
	return &eventObserver{}
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
		if len(listener.events) == maxEventsListenedTo {
			listener.discarded++
			p = rand.Intn(len(listener.events) + listener.discarded)
			if p >= maxEventsListenedTo {
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
		if events == nil {
			return []json.RawMessage{}, 0, nil
		}
		sort.Slice(events, func(i, j int) bool { return times[i] < times[j] })
		return events, discarded, nil
	}
	observer.RUnlock()
	return nil, 0, ErrEventListenerNotFound
}

// AddListener adds a processed event listener.
// See the (*EventListeners).Add documentation for details.
func (observer *eventObserver) AddListener(source, server, stream int) (string, error) {
	id := uuid.New().String()
	listener := eventListener{
		id:     id,
		source: source,
		server: server,
		stream: stream,
		events: make([]json.RawMessage, 0, maxEventsListenedTo),
		times:  make([]string, 0, maxEventsListenedTo),
	}
	observer.Lock()
	defer observer.Unlock()
	if len(observer.listeners) == maxEventListeners {
		return "", ErrTooManyEventListeners
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
