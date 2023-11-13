//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"chichi/apis/mappings"
	"chichi/apis/state"
	"chichi/connector"
	"chichi/connector/types"
)

// downloadUsersForExportMatch downloads the users of the external app for the
// matching of the export.
func (this *Action) downloadUsersForExportMatch(ctx context.Context) error {

	// Create a schema with only the matching property.
	externalProp := this.action.MatchingProperties.External
	schema := types.Object([]types.Property{externalProp})

	// TODO(Gianluca): here cursor.Next is set to "" as a workaround. See the
	// issue https://github.com/open2b/chichi/issues/183.
	var cursor connector.Cursor

	c := this.connection

	var eof bool
	app := this.app()

	// Importing users from a destination to match identities for the export.
	for !eof {

		users, next, err := app.Users(ctx, schema, cursor)
		if err != nil && err != io.EOF {
			return actionExecutionError{fmt.Errorf("cannot get users from the connector: %s", err)}
		}
		if err == io.EOF {
			eof = true
		} else if len(users) == 0 {
			return actionExecutionError{fmt.Errorf("connector %d has returned an empty users without returning EOF", c.ID)}
		}

		for _, user := range users {

			if user.Err != nil {
				return actionExecutionError{err}
			}

			value, ok := user.Properties[externalProp.Name]
			if !ok {
				// TODO(Gianluca): handle this error properly.
				return actionExecutionError{fmt.Errorf("user does not contain property %q", externalProp.Name)}
			}
			p, err := json.Marshal(value)
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

// exportUsersToApp exports the users to the app.
func (this *Action) exportUsersToApp(ctx context.Context) error {

	connection := this.action.Connection()

	// Ensure that the type of the internal matching property is equal to the
	// type of the corresponding property in the users schema.
	{
		internal := this.action.MatchingProperties.Internal
		usersSchema, ok := connection.Workspace().Schemas["users"]
		if !ok {
			return actionExecutionError{fmt.Errorf("users schema not found")}
		}
		prop, ok := usersSchema.Property(internal.Name)
		if !ok {
			return actionExecutionError{fmt.Errorf("property '%s' not found in users schema", internal.Name)}
		}
		if !internal.Type.EqualTo(prop.Type) {
			return actionExecutionError{fmt.Errorf("type of internal matching "+
				"property '%s' does not match with the type of the corresponding "+
				"property in users", internal.Name)}
		}

	}

	users, err := this.readUsersFromDataWarehouse(ctx, nil)
	if err != nil {
		return err
	}

	// Filter the users.
	if this.action.Filter != nil {
		filteredUsers := []userToExport{}
		for _, user := range users {
			ok, err := mappings.FilterApplies(this.action.Filter, user.Properties)
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

	// Download the users from this connection to match the identities for the
	// export.
	err = this.downloadUsersForExportMatch(ctx)
	if err != nil {
		return err
	}

	// TODO(Gianluca): here we assume that the user read from the data warehouse
	// is correctly normalized. We should investigate and discuss about this
	// behavior, and eventually add an additional normalization step.

	// Instantiate a new mapping.
	connector := connection.Connector()
	mapping, err := mappings.New(this.action.InSchema, this.action.OutSchema, this.action.Mapping,
		this.action.Transformation, this.action.ID, this.apis.transformer, &connector.Layouts)
	if err != nil {
		return err
	}

	app := this.app()
	connectorName := this.action.Connection().Connector().Name
	properties := this.action.InSchema.PropertiesNames()

	for _, user := range users {

		// Resolve the external identity.
		id, exists, err := this.resolveExternalIdentity(ctx, user)
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
		props := make(map[string]any, len(properties))
		for _, name := range properties {
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
			if err, ok := err.(mappings.Error); ok {
				return actionExecutionError{err}
			}
			return err
		}

		// Update the user, if it already exists on the app.
		if exists {
			err := app.UpdateUser(ctx, id, props)
			if err != nil {
				return actionExecutionError{fmt.Errorf("cannot update user: %s", err)}
			}
			slog.Info("user updated", "id", id, "connector", connectorName, "properties", props)
			continue
		}

		// Create the user.
		err = app.CreateUser(ctx, props)
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot create user: %s", err)}
		}
		slog.Info("a new user has been created", "connector", connectorName, "user", user)

	}

	return nil
}

// importUsersFromApp imports the users from an app.
func (this *Action) importUsersFromApp(ctx context.Context) error {

	cursor := this.action.UserCursor
	if exe, _ := this.action.Execution(); exe.Reimport {
		cursor = connector.Cursor{}
	}

	mapping, err := mappings.New(this.action.InSchema, this.action.OutSchema, this.action.Mapping,
		this.action.Transformation, this.action.ID, this.apis.transformer, nil)
	if err != nil {
		return actionExecutionError{err}
	}

	var eof bool
	app := this.app()

	for !eof {

		users, next, err := app.Users(ctx, this.action.InSchema, cursor)
		if err != nil && err != io.EOF {
			return actionExecutionError{fmt.Errorf("cannot get users from the connector: %s", err)}
		}
		if err == io.EOF {
			eof = true
		} else if len(users) == 0 {
			connectorID := this.action.Connection().Connector().ID
			return actionExecutionError{fmt.Errorf("connector %d has returned an empty users without returning EOF", connectorID)}
		}

		for _, user := range users {

			if user.Err != nil {
				return actionExecutionError{err}
			}

			// Map the properties of the user.
			identity, err := mapping.Apply(ctx, user.Properties)
			if err != nil {
				if err, ok := err.(mappings.Error); ok {
					return actionExecutionError{err}
				}
				return err
			}

			// Set the identity into the data warehouse.
			err = this.connection.store.SetIdentity(ctx, identity, user.ID, "", this.action.ID, false, user.Timestamp)
			if err != nil {
				return actionExecutionError{err}
			}

			// Update the connection stats.
			err = this.connection.updateConnectionsStats(ctx)
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

	// Resolve and sync the users.
	err = this.connection.store.ResolveSyncUsers(ctx)
	if err != nil {
		return fmt.Errorf("cannot resolve and sync users: %s", err)
	}

	return nil
}

// resolveExternalIdentity resolves the external identity of user and returns
// its external ID and true, if resolved, or the empty string and false if such
// user does not exist on the remote app.
func (this *Action) resolveExternalIdentity(ctx context.Context, user userToExport) (string, bool, error) {
	internalPropName := this.action.MatchingProperties.Internal
	property, ok := user.Properties[internalPropName.Name]
	if !ok {
		return "", false, fmt.Errorf("property %q not found", internalPropName.Name)
	}
	p, err := json.Marshal(property)
	if err != nil {
		return "", false, err
	}
	c := this.connection
	externalID, ok, err := c.store.DestinationUser(ctx, this.action.ID, string(p))
	if err != nil {
		return "", false, err
	}
	return externalID, ok, nil
}
