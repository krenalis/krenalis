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

// Connector icon.
var icon = "<svg></svg>"

var (
	//go:embed tables/destinations_users.sql
	createDestinationUsersTable string
	//go:embed tables/events.sql
	createEventsTable string
	//go:embed tables/operations.sql
	createOperationsTable string
	//go:embed tables/user_identities.sql
	createUserIdentitiesTable string
	//go:embed tables/users.sql
	createUsersTable string
	//go:embed tables/users_view.sql
	createUsersView string
)

var _ meergo.Warehouse = &Snowflake{}

func init() {
	meergo.RegisterWarehouse(meergo.WarehouseInfo{
		Name: "Snowflake",
		Icon: icon,
	}, New)
}

// New returns a new Snowflake data warehouse driver instance.
// It returns a SettingsError error if the settings are not valid.
func New(conf *meergo.WarehouseConfig) (*Snowflake, error) {
	var s sfSettings
	err := stdjson.Unmarshal(conf.Settings, &s)
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
	return &Snowflake{conf: conf, settings: &s}, nil
}

type Snowflake struct {
	mu       sync.Mutex // for the db field
	db       *sql.DB
	conf     *meergo.WarehouseConfig
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

// CanInitialize checks whether the data warehouse can be initialized.
func (warehouse *Snowflake) CanInitialize(ctx context.Context) error {
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return meergo.Error(err)
	}
	defer conn.Close()
	rows, err := conn.QueryContext(ctx, "SHOW TERSE OBJECTS")
	if err != nil {
		return meergo.Error(err)
	}
	defer rows.Close()
	var objects []string
	for rows.Next() {
		var createdOn, databaseName, schemaName any
		var name, kind string
		err := rows.Scan(&createdOn, &name, &kind, &databaseName, &schemaName)
		if err != nil {
			return meergo.Error(err)
		}
		typ := strings.ToLower(kind)
		objects = append(objects, fmt.Sprintf("%s '%s'", typ, name))
	}
	if err := rows.Err(); err != nil {
		return meergo.Error(err)
	}
	if objects != nil {
		reason := fmt.Sprintf("database contains these objects: %s", strings.Join(objects, ", "))
		return meergo.NewNotInitializableError(reason)
	}
	return nil
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

// Initialize initializes the database objects on the data warehouse in order to
// make it work with Meergo.
func (warehouse *Snowflake) Initialize(ctx context.Context) error {
	return warehouse.initRepair(ctx, false)
}

// Merge performs a table merge operation.
func (warehouse *Snowflake) Merge(ctx context.Context, table meergo.WarehouseTable, rows [][]any, deleted []any) error {

	db, err := warehouse.connection()
	if err != nil {
		return err
	}

	// Generate a unique name for the temporary table.
	tempTableName := "temp_table_" + strconv.FormatInt(time.Now().UnixNano(), 10)

	// Prepare the "create temporary table" statement.
	var b strings.Builder
	b.WriteString(`CREATE TEMPORARY TABLE "`)
	b.WriteString(tempTableName)
	b.WriteString("\" AS\n  SELECT ")
	for _, c := range table.Columns {
		b.WriteByte('"')
		b.WriteString(c.Name)
		b.WriteString(`",`)
	}
	b.WriteString(`FALSE AS "$purge" FROM "`)
	b.WriteString(table.Name)
	b.WriteString("\" LIMIT 0")
	create := b.String()

	// Create the 'merge' statement.
	b.Reset()
	b.WriteString(`MERGE INTO `)
	b.WriteString(quoteTable(table.Name))
	b.WriteString(" d\nUSING \"")
	b.WriteString(tempTableName)
	b.WriteString("\" s\nON ")
	for i, key := range table.Keys {
		if i > 0 {
			b.WriteString(" AND ")
		}
		b.WriteString(`d."`)
		b.WriteString(key)
		b.WriteString(`" = s."`)
		b.WriteString(key)
		b.WriteByte('"')
	}
	if len(rows) > 0 {
		b.WriteString("\nWHEN MATCHED AND NOT s.\"$purge\" THEN\n  UPDATE SET ")
		i := 0
		for _, c := range table.Columns {
			if slices.Contains(table.Keys, c.Name) {
				continue
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
		b.WriteString("\nWHEN NOT MATCHED AND NOT s.\"$purge\" THEN\n  INSERT (")
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
	merge := b.String()

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

	// Acquire a connection.
	conn, err := db.Conn(ctx)
	if err != nil {
		return meergo.Error(err)
	}
	defer conn.Close()

	// Create the temporary table.
	_, err = conn.ExecContext(ctx, create)
	if err != nil {
		return meergo.Error(err)
	}

	// Copy the rows into the temporary table.
	if len(rows) > 0 {
		// Put the rows into the temporary table's stage.
		_, err = conn.ExecContext(gosnowflake.WithFileStream(ctx, rowsCSV), `PUT file://rows.csv @%"`+tempTableName+`"`)
		if err != nil {
			return meergo.Error(err)
		}
		// Copy the rows from the stage into the temporary table.
		b.Reset()
		b.WriteString("COPY INTO \"")
		b.WriteString(tempTableName)
		b.WriteString("\"\nFROM @%\"")
		b.WriteString(tempTableName)
		b.WriteString("\"\nFILE_FORMAT = (TYPE=CSV PARSE_HEADER=TRUE FIELD_OPTIONALLY_ENCLOSED_BY='0x27' ESCAPE_UNENCLOSED_FIELD=NONE EMPTY_FIELD_AS_NULL=TRUE NULL_IF=())\n" +
			"FILES = ('rows.csv.gz')\n" +
			"MATCH_BY_COLUMN_NAME = CASE_SENSITIVE\n" +
			"ON_ERROR = ABORT_STATEMENT")
		_, err = conn.ExecContext(ctx, b.String())
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
		b.Reset()
		b.WriteString("COPY INTO \"")
		b.WriteString(tempTableName)
		b.WriteString("\"\nFROM @%\"")
		b.WriteString(tempTableName)
		b.WriteString("\"\nFILE_FORMAT = (TYPE=CSV PARSE_HEADER=TRUE FIELD_OPTIONALLY_ENCLOSED_BY='0x27' ESCAPE_UNENCLOSED_FIELD=NONE EMPTY_FIELD_AS_NULL=TRUE NULL_IF=())\n" +
			"FILES = ('rows.csv.gz')\n" +
			"MATCH_BY_COLUMN_NAME = CASE_SENSITIVE\n" +
			"OVERWRITE = TRUE\n" +
			"ON_ERROR = ABORT_STATEMENT")
		_, err = conn.ExecContext(ctx, b.String())
		if err != nil {
			return meergo.Error(err)
		}
	}

	// Merge the temporary table's rows with the destination table's rows.
	_, err = conn.ExecContext(ctx, merge)
	if err != nil {
		return meergo.Error(err)
	}

	return nil
}

// MergeIdentities merge existing identities, deletes them and inserts new ones.
func (warehouse *Snowflake) MergeIdentities(ctx context.Context, columns []meergo.Column, rows []map[string]any) error {
	panic("TODO: not implemented")
}

// Repair repairs the database objects on the data warehouse needed by Meergo.
func (warehouse *Snowflake) Repair(ctx context.Context) error {
	return warehouse.initRepair(ctx, true)
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

// initRepair initializes (or repairs) the database objects (as tables, types,
// etc...) on the data warehouse.
func (warehouse *Snowflake) initRepair(ctx context.Context, repair bool) error {
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return meergo.Error(err)
	}
	defer conn.Close()
	queries := []string{
		createDestinationUsersTable,
		createEventsTable,
		createOperationsTable,
		createUserIdentitiesTable,
		createUsersTable,
	}
	if !repair { // TODO(Gianluca): is this necessary in Snowflake?
		queries = append(queries, createUsersView)
	}
	for _, query := range queries {
		_, err := conn.ExecContext(ctx, query)
		if err != nil {
			return meergo.Error(err)
		}
	}
	return nil
}

// usersVersion returns the version of the "users" table.
func (warehouse *Snowflake) usersVersion(ctx context.Context) (int, error) {
	db, err := warehouse.connection()
	if err != nil {
		return 0, err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return 0, meergo.Error(err)
	}
	defer conn.Close()
	var v int
	err = conn.QueryRowContext(ctx, `SELECT COALESCE(MAX("users_version"), 0) FROM "_operations"`).Scan(&v)
	if err != nil {
		return 0, meergo.Error(err)
	}
	return v, nil
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

// execTransaction executes the function f within a transaction. If f returns an
// error or panics, the transaction will be rolled back.
func (warehouse *Snowflake) execTransaction(ctx context.Context, f func(*sql.Tx) error) error {
	// TODO(Gianluca): is the use of the context in this method correct?
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return meergo.Error(err)
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return meergo.Error(err)
	}
	defer tx.Rollback()
	err = f(tx)
	if err != nil {
		return meergo.Error(err)
	}
	err = tx.Commit()
	if err != nil && !errors.Is(err, sql.ErrTxDone) {
		return meergo.Error(err)
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
