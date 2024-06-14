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
	"time"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/types"
)

// An InvalidPathError is returned when a path name is not valid.
type InvalidPathError = chichi.InvalidPathError

type FileStorage struct {
	state   *state.State
	storage *state.Connection
	inner   chichi.FileStorage
	err     error
}

// FileStorage returns a file storage on the provided file storage connection.
// Errors are deferred until a file storage's method is called.
func (connectors *Connectors) FileStorage(storage *state.Connection) *FileStorage {
	s := &FileStorage{
		state:   connectors.state,
		storage: storage,
	}
	s.inner, s.err = chichi.RegisteredFileStorage(storage.Connector().Name).New(&chichi.FileStorageConfig{
		Settings:    storage.Settings,
		SetSettings: setConnectionSettingsFunc(connectors.state, storage),
	})
	return s
}

// CompletePath returns the complete representation of the provided path name or
// an InvalidPathError value if name is not valid for use in calls to Read and
// Write. name's length in runes must be in range [1, 1024].
//
// If nameReplacer is not nil, then the placeholders in name are replaced using
// it; in this case, a PlaceholderError error may be returned in case of an
// error with placeholders.
func (storage *FileStorage) CompletePath(ctx context.Context, name string, nameReplacer PlaceholderReplacer) (string, error) {
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
	return storage.inner.CompletePath(ctx, name)
}

// Read reads the records from file in the storage at the provided path name
// and returns the columns and the records. name must be UTF-8 encoded with a
// length in range [1, 1024].
//
// file refers to the file connector to use. If it supports multiple sheets,
// sheet is a valid sheet name; otherwise, it must be an empty string. A valid
// sheet name is UTF-8 encoded, has a length in the range [1, 31], does not
// start or end with "'", and does not contain any of "*", "/", ":", "?", "[",
// "\", and "]". Sheet names are case-insensitive.
//
// uiValues, if the file connector has a UI, represents the user-entered values
// as a JSON object. compression indicates if the file is compressed and how,
// and limit restricts the number of records to return. If limit is negative,
// there is no upper limit on the number of records returned.
//
// If the UI values are not valid, it returns an InvalidUIValuesError error.
// If the file has no columns, it returns the ErrNoColumns error. If the file
// does not have the provided sheet, it returns the ErrSheetNotExist error.
func (storage *FileStorage) Read(ctx context.Context, file *state.Connector, name, sheet string, uiValues []byte, compression state.Compression, limit int) (columns []types.Property, rows []map[string]any, err error) {
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
	if err = validateLastChangeTime(storageTimestamp); err != nil {
		return nil, nil, fmt.Errorf("invalid timestamp returned by the storage: %s", err)
	}

	_file, err := chichi.RegisteredFile(file.Name).New(&chichi.FileConfig{
		SetSettings: func(ctx context.Context, settings []byte) error { return nil },
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to register the file: %s", err)
	}
	if file.HasUI {
		_, err = _file.(chichi.UIHandler).ServeUI(ctx, "save", uiValues, chichi.Role(storage.storage.Role))
		if err != nil {
			return nil, nil, err
		}
	}

	rw := newRecordWriter(file.Name, types.Type{}, "", LastChangeTimeProperty{}, storageTimestamp, &file.TimeLayouts, time.Time{}, limit)
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
	err = _file.Read(ctx, r, sheet, rw)
	rw.close()
	if err != nil && err != errRecordStop {
		if err == chichi.ErrSheetNotExist {
			err = ErrSheetNotExist
		}
		return nil, nil, err
	}
	if err = r.Close(); err != nil {
		return nil, nil, err
	}
	if recordErr != nil {
		return nil, nil, recordErr
	}
	if rw.properties == nil {
		return nil, nil, ErrNoColumns
	}
	return rw.properties, records, nil
}

// Sheets returns the sheets of the file with the provided name. Sheet names
// are case-insensitive.
//
// uiValues, if the file connector has a UI, represents the user-entered values
// as a JSON object. compression indicates if the file is compressed and how.
//
// If the UI values are not valid, it returns an InvalidUIValuesError error.
// It panics if the file connector does not support sheets.
func (storage *FileStorage) Sheets(ctx context.Context, file *state.Connector, name string, uiValues []byte, compression state.Compression) ([]string, error) {
	if storage.err != nil {
		return nil, storage.err
	}

	_file, err := chichi.RegisteredFile(file.Name).New(&chichi.FileConfig{
		SetSettings: func(ctx context.Context, settings []byte) error { return nil },
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register the file: %s", err)
	}
	if file.HasUI {
		_, err = _file.(chichi.UIHandler).ServeUI(ctx, "save", uiValues, chichi.Role(storage.storage.Role))
		if err != nil {
			return nil, err
		}
	}

	sheetsFile := _file.(chichi.Sheets)
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
