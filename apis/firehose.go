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
	"fmt"
	"log"
	"strconv"
	"time"
	"unicode/utf8"

	"chichi/apis/mappings"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/connector"
)

// Make sure it implements the Firehose interface.
var _ connector.Firehose = &firehose{}

const maxSettingsLen = 10_000 // Maximum length of settings in runes.

// firehose is the Firehose API used by the connectors.
type firehose struct {
	db         *postgres.DB
	connection *state.Connection
	action     *Action
	resource   int
	ctx        context.Context
	cancel     context.CancelFunc
	mapping    *mappings.Mapping
	err        error
}

func (fh *firehose) ReceiveEvent(event connector.WebhookEvent) {

	// Return if the context has expired.
	select {
	case <-fh.ctx.Done():
		return
	default:
	}

	// TODO.

}

// SetSettings sets the given settings of the connection.
func (fh *firehose) SetSettings(settings []byte) error {

	// Return if the context has expired.
	select {
	case <-fh.ctx.Done():
		return fh.ctx.Err()
	default:
	}

	if !utf8.Valid(settings) {
		return errors.New("settings is not valid UTF-8")
	}
	if utf8.RuneCount(settings) > maxSettingsLen {
		return fmt.Errorf("settings is longer than %d runes", maxSettingsLen)
	}
	n := state.SetConnectionSettingsNotification{
		Connection: fh.connection.ID,
		Settings:   settings,
	}
	err := fh.db.Transaction(fh.ctx, func(tx *postgres.Tx) error {
		_, err := tx.Exec(fh.ctx, "UPDATE connections SET settings = $1 WHERE id = $2", n.Settings, n.Connection)
		if err != nil {
			return err
		}
		return tx.Notify(fh.ctx, n)
	})
	if err != nil {
		log.Printf("[error] %s", err)
		return errors.New("cannot set settings")
	}
	return nil
}

// WebhookURL returns the URL of the webhook.
// If the connector does not support webhooks, it returns an empty string.
func (fh *firehose) WebhookURL() string {
	c := fh.connection
	conn := c.Connector()
	u := "https://localhost:9090/webhook/"
	switch conn.WebhooksPer {
	case state.WebhooksPerNone:
		return ""
	case state.WebhooksPerConnector:
		return u + "c/" + strconv.Itoa(conn.ID) + "/"
	case state.WebhooksPerResource:
		return u + "r/" + strconv.Itoa(fh.resource) + "/"
	case state.WebhooksPerSource:
		return u + "s/" + strconv.Itoa(c.ID) + "/"
	}
	panic("unexpected webhooksPer value")
}

// setError sets fh.err and cancels the context.
func (fh *firehose) setError(err error) {
	fh.err = err
	fh.cancel()
}

// statsTimeSlot returns the stats time slot for the time t.
// t must be a UTC time.
func statsTimeSlot(t time.Time) int {
	epoc := int(t.Unix())
	return epoc / (60 * 60)
}
