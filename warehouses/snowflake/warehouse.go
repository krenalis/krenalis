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
	jsonstd "encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo"

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
	//go:embed tables/user_schema_versions.sql
	createUserSchemaVersionTable string
)

var _ meergo.Warehouse = &Snowflake{}

func init() {
	meergo.RegisterWarehouseDriver(meergo.WarehouseDriver{
		Name: "Snowflake",
		Icon: icon,
	}, New)
}

// accountFormat is the format of the account identifier in the settings.
var accountFormat = regexp.MustCompile(`^[a-zA-Z0-9]+[.-][a-zA-Z0-9]+$`)

// New returns a new Snowflake data warehouse driver instance.
// It returns a *meergo.WarehouseSettingsError if the settings are not valid.
func New(conf *meergo.WarehouseConfig) (*Snowflake, error) {
	var s sfSettings
	err := jsonstd.Unmarshal(conf.Settings, &s)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal settings: %s", err)
	}
	// Validate Account.
	if n := utf8.RuneCountInString(s.Account); n < 3 || n > 255 {
		return nil, meergo.WarehouseSettingsErrorf("account identifier length must be in range [3,255]")
	}
	if !accountFormat.MatchString(s.Account) {
		return nil, meergo.WarehouseSettingsErrorf("account identifier must be in the <organization>.<account> or <organization>-<account> format")
	}
	// Validate Username.
	if n := utf8.RuneCountInString(s.Username); n < 1 || n > 255 {
		return nil, meergo.WarehouseSettingsErrorf("username length must be in range [1,255]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 255 {
		return nil, meergo.WarehouseSettingsErrorf("password length must be in range [1,255]")
	}
	// Validate Database.
	if n := utf8.RuneCountInString(s.Database); n < 1 || n > 255 {
		return nil, meergo.WarehouseSettingsErrorf("database length must be in range [1,255]")
	}
	// Validate Schema.
	if n := utf8.RuneCountInString(s.Schema); n < 1 || n > 255 {
		return nil, meergo.WarehouseSettingsErrorf("schema length must be in range [1,255]")
	}
	// Validate Warehouse.
	if n := utf8.RuneCountInString(s.Warehouse); n < 1 || n > 255 {
		return nil, meergo.WarehouseSettingsErrorf("warehouse length must be in range [1,255]")
	}
	// Validate Role.
	if n := utf8.RuneCountInString(s.Role); n < 1 || n > 255 {
		return nil, meergo.WarehouseSettingsErrorf("role length must be in range [1,255]")
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

// Close closes the data warehouse.
func (warehouse *Snowflake) Close() error {
	if warehouse.db == nil {
		return nil
	}
	err := warehouse.db.Close()
	warehouse.db = nil
	if err != nil {
		return snowflake(err)
	}
	return nil
}

// Delete deletes rows from the specified table that match the provided where
// expression.
func (warehouse *Snowflake) Delete(ctx context.Context, table string, where meergo.Expr) error {
	if where == nil {
		return errors.New("where is nil")
	}
	var s strings.Builder
	s.WriteString("DELETE FROM " + quoteIdent(table) + " WHERE ")
	err := renderExpr(&s, where)
	if err != nil {
		return fmt.Errorf("cannot build WHERE expression: %s", err)
	}
	db := warehouse.openDB()
	_, err = db.ExecContext(ctx, s.String())
	if err != nil {
		return snowflake(err)
	}
	return nil
}

// Merge performs a table merge operation.
func (warehouse *Snowflake) Merge(ctx context.Context, table meergo.Table, rows [][]any, deleted []any) error {
	// Acquire a connection.
	db := warehouse.openDB()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	// Merge rows.
	err = merge(ctx, conn, table, rows, deleted)
	if err != nil {
		return snowflake(err)
	}
	return nil
}

// immutableMergeIdentitiesColumns are columns in the merge of identities that
// are immutable.
var immutableMergeIdentitiesColumns = []string{
	"__action__",
	"__identity_id__",
	"__is_anonymous__",
	"__connection__",
}

// MergeIdentities merges existing identities, deletes them, and inserts new
// ones.
func (warehouse *Snowflake) MergeIdentities(ctx context.Context, columns []meergo.Column, rows []map[string]any) error {

	quotedColumn := make(map[string]string, len(columns))
	for _, column := range columns {
		quotedColumn[column.Name] = quoteIdent(column.Name)
	}

	// Generate a unique name for the temporary table.
	tempTableName := "TEMP_TABLE_" + strconv.FormatInt(time.Now().UnixNano(), 10)

	// Prepare the "create temporary table" statement.
	var b strings.Builder
	b.WriteString(`CREATE TEMPORARY TABLE "`)
	b.WriteString(tempTableName)
	b.WriteString("\" AS\n  SELECT ")
	for _, c := range columns {
		b.WriteString(quotedColumn[c.Name])
		b.WriteByte(',')
	}
	b.WriteString(`FALSE AS "$PURGE" FROM "_USER_IDENTITIES" LIMIT 0`)
	create := b.String()

	// Prepare the "merge" statement.
	b.Reset()
	b.WriteString("MERGE INTO \"_USER_IDENTITIES\" AS \"D\"\nUSING \"")
	b.WriteString(tempTableName)
	b.WriteString(`" AS "S"` + "\n" + `ON "D"."__ACTION__" = "S"."__ACTION__" AND "D"."__IDENTITY_ID__" = "S"."__IDENTITY_ID__" AND "D"."__IS_ANONYMOUS__" = "S"."__IS_ANONYMOUS__"`)
	b.WriteString("\nWHEN MATCHED AND \"S\".\"$PURGE\" IS NULL THEN\n  UPDATE SET ")
	i := 0
	for _, c := range columns {
		if slices.Contains(immutableMergeIdentitiesColumns, c.Name) {
			continue
		}
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString("\n")
		b.WriteString(quotedColumn[c.Name])
		b.WriteString(` = `)
		if c.Name == "__anonymous_ids__" {
			b.WriteString(`CASE WHEN "S"."__ANONYMOUS_IDS__" IS NULL OR ARRAY_CONTAINS("D"."__ANONYMOUS_IDS__", "S"."__ANONYMOUS_IDS__"[0]) THEN "D"."__ANONYMOUS_IDS__" ELSE ARRAY_CAT("D"."__ANONYMOUS_IDS__", ARRAY_CONSTRUCT("S"."__ANONYMOUS_IDS__"[0])) END`)
		} else {
			b.WriteString(`"S".`)
			b.WriteString(quotedColumn[c.Name])
		}
		i++
	}
	if i == 0 {
		return errors.New("snowflake.MergeIdentities: there must be at least one column in 'columns' apart from the immutable identities columns")
	}
	b.WriteString("\nWHEN NOT MATCHED AND \"S\".\"$PURGE\" IS NULL THEN\n  INSERT (")
	for i, c := range columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(quotedColumn[c.Name])
	}
	b.WriteString(")\n  VALUES (")
	for i, c := range columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`"S".`)
		b.WriteString(quotedColumn[c.Name])
	}
	b.WriteString(")\nWHEN MATCHED AND \"S\".\"$PURGE\" = FALSE THEN\n  UPDATE SET \"__EXECUTION__\" = \"S\".\"__EXECUTION__\"")
	b.WriteString("\nWHEN MATCHED AND \"S\".\"$PURGE\" = TRUE THEN\n  DELETE")
	merge := b.String()

	// Serialize the rows in CSV format.
	csvReader, err := serializeIdentitiesToCSV(columns, rows)
	if err != nil {
		return err
	}

	// Acquire a connection.
	db := warehouse.openDB()
	conn, err := db.Conn(ctx)
	if err != nil {
		return snowflake(err)
	}
	defer conn.Close()

	// Create the temporary table.
	_, err = conn.ExecContext(ctx, create)
	if err != nil {
		return snowflake(err)
	}

	// Copy the rows into the temporary table.
	if len(rows) > 0 {
		// Put the rows into the temporary table's stage.
		_, err = conn.ExecContext(gosnowflake.WithFileStream(ctx, csvReader), `PUT file://rows.csv @%"`+tempTableName+`"`)
		if err != nil {
			return snowflake(err)
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
			return snowflake(err)
		}
	}

	// Merge the temporary table's rows with the destination table's rows.
	_, err = conn.ExecContext(ctx, merge)
	if err != nil {
		return snowflake(err)
	}

	return nil
}

