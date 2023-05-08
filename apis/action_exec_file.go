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
	"io"
	"math"
	"time"
	"unicode/utf8"

	"chichi/apis/normalization"
	_connector "chichi/connector"
	"chichi/connector/types"
)

// importFromFile imports the users from a file.
func (this *Action) importFromFile() error {

	c := this.action.Connection()
	connector := c.Connector()

	// Connect to the file connector.
	ctx := context.Background()
	fh := this.newFirehose(ctx)
	file, err := _connector.RegisteredFile(connector.Name).Open(fh.ctx, &_connector.FileConfig{
		Role:     _connector.SourceRole,
		Settings: c.Settings,
		Firehose: fh,
	})
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the file connector: %s", err)}
	}

	// Open the file.
	var r io.ReadCloser
	{
		s, _ := c.Storage()
		fh := this.newFirehoseForConnection(ctx, s)
		ctx = fh.ctx
		var err error
		storage, err := _connector.RegisteredStorage(s.Connector().Name).Open(ctx, &_connector.StorageConfig{
			Role:     _connector.SourceRole,
			Settings: s.Settings,
			Firehose: fh,
		})
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot connect to the storage connector: %s", err)}
		}
		r, _, err = storage.Open(this.action.Path)
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot get ReadCloser from storage: %s", err)}
		}
		defer r.Close()
	}

	// Read the records.
	var rw *recordWriter
	rw = newRecordWriter(c.ID, identityLabel, timestampLabel, math.MaxInt, func(record map[string]any) error {
		user := fmt.Sprintf("%s", record[rw.identityColumn])
		ts, err := time.Parse(time.DateTime, record[rw.timestampColumn].(string))
		if err != nil {
			return fmt.Errorf("invalid timestamp column value: %s", record[rw.timestampColumn])
		}
		timestamps := map[string]time.Time{}
		for name := range record {
			timestamps[name] = ts
		}
		return this.setUser(user, record, timestamps)
	})
	err = file.Read(r, this.action.Sheet, rw)
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot read the file: %s", err)}
	}
	err = r.Close()
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot close the storage: %s", err)}
	}

	// Handle errors occurred in the firehose.
	if fh.err != nil {
		return fh.err
	}

	return nil
}

// newRecordWriter returns a new record writer that writes at most limit
// records. If write is not nil, it calls the write function for each record
// written, otherwise it stores the records in the records field.
func newRecordWriter(connector int, identityLabel, timestampLabel string, limit int, write func(record map[string]any) error) *recordWriter {
	rw := recordWriter{
		connector:      connector,
		limit:          limit,
		write:          write,
		identityLabel:  identityLabel,
		timestampLabel: timestampLabel,

		textColumnsOnly: true,
	}
	if write == nil {
		rw.records = [][]any{}
	}
	return &rw
}

// recordWriter implements the connector.RecordWriter interface.
type recordWriter struct {
	connector       int
	limit           int
	write           func(record map[string]any) error
	columns         []types.Property
	timestamp       time.Time
	columnByName    map[string]types.Property
	identityLabel   string
	identityColumn  string
	timestampLabel  string
	timestampColumn string
	setUserCalled   bool
	textColumnsOnly bool
	records         [][]any
	err             error
}

// Columns sets the columns of the records as properties.
// Columns must be called before Record, RecordMap and RecordString.
func (rw *recordWriter) Columns(columns []types.Property) error {
	if rw.columns != nil {
		return fmt.Errorf("connector %d has called Columns twice", rw.connector)
	}
	if len(columns) == 0 {
		return _connector.ErrNoColumns
	}
	labelToName := make(map[string]string, len(columns))
	hasName := make(map[string]struct{}, len(columns))
	for _, c := range columns {
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
		if _, ok := labelToName[c.Label]; !ok {
			labelToName[c.Label] = c.Name
		}
		if !c.Type.Valid() {
			return fmt.Errorf("connector %d returned an invalid type", rw.connector)
		}
		if rw.textColumnsOnly {
			rw.textColumnsOnly = c.Type.PhysicalType() == types.PtText
		}
		if c.Label == rw.identityLabel {
			rw.identityColumn = c.Name
		}
		if c.Label == rw.timestampLabel {
			rw.timestampColumn = c.Name
		}
	}
	if rw.identityColumn == "" {
		return _connector.MissingIdentityColumnError{Column: rw.identityLabel}
	}
	if rw.timestampLabel != "" && rw.timestampColumn == "" {
		return _connector.MissingTimestampColumnError{Column: rw.timestampLabel}
	}
	rw.columns = columns
	if rw.limit == 0 {
		return errRecordStop
	}
	return nil
}

