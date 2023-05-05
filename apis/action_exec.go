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
		case state.DatabaseType:
			err = this.importFromDatabase()
		case state.FileType:
			err = this.importFromFile()
		default:
			if connection.Role == state.SourceRole {
				err = this.importUsers()
			} else {
				err = this.exportUsers()
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

// exportUsers exports the users of action.
// Note that this method is only a draft, and its code may be wrong and/or
// partially implemented.
func (this *Action) exportUsers() error {

	const role = _connector.SourceRole

	connection := this.action.Connection()
	connector := connection.Connector()

	ctx := context.Background()

	switch connector.Type {
	case state.AppType:

		var name, clientSecret, resourceCode, accessToken, refreshToken string
		var webhooksPer WebhooksPer
		var connector, resource int
		var settings []byte
		var expiration time.Time
		err := this.db.QueryRow(ctx,
			"SELECT `c`.`name`, `c`.`oAuthClientSecret`, `c`.`webhooksPer` - 1, `r`.`code`,"+
				" `r`.`oAuthAccessToken`, `r`.`oAuthRefreshToken`, `r`.`oAuthExpiresIn`, `s`.`connector`,"+
				" `s`.`resource`, `s`.`settings`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"INNER JOIN `resources` AS `r` ON `r`.`id` = `s`.`resource`\n"+
				"WHERE `s`.`id` = ?", connection.ID).Scan(
			&name, &clientSecret, &webhooksPer, &resourceCode, &accessToken, &refreshToken, &expiration, &connector,
			&resource, &settings)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}

		fh := this.newFirehose(context.Background())
		ws := this.action.Connection().Workspace()

		c, err := _connector.RegisteredApp(name).Open(fh.ctx, &_connector.AppConfig{
			Role:          role,
			Settings:      settings,
			Firehose:      fh,
			ClientSecret:  clientSecret,
			Resource:      resourceCode,
			AccessToken:   accessToken,
			PrivacyRegion: _connector.PrivacyRegion(ws.PrivacyRegion),
		})
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
		}

		// Prepare the users to export to the connection.
		users := []_connector.User{}
		{
			// TODO(Gianluca): populate this map:
			internalToExternalID := map[int]string{}
			rows, err := this.db.Query(ctx, "SELECT user, goldenRecord FROM connection_users WHERE connection = $1", connection.ID)
			if err != nil {
				return err
			}
			defer rows.Close()
			toRead := []int{}
			for rows.Next() {
				var user string
				var goldenRecord int
				err := rows.Scan(&user, &goldenRecord)
				if err != nil {
					return err
				}
				toRead = append(toRead, goldenRecord)
			}
			if err := rows.Err(); err != nil {
				return err
			}
			// Read the users from the Golden Record and apply the
			// transformation functions on them.
			grUsers, err := this.readGRUsers(toRead)
			if err != nil {
				return err
			}
			for _, user := range grUsers {
				id := internalToExternalID[user["id"].(int)]
				user, err := exportUser(id, user)
				if err != nil {
					return err
				}
				users = append(users, user)
			}
		}

		// Export the users to the connection.
		log.Printf("[info] exporting %d user(s) to the connection %d", len(users), connection.ID)
		err = c.(_connector.AppUsersConnection).SetUsers(users)
		if err != nil {
			return errors.New("cannot export users")
		}

		// Handle errors occurred in the firehose.
		if fh.err != nil {
			return fh.err
		}

	default:

		panic(fmt.Sprintf("export to %q not implemented", connector.Type))

	}

	return nil
}

// importUsers imports the users for the action.
func (this *Action) importUsers() error {

	const role = _connector.SourceRole

	connection := this.action.Connection()
	connector := connection.Connector()
	execution, _ := this.action.Execution()

	switch connector.Type {
	case state.AppType:

		var clientSecret, resourceCode, accessToken string
		if r, ok := connection.Resource(); ok {
			clientSecret = connector.OAuth.ClientSecret
			resourceCode = r.Code
			var err error
			accessToken, err = freshAccessToken(this.db, r)
			if err != nil {
				return actionExecutionError{fmt.Errorf("cannot retrive the OAuth access token: %s", err)}
			}
		}

		// Read the properties to read.
		_, properties, err := this.schema()
		if err != nil {
			return fmt.Errorf("cannot read user schema: %s", err)
		}

		fh := this.newFirehose(context.Background())
		ws := this.action.Connection().Workspace()
		c, err := _connector.RegisteredApp(connector.Name).Open(fh.ctx, &_connector.AppConfig{
			Role:          role,
			Settings:      connection.Settings,
			Firehose:      fh,
			ClientSecret:  clientSecret,
			Resource:      resourceCode,
			AccessToken:   accessToken,
			PrivacyRegion: _connector.PrivacyRegion(ws.PrivacyRegion),
		})
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
		}
		cursor := connection.UserCursor
		if execution.Reimport {
			cursor = ""
		}
		err = c.(_connector.AppUsersConnection).Users(cursor, properties)
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot get users from the connector: %s", err)}
		}

		// Handle errors occurred in the firehose.
		if fh.err != nil {
			return fh.err
		}

	case state.StreamType:

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c, err := _connector.RegisteredStream(connector.Name).Open(ctx, &_connector.StreamConfig{
			Role:     role,
			Settings: connection.Settings,
		})
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
		}
		defer c.Close()
		event, ack, err := c.Receive()
		if err != nil {
			return err
		}
		ack()
		log.Printf("[info] received event: %s", event)

	}

	return nil
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
