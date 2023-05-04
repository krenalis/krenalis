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
	"net/netip"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/apis/types"
	_connector "chichi/connector"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	identityLabel  = "identity"
	timestampLabel = "timestamp"
)

var ExecutionInProgress errors.Code = "ExecutionInProgress"

// addExecution adds an execution to the action.
func (ac *Action) addExecution(reimport bool) error {

	n := state.ExecuteActionNotification{
		Action:    ac.action.ID,
		Reimport:  reimport,
		StartTime: time.Now().UTC(),
	}
	c := ac.action.Connection()
	if storage, ok := c.Storage(); ok {
		n.Storage = storage.ID
	}

	ctx := context.Background()
	err := ac.db.Transaction(ctx, func(tx *postgres.Tx) error {
		err := tx.QueryVoid(ctx, "SELECT FROM actions_executions WHERE action = $1 AND end_time IS NULL", n.Action)
		if err != sql.ErrNoRows {
			if err == nil {
				err = errors.Unprocessable(ExecutionInProgress, "execution of action %d is in progress", ac.action.ID)
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
func (ac *Action) exec() {

	connection := ac.action.Connection()
	execution, _ := ac.action.Execution()
	connector := connection.Connector()

	var err error
	if ac.Target == GroupsTarget {
		err = actionExecutionError{fmt.Errorf("groups import and export are not implemented")}
	} else {
		switch connector.Type {
		case state.DatabaseType:
			err = ac.importFromDatabase()
		case state.FileType:
			err = ac.importFromFile()
		default:
			if connection.Role == state.SourceRole {
				err = ac.importUsers()
			} else {
				err = ac.exportUsers()
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
			log.Printf("[error] cannot execute action %d, execution %d failed: %s", ac.action.ID, execution.ID, err)
			errorMessage = "an internal error has occurred"
		}
	}

	n := state.EndActionExecutionNotification{
		ID:     execution.ID,
		Health: health,
	}

	// TODO(marco) retry if the transaction fails.
	ctx := context.Background()
	err = ac.db.Transaction(ctx, func(tx *postgres.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE actions_executions SET end_time = $1, error = $2 WHERE id = $3",
			endTime, errorMessage, n.ID)
		if err != nil {
			return err
		}
		var exists bool
		err = tx.QueryRow(ctx, "UPDATE actions SET health = $1 WHERE id = $2 RETURNING true",
			n.Health, ac.action.ID).Scan(&exists)
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
			execution.ID, ac.action.ID, err)
	}

}

// exportUsers exports the users of action.
// Note that this method is only a draft, and its code may be wrong and/or
// partially implemented.
func (ac *Action) exportUsers() error {

	const role = _connector.SourceRole

	connection := ac.action.Connection()
	connector := connection.Connector()

	ctx := context.Background()

	switch connector.Type {
	case state.AppType:

		var name, clientSecret, resourceCode, accessToken, refreshToken string
		var webhooksPer WebhooksPer
		var connector, resource int
		var settings []byte
		var expiration time.Time
		err := ac.db.QueryRow(ctx,
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

		fh := ac.newFirehose(context.Background())
		ws := ac.action.Connection().Workspace()
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
			rows, err := ac.db.Query(ctx, "SELECT user, goldenRecord FROM connection_users WHERE connection = $1", connection.ID)
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
			grUsers, err := ac.readGRUsers(toRead)
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
func (ac *Action) importUsers() error {

	const role = _connector.SourceRole

	connection := ac.action.Connection()
	connector := connection.Connector()
	execution, _ := ac.action.Execution()

	switch connector.Type {
	case state.AppType:

		var clientSecret, resourceCode, accessToken string
		if r, ok := connection.Resource(); ok {
			clientSecret = connector.OAuth.ClientSecret
			resourceCode = r.Code
			var err error
			accessToken, err = freshAccessToken(ac.db, r)
			if err != nil {
				return actionExecutionError{fmt.Errorf("cannot retrive the OAuth access token: %s", err)}
			}
		}

		// Read the properties to read.
		_, properties, err := ac.schema()
		if err != nil {
			return fmt.Errorf("cannot read user schema: %s", err)
		}

		fh := ac.newFirehose(context.Background())
		ws := ac.action.Connection().Workspace()
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
func (ac *Action) schema() (types.Type, []_connector.PropertyPath, error) {

	// Collect the paths of the properties used in transformation or mappings.
	var paths []_connector.PropertyPath
	if t := ac.action.Transformation; t != nil {
		for _, name := range t.In.PropertiesNames() {
			paths = append(paths, []string{name})
		}
	}
	for _, left := range ac.action.Mapping {
		paths = append(paths, strings.Split(left, "."))
	}

	// Create a schema with only the properties mapped.
	mapped := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		mapped[p[0]] = struct{}{}
	}
	mappedProperties := make([]types.Property, 0, len(paths))
	schema := ac.action.Schema
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

// readGRUsers reads the Golden Record users with the given IDs.
func (ac *Action) readGRUsers(ids []int) ([]map[string]any, error) {
	return nil, nil // TODO(Gianluca): implement.
}

// newFirehoseForConnection returns a new Firehose for the connection c.
func (ac *Action) newFirehoseForConnection(ctx context.Context, c *state.Connection) *firehose {
	var resource int
	if r, ok := c.Resource(); ok {
		resource = r.ID
	}
	fh := &firehose{
		db:         ac.db,
		connection: c,
		resource:   resource,
	}
	fh.ctx, fh.cancel = context.WithCancel(ctx)
	return fh
}

// newFirehose returns a new Firehose for the action.
func (ac *Action) newFirehose(ctx context.Context) *firehose {
	var resource int
	if r, ok := ac.action.Connection().Resource(); ok {
		resource = r.ID
	}
	fh := &firehose{
		db:         ac.db,
		action:     ac.action,
		connection: ac.action.Connection(),
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

// normalizePropertyValue normalizes a property value returned by a database or
// file connector, and returns its normalized value. If the value is not valid
// it returns an error.
func normalizePropertyValue(property types.Property, src any) (any, error) {
	name := property.Name
	if src == nil {
		if !property.Nullable {
			return nil, fmt.Errorf("column %s is non-nullable, but the database returned a NULL value", name)
		}
		return nil, nil
	}
	typ := property.Type
	var value any
	var valid bool
	switch typ.PhysicalType() {
	case types.PtBoolean:
		if _, ok := src.(bool); ok {
			value = src
			valid = true
		}
	case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64:
		var v int64
		switch src := src.(type) {
		case int32:
			v = int64(src)
			valid = true
		case int64:
			v = src
			valid = true
		case []byte:
			var err error
			v, err = strconv.ParseInt(string(src), 10, 64)
			valid = err == nil
		}
		if valid {
			min, max := typ.IntRange()
			if v < min || v > max {
				return nil, fmt.Errorf("database returnd a value of %d for column %s which is not within the expected range of [%d, %d]",
					v, name, min, max)
			}
			value = int(v)
		}
	case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
		var v uint64
		switch src := src.(type) {
		case []byte:
			var err error
			v, err = strconv.ParseUint(string(src), 10, 64)
			valid = err == nil
		}
		if valid {
			min, max := typ.UIntRange()
			if v < min || v > max {
				return nil, fmt.Errorf("database returnd a value of %d for column %s which is not within the expected range of [%d, %d]",
					v, name, min, max)
			}
			value = uint(v)
		}
	case types.PtFloat, types.PtFloat32:
		var v float64
		switch src := src.(type) {
		case float32:
			v = float64(src)
			valid = true
		case float64:
			v = src
			valid = true
		case []byte:
			var err error
			size := 64
			if typ.PhysicalType() == types.PtFloat32 {
				size = 32
			}
			v, err = strconv.ParseFloat(string(src), size)
			valid = err == nil
		}
		if valid {
			min, max := typ.FloatRange()
			if v < min || v > max {
				return nil, fmt.Errorf("database returnd a value of %f for column %s which is not within the expected range of [%f, %f]",
					v, name, min, max)
			}
			value = v
		}
	case types.PtDecimal:
		var v decimal.Decimal
		switch src := src.(type) {
		case string:
			var err error
			v, err = decimal.NewFromString(src)
			valid = err == nil
		case []byte:
			var err error
			v, err = decimal.NewFromString(string(src))
			valid = err == nil
		case int32:
			v = decimal.NewFromInt32(src)
			valid = true
		case int64:
			v = decimal.NewFromInt(src)
			valid = true
		}
		if valid {
			min, max := typ.DecimalRange()
			if v.LessThan(min) || v.GreaterThan(max) {
				return nil, fmt.Errorf("database returnd a value of %s for column %s which is not within the expected range of [%s, %s]",
					v, name, min, max)
			}
			value = v
		}
	case types.PtDateTime:
		if t, ok := src.(time.Time); ok {
			var err error
			value, err = _connector.AsDateTime(t)
			valid = err == nil
		}
	case types.PtDate:
		if t, ok := src.(time.Time); ok {
			var err error
			value, err = _connector.AsDate(t)
			valid = err == nil
		}
	case types.PtTime:
		if s, ok := src.([]byte); ok {
			var err error
			value, err = _connector.ParseTime(string(s))
			valid = err == nil
		}
	case types.PtYear:
		switch y := src.(type) {
		case int64:
			if valid = types.MinYear <= y && y <= types.MaxYear; valid {
				value = int(y)
			}
		case []byte:
			year, err := strconv.Atoi(string(y))
			value = year
			valid = err == nil && types.MinYear <= year && year <= types.MaxYear
		}
	case types.PtUUID:
		if s, ok := src.(string); ok {
			if v, err := uuid.Parse(s); err == nil {
				value = v.String()
				valid = true
			}
		}
	case types.PtJSON:
		if s, ok := src.([]byte); ok {
			if valid = json.Valid(s); valid {
				value = json.RawMessage(s)
			}
		}
	case types.PtInet:
		if s, ok := src.(string); ok {
			if v, err := netip.ParseAddr(s); err == nil {
				value = v.String()
				valid = true
			}
		}
	case types.PtText:
		var v string
		switch s := src.(type) {
		case string:
			v = s
			valid = true
		case []byte:
			v = string(s)
			valid = true
		}
		if valid {
			if !utf8.ValidString(v) {
				return nil, fmt.Errorf("database returned a value of %q for column %s, which does not contain valid UTF-8 characters",
					abbreviate(v, 20), name)
			}
			if l, ok := typ.ByteLen(); ok && len(v) > l {
				return nil, fmt.Errorf("database returned a value of %q for column %s, which is longer than %d bytes",
					abbreviate(v, 20), name, l)
			}
			if l, ok := typ.CharLen(); ok && utf8.RuneCountInString(v) > l {
				return nil, fmt.Errorf("database returned a value of %q for column %s, which is longer than %d characters",
					abbreviate(v, 20), name, l)
			}
			value = v
		}
	}
	if !valid {
		return nil, fmt.Errorf("database returned a value of %v for column %s, but it cannot be converted to the %s type",
			src, name, typ.PhysicalType())
	}
	return value, nil
}

// validateStringProperty validates a string property like
// normalizePropertyValue does.
func validateStringProperty(p types.Property, s string) error {
	if !utf8.ValidString(s) {
		return fmt.Errorf("database returned a value of %q for column %s, which does not contain valid UTF-8 characters",
			abbreviate(s, 20), p.Name)
	}
	if l, ok := p.Type.ByteLen(); ok && len(s) > l {
		return fmt.Errorf("database returned a value of %q for column %s, which is longer than %d bytes",
			abbreviate(s, 20), p.Name, l)
	}
	if l, ok := p.Type.CharLen(); ok && utf8.RuneCountInString(s) > l {
		return fmt.Errorf("database returned a value of %q for column %s, which is longer than %d characters",
			abbreviate(s, 20), p.Name, l)
	}
	return nil
}
