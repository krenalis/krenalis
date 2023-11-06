//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package events

import (
	"chichi/apis/connectors"
	"chichi/apis/state"
)

// eventsState holds the state for the events.
type eventsState struct {
	state      *state.State
	connectors *connectors.Connectors
}

// newEventsState returns a new eventsState based on the st state.
func newEventsState(st *state.State, connectors *connectors.Connectors) *eventsState {
	eventSt := &eventsState{
		state:      st,
		connectors: connectors,
	}
	return eventSt
}

// ConnectionByKey returns an enable source mobile, server or website connection
// given its key and true, if exists, otherwise returns nil and false.
func (st *eventsState) ConnectionByKey(key string) (*state.Connection, bool) {
	c, ok := st.state.ConnectionByKey(key)
	if ok && c.Enabled && c.Role == state.Source {
		switch c.Connector().Type {
		case state.MobileType, state.ServerType, state.WebsiteType:
			return c, true
		}
	}
	return nil, false
}

// Actions returns the app destination actions that are enabled, have the Events
// target, and their connection is enabled.
func (st *eventsState) Actions() []*state.Action {
	var actions []*state.Action
	for _, action := range st.state.Actions() {
		if !action.Enabled || action.Target != state.Events {
			continue
		}
		c := action.Connection()
		if !c.Enabled || c.Role != state.Destination || c.Connector().Type != state.AppType {
			continue
		}
		actions = append(actions, action)
	}
	return actions
}

// HasEnabledActions reports whether connection is enabled and has at least one
// enabled action.
func (st *eventsState) HasEnabledActions(connection *state.Connection) bool {
	if !connection.Enabled {
		return false
	}
	for _, a := range connection.Actions() {
		if a.Enabled {
			return true
		}
	}
	return false
}
