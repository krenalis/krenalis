//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
)

var (
	// Ensure that *Conn, *DB and *Tx implement Connection.
	_ Connection = (*Conn)(nil)
	_ Connection = (*DB)(nil)
	_ Connection = (*Tx)(nil)
)

// ErrTxClosed is the error returned by the Tx.Rollback method if the
// transaction has already been closed.
var ErrTxClosed = errors.New("transaction has already been closed")

// TxCommitRollbackError is returned when a commit results in a rollback.
type TxCommitRollbackError struct {
	Err error
}

func (err TxCommitRollbackError) Error() string {
	return fmt.Sprintf("transaction rolled back during commit: %s", err.Err)
}

// Result is the result returned by a call to the Exec method.
type Result struct {
	ct pgconn.CommandTag
}

// RowsAffected returns the number of rows affected.
func (res *Result) RowsAffected() int64 {
	return res.ct.RowsAffected()
}

type Connection interface {

	// Exec executes a query that does not return any rows. The args correspond to
	// placeholder parameters in the query, represented as $1, $2, etc.
	Exec(ctx context.Context, query string, args ...any) (*Result, error)

	// Query executes a SQL query that returns rows, such as a SELECT statement.
	// The args are the values for any placeholders in the query, like $1, $2, etc.
	Query(ctx context.Context, query string, args ...any) (*Rows, error)

	// QueryExists checks whether at least one row matches the query, similar to
	// Query, but returns a boolean instead of the result set.
	// The query should not include return columns, e.g., "SELECT FROM table".
	QueryExists(ctx context.Context, query string, args ...any) (bool, error)

	// QueryScan is like [Query], but the last element of args must be a function
	// that scans the returned rows. The function must have the following signature:
	//
	//   - func(*Rows) error
	//
	// If the scan function returns an error, QueryScan will return that error.
	QueryScan(ctx context.Context, query string, args ...any) error

	// QueryRow executes a query that is expected to return at most one row. It
	// always returns a non-nil *Row. Errors are deferred until the Scan method of
	// the Row is called.
	//
	// If the query returns no rows, *Row.Scan will return sql.ErrNoRows. Otherwise,
	// *Row.Scan will scan the first row and discard any additional rows.
	QueryRow(ctx context.Context, query string, args ...any) *Row
}

// DB represents a pool of zero or more underlying connections.
// It is safe for concurrent use by multiple goroutines.
type DB struct {
	db *pgxpool.Pool
}

// Options defines the configuration for connecting to a database, used by Open.
type Options struct {
	Host           string // Database server hostname or IP.
	Port           int    // Port number of the database server.
	Username       string // Username for authentication.
	Password       string // Password for authentication.
	Database       string // Name of the database to connect to.
	Schema         string // Default schema to use.
	MaxConnections int32  // Maximum number of connections (if > 0).
}

// Open opens a PostgreSQL database. It validates its arguments without creating
// a connection to the database.
func Open(opts *Options) (*DB, error) {
	if opts == nil {
		opts = &Options{}
	}
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(opts.Username, opts.Password),
		Host:   net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port)),
		Path:   "/" + url.PathEscape(opts.Database),
	}
	if opts.Schema != "" {
		u.RawQuery = "search_path=" + url.QueryEscape(opts.Schema)
	}
	config, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		return nil, err
	}
	if opts.MaxConnections > 0 {
		config.MaxConns = opts.MaxConnections
	}
	conn, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	return &DB{db: conn}, nil
}

// Begin begins a new transaction with the READ COMMITTED isolation level.
// The provided context only affects the Begin method and does not propagate
// to the entire transaction, unlike the behavior in the standard sql package.
func (db *DB) Begin(ctx context.Context) (*Tx, error) {
	tx, err := db.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	return &Tx{tx, false, nil}, nil
}

// Close the database.
func (db *DB) Close() {
	db.db.Close()
}

