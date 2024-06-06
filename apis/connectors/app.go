//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package connectors

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/apis/connectors/httpclient"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/types"

	"github.com/google/uuid"
)

type (
	EventRequest   = chichi.EventRequest
	WebhookPayload = chichi.WebhookPayload
)

// ErrUnsupportedTarget indicates that a target is not supported.
var ErrUnsupportedTarget = errors.New("target is not supported")

// App represents the app of an app connection.
type App struct {
	name        string
	role        state.Role
	timeLayouts *state.TimeLayouts
	httpClient  *httpclient.Client
	users       schema
	targets     state.ConnectorTargets
	inner       chichi.App
	err         error
}

// App returns an app for the provided connection. Errors are deferred until an
// app's method is called. It panics if connection is not an app connection.
func (connectors *Connectors) App(connection *state.Connection) *App {
	connector := connection.Connector()
	app := &App{
		name:        connector.Name,
		role:        connection.Role,
		timeLayouts: &connector.TimeLayouts,
		httpClient:  connectors.http.ConnectionClient(connection.ID),
		users:       schema{lock: make(chan struct{}, 1)},
		targets:     connector.Targets,
	}
	var accountID int
	var accountCode string
	if a, ok := connection.Account(); ok {
		accountID = a.ID
		accountCode = a.Code
	}
	app.inner, app.err = chichi.RegisteredApp(app.name).New(&chichi.AppConfig{
		Settings:     connection.Settings,
		SetSettings:  setConnectionSettingsFunc(connectors.state, connection),
		OAuthAccount: accountCode,
		HTTPClient:   app.httpClient,
		Region:       chichi.PrivacyRegion(connection.Workspace().PrivacyRegion),
		WebhookURL:   webhookURL(connection, accountID),
	})
	return app
}

// EventRequest returns a request to dispatch an event to the app. typ specifies
// the type of event to send, event is the received event, extra contains the
// extra information conforming to the schema of the event type, extraSchema is
// the schema of the event type, and redacted indicates whether authentication
// data must be redacted in the returned request.
//
// If extra is nil, extraSchema should the invalid schema and vice versa.
//
// If the event type does not exist, it returns the ErrEventTypeNotExist error.
// If the schema of values is incompatible with the event type's schema, it
// returns a *SchemaError error.
//
// It panics if the app does not support the Events target, or if extraSchema is
// valid but not an Object.
func (app *App) EventRequest(ctx context.Context, typ string, event *Event, extra map[string]any, extraSchema types.Type, redacted bool) (*EventRequest, error) {
	if app.err != nil {
		return nil, app.err
	}
	appEvents, ok := app.inner.(chichi.AppEvents)
	if !ok {
		panic("app does not support the Events target")
	}
	schema, err := app.inner.Schema(ctx, chichi.Events, chichi.Destination, typ)
	if err != nil {
		return nil, err
	}
	// Check compatibility between the schema of the extra information and the schema of the event type.
	if err = verifySchemaCompatibilityForSendEvents(extraSchema, schema); err != nil {
		return nil, err
	}
	// Return the event request.
	return appEvents.EventRequest(ctx, typ, event, extra, extraSchema, redacted)
}

// EventTypes returns the app's event types.
// It panics if the app does not support the events target.
func (app *App) EventTypes(ctx context.Context) ([]*EventType, error) {
	if app.err != nil {
		return nil, app.err
	}
	return app.inner.(chichi.AppEvents).EventTypes(ctx)
}

// Schema returns the app's schema for the provided target. If target is
// state.Events, eventType represents the type of the event.
//
// For the users and the groups target, the returned schema contains only the
// properties compatible with the app's role. For the events target, the
// returned schema can be the invalid schema.
//
// If the event type does not exist, it returns the ErrEventTypeNotExist error.
// It panics if the app does not support the provided target.
func (app *App) Schema(ctx context.Context, target state.Target, eventType string) (types.Type, error) {
	return app.SchemaAsRole(ctx, app.role, target, eventType)
}

