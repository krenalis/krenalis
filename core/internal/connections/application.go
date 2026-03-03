// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

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

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/internal/connections/appwriter"
	"github.com/meergo/meergo/core/internal/connections/httpclient"
	"github.com/meergo/meergo/core/internal/filters"
	"github.com/meergo/meergo/core/internal/schemas"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/util"
	"github.com/meergo/meergo/tools/decimal"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
)

// InvalidEventError is returned by the PreviewSendEvent method when an event
// is invalid.
type InvalidEventError struct {
	Err error
}

func (err *InvalidEventError) Error() string {
	return err.Err.Error()
}

// Application represents the application of an application connection.
type Application struct {
	id          int
	connector   string
	role        state.Role
	timeLayouts *state.TimeLayouts
	httpClient  *httpclient.Client
	users       schema
	targets     state.ConnectorTargets
	inner       any
	err         error
}

// TODO(marco): implement webhooks
//type webhookConnection interface {
//	// ReceiveWebhook receives a webhook request and returns its payloads. If
//	// webhooks are per connection, role is the connection's role, otherwise is
//	// Both. It returns the ErrWebhookUnauthorized error is the request was not
//	// authorized. The context is the request's context.
//	ReceiveWebhook(r *http.Request, role connectors.Role) ([]connectors.WebhookPayload, error)
//}

type applicationOAuthConnector interface {
	// OAuthAccount returns the application's account associated with the OAuth authorization.
	OAuthAccount(ctx context.Context) (string, error)
}

// Application returns the application for the provided connection.
// Errors are deferred until a method of the application is called.
// It panics if the connection is not an application connection.
func (c *Connections) Application(connection *state.Connection) *Application {
	connector := connection.Connector()
	var targets state.ConnectorTargets
	if connection.Role == state.Source {
		targets = connector.SourceTargets
	} else {
		targets = connector.DestinationTargets
	}
	app := &Application{
		id:          connection.ID,
		connector:   connector.Code,
		role:        connection.Role,
		timeLayouts: &connector.TimeLayouts,
		httpClient:  c.http.ConnectionClient(connection),
		users:       schema{lock: make(chan struct{}, 1)},
		targets:     targets,
	}
	// var accountID int // TODO(marco): implement webhooks
	var accountCode string
	if a, ok := connection.Account(); ok {
		// accountID = a.ID // TODO(marco): implement webhooks
		accountCode = a.Code
	}
	app.inner, app.err = connectors.RegisteredApplication(app.connector).New(&connectors.ApplicationEnv{
		Settings:     connection.Settings,
		SetSettings:  setConnectionSettingsFunc(c.state, connection),
		OAuthAccount: accountCode,
		HTTPClient:   app.httpClient,
		// WebhookURL:   webhookURL(connection, accountID), // TODO(marco): implement webhooks
	})
	app.err = connectorError(app.err)
	return app
}

// ID returns the ID of the application connection.
func (app *Application) ID() int {
	return app.id
}

// Connector returns the name of the application connector.
func (app *Application) Connector() string {
	return app.connector
}

