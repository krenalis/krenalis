//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package postgresql

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/apis/datastore/warehouses"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

var _ warehouses.Warehouse = &PostgreSQL{}

type PostgreSQL struct {
	mu       sync.Mutex // for the pool field
	pool     *pgxpool.Pool
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
// It returns a SettingsError error if the settings are not valid.
func Open(settings []byte) (*PostgreSQL, error) {
	var s psSettings
	err := json.Unmarshal(settings, &s)
	if err != nil {
		return nil, warehouses.SettingsErrorf("cannot unmarshal settings: %s", err)
	}
	// Validate Host.
	if n := len(s.Host); n == 0 || n > 253 {
		return nil, warehouses.SettingsErrorf("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65536 {
		return nil, warehouses.SettingsErrorf("port must be in range [1,65536]")
	}
	// Validate Username.
	if n := len(s.Username); n < 1 || n > 63 {
		return nil, warehouses.SettingsErrorf("username length in bytes must be in range [1,63]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 100 {
		return nil, warehouses.SettingsErrorf("password length must be in range [1,100]")
	}
	// Validate Database.
	if n := len(s.Database); n < 1 || n > 63 {
		return nil, warehouses.SettingsErrorf("database length in bytes must be in range [1,63]")
	}
	// Validate Schema.
	if n := len(s.Schema); n < 1 || n > 63 {
		return nil, warehouses.SettingsErrorf("schema length in bytes must be in range [1,63]")
	}
	if !warehouses.IsValidSchemaName(s.Schema) {
		return nil, warehouses.SettingsErrorf("schema must start with [A-Za-z_] and subsequently contain only [A-Za-z0-9_]")
	}
	if strings.HasPrefix(s.Schema, "pg_") {
		return nil, warehouses.SettingsErrorf("schema cannot start with 'pg_'")
	}
	return &PostgreSQL{settings: &s}, nil
}

// Close closes the data warehouse.
func (warehouse *PostgreSQL) Close() error {
	if warehouse.pool == nil {
		return nil
	}
	warehouse.pool.Close()
	warehouse.pool = nil
	return nil
}

// Delete deletes rows from the specified table that match the provided where
// expression.
func (warehouse *PostgreSQL) Delete(ctx context.Context, table string, where warehouses.Expr) error {
	if where == nil {
		return errors.New("where is nil")
	}
	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return err
	}
	var s strings.Builder
	s.WriteString(`DELETE FROM "` + table + `" WHERE `)
	err = renderExpr(&s, where)
	if err != nil {
		return fmt.Errorf("cannot build WHERE expression: %s", err)
	}
	_, err = pool.Exec(ctx, s.String())
	if err != nil {
		return warehouses.Error(err)
	}
	return nil
}

// Check checks if the necessary database objects on the data warehouse are
// correct to make Meergo work.
func (warehouse *PostgreSQL) Check(ctx context.Context) error {

	// Note that the checks here are partial, as full checks would require
	// reading the tables, their types, constraints, default values, etc... And
	// that would be very complex and broad. So, only basic checks are done
	// here.
	// Perhaps in the future, we will extend these checks.

	schema := warehouse.settings.Schema
	if schema == "" {
		schema = "public"
	}

	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return err
	}

	tables := []string{
		"_destinations_users",
		"_operations",
		"_user_identities",
		"_users_0",
		"events",
	}

	types := []string{
		"_operation",
		"event_browser_name",
		"event_os_name",
		"event_type",
	}

	missingDBObjects := map[string]struct{}{}

	// Check the existence of the tables.
	for _, table := range tables {
		var exists bool
		err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = $1 AND table_name = $2)`,
			schema, table).Scan(&exists)
		if err != nil {
			return warehouses.Error(err)
		}
		if !exists {
			missingDBObjects[table] = struct{}{}
		}
	}

	// Check the existence of types.
	for _, typ := range types {
		var exists bool
		err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT FROM pg_type WHERE typname = $1)`, typ).Scan(&exists)
		if err != nil {
			return warehouses.Error(err)
		}
		if !exists {
			missingDBObjects[typ] = struct{}{}
		}
	}

	// If there is any missing database object, return an
	// ErrDataWarehouseNotInitialized error (in case every object is missing,
	// meaning that the warehouse is not initialized yet) or a
	// DataWarehouseNeedsRepairError error, indicating which objects are missing
	// and thus needs to be repaired.
	if len(missingDBObjects) > 0 {
		if len(missingDBObjects) == len(tables)+len(types) {
			return warehouses.ErrDataWarehouseNotInitialized
		}
		toRepair := make([]string, 0, len(missingDBObjects))
		for decl := range missingDBObjects {
			toRepair = append(toRepair, fmt.Sprintf("database object %q not found", decl))
		}
		slices.Sort(toRepair)
		return warehouses.NewDataWarehouseNeedsRepairError(toRepair)
	}

	return nil
}

// Initialize initializes the database objects on the data warehouse in order to
// make it work with Meergo.
func (warehouse *PostgreSQL) Initialize(ctx context.Context) error {
	return warehouse.initRepair(ctx, false)
}

