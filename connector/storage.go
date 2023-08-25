//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"time"
)

// An InvalidPathError value is returned by StorageConnection.CompletePath when
// the path name is not valid for the storage connection.
type InvalidPathError struct {
	err error
}

// InvalidPathErrorf formats according to a format specifier and returns a
// InvalidPathError value.
func InvalidPathErrorf(format string, a ...any) error {
	return InvalidPathError{fmt.Errorf(format, a...)}
}

func (err InvalidPathError) Error() string {
	return err.err.Error()
}

// Storage represents a storage connector.
type Storage struct {
	Name                   string
	SourceDescription      string // It should complete the sentence "Add an action to ..."
	DestinationDescription string // It should complete the sentence "Add an action to ..."
	Icon                   string // icon in SVG format

	open reflect.Value
	ct   reflect.Type
}

// ConnectionReflectType returns the type of the value implementing the storage
// connection.
func (storage Storage) ConnectionReflectType() reflect.Type {
	return storage.ct
}

// Open opens a storage connection.
func (storage Storage) Open(conf *StorageConfig) (StorageConnection, error) {
	out := storage.open.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface().(StorageConnection)
	err, _ := out[1].Interface().(error)
	return c, err
}

// StorageConfig represents the configuration of a storage connection.
type StorageConfig struct {
	Role        Role
	Settings    []byte
	SetSettings SetSettingsFunc
}

// OpenStorageFunc represents functions that open storage connections.
type OpenStorageFunc[T StorageConnection] func(*StorageConfig) (T, error)

// StorageConnection is the interface implemented by storage connections.
type StorageConnection interface {

	// CompletePath returns the complete representation of the given path name or an
	// InvalidPathError if name is not valid for use in calls to Open and Write.
	//
	// name's length in runes will be in range [1, 1024].
	CompletePath(ctx context.Context, name string) (string, error)

	// Reader opens the file at the given path name and returns a ReadCloser from
	// which to read the file and its last update time.
	// It is the caller's responsibility to close the returned reader.
	Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error)

	// Write writes the data read from r into the file with the given path name.
	// contentType is the file's content type.
	Write(ctx context.Context, r io.Reader, name, contentType string) error
}
