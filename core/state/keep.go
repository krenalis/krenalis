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
		case "CreateAPIKey":
			state.createAPIKey(n)
		case "CreateAction":
			state.createAction(n)
		case "CreateConnection":
			state.createConnection(n)
		case "CreateWorkspace":
			state.createWorkspace(n)
		case "CreateEventWriteKey":
			state.createEventWriteKey(n)
		case "DeleteAPIKey":
			state.deleteAPIKey(n)
		case "DeleteAction":
			state.deleteAction(n)
		case "DeleteConnection":
			state.deleteConnection(n)
		case "DeleteWorkspace":
			state.deleteWorkspace(n)
		case "DeleteEventWriteKey":
			state.deleteEventWriteKey(n)
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
		case "SeeLeader":
			state.seeLeader(n)
		case "SetAccount":
			state.setAccount(n)
		case "SetActionFormatSettings":
			state.setActionFormatSettings(n)
		case "SetActionSchedulePeriod":
			state.setActionSchedulePeriod(n)
		case "SetActionStatus":
			state.setActionStatus(n)
		case "SetConnectionSettings":
			state.setConnectionSettings(n)
		case "UpdateAction":
			state.updateAction(n)
		case "UpdateConnection":
			state.updateConnection(n)
		case "UpdateIdentityResolutionSettings":
			state.updateIdentityResolutionSettings(n)
		case "UpdateUserSchema":
			state.updateUserSchema(n)
		case "UpdateWarehouse":
			state.updateWarehouse(n)
		case "UpdateWarehouseMode":
			state.updateWarehouseMode(n)
		case "UpdateWorkspace":
			state.updateWorkspace(n)
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

// CreateAPIKey is the event sent when an API key is created.
type CreateAPIKey struct {
	ID           int
	Organization int
	Workspace    int
	Token        string
}

// createAPIKey creates an API key.
func (state *State) createAPIKey(n notification) {
	e := CreateAPIKey{}
	if !decodeNotification(n, &e) {
		return
	}
	key := APIKey{
		ID:           e.ID,
		Organization: e.Organization,
		Workspace:    e.Workspace,
	}
	state.mu.Lock()
	state.apiKeyByToken[e.Token] = &key
	state.mu.Unlock()
}

// CreateAction is the event sent when an action is created.
type CreateAction struct {
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
	FormatSettings           []byte
	ExportMode               ExportMode
	Matching                 Matching
	ExportOnDuplicates       bool
	TableName                string
	TableKey                 string
	IdentityProperty         string
	LastChangeTimeProperty   string
	LastChangeTimeFormat     string
	FileOrderingPropertyPath string
}