// Merge performs a table merge operation.
func (warehouse *PostgreSQL) Merge(ctx context.Context, table warehouses.Table, rows [][]any, deleted []any) error {

	pool, err := warehouse.connectionPool(ctx)
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
	b.WriteString(`FALSE AS "$purge" FROM "`)
	b.WriteString(table.Name)
	b.WriteString("\"\nWITH NO DATA")
	_, err = pool.Exec(ctx, b.String())
	if err != nil {
		return warehouses.Error(err)
	}
	defer func() {
		_, err := pool.Exec(ctx, `DROP TABLE "`+tempTableName+`"`)
		if err != nil {
			slog.Error("cannot drop temporary table from data warehouse", "table", tempTableName, "err", err)
		}
	}()

	// Copy the rows into the temporary table.
	if len(rows) > 0 {
		columnNames := make([]string, len(table.Columns))
		for i, c := range table.Columns {
			columnNames[i] = c.Name
		}
		_, err = pool.CopyFrom(ctx, []string{tempTableName}, columnNames, pgx.CopyFromRows(rows))
		if err != nil {
			return warehouses.Error(err)
		}
	}

	// Copy the rows to delete into the temporary table.
	if len(deleted) > 0 {
		columnNames := make([]string, len(table.Keys)+1)
		copy(columnNames, table.Keys)
		columnNames[len(columnNames)-1] = "$purge"
		rowSrc := newCopyForDeleteFrom(len(table.Keys), deleted)
		_, err = pool.CopyFrom(ctx, []string{tempTableName}, columnNames, rowSrc)
		if err != nil {
			return warehouses.Error(err)
		}
	}

	// Merge the temporary table's rows with the destination table's rows.
	b.Reset()
	b.WriteString(`MERGE INTO "`)
	b.WriteString(table.Name)
	b.WriteString("\" d\nUSING \"")
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
		b.WriteString("\nWHEN MATCHED AND s.\"$purge\" IS NULL THEN\n  UPDATE SET ")
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
		if i == 0 {
			return errors.New("postgresql.Merge: there must be at least one column in 'columns' apart from the keys")
		}
		b.WriteString("\nWHEN NOT MATCHED AND s.\"$purge\" IS NULL THEN\n  INSERT (")
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
	_, err = pool.Exec(ctx, b.String())
	if err != nil {
		return warehouses.Error(err)
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
func (warehouse *PostgreSQL) MergeIdentities(ctx context.Context, columns []warehouses.Column, rows []map[string]any) error {

	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return err
	}

	var b strings.Builder

	// Create the temporary table.
	tempTableName := "temp_table_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	b.WriteString(`CREATE UNLOGGED TABLE "`)
	b.WriteString(tempTableName)
	b.WriteString("\" AS\n  SELECT ")
	for _, c := range columns {
		b.WriteByte('"')
		b.WriteString(c.Name)
		b.WriteString(`",`)
	}
	b.WriteString("FALSE AS \"$purge\" FROM \"_user_identities\"\nWITH NO DATA")
	_, err = pool.Exec(ctx, b.String())
	if err != nil {
		return warehouses.Error(err)
	}
	defer func() {
		_, err := pool.Exec(ctx, `DROP TABLE "`+tempTableName+`"`)
		if err != nil {
			slog.Error("cannot drop temporary table from data warehouse", "table", tempTableName, "err", err)
		}
	}()

	// Copy the rows into the temporary table.
	columnNames := make([]string, len(columns)+1)
	for i, c := range columns {
		columnNames[i] = c.Name
	}
	columnNames[len(columns)] = `$purge`
	_, err = pool.CopyFrom(ctx, []string{tempTableName}, columnNames, newCopyForIdentities(columns, rows))
	if err != nil {
		return warehouses.Error(err)
	}

	// Merge the temporary table's rows with the destination table's rows.
	b.Reset()
	b.WriteString("MERGE INTO \"_user_identities\" AS d\nUSING \"")
	b.WriteString(tempTableName)
	b.WriteString("\" AS s\nON d.\"__action__\" = s.\"__action__\" AND d.\"__identity_id__\" = s.\"__identity_id__\" AND d.\"__is_anonymous__\" = s.\"__is_anonymous__\"")
	b.WriteString("\nWHEN MATCHED AND s.\"$purge\" IS NULL THEN\n  UPDATE SET ")
	j := 0
	for _, c := range columns {
		if slices.Contains(immutableMergeIdentitiesColumns, c.Name) {
			continue
		}
		if j > 0 {
			b.WriteByte(',')
		}
		b.WriteString("\n\"")
		b.WriteString(c.Name)
		b.WriteString(`" = `)
		if c.Name == "__anonymous_ids__" {
			b.WriteString(`CASE WHEN s."__anonymous_ids__" IS NULL OR s."__anonymous_ids__"[1] = ANY(d."__anonymous_ids__") THEN d."__anonymous_ids__" ELSE d."__anonymous_ids__" || s."__anonymous_ids__"[1] END`)
		} else {
			b.WriteString(`s."`)
			b.WriteString(c.Name)
			b.WriteString(`"`)
		}
		j++
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
	b.WriteString(")\nWHEN MATCHED AND s.\"$purge\" IS FALSE THEN\n  UPDATE SET \"__execution__\" = s.\"__execution__\"")
	b.WriteString("\nWHEN MATCHED AND s.\"$purge\" IS TRUE THEN\n  DELETE")
	_, err = pool.Exec(ctx, b.String())
	if err != nil {
		return warehouses.Error(err)
	}

	return nil
}

// Ping checks the connection to the data warehouse.
func (warehouse *PostgreSQL) Ping(ctx context.Context) error {
	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return err
	}
	err = pool.Ping(ctx)
	if err != nil {
		return warehouses.Error(err)
	}
	return nil
}

