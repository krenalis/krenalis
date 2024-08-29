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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/apis/connectors/httpclient"
	"github.com/meergo/meergo/apis/schemas"
	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/types"

	"github.com/shopspring/decimal"
)

// appUpdateBatchSize is the number of app updates to process in each batch.
const appUpdateBatchSize = 1000

type (
	EventRequest   = meergo.EventRequest
	WebhookPayload = meergo.WebhookPayload
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
	inner       meergo.App
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
	app.inner, app.err = meergo.RegisteredApp(app.name).New(&meergo.AppConfig{
		Settings:     connection.Settings,
		SetSettings:  setConnectionSettingsFunc(connectors.state, connection),
		OAuthAccount: accountCode,
		HTTPClient:   app.httpClient,
		Region:       meergo.PrivacyRegion(connection.Workspace().PrivacyRegion),
		WebhookURL:   webhookURL(connection, accountID),
	})
	return app
}

// EventRequest returns a request to dispatch an event to the app. event is the
// event to dispatch, eventType is the type of event to dispatch, schema is its
// schema, properties are the property values conforming to the schema, and
// redacted indicates whether authentication data must be redacted in the
// returned request.
//
// If the event type does not have a schema, schema is the invalid schema and
// properties is nil.
//
// If the event type does not exist, it returns the ErrEventTypeNotExist error.
// If the schema is not aligned to the event type's schema, it returns a
// *schemas.Error error.
//
// It panics if the app does not support the Events target, or if schema is
// valid but not an Object.
func (app *App) EventRequest(ctx context.Context, event *Event, eventType string, schema types.Type, properties map[string]any, redacted bool) (*EventRequest, error) {
	if app.err != nil {
		return nil, app.err
	}
	appEvents, ok := app.inner.(meergo.AppEvents)
	if !ok {
		panic("app does not support the Events target")
	}
	eventTypeSchema, err := app.inner.Schema(ctx, meergo.Events, meergo.Destination, eventType)
	if err != nil {
		return nil, err
	}
	// Check that schema is aligned with the event type's schema.
	createOnly := state.CreateOnly
	err = schemas.CheckAlignment(schema, eventTypeSchema, &createOnly)
	if err != nil {
		return nil, err
	}
	// If schema is invalid, properties is nil. The EventRequest method,
	// when the event schema is valid, requires properties to be non-nil.
	if properties == nil && eventTypeSchema.Valid() {
		properties = map[string]any{}
	}
	// Return the event request.
	return appEvents.EventRequest(ctx, event, eventType, eventTypeSchema, properties, redacted)
}

// EventTypes returns the app's event types.
// It panics if the app does not support the events target.
func (app *App) EventTypes(ctx context.Context) ([]*EventType, error) {
	if app.err != nil {
		return nil, app.err
	}
	return app.inner.(meergo.AppEvents).EventTypes(ctx)
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
		return app.inner.Schema(ctx, meergo.Events, meergo.Role(role), eventType)
	case state.Users:
		return app.userSchema(ctx, types.Role(role))
	case state.Groups:
		schema, err := app.inner.Schema(ctx, meergo.Groups, meergo.Role(role), "")
		if err != nil {
			return types.Type{}, err
		}
		if !schema.Valid() {
			return types.Type{}, fmt.Errorf("connector %s returned an invalid group schema", app.name)
		}
		schema = types.AsRole(schema, types.Role(role))
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
	return app.httpClient.DoIdempotent(r, req.Idempotent)
}

// Users returns an iterator to iterate over the app's users. Each returned
// record will contain, in the Properties field, the properties in schema, with
// the same types.
//
// lastChangeTime is the most recent lastChangeTime value read from the previous
// import.
//
// If the provided schema, that must be valid, does not align with the app's
// source schema, it returns a *schemas.Error error.
func (app *App) Users(ctx context.Context, schema types.Type, lastChangeTime time.Time) (Records, error) {
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
	err = schemas.CheckAlignment(schema, userSchema, nil)
	if err != nil {
		return nil, err
	}
	records := &appRecords{
		schema:         schema,
		timeLayouts:    app.timeLayouts,
		lastChangeTime: lastChangeTime,
		appName:        app.name,
		inner:          app.inner,
	}
	return records, nil
}

