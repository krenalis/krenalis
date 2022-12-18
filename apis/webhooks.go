//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"

	_connector "chichi/connector"
)

// WebhooksPer values indicates if webhooks are per connector, resource or
// source.
type WebhooksPer int

const (
	WebhooksPerNone WebhooksPer = iota
	WebhooksPerConnector
	WebhooksPerResource
	WebhooksPerSource
)

// MarshalJSON implements the json.Marshaler interface.
// It panics if w is not a valid WebhooksPer value.
func (per WebhooksPer) MarshalJSON() ([]byte, error) {
	return []byte(`"` + per.String() + `"`), nil
}

// Scan implements the sql.Scanner interface.
func (per *WebhooksPer) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an api.WebhooksPer value", src)
	}
	var p WebhooksPer
	switch s {
	case "None":
		p = WebhooksPerNone
	case "Connector":
		p = WebhooksPerConnector
	case "Resource":
		p = WebhooksPerResource
	case "Source":
		p = WebhooksPerSource
	default:
		return fmt.Errorf("invalid api.WebhooksPer: %s", s)
	}
	*per = p
	return nil
}

// String returns the string representation of w.
// It panics if w is not a valid WebhooksPer value.
func (per WebhooksPer) String() string {
	s, err := per.Value()
	if err != nil {
		panic("invalid webhooksPer value")
	}
	return s.(string)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (per *WebhooksPer) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, null) {
		return nil
	}
	var s any
	err := json.Unmarshal(data, &s)
	if err != nil {
		return fmt.Errorf("json: cannot unmarshal into a apis.WebhooksPer value: %s", err)
	}
	return per.Scan(s)
}

// Value implements driver.Valuer interface.
// It returns an error if typ is not a valid ConnectionRole.
func (per WebhooksPer) Value() (driver.Value, error) {
	switch per {
	case WebhooksPerNone:
		return "None", nil
	case WebhooksPerConnector:
		return "Connector", nil
	case WebhooksPerResource:
		return "Resource", nil
	case WebhooksPerSource:
		return "Source", nil
	}
	return nil, fmt.Errorf("not a valid WebhooksPer: %d", per)
}

// Errors returned to and handled by the ServeWebhook method.
var (
	errBadRequest = errors.New("bad request")
	errNotFound   = errors.New("not found")
)

// ServeWebhook serves a webhook request. The request path starts with
// "/webhook/{connector}/" where {connector} is a connector identifier.
func (apis *APIs) ServeWebhook(w http.ResponseWriter, r *http.Request) {
	err := apis.receiveWebhook(r)
	if err != nil {
		switch err {
		case errBadRequest:
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		case errNotFound:
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		case _connector.ErrWebhookUnauthorized:
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		log.Printf("cannot serve webhook: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	return
}

var webhookPathReg = regexp.MustCompile(`^/webhook/([crs])/([^/]+)/`)

// receiveWebhook receives a webhook.
func (apis *APIs) receiveWebhook(r *http.Request) error {
	m := webhookPathReg.FindStringSubmatch(r.URL.Path)
	if m == nil {
		return errBadRequest
	}
	var connector *Connector
	var conf _connector.AppConfig
	switch m[1] {
	case "c":
		id, _ := strconv.Atoi(m[2])
		if id < 1 || id > maxInt32 {
			return errBadRequest
		}
		var err error
		connector, err = apis.Connectors.state.Get(id)
		if err != nil || connector.webhooksPer != WebhooksPerConnector {
			return errNotFound
		}
	case "r":
		id, _ := strconv.Atoi(m[2])
		if id < 1 || id > maxInt32 {
			return errBadRequest
		}
	Resource:
		for _, a := range apis.Accounts.state.List() {
			for _, ws := range a.Workspaces.state.List() {
				if r, ok := ws.resources.Get(id); ok {
					connector = r.connector
					conf.Resource = r.code
					break Resource
				}
			}
		}
		if connector == nil || connector.webhooksPer != WebhooksPerResource {
			return errNotFound
		}
	case "s":
		id, _ := strconv.Atoi(m[2])
		if id < 1 || id > maxInt32 {
			return errBadRequest
		}
		var connection *Connection
	Connection:
		for _, a := range apis.Accounts.state.List() {
			for _, ws := range a.Workspaces.state.List() {
				if c, err := ws.Connections.state.Get(id); err == nil {
					connection = c
					break Connection
				}
			}
		}
		if connection == nil || connection.resource == nil {
			return errNotFound
		}
		connector = connection.connector
		if connector.webhooksPer != WebhooksPerSource {
			return errNotFound
		}
		conf.Settings = connection.settings
		conf.Resource = connection.resource.code
		var err error
		conf.AccessToken, err = connection.resource.freshAccessToken()
		if err != nil {
			return err
		}
	}
	connection, err := _connector.RegisteredApp(connector.name).Connect(context.Background(), &conf)
	if err != nil {
		return err
	}
	events, err := connection.ReceiveWebhook(r)
	if err != nil {
		return err
	}
	// TODO(marco) store the events
	_ = events
	return nil
}
