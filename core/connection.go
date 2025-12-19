// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"math"
	mathrand "math/rand/v2"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/internal/connections"
	"github.com/meergo/meergo/core/internal/datastore"
	"github.com/meergo/meergo/core/internal/db"
	"github.com/meergo/meergo/core/internal/schemas"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/transformers"
	"github.com/meergo/meergo/core/internal/transformers/mappings"
	"github.com/meergo/meergo/core/internal/util"
	"github.com/meergo/meergo/tools/errors"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"

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
	LinkedConnections []int         `json:"linkedConnections,omitempty"`
	Health            Health        `json:"-"` // See issue https://github.com/meergo/meergo/issues/1255.
	Pipelines         []Pipeline    `json:"pipelines"`

	// EventTypes is populated only by the (*Workspace).Connection method.
	EventTypes *[]EventType `json:"eventTypes,omitzero"`
}

// EventType represents an event type of a destination API connection.
type EventType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Filter      string `json:"filter"`
}

type PipelineInfo struct {
	ID             int             `json:"id"`
	Target         Target          `json:"target"`
	Enabled        bool            `json:"enabled"`
	ScheduleStart  *int            `json:"scheduleStart"`
	SchedulePeriod *SchedulePeriod `json:"schedulePeriod"`
}

// Strategy represents a strategy.
// Can be "Conversion", "Fusion", "Isolation", and "Preservation".
type Strategy string

type PipelineSchemas struct {
	In        types.Type                `json:"in"`
	Out       types.Type                `json:"out"`
	Matchings *PipelineSchemasMatchings `json:"matchings,omitzero"` // only for destination APIs on users.
}

type PipelineSchemasMatchings struct {
	Internal types.Type `json:"internal"`
	External types.Type `json:"external"`
}

// dummyGroupsSchema is a dummy "groups" schema, that is used until the groups
// management is properly implemented in Meergo. For now, it serves only as a
// placeholder.
var dummyGroupsSchema = types.Object([]types.Property{
	{Name: "id", Type: types.Int(32)},
})

// PipelineType represents a pipeline type.
type PipelineType struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Target      Target  `json:"target"`
	EventType   *string `json:"eventType"`
}

// AbsolutePath returns the absolute representation of the given path, based
// on the connector that must be a file with a storage. path cannot be empty,
// cannot be longer than MaxFilePathSize runes, and must be UTF-8 encoded.
//
// It returns an errors.UnprocessableError error with code:
//   - InvalidPath, if path is not valid for the file storage connector.
//   - InvalidPlaceholder, if path for source connections contains a placeholder
//     or path for destination connections contains an invalid placeholder.
func (this *Connection) AbsolutePath(ctx context.Context, path string) (string, error) {
	this.core.mustBeOpen()
	c := this.connection
	if c.Connector().Type != state.FileStorage {
		return "", errors.BadRequest("connection %d is not a file storage connection", c.ID)
	}
	if err := util.ValidateStringField("path", path, MaxFilePathSize); err != nil {
		return "", errors.BadRequest("%s", err)
	}
	var replacer connections.PlaceholderReplacer
	switch c.Role {
	case state.Source:
		_, err := connections.ReplacePlaceholders(path, func(_ string) (string, bool) {
			return "", false
		})
		if err != nil {
			return "", errors.Unprocessable(InvalidPlaceholder, "the path contains a placeholder syntax, but it cannot be utilized for source pipelines")
		}
	case state.Destination:
		replacer = newPathPlaceholderReplacer(time.Now().UTC())
	}
	path, err := this.storage().AbsolutePath(ctx, path, replacer)
	if err != nil {
		switch err.(type) {
		case *connectors.InvalidPathError:
			err = errors.Unprocessable(InvalidPath, "%s", err)
		case *connections.PlaceholderError:
			err = errors.Unprocessable(InvalidPlaceholder, "%s", err)
		case *connections.UnavailableError:
			err = errors.Unavailable("%w", err)
		}
		return "", err
	}
	return path, nil
}

// APIEventSchema returns the schema of the provided event type of the
// connection. If the event type does not have a schema, it returns an invalid
// schema. The connection must be a destination API connection that supports
// events.
//
// It returns an errors.NotFoundError error if the event type does not exist.
func (this *Connection) APIEventSchema(ctx context.Context, eventType string) (types.Type, error) {
	this.core.mustBeOpen()
	if eventType == "" {
		return types.Type{}, errors.BadRequest("event type is empty")
	}
	c := this.connection
	connector := c.Connector()
	if connector.Type != state.API {
		return types.Type{}, errors.BadRequest("connection %d is not an API", c.ID)
	}
	if c.Role != state.Destination {
		return types.Type{}, errors.BadRequest("connection %d is not a destination", c.ID)
	}
	if !connector.DestinationTargets.Contains(state.TargetEvent) {
		return types.Type{}, errors.BadRequest("connection %d does not support events", c.ID)
	}
	schema, err := this.api().SchemaAsRole(ctx, state.Destination, state.TargetEvent, eventType)
	if err != nil {
		if _, ok := err.(*connections.UnavailableError); ok {
			err = errors.Unavailable("an error occurred fetching the schema: %w", err)
		}
		if err == connectors.ErrEventTypeNotExist {
			err = errors.NotFound("event type %q does not exist", eventType)
		}
		return types.Type{}, err
	}
	return schema, nil // schema can be invalid.
}

// APIGroupSchemas returns the group schemas for the connection. The connection
// must be an API connection that supports groups. For a source, it returns only
// the source schema. For a destination, it returns both the source and
// destination schemas.
//
// TODO(Gianluca): this method is currently unused, and it has been kept for the
// future, when we will re-expose the endpoint to retrieve group schemas. See
// the issue https://github.com/meergo/meergo/issues/895.
func (this *Connection) APIGroupSchemas(ctx context.Context) (src, dst types.Type, err error) {
	this.core.mustBeOpen()
	return this.appSchemas(ctx, state.TargetGroup)
}

// APIUserSchemas returns the user schemas for the connection. The connection
// must be an API connection that supports users. For a source, it returns only
// the source schema. For a destination, it returns both the source and
// destination schemas.
func (this *Connection) APIUserSchemas(ctx context.Context) (src, dst types.Type, err error) {
	this.core.mustBeOpen()
	return this.appSchemas(ctx, state.TargetUser)
}