// Writer returns a Writer to create and update records of the provided action.
// If the action's output schema does not align with the app's destination
// schema, it returns a *schemas.Error error.
func (app *App) Writer(ctx context.Context, action *state.Action, ack AckFunc) (Writer, error) {
	if app.err != nil {
		return nil, app.err
	}
	appSchema, err := app.SchemaAsRole(ctx, state.Destination, state.Users, "")
	if err != nil {
		return nil, err
	}
	// Check that the action's output schema is aligned with the app destination schema.
	err = schemas.CheckAlignment(action.OutSchema, appSchema, action.ExportMode)
	if err != nil {
		return nil, err
	}
	if ack == nil {
		return nil, errors.New("ack function is missing")
	}
	return newAppWriter(ack, action.Target, action.OutSchema, app.inner.(meergo.AppRecords)), nil
}

// appWriter implements the Writer interface for apps.
type appWriter struct {
	ack     AckFunc
	target  meergo.Targets
	types   map[string]types.Type
	records meergo.AppRecords

	// Following fields are used by the doUpdates method to update records in the app.
	updates    map[string]map[string]any // id -> property -> value
	properties []string
	ids        []string

	closed bool
}

func newAppWriter(ack AckFunc, target state.Target, schema types.Type, records meergo.AppRecords) *appWriter {
	w := &appWriter{
		ack:     ack,
		target:  meergo.Targets(target),
		types:   map[string]types.Type{},
		records: records,
		updates: map[string]map[string]any{},
	}
	for _, p := range schema.Properties() {
		w.types[p.Name] = p.Type
	}
	return w
}

func (w *appWriter) Close(ctx context.Context) error {
	w.closed = true
	if len(w.updates) > 0 {
		w.doUpdates(ctx)
	}
	return nil
}

func (w *appWriter) Write(ctx context.Context, id string, properties map[string]any) bool {
	if w.closed {
		panic("connectors: Write called on a closed writer")
	}
	if id == "" {
		err := w.records.Create(ctx, w.target, properties)
		w.ack([]string{id}, err)
		return true
	}
	w.updates[id] = properties
	if len(w.updates) == appUpdateBatchSize {
		w.doUpdates(ctx)
	}
	return true
}

// ErrRecordNotExist is returned when a record with the specified identifier
// does not exist in the application. It is returned in the Record.Err field.
var ErrRecordNotExist = errors.New("record not exist")

// doUpdates applies pending updates and avoids updating records that have not
// changed.
func (w *appWriter) doUpdates(ctx context.Context) {
	var err error
	defer func() {
		if len(w.updates) > 0 {
			if err == nil {
				err = ErrRecordNotExist
			}
			w.ids = slices.Grow(w.ids, len(w.updates))
			w.ids = w.ids[:len(w.updates)]
			i := 0
			for id := range w.updates {
				w.ids[i] = id
			}
			w.ack(w.ids, err)
		}
		clear(w.updates)
		w.properties = w.properties[0:0]
		w.ids = w.ids[0:0]
	}()
	w.ids = slices.Grow(w.ids, len(w.updates))
	w.ids = w.ids[:len(w.updates)]
	i := 0
	for id, properties := range w.updates {
		w.ids[i] = id
		for name := range properties {
			if k, ok := slices.BinarySearch(w.properties, name); !ok {
				w.properties = slices.Insert(w.properties, k, name)
			}
		}
		i++
	}
	var cursor string
	var records []meergo.Record
	var eof bool
	for !eof {
		records, cursor, err = w.records.Records(ctx, w.target, time.Time{}, w.ids, w.properties, cursor)
		if err != nil {
			eof = err == io.EOF
			if !eof {
				return
			}
			err = nil
		}
		for _, record := range records {
			properties, ok := w.updates[record.ID]
			if !ok {
				// Connectors may return more records than the requested ones.
				continue
			}
			update := false
			for name, v := range properties {
				v2, ok := record.Properties[name]
				if !ok {
					update = true
					break
				}
				t := w.types[name]
				if !sameValue(t, v, v2) {
					update = true
					break
				}
			}
			if update {
				err = w.records.Update(ctx, w.target, record.ID, properties)
			}
			w.ack([]string{record.ID}, err)
			delete(w.updates, record.ID)
		}
	}
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
	schema, err := app.inner.Schema(ctx, meergo.Users, meergo.Role(role), "")
	if err != nil {
		return types.Type{}, err
	}
	var schemas [3]types.Type
	for r := types.BothRole; r <= types.DestinationRole; r++ {
		schemas[r] = types.AsRole(schema, r)
		if !schemas[r].Valid() {
			return types.Type{}, fmt.Errorf("connection has returned a schema without %s properties", strings.ToLower(role.String()))
		}
	}
	app.users.schemas = schemas
	return app.users.schemas[role], nil
}

