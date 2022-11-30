//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import (
	"fmt"
	"time"

	"chichi/apis/types"
	"chichi/connector/ui"
)

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

// Connector represents a connector.
type Connector struct {
	Name        string      // name
	Type        Type        // type
	Icon        []byte      // icon in SVG format
	OAuth       OAuth       // OAuth 2.0 configuration. If the URL is empty the connector does not support OAuth 2.0
	WebhooksPer WebhooksPer // indicates if webhooks are per connector, resource or connection
}

// Type represents a connector type.
type Type int

const (
	AppType Type = iota + 1
	DatabaseType
	EventStreamType
	FileType
	MobileType
	ServerType
	StorageType
	WebsiteType
)

// OAuth represents the connector OAuth 2.0 info.
type OAuth struct {
	URL              string
	Scope            string
	DefaultTokenType string
	DefaultExpiresIn int
	ForcedExpiresIn  string
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

// Connection is the interface implemented by connections.
type Connection interface {

	// Connector returns the connector.
	Connector() *Connector

	// ServeUI serves the connector's user interface.
	ServeUI(event string, values []byte) (*ui.Form, *ui.Alert, error)
}

// Firehose is the interface implemented by a Firehose.
type Firehose interface {

	// ReceiveEvent receives the given event for the connection.
	// The event.Resource field must be empty.
	ReceiveEvent(event Event)

	// SetCursor sets the given cursor for the connection.
	SetCursor(cursor string)

	// SetGroup sets the properties of the given group. The last update time of
	// a property is the time in timestamps with the property name as key.
	// If no such key exists, the last update time is timestamp.
	SetGroup(group string, properties map[string]any, timestamp time.Time, timestamps map[string]time.Time)

	// SetGroupUsers sets the users of a group.
	SetGroupUsers(group string, users []string)

	// SetSettings sets the given settings of the connection.
	SetSettings(settings []byte) error

	// SetUser sets the properties of the given user. The last update time of a
	// property is the time in timestamps with the property name as key. If no
	// such key exists, the last update time is timestamp.
	SetUser(user string, properties map[string]any, timestamp time.Time, timestamps map[string]time.Time)

	// SetUserGroups sets the groups of a user.
	SetUserGroups(user string, groups []string)

	// WebhookURL returns the URL of the webhook.
	WebhookURL() string
}

type Event interface {
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

type User struct {
	ID         string
	Groups     []string
	Properties Properties
}
