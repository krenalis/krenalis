//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"context"
	"fmt"
	"time"

	"chichi/apis/errors"
	"chichi/apis/mappings"
	"chichi/apis/normalization"
	"chichi/apis/state"
	_connector "chichi/connector"
	"chichi/connector/types"
)

// importFromDatabase imports the users from a database.
func (this *Action) importFromDatabase() error {

	connection := this.action.Connection()
	connector := connection.Connector()

	// Compile the query.
	query, err := compileActionQuery(this.action.Query, noQueryLimit)
	if err != nil {
		return actionExecutionError{err}
	}

	// Instantiate a new firehose.
	ctx := context.Background()
	fh, err := this.newFirehose(ctx)
	if err != nil {
		return err
	}
	c, err := _connector.RegisteredDatabase(connector.Name).Open(fh.ctx, &_connector.DatabaseConfig{
		Role:     _connector.SourceRole,
		Settings: connection.Settings,
		Firehose: fh,
	})
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}

	// Execute the query and get the results and the properties.
	rawRows, properties, err := c.Query(query)
	if err != nil {
		return actionExecutionError{err}
	}
	defer rawRows.Close()

	// Determine the input and the output schema.
	apisConn := &Connection{
		db:         this.db,
		connection: this.action.Connection(),
		http:       this.http,
	}
	inputSchema, err := types.ObjectOf(properties)
	if err != nil {
		return actionExecutionError{err}
	}
	usersSchema, ok := this.action.Connection().Workspace().Schemas["users"]
	if !ok {
		return actionExecutionError{errors.New("users schema not loaded")}
	}
	outSchema := sourceMappingSchema(*usersSchema, state.DatabaseType)

	mapping, err := mappings.New(inputSchema, outSchema, this.action.Mapping, this.action.Transformation)
	if err != nil {
		return err
	}

	// Iterate over the database rows.
	dest := make([]any, len(properties))
	for rawRows.Next() {

		row := make(map[string]any, len(properties))
		for i, p := range properties {
			dest[i] = databaseScanValue{property: p, row: row}
		}
		if err := rawRows.Scan(dest...); err != nil {
			return actionExecutionError{fmt.Errorf("query execution failed: %s", err)}
		}

		// Apply the mapping or the transformation.
		mappedUser, err := mapping.Apply(ctx, row)
		if err != nil {
			return err
		}

		// Estrapolate the ID and the timestamp for the user.
		err = applyTimestampWorkaround(mappedUser)
		if err != nil {
			return err
		}
		id := mappedUser["id"].(string)
		delete(mappedUser, "id")
		timestamp, ok := mappedUser["timestamp"].(time.Time)
		if !ok {
			timestamp = time.Now().UTC()
		}
		delete(mappedUser, "timestamp")

		// Write the user and the mapped user on the database.
		err = apisConn.writeConnectionUsers(ctx, id, row, timestamp, nil)
		if err != nil {
			return err
		}
		err = apisConn.setUser(ctx, id, mappedUser)
		if err != nil {
			return err
		}

	}
	if err = rawRows.Err(); err != nil {
		return actionExecutionError{fmt.Errorf("an error occurred closing the database: %s", err)}
	}

	// Handle errors occurred in the firehose.
	if fh.err != nil {
		return fh.err
	}

	return nil
}

// databaseScanValue implements the sql.Scanner interface to read the database
// values from a database connector.
type databaseScanValue struct {
	property types.Property
	row      map[string]any
}

func (sv databaseScanValue) Scan(src any) error {
	p := sv.property
	value, err := normalization.NormalizeDatabaseFileProperty(p.Name, p.Nullable, p.Type, src)
	if err != nil {
		return err
	}
	sv.row[sv.property.Name] = value
	return nil
}
