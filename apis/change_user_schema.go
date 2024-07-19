//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package apis

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/meergo/meergo/apis/datastore"
	"github.com/meergo/meergo/apis/datastore/diffschemas"
	"github.com/meergo/meergo/apis/errors"
	"github.com/meergo/meergo/apis/postgres"
	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/types"
)

// ChangeUserSchema changes the user schema and the primary sources of the
// workspace. schema must be a valid schema.
//
// The properties within schema must meet the following requirements:
//
//   - properties with Array type cannot have elements of type Array, Object, or
//     Map;
//   - properties with Map type cannot have elements of type Array, Object, or
//     Map;
//   - properties cannot be nullable (as the NULL value of a data warehouse
//     column represents the fact that there is no value for that column);
//   - properties cannot specify a placeholder;
//   - properties cannot be required;
//   - properties cannot specify a role.
//
// Moreover, schema cannot contain conflicting properties, meaning properties
// whose representations as columns in the data warehouse would have the same
// column name.
//
// rePaths is a mapping containing the renamed property paths, where the key is
// the new property path and its value is the old property path. In case of new
// properties created with the same name of already existent properties, the
// value must be the untyped nil. rePaths cannot contain keys with the same path
// as their value. Any property path which does not refer to changed properties
// is ignored.
//
// It returns an errors.UnprocessableError error with code:
//
//   - ConnectionNotExist, if a connections used as primary source does not
//     exist.
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
//   - InspectionMode, if the data warehouse is in inspection mode.
//   - InvalidSchemaChange, if the schema change is invalid.
//   - NoWarehouse, if the workspace does not have a data warehouse.
func (this *Workspace) ChangeUserSchema(ctx context.Context, schema types.Type, primarySources map[string]int, rePaths map[string]any) error {
	this.apis.mustBeOpen()
	if primarySources == nil {
		primarySources = map[string]int{}
	}
	if !schema.Valid() {
		return errors.BadRequest("schema must be valid")
	}
	if schema.Kind() != types.ObjectKind {
		return errors.BadRequest("expected schema with kind Object, got %s", schema.Kind())
	}
	if err := validatePrimarySources(schema, primarySources); err != nil {
		return errors.BadRequest("primary sources are not valid: %w", err)
	}
	if err := validateRePaths(rePaths); err != nil {
		return errors.BadRequest("invalid rePaths: %s", err)
	}

	if err := checkAllowedPropertyUserSchema(schema); err != nil {
		return errors.BadRequest("%s", err)
	}

	if err := datastore.CheckConflictingProperties("users", schema); err != nil {
		return errors.BadRequest("%s", err)
	}

	operations, err := diffschemas.Diff(this.workspace.UserSchema, schema, rePaths, "")
	if err != nil {
		return errors.Unprocessable(InvalidSchemaChange, "cannot change the schema as specified: %s", err)
	}

	for _, s := range primarySources {
		source, ok := this.workspace.Connection(s)
		if !ok {
			return errors.Unprocessable(ConnectionNotExist, "primary source %d does not exist", s)
		}
		if source.Role != state.Source {
			return errors.BadRequest("primary source %d is not a source connection", s)
		}
		if !source.Connector().Targets.Contains(state.Users) {
			return errors.BadRequest("primary source %d does not support Users target", s)
		}
	}

	if this.store == nil {
		return errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", this.workspace.ID)
	}

	// Update the database and send the notification.
	n := state.SetWorkspaceUserSchema{
		Workspace:      this.ID,
		UserSchema:     schema,
		PrimarySources: primarySources,
	}
	schemaJSON, err := json.Marshal(n.UserSchema)
	if err != nil {
		return err
	}

	// Build the query to insert the primary paths.
	var insertPrimarySources string
	var paths []any
	if len(primarySources) > 0 {
		i := 0
		var b strings.Builder
		for path, source := range primarySources {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteByte('(')
			b.WriteString(strconv.Itoa(source))
			b.WriteString(",$")
			b.WriteString(strconv.Itoa(i + 1))
			b.WriteString(")")
			paths = append(paths, path)
			i++
		}
		insertPrimarySources = "INSERT INTO user_schema_primary_sources (source, path) VALUES " + b.String()
	}

	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		// Update the schema.
		_, err := tx.Exec(ctx, "UPDATE workspaces SET user_schema = $1 WHERE id = $2", schemaJSON, n.Workspace)
		if err != nil {
			return err
		}
		// Update the primary sources.
		_, err = tx.Exec(ctx, "DELETE FROM user_schema_primary_sources s USING connections c\n"+
			"WHERE c.workspace = $1 AND s.source = c.id", n.Workspace)
		if err != nil {
			return err
		}
		if insertPrimarySources != "" {
			_, err = tx.Exec(ctx, insertPrimarySources, paths...)
			if err != nil {
				if postgres.IsForeignKeyViolation(err) && postgres.ErrConstraintName(err) == "user_schema_primary_sources_source_fkey" {
					err = errors.Unprocessable(ConnectionNotExist, "a primary source does not exist")
				}
				return err
			}
		}
		return tx.Notify(ctx, n)
	})
	if err != nil {
		return err
	}

	// Alter the schema on the data warehouse.
	//
	// This must also be called even if operations is empty, as it is still
	// necessary to recreate the views (for example in the case where only the
	// ordering of properties has been changed).
	//
	err = this.store.AlterSchema(ctx, schema, operations)
	if err != nil {
		if err == datastore.ErrInspectionMode {
			return errors.Unprocessable(InspectionMode, "data warehouse is in inspection mode")
		}
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			return errors.Unprocessable(DataWarehouseFailed, "data warehouse has returned an error: %w", err.Err)
		}
		if err, ok := err.(datastore.UnsupportedAlterSchemaErr); ok {
			return errors.Unprocessable(InvalidSchemaChange, "cannot apply the schema change: %s", err)
		}
		return err
	}

	return nil
}

