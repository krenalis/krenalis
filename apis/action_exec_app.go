//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"fmt"
	"io"
	"strings"

	"chichi/apis/mappings"
	_connector "chichi/connector"
	"chichi/connector/types"
)

// importFromApp imports the users from an app.
func (this *Action) importFromApp(ctx context.Context) error {

	app, err := this.connection.openAppUsers(ctx)
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}
	cursor := this.action.UserCursor
	if exe, _ := this.action.Execution(); exe.Reimport {
		cursor = _connector.Cursor{}
	}

	mapping, err := mappings.New(this.action.InSchema, this.action.OutSchema, this.action.Mapping, this.action.Transformation, false)
	if err != nil {
		return actionExecutionError{err}
	}

	// Determine the properties to import.
	var properties []types.Path
	for _, path := range this.action.Mapping {
		properties = append(properties, strings.Split(path, "."))
	}
	if this.action.Transformation != nil {
		for _, name := range this.action.Transformation.In {
			if _, ok := this.action.Mapping[name]; !ok {
				properties = append(properties, types.Path{name})
			}
		}
	}

	var eof bool

	for !eof {

		users, next, err := app.Users(properties, cursor)
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
			mappedUser, err := mapping.Apply(ctx, userProps)
			if err != nil {
				return actionExecutionError{err}
			}

			// Write the user.
			err = this.connection.writeConnectionUsers(ctx, user.ID, user.Properties, user.Timestamp.UTC(), nil)
			if err != nil {
				return actionExecutionError{err}
			}
			err = this.setUser(ctx, user.ID, mappedUser)
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
