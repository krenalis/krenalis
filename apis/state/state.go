//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package state

import (
	"context"
	"database/sql/driver"
	"fmt"
	"sort"
	"sync"
	"time"

	"chichi/apis/postgres"
	"chichi/apis/warehouses"
	_connector "chichi/connector"
	"chichi/connector/types"

	"github.com/google/uuid"
	"golang.org/x/exp/maps"
)

// election represents a leader election.
type election struct {
	number   int
	leader   uuid.UUID
	lastSeen time.Time
}

// State represents the application state.
type State struct {
	id               uuid.UUID
	mu               *sync.Mutex
	db               *postgres.DB
	ctx              context.Context
	syncing          bool // reports whether the keeper has started synchronizing the state.
	election         election
	accounts         map[int]*Account
	connectors       map[int]*Connector
	workspaces       map[int]*Workspace
	connections      map[int]*Connection
	connectionsByKey map[string]*Connection
	actions          map[int]*Action
	resources        map[int]*Resource
	notifications    <-chan postgres.Notification
	listeners        struct {
		AddAction                 []func(AddActionNotification)
		AddConnection             []func(AddConnectionNotification)
		DeleteAction              []func(DeleteActionNotification)
		DeleteConnection          []func(DeleteConnectionNotification)
		DeleteWorkspace           []func(DeleteWorkspaceNotification)
		ElectLeader               []func(ElectLeaderNotification)
		ExecuteAction             []func(ExecuteActionNotification)
		SetAction                 []func(SetActionNotification)
		SetActionSchedulePeriod   []func(SetActionSchedulePeriodNotification)
		SetConnectionSettings     []func(SetConnectionSettingsNotification)
		SetConnectionStatus       []func(SetConnectionStatusNotification)
		SetWarehouseSettings      []func(SetWarehouseSettingsNotification)
		SetWorkspacePrivacyRegion []func(SetWorkspacePrivacyRegion)
	}
}

// Account returns the account with identifier id.
// The boolean return value reports whether the account exists.
func (state *State) Account(id int) (*Account, bool) {
	state.mu.Lock()
	a, ok := state.accounts[id]
	state.mu.Unlock()
	return a, ok
}

// Accounts returns all accounts.
func (state *State) Accounts() []*Account {
	state.mu.Lock()
	accounts := make([]*Account, len(state.accounts))
	i := 0
	for _, account := range state.accounts {
		accounts[i] = account
		i++
	}
	state.mu.Unlock()
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].ID < accounts[j].ID
	})
	return accounts
}

// Action returns the action with identifier id.
// The boolean return value reports whether the action exists.
func (state *State) Action(id int) (*Action, bool) {
	state.mu.Lock()
	a, ok := state.actions[id]
	state.mu.Unlock()
	return a, ok
}

// Actions returns all the actions from every connection.
func (state *State) Actions() []*Action {
	state.mu.Lock()
	actions := make([]*Action, len(state.actions))
	i := 0
	for _, action := range state.actions {
		actions[i] = action
		i++
	}
	state.mu.Unlock()
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].ID < actions[j].ID
	})
	return actions
}

// ConnectionByKey returns the connection with the given key.
// The boolean return value reports whether the key exists.
func (state *State) ConnectionByKey(key string) (*Connection, bool) {
	state.mu.Lock()
	c, ok := state.connectionsByKey[key]
	state.mu.Unlock()
	return c, ok
}

// Connection returns the connection with identifier id.
// The boolean return value reports whether the connection exists.
func (state *State) Connection(id int) (*Connection, bool) {
	state.mu.Lock()
	c, ok := state.connections[id]
	state.mu.Unlock()
	return c, ok
}

// Connections returns all connections.
func (state *State) Connections() []*Connection {
	state.mu.Lock()
	connections := maps.Values(state.connections)
	state.mu.Unlock()
	return connections
}

// Connector returns the connector with identifier id.
// The boolean return value reports whether the connector exists.
func (state *State) Connector(id int) (*Connector, bool) {
	state.mu.Lock()
	c, ok := state.connectors[id]
	state.mu.Unlock()
	return c, ok
}

// Connectors returns all connectors.
func (state *State) Connectors() []*Connector {
	state.mu.Lock()
	connectors := make([]*Connector, len(state.connectors))
	i := 0
	for _, connector := range state.connectors {
		connectors[i] = connector
		i++
	}
	state.mu.Unlock()
	sort.Slice(connectors, func(i, j int) bool {
		return connectors[i].ID < connectors[j].ID
	})
	return connectors
}

