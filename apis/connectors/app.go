//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package connectors

import (
	"context"
	"fmt"
	"io"
	"strings"

	"chichi/apis/connectors/httpclient"
	"chichi/apis/state"
	_connector "chichi/connector"
	"chichi/connector/types"
)

type WebhookPayload = _connector.WebhookPayload

// App represents the app of an app connection.
type App struct {
	name  string
	http  *httpclient.HTTP
	inner _connector.AppConnection
	err   error
}

// App returns an app for the provided connection. Errors are deferred until an
// app's method is called. It panics if connection is not an app connection.
func (connectors *Connectors) App(connection *state.Connection) *App {
	app := &App{
		name: connection.Connector().Name,
		http: connectors.http,
	}
	var resourceID int
	var resourceCode string
	if r, ok := connection.Resource(); ok {
		resourceID = r.ID
		resourceCode = r.Code
	}
	app.inner, app.err = _connector.RegisteredApp(app.name).New(&_connector.AppConfig{
		Role:        _connector.Role(connection.Role),
		Settings:    connection.Settings,
		SetSettings: setSettingsFunc(connectors.db, connection),
		Resource:    resourceCode,
		HTTPClient:  connectors.http.ConnectionClient(connection.ID),
		Region:      _connector.PrivacyRegion(connection.Workspace().PrivacyRegion),
		WebhookURL:  webhookURL(connection, resourceID),
	})
	return app
}

// CreateUser creates a user with the provided properties.
// It panics if the app does not support the users target.
func (app *App) CreateUser(ctx context.Context, user map[string]any) error {
	if app.err != nil {
		return app.err
	}
	return app.inner.(_connector.AppUsersConnection).CreateUser(ctx, user)
}

// EventTypes returns the app's event types.
// It panics if the app does not support the events target.
func (app *App) EventTypes(ctx context.Context) ([]*EventType, error) {
	if app.err != nil {
		return nil, app.err
	}
	return app.inner.(_connector.AppEventsConnection).EventTypes(ctx)
}

// PreviewSendEvent returns a preview of the event that would be sent when
// calling SendEvent with the same arguments. If the event type does not exist,
// it returns the ErrEventTypeNotExist error.
// It panics if the app does not support the events target.
func (app *App) PreviewSendEvent(ctx context.Context, eventType string, event *Event, data map[string]any) ([]byte, error) {
	if app.err != nil {
		return nil, app.err
	}
	preview, err := app.inner.(_connector.AppEventsConnection).PreviewSendEvent(ctx, eventType, event, data)
	if err != nil {
		if err == _connector.ErrEventTypeNotExist {
			err = ErrEventTypeNotExist
		}
		return nil, err
	}
	return preview, nil
}

// Schema returns the app's schema for the provided role and target. If target
// is state.Events, eventType represents the type of the event. If the event
// type does not exist, it returns the ErrEventTypeNotExist error.
//
// For the users and the groups target, the returned schema contains only the
// properties compatible with the provided role. For the events target, the
// returned schema can be the invalid schema.
//
// It panics if the app does not support the provided target.
func (app *App) Schema(ctx context.Context, role state.Role, target state.Target, eventType string) (types.Type, error) {
	if app.err != nil {
		return types.Type{}, app.err
	}
	var schema types.Type
	var err error
	switch target {
	case state.Events:
		inner := app.inner.(_connector.AppEventsConnection)
		eventTypes, err := inner.EventTypes(ctx)
		if err != nil {
			return types.Type{}, err
		}
		var found bool
		for _, t := range eventTypes {
			if t.ID == eventType {
				schema = t.Schema
				found = true
				break
			}
		}
		if !found {
			return types.Type{}, ErrEventTypeNotExist
		}
	case state.Users:
		inner := app.inner.(_connector.AppUsersConnection)
		schema, err = inner.UserSchema(ctx)
		if err != nil {
			return types.Type{}, err
		}
		if !schema.Valid() {
			return types.Type{}, fmt.Errorf("connector %s returned an invalid user schema", app.name)
		}
		schema = schema.AsRole(types.Role(role))
		if !schema.Valid() {
			return types.Type{}, fmt.Errorf("connection has returned a schema without %s properties", strings.ToLower(role.String()))
		}
	case state.Groups:
		inner := app.inner.(_connector.AppGroupsConnection)
		schema, err = inner.GroupSchema(ctx)
		if err != nil {
			return types.Type{}, err
		}
		if !schema.Valid() {
			return types.Type{}, fmt.Errorf("connector %s returned an invalid user schema", app.name)
		}
		schema = schema.AsRole(types.Role(role))
		if !schema.Valid() {
			return types.Type{}, fmt.Errorf("connector has returned a schema without %s properties", strings.ToLower(role.String()))
		}
	}
	return schema, nil
}

// SendEvent sends an event, with the give event type, along with the provided
// mapped data.
//
// It returns the ErrEventTypeNotExist error if the event type does not exist.
// It panics if the app does not support the events target.
func (app *App) SendEvent(ctx context.Context, eventType string, event *Event, data map[string]any) error {
	if app.err != nil {
		return app.err
	}
	err := app.inner.(_connector.AppEventsConnection).SendEvent(ctx, eventType, event, data)
	if err == _connector.ErrEventTypeNotExist {
		err = ErrEventTypeNotExist
	}
	return err
}

// UpdateUser updates the user with identifier id setting the provided
// properties.
//
// It panics if the app does not support the users target.
func (app *App) UpdateUser(ctx context.Context, id string, user map[string]any) error {
	if app.err != nil {
		return app.err
	}
	return app.inner.(_connector.AppUsersConnection).UpdateUser(ctx, id, user)
}

// Users returns the app's users starting from the provided cursor. The returned
// users comply with the specified schema. Values are returned only for the
// properties in the schema. If there is an error reading a user or validating
// a user against the schema, the user is still returned, but User.Err is set.
//
// It returns the io.EOF error when there are no more users other than the
// returned one. It panics if the app does not support the users target.
func (app *App) Users(ctx context.Context, schema types.Type, cursor Cursor) (users []User, next string, err error) {

	if app.err != nil {
		return nil, "", app.err
	}

	properties := schema.Properties()
	property := make(map[string]*types.Property, len(properties))
	names := make([]string, len(properties))
	for i, p := range properties {
		property[p.Name] = &p
		names[i] = p.Name
	}

	// Retrieve the users.
	users, next, err = app.inner.(_connector.AppUsersConnection).Users(ctx, names, cursor)
	eof := err == io.EOF
	if err != nil && !eof {
		return nil, "", err
	}
	if len(users) == 0 {
		if !eof {
			return nil, "", fmt.Errorf("%s returned zero users but did not return io.EOF", app.name)
		}
		if next != "" {
			return nil, "", fmt.Errorf("%s returned zero users but returned a non-empty next value", app.name)
		}
		if users == nil {
			users = []User{}
		}
		return users, "", io.EOF
	}

	// Normalize the properties.
	for _, user := range users {
		for name, value := range user.Properties {
			p, ok := property[name]
			if !ok {
				delete(user.Properties, name)
				continue
			}
			value, err = normalizeAppProperty(name, p.Type, value, p.Nullable)
			if err != nil {
				user.Err = err
				break
			}
			user.Properties[name] = value
		}
		user.Timestamp = user.Timestamp.UTC()
	}

	if eof {
		return users, "", io.EOF
	}
	return users, next, nil
}