// SchemaAsRole is like Schema but returns the schema as the provided role,
// instead of the role of the app's connection.
// If the event type does not exist, it returns the ErrEventTypeNotExist error.
func (app *App) SchemaAsRole(ctx context.Context, role state.Role, target state.Target, eventType string) (types.Type, error) {
	if app.err != nil {
		return types.Type{}, app.err
	}
	switch target {
	case state.Events:
		return app.inner.Schema(ctx, chichi.Events, chichi.Role(role), eventType)
	case state.Users:
		return app.userSchema(ctx, types.Role(role))
	case state.Groups:
		schema, err := app.inner.Schema(ctx, chichi.Groups, chichi.Role(role), "")
		if err != nil {
			return types.Type{}, err
		}
		if !schema.Valid() {
			return types.Type{}, fmt.Errorf("connector %s returned an invalid group schema", app.name)
		}
		schema = schema.AsRole(types.Role(role))
		if !schema.Valid() {
			return types.Type{}, fmt.Errorf("connector has returned a schema without %s properties", strings.ToLower(role.String()))
		}
		return schema, nil
	}
	panic("unexpected target")
}

// SendEvent sends an event to the app and returns the HTTP response.
func (app *App) SendEvent(ctx context.Context, req *EventRequest) (*http.Response, error) {
	r, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bytes.NewReader(req.Body))
	if err != nil {
		return nil, err
	}
	r.Header = req.Header.Clone()
	return app.httpClient.Do(r)
}

// Users returns an iterator to iterate over the app's users. Each returned
// record will contain, in the Properties field, the properties in schema, with
// the same types.
//
// displayedProperty, when not empty, is the app property name from which the
// displayed property should be imported. If such property does not exist in the
// app's schema, or exists but its type is not compatible, no errors are
// returned and the displayed property  is simply not imported.
//
// lastChangeTime is the most recent lastChangeTime value read from the previous
// import.
//
// If the provided schema, that must be valid, does not conform with the app's
// source user schema, it returns a *SchemaError error.
func (app *App) Users(ctx context.Context, schema types.Type, displayedProperty string, lastChangeTime time.Time) (Records, error) {
	if app.err != nil {
		return nil, app.err
	}
	if !schema.Valid() {
		return nil, fmt.Errorf("schema is not valid")
	}
	// Check that the schema is aligned with the source user schema.
	userSchema, err := app.userSchema(ctx, types.SourceRole)
	if err != nil {
		return nil, fmt.Errorf("cannot get user schema: %s", err)
	}
	err = checkSchemaAlignment(schema, userSchema)
	if err != nil {
		return nil, err
	}
	// Determine and validate the property for the displayed property.
	var dp types.Property
	if displayedProperty != "" {
		dp, err = displayedPropertyFromSchema(userSchema, displayedProperty)
		if err != nil {
			slog.Warn("cannot determine the displayed property", "err", err)
		}
	}
	records := &appRecords{
		schema:            schema,
		timeLayouts:       app.timeLayouts,
		lastChangeTime:    lastChangeTime,
		appName:           app.name,
		inner:             app.inner,
		displayedProperty: dp,
	}
	return records, nil
}

// Writer returns a Writer to create and update records of the provided target.
// The target can be either Users or Groups.
//
// If the app does not support the provided target, it returns an
// ErrUnsupportedTarget error.
func (app *App) Writer(target state.Target, ack AckFunc) (Writer, error) {
	if app.err != nil {
		return nil, app.err
	}
	if target != state.Users && target != state.Groups {
		return nil, fmt.Errorf("target must be either Users or Groups")
	}
	if !app.targets.Contains(target) {
		return nil, ErrUnsupportedTarget
	}
	if ack == nil {
		return nil, errors.New("ack function is missing")
	}
	w := appWriter{
		ack:     ack,
		target:  chichi.Targets(target),
		records: app.inner.(chichi.AppRecords),
	}
	return &w, nil
}

// appWriter implements the Writer interface for apps.
type appWriter struct {
	ack     AckFunc
	target  chichi.Targets
	records chichi.AppRecords
	closed  bool
}

