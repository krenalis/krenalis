//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package clickhouse

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata" // workaround for clickhouse-go issue #162

	"chichi/apis/types"
	"chichi/apis/warehouses"

	"github.com/ClickHouse/clickhouse-go/v2"
	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/slices"
)

//go:embed connections_users.sql
var createConnectionsUsersTable string

var _ warehouses.Warehouse = &ClickHouse{}
var _ warehouses.Batch = &batch{}

type ClickHouse struct {
	mu       sync.Mutex // for the conn and closed fields
	conn     chDriver.Conn
	closed   bool
	settings *chSettings
	err      error
}

type chSettings struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

// Open opens a ClickHouse data warehouse with the given settings.
func Open(settings []byte) (warehouses.Warehouse, error) {
	var s chSettings
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
	if s.Username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}
	// Validate Database.
	if s.Database == "" {
		return nil, fmt.Errorf("database cannot be empty")
	}
	return &ClickHouse{settings: &s}, nil
}

// Close closes the warehouse. It will not allow any new queries to run, and it
// waits for the current ones to finish.
func (warehouse *ClickHouse) Close() error {
	var err error
	warehouse.mu.Lock()
	if warehouse.conn != nil {
		err = warehouse.conn.Close()
		warehouse.conn = nil
		warehouse.closed = true
	}
	warehouse.mu.Unlock()
	return err
}

// Exec executes a query without returning any rows. args are the placeholders.
// If the query fails, it returns an Error value.
func (warehouse *ClickHouse) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}

// Init initializes the data warehouse by creating the supporting tables.
func (warehouse *ClickHouse) Init(ctx context.Context) error {
	conn, err := warehouse.connection()
	if err != nil {
		return err
	}
	err = conn.Exec(ctx, createConnectionsUsersTable)
	return warehouses.WrapError(err)
}

// Ping checks whether the connection to the data warehouse is active and, if
// necessary, establishes a new connection.
func (warehouse *ClickHouse) Ping(ctx context.Context) error {
	conn, err := warehouse.connection()
	if err != nil {
		return err
	}
	return conn.Ping(ctx)
}

// PrepareBatch creates a prepared batch statement for inserting rows in
// batch and returns it. table specifies the table in which the rows will be
// inserted, and columns specifies the columns.
func (warehouse *ClickHouse) PrepareBatch(ctx context.Context, table string, columns []string) (warehouses.Batch, error) {
	if !warehouses.IsValidIdentifier(table) {
		return nil, fmt.Errorf("table name %q is not a valid identifier", table)
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("columns cannot be empty")
	}
	var b strings.Builder
	b.WriteString("INSERT INTO ")
	b.WriteString(table)
	b.WriteString(" (")
	for i, column := range columns {
		if i > 0 {
			b.WriteByte(',')
		}
		if !warehouses.IsValidIdentifier(column) {
			return nil, fmt.Errorf("column name %q is not a valid identifier", column)
		}
		b.WriteString(column)
	}
	b.WriteString(") ")
	batch := &batch{columns: slices.Clone(columns)}
	var err error
	batch.batch, err = warehouse.conn.PrepareBatch(ctx, b.String())
	if err != nil {
		return nil, err
	}
	return batch, nil
}

