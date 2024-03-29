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

	"github.com/open2b/chichi/apis/connectors"
	"github.com/open2b/chichi/apis/datastore"
	"github.com/open2b/chichi/apis/datastore/expr"
	"github.com/open2b/chichi/apis/encoding"
	"github.com/open2b/chichi/apis/errors"
	"github.com/open2b/chichi/apis/events"
	"github.com/open2b/chichi/apis/events/eventschema"
	"github.com/open2b/chichi/apis/postgres"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/apis/transformers"
	"github.com/open2b/chichi/apis/transformers/mappings"
	"github.com/open2b/chichi/telemetry"
	"github.com/open2b/chichi/types"

	"github.com/google/uuid"
	"github.com/jxskiss/base62"
)

const (
	maxKeysPerConnection = 20 // maximum number of keys per connection.
	maxInt32             = math.MaxInt32
	rawSchemaMaxSize     = 16_777_215 // maximum size in runes for schemas stored in PostgreSQL.
	queryMaxSize         = 16_777_215 // maximum size in runes of a connection query.
)

const (
	ConnectionNotExist      errors.Code = "ConnectionNotExist"
	ConnectorNotExist       errors.Code = "ConnectorNotExist"
	EventConnectionNotExist errors.Code = "EventConnectionNotExist"
	EventNotExist           errors.Code = "EventNotExist"
	EventTypeNotExist       errors.Code = "EventTypeNotExist"
	FetchSchemaFailed       errors.Code = "FetchSchemaFailed"
	InvalidPath             errors.Code = "InvalidPath"
	InvalidPlaceholder      errors.Code = "InvalidPlaceholder"
	InvalidTable            errors.Code = "InvalidTable"
	KeyNotExist             errors.Code = "KeyNotExist"
	LanguageNotSupported    errors.Code = "LanguageNotSupported"
	NoColumns               errors.Code = "NoColumns"
	NotCompatibleSchema     errors.Code = "NotCompatibleSchema"
	ReadFileFailed          errors.Code = "ReadFileFailed"
	SheetNotExist           errors.Code = "SheetNotExist"
	TargetAlreadyExist      errors.Code = "TargetAlreadyExist"
	TooManyKeys             errors.Code = "TooManyKeys"
	UniqueKey               errors.Code = "UniqueKey"
	WorkspaceNotExist       errors.Code = "WorkspaceNotExist"
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
	apis             *APIs
	connection       *state.Connection
	store            *datastore.Store
	ID               int
	Name             string
	Type             ConnectorType
	Role             Role
	Enabled          bool
	Connector        int
	Strategy         *Strategy
	SendingMode      *SendingMode
	WebsiteHost      string
	EventConnections []int
	HasSettings      bool
	ActionsCount     int
	Health           Health

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

	case state.FileStorageType:
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
	actionOnFile := this.connection.Connector().Type == state.FileStorageType
	if actionOnFile && action.Connector == 0 {
		return 0, errors.BadRequest("actions on file storage connections must have a connector")
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
	err := validateActionToSet(action, state.Target(target), this.connection, fileConnector, this.apis.functionTransformer)
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
		UniqueIDColumn:          action.UniqueIDColumn,
		TimestampColumn:         action.TimestampColumn,
		TimestampFormat:         action.TimestampFormat,
		DisplayedID:             action.DisplayedID,
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
			"connector, path, sheet, compression, settings, table_name, unique_id_column,\n" +
			"timestamp_column, timestamp_format, export_mode, matching_properties_internal,\n" +
			"matching_properties_external, export_on_duplicated_users)\n" +
			"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,\n" +
			"$17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29)"
		_, err := tx.Exec(ctx, query, n.ID, n.Connection, n.Target, n.EventType,
			n.Name, n.Enabled, n.ScheduleStart, n.SchedulePeriod, rawInSchema, rawOutSchema,
			string(filter), mapping, function.Source, function.Language, function.Version,
			n.Query, connectorID, n.Path, n.Sheet, n.Compression, string(n.Settings), n.TableName,
			n.UniqueIDColumn, n.TimestampColumn, n.TimestampFormat, n.ExportMode,
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

// AddEventConnection adds the connection with identifier id to the event
// connections of the connection and vice versa.
// If the connection to add does not exist, it returns an
// errors.UnprocessableError error with code EventConnectionNotExist.
func (this *Connection) AddEventConnection(ctx context.Context, id int) error {
	this.apis.mustBeOpen()
	// If the connection already has the event connection,
	// it can return without additional validation.
	if slices.Contains(this.connection.EventConnections, id) {
		return nil
	}
	// Validate the event connection.
	c := this.connection.Connector()
	ws := this.connection.Workspace()
	role := this.connection.Role
	err := validateEventConnections([]int{id}, c, ws, role)
	if err != nil {
		return err
	}
	n := state.AddEventConnection{
		Connections: [2]int{this.connection.ID, id},
	}
	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		const add = "UPDATE connections\n" +
			"SET event_connections = (SELECT ARRAY(SELECT DISTINCT unnest(array_append(event_connections, $1)) ORDER BY 1))\n" +
			"WHERE id = $2"
		result, err := tx.Exec(ctx, add, n.Connections[1], n.Connections[0])
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return nil
		}
		result, err = tx.Exec(ctx, add, n.Connections[0], n.Connections[1])
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.Unprocessable(EventConnectionNotExist, "event connection %d does not exist", n.Connections[1])
		}
		return tx.Notify(ctx, n)
	})
	return err
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
	records, err := this.app().Users(ctx, schema, "", cur)
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
//   - InvalidPath, if path is not valid for the file storage connector.
//   - InvalidPlaceholder, if path for source connections contains a placeholder
//     or path for destination connections contains an invalid placeholder.
func (this *Connection) CompletePath(ctx context.Context, path string) (string, error) {
	this.apis.mustBeOpen()
	c := this.connection
	if c.Connector().Type != state.FileStorageType {
		return "", errors.BadRequest("connection %d is not a file storage connection", c.ID)
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
	workspace := this.connection.Workspace()
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "DELETE FROM connections WHERE id = $1", n.ID)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.NotFound("connection %d does not exist", n.ID)
		}
		role := "Source"
		if this.connection.Role == state.Source {
			role = "Destination"
		}
		// Remove the connection from the event connections.
		_, err = tx.Exec(ctx, "UPDATE connections\n"+
			"SET event_connections =\n"+
			"\tCASE\n"+
			"\t\tWHEN array_remove(event_connections, $1) = '{}' THEN NULL\n"+
			"\t\tELSE array_remove(event_connections, $1)\n"+
			"\tEND\n"+
			"WHERE workspace = $2 AND role = $3 AND event_connections IS NOT NULL AND $1 = ANY(event_connections)",
			n.ID, workspace.ID, role)
		if err != nil {
			return err
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
	case state.AppType, state.DatabaseType, state.FileStorageType, state.StreamType:
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
	if connector.Type != state.FileStorageType {
		return nil, types.Type{}, errors.BadRequest("connection %d is not a file storage connection", c.ID)
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

	columns, records, err := this.storage().Read(ctx, file, path, sheet, validatedSettings, state.Compression(compression), limit)
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

// RemoveEventConnection removes the connection with identifier id from the
// event connections of the connection and vice versa.
// If the connection to remove does not exist, it returns an
// errors.UnprocessableError error with code EventConnectionNotExist.
func (this *Connection) RemoveEventConnection(ctx context.Context, id int) error {
	this.apis.mustBeOpen()
	// Validate the event connection.
	c := this.connection.Connector()
	ws := this.connection.Workspace()
	role := this.connection.Role
	err := validateEventConnections([]int{id}, c, ws, role)
	if err != nil {
		return err
	}
	// If the connection does not have the event connection, it can return.
	if !slices.Contains(this.connection.EventConnections, id) {
		return nil
	}
	n := state.RemoveEventConnection{
		Connections: [2]int{this.connection.ID, id},
	}
	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		const remove = "UPDATE connections\n" +
			"SET event_connections =\n" +
			"\tCASE\n" +
			"\t\tWHEN array_remove(event_connections, $1) = '{}' THEN NULL\n" +
			"\t\tELSE array_remove(event_connections, $1)\n" +
			"\tEND\n" +
			"WHERE id = $2 AND $1 = ANY(event_connections)"
		result, err := tx.Exec(ctx, remove, n.Connections[1], n.Connections[0])
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return nil
		}
		result, err = tx.Exec(ctx, remove, n.Connections[0], n.Connections[1])
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.Unprocessable(EventConnectionNotExist, "event connection %d does not exist", n.Connections[1])
		}
		return tx.Notify(ctx, n)
	})
	return err
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
	if sm := connection.SendingMode; sm != nil && !isValidSendingMode(*sm) {
		return errors.BadRequest("sending mode %q is not valid", *sm)
	}

	n := state.SetConnection{
		Connection:  this.connection.ID,
		Name:        connection.Name,
		Enabled:     connection.Enabled,
		Strategy:    (*state.Strategy)(connection.Strategy),
		SendingMode: (*state.SendingMode)(connection.SendingMode),
		WebsiteHost: connection.WebsiteHost,
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

	// Validate the sending mode.
	if this.connection.Role == state.Destination {
		if c.SendingMode != nil {
			if connection.SendingMode == nil {
				return errors.BadRequest("connector %s requires a sending mode", c.Name)
			}
			if !c.SendingMode.Contains(state.SendingMode(*connection.SendingMode)) {
				return errors.BadRequest("connector %s does not support sending mode %s", c.Name, *c.SendingMode)
			}
		} else if connection.SendingMode != nil {
			return errors.BadRequest("connector %s does not support sending modes", c.Name)
		}
	} else if connection.SendingMode != nil {
		return errors.BadRequest("source connections cannot have a sending mode")
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

	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE connections SET name = $1, enabled = $2,"+
			" strategy = $3, sending_mode = $4, website_host = $5 WHERE id = $6",
			n.Name, n.Enabled, n.Strategy, n.SendingMode, n.WebsiteHost, n.Connection)
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
	if connector.Type != state.FileStorageType {
		return nil, errors.BadRequest("connection %d is not a file storage", this.connection.ID)
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
			state.FileStorageType:
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
			state.FileStorageType:
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
				description := "Import events from the "
				switch typ {
				case state.MobileType:
					description += "mobile app"
				case state.ServerType:
					description += "server"
				case state.WebsiteType:
					description += "website"
				}
				at := ActionType{
					Name:        "Import events",
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
	return this.apis.connectors.Database(this.connection)
}

// storage returns the storage of the connection.
func (this *Connection) storage() *connectors.FileStorage {
	return this.apis.connectors.FileStorage(this.connection)
}

// validateEventConnections validates the given event connections for a
// connection with the provided connector, workspace and role. If they are not
// valid, it returns an errors.BadRequestError error. If a connection does not
// exist, it returns an errors.UnprocessableError error with code
// EventConnectionNotExist.
func validateEventConnections(connections []int, c *state.Connector, ws *state.Workspace, role state.Role) error {
	if connections == nil {
		return nil
	}
	if len(connections) == 0 {
		return errors.BadRequest("event connections cannot be empty")
	}
	if !c.Targets.Contains(state.Events) {
		return errors.BadRequest("connector %d does not support event connections", c.ID)
	}
	for i, id := range connections {
		if id < 1 || id > maxInt32 {
			return errors.BadRequest("event connection %d is not a valid connection identifier", id)
		}
		for j := i + 1; j < len(connections); j++ {
			if connections[j] == id {
				return errors.BadRequest("event connection %d is repeated", id)
			}
		}
		ec, ok := ws.Connection(id)
		if !ok {
			return errors.Unprocessable(EventConnectionNotExist, "event connection %d does not exist", id)
		}
		if !ec.Connector().Targets.Contains(state.Events) {
			return errors.BadRequest("event connection %d does not support events", id)
		}
		if ec.Role == role {
			if ec.Role == state.Source {
				return errors.BadRequest("event connection %d is not a destination", id)
			}
			return errors.BadRequest("event connection %d is not a source", id)
		}
	}
	return nil
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

	// SendingMode is the mode used for sending events. It must be nil for
	// source connections and connections that does not support events.
	SendingMode *SendingMode

	// WebsiteHost is the host, in the form "host:port", of a website
	// connection. It must be empty if the connection is not a website. It
	// cannot be longer than 261 runes.
	WebsiteHost string
}

// isMetaProperty reports whether the given property name refers to a property
// considered a meta-property by a data warehouse.
func isMetaProperty(name string) bool {
	for _, r := range name {
		return unicode.IsUpper(r)
	}
	return false
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
	case state.DatabaseType, state.FileStorageType:
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
