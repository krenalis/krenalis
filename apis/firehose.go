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
	"sort"
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
	sources   *DataSources
	connector int
	resource  string
	context   context.Context
	cancel    context.CancelFunc
	err       error
}

func (fh *firehose) ReceiveEvent(event connectors.Event) {
	return
}

// SetCursor sets the user cursor.
func (fh *firehose) SetCursor(cursor string) {
	result, err := fh.sources.myDB.Exec("UPDATE `data_sources`\nSET `userCursor` = ?\nWHERE `workspace` = ? AND `connector` = ?",
		cursor, fh.sources.workspace, fh.connector)
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

func (fh *firehose) SetGroup(group string, updateTime time.Time, properties map[string]any) {
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
	_, err := fh.sources.myDB.Exec("UPDATE `data_sources`\nSET `settings` = ?\nWHERE `workspace` = ? AND `connector` = ?",
		settings, fh.sources.workspace, fh.connector)
	if err != nil {
		log.Printf("[error] %s", err)
		return errors.New("cannot set settings")
	}
	return nil
}

func (fh *firehose) SetUser(user string, updateTime time.Time, properties map[string]any) {
	data, err := json.Marshal(properties)
	if err != nil {
		fh.setError(err)
		return
	}
	_, err = fh.sources.myDB.Exec("INSERT INTO `data_sources_users`\n"+
		"SET `workspace` = ?, `connector` = ?, `user` = ?, `data` = ?\n"+
		"ON DUPLICATE KEY UPDATE `data` = ?",
		fh.sources.workspace, fh.connector, user, data, data)
	if err != nil {
		fh.setError(err)
		return
	}
	goldenRecordData, err := fh.transformProperties(properties)
	if err != nil {
		fh.setError(fmt.Errorf("cannot transform input properties to output properties: %s", err))
		return
	}
	// Serialize the data to the Golden Record.
	{
		columns := make([]string, len(goldenRecordData))
		i := 0
		for column := range goldenRecordData {
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
			values = append(values, goldenRecordData[column])
		}
		_, err = fh.sources.myDB.Exec(query, append(values, values...)...)
		if err != nil {
			fh.setError(fmt.Errorf("cannot write data to database: %s", err))
			return
		}
	}
}

func (fh *firehose) SetUserGroups(user string, groups []string) {
	return
}

// setError sets fh.err and cancels the context.
func (fh *firehose) setError(err error) {
	fh.err = err
	fh.cancel()
	return
}

// transformProperties transforms the incoming properties using the
// transformation function specified for the current connector.
func (fh *firehose) transformProperties(incoming map[string]any) (map[string]any, error) {
	transformationSource, err := fh.sources.TransformationFunc(fh.connector)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve transformation from DB: %s", err)
	}
	fullSourceCode := strings.Replace(
		`{% Transform = {{ transformationFunction }} %}`,
		"{{ transformationFunction }}",
		transformationSource,
		1,
	)
	opts := &scriggo.BuildOptions{
		Globals: native.Declarations{
			"Transform": (*func(map[string]any) (map[string]any, error))(nil),
		},
	}
	fs := scriggo.Files{"main.txt": []byte(fullSourceCode)}
	template, err := scriggo.BuildTemplate(fs, "main.txt", opts)
	if err != nil {
		return nil, err
	}
	transform := (func(map[string]any) (map[string]any, error))(nil)
	vars := map[string]interface{}{
		"Transform": &transform,
	}
	err = template.Run(io.Discard, vars, nil)
	if err != nil {
		return nil, err
	}
	data, err := transform(incoming)
	if err != nil {
		return nil, err
	}
	return data, nil
}
