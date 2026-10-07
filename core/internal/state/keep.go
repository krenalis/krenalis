// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"bytes"
	stdjson "encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"

	"github.com/krenalis/analytics-go"
)

const logNotifications = false // Set to true to enable logging of received notifications.

// applyNotification applies and dispatches a reconstructed trusted event,
// then publishes the event's version and wakes version waiters.
// The caller is responsible for ordering and deduplication. This method takes
// no context because applying an event must run to completion once started.
// A panic leaves the State unusable.
func (state *State) applyNotification(n notification, client analytics.Client) error {

	if logNotifications {
		slog.Info("core/state: received notification", "version", n.Version, "name", n.Name, "payload", n.Payload)
	}

	var org string
	state.changing.Lock()
	// Multiple goroutines may read different parts of the state concurrently, but
	// only this goroutine can write to it. Therefore, this goroutine can read the
	// state without acquiring the corresponding locks.
	switch n.Name {
	case "AcceptInvitation":
		org = state.acceptInvitation(n)
	case "AddConsentPurpose":
		org = state.addConsentPurpose(n)
	case "AddMember":
		org = state.addMember(n)
	case "CreateAccessKey":
		org = state.createAccessKey(n)
	case "CreateConnection":
		org = state.createConnection(n)
	case "CreateOrganization":
		org = state.createOrganization(n)
	case "CreatePipeline":
		org = state.createPipeline(n)
	case "CreateWorkspace":
		org = state.createWorkspace(n)
	case "CreateEventWriteKey":
		org = state.createEventWriteKey(n)
	case "DeleteAccessKey":
		org = state.deleteAccessKey(n)
	case "DeleteConnection":
		org = state.deleteConnection(n)
	case "DeleteConsentPurpose":
		org = state.deleteConsentPurpose(n)
	case "DeleteEventWriteKey":
		org = state.deleteEventWriteKey(n)
	case "DeleteMember":
		org = state.deleteMember(n)
	case "DeleteMembers":
		org = state.deleteMembers(n)
	case "DeleteOrganization":
		org = state.deleteOrganization(n)
	case "DeletePipeline":
		org = state.deletePipeline(n)
	case "DeleteWorkspace":
		org = state.deleteWorkspace(n)
	case "ElectLeader":
		state.electLeader(n)
	case "EndAlterProfileSchema":
		org = state.endAlterProfileSchema(n)
	case "EndIdentityResolution":
		org = state.endIdentityResolution(n)
	case "EndPipelineRun":
		org = state.endPipelineRun(n)
	case "InviteMember":
		org = state.inviteMember(n)
	case "LinkConnection":
		org = state.linkConnection(n)
	case "PurgePipelines":
		org = state.purgePipelines(n)
	case "RenameConnection":
		org = state.renameConnection(n)
	case "RenameWorkspace":
		org = state.renameWorkspace(n)
	case "RunPipeline":
		org = state.runPipeline(n)
	case "SeeLeader":
		state.seeLeader(n)
	case "SetAccount":
		org = state.setAccount(n)
	case "SetConnectionSettings":
		org = state.setConnectionSettings(n)
	case "SetOrganizationStatus":
		org = state.setOrganizationStatus(n)
	case "SetPipelineFormatSettings":
		org = state.setPipelineFormatSettings(n)
	case "SetPipelineSchedulePeriod":
		org = state.setPipelineSchedulePeriod(n)
	case "SetPipelineStatus":
		org = state.setPipelineStatus(n)
	case "StartAlterProfileSchema":
		org = state.startAlterProfileSchema(n)
	case "StartIdentityResolution":
		org = state.startIdentityResolution(n)
	case "UnlinkConnection":
		org = state.unlinkConnection(n)
	case "UpdateConnection":
		org = state.updateConnection(n)
	case "UpdateConsentPurpose":
		org = state.updateConsentPurpose(n)
	case "UpdateIdentityPropertiesToUnset":
		org = state.updateIdentityPropertiesToUnset(n)
	case "UpdateIdentityResolutionSettings":
		org = state.updateIdentityResolutionSettings(n)
	case "UpdateOrganization":
		org = state.updateOrganization(n)
	case "UpdatePipeline":
		org = state.updatePipeline(n)
	case "UpdateWarehouse":
		org = state.updateWarehouse(n)
	case "UpdateWarehouseMode":
		org = state.updateWarehouseMode(n)
	case "UpdateWorkspace":
		org = state.updateWorkspace(n)
	default:
		state.changing.Unlock()
		return &replicationError{message: fmt.Sprintf("unknown notification (version %d)", n.Version)}
	}

	// Notify any goroutines waiting for a new version.
	if n.Version > 0 {
		state.version.Lock()
		state.version.current = n.Version
		state.version.next.Broadcast()
		state.version.Unlock()
	}

	state.changing.Unlock()

	if client != nil && org != "" {
		state.sendNotificationStats(client, org, n)
	}

	return nil
}

