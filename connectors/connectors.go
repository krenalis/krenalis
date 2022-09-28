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

type Connecter interface {
	Groups(ctx context.Context, account, cursor string, properties []string) error
	Properties(ctx context.Context, account string) ([]Property, error)
	ServeWebhook(ctx context.Context, w http.ResponseWriter, r *http.Request) error
	SetUsers(ctx context.Context, token string, users []User) error
	Users(ctx context.Context, account, cursor string, properties []string) error
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

type Identity struct {
	Account string
	Group   string
	User    string
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

func SetCursor(cursor string) {
	return
}

func UpdateGroup(ident Identity, updateTime int64, properties map[string]string, users []string) {
	return
}

func UpdateUser(ident Identity, updateTime int64, properties map[string]string, groups []string) {
	return
}

func CreateGroup(ident Identity, creationTime int64, properties map[string]string) {
	return
}

func CreateUser(ident Identity, creationTime int64, properties map[string]string) {
	return
}

func DeleteGroup(ident Identity) {
	return
}

func DeleteUser(ident Identity) {
	return
}
