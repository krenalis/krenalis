//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"errors"
	"fmt"
	"io"

	"chichi/apis/mappings"
	"chichi/apis/normalization"
	"chichi/apis/state"
	_connector "chichi/connector"
	"chichi/connector/types"
)

// importFromApp imports the users from an app.
func (this *Action) importFromApp() error {

	c := this.action.Connection()

	var resourceID int
	var resourceCode string
	if r, ok := c.Resource(); ok {
		resourceID = r.ID
		resourceCode = r.Code
	}

	ctx := context.Background()
	ws := c.Workspace()
	connector, err := _connector.RegisteredApp(c.Connector().Name).Open(ctx, &_connector.AppConfig{
		Role:        _connector.SourceRole,
		Settings:    c.Settings,
		SetSettings: this.setSettingsFunc(ctx),
		Resource:    resourceCode,
		HTTPClient:  this.http.ConnectionClient(c.ID),
		Region:      _connector.PrivacyRegion(ws.PrivacyRegion),
		WebhookURL:  webhookURL(c, resourceID),
	})
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}
	cursor := this.action.UserCursor
	if exe, _ := this.action.Execution(); exe.Reimport {
		cursor = _connector.Cursor{}
	}
	app := connector.(_connector.AppUsersConnection)

	// Read the properties to read.
	_, propertiesPaths, err := this.schema()
	if err != nil {
		return fmt.Errorf("cannot read user schema: %s", err)
	}

	usersSchema, ok := ws.Schemas["users"]
	if !ok {
		return actionExecutionError{errors.New("users schema not loaded")}
	}
	outputSchema := sourceMappingSchema(*usersSchema, state.AppType)
	mapping, err := mappings.New(this.action.Schema, outputSchema, this.action.Mapping, this.action.Transformation)
	if err != nil {
		return actionExecutionError{err}
	}

	properties := this.action.Schema.Properties()

	var eof bool

	for !eof {

		users, next, err := app.Users(propertiesPaths, cursor)
		if err != nil && err != io.EOF {
			return actionExecutionError{fmt.Errorf("cannot get users from the connector: %s", err)}
		}
		if err == io.EOF {
			eof = true
		} else if len(users) == 0 {
			return actionExecutionError{fmt.Errorf("connector %d has returned an empty users without returning EOF", c.Connector().ID)}
		}

		for _, user := range users {

			// Normalize properties.
			propertyOf := map[string]types.Property{}
			for _, p := range properties {
				propertyOf[p.Name] = p
			}
			for name, value := range user.Properties {
				p, ok := propertyOf[name]
				if !ok {
					return actionExecutionError{fmt.Errorf("connector %d has returned an unknown property %q", c.Connector().ID, name)}
				}
				value, err := normalization.NormalizeAppProperty(name, p.Nullable, p.Type, value)
				if err != nil {
					return actionExecutionError{err}
				}
				user.Properties[name] = value
			}

			// Map properties.
			mappedUser, err := mapping.Apply(ctx, user.Properties)
			if err != nil {
				return actionExecutionError{err}
			}
			connection := &Connection{
				db:         this.db,
				connection: c,
				http:       this.http,
			}
			err = connection.writeConnectionUsers(ctx, user.ID, user.Properties, user.Timestamp.UTC(), nil)
			if err != nil {
				return actionExecutionError{err}
			}
			err = connection.setUser(ctx, user.ID, mappedUser)
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
