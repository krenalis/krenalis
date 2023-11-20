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

	"chichi/apis/postgres"
	"chichi/apis/state"
	_connector "chichi/connector"
	"chichi/connector/types"

	"github.com/golang/snappy"
)

// File represents the file of a file connection.
type File struct {
	db         *postgres.DB
	connection *state.Connection
	inner      _connector.FileConnection
	err        error
}

// File returns a file for the provided connection. Errors are deferred until a
// file's method is called. It panics if connection is not a file connections.
func (connectors *Connectors) File(connection *state.Connection) *File {
	file := &File{
		db:         connectors.db,
		connection: connection,
	}
	file.inner, file.err = _connector.RegisteredFile(connection.Connector().Name).New(&_connector.FileConfig{
		Role:        _connector.Role(connection.Role),
		Settings:    connection.Settings,
		SetSettings: setSettingsFunc(connectors.db, connection),
	})
	return file
}

// CompletePath returns the complete representation of the provided path name or
// an InvalidPathError value if name is not valid for use in calls to Read and
// Write. name's length in runes must be in range [1, 1024].
//
// It returns the ErrNoStorage error if the file does not have a storage.
func (file *File) CompletePath(ctx context.Context, name string) (string, error) {
	if file.err != nil {
		return "", file.err
	}
	storage, err := file.storage()
	if err != nil {
		return "", err
	}
	return storage.CompletePath(ctx, name)
}

// ContentType returns the content type of the file.
func (file *File) ContentType(ctx context.Context) (string, error) {
	if file.err != nil {
		return "", file.err
	}
	return file.inner.ContentType(ctx), nil
}

// Read reads the records from the file at the provided path name and returns
// the columns and the records. name must be UTF-8 encoded with a length in
// range [1, 1024].
//
// If the file connection supports multiple sheets, sheet is the
// sheet name and must be UTF-8 encoded with a length in range [1, 100],
// otherwise must be an empty string. limit restricts the number of records to
// return and should not exceed 100. If limit is negative, there is no upper
// limit on the number of records returned.
//
// It returns the ErrNoStorage error if the file does not have a storage, and
// it returns the ErrNoColumns error if the file has no columns.
func (file *File) Read(ctx context.Context, name, sheet string, limit int) (columns []types.Property, rows []map[string]any, err error) {
	if file.err != nil {
		return nil, nil, file.err
	}
	if limit < 0 {
		limit = math.MaxInt
	}
	storage, err := file.storage()
	if err != nil {
		return nil, nil, err
	}
	s := newCompressedStorage(storage, file.connection.Compression)
	r, _, err := s.Reader(ctx, name)
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()
	rw := newRecordWriter(file.connection.Connector().ID, types.Type{}, limit)
	err = file.inner.Read(ctx, r, sheet, rw)
	if err != nil && err != errRecordStop {
		return nil, nil, err
	}
	if err = r.Close(); err != nil {
		return nil, nil, err
	}
	if rw.properties == nil {
		return nil, nil, ErrNoColumns
	}
	return rw.properties, rw.records, nil
}

// ReadFunc reads the records from the file at the provided path name and calls
// the write function for each record read. The returned records conform to the
// provided schema, which must be valid and compatible with the file's schema.
//
// name must be UTF-8 encoded with a length in range [1, 1024]. If the file
// connection supports multiple sheets, sheet is the sheet name and must be
// UTF-8 encoded with a length in range [1, 100], otherwise must be an empty
// string. identityColumn is the column used as the identity, and if
// timestampColumn is not nil, it represents the column used as the timestamp
// along with its format.
//
// It returns the ErrNoStorage error if the file does not have a storage, and
// it returns the ErrNoColumns error if the file has no columns.
func (file *File) ReadFunc(ctx context.Context, name, sheet string, schema types.Type, identityColumn string, timestampColumn TimestampColumn, write func(Record) error) error {
	if file.err != nil {
		return file.err
	}
	storage, err := file.storage()
	if err != nil {
		return err
	}
	s := newCompressedStorage(storage, file.connection.Compression)
	r, _, err := s.Reader(ctx, name)
	if err != nil {
		return err
	}
	defer r.Close()
	rw := newRecordWriter(file.connection.Connector().ID, schema, math.MaxInt)
	rw.SetWriteFunc(write, identityColumn, timestampColumn)
	err = file.inner.Read(ctx, r, sheet, rw)
	if err != nil && err != errRecordStop {
		return err
	}
	if err = r.Close(); err != nil {
		return err
	}
	if rw.properties == nil {
		return ErrNoColumns
	}
	return nil
}

