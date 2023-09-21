//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package events

import (
	"net/http"
	"sync/atomic"

	"chichi/apis/datastore"
	"chichi/apis/httpclient"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/apis/transformers"
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

func New(db *postgres.DB, st *state.State, ds *datastore.Datastore, transformer transformers.Transformer, http *httpclient.HTTP) (*Events, error) {
	events := &Events{}
	events.state = newEventsState(db, st, http)
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
