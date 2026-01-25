// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
	_ "embed"
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

	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/prometheus"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	//go:embed tables/destination_profiles.sql
	createDestinationProfilesTable string
	//go:embed tables/events.sql
	createEventsTable string
	//go:embed tables/system_operations.sql
	createSystemOperationsTable string
	//go:embed tables/profile_schema_versions.sql
	createProfileSchemaVersionsTable string
)

var _ warehouses.Warehouse = &PostgreSQL{}

func init() {
	warehouses.Register(warehouses.Platform{
		Name: "PostgreSQL",
	}, New)
}

// New returns a new PostgreSQL data warehouse instance.
// It returns a *warehouses.SettingsError if the settings are not valid.
func New(conf *warehouses.Config) (*PostgreSQL, error) {
	var s psSettings
	err := json.Unmarshal(conf.Settings, &s)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal settings: %s", err)
	}
	// Validate Host.
	if n := len(s.Host); n == 0 || n > 253 {
		return nil, warehouses.SettingsErrorf("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65535 {
		return nil, warehouses.SettingsErrorf("port must be in range [1,65535]")
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
	return &PostgreSQL{conf: conf, settings: &s}, nil
}

type PostgreSQL struct {
	mu       sync.Mutex // for the pool field
	pool     *pgxpool.Pool
	conf     *warehouses.Config
	settings *psSettings
}

type psSettings struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	Schema   string `json:"schema"`
}

// CheckReadOnlyAccess checks that the warehouse access is read-only, returning
// a *warehouses.SettingsNotReadOnly error in case it is not, which may contain
// additional details.
func (warehouse *PostgreSQL) CheckReadOnlyAccess(ctx context.Context) error {

	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return err
	}

	// Define the privileges not allowed for a read-only user (in the format for
	// the 'has_table_privilege' function).
	const disallowedPrivileges = `INSERT,UPDATE,DELETE,TRUNCATE`

	// Retrieve the profiles table version.
	profileSchemaVersion, err := warehouse.profilesVersion(ctx)
	if err != nil {
		return err
	}

	// Determine if there are tables on the data warehouse for which the current
	// user has too many privileges.
	tables := []string{
		"meergo_destination_profiles",
		"meergo_system_operations",
		"meergo_identities",
		"meergo_profile_schema_versions",
		"meergo_events",
		fmt.Sprintf("meergo_profiles_%d", profileSchemaVersion),
	}
	var canWriteOnTable []any
	for range len(tables) {
		var b bool
		canWriteOnTable = append(canWriteOnTable, &b)
	}
	var query strings.Builder
	query.WriteString("SELECT ")
	for i, table := range tables {
		if i > 0 {
			query.WriteByte(',')
		}
		query.WriteString("has_table_privilege(")
		quoteString(&query, table)
		query.WriteString(", '" + disallowedPrivileges + "')")
	}
	err = pool.QueryRow(ctx, query.String()).Scan(canWriteOnTable...)
	if err != nil {
		return err
	}
	var tooPrivilegedTableNames []string
	for i, canWrite := range canWriteOnTable {
		if *canWrite.(*bool) {
			tooPrivilegedTableNames = append(tooPrivilegedTableNames, tables[i])
		}
	}
	if len(tooPrivilegedTableNames) > 0 {
		return &warehouses.SettingsNotReadOnly{
			Err: fmt.Errorf(
				"the credentials should be read-only, but they allow write operations on the following Meergo tables: %s",
				strings.Join(tooPrivilegedTableNames, ", "),
			)}
	}

	return nil
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

// ColumnTypeDescription returns a description for the warehouse column type
// corresponding to the given types.Type.
func (warehouse *PostgreSQL) ColumnTypeDescription(t types.Type) (string, error) {
	return typeToPostgresType(t), nil
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
func (warehouse *PostgreSQL) Merge(ctx context.Context, table warehouses.Table, rows [][]any, deleted []any) error {
	prometheus.Increment("warehouses.PostgreSQL.Merge.calls", 1)
	prometheus.Increment("warehouses.PostgreSQL.Merge.passed_rows", len(rows))
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
	"_pipeline",
	"_identity_id",
	"_is_anonymous",
	"_connection",
}

// MergeIdentities merges existing identities, deletes them, and inserts new
// ones.
func (warehouse *PostgreSQL) MergeIdentities(ctx context.Context, columns []warehouses.Column, rows []map[string]any) error {

	prometheus.Increment("warehouses.PostgreSQL.MergeIdentities.calls", 1)
	prometheus.Increment("warehouses.PostgreSQL.MergeIdentities.passed_rows", len(rows))

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
	b.WriteString(`FALSE AS "$purge" FROM "meergo_identities"` + "\n" + `WITH NO DATA`)
	create := b.String()

	// Prepare the "merge" statement.
	b.Reset()
	b.WriteString("MERGE INTO \"meergo_identities\" AS \"d\"\nUSING \"")
	b.WriteString(tempTableName)
	b.WriteString(`" AS "s"` + "\n" + `ON "d"."_pipeline" = "s"."_pipeline" AND "d"."_identity_id" = "s"."_identity_id" AND "d"."_is_anonymous" = "s"."_is_anonymous"`)
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
		if c.Name == "_anonymous_ids" {
			b.WriteString(`CASE WHEN "s"."_anonymous_ids" IS NULL OR "s"."_anonymous_ids"[1] = ANY("d"."_anonymous_ids") THEN "d"."_anonymous_ids" ELSE "d"."_anonymous_ids" || "s"."_anonymous_ids"[1] END`)
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
	b.WriteString(")\nWHEN MATCHED AND \"s\".\"$purge\" = FALSE THEN\n  UPDATE SET \"_run\" = \"s\".\"_run\"")
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
func (warehouse *PostgreSQL) Settings() json.Value {
	s, _ := json.Marshal(warehouse.settings)
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
// given pipeline.
func (warehouse *PostgreSQL) UnsetIdentityColumns(ctx context.Context, pipeline int, columns []warehouses.Column) error {
	var b strings.Builder
	b.WriteString("UPDATE \"meergo_identities\" SET ")
	for i, column := range columns {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(quoteIdent(column.Name))
		b.WriteString(" = NULL")
	}
	b.WriteString(" WHERE \"_pipeline\" = ")
	b.WriteString(strconv.Itoa(pipeline))
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

// profilesVersion returns the version of the "meergo_profiles" table.
func (warehouse *PostgreSQL) profilesVersion(ctx context.Context) (int, error) {
	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return 0, err
	}
	var v int
	err = pool.QueryRow(ctx, `SELECT COALESCE(MAX("version"), 0) FROM "meergo_profile_schema_versions"`).Scan(&v)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// copyForIdentities implements the pgx.CopyFromSource interface.
type copyForIdentities struct {
	columns []warehouses.Column
	encoder *rowEncoder
	rows    []map[string]any
	row     []any
}

// newCopyForIdentities returns a pgx.CopyFromSource implementation used to copy
// identities to add and delete to a temporary identity table.
func newCopyForIdentities(columns []warehouses.Column, rows []map[string]any) pgx.CopyFromSource {
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
