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
	"math"
	"net/netip"
	"reflect"
	"slices"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
)

// WarehouseDriver represents a warehouse driver.
type WarehouseDriver struct {
	Name string
	Icon string // icon in SVG format

	newFunc reflect.Value
	ct      reflect.Type
}

// ReflectType returns the type of the value implementing the warehouse driver.
func (info WarehouseDriver) ReflectType() reflect.Type {
	return info.ct
}

// New returns a new data warehouse instance.
func (info WarehouseDriver) New(conf *WarehouseConfig) (Warehouse, error) {
	out := info.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	d := out[0].Interface().(Warehouse)
	err, _ := out[1].Interface().(error)
	return d, err
}

// WarehouseConfig represents the configuration of a data warehouse.
type WarehouseConfig struct {
	Settings    []byte
	SetSettings SetSettingsFunc
}

// WarehouseDriverNewFunc represents functions that create new warehouse driver
// instance.
type WarehouseDriverNewFunc[T Warehouse] func(*WarehouseConfig) (T, error)

// AlterOperation represents an operation that alters the columns of the user
// tables.
//
// Every column is always nullable.
type AlterOperation struct {
	Operation AlterOperationType
	Column    string     // For "Add", "Drop" and "Rename" operations.
	Type      types.Type // For "Add" operations. object properties are expanded into single "Add" operations, so Type can never have kind object.
	NewColumn string     // For "Rename" operations.
}

// AlterOperationType represents the type of an operation on the data warehouse
// that alters the columns of the user tables.
type AlterOperationType int

const (
	OperationAddColumn AlterOperationType = iota + 1
	OperationDropColumn
	OperationRenameColumn
)

func (op AlterOperationType) String() string {
	switch op {
	case OperationAddColumn:
		return "AddColumn"
	case OperationDropColumn:
		return "DropColumn"
	case OperationRenameColumn:
		return "RenameColumn"
	default:
		return fmt.Sprintf("<invalid OperationType = %d>", int(op))
	}
}

func (op AlterOperationType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + op.String() + `"`), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (op *AlterOperationType) UnmarshalJSON(data []byte) error {
	var v any
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("json: cannot scan a %T value into a meergo.AlterOperationType value", v)
	}
	switch s {
	case "AddColumn":
		*op = OperationAddColumn
	case "DropColumn":
		*op = OperationDropColumn
	case "RenameColumn":
		*op = OperationRenameColumn
	default:
		return fmt.Errorf("json: invalid meergo.AlterOperationType: %s", s)
	}
	return nil
}

