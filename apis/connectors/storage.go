//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package connectors

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"

	"chichi/apis/state"
	_connector "chichi/connector"
	"chichi/connector/types"
)

type Storage struct {
	state   *state.State
	storage *state.Connection
	inner   _connector.StorageConnection
	err     error
}

// Storage returns a storage on the provided connection storage. Errors are deferred
// until a storage's method is called.
func (connectors *Connectors) Storage(storage *state.Connection) *Storage {
	s := &Storage{
		state:   connectors.state,
		storage: storage,
	}
	s.inner, s.err = _connector.RegisteredStorage(storage.Connector().Name).New(&_connector.StorageConfig{
		Role:        _connector.Role(storage.Role),
		Settings:    storage.Settings,
		SetSettings: setConnectionSettingsFunc(connectors.state, storage),
	})
	return s
}

// CompletePath returns the complete representation of the provided path name or
// an InvalidPathError value if name is not valid for use in calls to Read and
// Write. name's length in runes must be in range [1, 1024].
func (storage *Storage) CompletePath(ctx context.Context, name string) (string, error) {
	if storage.err != nil {
		return "", storage.err
	}
	return storage.inner.CompletePath(ctx, name)
}

// Read reads the records from file in the storage at the provided path name and returns
// the columns and the records. name must be UTF-8 encoded with a length in range [1,
// 1024].
//
// If the file connector supports multiple sheets, sheet is a valid sheet name;
// otherwise, it must be an empty string. A valid sheet name is UTF-8 encoded,
// has a length in the range [1, 31], does not start or end with "'", and does
// not contain any of "*", "/", ":", "?", "[", "\", and "]". Sheet names are
// case-insensitive.
//
// businessIDColumn, when not empty, is the column from which the Business ID
// should be read.
//
// limit restricts the number of records to return and should not exceed 100. If limit is
// negative, there is no upper limit on the number of records returned.
//
// If the file has no columns, it returns the ErrNoColumns error. If the file
// does not have the provided sheet, it returns the ErrSheetNotExist error.
func (storage *Storage) Read(ctx context.Context, file *state.Connector, name, sheet string, settings []byte, businessIDColumn string, compression state.Compression, limit int) (columns []types.Property, rows []map[string]any, err error) {
	if storage.err != nil {
		return nil, nil, storage.err
	}
	if limit < 0 {
		limit = math.MaxInt
	}
	s := newCompressedStorage(storage.inner, compression)
	r, storageTimestamp, err := s.Reader(ctx, name)
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()
	if err = validateTimestamp(storageTimestamp); err != nil {
		return nil, nil, fmt.Errorf("invalid timestamp returned by the storage: %s", err)
	}

	_file, err := _connector.RegisteredFile(file.Name).New(&_connector.FileConfig{
		Role:     _connector.Role(storage.storage.Role),
		Settings: settings,
	})

	rw := newRecordWriter(file.ID, types.Type{}, "", TimestampColumn{}, businessIDColumn, storageTimestamp, limit)
	err = _file.Read(ctx, r, sheet, rw)
	if err != nil && err != errRecordStop {
		if err == _connector.ErrSheetNotExist {
			err = ErrSheetNotExist
		}
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

// Sheets returns the sheets of the file with the provided name. Sheet names
// are case-insensitive. It panics if the file connector does not support
// sheets.
func (storage *Storage) Sheets(ctx context.Context, file *state.Connector, name string, settings []byte, compression state.Compression) ([]string, error) {
	if storage.err != nil {
		return nil, storage.err
	}

	_file, err := _connector.RegisteredFile(file.Name).New(&_connector.FileConfig{
		Role:     _connector.Role(storage.storage.Role),
		Settings: settings,
	})
	if err != nil {
		return nil, err
	}

	sheetsFile := _file.(_connector.Sheets)
	s := newCompressedStorage(storage.inner, compression)
	r, _, err := s.Reader(ctx, name)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	sheets, err := sheetsFile.Sheets(ctx, r)
	if err != nil {
		return nil, err
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
