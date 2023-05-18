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

	"chichi/apis/errors"
	"chichi/apis/mappings"
	"chichi/apis/state"
	"chichi/apis/warehouses"
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

	// Load the users schema.
	usersSchema, ok := this.action.Connection().Workspace().Schemas["users"]
	if !ok {
		return actionExecutionError{errors.New("users schema not loaded")}
	}
	inSchema := usersSchemaToConnectionSchema(*usersSchema, state.DatabaseType)

	// Instantiate a new mapping.
	mapping, err := mappings.New(inSchema, this.action.Schema, this.action.Mapping, this.action.Transformation)
	if err != nil {
		return err
	}

	// Open a connection to the app.
	connection := this.action.Connection()
	connector := connection.Connector()
	var clientSecret, resourceCode, accessToken string
	if r, ok := connection.Resource(); ok {
		clientSecret = connector.OAuth.ClientSecret
		resourceCode = r.Code
		var err error
		accessToken, err = freshAccessToken(this.db, r)
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot retrive the OAuth access token: %s", err)}
		}
	}
	fh, err := this.newFirehose(ctx)
	if err != nil {
		return actionExecutionError{err}
	}
	ws := this.action.Connection().Workspace()
	c, err := _connector.RegisteredApp(connector.Name).Open(fh.ctx, &_connector.AppConfig{
		Role:          _connector.DestinationRole,
		Settings:      connection.Settings,
		Firehose:      fh,
		ClientSecret:  clientSecret,
		Resource:      resourceCode,
		AccessToken:   accessToken,
		PrivacyRegion: _connector.PrivacyRegion(ws.PrivacyRegion),
	})
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}

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

		// Apply the mapping (or the transformation).
		props, err := mapping.Apply(ctx, user.Properties)
		if err != nil {
			return err
		}
		userToSet := _connector.User{
			ID:         id,
			Properties: props,
		}

		// Set the user to the app.
		err = c.(_connector.AppUsersConnection).SetUser(userToSet)
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot set user: %s", err)}
		}

		// Handle errors occurred in the firehose.
		if fh.err != nil {
			return fh.err
		}

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
	connector := connection.Connector()

	// Retrieve the storage associated to the file connection.
	var storage _connector.StorageConnection
	{
		s, _ := connection.Storage()
		fh := this.newFirehoseForConnection(ctx, s)
		ctx = fh.ctx
		var err error
		storage, err = _connector.RegisteredStorage(s.Connector().Name).Open(ctx, &_connector.StorageConfig{
			Role:     role,
			Settings: s.Settings,
			Firehose: fh,
		})
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot connect to the storage connector: %s", err)}
		}
	}

	fh, err := this.newFirehose(ctx)
	if err != nil {
		return err
	}
	c, err := _connector.RegisteredFile(connector.Name).Open(fh.ctx, &_connector.FileConfig{
		Role:     role,
		Settings: connection.Settings,
		Firehose: fh,
	})
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
	var records _connector.RecordReader = fh.newRecordReader(columns, usersSlices)

	// Write the file on the storage.
	err = writeFile(storage, c, this.action.Path, this.action.Sheet, c.ContentType(), records)
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot write file: %s", err)}
	}

	return nil
}

// downloadUsersForIdentityMatch downloads the users of the external app for
// resolving the external identity.
func (this *Action) downloadUsersForIdentityMatch() error {

	const role = _connector.SourceRole

	connection := this.action.Connection()
	connector := connection.Connector()

	var clientSecret, resourceCode, accessToken string
	if r, ok := connection.Resource(); ok {
		clientSecret = connector.OAuth.ClientSecret
		resourceCode = r.Code
		var err error
		accessToken, err = freshAccessToken(this.db, r)
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot retrive the OAuth access token: %s", err)}
		}
	}

	fh, err := this.newFirehose(context.Background())
	if err != nil {
		return actionExecutionError{err}
	}
	ws := this.action.Connection().Workspace()
	c, err := _connector.RegisteredApp(connector.Name).Open(fh.ctx, &_connector.AppConfig{
		Role:          role,
		Settings:      connection.Settings,
		Firehose:      fh,
		ClientSecret:  clientSecret,
		Resource:      resourceCode,
		AccessToken:   accessToken,
		PrivacyRegion: _connector.PrivacyRegion(ws.PrivacyRegion),
	})
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}

	// Read the users from the app.
	properties := []_connector.PropertyPath{
		{this.action.MatchingProperties.External},
	}
	err = c.(_connector.AppUsersConnection).Users(this.action.UserCursor, properties)
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot get users from the connector: %s", err)}
	}

	// Handle errors occurred in the firehose.
	if fh.err != nil {
		return fh.err
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
	ctx := context.Background()
	wh := this.action.Connection().Workspace().Warehouse
	externalID, ok, err := wh.DestinationUser(ctx, this.action.Connection().ID, string(p))
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
	columns := columnsOfProperties(schema.Properties())

	var where warehouses.Expr
	if len(ids) > 0 {
		operands := make([]warehouses.Expr, len(ids))
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

	users, err := ws.Warehouse.Select(context.Background(), "users", columns, where, idProperty, 0, 1000)
	if err != nil {
		if err2, ok := err.(*warehouses.Error); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			log.Printf("[error] cannot get users from the data warehouse of the workspace %d: %s", ws.ID, err)
			err = errors.Unprocessable(WarehouseFailed, "warehouse connection is failed: %w", err2.Err)
		}
		return nil, err
	}

	exportUsers := make([]userToExport, len(users))
	for i, user := range users {
		props, _ := deserializeDataWarehouseRowAsMap(schema.Properties(), user)
		gid, ok := props["id"].(int)
		if !ok {
			return nil, errors.New("missing or invalid GID")
		}
		exportUsers[i] = userToExport{
			GID:        gid,
			Properties: props,
		}
	}

	return exportUsers, nil
}

func writeFile(storage _connector.StorageConnection, file _connector.FileConnection, path, sheet, contentType string, records _connector.RecordReader) error {
	r, w := io.Pipe()
	var err2 error
	ch := make(chan struct{})
	var interruptErr = errors.New("interrupt")
	go func() {
		err2 = storage.Write(r, path, contentType)
		if err2 == interruptErr {
			err2 = nil
		}
		if err2 != nil {
			_ = r.CloseWithError(interruptErr)
		} else {
			_ = r.Close()
		}
		ch <- struct{}{}
	}()
	err := file.Write(w, sheet, records)
	if err == interruptErr {
		err = nil
	}
	if err != nil {
		_ = w.CloseWithError(interruptErr)
	} else {
		_ = w.Close()
	}
	<-ch
	if err != nil {
		return err
	}
	return err2
}
