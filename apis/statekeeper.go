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
		case "addEventDataType":
			s.addEventDataType(n)
		case "addEventType":
			s.addEventType(n)
		case "deleteConnection":
			s.deleteConnection(n)
		case "deleteEventDataType":
			s.deleteEventDataType(n)
		case "deleteEventType":
			s.deleteEventType(n)
		case "endImport":
			s.endImport(n)
		case "setConnectionSettings":
			s.setConnectionSettings(n)
		case "setConnectionStorage":
			s.setConnectionStorage(n)
		case "setConnectionStream":
			s.setConnectionStream(n)
		case "setConnectionTransformations":
			s.setConnectionTransformations(n)
		case "setConnectionUserQuery":
			s.setConnectionUserQuery(n)
		case "setConnectionUserSchema":
			s.setConnectionUserSchema(n)
		case "setEventDataTypeDescription":
			s.setEventDataTypeDescription(n)
		case "setEventDataTypeSchema":
			s.setEventDataTypeSchema(n)
		case "setEventTypeDescription":
			s.setEventTypeDescription(n)
		case "setEventTypeSchema":
			s.setEventTypeSchema(n)
		case "setWorkspaceGroupSchema":
			s.setWorkspaceGroupSchema(n)
		case "setWorkspaceUserSchema":
			s.setWorkspaceUserSchema(n)
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
}

