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

// Actions returns the destination actions where events received from the
// provided source are to be sent. Only active actions of active connections are
// returned.
func (st *eventsState) Actions(source *state.Connection) []*state.Action {
	var actions []*state.Action
	for _, id := range source.EventConnections {
		c, ok := st.state.Connection(id)
		if !ok || !c.Enabled {
			continue
		}
		for _, action := range c.Actions() {
			if action.Enabled && action.Target == state.Events {
				actions = append(actions, action)
			}
		}
	}
	return actions
}

// CanCollectEvents reports whether source can collect events.
// It can collect events if it is enabled and has an enabled action, or is
// enabled and has an enabled event destination with an enabled action on
// events.
func (st *eventsState) CanCollectEvents(source *state.Connection) bool {
	return source.Enabled && (st.HasImportEventsAction(source) ||
		st.HasImportUsersAction(source) || st.HasEventDestinations(source))
}

// HasEventDestinations reports whether source has an enabled event destination
// with an enabled action on events.
func (st *eventsState) HasEventDestinations(source *state.Connection) bool {
	for _, id := range source.EventConnections {
		c, ok := st.state.Connection(id)
		if !ok || !c.Enabled {
			continue
		}
		for _, action := range c.Actions() {
			if action.Enabled && action.Target == state.Events {
				return true
			}
		}
	}
	return false
}

// HasImportEventsAction reports whether source has an enabled action that
// import the events.
func (st *eventsState) HasImportEventsAction(source *state.Connection) bool {
	for _, a := range source.Actions() {
		if a.Enabled && a.Target == state.Events {
			return true
		}
	}
	return false
}

// HasImportUsersAction reports whether source has an enabled action that
// import the users.
func (st *eventsState) HasImportUsersAction(source *state.Connection) bool {
	for _, a := range source.Actions() {
		if a.Enabled && a.Target == state.Users {
			return true
		}
	}
	return false
}
