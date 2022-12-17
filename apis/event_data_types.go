//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b

package apis

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"unicode/utf8"

	"chichi/apis/postgres"
	"chichi/apis/types"
)

var (
	ErrEventDataTypeAlreadyExist = errors.New("event data type already exist")
	ErrEventDataTypeNotFound     = errors.New("event data type does not exist")
)

type EventDataTypes struct {
	*Workspace
	state eventDataTypesState
}

// newEventDataTypes returns a new *EventDataTypes value.
func newEventDataTypes(ws *Workspace) *EventDataTypes {
	return &EventDataTypes{Workspace: ws, state: eventDataTypesState{names: map[string]*EventDataType{}}}
}

type eventDataTypesState struct {
	sync.Mutex
	names map[string]*EventDataType
}

// A EventDataType represents a defined event type.
type EventDataType struct {
	name         string
	description  string
	schema       types.Schema
	schemaSource string
}

// An EventDataTypeInfo describes an event data type as returned by Get and
// List.
type EventDataTypeInfo struct {
	Name         string
	Description  string
	Schema       types.Schema
	SchemaSource string
}

// Add adds a new data type with the given name, description and schema.
//
// If a data type with the same name already exist, it returns the
// ErrEventDataTypeAlreadyExist error. If the workspace does not exist,
// it returns the ErrWorkspaceNotFound error.
func (this *EventDataTypes) Add(name, description, schema string) error {
	if !types.IsValidCustomTypeName(name) {
		return errors.New("name is not a valid type name")
	}
	if utf8.RuneCountInString(name) > 120 {
		return errors.New("name cannot be longer than 120 characters")
	}
	if utf8.RuneCountInString(description) > 400 {
		return errors.New("description cannot be longer than 400 characters")
	}
	if schema == "" {
		return errors.New("schema cannot be empty")
	}
	if utf8.RuneCountInString(schema) > 65535 {
		return errors.New("schema cannot be longer than 65535 characters")
	}
	n := addEventDataTypeNotification{
		Workspace:   this.id,
		Name:        name,
		Description: description,
		Schema:      json.RawMessage(schema),
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		_, err := types.ParseSchema(strings.NewReader(schema), eventDataTypeResolver(tx, n.Workspace))
		if err != nil {
			return &InvalidSchema{err}
		}
		_, err = tx.Exec("INSERT INTO event_data_types (workspace, name, description, schema) VALUES ($1, $2, $3, $4)",
			n.Workspace, n.Name, n.Description, schema)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) && postgres.ErrConstraintName(err) == "event_data_types_workspace_fkey" {
				err = ErrWorkspaceNotFound
			} else if postgres.IsDuplicateKeyValue(err) && postgres.ErrConstraintName(err) == "event_data_types_pkey" {
				err = ErrEventDataTypeAlreadyExist
			}
			return err
		}
		return tx.Notify(n)
	})
	return err
}

