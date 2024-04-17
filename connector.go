//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package chichi

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/open2b/chichi/types"
)

// ConnectorInfo is the interface implemented by connector infos.
type ConnectorInfo interface {
	ReflectType() reflect.Type
}

// An AccessDeniedError error is returned by a connector method when it is
// unable to access a requested resource due to insufficient permissions.
type AccessDeniedError struct {
	Err error
}

func (err *AccessDeniedError) Error() string {
	return err.Err.Error()
}

// A NotSupportedTypeError error is returned by File.Read and Database.Query
// methods when a column type is not supported.
type NotSupportedTypeError struct {
	Column string
	Type   string
}

// NewNotSupportedTypeError returns a NotSupportedTypeError error for the
// given column and type.
func NewNotSupportedTypeError(column, typ string) error {
	return NotSupportedTypeError{Column: column, Type: typ}
}

func (err NotSupportedTypeError) Error() string {
	return fmt.Sprintf("type %s of the column %q is not supported", err.Type, err.Column)
}

// SuggestPropertyName suggests a valid property name based on s.
// If no valid property name can be determined, it returns an empty string.
func SuggestPropertyName(s string) string {
	if types.IsValidPropertyName(s) {
		return s
	}
	// TODO(marco): implement the logic
	return ""
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

	// Do sends an HTTP request with an Authorization header if required. It
	// returns the response and ensures that the request body is closed, even in
	// the case of errors. Redirects are not followed.
	Do(req *http.Request) (res *http.Response, err error)

	// ClientSecret returns the OAuth client secret of the HTTP client.
	ClientSecret() (string, error)

	// AccessToken returns an OAuth access token.
	AccessToken(ctx context.Context) (string, error)
}

// OAuth represents the connector OAuth 2.0 info.
type OAuth struct {
	AuthURL  string
	TokenURL string

	Scopes []string

	// The lifetime in seconds of the access token.
	// If zero or negative, the lifetime is returned by the TokenURL endpoint.
	ExpiresIn int32
}

// WebhooksPer values indicates if webhooks are per connector, resource or
// source.
type WebhooksPer int

const (
	WebhooksPerNone WebhooksPer = iota
	WebhooksPerConnector
	WebhooksPerResource
	WebhooksPerSource
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
	Resource  string
	User      string
}

func (ev UserChangeEvent) webhookPayload() {}

type UserCreateEvent struct {
	Timestamp  time.Time
	Resource   string
	User       string
	Properties map[string]any
}

func (ev UserCreateEvent) webhookPayload() {}

type UserDeleteEvent struct {
	Timestamp time.Time
	Resource  string
	User      string
}

func (ev UserDeleteEvent) webhookPayload() {}

type UserPropertyChangeEvent struct {
	Timestamp time.Time
	Resource  string
	User      string
	Name      string
	Value     any
}

func (ev UserPropertyChangeEvent) webhookPayload() {}

type GroupChangeEvent struct {
	Timestamp time.Time
	Resource  string
	Group     string
}

func (ev GroupChangeEvent) webhookPayload() {}

type GroupCreateEvent struct {
	Timestamp  time.Time
	Resource   string
	Group      string
	Properties map[string]any
}

func (ev GroupCreateEvent) webhookPayload() {}

type GroupDeleteEvent struct {
	Timestamp time.Time
	Resource  string
	Group     string
}

func (ev GroupDeleteEvent) webhookPayload() {}

type GroupPropertyChangeEvent struct {
	Timestamp time.Time
	Resource  string
	Group     string
	Name      string
	Value     any
}

func (ev GroupPropertyChangeEvent) webhookPayload() {}
