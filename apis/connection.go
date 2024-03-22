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
	"log/slog"
	"math"
	mathrand "math/rand"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"chichi/apis/connectors"
	"chichi/apis/datastore"
	"chichi/apis/datastore/expr"
	"chichi/apis/encoding"
	"chichi/apis/errors"
	"chichi/apis/events"
	"chichi/apis/events/eventschema"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/apis/transformers"
	"chichi/apis/transformers/mappings"
	"chichi/telemetry"
	"chichi/types"

	"github.com/google/uuid"
	"github.com/jxskiss/base62"
	"golang.org/x/exp/maps"
)

const (
	maxKeysPerConnection = 20 // maximum number of keys per connection.
	maxInt32             = math.MaxInt32
	rawSchemaMaxSize     = 16_777_215 // maximum size in runes for schemas stored in PostgreSQL.
	queryMaxSize         = 16_777_215 // maximum size in runes of a connection query.
)

const (
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
	NotCompatibleSchema  errors.Code = "NotCompatibleSchema"
	ReadFileFailed       errors.Code = "ReadFileFailed"
	SheetNotExist        errors.Code = "SheetNotExist"
	TargetAlreadyExist   errors.Code = "TargetAlreadyExist"
	TooManyKeys          errors.Code = "TooManyKeys"
	UniqueKey            errors.Code = "UniqueKey"
	WorkspaceNotExist    errors.Code = "WorkspaceNotExist"
)

// Strategy represents a strategy. Can be "AB-C", "ABC", "A-B-C", and "AC-B".
type Strategy string

// isValidStrategy reports whether s is a valid strategy.
func isValidStrategy(s Strategy) bool {
	switch s {
	case "AB-C", "ABC", "A-B-C", "AC-B":
		return true
	}
	return false
}

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
	Strategy     *Strategy
	WebsiteHost  string
	BusinessID   string
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
	_, span := telemetry.TraceSpan(ctx, "Connection.Action", "id", id)
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
	In, Out   types.Type
	Matchings *ActionSchemasMatchings `json:",omitempty"` // only for destination apps on users.
}

type ActionSchemasMatchings struct {
	Internal, External types.Type
}

// dummyGroupsSchema is a dummy "groups" schema, that is used until the groups
// management is properly implemented in Chichi. For now, it serves only as a
// placeholder.
var dummyGroupsSchema = types.Object([]types.Property{
	{Name: "id", Type: types.Int(32)},
})

// ActionSchemas returns the input and the output schemas of an action with the
// given target and event type.
//
// It returns an errors.UnprocessableError error with code
//   - EventTypeNotExist, if the event type does not exist for the connection.
//   - FetchSchemaFailed, if an error occurred fetching the schema.
//   - NoWarehouse, if the workspace does not have a data warehouse.
func (this *Connection) ActionSchemas(ctx context.Context, target Target, eventType string) (*ActionSchemas, error) {

	this.apis.mustBeOpen()

	ctx, span := telemetry.TraceSpan(ctx, "Connection.ActionSchemas", "target", target, "eventType", eventType)
	defer span.End()

	// Verify that the workspace has a data warehouse.
	if this.store == nil {
		ws := this.connection.Workspace()
		return nil, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", ws.ID)
	}

	// Validate the target and the event type.
	eventTypeSchema, err := this.validateTargetAndEventType(ctx, target, eventType)
	if err != nil {
		return nil, err
	}

	users := this.connection.Workspace().UsersSchema
	groups := dummyGroupsSchema

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
			if c.Role == state.Source {
				return &ActionSchemas{
					In:  schema,
					Out: removeMetaProperties(users),
				}, nil
			} else {
				sourceSchema, err := this.app().SchemaAsRole(ctx, state.Source, state.Users, "")
				if err != nil {
					return nil, err
				}
				actionSchemas := &ActionSchemas{
					In:  users, // don't remove meta properties here, they may be useful in transformations.
					Out: schema,
				}
				actionSchemas.Matchings = &ActionSchemasMatchings{
					Internal: onlyForMatching(users),
					External: onlyForMatching(sourceSchema),
				}
				return actionSchemas, nil
			}
		case Groups:
			var err error
			schema, err := this.app().Schema(ctx, state.Groups, "")
			if err != nil {
				return nil, errors.Unprocessable(FetchSchemaFailed, "an error occurred fetching the schema: %w", err)
			}
			if c.Role == state.Source {
				return &ActionSchemas{
					In:  schema,
					Out: removeMetaProperties(groups),
				}, nil
			} else {
				sourceSchema, err := this.app().SchemaAsRole(ctx, state.Source, state.Groups, "")
				if err != nil {
					return nil, err
				}
				actionSchemas := &ActionSchemas{
					In:  groups, // don't remove meta properties here, they may be useful in transformations.
					Out: schema,
				}
				actionSchemas.Matchings = &ActionSchemasMatchings{
					Internal: onlyForMatching(groups),
					External: onlyForMatching(sourceSchema),
				}
				return actionSchemas, nil
			}
		case Events:
			// Use the schema without GID since events sent to the apps do not
			// have the GID, as they are sent as they are after enrichment; the
			// GID is added by the identity resolution directly to the data
			// warehouse, without affecting the events sent to the apps.
			return &ActionSchemas{
				In:  eventschema.SchemaWithoutGID,
				Out: eventTypeSchema,
			}, nil
		}

	case state.DatabaseType:
		switch target {
		case Users:
			if c.Role == state.Source {
				return &ActionSchemas{
					Out: removeMetaProperties(users),
				}, nil
			} else {
				return &ActionSchemas{
					In: users, // don't remove meta properties here, they may be useful in transformations.
				}, nil
			}
		case Groups:
			if c.Role == state.Source {
				return &ActionSchemas{
					Out: removeMetaProperties(groups),
				}, nil
			} else {
				return &ActionSchemas{
					In: groups, // don't remove meta properties here, they may be useful in transformations.
				}, nil
			}
		}

	case state.StorageType:
		switch target {
		case Users:
			if c.Role == state.Source {
				return &ActionSchemas{
					Out: removeMetaProperties(users),
				}, nil
			} else {
				return &ActionSchemas{
					In: users, // don't remove meta properties here, they may be useful in transformations.
				}, nil
			}
		case Groups:
			if c.Role == state.Source {
				return &ActionSchemas{
					Out: removeMetaProperties(groups),
				}, nil
			} else {
				return &ActionSchemas{
					In: groups, // don't remove meta properties here, they may be useful in transformations.
				}, nil
			}
		}

	case state.MobileType, state.ServerType, state.StreamType, state.WebsiteType:
		if eventType != "" {
			return nil, errors.NotFound("event type not expected")
		}
		// The input schema is the events schema without GID because these
		// actions import users identities from incoming events, which do not
		// have any user associated.
		switch target {
		case Users:
			return &ActionSchemas{
				In:  eventschema.SchemaWithoutGID,
				Out: removeMetaProperties(users),
			}, nil
		case Groups:
			return &ActionSchemas{
				In:  eventschema.SchemaWithoutGID,
				Out: removeMetaProperties(groups),
			}, nil
		}
		return &ActionSchemas{}, nil

	}

	panic("unreachable code")
}