// Delete deletes the data type with the given name. If the type does not
// exist, it does nothing.
func (this *EventDataTypes) Delete(name string) error {
	if !types.IsValidCustomTypeName(name) {
		return errors.New("name is not a valid custom type name")
	}
	if utf8.RuneCountInString(name) > 120 {
		return errors.New("name cannot be longer than 120 characters")
	}
	n := &deleteEventDataTypeNotification{
		Workspace: this.id,
		Name:      name,
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		result, err := tx.Exec("DELETE FROM event_data_types WHERE workspace = $1 AND name = $2",
			n.Workspace, n.Name)
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

var errEventDataTypeNotFound = errors.New("event data type does not exist")

// get returns the data type with the given name.
//
// It returns the errEventDataTypeNotFound error if the type does not exist.
func (this *EventDataTypes) get(name string) (*EventDataType, error) {
	this.state.Lock()
	t, ok := this.state.names[name]
	this.state.Unlock()
	if !ok {
		return nil, errEventDataTypeNotFound
	}
	return t, nil
}

// Get returns an EventDataTypeInfo describing the data type with the given
// name.
//
// It returns the ErrEventDataTypeNotFound error if the type does not exist.
func (this *EventDataTypes) Get(name string) (*EventDataTypeInfo, error) {
	if !types.IsValidCustomTypeName(name) {
		return nil, errors.New("name is not a valid data type name")
	}
	if utf8.RuneCountInString(name) > 120 {
		return nil, errors.New("name cannot be longer than 120 characters")
	}
	t, err := this.get(name)
	if err != nil {
		return nil, ErrEventDataTypeNotFound
	}
	info := EventDataTypeInfo{
		Name:         t.name,
		Description:  t.description,
		Schema:       t.schema,
		SchemaSource: t.schemaSource,
	}
	return &info, nil
}

// list returns all the data types.
func (this *EventDataTypes) list() []*EventDataType {
	this.state.Lock()
	dataTypes := make([]*EventDataType, len(this.state.names))
	i := 0
	for _, t := range this.state.names {
		dataTypes[i] = t
		i++
	}
	this.state.Unlock()
	return dataTypes
}

// List returns a list of EventDataTypeInfo describing all data types.
// Unlike Info, Schema and SchemaSource are not meaningful.
func (this *EventDataTypes) List() ([]*EventDataTypeInfo, error) {
	dataTypes := this.list()
	infos := make([]*EventDataTypeInfo, len(dataTypes))
	for i, t := range dataTypes {
		infos[i] = &EventDataTypeInfo{
			Name:        t.name,
			Description: t.description,
		}
	}
	return infos, nil
}

// SetDescription sets the description of the event data type with the given
// name.
//
// It returns the ErrEventDataTypeNotFound error if the type does not exist.
func (this *EventDataTypes) SetDescription(name string, description string) error {
	if !types.IsValidCustomTypeName(name) {
		return errors.New("name is not a valid custom type name")
	}
	if utf8.RuneCountInString(name) > 120 {
		return errors.New("name cannot be longer than 120 characters")
	}
	if utf8.RuneCountInString(description) > 400 {
		return errors.New("description cannot be longer than 400 characters")
	}
	n := setEventDataTypeDescriptionNotification{
		Workplace:   this.id,
		Name:        name,
		Description: description,
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		result, err := tx.Exec("UPDATE event_data_types SET description = $1 WHERE workspace = $2 AND name = $3",
			n.Description, n.Workplace, n.Name)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return ErrEventDataTypeNotFound
		}
		return tx.Notify(n)
	})
	return err
}

// SetSchema sets the schema of the event data type with the given name.
//
// It returns the ErrEventDataTypeNotFound error if the type does not exist,
// and an InvalidSchema if the schema is not valid.
func (this *EventDataTypes) SetSchema(name string, schema string) error {
	if !types.IsValidCustomTypeName(name) {
		return errors.New("name is not a valid custom type name")
	}
	if utf8.RuneCountInString(name) > 120 {
		return errors.New("name cannot be longer than 120 characters")
	}
	if schema == "" {
		return errors.New("invalid empty schema")
	}
	if utf8.RuneCountInString(schema) > 65535 {
		return errors.New("schema cannot be longer than 65535 characters")
	}
	n := setEventDataTypeSchemaNotification{
		Workspace: this.id,
		Name:      name,
		Schema:    json.RawMessage(schema),
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		_, err := types.ParseSchema(strings.NewReader(schema), eventDataTypeResolver(tx, n.Workspace))
		if err != nil {
			err2 := tx.QueryVoid("SELECT FROM event_data_types WHERE workspace = $1 AND name = $2", n.Workspace, n.Name)
			if err2 != nil {
				if err2 == sql.ErrNoRows {
					return ErrEventDataTypeNotFound
				}
				return err2
			}
			return &InvalidSchema{err}
		}
		result, err := tx.Exec("UPDATE event_data_types SET schema = $1 WHERE workspace = $2 AND name = $3",
			schema, n.Workspace, n.Name)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return ErrEventDataTypeNotFound
		}
		return tx.Notify(n)
	})
	return err
}

// eventDataTypeResolver returns a resolver that resolve the event data types.
func eventDataTypeResolver(tx *postgres.Tx, workspace int) types.Resolver {

	var schemaOf map[string]string
	var typeOf map[string]types.Type

	var resolve types.Resolver
	resolve = func(name string) (types.Type, error) {
		if schemaOf == nil {
			schemas := map[string]string{}
			err := tx.QueryScan(
				"SELECT name, schema FROM event_data_types WHERE workspace = $1", workspace,
				func(rows *sql.Rows) error {
					var name, schema string
					for rows.Next() {
						if err := rows.Scan(&name, &schema); err != nil {
							return err
						}
						schemas[name] = schema
					}
					return nil
				})
			if err != nil {
				return types.Type{}, err
			}
			schemaOf = schemas
			typeOf = map[string]types.Type{}
		}
		if typ, ok := typeOf[name]; ok {
			return typ, nil
		}
		if schema, ok := schemaOf[name]; ok {
			t, err := types.ParseType(schema, resolve)
			if err != nil {
				return types.Type{}, err
			}
			typeOf[name] = t
			return t, nil
		}
		return types.Type{}, nil
	}

	return resolve
}
