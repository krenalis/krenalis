//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package chichi

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"time"

	"github.com/open2b/chichi/types"

	"github.com/shopspring/decimal"
)

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

// AppInfo represents an app connector info.
type AppInfo struct {
	Name                   string
	SourceDescription      string // It should complete the sentence "Add an action to ..."
	DestinationDescription string // It should complete the sentence "Add an action to ..."
	TermForUsers           string
	TermForGroups          string
	ExternalIDLabel        string
	Icon                   string      // icon in SVG format
	WebhooksPer            WebhooksPer // indicates if webhooks are per connector, resource or connection
	OAuth                  OAuth       // OAuth 2.0 configuration. If the URL is empty the connector does not support OAuth 2.0
	SendingMode            SendingMode // mode of event sending. None for sources and non-supporting event apps.

	// Layout for date and time values. If left empty, it is ISO 8601.
	DateTimeLayout string // If left empty, values are formatted with the layout "2006-01-02T15:04:05.999Z".
	DateLayout     string // If left empty, values are formatted with the layout "2006-01-02".
	TimeLayout     string // If left empty, values are formatted with the layout "15:04:05.999Z".

	newFunc reflect.Value
	ct      reflect.Type
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
	Role        Role
	Settings    []byte
	SetSettings SetSettingsFunc
	Resource    string
	HTTPClient  HTTPClient
	Region      PrivacyRegion
	WebhookURL  string
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
// An app connector also implements at least one of the interfaces AppEvents,
// AppUsers, and AppUsersGroups.
type App interface {

	// Resource returns the resource.
	Resource(ctx context.Context) (string, error)
}

// EventRequest represents an event request.
type EventRequest struct {
	Method string
	URL    string
	Header http.Header
	Body   []byte
}

// AppEvents is the interface implemented by app connectors to which events can
// be sent.
type AppEvents interface {
	App

	// EventRequest returns an event request associated with the provided event
	// type, event, and transformation data. If redacted is true, sensitive
	// authentication data will be redacted in the returned request.
	// This method is safe for concurrent use by multiple goroutines.
	// If the specified event type does not exist, it returns the
	// ErrEventTypeNotExist error.
	EventRequest(ctx context.Context, eventType *EventType, event *Event, data map[string]any, redacted bool) (*EventRequest, error)

	// EventTypes returns the event types of the connector's instance.
	EventTypes(ctx context.Context) ([]*EventType, error)
}

// Cursor represents a cursor used to implement pagination.
type Cursor struct {
	ID        string    // Identifier of the last returned user or group.
	Timestamp time.Time // Timestamp of the last returned user or group, with preserved Location.
	Next      string    // Returned string value of the last call to Users or Groups.
}

// AppUsers is the interface implemented by app connectors that manage users.
type AppUsers interface {
	App

	// CreateUser creates a user with the given properties.
	CreateUser(ctx context.Context, user map[string]any) error

	// UpdateUser updates the user with identifier id setting the given properties.
	UpdateUser(ctx context.Context, id string, user map[string]any) error

	// UserSchema returns the user schema.
	UserSchema(ctx context.Context) (types.Type, error)

	// Users returns the users starting from the given cursor.
	Users(ctx context.Context, properties []string, cursor Cursor) (users []Record, next string, err error)
}

// AppGroups is the interface implemented by app connectors that manage groups.
type AppGroups interface {
	App

	// CreateGroup creates a group with the given properties.
	CreateGroup(ctx context.Context, group map[string]any) error

	// GroupSchema returns the group schema.
	GroupSchema(ctx context.Context) (types.Type, error)

	// Groups returns the groups starting from the given cursor.
	Groups(ctx context.Context, properties []string, cursor Cursor) (groups []Record, next string, err error)

	// UpdateGroup updates the group with identifier id setting the given
	// properties.
	UpdateGroup(ctx context.Context, id string, group map[string]any) error
}

// Webhooks is the interface implemented by app connectors that can receive
// webhooks.
type Webhooks interface {
	App

	// ReceiveWebhook receives a webhook request and returns its payloads. It
	// returns the ErrWebhookUnauthorized error is the request was not authorized.
	// The context is the request's context.
	ReceiveWebhook(r *http.Request) ([]WebhookPayload, error)
}

// Event represents an event.
type Event struct {

	// Keep these fields in sync with the event schema, except for Properties,
	// Source, Traits and Version fields.

	AnonymousId string
	Category    string
	Context     struct {
		Active bool
		App    struct {
			Name      string
			Version   string
			Build     string
			Namespace string
		}
		Browser struct {
			Name    string
			Other   string
			Version string
		}
		Campaign struct {
			Name    string
			Source  string
			Medium  string
			Term    string
			Content string
		}
		Device struct {
			Id                string
			AdvertisingId     string
			AdTrackingEnabled bool
			Manufacturer      string
			Model             string
			Name              string
			Type              string
			Token             string
		}
		IP      string
		Library struct {
			Name    string
			Version string
		}
		Locale   string
		Location struct {
			City      string
			Country   string
			Latitude  float64
			Longitude float64
			Speed     float64
		}
		Network struct {
			Bluetooth bool
			Carrier   string
			Cellular  bool
			WiFi      bool
		}
		OS struct {
			Name    string
			Version string
		}
		Page struct {
			Path     string
			Referrer string
			Search   string
			Title    string
			URL      string
		}
		Referrer struct {
			Id   string
			Type string
		}
		Screen struct {
			Width   int
			Height  int
			Density decimal.Decimal
		}
		SessionId    int
		SessionStart bool
		Timezone     string
		UserAgent    string
	}
	Event      string
	GroupId    string
	MessageId  string
	Name       string
	ReceivedAt time.Time
	SentAt     time.Time
	Timestamp  time.Time
	Type       string
	UserId     string
}
