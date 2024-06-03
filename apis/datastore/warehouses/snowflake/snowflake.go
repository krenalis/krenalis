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

	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/snowflakedb/gosnowflake"
)

var _ warehouses.Warehouse = &Snowflake{}

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

// AlterSchema alters the user schema.
func (warehouse *Snowflake) AlterSchema(ctx context.Context, usersColumns []warehouses.Column, operations []warehouses.AlterSchemaOperation) error {
	panic("TODO: not implemented")
}

// AlterSchemaQueries returns the queries of a schema altering operation.
func (warehouse *Snowflake) AlterSchemaQueries(ctx context.Context, usersColumns []warehouses.Column, operations []warehouses.AlterSchemaOperation) ([]string, error) {
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
		return warehouses.Error(err)
	}
	return nil
}

// DeleteConnectionIdentities deletes the identities of a connection.
func (warehouse *Snowflake) DeleteConnectionIdentities(ctx context.Context, connection int) error {
	panic("not implemented")
}

// DestinationUsers returns the destination users of the action.
func (warehouse *Snowflake) DestinationUsers(ctx context.Context, action int, propertyValue string) ([]string, error) {
	panic("not implemented")
}

// DuplicatedDestinationUsers retrieves duplicated destination users.
func (warehouse *Snowflake) DuplicatedDestinationUsers(ctx context.Context, action int) (string, string, bool, error) {
	panic("TODO: not implemented")
}

// DuplicatedUsers returns the GIDs of two duplicated users.
func (warehouse *Snowflake) DuplicatedUsers(ctx context.Context, column string) (uuid.UUID, uuid.UUID, bool, error) {
	panic("TODO: not implemented")
}

// Init initializes the data warehouse by creating the supporting tables.
func (warehouse *Snowflake) Init(ctx context.Context) error {
	return nil
}

// Merge performs a table merge operation.
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
		n := len(table.Keys)
		rows := make([][]any, 0, len(deleted)/n)
		for i := 0; i < len(deleted); i += n {
			rows = append(rows, deleted[i:i+n])
		}
		var err error
		deletedCSV, err = serializeRowsToCSV(table.Keys, rows, true)
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
	for i, key := range table.Keys {
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
			for _, key := range table.Keys {
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

// MergeIdentities merge existing identities, deletes them and inserts new ones.
func (warehouse *Snowflake) MergeIdentities(ctx context.Context, columns []warehouses.Column, rows []map[string]any) error {
	panic("TODO: not implemented")
}

// Ping checks the connection to the data warehouse.
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

// Query executes a query and returns the results as a Rows.
func (warehouse *Snowflake) Query(ctx context.Context, query warehouses.RowQuery) (warehouses.Rows, int, error) {
	panic("not implemented")
}

// RunIdentityResolution runs the Identity Resolution.
func (warehouse *Snowflake) RunIdentityResolution(ctx context.Context, connections []int, identifiers, usersColumns []warehouses.Column) error {
	panic("not implemented")
}

// SetDestinationUser sets the destination user for an action.
func (warehouse *Snowflake) SetDestinationUser(ctx context.Context, action int, externalUserID, externalProperty string) error {
	return errors.New("not implemented")
}

// Settings returns the data warehouse settings.
func (warehouse *Snowflake) Settings() []byte {
	s, _ := json.Marshal(warehouse.settings)
	return s
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
// returns it as an io.Reader. It also appends a boolean column called $deleted
// with the value of the 'deleted' argument as value for each row.
func serializeRowsToCSV(columns []warehouses.Column, rows [][]any, deleted bool) (io.Reader, error) {
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
				switch k := columns[j].Type.Kind(); k {
				case types.JSONKind, types.ArrayKind, types.MapKind:
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
