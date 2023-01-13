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
	"log"
	"strings"
	"sync"
	"time"

	"chichi/apis/postgres"
	"chichi/apis/types"
	"chichi/apis/warehouses"
	"chichi/apis/warehouses/clickhouse"
	"chichi/apis/warehouses/postgresql"
)

// logNotifications controls the logging of notifications on the log.
const logNotifications = false

// ReloadSchemas is called when a connection schema should be reloaded.
var ReloadSchemas func(connection int) error

// StartImport is called when an import should be started.
var StartImport func(imp *ImportInProgress)

// Keep keeps the state returns it.
func Keep(ctx context.Context, db *postgres.DB) (*State, error) {
	s := &State{
		db:          db,
		mu:          new(sync.Mutex),
		accounts:    map[int]*Account{},
		connectors:  map[int]*Connector{},
		workspaces:  map[int]*Workspace{},
		connections: map[int]*Connection{},
		resources:   map[int]*Resource{},
	}
	err := s.load()
	if err != nil {
		return nil, err
	}
	notifications := db.ListenToNotifications(ctx)
	go s.keep(ctx, notifications)
	return s, nil
}

// keep keeps state in sync with the database. It is called in its own
// goroutine.
func (s *State) keep(ctx context.Context, notifications <-chan postgres.Notification) {

	for {
		n := <-notifications
		if logNotifications {
			log.Printf("[info] received notification from pid %d and name %q : %s",
				n.PID, n.Name, n.Payload)
		}
		switch n.Name {
		case "AddConnection":
			s.addConnection(n)
		case "AddConnectionKey":
			s.addConnectionKey(n)
		case "DeleteConnection":
			s.deleteConnection(n)
		case "EndImport":
			s.endImport(n)
		case "RevokeConnectionKey":
			s.revokeConnectionKey(n)
		case "SetConnectionSettings":
			s.setConnectionSettings(n)
		case "SetConnectionStorage":
			s.setConnectionStorage(n)
		case "SetConnectionStream":
			s.setConnectionStream(n)
		case "SetConnectionMappings":
			s.setConnectionMappings(n)
		case "SetConnectionUserQuery":
			s.setConnectionUserQuery(n)
		case "SetConnectionUserSchema":
			s.setConnectionUserSchema(n)
		case "SetResource":
			s.setResource(n)
		case "SetWorkspaceSchemas":
			s.setWorkspaceSchemas(n)
		case "SetWorkspaceWarehouse":
			s.setWorkspaceWarehouse(n)
		case "StartImport":
			s.startImport(n)
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
func (s *State) replaceConnection(id int, f func(*Connection)) *Connection {
	c := s.connections[id]
	cc := new(Connection)
	*cc = *c
	f(cc)
	s.mu.Lock()
	s.connections[id] = cc
	s.mu.Unlock()
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
func (s *State) replaceResource(id int, f func(*Resource)) *Resource {
	r := s.resources[id]
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
	s.resources[id] = rr
	return rr
}

// replaceWorkspace calls the function f passing a copy of the workspace with
// identifier id. After f is returned, it replaces the workspace with its
// copy in the state and returns the latter.
func (s *State) replaceWorkspace(id int, f func(*Workspace)) *Workspace {
	w := s.workspaces[id]
	ww := new(Workspace)
	*ww = *w
	f(ww)
	s.mu.Lock()
	s.workspaces[id] = ww
	s.mu.Unlock()
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
func (s *State) addConnection(n postgres.Notification) {
	e := AddConnectionNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	workspace, _ := s.workspaces[e.Workspace]
	connector, _ := s.connectors[e.Connector]
	var r *Resource
	if connector.OAuth != nil {
		if _, ok := s.resources[e.Resource.ID]; ok {
			if e.Resource.AccessToken != "" {
				r = s.replaceResource(e.Resource.ID, func(r *Resource) {
					r.AccessToken = e.Resource.AccessToken
					r.RefreshToken = e.Resource.RefreshToken
					r.ExpiresIn = e.Resource.ExpiresIn
				})
				// Update the resources.
				s.mu.Lock()
				s.resources[r.ID] = r
				s.mu.Unlock()
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
			s.mu.Lock()
			s.resources[r.ID] = r
			s.mu.Unlock()
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
		storage:     s.connections[e.Storage],
		stream:      s.connections[e.Stream],
		resource:    r,
		WebsiteHost: e.WebsiteHost,
	}
	if e.Key != "" {
		c.Keys = []string{e.Key}
	}
	s.mu.Lock()
	s.connections[e.ID] = c
	s.mu.Unlock()
	// Update the workspace.
	workspace.mu.Lock()
	workspace.connections[c.ID] = c
	workspace.mu.Unlock()
	if connector.Type == AppType {
		// TODO(marco) only one server should reload the schema.
		if ReloadSchemas == nil {
			panic("state.ReloadSchemas is nil")
		}
		go func() {
			err := ReloadSchemas(c.ID)
			if err != nil {
				log.Printf("[error] cannot reload schema for connection %d: %s", c.ID, err)
			}
		}()
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
func (s *State) addConnectionKey(n postgres.Notification) {
	e := AddConnectionKeyNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	s.replaceConnection(e.Connection, func(c *Connection) {
		keys := make([]string, len(c.Keys)+1)
		copy(keys, c.Keys)
		keys[len(c.Keys)] = e.Value
		c.Keys = keys
	})
}

// DeleteConnectionNotification is the notification event sent when a
// connection is deleted.
type DeleteConnectionNotification struct {
	ID int
}

// deleteConnection deletes a connection.
func (s *State) deleteConnection(n postgres.Notification) {
	e := DeleteConnectionNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	connection := s.connections[e.ID]
	s.mu.Lock()
	delete(s.connections, e.ID)
	s.mu.Unlock()
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
}

// EndImportNotification is the notification event sent when an import ends.
type EndImportNotification struct {
	ID int
}

// endImport ends an import.
func (s *State) endImport(n postgres.Notification) {
	e := EndImportNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	for _, c := range s.connections {
		if imp := c.importInProgress; imp.ID == e.ID {
			s.replaceConnection(c.ID, func(c *Connection) {
				c.importInProgress = nil
			})
			break
		}
	}
}

// RevokeConnectionKeyNotification is the notification event sent when a
// connection key is revoked.
type RevokeConnectionKeyNotification struct {
	Connection int
	Value      string
}

// revokeConnectionKey revokes a connection key.
func (s *State) revokeConnectionKey(n postgres.Notification) {
	e := RevokeConnectionKeyNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	s.replaceConnection(e.Connection, func(c *Connection) {
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
}

// SetConnectionSettingsNotification is the notification event sent when the
// settings of a connection is changed.
type SetConnectionSettingsNotification struct {
	Connection int
	Settings   []byte
}

// setConnectionSettings sets the settings of a connection.
func (s *State) setConnectionSettings(n postgres.Notification) {
	e := SetConnectionSettingsNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	s.replaceConnection(e.Connection, func(c *Connection) {
		c.Settings = e.Settings
	})
}

// SetConnectionStorageNotification is the notification event sent when the
// settings of a connection is changed.
type SetConnectionStorageNotification struct {
	Connection int
	Storage    int
}

// setConnectionStorages sets the storage of a connection.
func (s *State) setConnectionStorage(n postgres.Notification) {
	e := SetConnectionStorageNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	c := s.connections[e.Connection]
	storage := s.connections[e.Storage]
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
func (s *State) setConnectionStream(n postgres.Notification) {
	e := SetConnectionStreamNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	c := s.connections[e.Connection]
	stream := s.connections[e.Stream]
	c.mu.Lock()
	c.stream = stream
	c.mu.Unlock()
}

// SetConnectionMappingsNotification is the notification event sent when the
// mappings of a connection are saved.
type SetConnectionMappingsNotification struct {
	Connection int
	Mappings   []*Mapping
}

// setConnectionMappings sets the mappings of a connection.
func (s *State) setConnectionMappings(n postgres.Notification) {
	e := SetConnectionMappingsNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	c := s.connections[e.Connection]
	c.mu.Lock()
	c.mappings = e.Mappings
	c.mu.Unlock()
}

// SetUserQueryNotification is the notification event sent when a user query of
// a connection is changed.
type SetUserQueryNotification struct {
	Connection int
	Query      string
}

// setConnectionUserQuery sets the user query of a connection.
func (s *State) setConnectionUserQuery(n postgres.Notification) {
	e := SetUserQueryNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	c := s.replaceConnection(e.Connection, func(c *Connection) {
		c.UsersQuery = e.Query
	})
	// TODO(marco) only one server should reload the schema.
	if ReloadSchemas == nil {
		panic("state.ReloadSchemas is nil")
	}
	go func() {
		err := ReloadSchemas(c.ID)
		if err != nil {
			log.Printf("[error] cannot reload schema for connection %d: %s", c.ID, err)
		}
	}()
}

// SetConnectionUserSchemaNotification is the notification event sent when the
// user schema of a connection is changed.
type SetConnectionUserSchemaNotification struct {
	Connection int
	Schema     types.Type
}

// setConnectionUserSchema sets the user schema of a connection.
func (s *State) setConnectionUserSchema(n postgres.Notification) {
	e := SetConnectionUserSchemaNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	s.replaceConnection(e.Connection, func(c *Connection) {
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
func (s *State) setResource(n postgres.Notification) {
	e := SetResourceNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	s.replaceResource(e.ID, func(r *Resource) {
		r.AccessToken = e.AccessToken
		r.RefreshToken = e.RefreshToken
		r.ExpiresIn = e.ExpiresIn
	})
}

// SetWorkspaceSchemasNotification is the notification event sent when schemas
// of a workspace are changed.
type SetWorkspaceSchemasNotification struct {
	Workspace int
	Schemas   map[string]*types.Type
}

// setWorkspaceSchemas sets the schemas of a workspace.
func (s *State) setWorkspaceSchemas(n postgres.Notification) {
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
	s.replaceWorkspace(e.Workspace, func(w *Workspace) {
		for _, name := range unchanged {
			e.Schemas[name] = w.Schemas[name]
		}
		w.Schemas = e.Schemas
	})
}

// SetWorkspaceWarehouseNotification is the notification event sent when a
// workspace warehouse is changed.
type SetWorkspaceWarehouseNotification struct {
	Workspace int
	Warehouse *NotifiedWarehouse
}

type NotifiedWarehouse struct {
	Type     WarehouseType
	Settings json.RawMessage `json:",omitempty"`
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

// setWorkspaceWarehouse sets the warehouse of a workspace.
func (s *State) setWorkspaceWarehouse(n postgres.Notification) {
	e := SetWorkspaceWarehouseNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	disconnected := s.workspaces[e.Workspace].Warehouse
	if e.Warehouse != nil {
		var err error
		s.replaceWorkspace(e.Workspace, func(w *Workspace) {
			w.Warehouse, err = openWarehouse(e.Warehouse.Type, e.Warehouse.Settings)
		})
		if err != nil {
			log.Printf("[error] cannot open data warehouse of workspace %d: %s", e.Workspace, err)
		}
	} else {
		s.replaceWorkspace(e.Workspace, func(w *Workspace) {
			w.Warehouse = nil
		})
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
}

// StartImportNotification is the notification event sent when an import
// starts.
type StartImportNotification struct {
	ID         int
	Connection int
	Storage    int
	Reimport   bool
	StartTime  time.Time
}

// startImport starts an import.
func (s *State) startImport(n postgres.Notification) {
	e := StartImportNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	c := s.connections[e.Connection]
	c.mu.Lock()
	c.importInProgress = &ImportInProgress{
		mu:         new(sync.Mutex),
		ID:         e.ID,
		connection: c,
		storage:    s.connections[e.Storage],
		Reimport:   e.Reimport,
		StartTime:  e.StartTime,
	}
	c.mu.Unlock()
	// Start the import.
	// TODO(marco): only one server should starts the import.
	go StartImport(c.importInProgress)
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
