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

	"chichi/apis/transformations"
	"chichi/connector"
)

// Make sure it implements the Firehose interface.
var _ connector.Firehose = &firehose{}

const noColumn = -1
const maxSettingsLen = 10_000 // Maximum length of settings in runes.

// firehose is the Firehose API used by the connectors.
type firehose struct {
	connections   *Connections
	connection    int
	resource      int
	connector     int
	connectorType ConnectorType
	role          connector.Role
	ctx           context.Context
	cancel        context.CancelFunc
	webhooksPer   string
	err           error
}

func (fh *firehose) ReceiveEvent(event connector.Event) {
	return
}

// SetCursor sets the user cursor.
func (fh *firehose) SetCursor(cursor string) {
	result, err := fh.connections.myDB.Exec("UPDATE `connections`\nSET `userCursor` = ?\nWHERE `id` = ?", cursor, fh.connection)
	if err != nil {
		fh.setError(err)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		fh.setError(err)
		return
	}
	if affected == 0 {
		fh.cancel()
	}
	return
}

func (fh *firehose) SetGroup(group string, timestamp time.Time, properties map[string]any) {
	return
}

func (fh *firehose) SetGroupUsers(group string, users []string) {
	return
}

// SetSettings sets the given settings of the connection.
func (fh *firehose) SetSettings(settings []byte) error {
	if !utf8.Valid(settings) {
		return errors.New("settings is not valid UTF-8")
	}
	if utf8.RuneCount(settings) > maxSettingsLen {
		return fmt.Errorf("settings is longer than %d runes", maxSettingsLen)
	}
	_, err := fh.connections.myDB.Exec("UPDATE `connections` SET `settings` = ? WHERE `id` = ?", settings, fh.connection)
	if err != nil {
		log.Printf("[error] %s", err)
		return errors.New("cannot set settings")
	}
	return nil
}

