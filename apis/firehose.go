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
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/apis/transformations"
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
	resource    int
	ctx         context.Context
	cancel      context.CancelFunc
	webhooksPer WebhooksPer
	err         error
}

func (fh *firehose) ReceiveEvent(event connector.Event) {

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

	// Set the timestamps.
	if timestamps == nil {
		timestamps = map[string]time.Time{}
	}
	for name := range properties {
		if _, ok := timestamps[name]; !ok {
			timestamps[name] = timestamp
		}
	}

	// Serialize the properties and the timestamps to the database.
	err := fh.writeConnectionUsers(user, properties, timestamps)
	if err != nil {
		fh.setError(err)
		return
	}

	// Create a pool of transformation VMs.
	//
	// TODO(Gianluca): this is not necessary when there are no custom
	// transformation function. When reviewing that part, also review this
	// initialization.
	pool := transformations.NewPool()

	// Prepare the candidate data and timestamps.
	candidateData := map[string]any{}
	candidateTimestamps := map[string]time.Time{}

	ctx := context.Background()

	// If the connection has an associated transformation, run it.
	if t := fh.connection.Transformation(); t != nil {

		// Prepare the user for the transformation.
		inPropsNames := t.In.PropertiesNames()
		user := make(map[string]any, len(inPropsNames))
		for _, name := range inPropsNames {
			value, ok := properties[name]
			if !ok {
				fh.setError(importError{fmt.Errorf("property %q not found in connection %d", name, fh.connection.ID)})
				return
			}
			user[name] = value
		}
		// Validate the input properties according to the input schema.
		{
			data, err := json.Marshal(user)
			if err != nil {
				fh.setError(err)
				return
			}
			_, err = types.Decode(bytes.NewReader(data), t.In)
			if err != nil {
				fh.setError(importError{fmt.Errorf("input schema validation failed: %s", err)})
				return
			}
		}
		// Run the Python transformation function.
		var err error
		candidateData, err = pool.Run2(ctx, t.PythonSource, user)
		if err != nil {
			fh.setError(importError{fmt.Errorf("error while calling transformation function of mapping: %s", err)})
			return
		}
		// Validate the properties returned by Python according to the output schema.
		{
			data, err := json.Marshal(candidateData)
			if err != nil {
				fh.setError(err)
				return
			}
			_, err = types.Decode(bytes.NewReader(data), t.Out)
			if err != nil {
				fh.setError(importError{fmt.Errorf("output schema validation failed: %s", err)})
				return
			}
		}
		// Determine the timestamps for the updated properties.
		ts := mostRecentTimestamp(timestamps, inPropsNames)
		for _, name := range inPropsNames {
			candidateTimestamps[name] = ts
		}
	}

	// Apply the transformations of mappings, calculate the Golden Record
	// properties and their relative timestamps for this user in this
	// connection.
	for _, m := range fh.connection.Mappings() {

		// Ensure that the input properties exist.
		for _, p := range m.InProperties {
			if _, ok := properties[p]; !ok {
				fh.setError(importError{fmt.Errorf("property %q not found in connection %d", p, fh.connection.ID)})
				return
			}
		}

		if m.PredefinedFunc == nil && m.CustomFunc == nil {

			// "One to one" mapping.
			candidateData[m.OutProperties[0]] = properties[m.InProperties[0]]
			candidateTimestamps[m.OutProperties[0]] = timestamps[m.InProperties[0]]

		} else if m.PredefinedFunc != nil {

			// Predefined transformation.
			f, _ := predefinedFuncDefinitionByID(*m.PredefinedFunc)
			in := make([]any, len(m.InProperties))
			// TODO(Gianluca): this code that makes the validation can be
			// simplified by changing the APIs of the 'types' package.
			values := map[string]any{}
			for i, p := range m.InProperties {
				values[p] = properties[p]
				in[i] = properties[m.InProperties[i]]
			}
			j, _ := json.Marshal(values)
			_, err := types.Decode(bytes.NewReader(j), f.In)
			if err != nil {
				fh.setError(importError{err})
				return
			}
			out := callPredefinedFunc(f, in)
			ts := mostRecentTimestamp(timestamps, m.InProperties)
			for i, outName := range m.OutProperties {
				candidateData[outName] = out[i]
				candidateTimestamps[outName] = ts
			}

		} else {

			// Mapping with a custom transformation function.

			// Validate input properties.
			{
				values := map[string]any{}
				for _, p := range m.InProperties {
					values[p] = properties[p]
				}
				schema, _ := typesToSchema(m.CustomFunc.InTypes, m.InProperties)
				j, _ := json.Marshal(values)
				_, err = types.Decode(bytes.NewReader(j), schema)
				if err != nil {
					fh.setError(importError{err})
					return
				}
			}

			in := make([]any, len(m.InProperties))
			for i := range in {
				in[i] = properties[m.InProperties[i]]
			}
			out, err := pool.Run(ctx, m.CustomFunc.Source, in)
			if err != nil {
				fh.setError(importError{fmt.Errorf("error while calling transformation function of mapping: %s", err)})
				return
			}

			// Validate output properties.
			{
				values := map[string]any{}
				for i, p := range m.OutProperties {
					values[p] = out[i]
				}
				schema, _ := typesToSchema(m.CustomFunc.OutTypes, m.OutProperties)
				j, _ := json.Marshal(values)
				_, err = types.Decode(bytes.NewReader(j), schema)
				if err != nil {
					fh.setError(importError{err})
					return
				}
			}

			ts := mostRecentTimestamp(timestamps, m.InProperties)
			for i, name := range m.OutProperties {
				candidateData[name] = out[i]
				candidateTimestamps[name] = ts
			}
		}

	}

	email, _ := candidateData["Email"].(string)
	if email == "" {
		fh.setError(importError{fmt.Errorf("expecting 'Email' to be a non-empty string, got %#v (of type %T)", candidateData["Email"], candidateData["Email"])})
		return
	}

	ids := identitySolver{fh}

	// Resolve the entity of this user.
	goldenRecordID, err := ids.ResolveEntity(fh.connection.ID, user, email)
	if err != nil {
		fh.setError(err)
		return
	}

	// Retrieve the entities which are the same user.
	sameEntities, err := ids.LookupSameEntities(fh.connection.ID, user)
	if err != nil {
		fh.setError(fmt.Errorf("cannot lookup same entities for user %q: %s", user, err))
		return
	}

	// Retrieve the mappings for the entities that match with the current user.
	otherMappings, connectionOfMapping, err := fh.listMappings(keys(sameEntities))
	if err != nil {
		fh.setError(fmt.Errorf("cannot retrieve mappings for other entities: %s", err))
		return
	}

	// Discard any incoming Golden Record property which is older than the
	// existent properties.
transfLoop:
	for _, m := range otherMappings {
		// For the connection of this mapping, determine the timestamps relative
		// to the users which refers to the same identity.
		for _, u := range sameEntities[connectionOfMapping[m]] {
			entityData, err := fh.entityData(connectionOfMapping[m], u)
			if err != nil {
				fh.setError(err)
				return
			}
			for name := range candidateData {
				if _, ok := entityData.Timestamps[name]; !ok {
					continue
				}
				ts := mostRecentTimestamp(entityData.Timestamps, m.InProperties)
				if ts.After(candidateTimestamps[name]) {
					// Don't update this Golden Record property.
					delete(candidateData, name)
					if len(candidateData) == 0 {
						// Avoid useless iterations.
						break transfLoop
					}
				}
			}
		}
	}

	// Write the data to the Golden Record.
	//
	// TODO(Gianluca): there may be a case when a property is written to the
	// Golden Record, but such property is not present in the database anymore
	// (or is no longer compatible).
	// Consider this kind of error when reviewing the implementation of the part
	// that writes values to the warehouse.
	//
	//
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
	u := "https://localhost:9090/webhook/"
	switch fh.webhooksPer {
	case WebhooksPerNone:
		return ""
	case WebhooksPerConnector:
		return u + "c/" + strconv.Itoa(fh.connection.Connector().ID) + "/"
	case WebhooksPerResource:
		return u + "r/" + strconv.Itoa(fh.resource) + "/"
	case WebhooksPerSource:
		return u + "s/" + strconv.Itoa(fh.connection.ID) + "/"
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
	ws := fh.connection.Workspace()
	_, err = ws.Warehouse.Exec(fh.ctx, "INSERT INTO connections_users (connection, \"user\", data, timestamps)\n"+
		"VALUES ($1, $2, $3, $4)\n"+
		"ON CONFLICT (connection, \"user\") DO UPDATE SET data = $3, timestamps = $4",
		fh.connection.ID, user, data, jsonTimestamps)
	if err != nil {
		return err
	}
	_, err = fh.db.Exec(fh.ctx, "INSERT INTO connections_stats AS cs (connection, time_slot, users_in)\n"+
		"VALUES ($1, $2, 1)\n"+
		"ON CONFLICT (connection, time_slot) DO UPDATE SET users_in = cs.users_in + 1",
		fh.connection.ID, statsTimeSlot(time.Now()))
	return err
}

// writeToGoldenRecord writes the given properties to the Golden Record.
func (fh *firehose) writeToGoldenRecord(id int, props map[string]any) error {
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
			return fmt.Errorf("connector %d returned an invalid type", rw.fh.connection.Connector().ID)
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
		return fmt.Errorf("connector %d did not call the Columns method before calling Record", rw.fh.connection.Connector().ID)
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
		var err error
		ts, err = time.Parse(time.DateTime, record[rw.timestampIndex].(string))
		if err != nil {
			return fmt.Errorf("invalid timestamp column value: %s", ts)
		}
	}
	user := fmt.Sprintf("%s", record[rw.identityIndex])
	rw.fh.SetUser(user, properties, ts, nil)
	rw.setUserCalled = true
	return nil
}

