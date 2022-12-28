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
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"chichi/apis/types"
	"github.com/shopspring/decimal"
)

//go:embed connections_users.sql
var createConnectionsUsersTable string

var _ Warehouse = &postgreSQL{}

type postgreSQL struct {
	mu       sync.Mutex // for the db and closed fields
	db       *sql.DB
	closed   bool
	settings *PostgreSQLSettings
}

type PostgreSQLSettings struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Schema   string
}

// OpenPostgres opens a PostgreSQL data warehouse with the given settings.
func OpenPostgres(settings *PostgreSQLSettings) *postgreSQL {
	return &postgreSQL{settings: settings}
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
		return wrapError(err)
	}
	// Create the tables.
	for i := len(createTables) - 1; i >= 0; i-- {
		_, err = tx.ExecContext(ctx, createTables[i])
		if err != nil {
			_ = tx.Rollback()
			return wrapError(err)
		}
	}
	// Create the "connections_users" table.
	_, err = tx.ExecContext(ctx, createConnectionsUsersTable)
	if err != nil {
		_ = tx.Rollback()
		return wrapError(err)
	}
	err = tx.Commit()
	return wrapError(err)
}

// DropTables drops the data warehouse tables created from the given schema. It
// does not return an error if a table does not exist.
func (warehouse *postgreSQL) DropTables(ctx context.Context, schema types.Type) error {
	tables := []string{"connections_users", "users"}
	for _, p := range schema.Properties() {
		if isArrayOfObjects(p.Type) {
			if !types.IsValidPropertyName(p.Name) {
				panic("property name is not valid")
			}
			tables = append(tables, tablesOfObject("users_"+p.Name, p.Type.ItemType())...)
		}
	}
	var b strings.Builder
	b.WriteString(`DROP TABLE IF EXISTS "`)
	for i, table := range tables {
		if i > 0 {
			b.WriteString(`", "`)
		}
		b.WriteString(table)
	}
	b.WriteByte('"')
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, b.String())
	return wrapError(err)
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
		return nil, wrapError(err)
	}
	return result{r}, nil
}

// Type returns the type of the warehouse.
func (warehouse *postgreSQL) Type() Type {
	return PostgreSQL
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

// Query executes a query that returns rows. args are the placeholders.
// If the query fails, it returns an Error value.
func (warehouse *postgreSQL) Query(ctx context.Context, query string, args ...any) (*Rows, error) {
	db, err := warehouse.connection()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, wrapError(err)
	}
	return &Rows{rows}, nil
}

// QueryRow executes a query that should return at most one row.
// If the query fails, it returns an Error value.
func (warehouse *postgreSQL) QueryRow(ctx context.Context, query string, args ...any) Row {
	db, err := warehouse.connection()
	if err != nil {
		return Row{err: err}
	}
	row := db.QueryRowContext(ctx, query, args...)
	return Row{row: row}
}