// Warehouse is the interface implemented by warehouse drivers.
type Warehouse interface {

	// AlterUserColumns alters the columns of the user tables.
	//
	// opID is an identifier that uniquely identifies a specific alter columns
	// operation; if the method is called again passing the same identifier, whether
	// the operation ended successfully or with a *meergo.OperationError error, that
	// result is returned again.
	//
	// columns contains the columns of the "users" table to obtain (this parameters
	// is useful for obtaining type information and for creating views), while
	// operations is the set of operations to apply in order to migrate the current
	// columns to the given columns.
	//
	// This method, once called, can then return in four distinct cases:
	//
	// (1) the operation was successful and no error was returned;
	//
	// (2) the context was cancelled;
	//
	// (3) the operation ended with an error of type *meergo.OperationError, and this
	// means that even if the method is called again with the same ID, this error is
	// still returned;
	//
	// (4) the operation ended with an unexpected and unknown error, and it is
	// therefore up to the caller to try calling this method again by providing the
	// same ID.
	AlterUserColumns(ctx context.Context, opID string, columns []Column, operations []AlterOperation) error

	// AlterUserColumnsQueries returns the queries that alter the columns of the
	// user tables.
	//
	// columns contains the columns of the users tables to obtain (this parameters
	// is useful for obtaining type information and for creating views), while
	// operations is the set of operations to apply in order to migrate the current
	// columns to the given columns.
	AlterUserColumnsQueries(ctx context.Context, columns []Column, operations []AlterOperation) ([]string, error)

	// CanInitialize checks whether the data warehouse can be initialized.
	// It returns a *WarehouseNonInitializableError error if the data warehouse
	// cannot be initialized.
	CanInitialize(ctx context.Context) error

	// Close closes the data warehouse. When Close is called, no other calls to
	// data warehouse's methods are in progress and no more will be made.
	Close() error

	// Delete deletes rows from the specified table that match the provided where
	// expression. Returns an error if the expression is nil.
	Delete(ctx context.Context, table string, where Expr) error

	// Initialize initializes the data warehouse.
	// The given user schema will be used by the initialization to build the user
	// tables on the warehouse with the corresponding columns.
	Initialize(ctx context.Context, userColumns []Column) error

	// Merge performs a table merge operation.
	// If handles row updates, inserts, and deletions. table specifies the target
	// table for the merge operation, rows contains the rows to insert or update in
	// the table, and deleted contains the key values of rows to delete, if they
	// exist.
	// rows or deleted can be empty but not both.
	// Note that rows may be changed by this method.
	Merge(ctx context.Context, table Table, rows [][]any, deleted []any) error

	// MergeIdentities merges existing identities, deletes them, and inserts new
	// ones. columns are the columns whose values are present in the rows and
	// contain at least the columns:
	//
	//   __action__
	//   __is_anonymous__
	//   __identity_id__
	//   __connection__
	//   __last_change_time__
	//
	// If there is the __anonymous_ids__ column, its values can contain at most one
	// non-NULL element, which is appended in the identity table if it does not
	// already exist.
	//
	// rows contains the rows to update or add if not already present. If a row
	// contains the $purge column with a value of true, the matching row is purged.
	// If the value is false, only the __execution__ column is updated to indicate
	// that the row should not be purged.
	MergeIdentities(ctx context.Context, columns []Column, rows []map[string]any) error

	// Query executes a query and returns the results as Rows. If withTotal is true,
	// it also returns an estimated total number of the records that would be
	// returned if the query did not include First and Limit clauses.
	//
	// Scan is called on the returned Rows with interface{} arguments. It copies
	// data directly into these arguments, rather than into the values they point
	// to.
	Query(ctx context.Context, query RowQuery, withTotal bool) (Rows, int, error)

	// ResolveIdentities resolves the identities.
	//
	// opID is an identifier that uniquely identifies a specific resolve identities
	// operation; if the method is called again passing the same identifier, whether
	// the operation ended successfully or with a *meergo.OperationError error, that
	// result is returned again.
	//
	// identifiers are the columns corresponding to the Identity Resolution
	// identifiers, ordered by priority.
	//
	// userColumns holds the columns of the user schema, without the meta
	// properties.
	//
	// userPrimarySources is a mapping between user column names (for which a
	// primary source connection have been set) and IDs of primary source
	// connections.
	//
	// This method, once called, can then return in four distinct cases:
	//
	// (1) the operation was successful and no error was returned;
	//
	// (2) the context was cancelled;
	//
	// (3) the operation ended with an error of type *meergo.OperationError, and this
	// means that even if the method is called again with the same ID, this error is
	// still returned;
	//
	// (4) the operation ended with an unexpected and unknown error, and it is
	// therefore up to the caller to try calling this method again by providing the
	// same ID.
	ResolveIdentities(ctx context.Context, opID string, identifiers, userColumns []Column, userPrimarySources map[string]int) error

	// Repair repairs the database objects on the data warehouse needed by Meergo.
	// It also takes care of correcting other inconsistent data (such as any tables
	// that store ongoing operations).
	// The given user schema will be used to repair the user tables.
	//
	// This method should only be called on warehouses that have already been
	// initialized, with the aim of correcting any extraordinary issues (such as
	// accidental table deletions) in an attempt to make Meergo functional again.
	Repair(ctx context.Context, userColumns []Column) error

	// Settings returns the data warehouse settings.
	Settings() []byte

	// Truncate truncates the specified table.
	Truncate(ctx context.Context, table string) error

	// UnsetIdentityColumns unsets values for the specified identity columns for the
	// given action. columns must not be empty. If the provided action does not
	// exist, it does nothing.
	UnsetIdentityColumns(ctx context.Context, action int, columns []Column) error
}

