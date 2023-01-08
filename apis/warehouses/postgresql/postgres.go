//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package postgresql

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"chichi/apis/types"
	"chichi/apis/warehouses"

	"github.com/shopspring/decimal"
	"golang.org/x/exp/slices"
)

//go:embed connections_users.sql
var createConnectionsUsersTable string

var _ warehouses.Warehouse = &postgreSQL{}
var _ warehouses.Batch = &batch{}

type postgreSQL struct {
	mu       sync.Mutex // for the db and closed fields
	db       *sql.DB
	closed   bool
	settings *psSettings
}

type psSettings struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Schema   string
}

// OpenPostgres opens a PostgreSQL data warehouse with the given settings.
func OpenPostgres(settings []byte) (warehouses.Warehouse, error) {
	var s psSettings
	err := json.Unmarshal(settings, &s)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal settings: %s", err)
	}
	// Validate Host.
	if n := len(s.Host); n == 0 || n > 253 {
		return nil, fmt.Errorf("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65536 {
		return nil, fmt.Errorf("port must be in range [1,65536]")
	}
	// Validate Username.
	if n := len(s.Username); n < 1 || n > 63 {
		return nil, fmt.Errorf("username length in bytes must be in range [1,63]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 100 {
		return nil, fmt.Errorf("password length must be in range [1,100]")
	}
	// Validate Database.
	if n := len(s.Database); n < 1 || n > 63 {
		return nil, fmt.Errorf("database length in bytes must be in range [1,63]")
	}
	// Validate Schema.
	if n := len(s.Schema); n < 1 || n > 63 {
		return nil, fmt.Errorf("schema length in bytes must be in range [1,63]")
	}
	if !warehouses.IsValidSchemaName(s.Schema) {
		return nil, fmt.Errorf("schema must start with [A-Za-z_] and subsequently contain only [A-Za-z0-9_]")
	}
	if strings.HasPrefix(s.Schema, "pg_") {
		return nil, fmt.Errorf("schema cannot start with 'pg_'")
	}
	return &postgreSQL{settings: &s}, nil
}

// Close closes the warehouse. It will not allow any new queries to run, and it
// waits for the current ones to finish.
func (warehouse *postgreSQL) Close() error {
	var err error
	warehouse.mu.Lock()
	if warehouse.db != nil {
		err = warehouse.db.Close()
		warehouse.db = nil
		warehouse.closed = true
	}
	warehouse.mu.Unlock()
	return err
}

// CreateTables creates the data warehouse tables. schema is the schema of the
// users table. If a table already exists it returns an Error error.
func (warehouse *postgreSQL) CreateTables(ctx context.Context, schema types.Type) error {
	// Build the "create" statement for the users table.
	var createTables []string
	var b strings.Builder
	b.WriteString("CREATE TABLE \"users\" (\nid SERIAL,\n")
	for _, p := range schema.Properties() {
		if !types.IsValidPropertyName(p.Name) {
			panic("property name is not valid")
		}
		tables, err := warehouse.serializeColumn(&b, "users", p.Name, p.Type)
		if err != nil {
			return err
		}
		createTables = append(createTables, tables...)
	}
	b.WriteString("PRIMARY KEY (id)\n)")
	createTables = append(createTables, b.String())
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return warehouses.WrapError(err)
	}
	// Create the tables.
	for i := len(createTables) - 1; i >= 0; i-- {
		_, err = tx.ExecContext(ctx, createTables[i])
		if err != nil {
			_ = tx.Rollback()
			return warehouses.WrapError(err)
		}
	}
	// Create the "connections_users" table.
	_, err = tx.ExecContext(ctx, createConnectionsUsersTable)
	if err != nil {
		_ = tx.Rollback()
		return warehouses.WrapError(err)
	}
	err = tx.Commit()
	return warehouses.WrapError(err)
}

// Exec executes a query without returning any rows. args are the placeholders.
// If the query fails, it returns an Error value.
func (warehouse *postgreSQL) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	db, err := warehouse.connection()
	if err != nil {
		return nil, err
	}
	r, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, warehouses.WrapError(err)
	}
	return warehouses.Result{Result: r}, nil
}