// TableNames returns the names of the tables in the warehouse.
func (warehouse *postgreSQL) TableNames(ctx context.Context) ([]string, error) {
	rows, err := warehouse.db.QueryContext(ctx,
		`SELECT tablename FROM pg_tables WHERE schemaname = "`+warehouse.settings.Schema+`"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var name string
	var names []string
	for rows.Next() {
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		if isValidTableName(name) {
			names = append(names, name)
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

// TableSchema returns the schema of the table called name.
func (warehouse *postgreSQL) TableSchema(ctx context.Context, name string) (types.Type, error) {
	if !types.IsValidPropertyName(name) {
		panic("table name is not valid")
	}
	db, err := warehouse.connection()
	if err != nil {
		return types.Type{}, err
	}
	query := "SELECT column_name, data_type, character_maximum_length, numeric_precision," +
		" numeric_precision_radix, numeric_scale\n" +
		"FROM information_schema.columns" +
		`WHERE table_name = "` + name + `"` +
		"ORDER BY ordinal_position"
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return types.Type{}, wrapError(err)
	}
	var properties []types.Property
	for rows.Next() {
		var name, typ, charLength, precision, precisionRadix, scale sql.NullString
		if err := rows.Scan(&name, &typ, &charLength, &precision, &precisionRadix, &scale); err != nil {
			_ = rows.Close()
			return types.Type{}, wrapError(err)
		}
		if !name.Valid {
			return types.Type{}, wrapError(fmt.Errorf("data warehouse has returned NULL as column name"))
		}
		if !typ.Valid {
			return types.Type{}, wrapError(fmt.Errorf("data warehouse has returned NULL as column data type"))
		}
		if !types.IsValidPropertyName(name.String) {
			return types.Type{}, fmt.Errorf("column name %q is not supported", name.String)
		}
		property := types.Property{Name: name.String}
		switch typ.String {
		case "smallint":
			property.Type = types.Int16()
		case "integer":
			property.Type = types.Int()
		case "bigint":
			property.Type = types.Int64()
		case "numeric":
			// Parse precision radix.
			if !precisionRadix.Valid {
				return types.Type{}, wrapError(fmt.Errorf(
					"data warehouse has returned NULL as precision radix for column %s", name.String))
			}
			radix, _ := strconv.Atoi(precisionRadix.String)
			if radix != 2 && radix != 10 {
				return types.Type{}, wrapError(fmt.Errorf(
					"data warehouse has returned an invalid precision radix for column %s", name.String))
			}
			// Parse precision.
			if !precision.Valid {
				return types.Type{}, wrapError(fmt.Errorf(
					"data warehouse has returned NULL as precision for column %s", name.String))
			}
			p, err := strconv.ParseInt(precision.String, radix, 64)
			if err != nil || p < 1 {
				return types.Type{}, wrapError(fmt.Errorf(
					"data warehouse has returned an invalid precision for column %s: %s", name.String, precision.String))
			}
			if p > types.MaxDecimalPrecision {
				return types.Type{}, fmt.Errorf("numeric precision of column %s is not supported: %d", name.String, p)
			}
			// Parse scale.
			if !scale.Valid {
				return types.Type{}, wrapError(fmt.Errorf(
					"data warehouse has returned NULL as scale for column %s", name.String))
			}
			s, err := strconv.ParseInt(scale.String, radix, 64)
			if err != nil || s < 0 || s > p {
				return types.Type{}, wrapError(fmt.Errorf(
					"data warehouse has returned an invalid scale for column %s: %s", name.String, scale.String))
			}
			if s > types.MaxDecimalScale {
				return types.Type{}, fmt.Errorf("numeric scale of column %s is not supported: %d", name.String, s)
			}
			property.Type = types.Decimal(int(p), int(s))
		case "real":
			property.Type = types.Float32()
		case "double precision":
			property.Type = types.Float()
		case "varchar", "char", "text":
			if charLength.Valid {
				chars, _ := strconv.Atoi(charLength.String)
				if chars < 1 {
					return types.Type{}, wrapError(fmt.Errorf(
						"data warehouse has returned an invalid character maximum length for column %s", name.String))
				}
				property.Type = types.Text(types.Chars(chars))
			} else {
				if typ.String == "char" {
					property.Type = types.Text(types.Chars(1))
				} else {
					property.Type = types.Text()
				}
			}
		case "timestamp", "timestamp with time zone":
			property.Type = types.DateTime("2006-01-02 15:04:05.999999")
		case "date":
			property.Type = types.Date("2006-01-02")
		case "time", "time with time zone":
			property.Type = types.Time("15:04:05")
		case "boolean":
			property.Type = types.Boolean()
		case "uuid":
			property.Type = types.UUID()
		case "jsonb":
			property.Type = types.JSON()
		}
		properties = append(properties, property)
	}
	if err := rows.Err(); err != nil {
		return types.Type{}, wrapError(err)
	}
	schema := types.Object(properties)
	return schema, nil
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
		return nil, wrapError(err)
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
			return nil, wrapError(err)
		}
		users = append(users, user)
	}
	if err = rows.Err(); err != nil {
		return nil, wrapError(err)
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

// Validate validates the settings and returns an error if they are not valid.
func (warehouse *postgreSQL) Validate() error {
	s := warehouse.settings
	// Validate Host.
	if n := len(s.Host); n == 0 || n > 253 {
		return newError("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65536 {
		return newError("port must be in range [1,65536]")
	}
	// Validate Username.
	if n := len(s.Username); n < 1 || n > 63 {
		return newError("username length in bytes must be in range [1,63]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 100 {
		return newError("password length must be in range [1,100]")
	}
	// Validate Database.
	if n := len(s.Database); n < 1 || n > 63 {
		return newError("database length in bytes must be in range [1,63]")
	}
	// Validate Schema.
	if n := len(s.Schema); n < 1 || n > 63 {
		return newError("schema length in bytes must be in range [1,63]")
	}
	if !isValidSchemaName(s.Schema) {
		return newError("schema must start with [A-Za-z_] and subsequently contain only [A-Za-z0-9_]")
	}
	if strings.HasPrefix(s.Schema, "pg_") {
		return newError("schema cannot start with 'pg_'")
	}
	return nil
}

// connection returns the database connection.
func (warehouse *postgreSQL) connection() (*sql.DB, error) {
	warehouse.mu.Lock()
	defer warehouse.mu.Unlock()
	if warehouse.closed {
		return nil, wrapError(errors.New("warehouse is closed"))
	}
	if warehouse.settings == nil {
		return nil, wrapError(errors.New("there are no settings"))
	}
	if warehouse.db != nil {
		return warehouse.db, nil
	}
	db, err := sql.Open("pgx", warehouse.settings.dsn())
	if err != nil {
		return nil, wrapError(err)
	}
	warehouse.db = db
	return db, nil
}

// dsn returns the connection string, from s, in the URL format.
func (s *PostgreSQLSettings) dsn() string {
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
func (s *PostgreSQLSettings) testConnection(ctx context.Context) error {
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

// tablesOfObject returns table names from the type t.
func tablesOfObject(table string, t types.Type) []string {
	tables := []string{table}
	for _, p := range t.Properties() {
		if isArrayOfObjects(p.Type) {
			if !types.IsValidPropertyName(p.Name) {
				panic("property name is not valid")
			}
			tables = append(tables, tablesOfObject(table+"_"+p.Name, p.Type.ItemType())...)
		}
	}
	return tables
}