// IsLeader reports whether this node is the leader.
func (state *State) IsLeader() bool {
	state.mu.Lock()
	election := state.election
	state.mu.Unlock()
	return election.leader == state.id
}

// Workspace returns the workspace with identifier id.
// The boolean return value reports whether the workspace exists.
func (state *State) Workspace(id int) (*Workspace, bool) {
	state.mu.Lock()
	ws, ok := state.workspaces[id]
	state.mu.Unlock()
	return ws, ok
}

// Account represents an account.
type Account struct {
	mu          *sync.Mutex
	workspaces  map[int]*Workspace
	ID          int
	Name        string
	Email       string
	InternalIPs []string
}

// Workspace returns the workspace of the account with identifier id.
// The boolean return value reports whether the workspace exists.
func (account *Account) Workspace(id int) (*Workspace, bool) {
	account.mu.Lock()
	w, ok := account.workspaces[id]
	account.mu.Unlock()
	return w, ok
}

// Workspaces returns all the workspaces of the account.
func (account *Account) Workspaces() []*Workspace {
	account.mu.Lock()
	workspaces := make([]*Workspace, len(account.workspaces))
	i := 0
	for _, w := range account.workspaces {
		workspaces[i] = w
		i++
	}
	account.mu.Unlock()
	return workspaces
}

// Workspace represents a workspace.
type Workspace struct {
	mu            *sync.Mutex
	Warehouse     warehouses.Warehouse
	Schemas       map[string]*types.Type
	connections   map[int]*Connection
	ID            int
	account       *Account
	Name          string
	resources     map[int]*Resource
	PrivacyRegion PrivacyRegion
}

// Account returns the account of the workspace.
func (workspace *Workspace) Account() *Account {
	workspace.mu.Lock()
	account := workspace.account
	workspace.mu.Unlock()
	return account
}

// Connection returns the connection of the workspace with identifier id.
// The boolean return value reports whether the connection exists.
func (workspace *Workspace) Connection(id int) (*Connection, bool) {
	workspace.mu.Lock()
	c, ok := workspace.connections[id]
	workspace.mu.Unlock()
	return c, ok
}

// Connections returns all the connections of the workspace.
func (workspace *Workspace) Connections() []*Connection {
	workspace.mu.Lock()
	connections := make([]*Connection, len(workspace.connections))
	i := 0
	for _, c := range workspace.connections {
		connections[i] = c
		i++
	}
	workspace.mu.Unlock()
	return connections
}

// Resource returns the resource with identifier id. The boolean return value
// reports whether the resource exists.
func (workspace *Workspace) Resource(id int) (*Resource, bool) {
	workspace.mu.Lock()
	r, ok := workspace.resources[id]
	workspace.mu.Unlock()
	return r, ok
}

// ResourceByCode returns the resource with the given code. The boolean return value
// reports whether the resource exists.
func (workspace *Workspace) ResourceByCode(code string) (*Resource, bool) {
	var r *Resource
	workspace.mu.Lock()
	for _, resource := range workspace.resources {
		if resource.Code == code {
			r = resource
			break
		}
	}
	workspace.mu.Unlock()
	return r, r != nil
}

// PrivacyRegion represents a privacy region.
type PrivacyRegion string

const (
	PrivacyRegionNotSpecified PrivacyRegion = ""
	PrivacyRegionEurope       PrivacyRegion = "Europe"
)

// Connector represents a connector.
type Connector struct {
	ID                     int
	Name                   string
	SourceDescription      string
	DestinationDescription string
	TermForUsers           string
	TermForGroups          string
	Type                   ConnectorType
	Targets                ConnectorTargets
	HasSheets              bool
	HasSettings            bool
	Icon                   string
	WebhooksPer            WebhooksPer
	OAuth                  *OAuth
}

// ConnectorTargets represents the targets of a connector.
type ConnectorTargets int

const (
	EventsFlag = 1 << iota
	UsersFlag
	GroupsFlag
)

// Contains reports whether t contains the given action target.
func (t ConnectorTargets) Contains(target ActionTarget) bool {
	switch target {
	case EventsTarget:
		return t&EventsFlag != 0
	case UsersTarget:
		return t&UsersFlag != 0
	case GroupsTarget:
		return t&GroupsFlag != 0
	}
	panic("invalid action target")
}

