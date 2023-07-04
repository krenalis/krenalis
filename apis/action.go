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
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"chichi/apis/errors"
	"chichi/apis/httpclient"
	"chichi/apis/mappings/mapexp"
	"chichi/apis/postgres"
	"chichi/apis/state"
	_connector "chichi/connector"
	"chichi/connector/types"

	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/maps"
)

var QueryExecutionFailed errors.Code = "QueryExecutionFailed"

// Action represents an action associated to a destination connection to send
// events.
type Action struct {
	db                 *postgres.DB
	redis              *redis.Client
	action             *state.Action
	connection         *Connection
	ID                 int
	Connection         int
	Target             ActionTarget
	Name               string
	Enabled            bool
	EventType          *string
	Running            bool
	ScheduleStart      *int
	SchedulePeriod     *SchedulePeriod
	Filter             *ActionFilter
	InSchema           types.Type
	OutSchema          types.Type
	Mapping            map[string]string
	PythonSource       string
	Query              *string
	Path               *string
	Table              *string
	Sheet              *string
	ExportMode         *ExportMode
	MatchingProperties *MatchingProperties
}

// ExportMode represents one of the three export modes.
type ExportMode string

const (
	CreateOnly     ExportMode = "CreateOnly"
	UpdateOnly     ExportMode = "UpdateOnly"
	CreateOrUpdate ExportMode = "CreateOrUpdate"
)

// fromState serializes action into this.
func (this *Action) fromState(db *postgres.DB, redis *redis.Client, http *httpclient.HTTP, action *state.Action) {
	c := action.Connection()
	this.db = db
	this.action = action
	this.redis = redis
	this.connection = &Connection{db: db, redis: redis, connection: c, http: http}
	this.ID = action.ID
	this.Connection = c.ID
	this.Target = ActionTarget(action.Target)
	this.Name = action.Name
	this.Enabled = action.Enabled
	if action.EventType != "" {
		et := action.EventType
		this.EventType = &et
	}
	_, this.Running = this.action.Execution()
	if action.Target == state.UsersTarget || action.Target == state.GroupsTarget {
		start := int(action.ScheduleStart)
		period := SchedulePeriod(action.SchedulePeriod)
		this.ScheduleStart = &start
		this.SchedulePeriod = &period
	}
	if action.Filter != nil {
		this.Filter = &ActionFilter{
			Logical:    ActionFilterLogical(action.Filter.Logical),
			Conditions: make([]ActionFilterCondition, len(action.Filter.Conditions)),
		}
		for i, condition := range action.Filter.Conditions {
			this.Filter.Conditions[i] = ActionFilterCondition(condition)
		}
	}
	this.InSchema = action.InSchema
	this.OutSchema = action.OutSchema
	if action.Mapping != nil {
		this.Mapping = make(map[string]string, len(action.Mapping))
		for out, in := range action.Mapping {
			this.Mapping[out] = in
		}
	}
	this.PythonSource = action.PythonSource
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
	this.ExportMode = (*ExportMode)(action.ExportMode)
	if props := action.MatchingProperties; props != nil {
		this.MatchingProperties = &MatchingProperties{
			Internal: props.Internal,
			External: props.External,
		}
	}
}

// ActionTarget represents the target of an action.
type ActionTarget int

const (
	EventsTarget ActionTarget = iota + 1
	UsersTarget
	GroupsTarget
)

// MarshalJSON implements the json.Marshaler interface.
func (at ActionTarget) MarshalJSON() ([]byte, error) {
	return []byte(`"` + at.String() + `"`), nil
}

