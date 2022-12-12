//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"chichi/apis/postgres"
	"chichi/apis/types"
	_connector "chichi/connector"
	"chichi/connector/ui"

	"github.com/jxskiss/base62"
)

type Connections struct {
	*Workspace
	mu          sync.Mutex
	connections map[int]*Connection
}

// newConnections returns a new *Connections value.
func newConnections(ws *Workspace) *Connections {
	return &Connections{Workspace: ws, connections: map[int]*Connection{}}
}

// Connection represents a connection.
type Connection struct {
	id              int
	name            string
	role            ConnectionRole
	enabled         bool
	connector       *Connector
	storage         *Connection
	stream          *Connection
	resource        *Resource
	websiteHost     string
	userCursor      string
	identityColumn  string
	timestampColumn string
	settings        []byte
	schema          types.Schema
	usersQuery      string
}

// A ConnectionInfo describes a connection as returned by Get and List.
type ConnectionInfo struct {
	ID         int
	Name       string
	Type       ConnectorType
	Role       ConnectionRole
	Storage    int    // zero if the connection is not a file or does not have a storage.
	OAuthURL   string // empty if the connection does not use OAuth.
	LogoURL    string
	Enabled    bool
	UsersQuery string // only for databases.
}

const (
	maxInt32         = math.MaxInt32
	rawSchemaMaxSize = 16_777_215 // maximum size in runes of the 'schema' column of the 'connections' table.
	queryMaxSize     = 16_777_215 // maximum size in runes of a connection query.
)

var (
	ErrConnectionDisabled       = errors.New("connection is disabled")
	ErrFileHasNoStorage         = errors.New("file connection has not a storage")
	ErrStorageHasConnectedFiles = errors.New("storage has connected files")
	ErrUIEventNotExist          = errors.New("UI event does not exist")
)

// A ConnectionNotFoundError error indicates that a connection does not exist.
type ConnectionNotFoundError struct {
	Type ConnectorType
}

func (err ConnectionNotFoundError) Error() string {
	if err.Type == 0 {
		return "connection does not exist"
	}
	return fmt.Sprintf("%s connection does not exist", strings.ToLower(err.Type.String()))
}

// A DatabaseQueryError error is returned from a database connector if an error
// occurs when executing a query.
type DatabaseQueryError struct {
	Message string
}

func (err *DatabaseQueryError) Error() string {
	return err.Message
}

// ConnectionRole represents a connection role.
type ConnectionRole int

const (
	SourceRole      ConnectionRole = iota + 1 // source
	DestinationRole                           // destination
)

// MarshalJSON implements the json.Marshaler interface.
// It panics if role is not a valid ConnectionRole value.
func (role ConnectionRole) MarshalJSON() ([]byte, error) {
	return []byte(`"` + role.String() + `"`), nil
}

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

var null = []byte("null")

// UnmarshalJSON implements the json.Unmarshaler interface.
func (role *ConnectionRole) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, null) {
		return nil
	}
	var s any
	err := json.Unmarshal(data, &s)
	if err != nil {
		return fmt.Errorf("json: cannot unmarshal into a apis.ConnectionRole value: %s", err)
	}
	return role.Scan(s)
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

// AddApp adds an app connection given its role, app connector, name, OAuth
// refresh and access tokens and returns its identifier. name cannot be empty
// and cannot be longer than 120 runes.
//
// If the connector does not exist, it returns a ConnectorNotFoundError error.
func (this *Connections) AddApp(role ConnectionRole, connector int, name string, refreshToken, accessToken string, expiresIn time.Time) (int, error) {

	if role != SourceRole && role != DestinationRole {
		return 0, errors.New("invalid role")
	}
	if connector < 1 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
	}
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return 0, errors.New("invalid name")
	}

	id, err := generateConnectionID()
	if err != nil {
		return 0, err
	}
	c := Connection{
		id:   id,
		name: name,
		role: role,
	}
	c.connector, err = this.account.apis.Connectors.get(connector)
	if err != nil {
		return 0, ConnectorNotFoundError{AppType}
	}
	if c.connector.typ != AppType {
		return 0, errors.New("connector is not an app connector")
	}

	var clientSecret string
	if c.connector.oAuth != nil {
		clientSecret = c.connector.oAuth.ClientSecret
	}
	connection, err := _connector.RegisteredApp(c.connector.name).Connect(context.Background(), &_connector.AppConfig{
		Role:         _connector.Role(role),
		ClientSecret: clientSecret,
		AccessToken:  accessToken,
	})
	if err != nil {
		return 0, err
	}
	resourceCode, err := connection.Resource()
	if err != nil {
		return 0, err
	}
	resource, _ := c.connector.resources.getByCode(resourceCode)

	var resourceID int
	err = this.db.Transaction(func(tx *postgres.Tx) error {
		if resource == nil {
			err = tx.QueryRow("INSERT INTO resources (connector, code, oauth_access_token,"+
				" oauth_refresh_token, oauth_expires_in) VALUES ($1, $2, $3, $4, $5) RETURNING id",
				connector, resourceCode, accessToken, refreshToken, expiresIn).Scan(&resourceID)
		} else if refreshToken != resource.oAuthRefreshToken {
			_, err = tx.Exec("UPDATE resources "+
				"SET oauth_access_token = $1, oauth_refresh_token = $2, oauth_expires_in = $3 WHERE id = $4",
				accessToken, refreshToken, expiresIn, resource.id)
			resourceID = resource.id
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec("INSERT INTO connections (id, workspace, name, type, role, connector, resource)"+
			" VALUES ($1, $2, $3, 'App', $4, $5, $6)", id, this.id, name, role, connector, resourceID)
		return err
	})
	if err != nil {
		return 0, err
	}

	if resourceID > 0 {
		c.resource = c.connector.resources.add(resourceID, resourceCode, accessToken, refreshToken, expiresIn)
	}
	this.add(&c)

	go func() {
		err := this.reloadSchema(id)
		if err != nil {
			log.Printf("[error] cannot reload schema for connection %d: %s", id, err)
		}
	}()

	return id, err
}

// AddDatabase adds a database connection given its role, database connector,
// name and returns its identifier. name cannot be empty and cannot be longer
// than 120 runes.
//
// If the connector does not exist, it returns a ConnectorNotFoundError error.
func (this *Connections) AddDatabase(role ConnectionRole, connector int, name string) (int, error) {

	if role != SourceRole && role != DestinationRole {
		return 0, errors.New("invalid role")
	}
	if connector < 1 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
	}
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return 0, errors.New("invalid name")
	}

	id, err := generateConnectionID()
	if err != nil {
		return 0, err
	}
	c := Connection{
		id:   id,
		name: name,
		role: role,
	}
	c.connector, err = this.account.apis.Connectors.get(connector)
	if err != nil {
		return 0, ConnectorNotFoundError{DatabaseType}
	}
	if c.connector.typ != DatabaseType {
		return 0, errors.New("connector is not a database connector")
	}

	err = this.db.Transaction(func(tx *postgres.Tx) error {
		_, err = tx.Exec("INSERT INTO connections (id, workspace, name, type, role, connector)"+
			" VALUES ($1, $2, $3, 'Database', $4, $5)", id, this.id, name, role, connector)
		return err
	})
	if err != nil {
		return 0, err
	}

	this.add(&c)

	return id, nil
}

