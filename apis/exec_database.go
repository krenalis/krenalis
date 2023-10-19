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
	"log/slog"

	"chichi/apis/datastore"
	"chichi/apis/errors"
	"chichi/apis/mappings"
	"chichi/apis/normalization"
	"chichi/connector/types"
)

// importFromDatabase imports the users from a database.
func (this *Action) importFromDatabase(ctx context.Context) error {

	// Compile the query.
	query, err := compileActionQuery(this.action.Query, noQueryLimit)
	if err != nil {
		return actionExecutionError{err}
	}

	database, err := this.connection.openDatabase()
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}

	// Execute the query and get the results and the properties.
	rawRows, properties, err := database.Query(ctx, query)
	if err != nil {
		return actionExecutionError{err}
	}
	defer rawRows.Close()

	mapping, err := mappings.New(this.action.InSchema, this.action.OutSchema, this.action.Mapping,
		this.action.Transformation, this.action.ID, this.apis.transformer, false)
	if err != nil {
		return err
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

		// Determine the external ID by taking the value for the column with
		// name "id" and normalizing it to a Text.
		var externalID string
		{
			rawID, ok := row["id"]
			if !ok {
				return actionExecutionError{errors.New("column 'id' not returned by the query")}
			}
			id, err := normalization.NormalizeDatabaseFileProperty("id", types.Text(), rawID, false)
			if err != nil {
				return actionExecutionError{err}
			}
			externalID = id.(string)
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
			if err, ok := err.(mappings.Error); ok {
				return actionExecutionError{err}
			}
			return err
		}

		// Set the identity into the data warehouse.
		err = this.connection.store.SetIdentity(ctx, mappedUser, externalID, "", this.action.ID, false)
		if err != nil {
			return err
		}

		// Update the connection stats.
		err = this.connection.updateConnectionsStats(ctx)
		if err != nil {
			return err
		}

	}
	if err = rawRows.Err(); err != nil {
		return actionExecutionError{fmt.Errorf("an error occurred closing the database: %s", err)}
	}

	// Resolve and sync the users.
	err = this.connection.store.ResolveSyncUsers(ctx)
	if err != nil {
		return actionExecutionError{err}
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

	users, err := this.readUsersFromDataWarehouse(ctx, nil)
	if err != nil {
		return err
	}

	// Filter the users.
	if this.action.Filter != nil {
		filteredUsers := []userToExport{}
		for _, user := range users {
			ok, err := mappings.FilterApplies(this.action.Filter, user.Properties)
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
	mapping, err := mappings.New(this.action.InSchema, this.action.OutSchema, this.action.Mapping,
		this.action.Transformation, this.action.ID, this.apis.transformer, true)
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
			if err, ok := err.(mappings.Error); ok {
				return actionExecutionError{err}
			}
			return err
		}

		// Serialize the props into column values.
		datastore.SerializeRow(props, this.action.OutSchema)
		row := make([]any, len(outSchemaProps)+1)
		row[0] = user.GID
		for i, p := range outSchemaProps {
			row[i+1] = props[p.Name]
		}
		rows = append(rows, row)
	}

	columns := append([]types.Property{{Name: "id", Type: types.Int()}},
		datastore.PropertiesToColumns(outSchemaProps)...)

	database, err := this.connection.openDatabase()
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}
	err = database.Upsert(ctx, this.action.TableName, rows, columns)
	_ = database.Close()

	slog.Info("user exported to database", "count", len(users), "table", this.action.TableName)

	return err
}
