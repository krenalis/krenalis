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
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/apis/datastore/warehouses"
	"github.com/meergo/meergo/apis/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	//go:embed tables/destinations_users.sql
	createDestinationUsersTable string
	//go:embed tables/events.sql
	createEventsTable string
	//go:embed tables/user_identities.sql
	createUserIdentitiesTable string
	//go:embed tables/users.sql
	createUsersTable string
)

var _ warehouses.Warehouse = &PostgreSQL{}

type PostgreSQL struct {
	mu       sync.Mutex // for the db field
	db       *postgres.DB
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

// Close closes the data warehouse.
func (warehouse *PostgreSQL) Close() error {
	if warehouse.db == nil {
		return nil
	}
	warehouse.db.Close()
	warehouse.db = nil
	return nil
}

// DestinationUsers returns the destination users of the action.
func (warehouse *PostgreSQL) DestinationUsers(ctx context.Context, action int, propertyValue string) ([]string, error) {
	db, err := warehouse.connection()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, `SELECT "user" FROM _destinations_users WHERE action = $1 AND property = $2`, action, propertyValue)
	if err != nil {
		return nil, warehouses.Error(err)
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			return nil, warehouses.Error(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, warehouses.Error(err)
	}
	return ids, nil
}

// DuplicatedDestinationUsers retrieves duplicated destination users.
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
			FROM _destinations_users
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

// DuplicatedUsers returns the GIDs of two duplicated users.
func (warehouse *PostgreSQL) DuplicatedUsers(ctx context.Context, column string) (uuid.UUID, uuid.UUID, bool, error) {
	db, err := warehouse.connection()
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, false, err
	}
	query := `SELECT gid1, gid2
		FROM (
			SELECT
				min("__id__"::text) AS gid1,
				max("__id__"::text) as gid2,
				count(*) AS cnt
			FROM _users
			GROUP BY "` + column + `") AS subquery
		WHERE subquery.cnt > 1
		LIMIT 1`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, false, warehouses.Error(err)
	}
	defer rows.Close()
	var gid1, gid2 uuid.UUID
	var found bool
	for rows.Next() {
		err := rows.Scan(&gid1, &gid2)
		if err != nil {
			return uuid.UUID{}, uuid.UUID{}, false, warehouses.Error(err)
		}
		found = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return uuid.UUID{}, uuid.UUID{}, false, warehouses.Error(err)
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
		createUserIdentitiesTable,
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

// Merge performs a table merge operation.
func (warehouse *PostgreSQL) Merge(ctx context.Context, table warehouses.MergeTable, rows [][]any, deleted []any) error {

	db, err := warehouse.connection()
	if err != nil {
		return err
	}

	var b strings.Builder

	// Determine the table name.
	tableName := table.Name
	if tableName == "users" {
		// Change the table name from "users" to "_users" because the PostgreSQL
		// driver has a view called "users", with columns sorted according to
		// the schema, while the actual table is called "_users".
		tableName = "_users"
	}

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
	b.WriteString(`false AS "$purge" FROM "`)
	b.WriteString(tableName)
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
		columnNames := make([]string, len(table.Keys)+1)
		for i, c := range table.Keys {
			columnNames[i] = c.Name
		}
		columnNames[len(columnNames)-1] = "$purge"
		rowSrc := newCopyForDeleteFrom(len(table.Keys), deleted)
		_, err = db.CopyFrom(ctx, postgres.Identifier{tempTableName}, columnNames, rowSrc)
		if err != nil {
			return warehouses.Error(err)
		}
	}

	// Merge the temporary table's rows with the destination table's rows.
	b.Reset()
	b.WriteString(`MERGE INTO "`)
	b.WriteString(tableName)
	b.WriteString("\" d\nUSING \"")
	b.WriteString(tempTableName)
	b.WriteString("\" s\nON ")
	for i, key := range table.Keys {
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
		b.WriteString("\nWHEN MATCHED AND s.\"$purge\" IS NULL THEN\n  UPDATE SET ")
		i := 0
	Set:
		for _, c := range table.Columns {
			for _, key := range table.Keys {
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
	_, err = db.Exec(ctx, b.String())
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
	for _, c := range columns {
		b.WriteByte('"')
		b.WriteString(c.Name)
		b.WriteString(`",`)
	}
	b.WriteString("FALSE AS \"$purge\" FROM \"_user_identities\"\nWITH NO DATA")
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
	columnNames := make([]string, len(columns)+1)
	for i, c := range columns {
		columnNames[i] = c.Name
	}
	columnNames[len(columns)] = `$purge`
	_, err = db.CopyFrom(ctx, postgres.Identifier{tempTableName}, columnNames, newCopyForIdentities(columns, rows))
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
	_, err = db.Exec(ctx, b.String())
	if err != nil {
		return warehouses.Error(err)
	}

	return nil
}

// Ping checks the connection to the data warehouse.
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

// PurgeIdentities purges identities associated with the provided actions.
func (warehouse *PostgreSQL) PurgeIdentities(ctx context.Context, actions []int, execution int) error {
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString(`DELETE FROM "_user_identities" WHERE "__action__" IN (`)
	for i, action := range actions {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Itoa(action))
	}
	b.WriteByte(')')
	if execution != 0 {
		b.WriteString(` AND "__execution__" != `)
		b.WriteString(strconv.Itoa(execution))
	}
	_, err = db.Exec(ctx, b.String())
	if err != nil {
		return warehouses.Error(err)
	}
	return nil
}

// SetDestinationUser sets the destination user for an action.
func (warehouse *PostgreSQL) SetDestinationUser(ctx context.Context, action int, externalUserID, externalProperty string) error {
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, "INSERT INTO _destinations_users (action, \"user\", property)\n"+
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

// connection returns the PostgreSQL connection.
func (warehouse *PostgreSQL) connection() (*postgres.DB, error) {
	warehouse.mu.Lock()
	defer warehouse.mu.Unlock()
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

func (c *copyForIdentities) Err() error {
	return nil
}
