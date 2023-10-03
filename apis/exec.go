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
	"time"

	"chichi/apis/datastore"
	"chichi/apis/datastore/expr"
	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/connector"
	"chichi/connector/types"
)

var ExecutionInProgress errors.Code = "ExecutionInProgress"

// addExecution adds an execution to the action.
func (this *Action) addExecution(ctx context.Context, reimport bool) error {

	n := state.ExecuteAction{
		Action:    this.action.ID,
		Reimport:  reimport,
		StartTime: time.Now().UTC(),
	}
	c := this.action.Connection()
	if storage, ok := c.Storage(); ok {
		n.Storage = storage.ID
	}

	err := this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
		err := tx.QueryVoid(ctx, "SELECT FROM actions_executions WHERE action = $1 AND end_time IS NULL", n.Action)
		if err != sql.ErrNoRows {
			if err == nil {
				err = errors.Unprocessable(ExecutionInProgress, "execution of action %d is in progress", this.action.ID)
			}
			return err
		}
		err = tx.QueryRow(ctx, "INSERT INTO actions_executions (action, storage, start_time)\n"+
			"VALUES ($1, NULLIF($2, 0), $3)\nRETURNING id", n.Action, n.Storage, n.StartTime).Scan(&n.ID)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				if postgres.ErrConstraintName(err) == "actions_executions_action_fkey" {
					err = errors.NotFound("action %d does not exit", n.Action)
				}
				if postgres.ErrConstraintName(err) == "actions_executions_storage_fkey" {
					err = errors.Unprocessable(NoStorage, "connection of action %d does not have a storage", n.Action)
				}
			}
			return err
		}
		return tx.Notify(ctx, n)
	})

	return err
}

// exec executes the action.
//
// It is called in its own goroutine and the action have an execution to
// execute. In case of error, it writes the error with the execution status in
// the actions_executions table.
func (this *Action) exec(ctx context.Context) {

	connection := this.action.Connection()
	execution, _ := this.action.Execution()
	c := connection.Connector()

	var err error
	if this.Target == GroupsTarget {
		err = actionExecutionError{fmt.Errorf("groups import and export are not implemented")}
	} else if !this.isLanguageSupported() {
		err = actionExecutionError{fmt.Errorf("%s transformation language is not supported", this.Transformation.Language)}
	} else {
		switch c.Type {
		case state.AppType:
			if connection.Role == state.SourceRole {
				err = this.importFromApp(ctx)
			} else {
				err = this.exportUsersToApp(ctx)
			}
		case state.DatabaseType:
			if connection.Role == state.SourceRole {
				err = this.importFromDatabase(ctx)
			} else {
				err = this.exportUsersToDatabase(ctx)
			}
		case state.FileType:
			if connection.Role == state.SourceRole {
				err = this.importFromFile(ctx)
			} else {
				err = this.exportUsersToFile(ctx)
			}
		}
	}
	endTime := time.Now().UTC()

	var health state.Health
	var errorMessage string

	if err != nil {
		health = state.RecentError
		select {
		case <-ctx.Done():
			errorMessage = "execution has been cancelled"
		default:
			if e, ok := err.(actionExecutionError); ok {
				errorMessage = abbreviate(e.Error(), 1000)
				if _, ok := e.err.(*connector.AccessDeniedError); ok {
					health = state.AccessDenied
				}
			} else {
				log.Printf("[error] cannot execute action %d, execution %d failed: %s", this.action.ID, execution.ID, err)
				errorMessage = "an internal error has occurred"
			}
		}
	}

	n := state.EndActionExecution{
		ID:     execution.ID,
		Health: health,
	}

	txCtx := context.Background()

	// TODO(marco) retry if the transaction fails.
	err = this.apis.db.Transaction(txCtx, func(tx *postgres.Tx) error {
		_, err := tx.Exec(txCtx, "UPDATE actions_executions SET end_time = $1, error = $2 WHERE id = $3",
			endTime, errorMessage, n.ID)
		if err != nil {
			return err
		}
		var exists bool
		err = tx.QueryRow(txCtx, "UPDATE actions SET health = $1 WHERE id = $2 RETURNING true",
			n.Health, this.action.ID).Scan(&exists)
		if err != nil {
			if err == sql.ErrNoRows {
				// The action does not exist anymore.
				return nil
			}
			return err
		}
		return tx.Notify(txCtx, n)
	})
	if err != nil {
		log.Printf("[error] cannot update the status of the execution %d of action %d: %s",
			execution.ID, this.action.ID, err)
	}

}

type userToExport struct {
	GID        int
	Properties map[string]any
}

// readUsersFromDataWarehouse reads the users with the given IDs from the data
// warehouse.
//
// TODO(Gianluca): this method returns at most 1000 users. This is wrong. We
// should find an alternative way to implement this; maybe we could read one
// user at a time.
func (this *Action) readUsersFromDataWarehouse(ctx context.Context, ids []int) ([]userToExport, error) {

	ws := this.action.Connection().Workspace()

	// Read the schema.
	schema, ok := ws.Schemas["users"]
	if !ok {
		return nil, errors.New("users schema not found")
	}

	// Read the users.

	var where expr.Expr
	if len(ids) > 0 {
		operands := make([]expr.Expr, len(ids))
		for i := range ids {
			operands[i] = expr.NewBaseExpr(
				expr.ExprColumn{Name: "id", Type: types.PtInt},
				expr.OperatorEqual,
				ids[i],
			)
		}
		where = expr.NewMultiExpr(expr.LogicalOperatorOr, operands)
	}
	idProperty, ok := schema.Property("id")
	if !ok {
		return nil, errors.New("property 'id' not found in schema")
	}

	store := this.connection.store
	users, err := store.Users(ctx, schema.Properties(), where, idProperty, 0, 1000)
	if err != nil {
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			log.Printf("[error] cannot get users from the data warehouse of the workspace %d: %s", ws.ID, err)
			return nil, errors.Unprocessable(DataWarehouseFailed, "warehouse connection is failed: %w", err.Err)
		}
		return nil, err
	}

	exportUsers := make([]userToExport, len(users))
	for i, user := range users {
		gid, ok := user["id"].(int)
		if !ok {
			return nil, errors.New("missing or invalid GID")
		}
		exportUsers[i] = userToExport{
			GID:        gid,
			Properties: user,
		}
	}

	return exportUsers, nil
}

// actionExecutionError represents a non-internal error during action execution.
type actionExecutionError struct {
	err error
}

func (err actionExecutionError) Error() string {
	return err.err.Error()
}