// Conn acquires a connection from the connection pool and returns it.
// The caller must invoke the Close method to release the connection when it is
// no longer needed.
func (db *DB) Conn(ctx context.Context) (*Conn, error) {
	conn, err := db.db.Acquire(ctx)
	if err != nil {
		return nil, convertErr(err)
	}
	return &Conn{conn: conn}, nil
}

// Exec implements the [Connection.Exec] method.
func (db *DB) Exec(ctx context.Context, query string, args ...any) (*Result, error) {
	ct, err := db.db.Exec(ctx, query, args...)
	if err != nil {
		return nil, convertErr(err)
	}
	return &Result{ct}, nil
}

// Ping executes a simple SQL statement to check database connectivity.
// If the statement runs successfully, the Ping is considered successful;
// otherwise, an error is returned.
func (db *DB) Ping(ctx context.Context) error {
	return db.db.Ping(ctx)
}

// Query implements the [Connection.Query] method.
func (db *DB) Query(ctx context.Context, query string, args ...any) (*Rows, error) {
	rows, err := db.db.Query(ctx, query, args...)
	if err != nil {
		return nil, convertErr(err)
	}
	return &Rows{rows}, nil
}

// QueryExists implements the [Connection.QueryExists] method.
func (db *DB) QueryExists(ctx context.Context, query string, args ...any) (bool, error) {
	err := db.db.QueryRow(ctx, query, args...).Scan()
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, convertErr(err)
	}
	return true, nil
}

// QueryRow implements the [Connection.QueryRow] method.
func (db *DB) QueryRow(ctx context.Context, query string, args ...any) *Row {
	row := db.db.QueryRow(ctx, query, args...)
	return &Row{Row: row}
}

// QueryScan implements the [Connection.QueryScan] method.
func (db *DB) QueryScan(ctx context.Context, query string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("missing scan function")
	}
	args, arg := args[:len(args)-1], args[len(args)-1]
	scan, ok := arg.(func(*Rows) error)
	if !ok {
		return fmt.Errorf("cannot use a %T value as scan function", arg)
	}
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return convertErr(err)
	}
	defer rows.Close()
	err = scan(rows)
	if err != nil {
		return convertErr(err)
	}
	if err = rows.Close(); err != nil {
		return convertErr(err)
	}
	return nil
}

// Transaction begins a new transaction and executes the provided function, f.
// If f completes without errors, the transaction is committed.
// If f returns an error, the transaction is rolled back and the error is
// returned.
// If f panics, the transaction is rolled back and the panic is propagated.
func (db *DB) Transaction(ctx context.Context, f func(tx *Tx) error) error {
	pqTx, err := db.db.Begin(ctx)
	if err != nil {
		return err
	}
	return func() error {
		defer func() {
			if err := recover(); err != nil {
				_ = pqTx.Rollback(ctx)
				panic(err)
			}
		}()
		tx := &Tx{pqTx, true, nil}
		err := f(tx)
		if err != nil {
			_ = pqTx.Rollback(ctx)
			return convertErr(err)
		}
		err = pqTx.Commit(ctx)
		if err != nil && err != sql.ErrTxDone {
			return convertErr(err)
		}
		if tx.ack != nil {
			<-tx.ack
		}
		return nil
	}()
}

// Conn represents a connection from the connection pool.
type Conn struct {
	conn *pgxpool.Conn
}

// Close releases the connection.
func (c *Conn) Close() error {
	c.conn.Release()
	return nil
}

// Exec implements the [Connection.Exec] method.
func (c *Conn) Exec(ctx context.Context, query string, args ...any) (*Result, error) {
	ct, err := c.conn.Exec(ctx, query, args...)
	if err != nil {
		return nil, convertErr(err)
	}
	return &Result{ct: ct}, nil
}

// Query implements the [Connection.Query] method.
func (c *Conn) Query(ctx context.Context, query string, args ...any) (*Rows, error) {
	rows, err := c.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, convertErr(err)
	}
	return &Rows{rows}, nil
}

