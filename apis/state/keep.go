//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package state

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"chichi/apis/postgres"
	"chichi/apis/types"
	"chichi/apis/warehouses"
	"chichi/apis/warehouses/clickhouse"
	"chichi/apis/warehouses/postgresql"

	"github.com/google/uuid"
)

// logNotifications controls the logging of notifications on the log.
const logNotifications = false

// AddListener adds a notification listener.
// It panics if it is called after Keep is called.
func (state *State) AddListener(listener any) {
	state.mu.Lock()
	keeping := state.keeping
	state.mu.Unlock()
	if keeping {
		panic(errors.New("state: cannot call AddListener after Keep has been called"))
	}
	switch l := listener.(type) {
	case func(AddConnectionNotification):
		state.listeners.AddConnection = append(state.listeners.AddConnection, l)
	case func(AddImportInProgressNotification):
		state.listeners.AddImportInProgress = append(state.listeners.AddImportInProgress, l)
	case func(DeleteConnectionNotification):
		state.listeners.DeleteConnection = append(state.listeners.DeleteConnection, l)
	case func(DeleteWorkspaceNotification):
		state.listeners.DeleteWorkspace = append(state.listeners.DeleteWorkspace, l)
	case func(ElectLeaderNotification):
		state.listeners.ElectLeader = append(state.listeners.ElectLeader, l)
	case func(SetConnectionSettingsNotification):
		state.listeners.SetConnectionSettings = append(state.listeners.SetConnectionSettings, l)
	case func(SetConnectionStatusNotification):
		state.listeners.SetConnectionStatus = append(state.listeners.SetConnectionStatus, l)
	case func(SetConnectionUserQueryNotification):
		state.listeners.SetConnectionUserQuery = append(state.listeners.SetConnectionUserQuery, l)
	case func(SetWarehouseSettingsNotification):
		state.listeners.SetWarehouseSettings = append(state.listeners.SetWarehouseSettings, l)
	default:
		panic(fmt.Sprintf("state: unexpected listener type %T", listener))
	}
}

// Keep keeps the state updated.
func (state *State) Keep() {
	state.mu.Lock()
	state.keeping = true
	state.mu.Unlock()
	go state.keepState()
}

// keepState keeps the state in sync with the database. It is called in its own
// goroutine.
func (state *State) keepState() {

	var n postgres.Notification

	for {
		select {
		case <-state.ctx.Done():
			return
		case n = <-state.notifications:
		}
		if logNotifications {
			log.Printf("[info] received notification from pid %d and name %q : %s",
				n.PID, n.Name, n.Payload)
		}
		if !state.syncing && n.Name != "LoadState" {
			continue
		}
		switch n.Name {
		case "AddConnection":
			state.addConnection(n)
		case "AddConnectionAction":
			state.addConnectionAction(n)
		case "AddConnectionKey":
			state.addConnectionKey(n)
		case "AddImportInProgress":
			state.addImportInProgress(n)
		case "AddWorkspace":
			state.addWorkspace(n)
		case "DeleteConnection":
			state.deleteConnection(n)
		case "DeleteConnectionAction":
			state.deleteConnectionAction(n)
		case "DeleteImportInProgress":
			state.deleteImportInProgress(n)
		case "DeleteWorkspace":
			state.deleteWorkspace(n)
		case "ElectLeader":
			state.electLeader(n)
		case "LoadState":
			state.loadState(n)
		case "RenameConnection":
			state.renameConnection(n)
		case "RenameWorkspace":
			state.renameWorkspace(n)
		case "RevokeConnectionKey":
			state.revokeConnectionKey(n)
		case "SetConnectionAction":
			state.setConnectionAction(n)
		case "SetConnectionActionStatus":
			state.setConnectionActionStatus(n)
		case "SetConnectionActionTypes":
			state.setConnectionActionTypes(n)
		case "SeeLeader":
			state.seeLeader(n)
		case "SetConnectionSettings":
			state.setConnectionSettings(n)
		case "SetConnectionStorage":
			state.setConnectionStorage(n)
		case "SetConnectionStatus":
			state.setConnectionStatus(n)
		case "SetConnectionTransformation":
			state.setConnectionTransformation(n)
		case "SetConnectionMappings":
			state.setConnectionMappings(n)
		case "SetConnectionUserQuery":
			state.setConnectionUserQuery(n)
		case "SetConnectionUserSchema":
			state.setConnectionUserSchema(n)
		case "SetResource":
			state.setResource(n)
		case "SetWarehouseSettings":
			state.setWarehouseSettings(n)
		case "SetWorkspaceSchemas":
			state.setWorkspaceSchemas(n)
		default:
			log.Printf("[warning] unknown notification %q received from %d: %s", n.Name, n.PID, n.Payload)
		}
	}

}

