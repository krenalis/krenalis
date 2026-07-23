// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package connectors

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
	Documentation RoleDocumentation
}

// AsFileStorageDestination represents the specific information of a file storage
// connector used as a destination.
type AsFileStorageDestination struct {
	Documentation RoleDocumentation
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

	// Settings holds the settings.
	Settings SettingsStore

	// Dial is the function the connector must use to establish its outbound
	// network connections, in place of its own default dialer.
	Dial DialFunc

	// DialWith is the function a connector that has its own dialer must use, in
	// place of Dial, to establish its outbound network connections. It returns a
	// dial function that dials with the given one, so that the connector keeps
	// its own dial options, like its timeouts and its keep-alive.
	//
	// If the given dial function is nil, the returned one dials with a plain
	// dialer, as Dial does.
	DialWith DialWith
}

// FileStorageNewFunc represents functions that create new file storage
// connector instances.
type FileStorageNewFunc[T any] func(*FileStorageEnv) (T, error)
