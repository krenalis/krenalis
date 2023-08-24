//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	pkgPath "path"
	"time"

	"chichi/apis/errors"
	"chichi/apis/mappings"
	"chichi/apis/normalization"
	"chichi/apis/state"
	"chichi/apis/userswarehouse"
	"chichi/connector"
	"chichi/connector/types"

	"github.com/golang/snappy"
)

var (
	errNoColumns       = errors.New("file does not contain columns")
	errEmptyColumnName = errors.New("file contains an empty column name")

	// errReadStopped is returned to a file connector when it calls w.Write and the
	// storage has already finished reading without an error.
	// If this error occurs, it indicates a bug in the storage connector.
	errReadStopped = errors.New("storage abruptly stopped reading")
)

// An invalidColumnNameError error is returned by the recordWriter.Columns
// method when a column name is not valid.
type invalidColumnNameError struct {
	name string
}

func (err invalidColumnNameError) Error() string {
	return fmt.Sprintf("file contains an invalid column name: %q", err.name)
}

// A sameColumnNameError error is returned by the recordWriter.Columns method
// when two columns have the same name.
type sameColumnNameError struct {
	name string
}

func (err sameColumnNameError) Error() string {
	return fmt.Sprintf("file contains columns with the same name: %s", err.name)
}

// exportUsersToFile exports the users to the file.
func (this *Action) exportUsersToFile(ctx context.Context) error {

	users, err := this.readUsersFromDataWarehouse(ctx, nil)
	if err != nil {
		return err
	}

	// Filter the users.
	if this.action.Filter != nil {
		filteredUsers := []userToExport{}
		for _, user := range users {
			ok, err := mappings.ActionFilterApplies(this.action.Filter, user.Properties)
			if err != nil {
				return err
			}
			if ok {
				filteredUsers = append(filteredUsers, user)
			}
		}
		users = filteredUsers
	}

	connection := this.action.Connection()

	// Retrieve the storage associated to the file connection.
	var storage *compressorStorage
	{
		st, err := this.connection.openStorage(ctx)
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot connect to the storage connector: %w", err)}
		}
		storage = newCompressedStorage(st, connection.Compression)
	}

	file, err := this.connection.openFile(ctx)
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %w", err)}
	}

	// Determine the columns.
	var columns []types.Property
	if len(users) > 0 {
		userSchema, ok := connection.Workspace().Schemas["users"]
		if !ok {
			return actionExecutionError{errors.New("'users' schema not found")}
		}
		for _, p := range userSchema.Properties() {
			if _, ok := users[0].Properties[p.Name]; ok {
				columns = append(columns, p)
			}
		}
	}

	// Prepare the users and the record reader.
	usersSlices := make([][]any, len(users))
	for i, u := range users {
		userSlice := make([]any, len(columns))
		for j, c := range columns {
			userSlice[j] = u.Properties[c.Name]
		}
		usersSlices[i] = userSlice
	}
	records := newRecordReader(columns, usersSlices)

	// Write the file to the storage.
	w, err := storage.Writer(this.action.Path, file.ContentType())
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot write file: %w", err)}
	}
	err = file.Write(w, this.action.Sheet, records)
	if err2 := w.CloseWithError(err); err2 != nil && err == nil {
		err = err2
	}
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot write file: %w", err)}
	}

	return nil
}