// ConnectorType represents a connector type.
type ConnectorType int

const (
	AppType ConnectorType = iota + 1
	DatabaseType
	FileType
	MobileType
	ServerType
	StorageType
	StreamType
	WebsiteType
)

// Scan implements the sql.Scanner interface.
func (typ *ConnectorType) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an state.ConnectorType value", src)
	}
	var t ConnectorType
	switch s {
	case "App":
		t = AppType
	case "Database":
		t = DatabaseType
	case "File":
		t = FileType
	case "Mobile":
		t = MobileType
	case "Server":
		t = ServerType
	case "Storage":
		t = StorageType
	case "Stream":
		t = StreamType
	case "Website":
		t = WebsiteType
	default:
		return fmt.Errorf("invalid state.ConnectionType: %s", s)
	}
	*typ = t
	return nil
}

// String returns the string representation of typ.
// It panics if typ is not a valid ConnectorType value.
func (typ ConnectorType) String() string {
	s, err := typ.Value()
	if err != nil {
		panic("invalid connector type")
	}
	return s.(string)
}

// Value implements driver.Valuer interface.
// It returns an error if typ is not a valid ConnectorType.
func (typ ConnectorType) Value() (driver.Value, error) {
	switch typ {
	case AppType:
		return "App", nil
	case DatabaseType:
		return "Database", nil
	case FileType:
		return "File", nil
	case MobileType:
		return "Mobile", nil
	case ServerType:
		return "Server", nil
	case StorageType:
		return "Storage", nil
	case StreamType:
		return "Stream", nil
	case WebsiteType:
		return "Website", nil
	}
	return nil, fmt.Errorf("not a valid ConnectorType: %d", typ)
}

type WebhooksPer int

const (
	WebhooksPerNone WebhooksPer = iota
	WebhooksPerConnector
	WebhooksPerResource
	WebhooksPerSource
)

// Scan implements the sql.Scanner interface.
func (per *WebhooksPer) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an state.WebhooksPer value", src)
	}
	var p WebhooksPer
	switch s {
	case "None":
		p = WebhooksPerNone
	case "Connector":
		p = WebhooksPerConnector
	case "Resource":
		p = WebhooksPerResource
	case "Source":
		p = WebhooksPerSource
	default:
		return fmt.Errorf("invalid state.WebhooksPer: %s", s)
	}
	*per = p
	return nil
}

// String returns the string representation of w.
// It panics if w is not a valid WebhooksPer value.
func (per WebhooksPer) String() string {
	s, err := per.Value()
	if err != nil {
		panic("invalid webhooksPer value")
	}
	return s.(string)
}

// Value implements driver.Valuer interface.
// It returns an error if typ is not a valid ConnectionRole.
func (per WebhooksPer) Value() (driver.Value, error) {
	switch per {
	case WebhooksPerNone:
		return "None", nil
	case WebhooksPerConnector:
		return "Connector", nil
	case WebhooksPerResource:
		return "Resource", nil
	case WebhooksPerSource:
		return "Source", nil
	}
	return nil, fmt.Errorf("not a valid WebhooksPer: %d", per)
}

// An OAuth represents OAuth data required to authenticate with a connector.
type OAuth struct {
	_connector.OAuth
	ClientID     string
	ClientSecret string
}

// Resource represents a resource.
type Resource struct {
	mu           *sync.Mutex
	ID           int
	workspace    *Workspace
	connector    *Connector
	Code         string
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Time
}

// Workspace returns the workspace of the resource.
func (resource *Resource) Workspace() *Workspace {
	resource.mu.Lock()
	w := resource.workspace
	resource.mu.Unlock()
	return w
}

// Connector returns the connector of the resource.
func (resource *Resource) Connector() *Connector {
	resource.mu.Lock()
	c := resource.connector
	resource.mu.Unlock()
	return c
}

// Connection represents a connection.
type Connection struct {
	mu              *sync.Mutex
	account         *Account
	workspace       *Workspace
	ID              int
	Name            string
	Role            ConnectionRole
	Enabled         bool
	connector       *Connector
	storage         *Connection
	resource        *Resource
	WebsiteHost     string
	Keys            []string
	IdentityColumn  string
	TimestampColumn string
	Settings        []byte
	UsersQuery      string
	actions         map[int]*Action
	Health          Health
}