// Settings returns the data warehouse settings.
func (warehouse *Snowflake) Settings() []byte {
	s, _ := jsonstd.Marshal(warehouse.settings)
	return s
}

// Truncate truncates the specified table.
func (warehouse *Snowflake) Truncate(ctx context.Context, table string) error {
	db := warehouse.openDB()
	_, err := db.ExecContext(ctx, "TRUNCATE TABLE "+quoteIdent(table))
	if err != nil {
		return snowflake(err)
	}
	return nil
}

// UnsetIdentityColumns unsets values for the specified identity columns for the
// given action.
func (warehouse *Snowflake) UnsetIdentityColumns(ctx context.Context, action int, columns []meergo.Column) error {
	var b strings.Builder
	b.WriteString("UPDATE \"_USER_IDENTITIES\" SET ")
	for i, column := range columns {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(quoteIdent(column.Name))
		b.WriteString(" = NULL")
	}
	b.WriteString(" WHERE \"__ACTION__\" = ")
	b.WriteString(strconv.Itoa(action))
	db := warehouse.openDB()
	_, err := db.ExecContext(ctx, b.String())
	if err != nil {
		return snowflake(err)
	}
	return err
}

// connector returns a gosnowflake.Connector from the settings.
func (s *sfSettings) connector() gosnowflake.Connector {
	account := s.Account
	if i := strings.IndexByte(account, '.'); i > 0 {
		account = account[:i] + "-" + account[i+1:]
	}
	return gosnowflake.NewConnector(gosnowflake.SnowflakeDriver{}, gosnowflake.Config{
		Account:   account,
		User:      s.Username,
		Password:  s.Password,
		Database:  s.Database,
		Schema:    s.Schema,
		Warehouse: s.Warehouse,
		Role:      s.Role,
		Params:    make(map[string]*string),
	})
}

