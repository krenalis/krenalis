//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package connectors

import (
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	pathPkg "path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/types"

	"github.com/golang/snappy"
	"github.com/itchyny/timefmt-go"
	"github.com/relvacode/iso8601"
)

// storageTimeout represents the duration between consecutive calls to the Read
// method of the io.Reader passed to a storage within the Write method.
var storageTimeout = 10 * time.Second

type File struct {
	state  *state.State
	action *state.Action
	inner  chichi.File
	err    error
}

// File returns a file for the provided action, on a connection with the given
// role. Errors are deferred until a file's method is called.
func (connectors *Connectors) File(action *state.Action, role state.Role) *File {
	file := &File{state: connectors.state,
		action: action,
	}
	file.inner, file.err = chichi.RegisteredFile(action.Connector().Name).New(&chichi.FileConfig{
		Role:        chichi.Role(role),
		Settings:    action.Settings,
		SetSettings: setActionSettingsFunc(connectors.state, action),
	})
	return file
}

// ContentType returns the content type of the file.
func (file *File) ContentType(ctx context.Context) (string, error) {
	if file.err != nil {
		return "", file.err
	}
	return file.inner.ContentType(ctx), nil
}

// Records returns an iterator to iterate over the file's records. Each returned
// record will contain, in the Properties field, the properties of the input
// schema of the action passed to the constructor of File, with the same types.
//
// If the schema of the action of the File (that must be valid) does not conform
// with the schema read from the file, the iterator will return a *SchemaError
// error.
//
// If the unique ID column specified in the action of the file is found within
// the file schema but its type is different, the iterator will return an error.
// The same applies for the timestamp, if specified.
//
// If the action's sheet is not found in the file, the For method of the
// iterator returns immediately, and a subsequent call to the Err method will
// return the ErrSheetNotExist error. The same occurs if the file has no
// columns; in this case, the error is ErrNoColumns.
func (file *File) Records(ctx context.Context) (Records, error) {
	if file.err != nil {
		return nil, file.err
	}
	if !file.action.InSchema.Valid() {
		return nil, fmt.Errorf("action input schema is not valid")
	}
	storage, err := file.storage()
	if err != nil {
		return nil, err
	}
	s := newCompressedStorage(storage, file.action.Compression)
	rc, storageTimestamp, err := s.Reader(ctx, file.action.Path)
	if err != nil {
		return nil, err
	}
	if err = validateTimestamp(storageTimestamp); err != nil {
		return nil, fmt.Errorf("invalid timestamp returned by the storage: %s", err)
	}
	timestampColumn := TimestampColumn{
		Name:   file.action.UpdatedAtColumn,
		Format: file.action.UpdatedAtFormat,
	}
	rw := newRecordWriter(file.action.Connector().ID, file.action.InSchema,
		file.action.UniqueIDColumn, timestampColumn, file.action.DisplayedID,
		storageTimestamp, math.MaxInt)
	records := &fileRecords{
		ctx:   ctx,
		rw:    rw,
		rc:    rc,
		sheet: file.action.Sheet,
		inner: file.inner,
	}
	return records, nil
}

// Writer returns a Writer for writing records into the file located at the path
// of the file's action. schema contains the properties of the records to be
// written.
//
// If pathReplacer is not nil, then the placeholders in path are replaced using
// it; in this case, a PlaceholderError error may be returned in case of an
// error with placeholders.
func (file *File) Writer(ctx context.Context, schema types.Type, ack AckFunc, pathReplacer PlaceholderReplacer) (Writer, error) {
	if file.err != nil {
		return nil, file.err
	}
	if ack == nil {
		return nil, errors.New("ack function is missing")
	}
	storage, err := file.storage()
	if err != nil {
		return nil, err
	}
	s := newCompressedStorage(storage, file.action.Compression)
	extension := file.action.Connector().FileExtension
	path := file.action.Path
	if pathReplacer != nil {
		var err error
		path, err = ReplacePlaceholders(path, pathReplacer)
		if err != nil {
			return nil, err
		}
	}
	sw, err := s.Writer(ctx, path, file.inner.ContentType(ctx), extension)
	if err != nil {
		return nil, err
	}
	columns := schema.Properties()
	records := make(chan fileRecord, 100)
	result := make(chan error, 1)
	writeCtx, cancelWrite := context.WithCancel(context.Background())
	// Call the connector's Write method in its own goroutine.
	go func() {
		r := newRecordReader(columns, records, ack)
		err = file.inner.Write(writeCtx, sw, file.action.Sheet, r)
		if err2 := sw.CloseWithError(err); err2 != nil && err == nil {
			err = err2
		}
		result <- err
	}()
	fw := &fileWriter{
		cancelWrite: cancelWrite,
		columns:     columns,
		records:     records,
		result:      result,
	}
	return fw, nil
}

