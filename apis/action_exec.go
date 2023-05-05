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
	"log"
	"strconv"
	"strings"
	"time"

	"chichi/apis/errors"
	"chichi/apis/mappings"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/apis/types"
	_connector "chichi/connector"
)

const (
	identityLabel  = "identity"
	timestampLabel = "timestamp"
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

	var err error
	if this.Target == GroupsTarget {
		err = actionExecutionError{fmt.Errorf("groups import and export are not implemented")}
	} else {
		switch connector.Type {
		case state.AppType:
			if connection.Role == state.SourceRole {
				err = this.importFromApp()
			} else {
				err = this.exportToApp()
			}
		case state.DatabaseType:
			err = this.importFromDatabase()
		case state.FileType:
			err = this.importFromFile()
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
	ctx := context.Background()
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
func (this *Action) schema() (types.Type, []_connector.PropertyPath, error) {

	// Collect the paths of the properties used in transformation or mappings.
	var paths []_connector.PropertyPath
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

// setUser sets a user. id is the identifier of the user for the connector, user
// contains the values of the properties to be set, and timestamps are the dates
// of the last modification of the properties. If a user with the identifier id
// does not exist, it is created.
func (this *Action) setUser(id string, user map[string]any, timestamps map[string]time.Time) error {

	c := this.action.Connection()

	ctx := context.Background()

	// Write the user properties to the database.
	err := this.writeConnectionUsers(ctx, c.ID, id, user, timestamps)
	if err != nil {
		return err
	}

	// Apply the mapping (or the transformation).
	candidateData, err := mappings.Apply(ctx, this.action, user, types.Type{})
	if err != nil {
		return fmt.Errorf("cannot apply mapping or transformation: %s", err)
	}

	// Resolve the entity of this user.
	ids := identitySolver{ctx, c}
	email, _ := candidateData["Email"].(string)
	if email == "" {
		return fmt.Errorf("expecting 'Email' to be a non-empty string, got %#v (of type %T)", candidateData["Email"], candidateData["Email"])
	}
	goldenRecordID, err := ids.ResolveEntity(c.ID, id, email)
	if err != nil {
		return err
	}

	// Write the data to the Golden Record, if necessary.
	if len(candidateData) > 0 {
		err = this.writeToGoldenRecord(ctx, goldenRecordID, candidateData)
		if err != nil {
			return err
		}
		log.Printf("[info] properties for user %q written to the Golden Record", candidateData["Email"])
	}

	return nil
}

// writeConnectionUsers writes the given connection users to the database.
func (this *Action) writeConnectionUsers(ctx context.Context, connection int, id string, user map[string]any, timestamps map[string]time.Time) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	jsonTimestamps, err := json.Marshal(timestamps)
	if err != nil {
		return err
	}
	ws := this.action.Connection().Workspace()
	_, err = ws.Warehouse.Exec(ctx, "INSERT INTO connections_users (connection, \"user\", data, timestamps)\n"+
		"VALUES ($1, $2, $3, $4)\n"+
		"ON CONFLICT (connection, \"user\") DO UPDATE SET data = $3, timestamps = $4",
		connection, id, data, jsonTimestamps)
	if err != nil {
		return err
	}
	_, err = this.db.Exec(ctx, "INSERT INTO connections_stats AS cs (connection, time_slot, users)\n"+
		"VALUES ($1, $2, 1)\n"+
		"ON CONFLICT (connection, time_slot) DO UPDATE SET users = cs.users + 1",
		connection, statsTimeSlot(time.Now()))
	return err
}

// writeToGoldenRecord writes the given properties to the Golden Record.
func (this *Action) writeToGoldenRecord(ctx context.Context, id int, props map[string]any) error {

	// TODO(Gianluca):
	for _, v := range props {
		if _, ok := v.(map[string]interface{}); ok {
			return errors.New("writeToGoldenRecord is still partially implemented and does not support objects")
		}
	}

	query := &strings.Builder{}
	query.WriteString("UPDATE users SET\n")
	var values []any
	i := 1
	for prop, value := range props {
		if i > 1 {
			query.WriteString(", ")
		}
		query.WriteString(postgres.QuoteIdent(prop))
		query.WriteString(" = $")
		query.WriteString(strconv.Itoa(i))
		values = append(values, value)
		i++
	}
	query.WriteString(`, "updateTime" = now()`)
	query.WriteString("\nWHERE id = $")
	query.WriteString(strconv.Itoa(i))
	values = append(values, id)
	ws := this.action.Connection().Workspace()
	res, err := ws.Warehouse.Exec(ctx, query.String(), values...)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("BUG: one row should be affected, got %d", affected)
	}
	return nil
}

// readGRUsers reads the Golden Record users with the given IDs.
func (this *Action) readGRUsers(ids []int) ([]map[string]any, error) {
	return nil, nil // TODO(Gianluca): implement.
}

// newFirehoseForConnection returns a new Firehose for the connection c.
func (this *Action) newFirehoseForConnection(ctx context.Context, c *state.Connection) *firehose {
	var resource int
	if r, ok := c.Resource(); ok {
		resource = r.ID
	}
	fh := &firehose{
		db:         this.db,
		connection: c,
		resource:   resource,
	}
	fh.ctx, fh.cancel = context.WithCancel(ctx)
	return fh
}

// newFirehose returns a new Firehose for the action.
func (this *Action) newFirehose(ctx context.Context) *firehose {
	var resource int
	if r, ok := this.action.Connection().Resource(); ok {
		resource = r.ID
	}
	fh := &firehose{
		db:         this.db,
		action:     this,
		connection: this.action.Connection(),
		resource:   resource,
	}
	fh.ctx, fh.cancel = context.WithCancel(ctx)
	return fh
}

// actionExecutionError represents a non-internal error during action execution.
type actionExecutionError struct {
	err error
}

func (err actionExecutionError) Error() string {
	return err.err.Error()
}