// Account returns the account of the connection.
func (connection *Connection) Account() *Account {
	connection.mu.Lock()
	a := connection.account
	connection.mu.Unlock()
	return a
}

// Workspace returns the workspace of the connection.
func (connection *Connection) Workspace() *Workspace {
	connection.mu.Lock()
	w := connection.workspace
	connection.mu.Unlock()
	return w
}

// Connector returns the connector of the connection.
func (connection *Connection) Connector() *Connector {
	connection.mu.Lock()
	c := connection.connector
	connection.mu.Unlock()
	return c
}

// Storage returns the storage of the connection.
// The boolean return value reports whether the connection has a storage.
func (connection *Connection) Storage() (*Connection, bool) {
	connection.mu.Lock()
	s := connection.storage
	connection.mu.Unlock()
	return s, s != nil
}

// Resource returns the resource of the connection.
// The boolean return value reports whether the connection has a resource.
func (connection *Connection) Resource() (*Resource, bool) {
	connection.mu.Lock()
	r := connection.resource
	connection.mu.Unlock()
	return r, r != nil
}

// Action returns the action of the connection with identifier id.
// The boolean return value reports whether the action exists.
func (connection *Connection) Action(id int) (*Action, bool) {
	connection.mu.Lock()
	a, ok := connection.actions[id]
	connection.mu.Unlock()
	return a, ok
}

// Actions returns the actions of the connection.
func (connection *Connection) Actions() []*Action {
	connection.mu.Lock()
	actions := make([]*Action, len(connection.actions))
	i := 0
	for _, a := range connection.actions {
		actions[i] = a
		i++
	}
	connection.mu.Unlock()
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].ID < actions[j].ID
	})
	return actions
}

// Health is an indicator of the current state of an action or a connection.
type Health int

const (
	Healthy Health = iota
	NoRecentData
	RecentError
	AccessDenied
)

// Scan implements the sql.Scanner interface.
func (health *Health) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an state.Health value", src)
	}
	var h Health
	switch s {
	case "Healthy":
		h = Healthy
	case "NoRecentData":
		h = NoRecentData
	case "RecentError":
		h = RecentError
	case "AccessDenied":
		h = AccessDenied
	default:
		return fmt.Errorf("invalid state.Health: %s", s)
	}
	*health = h
	return nil
}

// String returns the string representation of health.
// It panics if health is not a valid Health value.
func (health Health) String() string {
	switch health {
	case Healthy:
		return "Healthy"
	case NoRecentData:
		return "NoRecentData"
	case RecentError:
		return "RecentError"
	case AccessDenied:
		return "AccessDenied"
	}
	panic("invalid connection health")
}

// Value implements driver.Valuer interface.
// It returns an error if health is not a valid Health.
func (health Health) Value() (driver.Value, error) {
	switch health {
	case Healthy:
		return "Healthy", nil
	case NoRecentData:
		return "NoRecentData", nil
	case RecentError:
		return "RecentError", nil
	case AccessDenied:
		return "AccessDenied", nil
	}
	return nil, fmt.Errorf("not a valid Health: %d", health)
}

// ConnectionRole represents a connection role.
type ConnectionRole int

const (
	SourceRole      ConnectionRole = iota + 1 // source
	DestinationRole                           // destination
)

// Scan implements the sql.Scanner interface.
func (role *ConnectionRole) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an state.ConnectionRole value", src)
	}
	var r ConnectionRole
	switch s {
	case "Source":
		r = SourceRole
	case "Destination":
		r = DestinationRole
	default:
		return fmt.Errorf("invalid state.ConnectionRole: %s", s)
	}
	*role = r
	return nil
}

// String returns the string representation of role.
// It panics if role is not a valid ConnectionRole value.
func (role ConnectionRole) String() string {
	switch role {
	case SourceRole:
		return "Source"
	case DestinationRole:
		return "Destination"
	}
	panic("invalid connection role")
}

// Value implements driver.Valuer interface.
// It returns an error if role is not a valid ConnectionRole.
func (role ConnectionRole) Value() (driver.Value, error) {
	switch role {
	case SourceRole:
		return "Source", nil
	case DestinationRole:
		return "Destination", nil
	}
	return nil, fmt.Errorf("not a valid ConnectionRole: %d", role)
}

