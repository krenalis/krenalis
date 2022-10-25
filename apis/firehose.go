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
	source, err := fh.sources.Get(fh.source)
	if err != nil {
		fh.setError(fmt.Errorf("cannot retrieve transformation from DB: %s", err))
		return
	}
	if source.TransformationFunc == "" {
		fh.setError(fmt.Errorf("no transformation function for data source %d", source.ID))
		return
	}

	// Apply the transformation to the current entity.
	candidateData, candidateTimestamps, err := fh.transformProperties(source.TransformationFunc, properties, timestamps)
	if err != nil {
		fh.setError(fmt.Errorf("cannot transform input properties to output properties: %s", err))
		return
	}
	email, _ := candidateData["Email"].(string)
	if email == "" {
		fh.setError(fmt.Errorf("expecting 'Email' to be a non-empty string, got %#v (of type %T)", candidateData["Email"], candidateData["Email"]))
		return
	}

	ids := identitySolver{fh}

	// Resolve the entity.
	goldenRecordID, err := ids.ResolveEntity(fh.source, user, email)
	if err != nil {
		fh.setError(err)
		return
	}

	// Retrieve the entities which are the same user.
	sameEntities, err := ids.LookupSameEntities(fh.source, user)
	if err != nil {
		fh.setError(err)
		return
	}

	sources, err := fh.listTransformations(keys(sameEntities))
	if err != nil {
		fh.setError(err)
		return
	}
dataSourcesLoop:
	for source, transfSource := range sources {
		for _, user := range sameEntities[source] {
			userData, err := fh.userData(source, user)
			if err != nil {
				fh.setError(err)
				return
			}
			outProps, outTimestamps, err := fh.transformProperties(transfSource, userData.Data, userData.Timestamps)
			if err != nil {
				fh.setError(fmt.Errorf("cannot transform properties for data source %d: %s", source, err))
				return
			}
			for prop := range outProps {
				if _, ok := candidateData[prop]; !ok {
					// This prop is not candidate to be updated on the Golden
					// Record, so it can be skipped.
					continue
				}
				candidateTimestamp, ok := candidateTimestamps[prop]
				if !ok {
					fh.setError(fmt.Errorf("missing timestamp for prop %q", prop))
					return
				}
				if candidateTimestamp.After(outTimestamps[prop]) {
					// This property must be updated.
				} else {
					delete(candidateData, prop)
					if len(candidateData) == 0 {
						// Avoid useless iterations.
						break dataSourcesLoop
					}
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

type dataSourceUserData struct {
	Data       map[string]any
	Timestamps map[string]time.Time
}

// userData returns the data associated to the user from the given source.
func (fh *firehose) userData(source int, user string) (dataSourceUserData, error) {
	var userData dataSourceUserData
	row := fh.sources.myDB.QueryRow(
		"SELECT `data`, `timestamps` FROM `data_sources_users` WHERE `source` = ? AND `user` = ?",
		source, user)
	var rawData []byte
	var rawTimestamps []byte
	err := row.Scan(&rawData, &rawTimestamps)
	if err != nil {
		return dataSourceUserData{}, err
	}
	err = json.Unmarshal(rawData, &userData.Data)
	if err != nil {
		return dataSourceUserData{}, err
	}
	err = json.Unmarshal(rawTimestamps, &userData.Timestamps)
	if err != nil {
		return dataSourceUserData{}, err
	}
	return userData, nil
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
// transformation function specified for the current data source.
func (fh *firehose) transformProperties(transformationSource string, incoming map[string]any, timestamps map[string]time.Time) (map[string]any, map[string]time.Time, error) {
	fn, err := buildTransfFunc(transformationSource)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot build transformation function: %s", err)
	}
	data, updateTimeOut, err := fn(incoming, timestamps)
	if err != nil {
		return nil, nil, err
	}
	return data, updateTimeOut, nil
}

// listTransformations lists the transformations for the given data sources,
// returning a mapping from the data source ID to its corresponding
// transformation function.
func (fh *firehose) listTransformations(sources []int) (map[int]string, error) {
	if len(sources) == 0 {
		return map[int]string{}, nil
	}
	query := &strings.Builder{}
	query.WriteString("SELECT `id`, `transformation` FROM `data_sources`\n" +
		"WHERE `transformation` <> '' AND `id` IN (")
	for i, source := range sources {
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString(strconv.Itoa(source))
	}
	query.WriteString(")")
	rows, err := fh.sources.myDB.Query(query.String())
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
	_, err := fh.sources.myDB.Exec(query.String(), values...)
	if err != nil {
		return fmt.Errorf("cannot write data Golden Record: %s", err)
	}
	return nil
}

func keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// buildTransfFunc builds a transformation function from its source code and
// returns it.
func buildTransfFunc(source string) (TransformationFuncType, error) {
	if source == "" {
		return nil, errors.New("transformation function source cannot be empty")
	}
	src := `{% import time "time" %}{% Fn = ` + source + ` %}`
	opts := &scriggo.BuildOptions{
		Globals: native.Declarations{
			"Fn": (*TransformationFuncType)(nil),
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
	fs := scriggo.Files{"transform.txt": []byte(src)}
	template, err := scriggo.BuildTemplate(fs, "transform.txt", opts)
	if err != nil {
		return nil, err
	}
	var fn TransformationFuncType
	vars := map[string]interface{}{
		"Fn": &fn,
	}
	err = template.Run(io.Discard, vars, nil)
	if err != nil {
		return nil, err
	}
	return fn, nil
}
