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
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"chichi/apis/state"
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

// String returns the string representation of w.
// It panics if w is not a valid WebhooksPer value.
func (per WebhooksPer) String() string {
	switch per {
	case WebhooksPerNone:
		return "None"
	case WebhooksPerConnector:
		return "Connector"
	case WebhooksPerResource:
		return "Resource"
	case WebhooksPerSource:
		return "Source"
	}
	panic("invalid webhooksPer value")
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (per *WebhooksPer) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, null) {
		return nil
	}
	var v any
	err := json.Unmarshal(data, &v)
	if err != nil {
		return fmt.Errorf("json: cannot unmarshal into a apis.WebhooksPer value: %s", err)
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an api.WebhooksPer value", v)
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
		return fmt.Errorf("invalid state.WebhooksPer: %s", s)
	}
	*per = p
	return nil
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
		log.Printf("[error] cannot serve webhook: %s", err)
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
	var connector *state.Connector
	var conf _connector.AppConfig
	switch m[1] {
	case "c":
		id, _ := strconv.Atoi(m[2])
		if id < 1 || id > maxInt32 {
			return errBadRequest
		}
		var ok bool
		connector, ok = apis.state.Connector(id)
		if !ok || connector.WebhooksPer != state.WebhooksPerConnector {
			return errNotFound
		}
	case "r":
		id, _ := strconv.Atoi(m[2])
		if id < 1 || id > maxInt32 {
			return errBadRequest
		}
	Resource:
		for _, a := range apis.state.Accounts() {
			for _, ws := range a.Workspaces() {
				if r, ok := ws.Resource(id); ok {
					connector = r.Connector()
					conf.Resource = r.Code
					break Resource
				}
			}
		}
		if connector == nil || connector.WebhooksPer != state.WebhooksPerResource {
			return errNotFound
		}
	case "s":
		id, _ := strconv.Atoi(m[2])
		if id < 1 || id > maxInt32 {
			return errBadRequest
		}
		var connection *state.Connection
	Connection:
		for _, a := range apis.state.Accounts() {
			for _, ws := range a.Workspaces() {
				if c, ok := ws.Connection(id); ok {
					connection = c
					break Connection
				}
			}
		}
		if connection == nil {
			return errNotFound
		}
		resource, ok := connection.Resource()
		if !ok {
			return errNotFound
		}
		connector = connection.Connector()
		if connector.WebhooksPer != state.WebhooksPerSource {
			return errNotFound
		}
		conf.Settings = connection.Settings
		conf.Resource = resource.Code
		var err error
		conf.AccessToken, err = freshAccessToken(apis.db, resource)
		if err != nil {
			return err
		}
	}
	connection, err := _connector.RegisteredApp(connector.Name).Open(context.Background(), &conf)
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
