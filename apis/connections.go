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
	maxInt32             = math.MaxInt32
	rawPropertiesMaxSize = 16_777_215 // maximum size in runes of the 'property' column of the 'connections' table.
	queryMaxSize         = 16_777_215 // maximum size in runes of a connection query.
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
}

// ConnectionInfo represents a connection.
type ConnectionInfo struct {
	ID         int
	Type       ConnectorType
	Role       ConnectionRole
	Storage    int // zero if the connection is not a file or does not have a storage
	Name       string
	LogoURL    string
	UsersQuery string // only for databases.
}

// AddApp adds an app connection given its role, app connector, OAuth refresh
// and access tokens and returns its identifier.
//
// If the connector does not exist, it returns a ConnectorNotFoundError error.
func (this *Connections) AddApp(role ConnectionRole, connector int, refreshToken, accessToken, expiresIn string) (int, error) {
	if role != SourceRole && role != DestinationRole {
		return 0, errors.New("invalid role")
	}
	if connector <= 0 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
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
	c, err := newAppConnection(context.Background(), conn.Name, &_connector.AppConfig{
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
		err := tx.QueryRow("SELECT `id`, `oAuthRefreshToken` FROM `resources` WHERE `connector` = ? AND `code` = ?",
			connector, resourceCode).Scan(&resource, &currentRefreshToken)
		if err != nil {
			if err != sql.ErrNoRows {
				return err
			}
			err = nil
		}
		if resource == 0 {
			result, err := tx.Exec("INSERT INTO `resources` "+
				"SET `connector` = ?, `code` = ?, `oAuthAccessToken` = ?, `oAuthRefreshToken` = ?, `oAuthExpiresIn` = ?",
				connector, resourceCode, accessToken, refreshToken, expiresIn)
			if err != nil {
				return err
			}
			resourceID, err := result.LastInsertId()
			resource = int(resourceID)
		} else if refreshToken != currentRefreshToken {
			_, err = tx.Exec("UPDATE `resources` "+
				"SET `oAuthAccessToken` = ?, `oAuthRefreshToken` = ?, `oAuthExpiresIn` = ? WHERE `id` = ?",
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
			"SET `id` = ?, `workspace` = ?, `type` = 'App', `role` = ?, `connector` = ?, `resource` = ?",
			id, this.workspace, role, connector, resource)
		return err
	})
	if err != nil {
		return 0, err
	}

	go func() {
		err := this.reloadProperties(id)
		if err != nil {
			log.Printf("[error] cannot reload properties for connection %d: %s", id, err)
		}
	}()

	return id, err
}

// AddDatabase adds a database connection given its role, database connector
// and returns its identifier.
//
// If the connector does not exist, it returns a ConnectorNotFoundError error.
func (this *Connections) AddDatabase(role ConnectionRole, connector int) (int, error) {
	if role != SourceRole && role != DestinationRole {
		return 0, errors.New("invalid role")
	}
	if connector <= 0 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
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
			"SET `id` = ?, `workspace` = ?, `type` = 'Database', `role` = ?, `connector` = ?",
			id, this.workspace, role, connector)
		return err
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

// AddFile adds a file connection given its role, connector and storage
// connection and returns its identifier. If storage is 0, the file connection
// does not have a storage, otherwise storage must have the given role.
//
// If the connector does not exist, it returns a ConnectorNotFoundError error.
// If the storage does not exist, it returns a ConnectionNotFound error.
func (this *Connections) AddFile(role ConnectionRole, connector, storage int) (int, error) {
	if role != SourceRole && role != DestinationRole {
		return 0, errors.New("invalid role")
	}
	if connector <= 0 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
	}
	if storage < 0 || storage > maxInt32 {
		return 0, errors.New("invalid storage")
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
			"SET `id` = ?, `workspace` = ?, `type` = 'File', `role` = ?, `connector` = ?, `storage` = ?",
			id, this.workspace, role, connector, storage)
		return err
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

// AddServer adds a server connection given its role and server connector.
//
// If the connector does not exist, it returns a ConnectorNotFoundError error.
func (this *Connections) AddServer(role ConnectionRole, connector int) (int, error) {
	if role != SourceRole && role != DestinationRole {
		return 0, errors.New("invalid role")
	}
	if connector <= 0 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
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
			return errors.New("connector is not an server connector")
		}
		id, err = generateConnectionID()
		if err != nil {
			return err
		}
		_, err = tx.Exec("INSERT INTO `connections`\n"+
			"SET `id` = ?, `workspace` = ?, `type` = 'Server', `role` = ?, `connector`  = ?",
			id, this.workspace, role, connector)
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

// AddMobile adds a mobile connection given its role and mobile connector.
//
// If the connector does not exist, it returns a ConnectorNotFoundError error.
func (this *Connections) AddMobile(role ConnectionRole, connector int) (int, error) {
	if role != SourceRole && role != DestinationRole {
		return 0, errors.New("invalid role")
	}
	if connector <= 0 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
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
			"SET `id` = ?, `workspace` = ?, `type` = 'Server', `role` = ?, `connector` = ?",
			id, this.workspace, role, connector)
		return err
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

// AddStorage adds a storage connection given its role and connector and
// returns its identifier.
//
// If the connector does not exist, it returns a ConnectorNotFoundError error.
func (this *Connections) AddStorage(role ConnectionRole, connector int) (int, error) {
	if role != SourceRole && role != DestinationRole {
		return 0, errors.New("invalid role")
	}
	if connector <= 0 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
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
			"SET `id` = ?, `workspace` = ?, `type` = 'Storage', `role` = ?, `connector` = ?",
			id, this.workspace, role, connector)
		return err
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

// AddWebsite adds a website connection given its role, website connector and
// website host and returns its identifier. host may be of the form
// "host:port".
//
// If the connector does not exist, it returns a ConnectorNotFoundError error.
func (this *Connections) AddWebsite(role ConnectionRole, connector int, host string) (int, error) {
	if role != SourceRole && role != DestinationRole {
		return 0, errors.New("invalid role")
	}
	if connector <= 0 || connector > maxInt32 {
		return 0, errors.New("invalid connector")
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
			"SET `id` = ?, `workspace` = ?, `type` = 'Website', `role` = ?, `connector` = ?, `websiteHost` = ?",
			id, this.workspace, role, connector, host)
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
		"SELECT CAST(`s`.`type` AS UNSIGNED), CAST(`s`.`role` AS UNSIGNED), `c`.`name`, `c`.`logoURL`, `s`.`usersQuery`\n"+
			"FROM `connections` AS `s`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
			"WHERE `s`.`id` = ? AND `s`.`workspace` = ?",
		id, this.workspace).Scan(&s.Type, &s.Role, &s.Name, &s.LogoURL, &s.UsersQuery)
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
			sql.Columns{"CAST(`type` AS UNSIGNED)", "resource"})
		if err != nil {
			return err
		}
		if connection == nil {
			return nil
		}
		if connection["type"] == StorageType {
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
	case AppType, DatabaseType:
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
	transformations, err := this.Transformations.List(id)
	if err != nil {
		return fmt.Errorf("cannot list transformations for %d: %s", id, err)
	}
	if len(transformations) == 0 {
		return ErrConnectionDisabled
	}

	// Track the import in the database.
	importID, err := this.myDB.Table("ConnectionsImports").Add(sql.Set{
		"connection": id,
		"storage":    storage,
		"startTime":  time.Now().UTC(),
	}, nil)
	if err != nil {
		return err
	}

	// Start the import.
	go func() {
		err = this.startImport(id, typ, reimport)
		var errorMsg string
		if err != nil {
			errorMsg = abbreviate(err.Error(), 1000)
		}
		_, err2 := this.myDB.Table("ConnectionsImports").Update(
			sql.Set{"endTime": time.Now().UTC(), "error": errorMsg},
			sql.Where{"id": importID})
		if err2 != nil {
			log.Printf("[error] cannot update the end of import %d into the database: %s", importID, err2)
		}
	}()

	return nil
}

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
			"SELECT `c`.`name`, `c`.`oAuthClientSecret`, `c`.`webhooksPer` - 1, `r`.`code`,"+
				" `r`.`oAuthAccessToken`, `r`.`oAuthRefreshToken`, `r`.`oAuthExpiresIn`, `s`.`connector`,"+
				" `s`.`resource`, `s`.`userCursor`, `s`.`settings`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"INNER JOIN `resources` AS `r` ON `r`.`id` = `s`.`resource`\n"+
				"WHERE `s`.`id` = ?", id).Scan(
			&name, &clientSecret, &webhooksPer, &resourceCode, &accessToken, &refreshToken, &expiration, &connector,
			&resource, &cursor, &settings)
		if err != nil {
			if err == sql.ErrNoRows {
				return errors.New("connection does not exist anymore")
			}
			return err
		}
		if reimport {
			cursor = ""
		}

		// Read the used properties from the transformations of this connection.
		var properties [][]string
		rows, err := this.myDB.Query(
			"SELECT `property` FROM `transformations_connections`\n"+
				"WHERE `connection` = ?", id)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var prop string
			err = rows.Scan(&prop)
			if err != nil {
				return err
			}
			properties = append(properties, []string{prop})
		}
		if err := rows.Err(); err != nil {
			return err
		}

		accessTokenExpired := time.Now().UTC().Add(15 * time.Minute).After(expiration)

		if accessToken == "" || accessTokenExpired {
			accessToken, err = this.api.apis.refreshOAuthToken(resource)
			if err != nil {
				return err
			}
		}

		fh := this.newFirehose(context.Background(), id, connector, resource, typ, role, webhooksPer)
		c, err := newAppConnection(fh.ctx, name, &_connector.AppConfig{
			Role:         role,
			Settings:     settings,
			Firehose:     fh,
			ClientSecret: clientSecret,
			Resource:     resourceCode,
			AccessToken:  accessToken,
		})
		if err != nil {
			return fmt.Errorf("cannot connect to the connector %d of the connection %d: %s", connector, id, err)
		}
		c.Users(cursor, properties)
		if err != nil {
			return fmt.Errorf("call to the Users method of the connection %d failed: %s", id, err)
		}

	case DatabaseType:

		var connectorName, identityColumn, timestampColumn, usersQuery string
		var connector int
		var settings []byte
		err := this.myDB.QueryRow(
			"SELECT `c`.`name`, `s`.`connector`, `s`.`identityColumn`, `s`.`timestampColumn`, `s`.`settings`, `s`.`usersQuery`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"WHERE `s`.`id` = ?", id).Scan(&connectorName, &connector, &identityColumn, &timestampColumn, &settings, &usersQuery)
		if err != nil {
			if err == sql.ErrNoRows {
				return errors.New("connection does not exist anymore")
			}
			return err
		}

		usersQuery, err = this.compileQueryWithoutLimit(usersQuery)
		if err != nil {
			return err
		}
		fh := this.newFirehose(context.Background(), id, connector, 0, DatabaseType, role, WebhooksPerNone)
		c, err := newDatabaseConnection(fh.ctx, connectorName, &_connector.DatabaseConfig{
			Role:     role,
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
			return fmt.Errorf("missing identity column %q", identityColumn)
		}
		if timestampColumn != "" && timestampIndex == noColumn {
			return fmt.Errorf("missing timestamp column %q", timestampColumn)
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
				return err
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
			fh.SetUser(*identity, ts, user)
		}
		if err = rows.Err(); err != nil {
			return err
		}

	case FileType:

		var connectorName, identityColumn, timestampColumn string
		var connector, storage int
		var settings []byte
		err := this.myDB.QueryRow(
			"SELECT `c`.`name`, `s`.`connector`, `s`.`storage`, `s`.`identityColumn`, `s`.`timestampColumn`, `s`.`settings`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"WHERE `s`.`id` = ?", id).Scan(&connectorName, &connector, &storage, &identityColumn, &timestampColumn, &settings)
		if err != nil {
			if err == sql.ErrNoRows {
				return errors.New("connection does not exist anymore")
			}
			return err
		}
		if storage == 0 {
			return errors.New("connector has no more a storage")
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
					return errors.New("connector has no more a storage")
				}
				return err
			}
			fh := this.newFirehose(ctx, storage, connector, 0, StorageType, role, WebhooksPerNone)
			ctx = fh.ctx
			c, err := newStorageConnection(ctx, connectorName, &_connector.StorageConfig{
				Role:     role,
				Settings: settings,
				Firehose: fh,
			})
			if err != nil {
				return err
			}
			files = newFileReader(c)
		}

		// Connect to the file connector.
		fh := this.newFirehose(ctx, id, connector, 0, FileType, role, WebhooksPerNone)
		file, err := newFileConnection(fh.ctx, connectorName, &_connector.FileConfig{
			Role:     role,
			Settings: settings,
			Firehose: fh,
		})
		if err != nil {
			return err
		}

		// Read the records.
		records := fh.newRecordWriter(identityColumn, timestampColumn, false)
		err = file.Read(files, records)
		if err != nil {
			return err
		}

	}

	return nil
}

