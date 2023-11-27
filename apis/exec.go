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
	"log/slog"
	"strings"
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

	if this.connection.store == nil {
		err = actionExecutionError{fmt.Errorf("workspace %d does not have a data warehouse", connection.Workspace().ID)}
	} else if this.Target == Groups {
		err = actionExecutionError{fmt.Errorf("groups import and export are not implemented")}
	} else if !this.isLanguageSupported() {
		err = actionExecutionError{fmt.Errorf("%s transformation language is not supported", this.Transformation.Function.Language)}
	} else {
		if connection.Role == state.Source {
			err = this.importUsers(ctx)
		} else {
			switch c.Type {
			case state.AppType:
				err = this.exportUsersToApp(ctx)
			case state.DatabaseType:
				err = this.exportUsersToDatabase(ctx)
			case state.FileType:
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
				slog.Error("cannot execute action", "action", this.action.ID, "execution", execution.ID, "err", err)
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
		slog.Error("cannot update action execution status",
			"action", this.action.ID,
			"execution", execution.ID,
			"err", err,
		)
	}

}

type userToExport struct {
	ID         int
	Properties map[string]any
}

// readUsersFromDataWarehouse reads the users with the given IDs from the data
// warehouse.
//
// TODO(Gianluca): this method returns at most 1000 users. This is wrong. We
// should find an alternative way to implement this; maybe we could read one
// user at a time.
func (this *Action) readUsersFromDataWarehouse(ctx context.Context, ids []int) ([]userToExport, error) {

	// Read the schema.
	//
	//TODO(Gianluca): should the users / users_identities / events schema be
	// handled by Chichi, or internally by the data warehouse? See the issue
	// https://github.com/open2b/chichi/issues/392.
	//
	schema, err := this.connection.schema(ctx, "users")
	if err != nil {
		return nil, err
	}
	if !schema.Valid() {
		return nil, errors.New("users schema not found")
	}

	// Read the users.

	var where expr.Expr
	if len(ids) > 0 {
		operands := make([]expr.Expr, len(ids))
		for i := range ids {
			operands[i] = expr.NewBaseExpr(
				expr.Column{Name: "id", Type: types.IntKind},
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
			ws := this.action.Connection().Workspace()
			slog.Error("cannot get users from the data warehouse", "workspace", ws.ID, "err", err)
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
			ID:         gid,
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

// filterApplies reports whether the filter applies to props, which can be an
// event or a user. Returns error if one of the properties of the filter are not
// found within props.
func filterApplies(filter *state.Filter, props map[string]any) (bool, error) {
	if filter == nil {
		return true, nil
	}
	for _, cond := range filter.Conditions {
		value, ok := readPropertyFrom(props, strings.Split(cond.Property, "."))
		if !ok {
			return false, fmt.Errorf("property %q not found", cond.Property)
		}
		var conditionApplies bool
		switch cond.Operator {
		case "is":
			conditionApplies = value == cond.Value
		case "is not":
			conditionApplies = value != cond.Value
		}
		if conditionApplies && filter.Logical == "any" {
			return true, nil
		}
		if !conditionApplies && filter.Logical == "all" {
			return false, nil
		}
	}
	if filter.Logical == "any" {
		return false, nil // none of the conditions applied.
	}
	// All the conditions applied.
	return true, nil
}

// readPropertyFrom reads the property with the given path from m, returning its
// value (if found, otherwise nil) and a boolean indicating if the property path
// corresponds to a value in m or not.
func readPropertyFrom(m map[string]any, path types.Path) (any, bool) {
	name := path[0]
	v, ok := m[name]
	if !ok {
		return nil, false
	}
	if len(path) == 1 {
		return v, ok
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	return readPropertyFrom(obj, path[1:])
}
