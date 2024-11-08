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
	"net/http"
	"reflect"
	"time"

	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/types"
)

// ErrEventTypeNotExist is returned by the Schema method if the event type does
// not exist.
var ErrEventTypeNotExist = errors.New("event type does not exist")

// ErrWebhookUnauthorized is returned by the ReceiveWebhook method if the
// request was not authorized.
var ErrWebhookUnauthorized = errors.New("webhook unauthorized")

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
	Events = 1 << iota
	Users
	Groups
)

// AppInfo represents an app connector info.
type AppInfo struct {
	Name                   string
	Targets                Targets
	SourceDescription      string // It should complete the sentence "Add an action to ...".
	DestinationDescription string // It should complete the sentence "Add an action to ...".
	TermForUsers           string
	TermForGroups          string
	IdentityIDLabel        string
	WebhooksPer            WebhooksPer   // indicates if webhooks are per account, connection, or connector.
	OAuth                  OAuth         // OAuth 2.0 configuration. If the URL is empty the connector does not support OAuth 2.0.
	BackoffPolicy          BackoffPolicy // backoff policy. It controls retry timing using provided strategies or custom ones.
	SendingMode            SendingMode   // mode of event sending. None for sources and non-supporting event apps.
	TimeLayouts            TimeLayouts   // layouts for time values. If left empty, it is ISO 8601.
	Icon                   string        // icon in SVG format.

	newFunc reflect.Value
	ct      reflect.Type
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
func (app AppInfo) New(conf *AppConfig) (App, error) {
	out := app.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface().(App)
	err, _ := out[1].Interface().(error)
	return c, err
}

// AppConfig represents the configuration of an app connector.
type AppConfig struct {
	Settings     []byte
	SetSettings  SetSettingsFunc
	OAuthAccount string
	HTTPClient   HTTPClient
	Region       PrivacyRegion
	WebhookURL   string
}

// PrivacyRegion represents a privacy region.
type PrivacyRegion string

const (
	PrivacyRegionNotSpecified PrivacyRegion = ""
	PrivacyRegionEurope       PrivacyRegion = "Europe"
)

// AppNewFunc represents functions that create new app connector instances.
type AppNewFunc[T App] func(*AppConfig) (T, error)

// App is the interface implemented by app connectors.
//
// An app connector can also implement the AppEvents, AppOAuth, AppRecords, and
// Webhooks interfaces.
type App interface {

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
	Schema(ctx context.Context, target Targets, role Role, eventType string) (types.Type, error)
}

// EventRequest represents an event request.
type EventRequest struct {
	Endpoint   string      // Destination identifier, e.g., "us", "europe", or leave empty.
	Method     string      // HTTP method (e.g., "POST").
	URL        string      // URL to which the request will be sent.
	Idempotent bool        // Indicates whether the request is idempotent.
	Header     http.Header // Header fields to be included with the request.
	Body       []byte      // The body of the request.
}

// EventType represents a type of event that the app can receive.
type EventType struct {
	ID          string // identifier; must be unique for each event type
	Name        string // name to be displayed
	Description string // description to be displayed
}

// AppEvents is the interface implemented by app connectors to which events can
// be sent.
type AppEvents interface {
	App

	// EventRequest returns a request to dispatch an event to the app. event is the
	// event to dispatch, eventType is the type of event to dispatch, schema is its
	// schema, properties are the property values conforming to the schema, and
	// redacted indicates whether authentication data must be redacted in the
	// returned request.
	//
	// If the event type does not have a schema, schema is the invalid schema and
	// properties is nil.
	//
	// This method is safe for concurrent use by multiple goroutines. If the
	// specified event type does not exist, it returns the ErrEventTypeNotExist
	// error.
	EventRequest(ctx context.Context, event Event, eventType string, schema types.Type, properties map[string]any, redacted bool) (*EventRequest, error)

	// EventTypes returns the event types of the connector's instance.
	EventTypes(ctx context.Context) ([]*EventType, error)
}

// AppOAuth is the interface implemented by apps that support OAuth.
type AppOAuth interface {
	App

	// OAuthAccount returns the app's account associated with the OAuth
	// authorization.
	OAuthAccount(ctx context.Context) (string, error)
}

// Record represents an app record.
type Record struct {
	ID         string         // Identifier.
	Properties map[string]any // Properties.

	// LastChangeTime is the record's last change time, whose location can be
	// anything, not necessarily UTC. The precision of this time is limited to
	// microseconds; any precision beyond microseconds will be truncated.
	LastChangeTime time.Time

	// Associations contains the identifiers of the user's groups or the group's users.
	// It is not significant if it is nil.
	Associations []string

	// Err reports an error that occurred while reading the record.
	// If Err is not nil, only the ID field is significant.
	Err error
}

// UpsertRecord represents a record to update or create in the app.
type UpsertRecord struct {
	// Identifier of the record. It is empty when creating a new record.
	ID string

	// Properties of the record. It contains at least one property and conforms
	// to the schema as returned by the Schema method.
	Properties map[string]any // Properties of the record.
}

// AppRecords is the interface implemented by app connectors that manage users,
// groups, or both. The target parameter is Users or Groups depending on the
// connector supported targets.
type AppRecords interface {
	App

	// Records returns the records of the specified target. The target can only be
	// either Users or Groups, and it must be a target supported by the connector.
	// If lastChangeTime is not the zero time, only the records changed or created
	// at or after that time will be returned, and its precision is limited to
	// microseconds. If ids is not nil, only records with identifiers in ids will be
	// returned, if any. properties are the names of the properties to read, and
	// cursor represents the position from which to start reading the records; it is
	// the cursor value returned by the previous call in a paginated query.
	// Subsequent calls will use this cursor value to retrieve the next batch of
	// records.
	//
	// The properties returned in records may include more than those requested and
	// must conform to the schema returned by the Schema method. The string return
	// value is used as the cursor in the subsequent call. It can be any UTF-8
	// encoded string, including an empty string. If there are no more records to
	// return, the method returns the last records read (if any) along with the
	// io.EOF error.
	Records(ctx context.Context, target Targets, lastChangeTime time.Time, ids, properties []string, cursor string) ([]Record, string, error)

	// Upsert updates or creates records in the app for the specified target. It
	// processes the first record in the records slice and may process additional
	// records as needed based on the app's API capabilities.
	//
	// It returns a slice representing the indexes of the processed records, along
	// with any error encountered. Index 0 may be omitted from the returned slice,
	// and nil may be returned if the only record processed is the first one.
	Upsert(ctx context.Context, target Targets, records []UpsertRecord) ([]int, error)
}

// Webhooks is the interface implemented by app connectors that can receive
// webhooks.
type Webhooks interface {
	App

	// ReceiveWebhook receives a webhook request and returns its payloads. If
	// webhooks are per connection, role is the connection's role, otherwise is
	// Both. It returns the ErrWebhookUnauthorized error is the request was not
	// authorized. The context is the request's context.
	ReceiveWebhook(r *http.Request, role Role) ([]WebhookPayload, error)
}

// Event represents an event.
type Event interface {
	AnonymousId() string
	Category() string
	Context() EventContext
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

type EventContext interface {
	App() (EventContextApp, bool)
	Browser() (EventContextBrowser, bool)
	Campaign() (EventContextCampaign, bool)
	Device() (EventContextDevice, bool)
	IP() string
	Library() (EventContextLibrary, bool)
	Locale() string
	Location() (EventContextLocation, bool)
	Network() (EventContextNetwork, bool)
	OS() (EventContextOS, bool)
	Page() (EventContextPage, bool)
	Referrer() (EventContextReferrer, bool)
	Screen() (EventContextScreen, bool)
	Session() (EventContextSession, bool)
	Timezone() string
	UserAgent() string
}

type EventContextApp interface {
	Name() string
	Version() string
	Build() string
	Namespace() string
}

type EventContextBrowser interface {
	Name() string
	Other() string
	Version() string
}

type EventContextCampaign interface {
	Name() string
	Source() string
	Medium() string
	Term() string
	Content() string
}

type EventContextDevice interface {
	Id() string
	AdvertisingId() string
	AdTrackingEnabled() bool
	Manufacturer() string
	Model() string
	Name() string
	Type() string
	Token() string
}

type EventContextLibrary interface {
	Name() string
	Version() string
}
type EventContextLocation interface {
	City() string
	Country() string
	Latitude() float64
	Longitude() float64
	Speed() float64
}
type EventContextNetwork interface {
	Bluetooth() bool
	Carrier() string
	Cellular() bool
	WiFi() bool
}
type EventContextOS interface {
	Name() string
	Version() string
}

type EventContextPage interface {
	Path() string
	Referrer() string
	Search() string
	Title() string
	URL() string
}

type EventContextReferrer interface {
	Id() string
	Type() string
}

type EventContextScreen interface {
	Width() int
	Height() int
	Density() decimal.Decimal
}
type EventContextSession interface {
	Id() int
	Start() bool
}
