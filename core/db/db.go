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
	"io"
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

type RowsScanner interface {
	Next() bool
	Scan(...any) error
}

// Result is the result returned by a call to the Exec method.
type Result struct {
	ct pgconn.CommandTag
}

// RowsAffected returns the number of rows affected.
// TODO(Gianluca): ensure that this implementation has the same behavior of
// sql.Result.RowsAffected.
func (res *Result) RowsAffected() int64 {
	return res.ct.RowsAffected()
}

// Stmt represents a prepared statement in a transaction.
type Stmt struct {
	tx     *Tx
	closed bool
}

func (stmt *Stmt) Close() error {
	if stmt.closed {
		return nil
	}
	stmt.closed = true
	stmt.tx.preparedQuery = false
	return nil
}

const preparedStmtName = "meergo-postgres-prep-stmt"

func (stmt *Stmt) Exec(ctx context.Context, args ...any) (*Result, error) {
	if stmt.closed {
		return nil, errors.New("already closed")
	}
	return stmt.tx.Exec(ctx, preparedStmtName, args...)
}

func (stmt *Stmt) Query(ctx context.Context, args ...any) (*Rows, error) {
	if stmt.closed {
		return nil, errors.New("already closed")
	}
	return stmt.tx.Query(ctx, preparedStmtName, args...)
}

func (stmt *Stmt) QueryScan(ctx context.Context, args ...any) error {
	if stmt.closed {
		return errors.New("already closed")
	}
	return stmt.tx.QueryScan(ctx, preparedStmtName, args...)
}

func (stmt *Stmt) QueryRow(ctx context.Context, args ...any) *Row {
	if stmt.closed {
		return &Row{closed: true}
	}
	return stmt.tx.QueryRow(ctx, preparedStmtName, args...)
}

func (stmt *Stmt) QueryVoid(ctx context.Context, args ...any) error {
	if stmt.closed {
		return errors.New("already closed")
	}
	return stmt.tx.QueryVoid(ctx, preparedStmtName, args...)
}

type Connection interface {
	Exec(context.Context, string, ...any) (*Result, error)
	Query(context.Context, string, ...any) (*Rows, error)
	QueryScan(context.Context, string, ...any) error
	QueryRow(context.Context, string, ...any) *Row
	QueryVoid(context.Context, string, ...any) error
	Tables(context.Context, string) ([]string, error)
}

var (
	// Ensure that *Conn, *DB and *Tx implement Connection.
	_ Connection = (*Conn)(nil)
	_ Connection = (*DB)(nil)
	_ Connection = (*Tx)(nil)
)

type DB struct {
	db  *pgxpool.Pool
	log io.Writer
}

type Options struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Schema   string
	MaxConns int32
}

// Open opens a PostgreSQL database. It validates its arguments without creating
// a connection to the database.
// opts.MaxConns, when greater than zero, specifies the number of maximum
// connections.
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
	if opts.MaxConns > 0 {
		config.MaxConns = opts.MaxConns
	}
	conn, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	return &DB{db: conn}, nil
}

func (db *DB) Close() {
	db.db.Close()
}

func (db *DB) SetQueryLog(log io.Writer) {
	db.log = log
}

func (db *DB) Ping(ctx context.Context) error {
	return db.db.Ping(ctx)
}

func (db *DB) Exec(ctx context.Context, query string, args ...any) (*Result, error) {
	if db.log != nil {
		fmt.Fprint(db.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(db.log, "> args: %v\n", args)
		}
	}
	ct, err := db.db.Exec(ctx, query, args...)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = sql.ErrNoRows
		}
		return nil, err
	}
	return &Result{ct}, nil
}

func (db *DB) Query(ctx context.Context, query string, args ...any) (*Rows, error) {
	if db.log != nil {
		fmt.Fprint(db.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(db.log, "> args: %v\n", args)
		}
	}
	rows, err := db.db.Query(ctx, query, args...)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = sql.ErrNoRows
		}
		return nil, err
	}
	return &Rows{rows}, nil
}

