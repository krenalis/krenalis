//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"chichi/apis/errors"
	"chichi/apis/events"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/apis/types"
)

// Action represents an action associated to a destination connection to send
// events.
type Action struct {
	db             *postgres.DB
	action         *state.Action
	connection     *Connection
	ID             int
	Connection     int
	ActionType     int
	Name           string
	Enabled        bool
	Endpoint       int // 0 when the action type does not support endpoints.
	Filter         ActionFilter
	Mapping        map[string]string
	Transformation *Transformation
}

// ActionFilter represents an action filter associated to an action.
type ActionFilter struct {
	Logical    string
	Conditions []ActionFilterCondition
}

// ActionFilterCondition represents an action filter condition associated to an
// action's filter.
type ActionFilterCondition struct {
	Property string
	Operator string
	Value    string
}

// Delete deletes the action.
// It returns an errors.NotFoundError error if the action does not exist
// anymore.
func (this *Action) Delete() error {
	n := state.DeleteConnectionActionNotification{
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

// Set sets action.
//
// The action name must be a non-empty valid UTF-8 encoded string and cannot be
// longer than 60 runes. The action must have a mapping associated or a
// function, and cannot have both.
//
// The action endpoint must be the identifier of one the endpoints supported by
// the action type, if it supports them, otherwise must be 0.
//
// If it has a mapping, the names of the properties in which the values are
// mapped (the keys of the map) must be present in the action type schema, while
// the mapping properties (the values of the map) must be property names or
// property selectors (property names separated by a dot '.').
//
// If it has a transformation, such transformation should have at least one
// input and one output property, its source should be a valid Python source,
// and the names of the properties in the output schema must be present in the
// action type schema.
// TODO(Gianluca): specify how this transformation function should be written,
// depending on the use on the events dispatcher.
func (this *Action) Set(action ActionToSet) error {
	actionTypes := this.connection.actionTypes()
	err := validateAction(action, actionTypes)
	if err != nil {
		return errors.BadRequest(err.Error())
	}
	n := state.SetConnectionActionNotification{
		ID:             this.action.ID,
		Connection:     this.action.Connection().ID,
		ActionType:     action.ActionType,
		Name:           action.Name,
		Enabled:        action.Enabled,
		Endpoint:       action.Endpoint,
		Mapping:        action.Mapping,
		Transformation: (*state.Transformation)(action.Transformation),
	}
	n.Filter.Logical = action.Filter.Logical
	n.Filter.Conditions = make([]state.ActionFilterConditionNotification, len(action.Filter.Conditions))
	for i := range n.Filter.Conditions {
		n.Filter.Conditions[i] = (state.ActionFilterConditionNotification)(action.Filter.Conditions[i])
	}
	ctx := context.Background()
	var filter, mapping, tIn, tOut, tSource []byte
	filter, err = json.Marshal(action.Filter)
	if err != nil {
		return err
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
	err = this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE actions SET\n"+
			"name = $1, enabled = $2, endpoint = $3, filter = $4, mapping = $5,\n"+
			"transformation.in_types = $6, transformation.out_types = $7,\n"+
			"transformation.python_source = $8 WHERE id = $9",
			n.Name, n.Enabled, n.Endpoint, string(filter), string(mapping), string(tIn),
			string(tOut), string(tSource), n.ID,
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

// SetStatus sets the status of the action.
func (this *Action) SetStatus(enabled bool) error {
	if enabled == this.action.Enabled {
		return nil
	}
	n := state.SetConnectionActionStatusNotification{
		ID:      this.ID,
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
// action or updating an existing one.
type ActionToSet struct {
	ActionType     int
	Name           string
	Enabled        bool
	Endpoint       int
	Filter         ActionFilter
	Mapping        map[string]string
	Transformation *Transformation
}

// validateAction validates the given action, retuning nil if the action is
// valid or an error with an error message explaining why the action is invalid.
func validateAction(action ActionToSet, actionTypes []*ActionType) error {

	var actionType *state.ActionType
	for _, at := range actionTypes {
		if at.ID == action.ActionType {
			actionType = at.actionType
			break
		}
	}
	if actionType == nil {
		return fmt.Errorf("action type %d does not exist", action.ActionType)
	}

	if action.Name == "" {
		return errors.New("action name is empty")
	}
	if !utf8.ValidString(action.Name) {
		return errors.New("action name is not UTF-8 encoded")
	}
	if n := utf8.RuneCountInString(action.Name); n > 60 {
		return errors.New("action name is longer than 60 runes")
	}

	if e := action.Endpoint; e < 0 || e > math.MaxInt32 {
		return fmt.Errorf("invalid endpoint identifier")
	}
	if action.Endpoint != 0 && actionType.Endpoints == nil {
		return fmt.Errorf("endpoints not supported by this action type")
	}
	if actionType.Endpoints != nil {
		if action.Endpoint == 0 {
			return fmt.Errorf("endpoint is mandatory for this action type")
		}
		endpointFound := false
		for _, ee := range actionType.Endpoints {
			if ee == action.Endpoint {
				endpointFound = true
				break
			}
		}
		if !endpointFound {
			return fmt.Errorf("endpoint %d not found", action.Endpoint)
		}
	}

	switch action.Filter.Logical {
	case "":
		return errors.New("filter logical operator is empty")
	case "all", "any":
	default:
		return fmt.Errorf("filter logical operator %q does not exist", action.Filter.Logical)
	}

	for _, cond := range action.Filter.Conditions {
		switch cond.Property {
		case "AnonymousID", "Event", "UserID":
		default:
			return fmt.Errorf("filter property %q does not exist", cond.Property)
		}
		switch cond.Operator {
		case "is", "is not":
		default:
			return fmt.Errorf("filter condition operator %q does not exist", cond.Operator)
		}
		if !utf8.ValidString(cond.Value) {
			return errors.New("filter condition value is not UTF-8 encoded")
		}
		if n := utf8.RuneCountInString(cond.Value); n > 60 {
			return errors.New("filter condition value is longer than 50 runes")
		}
	}

	if actionType.Schema.Valid() {
		if action.Mapping != nil && action.Transformation != nil {
			return errors.New("action has both mapping and transformation associated")
		}
		if action.Mapping == nil && action.Transformation == nil {
			return errors.New("action has neither mapping nor transformation associated")
		}
	} else {
		if action.Mapping != nil {
			return errors.New("action has mappings but the action type does not have a schema")
		}
		if action.Transformation != nil {
			return errors.New("action has a transformation function but the action type does not have a schema")
		}
	}

	if actionType.Schema.Valid() {

		// Collect the top-level required properties in the action type schema,
		// which must be necessarily mapped by mappings or by the Python
		// transformation.
		actionTypeRequiredProps := map[string]struct{}{}
		for _, p := range actionType.Schema.Properties() {
			if p.Required {
				actionTypeRequiredProps[p.Name] = struct{}{}
			}
		}

		// Validate the mapping.
		if action.Mapping != nil {
			for right, left := range action.Mapping {
				// Validate the left expression, which can be an identifier or a
				// selector.
				if strings.Contains(left, ".") { // selector, eg. "traits.address.street1".
					path := strings.Split(left, ".")
					for _, p := range path {
						if !types.IsValidPropertyName(p) {
							return fmt.Errorf("property name %q in selector %q is not valid", p, left)
						}
					}
					if !existsInObject(path, events.Schema) {
						return fmt.Errorf("property %q does not exist in event schema", left)
					}
				} else { // identifier, eg. "addressStreet1".
					if !types.IsValidPropertyName(left) {
						return fmt.Errorf("property name %q is not valid", left)
					}
					if !existsInObject([]string{left}, events.Schema) {
						return fmt.Errorf("property %q does not exist in event schema", left)
					}
				}
				// Validate the right expression.
				rightPath := strings.Split(right, ".")
				if !existsInObject(rightPath, actionType.Schema) {
					return fmt.Errorf("property %q does not exist in action type schema", right)
				}
				delete(actionTypeRequiredProps, rightPath[0])
			}
		}

		// Validate the transformation.
		if action.Transformation != nil {
			// Validate the transformation itself.
			err := validateTransformation(action.Transformation)
			if err != nil {
				return err
			}
			// Validate the input properties of the transformation.
			eventProps := map[string]types.Property{}
			for _, prop := range events.Schema.Properties() {
				eventProps[prop.Name] = prop
			}
			for _, left := range action.Transformation.In.Properties() {
				p, ok := eventProps[left.Name]
				if !ok {
					return fmt.Errorf("property name %q does not exist in event schema", left.Name)
				}
				if !p.Type.EqualTo(left.Type) {
					return fmt.Errorf("expecting type %s for property %q, got %s", p.Type, p.Name, left.Type)
				}
			}
			// Validate the output properties of the transformation.
			actionTypeProps := map[string]types.Property{}
			for _, prop := range actionType.Schema.Properties() {
				actionTypeProps[prop.Name] = prop
			}
			for _, right := range action.Transformation.Out.Properties() {
				p, ok := actionTypeProps[right.Name]
				if !ok {
					return fmt.Errorf("property name %q does not exist in action type schema", right.Name)
				}
				if !p.Type.EqualTo(right.Type) {
					return fmt.Errorf("expecting type %s for property %q, got %s", p.Type, p.Name, right.Type)
				}
				delete(actionTypeRequiredProps, right.Name)
			}
		}

		// Check if every required property has been mapped/transformed.
		if len(actionTypeRequiredProps) > 0 {
			var name string
			for p := range actionTypeRequiredProps {
				if name == "" || p < name {
					name = p
				}
			}
			return fmt.Errorf("required property %s in action type schema not mapped", name)
		}
	}

	return nil
}

// ActionType represents an action type.
type ActionType struct {
	actionType  *state.ActionType
	ID          int
	Name        string
	Description string
	Endpoints   map[int]string // connector's endpoints supported by this action.
	Schema      types.Type
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
