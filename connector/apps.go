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
)

// ErrWebhookUnauthorized is returned by the ReceiveWebhook method if the
// request was not authorized.
var ErrWebhookUnauthorized = errors.New("webhook unauthorized")

// AppConfig represents the configuration of an app connection.
type AppConfig struct {
	Role         Role
	Settings     []byte
	Firehose     Firehose
	ClientSecret string
	Resource     string
	AccessToken  string
}

// AppConnectionFunc represents functions that create new app connections.
type AppConnectionFunc func(context.Context, *AppConfig) (AppConnection, error)

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
