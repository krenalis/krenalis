//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package warehouses

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/netip"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/open2b/chichi/apis/errors"
	"github.com/open2b/chichi/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AlterSchemaOperation represents an operation that alters the "users" (and the
// "users_identities") schema.
type AlterSchemaOperation struct {
	Operation OperationType
	Path      string     // For "Add", "Drop" and "Rename" operations.
	Type      types.Type // For "Add" operations.
	Nullable  bool       // For "Add" operations.
	NewPath   string     // For "Rename" operations.
}

// MergeTable represents a table in which rows will be merged.
type MergeTable struct {
	Name        string           // Name of the table
	Properties  []types.Property // Properties to merge
	PrimaryKeys []types.Property // Primary keys
}

// OperationType represents an operation to perform on the data warehouse to
// alter the "users" (and "users_identities") schema.
type OperationType int

const (
	OperationAddProperty OperationType = iota + 1
	OperationDropProperty
	OperationRenameProperty
)

func (op OperationType) String() string {
	switch op {
	case OperationAddProperty:
		return "AddProperty"
	case OperationDropProperty:
		return "DropProperty"
	case OperationRenameProperty:
		return "RenameProperty"
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

// Warehouse is the interface implemented by data warehouses.
//
// Methods return a *DataWarehouseError error if an error occurs with the data
// warehouse.
type Warehouse interface {

	// AlterSchema alters the users schema.
	//
	// usersColumns contains the columns of the "users" table to obtain (this
	// parameters is useful for obtaining type information and for creating views),
	// while operations is the set of operations to apply in order to migrate the
	// current columns to usersColumns.
	//
	// If one of the specified operations is not supported by the data warehouse,
	// for example if a type is not supported, this method returns a
	// warehouses.UnsupportedSchemaChangeErr error.
	//
	// If an error occurs with the data warehouse, it returns a
	// *warehouses.DataWarehouseError error.
	AlterSchema(ctx context.Context, usersColumns []Column, operations []AlterSchemaOperation) error

	// AlterSchemaQueries returns the queries of a schema altering operation.
	//
	// usersColumns contains the columns of the "users" table to obtain (this
	// parameters is useful for obtaining type information and for creating views),
	// while operations is the set of operations to apply in order to migrate the
	// current columns to usersColumns.
	//
	// If one of the specified operations is not supported by the data warehouse,
	// for example if a type is not supported, this method returns a
	// warehouses.UnsupportedSchemaChangeErr error.
	//
	// If an error occurs with the data warehouse, it returns a
	// *warehouses.DataWarehouseError error.
	AlterSchemaQueries(ctx context.Context, usersColumns []Column, operations []AlterSchemaOperation) ([]string, error)

	// Close closes the warehouse.
	// It will not allow any new queries to run, and it waits for the current ones
	// to finish.
	Close() error

	// DeleteConnectionIdentities deletes the identities of a connection.
	// If an error occurs with the data warehouse, it returns a *DataWarehouseError
	// error.
	DeleteConnectionIdentities(ctx context.Context, connection int) error

	// DestinationUsers returns the destination users of the action.
	// In particular, returns the external app identifiers of the destination users
	// of the action whose external matching property value matches with the given
	// property value. If it cannot be found, then the empty string and false are
	// returned.
	DestinationUsers(ctx context.Context, action int, propertyValue string) ([]string, error)

	// DuplicatedDestinationUsers retrieves duplicated destination users.
	// In particular, it returns the external app identifiers of two users on the
	// action which have the same value for the matching property, along with true.
	//
	// If there are no users on the action matching this condition, no external app
	// identifiers are returned and the returned boolean is false. If an error
	// occurs with the data warehouse, it returns a *DataWarehouseError error.
	DuplicatedDestinationUsers(ctx context.Context, action int) (string, string, bool, error)

	// DuplicatedUsers returns the GIDs of two duplicated users.
	// Two users are duplicated if they have the same value for the given property;
	// in that case, their GID is returned and 'true'.
	// If there are no users matching this condition, no GIDs are returned and the
	// returned boolean is false.
	// If an error occurs with the data warehouse, it returns a *DataWarehouseError
	// error.
	DuplicatedUsers(ctx context.Context, property string) (int, int, bool, error)

	// IdentitiesWriter returns an IdentitiesWriter.
	// An IdentitiesWriter can be used for writing user identities with the given
	// schema, relative to the connection, on the data warehouse. fromEvent
	// indicates if the user identities are imported from an event or not. ack is
	// the ack function (see the documentation of IdentitiesWriter for more details
	// about it).
	IdentitiesWriter(ctx context.Context, schema types.Type, connection int, fromEvent bool, ack IdentitiesAckFunc) IdentitiesWriter

	// Init initializes the data warehouse by creating the supporting tables.
	Init(ctx context.Context) error

	// Merge performs a table merge operation.
	// If handles row updates, inserts, and deletions. table specifies the target
	// table for the merge operation, rows contains the rows to insert or update in
	// the table, and deleted contains the key values of rows to delete, if they
	// exist.
	// rows or deleted can be empty but not both, and both may be changed by this
	// method.
	Merge(ctx context.Context, table MergeTable, rows []map[string]any, deleted map[string]any) error

	// Normalize normalizes a value v returned by the Query method.
	// In particular, Normalize handles the values obtained by the scan on the
	// rows returned by the Query method.
	Normalize(name string, typ types.Type, v any, nullable bool) (any, error)

	// Ping checks the connection to the data warehouse.
	// In particular, it checks whether the connection to the data warehouse is
	// active and, if necessary, establishes a new connection.
	Ping(ctx context.Context) error

	// RunWorkspaceIdentityResolution runs the Workspace Identity Resolution.
	//
	// connections holds the identifiers of the connections of the workspace and may
	// be empty to indicate that no connections are present in the workspace.
	//
	// identifiers are the columns corresponding to the Workspace Identity
	// Resolution identifiers, ordered by priority.
	//
	// usersColumns holds the columns of the "users" schema, as the "users"
	// table on the data warehouse is rebuilt by this procedure.
	RunWorkspaceIdentityResolution(ctx context.Context, connections []int, identifiers, usersColumns []Column) error

	// SetDestinationUser sets the destination user for an action.
	//
	// If an error occurs with the data warehouse, it returns a *DataWarehouseError
	// error.
	SetDestinationUser(ctx context.Context, action int, externalUserID, externalProperty string) error

	// Settings returns the data warehouse settings.
	Settings() []byte

	// Query executes a query and returns the results as a Rows.
	// It also returns an estimated count of the records that would be returned if
	// First and Limit were not provided in the query.
	//
	// If an error occurs with the data warehouse, it returns a *DataWarehouseError
	// error.
	Query(ctx context.Context, query RowQuery) (Rows, int, error)
}

// RecordsQuery represents the query for the Query method.

type RowQuery struct {

	// Columns are the columns to return for each row.
	// Always contains at least one column.
	Columns []Column

	// Table is the table from which the records are read.
	Table string

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

// IdentitiesAckFunc is the function called when the writing of one or more user
// identities terminates. The parameter err represents the error that occurred
// while writing the identities, if any, and ids holds the identifiers of such
// identities.
type IdentitiesAckFunc func(err error, ids []string)

// IdentitiesWriter is the interface implemented by the data warehouse drivers
// to write user identities on the data warehouse.
type IdentitiesWriter interface {

	// Close closes the IdentitiesWriter, ensuring the completion of all pending or
	// ongoing write operations. In the event of a canceled context, it interrupts
	// ongoing writes, discards pending ones, and returns.
	//
	// In case an error occurs with the data warehouse, a DataWarehouseError error
	// is returned.
	//
	// If the IdentitiesWriter is already closed, it does nothing and returns
	// immediately.
	Close(ctx context.Context) error

	// Write writes a user identity. Typically, Write returns immediately, deferring
	// the actual write operation to a later time. If it returns false, no further
	// Write operations can be performed, and a call to Close will return an error.
	//
	// The values of the Identity Properties must comply with the schema of the
	// table of the identities in the data warehouse (note that "internal columns",
	// starting and ending with "__", shall not be included in Properties since they
	// are not included in the schema).
	//
	// If the user identity is successfully written, the ack function is invoked
	// with a nil error and the record's ID as arguments. If writing the record
	// fails, the ack function is invoked with a non-nil error and the user
	// identity's ID as arguments. The ack function is invoked even if Write returns
	// false.
	//
	// It panics if called on a closed writer.
	Write(ctx context.Context, identity Identity) bool
}

// Identity is a user identity to be written on the data warehouse.
type Identity struct {
	ID                string
	Properties        map[string]any
	AnonymousID       string    // empty string if not present.
	DisplayedProperty string    // cannot be longer than 40 runes.
	LastChangeTime    time.Time // in UTC.
}

// Rows is the result of a database query. Its cursor starts before the first
// row of the result set. Use Next to advance from row to row.
type Rows interface {
	Close() error
	Err() error
	Next() bool
	Scan(dest ...any) error
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
