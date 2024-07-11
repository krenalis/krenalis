//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package meergo

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"time"
)

// An InvalidPathError value is returned by FileStorage.CompletePath when the
// path name is not valid for the file storage connector.
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

// FileStorageInfo represents a file storage connector info.
type FileStorageInfo struct {
	Name string
	Icon string // icon in SVG format

	newFunc reflect.Value
	ct      reflect.Type
}

// ReflectType returns the type of the value implementing the file storage
// connector info.
func (info FileStorageInfo) ReflectType() reflect.Type {
	return info.ct
}

// New returns a new file storage connector instance.
func (info FileStorageInfo) New(conf *FileStorageConfig) (FileStorage, error) {
	out := info.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface().(FileStorage)
	err, _ := out[1].Interface().(error)
	return c, err
}

// FileStorageConfig represents the configuration of a file storage connector.
type FileStorageConfig struct {
	Settings    []byte
	SetSettings SetSettingsFunc
}

// FileStorageNewFunc represents functions that create new file storage
// connector instances.
type FileStorageNewFunc[T FileStorage] func(*FileStorageConfig) (T, error)

// FileStorage is the interface implemented by file storage connectors.
type FileStorage interface {

	// CompletePath returns the complete representation of the given path name. It
	// returns InvalidPathError if name is not valid for use in calls to Reader and
	// Write.
	//
	// name's length in runes will be in range [1, 1024].
	CompletePath(ctx context.Context, name string) (string, error)

	// Reader opens a file and returns a ReadCloser from which to read its content.
	// name is the path name of the file to read and the returned time.Time is the
	// last update time of the file.
	//
	// The use of the provided context is extended to the Read method calls.
	// After the context is canceled, any subsequent Read invocations will result in
	// an error. It is the caller's responsibility to close the returned reader.
	Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error)

	// Write writes the data read from r into the file with the given path name.
	// contentType is the file's content type.
	Write(ctx context.Context, r io.Reader, name, contentType string) error
}