// QueryExists implements the [Connection.QueryExists] method.
func (c *Conn) QueryExists(ctx context.Context, query string, args ...any) (bool, error) {
	err := c.conn.QueryRow(ctx, query, args...).Scan()
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, convertErr(err)
	}
	return true, nil
}

// QueryScan implements the [Connection.QueryScan] method.
func (c *Conn) QueryScan(ctx context.Context, query string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("missing scan function")
	}
	args, arg := args[:len(args)-1], args[len(args)-1]
	scan, ok := arg.(func(*Rows) error)
	if !ok {
		return fmt.Errorf("cannot use a %T value as scan function", arg)
	}
	rows, err := c.Query(ctx, query, args...)
	if err != nil {
		return convertErr(err)
	}
	defer rows.Close()
	err = scan(rows)
	if err != nil {
		return convertErr(err)
	}
	if err = rows.Close(); err != nil {
		return convertErr(err)
	}
	return nil
}

// QueryRow implements the [Connection.QueryRow] method.
func (c *Conn) QueryRow(ctx context.Context, query string, args ...any) *Row {
	row := c.conn.QueryRow(ctx, query, args...)
	return &Row{Row: row}
}

// Underlying returns the underlying connection.
func (c *Conn) Underlying() *pgx.Conn {
	return c.conn.Conn()
}

// Tx represents a transaction.
type Tx struct {
	tx      pgx.Tx
	wrapped bool
	ack     <-chan struct{}
}

// Commit commits the transaction. It must not be called from within the
// function provided to the Transaction method.
//
// If no error is returned, the transaction has been committed successfully.
// If a TxRollbackError is returned, the commit resulted in a rollback.
// If any other error occurs, the connection is closed and the error is
// returned.
func (tx *Tx) Commit(ctx context.Context) error {
	if tx.wrapped {
		return errors.New("commit called in a wrapped transaction")
	}
	err := tx.tx.Commit(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrTxCommitRollback) {
			return TxCommitRollbackError{Err: err}
		}
		_ = tx.tx.Conn().Close(ctx)
		return err
	}
	return nil
}

// Exec implements the [Connection.Exec] method.
func (tx *Tx) Exec(ctx context.Context, query string, args ...any) (*Result, error) {
	ct, err := tx.tx.Exec(ctx, query, args...)
	if err != nil {
		return nil, convertErr(err)
	}
	return &Result{ct}, nil
}

// Query implements the [Connection.Query] method.
func (tx *Tx) Query(ctx context.Context, query string, args ...any) (*Rows, error) {
	rows, err := tx.tx.Query(ctx, query, args...)
	if err != nil {
		return nil, convertErr(err)
	}
	return &Rows{rows}, nil
}

// QueryExists implements the [Connection.QueryExists] method.
func (tx *Tx) QueryExists(ctx context.Context, query string, args ...any) (bool, error) {
	err := tx.tx.QueryRow(ctx, query, args...).Scan()
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, convertErr(err)
	}
	return true, nil
}

// QueryScan implements the [Connection.QueryScan] method.
func (tx *Tx) QueryScan(ctx context.Context, query string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("missing scan function")
	}
	args, arg := args[:len(args)-1], args[len(args)-1]
	scan, ok := arg.(func(*Rows) error)
	if !ok {
		return fmt.Errorf("cannot use a %T value as scan function", arg)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return convertErr(err)
	}
	defer rows.Close()
	err = scan(rows)
	if err != nil {
		return convertErr(err)
	}
	if err = rows.Close(); err != nil {
		return convertErr(err)
	}
	return nil
}

// QueryRow implements the [Connection.QueryRow] method.
func (tx *Tx) QueryRow(ctx context.Context, query string, args ...any) *Row {
	row := tx.tx.QueryRow(ctx, query, args...)
	return &Row{Row: row}
}