// EventTypes returns the application's event types.
// If the connector returns an error, it returns an *UnavailableError error.
// It panics if the application does not support the event target.
func (app *Application) EventTypes(ctx context.Context) ([]*EventType, error) {
	if app.err != nil {
		return nil, app.err
	}
	eventTypes, err := app.inner.(connectors.EventSender).EventTypes(ctx)
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

// PreviewSendEvent returns the request that would be used to send events to
// the application. If no error is returned, the request is non-nil.
//
// It validates the event schema, which must align with the schema of the event
// type, then passes that event type schema to the connector.
//
// If the event type does not exist, it returns the
// connectors.ErrEventTypeNotExist error. If the schema of the event is not
// aligned to the event type's schema, it returns a *schemas.Error error. If the
// event is invalid, it returns a *InvalidEventError error. If the connector
// returns an error, it returns a *UnavailableError error.
//
// It panics if the application does not support the event target, or if schema is valid
// but not an object.
func (app *Application) PreviewSendEvent(ctx context.Context, event connectors.Event) (*http.Request, error) {
	if app.err != nil {
		return nil, app.err
	}
	eventTypeSchema, err := app.inner.(connectors.EventSender).EventTypeSchema(ctx, event.Type.ID)
	if err != nil {
		return nil, connectorError(err)
	}
	// Check that schema is aligned with the event type's schema.
	err = schemas.CheckAlignment(event.Type.Schema, eventTypeSchema, new(state.CreateOnly))
	if err != nil {
		return nil, err
	}
	// Pass the event type's schema to the connector.
	event.Type.Schema = eventTypeSchema
	// Return the request that represents the event preview.
	iterator := newSingleEventIterator(&event, app.connector)
	req, err := app.inner.(connectors.EventSender).PreviewSendEvents(ctx, iterator)
	if err != nil {
		return nil, connectorError(err)
	}
	if err = iterator.Err(); err != nil {
		return nil, &InvalidEventError{Err: err}
	}
	if req == nil {
		return nil, fmt.Errorf("%s.PreviewSendEvents returned nil without discarding events", app.connector)
	}
	return req, nil
}

// Schema returns the application's schema for the provided target.
// If target is state.TargetEvent, eventType represents the type of the event.
//
// If the target is state.TargetEvent and the event type refers to an
// application event for which no schema is expected, this method returns the
// invalid type and no errors.
//
// For the users and the groups target, the returned schema contains only the
// properties compatible with the application's role. For the event target, the
// returned schema can be the invalid schema.
//
// If the event type does not exist, it returns the
// connectors.ErrEventTypeNotExist error. If the connector returns an error, it
// returns a *UnavailableError.
// It panics if the application does not support the provided target.
func (app *Application) Schema(ctx context.Context, target state.Target, eventType string) (types.Type, error) {
	return app.SchemaAsRole(ctx, app.role, target, eventType)
}

// SchemaAsRole is like Schema but returns the schema as the provided role,
// instead of the role of the application's connection.
//
// If the target is state.TargetEvent and the event type refers to an
// application event for which no schema is expected, this method returns an
// invalid type with no error.
//
// If the event type does not exist, it returns the
// connectors.ErrEventTypeNotExist error. If the connector returns an error, it
// returns a *UnavailableError.
// It panics if role is not Source or Destination.
func (app *Application) SchemaAsRole(ctx context.Context, role state.Role, target state.Target, eventType string) (types.Type, error) {
	if app.err != nil {
		return types.Type{}, app.err
	}
	if role != state.Source && role != state.Destination {
		panic("invalid role")
	}
	switch target {
	case state.TargetEvent:
		if role != state.Destination {
			panic("invalid role")
		}
		schema, err := app.inner.(connectors.EventSender).EventTypeSchema(ctx, eventType)
		if err != nil {
			return types.Type{}, connectorError(err)
		}
		if !schema.Valid() {
			return types.Type{}, nil
		}
		return types.AsRole(schema, types.Destination), nil
	case state.TargetUser:
		schema, err := app.userSchema(ctx, role)
		return schema, connectorError(err)
		// TODO(marco): Implement groups
		//case state.Groups:
		//	schema, err := app.inner.(applicationSchemaConnector).Schema(ctx, connectors.GroupTarget, connectors.Role(role), "")
		//	if err != nil {
		//		return types.Type{}, connectorError(err)
		//	}
		//	if !schema.Valid() {
		//		return types.Type{}, fmt.Errorf("connector %s returned an invalid group schema", app.connector)
		//	}
	}
	panic("unexpected target")
}

// SendEvents sends events to an application. events must be a non-empty
// sequence of events to send.
//
// If an event type does not exist, it returns the ErrEventTypeNotExist error.
//
// It panics if the application does not support the event target.
func (app *Application) SendEvents(ctx context.Context, events connectors.Events) error {
	if app.err != nil {
		return app.err
	}
	err := app.inner.(connectors.EventSender).SendEvents(ctx, events)
	return connectorError(err)
}

// Users returns an iterator to iterate over the application's users. Each
// returned record will contain, in the Attributes field, the properties in
// schema, with the same types. If where is not nil, only users matching its
// conditions will be returned.
//
// If updatedAt is not the zero time, it must be in UTC, and its year cannot be
// before 1900. In this case, only records changed or created at or after that
// time will be returned, with a precision limited to microseconds.
//
// If the connector returns an error, it returns an *UnavailableError error. If
// the provided schema, that must be valid, does not align with the
// application's source schema, it returns a *schemas.Error error.
//
// The Err method of the returned iterator may return an *UnavailableError if
// the connector encounters an error.
func (app *Application) Users(ctx context.Context, schema types.Type, where *state.Where, updatedAt time.Time) (Records, error) {
	if app.err != nil {
		return nil, app.err
	}
	if !schema.Valid() {
		return nil, fmt.Errorf("schema is not valid")
	}
	// Check that the user schema is aligned with the application's user schema.
	appSchema, err := app.userSchema(ctx, state.Source)
	if err != nil {
		return nil, err
	}
	err = schemas.CheckAlignment(schema, appSchema, nil)
	if err != nil {
		return nil, err
	}
	properties := schema.Properties()
	appSchema = types.Prune(appSchema, func(path string) bool {
		return properties.ContainsPath(path)
	})
	if !updatedAt.IsZero() {
		if updatedAt.Location() != time.UTC {
			return nil, fmt.Errorf("updatedAt is not UTC")
		}
		if updatedAt.Year() < 1900 {
			return nil, fmt.Errorf("updatedAt's year is before 1900")
		}
		updatedAt = updatedAt.Truncate(time.Microsecond)
	}
	records := &appRecords{
		schema:      schema,
		where:       where,
		appSchema:   appSchema,
		timeLayouts: app.timeLayouts,
		updatedAt:   updatedAt,
		connector:   app.connector,
		inner:       app.inner,
	}
	return records, nil
}

// WaitTime returns an estimate of how long to wait before sending an HTTP
// request to the client, helping to avoid being queued.
// pattern is the pattern of the rate limit.
func (app *Application) WaitTime(pattern string) (time.Duration, error) {
	return app.httpClient.WaitTime(pattern)
}

// Writer returns a Writer for creating and updating users or groups in the
// application. outSchema is the output schema of the pipeline, exportMode is
// the export mode, and target is the target of the pipeline. ack is the
// function that will receive the acknowledgments and cannot be nil.
//
// If the pipeline's output schema does not align with the application's
// destination schema, it returns a *schemas.Error indicating the mismatch.
func (app *Application) Writer(ctx context.Context, outSchema types.Type, exportMode state.ExportMode, target state.Target, ack AckFunc) (Writer, error) {
	if app.err != nil {
		return nil, app.err
	}
	if ack == nil {
		return nil, errors.New("ack function is missing")
	}
	// Get the destination schema.
	destinationSchema, err := app.SchemaAsRole(ctx, state.Destination, state.TargetUser, "")
	if err != nil {
		return nil, err
	}
	// Check that the output schema is aligned with the destination schema.
	err = schemas.CheckAlignment(outSchema, destinationSchema, &exportMode)
	if err != nil {
		return nil, err
	}
	inner := app.inner.(connectors.RecordUpserter)
	writer := appwriter.New(app.connector, target, inner.Upsert, outSchema, appwriter.AcksFunc(ack))
	return writer, nil
}

// userSchema returns the user schema with the provided role.
// If the connector returns an error, it returns an *UnavailableError error.
// It panics if role is not Source or Destination.
func (app *Application) userSchema(ctx context.Context, role state.Role) (types.Type, error) {
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
	schema, err := app.inner.(connectors.RecordFetcher).RecordSchema(ctx, connectors.TargetUser, connectors.Role(role))
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

// singleEventIterator implements the connectors.Events interface that iterates
// over a single event.
type singleEventIterator struct {
	app       string
	event     *connectors.Event
	consumed  bool
	iterating bool
	postponed bool
	discarded bool
	err       error
}

// newSingleEventIterator returns a singleEventIterator with the provided event.
// app is the name of the application connector.
func newSingleEventIterator(event *connectors.Event, app string) *singleEventIterator {
	return &singleEventIterator{app: app, event: event}
}

func (iter *singleEventIterator) All() iter.Seq[*connectors.Event] {
	if iter.consumed {
		panic(iter.app + " connector: SendEvents method called Events.All after the events were consumed")
	}
	iter.consumed = true
	return func(yield func(event *connectors.Event) bool) {
		iter.iterating = true
		yield(iter.event)
	}
}

func (iter *singleEventIterator) Discard(err error) {
	if !iter.iterating {
		panic(iter.app + " connector: SendEvents method called Events.Discard outside an iteration")
	}
	if iter.postponed {
		panic(iter.app + " connector: SendEvents method called Events.Discard on a postponed event")
	}
	if iter.discarded {
		panic(iter.app + " connector: SendEvents method called Events.Discard on a discarded event")
	}
	if err == nil {
		panic(iter.app + " connector: SendEvents method called Events.Discard passing a nil error")
	}
	iter.discarded = true
	iter.err = err
}

// Err returns the error passed to the Discard method if the event has been
// discarded.
func (iter *singleEventIterator) Err() error {
	return iter.err
}

func (iter *singleEventIterator) First() *connectors.Event {
	if iter.consumed {
		panic(iter.app + " connector: SendEvents method called Events.First after the events were consumed")
	}
	iter.consumed = true
	return iter.event
}

func (iter *singleEventIterator) Peek() (*connectors.Event, bool) {
	if iter.consumed && !iter.iterating {
		panic(iter.app + " connector: SendEvents method called Events.Peek outside of an iteration")
	}
	if iter.consumed {
		return nil, false
	}
	return iter.event, true
}

func (iter *singleEventIterator) Postpone() {
	if !iter.iterating {
		panic(iter.app + " connector: SendEvents method called Events.Postpone outside an iteration")
	}
	if iter.postponed {
		return
	}
	if iter.discarded {
		panic(iter.app + " connector: SendEvents method called Events.Postpone on a discarded event")
	}
	iter.postponed = true
}

func (iter *singleEventIterator) SameUser() iter.Seq[*connectors.Event] {
	if iter.consumed {
		panic(iter.app + " connector: SendEvents method called Events.Some after the events were consumed")
	}
	iter.consumed = true
	return func(yield func(event *connectors.Event) bool) {
		iter.iterating = true
		yield(iter.event)
	}
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
		for _, p := range t.Properties().All() {
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

// appRecords implements the Records interface for applications.
type appRecords struct {
	schema      types.Type
	where       *state.Where
	appSchema   types.Type
	timeLayouts *state.TimeLayouts
	updatedAt   time.Time
	connector   string
	inner       any
	last        bool
	err         error
	closed      bool
}

func (r *appRecords) All(ctx context.Context) iter.Seq[Record] {

	return func(yield func(Record) bool) {

		if r.closed {
			r.err = errors.New("connectors: For called on a closed Records")
			return
		}

		var cursor string

		properties := r.schema.Properties()

		// processedIDs contains the already read ID.
		// It is used to deduplicate returned records.
		processedIDs := map[string]any{}

		for {

			// Retrieve the users.
			var users []connectors.Record
			var err error
			users, cursor, err = r.inner.(connectors.RecordFetcher).Records(ctx, connectors.TargetUser, r.updatedAt, cursor, r.appSchema)
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

				if err := ctx.Err(); err != nil {
					r.err = err
					return
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
					ID:        user.ID,
					UpdatedAt: user.UpdatedAt.UTC().Truncate(time.Microsecond),
					// Associations:   user.Associations, TODO(marco): Implement groups
				}

				// Validate the update time.
				if err = validateUpdatedAt(record.UpdatedAt); err != nil {
					record.Err = errors.New("record's update time is before 1900 or in the future")
				}
				if !r.updatedAt.IsZero() && record.UpdatedAt.Before(r.updatedAt) {
					r.err = fmt.Errorf("%s returned a record whose update time is earlier than the required minimum", r.connector)
					return
				}

				if record.Err == nil {
					// Read the attributes.
					record.Attributes = make(map[string]any, properties.Len())
					for _, p := range properties.All() {
						v, ok := user.Attributes[p.Name]
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
						record.Attributes[p.Name] = v
					}
					if record.Err == nil && r.where != nil {
						if !filters.Applies(r.where, record.Attributes) {
							continue
						}
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
