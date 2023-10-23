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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	mathrand "math/rand"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/datastore"
	"chichi/apis/errors"
	"chichi/apis/events"
	"chichi/apis/mappings"
	"chichi/apis/mappings/mapexp"
	"chichi/apis/normalization"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/apis/transformers"
	_connector "chichi/connector"
	"chichi/connector/types"
	"chichi/connector/ui"
	"chichi/telemetry"

	"github.com/google/uuid"
	"github.com/jxskiss/base62"
	"golang.org/x/exp/maps"
)

// maxSettingsLen is the maximum length of settings in runes.
// Keep in sync with the events.maxSettingsLen constant.
const maxSettingsLen = 10_000

const (
	maxKeysPerConnection = 20 // maximum number of keys per connection.
	maxInt32             = math.MaxInt32
	rawSchemaMaxSize     = 16_777_215 // maximum size in runes for schemas stored in PostgreSQL.
	queryMaxSize         = 16_777_215 // maximum size in runes of a connection query.
)

var (
	ConnectionNotExist   errors.Code = "ConnectionNotExist"
	ConnectorNotExist    errors.Code = "ConnectorNotExist"
	EventNotExist        errors.Code = "EventNotExist"
	EventTypeNotExist    errors.Code = "EventTypeNotExist"
	FetchSchemaFailed    errors.Code = "FetchSchemaFailed"
	InvalidPath          errors.Code = "InvalidPath"
	InvalidTable         errors.Code = "InvalidTable"
	KeyNotExist          errors.Code = "KeyNotExist"
	LanguageNotSupported errors.Code = "LanguageNotSupported"
	NoGroupsSchema       errors.Code = "NoGroupsSchema"
	NoStorage            errors.Code = "NoStorage"
	NoUsersSchema        errors.Code = "NoUsersSchema"
	ReadFileFailed       errors.Code = "ReadFileFailed"
	StorageNotExist      errors.Code = "StorageNotExist"
	TargetAlreadyExist   errors.Code = "TargetAlreadyExist"
	TooManyKeys          errors.Code = "TooManyKeys"
	UniqueKey            errors.Code = "UniqueKey"
	WorkspaceNotExist    errors.Code = "WorkspaceNotExist"
)

// Connection represents a connection.
type Connection struct {
	apis         *APIs
	connection   *state.Connection
	store        *datastore.Store
	ID           int
	Name         string
	Type         ConnectorType
	Role         ConnectionRole
	Connector    int
	Storage      int // zero if the connection is not a file or does not have a storage.
	HasSettings  bool
	Enabled      bool
	ActionsCount int
	Health       Health

	// ActionTypes and Actions are populated only by the (*Workspace).Connection method.
	ActionTypes *[]ActionType `json:",omitempty"`
	Actions     *[]Action     `json:",omitempty"`
}

// Action returns the action with identifier id of the connection.
// It returns an errors.NotFound error if the action does not exist.
func (this *Connection) Action(ctx context.Context, id int) (*Action, error) {
	this.apis.mustBeOpen()
	ctx, span := telemetry.TraceSpan(ctx, "Connection.Action", "id", id)
	defer span.End()
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid action identifier", id)
	}
	a, ok := this.connection.Action(id)
	if !ok {
		return nil, errors.NotFound("action %d does not exist", id)
	}
	var action Action
	action.fromState(this.apis, this.store, a)
	return &action, nil
}

type ActionSchemas struct {
	In, Out types.Type
}