// APIUsers returns the users of an API connection and the cursor to get the
// next users. If filter is not nil, only users matching its conditions will be
// returned.The returned cursor is empty if there are no other users.
//
// It returns an errors.UnprocessableError error with code SchemaNotAligned if
// the provided schema is not aligned with the API's source schema.
func (this *Connection) APIUsers(ctx context.Context, schema types.Type, filter *Filter, cursor string) (json.Value, string, error) {

	this.core.mustBeOpen()

	if this.connection.Connector().Type != state.API {
		return nil, "", errors.BadRequest("connection %d is not an API connection", this.connection.ID)
	}
	if !this.connection.Connector().SourceTargets.Contains(state.TargetUser) {
		return nil, "", errors.BadRequest("connection %d does not support reading of users", this.connection.ID)
	}
	if !schema.Valid() {
		return nil, "", errors.BadRequest("schema is not valid")
	}

	// Validate the filter.
	var where *state.Where
	if filter != nil {
		_, err := validateFilter(filter, schema, state.Source, state.TargetUser)
		if err != nil {
			if err, ok := err.(types.PathNotExistError); ok {
				return nil, "", errors.BadRequest("filter's property %q does not exist", err.Path)
			}
			return nil, "", errors.BadRequest("filter is not valid: %w", err)
		}
		where = convertFilterToWhere(filter, schema)
	}

	// Validate the cursor.
	var lastChangeTime time.Time
	if cursor != "" {
		var err error
		lastChangeTime, err = deserializeCursor(cursor)
		if err != nil {
			return nil, "", errors.BadRequest("cursor is malformed")
		}
	}

	// Get the users.
	records, err := this.api().Users(ctx, schema, where, lastChangeTime)
	if err != nil {
		switch err.(type) {
		case *connections.UnavailableError:
			err = errors.Unavailable("%s", err)
		case *schemas.Error:
			err = errors.Unprocessable(SchemaNotAligned, "schema is not aligned with the API's source schema: %w", err)
		}
		return nil, "", err
	}
	defer records.Close()

	var last connections.Record
	users := make([]any, 0, 100)

	for user := range records.All(ctx) {
		if user.Err != nil {
			return nil, "", errors.Unavailable("%s has returned an invalid user; %s", this.api().Connector(), user.Err)
		}
		users = append(users, user.Attributes)
		if records.Last() {
			last = user
		}
		if len(users) == 100 {
			break
		}
	}
	if err = records.Err(); err != nil {
		if _, ok := err.(*connections.UnavailableError); ok {
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

// CreatePipeline creates a pipeline for the connection returning the identifier
// of the created pipeline. target is the target of the pipeline and must be
// supported by the connector of the connection.
//
// Refer to the specifications in the file "core/Pipelines.csv" for more
// details.
//
// It returns an errors.NotFoundError error if the connection does not exist
// anymore, and returns an errors.UnprocessableError error with code
//
//   - ConnectionNotExist, if the connection does not exist.
//   - EventTypeNotExist, if the event type does not exist for the connection.
//   - FormatNotExist, if the format of the pipeline does not exist.
//   - InvalidSettings, if the settings are not valid.
//   - TargetExist, if a pipeline already exists for a target for the
//     connection.
//   - UnsupportedLanguage, if the transformation language is not supported.
func (this *Connection) CreatePipeline(ctx context.Context, target Target, eventType string, pipeline PipelineToSet) (int, error) {

	this.core.mustBeOpen()

	// Retrieve the format, if specified in the pipeline.
	var format *state.Connector
	if pipeline.Format != "" {
		format, _ = this.core.state.Connector(pipeline.Format)
	}

	c := this.connection
	connector := c.Connector()

	// Validate the target.
	if target != TargetUser && target != TargetGroup && target != TargetEvent {
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
		return 0, errors.BadRequest("pipeline with target '%s' not allowed for %s %s connections", target, role, typ)
	}

	// Validate the event type.
	requiresEventType := c.Role == state.Destination && connector.Type == state.API && target == TargetEvent
	if requiresEventType && eventType == "" {
		return 0, errors.BadRequest("eventType is required for pipelines that send events to apps")
	}
	if !requiresEventType && eventType != "" {
		role := strings.ToLower(c.Role.String())
		typ := strings.ToLower(connector.Type.String())
		return 0, errors.BadRequest("pipelines with target '%s' on %s %s connections cannot specify an event type", target, role, typ)
	}
	if eventType != "" {
		if err := util.ValidateStringField("eventType", eventType, 100); err != nil {
			return 0, errors.BadRequest("%s", err)
		}
	}

	// Validate the pipeline.
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
	v.provider = this.core.functionProvider
	err := validatePipelineToSet(pipeline, v)
	if err != nil {
		return 0, err
	}

	// Determine the input schema.
	inSchema := pipeline.InSchema
	importUserIdentitiesFromEvents := isImportingUserIdentitiesFromEvents(connector.Type, c.Role, state.Target(target))
	dispatchEventsToAPIs := isDispatchingEventsToAPIs(connector.Type, c.Role, state.Target(target))
	importEventsIntoWarehouse := isImportingEventsIntoWarehouse(connector.Type, c.Role, state.Target(target))
	if importUserIdentitiesFromEvents || importEventsIntoWarehouse || dispatchEventsToAPIs {
		inSchema = eventPipelineSchema
	}

	n := state.CreatePipeline{
		Connection:           c.ID,
		Target:               state.Target(target),
		Name:                 pipeline.Name,
		Enabled:              pipeline.Enabled,
		EventType:            eventType,
		InSchema:             inSchema,
		OutSchema:            pipeline.OutSchema,
		Transformation:       toStateTransformation(pipeline.Transformation, inSchema, pipeline.OutSchema),
		Query:                pipeline.Query,
		Format:               pipeline.Format,
		Path:                 pipeline.Path,
		Sheet:                pipeline.Sheet,
		Compression:          state.Compression(pipeline.Compression),
		OrderBy:              pipeline.OrderBy,
		ExportMode:           state.ExportMode(pipeline.ExportMode),
		Matching:             state.Matching(pipeline.Matching),
		UpdateOnDuplicates:   pipeline.UpdateOnDuplicates,
		TableName:            pipeline.TableName,
		TableKey:             pipeline.TableKey,
		IdentityColumn:       pipeline.IdentityColumn,
		LastChangeTimeColumn: pipeline.LastChangeTimeColumn,
		LastChangeTimeFormat: pipeline.LastChangeTimeFormat,
		Incremental:          pipeline.Incremental,
	}

	// Set the scheduler.
	if n.Target == state.TargetUser || n.Target == state.TargetGroup {
		n.ScheduleStart = int16(mathrand.IntN(24 * 60))
		n.SchedulePeriod = 0 // do not automatically schedule the pipeline when creating it.
	}

	// Add the filter to the notification.
	if pipeline.Filter != nil {
		n.Filter, _ = convertFilterToWhere(pipeline.Filter, inSchema).MarshalJSON()
	}

	// Determine the connector code, for file pipelines.
	var formatCode *string
	if format != nil {
		code := format.Code
		formatCode = &code
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
	rawOutSchema, err := marshalSchema(pipeline.OutSchema)
	if err != nil {
		return 0, err
	}

	// Marshal the mapping.
	var mapping []byte
	if tr := pipeline.Transformation; tr != nil && tr.Mapping != nil {
		mapping, err = json.Marshal(tr.Mapping)
		if err != nil {
			return 0, err
		}
	}

	var function state.TransformationFunction
	if fn := n.Transformation.Function; fn != nil {
		name := transformationFunctionName(n.ID)
		fn.ID, fn.Version, err = this.core.functionProvider.Create(ctx, name, fn.Language, fn.Source)
		if err != nil {
			return 0, err
		}
		function = *n.Transformation.Function
	}

	// Format settings.
	if format != nil && pipeline.FormatSettings != nil {
		conf := &connections.ConnectorConfig{
			Role: this.connection.Role,
		}
		n.FormatSettings, err = this.core.connections.UpdatedSettings(ctx, format, conf, pipeline.FormatSettings)
		if err != nil {
			switch err.(type) {
			case *connectors.InvalidSettingsError:
				err = errors.Unprocessable(InvalidSettings, "%s", err)
			case *connections.UnavailableError:
				err = errors.Unavailable("%s", err)
			}
			return 0, err
		}
	}

	// Add the pipeline.
	err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		switch n.Target {
		case state.TargetEvent:
			switch connector.Type {
			case state.SDK, state.Webhook:
				exists, err := tx.QueryExists(ctx, "SELECT FROM pipelines WHERE connection = $1 AND target = 'Event'", n.Connection)
				if err != nil {
					return nil, err
				}
				if exists {
					return nil, errors.Unprocessable(TargetExist,
						"pipeline with target %s already exists for %s connection %d", n.Target, connector.Type, n.Connection)
				}
			}
		case state.TargetUser, state.TargetGroup:
			// Make sure that users and groups pipelines have the same schedule start.
			err = tx.QueryRow(ctx, "SELECT schedule_start FROM pipelines WHERE connection = $1\n"+
				" AND target IN ('User', 'Group') LIMIT 1", n.Connection).Scan(&n.ScheduleStart)
			if err != nil && err != sql.ErrNoRows {
				return nil, err
			}
		}
		query := "INSERT INTO pipelines (id, connection, target, event_type, name, enabled,\n" +
			"schedule_start, schedule_period, in_schema, out_schema, filter, transformation_mapping,\n" +
			"transformation_id, transformation_version, transformation_language, transformation_source,\n" +
			"transformation_preserve_json, transformation_in_paths, transformation_out_paths, query, format, path,\n" +
			"sheet, compression, order_by, format_settings, export_mode, matching_in, matching_out,\n" +
			"update_on_duplicates, table_name, table_key, identity_column, last_change_time_column,\n" +
			"last_change_time_format, incremental)\n" +
			"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21,\n" +
			"$22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36)"
		_, err := tx.Exec(ctx, query, n.ID, n.Connection, n.Target, n.EventType,
			n.Name, n.Enabled, n.ScheduleStart, n.SchedulePeriod, rawInSchema, rawOutSchema,
			n.Filter, mapping, function.ID, function.Version, function.Language, function.Source, function.PreserveJSON,
			n.Transformation.InPaths, n.Transformation.OutPaths, n.Query, formatCode, n.Path, n.Sheet,
			n.Compression, n.OrderBy, n.FormatSettings, n.ExportMode, n.Matching.In, n.Matching.Out, n.UpdateOnDuplicates,
			n.TableName, n.TableKey, n.IdentityColumn, n.LastChangeTimeColumn, n.LastChangeTimeFormat, n.Incremental)
		if err != nil {
			if db.IsForeignKeyViolation(err) && db.ErrConstraintName(err) == "pipelines_connection_fkey" {
				err = errors.Unprocessable(ConnectionNotExist, "connection %d does not exist", n.Connection)
			}
			return nil, err
		}
		return n, nil
	})
	if err != nil {
		return 0, err
	}

	return n.ID, nil
}

// CreateEventWriteKey creates a new event write key for the connection.
// The connection must be an SDK or webhook source.
//
// If the connection does not exist, it returns an errors.NotFoundError error.
// If the connection has already too many keys, it returns an
// errors.UnprocessableError error with code TooManyEventWriteKeys.
func (this *Connection) CreateEventWriteKey(ctx context.Context) (string, error) {
	this.core.mustBeOpen()
	c := this.connection
	connector := c.Connector()
	switch connector.Type {
	case state.SDK, state.Webhook:
	default:
		return "", errors.NotFound("connection %d is neither an SDK nor a webhook", c.ID)
	}
	if c.Role != state.Source {
		return "", errors.NotFound("connection %d is not a source", c.ID)
	}
	key, err := generateEventWriteKey()
	if err != nil {
		return "", err
	}
	n := state.CreateEventWriteKey{
		Connection: c.ID,
		Key:        key,
		CreatedAt:  time.Now().UTC(),
	}
	err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		var count int
		err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM event_write_keys WHERE connection = $1", n.Connection).Scan(&count)
		if err != nil {
			return nil, err
		}
		if count == maxKeysPerConnection {
			return nil, errors.Unprocessable(TooManyEventWriteKeys, "connection %d already has %d event write keys", n.Connection, maxKeysPerConnection)
		}
		_, err = tx.Exec(ctx, "INSERT INTO event_write_keys (connection, key, created_at) VALUES ($1, $2, $3)",
			n.Connection, n.Key, n.CreatedAt)
		if err != nil {
			if db.IsForeignKeyViolation(err) {
				if db.ErrConstraintName(err) == "event_write_keys_connection_fkey" {
					err = errors.NotFound("connection %d does not exist", n.Connection)
				}
			}
			return nil, err
		}
		return n, nil
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
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		// Mark the connection's functions as discontinued.
		now := time.Now().UTC()
		_, err := tx.Exec(ctx, "INSERT INTO discontinued_functions (id, discontinued_at)\n"+
			"SELECT p.transformation_id, $1\n"+
			"FROM pipelines AS p\n"+
			"WHERE p.transformation_id != '' AND p.connection = $2\n"+
			"ON CONFLICT (id) DO NOTHING", now, n.ID)
		if err != nil {
			return nil, err
		}
		// Mark the connection's pipelines on Users as deleted.
		if c.Role == state.Source {
			_, err := tx.Exec(ctx, "UPDATE workspaces SET pipelines_to_purge = array_cat(pipelines_to_purge, (\n"+
				"\tSELECT array_agg(a.id) FROM pipelines a WHERE a.connection = $1 AND a.target = 'User'\n"+
				"))\nWHERE id = $2", n.ID, workspace.ID)
			if err != nil {
				return nil, err
			}
		}
		// Delete the connection.
		result, err := tx.Exec(ctx, "DELETE FROM connections WHERE id = $1", n.ID)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("connection %d does not exist", n.ID)
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
			return nil, err
		}
		if connector.OAuth != nil {
			// Delete the account of the deleted connection if it has no other connections.
			result, err := tx.Exec(ctx, "DELETE FROM accounts a\nWHERE NOT EXISTS (\n"+
				"\tSELECT 1 FROM connections c\n"+
				"\tWHERE c.account = a.id\n)")
			if err != nil {
				return nil, err
			}
			if result.RowsAffected() > 0 {
				n.Account = true
			}
		}
		// Remove the connection as primary source, if any.
		_, err = tx.Exec(ctx, "DELETE FROM primary_sources WHERE source = $1", n.ID)
		if err != nil {
			return nil, err
		}
		// If there is an alter schema in progress, removes the connection from
		// the primary sources which will take effect when the new profile schema
		// is applied.
		query := "SELECT alter_profile_schema_primary_sources FROM workspaces" +
			" WHERE id = $1 AND alter_profile_schema_primary_sources IS NOT NULL"
		var primarySources map[string]int
		err = tx.QueryRow(ctx, query, workspace.ID).Scan(&primarySources)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if primarySources != nil {
			changed := false
			for prop, source := range primarySources {
				if source == n.ID {
					delete(primarySources, prop)
					changed = true
				}
			}
			if changed {
				_, err = tx.Exec(ctx, "UPDATE workspaces SET"+
					" alter_profile_schema_primary_sources = $1 WHERE id = $2",
					primarySources, workspace.ID)
				if err != nil {
					return nil, err
				}
			}
		}
		return n, nil
	})
	return err
}