// Sheets returns the sheets of the file with the provided name. It returns the
// ErrNoStorage error if the file does not have a storage.
// It panics if the file connector does not support sheets.
func (file *File) Sheets(ctx context.Context, name string) ([]string, error) {
	if file.err != nil {
		return nil, file.err
	}
	inner := file.inner.(_connector.Sheets)
	storage, err := file.storage()
	if err != nil {
		return nil, err
	}
	s := newCompressedStorage(storage, file.connection.Compression)
	r, _, err := s.Reader(ctx, name)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	sheets, err := inner.Sheets(ctx, r)
	if err != nil {
		return nil, err
	}
	if err = r.Close(); err != nil {
		return nil, err
	}
	return sheets, nil
}

// Write writes the provided records into the file located at the specified
// path. columns represents the columns of the records to be written.
// It returns the ErrNoStorage error if the file does not have a storage.
func (file *File) Write(ctx context.Context, name, sheet string, columns []types.Property, records [][]any) error {
	if file.err != nil {
		return file.err
	}
	storage, err := file.storage()
	if err != nil {
		return err
	}
	s := newCompressedStorage(storage, file.connection.Compression)
	extension := file.connection.Connector().FileExtension
	w, err := s.Writer(ctx, name, file.inner.ContentType(ctx), extension)
	if err != nil {
		return err
	}
	r := newRecordReader(columns, records)
	err = file.inner.Write(ctx, w, sheet, r)
	if err2 := w.CloseWithError(err); err2 != nil && err == nil {
		err = err2
	}
	return err
}

