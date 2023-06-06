//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	mathrand "math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/errors"
	"chichi/apis/events"
	"chichi/apis/httpclient"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/apis/warehouses"
	_connector "chichi/connector"
	"chichi/connector/types"
	"chichi/connector/ui"

	"github.com/jxskiss/base62"
	"golang.org/x/exp/slices"
)

const (
	maxKeysPerServer = 20 // maximum number of keys per server.
	maxInt32         = math.MaxInt32
	rawSchemaMaxSize = 16_777_215 // maximum size in runes of the 'schema' column of the 'connections' table.
	queryMaxSize     = 16_777_215 // maximum size in runes of a connection query.
)

var (
	ConnectorNotExists  errors.Code = "ConnectorNotExists"
	EventNotExists      errors.Code = "EventNotExists"
	EventTypeNotExists  errors.Code = "EventTypeNotExists"
	FetchSchemaFailed   errors.Code = "FetchSchemaFailed"
	InvalidRefreshToken errors.Code = "InvalidRefreshToken"
	KeyNotExists        errors.Code = "KeyNotExists"
	NoGroupsSchema      errors.Code = "NoGroupsSchema"
	NoStorage           errors.Code = "NoStorage"
	NoUsersSchema       errors.Code = "NoUsersSchema"
	ReadFileFailed      errors.Code = "ReadFileFailed"
	StorageNotExists    errors.Code = "StorageNotExists"
	TargetAlreadyExists errors.Code = "TargetAlreadyExists"
	TooManyKeys         errors.Code = "TooManyKeys"
	UniqueKey           errors.Code = "UniqueKey"
	WorkspaceNotExists  errors.Code = "WorkspaceNotExists"
)

// Connection represents a connection.
type Connection struct {
	db          *postgres.DB
	connection  *state.Connection
	http        *httpclient.HTTP
	ID          int
	Name        string
	Type        ConnectorType
	Role        ConnectionRole
	Connector   int
	Storage     int // zero if the connection is not a file or does not have a storage.
	HasSettings bool
	Enabled     bool
	Health      Health
}

// Action returns the action with identifier id of the connection.
// It returns an errors.NotFound error if the action does not exist.
func (this *Connection) Action(id int) (*Action, error) {
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid action identifier", id)
	}
	a, ok := this.connection.Action(id)
	if !ok {
		return nil, errors.NotFound("action %d does not exist", id)
	}
	var action Action
	action.fromState(this.db, this.http, a)
	return &action, nil
}

// Actions returns the actions of the connection.
func (this *Connection) Actions() ([]Action, error) {
	as := this.connection.Actions()
	actions := make([]Action, len(as))
	for i, a := range as {
		actions[i].fromState(this.db, this.http, a)
	}
	return actions, nil
}

// ActionType represents an action type.
type ActionType struct {
	Name          string
	Description   string
	Target        ActionTarget
	EventType     *string
	MissingSchema bool
}

// ActionTypes returns the action types for the connection.
//
// Refer to the specifications in the file "connector/Actions support.md" for
// more details.
//
// It returns an errors.UnprocessableError error with code
//
//   - FetchSchemaFailed, if an error occurred fetching the schema.
func (this *Connection) ActionTypes() ([]*ActionType, error) {
	var actionTypes []*ActionType
	c := this.connection
	connector := c.Connector()
	wsSchemas := c.Workspace().Schemas
	targets := connector.Targets
	if targets.Contains(state.UsersTarget) {
		switch typ := c.Connector().Type; typ {
		case
			state.AppType,
			state.FileType:
			var name, description string
			if c.Role == state.SourceRole {
				name = "Import " + connector.TermForUsers
				description = "Import the " + connector.TermForUsers
				if connector.TermForUsers != "users" {
					description += " as users"
				}
				description += " from " + connector.Name
			} else {
				name = "Export " + connector.TermForUsers
				description = "Export the users "
				if connector.TermForUsers != "users" {
					description += " as " + connector.TermForUsers
				}
				description += " to " + connector.Name
			}
			at := &ActionType{
				Name:          name,
				Description:   description,
				Target:        UsersTarget,
				MissingSchema: wsSchemas["users"] == nil,
			}
			actionTypes = append(actionTypes, at)
		case state.DatabaseType:
			if c.Role == state.SourceRole {
				description := "Import the " + connector.TermForUsers
				if connector.TermForUsers != "users" {
					description += " as users"
				}
				description += " from " + connector.Name
				at := &ActionType{
					Name:        "Import " + connector.TermForUsers,
					Description: description,
					Target:      UsersTarget,
				}
				actionTypes = append(actionTypes, at)
			}
		}
	}
	if targets.Contains(state.GroupsTarget) {
		switch typ := c.Connector().Type; typ {
		case
			state.AppType,
			state.FileType:
			var name, description string
			if c.Role == state.SourceRole {
				name = "Import " + connector.TermForGroups
				description = "Import the " + connector.TermForGroups
				if connector.TermForGroups != "groups" {
					description += " as groups"
				}
				description += " from " + connector.Name
			} else {
				name = "Export " + connector.TermForGroups
				description = "Export the groups "
				if connector.TermForGroups != "groups" {
					description += " as " + connector.TermForGroups
				}
				description += " to " + connector.Name
			}
			at := &ActionType{
				Name:          name,
				Description:   description,
				Target:        GroupsTarget,
				MissingSchema: wsSchemas["groups"] == nil,
			}
			actionTypes = append(actionTypes, at)
		case state.DatabaseType:
			if c.Role == state.SourceRole {
				description := "Import the " + connector.TermForGroups
				if connector.TermForGroups != "groups" {
					description += " as groups"
				}
				description += " from " + connector.Name
				at := &ActionType{
					Name:        "Import " + connector.TermForGroups,
					Description: description,
					Target:      GroupsTarget,
				}
				actionTypes = append(actionTypes, at)
			}
		}
	}
	if targets.Contains(state.EventsTarget) {
		switch typ := c.Connector().Type; typ {
		case state.MobileType, state.ServerType, state.WebsiteType:
			if c.Role == state.SourceRole {
				description := "Receive events from the "
				switch typ {
				case state.MobileType:
					description += "mobile app"
				case state.ServerType:
					description += "server"
				case state.WebsiteType:
					description += "website"
				}
				at := &ActionType{
					Name:        "Receive events",
					Description: description,
					Target:      EventsTarget,
				}
				actionTypes = append(actionTypes, at)
			}
		case state.AppType:
			eventTypes, err := this.fetchEventTypes()
			if err != nil {
				return nil, errors.Unprocessable(FetchSchemaFailed, "an error occurred fetching the schema: %w", err)
			}
			for _, et := range eventTypes {
				id := et.ID
				actionTypes = append(actionTypes, &ActionType{
					Name:        et.Name,
					Description: et.Description,
					Target:      EventsTarget,
					EventType:   &id,
				})
			}
		}
	}
	if actionTypes == nil {
		actionTypes = []*ActionType{}
	}
	return actionTypes, nil
}

type ActionSchemas struct {
	In, Out types.Type
}

