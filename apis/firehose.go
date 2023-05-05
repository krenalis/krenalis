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
	"log"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/mappings"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/apis/types"
	"chichi/connector"
)

// Make sure it implements the Firehose interface.
var _ connector.Firehose = &firehose{}

const noColumn = -1
const maxSettingsLen = 10_000 // Maximum length of settings in runes.

// firehose is the Firehose API used by the connectors.
type firehose struct {
	db          *postgres.DB
	connection  *state.Connection
	action      *state.Action
	resource    int
	ctx         context.Context
	cancel      context.CancelFunc
	webhooksPer WebhooksPer
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

	result, err := fh.db.Exec(fh.ctx, "UPDATE connections SET user_cursor = $1 WHERE id = $2", cursor, fh.connection.ID)
	if err != nil {
		fh.setError(err)
		return
	}
	if result.RowsAffected() == 0 {
		fh.cancel()
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

func (fh *firehose) SetUser(user string, properties map[string]any, timestamp time.Time, timestamps map[string]time.Time) {
	// Normalize the properties.
	propertyOf := map[string]types.Property{}
	for _, p := range fh.action.Schema.Properties() {
		propertyOf[p.Name] = p
	}
	for name, value := range properties {
		p, ok := propertyOf[name]
		if !ok {
			fh.setError(fmt.Errorf("connector %d has returned an unknown property %q", fh.connection.ID, name))
			return
		}
		value, err := normalizeAppPropertyValue(name, p.Nullable, p.Type, value)
		if err != nil {
			fh.setError(err)
			return
		}
		properties[name] = value
	}
	fh.setUser(user, properties, timestamp, timestamps)
}

func (fh *firehose) setUser(user string, properties map[string]any, timestamp time.Time, timestamps map[string]time.Time) {

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

	// Normalize the properties.
	propertyOf := map[string]types.Property{}
	for _, p := range fh.action.Schema.Properties() {
		propertyOf[p.Name] = p
	}
	for name, value := range properties {
		p, ok := propertyOf[name]
		if !ok {
			fh.setError(fmt.Errorf("connector %d has returned an unknown property %q", fh.connection.ID, name))
			return
		}
		value, err := normalizeAppPropertyValue(name, p.Nullable, p.Type, value)
		if err != nil {
			fh.setError(err)
			return
		}
		properties[name] = value
	}

	// Write the user properties to the database.
	if timestamps == nil {
		timestamps = map[string]time.Time{}
	}
	for name := range properties {
		if _, ok := timestamps[name]; !ok {
			timestamps[name] = timestamp
		}
	}
	err := fh.writeConnectionUsers(user, properties, timestamps)
	if err != nil {
		fh.setError(err)
		return
	}

	// Apply the mapping (or the transformation).
	ctx := context.Background()
	candidateData, err := mappings.Apply(ctx, fh.action, properties, types.Type{})
	if err != nil {
		fh.setError(actionExecutionError{fmt.Errorf("cannot apply mapping or transformation: %s", err)})
		return
	}

	// Resolve the entity of this user.
	ids := identitySolver{fh}
	email, _ := candidateData["Email"].(string)
	if email == "" {
		fh.setError(actionExecutionError{fmt.Errorf("expecting 'Email' to be a non-empty string, got %#v (of type %T)", candidateData["Email"], candidateData["Email"])})
		return
	}
	goldenRecordID, err := ids.ResolveEntity(fh.connection.ID, user, email)
	if err != nil {
		fh.setError(err)
		return
	}

	// Write the data to the Golden Record, if necessary.
	if len(candidateData) > 0 {
		err = fh.writeToGoldenRecord(goldenRecordID, candidateData)
		if err != nil {
			fh.setError(err)
			return
		}
		log.Printf("[info] properties for user %q written to the Golden Record", candidateData["Email"])
	}

}

type connectionEntityData struct {
	Data       map[string]any
	Timestamps map[string]time.Time
}

// entityData returns the data associated to the entity from the given
// connection.
func (fh *firehose) entityData(connection int, user string) (connectionEntityData, error) {
	var entityData connectionEntityData
	ws := fh.connection.Workspace()
	row := ws.Warehouse.QueryRow(fh.ctx,
		"SELECT data, timestamps FROM connections_users WHERE connection = $1 AND user = $2",
		connection, user)
	var rawData []byte
	var rawTimestamps []byte
	err := row.Scan(&rawData, &rawTimestamps)
	if err != nil {
		return connectionEntityData{}, err
	}
	err = json.Unmarshal(rawData, &entityData.Data)
	if err != nil {
		return connectionEntityData{}, err
	}
	err = json.Unmarshal(rawTimestamps, &entityData.Timestamps)
	if err != nil {
		return connectionEntityData{}, err
	}
	return entityData, nil
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

// writeConnectionUsers writes the given connection users to the database.
func (fh *firehose) writeConnectionUsers(user string, props map[string]any, timestamps map[string]time.Time) error {
	data, err := json.Marshal(props)
	if err != nil {
		return err
	}
	jsonTimestamps, err := json.Marshal(timestamps)
	if err != nil {
		return err
	}
	c := fh.connection
	ws := c.Workspace()
	_, err = ws.Warehouse.Exec(fh.ctx, "INSERT INTO connections_users (connection, \"user\", data, timestamps)\n"+
		"VALUES ($1, $2, $3, $4)\n"+
		"ON CONFLICT (connection, \"user\") DO UPDATE SET data = $3, timestamps = $4",
		c.ID, user, data, jsonTimestamps)
	if err != nil {
		return err
	}
	_, err = fh.db.Exec(fh.ctx, "INSERT INTO connections_stats AS cs (connection, time_slot, users)\n"+
		"VALUES ($1, $2, 1)\n"+
		"ON CONFLICT (connection, time_slot) DO UPDATE SET users = cs.users + 1",
		c.ID, statsTimeSlot(time.Now()))
	return err
}

// writeToGoldenRecord writes the given properties to the Golden Record.
func (fh *firehose) writeToGoldenRecord(id int, props map[string]any) error {

	// TODO(Gianluca):
	for _, v := range props {
		if _, ok := v.(map[string]interface{}); ok {
			return errors.New("writeToGoldenRecord is still partially implemented and does not support objects")
		}
	}

	query := &strings.Builder{}
	query.WriteString("UPDATE users SET\n")
	var values []any
	i := 1
	for prop, value := range props {
		if i > 1 {
			query.WriteString(", ")
		}
		query.WriteString(postgres.QuoteIdent(prop))
		query.WriteString(" = $")
		query.WriteString(strconv.Itoa(i))
		values = append(values, value)
		i++
	}
	query.WriteString(`, "updateTime" = now()`)
	query.WriteString("\nWHERE id = $")
	query.WriteString(strconv.Itoa(i))
	values = append(values, id)
	ws := fh.connection.Workspace()
	res, err := ws.Warehouse.Exec(fh.ctx, query.String(), values...)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("BUG: one row should be affected, got %d", affected)
	}
	return nil
}
