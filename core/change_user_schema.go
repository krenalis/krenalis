//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package core

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/datastore"
	"github.com/meergo/meergo/core/datastore/diffschemas"
	"github.com/meergo/meergo/core/db"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

// IdentityResolutionSettings returns the identity resolution settings of the
// workspace.
func (this *Workspace) IdentityResolutionSettings() (bool, []string) {
	this.core.mustBeOpen()
	ws := this.workspace
	return ws.ResolveIdentitiesOnBatchImport, ws.Identifiers
}

// PreviewUserSchemaUpdate previews a user schema update and returns the queries
// that would be executed to update the user schema of the workspace, without
// making any actual changes to the data or the schema.
//
// See the documentation of UpdateUserSchema for more details about this method.
//
// It returns an errors.UnprocessableError error with code InvalidSchemaUpdate
// if the schema update is invalid.
func (this *Workspace) PreviewUserSchemaUpdate(ctx context.Context, schema types.Type, rePaths map[string]any) ([]string, error) {
	this.core.mustBeOpen()
	if !schema.Valid() {
		return nil, errors.BadRequest("schema must be valid")
	}
	if schema.Kind() != types.ObjectKind {
		return nil, errors.BadRequest("expected schema with kind object, got %s", schema.Kind())
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
		return nil, errors.Unprocessable(InvalidSchemaUpdate, "cannot update the schema as specified: %s", err)
	}
	queries, err := this.store.PreviewUserSchemaUpdate(ctx, schema, operations)
	if err != nil {
		if err, ok := err.(*datastore.UnavailableError); ok {
			return nil, errors.Unavailable("%s", err)
		}
		return nil, err
	}
	return queries, nil
}

