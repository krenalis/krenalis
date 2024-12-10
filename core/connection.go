//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package core

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	mathrand "math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/connectors"
	"github.com/meergo/meergo/core/datastore"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/core/postgres"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/transformers"
	"github.com/meergo/meergo/core/transformers/mappings"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
	"github.com/jxskiss/base62"
)

const (
	maxKeysPerConnection = 20 // maximum number of keys per connection.
	maxInt32             = math.MaxInt32
	rawSchemaMaxSize     = 16_777_215 // maximum size in runes for schemas stored in PostgreSQL.
	queryMaxSize         = 16_777_215 // maximum size in runes of a connection query.
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
	core              *Core
	connection        *state.Connection
	store             *datastore.Store
	ID                int           `json:"id"`
	Name              string        `json:"name"`
	Type              ConnectorType `json:"type"`
	Role              Role          `json:"role"`
	Enabled           bool          `json:"enabled"`
	Connector         string        `json:"connector"`
	Strategy          *Strategy     `json:"strategy"`
	SendingMode       *SendingMode  `json:"sendingMode"`
	WebsiteHost       string        `json:"websiteHost"`
	LinkedConnections []int         `json:"linkedConnections,format:emitnull"`
	HasUI             bool          `json:"hasUI"`
	ActionsCount      int           `json:"actionsCount"`
	Health            Health        `json:"health"`

	// Actions is populated only by the (*Workspace).Connection method.
	Actions *[]Action `json:"actions,omitzero"`
}

// Action returns the action with identifier id of the connection.
// It returns an errors.NotFound error if the action does not exist.
func (this *Connection) Action(ctx context.Context, id int) (*Action, error) {
	this.core.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid action identifier", id)
	}
	a, ok := this.connection.Action(id)
	if !ok {
		return nil, errors.NotFound("action %d does not exist", id)
	}
	var action Action
	action.fromState(this.core, this.store, a)
	return &action, nil
}

type ActionSchemas struct {
	In        types.Type              `json:"in"`
	Out       types.Type              `json:"out"`
	Matchings *ActionSchemasMatchings `json:"matchings,omitzero"` // only for destination apps on users.
}

type ActionSchemasMatchings struct {
	Internal types.Type `json:"internal"`
	External types.Type `json:"external"`
}

// dummyGroupsSchema is a dummy "groups" schema, that is used until the groups
// management is properly implemented in Meergo. For now, it serves only as a
// placeholder.
var dummyGroupsSchema = types.Object([]types.Property{
	{Name: "id", Type: types.Int(32)},
})

// ActionSchemas returns the input and the output schemas of an action with the
// given target and event type.
//
// It returns an errors.UnprocessableError error with code EventTypeNotExist, if
// the event type does not exist for the connection.
func (this *Connection) ActionSchemas(ctx context.Context, target Target, eventType string) (*ActionSchemas, error) {

	this.core.mustBeOpen()

	// Validate the target and the event type.
	eventTypeSchema, err := this.validateTargetAndEventType(ctx, target, eventType)
	if err != nil {
		return nil, err
	}

	users := this.connection.Workspace().UserSchema
	groups := dummyGroupsSchema

	c := this.connection

	switch connector := c.Connector(); connector.Type {

	case state.App:
		switch target {
		case Users:
			var err error
			schema, err := this.app().Schema(ctx, state.Users, "")
			if err != nil {
				if _, ok := err.(*connectors.UnavailableError); ok {
					err = errors.Unavailable("an error occurred fetching the schema: %w", err)
				}
				return nil, err
			}
			if c.Role == state.Source {
				return &ActionSchemas{In: schema, Out: users}, nil
			} else {
				sourceSchema, err := this.app().SchemaAsRole(ctx, state.Source, state.Users, "")
				if err != nil {
					if _, ok := err.(*connectors.UnavailableError); ok {
						err = errors.Unavailable("an error occurred fetching the schema: %w", err)
					}
					return nil, err
				}
				actionSchemas := &ActionSchemas{In: users, Out: schema}
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
				if _, ok := err.(*connectors.UnavailableError); ok {
					err = errors.Unavailable("an error occurred fetching the schema: %w", err)
				}
				return nil, err
			}
			if c.Role == state.Source {
				return &ActionSchemas{In: schema, Out: groups}, nil
			} else {
				sourceSchema, err := this.app().SchemaAsRole(ctx, state.Source, state.Groups, "")
				if err != nil {
					if _, ok := err.(*connectors.UnavailableError); ok {
						err = errors.Unavailable("an error occurred fetching the schema: %w", err)
					}
					return nil, err
				}
				actionSchemas := &ActionSchemas{In: groups, Out: schema}
				actionSchemas.Matchings = &ActionSchemasMatchings{
					Internal: onlyForMatching(groups),
					External: onlyForMatching(sourceSchema),
				}
				return actionSchemas, nil
			}
		case Events:
			return &ActionSchemas{In: events.Schema, Out: eventTypeSchema}, nil
		}

	case state.Database:
		switch target {
		case Users:
			if c.Role == state.Source {
				return &ActionSchemas{Out: users}, nil
			} else {
				return &ActionSchemas{In: users}, nil
			}
		case Groups:
			if c.Role == state.Source {
				return &ActionSchemas{Out: groups}, nil
			} else {
				return &ActionSchemas{In: groups}, nil
			}
		}

	case state.FileStorage:
		switch target {
		case Users:
			if c.Role == state.Source {
				return &ActionSchemas{Out: users}, nil
			} else {
				return &ActionSchemas{In: users}, nil
			}
		case Groups:
			if c.Role == state.Source {
				return &ActionSchemas{Out: groups}, nil
			} else {
				return &ActionSchemas{In: groups}, nil
			}
		}

	case state.Mobile, state.Server, state.Stream, state.Website:
		if eventType != "" {
			return nil, errors.NotFound("event type not expected")
		}
		switch target {
		case Users:
			return &ActionSchemas{In: events.Schema, Out: users}, nil
		case Groups:
			return &ActionSchemas{In: events.Schema, Out: groups}, nil
		case Events:
			return &ActionSchemas{In: events.Schema}, nil
		}
		return &ActionSchemas{}, nil

	}

	panic("unreachable code")
}

// ActionType represents an action type.
type ActionType struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Target      Target  `json:"target"`
	EventType   *string `json:"eventType"`
}