func (db *DB) QueryRow(ctx context.Context, query string, args ...any) *Row {
	if db.log != nil {
		fmt.Fprint(db.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(db.log, "> args: %v\n", args)
		}
	}
	row := db.db.QueryRow(ctx, query, args...)
	return &Row{Row: row}
}

func (db *DB) QueryScan(ctx context.Context, query string, args ...any) error {
	// TODO(Gianluca): it's not clear which of the functions called here can, in
	// practice, return the 'pgx.ErrNoRows' error, as it's not documented; so,
	// to ensure that it always catch, it is checked everywhere.
	if db.log != nil {
		fmt.Fprint(db.log, query, "\n\n")
	}
	if len(args) == 0 {
		return fmt.Errorf("sql: missing scan function")
	}
	args, arg := args[:len(args)-1], args[len(args)-1]
	switch arg.(type) {
	case func(*Rows) error:
	case func(RowsScanner) error:
	default:
		return fmt.Errorf("sql: cannot use a %T value as scan function", arg)
	}
	if db.log != nil && len(args) > 0 {
		fmt.Fprintf(db.log, "> args: %v\n", args)
	}
	rows, err := db.db.Query(ctx, query, args...)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = sql.ErrNoRows
		}
		return err
	}
	defer rows.Close()
	switch scan := arg.(type) {
	case func(*Rows) error:
		err = scan(&Rows{rows})
	case func(RowsScanner) error:
		err = scan(&Rows{rows})
	}
	if err != nil {
		if err == pgx.ErrNoRows {
			err = sql.ErrNoRows
		}
		return err
	}
	rows.Close()
	err = rows.Err()
	if err == pgx.ErrNoRows {
		err = sql.ErrNoRows
	}
	return err
}

func (db *DB) QueryVoid(ctx context.Context, query string, args ...any) error {
	if db.log != nil {
		fmt.Fprint(db.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(db.log, "> args: %v\n", args)
		}
	}
	err := db.db.QueryRow(ctx, query, args...).Scan()
	if err == pgx.ErrNoRows {
		err = sql.ErrNoRows
	}
	return err
}

func (db *DB) Begin(ctx context.Context) (*Tx, error) {
	tx, err := db.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	return &Tx{tx, db.log, false, false, nil}, nil
}

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
		tx := &Tx{pqTx, db.log, true, false, nil}
		err := f(tx)
		if err != nil {
			_ = pqTx.Rollback(ctx)
			return err
		}
		err = pqTx.Commit(ctx)
		if err != nil && err != sql.ErrTxDone {
			return err
		}
		if tx.ack != nil {
			<-tx.ack
		}
		return nil
	}()
}

type Identifier []string

type CopyFromSource interface {
	Next() bool
	Values() ([]any, error)
	Err() error
}

func CopyFromRows(rows [][]any) CopyFromSource {
	return pgx.CopyFromRows(rows)
}

func (db *DB) CopyFrom(ctx context.Context, tableName Identifier, columnNames []string, rowSrc CopyFromSource) (int64, error) {
	return db.db.CopyFrom(ctx, pgx.Identifier(tableName), columnNames, pgx.CopyFromSource(rowSrc))
}

func (db *DB) Conn(ctx context.Context) (*Conn, error) {
	conn, err := db.db.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	return &Conn{
		conn: conn,
		log:  db.log,
	}, nil
}

func (db *DB) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	return db.db.Acquire(ctx)
}

func (db *DB) Tables(ctx context.Context, database string) ([]string, error) {
	// TODO(Gianluca): it's not clear which of the functions called here can, in
	// practice, return the 'pgx.ErrNoRows' error, as it's not documented; so,
	// to ensure that it always catch, it is checked everywhere.
	var stmt = "SHOW TABLES"
	if database != "" {
		stmt = "SHOW TABLES FROM `" + database + "`"
	}
	if db.log != nil {
		fmt.Fprint(db.log, stmt, "\n\n")
	}
	var rows, err = db.db.Query(ctx, stmt)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = sql.ErrNoRows
		}
		return nil, err
	}
	defer rows.Close()
	var tables = []string{}
	for rows.Next() {
		var table string
		err = rows.Scan(&table)
		if err != nil {
			if err == pgx.ErrNoRows {
				err = sql.ErrNoRows
			}
			return nil, err
		}
		tables = append(tables, table)
	}
	if err = rows.Err(); err != nil {
		if err == pgx.ErrNoRows {
			err = sql.ErrNoRows
		}
		return nil, err
	}
	return tables, nil
}

