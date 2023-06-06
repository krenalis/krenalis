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

	"chichi/apis/normalization"
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
	fh, err := this.newFirehose(ctx)
	if err != nil {
		return err
	}
	ws := this.action.Connection().Workspace()
	connector, err := _connector.RegisteredApp(c.Connector().Name).Open(fh.ctx, &_connector.AppConfig{
		Role:          _connector.SourceRole,
		Settings:      c.Settings,
		Firehose:      fh,
		Resource:      resourceCode,
		HTTPClient:    this.http.ConnectionClient(c.ID),
		PrivacyRegion: _connector.PrivacyRegion(ws.PrivacyRegion),
		WebhookURL:    webhookURL(c, resourceID),
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
					return actionExecutionError{fmt.Errorf("connector %d has returned an unknown property %q", fh.connection.ID, name)}
				}
				value, err := normalization.NormalizeAppProperty(name, p.Nullable, p.Type, value)
				if err != nil {
					return actionExecutionError{err}
				}
				user.Properties[name] = value
			}

			// Map properties.
			mappedUser, err := fh.mapping.Apply(ctx, user.Properties)
			if err != nil {
				return actionExecutionError{err}
			}
			connection := &Connection{
				db:         fh.db,
				connection: fh.connection,
				http:       fh.action.http,
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
