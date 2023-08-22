//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package events

import (
	"net/http"

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
func (events *Events) Close() {
	close(events.stopSenders)
	events.processor.Close()
	events.log.Close()
	events.state.Close()
}

// ServeHTTP serves an event request.
func (events *Events) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	events.collector.ServeHTTP(w, r)
}

// Observer returns the event observer.
func (events *Events) Observer() *Observer {
	return events.observer
}
