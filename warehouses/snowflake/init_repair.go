//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package snowflake

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/meergo/meergo"
)

// CanInitialize checks whether the data warehouse can be initialized.
func (warehouse *Snowflake) CanInitialize(ctx context.Context) error {
	db := warehouse.openDB()
	rows, err := db.QueryContext(ctx, "SHOW TERSE OBJECTS")
	if err != nil {
		return snowflake(err)
	}
	defer rows.Close()
	count := map[string]int{}
	for rows.Next() {
		var createdOn, databaseName, schemaName any
		var name, kind string
		err := rows.Scan(&createdOn, &name, &kind, &databaseName, &schemaName)
		if err != nil {
			return snowflake(err)
		}
		typ := strings.ToLower(kind)
		count[typ] = count[typ] + 1
	}
	if err := rows.Err(); err != nil {
		return snowflake(err)
	}
	// Populate 'errors' to return an error like: «database is not empty (it
	// contains 1 view, 3 sequences, 4 indexes, 5 tables)».
	var errors []string
	for typ, count := range count {
		if count == 1 {
			errors = append(errors, "1 "+typ)
			continue
		}
		if typ == "index" {
			typ += "es"
		} else {
			typ += "s"
		}
		errors = append(errors, strconv.Itoa(count)+" "+typ)
	}
	if errors != nil {
		slices.Sort(errors)
		err := fmt.Errorf("database is not empty (it contains %s)", strings.Join(errors, ", "))
		return meergo.NewWarehouseNonInitializableError(err)
	}
	return nil
}

// Initialize initializes the database objects on the data warehouse in order to
// make it work with Meergo.
func (warehouse *Snowflake) Initialize(ctx context.Context, userColumns []meergo.Column) error {
	return warehouse.initRepairDatabaseObjects(ctx, userColumns, false)
}

// Repair repairs the database objects on the data warehouse needed by Meergo.
func (warehouse *Snowflake) Repair(ctx context.Context, userColumns []meergo.Column) error {
	err := warehouse.initRepairDatabaseObjects(ctx, userColumns, true)
	if err != nil {
		return err
	}
	// Repair the rows of the "_OPERATIONS" table, setting an end timestamp for
	// all those rows that do not have one, which were produced by abrupt
	// interruptions of the Meergo process.
	//
	// This clearly requires that the repair is executed only when it is certain
	// that no operations are ongoing.
	db := warehouse.openDB()
	_, err = db.ExecContext(ctx, `UPDATE "_OPERATIONS"`+
		` SET "END_TIME" = SYSDATE()`+
		` WHERE "END_TIME" IS NULL`)
	return err
}

// initRepairDatabaseObjects initializes (or repairs) the database objects (as
// tables, types, etc...) on the data warehouse.
func (warehouse *Snowflake) initRepairDatabaseObjects(ctx context.Context, userColumns []meergo.Column, repair bool) error {
	queries := []string{
		createDestinationUsersTable,
		createEventsTable,
		createOperationsTable,
		createOperations2Table,
		userIdentitiesSQLSchema(userColumns),
		usersSQLSchema("_users_0", userColumns),
	}
	if !repair { // TODO(Gianluca): is this necessary in Snowflake?
		queries = append(queries, usersViewSQLSchema(userColumns, "_users_0"))
	}
	db := warehouse.openDB()
	for _, query := range queries {
		_, err := db.ExecContext(ctx, query)
		if err != nil {
			return snowflake(err)
		}
	}
	return nil
}

// userIdentitiesSQLSchema returns the SQL schema (in the form "CREATE TABLE
// ...") of the user identities table, which includes, in addition to the
// columns used internally by the driver, the given user columns.
func userIdentitiesSQLSchema(userColumns []meergo.Column) string {
	var b strings.Builder
	b.WriteString(`CREATE TABLE IF NOT EXISTS "_USER_IDENTITIES" (
		"__PK__" INT AUTOINCREMENT START 0 INCREMENT 1 ORDER,
		"__ACTION__" INT NOT NULL,
		"__IS_ANONYMOUS__" BOOLEAN NOT NULL DEFAULT FALSE,
		"__IDENTITY_ID__" VARCHAR NOT NULL,
		"__CONNECTION__" INT NOT NULL,
		"__ANONYMOUS_IDS__" ARRAY,
		"__LAST_CHANGE_TIME__" TIMESTAMP_NTZ NOT NULL,
		"__EXECUTION__" INT,
		"__GID__" VARCHAR(36),
		"__CLUSTER__" INT AUTOINCREMENT START 0 INCREMENT 1 ORDER`)
	for _, c := range userColumns {
		b.WriteString(",\n")
		b.WriteString(quoteIdent(c.Name))
		b.WriteByte(' ')
		b.WriteString(typeToSnowflakeType(c.Type))
	}
	b.WriteString(",\n" + `PRIMARY KEY ("__pk__"))`)
	return b.String()
}

// usersSQLSchema returns the SQL schema (in the form "CREATE TABLE ...") of the
// users table with the given name, which includes, in addition to the columns
// used internally by the driver, the given user columns.
func usersSQLSchema(name string, userColumns []meergo.Column) string {
	var b strings.Builder
	b.WriteString(`CREATE TABLE IF NOT EXISTS `)
	b.WriteString(quoteIdent(name))
	b.WriteString(` (
		"__ID__" VARCHAR(36),
		"__IDENTITIES__" ARRAY,
		"__LAST_CHANGE_TIME__" TIMESTAMP NOT NULL`)
	for _, c := range userColumns {
		b.WriteString(",\n")
		b.WriteString(quoteIdent(c.Name))
		b.WriteByte(' ')
		b.WriteString(typeToSnowflakeType(c.Type))
	}
	b.WriteByte(')')
	return b.String()
}

// usersViewSQLSchema returns the SQL schema (in the form "CREATE ... VIEW ...")
// of the users view which is based on the users table with the given name.
func usersViewSQLSchema(userColumns []meergo.Column, fromUsersTable string) string {
	var b strings.Builder
	b.WriteString(`CREATE OR REPLACE VIEW "USERS" AS
		SELECT
			"__ID__",
			"__LAST_CHANGE_TIME__"`)
	for _, c := range userColumns {
		b.WriteString(",\n")
		b.WriteString(quoteIdent(c.Name))
	}
	b.WriteString(` FROM `)
	b.WriteString(quoteIdent(fromUsersTable))
	return b.String()
}
