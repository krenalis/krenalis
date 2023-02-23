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
	"unicode/utf8"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/apis/types"
	"chichi/connector"
)

// Action represents an action associated to a destination connection to send
// events.
type Action struct {
	db             *postgres.DB
	action         *state.Action
	ID             int
	Connection     int
	ActionType     int
	Name           string
	Enabled        bool
	Filter         connector.ActionFilter
	Mapping        map[string]string
	Transformation *Transformation
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
// If it has a mapping, the names of the mapped properties must be valid
// property names.
//
// If it has a transformation, such transformation should have at least one
// input and one output property, and its source should be a valid Python
// source.
// TODO(Gianluca): specify how this transformation function should be written,
// depending on the use on the events dispatcher.
func (this *Action) Set(action ActionToSet) error {
	err := validateAction(action)
	if err != nil {
		return errors.BadRequest(err.Error())
	}
	n := state.SetConnectionActionNotification{
		ID:             this.action.ID,
		Connection:     this.action.Connection().ID,
		ActionType:     action.ActionType,
		Name:           action.Name,
		Enabled:        action.Enabled,
		Filter:         action.Filter,
		Mapping:        action.Mapping,
		Transformation: (*state.Transformation)(action.Transformation),
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
			"name = $1, enabled = $2, filter = $3, mapping = $4,\n"+
			"transformation.in_types = $5, transformation.out_types = $6,\n"+
			"transformation.python_source = $7 WHERE id = $8",
			n.Name, n.Enabled, string(filter), string(mapping), string(tIn),
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
	Filter         connector.ActionFilter
	Mapping        map[string]string
	Transformation *Transformation
}

// validateAction validates the given action, retuning nil if the action is
// valid or an error with an error message explaining why the action is invalid.
func validateAction(action ActionToSet) error {

	if action.Name == "" {
		return errors.New("action name is empty")
	}
	if !utf8.ValidString(action.Name) {
		return errors.New("action name is not UTF-8 encoded")
	}
	if n := utf8.RuneCountInString(action.Name); n > 60 {
		return errors.New("action name is longer than 60 runes")
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

	if action.Mapping != nil {
		for left, right := range action.Mapping {
			if !types.IsValidPropertyName(left) {
				return fmt.Errorf("property name %q is not valid", left)
			}
			if !types.IsValidPropertyName(right) {
				return fmt.Errorf("property name %q is not valid", left)
			}
		}
	}

	if action.Transformation != nil {
		err := validateTransformation(action.Transformation)
		if err != nil {
			return err
		}
	}

	return nil
}

// ActionType represents an action type.
type ActionType struct {
	ID                   int
	Name                 string
	Description          string
	Schema               types.Type
	AdditionalProperties bool
	SuggestedFilter      ActionFilter
}

// ActionFilter represents a filter of an action.
type ActionFilter struct {
	Logical    string // "all" or "any"
	Conditions []ActionFilterCondition
}

// ActionFilterCondition represents a condition in an action filter.
type ActionFilterCondition struct {
	Property string // "Event Type", "Event Name", "User ID"...
	Operator string // "is", "is not", "exists", ...
	Value    string // "Track", "Page", ...
}
