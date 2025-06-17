//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package meergo

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"reflect"
	"time"

	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/types"
)

// ErrEventTypeNotExist is returned by the EventSender.EventTypeSchema method if
// the event type does not exist.
var ErrEventTypeNotExist = errors.New("event type does not exist")

// SendingMode represents the mode of event sending.
type SendingMode int

const (
	None SendingMode = iota
	Cloud
	Device
	Combined
)

// Targets represents the targets.
type Targets int

const (
	TargetEvent Targets = 1 << iota
	TargetUser
	// TargetGroup // TOODO(marco) Implement groups
)

// AppInfo represents an app connector info.
type AppInfo struct {
	Name            string
	Categories      Categories // bitmask of connector's categories.
	AsSource        *AsAppSource
	AsDestination   *AsAppDestination
	Terms           AppTerms
	IdentityIDLabel string
	OAuth           OAuth         // OAuth 2.0 configuration. If the URL is empty the connector does not support OAuth 2.0.
	BackoffPolicy   BackoffPolicy // backoff policy. It controls retry timing using provided strategies or custom ones.
	TimeLayouts     TimeLayouts   // layouts for time values. If left empty, it is ISO 8601.
	Icon            string        // icon in SVG format.

	newFunc reflect.Value
	ct      reflect.Type
}

// AppTerms represents the terms that an app connector uses to refer to users.
type AppTerms struct {
	User  string
	Users string
	// Group  string TODO(marco) Implement groups
	// Groups string
}

// AsAppSource represents the specific information of an app connector used as a
// source.
type AsAppSource struct {
	Targets       Targets
	HasSettings   bool
	Documentation ConnectorRoleDocumentation
}

// AsAppDestination represents the specific information of an app connector used
// as a destination.
type AsAppDestination struct {
	Targets       Targets
	HasSettings   bool
	SendingMode   SendingMode // mode of event sending. 'None' for sources and non-supporting event apps.
	Documentation ConnectorRoleDocumentation
}

// OAuth represents the OAuth 2.0 connector information.
type OAuth struct {
	// AuthURL is the authorization endpoint. It's the URL of the app where
	// users are redirected to grant consent.
	AuthURL string

	// TokenURL is the token endpoint. It's the URL to retrieve the access token,
	// refresh token, and lifetime of the access token.
	TokenURL string

	// SourceScopes specifies the required scopes when used as a source.
	SourceScopes []string

	// DestinationScopes specifies the required scopes when used as a destination.
	DestinationScopes []string

	// ExpiresIn represents the lifetime of the access token in seconds.
	// If the value is zero or negative, the lifetime is provided by the TokenURL endpoint.
	ExpiresIn int32
}

// ReflectType returns the type of the value implementing the app connector
// info.
func (app AppInfo) ReflectType() reflect.Type {
	return app.ct
}

// New returns a new app connector instance.
func (app AppInfo) New(conf *AppConfig) (any, error) {
	out := app.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface()
	err, _ := out[1].Interface().(error)
	return c, err
}

// AppConfig represents the configuration of an app connector.
type AppConfig struct {
	Settings     []byte
	SetSettings  SetSettingsFunc
	OAuthAccount string
	HTTPClient   HTTPClient
}

// AppNewFunc represents functions that create new app connector instances.
type AppNewFunc[T any] func(*AppConfig) (T, error)

// EventType represents a type of event that can be sent to an app.
type EventType struct {
	// ID is the identifier of the event type. It must be unique for every event
	// type of the connection.
	//
	// It cannot be longer than 100 runes.
	ID string

	// Name is the name of the event type to be displayed.
	Name string

	// Description is the description of the event type to be displayed.
	Description string
}

// RecordFetcher is implemented by app connectors that support fetching records.
type RecordFetcher interface {

	// RecordSchema returns the schema of the specified target in the specified
	// role. Role can be Source or Destination, and it returns their respective
	// schemas.
	RecordSchema(ctx context.Context, target Targets, role Role) (types.Type, error)

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
	Records(ctx context.Context, target Targets, lastChangeTime time.Time, ids, properties []string, cursor string, schema types.Type) ([]Record, string, error)
}

// RecordUpserter is implemented by app connectors that support updating and
// creating records.
type RecordUpserter interface {
	RecordFetcher

	// Upsert updates or creates records in the app for the specified target.
	Upsert(ctx context.Context, target Targets, records Records) error
}

