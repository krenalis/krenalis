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
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/transformations"
	"chichi/apis/types"
	_connector "chichi/connector"
	"chichi/connector/ui"

	"github.com/jxskiss/base62"
	"github.com/open2b/nuts/sql"
)

const maxKeysPerServer = 20 // maximum number of keys per server.

var (
	ConnectorNotExist    errors.Code = "ConnectorNotExist"
	EventNotExist        errors.Code = "EventNotExist"
	InvalidRefreshToken  errors.Code = "InvalidRefreshToken"
	NoStorage            errors.Code = "NoStorage"
	NoMappings           errors.Code = "NoMappings"
	QueryExecutionFailed errors.Code = "QueryExecutionFailed"
	StorageNotExist      errors.Code = "StorageNotExist"
	StreamNotExist       errors.Code = "StreamNotExist"
	TooManyKeys          errors.Code = "TooManyKeys"
	UniqueKey            errors.Code = "UniqueKey"
	WorkspaceNotExist    errors.Code = "WorkspaceNotExist"
)

type Connections struct {
	*Workspace
	state *connectionsState
}

// newConnections returns a new *Connections value.
func newConnections(ws *Workspace, state *connectionsState) *Connections {
	return &Connections{Workspace: ws, state: state}
}

// Connection represents a connection.
type Connection struct {
	account          *Account
	workspace        *Workspace
	id               int
	name             string
	role             ConnectionRole
	enabled          bool
	connector        *Connector
	storage          *Connection
	stream           *Connection
	resource         *Resource
	websiteHost      string
	keys             []string
	userCursor       string
	identityColumn   string
	timestampColumn  string
	settings         []byte
	schema           types.Type
	usersQuery       string
	importInProgress *ImportInProgress
	mappings         []*Mapping
}

// Mapping represents a mapping from a kind of properties to another.
//
// In particular, if the mapping refers to a source connection, it is a mapping
// from the connection properties to a property of the Golden Record; otherwise,
// if it refers to a destination connection, it is a mapping from the Golden
// Record properties to a connection property.
//
// A mapping with just one input property and no source code is a "one to one"
// mapping (without transformation) from a property to another.
type Mapping struct {

	// id is the identifier of the mapping.
	id int

	// connection is the connection.
	connection *Connection

	// in is the schema of the input properties of the mapping. If the
	// connection is a source then the properties are the properties of the
	// connection, otherwise, if it is a destination, it contains the properties
	// of the Golden Record.
	//
	// In case of "one to one" mappings, this schema contains just one property.
	in types.Type

	// predefinedFunc is the predefined transformation function of this mapping,
	// otherwise is zero if this mapping is not a predefined transformation
	// mapping.
	predefinedFunc PredefinedFuncID

	// sourceCode is the source code of the transformation function, which
	// should be something like:
	//
	//    def transform(user):
	//        return {
	//            "FirstName": user["firstname"],
	//        }
	//
	// This is the empty string for "one to one" mappings.
	sourceCode string

	// out is the schema of the output properties of the mapping. If the
	// connection is a source then the properties are the properties of the
	// Golden Record, otherwise, if it is a destination, it contains the
	// properties of the connection.
	out types.Type
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
	Mappings   []*MappingInfo
}

// MappingInfo describes a mapping as returned by Get and List.
type MappingInfo struct {
	ID             int
	In             types.Type       // just one property if it refers to a "one to one" mapping.
	SourceCode     string           // empty string if it refers to a "one to one" mapping.
	PredefinedFunc PredefinedFuncID // zero if not a predefined transformation mapping.
	Out            types.Type       // just one property if it refers to a "one to one" mapping.
}

const (
	maxInt32         = math.MaxInt32
	rawSchemaMaxSize = 16_777_215 // maximum size in runes of the 'schema' column of the 'connections' table.
	queryMaxSize     = 16_777_215 // maximum size in runes of a connection query.
)

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

// ConnectionOptions values are passed to the Add method with options
// relative to the connection.
type ConnectionOptions struct {
	Storage     int
	Stream      int
	WebsiteHost string
	OAuth       *AddConnectionOAuthOptions
}

type AddConnectionOAuthOptions struct {
	RefreshToken string
	AccessToken  string
	ExpiresIn    time.Time
}

