// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package connectors

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/meergo/meergo/tools/decimal"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
)

// ErrEventTypeNotExist is returned by the EventSender.EventTypeSchema method if
// the event type does not exist.
var ErrEventTypeNotExist = errors.New("event type does not exist")

// SendingMode represents the mode of event sending.
type SendingMode int

const (
	None SendingMode = iota
	Client
	Server
	ClientAndServer
)

// Targets represents the targets.
type Targets int

const (
	TargetEvent Targets = 1 << iota
	TargetUser
	// TargetGroup // TOODO(marco) Implement groups
)

// ApplicationSpec represents an application connector specification.
type ApplicationSpec struct {
	Code           string
	Label          string
	Categories     Categories // bitmask of connector's categories.
	AsSource       *AsApplicationSource
	AsDestination  *AsApplicationDestination
	Terms          ApplicationTerms
	OAuth          OAuth           // OAuth 2.0 configuration. If the URL is empty the connector does not support OAuth 2.0.
	EndpointGroups []EndpointGroup // rate limiting and retry policies per endpoint group.
	TimeLayouts    TimeLayouts     // layouts for time values. If left empty, it is ISO 8601.

	newFunc reflect.Value
	ct      reflect.Type
}

// ApplicationTerms represents the terms that an application connector uses to
// refer to users.
type ApplicationTerms struct {
	User   string
	Users  string
	UserID string
	// Group  string TODO(marco) Implement groups
	// Groups string
}

// AsApplicationSource represents the specific information of an application
// connector used as a source.
type AsApplicationSource struct {
	Targets       Targets
	HasSettings   bool
	Documentation RoleDocumentation
}

// AsApplicationDestination represents the specific information of an
// application connector used as a destination.
type AsApplicationDestination struct {
	Targets       Targets
	HasSettings   bool
	SendingMode   SendingMode // mode of event sending. 'None' for sources and non-supporting event applications.
	Documentation RoleDocumentation
}

// OAuth represents the OAuth 2.0 connector information.
type OAuth struct {
	// AuthURL is the authorization endpoint. It's the URL of the application
	// where users are redirected to grant consent.
	AuthURL string

	// TokenURL is the token endpoint. It's the URL to retrieve the access token,
	// refresh token, and lifetime of the access token.
	TokenURL string

	// SourceScopes specifies the required scopes when used as a source.
	SourceScopes []string

	// DestinationScopes specifies the required scopes when used as a destination.
	DestinationScopes []string

	// ExpiresIn represents the lifetime of the access token in seconds.
	// If the value is zero or negative, the lifetime is provided by the TokenURL
	// endpoint.
	ExpiresIn int32

	// Disallow127_0_0_1 forbids using "127.0.0.1" as the host in the redirect URL.
	Disallow127_0_0_1 bool

	// DisallowLocalhost forbids using "localhost" as the host in the redirect URL.
	DisallowLocalhost bool
}

// ReflectType returns the type of the value implementing the application
// connector specification.
func (app ApplicationSpec) ReflectType() reflect.Type {
	return app.ct
}

// New returns a new application connector instance.
func (app ApplicationSpec) New(env *ApplicationEnv) (any, error) {
	out := app.newFunc.Call([]reflect.Value{reflect.ValueOf(env)})
	c := out[0].Interface()
	err, _ := reflect.TypeAssert[error](out[1])
	return c, err
}

// ApplicationEnv is the environment for an application connector.
type ApplicationEnv struct {

	// Settings holds the raw settings data.
	Settings json.Value

	// SetSettings is the function used to update the settings.
	SetSettings SetSettingsFunc

	// OAuthAccount is the OAuth account identifier for authentication.
	OAuthAccount string

	// HTTPClient is the HTTP client to use for all requests.
	HTTPClient HTTPClient
}

// ApplicationNewFunc represents functions that create new application connector
// instances.
type ApplicationNewFunc[T any] func(*ApplicationEnv) (T, error)

// EndpointGroup defines a group of application endpoints—specified by one or
// more patterns using the same syntax as http.ServeMux. All HTTP requests
// matching any of the given patterns will be subject to the specified rate
// limiting and retry policies.
type EndpointGroup struct {
	Patterns     []string    // patterns (e.g. "GET /api/users/", "api.example.com/v1/items") matched by ServeMux-style matching
	RequireOAuth bool        // require OAuth authentication
	RateLimit    RateLimit   // rate limiting configuration applied to all matching requests
	RetryPolicy  RetryPolicy // retry policy for handling failed requests to these endpoints
}

