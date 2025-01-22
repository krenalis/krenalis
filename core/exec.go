//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package core

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/metrics"
	"github.com/meergo/meergo/core/postgres"
	"github.com/meergo/meergo/core/state"
)

// actionError represents an action error.
type actionError struct {
	step metrics.Step
	err  error
}

func newActionError(step metrics.Step, err error) *actionError {
	return &actionError{step, err}
}

func (err actionError) Error() string {
	return err.err.Error()
}

// addExecution adds an execution to the action and returns its identifier.
//
// It returns an errors.NotFoundError error if the action does not exist
// anymore.
// It returns an errors.UnprocessableError error with code ExecutionInProgress
// if the action is already in progress.
func (this *Action) addExecution(ctx context.Context, reload bool) (int, error) {

	n := state.ExecuteAction{
		Action:    this.action.ID,
		Reload:    reload,
		StartTime: time.Now().UTC(),
	}

	err := this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		var cursor time.Time
		var executing bool
		err := tx.QueryRow(ctx, "SELECT a.reload, COALESCE(e.cursor, '0001-01-01 00:00:00+00'), e.id IS NOT NULL AND e.end_time IS NULL\n"+
			"FROM actions AS a\n"+
			"LEFT JOIN actions_executions AS e ON a.id = e.action\n"+
			"WHERE a.id = $1\n"+
			"ORDER BY e.id DESC\n"+
			"LIMIT 1", n.Action).Scan(&reload, &cursor, &executing)
		if err != nil {
			if err == sql.ErrNoRows {
				return errors.NotFound("action %d does not exist", n.Action)
			}
			return err
		}
		if executing {
			return errors.Unprocessable(ExecutionInProgress, "execution of action %d is in progress", this.action.ID)
		}
		if reload {
			_, err = tx.Exec(ctx, "UPDATE actions SET reload = FALSE WHERE id = $1", n.Action)
			if err != nil {
				return err
			}
		}
		n.Reload = n.Reload || reload
		if !n.Reload {
			n.Cursor = cursor
		}
		err = tx.QueryRow(ctx, "INSERT INTO actions_executions (action, cursor, reload, start_time)\n"+
			"VALUES ($1, $2, $3, $4)\nRETURNING id", n.Action, n.Cursor, n.Reload, n.StartTime).Scan(&n.ID)
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
	if err != nil {
		return 0, err
	}

	return n.ID, nil
}

// exec executes the action.
//
// It is called in its own goroutine and the action have an execution to
// execute. In case of error, it writes the error with the execution status in
// the actions_executions table.
func (this *Action) exec(ctx context.Context) {

	execution, _ := this.action.Execution()
	timeSlot := metrics.TimeSlotFromTime(execution.StartTime)

	var err error
	var errorStep metrics.Step
	var errorMessage string
	var actionImportedUsers bool

	defer func() {

		if err != nil {
			if actionErr, ok := err.(*actionError); ok {
				errorStep = actionErr.step
				errorMessage = err.Error()
				this.core.metrics.Failed(errorStep, this.action.ID, 0, errorMessage)
			} else {
				select {
				case <-ctx.Done():
					this.core.metrics.ReceiveFailed(this.action.ID, 0, "execution has been cancelled")
				default:
					this.core.metrics.ReceiveFailed(this.action.ID, 0, "an internal error has occurred")
					slog.Error("cannot execute action", "action", this.action.ID, "execution", execution.ID, "err", err)
				}
			}
		}

		this.core.metrics.WaitStore()
		endTime := time.Now().UTC()

		n := state.EndActionExecution{
			ID:     execution.ID,
			Action: this.action.ID,
			Health: state.Healthy,
		}

		// TODO(marco) retry if the transaction fails.
		err := this.core.state.Transaction(ctx, func(tx *state.Tx) error {
			_, err := tx.Exec(ctx,
				"WITH stats AS (\n"+
					"	SELECT COALESCE(SUM(passed_5), 0) as passed, COALESCE(SUM(failed_0 + failed_1 + failed_2 + failed_3 + failed_4 + failed_5), 0) as failed\n"+
					"	FROM actions_metrics\n"+
					"	WHERE action = $2 AND timeslot >= $3\n"+
					")\n"+
					"UPDATE actions_executions AS e\n"+
					"SET end_time = $4, passed = e.passed + stats.passed, failed = e.failed + stats.failed, error_step = $5, error_message = $6\n"+
					"FROM stats\n"+
					"WHERE id = $1", n.ID, this.action.ID, timeSlot, endTime, errorStep, errorMessage)
			if err != nil {
				return err
			}
			var exists bool
			err = tx.QueryRow(ctx, "UPDATE actions SET health = $1 WHERE id = $2 RETURNING true",
				n.Health, this.action.ID).Scan(&exists)
			if err != nil {
				if err == sql.ErrNoRows {
					// The action does not exist anymore.
					return nil
				}
				return err
			}
			return tx.Notify(ctx, n)
		})
		if err != nil {
			slog.Error("cannot update action execution status",
				"action", this.action.ID,
				"execution", execution.ID,
				"err", err,
			)
		}

		// Start the Identity Resolution, if necessary.
		ws := this.action.Connection().Workspace()
		if actionImportedUsers && ws.ResolveIdentitiesOnBatchImport {
			err = this.connection.store.StartIdentityResolution(ctx)
			if err != nil {
				slog.Error("cannot start Identity Resolution at the end of import", "action", this.action.ID, "execution", execution.ID, "err", err)
				return
			}
		}

	}()

	if this.Target == Groups {
		this.core.metrics.ReceiveFailed(this.action.ID, 0, "groups import and export are not implemented")
		return
	}
	if !this.isLanguageSupported() {
		this.core.metrics.ReceiveFailed(this.action.ID, 0, fmt.Sprintf("%s transformation language is not supported", this.Transformation.Function.Language))
		return
	}

	_, err = this.core.db.Exec(ctx,
		"WITH stats AS (\n"+
			"	SELECT -passed_5 as passed, -(failed_0 + failed_1 + failed_2 + failed_3 + failed_4 + failed_5) as failed\n"+
			"	FROM actions_metrics\n"+
			"	WHERE action = $2 AND timeslot = $3\n"+
			")\n"+
			"UPDATE actions_executions\n"+
			"SET passed = stats.passed, failed = stats.failed\n"+
			"FROM stats\n"+
			"WHERE id = $1", execution.ID, this.action.ID, timeSlot)
	if err != nil {
		return
	}

	if this.action.Connection().Role == state.Source {
		err = this.importUsers(ctx)
		actionImportedUsers = true
	} else {
		err = this.exportUsers(ctx)
	}

}
