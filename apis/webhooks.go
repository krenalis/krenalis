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
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

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
	var connector int
	var connection int
	var webhooksPer WebhooksPer
	var conf _connector.AppConfig
	switch m[1] {
	case "c":
		webhooksPer = WebhooksPerConnector
		connector, _ = strconv.Atoi(m[2])
		if connector <= 0 {
			return errBadRequest
		}
	case "r":
		webhooksPer = WebhooksPerResource
		r, _ := strconv.Atoi(m[2])
		if r <= 0 {
			return errBadRequest
		}
		err := apis.db.QueryRow("SELECT connector, code FROM resources WHERE id = $1", r).
			Scan(&connector, &conf.Resource)
		if err != nil {
			if err == sql.ErrNoRows {
				return errNotFound
			}
			return err
		}
	case "s":
		webhooksPer = WebhooksPerSource
		connection, _ = strconv.Atoi(m[2])
		if connection <= 0 {
			return errBadRequest
		}
		var resource int
		var hasOAuth bool
		var refreshToken string
		var expiresIn time.Time
		err := apis.db.QueryRow(
			"SELECT s.connector, s.resource, s.settings, c.oauth_client_secret <> '' AS has_oauth, r.code,"+
				" r.oauth_access_token, r.oauth_refresh_token, r.oauth_expires_in\n"+
				"FROM connections AS s\n"+
				"INNER JOIN connectors AS c ON c.id = s.connector\n"+
				"INNER JOIN resources AS r ON r.id = s.resource\n"+
				"WHERE s.id = $1", connection).
			Scan(&connector, &resource, &conf.Settings, hasOAuth, &conf.Resource, &conf.AccessToken, &refreshToken, &expiresIn)
		if err != nil {
			if err == sql.ErrNoRows {
				return errNotFound
			}
			return err
		}
		if hasOAuth {
			accessTokenExpired := time.Now().UTC().Add(15 * time.Minute).After(expiresIn)
			if conf.AccessToken == "" || accessTokenExpired {
				res, err := apis.Connectors.refreshOAuthToken(connector, resource)
				if err != nil {
					return err
				}
				conf.AccessToken = res.oAuthAccessToken
			}
		}
	}
	conn, err := apis.Connectors.get(connector)
	if err != nil {
		return errNotFound
	}
	if conn.webhooksPer != webhooksPer {
		return errBadRequest
	}
	c, err := _connector.RegisteredApp(conn.name).Connect(context.Background(), &conf)
	if err != nil {
		return err
	}
	events, err := c.ReceiveWebhook(r)
	if err != nil {
		return err
	}
	// TODO(marco) store the events
	_ = events
	return nil
}