// Ping checks whether the connection to the data warehouse is active and, if
// necessary, establishes a new connection.
func (warehouse *postgreSQL) Ping(ctx context.Context) error {
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	return db.PingContext(ctx)
}

// PrepareBatch creates a prepared batch statement for inserting rows in
// batch and returns it. table specifies the table in which the rows will be
// inserted, and columns specifies the columns.
func (warehouse *postgreSQL) PrepareBatch(ctx context.Context, table string, columns []string) (warehouses.Batch, error) {
	if !warehouses.IsValidIdentifier(table) {
		return nil, fmt.Errorf("table name %q is not a valid identifier", table)
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("columns cannot be empty")
	}
	batch := &batch{
		warehouse: warehouse,
		ctx:       ctx,
		columns:   slices.Clone(columns),
		buf:       strings.Builder{},
	}
	batch.buf.WriteString("INSERT INTO ")
	batch.buf.WriteString(table)
	batch.buf.WriteString(" (")
	for i, column := range columns {
		if i > 0 {
			batch.buf.WriteByte(',')
		}
		if !warehouses.IsValidIdentifier(column) {
			return nil, fmt.Errorf("column name %q is not a valid identifier", column)
		}
		batch.buf.WriteString(column)
	}
	batch.buf.WriteString(") ")
	return batch, nil
}

// Query executes a query that returns rows. args are the placeholders.
// If the query fails, it returns an Error value.
func (warehouse *postgreSQL) Query(ctx context.Context, query string, args ...any) (*warehouses.Rows, error) {
	db, err := warehouse.connection()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, warehouses.WrapError(err)
	}
	return &warehouses.Rows{Rows: rows}, nil
}

// QueryRow executes a query that should return at most one row.
// If the query fails, it returns an Error value.
func (warehouse *postgreSQL) QueryRow(ctx context.Context, query string, args ...any) warehouses.Row {
	db, err := warehouse.connection()
	if err != nil {
		return warehouses.Row{Error: err}
	}
	row := db.QueryRowContext(ctx, query, args...)
	return warehouses.Row{Row: row}
}

// Settings returns the data warehouse settings.
func (warehouse *postgreSQL) Settings() []byte {
	s, _ := json.Marshal(warehouse.settings)
	return s
}