// UpdateUserSchema updates the user schema and the primary sources of the
// workspace. schema must be a valid schema.
//
// The properties within user schema cannot specify a placeholder, cannot be
// required for creation or update, must be read optional and cannot be
// nullable; there are also limits on types, which are documented in
// "datastore/README.md".
//
// primarySources cannot specify a primary source for a property which has kind
// object or array.
//
// Moreover, schema cannot contain conflicting properties, meaning properties
// whose representations as columns in the data warehouse would have the same
// column name.
//
// rePaths is a mapping containing the renamed property paths, where the key is
// the new property path and its value is the old property path. In case of new
// properties created with the same name of already existent properties, the
// value must be the untyped nil. rePaths cannot contain keys with the same path
// as their value. Any property path which does not refer to updated properties
// is ignored.
//
// It returns an errors.UnprocessableError error with code:
//
//   - AlterSchemaInProgress, if an alter schema operation is already in
//     progress on the warehouse.
//   - ConnectionNotExist, if a connection used as primary source does not
//     exist.
//   - IdentityResolutionInProgress, if an Identity Resolution is currently in
//     progress on the warehouse.
//   - InspectionMode, if the data warehouse is in inspection mode.
//   - InvalidSchemaUpdate, if the schema update is invalid.
func (this *Workspace) UpdateUserSchema(ctx context.Context, schema types.Type, primarySources map[string]int, rePaths map[string]any) error {
	this.core.mustBeOpen()
	if primarySources == nil {
		primarySources = map[string]int{}
	}
	if !schema.Valid() {
		return errors.BadRequest("schema must be valid")
	}
	if schema.Kind() != types.ObjectKind {
		return errors.BadRequest("expected schema with kind object, got %s", schema.Kind())
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
		return errors.Unprocessable(InvalidSchemaUpdate, "cannot update the schema as specified: %s", err)
	}

	for _, s := range primarySources {
		source, ok := this.workspace.Connection(s)
		if !ok {
			return errors.Unprocessable(ConnectionNotExist, "primary source %d does not exist", s)
		}
		if source.Role != state.Source {
			return errors.BadRequest("primary source %d is not a source connection", s)
		}
		if !source.Connector().SourceTargets.Contains(state.Users) {
			return errors.BadRequest("primary source %d does not support Users target", s)
		}
	}

	// Update the identifiers.
	identifiers := make([]string, 0, len(this.workspace.Identifiers))
Identifiers:
	for _, identifier := range this.workspace.Identifiers {
		for _, operation := range operations {
			if operation.Operation == meergo.OperationAddColumn {
				continue
			}
			if path := strings.ReplaceAll(operation.Column, "_", "."); path != identifier {
				continue
			}
			if operation.Operation == meergo.OperationRenameColumn {
				identifiers = append(identifiers, strings.ReplaceAll(operation.NewColumn, "_", "."))
			}
			continue Identifiers
		}
		identifiers = append(identifiers, identifier)
	}

	// Update the database and send the notification.
	n := state.UpdateUserSchema{
		Workspace:      this.ID,
		UserSchema:     schema,
		PrimarySources: primarySources,
		Identifiers:    identifiers,
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

	err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {

		// TODO(Gianluca): the altering of the columns of the users table is
		// done within the transaction to avoid updating the state when in
		// reality the alter schema ends with an error, thus preventing the
		// creation of an inconsistent state that would require resetting the
		// databases.
		//
		// This is just a temporary workaround.
		//
		// The topic is discussed in the issue
		// https://github.com/meergo/meergo/issues/692.
		err = this.store.AlterUserSchema(ctx, schema, operations)
		if err != nil {
			if err == datastore.ErrAlterInProgress {
				return nil, errors.Unprocessable(AlterSchemaInProgress, "an alter schema operation is already in progress on the warehouse")
			}
			if err == datastore.ErrIdentityResolutionInProgress {
				return nil, errors.Unprocessable(IdentityResolutionInProgress, "an Identity Resolution is currently in progress on the warehouse")
			}
			if err == datastore.ErrInspectionMode {
				return nil, errors.Unprocessable(InspectionMode, "data warehouse is in inspection mode")
			}
			if err, ok := err.(*datastore.UnavailableError); ok {
				return nil, errors.Unavailable("%s", err)
			}
			return nil, err
		}

		// Update the schema.
		_, err := tx.Exec(ctx, "UPDATE workspaces SET user_schema = $1, identifiers = $2 WHERE id = $3", schemaJSON, n.Identifiers, n.Workspace)
		if err != nil {
			return nil, err
		}
		// Update the primary sources.
		_, err = tx.Exec(ctx, "DELETE FROM user_schema_primary_sources s USING connections c\n"+
			"WHERE c.workspace = $1 AND s.source = c.id", n.Workspace)
		if err != nil {
			return nil, err
		}
		if insertPrimarySources != "" {
			_, err = tx.Exec(ctx, insertPrimarySources, paths...)
			if err != nil {
				if db.IsForeignKeyViolation(err) && db.ErrConstraintName(err) == "user_schema_primary_sources_source_fkey" {
					err = errors.Unprocessable(ConnectionNotExist, "a primary source does not exist")
				}
				return nil, err
			}
		}

		return n, nil
	})
	if err != nil {
		return err
	}

	return nil
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
		case types.TextKind:
			if p.Type.Values() != nil {
				return fmt.Errorf("user schema properties with type text cannot specify values")
			}
			if p.Type.Regexp() != nil {
				return fmt.Errorf("user schema properties with type text cannot specify regexp")
			}
		case types.ArrayKind:
			k := p.Type.Elem().Kind()
			if k == types.ArrayKind || k == types.ObjectKind || k == types.MapKind {
				return fmt.Errorf("user schema properties cannot have type %s(%s)", p.Type.Kind(), k)
			}
			if p.Type.Unique() {
				return fmt.Errorf("user schema properties with type array cannot specify unique elements")
			}
			if p.Type.MinElements() != 0 {
				return fmt.Errorf("user schema properties with type array cannot specify minimum elements count")
			}
			if p.Type.MaxElements() != types.MaxElements {
				return fmt.Errorf("user schema properties with type array cannot specify maximum elements count")
			}
		case types.ObjectKind:
			err := checkAllowedPropertyUserSchema(p.Type)
			if err != nil {
				return err
			}
		case types.MapKind:
			k := p.Type.Elem().Kind()
			if k == types.ArrayKind || k == types.ObjectKind || k == types.MapKind {
				return fmt.Errorf("user schema properties cannot have type %s(%s)", p.Type.Kind(), k)
			}
		}
	}
	return nil
}

// validatePrimarySources validates a primary source returning an error if it is
// not valid.
func validatePrimarySources(schema types.Type, primarySources map[string]int) error {
	for path, source := range primarySources {
		p, err := types.PropertyByPath(schema, path)
		if err != nil {
			return err
		}
		if source < 1 || source > maxInt32 {
			return fmt.Errorf("primary source identifier %d is not valid", source)
		}
		if k := p.Type.Kind(); k == types.ObjectKind || k == types.ArrayKind {
			return fmt.Errorf("primary sources cannot be specified for %s properties", p.Type)
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