// fileWriter implements the Writer interface for files.
type fileWriter struct {
	cancelWrite context.CancelFunc
	columns     []types.Property
	records     chan<- fileRecord
	result      <-chan error
	closed      bool
	err         error
}

func (w *fileWriter) Close(ctx context.Context) error {
	if w.closed {
		return w.err
	}
	w.cancelWrite()
	w.closed = true
	return nil
}

func (w *fileWriter) Commit(ctx context.Context) error {
	if w.closed {
		panic("connectors: Commit called on a closed writer")
	}
	close(w.records)
	var err error
	select {
	case err = <-w.result:
		// The connector has terminated with or without an error.
	case <-ctx.Done():
		// The context has been canceled.
		w.cancelWrite()
		err = <-w.result
	}
	w.closed = true
	return err
}

func (w *fileWriter) Write(ctx context.Context, gid int, record Record) bool {
	if w.closed {
		panic("connectors: Write called on a closed writer")
	}
	r := fileRecord{
		gid:    gid,
		record: make([]any, len(w.columns)),
	}
	for i, c := range w.columns {
		r.record[i] = record.Properties[c.Name]
	}
	select {
	case w.records <- r:
		// The row has been sent to the connector.
		return true
	case err := <-w.result:
		// The connector has returned early with an error.
		w.err = err
		return false
	case <-ctx.Done():
		// The context has been canceled.
		w.err = ctx.Err()
		return false
	}
}

// storage returns the inner storage connection of the file.
func (file *File) storage() (chichi.FileStorage, error) {
	conn := file.action.Connection()
	connector := file.action.Connection().Connector()
	return chichi.RegisteredFileStorage(connector.Name).New(&chichi.FileStorageConfig{
		Role:        chichi.Role(conn.Role),
		Settings:    conn.Settings,
		SetSettings: setConnectionSettingsFunc(file.state, conn),
	})
}

// fileRecords implements the Records interface for files.
type fileRecords struct {
	ctx    context.Context
	rw     *recordWriter
	rc     io.ReadCloser
	sheet  string
	inner  chichi.File
	err    error
	closed bool
}

func (r *fileRecords) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	err := r.rc.Close()
	if err != nil && r.err == nil {
		r.err = err
	}
	return err
}

func (r *fileRecords) Err() error {
	return r.err
}

func (r *fileRecords) For(yield func(Record) error) error {
	if r.closed {
		r.err = errors.New("connectors: For called on a closed Records")
		return nil
	}
	defer func() {
		_ = r.Close()
		if r.err == nil && r.rw.properties == nil {
			r.err = ErrNoColumns
		}
	}()
	r.rw.yield = yield
	err := r.inner.Read(r.ctx, r.rc, r.sheet, r.rw)
	if err != nil && err != errRecordStop {
		if err, ok := err.(yieldError); ok {
			return err.err
		}
		if err == chichi.ErrSheetNotExist {
			err = ErrSheetNotExist
		}
		r.err = err
	}
	return nil
}

var (
	// errRecordStop is returned by recordWriter methods when the maximum row
	// limit has been reached, signaling the need to stop writing rows.
	errRecordStop = errors.New("stop record")

	// errReadStopped is returned to a file connector when it calls w.Write and the
	// storage has already finished reading without an error.
	// If this error occurs, it indicates a bug in the storage connector.
	errReadStopped = errors.New("storage abruptly stopped reading")
)

// newRecordReader returns a new record reader that read records.
func newRecordReader(columns []types.Property, records <-chan fileRecord, ack AckFunc) *recordReader {
	return &recordReader{
		columns: columns,
		records: records,
		ack:     ack,
	}
}

type fileRecord struct {
	gid    int
	record []any
}

// recordReader implements the connector.RecordReader interface.
type recordReader struct {
	columns []types.Property
	records <-chan fileRecord
	ack     AckFunc
}

// Ack acknowledges the processing of the record with the given GID.
// err is the error occurred processing the record, if any.
func (rr *recordReader) Ack(gid int, err error) {
	rr.ack(err, []int{gid})
}

// Columns returns the columns of the records.
func (rr *recordReader) Columns() []types.Property {
	return rr.columns
}

// Record returns the next record as a slice of any. It returns nil and io.EOF
// if there are no more records.
func (rr *recordReader) Record(ctx context.Context) (int, []any, error) {
	select {
	case r, ok := <-rr.records:
		if !ok {
			return 0, nil, io.EOF
		}
		return r.gid, r.record, nil
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	}
}