// ActionTypes returns the action types for the connection.
//
// Refer to the specifications in the file "core/Actions.md" for more details.
func (this *Connection) ActionTypes(ctx context.Context) ([]ActionType, error) {
	var actionTypes []ActionType
	c := this.connection
	connector := c.Connector()
	targets := connector.Targets
	if targets.Contains(state.Users) {
		switch typ := c.Connector().Type; typ {
		case
			state.App:
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
			state.Database,
			state.FileStorage:
			var name, description string
			if c.Role == state.Source {
				name = "Import users"
				description = "Import the users from " + connector.Name
			} else {
				name = "Export users"
				description = "Export the users to " + connector.Name
			}
			at := ActionType{
				Name:        name,
				Description: description,
				Target:      Users,
			}
			actionTypes = append(actionTypes, at)
		case
			state.Mobile,
			state.Server,
			state.Website:
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
			state.App:
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
			state.Database,
			state.FileStorage:
			var name, description string
			if c.Role == state.Source {
				name = "Import groups"
				description = "Import the groups from " + connector.Name
			} else {
				name = "Export groups"
				description = "Export the groups to " + connector.Name
			}
			at := ActionType{
				Name:        name,
				Description: description,
				Target:      Groups,
			}
			actionTypes = append(actionTypes, at)
		case
			state.Mobile,
			state.Server,
			state.Website:
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
		case state.Mobile, state.Server, state.Website:
			if c.Role == state.Source {
				description := "Import events from the "
				switch typ {
				case state.Mobile:
					description += "mobile app"
				case state.Server:
					description += "server"
				case state.Website:
					description += "website"
				}
				at := ActionType{
					Name:        "Import events",
					Description: description,
					Target:      Events,
				}
				actionTypes = slices.Insert(actionTypes, 0, at)
			}
		case state.App:
			if c.Role == state.Destination {
				eventTypes, err := this.app().EventTypes(ctx)
				if err != nil {
					if _, ok := err.(*connectors.UnavailableError); ok {
						err = errors.Unavailable("an error occurred fetching the schema: %w", err)
					}
					return nil, err
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
	}
	if actionTypes == nil {
		actionTypes = []ActionType{}
	}
	return actionTypes, nil
}

// AddAction adds action to the connection returning the identifier of the
// added action. target is the target of the action and must be supported by the
// connector of the connection.
//
// Refer to the specifications in the file "core/Actions.md" for more details.
//
// It returns an errors.NotFoundError error if the connection does not exist
// anymore, and returns an errors.UnprocessableError error with code
//
//   - ConnectionNotExist, if the connection does not exist.
//   - ConnectorNotExist, if the file connector of the action does not exist.
//   - InvalidUIValues, if the user-entered values are not valid.
//   - TargetExist, if an action already exists for a target for the connection.
//   - UnsupportedLanguage, if the transformation language is not supported.
func (this *Connection) AddAction(ctx context.Context, target Target, eventType string, action ActionToSet) (int, error) {

	this.core.mustBeOpen()

	// Retrieve the file connector, if specified in the action.
	var fileConnector *state.Connector
	if action.Connector != "" {
		fileConnector, _ = this.core.state.Connector(action.Connector)
	}

	// Validate the action.
	v := validationState{}
	v.connection.role = this.connection.Role
	v.connection.connector.typ = this.connection.Connector().Type
	if fileConnector != nil {
		v.connector.typ = fileConnector.Type
		v.connector.hasSheets = fileConnector.HasSheets
		v.connector.hasUI = fileConnector.HasUI
	}
	v.provider = this.core.transformerProvider
	err := validateAction(action, state.Target(target), v)
	if err != nil {
		return 0, err
	}

	connector := this.connection.Connector()

	// Determine the input schema.
	inSchema := action.InSchema
	importUserIdentitiesFromEvents := isImportingUserIdentitiesFromEvents(connector.Type, this.connection.Role, state.Target(target))
	dispatchEventsToApps := isDispatchingEventsToApps(connector.Type, this.connection.Role, state.Target(target))
	importEventsIntoWarehouse := isImportingEventsIntoWarehouse(connector.Type, this.connection.Role, state.Target(target))
	if importUserIdentitiesFromEvents || importEventsIntoWarehouse || dispatchEventsToApps {
		inSchema = events.Schema
	}

	n := state.AddAction{
		Connection:               this.connection.ID,
		Target:                   state.Target(target),
		Name:                     action.Name,
		Enabled:                  action.Enabled,
		EventType:                eventType,
		ScheduleStart:            int16(mathrand.IntN(24 * 60)),
		SchedulePeriod:           60,
		InSchema:                 inSchema,
		OutSchema:                action.OutSchema,
		Transformation:           toStateTransformation(action.Transformation, inSchema, action.OutSchema),
		Query:                    action.Query,
		Connector:                action.Connector,
		Path:                     action.Path,
		Sheet:                    action.Sheet,
		Compression:              state.Compression(action.Compression),
		TableName:                action.TableName,
		TableKeyProperty:         action.TableKeyProperty,
		IdentityProperty:         action.IdentityProperty,
		LastChangeTimeProperty:   action.LastChangeTimeProperty,
		LastChangeTimeFormat:     action.LastChangeTimeFormat,
		FileOrderingPropertyPath: action.FileOrderingPropertyPath,
		ExportMode:               (*state.ExportMode)(action.ExportMode),
		ExportOnDuplicatedUsers:  action.ExportOnDuplicatedUsers,
	}

	// Add the filter to the notification.
	if action.Filter != nil {
		n.Filter, _ = convertFilterToWhere(action.Filter, inSchema).MarshalJSON()
	}

	// Determine the connector name, for file actions.
	var connectorName *string
	if fileConnector != nil {
		name := fileConnector.Name
		connectorName = &name
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
		version, err := this.core.transformerProvider.Create(ctx, name, n.Transformation.Function.Source)
		if err != nil {
			return 0, err
		}
		n.Transformation.Function.Version = version
		function = *n.Transformation.Function
	}

	// Settings.
	if fileConnector != nil && fileConnector.HasUI {
		conf := &connectors.ConnectorConfig{
			Role:   this.connection.Role,
			Region: this.connection.Workspace().PrivacyRegion,
		}
		n.Settings, err = this.core.connectors.UpdatedSettings(ctx, fileConnector, conf, action.UIValues)
		if err != nil {
			switch err.(type) {
			case *meergo.InvalidUIValuesError:
				err = errors.Unprocessable(InvalidUIValues, "%s", err)
			case *connectors.UnavailableError:
				err = errors.Unavailable("%s", err)
			}
			return 0, err
		}
	}

	// Add the action.
	err = this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		switch n.Target {
		case state.Events:
			switch connector.Type {
			case state.Mobile, state.Server, state.Website:
				err = tx.QueryVoid(ctx, "SELECT FROM actions WHERE connection = $1 AND target = 'Events'", n.Connection)
				if err != sql.ErrNoRows {
					if err == nil {
						err = errors.Unprocessable(TargetExist,
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
			"transformation_source, transformation_language, transformation_version, transformation_preserve_json,\n" +
			"transformation_in_properties, transformation_out_properties, query, connector, path, sheet, compression,\n" +
			"settings, table_name, table_key_property, identity_property, last_change_time_property,\n" +
			"last_change_time_format, file_ordering_property_path, export_mode, matching_properties_internal,\n" +
			"matching_properties_external, export_on_duplicated_users)\n" +
			"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,\n" +
			"$17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34)"
		_, err := tx.Exec(ctx, query, n.ID, n.Connection, n.Target, n.EventType,
			n.Name, n.Enabled, n.ScheduleStart, n.SchedulePeriod, rawInSchema, rawOutSchema,
			string(n.Filter), mapping, function.Source, function.Language, function.Version, function.PreserveJSON,
			n.Transformation.InProperties, n.Transformation.OutProperties, n.Query, connectorName, n.Path, n.Sheet,
			n.Compression, string(n.Settings), n.TableName, n.TableKeyProperty, n.IdentityProperty, n.LastChangeTimeProperty,
			n.LastChangeTimeFormat, n.FileOrderingPropertyPath, n.ExportMode, string(matchPropInternal),
			string(matchPropExternal), n.ExportOnDuplicatedUsers)
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

	return n.ID, nil
}

// AppUsers returns the users of an app connection and the cursor to get the
// next users. The returned cursor is empty if there are no other users.
//
// It returns an errors.UnprocessableError error with code SchemaNotAligned if
// the provided schema is not aligned with the app's source schema.
func (this *Connection) AppUsers(ctx context.Context, schema types.Type, cursor string) (json.Value, string, error) {

	this.core.mustBeOpen()

	if this.connection.Connector().Type != state.App {
		return nil, "", errors.BadRequest("connection %d is not an app connection", this.connection.ID)
	}
	if !schema.Valid() {
		return nil, "", errors.BadRequest("schema is not valid")
	}
	var lastChangeTime time.Time
	if cursor != "" {
		var err error
		lastChangeTime, err = deserializeCursor(cursor)
		if err != nil {
			return nil, "", errors.BadRequest("cursor is malformed")
		}
	}

	// Get the users.
	records, err := this.app().Users(ctx, schema, lastChangeTime)
	if err != nil {
		switch err.(type) {
		case *connectors.UnavailableError:
			err = errors.Unavailable("%s", err)
		case *connectors.SchemaError:
			err = errors.Unprocessable(SchemaNotAligned, "schema is not aligned with the app's source schema: %w", err)
		}
		return nil, "", err
	}
	defer records.Close()

	var last connectors.Record
	users := make([]any, 0, 100)

	for user := range records.All(ctx) {
		if user.Err != nil {
			return nil, "", user.Err
		}
		users = append(users, user.Properties)
		if records.Last() {
			last = user
		}
		if len(users) == 100 {
			break
		}
	}
	if err = records.Err(); err != nil {
		if _, ok := err.(*connectors.UnavailableError); ok {
			err = errors.Unavailable("%s", err)
		}
		return nil, "", err
	}

	// Build the cursor.
	cursor, err = serializeCursor(last.LastChangeTime)
	if err != nil {
		return nil, "", err
	}

	marshaledUsers, err := types.Marshal(users, types.Array(schema))
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
	this.core.mustBeOpen()
	c := this.connection
	if c.Connector().Type != state.FileStorage {
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
		switch err.(type) {
		case *meergo.InvalidPathError:
			err = errors.Unprocessable(InvalidPath, "%s", err)
		case *connectors.PlaceholderError:
			err = errors.Unprocessable(InvalidPlaceholder, "%s", err)
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
	this.core.mustBeOpen()
	c := this.connection
	n := state.DeleteConnection{
		ID: c.ID,
	}
	connector := c.Connector()
	workspace := c.Workspace()
	err := this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		if c.Role == state.Source {
			// Mark the connection's actions on Users as deleted.
			_, err := tx.Exec(ctx, "UPDATE workspaces SET actions_to_purge = array_cat(actions_to_purge, (\n"+
				"\tSELECT array_agg(a.id) FROM actions a WHERE a.connection = $1 AND a.target = 'Users'\n"+
				"))\nWHERE id = $2 AND actions_to_purge IS NOT NULL", n.ID, workspace.ID)
			if err != nil {
				return err
			}
		}
		result, err := tx.Exec(ctx, "DELETE FROM connections WHERE id = $1", n.ID)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.NotFound("connection %d does not exist", n.ID)
		}
		role := "Source"
		if c.Role == state.Source {
			role = "Destination"
		}
		// Remove the connection from the linked connections.
		_, err = tx.Exec(ctx, "UPDATE connections\n"+
			"SET linked_connections =\n"+
			"\tCASE\n"+
			"\t\tWHEN array_remove(linked_connections, $1) = '{}' THEN NULL\n"+
			"\t\tELSE array_remove(linked_connections, $1)\n"+
			"\tEND\n"+
			"WHERE workspace = $2 AND role = $3 AND linked_connections IS NOT NULL AND $1 = ANY(linked_connections)",
			n.ID, workspace.ID, role)
		if err != nil {
			return err
		}
		if connector.OAuth != nil {
			// Delete the account of the deleted connection if it has no other connections.
			_, err := tx.Exec(ctx, "DELETE FROM accounts AS a WHERE NOT EXISTS (\n"+
				"\tSELECT FROM connections AS c\n"+
				"\tWHERE a.id = c.account AND c.id <> $1 AND c.account IS NULL\n)", n.ID)
			if err != nil {
				return err
			}
		}
		// Remove the connection as primary source, if any.
		_, err = tx.Exec(ctx, "DELETE FROM user_schema_primary_sources WHERE source = $1", n.ID)
		if err != nil {
			return err
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
//   - InvalidPlaceholder, if the query contains an invalid placeholder.
//   - UnsupportedColumnType, if a column type is not supported.
func (this *Connection) ExecQuery(ctx context.Context, query string, limit int) (json.Value, types.Type, error) {

	this.core.mustBeOpen()

	if !utf8.ValidString(query) {
		return nil, types.Type{}, errors.BadRequest("query is not UTF-8 encoded")
	}
	if containsNUL(query) {
		return nil, types.Type{}, errors.BadRequest("query contains NUL rune")
	}
	if utf8.RuneCountInString(query) > queryMaxSize {
		return nil, types.Type{}, errors.BadRequest("query is longer than 16,777,215 runes")
	}
	if limit < 0 || limit > 100 {
		return nil, types.Type{}, errors.BadRequest("limit %d is not valid", limit)
	}

	c := this.connection
	connector := c.Connector()
	if connector.Type != state.Database {
		return nil, types.Type{}, errors.BadRequest("connection %d is not a database", c.ID)
	}
	if c.Role != state.Source {
		return nil, types.Type{}, errors.BadRequest("database %d is not a source", c.ID)
	}

	// Execute the query.
	database := this.database()
	replacer := func(name string) (string, bool) {
		switch name {
		case "last_change_time":
			v, _ := database.LastChangeTimeCondition(nil)
			return v, true
		case "limit":
			return strconv.Itoa(limit), true
		}
		return "", false
	}
	defer database.Close()
	rows, err := database.Query(ctx, query, replacer)
	if err != nil {
		switch err.(type) {
		case *connectors.PlaceholderError:
			err = errors.Unprocessable(InvalidPlaceholder, "%s", err)
		case *meergo.UnsupportedColumnTypeError:
			err = errors.Unprocessable(UnsupportedColumnType, "%s", err)
		case *connectors.UnavailableError:
			err = errors.Unavailable("%s", err)
		}
		return nil, types.Type{}, err
	}
	defer rows.Close()

	// Scan the rows.
	var results []any
	for rows.Next() {
		row, err := rows.Scan()
		if err != nil {
			if _, ok := err.(*connectors.UnavailableError); ok {
				err = errors.Unavailable("%s", err)
			}
			return nil, types.Type{}, err
		}
		results = append(results, row)
	}
	err = rows.Err()
	if err != nil {
		if _, ok := err.(*connectors.UnavailableError); ok {
			err = errors.Unavailable("%s", err)
		}
		return nil, types.Type{}, err
	}

	schema := types.Object(rows.Columns())

	marshaledRows, err := types.Marshal(results, types.Array(schema))
	if err != nil {
		return nil, types.Type{}, err
	}

	return marshaledRows, schema, nil
}

// An Execution describes an action execution as returned by Executions.
type Execution struct {
	ID        int        `json:"id"`
	Action    int        `json:"action"`
	StartTime time.Time  `json:"startTime"`
	EndTime   *time.Time `json:"endTime"`
	Passed    int        `json:"passed"`
	Failed    int        `json:"failed"`
	Error     string     `json:"error"`
}

// Executions returns the executions of the actions of the connection.
// The connection must be an app, database, file, or stream connection.
func (this *Connection) Executions(ctx context.Context) ([]*Execution, error) {

	this.core.mustBeOpen()

	switch c := this.connection.Connector(); c.Type {
	case state.App, state.Database, state.FileStorage, state.Stream:
	default:
		return nil, errors.BadRequest("connection %d cannot have executions, it's a %s connection",
			this.connection.ID, strings.ToLower(c.Type.String()))
	}

	executions := []*Execution{}
	err := this.core.db.QueryScan(ctx,
		"SELECT e.id, e.action, e.start_time, e.end_time, e.passed, e.failed, e.error_message\n"+
			"FROM actions_executions e\n"+
			"INNER JOIN actions a ON a.id = e.action\n"+
			"WHERE a.connection = $1\n"+
			"ORDER BY id DESC", this.connection.ID, func(rows *postgres.Rows) error {
			var err error
			for rows.Next() {
				var exe Execution
				if err = rows.Scan(&exe.ID, &exe.Action, &exe.StartTime, &exe.EndTime, &exe.Passed, &exe.Failed, &exe.Error); err != nil {
					return err
				}
				executions = append(executions, &exe)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}

	for _, exe := range executions {
		if exe.EndTime == nil {
			exe.Passed = 0
			exe.Failed = 0
		}
	}

	return executions, nil
}

// Identities returns the user identities of the connection, and an estimate of
// their count without applying first and limit.
//
// It returns the user identities in range [first,first+limit] with first >= 0
// and 0 < limit <= 1000.
//
// It returns an errors.UnprocessableError error with code
//
//   - MaintenanceMode, if the data warehouse is in maintenance mode.
//   - WarehouseError, if an error occurred with the data warehouse.
func (this *Connection) Identities(ctx context.Context, first, limit int) ([]UserIdentity, int, error) {
	this.core.mustBeOpen()
	if first < 0 {
		return nil, 0, errors.BadRequest("first %d is not valid", limit)
	}
	if limit < 1 || limit > 1000 {
		return nil, 0, errors.BadRequest("limit %d is not valid", limit)
	}
	coreWs := &Workspace{
		core:      this.core,
		store:     this.store,
		workspace: this.connection.Workspace(),
	}
	where := &state.Where{Logical: state.OpAnd, Conditions: []state.WhereCondition{{
		Property: []string{"__connection__"},
		Operator: state.OpIs,
		Values:   []any{strconv.Itoa(this.connection.ID)},
	}}}
	identities, count, err := coreWs.userIdentities(ctx, where, first, limit)
	if err != nil {
		return nil, 0, err
	}
	if identities == nil {
		identities = []UserIdentity{}
	}
	return identities, count, err
}

// Keys returns the write keys of the connection.
// The connection must be a source mobile, server or website connection.
func (this *Connection) Keys() ([]string, error) {
	this.core.mustBeOpen()
	c := this.connection
	switch c.Connector().Type {
	case state.Mobile, state.Server, state.Website:
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
	this.core.mustBeOpen()
	c := this.connection
	connector := c.Connector()
	switch connector.Type {
	case state.Mobile, state.Server, state.Website:
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
	err = this.core.state.Transaction(ctx, func(tx *state.Tx) error {
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

// LinkConnection links the connection with identifier id to this connection and
// vice versa.
// If the connection to link does not exist, it returns an
// errors.UnprocessableError error with code LinkedConnectionNotExist.
func (this *Connection) LinkConnection(ctx context.Context, id int) error {
	this.core.mustBeOpen()
	// Return If the connections are already linked.
	if slices.Contains(this.connection.LinkedConnections, id) {
		return nil
	}
	// Validate the connection to link.
	c := this.connection.Connector()
	ws := this.connection.Workspace()
	role := this.connection.Role
	err := validateLinkedConnections([]int{id}, c, ws, role)
	if err != nil {
		return err
	}
	n := state.LinkConnection{
		Connections: [2]int{this.connection.ID, id},
	}
	err = this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		const add = "UPDATE connections\n" +
			"SET linked_connections = (SELECT ARRAY(SELECT DISTINCT unnest(array_append(linked_connections, $1)) ORDER BY 1))\n" +
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
			return errors.Unprocessable(LinkedConnectionNotExist, "linked connection %d does not exist", n.Connections[1])
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// Records returns the records and the schema of the file with the given path
// stored in the connection, that must be a file storage connection. path must
// be UTF-8 encoded with a length in range [1, 1024].
//
// fileConnector refers to the file connector to use. If it supports sheets,
// sheet must be a valid sheet name; otherwise, it must be an empty string. A
// valid sheet name is UTF-8 encoded, has a length in the range [1, 31], does
// not start or end with "'", and does not contain any of "*", "/", ":", "?",
// "[", "\", and "]". Sheet names are case-insensitive.
//
// compression indicates if the file is compressed and how. uiValues are the
// user-entered values in JSON format, and limit restricts the number of records
// to return, between 0 and 100.
//
// It returns an errors.UnprocessableError error with code
//
//   - ConnectorNotExist, if the connector does not exist.
//   - InvalidUIValues, if the user-entered values are not valid.
//   - NoColumnsFound, if the file has no columns.
//   - SheetNotExist, if the file does not contain the provided sheet.
//   - UnsupportedColumnType, if a column type is not supported.
func (this *Connection) Records(ctx context.Context, fileConnector string, path, sheet string, compression Compression, uiValues json.Value, limit int) (json.Value, types.Type, error) {

	this.core.mustBeOpen()

	c := this.connection

	// Validate the connection type.
	if c.Connector().Type != state.FileStorage {
		return nil, types.Type{}, errors.BadRequest("connection %d is not a file storage connection", c.ID)
	}
	// Validate the path.
	if path == "" {
		return nil, types.Type{}, errors.BadRequest("path cannot be empty")
	}
	if !utf8.ValidString(path) {
		return nil, types.Type{}, errors.BadRequest("path is not UTF-8 encoded")
	}
	if containsNUL(path) {
		return nil, types.Type{}, errors.BadRequest("path contains NUL rune")
	}
	if n := utf8.RuneCountInString(path); n > 1024 {
		return nil, types.Type{}, errors.BadRequest("path is longer than 1024 runes")
	}

	// Validate the connector.
	file, ok := this.core.state.Connector(fileConnector)
	if !ok {
		return nil, types.Type{}, errors.Unprocessable(ConnectorNotExist, "connector %q does not exist", fileConnector)
	}
	if file.Type != state.File {
		return nil, types.Type{}, errors.BadRequest("connector %s is not a file connector", file.Name)
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

	// Validate the UI values.
	if file.HasUI {
		if uiValues == nil {
			return nil, types.Type{}, errors.BadRequest("UI values must be provided because connector %s has a UI", file.Name)
		}
		if !json.Valid(uiValues) || !uiValues.IsObject() {
			return nil, types.Type{}, errors.BadRequest("UI values are not a valid JSON Object")
		}
	} else if uiValues != nil {
		return nil, types.Type{}, errors.BadRequest("UI values cannot be provided because connector %s has no UI", file.Name)
	}

	// Validate the limit.
	if limit < 0 || limit > 100 {
		return nil, types.Type{}, errors.BadRequest("limit %d is not valid", limit)
	}

	columns, records, err := this.storage().Read(ctx, file, path, sheet, uiValues, state.Compression(compression), limit)
	if err != nil {
		switch err {
		case connectors.ErrNoColumnsFound:
			err = errors.Unprocessable(NoColumnsFound, "file does not have columns")
		case meergo.ErrSheetNotExist:
			err = errors.Unprocessable(SheetNotExist, "file does not contain any sheet named %q", sheet)
		default:
			switch err.(type) {
			case *meergo.InvalidUIValuesError:
				err = errors.Unprocessable(InvalidUIValues, "%s", err)
			case *meergo.UnsupportedColumnTypeError:
				err = errors.Unprocessable(UnsupportedColumnType, "%s", err)
			case *connectors.UnavailableError:
				err = errors.Unavailable("cannot read records: %w", err)
			}
		}
		return nil, types.Type{}, err
	}

	recs := make([]any, len(records))
	for i, r := range records {
		recs[i] = r
	}

	schema := types.Object(columns)
	marshaledRecords, err := types.Marshal(recs, types.Array(schema))
	if err != nil {
		return nil, types.Type{}, err
	}

	return marshaledRecords, schema, nil
}

// UnlinkConnection unlinks the connection with the specified identifier id from
// this connection and vice versa.
// If the connection to unlink does not exist, it returns an
// errors.UnprocessableError with the code LinkedConnectionNotExist.
func (this *Connection) UnlinkConnection(ctx context.Context, id int) error {
	this.core.mustBeOpen()
	// Validate the connection to unlink.
	c := this.connection.Connector()
	ws := this.connection.Workspace()
	role := this.connection.Role
	err := validateLinkedConnections([]int{id}, c, ws, role)
	if err != nil {
		return err
	}
	// Return if this connection is not linked to any other connection.
	if !slices.Contains(this.connection.LinkedConnections, id) {
		return nil
	}
	n := state.UnlinkConnection{
		Connections: [2]int{this.connection.ID, id},
	}
	err = this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		const remove = "UPDATE connections\n" +
			"SET linked_connections =\n" +
			"\tCASE\n" +
			"\t\tWHEN array_remove(linked_connections, $1) = '{}' THEN NULL\n" +
			"\t\tELSE array_remove(linked_connections, $1)\n" +
			"\tEND\n" +
			"WHERE id = $2 AND $1 = ANY(linked_connections)"
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
			return errors.Unprocessable(LinkedConnectionNotExist, "linked connection %d does not exist", n.Connections[1])
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
	this.core.mustBeOpen()
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
	err := this.core.state.Transaction(ctx, func(tx *state.Tx) error {
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
// errors.UnprocessableError error with code CannotDeleteLastKey.
func (this *Connection) RevokeKey(ctx context.Context, key string) error {
	this.core.mustBeOpen()
	if key == "" {
		return errors.BadRequest("key is empty")
	}
	if !isWriteKey(key) {
		return errors.BadRequest("key %q is malformed", key)
	}
	c := this.connection
	connector := c.Connector()
	switch connector.Type {
	case state.Mobile, state.Server, state.Website:
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
	err := this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		var count int
		err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM connections_keys WHERE connection = $1", n.Connection).Scan(&count)
		if err != nil {
			return err
		}
		if count == 1 {
			return errors.Unprocessable(CannotDeleteLastKey, "key cannot be revoked because it's the unique key of the connection")
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

// PreviewSendEvent returns a preview of an event as it would be dispatches to
// an app. The connection must be a destination app connection, and it is
// expected to have an event type with identifier eventType. If there is a
// transformation, outSchema is the output schema of the transformation, and it
// must be a valid.
//
// It returns an errors.UnprocessableError error with code:
//   - EventTypeNotExist, if the event type does not exist for the connection.
//   - SchemaNotAligned, if the output schema is not aligned with the event
//     type's schema.
//   - TransformationFailed if the transformation fails due to an error in the
//     executed function.
//   - UnsupportedLanguage, if the transformation language is not supported.
func (this *Connection) PreviewSendEvent(ctx context.Context, eventType string, event json.Value, transformation DataTransformation, outSchema types.Type) ([]byte, error) {

	this.core.mustBeOpen()

	c := this.connection

	if c.Connector().Type != state.App {
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
	if transformation.Mapping != nil && transformation.Function != nil {
		return nil, errors.BadRequest("mapping and function transformations cannot both be present")
	}

	properties, err := types.Decode[map[string]any](bytes.NewReader(event), events.Schema)
	if err != nil {
		return nil, errors.BadRequest("event is not valid: %s", err)
	}

	var transformedProperties map[string]any

	if transformation.Mapping != nil || transformation.Function != nil {

		if !outSchema.Valid() {
			return nil, errors.BadRequest("a transformation has been provided but out schema is not valid")
		}
		if outSchema.Kind() != types.ObjectKind {
			return nil, errors.BadRequest("out schema is not an Object")
		}

		action := &state.Action{
			InSchema:  events.Schema,
			OutSchema: outSchema,
			Transformation: state.Transformation{
				Mapping: transformation.Mapping,
			},
		}

		// provider is a temporary function transformer provider.
		var provider transformers.Provider

		// Validate the mapping and the transformation.
		switch {
		case transformation.Mapping != nil:
			mapping, err := mappings.New(transformation.Mapping, events.Schema, outSchema, false, nil)
			if err != nil {
				return nil, errors.BadRequest("mapping is not valid: %s", err)
			}
			action.Transformation.InProperties = mapping.InProperties()
			action.Transformation.OutProperties = mapping.OutProperties()
		case transformation.Function != nil:
			if transformation.Function.Source == "" {
				return nil, errors.BadRequest("transformation source is empty")
			}
			switch transformation.Function.Language {
			case "JavaScript":
				if this.core.transformerProvider == nil || !this.core.transformerProvider.SupportLanguage(state.JavaScript) {
					return nil, errors.Unprocessable(UnsupportedLanguage, "JavaScript transformation language  is not supported")
				}
			case "Python":
				if this.core.transformerProvider == nil || !this.core.transformerProvider.SupportLanguage(state.Python) {
					return nil, errors.Unprocessable(UnsupportedLanguage, "Python transformation language is not supported")
				}
			case "":
				return nil, errors.BadRequest("transformation language is empty")
			default:
				return nil, errors.BadRequest("transformation language %q is not valid", transformation.Function.Language)
			}
			action.Transformation.Function = &state.TransformationFunction{
				Source:  transformation.Function.Source,
				Version: "1", // no matter the version, it will be overwritten by the temporary transformation.
			}
			name := "temp-" + uuid.NewString()
			switch transformation.Function.Language {
			case "JavaScript":
				name += ".js"
				action.Transformation.Function.Language = state.JavaScript
			case "Python":
				name += ".py"
				action.Transformation.Function.Language = state.Python
			}
			action.Transformation.Function.PreserveJSON = transformation.Function.PreserveJSON
			action.Transformation.InProperties = types.PropertyNames(action.InSchema)
			action.Transformation.OutProperties = types.PropertyNames(action.OutSchema)
			provider = newTempTransformerProvider(name, transformation.Function.Source, this.core.transformerProvider)
		default:
			return nil, errors.BadRequest("transformation mapping or function is required")
		}

		// Transform the properties.
		transformer, err := transformers.New(action, provider, nil)
		if err != nil {
			return nil, err
		}
		records := []transformers.Record{
			{Purpose: transformers.Create, Properties: properties},
		}
		err = transformer.Transform(ctx, records)
		if err != nil {
			return nil, err
		}
		if err = records[0].Err; err != nil {
			if err, ok := err.(transformers.FunctionExecutionError); ok {
				return nil, errors.Unprocessable(TransformationFailed, err.Error())
			}
			if err, ok := err.(validationError); ok {
				return nil, errors.Unprocessable(TransformationFailed, err.Error())
			}
			return nil, err
		}
		transformedProperties = records[0].Properties

	} else {

		if outSchema.Valid() {
			return nil, errors.BadRequest("output schema is a valid schema, but no transformation has been provided")
		}

	}

	// Convert data into a meergo.Event value.
	ev := events.NewConnectorEvent(properties)

	req, err := this.app().EventRequest(ctx, ev, eventType, outSchema, transformedProperties, true)
	if err != nil {
		if err == meergo.ErrEventTypeNotExist {
			err = errors.Unprocessable(EventTypeNotExist, "connection %d does not have event type %q", c.ID, eventType)
		} else {
			switch err.(type) {
			case *connectors.SchemaError:
				err = errors.Unprocessable(SchemaNotAligned, "output schema is not compatible with the event type's schema: %w", err)
			case *connectors.UnavailableError:
				err = errors.Unavailable("connector returned an error preparing the preview: %w", err)
			}
		}
		return nil, err
	}

	// Construct the preview.
	var b json.Buffer
	b.WriteString(req.Method)
	b.WriteString(" ")
	b.WriteString(req.URL)
	b.WriteByte('\n')
	err = req.Header.Write(&b)
	if err != nil {
		return nil, err
	}
	if req.Body != nil {
		b.WriteByte('\n')
		ct := req.Header.Get("Content-Type")
		switch ct {
		case "application/json":
			err = b.EncodeIndent(req.Body, "", "\t")
			if err != nil {
				return nil, err
			}
		case "application/x-ndjson":
			b.Write(req.Body)
		default:
			_, _ = fmt.Fprintf(&b, "[%d bytes body]", len(req.Body))
		}
	}

	return b.Bytes(), nil
}

// Set sets the connection.
func (this *Connection) Set(ctx context.Context, connection ConnectionToSet) error {

	this.core.mustBeOpen()

	if connection.Name == "" || containsNUL(connection.Name) || utf8.RuneCountInString(connection.Name) > 100 {
		return errors.BadRequest("name %q is not valid", connection.Name)
	}
	if s := connection.Strategy; s != nil && !isValidStrategy(*s) {
		return errors.BadRequest("strategy %q is not valid", *s)
	}
	if sm := connection.SendingMode; sm != nil && !isValidSendingMode(*sm) {
		return errors.BadRequest("sending mode %q is not valid", *sm)
	}
	if host := connection.WebsiteHost; host != "" {
		if _, _, err := parseWebsiteHost(host); err != nil {
			return errors.BadRequest("website host %q is not valid", host)
		}
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
		case state.Mobile, state.Website:
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
	if n.WebsiteHost != "" && c.Type != state.Website {
		return errors.BadRequest("connector %s cannot have a website host, it's a %s",
			c.Name, strings.ToLower(c.Type.String()))
	}

	err := this.core.state.Transaction(ctx, func(tx *state.Tx) error {
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
// values are the user-entered values in JSON format.
//
// It returns an errors.UnprocessableError error with code:
//
//   - EventNotExist, if the event does not exist.
//   - InvalidUIValues, if the user-entered values are not valid.
func (this *Connection) ServeUI(ctx context.Context, event string, values json.Value) (json.Value, error) {
	this.core.mustBeOpen()
	// TODO: check and delete alternative fieldsets keys that have 'null' value
	// before saving to database
	connector := this.connection.Connector()
	if !connector.HasUI {
		return nil, errors.BadRequest("connector %s does not have a UI", connector.Name)
	}
	ui, err := this.core.connectors.ServeConnectionUI(ctx, this.connection, event, values)
	if err != nil {
		if err == meergo.ErrUIEventNotExist {
			err = errors.Unprocessable(EventNotExist, "UI event %q does not exist for connector %s", event, connector.Name)
		} else {
			switch err.(type) {
			case *meergo.InvalidUIValuesError:
				err = errors.Unprocessable(InvalidUIValues, "%s", err)
			case *connectors.UnavailableError:
				err = errors.Unavailable("%s", err)
			}
		}
		return nil, err
	}
	return ui, nil
}

// Sheets returns the sheets of the file at the given path for the connection,
// that must be a file connection. path must be UTF-8 encoded with a length in
// range [1, 1024].
//
// fileConnector refers to the file connector with multi sheets to use.
// compression indicates if the file is compressed and how. uiValues are the
// user-entered values.
//
// It returns an errors.UnprocessableError error with code
//   - ConnectorNotExist, if the file connector does not exist.
//   - InvalidUIValues, if the UI values are not valid.
func (this *Connection) Sheets(ctx context.Context, fileConnector string, path string, uiValues json.Value, compression Compression) ([]string, error) {

	this.core.mustBeOpen()

	c := this.connection

	if c.Connector().Type != state.FileStorage {
		return nil, errors.BadRequest("connection %d is not a file storage", c.ID)
	}
	if path == "" {
		return nil, errors.BadRequest("path is empty")
	}
	if containsNUL(path) {
		return nil, errors.BadRequest("path contains NUL rune")
	}
	if !utf8.ValidString(path) {
		return nil, errors.BadRequest("path is not UTF-8 encoded")
	}

	// Validate the file connector.
	file, ok := this.core.state.Connector(fileConnector)
	if !ok {
		return nil, errors.Unprocessable(ConnectorNotExist, "connector %q does not exist", fileConnector)
	}
	if file.Type != state.File {
		return nil, errors.BadRequest("connector %s is not a file connector", file.Name)
	}

	// Validate the UI values.
	if file.HasUI {
		if uiValues == nil {
			return nil, errors.BadRequest("UI values must be provided because connector %s has a UI", file.Name)
		}
		if !json.Valid(uiValues) || !uiValues.IsObject() {
			return nil, errors.BadRequest("UI values are not a valid JSON Object")
		}
	} else if uiValues != nil {
		return nil, errors.BadRequest("UI values cannot be provided because connector %s has no UI", file.Name)
	}

	if !file.HasSheets {
		return nil, errors.BadRequest("connector %s does not have sheets", file.Name)
	}

	sheets, err := this.storage().Sheets(ctx, file, path, uiValues, state.Compression(compression))
	if err != nil {
		switch err.(type) {
		case *meergo.InvalidUIValuesError:
			err = errors.Unprocessable(InvalidUIValues, "%s", err)
		case *connectors.UnavailableError:
			err = errors.Unavailable("cannot read the file: %w", err)
		}
		return nil, err
	}

	return sheets, nil
}

// TableSchema returns the schema of the given table for the connection.
// connection must be a destination database connection, and table must be UTF-8
// encoded with a length in range [1, 1024].
//
// If the table contains a column with an unsupported type, it returns an
// errors.UnprocessableError error.
func (this *Connection) TableSchema(ctx context.Context, table string) (types.Type, error) {
	this.core.mustBeOpen()
	c := this.connection
	connector := c.Connector()
	if connector.Type != state.Database {
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
	schema, err := database.Schema(ctx, table)
	if err != nil {
		switch err.(type) {
		case *meergo.UnsupportedColumnTypeError:
			err = errors.Unprocessable(UnsupportedColumnType, "%s", err)
		case *connectors.UnavailableError:
			err = errors.Unavailable("an error occurred fetching the columns: %w", err)
		}
	}
	return schema, err
}

// app returns the app of the connection.
func (this *Connection) app() *connectors.App {
	return this.core.connectors.App(this.connection)
}

// database returns the database of the connection.
// The caller must call the database's Close method when the database is no
// longer needed.
func (this *Connection) database() *connectors.Database {
	return this.core.connectors.Database(this.connection)
}

// storage returns the storage of the connection.
func (this *Connection) storage() *connectors.FileStorage {
	return this.core.connectors.FileStorage(this.connection)
}

// validateTargetAndEventType validates a target and an event type and, if the
// event type is not empty, it returns its schema.
//
// It returns an errors.BadRequestError error if target or eventType is not
// valid, or the connection does not support them, and returns an
// errors.UnprocessableError error with code EventTypeNotExist, if the
// connection does not have the event type.
func (this *Connection) validateTargetAndEventType(ctx context.Context, target Target, eventType string) (types.Type, error) {
	// Perform a formal validation.
	if target != Users && target != Groups && target != Events {
		return types.Type{}, errors.BadRequest("target %d is not valid", int(target))
	}
	if eventType != "" && target != Events {
		return types.Type{}, errors.BadRequest("event type cannot be used with %s target", target)
	}
	// Perform a validation based on the connection's type and role.
	// (Refer to the specifications in the file "core/Actions.md" for more
	// details)
	c := this.connection
	connector := c.Connector()
	var supported bool
	switch connector.Type {
	case state.App:
		supported = c.Role == state.Destination || target != Events
	case state.Database, state.FileStorage:
		supported = target != Events
	case state.Mobile, state.Server, state.Website:
		supported = c.Role == state.Source
	case state.Stream:
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
			if err == meergo.ErrEventTypeNotExist {
				err = errors.Unprocessable(EventTypeNotExist, "connection %d does not have event type %q", c.ID, eventType)
			} else if _, ok := err.(*connectors.UnavailableError); ok {
				err = errors.Unavailable("an error occurred fetching the schema: %w", err)
			}
			return types.Type{}, err
		}
		return schema, nil
	}
	return types.Type{}, nil
}

// parseWebsiteHost parses a website host from the format "host:port" and
// returns the host and the port. The host cannot be empty, cannot contain the
// NUL rune and cannot be longer than 255 characters. If a port is present, it
// must be in the range [1,65535]. If no port is present, it defaults to
// returning 443 as the port.
func parseWebsiteHost(s string) (string, int, error) {
	h, p, found := strings.Cut(s, ":")
	if h == "" || len(s) > 255 || containsNUL(h) {
		return "", 0, errors.New("website host is not valid")
	}
	port := 443
	if found {
		if port, _ = strconv.Atoi(p); port < 1 || port > 65535 {
			return "", 0, errors.New("website host is not valid")
		}
	}
	return h, port, nil
}

// validateLinkedConnections checks whether the provided connections can be
// linked to, or unlinked from, a connection with the specified connector,
// workspace, and role.
//
// If the connections cannot be linked or unlinked, it returns an
// errors.BadRequestError. If any connection does not exist, it returns an
// errors.UnprocessableError with the code LinkedConnectionNotExist.
func validateLinkedConnections(connections []int, c *state.Connector, ws *state.Workspace, role state.Role) error {
	if connections == nil {
		return nil
	}
	if len(connections) == 0 {
		return errors.BadRequest("event connections cannot be empty")
	}
	if !c.Targets.Contains(state.Events) {
		return errors.BadRequest("connector %s does not support event connections", c.Name)
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
			return errors.Unprocessable(LinkedConnectionNotExist, "linked connection %d does not exist", id)
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
		return err
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("json: cannot scan a %T value into an api.Role value", v)
	}
	var r Role
	switch s {
	case "Source":
		r = Source
	case "Destination":
		r = Destination
	default:
		return fmt.Errorf("json: invalid core.Role: %s", s)
	}
	*role = r
	return nil
}

// ConnectionToSet represents a connection to set in a workspace, by adding a
// new connection (using the method Workspace.AddConnection) or updating an
// existing one (using the method Connection.Set).
type ConnectionToSet struct {

	// Name is the name of the connection. It cannot be longer than 100 runes.
	// If empty, the connection name will be the name of its connector.
	Name string `json:"name"`

	// Enable reports whether the connection is enabled or disabled when added.
	Enabled bool `json:"enabled"`

	// Strategy is the strategy that determines how to merge anonymous and
	// non-anonymous users. It can only be provided for source Mobile and Website
	// connections.
	Strategy *Strategy `json:"strategy"`

	// SendingMode is the mode used for sending events. It can only be provided for
	// destination app connections that support it. In this case, it must be one of
	// the sending modes supported by the app.
	SendingMode *SendingMode `json:"sendingMode"`

	// WebsiteHost is the host, in the form "host:port", of a website
	// connection. It must be empty if the connection is not a website. It
	// cannot be longer than 261 runes.
	WebsiteHost string `json:"websiteHost"`
}

// isMetaProperty reports whether the given property name refers to a property
// considered a meta property by a data warehouse.
func isMetaProperty(name string) bool {
	return len(name) >= 5 && strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__")
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

// deserializeCursor deserializes a cursor passed to the API.
func deserializeCursor(cursor string) (time.Time, error) {
	data, err := hex.DecodeString(cursor)
	if err != nil {
		return time.Time{}, err
	}
	var c time.Time
	err = json.Unmarshal(data, &c)
	if err != nil {
		return time.Time{}, err
	}
	// TODO(marco): validate the cursor's fields.
	return c, nil
}

// serializeCursor serializes a cursor to be returned by the API.
func serializeCursor(cursor time.Time) (string, error) {
	b, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// tempTransformerProvider is a function transformer provider that creates a
// function at each call and deletes it after the call returns. Any call to a
// method that is not CallFunction panics.
type tempTransformerProvider struct {
	name     string                // function name.
	source   string                // source code.
	provider transformers.Provider // underlying transformer provider.
}

func newTempTransformerProvider(name, source string, provider transformers.Provider) *tempTransformerProvider {
	return &tempTransformerProvider{name, source, provider}
}

func (tp *tempTransformerProvider) Call(ctx context.Context, _, _ string, inSchema, outSchema types.Type, preserveJSON bool, records []transformers.Record) error {
	version, err := tp.provider.Create(ctx, tp.name, tp.source)
	if err != nil {
		return nil
	}
	defer func() {
		go func() {
			err := tp.provider.Delete(context.Background(), tp.name)
			if err != nil {
				slog.Warn("cannot delete transformation function", "name", tp.name, "err", err)
			}
		}()
	}()
	return tp.provider.Call(ctx, tp.name, version, inSchema, outSchema, preserveJSON, records)
}

func (tp *tempTransformerProvider) Close(_ context.Context) error { panic("not supported") }
func (tp *tempTransformerProvider) Create(_ context.Context, _, _ string) (string, error) {
	panic("not supported")
}
func (tp *tempTransformerProvider) Delete(_ context.Context, _ string) error {
	panic("not supported")
}
func (tp *tempTransformerProvider) SupportLanguage(_ state.Language) bool {
	panic("not supported")
}
func (tp *tempTransformerProvider) Update(_ context.Context, _, _ string) (string, error) {
	panic("not supported")
}