// RowQuery represents the query for the Query method.
type RowQuery struct {

	// Columns are the columns to return for each row.
	// Always contains at least one column.
	Columns []Column

	// Table is the table from which the records are read.
	Table string

	// Joins.
	Joins []Join

	// Where, when not nil, filters the records to return.
	Where Expr

	// OrderBy, when provided, specifies the columns used to order the returned
	// rows.
	OrderBy []Column

	// OrderDesc, when true and OrderBy is provided, orders the returned rows in
	// descending order instead of ascending order.
	OrderDesc bool

	// First is the index of the first returned row and must be >= 0.
	First int

	// Limit controls how many rows should be returned and must be >= 0. If
	// 0, it means that there is no limit.
	Limit int
}

// JoinType represents a type of JOIN statement.
type JoinType int

const (
	InnerJoin JoinType = iota
	LeftJoin
	RightJoin
	FullJoin
)

// Join represents a JOIN statement in a query.
type Join struct {
	Type      JoinType
	Table     string
	Condition Expr
}

// NormalizeFunc is a function type representing the normalization function
// exposed by data warehouse drivers to normalize values returned by them.
type NormalizeFunc func(name string, typ types.Type, v any, nullable bool) (any, error)

// IsValidIdentifier reports whether name is a valid identifier.
// A valid identifier must:
//   - start with [A-Za-z_]
//   - subsequently contain only [A-Za-z0-9_]
func IsValidIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if !('a' <= c && c <= 'z' || c == '_' || 'A' <= c && c <= 'Z' || i > 0 && '0' <= c && c <= '9') {
			return false
		}
	}
	return true
}

// IsValidSchemaName reports whether name is a valid schema name.
func IsValidSchemaName(name string) bool {
	return IsValidIdentifier(name)
}

// OperationError represents an error that occurred in the data warehouse during
// an Identity Resolution or user schema update operation.
type OperationError struct{ err error }

// NewOperationError returns a new *OperationError.
func NewOperationError(err error) *OperationError {
	return &OperationError{err: err}
}

func (err OperationError) Error() string {
	return err.err.Error()
}

// ValidateInt validates an int value.
func ValidateInt(name string, t types.Type, n int) (any, error) {
	min, max := t.IntRange()
	if int64(n) < min || int64(n) > max {
		return nil, fmt.Errorf("data warehouse returned a value of %d for column %s which is not within the expected range of [%d, %d]", n, name, min, max)
	}
	return n, nil
}

// ValidateUint validates an uint value.
func ValidateUint(name string, t types.Type, n uint) (any, error) {
	min, max := t.UintRange()
	if uint64(n) < min || uint64(n) > max {
		return nil, fmt.Errorf("data warehouse returned a value of %d for column %s which is not within the expected range of [%d, %d]", n, name, min, max)
	}
	return n, nil
}

// ValidateFloat validates a float value.
func ValidateFloat(name string, t types.Type, n float64) (any, error) {
	if t.IsReal() && (math.IsNaN(n) || math.IsInf(n, 0)) {
		return nil, fmt.Errorf("data warehouse returned %f for column %s but its type does not allow it", n, name)
	}
	min, max := t.FloatRange()
	if n < min || n > max {
		return nil, fmt.Errorf("PostgreSQL returned a value of %f for column %s which is not within the expected range of [%f, %f]", n, name, min, max)
	}
	return n, nil
}

// ValidateDecimal validates a decimal value.
func ValidateDecimal(name string, t types.Type, n decimal.Decimal) (any, error) {
	min, max := t.DecimalRange()
	if n.Less(min) || n.Greater(max) {
		return nil, fmt.Errorf("data warehouse returned a value of %s for column %s which is not within the expected range of [%s, %s]", n, name, min, max)
	}
	return n, nil
}

// ValidateDecimalString validates a decimal value represented as a string.
func ValidateDecimalString(name string, t types.Type, s string) (any, error) {
	n, err := decimal.Parse(s, t.Precision(), t.Scale())
	if err != nil {
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s which is not a decimal value", s, name)
	}
	return ValidateDecimal(name, t, n)
}

// ValidateDateTime validates a datetime value.
func ValidateDateTime(name string, dt time.Time) (any, error) {
	dt = dt.UTC()
	if y := dt.Year(); y < 1 || y > 9999 {
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s, with year %d not in range [1, 9999]", dt.Format(time.RFC3339Nano), name, y)
	}
	return dt, nil
}

