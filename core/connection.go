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
	"github.com/meergo/meergo/core/util"
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

// Connection represents a connection.
type Connection struct {
	core              *Core
	connection        *state.Connection
	store             *datastore.Store
	ID                int           `json:"id"`
	Name              string        `json:"name"`
	Connector         string        `json:"connector"`
	ConnectorType     ConnectorType `json:"connectorType"`
	Role              Role          `json:"role"`
	Strategy          *Strategy     `json:"strategy"`
	SendingMode       *SendingMode  `json:"sendingMode"`
	WebsiteHost       string        `json:"websiteHost"`
	LinkedConnections []int         `json:"linkedConnections,format:emitnull"`
	ActionsCount      int           `json:"actionsCount"`
	Health            Health        `json:"-"` // See issue https://github.com/meergo/meergo/issues/1255.

	// Actions is populated only by the (*Workspace).Connection method.
	Actions *[]Action `json:"actions,omitzero"`

	// EventTypes is populated only by the (*Workspace).Connection method.
	EventTypes *[]EventType `json:"eventTypes,omitzero"`
}

// EventType represents an event type of a destination app connection.
type EventType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Strategy represents a strategy.
// Can be "Conversion", "Fusion", "Isolation", and "Preservation".
type Strategy string

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

// ActionType represents an action type.
type ActionType struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Target      Target  `json:"target"`
	EventType   *string `json:"eventType"`
}

// ActionSchemas returns the input and the output schemas of an action with the
// given target and event type.
//
// TODO(Gianluca): this method is deprecated. See the issue
// https://github.com/meergo/meergo/issues/1266.
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
			// Retrieve the app's source or target schema, depending on the
			// Connection's role.
			schema, err := this.app().Schema(ctx, state.Users, "")
			if err != nil {
				if _, ok := err.(*connectors.UnavailableError); ok {
					err = errors.Unavailable("an error occurred fetching the schema: %w", err)
				}
				return nil, err
			}
			if c.Role == state.Source {
				// Source/App/Users.
				return &ActionSchemas{In: schema, Out: users}, nil
			} else {
				// Destination/App/Users.
				//
				// The app's destination schema is already available here, but
				// we need to get the source one too because it's needed for the
				// matching properties.
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
				// Source/App/Groups.
				return &ActionSchemas{In: schema, Out: groups}, nil
			} else {
				// Destination/App/Groups.
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
				// Source/Database/Users.
				//
				// The input schema is not set here because it is retrieved via
				// a separate API call, since it depends on the query, which in
				// the UI case is entered interactively by the user.
				return &ActionSchemas{Out: users}, nil
			} else {
				// Destination/Database/Users.
				//
				// The output schema depends on the table chosen for export, and
				// must be retrieved separately.
				return &ActionSchemas{In: users}, nil
			}
		case Groups:
			if c.Role == state.Source {
				// Source/Database/Groups.
				return &ActionSchemas{Out: groups}, nil
			} else {
				// Destination/Database/Groups.
				return &ActionSchemas{In: groups}, nil
			}
		}

	case state.FileStorage:
		switch target {
		case Users:
			if c.Role == state.Source {
				// Source/FileStorage/Source.
				//
				// The input schema is not set here because it is retrieved via
				// a separate API call, since it depends on the file, which in
				// the UI case is entered interactively by the user.
				return &ActionSchemas{Out: users}, nil
			} else {
				// Destination/FileStorage/Source.
				return &ActionSchemas{In: users}, nil
			}
		case Groups:
			if c.Role == state.Source {
				// Source/FileStorage/Groups.
				return &ActionSchemas{Out: groups}, nil
			} else {
				// Destination/FileStorage/Groups.
				return &ActionSchemas{In: groups}, nil
			}
		}

	case state.Mobile, state.Server, state.Stream, state.Website:
		if eventType != "" {
			return nil, errors.NotFound("event type not expected")
		}
		// TODO(Gianluca): regarding Stream connectors, see the issue
		// https://github.com/meergo/meergo/issues/1264.
		switch target {
		case Users:
			// Source/Mobile/Users.
			// Source/Server/Users.
			// Source/Website/Users.
			return &ActionSchemas{In: events.Schema, Out: users}, nil
		case Groups:
			// Source/Mobile/Groups.
			// Source/Server/Groups.
			// Source/Website/Groups.
			return &ActionSchemas{In: events.Schema, Out: groups}, nil
		case Events:
			// Source/Mobile/Events.
			// Source/Server/Events.
			// Source/Website/Events.
			return &ActionSchemas{In: events.Schema}, nil
		}
		return &ActionSchemas{}, nil

	}

	panic("unreachable code")
}

