// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package connectors

import (
	"reflect"

	"github.com/krenalis/krenalis/tools/types"
)

// DatabaseSpec represents a database connector specification.
type DatabaseSpec struct {
	Code          string
	Label         string
	Categories    Categories  // categories
	SampleQuery   string      // sample query
	TimeLayouts   TimeLayouts // layouts for time values. If left empty, it is ISO 8601.
	Documentation Documentation

	newFunc reflect.Value
	ct      reflect.Type
}

// ReflectType returns the type of the value implementing the database
// connector specification.
func (spec DatabaseSpec) ReflectType() reflect.Type {
	return spec.ct
}

// New returns a new database connector instance.
func (spec DatabaseSpec) New(env *DatabaseEnv) (any, error) {
	out := spec.newFunc.Call([]reflect.Value{reflect.ValueOf(env)})
	c := out[0].Interface()
	err, _ := reflect.TypeAssert[error](out[1])
	return c, err
}

// DatabaseEnv is the environment for a database connector.
type DatabaseEnv struct {

	// Settings holds the settings.
	Settings SettingsStore
}

// DatabaseNewFunc represents functions that create new database connector
// instances.
type DatabaseNewFunc[T any] func(*DatabaseEnv) (T, error)

// Table represents a database table.
type Table struct {
	Name    string
	Columns []Column
	Keys    []string
}

// Column represents a database table column. If Type is invalid, Issue
// describes the problem, and the other fields are not meaningful.
type Column struct {
	Name     string     // column name
	Type     types.Type // data type of the column
	Nullable bool       // true if the column can contain NULL values
	Writable bool       // true if the column is writable
	Issue    string     // issue message
}

// Rows is the result of a database query. Its cursor starts before the first
// row of the result set. Use Next to advance from row to row.
type Rows interface {
	Close() error
	Err() error
	Next() bool
	Scan(dest ...any) error
}
