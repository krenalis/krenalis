//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"context"
	"log/slog"

	"chichi/apis/datastore"
	"chichi/apis/transformers"
	"chichi/connector/types"
)

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
			ok, err := filterApplies(this.action.Filter, user.Properties)
			if err != nil {
				return err
			}
			if ok {
				filteredUsers = append(filteredUsers, user)
			}
		}
		users = filteredUsers
	}

	// Instantiate a new transformer.
	transformer, err := transformers.New(this.action.InSchema, this.action.OutSchema, this.action.Mapping,
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

		// Transform the user.
		props, err = transformer.Transform(ctx, props)
		if err != nil {
			if err, ok := err.(transformers.Error); ok {
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
