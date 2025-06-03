//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

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

// FileStorageInfo represents a file storage connector info.
type FileStorageInfo struct {
	Name          string
	Categories    Category // bitmask of connector's categories
	AsSource      *AsFileStorageSource
	AsDestination *AsFileStorageDestination
	Icon          string // icon in SVG format

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
// connector info.
func (info FileStorageInfo) ReflectType() reflect.Type {
	return info.ct
}

// New returns a new file storage connector instance.
func (info FileStorageInfo) New(conf *FileStorageConfig) (any, error) {
	out := info.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface()
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
type FileStorageNewFunc[T any] func(*FileStorageConfig) (T, error)