func (at ActionTarget) String() string {
	switch at {
	case EventsTarget:
		return "Events"
	case UsersTarget:
		return "Users"
	case GroupsTarget:
		return "Groups"
	default:
		panic("invalid action target")
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (at *ActionTarget) UnmarshalJSON(data []byte) error {
	var v any
	err := json.Unmarshal(data, &v)
	if err != nil {
		return fmt.Errorf("json: cannot unmarshal into a apis.ActionTarget value: %s", err)
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an api.ActionTarget value", v)
	}
	switch s {
	case "Events":
		*at = EventsTarget
	case "Users":
		*at = UsersTarget
	case "Groups":
		*at = GroupsTarget
	default:
		return fmt.Errorf("invalid apis.ActionTarget: %s", s)
	}
	return nil
}

// ActionFilter represents a filter of an action.
type ActionFilter struct {
	Logical    ActionFilterLogical     // can be "all" or "any".
	Conditions []ActionFilterCondition // cannot be empty.
}

// ActionFilterLogical represents the logical operator of an action filter.
// It can be "all" or "any".
type ActionFilterLogical string

// ActionFilterCondition represents the condition of an action filter.
type ActionFilterCondition struct {
	Property string // A property identifier or selector (e.g. "street1" or "traits.address.street1").
	Operator string // "is", "is not".
	Value    string // "Track", "Page", ...
}

// Delete deletes the action.
// It returns an errors.NotFoundError error if the action does not exist
// anymore.
func (this *Action) Delete() error {
	n := state.DeleteActionNotification{
		Connection: this.action.Connection().ID,
		ID:         this.action.ID,
	}
	ctx := context.Background()
	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
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
func (this *Action) Execute(reimport bool) error {
	if _, ok := this.action.Execution(); ok {
		return errors.Unprocessable(ExecutionInProgress, "action %d is already in progress", this.action.ID)
	}
	if t := this.action.Target; t != state.UsersTarget && t != state.GroupsTarget {
		return errors.BadRequest("action %d with target %s cannot be executed", this.action.ID, t)
	}
	c := this.action.Connection()
	if c.Connector().Type == state.FileType {
		if _, ok := c.Storage(); !ok {
			return errors.Unprocessable(NoStorage, "file connection %d does not have a storage", c.ID)
		}
	}
	return this.addExecution(reimport)
}

// Set sets action.
//
// Refer to the specifications in the file "connector/Actions support.md" for
// more details.
func (this *Action) Set(action ActionToSet) error {
	err := this.connection.validateActionToSet(action, this.action.Target, this.action.EventType)
	if err != nil {
		return err
	}
	n := state.SetActionNotification{
		ID:           this.action.ID,
		Name:         action.Name,
		Enabled:      action.Enabled,
		InSchema:     action.InSchema,
		OutSchema:    action.OutSchema,
		Mapping:      action.Mapping,
		PythonSource: action.PythonSource,
		Query:        action.Query,
		Path:         action.Path,
		TableName:    action.TableName,
		Sheet:        action.Sheet,
		ExportMode:   (*state.ExportMode)(action.ExportMode),
	}
	var filter, mapping []byte
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
			return err
		}
	}

	if props := action.MatchingProperties; props != nil {
		n.MatchingProperties = &state.MatchingProperties{
			Internal: props.Internal,
			External: props.External,
		}
	}
	ctx := context.Background()

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
	if action.Mapping != nil {
		mapping, err = json.Marshal(action.Mapping)
		if err != nil {
			return err
		}
	}

	// Matching properties.
	var matchPropInternal, matchPropExternal string
	if n.MatchingProperties != nil {
		matchPropInternal = n.MatchingProperties.Internal
		matchPropExternal = n.MatchingProperties.External
	}
	err = this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE actions SET\n"+
			"name = $1, enabled = $2, filter = $3, in_schema = $4, out_schema = $5,\n"+
			"mapping = $6, python_source = $7, query = $8, path = $9, table_name = $10, sheet = $11,\n"+
			"export_mode = $12, matching_properties_internal = $13,\n"+
			"matching_properties_external = $14 WHERE id = $15",
			n.Name, n.Enabled, string(filter), rawInSchema, rawOutSchema, mapping,
			n.PythonSource, n.Query, n.Path, n.TableName, n.Sheet, n.ExportMode, matchPropInternal,
			matchPropExternal, n.ID,
		)
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

// setUserCursor sets the user cursor of the action.
func (this *Action) setUserCursor(ctx context.Context, cursor _connector.Cursor) error {
	n := state.SetActionUserCursorNotification{
		ID:         this.action.ID,
		UserCursor: cursor,
	}
	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE actions\n"+
			"SET user_cursor.id = $1, user_cursor.timestamp = $2, user_cursor.next = $3 WHERE id = $4",
			n.UserCursor.ID, n.UserCursor.Timestamp, n.UserCursor.Next, n.ID)
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
func (this *Action) SetSchedulePeriod(period SchedulePeriod) error {
	switch this.action.Target {
	case state.UsersTarget, state.GroupsTarget:
	default:
		return errors.BadRequest("cannot set schedule period of a %s action", this.action.Target)
	}
	switch period {
	case 5, 15, 30, 60, 120, 180, 360, 480, 720, 1440:
	default:
		return errors.BadRequest("schedule period %d is not valid", period)
	}
	n := state.SetActionSchedulePeriodNotification{
		ID:             this.action.ID,
		SchedulePeriod: int16(period),
	}
	ctx := context.Background()
	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
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
func (this *Action) SetStatus(enabled bool) error {
	if enabled == this.action.Enabled {
		return nil
	}
	n := state.SetActionStatusNotification{
		ID:      this.action.ID,
		Enabled: enabled,
	}
	ctx := context.Background()
	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
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

// ActionToSet represents an action to set to a connection, by adding a new
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
	Filter *ActionFilter

	// InSchema is the input schema of the mappings (of the transformation).
	InSchema types.Type

	// OutSchema is the output schema of the mappings (of the transformation).
	OutSchema types.Type

	// Mapping is the mapping of the action, if it has one, otherwise is nil.
	//
	// Every action that supports mappings / transformation must have an
	// associated mapping or a transformation, which are mutually exclusive.
	//
	// If it has a mapping, the names of the properties in which the values are
	// mapped (the keys of the map) must be present in the output schema of the
	// action, while the values of the map must be valid mapping expressions.
	Mapping map[string]string

	// PythonSource is the source code for the Python transformation function of
	// the action, if it has one, otherwise is the empty string.
	PythonSource string

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

	// Sheet if the sheet name for multiple sheets file actions. It cannot
	// be longer than 100 runes, and it is empty for non-file and non-multipart
	// sheets actions.
	Sheet string

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
	Internal string
	External string
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

// validateActionToSet validates the action to set (when adding or setting an
// action) for the given target and event type.
//
// Refer to the specifications in the file "connector/Actions support.md" for
// more details.
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
	// Validate the filter.
	var conditionProperties []types.Path
	if action.Filter != nil {
		if l := action.Filter.Logical; l != "all" && l != "any" {
			return errors.BadRequest("filter logical operator %q is not valid", action.Filter.Logical)
		}
		if len(action.Filter.Conditions) == 0 {
			return errors.BadRequest("filter does not contain conditions")
		}
		conditionProperties = make([]types.Path, len(action.Filter.Conditions))
		for i, condition := range action.Filter.Conditions {
			property, ok := parsePropertyPath(condition.Property)
			if !ok {
				return errors.BadRequest("filter condition property expression %q is not valid", condition.Property)
			}
			conditionProperties[i] = property
			if op := condition.Operator; op != "is" && op != "is not" {
				return errors.BadRequest("filter condition operator %q is not valid", op)
			}
			if !utf8.ValidString(condition.Value) {
				return errors.BadRequest("filter condition value is not UTF-8 encoded")
			}
			if n := utf8.RuneCountInString(condition.Value); n > 60 {
				return errors.BadRequest("filter condition value is longer than 60 runes")
			}
		}
	}
	// Validate the mapping and the transformation.
	if action.Mapping != nil && action.PythonSource != "" {
		return errors.BadRequest("action can not have both mapping and transformation")
	}
	if action.PythonSource != "" {
		if !strings.Contains(action.PythonSource, "def transform") {
			return errors.BadRequest("Python source code of transformation does not contain 'transform' function")
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

	// Second, do validations based on the connection.

	c := this.connection
	connector := c.Connector()

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

	// Check if the mapping (or the transformation) is mandatory.
	var mappingIsMandatory bool
	switch connector.Type {
	case state.AppType:
		mappingIsMandatory = targetUsersOrGroups
	case
		state.DatabaseType,
		state.FileType:
		mappingIsMandatory = c.Role == state.SourceRole && targetUsersOrGroups
	}
	if mappingIsMandatory && action.Mapping == nil && action.PythonSource == "" {
		return errors.BadRequest("mapping (or transformation) is required")
	}

	// If there is at least one property mapped, or a Python transformation
	// function is provided, then there must be both a valid input and output
	// schema.
	if requiresSchemas := len(action.Mapping) > 0 || action.PythonSource != ""; requiresSchemas {
		if !action.InSchema.Valid() {
			return errors.BadRequest("input schema must be valid")
		}
		if action.InSchema.PhysicalType() != types.PtObject {
			return errors.BadRequest("input schema must have physical type Object")
		}
		if !action.OutSchema.Valid() {
			return errors.BadRequest("output schema must be valid")
		}
		if action.OutSchema.PhysicalType() != types.PtObject {
			return errors.BadRequest("output schema must have physical type Object")
		}
		// In case of mappings, validate the mapped properties and ensure that
		// every property in the input and output schemas have been referenced
		// in the mappings.
		if len(action.Mapping) > 0 {
			var mappingsInPaths []types.Path
			var mappingsOutPaths []types.Path
			for out, expr := range action.Mapping {
				outPath, ok := parsePropertyPath(out)
				if !ok {
					return errors.BadRequest("output mapped property %q is not valid", out)
				}
				mappingsOutPaths = append(mappingsOutPaths, outPath)
				outProp, err := action.OutSchema.PropertyByPath(outPath)
				if err != nil {
					err := err.(types.PathNotExistError)
					return errors.BadRequest("output mapped property %q not found in output schema", err.Path)
				}
				expr, err := mapexp.Compile(expr, action.InSchema, outProp.Type, outProp.Nullable)
				if err != nil {
					return errors.BadRequest("invalid expression mapped to %q: %s", out, err)
				}
				mappingsInPaths = append(mappingsInPaths, expr.Properties()...)
			}
			// Ensure that every property in the input and output schemas have been
			// mapped.
			if props := unmappedProperties(action.InSchema, mappingsInPaths); props != nil {
				return errors.BadRequest("input schema contains unmapped properties: %s", strings.Join(props, ", "))
			}
			if props := unmappedProperties(action.OutSchema, mappingsOutPaths); props != nil {
				return errors.BadRequest("output schema contains unmapped properties: %s", strings.Join(props, ", "))
			}
		}
	} else {
		if action.InSchema.Valid() {
			return errors.BadRequest("input schema cannot be provided")
		}
		if action.OutSchema.Valid() {
			return errors.BadRequest("output schema cannot be provided")
		}
	}

	return nil
}

// setSettingsFunc returns a connector.SetSettingsFunc function that sets the
// settings for the action's connection.
func (this *Action) setSettingsFunc(ctx context.Context) _connector.SetSettingsFunc {
	return func(settings []byte) error {
		return setSettings(ctx, this.db, this.action.Connection().ID, settings)
	}
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
		state.ServerType,
		state.StreamType,
		state.WebsiteType:
		return isSource && target == state.EventsTarget
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
	sort.Strings(props)
	return props
}

// parsePropertyPath parses the property path p, returning a property path with
// a single element, if p is an identifier, or a path with the components of the
// path, if p is a selector.
// The boolean return parameter reports whether p is a valid property path or
// not; when not valid, the returned path is nil.
func parsePropertyPath(p string) (types.Path, bool) {
	if !types.IsValidPropertyPath(p) {
		return nil, false
	}
	return strings.Split(p, "."), true
}

// sourceMappingSchema returns the users schema to use in mappings for source
// connections.
func sourceMappingSchema(users types.Type, connTyp state.ConnectorType) types.Type {
	usersProps := users.Properties()
	var props []types.Property
	switch connTyp {
	case
		state.AppType:
		props = make([]types.Property, 0, len(usersProps)-2)
		for _, p := range users.Unflatten().Properties() {
			// Skip the "id", "creation_time" and "timestamp" properties, which
			// cannot be mapped explicitly by the user.
			switch p.Name {
			case "id", "creation_time", "timestamp":
				continue
			}
			props = append(props, p)
		}
	case
		state.DatabaseType,
		state.FileType:

		props = make([]types.Property, 0, len(usersProps))
		for _, p := range users.Unflatten().Properties() {
			// The "creation_time" property cannot be set by the user, while the
			// "timestamp" may be manually set by them.
			if p.Name == "creation_time" {
				continue
			}
			// Replace the "id" property with a required property "id" with type
			// Text instead of Int.
			if p.Name == "id" {
				p = types.Property{Name: "id", Label: "ID", Type: types.Text(), Required: true}
			}
			props = append(props, p)
		}
	default:
		panic(fmt.Sprintf("unexpected connection type %d", connTyp))
	}
	return types.Object(props)
}
