// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2013-2017 Open2b
//

package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"chichi/pkg/open2b/decimal"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
)

type RowsScanner interface {
	Next() bool
	Scan(...any) error
}

var ErrNoRows = sql.ErrNoRows
var ErrTxDone = sql.ErrTxDone

const (
	_                 = iota
	boolColumn        // 1
	nullBoolColumn    // 2
	float32Column     // 3
	nullFloat32Column // 4
	float64Column     // 5
	nullFloat64Column // 6
	intColumn         // 7
	nullIntColumn     // 8
	int64Column       // 9
	nullInt64Column   // 10
	uint64Column      // 11
	stringColumn      // 12
	nullStringColumn  // 13
	decimalColumn     // 14
	nullDecimalColumn // 15
	timeColumn        // 16
	nullTimeColumn    // 17
)

type Connection interface {
	Exec(string, ...any) (sql.Result, error)
	Prepare(string) (*sql.Stmt, error)
	Query(string, ...any) (*Rows, error)
	QueryScan(string, ...any) error
	QueryRow(string, ...any) *Row
	Table(string) *Table
	Tables(string) ([]string, error)
}

var (
	// Ensure that *Conn, *DB and *Tx implement Connection.
	_ Connection = (*Conn)(nil)
	_ Connection = (*DB)(nil)
	_ Connection = (*Tx)(nil)
)

type DB struct {
	db         *sql.DB
	quotedName string
	schemes    map[string]scheme
	log        io.Writer
}

type scheme struct {
	quotedName string
	columns    map[string]int
}

