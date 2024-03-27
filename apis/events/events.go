//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package events

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sync/atomic"

	"github.com/open2b/chichi/apis/connectors"
	"github.com/open2b/chichi/apis/datastore"
	"github.com/open2b/chichi/apis/postgres"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/apis/transformers"

	"github.com/segmentio/ksuid"
	"golang.org/x/text/unicode/norm"
)

type Events struct {
	state       *eventsState
	log         *eventsLog
	observer    *Observer
	collector   *collector
	processor   *Processor
	stopSenders chan<- struct{}
	closed      atomic.Bool
}

func New(db *postgres.DB, st *state.State, ds *datastore.Datastore, transformer transformers.Function, connectors *connectors.Connectors) (*Events, error) {
	events := &Events{}
	events.state = newEventsState(st, connectors)
	events.log = newEventsLog(db)
	events.observer = newObserver(db)
	var err error
	events.collector, err = newCollector(events.state, ds, events.log, transformer, events.observer)
	if err != nil {
		return nil, err
	}
	events.processor, err = newProcessor(events.state, events.log, transformer, events.collector.Events())
	if err != nil {
		return nil, err
	}
	d := newDispatcher(events.log, events.processor.Events())
	events.stopSenders = startSenders(d.Events(), d.Done(), events.state)
	return events, nil
}

// Close closes the events.
// It panics if it has already been called.
func (events *Events) Close() {
	if events.closed.Swap(true) {
		panic("apis/events already closed")
	}
	close(events.stopSenders)
	events.processor.Close()
	events.log.Close()
}

// ParseObservedEvent parses an observed event, enriches and returns it.
// It returns an error if the event is not valid.
func (events *Events) ParseObservedEvent(event *ObservedEvent) (Event, error) {

	// Validate the arguments.
	if event.Header == nil {
		return nil, errors.New("header is nil")
	}
	ip, _, err := net.SplitHostPort(event.Header.RemoteAddr)
	if err != nil {
		return nil, errors.New("header.RemoteAddr is not valid")
	}
	if _, err := netip.ParseAddr(ip); err != nil {
		return nil, errors.New("header.RemoteAddr is not valid")
	}
	if _, err := url.Parse(event.Header.URL); err != nil {
		return nil, errors.New("header.URL is not valid")
	}
	if len(event.Data) > maxRequestSize {
		return nil, errors.New("event is too long")
	}

	// Decode the event.
	nr := norm.NFC.Reader(bytes.NewReader(event.Data))
	dec := json.NewDecoder(nr)
	dec.UseNumber()
	ev := &collectedEvent{}
	err = dec.Decode(ev)
	if err != nil {
		return nil, errors.New("cannot decode JSON")
	}

	// Validate the event.
	if ev.Type == nil {
		return nil, errors.New("missing event type")
	}
	err = validateEvent(*ev.Type, ev)
	if err != nil {
		return nil, err
	}
	ev.id = ksuid.New()
	ev.header = event.Header

	// Enrich the event.
	events.collector.enrichEvent(ev)

	return ev, nil
}

// ServeHTTP serves an event request.
// It panics if events has been closed.
func (events *Events) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	events.mustBeOpen()
	events.collector.ServeHTTP(w, r)
}

// Observer returns the event observer.
// It panics if events has been closed.
func (events *Events) Observer() *Observer {
	events.mustBeOpen()
	return events.observer
}

// mustBeOpen panics if events has been closed.
func (events *Events) mustBeOpen() {
	if events.closed.Load() {
		panic("apis/events is closed")
	}
}
