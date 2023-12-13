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
	"log/slog"

	"chichi/apis/connectors"
	"chichi/apis/state"
	"chichi/apis/transformers"
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
	var cursor state.Cursor

	records, err := this.app().Users(ctx, schema, cursor)
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot get users from the connector: %s", err)}
	}
	defer records.Close()

	// Importing users from a destination to match identities for the export.
	err = records.For(func(user connectors.Record) error {

		if user.Err != nil {
			return actionExecutionError{user.Err}
		}

		p, err := json.Marshal(user.Properties[externalProp.Name])
		if err != nil {
			return actionExecutionError{err}
		}
		err = this.connection.store.SetDestinationUser(ctx, this.action.ID, user.ID, string(p))
		if err != nil {
			return actionExecutionError{err}
		}

		// Set the user cursor.
		err = this.setUserCursor(ctx, state.Cursor{ID: user.ID, Timestamp: user.Timestamp})
		if err != nil {
			return actionExecutionError{err}
		}

		return nil
	})
	if err != nil {
		return err
	}
	if err = records.Err(); err != nil {
		return actionExecutionError{fmt.Errorf("an error occurred closing the database: %s", err)}
	}

	return nil
}

// exportUsersToApp exports the users to the app.
func (this *Action) exportUsersToApp(ctx context.Context) error {

	usersSchema, err := this.connection.schema(ctx, "users")
	if err != nil {
		return actionExecutionError{err}
	}
	if !usersSchema.Valid() {
		return actionExecutionError{fmt.Errorf("users schema not found")}
	}

	// Ensure that the type of the internal matching property is equal to the
	// type of the corresponding property in the users schema.
	{
		internalName := this.action.MatchingProperties.Internal
		prop, ok := usersSchema.Property(internalName)
		if !ok {
			return actionExecutionError{fmt.Errorf("property '%s' not found in users schema", internalName)}
		}
		internal, ok := this.action.InSchema.Property(internalName)
		if !ok {
			return actionExecutionError{fmt.Errorf("property '%s' not found in action input schema", internalName)}
		}
		if !internal.Type.EqualTo(prop.Type) {
			return actionExecutionError{fmt.Errorf("type of internal matching "+
				"property '%s' does not match with the type of the corresponding "+
				"property in users", internalName)}
		}
	}

	users, err := this.readUsersFromDataWarehouse(ctx, nil, this.action.InSchema)
	if err != nil {
		return err
	}

	// Filter the users.
	if this.action.Filter != nil {
		filteredUsers := []userToExport{}
		for _, user := range users {
			ok, err := filterApplies(this.action.Filter, user.Properties)
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

	// Instantiate a new transformer.
	connection := this.action.Connection()
	connector := connection.Connector()
	transformer, err := transformers.New(this.action.InSchema, this.action.OutSchema, this.action.Transformation,
		this.action.ID, this.apis.functionTransformer, &connector.Layouts)
	if err != nil {
		return err
	}

	app := this.app()
	connectorName := connection.Connector().Name
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

		// Transform the user.
		props, err = transformer.Transform(ctx, props)
		if err != nil {
			if err, ok := err.(transformers.FunctionExecutionError); ok {
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
