//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package snowflake

import (
	"context"
	"database/sql"
	_ "embed"
	jsonstd "encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/snowflakedb/gosnowflake"

	"github.com/meergo/meergo"
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
	err := jsonstd.Unmarshal(conf.Settings, &s)
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
		reason := fmt.Sprintf("expected an empty database, got: %s", strings.Join(objects, ", "))
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
func (warehouse *Snowflake) Merge(ctx context.Context, table meergo.Table, rows [][]any, deleted []any) error {
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	// Acquire a connection.
	conn, err := db.Conn(ctx)
	if err != nil {
		return meergo.Error(err)
	}
	defer conn.Close()
	// Merge rows.
	return merge(ctx, conn, table, rows, deleted)
}

// immutableMergeIdentitiesColumns are columns in the merge of identities that
// are immutable.
var immutableMergeIdentitiesColumns = []string{
	"__action__",
	"__identity_id__",
	"__is_anonymous__",
	"__connection__",
}

// MergeIdentities merge existing identities, deletes them and inserts new ones.
func (warehouse *Snowflake) MergeIdentities(ctx context.Context, columns []meergo.Column, rows []map[string]any) error {

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
	for _, c := range columns {
		b.WriteByte('"')
		b.WriteString(c.Name)
		b.WriteString(`",`)
	}
	b.WriteString(`FALSE AS "$purge" FROM "_user_identities" LIMIT 0`)
	create := b.String()

	// Prepare the "merge" statement.
	b.Reset()
	b.WriteString("MERGE INTO \"_user_identities\" AS d\nUSING \"")
	b.WriteString(tempTableName)
	b.WriteString("\" AS s\nON d.\"__action__\" = s.\"__action__\" AND d.\"__identity_id__\" = s.\"__identity_id__\" AND d.\"__is_anonymous__\" = s.\"__is_anonymous__\"")
	b.WriteString("\nWHEN MATCHED AND s.\"$purge\" IS NULL THEN\n  UPDATE SET ")
	i := 0
	for _, c := range columns {
		if slices.Contains(immutableMergeIdentitiesColumns, c.Name) {
			continue
		}
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString("\n\"")
		b.WriteString(c.Name)
		b.WriteString(`" = `)
		if c.Name == "__anonymous_ids__" {
			b.WriteString(`CASE WHEN s."__anonymous_ids__" IS NULL OR ARRAY_CONTAINS(d."__anonymous_ids__", s."__anonymous_ids__"[0]) THEN d."__anonymous_ids__" ELSE ARRAY_CAT(d."__anonymous_ids__", ARRAY_CONSTRUCT(s."__anonymous_ids__"[0])) END`)
		} else {
			b.WriteString(`s."`)
			b.WriteString(c.Name)
			b.WriteString(`"`)
		}
		i++
	}
	if i == 0 {
		return errors.New("snowflake.MergeIdentities: there must be at least one column in 'columns' apart from the immutable identities columns")
	}
	b.WriteString("\nWHEN NOT MATCHED AND s.\"$purge\" IS NULL THEN\n  INSERT (")
	for i, c := range columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(c.Name)
		b.WriteByte('"')
	}
	b.WriteString(")\n  VALUES (")
	for i, c := range columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`s."`)
		b.WriteString(c.Name)
		b.WriteByte('"')
	}
	b.WriteString(")\nWHEN MATCHED AND s.\"$purge\" = FALSE THEN\n  UPDATE SET \"__execution__\" = s.\"__execution__\"")
	b.WriteString("\nWHEN MATCHED AND s.\"$purge\" = TRUE THEN\n  DELETE")
	merge := b.String()

	// Serialize the rows in CSV format.
	csvReader, err := serializeIdentitiesToCSV(columns, rows)
	if err != nil {
		return err
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
		_, err = conn.ExecContext(gosnowflake.WithFileStream(ctx, csvReader), `PUT file://rows.csv @%"`+tempTableName+`"`)
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

	// Merge the temporary table's rows with the destination table's rows.
	_, err = conn.ExecContext(ctx, merge)
	if err != nil {
		return meergo.Error(err)
	}

	return nil
}

// Repair repairs the database objects on the data warehouse needed by Meergo.
func (warehouse *Snowflake) Repair(ctx context.Context) error {
	return warehouse.initRepair(ctx, true)
}

// Settings returns the data warehouse settings.
func (warehouse *Snowflake) Settings() []byte {
	s, _ := jsonstd.Marshal(warehouse.settings)
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