// newRecordWriter returns a new record writer that writes at most limit
// records. If the yield function is not nil, it calls the yield function for
// each record, otherwise it stores the records in the records field.
// storageTimestamp is the timestamp provided by the storage connector, and it
// is used in the case when the file columns do not specify a timestamp.
func newRecordWriter(connector int, schema types.Type, uniqueIDColumn string, timestamp TimestampColumn, displayedID string, storageTimestamp time.Time, limit int) *recordWriter {
	rw := recordWriter{
		connector:       connector,
		schema:          schema,
		limit:           limit,
		textColumnsOnly: true,
		records:         []map[string]any{},
	}
	rw.displayedID.name = displayedID
	if uniqueIDColumn != "" {
		rw.uniqueIDColumn.name = uniqueIDColumn
		typ, _ := schema.Property(uniqueIDColumn)
		rw.uniqueIDColumn.typ = typ.Type
	}
	if timestamp.Name != "" {
		rw.timestampColumn.name = timestamp.Name
		typ, _ := schema.Property(timestamp.Name)
		rw.timestampColumn.typ = typ.Type
		rw.timestampColumn.format = timestamp.Format
	} else {
		rw.storageTimestamp = storageTimestamp
	}
	return &rw
}

// recordWriter implements the connector.RecordWriter interface.
type recordWriter struct {
	connector       int
	limit           int
	yield           func(Record) error
	schema          types.Type
	properties      []types.Property // schema's properties, or the file's columns if a schema has not been provided
	columnIndexOf   map[int]int      // map a property index in the schema to the corresponding file's column
	columns         int              // number of file's columns
	textColumnsOnly bool
	displayedID     struct {
		name   string
		column types.Property
		index  int
	}
	uniqueIDColumn struct {
		name  string
		typ   types.Type
		index int
	}
	timestampColumn struct {
		name   string
		typ    types.Type
		index  int
		format string
	}
	storageTimestamp time.Time
	records          []map[string]any
}

// Columns sets the columns of the records as properties.
// Columns must be called before Record, RecordMap and RecordString.
func (rw *recordWriter) Columns(columns []types.Property) error {
	if rw.properties != nil {
		return fmt.Errorf("connector %d has called Columns twice", rw.connector)
	}
	if len(columns) == 0 {
		return fmt.Errorf("connector %d has called Columns with an empty columns", rw.connector)
	}
	fileSchema, err := types.ObjectOf(columns)
	if err != nil {
		return fmt.Errorf("connector %d has returned invalid columns: %s", rw.connector, err)
	}
	columnByName := make(map[string]types.Property, len(columns))
	columnIndex := make(map[string]int, len(columns))
	for i, c := range columns {
		columnByName[c.Name] = c
		columnIndex[c.Name] = i
		if rw.textColumnsOnly {
			rw.textColumnsOnly = c.Type.Kind() == types.TextKind
		}
	}
	// Validate the unique ID column.
	if name := rw.uniqueIDColumn.name; name != "" {
		c, ok := columnByName[name]
		if !ok {
			return fmt.Errorf("there is no unique ID column %q", name)
		}
		if typ := rw.uniqueIDColumn.typ; c.Type.Kind() != typ.Kind() {
			return fmt.Errorf("unique ID column %q has type %s instead of %s", c.Name, c.Type.Kind(), typ.Kind())
		}
		rw.uniqueIDColumn.typ = c.Type
		rw.uniqueIDColumn.index = columnIndex[c.Name]
	}
	// Validate the timestamp column.
	if name := rw.timestampColumn.name; name != "" {
		c, ok := columnByName[name]
		if !ok {
			return fmt.Errorf("there is no timestamp column %q", name)
		}
		if typ := rw.timestampColumn.typ; c.Type.Kind() != typ.Kind() {
			return fmt.Errorf("timestamp column %q has type %s instead of %s", c.Name, c.Type.Kind(), typ.Kind())
		}
		rw.timestampColumn.typ = c.Type
		rw.timestampColumn.index = columnIndex[c.Name]
	}
	// Validate the displayed ID column.
	if rw.displayedID.name != "" {
		col, err := displayedIDFromSchema(fileSchema, rw.displayedID.name)
		if err != nil {
			slog.Warn("cannot determine the displayed ID column", "err", err)
			rw.displayedID.name = ""
		} else {
			rw.displayedID.column = col
			rw.displayedID.index = columnIndex[col.Name]
		}
	}
	// Check that the schema, if valid, is compatible with the file's schema.
	if rw.schema.Valid() {
		err := checkConformity("", rw.schema, fileSchema)
		if err != nil {
			return err
		}
		rw.properties = rw.schema.Properties()
		rw.columnIndexOf = make(map[int]int, len(rw.properties))
		for i, c := range rw.properties {
			rw.columnIndexOf[i] = columnIndex[c.Name]
		}
	} else {
		rw.properties = columns
	}
	rw.columns = len(columns)
	if rw.limit == 0 {
		return errRecordStop
	}
	return nil
}