// DeleteEventWriteKey deletes the given event write key of the connection.
// key cannot be empty and cannot be the only key for the connection.
// The connection must be an SDK or webhook source.
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
	case state.SDK, state.Webhook:
	default:
		return errors.BadRequest("connection %d is neither an SDK nor a webhook", c.ID)
	}
	if c.Role != state.Source {
		return errors.BadRequest("connection %d is not a source", c.ID)
	}
	n := state.DeleteEventWriteKey{
		Connection: c.ID,
		Key:        key,
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		var count int
		err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM event_write_keys WHERE connection = $1", n.Connection).Scan(&count)
		if err != nil {
			return nil, err
		}
		if count == 1 {
			return nil, errors.Unprocessable(SingleEventWriteKey, "key cannot be deleted as it is the connection’s only key")
		}
		result, err := tx.Exec(ctx, "DELETE FROM event_write_keys WHERE connection = $1 AND key = $2", n.Connection, n.Key)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("key %q does not exist", key)
		}
		return n, nil
	})

	return err
}

// ExecQuery executes the given query on the connection and returns the
// resulting rows and schema. The connection must be a source database
// connection.
//
// query must be UTF-8 encoded, it cannot be longer than 16,777,215 runes.
// limit must be in range [0, 100].
//
// The method may also return issues that did not prevent it from being
// processed. These issues are reported as a slice of strings.
//
// If the connection does not exist, it returns an errors.NotFoundError error.
// It returns an errors.UnprocessableError error with code:
//
//   - InvalidPlaceholder, if the query contains an invalid placeholder.
//   - UnsupportedColumnType, if a column type is not supported.
func (this *Connection) ExecQuery(ctx context.Context, query string, limit int) (json.Value, types.Type, []string, error) {

	this.core.mustBeOpen()

	if err := util.ValidateStringField("query", query, queryMaxSize); err != nil {
		return nil, types.Type{}, nil, errors.BadRequest("%s", err)
	}
	if limit < 0 || limit > 100 {
		return nil, types.Type{}, nil, errors.BadRequest("limit %d is not valid", limit)
	}

	c := this.connection
	connector := c.Connector()
	if connector.Type != state.Database {
		return nil, types.Type{}, nil, errors.BadRequest("connection %d is not a database", c.ID)
	}
	if c.Role != state.Source {
		return nil, types.Type{}, nil, errors.BadRequest("database %d is not a source", c.ID)
	}

	// Execute the query.
	database := this.database()
	replacer := func(name string) (string, bool) {
		switch name {
		case "updated_at":
			v, _ := database.UpdatedAtPlaceholder(nil)
			return v, true
		case "limit":
			return strconv.Itoa(limit), true
		}
		return "", false
	}
	defer database.Close()
	rows, schema, issues, err := database.Query(ctx, query, replacer)
	if err != nil {
		switch err.(type) {
		case *connections.PlaceholderError:
			err = errors.Unprocessable(InvalidPlaceholder, "%s", err)
		case *connections.UnavailableError:
			err = errors.Unavailable("%s", err)
		}
		return nil, types.Type{}, nil, err
	}
	defer rows.Close()
	if !schema.Valid() {
		return json.Value("[]"), types.Type{}, issues, nil
	}

	// Scan the rows.
	var results []any
	for rows.Next() {
		if len(results) == limit {
			break
		}
		row, err := rows.Scan()
		if err != nil {
			if _, ok := err.(*connections.UnavailableError); ok {
				err = errors.Unavailable("%s", err)
			}
			return nil, types.Type{}, nil, err
		}
		results = append(results, row)
	}
	err = rows.Err()
	if err != nil {
		if _, ok := err.(*connections.UnavailableError); ok {
			err = errors.Unavailable("%s", err)
		}
		return nil, types.Type{}, nil, err
	}
	_ = rows.Close()

	marshaledRows, err := types.Marshal(results, types.Array(schema))
	if err != nil {
		return nil, types.Type{}, nil, err
	}

	return marshaledRows, schema, issues, nil
}

