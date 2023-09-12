//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package snowflake

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"chichi/apis/datastore/warehouses"
	"chichi/connector/types"

	"github.com/shopspring/decimal"
	"github.com/snowflakedb/gosnowflake"
)

var _ warehouses.Warehouse = &Snowflake{}

type Snowflake struct {
	mu       sync.Mutex // for the db and closed fields
	db       *sql.DB
	closed   bool
	settings *sfSettings
}

type sfSettings struct {
	Username  string
	Password  string
	Account   string
	Warehouse string
	Database  string
	Schema    string
	Role      string
}

// Open opens a Snowflake data warehouse with the given settings.
// It returns a SettingsError error if the settings are not valid.
func Open(settings []byte) (warehouses.Warehouse, error) {
	var s sfSettings
	err := json.Unmarshal(settings, &s)
	if err != nil {
		return nil, warehouses.SettingsErrorf("cannot unmarshal settings: %s", err)
	}
	// Validate Account.
	if n := utf8.RuneCountInString(s.Account); n < 1 || n > 255 {
		return nil, warehouses.SettingsErrorf("account length must be in range [1,255]")
	}
	// Validate Username.
	if n := utf8.RuneCountInString(s.Username); n < 1 || n > 255 {
		return nil, warehouses.SettingsErrorf("username length must be in range [1,255]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 255 {
		return nil, warehouses.SettingsErrorf("password length must be in range [1,255]")
	}
	// Validate Database.
	if n := utf8.RuneCountInString(s.Database); n < 1 || n > 255 {
		return nil, warehouses.SettingsErrorf("database length must be in range [1,255]")
	}
	// Validate Schema.
	if n := utf8.RuneCountInString(s.Schema); n < 1 || n > 255 {
		return nil, warehouses.SettingsErrorf("schema length must be in range [1,255]")
	}
	// Validate Warehouse.
	if n := utf8.RuneCountInString(s.Warehouse); n < 1 || n > 255 {
		return nil, warehouses.SettingsErrorf("warehouse length must be in range [1,255]")
	}
	// Validate Role.
	if n := utf8.RuneCountInString(s.Role); n < 1 || n > 255 {
		return nil, warehouses.SettingsErrorf("role length must be in range [1,255]")
	}
	return &Snowflake{settings: &s}, nil
}

// Close closes the warehouse. It will not allow any new queries to run, and it
// waits for the current ones to finish.
func (warehouse *Snowflake) Close() error {
	var err error
	warehouse.mu.Lock()
	if warehouse.db != nil {
		err = warehouse.db.Close()
		warehouse.db = nil
		warehouse.closed = true
	}
	warehouse.mu.Unlock()
	if err != nil {
		return warehouses.Error(err)
	}
	return nil
}

// DestinationUser returns the external ID of the destination user of the action
// that matches with the corresponding property. If it cannot be found, then the
// empty string and false are returned.
func (warehouse *Snowflake) DestinationUser(ctx context.Context, action int, property string) (string, bool, error) {
	return "", false, errors.New("not implemented")
}

// Exec executes a query without returning any rows. args are the placeholders.
func (warehouse *Snowflake) Exec(ctx context.Context, query string, args ...any) (warehouses.Result, error) {
	return warehouses.Result{}, errors.New("not implemented")
}

// Init initializes the data warehouse by creating the supporting tables.
func (warehouse *Snowflake) Init(ctx context.Context) error {
	return nil
}

// Merge performs a table merge operation, handling row updates, inserts, and
// deletions. table specifies the target table for the merge operation, rows
// contains the rows to insert or update in the table, and deleted contains the
// key values of rows to delete, if they exist.
// rows or deleted can be empty but not both.
func (warehouse *Snowflake) Merge(ctx context.Context, table warehouses.MergeTable, rows [][]any, deleted []any) error {

	db, err := warehouse.connection()
	if err != nil {
		return err
	}

	// Serialize the rows in CSV format.
	var rowsCSV io.Reader
	if len(rows) > 0 {
		var err error
		rowsCSV, err = serializeRowsToCSV(table.Columns, rows, false)
		if err != nil {
			return err
		}
	}

	// Serialize the deleted rows in CSV format.
	var deletedCSV io.Reader
	if len(deleted) > 0 {
		n := len(table.PrimaryKeys)
		rows := make([][]any, 0, len(deleted)/n)
		for i := 0; i < len(deleted); i += n {
			rows = append(rows, deleted[i:i+n])
		}
		var err error
		deletedCSV, err = serializeRowsToCSV(table.PrimaryKeys, rows, true)
		if err != nil {
			return err
		}
	}

	// Create the temporary table.
	tempTableName := "temp_table_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	var q strings.Builder
	q.WriteString(`CREATE TEMPORARY TABLE "`)
	q.WriteString(tempTableName)
	q.WriteString("\" (\n")
	for _, c := range table.Columns {
		q.WriteByte('"')
		q.WriteString(c.Name)
		q.WriteString(`" `)
		switch c.Type.PhysicalType() {
		case types.PtBoolean:
			q.WriteString("BOOLEAN")
		case types.PtFloat, types.PtFloat32:
			q.WriteString("FLOAT")
		case
			types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64,
			types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64,
			types.PtYear:
			q.WriteString("NUMBER(38,0)")
		case types.PtDecimal:
			q.WriteString("NUMBER(")
			q.WriteString(strconv.Itoa(c.Type.Precision()))
			q.WriteByte(',')
			q.WriteString(strconv.Itoa(c.Type.Scale()))
			q.WriteByte(')')
		case types.PtDateTime:
			q.WriteString("TIMESTAMP_NTZ")
		case types.PtDate:
			q.WriteString("DATE")
		case types.PtTime:
			q.WriteString("TIME")
		case types.PtJSON:
			q.WriteString("VARIANT")
		case types.PtUUID:
			q.WriteString("VARCHAR(36)")
		case types.PtInet:
			q.WriteString("VARCHAR(45)")
		case types.PtText:
			q.WriteString("VARCHAR")
			if l, ok := c.Type.CharLen(); ok {
				q.WriteByte('(')
				q.WriteString(strconv.Itoa(l))
				q.WriteByte(')')
			}
		case types.PtArray:
			q.WriteString("ARRAY")
		case types.PtMap:
			q.WriteString("OBJECT")
		default:
			panic("unsupported type")
		}
		if !c.Nullable {
			q.WriteString(" NOT NULL")
		}
		q.WriteString(",\n")
	}
	q.WriteString("\"$deleted\" BOOLEAN NOT NULL\n)")

	conn, err := db.Conn(ctx)
	if err != nil {
		return warehouses.Error(err)
	}
	defer conn.Close()

	_, err = conn.ExecContext(ctx, q.String())
	if err != nil {
		return warehouses.Error(err)
	}
	defer func() {
		_, _ = conn.ExecContext(ctx, `DROP TABLE "`+tempTableName+`"`)
	}()

	// Copy the rows into the temporary table.
	if len(rows) > 0 {
		// Put the rows into the temporary table's stage.
		_, err = conn.ExecContext(gosnowflake.WithFileStream(ctx, rowsCSV), `PUT file://rows.csv @%"`+tempTableName+`"`)
		if err != nil {
			return warehouses.Error(err)
		}
		// Copy the rows from the stage into the temporary table.
		q.Reset()
		q.WriteString("COPY INTO \"")
		q.WriteString(tempTableName)
		q.WriteString("\"\nFROM @%\"")
		q.WriteString(tempTableName)
		q.WriteString("\"\nFILE_FORMAT = (TYPE=CSV PARSE_HEADER=TRUE FIELD_OPTIONALLY_ENCLOSED_BY='0x27' ESCAPE_UNENCLOSED_FIELD=NONE EMPTY_FIELD_AS_NULL=TRUE NULL_IF=())\n" +
			"FILES = ('rows.csv.gz')\n" +
			"MATCH_BY_COLUMN_NAME = CASE_SENSITIVE\n" +
			"ON_ERROR = ABORT_STATEMENT")
		_, err = conn.ExecContext(ctx, q.String())
		if err != nil {
			return warehouses.Error(err)
		}
	}

	// Copy the deleted rows into the temporary table.
	if len(deleted) > 0 {
		// Put the deleted rows into the temporary table's stage.
		_, err = conn.ExecContext(gosnowflake.WithFileStream(ctx, deletedCSV), `PUT file://rows.csv @%"`+tempTableName+`"`)
		if err != nil {
			return warehouses.Error(err)
		}
		// Copy the deleted rows from the stage into the temporary table.
		q.Reset()
		q.WriteString("COPY INTO \"")
		q.WriteString(tempTableName)
		q.WriteString("\"\nFROM @%\"")
		q.WriteString(tempTableName)
		q.WriteString("\"\nFILE_FORMAT = (TYPE=CSV PARSE_HEADER=TRUE FIELD_OPTIONALLY_ENCLOSED_BY='0x27' ESCAPE_UNENCLOSED_FIELD=NONE EMPTY_FIELD_AS_NULL=TRUE NULL_IF=())\n" +
			"FILES = ('rows.csv.gz')\n" +
			"MATCH_BY_COLUMN_NAME = CASE_SENSITIVE\n" +
			"OVERWRITE = TRUE\n" +
			"ON_ERROR = ABORT_STATEMENT")
		_, err = conn.ExecContext(ctx, q.String())
		if err != nil {
			return warehouses.Error(err)
		}
	}

	// Merge the temporary table's rows with the destination table's rows.
	q.Reset()
	q.WriteString(`MERGE INTO `)
	q.WriteString(quoteTable(table.Name))
	q.WriteString(" d\nUSING \"")
	q.WriteString(tempTableName)
	q.WriteString("\" s\nON ")
	for i, key := range table.PrimaryKeys {
		if i > 0 {
			q.WriteByte(',')
		}
		q.WriteString(`d."`)
		q.WriteString(key.Name)
		q.WriteString(`" = s."`)
		q.WriteString(key.Name)
		q.WriteByte('"')
	}
	if len(rows) > 0 {
		q.WriteString("\nWHEN MATCHED AND NOT s.\"$deleted\" THEN\n  UPDATE SET ")
		i := 0
	Set:
		for _, c := range table.Columns {
			for _, key := range table.PrimaryKeys {
				if c.Name == key.Name {
					continue Set
				}
			}
			if i > 0 {
				q.WriteByte(',')
			}
			q.WriteByte('"')
			q.WriteString(c.Name)
			q.WriteString(`" = s."`)
			q.WriteString(c.Name)
			q.WriteByte('"')
			i++
		}
		q.WriteString("\nWHEN NOT MATCHED AND NOT s.\"$deleted\" THEN\n  INSERT (")
		for i, c := range table.Columns {
			if i > 0 {
				q.WriteByte(',')
			}
			q.WriteByte('"')
			q.WriteString(c.Name)
			q.WriteByte('"')
		}
		q.WriteString(")\n  VALUES (")
		for i, c := range table.Columns {
			if i > 0 {
				q.WriteByte(',')
			}
			q.WriteString(`s."`)
			q.WriteString(c.Name)
			q.WriteByte('"')
		}
		q.WriteString(`)`)
	}
	if len(deleted) > 0 {
		q.WriteString("\nWHEN MATCHED THEN\n  DELETE")
	}
	_, err = conn.ExecContext(ctx, q.String())
	if err != nil {
		return warehouses.Error(err)
	}

	return nil
}

// Ping checks whether the connection to the data warehouse is active and, if
// necessary, establishes a new connection.
func (warehouse *Snowflake) Ping(ctx context.Context) error {
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	err = db.PingContext(ctx)
	if err != nil {
		return warehouses.Error(err)
	}
	return nil
}

// QueryRow executes a query that should return at most one row.
func (warehouse *Snowflake) QueryRow(ctx context.Context, query string, args ...any) warehouses.Row {
	return warehouses.Row{Error: errors.New("not implemented")}
}

// Select returns the rows from the given table that satisfies the where
// condition with only the given columns, ordered by order if order is not the
// zero Property, and in range [first,first+limit] with first >= 0 and
// 0 < limit <= 1000.
func (warehouse *Snowflake) Select(ctx context.Context, table string, columns []types.Property, where warehouses.Where, order types.Property, first, limit int) ([][]any, error) {

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
			return nil, fmt.Errorf("column name %q is not a valid property name", c.Name)
		}
		query.WriteByte('"')
		query.WriteString(c.Name)
		query.WriteByte('"')
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
			return nil, fmt.Errorf("column name %q is not a valid property name", order.Name)
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
	rawRows, err := db.QueryContext(ctx, query.String())
	if err != nil {
		return nil, warehouses.Error(err)
	}
	defer rawRows.Close()
	var rows [][]any
	values := newScanValues(columns, &rows)
	for rawRows.Next() {
		if err = rawRows.Scan(values...); err != nil {
			return nil, warehouses.Error(err)
		}
	}
	if err = rawRows.Close(); err != nil {
		return nil, warehouses.Error(err)
	}
	if err = rawRows.Err(); err != nil {
		return nil, warehouses.Error(err)
	}
	if rows == nil {
		rows = [][]any{}
	}

	return rows, nil
}

