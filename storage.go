//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package chichi

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"time"
)

// An InvalidPathError value is returned by Storage.CompletePath when the path
// name is not valid for the storage connector.
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

// StorageInfo represents a storage connector info.
type StorageInfo struct {
	Name                   string
	SourceDescription      string // It should complete the sentence "Add an action to ..."
	DestinationDescription string // It should complete the sentence "Add an action to ..."
	Icon                   string // icon in SVG format

	newFunc reflect.Value
	ct      reflect.Type
}

// ReflectType returns the type of the value implementing the storage connector
// info.
func (info StorageInfo) ReflectType() reflect.Type {
	return info.ct
}

// New returns a new storage connector instance.
func (info StorageInfo) New(conf *StorageConfig) (Storage, error) {
	out := info.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface().(Storage)
	err, _ := out[1].Interface().(error)
	return c, err
}

// StorageConfig represents the configuration of a storage connector.
type StorageConfig struct {
	Role        Role
	Settings    []byte
	SetSettings SetSettingsFunc
}

// StorageNewFunc represents functions that create new storage connector
// instances.
type StorageNewFunc[T Storage] func(*StorageConfig) (T, error)

// Storage is the interface implemented by storage connectors.
type Storage interface {

	// CompletePath returns the complete representation of the given path name or an
	// InvalidPathError if name is not valid for use in calls to Reader and Write.
	//
	// name's length in runes will be in range [1, 1024].
	CompletePath(ctx context.Context, name string) (string, error)

	// Reader opens the file at the given path name and returns a ReadCloser from
	// which to read the file and its last update time. The use of the provided
	// context is extended to the Read method calls. After the context is canceled,
	// any subsequent Read invocations will result in an error.
	// It is the caller's responsibility to close the returned reader.
	Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error)

	// Write writes the data read from r into the file with the given path name.
	// contentType is the file's content type.
	Write(ctx context.Context, r io.Reader, name, contentType string) error
}