func (w *appWriter) Close(ctx context.Context) error {
	w.closed = true
	return nil
}

func (w *appWriter) Write(ctx context.Context, gid uuid.UUID, record Record) bool {
	if w.closed {
		panic("connectors: Write called on a closed writer")
	}
	var err error
	if record.ID == "" {
		err = w.records.Create(ctx, w.target, record.Properties)
	} else {
		err = w.records.Update(ctx, w.target, record.ID, record.Properties)
	}
	w.ack(err, []uuid.UUID{gid})
	return true
}

// appRecords implements the Records interface for apps.
type appRecords struct {
	schema            types.Type
	timeLayouts       *state.TimeLayouts
	lastChangeTime    time.Time
	appName           string
	inner             chichi.App
	last              bool
	err               error
	closed            bool
	displayedProperty types.Property
}

func (r *appRecords) All(ctx context.Context) Seq[Record] {

	return func(yield func(Record) bool) {

		if r.closed {
			r.err = errors.New("connectors: For called on a closed Records")
			return
		}

		cursor := chichi.Cursor{
			LastChangeTime: r.lastChangeTime,
		}

		properties := types.Properties(r.schema)
		names := make([]string, len(properties))
		propertyByName := make(map[string]*types.Property, len(properties))
		for i, p := range properties {
			names[i] = p.Name
			propertyByName[p.Name] = &p
		}

		if r.displayedProperty.Name != "" && !slices.Contains(names, r.displayedProperty.Name) {
			names = append(names, r.displayedProperty.Name)
		}

		for {

			// Retrieve the users.
			users, next, err := r.inner.(chichi.AppRecords).Records(ctx, chichi.Users, names, cursor)
			eof := err == io.EOF
			if err != nil && !eof {
				r.err = err
				return
			}
			if len(users) == 0 {
				if !eof {
					r.err = fmt.Errorf("%s returned zero users but did not return io.EOF", r.appName)
					return
				}
				if next != "" {
					r.err = fmt.Errorf("%s returned zero users but returned a non-empty next value", r.appName)
				}
				return
			}
			cursor.Next = next
			last := len(users) - 1

			// Normalize the returned users.

			for i, appUser := range users {

				select {
				case <-ctx.Done():
					r.err = ctx.Err()
					return
				default:
				}

				user := Record{
					ID:           appUser.ID,
					Associations: appUser.Associations,
				}

				// Read the displayed property.
				if r.displayedProperty.Name != "" {
					for p, v := range appUser.Properties {
						if p == r.displayedProperty.Name {
							normalizedValue, err := normalize(r.displayedProperty.Name, r.displayedProperty.Type, v, r.displayedProperty.Nullable, r.timeLayouts)
							if err != nil {
								slog.Warn("displayed property value cannot be normalized", "err", err)
								break
							}
							user.DisplayedProperty, err = displayedPropertyToString(normalizedValue)
							if err != nil {
								slog.Warn("invalid displayed property value", "err", err)
								break
							}
							break
						}
					}
				}

				// Read the properties.
				user.Properties = make(map[string]any, len(properties))
				for _, p := range properties {
					value, ok := appUser.Properties[p.Name]
					if !ok {
						user.Err = fmt.Errorf(`app did not return a value for the property %q`, p.Name)
						break
					}
					value, err = normalize(p.Name, p.Type, value, p.Nullable, r.timeLayouts)
					if err != nil {
						user.Err = err
						break
					}
					user.Properties[p.Name] = value
				}
				if len(user.Properties) != len(properties) {
					// Users method of the connector is permitted to return more properties
					// than those requested, so if necessary, remove those that are not
					// requested.
					for name := range user.Properties {
						if _, ok := propertyByName[name]; !ok {
							delete(propertyByName, name)
						}
					}
				}

				// Read the last change time.
				user.LastChangeTime = appUser.LastChangeTime.UTC()
				if err = validateTimestamp(user.LastChangeTime); err != nil {
					return
				}
				if user.LastChangeTime.After(cursor.LastChangeTime) {
					cursor.LastChangeTime = user.LastChangeTime
				}

				r.last = i == last

				if !yield(user) {
					return
				}

			}

			if eof {
				return
			}

		}

	}

}

