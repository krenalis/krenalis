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
	"iter"
	"net/http"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/connectors/appwriter"
	"github.com/meergo/meergo/core/connectors/httpclient"
	"github.com/meergo/meergo/core/schemas"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/util"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

// App represents the app of an app connection.
type App struct {
	connector   string
	role        state.Role
	timeLayouts *state.TimeLayouts
	httpClient  *httpclient.Client
	users       schema
	targets     state.ConnectorTargets
	inner       any
	err         error
}

type appSchemaConnector interface {
	// Schema returns the schema of the specified target in the specified role. For
	// Users or Groups, role can be Source or Destination, and it returns their
	// respective schemas. For Events, role is Destination, and it returns the
	// schema of the specified event type.
	//
	// For events, the returned schema describes properties required by the
	// connector to dispatch an event of this type. Actions based on the specified
	// event type will have a transformation that, given the received event,
	// provides the properties required by the connector. These properties, along
	// with the received event, are passed to the connector's EventRequest method.
	//
	// If no extra information is needed for the event type, the returned schema is
	// the invalid schema. If the event type does not exist, it returns the
	// ErrEventTypeNotExist error.
	Schema(ctx context.Context, target meergo.Targets, role meergo.Role, eventType string) (types.Type, error)
}

type appRecordsConnector interface {
	// Records returns the records of the specified target. The target can only be
	// either Users or Groups, and it must be a target supported by the connector.
	//
	// If lastChangeTime is not the zero time, only the records changed or created
	// at or after that time will be returned, and its precision is limited to
	// microseconds. If ids is not nil, only records with identifiers in ids should
	// be returned, if any.
	//
	// properties are the names of the properties to read. cursor represents the
	// position from which to start reading the records; it is the cursor value
	// returned by the previous call in a paginated query. Subsequent calls will use
	// this cursor value to retrieve the next batch of records.
	//
	// schema must be a recent schema returned by the Schema method of the
	// connector. There is no guarantee that the returned properties will match this
	// schema, so the caller must validate them.
	//
	// Records may return duplicate records, i.e., records with the same ID. The
	// caller is responsible for deduplicating them.
	//
	// The string return value is used as the cursor in the subsequent call. It can
	// be any UTF-8 encoded string, including an empty string. If there are no more
	// records to return, the method returns the last records read (if any) along
	// with the io.EOF error.
	//
	// In case of an error, it returns a non-nil and non-EOF error.
	Records(ctx context.Context, target meergo.Targets, lastChangeTime time.Time, ids, properties []string, cursor string, schema types.Type) ([]meergo.Record, string, error)
}

type appEventsConnector interface {
	// EventRequest returns a request to send an event to the app. event is the
	// event to send, eventType is the type of event to send, schema is its schema,
	// properties are the property values conforming to the schema, and redacted
	// indicates whether authentication data must be redacted in the returned
	// request.
	//
	// If the event type does not have a schema, schema is the invalid schema and
	// properties is nil.
	//
	// This method is safe for concurrent use by multiple goroutines. If the
	// specified event type does not exist, it returns the ErrEventTypeNotExist
	// error.
	EventRequest(ctx context.Context, event RawEvent, eventType string, schema types.Type, properties map[string]any, redacted bool) (*meergo.EventRequest, error)

	// EventTypes returns the event types of the connector's instance.
	EventTypes(ctx context.Context) ([]*EventType, error)
}

// TODO(marco): implement webhooks
//type webhookConnector interface {
//	// ReceiveWebhook receives a webhook request and returns its payloads. If
//	// webhooks are per connection, role is the connection's role, otherwise is
//	// Both. It returns the ErrWebhookUnauthorized error is the request was not
//	// authorized. The context is the request's context.
//	ReceiveWebhook(r *http.Request, role meergo.Role) ([]meergo.WebhookPayload, error)
//}

type appOAuthConnector interface {
	// OAuthAccount returns the app's account associated with the OAuth
	// authorization.
	OAuthAccount(ctx context.Context) (string, error)
}

