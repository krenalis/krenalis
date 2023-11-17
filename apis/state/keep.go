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
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"chichi/apis/postgres"
	"chichi/connector"
	"chichi/connector/types"

	"github.com/google/uuid"
)

// logNotifications controls the logging of notifications on the log.
const logNotifications = false

// AddListener adds a notification listener.
// It must be called before Keep is called or in a listener execution.
func (state *State) AddListener(listener any) {
	switch l := listener.(type) {
	case func(AddAction):
		state.listeners.AddAction = append(state.listeners.AddAction, l)
	case func(AddConnection):
		state.listeners.AddConnection = append(state.listeners.AddConnection, l)
	case func(DeleteAction):
		state.listeners.DeleteAction = append(state.listeners.DeleteAction, l)
	case func(DeleteConnection):
		state.listeners.DeleteConnection = append(state.listeners.DeleteConnection, l)
	case func(DeleteWorkspace):
		state.listeners.DeleteWorkspace = append(state.listeners.DeleteWorkspace, l)
	case func(ElectLeader):
		state.listeners.ElectLeader = append(state.listeners.ElectLeader, l)
	case func(ExecuteAction):
		state.listeners.ExecuteAction = append(state.listeners.ExecuteAction, l)
	case func(SetAction):
		state.listeners.SetAction = append(state.listeners.SetAction, l)
	case func(SetActionSchedulePeriod):
		state.listeners.SetActionSchedulePeriod = append(state.listeners.SetActionSchedulePeriod, l)
	case func(SetConnection):
		state.listeners.SetConnection = append(state.listeners.SetConnection, l)
	case func(SetConnectionSettings):
		state.listeners.SetConnectionSettings = append(state.listeners.SetConnectionSettings, l)
	case func(SetWarehouse):
		state.listeners.SetWarehouse = append(state.listeners.SetWarehouse, l)
	case func(SetWorkspace):
		state.listeners.SetWorkspace = append(state.listeners.SetWorkspace, l)
	default:
		panic(fmt.Sprintf("state: unexpected listener type %T", listener))
	}
}

// keepState keeps the state in sync with the database. It is called in its own
// goroutine.
func (state *State) keepState() {

	var n postgres.Notification

	done := state.close.ctx.Done()

	for {
		select {
		case <-done:
			return
		case n = <-state.notifications.channel:
		}
		if logNotifications {
			slog.Info("received notification", "pid", n.PID, "name", n.Name, "payload", n.Payload)
		}
		if !state.syncing && n.Name != "LoadState" {
			if n.Ack != nil {
				n.Ack <- struct{}{}
			}
			continue
		}
		switch n.Name {
		case "AddAction":
			state.addAction(n)
		case "AddConnection":
			state.addConnection(n)
		case "AddConnectionKey":
			state.addConnectionKey(n)
		case "AddWorkspace":
			state.addWorkspace(n)
		case "DeleteAction":
			state.deleteAction(n)
		case "DeleteConnection":
			state.deleteConnection(n)
		case "DeleteWorkspace":
			state.deleteWorkspace(n)
		case "ElectLeader":
			state.electLeader(n)
		case "EndActionExecution":
			state.endActionExecution(n)
		case "ExecuteAction":
			state.executeAction(n)
		case "LoadState":
			state.loadState(n)
		case "RenameConnection":
			state.renameConnection(n)
		case "RenameWorkspace":
			state.renameWorkspace(n)
		case "RevokeConnectionKey":
			state.revokeConnectionKey(n)
		case "SeeLeader":
			state.seeLeader(n)
		case "SetAction":
			state.setAction(n)
		case "SetActionSchedulePeriod":
			state.setActionSchedulePeriod(n)
		case "SetActionStatus":
			state.setActionStatus(n)
		case "SetActionUserCursor":
			state.setActionUserCursor(n)
		case "SetConnection":
			state.setConnection(n)
		case "SetConnectionSettings":
			state.setConnectionSettings(n)
		case "SetResource":
			state.setResource(n)
		case "SetWarehouse":
			state.setWarehouse(n)
		case "SetWorkspace":
			state.setWorkspace(n)
		case "SetWorkspaceIdentifiers":
			state.setWorkspaceIdentifiers(n)
		case "SetWorkspaceSchemas":
			state.setWorkspaceSchemas(n)
		default:
			slog.Warn("unknown notification", "name", n.Name, "pid", n.PID, "payload", n.Payload)
		}
		if n.Ack != nil {
			n.Ack <- struct{}{}
		}
	}

}

