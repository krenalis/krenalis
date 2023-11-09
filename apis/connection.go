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
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/connectors"
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
	InvalidPlaceholder   errors.Code = "InvalidPlaceholder"
	InvalidTable         errors.Code = "InvalidTable"
	KeyNotExist          errors.Code = "KeyNotExist"
	LanguageNotSupported errors.Code = "LanguageNotSupported"
	NoColumns            errors.Code = "NoColumns"
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
	Role         Role
	Enabled      bool
	Connector    int
	Storage      int // zero if the connection is not a file or does not have a storage.
	Compression  Compression
	WebsiteHost  string
	HasSettings  bool
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
//
// It returns an errors.UnprocessableError error with code
//   - EventTypeNotExist, if the event type does not exist for the connection.
//   - FetchSchemaFailed, if an error occurred fetching the schema.
//   - NoGroupsSchema, if the data warehouse does not have groups schema.
//   - NoUsersSchema, if the data warehouse does not have users schema.
func (this *Connection) ActionSchemas(ctx context.Context, target Target, eventType string) (*ActionSchemas, error) {

	this.apis.mustBeOpen()

	ctx, span := telemetry.TraceSpan(ctx, "Connection.ActionSchemas", "target", target, "eventType", eventType)
	defer span.End()

	// Validate the target and the event type.
	eventTypeSchema, err := this.validateTargetAndEventType(ctx, target, eventType)
	if err != nil {
		return nil, err
	}

	c := this.connection

	switch connector := c.Connector(); connector.Type {

	case state.AppType:
		switch target {
		case Users:
			var err error
			schema, err := this.app().Schema(ctx, state.Users, "")
			if err != nil {
				return nil, errors.Unprocessable(FetchSchemaFailed, "an error occurred fetching the schema: %w", err)
			}
			var ok bool
			usersIdentities, ok := c.Workspace().Schemas["users_identities"]
			if !ok {
				return nil, errors.Unprocessable(NoUsersSchema, "users_identities schema not loaded from data warehouse")
			}
			if c.Role == state.Source {
				return &ActionSchemas{In: schema, Out: *usersIdentities}, nil
			} else {
				return &ActionSchemas{In: usersIdentities.Unflatten(), Out: schema}, nil
			}
		case Groups:
			var err error
			schema, err := this.app().Schema(ctx, state.Groups, "")
			if err != nil {
				return nil, errors.Unprocessable(FetchSchemaFailed, "an error occurred fetching the schema: %w", err)
			}
			var ok bool
			grSchema, ok := c.Workspace().Schemas["groups"]
			if !ok {
				return nil, errors.Unprocessable(NoGroupsSchema, "groups schema not loaded from data warehouse")
			}
			if c.Role == state.Source {
				return &ActionSchemas{In: schema, Out: grSchema.Unflatten()}, nil
			} else {
				return &ActionSchemas{In: grSchema.Unflatten(), Out: schema}, nil
			}
		case Events:
			return &ActionSchemas{In: events.Schema.Unflatten(), Out: eventTypeSchema}, nil
		}

	case state.DatabaseType:
		switch target {
		case Users:
			if c.Role == state.Source {
				usersIdentities, ok := c.Workspace().Schemas["users_identities"]
				if !ok {
					return nil, errors.Unprocessable(NoUsersSchema, "users_identities schema not loaded from data warehouse")
				}
				out := usersIdentities.Unflatten()
				return &ActionSchemas{Out: out}, nil
			} else {
				users, ok := c.Workspace().Schemas["users"]
				if !ok {
					return nil, errors.Unprocessable(NoUsersSchema, "users schema not loaded from data warehouse")
				}
				in := users.Unflatten()
				return &ActionSchemas{In: in}, nil
			}
		case Groups:
			if c.Role == state.Source {
				groupsIdentities, ok := c.Workspace().Schemas["groups_identities"]
				if !ok {
					return nil, errors.Unprocessable(NoGroupsSchema, "groups_identities schema not loaded from data warehouse")
				}
				out := groupsIdentities.Unflatten()
				return &ActionSchemas{Out: out}, nil
			} else {
				groups, ok := c.Workspace().Schemas["groups"]
				if !ok {
					return nil, errors.Unprocessable(NoGroupsSchema, "groups schema not loaded from data warehouse")
				}
				in := groups.Unflatten()
				return &ActionSchemas{In: in}, nil
			}
		}

	case state.FileType:
		switch target {
		case Users:
			if c.Role == state.Source {
				usersIdentities, ok := c.Workspace().Schemas["users_identities"]
				if !ok {
					return nil, errors.Unprocessable(NoUsersSchema, "users_identities schema not loaded from data warehouse")
				}
				out := usersIdentities.Unflatten()
				return &ActionSchemas{Out: out}, nil
			} else {
				users, ok := c.Workspace().Schemas["users"]
				if !ok {
					return nil, errors.Unprocessable(NoUsersSchema, "users schema not loaded from data warehouse")
				}
				in := users.Unflatten()
				return &ActionSchemas{In: in}, nil
			}
		case Groups:
			if c.Role == state.Source {
				groupsIdentities, ok := c.Workspace().Schemas["groups_identities"]
				if !ok {
					return nil, errors.Unprocessable(NoGroupsSchema, "groups_identities schema not loaded from data warehouse")
				}
				out := groupsIdentities.Unflatten()
				return &ActionSchemas{Out: out}, nil
			} else {
				groups, ok := c.Workspace().Schemas["groups"]
				if !ok {
					return nil, errors.Unprocessable(NoGroupsSchema, "groups schema not loaded from data warehouse")
				}
				in := groups.Unflatten()
				return &ActionSchemas{In: in}, nil
			}
		}

	case state.MobileType, state.ServerType, state.StreamType, state.WebsiteType:
		if eventType != "" {
			return nil, errors.NotFound("event type not expected")
		}
		switch target {
		case Users:
			usersIdentities, ok := c.Workspace().Schemas["users_identities"]
			if !ok {
				return nil, errors.Unprocessable(NoUsersSchema, "users_identities schema not loaded from data warehouse")
			}
			out := usersIdentities.Unflatten()
			return &ActionSchemas{In: events.Schema.Unflatten(), Out: out}, nil
		case Groups:
			groupsIdentities, ok := c.Workspace().Schemas["groups_identities"]
			if !ok {
				return nil, errors.Unprocessable(NoGroupsSchema, "groups_identities schema not loaded from data warehouse")
			}
			out := groupsIdentities.Unflatten()
			return &ActionSchemas{In: events.Schema.Unflatten(), Out: out}, nil
		}
		return &ActionSchemas{}, nil

	}

	panic("unreachable code")
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
//   - EventTypeNotExist, if the event type does not exist for the connection.
//   - LanguageNotSupported, if the transformation language is not supported.
//   - MappingOverAnonymousIdentifier, if the action maps over an anonymous
//     identifier.
//   - TargetAlreadyExist, if an action already exists for a target for the
//     connection.
func (this *Connection) AddAction(ctx context.Context, target Target, eventType string, action ActionToSet) (int, error) {

	this.apis.mustBeOpen()

	ctx, span := telemetry.TraceSpan(ctx, "Connection.AddAction", "target", target, "eventType", eventType)
	defer span.End()

	// Validate the target and the event type.
	eventTypeSchema, err := this.validateTargetAndEventType(ctx, target, eventType)
	if err != nil {
		return 0, err
	}

	// Validate the action.
	if err := this.validateActionToSet(action, state.Target(target), eventTypeSchema); err != nil {
		return 0, err
	}

	span.Log("action validated successfully")

	n := state.AddAction{
		Connection:        this.connection.ID,
		Target:            state.Target(target),
		Name:              action.Name,
		Enabled:           action.Enabled,
		EventType:         eventType,
		ScheduleStart:     int16(mathrand.Intn(24 * 60)),
		SchedulePeriod:    60,
		InSchema:          action.InSchema,
		OutSchema:         action.OutSchema,
		Mapping:           action.Mapping,
		Query:             action.Query,
		Path:              action.Path,
		TableName:         action.TableName,
		Sheet:             action.Sheet,
		IdentityProperty:  action.IdentityProperty,
		TimestampProperty: action.TimestampProperty,
		TimestampFormat:   action.TimestampFormat,
		ExportMode:        (*state.ExportMode)(action.ExportMode),
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
		case state.Events:
			switch typ := this.connection.Connector().Type; typ {
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
		case state.Users, state.Groups:
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
			"identity_property, timestamp_property, timestamp_format, " +
			"export_mode, matching_properties_internal, matching_properties_external)\n" +
			" VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)"
		_, err := tx.Exec(ctx, query, n.ID, n.Connection, n.Target, n.EventType,
			n.Name, n.Enabled, n.ScheduleStart, n.SchedulePeriod, rawInSchema, rawOutSchema, string(filter), mapping,
			transformation.Source, transformation.Language, transformation.Version, n.Query, n.Path, n.TableName,
			n.Sheet, n.IdentityProperty, n.TimestampProperty, n.TimestampFormat, n.ExportMode, matchPropInternal, matchPropExternal)
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

	// Get the users.
	objects, next, err := this.app().Users(ctx, schema, cur)
	eof := err == io.EOF
	if err != nil && !eof {
		return nil, "", err
	}
	users := make([]map[string]any, len(objects))
	for i, object := range objects {
		if object.Err != nil {
			return nil, "", err
		}
		users[i] = object.Properties
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
// on the connector that must be a file with a storage. path cannot be empty,
// cannot be longer than 1024 runes, and must be UTF-8 encoded.
//
// It returns an errors.UnprocessableError error with code:
//   - InvalidPath, if path is not valid for the storage connector.
//   - InvalidPlaceholder, if path for destination connections contains an
//     invalid placeholder.
//   - NoStorage, if the connection does not have a storage.
func (this *Connection) CompletePath(ctx context.Context, path string) (string, error) {
	this.apis.mustBeOpen()
	c := this.connection
	if c.Connector().Type != state.FileType {
		return "", errors.BadRequest("connection %d is not a file connection", c.ID)
	}
	if path == "" {
		return "", errors.BadRequest("path is empty")
	}
	if !utf8.ValidString(path) {
		return "", errors.BadRequest("path is not UTF-8 encoded")
	}
	if n := utf8.RuneCountInString(path); n > 1024 {
		return "", errors.BadRequest("path is longer than 1024 runes")
	}
	if c.Role == state.Destination {
		var err error
		path, err = replacePathPlaceholders(path, time.Now().UTC())
		if err != nil {
			return "", errors.Unprocessable(InvalidPlaceholder, "%s", err)
		}
	}
	path, err := this.file().CompletePath(ctx, path)
	if err != nil {
		if err == connectors.ErrNoStorage {
			return "", errors.Unprocessable(NoStorage, "connection %d does not have a storage", c.ID)
		}
		if err, ok := err.(*connectors.InvalidPathError); ok {
			return "", errors.Unprocessable(InvalidPath, "%w", err)
		}
		return "", err
	}
	return path, nil
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
// If a database error occurred, it returns an errors.UnprocessableError with
// code DatabaseFailed.
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
	if c.Role != state.Source {
		return nil, types.Type{}, errors.BadRequest("database %d is not a source", c.ID)
	}

	// Execute the query.
	query, err := compileActionQuery(query, limit)
	if err != nil {
		return nil, types.Type{}, errors.Unprocessable(DatabaseFailed, "a database error occurred: %w", err)
	}
	database := this.database()
	defer database.Close()
	rows, err := database.Query(ctx, query)
	if err != nil {
		return nil, types.Type{}, errors.Unprocessable(DatabaseFailed, "a database error occurred: %w", err)
	}
	defer rows.Close()

	// Scan the rows.
	var users []map[string]any
	for rows.Next() {
		row, err := rows.Scan()
		if err != nil {
			return nil, types.Type{}, errors.Unprocessable(DatabaseFailed, "a database error occurred: %w", err)
		}
		users = append(users, row)
	}
	err = rows.Err()
	if err != nil {
		return nil, types.Type{}, errors.Unprocessable(DatabaseFailed, "a database error occurred: %w", err)
	}

	schema := types.Object(rows.Columns())

	return users, schema, nil
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
	if c.Role != state.Source {
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
// encoded with a length in range [1, 1024]. If the connection supports sheets,
// sheet is the sheet name and must be UTF-8 encoded with a length in range
// [1, 100], otherwise must be an empty string. limit limits the number of
// records to return and must be in range [0, 100].
//
// It returns an errors.UnprocessableError error with code
//
//   - NoColumns, if the file has no columns.
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

	columns, records, err := this.file().Read(ctx, path, sheet, limit)
	if err != nil {
		switch err {
		case connectors.ErrNoColumns:
			return nil, types.Type{}, errors.Unprocessable(NoColumns, "file does not have columns")
		case connectors.ErrNoStorage:
			return nil, types.Type{}, errors.Unprocessable(NoStorage, "connection %d does not have a storage", c.ID)
		}
		return nil, types.Type{}, errors.Unprocessable(ReadFileFailed, "an error occurred reading the %s file: %w", connector.Name, err)
	}

	return records, types.Object(columns), nil
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
	if c.Role != state.Source {
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
	if c.Role != state.Destination {
		return nil, errors.BadRequest("connection %d is not a destination", c.ID)
	}
	if !c.Connector().Targets.Contains(state.Events) {
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

	// Get the event type.
	outSchema, err := this.app().Schema(ctx, state.Events, eventType)
	if err != nil {
		if err == connectors.ErrEventTypeNotExist {
			err = errors.Unprocessable(EventTypeNotExist, "connection %d does not have event type %q", c.ID, eventType)
		}
		return nil, err
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

	preview, err := this.app().PreviewSendEvent(ctx, eventType, ev.ConnectorEvent(), data)
	if err != nil {
		if err == _connector.ErrEventTypeNotExist {
			err = errors.Unprocessable(EventTypeNotExist, "connection %d does not have event type %q", c.ID, eventType)
		}
		return nil, err
	}

	return preview, nil
}

// Set sets the connection.
//
// It returns an errors.UnprocessableError error with code StorageNotExist, if
// the storage does not exist.
func (this *Connection) Set(ctx context.Context, connection ConnectionToSet) error {

	this.apis.mustBeOpen()

	ctx, span := telemetry.TraceSpan(ctx, "Connection.Set", "connection", this.connection.ID)
	defer span.End()

	if connection.Name == "" || utf8.RuneCountInString(connection.Name) > 100 {
		return errors.BadRequest("name %q is not valid", connection.Name)
	}
	if connection.Storage < 0 || connection.Storage > maxInt32 {
		return errors.BadRequest("storage identifier %d is not valid", connection.Storage)
	}
	switch connection.Compression {
	case NoCompression, ZipCompression, GzipCompression, SnappyCompression:
	default:
		return errors.BadRequest("compression %q is not valid", connection.Compression)
	}
	if connection.Storage == 0 && connection.Compression != NoCompression {
		return errors.BadRequest("compression requires a storage")
	}

	n := state.SetConnection{
		Connection:  this.connection.ID,
		Name:        connection.Name,
		Enabled:     connection.Enabled,
		Storage:     connection.Storage,
		Compression: state.Compression(connection.Compression),
		WebsiteHost: connection.WebsiteHost,
	}

	c := this.connection.Connector()

	// Validate the storage.
	if n.Storage > 0 {
		if c.Type != state.FileType {
			return errors.BadRequest("connector %d cannot have a storage, it's a %s",
				c.ID, strings.ToLower(c.Type.String()))
		}
		s, ok := this.connection.Workspace().Connection(n.Storage)
		if !ok {
			return errors.Unprocessable(StorageNotExist, "storage %d does not exist", n.Storage)
		}
		if s.Connector().Type != state.StorageType {
			return errors.BadRequest("connection %d is not a storage", n.Storage)
		}
		if s.Role != this.connection.Role {
			if this.connection.Role == state.Source {
				return errors.BadRequest("storage %d is not a source", n.Storage)
			}
			return errors.BadRequest("storage %d is not a destination", n.Storage)
		}
	}

	// Validate the website host.
	if n.WebsiteHost != "" {
		if c.Type != state.WebsiteType {
			return errors.BadRequest("connector %d cannot have a website host, it's a %s",
				c.ID, strings.ToLower(c.Type.String()))
		}
		if h, p, found := strings.Cut(n.WebsiteHost, ":"); h == "" || len(n.WebsiteHost) > 255 {
			return errors.BadRequest("website host %q is not valid", n.WebsiteHost)
		} else if found {
			if port, _ := strconv.Atoi(p); port < 1 || port > 65535 {
				return errors.BadRequest("website host %q is not valid", n.WebsiteHost)
			}
		}
	}

	err := this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE connections SET name = $1, enabled = $2, storage = NULLIF($3, 0), compression = $4, website_host = $5 WHERE id = $6",
			n.Name, n.Enabled, n.Storage, n.Compression, n.WebsiteHost, n.Connection)
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
	// TODO: check and delete alternative fieldsets keys that have 'null' value
	// before saving to database
	b, err := this.apis.connectors.ServeConnectionUI(ctx, this.connection, event, values)
	if err != nil {
		switch err {
		case connectors.ErrNoUserInterface:
			err = errors.BadRequest("connector %d does not have a UI", this.connection.ID)
		case connectors.ErrEventNotExist:
			err = errors.Unprocessable(EventNotExist, "UI event %q does not exist for %s connector", event, this.connection.Connector().Name)
		}
		return nil, err
	}
	return b, nil
}

// Sheets returns the sheets of the file at the given path. The connection must
// be a file connection with multi sheets support and path must be a not empty
// UTF-8 encoded string.
//
// If the connection does not exist anymore, it returns an errors.NotFoundError
// error.
//
// It returns an errors.UnprocessableError error with code
//   - NoStorage, if the file connection does not have a storage.
//   - ReadFileFailed, if an error occurred reading the file.
func (this *Connection) Sheets(ctx context.Context, path string) ([]string, error) {
	this.apis.mustBeOpen()
	connector := this.connection.Connector()
	if connector.Type != state.FileType {
		return nil, errors.BadRequest("connection %d is not a file", this.connection.ID)
	}
	if !connector.HasSheets {
		return nil, errors.BadRequest("connection %d does not supports sheets", this.connection.ID)
	}
	if path == "" {
		return nil, errors.BadRequest("path is empty")
	}
	if !utf8.ValidString(path) {
		return nil, errors.BadRequest("path is not UTF-8 encoded")
	}
	sheets, err := this.file().Sheets(ctx, path)
	if err != nil {
		if err == connectors.ErrNoStorage {
			return nil, errors.Unprocessable(NoStorage, "file connection %d does not have a storage", this.connection.ID)
		}
		return nil, errors.Unprocessable(ReadFileFailed, "%w", err)
	}
	return sheets, nil
}

// ConnectionsStats represents the statistics on a connection for the last 24
// hours.
type ConnectionsStats struct {
	UserIdentities [24]int // ingested or loaded user identities per hour
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
		UserIdentities: [24]int{},
	}
	query := "SELECT time_slot, user_identities\nFROM connections_stats\nWHERE connection = $1 AND time_slot BETWEEN $2 AND $3"
	err := this.apis.db.QueryScan(ctx, query, this.connection.ID, fromSlot, toSlot, func(rows *postgres.Rows) error {
		var err error
		var slot, userIdentities int
		for rows.Next() {
			if err = rows.Scan(&slot, &userIdentities); err != nil {
				return err
			}
			stats.UserIdentities[slot-fromSlot] = userIdentities
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
// It returns an error.Unprocessable error with code:
//   - DatabaseFailed, if a database error occurred.
//   - InvalidTable, if the table does not contain an unsigned 32-bit column
//     named "id" or if there are no other columns apart from "id".
func (this *Connection) TableSchema(ctx context.Context, table string) (types.Type, error) {
	this.apis.mustBeOpen()
	c := this.connection
	connector := c.Connector()
	if connector.Type != state.DatabaseType {
		return types.Type{}, errors.BadRequest("connection %d is not a database", c.ID)
	}
	if c.Role != state.Destination {
		return types.Type{}, errors.BadRequest("database %d is not a destination", c.ID)
	}
	if table == "" || utf8.RuneCountInString(table) > 1024 {
		return types.Type{}, errors.BadRequest("table name is not valid")
	}
	database := this.database()
	defer database.Close()
	columns, err := database.Columns(ctx, table)
	if err != nil {
		return types.Type{}, errors.Unprocessable(DatabaseFailed, "a database error occurred: %w", err)
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
	Target        Target
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
	if c.Role != state.Source {
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
	if targets.Contains(state.Users) {
		switch typ := c.Connector().Type; typ {
		case
			state.AppType,
			state.DatabaseType,
			state.FileType:
			var name, description string
			var missingSchema bool
			if c.Role == state.Source {
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
				Target:        Users,
				MissingSchema: missingSchema,
			}
			actionTypes = append(actionTypes, at)
		case
			state.MobileType,
			state.ServerType,
			state.WebsiteType:
			if c.Role == state.Source {
				at := ActionType{
					Name:          "Import users",
					Description:   "Import users from the events of the " + connector.Name,
					Target:        Users,
					MissingSchema: wsSchemas["users_identities"] == nil,
				}
				actionTypes = append(actionTypes, at)
			}
		}
	}
	if targets.Contains(state.Groups) {
		switch typ := c.Connector().Type; typ {
		case
			state.AppType,
			state.DatabaseType,
			state.FileType:
			var name, description string
			var missingSchema bool
			if c.Role == state.Source {
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
				Target:        Groups,
				MissingSchema: missingSchema,
			}
			actionTypes = append(actionTypes, at)
		case
			state.MobileType,
			state.ServerType,
			state.WebsiteType:
			if c.Role == state.Source {
				at := ActionType{
					Name:          "Import groups",
					Description:   "Import groups from the events of the " + connector.Name,
					Target:        Groups,
					MissingSchema: wsSchemas["groups"] == nil,
				}
				actionTypes = append(actionTypes, at)
			}
		}
	}
	if targets.Contains(state.Events) {
		switch typ := c.Connector().Type; typ {
		case state.MobileType, state.ServerType, state.WebsiteType:
			if c.Role == state.Source {
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
					Target:      Events,
				}
				actionTypes = append(actionTypes, at)
			}
		case state.AppType:
			eventTypes, err := this.app().EventTypes(ctx)
			if err != nil {
				return nil, errors.Unprocessable(FetchSchemaFailed, "an error occurred fetching the schema: %w", err)
			}
			for _, et := range eventTypes {
				id := et.ID
				actionTypes = append(actionTypes, ActionType{
					Name:        et.Name,
					Description: et.Description,
					Target:      Events,
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

// app returns the app of the connection.
func (this *Connection) app() *connectors.App {
	return this.apis.connectors.App(this.connection)
}

// database returns the database of the connection.
// The caller must call the database's Close method when the database is no
// longer needed.
func (this *Connection) database() *connectors.Database {
	return this.apis.connectors.Database(this.connection)
}

// file returns the file of the connection.
func (this *Connection) file() *connectors.File {
	return this.apis.connectors.File(this.connection)
}

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

// Role represents a role.
type Role int

const (
	Source      Role = iota + 1 // source
	Destination                 // destination
)

// MarshalJSON implements the json.Marshaler interface.
// It panics if role is not a valid Role value.
func (role Role) MarshalJSON() ([]byte, error) {
	return []byte(`"` + role.String() + `"`), nil
}

// String returns the string representation of role.
// It panics if role is not a valid Role value.
func (role Role) String() string {
	switch role {
	case Source:
		return "Source"
	case Destination:
		return "Destination"
	}
	panic("invalid connection role")
}

var null = []byte("null")

// UnmarshalJSON implements the json.Unmarshaler interface.
func (role *Role) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, null) {
		return nil
	}
	var v any
	err := json.Unmarshal(data, &v)
	if err != nil {
		return fmt.Errorf("json: cannot unmarshal into a apis.Role value: %s", err)
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an api.Role value", v)
	}
	var r Role
	switch s {
	case "Source":
		r = Source
	case "Destination":
		r = Destination
	default:
		return fmt.Errorf("invalid apis.Role: %s", s)
	}
	*role = r
	return nil
}

// updateConnectionsStats updates the statistics about the connection.
func (this *Connection) updateConnectionsStats(ctx context.Context) error {
	connection := this.connection.ID
	_, err := this.apis.db.Exec(ctx, "INSERT INTO connections_stats AS cs (connection, time_slot, user_identities)\n"+
		"VALUES ($1, $2, 1)\n"+
		"ON CONFLICT (connection, time_slot) DO UPDATE SET user_identities = cs.user_identities + 1",
		connection, statsTimeSlot(time.Now().UTC()))
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
func (this *Connection) validateActionToSet(action ActionToSet, target state.Target, eventTypeSchema types.Type) error {

	inSchema := action.InSchema
	outSchema := action.OutSchema

	// Check if the input schema must be the schema of the events.
	isInEventsSchema := false
	switch this.connection.Connector().Type {
	case state.AppType:
		isInEventsSchema = target == state.Events
	case state.MobileType, state.ServerType, state.WebsiteType:
		isInEventsSchema = true
	}
	if isInEventsSchema {
		inSchema = events.Schema
	}

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
	if inSchema.Valid() && inSchema.PhysicalType() != types.PtObject {
		return errors.BadRequest("input schema, if provided, must be an object")
	}
	if outSchema.Valid() && outSchema.PhysicalType() != types.PtObject {
		return errors.BadRequest("out schema, if provided, must be an object")
	}
	// Validate the filter.
	var inPaths []types.Path
	if action.Filter != nil {
		if !inSchema.Valid() {
			return errors.BadRequest("input schema is required by the filter")
		}
		properties, err := validateFilter(action.Filter, inSchema)
		if err != nil {
			return errors.BadRequest("filter is not valid: %w", err)
		}
		if !isInEventsSchema {
			inPaths = properties
		}
	}
	// An action cannot have both mappings and transformations.
	if action.Mapping != nil && action.Transformation != nil {
		return errors.BadRequest("action cannot have both mappings and transformation")
	}
	// Validate the mapping.
	var outPaths []types.Path
	if action.Mapping != nil && len(action.Mapping) > 0 {
		if !inSchema.Valid() {
			return errors.BadRequest("input schema is required by the mapping")
		}
		if !outSchema.Valid() {
			return errors.BadRequest("output schema is required by the mapping")
		}
		for path, expr := range action.Mapping {
			outPath, err := types.ParsePropertyPath(path)
			if err != nil {
				return errors.BadRequest("output mapped property %q is not valid", path)
			}
			outPaths = append(outPaths, outPath)
			p, err := outSchema.PropertyByPath(outPath)
			if err != nil {
				err := err.(types.PathNotExistError)
				return errors.BadRequest("output mapped property %s not found in output schema", err.Path)
			}
			expr, err := mapexp.Compile(expr, inSchema, p.Type, p.Nullable)
			if err != nil {
				return errors.BadRequest("invalid expression mapped to %s: %s", path, err)
			}
			if !isInEventsSchema {
				inPaths = append(inPaths, expr.Properties()...)
			}
		}
	}
	// Validate the transformation.
	if action.Transformation != nil {
		if !inSchema.Valid() {
			return errors.BadRequest("input schema is required by the transformation")
		}
		if !outSchema.Valid() {
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
	if action.Transformation == nil {
		if inPaths != nil {
			if props := unmappedProperties(inSchema, inPaths); props != nil {
				return errors.BadRequest("input schema contains unmapped properties: %s", strings.Join(props, ", "))
			}
		}
		if outPaths != nil {
			if props := unmappedProperties(outSchema, outPaths); props != nil {
				return errors.BadRequest("output schema contains unmapped properties: %s", strings.Join(props, ", "))
			}
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
		if this.connection.Role == state.Destination {
			_, err := replacePathPlaceholders(action.Path, time.Now().UTC())
			if err != nil {
				return errors.BadRequest("path is not valid: %s", err)
			}
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
	if importingUsers := c.Role == state.Source && target == state.Users; importingUsers {
		var tOutProps []string
		if action.Transformation != nil {
			tOutProps = outSchema.PropertiesNames()
		}
		for _, p := range ws.AnonymousIdentifiers.Priority {
			_, ok := action.Mapping[p]
			if ok || slices.Contains(tOutProps, p) {
				return errors.Unprocessable(MappingOverAnonymousIdentifier, "cannot map over the property %s because it is an anonymous identifier", p)
			}
		}
	}

	// Check if the query is allowed.
	if needsQuery := connector.Type == state.DatabaseType && c.Role == state.Source; needsQuery {
		if action.Query == "" {
			return errors.BadRequest("query cannot be empty for database actions")
		}
	} else {
		if action.Query != "" {
			return errors.BadRequest("%s actions cannot have a query", connector.Type)
		}
	}

	// Check if the filters are allowed.
	targetUsersOrGroups := target == state.Users || target == state.Groups
	var filtersAllowed bool
	switch connector.Type {
	case state.AppType:
		filtersAllowed = c.Role == state.Destination
	case state.DatabaseType:
		filtersAllowed = c.Role == state.Destination
	case state.FileType:
		filtersAllowed = targetUsersOrGroups && c.Role == state.Destination
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

	// Check the property for the identity and for the timestamp.
	if connector.Type == state.FileType && c.Role == state.Source {
		if action.IdentityProperty == "" {
			return errors.BadRequest("property for the identity is mandatory")
		}
		if !types.IsValidPropertyName(action.IdentityProperty) {
			return errors.BadRequest("property for the identity has not a valid property name")
		}
		if utf8.RuneCountInString(action.IdentityProperty) > 1024 {
			return errors.BadRequest("property for the identity is longer than 1024 runes")
		}
		if action.TimestampProperty != "" {
			if !types.IsValidPropertyName(action.TimestampProperty) {
				return errors.BadRequest("property for the timestamp has a not valid property name")
			}
			if utf8.RuneCountInString(action.TimestampProperty) > 1024 {
				return errors.BadRequest("property for the timestamp is longer than 1024 runes")
			}
			if action.TimestampFormat == "" {
				return errors.BadRequest("timestamp format is mandatory when a timestamp property is provided")
			}
			if !utf8.ValidString(action.TimestampFormat) {
				return errors.BadRequest("timestamp format must be UTF-8 valid")
			}
			if utf8.RuneCountInString(action.TimestampFormat) > 64 {
				return errors.BadRequest("timestamp format is longer than 64 runes")
			}
		} else {
			if action.TimestampFormat != "" {
				return errors.BadRequest("action cannot specify a timestamp format")
			}
		}
	} else {
		if action.IdentityProperty != "" {
			return errors.BadRequest("action cannot specify a property for the identity")
		}
		if action.TimestampProperty != "" {
			return errors.BadRequest("action cannot specify a property for the timestamp")
		}
		if action.TimestampFormat != "" {
			return errors.BadRequest("action cannot specify a timestamp format")
		}
	}

	// Check if the table name is allowed.
	needsTableName := connector.Type == state.DatabaseType && c.Role == state.Destination
	if needsTableName && action.TableName == "" {
		return errors.BadRequest("table name cannot be empty for destination database actions")
	} else if !needsTableName && action.TableName != "" {
		return errors.BadRequest("table name is not allowed")
	}

	// Check if the export options are needed.
	needsExportOptions := connector.Type == state.AppType &&
		c.Role == state.Destination && target == state.Users
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

	// Check if the mapping (or the transformation) is mandatory, and if the
	// transformation is allowed.
	var mappingIsMandatory bool
	var transformationIsAllowed bool
	switch connector.Type {
	case state.AppType:
		if c.Role == state.Destination && target == state.Events {
			mappingIsMandatory = eventTypeSchema.Valid()
			transformationIsAllowed = true
		} else {
			mappingIsMandatory = targetUsersOrGroups
			transformationIsAllowed = true
		}
	case state.MobileType, state.ServerType, state.WebsiteType:
		mappingIsMandatory = targetUsersOrGroups
		transformationIsAllowed = false
	case state.DatabaseType:
		mappingIsMandatory = targetUsersOrGroups
		transformationIsAllowed = mappingIsMandatory
	case state.FileType:
		mappingIsMandatory = c.Role == state.Source && targetUsersOrGroups
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

// ConnectionToSet represents a connection to set in a workspace, by adding a
// new connection (using the method Workspace.AddConnection) or updating an
// existing one (using the method Connection.Set).
type ConnectionToSet struct {

	// Name is the name of the connection. It cannot be longer than 100 runes.
	// If empty, the connection name will be the name of its connector.
	Name string

	// Enable reports whether the connection is enabled or disabled when added.
	Enabled bool

	// Storage is the storage of a file connection. It must be 0 if the
	// connection is not a file or has no storage.
	Storage int

	// Compression is the compression for file connections. It must be
	// NoCompression if there is no storage.
	Compression Compression

	// WebsiteHost is the host, in the form "host:port", of a website
	// connection. It must be empty if the connection is not a website. It
	// cannot be longer than 261 runes.
	WebsiteHost string
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

// validateTargetAndEventType validates a target and an event type and, if the
// event type is not empty, it returns its schema.
//
// It returns an errors.BadRequestError error if target or eventType is not
// valid, or the connection does not support them, and returns an
// errors.UnprocessableError error with code:
//   - EventTypeNotExist, if the connection does not have the event type.
//   - FetchSchemaFailed, if an error occurred fetching the event type schema.
func (this *Connection) validateTargetAndEventType(ctx context.Context, target Target, eventType string) (types.Type, error) {
	// Perform a formal validation.
	if target != Users && target != Groups && target != Events {
		return types.Type{}, errors.BadRequest("target %d is not valid", int(target))
	}
	if eventType != "" && target != Events {
		return types.Type{}, errors.BadRequest("event type cannot be used with %s target", target)
	}
	// Perform a validation based on the connection's type and role.
	// (Refer to the specifications in the file "connector/Actions support.md" for more details)
	c := this.connection
	connector := c.Connector()
	var supported bool
	switch connector.Type {
	case state.AppType:
		supported = c.Role == state.Destination || target != Events
	case state.DatabaseType, state.FileType:
		supported = target != Events
	case state.MobileType, state.ServerType, state.WebsiteType:
		supported = c.Role == state.Source
	case state.StreamType:
		supported = false
	}
	if !supported {
		return types.Type{}, errors.BadRequest("%s are not supported by %s connections", strings.ToLower(target.String()), connector.Type)
	}
	if target == Events {
		if c.Role == state.Source && eventType != "" {
			return types.Type{}, errors.BadRequest("source connections do not have an event type")
		}
		if c.Role == state.Destination && eventType == "" {
			return types.Type{}, errors.BadRequest("destination connections want an event type")
		}
	}
	// Check if the target is supported by the connection.
	if !connector.Targets.Contains(state.Target(target)) {
		return types.Type{}, errors.BadRequest("connection %d does not support %s target", c.ID, target)
	}
	// Check if the event type is supported by the connection.
	if eventType != "" {
		schema, err := this.app().Schema(ctx, state.Target(target), eventType)
		if err != nil {
			if err == connectors.ErrEventTypeNotExist {
				return types.Type{}, errors.Unprocessable(EventTypeNotExist, "connection %d does not have event type %q", c.ID, eventType)
			}
			return types.Type{}, errors.Unprocessable(FetchSchemaFailed, "an error occurred fetching the schema: %w", err)
		}
		return schema, nil
	}
	return types.Type{}, nil
}

// deserializeCursor deserializes a cursor passed to the API.
func deserializeCursor(cursor string) (connectors.Cursor, error) {
	data, err := hex.DecodeString(cursor)
	if err != nil {
		return connectors.Cursor{}, err
	}
	var c _connector.Cursor
	err = json.Unmarshal(data, &c)
	if err != nil {
		return connectors.Cursor{}, err
	}
	// TODO(marco): validate the cursor's fields.
	return c, nil
}

// serializeCursor serializes a cursor to be returned by the API.
func serializeCursor(cursor connectors.Cursor) (string, error) {
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
