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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/types"
	_connector "chichi/connector"
	"chichi/connector/ui"
	"chichi/pkg/open2b/sql"

	"github.com/jxskiss/base62"
)

type Connections struct {
	*WorkspaceAPI
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

// MarshalJSON implements the json.Marshaler interface.
// It panics if role is not a valid ConnectionRole value.
func (role ConnectionRole) MarshalJSON() ([]byte, error) {
	return []byte(`"` + role.String() + `"`), nil
}

// Connection represents a connection.
type Connection struct {
	ID       int
	Name     string
	Type     ConnectorType
	Role     ConnectionRole
	Storage  int // zero if the connection is not a file or does not have a storage
	OAuthURL string
	LogoURL  string
	Enabled  bool
}

// ConnectionInfo represents a connection.
type ConnectionInfo struct {
	ID         int
	Name       string
	Type       ConnectorType
	Role       ConnectionRole
	Storage    int // zero if the connection is not a file or does not have a storage
	LogoURL    string
	Enabled    bool
	UsersQuery string // only for databases.
}

// AddApp adds an app connection given its role, app connector, name, OAuth
// refresh and access tokens and returns its identifier. name cannot be empty
// and cannot be longer than 120 runes.
//
// If the connector does not exist, it returns a ConnectorNotFoundError error.
func (this *Connections) AddApp(role ConnectionRole, connector int, name string, refreshToken, accessToken, expiresIn string) (int, error) {
	if role != SourceRole && role != DestinationRole {
		return 0, errors.New("invalid role")
	}
	if connector <= 0 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
	}
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return 0, errors.New("invalid name")
	}
	conn, err := this.api.apis.Connector(connector)
	if err != nil {
		return 0, err
	}
	if conn == nil {
		return 0, ConnectorNotFoundError{AppType}
	}
	if conn.Type != AppType {
		return 0, errors.New("connector is not an app connector")
	}
	c, err := _connector.RegisteredApp(conn.Name).Connect(context.Background(), &_connector.AppConfig{
		Role:         _connector.Role(role),
		ClientSecret: conn.OAuth.ClientSecret,
		AccessToken:  accessToken,
	})
	if err != nil {
		return 0, err
	}
	resourceCode, err := c.Resource()
	if err != nil {
		return 0, err
	}
	var id int
	err = this.myDB.Transaction(func(tx *sql.Tx) error {
		var resource int
		var currentRefreshToken string
		err := tx.QueryRow("SELECT `id`, `oauth_refresh_token` FROM `resources` WHERE `connector` = ? AND `code` = ?",
			connector, resourceCode).Scan(&resource, &currentRefreshToken)
		if err != nil {
			if err != sql.ErrNoRows {
				return err
			}
			err = nil
		}
		if resource == 0 {
			result, err := tx.Exec("INSERT INTO `resources` "+
				"SET `connector` = ?, `code` = ?, `oauth_access_token` = ?, `oauth_refresh_token` = ?, `oauth_expires_in` = ?",
				connector, resourceCode, accessToken, refreshToken, expiresIn)
			if err != nil {
				return err
			}
			resourceID, err := result.LastInsertId()
			resource = int(resourceID)
		} else if refreshToken != currentRefreshToken {
			_, err = tx.Exec("UPDATE `resources` "+
				"SET `oauth_access_token` = ?, `oauth_refresh_token` = ?, `oauth_expires_in` = ? WHERE `id` = ?",
				accessToken, refreshToken, expiresIn, resource)
		}
		if err != nil {
			return err
		}
		id, err = generateConnectionID()
		if err != nil {
			return err
		}
		_, err = tx.Exec("INSERT INTO `connections`\n"+
			"SET `id` = ?, `workspace` = ?, `name` = ?, `type` = 'App', `role` = ?, `connector` = ?, `resource` = ?",
			id, this.workspace, name, role, connector, resource)
		return err
	})
	if err != nil {
		return 0, err
	}

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
	if connector <= 0 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
	}
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return 0, errors.New("invalid name")
	}
	var id int
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		var connectorType ConnectorType
		err := tx.QueryRow("SELECT CAST(`type` AS UNSIGNED) FROM `connectors` WHERE `id` = ?", connector).Scan(&connectorType)
		if err != nil {
			if err == sql.ErrNoRows {
				return ConnectorNotFoundError{DatabaseType}
			}
			return err
		}
		if connectorType != DatabaseType {
			return errors.New("connector is not a database connector")
		}
		id, err = generateConnectionID()
		if err != nil {
			return err
		}
		_, err = tx.Exec("INSERT INTO `connections`\n"+
			"SET `id` = ?, `workspace` = ?, `name` = ?, `type` = 'Database', `role` = ?, `connector` = ?",
			id, this.workspace, name, role, connector)
		return err
	})
	if err != nil {
		return 0, err
	}
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
	if connector <= 0 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
	}
	if storage < 0 || storage > maxInt32 {
		return 0, errors.New("invalid storage")
	}
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return 0, errors.New("invalid name")
	}
	var id int
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		var typ ConnectorType
		err := tx.QueryRow("SELECT CAST(`type` AS UNSIGNED) FROM `connectors` WHERE `id` = ?", connector).Scan(&typ)
		if err != nil {
			if err == sql.ErrNoRows {
				return ConnectorNotFoundError{FileType}
			}
			return err
		}
		if typ != FileType {
			return errors.New("connector is not a file connector")
		}
		if storage > 0 {
			// Check the storage.
			var storageRole ConnectionRole
			err = tx.QueryRow("SELECT CAST(`type` AS UNSIGNED), CAST(`role` AS UNSIGNED) FROM `connections` WHERE `id` = ?",
				storage).Scan(&typ, &storageRole)
			if err != nil {
				if err == sql.ErrNoRows {
					return ConnectionNotFoundError{StorageType}
				}
				return err
			}
			if typ != StorageType {
				return errors.New("storage is not a storage connection")
			}
			if storageRole != role {
				if role == SourceRole {
					return errors.New("storage is not a source")
				}
				return errors.New("storage is not a destination")
			}
		}
		id, err = generateConnectionID()
		if err != nil {
			return err
		}
		_, err = tx.Exec("INSERT INTO `connections`\n"+
			"SET `id` = ?, `workspace` = ?, `name` = ?, `type` = 'File', `role` = ?, `connector` = ?, `storage` = ?",
			id, this.workspace, name, role, connector, storage)
		return err
	})
	if err != nil {
		return 0, err
	}
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
	if connector <= 0 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
	}
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return 0, errors.New("invalid name")
	}
	var id int
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		var typ ConnectorType
		err := tx.QueryRow("SELECT CAST(`type` AS UNSIGNED) FROM `connectors` WHERE `id` = ?", connector).Scan(&typ)
		if err != nil {
			if err == sql.ErrNoRows {
				return ConnectorNotFoundError{EventStreamType}
			}
			return err
		}
		if typ != EventStreamType {
			return errors.New("connector is not an event stream connector")
		}
		id, err = generateConnectionID()
		if err != nil {
			return err
		}
		_, err = tx.Exec("INSERT INTO `connections`\n"+
			"SET `id` = ?, `workspace` = ?, `name` = ?, `type` = 'EventStream', `role` = ?, `connector` = ?",
			id, this.workspace, name, role, connector)
		return err
	})
	if err != nil {
		return 0, err
	}
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
	if connector <= 0 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
	}
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return 0, errors.New("invalid name")
	}
	// Generate the API key.
	key, err := generateAPIKey()
	if err != nil {
		return 0, err
	}
	var id int
	err = this.myDB.Transaction(func(tx *sql.Tx) error {
		var typ ConnectorType
		err := tx.QueryRow("SELECT CAST(`type` AS UNSIGNED) FROM `connectors` WHERE `id` = ?", connector).Scan(&typ)
		if err != nil {
			if err == sql.ErrNoRows {
				return ConnectorNotFoundError{ServerType}
			}
			return err
		}
		if typ != ServerType {
			return errors.New("connector is not a server connector")
		}
		id, err = generateConnectionID()
		if err != nil {
			return err
		}
		_, err = tx.Exec("INSERT INTO `connections`\n"+
			"SET `id` = ?, `workspace` = ?, `name` = ?, `type` = 'Server', `role` = ?, `connector` = ?",
			id, this.workspace, name, role, connector)
		if err != nil {
			return err
		}
		_, err = tx.Exec("INSERT INTO `connections_keys` (`connection`, `position`, `key`) VALUE (?, 0, ?)", id, key)
		return err
	})
	if err != nil {
		return 0, err
	}
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
	if connector <= 0 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
	}
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return 0, errors.New("invalid name")
	}
	var id int
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		var typ ConnectorType
		err := tx.QueryRow("SELECT CAST(`type` AS UNSIGNED) FROM `connectors` WHERE `id` = ?", connector).Scan(&typ)
		if err != nil {
			if err == sql.ErrNoRows {
				return ConnectorNotFoundError{MobileType}
			}
			return err
		}
		if typ != MobileType {
			return errors.New("connector is not a mobile connector")
		}
		id, err = generateConnectionID()
		if err != nil {
			return err
		}
		_, err = tx.Exec("INSERT INTO `connections`\n"+
			"SET `id` = ?, `workspace` = ?, `name` = ?, `type` = 'Server', `role` = ?, `connector` = ?",
			id, this.workspace, name, role, connector)
		return err
	})
	if err != nil {
		return 0, err
	}
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
	if connector <= 0 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
	}
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return 0, errors.New("invalid name")
	}
	var id int
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		var typ ConnectorType
		err := tx.QueryRow("SELECT CAST(`type` AS UNSIGNED) FROM `connectors` WHERE `id` = ?", connector).Scan(&typ)
		if err != nil {
			if err == sql.ErrNoRows {
				return ConnectorNotFoundError{StorageType}
			}
			return err
		}
		if typ != StorageType {
			return errors.New("connector is not a storage connector")
		}
		id, err = generateConnectionID()
		if err != nil {
			return err
		}
		_, err = tx.Exec("INSERT INTO `connections`\n"+
			"SET `id` = ?, `workspace` = ?, `name` = ?, `type` = 'Storage', `role` = ?, `connector` = ?",
			id, this.workspace, name, role, connector)
		return err
	})
	if err != nil {
		return 0, err
	}
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
	if connector <= 0 || connector > maxInt32 {
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
	var id int
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		var typ ConnectorType
		err := tx.QueryRow("SELECT CAST(`type` AS UNSIGNED) FROM `connectors` WHERE `id` = ?", connector).Scan(&typ)
		if err != nil {
			if err == sql.ErrNoRows {
				return ConnectorNotFoundError{WebsiteType}
			}
			return err
		}
		// Validate the type.
		if typ != WebsiteType {
			return errors.New("connector is not an website connector")
		}
		id, err = generateConnectionID()
		if err != nil {
			return err
		}
		_, err = tx.Exec("INSERT INTO `connections`\n"+
			"SET `id` = ?, `workspace` = ?, `name` = ?, `type` = 'Website', `role` = ?, `connector` = ?, `website_host` = ?",
			id, this.workspace, name, role, connector, host)
		return err
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

// Get returns the connection with identifier id. If the connection does not
// exist, it returns a ConnectionNotFoundError error.
func (this *Connections) Get(id int) (*ConnectionInfo, error) {
	if id <= 0 || id > maxInt32 {
		return nil, errors.New("invalid connection identifier")
	}
	s := ConnectionInfo{ID: id}
	err := this.myDB.QueryRow(
		"SELECT `s`.`name`, CAST(`s`.`type` AS UNSIGNED), CAST(`s`.`role` AS UNSIGNED), `c`.`logo_url`,"+
			" `s`.`enabled`, `s`.`users_query`\n"+
			"FROM `connections` AS `s`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
			"WHERE `s`.`id` = ? AND `s`.`workspace` = ?",
		id, this.workspace).Scan(&s.Name, &s.Type, &s.Role, &s.LogoURL, &s.Enabled, &s.UsersQuery)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ConnectionNotFoundError{}
		}
	}
	return &s, nil
}