// AddAction adds action to the connection returning the identifier of the
// added action. target is the target of the action and must be supported by the
// connector of the connection.
//
// Refer to the specifications in the file "apis/Actions.md" for more details.
//
// It returns an errors.NotFoundError error if the connection does not exist
// anymore, and returns an errors.UnprocessableError error with code
//   - ConnectionNotExist, if the connection does not exist.
//   - ConnectorNotExist, if the file connector of the action does not exist.
//   - InvalidSettings, if the settings are not valid.
//   - LanguageNotSupported, if the transformation language is not supported.
//   - TargetAlreadyExist, if an action already exists for a target for the
//     connection.
func (this *Connection) AddAction(ctx context.Context, target Target, eventType string, action ActionToSet) (int, error) {

	this.apis.mustBeOpen()

	ctx, span := telemetry.TraceSpan(ctx, "Connection.AddAction", "target", target, "eventType", eventType)
	defer span.End()

	// Validate the Connector.
	actionOnFile := this.connection.Connector().Type == state.StorageType
	if actionOnFile && action.Connector == 0 {
		return 0, errors.BadRequest("actions on Storage connections must have a connector")
	}
	if !actionOnFile && action.Connector != 0 {
		return 0, errors.BadRequest("actions on %v connections cannot have a connector", this.connection.Connector().Type)
	}
	var fileConnector *state.Connector
	if action.Connector != 0 {
		if action.Connector < 1 || action.Connector > maxInt32 {
			return 0, errors.BadRequest("connector identifier %d is not valid", action.Connector)
		}
		var ok bool
		fileConnector, ok = this.apis.state.Connector(action.Connector)
		if !ok {
			return 0, errors.Unprocessable(ConnectorNotExist, "connector %d does not exist", action.Connector)
		}
		if fileConnector.Type != state.FileType {
			return 0, errors.BadRequest("type of the action's connector must be File, got %v", fileConnector.Type)
		}
	}

	// Validate the action.
	err := this.validateActionToSet(action, state.Target(target), fileConnector)
	if err != nil {
		return 0, err
	}

	span.Log("action validated successfully")

	connector := this.connection.Connector()
	inSchema := action.InSchema
	if importsUsersIdentitiesFromEvents(connector.Type, this.connection.Role, state.Target(target)) {
		// The input schema is the events schema without GID because this
		// actions imports users identities from incoming events, which,
		// clearly, still do not have any user associated.
		inSchema = eventschema.SchemaWithoutGID
	}

	n := state.AddAction{
		Connection:     this.connection.ID,
		Target:         state.Target(target),
		Name:           action.Name,
		Enabled:        action.Enabled,
		EventType:      eventType,
		ScheduleStart:  int16(mathrand.Intn(24 * 60)),
		SchedulePeriod: 60,
		InSchema:       inSchema,
		OutSchema:      action.OutSchema,
		Transformation: state.Transformation{
			Mapping: action.Transformation.Mapping,
		},
		Query:                   action.Query,
		Connector:               action.Connector,
		Path:                    action.Path,
		Sheet:                   action.Sheet,
		Compression:             state.Compression(action.Compression),
		TableName:               action.TableName,
		IdentityColumn:          action.IdentityColumn,
		TimestampColumn:         action.TimestampColumn,
		TimestampFormat:         action.TimestampFormat,
		ExportMode:              (*state.ExportMode)(action.ExportMode),
		ExportOnDuplicatedUsers: action.ExportOnDuplicatedUsers,
	}
	if function := action.Transformation.Function; function != nil {
		n.Transformation.Function = &state.TransformationFunction{Source: function.Source}
		switch function.Language {
		case "JavaScript":
			n.Transformation.Function.Language = state.JavaScript
		case "Python":
			n.Transformation.Function.Language = state.Python
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

	// Determine the connector ID, for file actions.
	var connectorID *int
	if fileConnector != nil {
		id := fileConnector.ID
		connectorID = &id
	}

	// Generate a random identifier.
	n.ID, err = generateRandomID()
	if err != nil {
		return 0, err
	}

	// Marshal the input and the output schemas.
	rawInSchema, err := marshalSchema(inSchema)
	if err != nil {
		return 0, err
	}
	rawOutSchema, err := marshalSchema(action.OutSchema)
	if err != nil {
		return 0, err
	}

	// Marshal the mapping.
	var mapping []byte
	if action.Transformation.Mapping != nil {
		mapping, err = json.Marshal(action.Transformation.Mapping)
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
	var function state.TransformationFunction
	if n.Transformation.Function != nil {
		name := transformationFunctionName(n.ID, n.Transformation.Function.Language)
		version, err := this.apis.functionTransformer.Create(ctx, name, n.Transformation.Function.Source)
		if err != nil {
			return 0, err
		}
		n.Transformation.Function.Version = version
		function = *n.Transformation.Function
	}

	// Validate the settings.
	if action.Settings != nil {
		if fileConnector == nil {
			return 0, errors.BadRequest("settings cannot be provided because there is no connector")
		}
		if !fileConnector.HasSettings {
			return 0, errors.BadRequest("settings cannot be provided because the File connector has no settings")
		}
	}
	if fileConnector != nil && fileConnector.HasSettings {
		settings := action.Settings
		if settings == nil {
			settings = json.RawMessage("{}")
		}
		conf := &connectors.ConnectorConfig{
			Role:   this.connection.Role,
			Region: state.PrivacyRegion(this.connection.Workspace().PrivacyRegion),
		}
		var err error
		n.Settings, err = this.apis.connectors.ValidateSettings(ctx, fileConnector, conf, settings)
		if err != nil {
			if err != connectors.ErrNoUserInterface {
				return 0, errors.Unprocessable(InvalidSettings, "settings are not valid: %w", err)
			}
			if action.Settings != nil {
				return 0, errors.BadRequest("settings cannot be provided because %s connector %s does not have a UI",
					strings.ToLower(this.connection.Role.String()), fileConnector.Name)
			}
		} else if action.Settings == nil {
			return 0, errors.BadRequest("settings must be provided because %s connector %s has a UI",
				strings.ToLower(this.connection.Role.String()), fileConnector.Name)
		}
	}

	// Add the action.
	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		switch n.Target {
		case state.Events:
			switch connector.Type {
			case state.MobileType, state.ServerType, state.WebsiteType:
				err = tx.QueryVoid(ctx, "SELECT FROM actions WHERE connection = $1 AND target = 'Events'", n.Connection)
				if err != sql.ErrNoRows {
					if err == nil {
						err = errors.Unprocessable(TargetAlreadyExist,
							"action with target %s already exists for %s connection %d", n.Target, connector.Type, n.Connection)
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
		var matchPropInternal, matchPropExternal []byte
		if n.MatchingProperties != nil {
			var err error
			matchPropInternal, err = json.Marshal(n.MatchingProperties.Internal)
			if err != nil {
				return err
			}
			matchPropExternal, err = json.Marshal(n.MatchingProperties.External)
			if err != nil {
				return err
			}
		}
		query := "INSERT INTO actions (id, connection, target, event_type, name, enabled,\n" +
			"schedule_start, schedule_period, in_schema, out_schema, filter, transformation_mapping,\n" +
			"transformation_source, transformation_language, transformation_version, query,\n" +
			"connector, path, sheet, compression, settings, table_name, identity_column,\n" +
			"timestamp_column, timestamp_format, export_mode, matching_properties_internal,\n" +
			"matching_properties_external, export_on_duplicated_users)\n" +
			"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,\n" +
			"$17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29)"
		_, err := tx.Exec(ctx, query, n.ID, n.Connection, n.Target, n.EventType,
			n.Name, n.Enabled, n.ScheduleStart, n.SchedulePeriod, rawInSchema, rawOutSchema,
			string(filter), mapping, function.Source, function.Language, function.Version,
			n.Query, connectorID, n.Path, n.Sheet, n.Compression, string(n.Settings), n.TableName,
			n.IdentityColumn, n.TimestampColumn, n.TimestampFormat, n.ExportMode,
			string(matchPropInternal), string(matchPropExternal), n.ExportOnDuplicatedUsers)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				switch postgres.ErrConstraintName(err) {
				case "actions_connection_fkey":
					err = errors.Unprocessable(ConnectionNotExist, "connection %d does not exist", n.Connection)
				case "actions_connector_fkey":
					err = errors.Unprocessable(ConnectorNotExist, "connector %d does not exist", n.Connector)
				}
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
func (this *Connection) AppUsers(ctx context.Context, schema types.Type, cursor string) ([]byte, string, error) {

	this.apis.mustBeOpen()

	if this.connection.Connector().Type != state.AppType {
		return nil, "", errors.BadRequest("connection %d is not an app connection", this.connection.ID)
	}
	if !schema.Valid() {
		return nil, "", errors.BadRequest("schema is not valid")
	}
	var cur state.Cursor
	if cursor != "" {
		var err error
		cur, err = deserializeCursor(cursor)
		if err != nil {
			return nil, "", errors.BadRequest("cursor is malformed")
		}
	}

	// Get the users.
	records, err := this.app().Users(ctx, schema, cur)
	if err != nil {
		return nil, "", err
	}
	defer records.Close()

	var last connectors.Record
	users := make([]map[string]any, 0, 100)

	errBreak := errors.New("break")
	err = records.For(func(user connectors.Record) error {
		if user.Err != nil {
			return user.Err
		}
		last = user
		users = append(users, user.Properties)
		if len(users) == 100 {
			return errBreak
		}
		return nil
	})
	if err != nil && err != errBreak {
		return nil, "", err
	}
	if err = records.Err(); err != nil {
		return nil, "", err
	}

	// Build the cursor.
	cursor, err = serializeCursor(state.Cursor{
		ID:        last.ID,
		Timestamp: last.Timestamp,
	})
	if err != nil {
		return nil, "", err
	}

	marshaledUsers, err := encoding.MarshalSlice(schema, users)
	if err != nil {
		return nil, "", err
	}

	return marshaledUsers, cursor, nil
}

// CompletePath returns the complete representation of the given path, based
// on the connector that must be a file with a storage. path cannot be empty,
// cannot be longer than 1024 runes, and must be UTF-8 encoded.
//
// It returns an errors.UnprocessableError error with code:
//   - InvalidPath, if path is not valid for the storage connector.
//   - InvalidPlaceholder, if path for source connections contains a placeholder
//     or path for destination connections contains an invalid placeholder.
func (this *Connection) CompletePath(ctx context.Context, path string) (string, error) {
	this.apis.mustBeOpen()
	c := this.connection
	if c.Connector().Type != state.StorageType {
		return "", errors.BadRequest("connection %d is not a storage connection", c.ID)
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
	var replacer connectors.PlaceholderReplacer
	switch c.Role {
	case state.Source:
		_, err := connectors.ReplacePlaceholders(path, func(_ string) (string, bool) {
			return "", false
		})
		if err != nil {
			return "", errors.Unprocessable(InvalidPlaceholder, "the path contains a placeholder syntax, but it cannot be utilized for source actions")
		}
	case state.Destination:
		replacer = newPathPlaceholderReplacer(time.Now().UTC())
	}
	path, err := this.storage().CompletePath(ctx, path, replacer)
	if err != nil {
		switch err := err.(type) {
		case connectors.InvalidPathError:
			return "", errors.Unprocessable(InvalidPath, "%w", err)
		case connectors.PlaceholderError:
			return "", errors.Unprocessable(InvalidPlaceholder, "%w", err)
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
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
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
// must contain the "limit" placeholder. limit must be in range [0, 100].
//
// If the connection does not exist, it returns an errors.NotFoundError error.
// It returns an errors.UnprocessableError error with code:
//
//   - DatabaseFailed, if a database error occurred.
//   - InvalidPlaceholder, if the query contains an invalid placeholder.
func (this *Connection) ExecQuery(ctx context.Context, query string, limit int) ([]byte, types.Type, error) {

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
	replacer := func(name string) (string, bool) {
		if strings.ToLower(name) == "limit" {
			return strconv.Itoa(limit), true
		}
		return "", false
	}
	database := this.database()
	defer database.Close()
	rows, err := database.Query(ctx, query, replacer)
	if err != nil {
		if err, ok := err.(connectors.PlaceholderError); ok {
			return nil, types.Type{}, errors.Unprocessable(InvalidPlaceholder, "%s", err)
		}
		return nil, types.Type{}, errors.Unprocessable(DatabaseFailed, "a database error occurred: %w", err)
	}
	defer rows.Close()

	// Scan the rows.
	var results []map[string]any
	for rows.Next() {
		row, err := rows.Scan()
		if err != nil {
			return nil, types.Type{}, errors.Unprocessable(DatabaseFailed, "a database error occurred: %w", err)
		}
		results = append(results, row)
	}
	err = rows.Err()
	if err != nil {
		return nil, types.Type{}, errors.Unprocessable(DatabaseFailed, "a database error occurred: %w", err)
	}

	schema, err := types.ObjectOf(rows.Columns())
	if err != nil {
		switch e := err.(type) {
		case types.InvalidPropertyNameError:
			err = errors.Unprocessable(DatabaseFailed, "the %s column has an invalid name: %q", ordinal(e.Index+1), e.Name)
		case types.RepeatedPropertyNameError:
			err = errors.Unprocessable(DatabaseFailed, "the names of the %s and %s columns are the same: %q", ordinal(e.Index1+1), ordinal(e.Index2+1), e.Name)
		}
		return nil, types.Type{}, err
	}
	marshaledRows, err := encoding.MarshalSlice(schema, results)
	if err != nil {
		return nil, types.Type{}, err
	}

	return marshaledRows, schema, nil
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
	case state.AppType, state.DatabaseType, state.StorageType, state.StreamType:
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

// Identities returns the users identities of the connection, and an estimate of
// their count without applying first and limit.
//
// It returns the user identities in range [first,first+limit] with first >= 0
// and 0 < limit <= 1000.
//
// It returns an errors.UnprocessableError error with code
//
//   - NoWarehouse, if the workspace does not have a data warehouse.
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
func (this *Connection) Identities(ctx context.Context, first, limit int) ([]byte, int, error) {
	this.apis.mustBeOpen()
	if first < 0 {
		return nil, 0, errors.BadRequest("first %d is not valid", limit)
	}
	if limit < 1 || limit > 1000 {
		return nil, 0, errors.BadRequest("limit %d is not valid", limit)
	}
	ws := this.connection.Workspace()
	if this.store == nil {
		return nil, 0, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", ws.ID)
	}
	apisWs := &Workspace{
		apis:      this.apis,
		store:     this.store,
		workspace: ws,
	}
	where := expr.NewBaseExpr("Connection", expr.OperatorEqual, this.connection.ID)
	identities, count, err := apisWs.userIdentities(ctx, where, first, limit)
	if err != nil {
		return nil, 0, err
	}
	if identities == nil {
		identities = []identity{}
	}
	data, err := json.Marshal(identities)
	return data, count, err
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
	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
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
// sheet must be a valid sheet name; otherwise, it must be an empty string.
// limit limits the number of records to return and must be in range [0, 100].
//
// A valid sheet name is UTF-8 encoded, has a length in the range [1, 31], does
// not start or end with "'", and does not contain any of "*", "/", ":", "?",
// "[", "\", and "]". Sheet names are case-insensitive.
//
// It returns an errors.UnprocessableError error with code
//
//   - ConnectorNotExist, if the connector does not exist.
//   - InvalidSettings, if the settings are not valid.
//   - NoColumns, if the file has no columns.
//   - ReadFileFailed, if an error occurred reading the file.
//   - SheetNotExist, if the file does not contain the provided sheet.
func (this *Connection) Records(ctx context.Context, fileConnector int, path, sheet string, compression Compression, settings []byte, limit int) ([]byte, types.Type, error) {

	this.apis.mustBeOpen()

	c := this.connection
	connector := c.Connector()

	// Validate the connection type.
	if connector.Type != state.StorageType {
		return nil, types.Type{}, errors.BadRequest("connection %d is not a storage connection", c.ID)
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

	// Validate the connector.
	if fileConnector < 1 || fileConnector > maxInt32 {
		return nil, types.Type{}, errors.BadRequest("connector identifier %d is not valid", fileConnector)
	}
	file, ok := this.apis.state.Connector(fileConnector)
	if !ok {
		return nil, types.Type{}, errors.Unprocessable(ConnectorNotExist, "connector %d does not exist", fileConnector)
	}
	if file.Type != state.FileType {
		return nil, types.Type{}, errors.BadRequest("type of the file connector must be File, got %v", file.Type)
	}

	// Validate the settings.
	if settings != nil && !file.HasSettings {
		return nil, types.Type{}, errors.BadRequest("settings cannot be provided because the File connector has no settings")
	}
	var validatedSettings []byte
	if file != nil && file.HasSettings {
		normalizedSettings := settings
		if settings == nil {
			normalizedSettings = json.RawMessage("{}")
		}
		conf := &connectors.ConnectorConfig{
			Role:   this.connection.Role,
			Region: state.PrivacyRegion(this.connection.Workspace().PrivacyRegion),
		}
		var err error
		validatedSettings, err = this.apis.connectors.ValidateSettings(ctx, file, conf, normalizedSettings)
		if err != nil {
			if err != connectors.ErrNoUserInterface {
				return nil, types.Type{}, errors.Unprocessable(InvalidSettings, "settings are not valid: %w", err)
			}
			if settings != nil {
				return nil, types.Type{}, errors.BadRequest("settings cannot be provided because %s connector %s does not have a UI",
					strings.ToLower(this.connection.Role.String()), file.Name)
			}
		} else if settings == nil {
			return nil, types.Type{}, errors.BadRequest("settings must be provided because %s connector %s has a UI",
				strings.ToLower(this.connection.Role.String()), file.Name)
		}
	}

	// Validate the sheet.
	if file.HasSheets {
		if sheet == "" {
			return nil, types.Type{}, errors.BadRequest("sheet cannot be empty because connection %d has sheets", c.ID)
		}
		if !connectors.IsValidSheetName(sheet) {
			return nil, types.Type{}, errors.BadRequest("sheet is not valid")
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

	// TODO(Gianluca): we should review the passing of parameters here, see the issue
	// https://github.com/open2b/chichi/issues/608.
	columns, records, err := this.storage().Read(ctx, file, path, sheet, validatedSettings, c.BusinessID, state.Compression(compression), limit)
	if err != nil {
		switch err {
		case connectors.ErrSheetNotExist:
			return nil, types.Type{}, errors.Unprocessable(SheetNotExist, "file does not contain any sheet named %q", sheet)
		case connectors.ErrNoColumns:
			return nil, types.Type{}, errors.Unprocessable(NoColumns, "file does not have columns")
		}
		return nil, types.Type{}, errors.Unprocessable(ReadFileFailed, "an error occurred reading the %s file: %w", connector.Name, err)
	}

	schema := types.Object(columns)
	marshaledRecords, err := encoding.MarshalSlice(schema, records)
	if err != nil {
		return nil, types.Type{}, err
	}

	return marshaledRecords, schema, nil
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
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
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
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
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
// have an event type named eventType. If there is a transformation, outSchema
// is the out schema of the transformation, and it must be a valid.
//
// It returns an errors.UnprocessableError error with code:
//   - EventTypeNotExist, if the event type does not exist for the connection.
//   - LanguageNotSupported, if the transformation language is not supported.
//   - NotCompatibleSchema, if the out schema is not compatible with the event
//     type's schema.
//   - TransformationFailed if the transformation fails due to an error in the
//     executed function.
func (this *Connection) PreviewSendEvent(ctx context.Context, eventType string, event *ObservedEvent, transformation Transformation, outSchema types.Type) ([]byte, error) {

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
	if transformation.Mapping != nil && transformation.Function != nil {
		return nil, errors.BadRequest("mapping and function transformations cannot both be present")
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

	var values map[string]any

	if transformation.Mapping != nil || transformation.Function != nil {

		if !outSchema.Valid() {
			return nil, errors.BadRequest("a transformation has been provided but out schema is not valid")
		}
		if outSchema.Kind() != types.ObjectKind {
			return nil, errors.BadRequest("out schema is not an Object")
		}

		// Use the schema without GID since events sent to the apps do not have
		// the GID, as they are sent as they are after enrichment; the GID is
		// added by the identity resolution directly to the data warehouse,
		// without affecting the events sent to the apps.
		inSchema := eventschema.SchemaWithoutGID

		// Validate the mapping and the transformation.
		switch {
		case transformation.Mapping != nil:
			for path, expr := range transformation.Mapping {
				outPath, err := types.ParsePropertyPath(path)
				if err != nil {
					return nil, errors.BadRequest("output mapped property %q is not valid", path)
				}
				p, err := outSchema.PropertyByPath(outPath)
				if err != nil {
					err := err.(types.PathNotExistError)
					return nil, errors.BadRequest("output mapped property %s not found in output schema", err.Path)
				}
				_, err = mappings.Compile(expr, inSchema, p.Type, p.Required, p.Nullable, nil)
				if err != nil {
					return nil, errors.BadRequest("invalid expression mapped to %s: %s", path, err)
				}
			}
		case transformation.Function != nil:
			if transformation.Function.Source == "" {
				return nil, errors.BadRequest("transformation source is empty")
			}
			tr := this.apis.functionTransformer
			switch transformation.Function.Language {
			case "JavaScript":
				if tr == nil || !tr.SupportLanguage(state.JavaScript) {
					return nil, errors.Unprocessable(LanguageNotSupported, "JavaScript transformation language  is not supported")
				}
			case "Python":
				if tr == nil || !tr.SupportLanguage(state.Python) {
					return nil, errors.Unprocessable(LanguageNotSupported, "Python transformation language is not supported")
				}
			case "":
				return nil, errors.BadRequest("transformation language is empty")
			default:
				return nil, errors.BadRequest("transformation language %q is not valid", transformation.Function.Language)
			}
		default:
			return nil, errors.BadRequest("mapping (or transformation) is required")
		}

		// Create a temporary transformer.
		var transformer transformers.Function
		var function *state.TransformationFunction
		if transformation.Function != nil {
			function = &state.TransformationFunction{
				Source:  transformation.Function.Source,
				Version: "1", // no matter the version, it will be overwritten by the temporary transformation.
			}
			name := "temp-" + uuid.NewString()
			switch transformation.Function.Language {
			case "JavaScript":
				name += ".js"
				function.Language = state.JavaScript
			case "Python":
				name += ".py"
				function.Language = state.Python
			}
			transformer = newTemporaryTransformer(name, transformation.Function.Source, this.apis.functionTransformer)
		}

		// Transform the values.
		action := 1 // no matter the action, it will be overwritten by the temporary transformation.
		tr := state.Transformation{
			Mapping:  transformation.Mapping,
			Function: function,
		}
		m, err := transformers.New(inSchema, outSchema, tr, action, transformer, nil)
		if err != nil {
			return nil, err
		}
		values, err = m.Transform(ctx, ev.ToMap())
		if err != nil {
			if err, ok := err.(transformers.FunctionExecutionError); ok {
				return nil, errors.Unprocessable(TransformationFailed, err.Error())
			}
			if err, ok := err.(ValidationError); ok {
				return nil, errors.Unprocessable(TransformationFailed, err.Error())
			}
			return nil, err
		}

	} else {

		if outSchema.Valid() {
			return nil, errors.BadRequest("out schema is a valid schema, but no transformation has been provided")
		}

	}

	req, err := this.app().EventRequest(ctx, eventType, ev.ToConnectorEvent(), values, true, outSchema)
	if err != nil {
		if err == connectors.ErrEventTypeNotExist {
			return nil, errors.Unprocessable(EventTypeNotExist, "connection %d does not have event type %q", c.ID, eventType)
		}
		if err, ok := err.(*connectors.SchemaError); ok {
			return nil, errors.Unprocessable(NotCompatibleSchema, "out schema is not compatible with the event type's schema: %w", err)
		}
		return nil, err
	}

	// Construct the preview.
	var b bytes.Buffer
	b.WriteString(req.Method)
	b.WriteString(" ")
	b.WriteString(req.URL)
	b.WriteByte('\n')
	err = req.Header.Write(&b)
	if err != nil {
		return nil, err
	}
	b.WriteByte('\n')
	ct := req.Header.Get("Content-Type")
	switch ct {
	case "application/json":
		err = json.Indent(&b, req.Body, "", "\t")
		if err != nil {
			return nil, err
		}
	case "application/x-ndjson":
		b.Write(req.Body)
	default:
		_, _ = fmt.Fprintf(&b, "[%d bytes body]", len(req.Body))
	}

	return b.Bytes(), nil
}

// Set sets the connection.
func (this *Connection) Set(ctx context.Context, connection ConnectionToSet) error {

	this.apis.mustBeOpen()

	ctx, span := telemetry.TraceSpan(ctx, "Connection.Set", "connection", this.connection.ID)
	defer span.End()

	if connection.Name == "" || utf8.RuneCountInString(connection.Name) > 100 {
		return errors.BadRequest("name %q is not valid", connection.Name)
	}
	if s := connection.Strategy; s != nil && !isValidStrategy(*s) {
		return errors.BadRequest("strategy %q is not valid", *s)
	}

	n := state.SetConnection{
		Connection:  this.connection.ID,
		Name:        connection.Name,
		Enabled:     connection.Enabled,
		Strategy:    (*state.Strategy)(connection.Strategy),
		WebsiteHost: connection.WebsiteHost,
		BusinessID:  connection.BusinessID,
	}

	c := this.connection.Connector()

	// Validate the strategy.
	if this.connection.Role == state.Source {
		switch c.Type {
		case state.MobileType, state.WebsiteType:
			if connection.Strategy == nil {
				return errors.BadRequest("%s connections must have a strategy", strings.ToLower(c.Type.String()))
			}
		default:
			if connection.Strategy != nil {
				return errors.BadRequest("%s connections cannot have a strategy", strings.ToLower(c.Type.String()))
			}
		}
	} else if connection.Strategy != nil {
		return errors.BadRequest("destination connections cannot have a strategy")
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

	// Validate the BusinessID.
	err := validateBusinessID(c.Type, this.connection.Role, n.BusinessID)
	if err != nil {
		return errors.BadRequest(err.Error())
	}

	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE connections SET name = $1, enabled = $2,"+
			" strategy = $3, website_host = $4, business_id = $5 WHERE id = $6",
			n.Name, n.Enabled, n.Strategy, n.WebsiteHost, n.BusinessID, n.Connection)
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
//   - InvalidSettings, if the settings are not valid.
//   - ReadFileFailed, if an error occurred reading the file.
func (this *Connection) Sheets(ctx context.Context, fileConnector int, path string, settings []byte, compression Compression) ([]string, error) {
	this.apis.mustBeOpen()
	connector := this.connection.Connector()
	if connector.Type != state.StorageType {
		return nil, errors.BadRequest("connection %d is not a storage", this.connection.ID)
	}
	if path == "" {
		return nil, errors.BadRequest("path is empty")
	}
	if !utf8.ValidString(path) {
		return nil, errors.BadRequest("path is not UTF-8 encoded")
	}

	// Validate the connector.
	if fileConnector < 1 || fileConnector > maxInt32 {
		return nil, errors.BadRequest("connector identifier %d is not valid", fileConnector)
	}
	file, ok := this.apis.state.Connector(fileConnector)
	if !ok {
		return nil, errors.Unprocessable(ConnectorNotExist, "connector %d does not exist", fileConnector)
	}
	if file.Type != state.FileType {
		return nil, errors.BadRequest("type of the file connector must be File, got %v", file.Type)
	}

	// Validate the settings.
	if settings != nil && !file.HasSettings {
		return nil, errors.BadRequest("settings cannot be provided because the file connector has no settings")
	}
	var validatedSettings []byte
	if file != nil && file.HasSettings {
		normalizedSettings := settings
		if settings == nil {
			normalizedSettings = json.RawMessage("{}")
		}
		conf := &connectors.ConnectorConfig{
			Role:   this.connection.Role,
			Region: state.PrivacyRegion(this.connection.Workspace().PrivacyRegion),
		}
		var err error
		validatedSettings, err = this.apis.connectors.ValidateSettings(ctx, file, conf, normalizedSettings)
		if err != nil {
			if err != connectors.ErrNoUserInterface {
				return nil, errors.Unprocessable(InvalidSettings, "settings are not valid: %w", err)
			}
			if settings != nil {
				return nil, errors.BadRequest("settings cannot be provided because %s connector %s does not have a UI",
					strings.ToLower(this.connection.Role.String()), file.Name)
			}
		} else if settings == nil {
			return nil, errors.BadRequest("settings must be provided because %s connector %s has a UI",
				strings.ToLower(this.connection.Role.String()), file.Name)
		}
	}

	sheets, err := this.storage().Sheets(ctx, file, path, validatedSettings, state.Compression(compression))
	if err != nil {
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
			if column.Type.Kind() != types.IntKind {
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
	schema, err := types.ObjectOf(columns)
	if err != nil {
		return types.Type{}, err
	}
	return schema, nil
}

// ActionType represents an action type.
type ActionType struct {
	Name        string
	Description string
	Target      Target
	EventType   *string
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
// Refer to the specifications in the file "apis/Actions.md" for more details.
//
// It returns an errors.UnprocessableError error with code
//
//   - FetchSchemaFailed, if an error occurred fetching the schema.
func (this *Connection) actionTypes(ctx context.Context) ([]ActionType, error) {
	var actionTypes []ActionType
	c := this.connection
	connector := c.Connector()
	targets := connector.Targets
	if targets.Contains(state.Users) {
		switch typ := c.Connector().Type; typ {
		case
			state.AppType,
			state.DatabaseType,
			state.StorageType:
			var name, description string
			if c.Role == state.Source {
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
			at := ActionType{
				Name:        name,
				Description: description,
				Target:      Users,
			}
			actionTypes = append(actionTypes, at)
		case
			state.MobileType,
			state.ServerType,
			state.WebsiteType:
			if c.Role == state.Source {
				at := ActionType{
					Name:        "Import users",
					Description: "Import users from the events of the " + connector.Name,
					Target:      Users,
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
			state.StorageType:
			var name, description string
			if c.Role == state.Source {
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
			at := ActionType{
				Name:        name,
				Description: description,
				Target:      Groups,
			}
			actionTypes = append(actionTypes, at)
		case
			state.MobileType,
			state.ServerType,
			state.WebsiteType:
			if c.Role == state.Source {
				at := ActionType{
					Name:        "Import groups",
					Description: "Import groups from the events of the " + connector.Name,
					Target:      Groups,
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
				actionTypes = slices.Insert(actionTypes, 0, at)
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
	// TODO(Gianluca): here we are not passing the identity and timestamp columns names.
	// And what about the Business ID? We should review this, see the issue
	// https://github.com/open2b/chichi/issues/608.
	return this.apis.connectors.Database(this.connection, "", "", "")
}

// storage returns the storage of the connection.
func (this *Connection) storage() *connectors.Storage {
	return this.apis.connectors.Storage(this.connection)
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
func (this *Connection) updateConnectionsStats(ctx context.Context, count int) error {
	_, err := this.apis.db.Exec(ctx, "INSERT INTO connections_stats AS cs (connection, time_slot, user_identities)\n"+
		"VALUES ($1, $2, $3)\n"+
		"ON CONFLICT (connection, time_slot) DO UPDATE SET user_identities = cs.user_identities + $3",
		this.connection.ID, statsTimeSlot(time.Now().UTC()), count)
	return err
}

// validateActionToSet validates the action to set (when adding or setting an
// action) for the given target.
//
// fileConnector must be passed exclusively and necessarily when the connector of the
// storage has type Storage, otherwise it must be nil.
//
// Refer to the specifications in the file "apis/Actions.md" for more details.
//
// It returns an errors.UnprocessableError error with code LanguageNotSupported,
// if the transformation language is not supported.
func (this *Connection) validateActionToSet(action ActionToSet, target state.Target, fileConnector *state.Connector) error {

	inSchema := action.InSchema
	outSchema := action.OutSchema

	importUsersIdentitiesFromEvents := importsUsersIdentitiesFromEvents(this.connection.Connector().Type, this.connection.Role, target)
	if importUsersIdentitiesFromEvents {
		if inSchema.Valid() {
			return errors.BadRequest("input schema must be invalid for actions that import users identities from events")
		}
		// The input schema is the events schema without GID because this
		// actions imports users identities from incoming events, which,
		// clearly, still do not have any user associated.
		inSchema = eventschema.SchemaWithoutGID
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
	if inSchema.Valid() && inSchema.Kind() != types.ObjectKind {
		return errors.BadRequest("input schema, if provided, must be an object")
	}
	if outSchema.Valid() && outSchema.Kind() != types.ObjectKind {
		return errors.BadRequest("out schema, if provided, must be an object")
	}
	// Validate the filter.
	var usedInPaths []types.Path
	if action.Filter != nil {
		if !inSchema.Valid() {
			return errors.BadRequest("input schema is required by the filter")
		}
		properties, err := validateFilter(action.Filter, inSchema)
		if err != nil {
			return errors.BadRequest("filter is not valid: %w", err)
		}
		usedInPaths = properties
	}
	// An action cannot have both mappings and transformations.
	if action.Transformation.Mapping != nil && action.Transformation.Function != nil {
		return errors.BadRequest("action cannot have both mappings and transformation")
	}
	// Validate the mapping.
	var usedOutPaths []types.Path
	var mappingInProperties int
	if mapping := action.Transformation.Mapping; mapping != nil {
		if len(mapping) == 0 {
			return errors.BadRequest("transformation mapping must have mapped properties")
		}
		if !inSchema.Valid() {
			return errors.BadRequest("input schema is required by the mapping")
		}
		if !outSchema.Valid() {
			return errors.BadRequest("output schema is required by the mapping")
		}
		transformer, err := mappings.New(mapping, inSchema, outSchema, nil)
		if err != nil {
			return errors.BadRequest("invalid mapping: %s", err)
		}
		// Input properties.
		inProps := transformer.Properties()
		mappingInProperties = len(inProps)
		usedInPaths = append(usedInPaths, inProps...)
		// Output properties.
		for m := range mapping {
			path, err := types.ParsePropertyPath(m)
			if err != nil {
				return errors.BadRequest("invalid property path %q", m)
			}
			usedOutPaths = append(usedOutPaths, path)
		}
	}
	// Validate the transformation.
	if function := action.Transformation.Function; function != nil {
		if !inSchema.Valid() {
			return errors.BadRequest("input schema is required by the transformation")
		}
		if !outSchema.Valid() {
			return errors.BadRequest("output schema is required by the transformation")
		}
		if function.Source == "" {
			return errors.BadRequest("function transformation source is empty")
		}
		tr := this.apis.functionTransformer
		switch function.Language {
		case "JavaScript":
			if tr == nil || !tr.SupportLanguage(state.JavaScript) {
				return errors.Unprocessable(LanguageNotSupported, "JavaScript transformation language is not supported")
			}
		case "Python":
			if tr == nil || !tr.SupportLanguage(state.Python) {
				return errors.Unprocessable(LanguageNotSupported, "Python transformation language is not supported")
			}
		case "":
			return errors.BadRequest("transformation language is empty")
		default:
			return errors.BadRequest("transformation language %q is not valid", action.Transformation.Function.Language)
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
		switch this.connection.Role {
		case state.Source:
			_, err := connectors.ReplacePlaceholders(action.Path, func(_ string) (string, bool) {
				return "", false
			})
			if err != nil {
				return errors.BadRequest("placeholders syntax is not supported by source actions")
			}
		case state.Destination:
			_, err := connectors.ReplacePlaceholders(action.Path, func(name string) (string, bool) {
				name = strings.ToLower(name)
				return "", name == "today" || name == "now" || name == "unix"
			})
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
	if action.Sheet != "" && !connectors.IsValidSheetName(action.Sheet) {
		return errors.BadRequest("sheet name is not valid")
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
		// Validate the internal matching property.
		if !types.IsValidPropertyName(props.Internal) {
			return errors.BadRequest("internal matching property %q is not a valid property name", props.Internal)
		}
		if !inSchema.Valid() {
			return errors.BadRequest("input schema must be valid")
		}
		internal, ok := inSchema.Property(props.Internal)
		if !ok {
			return errors.BadRequest("internal matching property %q not found within the input schema", props.Internal)
		}
		if !canBeUsedAsAsMatchingProp(internal.Type.Kind()) {
			return errors.BadRequest("type %s cannot be used as matching property", internal.Type)
		}
		usedInPaths = append(usedInPaths, types.Path{props.Internal})
		// Validate the external matching property.
		if !types.IsValidPropertyName(props.External.Name) {
			return errors.BadRequest("external matching property %q is not a valid property name", props.External.Name)
		}
		if !props.External.Type.Valid() {
			return errors.BadRequest("external matching property type is not valid")
		}
		if !canBeUsedAsAsMatchingProp(props.External.Type.Kind()) {
			return errors.BadRequest("type %s cannot be used as matching property", props.External.Type)
		}
	}
	// Validate the compression.
	switch action.Compression {
	case NoCompression, ZipCompression, GzipCompression, SnappyCompression:
	default:
		return errors.BadRequest("compression %q is not valid", action.Compression)
	}

	// Second, do validations based on the workspace and the connection.

	c := this.connection
	connector := c.Connector()
	eventBasedConn := connector.Type == state.MobileType ||
		connector.Type == state.ServerType ||
		connector.Type == state.WebsiteType

	// In case of a source connection, since its actions write on the data
	// warehouse, the output schema cannot contain meta properties because such
	// properties are not writable by user transformations.
	if c.Role == state.Source && outSchema.Valid() {
		for _, p := range outSchema.Properties() {
			if isMetaProperty(p.Name) {
				return errors.BadRequest("output schema cannot contain meta properties")
			}
		}
	}

	// Check if the settings and the compression are allowed.
	if connector.Type == state.StorageType {
		if action.Settings == nil {
			return errors.BadRequest("actions on Storage connections must have settings")
		}
	} else {
		if action.Settings != nil {
			return errors.BadRequest("actions on %v connections cannot have settings", connector.Type)
		}
		if action.Compression != NoCompression {
			return errors.BadRequest("actions on %v connections cannot have a compression", connector.Type)
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
	case state.StorageType:
		filtersAllowed = targetUsersOrGroups && c.Role == state.Destination
	}
	if action.Filter != nil && !filtersAllowed {
		return errors.BadRequest("filters are not allowed")
	}

	// Check if the path and the sheet are allowed.
	if connector.Type == state.StorageType {
		if action.Path == "" {
			return errors.BadRequest("path cannot be empty for actions on storage connections")
		}
		if fileConnector.HasSheets && action.Sheet == "" {
			return errors.BadRequest("sheet cannot be empty because connector %d has sheets", fileConnector.ID)
		}
		if !fileConnector.HasSheets && action.Sheet != "" {
			return errors.BadRequest("connector %d does not have sheets", fileConnector.ID)
		}
	} else {
		if action.Path != "" {
			return errors.BadRequest("%s actions cannot have a path", connector.Type)
		}
		if action.Sheet != "" {
			return errors.BadRequest("%s actions cannot have a sheet", connector.Type)
		}
	}

	// Check the column for the identity and for the timestamp.
	importFromColumns := c.Role == state.Source &&
		(connector.Type == state.StorageType || connector.Type == state.DatabaseType)
	if importFromColumns {
		if !inSchema.Valid() {
			return errors.BadRequest("input schema must be valid")
		}
		// Validate the identity column.
		if action.IdentityColumn == "" {
			return errors.BadRequest("column name for the identity is mandatory")
		}
		if !types.IsValidPropertyName(action.IdentityColumn) {
			return errors.BadRequest("column name for the identity has not a valid property name")
		}
		if utf8.RuneCountInString(action.IdentityColumn) > 1024 {
			return errors.BadRequest("column name for the identity is longer than 1024 runes")
		}
		identityColumn, ok := inSchema.Property(action.IdentityColumn)
		if !ok {
			return errors.BadRequest("identity column %q not found within input schema", action.IdentityColumn)
		}
		switch k := identityColumn.Type.Kind(); k {
		case types.IntKind, types.UintKind, types.UUIDKind, types.JSONKind, types.TextKind:
		default:
			return fmt.Errorf("identity column %q has kind %s instead of Int, Uint, UUID, JSON, or Text", action.IdentityColumn, k)
		}
		usedInPaths = append(usedInPaths, types.Path{action.IdentityColumn})
		// Validate the timestamp column and format.
		var requiresTimestampFormat bool
		if action.TimestampColumn != "" {
			if !types.IsValidPropertyName(action.TimestampColumn) {
				return errors.BadRequest("column name for the timestamp has a not valid property name")
			}
			if utf8.RuneCountInString(action.TimestampColumn) > 1024 {
				return errors.BadRequest("column name for the timestamp is longer than 1024 runes")
			}
			timestampColumn, ok := inSchema.Property(action.TimestampColumn)
			if !ok {
				return errors.BadRequest("timestamp column %q not found within input schema", action.TimestampColumn)
			}
			switch k := timestampColumn.Type.Kind(); k {
			case types.DateTimeKind, types.DateKind:
			case types.JSONKind, types.TextKind:
				requiresTimestampFormat = true
			default:
				return fmt.Errorf("timestamp column %q has kind %s instead of DateTime, Date, JSON, or Text", action.TimestampColumn, k)
			}
			usedInPaths = append(usedInPaths, types.Path{action.TimestampColumn})
		}
		if !requiresTimestampFormat && action.TimestampFormat != "" {
			return errors.BadRequest("action cannot specify a timestamp format")
		} else if requiresTimestampFormat && action.TimestampFormat == "" {
			return errors.BadRequest("timestamp format is required")
		}
		if requiresTimestampFormat {
			if err := validateTimestampFormat(action.TimestampFormat); err != nil {
				return errors.BadRequest(err.Error())
			}
		}
	} else {
		if action.IdentityColumn != "" {
			return errors.BadRequest("action cannot specify a column name for the identity")
		}
		if action.TimestampColumn != "" {
			return errors.BadRequest("action cannot specify a column name for the timestamp")
		}
		if action.TimestampFormat != "" {
			return errors.BadRequest("action cannot specify a timestamp format")
		}
	}

	// When exporting users to file, ensure that the output schema is valid, as
	// it contains the properties that will be exported to the file.
	if connector.Type == state.StorageType && c.Role == state.Destination && target == state.Users {
		if !outSchema.Valid() {
			return errors.BadRequest("output schema cannot be empty when exporting users to file")
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
		if action.ExportOnDuplicatedUsers == nil {
			return errors.BadRequest("export on duplicated users setting cannot be nil")
		}
	} else {
		if action.ExportMode != nil {
			return errors.BadRequest("export mode must be nil")
		}
		if action.MatchingProperties != nil {
			return errors.BadRequest("matching properties must be nil")
		}
		if action.ExportOnDuplicatedUsers != nil {
			return errors.BadRequest("export on duplicated users setting must be nil")
		}
	}

	// Check the connections for which the transformation is prohibited.
	transformationProhibited := (c.Role == state.Source && eventBasedConn && target == state.Events) ||
		(c.Role == state.Destination && connector.Type == state.StorageType && targetUsersOrGroups)
	haveTransformation := action.Transformation.Mapping != nil || action.Transformation.Function != nil
	if transformationProhibited && haveTransformation {
		return errors.BadRequest("action cannot have a transformation")
	}

	// Check the connections for which the transformation function is
	// prohibited.
	if haveTransformation {
		funcForbidden := c.Role == state.Source && eventBasedConn && targetUsersOrGroups
		if funcForbidden && action.Transformation.Function != nil {
			return errors.BadRequest("action cannot have a transformation function")
		}
	}

	// Check if the transformation is mandatory, with at least one input
	// property.
	//
	// For mappings, at least one property path must appear in the input
	// expressions.
	//
	// For transformation functions, since every property of the input schema is
	// passed to the function, the input schema must be valid (thus it must
	// contain at least one property).
	transformationMandatory := targetUsersOrGroups &&
		(connector.Type == state.AppType || connector.Type == state.DatabaseType ||
			(c.Role == state.Source && connector.Type == state.StorageType))
	if transformationMandatory && !haveTransformation {
		return errors.BadRequest("action must have a transformation")
	}
	if action.Transformation.Mapping != nil && mappingInProperties == 0 {
		return errors.BadRequest("transformation must map at least one property")
	}
	if action.Transformation.Function != nil && !inSchema.Valid() {
		return errors.BadRequest("transformation function must have at least one input property")
	}

	// Ensure that every property in the input and output schemas have been used
	// (by the mappings, by the filters, etc...).
	if action.Transformation.Function != nil {
		// The action has a transformation function, so we do not know which
		// properties are used; consequently, this check would always pass
		// because we would consider every property of the schema as used.
	} else if importUsersIdentitiesFromEvents {
		// In this case the input schema is the full schema of the events, both
		// in case of mappings and transformation, so we cannot return the error
		// about unused properties in input schema because just a minor part of
		// them is generally used.
		if usedOutPaths != nil {
			if props := unusedProperties(outSchema, usedOutPaths); props != nil {
				return errors.BadRequest("output schema contains unused properties: %s", strings.Join(props, ", "))
			}
		}
	} else {
		if usedInPaths != nil {
			if props := unusedProperties(inSchema, usedInPaths); props != nil {
				return errors.BadRequest("input schema contains unused properties: %s", strings.Join(props, ", "))
			}
		}
		if usedOutPaths != nil {
			if props := unusedProperties(outSchema, usedOutPaths); props != nil {
				return errors.BadRequest("output schema contains unused properties: %s", strings.Join(props, ", "))
			}
		}
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

	// Strategy is the strategy that determines how to merge anonymous and
	// non-anonymous users. It must be nil for destination connections and
	// non-event source connections.
	Strategy *Strategy

	// WebsiteHost is the host, in the form "host:port", of a website
	// connection. It must be empty if the connection is not a website. It
	// cannot be longer than 261 runes.
	WebsiteHost string

	// BusinessID is the Business ID property or column (depending on the type of the
	// connection) for source connections that import users. May be the empty string to
	// indicate to not import the Business ID.
	BusinessID string
}

// canBeUsedAsAsMatchingProp reports whether a type with kind k can be used as a
// matching property for the export.
func canBeUsedAsAsMatchingProp(k types.Kind) bool {
	// Only integers, UUIDs and texts are allowed.
	return k == types.IntKind || k == types.UintKind || k == types.UUIDKind || k == types.TextKind
}

// importsUsersIdentitiesFromEvents reports whether a connector with the given
// type, on a connection with the given role, with an action with the given
// target, imports users identities from events.
func importsUsersIdentitiesFromEvents(connectorType state.ConnectorType, role state.Role, target state.Target) bool {
	if role == state.Source && target == state.Users {
		switch connectorType {
		case state.MobileType, state.ServerType, state.WebsiteType:
			return true
		}
	}
	return false
}

// isMetaProperty reports whether the given property name refers to a property
// considered a meta-property by a data warehouse.
func isMetaProperty(name string) bool {
	for _, r := range name {
		return unicode.IsUpper(r)
	}
	return false
}

// validateBusinessID validates a Business ID name (column or property). In case it is
// invalid returns an errors.BadRequest, otherwise returns nil.
func validateBusinessID(cType state.ConnectorType, role state.Role, businessID string) error {

	// An empty Business ID is always valid.
	if businessID == "" {
		return nil
	}

	// BusinessID can be defined only for source connections.
	if role == state.Destination {
		return errors.BadRequest("unexpected Business ID for destination connection")
	}

	// Validate the Business ID name.
	if n := utf8.RuneCountInString(businessID); n > 1024 {
		return errors.BadRequest("Business ID name is longer than 1024 runes")
	}
	switch cType {
	case state.AppType, state.MobileType, state.ServerType, state.WebsiteType:
		if !types.IsValidPropertyName(businessID) {
			return errors.BadRequest("Business ID name %q is not a valid property name", businessID)
		}
	case state.DatabaseType, state.StorageType:
		if !utf8.ValidString(businessID) {
			return errors.BadRequest("Business ID name is not UTF-8 encoded")
		}
	default:
		return errors.BadRequest("unexpected Business ID for %s connection", cType)
	}

	return nil
}

// validateTimestampFormat validates the given timestamp format for importing
// files, returning an error in case the format is not valid.
//
// NOTE: keep in sync with the function 'apis/connectors.parseTimestamp'.
func validateTimestampFormat(format string) error {
	if !utf8.ValidString(format) {
		return errors.New("timestamp format must be UTF-8 valid")
	}
	if utf8.RuneCountInString(format) > 64 {
		return errors.New("timestamp format is longer than 64 runes")
	}
	switch format {
	case
		"DateTime",
		"DateOnly",
		"ISO8601",
		"Excel":
		return nil
	default:
		f, ok := strings.CutPrefix(format, "'")
		if !ok {
			return fmt.Errorf("invalid timestamp format %q", format)
		}
		_, ok = strings.CutSuffix(f, "'")
		if !ok {
			return fmt.Errorf("invalid timestamp format %q", format)
		}
		return nil
	}
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

// onlyForMatching returns a schema which contains only the properties of schema
// which can be used for the apps export matching.
//
// Returns an invalid schema in case none of the properties of schema can be
// used.
func onlyForMatching(schema types.Type) types.Type {
	props := []types.Property{}
	for _, p := range schema.Properties() {
		if canBeUsedAsAsMatchingProp(p.Type.Kind()) {
			props = append(props, p)
		}
	}
	if len(props) == 0 {
		return types.Type{}
	}
	return types.Object(props)
}

// removeMetaProperties removes the properties considered meta properties by the
// data warehouses from the schema, and returns it as a new schema.
func removeMetaProperties(schema types.Type) types.Type {
	props := schema.Properties()
	noMetaProps := make([]types.Property, 0, len(props))
	for _, p := range props {
		if isMetaProperty(p.Name) {
			continue
		}
		noMetaProps = append(noMetaProps, p)
	}
	return types.Object(noMetaProps)
}

// statsTimeSlot returns the stats time slot for the time t.
// t must be a UTC time.
func statsTimeSlot(t time.Time) int {
	epoch := int(t.Unix())
	return epoch / (60 * 60)
}

// unusedProperties returns the names of the unused properties in schema, if
// there is at least one, otherwise returns nil. schema must be valid.
func unusedProperties(schema types.Type, used []types.Path) []string {
	schemaProps := schema.PropertiesNames()
	notUsed := make(map[string]struct{}, len(schemaProps))
	for _, p := range schemaProps {
		notUsed[p] = struct{}{}
	}
	for _, path := range used {
		delete(notUsed, path[0])
	}
	if len(notUsed) == 0 {
		return nil
	}
	props := maps.Keys(notUsed)
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
	// (Refer to the specifications in the file "apis/Actions.md" for more
	// details)
	c := this.connection
	connector := c.Connector()
	var supported bool
	switch connector.Type {
	case state.AppType:
		supported = c.Role == state.Destination || target != Events
	case state.DatabaseType, state.StorageType:
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
func deserializeCursor(cursor string) (state.Cursor, error) {
	data, err := hex.DecodeString(cursor)
	if err != nil {
		return state.Cursor{}, err
	}
	var c state.Cursor
	err = json.Unmarshal(data, &c)
	if err != nil {
		return state.Cursor{}, err
	}
	// TODO(marco): validate the cursor's fields.
	return c, nil
}

// Ordinal returns the ordinal form of n.
func ordinal(n int) string {
	if n >= 11 && n <= 13 {
		return fmt.Sprintf("%dth", n)
	}
	switch n % 10 {
	case 1:
		return fmt.Sprintf("%dst", n)
	case 2:
		return fmt.Sprintf("%dnd", n)
	case 3:
		return fmt.Sprintf("%drd", n)
	}
	return fmt.Sprintf("%dth", n)
}

// serializeCursor serializes a cursor to be returned by the API.
func serializeCursor(cursor state.Cursor) (string, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	err := enc.Encode(cursor)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b.Bytes()), nil
}

// temporaryTransformer is a transformers.Function that creates a function
// at each call and deletes it after the call returns. Any call to a method that
// is not CallFunction panics.
type temporaryTransformer struct {
	name        string                // function name.
	source      string                // source code.
	transformer transformers.Function // underlying transformer.
}

func newTemporaryTransformer(name, source string, transformer transformers.Function) *temporaryTransformer {
	return &temporaryTransformer{name, source, transformer}
}

func (tp *temporaryTransformer) Call(ctx context.Context, _, _ string, inSchema, outSchema types.Type, values []map[string]any) ([]transformers.Result, error) {
	version, err := tp.transformer.Create(ctx, tp.name, tp.source)
	if err != nil {
		return nil, nil
	}
	defer func() {
		go func() {
			err := tp.transformer.Delete(context.Background(), tp.name)
			if err != nil {
				slog.Warn("cannot delete transformation function", "name", tp.name, "err", err)
			}
		}()
	}()
	return tp.transformer.Call(ctx, tp.name, version, inSchema, outSchema, values)
}

func (tp *temporaryTransformer) Close(_ context.Context) error { panic("not supported") }
func (tp *temporaryTransformer) Create(_ context.Context, _, _ string) (string, error) {
	panic("not supported")
}
func (tp *temporaryTransformer) Delete(_ context.Context, _ string) error {
	panic("not supported")
}
func (tp *temporaryTransformer) SupportLanguage(_ state.Language) bool { panic("not supported") }
func (tp *temporaryTransformer) Update(_ context.Context, _, _ string) (string, error) {
	panic("not supported")
}
