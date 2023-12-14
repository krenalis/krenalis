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

	"chichi/apis/datastore/warehouses"
	"chichi/apis/transformers"
	"chichi/connector/types"
)

// exportUsersToDatabase exports the users to the database of the action.
func (this *Action) exportUsersToDatabase(ctx context.Context) error {

	users, err := this.readUsersFromDataWarehouse(ctx, this.action.InSchema)
	if err != nil {
		return err
	}

	// Instantiate a new transformer.
	transformer, err := transformers.New(this.action.InSchema, this.action.OutSchema, this.action.Transformation,
		this.action.ID, this.apis.functionTransformer, nil)
	if err != nil {
		return err
	}

	inSchemaProps := this.action.InSchema.Properties()
	outSchemaProps := this.action.OutSchema.Properties()

	var rows []map[string]any

	for _, user := range users {

		// Take only the necessary properties.
		row := make(map[string]any, len(inSchemaProps))
		for _, p := range inSchemaProps {
			row[p.Name] = user.Properties[p.Name]
		}

		// Transform the user.
		row, err = transformer.Transform(ctx, row)
		if err != nil {
			if err, ok := err.(transformers.FunctionExecutionError); ok {
				return actionExecutionError{err}
			}
			return err
		}

		// Serialize the properties as columns.
		warehouses.SerializeRow(row, this.action.OutSchema)
		row["id"] = user.ID
		rows = append(rows, row)

	}

	columns := append([]types.Property{{Name: "id", Type: types.Int(32)}},
		warehouses.PropertiesToColumns(outSchemaProps)...)

	database := this.database()
	defer database.Close()
	err = database.Upsert(ctx, this.action.TableName, rows, columns)
	if err != nil {
		return err
	}

	slog.Info("users exported to database", "count", len(users), "table", this.action.TableName)

	return nil
}