// ActionSchemas returns the input and the output schemas of an action with the
// given target and event type.
func (this *Connection) ActionSchemas(ctx context.Context, target ActionTarget, eventType string) (*ActionSchemas, error) {

	this.apis.mustBeOpen()

	ctx, span := telemetry.TraceSpan(ctx, "Connection.ActionSchemas", "target", target, "eventType", eventType)
	defer span.End()

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
			appSchema, err := this.fetchAppSchema(ctx, state.UsersTarget, "")
			if err != nil {
				return nil, errors.Unprocessable(FetchSchemaFailed, "an error occurred fetching the schema: %w", err)
			}
			var ok bool
			usersIdentities, ok := this.connection.Workspace().Schemas["users_identities"]
			if !ok {
				return nil, errors.Unprocessable(NoUsersSchema, "users_identities schema not loaded from data warehouse")
			}
			if this.connection.Role == state.SourceRole {
				return &ActionSchemas{In: appSchema, Out: *usersIdentities}, nil
			} else {
				return &ActionSchemas{In: usersIdentities.Unflatten(), Out: appSchema}, nil
			}
		case GroupsTarget:
			var err error
			appSchema, err := this.fetchAppSchema(ctx, state.GroupsTarget, "")
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
			eventTypes, err := this.fetchEventTypes(ctx)
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
				return nil, errors.Unprocessable(EventTypeNotExist, "event type %q not found", eventType)
			}
			etSchema, err := this.fetchAppSchema(ctx, state.EventsTarget, eventType)
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
			if this.connection.Role == state.SourceRole {
				usersIdentities, ok := this.connection.Workspace().Schemas["users_identities"]
				if !ok {
					return nil, errors.Unprocessable(NoUsersSchema, "users_identities schema not loaded from data warehouse")
				}
				out := usersIdentities.Unflatten()
				return &ActionSchemas{Out: out}, nil
			} else {
				users, ok := this.connection.Workspace().Schemas["users"]
				if !ok {
					return nil, errors.Unprocessable(NoUsersSchema, "users schema not loaded from data warehouse")
				}
				in := users.Unflatten()
				return &ActionSchemas{In: in}, nil
			}
		case GroupsTarget:
			if this.connection.Role == state.SourceRole {
				groupsIdentities, ok := this.connection.Workspace().Schemas["groups_identities"]
				if !ok {
					return nil, errors.Unprocessable(NoGroupsSchema, "groups_identities schema not loaded from data warehouse")
				}
				out := groupsIdentities.Unflatten()
				return &ActionSchemas{Out: out}, nil
			} else {
				groups, ok := this.connection.Workspace().Schemas["groups"]
				if !ok {
					return nil, errors.Unprocessable(NoGroupsSchema, "groups schema not loaded from data warehouse")
				}
				in := groups.Unflatten()
				return &ActionSchemas{In: in}, nil
			}
		default:
			panic("unexpected target")
		}

	case state.FileType:
		switch target {
		case UsersTarget:
			if this.connection.Role == state.SourceRole {
				usersIdentities, ok := this.connection.Workspace().Schemas["users_identities"]
				if !ok {
					return nil, errors.Unprocessable(NoUsersSchema, "users_identities schema not loaded from data warehouse")
				}
				out := usersIdentities.Unflatten()
				return &ActionSchemas{Out: out}, nil
			} else {
				users, ok := this.connection.Workspace().Schemas["users"]
				if !ok {
					return nil, errors.Unprocessable(NoUsersSchema, "users schema not loaded from data warehouse")
				}
				in := users.Unflatten()
				return &ActionSchemas{In: in}, nil
			}
		case GroupsTarget:
			if this.connection.Role == state.SourceRole {
				groupsIdentities, ok := this.connection.Workspace().Schemas["groups_identities"]
				if !ok {
					return nil, errors.Unprocessable(NoGroupsSchema, "groups_identities schema not loaded from data warehouse")
				}
				out := groupsIdentities.Unflatten()
				return &ActionSchemas{Out: out}, nil
			} else {
				groups, ok := this.connection.Workspace().Schemas["groups"]
				if !ok {
					return nil, errors.Unprocessable(NoGroupsSchema, "groups schema not loaded from data warehouse")
				}
				in := groups.Unflatten()
				return &ActionSchemas{In: in}, nil
			}
		default:
			panic("unexpected target")
		}

	case state.MobileType, state.ServerType, state.StreamType, state.WebsiteType:
		if eventType != "" {
			return nil, errors.NotFound("event type not expected")
		}
		switch target {
		case UsersTarget:
			usersIdentities, ok := this.connection.Workspace().Schemas["users_identities"]
			if !ok {
				return nil, errors.Unprocessable(NoUsersSchema, "users_identities schema not loaded from data warehouse")
			}
			out := usersIdentities.Unflatten()
			return &ActionSchemas{In: events.Schema.Unflatten(), Out: out}, nil
		case GroupsTarget:
			groupsIdentities, ok := this.connection.Workspace().Schemas["groups_identities"]
			if !ok {
				return nil, errors.Unprocessable(NoGroupsSchema, "groups_identities schema not loaded from data warehouse")
			}
			out := groupsIdentities.Unflatten()
			return &ActionSchemas{In: events.Schema.Unflatten(), Out: out}, nil
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
// It returns an errors.NotFoundError error if the connection does not exist
// anymore, and returns an errors.UnprocessableError error with code
//   - ConnectionNotExist, if the connection does not exist.
//   - LanguageNotSupported, if the transformation language is not supported.
//   - MappingOverAnonymousIdentifier, if the action maps over an anonymous
//     identifier.
//   - TargetAlreadyExist, if an action already exists for a target for the
//     connection.
func (this *Connection) AddAction(ctx context.Context, target ActionTarget, eventType string, action ActionToSet) (int, error) {

	this.apis.mustBeOpen()

	ctx, span := telemetry.TraceSpan(ctx, "Connection.AddAction", "target", target, "eventType", eventType)
	defer span.End()

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
	err := this.validateActionToSet(action, state.ActionTarget(target), eventType)
	if err != nil {
		return 0, err
	}
	span.Log("action validated successfully")

	n := state.AddAction{
		Connection:     c.ID,
		Target:         state.ActionTarget(target),
		Name:           action.Name,
		Enabled:        action.Enabled,
		EventType:      eventType,
		ScheduleStart:  int16(mathrand.Intn(24 * 60)),
		SchedulePeriod: 60,
		InSchema:       action.InSchema,
		OutSchema:      action.OutSchema,
		Mapping:        action.Mapping,
		Query:          action.Query,
		Path:           action.Path,
		TableName:      action.TableName,
		Sheet:          action.Sheet,
		ExportMode:     (*state.ExportMode)(action.ExportMode),
	}
	if action.Transformation != nil {
		n.Transformation = &state.Transformation{Source: action.Transformation.Source}
		switch action.Transformation.Language {
		case "JavaScript":
			n.Transformation.Language = state.JavaScript
		case "Python":
			n.Transformation.Language = state.Python
		}
	}

	// Add the filter to the notification and marshal it.
	var filter []byte
	if action.Filter != nil {
		n.Filter = &state.Filter{
			Logical:    state.FilterLogical(action.Filter.Logical),
			Conditions: make([]state.FilterCondition, len(action.Filter.Conditions)),
		}
		for i, condition := range action.Filter.Conditions {
			n.Filter.Conditions[i] = (state.FilterCondition)(condition)
		}
		filter, err = json.Marshal(action.Filter)
		if err != nil {
			return 0, err
		}
	}

	// Marshal the mapping.
	var mapping []byte
	if action.Mapping != nil {
		mapping, err = json.Marshal(action.Mapping)
		if err != nil {
			return 0, err
		}
	}

	// Generate a random identifier.
	n.ID, err = generateRandomID()
	if err != nil {
		return 0, err
	}

	// Marshal the input and the output schemas.
	rawInSchema, err := marshalSchema(action.InSchema)
	if err != nil {
		return 0, err
	}
	rawOutSchema, err := marshalSchema(action.OutSchema)
	if err != nil {
		return 0, err
	}

	// Marshal the mapping.
	if action.Mapping != nil {
		mapping, err = json.Marshal(action.Mapping)
		if err != nil {
			return 0, err
		}
	}

	// Handle the matching properties.
	if props := action.MatchingProperties; props != nil {
		n.MatchingProperties = &state.MatchingProperties{
			Internal: props.Internal,
			External: props.External,
		}
	}
	var transformation state.Transformation
	if n.Transformation != nil {
		name := transformationFunctionName(n.ID, n.Transformation.Language)
		version, err := this.apis.transformer.CreateFunction(ctx, name, n.Transformation.Source)
		if err != nil {
			return 0, err
		}
		n.Transformation.Version = version
		transformation = *n.Transformation
	}

	// Add the action.
	err = this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
		switch n.Target {
		case state.EventsTarget:
			switch typ := c.Connector().Type; typ {
			case state.MobileType, state.ServerType, state.WebsiteType:
				err = tx.QueryVoid(ctx, "SELECT FROM actions WHERE connection = $1 AND target = 'Events'", n.Connection)
				if err != sql.ErrNoRows {
					if err == nil {
						err = errors.Unprocessable(TargetAlreadyExist,
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
			"schedule_start, schedule_period, in_schema, out_schema, filter, mapping, transformation_source, " +
			"transformation_language, transformation_version, query, path, table_name, sheet, " +
			"export_mode, matching_properties_internal, matching_properties_external)\n" +
			" VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)"
		_, err := tx.Exec(ctx, query, n.ID, n.Connection, n.Target, n.EventType,
			n.Name, n.Enabled, n.ScheduleStart, n.SchedulePeriod, rawInSchema, rawOutSchema, string(filter), mapping,
			transformation.Source, transformation.Language, transformation.Version, n.Query, n.Path, n.TableName,
			n.Sheet, n.ExportMode, matchPropInternal, matchPropExternal)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) && postgres.ErrConstraintName(err) == "actions_connection_fkey" {
				err = errors.Unprocessable(ConnectionNotExist, "connection %d does not exist", n.Connection)
			}
			return err
		}
		return tx.Notify(ctx, n)
	})
	if err != nil {
		return 0, err
	}
	span.Log("action created successfully", "id", n.ID)

	return n.ID, nil
}

// AppUsers returns the users of an app connection and the cursor to get the
// next users. The returned cursor is empty if there are no other users.
func (this *Connection) AppUsers(ctx context.Context, schema types.Type, cursor string) ([]map[string]any, string, error) {

	this.apis.mustBeOpen()

	if this.connection.Connector().Type != state.AppType {
		return nil, "", errors.BadRequest("connection %d is not an app connection", this.connection.ID)
	}
	if !schema.Valid() {
		return nil, "", errors.BadRequest("schema is not valid")
	}
	var cur _connector.Cursor
	if cursor != "" {
		var err error
		cur, err = deserializeCursor(cursor)
		if err != nil {
			return nil, "", errors.BadRequest("cursor is malformed")
		}
	}

	app, err := this.openAppUsers()
	if err != nil {
		return nil, "", err
	}

	// Get the users.
	names := schema.PropertiesNames()
	properties := make([]types.Path, len(names))
	for i, name := range names {
		properties[i] = types.Path{name}
	}
	objects, next, err := app.Users(ctx, properties, cur)
	eof := err == io.EOF
	if err != nil && !eof {
		return nil, "", err
	}
	if len(objects) == 0 {
		return []map[string]any{}, "", nil
	}
	users := make([]map[string]any, len(objects))
	for i, object := range objects {
		user, err := normalize(object.Properties, schema)
		if err != nil {
			return nil, "", err
		}
		users[i] = user
	}
	if eof {
		return users, "", nil
	}

	// Build the cursor.
	last := objects[len(objects)-1]
	cursor, err = serializeCursor(_connector.Cursor{
		ID:        last.ID,
		Timestamp: last.Timestamp,
		Next:      next,
	})
	if err != nil {
		return nil, "", err
	}

	return users, cursor, nil
}

// CompletePath returns the complete representation of the given path, based
// on the connector that must be a storage. path cannot be empty, cannot be
// longer than 1024 runes, and must be UTF-8 encoded.
//
// If path is not valid for the storage connector, it returns an
// errors.UnprocessableError with code InvalidPath.
func (this *Connection) CompletePath(ctx context.Context, path string) (string, error) {
	this.apis.mustBeOpen()
	if path == "" {
		return "", errors.BadRequest("path is empty")
	}
	if !utf8.ValidString(path) {
		return "", errors.BadRequest("path is not UTF-8 encoded")
	}
	if n := utf8.RuneCountInString(path); n > 1024 {
		return "", errors.BadRequest("path is longer than 1024 runes")
	}
	c := this.connection
	connector := c.Connector()
	if connector.Type != state.StorageType {
		return "", errors.BadRequest("connection %d is not a storage connection", c.ID)
	}
	storage, err := this.openStorage()
	if err != nil {
		return "", err
	}
	path, err = storage.CompletePath(ctx, path)
	if err != nil {
		if err, ok := err.(_connector.InvalidPathError); ok {
			return "", errors.Unprocessable(InvalidPath, "%s", err)
		}
		return "", err
	}
	return path + c.Compression.Ext(), nil
}

// Delete deletes the connection.
//
// It returns an errors.NotFoundError error if the connection does not exist
// anymore.
func (this *Connection) Delete(ctx context.Context) error {
	this.apis.mustBeOpen()
	n := state.DeleteConnection{
		ID: this.connection.ID,
	}
	connector := this.connection.Connector()
	err := this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
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
func (this *Connection) ExecQuery(ctx context.Context, query string, limit int) ([]map[string]any, types.Type, error) {

	this.apis.mustBeOpen()

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

	// Execute the query.
	var err error
	query, err = compileActionQuery(query, limit)
	if err != nil {
		return nil, types.Type{}, errors.Unprocessable(QueryExecutionFailed, "query execution of connection %d failed: %w", c.ID, err)
	}
	database, err := this.openDatabase()
	if err != nil {
		return nil, types.Type{}, err
	}
	defer database.Close()
	rawRows, properties, err := database.Query(ctx, query)
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

	return rows, schema, nil
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
func (this *Connection) Executions(ctx context.Context) ([]*Execution, error) {
	this.apis.mustBeOpen()
	c := this.connection
	connector := c.Connector()
	switch connector.Type {
	case state.AppType, state.DatabaseType, state.FileType, state.StreamType:
	default:
		return nil, errors.BadRequest("connection %d cannot have executions, it's a %s connection",
			c.ID, strings.ToLower(connector.Type.String()))
	}
	executions := []*Execution{}
	err := this.apis.db.QueryScan(ctx,
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

// GenerateKey generates a new write key for the connection. The connection must
// be a source mobile, server or website connection.
//
// If the connection does not exist, it returns an errors.NotFoundError error.
// If the connection has already too many keys, it returns an
// errors.UnprocessableError error with code TooManyKeys.
func (this *Connection) GenerateKey(ctx context.Context) (string, error) {
	this.apis.mustBeOpen()
	c := this.connection
	connector := c.Connector()
	switch connector.Type {
	case state.MobileType, state.ServerType, state.WebsiteType:
	default:
		return "", errors.NotFound("connection %d is not a mobile, server or website", c.ID)
	}
	if c.Role != state.SourceRole {
		return "", errors.NotFound("connection %d is not a source", c.ID)
	}
	value, err := generateWriteKey()
	if err != nil {
		return "", err
	}
	n := state.AddConnectionKey{
		Connection:   c.ID,
		Value:        value,
		CreationTime: time.Now().UTC(),
	}
	err = this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
		var count int
		err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM connections_keys WHERE connection = $1", n.Connection).Scan(&count)
		if err != nil {
			return err
		}
		if count == maxKeysPerConnection {
			return errors.Unprocessable(TooManyKeys, "connection %d has already %d keys", n.Connection, maxKeysPerConnection)
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

// Records returns the records and the schema of the file with the given path
// for the connection, that must be a file connection. path must be UTF-8
// encoded with a length in range [1, 1024]. If the connection has multiple
// sheets, sheet is the sheet name and must be UTF-8 encoded with a length in
// range [1, 100], otherwise must be an empty string. limit limits the number of
// records to return and must be in range [0, 100].
//
// It returns an errors.UnprocessableError error with code
//
//   - NoStorage, if the connection does not have a storage.
//   - ReadFileFailed, if an error occurred reading the file.
func (this *Connection) Records(ctx context.Context, path, sheet string, limit int) ([]map[string]any, types.Type, error) {

	this.apis.mustBeOpen()

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
	if limit < 0 || limit > 100 {
		return nil, types.Type{}, errors.BadRequest("limit %d is not valid", limit)
	}
	// Validate the storage.
	if _, ok := c.Storage(); !ok {
		return nil, types.Type{}, errors.Unprocessable(NoStorage, "connection %d does not have a storage", c.ID)
	}

	// Connect to the file connector.
	file, err := this.openFile()
	if err != nil {
		return nil, types.Type{}, err
	}

	// Open the file.
	var r io.ReadCloser
	{
		storage, err := this.openStorage()
		if err != nil {
			return nil, types.Type{}, err
		}
		r, _, err = storage.Reader(ctx, path)
		if err != nil {
			return nil, types.Type{}, errors.Unprocessable(ReadFileFailed, "%w", err)
		}
		defer r.Close()
	}

	// Read the records.
	rw := newRecordWriter(c.ID, limit)
	err = file.Read(ctx, r, sheet, rw)
	if err != nil && err != errRecordStop {
		return nil, types.Type{}, errors.Unprocessable(ReadFileFailed, "%w", err)
	}
	if rw.columns == nil {
		return nil, types.Type{}, errors.Unprocessable(ReadFileFailed, "%w", errNoColumns)
	}
	schema := types.Object(rw.columns)

	return rw.records, schema, nil
}

// Rename renames the connection with the given new name.
// name must be between 1 and 100 runes long.
//
// It returns an errors.NotFoundError error if the connection does not exist
// anymore.
func (this *Connection) Rename(ctx context.Context, name string) error {
	this.apis.mustBeOpen()
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return errors.BadRequest("name %q is not valid", name)
	}
	if name == this.connection.Name {
		return nil
	}
	n := state.RenameConnection{
		Connection: this.connection.ID,
		Name:       name,
	}
	err := this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
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

// RevokeKey revokes the given write key of the connection. key cannot be empty
// and cannot be the unique key of the connection. The connection must be a
// source mobile, server or website connection.
//
// If the key does not exist, it returns an errors.NotFoundError error.
// If the key is the unique key of the server, it returns an
// errors.UnprocessableError error with code UniqueKey.
func (this *Connection) RevokeKey(ctx context.Context, key string) error {
	this.apis.mustBeOpen()
	if key == "" {
		return errors.BadRequest("key is empty")
	}
	if !isWriteKey(key) {
		return errors.BadRequest("key %q is malformed", key)
	}
	c := this.connection
	connector := c.Connector()
	switch connector.Type {
	case state.MobileType, state.ServerType, state.WebsiteType:
	default:
		return errors.BadRequest("connection %d is not a mobile, server or website", c.ID)
	}
	if c.Role != state.SourceRole {
		return errors.BadRequest("connection %d is not a source", c.ID)
	}
	n := state.RevokeConnectionKey{
		Connection: c.ID,
		Value:      key,
	}
	err := this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
		var count int
		err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM connections_keys WHERE connection = $1", n.Connection).Scan(&count)
		if err != nil {
			return err
		}
		if count == 1 {
			return errors.Unprocessable(UniqueKey, "key cannot be revoked because it's the unique key of the connection")
		}
		result, err := tx.Exec(ctx, "DELETE FROM connections_keys WHERE connection = $1 AND value = $2", n.Connection, n.Value)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.Unprocessable(KeyNotExist, "key %q does not exist", key)
		}
		return tx.Notify(ctx, n)
	})

	return err
}

// PreviewSendEvent returns a preview of an event as it would be sent to an app.
// The connection must be a destination app connection, and it is expected to
// have an event type named eventType. If the event type has a schema, then
// either the mapping or the transformation to apply to the event must be
// present.
//
// It returns an errors.UnprocessableError error with code:
//   - EventTypeNotExist, if the event type does not exist for the connection.
//   - LanguageNotSupported, if the transformation language is not supported.
//   - TransformationFailed if the transformation fails due to an error in the
//     executed function.
func (this *Connection) PreviewSendEvent(ctx context.Context, eventType string, event *ObservedEvent, mapping map[string]string, transformation *Transformation) ([]byte, error) {

	this.apis.mustBeOpen()

	c := this.connection

	if c.Connector().Type != state.AppType {
		return nil, errors.BadRequest("connection %d is not an app connection", c.ID)
	}
	if c.Role != state.DestinationRole {
		return nil, errors.BadRequest("connection %d is not a destination", c.ID)
	}
	if !c.Connector().Targets.Contains(state.EventsTarget) {
		return nil, errors.BadRequest("connection %d does not support events", c.ID)
	}
	if eventType == "" {
		return nil, errors.BadRequest("eventType is empty")
	}
	if event == nil {
		return nil, errors.BadRequest("event is missing")
	}
	if event.Header == nil {
		return nil, errors.BadRequest("event header is missing")
	}
	if mapping != nil && transformation != nil {
		return nil, errors.BadRequest("mapping and transformation cannot both be present")
	}

	// Parse the event.
	ev, err := this.apis.events.ParseObservedEvent(&events.ObservedEvent{
		Source: event.Source,
		Header: &events.EventHeader{
			ReceivedAt: event.Header.ReceivedAt,
			RemoteAddr: event.Header.RemoteAddr,
			Method:     event.Header.Method,
			Proto:      event.Header.Proto,
			URL:        event.Header.URL,
		},
		Data: event.Data,
	})
	if err != nil {
		return nil, errors.BadRequest("event is not valid: %s", err)
	}

	app, err := this.openAppEvents()
	if err != nil {
		return nil, fmt.Errorf("cannot connect to the connector: %s", err)
	}

	// Get the event type.
	eventTypes, err := app.EventTypes(ctx)
	if err != nil {
		return nil, err
	}
	var found bool
	var outSchema types.Type
	for _, t := range eventTypes {
		if t.ID == eventType {
			outSchema = t.Schema
			found = true
			break
		}
	}
	if !found {
		return nil, errors.Unprocessable(EventTypeNotExist, "connection %d does not have event type %q", c.ID, eventType)
	}

	var data map[string]any

	// If the event type has a schema, apply the mapping or the transformation.
	if outSchema.Valid() {

		inSchema := events.Schema

		// Validate the mapping and the transformation.
		switch {
		case mapping != nil:
			for path, expr := range mapping {
				outPath, err := types.ParsePropertyPath(path)
				if err != nil {
					return nil, errors.BadRequest("output mapped property %q is not valid", path)
				}
				p, err := outSchema.PropertyByPath(outPath)
				if err != nil {
					err := err.(types.PathNotExistError)
					return nil, errors.BadRequest("output mapped property %s not found in output schema", err.Path)
				}
				_, err = mapexp.Compile(expr, inSchema, p.Type, p.Nullable)
				if err != nil {
					return nil, errors.BadRequest("invalid expression mapped to %s: %s", path, err)
				}
			}
		case transformation != nil:
			if transformation.Source == "" {
				return nil, errors.BadRequest("transformation source is empty")
			}
			tr := this.apis.transformer
			switch transformation.Language {
			case "JavaScript":
				if tr == nil || !tr.SupportLanguage(state.JavaScript) {
					return nil, errors.Unprocessable(LanguageNotSupported, "Javascript transformation language  is not supported")
				}
			case "Python":
				if tr == nil || !tr.SupportLanguage(state.Python) {
					return nil, errors.Unprocessable(LanguageNotSupported, "Python transformation language is not supported")
				}
			case "":
				return nil, errors.BadRequest("transformation language is empty")
			default:
				return nil, errors.BadRequest("transformation language %q is not valid", transformation.Language)
			}
		default:
			return nil, errors.BadRequest("mapping (or transformation) is required")
		}

		// Create a temporary transformer.
		var tr *state.Transformation
		var transformer transformers.Transformer
		if transformation != nil {
			tr = &state.Transformation{
				Source:  transformation.Source,
				Version: "1", // no matter the version, it will be overwritten by the temporary transformation.
			}
			name := "temp-" + uuid.NewString()
			switch transformation.Language {
			case "JavaScript":
				name += ".js"
				tr.Language = state.JavaScript
			case "Python":
				name += ".py"
				tr.Language = state.Python
			}
			transformer = newTemporaryTransformer(name, transformation.Source, this.apis.transformer)
		}

		// Transform the data.
		action := 1 // no matter the action, it will be overwritten by the temporary transformation.
		m, err := mappings.New(inSchema, outSchema, mapping, tr, action, transformer, false)
		if err != nil {
			return nil, err
		}
		data, err = m.Apply(ctx, ev.MapEvent())
		if err != nil {
			if err, ok := err.(mappings.Error); ok {
				return nil, errors.Unprocessable(TransformationFailed, err.Error())
			}
			return nil, err
		}

	} else {

		if mapping != nil {
			return nil, errors.BadRequest("mapping is not allowed the event type %q does not have a schema", eventType)
		}
		if transformation != nil {
			return nil, errors.BadRequest("transformation is not allowed because the event type %q does not have a schema", eventType)
		}

	}

	return app.SendEventPreview(ctx, eventType, ev.ConnectorEvent(), data)
}

// SetStatus sets the status of the connection.
func (this *Connection) SetStatus(ctx context.Context, enabled bool) error {
	this.apis.mustBeOpen()
	if enabled == this.Enabled {
		return nil
	}
	n := state.SetConnectionStatus{
		Connection: this.connection.ID,
		Enabled:    enabled,
	}
	err := this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
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
// with code EventNotExist.
func (this *Connection) ServeUI(ctx context.Context, event string, values []byte) ([]byte, error) {

	this.apis.mustBeOpen()

	c := this.connection
	connector := c.Connector()

	var err error
	var connection any

	switch connector.Type {
	case state.AppType:
		connection, err = this.openApp()
	case state.DatabaseType:
		connection, err = this.openDatabase()
	case state.FileType:
		connection, err = this.openFile()
	case state.MobileType:
		connection, err = this.openMobile()
	case state.ServerType:
		connection, err = this.openServer()
	case state.StorageType:
		connection, err = this.openStorage()
	case state.StreamType:
		connection, err = this.openStream()
	case state.WebsiteType:
		connection, err = this.openWebsite()
	}

	if err != nil {
		return nil, err
	}
	if c, ok := connection.(io.Closer); ok {
		defer c.Close()
	}
	connectionUI, ok := connection.(_connector.UI)
	if !ok {
		return nil, errors.BadRequest("connector %d does not have a UI", c.ID)
	}

	// TODO: check and delete alternative fieldsets keys that have 'null' value
	// before saving to database
	form, alert, err := connectionUI.ServeUI(ctx, event, values)
	if err != nil {
		if err == ui.ErrEventNotExist {
			err = errors.Unprocessable(EventNotExist, "UI event %q does not exist for %s connector",
				event, connector.Name)
		}
		return nil, err
	}

	return marshalUIFormAlert(form, alert, ui.Role(c.Role))
}

// SetStorage sets the storage and the compression of the connection. The
// connection must be a file connection. storage is the storage connection. The
// connection and the storage must have the same role. As a special case, if the
// storage argument is 0, compression can only be NoCompression and the current
// storage of the file, if there is one, will be removed.
//
// If the connection does not exist anymore, it returns an errors.NotFoundError
// error.
// If the storage does not exist, it returns an errors.UnprocessableError error
// with code StorageNotExist.
func (this *Connection) SetStorage(ctx context.Context, storage int, compression Compression) error {

	this.apis.mustBeOpen()

	if storage < 0 || storage > maxInt32 {
		return errors.BadRequest("storage identifier %d is not valid", storage)
	}
	switch compression {
	case NoCompression, ZipCompression, GzipCompression, SnappyCompression:
	default:
		return errors.BadRequest("compression %q is not valid", compression)
	}
	if storage == 0 && compression != NoCompression {
		return errors.BadRequest("compression requires a storage")
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
			return errors.Unprocessable(StorageNotExist, "storage %d does not exist", storage)
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
	} else if compression != NoCompression {
		return errors.BadRequest("file cannot be compressed without a storage")
	}

	n := state.SetConnectionStorage{
		Connection:  c.ID,
		Storage:     storage,
		Compression: state.Compression(compression),
	}

	err := this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE connections SET storage = NULLIF($1, 0), compression = $2\n"+
			"WHERE id = $3", n.Storage, n.Compression, n.Connection)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				if postgres.ErrConstraintName(err) == "connections_storage_fkey" {
					err = errors.Unprocessable(StorageNotExist, "storage %d does not exist", storage)
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
func (this *Connection) Sheets(ctx context.Context, path string) ([]string, error) {
	this.apis.mustBeOpen()
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
	f, err := this.openFile()
	if err != nil {
		return nil, err
	}
	file, ok := f.(_connector.Sheets)
	if !ok {
		return nil, errors.BadRequest("file connection %d does not support multiple sheet", c.ID)
	}
	if _, ok := c.Storage(); !ok {
		return nil, errors.Unprocessable(NoStorage, "file connection %d does not have a storage", c.ID)
	}
	storage, err := this.openStorage()
	if err != nil {
		return nil, err
	}
	r, _, err := storage.Reader(ctx, path)
	if err != nil {
		return nil, errors.Unprocessable(ReadFileFailed, "%w", err)
	}
	defer r.Close()
	sheets, err := file.Sheets(ctx, r)
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
func (this *Connection) Stats(ctx context.Context) (*ConnectionsStats, error) {
	this.apis.mustBeOpen()
	now := time.Now().UTC()
	toSlot := statsTimeSlot(now)
	fromSlot := toSlot - 23
	stats := &ConnectionsStats{
		Users: [24]int{},
	}
	query := "SELECT time_slot, users\nFROM connections_stats\nWHERE connection = $1 AND time_slot BETWEEN $2 AND $3"
	err := this.apis.db.QueryScan(ctx, query, this.connection.ID, fromSlot, toSlot, func(rows *postgres.Rows) error {
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

// TableSchema returns the schema of the given table for the connection.
// connection must be a destination database connection, and table must be UTF-8
// encoded with a length in range [1, 1024].
//
// It returns an error.Unprocessable error with code InvalidTable if the table
// does not contain an unsigned 32-bit column named "id" or if there are no
// other columns apart from "id".
func (this *Connection) TableSchema(ctx context.Context, table string) (types.Type, error) {
	this.apis.mustBeOpen()
	c := this.connection
	connector := c.Connector()
	if connector.Type != state.DatabaseType {
		return types.Type{}, errors.BadRequest("connection %d is not a database", c.ID)
	}
	if c.Role != state.DestinationRole {
		return types.Type{}, errors.BadRequest("database %d is not a destination", c.ID)
	}
	if table == "" || utf8.RuneCountInString(table) > 1024 {
		return types.Type{}, errors.BadRequest("table name is not valid")
	}
	database, err := this.openDatabase()
	if err != nil {
		return types.Type{}, err
	}
	columns, err := database.Columns(ctx, table)
	_ = database.Close()
	if err != nil {
		return types.Type{}, err
	}
	var hasID bool
	for i, column := range columns {
		if column.Name == "id" {
			if column.Type.PhysicalType() != types.PtInt {
				return types.Type{}, errors.Unprocessable(InvalidTable, "column \"id\" of table %q is not a signed 32 bit integer", table)
			}
			columns = slices.Delete(columns, i, i+1)
			hasID = true
			break
		}
	}
	if !hasID {
		return types.Type{}, errors.Unprocessable(InvalidTable, "table %q does not have a signed 32-bit integer column named \"id\"", table)
	}
	if len(columns) == 0 {
		return types.Type{}, errors.Unprocessable(InvalidTable, "table %q only has the \"id\" column and no additional columns", table)
	}
	properties, err := datastore.ColumnsToProperties(columns)
	if err != nil {
		return types.Type{}, err
	}
	schema, err := types.ObjectOf(properties)
	if err != nil {
		return types.Type{}, err
	}
	return schema, nil
}

// ActionType represents an action type.
type ActionType struct {
	Name          string
	Description   string
	Target        ActionTarget
	EventType     *string
	MissingSchema bool
}

// Keys returns the write keys of the connection.
// The connection must be a source mobile, server or website connection.
func (this *Connection) Keys() ([]string, error) {
	this.apis.mustBeOpen()
	c := this.connection
	switch c.Connector().Type {
	case state.MobileType, state.ServerType, state.WebsiteType:
	default:
		return nil, errors.BadRequest("connection %d is not a mobile, server or website", c.ID)
	}
	if c.Role != state.SourceRole {
		return nil, errors.BadRequest("connection %d is not a source", c.ID)
	}
	return slices.Clone(c.Keys), nil
}

// actionTypes returns the action types for the connection.
//
// Refer to the specifications in the file "connector/Actions support.md" for
// more details.
//
// It returns an errors.UnprocessableError error with code
//
//   - FetchSchemaFailed, if an error occurred fetching the schema.
func (this *Connection) actionTypes(ctx context.Context) ([]ActionType, error) {
	var actionTypes []ActionType
	c := this.connection
	connector := c.Connector()
	wsSchemas := c.Workspace().Schemas
	targets := connector.Targets
	if targets.Contains(state.UsersTarget) {
		switch typ := c.Connector().Type; typ {
		case
			state.AppType,
			state.DatabaseType,
			state.FileType:
			var name, description string
			var missingSchema bool
			if c.Role == state.SourceRole {
				name = "Import " + connector.TermForUsers
				description = "Import the " + connector.TermForUsers
				if connector.TermForUsers != "users" {
					description += " as users"
				}
				description += " from " + connector.Name
				missingSchema = wsSchemas["users_identities"] == nil
			} else {
				name = "Export " + connector.TermForUsers
				description = "Export the users "
				if connector.TermForUsers != "users" {
					description += " as " + connector.TermForUsers
				}
				description += " to " + connector.Name
				missingSchema = wsSchemas["users"] == nil
			}
			at := ActionType{
				Name:          name,
				Description:   description,
				Target:        UsersTarget,
				MissingSchema: missingSchema,
			}
			actionTypes = append(actionTypes, at)
		case
			state.MobileType,
			state.ServerType,
			state.WebsiteType:
			if c.Role == state.SourceRole {
				at := ActionType{
					Name:          "Import users",
					Description:   "Import users from the events of the " + connector.Name,
					Target:        UsersTarget,
					MissingSchema: wsSchemas["users_identities"] == nil,
				}
				actionTypes = append(actionTypes, at)
			}
		}
	}
	if targets.Contains(state.GroupsTarget) {
		switch typ := c.Connector().Type; typ {
		case
			state.AppType,
			state.DatabaseType,
			state.FileType:
			var name, description string
			var missingSchema bool
			if c.Role == state.SourceRole {
				name = "Import " + connector.TermForGroups
				description = "Import the " + connector.TermForGroups
				if connector.TermForGroups != "groups" {
					description += " as groups"
				}
				description += " from " + connector.Name
				missingSchema = wsSchemas["groups_identities"] == nil
			} else {
				name = "Export " + connector.TermForGroups
				description = "Export the groups "
				if connector.TermForGroups != "groups" {
					description += " as " + connector.TermForGroups
				}
				description += " to " + connector.Name
				missingSchema = wsSchemas["groups"] == nil
			}
			at := ActionType{
				Name:          name,
				Description:   description,
				Target:        GroupsTarget,
				MissingSchema: missingSchema,
			}
			actionTypes = append(actionTypes, at)
		case
			state.MobileType,
			state.ServerType,
			state.WebsiteType:
			if c.Role == state.SourceRole {
				at := ActionType{
					Name:          "Import groups",
					Description:   "Import groups from the events of the " + connector.Name,
					Target:        GroupsTarget,
					MissingSchema: wsSchemas["groups"] == nil,
				}
				actionTypes = append(actionTypes, at)
			}
		}
	}
	if targets.Contains(state.EventsTarget) {
		switch typ := c.Connector().Type; typ {
		case state.MobileType, state.ServerType, state.WebsiteType:
			if c.Role == state.SourceRole {
				description := "Collect events from the "
				switch typ {
				case state.MobileType:
					description += "mobile app"
				case state.ServerType:
					description += "server"
				case state.WebsiteType:
					description += "website"
				}
				at := ActionType{
					Name:        "Collect events",
					Description: description,
					Target:      EventsTarget,
				}
				actionTypes = append(actionTypes, at)
			}
		case state.AppType:
			eventTypes, err := this.fetchEventTypes(ctx)
			if err != nil {
				return nil, errors.Unprocessable(FetchSchemaFailed, "an error occurred fetching the schema: %w", err)
			}
			for _, et := range eventTypes {
				id := et.ID
				actionTypes = append(actionTypes, ActionType{
					Name:        et.Name,
					Description: et.Description,
					Target:      EventsTarget,
					EventType:   &id,
				})
			}
		}
	}
	if actionTypes == nil {
		actionTypes = []ActionType{}
	}
	return actionTypes, nil
}

// openApp opens an app connection.
func (this *Connection) openApp() (_connector.AppConnection, error) {
	c := this.connection
	var resourceID int
	var resourceCode string
	if r, ok := c.Resource(); ok {
		resourceID = r.ID
		resourceCode = r.Code
	}
	app, err := _connector.RegisteredApp(c.Connector().Name).Open(&_connector.AppConfig{
		Role:        _connector.Role(c.Role),
		Settings:    c.Settings,
		SetSettings: this.setSettingsFunc(),
		Resource:    resourceCode,
		HTTPClient:  this.apis.http.ConnectionClient(c.ID),
		Region:      _connector.PrivacyRegion(c.Workspace().PrivacyRegion),
		WebhookURL:  webhookURL(c, resourceID),
	})
	return app, err
}

// openAppEvents opens an app events connection.
func (this *Connection) openAppEvents() (_connector.AppEventsConnection, error) {
	app, err := this.openApp()
	if err != nil {
		return nil, err
	}
	return app.(_connector.AppEventsConnection), nil
}

// openAppUsers opens an app users connection.
func (this *Connection) openAppUsers() (_connector.AppUsersConnection, error) {
	app, err := this.openApp()
	if err != nil {
		return nil, err
	}
	return app.(_connector.AppUsersConnection), nil
}

// openDatabase opens a database connection.
//
// It is the caller's responsibility to call the Close method on the returned
// value.
func (this *Connection) openDatabase() (_connector.DatabaseConnection, error) {
	c := this.connection
	database, err := _connector.RegisteredDatabase(c.Connector().Name).Open(&_connector.DatabaseConfig{
		Role:        _connector.Role(c.Role),
		Settings:    c.Settings,
		SetSettings: this.setSettingsFunc(),
	})
	return database, err
}

// openFile opens a file connection.
func (this *Connection) openFile() (_connector.FileConnection, error) {
	c := this.connection
	file, err := _connector.RegisteredFile(c.Connector().Name).Open(&_connector.FileConfig{
		Role:        _connector.Role(c.Role),
		Settings:    c.Settings,
		SetSettings: this.setSettingsFunc(),
	})
	return file, err
}

// openMobile opens a mobile connection.
func (this *Connection) openMobile() (_connector.MobileConnection, error) {
	c := this.connection
	mobile, err := _connector.RegisteredMobile(c.Connector().Name).Open(&_connector.MobileConfig{
		Role:        _connector.Role(c.Role),
		Settings:    c.Settings,
		SetSettings: this.setSettingsFunc(),
	})
	return mobile, err
}

// openServer opens a server connection.
func (this *Connection) openServer() (_connector.ServerConnection, error) {
	c := this.connection
	server, err := _connector.RegisteredServer(c.Connector().Name).Open(&_connector.ServerConfig{
		Role:        _connector.Role(c.Role),
		Settings:    c.Settings,
		SetSettings: this.setSettingsFunc(),
	})
	return server, err
}

// openStorage opens a storage connection.
func (this *Connection) openStorage() (_connector.StorageConnection, error) {
	c := this.connection
	if c.Connector().Type == state.FileType {
		c, _ = c.Storage()
	}
	storage, err := _connector.RegisteredStorage(c.Connector().Name).Open(&_connector.StorageConfig{
		Role:        _connector.Role(c.Role),
		Settings:    c.Settings,
		SetSettings: this.setSettingsFunc(),
	})
	return storage, err
}

// openStream opens a stream connection.
//
// It is the caller's responsibility to call the Close method on the returned
// value.
func (this *Connection) openStream() (_connector.StreamConnection, error) {
	c := this.connection
	database, err := _connector.RegisteredStream(c.Connector().Name).Open(&_connector.StreamConfig{
		Role:        _connector.Role(c.Role),
		Settings:    c.Settings,
		SetSettings: this.setSettingsFunc(),
	})
	return database, err
}

// openWebsite opens a website connection.
func (this *Connection) openWebsite() (_connector.WebsiteConnection, error) {
	c := this.connection
	website, err := _connector.RegisteredWebsite(c.Connector().Name).Open(&_connector.WebsiteConfig{
		Role:        _connector.Role(c.Role),
		Settings:    c.Settings,
		SetSettings: this.setSettingsFunc(),
	})
	return website, err
}

var errRecordStop = errors.New("stop record")

// isWriteKey reports whether key can be a write key.
func isWriteKey(key string) bool {
	if len(key) != 32 {
		return false
	}
	_, err := base62.DecodeString(key)
	return err == nil
}

// generateWriteKey generates a write key in its base62 form.
func generateWriteKey() (string, error) {
	key := make([]byte, 24)
	_, err := rand.Read(key)
	if err != nil {
		return "", errors.New("cannot generate a write key")
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

// Compression represents the compression of a file connection.
type Compression string

const (
	NoCompression     Compression = ""
	ZipCompression    Compression = "Zip"
	GzipCompression   Compression = "Gzip"
	SnappyCompression Compression = "Snappy"
)

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

// fetchAppSchema fetches the schema of an app connection for the given target
// and eventType.
//
// It returns an errors.UnprocessableError error with code:
//   - EventTypeNotExist, if the event type does not exist.
func (this *Connection) fetchAppSchema(ctx context.Context, target state.ActionTarget, eventType string) (types.Type, error) {

	app, err := this.openApp()
	if err != nil {
		return types.Type{}, fmt.Errorf("cannot connect to the connector: %s", err)
	}

	c := this.connection

	var schema types.Type

	switch target {
	case state.EventsTarget:
		if eventType != "" {
			eventTypes, err := app.(_connector.AppEventsConnection).EventTypes(ctx)
			if err != nil {
				return types.Type{}, err
			}
			var found bool
			for _, t := range eventTypes {
				if t.ID == eventType {
					schema = t.Schema
					found = true
					break
				}
			}
			if !found {
				return types.Type{}, errors.Unprocessable(EventTypeNotExist, "connection %d does not have event type %q", c.ID, eventType)
			}
		}
	case state.UsersTarget:
		schema, err = app.(_connector.AppUsersConnection).UserSchema(ctx)
		if err != nil {
			return types.Type{}, err
		}
		if !schema.Valid() {
			return types.Type{}, fmt.Errorf("connection %d returned an invalid user schema", c.ID)
		}
		schema = schema.AsRole(types.Role(c.Role))
		if !schema.Valid() {
			return types.Type{}, fmt.Errorf("connection has returned a schema without %s properties", strings.ToLower(c.Role.String()))
		}
	case state.GroupsTarget:
		schema, err = app.(_connector.AppGroupsConnection).GroupSchema(ctx)
		if err != nil {
			return types.Type{}, err
		}
		if !schema.Valid() {
			return types.Type{}, fmt.Errorf("connection %d returned an invalid group schema", c.ID)
		}
		schema = schema.AsRole(types.Role(c.Role))
		if !schema.Valid() {
			return types.Type{}, fmt.Errorf("connection has returned a schema without %s properties", strings.ToLower(c.Role.String()))
		}
	}
	return schema, nil
}

// fetchEventTypes fetches the event types for the connection.
func (this *Connection) fetchEventTypes(ctx context.Context) ([]*_connector.EventType, error) {
	app, err := this.openAppEvents()
	if err != nil {
		return nil, fmt.Errorf("cannot connect to the connector: %s", err)
	}
	return app.EventTypes(ctx)
}

// setSettingsFunc returns a connector.SetSettingsFunc function that sets the
// settings for the connection.
func (this *Connection) setSettingsFunc() _connector.SetSettingsFunc {
	return func(ctx context.Context, settings []byte) error {
		return setSettings(ctx, this.apis.db, this.connection.ID, settings)
	}
}

// updateConnectionsStats updates the statistics about the connection.
func (this *Connection) updateConnectionsStats(ctx context.Context) error {
	connection := this.connection.ID
	_, err := this.apis.db.Exec(ctx, "INSERT INTO connections_stats AS cs (connection, time_slot, users)\n"+
		"VALUES ($1, $2, 1)\n"+
		"ON CONFLICT (connection, time_slot) DO UPDATE SET users = cs.users + 1",
		connection, statsTimeSlot(time.Now()))
	return err
}

// validateActionToSet validates the action to set (when adding or setting an
// action) for the given target and event type.
//
// Refer to the specifications in the file "connector/Actions support.md" for
// more details.
//
// It returns an errors.UnprocessableError error with code
//   - LanguageNotSupported, if the transformation language is not supported.
//   - MappingOverAnonymousIdentifier, if the action maps over an anonymous
//     identifier.
func (this *Connection) validateActionToSet(action ActionToSet, target state.ActionTarget, eventType string) error {

	// First, do formal validations.

	// Validate the name.
	if action.Name == "" {
		return errors.BadRequest("name is empty")
	}
	if !utf8.ValidString(action.Name) {
		return errors.BadRequest("name is not UTF-8 encoded")
	}
	if n := utf8.RuneCountInString(action.Name); n > 60 {
		return errors.BadRequest("name is longer than 60 runes")
	}
	// Validate the schemas.
	if action.InSchema.Valid() && action.InSchema.PhysicalType() != types.PtObject {
		return errors.BadRequest("input schema, if provided, must be an object")
	}
	if action.OutSchema.Valid() && action.OutSchema.PhysicalType() != types.PtObject {
		return errors.BadRequest("out schema, if provided, must be an object")
	}
	// Validate the filter.
	var inPaths []types.Path
	if action.Filter != nil {
		if !action.InSchema.Valid() {
			return errors.BadRequest("input schema is required by the filter")
		}
		var err error
		inPaths, err = validateFilter(action.Filter, action.InSchema)
		if err != nil {
			return errors.BadRequest("filter is not valid: %w", err)
		}
	}
	// An action cannot have both mappings and transformations.
	if action.Mapping != nil && action.Transformation != nil {
		return errors.BadRequest("action cannot have both mappings and transformation")
	}
	// Validate the mapping.
	var outPaths []types.Path
	if action.Mapping != nil && len(action.Mapping) > 0 {
		if !action.InSchema.Valid() {
			return errors.BadRequest("input schema is required by the mapping")
		}
		if !action.OutSchema.Valid() {
			return errors.BadRequest("output schema is required by the mapping")
		}
		for path, expr := range action.Mapping {
			outPath, err := types.ParsePropertyPath(path)
			if err != nil {
				return errors.BadRequest("output mapped property %q is not valid", path)
			}
			outPaths = append(outPaths, outPath)
			p, err := action.OutSchema.PropertyByPath(outPath)
			if err != nil {
				err := err.(types.PathNotExistError)
				return errors.BadRequest("output mapped property %s not found in output schema", err.Path)
			}
			expr, err := mapexp.Compile(expr, action.InSchema, p.Type, p.Nullable)
			if err != nil {
				return errors.BadRequest("invalid expression mapped to %s: %s", path, err)
			}
			inPaths = append(inPaths, expr.Properties()...)
		}
	}
	// Validate the transformation.
	if action.Transformation != nil {
		if !action.InSchema.Valid() {
			return errors.BadRequest("input schema is required by the transformation")
		}
		if !action.OutSchema.Valid() {
			return errors.BadRequest("output schema is required by the transformation")
		}
		if action.Transformation.Source == "" {
			return errors.BadRequest("transformation source is empty")
		}
		tr := this.apis.transformer
		switch action.Transformation.Language {
		case "JavaScript":
			if tr == nil || !tr.SupportLanguage(state.JavaScript) {
				return errors.Unprocessable(LanguageNotSupported, "Javascript transformation language is not supported")
			}
		case "Python":
			if tr == nil || !tr.SupportLanguage(state.Python) {
				return errors.Unprocessable(LanguageNotSupported, "Python transformation language is not supported")
			}
		case "":
			return errors.BadRequest("transformation language is empty")
		default:
			return errors.BadRequest("transformation language %q is not valid", action.Transformation.Language)
		}
	}
	// Ensure that every property in the input and output schemas have been
	// mapped, unless the action has a transformation; in that case, we do not
	// know which properties have been mapped, so this check cannot be done.
	if inPaths != nil && action.Transformation == nil {
		if props := unmappedProperties(action.InSchema, inPaths); props != nil {
			return errors.BadRequest("input schema contains unmapped properties: %s", strings.Join(props, ", "))
		}
		if props := unmappedProperties(action.OutSchema, outPaths); props != nil {
			return errors.BadRequest("output schema contains unmapped properties: %s", strings.Join(props, ", "))
		}
	}
	// Validate the path.
	if action.Path != "" {
		if !utf8.ValidString(action.Path) {
			return errors.BadRequest("path is not UTF-8 encoded")
		}
		if n := utf8.RuneCountInString(action.Path); n > 1024 {
			return errors.BadRequest("path is longer than 1024 runes")
		}
	}
	// Validate the table name.
	if action.TableName != "" {
		if !utf8.ValidString(action.TableName) {
			return errors.BadRequest("table name is not UTF-8 encoded")
		}
		if n := utf8.RuneCountInString(action.TableName); n > 1024 {
			return errors.BadRequest("table name is longer than 1024 runes")
		}
	}
	// Validate the sheet.
	if action.Sheet != "" {
		if !utf8.ValidString(action.Sheet) {
			return errors.BadRequest("sheet is not UTF-8 encoded")
		}
		if n := utf8.RuneCountInString(action.Sheet); n > 100 {
			return errors.BadRequest("sheet is longer than 100 runes")
		}
	}
	// Validate the export options.
	if action.ExportMode != nil {
		switch *action.ExportMode {
		case CreateOnly, UpdateOnly, CreateOrUpdate:
		default:
			return errors.BadRequest("export mode %q is not valid", *action.ExportMode)
		}
	}
	if action.MatchingProperties != nil {
		props := *action.MatchingProperties
		if !types.IsValidPropertyName(props.Internal) {
			return errors.BadRequest("internal matching property %q is not a valid property name", props.Internal)
		}
		if !types.IsValidPropertyName(props.External) {
			return errors.BadRequest("external matching property %q is not a valid property name", props.External)
		}
	}

	// Second, do validations based on the workspace and the connection.

	c := this.connection
	connector := c.Connector()
	ws := this.connection.Workspace()

	// When importing users, ensure that there are no mappings over the
	// anonymous identifiers of the workspace.
	if importingUsers := c.Role == state.SourceRole && target == state.UsersTarget; importingUsers {
		var tOutProps []string
		if action.Transformation != nil {
			tOutProps = action.OutSchema.PropertiesNames()
		}
		for _, p := range ws.AnonymousIdentifiers.Priority {
			_, ok := action.Mapping[p]
			if ok || slices.Contains(tOutProps, p) {
				return errors.Unprocessable(MappingOverAnonymousIdentifier, "cannot map over the property %s because it is an anonymous identifier", p)
			}
		}
	}

	// Check if the query is allowed.
	if needsQuery := connector.Type == state.DatabaseType && c.Role == state.SourceRole; needsQuery {
		if action.Query == "" {
			return errors.BadRequest("query cannot be empty for database actions")
		}
	} else {
		if action.Query != "" {
			return errors.BadRequest("%s actions cannot have a query", connector.Type)
		}
	}

	// Check if the filters are allowed.
	targetUsersOrGroups := target == state.UsersTarget || target == state.GroupsTarget
	var filtersAllowed bool
	switch connector.Type {
	case state.AppType:
		filtersAllowed = c.Role == state.DestinationRole
	case state.DatabaseType:
		filtersAllowed = c.Role == state.DestinationRole
	case state.FileType:
		filtersAllowed = targetUsersOrGroups && c.Role == state.DestinationRole
	}
	if action.Filter != nil && !filtersAllowed {
		return errors.BadRequest("filters are not allowed")
	}

	// Check if the path and the sheet are allowed.
	if connector.Type == state.FileType {
		if action.Path == "" {
			return errors.BadRequest("path cannot be empty for file actions")
		}
		if connector.HasSheets && action.Sheet == "" {
			return errors.BadRequest("sheet cannot be empty because connection %d has sheets", c.ID)
		}
		if !connector.HasSheets && action.Sheet != "" {
			return errors.BadRequest("connection %d does not have sheets", c.ID)
		}
	} else {
		if action.Path != "" {
			return errors.BadRequest("%s actions cannot have a path", connector.Type)
		}
		if action.Sheet != "" {
			return errors.BadRequest("%s actions cannot have a sheet", connector.Type)
		}
	}

	// Check if the table name is allowed.
	needsTableName := connector.Type == state.DatabaseType && c.Role == state.DestinationRole
	if needsTableName && action.TableName == "" {
		return errors.BadRequest("table name cannot be empty for destination database actions")
	} else if !needsTableName && action.TableName != "" {
		return errors.BadRequest("table name is not allowed")
	}

	// Check if the export options are needed.
	needsExportOptions := connector.Type == state.AppType &&
		c.Role == state.DestinationRole &&
		targetUsersOrGroups
	if needsExportOptions {
		if action.ExportMode == nil {
			return errors.BadRequest("export mode cannot be nil")
		}
		if action.MatchingProperties == nil {
			return errors.BadRequest("matching properties cannot be nil")
		}
	} else {
		if action.ExportMode != nil {
			return errors.BadRequest("export mode must be nil")
		}
		if action.MatchingProperties != nil {
			return errors.BadRequest("matching properties must be nil")
		}
	}

	// Validate empty mappings; they are allowed only when sending events,
	// because the user may want to leave every property of the output schema
	// unmapped.
	if action.Mapping != nil && len(action.Mapping) == 0 && target != state.EventsTarget {
		return errors.BadRequest("action has a mapping with no mapped properties")
	}

	// Check if the mapping (or the transformation) is mandatory, and if the
	// transformation is allowed.
	var mappingIsMandatory bool
	var transformationIsAllowed bool
	switch connector.Type {
	case state.AppType:
		if c.Role == state.DestinationRole && target == state.EventsTarget {
			schema, err := this.fetchAppSchema(context.Background(), target, eventType)
			if err != nil {
				return err
			}
			mappingIsMandatory = schema.Valid()
			transformationIsAllowed = true
		} else {
			mappingIsMandatory = targetUsersOrGroups
			transformationIsAllowed = true
		}
	case state.MobileType, state.ServerType, state.WebsiteType:
		mappingIsMandatory = targetUsersOrGroups
		transformationIsAllowed = false
	case
		state.DatabaseType,
		state.FileType:
		mappingIsMandatory = c.Role == state.SourceRole && targetUsersOrGroups
		transformationIsAllowed = mappingIsMandatory
	}
	if mappingIsMandatory && action.Mapping == nil && action.Transformation == nil {
		if transformationIsAllowed {
			return errors.BadRequest("mapping (or transformation) is required")
		}
		return errors.BadRequest("mapping is required")
	}
	if !transformationIsAllowed && action.Transformation != nil {
		return errors.BadRequest("transformation is not allowed")
	}

	return nil
}

// allowsActionTarget reports whether a connection with the given connector type
// and role supports an action with the given target.
//
// Refer to the specifications in the file "connector/Actions support.md" for
// more details.
func allowsActionTarget(typ state.ConnectorType, role _connector.Role, target state.ActionTarget) bool {
	isSource := role == _connector.SourceRole
	usersOrGroups := target == state.UsersTarget || target == state.GroupsTarget
	switch typ {
	case state.AppType:
		return !isSource || (isSource && usersOrGroups)
	case state.DatabaseType:
		return usersOrGroups
	case state.FileType:
		return usersOrGroups
	case
		state.MobileType,
		state.ServerType,
		state.StreamType,
		state.WebsiteType:
		return isSource
	default:
		return false
	}
}

const noQueryLimit = -1

// compileActionQuery compiles the given query and returns it. If limit is
// noQueryLimit removes the $limit variable (along with '{{' and '}}');
// otherwise, replaces the variable with limit.
func compileActionQuery(query string, limit int) (string, error) {
	p := strings.Index(query, "$limit")
	if p == -1 {
		return "", errors.BadRequest("query does not contain the $limit variable")
	}
	s1 := strings.Index(query[:p], "{{")
	if s1 == -1 {
		return "", errors.BadRequest("query does not contain '{{'")
	}
	n := len("$limit")
	s2 := strings.Index(query[p+n:], "}}")
	if s2 == -1 {
		return "", errors.BadRequest("query does not contain '}}'")
	}
	s2 += p + n + 2
	if limit == noQueryLimit {
		return query[:s1] + query[s2:], nil
	}
	return query[:s1] + strings.ReplaceAll(query[s1+2:s2-2], "$limit", strconv.Itoa(limit)) + query[s2:], nil
}

// marshalSchema marshals the given schema.
// If schema is invalid, returns []byte("null") and no errors.
func marshalSchema(schema types.Type) ([]byte, error) {
	rawSchema, err := schema.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if utf8.RuneCount(rawSchema) > rawSchemaMaxSize {
		return nil, errors.New("data is too large")
	}
	return rawSchema, nil
}

func normalize(values map[string]any, schema types.Type) (map[string]any, error) {
	out := make(map[string]any, len(values))
	for name, value := range values {
		prop, ok := schema.Property(name)
		if !ok {
			return nil, fmt.Errorf("property %q not found", name)
		}
		// TODO(Gianluca): call the proper normalization function.
		v, err := normalization.NormalizeAppProperty(name, prop.Type, value, prop.Nullable)
		if err != nil {
			return nil, err
		}
		out[name] = v
	}
	return out, nil
}

// setSettings sets the given settings of the given connection.
// It is a copy of the apis.setSettings function, so keep in sync.
func setSettings(ctx context.Context, db *postgres.DB, connection int, settings []byte) error {
	if !utf8.Valid(settings) {
		return errors.New("settings is not valid UTF-8")
	}
	if len(settings) > maxSettingsLen && utf8.RuneCount(settings) > maxSettingsLen {
		return fmt.Errorf("settings is longer than %d runes", maxSettingsLen)
	}
	n := state.SetConnectionSettings{
		Connection: connection,
		Settings:   settings,
	}
	err := db.Transaction(ctx, func(tx *postgres.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE connections SET settings = $1 WHERE id = $2", n.Settings, n.Connection)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// statsTimeSlot returns the stats time slot for the time t.
// t must be a UTC time.
func statsTimeSlot(t time.Time) int {
	epoc := int(t.Unix())
	return epoc / (60 * 60)
}

// unmappedProperties returns the names of the unmapped properties in schema, if
// there is at least one, otherwise returns nil.
// schema must be valid.
func unmappedProperties(schema types.Type, mapped []types.Path) []string {
	schemaProps := schema.PropertiesNames()
	notMapped := make(map[string]struct{}, len(schemaProps))
	for _, p := range schemaProps {
		notMapped[p] = struct{}{}
	}
	for _, path := range mapped {
		delete(notMapped, path[0])
	}
	if len(notMapped) == 0 {
		return nil
	}
	props := maps.Keys(notMapped)
	slices.Sort(props)
	return props
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

// deserializeCursor deserializes a cursor passed to the API.
func deserializeCursor(cursor string) (_connector.Cursor, error) {
	data, err := hex.DecodeString(cursor)
	if err != nil {
		return _connector.Cursor{}, err
	}
	var c _connector.Cursor
	err = json.Unmarshal(data, &c)
	if err != nil {
		return _connector.Cursor{}, err
	}
	// TODO(marco): validate the cursor's fields.
	return c, nil
}

// serializeCursor serializes a cursor to be returned by the API.
func serializeCursor(cursor _connector.Cursor) (string, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	err := enc.Encode(cursor)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b.Bytes()), nil
}

// temporaryTransformer is a transformers.Transformer that creates a function
// at each call and deletes it after the call returns. Any call to a method that
// is not CallFunction panics.
type temporaryTransformer struct {
	name        string                   // function name.
	source      string                   // source code.
	transformer transformers.Transformer // underlying transformer.
}

func newTemporaryTransformer(name, source string, transformer transformers.Transformer) *temporaryTransformer {
	return &temporaryTransformer{name, source, transformer}
}

func (tp *temporaryTransformer) CallFunction(ctx context.Context, _, _ string, inSchema, outSchema types.Type, values []map[string]any) ([]transformers.Result, error) {
	version, err := tp.transformer.CreateFunction(ctx, tp.name, tp.source)
	if err != nil {
		return nil, nil
	}
	defer func() {
		go func() {
			err := tp.transformer.DeleteFunction(context.Background(), tp.name)
			if err != nil {
				slog.Warn("cannot delete transformation function", "name", tp.name, "err", err)
			}
		}()
	}()
	return tp.transformer.CallFunction(ctx, tp.name, version, inSchema, outSchema, values)
}

func (tp *temporaryTransformer) Close(_ context.Context) error { panic("not supported") }
func (tp *temporaryTransformer) CreateFunction(_ context.Context, _, _ string) (string, error) {
	panic("not supported")
}
func (tp *temporaryTransformer) DeleteFunction(_ context.Context, _ string) error {
	panic("not supported")
}
func (tp *temporaryTransformer) SupportLanguage(_ state.Language) bool { panic("not supported") }
func (tp *temporaryTransformer) UpdateFunction(_ context.Context, _, _ string) (string, error) {
	panic("not supported")
}
