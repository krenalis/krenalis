//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package events

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"unicode/utf8"

	"chichi/apis/httpclient"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/connector"
)

// maxSettingsLen is the maximum length of settings in runes.
// Keep in sync with the apis.maxSettingsLen constant.
const maxSettingsLen = 10_000

// eventsState holds the state for the events.
type eventsState struct {
	sync.Mutex
	db           *postgres.DB
	state        *state.State
	http         *httpclient.HTTP
	destinations map[int]connector.AppEventsConnection
}

// newEventsState returns a new eventsState based on the st state.
func newEventsState(db *postgres.DB, st *state.State, http *httpclient.HTTP) *eventsState {
	eventSt := &eventsState{
		db:           db,
		state:        st,
		http:         http,
		destinations: map[int]connector.AppEventsConnection{},
	}
	for _, c := range st.Connections() {
		if !isDestination(c) {
			continue
		}
		err := eventSt.openDestination(c)
		if err != nil {
			slog.Error("cannot open destination", "id", c.ID, "err", err)
			continue
		}
	}
	st.AddListener(eventSt.onAddConnection)
	st.AddListener(eventSt.onDeleteConnection)
	st.AddListener(eventSt.onDeleteWorkspace)
	st.AddListener(eventSt.onSetConnection)
	st.AddListener(eventSt.onSetConnectionSettings)
	st.AddListener(eventSt.onSetWarehouse)
	st.AddListener(eventSt.onSetWorkspace)
	return eventSt
}

// Source returns the enabled source connection with the identifier id and true,
// if it exists, otherwise returns nil and false.
func (st *eventsState) Source(id int) (*state.Connection, bool) {
	source, ok := st.state.Connection(id)
	if ok && source.Enabled && source.Role == state.Source {
		return source, true
	}
	return nil, false
}

// Destination returns an open connection to the destination with identifier id
// and true, if such destination has been opened, otherwise returns nil and
// false.
func (st *eventsState) Destination(id int) (connector.AppEventsConnection, bool) {
	st.Lock()
	d, ok := st.destinations[id]
	st.Unlock()
	return d, ok
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

// onAddConnection is called when a connection is added.
func (st *eventsState) onAddConnection(n state.AddConnection) {
	c, _ := st.state.Connection(n.ID)
	if !isDestination(c) {
		return
	}
	err := st.openDestination(c)
	if err != nil {
		slog.Error("cannot open destination", "id", c.ID, "err", err)
		return
	}
}

// onDeleteConnection is called when a connection is deleted.
func (st *eventsState) onDeleteConnection(n state.DeleteConnection) {
	st.deleteDestination(n.ID)
}

// onDeleteWorkspace is called when a workspace is deleted.
func (st *eventsState) onDeleteWorkspace(n state.DeleteWorkspace) {
	toKeep := map[int]struct{}{}
	for _, c := range st.state.Connections() {
		toKeep[c.ID] = struct{}{}
	}
	st.Lock()
	for id := range st.destinations {
		if _, ok := toKeep[id]; !ok {
			delete(st.destinations, id)
		}
	}
	st.Unlock()
}

// onSetConnection is called when a connection changes.
func (st *eventsState) onSetConnection(n state.SetConnection) {
	if n.Enabled {
		c, _ := st.state.Connection(n.Connection)
		if !isDestination(c) {
			return
		}
		err := st.openDestination(c)
		if err != nil {
			slog.Error("cannot open destination", "id", c.ID, "err", err)
			return
		}
		return
	}
	// Disabling a connection.
	st.deleteDestination(n.Connection)
}

// onSetConnectionSettings is called when the settings of a connections are
// changed.
func (st *eventsState) onSetConnectionSettings(n state.SetConnectionSettings) {
	c, _ := st.state.Connection(n.Connection)
	if !isDestination(c) {
		return
	}
	err := st.openDestination(c)
	if err != nil {
		slog.Error("cannot open destination", "id", c.ID, "err", err)
		return
	}
}

// onSetWarehouse is called when the warehouse settings of a workspace are
// changed.
func (st *eventsState) onSetWarehouse(n state.SetWarehouse) {
	ws, _ := st.state.Workspace(n.Workspace)
	for _, c := range ws.Connections() {
		if c.Enabled && c.Role == state.Destination {
			if ws.Warehouse == nil {
				st.deleteDestination(c.ID)
				continue
			}
			if isDestination(c) {
				err := st.openDestination(c)
				if err != nil {
					slog.Error("cannot open destination", "id", c.ID, "err", err)
					continue
				}
			}
		}
	}
}

// onSetWorkspace is called when the name and the privacy region of a workspace
// are changed.
func (st *eventsState) onSetWorkspace(n state.SetWorkspace) {
	ws, _ := st.state.Workspace(n.Workspace)
	for _, c := range ws.Connections() {
		if !isDestination(c) {
			continue
		}
		err := st.openDestination(c)
		if err != nil {
			slog.Error("cannot open destination", "id", c.ID, "err", err)
			continue
		}
	}
}

// openDestination opens the destination from the connection c and sets it into
// the state.
func (st *eventsState) openDestination(c *state.Connection) error {

	var resource string
	if r, ok := c.Resource(); ok {
		resource = r.Code
	}

	app := connector.RegisteredApp(c.Connector().Name)
	connection, err := app.Open(&connector.AppConfig{
		Role:     connector.Role(c.Role),
		Settings: c.Settings,
		SetSettings: func(ctx context.Context, settings []byte) error {
			return setSettings(ctx, st.db, c.ID, settings)
		},
		Resource:   resource,
		HTTPClient: st.http.ConnectionClient(c.ID),
		Region:     connector.PrivacyRegion(c.Workspace().PrivacyRegion),
	})
	if err != nil {
		return err
	}

	st.Lock()
	st.destinations[c.ID] = connection.(connector.AppEventsConnection)
	st.Unlock()

	return nil
}

// deleteDestination deletes the destination with identifier id from the state.
// It does nothing if the connection does not exist in the state.
func (st *eventsState) deleteDestination(id int) {
	st.Lock()
	delete(st.destinations, id)
	st.Unlock()
}

// isDestination reports whether c is a destination, that is an enabled
// destination app connection with Events target that belongs to a
// workspace with an associated warehouse.
func isDestination(c *state.Connection) bool {
	conn := c.Connector()
	ws := c.Workspace()
	return c.Enabled && c.Role == state.Destination &&
		conn.Type == state.AppType && ws.Warehouse != nil &&
		conn.Targets.Contains(state.Events)
}

// setSettings sets the given settings of the given connection.
// It is a copy of the apis.setSettings function, so keep in sync.
func setSettings(ctx context.Context, db *postgres.DB, connection int, settings []byte) error {
	if !utf8.Valid(settings) {
		return errors.New("settings is not valid UTF-8")
	}
	if len(settings) > maxSettingsLen && utf8.RuneCount(settings) > maxSettingsLen {
		return fmt.Errorf("settings is longer than %d runes", maxSettingsLen)
	}
	n := state.SetConnectionSettings{
		Connection: connection,
		Settings:   settings,
	}
	err := db.Transaction(ctx, func(tx *postgres.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE connections SET settings = $1 WHERE id = $2", n.Settings, n.Connection)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
	})
	return err
}
