// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	_ "embed"
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

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"

	"github.com/snowflakedb/gosnowflake/v2"
)

var (
	//go:embed tables/destination_profiles.sql
	createDestinationProfilesTable string
	//go:embed tables/events.sql
	createEventsTable string
	//go:embed tables/system_operations.sql
	createOperationsTable string
	//go:embed tables/profile_schema_versions.sql
	createProfileSchemaVersionTable string
)

var _ warehouses.Warehouse = &Snowflake{}

func init() {
	warehouses.Register(warehouses.Platform{
		Name: "Snowflake",
	}, New)
}

// New returns a new Snowflake data warehouse instance.
func New(settings warehouses.SettingsLoader) *Snowflake {
	return &Snowflake{settings: settings}
}

type Snowflake struct {
	mu       sync.Mutex // for the db field
	db       *sql.DB
	settings warehouses.SettingsLoader
}

type sfSettings struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	Account   string `json:"account"`
	Warehouse string `json:"warehouse"`
	Database  string `json:"database"`
	Schema    string `json:"schema"`
	Role      string `json:"role"`
}

// CheckReadOnlyAccess checks that the warehouse access is read-only, returning
// a *warehouses.SettingsNotReadOnly error in case it is not, which may contain
// additional details.
func (warehouse *Snowflake) CheckReadOnlyAccess(ctx context.Context) error {
	// TODO(Gianluca): see the issue https://github.com/krenalis/krenalis/issues/1693.
	return errors.New("the read-only access check is currently not implemented in Snowflake")
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

// ColumnTypeDescription returns a description for the warehouse column type
// corresponding to the given types.Type.
func (warehouse *Snowflake) ColumnTypeDescription(t types.Type) (string, error) {
	return typeToSnowflakeType(t), nil
}

// Delete deletes rows from the specified table that match the provided where
// expression.
func (warehouse *Snowflake) Delete(ctx context.Context, table string, where warehouses.Expr) error {
	if where == nil {
		return errors.New("where is nil")
	}
	var s strings.Builder
	s.WriteString("DELETE FROM " + quoteIdent(table) + " WHERE ")
	err := renderExpr(&s, where)
	if err != nil {
		return fmt.Errorf("cannot build WHERE expression: %s", err)
	}
	db, err := warehouse.openDB(ctx)
	if err != nil {
		return snowflake(err)
	}
	_, err = db.ExecContext(ctx, s.String())
	if err != nil {
		return snowflake(err)
	}
	return nil
}

// Merge performs a table merge operation.
func (warehouse *Snowflake) Merge(ctx context.Context, table warehouses.Table, rows [][]any, deleted []any) error {
	// Acquire a connection.
	db, err := warehouse.openDB(ctx)
	if err != nil {
		return snowflake(err)
	}
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
	"_pipeline",
	"_identity_id",
	"_is_anonymous",
	"_connection",
}

// MergeIdentities merges existing identities, deletes them, and inserts new
// ones.
func (warehouse *Snowflake) MergeIdentities(ctx context.Context, columns []warehouses.Column, rows []map[string]any) error {

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
	b.WriteString(`FALSE AS "$PURGE" FROM "KRENALIS_IDENTITIES" LIMIT 0`)
	create := b.String()

	// Prepare the "merge" statement.
	b.Reset()
	b.WriteString("MERGE INTO \"KRENALIS_IDENTITIES\" AS \"D\"\nUSING \"")
	b.WriteString(tempTableName)
	b.WriteString(`" AS "S"` + "\n" + `ON "D"."_PIPELINE" = "S"."_PIPELINE" AND "D"."_IDENTITY_ID" = "S"."_IDENTITY_ID" AND "D"."_IS_ANONYMOUS" = "S"."_IS_ANONYMOUS"`)
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
		if c.Name == "_anonymous_ids" {
			b.WriteString(`CASE WHEN "S"."_ANONYMOUS_IDS" IS NULL OR ARRAY_CONTAINS("D"."_ANONYMOUS_IDS", "S"."_ANONYMOUS_IDS"[0]) THEN "D"."_ANONYMOUS_IDS" ELSE ARRAY_CAT("D"."_ANONYMOUS_IDS", ARRAY_CONSTRUCT("S"."_ANONYMOUS_IDS"[0])) END`)
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
	b.WriteString(")\nWHEN MATCHED AND \"S\".\"$PURGE\" = FALSE THEN\n  UPDATE SET \"_RUN\" = \"S\".\"_RUN\"")
	b.WriteString("\nWHEN MATCHED AND \"S\".\"$PURGE\" = TRUE THEN\n  DELETE")
	merge := b.String()

	// Serialize the rows in CSV format.
	csvReader, err := serializeIdentitiesToCSV(columns, rows)
	if err != nil {
		return err
	}

	// Acquire a connection.
	db, err := warehouse.openDB(ctx)
	if err != nil {
		return snowflake(err)
	}
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
		_, err = conn.ExecContext(gosnowflake.WithFilePutStream(ctx, csvReader), `PUT file://rows.csv @%"`+tempTableName+`"`)
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

// Truncate truncates the specified table.
func (warehouse *Snowflake) Truncate(ctx context.Context, table string) error {
	db, err := warehouse.openDB(ctx)
	if err != nil {
		return snowflake(err)
	}
	_, err = db.ExecContext(ctx, "TRUNCATE TABLE "+quoteIdent(table))
	if err != nil {
		return snowflake(err)
	}
	return nil
}

// UnsetIdentityColumns unsets values for the specified identity columns for the
// given pipeline.
func (warehouse *Snowflake) UnsetIdentityColumns(ctx context.Context, pipeline int, columns []warehouses.Column) error {
	var b strings.Builder
	b.WriteString("UPDATE \"KRENALIS_IDENTITIES\" SET ")
	for i, column := range columns {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(quoteIdent(column.Name))
		b.WriteString(" = NULL")
	}
	b.WriteString(" WHERE \"_PIPELINE\" = ")
	b.WriteString(strconv.Itoa(pipeline))
	db, err := warehouse.openDB(ctx)
	if err != nil {
		return snowflake(err)
	}
	_, err = db.ExecContext(ctx, b.String())
	if err != nil {
		return snowflake(err)
	}
	return nil
}

// ValidateSettings validates the settings.
func (warehouse *Snowflake) ValidateSettings(ctx context.Context) (json.Value, error) {
	var s sfSettings
	err := warehouse.settings.Load(ctx, &s)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal settings: %s", err)
	}
	err = validateSettings(&s)
	if err != nil {
		return nil, err
	}
	v, _ := json.Marshal(s)
	return v, nil
}

// execTransaction executes the function f within a transaction. If f returns an
// error or panics, the transaction will be rolled back.
func (warehouse *Snowflake) execTransaction(ctx context.Context, f func(*sql.Tx) error) error {
	// TODO(Gianluca): is the use of the context in this method correct?
	db, err := warehouse.openDB(ctx)
	if err != nil {
		return snowflake(err)
	}
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

// openDB opens a Snowflake database and returns it.
func (warehouse *Snowflake) openDB(ctx context.Context) (*sql.DB, error) {
	warehouse.mu.Lock()
	defer warehouse.mu.Unlock()
	if warehouse.db != nil {
		return warehouse.db, nil
	}
	var s sfSettings
	err := warehouse.settings.Load(ctx, &s)
	if err != nil {
		return nil, err
	}
	err = validateSettings(&s)
	if err != nil {
		return nil, err
	}
	db := sql.OpenDB(connector(&s))
	warehouse.db = db
	return db, nil
}

// profilesVersion returns the version of the "KRENALIS_PROFILES" table.
func (warehouse *Snowflake) profilesVersion(ctx context.Context) (int, error) {
	db, err := warehouse.openDB(ctx)
	if err != nil {
		return 0, snowflake(err)
	}
	var v int
	err = db.QueryRowContext(ctx, `SELECT COALESCE(MAX("VERSION"), 0) FROM "KRENALIS_PROFILE_SCHEMA_VERSIONS"`).Scan(&v)
	if err != nil {
		return 0, snowflake(err)
	}
	return v, nil
}

// accountFormat is the format of the account identifier in the settings.
var accountFormat = regexp.MustCompile(`^[a-zA-Z0-9]+[.-][a-zA-Z0-9]+$`)

// validateSettings validates the settings.
// It returns a *warehouses.SettingsError if the settings are not valid.
func validateSettings(s *sfSettings) error {
	// Validate Account.
	if n := utf8.RuneCountInString(s.Account); n < 3 || n > 255 {
		return warehouses.SettingsErrorf("account identifier length must be in range [3,255]")
	}
	if !accountFormat.MatchString(s.Account) {
		return warehouses.SettingsErrorf("account identifier must be in the <organization>.<account> or <organization>-<account> format")
	}
	// Validate Username.
	if n := utf8.RuneCountInString(s.Username); n < 1 || n > 255 {
		return warehouses.SettingsErrorf("user name length must be in range [1,255]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 255 {
		return warehouses.SettingsErrorf("password length must be in range [1,255]")
	}
	// Validate Role.
	if n := utf8.RuneCountInString(s.Role); n < 1 || n > 255 {
		return warehouses.SettingsErrorf("role length must be in range [1,255]")
	}
	// Validate Database.
	if n := utf8.RuneCountInString(s.Database); n < 1 || n > 255 {
		return warehouses.SettingsErrorf("database length must be in range [1,255]")
	}
	// Validate Schema.
	if n := utf8.RuneCountInString(s.Schema); n < 1 || n > 255 {
		return warehouses.SettingsErrorf("schema length must be in range [1,255]")
	}
	// Validate Warehouse.
	if n := utf8.RuneCountInString(s.Warehouse); n < 1 || n > 255 {
		return warehouses.SettingsErrorf("warehouse length must be in range [1,255]")
	}
	return nil
}

var falseStrPtr = new("false")

// connector returns a driver.Connector from the settings.
func connector(s *sfSettings) driver.Connector {
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
		Params: map[string]*string{
			"CLIENT_TELEMETRY_ENABLED": falseStrPtr,
		},
	})
}

// serializeIdentitiesToCSV serializes identities as CSV, using columns as
// header, and returns it as an io.Reader. It also appends a boolean column
// called $PURGE with the value of the 'deleted' argument as value for each row.
func serializeIdentitiesToCSV(columns []warehouses.Column, rows []map[string]any) (io.Reader, error) {
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
