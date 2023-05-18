//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package postgresql

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"chichi/apis/postgres"
	"chichi/apis/warehouses"
	"chichi/connector/types"

	"github.com/google/uuid"
	"golang.org/x/exp/slices"
)

//go:embed connections_users.sql
var createConnectionsUsersTable string

//go:embed destinations_users.sql
var createDestinationUsersTable string

//go:embed events.sql
var createEventsTable string

var _ warehouses.Warehouse = &PostgreSQL{}
var _ warehouses.Batch = &batch{}

type PostgreSQL struct {
	mu       sync.Mutex // for the db and closed fields
	db       *postgres.DB
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

// Open opens a PostgreSQL data warehouse with the given settings.
func Open(settings []byte) (warehouses.Warehouse, error) {
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
	return &PostgreSQL{settings: &s}, nil
}

// Close closes the warehouse. It will not allow any new queries to run, and it
// waits for the current ones to finish.
func (warehouse *PostgreSQL) Close() error {
	var err error
	warehouse.mu.Lock()
	if warehouse.db != nil {
		warehouse.db.Close()
		warehouse.db = nil
		warehouse.closed = true
	}
	warehouse.mu.Unlock()
	return err
}

// DestinationUser returns the external ID of the destination user for the
// connection that matches with the corresponding property. If it cannot be
// found, then the empty string and false are returned.
func (warehouse *PostgreSQL) DestinationUser(ctx context.Context, connection int, property string) (string, bool, error) {
	db, err := warehouse.connection()
	if err != nil {
		return "", false, err
	}
	rows, err := db.Query(ctx, `SELECT "user" FROM destinations_users WHERE connection = $1 AND property = $2`, connection, property)
	if err != nil {
		return "", false, err
	}
	var externalID string
	for rows.Next() {
		if externalID != "" {
			// TODO(Gianluca): improve the handling of this error. This happens
			// when a property on the external app has the same value for more
			// than one user.
			return "", false, fmt.Errorf("too many users matching when using property")
		}
		err := rows.Scan(&externalID)
		if err != nil {
			return "", false, err
		}
	}
	if rows.Err() != nil {
		return "", false, err
	}
	return externalID, externalID != "", nil
}

// Exec executes a query without returning any rows. args are the placeholders.
// If the query fails, it returns an Error value.
func (warehouse *PostgreSQL) Exec(ctx context.Context, query string, args ...any) (warehouses.Result, error) {
	db, err := warehouse.connection()
	if err != nil {
		return warehouses.Result{}, err
	}
	r, err := db.Exec(ctx, query, args...)
	if err != nil {
		return warehouses.Result{}, warehouses.WrapError(err)
	}
	return warehouses.Result{Result: r}, nil
}

// Init initializes the data warehouse by creating the supporting tables.
func (warehouse *PostgreSQL) Init(ctx context.Context) error {
	conn, err := warehouse.connection()
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, createConnectionsUsersTable)
	if err != nil {
		return warehouses.WrapError(err)
	}
	_, err = conn.Exec(ctx, createDestinationUsersTable)
	if err != nil {
		return warehouses.WrapError(err)
	}
	_, err = conn.Exec(ctx, createEventsTable)
	return warehouses.WrapError(err)
}

// Ping checks whether the connection to the data warehouse is active and, if
// necessary, establishes a new connection.
func (warehouse *PostgreSQL) Ping(ctx context.Context) error {
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	return db.Ping(ctx)
}

// PrepareBatch creates a prepared batch statement for inserting rows in
// batch and returns it. table specifies the table in which the rows will be
// inserted, and columns specifies the columns.
func (warehouse *PostgreSQL) PrepareBatch(ctx context.Context, table string, columns []string) (warehouses.Batch, error) {
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
	batch.buf.WriteString(` ("`)
	for i, column := range columns {
		if i > 0 {
			batch.buf.WriteString(`,"`)
		}
		if !warehouses.IsValidIdentifier(column) {
			return nil, fmt.Errorf("column name %q is not a valid identifier", column)
		}
		batch.buf.WriteString(column)
		batch.buf.WriteByte('"')
	}
	batch.buf.WriteString(") VALUES ")
	return batch, nil
}

// Query executes a query that returns rows. args are the placeholders.
// If the query fails, it returns an Error value.
func (warehouse *PostgreSQL) Query(ctx context.Context, query string, args ...any) (*warehouses.Rows, error) {
	db, err := warehouse.connection()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, warehouses.WrapError(err)
	}
	return &warehouses.Rows{Rows: rows}, nil
}

// QueryRow executes a query that should return at most one row.
// If the query fails, it returns an Error value.
func (warehouse *PostgreSQL) QueryRow(ctx context.Context, query string, args ...any) warehouses.Row {
	db, err := warehouse.connection()
	if err != nil {
		return warehouses.Row{Error: err}
	}
	row := db.QueryRow(ctx, query, args...)
	return warehouses.Row{Row: row}
}