// ChangeUserSchemaQueries returns the queries that would be executed changing
// the user schema to schema.
//
// See the documentation of ChangeUserSchema for more details about this method.
//
// It returns an errors.UnprocessableError error with code:
//   - NoWarehouse, if the workspace does not have a data warehouse.
//   - InvalidSchemaChange, if the schema change is invalid.
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
func (this *Workspace) ChangeUserSchemaQueries(ctx context.Context, schema types.Type, rePaths map[string]any) ([]string, error) {
	this.apis.mustBeOpen()
	if !schema.Valid() {
		return nil, errors.BadRequest("schema must be valid")
	}
	if schema.Kind() != types.ObjectKind {
		return nil, errors.BadRequest("expected schema with kind Object, got %s", schema.Kind())
	}
	if err := validateRePaths(rePaths); err != nil {
		return nil, errors.BadRequest("invalid rePaths: %s", err)
	}
	if err := checkAllowedPropertyUserSchema(schema); err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	if err := datastore.CheckConflictingProperties("users", schema); err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	operations, err := diffschemas.Diff(this.workspace.UserSchema, schema, rePaths, "")
	if err != nil {
		return nil, errors.Unprocessable(InvalidSchemaChange, "cannot change the schema as specified: %s", err)
	}
	if this.store == nil {
		return nil, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", this.workspace.ID)
	}
	queries, err := this.store.AlterSchemaQueries(ctx, schema, operations)
	if err != nil {
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			return nil, errors.Unprocessable(DataWarehouseFailed, "data warehouse has returned an error: %w", err.Err)
		}
		if err, ok := err.(datastore.UnsupportedAlterSchemaErr); ok {
			return nil, errors.Unprocessable(InvalidSchemaChange, "cannot get the queries for the schema change: %s", err)
		}
		return nil, err
	}
	return queries, nil
}

// checkAllowedPropertyUserSchema checks the given user schema and returns
// error in case it contains properties which are not allowed in data warehouse
// user schemas.
func checkAllowedPropertyUserSchema(schema types.Type) error {
	for _, p := range schema.Properties() {
		if isMetaProperty(p.Name) {
			return errors.New("user schema cannot have meta properties")
		}
		if p.Placeholder != "" {
			return errors.New("user schema properties cannot have a placeholder")
		}
		if p.Role != types.BothRole {
			return errors.New("user schema properties can only have the Both role")
		}
		if p.CreateRequired {
			return errors.New("user schema properties cannot be required for creation")
		}
		if p.UpdateRequired {
			return errors.New("user schema properties cannot be required for the update")
		}
		if !p.ReadOptional {
			return errors.New("user schema properties must be optional")
		}
		if p.Nullable {
			return fmt.Errorf("user schema properties cannot be nullable")
		}
		switch p.Type.Kind() {
		// An Array or Map element type cannot be an Array, Object, or Map.
		case types.ArrayKind, types.MapKind:
			k := p.Type.Elem().Kind()
			if k == types.ArrayKind || k == types.ObjectKind || k == types.MapKind {
				return fmt.Errorf("user schema properties cannot have type '%s(%s)'", p.Type.Kind(), k)
			}
		case types.ObjectKind:
			err := checkAllowedPropertyUserSchema(p.Type)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// validatePrimarySources validates a primary source returning an error if it is
// not valid.
func validatePrimarySources(schema types.Type, primarySources map[string]int) error {
	for path, source := range primarySources {
		_, err := types.PropertyByPath(schema, path)
		if err != nil {
			return err
		}
		if source < 1 || source > maxInt32 {
			return fmt.Errorf("primary source identifier %d is not valid", source)
		}
	}
	return nil
}

func validateRePaths(rePaths map[string]any) error {
	for new, old := range rePaths {
		if !types.IsValidPropertyPath(new) {
			return fmt.Errorf("invalid property path: %q", new)
		}
		switch old := old.(type) {
		case string:
			if !types.IsValidPropertyPath(old) {
				return fmt.Errorf("invalid property path: %q", new)
			}
			if new == old {
				return fmt.Errorf("rePath key cannot match with its value")
			}
			if strings.Contains(old, ".") {
				oldParts := strings.Split(old, ".")
				oldPrefix := oldParts[:len(oldParts)-1]
				newParts := strings.Split(new, ".")
				newPrefix := newParts[:len(newParts)-1]
				if !slices.Equal(oldPrefix, newPrefix) {
					return fmt.Errorf("rePath contains a renamed property whose path is different")
				}
			}
		case nil:
			// Ok.
		default:
			return fmt.Errorf("unexpected value of type %T", old)
		}
	}
	return nil
}
