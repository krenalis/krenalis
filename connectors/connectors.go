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
type FirehoseContextKey struct{}

// Connecter is the interface implemented by the connectors.
type Connecter interface {

	// ApplyConfig applies the configuration config.
	ApplyConfig(ctx context.Context, config map[string]any) error

	// Groups returns the groups starting from the given cursor.
	Groups(ctx context.Context, cursor string) error

	// Properties returns all user and group properties.
	Properties(ctx context.Context) ([]Property, []Property, error)

	// ReceiveWebhook receives a webhook request and returns its events.
	// It returns the ErrWebhookUnauthorized error is the request was not authorized.
	ReceiveWebhook(ctx context.Context, r *http.Request) ([]*Event, error)

	// Resource returns the resource.
	Resource(ctx context.Context) (string, error)

	// SetUsers sets the given users.
	SetUsers(ctx context.Context, users []User) error

	// Users returns the users starting from the given cursor.
	Users(ctx context.Context, cursor string) error
}

// Firehose is the interface implemented by a Firehose.
type Firehose interface {
	ApplyConfig(config map[string]any)
	SetCursor(cursor string)
	UpdateGroup(ident Identity, updateTime time.Time, properties map[string]any, users []string)
	UpdateUser(ident Identity, updateTime time.Time, properties map[string]any, groups []string)
}

type EventType int

const (
	UserChanged EventType = iota
	GroupChanged
	UserCreated
	GroupCreated
	UserDeleted
	GroupDeleted
)

type Event struct {
	Type       EventType
	Time       time.Time
	Resource   string
	Group      string
	User       string
	Properties Properties
}

type Properties map[string]any

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

// Identity is an identity on the app.
type Identity struct {
	// Group identifies the group on the app.
	Group string
	// User identifies the user on the app.
	User string
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
