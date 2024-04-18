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
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/apis/postgres"
	"github.com/open2b/chichi/types"

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
	syncing          bool // reports whether the keeper has started synchronizing the state.
	election         election
	organizations    map[int]*Organization
	connectors       map[int]*Connector
	workspaces       map[int]*Workspace
	connections      map[int]*Connection
	connectionsByKey map[string]*Connection
	actions          map[int]*Action
	resources        map[int]*Resource
	notifications    struct {
		channel <-chan notification
		acks    *acks
		stop    func()
	}
	listeners struct {
		AddAction               []func(AddAction)
		AddConnection           []func(AddConnection)
		DeleteAction            []func(DeleteAction)
		DeleteConnection        []func(DeleteConnection)
		DeleteWorkspace         []func(DeleteWorkspace)
		ElectLeader             []func(ElectLeader)
		ExecuteAction           []func(ExecuteAction)
		SetAction               []func(SetAction)
		SetActionSchedulePeriod []func(SetActionSchedulePeriod)
		SetConnection           []func(SetConnection)
		SetConnectionSettings   []func(SetConnectionSettings)
		SetWarehouse            []func(SetWarehouse)
		SetWorkspace            []func(SetWorkspace)
	}
	close struct {
		ctx       context.Context
		CancelCtx context.CancelFunc
		sync.WaitGroup
	}
}

// New returns a *State instance.
func New(db *postgres.DB) (*State, error) {

	id, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	state := &State{
		id:               id,
		db:               db,
		mu:               new(sync.Mutex),
		organizations:    map[int]*Organization{},
		connectors:       map[int]*Connector{},
		workspaces:       map[int]*Workspace{},
		connections:      map[int]*Connection{},
		connectionsByKey: map[string]*Connection{},
		actions:          map[int]*Action{},
		resources:        map[int]*Resource{},
	}

	// Listen to notifications.
	state.notifications.acks = newAcks()
	state.notifications.channel, state.notifications.stop = state.listenToNotifications()

	state.close.ctx, state.close.CancelCtx = context.WithCancel(context.Background())

	err = state.load()
	if err != nil {
		state.notifications.stop()
		return nil, err
	}

	return state, nil
}

// Organization returns the organization with identifier id.
// The boolean return value reports whether the organization exists.
func (state *State) Organization(id int) (*Organization, bool) {
	state.mu.Lock()
	a, ok := state.organizations[id]
	state.mu.Unlock()
	return a, ok
}

