//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package postgresql

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"chichi/apis/datastore/warehouses"
	"chichi/apis/postgres"
	"chichi/connector/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

//go:embed destinations_users.sql
var createDestinationUsersTable string

//go:embed events.sql
var createEventsTable string

var _ warehouses.Warehouse = &PostgreSQL{}

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

// DestinationUser returns the external ID of the destination user of the action
// that matches with the corresponding property. If it cannot be found, then the
// empty string and false are returned.
func (warehouse *PostgreSQL) DestinationUser(ctx context.Context, action int, property string) (string, bool, error) {
	db, err := warehouse.connection()
	if err != nil {
		return "", false, err
	}
	rows, err := db.Query(ctx, `SELECT "user" FROM destinations_users WHERE action = $1 AND property = $2`, action, property)
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
	_, err = conn.Exec(ctx, createDestinationUsersTable)
	if err != nil {
		return warehouses.WrapError(err)
	}
	_, err = conn.Exec(ctx, createEventsTable)
	return warehouses.WrapError(err)
}

// Merge performs a table merge operation, handling row updates, inserts, and
// deletions. table specifies the target table for the merge operation, rows
// contains the rows to insert or update in the table, and deleted contains the
// key values of rows to delete, if they exist.
// rows or deleted can be empty but not both.
func (warehouse *PostgreSQL) Merge(ctx context.Context, table warehouses.MergeTable, rows [][]any, deleted []any) error {

	db, err := warehouse.connection()
	if err != nil {
		return err
	}

	var b strings.Builder

	// Create the temporary table.
	tempTableName := "temp_table_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	b.WriteString(`CREATE UNLOGGED TABLE "`)
	b.WriteString(tempTableName)
	b.WriteString("\" AS\n  SELECT ")
	for _, c := range table.Columns {
		b.WriteByte('"')
		b.WriteString(c.Name)
		b.WriteString(`",`)
	}
	b.WriteString(`false AS "$deleted" FROM "`)
	b.WriteString(table.Name)
	b.WriteString("\"\nWITH NO DATA")
	_, err = db.Exec(ctx, b.String())
	if err != nil {
		return warehouses.WrapError(err)
	}
	defer func() {
		_, err := warehouse.db.Exec(ctx, `DROP TABLE "`+tempTableName+`"`)
		if err != nil {
			log.Printf("cannot drop temporary table %q from data warehouse: %s", tempTableName, err)
		}
	}()

	// Copy the rows into the temporary table.
	if len(rows) > 0 {
		columnNames := make([]string, len(table.Columns))
		for i, c := range table.Columns {
			columnNames[i] = c.Name
		}
		_, err = db.CopyFrom(ctx, postgres.Identifier{tempTableName}, columnNames, postgres.CopyFromRows(rows))
		if err != nil {
			return warehouses.WrapError(err)
		}
	}

	// Copy the rows to delete into the temporary table.
	if len(deleted) > 0 {
		columnNames := make([]string, len(table.PrimaryKeys)+1)
		copy(columnNames, table.PrimaryKeys)
		columnNames[len(columnNames)-1] = "$deleted"
		rowSrc := newCopyForDeleteFrom(len(table.PrimaryKeys), deleted)
		_, err = db.CopyFrom(ctx, postgres.Identifier{tempTableName}, columnNames, rowSrc)
		if err != nil {
			return warehouses.WrapError(err)
		}
	}

	// Merge the temporary table's rows with the destination table's rows.
	b.Reset()
	b.WriteString(`MERGE INTO "`)
	b.WriteString(table.Name)
	b.WriteString("\" d\nUSING \"")
	b.WriteString(tempTableName)
	b.WriteString("\" s\nON ")
	for i, key := range table.PrimaryKeys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`d."`)
		b.WriteString(key)
		b.WriteString(`" = s."`)
		b.WriteString(key)
		b.WriteByte('"')
	}
	if len(rows) > 0 {
		b.WriteString("\nWHEN MATCHED AND s.\"$deleted\" IS NULL THEN\n  UPDATE SET ")
		i := 0
	Set:
		for _, c := range table.Columns {
			for _, key := range table.PrimaryKeys {
				if c.Name == key {
					continue Set
				}
			}
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteByte('"')
			b.WriteString(c.Name)
			b.WriteString(`" = s."`)
			b.WriteString(c.Name)
			b.WriteByte('"')
			i++
		}
		b.WriteString("\nWHEN NOT MATCHED AND s.\"$deleted\" IS NULL THEN\n  INSERT (")
		for i, c := range table.Columns {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteByte('"')
			b.WriteString(c.Name)
			b.WriteByte('"')
		}
		b.WriteString(")\n  VALUES (")
		for i, c := range table.Columns {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(`s."`)
			b.WriteString(c.Name)
			b.WriteByte('"')
		}
		b.WriteString(`)`)
	}
	if len(deleted) > 0 {
		b.WriteString("\nWHEN MATCHED THEN\n  DELETE")
	}
	_, err = db.Exec(ctx, b.String())
	if err != nil {
		return warehouses.WrapError(err)
	}

	return nil
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

