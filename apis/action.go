//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"chichi/apis/connectors"
	"chichi/apis/datastore"
	"chichi/apis/errors"
	"chichi/apis/state"
	"chichi/apis/transformers"
	"chichi/connector/types"
	"chichi/telemetry"
)

var (
	MappingOverAnonymousIdentifier errors.Code = "MappingOverAnonymousIdentifier"
	DatabaseFailed                 errors.Code = "DatabaseFailed"
)

// Action represents an action associated to a destination connection to send
// events.
type Action struct {
	apis               *APIs
	action             *state.Action
	connection         *Connection
	ID                 int
	Connection         int
	Target             Target
	Name               string
	Enabled            bool
	EventType          *string
	Running            bool
	ScheduleStart      *int
	SchedulePeriod     *SchedulePeriod
	InSchema           types.Type
	OutSchema          types.Type
	Filter             *Filter
	Transformation     Transformation
	Query              *string
	Path               *string
	Table              *string
	Sheet              *string
	IdentityColumn     *string
	TimestampColumn    *string
	TimestampFormat    *string
	ExportMode         *ExportMode
	MatchingProperties *MatchingProperties
}

// Language represents a transformation language. Valid values are "JavaScript"
// and "Python".
type Language string

// TransformationFunction represents a transformation function.
type TransformationFunction struct {
	Source   string
	Language Language
}

// Transformation represents a transformation.
type Transformation struct {
	Mapping  map[string]string
	Function *TransformationFunction
}

// ExportMode represents one of the three export modes.
type ExportMode string

const (
	CreateOnly     ExportMode = "CreateOnly"
	UpdateOnly     ExportMode = "UpdateOnly"
	CreateOrUpdate ExportMode = "CreateOrUpdate"
)

// fromState serializes action into this.
func (this *Action) fromState(apis *APIs, store *datastore.Store, action *state.Action) {
	c := action.Connection()
	this.apis = apis
	this.action = action
	this.connection = &Connection{apis: apis, store: store, connection: c}
	this.ID = action.ID
	this.Connection = c.ID
	this.Target = Target(action.Target)
	this.Name = action.Name
	this.Enabled = action.Enabled
	if action.EventType != "" {
		et := action.EventType
		this.EventType = &et
	}
	_, this.Running = this.action.Execution()
	if action.Target == state.Users || action.Target == state.Groups {
		start := int(action.ScheduleStart)
		period := SchedulePeriod(action.SchedulePeriod)
		this.ScheduleStart = &start
		this.SchedulePeriod = &period
	}
	this.InSchema = action.InSchema
	this.OutSchema = action.OutSchema
	if action.Filter != nil {
		this.Filter = &Filter{
			Logical:    FilterLogical(action.Filter.Logical),
			Conditions: make([]FilterCondition, len(action.Filter.Conditions)),
		}
		for i, condition := range action.Filter.Conditions {
			this.Filter.Conditions[i] = FilterCondition(condition)
		}
	}
	if mapping := action.Transformation.Mapping; mapping != nil {
		this.Transformation.Mapping = make(map[string]string, len(mapping))
		for out, in := range mapping {
			this.Transformation.Mapping[out] = in
		}
	}
	if function := action.Transformation.Function; function != nil {
		this.Transformation.Function = &TransformationFunction{
			Source:   function.Source,
			Language: Language(function.Language.String()),
		}
	}
	if action.Query != "" {
		query := action.Query
		this.Query = &query
	}
	if action.Path != "" {
		path := action.Path
		this.Path = &path
	}
	if action.TableName != "" {
		table := action.TableName
		this.Table = &table
	}
	if action.Sheet != "" {
		sheet := action.Sheet
		this.Sheet = &sheet
	}
	if action.IdentityColumn != "" {
		column := action.IdentityColumn
		this.IdentityColumn = &column
	}
	if action.TimestampColumn != "" {
		column := action.TimestampColumn
		this.TimestampColumn = &column
	}
	if action.TimestampFormat != "" {
		format := action.TimestampFormat
		this.TimestampFormat = &format
	}
	this.ExportMode = (*ExportMode)(action.ExportMode)
	if props := action.MatchingProperties; props != nil {
		this.MatchingProperties = &MatchingProperties{
			Internal: props.Internal,
			External: props.External,
		}
	}
}