// A PipelineRun describes a pipeline run as returned by Runs.
type PipelineRun struct {
	ID        int        `json:"id"`
	Pipeline  int        `json:"pipeline"`
	StartTime time.Time  `json:"startTime"`
	EndTime   *time.Time `json:"endTime"`
	Passed    [6]int     `json:"passed"`
	Failed    [6]int     `json:"failed"`
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
// The method may also return issues encountered during the reading process that
// did not prevent the file from being processed. These issues are reported as
// a slice of strings.
//
// It returns an errors.UnprocessableError error with code
//
//   - FormatNotExist, if the format does not exist.
//   - InvalidSettings, if the settings are not valid.
//   - NoColumnsFound, if the file has no columns.
//   - SheetNotExist, if the file does not contain the provided sheet.
//   - UnsupportedColumnType, if a column type is not supported.
func (this *Connection) File(ctx context.Context, path, format, sheet string, compression Compression, settings json.Value, limit int) (json.Value, types.Type, []string, error) {

	this.core.mustBeOpen()

	c := this.connection

	// Validate the connection type.
	if c.Connector().Type != state.FileStorage {
		return nil, types.Type{}, nil, errors.BadRequest("connection %d is not a file storage connection", c.ID)
	}

	// Ensure that the FileStorage connection supports read operations.
	if !c.Connector().SourceTargets.Contains(state.TargetUser) {
		return nil, types.Type{}, nil, errors.BadRequest("connection %d does not support read operations", c.ID)
	}

	// Validate the path.
	if err := util.ValidateStringField("path", path, MaxFilePathSize); err != nil {
		return nil, types.Type{}, nil, errors.BadRequest("%s", err)
	}

	// Validate the format.
	formatConnector, ok := this.core.state.Connector(format)
	if !ok {
		return nil, types.Type{}, nil, errors.Unprocessable(FormatNotExist, "format %q does not exist", format)
	}
	if formatConnector.Type != state.File {
		return nil, types.Type{}, nil, errors.BadRequest("format %q does not refer to a file connector", format)
	}
	if !formatConnector.SourceTargets.Contains(state.TargetUser) {
		return nil, types.Type{}, nil, errors.BadRequest("format %q does not support reading of users", format)
	}

	// Validate the sheet.
	if formatConnector.HasSheets {
		if sheet == "" {
			return nil, types.Type{}, nil, errors.BadRequest("sheet cannot be empty because connection %d has sheets", c.ID)
		}
		if !connections.IsValidSheetName(sheet) {
			return nil, types.Type{}, nil, errors.BadRequest("sheet is not valid")
		}
	} else {
		if sheet != "" {
			return nil, types.Type{}, nil, errors.BadRequest("sheet must be empty because connection %d does not have sheets", c.ID)
		}
	}

	// Validate the settings.
	if formatConnector.HasSourceSettings {
		if settings == nil {
			return nil, types.Type{}, nil, errors.BadRequest("format settings must be provided because connector %s has source settings", formatConnector.Code)
		}
		if !json.Valid(settings) || !settings.IsObject() {
			return nil, types.Type{}, nil, errors.BadRequest("format settings are not a valid JSON Object")
		}
	} else if settings != nil {
		return nil, types.Type{}, nil, errors.BadRequest("format settings cannot be provided because connector %s has no source settings", formatConnector.Code)
	}

	// Validate the limit.
	if limit < 0 || limit > 100 {
		return nil, types.Type{}, nil, errors.BadRequest("limit %d is not valid", limit)
	}

	columns, records, issues, err := this.storage().Read(ctx, formatConnector, path, sheet, settings, state.Compression(compression), limit)
	if err != nil {
		switch err {
		case connections.ErrNoColumnsFound:
			err = errors.Unprocessable(NoColumnsFound, "file does not have columns")
		case connectors.ErrSheetNotExist:
			err = errors.Unprocessable(SheetNotExist, "file does not contain any sheet named %q", sheet)
		default:
			switch err.(type) {
			case *connectors.InvalidSettingsError:
				err = errors.Unprocessable(InvalidSettings, "%s", err)
			case *connections.UnavailableError:
				err = errors.Unavailable("cannot read records: %w", err)
			}
		}
		return nil, types.Type{}, nil, err
	}

	recs := make([]any, len(records))
	for i, r := range records {
		recs[i] = r
	}

	schema := types.Object(columns)
	marshaledRecords, err := types.Marshal(recs, types.Array(schema))
	if err != nil {
		return nil, types.Type{}, nil, err
	}

	return marshaledRecords, schema, issues, nil
}

// Identities returns the identities of the connection, and an estimate of their
// total number without applying first and limit.
//
// It returns the identities in range [first,first+limit] with first >= 0 and
// 0 < limit <= 1000.
//
// Identities are sorted by updated-at time in descending order, so the most
// recently updated identities come first.
//
// It returns an errors.UnprocessableError error with code MaintenanceMode, if
// the data warehouse is in maintenance mode.
func (this *Connection) Identities(ctx context.Context, first, limit int) ([]Identity, int, error) {
	this.core.mustBeOpen()
	if first < 0 {
		return nil, 0, errors.BadRequest("first %d is not valid", first)
	}
	if limit < 1 || limit > 1000 {
		return nil, 0, errors.BadRequest("limit %d is not valid", limit)
	}
	ws := &Workspace{
		core:      this.core,
		store:     this.store,
		workspace: this.connection.Workspace(),
	}
	where := &state.Where{Logical: state.OpAnd, Conditions: []state.WhereCondition{{
		Property: []string{"_connection"},
		Operator: state.OpIs,
		Values:   []any{this.connection.ID},
	}}}
	identities, total, err := ws.identities(ctx, where, first, limit)
	if err != nil {
		return nil, 0, err
	}
	if identities == nil {
		identities = []Identity{}
	}
	return identities, total, err
}

// LinkConnection links an SDK or webhook connection to the destination
// connection identified by dst, which must support events. If the connections
// are already linked, the method does nothing.
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
	} else if !c.Connector().SourceTargets.Contains(state.TargetEvent) {
		return errors.BadRequest("source %d does not support events", this.connection.ID)
	}
	// Validate the destination connection.
	ws := this.connection.Workspace()
	if c, ok := ws.Connection(dst); !ok {
		return errors.NotFound("connection %d does not exist", dst)
	} else if c.Role != state.Destination {
		return errors.BadRequest("connection %d is not a destination", dst)
	} else if connector := c.Connector(); !connector.DestinationTargets.Contains(state.TargetEvent) {
		return errors.BadRequest("destination %d does not support events", dst)
	}
	n := state.LinkConnection{
		Connections: [2]int{this.connection.ID, dst},
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		const add = "UPDATE connections\n" +
			"SET linked_connections = (SELECT ARRAY(SELECT DISTINCT unnest(array_append(linked_connections, $2)) ORDER BY 1))\n" +
			"WHERE id = $1"
		result, err := tx.Exec(ctx, add+" AND NOT COALESCE($2 = ANY(linked_connections), FALSE)", n.Connections[0], n.Connections[1])
		if err != nil {
			return nil, err
		}
		// Do nothing if the source connection does not exist or if they are already linked.
		if result.RowsAffected() == 0 {
			return nil, nil
		}
		result, err = tx.Exec(ctx, add, n.Connections[1], n.Connections[0])
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("destination %d does not exist", n.Connections[1])
		}
		return n, nil
	})
	return err
}

