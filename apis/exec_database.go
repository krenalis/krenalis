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
	"time"

	"chichi/apis/datastore"
	"chichi/apis/errors"
	"chichi/apis/mappings"
	"chichi/apis/normalization"
	"chichi/connector/types"
)

// importUsersFromDatabase imports the users from a database.
func (this *Action) importUsersFromDatabase(ctx context.Context) error {

	// Compile the query.
	query, err := replacePlaceholders(this.action.Query, func(name string) (string, bool) {
		if name == "limit" {
			return strconv.FormatUint(math.MaxUint64, 10), true
		}
		return "", false
	})

	if err != nil {
		return actionExecutionError{err}
	}

	mapping, err := mappings.New(this.action.InSchema, this.action.OutSchema, this.action.Mapping,
		this.action.Transformation, this.action.ID, this.apis.transformer, false)
	if err != nil {
		return err
	}

	// Execute the query and get the results and the properties.
	database := this.database()
	defer database.Close()
	rows, err := database.Query(ctx, query)
	if err != nil {
		return actionExecutionError{err}
	}
	defer rows.Close()

	properties := this.action.InSchema.PropertiesNames()

	// Iterate over the database rows.
	for rows.Next() {

		// Scan values into a map.
		row, err := rows.Scan()
		if err != nil {
			return actionExecutionError{fmt.Errorf("query execution failed: %s", err)}
		}

		// Determine and validate the external id and the timestamp, reading
		// them from the SQL expression named "id" and "timestamp" returned by
		// the query.
		var idExpr, timestampExpr types.Property
		for _, c := range rows.Columns() {
			switch c.Name {
			case "id":
				idExpr = c
			case "timestamp":
				timestampExpr = c
			}
		}
		if idExpr.Name == "" {
			return actionExecutionError{errors.New("expression with name 'id' not returned by the query")}
		}
		var externalID string
		{
			rawID, ok := row["id"]
			if !ok {
				return actionExecutionError{errors.New("no values for expression 'id' returned by the query")}
			}
			switch pt := idExpr.Type.PhysicalType(); {
			case pt == types.PtText || pt == types.PtJSON || (pt >= types.PtInt && pt <= types.PtUInt64):
				externalID = fmt.Sprint(rawID)
			default:
				return actionExecutionError{fmt.Errorf("expression 'id' with type %s cannot be used as identifier", pt)}
			}
		}
		var timestamp time.Time
		if timestampExpr.Name != "" {
			rawTimestamp, ok := row["timestamp"]
			if !ok {
				return actionExecutionError{errors.New("no values for expression 'timestamp' returned by the query")}
			}
			ts, err := normalization.NormalizeDatabaseFileProperty("timestamp", types.DateTime(), rawTimestamp, false)
			if err != nil {
				return actionExecutionError{fmt.Errorf("expression 'timestamp' cannot be used as identifier: %s", err)}
			}
			timestamp = ts.(time.Time)
		}

		// Take only the necessary properties.
		user := make(map[string]any, len(properties))
		for _, name := range properties {
			if v, ok := row[name]; ok {
				user[name] = v
			}
		}

		// Normalize the user properties (read from the database) using the
		// action's mapping input schema.
		user, err = normalize(user, this.action.InSchema)
		if err != nil {
			return actionExecutionError{err}
		}

		// Map the properties of the user.
		mappedUser, err := mapping.Apply(ctx, user)
		if err != nil {
			if err, ok := err.(mappings.Error); ok {
				return actionExecutionError{err}
			}
			return err
		}

		// Set the identity into the data warehouse.
		err = this.connection.store.SetIdentity(ctx, mappedUser, externalID, "", this.action.ID, false, timestamp)
		if err != nil {
			return err
		}

		// Update the connection stats.
		err = this.connection.updateConnectionsStats(ctx)
		if err != nil {
			return err
		}

	}
	if err = rows.Err(); err != nil {
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

	database := this.database()
	defer database.Close()
	err = database.Upsert(ctx, this.action.TableName, rows, columns)
	if err != nil {
		return err
	}

	slog.Info("users exported to database", "count", len(users), "table", this.action.TableName)

	return nil
}