// decodeNotification decodes a notification.
func decodeNotification(n postgres.Notification, e any) bool {
	err := json.NewDecoder(strings.NewReader(n.Payload)).Decode(&e)
	if err != nil {
		log.Printf("[error] cannot unmarshal notification %s from %d: %s", n.Name, n.PID, err)
		return false
	}
	return true
}

// replaceAction calls the function f passing a copy of the action with
// identifier id. After f is returned, it replaces the action with its copy in
// the state and returns the latter.
func (state *State) replaceAction(id int, f func(*Action)) *Action {
	a := state.actions[id]
	aa := new(Action)
	*aa = *a
	f(aa)
	state.mu.Lock()
	state.actions[id] = aa
	state.mu.Unlock()
	// Update the connection.
	c := a.connection
	c.mu.Lock()
	c.actions[id] = aa
	c.mu.Unlock()
	return aa
}

// replaceConnection calls the function f passing a copy of the connection with
// identifier id. After f is returned, it replaces the connection with its
// copy in the state and returns the latter.
func (state *State) replaceConnection(id int, f func(*Connection)) *Connection {
	c := state.connections[id]
	cc := new(Connection)
	*cc = *c
	f(cc)
	state.mu.Lock()
	state.connections[id] = cc
	state.mu.Unlock()
	// Update the workspaces.
	ws := cc.workspace
	ws.mu.Lock()
	ws.connections[id] = cc
	ws.mu.Unlock()
	// Update the connections.
	for _, connection := range ws.connections {
		if connection.storage == c {
			connection.mu.Lock()
			connection.storage = cc
			connection.mu.Unlock()
		}
		if imp := connection.importInProgress; imp != nil {
			if imp.connection == c {
				imp.mu.Lock()
				imp.connection = cc
				imp.mu.Unlock()
			}
			if imp.storage == c {
				imp.mu.Lock()
				imp.storage = cc
				imp.mu.Unlock()
			}
		}
	}
	// Update the actions.
	for _, action := range c.actions {
		action.mu.Lock()
		action.connection = cc
		action.mu.Unlock()
	}
	return cc
}

// replaceResource calls the function f passing a copy of the resource with
// identifier id. After f is returned, it replaces the resource with its copy
// in the state and returns the latter.
func (state *State) replaceResource(id int, f func(*Resource)) *Resource {
	r := state.resources[id]
	rr := new(Resource)
	*rr = *r
	f(rr)
	ws := rr.workspace
	ws.mu.Lock()
	ws.resources[id] = rr
	ws.mu.Unlock()
	// Update the connections.
	for _, connection := range ws.connections {
		if connection.resource == r {
			connection.mu.Lock()
			connection.resource = rr
			connection.mu.Unlock()
		}
	}
	state.resources[id] = rr
	return rr
}

// replaceWorkspace calls the function f passing a copy of the workspace with
// identifier id. After f is returned, it replaces the workspace with its
// copy in the state and returns the latter.
func (state *State) replaceWorkspace(id int, f func(*Workspace)) *Workspace {
	w := state.workspaces[id]
	ww := new(Workspace)
	*ww = *w
	f(ww)
	state.mu.Lock()
	state.workspaces[id] = ww
	state.mu.Unlock()
	// Update the account.
	account := ww.account
	account.mu.Lock()
	account.workspaces[id] = ww
	account.mu.Unlock()
	// Update the connections.
	for _, connection := range ww.connections {
		if connection.workspace == w {
			connection.mu.Lock()
			connection.workspace = ww
			connection.mu.Unlock()
		}

	}
	// Update the resources.
	for _, resource := range ww.resources {
		if resource.workspace == w {
			resource.mu.Lock()
			resource.workspace = ww
			resource.mu.Unlock()
		}
	}
	return ww
}

