//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b

package apis

import (
	"encoding/json"
	"math"
	"strings"
	"unicode/utf8"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/types"
)

const maxReservedEventTypeID = 10

var TooManyTypes errors.Code = "TooManyTypes"

type EventTypes struct {
	*Workspace
	state *eventTypesState
}

// newEventTypes returns a new *EventTypes value.
func newEventTypes(ws *Workspace, state *eventTypesState) *EventTypes {
	return &EventTypes{Workspace: ws, state: state}
}

// An EventType represents an event type.
type EventType struct {
	id           int
	name         string
	description  string
	predefined   bool
	schema       types.Schema
	schemaSource string
}

// An EventTypeInfo describes an event type as returned by Get and List.
type EventTypeInfo struct {
	ID           int
	Name         string
	Description  string
	Predefined   bool
	Schema       types.Schema
	SchemaSource string
}

// Add adds a new type with the given name. description cannot be longer than
// 400 runes and schema cannot be longer than 65,535 runes. If schema is empty,
// the type will have no schema.
//
// If the workspace does not exist, it returns an errors.NotFoundError error.
// If the schema is not valid, a type with the same name already exists, or
// there are already too many listeners, it returns an
// errors.UnprocessableError error with code InvalidSchema, AlreadyExist, and
// TooManyTypes respectively.
func (this *EventTypes) Add(name, description, schema string) (int, error) {
	if !types.IsValidCustomTypeName(name) {
		return 0, errors.BadRequest("name %q is not a valid type name", name)
	}
	if utf8.RuneCountInString(name) > 120 {
		return 0, errors.BadRequest("name %s is longer than 120 runes", name)
	}
	if utf8.RuneCountInString(description) > 400 {
		return 0, errors.BadRequest("description is longer than 400 runes")
	}
	if utf8.RuneCountInString(schema) > 65535 {
		return 0, errors.BadRequest("schema is longer than 65,535 runes")
	}
	n := addEventTypeNotification{
		Workspace:   this.id,
		ID:          maxReservedEventTypeID + 1,
		Name:        name,
		Description: description,
		Schema:      json.RawMessage(schema),
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		if schema != "" {
			_, err := types.ParseSchema(strings.NewReader(schema), eventDataTypeResolver(tx, n.Workspace))
			if err != nil {
				return errors.Unprocessable(InvalidSchema, "schema is not valid: %w", err)
			}
		}
		err := this.db.QueryScan(
			"SELECT id, name FROM event_types WHERE workspace = $1 ORDER BY id", n.Workspace,
			func(rows *postgres.Rows) error {
				var err error
				var id int
				var name string
				for rows.Next() {
					if err = rows.Scan(&id, &name); err != nil {
						return err
					}
					if n.Name == name {
						return errors.Unprocessable(AlreadyExist, "event type %s already exists", name)
					}
					if n.ID == id {
						n.ID++
					}
				}
				return nil
			})
		if err != nil {
			return err
		}
		if n.ID > math.MaxUint8 {
			return errors.Unprocessable(TooManyTypes, "there are already %d types", math.MaxUint8)
		}
		_, err = this.db.Exec(
			"INSERT INTO event_types (workspace, id, name, description, schema) VALUES ($1, $2, $3, $4, $5)",
			n.Workspace, n.ID, n.Name, n.Description, schema)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				if postgres.ErrConstraintName(err) == "event_types_workspace_fkey" {
					err = errors.NotFound("workspace %d does not exist", n.Workspace)
				}
			}
			return err
		}
		return tx.Notify(n)
	})
	if err != nil {
		return 0, err
	}
	return n.ID, nil
}

