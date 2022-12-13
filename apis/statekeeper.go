package apis

import (
	"bytes"
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

// keepState starts the state keeper. It is called in its own goroutine.
func (apis *APIs) keepState(ctx context.Context) {

	sk := stateKeeper{apis}

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
		case "setConnectionUserQuery":
			sk.setConnectionUserQuery(n)
		case "setConnectionSettings":
			sk.setConnectionSettings(n)
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
}

// addConnectionNotification is the notification event sent when a new
// connection is added.
type addConnectionNotification struct {
	Account   int            // account identifier
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
func (s stateKeeper) addConnection(n postgres.Notification) {
	e := addConnectionNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	account, _ := s.Accounts.get(e.Account)
	workspace, _ := account.Workspaces.get(e.Workspace)
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
	var storage *Connection
	if e.Storage > 0 {
		storage, _ = workspace.Connections.get(e.Storage)
	}
	var stream *Connection
	if e.Stream > 0 {
		stream, _ = workspace.Connections.get(e.Stream)
	}
	workspace.Connections.add(&Connection{
		account:   account,
		workspace: workspace,
		id:        e.ID,
		name:      e.Name,
		role:      e.Role,
		connector: connector,
		storage:   storage,
		stream:    stream,
		resource:  resource,
	})
	if connector.typ == AppType {
		// TODO(marco) only one server should reload the schema.
		go func() {
			err := workspace.Connections.reloadSchema(e.ID)
			if err != nil {
				log.Printf("[error] cannot reload schema for connection %d: %s", e.ID, err)
			}
		}()
	}
	return
}

// deleteConnectionNotification is the notification event sent when a
// connection is deleted.
type deleteConnectionNotification struct {
	Account   int
	Workspace int
	ID        int
}

// deleteConnection deletes a connection.
func (s stateKeeper) deleteConnection(n postgres.Notification) {
	e := deleteConnectionNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	account, _ := s.Accounts.get(e.Account)
	workspace, _ := account.Workspaces.get(e.Workspace)
	workspace.Connections.delete(e.ID)
	return
}

// setUserQueryNotification is the notification event sent when a user query of
// a connection is changed.
type setUserQueryNotification struct {
	Account    int
	Workspace  int
	Connection int
	Query      string
}

// setConnectionUserQuery sets the user query of a connection.
func (s stateKeeper) setConnectionUserQuery(n postgres.Notification) {
	e := setUserQueryNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	account, _ := s.Accounts.get(e.Account)
	workspace, _ := account.Workspaces.get(e.Workspace)
	connection := workspace.Connections.clone(e.Connection)
	connection.usersQuery = e.Query
	workspace.Connections.add(connection)
	// TODO(marco) only one server should reload the schema.
	go func() {
		err := workspace.Connections.reloadSchema(e.Connection)
		if err != nil {
			log.Printf("[error] cannot reload schema for connection %d: %s", e.Connection, err)
		}
	}()
	return
}

// setConnectionUserSchemaNotification is the notification event sent when the
// user schema of a connection is changed.
type setConnectionUserSchemaNotification struct {
	Account    int
	Workspace  int
	Connection int
	Schema     json.RawMessage
}

// setConnectionUserSchema sets the user schema of a connection.
func (s stateKeeper) setConnectionUserSchema(n postgres.Notification) {
	e := setConnectionUserSchemaNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	schema, err := types.ParseSchema(bytes.NewReader(e.Schema), nil)
	if err != nil {
		log.Printf("[error] cannot parse user schema for connection %d received from notification: %s", e.Connection, err)
		return
	}
	account, _ := s.Accounts.get(e.Account)
	workspace, _ := account.Workspaces.get(e.Workspace)
	connection := workspace.Connections.clone(e.Connection)
	connection.schema = schema
	workspace.Connections.add(connection)
	return
}

// setConnectionSettingsNotification is the notification event sent when the
// settings of a connection is changed.
type setConnectionSettingsNotification struct {
	Account    int
	Workspace  int
	Connection int
	Settings   []byte
}

// setConnectionSettings sets the settings of a connection.
func (s stateKeeper) setConnectionSettings(n postgres.Notification) {
	e := setConnectionSettingsNotification{}
	if !decodeStateNotification(n, &e) {
		return
	}
	account, _ := s.Accounts.get(e.Account)
	workspace, _ := account.Workspaces.get(e.Workspace)
	connection := workspace.Connections.clone(e.Connection)
	connection.settings = e.Settings
	workspace.Connections.add(connection)
	return
}
