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
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/types"
	_connector "chichi/connector"
	"chichi/connector/ui"
	"chichi/pkg/open2b/sql"
)

type Connections struct {
	*WorkspaceAPI
}

var (
	ErrConnectorNotFound        = errors.New("connector does not exist")
	ErrConnectionNotFound       = errors.New("connection does not exist")
	ErrConnectionDisabled       = errors.New("connection is disabled")
	ErrFileHasNoStorage         = errors.New("file connection has not a storage")
	ErrStorageNotFound          = errors.New("storage does not exist")
	ErrStorageHasConnectedFiles = errors.New("storage has connected files")
	ErrUIEventNotExist          = errors.New("UI event does not exist")
)

const (
	rawPropertiesMaxSize = 16_777_215 // maximum size in runes of the 'property' column of the 'connections' table.
	queryMaxSize         = 16_777_215 // maximum size in runes of a connection query.
)

// A DatabaseQueryError error is returned from a database connector if an error
// occurs when executing a query.
type DatabaseQueryError struct {
	Message string
}

func (err DatabaseQueryError) Error() string {
	return err.Message
}

// Direction represents a connection direction.
type Direction int

const (
	SourceDir Direction = iota + 1 // source
	DestDir                        // destination
)

// String returns the string representation of dir.
// It panics if dir is not a valid Direction value.
func (dir Direction) String() string {
	switch dir {
	case SourceDir:
		return "Source"
	case DestDir:
		return "Destination"
	}
	panic("invalid direction")
}

// Connection represents a connection.
type Connection struct {
	ID        int
	Name      string
	Type      string
	Direction Direction
	Storage   int // zero if the connection is not a file or does not have a storage
	OauthURL  string
	LogoURL   string
}

// ConnectionInfo represents a connection.
type ConnectionInfo struct {
	ID         int
	Type       string
	Direction  Direction
	Storage    int // zero if the connection is not a file or does not have a storage
	Name       string
	LogoURL    string
	UsersQuery string // only for databases.
}

// PropertyType represents the type of a property.
type PropertyType string

// ConnectionPropertyOption represents an option of a connection property.
type ConnectionPropertyOption struct {
	Label string
	Value string
}

// ConnectionProperty represents a connection property.
type ConnectionProperty struct {
	Name       string
	Type       PropertyType
	Label      string
	Options    []ConnectionPropertyOption
	Properties []ConnectionProperty
}

// AddApp adds an app connection given its direction, app connector, OAuth
// refresh and access tokens and returns its identifier.
//
// If the connector does not exist, it returns the ErrConnectorNotFound error.
func (this *Connections) AddApp(dir Direction, connector int, refreshToken, accessToken, accessTokenExpirationTime string) (int, error) {
	if dir != SourceDir && dir != DestDir {
		return 0, errors.New("invalid direction")
	}
	if connector <= 0 {
		return 0, errors.New("invalid connector")
	}
	conn, err := this.api.apis.Connector(connector)
	if err != nil {
		return 0, err
	}
	if conn == nil {
		return 0, ErrConnectorNotFound
	}
	if conn.Type != "App" {
		return 0, errors.New("connector is not an app connector")
	}
	direction := _connector.Direction(dir)
	c, err := newAppConnection(context.Background(), conn.Name, &_connector.AppConfig{
		Direction:    direction,
		ClientSecret: conn.ClientSecret,
		AccessToken:  accessToken,
	})
	if err != nil {
		return 0, err
	}
	resourceCode, err := c.Resource()
	if err != nil {
		return 0, err
	}
	var id int64
	err = this.myDB.Transaction(func(tx *sql.Tx) error {
		var resource int
		var currentRefreshToken string
		err := tx.QueryRow("SELECT `id`, `refreshToken` FROM `resources` WHERE `connector` = ? AND `code` = ?",
			connector, resourceCode).Scan(&resource, &currentRefreshToken)
		if err != nil {
			if err != sql.ErrNoRows {
				return err
			}
			err = nil
		}
		if resource == 0 {
			result, err := tx.Exec("INSERT INTO `resources` "+
				"SET `connector` = ?, `code` = ?, `accessToken` = ?, `refreshToken` = ?, `accessTokenExpirationTime` = ?",
				connector, resourceCode, accessToken, refreshToken, accessTokenExpirationTime)
			if err != nil {
				return err
			}
			resourceID, err := result.LastInsertId()
			resource = int(resourceID)
		} else if refreshToken != currentRefreshToken {
			_, err = tx.Exec("UPDATE `resources` "+
				"SET `accessToken` = ?, `refreshToken` = ?, `accessTokenExpirationTime` = ? WHERE `id` = ?",
				accessToken, refreshToken, accessTokenExpirationTime, resource)
		}
		if err != nil {
			return err
		}
		result, err := tx.Exec("INSERT INTO `connections`\n"+
			"SET `workspace` = ?, `type` = 'App', `direction` = ?, `connector` = ?, `resource` = ?",
			this.workspace, dir, connector, resource)
		if err != nil {
			return err
		}
		id, err = result.LastInsertId()
		return err
	})
	if err != nil {
		return 0, err
	}

	go func() {
		err := this.reloadProperties(int(id))
		if err != nil {
			log.Printf("[error] cannot reload properties for connection %d: %s", id, err)
		}
	}()

	return int(id), err
}

