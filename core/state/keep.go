//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package state

import (
	"encoding/json"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
)

// keep keeps the state updated and in sync with the database.
// It is called in its own goroutine.
func (state *State) keep() {

	defer state.close.Done()
	done := state.close.ctx.Done()

	var n notification

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
		state.changing.Lock()
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
		case "LinkConnection":
			state.linkConnection(n)
		case "LoadState":
			state.loadState(n)
		case "PurgeActions":
			state.purgeActions(n)
		case "RenameConnection":
			state.renameConnection(n)
		case "RenameWorkspace":
			state.renameWorkspace(n)
		case "RevokeConnectionKey":
			state.revokeConnectionKey(n)
		case "SeeLeader":
			state.seeLeader(n)
		case "SetAccount":
			state.setAccount(n)
		case "SetAction":
			state.setAction(n)
		case "SetActionSchedulePeriod":
			state.setActionSchedulePeriod(n)
		case "SetActionSettings":
			state.setActionSettings(n)
		case "SetActionStatus":
			state.setActionStatus(n)
		case "SetConnection":
			state.setConnection(n)
		case "SetConnectionSettings":
			state.setConnectionSettings(n)
		case "SetIdentityResolutionSettings":
			state.setIdentityResolutionSettings(n)
		case "SetWarehouse":
			state.setWarehouse(n)
		case "SetWarehouseMode":
			state.setWarehouseMode(n)
		case "SetWorkspace":
			state.setWorkspace(n)
		case "SetWorkspaceUserSchema":
			state.setWorkspaceUserSchema(n)
		case "UnlinkConnection":
			state.unlinkConnection(n)
		default:
			slog.Warn("unknown notification", "name", n.Name, "pid", n.PID, "payload", n.Payload)
		}
		state.changing.Unlock()
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

// replaceAccount calls the function f passing a copy of the account with
// identifier id. After f is returned, it replaces the account with its copy in
// the state and returns the latter.
func (state *State) replaceAccount(id int, f func(*Account)) *Account {
	a := state.accounts[id]
	aa := new(Account)
	*aa = *a
	f(aa)
	ws := aa.workspace
	ws.mu.Lock()
	ws.accounts[id] = aa
	ws.mu.Unlock()
	// Update the connections.
	for _, connection := range ws.connections {
		if connection.account == a {
			connection.mu.Lock()
			connection.account = aa
			connection.mu.Unlock()
		}
	}
	state.accounts[id] = aa
	return aa
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
	for _, key := range c.Keys {
		state.connectionsByKey[key] = cc
	}
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
	// Update the accounts.
	for _, account := range ww.accounts {
		if account.workspace == w {
			account.mu.Lock()
			account.workspace = ww
			account.mu.Unlock()
		}
	}
	return ww
}

// AddAction is the event sent when an action is added.
type AddAction struct {
	ID                       int
	Connection               int
	Target                   Target
	EventType                string
	Name                     string
	Enabled                  bool
	ScheduleStart            int16
	SchedulePeriod           int16
	InSchema                 types.Type
	OutSchema                types.Type
	Filter                   json.RawMessage `json:",omitempty"`
	Transformation           Transformation
	Query                    string
	Format                   string
	Path                     string
	Sheet                    string
	Compression              Compression
	Settings                 []byte
	ExportMode               ExportMode
	Matching                 Matching
	ExportOnDuplicates       bool
	TableName                string
	TableKeyProperty         string
	IdentityProperty         string
	LastChangeTimeProperty   string
	LastChangeTimeFormat     string
	FileOrderingPropertyPath string
}

// addAction adds a new action.
func (state *State) addAction(n notification) {
	e := AddAction{}
	if !decodeNotification(n, &e) {
		return
	}
	c := state.connections[e.Connection]
	format := state.connectors[e.Format]
	action := &Action{
		mu:                       new(sync.Mutex),
		ID:                       e.ID,
		connection:               c,
		format:                   format,
		Target:                   e.Target,
		Name:                     e.Name,
		Enabled:                  e.Enabled,
		EventType:                e.EventType,
		ScheduleStart:            e.ScheduleStart,
		SchedulePeriod:           e.SchedulePeriod,
		InSchema:                 e.InSchema,
		OutSchema:                e.OutSchema,
		Transformation:           e.Transformation,
		Query:                    e.Query,
		Path:                     e.Path,
		Sheet:                    e.Sheet,
		Compression:              e.Compression,
		Settings:                 e.Settings,
		ExportMode:               e.ExportMode,
		Matching:                 e.Matching,
		ExportOnDuplicates:       e.ExportOnDuplicates,
		TableName:                e.TableName,
		TableKeyProperty:         e.TableKeyProperty,
		IdentityProperty:         e.IdentityProperty,
		LastChangeTimeProperty:   e.LastChangeTimeProperty,
		LastChangeTimeFormat:     e.LastChangeTimeFormat,
		FileOrderingPropertyPath: e.FileOrderingPropertyPath,
	}
	if e.Filter != nil {
		action.Filter, _ = unmarshalWhere(e.Filter, e.InSchema)
	}

	state.mu.Lock()
	state.actions[e.ID] = action
	state.mu.Unlock()
	c.mu.Lock()
	c.actions[e.ID] = action
	c.mu.Unlock()
	dispatchNotification(state, e)

}

