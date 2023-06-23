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
	"log"
	"time"

	"chichi/apis/mappings"
	"chichi/apis/normalization"
	"chichi/apis/warehouses"
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

	ctx := context.Background()
	c, err := _connector.RegisteredDatabase(connector.Name).Open(ctx, &_connector.DatabaseConfig{
		Role:        _connector.SourceRole,
		Settings:    connection.Settings,
		SetSettings: this.setSettingsFunc(ctx),
	})
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}
	defer c.Close()

	// Execute the query and get the results and the properties.
	rawRows, properties, err := c.Query(query)
	if err != nil {
		return actionExecutionError{err}
	}
	defer rawRows.Close()

	mapping, err := mappings.New(this.action.OutSchema, this.action.InSchema, this.action.Mapping, this.action.PythonSource, false)
	if err != nil {
		return err
	}

	apisConn := &Connection{
		db:         this.db,
		connection: this.action.Connection(),
		http:       this.http,
	}

	inSchemaProps := this.action.InSchema.PropertiesNames()

	// Iterate over the database rows.
	dest := make([]any, len(properties))
	for rawRows.Next() {

		// Scan values into a map.
		row := make(map[string]any, len(properties))
		for i, p := range properties {
			dest[i] = databaseScanValue{property: p, row: row}
		}
		if err := rawRows.Scan(dest...); err != nil {
			return actionExecutionError{fmt.Errorf("query execution failed: %s", err)}
		}

		// Take only the necessary properties.
		props := make(map[string]any, len(inSchemaProps))
		for _, name := range inSchemaProps {
			if v, ok := row[name]; ok {
				props[name] = v
			}
		}

		// Normalize the user properties (read from the database) using the
		// action's mapping input schema.
		props, err := normalize(props, this.action.InSchema)
		if err != nil {
			return actionExecutionError{err}
		}

		// Map the properties of the user.
		mappedUser, err := mapping.Apply(ctx, props)
		if err != nil {
			return err
		}

		// Extrapolate the ID and the timestamp for the user.
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
	value, err := normalization.NormalizeDatabaseFileProperty(p.Name, p.Type, src, p.Nullable)
	if err != nil {
		return err
	}
	sv.row[sv.property.Name] = value
	return nil
}

// exportUsersToDatabase exports the users to the database of the action.
func (this *Action) exportUsersToDatabase(ctx context.Context) error {

	connection := this.action.Connection()
	connector := connection.Connector()

	users, err := this.readUsersFromDataWarehouse(nil)
	if err != nil {
		return err
	}

	// Filter the users.
	if this.action.Filter != nil {
		filteredUsers := []userToExport{}
		for _, user := range users {
			ok, err := mappings.ActionFilterApplies(this.action.Filter, user.Properties)
			if err != nil {
				return err
			}
			if ok {
				filteredUsers = append(filteredUsers, user)
			}
		}
		users = filteredUsers
	}

	// Instantiate a new mapping.
	mapping, err := mappings.New(this.action.InSchema, this.action.OutSchema, this.action.Mapping, this.action.PythonSource, true)
	if err != nil {
		return err
	}

	inSchemaProps := this.action.InSchema.Properties()
	outSchemaProps := this.action.OutSchema.Properties()

	var rows [][]any

	for _, user := range users {

		// Take only the necessary properties.
		props := make(map[string]any, len(inSchemaProps))
		for _, p := range inSchemaProps {
			props[p.Name] = user.Properties[p.Name]
		}

		// Normalize the user properties (read from the data warehouse) using
		// the action's mapping input schema.
		props, err = normalize(props, this.action.InSchema)
		if err != nil {
			return actionExecutionError{err}
		}

		// Map the properties of the user.
		props, err = mapping.Apply(ctx, props)
		if err != nil {
			return actionExecutionError{err}
		}

		// Serialize the props into column values.
		warehouses.SerializeRow(props, this.action.OutSchema)
		row := make([]any, len(outSchemaProps)+1)
		row[0] = user.GID
		for i, p := range outSchemaProps {
			row[i+1] = props[p.Name]
		}
		rows = append(rows, row)
	}

	columns := append([]types.Property{{Name: "id", Type: types.Int()}},
		warehouses.PropertiesToColumns(outSchemaProps)...)

	c, err := _connector.RegisteredDatabase(connector.Name).Open(ctx, &_connector.DatabaseConfig{
		Role:        _connector.SourceRole,
		Settings:    connection.Settings,
		SetSettings: this.setSettingsFunc(ctx),
	})
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}
	err = c.Upsert(this.action.TableName, rows, columns)
	_ = c.Close()

	log.Printf("[info] %d user(s) exported to database on the table %q", len(users), this.action.TableName)

	return err
}
