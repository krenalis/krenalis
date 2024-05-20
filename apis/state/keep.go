//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package state

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/open2b/chichi/types"

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
	case func(SetWarehouseMode):
		state.listeners.SetWarehouseMode = append(state.listeners.SetWarehouseMode, l)
	case func(SetWorkspace):
		state.listeners.SetWorkspace = append(state.listeners.SetWorkspace, l)
	case func(schema SetWorkspaceUsersSchema):
		state.listeners.SetWorkspaceUsersSchema = append(state.listeners.SetWorkspaceUsersSchema, l)
	default:
		panic(fmt.Sprintf("state: unexpected listener type %T", listener))
	}
}

// Keep keeps the state updated.
func (state *State) Keep() {
	go state.keepState()
}

// keepState keeps the state in sync with the database. It is called in its own
// goroutine.
func (state *State) keepState() {

	var n notification

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
		case "AddEventConnection":
			state.addEventConnection(n)
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
		case "RemoveEventConnection":
			state.removeEventConnection(n)
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
		case "SetActionSettings":
			state.setActionSettings(n)
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
		case "SetWarehouseMode":
			state.setWarehouseMode(n)
		case "SetWorkspace":
			state.setWorkspace(n)
		case "SetWorkspaceIdentifiers":
			state.setWorkspaceIdentifiers(n)
		case "SetWorkspaceUsersSchema":
			state.setWorkspaceUsersSchema(n)
		default:
			slog.Warn("unknown notification", "name", n.Name, "pid", n.PID, "payload", n.Payload)
		}
		if n.Ack != nil {
			n.Ack <- struct{}{}
		}
	}

}

// decodeNotification decodes a notification.
func decodeNotification(n notification, e any) bool {
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
	// Update the organization.
	organization := ww.organization
	organization.mu.Lock()
	organization.workspaces[id] = ww
	organization.mu.Unlock()
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
	ID                      int
	Connection              int
	Target                  Target
	EventType               string
	Name                    string
	Enabled                 bool
	ScheduleStart           int16
	SchedulePeriod          int16
	InSchema                types.Type
	OutSchema               types.Type
	Filter                  *Filter
	Transformation          Transformation
	Query                   string
	Connector               string
	Path                    string
	Sheet                   string
	Compression             Compression
	Settings                []byte
	TableName               string
	IdentityProperty        string
	DisplayedProperty       string
	LastChangeTimeProperty  string
	LastChangeTimeFormat    string
	ExportMode              *ExportMode
	MatchingProperties      *MatchingProperties
	ExportOnDuplicatedUsers *bool
}

