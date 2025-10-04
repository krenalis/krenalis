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
	"iter"
	"log/slog"
	"math"
	"os"
	pathPkg "path"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/internal/schemas"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/types"

	"github.com/klauspost/compress/snappy"
)

type fileContentTypeConnector interface {
	// ContentType returns the content type of the file.
	ContentType(ctx context.Context) string
}

type fileReadConnector interface {
	// Read reads the records from r and writes them to records. If the connector
	// has multiple sheets, sheet is the name of the sheet to be read.
	// If the provided sheet does not exist, it returns the ErrSheetNotExist error.
	Read(ctx context.Context, r io.Reader, sheet string, records meergo.RecordWriter) error
}

type fileWriteConnector interface {
	// Write writes to w the records read from records. If the connector has
	// multiple sheets, sheet is the name of the sheet to be written to.
	Write(ctx context.Context, w io.Writer, sheet string, records meergo.RecordReader) error
}

type fileSheetConnector interface {
	// Sheets returns the sheets of the file read from r.
	Sheets(ctx context.Context, r io.Reader) ([]string, error)
}

// storageTimeout represents the duration between consecutive calls to the Read
// method of the io.Reader passed to a storage within the Write method.
var storageTimeout = 10 * time.Second

type File struct {
	connector   string
	state       *state.State
	action      *state.Action
	timeLayouts *state.TimeLayouts
	inner       any
	err         error
}

// File returns a file for the provided action.
// Errors are deferred until a file's method is called.
func (connectors *Connectors) File(action *state.Action) *File {
	format := action.Format()
	connection := action.Connection()
	file := &File{
		connector:   connection.Connector().Code,
		state:       connectors.state,
		action:      action,
		timeLayouts: &format.TimeLayouts,
	}
	file.inner, file.err = meergo.RegisteredFile(format.Code).New(&meergo.FileEnv{
		Settings:    action.FormatSettings,
		SetSettings: setActionSettingsFunc(connectors.state, action),
	})
	file.err = connectorError(file.err)
	return file
}

// Connector returns the name of the file connector.
func (file *File) Connector() string {
	return file.connector
}

