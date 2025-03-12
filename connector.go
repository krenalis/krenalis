//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package meergo

import (
	"context"
	"net/http"
	"reflect"
	"time"
)

// ConnectorInfo is the interface implemented by connector infos.
type ConnectorInfo interface {
	ReflectType() reflect.Type
}

// A SetSettingsFunc value is a function used by connectors to set settings.
type SetSettingsFunc func(context.Context, []byte) error

// TimeLayouts represents the layouts for time values.
// If a layout is left empty, it is ISO 8601.
type TimeLayouts struct {
	DateTime string // if left empty, values are formatted with the layout "2006-01-02T15:04:05.999Z"
	Date     string // if left empty, values are formatted with the layout "2006-01-02"
	Time     string // if left empty, values are formatted with the layout "15:04:05.999Z"
}

// HTTPClient is the interface implemented by the HTTP client used by
// connectors.
type HTTPClient interface {

	// Do sends an HTTP request with an Authorization header if required. It returns
	// the response and ensures that the request body is closed, even in the case of
	// errors. Redirects are not followed.
	//
	// If an error occurs during GET, PUT, DELETE, or HEAD requests, it retries
	// using the client's backoff policy or a default policy if the client has no
	// policy.
	Do(req *http.Request) (res *http.Response, err error)

	// DoIdempotent behaves like Do, but unlike Do, which assumes GET, PUT, DELETE,
	// and HEAD requests are idempotent by default, it allows to explicitly specify
	// idempotency.
	//
	// If an error occurs during an idempotent request, it retries using the
	// client's backoff policy or a default policy if the client has no policy.
	DoIdempotent(req *http.Request, idempotent bool) (*http.Response, error)

	// ClientSecret returns the OAuth client secret of the HTTP client.
	ClientSecret() (string, error)

	// AccessToken returns an OAuth access token.
	AccessToken(ctx context.Context) (string, error)

	// UUID returns a random version 4 UUID, suitable for use as an idempotency key.
	UUID() string
}

// WebhooksPer values indicates if webhooks are per account, connection, or
// connector.
type WebhooksPer int

const (
	WebhooksPerNone WebhooksPer = iota
	WebhooksPerAccount
	WebhooksPerConnection
	WebhooksPerConnector
)

// Role represents a role.
type Role int

const (
	Both        Role = iota // both
	Source                  // source
	Destination             // destination
)

// String returns the string representation of role.
// It panics if role is not a valid Role value.
func (role Role) String() string {
	switch role {
	case Both:
		return "Both"
	case Source:
		return "Source"
	case Destination:
		return "Destination"
	}
	panic("invalid role")
}

type WebhookPayload interface {
	webhookPayload()
}

type UserChangeEvent struct {
	Timestamp time.Time
	Account   string
	User      string
}

func (ev UserChangeEvent) webhookPayload() {}

type UserCreateEvent struct {
	Timestamp  time.Time
	Account    string
	User       string
	Properties map[string]any
}

func (ev UserCreateEvent) webhookPayload() {}

type UserDeleteEvent struct {
	Timestamp time.Time
	Account   string
	User      string
}

func (ev UserDeleteEvent) webhookPayload() {}

type UserPropertyChangeEvent struct {
	Timestamp time.Time
	Account   string
	User      string
	Name      string
	Value     any
}

func (ev UserPropertyChangeEvent) webhookPayload() {}

type GroupChangeEvent struct {
	Timestamp time.Time
	Account   string
	Group     string
}

func (ev GroupChangeEvent) webhookPayload() {}

type GroupCreateEvent struct {
	Timestamp  time.Time
	Account    string
	Group      string
	Properties map[string]any
}

func (ev GroupCreateEvent) webhookPayload() {}

type GroupDeleteEvent struct {
	Timestamp time.Time
	Account   string
	Group     string
}

func (ev GroupDeleteEvent) webhookPayload() {}

type GroupPropertyChangeEvent struct {
	Timestamp time.Time
	Account   string
	Group     string
	Name      string
	Value     any
}

func (ev GroupPropertyChangeEvent) webhookPayload() {}