// ActionSchemas returns the input and the output schemas of an action with the
// given target and event type.
func (this *Connection) ActionSchemas(target ActionTarget, eventType string) (*ActionSchemas, error) {

	connector := this.connection.Connector()
	role := _connector.Role(this.connection.Role)

	// Validate the target.
	switch target {
	case UsersTarget, GroupsTarget, EventsTarget:
		ok := allowsActionTarget(connector.Type, role, state.ActionTarget(target))
		if !ok {
			return nil, errors.NotFound("target not supported")
		}
	default:
		return nil, errors.BadRequest("invalid target")
	}
	if !connector.Targets.Contains(state.ActionTarget(target)) {
		return nil, errors.NotFound("connection does not support %s", target)
	}

	// Validate the event type.
	if target != EventsTarget && eventType != "" {
		return nil, errors.NotFound("%s target does not support event types", target)
	}

	switch connector.Type {

	case state.AppType:
		switch target {
		case UsersTarget:
			var err error
			appSchema, err := this.fetchAppSchema(state.UsersTarget, "")
			if err != nil {
				return nil, errors.Unprocessable(FetchSchemaFailed, "an error occurred fetching the schema: %w", err)
			}
			var ok bool
			grSchema, ok := this.connection.Workspace().Schemas["users"]
			if !ok {
				return nil, errors.Unprocessable(NoUsersSchema, "users schema not loaded from data warehouse")
			}
			if this.connection.Role == state.SourceRole {
				outputSchema := sourceMappingSchema(*grSchema, state.AppType)
				return &ActionSchemas{In: appSchema, Out: outputSchema}, nil
			} else {
				return &ActionSchemas{In: grSchema.Unflatten(), Out: appSchema}, nil
			}
		case GroupsTarget:
			var err error
			appSchema, err := this.fetchAppSchema(state.GroupsTarget, "")
			if err != nil {
				return nil, errors.Unprocessable(FetchSchemaFailed, "an error occurred fetching the schema: %w", err)
			}
			var ok bool
			grSchema, ok := this.connection.Workspace().Schemas["groups"]
			if !ok {
				return nil, errors.Unprocessable(NoGroupsSchema, "groups schema not loaded from data warehouse")
			}
			if this.connection.Role == state.SourceRole {
				return &ActionSchemas{In: appSchema, Out: grSchema.Unflatten()}, nil
			} else {
				return &ActionSchemas{In: grSchema.Unflatten(), Out: appSchema}, nil
			}
		case EventsTarget:
			if eventType == "" {
				return nil, errors.NotFound("an event type is required")
			}
			eventTypes, err := this.fetchEventTypes()
			if err != nil {
				return nil, errors.Unprocessable(FetchSchemaFailed, "an error occurred fetching the schema: %w", err)
			}
			var et *_connector.EventType
			for _, e := range eventTypes {
				if e.ID == eventType {
					et = e
					break
				}
			}
			if et == nil {
				return nil, errors.Unprocessable(EventTypeNotExists, "event type %q not found", eventType)
			}
			etSchema, err := this.fetchAppSchema(state.EventsTarget, eventType)
			if err != nil {
				return nil, errors.Unprocessable(FetchSchemaFailed, "an error occurred fetching the schema: %w", err)
			}
			// Note that etSchema may be invalid.
			return &ActionSchemas{In: events.Schema.Unflatten(), Out: etSchema}, nil
		default:
			panic("unexpected target")
		}

	case state.DatabaseType:
		switch target {
		case UsersTarget:
			users, ok := this.connection.Workspace().Schemas["users"]
			if !ok {
				return nil, errors.Unprocessable(NoUsersSchema, "users schema not loaded from data warehouse")
			}
			out := sourceMappingSchema(*users, state.DatabaseType)
			return &ActionSchemas{Out: out}, nil
		case GroupsTarget:
			groups, ok := this.connection.Workspace().Schemas["groups"]
			if !ok {
				return nil, errors.Unprocessable(NoGroupsSchema, "groups schema not loaded from data warehouse")
			}
			return &ActionSchemas{Out: groups.Unflatten()}, nil
		default:
			panic("unexpected target")
		}

	case state.FileType:
		switch target {
		case UsersTarget:
			users, ok := this.connection.Workspace().Schemas["users"]
			if !ok {
				return nil, errors.Unprocessable(NoUsersSchema, "users schema not loaded from data warehouse")
			}
			schemas := &ActionSchemas{}
			if this.connection.Role == state.SourceRole {
				schemas.Out = sourceMappingSchema(*users, state.FileType)
			} else {
				schemas.In = users.Unflatten()
			}
			return schemas, nil
		case GroupsTarget:
			groups, ok := this.connection.Workspace().Schemas["groups"]
			if !ok {
				return nil, errors.Unprocessable(NoGroupsSchema, "groups schema not loaded from data warehouse")
			}
			schemas := &ActionSchemas{}
			if this.connection.Role == state.SourceRole {
				schemas.Out = groups.Unflatten()
			} else {
				schemas.In = groups.Unflatten()
			}
			return schemas, nil
		default:
			panic("unexpected target")
		}

	case state.ServerType, state.StreamType, state.WebsiteType:
		if eventType != "" {
			return nil, errors.NotFound("event type not expected")
		}
		return &ActionSchemas{}, nil

	default:
		panic("unexpected connection type")

	}

}