// RateLimit defines the maximum requests per second, burst capacity, and the
// limit on simultaneous in-flight requests.
type RateLimit struct {

	// RequestsPerSecond specifies the maximum number of requests allowed per
	// second. Must be > 0.
	RequestsPerSecond float64

	// Burst specifies how many requests can be sent in a burst without waiting.
	// When requests are sent below RequestsPerSecond, the unused quota accumulates
	// up to Burst. This means, after a period of inactivity or low usage, you can
	// send a burst of up to Burst requests instantly, temporarily exceeding the
	// per-second limit. Must be > 0.
	Burst int

	// MaxConcurrentRequests limits the maximum number of concurrent (simultaneous)
	// in-flight requests. Must be >= 0. If set to 0, there is no limit on the
	// number of concurrent requests.
	MaxConcurrentRequests int
}

// RetryPolicy defines a mapping between HTTP status codes and their
// corresponding backoff strategies. Each key in the map is a string containing
// one or more space-separated HTTP status codes. The associated value is the
// strategy to apply when an HTTP response returns one of the specified status
// codes.
//
// For example:
//
//	RetryPolicy{
//	    "429":     connectors.RetryAfterStrategy(),
//	    "500 503": connectors.ExponentialStrategy(connectors.NetFailure, time.Second),
//	}
type RetryPolicy map[string]RetryStrategy

// RetryStrategy represents a strategy for determining retry behavior.
// It returns a FailureReason and the duration to wait before the next attempt,
// based on the HTTP response from the previous attempt and the number of
// retries made. retries parameter starts at 0 before the first retry and
// increments by 1 on each retry.
//
// If the returned waitTime is negative, it is considered zero.
type RetryStrategy func(res *http.Response, retries int) (reason FailureReason, waitTime time.Duration)

// FailureReason defines how the client should retry after a failed HTTP
// request.
type FailureReason int

const (
	// PermanentFailure indicates a permanent failure that cannot be retried.
	PermanentFailure FailureReason = iota
	// NetFailure indicates a network failure.
	NetFailure
	// Unauthorized indicates an unauthorized request.
	Unauthorized
	// Slowdown indicates that the client should slow down its request rate.
	Slowdown
	// RateLimited indicates that the request was rejected due to rate limiting.
	RateLimited
)

func (r FailureReason) String() string {
	switch r {
	case PermanentFailure:
		return "PermanentFailure"
	case NetFailure:
		return "NetFailure"
	case Unauthorized:
		return "Unauthorized"
	case Slowdown:
		return "Slowdown"
	case RateLimited:
		return "RateLimited"
	}
	panic(fmt.Errorf("unexpected FailureReason %d", r))
}

// EventType represents a type of event that can be sent to an application.
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

	// Filter is the recommended default filter to use for pipelines that use
	// this event type.
	Filter string
}

// RecordFetcher is implemented by application connectors that support fetching
// records.
type RecordFetcher interface {

	// RecordSchema returns the schema of the specified target in the specified
	// role. Role can be Source or Destination, and it returns their respective
	// schemas.
	RecordSchema(ctx context.Context, target Targets, role Role) (types.Type, error)

	// Records returns the records of the specified target. The target can only be
	// either Users or Groups, and it must be a target supported by the connector.
	//
	// If updatedAt is not the zero time, only records created or modified at or
	// after that time are returned. The precision of updatedAt is limited to
	// microseconds.
	//
	// cursor represents the position from which to start reading the records; it is
	// the cursor value returned by the previous call in a paginated query.
	// Subsequent calls will use this cursor value to retrieve the next batch of
	// records.
	//
	// schema is a recent schema from the connector's Schema method, restricted to
	// the properties to return.
	//
	// Records may return duplicate records, i.e., records with the same ID. The
	// caller is responsible for deduplicating them.
	//
	// For each record, Records must return attributes for all properties defined in
	// schema, unless a property is read-optional. Records may also return
	// attributes for properties that were not explicitly requested.
	//
	// The string return value is used as the cursor in the subsequent call. It can
	// be any UTF-8 encoded string, including an empty string. If there are no more
	// records to return, the method returns the last records read (if any) along
	// with the io.EOF error.
	//
	// In case of an error, it returns a non-nil and non-EOF error.
	Records(ctx context.Context, target Targets, updatedAt time.Time, cursor string, schema types.Type) ([]Record, string, error)
}