// AddFile adds a file connection given its role, connector, storage
// connection, name and returns its identifier. If storage is 0, the file
// connection does not have a storage, otherwise storage must have the given
// role. name cannot be empty and cannot be longer than 120 runes.
//
// If the connector does not exist, it returns a ConnectorNotFoundError error.
// If the storage does not exist, it returns a ConnectionNotFound error.
func (this *Connections) AddFile(role ConnectionRole, connector, storage int, name string) (int, error) {

	if role != SourceRole && role != DestinationRole {
		return 0, errors.New("invalid role")
	}
	if connector < 1 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
	}
	if storage < 0 || storage > maxInt32 {
		return 0, errors.New("invalid storage")
	}
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return 0, errors.New("invalid name")
	}

	id, err := generateConnectionID()
	if err != nil {
		return 0, err
	}
	c := Connection{
		id:   id,
		name: name,
		role: role,
	}
	c.connector, err = this.account.apis.Connectors.get(connector)
	if err != nil {
		return 0, ConnectorNotFoundError{FileType}
	}
	if c.connector.typ != FileType {
		return 0, errors.New("connector is not a file connector")
	}
	if storage > 0 {
		s, err := this.get(storage)
		if err != nil {
			return 0, ConnectionNotFoundError{StorageType}
		}
		if s.connector.typ != StorageType {
			return 0, errors.New("storage is not a storage connection")
		}
		if s.role != role {
			if role == SourceRole {
				return 0, errors.New("storage is not a source")
			}
			return 0, errors.New("storage is not a destination")
		}
	}

	err = this.db.Transaction(func(tx *postgres.Tx) error {
		_, err = tx.Exec("INSERT INTO connections (id, workspace, name, type, role, connector, storage)"+
			" VALUES ($1, $2, $3, 'File', $4, $5, $6)", id, this.id, name, role, connector, storage)
		return err
	})
	if err != nil {
		return 0, err
	}

	this.add(&c)

	return id, nil
}

// AddEventStream adds an event stream connection given its role, event stream
// connector and name. name cannot be empty and cannot be longer than 120
// runes.
//
// If the connector does not exist, it returns a ConnectorNotFoundError error.
func (this *Connections) AddEventStream(role ConnectionRole, connector int, name string) (int, error) {

	if role != SourceRole && role != DestinationRole {
		return 0, errors.New("invalid role")
	}
	if connector < 1 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
	}
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return 0, errors.New("invalid name")
	}

	id, err := generateConnectionID()
	if err != nil {
		return 0, err
	}
	c := Connection{
		id:   id,
		name: name,
		role: role,
	}
	c.connector, err = this.account.apis.Connectors.get(connector)
	if err != nil {
		return 0, ConnectorNotFoundError{EventStreamType}
	}
	if c.connector.typ != EventStreamType {
		return 0, errors.New("connector is not an event stream connector")
	}

	err = this.db.Transaction(func(tx *postgres.Tx) error {
		_, err = tx.Exec("INSERT INTO connections (id, workspace, name, type, role, connector)"+
			" VALUES ($1, $2, $3, 'EventStream', $4, $5)", id, this.id, name, role, connector)
		return err
	})
	if err != nil {
		return 0, err
	}

	this.add(&c)

	return id, nil
}

// AddServer adds a server connection given its role, server connector and
// name. name cannot be empty and cannot be longer than 120 runes.
//
// If the connector does not exist, it returns a ConnectorNotFoundError error.
func (this *Connections) AddServer(role ConnectionRole, connector int, name string) (int, error) {

	if role != SourceRole && role != DestinationRole {
		return 0, errors.New("invalid role")
	}
	if connector < 1 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
	}
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return 0, errors.New("invalid name")
	}

	id, err := generateConnectionID()
	if err != nil {
		return 0, err
	}
	c := Connection{
		id:   id,
		name: name,
		role: role,
	}
	c.connector, err = this.account.apis.Connectors.get(connector)
	if err != nil {
		return 0, ConnectorNotFoundError{ServerType}
	}
	if c.connector.typ != ServerType {
		return 0, errors.New("connector is not a server connector")
	}

	// Generate the API key.
	key, err := generateAPIKey()
	if err != nil {
		return 0, err
	}

	err = this.db.Transaction(func(tx *postgres.Tx) error {
		_, err = tx.Exec("INSERT INTO connections (id, workspace, name, type, role, connector)"+
			" VALUES ($1, $2, $3, 'Server', $4, $5)", id, this.id, name, role, connector)
		if err != nil {
			return err
		}
		_, err = tx.Exec("INSERT INTO connections_keys (connection, position, \"key\") VALUE ($1, 0, $2)", id, key)
		return err
	})
	if err != nil {
		return 0, err
	}

	this.add(&c)

	return id, nil
}

// AddMobile adds a mobile connection given its role, mobile connector and
// name. name cannot be empty and cannot be longer than 120 runes.
//
// If the connector does not exist, it returns a ConnectorNotFoundError error.
func (this *Connections) AddMobile(role ConnectionRole, connector int, name string) (int, error) {

	if role != SourceRole && role != DestinationRole {
		return 0, errors.New("invalid role")
	}
	if connector < 1 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
	}
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return 0, errors.New("invalid name")
	}

	id, err := generateConnectionID()
	if err != nil {
		return 0, err
	}
	c := Connection{
		id:   id,
		name: name,
		role: role,
	}
	c.connector, err = this.account.apis.Connectors.get(connector)
	if err != nil {
		return 0, ConnectorNotFoundError{MobileType}
	}
	if c.connector.typ != MobileType {
		return 0, errors.New("connector is not a mobile connector")
	}

	err = this.db.Transaction(func(tx *postgres.Tx) error {
		_, err = tx.Exec("INSERT INTO connections (id, workspace, name, type, role, connector)"+
			" VALUES ($1, $2, $3, 'Mobile', $4, $5)", id, this.id, name, role, connector)
		return err
	})
	if err != nil {
		return 0, err
	}

	this.add(&c)

	return id, nil
}

// AddStorage adds a storage connection given its role, connector, name and
// returns its identifier. name cannot be empty and cannot be longer than 120
// runes.
//
// If the connector does not exist, it returns a ConnectorNotFoundError error.
func (this *Connections) AddStorage(role ConnectionRole, connector int, name string) (int, error) {

	if role != SourceRole && role != DestinationRole {
		return 0, errors.New("invalid role")
	}
	if connector < 1 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
	}
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return 0, errors.New("invalid name")
	}

	id, err := generateConnectionID()
	if err != nil {
		return 0, err
	}
	c := Connection{
		id:   id,
		name: name,
		role: role,
	}
	c.connector, err = this.account.apis.Connectors.get(connector)
	if err != nil {
		return 0, ConnectorNotFoundError{StorageType}
	}
	if c.connector.typ != StorageType {
		return 0, errors.New("connector is not a storage connector")
	}

	err = this.db.Transaction(func(tx *postgres.Tx) error {
		_, err = tx.Exec("INSERT INTO connections (id, workspace, name, type, role, connector)"+
			" VALUES ($1, $2, $3, 'Storage', $4, $5)", id, this.id, name, role, connector)
		return err
	})
	if err != nil {
		return 0, err
	}

	this.add(&c)

	return id, nil
}