// Target represents a target.
type Target int

const (
	Events Target = iota + 1
	Users
	Groups
)

// MarshalJSON implements the json.Marshaler interface.
func (at Target) MarshalJSON() ([]byte, error) {
	return []byte(`"` + at.String() + `"`), nil
}

func (at Target) String() string {
	switch at {
	case Events:
		return "Events"
	case Users:
		return "Users"
	case Groups:
		return "Groups"
	default:
		panic("invalid target")
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (at *Target) UnmarshalJSON(data []byte) error {
	var v any
	err := json.Unmarshal(data, &v)
	if err != nil {
		return fmt.Errorf("json: cannot unmarshal into a apis.Target value: %s", err)
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an api.Target value", v)
	}
	switch s {
	case "Events":
		*at = Events
	case "Users":
		*at = Users
	case "Groups":
		*at = Groups
	default:
		return fmt.Errorf("invalid apis.Target: %s", s)
	}
	return nil
}

// Delete deletes the action.
// It returns an errors.NotFoundError error if the action does not exist
// anymore.
func (this *Action) Delete(ctx context.Context) error {
	this.apis.mustBeOpen()
	n := state.DeleteAction{
		ID: this.action.ID,
	}
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "DELETE FROM actions WHERE id = $1", n.ID)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.NotFound("action %d does not exist", n.ID)
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// Execute executes the action.
//
// It returns an errors.NotFoundError error if the action does not exist
// anymore.
// It returns an errors.UnprocessableError error with code
//   - ExecutionInProgress, if the action is already in progress.
//   - NoStorage, if the connection of the action is a file and has no storage.
//   - NoWarehouse, if the workspace does not have a data warehouse.
func (this *Action) Execute(ctx context.Context, reimport bool) error {
	this.apis.mustBeOpen()
	ctx, span := telemetry.TraceSpan(ctx, "Action.Execute", "id", this.action.ID, "reimport", reimport)
	defer span.End()
	if _, ok := this.action.Execution(); ok {
		return errors.Unprocessable(ExecutionInProgress, "action %d is already in progress", this.action.ID)
	}
	if this.connection.store == nil {
		ws := this.action.Connection().Workspace()
		return errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", ws.ID)
	}
	if t := this.action.Target; t != state.Users && t != state.Groups {
		return errors.BadRequest("action %d with target %s cannot be executed", this.action.ID, t)
	}
	c := this.action.Connection()
	if c.Connector().Type == state.FileType {
		if _, ok := c.Storage(); !ok {
			return errors.Unprocessable(NoStorage, "file connection %d does not have a storage", c.ID)
		}
	}
	return this.addExecution(ctx, reimport)
}

// Set sets the action.
//
// Refer to the specifications in the file "connector/Actions support.md" for
// more details.
//
// It returns an errors.UnprocessableError error with code
//   - EventTypeNotExist, if the event type does not exist anymore for the
//     connection.
//   - FetchSchemaFailed, if an error occurred fetching the event type schema.
//   - LanguageNotSupported, if the transformation language is not supported.
//   - MappingOverAnonymousIdentifier, if the action maps over an anonymous
//     identifier.
func (this *Action) Set(ctx context.Context, action ActionToSet) error {

	this.apis.mustBeOpen()
	ctx, span := telemetry.TraceSpan(ctx, "Action.Set", "action", this.action.ID)
	defer span.End()

	c := this.action.Connection()

	// Validate the action.
	var eventTypeSchema types.Type
	var err error
	if this.action.EventType != "" {
		eventTypeSchema, err = this.app().Schema(ctx, state.Events, this.action.EventType)
		if err != nil {
			if err == connectors.ErrEventTypeNotExist {
				return errors.Unprocessable(EventTypeNotExist, "connection %d no longer has the event type %q", c.ID, this.action.EventType)
			}
			return errors.Unprocessable(FetchSchemaFailed, "an error occurred fetching the event type schema: %w", err)
		}
	}
	err = this.connection.validateActionToSet(action, this.action.Target, eventTypeSchema)
	if err != nil {
		return err
	}

	span.Log("action validated successfully")

	n := state.SetAction{
		ID:        this.action.ID,
		Name:      action.Name,
		Enabled:   action.Enabled,
		InSchema:  action.InSchema,
		OutSchema: action.OutSchema,
		Transformation: state.Transformation{
			Mapping: action.Transformation.Mapping,
		},
		Query:           action.Query,
		Path:            action.Path,
		TableName:       action.TableName,
		Sheet:           action.Sheet,
		IdentityColumn:  action.IdentityColumn,
		TimestampColumn: action.TimestampColumn,
		TimestampFormat: action.TimestampFormat,
		ExportMode:      (*state.ExportMode)(action.ExportMode),
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
			return err
		}
	}

	if props := action.MatchingProperties; props != nil {
		n.MatchingProperties = &state.MatchingProperties{
			Internal: props.Internal,
			External: props.External,
		}
	}

	// Marshal the input and the output schemas.
	rawInSchema, err := marshalSchema(action.InSchema)
	if err != nil {
		return err
	}
	rawOutSchema, err := marshalSchema(action.OutSchema)
	if err != nil {
		return err
	}

	// Marshal the mapping.
	var mapping []byte
	if action.Transformation.Mapping != nil {
		mapping, err = json.Marshal(action.Transformation.Mapping)
		if err != nil {
			return err
		}
	}

	// Matching properties.
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

	// Transformation.
	if fn := n.Transformation.Function; fn != nil {
		if this.action.Transformation.Function == nil {
			name := transformationFunctionName(n.ID, fn.Language)
			version, err := this.apis.functionTransformer.Create(ctx, name, fn.Source)
			if err == transformers.ErrFunctionExist {
				version, err = this.apis.functionTransformer.Update(ctx, name, fn.Source)
			}
			if err != nil {
				return err
			}
			n.Transformation.Function.Version = version
		} else if this.action.Transformation.Function.Source != fn.Source || this.action.Transformation.Function.Language != fn.Language {
			name := transformationFunctionName(n.ID, fn.Language)
			version, err := this.apis.functionTransformer.Update(ctx, name, fn.Source)
			if err == transformers.ErrFunctionNotExist {
				version, err = this.apis.functionTransformer.Create(ctx, name, fn.Source)
			}
			if err != nil {
				return err
			}
			n.Transformation.Function.Version = version
		} else {
			// The function's source code and language should not be changed.
			// It will be verified during the transaction and assigned the current version.
		}
	}

	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		var function state.TransformationFunction
		if n.Transformation.Function != nil {
			var current state.TransformationFunction
			if n.Transformation.Function.Version == "" {
				err := tx.QueryRow(ctx, "SELECT transformation_source, transformation_language, transformation_version "+
					"FROM actions WHERE id = $1", n.ID).Scan(&current.Source, &current.Language, &current.Version)
				if err != nil {
					return err
				}
				if current.Source != n.Transformation.Function.Source || current.Language != n.Transformation.Function.Language {
					return fmt.Errorf("abort update action %d: it was optimistically assumed that the transformation"+
						" had not changed, but it has indeed changed", n.ID)
				}
				n.Transformation.Function.Version = current.Version
			}
			function = *n.Transformation.Function
		}
		result, err := tx.Exec(ctx, "UPDATE actions SET\n"+
			"name = $1, enabled = $2, in_schema = $3, out_schema = $4, filter = $5, transformation_mapping = $6, "+
			"transformation_source = $7, transformation_language = $8, transformation_version = $9, "+
			"query = $10, path = $11, table_name = $12, sheet = $13, identity_column = $14, "+
			"timestamp_column = $15, timestamp_format = $16, export_mode = $17, "+
			"matching_properties_internal = $18, matching_properties_external = $19\nWHERE id = $20",
			n.Name, n.Enabled, rawInSchema, rawOutSchema, string(filter), mapping, function.Source,
			function.Language, function.Version, n.Query, n.Path, n.TableName, n.Sheet,
			n.IdentityColumn, n.TimestampColumn, n.TimestampFormat, n.ExportMode, string(matchPropInternal),
			string(matchPropExternal), n.ID,
		)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return nil
		}
		return tx.Notify(ctx, n)
	})
	span.Log("action set successfully", "id", this.action.ID)

	return err
}

