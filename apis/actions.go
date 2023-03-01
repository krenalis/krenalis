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
	"unicode/utf8"

	"chichi/apis/errors"
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
	Endpoint       int
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
// the action type.
//
// If it has a mapping, the names of the properties in which the values are
// mapped must be present in the action type schema.
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

	if e := action.Endpoint; e < 1 || e > math.MaxInt32 {
		return fmt.Errorf("invalid endpoint identifier")
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

	switch action.Filter.Logical {
	case "":
		return errors.New("filter logical operator is empty")
	case "all", "any":
	default:
		return fmt.Errorf("filter logical operator %q does not exist", action.Filter.Logical)
	}

	// TODO(Gianluca): validate the other fields of the filter, depending on how
	// we decide to implement them.

	if action.Mapping != nil && action.Transformation != nil {
		return errors.New("action has both mapping and transformation associated")
	}
	if action.Mapping == nil && action.Transformation == nil {
		return errors.New("action has neither mapping nor transformation associated")
	}

	schemaProps := map[string]bool{}
	for _, name := range actionType.Schema.PropertiesNames() {
		schemaProps[name] = true
	}

	if action.Mapping != nil {
		for left, right := range action.Mapping {
			if !types.IsValidPropertyName(left) {
				return fmt.Errorf("property name %q is not valid", left)
			}
			if !schemaProps[right] {
				return fmt.Errorf("property name %q does not exist in action type schema", right)
			}
		}
	}

	if action.Transformation != nil {
		err := validateTransformation(action.Transformation)
		if err != nil {
			return err
		}
		for _, right := range action.Transformation.Out.PropertiesNames() {
			if !schemaProps[right] {
				return fmt.Errorf("property name %q does not exist in action type schema", right)
			}
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
