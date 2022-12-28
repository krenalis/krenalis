//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b

package apis

import (
	"database/sql"
	"encoding/json"
	"math"
	"strings"
	"unicode/utf8"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/types"
)

const maxReservedEventTypeID = 10

var (
	AlreadyExist  errors.Code = "AlreadyExist"
	TooManyEvents errors.Code = "TooManyEvents"
)

type Types struct {
	*Workspace
	state *typesState
}

// newTypes returns a new *Types value.
func newTypes(ws *Workspace, state *typesState) *Types {
	return &Types{Workspace: ws, state: state}
}

// A Type represents a type.
type Type struct {
	name        string
	description string
	definition  string
	event       uint8
	typ         types.Type
}

// A TypeInfo describes a type as returned by Get and List.
type TypeInfo struct {
	Name        string
	Description string
	Definition  string
	Type        types.Type
}

// Add adds a new type with the given name, description and definition.
// description cannot be longer than 400 runes and definition cannot be longer
// than 65,535 runes.
//
// If the workplace does not exist, it returns an errors.NotFound error.
// If a type with the same name already exists, it returns an
// errors.UnprocessableError error with code AlreadyExist and if the definition
// is not valid, it returns an errors.UnprocessableError error with code
// InvalidDefinition.
func (this *Types) Add(name, description, definition string) error {
	if !types.IsValidCustomTypeName(name) {
		return errors.BadRequest("name %q is not a valid type name", name)
	}
	if utf8.RuneCountInString(name) > 120 {
		return errors.BadRequest("name %q is longer than 120 runes", name)
	}
	if utf8.RuneCountInString(description) > 400 {
		return errors.BadRequest("description is longer than 400 runes")
	}
	if definition == "" {
		return errors.BadRequest("definition is empty")
	}
	if utf8.RuneCountInString(definition) > 65535 {
		return errors.BadRequest("definition is longer than 65,535 runes")
	}
	n := addTypeNotification{
		Workspace:   this.id,
		Name:        name,
		Description: description,
		Definition:  json.RawMessage(definition),
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		_, err := types.Parse(definition, typeResolver(tx, n.Workspace))
		if err != nil {
			return errors.Unprocessable(InvalidDefinition, "definition is not valid: %w", err)
		}
		if strings.HasSuffix(name, "_event") {
			n.Event = maxReservedEventTypeID + 1
			err := this.db.QueryScan(
				"SELECT event FROM types WHERE workspace = $1 AND event IS NOT NULL ORDER BY event", n.Workspace,
				func(rows *postgres.Rows) error {
					var err error
					var event uint8
					for rows.Next() {
						if err = rows.Scan(&event); err != nil {
							return err
						}
						if n.Event == event {
							n.Event++
						}
					}
					return nil
				})
			if err != nil {
				return err
			}
			if n.Event > math.MaxUint8 {
				return errors.Unprocessable(TooManyEvents, "there are already %d events", math.MaxUint8)
			}
		}
		_, err = tx.Exec("INSERT INTO types (workspace, name, description, definition, event)"+
			" VALUES ($1, $2, $3, $4, NULLIF($5, 0))", n.Workspace, n.Name, n.Description, definition, n.Event)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				if postgres.ErrConstraintName(err) == "types_workspace_fkey" {
					err = errors.NotFound("workspace %d does not exist", n.Workspace)
				}
			} else if postgres.IsDuplicateKeyValue(err) {
				if postgres.ErrConstraintName(err) == "types_pkey" {
					err = errors.Unprocessable(AlreadyExist, "type %s already exists", n.Name)
				}
			}
			return err
		}
		return tx.Notify(n)
	})
	return err
}