// AddConnectionNotification is the notification event sent when a new
// connection is added.
type AddConnectionNotification struct {
	Workspace int            // workspace identifier
	ID        int            // identifier
	Name      string         // name
	Role      ConnectionRole // role
	Enabled   bool           // enabled or disabled
	Connector int            // connector identifier
	Storage   int            // storage identifier, can be zero
	Resource  struct {       // resource.
		ID           int       // identifier, can be zero
		Code         string    // code, can be empty.
		AccessToken  string    // access token, can be empty.
		RefreshToken string    // refresh token, can be empty.
		ExpiresIn    time.Time // expiration time, can be the zero time.
	}
	WebsiteHost string // website host in form host:port
	Key         string // server key to add
	Settings    []byte
}

// addConnection adds a new connection.
func (state *State) addConnection(n postgres.Notification) {
	e := AddConnectionNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	workspace := state.workspaces[e.Workspace]
	connector := state.connectors[e.Connector]
	var r *Resource
	if connector.OAuth != nil {
		if _, ok := state.resources[e.Resource.ID]; ok {
			if e.Resource.AccessToken != "" {
				r = state.replaceResource(e.Resource.ID, func(r *Resource) {
					r.AccessToken = e.Resource.AccessToken
					r.RefreshToken = e.Resource.RefreshToken
					r.ExpiresIn = e.Resource.ExpiresIn
				})
				// Update the resources.
				state.mu.Lock()
				state.resources[r.ID] = r
				state.mu.Unlock()
			}
		} else {
			r = &Resource{
				mu:           new(sync.Mutex),
				ID:           e.Resource.ID,
				workspace:    workspace,
				connector:    connector,
				Code:         e.Resource.Code,
				AccessToken:  e.Resource.AccessToken,
				RefreshToken: e.Resource.RefreshToken,
				ExpiresIn:    e.Resource.ExpiresIn,
			}
			// Update the resources.
			state.mu.Lock()
			state.resources[r.ID] = r
			state.mu.Unlock()
			// Update the workspaces.
			workspace.mu.Lock()
			workspace.resources[r.ID] = r
			workspace.mu.Unlock()
		}
	}
	c := &Connection{
		mu:          new(sync.Mutex),
		account:     workspace.account,
		workspace:   workspace,
		ID:          e.ID,
		Name:        e.Name,
		Role:        e.Role,
		Enabled:     e.Enabled,
		connector:   connector,
		storage:     state.connections[e.Storage],
		resource:    r,
		WebsiteHost: e.WebsiteHost,
		Settings:    e.Settings,
		actions:     map[int]*Action{},
	}
	if e.Key != "" {
		c.Keys = []string{e.Key}
	}
	state.mu.Lock()
	state.connections[e.ID] = c
	if e.Key != "" {
		state.connectionsByKey[e.Key] = c
	}
	state.mu.Unlock()
	// Update the workspace.
	workspace.mu.Lock()
	workspace.connections[c.ID] = c
	workspace.mu.Unlock()
	for _, listener := range state.listeners.AddConnection {
		listener(e)
	}
}

// AddConnectionKeyNotification is the notification event sent when a
// connection key is added.
type AddConnectionKeyNotification struct {
	Connection   int
	Value        string
	CreationTime time.Time
}

// addConnectionKey adds a new connection key.
func (state *State) addConnectionKey(n postgres.Notification) {
	e := AddConnectionKeyNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	c := state.replaceConnection(e.Connection, func(c *Connection) {
		keys := make([]string, len(c.Keys)+1)
		copy(keys, c.Keys)
		keys[len(c.Keys)] = e.Value
		c.Keys = keys
	})
	state.mu.Lock()
	state.connectionsByKey[e.Value] = c
	state.mu.Unlock()
}

