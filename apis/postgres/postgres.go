// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
)

type RowsScanner interface {
	Next() bool
	Scan(...any) error
}

var ErrNoRows = sql.ErrNoRows
var ErrTxDone = sql.ErrTxDone

type Connection interface {
	Exec(string, ...any) (sql.Result, error)
	Prepare(string) (*sql.Stmt, error)
	Query(string, ...any) (*Rows, error)
	QueryScan(string, ...any) error
	QueryRow(string, ...any) *Row
	Tables(string) ([]string, error)
}

var (
	// Ensure that *Conn, *DB and *Tx implement Connection.
	_ Connection = (*Conn)(nil)
	_ Connection = (*DB)(nil)
	_ Connection = (*Tx)(nil)
)

type DB struct {
	db  *sql.DB
	log io.Writer
}

type Options struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

// Open opens a PostgreSQL database. It validates its arguments without creating a connection to the database
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
	var db, err = sql.Open("pgx", u.String())
	if err != nil {
		return nil, err
	}
	return &DB{db, nil}, nil
}

func (db *DB) Close() error {
	return db.db.Close()
}

func (db *DB) SetQueryLog(log io.Writer) {
	db.log = log
}

func (db *DB) Ping() error {
	return db.db.Ping()
}

func (db *DB) SetMaxIdleConns(n int) {
	db.db.SetMaxIdleConns(n)
}

func (db *DB) SetMaxOpenConns(n int) {
	db.db.SetMaxOpenConns(n)
}

func (db *DB) Exec(query string, args ...any) (sql.Result, error) {
	if db.log != nil {
		fmt.Fprint(db.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(db.log, "> args: %v\n", args)
		}
	}
	return db.db.Exec(query, args...)
}

func (db *DB) Prepare(query string) (*sql.Stmt, error) {
	if db.log != nil {
		fmt.Fprint(db.log, query, "\n\n")
	}
	return db.db.Prepare(query)
}

func (db *DB) Query(query string, args ...any) (*Rows, error) {
	if db.log != nil {
		fmt.Fprint(db.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(db.log, "> args: %v\n", args)
		}
	}
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return &Rows{rows}, nil
}

func (db *DB) QueryRow(query string, args ...any) *Row {
	if db.log != nil {
		fmt.Fprint(db.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(db.log, "> args: %v\n", args)
		}
	}
	row := db.db.QueryRow(query, args...)
	return &Row{row}
}

func (db *DB) QueryScan(query string, args ...any) error {
	if db.log != nil {
		fmt.Fprint(db.log, query, "\n\n")
	}
	if len(args) == 0 {
		return fmt.Errorf("open2b/sql: missing scan function")
	}
	args, arg := args[:len(args)-1], args[len(args)-1]
	switch arg.(type) {
	case func(*Rows) error:
	case func(RowsScanner) error:
	default:
		return fmt.Errorf("open2b/sql: cannot use a %T value as scan function", arg)
	}
	if db.log != nil && len(args) > 0 {
		fmt.Fprintf(db.log, "> args: %v\n", args)
	}
	rows, err := db.db.Query(query, args...)
	if err != nil {
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
		return err
	}
	if err = rows.Close(); err != nil {
		return err
	}
	return rows.Err()
}

func (db *DB) Begin() (*Tx, error) {
	tx, err := db.db.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx, log: db.log}, nil
}

func (db *DB) SerializableTransaction(f func(tx *Tx) error) error {
	var tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	return func() error {
		defer func() {
			if err := recover(); err != nil {
				_ = tx.Rollback()
				panic(err)
			}
		}()
		var err = f(&Tx{tx, db.log})
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		err = tx.Commit()
		if err != nil && err != sql.ErrTxDone {
			return err
		}
		return nil
	}()
}

func (db *DB) Transaction(f func(tx *Tx) error) error {
	var tx, err = db.db.Begin()
	if err != nil {
		return err
	}
	return func() error {
		defer func() {
			if err := recover(); err != nil {
				_ = tx.Rollback()
				panic(err)
			}
		}()
		var err = f(&Tx{tx, db.log})
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		err = tx.Commit()
		if err != nil && err != sql.ErrTxDone {
			return err
		}
		return nil
	}()
}

func (db *DB) Conn(ctx context.Context) (*Conn, error) {
	conn, err := db.db.Conn(context.Background())
	if err != nil {
		return nil, err
	}
	return &Conn{
		conn: conn,
		log:  db.log,
	}, nil
}

