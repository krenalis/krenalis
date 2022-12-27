//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b

package apis

import (
	"database/sql"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/types"
)

var AlreadyExist errors.Code = "AlreadyExist"

type DataTypes struct {
	*Workspace
	state *dataTypesState
}

// newDataTypes returns a new *DataTypes value.
func newDataTypes(ws *Workspace, state *dataTypesState) *DataTypes {
	return &DataTypes{Workspace: ws, state: state}
}

// A DataType represents a data type.
type DataType struct {
	name         string
	description  string
	schema       types.Schema
	schemaSource string
}

// An DataTypeInfo describes a data type as returned by Get and List.
type DataTypeInfo struct {
	Name         string
	Description  string
	Schema       types.Schema
	SchemaSource string
}

// Add adds a new data type with the given name, description and schema.
// description cannot be longer than 400 runes and schema cannot be longer than
// 65,535 runes.
//
// If the workplace does not exist, it returns an errors.NotFound error.
// If a data type with the same name already exists, it returns an
// errors.UnprocessableError error with code AlreadyExist and if the schema is
// not valid, it returns an errors.UnprocessableError error with code
// InvalidSchema.
func (this *DataTypes) Add(name, description, schema string) error {
	if !types.IsValidCustomTypeName(name) {
		return errors.BadRequest("name %q is not a valid data type name", name)
	}
	if utf8.RuneCountInString(name) > 120 {
		return errors.BadRequest("name %q is longer than 120 runes", name)
	}
	if utf8.RuneCountInString(description) > 400 {
		return errors.BadRequest("description is longer than 400 runes")
	}
	if schema == "" {
		return errors.BadRequest("schema is empty")
	}
	if utf8.RuneCountInString(schema) > 65535 {
		return errors.BadRequest("schema is longer than 65,535 runes")
	}
	n := addDataTypeNotification{
		Workspace:   this.id,
		Name:        name,
		Description: description,
		Schema:      json.RawMessage(schema),
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		_, err := types.ParseSchema(strings.NewReader(schema), dataTypeResolver(tx, n.Workspace))
		if err != nil {
			//return errors.UnprocessableData(InvalidSchema, "schema is not valid", errors.Data{"error": err})
			return errors.Unprocessable(InvalidSchema, "schema is not valid: %w", err)
		}
		_, err = tx.Exec("INSERT INTO data_types (workspace, name, description, schema) VALUES ($1, $2, $3, $4)",
			n.Workspace, n.Name, n.Description, schema)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				if postgres.ErrConstraintName(err) == "data_types_workspace_fkey" {
					err = errors.NotFound("workspace %d does not exist", n.Workspace)
				}
			} else if postgres.IsDuplicateKeyValue(err) {
				if postgres.ErrConstraintName(err) == "data_types_pkey" {
					err = errors.Unprocessable(AlreadyExist, "data type %s already exists", n.Name)
				}
			}
			return err
		}
		return tx.Notify(n)
	})
	return err
}