// AddConnectionActionNotification is the notification event sent when a
// connection action is added.
type AddConnectionActionNotification struct {
	ID             int
	Connection     int
	ActionType     int
	Name           string
	Enabled        bool
	Endpoint       int
	Filter         ActionFilterNotification
	Mapping        map[string]string
	Transformation *Transformation
}

// ActionFilterNotification represents the action filter associated to a
// notification which adds or sets an action.
type ActionFilterNotification struct {
	Logical    string
	Conditions []ActionFilterConditionNotification
}

// ActionFilterConditionNotification represents one of the action filter
// conditions associated to a notification which adds or sets an action.
type ActionFilterConditionNotification struct {
	Property string
	Operator string
	Value    string
}

// addConnectionAction adds a new connection action.
func (state *State) addConnectionAction(n postgres.Notification) {
	e := AddConnectionActionNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	c := state.connections[e.Connection]
	action := &Action{
		mu:             new(sync.Mutex),
		ID:             e.ID,
		connection:     c,
		ActionType:     e.ActionType,
		Name:           e.Name,
		Enabled:        e.Enabled,
		Endpoint:       e.Endpoint,
		Mapping:        e.Mapping,
		Transformation: e.Transformation,
	}
	action.Filter.Logical = e.Filter.Logical
	action.Filter.Conditions = make([]ActionFilterCondition, len(e.Filter.Conditions))
	for i := range action.Filter.Conditions {
		action.Filter.Conditions[i] = ActionFilterCondition(e.Filter.Conditions[i])
	}
	state.mu.Lock()
	state.actions[e.ID] = action
	state.mu.Unlock()
	c.mu.Lock()
	c.actions[e.ID] = action
	c.mu.Unlock()
}

// AddImportInProgressNotification is the notification event sent when an
// import in progress is added.
type AddImportInProgressNotification struct {
	ID         int
	Connection int
	Storage    int
	Reimport   bool
	StartTime  time.Time
}

// addImportInProgress adds an import in progress.
func (state *State) addImportInProgress(n postgres.Notification) {
	e := AddImportInProgressNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	c := state.connections[e.Connection]
	c.mu.Lock()
	c.importInProgress = &ImportInProgress{
		mu:         new(sync.Mutex),
		ID:         e.ID,
		connection: c,
		storage:    state.connections[e.Storage],
		Reimport:   e.Reimport,
		StartTime:  e.StartTime,
	}
	c.mu.Unlock()
	for _, listener := range state.listeners.AddImportInProgress {
		listener(e)
	}
}

// AddWorkspaceNotification is the notification event sent when a workspace is added.
type AddWorkspaceNotification struct {
	ID        int
	Account   int
	Name      string
	Warehouse struct {
		Type     WarehouseType
		Settings json.RawMessage `json:",omitempty"`
	}
}

// addWorkspace adds a workspace.
func (state *State) addWorkspace(n postgres.Notification) {
	e := AddWorkspaceNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	account := state.accounts[e.Account]
	ws := Workspace{
		mu:          &sync.Mutex{},
		Schemas:     map[string]*types.Type{},
		connections: map[int]*Connection{},
		ID:          e.ID,
		account:     account,
	}
	if e.Warehouse.Settings != nil {
		warehouse, err := openWarehouse(e.Warehouse.Type, e.Warehouse.Settings)
		if err != nil {
			log.Printf("[error] cannot open data warehouse of workspace %d: %s", e.ID, err)
		}
		ws.Warehouse = warehouse
	}
	state.mu.Lock()
	state.workspaces[e.ID] = &ws
	state.mu.Unlock()
	account.mu.Lock()
	account.workspaces[e.ID] = &ws
	account.mu.Unlock()
}

// DeleteConnectionNotification is the notification event sent when a
// connection is deleted.
type DeleteConnectionNotification struct {
	ID int
}

