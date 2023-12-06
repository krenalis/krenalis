//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"context"
	"math"
	"strconv"

	"chichi/apis/connectors"
	"chichi/apis/state"
	"chichi/apis/transformers"
)

// importUsers imports the users of the action.
func (this *Action) importUsers(ctx context.Context) error {

	action := this.action
	connector := action.Connection().Connector()

	transformer, err := transformers.New(action.InSchema, action.OutSchema, action.Transformation, action.ID,
		this.apis.functionTransformer, nil)
	if err != nil {
		return actionExecutionError{err}
	}

	var records connectors.Records

	switch connector.Type {
	case state.AppType:
		var cursor state.Cursor
		if exe, _ := action.Execution(); !exe.Reimport {
			cursor.ID = action.UserCursor.ID
			cursor.Timestamp = action.UserCursor.Timestamp
		}
		records, err = this.app().Users(ctx, action.InSchema, cursor)
	case state.DatabaseType:
		var query string
		query, err = replacePlaceholders(action.Query, func(name string) (string, bool) {
			if name == "limit" {
				return strconv.FormatUint(math.MaxInt64, 10), true
			}
			return "", false
		})
		if err != nil {
			return actionExecutionError{err}
		}
		database := this.database()
		defer database.Close()
		records, err = database.Records(ctx, query, action.InSchema)
	case state.FileType:
		timestampColumn := connectors.TimestampColumn{
			Name:   action.TimestampColumn,
			Format: action.TimestampFormat,
		}
		records, err = this.file().Records(ctx, action.Path, action.Sheet, action.InSchema, action.IdentityColumn, timestampColumn)
	}
	if err != nil {
		if err, ok := err.(*connectors.SchemaError); ok {
			err.Msg += ". Please review and update the action before attempting to import the users."
		}
		return actionExecutionError{err}
	}
	defer records.Close()

	// processUsers does a bach processing of users.
	processUsers := func(users []connectors.Record) error {

		// Transform the users.
		err := transformer.TransformRecords(ctx, users)
		if err != nil {
			if err, ok := err.(transformers.FunctionExecutionError); ok {
				return actionExecutionError{err}
			}
			return err
		}

		// Set the identities into the data warehouse.
		for _, user := range users {
			if user.Err != nil {
				continue
			}
			err = this.connection.store.SetIdentity(ctx, user.Properties, user.ID, "", action.ID, false, user.Timestamp)
			if err != nil {
				return actionExecutionError{err}
			}
		}

		// Update the connection stats.
		err = this.connection.updateConnectionsStats(ctx, len(users))
		if err != nil {
			return actionExecutionError{err}
		}

		// Set the user cursor.
		if connector.Type == state.AppType {
			last := users[len(users)-1]
			err = this.setUserCursor(ctx, state.Cursor{ID: last.ID, Timestamp: last.Timestamp})
			if err != nil {
				return actionExecutionError{err}
			}
		}

		return nil
	}

	users := make([]connectors.Record, 0, 100)

	// Read the users.
	err = records.For(func(user connectors.Record) error {
		if user.Err != nil {
			return actionExecutionError{user.Err}
		}
		users = append(users, user)
		if len(users) == 100 {
			err := processUsers(users)
			if err != nil {
				return err
			}
			clear(users)
			users = users[0:0]
		}
		return err
	})
	if err != nil {
		return err
	}
	if err = records.Err(); err != nil {
		return actionExecutionError{err}
	}

	// Process the remaining users.
	if len(users) > 0 {
		err = processUsers(users)
		if err != nil {
			return err
		}
	}
	users = nil

	// Resolve and sync the users.
	err = this.connection.store.ResolveSyncUsers(ctx)
	if err != nil {
		return actionExecutionError{err}
	}

	return nil
}