// Tables returns the tables of the data warehouse.
// It returns only the tables 'users', 'groups', 'events', and the tables with
// prefix 'users_', 'groups_' and 'events_'.
func (warehouse *postgreSQL) Tables(ctx context.Context) ([]*warehouses.Table, error) {

	// Get the connection.
	db, err := warehouse.connection()
	if err != nil {
		return nil, err
	}

	var table *warehouses.Table
	var tables []*warehouses.Table

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, warehouses.WrapError(err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	// Read columns.
	query := "SELECT c.table_name, c.column_name, c.is_nullable, c.data_type, c.character_maximum_length," +
		" c.numeric_precision, c.numeric_precision_radix, c.numeric_scale, c.is_updatable," +
		" col_description(c.table_name::regclass::oid, c.ordinal_position)\n" +
		"FROM information_schema.columns c\n" +
		"INNER JOIN information_schema.tables t ON c.table_name = t.table_name AND c.table_schema = t.table_schema\n" +
		"WHERE t.table_schema = '" + warehouse.settings.Schema + "' AND t.table_type = 'BASE TABLE' AND" +
		" ( t.table_name IN ('users', 'groups', 'events') OR t.table_name LIKE 'users\\__%' OR" +
		" t.table_name LIKE 'groups\\__%' OR t.table_name LIKE 'events\\__%' )\n" +
		"ORDER BY c.table_name, c.ordinal_position"

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, warehouses.WrapError(err)
	}
	for rows.Next() {
		var tableName, columnName, isNullable, typ, charLength, precision, radix, scale, isUpdatable, description sql.NullString
		if err = rows.Scan(&tableName, &columnName, &isNullable, &typ, &charLength, &precision, &radix, &scale, &isUpdatable, &description); err != nil {
			_ = rows.Close()
			return nil, warehouses.WrapError(err)
		}
		if !columnName.Valid {
			return nil, warehouses.NewError("data warehouse has returned NULL as column name")
		}
		if !typ.Valid {
			return nil, warehouses.NewError("data warehouse has returned NULL as column data type")
		}
		if !types.IsValidPropertyName(columnName.String) {
			return nil, warehouses.NewError("column name %q is not supported", columnName.String)
		}
		var t types.Type
		switch typ.String {
		case "smallint":
			t = types.Int16()
		case "integer":
			t = types.Int()
		case "bigint":
			t = types.Int64()
		case "numeric":
			// Parse precision radix.
			if !radix.Valid {
				return nil, warehouses.NewError("data warehouse has returned NULL as precision radix for column %s", columnName.String)
			}
			radix, _ := strconv.Atoi(radix.String)
			if radix != 2 && radix != 10 {
				return nil, warehouses.NewError("data warehouse has returned an invalid precision radix for column %s", columnName.String)
			}
			// Parse precision.
			if !precision.Valid {
				return nil, warehouses.NewError("data warehouse has returned NULL as precision for column %s", columnName.String)
			}
			p, err := strconv.ParseInt(precision.String, radix, 64)
			if err != nil || p < 1 {
				return nil, warehouses.NewError("data warehouse has returned an invalid precision for column %s: %s", columnName.String, precision.String)
			}
			// Parse scale.
			if !scale.Valid {
				return nil, warehouses.NewError("data warehouse has returned NULL as scale for column %s", columnName.String)
			}
			s, err := strconv.ParseInt(scale.String, radix, 64)
			if err != nil || s < 0 || s > p {
				return nil, warehouses.NewError("data warehouse has returned an invalid scale for column %s: %s", columnName.String, scale.String)
			}
			t = types.Decimal(int(p), int(s))
		case "real":
			t = types.Float32()
		case "double precision":
			t = types.Float()
		case "character varying", "character":
			if charLength.Valid {
				chars, _ := strconv.Atoi(charLength.String)
				if chars < 1 {
					return nil, warehouses.NewError("data warehouse has returned an invalid character maximum length for column %s", columnName.String)
				}
				t = types.Text(types.Chars(chars))
			} else {
				t = types.Text()
			}
		case "text":
			t = types.Text()
		case "timestamp without time zone", "timestamp with time zone":
			t = types.DateTime("2006-01-02 15:04:05.999999")
		case "date":
			t = types.Date("2006-01-02")
		case "time without time zone", "time with time zone":
			t = types.Time("15:04:05")
		case "boolean":
			t = types.Boolean()
		case "uuid":
			t = types.UUID()
		case "json", "jsonb":
			t = types.JSON()
		default:
			return nil, warehouses.NewError("type of column %q.%q is not supported: %s", tableName.String, columnName.String, typ.String)
		}
		column := &warehouses.Column{
			Name:        columnName.String,
			Type:        t,
			IsNullable:  isNullable.String == "YES",
			IsUpdatable: isUpdatable.String == "YES",
		}
		if description.Valid {
			column.Description = description.String
		}
		if table == nil || tableName.String != table.Name {
			table = &warehouses.Table{Name: tableName.String}
			tables = append(tables, table)
		}
		table.Columns = append(table.Columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, warehouses.WrapError(err)
	}

	err = tx.Commit()
	tx = nil
	if err != nil {
		return nil, err
	}

	return tables, nil
}

// Type returns the type of the warehouse.
func (warehouse *postgreSQL) Type() warehouses.Type {
	return warehouses.PostgreSQL
}

// Users returns the users, with only the properties in schema, ordered by
// order if order is not the zero Property, and in range [first,first+limit]
// with first >= 0 and 0 < limit <= 1000.
//
// If a query to the warehouse fails, it returns an Error value.
// If an argument is not valid, it panics.
func (warehouse *postgreSQL) Users(ctx context.Context, schema types.Type, order types.Property, first, limit int) ([][]any, error) {

	db, err := warehouse.connection()
	if err != nil {
		return nil, err
	}

	properties := schema.Properties()

	// Build the query.
	var query strings.Builder
	query.WriteString(`SELECT "`)
	for i, p := range properties {
		if i > 0 {
			query.WriteString(`", "`)
		}
		if !types.IsValidPropertyName(p.Name) {
			panic(fmt.Sprintf("invalid property name: %q", p.Name))
		}
		query.WriteString(p.Name)
	}
	query.WriteString(`" FROM users`)
	if order.Name != "" {
		if !types.IsValidPropertyName(order.Name) {
			panic(fmt.Sprintf("invalid property name: %q", order.Name))
		}
		query.WriteString(" ORDER BY ")
		query.WriteString(order.Name)
	}
	query.WriteString(" LIMIT ")
	query.WriteString(strconv.Itoa(limit))
	if first > 0 {
		query.WriteString(" OFFSET ")
		query.WriteString(strconv.Itoa(first))
	}

	// Execute the query.
	var users [][]any
	rows, err := db.QueryContext(ctx, query.String())
	if err != nil {
		return nil, warehouses.WrapError(err)
	}
	for rows.Next() {
		user := make([]any, len(properties))
		for i := range user {
			typ := properties[i].Type
			switch typ.PhysicalType() {
			case types.PtBoolean:
				var v bool
				user[i] = &v
			case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64:
				var v int
				user[i] = &v
			case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
				var v uint
				user[i] = &v
			case types.PtFloat, types.PtFloat32:
				var v float64
				user[i] = &v
			case types.PtDecimal:
				var v decimal.Decimal
				user[i] = &v
			case types.PtDateTime, types.PtDate:
				var v time.Time
				user[i] = &v
			case types.PtTime, types.PtYear:
				var v int
				user[i] = &v
			case types.PtUUID, types.PtJSON, types.PtText, types.PtArray, types.PtObject, types.PtMap:
				var v string
				user[i] = &v
			}
		}
		if err = rows.Scan(user...); err != nil {
			_ = rows.Close()
			return nil, warehouses.WrapError(err)
		}
		users = append(users, user)
	}
	if err = rows.Err(); err != nil {
		return nil, warehouses.WrapError(err)
	}
	err = rows.Close()
	if err != nil {
		log.Printf("cannot close rows: %s", err)
	}
	if users == nil {
		users = [][]any{}
	}

	return users, nil
}

// connection returns the database connection.
func (warehouse *postgreSQL) connection() (*sql.DB, error) {
	warehouse.mu.Lock()
	defer warehouse.mu.Unlock()
	if warehouse.closed {
		return nil, warehouses.WrapError(errors.New("warehouse is closed"))
	}
	if warehouse.settings == nil {
		return nil, warehouses.WrapError(errors.New("there are no settings"))
	}
	if warehouse.db != nil {
		return warehouse.db, nil
	}
	db, err := sql.Open("pgx", warehouse.settings.dsn())
	if err != nil {
		return nil, warehouses.WrapError(err)
	}
	warehouse.db = db
	return db, nil
}

// dsn returns the connection string, from s, in the URL format.
func (s *psSettings) dsn() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(s.Username, s.Password),
		Host:     net.JoinHostPort(s.Host, strconv.Itoa(s.Port)),
		Path:     "/" + url.PathEscape(s.Database),
		RawQuery: "search_path=" + url.QueryEscape(s.Schema),
	}
	return u.String()
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func (s *psSettings) testConnection(ctx context.Context) error {
	db, err := sql.Open("pgx", s.dsn())
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxIdleConns(0)
	return db.PingContext(ctx)
}