// SetDestinationUser sets the destination user in the connection with the given
// external user ID and external property.
func (warehouse *PostgreSQL) SetDestinationUser(ctx context.Context, connection int, externalUserID, externalProperty string) error {
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, "INSERT INTO destinations_users (connection, \"user\", property)\n"+
		"VALUES ($1, $2, $3)\n"+
		"ON CONFLICT (connection, \"user\") DO UPDATE SET property = $3",
		connection, externalUserID, externalProperty)
	return err
}

// Settings returns the data warehouse settings.
func (warehouse *PostgreSQL) Settings() []byte {
	s, _ := json.Marshal(warehouse.settings)
	return s
}

// Tables returns the tables of the data warehouse.
// It returns only the tables 'users', 'groups', 'events', and the tables with
// prefix 'users_', 'groups_' and 'events_'.
func (warehouse *PostgreSQL) Tables(ctx context.Context) ([]*warehouses.Table, error) {

	// Get the connection.
	db, err := warehouse.connection()
	if err != nil {
		return nil, err
	}

	var table *warehouses.Table
	var tables []*warehouses.Table

	err = db.Transaction(ctx, func(tx *postgres.Tx) error {

		// Read the available enums.
		query := "SELECT pg_type.typname, pg_enum.enumlabel FROM pg_type JOIN pg_enum ON pg_enum.enumtypid = pg_type.oid"
		rows, err := tx.Query(ctx, query)
		if err != nil {
			return err
		}
		rawEnums := map[string][]string{}
		for rows.Next() {
			var typName, enumLabel string
			if err = rows.Scan(&typName, &enumLabel); err != nil {
				rows.Close()
				return err
			}
			if typName == "" {
				rows.Close()
				return errors.New("invalid empty enum name")
			}
			if enumLabel == "" {
				rows.Close()
				return fmt.Errorf("empty enum label for type %q", typName)
			}
			if !utf8.ValidString(enumLabel) {
				rows.Close()
				return fmt.Errorf("not-valid UTF-8 encoded enum label for type %q", typName)
			}
			rawEnums[typName] = append(rawEnums[typName], enumLabel)
		}
		enums := map[string]types.Type{}
		for name, values := range rawEnums {
			enums[name] = types.Text().WithEnum(values)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		// Read the 'atttypmod' attribute of column types, where relevant.
		query = "SELECT c.relname, a.attname, a.atttypmod\n" +
			"FROM pg_attribute AS a\n" +
			"INNER JOIN pg_class AS c ON a.attrelid = c.oid\n" +
			"INNER JOIN pg_namespace AS n ON c.relnamespace = n.oid\n" +
			"WHERE n.nspname = '" + warehouse.settings.Schema + "' AND a.atttypmod <> -1"
		rows, err = tx.Query(ctx, query)
		if err != nil {
			return err
		}
		attTypMods := map[string]map[string]*int{}
		for rows.Next() {
			var relname, attname string
			var atttypmod int
			err := rows.Scan(&relname, &attname, &atttypmod)
			if err != nil {
				return err
			}
			if attTypMods[relname] == nil {
				attTypMods[relname] = map[string]*int{attname: &atttypmod}
			} else {
				attTypMods[relname][attname] = &atttypmod
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}

		// Instantiate a resolver for the composite types.
		ctResolver, err := initCompositeTypeResolver(ctx, tx, enums, attTypMods)
		if err != nil {
			return err
		}

		// Read columns.
		query = "SELECT c.table_name, c.column_name, c.is_nullable, c.data_type, c.udt_name, c.character_maximum_length," +
			" c.numeric_precision, c.numeric_precision_radix, c.numeric_scale, c.is_updatable," +
			" col_description(c.table_name::regclass::oid, c.ordinal_position)\n" +
			"FROM information_schema.columns c\n" +
			"INNER JOIN information_schema.tables t ON c.table_name = t.table_name AND c.table_schema = t.table_schema\n" +
			"WHERE t.table_schema = '" + warehouse.settings.Schema + "' AND t.table_type = 'BASE TABLE' AND" +
			" ( t.table_name IN ('users', 'groups', 'events') OR t.table_name LIKE 'users\\__%' OR" +
			" t.table_name LIKE 'groups\\__%' OR t.table_name LIKE 'events\\__%' )\n" +
			"ORDER BY c.table_name, c.ordinal_position"

		rows, err = tx.Query(ctx, query)
		if err != nil {
			return err
		}
		for rows.Next() {
			var row pgTypeInfo
			var tableName, columnName, dataType, udtName, isNullable, isUpdatable, description *string
			if err = rows.Scan(&tableName, &columnName, &isNullable, &dataType,
				&udtName, &row.charLength, &row.precision, &row.radix, &row.scale, &isUpdatable, &description); err != nil {
				rows.Close()
				return err
			}
			if tableName == nil {
				return errors.New("data warehouse has returned NULL as table name")
			}
			row.table = *tableName
			if columnName == nil {
				return errors.New("data warehouse has returned NULL as column name")
			}
			if !types.IsValidPropertyName(*columnName) {
				return fmt.Errorf("column name %q is not supported", *columnName)
			}
			row.column = *columnName
			if isNullable == nil {
				return errors.New("data warehouse has returned NULL as nullability of column")
			}
			if dataType == nil {
				return errors.New("data warehouse has returned NULL as column data type")
			}
			row.dataType = *dataType
			if udtName == nil {
				return errors.New("data warehouse has returned NULL as column udt name")
			}
			row.udtName = *udtName
			if isUpdatable == nil {
				return errors.New("data warehouse has returned NULL as updatability of column")
			}
			column := &warehouses.Column{
				Name:        row.column,
				IsUpdatable: *isUpdatable == "YES",
				Nullable:    *isNullable == "YES",
			}
			column.Type, err = columnType(row, enums, ctResolver, attTypMods)
			if err != nil {
				return fmt.Errorf("data warehouse has returned an invalid type: %s", err)
			}
			if !column.Type.Valid() {
				return fmt.Errorf("type of column %s.%s is not supported", row.table, column.Name)
			}
			if description != nil {
				column.Description = *description
			}
			if table == nil || row.table != table.Name {
				table = &warehouses.Table{Name: row.table}
				tables = append(tables, table)
			}
			table.Columns = append(table.Columns, column)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, warehouses.WrapError(err)
	}

	return tables, nil
}

// pgTypeInfo holds information about a PostgreSQL type, as read from the
// PostgreSQL information tables (as 'information_schema.columns' and
// 'information_schema.attributes').
type pgTypeInfo struct {
	table      string
	column     string
	dataType   string
	udtName    string
	charLength *string
	precision  *string
	radix      *string
	scale      *string
}

// Select returns the rows from the given table that satisfies the where
// condition with only the given columns, ordered by order if order is not the
// zero Property, and in range [first,first+limit] with first >= 0 and
// 0 < limit <= 1000.
//
// If a query to the warehouse fails, it returns an Error value.
// If an argument is not valid, it panics.
func (warehouse *PostgreSQL) Select(ctx context.Context, table string, columns []types.Property, where warehouses.Where, order types.Property, first, limit int) ([][]any, error) {

	if !warehouses.IsValidIdentifier(table) {
		return nil, fmt.Errorf("table name %q is not a valid identifier", table)
	}

	db, err := warehouse.connection()
	if err != nil {
		return nil, err
	}

	// Build the query.
	var query strings.Builder
	query.WriteString(`SELECT "`)
	for i, c := range columns {
		if i > 0 {
			query.WriteString(`", "`)
		}
		if !types.IsValidPropertyName(c.Name) {
			panic(fmt.Sprintf("invalid property name: %q", c.Name))
		}
		query.WriteString(c.Name)
	}
	query.WriteString(`" FROM "`)
	query.WriteString(table)
	query.WriteByte('"')

	if where != nil {
		query.WriteString(` WHERE `)
		expr, err := renderExpr(where)
		if err != nil {
			return nil, fmt.Errorf("cannot build WHERE expression: %s", err)
		}
		query.WriteString(expr)
	}

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
	rawRows, err := db.Query(ctx, query.String())
	if err != nil {
		return nil, warehouses.WrapError(err)
	}
	var rows [][]any
	values := warehouses.NewScanValues(columns, &rows)
	for rawRows.Next() {
		if err = rawRows.Scan(values...); err != nil {
			rawRows.Close()
			return nil, warehouses.WrapError(err)
		}
	}
	if err = rawRows.Err(); err != nil {
		return nil, warehouses.WrapError(err)
	}
	rawRows.Close()
	if rows == nil {
		rows = [][]any{}
	}

	return rows, nil
}

// connection returns the database connection.
func (warehouse *PostgreSQL) connection() (*postgres.DB, error) {
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
	db, err := postgres.Open(warehouse.settings.options())
	if err != nil {
		return nil, warehouses.WrapError(err)
	}
	warehouse.db = db
	return db, nil
}

// dsn returns the connection string, from s, in the URL format.
func (s *psSettings) options() *postgres.Options {
	return &postgres.Options{
		Host:     s.Host,
		Port:     s.Port,
		Username: s.Username,
		Password: s.Password,
		Database: s.Database,
	}
}

// batch implements the Batch interface.
type batch struct {
	warehouse *PostgreSQL
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
	_, err = db.Exec(batch.ctx, batch.buf.String())
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
	case int16:
		b.WriteString(strconv.FormatInt(int64(v), 10))
	case int32:
		b.WriteString(strconv.FormatInt(int64(v), 10))
	case int64:
		b.WriteString(strconv.FormatInt(v, 10))
	case uint:
		b.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint16:
		b.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint32:
		b.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint64:
		b.WriteString(strconv.FormatUint(v, 10))
	case float32:
		b.WriteString(strconv.FormatFloat(float64(v), 'G', -1, 32))
	case float64:
		b.WriteString(strconv.FormatFloat(v, 'G', -1, 64))
	case netip.Addr:
		quoteString(b, v.String())
	case string:
		quoteString(b, v)
	case time.Time:
		b.WriteByte('\'')
		b.WriteString(v.Format("2006-01-02 15:04:05.999999"))
		b.WriteByte('\'')
	case uuid.UUID:
		b.WriteString(v.String())
	default:
		panic(fmt.Errorf("unsupported type '%T'", v))
	}
}
