//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/state"
	_connector "chichi/connector"
)

const (
	identityColumn  = "identity"
	timestampColumn = "timestamp"
)

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
func (this *Connection) Export() (err error) {

	c := this.connection
	ws := c.Workspace()
	connector := c.Connector()

	// Verify that the workspace has a data warehouse.
	if ws.Warehouse == nil {
		return errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", ws.ID)
	}

	// Check that the connection has an allowed type and is a source.
	var storage int
	switch connector.Type {
	case state.AppType:
	default:
		return errors.BadRequest("cannot export to connection %d, it's a %s connection",
			c.ID, strings.ToLower(connector.Type.String()))
	}
	if c.Role == state.SourceRole {
		return errors.BadRequest("connection %d is not a destination", c.ID)
	}

	// Check that the connection has at least one mapping associated to it.
	if len(c.Mappings()) == 0 {
		return errors.Unprocessable(NoMappings, "connection %d has no mappings", c.ID)
	}

	// Track the export in the database.
	var exportID int
	err = this.db.QueryRow("INSERT INTO connections_exports (connection, storage, start_time)\n"+
		"VALUES ($1, $2, $3)\nRETURNING id", c.ID, storage, time.Now().UTC()).Scan(&exportID)
	if err != nil {
		return err
	}

	// Start the export.
	go func() {
		err = this.startExport()
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

// Import starts the import of the users from the connection. The connection
// must be a source app, database or file. If the connection is an app and
// reimport is false, it imports the users from the current cursor, otherwise
// imports all users.
//
// If the workspace does not have a data warehouse, it returns an
// errors.UnprocessableError error with code NoWarehouse.
// If the connection is a file and does not have a storage, it returns an
// errors.UnprocessableError error with code NoStorage.
// If the connection has no mappings, it returns an errors.UnprocessableError
// error with code NoMappings.
func (this *Connection) Import(reimport bool) (err error) {

	c := this.connection
	ws := c.Workspace()
	connector := c.Connector()

	// Verify that the workspace has a data warehouse.
	if ws.Warehouse == nil {
		return errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", ws.ID)
	}

	// Check that the connection has an allowed type and is a source.
	var storage int
	switch connector.Type {
	case state.AppType, state.DatabaseType, state.StreamType:
	case state.FileType:
		if s := c.Storage(); s == nil {
			return errors.Unprocessable(NoStorage, "file connection %d does not have a storage", c.ID)
		}
		storage = c.ID
	default:
		return errors.BadRequest("cannot import from connection %d, it's a %s connection",
			c.ID, strings.ToLower(connector.Type.String()))
	}
	if c.Role == state.DestinationRole {
		return errors.BadRequest("connection %d is not a source", c.ID)
	}

	// Check that the connection has at least one mapping associated to it.
	if connector.Type != state.StreamType {
		if len(c.Mappings()) == 0 {
			return errors.Unprocessable(NoMappings, "connection %d has no mappings", c.ID)
		}
	}

	// Track the import in the database.
	n := state.AddImportInProgressNotification{
		Connection: c.ID,
		Storage:    storage,
		Reimport:   reimport,
		StartTime:  time.Now().UTC(),
	}
	err = this.db.Transaction(func(tx *postgres.Tx) error {
		err := tx.QueryRow("INSERT INTO connections_imports (connection, storage, start_time)\n"+
			"VALUES ($1, $2, $3)\nRETURNING id", n.Connection, n.Storage, n.StartTime).Scan(&n.ID)
		if err != nil {
			return err
		}
		return tx.Notify(n)
	})

	return err
}

// startImport starts the imp import.
// It is called by the state keeper in its own goroutine.
func (this *Connection) startImport(imp *state.ImportInProgress) {

	var errorMsg string

	err := this._startImport(imp)
	if err != nil {
		if e, ok := err.(importError); ok {
			errorMsg = abbreviate(e.Error(), 1000)
		} else {
			log.Printf("[error] cannot do import %d: %s", imp.ID, err)
			errorMsg = "an internal error has occurred"
		}
	}
	n := state.EndImportNotification{
		ID: imp.ID,
	}
	// TODO(marco) retry if the transaction fails.
	err2 := this.db.Transaction(func(tx *postgres.Tx) error {
		_, err := tx.Exec("UPDATE connections_imports SET end_time = $1, error = $2 WHERE id = $3",
			time.Now().UTC(), errorMsg, imp.ID)
		if err != nil {
			return err
		}
		return tx.Notify(n)
	})
	if err2 != nil {
		log.Printf("[error] cannot update the end of import %d into the database: %s", imp.ID, err2)
	}

}

// _startImport is called by the startImport method to start the imp import.
func (this *Connection) _startImport(imp *state.ImportInProgress) error {

	const noColumn = -1
	const role = _connector.SourceRole

	connection := imp.Connection()
	connector := connection.Connector()

	switch connector.Type {
	case state.AppType:

		var clientSecret, resourceCode, accessToken string
		if r := connection.Resource(); r != nil {
			clientSecret = connector.OAuth.ClientSecret
			resourceCode = r.Code
			var err error
			accessToken, err = freshAccessToken(this.db, r)
			if err != nil {
				return importError{fmt.Errorf("cannot retrive the OAuth access token: %s", err)}
			}
		}

		// Read the properties to read.
		_, properties, err := this.userSchema()
		if err != nil {
			return fmt.Errorf("cannot read user schema: %s", err)
		}

		fh := this.newFirehose(context.Background())
		c, err := _connector.RegisteredApp(connector.Name).Connect(fh.ctx, &_connector.AppConfig{
			Role:         role,
			Settings:     connection.Settings,
			Firehose:     fh,
			ClientSecret: clientSecret,
			Resource:     resourceCode,
			AccessToken:  accessToken,
		})
		if err != nil {
			return importError{fmt.Errorf("cannot connect to the connector: %s", err)}
		}
		cursor := connection.UserCursor
		if imp.Reimport {
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

	case state.DatabaseType:

		usersQuery, err := compileConnectionQuery(connection.UsersQuery, noQueryLimit)
		if err != nil {
			return importError{err}
		}
		fh := this.newFirehose(context.Background())
		c, err := _connector.RegisteredDatabase(connector.Name).Connect(fh.ctx, &_connector.DatabaseConfig{
			Role:     role,
			Settings: connection.Settings,
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

	case state.FileType:

		var ctx = context.Background()

		// Get the file reader.
		var files *fileReader
		{
			fh := this.newFirehoseForConnection(ctx, connection.Storage())
			ctx = fh.ctx
			c, err := _connector.RegisteredStorage(connector.Name).Connect(ctx, &_connector.StorageConfig{
				Role:     role,
				Settings: connection.Settings,
				Firehose: fh,
			})
			if err != nil {
				return importError{fmt.Errorf("cannot connect to the storage connector: %s", err)}
			}
			files = newFileReader(c)
		}

		// Connect to the file connector.
		fh := this.newFirehose(ctx)
		file, err := _connector.RegisteredFile(connector.Name).Connect(fh.ctx, &_connector.FileConfig{
			Role:     role,
			Settings: connection.Settings,
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

	case state.StreamType:

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c, err := _connector.RegisteredStream(connector.Name).Connect(ctx, &_connector.StreamConfig{
			Role:     role,
			Settings: connection.Settings,
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

	}

	return nil
}

// startExport starts the export for the connection. Note that this method is
// only a draft, and its code may be wrong and/or partially implemented.
func (this *Connection) startExport() error {

	const role = _connector.SourceRole

	connection := this.connection
	connector := connection.Connector()

	switch connector.Type {
	case state.AppType:

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
				"WHERE `s`.`id` = ?", connection.ID).Scan(
			&name, &clientSecret, &webhooksPer, &resourceCode, &accessToken, &refreshToken, &expiration, &connector,
			&resource, &settings)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}

		fh := this.newFirehose(context.Background())
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

		// Prepare the users to export to the connection.
		users := []_connector.User{}
		{
			// TODO(Gianluca): populate this map:
			internalToExternalID := map[int]string{}
			rows, err := this.db.Query("SELECT user, goldenRecord FROM connection_users WHERE connection = $1", connection.ID)
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
				user, err := exportUser(id, user, connection.Mappings())
				if err != nil {
					return err
				}
				users = append(users, user)
			}
		}

		// Export the users to the connection.
		log.Printf("[info] exporting %d user(s) to the connection %d", len(users), connection.ID)
		err = c.SetUsers(users)
		if err != nil {
			return errors.New("cannot export users")
		}

		// Handle errors occurred in the firehose.
		if fh.err != nil {
			return fh.err
		}

	default:

		panic(fmt.Sprintf("export to %q not implemented", connector.Type))

	}

	return nil
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