// RecordUpserter is implemented by application connectors that support updating
// and creating records.
type RecordUpserter interface {
	RecordFetcher

	// Upsert updates or creates records in the application for the specified
	// target.
	//
	// The attributes of each record are compliant with the provided schema,
	// and the schema is compliant with a recent destination schema.
	Upsert(ctx context.Context, target Targets, records Records, schema types.Type) error
}

// Record represents an application record.
type Record struct {
	ID         string         // Identifier.
	Attributes map[string]any // Attributes.

	// UpdatedAt is the record's last update time. Its location may be any time
	// zone and is not necessarily UTC. The precision is limited to microseconds;
	// any finer precision is truncated.
	UpdatedAt time.Time

	// TODO(marco): implements groups
	//// Associations contains the identifiers of the user's groups or the group's users.
	//// It is not significant if it is nil.
	//Associations []string

	// Err reports an error that occurred while reading the record.
	// If Err is not nil, only the ID field is significant.
	Err error
}

// RecordsError is returned by the Upsert method of an application connector
// when only some records have failed or when the method can distinguish errors
// based on individual records. It maps record indices to their respective
// errors.
type RecordsError map[int]error

func (err RecordsError) Error() string {
	var msg strings.Builder
	for i, e := range err {
		msg.WriteString(fmt.Sprintf("record %d: %v\n", i, e))
	}
	return msg.String()
}

// Records provides access to a non-empty sequence of records to be created or
// updated by Upsert. A record to be created has an empty ID.
//
// To iterate over records, call either All, Same, or First — only one of these
// can be used per Records value:
//   - All returns an iterator over all records.
//   - Same returns an iterator over records with the same operation type
//     (create or update) as the first record.
//   - First returns the first record.
//
// Records are consumed as they are yielded by the iterator. A record is
// considered consumed once produced by the iterator, unless Postpone is called.
//
// Example:
//
//	for record := range records.All() {
//		...
//		// record is now consumed unless Postpone is called here
//		if postpone {
//			records.Postpone()
//			continue
//		}
//		...
//	}
//
// Calling Postpone during iteration marks the current record as not consumed,
// so it will be available in subsequent Upsert calls.
//
// Only one iteration (using All or Same) or call to First may be active on a
// Records value. After an iteration completes or First is called, the Records
// value must not be used again.
type Records interface {

	// All returns an iterator to read all records. Attributes of the records in the
	// sequence may be modified unless the record is subsequently postponed.
	All() iter.Seq[Record]

	// Discard discards the current record in the iteration with the provided error.
	// Discard may only be called during iterations from All or Same.
	// It panics if err is nil, or if the record has already been postponed or
	// discarded.
	Discard(err error)

	// First returns the first record. The record's attributes may be modified.
	// Use it instead of All or Some when the application only needs to create or
	// update one record at a time.
	First() Record

	// Peek retrieves the next record without advancing the iterator. It returns the
	// record and true if a record is available, or false if there are no further
	// records. Can only be called during an iteration with All or Same.
	// The returned record must not be modified.
	Peek() (Record, bool)

	// Postpone postpones the current record in the iteration and marks it as
	// unread. Postpone may only be called during iterations from All or Same, and
	// only if the record's attributes have not been modified.
	//
	// The first event must always be consumed. Calling Postpone on it will cause a
	// panic. It is safe to call Postpone multiple times on the same record.
	// A panic occurs if the event has already been discarded.
	Postpone()

	// Same returns an iterator for records: either all records to update
	// (if the first record is for update) or all records to create
	// (if the first record is for creation). Attributes of the records in the
	// sequence may be modified unless the record is subsequently postponed.
	Same() iter.Seq[Record]
}