// RecordMap receives a record and calls the SetUser of the Firehose.
func (rw *recordWriter) RecordMap(record map[string]any) error {
	if rw.columns == nil {
		return fmt.Errorf("connector %d did not call the Columns method before calling RecordMap", rw.fh.connection.Connector().ID)
	}
	ts := rw.timestamp
	if rw.timestampIndex != noColumn {
		var err error
		ts, err = time.Parse(time.DateTime, record[rw.timestampColumn].(string))
		if err != nil {
			return fmt.Errorf("invalid timestamp column value: %s", ts)
		}
	}
	user := fmt.Sprintf("%s", record[rw.identityColumn])
	rw.fh.SetUser(user, record, ts, nil)
	rw.setUserCalled = true
	return nil
}

// RecordString receives a record and calls the SetUser of the Firehose.
func (rw *recordWriter) RecordString(record []string) error {
	if rw.columns == nil {
		return fmt.Errorf("connector %d did not call the Columns method before calling RecordString", rw.fh.connection.Connector().ID)
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
		var err error
		ts, err = time.Parse(time.DateTime, record[rw.timestampIndex])
		if err != nil {
			return fmt.Errorf("invalid timestamp column value: %s", ts)
		}
	}
	user := fmt.Sprintf("%s", record[rw.identityIndex])
	rw.fh.SetUser(user, properties, ts, nil)
	rw.setUserCalled = true
	return nil
}

