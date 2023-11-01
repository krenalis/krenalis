//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"time"

	"chichi/connector/types"
)

// ErrWebhookUnauthorized is returned by the ReceiveWebhook method if the
// request was not authorized.
var ErrWebhookUnauthorized = errors.New("webhook unauthorized")

// App represents an app connector.
type App struct {
	Name                   string
	SourceDescription      string // It should complete the sentence "Add an action to ..."
	DestinationDescription string // It should complete the sentence "Add an action to ..."
	TermForUsers           string
	TermForGroups          string
	Icon                   string      // icon in SVG format
	WebhooksPer            WebhooksPer // indicates if webhooks are per connector, resource or connection
	OAuth                  OAuth       // OAuth 2.0 configuration. If the URL is empty the connector does not support OAuth 2.0

	open reflect.Value
	ct   reflect.Type
}

// ConnectionReflectType returns the type of the value implementing the app
// connection.
func (app App) ConnectionReflectType() reflect.Type {
	return app.ct
}

// Open opens an app connection.
func (app App) Open(conf *AppConfig) (AppConnection, error) {
	out := app.open.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface().(AppConnection)
	err, _ := out[1].Interface().(error)
	return c, err
}

// AppConfig represents the configuration of an app connection.
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

// OpenAppFunc represents functions that open app connections.
type OpenAppFunc[T AppConnection] func(*AppConfig) (T, error)

// AppConnection is the interface implemented by app connections.
//
// An app connection also implements at least one of the interfaces
// AppEventsConnection, AppUsersConnection, and AppUsersGroupsConnection.
type AppConnection interface {

	// Resource returns the resource.
	Resource(ctx context.Context) (string, error)
}

// AppEventsConnection is the interface implemented by app connections to which
// events can be sent.
type AppEventsConnection interface {
	AppConnection

	// EventTypes returns the connection's event types.
	EventTypes(ctx context.Context) ([]*EventType, error)

	// PreviewSendEvent returns a preview of the event that would be sent when
	// calling SendEvent with the same arguments.
	PreviewSendEvent(ctx context.Context, eventType string, event *Event, data map[string]any) ([]byte, error)

	// SendEvent sends the event, along with the given mapped data.
	// Can be used by multiple goroutines at the same time.
	SendEvent(ctx context.Context, eventType string, event *Event, data map[string]any) error
}

// Object represents either a user or a group.
type Object struct {
	ID         string         // Identifier.
	Properties map[string]any // Properties.
	Timestamp  time.Time      // Last modification time, not necessarily in UTC.

	// Associations contains the identifiers of the user's groups or the group's users.
	// It is not significant if it is nil.
	Associations []string
}

// Cursor represents a cursor used to implement pagination.
type Cursor struct {
	ID        string    // Identifier of the last returned user or group.
	Timestamp time.Time // Timestamp of the last returned user or group, with preserved Location.
	Next      string    // Returned string value of the last call to Users or Groups.
}

// AppUsersConnection is the interface implemented by app connections that
// manage users.
type AppUsersConnection interface {
	AppConnection

	// CreateUser creates a user with the given properties.
	CreateUser(ctx context.Context, properties map[string]any) error

	// ReceiveWebhook receives a webhook request and returns its events.
	// It returns the ErrWebhookUnauthorized error is the request was not
	// authorized. The context is the request's context.
	ReceiveWebhook(r *http.Request) ([]WebhookPayload, error)

	// UpdateUser updates the user with identifier id setting the given
	// properties.
	UpdateUser(ctx context.Context, id string, properties map[string]any) error

	// UserSchema returns the user schema.
	UserSchema(ctx context.Context) (types.Type, error)

	// Users returns the users starting from the given cursor.
	Users(ctx context.Context, properties []string, cursor Cursor) (users []Object, next string, err error)
}

// AppGroupsConnection is the interface implemented by app connections that
// manage groups.
type AppGroupsConnection interface {
	AppConnection

	// CreateGroup creates a group with the given properties.
	CreateGroup(ctx context.Context, properties map[string]any) error

	// GroupSchema returns the group schema.
	GroupSchema(ctx context.Context) (types.Type, error)

	// Groups returns the groups starting from the given cursor.
	Groups(ctx context.Context, properties []string, cursor Cursor) (groups []Object, next string, err error)

	// ReceiveWebhook receives a webhook request and returns its events.
	// It returns the ErrWebhookUnauthorized error is the request was not
	// authorized. The context is the request's context.
	ReceiveWebhook(r *http.Request) ([]WebhookPayload, error)

	// UpdateGroup updates the group with identifier id setting the given
	// properties.
	UpdateGroup(ctx context.Context, id string, properties map[string]any) error
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
			Density float64
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