// SetDestinationUser sets the destination user relative to the action, with the
// given external user ID and external property.
func (warehouse *Snowflake) SetDestinationUser(ctx context.Context, action int, externalUserID, externalProperty string) error {
	return errors.New("not implemented")
}

// Settings returns the data warehouse settings.
func (warehouse *Snowflake) Settings() []byte {
	s, _ := json.Marshal(warehouse.settings)
	return s
}

// Tables returns the tables of the data warehouse.
// It returns only the tables 'users', 'groups', 'events', and the tables with
// prefix 'users_', 'groups_' and 'events_'.
func (warehouse *Snowflake) Tables(ctx context.Context) ([]*warehouses.Table, error) {

	// Get the connection.
	db, err := warehouse.connection()
	if err != nil {
		return nil, err
	}

	var table *warehouses.Table
	var tables []*warehouses.Table

	// Read the columns.
	var b strings.Builder
	b.WriteString("SELECT c.table_name, c.column_name, c.is_nullable, c.data_type, c.character_maximum_length," +
		" c.numeric_precision_radix, c.numeric_precision, c.numeric_scale, c.comment\n" +
		"FROM information_schema.columns c\n" +
		"INNER JOIN information_schema.tables t ON c.table_name = t.table_name AND c.table_schema = t.table_schema\n" +
		"WHERE t.table_catalog = ")
	quoteString(&b, warehouse.settings.Database)
	b.WriteString(" AND t.table_schema = ")
	quoteString(&b, warehouse.settings.Schema)
	b.WriteString(" AND t.table_type = 'BASE TABLE' AND" +
		" ( t.table_name IN ('users', 'groups', 'events') OR t.table_name LIKE 'users\\__%' OR" +
		" t.table_name LIKE 'groups\\__%' OR t.table_name LIKE 'events\\__%' )\n" +
		"ORDER BY c.table_name, c.ordinal_position")

	rows, err := db.QueryContext(ctx, b.String())
	if err != nil {
		return nil, warehouses.Error(err)
	}
	defer rows.Close()

	var tableName, columnName, dataType, isNullable, charLength, radix, precision, scale, comment *string
	for rows.Next() {
		if err = rows.Scan(&tableName, &columnName, &isNullable, &dataType, &charLength, &radix, &precision, &scale, &comment); err != nil {
			return nil, warehouses.Error(err)
		}
		if tableName == nil {
			return nil, warehouses.Errorf("data warehouse has returned NULL as table name")
		}
		if columnName == nil {
			return nil, warehouses.Errorf("data warehouse has returned NULL as column name")
		}
		if !types.IsValidPropertyName(*columnName) {
			return nil, warehouses.Errorf("column name %q is not supported", *columnName)
		}
		if isNullable == nil {
			return nil, warehouses.Errorf("data warehouse has returned NULL as nullability of column")
		}
		if dataType == nil {
			return nil, warehouses.Errorf("data warehouse has returned NULL as column data type")
		}
		column := types.Property{
			Name:     *columnName,
			Nullable: *isNullable == "YES",
		}
		switch *dataType {
		case "ARRAY":
			column.Type = types.Array(types.JSON())
		case "BOOLEAN":
			column.Type = types.Boolean()
		case "DATE":
			column.Type = types.Date().WithLayout("2006-01-02")
		case "FLOAT":
			column.Type = types.Float()
		case "OBJECT":
			column.Type = types.Map(types.JSON())
		case "NUMBER":
			// Parse precision radix.
			if radix == nil {
				return nil, warehouses.Errorf("numeric_precision_radix value is NULL")
			}
			rdx, _ := strconv.Atoi(*radix)
			if rdx != 2 && rdx != 10 {
				return nil, warehouses.Errorf("numeric_precision_radix value %q is not valid", *radix)
			}
			// Parse precision.
			if precision == nil {
				return nil, warehouses.Errorf("numeric_precision value is NULL")
			}
			p, err := strconv.ParseInt(*precision, rdx, 64)
			if err != nil || p < 1 {
				return nil, warehouses.Errorf("numeric_precision value %q is not valid", *precision)
			}
			// Parse scale.
			if scale == nil {
				return nil, warehouses.Errorf("numeric_scale value is NULL")
			}
			s, err := strconv.ParseInt(*scale, rdx, 64)
			if err != nil || s < 0 || s > p {
				return nil, warehouses.Errorf("numeric_scale value %q is not valid", *scale)
			}
			column.Type = types.Decimal(int(p), int(s))
		case "TEXT":
			const maxByteLen = 16_777_216
			column.Type = types.Text().WithByteLen(maxByteLen)
			if charLength != nil {
				chars, _ := strconv.Atoi(*charLength)
				if chars < 1 {
					return nil, warehouses.Errorf("character_maximum_length value %q is not valid", *charLength)
				}
				if chars > types.MaxTextLen {
					return nil, warehouses.Errorf("length of column %s.%s exceeds %d characters", *tableName, *columnName, types.MaxTextLen)
				}
				column.Type = column.Type.WithCharLen(chars)
			}
		case "TIME":
			column.Type = types.Time().WithLayout("15:04:05.999999999")
		case "TIMESTAMP_NTZ":
			column.Type = types.DateTime().WithLayout("2006-01-02 15:04:05.999999999")
		case "VARIANT":
			column.Type = types.JSON()
		}
		if comment != nil {
			column.Description = *comment
		}
		if table == nil || *tableName != table.Name {
			table = &warehouses.Table{Name: *tableName}
			tables = append(tables, table)
		}
		table.Columns = append(table.Columns, column)
	}
	if err := rows.Close(); err != nil {
		return nil, warehouses.Error(err)
	}
	if err := rows.Err(); err != nil {
		return nil, warehouses.Error(err)
	}

	return tables, nil
}