// Add adds a connection given its role, connector, name, and, options related
// to the connector and returns its identifier. name cannot be empty and cannot
// be longer than 120 runes.
//
// If the connector, storage or stream does not exist, it returns an
// errors.UnprocessableError error with code ConnectorNotExist, StorageNotExist
// and StreamNotExist respectively.
func (this *Connections) Add(role ConnectionRole, connector int, name string, opts ConnectionOptions) (int, error) {

	if role != SourceRole && role != DestinationRole {
		return 0, errors.BadRequest("role %q is not valid", role)
	}
	if connector < 1 || connector > maxInt32 {
		return 0, errors.BadRequest("connector identifier %d is not valid", connector)
	}
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return 0, errors.BadRequest("name %q is not valid", name)
	}
	if opts.Storage < 0 || opts.Storage > maxInt32 {
		return 0, errors.BadRequest("storage identifier %d is not valid", opts.Storage)
	}
	if opts.Stream < 0 || opts.Stream > maxInt32 {
		return 0, errors.BadRequest("stream identifier %d is not valid", opts.Stream)
	}
	if opts.OAuth != nil {
		if opts.OAuth.AccessToken == "" {
			return 0, errors.BadRequest("OAuth access token is empty")
		}
		if opts.OAuth.RefreshToken == "" {
			return 0, errors.BadRequest("OAuth refresh token is empty")
		}
	}

	n := addConnectionNotification{
		Workspace: this.id,
		Name:      name,
		Role:      role,
		Connector: connector,
	}
	c, ok := this.account.apis.Connectors.state.Get(connector)
	if !ok {
		return 0, errors.Unprocessable(ConnectorNotExist, "connector %d does not exist", connector)
	}

	// Validate the storage.
	if opts.Storage > 0 {
		if c.typ != FileType {
			return 0, errors.BadRequest("connector %d cannot have a storage, it's a %s",
				c.id, strings.ToLower(c.typ.String()))
		}
		s, ok := this.state.Get(opts.Storage)
		if !ok {
			return 0, errors.Unprocessable(StorageNotExist, "storage %d does not exist", opts.Storage)
		}
		if s.connector.typ != StorageType {
			return 0, errors.BadRequest("connection %d is not a storage", opts.Storage)
		}
		if s.role != role {
			if role == SourceRole {
				return 0, errors.BadRequest("storage %d is not a source", opts.Storage)
			}
			return 0, errors.BadRequest("storage %d is not a destination", opts.Storage)
		}
		n.Storage = opts.Storage
	}

	// Validate the stream.
	if opts.Stream > 0 {
		if c.typ == MobileType || c.typ == ServerType || c.typ == WebsiteType {
			return 0, errors.BadRequest("connector %d cannot have a stream, it's a %s",
				c.id, strings.ToLower(c.typ.String()))
		}
		s, ok := this.state.Get(opts.Stream)
		if !ok {
			return 0, errors.Unprocessable(StreamNotExist, "stream %d does not exist", opts.Stream)
		}
		if s.connector.typ != EventStreamType {
			return 0, errors.BadRequest("connection %d is not a stream", opts.Stream)
		}
		if s.role != role {
			if role == SourceRole {
				return 0, errors.BadRequest("stream %d is not a source", opts.Stream)
			}
			return 0, errors.BadRequest("stream %d is not a destination", opts.Stream)
		}
		n.Stream = opts.Stream
	}

	// Validate the website host.
	if opts.WebsiteHost != "" {
		if c.typ != WebsiteType {
			return 0, errors.BadRequest("connector %d cannot have a website host, it's a %s",
				c.id, strings.ToLower(c.typ.String()))
		}
		if h, p, found := strings.Cut(opts.WebsiteHost, ":"); h == "" || len(opts.WebsiteHost) > 255 {
			return 0, errors.BadRequest("website host %q is not valid", opts.WebsiteHost)
		} else if found {
			if port, _ := strconv.Atoi(p); port <= 0 || port > 65535 {
				return 0, errors.BadRequest("website host %q is not valid", opts.WebsiteHost)
			}
		}
		n.WebsiteHost = opts.WebsiteHost
	}

	// Validate OAuth.
	if (opts.OAuth == nil) != (c.oAuth == nil) {
		if opts.OAuth == nil {
			return 0, errors.BadRequest("OAuth is required by connector %d", connector)
		}
		return 0, errors.BadRequest("connector %d does not support OAuth", connector)
	}

	// Set the resource. It can be an existent resource or a resource to be created.
	if opts.OAuth != nil {
		connection, err := _connector.RegisteredApp(c.name).Connect(context.Background(), &_connector.AppConfig{
			Role:         _connector.Role(role),
			ClientSecret: c.oAuth.ClientSecret,
			AccessToken:  opts.OAuth.AccessToken,
		})
		if err != nil {
			return 0, err
		}
		code, err := connection.Resource()
		if err != nil {
			return 0, err
		}
		n.Resource.Code = code
		resource, _ := this.resources.GetByCode(code)
		if resource != nil {
			n.Resource.ID = resource.id
		}
		if resource == nil || opts.OAuth.AccessToken != resource.accessToken ||
			opts.OAuth.RefreshToken != resource.refreshToken ||
			opts.OAuth.ExpiresIn != resource.expiresIn {
			n.Resource.AccessToken = opts.OAuth.AccessToken
			n.Resource.RefreshToken = opts.OAuth.RefreshToken
			n.Resource.ExpiresIn = opts.OAuth.ExpiresIn
		}
	}

	// Generate a connection identifier.
	var err error
	n.ID, err = generateConnectionID()
	if err != nil {
		return 0, err
	}

	// Generate a server key.
	if c.typ == ServerType {
		n.Key, err = generateServerKey()
		if err != nil {
			return 0, err
		}
	}

	err = this.db.Transaction(func(tx *postgres.Tx) error {
		if n.Resource.Code != "" {
			if n.Resource.ID == 0 {
				// Insert a new resource.
				err = tx.QueryRow("INSERT INTO resources (workspace, connector, code, access_token,"+
					" refresh_token, expires_in) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
					n.Workspace, connector, n.Resource.Code, n.Resource.AccessToken, n.Resource.RefreshToken, n.Resource.ExpiresIn).
					Scan(&n.Resource.ID)
			} else if n.Resource.AccessToken != "" {
				// Update the current resource.
				_, err = tx.Exec("UPDATE resources "+
					"SET access_token = $1, refresh_token = $2, expires_in = $3 WHERE id = $4",
					n.Resource.AccessToken, n.Resource.RefreshToken, n.Resource.ExpiresIn, n.Resource.ID)
			}
			if err != nil {
				if postgres.IsForeignKeyViolation(err) {
					switch postgres.ErrConstraintName(err) {
					case "resources_workspace_fkey":
						err = errors.Unprocessable(WorkspaceNotExist, "workspace %d does not exist", n.Workspace)
					case "resources_connector_fkey":
						err = errors.Unprocessable(ConnectorNotExist, "connector %d does not exist", n.Connector)
					}
				}
				return err
			}
		}
		// Insert the connection.
		_, err = tx.Exec("INSERT INTO connections (id, workspace, name, type, role, connector, storage, stream,"+
			" resource, website_host) VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, 0), NULLIF($8, 0), $9, $10)",
			n.ID, n.Workspace, n.Name, c.typ, n.Role, n.Connector, n.Storage, n.Stream, n.Resource.ID, n.WebsiteHost)
		if err != nil {
			if err != nil {
				if postgres.IsForeignKeyViolation(err) {
					switch postgres.ErrConstraintName(err) {
					case "connections_workspace_fkey":
						err = errors.Unprocessable(WorkspaceNotExist, "workspace %d does not exist", n.Workspace)
					case "connections_connector_fkey":
						err = errors.Unprocessable(ConnectorNotExist, "connector %d does not exist", n.Connector)
					case "connections_storage_fkey":
						err = errors.Unprocessable(StorageNotExist, "storage %d does not exist", n.Storage)
					case "connections_stream_fkey":
						err = errors.Unprocessable(StreamNotExist, "stream %d does not exist", n.Stream)
					}
				}
			}
			return err
		}
		if n.Key != nil {
			// Insert the server key.
			_, err = tx.Exec("INSERT INTO connections_keys (connection, value, creation_time) VALUES ($1, $2, $3)",
				n.ID, n.Key, time.Now().UTC())
			if err != nil {
				return err
			}
		}
		return tx.Notify(n)
	})
	if err != nil {
		return 0, err
	}

	return n.ID, nil
}

