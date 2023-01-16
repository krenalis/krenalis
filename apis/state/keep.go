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
	"log"
	"strings"
	"sync"
	"time"

	"chichi/apis/postgres"
	"chichi/apis/types"
	"chichi/apis/warehouses"
	"chichi/apis/warehouses/clickhouse"
	"chichi/apis/warehouses/postgresql"

	"github.com/google/uuid"
)

// logNotifications controls the logging of notifications on the log.
const logNotifications = false

// AddListener adds a notification listener.
// It panics if it is called after Keep is called.
func (state *State) AddListener(listener any) {
	state.mu.Lock()
	keeping := state.keeping
	state.mu.Unlock()
	if keeping {
		panic(fmt.Sprintf("state: cannot call AddListener after Keep has been called"))
	}
	switch l := listener.(type) {
	case func(AddConnectionNotification):
		state.listeners.AddConnection = append(state.listeners.AddConnection, l)
	case func(AddImportInProgressNotification):
		state.listeners.AddImportInProgress = append(state.listeners.AddImportInProgress, l)
	case func(DeleteConnectionNotification):
		state.listeners.DeleteConnection = append(state.listeners.DeleteConnection, l)
	case func(SetConnectionSettingsNotification):
		state.listeners.SetConnectionSettings = append(state.listeners.SetConnectionSettings, l)
	case func(SetConnectionStatusNotification):
		state.listeners.SetConnectionStatus = append(state.listeners.SetConnectionStatus, l)
	case func(SetConnectionStreamNotification):
		state.listeners.SetConnectionStream = append(state.listeners.SetConnectionStream, l)
	case func(SetConnectionUserQueryNotification):
		state.listeners.SetConnectionUserQuery = append(state.listeners.SetConnectionUserQuery, l)
	case func(SetWarehouseSettingsNotification):
		state.listeners.SetWarehouseSettings = append(state.listeners.SetWarehouseSettings, l)
	default:
		panic(fmt.Sprintf("state: unexpected listener type %T", listener))
	}
}

// Keep keeps the state updated.
func (state *State) Keep() {
	state.mu.Lock()
	state.keeping = true
	state.mu.Unlock()
	go state.keep()
}

// keep keeps the state in sync with the database. It is called in its own
// goroutine.
func (state *State) keep() {

	for {
		n := <-state.notifications
		if logNotifications {
			log.Printf("[info] received notification from pid %d and name %q : %s",
				n.PID, n.Name, n.Payload)
		}
		if !state.syncing && n.Name != "LoadState" {
			continue
		}
		switch n.Name {
		case "AddConnection":
			state.addConnection(n)
		case "AddConnectionKey":
			state.addConnectionKey(n)
		case "AddImportInProgress":
			state.addImportInProgress(n)
		case "DeleteConnection":
			state.deleteConnection(n)
		case "EndImport":
			state.endImport(n)
		case "LoadState":
			state.loadState(n)
		case "RevokeConnectionKey":
			state.revokeConnectionKey(n)
		case "SetConnectionSettings":
			state.setConnectionSettings(n)
		case "SetConnectionStorage":
			state.setConnectionStorage(n)
		case "SetConnectionStream":
			state.setConnectionStream(n)
		case "SetConnectionStatus":
			state.setConnectionStatus(n)
		case "SetConnectionMappings":
			state.setConnectionMappings(n)
		case "SetConnectionUserQuery":
			state.setConnectionUserQuery(n)
		case "SetConnectionUserSchema":
			state.setConnectionUserSchema(n)
		case "SetResource":
			state.setResource(n)
		case "SetWarehouseSettings":
			state.setWarehouseSettings(n)
		case "SetWorkspaceSchemas":
			state.setWorkspaceSchemas(n)
		default:
			log.Printf("[warning] unknown notification %q received from %d: %s", n.Name, n.PID, n.Payload)
		}
	}

}