func (r *appRecords) Close() error {
	r.closed = true
	return nil
}

func (r *appRecords) Err() error {
	return r.err
}

func (r *appRecords) Last() bool {
	return r.last
}

type schema struct {
	lock    chan struct{}
	schemas [3]types.Type
}

// userSchema returns the user schema with the provided role.
func (app *App) userSchema(ctx context.Context, role types.Role) (types.Type, error) {
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
	schema, err := app.inner.Schema(ctx, chichi.Users, chichi.Role(role), "")
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

// verifySchemaCompatibilityForSendEvents verifies whether t1 and t2 are
// compatible for sending events. If they are compatible, values satisfying t1
// can be safely passed to the SendEvent and PreviewSendEvent methods, where t2
// represents the schema of the event type to send. In such cases, it guarantees
// that the values also satisfy t2.
//
// An invalid schema is handled as if it were an object without properties.
//
// It returns a *SchemaError error if t1 and t2 are not compatible. It panics if
// the schema is invalid.
//
// It panics if t1 and t2 are valid but non-Object.
func verifySchemaCompatibilityForSendEvents(t1, t2 types.Type) error {
	if !t1.Valid() || !t2.Valid() {
		switch {
		case t1.Valid():
			properties := types.Properties(t1)
			return &SchemaError{Msg: fmt.Sprintf(`property %q is no longer present`, properties[0].Name)}
		case t2.Valid():
			for _, p2 := range t2.Properties() {
				if p2.Required {
					return &SchemaError{Msg: fmt.Sprintf(`there is a new required property %q`, p2.Name)}
				}
			}
		}
		return nil
	}
	if t1.Kind() != types.ObjectKind || t2.Kind() != types.ObjectKind {
		panic("t1 and t2 must be invalid or objects")
	}
	// t1 and t2 are valid.
	var verify func(name string, t1, t2 types.Type) error
	verify = func(name string, t1, t2 types.Type) error {
		if t1.EqualTo(t2) {
			return nil
		}
		pt1 := t1.Kind()
		pt2 := t2.Kind()
		if pt1 != pt2 {
			return &SchemaError{Msg: fmt.Sprintf("type of the %q property has changed from %s to %s", name, t1, t2)}
		}
		switch pt1 {
		case types.IntKind:
			min1, max1 := t1.IntRange()
			min2, max2 := t2.IntRange()
			if min1 < min2 {
				return &SchemaError{Msg: fmt.Sprintf("type of the %q property is changed; minimum value is changed from %d to %d", name, min1, min2)}
			}
			if max2 < max1 {
				return &SchemaError{Msg: fmt.Sprintf("type of the %q property is changed; maximum value is changed from %d to %d", name, max1, max2)}
			}
			return nil
		case types.UintKind:
			min1, max1 := t1.UintRange()
			min2, max2 := t2.UintRange()
			if min1 < min2 {
				return &SchemaError{Msg: fmt.Sprintf("type of the %q property is changed; minimum value is changed from %d to %d", name, min1, min2)}
			}
			if max2 < max1 {
				return &SchemaError{Msg: fmt.Sprintf("type of the %q property is changed; maximum value is changed from %d to %d", name, max1, max2)}
			}
			return nil
		case types.FloatKind:
			if t2.BitSize() < t1.BitSize() {
				return &SchemaError{Msg: fmt.Sprintf("type of the %q property is changed; bit size is changed from 64 to 32", name)}
			}
			min1, max1 := t1.FloatRange()
			min2, max2 := t2.FloatRange()
			if min1 < min2 {
				return &SchemaError{Msg: fmt.Sprintf("type of the %q property is changed; minimum value is changed from %g to %g", name, min1, min2)}
			}
			if max2 < max1 {
				return &SchemaError{Msg: fmt.Sprintf("type of the %q property is changed; maximum value is changed from %g to %g", name, max1, max2)}
			}
			return nil
		case types.DecimalKind:
			s1 := t1.Scale()
			s2 := t2.Scale()
			if s1 > s2 || t1.Precision()-s1 > t2.Precision()-s2 {
				return &SchemaError{Msg: fmt.Sprintf(`type of property %q has changed from %s to %s`, name, t1, t2)}
			}
			min1, max1 := t1.DecimalRange()
			min2, max2 := t2.DecimalRange()
			if min1.LessThan(min2) {
				return &SchemaError{Msg: fmt.Sprintf(`type of property %q has changed; minimum value is changed from %s to %s`, name, min1, min2)}
			}
			if max2.LessThan(max1) {
				return &SchemaError{Msg: fmt.Sprintf(`type of property %q has changed; maximum value is changed from %s to %s`, name, max1, max2)}
			}
			return nil
		case types.TextKind:
			if values1 := t1.Values(); values1 != nil {
				values2 := t2.Values()
				if values2 == nil {
					return &SchemaError{Msg: fmt.Sprintf(`type of property %q has changed; it is no longer limited to specific values`, name)}
				}
				for _, v1 := range values1 {
					found := false
					for _, v2 := range values2 {
						if v1 == v2 {
							found = true
							break
						}
					}
					if !found {
						return &SchemaError{Msg: fmt.Sprintf(`type of property %q has changed; value %q is no longer allowed`, name, v1)}
					}
				}
				return nil
			}
			if rx1 := t1.Regexp(); rx1 != nil {
				rx2 := t2.Regexp()
				if rx2 == nil {
					return &SchemaError{Msg: fmt.Sprintf(`type of property %q has changed; it no longer validates against a regular expression`, name)}
				}
				if rx1.String() != rx2.String() {
					return &SchemaError{Msg: fmt.Sprintf(`type of property %q has changed; it validates against a different regular expression`, name)}
				}
				return nil
			}
			if max2, ok := t2.ByteLen(); ok {
				max1, ok := t1.ByteLen()
				if !ok {
					return &SchemaError{Msg: fmt.Sprintf(`type of property %q has changed; it is now restricted in byte length`, name)}
				}
				if max1 > max2 {
					return &SchemaError{Msg: fmt.Sprintf(`type of property %q has changed; maximum length in bytes has changed from %d to %d`, name, max1, max2)}
				}
			}
			if max2, ok := t2.CharLen(); ok {
				max1, ok := t1.CharLen()
				if !ok {
					return &SchemaError{Msg: fmt.Sprintf(`type of property %q has changed; it is now restricted in character length`, name)}
				}
				if max1 > max2 {
					return &SchemaError{Msg: fmt.Sprintf(`type of property %q has changed; maximum length in characters has changed from %d to %d`, name, max1, max2)}
				}
			}
			return nil
		case types.ArrayKind:
			return verify(name+"[]", t1.Elem(), t2.Elem())
		case types.ObjectKind:
			for _, p1 := range t1.Properties() {
				path := p1.Name
				if name != "" {
					path = name + "." + path
				}
				p2, ok := t2.Property(p1.Name)
				if !ok {
					return &SchemaError{Msg: fmt.Sprintf(`property %q is no longer present`, path)}
				}
				if !p1.Required && p2.Required {
					return &SchemaError{Msg: fmt.Sprintf(`property %q has become required`, path)}
				}
				err := verify(path, p1.Type, p2.Type)
				if err != nil {
					return err
				}
			}
			for _, p2 := range t2.Properties() {
				if p2.Required {
					if _, ok := t1.Property(p2.Name); !ok {
						path2 := p2.Name
						if name != "" {
							path2 = name + "." + path2
						}
						return &SchemaError{Msg: fmt.Sprintf(`there is a new required property %q`, path2)}
					}
				}
			}
			return nil
		case types.MapKind:
			return verify(name, t1.Elem(), t2.Elem())
		}
		return nil
	}
	return verify("", t1, t2)
}