// UnderlyingPool returns the underlying pool.
func (db *DB) UnderlyingPool() *pgxpool.Pool {
	return db.db
}

type Conn struct {
	conn *pgxpool.Conn
	log  io.Writer
}

func (c *Conn) Exec(ctx context.Context, query string, args ...any) (*Result, error) {
	if c.log != nil {
		fmt.Fprint(c.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(c.log, "> args: %v\n", args)
		}
	}
	ct, err := c.conn.Exec(ctx, query, args...)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = sql.ErrNoRows
		}
		return nil, err
	}
	return &Result{ct: ct}, nil
}

func (c *Conn) Query(ctx context.Context, query string, args ...any) (*Rows, error) {
	if c.log != nil {
		fmt.Fprint(c.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(c.log, "> args: %v\n", args)
		}
	}
	rows, err := c.conn.Query(ctx, query, args...)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = sql.ErrNoRows
		}
		return nil, err
	}
	return &Rows{rows}, nil
}

func (c *Conn) QueryScan(ctx context.Context, query string, args ...any) error {
	// TODO(Gianluca): it's not clear which of the functions called here can, in
	// practice, return the 'pgx.ErrNoRows' error, as it's not documented; so,
	// to ensure that it always catch, it is checked everywhere.
	if c.log != nil {
		fmt.Fprint(c.log, query, "\n\n")
	}
	if len(args) == 0 {
		return fmt.Errorf("sql: missing scan function")
	}
	args, arg := args[:len(args)-1], args[len(args)-1]
	switch arg.(type) {
	case func(*Rows) error:
	case func(RowsScanner) error:
	default:
		return fmt.Errorf("sql: cannot use a %T value as scan function", arg)
	}
	if c.log != nil && len(args) > 0 {
		fmt.Fprintf(c.log, "> args: %v\n", args)
	}
	rows, err := c.Query(ctx, query, args...)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = sql.ErrNoRows
		}
		return err
	}
	defer rows.Close()
	switch scan := arg.(type) {
	case func(*Rows) error:
		err = scan(rows)
	case func(RowsScanner) error:
		err = scan(rows)
	}
	if err != nil {
		if err == pgx.ErrNoRows {
			err = sql.ErrNoRows
		}
		return err
	}
	rows.Close()
	err = rows.Err()
	if err == pgx.ErrNoRows {
		err = sql.ErrNoRows
	}
	return err
}

func (c *Conn) QueryRow(ctx context.Context, query string, args ...any) *Row {
	if c.log != nil {
		fmt.Fprint(c.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(c.log, "> args: %v\n", args)
		}
	}
	row := c.conn.QueryRow(ctx, query, args...)
	return &Row{Row: row}
}

func (c *Conn) QueryVoid(ctx context.Context, query string, args ...any) error {
	if c.log != nil {
		fmt.Fprint(c.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(c.log, "> args: %v\n", args)
		}
	}
	err := c.conn.QueryRow(ctx, query, args...).Scan()
	if err == pgx.ErrNoRows {
		err = sql.ErrNoRows
	}
	return err
}

