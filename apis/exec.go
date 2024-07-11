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

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/apis/errors"
	"github.com/meergo/meergo/apis/postgres"
	"github.com/meergo/meergo/apis/state"
)

// addExecution adds an execution to the action.
func (this *Action) addExecution(ctx context.Context, reimport bool) error {

	n := state.ExecuteAction{
		Action:    this.action.ID,
		Reimport:  reimport,
		StartTime: time.Now().UTC(),
	}
	c := this.action.Connection()
	if c.Connector().Type == state.FileStorageType {
		n.Storage = c.ID
	}

	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
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

	var err error
	var passed, failed [6]int

	if this.Target == Groups {
		err = actionExecutionError{fmt.Errorf("groups import and export are not implemented")}
	} else if !this.isLanguageSupported() {
		err = actionExecutionError{fmt.Errorf("%s transformation language is not supported", this.Transformation.Function.Language)}
	} else if this.connection.store == nil {
		err = actionExecutionError{fmt.Errorf("workspace %d does not have a data warehouse", connection.Workspace().ID)}
	} else {
		stats := this.apis.statistics.Action(this.action.ID)
		if connection.Role == state.Source {
			err = this.importUsers(ctx, stats)
		} else {
			err = this.exportUsers(ctx, stats)
		}
		passed, failed = stats.Stats()
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
				errorMessage = errors.Abbreviate(e.Error(), 1000)
				if _, ok := e.err.(*meergo.AccessDeniedError); ok {
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
	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		_, err := tx.Exec(txCtx, "UPDATE actions_executions SET end_time = $1, passed = $2, failed = $3, error = $4 WHERE id = $5",
			endTime, passed, failed, errorMessage, n.ID)
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

// actionExecutionError represents a non-internal error during action execution.
type actionExecutionError struct {
	err error
}

func (err actionExecutionError) Error() string {
	return err.err.Error()
}

// readPropertyFrom reads the property with the given path from m, returning its
// value (if found, otherwise nil) and a boolean indicating if the property path
// corresponds to a value in m or not.
func readPropertyFrom(m map[string]any, path string) (any, bool) {
	var name string
	for {
		name, path, _ = strings.Cut(path, ".")
		v, ok := m[name]
		if !ok {
			return nil, false
		}
		if path == "" {
			return v, true
		}
		m, ok = v.(map[string]any)
		if !ok {
			return nil, false
		}
	}
}
