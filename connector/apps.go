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
	"time"

	"chichi/apis/types"
)

// ErrWebhookUnauthorized is returned by the ReceiveWebhook method if the
// request was not authorized.
var ErrWebhookUnauthorized = errors.New("webhook unauthorized")

// PropertyPath represents a property path.
type PropertyPath []string

// App represents an app connector.
type App struct {
	Name        string
	Icon        string         // icon in SVG format
	Endpoints   map[int]string // endpoints' names by their identifier. nil for connectors that do not use endpoints, otherwise contains at least one value.
	OAuth       OAuth          // OAuth 2.0 configuration. If the URL is empty the connector does not support OAuth 2.0
	WebhooksPer WebhooksPer    // indicates if webhooks are per connector, resource or connection
	Open        OpenAppFunc
}

// AppConfig represents the configuration of an app connection.
type AppConfig struct {
	Role         Role
	Settings     []byte
	Firehose     Firehose
	ClientSecret string
	Resource     string
	AccessToken  string
}

// OpenAppFunc represents functions that open app connections. Such functions
// are not blocking functions and the context is used by the app methods.
type OpenAppFunc func(context.Context, *AppConfig) (AppConnection, error)

// AppConnection is the interface implemented by app connections.
type AppConnection interface {

	// ActionTypes returns the connection's action types.
	ActionTypes() ([]*ActionType, error)

	// Groups returns the groups starting from the given cursor.
	Groups(cursor string, properties []PropertyPath) error

	// ReceiveWebhook receives a webhook request and returns its events.
	// It returns the ErrWebhookUnauthorized error is the request was not authorized.
	ReceiveWebhook(r *http.Request) ([]WebhookEvent, error)

	// Resource returns the resource.
	Resource() (string, error)

	// Schemas returns user and group schemas.
	Schemas() (types.Type, types.Type, error)

	// SendEvent sends event, along with the given mapped event, to the
	// endpoint. actionType specifies the action type corresponding to the
	// event.
	//
	// SendEvent can be used by multiple goroutines at the same time.
	SendEvent(event Event, mappedEvent map[string]any, actionType, endpoint int) error

	// SetUsers sets the given users.
	SetUsers(users []User) error

	// Users returns the users starting from the given cursor.
	Users(cursor string, properties []PropertyPath) error
}

// Event represents an event.
//
// Keep these fields in sync with the schema in "apis/events/schema.go".
type Event struct {
	Source      int32
	Event       string
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
		Density uint16
		Width   uint16
		Height  uint16
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