// Organizations returns all organizations.
func (state *State) Organizations() []*Organization {
	state.mu.Lock()
	organizations := make([]*Organization, len(state.organizations))
	i := 0
	for _, organization := range state.organizations {
		organizations[i] = organization
		i++
	}
	state.mu.Unlock()
	sort.Slice(organizations, func(i, j int) bool {
		return organizations[i].ID < organizations[j].ID
	})
	return organizations
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

// Close closes the state.
func (state *State) Close() {
	state.close.CancelCtx()
	state.close.Wait()
	state.notifications.stop()
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

// Resource returns the resource with identifier id.
// The boolean return value reports whether the resource exists.
func (state *State) Resource(id int) (*Resource, bool) {
	// TODO(marco): optimize.
	for _, o := range state.Organizations() {
		for _, ws := range o.Workspaces() {
			if r, ok := ws.Resource(id); ok {
				return r, true
			}
		}
	}
	return nil, false
}

// Workspace returns the workspace with identifier id.
// The boolean return value reports whether the workspace exists.
func (state *State) Workspace(id int) (*Workspace, bool) {
	state.mu.Lock()
	ws, ok := state.workspaces[id]
	state.mu.Unlock()
	return ws, ok
}

// Organization represents an organization.
type Organization struct {
	mu         *sync.Mutex
	workspaces map[int]*Workspace
	ID         int
	Name       string
}

// Workspace returns the workspace of the organization with identifier id.
// The boolean return value reports whether the workspace exists.
func (organization *Organization) Workspace(id int) (*Workspace, bool) {
	organization.mu.Lock()
	w, ok := organization.workspaces[id]
	organization.mu.Unlock()
	return w, ok
}

// Workspaces returns all the workspaces of the organization.
func (organization *Organization) Workspaces() []*Workspace {
	organization.mu.Lock()
	workspaces := make([]*Workspace, len(organization.workspaces))
	i := 0
	for _, w := range organization.workspaces {
		workspaces[i] = w
		i++
	}
	organization.mu.Unlock()
	return workspaces
}

type Warehouse struct {
	Type     WarehouseType
	Settings json.RawMessage
}

// Workspace represents a workspace.
type Workspace struct {
	mu                  *sync.Mutex
	Warehouse           *Warehouse
	connections         map[int]*Connection
	ID                  int
	organization        *Organization
	Name                string
	UsersSchema         types.Type
	resources           map[int]*Resource
	Identifiers         []string
	PrivacyRegion       PrivacyRegion
	DisplayedProperties DisplayedProperties
}

// Organization returns the organization of the workspace.
func (workspace *Workspace) Organization() *Organization {
	workspace.mu.Lock()
	organization := workspace.organization
	workspace.mu.Unlock()
	return organization
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

// DisplayedProperties represents the displayed properties.
type DisplayedProperties struct {
	Image       string
	FirstName   string
	LastName    string
	Information string
}

// TimeLayouts represents the layouts used to format DateTime, Date, and Time
// values.
type TimeLayouts struct {
	DateTime string
	Date     string
	Time     string
}

// Connector represents a connector.
type Connector struct {
	ID                         int
	Name                       string
	SourceDescription          string
	DestinationDescription     string
	TermForUsers               string
	TermForGroups              string
	Type                       ConnectorType
	Targets                    ConnectorTargets
	SendingMode                *SendingMode
	HasSheets                  bool
	HasUI                      bool
	IdentityIDLabel            string
	SuggestedDisplayedProperty string
	Icon                       string
	TimeLayouts                TimeLayouts
	FileExtension              string
	SampleQuery                string
	WebhooksPer                WebhooksPer
	OAuth                      *OAuth
}

// ConnectorTargets represents the targets of a connector.
type ConnectorTargets int

const (
	EventsFlag = 1 << iota
	UsersFlag
	GroupsFlag
)

// Contains reports whether t contains the given target.
func (t ConnectorTargets) Contains(target Target) bool {
	switch target {
	case Events:
		return t&EventsFlag != 0
	case Users:
		return t&UsersFlag != 0
	case Groups:
		return t&GroupsFlag != 0
	}
	panic("invalid target")
}

// ConnectorType represents a connector type.
type ConnectorType int

const (
	AppType ConnectorType = iota + 1
	DatabaseType
	FileType
	FileStorageType
	MobileType
	ServerType
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
	case "FileStorage":
		t = FileStorageType
	case "Mobile":
		t = MobileType
	case "Server":
		t = ServerType
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
	case FileStorageType:
		return "FileStorage", nil
	case MobileType:
		return "Mobile", nil
	case ServerType:
		return "Server", nil
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
	WebhooksPerConnection
	WebhooksPerConnector
	WebhooksPerResource
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
	case "Connection":
		p = WebhooksPerConnection
	case "Connector":
		p = WebhooksPerConnector
	case "Resource":
		p = WebhooksPerResource
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
// It returns an error if typ is not a valid Role.
func (per WebhooksPer) Value() (driver.Value, error) {
	switch per {
	case WebhooksPerNone:
		return "None", nil
	case WebhooksPerConnection:
		return "Connection", nil
	case WebhooksPerConnector:
		return "Connector", nil
	case WebhooksPerResource:
		return "Resource", nil
	}
	return nil, fmt.Errorf("not a valid WebhooksPer: %d", per)
}

// An OAuth represents OAuth data required to authenticate with a connector.
type OAuth struct {
	chichi.OAuth
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

// Strategy represents a strategy. Can be "AB-C", "ABC", "A-B-C", and "AC-B".
type Strategy string

// Connection represents a connection.
type Connection struct {
	mu               *sync.Mutex
	organization     *Organization
	workspace        *Workspace
	ID               int
	Name             string
	Role             Role
	Enabled          bool
	connector        *Connector
	resource         *Resource
	Strategy         *Strategy
	SendingMode      *SendingMode
	WebsiteHost      string
	EventConnections []int
	Keys             []string
	Settings         []byte
	UsersQuery       string
	actions          map[int]*Action
	Health           Health
}

// Organization returns the organization of the connection.
func (connection *Connection) Organization() *Organization {
	connection.mu.Lock()
	o := connection.organization
	connection.mu.Unlock()
	return o
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

// Role represents a role.
type Role int

const (
	Source      Role = iota + 1 // source
	Destination                 // destination
)

// Scan implements the sql.Scanner interface.
func (role *Role) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an state.Role value", src)
	}
	var r Role
	switch s {
	case "Source":
		r = Source
	case "Destination":
		r = Destination
	default:
		return fmt.Errorf("invalid state.Role: %s", s)
	}
	*role = r
	return nil
}

// String returns the string representation of role.
// It panics if role is not a valid Role value.
func (role Role) String() string {
	switch role {
	case Source:
		return "Source"
	case Destination:
		return "Destination"
	}
	panic("invalid connection role")
}

// Value implements driver.Valuer interface.
// It returns an error if role is not a valid Role.
func (role Role) Value() (driver.Value, error) {
	switch role {
	case Source:
		return "Source", nil
	case Destination:
		return "Destination", nil
	}
	return nil, fmt.Errorf("not a valid Role: %d", role)
}

// SendingMode represents a sending mode.
type SendingMode string

const (
	Cloud    SendingMode = "Cloud"
	Device   SendingMode = "Device"
	Combined SendingMode = "Combined"
)

func (sm SendingMode) Contains(mode SendingMode) bool {
	return sm == Combined || sm == mode
}

// Compression represents the compression of a file connection.
type Compression string

const (
	NoCompression     Compression = ""
	ZipCompression    Compression = "Zip"
	GzipCompression   Compression = "Gzip"
	SnappyCompression Compression = "Snappy"
)

// ContentType returns the content type to use for a file compressed with
// compression c. It returns an empty string if c is NoCompression.
func (c Compression) ContentType() string {
	switch c {
	case NoCompression:
		return ""
	case ZipCompression:
		return "application/zip"
	case GzipCompression:
		return "application/gzip"
	case SnappyCompression:
		return "application/x-snappy-framed"
	}
	panic(fmt.Sprintf("invalid state.Compression: %s", c))
}

// Ext returns the file extension to use when the file is compressed with
// compression c. It returns an empty string if c is NoCompression.
func (c Compression) Ext() string {
	switch c {
	case NoCompression:
		return ""
	case ZipCompression:
		return ".zip"
	case GzipCompression:
		return ".gz"
	case SnappyCompression:
		return ".sz"
	}
	panic(fmt.Sprintf("invalid state.Compression: %s", c))
}

// Target represents a target.
type Target int

const (
	Events Target = iota + 1
	Users
	Groups
)

// Scan implements the sql.Scanner interface.
func (target *Target) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an state.Target value", src)
	}
	switch s {
	case "Events":
		*target = Events
	case "Users":
		*target = Users
	case "Groups":
		*target = Groups
	default:
		return fmt.Errorf("invalid state.Target: %s", s)
	}
	return nil
}

// String returns the string representation of target.
// It panics if target is not a valid Target value.
func (target Target) String() string {
	switch target {
	case Events:
		return "Events"
	case Users:
		return "Users"
	case Groups:
		return "Groups"
	}
	panic("invalid target")
}

// Value implements driver.Valuer interface.
// It returns an error if target is not a valid Target.
func (target Target) Value() (driver.Value, error) {
	switch target {
	case Events:
		return "Events", nil
	case Users:
		return "Users", nil
	case Groups:
		return "Groups", nil
	}
	return nil, fmt.Errorf("not a valid Target: %d", target)
}

// Cursor represents a cursor.
type Cursor struct {
	ID             string
	LastChangeTime time.Time
}

type Action struct {
	mu                      *sync.Mutex
	ID                      int
	connection              *Connection
	connector               *Connector
	execution               *ActionExecution
	Target                  Target
	Name                    string
	Enabled                 bool
	EventType               string
	ScheduleStart           int16
	SchedulePeriod          int16
	InSchema                types.Type
	OutSchema               types.Type
	Filter                  *Filter
	Transformation          Transformation
	Query                   string
	Path                    string
	Sheet                   string
	Compression             Compression
	Settings                []byte
	TableName               string
	IdentityProperty        string
	LastChangeTimeProperty  string
	LastChangeTimeFormat    string
	DisplayedProperty       string
	UserCursor              Cursor
	Health                  Health
	ExportMode              *ExportMode
	MatchingProperties      *MatchingProperties
	ExportOnDuplicatedUsers *bool
}

// Language represents a transformation language.
type Language int

const (
	JavaScript Language = iota
	Python
)

func (lang Language) String() string {
	switch lang {
	case JavaScript:
		return "JavaScript"
	case Python:
		return "Python"
	}
	panic("invalid language")
}

// Scan implements the sql.Scanner interface.
func (lang *Language) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an state.Language value", src)
	}
	switch s {
	case "JavaScript":
		*lang = JavaScript
	case "Python":
		*lang = Python
	default:
		return fmt.Errorf("invalid state.Language: %s", s)
	}
	return nil
}

// Transformation represents a transformation.
type Transformation struct {
	Mapping  map[string]string
	Function *TransformationFunction
}

// TransformationFunction represents a transformation function.
type TransformationFunction struct {
	Source   string
	Language Language
	Version  string
}

// ExportMode represents one of the three export modes.
type ExportMode string

const (
	CreateOnly     ExportMode = "CreateOnly"
	UpdateOnly     ExportMode = "UpdateOnly"
	CreateOrUpdate ExportMode = "CreateOrUpdate"
)

// MatchingProperties contains an internal property (belonging to the Golden
// Record) and an external property (belonging to the app) which are used to
// match identities of users in the data warehouse with users on the external
// app, during export.
type MatchingProperties struct {
	Internal string // the corresponding property is stored within the action's input schema.
	External types.Property
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

// Filter represents a filter.
type Filter struct {
	Logical    FilterLogical     // can be "all" or "any".
	Conditions []FilterCondition // cannot be empty.
}

// FilterLogical represents the logical operator of a filter.
// It can be "all" or "any".
type FilterLogical string

// FilterCondition represents the condition of a filter.
type FilterCondition struct {
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

// Connector returns the connector of the action.
func (action *Action) Connector() *Connector {
	action.mu.Lock()
	c := action.connector
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