func (c *Conn) Tables(ctx context.Context, database string) ([]string, error) {
	// TODO(Gianluca): it's not clear which of the functions called here can, in
	// practice, return the 'pgx.ErrNoRows' error, as it's not documented; so,
	// to ensure that it always catch, it is checked everywhere.
	var stmt = "SHOW TABLES"
	if database != "" {
		stmt = "SHOW TABLES FROM `" + database + "`"
	}
	if c.log != nil {
		fmt.Fprint(c.log, stmt, "\n\n")
	}
	var rows, err = c.conn.Query(ctx, stmt)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = sql.ErrNoRows
		}
		return nil, err
	}
	defer rows.Close()
	var tables = []string{}
	for rows.Next() {
		var table string
		err = rows.Scan(&table)
		if err != nil {
			if err == pgx.ErrNoRows {
				err = sql.ErrNoRows
			}
			return nil, err
		}
		tables = append(tables, table)
	}
	if err = rows.Err(); err != nil {
		if err == pgx.ErrNoRows {
			err = sql.ErrNoRows
		}
		return nil, err
	}
	return tables, nil
}

func (c *Conn) Close(ctx context.Context) error {
	c.conn.Release()
	return nil
}

type Tx struct {
	tx            pgx.Tx
	log           io.Writer
	wrapped       bool
	preparedQuery bool
	ack           <-chan struct{}
}

func (tx *Tx) Exec(ctx context.Context, query string, args ...any) (*Result, error) {
	if tx.log != nil {
		fmt.Fprint(tx.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(tx.log, "> args: %v\n", args)
		}
	}
	ct, err := tx.tx.Exec(ctx, query, args...)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = sql.ErrNoRows
		}
		return nil, err
	}
	return &Result{ct}, nil
}

// Prepare prepares a statement.
// This method is not meant to be called concurrently.
// Also, only one statement can be created before calling the Close method on
// the returned Stmt.
func (tx *Tx) Prepare(ctx context.Context, query string) (*Stmt, error) {
	if tx.preparedQuery {
		return nil, errors.New("query already prepared")
	}
	tx.preparedQuery = true
	if tx.log != nil {
		fmt.Fprint(tx.log, query, "\n\n")
	}
	_, err := tx.tx.Prepare(ctx, preparedStmtName, query)
	if err != nil {
		return nil, err
	}
	return &Stmt{tx: tx}, nil
}

func (tx *Tx) Query(ctx context.Context, query string, args ...any) (*Rows, error) {
	if tx.log != nil {
		fmt.Fprint(tx.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(tx.log, "> args: %v\n", args)
		}
	}
	rows, err := tx.tx.Query(ctx, query, args...)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = sql.ErrNoRows
		}
		return nil, err
	}
	return &Rows{rows}, nil
}

func (tx *Tx) QueryScan(ctx context.Context, query string, args ...any) error {
	// TODO(Gianluca): it's not clear which of the functions called here can, in
	// practice, return the 'pgx.ErrNoRows' error, as it's not documented; so,
	// to ensure that it always catch, it is checked everywhere.
	if tx.log != nil {
		fmt.Fprint(tx.log, query, "\n\n")
	}
	if len(args) == 0 {
		return fmt.Errorf("sql: missing scan function")
	}
	args, arg := args[:len(args)-1], args[len(args)-1]
	switch arg.(type) {
	case func(*Rows) error:
	case func(RowsScanner) error:
	default:
		return fmt.Errorf("sql: cannot use a %T value as scan function", arg)
	}
	if tx.log != nil && len(args) > 0 {
		fmt.Fprintf(tx.log, "> args: %v\n", args)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = sql.ErrNoRows
		}
		return err
	}
	defer rows.Close()
	switch scan := arg.(type) {
	case func(*Rows) error:
		err = scan(rows)
	case func(RowsScanner) error:
		err = scan(rows)
	}
	if err != nil {
		if err == pgx.ErrNoRows {
			err = sql.ErrNoRows
		}
		return err
	}
	rows.Close()
	err = rows.Err()
	if err == pgx.ErrNoRows {
		err = sql.ErrNoRows
	}
	return err
}

func (tx *Tx) QueryRow(ctx context.Context, query string, args ...any) *Row {
	if tx.log != nil {
		fmt.Fprint(tx.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(tx.log, "> args: %v\n", args)
		}
	}
	row := tx.tx.QueryRow(ctx, query, args...)
	return &Row{Row: row}
}