// Open opens a mysql database. It validates its arguments without creating a connection to the database
func Open(args map[string]string) (*DB, error) {
	if args == nil {
		args = map[string]string{}
	}
	var db, err = sql.Open("mysql", args["Username"]+":"+args["Password"]+"@"+args["Address"]+"/"+args["Database"]+
		"?clientFoundRows=true&charset=utf8mb4,utf8&parseTime=true&loc=Local&allowOldPasswords=true")
	if err != nil {
		return nil, err
	}
	return &DB{db, "", map[string]scheme{}, nil}, nil
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

func (db *DB) Scheme(name, table string, columns any) {
	var s = scheme{
		quotedName: QuoteTable(table),
		columns:    map[string]int{},
	}
	var typ = reflect.TypeOf(columns)
	if typ.Kind() != reflect.Struct {
		panic(fmt.Errorf("open2b/sql: Columns definition for scheme %q must be a struct", name))
	}
	for i := 0; i < typ.NumField(); i++ {
		var column = typ.Field(i)
		var canBeNull bool
		var columnName = column.Name
		if sqlTag := column.Tag.Get("sql"); sqlTag != "" {
			parts := strings.SplitN(sqlTag, ",", 2)
			if parts[0] != "" {
				columnName = sqlTag
			}
			if len(parts) > 1 {
				canBeNull = parts[1] == "null"
			}
		}
		switch column.Type.Kind() {
		case reflect.String:
			if canBeNull {
				s.columns[columnName] = nullStringColumn
			} else {
				s.columns[columnName] = stringColumn
			}
		case reflect.Bool:
			if canBeNull {
				s.columns[columnName] = nullBoolColumn
			} else {
				s.columns[columnName] = boolColumn
			}
		case reflect.Int:
			if canBeNull {
				s.columns[columnName] = nullIntColumn
			} else {
				s.columns[columnName] = intColumn
			}
		case reflect.Int64:
			if canBeNull {
				s.columns[columnName] = nullInt64Column
			} else {
				s.columns[columnName] = int64Column
			}
		case reflect.Uint64:
			if canBeNull {
				panic("nullable uint64 is not supported")
			} else {
				s.columns[columnName] = uint64Column
			}
		case reflect.Float32:
			if canBeNull {
				s.columns[columnName] = nullFloat32Column
			} else {
				s.columns[columnName] = float32Column
			}
		case reflect.Float64:
			if canBeNull {
				s.columns[columnName] = nullFloat64Column
			} else {
				s.columns[columnName] = float64Column
			}
		case reflect.Ptr: // *decimal.Dec
			var t = column.Type.Elem()
			if t.Kind() == reflect.Struct && t.Name() == "Dec" {
				if canBeNull {
					s.columns[columnName] = nullDecimalColumn
				} else {
					s.columns[columnName] = decimalColumn
				}
			}
		case reflect.Struct: // time.Time
			if column.Type.Name() == "Time" {
				if canBeNull {
					s.columns[columnName] = nullTimeColumn
				} else {
					s.columns[columnName] = timeColumn
				}
			}
		}
		if _, ok := s.columns[columnName]; !ok {
			panic(fmt.Errorf("open2b/sql: Type %q of table field \"%s.%s\" is not supported", column.Type.Name(), table, columnName))
		}
	}
	db.schemes[name] = s
}

func (db *DB) Table(name string) *Table {
	var s, ok = db.schemes[name]
	if !ok {
		panic(fmt.Errorf("open2b/sql: Scheme %q does not exist", name))
	}
	return newTable(db, s)
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
	return &Tx{tx: tx, schemes: db.schemes, log: db.log}, nil
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
		var err = f(&Tx{tx, db.schemes, db.log})
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
		var err = f(&Tx{tx, db.schemes, db.log})
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
		conn:    conn,
		log:     db.log,
		schemes: db.schemes,
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
	conn    *sql.Conn
	schemes map[string]scheme
	log     io.Writer
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

func (c *Conn) Table(name string) *Table {
	var s, ok = c.schemes[name]
	if !ok {
		panic(fmt.Errorf("open2b/sql: Scheme %q does not exist", name))
	}
	return newTable(c, s)
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
	tx      *sql.Tx
	schemes map[string]scheme
	log     io.Writer
}

func (tx *Tx) Table(name string) *Table {
	var s, ok = tx.schemes[name]
	if !ok {
		panic(fmt.Errorf("open2b/sql: Scheme %q does not exist", name))
	}
	return newTable(tx, s)
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

func QuoteColumn(name string) string {
	if strings.ContainsAny(name, ".") {
		// !!! verificare che non ci siano caratteri unicode che contengano '.' nel codice !!!
		name = "`" + strings.Join(strings.Split(name, "."), "`.`") + "`"
	} else {
		name = "`" + name + "`"
	}
	return name
}

func QuoteTable(name string) string {
	if strings.ContainsAny(name, ".") {
		name = "`" + strings.Join(strings.Split(name, "."), "`.`") + "`"
	} else {
		name = "`" + name + "`"
	}
	return name
}

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
	case *decimal.Dec:
		return val.String()
	case time.Time:
		return "'" + val.Format("2006-01-02 15:04:05") + "'"
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
	case []*decimal.Dec:
		if len(val) == 1 {
			return "(" + val[0].String() + ")"
		}
		var values = make([]string, len(val))
		for i, v := range val {
			values[i] = "'" + v.String() + "'"
		}
		return "(" + strings.Join(values, ",") + ")"
	case []any:
		var values = make([]string, len(val))
		for i, v := range val {
			if v == nil {
				values[i] = "NULL"
			} else {
				switch v.(type) {
				case bool, int, int64, uint, uint64, float32, float64, string, *decimal.Dec, time.Time:
				default:
					panic(fmt.Errorf("open2b/sql: Unsupported nested type '%T'", v))
				}
				values[i] = Quote(v)
			}
		}
		return "(" + strings.Join(values, ",") + ")"
	default:
		panic(fmt.Errorf("open2b/sql: Unsupported type '%T'", val))
	}
}

func quote(s string) string {
	if len(s) == 0 {
		return "''"
	}
	var toQuote = false
	for _, c := range []byte{'\n', '\r', '\'', '"', '\\', '\032', 0} {
		if strings.IndexByte(s, c) != -1 {
			toQuote = true
			break
		}
	}
	if toQuote {
		var quoted = make([]byte, 2+len(s)*2)
		quoted[0] = '\''
		var p = 1
		for i := 0; i < len(s); i++ {
			var escape byte
			switch s[i] {
			case 0:
				escape = '0'
			case '\n':
				escape = 'n'
			case '\r':
				escape = 'r'
			case '\\':
				escape = '\\'
			case '\'':
				escape = '\''
			case '"':
				escape = '"'
			case '\032':
				escape = 'Z'
			}
			if escape == 0 {
				quoted[p] = s[i]
				p++
			} else {
				quoted[p] = '\\'
				quoted[p+1] = escape
				p += 2
			}
		}
		quoted[p] = '\''
		return string(quoted[:p+1])
	}
	return "'" + s + "'"
}

type Rows struct {
	*sql.Rows
}

func (rs *Rows) Scan(dest ...any) error {
	for i, d := range dest {
		if v, ok := d.(**decimal.Dec); ok {
			dest[i] = &nullDecimal{v}
		}
	}
	return rs.Rows.Scan(dest...)
}

type Row struct {
	*sql.Row
}

func (r *Row) Scan(dest ...any) error {
	for i, d := range dest {
		if v, ok := d.(**decimal.Dec); ok {
			dest[i] = &nullDecimal{v}
		}
		if v, ok := d.(**decimal.Dec); ok {
			dest[i] = &nullDecimal{v}
		}
	}
	return r.Row.Scan(dest...)
}

type nullDecimal struct {
	d **decimal.Dec
}

func (nd nullDecimal) Scan(src any) error {
	if src == nil {
		*nd.d = nil
		return nil
	}
	var b *decimal.Dec
	var ok = true
	switch s := src.(type) {
	case string:
		b, ok = decimal.ParseString(s)
	case int:
		b = decimal.Int(s)
	case int64:
		b = decimal.Int64(s)
	case float64:
		b = decimal.Float64(s, int(decimal.Accuracy))
	case []byte:
		b, ok = decimal.ParseString(string(s))
	default:
		return errors.New("Incompatible type for *decimal.Dec")
	}
	if !ok {
		return errors.New("Failed to parse *decimal.Dec")
	}
	*nd.d = b
	return nil
}
