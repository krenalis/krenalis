//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package warehouses

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/netip"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/datastore/expr"
	"chichi/apis/postgres"
	"chichi/connector/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// MergeTable represents a table in which rows will be merged.
type MergeTable struct {
	Name        string           // Name of the table
	Columns     []types.Property // Columns to merge
	PrimaryKeys []types.Property // Primary keys
}

// Warehouse is the interface implemented by data warehouses.
//
// Methods return a *DataWarehouseError error if an error occurs with the data
// warehouse.
type Warehouse interface {

	// Close closes the warehouse. It will not allow any new queries to run, and it
	// waits for the current ones to finish.
	Close() error

	// DestinationUser returns the external ID of the destination user of the
	// action that matches with the corresponding property. If it cannot be
	// found, then the empty string and false are returned.
	DestinationUser(ctx context.Context, action int, property string) (string, bool, error)

	// IdentitiesWriter returns an IdentitiesWriter for writing user identities,
	// relative to the action, on the data warehouse.
	// fromEvent indicates if the user identities are imported from an event or not.
	// ack is the ack function; see the documentation of IdentitiesWriter for more
	// details about it.
	IdentitiesWriter(ctx context.Context, action int, fromEvent bool, ack IdentitiesAckFunc) IdentitiesWriter

	// Init initializes the data warehouse by creating the supporting tables.
	Init(ctx context.Context) error

	// Merge performs a table merge operation, handling row updates, inserts, and
	// deletions. table specifies the target table for the merge operation, rows
	// contains the rows to insert or update in the table, and deleted contains the
	// key values of rows to delete, if they exist.
	// rows or deleted can be empty but not both.
	Merge(ctx context.Context, table MergeTable, rows [][]any, deleted []any) error

	// Ping checks whether the connection to the data warehouse is active and, if
	// necessary, establishes a new connection.
	Ping(ctx context.Context) error

	// SetDestinationUser sets the destination user relative to the action, with
	// the given external user ID and external property.
	SetDestinationUser(ctx context.Context, action int, externalUserID, externalProperty string) error

	// Settings returns the data warehouse settings.
	Settings() []byte

	// Tables returns the tables of the data warehouse.
	// It returns only the tables 'users', 'users_identities', 'groups',
	// 'groups_identities' and 'events'.
	Tables(ctx context.Context) ([]*Table, error)

	// QueryRow executes a query that should return at most one row.
	QueryRow(ctx context.Context, query string, args ...any) Row

	// ResolveSyncUsers resolves and sync the users.
	// actions holds the identifiers of the actions of the workspace and must always
	// contain at least one action; identifiers are the columns of the
	// 'users_identities' table which are identifiers, ordered by priority;
	// usersColumns are the columns of the 'users' table which will be populated
	// during the users synchronization.
	ResolveSyncUsers(ctx context.Context, actions []int, identifiersColumns, usersColumns []types.Property) error

	// Records returns an iterator over the results of the query and an estimated
	// count of the records that would be returned if First and Limit were not
	// provided in the query.
	//
	// If an error occurs with the data warehouse, it returns a *DataWarehouseError
	// error. If the schema specified in the query is not conform to the schema of
	// the table in the data warehouse, it returns a *SchemaError error.
	//
	// As a simplification, it is currently assumed that the table schema does not
	// change in the data warehouse during the execution of this method.
	Records(ctx context.Context, query RecordsQuery) (Records, int, error)
}

// RecordsQuery represents the query for the Records method.

type RecordsQuery struct {

	// ID is the property to return for each record in the Record.ID field.
	ID types.Property

	// Properties are the properties to return for each record in the
	// Record.Properties field.
	Properties []types.Path

	// Table is the table from which the records are read.
	Table string

	// Where, when not nil, filters the records to return.
	Where expr.Expr

	// OrderBy, when provided, is the property for which the returned records
	// are ordered.
	OrderBy types.Property

	// OrderDesc, when true and OrderBy is provided, orders the returned records
	// in descending order instead of ascending order.
	OrderDesc bool

	// Schema contains the types of the properties in Properties and Where.
	Schema types.Type

	// First is the index of the first returned record and must be >= 0.
	First int

	// Limit controls how many records should be returned and must be >= 0. If
	// 0, it means that there is no limit.
	Limit int
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
	ID          string // external ID.
	Properties  map[string]any
	AnonymousID string    // empty string if not present.
	Timestamp   time.Time // in UTC.
}

// Records is the iterator interface used to iterate over the records read from
// a data warehouse.
type Records interface {

	// Close closes the iterator. It is automatically called by the For method
	// before returning. Close is idempotent and does not impact the result of Err.
	Close() error

	// Err returns any error encountered during iteration, excluding errors returned
	// by the yield function, which may have occurred after an explicit or implicit
	// Close.
	Err() error

	// For calls the yield function for each record (r) in the sequence. If yield
	// returns an error, For stops and returns the error. After For completes, it
	// is also necessary to check the result of Err for any potential errors.
	For(yield func(Record) error) error
}

