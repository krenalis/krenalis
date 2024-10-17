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
	stdjson "encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/snowflakedb/gosnowflake"
)

var _ meergo.Warehouse = &Snowflake{}

type Snowflake struct {
	mu       sync.Mutex // for the db field
	db       *sql.DB
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
func Open(settings []byte) (*Snowflake, error) {
	var s sfSettings
	err := stdjson.Unmarshal(settings, &s)
	if err != nil {
		return nil, meergo.SettingsErrorf("cannot unmarshal settings: %s", err)
	}
	// Validate Account.
	if n := utf8.RuneCountInString(s.Account); n < 1 || n > 255 {
		return nil, meergo.SettingsErrorf("account length must be in range [1,255]")
	}
	// Validate Username.
	if n := utf8.RuneCountInString(s.Username); n < 1 || n > 255 {
		return nil, meergo.SettingsErrorf("username length must be in range [1,255]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 255 {
		return nil, meergo.SettingsErrorf("password length must be in range [1,255]")
	}
	// Validate Database.
	if n := utf8.RuneCountInString(s.Database); n < 1 || n > 255 {
		return nil, meergo.SettingsErrorf("database length must be in range [1,255]")
	}
	// Validate Schema.
	if n := utf8.RuneCountInString(s.Schema); n < 1 || n > 255 {
		return nil, meergo.SettingsErrorf("schema length must be in range [1,255]")
	}
	// Validate Warehouse.
	if n := utf8.RuneCountInString(s.Warehouse); n < 1 || n > 255 {
		return nil, meergo.SettingsErrorf("warehouse length must be in range [1,255]")
	}
	// Validate Role.
	if n := utf8.RuneCountInString(s.Role); n < 1 || n > 255 {
		return nil, meergo.SettingsErrorf("role length must be in range [1,255]")
	}
	return &Snowflake{settings: &s}, nil
}

// AlterSchema alters the user schema.
func (warehouse *Snowflake) AlterSchema(ctx context.Context, userColumns []meergo.Column, operations []meergo.AlterSchemaOperation) error {
	panic("TODO: not implemented")
}

// AlterSchemaQueries returns the queries of a schema altering operation.
func (warehouse *Snowflake) AlterSchemaQueries(ctx context.Context, userColumns []meergo.Column, operations []meergo.AlterSchemaOperation) ([]string, error) {
	panic("TODO: not implemented")
}

// CanInitialize checks whether the data warehouse can be initialized.
func (warehouse *Snowflake) CanInitialize(ctx context.Context) error {
	panic("TODO: not implemented")
}

// Close closes the data warehouse.
func (warehouse *Snowflake) Close() error {
	if warehouse.db == nil {
		return nil
	}
	err := warehouse.db.Close()
	warehouse.db = nil
	if err != nil {
		return meergo.Error(err)
	}
	return nil
}

// Delete deletes rows from the specified table that match the provided where
// expression.
func (warehouse *Snowflake) Delete(ctx context.Context, table string, where meergo.Expr) error {
	if where == nil {
		return errors.New("where is nil")
	}
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	var s strings.Builder
	s.WriteString(`DELETE FROM "` + table + `" WHERE `)
	err = renderExpr(&s, where)
	if err != nil {
		return fmt.Errorf("cannot build WHERE expression: %s", err)
	}
	_, err = db.ExecContext(ctx, s.String())
	if err != nil {
		return meergo.Error(err)
	}
	return nil
}

// LastIdentityResolution returns information about the last Identity
// Resolution.
func (warehouse *Snowflake) LastIdentityResolution(ctx context.Context) (startTime, endTime *time.Time, err error) {
	panic("not implemented")
}

// Initialize initializes the database objects on the data warehouse in order to
// make it work with Meergo.
func (warehouse *Snowflake) Initialize(ctx context.Context) error {
	panic("TODO: not implemented")
}

// Merge performs a table merge operation.
func (warehouse *Snowflake) Merge(ctx context.Context, table meergo.WarehouseTable, rows [][]any, deleted []any) error {

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
		n := len(table.Keys)
		rows := make([][]any, 0, len(deleted)/n)
		for i := 0; i < len(deleted); i += n {
			rows = append(rows, deleted[i:i+n])
		}
		keys := make([]meergo.Column, len(table.Keys))
		for i, key := range table.Keys {
			for _, c := range table.Columns {
				if c.Name == key {
					keys[i] = c
					break
				}
			}
		}
		var err error
		deletedCSV, err = serializeRowsToCSV(keys, rows, true)
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
		switch c.Type.Kind() {
		case types.BooleanKind:
			q.WriteString("BOOLEAN")
		case types.FloatKind:
			q.WriteString("FLOAT")
		case
			types.IntKind,
			types.UintKind,
			types.YearKind:
			q.WriteString("NUMBER(38,0)")
		case types.DecimalKind:
			q.WriteString("NUMBER(")
			q.WriteString(strconv.Itoa(c.Type.Precision()))
			q.WriteByte(',')
			q.WriteString(strconv.Itoa(c.Type.Scale()))
			q.WriteByte(')')
		case types.DateTimeKind:
			q.WriteString("TIMESTAMP_NTZ")
		case types.DateKind:
			q.WriteString("DATE")
		case types.TimeKind:
			q.WriteString("TIME")
		case types.JSONKind:
			q.WriteString("VARIANT")
		case types.UUIDKind:
			q.WriteString("VARCHAR(36)")
		case types.InetKind:
			q.WriteString("VARCHAR(45)")
		case types.TextKind:
			q.WriteString("VARCHAR")
			if l, ok := c.Type.CharLen(); ok {
				q.WriteByte('(')
				q.WriteString(strconv.Itoa(l))
				q.WriteByte(')')
			}
		case types.ArrayKind:
			q.WriteString("ARRAY")
		case types.MapKind:
			q.WriteString("OBJECT")
		default:
			panic("unsupported type")
		}
		if !c.Nullable {
			q.WriteString(" NOT NULL")
		}
		q.WriteString(",\n")
	}
	q.WriteString("\"$purge\" BOOLEAN NOT NULL\n)")

	conn, err := db.Conn(ctx)
	if err != nil {
		return meergo.Error(err)
	}
	defer conn.Close()

	_, err = conn.ExecContext(ctx, q.String())
	if err != nil {
		return meergo.Error(err)
	}
	defer func() {
		_, _ = conn.ExecContext(ctx, `DROP TABLE "`+tempTableName+`"`)
	}()

	// Copy the rows into the temporary table.
	if len(rows) > 0 {
		// Put the rows into the temporary table's stage.
		_, err = conn.ExecContext(gosnowflake.WithFileStream(ctx, rowsCSV), `PUT file://rows.csv @%"`+tempTableName+`"`)
		if err != nil {
			return meergo.Error(err)
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
			return meergo.Error(err)
		}
	}

	// Copy the deleted rows into the temporary table.
	if len(deleted) > 0 {
		// Put the deleted rows into the temporary table's stage.
		_, err = conn.ExecContext(gosnowflake.WithFileStream(ctx, deletedCSV), `PUT file://rows.csv @%"`+tempTableName+`"`)
		if err != nil {
			return meergo.Error(err)
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
			return meergo.Error(err)
		}
	}

	// Merge the temporary table's rows with the destination table's rows.
	q.Reset()
	q.WriteString(`MERGE INTO `)
	q.WriteString(quoteTable(table.Name))
	q.WriteString(" d\nUSING \"")
	q.WriteString(tempTableName)
	q.WriteString("\" s\nON ")
	for i, key := range table.Keys {
		if i > 0 {
			q.WriteString(" AND ")
		}
		q.WriteString(`d."`)
		q.WriteString(key)
		q.WriteString(`" = s."`)
		q.WriteString(key)
		q.WriteByte('"')
	}
	if len(rows) > 0 {
		q.WriteString("\nWHEN MATCHED AND NOT s.\"$purge\" THEN\n  UPDATE SET ")
		i := 0
		for _, c := range table.Columns {
			if slices.Contains(table.Keys, c.Name) {
				continue
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
		q.WriteString("\nWHEN NOT MATCHED AND NOT s.\"$purge\" THEN\n  INSERT (")
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
		return meergo.Error(err)
	}

	return nil
}

// MergeIdentities merge existing identities, deletes them and inserts new ones.
func (warehouse *Snowflake) MergeIdentities(ctx context.Context, columns []meergo.Column, rows []map[string]any) error {
	panic("TODO: not implemented")
}

// Query executes a query and returns the results as Rows.
func (warehouse *Snowflake) Query(ctx context.Context, query meergo.RowQuery, withCount bool) (meergo.Rows, int, error) {

	db, err := warehouse.connection()
	if err != nil {
		return nil, 0, err
	}

	// Build the WHERE expression, if necessary.
	var whereExpr string
	if query.Where != nil {
		var s strings.Builder
		err = renderExpr(&s, query.Where)
		if err != nil {
			return nil, 0, fmt.Errorf("cannot build WHERE expression: %s", err)
		}
		whereExpr = s.String()
	}

	var b strings.Builder

	// Count the total number of records.
	var count int
	if withCount {
		b.WriteString(`SELECT COUNT(*) FROM "`)
		b.WriteString(query.Table)
		b.WriteByte('"')
		err = appendJoins(&b, query.Joins)
		if err != nil {
			return nil, 0, err
		}
		if query.Where != nil {
			b.WriteString(` WHERE `)
			b.WriteString(whereExpr)
		}
		err = db.QueryRowContext(ctx, b.String()).Scan(&count)
		if err != nil {
			return nil, 0, meergo.Error(err)
		}
		b.Reset()
	}

	// Build the query.
	b.WriteString(`SELECT `)
	for i, c := range query.Columns {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteByte('"')
		b.WriteString(c.Name)
		b.WriteByte('"')
	}
	b.WriteString(` FROM "`)
	b.WriteString(query.Table)
	b.WriteByte('"')

	err = appendJoins(&b, query.Joins)
	if err != nil {
		return nil, 0, err
	}

	if query.Where != nil {
		b.WriteString(` WHERE `)
		b.WriteString(whereExpr)
	}

	if query.OrderBy.Name != "" {
		b.WriteString(" ORDER BY \"")
		b.WriteString(query.OrderBy.Name)
		b.WriteRune('"')
		if query.OrderDesc {
			b.WriteString(" DESC")
		}
	}
	if query.Limit > 0 {
		b.WriteString(" LIMIT ")
		b.WriteString(strconv.Itoa(query.Limit))
	}
	if query.First > 0 {
		b.WriteString(" OFFSET ")
		b.WriteString(strconv.Itoa(query.First))
	}

	// Execute the query.
	rows, err := db.QueryContext(ctx, b.String())
	if err != nil {
		return nil, 0, meergo.Error(err)
	}

	return rows, count, nil
}

// ResolveIdentities resolves the identities.
func (warehouse *Snowflake) ResolveIdentities(ctx context.Context, identifiers, userColumns []meergo.Column, userPrimarySources map[string]int) error {
	panic("not implemented")
}

// Repair repairs the database objects on the data warehouse needed by Meergo.
func (warehouse *Snowflake) Repair(ctx context.Context) error {
	panic("TODO: not implemented")
}

// Settings returns the data warehouse settings.
func (warehouse *Snowflake) Settings() []byte {
	s, _ := stdjson.Marshal(warehouse.settings)
	return s
}

// Truncate truncates the specified table.
func (warehouse *Snowflake) Truncate(ctx context.Context, table string) error {
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `TRUNCATE TABLE "`+table+`"`)
	if err != nil {
		return meergo.Error(err)
	}
	return nil
}

// connection returns the Snowflake connection.
func (warehouse *Snowflake) connection() (*sql.DB, error) {
	warehouse.mu.Lock()
	defer warehouse.mu.Unlock()
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
// returns it as an io.Reader. It also appends a boolean column called $purge
// with the value of the 'deleted' argument as value for each row.
func serializeRowsToCSV(columns []meergo.Column, rows [][]any, deleted bool) (io.Reader, error) {
	var b bytes.Buffer
	var bj bytes.Buffer
	for i, c := range columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(c.Name)
	}
	b.WriteString(",$purge\n")
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
				v.WriteTo(&b)
			case json.Value:
				quoteCSVBytes(&b, v)
			case string:
				if v == "" || v[0] == '\'' || strings.ContainsAny(v, ",\n") {
					quoteCSVString(&b, v)
				} else {
					b.WriteString(v)
				}
			default:
				switch k := columns[j].Type.Kind(); k {
				case types.ArrayKind, types.MapKind:
					bj.Reset()
					enc := stdjson.NewEncoder(&bj)
					enc.SetEscapeHTML(false)
					err := enc.Encode(v)
					if err != nil {
						return nil, err
					}
					quoteCSVBytes(&b, bj.Bytes())
				case types.DateTimeKind:
					b.WriteString(v.(time.Time).Format("2006-01-02 15:04:05.999999999"))
				case types.DateKind:
					b.WriteString(v.(time.Time).Format("2006-01-02"))
				case types.TimeKind:
					b.WriteString(v.(time.Time).Format("15:04:05.999999999"))
				default:
					return nil, fmt.Errorf("cannot serialize as Snowflake CSV: unsupported type %T for column type %s", v, k)
				}
			}
		}
		// Add the value for the column $purge.
		if deleted {
			b.WriteString(",true")
		} else {
			b.WriteString(",false")
		}
	}
	return &b, nil
}

// appendJoins appends the string serialization of the provided joins to b.
func appendJoins(b *strings.Builder, joins []meergo.Join) error {
	for _, join := range joins {
		switch join.Type {
		case meergo.Inner:
			b.WriteString(` JOIN "`)
		case meergo.Left:
			b.WriteString(` LEFT JOIN "`)
		case meergo.Right:
			b.WriteString(` RIGHT JOIN "`)
		case meergo.Full:
			b.WriteString(` FULL JOIN "`)
		}
		b.WriteString(join.Table)
		b.WriteString(`" ON `)
		err := renderExpr(b, join.Condition)
		if err != nil {
			return fmt.Errorf("cannot build JOIN condition: %s", err)
		}
	}
	return nil
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