// PipelineSchemas returns the input and the output schemas of a pipeline with
// the given target and event type.
//
// TODO(Gianluca): this method is deprecated. See the issue
// https://github.com/meergo/meergo/issues/1266.
//
// It returns an errors.UnprocessableError error with code EventTypeNotExist, if
// the event type does not exist for the connection.
func (this *Connection) PipelineSchemas(ctx context.Context, target Target, eventType string) (*PipelineSchemas, error) {

	this.core.mustBeOpen()

	// Validate the target and the event type.
	eventTypeSchema, err := this.validateTargetAndEventType(ctx, target, eventType)
	if err != nil {
		return nil, err
	}

	profiles := this.connection.Workspace().ProfileSchema
	groups := dummyGroupsSchema

	c := this.connection

	switch connector := c.Connector(); connector.Type {

	case state.API:
		switch target {
		case TargetUser:
			var err error
			// Retrieve the API's source or target schema, depending on the connection's role.
			schema, err := this.api().Schema(ctx, state.TargetUser, "")
			if err != nil {
				if _, ok := err.(*connections.UnavailableError); ok {
					err = errors.Unavailable("an error occurred fetching the schema: %w", err)
				}
				return nil, err
			}
			if c.Role == state.Source {
				// Source/API/User.
				return &PipelineSchemas{In: schema, Out: profiles}, nil
			} else {
				// Destination/API/User.
				//
				// The API's destination schema is already available here, but
				// we need to get the source one too because it's needed for the
				// matching properties.
				sourceSchema, err := this.api().SchemaAsRole(ctx, state.Source, state.TargetUser, "")
				if err != nil {
					if _, ok := err.(*connections.UnavailableError); ok {
						err = errors.Unavailable("an error occurred fetching the schema: %w", err)
					}
					return nil, err
				}
				pipelineSchemas := &PipelineSchemas{In: profiles, Out: schema}
				pipelineSchemas.Matchings = &PipelineSchemasMatchings{
					Internal: onlyForMatching(profiles),
					External: onlyForMatching(sourceSchema),
				}
				return pipelineSchemas, nil
			}
		case TargetGroup:
			var err error
			schema, err := this.api().Schema(ctx, state.TargetGroup, "")
			if err != nil {
				if _, ok := err.(*connections.UnavailableError); ok {
					err = errors.Unavailable("an error occurred fetching the schema: %w", err)
				}
				return nil, err
			}
			if c.Role == state.Source {
				// Source/API/Group.
				return &PipelineSchemas{In: schema, Out: groups}, nil
			} else {
				// Destination/API/Group.
				sourceSchema, err := this.api().SchemaAsRole(ctx, state.Source, state.TargetGroup, "")
				if err != nil {
					if _, ok := err.(*connections.UnavailableError); ok {
						err = errors.Unavailable("an error occurred fetching the schema: %w", err)
					}
					return nil, err
				}
				pipelineSchemas := &PipelineSchemas{In: groups, Out: schema}
				pipelineSchemas.Matchings = &PipelineSchemasMatchings{
					Internal: onlyForMatching(groups),
					External: onlyForMatching(sourceSchema),
				}
				return pipelineSchemas, nil
			}
		case TargetEvent:
			return &PipelineSchemas{In: schemas.Event, Out: eventTypeSchema}, nil
		}

	case state.Database:
		switch target {
		case TargetUser:
			if c.Role == state.Source {
				// Source/Database/User.
				//
				// The input schema is not set here because it is retrieved via
				// a separate API call, since it depends on the query, which in
				// the UI case is entered interactively by the user.
				return &PipelineSchemas{Out: profiles}, nil
			} else {
				// Destination/Database/User.
				//
				// The output schema depends on the table chosen for export, and
				// must be retrieved separately.
				return &PipelineSchemas{In: profiles}, nil
			}
		case TargetGroup:
			if c.Role == state.Source {
				// Source/Database/Group.
				return &PipelineSchemas{Out: groups}, nil
			} else {
				// Destination/Database/Group.
				return &PipelineSchemas{In: groups}, nil
			}
		}

	case state.FileStorage:
		switch target {
		case TargetUser:
			if c.Role == state.Source {
				// Source/FileStorage/Source.
				//
				// The input schema is not set here because it is retrieved via
				// a separate API call, since it depends on the file, which in
				// the UI case is entered interactively by the user.
				return &PipelineSchemas{Out: profiles}, nil
			} else {
				// Destination/FileStorage/Source.
				return &PipelineSchemas{In: profiles}, nil
			}
		case TargetGroup:
			if c.Role == state.Source {
				// Source/FileStorage/Group.
				return &PipelineSchemas{Out: groups}, nil
			} else {
				// Destination/FileStorage/Group.
				return &PipelineSchemas{In: groups}, nil
			}
		}

	case state.MessageBroker, state.SDK, state.Webhook:
		if eventType != "" {
			return nil, errors.NotFound("event type not expected")
		}
		// TODO(Gianluca): regarding message broker connectors, see the issue
		// https://github.com/meergo/meergo/issues/1264.
		switch target {
		case TargetUser:
			// Source/SDK/User.
			return &PipelineSchemas{In: schemas.Event, Out: profiles}, nil
		case TargetGroup:
			// Source/SDK/Group.
			return &PipelineSchemas{In: schemas.Event, Out: groups}, nil
		case TargetEvent:
			// Source/SDK/Event.
			return &PipelineSchemas{In: schemas.Event}, nil
		}
		return &PipelineSchemas{}, nil

	}

	panic("unreachable code")
}

