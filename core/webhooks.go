// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"fmt"

	"github.com/meergo/meergo/tools/json"
)

// WebhooksPer values indicates if webhooks are per account, connection, or
// connector.
type WebhooksPer int

const (
	WebhooksPerNone WebhooksPer = iota
	WebhooksPerAccount
	WebhooksPerConnection
	WebhooksPerConnector
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
	case WebhooksPerAccount:
		return "Account"
	case WebhooksPerConnection:
		return "Connection"
	case WebhooksPerConnector:
		return "Connector"
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
		return fmt.Errorf("json: cannot scan a %T value into an core.WebhooksPer value", v)
	}
	var p WebhooksPer
	switch s {
	case "None":
		p = WebhooksPerNone
	case "Account":
		p = WebhooksPerAccount
	case "Connection":
		p = WebhooksPerConnection
	case "Connector":
		p = WebhooksPerConnector
	default:
		return fmt.Errorf("json: invalid core.WebhooksPer: %s", s)
	}
	*per = p
	return nil
}

// Errors returned to and handled by the ServeWebhook method.
//var (
//	errBadRequest = errors.New("bad request")
//	errNotFound   = errors.New("not found")
//)

// TODO(marco): implement webhooks
//// ServeWebhook serves a webhook request. The request path starts with
//// "/webhook/{connector}/" where {connector} is a connector identifier.
//func (core *Core) ServeWebhook(w http.ResponseWriter, r *http.Request) {
//	core.mustBeOpen()
//	err := core.receiveWebhook(r)
//	if err != nil {
//		switch err {
//		case errNotFound:
//			http.Error(w, "Not Found", http.StatusNotFound)
//			return
//		case connectors.ErrNoWebhooks:
//			http.Error(w, "Not Found", http.StatusNotFound)
//			return
//		case connectors.ErrWebhookUnauthorized:
//			http.Error(w, "Unauthorized", http.StatusUnauthorized)
//			return
//		}
//		slog.Error("core: cannot serve webhook", "err", err)
//		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
//		return
//	}
//}

// TODO(marco): implement webhooks
//var webhookPathReg = regexp.MustCompile(`^/webhook/([scr])/([^/]+)/`)
//
//// receiveWebhook receives a webhook.
//func (core *Core) receiveWebhook(req *http.Request) error {
//	m := webhookPathReg.FindStringSubmatch(req.URL.Path)
//	if m == nil {
//		return errNotFound
//	}
//	var events []connectors.WebhookPayload
//	var err error
//	switch m[1] {
//	case "a":
//		id, _ := strconv.Atoi(m[2])
//		if id < 1 || id > maxInt32 {
//			return errBadRequest
//		}
//		account, ok := core.state.Account(id)
//		if !ok {
//			return errNotFound
//		}
//		events, err = core.connections.ReceivePerAccountWebhook(account, req)
//	case "s":
//		id, _ := strconv.Atoi(m[2])
//		if id < 1 || id > maxInt32 {
//			return errBadRequest
//		}
//		connection, ok := core.state.Connection(id)
//		if !ok {
//			return errNotFound
//		}
//		events, err = core.connections.ReceivePerConnectionWebhook(connection, req)
//	case "c":
//		name := url.PathEscape(m[2])
//		connector, ok := core.state.Connector(name)
//		if !ok || connector.WebhooksPer != state.WebhooksPerConnector {
//			return errNotFound
//		}
//		events, err = core.connections.ReceivePerConnectorWebhook(connector, req)
//	}
//	if err != nil {
//		return err
//	}
//	// TODO(marco) store the events
//	_ = events
//	return nil
//}