// decodeNotification decodes a notification.
func decodeNotification(n postgres.Notification, e any) bool {
	err := json.NewDecoder(strings.NewReader(n.Payload)).Decode(&e)
	if err != nil {
		slog.Error("cannot unmarshal notification", "name", n.Name, "pid", n.PID, "err", err)
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
	// Update the storage in the connections.
	if c.connector.Type == StorageType {
		for _, connection := range ws.connections {
			if connection == c {
				continue
			}
			if connection.storage == c {
				connection.mu.Lock()
				connection.storage = cc
				connection.mu.Unlock()
			}
			for _, action := range connection.actions {
				if ex := action.execution; ex != nil && ex.storage == c {
					ex.mu.Lock()
					ex.storage = cc
					ex.mu.Unlock()
				}
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

// AddAction is the event sent when an action is added.
type AddAction struct {
	ID                 int
	Connection         int
	Target             Target
	EventType          string
	Name               string
	Enabled            bool
	ScheduleStart      int16
	SchedulePeriod     int16
	InSchema           types.Type
	OutSchema          types.Type
	Filter             *Filter
	Mapping            map[string]string
	Transformation     *Transformation
	Query              string
	Path               string
	TableName          string
	Sheet              string
	IdentityColumn     string
	TimestampColumn    string
	TimestampFormat    string
	ExportMode         *ExportMode
	MatchingProperties *MatchingProperties
}

// addAction adds a new action.
func (state *State) addAction(n postgres.Notification) {
	e := AddAction{}
	if !decodeNotification(n, &e) {
		return
	}
	c := state.connections[e.Connection]
	action := &Action{
		mu:                 new(sync.Mutex),
		ID:                 e.ID,
		connection:         c,
		Target:             e.Target,
		Name:               e.Name,
		Enabled:            e.Enabled,
		EventType:          e.EventType,
		ScheduleStart:      e.ScheduleStart,
		SchedulePeriod:     e.SchedulePeriod,
		InSchema:           e.InSchema,
		OutSchema:          e.OutSchema,
		Filter:             e.Filter,
		Mapping:            e.Mapping,
		Transformation:     e.Transformation,
		Query:              e.Query,
		Path:               e.Path,
		TableName:          e.TableName,
		Sheet:              e.Sheet,
		IdentityColumn:     e.IdentityColumn,
		TimestampColumn:    e.TimestampColumn,
		TimestampFormat:    e.TimestampFormat,
		ExportMode:         e.ExportMode,
		MatchingProperties: e.MatchingProperties,
	}
	state.mu.Lock()
	state.actions[e.ID] = action
	state.mu.Unlock()
	c.mu.Lock()
	c.actions[e.ID] = action
	c.mu.Unlock()
	for _, listener := range state.listeners.AddAction {
		listener(e)
	}
}

// AddConnection is the event sent when a new connection is added.
type AddConnection struct {
	Workspace   int         // workspace identifier
	ID          int         // identifier
	Name        string      // name
	Role        Role        // role
	Enabled     bool        // enabled or disabled
	Connector   int         // connector identifier
	Storage     int         // storage identifier, can be zero
	Compression Compression // compression
	Resource    struct {    // resource.
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
	e := AddConnection{}
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
		Compression: e.Compression,
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

// AddConnectionKey is the event sent when a connection key is added.
type AddConnectionKey struct {
	Connection   int
	Value        string
	CreationTime time.Time
}

// addConnectionKey adds a new connection key.
func (state *State) addConnectionKey(n postgres.Notification) {
	e := AddConnectionKey{}
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

// ExecuteAction is the event sent when an action is executed.
type ExecuteAction struct {
	ID        int
	Action    int
	Storage   int
	Reimport  bool
	StartTime time.Time
}

// executeAction executes an action.
func (state *State) executeAction(n postgres.Notification) {
	e := ExecuteAction{}
	if !decodeNotification(n, &e) {
		return
	}
	a := state.actions[e.Action]
	var storage *Connection
	if e.Storage > 0 {
		storage = state.connections[e.Storage]
	}
	a.mu.Lock()
	a.execution = &ActionExecution{
		mu:        &sync.Mutex{},
		ID:        e.ID,
		action:    a,
		storage:   storage,
		Reimport:  e.Reimport,
		StartTime: e.StartTime,
	}
	a.mu.Unlock()
	for _, listener := range state.listeners.ExecuteAction {
		listener(e)
	}
}

// AddWorkspace is the event sent when a workspace is added.
type AddWorkspace struct {
	ID            int
	Account       int
	Name          string
	PrivacyRegion PrivacyRegion
}

// addWorkspace adds a workspace.
func (state *State) addWorkspace(n postgres.Notification) {
	e := AddWorkspace{}
	if !decodeNotification(n, &e) {
		return
	}
	account := state.accounts[e.Account]
	ws := Workspace{
		mu:            &sync.Mutex{},
		Schemas:       map[string]*types.Type{},
		connections:   map[int]*Connection{},
		ID:            e.ID,
		account:       account,
		Name:          e.Name,
		PrivacyRegion: e.PrivacyRegion,
	}
	state.mu.Lock()
	state.workspaces[e.ID] = &ws
	state.mu.Unlock()
	account.mu.Lock()
	account.workspaces[e.ID] = &ws
	account.mu.Unlock()
}

// DeleteAction is the event sent when an action is deleted.
type DeleteAction struct {
	Connection int
	ID         int
}

// deleteAction deletes an action.
func (state *State) deleteAction(n postgres.Notification) {
	e := DeleteAction{}
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
	for _, listener := range state.listeners.DeleteAction {
		listener(e)
	}
}

// DeleteConnection is the event sent when a connection is deleted.
type DeleteConnection struct {
	ID int
}

// deleteConnection deletes a connection.
func (state *State) deleteConnection(n postgres.Notification) {
	e := DeleteConnection{}
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

// EndActionExecution is the event sent when action execution ends.
type EndActionExecution struct {
	ID     int
	Health Health
}

// endActionExecution ends an action execution in progress.
func (state *State) endActionExecution(n postgres.Notification) {
	e := EndActionExecution{}
	if !decodeNotification(n, &e) {
		return
	}
	for _, a := range state.actions {
		if ex := a.execution; ex != nil && ex.ID == e.ID {
			state.replaceAction(a.ID, func(a *Action) {
				a.execution = nil
				a.Health = e.Health
			})
			break
		}
	}
}

// DeleteWorkspace is the event sent when a workspace is deleted.
type DeleteWorkspace struct {
	ID int
}

// deleteWorkspace deletes a workspace.
func (state *State) deleteWorkspace(n postgres.Notification) {
	e := DeleteWorkspace{}
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

// ElectLeader is the event sent when a leader is elected.
type ElectLeader struct {
	Number int
	Leader uuid.UUID
}

// electLeader elects a leader.
func (state *State) electLeader(n postgres.Notification) {
	e := ElectLeader{}
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
	previous := state.election.leader
	state.election = election
	state.mu.Unlock()
	if e.Leader != previous {
		for _, listener := range state.listeners.ElectLeader {
			listener(e)
		}
	}
}

// LoadState is the event sent when a state is loaded.
type LoadState struct {
	ID uuid.UUID
}

// loadState loads the state.
func (state *State) loadState(n postgres.Notification) {
	e := LoadState{}
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

// RenameConnection is the event sent when a connection is renamed.
type RenameConnection struct {
	Connection int
	Name       string
}

// renameConnection renames a connection.
func (state *State) renameConnection(n postgres.Notification) {
	e := RenameConnection{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceConnection(e.Connection, func(c *Connection) {
		c.Name = e.Name
	})
}

// RenameWorkspace is the event sent when a workspace is renamed.
type RenameWorkspace struct {
	Workspace int
	Name      string
}

// renameWorkspace renames a workspace.
func (state *State) renameWorkspace(n postgres.Notification) {
	e := RenameWorkspace{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(ws *Workspace) {
		ws.Name = e.Name
	})
}

// RevokeConnectionKey is the event sent when a connection key is revoked.
type RevokeConnectionKey struct {
	Connection int
	Value      string
}

// revokeConnectionKey revokes a connection key.
func (state *State) revokeConnectionKey(n postgres.Notification) {
	e := RevokeConnectionKey{}
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

// SeeLeader is the event sent when the leader is seen.
type SeeLeader struct {
	Election int
}

// seeLeader sees the leader.
func (state *State) seeLeader(n postgres.Notification) {
	e := SeeLeader{}
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

// SetAction is the event sent when an action is set.
type SetAction struct {
	ID                 int
	Name               string
	Enabled            bool
	InSchema           types.Type
	OutSchema          types.Type
	Filter             *Filter
	Mapping            map[string]string
	Transformation     *Transformation
	Query              string
	Path               string
	TableName          string
	Sheet              string
	IdentityColumn     string
	TimestampColumn    string
	TimestampFormat    string
	ExportMode         *ExportMode
	MatchingProperties *MatchingProperties
}

// setAction sets an action.
func (state *State) setAction(n postgres.Notification) {
	e := SetAction{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceAction(e.ID, func(a *Action) {
		a.Name = e.Name
		a.Enabled = e.Enabled
		a.InSchema = e.InSchema
		a.OutSchema = e.OutSchema
		a.Filter = e.Filter
		a.Mapping = e.Mapping
		a.Transformation = e.Transformation
		a.Query = e.Query
		a.Path = e.Path
		a.TableName = e.TableName
		a.Sheet = e.Sheet
		a.IdentityColumn = e.IdentityColumn
		a.TimestampColumn = e.TimestampColumn
		a.TimestampFormat = e.TimestampFormat
		a.ExportMode = e.ExportMode
		a.MatchingProperties = e.MatchingProperties
	})
	for _, listener := range state.listeners.SetAction {
		listener(e)
	}
}

// SetActionSchedulePeriod is the event sent when the schedule period of an
// action is set.
type SetActionSchedulePeriod struct {
	ID             int
	SchedulePeriod int16
}

// setActionSchedulePeriod sets the schedule period of an action.
func (state *State) setActionSchedulePeriod(n postgres.Notification) {
	e := SetActionSchedulePeriod{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceAction(e.ID, func(a *Action) {
		a.SchedulePeriod = e.SchedulePeriod
	})
	for _, listener := range state.listeners.SetActionSchedulePeriod {
		listener(e)
	}
}

// SetActionStatus is the event sent when the status of an action is set.
type SetActionStatus struct {
	ID      int
	Enabled bool
}

// setActionStatus sets the status of an action.
func (state *State) setActionStatus(n postgres.Notification) {
	e := SetActionStatus{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceAction(e.ID, func(a *Action) {
		a.Enabled = e.Enabled
	})
}

// SetActionUserCursor is the event sent when the user cursor of an action is
// set.
type SetActionUserCursor struct {
	ID         int
	UserCursor connector.Cursor
}

// setActionUserCursor sets the user cursor of an action.
func (state *State) setActionUserCursor(n postgres.Notification) {
	e := SetActionUserCursor{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceAction(e.ID, func(a *Action) {
		a.UserCursor = e.UserCursor
	})
}

// SetConnection is the event sent when a connection is changed.
type SetConnection struct {
	Connection  int
	Name        string
	Enabled     bool
	Storage     int
	Compression Compression
	WebsiteHost string
}

// setConnection sets a connection.
func (state *State) setConnection(n postgres.Notification) {
	e := SetConnection{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceConnection(e.Connection, func(c *Connection) {
		c.Name = e.Name
		c.Enabled = e.Enabled
		c.storage = state.connections[e.Storage]
		c.Compression = e.Compression
		c.WebsiteHost = e.WebsiteHost
	})
	for _, listener := range state.listeners.SetConnection {
		listener(e)
	}
}

// SetConnectionSettings is the event sent when the settings of a connection is
// changed.
type SetConnectionSettings struct {
	Connection int
	Settings   []byte
}

// setConnectionSettings sets the settings of a connection.
func (state *State) setConnectionSettings(n postgres.Notification) {
	e := SetConnectionSettings{}
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

// SetResource is the event sent when a resource is changed.
type SetResource struct {
	ID           int
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Time
}

// setResource sets a resource.
func (state *State) setResource(n postgres.Notification) {
	e := SetResource{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceResource(e.ID, func(r *Resource) {
		r.AccessToken = e.AccessToken
		r.RefreshToken = e.RefreshToken
		r.ExpiresIn = e.ExpiresIn
	})
}

// SetWarehouse is the event sent when the settings of a data warehouse are
// changed.
type SetWarehouse struct {
	Workspace int
	Warehouse *Warehouse
	Schemas   map[string]*types.Type // nil if the schemas are not changed.
}

// setWarehouse sets the settings of a data warehouse.
func (state *State) setWarehouse(n postgres.Notification) {
	e := SetWarehouse{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.Warehouse = e.Warehouse
		if e.Schemas != nil {
			w.Schemas = e.Schemas
		}
	})
	for _, listener := range state.listeners.SetWarehouse {
		listener(e)
	}
}

// SetWorkspace is the event sent when the name and the privacy region of a
// workspace are changed.
type SetWorkspace struct {
	Workspace     int
	Name          string
	PrivacyRegion PrivacyRegion
}

// setWorkspace sets the name and the privacy region of a workspace.
func (state *State) setWorkspace(n postgres.Notification) {
	e := SetWorkspace{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.Name = e.Name
		w.PrivacyRegion = e.PrivacyRegion
	})
	for _, listener := range state.listeners.SetWorkspace {
		listener(e)
	}
}

// SetWorkspaceIdentifiers is the event sent when the identifiers and the
// anonymous identifiers of a workspace are changed.
type SetWorkspaceIdentifiers struct {
	Workspace            int
	Identifiers          []string
	AnonymousIdentifiers AnonymousIdentifiers
}

// setWorkspaceIdentifiers sets the identifiers and the anonymous identifier of
// a workspace.
func (state *State) setWorkspaceIdentifiers(n postgres.Notification) {
	e := SetWorkspaceIdentifiers{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.Identifiers = e.Identifiers
		w.AnonymousIdentifiers = e.AnonymousIdentifiers
	})
}

// SetWorkspaceSchemas is the event sent when schemas of a workspace are
// changed.
type SetWorkspaceSchemas struct {
	Workspace int
	Schemas   map[string]*types.Type
}

// setWorkspaceSchemas sets the schemas of a workspace.
func (state *State) setWorkspaceSchemas(n postgres.Notification) {
	e := SetWorkspaceSchemas{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		for name, typ := range e.Schemas {
			if typ == nil {
				e.Schemas[name] = w.Schemas[name]
			}
		}
		w.Schemas = e.Schemas
	})
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
