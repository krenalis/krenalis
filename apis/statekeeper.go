//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"chichi/apis/postgres"
	"chichi/apis/types"
	"chichi/apis/warehouses"
)

// logNotifications controls the logging of notifications on the log.
const logNotifications = false

// startStateKeeper starts the state keeper. It loads the state from the
// database and keeps it in sync.
func startStateKeeper(ctx context.Context, apis *APIs) error {
	s := stateKeeper{
		APIs:        apis,
		workspaces:  map[int]*Workspace{},
		connections: map[int]*Connection{},
		resources:   map[int]*Resource{},
	}
	err := s.loadState()
	if err != nil {
		return err
	}
	notifications := apis.db.ListenToNotifications(ctx)
	go s.keepState(ctx, notifications)
	return nil
}

// keepState keeps state in sync with the database. It is called in its own
// goroutine.
func (s *stateKeeper) keepState(ctx context.Context, notifications <-chan postgres.Notification) {

	for {
		n := <-notifications
		if logNotifications {
			log.Printf("[info] received notification from pid %d and name %q : %s",
				n.PID, n.Name, n.Payload)
		}
		switch n.Name {
		case "addConnection":
			s.addConnection(n)
		case "addDataType":
			s.addDataType(n)
		case "addEventType":
			s.addEventType(n)
		case "deleteConnection":
			s.deleteConnection(n)
		case "deleteDataType":
			s.deleteDataType(n)
		case "deleteEventType":
			s.deleteEventType(n)
		case "endImport":
			s.endImport(n)
		case "generateConnectionKey":
			s.generateConnectionKey(n)
		case "revokeConnectionKey":
			s.revokeConnectionKey(n)
		case "setConnectionSettings":
			s.setConnectionSettings(n)
		case "setConnectionStorage":
			s.setConnectionStorage(n)
		case "setConnectionStream":
			s.setConnectionStream(n)
		case "setConnectionMappings":
			s.setConnectionMappings(n)
		case "setConnectionUserQuery":
			s.setConnectionUserQuery(n)
		case "setConnectionUserSchema":
			s.setConnectionUserSchema(n)
		case "setDataTypeDescription":
			s.setDataTypeDescription(n)
		case "setDataTypeDefinition":
			s.setDataTypeDefinition(n)
		case "setEventTypeDescription":
			s.setEventTypeDescription(n)
		case "setEventTypeSchema":
			s.setEventTypeSchema(n)
		case "setResource":
			s.setResource(n)
		case "setWorkspaceWarehouse":
			s.setWorkspaceWarehouse(n)
		case "startImport":
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

// A state keeper keeps the state updated. It receives notifications of the
// state changes from PostgreSQL for this and other instances and updates the
// state of this instance.
type stateKeeper struct {
	*APIs
	workspaces  map[int]*Workspace
	connections map[int]*Connection
	resources   map[int]*Resource
}

// replaceConnection calls the function f passing a copy of the connection with
// identifier id. After f is returned, it replaces the connection with its
// copy in the state and returns the latter.
func (s *stateKeeper) replaceConnection(id int, f func(c *Connection)) *Connection {
	c := s.connections[id]
	cc := new(Connection)
	*cc = *c
	f(cc)
	connections := c.workspace.Connections
	connections.state.Lock()
	connections.state.ids[c.id] = cc
	connections.state.Unlock()
	s.connections[c.id] = cc
	return cc
}

// replaceDataType calls the function f passing a copy of the data type called
// name of the given workspace. After f is returned, it replaces the data type
// with its copy in the state and returns the latter.
func (s *stateKeeper) replaceDataType(workspace int, name string, f func(c *DataType)) *DataType {
	tt := new(DataType)
	dt := s.workspaces[workspace].DataTypes
	dt.state.Lock()
	t := dt.state.names[name]
	*tt = *t
	f(tt)
	dt.state.names[name] = tt
	dt.state.Unlock()
	return tt
}

// replaceEventType calls the function f passing a copy of the event type with
// identifier id of the given workspace. After f is returned, it replaces the
// type with its copy in the state and returns the latter.
func (s *stateKeeper) replaceEventType(workspace int, id int, f func(c *EventType)) *EventType {
	tt := new(EventType)
	dt := s.workspaces[workspace].EventTypes
	dt.state.Lock()
	t := dt.state.ids[id]
	*tt = *t
	f(tt)
	dt.state.ids[id] = tt
	dt.state.Unlock()
	return tt
}

// replaceResource calls the function f passing a copy of the resource with
// identifier id. After f is returned, it replaces the resource with its copy
// in the state and returns the latter.
func (s *stateKeeper) replaceResource(id int, f func(c *Resource)) *Resource {
	r := s.resources[id]
	rr := new(Resource)
	*rr = *r
	f(rr)
	state := r.workspace.resources
	state.Lock()
	state.ids[r.id] = rr
	state.Unlock()
	s.resources[r.id] = rr
	return rr
}

// replaceWorkspace calls the function f passing a copy of the workspace with
// identifier id. After f is returned, it replaces the workspace with its
// copy in the state and returns the latter.
func (s *stateKeeper) replaceWorkspace(id int, f func(c *Workspace)) *Workspace {
	w := s.workspaces[id]
	ww := new(Workspace)
	*ww = *w
	f(ww)
	workspaces := w.account.Workspaces
	workspaces.state.Lock()
	workspaces.state.ids[w.id] = ww
	workspaces.state.Unlock()
	s.workspaces[w.id] = ww
	return ww
}

// addConnectionNotification is the notification event sent when a new
// connection is added.
type addConnectionNotification struct {
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
	Key         []byte // server key to add
}

// addConnection adds a new connection.
func (s *stateKeeper) addConnection(n postgres.Notification) {
	e := addConnectionNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	workspace, _ := s.workspaces[e.Workspace]
	connector, _ := s.Connectors.state.Get(e.Connector)
	var r *Resource
	if connector.oAuth != nil {
		if _, ok := s.resources[e.Resource.ID]; ok {
			if e.Resource.AccessToken != "" {
				r = s.replaceResource(e.Resource.ID, func(r *Resource) {
					r.accessToken = e.Resource.AccessToken
					r.refreshToken = e.Resource.RefreshToken
					r.expiresIn = e.Resource.ExpiresIn
				})
				s.resources[r.id] = r
			}
		} else {
			r = &Resource{
				id:           e.Resource.ID,
				workspace:    workspace,
				connector:    connector,
				code:         e.Resource.Code,
				accessToken:  e.Resource.AccessToken,
				refreshToken: e.Resource.RefreshToken,
				expiresIn:    e.Resource.ExpiresIn,
			}
			state := workspace.resources
			state.Lock()
			state.ids[r.id] = r
			state.Unlock()
			s.resources[r.id] = r
		}
	}
	c := &Connection{
		account:     workspace.account,
		workspace:   workspace,
		id:          e.ID,
		name:        e.Name,
		role:        e.Role,
		connector:   connector,
		storage:     s.connections[e.Storage],
		stream:      s.connections[e.Stream],
		resource:    r,
		websiteHost: e.WebsiteHost,
	}
	if e.Key != nil {
		c.keys = []string{string(e.Key)}
	}
	state := workspace.Connections.state
	state.Lock()
	state.ids[c.id] = c
	state.Unlock()
	s.connections[e.ID] = c
	if connector.typ == AppType {
		// TODO(marco) only one server should reload the schema.
		go func() {
			err := workspace.Connections.reloadSchema(c.id)
			if err != nil {
				log.Printf("[error] cannot reload schema for connection %d: %s", c.id, err)
			}
		}()
	}
}

// addDataTypeNotification is the notification event sent when a data type is
// added.
type addDataTypeNotification struct {
	Workspace   int
	Name        string
	Description string
	Definition  json.RawMessage `json:",omitempty"`
}

// addDataType adds a new data type.
func (s *stateKeeper) addDataType(n postgres.Notification) {
	e := addDataTypeNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	typ, err := types.ParseType(string(e.Definition), nil)
	if err != nil {
		log.Printf("[error] cannot parse data type definition of notification %s from %d: %s", n.Name, n.PID, err)
		return
	}
	t := DataType{
		name:        e.Name,
		description: e.Description,
		definition:  string(e.Definition),
		typ:         typ,
	}
	eventTypes := s.workspaces[e.Workspace].DataTypes
	eventTypes.state.Lock()
	eventTypes.state.names[e.Name] = &t
	eventTypes.state.Unlock()
}

// addEventTypeNotification is the notification event sent when an event type
// is added.
type addEventTypeNotification struct {
	Workspace   int
	ID          int
	Name        string
	Description string
	Schema      json.RawMessage `json:",omitempty"`
}

// addEventType adds a new event type.
func (s *stateKeeper) addEventType(n postgres.Notification) {
	e := addEventTypeNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	var err error
	var schema types.Schema
	if len(e.Schema) > 0 {
		schema, err = types.ParseSchema(strings.NewReader(string(e.Schema)), nil)
		if err != nil {
			log.Printf("[error] cannot parse event type schema of notification %s from %d: %s", n.Name, n.PID, err)
			return
		}
	}
	t := EventType{
		id:           e.ID,
		name:         e.Name,
		description:  e.Description,
		schema:       schema,
		schemaSource: string(e.Schema),
	}
	eventTypes := s.workspaces[e.Workspace].EventTypes
	eventTypes.state.Lock()
	eventTypes.state.ids[e.ID] = &t
	eventTypes.state.Unlock()
}

// deleteConnectionNotification is the notification event sent when a
// connection is deleted.
type deleteConnectionNotification struct {
	ID int
}

// deleteConnection deletes a connection.
func (s *stateKeeper) deleteConnection(n postgres.Notification) {

	e := deleteConnectionNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}

	var usages []*Connection
	connections := s.connections[e.ID].workspace.Connections
	delete(s.connections, e.ID)

	connections.state.Lock()
	for _, c := range connections.state.ids {
		if c.storage != nil && c.storage.id == e.ID || c.stream != nil && c.stream.id == e.ID {
			usages = append(usages, c)
		}
	}
	for _, c := range usages {
		cc := Connection{}
		cc = *c
		if cc.storage != nil && cc.storage.id == e.ID {
			cc.storage = nil
		} else {
			cc.stream = nil
		}
		connections.state.ids[c.id] = &cc
	}
	delete(connections.state.ids, e.ID)
	connections.state.Unlock()
}

// deleteDataTypeNotification is the notification event sent when a data type
// is deleted.
type deleteDataTypeNotification struct {
	Workspace int
	Name      string
}

// deleteDataType deletes a data type.
func (s *stateKeeper) deleteDataType(n postgres.Notification) {
	e := deleteDataTypeNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	eventTypes := s.workspaces[e.Workspace].DataTypes
	eventTypes.state.Lock()
	delete(eventTypes.state.names, e.Name)
	eventTypes.state.Unlock()
}

// deleteEventTypeNotification is the notification event sent when an event
// type is deleted.
type deleteEventTypeNotification struct {
	Workspace int
	ID        int
}

// deleteEventType deletes an event type.
func (s *stateKeeper) deleteEventType(n postgres.Notification) {
	e := deleteEventTypeNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	eventTypes := s.workspaces[e.Workspace].EventTypes
	eventTypes.state.Lock()
	delete(eventTypes.state.ids, e.ID)
	eventTypes.state.Unlock()
	// TODO(marco): remove events from ClickHouse and then remove definitively the event type from Postgres.
}

// endImportNotification is the notification event sent when an import ends.
type endImportNotification struct {
	id int
}

// endImport ends an import.
func (s *stateKeeper) endImport(n postgres.Notification) {
	e := endImportNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	for _, c := range s.connections {
		if c.importInProgress != nil && c.importInProgress.id == e.id {
			s.replaceConnection(c.id, func(c *Connection) {
				c.importInProgress = nil
			})
			break
		}
	}
}

// generateConnectionKeyNotification is the notification event sent when a
// connection key is generated.
type generateConnectionKeyNotification struct {
	Connection   int
	Value        []byte
	CreationTime time.Time
}

// generateConnectionKey generates a new connection key.
func (s *stateKeeper) generateConnectionKey(n postgres.Notification) {
	e := generateConnectionKeyNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	key := string(e.Value)
	s.replaceConnection(e.Connection, func(c *Connection) {
		c.keys = append(c.keys, key)
	})
}

// revokeConnectionKeyNotification is the notification event sent when a
// connection key is revoked.
type revokeConnectionKeyNotification struct {
	Connection int
	Value      []byte
}

// revokeConnectionKey revokes a connection key.
func (s *stateKeeper) revokeConnectionKey(n postgres.Notification) {
	e := revokeConnectionKeyNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	c := s.connections[e.Connection]
	keys := make([]string, 0, len(c.keys)-1)
	key := string(e.Value)
	for _, k := range c.keys {
		if k != key {
			keys = append(keys, k)
		}
	}
	s.replaceConnection(e.Connection, func(c *Connection) {
		c.keys = keys
	})
}

// setConnectionSettingsNotification is the notification event sent when the
// settings of a connection is changed.
type setConnectionSettingsNotification struct {
	Connection int
	Settings   []byte
}

// setConnectionSettings sets the settings of a connection.
func (s *stateKeeper) setConnectionSettings(n postgres.Notification) {
	e := setConnectionSettingsNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	s.replaceConnection(e.Connection, func(c *Connection) {
		c.settings = e.Settings
	})
}

// setConnectionSettingsNotification is the notification event sent when the
// settings of a connection is changed.
type setConnectionStorageNotification struct {
	Connection int
	Storage    int
}

// setConnectionStorages sets the storage of a connection.
func (s *stateKeeper) setConnectionStorage(n postgres.Notification) {
	e := setConnectionStorageNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	s.replaceConnection(e.Connection, func(c *Connection) {
		c.storage = s.connections[e.Storage]
	})
}

// setConnectionStreamNotification is the notification event sent when the
// settings of a connection is changed.
type setConnectionStreamNotification struct {
	Connection int
	Stream     int
}

// setConnectionStream sets the stream of a connection.
func (s *stateKeeper) setConnectionStream(n postgres.Notification) {
	e := setConnectionStreamNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	s.replaceConnection(e.Connection, func(c *Connection) {
		c.stream = s.connections[e.Stream]
	})
}

// setConnectionMappingsNotification is the notification event sent when the
// mappings of a connection are saved.
type setConnectionMappingsNotification struct {
	Connection int
	Mappings   []notifiedMapping
}

// notifiedMapping is a mapping to set notified to the state keeper.
type notifiedMapping struct {
	ID         int
	In         types.Schema
	SourceCode string
	Out        types.Schema
}

// setConnectionMappings sets the mappings of a connection.
func (s stateKeeper) setConnectionMappings(n postgres.Notification) {
	e := setConnectionMappingsNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	s.replaceConnection(e.Connection, func(c *Connection) {
		c.mappings = make([]*Mapping, len(e.Mappings))
		for i, m := range e.Mappings {
			c.mappings[i] = &Mapping{
				id:         m.ID,
				connection: c,
				in:         m.In,
				sourceCode: m.SourceCode,
				out:        m.Out,
			}
		}
	})
}

// setUserQueryNotification is the notification event sent when a user query of
// a connection is changed.
type setUserQueryNotification struct {
	Connection int
	Query      string
}

// setConnectionUserQuery sets the user query of a connection.
func (s *stateKeeper) setConnectionUserQuery(n postgres.Notification) {
	e := setUserQueryNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	c := s.replaceConnection(e.Connection, func(c *Connection) {
		c.usersQuery = e.Query
	})
	// TODO(marco) only one server should reload the schema.
	go func() {
		err := c.workspace.Connections.reloadSchema(c.id)
		if err != nil {
			log.Printf("[error] cannot reload schema for connection %d: %s", c.id, err)
		}
	}()
}

// setConnectionUserSchemaNotification is the notification event sent when the
// user schema of a connection is changed.
type setConnectionUserSchemaNotification struct {
	Connection int
	Schema     types.Schema
}

// setConnectionUserSchema sets the user schema of a connection.
func (s *stateKeeper) setConnectionUserSchema(n postgres.Notification) {
	e := setConnectionUserSchemaNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	s.replaceConnection(e.Connection, func(c *Connection) {
		c.schema = e.Schema
	})
}

// setDataTypeDescriptionNotification is the notification event sent when the
// description of a data type is changed.
type setDataTypeDescriptionNotification struct {
	Workplace   int
	Name        string
	Description string
}

// setDataTypeDescription sets the description of a data type.
func (s *stateKeeper) setDataTypeDescription(n postgres.Notification) {
	e := setDataTypeDescriptionNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	s.replaceDataType(e.Workplace, e.Name, func(t *DataType) {
		t.description = e.Description
	})
}

// setDataTypeDefinitionNotification is the notification event sent when the
// definition of a data type is changed.
type setDataTypeDefinitionNotification struct {
	Workspace  int
	Name       string
	Definition json.RawMessage `json:",omitempty"`
}

// setDataTypeDefinition sets the definition of a data type.
func (s *stateKeeper) setDataTypeDefinition(n postgres.Notification) {
	e := setDataTypeDefinitionNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	typ, err := types.ParseType(string(e.Definition), nil)
	if err != nil {
		log.Printf("[error] cannot parse data type definition of notification %s from %d: %s", n.Name, n.PID, err)
		return
	}
	s.replaceDataType(e.Workspace, e.Name, func(t *DataType) {
		t.definition = string(e.Definition)
		t.typ = typ
	})
}

// setEventTypeDescriptionNotification is the notification event sent when the
// description of an event type is changed.
type setEventTypeDescriptionNotification struct {
	Workplace   int
	ID          int
	Description string
}

// setEventTypeDescription sets the description of an event type.
func (s *stateKeeper) setEventTypeDescription(n postgres.Notification) {
	e := setEventTypeDescriptionNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	s.replaceEventType(e.Workplace, e.ID, func(t *EventType) {
		t.description = e.Description
	})
}

// setEventTypeSchemaNotification is the notification event sent when the
// schema of an event type is changed.
type setEventTypeSchemaNotification struct {
	Workspace int
	ID        int
	Schema    json.RawMessage `json:",omitempty"`
}

// setEventTypeSchema sets the schema of an event type.
func (s *stateKeeper) setEventTypeSchema(n postgres.Notification) {
	e := setEventTypeSchemaNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	var err error
	var schema types.Schema
	if len(e.Schema) > 0 {
		schema, err = types.ParseSchema(strings.NewReader(string(e.Schema)), nil)
		if err != nil {
			log.Printf("[error] cannot parse event type schema of notification %s from %d: %s", n.Name, n.PID, err)
			return
		}
	}
	s.replaceEventType(e.Workspace, e.ID, func(t *EventType) {
		t.schema = schema
		t.schemaSource = string(e.Schema)
	})
}

type setResourceNotification struct {
	ID           int
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Time
}

// setResource sets a resource.
func (s *stateKeeper) setResource(n postgres.Notification) {
	e := setResourceNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	s.replaceResource(e.ID, func(r *Resource) {
		r.accessToken = e.AccessToken
		r.refreshToken = e.RefreshToken
		r.expiresIn = e.ExpiresIn
	})
}

// setWorkspaceWarehouse is the notification event sent when a workspace
// warehouse is changed.
type setWorkspaceWarehouseNotification struct {
	Workspace int
	Warehouse *notifiedWarehouse
}

type notifiedWarehouse struct {
	Type     warehouses.Type
	Settings json.RawMessage `json:",omitempty"`
}

// setWorkspaceUserSchema sets the warehouse of a workspace.
func (s *stateKeeper) setWorkspaceWarehouse(n postgres.Notification) {
	e := setWorkspaceWarehouseNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	disconnected := s.workspaces[e.Workspace].warehouse
	if e.Warehouse != nil {
		s.replaceWorkspace(e.Workspace, func(w *Workspace) {
			var settings warehouses.PostgreSQLSettings
			_ = json.Unmarshal(e.Warehouse.Settings, &settings)
			w.warehouse = warehouses.OpenPostgres(&settings)
		})
	} else {
		s.replaceWorkspace(e.Workspace, func(w *Workspace) {
			w.warehouse = nil
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

// startImportNotification is the notification event sent when an import
// starts.
type startImportNotification struct {
	ID         int
	Connection int
	Storage    int
	Reimport   bool
	StartTime  time.Time
}

// startImport starts an import.
func (s *stateKeeper) startImport(n postgres.Notification) {
	e := startImportNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	c := s.replaceConnection(e.Connection, func(c *Connection) {
		c.importInProgress = &ImportInProgress{
			id:         e.ID,
			connection: c,
			storage:    s.connections[e.Storage],
			reimport:   e.Reimport,
			startTime:  e.StartTime,
		}
	})
	// Start the import.
	// TODO(marco): only one server should starts the import.
	go c.workspace.Connections.startImport(c.importInProgress)
}