// Record represents an app record.
type Record struct {
	ID         string         // Identifier.
	Properties map[string]any // Properties.

	// LastChangeTime is the record's last change time, whose location can be
	// anything, not necessarily UTC. The precision of this time is limited to
	// microseconds; any precision beyond microseconds will be truncated.
	LastChangeTime time.Time

	// TODO(marco): implements groups
	//// Associations contains the identifiers of the user's groups or the group's users.
	//// It is not significant if it is nil.
	//Associations []string

	// Err reports an error that occurred while reading the record.
	// If Err is not nil, only the ID field is significant.
	Err error
}

// RecordsError is returned by the Upsert method of an app connector when only
// some records have failed or when the method can distinguish errors based on
// individual records. It maps record indices to their respective errors.
type RecordsError map[int]error

func (err RecordsError) Error() string {
	var msg string
	for i, e := range err {
		msg += fmt.Sprintf("record %d: %v\n", i, e)
	}
	return msg
}

// Records provides access to a non-empty sequence of records to be created or
// updated by Upsert. A record to be created has an empty ID.
//
// To iterate over records, call either All, Same, or First — only one of these
// can be used per Records value:
//   - All returns an iterator over all records.
//   - Same returns an iterator over records with the same operation type (create
//     or update) as the first record.
//   - First returns the first record.
//
// Records are consumed as they are yielded by the iterator. A record is
// considered consumed once produced by the iterator, unless Skip is called.
//
// Example:
//
//	for i, rec := range records.All() {
//	    // rec is now consumed unless Skip is called here
//	    if i > 0 && !shouldProcess(rec) {
//	        records.Skip()
//	        continue
//	    }
//	    process(rec)
//	}
//
// Calling Skip during iteration marks the current record as not consumed,
// so it will be available in subsequent Upsert calls.
//
// Only one iteration (using All or Same) or call to First may be active on a
// Records value. After an iteration completes or First is called, the Records
// value must not be used again.
type Records interface {

	// All returns an iterator to read all records. Properties of the records in the
	// sequence may be modified unless the record is subsequently skipped.
	All() iter.Seq2[int, Record]

	// First returns the first record. The record's properties may be modified.
	// First può essere chiamato al posto di All e Some se l'app consente di aggiornare o creare un solo record alla volta.
	First() Record

	// Peek retrieves the next record without advancing the iterator. It returns the
	// record and true if a record is available, or false if there are no further
	// records. Can only be called during an iteration with All or Same.
	// The returned record must not be modified.
	Peek() (Record, bool)

	// Same returns an iterator for records: either all records to update
	// (if the first record is for update) or all records to create
	// (if the first record is for creation). Properties of the records in the
	// sequence may be modified unless the record is subsequently skipped.
	Same() iter.Seq2[int, Record]

	// Skip skips the current record in the iteration and marks it as unread. The
	// subsequent iteration will resume at the next record while preserving the same
	// index. Skip may only be called during iterations from All or Same, and only
	// if the record's properties have not been modified.
	//
	// The first event must always be consumed. Calling Skip on it will cause a
	// panic. It is safe to call Skip multiple times on the same record.
	Skip()
}

// EventSender is implemented by app connectors that support event sending.
type EventSender interface {

	// EventTypeSchema returns the schema of the specified event type.
	//
	// The returned schema describes properties required by the connector to
	// send an event of this type. Actions based on the specified event type
	// will have a transformation that, given the received event, provides the
	// properties required by the connector. These properties, along with the
	// raw event, are passed to the connector's PreviewSendEvents and SendEvents
	// methods.
	//
	// If no extra information is needed for the event type, the returned schema
	// is the invalid schema. If the event type does not exist, it returns the
	// ErrEventTypeNotExist error.
	EventTypeSchema(ctx context.Context, eventType string) (types.Type, error)

	// EventTypes returns the event types of the connector's instance.
	EventTypes(ctx context.Context) ([]*EventType, error)

	// PreviewSendEvents builds and returns the HTTP request that would be used to
	// send the given events to the app, without actually sending it.
	//
	// Authentication data in the returned request is redacted (i.e., replaced with
	// "[REDACTED]").
	PreviewSendEvents(ctx context.Context, events Events) (*http.Request, error)

	// SendEvents sends events to an app. events is a non-empty sequence of
	// events to send.
	//
	// This method is safe for concurrent use, on the same instance, by multiple
	// goroutines. If an event type does not exist, it returns the
	// ErrEventTypeNotExist error.
	SendEvents(ctx context.Context, events Events) error
}