// AddDatabase adds a database connection given its direction, database
// connector and returns its identifier.
//
// If the connector does not exist, it returns the ErrConnectorNotFound error.
func (this *Connections) AddDatabase(dir Direction, connector int) (int, error) {
	if dir != SourceDir && dir != DestDir {
		return 0, errors.New("invalid direction")
	}
	if connector <= 0 {
		return 0, errors.New("invalid connector")
	}
	var id int64
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		var connectorType string
		err := tx.QueryRow("SELECT `type` FROM `connectors` WHERE `id` = ?", connector).Scan(&connectorType)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrConnectorNotFound
			}
			return err
		}
		if connectorType != "Database" {
			return errors.New("connector is not a database connector")
		}
		result, err := tx.Exec("INSERT INTO `connections`\n"+
			"SET `workspace` = ?, `type` = 'Database', `direction` = ?, `connector` = ?",
			this.workspace, dir, connector)
		id, err = result.LastInsertId()
		return err
	})
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

// AddFile adds a file connection given its direction, connector and storage
// connection and returns its identifier. If storage is 0, the file connection
// does not have a storage, otherwise storage must have direction dir.
//
// If the connector does not exist, it returns the ErrConnectorNotFound error.
// If the storage does not exist, it returns the ErrStorageNotFound error.
func (this *Connections) AddFile(dir Direction, connector, storage int) (int, error) {
	if dir != SourceDir && dir != DestDir {
		return 0, errors.New("invalid direction")
	}
	if connector <= 0 {
		return 0, errors.New("invalid connector")
	}
	if storage < 0 {
		return 0, errors.New("invalid storage")
	}
	var id int64
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		var typ string
		err := tx.QueryRow("SELECT `type` FROM `connectors` WHERE `id` = ?", connector).Scan(&typ)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrConnectorNotFound
			}
			return err
		}
		if typ != "File" {
			return errors.New("connector is not a file connector")
		}
		if storage > 0 {
			// Check the storage.
			var storageDir Direction
			err = tx.QueryRow("SELECT `type`, CAST(`direction` AS UNSIGNED) FROM `connections` WHERE `id` = ?", storage).Scan(&typ, &storageDir)
			if err != nil {
				if err == sql.ErrNoRows {
					return ErrStorageNotFound
				}
				return err
			}
			if typ != "Storage" {
				return errors.New("storage is not a storage connection")
			}
			if storageDir != dir {
				if dir == SourceDir {
					return errors.New("storage is not a source")
				}
				return errors.New("storage is not a destination")
			}
		}
		result, err := tx.Exec("INSERT INTO `connections`\n"+
			"SET `workspace` = ?, `type` = 'File', `direction` = ?, `connector` = ? AND `storage` = ?",
			this.workspace, dir, connector, storage)
		if err != nil {
			return err
		}
		id, err = result.LastInsertId()
		return err
	})
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

// AddStorage adds a storage connection given its direction and connector and
// returns its identifier.
//
// If the connector does not exist, it returns the ErrConnectorNotFound error.
func (this *Connections) AddStorage(dir Direction, connector int) (int, error) {
	if dir != SourceDir && dir != DestDir {
		return 0, errors.New("invalid direction")
	}
	if connector <= 0 {
		return 0, errors.New("invalid connector")
	}
	var id int64
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		var typ string
		err := tx.QueryRow("SELECT `type` FROM `connectors` WHERE `id` = ?", connector).Scan(&typ)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrConnectorNotFound
			}
			return err
		}
		if typ != "Storage" {
			return errors.New("connector is not a storage connector")
		}
		result, err := tx.Exec("INSERT INTO `connections`\n"+
			"SET `workspace` = ?, `type` = 'Storage', `direction` = ?, `connector` = ?",
			this.workspace, dir, connector)
		if err != nil {
			return err
		}
		id, err = result.LastInsertId()
		return err
	})
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

