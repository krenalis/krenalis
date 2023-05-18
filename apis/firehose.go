//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"
	"unicode/utf8"

	"chichi/apis/mappings"
	"chichi/apis/normalization"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/connector"
	"chichi/connector/types"
)

// Make sure it implements the Firehose interface.
var _ connector.Firehose = &firehose{}

const maxSettingsLen = 10_000 // Maximum length of settings in runes.

// firehose is the Firehose API used by the connectors.
type firehose struct {
	db          *postgres.DB
	connection  *state.Connection
	action      *Action
	resource    int
	ctx         context.Context
	cancel      context.CancelFunc
	webhooksPer WebhooksPer
	mapping     *mappings.Mapping
	err         error
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

// SetCursor sets the user cursor.
func (fh *firehose) SetCursor(cursor string) {

	// Return if the context has expired.
	select {
	case <-fh.ctx.Done():
		return
	default:
	}

	// Set the user cursor of the action.
	err := fh.action.setUserCursor(fh.ctx, cursor)
	if err != nil {
		fh.setError(err)
		return
	}

}

func (fh *firehose) SetGroup(group string, properties map[string]any, timestamp time.Time, timestamps map[string]time.Time) {

	// Return if the context has expired.
	select {
	case <-fh.ctx.Done():
		return
	default:
	}

	// TODO.

}

func (fh *firehose) SetGroupUsers(group string, users []string) {

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

func (fh *firehose) SetUser(id string, user map[string]any, timestamp time.Time, timestamps map[string]time.Time) {

	// Return if the context has expired.
	select {
	case <-fh.ctx.Done():
		return
	default:
	}

	ws := fh.connection.Workspace()
	if ws.Warehouse == nil {
		fh.err = fmt.Errorf("workspace %d does not have a warehouse", ws.ID)
		return
	}

	// Importing users from a destination to match identities for the export.
	if fh.connection.Role == state.DestinationRole {
		externalPropName := fh.action.action.ExportMatchingProperties.External
		externalProp, ok := user[externalPropName]
		if !ok {
			// TODO(Gianluca): handle this error properly.
			fh.setError(fmt.Errorf("user does not contain property %q", externalPropName))
			return
		}
		p, err := json.Marshal(externalProp)
		if err != nil {
			fh.setError(err)
			return
		}
		err = ws.Warehouse.SetDestinationUser(fh.ctx, fh.connection.ID, id, string(p))
		if err != nil {
			fh.setError(err)
			return
		}
		return
	}

	// Normalize the user properties.
	propertyOf := map[string]types.Property{}
	for _, p := range fh.action.action.Schema.Properties() {
		propertyOf[p.Name] = p
	}
	for name, value := range user {
		p, ok := propertyOf[name]
		if !ok {
			fh.setError(fmt.Errorf("connector %d has returned an unknown property %q", fh.connection.ID, name))
			return
		}
		value, err := normalization.NormalizeAppProperty(name, p.Nullable, p.Type, value)
		if err != nil {
			fh.setError(err)
			return
		}
		user[name] = value
	}

	mappedUser, err := fh.mapping.Apply(fh.ctx, user)
	if err != nil {
		fh.setError(err)
		return
	}
	connection := &Connection{
		db:         fh.db,
		connection: fh.connection,
	}
	err = connection.writeConnectionUsers(fh.ctx, id, user, timestamp, timestamps)
	if err != nil {
		fh.setError(err)
		return
	}
	err = connection.setUser(fh.ctx, id, mappedUser)
	if err != nil {
		fh.setError(err)
		return
	}

}

func (fh *firehose) SetUserGroups(user string, groups []string) {

	// Return if the context has expired.
	select {
	case <-fh.ctx.Done():
		return
	default:
	}

	// TODO.

}

// WebhookURL returns the URL of the webhook.
// If the connector does not support webhooks, it returns an empty string.
func (fh *firehose) WebhookURL() string {
	c := fh.connection
	u := "https://localhost:9090/webhook/"
	switch fh.webhooksPer {
	case WebhooksPerNone:
		return ""
	case WebhooksPerConnector:
		return u + "c/" + strconv.Itoa(c.Connector().ID) + "/"
	case WebhooksPerResource:
		return u + "r/" + strconv.Itoa(fh.resource) + "/"
	case WebhooksPerSource:
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

type recordReader struct {
	columns []types.Property
	users   [][]any
	cursor  int
}

func (fh *firehose) newRecordReader(columns []types.Property, users [][]any) *recordReader {
	return &recordReader{
		columns: columns,
		users:   users,
	}
}

// Columns returns the columns of the records.
func (rr *recordReader) Columns() []types.Property {
	return rr.columns
}

// Record returns the next record as a slice of any. It returns nil and io.EOF
// if there are no more records.
func (rr *recordReader) Record() ([]any, error) {
	if rr.cursor >= len(rr.users) {
		return nil, io.EOF
	}
	user := rr.users[rr.cursor]
	rr.cursor++
	return user, nil
}

// RecordString returns the next record as a string slice. It returns nil and
// io.EOF if there are no more records.
func (rr *recordReader) RecordString() ([]string, error) {
	if rr.cursor >= len(rr.users) {
		return nil, io.EOF
	}
	u := rr.users[rr.cursor]
	users := make([]string, len(u))
	for i, prop := range u {
		users[i] = fmt.Sprintf("%v", prop)
	}
	rr.cursor++
	return users, nil
}