// App returns an app for the provided connection. Errors are deferred until an
// app's method is called. It panics if connection is not an app connection.
func (connectors *Connectors) App(connection *state.Connection) *App {
	connector := connection.Connector()
	var targets state.ConnectorTargets
	if connection.Role == state.Source {
		targets = connector.SourceTargets
	} else {
		targets = connector.DestinationTargets
	}
	app := &App{
		connector:   connector.Name,
		role:        connection.Role,
		timeLayouts: &connector.TimeLayouts,
		httpClient:  connectors.http.ConnectionClient(connection.ID),
		users:       schema{lock: make(chan struct{}, 1)},
		targets:     targets,
	}
	// var accountID int // TODO(marco): implement webhooks
	var accountCode string
	if a, ok := connection.Account(); ok {
		// accountID = a.ID // TODO(marco): implement webhooks
		accountCode = a.Code
	}
	app.inner, app.err = meergo.RegisteredApp(app.connector).New(&meergo.AppConfig{
		Settings:     connection.Settings,
		SetSettings:  setConnectionSettingsFunc(connectors.state, connection),
		OAuthAccount: accountCode,
		HTTPClient:   app.httpClient,
		// WebhookURL:   webhookURL(connection, accountID), // TODO(marco): implement webhooks
	})
	return app
}

// Connector returns the name of the app connector.
func (app *App) Connector() string {
	return app.connector
}

// EventRequest returns a request to send an event to the app. event is the
// event to send, eventType is the type of event to send, schema is its schema,
// properties are the property values conforming to the schema, and redacted
// indicates whether authentication data must be redacted in the returned
// request.
//
// If the event type does not have a schema, schema is the invalid schema and
// properties is nil.
//
// If the event type does not exist, it returns the meergo.ErrEventTypeNotExist
// error. If the schema is not aligned to the event type's schema, it returns a
// *schemas.Error error. If the connector returns an error it returns a
// *UnavailableError error.
//
// It panics if the app does not support the Events target, or if schema is
// valid but not an object.
func (app *App) EventRequest(ctx context.Context, event RawEvent, eventType string, schema types.Type, properties map[string]any, redacted bool) (*meergo.EventRequest, error) {
	if app.err != nil {
		return nil, app.err
	}
	eventTypeSchema, err := app.inner.(appSchemaConnector).Schema(ctx, meergo.EventsTarget, meergo.Destination, eventType)
	if err != nil {
		return nil, connectorError(err)
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
	request, err := app.inner.(appEventsConnector).EventRequest(ctx, event, eventType, eventTypeSchema, properties, redacted)
	if err != nil {
		return nil, connectorError(err)
	}
	return request, nil
}

// EventTypes returns the app's event types.
// If the connector returns an error, it returns an *UnavailableError error.
// It panics if the app does not support the events target.
func (app *App) EventTypes(ctx context.Context) ([]*EventType, error) {
	if app.err != nil {
		return nil, app.err
	}
	eventTypes, err := app.inner.(appEventsConnector).EventTypes(ctx)
	if err != nil {
		return nil, connectorError(err)
	}
	for _, typ := range eventTypes {
		if err := util.ValidateStringField("event type", typ.ID, 100); err != nil {
			return nil, err
		}
	}
	return eventTypes, nil
}

// Schema returns the app's schema for the provided target. If target is
// state.Events, eventType represents the type of the event.
//
// If the target is state.Events and the event type refers to an app event for
// which no schema is expected, this method returns the invalid type and no
// errors.
//
// For the users and the groups target, the returned schema contains only the
// properties compatible with the app's role. For the events target, the
// returned schema can be the invalid schema.
//
// If the event type does not exist, it returns the meergo.ErrEventTypeNotExist
// error. It panics if the app does not support the provided target.
func (app *App) Schema(ctx context.Context, target state.Target, eventType string) (types.Type, error) {
	return app.SchemaAsRole(ctx, app.role, target, eventType)
}

// SchemaAsRole is like Schema but returns the schema as the provided role,
// instead of the role of the app's connection.
//
// If the target is state.Events and the event type refers to an app event for
// which no schema is expected, this method returns the invalid type and no
// errors.
//
// If the event type does not exist, it returns the meergo.ErrEventTypeNotExist
// error. If the connector returns an error, it returns a *UnavailableError
// error.
// It panics if role is not Source or Destination.
func (app *App) SchemaAsRole(ctx context.Context, role state.Role, target state.Target, eventType string) (types.Type, error) {
	if app.err != nil {
		return types.Type{}, app.err
	}
	if role != state.Source && role != state.Destination {
		panic("invalid role")
	}
	switch target {
	case state.Events:
		if role != state.Destination {
			panic("invalid role")
		}
		schema, err := app.inner.(appSchemaConnector).Schema(ctx, meergo.EventsTarget, meergo.Destination, eventType)
		if err != nil {
			return types.Type{}, connectorError(err)
		}
		if !schema.Valid() {
			return types.Type{}, nil
		}
		return types.AsRole(schema, types.Destination), nil
	case state.Users:
		schema, err := app.userSchema(ctx, role)
		if err != nil {
			return types.Type{}, connectorError(err)
		}
		return schema, nil
	case state.Groups:
		schema, err := app.inner.(appSchemaConnector).Schema(ctx, meergo.GroupsTarget, meergo.Role(role), "")
		if err != nil {
			return types.Type{}, connectorError(err)
		}
		if !schema.Valid() {
			return types.Type{}, fmt.Errorf("connector %s returned an invalid group schema", app.connector)
		}
	}
	panic("unexpected target")
}

// SendEvent sends an event to the app and returns the HTTP response.
func (app *App) SendEvent(ctx context.Context, req *meergo.EventRequest) (*http.Response, error) {
	r, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bytes.NewReader(req.Body))
	if err != nil {
		return nil, err
	}
	r.Header = req.Header.Clone()
	return app.httpClient.DoIdempotent(r, false)
}