// Tables returns the tables of the data warehouse.
// It returns only the tables 'users', 'groups', 'events', and the tables with
// prefix 'users_', 'groups_' and 'events_'. Also, it does not return columns
// starting with an underscore.
func (warehouse *ClickHouse) Tables(ctx context.Context) ([]*warehouses.Table, error) {

	// Get the connection.
	conn, err := warehouse.connection()
	if err != nil {
		return nil, err
	}

	// Read columns.
	query := "SELECT c.table_name, c.column_name, c.data_type, c.column_comment\n" +
		"FROM information_schema.columns c\n" +
		"INNER JOIN information_schema.tables t ON c.table_name = t.table_name AND c.table_schema = t.table_schema\n" +
		"WHERE t.table_schema = '" + warehouse.settings.Database + "' AND t.table_type = 'BASE TABLE' AND" +
		" ( t.table_name IN ('users', 'groups', 'events') OR t.table_name LIKE 'users\\__%' OR" +
		" t.table_name LIKE 'groups\\__%' OR t.table_name LIKE 'events\\__%' ) AND" +
		" NOT startsWith(c.column_name, '_')\n" +
		"ORDER BY c.table_name, c.ordinal_position"

	var table *warehouses.Table
	var tables []*warehouses.Table

	rows, err := conn.Query(ctx, query)
	for rows.Next() {
		var tableName, columnName, typ, comment string
		if err = rows.Scan(&tableName, &columnName, &typ, &comment); err != nil {
			return nil, warehouses.WrapError(err)
		}
		if !types.IsValidPropertyName(columnName) {
			return nil, warehouses.NewError("column name %q is not supported", columnName)
		}
		column := &warehouses.Column{
			Name:        columnName,
			Description: comment,
		}
		column.Type = propertyType(typ)
		if !column.Type.Valid() {
			return nil, fmt.Errorf("type %q of column %s is not supported", typ, column.Name)
		}
		if table == nil || tableName != table.Name {
			table = &warehouses.Table{Name: tableName}
			tables = append(tables, table)
		}
		table.Columns = append(table.Columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, warehouses.WrapError(err)
	}

	return tables, nil
}

// Query executes a query that returns rows. args are the placeholders.
// If the query fails, it returns an Error value.
func (warehouse *ClickHouse) Query(ctx context.Context, query string, args ...any) (*warehouses.Rows, error) {
	return nil, nil
}

// QueryRow executes a query that should return at most one row.
func (warehouse *ClickHouse) QueryRow(ctx context.Context, query string, args ...any) warehouses.Row {
	return warehouses.Row{}
}

// Settings returns the data warehouse settings.
func (warehouse *ClickHouse) Settings() []byte {
	s, _ := json.Marshal(warehouse.settings)
	return s
}

// Users returns the users, with only the properties in schema, ordered by
// order if order is not the zero Property, and in range [first,first+limit]
// with first >= 0 and 0 < limit <= 1000.
//
// If a query to the warehouse fails, it returns an Error value.
// If an argument is not valid, it panics.
func (warehouse *ClickHouse) Users(ctx context.Context, schema types.Type, order types.Property, first, limit int) ([][]any, error) {

	conn, err := warehouse.connection()
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
	rows, err := conn.Query(ctx, query.String())
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
func (warehouse *ClickHouse) connection() (clickhouse.Conn, error) {
	warehouse.mu.Lock()
	defer warehouse.mu.Unlock()
	if warehouse.closed {
		return nil, warehouses.WrapError(errors.New("warehouse is closed"))
	}
	if warehouse.settings == nil {
		return nil, warehouses.WrapError(errors.New("there are no settings"))
	}
	if warehouse.conn != nil {
		return warehouse.conn, nil
	}
	conn, err := clickhouse.Open(warehouse.settings.options())
	if err != nil {
		return nil, warehouses.WrapError(err)
	}
	warehouse.conn = conn
	return conn, nil
}

// options returns s as a *clickhouse.Options value.
func (s *chSettings) options() *clickhouse.Options {
	return &clickhouse.Options{
		Addr: []string{net.JoinHostPort(s.Host, strconv.Itoa(s.Port))},
		Auth: clickhouse.Auth{
			Database: s.Database,
			Username: s.Username,
			Password: s.Password,
		},
	}
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func (s *chSettings) testConnection(ctx context.Context) error {
	conn, err := clickhouse.Open(s.options())
	if err != nil {
		return err
	}
	err = conn.Ping(ctx)
	if err != nil {
		return err
	}
	return conn.Close()
}

// batch implements the Batch interface.
type batch struct {
	warehouse *ClickHouse
	columns   []string
	batch     chDriver.Batch
	err       error
}

// Abort aborts the execution of the batch insert.
func (batch *batch) Abort() error {
	if batch.err != nil {
		return batch.err
	}
	err := batch.batch.Abort()
	if err != nil {
		batch.err = warehouses.WrapError(err)
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
	if err := batch.batch.Append(v...); err != nil {
		batch.err = warehouses.WrapError(err)
	}
	return batch.err
}

// AppendStruct appends the values of a row, read from the fields of the struct
// v, to batch. It returns an error if v is not a struct or pointer to a struct.
func (batch *batch) AppendStruct(v any) error {
	if batch.err != nil {
		return batch.err
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
	values := make([]any, len(batch.columns))
	for i, name := range batch.columns {
		index, ok := indexOf[name]
		if !ok {
			batch.err = fmt.Errorf("cannot append struct: field for column %q does not exist", name)
			return batch.err
		}
		value := rv.FieldByIndex(index)
		values[i] = value.Interface()
	}
	if err := batch.batch.Append(values...); err != nil {
		batch.err = warehouses.WrapError(err)
	}
	return batch.err
}

// Send sends the batch to the data warehouse.
func (batch *batch) Send() error {
	if batch.err != nil {
		return batch.err
	}
	if err := batch.batch.Send(); err != nil {
		batch.err = warehouses.WrapError(err)
		return batch.err
	}
	batch.err = errors.New("the Send method has already been called")
	return nil
}