// Delete deletes the connection with the given identifier.
// If the connection does not exist, it does nothing.
//
// If the connection is a storage and has connected files, it returns the
// ErrStorageHasConnectedFiles error.
func (this *Connections) Delete(id int) error {
	if id <= 0 || id > maxInt32 {
		return errors.New("invalid connection identifier")
	}
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		connection, err := tx.Table("Connections").Get(
			sql.Where{"id": id, "workspace": this.workspace},
			sql.Columns{"CAST(`type` AS UNSIGNED) AS `type`", "resource"})
		if err != nil {
			return err
		}
		if connection == nil {
			return nil
		}
		typ := ConnectorType(connection["type"].(int))
		if typ == StorageType {
			hasFiles, err := tx.Table("Connections").Exists(sql.Where{"workspace": this.workspace, "storage": id})
			if err != nil {
				return err
			}
			if hasFiles {
				return ErrStorageHasConnectedFiles
			}
		}
		_, err = tx.Table("Connections").Delete(sql.Where{"id": id})
		if err != nil {
			return err
		}
		_, err = tx.Table("ConnectionsKeys").Delete(sql.Where{"connection": id})
		if err != nil {
			return err
		}
		_, err = tx.Table("ConnectionsImports").Delete(sql.Where{"connection": id})
		if err != nil {
			return err
		}
		_, err = tx.Table("ConnectionsStats").Delete(sql.Where{"connection": id})
		if err != nil {
			return err
		}
		var connectionColumn string
		switch typ {
		case MobileType, WebsiteType:
			connectionColumn = "source"
		case ServerType:
			connectionColumn = "server"
		case EventStreamType:
			connectionColumn = "stream"
		}
		if connectionColumn != "" {
			_, err = tx.Table("ConnectionsStatsEvents").Delete(sql.Where{connectionColumn: id})
			if err != nil {
				return err
			}
		}
		_, err = tx.Table("ConnectionsUsers").Delete(sql.Where{"connection": id})
		if err != nil {
			return err
		}
		// Delete the resource of the deleted connection if it has no other connections.
		_, err = tx.Exec("DELETE `r`\n"+
			"FROM `resources` AS `r`\n"+
			"LEFT JOIN `connections` AS `s` ON `s`.`resource` = `r`.`id`\n"+
			"WHERE `r`.`id` = ? AND `s`.`resource` IS NULL", connection["resource"])
		return err
	})
	return err
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

	if id <= 0 || id > maxInt32 {
		return errors.New("invalid connection identifier")
	}

	// Check that the connection exists, has an allowed type and is a source.
	var typ ConnectorType
	var role ConnectionRole
	var storage int
	err = this.myDB.QueryRow("SELECT CAST(`type` AS UNSIGNED), CAST(`role` AS UNSIGNED), `storage`\n"+
		"FROM `connections`"+
		"WHERE `id` = ? AND `workspace` = ?", id, this.workspace).Scan(&typ, &role, &storage)
	if err != nil {
		if err == sql.ErrNoRows {
			return ConnectionNotFoundError{}
		}
		return err
	}
	switch typ {
	case AppType, DatabaseType, EventStreamType:
	case FileType:
		if storage == 0 {
			return ErrFileHasNoStorage
		}
	default:
		return fmt.Errorf("cannot import from a %s connection", strings.ToLower(typ.String()))
	}
	if role == DestinationRole {
		return errors.New("cannot import from a destination")
	}

	// Check that the connection has at least one transformation associated to it.
	if typ != EventStreamType {
		transformations, err := this.Transformations.List(id)
		if err != nil {
			return fmt.Errorf("cannot list transformations for %d: %s", id, err)
		}
		if len(transformations) == 0 {
			return ErrConnectionDisabled
		}
	}

	// Track the import in the database.
	importID, err := this.myDB.Table("ConnectionsImports").Add(sql.Set{
		"connection": id,
		"storage":    storage,
		"start_time": time.Now().UTC(),
	}, nil)
	if err != nil {
		return err
	}

	// Start the import.
	go func() {
		err = this.startImport(id, typ, reimport)
		var errorMsg string
		if err != nil {
			if e, ok := err.(importError); ok {
				errorMsg = abbreviate(e.Error(), 1000)
			} else {
				log.Printf("[error] cannot do import %d: %s", importID, err)
				errorMsg = "an internal error has occurred"
			}
		}
		_, err2 := this.myDB.Table("ConnectionsImports").Update(
			sql.Set{"end_time": time.Now().UTC(), "error": errorMsg},
			sql.Where{"id": importID})
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

// startImport starts an import for the connection with identifier id and type
// typ. It is called by the Import method in its own goroutine.
// The returned error is stored in the databases with the import.
func (this *Connections) startImport(id int, typ ConnectorType, reimport bool) error {

	const noColumn = -1
	const role = _connector.SourceRole

	switch typ {
	case AppType:

		var name, clientSecret, resourceCode, accessToken, refreshToken, cursor string
		var webhooksPer WebhooksPer
		var connector, resource int
		var settings []byte
		var expiration time.Time
		err := this.myDB.QueryRow(
			"SELECT `c`.`name`, `c`.`oauth_client_secret`, `c`.`webhooks_per` - 1, `r`.`code`,"+
				" `r`.`oauth_access_token`, `r`.`oauth_refresh_token`, `r`.`oauth_expires_in`, `s`.`connector`,"+
				" `s`.`resource`, `s`.`user_cursor`, `s`.`settings`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"INNER JOIN `resources` AS `r` ON `r`.`id` = `s`.`resource`\n"+
				"WHERE `s`.`id` = ?", id).Scan(
			&name, &clientSecret, &webhooksPer, &resourceCode, &accessToken, &refreshToken, &expiration, &connector,
			&resource, &cursor, &settings)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}
		if reimport {
			cursor = ""
		}

		// Read the user schema and the properties to read.
		schema, properties, err := this.userSchema(id)
		if err != nil {
			return fmt.Errorf("cannot read user schema: %s", err)
		}

		if hasOAuth := clientSecret != ""; hasOAuth {
			accessTokenExpired := time.Now().UTC().Add(15 * time.Minute).After(expiration)
			if accessToken == "" || accessTokenExpired {
				accessToken, err = this.api.apis.refreshOAuthToken(resource)
				if err != nil {
					return importError{err}
				}
			}
		}

		fh := this.newFirehose(context.Background(), id, connector, resource, schema, typ, role, webhooksPer)
		c, err := _connector.RegisteredApp(name).Connect(fh.ctx, &_connector.AppConfig{
			Role:         role,
			Settings:     settings,
			Firehose:     fh,
			ClientSecret: clientSecret,
			Resource:     resourceCode,
			AccessToken:  accessToken,
		})
		if err != nil {
			return importError{fmt.Errorf("cannot connect to the connector: %s", err)}
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

		var connectorName, usersQuery string
		var connector int
		var settings []byte
		err := this.myDB.QueryRow(
			"SELECT `c`.`name`, `s`.`connector`,  `s`.`settings`, `s`.`users_query`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"WHERE `s`.`id` = ?", id).Scan(&connectorName, &connector, &settings, &usersQuery)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}

		// Read the user schema.
		schema, _, err := this.userSchema(id)
		if err != nil {
			return fmt.Errorf("cannot read user schema: %s", err)
		}

		usersQuery, err = this.compileQuery(usersQuery, noQueryLimit)
		if err != nil {
			return importError{err}
		}
		fh := this.newFirehose(context.Background(), id, connector, 0, schema, DatabaseType, role, WebhooksPerNone)
		c, err := _connector.RegisteredDatabase(connectorName).Connect(fh.ctx, &_connector.DatabaseConfig{
			Role:     role,
			Settings: settings,
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

		var connectorName string
		var settings []byte
		err := this.myDB.QueryRow(
			"SELECT `c`.`name`, `s`.`settings`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"WHERE `s`.`id` = ?", id).Scan(&connectorName, &settings)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c, err := _connector.RegisteredEventStream(connectorName).Connect(ctx, &_connector.EventStreamConfig{
			Role:     role,
			Settings: settings,
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

		var connectorName, identityColumn, timestampColumn string
		var connector, storage int
		var settings []byte
		err := this.myDB.QueryRow(
			"SELECT `c`.`name`, `s`.`connector`, `s`.`storage`, `s`.`identity_column`, `s`.`timestamp_column`, `s`.`settings`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"WHERE `s`.`id` = ?", id).Scan(&connectorName, &connector, &storage, &identityColumn, &timestampColumn, &settings)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}
		if storage == 0 {
			return importError{errors.New("connector has no more a storage")}
		}

		// Read the user schema.
		schema, _, err := this.userSchema(id)
		if err != nil {
			return fmt.Errorf("cannot read user schema: %s", err)
		}

		var ctx = context.Background()

		// Get the file reader.
		var files *fileReader
		{
			var connectorName string
			var connector int
			var settings []byte
			err = this.myDB.QueryRow(
				"SELECT `c`.`name`, `s`.`connector`, `s`.`settings`\n"+
					"FROM `connections` AS `s`\n"+
					"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
					"WHERE `s`.`id` = ?", storage).Scan(&connectorName, &connector, &settings)
			if err != nil {
				if err != sql.ErrNoRows {
					return nil
				}
				return err
			}
			fh := this.newFirehose(ctx, storage, connector, 0, schema, StorageType, role, WebhooksPerNone)
			ctx = fh.ctx
			c, err := _connector.RegisteredStorage(connectorName).Connect(ctx, &_connector.StorageConfig{
				Role:     role,
				Settings: settings,
				Firehose: fh,
			})
			if err != nil {
				return importError{fmt.Errorf("cannot connect to the storage connector: %s", err)}
			}
			files = newFileReader(c)
		}

		// Connect to the file connector.
		fh := this.newFirehose(ctx, id, connector, 0, types.Schema{}, FileType, role, WebhooksPerNone)
		file, err := _connector.RegisteredFile(connectorName).Connect(fh.ctx, &_connector.FileConfig{
			Role:     role,
			Settings: settings,
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
	EndTime   time.Time
	Error     string
}

// Imports returns all the imports of the source connection with identifier id.
// The connection must be an app, database, event stream or file connection.
// Returns a ConnectionNotFoundError error if the connection does not exist.
func (this *Connections) Imports(id int) ([]*Import, error) {
	if id <= 0 || id > maxInt32 {
		return nil, errors.New("invalid connection identifier")
	}
	imports := []*Import{}
	err := this.myDB.QueryScan(
		"SELECT `i`.`id`, `i`.`start_time`, `i`.`end_time`, `i`.`error`\n"+
			"FROM `connections_imports` AS `i`\n"+
			"INNER JOIN `connections` AS `c` ON `i`.`connection` = `c`.`id`\n"+
			"WHERE `c`.`workspace` = ? AND `i`.`connection` = ?\n"+
			"ORDER BY `i`.`id` DESC", this.workspace, id, func(rows *sql.Rows) error {
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
		var typ ConnectorType
		var role ConnectionRole
		err = this.myDB.QueryRow(
			"SELECT CAST(`type` AS UNSIGNED), CAST(`role` AS UNSIGNED) FROM `connections` WHERE `id` = ? AND `workspace` = ?",
			id, this.workspace).Scan(&typ, &role)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, ConnectionNotFoundError{}
			}
			return nil, err
		}
		switch typ {
		case AppType, DatabaseType, EventStreamType, FileType:
		default:
			return nil, fmt.Errorf("%s connections cannot have imports", strings.ToLower(typ.String()))
		}
		if role == DestinationRole {
			return nil, errors.New("destination connections cannot have imports")
		}
	}
	return imports, nil
}

// List returns all connections.
func (this *Connections) List() ([]*Connection, error) {
	sources := []*Connection{}
	err := this.myDB.QueryScan(
		"SELECT `s`.`id`, `s`.`name`, CAST(`s`.`type` AS UNSIGNED), CAST(`s`.`role` AS UNSIGNED), `s`.`storage`,"+
			" `c`.`oauth_url`, `c`.`logo_url`, `s`.`enabled`\n"+
			"FROM `connections` as `s`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
			"WHERE `s`.`workspace` = ?", this.workspace, func(rows *sql.Rows) error {
			var err error
			for rows.Next() {
				var c Connection
				if err = rows.Scan(&c.ID, &c.Name, &c.Type, &c.Role, &c.Storage, &c.OAuthURL, &c.LogoURL, &c.Enabled); err != nil {
					return err
				}
				sources = append(sources, &c)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return sources, nil
}

// Schema returns the schema of the connection with identifier id. The
// connection must be an app, database of file connection. If the
// connection does not have a schema, it returns an invalid schema.
//
// Returns a ConnectionNotFoundError error if the connection does not exist.
func (this *Connections) Schema(id int) (types.Schema, error) {
	if id <= 0 || id > maxInt32 {
		return types.Schema{}, errors.New("invalid connection identifier")
	}
	var typ ConnectorType
	var rawSchema string
	err := this.myDB.QueryRow("SELECT CAST(`type` AS UNSIGNED), `schema`\n"+
		"FROM `connections`\n"+
		"WHERE `id` = ? AND `workspace` = ?", id, this.workspace).Scan(&typ, &rawSchema)
	if err != nil {
		if err == sql.ErrNoRows {
			return types.Schema{}, ConnectionNotFoundError{}
		}
		return types.Schema{}, err
	}
	if typ == StorageType {
		return types.Schema{}, errors.New("cannot read properties from a storage")
	}
	if len(rawSchema) == 0 {
		return types.Schema{}, nil
	}
	schema, err := types.ParseSchema(rawSchema, nil)
	if err != nil {
		return types.Schema{}, fmt.Errorf("cannot unmarshal schema of connection %d: %s", id, err)
	}
	return schema, nil
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

	var typ ConnectorType
	var connectorName string
	var role ConnectionRole
	var connector int
	var settings []byte
	err := this.myDB.QueryRow(
		"SELECT CAST(`s`.`type` AS UNSIGNED), CAST(`s`.`role` AS UNSIGNED), `s`.`connector`, `s`.`settings`, `c`.`name`\n"+
			"FROM `connections` AS `s`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
			"WHERE `s`.`id` = ? AND `s`.`workspace` = ?", id, this.workspace).Scan(
		&typ, &role, &connector, &settings, &connectorName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ConnectionNotFoundError{}
		}
		return nil, nil, err
	}
	if typ != DatabaseType {
		return nil, nil, errors.New("connection is not a database")
	}
	if role != SourceRole {
		return nil, nil, errors.New("connection is not a source")
	}
	const cRole = _connector.SourceRole

	// Execute the query.
	query, err = this.compileQuery(query, limit)
	if err != nil {
		return nil, nil, err
	}
	fh := this.newFirehose(context.Background(), id, connector, 0, types.Schema{}, typ, cRole, WebhooksPerNone)
	c, err := _connector.RegisteredDatabase(connectorName).Connect(fh.ctx, &_connector.DatabaseConfig{
		Role:     cRole,
		Settings: settings,
		Firehose: fh,
	})
	if err != nil {
		return nil, nil, err
	}
	rawColumns, rawRows, err := c.Query(query)
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

	if id <= 0 || id > maxInt32 {
		return nil, errors.New("invalid connection identifier")
	}

	var typ ConnectorType
	var role ConnectionRole
	err := this.myDB.QueryRow("SELECT CAST(`type` AS UNSIGNED), CAST(`role` AS UNSIGNED) FROM `connections` WHERE `id` = ? AND `workspace` = ?",
		id, this.workspace).Scan(&typ, &role)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ConnectionNotFoundError{}
		}
		return nil, err
	}

	cRole := _connector.Role(role)

	var connection _connector.Connection

	switch typ {
	case AppType:

		var connectorName, clientSecret, resourceCode, accessToken string
		var webhooksPer WebhooksPer
		var connector, resource int
		var settings []byte
		var expiration time.Time
		err = this.myDB.QueryRow(
			"SELECT `c`.`name`, `c`.`oauth_client_secret`, `c`.`webhooks_per` - 1, `r`.`code`, "+
				" `r`.`oauth_access_token`, `r`.`oauth_expires_in`, `s`.`connector`, `s`.`resource`, `s`.`settings`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"INNER JOIN `resources` AS `r` ON `r`.`id` = `s`.`resource`\n"+
				"WHERE `s`.`id` = ?", id).Scan(
			&connectorName, &clientSecret, &webhooksPer, &resourceCode, &accessToken, &expiration,
			&connector, &resource, &settings)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, ConnectionNotFoundError{}
			}
			return nil, err
		}

		if hasOAuth := clientSecret != ""; hasOAuth {
			accessTokenExpired := time.Now().UTC().Add(15 * time.Minute).After(expiration)
			if accessToken == "" || accessTokenExpired {
				accessToken, err = this.api.apis.refreshOAuthToken(resource)
				if err != nil {
					return nil, err
				}
			}
		}

		fh := this.newFirehose(context.Background(), id, connector, resource, types.Schema{}, typ, cRole, webhooksPer)
		connection, err = _connector.RegisteredApp(connectorName).Connect(fh.ctx, &_connector.AppConfig{
			Role:         cRole,
			Settings:     settings,
			Firehose:     fh,
			ClientSecret: clientSecret,
			Resource:     resourceCode,
			AccessToken:  accessToken,
		})

	default:

		var connectorName string
		var connector int
		var settings []byte
		err = this.myDB.QueryRow(
			"SELECT `c`.`name`, `s`.`connector`, `s`.`settings`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"WHERE `s`.`id` = ?", id).Scan(&connectorName, &connector, &settings)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, ConnectionNotFoundError{}
			}
			return nil, err
		}

		fh := this.newFirehose(context.Background(), id, connector, 0, types.Schema{}, typ, cRole, WebhooksPerNone)

		switch typ {
		case DatabaseType:
			connection, err = _connector.RegisteredDatabase(connectorName).Connect(fh.ctx, &_connector.DatabaseConfig{
				Role:     cRole,
				Settings: settings,
				Firehose: fh,
			})
		case EventStreamType:
			connection, err = _connector.RegisteredEventStream(connectorName).Connect(fh.ctx, &_connector.EventStreamConfig{
				Role:     cRole,
				Settings: settings,
				Firehose: fh,
			})
		case FileType:
			connection, err = _connector.RegisteredFile(connectorName).Connect(fh.ctx, &_connector.FileConfig{
				Role:     cRole,
				Settings: settings,
				Firehose: fh,
			})
		case MobileType:
			connection, err = _connector.RegisteredMobile(connectorName).Connect(fh.ctx, &_connector.MobileConfig{
				Role:     cRole,
				Settings: settings,
				Firehose: fh,
			})
		case ServerType:
			connection, err = _connector.RegisteredServer(connectorName).Connect(fh.ctx, &_connector.ServerConfig{
				Role:     cRole,
				Settings: settings,
				Firehose: fh,
			})
		case StorageType:
			connection, err = _connector.RegisteredStorage(connectorName).Connect(fh.ctx, &_connector.StorageConfig{
				Role:     cRole,
				Settings: settings,
				Firehose: fh,
			})
		case WebsiteType:
			connection, err = _connector.RegisteredWebsite(connectorName).Connect(fh.ctx, &_connector.WebsiteConfig{
				Role:     cRole,
				Settings: settings,
				Firehose: fh,
			})
		}

	}
	if err != nil {
		return nil, err
	}

	form, alert, err := connection.ServeUI(event, values)
	if err != nil {
		if err == ui.ErrEventNotExist {
			err = ErrUIEventNotExist
		}
		return nil, err
	}

	return marshalUIFormAlert(form, alert, role)
}

// SetFileStorage sets the storage of the file connection with identifier file.
// storage is the storage connection. The file and the storage must have the
// same role. As a special case, the current storage of the file is removed if
// the storage argument is 0.
//
// It returns a ConnectionNotFound error if the file or storage does not exist.
func (this *Connections) SetFileStorage(file, storage int) error {
	if file <= 0 || file > maxInt32 {
		return errors.New("invalid file connection identifier")
	}
	if storage < 0 || storage > maxInt32 {
		return errors.New("invalid storage connection identifier")
	}
	if file == storage {
		return errors.New("file and storage cannot be the same connection")
	}
	if storage == 0 {
		result, err := this.myDB.Exec("UPDATE `connections` SET `storage` = 0 WHERE `id` = ?", file)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return ConnectionNotFoundError{FileType}
		}
		return nil
	}
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		var fileRole, storageRole ConnectionRole
		err := tx.QueryRow("SELECT\n"+
			"\t(SELECT IFNULL(`role`, 0) FROM `connections` WHERE `id` = ?),\n"+
			"\t(SELECT IFNULL(`role`, 0) FROM `connections` WHERE `id` = ?)", file, storage).Scan(&fileRole, &storageRole)
		if err != nil {
			return err
		}
		if fileRole == 0 {
			return ConnectionNotFoundError{FileType}
		}
		if storageRole == 0 {
			return ConnectionNotFoundError{StorageType}
		}
		if fileRole != storageRole {
			if fileRole == SourceRole {
				return errors.New("storage connection is not a source")
			}
			return errors.New("storage connection is not a destination")
		}
		_, err = tx.Exec("UPDATE `connections` SET `storage` = ? WHERE `id` = ?", storage, file)
		return err
	})
	return err
}

// SetUsersQuery sets the users query of the database connection with
// identifier id. query must be UTF-8 encoded, it cannot be longer than
// 16,777,215 runes and must contain the ':limit' placeholder.
//
// It returns an error if the connection is a destination.
// It returns a ConnectionNotFoundError error if the connection does not exist.
func (this *Connections) SetUsersQuery(id int, query string) error {

	if id <= 0 || id > maxInt32 {
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

	result, err := this.myDB.Exec("UPDATE `connections`\nSET `users_query` = ?\n"+
		"WHERE `id` = ? AND `workspace` = ? AND `type` = 'Database' AND `role` = 'Source'",
		query, id, this.workspace)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		var typ ConnectorType
		var role ConnectionRole
		err = this.myDB.QueryRow("SELECT CAST(`type` AS UNSIGNED), CAST(`role` AS UNSIGNED) FROM `connections` WHERE `id` = ? AND `workspace` = ?",
			id, this.workspace).Scan(&typ, &role)
		if err != nil {
			return err
		}
		if typ != DatabaseType {
			return errors.New("connection is not a database")
		}
		if role != SourceRole {
			return errors.New("connection is not a source")
		}
		return ConnectionNotFoundError{DatabaseType}
	}

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
// 24 hours.
func (this *Connections) Stats(id int) (*ConnectionsStats, error) {
	if id <= 0 || id > maxInt32 {
		return nil, errors.New("invalid connection identifier")
	}
	now := time.Now().UTC()
	toSlot := statsTimeSlot(now)
	fromSlot := toSlot - 23
	stats := &ConnectionsStats{
		UsersIn: [24]int{},
	}
	query := "SELECT `time_slot`, `users_in`\nFROM `connections_stats`\nWHERE `connection` = ? AND `time_slot` BETWEEN ? AND ?"
	err := this.myDB.QueryScan(query, id, fromSlot, toSlot, func(rows *sql.Rows) error {
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
func (this *Connections) newFirehose(ctx context.Context, connection, connector, resource int, userSchema types.Schema, typ ConnectorType, role _connector.Role, webhooksPer WebhooksPer) *firehose {
	fh := &firehose{
		connections:   this,
		connection:    connection,
		resource:      resource,
		connector:     connector,
		connectorType: typ,
		role:          role,
		webhooksPer:   webhooksPer,
		userSchema:    userSchema,
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

	if id <= 0 || id > maxInt32 {
		return errors.New("invalid connection identifier")
	}

	var typ ConnectorType
	var role ConnectionRole
	var storage int
	err := this.myDB.QueryRow("SELECT CAST(`type` AS UNSIGNED), CAST(`role` AS UNSIGNED), `storage`\n"+
		"FROM `connections`\n"+
		"WHERE `id` = ? AND `workspace` = ?", id, this.workspace).Scan(&typ, &role, &storage)
	if err != nil {
		if err == sql.ErrNoRows {
			return ConnectionNotFoundError{}
		}
		return err
	}
	switch typ {
	case AppType, DatabaseType:
	case FileType:
		if storage == 0 {
			return ErrFileHasNoStorage
		}
	default:
		return fmt.Errorf("cannot import properties from a %s connection", strings.ToLower(typ.String()))
	}
	if role == DestinationRole {
		return errors.New("cannot import from a destination")
	}

	cRole := _connector.Role(role)

	var schema types.Schema

	switch typ {
	case AppType:

		// TODO(marco) The following code is duplicated in the Import method.
		var connectorName, clientSecret, resourceCode, accessToken, refreshToken, cursor string
		var webhooksPer WebhooksPer
		var connector, resource int
		var settings []byte
		var expiration *time.Time
		err = this.myDB.QueryRow(
			"SELECT `c`.`name`, `c`.`oauth_client_secret`, `c`.`webhooks_per` - 1, IFNULL(`r`.`code`, ''), "+
				" IFNULL(`r`.`oauth_access_token`, ''), IFNULL(`r`.`oauth_refresh_token`, ''), `r`.`oauth_expires_in`, "+
				" `s`.`connector`, `s`.`resource`, `s`.`user_cursor`, `s`.`settings`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"LEFT JOIN `resources` AS `r` ON `r`.`id` = `s`.`resource`\n"+
				"WHERE `s`.`id` = ?", id).Scan(
			&connectorName, &clientSecret, &webhooksPer, &resourceCode, &accessToken, &refreshToken, &expiration,
			&connector, &resource, &cursor, &settings)
		if err != nil {
			if err == sql.ErrNoRows {
				return ConnectionNotFoundError{}
			}
			return err
		}

		if hasOAuth := clientSecret != ""; hasOAuth {
			accessTokenExpired := time.Now().UTC().Add(15 * time.Minute).After(*expiration)
			if accessToken == "" || accessTokenExpired {
				accessToken, err = this.api.apis.refreshOAuthToken(resource)
				if err != nil {
					return err
				}
			}
		}

		fh := this.newFirehose(context.Background(), id, connector, resource, types.Schema{}, AppType, cRole, webhooksPer)
		c, err := _connector.RegisteredApp(connectorName).Connect(fh.ctx, &_connector.AppConfig{
			Role:         cRole,
			Settings:     settings,
			Firehose:     fh,
			ClientSecret: clientSecret,
			Resource:     resourceCode,
			AccessToken:  accessToken,
		})
		if err != nil {
			return err
		}
		schema, _, err = c.Schemas()
		if err != nil {
			return err
		}
		if !schema.Valid() {
			return fmt.Errorf("connection %d returned an invalid schema", id)
		}
		schema = schema.AsRole(types.Role(role))
		if !schema.Valid() {
			return errors.New("connection has returned a schema without source properties")
		}

	case DatabaseType:

		var connectorName, usersQuery string
		var connector int
		var settings []byte
		err = this.myDB.QueryRow(
			"SELECT `c`.`name`, `s`.`connector`, `s`.`settings`, `s`.`users_query`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"WHERE `s`.`id` = ?", id).Scan(&connectorName, &connector, &settings, &usersQuery)
		if err != nil {
			if err == sql.ErrNoRows {
				return ConnectionNotFoundError{}
			}
			return err
		}

		usersQuery, err := this.compileQuery(usersQuery, 0)
		if err != nil {
			return err
		}
		fh := this.newFirehose(context.Background(), id, connector, 0, types.Schema{}, DatabaseType, cRole, WebhooksPerNone)
		c, err := _connector.RegisteredDatabase(connectorName).Connect(fh.ctx, &_connector.DatabaseConfig{
			Role:     cRole,
			Settings: settings,
			Firehose: fh,
		})
		if err != nil {
			return err
		}
		columns, rows, err := c.Query(usersQuery)
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

		var connectorName, identityColumn, timestampColumn string
		var connector, storage int
		var settings []byte
		err = this.myDB.QueryRow(
			"SELECT `c`.`name`, `s`.`connector`, `s`.`storage`, `s`.`identity_column`, `s`.`timestamp_column`, `s`.`settings`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"WHERE `s`.`id` = ?", id).Scan(&connectorName, &connector, &storage, &identityColumn, &timestampColumn, &settings)
		if err != nil {
			if err == sql.ErrNoRows {
				return ConnectionNotFoundError{}
			}
			return err
		}
		if storage == 0 {
			return ErrFileHasNoStorage
		}

		var ctx = context.Background()

		// Get the file reader.
		var files *fileReader
		{
			var connectorName string
			var connector int
			var settings []byte
			err = this.myDB.QueryRow(
				"SELECT `c`.`name`, `s`.`connector`, `s`.`settings`\n"+
					"FROM `connections` AS `s`\n"+
					"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
					"WHERE `s`.`id` = ?", storage).Scan(&connectorName, &connector, &settings)
			if err != nil {
				if err != sql.ErrNoRows {
					return ErrFileHasNoStorage
				}
				return err
			}
			fh := this.newFirehose(ctx, storage, connector, 0, types.Schema{}, StorageType, cRole, WebhooksPerNone)
			ctx = fh.ctx
			c, err := _connector.RegisteredStorage(connectorName).Connect(ctx, &_connector.StorageConfig{
				Role:     cRole,
				Settings: settings,
				Firehose: fh,
			})
			if err != nil {
				return err
			}
			files = newFileReader(c)
		}

		// Connect to the file connector and read only the columns.
		fh := this.newFirehose(ctx, id, connector, 0, types.Schema{}, FileType, cRole, WebhooksPerNone)
		file, err := _connector.RegisteredFile(connectorName).Connect(fh.ctx, &_connector.FileConfig{
			Role:     cRole,
			Settings: settings,
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
	_, err = this.myDB.Exec("UPDATE `connections` SET `schema` = ? WHERE `id` = ?", rawSchema, id)

	return err
}

// userSchema returns the user schema and the paths of the mapped properties of
// the connection with identifier id.
//
// If the connection does not exist it returns a ConnectionNotFoundError error.
func (this *Connections) userSchema(id int) (types.Schema, []_connector.PropertyPath, error) {

	// Read the schema.
	var rawSchema string
	err := this.myDB.QueryRow("SELECT `schema` FROM `connections` WHERE `workspace` = ? AND `id` = ?",
		this.workspace, id).Scan(&rawSchema)
	if err != nil {
		if err == sql.ErrNoRows {
			return types.Schema{}, nil, ConnectionNotFoundError{}
		}
		return types.Schema{}, nil, err
	}
	schema, err := types.ParseSchema(rawSchema, nil)
	if err != nil {
		return types.Schema{}, nil, fmt.Errorf("cannot unmarshal schema of connection %d", id)
	}

	// Read the paths of the mapped properties from the transformations of this connection.
	var paths []_connector.PropertyPath
	err = this.myDB.QueryScan(
		"SELECT `property` FROM `transformations_connections` WHERE `connection` = ?", id, func(rows *sql.Rows) error {
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
		if err == sql.ErrNoRows {
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
	for _, property := range schema.Properties() {
		if _, ok := mapped[property.Name]; ok {
			mappedProperties = append(mappedProperties, property)
		}
	}
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
		comma := false
		b.WriteString(`"Form":{"Fields":[`)
		for _, field := range form.Fields {
			ok, err := marshalUIComponent(&b, field, role, comma)
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

// marshalUIComponent marshals component with the given role in JSON format.
// If comma is true, it prepends a comma. Returns whether it has been marhalled
func marshalUIComponent(b *bytes.Buffer, component ui.Component, role ConnectionRole, comma bool) (bool, error) {
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
		b.WriteString(`,"`)
		b.WriteString(name)
		b.WriteString(`":`)
		field := rv.Field(j).Interface()
		var err error
		if c, ok := field.(ui.Component); ok {
			_, err = marshalUIComponent(b, c, role, false)
		} else {
			err = json.NewEncoder(b).Encode(field)
		}
		if err != nil {
			return false, err
		}
	}
	b.WriteString(`}`)
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
