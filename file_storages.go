// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package meergo

import (
	"fmt"
	"reflect"
)

// An InvalidPathError value is returned by FileStorage.AbsolutePath when the
// path name is not valid for the file storage connector.
type InvalidPathError struct {
	err error
}

// InvalidPathErrorf formats according to a format specifier and returns a
// *InvalidPathError value.
func InvalidPathErrorf(format string, a ...any) error {
	return &InvalidPathError{fmt.Errorf(format, a...)}
}

func (err *InvalidPathError) Error() string {
	return err.err.Error()
}

// FileStorageSpec represents a file storage connector specification.
type FileStorageSpec struct {
	Code          string
	Label         string
	Categories    Categories // categories
	AsSource      *AsFileStorageSource
	AsDestination *AsFileStorageDestination

	newFunc reflect.Value
	ct      reflect.Type
}

// AsFileStorageSource represents the specific information of a file storage
// connector used as a source.
type AsFileStorageSource struct {
	Documentation ConnectorRoleDocumentation
}

// AsFileStorageDestination represents the specific information of a file storage
// connector used as a destination.
type AsFileStorageDestination struct {
	Documentation ConnectorRoleDocumentation
}

// ReflectType returns the type of the value implementing the file storage
// connector specification.
func (spec FileStorageSpec) ReflectType() reflect.Type {
	return spec.ct
}

// New returns a new file storage connector instance.
func (spec FileStorageSpec) New(env *FileStorageEnv) (any, error) {
	out := spec.newFunc.Call([]reflect.Value{reflect.ValueOf(env)})
	c := out[0].Interface()
	err, _ := reflect.TypeAssert[error](out[1])
	return c, err
}

// FileStorageEnv is the environment for a file storage connector.
type FileStorageEnv struct {

	// Settings is the raw settings data.
	Settings []byte

	// SetSettings is the function used to update the settings.
	SetSettings SetSettingsFunc
}

// FileStorageNewFunc represents functions that create new file storage
// connector instances.
type FileStorageNewFunc[T any] func(*FileStorageEnv) (T, error)