// addAction adds a new action.
func (state *State) addAction(n notification) {
	e := AddAction{}
	if !decodeNotification(n, &e) {
		return
	}
	c := state.connections[e.Connection]
	connector := state.connectors[e.Connector]
	action := &Action{
		mu:                      new(sync.Mutex),
		ID:                      e.ID,
		connection:              c,
		connector:               connector,
		Target:                  e.Target,
		Name:                    e.Name,
		Enabled:                 e.Enabled,
		EventType:               e.EventType,
		ScheduleStart:           e.ScheduleStart,
		SchedulePeriod:          e.SchedulePeriod,
		InSchema:                e.InSchema,
		OutSchema:               e.OutSchema,
		Filter:                  e.Filter,
		Transformation:          e.Transformation,
		Query:                   e.Query,
		Path:                    e.Path,
		Sheet:                   e.Sheet,
		Compression:             e.Compression,
		Settings:                e.Settings,
		TableName:               e.TableName,
		IdentityProperty:        e.IdentityProperty,
		DisplayedProperty:       e.DisplayedProperty,
		LastChangeTimeProperty:  e.LastChangeTimeProperty,
		LastChangeTimeFormat:    e.LastChangeTimeFormat,
		ExportMode:              e.ExportMode,
		MatchingProperties:      e.MatchingProperties,
		ExportOnDuplicatedUsers: e.ExportOnDuplicatedUsers,
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
	Workspace int      // workspace identifier
	ID        int      // identifier
	Name      string   // name
	Role      Role     // role
	Enabled   bool     // enabled or disabled
	Connector string   // connector name
	Resource  struct { // resource.
		ID           int       // identifier, can be zero
		Code         string    // code, can be empty.
		AccessToken  string    // access token, can be empty.
		RefreshToken string    // refresh token, can be empty.
		ExpiresIn    time.Time // expiration time, can be the zero time.
	}
	Strategy         *Strategy    // strategy
	SendingMode      *SendingMode // sending mode
	WebsiteHost      string       // website host in form host:port
	EventConnections []int        // event connections
	Key              string       // server key to add
	Settings         []byte
}

// addConnection adds a new connection.
func (state *State) addConnection(n notification) {
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
		mu:               new(sync.Mutex),
		organization:     workspace.organization,
		workspace:        workspace,
		ID:               e.ID,
		Name:             e.Name,
		Role:             e.Role,
		Enabled:          e.Enabled,
		connector:        connector,
		resource:         r,
		Strategy:         e.Strategy,
		SendingMode:      e.SendingMode,
		WebsiteHost:      e.WebsiteHost,
		EventConnections: e.EventConnections,
		Settings:         e.Settings,
		actions:          map[int]*Action{},
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
	// Update the event connections.
	for _, ec := range c.EventConnections {
		state.replaceConnection(ec, func(ec *Connection) {
			ec.EventConnections = addEventConnection(ec.EventConnections, c.ID)
		})
	}
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
func (state *State) addConnectionKey(n notification) {
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

// AddEventConnection is the event sent when a connection is added as event
// connection.
type AddEventConnection struct {
	Connections [2]int
}

// addEventConnection adds a connection as event connection.
func (state *State) addEventConnection(n notification) {
	e := AddEventConnection{}
	if !decodeNotification(n, &e) {
		return
	}
	c := state.connections[e.Connections[0]]
	if !slices.Contains(c.EventConnections, e.Connections[1]) {
		for i := range 2 {
			state.replaceConnection(e.Connections[i], func(c *Connection) {
				c.EventConnections = addEventConnection(c.EventConnections, e.Connections[(i+1)%2])
			})
		}
	}
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
func (state *State) executeAction(n notification) {
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
	ID                  int
	Organization        int
	Name                string
	UsersSchema         types.Type
	PrivacyRegion       PrivacyRegion
	DisplayedProperties DisplayedProperties
}

// addWorkspace adds a workspace.
func (state *State) addWorkspace(n notification) {
	e := AddWorkspace{}
	if !decodeNotification(n, &e) {
		return
	}
	organization := state.organizations[e.Organization]
	ws := Workspace{
		mu:                  &sync.Mutex{},
		connections:         map[int]*Connection{},
		ID:                  e.ID,
		organization:        organization,
		Name:                e.Name,
		UsersSchema:         e.UsersSchema,
		PrivacyRegion:       e.PrivacyRegion,
		DisplayedProperties: e.DisplayedProperties,
	}
	state.mu.Lock()
	state.workspaces[e.ID] = &ws
	state.mu.Unlock()
	organization.mu.Lock()
	organization.workspaces[e.ID] = &ws
	organization.mu.Unlock()
}

// DeleteAction is the event sent when an action is deleted.
type DeleteAction struct {
	ID     int
	action *Action
}

func (n DeleteAction) Action() *Action {
	return n.action
}

// deleteAction deletes an action.
func (state *State) deleteAction(n notification) {
	e := DeleteAction{}
	if !decodeNotification(n, &e) {
		return
	}
	e.action = state.actions[e.ID]
	state.mu.Lock()
	delete(state.actions, e.ID)
	state.mu.Unlock()
	c := e.action.connection
	c.mu.Lock()
	delete(c.actions, e.ID)
	c.mu.Unlock()
	for _, listener := range state.listeners.DeleteAction {
		listener(e)
	}
}

// DeleteConnection is the event sent when a connection is deleted.
type DeleteConnection struct {
	ID         int
	connection *Connection
}

func (n DeleteConnection) Connection() *Connection {
	return n.connection
}

// deleteConnection deletes a connection.
func (state *State) deleteConnection(n notification) {
	e := DeleteConnection{}
	if !decodeNotification(n, &e) {
		return
	}
	e.connection = state.connections[e.ID]
	// Update connections and keys.
	state.mu.Lock()
	delete(state.connections, e.ID)
	for _, key := range e.connection.Keys {
		delete(state.connectionsByKey, key)
	}
	state.mu.Unlock()
	// Update the workspace.
	ws := e.connection.workspace
	ws.mu.Lock()
	delete(ws.connections, e.ID)
	ws.mu.Unlock()
	// Update the actions.
	state.mu.Lock()
	for _, a := range e.connection.actions {
		delete(state.actions, a.ID)
	}
	state.mu.Unlock()
	// Remove the connection from event connections.
	for _, ec := range e.connection.EventConnections {
		state.replaceConnection(ec, func(ec *Connection) {
			ec.EventConnections = removeEventConnection(ec.EventConnections, e.ID)
		})
	}
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
func (state *State) endActionExecution(n notification) {
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
	ID        int
	workspace *Workspace
}

func (n DeleteWorkspace) Workspace() *Workspace {
	return n.workspace
}

// deleteWorkspace deletes a workspace.
func (state *State) deleteWorkspace(n notification) {
	e := DeleteWorkspace{}
	if !decodeNotification(n, &e) {
		return
	}
	e.workspace = state.workspaces[e.ID]
	organization := e.workspace.organization
	state.mu.Lock()
	// Delete the workspace.
	delete(state.workspaces, e.ID)
	delete(organization.workspaces, e.ID)
	// Delete the connections.
	for _, c := range e.workspace.connections {
		for _, key := range c.Keys {
			delete(state.connectionsByKey, key)
		}
		delete(state.connections, c.ID)
	}
	// Delete the resources.
	for _, r := range e.workspace.resources {
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
func (state *State) electLeader(n notification) {
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
func (state *State) loadState(n notification) {
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

// RemoveEventConnection is the event sent when a connection is removed as event
// connection.
type RemoveEventConnection struct {
	Connections [2]int
}

// removeEventConnection removes a connection as event connection.
func (state *State) removeEventConnection(n notification) {
	e := RemoveEventConnection{}
	if !decodeNotification(n, &e) {
		return
	}
	c := state.connections[e.Connections[0]]
	if slices.Contains(c.EventConnections, e.Connections[1]) {
		for i := range 2 {
			state.replaceConnection(e.Connections[i], func(c *Connection) {
				c.EventConnections = removeEventConnection(c.EventConnections, e.Connections[(i+1)%2])
			})
		}
	}
}

// RenameConnection is the event sent when a connection is renamed.
type RenameConnection struct {
	Connection int
	Name       string
}

// renameConnection renames a connection.
func (state *State) renameConnection(n notification) {
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
func (state *State) renameWorkspace(n notification) {
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
func (state *State) revokeConnectionKey(n notification) {
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
func (state *State) seeLeader(n notification) {
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
	ID                      int
	Name                    string
	Enabled                 bool
	InSchema                types.Type
	OutSchema               types.Type
	Filter                  *Filter
	Transformation          Transformation
	Query                   string
	Connector               string
	Path                    string
	Sheet                   string
	Compression             Compression
	Settings                []byte
	TableName               string
	IdentityProperty        string
	DisplayedProperty       string
	LastChangeTimeProperty  string
	LastChangeTimeFormat    string
	ExportMode              *ExportMode
	MatchingProperties      *MatchingProperties
	ExportOnDuplicatedUsers *bool
}

// setAction sets an action.
func (state *State) setAction(n notification) {
	e := SetAction{}
	if !decodeNotification(n, &e) {
		return
	}
	connector := state.connectors[e.Connector]
	state.replaceAction(e.ID, func(a *Action) {
		a.connector = connector
		a.Name = e.Name
		a.Enabled = e.Enabled
		a.InSchema = e.InSchema
		a.OutSchema = e.OutSchema
		a.Filter = e.Filter
		a.Transformation = e.Transformation
		a.Query = e.Query
		a.Path = e.Path
		a.Sheet = e.Sheet
		a.Compression = e.Compression
		a.Settings = e.Settings
		a.TableName = e.TableName
		a.IdentityProperty = e.IdentityProperty
		a.DisplayedProperty = e.DisplayedProperty
		a.LastChangeTimeProperty = e.LastChangeTimeProperty
		a.LastChangeTimeFormat = e.LastChangeTimeFormat
		a.ExportMode = e.ExportMode
		a.MatchingProperties = e.MatchingProperties
		a.ExportOnDuplicatedUsers = e.ExportOnDuplicatedUsers
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
func (state *State) setActionSchedulePeriod(n notification) {
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
func (state *State) setActionStatus(n notification) {
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
	UserCursor Cursor
}

// setActionUserCursor sets the user cursor of an action.
func (state *State) setActionUserCursor(n notification) {
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
	Strategy    *Strategy
	SendingMode *SendingMode
	WebsiteHost string
}

// setConnection sets a connection.
func (state *State) setConnection(n notification) {
	e := SetConnection{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceConnection(e.Connection, func(c *Connection) {
		c.Name = e.Name
		c.Enabled = e.Enabled
		c.Strategy = e.Strategy
		c.SendingMode = e.SendingMode
		c.WebsiteHost = e.WebsiteHost
	})
	for _, listener := range state.listeners.SetConnection {
		listener(e)
	}
}

// SetActionSettings is the event sent when the settings of an action is
// changed.
type SetActionSettings struct {
	Action   int
	Settings []byte
}

// setConnectionSettings sets the settings of an action.
func (state *State) setActionSettings(n notification) {
	e := SetActionSettings{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceAction(e.Action, func(a *Action) {
		a.Settings = e.Settings
	})
}

// SetConnectionSettings is the event sent when the settings of a connection is
// changed.
type SetConnectionSettings struct {
	Connection int
	Settings   []byte
}

// setConnectionSettings sets the settings of a connection.
func (state *State) setConnectionSettings(n notification) {
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
func (state *State) setResource(n notification) {
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
}

// setWarehouse sets the settings of a data warehouse.
func (state *State) setWarehouse(n notification) {
	e := SetWarehouse{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.Warehouse = e.Warehouse
	})
	for _, listener := range state.listeners.SetWarehouse {
		listener(e)
	}
}

// SetWarehouseMode is the event sent when the mode of a data warehouse is
// changed.
type SetWarehouseMode struct {
	Workspace int
	Mode      WarehouseMode
}

// setWarehouseMode sets the mode of a data warehouse.
func (state *State) setWarehouseMode(n notification) {
	e := SetWarehouseMode{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.Warehouse = &Warehouse{
			Type:     w.Warehouse.Type,
			Mode:     e.Mode,
			Settings: w.Warehouse.Settings,
		}
	})
	for _, listener := range state.listeners.SetWarehouseMode {
		listener(e)
	}
}

// SetWorkspace is the event sent when the name, the privacy region and the
// displayed properties of a workspace are changed.
type SetWorkspace struct {
	Workspace           int
	Name                string
	PrivacyRegion       PrivacyRegion
	DisplayedProperties DisplayedProperties
}

// setWorkspace sets the name and the privacy region of a workspace.
func (state *State) setWorkspace(n notification) {
	e := SetWorkspace{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.Name = e.Name
		w.PrivacyRegion = e.PrivacyRegion
		w.DisplayedProperties = e.DisplayedProperties
	})
	for _, listener := range state.listeners.SetWorkspace {
		listener(e)
	}
}

// SetWorkspaceIdentifiers is the event sent when the identifiers of a workspace
// are changed.
type SetWorkspaceIdentifiers struct {
	Workspace   int
	Identifiers []string
}

// setWorkspaceIdentifiers sets the identifiers of a workspace.
func (state *State) setWorkspaceIdentifiers(n notification) {
	e := SetWorkspaceIdentifiers{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.Identifiers = e.Identifiers
	})
}

// SetWorkspaceUsersSchema is the event sent when the "users" schema of a
// workspace is changed.
type SetWorkspaceUsersSchema struct {
	Workspace   int
	UsersSchema types.Type
}

// setWorkspaceUsersSchema sets the "users" schema of a workspace.
func (state *State) setWorkspaceUsersSchema(n notification) {
	e := SetWorkspaceUsersSchema{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.UsersSchema = e.UsersSchema
	})
	for _, listener := range state.listeners.SetWorkspaceUsersSchema {
		listener(e)
	}
}

// addEventConnection adds id to connections. It returns a copy of connections
// with id added in numerical order. It is assumed that connections is already
// sorted and id does not already exist in connections.
func addEventConnection(connections []int, id int) []int {
	cc := make([]int, len(connections)+1)
	j := 0
	var added bool
	for _, c := range connections {
		if !added && id < c {
			added = true
			cc[j] = id
			j++
		}
		cc[j] = c
		j++
	}
	if !added {
		cc[j] = id
	}
	return cc
}

// removeEventConnection removes id from connections. It returns a copy of
// connections with id removed. It is assumed that connections is sorted and id
// exists in connections. If id is the sole connection in connections, it
// returns nil.
func removeEventConnection(connections []int, id int) []int {
	if len(connections) == 1 {
		return nil
	}
	cc := make([]int, len(connections)-1)
	j := 0
	var removed bool
	for _, c := range connections {
		if !removed && id == c {
			removed = true
			continue
		}
		cc[j] = c
		j++
	}
	return cc
}
