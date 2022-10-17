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
	"net/http"
	"reflect"
	"time"
)

// ErrWebhookUnauthorized is returned by the ReceiveWebhook method if the
// request was not authorized.
var ErrWebhookUnauthorized = errors.New("webhook unauthorized")

type AccessTokenContextKey struct{}
type SettingsContextKey struct{}
type FirehoseContextKey struct{}

// Connecter is the interface implemented by the connectors.
type Connecter interface {

	// Groups returns the groups starting from the given cursor.
	Groups(ctx context.Context, cursor string, properties []string) error

	// Properties returns all user and group properties.
	Properties(ctx context.Context) ([]Property, []Property, error)

	// ReceiveWebhook receives a webhook request and returns its events.
	// It returns the ErrWebhookUnauthorized error is the request was not authorized.
	ReceiveWebhook(ctx context.Context, r *http.Request) ([]Event, error)

	// Resource returns the resource.
	Resource(ctx context.Context) (string, error)

	// SetUsers sets the given users.
	SetUsers(ctx context.Context, users []User) error

	// Users returns the users starting from the given cursor.
	Users(ctx context.Context, cursor string, properties []string) error
}

// Firehose is the interface implemented by a Firehose.
type Firehose interface {

	// ReceiveEvent receives the given event for the data source.
	// The event.Resource field must be empty.
	ReceiveEvent(event Event)

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
	Name    string
	Options []PropertyOption
	Label   string
	Type    string
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

type Conf struct {
	ClientSecret string
}

var connectors = map[string]any{}

func RegisterConnector(name string, value any) {
	connectors[name] = value
}

func Connector(name string, clientSecret string) Connecter {
	t := reflect.TypeOf(connectors[name])
	v := reflect.New(t.Elem())
	reflect.Indirect(v).FieldByName("ClientSecret").Set(reflect.ValueOf(clientSecret))
	return v.Interface().(Connecter)
}