// serializeColumn serializes a column where name and typ are the name and the
// type of the column. If typ is an object, it will serialize each property of
// the object as a column.
func (warehouse *postgreSQL) serializeColumn(b *strings.Builder, table, name string, typ types.Type) ([]string, error) {
	var createTables []string
	pt := typ.PhysicalType()
	if pt == types.PtObject {
		for _, p := range typ.Properties() {
			if !types.IsValidPropertyName(p.Name) {
				panic("property name is not valid")
			}
			tables, err := warehouse.serializeColumn(b, table, name+"_"+p.Name, p.Type)
			if err != nil {
				return nil, err
			}
			createTables = append(createTables, tables...)
		}
		return createTables, nil
	}
	if pt == types.PtArray {
		if itemType := typ.ItemType(); itemType.PhysicalType() == types.PtObject {
			// Build the "create" statement for the table of the object.
			refTable := table + "_" + name
			var b strings.Builder
			b.WriteString(`CREATE TABLE "`)
			b.WriteString(refTable)
			b.WriteString("\" (\n\"id\" SERIAL,\n\"")
			b.WriteString(table)
			b.WriteString("_id\" integer NOT NULL REFERENCES \"users\" ON DELETE CASCADE,\n")
			for _, p := range itemType.Properties() {
				if !types.IsValidPropertyName(p.Name) {
					panic("property name is not valid")
				}
				tables, err := warehouse.serializeColumn(&b, refTable, p.Name, p.Type)
				if err != nil {
					return nil, err
				}
				createTables = append(createTables, tables...)
			}
			b.WriteString("PRIMARY KEY (id)\n)")
			createTables = append(createTables, b.String())
			return createTables, nil
		}
		return nil, errors.New("PtArray type is not supported")
	}
	b.WriteByte('"')
	b.WriteString(name)
	b.WriteString(`" `)
	switch pt {
	case types.PtBoolean:
		b.WriteString("boolean")
	case types.PtInt:
		b.WriteString("integer")
	case types.PtInt8:
		b.WriteString("smallint")
		if name != "" {
			b.WriteString(" CHECK(")
			b.WriteString(name)
			b.WriteString("BETWEEN -128 AND 127)")
		}
	case types.PtInt16:
		b.WriteString("smallint")
	case types.PtInt64:
		b.WriteString("bigint")
	case types.PtUInt:
		b.WriteString("bigint")
		if name != "" {
			b.WriteString(" CHECK (")
			b.WriteString(name)
			b.WriteString(" BETWEEN 0 AND 2^32)")
		}
	case types.PtUInt8:
		b.WriteString("smallint")
		if name != "" {
			b.WriteString(" CHECK (")
			b.WriteString(name)
			b.WriteString(" BETWEEN 0 AND 2^8)")
		}
	case types.PtUInt16:
		b.WriteString("int")
		if name != "" {
			b.WriteString(" CHECK (")
			b.WriteString(name)
			b.WriteString(" BETWEEN 0 AND 2^16)")
		}
	case types.PtUInt64:
		return nil, errors.New("PtUint64 type is not supported")
	case types.PtFloat:
		b.WriteString("double precision")
	case types.PtFloat32:
		b.WriteString("real")
	case types.PtDecimal:
		b.WriteString("numeric(")
		b.WriteString(strconv.Itoa(typ.Precision()))
		b.WriteByte(',')
		b.WriteString(strconv.Itoa(typ.Scale()))
		b.WriteByte(')')
	case types.PtDateTime:
		b.WriteString("timestamp")
	case types.PtDate:
		b.WriteString("date")
	case types.PtTime:
		b.WriteString("time")
	case types.PtYear:
		b.WriteString("smallint CHECK (")
		b.WriteString(name)
		b.WriteString(" BETWEEN 1 AND 9999)")
	case types.PtUUID:
		b.WriteString("uuid")
	case types.PtJSON:
		b.WriteString("jsonb")
	case types.PtText:
		n, ok := typ.CharLen()
		if ok {
			b.WriteString("varchar(")
			b.WriteString(strconv.Itoa(n))
			b.WriteByte(')')
		} else {
			b.WriteString("varchar")
		}
		n, ok = typ.ByteLen()
		if ok {
			b.WriteString(" CHECK (octet_length(")
			b.WriteString(name)
			b.WriteString(") <= ")
			b.WriteString(strconv.Itoa(n))
			b.WriteByte(')')
		}
	case types.PtMap:
		b.WriteString("jsonb")
	default:
		panic(fmt.Errorf("unexpected schema physical type: %d", typ.PhysicalType()))
	}
	if !typ.Null() {
		b.WriteString(" NOT NULL")
	}
	b.WriteString(",\n")
	return createTables, nil
}