// deleteConnection deletes a connection.
func (state *State) deleteConnection(n postgres.Notification) {
	e := DeleteConnectionNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	connection := state.connections[e.ID]
	// Update connections and keys.
	state.mu.Lock()
	delete(state.connections, e.ID)
	for _, key := range connection.Keys {
		delete(state.connectionsByKey, key)
	}
	state.mu.Unlock()
	// Update the workspace.
	ws := connection.workspace
	ws.mu.Lock()
	delete(ws.connections, e.ID)
	ws.mu.Unlock()
	// Update the connections.
	for _, c := range connection.workspace.connections {
		if c.storage == connection {
			c.mu.Lock()
			c.storage = nil
			c.mu.Unlock()
		}
	}
	// Update the actions.
	state.mu.Lock()
	for _, a := range connection.actions {
		delete(state.actions, a.ID)
	}
	state.mu.Unlock()
	for _, listener := range state.listeners.DeleteConnection {
		listener(e)
	}
}

// DeleteConnectionActionNotification is the notification event sent when a
// connection action is deleted.
type DeleteConnectionActionNotification struct {
	Connection int
	ID         int
}

// deleteConnectionAction deletes a connection action.
func (state *State) deleteConnectionAction(n postgres.Notification) {
	e := DeleteConnectionActionNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	state.mu.Lock()
	delete(state.actions, e.ID)
	state.mu.Unlock()
	c := state.connections[e.Connection]
	c.mu.Lock()
	delete(c.actions, e.ID)
	c.mu.Unlock()
}

// DeleteImportInProgressNotification is the notification event sent when an
// import in progress is deleted.
type DeleteImportInProgressNotification struct {
	ID     int
	Health ConnectionHealth
}

// deleteImportInProgress deletes an import in progress.
func (state *State) deleteImportInProgress(n postgres.Notification) {
	e := DeleteImportInProgressNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	for _, c := range state.connections {
		if imp := c.importInProgress; imp != nil && imp.ID == e.ID {
			state.replaceConnection(c.ID, func(c *Connection) {
				c.importInProgress = nil
				c.Health = e.Health
			})
			break
		}
	}
}

// DeleteWorkspaceNotification is the notification event sent when a workspace
// is deleted.
type DeleteWorkspaceNotification struct {
	ID int
}

// deleteWorkspace deletes a workspace.
func (state *State) deleteWorkspace(n postgres.Notification) {
	e := DeleteWorkspaceNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	ws := state.workspaces[e.ID]
	account := state.accounts[ws.account.ID]
	state.mu.Lock()
	// Delete the workspace.
	delete(state.workspaces, e.ID)
	delete(account.workspaces, e.ID)
	// Delete the connections.
	for _, c := range ws.connections {
		for _, key := range c.Keys {
			delete(state.connectionsByKey, key)
		}
		delete(state.connections, c.ID)
	}
	// Delete the resources.
	for _, r := range ws.resources {
		delete(state.resources, r.ID)
	}
	state.mu.Unlock()
	for _, listener := range state.listeners.DeleteWorkspace {
		listener(e)
	}
}

// ElectLeaderNotification is the notification sent when a leader is elected.
type ElectLeaderNotification struct {
	Number int
	Leader uuid.UUID
}

// electLeader elects a leader.
func (state *State) electLeader(n postgres.Notification) {
	e := ElectLeaderNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	// Update election.
	election := election{
		number:   e.Number,
		leader:   e.Leader,
		lastSeen: time.Now(),
	}
	state.mu.Lock()
	state.election = election
	state.mu.Unlock()
	for _, listener := range state.listeners.ElectLeader {
		listener(e)
	}
}

// LoadStateNotification is the notification sent when a state is loaded.
type LoadStateNotification struct {
	ID uuid.UUID
}

// loadState loads the state.
func (state *State) loadState(n postgres.Notification) {
	e := LoadStateNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	if e.ID == state.id {
		state.syncing = true
		state.mu.Lock()
		state.election.lastSeen = time.Now()
		state.mu.Unlock()
		go state.keepElections()
	}
}

