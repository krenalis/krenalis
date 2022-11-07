//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import (
	"time"

	"chichi/apis/types"
	"chichi/connector/ui"
)

// Connector represents a connector.
type Connector struct {
	Name        string      // name
	Type        Type        // type
	Icon        []byte      // icon in SVG format
	OAuth       OAuth       // OAuth 2.0 configuration. If the URL is empty the connector does not support OAuth 2.0
	WebhooksPer WebhooksPer // indicates if webhooks are per connector, resource or data source
}

// Type represents a connector type.
type Type int

const (
	TypeApp Type = iota + 1
	TypeDatabase
	TypeFile
	TypeStream
)

// OAuth represents the connector OAuth 2.0 info.
type OAuth struct {
	URL              string
	Scope            string
	DefaultTokenType string
	DefaultExpiresIn int
	ForcedExpiresIn  string
}

// WebhooksPer values indicates if webhooks are per connector, resource or data
// source.
type WebhooksPer int

const (
	WebhooksPerNone WebhooksPer = iota
	WebhooksPerConnector
	WebhooksPerResource
	WebhooksPerDataSource
)

// Connection is the interface implemented by connections.
type Connection interface {

	// Connector returns the connector.
	Connector() *Connector

	// ServeUI serves the connector's user interface.
	ServeUI(event string, values []byte) (*ui.Form, error)
}

// Firehose is the interface implemented by a Firehose.
type Firehose interface {

	// PutRecord puts a record of a File connection as a slice of any.
	PutRecord(record []any)

	// PutRecordString puts a record of a File connection as a string slice.
	PutRecordString(record []string)

	// ReceiveEvent receives the given event for the data source.
	// The event.Resource field must be empty.
	ReceiveEvent(event Event)

	// SetColumns sets the columns of a File connection.
	SetColumns(columns []Column) error

	// SetCursor sets the given cursor for the data source.
	SetCursor(cursor string)

	// SetGroup sets the properties of the given group. timestamp is the last
	// update time of the properties. If a property value has the
	// TimestampedValue type, the Timestamp field represents the timestamp of
	// the property.
	SetGroup(group string, timestamp time.Time, properties map[string]any)

	// SetGroupUsers sets the users of a group.
	SetGroupUsers(group string, users []string)

	// SetSettings sets the given settings of the data source.
	SetSettings(settings []byte) error

	// SetUser sets the properties of the given user. timestamp is the last
	// update time of the properties. If a property value has the
	// TimestampedValue type, the Timestamp field represents the timestamp of
	// the property.
	SetUser(user string, timestamp time.Time, properties map[string]any)

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

type TimestampedValue struct {
	Timestamp time.Time
	Value     any
}

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
