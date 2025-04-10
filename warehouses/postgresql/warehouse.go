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
	jsonstd "encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/metrics"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connector icon.
var icon = "<svg></svg>"

var (
	//go:embed tables/destinations_users.sql
	createDestinationUsersTable string
	//go:embed tables/events.sql
	createEventsTable string
	//go:embed tables/operations2.sql
	createOperations2Table string
	//go:embed tables/user_schema_versions.sql
	createUserSchemaVersionTable string
)

var _ meergo.Warehouse = &PostgreSQL{}

func init() {
	meergo.RegisterWarehouseDriver(meergo.WarehouseDriver{
		Name: "PostgreSQL",
		Icon: icon,
	}, New)
}

// New returns a new PostgreSQL data warehouse driver instance.
// It returns a *meergo.WarehouseSettingsError if the settings are not valid.
func New(conf *meergo.WarehouseConfig) (*PostgreSQL, error) {
	var s psSettings
	err := jsonstd.Unmarshal(conf.Settings, &s)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal settings: %s", err)
	}
	// Validate Host.
	if n := len(s.Host); n == 0 || n > 253 {
		return nil, meergo.WarehouseSettingsErrorf("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65536 {
		return nil, meergo.WarehouseSettingsErrorf("port must be in range [1,65536]")
	}
	// Validate Username.
	if n := len(s.Username); n < 1 || n > 63 {
		return nil, meergo.WarehouseSettingsErrorf("username length in bytes must be in range [1,63]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 100 {
		return nil, meergo.WarehouseSettingsErrorf("password length must be in range [1,100]")
	}
	// Validate Database.
	if n := len(s.Database); n < 1 || n > 63 {
		return nil, meergo.WarehouseSettingsErrorf("database length in bytes must be in range [1,63]")
	}
	// Validate Schema.
	if n := len(s.Schema); n < 1 || n > 63 {
		return nil, meergo.WarehouseSettingsErrorf("schema length in bytes must be in range [1,63]")
	}
	if !meergo.IsValidSchemaName(s.Schema) {
		return nil, meergo.WarehouseSettingsErrorf("schema must start with [A-Za-z_] and subsequently contain only [A-Za-z0-9_]")
	}
	if strings.HasPrefix(s.Schema, "pg_") {
		return nil, meergo.WarehouseSettingsErrorf("schema cannot start with 'pg_'")
	}
	return &PostgreSQL{conf: conf, settings: &s}, nil
}

type PostgreSQL struct {
	mu       sync.Mutex // for the pool field
	pool     *pgxpool.Pool
	conf     *meergo.WarehouseConfig
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
func (warehouse *PostgreSQL) Delete(ctx context.Context, table string, where meergo.Expr) error {
	if where == nil {
		return errors.New("where is nil")
	}
	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return err
	}
	var s strings.Builder
	s.WriteString("DELETE FROM " + quoteIdent(table) + " WHERE ")
	err = renderExpr(&s, where)
	if err != nil {
		return fmt.Errorf("cannot build WHERE expression: %s", err)
	}
	_, err = pool.Exec(ctx, s.String())
	if err != nil {
		return err
	}
	return nil
}

// Merge performs a table merge operation.
func (warehouse *PostgreSQL) Merge(ctx context.Context, table meergo.Table, rows [][]any, deleted []any) error {
	metrics.Increment("warehouses.PostgreSQL.Merge.calls", 1)
	metrics.Increment("warehouses.PostgreSQL.Merge.passed_rows", len(rows))
	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return err
	}
	// Acquire a connection.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
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

// MergeIdentities merges existing identities, deletes them, and inserts new
// ones.
func (warehouse *PostgreSQL) MergeIdentities(ctx context.Context, columns []meergo.Column, rows []map[string]any) error {

	metrics.Increment("warehouses.PostgreSQL.MergeIdentities.calls", 1)
	metrics.Increment("warehouses.PostgreSQL.MergeIdentities.passed_rows", len(rows))

	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return err
	}

	quotedColumn := make(map[string]string, len(columns))
	for _, column := range columns {
		quotedColumn[column.Name] = quoteIdent(column.Name)
	}

	// Generate a unique name for the temporary table.
	tempTableName := "temp_table_" + strconv.FormatInt(time.Now().UnixNano(), 10)

	// Prepare the "create temporary table" statement.
	var b strings.Builder
	b.WriteString(`CREATE TEMPORARY TABLE "`)
	b.WriteString(tempTableName)
	b.WriteString("\" AS\n  SELECT ")
	for _, c := range columns {
		b.WriteString(quotedColumn[c.Name])
		b.WriteByte(',')
	}
	b.WriteString(`FALSE AS "$purge" FROM "_user_identities"` + "\n" + `WITH NO DATA`)
	create := b.String()

	// Prepare the "merge" statement.
	b.Reset()
	b.WriteString("MERGE INTO \"_user_identities\" AS \"d\"\nUSING \"")
	b.WriteString(tempTableName)
	b.WriteString(`" AS "s"` + "\n" + `ON "d"."__action__" = "s"."__action__" AND "d"."__identity_id__" = "s"."__identity_id__" AND "d"."__is_anonymous__" = "s"."__is_anonymous__"`)
	b.WriteString("\nWHEN MATCHED AND \"s\".\"$purge\" IS NULL THEN\n  UPDATE SET ")
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
			b.WriteString(`CASE WHEN "s"."__anonymous_ids__" IS NULL OR "s"."__anonymous_ids__"[1] = ANY("d"."__anonymous_ids__") THEN "d"."__anonymous_ids__" ELSE "d"."__anonymous_ids__" || "s"."__anonymous_ids__"[1] END`)
		} else {
			b.WriteString(`"s".`)
			b.WriteString(quotedColumn[c.Name])
		}
		i++
	}
	if i == 0 {
		return errors.New("postgresql.MergeIdentities: there must be at least one column in 'columns' apart from the immutable identities columns")
	}
	b.WriteString("\nWHEN NOT MATCHED AND \"s\".\"$purge\" IS NULL THEN\n  INSERT (")
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
		b.WriteString(`"s".`)
		b.WriteString(quotedColumn[c.Name])
	}
	b.WriteString(")\nWHEN MATCHED AND \"s\".\"$purge\" = FALSE THEN\n  UPDATE SET \"__execution__\" = \"s\".\"__execution__\"")
	b.WriteString("\nWHEN MATCHED AND \"s\".\"$purge\" = TRUE THEN\n  DELETE")
	merge := b.String()

	// Prepare the columns names.
	columnNames := make([]string, len(columns)+1)
	for i, c := range columns {
		columnNames[i] = c.Name
	}
	columnNames[len(columns)] = `$purge`

	// Acquire a connection.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	// Create the temporary table.
	_, err = conn.Exec(ctx, create)
	if err != nil {
		return err
	}
	// Copy the rows into the temporary table.
	_, err = conn.CopyFrom(ctx, []string{tempTableName}, columnNames, newCopyForIdentities(columns, rows))
	if err != nil {
		return err
	}
	// Merge the temporary table's rows with the destination table's rows.
	_, err = conn.Exec(ctx, merge)
	if err != nil {
		return err
	}

	return nil
}

