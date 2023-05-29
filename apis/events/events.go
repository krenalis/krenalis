//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package events

import (
	"context"
	"net/http"

	"chichi/apis/httpclient"
	"chichi/apis/postgres"
	"chichi/apis/state"
)

type Events struct {
	collector *collector
	observer  *Observer
}

func New(ctx context.Context, db *postgres.DB, st *state.State, http *httpclient.HTTP) (*Events, error) {
	state := newEventsState(ctx, st, http)
	eventLog := newEventsLog(ctx, db)
	observer := newObserver(db)
	warehouse := newWarehouses(state)
	collector, err := newCollector(state, eventLog, observer, warehouse)
	if err != nil {
		return nil, err
	}
	p, err := newProcessor(ctx, state, eventLog, collector.Events())
	if err != nil {
		return nil, err
	}
	d := newDispatcher(eventLog, p.Events())
	startSenders(ctx, d.Events(), d.Done(), state)
	return &Events{collector: collector, observer: observer}, nil
}

// ServeHTTP serves an event request.
func (events *Events) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	events.collector.ServeHTTP(w, r)
}

// Observer returns the event observer.
func (events *Events) Observer() *Observer {
	return events.observer
}