// isArrayOfObjects reports whether t is an array of objects.
func isArrayOfObjects(t types.Type) bool {
	return t.PhysicalType() == types.PtArray && t.ItemType().PhysicalType() == types.PtObject
}

// batch implements the Batch interface.
type batch struct {
	warehouse *postgreSQL
	ctx       context.Context
	columns   []string
	buf       strings.Builder
	appended  bool
	err       error
}

// Abort aborts the execution of the batch insert.
func (batch *batch) Abort() error {
	if batch.err != nil {
		return batch.err
	}
	batch.err = errors.New("batch execution aborted")
	return nil
}

// Append appends the values of a row to batch.
func (batch *batch) Append(v ...any) error {
	if batch.err != nil {
		return batch.err
	}
	if len(v) != len(batch.columns) {
		return fmt.Errorf("cannot append values: expected %d values, got %d", len(batch.columns), len(v))
	}
	if batch.appended {
		batch.buf.WriteString(",(")
	} else {
		batch.buf.WriteString("(")
		batch.appended = true
	}
	for i, value := range v {
		if i > 0 {
			batch.buf.WriteByte(',')
		}
		quoteValue(&batch.buf, value)
	}
	batch.buf.WriteString(")")
	return nil
}

// AppendStruct appends the values of a row, read from the fields of the struct
// v, to batch. It returns an error if v is not a struct or pointer to a struct.
func (batch *batch) AppendStruct(v any) error {
	if batch.err != nil {
		return batch.err
	}
	if batch.appended {
		batch.buf.WriteString(",(")
	} else {
		batch.buf.WriteString("(")
		batch.appended = true
	}
	rv := reflect.Indirect(reflect.ValueOf(v))
	if rv.Kind() != reflect.Struct {
		return errors.New("v is not a struct or pointer to a struct")
	}
	rt := rv.Type()
	indexOf, err := warehouses.ColumnsIndex(rt)
	if err != nil {
		return err
	}
	for i, name := range batch.columns {
		if i > 0 {
			batch.buf.WriteByte(',')
		}
		index, ok := indexOf[name]
		if !ok {
			batch.err = fmt.Errorf("cannot append struct: field for column %q does not exist", name)
			return batch.err
		}
		value := rv.FieldByIndex(index)
		quoteValue(&batch.buf, value.Interface())
	}
	batch.buf.WriteString(")")
	return nil
}

