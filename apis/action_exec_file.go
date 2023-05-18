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

	"chichi/apis/errors"
	"chichi/apis/mappings"
	"chichi/apis/normalization"
	"chichi/apis/state"
	_connector "chichi/connector"
	"chichi/connector/types"
)

// importFromFile imports the users from a file.
func (this *Action) importFromFile() error {

	c := this.action.Connection()
	connector := c.Connector()

	// Connect to the file connector.
	ctx := context.Background()
	fh, err := this.newFirehose(ctx)
	if err != nil {
		return err
	}
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

	// Determine the input and the output schema.
	apisConn := &Connection{
		db:         this.db,
		connection: this.action.Connection(),
	}
	inputSchema, err := apisConn.fetchFileSchema(this.action.Path, this.action.Sheet)
	if err != nil {
		return actionExecutionError{fmt.Errorf("an error occurred fetching the schema: %s", err)}
	}
	usersSchema, ok := this.action.Connection().Workspace().Schemas["users"]
	if !ok {
		return actionExecutionError{errors.New("users schema not loaded")}
	}
	outputSchema := usersSchemaToConnectionSchema(*usersSchema, state.DatabaseType)

	mapping, err := mappings.New(inputSchema, outputSchema, this.action.Mapping, this.action.Transformation)
	if err != nil {
		return err
	}

	// Read the records.
	rw := newRecordWriter(c.ID, math.MaxInt, func(record map[string]any) error {

		// Apply the mapping or the transformation.
		mappedUser, err := mapping.Apply(ctx, record)
		if err != nil {
			return err
		}

		// Estrapolate the ID and the timestamp for the user.
		err = applyTimestampWorkaround(mappedUser)
		if err != nil {
			return err
		}
		id := mappedUser["id"].(string)
		delete(mappedUser, "id")
		timestamp, ok := mappedUser["timestamp"].(time.Time)
		if !ok {
			timestamp = time.Now().UTC()
		}
		delete(mappedUser, "timestamp")

		// Write the user and the mapped user on the database.
		err = apisConn.writeConnectionUsers(ctx, id, record, timestamp, nil)
		if err != nil {
			return err
		}
		err = apisConn.setUser(ctx, id, mappedUser)

		return err
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

// newRecordReader returns a new record reader that read records.
func newRecordReader(columns []types.Property, records [][]any) *recordReader {
	return &recordReader{
		columns: columns,
		records: records,
	}
}

// recordReader implements the connector.RecordReader interface.
type recordReader struct {
	columns []types.Property
	records [][]any
	cursor  int
}

// Columns returns the columns of the records.
func (rr *recordReader) Columns() []types.Property {
	return rr.columns
}

// Record returns the next record as a slice of any. It returns nil and io.EOF
// if there are no more records.
func (rr *recordReader) Record() ([]any, error) {
	if rr.cursor >= len(rr.records) {
		return nil, io.EOF
	}
	record := rr.records[rr.cursor]
	rr.cursor++
	return record, nil
}

// RecordString returns the next record as a string slice. It returns nil and
// io.EOF if there are no more records.
func (rr *recordReader) RecordString() ([]string, error) {
	if rr.cursor >= len(rr.records) {
		return nil, io.EOF
	}
	record := rr.records[rr.cursor]
	records := make([]string, len(record))
	for i, prop := range record {
		records[i] = fmt.Sprintf("%v", prop) // TODO(Marco): revise
	}
	rr.cursor++
	return records, nil
}

// newRecordWriter returns a new record writer that writes at most limit
// records. If write is not nil, it calls the write function for each record
// written, otherwise it stores the records in the records field.
func newRecordWriter(connector int, limit int, write func(record map[string]any) error) *recordWriter {
	rw := recordWriter{
		connector:       connector,
		limit:           limit,
		write:           write,
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
	} else {
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
	} else {
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
	} else {
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
