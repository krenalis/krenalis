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
	"encoding/json"
	"fmt"
	"log/slog"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"chichi"
	"chichi/apis/datastore/expr"
	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const ExecutionInProgress errors.Code = "ExecutionInProgress"

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
			err = this.exportUsers(ctx)
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
				errorMessage = errors.Abbreviate(e.Error(), 1000)
				if _, ok := e.err.(*chichi.AccessDeniedError); ok {
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

// convertActionFilterToExpr converts a well-formed action filter to an
// expr.Expr expression. schema defines the types of properties referenced
// within the filter.
//
// Take in sync with the convertFilterToExpr function.
func convertActionFilterToExpr(filter *state.Filter, schema types.Type) (expr.Expr, error) {
	op := expr.LogicalOperatorAnd
	if filter.Logical == "any" {
		op = expr.LogicalOperatorOr
	}
	exp := expr.NewMultiExpr(op, make([]expr.Expr, len(filter.Conditions)))
	for i, cond := range filter.Conditions {
		property, err := schema.PropertyByPath(strings.Split(cond.Property, "."))
		if err != nil {
			return nil, fmt.Errorf("property path %s does not exist", cond.Property)
		}
		var op expr.Operator
		switch cond.Operator {
		case "is":
			op = expr.OperatorEqual
		case "is not":
			op = expr.OperatorNotEqual
		default:
			return nil, errors.New("invalid operator")
		}
		var value any
		switch property.Type.Kind() {
		case types.BooleanKind:
			value = false
			if cond.Value == "true" {
				value = true
			}
		case types.IntKind:
			value, _ = strconv.ParseInt(cond.Value, 10, 64)
		case types.UintKind:
			value, _ = strconv.ParseUint(cond.Value, 10, 64)
		case types.FloatKind:
			value, _ = strconv.ParseFloat(cond.Value, 64)
		case types.DecimalKind:
			value = decimal.RequireFromString(cond.Value)
		case types.DateTimeKind:
			value, _ = time.Parse(time.DateTime, cond.Value)
		case types.DateKind:
			value, _ = time.Parse(time.DateOnly, cond.Value)
		case types.TimeKind:
			value, _ = time.Parse("15:04:05.999999999", cond.Value)
		case types.YearKind:
			value, _ = strconv.Atoi(cond.Value)
		case types.UUIDKind:
			value, _ = uuid.Parse(cond.Value)
		case types.JSONKind:
			value = json.RawMessage(cond.Value)
		case types.InetKind:
			value, _ = netip.ParseAddr(cond.Value)
		case types.TextKind:
			value = cond.Value
		default:
			return nil, fmt.Errorf("unexpected type %s", property.Type)
		}
		exp.Operands[i] = expr.NewBaseExpr(cond.Property, op, value)
	}
	return exp, nil
}