// setUserCursor sets the user cursor of the action.
func (this *Action) setUserCursor(ctx context.Context, cursor state.Cursor) error {
	n := state.SetActionUserCursor{
		ID:         this.action.ID,
		UserCursor: cursor,
	}
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE actions\n"+
			"SET user_cursor.id = $1, user_cursor.timestamp = $2 WHERE id = $3",
			n.UserCursor.ID, n.UserCursor.Timestamp, n.ID)
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

// SetSchedulePeriod sets the schedule period, in minutes, of the action. The
// action must be a Users or Groups action and period can be 5, 15, 30, 60, 120,
// 180, 360, 480, 720, or 1440.
func (this *Action) SetSchedulePeriod(ctx context.Context, period SchedulePeriod) error {
	this.apis.mustBeOpen()
	switch this.action.Target {
	case state.Users, state.Groups:
	default:
		return errors.BadRequest("cannot set schedule period of a %s action", this.action.Target)
	}
	switch period {
	case 5, 15, 30, 60, 120, 180, 360, 480, 720, 1440:
	default:
		return errors.BadRequest("schedule period %d is not valid", period)
	}
	n := state.SetActionSchedulePeriod{
		ID:             this.action.ID,
		SchedulePeriod: int16(period),
	}
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE actions SET schedule_period = $1 WHERE id = $2 AND schedule_period <> $1", n.SchedulePeriod, n.ID)
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

// SetStatus sets the status of the action.
func (this *Action) SetStatus(ctx context.Context, enabled bool) error {
	this.apis.mustBeOpen()
	if enabled == this.action.Enabled {
		return nil
	}
	n := state.SetActionStatus{
		ID:      this.action.ID,
		Enabled: enabled,
	}
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE actions SET enabled = $1 WHERE id = $2 AND enabled <> $1", n.Enabled, n.ID)
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

// app returns the app of the action.
func (this *Action) app() *connectors.App {
	return this.apis.connectors.App(this.action.Connection())
}

// database returns the database of the action.
// The caller must call the database's Close method when the database is no
// longer needed.
func (this *Action) database() *connectors.Database {
	return this.apis.connectors.Database(this.action.Connection())
}

// isLanguageSupported reports whether the transformation language of the action
// is supported. If the action does not have a transformation, it returns true.
func (this *Action) isLanguageSupported() bool {
	transformation := this.action.Transformation.Function
	if transformation == nil {
		return true
	}
	if this.apis.functionTransformer != nil && this.apis.functionTransformer.SupportLanguage(transformation.Language) {
		return true
	}
	return false
}

// file returns the file of the action.
func (this *Action) file() *connectors.File {
	return this.apis.connectors.File(this.action.Connection())
}

// ActionToSet represents an action to set in a connection, by adding a new
// action (using the method Connection.AddAction) or updating an existing one
// (using the method Action.Set).
//
// Refer to the specifications in the file "connector/Actions support.md" for
// more details.
type ActionToSet struct {

	// Name must be a non-empty valid UTF-8 encoded string and cannot be longer
	// than 60 runes.
	Name string

	// Enabled indicates whether the action is enabled or not.
	Enabled bool

	// Filter is the filter of the action, if it has one, otherwise is nil.
	Filter *Filter

	// InSchema is the input schema of the action, which may contain the
	// properties used in the transformation and the internal matching property.
	InSchema types.Type

	// OutSchema is the output schema of the action, which may contain the
	// properties used in the transformation.
	OutSchema types.Type

	// Transformation is the mapping or function transformation, if it has one.
	//
	// Every action that supports transformations must have an associated
	// mapping or function, which are mutually exclusive.
	//
	// If it has a mapping, the names of the properties in which the values are
	// mapped (the keys of the map) must be present in the output schema of the
	// action, while the values of the map must be valid mapping expressions.
	Transformation Transformation

	// Query is the query of the action, if it has one, otherwise it is the
	// empty string.
	Query string

	// Path is the path of the file. It cannot be longer than 1024 runes,
	// and it is empty for non-file actions.
	Path string

	// TableName is the name of the table for the export and it is defined for
	// destination database-actions; in any other case, it is the empty string.
	// It cannot be longer than 1024 runes.
	TableName string

	// Sheet is the sheet name for multiple sheets file actions. It must be UTF-8
	// encoded, have a length in the range [1, 31], should not start or end with
	// "'", and cannot contain any of "*", "/", ":", "?", "[", "\", and "]". It is
	// empty for non-file and non-multipart sheets actions. Sheet names are
	// case-insensitive.
	Sheet string

	// IdentityColumn is the column name used as identity in source file
	// connections.
	// It cannot be longer than 1024 runes.
	IdentityColumn string

	// TimestampColumn is the column name used as timestamp in source file
	// connections. May be empty to indicate that no properties should be used as
	// timestamp.
	// When not empty, requires a TimestampFormat.
	// It cannot be longer than 1024 runes.
	TimestampColumn string

	// TimestampFormat indicates the timestamp format for parsing the timestamp.
	//
	// Represents a format when a TimestampColumn is provided and its
	// corresponding property kind is JSON or Text, otherwise it is the empty
	// string.
	//
	// In case it is provided, accepted values are:
	//
	//   - "ISO8601", to parse timestamps as a ISO 8601 timestamps.
	//   - "Excel", to parse timestamps as strings representing a float value
	//     stored in a Excel cell representing a date / datetime.
	//   - a strptime format, enclosed by single quote characters, compatible
	//     with the standard C89 functions strptime/strftime.
	//
	// It cannot be longer than 64 runes.
	TimestampFormat string

	// ExportMode is the export mode, if it has one.
	ExportMode *ExportMode

	// MatchingProperties are the internal and external properties used for matching
	// users during export to apps.
	MatchingProperties *MatchingProperties
}

// MatchingProperties contains an internal property (belonging to the Golden
// Record) and an external property (belonging to the app) which are used to
// match identities of users in the data warehouse with users on the external
// app, during export.
type MatchingProperties struct {
	Internal string // the corresponding property is stored within the action's input schema.
	External types.Property
}

// SchedulePeriod represents a scheduler period in minutes.
// Valid values are 5, 15, 30, 60, 120, 180, 360, 480, 720, and 1440.
type SchedulePeriod int

// MarshalJSON implements the json.Marshaler interface.
// It panics if period is not a valid SchedulePeriod value.
func (period SchedulePeriod) MarshalJSON() ([]byte, error) {
	return []byte(`"` + period.String() + `"`), nil
}

// String returns the string representation of period.
// It panics if period is not a valid SchedulePeriod value.
func (period SchedulePeriod) String() string {
	switch period {
	case 5:
		return "5m"
	case 15:
		return "15m"
	case 30:
		return "30m"
	case 60:
		return "1h"
	case 120:
		return "2h"
	case 180:
		return "3h"
	case 360:
		return "6h"
	case 480:
		return "8h"
	case 720:
		return "12h"
	case 1440:
		return "24h"
	}
	panic("invalid schedule period")
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (period *SchedulePeriod) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, null) {
		return nil
	}
	var v any
	err := json.Unmarshal(data, &v)
	if err != nil {
		return fmt.Errorf("json: cannot unmarshal into an apis.SchedulePeriod value: %s", err)
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an api.SchedulePeriod value", v)
	}
	var p SchedulePeriod
	switch s {
	case "5m":
		p = 5
	case "15m":
		p = 15
	case "30m":
		p = 30
	case "1h":
		p = 60
	case "2h":
		p = 120
	case "3h":
		p = 180
	case "6h":
		p = 360
	case "8h":
		p = 480
	case "12h":
		p = 720
	case "24h":
		p = 1440
	default:
		return fmt.Errorf("invalid apis.SchedulePeriod: %s", s)
	}
	*period = p
	return nil
}

// transformationFunctionName returns the name the transformation function for
// an action in the specified language.
//
// Keep in sync with the function having the same name in the mappings package.
func transformationFunctionName(action int, language state.Language) string {
	var ext string
	switch language {
	case state.JavaScript:
		ext = ".js"
	case state.Python:
		ext = ".py"
	default:
		panic("unexpected language")
	}
	return "action-" + strconv.Itoa(action) + ext
}