// openDB opens a Snowflake database and returns it.
func (warehouse *Snowflake) openDB() *sql.DB {
	warehouse.mu.Lock()
	defer warehouse.mu.Unlock()
	if warehouse.db != nil {
		return warehouse.db
	}
	db := sql.OpenDB(warehouse.settings.connector())
	warehouse.db = db
	return db
}

// usersVersion returns the version of the "USERS" table.
func (warehouse *Snowflake) usersVersion(ctx context.Context) (int, error) {
	db := warehouse.openDB()
	var v int
	err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX("VERSION"), 0) FROM "_USER_SCHEMA_VERSIONS"`).Scan(&v)
	if err != nil {
		return 0, snowflake(err)
	}
	return v, nil
}

// execTransaction executes the function f within a transaction. If f returns an
// error or panics, the transaction will be rolled back.
func (warehouse *Snowflake) execTransaction(ctx context.Context, f func(*sql.Tx) error) error {
	// TODO(Gianluca): is the use of the context in this method correct?
	db := warehouse.openDB()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return snowflake(err)
	}
	defer tx.Rollback()
	err = f(tx)
	if err != nil {
		return snowflake(err)
	}
	err = tx.Commit()
	if err != nil && !errors.Is(err, sql.ErrTxDone) {
		return snowflake(err)
	}
	return nil
}

// serializeIdentitiesToCSV serializes identities as CSV, using columns as
// header, and returns it as an io.Reader. It also appends a boolean column
// called $PURGE with the value of the 'deleted' argument as value for each row.
func serializeIdentitiesToCSV(columns []meergo.Column, rows []map[string]any) (io.Reader, error) {
	var err error
	var b bytes.Buffer
	for i, c := range columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strings.ToUpper(c.Name))
	}
	b.WriteString(",$PURGE\n")
	for i, row := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		for j, column := range columns {
			if j > 0 {
				b.WriteByte(',')
			}
			v, ok := row[column.Name]
			if !ok {
				continue
			}
			err = serializeValueToCSV(&b, columns[j].Type, v)
			if err != nil {
				return nil, err
			}
		}
		// Add the value for the column $PURGE.
		if purge, ok := row["$PURGE"].(bool); ok {
			if purge {
				b.WriteString(",true")
			} else {
				b.WriteString(",false")
			}
		} else {
			b.WriteString(",")
		}
	}
	return &b, nil
}

// snowflake transforms Snowflake error messages into a more user-friendly,
// readable format. It must be called for each error returned by the underlying
// SQL driver.
func snowflake(err error) error {
	switch err := err.(type) {
	case *gosnowflake.SnowflakeError:
		switch err.Number {
		case 261004:
			err.Message = "Authentication failed. Ensure that the account identifier is correct."
			err.MessageArgs = nil
		}
	}
	return err
}
