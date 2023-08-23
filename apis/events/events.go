//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package events

import (
	"net/http"
	"sync"

	"chichi/apis/datastore"
	"chichi/apis/httpclient"
	"chichi/apis/postgres"
	"chichi/apis/state"
)

type Events struct {
	state       *eventsState
	log         *eventsLog
	observer    *Observer
	collector   *collector
	processor   *Processor
	stopSenders chan<- struct{}
	closed      bool
	closedMu    sync.Mutex
}

func New(db *postgres.DB, st *state.State, ds *datastore.Datastore, http *httpclient.HTTP) (*Events, error) {
	events := &Events{}
	events.state = newEventsState(db, st, http)
	events.log = newEventsLog(db)
	events.observer = newObserver(db)
	var err error
	events.collector, err = newCollector(events.state, ds, events.log, events.observer)
	if err != nil {
		return nil, err
	}
	events.processor, err = newProcessor(events.state, events.log, events.collector.Events())
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
	events.closedMu.Lock()
	defer events.closedMu.Unlock()
	if events.closed {
		panic("apis/events already closed")
	}
	close(events.stopSenders)
	events.processor.Close()
	events.log.Close()
	events.state.Close()
	events.closed = true
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
	events.closedMu.Lock()
	defer events.closedMu.Unlock()
	if events.closed {
		panic("apis/events is closed")
	}
}
