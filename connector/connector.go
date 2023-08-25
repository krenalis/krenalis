//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"chichi/connector/types"
	"chichi/connector/ui"
)

// Connector is the interface implemented by connectors.
type Connector interface {
	ConnectionReflectType() reflect.Type
}

// An AccessDeniedError error is returned by a connector method when it is
// unable to access a requested resource due to insufficient permissions.
type AccessDeniedError struct {
	Err error
}

func (err *AccessDeniedError) Error() string {
	return err.Err.Error()
}

// A NotSupportedTypeError error is returned by FileConnector.Read and
// DatabaseConnector.Query methods when a column type is not supported.
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

// Type represents a connector type.
type Type int

const (
	AppType Type = iota + 1
	DatabaseType
	FileType
	MobileType
	ServerType
	StorageType
	StreamType
	WebsiteType
)

// A SetSettingsFunc value is a function used by connectors to set settings.
type SetSettingsFunc func(context.Context, []byte) error

// HTTPClient is the interface implemented by the HTTP client used by
// connections.
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

// Role represents the role of a connection.
type Role int

const (
	SourceRole      Role = iota + 1 // source
	DestinationRole                 // destination
)

// String returns the string representation of role.
// It panics if role is not a valid Role value.
func (role Role) String() string {
	switch role {
	case SourceRole:
		return "Source"
	case DestinationRole:
		return "Destination"
	}
	panic("invalid role")
}

// UI is the interface implemented by connections that have a UI.
type UI interface {

	// ServeUI serves the connector's user interface.
	ServeUI(event string, values []byte) (*ui.Form, *ui.Alert, error)

	// ValidateSettings validates the settings received from the UI and returns them
	// in a format suitable for storage.
	ValidateSettings(values []byte) ([]byte, error)
}

type WebhookEvent interface {
	event()
}

type UserChangeEvent struct {
	Timestamp time.Time
	Resource  string
	User      string
}

func (ev UserChangeEvent) event() {}

type UserCreateEvent struct {
	Timestamp  time.Time
	Resource   string
	User       string
	Properties Properties
}

func (ev UserCreateEvent) event() {}

type UserDeleteEvent struct {
	Timestamp time.Time
	Resource  string
	User      string
}

func (ev UserDeleteEvent) event() {}

type UserPropertyChangeEvent struct {
	Timestamp time.Time
	Resource  string
	User      string
	Name      string
	Value     any
}

func (ev UserPropertyChangeEvent) event() {}

type GroupChangeEvent struct {
	Timestamp time.Time
	Resource  string
	Group     string
}

func (ev GroupChangeEvent) event() {}

type GroupCreateEvent struct {
	Timestamp  time.Time
	Resource   string
	Group      string
	Properties Properties
}

func (ev GroupCreateEvent) event() {}

type GroupDeleteEvent struct {
	Timestamp time.Time
	Resource  string
	Group     string
}

func (ev GroupDeleteEvent) event() {}

type GroupPropertyChangeEvent struct {
	Timestamp time.Time
	Resource  string
	Group     string
	Name      string
	Value     any
}

func (ev GroupPropertyChangeEvent) event() {}

type Properties map[string]any

type Property struct {
	Name       string           `json:"name,omitempty"`
	Options    []PropertyOption `json:"options,omitempty"`
	Label      string           `json:"label,omitempty"`
	Type       types.Type       `json:"type,omitempty"`
	Properties []Property       `json:"properties,omitempty"`
}

type PropertyOption struct {
	Label string
	Value string
}

type Group struct {
	ID         string
	Properties Properties
}

type EventType struct {
	ID          string
	Name        string
	Description string
	Schema      types.Type
}
