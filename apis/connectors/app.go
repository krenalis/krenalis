//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package connectors

import (
	"context"
	"errors"
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
	name    string
	role    state.Role
	layouts *state.Layouts
	http    *httpclient.HTTP
	users   schema
	inner   _connector.AppConnection
	err     error
}

// App returns an app for the provided connection. Errors are deferred until an
// app's method is called. It panics if connection is not an app connection.
func (connectors *Connectors) App(connection *state.Connection) *App {
	connector := connection.Connector()
	app := &App{
		name:    connector.Name,
		role:    connection.Role,
		layouts: &connector.Layouts,
		http:    connectors.http,
		users:   schema{lock: make(chan struct{}, 1)},
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

// Schema returns the app's schema for the provided target. If target is
// state.Events, eventType represents the type of the event. If the event type
// does not exist, it returns the ErrEventTypeNotExist error.
//
// For the users and the groups target, the returned schema contains only the
// properties compatible with the provided role. For the events target, the
// returned schema can be the invalid schema.
//
// It panics if the app does not support the provided target.
func (app *App) Schema(ctx context.Context, target state.Target, eventType string) (types.Type, error) {
	return app.SchemaAsRole(ctx, app.role, target, eventType)
}

// SchemaAsRole is like Schema but returns the schema as the provided role,
// instead of the role of the app's connection.
func (app *App) SchemaAsRole(ctx context.Context, role state.Role, target state.Target, eventType string) (types.Type, error) {
	if app.err != nil {
		return types.Type{}, app.err
	}
	switch target {
	case state.Events:
		eventTypes, err := app.inner.(_connector.AppEventsConnection).EventTypes(ctx)
		if err != nil {
			return types.Type{}, err
		}
		var found bool
		var schema types.Type
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
		return schema, nil
	case state.Users:
		return app.usersSchema(ctx, types.Role(role))
	case state.Groups:
		schema, err := app.inner.(_connector.AppGroupsConnection).GroupSchema(ctx)
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
		return schema, nil
	}
	panic("unexpected target")
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

type SchemaError struct {
	Msg string
}

func (err SchemaError) Error() string {
	return err.Msg
}

// Users returns a Records to iterate over the app's users starting from the
// provided cursor. The returned users conform to the provided schema, which
// must be compatible with the source users schema.
//
// It returns a SchemaError error if the provided schema does not conform to the
// source users schema.
func (app *App) Users(ctx context.Context, schema types.Type, cursor state.Cursor) (Records, error) {
	if app.err != nil {
		return nil, app.err
	}
	// Check that the schema conforms to the source users schema.
	usersSchema, err := app.usersSchema(ctx, types.SourceRole)
	if err != nil {
		return nil, fmt.Errorf("cannot get users schema: %s", err)
	}
	err = checkConformity("", schema, usersSchema)
	if err != nil {
		return nil, err
	}
	records := &appRecords{
		ctx:     ctx,
		schema:  schema,
		layouts: app.layouts,
		cursor:  cursor,
		appName: app.name,
		inner:   app.inner,
	}
	return records, nil
}

// appRecords implements the Records interface for apps.
type appRecords struct {
	ctx     context.Context
	schema  types.Type
	layouts *state.Layouts
	cursor  state.Cursor
	appName string
	inner   _connector.AppConnection
	err     error
	closed  bool
}

func (r *appRecords) Close() error {
	r.closed = true
	return nil
}

func (r *appRecords) Err() error {
	return r.err
}

func (r *appRecords) For(yield func(Record) error) error {

	if r.closed {
		r.err = errors.New("connectors: For called on a closed Records")
		return nil
	}

	cursor := _connector.Cursor{
		ID:        r.cursor.ID,
		Timestamp: r.cursor.Timestamp,
	}

	properties := r.schema.Properties()
	names := make([]string, len(properties))
	propertyByName := make(map[string]*types.Property, len(properties))
	for i, p := range properties {
		names[i] = p.Name
		propertyByName[p.Name] = &p
	}

	for {

		// Retrieve the users.
		users, next, err := r.inner.(_connector.AppUsersConnection).Users(r.ctx, names, cursor)
		eof := err == io.EOF
		if err != nil && !eof {
			r.err = err
			return nil
		}
		if len(users) == 0 {
			if !eof {
				r.err = fmt.Errorf("%s returned zero users but did not return io.EOF", r.appName)
				return nil
			}
			if next != "" {
				r.err = fmt.Errorf("%s returned zero users but returned a non-empty next value", r.appName)
			}
			return nil
		}

		// Normalize the returned users.
		for _, user := range users {
			for _, p := range properties {
				value, ok := user.Properties[p.Name]
				if !ok {
					user.Err = fmt.Errorf(`app did not return a value for the property %q`, p.Name)
					break
				}
				value, err = normalizeAppProperty(p.Name, p.Type, value, p.Nullable, r.layouts)
				if err != nil {
					user.Err = err
					break
				}
				user.Properties[p.Name] = value
			}
			// Users method of the connector is permitted to return more properties than those requested,
			// so if necessary, remove those that are not requested.
			if len(user.Properties) != len(properties) {
				for name := range user.Properties {
					if _, ok := propertyByName[name]; !ok {
						delete(propertyByName, name)
					}
				}
			}
			user.Timestamp = user.Timestamp.UTC()
			if err := yield(user); err != nil {
				return err
			}
		}

		if eof {
			return nil
		}

		last := users[len(users)-1]
		cursor.ID = last.ID
		cursor.Timestamp = last.Timestamp
		cursor.Next = next

	}

}

type schema struct {
	lock    chan struct{}
	schemas [3]types.Type
}

// usersSchema returns the users schema with the provided role.
func (app *App) usersSchema(ctx context.Context, role types.Role) (types.Type, error) {
	select {
	case <-ctx.Done():
		return types.Type{}, errors.New("canceled context")
	case app.users.lock <- struct{}{}:
	}
	defer func() {
		<-app.users.lock
	}()
	if schema := app.users.schemas[role]; schema.Valid() {
		return schema, nil
	}
	schema, err := app.inner.(_connector.AppUsersConnection).UserSchema(ctx)
	if err != nil {
		return types.Type{}, err
	}
	var schemas [3]types.Type
	for r := types.BothRole; r <= types.DestinationRole; r++ {
		schemas[r] = schema.AsRole(r)
		if !schemas[r].Valid() {
			return types.Type{}, fmt.Errorf("connection has returned a schema without %s properties", strings.ToLower(role.String()))
		}
	}
	app.users.schemas = schemas
	return app.users.schemas[role], nil
}