// PipelineTypes returns the pipeline types for the connection.
//
// TODO(Gianluca): this method is deprecated. See the issue
// https://github.com/meergo/meergo/issues/1265.
//
// Refer to the specifications in the file "core/Pipelines.csv" for more
// details.
func (this *Connection) PipelineTypes(ctx context.Context) ([]PipelineType, error) {
	this.core.mustBeOpen()
	var pipelineTypes []PipelineType
	c := this.connection
	connector := c.Connector()
	var targets state.ConnectorTargets
	if c.Role == state.Source {
		targets = connector.SourceTargets
	} else {
		targets = connector.DestinationTargets
	}
	if targets.Contains(state.TargetUser) {
		switch typ := c.Connector().Type; typ {
		case
			state.API:
			var name, description string
			if c.Role == state.Source {
				// Source/API/User.
				name = "Import " + connector.Label + " " + strings.ToLower(connector.Terms.Users)
				description = "Import " + strings.ToLower(connector.Terms.Users)
				if connector.Terms.Users != "Users" {
					description += " as users"
				}
				description += " into the data warehouse"
			} else {
				// Destination/API/User.
				name = "Export " + strings.ToLower(connector.Terms.Users)
				description = "Export users from the data warehouse"
				if connector.Terms.Users != "Users" {
					description += " as " + strings.ToLower(connector.Terms.Users)
				}
				description += " to " + connector.Label
			}
			at := PipelineType{
				Name:        name,
				Description: description,
				Target:      TargetUser,
			}
			pipelineTypes = append(pipelineTypes, at)
		case
			state.Database,
			state.FileStorage:
			var name, description string
			if c.Role == state.Source {
				// Source/FileStorage/Users.
				// Source/Database/Users.
				name = "Import users"
				description = "Import users from " + connector.Label + " into the data warehouse"
			} else {
				// Destination/FileStorage/Users.
				// Destination/Database/Users.
				name = "Export users"
				description = "Export users to " + connector.Label
			}
			at := PipelineType{
				Name:        name,
				Description: description,
				Target:      TargetUser,
			}
			pipelineTypes = append(pipelineTypes, at)
		case state.SDK, state.Webhook:
			if c.Role == state.Source {
				// Source/SDK/Users.
				// Source/Webhook/Users.
				at := PipelineType{
					Name:        "Import users into warehouse",
					Description: "Import users from " + connector.Label + " into the data warehouse",
					Target:      TargetUser,
				}
				pipelineTypes = append(pipelineTypes, at)
			}
		}
	}
	// TODO(marco): Implement groups
	//if targets.Contains(state.Group) {
	//	switch typ := c.Connector().Type; typ {
	//	case
	//		state.API:
	//		var name, description string
	//		if c.Role == state.Source {
	//			// Source/API/Group.
	//		    name = "Import " + connector.Name + " " + connector.Terms.Groups
	//			description = "Import " + connector.Terms.Groups
	//			if connector.Terms.Groups != "groups" {
	//				description += " as groups"
	//			}
	//			description += " into the data warehouse"
	//		} else {
	//			// Destination/API/Group.
	//			name = "Export " + connector.Terms.Groups
	//			description = "Export groups "
	//			if connector.Terms.Groups != "groups" {
	//				description += " as " + connector.Terms.Groups
	//			}
	//			description += " to " + connector.Name
	//		}
	//		at := PipelineType{
	//			Name:        name,
	//			Description: description,
	//			Target:      Group,
	//		}
	//		pipelineTypes = append(pipelineTypes, at)
	//	case
	//		state.Database,
	//		state.FileStorage:
	//		var name, description string
	//		if c.Role == state.Source {
	//			// Source/FileStorage/Group.
	//			// Source/Database/Group.
	//			name = "Import groups"
	//			description = "Import groups from " + connector.Name + " into the data warehouse"
	//		} else {
	//			// Destination/FileStorage/Group.
	//			// Destination/Database/Group.
	//			name = "Export groups"
	//			description = "Export groups to " + connector.Name
	//		}
	//		at := PipelineType{
	//			Name:        name,
	//			Description: description,
	//			Target:      Group,
	//		}
	//		pipelineTypes = append(pipelineTypes, at)
	//	case state.SDK:
	//		if c.Role == state.Source {
	//			// Source/SDK/Group.
	//			at := PipelineType{
	//				Name:        "Import groups into warehouse",
	//				Description: "Import groups from " + connector.Name + " into the data warehouse",
	//				Target:      Group,
	//			}
	//			pipelineTypes = append(pipelineTypes, at)
	//		}
	//	}
	//}
	if targets.Contains(state.TargetEvent) {
		switch typ := c.Connector().Type; typ {
		case state.SDK, state.Webhook:
			if c.Role == state.Source {
				// Source/SDK/Event.
				// Source/Webhook/Event.
				at := PipelineType{
					Name:        "Import events into warehouse",
					Description: "Import events from " + connector.Label + " into the data warehouse",
					Target:      TargetEvent,
				}
				pipelineTypes = slices.Insert(pipelineTypes, 0, at)
			}
		case state.API:
			if c.Role == state.Destination {
				eventTypes, err := this.api().EventTypes(ctx)
				if err != nil {
					if _, ok := err.(*connections.UnavailableError); ok {
						err = errors.Unavailable("an error occurred fetching the schema: %w", err)
					}
					return nil, err
				}
				// Destination/API/Event.
				for _, et := range eventTypes {
					id := et.ID
					pipelineTypes = append(pipelineTypes, PipelineType{
						Name:        et.Name,
						Description: et.Description,
						Target:      TargetEvent,
						EventType:   &id,
					})
				}
			}
		}
	}
	if pipelineTypes == nil {
		pipelineTypes = []PipelineType{}
	}
	return pipelineTypes, nil
}

// PreviewSendEvent returns a preview of an event as it would be sent to an API.
// The connection must be a destination API connection, and it is expected to
// have an event type with identifier typ. If there is a transformation,
// outSchema is the output schema of the transformation, and it must be a valid.
//
// It returns an errors.UnprocessableError error with code:
//   - EventTypeNotExist, if the event type does not exist for the connection.
//   - InvalidEvent, if the event is not valid.
//   - SchemaNotAligned, if the output schema is not aligned with the event
//     type's schema.
//   - TransformationFailed if the transformation fails due to an error in the
//     executed function.
//   - UnsupportedLanguage, if the transformation language is not supported.
func (this *Connection) PreviewSendEvent(ctx context.Context, typ string, event json.Value, transformation DataTransformation, outSchema types.Type) ([]byte, error) {

	this.core.mustBeOpen()

	c := this.connection

	if c.Connector().Type != state.API {
		return nil, errors.BadRequest("connection %d is not an API connection", c.ID)
	}
	if c.Role != state.Destination {
		return nil, errors.BadRequest("connection %d is not a destination", c.ID)
	}
	if !c.Connector().DestinationTargets.Contains(state.TargetEvent) {
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

	attributes, err := types.Decode[map[string]any](bytes.NewReader(event), schemas.Event)
	if err != nil {
		return nil, errors.BadRequest("event is not valid: %s", err)
	}

	ev := connectors.Event{
		Received: connections.ReceivedEvent(attributes),
		Type: connectors.EventTypeInfo{
			ID:     typ,
			Schema: outSchema,
		},
	}

	if transformation.Mapping != nil || transformation.Function != nil {

		if !outSchema.Valid() {
			return nil, errors.BadRequest("a transformation has been provided but out schema is not valid")
		}
		if outSchema.Kind() != types.ObjectKind {
			return nil, errors.BadRequest("out schema is not an object")
		}

		pipeline := &state.Pipeline{
			InSchema:  schemas.Event,
			OutSchema: outSchema,
			Transformation: state.Transformation{
				Mapping: transformation.Mapping,
			},
		}

		// provider is a temporary function provider.
		var provider transformers.FunctionProvider

		// Validate the mapping and the transformation.
		switch {
		case transformation.Mapping != nil:
			mapping, err := mappings.New(transformation.Mapping, schemas.Event, outSchema, false, nil)
			if err != nil {
				return nil, errors.BadRequest("mapping is not valid: %s", err)
			}
			pipeline.Transformation.InPaths = mapping.InPaths()
			pipeline.Transformation.OutPaths = mapping.OutPaths()
		case transformation.Function != nil:
			if transformation.Function.Source == "" {
				return nil, errors.BadRequest("function source is empty")
			}
			switch transformation.Function.Language {
			case "JavaScript":
				if this.core.functionProvider == nil || !this.core.functionProvider.SupportLanguage(state.JavaScript) {
					return nil, errors.Unprocessable(UnsupportedLanguage, "JavaScript function language is not supported")
				}
			case "Python":
				if this.core.functionProvider == nil || !this.core.functionProvider.SupportLanguage(state.Python) {
					return nil, errors.Unprocessable(UnsupportedLanguage, "Python function language is not supported")
				}
			case "":
				return nil, errors.BadRequest("function language is empty")
			default:
				return nil, errors.BadRequest("function language %q is not valid", transformation.Function.Language)
			}
			pipeline.Transformation.Function = &state.TransformationFunction{
				Source:  transformation.Function.Source,
				Version: "1", // no matter the version, it will be overwritten by the temporary function.
			}
			name := transformationFunctionName(0)
			switch transformation.Function.Language {
			case "JavaScript":
				pipeline.Transformation.Function.Language = state.JavaScript
			case "Python":
				pipeline.Transformation.Function.Language = state.Python
			}
			pipeline.Transformation.Function.PreserveJSON = transformation.Function.PreserveJSON
			// In InPaths and OutPaths, list only top-level property names;
			// there is no need to list sub-property paths (as the behavior is
			// the same).
			pipeline.Transformation.InPaths = pipeline.InSchema.Properties().SortedNames()
			pipeline.Transformation.OutPaths = pipeline.OutSchema.Properties().SortedNames()
			provider = newTempTransformerProvider(name, pipeline.Transformation.Function.Language, pipeline.Transformation.Function.Source, this.core.functionProvider)
		default:
			return nil, errors.BadRequest("transformation mapping or function is required")
		}

		// Transform the attributes.
		transformer, err := transformers.New(pipeline, provider, nil)
		if err != nil {
			return nil, err
		}
		records := []transformers.Record{
			{Purpose: transformers.Create, Attributes: attributes},
		}
		err = transformer.Transform(ctx, records)
		if err != nil {
			if _, ok := err.(transformers.FunctionExecError); ok {
				err = errors.Unprocessable(TransformationFailed, "%s", err)
			}
			return nil, err
		}
		if err = records[0].Err; err != nil {
			return nil, errors.Unprocessable(TransformationFailed, "%s", err)
		}
		ev.Type.Values = records[0].Attributes

	} else {

		if outSchema.Valid() {
			return nil, errors.BadRequest("output schema is a valid schema, but no transformation has been provided")
		}

	}

	// Create a preview before sending the event.
	req, err := this.api().PreviewSendEvent(ctx, ev)
	if err != nil {
		if err == connectors.ErrEventTypeNotExist {
			err = errors.Unprocessable(EventTypeNotExist, "connection %d does not have event type %q", c.ID, typ)
		} else {
			switch err.(type) {
			case *schemas.Error:
				err = errors.Unprocessable(SchemaNotAligned, "output schema is not compatible with the event type's schema: %w", err)
			case *connections.InvalidEventError:
				err = errors.Unprocessable(InvalidEvent, "event is invalid: %w", err)
			case *connections.UnavailableError:
				err = errors.Unavailable("connector returned an error preparing the preview: %w", err)
			}
		}
		return nil, err
	}

	return dumpPreviewEventRequest(req)
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
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "UPDATE connections SET name = $1 WHERE id = $2", n.Name, n.Connection)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("connection %d does not exist", n.Connection)
		}
		return n, nil
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
		return nil, errors.BadRequest("connector %s does not have source settings", connector.Code)
	}
	if c.Role == state.Destination && !connector.HasDestinationSettings {
		return nil, errors.BadRequest("connector %s does not have destination settings", connector.Code)
	}
	ui, err := this.core.connections.ServeConnectionUI(ctx, c, event, settings)
	if err != nil {
		if err == connectors.ErrUIEventNotExist {
			err = errors.Unprocessable(EventNotExist, "UI event %q does not exist for connector %s", event, connector.Code)
		} else {
			switch err.(type) {
			case *connectors.InvalidSettingsError:
				err = errors.Unprocessable(InvalidSettings, "%s", err)
			case *connections.UnavailableError:
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
			return nil, errors.BadRequest("format settings must be provided because format %s has settings", formatConnector.Code)
		}
		if !json.Valid(settings) || !settings.IsObject() {
			return nil, errors.BadRequest("format settings are not a valid JSON Object")
		}
	} else if settings != nil {
		return nil, errors.BadRequest("format settings cannot be provided because format %s has no settings", formatConnector.Code)
	}

	if !formatConnector.HasSheets {
		return nil, errors.BadRequest("format %s does not have sheets", formatConnector.Code)
	}

	sheets, err := this.storage().Sheets(ctx, formatConnector, path, settings, state.Compression(compression))
	if err != nil {
		switch err.(type) {
		case *connectors.InvalidSettingsError:
			err = errors.Unprocessable(InvalidSettings, "%s", err)
		case *connections.UnavailableError:
			err = errors.Unavailable("%w", err)
		}
		return nil, err
	}

	return sheets, nil
}