// Get returns the connection with identifier id. If the connection does not
// exist, it returns the ErrConnectionNotFound error.
func (this *Connections) Get(id int) (*ConnectionInfo, error) {
	if id <= 0 {
		return nil, errors.New("invalid connection identifier")
	}
	s := ConnectionInfo{ID: id}
	err := this.myDB.QueryRow(
		"SELECT `s`.`type`, CAST(`s`.`direction` AS UNSIGNED), `c`.`name`, `c`.`logoURL`, `s`.`usersQuery`\n"+
			"FROM `connections` AS `s`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
			"WHERE `s`.`id` = ? AND `s`.`workspace` = ?",
		id, this.workspace).Scan(&s.Type, &s.Direction, &s.Name, &s.LogoURL, &s.UsersQuery)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrConnectionNotFound
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
	if id <= 0 {
		return errors.New("invalid connection identifier")
	}
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		connection, err := tx.Table("Connections").Get(
			sql.Where{"id": id, "workspace": this.workspace},
			sql.Columns{"type", "resource"})
		if err != nil {
			return err
		}
		if connection == nil {
			return nil
		}
		if connection["type"] == "Storage" {
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
// connection must be a source and cannot be a storage.
//
// Returns the ErrConnectionNotFound error if the connection does not exist.
// Returns the ErrConnectionDisabled error if the connection does not have any
// transformation function associated to it.
// Returns the ErrFileHasNoStorage error if the connection is a file and does
// not have a storage.
func (this *Connections) Import(id int, reimport bool) error {

	if id <= 0 {
		return errors.New("invalid connection identifier")
	}

	// Check that the connection exists, is a source and has a transformation.
	var typ string
	var dir Direction
	err := this.myDB.QueryRow("SELECT `type`, CAST(`direction` AS UNSIGNED)\n"+
		"FROM `connections`"+
		"WHERE `id` = ? AND `workspace` = ?",
		id, this.workspace).Scan(&typ, &dir)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrConnectionNotFound
		}
		return err
	}
	if typ == "Storage" {
		return errors.New("cannot import from a storage")
	}
	if dir == DestDir {
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

	const noColumn = -1
	const direction = _connector.SourceDir

	switch typ {
	case "App":

		var name, connectorType, clientSecret, webhooksPer, resourceCode, accessToken, refreshToken, cursor string
		var connector, resource int
		var settings, rawUsedProperties []byte
		var expiration time.Time
		err = this.myDB.QueryRow(
			"SELECT `c`.`name`, `c`.`type`, `c`.`clientSecret`, `c`.`webhooksPer`, `r`.`code`, `r`.`accessToken`,"+
				" `r`.`refreshToken`, `r`.`accessTokenExpirationTime`, `s`.`connector`,"+
				" `s`.`resource`, `s`.`userCursor`, `s`.`settings`, `s`.`usedProperties`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"INNER JOIN `resources` AS `r` ON `r`.`id` = `s`.`resource`\n"+
				"WHERE `s`.`id` = ?", id).Scan(
			&name, &connectorType, &clientSecret, &webhooksPer, &resourceCode, &accessToken, &refreshToken, &expiration, &connector,
			&resource, &cursor, &settings, &rawUsedProperties)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrConnectionNotFound
			}
			return err
		}
		if reimport {
			cursor = ""
		}
		var properties [][]string
		err = json.Unmarshal(rawUsedProperties, &properties)
		if err != nil {
			return fmt.Errorf("cannot unmarshal used properties of connection %d: %s", id, err)
		}

		accessTokenExpired := time.Now().UTC().Add(15 * time.Minute).After(expiration)

		if accessToken == "" || accessTokenExpired {
			accessToken, err = this.api.apis.refreshOAuthToken(resource)
			if err != nil {
				return err
			}
		}

		go func() {
			fh := this.newFirehose(context.Background(), id, connector, resource, connectorType, direction, webhooksPer)
			c, err := newAppConnection(fh.ctx, name, &_connector.AppConfig{
				Direction:    direction,
				Settings:     settings,
				Firehose:     fh,
				ClientSecret: clientSecret,
				Resource:     resourceCode,
				AccessToken:  accessToken,
			})
			if err != nil {
				log.Printf("[error] cannot connect to the connector %d of the connection %d: %s", connector, id, err)
				return
			}
			err = c.Users(cursor, properties)
			if err != nil {
				log.Printf("[error] call to the Users method of the connection %d failed: %s", id, err)
			}
		}()

	case "Database":

		var connectorName, identityColumn, timestampColumn, usersQuery string
		var connector int
		var settings []byte
		err = this.myDB.QueryRow(
			"SELECT `c`.`name`, `s`.`connector`, `s`.`identityColumn`, `s`.`timestampColumn`, `s`.`settings`, `s`.`usersQuery`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"WHERE `s`.`id` = ?", id).Scan(&connectorName, &connector, &identityColumn, &timestampColumn, &settings, &usersQuery)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrConnectionNotFound
			}
			return err
		}

		usersQuery, err := this.compileQueryWithoutLimit(usersQuery)
		if err != nil {
			return err
		}
		fh := this.newFirehose(context.Background(), id, connector, 0, "Database", direction, "")
		c, err := newDatabaseConnection(fh.ctx, connectorName, &_connector.DatabaseConfig{
			Direction: direction,
			Settings:  settings,
			Firehose:  fh,
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

	case "File":

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
				return ErrConnectionNotFound
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
			fh := this.newFirehose(ctx, storage, connector, 0, "Storage", direction, "")
			ctx = fh.ctx
			c, err := newStorageConnection(ctx, connectorName, &_connector.StorageConfig{
				Direction: direction,
				Settings:  settings,
				Firehose:  fh,
			})
			if err != nil {
				return err
			}
			files = newFileReader(c)
		}

		// Connect to the file connector.
		fh := this.newFirehose(ctx, id, connector, 0, "File", direction, "")
		file, err := newFileConnection(fh.ctx, connectorName, &_connector.FileConfig{
			Direction: direction,
			Settings:  settings,
			Firehose:  fh,
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
		"SELECT `s`.`id`, `s`.`type`, CAST(`s`.`direction` AS UNSIGNED), `s`.`storage`, `c`.`name`, `c`.`oauthURL`, `c`.`logoURL`\n"+
			"FROM `connections` as `s`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
			"WHERE `s`.`workspace` = ?", this.workspace, func(rows *sql.Rows) error {
			var err error
			for rows.Next() {
				var c Connection
				if err = rows.Scan(&c.ID, &c.Type, &c.Direction, &c.Storage, &c.Name,
					&c.OauthURL, &c.LogoURL); err != nil {
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
// with the given identifier. The connection cannot be a storage.
// Returns the ErrConnectionNotFound error if the connection does not exist.
func (this *Connections) Properties(id int) ([]ConnectionProperty, [][]string, error) {
	if id <= 0 {
		return nil, nil, errors.New("invalid connection identifier")
	}
	var typ string
	var rawProperties, rawUsedProperties []byte
	err := this.myDB.QueryRow("SELECT `type`, `properties`, `usedProperties`\n"+
		"FROM `connections`\n"+
		"WHERE `id` = ? AND `workspace` = ?", id, this.workspace).Scan(&typ, &rawProperties, &rawUsedProperties)
	if err != nil {
		return nil, nil, err
	}
	if typ == "Storage" {
		return nil, nil, errors.New("cannot read properties from a storage")
	}
	var properties []ConnectionProperty
	if len(rawProperties) > 0 {
		err = json.Unmarshal(rawProperties, &properties)
		if err != nil {
			return nil, nil, errors.New("cannot unmarshal connection properties")
		}
	} else {
		properties = []ConnectionProperty{}
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
// It returns the ErrConnectionNotFound error if the connection does not exist
// and returns a DatabaseQueryError error if an error occurred while executing
// the query.
func (this *Connections) Query(id int, query string, limit int) ([]Column, [][]string, error) {

	if id <= 0 {
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

	var typ, connectorName string
	var dir Direction
	var connector int
	var settings []byte
	err := this.myDB.QueryRow(
		"SELECT `s`.`type`, CAST(`s`.`direction` AS UNSIGNED), `s`.`connector`, `s`.`settings`, `c`.`name`\n"+
			"FROM `connections` AS `s`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
			"WHERE `s`.`id` = ? AND `s`.`workspace` = ?", id, this.workspace).Scan(
		&typ, &dir, &connector, &settings, &connectorName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ErrConnectionNotFound
		}
		return nil, nil, err
	}
	if typ != "Database" {
		return nil, nil, errors.New("connection is not a database")
	}
	if dir != SourceDir {
		return nil, nil, errors.New("connection is not a source")
	}
	const direction = _connector.SourceDir

	// Execute the query.
	query, err = this.compileQueryWithLimit(query, limit)
	if err != nil {
		return nil, nil, err
	}
	fh := this.newFirehose(context.Background(), id, connector, 0, typ, direction, "")
	c, err := newDatabaseConnection(fh.ctx, connectorName, &_connector.DatabaseConfig{
		Direction: direction,
		Settings:  settings,
		Firehose:  fh,
	})
	if err != nil {
		return nil, nil, err
	}
	rawColumns, rawRows, err := c.Query(query)
	if err != nil {
		if err, ok := err.(_connector.DatabaseQueryError); ok {
			return nil, nil, DatabaseQueryError{Message: err.Message}
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
// Returns the ErrConnectionNotFound error if the connection does not exist and
// the ErrUIEventNotExist error if the event does not exist.
func (this *Connections) ServeUI(id int, event string, values []byte) ([]byte, error) {

	if id <= 0 {
		return nil, errors.New("invalid connection identifier")
	}

	var typ string
	var dir Direction
	err := this.myDB.QueryRow("SELECT `type`, CAST(`direction` AS UNSIGNED) FROM `connections` WHERE `id` = ? AND `workspace` = ?",
		id, this.workspace).Scan(&typ, &dir)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrConnectionNotFound
		}
		return nil, err
	}

	direction := _connector.Direction(dir)

	var connection _connector.Connection

	switch typ {
	case "App":

		var connectorName, connectorType, clientSecret, webhooksPer, resourceCode, accessToken string
		var connector, resource int
		var settings []byte
		var expiration time.Time
		err = this.myDB.QueryRow(
			"SELECT `c`.`name`, `c`.`type`, `c`.`clientSecret`, `c`.`webhooksPer`, `r`.`code`, `r`.`accessToken`,"+
				" `r`.`accessTokenExpirationTime`, `s`.`connector`, `s`.`resource`, `s`.`settings`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"INNER JOIN `resources` AS `r` ON `r`.`id` = `s`.`resource`\n"+
				"WHERE `s`.`id` = ?", id).Scan(
			&connectorName, &connectorType, &clientSecret, &webhooksPer, &resourceCode, &accessToken, &expiration,
			&connector, &resource, &settings)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, ErrConnectionNotFound
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

		fh := this.newFirehose(context.Background(), id, connector, resource, connectorType, direction, webhooksPer)
		connection, err = newAppConnection(fh.ctx, connectorName, &_connector.AppConfig{
			Direction:    direction,
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
				return nil, ErrConnectionNotFound
			}
			return nil, err
		}

		fh := this.newFirehose(context.Background(), id, connector, 0, typ, direction, "")

		switch typ {
		case "Database":
			connection, err = newDatabaseConnection(fh.ctx, connectorName, &_connector.DatabaseConfig{
				Direction: direction,
				Settings:  settings,
				Firehose:  fh,
			})
		case "File":
			connection, err = newFileConnection(fh.ctx, connectorName, &_connector.FileConfig{
				Direction: direction,
				Settings:  settings,
				Firehose:  fh,
			})
		case "Storage":
			connection, err = newStorageConnection(fh.ctx, connectorName, &_connector.StorageConfig{
				Direction: direction,
				Settings:  settings,
				Firehose:  fh,
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

	return json.Marshal(form)
}

// SetUsersQuery sets the users query of the database connection with
// identifier id. query must be UTF-8 encoded, it cannot be longer than
// 16,777,215 runes and must contain the ':limit' placeholder.
//
// It returns an error if the connection is a destination.
// It returns the ErrConnectionNotFound error if the connection does not exist.
func (this *Connections) SetUsersQuery(id int, query string) error {

	if id <= 0 {
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
		"WHERE `id` = ? AND `workspace` = ? AND `type` = 'Database' AND `direction` = 'Source'",
		query, id, this.workspace)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		var typ string
		var dir Direction
		err = this.myDB.QueryRow("SELECT `type`, CAST(`direction` AS UNSIGNED) FROM `connections` WHERE `id` = ? AND `workspace` = ?",
			id, this.workspace).Scan(&typ, &dir)
		if err != nil {
			return err
		}
		if typ != "Database" {
			return errors.New("connection is not a database")
		}
		if dir != SourceDir {
			return errors.New("connection is not a source")
		}
		return ErrConnectionNotFound
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
	if id <= 0 {
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
func (this *Connections) newFirehose(ctx context.Context, connection, connector, resource int, connectorType string, direction _connector.Direction, webhooksPer string) *firehose {
	fh := &firehose{
		connections:   this,
		connection:    connection,
		resource:      resource,
		connector:     connector,
		connectorType: connectorType,
		direction:     direction,
		webhooksPer:   webhooksPer,
	}
	fh.ctx, fh.cancel = context.WithCancel(ctx)
	return fh
}

var errRecordStop = errors.New("stop record")

// reloadProperties reloads the properties of the connection with identifier id.
// The connection cannot be a storage.
//
// If the connection does not exist it returns the ErrConnectionNotFound error.
func (this *Connections) reloadProperties(id int) error {

	if id <= 0 {
		return errors.New("invalid connection identifier")
	}

	var typ string
	var dir Direction
	err := this.myDB.QueryRow("SELECT `type`, CAST(`direction` AS UNSIGNED) FROM `connections` WHERE `id` = ? AND `workspace` = ?",
		id, this.workspace).Scan(&typ, &dir)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrConnectionNotFound
		}
		return err
	}
	if typ == "Storage" {
		return errors.New("cannot reload properties of a storage")
	}

	direction := _connector.Direction(dir)

	var properties []_connector.Property

	switch typ {
	case "App":

		// TODO(marco) The following code is duplicated in the Import method.
		var connectorName, clientSecret, webhooksPer, resourceCode, accessToken, refreshToken, cursor string
		var connector, resource int
		var settings []byte
		var expiration *time.Time
		err = this.myDB.QueryRow(
			"SELECT `c`.`name`, `c`.`clientSecret`, `c`.`webhooksPer`, IFNULL(`r`.`code`, ''), IFNULL(`r`.`accessToken`, ''),"+
				" IFNULL(`r`.`refreshToken`, ''), `r`.`accessTokenExpirationTime`, `s`.`connector`,"+
				" `s`.`resource`, `s`.`userCursor`, `s`.`settings`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"LEFT JOIN `resources` AS `r` ON `r`.`id` = `s`.`resource`\n"+
				"WHERE `s`.`id` = ?", id).Scan(
			&connectorName, &clientSecret, &webhooksPer, &resourceCode, &accessToken, &refreshToken, &expiration,
			&connector, &resource, &cursor, &settings)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrConnectionNotFound
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
		fh := this.newFirehose(context.Background(), id, connector, resource, "App", direction, webhooksPer)
		c, err := newAppConnection(fh.ctx, connectorName, &_connector.AppConfig{
			Direction:    direction,
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

	case "Database":

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
				return ErrConnectionNotFound
			}
			return err
		}

		usersQuery, err := this.compileQueryWithLimit(usersQuery, 0)
		if err != nil {
			return err
		}
		fh := this.newFirehose(context.Background(), id, connector, 0, "Database", direction, "")
		c, err := newDatabaseConnection(fh.ctx, connectorName, &_connector.DatabaseConfig{
			Direction: direction,
			Settings:  settings,
			Firehose:  fh,
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
		properties = make([]_connector.Property, len(columns))
		for i := 0; i < len(properties); i++ {
			properties[i].Name = columns[i].Name
			properties[i].Type = columns[i].Type
		}

	case "File":

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
				return ErrConnectionNotFound
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
			fh := this.newFirehose(ctx, storage, connector, 0, "Storage", direction, "")
			ctx = fh.ctx
			c, err := newStorageConnection(ctx, connectorName, &_connector.StorageConfig{
				Direction: direction,
				Settings:  settings,
				Firehose:  fh,
			})
			if err != nil {
				return err
			}
			files = newFileReader(c)
		}

		// Connect to the file connector and read only the columns.
		fh := this.newFirehose(ctx, id, connector, 0, "File", direction, "")
		file, err := newFileConnection(fh.ctx, connectorName, &_connector.FileConfig{
			Direction: direction,
			Settings:  settings,
			Firehose:  fh,
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

		properties = make([]_connector.Property, len(records.columns))
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