// Delete deletes the data type with the given name. If the type does not
// exist, it does nothing.
func (this *DataTypes) Delete(name string) error {
	if !types.IsValidCustomTypeName(name) {
		return errors.BadRequest("name %q is not a valid data type name", name)
	}
	if utf8.RuneCountInString(name) > 120 {
		return errors.BadRequest("name %s is longer than 120 runes", name)
	}
	n := &deleteDataTypeNotification{
		Workspace: this.id,
		Name:      name,
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		result, err := tx.Exec("DELETE FROM data_types WHERE workspace = $1 AND name = $2",
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

// Get returns an DataTypeInfo describing the data type with the given name.
//
// It returns an errors.NotFoundError error if the data type does not exist.
func (this *DataTypes) Get(name string) (*DataTypeInfo, error) {
	if !types.IsValidCustomTypeName(name) {
		return nil, errors.BadRequest("name %q is not a valid data type name", name)
	}
	if utf8.RuneCountInString(name) > 120 {
		return nil, errors.BadRequest("name %s is longer than 120 runes", name)
	}
	t, ok := this.state.Get(name)
	if !ok {
		return nil, errors.NotFound("data type %s does not exist", name)
	}
	info := DataTypeInfo{
		Name:         t.name,
		Description:  t.description,
		Schema:       t.schema,
		SchemaSource: t.schemaSource,
	}
	return &info, nil
}

// List returns a list of DataTypeInfo describing all data types.
// Unlike Info, Schema and SchemaSource are not meaningful.
func (this *DataTypes) List() ([]*DataTypeInfo, error) {
	dataTypes := this.state.List()
	infos := make([]*DataTypeInfo, len(dataTypes))
	for i, t := range dataTypes {
		infos[i] = &DataTypeInfo{
			Name:        t.name,
			Description: t.description,
		}
	}
	return infos, nil
}

// SetDescription sets the description of the data type with the given name.
// description cannot be longer than 400 runes.
//
// If the data type does not exist, it returns an errors.NotFoundError error.
func (this *DataTypes) SetDescription(name string, description string) error {
	if !types.IsValidCustomTypeName(name) {
		return errors.BadRequest("name %q is not a valid data type name", name)
	}
	if utf8.RuneCountInString(name) > 120 {
		return errors.BadRequest("name %s is longer than 120 runes", name)
	}
	if utf8.RuneCountInString(description) > 400 {
		return errors.BadRequest("description is longer than 400 runes")
	}
	n := setDataTypeDescriptionNotification{
		Workplace:   this.id,
		Name:        name,
		Description: description,
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		result, err := tx.Exec("UPDATE data_types SET description = $1 WHERE workspace = $2 AND name = $3",
			n.Description, n.Workplace, n.Name)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.NotFound("data type %s does not exist", n.Name)
		}
		return tx.Notify(n)
	})
	return err
}

// SetSchema sets the schema of the data type with the given name. schema
// cannot be longer than 65,535 runes.
//
// If the data type does not exist, it returns an errors.NotFoundError error.
// If the schema is not valid, it returns an errors.UnprocessableError error
// with code InvalidSchema.
func (this *DataTypes) SetSchema(name string, schema string) error {
	if !types.IsValidCustomTypeName(name) {
		return errors.BadRequest("name %q is not a valid data type name", name)
	}
	if utf8.RuneCountInString(name) > 120 {
		return errors.BadRequest("name %s is longer than 120 runes", name)
	}
	if schema == "" {
		return errors.BadRequest("schema is empty")
	}
	if utf8.RuneCountInString(schema) > 65535 {
		return errors.BadRequest("schema is longer than 65,535 runes")
	}
	n := setDataTypeSchemaNotification{
		Workspace: this.id,
		Name:      name,
		Schema:    json.RawMessage(schema),
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		_, err := types.ParseSchema(strings.NewReader(schema), dataTypeResolver(tx, n.Workspace))
		if err != nil {
			err2 := tx.QueryVoid("SELECT FROM data_types WHERE workspace = $1 AND name = $2", n.Workspace, n.Name)
			if err2 != nil {
				if err2 == sql.ErrNoRows {
					return errors.NotFound("data type %s does not exist", n.Name)
				}
				return err2
			}
			return errors.Unprocessable(InvalidSchema, "schema is not valid: %w", err)
		}
		result, err := tx.Exec("UPDATE data_types SET schema = $1 WHERE workspace = $2 AND name = $3",
			schema, n.Workspace, n.Name)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.NotFound("data type %s does not exist", n.Name)
		}
		return tx.Notify(n)
	})
	return err
}

// dataTypeResolver returns a resolver that resolve the data types.
func dataTypeResolver(tx *postgres.Tx, workspace int) types.Resolver {

	var schemaOf map[string]string
	var typeOf map[string]types.Type

	var resolve types.Resolver
	resolve = func(name string) (types.Type, error) {
		if schemaOf == nil {
			schemas := map[string]string{}
			err := tx.QueryScan(
				"SELECT name, schema FROM data_types WHERE workspace = $1", workspace,
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