// DeleteType deletes the event type with identifier id. If the type does not
// exist, it does nothing.
func (this *EventTypes) DeleteType(id int) error {
	if id < 1 || id > math.MaxUint8 {
		return errors.BadRequest("event type identifier %d is not valid", id)
	}
	if id <= maxReservedEventTypeID {
		return errors.BadRequest("event type %d cannot be deleted, it's predefined", id)
	}
	n := deleteEventTypeNotification{
		Workspace: this.id,
		ID:        id,
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		result, err := this.db.Exec("UPDATE event_types\nSET name = '', schema = '', deleted = true\n"+
			"WHERE workspace = $1 AND id = $2", n.Workspace, n.ID)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil || affected == 0 {
			return err
		}
		return tx.Notify(n)
	})
	return err
}

// Get returns an EventTypeInfo describing the type with identifier id.
//
// If the type does not exist, it returns an errors.NotFoundError error.
func (this *EventTypes) Get(id int) (*EventTypeInfo, error) {
	if id < 1 || id > types.MaxUInt8 {
		return nil, errors.BadRequest("event type identifier %d is not valid", id)
	}
	t, ok := this.state.Get(id)
	if !ok {
		return nil, errors.NotFound("event type %d does not exist", id)
	}
	info := EventTypeInfo{
		ID:           t.id,
		Name:         t.name,
		Description:  t.description,
		Predefined:   t.id <= maxReservedEventTypeID,
		Schema:       t.schema,
		SchemaSource: t.schemaSource,
	}
	return &info, nil
}

// List returns a list of EventTypeInfo describing all event types.
// Unlike Get, Schema and SchemaSource are not meaningful.
func (this *EventTypes) List() []*EventTypeInfo {
	eventTypes := this.state.List()
	infos := make([]*EventTypeInfo, len(eventTypes))
	for i, t := range eventTypes {
		infos[i] = &EventTypeInfo{
			ID:          t.id,
			Name:        t.name,
			Description: t.description,
			Predefined:  t.id <= maxReservedEventTypeID,
		}
	}
	return infos
}

// SetDescription sets the description of the event type with the given
// identifier. description cannot be longer than 400 runes.
//
// If the type does not exist, it returns an errors.NotFoundError error.
func (this *EventTypes) SetDescription(id int, description string) error {
	if id < 1 || id > types.MaxUInt8 {
		return errors.BadRequest("event type identifier %d is not valid", id)
	}
	if utf8.RuneCountInString(description) > 400 {
		return errors.BadRequest("description is longer than 400 runes")
	}
	n := setEventTypeDescriptionNotification{
		Workplace:   this.id,
		ID:          id,
		Description: description,
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		result, err := tx.Exec("UPDATE event_types SET description = $1 WHERE workspace = $2 AND id = $3",
			n.Description, n.Workplace, n.ID)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.NotFound("event type %d does not exist", id)
		}
		return tx.Notify(n)
	})
	return err
}

// SetSchema sets the schema of the event type with identifier id. schema
// cannot be longer than 65,535 runes. If schema is empty, the type will have
// no schema.
//
// If the type does not exist, it returns an errors.NotFoundError error. If the
// schema is not valid, it returns the errors.UnprocessableError error with
// code InvalidSchema.
func (this *EventTypes) SetSchema(id int, schema string) error {
	if id < 1 || id > types.MaxUInt8 {
		return errors.BadRequest("event type identifier %d is not valid", id)
	}
	if utf8.RuneCountInString(schema) > 65535 {
		return errors.BadRequest("schema is longer than 65,535 runes")
	}
	n := setEventTypeSchemaNotification{
		Workspace: this.id,
		ID:        id,
		Schema:    json.RawMessage(schema),
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		if schema != "" {
			_, err := types.ParseSchema(strings.NewReader(schema), eventDataTypeResolver(tx, n.Workspace))
			if err != nil {
				return errors.Unprocessable(InvalidSchema, "schema is not valid: %w", err)
			}
		}
		result, err := tx.Exec("UPDATE event_types SET schema = $1 WHERE workspace = $2 AND id = $3",
			schema, n.Workspace, n.ID)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.NotFound("event type %d does not exist", id)
		}
		return tx.Notify(n)
	})
	return err
}