// Get returns a ConnectionInfo describing the connection with identifier id.
// If the connection does not exist, it returns an errors.NotFoundError error.
func (this *Connections) Get(id int) (*ConnectionInfo, error) {
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("connection identifier %d is not valid", id)
	}
	c, ok := this.state.Get(id)
	if !ok {
		return nil, errors.NotFound("connection %d does not exist", id)
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
	for _, m := range c.mappings {
		info.Mappings = append(info.Mappings, &MappingInfo{
			ID:             m.id,
			In:             m.in,
			PredefinedFunc: m.predefinedFunc,
			SourceCode:     m.sourceCode,
			Out:            m.out,
		})
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
func (this *Connections) Delete(id int) error {
	if id < 1 || id > maxInt32 {
		return errors.BadRequest("connection identifier %d is not valid", id)
	}
	c, ok := this.state.Get(id)
	if !ok {
		return nil
	}
	n := deleteConnectionNotification{
		ID: id,
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		if c.connector.oAuth != nil {
			// Delete the resource of the deleted connection if it has no other connections.
			_, err := tx.Exec("DELETE FROM resources AS r WHERE NOT EXISTS (\n"+
				"\tSELECT FROM connections AS c\n"+
				"\tWHERE r.id = c.resource AND c.id <> $1 AND c.resource IS NULL\n)", id)
			if err != nil {
				return err
			}
		}
		_, err := tx.Exec("DELETE FROM connections WHERE id = $1", id)
		if err != nil {
			return err
		}
		return tx.Notify(n)
	})
	return err
}

// Export starts the export of the users to the connection with the given
// identifier.
//
// If the connection does not exist, it returns an errors.NotFoundError error.
// If the workspace does not have a data warehouse, it returns an
// errors.UnprocessableError error with code NoWarehouse.
// If the connection has no mappings, it returns an errors.UnprocessableError
// error with code NoMappings.
//
// Note that this method is only a draft, and its code may be wrong and/or
// partially implemented.
func (this *Connections) Export(id int) (err error) {

	// Verify that the workspace has a data warehouse.
	if this.warehouse == nil {
		return errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", this.id)
	}

	if id <= 0 || id > maxInt32 {
		return errors.BadRequest("connection identifier %d is not valid", id)
	}

	// Check that the connection exists, has an allowed type and is a
	// destination.
	c, ok := this.state.Get(id)
	if !ok {
		return errors.NotFound("connection %d does not exist", id)
	}
	var storage int
	switch c.connector.typ {
	case AppType:
	default:
		return errors.BadRequest("cannot export to connection %d, it's a %s connection",
			id, strings.ToLower(c.connector.typ.String()))
	}
	if c.role == SourceRole {
		return errors.BadRequest("connection %d is not a destination", id)
	}

	// Check that the connection has at least one mapping associated to it.
	mappings, err := this.Mappings(id)
	if err != nil {
		// TODO(marco): it should not return an internal error if the connection does not exist.
		return fmt.Errorf("cannot list mappings for %d: %s", id, err)
	}
	if len(mappings) == 0 {
		return errors.Unprocessable(NoMappings, "connection %d has no mappings", id)
	}

	// Track the export in the database.
	var exportID int
	err = this.db.QueryRow("INSERT INTO connections_exports (connection, storage, start_time)\n"+
		"VALUES ($1, $2, $3)\nRETURNING id", id, storage, time.Now().UTC()).Scan(&exportID)
	if err != nil {
		return err
	}

	// Start the export.
	go func() {
		err = this.startExport(c)
		var errorMsg string
		if err != nil {
			if e, ok := err.(exportError); ok {
				errorMsg = abbreviate(e.Error(), 1000)
			} else {
				log.Printf("[error] cannot do export %d: %s", exportID, err)
				errorMsg = "an internal error has occurred"
			}
		}
		_, err2 := this.db.Exec("UPDATE connections_exports SET end_time = $1, error = $2 WHERE id = $3",
			time.Now().UTC(), errorMsg, exportID)
		if err2 != nil {
			log.Printf("[error] cannot update the end of export %d into the database: %s", exportID, err2)
		}
	}()

	return nil
}

// GenerateKey generates a new key for the source server with identifier id.
//
// If the server does not exist, it returns an errors.NotFoundError error.
// If the server has already too many keys, it returns an
// errors.UnprocessableError error with code TooManyKeys.
func (this *Connections) GenerateKey(id int) (string, error) {
	if id < 1 || id > maxInt32 {
		return "", errors.BadRequest("connection identifier %d is not valid", id)
	}
	c, ok := this.state.Get(id)
	if !ok {
		return "", errors.NotFound("connection %d does not exist", id)
	}
	if c.connector.typ != ServerType {
		return "", errors.NotFound("connection %d is not a server", id)
	}
	if c.role != SourceRole {
		return "", errors.NotFound("server %d is not a source", id)
	}
	binaryKey, err := generateServerKey()
	if err != nil {
		return "", err
	}
	n := generateConnectionKeyNotification{
		Connection:   id,
		Value:        binaryKey,
		CreationTime: time.Now().UTC(),
	}
	err = this.db.Transaction(func(tx *postgres.Tx) error {
		var count int
		err := tx.QueryRow("SELECT COUNT(*) FROM connections_keys WHERE connection = $1", n.Connection).Scan(&count)
		if err != nil {
			return err
		}
		if count == maxKeysPerServer {
			return errors.Unprocessable(TooManyKeys, "server %d has already %d types", n.Connection, maxKeysPerServer)
		}
		_, err = tx.Exec("INSERT INTO connections_keys (connection, value, creation_time) VALUES ($1, $2, $3)",
			n.Connection, n.Value, n.CreationTime)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				if postgres.ErrConstraintName(err) == "connections_keys_connection_fkey" {
					err = errors.NotFound("connection %d does not exist", n.Connection)
				}
			}
			return err
		}
		return tx.Notify(n)
	})
	if err != nil {
		return "", err
	}

	// Encode the key in the base62 form.
	key := encodeServerKey(n.Value)

	return key, nil
}

// Import starts the import of the users from the connection with the given
// identifier. The connection must be a source app, database or file. If the
// connection is an app and reimport is false, it imports the users from the
// current cursor, otherwise imports all users.
//
// If the connection does not exist, it returns an errors.NotFound error.
// If the workspace does not have a data warehouse, it returns an
// errors.UnprocessableError error with code NoWarehouse.
// If the connection is a file and does not have a storage, it returns an
// errors.UnprocessableError error with code NoStorage.
// If the connection has no mappings, it returns an errors.UnprocessableError
// error with code NoMappings.
func (this *Connections) Import(id int, reimport bool) (err error) {

	// Verify that the workspace has a data warehouse.
	if this.warehouse == nil {
		return errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", this.id)
	}

	if id < 1 || id > maxInt32 {
		return errors.BadRequest("connection identifier %d is not valid", id)
	}

	// Check that the connection exists, has an allowed type and is a source.
	c, ok := this.state.Get(id)
	if !ok {
		return errors.NotFound("connection %d does not exist", id)
	}
	var storage int
	switch c.connector.typ {
	case AppType, DatabaseType, EventStreamType:
	case FileType:
		if c.storage == nil {
			return errors.Unprocessable(NoStorage, "file connection %d does not have a storage", id)
		}
		storage = c.storage.id
	default:
		return errors.BadRequest("cannot import from connection %d, it's a %s connection",
			id, strings.ToLower(c.connector.typ.String()))
	}
	if c.role == DestinationRole {
		return errors.BadRequest("connection %d is not a source", id)
	}

	// Check that the connection has at least one mapping associated to it.
	if c.connector.typ != EventStreamType {
		mappings, err := this.Mappings(id)
		if err != nil {
			// TODO(marco): it should not return an internal error if the connection does not exist.
			return fmt.Errorf("cannot list mappings for %d: %s", id, err)
		}
		if len(mappings) == 0 {
			return errors.Unprocessable(NoMappings, "connection %d has no mappings", id)
		}
	}

	// Track the import in the database.
	n := startImportNotification{
		Connection: id,
		Storage:    storage,
		Reimport:   reimport,
		StartTime:  time.Now().UTC(),
	}
	err = this.db.Transaction(func(tx *postgres.Tx) error {
		err := this.db.QueryRow("INSERT INTO connections_imports (connection, storage, start_time)\n"+
			"VALUES ($1, $2, $3)\nRETURNING id", n.Connection, n.Storage, n.StartTime).Scan(&n.ID)
		if err != nil {
			return err
		}
		return tx.Notify(n)
	})

	return err
}