// Records returns an iterator over the file's records that are not older than
// the provided starting time. If the starting time is the zero value, it
// iterates over all the records.
//
// file must support reading of records, otherwise this method panics.
//
// Each returned record contains, in its Properties field, the properties of the
// action's input schema with the same types.
//
// If the action's input schema does not align with the file's schema, the
// iterator, not Records, will return a *schemas.Error error.
//
// If the identity column specified in the action exists in the file schema but
// its type differs, the iterator returns an error. The same applies to the last
// change time column, if specified.
//
// If the action's sheet is not found in the file, the All method of the
// iterator returns immediately and a subsequent call to Err returns
// meergo.ErrSheetNotExist. The same happens if the file has no columns; in that
// case Err returns ErrNoColumnsFound.
//
// It returns an error if a non-zero starting time is provided and the action
// has no last change property.
func (file *File) Records(ctx context.Context, startTime time.Time) (Records, error) {
	if file.err != nil {
		return nil, file.err
	}
	if !startTime.IsZero() && file.action.LastChangeTimeColumn == "" {
		return nil, fmt.Errorf("a start time has been provided, but the action does not have the last change property")
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
	if err = validateLastChangeTime(storageLastChangeTime); err != nil {
		return nil, fmt.Errorf("invalid last change time returned by the storage: %s", err)
	}
	rw := newRecordWriter(file.action.Format().Code, file.action,
		storageLastChangeTime, file.timeLayouts, startTime, math.MaxInt)
	records := &fileRecords{
		rw:    rw,
		rc:    rc,
		sheet: file.action.Sheet,
		inner: file.inner.(fileReadConnector),
	}
	return records, nil
}

// Writer returns a Writer for writing records into the file located at the path
// of the file's action. schema contains the properties of the records to be
// written.
//
// This method panics if either the FileStorage or the file does not support
// writing operations.
//
// If pathReplacer is not nil, placeholders in path are replaced using it; in
// this case a *PlaceholderError may be returned when placeholders are invalid.
//
// It returns an *UnavailableError if the connector returns an error.
func (file *File) Writer(ctx context.Context, pathReplacer PlaceholderReplacer) (Writer, error) {
	if file.err != nil {
		return nil, file.err
	}
	storage, err := file.storage()
	if err != nil {
		return nil, err
	}
	s := newCompressedStorage(storage, file.action.Compression)
	extension := file.action.Format().FileExtension
	path := file.action.Path
	if pathReplacer != nil {
		var err error
		path, err = ReplacePlaceholders(path, pathReplacer)
		if err != nil {
			return nil, err
		}
	}
	sw, err := s.Writer(ctx, path, file.inner.(fileContentTypeConnector).ContentType(ctx), extension)
	if err != nil {
		return nil, connectorError(err)
	}
	records := make(chan fileRecord, 100)
	result := make(chan error, 1)
	writeCtx, cancelWrite := context.WithCancel(context.Background())
	// Call the connector's Write method in its own goroutine.
	go func() {
		columns := file.action.InSchema.Properties().Slice()
		r := newRecordReader(columns, records)
		err = file.inner.(fileWriteConnector).Write(writeCtx, sw, file.action.Sheet, r)
		if err2 := sw.CloseWithError(err); err2 != nil && err == nil {
			err = err2
		}
		result <- err // err will be nil if no error occurred.
	}()
	fw := &fileWriter{
		cancelWrite: cancelWrite,
		records:     records,
		result:      result,
	}
	return fw, nil
}

// storage returns the inner storage connection of the file.
func (file *File) storage() (any, error) {
	conn := file.action.Connection()
	connector := file.action.Connection().Connector()
	return meergo.RegisteredFileStorage(connector.Code).New(&meergo.FileStorageEnv{
		Settings:    conn.Settings,
		SetSettings: setConnectionSettingsFunc(file.state, conn),
	})
}

// IsValidSheetName reports whether name is a valid sheet name.
func IsValidSheetName(name string) bool {
	if name == "" || !utf8.ValidString(name) || strings.ContainsRune(name, '\x00') {
		return false
	}
	if utf8.RuneCountInString(name) > 31 {
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
	storage     any // can implement read, write or both operations.
	compression state.Compression
}

// newCompressedStorage returns a compressor storage that wraps s and performs
// file compression and decompression using c as the compression method.
// If c is NoCompression, it does not perform any compression or decompression.
func newCompressedStorage(s any, c state.Compression) *compressorStorage {
	return &compressorStorage{s, c}
}

// Reader opens the file at the provided path name and returns an io.ReadCloser
// from which to read the file and its last change time.
//
// The storage must support read operations, otherwise this method panics.
//
// It returns an *UnavailableError if the connector returns an error.
// It is the caller's responsibility to close the returned reader.
func (cs compressorStorage) Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error) {
	r, t, err := cs.storage.(fileStorageReaderConnector).Reader(ctx, name)
	if err != nil {
		return nil, time.Time{}, connectorError(err)
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
// The storage must support write operations, otherwise this method panics.
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
		name = strings.TrimSuffix(name, ".zip")
		if ext := pathPkg.Ext(name); ext == "" {
			name += "." + extension
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
	if ct := cs.compression.ContentType(); ct != "" {
		contentType = ct
	}
	ch := make(chan error)
	go func() {
		ctx, cancel := context.WithCancel(ctx)
		r := newTimeoutReader(pr, storageTimeout, cancel)
		defer r.Close()
		err := cs.storage.(fileStorageWriteConnector).Write(ctx, r, path, contentType)
		if err != nil {
			_ = pr.CloseWithError(connectorError(err))
		} else {
			// errReadStopped will be returned to the file connector only if it
			// calls w.Write when the storage is returned.
			_ = pr.CloseWithError(errReadStopped)
		}
		ch <- connectorError(err)
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

// fileWriter implements the Writer interface for files.
type fileWriter struct {
	cancelWrite context.CancelFunc
	records     chan<- fileRecord
	result      <-chan error
	closed      bool
	err         error
}

func (w *fileWriter) Close(ctx context.Context) error {
	if w.closed {
		return w.err
	}
	w.closed = true
	close(w.records)
	// If Write has already recorded an error, return it immediately.
	if w.err != nil {
		return w.err
	}
	var err error
	select {
	case err = <-w.result:
		// The connector has terminated with or without an error.
	case <-ctx.Done():
		// The context has been canceled.
		w.cancelWrite()
		err = <-w.result
	}
	return err
}

func (w *fileWriter) Write(ctx context.Context, id string, properties map[string]any) bool {
	if w.closed {
		panic("connectors: Write called on a closed writer")
	}
	r := fileRecord{
		id:     id,
		record: properties,
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

// fileRecords implements the Records interface for files.
type fileRecords struct {
	rw     *recordWriter
	rc     io.ReadCloser
	sheet  string
	inner  fileReadConnector
	err    error
	closed bool
}

func (r *fileRecords) All(ctx context.Context) iter.Seq[Record] {
	return func(yield func(Record) bool) {
		if r.closed {
			r.err = errors.New("connectors: For called on a closed Records")
			return
		}
		defer func() {
			_ = r.Close()
			if r.err == nil && r.rw.properties == nil {
				r.err = ErrNoColumnsFound
			}
		}()
		r.rw.setYieldFunc(yield)
		err := r.inner.Read(ctx, r.rc, r.sheet, r.rw)
		r.rw.close()
		if err != nil && err != errRecordStop {
			r.err = connectorError(err)
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
	return r.rw.last()
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

var (
	// errRecordStop is returned by recordWriter methods when the maximum row
	// limit has been reached, signaling the need to stop writing rows.
	errRecordStop = errors.New("stop record")

	// errReadStopped is returned to a file connector when it calls w.Write and the
	// storage has already finished reading without an error.
	// If this error occurs, it indicates a bug in the storage connector.
	errReadStopped = errors.New("storage abruptly stopped reading")
)

// newFuncWriteCloser returns an io.WriteCloser that wraps w and calls close
// when Close is invoked.
func newFuncWriteCloser(w io.Writer, close func(err error) error) *storageWriteCloser {
	return &storageWriteCloser{w, close}
}

// newRecordReader returns a new record reader that read records.
func newRecordReader(columns []types.Property, records <-chan fileRecord) *recordReader {
	return &recordReader{
		columns: columns,
		records: records,
	}
}

// recordReader implements the connector.RecordReader interface.
type recordReader struct {
	columns []types.Property
	records <-chan fileRecord
}

type fileRecord struct {
	id     string
	record map[string]any
}

// Columns returns the columns of the records as properties.
func (rr *recordReader) Columns() []types.Property {
	return rr.columns
}

// Record returns the next record. The keys of the record are column names.
// A record may be empty or contain only a subset of columns.
// It returns nil and io.EOF if there are no more records.
func (rr *recordReader) Record(ctx context.Context) (map[string]any, error) {
	select {
	case r, ok := <-rr.records:
		if !ok {
			return nil, io.EOF
		}
		return r.record, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// newRecordWriter returns a new record writer implementing the RecordWriter
// interface. It calls the yield function, which must be set by calling the
// setYieldFunc method, for each record written, up to a maximum of limit
// records.
//
// storageLastChangeTime is the last change time provided by the storage
// connector, used when the file columns do not specify a last change time
// property.
//
// The close method should be called when there are no more records to write.
func newRecordWriter(format string, action *state.Action, storageLastChangeTime time.Time, layout *state.TimeLayouts, startTime time.Time, limit int) *recordWriter {
	rw := recordWriter{
		format:                format,
		action:                action,
		storageLastChangeTime: storageLastChangeTime,
		timeLayouts:           layout,
		startTime:             startTime,
		limit:                 limit,
		identityColumnIndex:   -1,
		lastChangeTimeIndex:   -1,
	}
	return &rw
}

// recordWriter implements the connector.RecordWriter interface.
type recordWriter struct {
	format                 string
	action                 *state.Action
	storageLastChangeTime  time.Time
	timeLayouts            *state.TimeLayouts
	startTime              time.Time
	limit                  int
	identityColumnIndex    int
	lastChangeTimeIndex    int
	numPropertiesPerRecord int
	properties             []types.Property // properties of the action's schema, or the file's columns if an action has not been provided
	issues                 []string
	record                 Record
	yield                  func(Record) bool
	isLast                 bool
}

// Columns sets the columns of the records as properties.
// Columns must be called before Record, RecordSlice and RecordStrings.
func (rw *recordWriter) Columns(columns []types.Property) error {
	if rw.properties != nil {
		return fmt.Errorf("connector %s has called Columns twice", rw.format)
	}
	if len(columns) == 0 {
		return fmt.Errorf("connector %s has called Columns with an empty columns", rw.format)
	}
	fileSchema, err := types.ObjectOf(columns)
	if err != nil {
		return connectorError(rewriteColumnErrors(err))
	}
	if rw.action == nil {
		rw.properties = columns
	} else {
		// Check that the action's input schema is aligned with the file's schema.
		err := schemas.CheckAlignment(rw.action.InSchema, fileSchema, nil)
		if err != nil {
			return err
		}
		inSchemaProperties := rw.action.InSchema.Properties()
		rw.properties = make([]types.Property, len(columns))
		for i, c := range columns {
			p, ok := inSchemaProperties.ByName(c.Name)
			if !ok {
				continue
			}
			rw.properties[i] = p
			if p.Name == rw.action.IdentityColumn {
				rw.identityColumnIndex = i
			}
			if p.Name == rw.action.LastChangeTimeColumn {
				rw.lastChangeTimeIndex = i
			}
			rw.numPropertiesPerRecord++
		}
		if rw.action.IdentityColumn != "" && rw.identityColumnIndex == -1 {
			return fmt.Errorf("there is no identity column %q", rw.action.IdentityColumn)
		}
		if rw.action.LastChangeTimeColumn != "" && rw.lastChangeTimeIndex == -1 {
			return fmt.Errorf("there is no last change time column %q", rw.action.LastChangeTimeColumn)
		}
	}
	if rw.limit == 0 {
		return errRecordStop
	}
	return nil
}

// Issue reports an issue encountered during file reading that did not prevent
// the file from being processed.
func (rw *recordWriter) Issue(format string, a ...any) {
	rw.issues = append(rw.issues, fmt.Sprintf(format, a...))
}

// Record writes a record as a map.
func (rw *recordWriter) Record(record map[string]any) error {
	if rw.properties == nil {
		return fmt.Errorf("connector %s did not call the Columns method before calling RecordMap", rw.format)
	}
	// Get the last change time.
	var err error
	lastChangeTime := rw.storageLastChangeTime
	if i := rw.lastChangeTimeIndex; i >= 0 {
		p := rw.properties[i]
		var t time.Time
		t, err = parseLastChangeTimeColumn(p.Name, p.Type, rw.action.LastChangeTimeFormat, record[p.Name], p.Nullable, rw.timeLayouts)
		if err == nil {
			if !t.IsZero() {
				lastChangeTime = t
			}
			if !rw.startTime.IsZero() && lastChangeTime.Before(rw.startTime) {
				// Skip the record because it is older than the specified starting time.
				return nil
			}
		}
	}
	// Call the yield function passing the previous record.
	if rw.record.Properties != nil || rw.record.Err != nil {
		if !rw.yield(rw.record) {
			return errRecordStop
		}
	}
	rw.record = Record{
		Properties:     make(map[string]any, rw.numPropertiesPerRecord),
		LastChangeTime: lastChangeTime,
		Err:            err,
	}
	// Get the identity column.
	if i := rw.identityColumnIndex; i >= 0 {
		p := rw.properties[i]
		rw.record.ID, err = parseIdentityColumn(p.Name, p.Type, record[p.Name], rw.timeLayouts)
		if err != nil {
			rw.record.Err = err
		}
	}
	// Get the properties.
	if rw.record.Err == nil {
		for _, p := range rw.properties {
			if p.Name == "" {
				continue
			}
			v, ok := record[p.Name]
			if !ok {
				if !p.ReadOptional {
					rw.record.Err = inputValidationErrorf(p.Name, "does not have a value, but the property is not optional for reading")
					break
				}
				continue
			}
			v, err = normalize(p.Name, p.Type, v, p.Nullable, rw.timeLayouts)
			if err != nil {
				rw.record.Err = err
				break
			}
			rw.record.Properties[p.Name] = v
		}
	}
	rw.limit--
	if rw.limit == 0 {
		return errRecordStop
	}
	return nil
}

// RecordSlice writes a record.
// The record's length must equal to the number of columns.
func (rw *recordWriter) RecordSlice(record []any) error {
	if rw.properties == nil {
		return fmt.Errorf("connector %s did not call the Columns method before calling Record", rw.format)
	}
	if len(record) != len(rw.properties) {
		return fmt.Errorf("connector %s has returned records with different lengths", rw.format)
	}
	// Get the last change time.
	var err error
	lastChangeTime := rw.storageLastChangeTime
	if i := rw.lastChangeTimeIndex; i >= 0 {
		p := rw.properties[i]
		var t time.Time
		t, err = parseLastChangeTimeColumn(p.Name, p.Type, rw.action.LastChangeTimeFormat, record[i], p.Nullable, rw.timeLayouts)
		if err == nil {
			if !t.IsZero() {
				lastChangeTime = t
			}
			if !rw.startTime.IsZero() && lastChangeTime.Before(rw.startTime) {
				// Skip the record because it is older than the specified starting time.
				return nil
			}
		}
	}
	// Call the yield function passing the previous record.
	if rw.record.Properties != nil || rw.record.Err != nil {
		if !rw.yield(rw.record) {
			return errRecordStop
		}
	}
	rw.record = Record{
		Properties:     make(map[string]any, rw.numPropertiesPerRecord),
		LastChangeTime: lastChangeTime,
		Err:            err,
	}
	// Get the identity column.
	if i := rw.identityColumnIndex; i >= 0 {
		p := rw.properties[i]
		rw.record.ID, err = parseIdentityColumn(p.Name, p.Type, record[i], rw.timeLayouts)
		if err != nil {
			rw.record.Err = err
		}
	}
	// Get the properties.
	if rw.record.Err == nil {
		for i, p := range rw.properties {
			if p.Name == "" {
				continue
			}
			v, err := normalize(p.Name, p.Type, record[i], p.Nullable, rw.timeLayouts)
			if err != nil {
				rw.record.Err = err
				break
			}
			rw.record.Properties[p.Name] = v
		}
	}
	rw.limit--
	if rw.limit == 0 {
		return errRecordStop
	}
	return nil
}

// RecordStrings writes a record represented as a slice of strings. The slice
// length must not exceed the number of columns, and record must not be nil.
//
// RecordStrings may modify the elements of the record.
func (rw *recordWriter) RecordStrings(record []string) error {
	if rw.properties == nil {
		return fmt.Errorf("connector %s did not call the Columns method before calling RecordStrings", rw.format)
	}
	if len(record) > len(rw.properties) {
		return fmt.Errorf("connector %s has returned a record with %d elements, but there are %d columns", rw.format, len(record), len(rw.properties))
	}
	// Ensure the record has the same number of elements as the columns.
	if n := len(record); n < len(rw.properties) {
		record = slices.Grow(record, len(rw.properties)-len(record))[:len(rw.properties)]
		for c := n; c < len(rw.properties); c++ {
			record[c] = ""
		}
	}
	// Get the last change time.
	var err error
	lastChangeTime := rw.storageLastChangeTime
	if i := rw.lastChangeTimeIndex; i >= 0 {
		p := rw.properties[i]
		var t time.Time
		t, err = parseLastChangeTimeColumn(p.Name, p.Type, rw.action.LastChangeTimeFormat, record[i], p.Nullable, rw.timeLayouts)
		if err == nil {
			if !t.IsZero() {
				lastChangeTime = t
			}
			if !rw.startTime.IsZero() && lastChangeTime.Before(rw.startTime) {
				// Skip the record because it is older than the specified starting time.
				return nil
			}
		}
	}
	// Call the yield function passing the previous record.
	if rw.record.Properties != nil || rw.record.Err != nil {
		if !rw.yield(rw.record) {
			return errRecordStop
		}
	}
	rw.record = Record{
		Properties:     make(map[string]any, rw.numPropertiesPerRecord),
		LastChangeTime: lastChangeTime,
		Err:            err,
	}
	// Get the identity column.
	if i := rw.identityColumnIndex; i >= 0 {
		p := rw.properties[i]
		rw.record.ID, err = parseIdentityColumn(p.Name, p.Type, record[i], rw.timeLayouts)
		if err != nil {
			rw.record.Err = err
		}
	}
	// Get the properties.
	if rw.record.Err == nil {
		for i, p := range rw.properties {
			if p.Name == "" {
				continue
			}
			v, err := normalize(p.Name, p.Type, record[i], p.Nullable, rw.timeLayouts)
			if err != nil {
				rw.record.Err = err
				break
			}
			rw.record.Properties[p.Name] = v
		}
	}
	rw.limit--
	if rw.limit == 0 {
		return errRecordStop
	}
	return nil
}

// close closes the record writer. It should be called when there are no more
// records to write. If there is a last record, it calls the yield function with
// that record. After close is called, no other methods of the record writer
// should be called except for the 'last' method.
func (rw *recordWriter) close() {
	if rw.record.Properties == nil && rw.record.Err == nil {
		return
	}
	rw.isLast = true
	rw.yield(rw.record)
	rw.record.Properties = nil
	rw.record.Err = nil
}

// last reports whether the record passed to the yield function is the last
// record. It should be only called in the yield function.
func (rw *recordWriter) last() bool {
	return rw.isLast
}

// setYieldFunc sets the yield function that will be called for each written
// record. setYieldFunc must be set before calling the Record, RecordSlice, and
// RecordStrings methods.
func (rw *recordWriter) setYieldFunc(yield func(Record) bool) {
	rw.yield = yield
}

// newTimeoutReader returns a TimeoutReader that reads from r. If more than
// the specified timeout elapses between two Read calls, it invokes f.
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

// storageWriteCloser wraps an io.Writer and implements io.WriteCloser. It calls a
// specified function when Close is invoked.
type storageWriteCloser struct {
	io.Writer
	close func(err error) error
}

// Close closes the underlying writer. Storage will receive io.EOF from a read.
// It returns the error returned by the storage if any.
func (c storageWriteCloser) Close() error {
	return c.close(nil)
}

// CloseWithError closes the underlying writer. Storage will receive err as the
// error from a read, or io.EOF if err is nil.
// It returns the error returned by the storage if any.
func (c storageWriteCloser) CloseWithError(err error) error {
	return c.close(err)
}

// removeTempFile removes fi and returns the error, if any. Any error
// encountered will be logged.
func removeTempFile(fi *os.File) error {
	err := fi.Close()
	if err := os.Remove(fi.Name()); err != nil {
		slog.Warn("core/connectors: cannot remove temporary file", "path", fi.Name(), "err", err)
	}
	return err
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

// zipWriter wraps a Writer and implements the Close method that closes a
// zip.Writer when called.
type zipWriter struct {
	z *zip.Writer
	io.Writer
}

func (zw zipWriter) Close() error {
	return zw.z.Close()
}
