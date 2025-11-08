// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connectors

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"time"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/core/types"
)

type FileStorage struct {
	connector string
	state     *state.State
	storage   *state.Connection
	inner     any
	err       error
}

type fileStorageAbsolutePathConnector interface {
	// AbsolutePath returns the absolute representation of the given path name. It
	// returns *InvalidPathError if name is not valid for use in calls to Reader and
	// Write.
	//
	// name's length in runes will be in range [1, 1024].
	AbsolutePath(ctx context.Context, name string) (string, error)
}

type fileStorageReaderConnector interface {
	// Reader opens a file and returns a ReadCloser from which to read its content.
	// name is the path name of the file to read and the returned time.Time is the
	// last update time of the file.
	//
	// The use of the provided context is extended to the Read method calls.
	// After the context is canceled, any subsequent Read invocations will result in
	// an error. It is the caller's responsibility to close the returned reader.
	Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error)
}

type fileStorageWriteConnector interface {
	// Write writes the data read from r into the file with the given path name.
	// contentType is the file's content type.
	Write(ctx context.Context, r io.Reader, name, contentType string) error
}

// FileStorage returns a file storage on the provided file storage connection.
// Errors are deferred until a file storage's method is called.
func (c *Connectors) FileStorage(storage *state.Connection) *FileStorage {
	s := &FileStorage{
		connector: storage.Connector().Code,
		state:     c.state,
		storage:   storage,
	}
	s.inner, s.err = connectors.RegisteredFileStorage(storage.Connector().Code).New(&connectors.FileStorageEnv{
		Settings:    storage.Settings,
		SetSettings: setConnectionSettingsFunc(c.state, storage),
	})
	s.err = connectorError(s.err)
	return s
}

// AbsolutePath returns the absolute representation of the provided path name or
// an InvalidPathError if name is not valid for use in calls to Read and Write.
// The length of name in runes must be within [1, 1024].
//
// If nameReplacer is not nil, then the placeholders in name are replaced using
// it; in this case, a *PlaceholderError error may be returned in case of an
// error with placeholders.
//
// If the connector returns an error, it returns an *UnavailableError.
func (storage *FileStorage) AbsolutePath(ctx context.Context, name string, nameReplacer PlaceholderReplacer) (string, error) {
	if storage.err != nil {
		return "", storage.err
	}
	if nameReplacer != nil {
		var err error
		name, err = ReplacePlaceholders(name, nameReplacer)
		if err != nil {
			return "", err
		}
	}
	path, err := storage.inner.(fileStorageAbsolutePathConnector).AbsolutePath(ctx, name)
	if err != nil {
		return "", connectorError(err)
	}
	return path, nil
}

// Connector returns the name of the file storage connector.
func (storage *FileStorage) Connector() string {
	return storage.connector
}