// Delete deletes the type with the given name. If the type does not exist, it
// does nothing.
func (this *Types) Delete(name string) error {
	if !types.IsValidCustomTypeName(name) {
		return errors.BadRequest("name %q is not a valid type name", name)
	}
	if utf8.RuneCountInString(name) > 120 {
		return errors.BadRequest("name %s is longer than 120 runes", name)
	}
	n := &deleteTypeNotification{
		Workspace: this.id,
		Name:      name,
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		result, err := tx.Exec("DELETE FROM types WHERE workspace = $1 AND name = $2", n.Workspace, n.Name)
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

// Get returns an TypeInfo describing the type with the given name.
//
// It returns an errors.NotFoundError error if the type does not exist.
func (this *Types) Get(name string) (*TypeInfo, error) {
	if !types.IsValidCustomTypeName(name) {
		return nil, errors.BadRequest("name %q is not a valid type name", name)
	}
	if utf8.RuneCountInString(name) > 120 {
		return nil, errors.BadRequest("name %s is longer than 120 runes", name)
	}
	t, ok := this.state.Get(name)
	if !ok {
		return nil, errors.NotFound("type %s does not exist", name)
	}
	info := TypeInfo{
		Name:        t.name,
		Description: t.description,
		Definition:  t.definition,
		Type:        t.typ,
	}
	return &info, nil
}

// List returns a list of TypeInfo describing all types.
// Unlike Info, Definition and Type are not meaningful.
func (this *Types) List() ([]*TypeInfo, error) {
	types := this.state.List()
	infos := make([]*TypeInfo, len(types))
	for i, t := range types {
		infos[i] = &TypeInfo{
			Name:        t.name,
			Description: t.description,
		}
	}
	return infos, nil
}

// SetDescription sets the description of the type with the given name.
// description cannot be longer than 400 runes.
//
// If the type does not exist, it returns an errors.NotFoundError error.
func (this *Types) SetDescription(name string, description string) error {
	if !types.IsValidCustomTypeName(name) {
		return errors.BadRequest("name %q is not a valid type name", name)
	}
	if utf8.RuneCountInString(name) > 120 {
		return errors.BadRequest("name %s is longer than 120 runes", name)
	}
	if utf8.RuneCountInString(description) > 400 {
		return errors.BadRequest("description is longer than 400 runes")
	}
	n := setTypeDescriptionNotification{
		Workplace:   this.id,
		Name:        name,
		Description: description,
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		result, err := tx.Exec("UPDATE types SET description = $1 WHERE workspace = $2 AND name = $3",
			n.Description, n.Workplace, n.Name)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.NotFound("type %s does not exist", n.Name)
		}
		return tx.Notify(n)
	})
	return err
}

// SetDefinition sets the definition of the type with the given name.
// definition cannot be longer than 65,535 runes.
//
// If the type does not exist, it returns an errors.NotFoundError error.
// If the definition is not valid, it returns an errors.UnprocessableError
// error with code InvalidDefinition.
func (this *Types) SetDefinition(name string, definition string) error {
	if !types.IsValidCustomTypeName(name) {
		return errors.BadRequest("name %q is not a valid type name", name)
	}
	if utf8.RuneCountInString(name) > 120 {
		return errors.BadRequest("name %s is longer than 120 runes", name)
	}
	if definition == "" {
		return errors.BadRequest("definition is empty")
	}
	if utf8.RuneCountInString(definition) > 65535 {
		return errors.BadRequest("definition is longer than 65,535 runes")
	}
	n := setTypeDefinitionNotification{
		Workspace:  this.id,
		Name:       name,
		Definition: json.RawMessage(definition),
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		_, err := types.Parse(definition, typeResolver(tx, n.Workspace))
		if err != nil {
			err2 := tx.QueryVoid("SELECT FROM types WHERE workspace = $1 AND name = $2", n.Workspace, n.Name)
			if err2 != nil {
				if err2 == sql.ErrNoRows {
					return errors.NotFound("type %s does not exist", n.Name)
				}
				return err2
			}
			return errors.Unprocessable(InvalidDefinition, "definition is not valid: %w", err)
		}
		result, err := tx.Exec("UPDATE types SET schema = $1 WHERE workspace = $2 AND name = $3",
			definition, n.Workspace, n.Name)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.NotFound("type %s does not exist", n.Name)
		}
		return tx.Notify(n)
	})
	return err
}

// typeResolver returns a resolver that resolve the types.
func typeResolver(tx *postgres.Tx, workspace int) types.Resolver {

	var definitionOf map[string]string
	var typeOf map[string]types.Type

	var resolve types.Resolver
	resolve = func(name string) (types.Type, error) {
		if definitionOf == nil {
			definitions := map[string]string{}
			err := tx.QueryScan(
				"SELECT name, schema FROM types WHERE workspace = $1", workspace,
				func(rows *sql.Rows) error {
					var name, definition string
					for rows.Next() {
						if err := rows.Scan(&name, &definition); err != nil {
							return err
						}
						definitions[name] = definition
					}
					return nil
				})
			if err != nil {
				return types.Type{}, err
			}
			definitionOf = definitions
			typeOf = map[string]types.Type{}
		}
		if typ, ok := typeOf[name]; ok {
			return typ, nil
		}
		if schema, ok := definitionOf[name]; ok {
			t, err := types.Parse(schema, resolve)
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
