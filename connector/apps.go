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
	Icon        string      // icon in SVG format
	OAuth       OAuth       // OAuth 2.0 configuration. If the URL is empty the connector does not support OAuth 2.0
	WebhooksPer WebhooksPer // indicates if webhooks are per connector, resource or connection
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

// OpenAppFunc represents functions that open app connections.
type OpenAppFunc func(context.Context, *AppConfig) (AppConnection, error)

// AppConnection is the interface implemented by app connections.
type AppConnection interface {

	// ActionTypes returns the connection's action types.
	ActionTypes() ([]*ActionType, error)

	// Groups returns the groups starting from the given cursor.
	Groups(cursor string, properties []PropertyPath) error

	// ReceiveWebhook receives a webhook request and returns its events.
	// It returns the ErrWebhookUnauthorized error is the request was not authorized.
	ReceiveWebhook(r *http.Request) ([]Event, error)

	// Resource returns the resource.
	Resource() (string, error)

	// Schemas returns user and group schemas.
	Schemas() (types.Type, types.Type, error)

	// SetUsers sets the given users.
	SetUsers(users []User) error

	// Users returns the users starting from the given cursor.
	Users(cursor string, properties []PropertyPath) error
}
