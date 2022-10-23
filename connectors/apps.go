//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package connectors

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// ErrWebhookUnauthorized is returned by the ReceiveWebhook method if the
// request was not authorized.
var ErrWebhookUnauthorized = errors.New("webhook unauthorized")

// AppConfig represents the configuration of an app connection.
type AppConfig struct {
	Settings     []byte
	Firehose     Firehose
	ClientSecret string
	Resource     string
	AccessToken  string
}

// AppConnectionFunc represents functions that create new app connections.
type AppConnectionFunc func(context.Context, *AppConfig) (AppConnection, error)

// RegisterAppConnector makes an app connector available by the provided name.
// If RegisterAppConnector is called twice with the same name or if fn is nil,
// it panics.
func RegisterAppConnector(name string, f AppConnectionFunc) {
	if f == nil {
		panic("connectors: Register new app function is nil")
	}
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	if _, dup := connectors.apps[name]; dup {
		panic("connectors: Register called twice for app connector " + name)
	}
	connectors.apps[name] = f
}

// AppConnection is the interface implemented by app connections.
type AppConnection interface {
	Connection

	// Groups returns the groups starting from the given cursor.
	Groups(cursor string, properties [][]string) error

	// Properties returns all user and group properties.
	Properties() ([]Property, []Property, error)

	// ReceiveWebhook receives a webhook request and returns its events.
	// It returns the ErrWebhookUnauthorized error is the request was not authorized.
	ReceiveWebhook(r *http.Request) ([]Event, error)

	// Resource returns the resource.
	Resource() (string, error)

	// SetUsers sets the given users.
	SetUsers(users []User) error

	// Users returns the users starting from the given cursor.
	Users(cursor string, properties [][]string) error
}

// NewAppConnection returns a new app connection for the app connector with the
// given name.
func NewAppConnection(ctx context.Context, name string, conf *AppConfig) (AppConnection, error) {
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	f, ok := connectors.apps[name]
	if !ok {
		return nil, fmt.Errorf("connectors: unknown app connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}
