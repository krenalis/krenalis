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
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"chichi/apis/datastore/warehouses"
	"chichi/apis/postgres"
	"chichi/connector/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	//go:embed destinations_users.sql
	createDestinationUsersTable string
	//go:embed stored_procedure_queries.sql
	storeProcedureQueries string
)

var _ warehouses.Warehouse = &PostgreSQL{}

type PostgreSQL struct {
	mu           sync.Mutex // for the db and closed fields
	db           *postgres.DB
	closed       bool
	settings     *psSettings
	tableInfos   map[string]tableInfo
	tableInfosMu sync.Mutex
}

// A tableInfo holds information about a table.
type tableInfo struct {
	schema types.Type
	fds    []pgconn.FieldDescription
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
func Open(settings []byte) (warehouses.Warehouse, error) {
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

// Close closes the warehouse. It will not allow any new queries to run, and it
// waits for the current ones to finish.
func (warehouse *PostgreSQL) Close() error {
	warehouse.mu.Lock()
	if warehouse.db != nil {
		warehouse.db.Close()
		warehouse.db = nil
		warehouse.closed = true
	}
	warehouse.mu.Unlock()
	return nil
}

// DestinationUser returns the external ID of the destination user of the action
// that matches with the corresponding property. If it cannot be found, then the
// empty string and false are returned.
func (warehouse *PostgreSQL) DestinationUser(ctx context.Context, action int, property string) (string, bool, error) {
	db, err := warehouse.connection()
	if err != nil {
		return "", false, err
	}
	rows, err := db.Query(ctx, `SELECT "user" FROM destinations_users WHERE action = $1 AND property = $2`, action, property)
	if err != nil {
		return "", false, warehouses.Error(err)
	}
	defer rows.Close()
	var externalID string
	for rows.Next() {
		if externalID != "" {
			// TODO(Gianluca): improve the handling of this error. This happens
			// when a property on the external app has the same value for more
			// than one user.
			return "", false, warehouses.Errorf("too many users matching when using property")
		}
		err := rows.Scan(&externalID)
		if err != nil {
			return "", false, warehouses.Error(err)
		}
	}
	rows.Close()
	if rows.Err() != nil {
		return "", false, warehouses.Error(err)
	}
	return externalID, externalID != "", nil
}

// Init initializes the data warehouse by creating the supporting tables.
func (warehouse *PostgreSQL) Init(ctx context.Context) error {
	conn, err := warehouse.connection()
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, createDestinationUsersTable)
	if err != nil {
		return warehouses.Error(err)
	}
	return nil
}

// Merge performs a table merge operation, handling row updates, inserts, and
// deletions. table specifies the target table for the merge operation, rows
// contains the rows to insert or update in the table, and deleted contains the
// key values of rows to delete, if they exist.
// rows or deleted can be empty but not both.
func (warehouse *PostgreSQL) Merge(ctx context.Context, table warehouses.MergeTable, rows [][]any, deleted []any) error {

	db, err := warehouse.connection()
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
	b.WriteString(`false AS "$deleted" FROM "`)
	b.WriteString(table.Name)
	b.WriteString("\"\nWITH NO DATA")
	_, err = db.Exec(ctx, b.String())
	if err != nil {
		return warehouses.Error(err)
	}
	defer func() {
		_, err := warehouse.db.Exec(ctx, `DROP TABLE "`+tempTableName+`"`)
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
		_, err = db.CopyFrom(ctx, postgres.Identifier{tempTableName}, columnNames, postgres.CopyFromRows(rows))
		if err != nil {
			return warehouses.Error(err)
		}
	}

	// Copy the rows to delete into the temporary table.
	if len(deleted) > 0 {
		columnNames := make([]string, len(table.PrimaryKeys)+1)
		for i, c := range table.PrimaryKeys {
			columnNames[i] = c.Name
		}
		columnNames[len(columnNames)-1] = "$deleted"
		rowSrc := newCopyForDeleteFrom(len(table.PrimaryKeys), deleted)
		_, err = db.CopyFrom(ctx, postgres.Identifier{tempTableName}, columnNames, rowSrc)
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
	for i, key := range table.PrimaryKeys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`d."`)
		b.WriteString(key.Name)
		b.WriteString(`" = s."`)
		b.WriteString(key.Name)
		b.WriteByte('"')
	}
	if len(rows) > 0 {
		b.WriteString("\nWHEN MATCHED AND s.\"$deleted\" IS NULL THEN\n  UPDATE SET ")
		i := 0
	Set:
		for _, c := range table.Columns {
			for _, key := range table.PrimaryKeys {
				if c.Name == key.Name {
					continue Set
				}
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
		b.WriteString("\nWHEN NOT MATCHED AND s.\"$deleted\" IS NULL THEN\n  INSERT (")
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
	_, err = db.Exec(ctx, b.String())
	if err != nil {
		return warehouses.Error(err)
	}

	return nil
}

// Ping checks whether the connection to the data warehouse is active and, if
// necessary, establishes a new connection.
func (warehouse *PostgreSQL) Ping(ctx context.Context) error {
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	err = db.Ping(ctx)
	if err != nil {
		return warehouses.Error(err)
	}
	return nil
}

// QueryRow executes a query that should return at most one row.
// If the query fails, it returns a *DataWarehouseError error.
func (warehouse *PostgreSQL) QueryRow(ctx context.Context, query string, args ...any) warehouses.Row {
	db, err := warehouse.connection()
	if err != nil {
		return warehouses.Row{Error: err}
	}
	row := db.QueryRow(ctx, query, args...)
	return warehouses.Row{Row: row}
}

// ResolveSyncUsers resolves and sync the users.
// actions holds the identifiers of the actions of the workspace and must
// always contain at least one action; identifiers are the columns of the
// 'users_identities' table which are identifiers, ordered by priority;
// usersColumns are the columns of the 'users' table which will be populated
// during the users synchronization.
func (warehouse *PostgreSQL) ResolveSyncUsers(ctx context.Context, actions []int, identifiersColumns, usersColumns []types.Property) error {

	if len(actions) == 0 {
		panic("invalid empty actions")
	}

	db, err := warehouse.connection()
	if err != nil {
		return err
	}

	var b strings.Builder

	// Delete the orphan user identities, which are the identities that belong
	// to actions that no longer exist.
	b.WriteString(`DELETE FROM "users_identities" WHERE "__action__" NOT IN (`)
	for i, action := range actions {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Itoa(action))
	}
	b.WriteByte(')')
	_, err = db.Exec(ctx, b.String())
	if err != nil {
		return warehouses.Error(err)
	}

	// Generate the SQL matching expression.
	var matchingExpr strings.Builder
	if len(identifiersColumns) > 0 {
		matchingExpr.WriteString("matching_func(")
		for i, ident := range identifiersColumns {
			if i > 0 {
				matchingExpr.WriteByte(',')
			}
			matchingExpr.WriteString(`i1."`)
			matchingExpr.WriteString(ident.Name)
			matchingExpr.WriteString(`"::text,i2."`)
			matchingExpr.WriteString(ident.Name)
			matchingExpr.WriteString(`"::text`)
		}
		matchingExpr.WriteString(")")
	} else {
		matchingExpr.WriteString("false")
	}

	// Generate the SQL queries that will perform the users synchronization.
	var usersSyncQueries strings.Builder
	usersSyncQueries.WriteString(`TRUNCATE users; INSERT INTO users (`)
	comma := false
	for _, c := range usersColumns {
		if c.Name == "id" {
			continue
		}
		if comma {
			usersSyncQueries.WriteByte(',')
		}
		usersSyncQueries.WriteByte('"')
		usersSyncQueries.WriteString(c.Name)
		usersSyncQueries.WriteByte('"')
		comma = true
	}
	usersSyncQueries.WriteString(") SELECT\n")
	comma = false
	for _, c := range usersColumns {
		if c.Name == "id" {
			continue
		}
		if comma {
			usersSyncQueries.WriteByte(',')
		}
		if c.Type.Kind() == types.ObjectKind {
			usersSyncQueries.WriteString(`(ARRAY_AGG(DISTINCT "`)
			usersSyncQueries.WriteString(c.Name)
			usersSyncQueries.WriteString(`"))[1] AS "`)
			usersSyncQueries.WriteString(c.Name)
			usersSyncQueries.WriteByte('"')
		} else {
			usersSyncQueries.WriteString(`MAX(DISTINCT "`)
			usersSyncQueries.WriteString(c.Name)
			usersSyncQueries.WriteString(`") AS "`)
			usersSyncQueries.WriteString(c.Name)
			usersSyncQueries.WriteByte('"')
		}
		comma = true
	}
	usersSyncQueries.WriteString(" FROM users_identities GROUP BY __cluster__")

	// Replace the placeholders in the stored procedure query and run it.
	query := strings.Replace(storeProcedureQueries, "{{ matching_expr }}", matchingExpr.String(), 1)
	query = strings.Replace(query, "{{ users_sync_queries }}", usersSyncQueries.String(), 1)
	_, err = warehouse.db.Exec(ctx, query)
	if err != nil {
		return warehouses.Error(err)
	}

	// Call the 'resolve_sync_users' stored procedure, which performs the
	// identity resolution and updates the 'users' table.
	_, err = db.Exec(ctx, "CALL resolve_sync_users()")
	if err != nil {
		return warehouses.Error(err)
	}

	return nil
}

// tableInfo returns the table info for the table. The 'fresh' parameter
// controls whether the returned 'tableInfo' should be reloaded from the
// database or if it is not necessary.
func (warehouse *PostgreSQL) tableInfo(ctx context.Context, table string, fresh bool) (tableInfo, error) {

	// Determine if there is the need to refresh.
	warehouse.tableInfosMu.Lock()
	fresh = fresh || warehouse.tableInfos == nil
	warehouse.tableInfosMu.Unlock()

	// Read a fresh tableInfos, if necessary.
	if fresh {
		tables, err := warehouse.tables(ctx)
		if err != nil {
			return tableInfo{}, err
		}
		tableInfos := map[string]tableInfo{}
		for _, table := range tables {
			props, err := warehouses.ColumnsToProperties(table.columns)
			if err != nil {
				return tableInfo{}, err
			}
			schema, err := types.ObjectOf(props)
			if err != nil {
				return tableInfo{}, err
			}
			tableInfos[table.name] = tableInfo{
				schema: schema,
				fds:    table.fds,
			}
		}
		warehouse.tableInfosMu.Lock()
		warehouse.tableInfos = tableInfos
		warehouse.tableInfosMu.Unlock()
	}

	// Take the tableInfo for the given table.
	warehouse.tableInfosMu.Lock()
	ti, ok := warehouse.tableInfos[table]
	warehouse.tableInfosMu.Unlock()
	if !ok {
		// TODO(Gianluca): see the issue https://github.com/open2b/chichi/issues/413.
		return tableInfo{}, fmt.Errorf("schema '%s' not loaded from data warehouse", table)
	}

	return ti, nil
}

// SetDestinationUser sets the destination user relative to the action, with the
// given external user ID and external property.
func (warehouse *PostgreSQL) SetDestinationUser(ctx context.Context, action int, externalUserID, externalProperty string) error {
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, "INSERT INTO destinations_users (action, \"user\", property)\n"+
		"VALUES ($1, $2, $3)\n"+
		"ON CONFLICT (action, \"user\") DO UPDATE SET property = $3",
		action, externalUserID, externalProperty)
	if err != nil {
		return warehouses.Error(err)
	}
	return nil
}

// Settings returns the data warehouse settings.
func (warehouse *PostgreSQL) Settings() []byte {
	s, _ := json.Marshal(warehouse.settings)
	return s
}

// Tables returns the tables of the data warehouse.
// It returns only the tables 'users', 'users_identities', 'groups',
// 'groups_identities' and 'events'.
func (warehouse *PostgreSQL) Tables(ctx context.Context) ([]*warehouses.Table, error) {
	tables, err := warehouse.tables(ctx)
	if err != nil {
		return nil, err
	}
	whTables := make([]*warehouses.Table, len(tables))
	for i, t := range tables {
		props, err := warehouses.ColumnsToProperties(t.columns)
		if err != nil {
			return nil, warehouses.Error(err)
		}
		schema, err := types.ObjectOf(props)
		if err != nil {
			return nil, warehouses.Error(err)
		}
		whTables[i] = &warehouses.Table{
			Name:   t.name,
			Schema: schema,
		}
	}
	return whTables, nil
}

func (warehouse *PostgreSQL) tables(ctx context.Context) ([]*tableSchema, error) {
	// Get the connection.
	db, err := warehouse.connection()
	if err != nil {
		return nil, err
	}

	// Read the table schemas.
	tx, err := db.UnderlyingPool().Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	tables, err := tablesSchemas(ctx, tx, warehouse.settings.Schema,
		[]string{"users", "users_identities", "groups", "groups_identities", "events"})
	if err != nil {
		return nil, warehouses.Error(err)
	}
	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}
	return tables, nil
}

// connection returns the database connection.
func (warehouse *PostgreSQL) connection() (*postgres.DB, error) {
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
	db, err := postgres.Open(warehouse.settings.options())
	if err != nil {
		return nil, warehouses.Error(err)
	}
	warehouse.db = db
	return db, nil
}

// dsn returns the connection string, from s, in the URL format.
func (s *psSettings) options() *postgres.Options {
	return &postgres.Options{
		Host:     s.Host,
		Port:     s.Port,
		Username: s.Username,
		Password: s.Password,
		Database: s.Database,
	}
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

func (c *copyForDeleteFrom) Err() error {
	return nil
}