// RenameConnectionNotification is the notification event sent when a
// connection is renamed.
type RenameConnectionNotification struct {
	Connection int
	Name       string
}

// renameConnection renames a connection.
func (state *State) renameConnection(n postgres.Notification) {
	e := RenameConnectionNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceConnection(e.Connection, func(c *Connection) {
		c.Name = e.Name
	})
}

// RenameWorkspaceNotification is the notification event sent when a
// workspace is renamed.
type RenameWorkspaceNotification struct {
	Workspace int
	Name      string
}

// renameWorkspace renames a workspace.
func (state *State) renameWorkspace(n postgres.Notification) {
	e := RenameWorkspaceNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(ws *Workspace) {
		ws.Name = e.Name
	})
}

// RevokeConnectionKeyNotification is the notification event sent when a
// connection key is revoked.
type RevokeConnectionKeyNotification struct {
	Connection int
	Value      string
}

// revokeConnectionKey revokes a connection key.
func (state *State) revokeConnectionKey(n postgres.Notification) {
	e := RevokeConnectionKeyNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceConnection(e.Connection, func(c *Connection) {
		keys := make([]string, len(c.Keys)-1)
		i := 0
		for _, key := range c.Keys {
			if key != e.Value {
				keys[i] = key
				i++
			}
		}
		c.Keys = keys
	})
	state.mu.Lock()
	delete(state.connectionsByKey, e.Value)
	state.mu.Unlock()
}

// SeeLeaderNotification is the notification sent when the leader is seen.
type SeeLeaderNotification struct {
	Election int
}

// seeLeader sees the leader.
func (state *State) seeLeader(n postgres.Notification) {
	e := SeeLeaderNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	now := time.Now()
	state.mu.Lock()
	if state.election.number == e.Election {
		state.election.lastSeen = now
	}
	state.mu.Unlock()
}

// SetConnectionActionNotification is the notification sent when a connection
// action is set.
type SetConnectionActionNotification struct {
	ID             int
	Connection     int
	ActionType     int
	Name           string
	Enabled        bool
	Endpoint       int
	Filter         ActionFilterNotification
	Mapping        map[string]string
	Transformation *Transformation
}

// setConnectionAction sets a connection action.
func (state *State) setConnectionAction(n postgres.Notification) {
	e := SetConnectionActionNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceAction(e.ID, func(a *Action) {
		a.ActionType = e.ActionType
		a.Name = e.Name
		a.Enabled = e.Enabled
		a.Endpoint = e.Endpoint
		a.Filter.Logical = e.Filter.Logical
		a.Filter.Conditions = make([]ActionFilterCondition, len(e.Filter.Conditions))
		for i := range a.Filter.Conditions {
			a.Filter.Conditions[i] = ActionFilterCondition(e.Filter.Conditions[i])
		}
		a.Mapping = e.Mapping
		a.Transformation = e.Transformation
	})
}

// SetConnectionActionStatusNotification is the notification sent when the
// status of a connection action is set.
type SetConnectionActionStatusNotification struct {
	ID      int
	Enabled bool
}

// setConnectionActionStatus sets the status of a connection action.
func (state *State) setConnectionActionStatus(n postgres.Notification) {
	e := AddConnectionActionNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceAction(e.ID, func(a *Action) {
		a.Enabled = e.Enabled
	})
}

// SetConnectionActionTypesNotification is the notification sent when the
// action types of a connection are set.
type SetConnectionActionTypesNotification struct {
	Connection  int
	ActionTypes []*ActionType
}

// setConnectionActionTypes sets the action types of a connection.
func (state *State) setConnectionActionTypes(n postgres.Notification) {
	e := SetConnectionActionTypesNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	c := state.connections[e.Connection]
	actionTypes := make(map[int]*ActionType, len(e.ActionTypes))
	for _, at := range e.ActionTypes {
		actionTypes[at.ID] = at
	}
	c.mu.Lock()
	c.actionTypes = actionTypes
	c.mu.Unlock()
}