// Repair repairs the database objects on the data warehouse needed by Meergo.
func (warehouse *PostgreSQL) Repair(ctx context.Context) error {
	return warehouse.initRepair(ctx, true)
}

// Settings returns the data warehouse settings.
func (warehouse *PostgreSQL) Settings() []byte {
	s, _ := json.Marshal(warehouse.settings)
	return s
}

// Truncate truncates the specified table.
func (warehouse *PostgreSQL) Truncate(ctx context.Context, table string) error {
	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `TRUNCATE TABLE "`+table+`"`)
	if err != nil {
		return warehouses.Error(err)
	}
	return nil
}

// connection returns the PostgreSQL connection pool.
func (warehouse *PostgreSQL) connectionPool(ctx context.Context) (*pgxpool.Pool, error) {
	warehouse.mu.Lock()
	defer warehouse.mu.Unlock()
	if warehouse.pool != nil {
		return warehouse.pool, nil
	}
	s := warehouse.settings
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(s.Username, s.Password),
		Host:   net.JoinHostPort(s.Host, strconv.Itoa(s.Port)),
		Path:   "/" + url.PathEscape(s.Database),
	}
	if s.Schema != "" {
		u.RawQuery = "search_path=" + url.QueryEscape(s.Schema)
	}
	config, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		return nil, warehouses.Error(err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, warehouses.Error(err)
	}
	warehouse.pool = pool
	return pool, nil
}

// execTransaction executes the function f within a transaction. If f returns an
// error or panics, the transaction will be rolled back.
func (warehouse *PostgreSQL) execTransaction(ctx context.Context, f func(pgx.Tx) error) error {
	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return warehouses.Error(err)
	}
	defer tx.Rollback(ctx)
	err = f(tx)
	if err != nil {
		return warehouses.Error(err)
	}
	err = tx.Commit(ctx)
	if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return warehouses.Error(err)
	}
	return nil
}

// initRepair initializes (or repairs) the database objects (as tables, types,
// etc...) on the data warehouse.
func (warehouse *PostgreSQL) initRepair(ctx context.Context, repair bool) error {
	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return err
	}
	queries := []string{
		createDestinationUsersTable,
		createEventsTable,
		createOperationsTable,
		createUserIdentitiesTable,
		createUsersTable,
	}
	if !repair {
		// Since the "CREATE VIEW IF EXISTS" statement does not exist in
		// PostgreSQL, the view is recreated only if initializing, not when
		// repairing, otherwise a "cannot drop columns from view" error is
		// returned by PostgreSQL in cases where the users table has different
		// columns than the default one.
		queries = append(queries, createUsersView)
	}
	for _, query := range queries {
		_, err := pool.Exec(ctx, query)
		if err != nil {
			return warehouses.Error(err)
		}
	}
	return nil
}

// usersVersion returns the version of the "users" table.
func (warehouse *PostgreSQL) usersVersion(ctx context.Context) (int, error) {
	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return 0, err
	}
	var v int
	err = pool.QueryRow(ctx, "SELECT COALESCE(MAX(users_version), 0) FROM _operations").Scan(&v)
	if err != nil {
		return 0, warehouses.Error(err)
	}
	return v, nil
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

func (c *copyForDeleteFrom) Err() error {
	return nil
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

// copyForIdentities implements the pgx.CopyFromSource interface.
type copyForIdentities struct {
	columns []warehouses.Column
	rows    []map[string]any
	values  []any
}

// newCopyForIdentities returns a pgx.CopyFromSource implementation used to copy
// identities to add and delete to a temporary identity table.
func newCopyForIdentities(columns []warehouses.Column, rows []map[string]any) pgx.CopyFromSource {
	c := &copyForIdentities{
		columns: columns,
		rows:    rows,
		values:  make([]any, len(columns)+1),
	}
	return c
}

func (c *copyForIdentities) Err() error {
	return nil
}

func (c *copyForIdentities) Next() bool {
	return len(c.rows) > 0
}

func (c *copyForIdentities) Values() ([]any, error) {
	row := c.rows[0]
	for i, column := range c.columns {
		c.values[i] = row[column.Name]
	}
	if purge, ok := row["$purge"].(bool); ok {
		c.values[len(c.values)-1] = purge
	} else {
		c.values[len(c.values)-1] = nil
	}
	c.rows = c.rows[1:]
	return c.values, nil
}