// AddAction adds action to the connection returning the identifier of the
// added action. target is the target of the action and must be supported by the
// connector of the connection.
//
// Refer to the specifications in the file "connector/Actions support.md" for
// more details.
//
// It returns an errors.UnprocessableError error with code
//
//   - EventTypeNotExists, if the specified event type does not exist.
//   - FetchSchemaFailed, if an error occurred fetching the action's schema.
//   - PropertyNotExists, if a property of a mapping / transformation does not
//     exist in the schema (except for properties of the event type schema,
//     which is specified and thus returned as an errors.BadRequest error).
//   - QueryExecutionFailed, if the execution of the action's query fails.
//   - TargetAlreadyExists, if the connection already has an action with the
//     same target.
//
// It returns an errors.NotFoundError error if the connection does not exist
// anymore.
func (this *Connection) AddAction(target ActionTarget, eventType string, action ActionToSet) (int, error) {

	c := this.connection
	connector := c.Connector()

	// Validate the target and the event type.
	switch target {
	case EventsTarget:
	case UsersTarget, GroupsTarget:
		if eventType != "" {
			return 0, errors.BadRequest("users and groups actions cannot have an event type")
		}
	default:
		return 0, errors.BadRequest("target %q is not valid", target)
	}

	// Check if the connection, with its connector type and role, allows the
	// given target.
	ok := allowsActionTarget(connector.Type, _connector.Role(c.Role), state.ActionTarget(target))
	if !ok {
		return 0, errors.BadRequest("target %q is not supported", target)
	}

	// Validate the arguments.
	schema, err := this.validateActionToSet(action, state.ActionTarget(target), eventType)
	if err != nil {
		return 0, err
	}

	n := state.AddActionNotification{
		Connection:     c.ID,
		Target:         state.ActionTarget(target),
		Name:           action.Name,
		Enabled:        action.Enabled,
		EventType:      eventType,
		ScheduleStart:  int16(mathrand.Intn(24 * 60)),
		SchedulePeriod: 60,
		Mapping:        action.Mapping,
		Transformation: (*state.Transformation)(action.Transformation),
		Query:          action.Query,
		Path:           action.Path,
		Sheet:          action.Sheet,
		ExportMode:     (*state.ExportMode)(action.ExportMode),
	}
	if shouldStoreActionSchema(connector.Type, c.Role, n.Target) {
		n.Schema = schema
	}

	// Marshal the filter.
	var filter, mapping, tIn, tOut, tSource []byte
	if action.Filter != nil {
		n.Filter = &state.ActionFilter{
			Logical:    state.ActionFilterLogical(action.Filter.Logical),
			Conditions: make([]state.ActionFilterCondition, len(action.Filter.Conditions)),
		}
		for i, condition := range action.Filter.Conditions {
			n.Filter.Conditions[i] = (state.ActionFilterCondition)(condition)
		}
		filter, err = json.Marshal(action.Filter)
		if err != nil {
			return 0, err
		}
	}

	// Marshal the mapping.
	if action.Mapping != nil {
		mapping, err = json.Marshal(action.Mapping)
		if err != nil {
			return 0, err
		}
	}

	// Marshal the transformation.
	if t := action.Transformation; t != nil {
		tIn, err = json.Marshal(t.In)
		if err != nil {
			return 0, err
		}
		tOut, err = json.Marshal(t.Out)
		if err != nil {
			return 0, err
		}
		tSource = []byte(t.PythonSource)
	}

	// Generate a random identifier.
	n.ID, err = generateRandomID()
	if err != nil {
		return 0, err
	}

	// Marshal the schema.
	var rawSchema []byte
	if n.Schema.Valid() {
		rawSchema, err = n.Schema.MarshalJSON()
		if err != nil {
			if eventType == "" {
				return 0, fmt.Errorf("cannot marshal fetched schema for target %s of connection %d: %s", target, c.ID, err)
			}
			return 0, fmt.Errorf("cannot marshal fetched schema for event type %q of connection %d: %s", target, c.ID, err)
		}
		if utf8.RuneCount(rawSchema) > rawSchemaMaxSize {
			if eventType == "" {
				return 0, fmt.Errorf("cannot marshal fetched schema for target %s of connection %d: data is too large", target, c.ID)
			}
			return 0, fmt.Errorf("cannot marshal fetched schema for event type %q of connection %d: data is too large", target, c.ID)
		}
	} else {
		rawSchema = []byte{}
	}

	// Handle the matching properties.
	if props := action.MatchingProperties; props != nil {
		n.MatchingProperties = &state.MatchingProperties{
			Internal: props.Internal,
			External: props.External,
		}
	}

	// Add the action.
	ctx := context.Background()
	err = this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		switch n.Target {
		case state.EventsTarget:
			switch typ := c.Connector().Type; typ {
			case state.MobileType, state.ServerType, state.WebsiteType:
				err = tx.QueryVoid(ctx, "SELECT FROM actions WHERE connection = $1 AND target = 'Events'", n.Connection)
				if err != sql.ErrNoRows {
					if err == nil {
						err = errors.Unprocessable(TargetAlreadyExists,
							"action with target %s already exists for %s connection %d", n.Target, typ, n.Connection)
					}
					return err
				}
			}
		case state.UsersTarget, state.GroupsTarget:
			// Make sure that users and groups actions have the same schedule start.
			err = tx.QueryRow(ctx, "SELECT schedule_start FROM actions WHERE connection = $1\n"+
				" AND target IN ('Users', 'Groups') LIMIT 1", n.Connection).Scan(&n.ScheduleStart)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
		}
		var matchPropInternal, matchPropExternal string
		if n.MatchingProperties != nil {
			matchPropInternal = n.MatchingProperties.Internal
			matchPropExternal = n.MatchingProperties.External
		}
		query := "INSERT INTO actions (id, connection, target, event_type, name, enabled,\n" +
			"schedule_start, schedule_period, filter, schema, mapping,\n" +
			"transformation.in_types, transformation.out_types, transformation.python_source,\n" +
			"query, path, sheet,\n" +
			"export_mode, matching_properties_internal, matching_properties_external)\n" +
			" VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)"
		_, err := tx.Exec(ctx, query, n.ID, n.Connection, n.Target, n.EventType, n.Name,
			n.Enabled, n.ScheduleStart, n.SchedulePeriod,
			string(filter), rawSchema, string(mapping), string(tIn), string(tOut), string(tSource),
			n.Query, n.Path, n.Sheet, n.ExportMode, matchPropInternal, matchPropExternal)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) && postgres.ErrConstraintName(err) == "connections_connection_fkey" {
				err = errors.Unprocessable(ConnectorNotExists, "connection %d does not exist", n.Connection)
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

// Delete deletes the connection.
//
// It returns an errors.NotFoundError error if the connection does not exist
// anymore.
func (this *Connection) Delete() error {
	n := state.DeleteConnectionNotification{
		ID: this.connection.ID,
	}
	connector := this.connection.Connector()
	ctx := context.Background()
	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "DELETE FROM connections WHERE id = $1", n.ID)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.NotFound("connection %d does not exist", n.ID)
		}
		if connector.OAuth != nil {
			// Delete the resource of the deleted connection if it has no other connections.
			_, err := tx.Exec(ctx, "DELETE FROM resources AS r WHERE NOT EXISTS (\n"+
				"\tSELECT FROM connections AS c\n"+
				"\tWHERE r.id = c.resource AND c.id <> $1 AND c.resource IS NULL\n)", n.ID)
			if err != nil {
				return err
			}
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// ExecQuery executes the given query on the connection and returns the
// resulting rows and schema. The connection must be a source database
// connection.
//
// query must be UTF-8 encoded, it cannot be longer than 16,777,215 runes and
// must contain the '$limit' variable (between '{{' and '}}'). limit must be
// in range [0, 100].
//
// If the connection does not exist, it returns an errors.NotFoundError error.
// If the execution of the query fails, it returns an errors.UnprocessableError
// with code QueryExecutionFailed.
func (this *Connection) ExecQuery(query string, limit int) ([][]string, types.Type, error) {

	if !utf8.ValidString(query) {
		return nil, types.Type{}, errors.BadRequest("query is not UTF-8 encoded")
	}
	if utf8.RuneCountInString(query) > queryMaxSize {
		return nil, types.Type{}, errors.BadRequest("query is longer than 16,777,215 runes")
	}
	if limit < 0 || limit > 100 {
		return nil, types.Type{}, errors.BadRequest("limit %d is not valid", limit)
	}

	c := this.connection
	connector := c.Connector()
	if connector.Type != state.DatabaseType {
		return nil, types.Type{}, errors.BadRequest("connection %d is not a database", c.ID)
	}
	if c.Role != state.SourceRole {
		return nil, types.Type{}, errors.BadRequest("database %d is not a source", c.ID)
	}

	const cRole = _connector.SourceRole

	// Execute the query.
	var err error
	query, err = compileActionQuery(query, limit)
	if err != nil {
		return nil, types.Type{}, errors.Unprocessable(QueryExecutionFailed, "query execution of connection %d failed: %w", c.ID, err)
	}
	fh := this.newFirehose(context.Background())
	connection, err := _connector.RegisteredDatabase(connector.Name).Open(fh.ctx, &_connector.DatabaseConfig{
		Role:     cRole,
		Settings: c.Settings,
		Firehose: fh,
	})
	if err != nil {
		return nil, types.Type{}, err
	}
	rawRows, properties, err := connection.Query(query)
	if err != nil {
		return nil, types.Type{}, errors.Unprocessable(QueryExecutionFailed, "query execution of connection %d failed: %w", c.ID, err)
	}

	schema, err := types.ObjectOf(properties)
	if err != nil {
		_ = rawRows.Close()
		for _, p := range properties {
			if !types.IsValidPropertyName(p.Name) {
				return nil, types.Type{}, errors.Unprocessable(QueryExecutionFailed, "database has returned an invalid column name: %q", p.Name)
			}
		}
		return nil, types.Type{}, errors.Unprocessable(QueryExecutionFailed, "%w", err)
	}

	// Fill the rows.
	dest := make([]any, len(properties))

	var rows []map[string]any
	for rawRows.Next() {
		row := make(map[string]any, len(properties))
		for i, p := range properties {
			dest[i] = databaseScanValue{property: p, row: row}
		}
		if err := rawRows.Scan(dest...); err != nil {
			return nil, types.Type{}, errors.Unprocessable(QueryExecutionFailed, "query execution of connection %d failed: %w", c.ID, err)
		}
		rows = append(rows, row)
	}
	err = rawRows.Close()
	if err != nil {
		return nil, types.Type{}, errors.Unprocessable(QueryExecutionFailed, "query execution of connection %d failed: %w", c.ID, err)
	}

	stringRows := make([][]string, len(rows))
	for i, row := range rows {
		stringRows[i] = make([]string, len(properties))
		for j, n := range properties {
			if row[n.Name] != nil {
				stringRows[i][j] = fmt.Sprintf("%v", row[n.Name])
			}
		}
	}

	return stringRows, schema, nil
}

// An Execution describes an action execution as returned by Executions.
type Execution struct {
	ID        int
	Action    int
	StartTime time.Time
	EndTime   *time.Time
	Error     string
}

// Executions returns a list of Execution describing all executions of the
// actions of the connection.
// The connection must be an app, database, or file connection.
func (this *Connection) Executions() ([]*Execution, error) {
	c := this.connection
	connector := c.Connector()
	switch connector.Type {
	case state.AppType, state.DatabaseType, state.FileType, state.StreamType:
	default:
		return nil, errors.BadRequest("connection %d cannot have executions, it's a %s connection",
			c.ID, strings.ToLower(connector.Type.String()))
	}
	executions := []*Execution{}
	err := this.db.QueryScan(context.Background(),
		"SELECT e.id, e.action, e.start_time, e.end_time, e.error\n"+
			"FROM actions_executions e\n"+
			"INNER JOIN actions a ON a.id = e.action\n"+
			"WHERE a.connection = $1\n"+
			"ORDER BY id DESC", c.ID, func(rows *postgres.Rows) error {
			var err error
			for rows.Next() {
				var exe Execution
				if err = rows.Scan(&exe.ID, &exe.Action, &exe.StartTime, &exe.EndTime, &exe.Error); err != nil {
					return err
				}
				executions = append(executions, &exe)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return executions, nil
}

// GenerateKey generates a new key for the connection. The connection must be a
// source server connection.
//
// If the server does not exist, it returns an errors.NotFoundError error.
// If the server has already too many keys, it returns an
// errors.UnprocessableError error with code TooManyKeys.
func (this *Connection) GenerateKey() (string, error) {
	c := this.connection
	connector := c.Connector()
	if connector.Type != state.ServerType {
		return "", errors.NotFound("connection %d is not a server", c.ID)
	}
	if c.Role != state.SourceRole {
		return "", errors.NotFound("server %d is not a source", c.ID)
	}
	value, err := generateServerKey()
	if err != nil {
		return "", err
	}
	n := state.AddConnectionKeyNotification{
		Connection:   c.ID,
		Value:        value,
		CreationTime: time.Now().UTC(),
	}
	ctx := context.Background()
	err = this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		var count int
		err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM connections_keys WHERE connection = $1", n.Connection).Scan(&count)
		if err != nil {
			return err
		}
		if count == maxKeysPerServer {
			return errors.Unprocessable(TooManyKeys, "server %d has already %d types", n.Connection, maxKeysPerServer)
		}
		_, err = tx.Exec(ctx, "INSERT INTO connections_keys (connection, value, creation_time) VALUES ($1, $2, $3)",
			n.Connection, n.Value, n.CreationTime)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				if postgres.ErrConstraintName(err) == "connections_keys_connection_fkey" {
					err = errors.NotFound("connection %d does not exist", n.Connection)
				}
			}
			return err
		}
		return tx.Notify(ctx, n)
	})
	if err != nil {
		return "", err
	}

	return value, nil
}