// EventSender is implemented by application connectors that support event
// sending.
type EventSender interface {

	// EventTypeSchema returns the schema of the specified event type.
	//
	// The returned schema describes values required by the connector to send an
	// event of this type. Actions based on the specified event type will have a
	// transformation that, given the received event, provides the values required
	// by the connector. These values, along with the received event, are passed to
	// the connector's PreviewSendEvents and SendEvents methods.
	//
	// If no extra information is needed for the event type, the returned schema
	// is the invalid schema. If the event type does not exist, it returns the
	// ErrEventTypeNotExist error.
	EventTypeSchema(ctx context.Context, eventType string) (types.Type, error)

	// EventTypes returns the event types of the connector's instance.
	EventTypes(ctx context.Context) ([]*EventType, error)

	// PreviewSendEvents builds and returns the HTTP request that would be used
	// to send the given events to the application, without actually sending it.
	//
	// If no events were sent (for example, because they were all discarded), it
	// returns nil, nil. If any event type does not exist, it returns the
	// ErrEventTypeNotExist error.
	//
	// Authentication data in the returned request is redacted (i.e., replaced
	// with "[REDACTED]"). If the destination pipeline's identifier would appear
	// in an event identifier, it is replaced with "[PIPELINE]".
	//
	// This method is safe for concurrent use, on the same instance, by multiple
	// goroutines.
	PreviewSendEvents(ctx context.Context, events Events) (*http.Request, error)

	// SendEvents sends a non-empty sequence of events to an application.
	//
	// If any event type does not exist, it returns the ErrEventTypeNotExist error.
	//
	// If one or more events fail to be delivered, it returns an EventsError, which
	// includes a key for each failed event along with the corresponding error.
	//
	// If the returned error is not nil and not one of the above cases, it indicates
	// a failure in the request itself that cannot be retried.
	//
	// If all events are delivered successfully, it returns nil.
	//
	// This method is safe for concurrent use, on the same instance, by multiple
	// goroutines.
	SendEvents(ctx context.Context, events Events) error
}

// Event represents an event that will be sent to an application.
type Event struct {
	DestinationPipeline int           // Destination pipeline that processes the event.
	Received            ReceivedEvent // Event as it was received.
	Type                EventTypeInfo // Event type.
}

// EventTypeInfo represents the event type in the context of a specific event
// to send to an application.
type EventTypeInfo struct {
	ID     string         // Identifier.
	Schema types.Type     // Schema; invalid if the type has no properties.
	Values map[string]any // Values; nil if the type has no properties.
}

// EventsError can be returned by the SendEvents and PreviewSendEvents methods
// of an application connector when one or more events are rejected by the
// application due to validation issues—such as schema mismatches, missing
// required fields, or invalid values. It maps the index of each failed event
// (starting from 0) to the corresponding error.
//
// This error type only reports validation-related failures. Other kinds of
// errors (e.g., network issues or internal failures) may be returned
// separately.
//
// For example, if the third event is rejected due to a validation error while
// all other events are accepted, the returned error would be:
//
// EventsError{2: errors.New("event is not valid")}
type EventsError map[int]error

func (err EventsError) Error() string {
	var msg strings.Builder
	for i, e := range err {
		msg.WriteString(fmt.Sprintf("event %d: %v\n", i, e))
	}
	return msg.String()
}

// Events provides access to a non-empty sequence of events to be sent to an
// application.
//
// To iterate over events, call either All, SameUser, or First — only one of
// these can be used per Events value:
//   - All returns an iterator over all events.
//   - SameUser returns an iterator over events with the same user (events with
//     the same anonymous ID) as the first event.
//   - First returns the first event.
//
// Events are consumed as they are yielded by the iterator. An event is
// considered consumed once produced by the iterator, unless Postpone is called.
//
// Example:
//
//	for event := range events.All() {
//		...
//		// event is now consumed unless Postpone is called here
//		if postpone {
//			events.Postpone()
//			continue
//		}
//		...
//	}
//
// Calling Postpone during iteration marks the current event as not consumed, so
// it will be available in subsequent SendEvents or PreviewSendEvents calls.
//
// Only one iteration (using All or SameUser) or call to First may be active on
// an Events value. After an iteration completes or First is called, the Events
// value must not be used again.
type Events interface {

	// All returns an iterator to read all events. Type.Values of the events in the
	// sequence may be modified unless the event is subsequently postponed.
	All() iter.Seq[*Event]

	// Discard discards the current event in the iteration with the provided error.
	// Discard may only be called during iterations from All or SameUser.
	// It panics if err is nil, or if the event has already been postponed or
	// discarded.
	Discard(err error)

	// First returns the first event. The event's Type.Values may be modified.
	// After First is called, no further method calls on Events are allowed.
	First() *Event

	// Peek retrieves the next event without advancing the iterator. It returns the
	// event and true if an event is available, or false if there are no further
	// events. The returned event must not be modified.
	Peek() (*Event, bool)

	// Postpone postpones the current event in the iteration and marks it as unread.
	// Postpone may only be called during iterations from All or SameUser, and only
	// if the event's Type.Values have not been modified.
	// A panic occurs if the event has already been discarded.
	Postpone()

	// SameUser returns an iterator over the events of the same user. Type.Values of
	// the events in the sequence may be modified unless the event is subsequently
	// postponed.
	SameUser() iter.Seq[*Event]
}