// SetConnectionSettingsNotification is the notification event sent when the
// settings of a connection is changed.
type SetConnectionSettingsNotification struct {
	Connection int
	Settings   []byte
}

// setConnectionSettings sets the settings of a connection.
func (state *State) setConnectionSettings(n postgres.Notification) {
	e := SetConnectionSettingsNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceConnection(e.Connection, func(c *Connection) {
		c.Settings = e.Settings
	})
	for _, listener := range state.listeners.SetConnectionSettings {
		listener(e)
	}
}

// SetConnectionStatusNotification is the notification event sent when a
// connection status is changed.
type SetConnectionStatusNotification struct {
	Connection int
	Enabled    bool
}

// setConnectionStatus changes a connection status.
func (state *State) setConnectionStatus(n postgres.Notification) {
	e := SetConnectionStatusNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceConnection(e.Connection, func(c *Connection) {
		c.Enabled = e.Enabled
	})
	for _, listener := range state.listeners.SetConnectionStatus {
		listener(e)
	}
}

// SetConnectionStorageNotification is the notification event sent when the
// settings of a connection is changed.
type SetConnectionStorageNotification struct {
	Connection int
	Storage    int
}

// setConnectionStorages sets the storage of a connection.
func (state *State) setConnectionStorage(n postgres.Notification) {
	e := SetConnectionStorageNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	c := state.connections[e.Connection]
	storage := state.connections[e.Storage]
	c.mu.Lock()
	c.storage = storage
	c.mu.Unlock()
}

// SetConnectionTransformationNotification is the notification event sent when
// the transformation of a connection is set.
type SetConnectionTransformationNotification struct {
	Connection     int
	Transformation *Transformation // nil means no transformation.
}

// setConnectionTransformation sets the transformation of a connection.
func (state *State) setConnectionTransformation(n postgres.Notification) {
	e := SetConnectionTransformationNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	c := state.connections[e.Connection]
	c.mu.Lock()
	c.transformation = e.Transformation
	c.mu.Unlock()
}

// SetConnectionMappingsNotification is the notification event sent when the
// mappings of a connection are saved.
type SetConnectionMappingsNotification struct {
	Connection int
	Mappings   []*Mapping
}

// setConnectionMappings sets the mappings of a connection.
func (state *State) setConnectionMappings(n postgres.Notification) {
	e := SetConnectionMappingsNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	c := state.connections[e.Connection]
	c.mu.Lock()
	c.mappings = e.Mappings
	c.mu.Unlock()
}

// SetConnectionUserQueryNotification is the notification event sent when a
// user query of a connection is changed.
type SetConnectionUserQueryNotification struct {
	Connection int
	Query      string
}

// setConnectionUserQuery sets the user query of a connection.
func (state *State) setConnectionUserQuery(n postgres.Notification) {
	e := SetConnectionUserQueryNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceConnection(e.Connection, func(c *Connection) {
		c.UsersQuery = e.Query
	})
	for _, listener := range state.listeners.SetConnectionUserQuery {
		listener(e)
	}
}

// SetConnectionUserSchemaNotification is the notification event sent when the
// user schema of a connection is changed.
type SetConnectionUserSchemaNotification struct {
	Connection int
	Schema     types.Type
}

// setConnectionUserSchema sets the user schema of a connection.
func (state *State) setConnectionUserSchema(n postgres.Notification) {
	e := SetConnectionUserSchemaNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceConnection(e.Connection, func(c *Connection) {
		c.Schema = e.Schema
	})
}

// SetResourceNotification is the notification event sent when a resource is
// changed.
type SetResourceNotification struct {
	ID           int
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Time
}

// setResource sets a resource.
func (state *State) setResource(n postgres.Notification) {
	e := SetResourceNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceResource(e.ID, func(r *Resource) {
		r.AccessToken = e.AccessToken
		r.RefreshToken = e.RefreshToken
		r.ExpiresIn = e.ExpiresIn
	})
}