// AddWebsite adds a website connection given its role, website connector,
// name, website host and returns its identifier. name cannot be empty and
// cannot be longer than 120 runes. host may be of the form "host:port".
//
// If the connector does not exist, it returns a ConnectorNotFoundError error.
func (this *Connections) AddWebsite(role ConnectionRole, connector int, name, host string) (int, error) {

	if role != SourceRole && role != DestinationRole {
		return 0, errors.New("invalid role")
	}
	if connector < 1 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
	}
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return 0, errors.New("invalid name")
	}
	if h, p, found := strings.Cut(host, ":"); h == "" || len(host) > 255 {
		return 0, errors.New("invalid website host")
	} else if found {
		if port, _ := strconv.Atoi(p); port <= 0 || port > 65535 {
			return 0, errors.New("invalid website host")
		}
	}

	id, err := generateConnectionID()
	if err != nil {
		return 0, err
	}
	c := Connection{
		id:          id,
		name:        name,
		role:        role,
		websiteHost: host,
	}
	c.connector, err = this.account.apis.Connectors.get(connector)
	if err != nil {
		return 0, ConnectorNotFoundError{StorageType}
	}
	if c.connector.typ != StorageType {
		return 0, errors.New("connector is not a storage connector")
	}

	err = this.db.Transaction(func(tx *postgres.Tx) error {
		_, err = tx.Exec("INSERT INTO connections (id, workspace, name, type, role, connector, website_host)"+
			" VALUES ($1, $2, $3, 'Website', $4, $5, $6)", id, this.id, name, role, connector, host)
		return err
	})
	if err != nil {
		return 0, err
	}

	this.add(&c)

	return id, nil
}

// Get returns a ConnectionInfo describing the connection with identifier id.
// Returns a ConnectionNotFoundError error if the connection does not exist.
func (this *Connections) Get(id int) (*ConnectionInfo, error) {
	if id < 1 || id > maxInt32 {
		return nil, errors.New("invalid connection identifier")
	}
	c, err := this.get(id)
	if err != nil {
		return nil, ConnectionNotFoundError{}
	}
	info := ConnectionInfo{
		ID:         c.id,
		Name:       c.name,
		Type:       c.connector.typ,
		Role:       c.role,
		LogoURL:    c.connector.logoURL,
		Enabled:    c.enabled,
		UsersQuery: c.usersQuery,
	}
	if c.storage != nil {
		info.Storage = c.storage.id
	}
	if c.connector.oAuth != nil {
		info.OAuthURL = c.connector.oAuth.URL
	}
	return &info, nil
}

// Delete deletes the connection with the given identifier.
// If the connection does not exist, it does nothing.
//
// If the connection is a storage and has connected files, it returns the
// ErrStorageHasConnectedFiles error.
func (this *Connections) Delete(id int) error {

	if id < 1 || id > maxInt32 {
		return errors.New("invalid connection identifier")
	}

	c, err := this.get(id)
	if err != nil {
		return nil
	}

	var connectionColumn string
	switch c.connector.typ {
	case MobileType, WebsiteType:
		connectionColumn = "source"
	case ServerType:
		connectionColumn = "server"
	case EventStreamType:
		connectionColumn = "stream"
	}

	var deletedResources []int
	err = this.db.Transaction(func(tx *postgres.Tx) error {
		var resource string
		err := tx.QueryRow("SELECT resource FROM connections WHERE id = $1", id).Scan(&resource)
		if err != nil {
			if err == postgres.ErrNoRows {
				return nil
			}
			return err
		}
		if c.connector.typ == StorageType {
			var hasFiles bool
			err = tx.QueryRow("SELECT TRUE FROM connections WHERE storage = $1", id).Scan(&hasFiles)
			if err != nil {
				if err == postgres.ErrNoRows {
					return ErrStorageHasConnectedFiles
				}
				return err
			}
		}
		_, err = tx.Exec("DELETE FROM connections WHERE id = $1", id)
		if err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM connections_keys WHERE connection = $1", id)
		if err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM connections_imports WHERE connection = $1", id)
		if err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM connections_stats WHERE connection = $1", id)
		if err != nil {
			return err
		}
		if connectionColumn != "" {
			_, err = tx.Exec("DELETE FROM connections_stats_events WHERE "+connectionColumn+" = $1", id)
			if err != nil {
				return err
			}
		}
		_, err = tx.Exec("DELETE FROM connections_users WHERE connection = $1", id)
		if err != nil {
			return err
		}
		// Delete the resource of the deleted connection if it has no other connections.
		err = tx.QueryScan("DELETE FROM resources AS r WHERE NOT EXISTS (\n"+
			"\tSELECT FROM connections AS s\n"+
			"\tWHERE r.id = $1 AND s.resource IS NULL\n)\nRETURNING r.id", resource,
			func(rows *postgres.Rows) error {
				var id int
				for rows.Next() {
					if err := rows.Scan(&id); err != nil {
						return err
					}
					deletedResources = append(deletedResources, id)
				}
				return nil
			})
		return err
	})
	if err != nil {
		return err
	}

	if deletedResources != nil {
		for _, id := range deletedResources {
			c.connector.resources.delete(id)
		}
	}
	this.delete(id)

	return nil
}

// Import starts the import of the users from the connection with the given
// identifier. If the connection is an app and reimport is false, it imports
// the users from the current cursor, otherwise imports all users. The
// connection must be a source app, database or file connection.
//
// Returns a ConnectionNotFoundError error if the connection does not exist.
// Returns the ErrConnectionDisabled error if the connection does not have any
// transformation function associated to it.
// Returns the ErrFileHasNoStorage error if the connection is a file and does
// not have a storage.
func (this *Connections) Import(id int, reimport bool) (err error) {

	if id < 1 || id > maxInt32 {
		return errors.New("invalid connection identifier")
	}

	// Check that the connection exists, has an allowed type and is a source.
	c, err := this.get(id)
	if err != nil {
		return ConnectionNotFoundError{}
	}
	var storage int
	switch c.connector.typ {
	case AppType, DatabaseType, EventStreamType:
	case FileType:
		if c.storage == nil {
			return ErrFileHasNoStorage
		}
		storage = c.storage.id
	default:
		return fmt.Errorf("cannot import from a %s connection", strings.ToLower(c.connector.typ.String()))
	}
	if c.role == DestinationRole {
		return errors.New("cannot import from a destination")
	}

	// Check that the connection has at least one transformation associated to it.
	if c.connector.typ != EventStreamType {
		transformations, err := this.Transformations.List(id)
		if err != nil {
			return fmt.Errorf("cannot list transformations for %d: %s", id, err)
		}
		if len(transformations) == 0 {
			return ErrConnectionDisabled
		}
	}

	// Track the import in the database.
	var importID int
	err = this.db.QueryRow("INSERT INTO connections_imports (connection, storage, start_time)\n"+
		"VALUES ($1, $2, $3)\nRETURNING id", id, storage, time.Now().UTC()).Scan(&importID)
	if err != nil {
		return err
	}

	// Start the import.
	go func() {
		err = this.startImport(c, reimport)
		var errorMsg string
		if err != nil {
			if e, ok := err.(importError); ok {
				errorMsg = abbreviate(e.Error(), 1000)
			} else {
				log.Printf("[error] cannot do import %d: %s", importID, err)
				errorMsg = "an internal error has occurred"
			}
		}
		_, err2 := this.db.Exec("UPDATE connections_imports SET end_time = $1, error = $2 WHERE id = $3",
			time.Now().UTC(), errorMsg, importID)
		if err2 != nil {
			log.Printf("[error] cannot update the end of import %d into the database: %s", importID, err2)
		}
	}()

	return nil
}

// importError represents a non-internal error during import.
type importError struct {
	err error
}

func (err importError) Error() string {
	return err.err.Error()
}

const (
	identityColumn  = "identity"
	timestampColumn = "timestamp"
)