// ActionTypes returns the action types for the connection.
//
// TODO(Gianluca): this method is deprecated. See the issue
// https://github.com/meergo/meergo/issues/1265.
//
// Refer to the specifications in the file "core/Actions.csv" for more details.
func (this *Connection) ActionTypes(ctx context.Context) ([]ActionType, error) {
	var actionTypes []ActionType
	c := this.connection
	connector := c.Connector()
	var targets state.ConnectorTargets
	if c.Role == state.Source {
		targets = connector.SourceTargets
	} else {
		targets = connector.DestinationTargets
	}
	if targets.Contains(state.Users) {
		switch typ := c.Connector().Type; typ {
		case
			state.App:
			var name, description string
			if c.Role == state.Source {
				// Source/App/Users.
				name = "Import " + connector.TermForUsers
				description = "Import the " + connector.TermForUsers
				if connector.TermForUsers != "users" {
					description += " as users"
				}
				description += " from " + connector.Name
			} else {
				// Destination/App/Users.
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
				// Source/FileStorage/Users.
				// Source/Database/Users.
				name = "Import users"
				description = "Import the users from " + connector.Name
			} else {
				// Destination/FileStorage/Users.
				// Destination/Database/Users.
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
				// Source/Mobile/Users.
				// Source/Server/Users.
				// Source/Website/Users.
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
				// Source/App/Groups.
				name = "Import " + connector.TermForGroups
				description = "Import the " + connector.TermForGroups
				if connector.TermForGroups != "groups" {
					description += " as groups"
				}
				description += " from " + connector.Name
			} else {
				// Destination/App/Groups.
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
				// Source/FileStorage/Groups.
				// Source/Database/Groups.
				name = "Import groups"
				description = "Import the groups from " + connector.Name
			} else {
				// Destination/FileStorage/Groups.
				// Destination/Database/Groups.
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
				// Source/Mobile/Groups.
				// Source/Server/Groups.
				// Source/Website/Groups.
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
				// Source/Mobile/Events.
				// Source/Server/Events.
				// Source/Website/Events.
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
				// Destination/App/Events.
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

// AppEventSchema returns the schema of the provided event type of the
// connection. If the event type does not have a schema, it returns an invalid
// schema. The connection must be a destination app connection that supports
// events.
//
// It returns an errors.NotFoundError error if the event type does not exist.
func (this *Connection) AppEventSchema(ctx context.Context, eventType string) (types.Type, error) {
	this.core.mustBeOpen()
	if eventType == "" {
		return types.Type{}, errors.BadRequest("event type is empty")
	}
	c := this.connection
	connector := c.Connector()
	if connector.Type != state.App {
		return types.Type{}, errors.BadRequest("connection %d is not an app", c.ID)
	}
	if c.Role != state.Destination {
		return types.Type{}, errors.BadRequest("connection %d is not a destination", c.ID)
	}
	if !connector.DestinationTargets.Contains(state.Events) {
		return types.Type{}, errors.BadRequest("connection %d does not support events", c.ID)
	}
	schema, err := this.app().SchemaAsRole(ctx, state.Destination, state.Events, eventType)
	if err != nil {
		if _, ok := err.(*connectors.UnavailableError); ok {
			err = errors.Unavailable("an error occurred fetching the schema: %w", err)
		}
		if err == meergo.ErrEventTypeNotExist {
			err = errors.NotFound("event type %q does not exist", eventType)
		}
		return types.Type{}, err
	}
	return schema, nil
}

// AppGroupSchemas returns the group schemas for the connection. The connection
// must be an app connection that supports groups. For a source, it returns only
// the source schema. For a destination, it returns both the source and
// destination schemas.
//
// TODO(Gianluca): this method is currently unused, and it has been kept for the
// future, when we will re-expose the endpoint to retrieve group schemas. See
// the issue https://github.com/meergo/meergo/issues/895.
func (this *Connection) AppGroupSchemas(ctx context.Context) (src, dst types.Type, err error) {
	this.core.mustBeOpen()
	return this.appSchemas(ctx, state.Groups)
}

// AppUserSchemas returns the user schemas for the connection. The connection
// must be an app connection that supports users. For a source, it returns only
// the source schema. For a destination, it returns both the source and
// destination schemas.
func (this *Connection) AppUserSchemas(ctx context.Context) (src, dst types.Type, err error) {
	this.core.mustBeOpen()
	return this.appSchemas(ctx, state.Users)
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
	if !this.connection.Connector().SourceTargets.Contains(state.Users) {
		return nil, "", errors.BadRequest("connection %d does not support reading of users", this.connection.ID)
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
// cannot be longer than MaxFilePathSize runes, and must be UTF-8 encoded.
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
	if err := util.ValidateStringField("path", path, MaxFilePathSize); err != nil {
		return "", errors.BadRequest("%s", err)
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

// CreateAction creates an action for the connection returning the identifier of
// the created action. target is the target of the action and must be supported
// by the connector of the connection.
//
// Refer to the specifications in the file "core/Actions.csv" for more details.
//
// It returns an errors.NotFoundError error if the connection does not exist
// anymore, and returns an errors.UnprocessableError error with code
//
//   - ConnectionNotExist, if the connection does not exist.
//   - EventTypeNotExists, if the event type does not exist for the connection.
//   - FormatNotExist, if the format of the action does not exist.
//   - InvalidSettings, if the settings are not valid.
//   - TargetExist, if an action already exists for a target for the connection.
//   - UnsupportedLanguage, if the transformation language is not supported.
func (this *Connection) CreateAction(ctx context.Context, target Target, eventType string, action ActionToSet) (int, error) {

	this.core.mustBeOpen()

	// Retrieve the format, if specified in the action.
	var format *state.Connector
	if action.Format != "" {
		format, _ = this.core.state.Connector(action.Format)
	}

	c := this.connection
	connector := c.Connector()

	// Validate the target.
	if target != Users && target != Groups && target != Events {
		return 0, errors.BadRequest("target %d is not valid", int(target))
	}
	var connectorsTargets state.ConnectorTargets
	if c.Role == state.Source {
		connectorsTargets = connector.SourceTargets
	} else {
		connectorsTargets = connector.DestinationTargets
	}
	if !connectorsTargets.Contains(state.Target(target)) {
		role := strings.ToLower(c.Role.String())
		typ := connector.Type.String()
		return 0, errors.BadRequest("action with target '%s' not allowed for %s %s connections", target, role, typ)
	}

	// Validate the event type.
	requiresEventType := c.Role == state.Destination && connector.Type == state.App && target == Events
	if requiresEventType && eventType == "" {
		return 0, errors.BadRequest("eventType is required for actions that send events to apps")
	}
	if !requiresEventType && eventType != "" {
		role := strings.ToLower(c.Role.String())
		typ := strings.ToLower(connector.Type.String())
		return 0, errors.BadRequest("actions with target '%s' on %s %s connections cannot specify an event type", target, role, typ)
	}
	if eventType != "" {
		if err := util.ValidateStringField("eventType", eventType, 100); err != nil {
			return 0, errors.BadRequest("%s", err)
		}
	}

	// Validate the action.
	v := validationState{}
	v.target = state.Target(target)
	v.connection.role = c.Role
	v.connection.connector.typ = connector.Type
	if format != nil {
		v.format.typ = format.Type
		if c.Role == state.Source {
			v.format.targets = format.SourceTargets
		} else {
			v.format.targets = format.DestinationTargets
		}
		v.format.hasSheets = format.HasSheets
		v.format.hasSettings = c.Role == state.Source && format.HasSourceSettings || c.Role == state.Destination && format.HasDestinationSettings
	}
	v.provider = this.core.transformerProvider
	err := validateActionToSet(action, v)
	if err != nil {
		return 0, err
	}

	// Determine the input schema.
	inSchema := action.InSchema
	importUserIdentitiesFromEvents := isImportingUserIdentitiesFromEvents(connector.Type, c.Role, state.Target(target))
	dispatchEventsToApps := isDispatchingEventsToApps(connector.Type, c.Role, state.Target(target))
	importEventsIntoWarehouse := isImportingEventsIntoWarehouse(connector.Type, c.Role, state.Target(target))
	if importUserIdentitiesFromEvents || importEventsIntoWarehouse || dispatchEventsToApps {
		inSchema = events.Schema
	}

	n := state.CreateAction{
		Connection:             c.ID,
		Target:                 state.Target(target),
		Name:                   action.Name,
		Enabled:                action.Enabled,
		EventType:              eventType,
		InSchema:               inSchema,
		OutSchema:              action.OutSchema,
		Transformation:         toStateTransformation(action.Transformation, inSchema, action.OutSchema),
		Query:                  action.Query,
		Format:                 action.Format,
		Path:                   action.Path,
		Sheet:                  action.Sheet,
		Compression:            state.Compression(action.Compression),
		OrderBy:                action.OrderBy,
		ExportMode:             state.ExportMode(action.ExportMode),
		Matching:               state.Matching(action.Matching),
		ExportOnDuplicates:     action.ExportOnDuplicates,
		TableName:              action.TableName,
		TableKey:               action.TableKey,
		IdentityProperty:       action.IdentityProperty,
		LastChangeTimeProperty: action.LastChangeTimeProperty,
		LastChangeTimeFormat:   action.LastChangeTimeFormat,
	}

	// Set the scheduler.
	if n.Target == state.Users || n.Target == state.Groups {
		n.ScheduleStart = int16(mathrand.IntN(24 * 60))
		n.SchedulePeriod = 60
	}

	// Add the filter to the notification.
	if action.Filter != nil {
		n.Filter, _ = convertFilterToWhere(action.Filter, inSchema).MarshalJSON()
	}

	// Determine the connector name, for file actions.
	var formatName *string
	if format != nil {
		name := format.Name
		formatName = &name
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
	if tr := action.Transformation; tr != nil && tr.Mapping != nil {
		mapping, err = json.Marshal(tr.Mapping)
		if err != nil {
			return 0, err
		}
	}

	var function state.TransformationFunction
	if n.Transformation.Function != nil {
		name := util.TransformationFunctionName(n.ID, n.Transformation.Function.Language)
		version, err := this.core.transformerProvider.Create(ctx, name, n.Transformation.Function.Source)
		if err != nil {
			return 0, err
		}
		n.Transformation.Function.Version = version
		function = *n.Transformation.Function
	}

	// Format settings.
	if format != nil && action.FormatSettings != nil {
		conf := &connectors.ConnectorConfig{
			Role: this.connection.Role,
		}
		n.FormatSettings, err = this.core.connectors.UpdatedSettings(ctx, format, conf, action.FormatSettings)
		if err != nil {
			switch err.(type) {
			case *meergo.InvalidSettingsError:
				err = errors.Unprocessable(InvalidSettings, "%s", err)
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
		query := "INSERT INTO actions (id, connection, target, event_type, name, enabled,\n" +
			"schedule_start, schedule_period, in_schema, out_schema, filter, transformation_mapping,\n" +
			"transformation_source, transformation_language, transformation_version, transformation_preserve_json,\n" +
			"transformation_in_paths, transformation_out_paths, query, format, path, sheet, compression, order_by,\n" +
			"format_settings, export_mode, matching_in, matching_out, allow_duplicates, table_name, table_key,\n" +
			"identity_property, last_change_time_property, last_change_time_format)\n" +
			"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21,\n" +
			"$22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34)"
		_, err := tx.Exec(ctx, query, n.ID, n.Connection, n.Target, n.EventType,
			n.Name, n.Enabled, n.ScheduleStart, n.SchedulePeriod, rawInSchema, rawOutSchema,
			string(n.Filter), mapping, function.Source, function.Language, function.Version, function.PreserveJSON,
			n.Transformation.InPaths, n.Transformation.OutPaths, n.Query, formatName, n.Path, n.Sheet,
			n.Compression, n.OrderBy, string(n.FormatSettings), n.ExportMode, n.Matching.In, n.Matching.Out, n.ExportOnDuplicates,
			n.TableName, n.TableKey, n.IdentityProperty, n.LastChangeTimeProperty, n.LastChangeTimeFormat)
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

// CreateEventWriteKey creates a new event write key for the connection.
// The connection must be a source mobile, server or website connection.
//
// If the connection does not exist, it returns an errors.NotFoundError error.
// If the connection has already too many keys, it returns an
// errors.UnprocessableError error with code TooManyEventWriteKeys.
func (this *Connection) CreateEventWriteKey(ctx context.Context) (string, error) {
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
	key, err := generateEventWriteKey()
	if err != nil {
		return "", err
	}
	n := state.CreateEventWriteKey{
		Connection:   c.ID,
		Key:          key,
		CreationTime: time.Now().UTC(),
	}
	err = this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		var count int
		err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM event_write_keys WHERE connection = $1", n.Connection).Scan(&count)
		if err != nil {
			return err
		}
		if count == maxKeysPerConnection {
			return errors.Unprocessable(TooManyEventWriteKeys, "connection %d has already %d event write keys", n.Connection, maxKeysPerConnection)
		}
		_, err = tx.Exec(ctx, "INSERT INTO event_write_keys (connection, key, creation_time) VALUES ($1, $2, $3)",
			n.Connection, n.Key, n.CreationTime)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				if postgres.ErrConstraintName(err) == "event_write_keys_connection_fkey" {
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
	return key, nil
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

// DeleteEventWriteKey deletes the given event write key of the connection.
// key cannot be empty and cannot be the only key for the connection.
// The connection must be a source mobile, server or website connection.
//
// If the key does not exist, it returns an errors.NotFoundError error.
// If the key is the only key for the connection, it returns an
// errors.UnprocessableError error with code SingleEventWriteKey.
func (this *Connection) DeleteEventWriteKey(ctx context.Context, key string) error {
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
	n := state.DeleteEventWriteKey{
		Connection: c.ID,
		Key:        key,
	}
	err := this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		var count int
		err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM event_write_keys WHERE connection = $1", n.Connection).Scan(&count)
		if err != nil {
			return err
		}
		if count == 1 {
			return errors.Unprocessable(SingleEventWriteKey, "key cannot be deleted as it is the connection’s only key")
		}
		result, err := tx.Exec(ctx, "DELETE FROM event_write_keys WHERE connection = $1 AND key = $2", n.Connection, n.Key)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.NotFound("key %q does not exist", key)
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

	if err := util.ValidateStringField("query", query, queryMaxSize); err != nil {
		return nil, types.Type{}, errors.BadRequest("%s", err)
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

// File returns the records and schema of the file located at the specified path
// within the connection. The connection must be a file storage connection which
// supports read operations. path must be UTF-8 encoded with a length in range
// [1, MaxFilePathSize].
//
// format specifies the file format and must match the name of a file connector
// which supports reading of records. If the connector supports sheets, sheet
// must be a valid sheet name; otherwise, it must be an empty string. A valid
// sheet name is UTF-8 encoded, has a length in the range [1, 31], does not
// start or end with "'", and does not contain any of "*", "/", ":", "?", "[",
// "\", and "]". Sheet names are case-insensitive.
//
// compression indicates if the file is compressed and how. settings are the
// format settings, and limit restricts the number of records to return, between
// 0 and 100.
//
// It returns an errors.UnprocessableError error with code
//
//   - FormatNotExist, if the format does not exist.
//   - InvalidSettings, if the settings are not valid.
//   - NoColumnsFound, if the file has no columns.
//   - SheetNotExist, if the file does not contain the provided sheet.
//   - UnsupportedColumnType, if a column type is not supported.
func (this *Connection) File(ctx context.Context, path, format, sheet string, compression Compression, settings json.Value, limit int) (json.Value, types.Type, error) {

	this.core.mustBeOpen()

	c := this.connection

	// Validate the connection type.
	if c.Connector().Type != state.FileStorage {
		return nil, types.Type{}, errors.BadRequest("connection %d is not a file storage connection", c.ID)
	}

	// Ensure that the FileStorage connection supports read operations.
	if !c.Connector().SourceTargets.Contains(state.Users) {
		return nil, types.Type{}, errors.BadRequest("connection %d does not support read operations", c.ID)
	}

	// Validate the path.
	if err := util.ValidateStringField("path", path, MaxFilePathSize); err != nil {
		return nil, types.Type{}, errors.BadRequest("%s", err)
	}

	// Validate the format.
	formatConnector, ok := this.core.state.Connector(format)
	if !ok {
		return nil, types.Type{}, errors.Unprocessable(FormatNotExist, "format %q does not exist", format)
	}
	if formatConnector.Type != state.File {
		return nil, types.Type{}, errors.BadRequest("format %q does not refer to a file connector", format)
	}
	if !formatConnector.SourceTargets.Contains(state.Users) {
		return nil, types.Type{}, errors.BadRequest("format %q does not support reading of users", format)
	}

	// Validate the sheet.
	if formatConnector.HasSheets {
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

	// Validate the settings.
	if formatConnector.HasSourceSettings {
		if settings == nil {
			return nil, types.Type{}, errors.BadRequest("format settings must be provided because connector %s has source settings", formatConnector.Name)
		}
		if !json.Valid(settings) || !settings.IsObject() {
			return nil, types.Type{}, errors.BadRequest("format settings are not a valid JSON Object")
		}
	} else if settings != nil {
		return nil, types.Type{}, errors.BadRequest("format settings cannot be provided because connector %s has no source settings", formatConnector.Name)
	}

	// Validate the limit.
	if limit < 0 || limit > 100 {
		return nil, types.Type{}, errors.BadRequest("limit %d is not valid", limit)
	}

	columns, records, err := this.storage().Read(ctx, formatConnector, path, sheet, settings, state.Compression(compression), limit)
	if err != nil {
		switch err {
		case connectors.ErrNoColumnsFound:
			err = errors.Unprocessable(NoColumnsFound, "file does not have columns")
		case meergo.ErrSheetNotExist:
			err = errors.Unprocessable(SheetNotExist, "file does not contain any sheet named %q", sheet)
		default:
			switch err.(type) {
			case *meergo.InvalidSettingsError:
				err = errors.Unprocessable(InvalidSettings, "%s", err)
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

// Identities returns the user identities of the connection, and an estimate of
// their total number without applying first and limit.
//
// It returns the user identities in range [first,first+limit] with first >= 0
// and 0 < limit <= 1000.
//
// It returns an errors.UnprocessableError error with code MaintenanceMode, if
// the data warehouse is in maintenance mode.
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
	identities, total, err := coreWs.userIdentities(ctx, where, first, limit)
	if err != nil {
		return nil, 0, err
	}
	if identities == nil {
		identities = []UserIdentity{}
	}
	return identities, total, err
}

// LinkConnection links the connection (which must be a website, mobile, or
// server connection) to the connection identified by dst, which must be a
// destination app connection that supports events. If the two connections are
// already linked, this method does nothing.
//
// Returns an errors.NotFoundError if the destination connection does not exist.
func (this *Connection) LinkConnection(ctx context.Context, dst int) error {
	this.core.mustBeOpen()
	if dst < 1 || dst > maxInt32 {
		return errors.BadRequest("dst is not a valid connection identifier")
	}
	// Return if the connections are already linked.
	if slices.Contains(this.connection.LinkedConnections, dst) {
		return nil
	}
	// Validate the source connection.
	if c := this.connection; c.Role == state.Destination {
		return errors.BadRequest("connection %d is not a source", this.connection.ID)
	} else if !c.Connector().SourceTargets.Contains(state.Events) {
		return errors.BadRequest("source %d does not support events", this.connection.ID)
	}
	// Validate the destination connection.
	ws := this.connection.Workspace()
	if c, ok := ws.Connection(dst); !ok {
		return errors.NotFound("connection %d does not exist", dst)
	} else if c.Role == state.Source {
		return errors.BadRequest("connection %d is not a destination", dst)
	} else if connector := c.Connector(); connector.Type != state.App {
		return errors.BadRequest("destination %d is not an app", dst)
	} else if !connector.DestinationTargets.Contains(state.Events) {
		return errors.BadRequest("destination %d does not support events", dst)
	}
	n := state.LinkConnection{
		Connections: [2]int{this.connection.ID, dst},
	}
	err := this.core.state.Transaction(ctx, func(tx *state.Tx) error {
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
			return errors.NotFound("destination %d does not exist", n.Connections[1])
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// PreviewSendEvent returns a preview of an event as it would be dispatches to
// an app. The connection must be a destination app connection, and it is
// expected to have an event type with identifier typ. If there is a
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
func (this *Connection) PreviewSendEvent(ctx context.Context, typ string, event json.Value, transformation DataTransformation, outSchema types.Type) ([]byte, error) {

	this.core.mustBeOpen()

	c := this.connection

	if c.Connector().Type != state.App {
		return nil, errors.BadRequest("connection %d is not an app connection", c.ID)
	}
	if c.Role != state.Destination {
		return nil, errors.BadRequest("connection %d is not a destination", c.ID)
	}
	if !c.Connector().DestinationTargets.Contains(state.Events) {
		return nil, errors.BadRequest("connection %d does not support events", c.ID)
	}
	err := util.ValidateStringField("type", typ, 100)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
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
			action.Transformation.InPaths = mapping.InPaths()
			action.Transformation.OutPaths = mapping.OutPaths()
		case transformation.Function != nil:
			if transformation.Function.Source == "" {
				return nil, errors.BadRequest("transformation source is empty")
			}
			switch transformation.Function.Language {
			case "JavaScript":
				if this.core.transformerProvider == nil || !this.core.transformerProvider.SupportLanguage(state.JavaScript) {
					return nil, errors.Unprocessable(UnsupportedLanguage, "JavaScript transformation language is not supported")
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
			action.Transformation.InPaths = types.PropertyNames(action.InSchema)
			action.Transformation.OutPaths = types.PropertyNames(action.OutSchema)
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
				return nil, errors.Unprocessable(TransformationFailed, "%s", err.Error())
			}
			if err, ok := err.(validationError); ok {
				return nil, errors.Unprocessable(TransformationFailed, "%s", err.Error())
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

	req, err := this.app().EventRequest(ctx, ev, typ, outSchema, transformedProperties, true)
	if err != nil {
		if err == meergo.ErrEventTypeNotExist {
			err = errors.Unprocessable(EventTypeNotExist, "connection %d does not have event type %q", c.ID, typ)
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

// Rename renames the connection with the given new name.
// name must be between 1 and 100 runes long.
//
// It returns an errors.NotFoundError error if the connection does not exist
// anymore.
func (this *Connection) Rename(ctx context.Context, name string) error {
	this.core.mustBeOpen()
	if name == this.connection.Name {
		return nil
	}
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return errors.BadRequest("%s", err)
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

// ServeUI serves the user interface for the connection. event is the event and
// settings are the connection's settings.
//
// It returns an errors.UnprocessableError error with code:
//
//   - EventNotExist, if the event does not exist.
//   - InvalidSettings, if the settings are not valid.
func (this *Connection) ServeUI(ctx context.Context, event string, settings json.Value) (json.Value, error) {
	this.core.mustBeOpen()
	// TODO: check and delete alternative fieldsets keys that have 'null' value
	// before saving to database
	c := this.connection
	connector := c.Connector()
	if c.Role == state.Source && !connector.HasSourceSettings {
		return nil, errors.BadRequest("connector %s does not have source settings", connector.Name)
	}
	if c.Role == state.Destination && !connector.HasDestinationSettings {
		return nil, errors.BadRequest("connector %s does not have destination settings", connector.Name)
	}
	ui, err := this.core.connectors.ServeConnectionUI(ctx, c, event, settings)
	if err != nil {
		if err == meergo.ErrUIEventNotExist {
			err = errors.Unprocessable(EventNotExist, "UI event %q does not exist for connector %s", event, connector.Name)
		} else {
			switch err.(type) {
			case *meergo.InvalidSettingsError:
				err = errors.Unprocessable(InvalidSettings, "%s", err)
			case *connectors.UnavailableError:
				err = errors.Unavailable("%s", err)
			}
		}
		return nil, err
	}
	return ui, nil
}

// Sheets returns the sheets of the file located at the specified path within
// the connection. The connection must be a file storage connection. path must
// be UTF-8 encoded with a length in range [1, MaxFilePathSize].
//
// format specifies the file format and must match the name of a file connector
// that has sheets. compression indicates if the file is compressed and how.
// settings are the format settings.
//
// It returns an errors.UnprocessableError error with code
//   - FormatNotExist, if the format does not exist.
//   - InvalidSettings, if the settings are not valid.
func (this *Connection) Sheets(ctx context.Context, path string, format string, compression Compression, settings json.Value) ([]string, error) {

	this.core.mustBeOpen()

	c := this.connection

	if c.Connector().Type != state.FileStorage {
		return nil, errors.BadRequest("connection %d is not a file storage", c.ID)
	}
	if err := util.ValidateStringField("path", path, MaxFilePathSize); err != nil {
		return nil, errors.BadRequest("%s", err)
	}

	// Validate the file format.
	formatConnector, ok := this.core.state.Connector(format)
	if !ok {
		return nil, errors.Unprocessable(FormatNotExist, "format %q does not exist", format)
	}
	if formatConnector.Type != state.File {
		return nil, errors.BadRequest("format %q does not refer to a file connector", format)
	}
	if !formatConnector.HasSheets {
		return nil, errors.BadRequest("format %q does not have sheets", format)
	}

	// Validate the settings.
	if formatConnector.HasSourceSettings {
		if settings == nil {
			return nil, errors.BadRequest("format settings must be provided because format %s has settings", formatConnector.Name)
		}
		if !json.Valid(settings) || !settings.IsObject() {
			return nil, errors.BadRequest("format settings are not a valid JSON Object")
		}
	} else if settings != nil {
		return nil, errors.BadRequest("format settings cannot be provided because format %s has no settings", formatConnector.Name)
	}

	if !formatConnector.HasSheets {
		return nil, errors.BadRequest("format %s does not have sheets", formatConnector.Name)
	}

	sheets, err := this.storage().Sheets(ctx, formatConnector, path, settings, state.Compression(compression))
	if err != nil {
		switch err.(type) {
		case *meergo.InvalidSettingsError:
			err = errors.Unprocessable(InvalidSettings, "%s", err)
		case *connectors.UnavailableError:
			err = errors.Unavailable("cannot read the file: %w", err)
		}
		return nil, err
	}

	return sheets, nil
}

// TableSchema returns the destination schema of the given table for the
// connection. connection must be a destination database connection, and table
// must be UTF-8 encoded with a length in range [1, MaxTableNameSize].
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
	if err := util.ValidateStringField("table name", table, MaxTableNameSize); err != nil {
		return types.Type{}, errors.BadRequest("%s", err)
	}
	database := this.database()
	defer database.Close()
	schema, err := database.Schema(ctx, table, state.Destination)
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

// UnlinkConnection unlinks the connection (which must be a website, mobile, or
// server connection) from the connection identified by dst, which must be a
// destination app connection that supports events. If the two connections are
// not linked, this method does nothing.
//
// If the destination connection does not exist, it returns an
// errors.NotFoundError.
func (this *Connection) UnlinkConnection(ctx context.Context, dst int) error {
	this.core.mustBeOpen()
	if dst < 1 || dst > maxInt32 {
		return errors.BadRequest("dst is not a valid connection identifier")
	}
	// Return if the connections are not linked.
	if !slices.Contains(this.connection.LinkedConnections, dst) {
		return nil
	}
	// Validate the source connection.
	if c := this.connection; c.Role == state.Destination {
		return errors.BadRequest("connection %d is not a source", this.connection.ID)
	} else if !c.Connector().SourceTargets.Contains(state.Events) {
		return errors.BadRequest("source %d does not support events", this.connection.ID)
	}
	// Validate the destination connection.
	ws := this.connection.Workspace()
	if c, ok := ws.Connection(dst); !ok {
		return errors.NotFound("connection %d does not exist", dst)
	} else if c.Role == state.Source {
		return errors.BadRequest("connection %d is not a destination", dst)
	} else if connector := c.Connector(); connector.Type != state.App {
		return errors.BadRequest("destination %d is not an app", dst)
	} else if !connector.DestinationTargets.Contains(state.Events) {
		return errors.BadRequest("destination %d does not support events", dst)
	}

	n := state.UnlinkConnection{
		Connections: [2]int{this.connection.ID, dst},
	}
	err := this.core.state.Transaction(ctx, func(tx *state.Tx) error {
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
			return errors.NotFound("destination %d does not exist", n.Connections[1])
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// Update updates the connection.
func (this *Connection) Update(ctx context.Context, connection ConnectionToSet) error {

	this.core.mustBeOpen()

	if err := util.ValidateStringField("name", connection.Name, 100); err != nil {
		return errors.BadRequest("%s", err)
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

	n := state.UpdateConnection{
		Connection:  this.connection.ID,
		Name:        connection.Name,
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
		result, err := tx.Exec(ctx, "UPDATE connections SET name = $1,"+
			" strategy = $2, sending_mode = $3, website_host = $4 WHERE id = $5",
			n.Name, n.Strategy, n.SendingMode, n.WebsiteHost, n.Connection)
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

// EventWriteKeys returns the event write keys of the connection.
// The connection must be a source mobile, server or website connection.
func (this *Connection) EventWriteKeys() ([]string, error) {
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

// app returns the app of the connection.
func (this *Connection) app() *connectors.App {
	return this.core.connectors.App(this.connection)
}

// appSchemas returns the user or group schemas, based on target, for an app
// connection. The connection must support the provided target.
//
// For a source connection, it returns only the group source schema.
// For a destination connection, it returns both the group source and
// destination schemas.
func (this *Connection) appSchemas(ctx context.Context, target state.Target) (src, dst types.Type, err error) {
	c := this.connection
	connector := c.Connector()
	if connector.Type != state.App {
		err = errors.BadRequest("connection %d is not an app", c.ID)
		return
	}
	if !connector.DestinationTargets.Contains(target) {
		err = errors.BadRequest("connection %d does not support %s", c.ID, target)
		return
	}
	app := this.app()
	src, err = app.SchemaAsRole(ctx, state.Source, target, "")
	if err != nil {
		if _, ok := err.(*connectors.UnavailableError); ok {
			err = errors.Unavailable("an error occurred fetching the source schema: %w", err)
		}
		return
	}
	if c.Role == state.Destination {
		dst, err = app.SchemaAsRole(ctx, state.Destination, target, "")
		if err != nil {
			if _, ok := err.(*connectors.UnavailableError); ok {
				err = errors.Unavailable("an error occurred fetching the destination schema: %w", err)
			}
			return
		}
	}
	return src, dst, nil
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
// TODO(Gianluca): this function is deprecated and should no longer be used.
// This is retained until https://github.com/meergo/meergo/issues/1266 is
// resolved, then this method will be removed.
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
	// Perform a validation based on the connection's type and role (refer to
	// the specifications in the file "core/Actions.csv" for more details).
	c := this.connection
	connector := c.Connector()
	if target == Events {
		if c.Role == state.Source && eventType != "" {
			return types.Type{}, errors.BadRequest("source connections do not have an event type")
		}
		if c.Role == state.Destination && eventType == "" {
			return types.Type{}, errors.BadRequest("destination connections want an event type")
		}
	}
	// Check if the target is supported by the connection.
	var supportedTargets state.ConnectorTargets
	if c.Role == state.Source {
		supportedTargets = connector.SourceTargets
	} else {
		supportedTargets = connector.DestinationTargets
	}
	if !supportedTargets.Contains(state.Target(target)) {
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

// isMetaProperty reports whether the given property name refers to a property
// considered a meta property by a data warehouse.
func isMetaProperty(name string) bool {
	return len(name) >= 5 && strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__")
}

// isValidStrategy reports whether s is a valid strategy.
func isValidStrategy(s Strategy) bool {
	switch s {
	case "Conversion", "Fusion", "Isolation", "Preservation":
		return true
	}
	return false
}

// isWriteKey reports whether key can be a write key.
func isWriteKey(key string) bool {
	if len(key) != 32 {
		return false
	}
	_, err := base62.DecodeString(key)
	return err == nil
}

// generateEventWriteKey generates an event write key in its base62 form.
func generateEventWriteKey() (string, error) {
	key := make([]byte, 24)
	_, err := rand.Read(key)
	if err != nil {
		return "", errors.New("cannot generate an event write key")
	}
	return base62.EncodeToString(key)[0:32], nil
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

// parseWebsiteHost parses a website host from the format "host:port" and
// returns the host and the port. The host cannot be empty, cannot contain
// invalid UTF-8 characters, cannot contain the NUL byte, and cannot be longer
// than 255 characters. If a port is present, it must be in the range [1,65535].
// If no port is present, it defaults to returning 443 as the port.
func parseWebsiteHost(s string) (string, int, error) {
	h, p, found := strings.Cut(s, ":")
	if err := util.ValidateStringField("website host", h, 255); err != nil {
		return "", 0, err
	}
	port := 443
	if found {
		if port, _ = strconv.Atoi(p); port < 1 || port > 65535 {
			return "", 0, errors.New("website host's port is not valid")
		}
	}
	return h, port, nil
}

// serializeCursor serializes a cursor to be returned by the API.
func serializeCursor(cursor time.Time) (string, error) {
	b, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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
	if role == state.Source {
		if !c.SourceTargets.Contains(state.Events) {
			return errors.BadRequest("connector %s, used as destination, does not support event connections", c.Name)
		}
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
		if role == state.Source {
			// If the connector is Source, the connection's connector must
			// support events as Destination.
			if !ec.Connector().DestinationTargets.Contains(state.Events) {
				return errors.BadRequest("event connection %d does not support events", id)
			}
		} else {
			// If the connector is Destination, the connection's connector must
			// support events as Source.
			if !ec.Connector().SourceTargets.Contains(state.Events) {
				return errors.BadRequest("event connection %d does not support events", id)
			}
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

// ConnectionToSet represents a connection to set in a workspace, by creating a
// new connection (using the method Workspace.CreateConnection) or updating an
// existing one (using the method Connection.Update).
type ConnectionToSet struct {

	// Name is the name of the connection. It cannot be longer than 100 runes.
	// If empty, the connection name will be the name of its connector.
	Name string `json:"name"`

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