// connection returns the database connection.
func (warehouse *Snowflake) connection() (*sql.DB, error) {
	warehouse.mu.Lock()
	defer warehouse.mu.Unlock()
	if warehouse.closed {
		return nil, errors.New("warehouse is closed")
	}
	if warehouse.settings == nil {
		return nil, errors.New("there are no settings")
	}
	if warehouse.db != nil {
		return warehouse.db, nil
	}
	db := sql.OpenDB(warehouse.settings.connector())
	warehouse.db = db
	return db, nil
}

// connector returns a driver.Connector from the settings.
func (s *sfSettings) connector() gosnowflake.Connector {
	return gosnowflake.NewConnector(gosnowflake.SnowflakeDriver{}, gosnowflake.Config{
		Account:   s.Account,
		User:      s.Username,
		Password:  s.Password,
		Database:  s.Database,
		Schema:    s.Schema,
		Warehouse: s.Warehouse,
		Role:      s.Role,
		Params:    make(map[string]*string),
	})
}

// serializeRowsToCSV serializes rows as CSV, using columns as header, and
// returns it as an io.Reader. It also appends a boolean column called $deleted
// with the value of the 'deleted' argument as value for each row.
func serializeRowsToCSV(columns []types.Property, rows [][]any, deleted bool) (io.Reader, error) {
	var b bytes.Buffer
	var bj bytes.Buffer
	for i, c := range columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(c.Name)
	}
	b.WriteString(",$deleted\n")
	for i, row := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		for j, v := range row {
			if j > 0 {
				b.WriteByte(',')
			}
			switch v := v.(type) {
			case bool:
				if v {
					b.WriteString("true")
				} else {
					b.WriteString("false")
				}
			case int:
				b.WriteString(strconv.Itoa(v))
			case int16:
				b.WriteString(strconv.Itoa(int(v)))
			case int32:
				b.WriteString(strconv.Itoa(int(v)))
			case int64:
				b.WriteString(strconv.Itoa(int(v)))
			case float64:
				b.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
			case decimal.Decimal:
				b.WriteString(v.String())
			case string:
				if v == "" || v[0] == '\'' || strings.ContainsAny(v, ",\n") {
					quoteCSVString(&b, v)
				} else {
					b.WriteString(v)
				}
			default:
				switch pt := columns[j].Type.PhysicalType(); pt {
				case types.PtJSON, types.PtArray, types.PtMap:
					switch v := v.(type) {
					case json.RawMessage:
						quoteCSVBytes(&b, v)
					case json.Number:
						b.WriteString(string(v))
					default:
						bj.Reset()
						enc := json.NewEncoder(&bj)
						enc.SetEscapeHTML(false)
						err := enc.Encode(v)
						if err != nil {
							return nil, err
						}
						quoteCSVBytes(&b, bj.Bytes())
					}
				case types.PtDateTime:
					b.WriteString(v.(time.Time).Format("2006-01-02 15:04:05.999999999"))
				case types.PtDate:
					b.WriteString(v.(time.Time).Format("2006-01-02"))
				case types.PtTime:
					b.WriteString(v.(time.Time).Format("15:04:05.999999999"))
				default:
					return nil, fmt.Errorf("cannot serialize as Snowflake CSV: unsupported type %T for column type %s", v, pt)
				}
			}
		}
		// Add the value for the column $deleted.
		if deleted {
			b.WriteString(",true")
		} else {
			b.WriteString(",false")
		}
	}
	return &b, nil
}

// quoteCSVString quotes the string s for use in a CSV file and writes it to b.
// A string must be quoted if is empty, or starts with the character "'", or
// contains characters "," or "\n".
func quoteCSVString(b *bytes.Buffer, s string) {
	b.WriteByte('\'')
	for len(s) > 0 {
		i := strings.IndexByte(s, '\'')
		if i == -1 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:i+1])
		b.WriteByte('\'')
		s = s[i+1:]
	}
	b.WriteByte('\'')
}

// quoteCSVBytes is like quoteCSVString but gets a []byte value as argument.
func quoteCSVBytes(b *bytes.Buffer, s []byte) {
	b.WriteByte('\'')
	for len(s) > 0 {
		i := bytes.IndexByte(s, '\'')
		if i == -1 {
			b.Write(s)
			break
		}
		b.Write(s[:i+1])
		b.WriteByte('\'')
		s = s[i+1:]
	}
	b.WriteByte('\'')
}