// createAction creates a new action.
func (state *State) createAction(n notification) {
	e := CreateAction{}
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
		FormatSettings:           e.FormatSettings,
		ExportMode:               e.ExportMode,
		Matching:                 e.Matching,
		ExportOnDuplicates:       e.ExportOnDuplicates,
		TableName:                e.TableName,
		TableKey:                 e.TableKey,
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

// CreateConnection is the event sent when a new connection is created.
type CreateConnection struct {
	Workspace int      // workspace identifier
	ID        int      // identifier
	Name      string   // name
	Role      Role     // role
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
	EventWriteKey     string       // event write key to add
	Settings          []byte
}

// createConnection creates a new connection.
func (state *State) createConnection(n notification) {
	e := CreateConnection{}
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
		connector:         connector,
		account:           a,
		Strategy:          e.Strategy,
		SendingMode:       e.SendingMode,
		WebsiteHost:       e.WebsiteHost,
		LinkedConnections: e.LinkedConnections,
		Settings:          e.Settings,
		actions:           map[int]*Action{},
	}
	if e.EventWriteKey != "" {
		c.Keys = []string{e.EventWriteKey}
	}
	state.mu.Lock()
	state.connections[e.ID] = c
	if e.EventWriteKey != "" {
		state.connectionsByKey[e.EventWriteKey] = c
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

// CreateWorkspace is the event sent when a workspace is created.
type CreateWorkspace struct {
	ID                             int
	Organization                   int
	Name                           string
	UserSchema                     types.Type
	ResolveIdentitiesOnBatchImport bool
	Warehouse                      struct {
		Type     string
		Mode     WarehouseMode
		Settings json.RawMessage
	}
	UIPreferences UIPreferences
}

// createWorkspace creates a workspace.
func (state *State) createWorkspace(n notification) {
	e := CreateWorkspace{}
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
		Warehouse:                      e.Warehouse,
		UIPreferences:                  e.UIPreferences,
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

// CreateEventWriteKey is the event sent when an event write key is created.
type CreateEventWriteKey struct {
	Connection   int
	Key          string
	CreationTime time.Time
}

// createEventWriteKey creates an event write key.
func (state *State) createEventWriteKey(n notification) {
	e := CreateEventWriteKey{}
	if !decodeNotification(n, &e) {
		return
	}
	c := state.replaceConnection(e.Connection, func(c *Connection) {
		keys := make([]string, len(c.Keys)+1)
		copy(keys, c.Keys)
		keys[len(c.Keys)] = e.Key
		c.Keys = keys
	})
	state.mu.Lock()
	state.connectionsByKey[e.Key] = c
	state.mu.Unlock()
}

// DeleteAPIKey is the event sent when an API key is deleted.
type DeleteAPIKey struct {
	ID int
}

// deleteAPIKey deletes an API key.
func (state *State) deleteAPIKey(n notification) {
	e := DeleteAPIKey{}
	if !decodeNotification(n, &e) {
		return
	}
	state.mu.Lock()
	for token, key := range state.apiKeyByToken {
		if key.ID == e.ID {
			delete(state.apiKeyByToken, token)
			break
		}
	}
	state.mu.Unlock()
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

// DeleteEventWriteKey is the event sent when an event write key is deleted.
type DeleteEventWriteKey struct {
	Connection int
	Key        string
}

// deleteEventWriteKey deletes an event write key.
func (state *State) deleteEventWriteKey(n notification) {
	e := DeleteEventWriteKey{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceConnection(e.Connection, func(c *Connection) {
		keys := make([]string, len(c.Keys)-1)
		i := 0
		for _, key := range c.Keys {
			if key != e.Key {
				keys[i] = key
				i++
			}
		}
		c.Keys = keys
	})
	state.mu.Lock()
	delete(state.connectionsByKey, e.Key)
	state.mu.Unlock()
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
	a.mu.Lock()
	a.execution = &ActionExecution{
		mu:        &sync.Mutex{},
		ID:        e.ID,
		action:    a,
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

// SetActionFormatSettings is the event sent when the format settings of an
// action are changed.
type SetActionFormatSettings struct {
	Action   int
	Settings []byte
}

// setActionFormatSettings sets the format settings of an action.
func (state *State) setActionFormatSettings(n notification) {
	e := SetActionFormatSettings{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceAction(e.Action, func(a *Action) {
		a.FormatSettings = e.Settings
	})
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

// UpdateAction is the event sent when an action is updated.
type UpdateAction struct {
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
	FormatSettings           []byte
	ExportMode               ExportMode
	Matching                 Matching
	ExportOnDuplicates       bool
	TableName                string
	TableKey                 string
	IdentityProperty         string
	LastChangeTimeProperty   string
	LastChangeTimeFormat     string
	FileOrderingPropertyPath string
}

// updateAction updates an action.
func (state *State) updateAction(n notification) {
	e := UpdateAction{}
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
		a.FormatSettings = e.FormatSettings
		a.ExportMode = e.ExportMode
		a.Matching = e.Matching
		a.ExportOnDuplicates = e.ExportOnDuplicates
		a.TableName = e.TableName
		a.TableKey = e.TableKey
		a.IdentityProperty = e.IdentityProperty
		a.LastChangeTimeProperty = e.LastChangeTimeProperty
		a.LastChangeTimeFormat = e.LastChangeTimeFormat
		a.FileOrderingPropertyPath = e.FileOrderingPropertyPath
	})
	dispatchNotification(state, e)
}

// UpdateConnection is the event sent when a connection is updated.
type UpdateConnection struct {
	Connection  int
	Name        string
	Strategy    *Strategy
	SendingMode *SendingMode
	WebsiteHost string
}

// updateConnection updates a connection.
func (state *State) updateConnection(n notification) {
	e := UpdateConnection{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceConnection(e.Connection, func(c *Connection) {
		c.Name = e.Name
		c.Strategy = e.Strategy
		c.SendingMode = e.SendingMode
		c.WebsiteHost = e.WebsiteHost
	})
	dispatchNotification(state, e)
}

// UpdateIdentityResolutionSettings is the event sent when the identity
// resolution settings of a workspace are updated.
type UpdateIdentityResolutionSettings struct {
	Workspace                      int
	ResolveIdentitiesOnBatchImport bool
	Identifiers                    []string
}

// updateIdentityResolutionSettings updates the identity resolution settings of
// a workspace.
func (state *State) updateIdentityResolutionSettings(n notification) {
	e := UpdateIdentityResolutionSettings{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.ResolveIdentitiesOnBatchImport = e.ResolveIdentitiesOnBatchImport
		w.Identifiers = e.Identifiers
	})
}

// UpdateUserSchema is the event sent when a user schema is updated.
type UpdateUserSchema struct {
	Workspace      int
	UserSchema     types.Type
	PrimarySources map[string]int
	Identifiers    []string
}

// updateUserSchema updates a user schema.
func (state *State) updateUserSchema(n notification) {
	e := UpdateUserSchema{}
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

// UpdateWarehouse is the event sent when a warehouse is updated.
type UpdateWarehouse struct {
	Workspace                    int
	Mode                         WarehouseMode
	Settings                     json.RawMessage
	CancelIncompatibleOperations bool
}

// updateWarehouse updates a warehouse.
func (state *State) updateWarehouse(n notification) {
	e := UpdateWarehouse{}
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

// UpdateWarehouseMode is the event sent when the mode of a data warehouse is
// updated.
type UpdateWarehouseMode struct {
	Workspace                    int
	Mode                         WarehouseMode
	CancelIncompatibleOperations bool
}

// updateWarehouseMode updates the mode of a data warehouse.
func (state *State) updateWarehouseMode(n notification) {
	e := UpdateWarehouseMode{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.Warehouse.Mode = e.Mode
	})
	dispatchNotification(state, e)
}

// UpdateWorkspace is the event sent when the name and the displayed properties
// of a workspace are updated.
type UpdateWorkspace struct {
	Workspace     int
	Name          string
	UIPreferences UIPreferences
}

// updateWorkspace updates the name and the displayed properties of a workspace.
func (state *State) updateWorkspace(n notification) {
	e := UpdateWorkspace{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.Name = e.Name
		w.UIPreferences = e.UIPreferences
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
