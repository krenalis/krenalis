//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package warehouses

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/netip"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/apis/errors"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AlterSchemaOperation represents an operation that alters the user schema.
//
// Every column is always nullable.
type AlterSchemaOperation struct {
	Operation OperationType
	Column    string     // For "Add", "Drop" and "Rename" operations.
	Type      types.Type // For "Add" operations. Object properties are expanded into single "Add" operations, so Type can never have kind Object.
	NewColumn string     // For "Rename" operations.
}

// OperationType represents an operation to perform on the data warehouse to
// alter the "users" (and "_user_identities") schema.
type OperationType int

const (
	OperationAddColumn OperationType = iota + 1
	OperationDropColumn
	OperationRenameColumn
)

func (op OperationType) String() string {
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

func (op OperationType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + op.String() + `"`), nil
}

// UnsupportedAlterSchemaErr is an error indicating that a schema alter
// operation is not supported by a data warehouse.
type UnsupportedAlterSchemaErr string

func (e UnsupportedAlterSchemaErr) Error() string {
	return fmt.Sprintf("unsupported alter schema operation: %s", string(e))
}

var (
	ErrAlterSchemaInProgress        = errors.New("alter schema currently in progress on the data warehouse")
	ErrIdentityResolutionInProgress = errors.New("the Identity Resolution is currently in progress on the data warehouse")
)

// Warehouse is the interface implemented by data warehouses.
//
// Methods return a *DataWarehouseError error if an error occurs with the data
// warehouse.
type Warehouse interface {

	// AlterSchema alters the user schema.
	//
	// userColumns contains the columns of the "users" table to obtain (this
	// parameters is useful for obtaining type information and for creating views),
	// while operations is the set of operations to apply in order to migrate the
	// current columns to userColumns.
	//
	// If one of the specified operations is not supported by the data warehouse,
	// for example if a type is not supported, this method returns a
	// warehouses.UnsupportedSchemaChangeErr error.
	//
	// If another alter schema operation is in progress on the data warehouse,
	// returns an ErrAlterSchemaInProgress error.
	//
	// If an Identity Resolution is in progress, returns an
	// ErrIdentityResolutionInProgress error.
	//
	// If an error occurs with the data warehouse, it returns a *DataWarehouseError
	// error.
	//
	// If an error occurs with the data warehouse, it returns a
	// *warehouses.DataWarehouseError error.
	AlterSchema(ctx context.Context, userColumns []Column, operations []AlterSchemaOperation) error

	// AlterSchemaQueries returns the queries of a schema altering operation.
	//
	// userColumns contains the columns of the "users" table to obtain (this
	// parameters is useful for obtaining type information and for creating views),
	// while operations is the set of operations to apply in order to migrate the
	// current columns to userColumns.
	//
	// If one of the specified operations is not supported by the data
	// warehouse, for example if a type is not supported, this method returns a
	// warehouses.UnsupportedSchemaChangeErr error.
	//
	// If an error occurs with the data warehouse, it returns a
	// *warehouses.DataWarehouseError error.
	AlterSchemaQueries(ctx context.Context, userColumns []Column, operations []AlterSchemaOperation) ([]string, error)

	// Close closes the data warehouse. When Close is called, no other calls to
	// data warehouse's methods are in progress and no more will be made.
	Close() error

	// Delete deletes rows from the specified table that match the provided where
	// expression. Returns an error if the expression is nil.
	//
	// If an error occurs with the data warehouse, it returns a *DataWarehouseError
	// error.
	Delete(ctx context.Context, table string, where Expr) error

	// IdentityResolutionExecution returns information about the execution of the
	// Identity Resolution.
	//
	// - if the procedure has been started and completed, returns its start time and
	//   end time;
	// - if it is in progress, returns its start time and nil for the end time;
	// - if no Identity Resolution has ever been executed, returns nil and nil.
	//
	// If an error occurs with the data warehouse, it returns a *DataWarehouseError
	// error.
	IdentityResolutionExecution(ctx context.Context) (startTime, endTime *time.Time, err error)

	// Init initializes the data warehouse by creating the supporting tables.
	Init(ctx context.Context) error

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

	// Normalize normalizes a value v returned by the Query method.
	// In particular, Normalize handles the values obtained by the scan on the
	// rows returned by the Query method.
	Normalize(name string, typ types.Type, v any, nullable bool) (any, error)

	// Ping checks the connection to the data warehouse.
	// In particular, it checks whether the connection to the data warehouse is
	// active and, if necessary, establishes a new connection.
	Ping(ctx context.Context) error

	// Query executes a query and returns the results as Rows. If withCount is true,
	// it also returns an estimated total count of the records that would be
	// returned if the query did not include First and Limit clauses.
	//
	// If an error occurs with the data warehouse, it returns a *DataWarehouseError
	// error.
	Query(ctx context.Context, query RowQuery, withCount bool) (Rows, int, error)

	// RunIdentityResolution runs the Identity Resolution.
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
	// If an Identity Resolution is already in execution, returns an
	// ErrIdentityResolutionInProgress error.
	//
	// If an alter schema operation is in progress on the data warehouse, returns a
	// ErrAlterSchemaInProgress error.
	//
	// If an error occurs with the data warehouse, it returns a *DataWarehouseError
	// error.
	RunIdentityResolution(ctx context.Context, identifiers, userColumns []Column, userPrimarySources map[string]int) error

	// Settings returns the data warehouse settings.
	Settings() []byte

	// Truncate truncates the specified table.
	//
	// If an error occurs with the data warehouse, it returns a *DataWarehouseError
	// error.
	Truncate(ctx context.Context, table string) error
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

	// OrderBy, when provided, is the column for which the returned rows are
	// ordered.
	OrderBy Column

	// OrderDesc, when true and OrderBy is provided, orders the returned rows in
	// descending order instead of ascending order.
	OrderDesc bool

	// First is the index of the first returned row and must be >= 0.
	First int

	// Limit controls how many rows should be returned and must be >= 0. If
	// 0, it means that there is no limit.
	Limit int
}