// SetWarehouseSettingsNotification is the notification event sent when the
// settings of a data warehouse are changed.
type SetWarehouseSettingsNotification struct {
	Workspace int
	Type      WarehouseType
	Settings  json.RawMessage `json:",omitempty"`
}

// setWarehouseSettings sets the settings of a data warehouse.
func (state *State) setWarehouseSettings(n postgres.Notification) {
	e := SetWarehouseSettingsNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	disconnected := state.workspaces[e.Workspace].Warehouse
	if e.Settings == nil {
		state.replaceWorkspace(e.Workspace, func(w *Workspace) {
			w.Warehouse = nil
		})
	} else {
		var err error
		state.replaceWorkspace(e.Workspace, func(w *Workspace) {
			w.Warehouse, err = openWarehouse(e.Type, e.Settings)
		})
		if err != nil {
			log.Printf("[error] cannot open data warehouse of workspace %d: %s", e.Workspace, err)
		}
	}
	// Close the disconnected warehouse.
	if disconnected != nil {
		go func() {
			err := disconnected.Close()
			if err != nil {
				// TODO(marco): write the error into a workspace specific log
				log.Printf("[error] error occurred disconnecting the warehouse: %s", err)
			}
		}()
	}
	for _, listener := range state.listeners.SetWarehouseSettings {
		listener(e)
	}
}

// SetWorkspaceSchemasNotification is the notification event sent when schemas
// of a workspace are changed.
type SetWorkspaceSchemasNotification struct {
	Workspace int
	Schemas   map[string]*types.Type
}

// setWorkspaceSchemas sets the schemas of a workspace.
func (state *State) setWorkspaceSchemas(n postgres.Notification) {
	e := SetWorkspaceSchemasNotification{}
	if !decodeNotification(n, &e) {
		return
	}
	var unchanged []string
	for name, typ := range e.Schemas {
		if typ == nil {
			unchanged = append(unchanged, name)
		}
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		for _, name := range unchanged {
			e.Schemas[name] = w.Schemas[name]
		}
		w.Schemas = e.Schemas
	})
}

// openWarehouse opens a data warehouse with the given type and settings.
// It returns an error if typ or settings are not valid.
func openWarehouse(typ WarehouseType, settings []byte) (warehouses.Warehouse, error) {
	switch typ {
	case BigQuery, Redshift, Snowflake:
		return nil, fmt.Errorf("warehouse type %s is not yet supported", typ)
	case PostgreSQL:
		return postgresql.Open(settings)
	case ClickHouse:
		return clickhouse.Open(settings)
	}
	return nil, fmt.Errorf("warehouse type %d is not valid", typ)
}

// WarehouseType represents a data warehouse type.
type WarehouseType int

const (
	BigQuery WarehouseType = iota + 1
	ClickHouse
	PostgreSQL
	Redshift
	Snowflake
)

// String returns the string representation of typ.
// It panics if typ is not a valid WarehouseType value.
func (typ WarehouseType) String() string {
	s, err := typ.Value()
	if err != nil {
		panic("invalid warehouse type")
	}
	return s.(string)
}

// Scan implements the sql.Scanner interface.
func (typ *WarehouseType) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an WarehouseType value", src)
	}
	var t WarehouseType
	switch s {
	case "BigQuery":
		t = BigQuery
	case "ClickHouse":
		t = ClickHouse
	case "PostgreSQL":
		t = PostgreSQL
	case "Redshift":
		t = Redshift
	case "Snowflake":
		t = Snowflake
	default:
		return fmt.Errorf("invalid WarehouseType: %s", s)
	}
	*typ = t
	return nil
}

// Value implements driver.Valuer interface.
// It returns an error if typ is not a valid WarehouseType.
func (typ WarehouseType) Value() (driver.Value, error) {
	switch typ {
	case BigQuery:
		return "BigQuery", nil
	case ClickHouse:
		return "ClickHouse", nil
	case PostgreSQL:
		return "PostgreSQL", nil
	case Redshift:
		return "Redshift", nil
	case Snowflake:
		return "Snowflake", nil
	}
	return nil, fmt.Errorf("not a valid WarehouseType: %d", typ)
}
