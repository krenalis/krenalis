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

	"chichi/apis/datastore/expr"
	"chichi/apis/datastore/warehouses"
	"chichi/apis/postgres"
	"chichi/connector/types"

	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/maps"
)

var (
	//go:embed destinations_users.sql
	createDestinationUsersTable string
	//go:embed stored_procedure_queries.sql
	storeProcedureQueries string
)

var _ warehouses.Warehouse = &PostgreSQL{}

type PostgreSQL struct {
	mu       sync.Mutex // for the db and closed fields
	db       *postgres.DB
	closed   bool
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

// Exec executes a query without returning any rows. args are the placeholders.
func (warehouse *PostgreSQL) Exec(ctx context.Context, query string, args ...any) (warehouses.Result, error) {
	db, err := warehouse.connection()
	if err != nil {
		return warehouses.Result{}, err
	}
	r, err := db.Exec(ctx, query, args...)
	if err != nil {
		return warehouses.Result{}, warehouses.Error(err)
	}
	return warehouses.Result{Result: r}, nil
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
// If the query fails, it returns a DataWarehouseError error.
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
		matchingExpr.WriteString("coalesce(matching_func(")
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
		matchingExpr.WriteString("), false)")
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
		if c.Type.PhysicalType() == types.PtObject {
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

// Select returns the rows from the given table that satisfies the where
// condition with only the given columns, ordered by order if order is not the
// zero Property, and in range [first,first+limit] with first >= 0 and
// 0 < limit <= 1000.
func (warehouse *PostgreSQL) Select(ctx context.Context, table string, columns []types.Property, where expr.Expr, order types.Property, first, limit int) ([][]any, error) {

	if !warehouses.IsValidIdentifier(table) {
		return nil, fmt.Errorf("table name %q is not a valid identifier", table)
	}

	db, err := warehouse.connection()
	if err != nil {
		return nil, err
	}

	// Build the query.
	var query strings.Builder
	query.WriteString(`SELECT `)
	for i, c := range columns {
		if i > 0 {
			query.WriteString(", ")
		}
		if !types.IsValidPropertyName(c.Name) {
			return nil, fmt.Errorf("column name %q is not a valid property name", c.Name)
		}
		switch c.Type.PhysicalType() {
		default:
			query.WriteByte('"')
			query.WriteString(c.Name)
			query.WriteByte('"')
		case types.PtInet:
			query.WriteString(`host("`)
			query.WriteString(c.Name)
			query.WriteString(`")`)
		}
	}
	query.WriteString(` FROM "`)
	query.WriteString(table)
	query.WriteByte('"')

	if where != nil {
		query.WriteString(` WHERE `)
		expr, err := renderExpr(where)
		if err != nil {
			return nil, fmt.Errorf("cannot build WHERE expression: %s", err)
		}
		query.WriteString(expr)
	}

	if order.Name != "" {
		if !types.IsValidPropertyName(order.Name) {
			return nil, fmt.Errorf("column name %q is not a valid property name", order.Name)
		}
		query.WriteString(" ORDER BY ")
		query.WriteString(order.Name)
	}
	query.WriteString(" LIMIT ")
	query.WriteString(strconv.Itoa(limit))
	if first > 0 {
		query.WriteString(" OFFSET ")
		query.WriteString(strconv.Itoa(first))
	}

	// Execute the query.
	rawRows, err := db.Query(ctx, query.String())
	if err != nil {
		return nil, warehouses.Error(err)
	}
	defer rawRows.Close()
	var rows [][]any
	values := newScanValues(columns, &rows)
	for rawRows.Next() {
		if err = rawRows.Scan(values...); err != nil {
			return nil, warehouses.Error(err)
		}
	}
	rawRows.Close()
	if err = rawRows.Err(); err != nil {
		return nil, warehouses.Error(err)
	}
	if rows == nil {
		rows = [][]any{}
	}

	return rows, nil
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

// SetIdentity sets the identity id (which may have an anonymous ID) imported from
// the action. fromEvents indicates if the identity has been imported from an
// event or not.
// timestamp is the timestamp that will be associated to the imported identity.
func (warehouse *PostgreSQL) SetIdentity(ctx context.Context, identity map[string]any, id string, anonID string, action int, fromEvent bool, timestamp time.Time) error {

	// Retrieve the database connection.
	db, err := warehouse.connection()
	if err != nil {
		return err
	}

	// Query the matching user identities, which can be 0 (the identity is a new
	// identity), 1 (the identity already exists and must be updated) or more
	// (the new identity requires a merging of already existing identities).
	var query string
	var args []any
	if fromEvent {
		if isAnon := id == ""; isAnon {
			query = "SELECT __identity_id__ FROM users_identities WHERE " +
				"$1 IN __anonymous_ids__ ORDER BY __timestamp__, __identity_id__"
			args = []any{anonID}
		} else {
			query = "SELECT __identity_id__ FROM users_identities WHERE " +
				"(__external_id__ = $1) OR ($2 = ANY(__anonymous_ids__)) ORDER BY __timestamp__, __identity_id__"
			args = []any{id, anonID}
		}
	} else { // app, file or database.
		query = "SELECT __identity_id__ FROM users_identities WHERE " +
			"__external_id__ = $1 ORDER BY __timestamp__, __identity_id__"
		args = []any{id}
	}
	var matchingIdentities []int
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return warehouses.Error(err)
	}
	defer rows.Close()
	for rows.Next() {
		var e int
		if err = rows.Scan(&e); err != nil {
			return warehouses.Error(err)
		}
		matchingIdentities = append(matchingIdentities, e)
	}
	rows.Close()
	if rows.Err() != nil {
		return warehouses.Error(err)
	}

	// Create the new identity.
	var newIdentityID int
	identity["__action__"] = action
	identity["__external_id__"] = id
	identity["__timestamp__"] = timestamp.Format(time.DateTime)
	if anonID != "" {
		identity["__anonymous_ids__"] = []string{anonID}
	}
	b := strings.Builder{}
	b.WriteString("INSERT INTO users_identities (")
	properties := maps.Keys(identity)
	for i, name := range properties {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(name)
		b.WriteByte('"')
	}
	b.WriteString(") VALUES (")
	values := make([]any, len(properties))
	for i, name := range properties {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('$')
		b.WriteString(strconv.Itoa(i + 1))
		values[i] = identity[name]
	}
	b.WriteString(") RETURNING __identity_id__")
	err = db.QueryRow(ctx, b.String(), values...).Scan(&newIdentityID)
	if err != nil {
		return warehouses.Error(err)
	}

	// There are no matching identities, so the identity has been created and
	// there's nothing else to do.
	if len(matchingIdentities) == 0 {
		return nil
	}

	// Merge the matching identity (or identities) into the new one.

	var idsStr strings.Builder
	for _, id := range matchingIdentities {
		idsStr.WriteString(strconv.Itoa(id))
		idsStr.WriteByte(',')
	}
	idsStr.WriteString(strconv.Itoa(newIdentityID))

	// Merge the anonymous IDS.
	b.Reset()
	b.WriteString(`UPDATE users_identities SET __anonymous_ids__ = (
		SELECT array_agg(anon_ids.ids) as __anonymous_ids__
		FROM (
			SELECT unnest("__anonymous_ids__") as ids
			FROM users_identities
			WHERE __identity_id__ IN (`)
	b.WriteString(idsStr.String())
	b.WriteString(`) AND __anonymous_ids__ IS NOT NULL
			) AS anon_ids
		) WHERE __identity_id__ = $1`)
	_, err = db.Exec(ctx, b.String(), newIdentityID)
	if err != nil {
		return warehouses.Error(err)
	}

	// Merge the other fields.
	b.Reset()
	b.WriteString("UPDATE users_identities SET ")
	comma := false
	for _, p := range properties {
		if p == "__action__" || p == "__anonymous_ids__" || p == "__external_id__" || p == "__timestamp__" {
			continue
		}
		if comma {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(p)
		b.WriteString(`" = (SELECT "`)
		b.WriteString(p)
		b.WriteString(`" FROM users_identities WHERE "`)
		b.WriteString(p)
		b.WriteString(`" IS NOT NULL AND __identity_id__ IN (`)
		b.WriteString(idsStr.String())
		b.WriteString(") ORDER BY __timestamp__ DESC, __identity_id__ DESC LIMIT 1)\n")
		comma = true
	}
	b.WriteString(` WHERE __identity_id__ = $1`)
	_, err = db.Exec(ctx, b.String(), newIdentityID)
	if err != nil {
		return warehouses.Error(err)
	}

	// Delete the merged identities.
	var idsToDelete strings.Builder
	for i, id := range matchingIdentities {
		if i > 0 {
			idsToDelete.WriteByte(',')
		}
		idsToDelete.WriteString(strconv.Itoa(id))
	}
	b.Reset()
	b.WriteString("DELETE FROM users_identities WHERE __identity_id__ IN (")
	b.WriteString(idsToDelete.String())
	b.WriteByte(')')
	_, err = db.Exec(ctx, b.String())
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
// It returns only the tables 'users', 'groups', 'events', and the tables with
// prefix 'users_', 'groups_' and 'events_'.
func (warehouse *PostgreSQL) Tables(ctx context.Context) ([]*warehouses.Table, error) {

	// Get the connection.
	db, err := warehouse.connection()
	if err != nil {
		return nil, err
	}

	var table *warehouses.Table
	var tables []*warehouses.Table

	err = db.Transaction(ctx, func(tx *postgres.Tx) error {

		// Read the available enums.
		query := "SELECT pg_type.typname, pg_enum.enumlabel FROM pg_type JOIN pg_enum ON pg_enum.enumtypid = pg_type.oid"
		rows, err := tx.Query(ctx, query)
		if err != nil {
			return warehouses.Error(err)
		}
		defer rows.Close()
		rawEnums := map[string][]string{}
		for rows.Next() {
			var typName, enumLabel string
			if err = rows.Scan(&typName, &enumLabel); err != nil {
				return warehouses.Error(err)
			}
			if typName == "" {
				return warehouses.Errorf("invalid empty enum name")
			}
			if !utf8.ValidString(enumLabel) {
				return warehouses.Errorf("not-valid UTF-8 encoded enum label for type %q", typName)
			}
			rawEnums[typName] = append(rawEnums[typName], enumLabel)
		}
		rows.Close()
		enums := map[string]types.Type{}
		for name, values := range rawEnums {
			enums[name] = types.Text().WithValues(values...)
		}
		if err := rows.Err(); err != nil {
			return warehouses.Error(err)
		}

		// Read the 'atttypmod' attribute of column types, where relevant.
		query = "SELECT c.relname, a.attname, a.atttypmod\n" +
			"FROM pg_attribute AS a\n" +
			"INNER JOIN pg_class AS c ON a.attrelid = c.oid\n" +
			"INNER JOIN pg_namespace AS n ON c.relnamespace = n.oid\n" +
			"WHERE n.nspname = '" + warehouse.settings.Schema + "' AND a.atttypmod <> -1"
		rows, err = tx.Query(ctx, query)
		if err != nil {
			return warehouses.Error(err)
		}
		defer rows.Close()
		attTypMods := map[string]map[string]*int{}
		for rows.Next() {
			var relname, attname string
			var atttypmod int
			err := rows.Scan(&relname, &attname, &atttypmod)
			if err != nil {
				return warehouses.Error(err)
			}
			if attTypMods[relname] == nil {
				attTypMods[relname] = map[string]*int{attname: &atttypmod}
			} else {
				attTypMods[relname][attname] = &atttypmod
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return warehouses.Error(err)
		}

		// Instantiate a resolver for the composite types.
		ctResolver, err := initCompositeTypeResolver(ctx, tx, enums, attTypMods)
		if err != nil {
			return warehouses.Error(err)
		}

		// Read columns.
		query = "SELECT c.table_name, c.column_name, c.is_nullable, c.data_type, c.udt_name, c.character_maximum_length," +
			" c.numeric_precision, c.numeric_precision_radix, c.numeric_scale, c.is_updatable," +
			" col_description(c.table_name::regclass::oid, c.ordinal_position)\n" +
			"FROM information_schema.columns c\n" +
			"INNER JOIN information_schema.tables t ON c.table_name = t.table_name AND c.table_schema = t.table_schema\n" +
			"WHERE t.table_schema = '" + warehouse.settings.Schema + "' AND t.table_type = 'BASE TABLE' AND" +
			" ( t.table_name IN ('users', 'users_identities', 'groups', 'groups_identities', 'events') OR t.table_name LIKE 'users\\__%' OR" +
			" t.table_name LIKE 'groups\\__%' OR t.table_name LIKE 'events\\__%' )\n" +
			"ORDER BY c.table_name, c.ordinal_position"

		rows, err = tx.Query(ctx, query)
		if err != nil {
			return warehouses.Error(err)
		}
		defer rows.Close()
		for rows.Next() {
			var row pgTypeInfo
			var tableName, columnName, dataType, udtName, isNullable, isUpdatable, description *string
			if err = rows.Scan(&tableName, &columnName, &isNullable, &dataType,
				&udtName, &row.charLength, &row.precision, &row.radix, &row.scale, &isUpdatable, &description); err != nil {
				return warehouses.Error(err)
			}
			if tableName == nil {
				return warehouses.Errorf("data warehouse has returned NULL as table name")
			}
			row.table = *tableName
			if columnName == nil {
				return warehouses.Errorf("data warehouse has returned NULL as column name")
			}
			if strings.HasPrefix(*columnName, "__") && strings.HasSuffix(*columnName, "__") { // used internally by Chichi.
				continue
			}
			if !types.IsValidPropertyName(*columnName) {
				return warehouses.Errorf("column name %q is not supported", *columnName)
			}
			row.column = *columnName
			if isNullable == nil {
				return warehouses.Errorf("data warehouse has returned NULL as nullability of column")
			}
			if dataType == nil {
				return warehouses.Errorf("data warehouse has returned NULL as column data type")
			}
			row.dataType = *dataType
			if udtName == nil {
				return warehouses.Errorf("data warehouse has returned NULL as column udt name")
			}
			row.udtName = *udtName
			if isUpdatable == nil {
				return warehouses.Errorf("data warehouse has returned NULL as updatability of column")
			}
			var role types.Role
			if *isUpdatable != "YES" {
				role = types.SourceRole
			}
			column := types.Property{
				Name:     row.column,
				Role:     role,
				Nullable: *isNullable == "YES",
			}
			column.Type, err = columnType(row, enums, ctResolver, attTypMods)
			if err != nil {
				return warehouses.Error(fmt.Errorf("data warehouse has returned an invalid type: %s", err))
			}
			if !column.Type.Valid() {
				return warehouses.Errorf("type of column %s.%s is not supported", row.table, column.Name)
			}
			if description != nil {
				column.Description = *description
			}
			if table == nil || row.table != table.Name {
				table = &warehouses.Table{Name: row.table}
				tables = append(tables, table)
			}
			table.Columns = append(table.Columns, column)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return warehouses.Error(err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return tables, nil
}

// pgTypeInfo holds information about a PostgreSQL type, as read from the
// PostgreSQL information tables (as 'information_schema.columns' and
// 'information_schema.attributes').
type pgTypeInfo struct {
	table      string
	column     string
	dataType   string
	udtName    string
	charLength *string
	precision  *string
	radix      *string
	scale      *string
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