// Record represents a record.
type Record struct {
	ID         int            // Identifier.
	Properties map[string]any // Properties.
	// Err reports an error that occurred while reading the record.
	// If Err is not nil, only the ID field is significant.
	Err error
}

// Table represents a table.
type Table struct {
	Name    string
	Columns []types.Property
}

// Row returns a single row as a result of calling QueryRow.
type Row struct {
	Row   *postgres.Row
	Error error
}

func (row Row) Scan(dest ...any) error {
	if row.Error != nil {
		return row.Error
	}
	err := row.Row.Scan(dest...)
	if err != nil {
		if err == sql.ErrNoRows {
			return err
		}
		return Error(err)
	}
	return nil
}

// Rows represents the result of a query. Its methods, on error, return a
// *DataWarehouseError error.
type Rows struct {
	Rows *postgres.Rows
}

func (rows Rows) Close() {
	rows.Rows.Close()
}

func (rows Rows) Err() error {
	err := rows.Rows.Err()
	if err != nil {
		return Error(err)
	}
	return nil
}

func (rows Rows) Next() bool {
	return rows.Rows.Next()
}

func (rows Rows) Scan(dest ...any) error {
	err := rows.Rows.Scan(dest...)
	if err != nil {
		return Error(err)
	}
	return nil
}

// Result implements the sql.Result interface but on error it returns a
// *DataWarehouseError error.
type Result struct {
	Result *postgres.Result
}

func (r Result) RowsAffected() (int64, error) {
	n := r.Result.RowsAffected()
	return n, nil
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
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s, which contains invalid UTF-8 characters", abbreviate(s, 20), name)
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
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s, which is longer than %d bytes", abbreviate(s, 20), name, max)
	}
	if max, ok := t.CharLen(); ok && utf8.RuneCountInString(s) > max {
		return nil, fmt.Errorf("data warehouse returned a value of %q for column %s, which is longer than %d characters", abbreviate(s, 20), name, max)
	}
	return s, nil
}

// abbreviate abbreviates s to almost n runes. If s is longer than n runes,
// the abbreviated string terminates with "...".
func abbreviate(s string, n int) string {
	const spaces = " \n\r\t\f" // https://infra.spec.whatwg.org/#ascii-whitespace
	s = strings.TrimRight(s, spaces)
	if len(s) <= n {
		return s
	}
	if n < 3 {
		return ""
	}
	p := 0
	n2 := 0
	for i := range s {
		switch p {
		case n - 2:
			n2 = i
		case n:
			break
		}
		p++
	}
	if p < n {
		return s
	}
	if p = strings.LastIndexAny(s[:n2], spaces); p > 0 {
		s = strings.TrimRight(s[:p], spaces)
	} else {
		s = ""
	}
	if l := len(s) - 1; l >= 0 && (s[l] == '.' || s[l] == ',') {
		s = s[:l]
	}
	return s + "..."
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

type SchemaError struct {
	Msg string
}

func (err *SchemaError) Error() string {
	return err.Msg
}

// CheckConformity checks whether the schema t1 conforms to the new schema t2
// and returns a *SchemaError error if it does not conform.
//
// The conformity check must take into account:
//
// - the possibility that there exists a value of one type that is also a valid
// value for another type, deferring the check to runtime with actual values;
// otherwise, if this can never occur, the two types can be considered
// non-conform.
//
// - the real user use case of modifying a column of a certain type in the
// database
//
// - the impact of changing the type on the operation of the 'where' clause
// (does it return errors? does it behave unexpectedly?)
func CheckConformity(name string, t1, t2 types.Type) error {
	if t1.EqualTo(t2) {
		return nil
	}
	pt1 := t1.Kind()
	pt2 := t2.Kind()
	if pt1 != pt2 {
		return &SchemaError{Msg: fmt.Sprintf("type of the %q property has changed from %s to %s", name, t1, t2)}
	}
	switch pt1 {
	case types.ArrayKind:
		return CheckConformity(name, t1.Elem(), t2.Elem())
	case types.ObjectKind:
		for _, p1 := range t1.Properties() {
			path := p1.Name
			if name != "" {
				path = name + "." + path
			}
			p2, ok := t2.Property(p1.Name)
			if !ok {
				return &SchemaError{Msg: fmt.Sprintf(`"%s" property no longer exists`, path)}
			}
			err := CheckConformity(path, p1.Type, p2.Type)
			if err != nil {
				return err
			}
		}
	case types.MapKind:
		return CheckConformity(name, t1.Elem(), t2.Elem())
	}
	return nil
}