// importFromFile imports the users from a file.
func (this *Action) importFromFile(ctx context.Context) error {

	// Connect to the file connector.
	file, err := this.connection.openFile(ctx)
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the file connector: %w", err)}
	}

	// Open the file.
	var r io.ReadCloser
	{
		storage, err := this.connection.openStorage(ctx)
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot connect to the storage connector: %w", err)}
		}
		r, _, err = storage.Reader(this.action.Path)
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot get ReadCloser from storage: %w", err)}
		}
		defer r.Close()
	}

	// Determine the input and the output schema.
	mapping, err := mappings.New(this.action.InSchema, this.action.OutSchema, this.action.Mapping, this.action.Transformation, false)
	if err != nil {
		return err
	}

	inSchemaProps := this.action.InSchema.PropertiesNames()

	// Read the records.
	c := this.connection
	rw := newRecordWriter(c.ID, math.MaxInt, func(record map[string]any) error {

		// Take only the necessary properties.
		props := make(map[string]any, len(inSchemaProps))
		for _, name := range inSchemaProps {
			if v, ok := record[name]; ok {
				props[name] = v
			}
		}

		// Normalize the user properties (read from the file) using the action's
		// mapping input schema.
		props, err := normalize(props, this.action.InSchema)
		if err != nil {
			return actionExecutionError{err}
		}

		// Map the properties of the user.
		mappedUser, err := mapping.Apply(ctx, props)
		if err != nil {
			return actionExecutionError{err}
		}

		// Set the user into the data warehouse.
		err = userswarehouse.SetUser(ctx, c.store, this.action, mappedUser)
		if err != nil {
			return actionExecutionError{err}
		}

		// Update the connection stats.
		err = this.connection.updateConnectionsStats(ctx)
		if err != nil {
			return actionExecutionError{err}
		}

		return nil
	})
	err = file.Read(r, this.action.Sheet, rw)
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot read the file: %w", err)}
	}
	err = r.Close()
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot close the storage: %w", err)}
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
		return errNoColumns
	}
	labelToName := make(map[string]string, len(columns))
	hasName := make(map[string]struct{}, len(columns))
	for _, c := range columns {
		if c.Name == "" {
			return errEmptyColumnName
		}
		if !types.IsValidPropertyName(c.Name) {
			return invalidColumnNameError{c.Name}
		}
		if _, ok := hasName[c.Name]; ok {
			return sameColumnNameError{c.Name}
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
			rd[i], err = normalization.NormalizeDatabaseFileProperty(c.Name, c.Type, record[i], c.Nullable)
			if err != nil {
				return err
			}
		}
		rw.records = append(rw.records, rd)
	} else {
		// Call the rw.write function to store the record.
		rd := map[string]any{}
		for i, c := range rw.columns {
			rd[c.Name], err = normalization.NormalizeDatabaseFileProperty(c.Name, c.Type, record[i], c.Nullable)
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
			rd[i], err = normalization.NormalizeDatabaseFileProperty(c.Name, c.Type, record[c.Name], c.Nullable)
			if err != nil {
				return err
			}
		}
		rw.records = append(rw.records, rd)
	} else {
		// Call the rw.write function to store the record.
		for _, c := range rw.columns {
			v, err := normalization.NormalizeDatabaseFileProperty(c.Name, c.Type, record[c.Name], c.Nullable)
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

// compressorStorage implements a storage capable of compressing and
// decompressing data read from or written to a connector.StorageConnection.
type compressorStorage struct {
	storage     connector.StorageConnection
	compression state.Compression
}

// newCompressedStorage returns a compressor storage that wraps s and performs
// file compression and decompression using c as the compression method.
// If c is NoCompression, it does not perform any compression or decompression.
func newCompressedStorage(s connector.StorageConnection, c state.Compression) *compressorStorage {
	return &compressorStorage{s, c}
}

// Reader opens the file at the given path name and returns a ReadCloser from
// which to read the file and its last update time.
// It is the caller's responsibility to close the returned reader.
func (cs compressorStorage) Reader(name string) (io.ReadCloser, time.Time, error) {
	originalName := name
	ext := cs.compression.Ext()
	name += ext
	r, t, err := cs.storage.Reader(name)
	if err != nil {
		return nil, time.Time{}, err
	}
	switch cs.compression {
	case state.ZipCompression:
		var err error
		var fi *os.File
		defer func() {
			if err != nil {
				if fi != nil {
					_ = closeTempFile(fi)
				}
				_ = r.Close()
			}
		}()
		fi, err = os.CreateTemp("", "")
		if err != nil {
			return nil, time.Time{}, err
		}
		st, err := fi.Stat()
		if err != nil {
			return nil, time.Time{}, err
		}
		z, err := zip.NewReader(fi, st.Size())
		if err != nil {
			return nil, time.Time{}, err
		}
		name := pkgPath.Base(originalName)
		r3, err := z.Open(name)
		if err != nil {
			return nil, time.Time{}, err
		}
		r = newFuncReadCloser(r3, func() error {
			err3 := r3.Close()
			err2 := closeTempFile(fi)
			err := r.Close()
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
func (cs compressorStorage) Writer(path, contentType string) (*storageWriteCloser, error) {
	pr, pw := io.Pipe()
	var w io.WriteCloser
	switch cs.compression {
	case state.NoCompression:
		w = pw
	case state.ZipCompression:
		z := zip.NewWriter(pw)
		name := pkgPath.Base(path)
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
		err := cs.storage.Write(pr, path, contentType)
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

// closeTempFile closes fi and returns the error, if any. Any error encountered
// will be logged.
func closeTempFile(fi *os.File) error {
	err := fi.Close()
	if err := os.Remove(fi.Name()); err != nil {
		log.Printf("[warning] cannot remove temporary file %q: %s", fi.Name(), err)
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