// Keys returns the keys of the source server with identifier id.
//
// If the server does not exist, it returns an errors.NotFoundError error.
func (this *Connection) Keys() ([]string, error) {
	c := this.connection
	if c.Connector().Type != state.ServerType {
		return nil, errors.NotFound("connection %d is not a server", c.ID)
	}
	if c.Role != state.SourceRole {
		return nil, errors.NotFound("server %d is not a source", c.ID)
	}
	return slices.Clone(c.Keys), nil
}

// Records returns the records and the schema of the file with the given path
// for the connection, that must be a file connection. path must be UTF-8
// encoded with a length in range [1, 1024]. If the connection has multiple
// sheets, sheet is the sheet name and must be UTF-8 encoded with a length in
// range [1, 100], otherwise must be an empty string. limit limits the number of
// records to return and must be in range [0, 100].
//
// It returns an errors.UnprocessableError error with code ReadFileFailed, if
// an error occurred reading the file.
func (this *Connection) Records(path, sheet string, limit int) ([][]any, types.Type, error) {

	c := this.connection
	connector := c.Connector()

	// Validate the connection type.
	if connector.Type != state.FileType {
		return nil, types.Type{}, errors.BadRequest("connection %d is not a file connection", c.ID)
	}
	// Validate the path.
	if path == "" {
		return nil, types.Type{}, errors.BadRequest("path cannot be empty")
	}
	if !utf8.ValidString(path) {
		return nil, types.Type{}, errors.BadRequest("path is not UTF-8 encoded")
	}
	if n := utf8.RuneCountInString(path); n > 1024 {
		return nil, types.Type{}, errors.BadRequest("path is longer than 1024 runes")
	}
	// Validate the sheet.
	if connector.HasSheets {
		if sheet == "" {
			return nil, types.Type{}, errors.BadRequest("sheet cannot be empty because connection %d has sheets", c.ID)
		}
		if !utf8.ValidString(sheet) {
			return nil, types.Type{}, errors.BadRequest("sheet is not UTF-8 encoded")
		}
		if n := utf8.RuneCountInString(sheet); n > 100 {
			return nil, types.Type{}, errors.BadRequest("sheet is longer than 100 runes")
		}
	} else {
		if sheet != "" {
			return nil, types.Type{}, errors.BadRequest("sheet must be empty because connection %d does not have sheets", c.ID)
		}
	}
	// Validate the limit.
	if limit < 1 || limit > 100 {
		return nil, types.Type{}, errors.BadRequest("limit %d is not valid", limit)
	}

	// Connect to the file connector.
	ctx := context.Background()
	fh := this.newFirehose(ctx)
	file, err := _connector.RegisteredFile(connector.Name).Open(fh.ctx, &_connector.FileConfig{
		Role:     _connector.SourceRole,
		Settings: c.Settings,
		Firehose: fh,
	})
	if err != nil {
		return nil, types.Type{}, err
	}

	// Open the file.
	var r io.ReadCloser
	{
		s, _ := c.Storage()
		fh := this.newFirehoseForConnection(ctx, s)
		ctx = fh.ctx
		var storage _connector.StorageConnection
		storage, err = _connector.RegisteredStorage(s.Connector().Name).Open(ctx, &_connector.StorageConfig{
			Role:     _connector.SourceRole,
			Settings: s.Settings,
			Firehose: fh,
		})
		if err != nil {
			return nil, types.Type{}, err
		}
		r, _, err = storage.Open(path)
		if err != nil {
			return nil, types.Type{}, errors.Unprocessable(ReadFileFailed, "%w", err)
		}
		defer r.Close()
	}

	// Read the records.
	rw := newRecordWriter(c.ID, limit, nil)
	err = file.Read(r, sheet, rw)
	if err != nil && err != errRecordStop {
		return nil, types.Type{}, errors.Unprocessable(ReadFileFailed, "%w", err)
	}
	if rw.columns == nil {
		return nil, types.Type{}, errors.Unprocessable(ReadFileFailed, "%w", errNoColumns)
	}
	schema := types.Object(rw.columns)

	return rw.records, schema, nil
}

