//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"

	"chichi/apis/types"
	_connector "chichi/connector"
)

// importFromFile imports the users from a file.
func (ac *Action) importFromFile() error {

	connection := ac.action.Connection()
	connector := connection.Connector()

	var ctx = context.Background()

	// Retrieve the storage associated to the file connection.
	var storage _connector.StorageConnection
	{
		s, _ := connection.Storage()
		fh := ac.newFirehoseForConnection(ctx, s)
		ctx = fh.ctx
		var err error
		storage, err = _connector.RegisteredStorage(s.Connector().Name).Open(ctx, &_connector.StorageConfig{
			Role:     _connector.SourceRole,
			Settings: s.Settings,
			Firehose: fh,
		})
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot connect to the storage connector: %s", err)}
		}
	}

	// Connect to the file connector.
	fh := ac.newFirehose(context.Background())
	file, err := _connector.RegisteredFile(connector.Name).Open(fh.ctx, &_connector.FileConfig{
		Role:     _connector.SourceRole,
		Settings: connection.Settings,
		Firehose: fh,
	})
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the file connector: %s", err)}
	}

	// Read the records.
	rc, timestamp, err := storage.Reader(file.Path())
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot get ReadCloser from storage: %s", err)}
	}
	defer rc.Close()
	records := newRecordWriter(fh, identityColumn, timestampColumn, timestamp, false)
	err = file.Read(rc, records)
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot read the file: %s", err)}
	}
	err = rc.Close()
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot close the storage: %s", err)}
	}

	// Handle errors occurred in the firehose.
	if fh.err != nil {
		return fh.err
	}

	return nil
}

// newRecordWriter returns a new record writer.
func newRecordWriter(fh *firehose, identityColumn, timestampColumn string, timestamp time.Time, onlyColumns bool) *recordWriter {
	return &recordWriter{
		fh:              fh,
		onlyColumns:     onlyColumns,
		identityColumn:  identityColumn,
		timestampColumn: timestampColumn,
		timestampIndex:  noColumn,
		timestamp:       timestamp,
	}
}

// recordWriter implements the connector.RecordWriter interface.
type recordWriter struct {
	fh              *firehose
	onlyColumns     bool
	columns         []types.Property
	identityColumn  string
	timestampColumn string
	identityIndex   int
	timestampIndex  int
	timestamp       time.Time
	setUserCalled   bool
}

// Columns sets the columns of the records as properties.
// Columns must be called before Record, RecordMap and RecordString.
func (rw *recordWriter) Columns(columns []types.Property) error {
	if len(columns) == 0 {
		return _connector.ErrNoColumns
	}
	labelIndex := make(map[string]int, len(columns))
	hasName := make(map[string]struct{}, len(columns))
	for i, c := range columns {
		if c.Name == "" {
			return _connector.ErrEmptyColumnName
		}
		if !utf8.ValidString(c.Name) {
			return _connector.ErrInvalidEncodedColumnName
		}
		if _, ok := hasName[c.Name]; ok {
			return _connector.SameColumnNameError{Name: c.Name}
		}
		hasName[c.Name] = struct{}{}
		if _, ok := labelIndex[c.Label]; !ok {
			labelIndex[c.Label] = i
		}
		if !c.Type.Valid() {
			return fmt.Errorf("connector %d returned an invalid type", rw.fh.connection.Connector().ID)
		}
	}
	var ok bool
	if rw.identityIndex, ok = labelIndex[rw.identityColumn]; !ok {
		return _connector.MissingIdentityColumnError{Column: rw.identityColumn}
	}
	if rw.timestampColumn != "" {
		rw.timestampIndex, ok = labelIndex[rw.timestampColumn]
		if !ok {
			return _connector.MissingTimestampColumnError{Column: rw.timestampColumn}
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
		c := rw.fh.connection.Connector()
		return fmt.Errorf("connector %d did not call the Columns method before calling Record", c.ID)
	}
	if len(record) != len(rw.columns) {
		c := rw.fh.connection.Connector()
		return fmt.Errorf("connector %d has returned records with different lengths", c.ID)
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
		c := rw.fh.connection.Connector()
		return fmt.Errorf("connector %d did not call the Columns method before calling RecordString", c.ID)
	}
	if len(record) != len(rw.columns) {
		c := rw.fh.connection.Connector()
		return fmt.Errorf("connector %d has returned records with different lengths", c.ID)
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
