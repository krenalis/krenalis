//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package state

import (
	"database/sql/driver"
	"fmt"
	"sort"
	"sync"
	"time"

	"chichi/apis/postgres"
	"chichi/apis/types"
	"chichi/apis/warehouses"

	"golang.org/x/exp/maps"
)

// State represents the application state.
type State struct {
	mu               *sync.Mutex
	db               *postgres.DB
	accounts         map[int]*Account
	connectors       map[int]*Connector
	workspaces       map[int]*Workspace
	connections      map[int]*Connection
	connectionsByKey map[string]*Connection
	resources        map[int]*Resource
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

// ConnectionByKey returns the connection with the given key.
// The boolean return value reports whether the key exists.
func (state *State) ConnectionByKey(key string) (*Connection, bool) {
	state.mu.Lock()
	c, ok := state.connectionsByKey[key]
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
	mu          *sync.Mutex
	Warehouse   warehouses.Warehouse
	Schemas     map[string]*types.Type
	connections map[int]*Connection
	//EventListeners *EventListeners
	ID        int
	account   *Account
	resources map[int]*Resource
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

// Connector represents a connector.
type Connector struct {
	ID          int
	Name        string
	Type        ConnectorType
	LogoURL     string
	WebhooksPer WebhooksPer
	OAuth       *ConnectorOAuth
}

// ConnectorType represents a connector type.
type ConnectorType int

const (
	AppType ConnectorType = iota + 1
	DatabaseType
	EventStreamType
	FileType
	MobileType
	ServerType
	StorageType
	WebsiteType
)

// Scan implements the sql.Scanner interface.
func (typ *ConnectorType) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an api.ConnectorType value", src)
	}
	var t ConnectorType
	switch s {
	case "App":
		t = AppType
	case "Database":
		t = DatabaseType
	case "EventStream":
		t = EventStreamType
	case "File":
		t = FileType
	case "Mobile":
		t = MobileType
	case "Server":
		t = ServerType
	case "Storage":
		t = StorageType
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
	case EventStreamType:
		return "EventStream", nil
	case FileType:
		return "File", nil
	case MobileType:
		return "Mobile", nil
	case ServerType:
		return "Server", nil
	case StorageType:
		return "Storage", nil
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
		return fmt.Errorf("cannot scan a %T value into an api.WebhooksPer value", src)
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

// A ConnectorOAuth represents OAuth data required to authenticate with a
// connector.
type ConnectorOAuth struct {
	URL              string
	ClientID         string
	ClientSecret     string
	TokenEndpoint    string
	DefaultTokenType string
	DefaultExpiresIn int
	ForcedExpiresIn  int
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
	mu               *sync.Mutex
	account          *Account
	workspace        *Workspace
	ID               int
	Name             string
	Role             ConnectionRole
	Enabled          bool
	connector        *Connector
	storage          *Connection
	stream           *Connection
	resource         *Resource
	WebsiteHost      string
	Keys             []string
	UserCursor       string
	IdentityColumn   string
	TimestampColumn  string
	Settings         []byte
	Schema           types.Type
	UsersQuery       string
	importInProgress *ImportInProgress
	mappings         []*Mapping
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

// Stream returns the stream of the connection.
// If there is no stream, it returns nil.
func (connection *Connection) Stream() *Connection {
	connection.mu.Lock()
	s := connection.stream
	connection.mu.Unlock()
	return s
}

// Storage returns the storage of the connection.
// If there is no storage, it returns nil.
func (connection *Connection) Storage() *Connection {
	connection.mu.Lock()
	s := connection.storage
	connection.mu.Unlock()
	return s
}

// Resource returns the resource of the connection.
// If there is no resource, it returns nil.
func (connection *Connection) Resource() *Resource {
	connection.mu.Lock()
	r := connection.resource
	connection.mu.Unlock()
	return r
}

// ImportInProgress returns the in progress import of the connection.
// If there is no import in progress, it returns nil.
func (connection *Connection) ImportInProgress() *ImportInProgress {
	connection.mu.Lock()
	im := connection.importInProgress
	connection.mu.Unlock()
	return im
}

// Mappings returns the mappings of the connection.
// If there is no mappings, it returns nil.
func (connection *Connection) Mappings() []*Mapping {
	connection.mu.Lock()
	ms := connection.mappings
	connection.mu.Unlock()
	return ms
}

// ImportInProgress represents a connection import in progress.
type ImportInProgress struct {
	mu         *sync.Mutex
	ID         int
	connection *Connection
	storage    *Connection
	Reimport   bool
	StartTime  time.Time
}

// Connection returns the connection of the import.
func (imp *ImportInProgress) Connection() *Connection {
	imp.mu.Lock()
	c := imp.connection
	imp.mu.Unlock()
	return c
}

// Storage returns the storage of the import.
// If there is no storage, it returns nil.
func (imp *ImportInProgress) Storage() *Connection {
	imp.mu.Lock()
	s := imp.storage
	imp.mu.Unlock()
	return s
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
		return fmt.Errorf("cannot scan a %T value into an api.ConnectionRole value", src)
	}
	var r ConnectionRole
	switch s {
	case "Source":
		r = SourceRole
	case "Destination":
		r = DestinationRole
	default:
		return fmt.Errorf("invalid api.ConnectionRole: %s", s)
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
// It returns an error if typ is not a valid ConnectionRole.
func (role ConnectionRole) Value() (driver.Value, error) {
	switch role {
	case SourceRole:
		return "Source", nil
	case DestinationRole:
		return "Destination", nil
	}
	return nil, fmt.Errorf("not a valid ConnectionRole: %d", role)
}

// Mapping represents a mapping from a kind of properties to another.
type Mapping struct {
	mu             *sync.Mutex
	ID             int
	connection     *Connection
	In             types.Type
	PredefinedFunc int
	SourceCode     string
	Out            types.Type
}

// Connection returns the connection of the mapping.
func (mapping *Mapping) Connection() *Connection {
	mapping.mu.Lock()
	c := mapping.connection
	mapping.mu.Unlock()
	return c
}