// sameValue checks if v and v2 have the same value, with t being the type of v.
// If it returns true, it means that v and v2 have the same value; otherwise,
// nothing can be concluded.
func sameValue(t types.Type, v, v2 any) bool {
	if v == nil {
		return v2 == nil
	}
	switch t.Kind() {
	default:
		return v == v2
	case types.DecimalKind:
		v1 := v.(decimal.Decimal)
		v2, ok := v2.(decimal.Decimal)
		return ok && v1.Equal(v2)
	case types.JSONKind:
		v1, ok1 := v.(json.RawMessage)
		v2, ok2 := v2.(json.RawMessage)
		return ok1 && ok2 && bytes.Equal(v1, v2)
	case types.ArrayKind:
		v1 := v.([]any)
		a2, ok := v2.([]any)
		if !ok || len(v1) != len(a2) {
			return false
		}
		et := t.Elem()
		for i, e1 := range v1 {
			if !sameValue(et, e1, a2[i]) {
				return false
			}
		}
		return true
	case types.ObjectKind:
		v1 := v.(map[string]any)
		v2, ok := v2.(map[string]any)
		if !ok || len(v1) != len(v2) {
			return false
		}
		for _, p := range t.Properties() {
			e1, ok := v1[p.Name]
			if !ok {
				continue
			}
			e2, ok := v2[p.Name]
			if !ok || !sameValue(p.Type, e1, e2) {
				return false
			}
		}
		return true
	case types.MapKind:
		v1 := v.(map[string]any)
		v2, ok := v2.(map[string]any)
		if !ok || len(v1) != len(v2) {
			return false
		}
		et := t.Elem()
		for k, e1 := range v1 {
			e2, ok := v2[k]
			if !ok || !sameValue(et, e1, e2) {
				return false
			}
		}
		return true
	}
}

// appRecords implements the Records interface for apps.
type appRecords struct {
	schema         types.Type
	timeLayouts    *state.TimeLayouts
	lastChangeTime time.Time
	appName        string
	inner          meergo.App
	last           bool
	err            error
	closed         bool
}

func (r *appRecords) All(ctx context.Context) iter.Seq[Record] {

	return func(yield func(Record) bool) {

		if r.closed {
			r.err = errors.New("connectors: For called on a closed Records")
			return
		}

		var cursor string

		properties := types.Properties(r.schema)
		names := make([]string, len(properties))
		for i, p := range properties {
			names[i] = p.Name
		}

		for {

			// Retrieve the users.
			var users []meergo.Record
			var err error
			users, cursor, err = r.inner.(meergo.AppRecords).Records(ctx, meergo.Users, r.lastChangeTime, nil, names, cursor)
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
				if cursor != "" {
					r.err = fmt.Errorf("%s returned zero users but returned a non-empty next value", r.appName)
				}
				return
			}
			last := len(users) - 1

			// Normalize the returned users.

			for i, user := range users {

				select {
				case <-ctx.Done():
					r.err = ctx.Err()
					return
				default:
				}

				record := Record{
					ID:             user.ID,
					LastChangeTime: user.LastChangeTime.UTC(),
					Associations:   user.Associations,
				}

				// Read the properties.
				record.Properties = make(map[string]any, len(properties))
				for _, p := range properties {
					v, ok := user.Properties[p.Name]
					if !ok {
						if !p.ReadOptional {
							record.Err = newNormalizationErrorf(p.Name, "does not have a value, but the property is not optional for reading")
							break
						}
						continue
					}
					v, err = normalize(p.Name, p.Type, v, p.Nullable, r.timeLayouts)
					if err != nil {
						record.Err = err
						break
					}
					record.Properties[p.Name] = v
				}

				// validate the last change time.
				if err = validateLastChangeTime(record.LastChangeTime); err != nil {
					return
				}

				r.last = i == last

				if !yield(record) {
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
