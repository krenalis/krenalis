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
func (app App) Open(ctx context.Context, conf *AppConfig) (AppConnection, error) {
	out := app.open.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(conf)})
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

// OpenAppFunc represents functions that open app connections. Such functions
// are not blocking functions and the context is used by the app methods.
type OpenAppFunc[T AppConnection] func(context.Context, *AppConfig) (T, error)

// AppConnection is the interface implemented by app connections.
//
// An app connection also implements at least one of the interfaces
// AppEventsConnection, AppUsersConnection, and AppUsersGroupsConnection.
type AppConnection interface {

	// Resource returns the resource.
	Resource() (string, error)
}

// AppEventsConnection is the interface implemented by app connections to which
// events can be sent.
type AppEventsConnection interface {
	AppConnection

	// EventTypes returns the connection's event types.
	EventTypes() ([]*EventType, error)

	// SendEvent sends the event, along with the given mapped event.
	// Can be used by multiple goroutines at the same time.
	SendEvent(event Event, mappedEvent map[string]any, eventType string) error
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
	CreateUser(properties Properties) error

	// ReceiveWebhook receives a webhook request and returns its events.
	// It returns the ErrWebhookUnauthorized error is the request was not
	// authorized.
	ReceiveWebhook(r *http.Request) ([]WebhookEvent, error)

	// UserSchema returns the user schema.
	UserSchema() (types.Type, error)

	// UpdateUser updates the user with identifier id setting the given
	// properties.
	UpdateUser(id string, properties Properties) error

	// Users returns the users starting from the given cursor.
	Users(properties []types.Path, cursor Cursor) (users []Object, next string, err error)
}

// AppGroupsConnection is the interface implemented by app connections that
// manage groups.
type AppGroupsConnection interface {
	AppConnection

	// GroupSchema returns the group schema.
	GroupSchema() (types.Type, error)

	// Groups returns the groups starting from the given cursor.
	Groups(properties []types.Path, after Cursor) (groups []Object, cursor string, err error)

	// ReceiveWebhook receives a webhook request and returns its events.
	// It returns the ErrWebhookUnauthorized error is the request was not
	// authorized.
	ReceiveWebhook(r *http.Request) ([]WebhookEvent, error)

	// SetGroup sets the given group.
	SetGroup(group Group) error
}

// Event represents an event.
//
// Keep these fields in sync with the schema in "apis/events/schema.go".
type Event struct {
	Source      int32
	Event       string
	Name        string
	MessageID   string
	AnonymousID string
	UserID      string
	Date        string
	Timestamp   time.Time
	SentAt      time.Time
	ReceivedAt  time.Time
	IP          string
	Network     struct {
		Cellular  bool
		WiFi      bool
		Bluetooth bool
		Carrier   string
	}
	OS struct {
		Name    string
		Version string
	}
	App struct {
		Name      string
		Version   string
		Build     string
		Namespace string
	}
	Screen struct {
		Density int16
		Width   int16
		Height  int16
	}
	UserAgent string
	Browser   struct {
		Name    string
		Other   string
		Version string
	}
	Device struct {
		ID            string
		Name          string
		Manufacturer  string
		Model         string
		Type          string
		Version       string
		AdvertisingID string
	}
	Location struct {
		City      string
		Country   string
		Region    string
		Latitude  float64
		Longitude float64
		Speed     float64
	}
	Locale   string
	Timezone string
	Page     struct {
		URL      string
		Path     string
		Search   string
		Hash     string
		Title    string
		Referrer string
	}
	Referrer struct {
		Type string
		Name string
		URL  string
		Link string
	}
	Campaign struct {
		Name    string
		Source  string
		Medium  string
		Term    string
		Content string
	}
	Library struct {
		Name    string
		Version string
	}
	Properties string
}
