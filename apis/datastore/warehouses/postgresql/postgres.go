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
	"chichi/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	//go:embed tables/destinations_users.sql
	createDestinationUsersTable string
	//go:embed tables/events.sql
	createEventsTable string
	//go:embed tables/groups_identities.sql
	createGroupsIdentitiesTable string
	//go:embed tables/groups.sql
	createGroupsTable string
	//go:embed tables/users_identities.sql
	createUsersIdentitiesTable string
	//go:embed tables/users.sql
	createUsersTable string
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

// DestinationUsers returns the external IDs of the destination users of the
// action whose external matching property value matches with the given property
// value. If it cannot be found, then an empty slice and false are returned.
func (warehouse *PostgreSQL) DestinationUsers(ctx context.Context, action int, propertyValue string) ([]string, error) {
	db, err := warehouse.connection()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, `SELECT "user" FROM destinations_users WHERE action = $1 AND property = $2`, action, propertyValue)
	if err != nil {
		return nil, warehouses.Error(err)
	}
	defer rows.Close()
	externalIDs := []string{}
	for rows.Next() {
		var extID string
		err := rows.Scan(&extID)
		if err != nil {
			return nil, warehouses.Error(err)
		}
		externalIDs = append(externalIDs, extID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, warehouses.Error(err)
	}
	return externalIDs, nil
}

// DuplicatedDestinationUsers returns the external IDs of two users on the
// action which have the same value for the matching property, along with true.
//
// If there are no users on the action matching this condition, no external IDs
// are returned and the returned boolean is false. If an error occurs with the
// data warehouse, it returns a *DataWarehouseError error.
func (warehouse *PostgreSQL) DuplicatedDestinationUsers(ctx context.Context, action int) (string, string, bool, error) {
	db, err := warehouse.connection()
	if err != nil {
		return "", "", false, err
	}
	query := `SELECT user1, user2
		FROM (
			SELECT
				min("user") AS user1,
				max("user") as user2,
				count(*) AS cnt
			FROM destinations_users
			WHERE action = $1 
			GROUP BY property) AS subquery
		WHERE subquery.cnt > 1
		LIMIT 1`
	rows, err := db.Query(ctx, query, action)
	if err != nil {
		return "", "", false, warehouses.Error(err)
	}
	defer rows.Close()
	var user1, user2 string
	var found bool
	for rows.Next() {
		err := rows.Scan(&user1, &user2)
		if err != nil {
			return "", "", false, warehouses.Error(err)
		}
		found = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return "", "", false, warehouses.Error(err)
	}
	return user1, user2, found, nil
}

// DuplicatedUsers returns the GIDs of two users which have the same value for
// the given property, along with true.
// If there are no users matching this condition, no GIDs are returned and the
// returned boolean is false.
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error.
func (warehouse *PostgreSQL) DuplicatedUsers(ctx context.Context, property string) (int, int, bool, error) {
	db, err := warehouse.connection()
	if err != nil {
		return 0, 0, false, err
	}
	column := warehouses.PropertyNameToColumnName(property)
	query := `SELECT gid1, gid2
		FROM (
			SELECT
				min("_id") AS gid1,
				max("_id") as gid2,
				count(*) AS cnt
			FROM users
			GROUP BY "` + column + `") AS subquery
		WHERE subquery.cnt > 1
		LIMIT 1`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return 0, 0, false, warehouses.Error(err)
	}
	defer rows.Close()
	var gid1, gid2 int
	var found bool
	for rows.Next() {
		err := rows.Scan(&gid1, &gid2)
		if err != nil {
			return 0, 0, false, warehouses.Error(err)
		}
		found = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, false, warehouses.Error(err)
	}
	return gid1, gid2, found, nil
}

// Init initializes the data warehouse by creating the supporting tables.
func (warehouse *PostgreSQL) Init(ctx context.Context) error {
	conn, err := warehouse.connection()
	if err != nil {
		return err
	}
	tables := []string{
		createDestinationUsersTable,
		createEventsTable,
		createGroupsIdentitiesTable,
		createGroupsTable,
		createUsersIdentitiesTable,
		createUsersTable,
	}
	for _, table := range tables {
		_, err := conn.Exec(ctx, table)
		if err != nil {
			return warehouses.Error(err)
		}
	}
	return nil
}

