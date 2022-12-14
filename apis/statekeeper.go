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
		case "setConnectionSettings":
			sk.setConnectionSettings(n)
		case "setConnectionStorage":
			sk.setConnectionStorage(n)
		case "setConnectionStream":
			sk.setConnectionStream(n)
		case "setConnectionUserQuery":
			sk.setConnectionUserQuery(n)
		case "setConnectionUserSchema":
			sk.setConnectionUserSchema(n)
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
	c.workspace.Connections.add(cc)
	s.connections[c.id] = cc
	return cc
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
	workspace.Connections.add(c)
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
	ws.Connections.delete(e.ID)
	delete(s.connections, e.ID)
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
