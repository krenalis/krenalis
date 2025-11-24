// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	_json "github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/warehouses"

	"github.com/google/uuid"
	"github.com/meergo/analytics-go"
)

const logNotifications = false // Set to true to enable logging of received notifications.

// keep keeps the state updated and in sync with the database.
// It is called in its own goroutine.
func (state *State) keep() {

	// If sending statistics is enabled, initialize the Meergo analytics client.
	var client analytics.Client
	if state.sendStats {
		client, _ = analytics.NewWithConfig("eEC2uyWaJ1XmFNEq0dkH0a872GzZChUV", analytics.Config{
			Endpoint: "https://telemetry.meergo.com/events",
			Logger:   discardLogger{},
		})
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
		var org uuid.UUID
		state.changing.Lock()
		switch n.Name {
		case "AddMember":
			org = state.addMember(n)
		case "CreateAccessKey":
			org = state.createAccessKey(n)
		case "CreateAction":
			org = state.createAction(n)
		case "CreateConnection":
			org = state.createConnection(n)
		case "CreateWorkspace":
			org = state.createWorkspace(n)
		case "CreateEventWriteKey":
			org = state.createEventWriteKey(n)
		case "DeleteAccessKey":
			org = state.deleteAccessKey(n)
		case "DeleteAction":
			org = state.deleteAction(n)
		case "DeleteConnection":
			org = state.deleteConnection(n)
		case "DeleteEventWriteKey":
			org = state.deleteEventWriteKey(n)
		case "DeleteMember":
			org = state.deleteMember(n)
		case "DeleteWorkspace":
			org = state.deleteWorkspace(n)
		case "ElectLeader":
			state.electLeader(n)
		case "EndActionExecution":
			org = state.endActionExecution(n)
		case "EndAlterProfileSchema":
			org = state.endAlterProfileSchema(n)
		case "EndIdentityResolution":
			org = state.endIdentityResolution(n)
		case "ExecuteAction":
			org = state.executeAction(n)
		case "LinkConnection":
			org = state.linkConnection(n)
		case "PurgeActions":
			org = state.purgeActions(n)
		case "RenameConnection":
			org = state.renameConnection(n)
		case "RenameWorkspace":
			org = state.renameWorkspace(n)
		case "SeeLeader":
			state.seeLeader(n)
		case "SetAccount":
			org = state.setAccount(n)
		case "SetActionFormatSettings":
			org = state.setActionFormatSettings(n)
		case "SetActionSchedulePeriod":
			org = state.setActionSchedulePeriod(n)
		case "SetActionStatus":
			org = state.setActionStatus(n)
		case "SetConnectionSettings":
			org = state.setConnectionSettings(n)
		case "StartAlterProfileSchema":
			org = state.startAlterProfileSchema(n)
		case "StartIdentityResolution":
			org = state.startIdentityResolution(n)
		case "UpdateAction":
			org = state.updateAction(n)
		case "UpdateConnection":
			org = state.updateConnection(n)
		case "UpdateIdentityPropertiesToUnset":
			org = state.updateIdentityPropertiesToUnset(n)
		case "UpdateIdentityResolutionSettings":
			org = state.updateIdentityResolutionSettings(n)
		case "UpdateWarehouse":
			org = state.updateWarehouse(n)
		case "UpdateWarehouseMode":
			org = state.updateWarehouseMode(n)
		case "UpdateWorkspace":
			org = state.updateWorkspace(n)
		case "UnlinkConnection":
			org = state.unlinkConnection(n)
		default:
			slog.Warn("core/internal/state: unknown notification", "id", n.ID, "name", n.Name, "payload", n.Payload)
		}
		state.changing.Unlock()
		if n.ID > 0 {
			// Acknowledge that the notification has been received.
			if ack, ok := state.notifications.acks.LoadAndDelete(n.ID); ok {
				ack.(chan struct{}) <- struct{}{}
			}
		}
		if state.sendStats && org != uuid.Nil {
			state.sendNotificationStats(client, org, n)
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

// AddMember is the event sent when a member is added.
type AddMember struct {
	ID           int
	Organization uuid.UUID
}

// addMember adds a member.
func (state *State) addMember(n notification) uuid.UUID {
	e := AddMember{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	state.mu.Lock()
	org := state.organizations[e.Organization]
	state.mu.Unlock()
	org.mu.Lock()
	org.members[e.ID] = struct{}{}
	org.mu.Unlock()
	return org.ID
}

// CreateAccessKey is the event sent when an access key is created.
type CreateAccessKey struct {
	ID           int
	Organization uuid.UUID
	Workspace    int
	Type         AccessKeyType
	Token        string
}

// createAccessKey creates an access key.
func (state *State) createAccessKey(n notification) uuid.UUID {
	e := CreateAccessKey{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
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
	return e.Organization
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
func (state *State) createAction(n notification) uuid.UUID {
	e := CreateAction{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
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

	return c.organization.ID
}

// CreateConnection is the event sent when a new connection is created.
type CreateConnection struct {
	Workspace int      // workspace identifier
	ID        int      // identifier
	Name      string   // name
	Connector string   // connector
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
func (state *State) createConnection(n notification) uuid.UUID {
	e := CreateConnection{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
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
	return ws.organization.ID
}

// CreateWorkspace is the event sent when a workspace is created.
type CreateWorkspace struct {
	ID                             int
	Organization                   uuid.UUID
	Name                           string
	ProfileSchema                  types.Type
	ResolveIdentitiesOnBatchImport bool
	Warehouse                      struct {
		Name        string
		Mode        WarehouseMode
		Settings    json.RawMessage
		MCPSettings json.RawMessage
	}
	UIPreferences UIPreferences
}

// createWorkspace creates a workspace.
func (state *State) createWorkspace(n notification) uuid.UUID {
	e := CreateWorkspace{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
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
		ProfileSchema:                  e.ProfileSchema,
		PrimarySources:                 map[string]int{},
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
	return organization.ID
}

// CreateEventWriteKey is the event sent when an event write key is created.
type CreateEventWriteKey struct {
	Connection int
	Key        string
	CreatedAt  time.Time
}

// createEventWriteKey creates an event write key.
func (state *State) createEventWriteKey(n notification) uuid.UUID {
	e := CreateEventWriteKey{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
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
	return c.organization.ID
}

// DeleteAccessKey is the event sent when an access key is deleted.
type DeleteAccessKey struct {
	ID int
}

// deleteAccessKey deletes an access key.
func (state *State) deleteAccessKey(n notification) uuid.UUID {
	e := DeleteAccessKey{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	var org uuid.UUID
	state.mu.Lock()
	for token, key := range state.accessKeyByToken {
		if key.ID == e.ID {
			delete(state.accessKeyByToken, token)
			org = key.Organization
			break
		}
	}
	state.mu.Unlock()
	return org
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
func (state *State) deleteAction(n notification) uuid.UUID {
	e := DeleteAction{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
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
	return ws.organization.ID
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
func (state *State) deleteConnection(n notification) uuid.UUID {
	e := DeleteConnection{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
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
	actionsToPurge := ws.actionsToPurge
	if e.connection.Role == Source {
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
	for _, source := range ws.PrimarySources {
		if source == e.ID {
			found = true
			break
		}
	}
	for _, source := range ws.AlterProfileSchema.PrimarySources {
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
		for path, source := range ws.PrimarySources {
			if source != e.ID {
				sources[path] = source
			}
		}
		var pendingSources map[string]int
		if ws.AlterProfileSchema.ID != nil {
			pendingSources = map[string]int{}
			for path, source := range ws.AlterProfileSchema.PrimarySources {
				if source != e.ID {
					pendingSources[path] = source
				}
			}
		}
		state.replaceWorkspace(ws.ID, func(ws *Workspace) {
			ws.PrimarySources = sources
			ws.AlterProfileSchema.PrimarySources = pendingSources
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
	return ws.organization.ID
}

// DeleteEventWriteKey is the event sent when an event write key is deleted.
type DeleteEventWriteKey struct {
	Connection int
	Key        string
}

// deleteEventWriteKey deletes an event write key.
func (state *State) deleteEventWriteKey(n notification) uuid.UUID {
	e := DeleteEventWriteKey{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	c := state.replaceConnection(e.Connection, func(c *Connection) {
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
	return c.organization.ID
}

// DeleteMember is the event sent when a member is deleted.
type DeleteMember struct {
	ID           int
	Organization uuid.UUID
}

// deleteMember deletes a member.
func (state *State) deleteMember(n notification) uuid.UUID {
	e := DeleteMember{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	state.mu.Lock()
	org := state.organizations[e.Organization]
	state.mu.Unlock()
	org.mu.Lock()
	delete(org.members, e.ID)
	org.mu.Unlock()
	return e.Organization
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
func (state *State) deleteWorkspace(n notification) uuid.UUID {
	e := DeleteWorkspace{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
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
	return organization.ID
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
func (state *State) endActionExecution(n notification) uuid.UUID {
	e := EndActionExecution{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
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
	return ws.organization.ID
}

// EndAlterProfileSchema is the event sent when the alter of a profile schema
// ends.
type EndAlterProfileSchema struct {
	Workspace   int
	ID          string
	EndTime     time.Time
	Err         string
	Schema      types.Type
	Identifiers []string
}

// endAlterProfileSchema ends the alter of the profile schema.
func (state *State) endAlterProfileSchema(n notification) uuid.UUID {
	e := EndAlterProfileSchema{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	ws := state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		if e.Err == "" {
			// These fields should be updated only in case of success,
			// otherwise, in case of error, the current ones should be left.
			w.ProfileSchema = w.AlterProfileSchema.Schema
			w.PrimarySources = w.AlterProfileSchema.PrimarySources
			w.Identifiers = e.Identifiers
		}
		w.AlterProfileSchema.ID = nil
		w.AlterProfileSchema.EndTime = &e.EndTime
		w.AlterProfileSchema.Err = &e.Err
		w.AlterProfileSchema.Schema = types.Type{}
		w.AlterProfileSchema.PrimarySources = nil
		w.AlterProfileSchema.Operations = nil
	})
	dispatchNotification(state, e)
	return ws.organization.ID
}

// EndIdentityResolution is the event sent when the execution of the Identity
// Resolution ends.
type EndIdentityResolution struct {
	Workspace int
	ID        string
	EndTime   time.Time
}

// endIdentityResolution ends the Identity Resolution.
func (state *State) endIdentityResolution(n notification) uuid.UUID {
	e := EndIdentityResolution{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	ws := state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.IR.ID = nil
		w.IR.EndTime = &e.EndTime
	})
	dispatchNotification(state, e)
	return ws.organization.ID
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
func (state *State) executeAction(n notification) uuid.UUID {
	e := ExecuteAction{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
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
	return ws.organization.ID
}

// LinkConnection is the event sent when two unlinked connections are linked.
type LinkConnection struct {
	Connections [2]int
}

// linkConnection links two unlinked connections.
func (state *State) linkConnection(n notification) uuid.UUID {
	e := LinkConnection{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	state.replaceConnection(e.Connections[0], func(c *Connection) {
		c.LinkedConnections = addLinkedConnection(c.LinkedConnections, e.Connections[1])
	})
	c := state.replaceConnection(e.Connections[1], func(c *Connection) {
		c.LinkedConnections = addLinkedConnection(c.LinkedConnections, e.Connections[0])
	})
	return c.organization.ID
}

// PurgeActions is the event sent when actions of a workspace are purged.
type PurgeActions struct {
	Workspace      int
	ActionsToPurge []int // remaining actions to purge. Never nil.
}

// purgeActions purges actions of a workspace.
func (state *State) purgeActions(n notification) uuid.UUID {
	e := PurgeActions{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	ws, _ := state.Workspace(e.Workspace)
	ws.mu.Lock()
	ws.actionsToPurge = e.ActionsToPurge
	ws.mu.Unlock()
	return ws.organization.ID
}

// RenameConnection is the event sent when a connection is renamed.
type RenameConnection struct {
	Connection int
	Name       string
}

// renameConnection renames a connection.
func (state *State) renameConnection(n notification) uuid.UUID {
	e := RenameConnection{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	c := state.replaceConnection(e.Connection, func(c *Connection) {
		c.Name = e.Name
	})
	return c.organization.ID
}

// RenameWorkspace is the event sent when a workspace is renamed.
type RenameWorkspace struct {
	Workspace int
	Name      string
}

// renameWorkspace renames a workspace.
func (state *State) renameWorkspace(n notification) uuid.UUID {
	e := RenameWorkspace{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	ws := state.replaceWorkspace(e.Workspace, func(ws *Workspace) {
		ws.Name = e.Name
	})
	return ws.organization.ID
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
func (state *State) setAccount(n notification) uuid.UUID {
	e := SetAccount{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	ws := state.workspaces[e.Workspace]
	ws.replaceAccount(e.ID, func(a *Account) {
		a.AccessToken = e.AccessToken
		a.RefreshToken = e.RefreshToken
		a.ExpiresIn = e.ExpiresIn
	})
	return ws.organization.ID
}

// SetActionFormatSettings is the event sent when the format settings of an
// action are changed.
type SetActionFormatSettings struct {
	Action   int
	Settings []byte
}

// setActionFormatSettings sets the format settings of an action.
func (state *State) setActionFormatSettings(n notification) uuid.UUID {
	e := SetActionFormatSettings{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	a := state.replaceAction(e.Action, func(a *Action) {
		a.FormatSettings = e.Settings
	})
	return a.connection.organization.ID
}

// SetActionSchedulePeriod is the event sent when the schedule period of an
// action is set.
type SetActionSchedulePeriod struct {
	ID             int
	SchedulePeriod int16
}

// setActionSchedulePeriod sets the schedule period of an action.
func (state *State) setActionSchedulePeriod(n notification) uuid.UUID {
	e := SetActionSchedulePeriod{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	a := state.replaceAction(e.ID, func(a *Action) {
		a.SchedulePeriod = e.SchedulePeriod
	})
	dispatchNotification(state, e)
	return a.connection.organization.ID
}

// SetActionStatus is the event sent when the status of an action is set.
type SetActionStatus struct {
	ID      int
	Enabled bool
}

// setActionStatus sets the status of an action.
func (state *State) setActionStatus(n notification) uuid.UUID {
	e := SetActionStatus{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	a := state.replaceAction(e.ID, func(a *Action) {
		a.Enabled = e.Enabled
	})
	dispatchNotification(state, e)
	return a.connection.organization.ID
}

// SetConnectionSettings is the event sent when the settings of a connection is
// changed.
type SetConnectionSettings struct {
	Connection int
	Settings   []byte
}

// setConnectionSettings sets the settings of a connection.
func (state *State) setConnectionSettings(n notification) uuid.UUID {
	e := SetConnectionSettings{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	c := state.replaceConnection(e.Connection, func(c *Connection) {
		c.Settings = e.Settings
	})
	dispatchNotification(state, e)
	return c.organization.ID
}

// StartAlterProfileSchema is the event sent when the alter of the profile
// schema starts.
type StartAlterProfileSchema struct {
	Workspace      int
	ID             string
	Schema         types.Type
	PrimarySources map[string]int // always != nil.
	Operations     []warehouses.AlterOperation
	StartTime      time.Time
}

// startAlterProfileSchema starts the alter of the profile schema.
func (state *State) startAlterProfileSchema(n notification) uuid.UUID {
	e := StartAlterProfileSchema{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	ws := state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.AlterProfileSchema.ID = &e.ID
		w.AlterProfileSchema.Schema = e.Schema
		w.AlterProfileSchema.PrimarySources = e.PrimarySources
		w.AlterProfileSchema.Operations = e.Operations
		w.AlterProfileSchema.StartTime = &e.StartTime
		w.AlterProfileSchema.EndTime = nil
		w.AlterProfileSchema.Err = nil
	})
	dispatchNotification(state, e)
	return ws.organization.ID
}

// StartIdentityResolution is the event sent when the execution of the Identity
// Resolution starts.
type StartIdentityResolution struct {
	Workspace int
	ID        string
	StartTime time.Time
}

// startIdentityResolution starts the Identity Resolution.
func (state *State) startIdentityResolution(n notification) uuid.UUID {
	e := StartIdentityResolution{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	ws := state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.IR.ID = &e.ID
		w.IR.StartTime = &e.StartTime
		w.IR.EndTime = nil
	})
	dispatchNotification(state, e)
	return ws.organization.ID
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
func (state *State) updateAction(n notification) uuid.UUID {
	e := UpdateAction{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
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
	a := state.replaceAction(e.ID, func(a *Action) {
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
	return a.connection.organization.ID
}

// UpdateConnection is the event sent when a connection is updated.
type UpdateConnection struct {
	Connection  int
	Name        string
	Strategy    *Strategy
	SendingMode *SendingMode
}

// updateConnection updates a connection.
func (state *State) updateConnection(n notification) uuid.UUID {
	e := UpdateConnection{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	c := state.replaceConnection(e.Connection, func(c *Connection) {
		c.Name = e.Name
		c.Strategy = e.Strategy
		c.SendingMode = e.SendingMode
	})
	dispatchNotification(state, e)
	return c.organization.ID
}

// UpdateIdentityPropertiesToUnset is the event sent when the identity
// properties to unset of an action are updated.
type UpdateIdentityPropertiesToUnset struct {
	Action     int
	Properties []string // Always non-nil.
}

// updateIdentityPropertiesToUnset updates the identity properties to unset of
// an action.
func (state *State) updateIdentityPropertiesToUnset(n notification) uuid.UUID {
	e := UpdateIdentityPropertiesToUnset{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	a := state.actions[e.Action]
	a.mu.Lock()
	a.propertiesToUnset = e.Properties
	a.mu.Unlock()
	return a.connection.organization.ID
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
func (state *State) updateIdentityResolutionSettings(n notification) uuid.UUID {
	e := UpdateIdentityResolutionSettings{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	ws := state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.ResolveIdentitiesOnBatchImport = e.ResolveIdentitiesOnBatchImport
		w.Identifiers = e.Identifiers
	})
	return ws.organization.ID
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
func (state *State) updateWarehouse(n notification) uuid.UUID {
	e := UpdateWarehouse{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	// json.RawMessage(nil) is marshaled into "null", but when it is
	// deserialized it becomes json.RawMessage("null"), so this code converts it
	// back to json.RawMessage(nil).
	if _json.Value(e.MCPSettings).IsNull() {
		e.MCPSettings = nil
	}
	ws := state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.Warehouse.Mode = e.Mode
		w.Warehouse.Settings = e.Settings
		w.Warehouse.MCPSettings = e.MCPSettings
	})
	dispatchNotification(state, e)
	return ws.organization.ID
}

// UpdateWarehouseMode is the event sent when the mode of a data warehouse is
// updated.
type UpdateWarehouseMode struct {
	Workspace                    int
	Mode                         WarehouseMode
	CancelIncompatibleOperations bool
}

// updateWarehouseMode updates the mode of a data warehouse.
func (state *State) updateWarehouseMode(n notification) uuid.UUID {
	e := UpdateWarehouseMode{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	ws := state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.Warehouse.Mode = e.Mode
	})
	dispatchNotification(state, e)
	return ws.organization.ID
}

// UpdateWorkspace is the event sent when the name and the displayed properties
// of a workspace are updated.
type UpdateWorkspace struct {
	Workspace     int
	Name          string
	UIPreferences UIPreferences
}

// updateWorkspace updates the name and the displayed properties of a workspace.
func (state *State) updateWorkspace(n notification) uuid.UUID {
	e := UpdateWorkspace{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	ws := state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.Name = e.Name
		w.UIPreferences = e.UIPreferences
	})
	dispatchNotification(state, e)
	return ws.organization.ID
}

// UnlinkConnection is the event sent when two linked connections are unlinked.
type UnlinkConnection struct {
	Connections [2]int
}

// unlinkConnection unlinks two linked connections.
func (state *State) unlinkConnection(n notification) uuid.UUID {
	e := UnlinkConnection{}
	if !decodeNotification(n, &e) {
		return uuid.Nil
	}
	state.replaceConnection(e.Connections[0], func(c *Connection) {
		c.LinkedConnections = removeLinkedConnection(c.LinkedConnections, e.Connections[1])
	})
	c := state.replaceConnection(e.Connections[1], func(c *Connection) {
		c.LinkedConnections = removeLinkedConnection(c.LinkedConnections, e.Connections[0])
	})
	return c.organization.ID
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