// Users returns an iterator to iterate over the app's users. Each returned
// record will contain, in the Properties field, the properties in schema, with
// the same types.
//
// If lastChangeTime is not the zero time, it must be in UTC, and its year
// cannot be before 1900. In this case, only records changed or created at or
// after that time will be returned, with a precision limited to microseconds.
//
// If the connector returns an error, it returns an *UnavailableError error. If
// the provided schema, that must be valid, does not align with the app's source
// schema, it returns a *schemas.Error error.
//
// The Err method of the returned iterator may return an *UnavailableError if
// the connector encounters an error.
func (app *App) Users(ctx context.Context, schema types.Type, lastChangeTime time.Time) (Records, error) {
	if app.err != nil {
		return nil, app.err
	}
	if !schema.Valid() {
		return nil, fmt.Errorf("schema is not valid")
	}
	// Check that the user schema is aligned with the app user schema.
	appSchema, err := app.userSchema(ctx, state.Source)
	if err != nil {
		return nil, err
	}
	err = schemas.CheckAlignment(schema, appSchema, nil)
	if err != nil {
		return nil, err
	}
	if !lastChangeTime.IsZero() {
		if lastChangeTime.Location() != time.UTC {
			return nil, fmt.Errorf("lastChangeTime is not UTC")
		}
		if lastChangeTime.Year() < 1900 {
			return nil, fmt.Errorf("lastChangeTime's year is before 1900")
		}
		lastChangeTime = lastChangeTime.Truncate(time.Microsecond)
	}
	records := &appRecords{
		schema:         schema,
		appSchema:      appSchema,
		timeLayouts:    app.timeLayouts,
		lastChangeTime: lastChangeTime,
		connector:      app.connector,
		inner:          app.inner,
	}
	return records, nil
}

// Writer returns a Writer for creating and updating users or groups in the app.
// outSchema is the output schema of the action, exportMode is the export mode,
// and target is the target of the action. ack is the function that will receive
// the acknowledgments and cannot be nil.
//
// If the action's output schema does not align with the app's destination
// schema, it returns a *schemas.Error indicating the mismatch.
func (app *App) Writer(ctx context.Context, outSchema types.Type, exportMode state.ExportMode, target state.Target, ack AckFunc) (Writer, error) {
	if app.err != nil {
		return nil, app.err
	}
	if ack == nil {
		return nil, errors.New("ack function is missing")
	}
	// Get the destination schema.
	destinationSchema, err := app.SchemaAsRole(ctx, state.Destination, state.Users, "")
	if err != nil {
		return nil, err
	}
	// Check that the output schema is aligned with the destination schema.
	err = schemas.CheckAlignment(outSchema, destinationSchema, &exportMode)
	if err != nil {
		return nil, err
	}
	writer := appwriter.New(appwriter.AckFunc(ack), target, app.inner.(appwriter.UpsertableApp), app.connector)
	return writer, nil
}