// Rollback aborts the transaction. It must not be called from within the
// function provided to the Transaction method.
//
// If the transaction is already closed, it returns ErrTxClosed.
// If any other error occurs, the connection is closed and the error is
// returned.
//
// It is safe to call Rollback in a defer statement to ensure that the
// transaction is properly closed, even if it has already been closed by a
// commit.
func (tx *Tx) Rollback(ctx context.Context) error {
	if tx.wrapped {
		return errors.New("called in a wrapped transaction")
	}
	err := tx.tx.Rollback(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrTxClosed) {
			return ErrTxClosed
		}
		_ = tx.tx.Conn().Close(ctx)
		return err
	}
	return nil
}

// ErrConstraintName returns the name of the constraint in err, if any.
func ErrConstraintName(err error) string {
	if err, ok := err.(*pgconn.PgError); ok {
		return err.ConstraintName
	}
	return ""
}

// IsForeignKeyViolation reports whether err is a foreign key violation error.
func IsForeignKeyViolation(err error) bool {
	if err, ok := err.(*pgconn.PgError); ok {
		return err.Code == "23503"
	}
	return false
}

// Quote escapes a value to safely insert it into a query.
func Quote(value any) string {
	if value == nil {
		return "NULL"
	}
	switch val := value.(type) {
	case bool:
		if val {
			return "TRUE"
		}
		return "FALSE"
	case int:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case float32:
		return strconv.FormatFloat(float64(val), 'G', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'G', -1, 64)
	case string:
		return quote(val)
	case time.Time:
		return "'" + val.Format("2006-01-02 15:04:05.999999") + "'"
	case []string:
		if len(val) == 1 {
			return `(` + quote(val[0]) + `)`
		}
		var values = make([]string, len(val))
		for i, v := range val {
			values[i] = quote(v)
		}
		return "(" + strings.Join(values, ",") + ")"
	case []int:
		if len(val) == 1 {
			return "(" + strconv.Itoa(val[0]) + ")"
		}
		var values = make([]string, len(val))
		for i, v := range val {
			values[i] = strconv.Itoa(v)
		}
		return "(" + strings.Join(values, ",") + ")"
	case []int64:
		if len(val) == 1 {
			return "(" + strconv.FormatInt(val[0], 10) + ")"
		}
		var values = make([]string, len(val))
		for i, v := range val {
			values[i] = strconv.FormatInt(v, 10)
		}
		return "(" + strings.Join(values, ",") + ")"
	case []time.Time:
		if len(val) == 1 {
			return "('" + val[0].Format(time.DateTime) + "')"
		}
		var values = make([]string, len(val))
		for i, v := range val {
			values[i] = "'" + v.Format(time.DateTime) + "'"
		}
		return "(" + strings.Join(values, ",") + ")"
	case []any:
		var values = make([]string, len(val))
		for i, v := range val {
			if v == nil {
				values[i] = "NULL"
			} else {
				switch v.(type) {
				case bool, int, int64, uint, uint64, float32, float64, string, time.Time:
				default:
					panic(fmt.Errorf("core/db: unsupported nested type '%T'", v))
				}
				values[i] = Quote(v)
			}
		}
		return "(" + strings.Join(values, ",") + ")"
	default:
		panic(fmt.Errorf("core/db: unsupported type '%T'", val))
	}
}

func quote(s string) string {
	return "'" + escape(s) + "'"
}

func escape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// Rows represents the result of a query. The cursor starts before the first row
// of the result set. Use [Rows.Next] to iterate through the rows.
type Rows struct {
	pgx.Rows
}

func (rs *Rows) Close() error {
	rs.Rows.Close()
	return nil
}

func (rs *Rows) Scan(dest ...any) error {
	return rs.Rows.Scan(dest...)
}

// Row represents the result of a [Connection.QueryRow] call that selects a
// single row.
type Row struct {
	pgx.Row
	closed bool
}

func (r *Row) Scan(dest ...any) error {
	if r.closed {
		return errors.New("already closed")
	}
	r.closed = true
	err := r.Row.Scan(dest...)
	return convertErr(err)
}

// convertErr converts a pgx error and returns it.
func convertErr(err error) error {
	if err == pgx.ErrNoRows {
		return sql.ErrNoRows
	}
	return err
}