// decodeNotification decodes a state change notification.
func decodeStateNotification(n postgres.Notification, e any) bool {
	err := json.NewDecoder(strings.NewReader(n.Payload)).Decode(&e)
	if err != nil {
		log.Printf("[error] cannot unmarshal notification %s from %d: %s", n.Name, n.PID, err)
		return false
	}
	return true
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
	// Update the connections.
	for _, connection := range ws.connections {
		if connection.storage == c {
			connection.mu.Lock()
			connection.storage = cc
			connection.mu.Unlock()
		}
		if connection.stream == c {
			connection.mu.Lock()
			connection.stream = cc
			connection.mu.Unlock()
		}
		if imp := connection.importInProgress; imp != nil {
			if imp.connection == c {
				imp.mu.Lock()
				imp.connection = cc
				imp.mu.Unlock()
			}
			if imp.storage == c {
				imp.mu.Lock()
				imp.storage = cc
				imp.mu.Unlock()
			}
		}
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

// AddConnectionNotification is the notification event sent when a new
// connection is added.
type AddConnectionNotification struct {
	Workspace int            // workspace identifier
	ID        int            // identifier
	Name      string         // name
	Role      ConnectionRole // role
	Connector int            // connector identifier
	Storage   int            // storage identifier, can be zero
	Stream    int            // stream identifier, can be zero
	Resource  struct {       // resource.
		ID           int       // identifier, can be zero
		Code         string    // code, can be empty.
		AccessToken  string    // access token, can be empty.
		RefreshToken string    // refresh token, can be empty.
		ExpiresIn    time.Time // expiration time, can be the zero time.
	}
	WebsiteHost string // website host in form host:port
	Key         string // server key to add
}

// addConnection adds a new connection.
func (state *State) addConnection(n postgres.Notification) {
	e := AddConnectionNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	workspace, _ := state.workspaces[e.Workspace]
	connector, _ := state.connectors[e.Connector]
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
		connector:   connector,
		storage:     state.connections[e.Storage],
		stream:      state.connections[e.Stream],
		resource:    r,
		WebsiteHost: e.WebsiteHost,
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

// AddConnectionKeyNotification is the notification event sent when a
// connection key is added.
type AddConnectionKeyNotification struct {
	Connection   int
	Value        string
	CreationTime time.Time
}

// addConnectionKey adds a new connection key.
func (state *State) addConnectionKey(n postgres.Notification) {
	e := AddConnectionKeyNotification{}
	if !decodeStateNotification(n, &e) {
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

// AddImportInProgressNotification is the notification event sent when an
// import in progress is added.
type AddImportInProgressNotification struct {
	ID         int
	Connection int
	Storage    int
	Reimport   bool
	StartTime  time.Time
}

// addImportInProgress adds an import in progress.
func (state *State) addImportInProgress(n postgres.Notification) {
	e := AddImportInProgressNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	c := state.connections[e.Connection]
	c.mu.Lock()
	c.importInProgress = &ImportInProgress{
		mu:         new(sync.Mutex),
		ID:         e.ID,
		connection: c,
		storage:    state.connections[e.Storage],
		Reimport:   e.Reimport,
		StartTime:  e.StartTime,
	}
	c.mu.Unlock()
	for _, listener := range state.listeners.AddImportInProgress {
		listener(e)
	}
}

// DeleteConnectionNotification is the notification event sent when a
// connection is deleted.
type DeleteConnectionNotification struct {
	ID int
}

// deleteConnection deletes a connection.
func (state *State) deleteConnection(n postgres.Notification) {
	e := DeleteConnectionNotification{}
	if !decodeStateNotification(n, &e) {
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
		if c.stream == connection {
			c.mu.Lock()
			c.stream = nil
			c.mu.Unlock()
		}
	}
	for _, listener := range state.listeners.DeleteConnection {
		listener(e)
	}
}

// EndImportNotification is the notification event sent when an import ends.
type EndImportNotification struct {
	ID int
}

// endImport ends an import.
func (state *State) endImport(n postgres.Notification) {
	e := EndImportNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	for _, c := range state.connections {
		if imp := c.importInProgress; imp != nil && imp.ID == e.ID {
			state.replaceConnection(c.ID, func(c *Connection) {
				c.importInProgress = nil
			})
			break
		}
	}
}

// LoadStateNotification is the notification sent when a state is loaded.
type LoadStateNotification struct {
	ID uuid.UUID
}

// loadState loads the state.
func (state *State) loadState(n postgres.Notification) {
	e := LoadStateNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	if e.ID == state.id {
		state.syncing = true
	}
	return
}

// RevokeConnectionKeyNotification is the notification event sent when a
// connection key is revoked.
type RevokeConnectionKeyNotification struct {
	Connection int
	Value      string
}

// revokeConnectionKey revokes a connection key.
func (state *State) revokeConnectionKey(n postgres.Notification) {
	e := RevokeConnectionKeyNotification{}
	if !decodeStateNotification(n, &e) {
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

// SetConnectionSettingsNotification is the notification event sent when the
// settings of a connection is changed.
type SetConnectionSettingsNotification struct {
	Connection int
	Settings   []byte
}

// setConnectionSettings sets the settings of a connection.
func (state *State) setConnectionSettings(n postgres.Notification) {
	e := SetConnectionSettingsNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	state.replaceConnection(e.Connection, func(c *Connection) {
		c.Settings = e.Settings
	})
	for _, listener := range state.listeners.SetConnectionSettings {
		listener(e)
	}
}

// SetConnectionStatusNotification is the notification event sent when a
// connection status is changed.
type SetConnectionStatusNotification struct {
	ID      int
	Enabled bool
}

// setConnectionStatus changes a connection status.
func (state *State) setConnectionStatus(n postgres.Notification) {
	e := SetConnectionStatusNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	state.replaceConnection(e.ID, func(c *Connection) {
		c.Enabled = e.Enabled
	})
	for _, listener := range state.listeners.SetConnectionStatus {
		listener(e)
	}
}

// SetConnectionStorageNotification is the notification event sent when the
// settings of a connection is changed.
type SetConnectionStorageNotification struct {
	Connection int
	Storage    int
}

// setConnectionStorages sets the storage of a connection.
func (state *State) setConnectionStorage(n postgres.Notification) {
	e := SetConnectionStorageNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	c := state.connections[e.Connection]
	storage := state.connections[e.Storage]
	c.mu.Lock()
	c.storage = storage
	c.mu.Unlock()
}

// SetConnectionStreamNotification is the notification event sent when the
// settings of a connection is changed.
type SetConnectionStreamNotification struct {
	Connection int
	Stream     int
}

// setConnectionStream sets the stream of a connection.
func (state *State) setConnectionStream(n postgres.Notification) {
	e := SetConnectionStreamNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	c := state.connections[e.Connection]
	stream := state.connections[e.Stream]
	c.mu.Lock()
	c.stream = stream
	c.mu.Unlock()
	for _, listener := range state.listeners.SetConnectionStream {
		listener(e)
	}
}

// SetConnectionMappingsNotification is the notification event sent when the
// mappings of a connection are saved.
type SetConnectionMappingsNotification struct {
	Connection int
	Mappings   []*Mapping
}

// setConnectionMappings sets the mappings of a connection.
func (state *State) setConnectionMappings(n postgres.Notification) {
	e := SetConnectionMappingsNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	c := state.connections[e.Connection]
	c.mu.Lock()
	c.mappings = e.Mappings
	c.mu.Unlock()
}

// SetConnectionUserQueryNotification is the notification event sent when a
// user query of a connection is changed.
type SetConnectionUserQueryNotification struct {
	Connection int
	Query      string
}

// setConnectionUserQuery sets the user query of a connection.
func (state *State) setConnectionUserQuery(n postgres.Notification) {
	e := SetConnectionUserQueryNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	state.replaceConnection(e.Connection, func(c *Connection) {
		c.UsersQuery = e.Query
	})
	for _, listener := range state.listeners.SetConnectionUserQuery {
		listener(e)
	}
}

// SetConnectionUserSchemaNotification is the notification event sent when the
// user schema of a connection is changed.
type SetConnectionUserSchemaNotification struct {
	Connection int
	Schema     types.Type
}

// setConnectionUserSchema sets the user schema of a connection.
func (state *State) setConnectionUserSchema(n postgres.Notification) {
	e := SetConnectionUserSchemaNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	state.replaceConnection(e.Connection, func(c *Connection) {
		c.Schema = e.Schema
	})
}

type SetResourceNotification struct {
	ID           int
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Time
}

// setResource sets a resource.
func (state *State) setResource(n postgres.Notification) {
	e := SetResourceNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	state.replaceResource(e.ID, func(r *Resource) {
		r.AccessToken = e.AccessToken
		r.RefreshToken = e.RefreshToken
		r.ExpiresIn = e.ExpiresIn
	})
}

// SetWarehouseSettingsNotification is the notification event sent when the
// settings of a data warehouse are changed.
type SetWarehouseSettingsNotification struct {
	Workspace int
	Type      WarehouseType
	Settings  json.RawMessage `json:",omitempty"`
}

// setWarehouseSettings sets the settings of a data warehouse.
func (state *State) setWarehouseSettings(n postgres.Notification) {
	e := SetWarehouseSettingsNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	disconnected := state.workspaces[e.Workspace].Warehouse
	if e.Settings == nil {
		state.replaceWorkspace(e.Workspace, func(w *Workspace) {
			w.Warehouse = nil
		})
	} else {
		var err error
		state.replaceWorkspace(e.Workspace, func(w *Workspace) {
			w.Warehouse, err = openWarehouse(e.Type, e.Settings)
		})
		if err != nil {
			log.Printf("[error] cannot open data warehouse of workspace %d: %s", e.Workspace, err)
		}
	}
	// Close the disconnected warehouse.
	if disconnected != nil {
		go func() {
			err := disconnected.Close()
			if err != nil {
				// TODO(marco): write the error into a workspace specific log
				log.Printf("error occurred disconnecting the warehouse: %s", err)
			}
		}()
	}
	for _, listener := range state.listeners.SetWarehouseSettings {
		listener(e)
	}
}

// SetWorkspaceSchemasNotification is the notification event sent when schemas
// of a workspace are changed.
type SetWorkspaceSchemasNotification struct {
	Workspace int
	Schemas   map[string]*types.Type
}

// setWorkspaceSchemas sets the schemas of a workspace.
func (state *State) setWorkspaceSchemas(n postgres.Notification) {
	e := SetWorkspaceSchemasNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	var unchanged []string
	for name, typ := range e.Schemas {
		if typ == nil {
			unchanged = append(unchanged, name)
		}
	}
	state.replaceWorkspace(e.Workspace, func(w *Workspace) {
		for _, name := range unchanged {
			e.Schemas[name] = w.Schemas[name]
		}
		w.Schemas = e.Schemas
	})
}

// openWarehouse opens a data warehouse with the given type and settings.
// It returns an error if typ or settings are not valid.
func openWarehouse(typ WarehouseType, settings []byte) (warehouses.Warehouse, error) {
	switch typ {
	case BigQuery, Redshift, Snowflake:
		return nil, fmt.Errorf("warehouse type %s is not yet supported", typ)
	case PostgreSQL:
		return postgresql.Open(settings)
	case ClickHouse:
		return clickhouse.Open(settings)
	}
	return nil, fmt.Errorf("warehouse type %d is not valid", typ)
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