// Settings returns the data warehouse settings.
func (warehouse *PostgreSQL) Settings() []byte {
	s, _ := jsonstd.Marshal(warehouse.settings)
	return s
}

// Truncate truncates the specified table.
func (warehouse *PostgreSQL) Truncate(ctx context.Context, table string) error {
	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, "TRUNCATE TABLE "+quoteIdent(table))
	return err
}

// UnsetIdentityColumns unsets values for the specified identity columns for the
// given action.
func (warehouse *PostgreSQL) UnsetIdentityColumns(ctx context.Context, action int, columns []meergo.Column) error {
	var b strings.Builder
	b.WriteString("UPDATE \"_user_identities\" SET ")
	for i, column := range columns {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(quoteIdent(column.Name))
		b.WriteString(" = NULL")
	}
	b.WriteString(" WHERE \"__action__\" = ")
	b.WriteString(strconv.Itoa(action))
	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, b.String())
	return err
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
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
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
		return err
	}
	defer tx.Rollback(ctx)
	err = f(tx)
	if err != nil {
		return err
	}
	err = tx.Commit(ctx)
	if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return err
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
	err = pool.QueryRow(ctx, `SELECT COALESCE(MAX("version"), 0) FROM "_user_schema_versions"`).Scan(&v)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// copyForIdentities implements the pgx.CopyFromSource interface.
type copyForIdentities struct {
	columns []meergo.Column
	encoder *rowEncoder
	rows    []map[string]any
	row     []any
}

// newCopyForIdentities returns a pgx.CopyFromSource implementation used to copy
// identities to add and delete to a temporary identity table.
func newCopyForIdentities(columns []meergo.Column, rows []map[string]any) pgx.CopyFromSource {
	c := &copyForIdentities{
		columns: columns,
		rows:    rows,
		row:     make([]any, len(columns)+1),
	}
	if enc, ok := newRowEncoder(columns); ok {
		c.encoder = enc
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
		c.row[i] = row[column.Name]
	}
	if c.encoder != nil {
		err := c.encoder.encode(c.row)
		if err != nil {
			return nil, err
		}
	}
	if purge, ok := row["$purge"].(bool); ok {
		c.row[len(c.row)-1] = purge
	} else {
		c.row[len(c.row)-1] = nil
	}
	c.rows = c.rows[1:]
	return c.row, nil
}