// importError represents a non-internal error during import.
type importError struct {
	err error
}

func (err importError) Error() string {
	return err.err.Error()
}

// exportError represents a non-internal error during export.
type exportError struct {
	err error
}

func (err exportError) Error() string {
	return err.err.Error()
}

const (
	identityColumn  = "identity"
	timestampColumn = "timestamp"
)

// startImport starts the imp import.
// It is called by the state keeper in its own goroutine.
func (this *Connections) startImport(imp *ImportInProgress) {

	var errorMsg string

	err := this._startImport(imp)
	if err != nil {
		if e, ok := err.(importError); ok {
			errorMsg = abbreviate(e.Error(), 1000)
		} else {
			log.Printf("[error] cannot do import %d: %s", imp.id, err)
			errorMsg = "an internal error has occurred"
		}
	}
	n := endImportNotification{imp.id}
	// TODO(marco) retry if the transaction fails.
	err2 := this.db.Transaction(func(tx *postgres.Tx) error {
		_, err := this.db.Exec("UPDATE connections_imports SET end_time = $1, error = $2 WHERE id = $3",
			time.Now().UTC(), errorMsg, imp.id)
		if err != nil {
			return err
		}
		return tx.Notify(n)
	})
	if err2 != nil {
		log.Printf("[error] cannot update the end of import %d into the database: %s", imp.id, err2)
	}

}