// ReceivedEvent represents an event as received from a source connector.
type ReceivedEvent interface {
	AnonymousID() string
	Channel() (string, bool)
	Category() (string, bool)
	Context() (ReceivedEventContext, bool)
	Event() (string, bool)
	GroupID() (string, bool)
	MessageID() string
	Name() (string, bool)
	ReceivedAt() time.Time
	SentAt() time.Time
	Timestamp() time.Time
	Type() string
	PreviousID() (string, bool)
	UserID() (string, bool)
}

type ReceivedEventContext interface {
	App() (ReceivedEventContextApp, bool)
	Browser() (ReceivedEventContextBrowser, bool)
	Campaign() (ReceivedEventContextCampaign, bool)
	Device() (ReceivedEventContextDevice, bool)
	IP() (string, bool)
	Library() (ReceivedEventContextLibrary, bool)
	Locale() (string, bool)
	Location() (ReceivedEventContextLocation, bool)
	Network() (ReceivedEventContextNetwork, bool)
	OS() (ReceivedEventContextOS, bool)
	Page() (ReceivedEventContextPage, bool)
	Referrer() (ReceivedEventContextReferrer, bool)
	Screen() (ReceivedEventContextScreen, bool)
	Session() (ReceivedEventContextSession, bool)
	Timezone() (string, bool)
	UserAgent() (string, bool)
}

type ReceivedEventContextApp interface {
	Name() (string, bool)
	Version() (string, bool)
	Build() (string, bool)
	Namespace() (string, bool)
}

type ReceivedEventContextBrowser interface {
	Name() (string, bool)
	Other() (string, bool)
	Version() (string, bool)
}

type ReceivedEventContextCampaign interface {
	Name() (string, bool)
	Source() (string, bool)
	Medium() (string, bool)
	Term() (string, bool)
	Content() (string, bool)
}

type ReceivedEventContextDevice interface {
	ID() (string, bool)
	AdvertisingID() (string, bool)
	AdTrackingEnabled() (bool, bool)
	Manufacturer() (string, bool)
	Model() (string, bool)
	Name() (string, bool)
	Type() (string, bool)
	Token() (string, bool)
}

type ReceivedEventContextLibrary interface {
	Name() (string, bool)
	Version() (string, bool)
}

type ReceivedEventContextLocation interface {
	City() (string, bool)
	Country() (string, bool)
	Latitude() (float64, bool)
	Longitude() (float64, bool)
	Speed() (float64, bool)
}

type ReceivedEventContextNetwork interface {
	Bluetooth() (bool, bool)
	Carrier() (string, bool)
	Cellular() (bool, bool)
	WiFi() (bool, bool)
}

type ReceivedEventContextOS interface {
	Name() (string, bool)
	Version() (string, bool)
}

type ReceivedEventContextPage interface {
	Path() (string, bool)
	Referrer() (string, bool)
	Search() (string, bool)
	Title() (string, bool)
	URL() (string, bool)
}

type ReceivedEventContextReferrer interface {
	ID() (string, bool)
	Type() (string, bool)
}

type ReceivedEventContextScreen interface {
	Width() (int, bool)
	Height() (int, bool)
	Density() (decimal.Decimal, bool)
}

type ReceivedEventContextSession interface {
	ID() (int, bool)
	Start() (bool, bool)
}