// decodeNotification decodes a trusted notification and panics if its payload
// is not formally compatible with the expected event type.
func decodeNotification(n notification, e any) {
	err := json.Unmarshal([]byte(n.Payload), e)
	if err != nil {
		panic(fmt.Sprintf("invalid notification payload %s (version %d)", n.Name, n.Version))
	}
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

// replaceConsentPurpose calls f with a copy of the consent purpose, replaces
// the purpose in the workspace, and returns the copy.
func (workspace *Workspace) replaceConsentPurpose(id string, f func(*ConsentPurpose)) *ConsentPurpose {
	c := workspace.consentPurposes[id]
	cc := new(ConsentPurpose)
	*cc = *c
	f(cc)
	workspace.mu.Lock()
	workspace.consentPurposes[id] = cc
	workspace.mu.Unlock()
	return cc
}

// replacePipeline calls the function f passing a copy of the pipeline with
// identifier id. After f is returned, it replaces the pipeline with its copy in
// the state and returns the latter.
func (state *State) replacePipeline(id string, f func(*Pipeline)) *Pipeline {
	p := state.pipelines[id]
	pp := new(Pipeline)
	*pp = *p
	f(pp)
	state.mu.Lock()
	state.pipelines[id] = pp
	state.mu.Unlock()
	// Update the connection.
	c := p.connection
	c.mu.Lock()
	c.pipelines[id] = pp
	c.mu.Unlock()
	return pp
}

// replaceConnection calls the function f passing a copy of the connection with
// identifier id. After f is returned, it replaces the connection with its
// copy in the state and returns the latter.
func (state *State) replaceConnection(id string, f func(*Connection)) *Connection {
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
	// Update the pipelines.
	for _, pipeline := range c.pipelines {
		pipeline.mu.Lock()
		pipeline.connection = cc
		pipeline.mu.Unlock()
	}
	return cc
}

// replaceWorkspace calls the function f passing a copy of the workspace with
// identifier id. After f is returned, it replaces the workspace with its
// copy in the state and returns the latter.
func (state *State) replaceWorkspace(id string, f func(*Workspace)) *Workspace {
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

// replaceOrganization calls the function f passing a copy of the organization
// with identifier id. After f returns, it replaces the organization with its
// copy in the state and updates all workspace back-pointers. Returns the copy
// of the organization.
func (state *State) replaceOrganization(id string, f func(*Organization)) *Organization {
	o := state.organizations[id]
	oo := new(Organization)
	*oo = *o
	f(oo)
	state.mu.Lock()
	state.organizations[id] = oo
	state.mu.Unlock()
	for _, ws := range oo.workspaces {
		ws.mu.Lock()
		ws.organization = oo
		ws.mu.Unlock()
		for _, connection := range ws.connections {
			connection.mu.Lock()
			connection.organization = oo
			connection.mu.Unlock()
			for _, pipeline := range connection.pipelines {
				pipeline.mu.Lock()
				pipeline.organization = oo
				pipeline.mu.Unlock()
			}
		}
	}
	return oo
}

// AcceptInvitation is the event sent when a member accept an invitation.
type AcceptInvitation struct {
	Member       string
	Organization string
}

// acceptInvitation accepts a member invitation.
func (state *State) acceptInvitation(n notification) string {
	e := AcceptInvitation{}
	decodeNotification(n, &e)
	org := state.organizations[e.Organization]
	org.mu.Lock()
	org.members[e.Member] = true
	org.mu.Unlock()
	return org.ID
}

// AddConsentPurpose is the event sent when a new consent purpose is added.
type AddConsentPurpose struct {
	Workspace              string
	ID                     string
	Name                   string
	EventConsentLocations  []EventConsentLocation
	ProfileConsentLocation *ProfileConsentLocation
}

// addConsentPurpose adds a new consent purpose.
func (state *State) addConsentPurpose(n notification) string {
	e := AddConsentPurpose{}
	decodeNotification(n, &e)
	cp := &ConsentPurpose{
		ID:                     e.ID,
		Name:                   e.Name,
		EventConsentLocations:  e.EventConsentLocations,
		ProfileConsentLocation: e.ProfileConsentLocation,
	}
	ws := state.workspaces[e.Workspace]
	ws.mu.Lock()
	ws.consentPurposes[cp.ID] = cp
	ws.mu.Unlock()
	dispatchNotification(state, e)
	return ws.organization.ID
}

// AddMember is the event sent when a member is added.
type AddMember struct {
	ID           string
	Organization string
}

// addMember adds a member.
func (state *State) addMember(n notification) string {
	e := AddMember{}
	decodeNotification(n, &e)
	org := state.organizations[e.Organization]
	org.mu.Lock()
	org.members[e.ID] = true
	org.usage.addMember()
	org.mu.Unlock()
	return org.ID
}

// CreateAccessKey is the event sent when an access key is created.
type CreateAccessKey struct {
	ID           string
	Organization string
	Workspace    string
	Type         AccessKeyType
	HMAC         []byte
}

// createAccessKey creates an access key.
func (state *State) createAccessKey(n notification) string {
	e := CreateAccessKey{}
	decodeNotification(n, &e)
	key := AccessKey{
		ID:           e.ID,
		Organization: e.Organization,
		Workspace:    e.Workspace,
		Type:         e.Type,
	}
	state.mu.Lock()
	state.accessKeyByHMAC[string(e.HMAC)] = &key
	state.mu.Unlock()
	org := state.organizations[e.Organization]
	org.mu.Lock()
	org.usage.addAccessKey()
	org.mu.Unlock()
	return e.Organization
}

// CreateConnection is the event sent when a new connection is created.
type CreateConnection struct {
	Workspace string   // workspace identifier
	ID        string   // identifier
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
	LinkedConnections []string     // linked connections
	EventWriteKey     string       // event write key to add
	Settings          []byte
	SettingsKey       []byte
}

// createConnection creates a new connection.
func (state *State) createConnection(n notification) string {
	e := CreateConnection{}
	decodeNotification(n, &e)
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
		settings:          e.Settings,
		settingsKey:       state.cipher.Key(e.SettingsKey),
		pipelines:         map[string]*Pipeline{},
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
	// Update the organization.
	org := c.organization
	org.mu.Lock()
	org.usage.addConnection(c.connector)
	org.mu.Unlock()
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

// CreateOrganization is the event sent when an organization is created.
type CreateOrganization struct {
	ID      string
	Name    string
	Enabled bool
	Limits  OrganizationLimits
}

// createOrganization creates an organization.
func (state *State) createOrganization(n notification) string {
	e := CreateOrganization{}
	decodeNotification(n, &e)
	org := &Organization{
		mu:         &sync.Mutex{},
		bucket:     state.rateLimiter.NewBucket("organization", e.ID, requestLeaseSize, requestMaxUnits),
		workspaces: map[string]*Workspace{},
		members:    map[string]bool{},
		usage:      newOrganizationUsage(e.Limits),
		ID:         e.ID,
		Name:       e.Name,
		Enabled:    e.Enabled,
	}
	state.mu.Lock()
	state.organizations[e.ID] = org
	state.mu.Unlock()
	dispatchNotification(state, e)
	return e.ID
}

// CreatePipeline is the event sent when a pipeline is created.
type CreatePipeline struct {
	ID                 string
	Connection         string
	Target             Target
	EventType          string
	OrderingGroup      string
	DeliveryEndpoint   string
	Name               string
	Enabled            bool
	ScheduleStart      int16
	SchedulePeriod     int16
	InSchema           types.Type
	OutSchema          types.Type
	Filter             stdjson.RawMessage
	RequiredConsents   RequiredConsentsByIDs
	Transformation     Transformation
	Query              string
	Format             string
	Path               string
	Sheet              string
	Compression        Compression
	OrderBy            string
	FormatSettings     json.Value
	ExportMode         ExportMode
	Matching           Matching
	UpdateOnDuplicates bool
	TableName          string
	TableKey           string
	UserIDColumn       string
	UpdatedAtColumn    string
	UpdatedAtFormat    string
	Incremental        bool
}

// createPipeline creates a new pipeline.
func (state *State) createPipeline(n notification) string {
	e := CreatePipeline{}
	decodeNotification(n, &e)
	// json.Value(nil) is marshaled into "null", but when it is
	// deserialized it becomes json.Value("null"), so this code converts it
	// back to json.Value(nil).
	if json.Value(e.Filter).IsNull() {
		e.Filter = nil
	}
	if e.FormatSettings.IsNull() {
		e.FormatSettings = nil
	}
	c := state.connections[e.Connection]
	format := state.connectors[e.Format]
	requiredConsents := RequiredConsents{
		Operator: e.RequiredConsents.Operator,
		Purposes: make([]*ConsentPurpose, len(e.RequiredConsents.Purposes)),
	}
	for i, id := range e.RequiredConsents.Purposes {
		requiredConsents.Purposes[i] = c.workspace.consentPurposes[id]
	}
	pipeline := &Pipeline{
		mu:                 new(sync.Mutex),
		ID:                 e.ID,
		connection:         c,
		organization:       c.organization,
		format:             format,
		Target:             e.Target,
		Name:               e.Name,
		Enabled:            e.Enabled,
		EventType:          e.EventType,
		OrderingGroup:      e.OrderingGroup,
		DeliveryEndpoint:   e.DeliveryEndpoint,
		ScheduleStart:      e.ScheduleStart,
		SchedulePeriod:     e.SchedulePeriod,
		InSchema:           e.InSchema,
		OutSchema:          e.OutSchema,
		RequiredConsents:   requiredConsents,
		Transformation:     e.Transformation,
		Query:              e.Query,
		Path:               e.Path,
		Sheet:              e.Sheet,
		Compression:        e.Compression,
		OrderBy:            e.OrderBy,
		FormatSettings:     e.FormatSettings,
		ExportMode:         e.ExportMode,
		Matching:           e.Matching,
		UpdateOnDuplicates: e.UpdateOnDuplicates,
		TableName:          e.TableName,
		TableKey:           e.TableKey,
		UserIDColumn:       e.UserIDColumn,
		UpdatedAtColumn:    e.UpdatedAtColumn,
		UpdatedAtFormat:    e.UpdatedAtFormat,
		Incremental:        e.Incremental,
	}
	if c.Role == Source && e.Target == TargetUser {
		pipeline.propertiesToUnset = []string{}
	}
	if e.Filter != nil {
		var err error
		pipeline.Filter, err = unmarshalWhere(e.Filter, e.InSchema)
		if err != nil {
			panic(fmt.Sprintf("invalid notification payload %s (version %d)", n.Name, n.Version))
		}
	}

	state.mu.Lock()
	state.pipelines[e.ID] = pipeline
	state.mu.Unlock()
	org := c.organization
	org.mu.Lock()
	org.usage.addPipeline(pipeline.format)
	org.mu.Unlock()
	c.mu.Lock()
	c.pipelines[e.ID] = pipeline
	c.mu.Unlock()
	dispatchNotification(state, e)

	return c.organization.ID
}

// CreateWorkspace is the event sent when a workspace is created.
type CreateWorkspace struct {
	ID                             string
	Organization                   string
	Name                           string
	ProfileSchema                  types.Type
	ResolveIdentitiesOnBatchImport bool
	Warehouse                      struct {
		Platform       string
		Mode           WarehouseMode
		Settings       []byte
		SettingsKey    []byte
		MCPSettingsKey []byte
	}
	UIPreferences UIPreferences
}

// createWorkspace creates a workspace.
func (state *State) createWorkspace(n notification) string {
	e := CreateWorkspace{}
	decodeNotification(n, &e)
	organization := state.organizations[e.Organization]
	ws := Workspace{
		mu:                             &sync.Mutex{},
		bucket:                         state.rateLimiter.NewBucket("workspace", e.ID, requestLeaseSize, requestMaxUnits),
		eventBucket:                    state.rateLimiter.NewBucket("events", e.ID, eventLeaseSize, eventMaxUnits),
		connections:                    map[string]*Connection{},
		ID:                             e.ID,
		organization:                   organization,
		Name:                           e.Name,
		ProfileSchema:                  e.ProfileSchema,
		PrimarySources:                 map[string]string{},
		accounts:                       map[int]*Account{},
		consentPurposes:                map[string]*ConsentPurpose{},
		ResolveIdentitiesOnBatchImport: e.ResolveIdentitiesOnBatchImport,
		Identifiers:                    []string{},
		UIPreferences:                  e.UIPreferences,
		pipelinesToPurge:               []string{},
	}
	ws.Warehouse.Platform = e.Warehouse.Platform
	ws.Warehouse.Mode = e.Warehouse.Mode
	ws.Warehouse.settings = e.Warehouse.Settings
	ws.Warehouse.settingsKey = state.cipher.Key(e.Warehouse.SettingsKey)
	ws.Warehouse.mcpSettingsKey = state.cipher.Key(e.Warehouse.MCPSettingsKey)
	state.mu.Lock()
	state.workspaces[e.ID] = &ws
	state.mu.Unlock()
	organization.mu.Lock()
	organization.workspaces[e.ID] = &ws
	organization.usage.addWorkspace()
	organization.mu.Unlock()
	dispatchNotification(state, e)
	return organization.ID
}

// CreateEventWriteKey is the event sent when an event write key is created.
type CreateEventWriteKey struct {
	Connection string
	Key        string
	CreatedAt  time.Time
}

// createEventWriteKey creates an event write key.
func (state *State) createEventWriteKey(n notification) string {
	e := CreateEventWriteKey{}
	decodeNotification(n, &e)
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
	ID string
}

// deleteAccessKey deletes an access key.
func (state *State) deleteAccessKey(n notification) string {
	e := DeleteAccessKey{}
	decodeNotification(n, &e)
	state.mu.Lock()
	var hmac string
	var key *AccessKey
	for hmac, key = range state.accessKeyByHMAC {
		if key.ID == e.ID {
			break
		}
	}
	delete(state.accessKeyByHMAC, hmac)
	state.mu.Unlock()
	org := state.organizations[key.Organization]
	org.mu.Lock()
	org.usage.removeAccessKey()
	org.mu.Unlock()
	return org.ID
}

// DeleteConnection is the event sent when a connection is deleted.
type DeleteConnection struct {
	ID         string
	Account    bool // indicates whether the associated account was also deleted.
	connection *Connection
}

func (n DeleteConnection) Connection() *Connection {
	return n.connection
}

// deleteConnection deletes a connection.
func (state *State) deleteConnection(n notification) string {
	e := DeleteConnection{}
	decodeNotification(n, &e)
	e.connection = state.connections[e.ID]
	// Update connections and keys.
	state.mu.Lock()
	delete(state.connections, e.ID)
	for _, key := range e.connection.Keys {
		delete(state.connectionsByKey, key)
	}
	state.mu.Unlock()
	// Update the organization.
	org := e.connection.organization
	org.mu.Lock()
	org.usage.removeConnection(e.connection)
	org.mu.Unlock()
	// Update the workspace.
	ws := e.connection.workspace
	if e.Account {
		ws.mu.Lock()
		delete(ws.accounts, e.connection.account.ID)
		ws.mu.Unlock()
	}
	pipelinesToPurge := ws.pipelinesToPurge
	if e.connection.Role == Source {
		for _, pipeline := range e.connection.pipelines {
			if pipeline.Target == TargetUser {
				pipelinesToPurge = append(pipelinesToPurge, pipeline.ID)
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
	ws.pipelinesToPurge = pipelinesToPurge
	ws.mu.Unlock()
	// Update the current and pending primary sources, removing the deleted
	// connection.
	if found {
		sources := map[string]string{}
		for path, source := range ws.PrimarySources {
			if source != e.ID {
				sources[path] = source
			}
		}
		var pendingSources map[string]string
		if ws.AlterProfileSchema.ID != nil {
			pendingSources = map[string]string{}
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
	// Update the pipelines.
	state.mu.Lock()
	for _, p := range e.connection.pipelines {
		delete(state.pipelines, p.ID)
		if p.run != nil {
			delete(state.liveRuns, p.run.ID)
		}
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

// DeleteConsentPurpose is the event sent when a consent purpose is deleted.
type DeleteConsentPurpose struct {
	Workspace string
	ID        string
}

// deleteConsentPurpose deletes a consent purpose.
func (state *State) deleteConsentPurpose(n notification) string {
	e := DeleteConsentPurpose{}
	decodeNotification(n, &e)
	ws := state.workspaces[e.Workspace]
	ws.mu.Lock()
	delete(ws.consentPurposes, e.ID)
	ws.mu.Unlock()
	dispatchNotification(state, e)
	return ws.organization.ID
}

// DeleteEventWriteKey is the event sent when an event write key is deleted.
type DeleteEventWriteKey struct {
	Connection string
	Key        string
}

// deleteEventWriteKey deletes an event write key.
func (state *State) deleteEventWriteKey(n notification) string {
	e := DeleteEventWriteKey{}
	decodeNotification(n, &e)
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
	ID           string
	Organization string
}

// deleteMember deletes a member.
func (state *State) deleteMember(n notification) string {
	e := DeleteMember{}
	decodeNotification(n, &e)
	org := state.organizations[e.Organization]
	org.mu.Lock()
	delete(org.members, e.ID)
	org.usage.removeMember()
	org.mu.Unlock()
	return e.Organization
}

// DeleteMembers is the event sent when multiple members are deleted at once.
type DeleteMembers struct {
	IDs []string
}

// deleteMembers deletes multiple members.
func (state *State) deleteMembers(n notification) string {
	e := DeleteMembers{}
	decodeNotification(n, &e)
	for _, org := range state.organizations {
		org.mu.Lock()
		for _, id := range e.IDs {
			if _, ok := org.members[id]; ok {
				delete(org.members, id)
				org.usage.removeMember()
			}
		}
		org.mu.Unlock()
	}
	return ""
}

// DeleteOrganization is the event sent when an organization is deleted.
type DeleteOrganization struct {
	ID           string
	organization *Organization
}

func (n DeleteOrganization) Organization() *Organization {
	return n.organization
}

// deleteOrganization deletes an organization.
func (state *State) deleteOrganization(n notification) string {
	e := DeleteOrganization{}
	decodeNotification(n, &e)
	state.mu.Lock()
	e.organization = state.organizations[e.ID]
	delete(state.organizations, e.ID)
	// Delete all workspaces belonging to the organization.
	for id, ws := range e.organization.workspaces {
		for _, c := range ws.connections {
			for _, key := range c.Keys {
				delete(state.connectionsByKey, key)
			}
			delete(state.connections, c.ID)
			// Delete the connection's pipelines.
			for _, p := range c.pipelines {
				delete(state.pipelines, p.ID)
				if p.run != nil {
					delete(state.liveRuns, p.run.ID)
				}
			}
		}
		delete(state.workspaces, id)
	}
	// Delete all access keys belonging to the organization.
	for hmac, key := range state.accessKeyByHMAC {
		if key.Organization == e.ID {
			delete(state.accessKeyByHMAC, hmac)
		}
	}
	state.mu.Unlock()
	dispatchNotification(state, e)
	return e.ID
}

// DeletePipeline is the event sent when a pipeline is deleted.
type DeletePipeline struct {
	ID       string
	pipeline *Pipeline
}

func (n DeletePipeline) Pipeline() *Pipeline {
	return n.pipeline
}

// deletePipeline deletes a pipeline.
func (state *State) deletePipeline(n notification) string {
	e := DeletePipeline{}
	decodeNotification(n, &e)
	e.pipeline = state.pipelines[e.ID]
	state.mu.Lock()
	delete(state.pipelines, e.ID)
	if run := e.pipeline.run; run != nil {
		delete(state.liveRuns, run.ID)
	}
	state.mu.Unlock()
	org := e.pipeline.organization
	org.mu.Lock()
	org.usage.removePipeline(e.pipeline.format)
	org.mu.Unlock()
	c := e.pipeline.connection
	c.mu.Lock()
	delete(c.pipelines, e.ID)
	c.mu.Unlock()
	ws := c.workspace
	if c.Role == Source && e.pipeline.Target == TargetUser {
		pipelinesToPurge := append(ws.pipelinesToPurge, e.ID)
		ws.mu.Lock()
		ws.pipelinesToPurge = pipelinesToPurge
		ws.mu.Unlock()
	}
	dispatchNotification(state, e)
	return org.ID
}

// DeleteWorkspace is the event sent when a workspace is deleted.
type DeleteWorkspace struct {
	ID        string
	workspace *Workspace
}

func (n DeleteWorkspace) Workspace() *Workspace {
	return n.workspace
}

// deleteWorkspace deletes a workspace.
func (state *State) deleteWorkspace(n notification) string {
	e := DeleteWorkspace{}
	decodeNotification(n, &e)
	e.workspace = state.workspaces[e.ID]
	org := e.workspace.organization
	// Update the organization.
	org.mu.Lock()
	delete(org.workspaces, e.ID)
	for _, key := range state.accessKeyByHMAC {
		if key.Workspace == e.ID {
			org.usage.removeAccessKey()
		}
	}
	org.usage.removeWorkspace(e.workspace)
	org.mu.Unlock()
	// Delete the workspace.
	state.mu.Lock()
	delete(state.workspaces, e.ID)
	// Delete access keys restricted to the workspace.
	for hmac, key := range state.accessKeyByHMAC {
		if key.Workspace == e.ID {
			delete(state.accessKeyByHMAC, hmac)
		}
	}
	// Delete the connections.
	for _, c := range e.workspace.connections {
		for _, key := range c.Keys {
			delete(state.connectionsByKey, key)
		}
		delete(state.connections, c.ID)
		// Delete the connection's pipelines.
		for _, p := range c.pipelines {
			delete(state.pipelines, p.ID)
			if p.run != nil {
				delete(state.liveRuns, p.run.ID)
			}
		}
	}
	state.mu.Unlock()
	dispatchNotification(state, e)
	return org.ID
}

// ElectLeader is the event sent when a leader is elected.
type ElectLeader struct {
	Number int
	Leader string
}

// electLeader elects a leader.
func (state *State) electLeader(n notification) {
	e := ElectLeader{}
	decodeNotification(n, &e)
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

// EndAlterProfileSchema is the event sent when the alter of a profile schema
// ends.
type EndAlterProfileSchema struct {
	Workspace   string
	ID          string
	EndTime     time.Time
	Err         string
	Schema      types.Type
	Identifiers []string
}

// endAlterProfileSchema ends the alter of the profile schema.
func (state *State) endAlterProfileSchema(n notification) string {
	e := EndAlterProfileSchema{}
	decodeNotification(n, &e)
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
	Workspace string
	ID        string
	EndTime   time.Time
}

// endIdentityResolution ends the Identity Resolution.
func (state *State) endIdentityResolution(n notification) string {
	e := EndIdentityResolution{}
	decodeNotification(n, &e)
	ws := state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.IR.ID = nil
		w.IR.EndTime = &e.EndTime
	})
	dispatchNotification(state, e)
	return ws.organization.ID
}

// EndPipelineRun is the event sent when pipeline run ends.
type EndPipelineRun struct {
	ID       string
	Pipeline string
	Health   Health
}

// endPipelineRun marks an in-progress pipeline run as finished.
func (state *State) endPipelineRun(n notification) string {
	e := EndPipelineRun{}
	decodeNotification(n, &e)
	state.mu.Lock()
	delete(state.liveRuns, e.ID)
	state.mu.Unlock()
	p := state.replacePipeline(e.Pipeline, func(p *Pipeline) {
		p.run = nil
		p.Health = e.Health
	})
	dispatchNotification(state, e)
	return p.organization.ID
}

// LinkConnection is the event sent when two unlinked connections are linked.
type LinkConnection struct {
	Connections [2]string
}

// InviteMember is the event sent when a member is invited.
type InviteMember struct {
	Member       string
	Organization string
}

// inviteMember invites a member.
func (state *State) inviteMember(n notification) string {
	e := InviteMember{}
	decodeNotification(n, &e)
	org := state.organizations[e.Organization]
	org.mu.Lock()
	org.members[e.Member] = false
	org.usage.addMember()
	org.mu.Unlock()
	return org.ID
}

// linkConnection links two unlinked connections.
func (state *State) linkConnection(n notification) string {
	e := LinkConnection{}
	decodeNotification(n, &e)
	state.replaceConnection(e.Connections[0], func(c *Connection) {
		c.LinkedConnections = addLinkedConnection(c.LinkedConnections, e.Connections[1])
	})
	c := state.replaceConnection(e.Connections[1], func(c *Connection) {
		c.LinkedConnections = addLinkedConnection(c.LinkedConnections, e.Connections[0])
	})
	dispatchNotification(state, e)
	return c.organization.ID
}

// PurgePipelines is the event sent when pipelines of a workspace are purged.
type PurgePipelines struct {
	Workspace        string
	PipelinesToPurge []string // remaining pipelines to purge. Never nil.
}

// purgePipelines purges pipelines of a workspace.
func (state *State) purgePipelines(n notification) string {
	e := PurgePipelines{}
	decodeNotification(n, &e)
	ws, _ := state.Workspace(e.Workspace)
	ws.mu.Lock()
	ws.pipelinesToPurge = e.PipelinesToPurge
	ws.mu.Unlock()
	return ws.organization.ID
}

// RenameConnection is the event sent when a connection is renamed.
type RenameConnection struct {
	Connection string
	Name       string
}

// renameConnection renames a connection.
func (state *State) renameConnection(n notification) string {
	e := RenameConnection{}
	decodeNotification(n, &e)
	c := state.replaceConnection(e.Connection, func(c *Connection) {
		c.Name = e.Name
	})
	return c.organization.ID
}

// RenameWorkspace is the event sent when a workspace is renamed.
type RenameWorkspace struct {
	Workspace string
	Name      string
}

// renameWorkspace renames a workspace.
func (state *State) renameWorkspace(n notification) string {
	e := RenameWorkspace{}
	decodeNotification(n, &e)
	ws := state.replaceWorkspace(e.Workspace, func(ws *Workspace) {
		ws.Name = e.Name
	})
	return ws.organization.ID
}

// RunPipeline is the event sent when a pipeline run starts.
type RunPipeline struct {
	ID          string
	Pipeline    string
	Incremental bool
	Cursor      time.Time
	StartTime   time.Time
}

// runPipeline runs a pipeline.
func (state *State) runPipeline(n notification) string {
	e := RunPipeline{}
	decodeNotification(n, &e)
	p := state.pipelines[e.Pipeline]
	run := &PipelineRun{
		mu:          &sync.Mutex{},
		ID:          e.ID,
		pipeline:    p,
		Incremental: e.Incremental,
		Cursor:      e.Cursor,
		StartTime:   e.StartTime,
	}
	p.mu.Lock()
	p.run = run
	p.mu.Unlock()
	state.mu.Lock()
	state.liveRuns[run.ID] = run
	state.mu.Unlock()
	dispatchNotification(state, e)
	return p.organization.ID
}

// SeeLeader is the event sent when the leader is seen.
type SeeLeader struct {
	Election int
}

// seeLeader sees the leader.
func (state *State) seeLeader(n notification) {
	e := SeeLeader{}
	decodeNotification(n, &e)
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
	Workspace    string
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Time
}

// setAccount sets an account.
func (state *State) setAccount(n notification) string {
	e := SetAccount{}
	decodeNotification(n, &e)
	ws := state.workspaces[e.Workspace]
	ws.replaceAccount(e.ID, func(a *Account) {
		a.AccessToken = e.AccessToken
		a.RefreshToken = e.RefreshToken
		a.ExpiresIn = e.ExpiresIn
	})
	return ws.organization.ID
}

// SetConnectionSettings is the event sent when the settings of a connection is
// changed.
type SetConnectionSettings struct {
	Connection string
	Settings   []byte
}

// setConnectionSettings sets the settings of a connection.
func (state *State) setConnectionSettings(n notification) string {
	e := SetConnectionSettings{}
	decodeNotification(n, &e)
	c := state.connections[e.Connection]
	c.mu.Lock()
	c.settings = e.Settings
	c.mu.Unlock()
	dispatchNotification(state, e)
	return c.organization.ID
}

// SetPipelineFormatSettings is the event sent when the format settings of a
// pipeline are changed.
type SetPipelineFormatSettings struct {
	Pipeline string
	Settings json.Value
}

// setPipelineFormatSettings sets the format settings of a pipeline.
func (state *State) setPipelineFormatSettings(n notification) string {
	e := SetPipelineFormatSettings{}
	decodeNotification(n, &e)
	p := state.replacePipeline(e.Pipeline, func(p *Pipeline) {
		p.FormatSettings = e.Settings
	})
	return p.organization.ID
}

// SetPipelineSchedulePeriod is the event sent when the schedule period of a
// pipeline is set.
type SetPipelineSchedulePeriod struct {
	ID             string
	SchedulePeriod int16
}

// setPipelineSchedulePeriod sets the schedule period of a pipeline.
func (state *State) setPipelineSchedulePeriod(n notification) string {
	e := SetPipelineSchedulePeriod{}
	decodeNotification(n, &e)
	p := state.replacePipeline(e.ID, func(p *Pipeline) {
		p.SchedulePeriod = e.SchedulePeriod
	})
	dispatchNotification(state, e)
	return p.organization.ID
}

// SetOrganizationStatus is the event sent when the status of an organization is
// set.
type SetOrganizationStatus struct {
	ID            string
	Enabled       bool
	endedLiveRuns []*PipelineRun
}

// EndedLiveRuns returns the organization's live runs that were ended.
func (n SetOrganizationStatus) EndedLiveRuns() []*PipelineRun {
	if n.endedLiveRuns == nil {
		return []*PipelineRun{}
	}
	return n.endedLiveRuns
}

// setOrganizationStatus sets the status of an organization.
func (state *State) setOrganizationStatus(n notification) string {
	e := SetOrganizationStatus{}
	decodeNotification(n, &e)
	o := state.replaceOrganization(e.ID, func(p *Organization) {
		p.Enabled = e.Enabled
	})
	// End the organization's live pipeline runs.
	if !e.Enabled {
		e.endedLiveRuns = []*PipelineRun{}
		for _, ws := range o.workspaces {
			for _, c := range ws.connections {
				for _, p := range c.pipelines {
					if p.run != nil {
						state.replacePipeline(p.ID, func(p *Pipeline) {
							p.run = nil
							p.Health = Healthy
						})
						e.endedLiveRuns = append(e.endedLiveRuns, p.run)
					}
				}
			}
		}
		if len(e.endedLiveRuns) > 0 {
			slices.SortFunc(e.endedLiveRuns, func(a, b *PipelineRun) int {
				return strings.Compare(a.ID, b.ID)
			})
			state.mu.Lock()
			for _, run := range e.endedLiveRuns {
				delete(state.liveRuns, run.ID)
			}
			state.mu.Unlock()
		}
	}
	dispatchNotification(state, e)
	return o.ID
}

// SetPipelineStatus is the event sent when the status of a pipeline is set.
type SetPipelineStatus struct {
	ID      string
	Enabled bool
}

// setPipelineStatus sets the status of a pipeline.
func (state *State) setPipelineStatus(n notification) string {
	e := SetPipelineStatus{}
	decodeNotification(n, &e)
	p := state.replacePipeline(e.ID, func(p *Pipeline) {
		p.Enabled = e.Enabled
	})
	dispatchNotification(state, e)
	return p.organization.ID
}

// StartAlterProfileSchema is the event sent when the alter of the profile
// schema starts.
type StartAlterProfileSchema struct {
	Workspace      string
	ID             string
	Schema         types.Type
	PrimarySources map[string]string // always != nil.
	Operations     []warehouses.AlterOperation
	StartTime      time.Time
}

// startAlterProfileSchema starts the alter of the profile schema.
func (state *State) startAlterProfileSchema(n notification) string {
	e := StartAlterProfileSchema{}
	decodeNotification(n, &e)
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
	Workspace string
	ID        string
	StartTime time.Time
}

// startIdentityResolution starts the Identity Resolution.
func (state *State) startIdentityResolution(n notification) string {
	e := StartIdentityResolution{}
	decodeNotification(n, &e)
	ws := state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.IR.ID = &e.ID
		w.IR.StartTime = &e.StartTime
		w.IR.EndTime = nil
	})
	dispatchNotification(state, e)
	return ws.organization.ID
}

// UnlinkConnection is the event sent when two linked connections are unlinked.
type UnlinkConnection struct {
	Connections [2]string
}

// unlinkConnection unlinks two linked connections.
func (state *State) unlinkConnection(n notification) string {
	e := UnlinkConnection{}
	decodeNotification(n, &e)
	state.replaceConnection(e.Connections[0], func(c *Connection) {
		c.LinkedConnections = removeLinkedConnection(c.LinkedConnections, e.Connections[1])
	})
	c := state.replaceConnection(e.Connections[1], func(c *Connection) {
		c.LinkedConnections = removeLinkedConnection(c.LinkedConnections, e.Connections[0])
	})
	dispatchNotification(state, e)
	return c.organization.ID
}

// UpdateConnection is the event sent when a connection is updated.
type UpdateConnection struct {
	Connection  string
	Name        string
	Strategy    *Strategy
	SendingMode *SendingMode
}

// updateConnection updates a connection.
func (state *State) updateConnection(n notification) string {
	e := UpdateConnection{}
	decodeNotification(n, &e)
	c := state.replaceConnection(e.Connection, func(c *Connection) {
		c.Name = e.Name
		c.Strategy = e.Strategy
		c.SendingMode = e.SendingMode
	})
	dispatchNotification(state, e)
	return c.organization.ID
}

// UpdateConsentPurpose is the event sent when a consent purpose is updated.
type UpdateConsentPurpose struct {
	Workspace              string
	ID                     string
	Name                   string
	EventConsentLocations  []EventConsentLocation
	ProfileConsentLocation *ProfileConsentLocation
}

// updateConsentPurpose updates a consent purpose.
func (state *State) updateConsentPurpose(n notification) string {
	e := UpdateConsentPurpose{}
	decodeNotification(n, &e)
	ws := state.workspaces[e.Workspace]
	previous := ws.consentPurposes[e.ID]
	cp := ws.replaceConsentPurpose(e.ID, func(cp *ConsentPurpose) {
		cp.Name = e.Name
		cp.EventConsentLocations = e.EventConsentLocations
		cp.ProfileConsentLocation = e.ProfileConsentLocation
	})
	// Replace the consent purpose in the pipelines that require it.
	for _, c := range ws.connections {
		for _, p := range c.pipelines {
			i := slices.Index(p.RequiredConsents.Purposes, previous)
			if i == -1 {
				continue
			}
			purposes := slices.Clone(p.RequiredConsents.Purposes)
			purposes[i] = cp
			state.replacePipeline(p.ID, func(p *Pipeline) {
				p.RequiredConsents.Purposes = purposes
			})
		}
	}
	dispatchNotification(state, e)
	return ws.organization.ID
}

// UpdateIdentityPropertiesToUnset is the event sent when the identity
// properties to unset of a pipeline are updated.
type UpdateIdentityPropertiesToUnset struct {
	Pipeline   string
	Properties []string // Always non-nil.
}

// updateIdentityPropertiesToUnset updates the identity properties to unset of
// a pipeline.
func (state *State) updateIdentityPropertiesToUnset(n notification) string {
	e := UpdateIdentityPropertiesToUnset{}
	decodeNotification(n, &e)
	p := state.pipelines[e.Pipeline]
	p.mu.Lock()
	p.propertiesToUnset = e.Properties
	p.mu.Unlock()
	return p.organization.ID
}

// UpdateIdentityResolutionSettings is the event sent when the identity
// resolution settings of a workspace are updated.
type UpdateIdentityResolutionSettings struct {
	Workspace                      string
	ResolveIdentitiesOnBatchImport bool
	Identifiers                    []string
}

// updateIdentityResolutionSettings updates the identity resolution settings of
// a workspace.
func (state *State) updateIdentityResolutionSettings(n notification) string {
	e := UpdateIdentityResolutionSettings{}
	decodeNotification(n, &e)
	ws := state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.ResolveIdentitiesOnBatchImport = e.ResolveIdentitiesOnBatchImport
		w.Identifiers = e.Identifiers
	})
	return ws.organization.ID
}

// UpdateOrganization is the event sent when an organization is updated.
type UpdateOrganization struct {
	ID     string
	Name   string
	Limits *OrganizationLimits // if nil, limits is not updated.
}

// updateOrganization updates an organization.
func (state *State) updateOrganization(n notification) string {
	e := UpdateOrganization{}
	decodeNotification(n, &e)
	state.replaceOrganization(e.ID, func(org *Organization) {
		org.Name = e.Name
		if e.Limits != nil {
			org.usage.setLimits(*e.Limits)
		}
	})
	return e.ID
}

// UpdatePipeline is the event sent when a pipeline is updated.
type UpdatePipeline struct {
	ID                 string
	Name               string
	Enabled            bool
	InSchema           types.Type
	OutSchema          types.Type
	Filter             stdjson.RawMessage
	RequiredConsents   RequiredConsentsByIDs
	Transformation     Transformation
	Query              string
	Format             string
	Path               string
	Sheet              string
	Compression        Compression
	OrderBy            string
	FormatSettings     json.Value
	ExportMode         ExportMode
	Matching           Matching
	UpdateOnDuplicates bool
	TableName          string
	TableKey           string
	UserIDColumn       string
	UpdatedAtColumn    string
	UpdatedAtFormat    string
	Incremental        bool
	PropertiesToUnset  []string
}

// updatePipeline updates a pipeline.
func (state *State) updatePipeline(n notification) string {
	e := UpdatePipeline{}
	decodeNotification(n, &e)
	// json.Value(nil) is marshaled into "null", but when it is
	// deserialized it becomes json.Value("null"), so this code converts it
	// back to json.Value(nil).
	if json.Value(e.Filter).IsNull() {
		e.Filter = nil
	}
	if e.FormatSettings.IsNull() {
		e.FormatSettings = nil
	}
	format := state.connectors[e.Format]
	var filter *Where
	if e.Filter != nil {
		var err error
		filter, err = unmarshalWhere(e.Filter, e.InSchema)
		if err != nil {
			panic(fmt.Sprintf("invalid notification payload %s (version %d)", n.Name, n.Version))
		}
	}
	previous := state.pipelines[e.ID]
	ws := previous.connection.workspace
	requiredConsents := RequiredConsents{
		Operator: e.RequiredConsents.Operator,
		Purposes: make([]*ConsentPurpose, len(e.RequiredConsents.Purposes)),
	}
	for i, id := range e.RequiredConsents.Purposes {
		requiredConsents.Purposes[i] = ws.consentPurposes[id]
	}
	p := state.replacePipeline(e.ID, func(p *Pipeline) {
		p.format = format
		p.propertiesToUnset = e.PropertiesToUnset
		p.Name = e.Name
		p.Enabled = e.Enabled
		p.InSchema = e.InSchema
		p.OutSchema = e.OutSchema
		p.Filter = filter
		p.RequiredConsents = requiredConsents
		p.Transformation = e.Transformation
		p.Query = e.Query
		p.Path = e.Path
		p.Sheet = e.Sheet
		p.Compression = e.Compression
		p.OrderBy = e.OrderBy
		p.FormatSettings = e.FormatSettings
		p.ExportMode = e.ExportMode
		p.Matching = e.Matching
		p.UpdateOnDuplicates = e.UpdateOnDuplicates
		p.TableName = e.TableName
		p.TableKey = e.TableKey
		p.UserIDColumn = e.UserIDColumn
		p.UpdatedAtColumn = e.UpdatedAtColumn
		p.UpdatedAtFormat = e.UpdatedAtFormat
		p.Incremental = e.Incremental
	})
	org := p.organization
	// When the format changes, both oldFormat and format are non-nil.
	if previous.format != format {
		org.mu.Lock()
		org.usage.updatePipelineFormat(previous.format, format)
		org.mu.Unlock()
	}
	dispatchNotification(state, e)
	return org.ID
}

// UpdateWarehouse is the event sent when a warehouse is updated.
type UpdateWarehouse struct {
	Workspace                    string
	Mode                         WarehouseMode
	Settings                     []byte
	MCPSettings                  []byte
	CancelIncompatibleOperations bool
	settingsHaveChanged          bool
	mcpSettingsHaveChanged       bool
}

// SettingsHaveChanged reports whether settings have changed.
func (n UpdateWarehouse) SettingsHaveChanged() bool {
	return n.settingsHaveChanged
}

// MCPSettingsHaveChanged reports whether MCP settings have changed.
func (n UpdateWarehouse) MCPSettingsHaveChanged() bool {
	return n.mcpSettingsHaveChanged
}

// updateWarehouse updates a warehouse.
func (state *State) updateWarehouse(n notification) string {
	e := UpdateWarehouse{}
	decodeNotification(n, &e)
	ws := state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.Warehouse.Mode = e.Mode
		if e.settingsHaveChanged = !bytes.Equal(w.Warehouse.settings, e.Settings); e.settingsHaveChanged {
			w.Warehouse.settings = e.Settings
		}
		if e.mcpSettingsHaveChanged = !bytes.Equal(w.Warehouse.mcpSettings, e.MCPSettings); e.mcpSettingsHaveChanged {
			w.Warehouse.mcpSettings = e.MCPSettings
		}
	})
	dispatchNotification(state, e)
	return ws.organization.ID
}

// UpdateWarehouseMode is the event sent when the mode of a data warehouse is
// updated.
type UpdateWarehouseMode struct {
	Workspace                    string
	Mode                         WarehouseMode
	CancelIncompatibleOperations bool
}

// updateWarehouseMode updates the mode of a data warehouse.
func (state *State) updateWarehouseMode(n notification) string {
	e := UpdateWarehouseMode{}
	decodeNotification(n, &e)
	ws := state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.Warehouse.Mode = e.Mode
	})
	dispatchNotification(state, e)
	return ws.organization.ID
}

// UpdateWorkspace is the event sent when the name and the displayed properties
// of a workspace are updated.
type UpdateWorkspace struct {
	Workspace     string
	Name          string
	UIPreferences UIPreferences
}

// updateWorkspace updates the name and the displayed properties of a workspace.
func (state *State) updateWorkspace(n notification) string {
	e := UpdateWorkspace{}
	decodeNotification(n, &e)
	ws := state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		w.Name = e.Name
		w.UIPreferences = e.UIPreferences
	})
	dispatchNotification(state, e)
	return ws.organization.ID
}

// addLinkedConnection adds id to the provided linked connections. It returns
// a copy of connections with id added in numerical order. It is assumed that
// connections is already sorted and id does not already exist in connections.
func addLinkedConnection(connections []string, id string) []string {
	cc := make([]string, len(connections)+1)
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
func removeLinkedConnection(connections []string, id string) []string {
	if len(connections) == 1 {
		return []string{}
	}
	cc := make([]string, len(connections)-1)
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
