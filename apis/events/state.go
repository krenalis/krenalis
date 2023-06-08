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
	"log"
	"sync"
	"unicode/utf8"

	"chichi/apis/httpclient"
	"chichi/apis/postgres"
	"chichi/apis/state"
	_warehouses "chichi/apis/warehouses"
	"chichi/connector"
)

// maxSettingsLen is the maximum length of settings in runes.
// Keep in sync with the apis.maxSettingsLen constant.
const maxSettingsLen = 10_000

// eventsState holds the state for the events.
type eventsState struct {
	sync.Mutex
	ctx          context.Context
	db           *postgres.DB
	state        *state.State
	http         *httpclient.HTTP
	destinations map[int]connector.AppEventsConnection
}

// newEventsState returns a new eventsState based on the st state.
func newEventsState(ctx context.Context, db *postgres.DB, st *state.State, http *httpclient.HTTP) *eventsState {
	eventSt := &eventsState{
		ctx:          ctx,
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
			log.Printf("cannot open destination %d: %s", c.ID, err)
			continue
		}
	}
	st.AddListener(eventSt.onAddConnection)
	st.AddListener(eventSt.onDeleteConnection)
	st.AddListener(eventSt.onDeleteWorkspace)
	st.AddListener(eventSt.onSetConnectionSettings)
	st.AddListener(eventSt.onSetConnectionStatus)
	st.AddListener(eventSt.onSetWarehouseSettings)
	st.AddListener(eventSt.onSetWorkspacePrivacyRegion)
	return eventSt
}

// Source returns the enabled source connection with the identifier id and true,
// if it exists, otherwise returns nil and false.
func (st *eventsState) Source(id int) (*state.Connection, bool) {
	source, ok := st.state.Connection(id)
	if ok && source.Enabled && source.Role == state.SourceRole {
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

// ServerByKey returns an enable destination server connection given its key and
// true, if exists, otherwise returns nil and false.
func (st *eventsState) ServerByKey(key string) (*state.Connection, bool) {
	server, ok := st.state.ConnectionByKey(key)
	if ok && server.Enabled && server.Role == state.SourceRole &&
		server.Connector().Type == state.ServerType {
		return server, true
	}
	return nil, false
}

// Warehouse returns the warehouse of a workspace and true, if it exists and has
// one, otherwise nil and false.
func (st *eventsState) Warehouse(workspace int) (_warehouses.Warehouse, bool) {
	ws, ok := st.state.Workspace(workspace)
	if !ok || ws.Warehouse == nil {
		return nil, false
	}
	return ws.Warehouse, true
}

// Actions returns the enabled actions for every enabled connection.
func (st *eventsState) Actions() []*state.Action {
	var actions []*state.Action
	for _, action := range st.state.Actions() {
		if !action.Enabled || action.Target != state.EventsTarget {
			continue
		}
		c := action.Connection()
		if !c.Enabled || c.Role != state.DestinationRole || c.Connector().Type != state.AppType {
			continue
		}
		actions = append(actions, action)
	}
	return actions
}

// HasEnabledActions reports whether connection is enabled and has at least one
// enabled action.
func (st *eventsState) HasEnabledActions(connection int) bool {
	c, _ := st.state.Connection(connection)
	if !c.Enabled {
		return false
	}
	for _, a := range c.Actions() {
		if a.Enabled {
			return true
		}
	}
	return false
}

// onAddConnection is called when a connection is added.
func (st *eventsState) onAddConnection(n state.AddConnectionNotification) {
	c, _ := st.state.Connection(n.ID)
	if !isDestination(c) {
		return
	}
	err := st.openDestination(c)
	if err != nil {
		log.Printf("cannot open destination %d: %s", c.ID, err)
		return
	}
}

// onDeleteConnection is called when a connection is deleted.
func (st *eventsState) onDeleteConnection(n state.DeleteConnectionNotification) {
	st.deleteDestination(n.ID)
}

// onDeleteWorkspace is called when a workspace is deleted.
func (st *eventsState) onDeleteWorkspace(n state.DeleteWorkspaceNotification) {
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

// onSetConnectionSettings is called when the settings of a connections are
// changed.
func (st *eventsState) onSetConnectionSettings(n state.SetConnectionSettingsNotification) {
	c, _ := st.state.Connection(n.Connection)
	if !isDestination(c) {
		return
	}
	err := st.openDestination(c)
	if err != nil {
		log.Printf("cannot open destination %d: %s", c.ID, err)
		return
	}
}

// onSetConnectionStatus is called when the status of a connection changes.
func (st *eventsState) onSetConnectionStatus(n state.SetConnectionStatusNotification) {
	if n.Enabled {
		c, _ := st.state.Connection(n.Connection)
		if !isDestination(c) {
			return
		}
		err := st.openDestination(c)
		if err != nil {
			log.Printf("cannot open destination %d: %s", c.ID, err)
			return
		}
		return
	}
	// Disabling a connection.
	st.deleteDestination(n.Connection)
}

// onSetWarehouseSettings is called when the warehouse settings of a workspace
// are changed.
func (st *eventsState) onSetWarehouseSettings(n state.SetWarehouseSettingsNotification) {
	ws, _ := st.state.Workspace(n.Workspace)
	for _, c := range ws.Connections() {
		if c.Enabled && c.Role == state.DestinationRole {
			if c.Workspace().Warehouse == nil {
				st.deleteDestination(c.ID)
				continue
			}
			if isDestination(c) {
				err := st.openDestination(c)
				if err != nil {
					log.Printf("cannot open destination %d: %s", c.ID, err)
					continue
				}
			}
		}
	}
}

// onSetWorkspacePrivacyRegion is called when the privacy region of a workspace
// is changed.
func (st *eventsState) onSetWorkspacePrivacyRegion(n state.SetWorkspacePrivacyRegion) {
	ws, _ := st.state.Workspace(n.Workspace)
	for _, c := range ws.Connections() {
		if !isDestination(c) {
			continue
		}
		err := st.openDestination(c)
		if err != nil {
			log.Printf("cannot open destination %d: %s", c.ID, err)
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
	connection, err := app.Open(st.ctx, &connector.AppConfig{
		Role:     connector.Role(c.Role),
		Settings: c.Settings,
		SetSettings: func(settings []byte) error {
			return setSettings(st.ctx, st.db, c.ID, settings)
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
	return c.Enabled && c.Role == state.DestinationRole &&
		conn.Type == state.AppType && c.Workspace().Warehouse != nil &&
		conn.Targets.Contains(state.EventsTarget)
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
	n := state.SetConnectionSettingsNotification{
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
