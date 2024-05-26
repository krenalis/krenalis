//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/open2b/chichi/apis/connectors"
	"github.com/open2b/chichi/apis/state"
)

// WebhooksPer values indicates if webhooks are per connection, connector, or
// resource.
type WebhooksPer int

const (
	WebhooksPerNone WebhooksPer = iota
	WebhooksPerConnection
	WebhooksPerConnector
	WebhooksPerResource
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
	case WebhooksPerConnection:
		return "Connection"
	case WebhooksPerConnector:
		return "Connector"
	case WebhooksPerResource:
		return "Resource"
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
		return err
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("json: cannot scan a %T value into an api.WebhooksPer value", v)
	}
	var p WebhooksPer
	switch s {
	case "None":
		p = WebhooksPerNone
	case "Connection":
		p = WebhooksPerConnection
	case "Connector":
		p = WebhooksPerConnector
	case "Resource":
		p = WebhooksPerResource
	default:
		return fmt.Errorf("json: invalid state.WebhooksPer: %s", s)
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
	apis.mustBeOpen()
	err := apis.receiveWebhook(r)
	if err != nil {
		switch err {
		case errNotFound:
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		case connectors.ErrNoWebhooks:
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		case connectors.ErrWebhookUnauthorized:
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		slog.Error("cannot serve webhook", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

var webhookPathReg = regexp.MustCompile(`^/webhook/([scr])/([^/]+)/`)

// receiveWebhook receives a webhook.
func (apis *APIs) receiveWebhook(req *http.Request) error {
	m := webhookPathReg.FindStringSubmatch(req.URL.Path)
	if m == nil {
		return errNotFound
	}
	var events []connectors.WebhookPayload
	var err error
	switch m[1] {
	case "s":
		id, _ := strconv.Atoi(m[2])
		if id < 1 || id > maxInt32 {
			return errBadRequest
		}
		connection, ok := apis.state.Connection(id)
		if !ok {
			return errNotFound
		}
		events, err = apis.connectors.ReceivePerConnectionWebhook(connection, req)
	case "c":
		name := url.PathEscape(m[2])
		connector, ok := apis.state.Connector(name)
		if !ok || connector.WebhooksPer != state.WebhooksPerConnector {
			return errNotFound
		}
		events, err = apis.connectors.ReceivePerConnectorWebhook(connector, req)
	case "r":
		id, _ := strconv.Atoi(m[2])
		if id < 1 || id > maxInt32 {
			return errBadRequest
		}
		resource, ok := apis.state.Resource(id)
		if !ok {
			return errNotFound
		}
		events, err = apis.connectors.ReceivePerResourceWebhook(resource, req)
	}
	if err != nil {
		return err
	}
	// TODO(marco) store the events
	_ = events
	return nil
}
