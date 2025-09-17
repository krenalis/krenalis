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
	"strings"
	"sync"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/analytics-go"
	_json "github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/core/types"

	"github.com/google/uuid"
)

const logNotifications = false // Set to true to enable logging of received notifications.

// keep keeps the state updated and in sync with the database.
// It is called in its own goroutine.
func (state *State) keep() {

	// If sending statistics is enabled, initialize the Meergo analytics client.
	var client analytics.Client
	if state.sendStats {
		client = analytics.New("eEC2uyWaJ1XmFNEq0dkH0a872GzZChUV", "https://chichi.open2b.net/api/v1/events")
		defer func() {
			err := client.Close()
			if err != nil {
				slog.Error("error while closing analytics.Client", "err", err)
			}
		}()
	}

	defer state.close.Done()

	done := state.close.ctx.Done()
	notifications := state.notifications.ch

	var n notification

	for {
		select {
		case <-done:
			return
		case n = <-notifications:
		}
		if logNotifications {
			slog.Info("core/state: received notification", "id", n.ID, "name", n.Name, "payload", n.Payload)
		}
		state.changing.Lock()
		switch n.Name {
		case "CreateAccessKey":
			state.createAccessKey(n)
		case "CreateAction":
			state.createAction(n)
		case "CreateConnection":
			state.createConnection(n)
		case "CreateWorkspace":
			state.createWorkspace(n)
		case "CreateEventWriteKey":
			state.createEventWriteKey(n)
		case "DeleteAccessKey":
			state.deleteAccessKey(n)
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
		case "EndAlterUserSchema":
			state.endAlterUserSchema(n)
		case "EndIdentityResolution":
			state.endIdentityResolution(n)
		case "ExecuteAction":
			state.executeAction(n)
		case "LinkConnection":
			state.linkConnection(n)
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
		case "StartAlterUserSchema":
			state.startAlterUserSchema(n)
		case "StartIdentityResolution":
			state.startIdentityResolution(n)
		case "UpdateAction":
			state.updateAction(n)
		case "UpdateConnection":
			state.updateConnection(n)
		case "UpdateIdentityPropertiesToUnset":
			state.updateIdentityPropertiesToUnset(n)
		case "UpdateIdentityResolutionSettings":
			state.updateIdentityResolutionSettings(n)
		case "UpdateWarehouse":
			state.updateWarehouse(n)
		case "UpdateWarehouseMode":
			state.updateWarehouseMode(n)
		case "UpdateWorkspace":
			state.updateWorkspace(n)
		case "UnlinkConnection":
			state.unlinkConnection(n)
		default:
			slog.Warn("core/state: unknown notification", "id", n.ID, "name", n.Name, "payload", n.Payload)
		}
		state.changing.Unlock()
		if n.ID > 0 {
			// Acknowledge that the notification has been received.
			if ack, ok := state.notifications.acks.LoadAndDelete(n.ID); ok {
				ack.(chan struct{}) <- struct{}{}
			}
		}
		if state.sendStats {
			state.sendNotificationStats(client, n)
		}
	}

}

// decodeNotification decodes a notification.
func decodeNotification(n notification, e any) bool {
	err := json.NewDecoder(strings.NewReader(n.Payload)).Decode(&e)
	if err != nil {
		slog.Error("core/state: cannot unmarshal notification", "id", n.ID, "name", n.Name, "err", err)
		return false
	}
	return true
}

