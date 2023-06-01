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
	"strings"
	"unicode/utf8"

	"chichi/apis/errors"
	"chichi/apis/httpclient"
	"chichi/apis/postgres"
	"chichi/apis/state"
	_connector "chichi/connector"
	"chichi/connector/types"
)

var QueryExecutionFailed errors.Code = "QueryExecutionFailed"

// Action represents an action associated to a destination connection to send
// events.
type Action struct {
	db                 *postgres.DB
	action             *state.Action
	http               *httpclient.HTTP
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
	Schema             types.Type
	Mapping            map[string]string
	Transformation     *Transformation
	Query              *string
	Path               *string
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

// fromState serializes action into ac.
func (this *Action) fromState(db *postgres.DB, http *httpclient.HTTP, action *state.Action) {
	c := action.Connection()
	this.db = db
	this.action = action
	this.http = http
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
	this.Schema = action.Schema
	if action.Mapping != nil {
		this.Mapping = make(map[string]string, len(action.Mapping))
		for out, in := range action.Mapping {
			this.Mapping[out] = in
		}
	}
	if t := action.Transformation; t != nil {
		this.Transformation = &Transformation{
			In:           t.In,
			Out:          t.Out,
			PythonSource: t.PythonSource,
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

// Transformation represents the transformation of an action.
type Transformation struct {
	In           types.Type
	Out          types.Type
	PythonSource string
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
//
// It returns an errors.UnprocessableError error with code
//
//   - FetchSchemaFailed, if an error occurred fetching the action's schema.
//   - PropertyNotExists, if a property of a mapping / transformation does not
//     exist in the schema (except for properties of the event type schema,
//     which is specified and thus returned as an errors.BadRequest error).
//   - QueryExecutionFailed, if the execution of the action's query fails.
func (this *Action) Set(action ActionToSet) error {
	c := this.action.Connection()
	connection := &Connection{
		db:         this.db,
		connection: c,
		http:       this.http,
	}
	schema, err := connection.validateActionToSet(action, this.action.Target, this.action.EventType)
	if err != nil {
		return err
	}
	n := state.SetActionNotification{
		ID:             this.action.ID,
		Name:           action.Name,
		Enabled:        action.Enabled,
		Mapping:        action.Mapping,
		Transformation: (*state.Transformation)(action.Transformation),
		Query:          action.Query,
		Path:           action.Path,
		Sheet:          action.Sheet,
		ExportMode:     (*state.ExportMode)(action.ExportMode),
	}
	if shouldStoreActionSchema(c.Connector().Type, c.Role, this.action.Target) {
		n.Schema = schema
	}
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
			return err
		}
	}
	if action.Mapping != nil {
		mapping, err = json.Marshal(action.Mapping)
		if err != nil {
			return err
		}
	}
	if t := action.Transformation; t != nil {
		tIn, err = json.Marshal(t.In)
		if err != nil {
			return err
		}
		tOut, err = json.Marshal(t.Out)
		if err != nil {
			return err
		}
		tSource = []byte(t.PythonSource)
	}
	if props := action.MatchingProperties; props != nil {
		n.MatchingProperties = &state.MatchingProperties{
			Internal: props.Internal,
			External: props.External,
		}
	}
	ctx := context.Background()
	// Marshal the schema.
	var rawSchema []byte
	if n.Schema.Valid() {
		rawSchema, err = n.Schema.MarshalJSON()
		if err != nil {
			if this.EventType == nil {
				return fmt.Errorf("cannot marshal fetched schema for action %d of connection %d: %s", this.ID, c.ID, err)
			}
			return fmt.Errorf("cannot marshal fetched schema for event type %q of connection %d: %s", *this.EventType, c.ID, err)
		}
		if utf8.RuneCount(rawSchema) > rawSchemaMaxSize {
			if this.EventType == nil {
				return fmt.Errorf("cannot marshal fetched schema for action %d of connection %d: data is too large", this.ID, c.ID)
			}
			return fmt.Errorf("cannot marshal fetched schema for event type %q of connection %d: data is too large", *this.EventType, c.ID)
		}
	} else {
		rawSchema = []byte{}
	}
	var matchPropInternal, matchPropExternal string
	if n.MatchingProperties != nil {
		matchPropInternal = n.MatchingProperties.Internal
		matchPropExternal = n.MatchingProperties.External
	}
	err = this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE actions SET\n"+
			"name = $1, enabled = $2, filter = $3, schema = $4, mapping = $5,\n"+
			"transformation.in_types = $6, transformation.out_types = $7,\n"+
			"transformation.python_source = $8, query = $9, path = $10, sheet = $11,\n"+
			"export_mode = $12, matching_properties_internal = $13,\n"+
			"matching_properties_external = $14 WHERE id = $15",
			n.Name, n.Enabled, string(filter), rawSchema, string(mapping),
			string(tIn), string(tOut), string(tSource), n.Query, n.Path, n.Sheet,
			n.ExportMode, matchPropInternal, matchPropExternal, n.ID,
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
func (this *Action) setUserCursor(ctx context.Context, cursor string) error {
	n := state.SetActionUserCursorNotification{
		ID:         this.action.ID,
		UserCursor: cursor,
	}
	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE actions SET user_cursor = $1 WHERE id = $2", n.UserCursor, n.ID)
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

	// Mapping is the mapping of the action, if it has one, otherwise is nil.
	//
	// Every action that supports mappings / transformation must have an
	// associated mapping or a transformation, which are mutually exclusive.
	//
	// If it has a mapping, the names of the properties in which the values are
	// mapped (the keys of the map) must be present in the output schema of the
	// action, while the mapping properties (the values of the map) must be
	// property names or property selectors (property names separated by a dot
	// '.').
	Mapping map[string]string

	// Transformation is the transformation of the action, if it has one,
	// otherwise is nil.
	// Every action that supports mappings / transformation must have an
	// associated mapping or a transformation, which are mutually exclusive.
	//
	// If it has a transformation:
	//
	// - it must have at least one input and one output property
	// - the names of the properties in the input schema of the transformation
	//   must be present in the input schema of the action.
	// - the names of the properties in the output schema of the transformation
	//   must be present in the output schema of the action.
	// - every required property in the output schema of the action must be
	//   present in the output schema of the transformation
	Transformation *Transformation

	// Query is the query of the action, if it has one, otherwise it is the
	// empty string.
	Query string

	// Path is the path of the file. It cannot be longer than 1024 runes,
	// and it is empty for non-file actions.
	Path string

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
//
// It returns an errors.UnprocessableError error with code
//
//   - EventTypeNotExists, if the specified event type does not exist.
//   - FetchSchemaFailed, if an error occurred fetching the schema.
//   - NoStorage, if the file connection does not have a storage.
//   - PropertyNotExists, if a property of a mapping / transformation does not
//     exist in the schema (except for properties of the event type schema,
//     which is specified and thus returned as an errors.BadRequest error).
//   - ReadFileFailed, if an error occurred reading the file.
//   - QueryExecutionFailed, if the execution of the specified query fails.
func (this *Connection) validateActionToSet(action ActionToSet, target state.ActionTarget, eventType string) (types.Type, error) {

	// First, do formal validations.

	// Validate the name.
	if action.Name == "" {
		return types.Type{}, errors.BadRequest("name is empty")
	}
	if !utf8.ValidString(action.Name) {
		return types.Type{}, errors.BadRequest("name is not UTF-8 encoded")
	}
	if n := utf8.RuneCountInString(action.Name); n > 60 {
		return types.Type{}, errors.BadRequest("name is longer than 60 runes")
	}
	// Validate the filter.
	var conditionProperties [][]string
	if action.Filter != nil {
		if l := action.Filter.Logical; l != "all" && l != "any" {
			return types.Type{}, errors.BadRequest("filter logical operator %q is not valid", action.Filter.Logical)
		}
		if len(action.Filter.Conditions) == 0 {
			return types.Type{}, errors.BadRequest("filter does not contain conditions")
		}
		conditionProperties = make([][]string, len(action.Filter.Conditions))
		for i, condition := range action.Filter.Conditions {
			property, ok := parsePropertyExpression(condition.Property)
			if !ok {
				return types.Type{}, errors.BadRequest("filter condition property expression %q is not valid", condition.Property)
			}
			conditionProperties[i] = property
			if op := condition.Operator; op != "is" && op != "is not" {
				return types.Type{}, errors.BadRequest("filter condition operator %q is not valid", op)
			}
			if !utf8.ValidString(condition.Value) {
				return types.Type{}, errors.BadRequest("filter condition value is not UTF-8 encoded")
			}
			if n := utf8.RuneCountInString(condition.Value); n > 60 {
				return types.Type{}, errors.BadRequest("filter condition value is longer than 60 runes")
			}
		}
	}
	// Validate the mapping and the transformation.
	if action.Mapping != nil && action.Transformation != nil {
		return types.Type{}, errors.BadRequest("action can not have both mapping and transformation")
	}
	var mappingInPaths [][]string
	var mappingOutPaths [][]string
	if action.Mapping != nil {
		mappingInPaths = make([][]string, 0, len(action.Mapping))
		mappingOutPaths = make([][]string, 0, len(action.Mapping))
		for out, in := range action.Mapping {
			// Validate the input property expression.
			path, ok := parsePropertyExpression(in)
			if !ok {
				return types.Type{}, errors.BadRequest("input property expression %q of mapping is not valid", in)
			}
			mappingInPaths = append(mappingInPaths, path)
			// Validate the output property expression.
			path, ok = parsePropertyExpression(out)
			if !ok {
				return types.Type{}, errors.BadRequest("output property expression %q of mapping is not valid", out)
			}
			mappingOutPaths = append(mappingOutPaths, path)
		}
	}
	if t := action.Transformation; t != nil {
		if !t.In.Valid() || t.In.PhysicalType() != types.PtObject {
			return types.Type{}, errors.BadRequest("input schema of transformation is not valid")
		}
		if !t.Out.Valid() || t.Out.PhysicalType() != types.PtObject {
			return types.Type{}, errors.BadRequest("output schema of transformation is not valid")
		}
		// TODO(Gianluca): do a proper validation of the Python source code.
		// See the issue https://github.com/open2b/chichi/issues/188.
		if !strings.Contains(t.PythonSource, "def transform") {
			return types.Type{}, errors.BadRequest("Python source code of transformation does not contain 'transform' function")
		}
	}
	// Validate the path.
	if action.Path != "" {
		if !utf8.ValidString(action.Path) {
			return types.Type{}, errors.BadRequest("path is not UTF-8 encoded")
		}
		if n := utf8.RuneCountInString(action.Path); n > 1024 {
			return types.Type{}, errors.BadRequest("path is longer than 1024 runes")
		}
	}
	// Validate the sheet.
	if action.Sheet != "" {
		if !utf8.ValidString(action.Sheet) {
			return types.Type{}, errors.BadRequest("sheet is not UTF-8 encoded")
		}
		if n := utf8.RuneCountInString(action.Sheet); n > 100 {
			return types.Type{}, errors.BadRequest("sheet is longer than 100 runes")
		}
	}
	// Validate the export options.
	if action.ExportMode != nil {
		switch *action.ExportMode {
		case CreateOnly, UpdateOnly, CreateOrUpdate:
		default:
			return types.Type{}, errors.BadRequest("export mode %q is not valid", *action.ExportMode)
		}
	}
	if action.MatchingProperties != nil {
		props := *action.MatchingProperties
		if !types.IsValidPropertyName(props.Internal) {
			return types.Type{}, errors.BadRequest("internal matching property %q is not a valid property name", props.Internal)
		}
		if !types.IsValidPropertyName(props.External) {
			return types.Type{}, errors.BadRequest("external matching property %q is not a valid property name", props.External)
		}
	}

	// Second, do validations based on the connection.

	c := this.connection
	connector := c.Connector()

	// Check if the query is allowed.
	if connector.Type == state.DatabaseType {
		if action.Query == "" {
			return types.Type{}, errors.BadRequest("query cannot be empty for database actions")
		}
	} else {
		if action.Query != "" {
			return types.Type{}, errors.BadRequest("%s actions cannot have a query", connector.Type)
		}
	}

	// Check if the filters are allowed.
	targetUsersOrGroups := target == state.UsersTarget || target == state.GroupsTarget
	var filtersAllowed bool
	switch connector.Type {
	case state.AppType:
		filtersAllowed = c.Role == state.DestinationRole
	case state.FileType:
		filtersAllowed = targetUsersOrGroups && c.Role == state.DestinationRole
	}
	if action.Filter != nil && !filtersAllowed {
		return types.Type{}, errors.BadRequest("filters are not allowed")
	}

	// Check if the path and the sheet are allowed.
	if connector.Type == state.FileType {
		if action.Path == "" {
			return types.Type{}, errors.BadRequest("path cannot be empty for file actions")
		}
		if connector.HasSheets && action.Sheet == "" {
			return types.Type{}, errors.BadRequest("sheet cannot be empty because connection %d has sheets", c.ID)
		}
		if !connector.HasSheets && action.Sheet != "" {
			return types.Type{}, errors.BadRequest("connection %d does not have sheets", c.ID)
		}
	} else {
		if action.Path != "" {
			return types.Type{}, errors.BadRequest("%s actions cannot have a path", connector.Type)
		}
		if action.Sheet != "" {
			return types.Type{}, errors.BadRequest("%s actions cannot have a sheet", connector.Type)
		}
	}

	// Check if the export options are needed.
	needsExportOptions := connector.Type == state.AppType &&
		c.Role == state.DestinationRole &&
		targetUsersOrGroups
	if needsExportOptions {
		if action.ExportMode == nil {
			return types.Type{}, errors.BadRequest("export mode cannot be nil")
		}
		if action.MatchingProperties == nil {
			return types.Type{}, errors.BadRequest("matching properties cannot be nil")
		}
	} else {
		if action.ExportMode != nil {
			return types.Type{}, errors.BadRequest("export mode must be nil")
		}
		if action.MatchingProperties != nil {
			return types.Type{}, errors.BadRequest("matching properties must be nil")
		}
	}

	// Fetch the schema with which to validate an action to be added.
	var schema types.Type
	switch connector.Type {
	case state.AppType:
		switch target {
		case
			state.UsersTarget,
			state.GroupsTarget:
			if !connector.Targets.Contains(target) {
				return types.Type{}, errors.BadRequest("connection %d does not have target %s", c.ID, target)
			}
			s, err := this.fetchAppSchema(target, eventType)
			if err != nil {
				return types.Type{}, errors.Unprocessable(FetchSchemaFailed, "an error occurred fetching the schema: %w", err)
			}
			schema = s
		case state.EventsTarget:
			if !connector.Targets.Contains(state.EventsTarget) {
				return types.Type{}, errors.BadRequest("connection %d cannot have actions on events", c.ID)
			}
			switch connector.Type {
			case state.AppType:
				eventTypes, err := this.fetchEventTypes()
				if err != nil {
					return types.Type{}, errors.Unprocessable(FetchSchemaFailed, "an error occurred fetching the schema: %w", err)
				}
				var et *_connector.EventType
				for _, e := range eventTypes {
					if e.ID == eventType {
						et = e
						break
					}
				}
				if et == nil {
					return types.Type{}, errors.Unprocessable(EventNotExists, "connection %d does not have event type %q", c.ID, eventType)
				}
				schema = et.Schema // invalid if the event type has no schema.
			case state.MobileType, state.ServerType, state.WebsiteType:
				if eventType != "" {
					return types.Type{}, errors.Unprocessable(EventNotExists, "connection %d does not have event type %q", c.ID, eventType)
				}
			}
		}
	case state.DatabaseType:
		if c.Role == state.SourceRole {
			s, err := this.fetchDatabaseSchema(action.Query)
			if err != nil {
				return types.Type{}, err
			}
			schema = s
		}
	case state.FileType:
		if c.Role == state.SourceRole {
			s, err := this.fetchFileSchema(action.Path, action.Sheet)
			if err != nil {
				return types.Type{}, err
			}
			schema = s
		}
	}

	// Check if the mapping (and the transformations) are allowed and required
	// for this action.
	var requiresMapping bool
	switch connector.Type {
	case state.AppType:
		requiresMapping = targetUsersOrGroups || (target == state.EventsTarget && schema.Valid())
	case
		state.DatabaseType,
		state.FileType:
		requiresMapping = c.Role == state.SourceRole && targetUsersOrGroups
	}
	if requiresMapping {
		if action.Mapping == nil && action.Transformation == nil {
			return types.Type{}, errors.BadRequest("mapping (or transformation) is required")
		}
	} else {
		if action.Mapping != nil {
			return types.Type{}, errors.BadRequest("mapping not allowed")
		}
		if action.Transformation != nil {
			return types.Type{}, errors.BadRequest("transformation not allowed")
		}
	}

	return schema, nil
}

// fetchEventTypes fetches the event types for the connection.
func (this *Connection) fetchEventTypes() ([]*_connector.EventType, error) {

	c := this.connection

	var resource string
	if r, ok := c.Resource(); ok {
		resource = r.Code
	}

	ctx := context.Background()
	app, err := _connector.RegisteredApp(c.Connector().Name).Open(ctx, &_connector.AppConfig{
		Role:          _connector.Role(c.Role),
		Settings:      c.Settings,
		Resource:      resource,
		HTTPClient:    this.http.ConnectionClient(c.ID),
		PrivacyRegion: _connector.PrivacyRegion(c.Workspace().PrivacyRegion),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot connect to the connector: %s", err)
	}
	eventTypes, err := app.(_connector.AppEventsConnection).EventTypes()
	if err != nil {
		return nil, err
	}

	return eventTypes, nil
}

// fetchAppSchema fetches the schema of an app connection for the given target
// and eventType.
func (this *Connection) fetchAppSchema(target state.ActionTarget, eventType string) (types.Type, error) {

	c := this.connection

	var resource string
	if r, ok := c.Resource(); ok {
		resource = r.Code
	}

	ctx := context.Background()
	app, err := _connector.RegisteredApp(c.Connector().Name).Open(ctx, &_connector.AppConfig{
		Role:          _connector.Role(c.Role),
		Settings:      c.Settings,
		Resource:      resource,
		HTTPClient:    this.http.ConnectionClient(c.ID),
		PrivacyRegion: _connector.PrivacyRegion(c.Workspace().PrivacyRegion),
	})
	if err != nil {
		return types.Type{}, fmt.Errorf("cannot connect to the connector: %s", err)
	}

	var schema types.Type

	switch target {
	case state.EventsTarget:
		if eventType != "" {
			eventTypes, err := app.(_connector.AppEventsConnection).EventTypes()
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
				return types.Type{}, errors.Unprocessable(EventNotExists, "connection %d does not have event type %q", c.ID, eventType)
			}
		}
	case state.UsersTarget:
		schema, err = app.(_connector.AppUsersConnection).UserSchema()
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
		schema, err = app.(_connector.AppGroupsConnection).GroupSchema()
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

// fetchDatabaseSchema fetches the schema of a database connection executing the
// given query.
//
// It returns an errors.UnprocessableError error with code QueryExecutionFailed
// if the execution of the specified query fails.
func (this *Connection) fetchDatabaseSchema(query string) (types.Type, error) {

	c := this.connection
	connector := c.Connector()

	usersQuery, err := compileActionQuery(query, 0)
	if err != nil {
		return types.Type{}, err
	}
	fh := this.newFirehose(context.Background())
	connection, err := _connector.RegisteredDatabase(connector.Name).Open(fh.ctx, &_connector.DatabaseConfig{
		Role:     _connector.Role(c.Role),
		Settings: c.Settings,
		Firehose: fh,
	})
	if err != nil {
		return types.Type{}, err
	}
	var rows _connector.Rows
	rows, properties, err := connection.Query(usersQuery)
	if err != nil {
		return types.Type{}, errors.Unprocessable(QueryExecutionFailed, "query execution of connection %d failed: %w", c.ID, err)
	}
	err = rows.Close()
	if err != nil {
		return types.Type{}, err
	}

	return types.ObjectOf(properties)
}

// fetchFileSchema fetches the schema of a file connection.
//
// It returns an errors.UnprocessableError error with code
//
//   - NoStorage, if the file connection does not have a storage.
//   - ReadFileFailed, if an error occurred reading the file.
func (this *Connection) fetchFileSchema(path, sheet string) (types.Type, error) {

	c := this.connection
	connector := c.Connector()
	cRole := _connector.Role(c.Role)

	var ctx = context.Background()

	// Retrieve the storage associated to the file connection.
	var storage _connector.StorageConnection
	{
		s, ok := c.Storage()
		if !ok {
			return types.Type{}, errors.Unprocessable(NoStorage, "file connection %d does not have a storage", c.ID)
		}
		fh := this.newFirehose(ctx)
		ctx = fh.ctx
		var err error
		storage, err = _connector.RegisteredStorage(s.Connector().Name).Open(ctx, &_connector.StorageConfig{
			Role:     cRole,
			Settings: s.Settings,
			Firehose: fh,
		})
		if err != nil {
			return types.Type{}, errors.Unprocessable(ReadFileFailed, "%w", err)
		}
	}

	// Connect to the file connector and read only the columns.
	fh := this.newFirehose(ctx)
	file, err := _connector.RegisteredFile(connector.Name).Open(fh.ctx, &_connector.FileConfig{
		Role:     cRole,
		Settings: c.Settings,
		Firehose: fh,
	})
	if err != nil {
		return types.Type{}, errors.Unprocessable(ReadFileFailed, "%w", err)
	}

	// Read only the columns.
	rc, _, err := storage.Open(path)
	if err != nil {
		return types.Type{}, errors.Unprocessable(ReadFileFailed, "%w", err)
	}
	defer rc.Close()
	rw := newRecordWriter(c.ID, 0, nil)
	err = file.Read(rc, sheet, rw)
	if err != nil && err != errRecordStop {
		return types.Type{}, errors.Unprocessable(ReadFileFailed, "%w", err)
	}
	if rw.columns == nil {
		return types.Type{}, errors.Unprocessable(ReadFileFailed, "%w", errNoColumns)
	}

	return types.ObjectOf(rw.columns)
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
		return isSource && usersOrGroups
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

// existsInObject reports whether a property, denoted by its path - for example
// ["traits", "address", "street1"] - exists in the given object type (which may
// be, in the previous example, an object type with property "traits", which
// contains an object with a property "address", which contains a property with
// name "street1").
// object must have an object physical type.
// If one of the sub-properties has type map or JSON, it is assumed that such
// property exists (the validation can be done only at runtime, with the
// effective value).
func existsInObject(propPath []string, object types.Type) bool {
	if object.PhysicalType() != types.PtObject {
		panic("not an object")
	}
	name := propPath[0]
	for _, prop := range object.Properties() {
		if prop.Name != name {
			continue
		}
		// Found.
		rest := propPath[1:]
		if len(rest) == 0 {
			return true
		}
		switch prop.Type.PhysicalType() {
		case types.PtObject:
			return existsInObject(rest, prop.Type)
		case types.PtMap, types.PtJSON:
			return true
		default:
			return false
		}
	}
	return false
}

// parsePropertyExpression parses the property expression p, returning a slice
// with a single element, if p is an identifier, or a slice with the components
// of the selector.
// The boolean return parameter reports whether p is a valid property expression
// or not; when not valid, the returned slice is nil.
func parsePropertyExpression(p string) ([]string, bool) {

	// A selector.
	if strings.Contains(p, ".") {
		parts := strings.Split(p, ".")
		for _, c := range parts {
			if !types.IsValidPropertyName(c) {
				return nil, false
			}
		}
		return parts, true
	}

	// An identifier.
	ok := types.IsValidPropertyName(p)
	if !ok {
		return nil, false
	}

	return []string{p}, true
}

// shouldStoreActionSchema reports whether the schema for an action with the
// given connector type, connection role and target should be stored with the
// action.
func shouldStoreActionSchema(typ state.ConnectorType, role state.ConnectionRole, target state.ActionTarget) bool {
	if typ != state.AppType {
		return false
	}
	if target == state.EventsTarget {
		return role == state.DestinationRole
	}
	return true
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
			// "timestamp".
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