// Record writes a record.
func (rw *recordWriter) Record(record []any) error {
	if rw.columns == nil {
		return fmt.Errorf("connector %d did not call the Columns method before calling Record", rw.connector)
	}
	if len(record) != len(rw.columns) {
		return fmt.Errorf("connector %d has returned records with different lengths", rw.connector)
	}
	var err error
	if rw.write == nil {
		// Store the record in the records field.
		rd := make([]any, len(rw.columns))
		for i, c := range rw.columns {
			rd[i], err = normalization.NormalizeDatabaseFileProperty(c.Name, c.Nullable, c.Type, record[i])
			if err != nil {
				return err
			}
		}
		rw.records = append(rw.records, rd)
		return nil
	}
	// Call the rw.write function to store the record.
	rd := map[string]any{}
	for i, c := range rw.columns {
		rd[c.Name], err = normalization.NormalizeDatabaseFileProperty(c.Name, c.Nullable, c.Type, record[i])
		if err != nil {
			return err
		}
	}
	err = rw.write(rd)
	if err != nil {
		return err
	}
	rw.limit--
	if rw.limit == 0 {
		return errRecordStop
	}
	return nil
}

// RecordMap writes a record as a map.
func (rw *recordWriter) RecordMap(record map[string]any) error {
	if rw.columns == nil {
		return fmt.Errorf("connector %d did not call the Columns method before calling RecordMap", rw.connector)
	}
	var err error
	if rw.columnByName == nil {
		rw.columnByName = make(map[string]types.Property, len(rw.columns))
		for _, c := range rw.columns {
			rw.columnByName[c.Name] = c
		}
	}
	if rw.write == nil {
		// Store the record in the records field.
		rd := make([]any, len(rw.columns))
		for i, c := range rw.columns {
			rd[i], err = normalization.NormalizeDatabaseFileProperty(c.Name, c.Nullable, c.Type, record[c.Name])
			if err != nil {
				return err
			}
		}
		rw.records = append(rw.records, rd)
		return nil
	}
	// Call the rw.write function to store the record.
	for _, c := range rw.columns {
		v, err := normalization.NormalizeDatabaseFileProperty(c.Name, c.Nullable, c.Type, record[c.Name])
		if err != nil {
			return err
		}
		record[c.Name] = v
	}
	err = rw.write(record)
	if err != nil {
		return err
	}
	rw.limit--
	if rw.limit == 0 {
		return errRecordStop
	}
	return nil
}

// RecordString writes a record as a string slice.
func (rw *recordWriter) RecordString(record []string) error {
	if rw.columns == nil {
		return fmt.Errorf("connector %d did not call the Columns method before calling RecordString", rw.connector)
	}
	if len(record) != len(rw.columns) {
		return fmt.Errorf("connector %d has returned records with different lengths", rw.connector)
	}
	if !rw.textColumnsOnly {
		return fmt.Errorf("connector %d has called RecordString when there are non-text columns", rw.connector)
	}
	var err error
	if rw.write == nil {
		// Store the record in the records field.
		rd := make([]any, len(rw.columns))
		for i, c := range rw.columns {
			err = normalization.ValidateStringProperty(c, record[i])
			if err != nil {
				return err
			}
			rd[i] = record[i]
		}
		rw.records = append(rw.records, rd)
		return nil
	}
	// Call the rw.write function to store the record.
	rd := map[string]any{}
	for i, c := range rw.columns {
		err = normalization.ValidateStringProperty(c, record[i])
		if err != nil {
			return err
		}
		rd[c.Name] = record[i]
	}
	err = rw.write(rd)
	if err != nil {
		return err
	}
	rw.limit--
	if rw.limit == 0 {
		return errRecordStop
	}
	return nil
}

// Timestamp sets the last modified time for all records.
// If ts is zero time, it means that the timestamp is unknown.
// Timestamp can be called before Record, RecordMap and RecordString.
func (rw *recordWriter) Timestamp(ts time.Time) error {
	if rw.setUserCalled {
		return fmt.Errorf("connector %d called the Timestamp method after a record method", rw.connector)
	}
	rw.timestamp = ts
	return nil
}