// Event represents an event that will be sent to an app.
type Event struct {
	ID         string         // identifier for the event. Guaranteed to be unique per event within the same connection.
	Type       string         // event type (e.g., "user.signup", "order.placed").
	Schema     types.Type     // schema of the event type; may be the invalid schema.
	Properties map[string]any // event data after transformation based on the schema; empty if no transformation exists.
	Raw        RawEvent       // original, untransformed event data as it was received.
}

// EventsError is returned by the SendEvents method of an app connector when
// some events have been consumed, but they cannot be included in the current or
// any future requests. It maps the event indices to the errors related to those
// events.
//
// The EventsRequest method may return both a request with the events that were
// included in the request and an EventsError error with the events that could
// not be included in any request.
type EventsError map[int]error

func (err EventsError) Error() string {
	var msg string
	for i, e := range err {
		msg += fmt.Sprintf("event %d: %v\n", i, e)
	}
	return msg
}

// Events represents a collection of events to be sent to an app. The collection
// is guaranteed to contain at least one event.
//
// After calling First or once the iterator returned by All or SameUser stops,
// no further method calls on Events are allowed.
type Events interface {

	// All returns an iterator to read all events. Properties of the events in the
	// sequence may be modified unless the event is subsequently skipped.
	All() iter.Seq2[int, *Event]

	// First returns the first event. The event's properties may be modified.
	// After First is called, no further method calls on Events are allowed.
	First() *Event

	// Peek retrieves the next event without advancing the iterator. It returns the
	// event and true if an event is available, or false if there are no further
	// events. The returned event must not be modified.
	Peek() (*Event, bool)

	// SameUser returns an iterator over the events of the same user. Properties of
	// the events in the sequence may be modified unless the event is subsequently
	// skipped.
	SameUser() iter.Seq2[int, *Event]

	// Skip skips the current event in the iteration and marks it as unread. The
	// subsequent iteration will resume at the next event while preserving the same
	// index. Skip may only be called during iterations from All or SameUser, and only
	// if the event's properties have not been modified.
	Skip()
}

// RawEvent represents a raw event as received from a source connector.
type RawEvent interface {
	AnonymousId() string
	Channel() string
	Category() string
	Context() RawEventContext
	Event() string
	GroupId() string
	MessageId() string
	Name() string
	ReceivedAt() time.Time
	SentAt() time.Time
	Timestamp() time.Time
	Type() string
	UserId() string
}

type RawEventContext interface {
	App() (RawEventContextApp, bool)
	Browser() (RawEventContextBrowser, bool)
	Campaign() (RawEventContextCampaign, bool)
	Device() (RawEventContextDevice, bool)
	IP() string
	Library() (RawEventContextLibrary, bool)
	Locale() string
	Location() (RawEventContextLocation, bool)
	Network() (RawEventContextNetwork, bool)
	OS() (RawEventContextOS, bool)
	Page() (RawEventContextPage, bool)
	Referrer() (RawEventContextReferrer, bool)
	Screen() (RawEventContextScreen, bool)
	Session() (RawEventContextSession, bool)
	Timezone() string
	UserAgent() string
}

type RawEventContextApp interface {
	Name() string
	Version() string
	Build() string
	Namespace() string
}

type RawEventContextBrowser interface {
	Name() string
	Other() string
	Version() string
}

type RawEventContextCampaign interface {
	Name() string
	Source() string
	Medium() string
	Term() string
	Content() string
}

type RawEventContextDevice interface {
	Id() string
	AdvertisingId() string
	AdTrackingEnabled() bool
	Manufacturer() string
	Model() string
	Name() string
	Type() string
	Token() string
}

type RawEventContextLibrary interface {
	Name() string
	Version() string
}
type RawEventContextLocation interface {
	City() string
	Country() string
	Latitude() float64
	Longitude() float64
	Speed() float64
}
type RawEventContextNetwork interface {
	Bluetooth() bool
	Carrier() string
	Cellular() bool
	WiFi() bool
}
type RawEventContextOS interface {
	Name() string
	Version() string
}

type RawEventContextPage interface {
	Path() string
	Referrer() string
	Search() string
	Title() string
	URL() string
}

type RawEventContextReferrer interface {
	Id() string
	Type() string
}

type RawEventContextScreen interface {
	Width() int
	Height() int
	Density() decimal.Decimal
}

type RawEventContextSession interface {
	Id() int
	Start() bool
}