// ValidateDate validates a date value.
func ValidateDate(name string, d time.Time) (any, error) {
	d = d.UTC()
	if y := d.Year(); y < 1 || y > 9999 {
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s, with year %d not in range [1, 9999]", d.Format(time.RFC3339Nano), name, y)
	}
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC), nil
}

// ValidateTime validates a time value.
func ValidateTime(t time.Time) (any, error) {
	t = t.UTC()
	return time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC), nil
}

// ValidateTimeString validates a time value represented as a string.
func ValidateTimeString(name string, format string, s string) (any, error) {
	t, err := time.Parse(format, s)
	if err != nil {
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s which is not a time type", s, name)
	}
	t = t.UTC()
	return time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC), nil
}

// ValidateYear validates a year value.
func ValidateYear(name string, year int) (any, error) {
	if year < types.MinYear || year > types.MaxYear {
		return nil, fmt.Errorf("data warehouse returned a value of %d for column %s which is not a year type", year, name)
	}
	return year, nil
}

// ValidateYearString validates a year value represented as a string.
func ValidateYearString(name string, year string) (any, error) {
	y, err := strconv.Atoi(year)
	if err != nil || y < types.MinYear || y > types.MaxYear {
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s which is not a year type", year, name)
	}
	return y, nil
}

// ValidateUUID validates a uuid value.
func ValidateUUID(name string, s string) (any, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s which is not a time type", s, name)
	}
	return u.String(), nil
}

// ValidateJSON validates a json value.
func ValidateJSON(name string, v any) (any, error) {
	var data []byte
	switch v := v.(type) {
	case json.Value:
		data = v
	case []byte:
		data = v
	case string:
		data = []byte(v)
	case json.Marshaler:
		var err error
		data, err = v.MarshalJSON()
		if err != nil {
			data = nil
		}
	}
	if data == nil {
		return nil, fmt.Errorf("data warehouse returned a value for column %s which is not a json type", name)
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("data warehouse returned an invalid JSON for column %s", name)
	}
	return json.Value(data), nil
}

// ValidateInet validates an inet value.
func ValidateInet(name string, s string) (any, error) {
	ip, err := netip.ParseAddr(s)
	if err != nil {
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s which is not an inet type", s, name)
	}
	return ip.String(), nil
}

// ValidateText validates a text value.
func ValidateText(name string, t types.Type, s string) (any, error) {
	if !utf8.ValidString(s) {
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s, which contains invalid UTF-8 characters", errors.Abbreviate(s, 20), name)
	}
	if values := t.Values(); values != nil {
		if !slices.Contains(values, s) {
			return nil, fmt.Errorf("data warehouse returned a value of %q for column %s, which is not valid", s, name)
		}
		return s, nil
	}
	if rx := t.Regexp(); rx != nil {
		if !rx.MatchString(s) {
			return nil, fmt.Errorf("data warehouse returned a value of %q for column %s, which is not valid", s, name)
		}
		return s, nil
	}
	if max, ok := t.ByteLen(); ok && len(s) > max {
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s, which is longer than %d bytes", errors.Abbreviate(s, 20), name, max)
	}
	if max, ok := t.CharLen(); ok && utf8.RuneCountInString(s) > max {
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s, which is longer than %d characters", errors.Abbreviate(s, 20), name, max)
	}
	return s, nil
}

// WarehouseNonInitializableError indicates that the data warehouse is not
// initializable.
type WarehouseNonInitializableError struct {
	Err error
}

// NewWarehouseNonInitializableError returns a new
// WarehouseNonInitializableError error.
func NewWarehouseNonInitializableError(err error) error {
	return &WarehouseNonInitializableError{Err: err}
}

func (err *WarehouseNonInitializableError) Error() string {
	return fmt.Sprintf("data warehouse is not initializable: %s", err.Err)
}

// WarehouseSettingsError represents an error in the data warehouse settings.
type WarehouseSettingsError struct {
	Err error
}

func (e *WarehouseSettingsError) Error() string {
	return fmt.Sprintf("settings error: %s", e.Err)
}

// WarehouseSettingsErrorf returns a new WarehouseSettingsError error with a
// fmt.Errorf(format, a...) error.
func WarehouseSettingsErrorf(format string, a ...any) error {
	return &WarehouseSettingsError{Err: fmt.Errorf(format, a...)}
}
