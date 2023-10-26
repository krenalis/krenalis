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
)

// downloadUsersForIdentityMatch downloads the users of the external app for
// resolving the external identity.
func (this *Action) downloadUsersForIdentityMatch(ctx context.Context) error {

	app, err := this.connection.openAppUsers()
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}

	// Read the users from the app.
	properties := []string{this.action.MatchingProperties.External}

	// TODO(Gianluca): here cursor.Next is set to "" as a workaround. See the
	// issue https://github.com/open2b/chichi/issues/183.
	var cursor connector.Cursor

	c := this.connection

	var eof bool

	// Importing users from a destination to match identities for the export.
	for !eof {

		users, next, err := app.Users(ctx, properties, cursor)
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

// exportUsersToApp exports the users to the app.
func (this *Action) exportUsersToApp(ctx context.Context) error {

	// TODO(Gianluca): we should export only the users modified since last
	// export.

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

	// Download the users from this connection to match the identities.
	err = this.downloadUsersForIdentityMatch(ctx)
	if err != nil {
		return err
	}

	// TODO(Gianluca): here we assume that the user read from the data warehouse
	// is correctly normalized. We should investigate and discuss about this
	// behavior, and eventually add an additional normalization step.

	// Instantiate a new mapping.
	mapping, err := mappings.New(this.action.InSchema, this.action.OutSchema, this.action.Mapping,
		this.action.Transformation, this.action.ID, this.apis.transformer, true)
	if err != nil {
		return err
	}

	// Open a connection to the app.
	app, err := this.connection.openAppUsers()
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}

	connector := this.action.Connection().Connector()
	inSchemaProps := this.action.InSchema.PropertiesNames()

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
			slog.Info("user updated", "id", id, "connector", connector.Name, "properties", props)
			continue
		}

		// Create the user.
		err = app.CreateUser(ctx, props)
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot create user: %s", err)}
		}
		slog.Info("a new user has been created", "connector", connector.Name, "user", user)

	}

	return nil
}

// importUsersFromApp imports the users from an app.
func (this *Action) importUsersFromApp(ctx context.Context) error {

	app, err := this.connection.openAppUsers()
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}
	cursor := this.action.UserCursor
	if exe, _ := this.action.Execution(); exe.Reimport {
		cursor = connector.Cursor{}
	}

	mapping, err := mappings.New(this.action.InSchema, this.action.OutSchema, this.action.Mapping,
		this.action.Transformation, this.action.ID, this.apis.transformer, false)
	if err != nil {
		return actionExecutionError{err}
	}

	// Determine the properties to import.
	var properties []string
	for _, path := range this.action.Mapping {
		properties = append(properties, path)
	}
	// In case of transformation, also import every property declared in the
	// input schema of the action.
	if this.action.Transformation != nil {
		for _, name := range this.action.InSchema.PropertiesNames() {
			if _, ok := this.action.Mapping[name]; !ok {
				properties = append(properties, name)
			}
		}
	}

	var eof bool

	for !eof {

		users, next, err := app.Users(ctx, properties, cursor)
		if err != nil && err != io.EOF {
			return actionExecutionError{fmt.Errorf("cannot get users from the connector: %s", err)}
		}
		if err == io.EOF {
			eof = true
		} else if len(users) == 0 {
			connector := this.action.Connection().Connector()
			return actionExecutionError{fmt.Errorf("connector %d has returned an empty users without returning EOF", connector.ID)}
		}

		inSchemaProps := this.action.InSchema.PropertiesNames()

		for _, user := range users {

			// Take only the necessary properties.
			props := make(map[string]any, len(inSchemaProps))
			for _, name := range inSchemaProps {
				if v, ok := user.Properties[name]; ok {
					props[name] = v
				}
			}

			// Normalize the user properties (read from the app) using the
			// action's mapping input schema.
			userProps, err := normalize(props, this.action.InSchema)
			if err != nil {
				return actionExecutionError{err}
			}

			// Map the properties of the user.
			identity, err := mapping.Apply(ctx, userProps)
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
	property, ok := user.Properties[internalPropName]
	if !ok {
		return "", false, fmt.Errorf("property %q not found", internalPropName)
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