// storage returns the inner storage connection of the file. If the file does
// not have a storage, it returns the ErrNoStorage error.
func (file *File) storage() (_connector.StorageConnection, error) {
	storage, ok := file.connection.Storage()
	if !ok {
		return nil, ErrNoStorage
	}
	return _connector.RegisteredStorage(storage.Name).New(&_connector.StorageConfig{
		Role:        _connector.Role(storage.Role),
		Settings:    storage.Settings,
		SetSettings: setSettingsFunc(file.db, storage),
	})
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

// newRecordWriter returns a new record writer that writes at most limit
// records. If the write function is set with SetWriteFunc, the recordWriter
// calls the write function for each record written, otherwise it stores the
// records in the records field.
func newRecordWriter(connector int, schema types.Type, limit int) *recordWriter {
	rw := recordWriter{
		connector:       connector,
		schema:          schema,
		limit:           limit,
		textColumnsOnly: true,
		records:         []map[string]any{},
	}
	rw.identityColumn.index = -1
	rw.timestampColumn.index = -1
	return &rw
}

// recordWriter implements the connector.RecordWriter interface.
type recordWriter struct {
	connector       int
	limit           int
	write           WriteFunc
	schema          types.Type
	properties      []types.Property // schema's properties, or the file's columns if a schema has not been provided
	columnIndexOf   map[int]int      // map a property index in the schema to the corresponding file's column
	columns         int              // number of file's columns
	textColumnsOnly bool
	identityColumn  struct {
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
	records []map[string]any
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
			rw.textColumnsOnly = c.Type.PhysicalType() == types.PtText
		}
	}
	// Validate the identity column.
	if name := rw.identityColumn.name; name != "" {
		c, ok := columnByName[name]
		if !ok {
			return fmt.Errorf("there is no identity column %q", name)
		}
		switch pt := c.Type.PhysicalType(); pt {
		case types.PtInt, types.PtUint, types.PtUUID, types.PtJSON, types.PtText:
		default:
			return fmt.Errorf("identity column %q has type %s instead of Int, Uint, UUID, JSON, or Text", c.Name, pt)
		}
		rw.identityColumn.typ = c.Type
		rw.identityColumn.index = columnIndex[c.Name]
	}
	// Validate the timestamp column.
	if name := rw.timestampColumn.name; name != "" {
		c, ok := columnByName[name]
		if !ok {
			return fmt.Errorf("there is no timestamp column %q", name)
		}
		switch pt := c.Type.PhysicalType(); pt {
		case types.PtDateTime, types.PtDate, types.PtJSON, types.PtText:
		default:
			return fmt.Errorf("timestamp column %q has type %s instead of DateTime, Date, JSON, or Text", c.Name, pt)
		}
		rw.timestampColumn.typ = c.Type
		rw.timestampColumn.index = columnIndex[c.Name]
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
	if rw.write == nil {
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
		// Parse the identity column.
		rd.ID, err = parseIdentityColumn(rw.identityColumn.name, rw.identityColumn.typ, record[rw.identityColumn.index])
		if err != nil {
			if rd.Err != nil {
				rd.Err = err
			}
		}
		// Parse the timestamp column.
		if rd.Err == nil && rw.timestampColumn.name != "" {
			ts := record[rw.timestampColumn.index]
			rd.Timestamp, err = parseTimestampColumn(rw.timestampColumn.name, rw.timestampColumn.typ, rw.timestampColumn.format, ts)
			if err != nil {
				rd.Err = err
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
	if rw.properties == nil {
		return fmt.Errorf("connector %d did not call the Columns method before calling RecordMap", rw.connector)
	}
	var err error
	if rw.write == nil {
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
		// Parse the identity column.
		rd.ID, err = parseIdentityColumn(rw.identityColumn.name, rw.identityColumn.typ, record[rw.identityColumn.name])
		if err != nil {
			if rd.Err != nil {
				rd.Err = err
			}
		}
		// Parse the timestamp column.
		if rd.Err == nil && rw.timestampColumn.name != "" {
			ts := record[rw.timestampColumn.name]
			rd.Timestamp, err = parseTimestampColumn(rw.timestampColumn.name, rw.timestampColumn.typ, rw.timestampColumn.format, ts)
			if err != nil {
				rd.Err = err
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
	if rw.write == nil {
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
		// Parse the identity column.
		rd.ID, err = parseIdentityColumn(rw.identityColumn.name, rw.identityColumn.typ, record[rw.identityColumn.index])
		if err != nil {
			if rd.Err != nil {
				rd.Err = err
			}
		}
		// Parse the timestamp column.
		if rd.Err == nil && rw.timestampColumn.name != "" {
			ts := record[rw.timestampColumn.index]
			rd.Timestamp, err = parseTimestampColumn(rw.timestampColumn.name, rw.timestampColumn.typ, rw.timestampColumn.format, ts)
			if err != nil {
				rd.Err = err
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

// SetWriteFunc sets the write function for the recordWriter.
func (rw *recordWriter) SetWriteFunc(write WriteFunc, identity string, timestamp TimestampColumn) {
	rw.write = write
	rw.identityColumn.name = identity
	rw.timestampColumn.name = timestamp.Name
	rw.timestampColumn.format = timestamp.Format
}

// parseIdentityColumn parses an identity column value.
func parseIdentityColumn(name string, typ types.Type, value any) (string, error) {
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

// parseTimestampColumn parses a timestamp column value.
func parseTimestampColumn(name string, typ types.Type, format string, value any) (time.Time, error) {
	timestamp, err := normalizeDatabaseFileProperty(name, typ, value, false)
	if err != nil {
		return time.Time{}, err
	}
	switch timestamp := timestamp.(type) {
	case nil:
		return time.Time{}, errors.New("timestamp value is null")
	case time.Time:
		return timestamp, nil
	case string:
		ts, err := time.Parse(format, value.(string))
		if err != nil {
			return time.Time{}, fmt.Errorf("timestamp %q does not conform to the %q format", value, format)
		}
		return ts.UTC(), nil
	case json.RawMessage:
		var s string
		err := json.Unmarshal(timestamp, &s)
		if err != nil {
			return time.Time{}, fmt.Errorf("timestamp value is not a JSON string")
		}
		ts, err := time.Parse(format, value.(string))
		if err != nil {
			return time.Time{}, fmt.Errorf("timestamp %q does not conform to the %q format", value, format)
		}
		return ts.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("timestamp value is not a JSON string")
}

// compressorStorage implements a storage capable of compressing and
// decompressing data read from or written to a connector.StorageConnection.
type compressorStorage struct {
	storage     _connector.StorageConnection
	compression state.Compression
}

// newCompressedStorage returns a compressor storage that wraps s and performs
// file compression and decompression using c as the compression method.
// If c is NoCompression, it does not perform any compression or decompression.
func newCompressedStorage(s _connector.StorageConnection, c state.Compression) *compressorStorage {
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
		r = newFuncReadCloser(r2, func() error {
			err2 := r2.Close()
			err := r.Close()
			if err2 != nil {
				return err2
			}
			return err
		})
		r = r2
	case state.SnappyCompression:
		r2 := snappy.NewReader(r)
		r = newFuncReadCloser(r2, func() error {
			return r.Close()
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
		err := cs.storage.Write(ctx, pr, path, contentType)
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