// Transformation represents a Python transformation which can be associated to
// an action.
type Transformation struct {

	// In is the input schema of the transformation, which should have at least
	// one property in it.
	In types.Type

	// Out is the output schema of the transformation, which should have at
	// least one property in it.
	Out types.Type

	// PythonSource is the Python source code of this transformation, which
	// declares the 'transform' function which takes in input and returns a
	// Python dictionary.
	PythonSource string
}

// ActionTarget represents the action target of a connection.
type ActionTarget int

const (
	EventsTarget ActionTarget = iota + 1
	UsersTarget
	GroupsTarget
)

// Scan implements the sql.Scanner interface.
func (target *ActionTarget) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an state.ActionTarget value", src)
	}
	switch s {
	case "Events":
		*target = EventsTarget
	case "Users":
		*target = UsersTarget
	case "Groups":
		*target = GroupsTarget
	default:
		return fmt.Errorf("invalid state.ActionTarget: %s", s)
	}
	return nil
}

// String returns the string representation of target.
// It panics if target is not a valid ActionTarget value.
func (target ActionTarget) String() string {
	switch target {
	case EventsTarget:
		return "Events"
	case UsersTarget:
		return "Users"
	case GroupsTarget:
		return "Groups"
	}
	panic("invalid action target")
}

// Value implements driver.Valuer interface.
// It returns an error if target is not a valid ActionTarget.
func (target ActionTarget) Value() (driver.Value, error) {
	switch target {
	case EventsTarget:
		return "Events", nil
	case UsersTarget:
		return "Users", nil
	case GroupsTarget:
		return "Groups", nil
	}
	return nil, fmt.Errorf("not a valid ActionTarget: %d", target)
}

type Action struct {
	mu                       *sync.Mutex
	ID                       int
	connection               *Connection
	execution                *ActionExecution
	Target                   ActionTarget
	Name                     string
	Enabled                  bool
	EventType                string
	ScheduleStart            int16
	SchedulePeriod           int16
	Filter                   *ActionFilter
	Schema                   types.Type
	Mapping                  map[string]string
	Transformation           *Transformation
	Query                    string
	Path                     string
	Sheet                    string
	UserCursor               string
	Health                   Health
	ExportMode               *ExportMode
	ExportMatchingProperties *ExportMatchingProperties
}

// ExportMode represents one of the three export modes.
type ExportMode string

const (
	CreateOnly     ExportMode = "CreateOnly"
	UpdateOnly     ExportMode = "UpdateOnly"
	CreateOrUpdate ExportMode = "CreateOrUpdate"
)

// ExportMatchingProperties contains an internal property (belonging to the
// Golden Record) and an external property (belonging to the app) which are used
// to match identities of users in the data warehouse with users on the external
// app, during export.
type ExportMatchingProperties struct {
	Internal string
	External string
}

// ActionExecution represents an action execution.
type ActionExecution struct {
	mu        *sync.Mutex
	ID        int
	action    *Action
	storage   *Connection
	Reimport  bool
	StartTime time.Time
}

// Action returns the action of the execution.
func (ex *ActionExecution) Action() *Action {
	ex.mu.Lock()
	a := ex.action
	ex.mu.Unlock()
	return a
}

// Storage returns the storage of the execution.
// The boolean return value reports whether the execution has a storage.
func (ex *ActionExecution) Storage() (*Connection, bool) {
	ex.mu.Lock()
	s := ex.storage
	ex.mu.Unlock()
	return s, s != nil
}

// ActionFilter represents a filter of an action.
type ActionFilter struct {
	Logical    ActionFilterLogical     // can be "all" or "any".
	Conditions []ActionFilterCondition // cannot be empty.
}

// ActionFilterLogical represents the logical operator of an action filter.
// It can be "all" or "any".
type ActionFilterLogical string

// ActionFilterCondition represents the condition of an action filter.
type ActionFilterCondition struct {
	Property string // A property identifier or selector (e.g. "street1" or "traits.address.street1").
	Operator string // "is", "is not".
	Value    string // "Track", "Page", ...
}

// Connection returns the connection of the action.
func (action *Action) Connection() *Connection {
	action.mu.Lock()
	c := action.connection
	action.mu.Unlock()
	return c
}

// Execution returns the execution of the action.
// The boolean return value reports whether the action is running.
func (action *Action) Execution() (*ActionExecution, bool) {
	action.mu.Lock()
	ex := action.execution
	action.mu.Unlock()
	return ex, ex != nil
}