// Column represents a table column.
type Column struct {
	Name     string
	Type     types.Type
	Nullable bool
}

// JoinType represents a type of JOIN statement.
type JoinType int

const (
	Inner JoinType = iota
	Left
	Right
	Full
)

// Join represents a JOIN statement in a query.
type Join struct {
	Type      JoinType
	Table     string
	Condition Expr
}

// Rows is the result of a database query. Its cursor starts before the first
// row of the result set. Use Next to advance from row to row.
type Rows interface {
	Close() error
	Err() error
	Next() bool
	Scan(dest ...any) error
}

// Table represents a table.
type Table struct {
	Name    string
	Columns []Column
	Keys    []string
}

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

// ValidateInt validates an Int value.
func ValidateInt(name string, t types.Type, n int) (any, error) {
	min, max := t.IntRange()
	if int64(n) < min || int64(n) > max {
		return nil, fmt.Errorf("data warehouse returned a value of %d for column %s which is not within the expected range of [%d, %d]", n, name, min, max)
	}
	return n, nil
}

// ValidateUint validates an Uint value.
func ValidateUint(name string, t types.Type, n uint) (any, error) {
	min, max := t.UintRange()
	if uint64(n) < min || uint64(n) > max {
		return nil, fmt.Errorf("data warehouse returned a value of %d for column %s which is not within the expected range of [%d, %d]", n, name, min, max)
	}
	return n, nil
}

// ValidateFloat validates a Float value.
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

// ValidateDecimal validates a Decimal value.
func ValidateDecimal(name string, t types.Type, n decimal.Decimal) (any, error) {
	min, max := t.DecimalRange()
	if n.LessThan(min) || n.GreaterThan(max) {
		return nil, fmt.Errorf("data warehouse returned a value of %s for column %s which is not within the expected range of [%s, %s]", n, name, min, max)
	}
	return n, nil
}

// ValidateDecimalString validates a Decimal value represented as a string.
func ValidateDecimalString(name string, t types.Type, s string) (any, error) {
	n, err := decimal.NewFromString(s)
	if err != nil {
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s which is not a Decimal value", s, name)
	}
	min, max := t.DecimalRange()
	if n.LessThan(min) || n.GreaterThan(max) {
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s which is not within the expected range of [%s, %s]", s, name, min, max)
	}
	return n, nil
}

// ValidateDateTime validates a DateTime value.
func ValidateDateTime(name string, dt time.Time) (any, error) {
	dt = dt.UTC()
	if y := dt.Year(); y < 1 || y > 9999 {
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s, with year %d not in range [1, 9999]", dt.Format(time.RFC3339Nano), name, y)
	}
	return dt, nil
}

// ValidateDate validates a Date value.
func ValidateDate(name string, d time.Time) (any, error) {
	d = d.UTC()
	if y := d.Year(); y < 1 || y > 9999 {
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s, with year %d not in range [1, 9999]", d.Format(time.RFC3339Nano), name, y)
	}
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC), nil
}

// ValidateTime validates a Time value.
func ValidateTime(t time.Time) (any, error) {
	t = t.UTC()
	return time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC), nil
}

// ValidateTimeString validates a Time value represented as a string.
func ValidateTimeString(name string, format string, s string) (any, error) {
	t, err := time.Parse(format, s)
	if err != nil {
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s which is not a Time type", s, name)
	}
	t = t.UTC()
	return time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC), nil
}

// ValidateUUID validates an UUID value.
func ValidateUUID(name string, s string) (any, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s which is not a Time type", s, name)
	}
	return u.String(), nil
}

// ValidateJSON validates a JSON value.
func ValidateJSON(name string, v any) (any, error) {
	if !isValidJSON(v) {
		return nil, fmt.Errorf("data warehouse returned an invalid JSON value for column %s", name)
	}
	return v, nil
}

// ValidateJSONRaw validates a JSON value represented as a json.RawMessage.
func ValidateJSONRaw(name string, b json.RawMessage) (any, error) {
	if !json.Valid(b) {
		return nil, fmt.Errorf("data warehouse returned an invalid JSON value for column %s", name)
	}
	b = bytes.TrimSpace(b)
	return b, nil
}

// ValidateInet validates an Inet value.
func ValidateInet(name string, s string) (any, error) {
	ip, err := netip.ParseAddr(s)
	if err != nil {
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s which is not an Inet type", s, name)
	}
	return ip.String(), nil
}

// ValidateText validates a Text value.
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

// isValidJSON reports whether src is a valid JSON value.
func isValidJSON(src any) bool {
	switch src := src.(type) {
	case string:
		return utf8.ValidString(src)
	case bool:
		return true
	case float64:
		return !math.IsNaN(src) && !math.IsInf(src, 0)
	case []any:
		for _, v := range src {
			if v != nil {
				if ok := isValidJSON(v); !ok {
					return false
				}
			}
		}
		return true
	case map[string]any:
		for _, v := range src {
			if v != nil {
				if ok := isValidJSON(v); !ok {
					return false
				}
			}
		}
		return true
	case json.Number:
		return src != "" && (src[0] == '-' || src[0] >= '0' && src[0] <= '9') && json.Valid([]byte(src))
	case json.RawMessage:
		return json.Valid(src)
	}
	return false
}
