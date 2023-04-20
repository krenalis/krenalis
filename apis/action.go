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
	"chichi/apis/events"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/apis/types"
	_connector "chichi/connector"
)

var QueryExecutionFailed errors.Code = "QueryExecutionFailed"

// Action represents an action associated to a destination connection to send
// events.
type Action struct {
	db             *postgres.DB
	action         *state.Action
	ID             int
	Connection     int
	Target         ActionTarget
	Name           string
	Enabled        bool
	EventType      *string
	ScheduleStart  *int
	SchedulePeriod *SchedulePeriod
	Filter         *ActionFilter
	Schema         types.Type
	Mapping        map[string]string
	Transformation *Transformation
	Query          *string
}

// fromState serializes action into ac.
func (ac *Action) fromState(db *postgres.DB, action *state.Action) {
	c := action.Connection()
	ac.db = db
	ac.action = action
	ac.ID = action.ID
	ac.Connection = c.ID
	ac.Target = ActionTarget(action.Target)
	ac.Name = action.Name
	ac.Enabled = action.Enabled
	if action.EventType != "" {
		et := action.EventType
		ac.EventType = &et
	}
	if action.Target == state.UsersTarget || action.Target == state.GroupsTarget {
		start := int(action.ScheduleStart)
		period := SchedulePeriod(action.SchedulePeriod)
		ac.ScheduleStart = &start
		ac.SchedulePeriod = &period
	}
	if action.Filter != nil {
		ac.Filter = &ActionFilter{
			Logical:    ActionFilterLogical(action.Filter.Logical),
			Conditions: make([]ActionFilterCondition, len(action.Filter.Conditions)),
		}
		for i, condition := range action.Filter.Conditions {
			ac.Filter.Conditions[i] = ActionFilterCondition(condition)
		}
	}
	ac.Schema = action.Schema
	if action.Mapping != nil {
		ac.Mapping = make(map[string]string, len(action.Mapping))
		for out, in := range action.Mapping {
			ac.Mapping[out] = in
		}
	}
	if t := action.Transformation; t != nil {
		ac.Transformation = &Transformation{
			In:           t.In,
			Out:          t.Out,
			PythonSource: t.PythonSource,
		}
	}
	if typ := c.Connector().Type; typ == state.DatabaseType {
		query := action.Query
		ac.Query = &query
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
func (ac *Action) Delete() error {
	n := state.DeleteActionNotification{
		Connection: ac.action.Connection().ID,
		ID:         ac.action.ID,
	}
	ctx := context.Background()
	err := ac.db.Transaction(ctx, func(tx *postgres.Tx) error {
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
func (ac *Action) Execute(reimport bool) error {
	if _, ok := ac.action.Execution(); ok {
		return errors.Unprocessable(ExecutionInProgress, "action %d is already in progress", ac.action.ID)
	}
	if t := ac.action.Target; t != state.UsersTarget && t != state.GroupsTarget {
		return errors.BadRequest("action %d with target %s cannot be executed", ac.action.ID, t)
	}
	c := ac.action.Connection()
	if c.Connector().Type == state.FileType {
		if _, ok := c.Storage(); !ok {
			return errors.Unprocessable(NoStorage, "file connection %d does not have a storage", c.ID)
		}
	}
	return ac.addExecution(reimport)
}

// Set sets action.
//
// Refer to the specifications in the file "connector/Actions support.md" for
// more details.
//
// It returns an errors.UnprocessableError error with code
//
//   - PropertyNotExists, if a property of a mapping / transformation does not
//     exist in the schema (except for properties of the event type schema,
//     which is specified and thus returned as an errors.BadRequest error).
//   - QueryExecutionFailed, if the execution of the specified query fails.
func (ac *Action) Set(action ActionToSet) error {
	connection := &Connection{
		db:         ac.db,
		connection: ac.action.Connection(),
	}
	schema, err := connection.validateActionToSet(action, ac.action.Target, ac.action.EventType)
	if err != nil {
		return err
	}
	n := state.SetActionNotification{
		ID:             ac.action.ID,
		Name:           action.Name,
		Enabled:        action.Enabled,
		Schema:         schema,
		Mapping:        action.Mapping,
		Transformation: (*state.Transformation)(action.Transformation),
		Query:          action.Query,
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
	ctx := context.Background()
	c := ac.action.Connection()
	// Marshal the schema.
	rawSchema, err := schema.MarshalJSON()
	if err != nil {
		if ac.EventType == nil {
			return fmt.Errorf("cannot marshal fetched schema for action %d of connection %d: %s", ac.ID, c.ID, err)
		}
		return fmt.Errorf("cannot marshal fetched schema for event type %q of connection %d: %s", *ac.EventType, c.ID, err)
	}
	if utf8.RuneCount(rawSchema) > rawSchemaMaxSize {
		if ac.EventType == nil {
			return fmt.Errorf("cannot marshal fetched schema for action %d of connection %d: data is too large", ac.ID, c.ID)
		}
		return fmt.Errorf("cannot marshal fetched schema for event type %q of connection %d: data is too large", *ac.EventType, c.ID)
	}
	err = ac.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE actions SET\n"+
			"name = $1, enabled = $2, filter = $3, schema = $4, mapping = $5,\n"+
			"transformation.in_types = $6, transformation.out_types = $7,\n"+
			"transformation.python_source = $8, query = $9 WHERE id = $10",
			n.Name, n.Enabled, string(filter), rawSchema, string(mapping),
			string(tIn), string(tOut), string(tSource), n.Query, n.ID,
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

// SetSchedulePeriod sets the schedule period, in minutes, of the action. The
// action must be a Users or Groups action and period can be 5, 15, 30, 60, 120,
// 180, 360, 480, 720, or 1440.
func (ac *Action) SetSchedulePeriod(period SchedulePeriod) error {
	switch ac.action.Target {
	case state.UsersTarget, state.GroupsTarget:
	default:
		return errors.BadRequest("cannot set schedule period of a %s action", ac.action.Target)
	}
	switch period {
	case 5, 15, 30, 60, 120, 180, 360, 480, 720, 1440:
	default:
		return errors.BadRequest("schedule period %d is not valid", period)
	}
	n := state.SetActionSchedulePeriodNotification{
		ID:             ac.action.ID,
		SchedulePeriod: int16(period),
	}
	ctx := context.Background()
	err := ac.db.Transaction(ctx, func(tx *postgres.Tx) error {
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
func (ac *Action) SetStatus(enabled bool) error {
	if enabled == ac.action.Enabled {
		return nil
	}
	n := state.SetActionStatusNotification{
		ID:      ac.action.ID,
		Enabled: enabled,
	}
	ctx := context.Background()
	err := ac.db.Transaction(ctx, func(tx *postgres.Tx) error {
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
//   - PropertyNotExists, if a property of a mapping / transformation does not
//     exist in the schema (except for properties of the event type schema,
//     which is specified and thus returned as an errors.BadRequest error).
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
		if len(t.In.Properties()) == 0 {
			return types.Type{}, errors.BadRequest("input schema of transformation does not have properties")
		}
		if !t.Out.Valid() || t.Out.PhysicalType() != types.PtObject {
			return types.Type{}, errors.BadRequest("output schema of transformation is not valid")
		}
		if len(t.Out.Properties()) == 0 {
			return types.Type{}, errors.BadRequest("output schema of transformation does not have properties")
		}
		// TODO(Gianluca): do a proper validation of the Python source code.
		if !strings.Contains(t.PythonSource, "def transform") {
			return types.Type{}, errors.BadRequest("Python source code of transformation does not contain 'transform' function")
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
	filtersAllowed := connector.Type == state.AppType &&
		c.Role == state.DestinationRole &&
		target == state.EventsTarget
	if action.Filter != nil && !filtersAllowed {
		return types.Type{}, errors.BadRequest("filters are not allowed")
	}

	// Fetch the schema with which to validate an action to be added.
	var schema types.Type
	if c.Role == state.SourceRole {
		switch connector.Type {
		case state.AppType:
			s, err := this.fetchAppSchema(target, eventType)
			if err != nil {
				return types.Type{}, err
			}
			schema = s
		case state.DatabaseType:
			s, err := this.fetchDatabaseSchema(action.Query)
			if err != nil {
				if _, ok := err.(*_connector.DatabaseQueryError); ok {
					return types.Type{}, errors.Unprocessable(QueryExecutionFailed, "query execution of connection %d failed: %w", c.ID, err)
				}
				return types.Type{}, err
			}
			schema = s
		case state.FileType:
			s, err := this.fetchFileSchema()
			if err != nil {
				return types.Type{}, err
			}
			schema = s
		}
	}

	// Determine the input and the output schema and check if the connector has
	// the action target.
	var inSchema, outSchema types.Type
	switch target {
	case state.UsersTarget, state.GroupsTarget:
		if !connector.Targets.Contains(target) {
			return types.Type{}, errors.BadRequest("connection %d does not have target %s", c.ID, target)
		}
		ws := c.Workspace()
		schemaName := strings.ToLower(target.String())
		grSchema, ok := ws.Schemas[schemaName]
		if !ok {
			return types.Type{}, errors.BadRequest("workspace %d of connection %d does not have %s schema", ws.ID, c.ID, schemaName)
		}
		if c.Role == state.SourceRole {
			inSchema = schema
			outSchema = *grSchema
		} else {
			inSchema = *grSchema
			outSchema = schema
		}
	case state.EventsTarget:
		if !connector.Targets.Contains(state.EventsTarget) {
			return types.Type{}, errors.BadRequest("connection %d cannot have actions on events", c.ID)
		}
		switch connector.Type {
		case state.MobileType, state.ServerType, state.WebsiteType:
			if eventType != "" {
				return types.Type{}, errors.Unprocessable(EventNotExists, "connection %d does not have event type %q", c.ID, eventType)
			}
		default:
			eventTypes, err := this.fetchEventTypes()
			if err != nil {
				return types.Type{}, err
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
			inSchema = events.Schema
			outSchema = schema // if there is not a schema, it is the invalid schema.
		}
	}

	// Check if the mapping (and the transformations) are allowed and required
	// for this action.
	var requiresMapping bool
	switch connector.Type {
	case
		state.AppType,
		state.DatabaseType,
		state.FileType:
		if c.Role == state.SourceRole {
			requiresMapping = target == state.UsersTarget || target == state.GroupsTarget
		} else {
			requiresMapping = target == state.EventsTarget && schema.Valid()
		}
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

	// Validate the filter properties.
	if action.Filter != nil {
		for i, cond := range action.Filter.Conditions {
			if existsInObject(conditionProperties[i], inSchema) {
				continue
			}
			msg := fmt.Sprintf("property expression %q of filter condition does not exist", cond.Property)
			if target == state.EventsTarget {
				return types.Type{}, errors.BadRequest(msg)
			}
			return types.Type{}, errors.Unprocessable(PropertyNotExists, msg)
		}
	}

	// Collect the top-level required properties in the action's output schema,
	// which must be necessarily mapped by mappings or by the Python transformation.
	var requiredOutProperties map[string]struct{}
	if outSchema.Valid() {
		requiredOutProperties = map[string]struct{}{}
		for _, p := range outSchema.Properties() {
			if p.Required {
				requiredOutProperties[p.Name] = struct{}{}
			}
		}
	}

	// Validate the mapping.
	if action.Mapping != nil {
		i := 0
		for out, in := range action.Mapping {
			// Validate the input property expression.
			if !existsInObject(mappingInPaths[i], inSchema) {
				msg := fmt.Sprintf("input property %q does not exist in schema", in)
				if target == state.EventsTarget {
					return types.Type{}, errors.BadRequest(msg)
				}
				return types.Type{}, errors.Unprocessable(PropertyNotExists, msg)
			}
			// Validate the output property expression.
			if !existsInObject(mappingOutPaths[i], outSchema) {
				return types.Type{}, errors.Unprocessable(PropertyNotExists, "output property %q does not exist in schema", out)
			}
			delete(requiredOutProperties, mappingOutPaths[i][0])
			i++
		}
	}

	// Validate the transformation.
	if action.Transformation != nil {
		// Validate the input properties.
		inProps := map[string]types.Property{}
		for _, prop := range inSchema.Properties() {
			inProps[prop.Name] = prop
		}
		for _, in := range action.Transformation.In.Properties() {
			p, ok := inProps[in.Name]
			if !ok {
				return types.Type{}, errors.BadRequest("property name %q does not exist in schema", in.Name)
			}
			if !p.Type.EqualTo(in.Type) {
				return types.Type{}, errors.BadRequest("expecting type %s for property %q, got %s", p.Type, p.Name, in.Type)
			}
		}
		// Validate the output properties.
		outProps := map[string]types.Property{}
		for _, prop := range outSchema.Properties() {
			outProps[prop.Name] = prop
		}
		for _, out := range action.Transformation.Out.Properties() {
			p, ok := outProps[out.Name]
			if !ok {
				return types.Type{}, errors.BadRequest("property name %q does not exist", out.Name)
			}
			if !p.Type.EqualTo(out.Type) {
				return types.Type{}, errors.BadRequest("expecting type %s for property %q, got %s", p.Type, p.Name, out.Type)
			}
			delete(requiredOutProperties, out.Name)
		}
	}

	// Check if every required property has been mapped/transformed.
	if len(requiredOutProperties) > 0 {
		var name string
		for p := range requiredOutProperties {
			if name == "" || p < name {
				name = p
			}
		}
		return types.Type{}, errors.BadRequest("schema property %q is required but is not mapped", name)
	}

	return schema, nil
}

// fetchEventTypes fetches the event types for the connection.
func (this *Connection) fetchEventTypes() ([]*_connector.EventType, error) {

	c := this.connection
	connector := c.Connector()

	var clientSecret, resourceCode, accessToken string
	if r, ok := c.Resource(); ok {
		clientSecret = connector.OAuth.ClientSecret
		resourceCode = r.Code
		var err error
		accessToken, err = freshAccessToken(this.db, r)
		if err != nil {
			return nil, fmt.Errorf("cannot retrive the OAuth access token: %s", err)
		}
	}
	ctx := context.Background()
	app, err := _connector.RegisteredApp(connector.Name).Open(ctx, &_connector.AppConfig{
		Role:          _connector.Role(c.Role),
		Settings:      c.Settings,
		ClientSecret:  clientSecret,
		Resource:      resourceCode,
		AccessToken:   accessToken,
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

	var schema types.Type

	c := this.connection
	connector := c.Connector()

	var clientSecret, resourceCode, accessToken string
	if r, ok := c.Resource(); ok {
		clientSecret = connector.OAuth.ClientSecret
		resourceCode = r.Code
		var err error
		accessToken, err = freshAccessToken(this.db, r)
		if err != nil {
			return types.Type{}, fmt.Errorf("cannot retrive the OAuth access token: %s", err)
		}
	}
	ctx := context.Background()
	app, err := _connector.RegisteredApp(connector.Name).Open(ctx, &_connector.AppConfig{
		Role:          _connector.Role(c.Role),
		Settings:      c.Settings,
		ClientSecret:  clientSecret,
		Resource:      resourceCode,
		AccessToken:   accessToken,
		PrivacyRegion: _connector.PrivacyRegion(c.Workspace().PrivacyRegion),
	})
	if err != nil {
		return types.Type{}, fmt.Errorf("cannot connect to the connector: %s", err)
	}

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
			return types.Type{}, errors.New("connection has returned a schema without source properties")
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
			return types.Type{}, errors.New("connection has returned a schema without source properties")
		}
	}
	return schema, nil
}

// fetchDatabaseSchema fetches the schema of a database connection executing the
// given query. It returns a *connectors.DatabaseQueryError error if the database
// returns an error executing the query.
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
	schema, rows, err := connection.Query(usersQuery)
	if err != nil {
		return types.Type{}, err
	}
	err = rows.Close()
	if err != nil {
		return types.Type{}, err
	}

	return schema, nil
}

// fetchFileSchema fetches the schema of a file connection.
func (this *Connection) fetchFileSchema() (types.Type, error) {

	c := this.connection
	connector := c.Connector()
	cRole := _connector.Role(c.Role)

	var ctx = context.Background()

	// Retrieve the storage associated to the file connection.
	var storage _connector.StorageConnection
	{
		s, ok := c.Storage()
		if !ok {
			return types.Type{}, errors.New("file connection has not storage")
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
			return types.Type{}, err
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
		return types.Type{}, err
	}

	// Read only the columns.
	rc, updateTime, err := storage.Reader(file.Path())
	if err != nil {
		return types.Type{}, err
	}
	defer rc.Close()
	records := fh.newRecordWriter(identityColumn, timestampColumn, true)
	err = file.Read(rc, updateTime, records)
	if err != nil && err != errRecordStop {
		return types.Type{}, err
	}
	properties := make([]types.Property, len(records.columns))
	for i, col := range records.columns {
		properties[i].Name = col.Name
		properties[i].Type = col.Type
	}
	schema := types.Object(properties)

	return schema, nil
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
		if isSource {
			return usersOrGroups
		} else {
			return target == state.EventsTarget
		}
	case
		state.DatabaseType,
		state.FileType:
		return isSource && usersOrGroups
	case
		state.ServerType,
		state.StreamType,
		state.WebsiteType:
		return !isSource && target == state.EventsTarget
	default:
		return false
	}
}

const noQueryLimit = -1

// compileActionQuery compiles the given query and returns it. If limit is
// noQueryLimit removes the ':limit' placeholder (along with '[[' and ']]');
// otherwise, replaces the placeholders with limit.
func compileActionQuery(query string, limit int) (string, error) {
	p := strings.Index(query, ":limit")
	if p == -1 {
		return "", errors.BadRequest("query does not contain the ':limit' placeholder")
	}
	s1 := strings.Index(query[:p], "[[")
	if s1 == -1 {
		return "", errors.BadRequest("query does not contain '[['")
	}
	n := len(":limit")
	s2 := strings.Index(query[p+n:], "]]")
	if s2 == -1 {
		return "", errors.BadRequest("query does not contain ']]'")
	}
	s2 += p + n + 2
	if limit == noQueryLimit {
		return query[:s1] + query[s2:], nil
	}
	return query[:s1] + strings.ReplaceAll(query[s1+2:s2-2], ":limit", strconv.Itoa(limit)) + query[s2:], nil
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