// Reload reloads the user, group and events schema for the actions of the
// connection. If the actions do not have schemas to reload, it does nothing.
func (this *Connection) Reload() error {
	c := this.connection
	connector := c.Connector()
	if connector.Targets.Contains(state.UsersTarget) && shouldStoreActionSchema(connector.Type, c.Role, state.UsersTarget) {
		err := this.reloadUserSchema()
		if err != nil {
			return err
		}
	}
	if connector.Targets.Contains(state.EventsTarget) && shouldStoreActionSchema(connector.Type, c.Role, state.EventsTarget) {
		err := this.reloadEventSchemas()
		if err != nil {
			return err
		}
	}
	return nil
}

// Rename renames the connection with the given new name.
// name must be between 1 and 100 runes long.
//
// It returns an errors.NotFoundError error if the connection does not exist
// anymore.
func (this *Connection) Rename(name string) error {
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return errors.BadRequest("name %q is not valid", name)
	}
	if name == this.connection.Name {
		return nil
	}
	n := state.RenameConnectionNotification{
		Connection: this.connection.ID,
		Name:       name,
	}
	ctx := context.Background()
	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE connections SET name = $1 WHERE id = $2", n.Name, n.Connection)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.NotFound("connection %d does not exist", n.Connection)
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// RevokeKey revokes the given key of the source server with identifier id. key
// cannot be empty and cannot be the unique key of the server.
//
// If the key does not exist, it returns an errors.NotFoundError error.
// If the key is the unique key of the server, it returns an
// errors.UnprocessableError error with code UniqueKey.
func (this *Connection) RevokeKey(key string) error {
	if key == "" {
		return errors.BadRequest("key is empty")
	}
	if !isServerKey(key) {
		return errors.BadRequest("key %q is malformed", key)
	}
	c := this.connection
	connector := c.Connector()
	if connector.Type != state.ServerType {
		return errors.BadRequest("connection %d is not a server", c.ID)
	}
	if c.Role != state.SourceRole {
		return errors.BadRequest("server %d is not a source", c.ID)
	}
	n := state.RevokeConnectionKeyNotification{
		Connection: c.ID,
		Value:      key,
	}
	ctx := context.Background()
	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		var count int
		err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM connections_keys WHERE connection = $1", n.Connection).Scan(&count)
		if err != nil {
			return err
		}
		if count == 1 {
			return errors.Unprocessable(UniqueKey, "key cannot be revoked because it's the unique key of the server")
		}
		result, err := tx.Exec(ctx, "DELETE FROM connections_keys WHERE connection = $1 AND value = $2", n.Connection, n.Value)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.Unprocessable(KeyNotExists, "key %q does not exist", key)
		}
		return tx.Notify(ctx, n)
	})

	return err
}

