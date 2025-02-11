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

// createExecution creates an execution for the action and returns its
// identifier.
//
// It returns an errors.NotFoundError error if the action does not exist
// anymore.
// It returns an errors.UnprocessableError error with code ExecutionInProgress
// if the action is already in progress.
func (this *Action) createExecution(ctx context.Context, incremental *bool) (int, error) {

	n := state.ExecuteAction{
		Action:    this.action.ID,
		StartTime: time.Now().UTC(),
	}

	err := this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		var inc, executing bool
		var cursor time.Time
		err := tx.QueryRow(ctx, "SELECT a.incremental, a.cursor, e.id IS NOT NULL AND e.end_time IS NULL\n"+
			"FROM actions AS a\n"+
			"LEFT JOIN actions_executions AS e ON a.id = e.action\n"+
			"WHERE a.id = $1\n"+
			"ORDER BY e.id DESC\n"+
			"LIMIT 1", n.Action).Scan(&inc, &cursor, &executing)
		if err != nil {
			if err == sql.ErrNoRows {
				return errors.NotFound("action %d does not exist", n.Action)
			}
			return err
		}
		if executing {
			return errors.Unprocessable(ExecutionInProgress, "execution of action %d is in progress", this.action.ID)
		}
		if incremental == nil {
			n.Incremental = inc
		} else {
			n.Incremental = *incremental
		}
		if n.Incremental {
			n.Cursor = cursor
		}
		err = tx.QueryRow(ctx, "INSERT INTO actions_executions (action, cursor, incremental, start_time)\n"+
			"VALUES ($1, $2, $3, $4)\nRETURNING id", n.Action, n.Cursor, n.Incremental, n.StartTime).Scan(&n.ID)
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
				"WITH s AS (\n"+
					"\tSELECT COALESCE(SUM(passed_0), 0) as passed_0, COALESCE(SUM(passed_1), 0) as passed_1, COALESCE(SUM(passed_2), 0) as passed_2,"+
					" COALESCE(SUM(passed_3), 0) as passed_3, COALESCE(SUM(passed_4), 0) as passed_4, COALESCE(SUM(passed_5), 0) as passed_5,"+
					" COALESCE(SUM(failed_0), 0) as failed_0, COALESCE(SUM(failed_1), 0) as failed_1, COALESCE(SUM(failed_2), 0) as failed_2,"+
					" COALESCE(SUM(failed_3), 0) as failed_3, COALESCE(SUM(failed_4), 0) as failed_4, COALESCE(SUM(failed_5), 0) as failed_5\n"+
					" FROM actions_metrics\n"+
					"\tWHERE action = $2 AND timeslot >= $3\n"+
					")\n"+
					"UPDATE actions_executions AS e\n"+
					"SET end_time = $4, passed_0 = e.passed_0 + s.passed_0, passed_1 = e.passed_1 + s.passed_1, passed_2 = e.passed_2 + s.passed_2,"+
					" passed_3 = e.passed_3 + s.passed_3, passed_4 = e.passed_4 + s.passed_4, passed_5 = e.passed_5 + s.passed_5,"+
					" failed_0 = e.failed_0 + s.failed_0, failed_1 = e.failed_1 + s.failed_1, failed_2 = e.failed_2 + s.failed_2,"+
					" failed_3 = e.failed_3 + s.failed_3, failed_4 = e.failed_4 + s.failed_4, failed_5 = e.failed_5 + s.failed_5,"+
					" error_step = $5, error_message = $6\n"+
					"FROM s\n"+
					"WHERE id = $1", n.ID, this.action.ID, timeSlot, endTime, errorStep, errorMessage)
			if err != nil {
				return err
			}
			var exists bool
			err = tx.QueryRow(ctx, "UPDATE actions SET cursor = (SELECT cursor FROM actions_executions WHERE id = $1 LIMIT 1), health = $2 WHERE id = $3 RETURNING true",
				n.ID, n.Health, this.action.ID).Scan(&exists)
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
		"WITH s AS (\n"+
			"	SELECT -passed_0 as passed_0, -passed_1 as passed_1, -passed_2 as passed_2, -passed_3 as passed_3,"+
			" -passed_4 as passed_4, -passed_5 as passed_5, -failed_0 as failed_0, -failed_1 as failed_1,"+
			" -failed_2 as failed_2, -failed_3 as failed_3, -failed_4 as failed_4, -failed_5 as failed_5\n"+
			"	FROM actions_metrics\n"+
			"	WHERE action = $2 AND timeslot = $3\n"+
			")\n"+
			"UPDATE actions_executions\n"+
			"SET passed_0 = s.passed_0, passed_1 = s.passed_1, passed_2 = s.passed_2, passed_3 = s.passed_3,"+
			" passed_4 = s.passed_4, passed_5 = s.passed_5, failed_0 = s.failed_0, failed_1 = s.failed_1,"+
			" failed_2 = s.failed_2, failed_3 = s.failed_3, failed_4 = s.failed_4, failed_5 = s.failed_5\n"+
			"FROM s\n"+
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