// Merge performs a table merge operation, handling row updates, inserts, and
// deletions. table specifies the target table for the merge operation, rows
// contains the rows to insert or update in the table, and deleted contains the
// key values of rows to delete, if they exist.
// rows or deleted can be empty but not both, and both may be changed by this
// method.
// If the table schema is not conform to the schema of the table on the data
// warehouse, this method returns a *SchemaError error.
func (warehouse *PostgreSQL) Merge(ctx context.Context, table warehouses.MergeTable, rows []map[string]any, deleted map[string]any) error {

	db, err := warehouse.connection()
	if err != nil {
		return err
	}

	tableSchema := types.Object(table.Properties)
	primaryKeysSchema := types.Object(table.PrimaryKeys)

	columns := warehouses.PropertiesToColumns(table.Properties)
	primaryKeysColumns := warehouses.PropertiesToColumns(table.PrimaryKeys)

	// Check the conformity of the passed table schema with the schema of the
	// table on the data warehouse.
	ti, err := warehouse.tableInfo(ctx, table.Name, false)
	if err != nil {
		return warehouses.Error(err)
	}
	err = warehouses.CheckConformity("", tableSchema, ti.schema)
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
		columnNames := make([]string, len(columns))
		for i, c := range columns {
			columnNames[i] = c.Name
		}
		rowsSlice := serializeRowsToSlice(rows, tableSchema, columns)
		_, err = db.CopyFrom(ctx, postgres.Identifier{tempTableName}, columnNames, postgres.CopyFromRows(rowsSlice))
		if err != nil {
			return warehouses.Error(err)
		}
	}

	// Copy the rows to delete into the temporary table.
	if len(deleted) > 0 {
		columnNames := make([]string, len(primaryKeysColumns)+1)
		for i, c := range primaryKeysColumns {
			columnNames[i] = c.Name
		}
		columnNames[len(columnNames)-1] = "$deleted"
		deletedSlice := serializeRowToSlice(deleted, primaryKeysSchema, primaryKeysColumns)
		rowSrc := newCopyForDeleteFrom(len(primaryKeysColumns), deletedSlice)
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
	for i, key := range primaryKeysColumns {
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
		for _, c := range columns {
			for _, key := range primaryKeysColumns {
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

// RunWorkspaceIdentityResolution runs the Workspace Identity Resolution.
//
// connections holds the identifiers of the connections of the workspace and may
// be empty to indicate that no connections are present in the workspace.
//
// Identifiers are the Workspace Identity Resolution identifiers, ordered by
// priority.
//
// usersSchema is the "users" schema, as the "users" table on the data
// warehouse is rebuilt by this procedure.
func (warehouse *PostgreSQL) RunWorkspaceIdentityResolution(ctx context.Context, connections []int, identifiers []types.Property, usersSchema types.Type) error {

	db, err := warehouse.connection()
	if err != nil {
		return err
	}

	var b strings.Builder

	// Delete the orphan user identities, which are the identities that belong
	// to connections that no longer exist.
	if len(connections) == 0 {
		b.WriteString(`DELETE FROM "users_identities"`)
	} else {
		b.WriteString(`DELETE FROM "users_identities" WHERE "_connection" NOT IN (`)
		for i, connection := range connections {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(strconv.Itoa(connection))
		}
		b.WriteByte(')')
	}
	_, err = db.Exec(ctx, b.String())
	if err != nil {
		return warehouses.Error(err)
	}

	// Generate the SQL matching expression.
	identifiersColumns := warehouses.PropertiesToColumns(identifiers)
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
	usersColumns := warehouses.PropertiesToColumns(usersSchema.Properties())
	var usersSyncQueries strings.Builder
	usersSyncQueries.WriteString(`TRUNCATE users; INSERT INTO users (`)
	for _, c := range usersColumns {
		if c.Name == "_id" {
			continue
		}
		usersSyncQueries.WriteByte('"')
		usersSyncQueries.WriteString(c.Name)
		usersSyncQueries.WriteByte('"')
		usersSyncQueries.WriteByte(',')
	}
	usersSyncQueries.WriteString(`"__identity_ids__"`)
	usersSyncQueries.WriteString(") SELECT\n")
	for _, c := range usersColumns {
		if c.Name == "_id" {
			continue
		}
		usersSyncQueries.WriteString(`MAX(DISTINCT "`)
		usersSyncQueries.WriteString(c.Name)
		usersSyncQueries.WriteString(`") AS "`)
		usersSyncQueries.WriteString(c.Name)
		usersSyncQueries.WriteByte('"')
		usersSyncQueries.WriteByte(',')
	}
	usersSyncQueries.WriteString(`ARRAY_AGG(DISTINCT "_identity_id")`)
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
		return nil, err
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

func serializeRowToSlice(row map[string]any, schema types.Type, columns []types.Property) []any {
	warehouses.SerializeRow(row, schema)
	rr := make([]any, len(columns))
	for i, c := range columns {
		rr[i] = row[c.Name]
	}
	return rr
}

func serializeRowsToSlice(rows []map[string]any, schema types.Type, columns []types.Property) [][]any {
	rs := make([][]any, len(rows))
	for i, r := range rows {
		rs[i] = serializeRowToSlice(r, schema, columns)
	}
	return rs
}