// SetStatus sets the status of the connection.
func (this *Connection) SetStatus(enabled bool) error {
	if enabled == this.Enabled {
		return nil
	}
	n := state.SetConnectionStatusNotification{
		Connection: this.connection.ID,
		Enabled:    enabled,
	}
	ctx := context.Background()
	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE connections SET enabled = $1 WHERE id = $2 AND enabled <> $1", n.Enabled, n.Connection)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return nil
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// ServeUI serves the user interface for the connection. event is the event and
// values contains the form values in JSON format.
//
// If the connection does not exist, it returns an errors.NotFoundError error.
// If the event does not exist, it returns an errors.UnprocessableError error
// with code EventNotExists.
func (this *Connection) ServeUI(event string, values []byte) ([]byte, error) {

	c := this.connection
	role := _connector.Role(c.Role)
	connector := c.Connector()

	var err error
	var connection any

	switch connector.Type {
	case state.AppType:

		var resource string
		if r, ok := c.Resource(); ok {
			resource = r.Code
		}

		fh := this.newFirehose(context.Background())
		connection, err = _connector.RegisteredApp(connector.Name).Open(fh.ctx, &_connector.AppConfig{
			Role:          role,
			Settings:      c.Settings,
			Firehose:      fh,
			Resource:      resource,
			HTTPClient:    this.http.ConnectionClient(c.ID),
			PrivacyRegion: _connector.PrivacyRegion(c.Workspace().PrivacyRegion),
		})

	default:

		fh := this.newFirehose(context.Background())

		switch connector.Type {
		case state.DatabaseType:
			connection, err = _connector.RegisteredDatabase(connector.Name).Open(fh.ctx, &_connector.DatabaseConfig{
				Role:     role,
				Settings: c.Settings,
				Firehose: fh,
			})
		case state.FileType:
			connection, err = _connector.RegisteredFile(connector.Name).Open(fh.ctx, &_connector.FileConfig{
				Role:     role,
				Settings: c.Settings,
				Firehose: fh,
			})
		case state.MobileType:
			connection, err = _connector.RegisteredMobile(connector.Name).Open(fh.ctx, &_connector.MobileConfig{
				Role:     role,
				Settings: c.Settings,
				Firehose: fh,
			})
		case state.ServerType:
			connection, err = _connector.RegisteredServer(connector.Name).Open(fh.ctx, &_connector.ServerConfig{
				Role:     role,
				Settings: c.Settings,
				Firehose: fh,
			})
		case state.StorageType:
			connection, err = _connector.RegisteredStorage(connector.Name).Open(fh.ctx, &_connector.StorageConfig{
				Role:     role,
				Settings: c.Settings,
				Firehose: fh,
			})
		case state.StreamType:
			connection, err = _connector.RegisteredStream(connector.Name).Open(fh.ctx, &_connector.StreamConfig{
				Role:     role,
				Settings: c.Settings,
				Firehose: fh,
			})
		case state.WebsiteType:
			connection, err = _connector.RegisteredWebsite(connector.Name).Open(fh.ctx, &_connector.WebsiteConfig{
				Role:     role,
				Settings: c.Settings,
				Firehose: fh,
			})
		}

	}
	if err != nil {
		return nil, err
	}
	connectionUI, ok := connection.(_connector.UI)
	if !ok {
		return nil, errors.BadRequest("connector %d does not have a UI", c.ID)
	}

	// TODO: check and delete alternative fieldsets keys that have 'null' value
	// before saving to database
	form, alert, err := connectionUI.ServeUI(event, values)
	if err != nil {
		if err == ui.ErrEventNotExist {
			err = errors.Unprocessable(EventNotExists, "UI event %q does not exist for %s connector",
				event, connector.Name)
		}
		return nil, err
	}

	return marshalUIFormAlert(form, alert, ui.Role(c.Role))
}

// SetStorage sets the storage of the connection. The connection must be a file
// connection. storage is the storage connection. The connection and the
// storage must have the same role. As a special case, the current storage of
// the file, if there is one, is removed if the storage argument is 0.
//
// If the connection does not exist anymore, it returns an errors.NotFoundError
// error.
// If the storage does not exist, it returns an errors.UnprocessableError error
// with code StorageNotExists.
func (this *Connection) SetStorage(storage int) error {

	if storage < 0 || storage > maxInt32 {
		return errors.BadRequest("storage identifier %d is not valid", storage)
	}

	c := this.connection
	if c.Connector().Type != state.FileType {
		return errors.BadRequest("file is not a file connector")
	}
	var s *state.Connection
	if storage > 0 {
		var ok bool
		s, ok = c.Workspace().Connection(storage)
		if !ok {
			return errors.Unprocessable(StorageNotExists, "storage %d does not exist", storage)
		}
		if s.Connector().Type != state.StorageType {
			return errors.BadRequest("connection %d is not a storage", storage)
		}
		if s.Role != c.Role {
			if c.Role == state.SourceRole {
				return errors.BadRequest("storage %d is not a source", storage)
			}
			return errors.BadRequest("storage %d is not a destination", storage)
		}
	}

	n := state.SetConnectionStorageNotification{
		Connection: c.ID,
		Storage:    storage,
	}

	ctx := context.Background()

	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE connections SET storage = NULLIF($1, 0) WHERE id = $2", n.Storage, n.Connection)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				if postgres.ErrConstraintName(err) == "connections_storage_fkey" {
					err = errors.Unprocessable(StorageNotExists, "storage %d does not exist", storage)
				}
			}
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.NotFound("connection %d does not exist", n.Connection)
		}
		return tx.Notify(ctx, n)
	})

	return err
}

// Sheets returns the sheets of the file at the given path. The connection must
// be a file connection with multi sheets support and path must be a not empty
// UTF-8 encoded string.
//
// If the connection does not exist anymore, it returns an errors.NotFoundError
// error.
//
// It returns an errors.UnprocessableError error with code
//
//   - NoStorage, if the file connection does not have a storage.
//   - ReadFileFailed, if an error occurred reading the file.
func (this *Connection) Sheets(path string) ([]string, error) {
	c := this.connection
	connector := c.Connector()
	if connector.Type != state.FileType {
		return nil, errors.BadRequest("connection %d is not a file", c.ID)
	}
	if path == "" {
		return nil, errors.BadRequest("path is empty")
	}
	if !utf8.ValidString(path) {
		return nil, errors.BadRequest("path is not UTF-8 encoded")
	}
	ctx := context.Background()
	f, err := _connector.RegisteredFile(connector.Name).Open(ctx, &_connector.FileConfig{
		Role:     _connector.Role(c.Role),
		Settings: c.Settings,
	})
	if err != nil {
		return nil, err
	}
	file, ok := f.(_connector.Sheets)
	if !ok {
		return nil, errors.BadRequest("file connection %d does not support multiple sheet", c.ID)
	}
	s, ok := c.Storage()
	if !ok {
		return nil, errors.Unprocessable(NoStorage, "file connection %d does not have a storage", c.ID)
	}
	storage, err := _connector.RegisteredStorage(s.Connector().Name).Open(ctx, &_connector.StorageConfig{
		Role:     _connector.Role(s.Role),
		Settings: s.Settings,
	})
	if err != nil {
		return nil, err
	}
	r, _, err := storage.Open(path)
	if err != nil {
		return nil, errors.Unprocessable(ReadFileFailed, "%w", err)
	}
	defer r.Close()
	sheets, err := file.Sheets(r)
	if err != nil {
		return nil, errors.Unprocessable(ReadFileFailed, "%w", err)
	}
	return sheets, nil
}

// ConnectionsStats represents the statistics on a connection for the last 24
// hours.
type ConnectionsStats struct {
	Users [24]int // ingested or loaded users per hour
}

