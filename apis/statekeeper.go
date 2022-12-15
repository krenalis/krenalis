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

// keepState starts the state keeper. It is called in its own goroutine. The
// workspaces argument contains all workspaces and connections all connections.
func (apis *APIs) keepState(ctx context.Context, workspaces map[int]*Workspace, connections map[int]*Connection) {

	sk := stateKeeper{apis, workspaces, connections}

	notifications := apis.db.ListenToNotifications(ctx)
	for {
		n := <-notifications
		if logNotifications {
			log.Printf("[info] received notification from pid %d and name %q : %s",
				n.PID, n.Name, n.Payload)
		}
		switch n.Name {
		case "addConnection":
			sk.addConnection(n)
		case "deleteConnection":
			sk.deleteConnection(n)
		case "endImport":
			sk.endImport(n)
		case "setConnectionSettings":
			sk.setConnectionSettings(n)
		case "setConnectionStorage":
			sk.setConnectionStorage(n)
		case "setConnectionStream":
			sk.setConnectionStream(n)
		case "setConnectionTransformations":
			sk.setConnectionTransformations(n)
		case "setConnectionUserQuery":
			sk.setConnectionUserQuery(n)
		case "setConnectionUserSchema":
			sk.setConnectionUserSchema(n)
		case "setWorkspaceEventSchema":
			sk.setWorkspaceEventSchema(n)
		case "setWorkspaceGroupSchema":
			sk.setWorkspaceGroupSchema(n)
		case "setWorkspaceUserSchema":
			sk.setWorkspaceUserSchema(n)
		case "startImport":
			sk.startImport(n)
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
		account:   workspace.account,
		workspace: workspace,
		id:        e.ID,
		name:      e.Name,
		role:      e.Role,
		connector: connector,
		storage:   s.connections[e.Storage],
		stream:    s.connections[e.Stream],
		resource:  resource,
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

// setWorkspaceEventSchemaNotification is the notification event sent when a
// workspace event schema is changed.
type setWorkspaceEventSchemaNotification struct {
	Workspace int
	Schema    string
}

// setWorkspaceGroupSchema sets the user schema of a workspace.
func (s *stateKeeper) setWorkspaceEventSchema(n postgres.Notification) {
	e := setWorkspaceEventSchemaNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	schema, err := types.ParseSchema(strings.NewReader(e.Schema), nil)
	if err != nil {
		log.Printf("[error] cannot parse workspace event schema of notification %s from %d: %s", n.Name, n.PID, err)
		return
	}
	s.setWorkspace(e.Workspace, func(w *Workspace) {
		w.schema.event = schema
		w.schemaSources.event = e.Schema
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