// _startImport is called by the startImport method to start the imp import.
func (this *Connections) _startImport(imp *ImportInProgress) error {

	const noColumn = -1
	const role = _connector.SourceRole

	connector := imp.connection.connector

	switch connector.typ {
	case AppType:

		var clientSecret, resourceCode, accessToken string
		if r := imp.connection.resource; r != nil {
			clientSecret = connector.oAuth.ClientSecret
			resourceCode = r.code
			var err error
			accessToken, err = r.freshAccessToken()
			if err != nil {
				return importError{fmt.Errorf("cannot retrive the OAuth access token: %s", err)}
			}
		}

		// Read the properties to read.
		_, properties, err := this.userSchema(imp.connection.id)
		if err != nil {
			return fmt.Errorf("cannot read user schema: %s", err)
		}

		fh := this.newFirehose(context.Background(), imp.connection)
		c, err := _connector.RegisteredApp(connector.name).Connect(fh.ctx, &_connector.AppConfig{
			Role:         role,
			Settings:     imp.connection.settings,
			Firehose:     fh,
			ClientSecret: clientSecret,
			Resource:     resourceCode,
			AccessToken:  accessToken,
		})
		if err != nil {
			return importError{fmt.Errorf("cannot connect to the connector: %s", err)}
		}
		cursor := imp.connection.userCursor
		if imp.reimport {
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

		usersQuery, err := this.compileQuery(imp.connection.usersQuery, noQueryLimit)
		if err != nil {
			return importError{err}
		}
		fh := this.newFirehose(context.Background(), imp.connection)
		c, err := _connector.RegisteredDatabase(connector.name).Connect(fh.ctx, &_connector.DatabaseConfig{
			Role:     role,
			Settings: imp.connection.settings,
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
		connector := imp.connection.connector
		c, err := _connector.RegisteredEventStream(connector.name).Connect(ctx, &_connector.EventStreamConfig{
			Role:     role,
			Settings: imp.connection.settings,
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

		var ctx = context.Background()

		// Get the file reader.
		var files *fileReader
		{
			fh := this.newFirehose(ctx, imp.connection.storage)
			ctx = fh.ctx
			c, err := _connector.RegisteredStorage(connector.name).Connect(ctx, &_connector.StorageConfig{
				Role:     role,
				Settings: imp.connection.settings,
				Firehose: fh,
			})
			if err != nil {
				return importError{fmt.Errorf("cannot connect to the storage connector: %s", err)}
			}
			files = newFileReader(c)
		}

		// Connect to the file connector.
		fh := this.newFirehose(ctx, imp.connection)
		file, err := _connector.RegisteredFile(connector.name).Connect(fh.ctx, &_connector.FileConfig{
			Role:     role,
			Settings: imp.connection.settings,
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

// startExport starts the export for the connection. Note that this method is
// only a draft, and its code may be wrong and/or partially implemented.
func (this *Connections) startExport(connection *Connection) error {

	const role = _connector.SourceRole

	switch connection.connector.typ {
	case AppType:

		var name, clientSecret, resourceCode, accessToken, refreshToken string
		var webhooksPer WebhooksPer
		var connector, resource int
		var settings []byte
		var expiration time.Time
		err := this.db.QueryRow(
			"SELECT `c`.`name`, `c`.`oAuthClientSecret`, `c`.`webhooksPer` - 1, `r`.`code`,"+
				" `r`.`oAuthAccessToken`, `r`.`oAuthRefreshToken`, `r`.`oAuthExpiresIn`, `s`.`connector`,"+
				" `s`.`resource`, `s`.`settings`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"INNER JOIN `resources` AS `r` ON `r`.`id` = `s`.`resource`\n"+
				"WHERE `s`.`id` = ?", connection.id).Scan(
			&name, &clientSecret, &webhooksPer, &resourceCode, &accessToken, &refreshToken, &expiration, &connector,
			&resource, &settings)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}

		fh := this.newFirehose(context.Background(), connection)
		c, err := _connector.RegisteredApp(name).Connect(fh.ctx, &_connector.AppConfig{
			Role:         role,
			Settings:     settings,
			Firehose:     fh,
			ClientSecret: clientSecret,
			Resource:     resourceCode,
			AccessToken:  accessToken,
		})
		if err != nil {
			return exportError{fmt.Errorf("cannot connect to the connector: %s", err)}
		}

		// Read the mappings.
		mappings, err := this.Mappings(connection.id)
		if err != nil {
			return err
		}

		// Prepare the users to export to the connection.
		users := []_connector.User{}
		{
			// TODO(Gianluca): populate this map:
			internalToExternalID := map[int]string{}
			rows, err := this.db.Query("SELECT user, goldenRecord FROM connection_users WHERE connection = $1", connection.id)
			if err != nil {
				return err
			}
			defer rows.Close()
			toRead := []int{}
			for rows.Next() {
				var user string
				var goldenRecord int
				err := rows.Scan(&user, &goldenRecord)
				if err != nil {
					return err
				}
				toRead = append(toRead, goldenRecord)
			}
			if err := rows.Err(); err != nil {
				return err
			}
			// Read the users from the Golden Record and apply the
			// transformation functions on them.
			grUsers, err := this.readGRUsers(toRead)
			if err != nil {
				return err
			}
			for _, user := range grUsers {
				id := internalToExternalID[user["id"].(int)]
				user, err := exportUser(id, user, mappings)
				if err != nil {
					return err
				}
				users = append(users, user)
			}
		}

		// Export the users to the connection.
		log.Printf("[info] exporting %d user(s) to the connection %d", len(users), connection.id)
		err = c.SetUsers(users)
		if err != nil {
			return errors.New("cannot export users")
		}

		// Handle errors occurred in the firehose.
		if fh.err != nil {
			return fh.err
		}

	default:

		panic(fmt.Sprintf("export to %q not implemented", connection.connector.typ))

	}

	return nil

}

// Keys returns the keys of the source server with identifier id.
//
// If the server does not exist, it returns an errors.NotFoundError error.
func (this *Connections) Keys(id int) ([]string, error) {
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("connection identifier %d is not valid", id)
	}
	c, ok := this.state.Get(id)
	if !ok {
		return nil, errors.NotFound("connection %d does not exist", id)
	}
	if c.connector.typ != ServerType {
		return nil, errors.NotFound("connection %d is not a server", id)
	}
	if c.role != SourceRole {
		return nil, errors.NotFound("server %d is not a source", id)
	}
	keys := make([]string, len(c.keys))
	for i, key := range c.keys {
		keys[i] = encodeServerKey([]byte(key))
	}
	return keys, nil
}

// readGRUsers reads the Golden Record users with the given IDs.
func (this *Connections) readGRUsers(ids []int) ([]map[string]any, error) {
	return nil, nil // TODO(Gianluca): implement.
}

// ImportInProgress represents a connection import in progress.
type ImportInProgress struct {
	id         int
	connection *Connection
	storage    *Connection
	reimport   bool
	startTime  time.Time
}

// An ImportInfo describes a connection import as returned by Imports.
type ImportInfo struct {
	ID        int
	StartTime time.Time
	EndTime   *time.Time
	Error     string
}

// Imports returns a list of ImportInfo describing all imports of the
// connection with identifier id. The connection must be a source app,
// database, or file.
//
// If the connection does not exist, it returns an errors.NotFoundError error.
func (this *Connections) Imports(id int) ([]*ImportInfo, error) {
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("connection identifier %d is not valid", id)
	}
	c, ok := this.state.Get(id)
	if !ok {
		return nil, errors.NotFound("connection %d does not exist", id)
	}
	switch c.connector.typ {
	case AppType, DatabaseType, EventStreamType, FileType:
	default:
		return nil, errors.BadRequest("connection %d cannot have imports, it's a %s connection",
			id, strings.ToLower(c.connector.typ.String()))
	}
	if c.role == DestinationRole {
		return nil, errors.BadRequest("connection %d cannot have imports, it's a destination", id)
	}
	imports := []*ImportInfo{}
	err := this.db.QueryScan(
		"SELECT i.id, i.start_time, i.end_time, i.error\n"+
			"FROM connections_imports AS i\n"+
			"INNER JOIN connections AS c ON i.connection = c.id\n"+
			"WHERE c.workspace = $1 AND i.connection = $2\n"+
			"ORDER BY i.id DESC", this.id, id, func(rows *postgres.Rows) error {
			var err error
			for rows.Next() {
				var imp ImportInfo
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
		if _, ok := this.state.Get(id); !ok {
			return nil, errors.NotFound("connection %d does not exist", id)
		}
	}
	return imports, nil
}

// List returns a list of ConnectionInfo describing all connections.
func (this *Connections) List() []*ConnectionInfo {
	connections := this.state.List()
	infos := make([]*ConnectionInfo, len(connections))
	for i, c := range connections {
		info := ConnectionInfo{
			ID:         c.id,
			Name:       c.name,
			Type:       c.connector.typ,
			Role:       c.role,
			LogoURL:    c.connector.logoURL,
			Enabled:    c.enabled,
			UsersQuery: c.usersQuery,
		}
		for _, t := range c.mappings {
			info.Mappings = append(info.Mappings, &MappingInfo{
				ID:         t.id,
				In:         t.in,
				SourceCode: t.sourceCode,
				Out:        t.out,
			})
		}
		if c.storage != nil {
			info.Storage = c.storage.id
		}
		if c.connector.oAuth != nil {
			info.OAuthURL = c.connector.oAuth.URL
		}
		infos[i] = &info
	}
	sort.Slice(infos, func(i, j int) bool {
		a, b := infos[i], infos[j]
		return a.Name < b.Name || a.Name == b.Name && a.ID == b.ID
	})
	return infos
}

// RevokeKey revokes the given key of the source server with identifier id. key
// cannot be empty and cannot be the unique key of the server.
//
// If the key does not exist, it returns an errors.NotFoundError error.
// If the key is the unique key of the server, it returns an
// errors.UnprocessableError error with code UniqueKey.
func (this *Connections) RevokeKey(id int, key string) error {
	if id < 1 || id > maxInt32 {
		return errors.BadRequest("connection identifier %d is not valid", id)
	}
	if key == "" {
		return errors.BadRequest("key is empty")
	}
	binaryKey, ok := decodeServerKey(key)
	if !ok {
		return errors.BadRequest("key %q is malformed", key)
	}
	c, ok := this.state.Get(id)
	if !ok {
		return errors.NotFound("connection %d does not exist", id)
	}
	if c.connector.typ != ServerType {
		return errors.NotFound("connection %d is not a server", id)
	}
	if c.role != SourceRole {
		return errors.NotFound("server %d is not a source", id)
	}
	n := revokeConnectionKeyNotification{
		Connection: id,
		Value:      binaryKey,
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		var count int
		err := tx.QueryRow("SELECT COUNT(*) FROM connections_keys WHERE connection = $1", n.Connection).Scan(&count)
		if err != nil {
			return err
		}
		if count == 1 {
			return errors.Unprocessable(UniqueKey, "key cannot be revoked because it's the unique key of the server")
		}
		result, err := tx.Exec("DELETE FROM connections_keys WHERE connection = $1 AND value = $2", n.Connection, n.Value)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if affected == 0 {
			return errors.NotFound("key %q does not exist", n.Value)
		}
		return tx.Notify(n)
	})

	return err
}

// Schema returns the schema of the connection with identifier id. The
// connection must be an app, database, or file. If the connection does not
// have a schema, it returns an invalid schema.
//
// If the connection does not exist, it returns an errors.NotFoundError error.
func (this *Connections) Schema(id int) (types.Type, error) {
	if id < 1 || id > maxInt32 {
		return types.Type{}, errors.BadRequest("connection identifier %d is not valid", id)
	}
	c, ok := this.state.Get(id)
	if !ok {
		return types.Type{}, errors.NotFound("connection %d does not exist", id)
	}
	if c.connector.typ == StorageType {
		return types.Type{}, errors.BadRequest("connection %d has no properties, it's a storage", id)
	}
	if c.connector.typ == EventStreamType {
		return types.Type{}, errors.BadRequest("connection %d has no properties, it's a stream", id)
	}
	return c.schema, nil
}

// Mappings returns the mappings of the connection with identifier id.
func (this *Connections) Mappings(connection int) ([]*Mapping, error) {
	ts, ok := this.state.Get(connection)
	if !ok {
		return nil, errors.NotFound("connection %d does not exist", connection)
	}
	return ts.mappings, nil
}

// Column represents a column of a database connection.
type Column struct {
	Name string
	Type types.Type
}

// Query executes the given query on the source database connection with
// identifier id and returns the resulting columns and rows.
//
// query must be UTF-8 encoded, it cannot be longer than 16,777,215 runes and
// must contain the ':limit' placeholder between '[[' and ']]'. limit must be
// between 1 and 100.
//
// If the connection does not exist, it returns an errors.NotFoundError error.
// If the execution of the query fails, it returns an errors.UnprocessableError
// with code QueryExecutionFailed.
func (this *Connections) Query(id int, query string, limit int) ([]Column, [][]string, error) {

	if id < 1 || id > maxInt32 {
		return nil, nil, errors.BadRequest("connection identifier %d is not valid", id)
	}

	if !utf8.ValidString(query) {
		return nil, nil, errors.BadRequest("query is not UTF-8 encoded")
	}
	if utf8.RuneCountInString(query) > queryMaxSize {
		return nil, nil, errors.BadRequest("query is longer than 16,777,215 runes")
	}
	if limit < 1 || limit > 100 {
		return nil, nil, errors.BadRequest("limit %d is not valid", limit)
	}

	c, ok := this.state.Get(id)
	if !ok {
		return nil, nil, errors.NotFound("connector %d does not exist", id)
	}
	if c.connector.typ != DatabaseType {
		return nil, nil, errors.BadRequest("connection %d is not a database", id)
	}
	if c.role != SourceRole {
		return nil, nil, errors.BadRequest("database %d is not a source", id)
	}

	const cRole = _connector.SourceRole

	// Execute the query.
	var err error
	query, err = this.compileQuery(query, limit)
	if err != nil {
		return nil, nil, err
	}
	fh := this.newFirehose(context.Background(), c)
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
			return nil, nil, errors.Unprocessable(QueryExecutionFailed, "query execution of connection %d failed: %w", id, err)
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
//
// If the connection does not exist, it returns an errors.NotFoundError error.
// If the event does not exist, it returns an errors.UnprocessableError error
// with code EventNotExist.
func (this *Connections) ServeUI(id int, event string, values []byte) ([]byte, error) {

	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("connection identifier %d is not valid", id)
	}

	c, ok := this.state.Get(id)
	if !ok {
		return nil, errors.NotFound("connection %d does not exist", id)
	}

	cRole := _connector.Role(c.role)

	var err error
	var connection _connector.Connection

	switch c.connector.typ {
	case AppType:

		var clientSecret, resourceCode, accessToken string
		if r := c.resource; r != nil {
			clientSecret = c.connector.oAuth.ClientSecret
			resourceCode = r.code
			var err error
			accessToken, err = r.freshAccessToken()
			if err != nil {
				return nil, importError{fmt.Errorf("cannot retrive the OAuth access token: %s", err)}
			}
		}

		fh := this.newFirehose(context.Background(), c)
		connection, err = _connector.RegisteredApp(c.connector.name).Connect(fh.ctx, &_connector.AppConfig{
			Role:         cRole,
			Settings:     c.settings,
			Firehose:     fh,
			ClientSecret: clientSecret,
			Resource:     resourceCode,
			AccessToken:  accessToken,
		})

	default:

		fh := this.newFirehose(context.Background(), c)

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
			err = errors.Unprocessable(EventNotExist, "UI event %q does not exist for %s connector",
				event, c.connector.name)
		}
		return nil, err
	}

	return marshalUIFormAlert(form, alert, ui.Role(c.role))
}

// SetStorage sets the storage of the file connection with identifier file.
// storage is the storage connection. The file and the storage must have the
// same role. As a special case, the current storage of the file, if there is
// one, is removed if the storage argument is 0.
//
// If the file does not exist, it returns an errors.NotFoundError error.
// If the storage does not exist, it returns an errors.UnprocessableError error
// with code StorageNotExist.
func (this *Connections) SetStorage(file, storage int) error {

	if file < 1 || file > maxInt32 {
		return errors.BadRequest("file identifier %d is not valid", file)
	}
	if storage < 0 || storage > maxInt32 {
		return errors.BadRequest("storage identifier %d is not valid", storage)
	}

	f, ok := this.state.Get(file)
	if !ok {
		return errors.NotFound("file %d does not exist", file)
	}
	if f.connector.typ != FileType {
		return errors.BadRequest("file is not a file connector")
	}
	var s *Connection
	if storage > 0 {
		s, ok = this.state.Get(storage)
		if !ok {
			return errors.Unprocessable(StorageNotExist, "storage %d does not exist", storage)
		}
		if s.connector.typ != StorageType {
			return errors.BadRequest("connection %d is not a storage", storage)
		}
		if s.role != f.role {
			if f.role == SourceRole {
				return errors.BadRequest("storage %d is not a source", storage)
			}
			return errors.BadRequest("storage %d is not a destination", storage)
		}
	}

	n := setConnectionStorageNotification{
		Connection: file,
		Storage:    storage,
	}

	err := this.db.Transaction(func(tx *postgres.Tx) error {
		result, err := tx.Exec("UPDATE connections SET storage = NULLIF($1, 0) WHERE id = $2", storage, file)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				if postgres.ErrConstraintName(err) == "connections_storage_fkey" {
					err = errors.Unprocessable(StorageNotExist, "storage %d does not exist", storage)
				}
			}
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.NotFound("file %d does not exist", file)
		}
		return tx.Notify(n)
	})

	return err
}

// SetStream sets the stream of the mobile, server or website connection with
// identifier source. stream is the stream connection. The source and the
// stream must have the same role. As a special case, the current stream of the
// source, if there is one, is removed if the stream argument is 0.
//
// If the source does not exist, it returns an errors.NotFoundError error.
// If the stream does exist, it returns an errors.UnprocessableError error with
// code StreamNotExist.
func (this *Connections) SetStream(source, stream int) error {

	if source < 1 || source > maxInt32 {
		return errors.BadRequest("source identifier %d is not valid", source)
	}
	if stream < 0 || stream > maxInt32 {
		return errors.BadRequest("stream identifier %d is not valid", stream)
	}

	c, ok := this.state.Get(source)
	if !ok {
		return errors.NotFound("source %d does not exist", source)
	}
	switch c.connector.typ {
	case MobileType, ServerType, WebsiteType:
	default:
		return errors.BadRequest("source is not a mobile, server or website connector")
	}
	var s *Connection
	if stream > 0 {
		s, ok = this.state.Get(stream)
		if !ok {
			return errors.Unprocessable(StreamNotExist, "stream %d does not exist", stream)
		}
		if s.connector.typ != EventStreamType {
			return errors.BadRequest("connection %d is not a stream", stream)
		}
		if s.role != c.role {
			if c.role == SourceRole {
				return errors.BadRequest("stream %d is not a source", stream)
			}
			return errors.BadRequest("stream %d is not a destination", stream)
		}
	}

	n := setConnectionStreamNotification{
		Connection: source,
		Stream:     stream,
	}

	err := this.db.Transaction(func(tx *postgres.Tx) error {
		result, err := tx.Exec("UPDATE connections SET stream = NULLIF($1, 0) WHERE id = $2", stream, source)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				if postgres.ErrConstraintName(err) == "connections_stream_fkey" {
					err = errors.Unprocessable(StreamNotExist, "stream %d does not exist", stream)
				}
			}
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.NotFound("source %d does not exist", source)
		}
		return tx.Notify(n)
	})

	return err
}

// MappingToCreate represents a mapping to create.
//
// A mapping with just one input property and no source code is a "one to one"
// mapping (without transformation) from a property to another.
type MappingToCreate struct {
	// In is the schema of the input properties of the mapping. If the
	// connection is a source then the properties are the properties of the
	// connection, otherwise, if it is a destination, it contains the properties
	// of the Golden Record.
	//
	// In case of "one to one" mappings, this schema contains just one property.
	In types.Type

	// PredefinedFunc is the predefined transformation function of this mapping,
	// otherwise is zero if this mapping is not a predefined transformation
	// mapping.
	PredefinedFunc PredefinedFuncID

	// SourceCode is the source code of the transformation function, which
	// should be something like:
	//
	//    def transform(user):
	//        return {
	//            "FirstName": user["firstname"],
	//        }
	//
	// In case of "one to one" mappings, this is the empty string.
	SourceCode string

	// out is the schema of the output properties of the mapping. If the
	// connection is a source then the properties are the properties of the
	// Golden Record, otherwise, if it is a destination, it contains the
	// properties of the connection.
	Out types.Type
}

// SetMappings sets the mappings of the connection with identifier id.
func (this *Connections) SetMappings(connection int, mappings []*MappingToCreate) error {

	if connection < 1 || connection > maxInt32 {
		return errors.BadRequest("connection identifier %d is not valid", connection)
	}

	// Validate the mappings.
	for _, t := range mappings {
		// TODO(Gianluca): validate the Python function here.
		if !t.In.Valid() {
			return errors.BadRequest("input schema is invalid")
		}
		if !t.Out.Valid() {
			return errors.BadRequest("output schema is invalid")
		}
		inProps := t.In.PropertiesNames()
		if len(inProps) == 0 {
			return errors.BadRequest("should have at least one input property")
		}
		outProps := t.Out.PropertiesNames()
		if len(outProps) == 0 {
			return errors.BadRequest("should have at least one output property")
		}
		// Validate "one to one" mappings.
		if t.SourceCode == "" && t.PredefinedFunc == 0 {
			if len(inProps) != 1 || len(outProps) != 1 {
				return errors.BadRequest("invalid one-to-one mapping")
			}
		}
		// TODO(Gianluca): validate predefined function input/outputs.
		if t.SourceCode != "" && t.PredefinedFunc > 0 {
			return errors.BadRequest("invalid mapping (cannot have both source code and predefined function)")
		}
		for _, in := range inProps {
			if !types.IsValidPropertyName(in) {
				return errors.BadRequest("input property name %q is not valid", in)
			}
		}
		for _, out := range outProps {
			if !types.IsValidPropertyName(out) {
				return errors.BadRequest("output property name %q is not valid", out)
			}
		}
	}

	n := setConnectionMappingsNotification{Connection: connection}

	// Prepare the mappings for the notification and marshal the schemas into
	// JSON.
	n.Mappings = make([]notifiedMapping, len(mappings))
	inSchemas := make([][]byte, len(mappings))
	outSchemas := make([][]byte, len(mappings))
	for i, t := range mappings {
		n.Mappings[i] = notifiedMapping{
			In:             t.In,
			PredefinedFunc: t.PredefinedFunc,
			SourceCode:     t.SourceCode,
			Out:            t.Out,
		}
		var err error
		inSchemas[i], err = json.Marshal(t.In)
		if err != nil {
			return err
		}
		outSchemas[i], err = json.Marshal(t.Out)
		if err != nil {
			return err
		}
	}

	err := this.db.Transaction(func(tx *postgres.Tx) error {
		_, err := tx.Exec("DELETE FROM connections_mappings WHERE connection = $1", n.Connection)
		if err != nil {
			return err
		}
		query, err := tx.Prepare("INSERT INTO connections_mappings\n" +
			"(connection, \"in\", predefined_func, source_code, out)\n" +
			"VALUES ($1, $2, $3, $4, $5) RETURNING id")
		if err != nil {
			if err != nil {
				if postgres.IsForeignKeyViolation(err) {
					if postgres.ErrConstraintName(err) == "connections_mappings_connection_fkey" {
						err = errors.NotFound("connection %d does not exist", connection)
					}
				}
			}
			return err
		}
		for i, t := range n.Mappings {
			var id int
			err := query.QueryRow(connection, inSchemas[i], t.PredefinedFunc, t.SourceCode, outSchemas[i]).Scan(&id)
			if err != nil {
				return err
			}
			t.ID = id
		}
		return tx.Notify(n)
	})

	return err
}

// SetUsersQuery sets the users query of the database source connection with
// identifier id. query must be UTF-8 encoded, it cannot be longer than
// 16,777,215 runes and must contain the ':limit' placeholder.
//
// If the connection does not exist, it returns an errors.NotFoundError error.
func (this *Connections) SetUsersQuery(id int, query string) error {

	if id < 1 || id > maxInt32 {
		return errors.BadRequest("connection identifier %d is not valid", id)
	}
	if !utf8.ValidString(query) {
		return errors.BadRequest("query is not UTF-8 encoded")
	}
	if utf8.RuneCountInString(query) > queryMaxSize {
		return errors.BadRequest("query is longer than %d", queryMaxSize)
	}
	if !strings.Contains(query, ":limit") {
		return errors.BadRequest("query does not contain the ':limit' placeholder")
	}

	c, ok := this.state.Get(id)
	if !ok {
		return errors.NotFound("connection %d does not exist", id)
	}
	if c.connector.typ != DatabaseType {
		return errors.BadRequest("connection %d is not a database", id)
	}
	if c.role != SourceRole {
		return errors.BadRequest("database %d is not a source", id)
	}

	n := setUserQueryNotification{
		Connection: id,
		Query:      query,
	}

	err := this.db.Transaction(func(tx *postgres.Tx) error {
		result, err := tx.Exec("UPDATE connections\nSET users_query = $1 WHERE id = $2", query, id)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.NotFound("connection %d does not exist", id)
		}
		return tx.Notify(n)
	})

	return err
}

// ConnectionsStats represents the statistics on a connection for the last 24
// hours.
type ConnectionsStats struct {
	UsersIn [24]int // ingested users per hour
}

// Stats returns statistics on the connection with identifier id for the last
// 24 hours.
//
// If the connection does not exist, it returns an errors.Notfound error.
func (this *Connections) Stats(id int) (*ConnectionsStats, error) {
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("connection identifier %d is not valid", id)
	}
	if _, ok := this.state.Get(id); !ok {
		return nil, errors.NotFound("connection %d does not exist", id)
	}
	now := time.Now().UTC()
	toSlot := statsTimeSlot(now)
	fromSlot := toSlot - 23
	stats := &ConnectionsStats{
		UsersIn: [24]int{},
	}
	query := "SELECT time_slot, users_in\nFROM connections_stats\nWHERE connection = $1 AND time_slot BETWEEN $2 AND $3"
	err := this.db.QueryScan(query, id, fromSlot, toSlot, func(rows *postgres.Rows) error {
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

// newFirehose returns a new Firehose used to call a connection method.
func (this *Connections) newFirehose(ctx context.Context, connection *Connection) *firehose {
	var resource int
	if connection.resource != nil {
		resource = connection.resource.id
	}
	fh := &firehose{
		workspace:  this.Workspace,
		connection: connection,
		resource:   resource,
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
		return errors.New("connection identifier is not valid")
	}

	c, ok := this.state.Get(id)
	if !ok {
		return errors.New("connection does not exist")
	}
	switch c.connector.typ {
	case AppType, DatabaseType:
	case FileType:
		if c.storage == nil {
			return errors.New("file connection has not storage")
		}
	default:
		return fmt.Errorf("cannot import properties from a %s connection",
			strings.ToLower(c.connector.typ.String()))
	}
	if c.role == DestinationRole {
		return errors.New("cannot import from a destination")
	}

	cRole := _connector.Role(c.role)

	var schema types.Type

	switch c.connector.typ {
	case AppType:

		var clientSecret, resourceCode, accessToken string
		if r := c.resource; r != nil {
			clientSecret = c.connector.oAuth.ClientSecret
			resourceCode = r.code
			var err error
			accessToken, err = r.freshAccessToken()
			if err != nil {
				return importError{fmt.Errorf("cannot retrive the OAuth access token: %s", err)}
			}
		}

		fh := this.newFirehose(context.Background(), c)
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
		fh := this.newFirehose(context.Background(), c)
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
		schema = types.Object(properties)

	case FileType:

		if c.storage == nil {
			return errors.New("file connection has not storage")
		}

		var ctx = context.Background()

		// Get the file reader.
		var files *fileReader
		{
			connector := c.storage.connector
			fh := this.newFirehose(ctx, c.storage)
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
		fh := this.newFirehose(ctx, c)
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
		schema = types.Object(properties)

	}

	// Update the schema.
	rawSchema, err := schema.MarshalJSON()
	if err != nil {
		return fmt.Errorf("cannot marshal schema of connection %d: %s", id, err)
	}
	if utf8.RuneCount(rawSchema) > rawSchemaMaxSize {
		return fmt.Errorf("cannot marshal schema of the connection %d: data is too large", id)
	}

	n := setConnectionUserSchemaNotification{
		Connection: id,
		Schema:     schema,
	}

	err = this.db.Transaction(func(tx *postgres.Tx) error {
		_, err = tx.Exec("UPDATE connections SET \"schema\" = $1 WHERE id = $2", rawSchema, id)
		if err != nil {
			return err
		}
		return tx.Notify(n)
	})

	return err
}

// userSchema returns the user schema and the paths of the mapped properties of
// the connection with identifier id.
//
// If the connection does not exist it returns a ConnectionNotFoundError error.
func (this *Connections) userSchema(id int) (types.Type, []_connector.PropertyPath, error) {

	c, ok := this.state.Get(id)
	if !ok {
		return types.Type{}, nil, errors.New("connection does not exist")
	}

	// Read the paths of the mapped properties from the transformations of this
	// connection.
	var paths []_connector.PropertyPath
	ts, err := this.Mappings(id)
	if err != nil {
		return types.Type{}, nil, err
	}
	for _, t := range ts {
		for _, in := range t.in.PropertiesNames() {
			paths = append(paths, []string{in})
		}
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
		schema = types.Object(mappedProperties)
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
		return "", errors.BadRequest("query does not contain the ':limit' placeholder")
	}
	s1 := strings.Index(query[:p], "[[")
	if s1 == -1 {
		return "", errors.BadRequest("query does not contain '[['")
	}
	n := len(":limit")
	s2 := strings.Index(query[p+n:], "]]")
	if s2 == -1 {
		return "", errors.BadRequest("query does not contain ']]'")
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

// decodeServerKey decodes a server key encoded as base62 to its binary form.
// It returns an error if key is malformed.
func decodeServerKey(key string) ([]byte, bool) {
	if len(key) != 32 {
		return nil, false
	}
	b, err := base62.DecodeString(key)
	if err != nil {
		return nil, false
	}
	return b, true
}

// encodeServerKey encodes a binary server key to its base62 form and returns
// it.
func encodeServerKey(key []byte) string {
	return base62.EncodeToString(key)[0:32]
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

// generateServerKey generates a server key.
func generateServerKey() ([]byte, error) {
	key := make([]byte, 24)
	_, err := rand.Read(key)
	if err != nil {
		return nil, errors.New("cannot generate a server key")
	}
	return base62.Decode(base62.Encode(key)[0:32])
}

// marshalUIFormAlert marshals form with given role and alert in JSON format.
// form and alert can be nil or not, independently of each other.
func marshalUIFormAlert(form *ui.Form, alert *ui.Alert, role ui.Role) ([]byte, error) {

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
func marshalUIComponent(b *bytes.Buffer, component ui.Component, role ui.Role, values map[string]any, comma bool) (bool, error) {
	rv := reflect.ValueOf(component).Elem()
	rt := rv.Type()
	if role != ui.BothRole {
		if r := ui.Role(rv.FieldByName("Role").Int()); r != ui.BothRole && r != role {
			return false, nil
		}
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
func marshalUIFieldSet(b *bytes.Buffer, fieldSet ui.FieldSet, role ui.Role, values map[string]any, comma bool) (bool, error) {
	if role != ui.BothRole {
		if fieldSet.Role != ui.BothRole && fieldSet.Role != role {
			return false, nil
		}
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

// exportUser returns a user to export (with the given ID) applying the given
// mappings to the properties.
//
// TODO(Gianluca): note that this code has never been tested, as the export
// procedure is still work in progress.
func exportUser(id string, properties map[string]any, mappings []*Mapping) (_connector.User, error) {
	user := _connector.User{
		ID:         id,
		Properties: map[string]any{},
	}
	pool := transformations.NewPool()
	for _, m := range mappings {
		input := map[string]any{}
		inNames := m.in.PropertiesNames()
		for _, in := range inNames {
			input[in] = properties[in]
		}
		outNames := m.out.PropertiesNames()
		if m.sourceCode == "" {
			// "One to one" mapping.
			user.Properties[outNames[0]] = input[inNames[0]]
		} else if m.predefinedFunc != 0 {
			// Predefined transformation.
			in := make([]any, len(inNames))
			for i := range in {
				in[i] = input[inNames[i]]
			}
			out := callPredefinedFunction(m.predefinedFunc, in)
			for i, outName := range outNames {
				user.Properties[outName] = out[i]
			}
		} else {
			// Mapping with a transformation function.
			props, err := pool.Run(context.Background(), m.sourceCode, input)
			if err != nil {
				return _connector.User{}, err
			}
			for name, v := range props {
				user.Properties[name] = v
			}
		}
	}
	return user, nil
}
