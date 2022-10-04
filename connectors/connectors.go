//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package connectors

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"

	"chichi/pkg/open2b/sql"
)

// Connecter is the interface implemented by the connectors.
type Connecter interface {

	// Groups returns the groups starting from the given cursor.
	Groups(account, cursor string, properties []string) error

	// Properties returns all user and group properties.
	Properties(account string) ([]Property, []Property, error)

	// ServeWebhook serves a webhook request.
	ServeWebhook(w http.ResponseWriter, r *http.Request) error

	// SetUsers sets the given users.
	SetUsers(token string, users []User) error

	// Users returns the users starting from the given cursor.
	Users(account, cursor string, properties []string) error
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

// TODO(Gianluca): this should be removed, it's just for prototyping:
var MySQLDB *sql.DB

var connectors = map[string]any{}

func RegisterConnector(name string, value any) {
	connectors[name] = value
}

func Connector(ctx context.Context, name string, clientSecret string) Connecter {
	t := reflect.TypeOf(connectors[name])
	v := reflect.New(t.Elem())
	reflect.Indirect(v).FieldByName("ClientSecret").Set(reflect.ValueOf(clientSecret))
	reflect.Indirect(v).FieldByName("Context").Set(reflect.ValueOf(ctx))
	return v.Interface().(Connecter)
}

func SetCursor(cursor string) {
	return
}

func UpdateGroup(ident Identity, updateTime int64, properties map[string]string, users []string) {
	return
}

func UpdateUser(ident Identity, updateTime int64, properties map[string]string, groups []string) {
	// TODO(Gianluca): use the correct user. Where should we take it?
	connector := 1
	data, err := json.Marshal(properties)
	if err != nil {
		// TODO(Gianluca): improve error handling here.
		panic(err)
	}
	_, err = MySQLDB.Table("ConnectorsRawUserData").Add(
		map[string]any{
			"account":   ident.Account,
			"connector": connector,
			"data":      string(data),
		},
		nil, // TODO(Gianluca): pass the correct value to the "onDuplicate" parameter.
	)
	if err != nil {
		panic(err)
	}
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