// Stats returns statistics on the connection for the last 24 hours.
//
// It returns an errors.NotFound error if the connection does not exist
// anymore.
func (this *Connection) Stats() (*ConnectionsStats, error) {
	now := time.Now().UTC()
	toSlot := statsTimeSlot(now)
	fromSlot := toSlot - 23
	stats := &ConnectionsStats{
		Users: [24]int{},
	}
	query := "SELECT time_slot, users\nFROM connections_stats\nWHERE connection = $1 AND time_slot BETWEEN $2 AND $3"
	err := this.db.QueryScan(context.Background(), query, this.connection.ID, fromSlot, toSlot, func(rows *postgres.Rows) error {
		var err error
		var slot, users int
		for rows.Next() {
			if err = rows.Scan(&slot, &users); err != nil {
				return err
			}
			stats.Users[slot-fromSlot] = users
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// newFirehoseForConnection returns a new Firehose for the connection c.
func (this *Connection) newFirehoseForConnection(ctx context.Context, c *state.Connection) *firehose {
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

// newFirehose returns a new Firehose.
func (this *Connection) newFirehose(ctx context.Context) *firehose {
	var resource int
	if r, ok := this.connection.Resource(); ok {
		resource = r.ID
	}
	fh := &firehose{
		// TODO(Gianluca): here the action is not set, as the action is not
		// available in contexts where this methods in called. Refactor the code
		// and then change / review this method.
		db:         this.db,
		connection: this.connection,
		resource:   resource,
	}
	fh.ctx, fh.cancel = context.WithCancel(ctx)
	return fh
}

var errRecordStop = errors.New("stop record")

// reloadEventSchemas reloads the events schemas of the connection.
//
// It returns an errors.UnprocessableError error with code
//
//   - FetchSchemaFailed, if an error occurred fetching the schema.
func (this *Connection) reloadEventSchemas() error {

	for _, action := range this.connection.Actions() {
		if action.Target != state.EventsTarget {
			continue
		}
		// Fetch the schema.
		schema, err := this.fetchAppSchema(state.EventsTarget, action.EventType)
		if err != nil {
			return errors.Unprocessable(FetchSchemaFailed, "an error occurred fetching the schema: %w", err)
		}
		if schema.EqualTo(action.Schema) {
			continue
		}
		// Update the schema.
		rawSchema, err := schema.MarshalJSON()
		if err != nil {
			return fmt.Errorf("cannot marshal fetched schema of action %d: %s", action.ID, err)
		}
		if utf8.RuneCount(rawSchema) > rawSchemaMaxSize {
			return fmt.Errorf("cannot marshal fetched schema of the action %d: data is too large", action.ID)
		}
		n := state.SetActionSchemaNotification{
			ID:     action.ID,
			Schema: schema,
		}
		ctx := context.Background()
		err = this.db.Transaction(ctx, func(tx *postgres.Tx) error {
			result, err := tx.Exec(ctx, "UPDATE actions SET \"schema\" = $1 WHERE id = $2 AND \"schema\" <> $1", rawSchema, n.ID)
			if err != nil {
				return err
			}
			if result.RowsAffected() == 0 {
				return nil
			}
			return tx.Notify(ctx, n)
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// reloadUserSchema reloads the user schema of the connection. The connection
// must be an app connection.
//
// It returns an errors.UnprocessableError error with code
//
//   - FetchSchemaFailed, if an error occurred fetching the schema.
func (this *Connection) reloadUserSchema() error {

	c := this.connection
	if typ := c.Connector().Type; typ != state.AppType {
		return fmt.Errorf("cannot reload user schema for %s connections", typ)
	}

	// Fetch the schema.
	schema, err := this.fetchAppSchema(state.UsersTarget, "")
	if err != nil {
		return errors.Unprocessable(FetchSchemaFailed, "an error occurred fetching the schema: %w", err)
	}

	// Update the schema.
	rawSchema, err := schema.MarshalJSON()
	if err != nil {
		return fmt.Errorf("cannot marshal schema of connection %d: %s", c.ID, err)
	}
	if utf8.RuneCount(rawSchema) > rawSchemaMaxSize {
		return fmt.Errorf("cannot marshal schema of the connection %d: data is too large", c.ID)
	}

	n := state.SetActionSchemaNotification{
		Schema: schema,
	}

	ctx := context.Background()

	err = this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		err := tx.QueryRow(ctx, "UPDATE actions\nSET \"schema\" = $1\n"+
			"WHERE connection = $2 AND target = 'Users' AND \"schema\" <> $1\n"+
			"RETURNING id", rawSchema, c.ID).Scan(&n.ID)
		if err != nil {
			return err
		}
		if n.ID == 0 {
			return nil
		}
		return tx.Notify(ctx, n)
	})
	if err == sql.ErrNoRows {
		err = nil
	}

	return err
}

// isServerKey reports whether key can be a server key.
func isServerKey(key string) bool {
	if len(key) != 32 {
		return false
	}
	_, err := base62.DecodeString(key)
	return err == nil
}

// generateServerKey generates a server key in its base62 form.
func generateServerKey() (string, error) {
	key := make([]byte, 24)
	_, err := rand.Read(key)
	if err != nil {
		return "", errors.New("cannot generate a server key")
	}
	return base62.EncodeToString(key)[0:32], nil
}

// marshalUIFormAlert marshals form with given role and alert in JSON format.
// form and alert can be nil or not, independently of each other.
func marshalUIFormAlert(form *ui.Form, alert *ui.Alert, role ui.Role) ([]byte, error) {

	if form == nil && alert == nil {
		return []byte("null"), nil
	}

	var b bytes.Buffer
	enc := json.NewEncoder(&b)

	b.WriteString("{")

	// Serialize the form, if present.
	if form != nil {

		// Makes the keys of form.Values to have the same case as the Name field of the components.
		values := map[string]any{}
		if len(form.Values) > 0 {
			err := json.Unmarshal(form.Values, &values)
			if err != nil {
				return nil, err
			}
		}

		comma := false
		b.WriteString(`"Form":{"Fields":[`)
		for _, field := range form.Fields {
			ok, err := marshalUIComponent(&b, field, role, values, comma)
			if err != nil {
				return nil, err
			}
			if ok {
				comma = true
			}
		}
		b.WriteString(`],"Actions":`)
		err := enc.Encode(form.Actions)
		if err != nil {
			return nil, err
		}
		if len(form.Values) > 0 {
			b.WriteString(`,"Values":`)
			err = json.NewEncoder(&b).Encode(values)
			if err != nil {
				return nil, err
			}
		}
		b.WriteString("}")

	}

	// Serialize the alert, if present.
	if alert != nil {
		if form != nil {
			b.WriteString(",")
		}
		b.WriteString(`"Alert":{"Message":`)
		err := enc.Encode(alert.Message)
		if err != nil {
			return nil, err
		}
		b.WriteString(`,"Variant":"`)
		b.WriteString(alert.Variant.String())
		b.WriteString(`"`)
		b.WriteString("}")
	}

	b.WriteString(`}`)

	return b.Bytes(), nil
}

// adjustValuesCase adjusts the case of keys of values.
func adjustValuesCase(key string, values map[string]any) {
	var found struct {
		key   string
		value any
	}
	for k, v := range values {
		if strings.EqualFold(k, key) {
			found.key = k
			found.value = v
			break
		}
	}
	if found.key == "" {
		return
	}
	delete(values, found.key)
	values[key] = found.value
}

// marshalUIComponent marshals component with the given role in JSON format. If
// comma is true, it prepends a comma. Returns whether it has been marhalled.
func marshalUIComponent(b *bytes.Buffer, component ui.Component, role ui.Role, values map[string]any, comma bool) (bool, error) {
	rv := reflect.ValueOf(component).Elem()
	rt := rv.Type()
	if role != ui.BothRole {
		if r := ui.Role(rv.FieldByName("Role").Int()); r != ui.BothRole && r != role {
			return false, nil
		}
	}
	if comma {
		b.WriteString(`,`)
	}
	b.WriteString(`{"ComponentType":"`)
	b.WriteString(rt.Name())
	b.WriteString(`"`)
	for j := 0; j < rt.NumField(); j++ {
		name := rt.Field(j).Name
		if name == "Role" {
			continue
		}
		field := rv.Field(j)
		if name == "Name" && values != nil {
			adjustValuesCase(field.String(), values)
		}
		b.WriteString(`,"`)
		b.WriteString(name)
		b.WriteString(`":`)
		var err error
		switch field := field.Interface().(type) {
		case ui.Component:
			_, err = marshalUIComponent(b, field, role, values, false)
		case []ui.FieldSet:
			b.WriteByte('[')
			comma = false
			for _, set := range field {
				var ok bool
				ok, err = marshalUIFieldSet(b, set, role, values, comma)
				if ok {
					comma = true
				}
			}
			b.WriteByte(']')
		default:
			err = json.NewEncoder(b).Encode(field)
		}
		if err != nil {
			return false, err
		}
	}
	b.WriteString(`}`)
	return true, nil
}

// marshalUIFieldSet marshals fieldSet with the given role in JSON format. If
// comma is true, it prepends a comma. Returns whether it has been marhalled.
func marshalUIFieldSet(b *bytes.Buffer, fieldSet ui.FieldSet, role ui.Role, values map[string]any, comma bool) (bool, error) {
	if role != ui.BothRole {
		if fieldSet.Role != ui.BothRole && fieldSet.Role != role {
			return false, nil
		}
	}
	name := fieldSet.Name
	if values != nil {
		adjustValuesCase(name, values)
	}
	if comma {
		b.WriteByte(',')
	}
	b.WriteString(`{"Name":`)
	_ = json.NewEncoder(b).Encode(name)
	b.WriteString(`,"Label":`)
	_ = json.NewEncoder(b).Encode(fieldSet.Label)
	b.WriteString(`,"Fields":[`)
	comma = false
	for _, c := range fieldSet.Fields {
		var valuesOfSet map[string]any
		switch vs := values[name].(type) {
		case nil:
		case map[string]any:
			valuesOfSet = vs
		default:
			return false, fmt.Errorf("expected a map[string]any value for field set %s, got %T", name, values[name])
		}
		ok, err := marshalUIComponent(b, c, role, valuesOfSet, comma)
		if err != nil {
			return false, err
		}
		if ok {
			comma = true
		}
	}
	b.WriteString(`]}`)
	return true, nil
}

// abbreviate abbreviates s to almost n runes. If s is longer than n runes,
// the abbreviated string terminates with "...".
func abbreviate(s string, n int) string {
	const spaces = " \n\r\t\f" // https://infra.spec.whatwg.org/#ascii-whitespace
	s = strings.TrimRight(s, spaces)
	if len(s) <= n {
		return s
	}
	if n < 3 {
		return ""
	}
	p := 0
	n2 := 0
	for i := range s {
		switch p {
		case n - 2:
			n2 = i
		case n:
			break
		}
		p++
	}
	if p < n {
		return s
	}
	if p = strings.LastIndexAny(s[:n2], spaces); p > 0 {
		s = strings.TrimRight(s[:p], spaces)
	} else {
		s = ""
	}
	if l := len(s) - 1; l >= 0 && (s[l] == '.' || s[l] == ',') {
		s = s[:l]
	}
	return s + "..."
}

// Health is an indicator of the current state of a connection.
type Health int

const (
	Healthy Health = iota
	NoRecentData
	RecentError
	AccessDenied
)

// MarshalJSON implements the json.Marshaler interface.
// It panics if health is not a valid Health value.
func (health Health) MarshalJSON() ([]byte, error) {
	return []byte(`"` + health.String() + `"`), nil
}

// String returns the string representation of health.
// It panics if health is not a valid Health value.
func (health Health) String() string {
	switch health {
	case Healthy:
		return "Healthy"
	case NoRecentData:
		return "NoRecentData"
	case RecentError:
		return "RecentError"
	case AccessDenied:
		return "AccessDenied"
	}
	panic("invalid connection health")
}

// ConnectionRole represents a connection role.
type ConnectionRole int

const (
	SourceRole      ConnectionRole = iota + 1 // source
	DestinationRole                           // destination
)

// MarshalJSON implements the json.Marshaler interface.
// It panics if role is not a valid ConnectionRole value.
func (role ConnectionRole) MarshalJSON() ([]byte, error) {
	return []byte(`"` + role.String() + `"`), nil
}

// String returns the string representation of role.
// It panics if role is not a valid ConnectionRole value.
func (role ConnectionRole) String() string {
	switch role {
	case SourceRole:
		return "Source"
	case DestinationRole:
		return "Destination"
	}
	panic("invalid connection role")
}

var null = []byte("null")

// UnmarshalJSON implements the json.Unmarshaler interface.
func (role *ConnectionRole) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, null) {
		return nil
	}
	var v any
	err := json.Unmarshal(data, &v)
	if err != nil {
		return fmt.Errorf("json: cannot unmarshal into a apis.ConnectionRole value: %s", err)
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an api.ConnectionRole value", v)
	}
	var r ConnectionRole
	switch s {
	case "Source":
		r = SourceRole
	case "Destination":
		r = DestinationRole
	default:
		return fmt.Errorf("invalid apis.ConnectionRole: %s", s)
	}
	*role = r
	return nil
}

// setUsers sets the user with the given ID into the database and into the data
// warehouse.
func (this *Connection) setUser(ctx context.Context, id string, user map[string]any) error {

	// Resolve the entity of this user.
	ids := identitySolver{ctx, this.connection}
	email, _ := user["Email"].(string)
	if email == "" {
		return fmt.Errorf("expecting 'Email' to be a non-empty string, got %#v (of type %T)", user["Email"], user["Email"])
	}
	goldenRecordID, err := ids.ResolveEntity(this.connection.ID, id, email)
	if err != nil {
		return err
	}

	ws := this.connection.Workspace()

	// Serialize the row.
	schema, ok := ws.Schemas["users"]
	if !ok {
		return errors.New("users schema not found")
	}
	warehouses.SerializeRow(user, *schema)

	query := &strings.Builder{}
	query.WriteString("UPDATE users SET\n")
	var values []any
	i := 1
	for prop, value := range user {
		if i > 1 {
			query.WriteString(", ")
		}
		query.WriteString(postgres.QuoteIdent(prop))
		query.WriteString(" = $")
		query.WriteString(strconv.Itoa(i))
		values = append(values, value)
		i++
	}
	query.WriteString(`, "timestamp" = now()`)
	query.WriteString("\nWHERE id = $")
	query.WriteString(strconv.Itoa(i))
	values = append(values, goldenRecordID)
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

	log.Printf("[info] user %q written to the data warehouse: %#v", email, user)

	return nil
}

// writeConnectionUsers writes the user properties (before being mapped) with
// the given ID and the relative timestamps to the database.
// It also updates the statistics about the connection.
// timestamp refers to the entire set of user properties, while timestamp may
// contain timestamps for specific properties.
func (this *Connection) writeConnectionUsers(ctx context.Context, id string, user map[string]any, timestamp time.Time, timestamps map[string]time.Time) error {

	// Prepare the data that will be written to the database.
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	timestampsToWrite := map[string]time.Time{}
	for prop := range user {
		ts, ok := timestamps[prop]
		if !ok {
			ts = timestamp
		}
		timestampsToWrite[prop] = ts
	}
	jsonTimestamps, err := json.Marshal(timestampsToWrite)
	if err != nil {
		return err
	}

	// Write to the database.
	ws := this.connection.Workspace()
	connection := this.connection.ID
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

// webhookURL returns the URL of the webhook for the given connection and
// resource.
// If the connector does not support webhooks, it returns an empty string.
func webhookURL(connection *state.Connection, resource int) string {
	connector := connection.Connector()
	u := "https://localhost:9090/webhook/"
	switch connector.WebhooksPer {
	case state.WebhooksPerNone:
		return ""
	case state.WebhooksPerConnector:
		return u + "c/" + strconv.Itoa(connector.ID) + "/"
	case state.WebhooksPerResource:
		return u + "r/" + strconv.Itoa(resource) + "/"
	case state.WebhooksPerSource:
		return u + "s/" + strconv.Itoa(connection.ID) + "/"
	}
	panic("unexpected webhooksPer value")
}