// startImport starts an import for the given connection.
// It is called by the Import method in its own goroutine.
// The returned error is stored in the databases with the import.
func (this *Connections) startImport(connection *Connection, reimport bool) error {

	const noColumn = -1
	const role = _connector.SourceRole

	connector := connection.connector

	switch connector.typ {
	case AppType:

		// Refresh the access token if necessary.
		var clientSecret, resourceCode, accessToken string
		if r := connection.resource; r != nil {
			expired := time.Now().UTC().Add(15 * time.Minute).After(r.oAuthExpiresIn)
			if r.oAuthAccessToken == "" || expired {
				var err error
				r, err = this.account.apis.Connectors.refreshOAuthToken(connector.id, r.id)
				if err != nil {
					return importError{err}
				}
			}
			clientSecret = connector.oAuth.ClientSecret
			resourceCode = r.code
			accessToken = r.oAuthAccessToken
		}

		// Read the user schema and the properties to read.
		schema, properties, err := this.userSchema(connection.id)
		if err != nil {
			return fmt.Errorf("cannot read user schema: %s", err)
		}

		fh := this.newFirehose(context.Background(), connection, schema)
		c, err := _connector.RegisteredApp(connector.name).Connect(fh.ctx, &_connector.AppConfig{
			Role:         role,
			Settings:     connection.settings,
			Firehose:     fh,
			ClientSecret: clientSecret,
			Resource:     resourceCode,
			AccessToken:  accessToken,
		})
		if err != nil {
			return importError{fmt.Errorf("cannot connect to the connector: %s", err)}
		}
		cursor := connection.userCursor
		if reimport {
			cursor = ""
		}
		err = c.Users(cursor, properties)
		if err != nil {
			return importError{fmt.Errorf("cannot get users from the connector: %s", err)}
		}

		// Handle errors occurred in the firehose.
		if fh.err != nil {
			return fh.err
		}

	case DatabaseType:

		// Read the user schema.
		schema, _, err := this.userSchema(connection.id)
		if err != nil {
			return fmt.Errorf("cannot read user schema: %s", err)
		}

		usersQuery, err := this.compileQuery(connection.usersQuery, noQueryLimit)
		if err != nil {
			return importError{err}
		}
		fh := this.newFirehose(context.Background(), connection, schema)
		c, err := _connector.RegisteredDatabase(connector.name).Connect(fh.ctx, &_connector.DatabaseConfig{
			Role:     role,
			Settings: connection.settings,
			Firehose: fh,
		})
		if err != nil {
			return importError{fmt.Errorf("cannot connect to the connector: %s", err)}
		}
		columns, rows, err := c.Query(usersQuery)
		if err != nil {
			if err, ok := err.(*_connector.DatabaseQueryError); ok {
				return importError{err}
			}
			return err
		}
		defer rows.Close()
		identityIndex := noColumn
		timestampIndex := noColumn
		for i, c := range columns {
			switch c.Name {
			case identityColumn:
				identityIndex = i
			case timestampColumn:
				timestampIndex = i
			}
		}
		if identityIndex == noColumn {
			return importError{fmt.Errorf("missing identity column %q", identityColumn)}
		}
		var now time.Time
		if timestampIndex == noColumn {
			now = time.Now().UTC()
		}
		row := make([]any, len(columns))
		for rows.Next() {
			for i := range row {
				var v string
				row[i] = &v
			}
			if err = rows.Scan(row...); err != nil {
				return importError{fmt.Errorf("cannot read users from database: %s", err)}
			}
			identity := row[identityIndex].(*string)
			var ts time.Time
			if timestampIndex == noColumn {
				ts = now
			} else {
				ts = row[timestampIndex].(time.Time)
			}
			user := map[string]any{}
			for i, c := range columns {
				v := row[i].(*string)
				user[c.Name] = *v
			}
			fh.SetUser(*identity, user, ts, nil)
		}
		if err = rows.Err(); err != nil {
			return importError{fmt.Errorf("an error occurred closing the database: %s", err)}
		}
		// Handle errors occurred in the firehose.
		if fh.err != nil {
			return fh.err
		}

	case EventStreamType:

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		connector := connection.connector
		c, err := _connector.RegisteredEventStream(connector.name).Connect(ctx, &_connector.EventStreamConfig{
			Role:     role,
			Settings: connection.settings,
		})
		if err != nil {
			return importError{fmt.Errorf("cannot connect to the connector: %s", err)}
		}
		defer c.Close()
		event, ack, err := c.Receive()
		if err != nil {
			return err
		}
		ack()
		log.Printf("received event: %s", event)

	case FileType:

		// Read the user schema.
		schema, _, err := this.userSchema(connection.id)
		if err != nil {
			return fmt.Errorf("cannot read user schema: %s", err)
		}

		var ctx = context.Background()

		// Get the file reader.
		var files *fileReader
		{
			fh := this.newFirehose(ctx, connection.storage, schema)
			ctx = fh.ctx
			c, err := _connector.RegisteredStorage(connector.name).Connect(ctx, &_connector.StorageConfig{
				Role:     role,
				Settings: connection.settings,
				Firehose: fh,
			})
			if err != nil {
				return importError{fmt.Errorf("cannot connect to the storage connector: %s", err)}
			}
			files = newFileReader(c)
		}

		// Connect to the file connector.
		fh := this.newFirehose(ctx, connection, types.Schema{})
		file, err := _connector.RegisteredFile(connector.name).Connect(fh.ctx, &_connector.FileConfig{
			Role:     role,
			Settings: connection.settings,
			Firehose: fh,
		})
		if err != nil {
			return importError{fmt.Errorf("cannot connect to the file connector: %s", err)}
		}

		// Read the records.
		records := fh.newRecordWriter(identityColumn, timestampColumn, false)
		err = file.Read(files, records)
		if err != nil {
			return importError{fmt.Errorf("cannot read the file: %s", err)}
		}

		// Handle errors occurred in the firehose.
		if fh.err != nil {
			return fh.err
		}
	}

	return nil
}

// Import represents a connection import.
type Import struct {
	ID        int
	StartTime time.Time
	EndTime   *time.Time
	Error     string
}