// TableSchema returns the destination schema of the given table for the
// connection. connection must be a destination database connection, and table
// must be UTF-8 encoded with a length in range [1, MaxTableNameSize].
//
// The method may also return issues that did not prevent it from being
// processed. These issues are reported as a slice of strings.
//
// If the table contains a column with an unsupported type, it returns an
// errors.UnprocessableError error.
func (this *Connection) TableSchema(ctx context.Context, table string) (types.Type, []string, error) {
	this.core.mustBeOpen()
	c := this.connection
	connector := c.Connector()
	if connector.Type != state.Database {
		return types.Type{}, nil, errors.BadRequest("connection %d is not a database", c.ID)
	}
	if c.Role != state.Destination {
		return types.Type{}, nil, errors.BadRequest("database %d is not a destination", c.ID)
	}
	if err := util.ValidateStringField("table name", table, MaxTableNameSize); err != nil {
		return types.Type{}, nil, errors.BadRequest("%s", err)
	}
	database := this.database()
	defer database.Close()
	schema, issues, err := database.Schema(ctx, table, state.Destination)
	if err != nil {
		switch err.(type) {
		case *connections.UnavailableError:
			err = errors.Unavailable("an error occurred fetching the columns: %w", err)
		}
	}
	return schema, issues, err
}

// UnlinkConnection unlinks an SDK or webhook connection from the destination
// connection identified by dst, which must support events. If the connections
// are not linked, the method does nothing.
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
	} else if !c.Connector().SourceTargets.Contains(state.TargetEvent) {
		return errors.BadRequest("source %d does not support events", this.connection.ID)
	}
	// Validate the destination connection.
	ws := this.connection.Workspace()
	if c, ok := ws.Connection(dst); !ok {
		return errors.NotFound("connection %d does not exist", dst)
	} else if c.Role == state.Source {
		return errors.BadRequest("connection %d is not a destination", dst)
	} else if connector := c.Connector(); !connector.DestinationTargets.Contains(state.TargetEvent) {
		return errors.BadRequest("destination %d does not support events", dst)
	}

	n := state.UnlinkConnection{
		Connections: [2]int{this.connection.ID, dst},
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		const remove = "UPDATE connections\n" +
			"SET linked_connections =\n" +
			"\tCASE\n" +
			"\t\tWHEN array_remove(linked_connections, $2) = '{}' THEN NULL\n" +
			"\t\tELSE array_remove(linked_connections, $2)\n" +
			"\tEND\n" +
			"WHERE id = $1"
		result, err := tx.Exec(ctx, remove+" AND $2 = ANY(linked_connections)", n.Connections[0], n.Connections[1])
		if err != nil {
			return nil, err
		}
		// Do nothing if the source connection does not exist or if they are not linked.
		if result.RowsAffected() == 0 {
			return nil, nil
		}
		_, err = tx.Exec(ctx, remove, n.Connections[1], n.Connections[0])
		if err != nil {
			return nil, err
		}
		// No result check is needed because the destination is guaranteed to still exist.
		return n, nil
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

	n := state.UpdateConnection{
		Connection:  this.connection.ID,
		Name:        connection.Name,
		Strategy:    (*state.Strategy)(connection.Strategy),
		SendingMode: (*state.SendingMode)(connection.SendingMode),
	}

	c := this.connection.Connector()

	// Validate the strategy.
	if this.connection.Role == state.Source {
		if c.Strategies {
			if connection.Strategy == nil {
				return errors.BadRequest("%s connections must have a strategy", strings.ToLower(c.Type.String()))
			}
		} else {
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
				return errors.BadRequest("connector %s requires a sending mode", c.Code)
			}
			if !c.SendingMode.Contains(state.SendingMode(*connection.SendingMode)) {
				return errors.BadRequest("connector %s does not support sending mode %s", c.Code, *c.SendingMode)
			}
		} else if connection.SendingMode != nil {
			return errors.BadRequest("connector %s does not support sending modes", c.Code)
		}
	} else if connection.SendingMode != nil {
		return errors.BadRequest("source connections cannot have a sending mode")
	}

	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "UPDATE connections SET name = $1,"+
			" strategy = $2, sending_mode = $3 WHERE id = $4",
			n.Name, n.Strategy, n.SendingMode, n.Connection)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, nil
		}
		return n, nil
	})

	return err
}

// EventWriteKeys returns the event write keys of the connection.
// The connection must be an SDK or webhook source.
func (this *Connection) EventWriteKeys() ([]string, error) {
	this.core.mustBeOpen()
	c := this.connection
	switch c.Connector().Type {
	case state.SDK, state.Webhook:
	default:
		return nil, errors.BadRequest("connection %d is neither an SDK nor a webhook", c.ID)
	}
	if c.Role != state.Source {
		return nil, errors.BadRequest("connection %d is not a source", c.ID)
	}
	return slices.Clone(c.Keys), nil
}

// api returns the API of the connection.
func (this *Connection) api() *connections.API {
	return this.core.connections.API(this.connection)
}