// List returns all connections.
func (this *Connections) List() ([]*Connection, error) {
	sources := []*Connection{}
	err := this.myDB.QueryScan(
		"SELECT `s`.`id`, CAST(`s`.`type` AS UNSIGNED), CAST(`s`.`role` AS UNSIGNED), `s`.`storage`, `c`.`name`, `c`.`oAuthURL`, `c`.`logoURL`\n"+
			"FROM `connections` as `s`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
			"WHERE `s`.`workspace` = ?", this.workspace, func(rows *sql.Rows) error {
			var err error
			for rows.Next() {
				var c Connection
				if err = rows.Scan(&c.ID, &c.Type, &c.Role, &c.Storage, &c.Name,
					&c.OAuthURL, &c.LogoURL); err != nil {
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

// Properties returns the properties and the used properties of the connection
// with the given identifier. The connection must be an app, database of file
// connection.
//
// Returns a ConnectionNotFoundError error if the connection does not exist.
func (this *Connections) Properties(id int) ([]types.Property, [][]string, error) {
	if id <= 0 || id > maxInt32 {
		return nil, nil, errors.New("invalid connection identifier")
	}
	var typ ConnectorType
	var rawProperties, rawUsedProperties []byte
	err := this.myDB.QueryRow("SELECT CAST(`type` AS UNSIGNED), `properties`, `usedProperties`\n"+
		"FROM `connections`\n"+
		"WHERE `id` = ? AND `workspace` = ?", id, this.workspace).Scan(&typ, &rawProperties, &rawUsedProperties)
	if err != nil {
		return nil, nil, err
	}
	if typ == StorageType {
		return nil, nil, errors.New("cannot read properties from a storage")
	}
	var properties []types.Property
	if len(rawProperties) > 0 {
		err = json.Unmarshal(rawProperties, &properties)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot unmarshal connection properties: %s", err)
		}
	} else {
		properties = []types.Property{}
	}
	var usedProperties [][]string
	if len(rawUsedProperties) > 0 {
		err = json.Unmarshal(rawUsedProperties, &usedProperties)
		if err != nil {
			return nil, nil, errors.New("cannot unmarshal connection used properties")
		}
	} else {
		usedProperties = [][]string{}
	}

	return properties, usedProperties, nil
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
	query, err = this.compileQueryWithLimit(query, limit)
	if err != nil {
		return nil, nil, err
	}
	fh := this.newFirehose(context.Background(), id, connector, 0, typ, cRole, WebhooksPerNone)
	c, err := newDatabaseConnection(fh.ctx, connectorName, &_connector.DatabaseConfig{
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
			"SELECT `c`.`name`, `c`.`oAuthClientSecret`, `c`.`webhooksPer` - 1, `r`.`code`, "+
				" `r`.`oAuthAccessToken`, `r`.`oAuthExpiresIn`, `s`.`connector`, `s`.`resource`, `s`.`settings`\n"+
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

		accessTokenExpired := time.Now().UTC().Add(15 * time.Minute).After(expiration)

		if accessToken == "" || accessTokenExpired {
			accessToken, err = this.api.apis.refreshOAuthToken(resource)
			if err != nil {
				return nil, err
			}
		}

		fh := this.newFirehose(context.Background(), id, connector, resource, typ, cRole, webhooksPer)
		connection, err = newAppConnection(fh.ctx, connectorName, &_connector.AppConfig{
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

		fh := this.newFirehose(context.Background(), id, connector, 0, typ, cRole, WebhooksPerNone)

		switch typ {
		case DatabaseType:
			connection, err = newDatabaseConnection(fh.ctx, connectorName, &_connector.DatabaseConfig{
				Role:     cRole,
				Settings: settings,
				Firehose: fh,
			})
		case FileType:
			connection, err = newFileConnection(fh.ctx, connectorName, &_connector.FileConfig{
				Role:     cRole,
				Settings: settings,
				Firehose: fh,
			})
		case MobileType:
			connection, err = newMobileConnection(fh.ctx, connectorName, &_connector.MobileConfig{
				Role:     cRole,
				Settings: settings,
				Firehose: fh,
			})
		case ServerType:
			connection, err = newServerConnection(fh.ctx, connectorName, &_connector.ServerConfig{
				Role:     cRole,
				Settings: settings,
				Firehose: fh,
			})
		case StorageType:
			connection, err = newStorageConnection(fh.ctx, connectorName, &_connector.StorageConfig{
				Role:     cRole,
				Settings: settings,
				Firehose: fh,
			})
		case WebsiteType:
			connection, err = newWebsiteConnection(fh.ctx, connectorName, &_connector.WebsiteConfig{
				Role:     cRole,
				Settings: settings,
				Firehose: fh,
			})
		}

	}
	if err != nil {
		return nil, err
	}

	form, err := connection.ServeUI(event, values)
	if err != nil {
		if err == ui.ErrEventNotExist {
			err = ErrUIEventNotExist
		}
		return nil, err
	}

	return marshalUIForm(form, role)
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

	result, err := this.myDB.Exec("UPDATE `connections`\nSET `usersQuery` = ?\n"+
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
	query := "SELECT `timeSlot`, `usersIn`\nFROM `connections_stats`\nWHERE `connection` = ? AND `timeSlot` BETWEEN ? AND ?"
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
func (this *Connections) newFirehose(ctx context.Context, connection, connector, resource int, typ ConnectorType, role _connector.Role, webhooksPer WebhooksPer) *firehose {
	fh := &firehose{
		connections:   this,
		connection:    connection,
		resource:      resource,
		connector:     connector,
		connectorType: typ,
		role:          role,
		webhooksPer:   webhooksPer,
	}
	fh.ctx, fh.cancel = context.WithCancel(ctx)
	return fh
}

var errRecordStop = errors.New("stop record")

// reloadProperties reloads the properties of the connection with identifier id.
// The connection must be a source app, database or file.
//
// If the connection does not exist it returns a ConnectionNotFoundError error.
func (this *Connections) reloadProperties(id int) error {

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

	var properties []types.Property

	switch typ {
	case AppType:

		// TODO(marco) The following code is duplicated in the Import method.
		var connectorName, clientSecret, resourceCode, accessToken, refreshToken, cursor string
		var webhooksPer WebhooksPer
		var connector, resource int
		var settings []byte
		var expiration *time.Time
		err = this.myDB.QueryRow(
			"SELECT `c`.`name`, `c`.`oAuthClientSecret`, `c`.`webhooksPer` - 1, IFNULL(`r`.`code`, ''), "+
				" IFNULL(`r`.`oAuthAccessToken`, ''), IFNULL(`r`.`oAuthRefreshToken`, ''), `r`.`oAuthExpiresIn`, "+
				" `s`.`connector`, `s`.`resource`, `s`.`userCursor`, `s`.`settings`\n"+
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

		accessTokenExpired := time.Now().UTC().Add(15 * time.Minute).After(*expiration)

		if accessToken == "" || accessTokenExpired {
			accessToken, err = this.api.apis.refreshOAuthToken(resource)
			if err != nil {
				return err
			}
		}
		fh := this.newFirehose(context.Background(), id, connector, resource, AppType, cRole, webhooksPer)
		c, err := newAppConnection(fh.ctx, connectorName, &_connector.AppConfig{
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
		properties, _, err = c.Properties()
		if err != nil {
			return err
		}

	case DatabaseType:

		var connectorName, usersQuery string
		var connector int
		var settings []byte
		err = this.myDB.QueryRow(
			"SELECT `c`.`name`, `s`.`connector`, `s`.`settings`, `s`.`usersQuery`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"WHERE `s`.`id` = ?", id).Scan(&connectorName, &connector, &settings, &usersQuery)
		if err != nil {
			if err == sql.ErrNoRows {
				return ConnectionNotFoundError{}
			}
			return err
		}

		usersQuery, err := this.compileQueryWithLimit(usersQuery, 0)
		if err != nil {
			return err
		}
		fh := this.newFirehose(context.Background(), id, connector, 0, DatabaseType, cRole, WebhooksPerNone)
		c, err := newDatabaseConnection(fh.ctx, connectorName, &_connector.DatabaseConfig{
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
		properties = make([]types.Property, len(columns))
		for i := 0; i < len(properties); i++ {
			properties[i].Name = columns[i].Name
			properties[i].Type = columns[i].Type
		}

	case FileType:

		var connectorName, identityColumn, timestampColumn string
		var connector, storage int
		var settings []byte
		err = this.myDB.QueryRow(
			"SELECT `c`.`name`, `s`.`connector`, `s`.`storage`, `s`.`identityColumn`, `s`.`timestampColumn`, `s`.`settings`\n"+
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
			fh := this.newFirehose(ctx, storage, connector, 0, StorageType, cRole, WebhooksPerNone)
			ctx = fh.ctx
			c, err := newStorageConnection(ctx, connectorName, &_connector.StorageConfig{
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
		fh := this.newFirehose(ctx, id, connector, 0, FileType, cRole, WebhooksPerNone)
		file, err := newFileConnection(fh.ctx, connectorName, &_connector.FileConfig{
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

		properties = make([]types.Property, len(records.columns))
		for i := 0; i < len(properties); i++ {
			properties[i].Name = records.columns[i].Name
			properties[i].Type = records.columns[i].Type
		}
	}

	rawProperties, err := json.Marshal(properties)
	if err != nil {
		return fmt.Errorf("cannot marshal the properties of the connection %d : %s", id, err)
	}
	if utf8.RuneCount(rawProperties) > rawPropertiesMaxSize {
		return fmt.Errorf("cannot marshal the properties of the connection %d: data is too large", id)
	}

	_, err = this.myDB.Exec("UPDATE `connections` SET `properties` = ? WHERE `id` = ?", rawProperties, id)

	return err
}

// compileQueryWithLimit compiles the given query, replacing the ':limit'
// placeholder with limit, and returns it.
func (this *Connections) compileQueryWithLimit(query string, limit int) (string, error) {
	return strings.ReplaceAll(query, ":limit", strconv.Itoa(limit)), nil
}

// compileQueryWithoutLimit compiles the given query, removing the ':limit'
// placeholder, and returns it.
func (this *Connections) compileQueryWithoutLimit(query string) (string, error) {
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
	return query[:s1] + query[s2:], nil
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

// marshalUIForm marshals form with given role in JSON format.
func marshalUIForm(form *ui.Form, role ConnectionRole) ([]byte, error) {

	if form == nil {
		return []byte("null"), nil
	}

	var b bytes.Buffer
	dec := json.NewEncoder(&b)

	b.WriteString(`{"Fields":[`)

	for i, field := range form.Fields {
		rv := reflect.ValueOf(field).Elem()
		rt := rv.Type()
		if r := ui.Role(rv.FieldByName("Role").Int()); r != ui.BothRole && ConnectionRole(r) != role {
			continue
		}
		if i > 0 {
			b.WriteString(`,`)
		}
		b.WriteString(`{"ComponentType":"`)
		b.WriteString(rt.Name())
		b.WriteString(`"`)
		for j := 0; j < rt.NumField(); j++ {
			if rt.Field(j).Name == "Destination" {
				continue
			}
			b.WriteString(`,"`)
			b.WriteString(rt.Field(j).Name)
			b.WriteString(`":`)
			err := dec.Encode(rv.Field(j).Interface())
			if err != nil {
				return nil, err
			}
		}
		b.WriteString(`}`)
	}

	b.WriteString(`],"Actions":`)
	err := dec.Encode(form.Actions)
	if err != nil {
		return nil, err
	}
	b.WriteString(`}`)

	return b.Bytes(), nil
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
