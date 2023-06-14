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

	mapping, err := mappings.New(this.action.InSchema, this.action.OutSchema, this.action.Mapping, this.action.PythonSource, false)
	if err != nil {
		return actionExecutionError{err}
	}

	// Determine the properties to import.
	var propertiesPaths []types.Path
	if this.action.PythonSource != "" { // Transformation function.
		for _, name := range this.action.InSchema.PropertiesNames() {
			propertiesPaths = append(propertiesPaths, types.Path{name})
		}
	} else { // Mappings.
		for _, in := range this.action.Mapping {
			propertiesPaths = append(propertiesPaths, strings.Split(in, "."))
		}
	}

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