// appSchemas returns the user or group schemas, based on target, for an API
// connection. The connection must support the provided target.
//
// For a source connection, it returns only the group source schema.
// For a destination connection, it returns both the group source and
// destination schemas.
func (this *Connection) appSchemas(ctx context.Context, target state.Target) (src, dst types.Type, err error) {
	c := this.connection
	connector := c.Connector()
	if connector.Type != state.API {
		err = errors.BadRequest("connection %d is not an API", c.ID)
		return
	}
	if !connector.DestinationTargets.Contains(target) {
		err = errors.BadRequest("connection %d does not support %s", c.ID, target)
		return
	}
	api := this.api()
	src, err = api.SchemaAsRole(ctx, state.Source, target, "")
	if err != nil {
		if _, ok := err.(*connections.UnavailableError); ok {
			err = errors.Unavailable("an error occurred fetching the source schema: %w", err)
		}
		return
	}
	if c.Role == state.Destination {
		dst, err = api.SchemaAsRole(ctx, state.Destination, target, "")
		if err != nil {
			if _, ok := err.(*connections.UnavailableError); ok {
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
func (this *Connection) database() *connections.Database {
	return this.core.connections.Database(this.connection)
}

// storage returns the storage of the connection.
func (this *Connection) storage() *connections.FileStorage {
	return this.core.connections.FileStorage(this.connection)
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
	// Perform a formal validation first.
	if target != TargetUser && target != TargetGroup && target != TargetEvent {
		return types.Type{}, errors.BadRequest("target %d is not valid", int(target))
	}
	if eventType != "" && target != TargetEvent {
		return types.Type{}, errors.BadRequest("event type cannot be used with %s target", target)
	}
	// Perform a validation based on the connection's type and role (refer to
	// the specifications in the file "core/Pipelines.csv" for more details).
	c := this.connection
	connector := c.Connector()
	if target == TargetEvent {
		if c.Role == state.Source && eventType != "" {
			return types.Type{}, errors.BadRequest("source connections do not have an event type")
		}
		if c.Role == state.Destination && eventType == "" {
			return types.Type{}, errors.BadRequest("destination connections require an event type")
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
		schema, err := this.api().Schema(ctx, state.Target(target), eventType)
		if err != nil {
			if err == connectors.ErrEventTypeNotExist {
				err = errors.Unprocessable(EventTypeNotExist, "connection %d does not have event type %q", c.ID, eventType)
			} else if _, ok := err.(*connections.UnavailableError); ok {
				err = errors.Unavailable("an error occurred fetching the schema: %w", err)
			}
			return types.Type{}, err
		}
		return schema, nil // schema can be invalid.
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

// dumpPreviewEventRequest dumps the HTTP request used to preview an event, and
// returns it as a byte slice.
func dumpPreviewEventRequest(req *http.Request) ([]byte, error) {

	var b json.Buffer

	b.WriteString(req.Method)
	b.WriteString(" ")
	b.WriteString(req.URL.String())
	b.WriteByte('\n')
	err := req.Header.Write(&b)
	if err != nil {
		return nil, err
	}

	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		b.WriteByte('\n')
		contentEncoding := strings.TrimSpace(req.Header.Get("Content-Encoding"))
		if contentEncoding != "" {
			encodings := strings.Split(contentEncoding, ",")
			if len(encodings) == 1 && strings.EqualFold(strings.TrimSpace(encodings[0]), "gzip") {
				reader, err := gzip.NewReader(bytes.NewReader(body))
				if err == nil {
					decoded, readErr := io.ReadAll(reader)
					closeErr := reader.Close()
					if readErr == nil && closeErr == nil {
						body = decoded
					}
				}
			}
		}
		ct := req.Header.Get("Content-Type")
		if !utf8.Valid(body) {
			ct = "application/octet-stream"
		}
		switch ct {
		case "application/json":
			indented, err := json.Indent(body, "", "  ")
			if err != nil {
				b.Write(body)
				return b.Bytes(), nil
			}
			b.Write(indented)
		case "application/x-ndjson":
			n := b.Len()
			dec := json.NewDecoder(bytes.NewReader(body))
			first := true
			for {
				var value json.Value
				value, err = dec.ReadValue()
				if err != nil {
					if err == io.EOF {
						if !first {
							err = nil
						}
					}
					break
				}
				if !first {
					b.WriteByte('\n')
				}
				indented, err := json.Indent(value, "", "    ")
				if err != nil {
					break
				}
				if err == nil {
					b.Write(indented)
				} else {
					b.Write(value)
				}
				first = false
			}
			if err != nil {
				b.Truncate(n)
				b.Write(body)
			}
		case "application/octet-stream":
			_, _ = fmt.Fprintf(&b, "[A binary body of %d bytes]", len(body))
		default:
			b.Write(body)
		}
	}

	return b.Bytes(), nil
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

// serializeCursor serializes a cursor to be returned by the API.
func serializeCursor(cursor time.Time) (string, error) {
	b, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// validateLinkedConnections checks whether the provided connections can be
// linked to a connection with the specified connector, workspace, and role.
// If they can be linked, it returns the normalized set of connection IDs.
//
// If the connections cannot be linked or unlinked, it returns an
// errors.BadRequestError. If any connection does not exist, it returns an
// errors.UnprocessableError with the code LinkedConnectionNotExist.
func validateLinkedConnections(connections []int, c *state.Connector, ws *state.Workspace, role state.Role) ([]int, error) {
	targets := c.SourceTargets
	if role == state.Destination {
		targets = c.DestinationTargets
	}
	if !targets.Contains(state.TargetEvent) {
		if connections != nil {
			return nil, errors.BadRequest("connector %q, used as %s, does not support events", c.Code, strings.ToLower(role.String()))
		}
		return nil, nil
	}
	if connections == nil {
		connections = []int{}
	}
	if len(connections) == 0 {
		return connections, nil
	}
	for i, id := range connections {
		if id < 1 || id > maxInt32 {
			return nil, errors.BadRequest("event connection %d is not a valid connection identifier", id)
		}
		for j := i + 1; j < len(connections); j++ {
			if connections[j] == id {
				return nil, errors.BadRequest("event connection %d is repeated", id)
			}
		}
		ec, ok := ws.Connection(id)
		if !ok {
			return nil, errors.Unprocessable(LinkedConnectionNotExist, "linked connection %d does not exist", id)
		}
		if role == state.Source {
			// If the connector is Source, the connection's connector must
			// support events as Destination.
			if !ec.Connector().DestinationTargets.Contains(state.TargetEvent) {
				return nil, errors.BadRequest("event connection %d does not support events", id)
			}
		} else {
			// If the connector is Destination, the connection's connector must
			// support events as Source.
			if !ec.Connector().SourceTargets.Contains(state.TargetEvent) {
				return nil, errors.BadRequest("event connection %d does not support events", id)
			}
		}
		if ec.Role == role {
			if ec.Role == state.Source {
				return nil, errors.BadRequest("event connection %d is not a destination", id)
			}
			return nil, errors.BadRequest("event connection %d is not a source", id)
		}
	}
	slices.Sort(connections)
	return connections, nil
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
	// non-anonymous users. It can only be provided for Source SDK connections
	// whose connector supports the strategies.
	Strategy *Strategy `json:"strategy"`

	// SendingMode is the mode used for sending events. It can only be provided for
	// destination API connections that support it. In this case, it must be one of
	// the sending modes supported by the API.
	SendingMode *SendingMode `json:"sendingMode"`
}

// tempFunctionProvider is a function provider that creates a function at each
// call and deletes it after the call returns. Any call to a method that is not
// CallFunction panics.
type tempFunctionProvider struct {
	name     string                        // function name.
	language state.Language                // language.
	source   string                        // source code.
	provider transformers.FunctionProvider // underlying function provider.
}

func newTempTransformerProvider(name string, language state.Language, source string, provider transformers.FunctionProvider) *tempFunctionProvider {
	return &tempFunctionProvider{name, language, source, provider}
}

func (tp *tempFunctionProvider) Call(ctx context.Context, _, _ string, inSchema, outSchema types.Type, preserveJSON bool, records []transformers.Record) error {
	id, version, err := tp.provider.Create(ctx, tp.name, tp.language, tp.source)
	if err != nil {
		return err
	}
	defer func() {
		go func() {
			err := tp.provider.Delete(context.Background(), id)
			if err != nil {
				slog.Warn("core: cannot delete transformation function", "id", id, "err", err)
			}
		}()
	}()
	return tp.provider.Call(ctx, id, version, inSchema, outSchema, preserveJSON, records)
}

func (tp *tempFunctionProvider) Close(_ context.Context) error { panic("not supported") }
func (tp *tempFunctionProvider) Create(_ context.Context, _ string, _ state.Language, _ string) (string, string, error) {
	panic("not supported")
}
func (tp *tempFunctionProvider) Delete(_ context.Context, _ string) error {
	panic("not supported")
}
func (tp *tempFunctionProvider) SupportLanguage(_ state.Language) bool {
	panic("not supported")
}
func (tp *tempFunctionProvider) Update(_ context.Context, _, _ string) (string, error) {
	panic("not supported")
}
