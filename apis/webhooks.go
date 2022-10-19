//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package apis

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"chichi/connectors"
	"chichi/pkg/open2b/sql"
)

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
		case connectors.ErrWebhookUnauthorized:
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
	ctx := context.Background()
	var connector int
	var resource string
	var source int
	var webhookPer string
	switch m[1] {
	case "c":
		connector, _ = strconv.Atoi(m[2])
		if connector <= 0 {
			return errBadRequest
		}
		webhookPer = "Connector"
	case "r":
		c, r, ok := strings.Cut(m[2], "-")
		if !ok || r == "" {
			return errBadRequest
		}
		connector, _ = strconv.Atoi(c)
		if connector <= 0 {
			return errBadRequest
		}
		err := apis.myDB.QueryRow("SELECT `resource` FROM `resources` WHERE `connector` = ? AND `resource` = ?",
			c, r).Scan(&resource)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if r != resource {
			return errNotFound
		}
		ctx = context.WithValue(ctx, connectors.ResourceContextKey{}, resource)
		webhookPer = "Resource"
	case "s":
		source, _ = strconv.Atoi(m[2])
		if source <= 0 {
			return errBadRequest
		}
		var rawSettings []byte
		var accessToken, refreshToken string
		var expiration time.Time
		err := apis.myDB.QueryRow(
			"SELECT `s`.`connector`, `s`.`resource`, `s`.`settings`, `r`.`accessToken`, `r`.`refreshToken`, `r`.`accessTokenExpirationTimestamp`\n"+
				"FROM `data_sources` AS `s`\n"+
				"INNER JOIN `resources` AS `r` ON `r`.`connector` = `s`.`connector` AND `r`.`resource` = `s`.`resource`\n"+
				"WHERE `s`.`id` = ?", source).Scan(&connector, &resource, &rawSettings, &accessToken, &refreshToken, &expiration)
		if err != nil {
			if err == sql.ErrNoRows {
				return errNotFound
			}
			return err
		}
		accessTokenExpired := time.Now().UTC().Add(15 * time.Minute).After(expiration)
		if accessToken == "" || accessTokenExpired {
			accessToken, err = apis.refreshOAuthToken(connector, resource)
			if err != nil {
				return err
			}
		}
		settings := map[string]any{}
		if len(rawSettings) > 0 {
			err = json.Unmarshal(rawSettings, &settings)
			if err != nil {
				return errors.New("cannot unmarshal data source settings")
			}
		}
		ctx = context.WithValue(ctx, connectors.ResourceContextKey{}, resource)
		ctx = context.WithValue(ctx, connectors.AccessTokenContextKey{}, accessToken)
		ctx = context.WithValue(ctx, connectors.SettingsContextKey{}, settings)
		webhookPer = "DataSource"
	}
	conn, err := apis.Connector(connector)
	if err != nil {
		return err
	}
	if conn == nil {
		return errNotFound
	}
	if conn.WebhooksPer != webhookPer {
		return errBadRequest
	}
	c := connectors.Connector(conn.Name, conn.ClientSecret)
	events, err := c.ReceiveWebhook(ctx, r)
	if err != nil {
		return err
	}
	// TODO(marco) store the events
	_ = events
	return nil
}