// Imports returns all the imports of the source connection with identifier id.
// The connection must be an app, database, event stream or file connection.
// Returns a ConnectionNotFoundError error if the connection does not exist.
func (this *Connections) Imports(id int) ([]*Import, error) {
	if id < 1 || id > maxInt32 {
		return nil, errors.New("invalid connection identifier")
	}
	c, err := this.get(id)
	if err != nil {
		return nil, ConnectionNotFoundError{}
	}
	switch c.connector.typ {
	case AppType, DatabaseType, EventStreamType, FileType:
	default:
		return nil, fmt.Errorf("%s connections cannot have imports",
			strings.ToLower(c.connector.typ.String()))
	}
	if c.role == DestinationRole {
		return nil, errors.New("destination connections cannot have imports")
	}
	imports := []*Import{}
	err = this.db.QueryScan(
		"SELECT i.id, i.start_time, i.end_time, i.error\n"+
			"FROM connections_imports AS i\n"+
			"INNER JOIN connections AS c ON i.connection = c.id\n"+
			"WHERE c.workspace = $1 AND i.connection = $2\n"+
			"ORDER BY i.id DESC", this.id, id, func(rows *postgres.Rows) error {
			var err error
			for rows.Next() {
				var imp Import
				if err = rows.Scan(&imp.ID, &imp.StartTime, &imp.EndTime, &imp.Error); err != nil {
					return err
				}
				imports = append(imports, &imp)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	if len(imports) == 0 {
		_, err := this.get(id)
		if err != nil {
			return nil, ConnectionNotFoundError{}
		}
	}
	return imports, nil
}

// List returns a list of ConnectionInfo describing all connections.
func (this *Connections) List() []*ConnectionInfo {
	this.mu.Lock()
	infos := make([]*ConnectionInfo, len(this.connections))
	i := 0
	for _, c := range this.connections {
		info := ConnectionInfo{
			ID:         c.id,
			Name:       c.name,
			Type:       c.connector.typ,
			Role:       c.role,
			LogoURL:    c.connector.logoURL,
			Enabled:    c.enabled,
			UsersQuery: c.usersQuery,
		}
		if c.storage != nil {
			info.Storage = c.storage.id
		}
		if c.connector.oAuth != nil {
			info.OAuthURL = c.connector.oAuth.URL
		}
		infos[i] = &info
		i++
	}
	this.mu.Unlock()
	sort.Slice(infos, func(i, j int) bool {
		a, b := infos[i], infos[j]
		return a.Name < b.Name || a.Name == b.Name && a.ID == b.ID
	})
	return infos
}

// Schema returns the schema of the connection with identifier id. The
// connection must be an app, database of file connection. If the
// connection does not have a schema, it returns an invalid schema.
//
// Returns a ConnectionNotFoundError error if the connection does not exist.
func (this *Connections) Schema(id int) (types.Schema, error) {
	if id < 1 || id > maxInt32 {
		return types.Schema{}, errors.New("invalid connection identifier")
	}
	c, err := this.get(id)
	if err != nil {
		return types.Schema{}, ConnectionNotFoundError{}
	}
	if c.connector.typ == StorageType {
		return types.Schema{}, errors.New("cannot read properties from a storage")
	}
	return c.schema, nil
}

// Column represents a column of a database connection.
type Column struct {
	Name string
	Type types.Type
}

// Query executes the given query on the database connection with identifier
// id and returns the resulting columns and rows.
//
// query must be UTF-8 encoded, it cannot be longer than 16,777,215 runes and
// must contain the ':limit' placeholder. limit must be between 1 and 100.
//
// It returns an error if the connection is a destination.
// It returns a ConnectionNotFoundError error if the connection does not exist
// and returns a DatabaseQueryError error if an error occurred while executing
// the query.
func (this *Connections) Query(id int, query string, limit int) ([]Column, [][]string, error) {

	if id <= 0 || id > maxInt32 {
		return nil, nil, errors.New("invalid connection identifier")
	}

	if !utf8.ValidString(query) {
		return nil, nil, errors.New("query is not UTF-8 encoded")
	}
	if utf8.RuneCountInString(query) > queryMaxSize {
		return nil, nil, fmt.Errorf("query is longer than %d", queryMaxSize)
	}
	if !strings.Contains(query, ":limit") {
		return nil, nil, errors.New("query does not contain the placeholder \":limit\"")
	}
	if limit < 1 || limit > 100 {
		return nil, nil, errors.New("invalid limit")
	}

	c, err := this.get(id)
	if err != nil {
		return nil, nil, ConnectorNotFoundError{DatabaseType}
	}
	if c.connector.typ != DatabaseType {
		return nil, nil, errors.New("connection is not a database")
	}
	if c.role != SourceRole {
		return nil, nil, errors.New("connection is not a source")
	}

	const cRole = _connector.SourceRole

	// Execute the query.
	query, err = this.compileQuery(query, limit)
	if err != nil {
		return nil, nil, err
	}
	fh := this.newFirehose(context.Background(), c, types.Schema{})
	connection, err := _connector.RegisteredDatabase(c.connector.name).Connect(fh.ctx, &_connector.DatabaseConfig{
		Role:     cRole,
		Settings: c.settings,
		Firehose: fh,
	})
	if err != nil {
		return nil, nil, err
	}
	rawColumns, rawRows, err := connection.Query(query)
	if err != nil {
		if err, ok := err.(*_connector.DatabaseQueryError); ok {
			return nil, nil, &DatabaseQueryError{Message: err.Message}
		}
		return nil, nil, err
	}

	// Fill the columns.
	columns := make([]Column, len(rawColumns))
	for i, c := range rawColumns {
		columns[i].Name = c.Name
		columns[i].Type = c.Type
	}

	// Fill the rows.
	var rows [][]string
	values := make([]any, len(columns))
	for i := range values {
		var value string
		values[i] = &value
	}
	for rawRows.Next() {
		if err := rawRows.Scan(values...); err != nil {
			return nil, nil, err
		}
		row := make([]string, len(rawColumns))
		for i, v := range values {
			row[i] = *(v.(*string))
		}
		rows = append(rows, row)
	}
	err = rawRows.Close()
	if err != nil {
		return nil, nil, err
	}
	if rows == nil {
		rows = [][]string{}
	}

	return columns, rows, nil
}

// ServeUI serves the user interface for the connection with identifier id.
// event is the event and values contains the form values in JSON format.
// Returns a ConnectionNotFoundError error if the connection does not exist and
// the ErrUIEventNotExist error if the event does not exist.
func (this *Connections) ServeUI(id int, event string, values []byte) ([]byte, error) {

	if id < 1 || id > maxInt32 {
		return nil, errors.New("invalid connection identifier")
	}

	c, err := this.get(id)
	if err != nil {
		return nil, ConnectionNotFoundError{}
	}

	cRole := _connector.Role(c.role)

	var connection _connector.Connection

	switch c.connector.typ {
	case AppType:

		// Refresh the access token if necessary.
		var clientSecret, resourceCode, accessToken string
		if r := c.resource; r != nil {
			expired := time.Now().UTC().Add(15 * time.Minute).After(r.oAuthExpiresIn)
			if r.oAuthAccessToken == "" || expired {
				var err error
				r, err = this.account.apis.Connectors.refreshOAuthToken(c.connector.id, r.id)
				if err != nil {
					return nil, err
				}
			}
			clientSecret = c.connector.oAuth.ClientSecret
			resourceCode = r.code
			accessToken = r.oAuthAccessToken
		}

		fh := this.newFirehose(context.Background(), c, types.Schema{})
		connection, err = _connector.RegisteredApp(c.connector.name).Connect(fh.ctx, &_connector.AppConfig{
			Role:         cRole,
			Settings:     c.settings,
			Firehose:     fh,
			ClientSecret: clientSecret,
			Resource:     resourceCode,
			AccessToken:  accessToken,
		})

	default:

		fh := this.newFirehose(context.Background(), c, types.Schema{})

		switch c.connector.typ {
		case DatabaseType:
			connection, err = _connector.RegisteredDatabase(c.connector.name).Connect(fh.ctx, &_connector.DatabaseConfig{
				Role:     cRole,
				Settings: c.settings,
				Firehose: fh,
			})
		case EventStreamType:
			connection, err = _connector.RegisteredEventStream(c.connector.name).Connect(fh.ctx, &_connector.EventStreamConfig{
				Role:     cRole,
				Settings: c.settings,
				Firehose: fh,
			})
		case FileType:
			connection, err = _connector.RegisteredFile(c.connector.name).Connect(fh.ctx, &_connector.FileConfig{
				Role:     cRole,
				Settings: c.settings,
				Firehose: fh,
			})
		case MobileType:
			connection, err = _connector.RegisteredMobile(c.connector.name).Connect(fh.ctx, &_connector.MobileConfig{
				Role:     cRole,
				Settings: c.settings,
				Firehose: fh,
			})
		case ServerType:
			connection, err = _connector.RegisteredServer(c.connector.name).Connect(fh.ctx, &_connector.ServerConfig{
				Role:     cRole,
				Settings: c.settings,
				Firehose: fh,
			})
		case StorageType:
			connection, err = _connector.RegisteredStorage(c.connector.name).Connect(fh.ctx, &_connector.StorageConfig{
				Role:     cRole,
				Settings: c.settings,
				Firehose: fh,
			})
		case WebsiteType:
			connection, err = _connector.RegisteredWebsite(c.connector.name).Connect(fh.ctx, &_connector.WebsiteConfig{
				Role:     cRole,
				Settings: c.settings,
				Firehose: fh,
			})
		}

	}
	if err != nil {
		return nil, err
	}

	// TODO: check and delete alternative fieldsets keys that have 'null' value
	// before saving to database
	form, alert, err := connection.ServeUI(event, values)
	if err != nil {
		if err == ui.ErrEventNotExist {
			err = ErrUIEventNotExist
		}
		return nil, err
	}

	return marshalUIFormAlert(form, alert, c.role)
}

// SetFileStorage sets the storage of the file connection with identifier file.
// storage is the storage connection. The file and the storage must have the
// same role. As a special case, the current storage of the file is removed if
// the storage argument is 0.
//
// It returns a ConnectionNotFound error if the file or storage does not exist.
func (this *Connections) SetFileStorage(file, storage int) error {
	if file < 1 || file > maxInt32 {
		return errors.New("invalid file connection identifier")
	}
	if storage < 0 || storage > maxInt32 {
		return errors.New("invalid storage connection identifier")
	}
	if file == storage {
		return errors.New("file and storage cannot be the same connection")
	}
	f, err := this.get(file)
	if err != nil {
		return ConnectionNotFoundError{FileType}
	}
	var s *Connection
	if storage > 0 {
		s, err = this.get(storage)
		if err != nil {
			return ConnectionNotFoundError{StorageType}
		}
	}
	if f.storage == s {
		return nil
	}
	if s != nil && s.role != f.role {
		if f.role == SourceRole {
			return errors.New("storage connection is not a source")
		}
		return errors.New("storage connection is not a destination")
	}
	err = this.db.Transaction(func(tx *postgres.Tx) error {
		// TODO(marco): check that store, if not zero, still exists
		_, err = tx.Exec("UPDATE connections SET storage = $1 WHERE id = $2", storage, file)
		return err
	})
	if err != nil {
		return err
	}

	this.setStorage(file, storage)

	return err
}

// SetUsersQuery sets the users query of the database connection with
// identifier id. query must be UTF-8 encoded, it cannot be longer than
// 16,777,215 runes and must contain the ':limit' placeholder.
//
// It returns an error if the connection is a destination.
// It returns a ConnectionNotFoundError error if the connection does not exist.
func (this *Connections) SetUsersQuery(id int, query string) error {

	if id < 1 || id > maxInt32 {
		return errors.New("invalid connection identifier")
	}

	if !utf8.ValidString(query) {
		return errors.New("query is not UTF-8 encoded")
	}
	if utf8.RuneCountInString(query) > queryMaxSize {
		return fmt.Errorf("query is longer than %d", queryMaxSize)
	}
	if !strings.Contains(query, ":limit") {
		return errors.New("query does not contain the placeholder \":limit\"")
	}

	c, err := this.get(id)
	if err != nil {
		return ConnectionNotFoundError{DatabaseType}
	}
	if c.connector.typ != DatabaseType {
		return errors.New("connection is not a database")
	}
	if c.role != SourceRole {
		return errors.New("connection is not a source")
	}

	_, err = this.db.Exec("UPDATE connections\nSET users_query = $1 WHERE id = $2", query, id)
	if err != nil {
		return err
	}

	this.setUserQuery(id, query)

	// Reload the schema of the connection.
	go func() {
		err := this.reloadSchema(id)
		if err != nil {
			log.Printf("[error] cannot reload schema for connection %d: %s", id, err)
		}
	}()

	return nil
}

// ConnectionsStats represents the statistics on a connection for the last 24
// hours.
type ConnectionsStats struct {
	UsersIn [24]int // ingested users per hour
}

// Stats returns statistics on the connection with identifier id for the last
// 24 hours. It returns a ConnectionNotFoundError error if the connection does
// not exist.
func (this *Connections) Stats(id int) (*ConnectionsStats, error) {
	if id < 1 || id > maxInt32 {
		return nil, errors.New("invalid connection identifier")
	}
	_, err := this.get(id)
	if err != nil {
		return nil, ConnectionNotFoundError{}
	}
	now := time.Now().UTC()
	toSlot := statsTimeSlot(now)
	fromSlot := toSlot - 23
	stats := &ConnectionsStats{
		UsersIn: [24]int{},
	}
	query := "SELECT time_slot, users_in\nFROM connections_stats\nWHERE connection = $1 AND time_slot BETWEEN $2 AND $3"
	err = this.db.QueryScan(query, id, fromSlot, toSlot, func(rows *postgres.Rows) error {
		var err error
		var slot, usersIn int
		for rows.Next() {
			if err = rows.Scan(&slot, &usersIn); err != nil {
				return err
			}
			stats.UsersIn[slot-fromSlot] = usersIn
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// add adds the connection c to the connections.
// It panics if a connection with the same identifier already exists.
func (this *Connections) add(c *Connection) {
	// TODO(marco) also checks the existence of connection, storage and stream.
	this.mu.Lock()
	defer this.mu.Unlock()
	_, ok := this.connections[c.id]
	if ok {
		panic("attempted to add an already existing connection")
	}
	this.connections[c.id] = c
}

// delete deletes the connection with identifier id from the connections.
// If the connection does not exist, it does nothing.
func (this *Connections) delete(id int) {
	this.mu.Lock()
	delete(this.connections, id)
	this.mu.Unlock()
}

var errConnectionNotFound = errors.New("connection does not exist")

// get returns the connection with identifier id.
// Returns the errConnectionNotFound error if the connection does not exist.
func (this *Connections) get(id int) (*Connection, error) {
	this.mu.Lock()
	defer this.mu.Unlock()
	c, ok := this.connections[id]
	if !ok {
		return nil, errConnectionNotFound
	}
	return c, nil
}

// newFirehose returns a new Firehose used to call a connection method.
func (this *Connections) newFirehose(ctx context.Context, connection *Connection, userSchema types.Schema) *firehose {
	var resource int
	if connection.resource != nil {
		resource = connection.resource.id
	}
	fh := &firehose{
		connections: this,
		connection:  connection,
		resource:    resource,
		userSchema:  userSchema,
	}
	fh.ctx, fh.cancel = context.WithCancel(ctx)
	return fh
}

var errRecordStop = errors.New("stop record")

// reloadSchema reloads the schema of the connection with identifier id. The
// connection must be a source app, database or file.
//
// If the connection does not exist it returns a ConnectionNotFoundError error.
func (this *Connections) reloadSchema(id int) error {

	if id < 1 || id > maxInt32 {
		return errors.New("invalid connection identifier")
	}

	c, err := this.get(id)
	if err != nil {
		return err
	}
	switch c.connector.typ {
	case AppType, DatabaseType:
	case FileType:
		if c.storage == nil {
			return ErrFileHasNoStorage
		}
	default:
		return fmt.Errorf("cannot import properties from a %s connection",
			strings.ToLower(c.connector.typ.String()))
	}
	if c.role == DestinationRole {
		return errors.New("cannot import from a destination")
	}

	cRole := _connector.Role(c.role)

	var schema types.Schema

	switch c.connector.typ {
	case AppType:

		var clientSecret, resourceCode, accessToken string
		if r := c.resource; r != nil {
			// Refresh the access token if necessary.
			expired := time.Now().UTC().Add(15 * time.Minute).After(r.oAuthExpiresIn)
			if r.oAuthAccessToken == "" || expired {
				var err error
				r, err = this.account.apis.Connectors.refreshOAuthToken(c.connector.id, r.id)
				if err != nil {
					return importError{err}
				}
			}
			clientSecret = c.connector.oAuth.ClientSecret
			resourceCode = r.code
			accessToken = r.oAuthAccessToken
		}

		fh := this.newFirehose(context.Background(), c, types.Schema{})
		connection, err := _connector.RegisteredApp(c.connector.name).Connect(fh.ctx, &_connector.AppConfig{
			Role:         cRole,
			Settings:     c.settings,
			Firehose:     fh,
			ClientSecret: clientSecret,
			Resource:     resourceCode,
			AccessToken:  accessToken,
		})
		if err != nil {
			return err
		}
		schema, _, err = connection.Schemas()
		if err != nil {
			return err
		}
		if !schema.Valid() {
			return fmt.Errorf("connection %d returned an invalid schema", id)
		}
		schema = schema.AsRole(types.Role(c.role))
		if !schema.Valid() {
			return errors.New("connection has returned a schema without source properties")
		}

	case DatabaseType:

		usersQuery, err := this.compileQuery(c.usersQuery, 0)
		if err != nil {
			return err
		}
		fh := this.newFirehose(context.Background(), c, types.Schema{})
		connection, err := _connector.RegisteredDatabase(c.connector.name).Connect(fh.ctx, &_connector.DatabaseConfig{
			Role:     cRole,
			Settings: c.settings,
			Firehose: fh,
		})
		if err != nil {
			return err
		}
		columns, rows, err := connection.Query(usersQuery)
		if err != nil {
			return err
		}
		err = rows.Close()
		if err != nil {
			return err
		}
		properties := make([]types.Property, len(columns))
		for i, col := range columns {
			properties[i].Name = col.Name
			properties[i].Type = col.Type
		}
		schema, err = types.SchemaOf(properties)
		if err != nil {
			return fmt.Errorf("connection %d returned an invalid column: %s", id, err)
		}

	case FileType:

		if c.storage == nil {
			return ErrFileHasNoStorage
		}

		var ctx = context.Background()

		// Get the file reader.
		var files *fileReader
		{
			connector := c.storage.connector
			fh := this.newFirehose(ctx, c.storage, types.Schema{})
			ctx = fh.ctx
			connection, err := _connector.RegisteredStorage(connector.name).Connect(ctx, &_connector.StorageConfig{
				Role:     cRole,
				Settings: c.settings,
				Firehose: fh,
			})
			if err != nil {
				return err
			}
			files = newFileReader(connection)
		}

		// Connect to the file connector and read only the columns.
		fh := this.newFirehose(ctx, c, types.Schema{})
		file, err := _connector.RegisteredFile(c.connector.name).Connect(fh.ctx, &_connector.FileConfig{
			Role:     cRole,
			Settings: c.settings,
			Firehose: fh,
		})
		if err != nil {
			return err
		}

		// Read only the columns.
		records := fh.newRecordWriter(identityColumn, timestampColumn, true)
		err = file.Read(files, records)
		if err != nil && err != errRecordStop {
			return err
		}
		properties := make([]types.Property, len(records.columns))
		for i, col := range records.columns {
			properties[i].Name = col.Name
			properties[i].Type = col.Type
		}
		schema, err = types.SchemaOf(properties)
		if err != nil {
			return fmt.Errorf("connection %d returned an invalid column: %s", id, err)
		}

	}

	// Update the schema.
	rawSchema, err := schema.MarshalJSON()
	if err != nil {
		return fmt.Errorf("cannot marshal schema of connection %d: %s", id, err)
	}
	if utf8.RuneCount(rawSchema) > rawSchemaMaxSize {
		return fmt.Errorf("cannot marshal schema of the connection %d: data is too large", id)
	}
	_, err = this.db.Exec("UPDATE connections SET \"schema\" = $1 WHERE id = $2", rawSchema, id)
	if err != nil {
		return err
	}

	this.setUserSchema(id, schema)

	return err
}

// setStorage sets the storage of a file connection. file and storage are the
// identifiers of the file and storage connections.
// If file or storage does not exist, it does nothing.
func (this *Connections) setStorage(file, storage int) {
	cc := new(Connection)
	this.mu.Lock()
	defer this.mu.Unlock()
	f, ok := this.connections[file]
	if !ok {
		return
	}
	s, ok := this.connections[storage]
	if !ok {
		return
	}
	*cc = *f
	cc.storage = s
	this.connections[file] = cc
}

// setUserQuery sets the user query of the connection with identifier id.
// If the connection does not exist, it does nothing.
func (this *Connections) setUserQuery(id int, query string) {
	this.mu.Lock()
	defer this.mu.Unlock()
	c, ok := this.connections[id]
	if !ok || c.usersQuery == query {
		return
	}
	cc := new(Connection)
	*cc = *c
	cc.usersQuery = query
	this.connections[id] = cc
}

// setUserSchema sets the user schema of the connection with identifier id.
// If the connection does not exist, it does nothing.
func (this *Connections) setUserSchema(id int, schema types.Schema) {
	cc := new(Connection)
	this.mu.Lock()
	defer this.mu.Unlock()
	c, ok := this.connections[id]
	if !ok {
		return
	}
	*cc = *c
	cc.schema = schema
	this.connections[id] = cc
}

// userSchema returns the user schema and the paths of the mapped properties of
// the connection with identifier id.
//
// If the connection does not exist it returns a ConnectionNotFoundError error.
func (this *Connections) userSchema(id int) (types.Schema, []_connector.PropertyPath, error) {

	c, err := this.get(id)
	if err != nil {
		return types.Schema{}, nil, ConnectionNotFoundError{}
	}

	// Read the paths of the mapped properties from the transformations of this connection.
	var paths []_connector.PropertyPath
	err = this.db.QueryScan(
		"SELECT property FROM transformations_connections WHERE connection = $1", id, func(rows *postgres.Rows) error {
			var name string
			for rows.Next() {
				if err := rows.Scan(&name); err != nil {
					return err
				}
				paths = append(paths, []string{name})
			}
			return nil
		})
	if err != nil {
		if err == postgres.ErrNoRows {
			return types.Schema{}, nil, ConnectionNotFoundError{}
		}
		return types.Schema{}, nil, err
	}

	// Create a schema with only the properties mapped.
	mapped := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		mapped[p[0]] = struct{}{}
	}
	mappedProperties := make([]types.Property, 0, len(paths))
	for _, property := range c.schema.Properties() {
		if _, ok := mapped[property.Name]; ok {
			mappedProperties = append(mappedProperties, property)
		}
	}
	schema := c.schema
	if mappedProperties != nil {
		schema, err = types.SchemaOf(mappedProperties)
		if err != nil {
			return types.Schema{}, nil, fmt.Errorf("cannot create a new schema from the schema of connection %d: %s", id, err)
		}
	}

	return schema, paths, nil
}

const noQueryLimit = -1

// compileQuery compiles the given query and returns it. If limit is
// noQueryLimit removes the ':limit' placeholder (along with '[[' and ']]');
// otherwise, replaces the placeholders with limit.
func (this *Connections) compileQuery(query string, limit int) (string, error) {
	p := strings.Index(query, ":limit")
	if p == -1 {
		return "", errors.New("missing ':limit' placeholder in query")
	}
	s1 := strings.Index(query[:p], "[[")
	if s1 == -1 {
		return "", errors.New("missing '[[' in query")
	}
	n := len(":limit")
	s2 := strings.Index(query[p+n:], "]]")
	if s2 == -1 {
		return "", errors.New("missing ']]' in query")
	}
	s2 += p + n + 2
	if limit == noQueryLimit {
		return query[:s1] + query[s2:], nil
	}
	return query[:s1] + strings.ReplaceAll(query[s1+2:s2-2], ":limit", strconv.Itoa(limit)) + query[s2:], nil
}

// fileReader implements the connector.FileReader interface.
type fileReader struct {
	s _connector.StorageConnection
}

// newFileReader returns a new file reader for the given storage.
func newFileReader(storage _connector.StorageConnection) *fileReader {
	return &fileReader{s: storage}
}

// Reader returns a ReadCloser from which to read the file at the given
// path and its last update time.
// It is the caller's responsibility to close the returned reader.
func (files *fileReader) Reader(path string) (io.ReadCloser, time.Time, error) {
	return files.s.Reader(path)
}

var bigMaxInt32 = big.NewInt(math.MaxInt32)

// generateConnectionID generates a connection ID in [1, maxInt32].
func generateConnectionID() (int, error) {
	n, err := rand.Int(rand.Reader, bigMaxInt32)
	if err != nil {
		return 0, err
	}
	return int(n.Int64()) + 1, nil
}

// generateAPIKey generates an API key.
func generateAPIKey() (string, error) {
	key := make([]byte, 24)
	_, err := rand.Read(key)
	if err != nil {
		return "", errors.New("cannot generate an API key")
	}
	return base62.EncodeToString(key)[0:32], nil
}

// marshalUIFormAlert marshals form with given role and alert in JSON format.
// form and alert can be nil or not, independently of each other.
func marshalUIFormAlert(form *ui.Form, alert *ui.Alert, role ConnectionRole) ([]byte, error) {

	if form == nil && alert == nil {
		return []byte("null"), nil
	}

	var b bytes.Buffer
	enc := json.NewEncoder(&b)

	b.WriteString("{")

	// Serialize the form, if present.
	if form != nil {

		// Makes the keys of form.Values to have the same case as the Name field of the components.
		values := map[string]any{}
		if len(form.Values) > 0 {
			err := json.Unmarshal(form.Values, &values)
			if err != nil {
				return nil, err
			}
		}

		comma := false
		b.WriteString(`"Form":{"Fields":[`)
		for _, field := range form.Fields {
			ok, err := marshalUIComponent(&b, field, role, values, comma)
			if err != nil {
				return nil, err
			}
			if ok {
				comma = true
			}
		}
		b.WriteString(`],"Actions":`)
		err := enc.Encode(form.Actions)
		if err != nil {
			return nil, err
		}
		if len(form.Values) > 0 {
			b.WriteString(`,"Values":`)
			err = json.NewEncoder(&b).Encode(values)
			if err != nil {
				return nil, err
			}
		}
		b.WriteString("}")

	}

	// Serialize the alert, if present.
	if alert != nil {
		if form != nil {
			b.WriteString(",")
		}
		b.WriteString(`"Alert":{"Message":`)
		err := enc.Encode(alert.Message)
		if err != nil {
			return nil, err
		}
		b.WriteString(`,"Variant":"`)
		b.WriteString(alert.Variant.String())
		b.WriteString(`"`)
		b.WriteString("}")
	}

	b.WriteString(`}`)

	return b.Bytes(), nil
}

// adjustValuesCase adjusts the case of keys of values.
func adjustValuesCase(key string, values map[string]any) {
	var found struct {
		key   string
		value any
	}
	for k, v := range values {
		if strings.EqualFold(k, key) {
			found.key = k
			found.value = v
			break
		}
	}
	if found.key == "" {
		return
	}
	delete(values, found.key)
	values[key] = found.value
}

// marshalUIComponent marshals component with the given role in JSON format. If
// comma is true, it prepends a comma. Returns whether it has been marhalled.
func marshalUIComponent(b *bytes.Buffer, component ui.Component, role ConnectionRole, values map[string]any, comma bool) (bool, error) {
	rv := reflect.ValueOf(component).Elem()
	rt := rv.Type()
	if r := ui.Role(rv.FieldByName("Role").Int()); r != ui.BothRole && ConnectionRole(r) != role {
		return false, nil
	}
	if comma {
		b.WriteString(`,`)
	}
	b.WriteString(`{"ComponentType":"`)
	b.WriteString(rt.Name())
	b.WriteString(`"`)
	for j := 0; j < rt.NumField(); j++ {
		name := rt.Field(j).Name
		if name == "Role" {
			continue
		}
		field := rv.Field(j)
		if name == "Name" && values != nil {
			adjustValuesCase(field.String(), values)
		}
		b.WriteString(`,"`)
		b.WriteString(name)
		b.WriteString(`":`)
		var err error
		switch field := field.Interface().(type) {
		case ui.Component:
			_, err = marshalUIComponent(b, field, role, values, false)
		case []ui.FieldSet:
			b.WriteByte('[')
			comma = false
			for _, set := range field {
				var ok bool
				ok, err = marshalUIFieldSet(b, set, role, values, comma)
				if ok {
					comma = true
				}
			}
			b.WriteByte(']')
		default:
			err = json.NewEncoder(b).Encode(field)
		}
		if err != nil {
			return false, err
		}
	}
	b.WriteString(`}`)
	return true, nil
}

// marshalUIFieldSet marshals fieldSet with the given role in JSON format. If
// comma is true, it prepends a comma. Returns whether it has been marhalled.
func marshalUIFieldSet(b *bytes.Buffer, fieldSet ui.FieldSet, role ConnectionRole, values map[string]any, comma bool) (bool, error) {
	if fieldSet.Role != ui.BothRole && ConnectionRole(fieldSet.Role) != role {
		return false, nil
	}
	name := fieldSet.Name
	if values != nil {
		adjustValuesCase(name, values)
	}
	if comma {
		b.WriteByte(',')
	}
	b.WriteString(`{"Name":`)
	_ = json.NewEncoder(b).Encode(name)
	b.WriteString(`,"Label":`)
	_ = json.NewEncoder(b).Encode(fieldSet.Label)
	b.WriteString(`,"Fields":[`)
	comma = false
	for _, c := range fieldSet.Fields {
		var valuesOfSet map[string]any
		switch vs := values[name].(type) {
		case nil:
		case map[string]any:
			valuesOfSet = vs
		default:
			return false, fmt.Errorf("expected a map[string]any value for field set %s, got %T", name, values[name])
		}
		ok, err := marshalUIComponent(b, c, role, valuesOfSet, comma)
		if err != nil {
			return false, err
		}
		if ok {
			comma = true
		}
	}
	b.WriteString(`]}`)
	return true, nil
}

// abbreviate abbreviates s to almost n runes. If s is longer than n runes,
// the abbreviated string terminates with "...".
func abbreviate(s string, n int) string {
	const spaces = " \n\r\t\f" // https://infra.spec.whatwg.org/#ascii-whitespace
	s = strings.TrimRight(s, spaces)
	if len(s) <= n {
		return s
	}
	if n < 3 {
		return ""
	}
	p := 0
	n2 := 0
	for i := range s {
		switch p {
		case n - 2:
			n2 = i
		case n:
			break
		}
		p++
	}
	if p < n {
		return s
	}
	if p = strings.LastIndexAny(s[:n2], spaces); p > 0 {
		s = strings.TrimRight(s[:p], spaces)
	} else {
		s = ""
	}
	if l := len(s) - 1; l >= 0 && (s[l] == '.' || s[l] == ',') {
		s = s[:l]
	}
	return s + "..."
}