// SetDestinationUser sets the destination user relative to the action, with the
// given external user ID and external property.
func (warehouse *PostgreSQL) SetDestinationUser(ctx context.Context, action int, externalUserID, externalProperty string) error {
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, "INSERT INTO destinations_users (action, \"user\", property)\n"+
		"VALUES ($1, $2, $3)\n"+
		"ON CONFLICT (action, \"user\") DO UPDATE SET property = $3",
		action, externalUserID, externalProperty)
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
			var role types.Role
			if *isUpdatable != "YES" {
				role = types.SourceRole
			}
			column := types.Property{
				Name:     row.column,
				Role:     role,
				Nullable: *isNullable == "YES",
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
	query.WriteString(`SELECT `)
	for i, c := range columns {
		if i > 0 {
			query.WriteString(", ")
		}
		if !types.IsValidPropertyName(c.Name) {
			panic(fmt.Sprintf("invalid property name: %q", c.Name))
		}
		switch c.Type.PhysicalType() {
		default:
			query.WriteByte('"')
			query.WriteString(c.Name)
			query.WriteByte('"')
		case types.PtInet:
			query.WriteString(`host("`)
			query.WriteString(c.Name)
			query.WriteString(`")`)
		}
	}
	query.WriteString(` FROM "`)
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
	values := newScanValues(columns, &rows)
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

// copyForDeleteFrom implements the pgx.CopyFromSource interface.
type copyForDeleteFrom struct {
	values []any
	row    []any
}

// newCopyForDeleteFrom returns a pgx.CopyFromSource implementation used to
// delete rows from a table. Rows are read from deleted, where each row contains
// numColumns consecutive elements from deleted and true at the end.
func newCopyForDeleteFrom(numColumns int, deleted []any) pgx.CopyFromSource {
	c := &copyForDeleteFrom{
		values: deleted,
		row:    make([]any, numColumns+1),
	}
	c.row[numColumns] = true
	return c
}

func (c *copyForDeleteFrom) Next() bool {
	return len(c.values) > 0
}

func (c *copyForDeleteFrom) Values() ([]any, error) {
	numKeys := len(c.row) - 1
	for i := 0; i < numKeys; i++ {
		c.row[i] = c.values[i]
	}
	c.values = c.values[numKeys:]
	return c.row, nil
}

func (c *copyForDeleteFrom) Err() error {
	return nil
}

// quoteString quotes s as a string and writes it into b.
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

// quoteBytes quotes s as a string and writes it into b.
func quoteBytes(b *strings.Builder, s []byte) {
	if len(s) == 0 {
		b.WriteString("''")
		return
	}
	b.WriteByte('\'')
	for {
		p := bytes.IndexAny(s, "\x00'")
		if p == -1 {
			p = len(s)
		}
		b.Write(s[:p])
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
		} else {
			b.WriteString("FALSE")
		}
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
	case []byte:
		quoteBytes(b, v)
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
