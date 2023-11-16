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
// Methods return a DataWarehouseError error if an error occurs with the data
// warehouse.
type Warehouse interface {

	// Close closes the warehouse. It will not allow any new queries to run, and it
	// waits for the current ones to finish.
	Close() error

	// DestinationUser returns the external ID of the destination user of the
	// action that matches with the corresponding property. If it cannot be
	// found, then the empty string and false are returned.
	DestinationUser(ctx context.Context, action int, property string) (string, bool, error)

	// Exec executes a query without returning any rows. args are the placeholders.
	Exec(ctx context.Context, query string, args ...any) (Result, error)

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

	// SetIdentity sets the identity id (which may have an anonymous ID) imported
	// from the action. fromEvents indicates if the identity has been imported from
	// an event or not.
	// timestamp is the timestamp that will be associated to the imported identity.
	SetIdentity(ctx context.Context, identity map[string]any, id string, anonID string, action int, fromEvent bool, timestamp time.Time) error

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

	// Select returns the rows from the given table that satisfies the where
	// condition with only the given columns, ordered by order if order is not the
	// zero Property, and in range [first,first+limit] with first >= 0 and
	// 0 < limit <= 1000.
	Select(ctx context.Context, table string, columns []types.Property, where expr.Expr, order types.Property, first, limit int) ([][]any, error)
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
// DataWarehouseError error.
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
// DataWarehouseError error.
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