func (db *DB) Tables(database string) ([]string, error) {
	var stmt = "SHOW TABLES"
	if database != "" {
		stmt = "SHOW TABLES FROM `" + database + "`"
	}
	if db.log != nil {
		fmt.Fprint(db.log, stmt, "\n\n")
	}
	var rows, err = db.db.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables = []string{}
	for rows.Next() {
		var table string
		err = rows.Scan(&table)
		if err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return tables, nil
}

type Conn struct {
	conn *sql.Conn
	log  io.Writer
}

func (c *Conn) Exec(query string, args ...any) (sql.Result, error) {
	if c.log != nil {
		fmt.Fprint(c.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(c.log, "> args: %v\n", args)
		}
	}
	return c.conn.ExecContext(context.Background(), query, args...)
}

func (c *Conn) Prepare(query string) (*sql.Stmt, error) {
	if c.log != nil {
		fmt.Fprint(c.log, query, "\n\n")
	}
	return c.conn.PrepareContext(context.Background(), query)
}

func (c *Conn) Query(query string, args ...any) (*Rows, error) {
	if c.log != nil {
		fmt.Fprint(c.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(c.log, "> args: %v\n", args)
		}
	}
	rows, err := c.conn.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	return &Rows{rows}, nil
}

func (conn *Conn) QueryScan(query string, args ...any) error {
	if conn.log != nil {
		fmt.Fprint(conn.log, query, "\n\n")
	}
	if len(args) == 0 {
		return fmt.Errorf("open2b/sql: missing scan function")
	}
	args, arg := args[:len(args)-1], args[len(args)-1]
	switch arg.(type) {
	case func(*Rows) error:
	case func(RowsScanner) error:
	default:
		return fmt.Errorf("open2b/sql: cannot use a %T value as scan function", arg)
	}
	if conn.log != nil && len(args) > 0 {
		fmt.Fprintf(conn.log, "> args: %v\n", args)
	}
	rows, err := conn.Query(query, args...)
	if err != nil {
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
		return err
	}
	if err = rows.Close(); err != nil {
		return err
	}
	return rows.Err()
}

func (c *Conn) QueryRow(query string, args ...any) *Row {
	if c.log != nil {
		fmt.Fprint(c.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(c.log, "> args: %v\n", args)
		}
	}
	row := c.conn.QueryRowContext(context.Background(), query, args...)
	return &Row{row}
}

func (c *Conn) Tables(database string) ([]string, error) {
	var stmt = "SHOW TABLES"
	if database != "" {
		stmt = "SHOW TABLES FROM `" + database + "`"
	}
	if c.log != nil {
		fmt.Fprint(c.log, stmt, "\n\n")
	}
	var rows, err = c.conn.QueryContext(context.Background(), stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables = []string{}
	for rows.Next() {
		var table string
		err = rows.Scan(&table)
		if err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return tables, nil
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

type Tx struct {
	tx  *sql.Tx
	log io.Writer
}

func (tx *Tx) Exec(query string, args ...any) (sql.Result, error) {
	if tx.log != nil {
		fmt.Fprint(tx.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(tx.log, "> args: %v\n", args)
		}
	}
	return tx.tx.Exec(query, args...)
}

func (tx *Tx) Prepare(query string) (*sql.Stmt, error) {
	if tx.log != nil {
		fmt.Fprint(tx.log, query, "\n\n")
	}
	return tx.tx.Prepare(query)
}

func (tx *Tx) Query(query string, args ...any) (*Rows, error) {
	if tx.log != nil {
		fmt.Fprint(tx.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(tx.log, "> args: %v\n", args)
		}
	}
	rows, err := tx.tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return &Rows{rows}, nil
}

func (tx *Tx) QueryScan(query string, args ...any) error {
	if tx.log != nil {
		fmt.Fprint(tx.log, query, "\n\n")
	}
	if len(args) == 0 {
		return fmt.Errorf("open2b/sql: missing scan function")
	}
	args, arg := args[:len(args)-1], args[len(args)-1]
	switch arg.(type) {
	case func(*Rows) error:
	case func(RowsScanner) error:
	default:
		return fmt.Errorf("open2b/sql: cannot use a %T value as scan function", arg)
	}
	if tx.log != nil && len(args) > 0 {
		fmt.Fprintf(tx.log, "> args: %v\n", args)
	}
	rows, err := tx.Query(query, args...)
	if err != nil {
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
		return err
	}
	if err = rows.Close(); err != nil {
		return err
	}
	return rows.Err()
}

func (tx *Tx) QueryRow(query string, args ...any) *Row {
	if tx.log != nil {
		fmt.Fprint(tx.log, query, "\n\n")
		if len(args) > 0 {
			fmt.Fprintf(tx.log, "> args: %v\n", args)
		}
	}
	row := tx.tx.QueryRow(query, args...)
	return &Row{row}
}

func (tx *Tx) Rollback() error {
	return tx.tx.Rollback()
}

func (tx *Tx) Commit() error {
	return tx.tx.Commit()
}

func (tx *Tx) Tables(database string) ([]string, error) {
	var rows *sql.Rows
	var err error
	if database == "" {
		rows, err = tx.tx.Query("SHOW TABLES")
	} else {
		rows, err = tx.tx.Query("SHOW TABLES FROM `" + database + "`")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables = []string{}
	for rows.Next() {
		var table string
		err = rows.Scan(&table)
		if err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	if err = rows.Err(); err != nil {
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
			return "('" + val[0].Format("2006-01-02 15:04:05") + "')"
		}
		var values = make([]string, len(val))
		for i, v := range val {
			values[i] = "'" + v.Format("2006-01-02 15:04:05") + "'"
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
					panic(fmt.Errorf("open2b/sql: Unsupported nested type '%T'", v))
				}
				values[i] = QuoteValue(v)
			}
		}
		return "(" + strings.Join(values, ",") + ")"
	default:
		panic(fmt.Errorf("open2b/sql: Unsupported type '%T'", val))
	}
}

func QuoteIdent(name string) string {
	name = strings.ReplaceAll(name, "\x00", "")
	name = strings.ReplaceAll(name, `"`, `""`)
	return `"` + name + `"`
}

func quote(s string) string {
	s = strings.ReplaceAll(s, "\x00", "")
	s = strings.ReplaceAll(s, "'", "''")
	return "'" + s + "'"
}

type Rows struct {
	*sql.Rows
}

func (rs *Rows) Scan(dest ...any) error {
	return rs.Rows.Scan(dest...)
}

type Row struct {
	*sql.Row
}

func (r *Row) Scan(dest ...any) error {
	return r.Row.Scan(dest...)
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
