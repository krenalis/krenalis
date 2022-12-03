//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	_connector "chichi/connector"
	"chichi/pkg/open2b/sql"
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

// String returns the string representation of w.
// It panics if w is not a valid WebhooksPer value.
func (w WebhooksPer) String() string {
	switch w {
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

// MarshalJSON implements the json.Marshaler interface.
// It panics if w is not a valid WebhooksPer value.
func (w WebhooksPer) MarshalJSON() ([]byte, error) {
	return []byte(`"` + w.String() + `"`), nil
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
		connector, _ = strconv.Atoi(m[2])
		if connector <= 0 {
			return errBadRequest
		}
		webhooksPer = WebhooksPerConnector
	case "r":
		r, _ := strconv.Atoi(m[2])
		if r <= 0 {
			return errBadRequest
		}
		err := apis.myDB.QueryRow("SELECT `connector`, `code` FROM `resources` WHERE `id` = ?", r).
			Scan(&connector, &conf.Resource)
		if err != nil {
			if err == sql.ErrNoRows {
				return errNotFound
			}
			return err
		}
		webhooksPer = WebhooksPerResource
	case "s":
		connection, _ = strconv.Atoi(m[2])
		if connection <= 0 {
			return errBadRequest
		}
		var resource int
		var refreshToken string
		var expiresIn time.Time
		err := apis.myDB.QueryRow(
			"SELECT `s`.`connector`, `s`.`resource`, `s`.`settings`, `r`.`code`, `r`.`oAuthAccessToken`,"+
				" `r`.`oAuthRefreshToken`, `r`.`oAuthExpiresIn`\n"+
				"FROM `connections` AS `s`\n"+
				"INNER JOIN `resources` AS `r` ON `r`.`id` = `s`.`resource`\n"+
				"WHERE `s`.`id` = ?", connection).
			Scan(&connector, &resource, &conf.Settings, &conf.Resource, &conf.AccessToken, &refreshToken, &expiresIn)
		if err != nil {
			if err == sql.ErrNoRows {
				return errNotFound
			}
			return err
		}
		accessTokenExpired := time.Now().UTC().Add(15 * time.Minute).After(expiresIn)
		if conf.AccessToken == "" || accessTokenExpired {
			conf.AccessToken, err = apis.refreshOAuthToken(resource)
			if err != nil {
				return err
			}
		}
		webhooksPer = WebhooksPerSource
	}
	conn, err := apis.Connector(connector)
	if err != nil {
		return err
	}
	if conn == nil {
		return errNotFound
	}
	if conn.WebhooksPer != webhooksPer {
		return errBadRequest
	}
	c, err := _connector.RegisteredApp(conn.Name).Connect(context.Background(), &conf)
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
