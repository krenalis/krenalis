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
	"fmt"
	"io"
	"log"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/connectors"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/native"
)

// Make sure it implements the Firehose interface.
var _ connectors.Firehose = &firehose{}

const maxSettingsLen = 10_000 // Maximum length of settings in runes.

// firehose is the Firehose API used by the connectors.
type firehose struct {
	sources       *DataSources
	source        int
	resource      int
	connector     int
	connectorType string
	ctx           context.Context
	cancel        context.CancelFunc
	webhooksPer   string
	err           error
}

func (fh *firehose) ReceiveEvent(event connectors.Event) {
	return
}

// SetCursor sets the user cursor.
func (fh *firehose) SetCursor(cursor string) {
	result, err := fh.sources.myDB.Exec("UPDATE `data_sources`\nSET `userCursor` = ?\nWHERE `id` = ?", cursor, fh.source)
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

// SetSettings sets the given settings of the data source.
func (fh *firehose) SetSettings(settings []byte) error {
	if !utf8.Valid(settings) {
		return errors.New("settings is not valid UTF-8")
	}
	if utf8.RuneCount(settings) > maxSettingsLen {
		return fmt.Errorf("settings is longer than %d runes", maxSettingsLen)
	}
	settingsColumn := "`settings`"
	if fh.connectorType == "Stream" {
		settingsColumn = "`streamSettings`"
	}
	_, err := fh.sources.myDB.Exec("UPDATE `data_sources`\nSET "+settingsColumn+" = ?\nWHERE `id` = ?", settings, fh.source)
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
			if tv, ok := v.(connectors.TimestampedValue); ok {
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
	err := fh.writeDataSourceUsers(user, properties, timestamps)
	if err != nil {
		fh.setError(err)
		return
	}

	// Apply the transformation of this data-source to the properties and
	// timestamps.
	transformationSource, err := fh.sources.TransformationFunc(fh.source)
	if err != nil {
		fh.setError(fmt.Errorf("cannot retrieve transformation from DB: %s", err))
		return
	}
	candidateData, candidateTimestamps, err := fh.transformProperties(transformationSource, properties, timestamps)
	if err != nil {
		fh.setError(fmt.Errorf("cannot transform input properties to output properties: %s", err))
		return
	}

	// Determine which properties should be updated, basing on the last update
	// timestamp.
	sources, err := fh.listAllTransformations()
	if err != nil {
		fh.setError(err)
		return
	}
	for source, transfSource := range sources {
		if source == fh.source {
			// Skip the current source.
			continue
		}
		users, timestamps, err := fh.usersForDataSource(source)
		if err != nil {
			fh.setError(err)
			return
		}
		for i := range users {
			outProps, outTimestamps, err := fh.transformProperties(transfSource, users[i], timestamps[i])
			if err != nil {
				fh.setError(err)
				return
			}
			isSameUser, err := fh.sameUser(outProps, candidateData)
			if err != nil {
				fh.setError(err)
				return
			}
			if !isSameUser {
				// This is another user, so it can be skipped.
				continue
			}
			for prop := range outProps {
				if _, ok := candidateData[prop]; !ok {
					// This prop is not candidate to be updated on the Golden
					// Record, so it can be skipped.
					continue
				}
				if candidateTimestamps[prop].After(outTimestamps[prop]) {
					// This property must be updated.
				} else {
					delete(candidateData, prop)
					log.Printf("[info] property %q is already up-to-date", prop)
				}
			}
		}
	}

	// Write the data to the Golden Record.
	if len(candidateData) > 0 {
		err := fh.writeToGoldenRecord(candidateData)
		if err != nil {
			fh.setError(err)
			return
		}
		log.Printf("[info] properties for user %q written to the Golden Record", candidateData["Email"])
	}

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
	case "DataSource":
		return u + "s/" + strconv.Itoa(fh.source) + "/"
	}
	panic("unexpected webhookPer value")
}

// setError sets fh.err and cancels the context.
func (fh *firehose) setError(err error) {
	fh.err = err
	fh.cancel()
	log.Printf("[error] firehose error: %s", err)
}

// TransformationFuncType is the type of a transformation function.
type TransformationFuncType = func(
	dataIn map[string]any,
	timestampsIn map[string]time.Time,
) (
	dataOut map[string]any,
	timestampsOut map[string]time.Time,
	err error,
)

// transformProperties transforms the incoming properties using the
// transformation function specified for the current connector.
func (fh *firehose) transformProperties(transformationSource string, incoming map[string]any, timestamps map[string]time.Time) (map[string]any, map[string]time.Time, error) {
	fullSourceCode := strings.Replace(
		`{% import time "time" %}
		{% Transform = {{ transformationFunction }} %}`,
		"{{ transformationFunction }}",
		transformationSource,
		1,
	)
	opts := &scriggo.BuildOptions{
		Globals: native.Declarations{
			"Transform": (*TransformationFuncType)(nil),
		},
		Packages: native.Packages{
			"time": native.Package{
				Name: "time",
				Declarations: native.Declarations{
					"Time": reflect.TypeOf(time.Time{}),
				},
			},
		},
	}
	fs := scriggo.Files{"main.txt": []byte(fullSourceCode)}
	template, err := scriggo.BuildTemplate(fs, "main.txt", opts)
	if err != nil {
		return nil, nil, err
	}
	transform := TransformationFuncType(nil)
	vars := map[string]interface{}{
		"Transform": &transform,
	}
	err = template.Run(io.Discard, vars, nil)
	if err != nil {
		return nil, nil, err
	}
	data, updateTimeOut, err := transform(incoming, timestamps)
	if err != nil {
		return nil, nil, err
	}
	return data, updateTimeOut, nil
}

// listAllTransformations returns a mapping between the data source IDs and
// their corresponding transformation functions.
func (fh *firehose) listAllTransformations() (map[int]string, error) {
	rows, err := fh.sources.myDB.Query("SELECT `id`, `transformation` FROM `data_sources`")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	transformations := map[int]string{}
	for rows.Next() {
		var id int
		var transformation string
		err := rows.Scan(&id, &transformation)
		if err != nil {
			return nil, err
		}
		transformations[id] = transformation
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return transformations, nil
}

// usersForDataSource returns every user and timestamps for the given data
// source.
func (fh *firehose) usersForDataSource(source int) ([]map[string]any, []map[string]time.Time, error) {
	rows, err := fh.sources.myDB.Query("SELECT `data`, `timestamps` FROM `data_sources_users` WHERE `source` = ?", source)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var users []map[string]any
	var allTimestamps []map[string]time.Time
	for rows.Next() {
		var rawData, rawTimestamps []byte
		err := rows.Scan(&rawData, &rawTimestamps)
		if err != nil {
			return nil, nil, err
		}
		// Deserialize the user data.
		var data map[string]any
		err = json.Unmarshal(rawData, &data)
		if err != nil {
			return nil, nil, err
		}
		users = append(users, data)
		// Deserialize the timestamps.
		var ts map[string]time.Time
		err = json.Unmarshal(rawTimestamps, &ts)
		if err != nil {
			return nil, nil, err
		}
		allTimestamps = append(allTimestamps, ts)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return users, allTimestamps, nil
}

// statsTimeSlot returns the stats time slot for the time t.
// t must be a UTC time.
func statsTimeSlot(t time.Time) int {
	epoc := int(t.Unix())
	return epoc / (60 * 60)
}

// writeDataSourceUsers writes the given data user users to the database.
func (fh *firehose) writeDataSourceUsers(user string, props map[string]any, timestamps map[string]time.Time) error {
	data, err := json.Marshal(props)
	if err != nil {
		return err
	}
	jsonTimestamps, err := json.Marshal(timestamps)
	if err != nil {
		return err
	}
	_, err = fh.sources.myDB.Exec("INSERT INTO `data_sources_users`\n"+
		"SET `source` = ?, `user` = ?, `data` = ?, `timestamps` = ?\n"+
		"ON DUPLICATE KEY UPDATE `data` = ?, `timestamps` = ?",
		fh.source, user, data, jsonTimestamps, data, jsonTimestamps)
	if err != nil {
		return err
	}
	_, err = fh.sources.myDB.Exec("INSERT INTO `data_sources_stats`\n"+
		"SET `source` = ?, `timeSlot` = ?, `usersIn` = 1\n"+
		"ON DUPLICATE KEY UPDATE `usersIn` = `usersIn` + 1",
		fh.source, statsTimeSlot(time.Now()))
	return err
}

// writeToGoldenRecord writes the given properties to the Golden Record.
func (fh *firehose) writeToGoldenRecord(props map[string]any) error {
	columns := make([]string, len(props))
	i := 0
	for column := range props {
		columns[i] = column
		i++
	}
	sort.Strings(columns)
	query := "INSERT INTO `warehouse_users` (" + strings.Join(columns, ", ") + ") VALUES ("
	for i := range columns {
		if i > 0 {
			query += ","
		}
		query += "?"
	}
	query += ") ON DUPLICATE KEY UPDATE "
	for i := range columns {
		if i > 0 {
			query += ", "
		}
		query += columns[i] + " = ?"
	}
	values := []any{}
	for _, column := range columns {
		values = append(values, props[column])
	}
	_, err := fh.sources.myDB.Exec(query, append(values, values...)...)
	if err != nil {
		return fmt.Errorf("cannot write data to database: %s", err)
	}
	return nil
}

// sameUser reports whether the given properties refer to the same user.
// Note that the properties should be the result of the transformation
// functions.
func (fh *firehose) sameUser(props1, props2 map[string]any) (bool, error) {
	email1, ok := props1["Email"]
	if !ok {
		return false, fmt.Errorf("user has no 'Email' (%#v)", props1)
	}
	if email1 == "" {
		return false, fmt.Errorf("user has empty 'Email' (%#v)", props1)
	}
	email2, ok := props2["Email"]
	if !ok {
		return false, fmt.Errorf("user has no 'Email' (%#v)", props2)
	}
	if email2 == "" {
		return false, fmt.Errorf("user has empty 'Email' (%#v)", props2)
	}
	return email1 == email2, nil
}