// Read reads the records from the file located in the storage at the provided
// path and returns the columns and records. name must be UTF-8 encoded with a
// length in the range [1, 1024].
//
// This method can only be called if both the FileStorage and the file allow
// reading; otherwise, it will panic.
//
// file refers to the file connector to use. It must support reading of records.
// If it supports multiple sheets, sheet is a valid sheet name; otherwise, it
// must be an empty string. A valid sheet name is UTF-8 encoded, has a length in
// the range [1, 31], does not start or end with "'", and does not contain any
// of "*", "/", ":", "?", "[", "\", and "]". Sheet names are case-insensitive.
//
// settings contains the file connector settings, if any. compression indicates
// whether and how the file is compressed, and limit restricts the number of
// records returned. If limit is negative, there is no upper bound.
//
// The method may also return issues encountered during the reading process that
// did not prevent the file from being processed. These issues are reported as a
// slice of strings. The slice will be nil if there are no issues.
//
// If the settings are invalid, it returns a *connectors.InvalidSettingsError. If
// the file has no columns, it returns ErrNoColumnsFound. If the file does not
// have the specified sheet, it returns connectors.ErrSheetNotExist. If the
// connector returns an error, it returns an *UnavailableError.
func (storage *FileStorage) Read(ctx context.Context, file *state.Connector, name, sheet string, settings json.Value, compression state.Compression, limit int) (columns []types.Property, rows []map[string]any, issues []string, err error) {
	if storage.err != nil {
		return nil, nil, nil, storage.err
	}
	if limit < 0 {
		limit = math.MaxInt
	}
	s := newCompressedStorage(storage.inner, compression)
	r, storageTimestamp, err := s.Reader(ctx, name)
	if err != nil {
		return nil, nil, nil, connectorError(err)
	}
	defer r.Close()
	if err = validateLastChangeTime(storageTimestamp); err != nil {
		return nil, nil, nil, fmt.Errorf("invalid timestamp returned by the storage: %s", err)
	}

	_file, err := connectors.RegisteredFile(file.Code).New(&connectors.FileEnv{
		SetSettings: func(ctx context.Context, innerSettings []byte) error { return nil },
	})
	if err != nil {
		return nil, nil, nil, connectorError(fmt.Errorf("failed to register the file: %s", err))
	}
	if file.HasSourceSettings {
		_, err = _file.(uiHandlerConnector).ServeUI(ctx, "save", settings, connectors.Role(storage.storage.Role))
		if err != nil {
			return nil, nil, nil, connectorError(err)
		}
	}

	rw := newRecordWriter(file.Code, nil, storageTimestamp, &file.TimeLayouts, time.Time{}, limit)
	var records []map[string]any
	var recordErr error
	rw.setYieldFunc(func(record Record) bool {
		if record.Err != nil {
			recordErr = record.Err
			return false
		}
		records = append(records, record.Properties)
		return true
	})
	err = _file.(fileReadConnector).Read(ctx, r, sheet, rw)
	rw.close()
	if err != nil && err != errRecordStop {
		return nil, nil, nil, connectorError(err)
	}
	if err = r.Close(); err != nil {
		return nil, nil, nil, connectorError(err)
	}
	if recordErr != nil {
		return nil, nil, nil, connectorError(recordErr)
	}
	if rw.properties == nil {
		return nil, nil, nil, ErrNoColumnsFound
	}
	if len(rw.issues) > 0 {
		issues = rw.issues
	}
	return rw.properties, records, issues, nil
}

// Sheets returns the sheets of the file with the given name. Sheet names are
// case-insensitive.
//
// If the file does not have sheets, this method panics.
//
// settings, if the file connector has settings, represents its settings.
// compression indicates if the file is compressed and how.
//
// If the settings are invalid, it returns a *connectors.InvalidSettingsError. If
// the connector returns an error, it returns an *UnavailableError. This method
// panics if the file connector does not support sheets.
func (storage *FileStorage) Sheets(ctx context.Context, file *state.Connector, name string, settings json.Value, compression state.Compression) ([]string, error) {
	if storage.err != nil {
		return nil, storage.err
	}

	_file, err := connectors.RegisteredFile(file.Code).New(&connectors.FileEnv{
		SetSettings: func(ctx context.Context, settings []byte) error { return nil },
	})
	if err != nil {
		return nil, connectorError(fmt.Errorf("failed to register the file: %s", err))
	}
	if file.HasSourceSettings {
		_, err = _file.(uiHandlerConnector).ServeUI(ctx, "save", settings, connectors.Role(storage.storage.Role))
		if err != nil {
			return nil, connectorError(err)
		}
	}

	sheetsFile := _file.(fileSheetConnector)
	s := newCompressedStorage(storage.inner, compression)
	r, _, err := s.Reader(ctx, name)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	sheets, err := sheetsFile.Sheets(ctx, r)
	if err != nil {
		return nil, connectorError(err)
	}
	if err = r.Close(); err != nil {
		return nil, err
	}
	sheets = slices.DeleteFunc(sheets, func(name string) bool {
		return !IsValidSheetName(name)
	})
	if len(sheets) == 0 {
		return nil, errors.New("file does not contain any valid sheet names")
	}
	return sheets, nil
}
