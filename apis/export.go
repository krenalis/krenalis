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
	"io"
	"log"

	"chichi/apis/datastore"
	"chichi/apis/datastore/warehouses"
	"chichi/apis/errors"
	"chichi/apis/mappings"
	"chichi/apis/state"
	_connector "chichi/connector"
	"chichi/connector/types"
)

type userToExport struct {
	GID        int
	Properties map[string]any
}

// exportUsersToApp exports the users to the app.
func (this *Action) exportUsersToApp(ctx context.Context) error {

	// TODO(Gianluca): we should export only the users modified since last
	// export.

	users, err := this.readUsersFromDataWarehouse(nil)
	if err != nil {
		return err
	}

	// Filter the users.
	if this.action.Filter != nil {
		filteredUsers := []userToExport{}
		for _, user := range users {
			ok, err := mappings.ActionFilterApplies(this.action.Filter, user.Properties)
			if err != nil {
				return err
			}
			if ok {
				filteredUsers = append(filteredUsers, user)
			}
		}
		users = filteredUsers
	}
	if len(users) == 0 {
		return nil
	}

	// Download the users from this connection to match the identities.
	err = this.downloadUsersForIdentityMatch()
	if err != nil {
		return err
	}

	// TODO(Gianluca): here we assume that the user read from the data warehouse
	// is correctly normalized. We should investigate and discuss about this
	// behavior, and eventually add an additional normalization step.

	// Instantiate a new mapping.
	mapping, err := mappings.New(this.action.InSchema, this.action.OutSchema, this.action.Mapping, this.action.Transformation, true)
	if err != nil {
		return err
	}

	// Open a connection to the app.
	app, err := this.connection.openAppUsers(ctx)
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}

	connector := this.action.Connection().Connector()
	inSchemaProps := this.action.InSchema.PropertiesNames()

	for _, user := range users {

		// Resolve the external identity.
		id, exists, err := this.resolveExternalIdentity(user)
		if err != nil {
			return err
		}

		// Determine if this user must be exported or not.
		mode := *this.action.ExportMode
		if (mode == state.CreateOnly && exists) ||
			(mode == state.UpdateOnly && !exists) {
			continue
		}

		// Take only the necessary properties.
		props := make(map[string]any, len(inSchemaProps))
		for _, name := range inSchemaProps {
			props[name] = user.Properties[name]
		}

		// Normalize the user properties (read from the data warehouse) using
		// the action's mapping input schema.
		props, err = normalize(props, this.action.InSchema)
		if err != nil {
			return actionExecutionError{err}
		}

		// Map the properties of the user.
		props, err = mapping.Apply(ctx, props)
		if err != nil {
			return actionExecutionError{err}
		}

		// Update the user, if it already exists on the app.
		if exists {
			err := app.UpdateUser(id, props)
			if err != nil {
				return actionExecutionError{fmt.Errorf("cannot update user: %s", err)}
			}
			log.Printf("[info] user %q updated on %s: %#v", id, connector.Name, props)
			continue
		}

		// Create the user.
		err = app.CreateUser(props)
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot create user: %s", err)}
		}
		log.Printf("[info] a new user has been created on %s: %#v", connector.Name, user)

	}

	return nil
}

// exportUsersToFile exports the users to the file.
func (this *Action) exportUsersToFile(ctx context.Context) error {

	users, err := this.readUsersFromDataWarehouse(nil)
	if err != nil {
		return err
	}

	// Filter the users.
	if this.action.Filter != nil {
		filteredUsers := []userToExport{}
		for _, user := range users {
			ok, err := mappings.ActionFilterApplies(this.action.Filter, user.Properties)
			if err != nil {
				return err
			}
			if ok {
				filteredUsers = append(filteredUsers, user)
			}
		}
		users = filteredUsers
	}

	const role = _connector.DestinationRole

	connection := this.action.Connection()

	// Retrieve the storage associated to the file connection.
	var storage *compressorStorage
	{
		st, err := this.connection.openStorage(ctx)
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot connect to the storage connector: %s", err)}
		}
		storage = newCompressedStorage(st, connection.Compression)
	}

	file, err := this.connection.openFile(ctx)
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}

	// Determine the columns.
	var columns []types.Property
	if len(users) > 0 {
		userSchema, ok := connection.Workspace().Schemas["users"]
		if !ok {
			return actionExecutionError{errors.New("'users' schema not found")}
		}
		for _, p := range userSchema.Properties() {
			if _, ok := users[0].Properties[p.Name]; ok {
				columns = append(columns, p)
			}
		}
	}

	// Prepare the users and the record reader.
	usersSlices := make([][]any, len(users))
	for i, u := range users {
		userSlice := make([]any, len(columns))
		for j, c := range columns {
			userSlice[j] = u.Properties[c.Name]
		}
		usersSlices[i] = userSlice
	}
	records := newRecordReader(columns, usersSlices)

	// Write the file to the storage.
	w, err := storage.Writer(this.action.Path, file.ContentType())
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot write file: %s", err)}
	}
	err = file.Write(w, this.action.Sheet, records)
	if err2 := w.CloseWithError(err); err2 != nil && err == nil {
		err = err2
	}
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot write file: %s", err)}
	}

	return nil
}

