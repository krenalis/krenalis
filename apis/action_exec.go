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
	"strings"
	"time"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/state"
	_connector "chichi/connector"
	"chichi/connector/types"
)

var ExecutionInProgress errors.Code = "ExecutionInProgress"

// addExecution adds an execution to the action.
func (this *Action) addExecution(reimport bool) error {

	n := state.ExecuteActionNotification{
		Action:    this.action.ID,
		Reimport:  reimport,
		StartTime: time.Now().UTC(),
	}
	c := this.action.Connection()
	if storage, ok := c.Storage(); ok {
		n.Storage = storage.ID
	}

	ctx := context.Background()
	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
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
func (this *Action) exec() {

	connection := this.action.Connection()
	execution, _ := this.action.Execution()
	connector := connection.Connector()

	ctx := context.Background()

	var err error
	if this.Target == GroupsTarget {
		err = actionExecutionError{fmt.Errorf("groups import and export are not implemented")}
	} else {
		switch connector.Type {
		case state.AppType:
			if connection.Role == state.SourceRole {
				err = this.importFromApp()
			} else {
				err = this.exportUsersToApp(ctx)
			}
		case state.DatabaseType:
			err = this.importFromDatabase()
		case state.FileType:
			if connection.Role == state.SourceRole {
				err = this.importFromFile()
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
		if e, ok := err.(actionExecutionError); ok {
			errorMessage = abbreviate(e.Error(), 1000)
			if _, ok := e.err.(*_connector.AccessDeniedError); ok {
				health = state.AccessDenied
			}
		} else {
			log.Printf("[error] cannot execute action %d, execution %d failed: %s", this.action.ID, execution.ID, err)
			errorMessage = "an internal error has occurred"
		}
	}

	n := state.EndActionExecutionNotification{
		ID:     execution.ID,
		Health: health,
	}

	// TODO(marco) retry if the transaction fails.
	err = this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE actions_executions SET end_time = $1, error = $2 WHERE id = $3",
			endTime, errorMessage, n.ID)
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
		log.Printf("[error] cannot update the status of the execution %d of action %d: %s",
			execution.ID, this.action.ID, err)
	}

}

// schema returns the schema and the paths of the mapped properties of
// the connection.
func (this *Action) schema() (types.Type, []types.Path, error) {

	// Collect the paths of the properties used in transformation or mappings.
	var paths []types.Path
	if t := this.action.Transformation; t != nil {
		for _, name := range t.In.PropertiesNames() {
			paths = append(paths, []string{name})
		}
	}
	for _, left := range this.action.Mapping {
		paths = append(paths, strings.Split(left, "."))
	}

	// Create a schema with only the properties mapped.
	mapped := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		mapped[p[0]] = struct{}{}
	}
	mappedProperties := make([]types.Property, 0, len(paths))
	schema := this.action.Schema
	for _, property := range schema.Properties() {
		if _, ok := mapped[property.Name]; ok {
			mappedProperties = append(mappedProperties, property)
		}
	}
	if len(mappedProperties) > 0 {
		schema = types.Object(mappedProperties)
	}

	return schema, paths, nil
}

// actionExecutionError represents a non-internal error during action execution.
type actionExecutionError struct {
	err error
}

func (err actionExecutionError) Error() string {
	return err.err.Error()
}

// applyTimestampWorkaround applies a workaround on the "timestamp" property
// until the normalization / conversion is implemented.
// TODO(Gianluca): see https://github.com/open2b/chichi/issues/186.
func applyTimestampWorkaround(mappedUser map[string]any) error {
	if ts, ok := mappedUser["timestamp"].(string); ok {
		t, err := time.Parse(time.DateTime, ts)
		if err != nil {
			return fmt.Errorf("bad timestamp %q parsed: %s", ts, err)
		}
		mappedUser["timestamp"] = t
	}
	return nil
}