// Send sends the batch to the data warehouse.
func (batch *batch) Send() error {
	if batch.err != nil {
		return batch.err
	}
	db, err := batch.warehouse.connection()
	if err != nil {
		return err
	}
	_, err = db.ExecContext(batch.ctx, batch.buf.String())
	if err != nil {
		batch.err = warehouses.WrapError(err)
		return batch.err
	}
	batch.err = errors.New("the Send method has already been called")
	return nil
}

// quoteValue quotes s as a string and writes it into b.
func quoteString(b *strings.Builder, s string) {
	if s == "" {
		b.WriteString("''")
		return
	}
	b.WriteByte('\'')
	for {
		p := strings.IndexAny(s, "\x00'")
		if p == -1 {
			p = len(s)
		}
		b.WriteString(s[:p])
		if p == len(s) {
			break
		}
		if s[p] == '\'' {
			b.WriteByte('\'')
		}
		s = s[p+1:]
		if len(s) == 0 {
			break
		}
	}
	b.WriteByte('\'')
}

// quoteValue quotes value and writes it into b.
func quoteValue(b *strings.Builder, value any) {
	if value == nil {
		b.WriteString("NULL")
		return
	}
	switch v := value.(type) {
	case bool:
		if v {
			b.WriteString("TRUE")
		}
		b.WriteString("FALSE")
	case int:
		b.WriteString(strconv.FormatInt(int64(v), 10))
	case int32:
		b.WriteString(strconv.FormatInt(int64(v), 10))
	case int64:
		b.WriteString(strconv.FormatInt(v, 10))
	case uint:
		b.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint32:
		b.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint64:
		b.WriteString(strconv.FormatUint(v, 10))
	case float32:
		b.WriteString(strconv.FormatFloat(float64(v), 'G', -1, 32))
	case float64:
		b.WriteString(strconv.FormatFloat(v, 'G', -1, 64))
	case string:
		quoteString(b, v)
	case time.Time:
		b.WriteByte('\'')
		b.WriteString(v.Format("2006-01-02 15:04:05.999999"))
		b.WriteByte('\'')
	default:
		panic(fmt.Errorf("unsupported type '%T'", v))
	}
}
