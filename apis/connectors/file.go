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
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	pathPkg "path"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/types"

	"github.com/golang/snappy"
	"github.com/google/uuid"
)

// storageTimeout represents the duration between consecutive calls to the Read
// method of the io.Reader passed to a storage within the Write method.
var storageTimeout = 10 * time.Second

type File struct {
	state       *state.State
	action      *state.Action
	timeLayouts *state.TimeLayouts
	inner       chichi.File
	err         error
}

// File returns a file for the provided action, on a connection with the given
// role. Errors are deferred until a file's method is called.
func (connectors *Connectors) File(action *state.Action, role state.Role) *File {
	connector := action.Connector()
	file := &File{
		state:       connectors.state,
		action:      action,
		timeLayouts: &connector.TimeLayouts,
	}
	file.inner, file.err = chichi.RegisteredFile(connector.Name).New(&chichi.FileConfig{
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
// If the identity property specified in the action of the file is found within
// the file schema but its type is different, the iterator will return an error.
// The same applies for the last change time property, if specified.
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
	rc, storageLastChangeTime, err := s.Reader(ctx, file.action.Path)
	if err != nil {
		return nil, err
	}
	if err = validateTimestamp(storageLastChangeTime); err != nil {
		return nil, fmt.Errorf("invalid last change time returned by the storage: %s", err)
	}
	lastChangeTimeProperty := LastChangeTimeProperty{
		Name:   file.action.LastChangeTimeProperty,
		Format: file.action.LastChangeTimeFormat,
	}
	rw := newRecordWriter(file.action.Connector().Name, file.action.InSchema,
		file.action.IdentityProperty, lastChangeTimeProperty, file.action.DisplayedProperty,
		storageLastChangeTime, file.timeLayouts, math.MaxInt)
	records := &fileRecords{
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
	columns := types.Properties(schema)
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

func (w *fileWriter) Write(ctx context.Context, gid uuid.UUID, record Record) bool {
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
		Settings:    conn.Settings,
		SetSettings: setConnectionSettingsFunc(file.state, conn),
	})
}

// fileRecords implements the Records interface for files.
type fileRecords struct {
	rw     *recordWriter
	rc     io.ReadCloser
	sheet  string
	inner  chichi.File
	last   bool
	err    error
	closed bool
}

func (r *fileRecords) All(ctx context.Context) Seq[Record] {
	return func(yield func(Record) bool) {
		if r.closed {
			r.err = errors.New("connectors: For called on a closed Records")
			return
		}
		defer func() {
			_ = r.Close()
			if r.err == nil && r.rw.properties == nil {
				r.err = ErrNoColumns
			}
		}()
		r.rw.yield = yield
		err := r.inner.Read(ctx, r.rc, r.sheet, r.rw)
		if r.rw.record.Properties != nil || r.rw.record.Err != nil {
			r.last = true
			r.rw.yield(r.rw.record)
		}
		if err != nil && err != errRecordStop {
			if err == chichi.ErrSheetNotExist {
				err = ErrSheetNotExist
			}
			r.err = err
		}
	}
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

func (r *fileRecords) Last() bool {
	return r.last
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
	gid    uuid.UUID
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
func (rr *recordReader) Ack(gid uuid.UUID, err error) {
	rr.ack(err, []uuid.UUID{gid})
}

// Columns returns the columns of the records.
func (rr *recordReader) Columns() []types.Property {
	return rr.columns
}

// Record returns the next record as a slice of any. It returns uuid.UUID{}, nil
// and io.EOF if there are no more records.
func (rr *recordReader) Record(ctx context.Context) (uuid.UUID, []any, error) {
	select {
	case r, ok := <-rr.records:
		if !ok {
			return uuid.UUID{}, nil, io.EOF
		}
		return r.gid, r.record, nil
	case <-ctx.Done():
		return uuid.UUID{}, nil, ctx.Err()
	}
}

// newRecordWriter returns a new record writer that writes at most limit
// records. If the yield function is not nil, it calls the yield function for
// each record, otherwise it stores the records in the records field.
// storageLastChangeTime is the lat change time provided by the storage
// connector, and it is used in the case when the file columns do not specify a
// last change time property.
func newRecordWriter(connector string, schema types.Type, identityProperty string, lastChangeTime LastChangeTimeProperty, displayedProperty string, storageLastChangeTime time.Time, layout *state.TimeLayouts, limit int) *recordWriter {
	rw := recordWriter{
		connector:       connector,
		schema:          schema,
		limit:           limit,
		textColumnsOnly: true,
		timeLayouts:     layout,
		records:         []map[string]any{},
	}
	rw.displayedProperty.name = displayedProperty
	if identityProperty != "" {
		rw.identityProperty.name = identityProperty
		typ, _ := schema.Property(identityProperty)
		rw.identityProperty.typ = typ.Type
	}
	if lastChangeTime.Name != "" {
		rw.lastChangeTimeProperty.name = lastChangeTime.Name
		typ, _ := schema.Property(lastChangeTime.Name)
		rw.lastChangeTimeProperty.typ = typ.Type
		rw.lastChangeTimeProperty.format = lastChangeTime.Format
	} else {
		rw.storageLastChangeTime = storageLastChangeTime
	}
	return &rw
}

// recordWriter implements the connector.RecordWriter interface.
type recordWriter struct {
	connector         string
	limit             int
	record            Record
	yield             func(Record) bool
	schema            types.Type
	properties        []types.Property // schema's properties, or the file's columns if a schema has not been provided
	columnIndexOf     map[int]int      // map a property index in the schema to the corresponding file's column
	columns           int              // number of file's columns
	textColumnsOnly   bool
	displayedProperty struct {
		name   string
		column types.Property
		index  int
	}
	identityProperty struct {
		name  string
		typ   types.Type
		index int
	}
	lastChangeTimeProperty struct {
		name   string
		typ    types.Type
		index  int
		format string
	}
	storageLastChangeTime time.Time
	timeLayouts           *state.TimeLayouts
	records               []map[string]any
}

// Columns sets the columns of the records as properties.
// Columns must be called before Record, RecordMap and RecordString.
func (rw *recordWriter) Columns(columns []types.Property) error {
	if rw.properties != nil {
		return fmt.Errorf("connector %s has called Columns twice", rw.connector)
	}
	if len(columns) == 0 {
		return fmt.Errorf("connector %s has called Columns with an empty columns", rw.connector)
	}
	fileSchema, err := types.ObjectOf(columns)
	if err != nil {
		return fmt.Errorf("connector %s has returned invalid columns: %s", rw.connector, err)
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
	// Validate the identity property.
	if name := rw.identityProperty.name; name != "" {
		c, ok := columnByName[name]
		if !ok {
			return fmt.Errorf("there is no identity property %q", name)
		}
		if typ := rw.identityProperty.typ; c.Type.Kind() != typ.Kind() {
			return fmt.Errorf("identity property %q has type %s instead of %s", c.Name, c.Type.Kind(), typ.Kind())
		}
		rw.identityProperty.typ = c.Type
		rw.identityProperty.index = columnIndex[c.Name]
	}
	// Validate the last change time property.
	if name := rw.lastChangeTimeProperty.name; name != "" {
		c, ok := columnByName[name]
		if !ok {
			return fmt.Errorf("there is no last change time property %q", name)
		}
		if typ := rw.lastChangeTimeProperty.typ; c.Type.Kind() != typ.Kind() {
			return fmt.Errorf("last change time property %q has type %s instead of %s", c.Name, c.Type.Kind(), typ.Kind())
		}
		rw.lastChangeTimeProperty.typ = c.Type
		rw.lastChangeTimeProperty.index = columnIndex[c.Name]
	}
	// Validate the displayed property.
	if rw.displayedProperty.name != "" {
		col, err := displayedPropertyFromSchema(fileSchema, rw.displayedProperty.name)
		if err != nil {
			slog.Warn("cannot determine the displayed property", "err", err)
			rw.displayedProperty.name = ""
		} else {
			rw.displayedProperty.column = col
			rw.displayedProperty.index = columnIndex[col.Name]
		}
	}
	// Check that the schema, if valid, is aligned with the file's schema.
	if rw.schema.Valid() {
		err := checkSchemaAlignment(rw.schema, fileSchema)
		if err != nil {
			return err
		}
		rw.properties = types.Properties(rw.schema)
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
		return fmt.Errorf("connector %s did not call the Columns method before calling Record", rw.connector)
	}
	if len(record) != rw.columns {
		return fmt.Errorf("connector %s has returned records with different lengths", rw.connector)
	}
	var err error
	if rw.yield == nil {
		// Store the record in the records field.
		rd := make(map[string]any, len(rw.properties))
		for i, c := range rw.properties {
			rd[c.Name], err = normalize(c.Name, c.Type, record[i], c.Nullable, rw.timeLayouts)
			if err != nil {
				return err
			}
		}
		rw.records = append(rw.records, rd)
	} else {
		if rw.record.Properties != nil || rw.record.Err != nil {
			if !rw.yield(rw.record) {
				rw.record.Properties = nil
				rw.record.Err = nil
				return errRecordStop
			}
		}
		// Call the rw.write function to store the record.
		rw.record = Record{Properties: map[string]any{}}
		for i, c := range rw.properties {
			j := rw.columnIndexOf[i]
			value, err := normalize(c.Name, c.Type, record[j], c.Nullable, rw.timeLayouts)
			if err != nil {
				rw.record.Err = err
				break
			}
			rw.record.Properties[c.Name] = value
		}
		// Parse the identity property.
		rw.record.ID, err = parseIdentityProperty(rw.identityProperty.name, rw.identityProperty.typ, record[rw.identityProperty.index], rw.timeLayouts)
		if err != nil {
			if rw.record.Err != nil {
				rw.record.Err = err
			}
		}
		// Parse the last change time property.
		if rw.record.Err == nil {
			if rw.lastChangeTimeProperty.name != "" {
				ts := record[rw.lastChangeTimeProperty.index]
				rw.record.LastChangeTime, err = parseTimestampColumn(rw.lastChangeTimeProperty.name, rw.lastChangeTimeProperty.typ, rw.lastChangeTimeProperty.format, ts, rw.timeLayouts)
				if err != nil {
					rw.record.Err = err
				}
			} else {
				rw.record.LastChangeTime = rw.storageLastChangeTime
			}
		}
		// Parse the displayed property.
		if rw.record.Err == nil && rw.displayedProperty.name != "" {
			v := record[rw.displayedProperty.index]
			c := rw.displayedProperty.column
			vv, err := normalize(c.Name, c.Type, v, c.Nullable, rw.timeLayouts)
			if err != nil {
				slog.Warn("the displayed property value cannot be normalized", "err", err)
			} else {
				s, err := displayedPropertyToString(vv)
				if err != nil {
					slog.Warn("invalid displayed property value", "err", err)
				} else {
					rw.record.DisplayedProperty = s
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

// RecordMap writes a record as a map.
func (rw *recordWriter) RecordMap(record map[string]any) error {
	if rw.properties == nil {
		return fmt.Errorf("connector %s did not call the Columns method before calling RecordMap", rw.connector)
	}
	var err error
	if rw.yield == nil {
		// Store the record in the records field.
		rd := make(map[string]any, len(rw.properties))
		for _, c := range rw.properties {
			rd[c.Name], err = normalize(c.Name, c.Type, record[c.Name], c.Nullable, rw.timeLayouts)
			if err != nil {
				return err
			}
		}
		rw.records = append(rw.records, rd)
	} else {
		if rw.record.Properties != nil || rw.record.Err != nil {
			if !rw.yield(rw.record) {
				rw.record.Properties = nil
				rw.record.Err = nil
				return errRecordStop
			}
		}
		// Call the rw.write function to store the record.
		rw.record = Record{Properties: record}
		for _, c := range rw.properties {
			value, err := normalize(c.Name, c.Type, record[c.Name], c.Nullable, rw.timeLayouts)
			if err != nil {
				rw.record.Err = err
				break
			}
			rw.record.Properties[c.Name] = value
		}
		// Parse the identity property.
		rw.record.ID, err = parseIdentityProperty(rw.identityProperty.name, rw.identityProperty.typ, record[rw.identityProperty.name], rw.timeLayouts)
		if err != nil {
			if rw.record.Err != nil {
				rw.record.Err = err
			}
		}
		// Parse the last change time property.
		if rw.record.Err == nil {
			if rw.lastChangeTimeProperty.name != "" {
				ts := record[rw.lastChangeTimeProperty.name]
				rw.record.LastChangeTime, err = parseTimestampColumn(rw.lastChangeTimeProperty.name, rw.lastChangeTimeProperty.typ, rw.lastChangeTimeProperty.format, ts, rw.timeLayouts)
				if err != nil {
					rw.record.Err = err
				}
			} else {
				rw.record.LastChangeTime = rw.storageLastChangeTime
			}
		}
		// Parse the displayed property.
		if rw.record.Err == nil && rw.displayedProperty.name != "" {
			v := record[rw.displayedProperty.name]
			c := rw.displayedProperty.column
			vv, err := normalize(c.Name, c.Type, v, c.Nullable, rw.timeLayouts)
			if err != nil {
				slog.Warn("displayed property value cannot be normalized", "err", err)
			} else {
				s, err := displayedPropertyToString(vv)
				if err != nil {
					slog.Warn("invalid displayed property value", "err", err)
				} else {
					rw.record.DisplayedProperty = s
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
		return fmt.Errorf("connector %s did not call the Columns method before calling RecordString", rw.connector)
	}
	if len(record) != rw.columns {
		return fmt.Errorf("connector %s has returned records with different lengths", rw.connector)
	}
	if !rw.textColumnsOnly {
		return fmt.Errorf("connector %s has called RecordString when there are non-text columns", rw.connector)
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
		if rw.record.Properties != nil || rw.record.Err != nil {
			if !rw.yield(rw.record) {
				rw.record.Properties = nil
				rw.record.Err = nil
				return errRecordStop
			}
		}
		// Call the rw.write function to store the record.
		rw.record = Record{Properties: make(map[string]any, len(rw.properties))}
		for i, c := range rw.properties {
			j := rw.columnIndexOf[i]
			value := record[j]
			err = validateStringProperty(c, value)
			if err != nil {
				rw.record.Err = err
				break
			}
			rw.record.Properties[c.Name] = value
		}
		// Parse the identity property.
		rw.record.ID, err = parseIdentityProperty(rw.identityProperty.name, rw.identityProperty.typ, record[rw.identityProperty.index], rw.timeLayouts)
		if err != nil {
			if rw.record.Err != nil {
				rw.record.Err = err
			}
		}
		// Parse the last change time property.
		if rw.record.Err == nil {
			if rw.lastChangeTimeProperty.name != "" {
				ts := record[rw.lastChangeTimeProperty.index]
				rw.record.LastChangeTime, err = parseTimestampColumn(rw.lastChangeTimeProperty.name, rw.lastChangeTimeProperty.typ, rw.lastChangeTimeProperty.format, ts, rw.timeLayouts)
				if err != nil {
					rw.record.Err = err
				}
			} else {
				rw.record.LastChangeTime = rw.storageLastChangeTime
			}
		}
		// Parse the displayed property.
		if rw.record.Err == nil && rw.displayedProperty.name != "" {
			v := record[rw.displayedProperty.index]
			c := rw.displayedProperty.column
			vv, err := normalize(c.Name, c.Type, v, c.Nullable, rw.timeLayouts)
			if err != nil {
				slog.Warn("displayed property value cannot be normalized", "err", err)
			} else {
				s, err := displayedPropertyToString(vv)
				if err != nil {
					slog.Warn("invalid displayed property value", "err", err)
				} else {
					rw.record.DisplayedProperty = s
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
// from which to read the file and its last change time.
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