// userSchema returns the user schema with the provided role.
// If the connector returns an error, it returns an *UnavailableError error.
// It panics if role is not Source or Destination.
func (app *App) userSchema(ctx context.Context, role state.Role) (types.Type, error) {
	if role != state.Source && role != state.Destination {
		panic("invalid role")
	}
	select {
	case <-ctx.Done():
		return types.Type{}, errors.New("canceled context")
	case app.users.lock <- struct{}{}:
	}
	defer func() {
		<-app.users.lock
	}()
	if schema := app.users.schemas[role-1]; schema.Valid() {
		return schema, nil
	}
	schema, err := app.inner.(appSchemaConnector).Schema(ctx, meergo.UsersTarget, meergo.Role(role), "")
	if err != nil {
		return types.Type{}, connectorError(fmt.Errorf("cannot get user schema: %s", err))
	}
	if !schema.Valid() {
		return types.Type{}, connectorError(fmt.Errorf("connector %s returned an invalid %s schema", app.connector, strings.ToLower(role.String())))
	}
	schema = types.AsRole(schema, types.Role(role))
	app.users.schemas[role-1] = schema
	return schema, nil
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
		return bytes.Equal(v.(json.Value), v2.(json.Value))
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
	appSchema      types.Type
	timeLayouts    *state.TimeLayouts
	lastChangeTime time.Time
	connector      string
	inner          any
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

		// processedIDs contains the already read ID.
		// It is used to deduplicate returned records.
		processedIDs := map[string]any{}

		for {

			// Retrieve the users.
			var users []meergo.Record
			var err error
			users, cursor, err = r.inner.(appRecordsConnector).Records(ctx, meergo.UsersTarget, r.lastChangeTime, nil, names, cursor, r.appSchema)
			eof := err == io.EOF
			if err != nil && !eof {
				r.err = connectorError(err)
				return
			}
			if len(users) == 0 {
				if !eof {
					r.err = fmt.Errorf("%s returned zero users but did not return io.EOF", r.connector)
					return
				}
				if cursor != "" {
					r.err = fmt.Errorf("%s returned zero users but returned a non-empty next value", r.connector)
				}
				return
			}

			// previous is the previous read record.
			var previous Record

			for _, user := range users {

				select {
				case <-ctx.Done():
					r.err = ctx.Err()
					return
				default:
				}

				if user.ID == "" {
					r.err = fmt.Errorf("%s returned a record with an empty ID", r.connector)
					return
				}
				// Skip the record if its ID has already been processed.
				if _, ok := processedIDs[user.ID]; ok {
					continue
				}
				processedIDs[user.ID] = struct{}{}

				record := Record{
					ID:             user.ID,
					LastChangeTime: user.LastChangeTime.UTC().Truncate(time.Microsecond),
					Associations:   user.Associations,
				}

				// Validate the last change time.
				if err = validateLastChangeTime(record.LastChangeTime); err != nil {
					record.Err = errors.New("record's last change time is before 1900 or in the future")
				}
				if !r.lastChangeTime.IsZero() && record.LastChangeTime.Before(r.lastChangeTime) {
					r.err = fmt.Errorf("%s returned a record with a last change %s earlier than the required minimum",
						r.connector, r.lastChangeTime.Sub(record.LastChangeTime))
					return
				}

				if record.Err == nil {
					// Read the properties.
					record.Properties = make(map[string]any, len(properties))
					for _, p := range properties {
						v, ok := user.Properties[p.Name]
						if !ok {
							if !p.ReadOptional {
								record.Err = inputValidationErrorf(p.Name, "(returned by %s connector) does not have a value, but the property is not optional for reading", r.connector)
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
				}

				if previous.ID != "" {
					if !yield(previous) {
						return
					}
				}
				previous = record

			}

			if previous.ID != "" {
				r.last = true
				if !yield(previous) {
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
	schemas [2]types.Type
}