func (fh *firehose) SetUser(user string, timestamp time.Time, properties map[string]any) {

	// Normalize the properties and the timestamps.
	timestamps := make(map[string]time.Time, len(properties))
	{
		props := make(map[string]any, len(properties))
		for name, v := range properties {
			if tv, ok := v.(connector.TimestampedValue); ok {
				props[name] = tv.Value
				timestamps[name] = tv.Timestamp
			} else {
				props[name] = v
				timestamps[name] = timestamp
			}
		}
		properties = props
	}

	// Serialize the properties and the timestamps to the database.
	err := fh.writeConnectionUsers(user, properties, timestamps)
	if err != nil {
		fh.setError(err)
		return
	}

	// Retrieve the transformations for this connection.
	connectionsTransformations, err := fh.connections.Transformations.List(fh.connection)
	if err != nil {
		fh.setError(fmt.Errorf("cannot list transformations for %d: %s", fh.connection, err))
		return
	}

	// Create a pool of transformation VMs.
	pool := transformations.NewPool()

	// Applying the transformations, calculate the Golden Record properties and
	// their relative timestamps for this user in this connection.
	candidateData := map[string]any{}
	candidateTimestamps := map[string]time.Time{}
	for _, t := range connectionsTransformations {
		props := map[string]any{}
		for _, ip := range t.InputProperties {
			props[ip.Name] = properties[ip.Name]
		}

		// Apply the transformation function.
		grProp, err := pool.Run(context.Background(), t.SourceCode, props)
		if err != nil {
			fh.setError(fmt.Errorf("error while calling transformation function %d: %s", t.ID, err))
			return
		}
		if grProp != nil {
			candidateData[t.GRProperty] = grProp
			candidateTimestamps[t.GRProperty] = mostRecentTimestamp(timestamps, t.InputProperties)
		}
	}

	email, _ := candidateData["Email"].(string)
	if email == "" {
		fh.setError(fmt.Errorf("expecting 'Email' to be a non-empty string, got %#v (of type %T)", candidateData["Email"], candidateData["Email"]))
		return
	}

	ids := identitySolver{fh}

	// Resolve the entity of this user.
	goldenRecordID, err := ids.ResolveEntity(fh.connection, user, email)
	if err != nil {
		fh.setError(err)
		return
	}

	// Retrieve the entities which are the same user.
	sameEntities, err := ids.LookupSameEntities(fh.connection, user)
	if err != nil {
		fh.setError(fmt.Errorf("cannot lookup same entities for user %q: %s", user, err))
		return
	}

	// Retrieve the transformation functions for the entities that match with
	// the current user.
	otherTransformations, err := fh.listTransformations(keys(sameEntities))
	if err != nil {
		fh.setError(fmt.Errorf("cannot retrieve transformations for other entities: %s", err))
		return
	}

	// Discard any incoming Golden Record property which is older than the
	// existent properties.
transfLoop:
	for _, t := range otherTransformations {
		// For the connection of this transformation, determine the timestamps
		// relative to the users which refers to the same identity.
		for _, u := range sameEntities[t.Connection] {
			entityData, err := fh.entityData(t.Connection, u)
			if err != nil {
				fh.setError(err)
				return
			}
			ts := mostRecentTimestamp(entityData.Timestamps, t.InputProperties)
			if ts.After(candidateTimestamps[t.GRProperty]) {
				// Don't update this Golden Record property.
				delete(candidateData, t.GRProperty)
				if len(candidateData) == 0 {
					// Avoid useless iterations.
					break transfLoop
				}
			}
		}
	}

	// Write the data to the Golden Record.
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
	row := fh.connections.myDB.QueryRow(
		"SELECT `data`, `timestamps` FROM `connections_users` WHERE `connection` = ? AND `user` = ?",
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
	return
}

// WebhookURL returns the URL of the webhook.
// If the connector does not support webhooks, it returns an empty string.
func (fh *firehose) WebhookURL() string {
	u := "https://localhost:9090/webhook/"
	switch fh.webhooksPer {
	case "None":
		return ""
	case "Connector":
		return u + "c/" + strconv.Itoa(fh.connector) + "/"
	case "Resource":
		return u + "r/" + strconv.Itoa(fh.resource) + "/"
	case "Source":
		return u + "s/" + strconv.Itoa(fh.connection) + "/"
	}
	panic("unexpected webhookPer value")
}

// setError sets fh.err and cancels the context.
func (fh *firehose) setError(err error) {
	fh.err = err
	fh.cancel()
	log.Printf("[error] firehose error: %s", err)
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
	_, err = fh.connections.myDB.Exec("INSERT INTO `connections_users`\n"+
		"SET `connection` = ?, `user` = ?, `data` = ?, `timestamps` = ?\n"+
		"ON DUPLICATE KEY UPDATE `data` = ?, `timestamps` = ?",
		fh.connection, user, data, jsonTimestamps, data, jsonTimestamps)
	if err != nil {
		return err
	}
	_, err = fh.connections.myDB.Exec("INSERT INTO `connections_stats`\n"+
		"SET `connection` = ?, `timeSlot` = ?, `usersIn` = 1\n"+
		"ON DUPLICATE KEY UPDATE `usersIn` = `usersIn` + 1",
		fh.connection, statsTimeSlot(time.Now()))
	return err
}

// writeToGoldenRecord writes the given properties to the Golden Record.
func (fh *firehose) writeToGoldenRecord(id int, props map[string]any) error {

	query := &strings.Builder{}
	query.WriteString("UPDATE `warehouse_users` SET\n")
	var values []any
	i := 0
	for prop, value := range props {
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString("`" + prop + "` = ?\n")
		values = append(values, value)
		i++
	}
	query.WriteString("\nWHERE `id` = ?")
	values = append(values, id)
	_, err := fh.connections.myDB.Exec(query.String(), values...)
	if err != nil {
		return fmt.Errorf("cannot write data Golden Record: %s", err)
	}
	return nil
}

// newRecordWriter returns a new record writer.
func (fh *firehose) newRecordWriter(identityColumn, timestampColumn string, onlyColumns bool) *recordWriter {
	return &recordWriter{
		fh:              fh,
		onlyColumns:     onlyColumns,
		identityColumn:  identityColumn,
		timestampColumn: timestampColumn,
	}
}

// recordWriter implements the connector.RecordWriter interface.
type recordWriter struct {
	fh              *firehose
	onlyColumns     bool
	columns         []connector.Column
	identityColumn  string
	timestampColumn string
	identityIndex   int
	timestampIndex  int
	timestamp       time.Time
	setUserCalled   bool
}

// Columns sets the columns of the records.
// Columns must be called before Record, RecordMap and RecordString.
func (rw *recordWriter) Columns(columns []connector.Column) error {
	if len(columns) == 0 {
		return connector.ErrNoColumns
	}
	index := make(map[string]int, len(columns))
	for i, c := range columns {
		if c.Name == "" {
			return connector.ErrEmptyColumnName
		}
		if !utf8.ValidString(c.Name) {
			return connector.ErrInvalidEncodedColumnName
		}
		if _, ok := index[c.Name]; ok {
			return connector.SameColumnNameError{Name: c.Name}
		}
		index[c.Name] = i
		if !c.Type.Valid() {
			return fmt.Errorf("connector %d returned an invalid type", rw.fh.connector)
		}
	}
	var ok bool
	if rw.identityIndex, ok = index[rw.identityColumn]; !ok {
		return connector.MissingIdentityColumnError{Column: rw.identityColumn}
	}
	if rw.timestampColumn != "" {
		if rw.timestampIndex, ok = index[rw.identityColumn]; !ok {
			return connector.MissingTimestampColumnError{Column: rw.timestampColumn}
		}
	}
	rw.columns = columns
	if rw.onlyColumns {
		return errRecordStop
	}
	return nil
}

// Record receives a record and calls the SetUser of the Firehose.
func (rw *recordWriter) Record(record []any) error {
	if rw.columns == nil {
		return fmt.Errorf("connector %d did not call the Columns method before calling Record", rw.fh.connector)
	}
	if len(record) != len(rw.columns) {
		return errors.New("connector %q has returned records with different lengths")
	}
	properties := map[string]any{}
	for i, c := range rw.columns {
		properties[c.Name] = record[i]
	}
	ts := rw.timestamp
	if rw.timestampIndex != noColumn {
		ts, err := time.Parse("2006-01-02 15:04:05", record[rw.timestampIndex].(string))
		if err != nil {
			return fmt.Errorf("invalid timestamp column value: %s", ts)
		}
	}
	user := fmt.Sprintf("%s", record[rw.identityIndex])
	rw.fh.SetUser(user, ts, properties)
	rw.setUserCalled = true
	return nil
}

// RecordMap receives a record and calls the SetUser of the Firehose.
func (rw *recordWriter) RecordMap(record map[string]any) error {
	if rw.columns == nil {
		return fmt.Errorf("connector %d did not call the Columns method before calling RecordMap", rw.fh.connector)
	}
	ts := rw.timestamp
	if rw.timestampIndex != noColumn {
		ts, err := time.Parse("2006-01-02 15:04:05", record[rw.timestampColumn].(string))
		if err != nil {
			return fmt.Errorf("invalid timestamp column value: %s", ts)
		}
	}
	user := fmt.Sprintf("%s", record[rw.identityColumn])
	rw.fh.SetUser(user, ts, record)
	rw.setUserCalled = true
	return nil
}

// RecordString receives a record and calls the SetUser of the Firehose.
func (rw *recordWriter) RecordString(record []string) error {
	if rw.columns == nil {
		return fmt.Errorf("connector %d did not call the Columns method before calling RecordString", rw.fh.connector)
	}
	if len(record) != len(rw.columns) {
		return errors.New("connector %q has returned records with different lengths")
	}
	properties := map[string]any{}
	for i, c := range rw.columns {
		properties[c.Name] = record[i]
	}
	ts := rw.timestamp
	if rw.timestampIndex != noColumn {
		ts, err := time.Parse("2006-01-02 15:04:05", record[rw.timestampIndex])
		if err != nil {
			return fmt.Errorf("invalid timestamp column value: %s", ts)
		}
	}
	user := fmt.Sprintf("%s", record[rw.identityIndex])
	rw.fh.SetUser(user, ts, properties)
	rw.setUserCalled = true
	return nil
}

// Timestamp sets the last modified time for all records.
// If ts is zero time, it means that the timestamp is unknown.
// Timestamp can be called before Record, RecordMap and RecordString.
func (rw *recordWriter) Timestamp(ts time.Time) error {
	if rw.setUserCalled {
		return fmt.Errorf("connector %d called the Timestamp method after a record method", rw.fh.connector)
	}
	rw.timestamp = ts
	return nil
}

func keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// listTransformations lists the transformations for the given connections.
func (fh *firehose) listTransformations(connections []int) ([]Transformation, error) {
	var transformations []Transformation
	for _, c := range connections {
		ts, err := fh.connections.Transformations.List(c)
		if err != nil {
			return nil, err
		}
		for _, t := range ts {
			add := true
			for _, t2 := range transformations {
				if t.ID == t2.ID {
					add = false
					break
				}
			}
			if add {
				transformations = append(transformations, t)
			}
		}
	}
	return transformations, nil
}

// mostRecentTimestamp returns the most recent timestamp referred by a property.
// If there are no timestamps or properties, returns 'time.Time{}'.
func mostRecentTimestamp(timestamps map[string]time.Time, props []InputProperty) time.Time {
	var recent time.Time
	for _, p := range props {
		t := timestamps[p.Name]
		if t.After(recent) {
			recent = t
		}
	}
	return recent
}