// downloadUsersForIdentityMatch downloads the users of the external app for
// resolving the external identity.
func (this *Action) downloadUsersForIdentityMatch() error {

	ctx := context.Background()
	app, err := this.connection.openAppUsers(ctx)
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}

	// Read the users from the app.
	properties := []types.Path{
		{this.action.MatchingProperties.External},
	}

	// TODO(Gianluca): here cursor.Next is set to "" as a workaround. See the
	// issue https://github.com/open2b/chichi/issues/183.
	var cursor _connector.Cursor

	c := this.connection

	var eof bool

	// Importing users from a destination to match identities for the export.
	for !eof {

		users, next, err := app.Users(properties, cursor)
		if err != nil && err != io.EOF {
			return actionExecutionError{fmt.Errorf("cannot get users from the connector: %s", err)}
		}
		if err == io.EOF {
			eof = true
		} else if len(users) == 0 {
			return actionExecutionError{fmt.Errorf("connector %d has returned an empty users without returning EOF", c.ID)}
		}

		for _, user := range users {

			externalPropName := this.action.MatchingProperties.External
			externalProp, ok := user.Properties[externalPropName]
			if !ok {
				// TODO(Gianluca): handle this error properly.
				return actionExecutionError{fmt.Errorf("user does not contain property %q", externalPropName)}
			}
			p, err := json.Marshal(externalProp)
			if err != nil {
				return actionExecutionError{err}
			}
			err = c.store.SetDestinationUser(ctx, this.action.ID, user.ID, string(p))
			if err != nil {
				return actionExecutionError{err}
			}

		}

		// Set the user cursor.
		if len(users) > 0 {
			last := users[len(users)-1]
			cursor.ID = last.ID
			cursor.Timestamp = last.Timestamp
		}
		cursor.Next = next
		err = this.setUserCursor(ctx, cursor)
		if err != nil {
			return actionExecutionError{err}
		}

	}

	return nil
}

// resolveExternalIdentity resolves the external identity of user and returns
// its external ID and true, if resolved, or the empty string and false if such
// user does not exist on the remote app.
func (this *Action) resolveExternalIdentity(user userToExport) (string, bool, error) {
	internalPropName := this.action.MatchingProperties.Internal
	property, ok := user.Properties[internalPropName]
	if !ok {
		return "", false, fmt.Errorf("property %q not found", internalPropName)
	}
	p, err := json.Marshal(property)
	if err != nil {
		return "", false, err
	}
	c := this.connection
	ctx := context.Background()
	externalID, ok, err := c.store.DestinationUser(ctx, this.action.ID, string(p))
	if err != nil {
		return "", false, err
	}
	return externalID, ok, nil
}

// readUsersFromDataWarehouse reads the users with the given IDs from the data
// warehouse.
//
// TODO(Gianluca): this method returns at most 1000 users. This is wrong. We
// should find an alternative way to implement this; maybe we could read one
// user at a time.
func (this *Action) readUsersFromDataWarehouse(ids []int) ([]userToExport, error) {

	ws := this.action.Connection().Workspace()

	// Read the schema.
	schema, ok := ws.Schemas["users"]
	if !ok {
		return nil, errors.New("users schema not found")
	}

	// Read the users.

	var where datastore.Expr
	if len(ids) > 0 {
		operands := make([]datastore.Expr, len(ids))
		for i := range ids {
			operands[i] = warehouses.NewBaseExpr(
				warehouses.ExprColumn{Name: "id", Type: types.PtInt},
				warehouses.OperatorEqual,
				ids[i],
			)
		}
		where = warehouses.NewMultiExpr(warehouses.LogicalOperatorOr, operands)
	}
	idProperty, ok := schema.Property("id")
	if !ok {
		return nil, errors.New("property 'id' not found in schema")
	}

	store := this.connection.store
	users, err := store.Users(context.Background(), schema.Properties(), where, idProperty, 0, 1000)
	if err != nil {
		if err2, ok := err.(*datastore.Error); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			log.Printf("[error] cannot get users from the data warehouse of the workspace %d: %s", ws.ID, err)
			err = errors.Unprocessable(WarehouseFailed, "warehouse connection is failed: %w", err2.Err)
		}
		return nil, err
	}

	exportUsers := make([]userToExport, len(users))
	for i, user := range users {
		gid, ok := user["id"].(int)
		if !ok {
			return nil, errors.New("missing or invalid GID")
		}
		exportUsers[i] = userToExport{
			GID:        gid,
			Properties: user,
		}
	}

	return exportUsers, nil
}