// Record writes a record.
func (rw *recordWriter) Record(record []any) error {
	if rw.properties == nil {
		return fmt.Errorf("connector %d did not call the Columns method before calling Record", rw.connector)
	}
	if len(record) != rw.columns {
		return fmt.Errorf("connector %d has returned records with different lengths", rw.connector)
	}
	var err error
	if rw.yield == nil {
		// Store the record in the records field.
		rd := make(map[string]any, len(rw.properties))
		for i, c := range rw.properties {
			rd[c.Name], err = normalizeDatabaseFileProperty(c.Name, c.Type, record[i], c.Nullable)
			if err != nil {
				return err
			}
		}
		rw.records = append(rw.records, rd)
	} else {
		// Call the rw.write function to store the record.
		rd := Record{Properties: map[string]any{}}
		for i, c := range rw.properties {
			j := rw.columnIndexOf[i]
			value, err := normalizeDatabaseFileProperty(c.Name, c.Type, record[j], c.Nullable)
			if err != nil {
				rd.Err = err
				break
			}
			rd.Properties[c.Name] = value
		}
		// Parse the unique ID column.
		rd.ID, err = parseUniqueIDColumn(rw.uniqueIDColumn.name, rw.uniqueIDColumn.typ, record[rw.uniqueIDColumn.index])
		if err != nil {
			if rd.Err != nil {
				rd.Err = err
			}
		}
		// Parse the timestamp column.
		if rd.Err == nil {
			if rw.timestampColumn.name != "" {
				ts := record[rw.timestampColumn.index]
				rd.Timestamp, err = parseTimestampColumn(rw.timestampColumn.name, rw.timestampColumn.typ, rw.timestampColumn.format, ts)
				if err != nil {
					rd.Err = err
				}
			} else {
				rd.Timestamp = rw.storageTimestamp
			}
		}
		// Parse the displayed ID column.
		if rd.Err == nil && rw.displayedID.name != "" {
			v := record[rw.displayedID.index]
			c := rw.displayedID.column
			vv, err := normalizeDatabaseFileProperty(c.Name, c.Type, v, c.Nullable)
			if err != nil {
				slog.Warn("the displayed ID value cannot be normalized", "err", err)
			} else {
				s, err := displayedIDToString(vv)
				if err != nil {
					slog.Warn("invalid displayed ID value", "err", err)
				} else {
					rd.DisplayedID = s
				}
			}
		}
		if err := rw.yield(rd); err != nil {
			return yieldError{err: err}
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
	if rw.properties == nil {
		return fmt.Errorf("connector %d did not call the Columns method before calling RecordMap", rw.connector)
	}
	var err error
	if rw.yield == nil {
		// Store the record in the records field.
		rd := make(map[string]any, len(rw.properties))
		for _, c := range rw.properties {
			rd[c.Name], err = normalizeDatabaseFileProperty(c.Name, c.Type, record[c.Name], c.Nullable)
			if err != nil {
				return err
			}
		}
		rw.records = append(rw.records, rd)
	} else {
		// Call the rw.write function to store the record.
		rd := Record{Properties: record}
		for _, c := range rw.properties {
			value, err := normalizeDatabaseFileProperty(c.Name, c.Type, record[c.Name], c.Nullable)
			if err != nil {
				rd.Err = err
				break
			}
			rd.Properties[c.Name] = value
		}
		// Parse the unique ID column.
		rd.ID, err = parseUniqueIDColumn(rw.uniqueIDColumn.name, rw.uniqueIDColumn.typ, record[rw.uniqueIDColumn.name])
		if err != nil {
			if rd.Err != nil {
				rd.Err = err
			}
		}
		// Parse the timestamp column.
		if rd.Err == nil {
			if rw.timestampColumn.name != "" {
				ts := record[rw.timestampColumn.name]
				rd.Timestamp, err = parseTimestampColumn(rw.timestampColumn.name, rw.timestampColumn.typ, rw.timestampColumn.format, ts)
				if err != nil {
					rd.Err = err
				}
			} else {
				rd.Timestamp = rw.storageTimestamp
			}
		}
		if err := rw.yield(rd); err != nil {
			return yieldError{err: err}
		}
		// Parse the displayed ID column.
		if rd.Err == nil && rw.displayedID.name != "" {
			v := record[rw.displayedID.name]
			c := rw.displayedID.column
			vv, err := normalizeDatabaseFileProperty(c.Name, c.Type, v, c.Nullable)
			if err != nil {
				slog.Warn("displayed ID value cannot be normalized", "err", err)
			} else {
				s, err := displayedIDToString(vv)
				if err != nil {
					slog.Warn("invalid displayed ID value", "err", err)
				} else {
					rd.DisplayedID = s
				}
			}
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
	if rw.properties == nil {
		return fmt.Errorf("connector %d did not call the Columns method before calling RecordString", rw.connector)
	}
	if len(record) != rw.columns {
		return fmt.Errorf("connector %d has returned records with different lengths", rw.connector)
	}
	if !rw.textColumnsOnly {
		return fmt.Errorf("connector %d has called RecordString when there are non-text columns", rw.connector)
	}
	var err error
	if rw.yield == nil {
		// Store the record in the records field.
		rd := make(map[string]any, len(rw.properties))
		for i, c := range rw.properties {
			err = validateStringProperty(c, record[i])
			if err != nil {
				return err
			}
			rd[c.Name] = record[i]
		}
		rw.records = append(rw.records, rd)
	} else {
		// Call the rw.write function to store the record.
		rd := Record{Properties: make(map[string]any, len(rw.properties))}
		for i, c := range rw.properties {
			j := rw.columnIndexOf[i]
			value := record[j]
			err = validateStringProperty(c, value)
			if err != nil {
				rd.Err = err
				break
			}
			rd.Properties[c.Name] = value
		}
		// Parse the unique ID column.
		rd.ID, err = parseUniqueIDColumn(rw.uniqueIDColumn.name, rw.uniqueIDColumn.typ, record[rw.uniqueIDColumn.index])
		if err != nil {
			if rd.Err != nil {
				rd.Err = err
			}
		}
		// Parse the timestamp column.
		if rd.Err == nil {
			if rw.timestampColumn.name != "" {
				ts := record[rw.timestampColumn.index]
				rd.Timestamp, err = parseTimestampColumn(rw.timestampColumn.name, rw.timestampColumn.typ, rw.timestampColumn.format, ts)
				if err != nil {
					rd.Err = err
				}
			} else {
				rd.Timestamp = rw.storageTimestamp
			}
		}
		// Parse the displayed ID column.
		if rd.Err == nil && rw.displayedID.name != "" {
			v := record[rw.displayedID.index]
			c := rw.displayedID.column
			vv, err := normalizeDatabaseFileProperty(c.Name, c.Type, v, c.Nullable)
			if err != nil {
				slog.Warn("displayed ID value cannot be normalized", "err", err)
			} else {
				s, err := displayedIDToString(vv)
				if err != nil {
					slog.Warn("invalid displayed ID value", "err", err)
				} else {
					rd.DisplayedID = s
				}
			}
		}
		if err := rw.yield(rd); err != nil {
			return yieldError{err: err}
		}
	}
	rw.limit--
	if rw.limit == 0 {
		return errRecordStop
	}
	return nil
}

// IsValidSheetName reports whether name is a valid sheet name.
func IsValidSheetName(name string) bool {
	const maxLength = 31
	if !utf8.ValidString(name) {
		return false
	}
	if length := utf8.RuneCountInString(name); length < 1 || length > maxLength {
		return false
	}
	if name[0] == '\'' || name[len(name)-1] == '\'' {
		return false
	}
	if strings.ContainsAny(name, `*/:?[\]`) {
		return false
	}
	return true
}

// parseUniqueIDColumn parses a unique ID column value.
func parseUniqueIDColumn(name string, typ types.Type, value any) (string, error) {
	id, err := normalizeDatabaseFileProperty(name, typ, value, false)
	if err != nil {
		return "", err
	}
	switch id := id.(type) {
	case nil:
		return "", fmt.Errorf("identify value is null")
	case int:

		return strconv.FormatInt(int64(id), 10), nil
	case uint:
		return strconv.FormatUint(uint64(id), 10), nil
	case string:
		if id == "" {
			return "", fmt.Errorf("identify value is empty")
		}
		return id, nil
	case float64:
		if int(math.Round(id)) == int(id) {
			return strconv.FormatInt(int64(id), 10), nil
		}
	case json.Number:
		var n int64
		err := json.Unmarshal([]byte(id), &n)
		if err == nil {
			return strconv.FormatInt(n, 10), nil
		}
	case json.RawMessage:
		if id[0] == '"' {
			var s string
			_ = json.Unmarshal(id, &s)
			if s == "" {
				return "", fmt.Errorf("identify value is empty")
			}
			return s, nil
		} else {
			var n int64
			err := json.Unmarshal(id, &n)
			if err == nil {
				return strconv.FormatInt(n, 10), nil
			}
		}
	}
	return "", fmt.Errorf("identify value is not a JSON string or JSON integer number")
}

// parseTimestampColumn parses a timestamp column value. If the timestamp cannot
// be parsed or it is not valid, returns an error.
//
// To see a list of accepted format values, see the documentation of
// 'parseTimestamp'.
func parseTimestampColumn(name string, typ types.Type, format string, value any) (time.Time, error) {
	timestamp, err := normalizeDatabaseFileProperty(name, typ, value, false)
	if err != nil {
		return time.Time{}, err
	}
	switch timestamp := timestamp.(type) {
	case nil:
		return time.Time{}, errors.New("timestamp value is null")
	case time.Time:
		err = validateTimestamp(timestamp)
		if err != nil {
			return time.Time{}, err
		}
		return timestamp, nil
	case string:
		ts, err := parseTimestamp(format, value.(string))
		if err != nil {
			return time.Time{}, fmt.Errorf("timestamp %q does not conform to the %q format", value, format)
		}
		err = validateTimestamp(ts)
		if err != nil {
			return time.Time{}, err
		}
		return ts, nil
	case json.RawMessage:
		var s string
		err := json.Unmarshal(timestamp, &s)
		if err != nil {
			return time.Time{}, fmt.Errorf("timestamp value is not a JSON string")
		}
		ts, err := parseTimestamp(format, value.(string))
		if err != nil {
			return time.Time{}, fmt.Errorf("timestamp %q does not conform to the %q format", value, format)
		}
		err = validateTimestamp(ts)
		if err != nil {
			return time.Time{}, err
		}
		return ts, nil
	}
	return time.Time{}, fmt.Errorf("timestamp value is not a JSON string")
}

var excelEpoch = time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC)

// parseTimestamp parses a timestamp with the given format.
//
// Accepted values for format are:
//
//   - "DateTime", to parse timestamps in the format "2006-01-02 15:04:05"
//   - "DateOnly", to parse date-only timestamps in the format "2006-01-02"
//   - "ISO8601", to parse the timestamp as a ISO 8601 timestamp.
//   - "Excel", to parse the timestamp as a string representing a float value
//     stored in a Excel cell representing a date / datetime.
//   - a strptime format, enclosed by single quote characters, compatible with
//     the standard C89 functions strptime/strftime.
//
// NOTE: keep in sync with the function 'apis.validateTimestampFormat'.
func parseTimestamp(format, timestamp string) (time.Time, error) {
	switch format {
	case "DateTime":
		dt, err := time.Parse("2006-01-02 15:04:05", timestamp)
		if err != nil {
			return time.Time{}, fmt.Errorf("timestamp has not the format '2006-01-02 15:04:05'")
		}
		return dt.UTC(), nil
	case "DateOnly":
		date, err := time.Parse("2006-01-02", timestamp)
		if err != nil {
			return time.Time{}, fmt.Errorf("timestamp has not the format '2006-01-02'")
		}
		return date.UTC(), nil
	case "ISO8601":
		dt, err := iso8601.ParseString(timestamp)
		if err != nil {
			return time.Time{}, errors.New("timestamp format is not compatible with ISO 8601")
		}
		return dt.UTC(), err
	case "Excel":
		if !isExcelSimpleFloat(timestamp) {
			return time.Time{}, errors.New("invalid timestamp for Excel")
		}
		// Parse as Excel serial date-time.
		// https://support.microsoft.com/en-us/office/datetime-function-812ad674-f7dd-4f31-9245-e79cfa358a4e
		// https://support.microsoft.com/en-us/office/datevalue-function-df8b07d4-7761-4a93-bc33-b7471bbff252
		days, err := strconv.ParseFloat(timestamp, 64)
		if err != nil {
			return time.Time{}, errors.New("invalid timestamp for Excel")
		}
		if days == 60 {
			// 1900-02-29 does not exist. Excel returns it for compatibility with Lotus 1-2-3.
			return time.Time{}, errors.New("invalid timestamp for Excel")
		}
		if days > 60 {
			days--
		}
		d := time.Duration(days * 24 * 3600 * 1e9)
		t := excelEpoch.Add(d)
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC), nil
	default: // a format compatible with strptime, for example: '%Y-%m-%d'.
		f, ok := strings.CutPrefix(format, "'")
		if !ok {
			return time.Time{}, fmt.Errorf("invalid format %q", format)
		}
		f, ok = strings.CutSuffix(f, "'")
		if !ok {
			return time.Time{}, fmt.Errorf("invalid format %q", format)
		}
		t, err := timefmt.Parse(timestamp, f)
		if err != nil {
			return time.Time{}, err
		}
		return t.UTC(), nil
	}
}

// isExcelSimpleFloat reports whether s is a string representing a float value
// encoding an Excel date / datetime value.
func isExcelSimpleFloat(s string) bool {
	// NOTE: keep in sync with the function within 'apis/transformers/mappings'.
	if len(s) < 3 {
		return false
	}
	var dot bool
	for i, c := range []byte(s) {
		if c == '.' {
			if dot || i == 0 || i == len(s)-1 {
				return false
			}
			dot = true
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// compressorStorage implements a storage capable of compressing and
// decompressing data read from or written to a FileStorage.
type compressorStorage struct {
	storage     chichi.FileStorage
	compression state.Compression
}

// newCompressedStorage returns a compressor storage that wraps s and performs
// file compression and decompression using c as the compression method.
// If c is NoCompression, it does not perform any compression or decompression.
func newCompressedStorage(s chichi.FileStorage, c state.Compression) *compressorStorage {
	return &compressorStorage{s, c}
}

// Reader opens the file at the provided path name and returns an io.ReadCloser
// from which to read the file and its timestamp.
// It is the caller's responsibility to close the returned reader.
func (cs compressorStorage) Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error) {
	r, t, err := cs.storage.Reader(ctx, name)
	if err != nil {
		return nil, time.Time{}, err
	}
	switch cs.compression {
	case state.ZipCompression:
		var err error
		var fi *os.File
		var r2 *zip.ReadCloser
		defer func() {
			if err != nil {
				if r2 != nil {
					_ = r2.Close()
				}
				if fi != nil {
					_ = removeTempFile(fi)
				}
				if r != nil {
					_ = r.Close()
				}
			}
		}()
		fi, err = os.CreateTemp("", "")
		if err != nil {
			return nil, time.Time{}, err
		}
		_, err = io.Copy(fi, r)
		if err != nil {
			return nil, time.Time{}, err
		}
		err = r.Close()
		r = nil
		if err != nil {
			return nil, time.Time{}, err
		}
		_, err = fi.Seek(0, io.SeekStart)
		if err != nil {
			return nil, time.Time{}, err
		}
		r2, err = zip.OpenReader(fi.Name())
		if err != nil {
			return nil, time.Time{}, err
		}
		var r3 io.ReadCloser
		for _, file := range r2.File {
			// Skip directories.
			if strings.HasSuffix(file.Name, "/") {
				continue
			}
			if r3 != nil {
				return nil, time.Time{}, errors.New("the ZIP archive contains not just a single file, but multiple files")
			}
			r3, err = file.Open()
			if err != nil {
				return nil, time.Time{}, err
			}
			t = file.Modified
		}
		if r3 == nil {
			return nil, time.Time{}, errors.New("the ZIP archive does not contain any files")
		}
		r = newFuncReadCloser(r3, func() error {
			err3 := r3.Close()
			err2 := r2.Close()
			err := removeTempFile(fi)
			if err3 != nil {
				return err3
			}
			if err2 != nil {
				return err2
			}
			return err
		})
	case state.GzipCompression:
		r2, err := gzip.NewReader(r)
		if err != nil {
			_ = r.Close()
			return nil, time.Time{}, err
		}
		r1 := r
		r = newFuncReadCloser(r2, func() error {
			err2 := r2.Close()
			err := r1.Close()
			if err2 != nil {
				return err2
			}
			return err
		})
	case state.SnappyCompression:
		r2 := snappy.NewReader(r)
		r1 := r
		r = newFuncReadCloser(r2, func() error {
			return r1.Close()
		})
	}
	return r, t, nil
}

// Writer returns a Writer that compress the data if needed, and then writes it
// directly to the underlying storage.
//
// If the data should be compressed, it passes path to the underlying storage
// with an appended extension, and an appropriate content type.
//
// It is the caller's responsibility to call Close on the returned Writer.
func (cs compressorStorage) Writer(ctx context.Context, path, contentType, extension string) (*storageWriteCloser, error) {
	pr, pw := io.Pipe()
	var w io.WriteCloser
	switch cs.compression {
	case state.NoCompression:
		w = pw
	case state.ZipCompression:
		z := zip.NewWriter(pw)
		name := pathPkg.Base(path)
		if ext := strings.ToLower(pathPkg.Ext(name)); ext == "" {
			name += "." + extension
		} else if ext == "." {
			name += extension
		} else if ext[1:] != extension {
			name = strings.TrimSuffix(name, ext[1:]) + extension
		}
		zw, err := z.Create(name)
		if err != nil {
			_ = z.Close()
			_ = pr.Close()
			_ = pw.Close()
			return nil, err
		}
		w = zipWriter{Writer: zw, z: z}
	case state.GzipCompression:
		w = gzip.NewWriter(pw)
	case state.SnappyCompression:
		w = snappy.NewBufferedWriter(pw)
	}
	path += cs.compression.Ext()
	if ct := cs.compression.ContentType(); ct != "" {
		contentType = ct
	}
	ch := make(chan error)
	go func() {
		ctx, cancel := context.WithCancel(ctx)
		r := newTimeoutReader(pr, storageTimeout, cancel)
		defer r.Close()
		err := cs.storage.Write(ctx, r, path, contentType)
		if err != nil {
			_ = pr.CloseWithError(err)
		} else {
			// errReadStopped will be returned to the file connector only if it
			// calls w.Write when the storage is returned.
			_ = pr.CloseWithError(errReadStopped)
		}
		ch <- err
	}()
	wc := newFuncWriteCloser(w, func(err error) error {
		if w != pw {
			err2 := w.Close()
			if err == nil {
				err = err2
			}
		}
		_ = pw.CloseWithError(err)
		return <-ch
	})
	return wc, nil
}

// removeTempFile removes fi and returns the error, if any. Any error
// encountered will be logged.
func removeTempFile(fi *os.File) error {
	err := fi.Close()
	if err := os.Remove(fi.Name()); err != nil {
		slog.Warn("cannot remove temporary file", "path", fi.Name(), "err", err)
	}
	return err
}

// funcReadCloser wraps an io.Reader and implements io.ReadCloser. It calls a
// specified function when Close is invoked.
type funcReadCloser struct {
	io.Reader
	close func() error
}

// newFuncReadCloser returns an io.ReadCloser that wraps r and calls close when
// Close is invoked.
func newFuncReadCloser(r io.Reader, close func() error) io.ReadCloser {
	return funcReadCloser{r, close}
}

func (c funcReadCloser) Close() error {
	return c.close()
}

// storageWriteCloser wraps an io.Writer and implements io.WriteCloser. It calls a
// specified function when Close is invoked.
type storageWriteCloser struct {
	io.Writer
	close func(err error) error
}

// newFuncWriteCloser returns an io.WriteCloser that wraps w and calls close
// when Close is invoked.
func newFuncWriteCloser(w io.Writer, close func(err error) error) *storageWriteCloser {
	return &storageWriteCloser{w, close}
}

// Close closes the underlying writer. Storage will receive io.EOF from a read.
// It returns the error returned by the storage if any.
func (c storageWriteCloser) Close() error {
	return c.close(nil)
}

// CloseWithError closes the underlying writer. Storage will receive err as
// error from a read, or io.EOF is err is nil.
// It returns the error returned by the storage if any.
func (c storageWriteCloser) CloseWithError(err error) error {
	return c.close(err)
}

// zipWriter wraps a Writer and implements the Close method that closes a
// zip.Writer when called.
type zipWriter struct {
	z *zip.Writer
	io.Writer
}

func (zw zipWriter) Close() error {
	return zw.z.Close()
}

// timeoutReader implements an io.ReadCloser with a timeout between two
// consecutive Read calls.
type timeoutReader struct {
	reader  io.Reader
	timeout time.Duration
	timer   *time.Timer
	f       func()
	stop    chan struct{}
	closed  bool
}

func (r *timeoutReader) Read(p []byte) (int, error) {
	if r.closed {
		return 0, errors.New("read on a closed reader")
	}
	r.timer.Stop()
	n, err := r.reader.Read(p)
	r.timer.Reset(r.timeout)
	return n, err
}

func (r *timeoutReader) Close() error {
	if r.closed {
		return nil
	}
	r.stop <- struct{}{}
	r.closed = true
	return nil
}

// newTimeoutReader returns a TimeoutReader that reads from r. If more than
// timeout time elapses between two Read method calls, it calls the f function.
// The caller must close the returned reader using the Close method when no
// further calls to the Read method are expected.
func newTimeoutReader(r io.Reader, timeout time.Duration, f func()) io.ReadCloser {
	stop := make(chan struct{})
	timer := time.NewTimer(timeout)
	go func() {
		select {
		case <-timer.C:
			close(stop)
			f()
		case <-stop:
		}
	}()
	return &timeoutReader{reader: r, timeout: timeout, timer: timer, f: f, stop: stop}
}