func (tx *Tx) QueryVoid(ctx context.Context, query string, args ...any) error {
	if tx.log != nil {
		fmt.Fprint(tx.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(tx.log, "> args: %v\n", args)
		}
	}
	err := tx.tx.QueryRow(ctx, query, args...).Scan()
	if err == pgx.ErrNoRows {
		err = sql.ErrNoRows
	}
	return err
}

func (tx *Tx) Rollback(ctx context.Context) error {
	if tx.wrapped {
		return errors.New("rollback called in a wrapped transaction")
	}
	if tx.log != nil {
		fmt.Fprint(tx.log, "ROLLBACK\n\n")
	}
	return tx.tx.Rollback(ctx)
}

func (tx *Tx) Commit(ctx context.Context) error {
	if tx.wrapped {
		return errors.New("commit called in a wrapped transaction")
	}
	if tx.log != nil {
		fmt.Fprint(tx.log, "COMMIT\n\n")
	}
	return tx.tx.Commit(ctx)
}

func (tx *Tx) Tables(ctx context.Context, database string) ([]string, error) {
	// TODO(Gianluca): it's not clear which of the functions called here can, in
	// practice, return the 'pgx.ErrNoRows' error, as it's not documented; so,
	// to ensure that it always catch, it is checked everywhere.
	var rows pgx.Rows
	var err error
	if database == "" {
		rows, err = tx.tx.Query(ctx, "SHOW TABLES")
	} else {
		rows, err = tx.tx.Query(ctx, "SHOW TABLES FROM `"+database+"`")
	}
	if err != nil {
		if err == pgx.ErrNoRows {
			err = sql.ErrNoRows
		}
		return nil, err
	}
	defer rows.Close()
	var tables = []string{}
	for rows.Next() {
		var table string
		err = rows.Scan(&table)
		if err != nil {
			if err == pgx.ErrNoRows {
				err = sql.ErrNoRows
			}
			return nil, err
		}
		tables = append(tables, table)
	}
	if err = rows.Err(); err != nil {
		if err == pgx.ErrNoRows {
			err = sql.ErrNoRows
		}
		return nil, err
	}
	return tables, nil
}

func QuoteValue(value any) string {
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
					panic(fmt.Errorf("sql: Unsupported nested type '%T'", v))
				}
				values[i] = QuoteValue(v)
			}
		}
		return "(" + strings.Join(values, ",") + ")"
	default:
		panic(fmt.Errorf("sql: Unsupported type '%T'", val))
	}
}

func QuoteIdent(name string) string {
	name = strings.ReplaceAll(name, `"`, `""`)
	return `"` + name + `"`
}

func quote(s string) string {
	return "'" + escape(s) + "'"
}

func escape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

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

type Row struct {
	pgx.Row
	closed bool
}

func (r *Row) Scan(dest ...any) error {
	if r.closed {
		return errors.New("already closed")
	}
	err := r.Row.Scan(dest...)
	if err == pgx.ErrNoRows {
		err = sql.ErrNoRows
	}
	return err
}

func LimitFirstStatement(limit, first int) string {
	var statement = ""
	if limit > 0 {
		statement = "\nLIMIT " + strconv.Itoa(limit)
	}
	if first > 0 {
		statement = "\nOFFSET " + strconv.Itoa(first)
	}
	return statement
}

// IsForeignKeyViolation reports whether err is a foreign key violation error.
func IsForeignKeyViolation(err error) bool {
	if err, ok := err.(*pgconn.PgError); ok {
		return err.Code == "23503"
	}
	return false
}

// IsDuplicateKeyValue reports whether err is a duplicate key value error.
func IsDuplicateKeyValue(err error) bool {
	if err, ok := err.(*pgconn.PgError); ok {
		return err.Code == "23505"
	}
	return false
}

// ErrConstraintName returns the name of the constraint in err, if any.
func ErrConstraintName(err error) string {
	if err, ok := err.(*pgconn.PgError); ok {
		return err.ConstraintName
	}
	return ""
}
