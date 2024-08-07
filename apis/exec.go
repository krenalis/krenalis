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
	"time"

	"github.com/meergo/meergo/apis/datastore"
	"github.com/meergo/meergo/apis/errors"
	"github.com/meergo/meergo/apis/postgres"
	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/apis/statistics"
)

// actionError represents an action error.
type actionError struct {
	step statistics.Step
	err  error
}

func newActionError(step statistics.Step, err error) *actionError {
	return &actionError{step, err}
}

func (err actionError) Error() string {
	return err.err.Error()
}

// addExecution adds an execution to the action.
func (this *Action) addExecution(ctx context.Context, reimport bool) error {

	n := state.ExecuteAction{
		Action:    this.action.ID,
		Reimport:  reimport,
		StartTime: time.Now().UTC(),
	}
	c := this.action.Connection()
	if c.Connector().Type == state.FileStorage {
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

	execution, _ := this.action.Execution()
	timeSlot := statistics.TimeSlotFromTime(execution.StartTime)
	statCollector := this.apis.statistics.Collector(this.action.ID)

	var err error
	var errorStep statistics.Step
	var errorMessage string
	var actionImportedUsers bool

	defer func() {

		if err != nil {
			if err, ok := err.(*actionError); ok {
				errorStep = err.step
				errorMessage = err.Error()
				statCollector.FailedStep(errorStep, 0, errorMessage)
			} else {
				select {
				case <-ctx.Done():
					statCollector.FailedReceiving(0, "execution has been cancelled")
				default:
					statCollector.FailedReceiving(0, "an internal error has occurred")
					slog.Error("cannot execute action", "action", this.action.ID, "execution", execution.ID, "err", err)
				}
			}
		}

		statCollector.Close()
		endTime := time.Now().UTC()

		n := state.EndActionExecution{
			ID:     execution.ID,
			Health: state.Healthy,
		}

		// TODO(marco) retry if the transaction fails.
		err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
			_, err := tx.Exec(ctx,
				"WITH stats AS (\n"+
					"	SELECT COALESCE(SUM(passed_5), 0) as passed, COALESCE(SUM(failed_0 + failed_1 + failed_2 + failed_3 + failed_4 + failed_5), 0) as failed\n"+
					"	FROM actions_stats\n"+
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

		// Run the Identity Resolution, if necessary.
		ws := this.action.Connection().Workspace()
		if actionImportedUsers && ws.RunIdentityResolutionOnBatchImport {
			err = this.connection.store.RunIdentityResolution(ctx)
			if err != nil && err != datastore.ErrIdentityResolutionInProgress {
				slog.Error("error while running the Identity Resolution at the end of user import", "err", err)
				return
			}
		}

	}()

	connection := this.action.Connection()

	if this.Target == Groups {
		statCollector.FailedReceiving(0, "groups import and export are not implemented")
		return
	}
	if !this.isLanguageSupported() {
		statCollector.FailedReceiving(0, fmt.Sprintf("%s transformation language is not supported", this.Transformation.Function.Language))
		return
	}
	if this.connection.store == nil {
		statCollector.FailedReceiving(0, fmt.Sprintf("workspace %d does not have a data warehouse", connection.Workspace().ID))
		return
	}

	_, err = this.apis.db.Exec(ctx,
		"WITH stats AS (\n"+
			"	SELECT -passed_5 as passed, -(failed_0 + failed_1 + failed_2 + failed_3 + failed_4 + failed_5) as failed\n"+
			"	FROM actions_stats\n"+
			"	WHERE action = $2 AND timeslot = $3\n"+
			")\n"+
			"UPDATE actions_executions\n"+
			"SET passed = stats.passed, failed = stats.failed\n"+
			"FROM stats\n"+
			"WHERE id = $1", execution.ID, this.action.ID, timeSlot)
	if err != nil {
		return
	}

	if connection.Role == state.Source {
		err = this.importUsers(ctx, statCollector)
		actionImportedUsers = true
	} else {
		err = this.exportUsers(ctx, statCollector)
	}

	return
}
