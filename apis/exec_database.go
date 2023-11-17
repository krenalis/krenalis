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
	"math"
	"strconv"

	"chichi/apis/datastore"
	"chichi/apis/mappings"
	"chichi/connector/types"
)

// importUsersFromDatabase imports the users from a database.
func (this *Action) importUsersFromDatabase(ctx context.Context) error {

	action := this.action

	// Replace the placeholders.
	query, err := replacePlaceholders(action.Query, func(name string) (string, bool) {
		if name == "limit" {
			return strconv.FormatUint(math.MaxInt64, 10), true
		}
		return "", false
	})
	if err != nil {
		return actionExecutionError{err}
	}

	mapping, err := mappings.New(action.InSchema, action.OutSchema, action.Mapping, action.Transformation, action.ID,
		this.apis.transformer, nil)
	if err != nil {
		return err
	}

	// Execute the query.
	database := this.database()
	defer database.Close()
	records, err := database.Records(ctx, query, action.InSchema)
	if err != nil {
		return actionExecutionError{err}
	}
	defer records.Close()

	// Read the users.
	for records.Next() {

		user, err := records.Scan()
		if err != nil {
			return actionExecutionError{fmt.Errorf("query execution failed: %s", err)}
		}

		// Transform the user's properties.
		user.Properties, err = mapping.Apply(ctx, user.Properties)
		if err != nil {
			if err, ok := err.(mappings.Error); ok {
				return actionExecutionError{err}
			}
			return err
		}

		// Set the identity into the data warehouse.
		err = this.connection.store.SetIdentity(ctx, user.Properties, user.ID, "", action.ID, false, user.Timestamp)
		if err != nil {
			return err
		}

		// Update the connection stats.
		err = this.connection.updateConnectionsStats(ctx)
		if err != nil {
			return err
		}

	}
	if err = records.Err(); err != nil {
		return actionExecutionError{fmt.Errorf("an error occurred closing the database: %s", err)}
	}

	// Resolve and sync the users.
	err = this.connection.store.ResolveSyncUsers(ctx)
	if err != nil {
		return actionExecutionError{err}
	}

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
		this.action.Transformation, this.action.ID, this.apis.transformer, nil)
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
		row[0] = user.ID
		for i, p := range outSchemaProps {
			row[i+1] = props[p.Name]
		}
		rows = append(rows, row)
	}

	columns := append([]types.Property{{Name: "id", Type: types.Int(32)}},
		datastore.PropertiesToColumns(outSchemaProps)...)

	database := this.database()
	defer database.Close()
	err = database.Upsert(ctx, this.action.TableName, rows, columns)
	if err != nil {
		return err
	}

	slog.Info("users exported to database", "count", len(users), "table", this.action.TableName)

	return nil
}