// replaceAccount calls the function f passing a copy of the account with
// identifier id. After f is returned, it replaces the account with its copy in
// the workspace and returns the latter.
func (workspace *Workspace) replaceAccount(id int, f func(*Account)) *Account {
	a := workspace.accounts[id]
	aa := new(Account)
	*aa = *a
	f(aa)
	workspace.mu.Lock()
	workspace.accounts[id] = aa
	workspace.mu.Unlock()
	// Update the connections.
	for _, connection := range workspace.connections {
		if connection.account == a {
			connection.mu.Lock()
			connection.account = aa
			connection.mu.Unlock()
		}
	}
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

// CreateAccessKey is the event sent when an access key is created.
type CreateAccessKey struct {
	ID           int
	Organization int
	Workspace    int
	Type         AccessKeyType
	Token        string
}

// createAccessKey creates an access key.
func (state *State) createAccessKey(n notification) {
	e := CreateAccessKey{}
	if !decodeNotification(n, &e) {
		return
	}
	key := AccessKey{
		ID:           e.ID,
		Organization: e.Organization,
		Workspace:    e.Workspace,
		Type:         e.Type,
	}
	state.mu.Lock()
	state.accessKeyByToken[e.Token] = &key
	state.mu.Unlock()
}

// CreateAction is the event sent when an action is created.
type CreateAction struct {
	ID                   int
	Connection           int
	Target               Target
	EventType            string
	Name                 string
	Enabled              bool
	ScheduleStart        int16
	SchedulePeriod       int16
	InSchema             types.Type
	OutSchema            types.Type
	Filter               json.RawMessage
	Transformation       Transformation
	Query                string
	Format               string
	Path                 string
	Sheet                string
	Compression          Compression
	OrderBy              string
	FormatSettings       []byte
	ExportMode           ExportMode
	Matching             Matching
	UpdateOnDuplicates   bool
	TableName            string
	TableKey             string
	IdentityColumn       string
	LastChangeTimeColumn string
	LastChangeTimeFormat string
	Incremental          bool
}

// createAction creates a new action.
func (state *State) createAction(n notification) {
	e := CreateAction{}
	if !decodeNotification(n, &e) {
		return
	}
	// json.RawMessage(nil) is marshaled into "null", but when it is
	// deserialized it becomes json.RawMessage("null"), so this code converts it
	// back to json.RawMessage(nil).
	if _json.Value(e.Filter).IsNull() {
		e.Filter = nil
	}
	c := state.connections[e.Connection]
	format := state.connectors[e.Format]
	action := &Action{
		mu:                   new(sync.Mutex),
		ID:                   e.ID,
		connection:           c,
		format:               format,
		Target:               e.Target,
		Name:                 e.Name,
		Enabled:              e.Enabled,
		EventType:            e.EventType,
		ScheduleStart:        e.ScheduleStart,
		SchedulePeriod:       e.SchedulePeriod,
		InSchema:             e.InSchema,
		OutSchema:            e.OutSchema,
		Transformation:       e.Transformation,
		Query:                e.Query,
		Path:                 e.Path,
		Sheet:                e.Sheet,
		Compression:          e.Compression,
		OrderBy:              e.OrderBy,
		FormatSettings:       e.FormatSettings,
		ExportMode:           e.ExportMode,
		Matching:             e.Matching,
		UpdateOnDuplicates:   e.UpdateOnDuplicates,
		TableName:            e.TableName,
		TableKey:             e.TableKey,
		IdentityColumn:       e.IdentityColumn,
		LastChangeTimeColumn: e.LastChangeTimeColumn,
		LastChangeTimeFormat: e.LastChangeTimeFormat,
		Incremental:          e.Incremental,
	}
	if c.Role == Source && e.Target == TargetUser {
		action.propertiesToUnset = []string{}
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
	Connector string   // connector name
	Role      Role     // role
	Account   struct { // account.
		ID           int       // identifier, can be zero
		Code         string    // code, can be empty.
		AccessToken  string    // access token, can be empty.
		RefreshToken string    // refresh token, can be empty.
		ExpiresIn    time.Time // expiration time, can be the zero time.
	}
	Strategy          *Strategy    // strategy
	SendingMode       *SendingMode // sending mode
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
	ws := state.workspaces[e.Workspace]
	connector := state.connectors[e.Connector]
	var a *Account
	if connector.OAuth != nil {
		if _, ok := ws.accounts[e.Account.ID]; ok {
			if e.Account.AccessToken != "" {
				// Update the workspace.
				a = ws.replaceAccount(e.Account.ID, func(a *Account) {
					a.AccessToken = e.Account.AccessToken
					a.RefreshToken = e.Account.RefreshToken
					a.ExpiresIn = e.Account.ExpiresIn
				})
			}
		} else {
			a = &Account{
				mu:           new(sync.Mutex),
				ID:           e.Account.ID,
				workspace:    ws,
				connector:    connector,
				Code:         e.Account.Code,
				AccessToken:  e.Account.AccessToken,
				RefreshToken: e.Account.RefreshToken,
				ExpiresIn:    e.Account.ExpiresIn,
			}
			// Update the workspace.
			ws.mu.Lock()
			ws.accounts[a.ID] = a
			ws.mu.Unlock()
		}
	}
	c := &Connection{
		mu:                new(sync.Mutex),
		organization:      ws.organization,
		workspace:         ws,
		ID:                e.ID,
		Name:              e.Name,
		connector:         connector,
		Role:              e.Role,
		account:           a,
		Strategy:          e.Strategy,
		SendingMode:       e.SendingMode,
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
	ws.mu.Lock()
	ws.connections[c.ID] = c
	ws.mu.Unlock()
	// Update the linked connections.
	for _, lc := range c.LinkedConnections {
		state.replaceConnection(lc, func(lc *Connection) {
			lc.LinkedConnections = addLinkedConnection(lc.LinkedConnections, c.ID)
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
		Type        string
		Mode        WarehouseMode
		Settings    json.RawMessage
		MCPSettings json.RawMessage
	}
	UIPreferences UIPreferences
}

// createWorkspace creates a workspace.
func (state *State) createWorkspace(n notification) {
	e := CreateWorkspace{}
	if !decodeNotification(n, &e) {
		return
	}
	// json.RawMessage(nil) is marshaled into "null", but when it is
	// deserialized it becomes json.RawMessage("null"), so this code converts it
	// back to json.RawMessage(nil).
	if _json.Value(e.Warehouse.MCPSettings).IsNull() {
		e.Warehouse.MCPSettings = nil
	}
	organization := state.organizations[e.Organization]
	ws := Workspace{
		mu:                             &sync.Mutex{},
		connections:                    map[int]*Connection{},
		executions:                     map[int]*ActionExecution{},
		ID:                             e.ID,
		organization:                   organization,
		Name:                           e.Name,
		UserSchema:                     e.UserSchema,
		UserPrimarySources:             map[string]int{},
		accounts:                       map[int]*Account{},
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
	Connection int
	Key        string
	CreatedAt  time.Time
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

// DeleteAccessKey is the event sent when an access key is deleted.
type DeleteAccessKey struct {
	ID int
}

// deleteAccessKey deletes an access key.
func (state *State) deleteAccessKey(n notification) {
	e := DeleteAccessKey{}
	if !decodeNotification(n, &e) {
		return
	}
	state.mu.Lock()
	for token, key := range state.accessKeyByToken {
		if key.ID == e.ID {
			delete(state.accessKeyByToken, token)
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
	if c.Role == Source && e.action.Target == TargetUser {
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
	Account    bool // indicates whether the associated account was also deleted.
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
	if e.Account {
		ws.mu.Lock()
		delete(ws.accounts, e.connection.account.ID)
		ws.mu.Unlock()
	}
	var actionsToPurge []int
	if e.connection.Role == Source {
		actionsToPurge = ws.actionsToPurge
		for _, action := range e.connection.actions {
			if action.Target == TargetUser {
				actionsToPurge = append(actionsToPurge, action.ID)
			}
		}
	}
	ws.mu.Lock()
	delete(ws.connections, e.ID)
	if e.Account {
		delete(ws.accounts, e.connection.account.ID)
	}
	// Mark whether the connection is found between the current primary sources
	// or between the pending ones.
	var found bool
	for _, source := range ws.UserPrimarySources {
		if source == e.ID {
			found = true
			break
		}
	}
	for _, source := range ws.AlterUserSchema.PrimarySources {
		if source == e.ID {
			found = true
			break
		}
	}
	ws.actionsToPurge = actionsToPurge
	ws.mu.Unlock()
	// Update the current and pending primary sources, removing the deleted
	// connection.
	if found {
		sources := map[string]int{}
		for path, source := range ws.UserPrimarySources {
			if source != e.ID {
				sources[path] = source
			}
		}
		var pendingSources map[string]int
		if ws.AlterUserSchema.ID != nil {
			pendingSources = map[string]int{}
			for path, source := range ws.AlterUserSchema.PrimarySources {
				if source != e.ID {
					pendingSources[path] = source
				}
			}
		}
		state.replaceWorkspace(ws.ID, func(ws *Workspace) {
			ws.UserPrimarySources = sources
			ws.AlterUserSchema.PrimarySources = pendingSources
		})
	}
	// Update the actions.
	state.mu.Lock()
	for _, a := range e.connection.actions {
		delete(state.actions, a.ID)
	}
	state.mu.Unlock()
	// Remove the connection from the linked connections.
	for _, lc := range e.connection.LinkedConnections {
		state.replaceConnection(lc, func(lc *Connection) {
			lc.LinkedConnections = removeLinkedConnection(lc.LinkedConnections, e.ID)
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
	Action int
	Health Health
}

// endActionExecution ends an action execution in progress.
func (state *State) endActionExecution(n notification) {
	e := EndActionExecution{}
	if !decodeNotification(n, &e) {
		return
	}
	a := state.actions[e.Action]
	ws := a.connection.workspace
	ws.mu.Lock()
	delete(ws.executions, e.ID)
	ws.mu.Unlock()
	state.replaceAction(a.ID, func(a *Action) {
		a.execution = nil
		a.Health = e.Health
	})
}

// EndAlterUserSchema is the event sent when the alter of a user schema ends.
type EndAlterUserSchema struct {
	Workspace   int
	ID          string
	EndTime     time.Time
	Err         string
	Schema      types.Type
	Identifiers []string
}

// endAlterUserSchema ends the alter of the user schema.
func (state *State) endAlterUserSchema(n notification) {
	e := EndAlterUserSchema{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		if e.Err == "" {
			// These fields should be updated only in case of success,
			// otherwise, in case of error, the current ones should be left.
			w.UserSchema = w.AlterUserSchema.Schema
			w.UserPrimarySources = w.AlterUserSchema.PrimarySources
			w.Identifiers = e.Identifiers
		}
		w.AlterUserSchema.ID = nil
		w.AlterUserSchema.EndTime = &e.EndTime
		w.AlterUserSchema.Err = &e.Err
		w.AlterUserSchema.Schema = types.Type{}
		w.AlterUserSchema.PrimarySources = nil
		w.AlterUserSchema.Operations = nil
	})
	dispatchNotification(state, e)
}

// EndIdentityResolution is the event sent when the execution of the Identity
// Resolution ends.
type EndIdentityResolution struct {
	Workspace int
	ID        string
	EndTime   time.Time
}

// endIdentityResolution ends the Identity Resolution.
func (state *State) endIdentityResolution(n notification) {
	e := EndIdentityResolution{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.IR.ID = nil
		w.IR.EndTime = &e.EndTime
	})
	dispatchNotification(state, e)
}

// ExecuteAction is the event sent when an action is executed.
type ExecuteAction struct {
	ID          int
	Action      int
	Incremental bool
	Cursor      time.Time
	StartTime   time.Time
}

// executeAction executes an action.
func (state *State) executeAction(n notification) {
	e := ExecuteAction{}
	if !decodeNotification(n, &e) {
		return
	}
	a := state.actions[e.Action]
	ws := a.connection.workspace
	exe := &ActionExecution{
		mu:          &sync.Mutex{},
		ID:          e.ID,
		action:      a,
		Incremental: e.Incremental,
		Cursor:      e.Cursor,
		StartTime:   e.StartTime,
	}
	ws.mu.Lock()
	ws.executions[exe.ID] = exe
	ws.mu.Unlock()
	a.mu.Lock()
	a.execution = exe
	a.mu.Unlock()
	dispatchNotification(state, e)
}

// LinkConnection is the event sent when two unlinked connections are linked.
type LinkConnection struct {
	Connections [2]int
}

// linkConnection links two unlinked connections.
func (state *State) linkConnection(n notification) {
	e := LinkConnection{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceConnection(e.Connections[0], func(c *Connection) {
		c.LinkedConnections = addLinkedConnection(c.LinkedConnections, e.Connections[1])
	})
	state.replaceConnection(e.Connections[1], func(c *Connection) {
		c.LinkedConnections = addLinkedConnection(c.LinkedConnections, e.Connections[0])
	})
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
	Workspace    int
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
	ws := state.workspaces[e.Workspace]
	ws.replaceAccount(e.ID, func(a *Account) {
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

// StartAlterUserSchema is the event sent when the alter of the user schema
// starts.
type StartAlterUserSchema struct {
	Workspace      int
	ID             string
	Schema         types.Type
	PrimarySources map[string]int // always != nil.
	Operations     []meergo.AlterOperation
	StartTime      time.Time
}

// startAlterUserSchema starts the alter of the user schema.
func (state *State) startAlterUserSchema(n notification) {
	e := StartAlterUserSchema{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.AlterUserSchema.ID = &e.ID
		w.AlterUserSchema.Schema = e.Schema
		w.AlterUserSchema.PrimarySources = e.PrimarySources
		w.AlterUserSchema.Operations = e.Operations
		w.AlterUserSchema.StartTime = &e.StartTime
		w.AlterUserSchema.EndTime = nil
		w.AlterUserSchema.Err = nil
	})
	dispatchNotification(state, e)
}

// StartIdentityResolution is the event sent when the execution of the Identity
// Resolution starts.
type StartIdentityResolution struct {
	Workspace int
	ID        string
	StartTime time.Time
}

// startIdentityResolution starts the Identity Resolution.
func (state *State) startIdentityResolution(n notification) {
	e := StartIdentityResolution{}
	if !decodeNotification(n, &e) {
		return
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.IR.ID = &e.ID
		w.IR.StartTime = &e.StartTime
		w.IR.EndTime = nil
	})
	dispatchNotification(state, e)
}

// UpdateAction is the event sent when an action is updated.
type UpdateAction struct {
	ID                   int
	Name                 string
	Enabled              bool
	InSchema             types.Type
	OutSchema            types.Type
	Filter               json.RawMessage
	Transformation       Transformation
	Query                string
	Format               string
	Path                 string
	Sheet                string
	Compression          Compression
	OrderBy              string
	FormatSettings       []byte
	ExportMode           ExportMode
	Matching             Matching
	UpdateOnDuplicates   bool
	TableName            string
	TableKey             string
	IdentityColumn       string
	LastChangeTimeColumn string
	LastChangeTimeFormat string
	Incremental          bool
	PropertiesToUnset    []string
}

// updateAction updates an action.
func (state *State) updateAction(n notification) {
	e := UpdateAction{}
	if !decodeNotification(n, &e) {
		return
	}
	// json.RawMessage(nil) is marshaled into "null", but when it is
	// deserialized it becomes json.RawMessage("null"), so this code converts it
	// back to json.RawMessage(nil).
	if _json.Value(e.Filter).IsNull() {
		e.Filter = nil
	}
	format := state.connectors[e.Format]
	var filter *Where
	if e.Filter != nil {
		filter, _ = unmarshalWhere(e.Filter, e.InSchema)
	}
	state.replaceAction(e.ID, func(a *Action) {
		a.format = format
		a.propertiesToUnset = e.PropertiesToUnset
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
		a.OrderBy = e.OrderBy
		a.FormatSettings = e.FormatSettings
		a.ExportMode = e.ExportMode
		a.Matching = e.Matching
		a.UpdateOnDuplicates = e.UpdateOnDuplicates
		a.TableName = e.TableName
		a.TableKey = e.TableKey
		a.IdentityColumn = e.IdentityColumn
		a.LastChangeTimeColumn = e.LastChangeTimeColumn
		a.LastChangeTimeFormat = e.LastChangeTimeFormat
		a.Incremental = e.Incremental
	})
	dispatchNotification(state, e)
}

// UpdateConnection is the event sent when a connection is updated.
type UpdateConnection struct {
	Connection  int
	Name        string
	Strategy    *Strategy
	SendingMode *SendingMode
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
	})
	dispatchNotification(state, e)
}

// UpdateIdentityPropertiesToUnset is the event sent when the identity
// properties to unset of an action are updated.
type UpdateIdentityPropertiesToUnset struct {
	Action     int
	Properties []string // Always non-nil.
}

// updateIdentityPropertiesToUnset updates the identity properties to unset of
// an action.
func (state *State) updateIdentityPropertiesToUnset(n notification) {
	e := UpdateIdentityPropertiesToUnset{}
	if !decodeNotification(n, &e) {
		return
	}
	a := state.actions[e.Action]
	a.mu.Lock()
	a.propertiesToUnset = e.Properties
	a.mu.Unlock()
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

// UpdateWarehouse is the event sent when a warehouse is updated.
type UpdateWarehouse struct {
	Workspace                    int
	Mode                         WarehouseMode
	Settings                     json.RawMessage
	MCPSettings                  json.RawMessage // it can be a JSON object or json.RawMessage(nil).
	CancelIncompatibleOperations bool
}

// updateWarehouse updates a warehouse.
func (state *State) updateWarehouse(n notification) {
	e := UpdateWarehouse{}
	if !decodeNotification(n, &e) {
		return
	}
	// json.RawMessage(nil) is marshaled into "null", but when it is
	// deserialized it becomes json.RawMessage("null"), so this code converts it
	// back to json.RawMessage(nil).
	if _json.Value(e.MCPSettings).IsNull() {
		e.MCPSettings = nil
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.Warehouse.Mode = e.Mode
		w.Warehouse.Settings = e.Settings
		w.Warehouse.MCPSettings = e.MCPSettings
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
	state.replaceConnection(e.Connections[0], func(c *Connection) {
		c.LinkedConnections = removeLinkedConnection(c.LinkedConnections, e.Connections[1])
	})
	state.replaceConnection(e.Connections[1], func(c *Connection) {
		c.LinkedConnections = removeLinkedConnection(c.LinkedConnections, e.Connections[0])
	})
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
// connections, it returns an empty slice.
func removeLinkedConnection(connections []int, id int) []int {
	if len(connections) == 1 {
		return []int{}
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