// setConnection calls the function f passing a copy of the connection with
// identifier id. After f is returned, it replaces the connection with its
// copy in the state and returns the latter.
func (s *stateKeeper) setConnection(id int, f func(c *Connection)) *Connection {
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

// setEventDataType calls the function f passing a copy of the event data type
// called name of the given workspace. After f is returned, it replaces the
// data type with its copy in the state and returns the latter.
func (s *stateKeeper) setEventDataType(workspace int, name string, f func(c *EventDataType)) *EventDataType {
	tt := new(EventDataType)
	dt := s.workspaces[workspace].EventDataTypes
	dt.state.Lock()
	t := dt.state.names[name]
	*tt = *t
	f(tt)
	dt.state.names[name] = tt
	dt.state.Unlock()
	return tt
}

// setEventType calls the function f passing a copy of the event type with
// identifier id of the given workspace. After f is returned, it replaces the
// type with its copy in the state and returns the latter.
func (s *stateKeeper) setEventType(workspace int, id int, f func(c *EventType)) *EventType {
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

// setWorkspace calls the function f passing a copy of the workspace with
// identifier id. After f is returned, it replaces the workspace with its
// copy in the state and returns the latter.
func (s *stateKeeper) setWorkspace(id int, f func(c *Workspace)) *Workspace {
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
	ServerKey string         // server key to add (currently not used)
	Resource  struct {       // resource.
		ID                int       // identifier, can be zero
		Code              string    // code, can be empty.
		OAuthAccessToken  string    // access token, can be empty.
		OAuthRefreshToken string    // refresh token, can be empty.
		OAuthExpiresIn    time.Time // expiration time, can be the zero time.
	}
	WebsiteHost string
}

// addConnection adds a new connection.
func (s *stateKeeper) addConnection(n postgres.Notification) {
	e := addConnectionNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	workspace, _ := s.workspaces[e.Workspace]
	connector, _ := s.Connectors.get(e.Connector)
	var resource *Resource
	if connector.oAuth != nil {
		r, ok := connector.getResource(e.Resource.ID)
		if ok {
			if e.Resource.OAuthAccessToken != "" {
				// Update the current resource.
				resource = &Resource{}
				*resource = *r
				resource.oAuthAccessToken = e.Resource.OAuthAccessToken
				resource.oAuthRefreshToken = e.Resource.OAuthRefreshToken
				resource.oAuthRefreshToken = e.Resource.OAuthRefreshToken
				connector.addResource(resource)
			}
		} else {
			// Add a new resource.
			resource = &Resource{
				id:                e.Resource.ID,
				code:              e.Resource.Code,
				oAuthAccessToken:  e.Resource.OAuthAccessToken,
				oAuthRefreshToken: e.Resource.OAuthRefreshToken,
				oAuthExpiresIn:    e.Resource.OAuthExpiresIn,
			}
			connector.addResource(resource)
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
		resource:    resource,
		websiteHost: e.WebsiteHost,
	}
	workspace.Connections.state.Lock()
	workspace.Connections.state.ids[c.id] = c
	workspace.Connections.state.Unlock()
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

// addEventDataTypeNotification is the notification event sent when an event
// data type is added.
type addEventDataTypeNotification struct {
	Workspace   int
	Name        string
	Description string
	Schema      json.RawMessage `json:",omitempty"`
}

// addEventDataType adds a new event type.
func (s *stateKeeper) addEventDataType(n postgres.Notification) {
	e := addEventDataTypeNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	schema, err := types.ParseSchema(strings.NewReader(string(e.Schema)), nil)
	if err != nil {
		log.Printf("[error] cannot parse event data type schema of notification %s from %d: %s", n.Name, n.PID, err)
		return
	}
	t := EventDataType{
		name:         e.Name,
		description:  e.Description,
		schema:       schema,
		schemaSource: string(e.Schema),
	}
	eventTypes := s.workspaces[e.Workspace].EventDataTypes
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

	ws := s.connections[e.ID].workspace
	ws.Transformations.set(e.ID, nil)

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

// deleteEventDataTypeNotification is the notification event sent when an event
// data type is deleted.
type deleteEventDataTypeNotification struct {
	Workspace int
	Name      string
}

// deleteEventDataType deletes an event data type.
func (s *stateKeeper) deleteEventDataType(n postgres.Notification) {
	e := deleteEventDataTypeNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	eventTypes := s.workspaces[e.Workspace].EventDataTypes
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
			s.setConnection(c.id, func(c *Connection) {
				c.importInProgress = nil
			})
			break
		}
	}
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
	s.setConnection(e.Connection, func(c *Connection) {
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
	s.setConnection(e.Connection, func(c *Connection) {
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
	s.setConnection(e.Connection, func(c *Connection) {
		c.stream = s.connections[e.Stream]
	})
}

// setConnectionTransformations is the notification event sent when the
// transformations of a connection are saved.
type setConnectionTransformations struct {
	Connection      int
	Transformations []*Transformation
}

// setConnectionTransformations sets the transformations of a connection.
func (s stateKeeper) setConnectionTransformations(n postgres.Notification) {
	e := setConnectionTransformations{}
	if !decodeStateNotification(n, &e) {
		return
	}
	ws := s.connections[e.Connection].workspace
	ws.Transformations.set(e.Connection, e.Transformations)
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
	c := s.setConnection(e.Connection, func(c *Connection) {
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
	s.setConnection(e.Connection, func(c *Connection) {
		c.schema = e.Schema
	})
}

// setEventDataTypeDescriptionNotification is the notification event sent when
// the description of an event data type is changed.
type setEventDataTypeDescriptionNotification struct {
	Workplace   int
	Name        string
	Description string
}

// setEventDataTypeDescription sets the description of an event data type.
func (s *stateKeeper) setEventDataTypeDescription(n postgres.Notification) {
	e := setEventDataTypeDescriptionNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	s.setEventDataType(e.Workplace, e.Name, func(t *EventDataType) {
		t.description = e.Description
	})
}

// setEventDataTypeSchemaNotification is the notification event sent when the
// schema of an event data type is changed.
type setEventDataTypeSchemaNotification struct {
	Workspace int
	Name      string
	Schema    json.RawMessage `json:",omitempty"`
}

// setEventDataTypeSchema sets the schema of an event dta type.
func (s *stateKeeper) setEventDataTypeSchema(n postgres.Notification) {
	e := setEventDataTypeSchemaNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	schema, err := types.ParseSchema(strings.NewReader(string(e.Schema)), nil)
	if err != nil {
		log.Printf("[error] cannot parse event data type schema of notification %s from %d: %s", n.Name, n.PID, err)
		return
	}
	s.setEventDataType(e.Workspace, e.Name, func(t *EventDataType) {
		t.schema = schema
		t.schemaSource = string(e.Schema)
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
	s.setEventType(e.Workplace, e.ID, func(t *EventType) {
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
	s.setEventType(e.Workspace, e.ID, func(t *EventType) {
		t.schema = schema
		t.schemaSource = string(e.Schema)
	})
}

// setWorkspaceGroupSchemaNotification is the notification event sent when a
// workspace group schema is changed.
type setWorkspaceGroupSchemaNotification struct {
	Workspace int
	Schema    string
}

// setWorkspaceGroupSchema sets the user schema of a workspace.
func (s *stateKeeper) setWorkspaceGroupSchema(n postgres.Notification) {
	e := setWorkspaceGroupSchemaNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	schema, err := types.ParseSchema(strings.NewReader(e.Schema), nil)
	if err != nil {
		log.Printf("[error] cannot parse workspace group schema of notification %s from %d: %s", n.Name, n.PID, err)
		return
	}
	s.setWorkspace(e.Workspace, func(w *Workspace) {
		w.schema.group = schema
		w.schemaSources.group = e.Schema
	})
}

// setWorkspaceUserSchemaNotification is the notification event sent when a
// workspace user schema is changed.
type setWorkspaceUserSchemaNotification struct {
	Workspace int
	Schema    string
}

// setWorkspaceUserSchema sets the user schema of a workspace.
func (s *stateKeeper) setWorkspaceUserSchema(n postgres.Notification) {
	e := setWorkspaceUserSchemaNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	schema, err := types.ParseSchema(strings.NewReader(e.Schema), nil)
	if err != nil {
		log.Printf("[error] cannot parse workspace user schema of notification %s from %d: %s", n.Name, n.PID, err)
		return
	}
	s.setWorkspace(e.Workspace, func(w *Workspace) {
		w.schema.user = schema
		w.schemaSources.user = e.Schema
	})
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
	c := s.setConnection(e.Connection, func(c *Connection) {
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
