//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package connectors

import (
	"context"
	"net/http"
	"reflect"
)

// Connecter is the interface implemented by the connectors.
type Connecter interface {

	// ApplyConfig applies the configuration config.
	ApplyConfig(account string, config map[string]any) error

	// Groups returns the groups starting from the given cursor.
	Groups(account, cursor string) error

	// Properties returns all user and group properties.
	Properties(account string) ([]Property, []Property, error)

	// ServeWebhook serves a webhook request.
	ServeWebhook(w http.ResponseWriter, r *http.Request) error

	// SetUsers sets the given users.
	SetUsers(token string, users []User) error

	// Users returns the users starting from the given cursor.
	Users(account, cursor string) error
}

// Firehose is the interface implemented by a Firehose.
type Firehose interface {
	CreateGroup(ident Identity, creationTime int64, properties map[string]string)
	CreateUser(ident Identity, creationTime int64, properties map[string]string)
	DeleteGroup(ident Identity)
	DeleteUser(ident Identity)
	SetCursor(cursor string)
	UpdateGroup(ident Identity, updateTime int64, properties map[string]string, users []string)
	UpdateUser(ident Identity, updateTime int64, properties map[string]string, groups []string)
}

type Properties map[string]string

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
	// Account identifies the account on the app.
	Account string
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

func Connector(ctx context.Context, name string, clientSecret string, fh Firehose) Connecter {
	t := reflect.TypeOf(connectors[name])
	v := reflect.New(t.Elem())
	reflect.Indirect(v).FieldByName("ClientSecret").Set(reflect.ValueOf(clientSecret))
	reflect.Indirect(v).FieldByName("Context").Set(reflect.ValueOf(ctx))
	reflect.Indirect(v).FieldByName("Firehose").Set(reflect.ValueOf(fh))
	return v.Interface().(Connecter)
}