// AddConnection is the event sent when a new connection is added.
type AddConnection struct {
	Workspace int      // workspace identifier
	ID        int      // identifier
	Name      string   // name
	Role      Role     // role
	Enabled   bool     // enabled or disabled
	Connector string   // connector name
	Account   struct { // account.
		ID           int       // identifier, can be zero
		Code         string    // code, can be empty.
		AccessToken  string    // access token, can be empty.
		RefreshToken string    // refresh token, can be empty.
		ExpiresIn    time.Time // expiration time, can be the zero time.
	}
	Strategy          *Strategy    // strategy
	SendingMode       *SendingMode // sending mode
	WebsiteHost       string       // website host in form host:port
	LinkedConnections []int        // linked connections
	Key               string       // server key to add
	Settings          []byte
}

// addConnection adds a new connection.
func (state *State) addConnection(n notification) {
	e := AddConnection{}
	if !decodeNotification(n, &e) {
		return
	}
	workspace := state.workspaces[e.Workspace]
	connector := state.connectors[e.Connector]
	var a *Account
	if connector.OAuth != nil {
		if _, ok := state.accounts[e.Account.ID]; ok {
			if e.Account.AccessToken != "" {
				a = state.replaceAccount(e.Account.ID, func(a *Account) {
					a.AccessToken = e.Account.AccessToken
					a.RefreshToken = e.Account.RefreshToken
					a.ExpiresIn = e.Account.ExpiresIn
				})
				// Update the accounts.
				state.mu.Lock()
				state.accounts[a.ID] = a
				state.mu.Unlock()
			}
		} else {
			a = &Account{
				mu:           new(sync.Mutex),
				ID:           e.Account.ID,
				workspace:    workspace,
				connector:    connector,
				Code:         e.Account.Code,
				AccessToken:  e.Account.AccessToken,
				RefreshToken: e.Account.RefreshToken,
				ExpiresIn:    e.Account.ExpiresIn,
			}
			// Update the accounts.
			state.mu.Lock()
			state.accounts[a.ID] = a
			state.mu.Unlock()
			// Update the workspaces.
			workspace.mu.Lock()
			workspace.accounts[a.ID] = a
			workspace.mu.Unlock()
		}
	}
	c := &Connection{
		mu:                new(sync.Mutex),
		organization:      workspace.organization,
		workspace:         workspace,
		ID:                e.ID,
		Name:              e.Name,
		Role:              e.Role,
		Enabled:           e.Enabled,
		connector:         connector,
		account:           a,
		Strategy:          e.Strategy,
		SendingMode:       e.SendingMode,
		WebsiteHost:       e.WebsiteHost,
		LinkedConnections: e.LinkedConnections,
		Settings:          e.Settings,
		actions:           map[int]*Action{},
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
	// Update the linked connections.
	for _, ec := range c.LinkedConnections {
		state.replaceConnection(ec, func(ec *Connection) {
			ec.LinkedConnections = addLinkedConnection(ec.LinkedConnections, c.ID)
		})
	}
	dispatchNotification(state, e)
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

// AddWorkspace is the event sent when a workspace is added.
type AddWorkspace struct {
	ID                             int
	Organization                   int
	Name                           string
	UserSchema                     types.Type
	ResolveIdentitiesOnBatchImport bool
	PrivacyRegion                  PrivacyRegion
	DisplayedProperties            DisplayedProperties
	Warehouse                      struct {
		Name     string
		Mode     WarehouseMode
		Settings json.RawMessage
	}
}

// addWorkspace adds a workspace.
func (state *State) addWorkspace(n notification) {
	e := AddWorkspace{}
	if !decodeNotification(n, &e) {
		return
	}
	organization := state.organizations[e.Organization]
	ws := Workspace{
		mu:                             &sync.Mutex{},
		connections:                    map[int]*Connection{},
		ID:                             e.ID,
		organization:                   organization,
		Name:                           e.Name,
		UserSchema:                     e.UserSchema,
		UserPrimarySources:             map[string]int{},
		ResolveIdentitiesOnBatchImport: e.ResolveIdentitiesOnBatchImport,
		Identifiers:                    []string{},
		PrivacyRegion:                  e.PrivacyRegion,
		DisplayedProperties:            e.DisplayedProperties,
		Warehouse:                      e.Warehouse,
		actionsToPurge:                 []int{},
	}
	state.mu.Lock()
	state.workspaces[e.ID] = &ws
	state.mu.Unlock()
	organization.mu.Lock()
	organization.workspaces[e.ID] = &ws
	organization.mu.Unlock()
	dispatchNotification(state, e)
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
	ws := c.workspace
	if c.Role == Source && e.action.Target == Users {
		actionsToPurge := append(ws.actionsToPurge, e.ID)
		ws.mu.Lock()
		ws.actionsToPurge = actionsToPurge
		ws.mu.Unlock()
	}
	dispatchNotification(state, e)
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
	var actionsToPurge []int
	if e.connection.Role == Source {
		actionsToPurge = ws.actionsToPurge
		for _, action := range e.connection.actions {
			if action.Target == Users {
				actionsToPurge = append(actionsToPurge, action.ID)
			}
		}
	}
	ws.mu.Lock()
	delete(ws.connections, e.ID)
	var found bool
	for _, source := range ws.UserPrimarySources {
		if source == e.ID {
			found = true
			break
		}
	}
	ws.actionsToPurge = actionsToPurge
	ws.mu.Unlock()
	if found {
		sources := map[string]int{}
		for path, source := range ws.UserPrimarySources {
			if source != e.ID {
				sources[path] = source
			}
		}
		state.replaceWorkspace(ws.ID, func(ws *Workspace) {
			ws.UserPrimarySources = sources
		})
	}
	// Update the actions.
	state.mu.Lock()
	for _, a := range e.connection.actions {
		delete(state.actions, a.ID)
	}
	state.mu.Unlock()
	// Remove the connection from the linked connections.
	for _, ec := range e.connection.LinkedConnections {
		state.replaceConnection(ec, func(ec *Connection) {
			ec.LinkedConnections = removeLinkedConnection(ec.LinkedConnections, e.ID)
		})
	}
	dispatchNotification(state, e)
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
	// Delete the accounts.
	for _, a := range e.workspace.accounts {
		delete(state.accounts, a.ID)
	}
	state.mu.Unlock()
	dispatchNotification(state, e)
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
		dispatchNotification(state, e)
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

// ExecuteAction is the event sent when an action is executed.
type ExecuteAction struct {
	ID        int
	Action    int
	Storage   int
	Reload    bool
	Cursor    time.Time
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
		Reload:    e.Reload,
		Cursor:    e.Cursor,
		StartTime: e.StartTime,
	}
	a.mu.Unlock()
	dispatchNotification(state, e)
}

// LinkConnection is the event sent when two unlinked connections are linked.
type LinkConnection struct {
	Connections [2]int
}

// addLinkedConnection links two unlinked connections.
func (state *State) linkConnection(n notification) {
	e := LinkConnection{}
	if !decodeNotification(n, &e) {
		return
	}
	c := state.connections[e.Connections[0]]
	if !slices.Contains(c.LinkedConnections, e.Connections[1]) {
		for i := range 2 {
			state.replaceConnection(e.Connections[i], func(c *Connection) {
				c.LinkedConnections = addLinkedConnection(c.LinkedConnections, e.Connections[(i+1)%2])
			})
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

// PurgeActions is the event sent when actions of a workspace are purged.
type PurgeActions struct {
	Workspace      int
	ActionsToPurge []int // remaining actions to purge.
}

// purgeActions purges actions of a workspace.
func (state *State) purgeActions(n notification) {
	e := PurgeActions{}
	if !decodeNotification(n, &e) {
		return
	}
	ws, _ := state.Workspace(e.Workspace)
	ws.mu.Lock()
	ws.actionsToPurge = e.ActionsToPurge
	ws.mu.Unlock()
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

// SetAccount is the event sent when an account is changed.
type SetAccount struct {
	ID           int
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Time
}

// setAccount sets an account.
func (state *State) setAccount(n notification) {
	e := SetAccount{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceAccount(e.ID, func(a *Account) {
		a.AccessToken = e.AccessToken
		a.RefreshToken = e.RefreshToken
		a.ExpiresIn = e.ExpiresIn
	})
}

// SetAction is the event sent when an action is set.
type SetAction struct {
	ID                       int
	Name                     string
	Enabled                  bool
	InSchema                 types.Type
	OutSchema                types.Type
	Filter                   json.RawMessage `json:",omitempty"`
	Transformation           Transformation
	Query                    string
	Format                   string
	Path                     string
	Sheet                    string
	Compression              Compression
	Settings                 []byte
	ExportMode               ExportMode
	Matching                 Matching
	ExportOnDuplicates       bool
	TableName                string
	TableKeyProperty         string
	IdentityProperty         string
	LastChangeTimeProperty   string
	LastChangeTimeFormat     string
	FileOrderingPropertyPath string
}

// setAction sets an action.
func (state *State) setAction(n notification) {
	e := SetAction{}
	if !decodeNotification(n, &e) {
		return
	}
	format := state.connectors[e.Format]
	var filter *Where
	if e.Filter != nil {
		filter, _ = unmarshalWhere(e.Filter, e.InSchema)
	}
	state.replaceAction(e.ID, func(a *Action) {
		a.format = format
		a.Name = e.Name
		a.Enabled = e.Enabled
		a.InSchema = e.InSchema
		a.OutSchema = e.OutSchema
		a.Filter = filter
		a.Transformation = e.Transformation
		a.Query = e.Query
		a.Path = e.Path
		a.Sheet = e.Sheet
		a.Compression = e.Compression
		a.Settings = e.Settings
		a.ExportMode = e.ExportMode
		a.Matching = e.Matching
		a.ExportOnDuplicates = e.ExportOnDuplicates
		a.TableName = e.TableName
		a.TableKeyProperty = e.TableKeyProperty
		a.IdentityProperty = e.IdentityProperty
		a.LastChangeTimeProperty = e.LastChangeTimeProperty
		a.LastChangeTimeFormat = e.LastChangeTimeFormat
		a.FileOrderingPropertyPath = e.FileOrderingPropertyPath
	})
	dispatchNotification(state, e)
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
	dispatchNotification(state, e)
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
	dispatchNotification(state, e)
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
	dispatchNotification(state, e)
}

// SetWarehouse is the event sent when the settings of a data warehouse are
// changed.
type SetWarehouse struct {
	Workspace                    int
	Mode                         WarehouseMode
	Settings                     json.RawMessage
	CancelIncompatibleOperations bool
}

// setWarehouse sets the settings of a data warehouse.
func (state *State) setWarehouse(n notification) {
	e := SetWarehouse{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.Warehouse.Mode = e.Mode
		w.Warehouse.Settings = e.Settings
		w.actionsToPurge = []int{}
	})
	dispatchNotification(state, e)
}

// SetWarehouseMode is the event sent when the mode of a data warehouse is
// changed.
type SetWarehouseMode struct {
	Workspace                    int
	Mode                         WarehouseMode
	CancelIncompatibleOperations bool
}

// setWarehouseMode sets the mode of a data warehouse.
func (state *State) setWarehouseMode(n notification) {
	e := SetWarehouseMode{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.Warehouse.Mode = e.Mode
	})
	dispatchNotification(state, e)
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
	dispatchNotification(state, e)
}

// SetIdentityResolutionSettings is the event sent when the settings of the
// Identity Resolution of a workspace are changed.
type SetIdentityResolutionSettings struct {
	Workspace                      int
	ResolveIdentitiesOnBatchImport bool
	Identifiers                    []string
}

// setIdentityResolutionSettings sets the Identity Resolution settings of a
// workspace.
func (state *State) setIdentityResolutionSettings(n notification) {
	e := SetIdentityResolutionSettings{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.ResolveIdentitiesOnBatchImport = e.ResolveIdentitiesOnBatchImport
		w.Identifiers = e.Identifiers
	})
}

// SetWorkspaceUserSchema is the event sent when the "users" schema of a
// workspace is changed.
type SetWorkspaceUserSchema struct {
	Workspace      int
	UserSchema     types.Type
	PrimarySources map[string]int
	Identifiers    []string
}

// setWorkspaceUserSchema sets the "users" schema of a workspace.
func (state *State) setWorkspaceUserSchema(n notification) {
	e := SetWorkspaceUserSchema{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.UserSchema = e.UserSchema
		w.UserPrimarySources = e.PrimarySources
		w.Identifiers = e.Identifiers
	})
	dispatchNotification(state, e)
}

// UnlinkConnection is the event sent when two linked connections are unlinked.
type UnlinkConnection struct {
	Connections [2]int
}

// unlinkConnection unlinks two linked connections.
func (state *State) unlinkConnection(n notification) {
	e := UnlinkConnection{}
	if !decodeNotification(n, &e) {
		return
	}
	c := state.connections[e.Connections[0]]
	if slices.Contains(c.LinkedConnections, e.Connections[1]) {
		for i := range 2 {
			state.replaceConnection(e.Connections[i], func(c *Connection) {
				c.LinkedConnections = removeLinkedConnection(c.LinkedConnections, e.Connections[(i+1)%2])
			})
		}
	}
}

// addLinkedConnection adds id to the provided linked connections. It returns
// a copy of connections with id added in numerical order. It is assumed that
// connections is already sorted and id does not already exist in connections.
func addLinkedConnection(connections []int, id int) []int {
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

// removeLinkedConnection removes id from the provided linked connections. It
// returns a copy of connections with id removed. It is assumed that connections
// is sorted and id exists in connections. If id is the sole connection in
// connections, it returns nil.
func removeLinkedConnection(connections []int, id int) []int {
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
