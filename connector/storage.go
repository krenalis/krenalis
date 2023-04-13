//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import (
	"context"
	"io"
	"reflect"
	"time"
)

// Storage represents a storage connector.
type Storage struct {
	Name                   string
	SourceDescription      string // It should complete the sentence "Add an action to ..."
	DestinationDescription string // It should complete the sentence "Add an action to ..."
	Icon                   string // icon in SVG format

	open reflect.Value
}

// Open opens a storage connection.
func (storage Storage) Open(ctx context.Context, conf *StorageConfig) (StorageConnection, error) {
	out := storage.open.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(conf)})
	c := out[0].Interface().(StorageConnection)
	err, _ := out[1].Interface().(error)
	return c, err
}

// StorageConfig represents the configuration of a storage connection.
type StorageConfig struct {
	Role     Role
	Settings []byte
	Firehose Firehose
}

// OpenStorageFunc represents functions that open storage connections.
type OpenStorageFunc[T StorageConnection] func(context.Context, *StorageConfig) (T, error)

// StorageConnection is the interface implemented by storage connections.
type StorageConnection interface {

	// Reader returns a ReadCloser from which to read the file with the given path
	// and its last update time.
	// It is the caller's responsibility to close the returned reader.
	Reader(path string) (io.ReadCloser, time.Time, error)

	// Write writes the data read from p into the file with the given path.
	// contentType is the file's content type.
	Write(p io.Reader, path, contentType string) error
}