// Timestamp sets the last modified time for all records.
// If ts is zero time, it means that the timestamp is unknown.
// Timestamp can be called before Record, RecordMap and RecordString.
func (rw *recordWriter) Timestamp(ts time.Time) error {
	if rw.setUserCalled {
		return fmt.Errorf("connector %d called the Timestamp method after a record method", rw.fh.connection.Connector().ID)
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

// listMappings lists the mappings for the given connections, and a map from the
// Mapping to the corresponding connection ID.
func (fh *firehose) listMappings(connections []int) ([]*state.Mapping, map[*state.Mapping]int, error) {
	var mappings []*state.Mapping
	connectionOfMapping := map[*state.Mapping]int{}
	for _, c := range connections {
		ws := fh.connection.Workspace()
		conn, ok := ws.Connection(c)
		if !ok {
			return nil, nil, fmt.Errorf("connection %d does not exist anymore", c)
		}
		mappings = append(mappings, conn.Mappings()...)
		for _, m := range conn.Mappings() {
			connectionOfMapping[m] = c
		}
	}
	return mappings, connectionOfMapping, nil
}

// mostRecentTimestamp returns the most recent timestamp referred by a property.
// If there are no timestamps or properties, returns 'time.Time{}'.
func mostRecentTimestamp(timestamps map[string]time.Time, props []string) time.Time {
	var recent time.Time
	for _, p := range props {
		t := timestamps[p]
		if t.After(recent) {
			recent = t
		}
	}
	return recent
}

func typesToSchema(ts []types.Type, names []string) (types.Type, error) {
	if len(ts) != len(names) {
		panic("different lengths")
	}
	props := make([]types.Property, len(ts))
	for i := range props {
		props[i] = types.Property{
			Name: names[i],
			Type: ts[i],
		}
	}
	return types.ObjectOf(props)
}
