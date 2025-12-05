// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/meergo/meergo/warehouses"
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
		return warehouses.NewNonInitializableError(err)
	}
	return nil
}

// Initialize initializes the database objects on the data warehouse in order to
// make it work with Meergo.
func (warehouse *Snowflake) Initialize(ctx context.Context, profileColumns []warehouses.Column) error {
	return warehouse.initRepairDatabaseObjects(ctx, profileColumns, false)
}

// Repair repairs the database objects on the data warehouse needed by
// warehouses.
func (warehouse *Snowflake) Repair(ctx context.Context, profileColumns []warehouses.Column) error {
	return warehouse.initRepairDatabaseObjects(ctx, profileColumns, true)
}

// identitiesSQLSchema returns the SQL schema (in the form "CREATE TABLE ...")
// of the identities table, which includes, in addition to the columns used
// internally by the driver, the given profile columns.
func identitiesSQLSchema(profileColumns []warehouses.Column) string {
	var b strings.Builder
	b.WriteString(`CREATE TABLE IF NOT EXISTS "_IDENTITIES" (
		"__PK__" INT AUTOINCREMENT START 0 INCREMENT 1 ORDER,
		"__PIPELINE__" INT NOT NULL,
		"__IS_ANONYMOUS__" BOOLEAN NOT NULL DEFAULT FALSE,
		"__IDENTITY_ID__" VARCHAR NOT NULL,
		"__CONNECTION__" INT NOT NULL,
		"__ANONYMOUS_IDS__" ARRAY,
		"__LAST_CHANGE_TIME__" TIMESTAMP_NTZ NOT NULL,
		"__EXECUTION__" INT,
		"__mpid__" VARCHAR(36),
		"__CLUSTER__" INT AUTOINCREMENT START 0 INCREMENT 1 ORDER`)
	for _, c := range profileColumns {
		b.WriteString(",\n")
		b.WriteString(quoteIdent(c.Name))
		b.WriteByte(' ')
		b.WriteString(typeToSnowflakeType(c.Type))
	}
	b.WriteString(",\n" + `PRIMARY KEY ("__PK__"))`)
	return b.String()
}

// initRepairDatabaseObjects initializes (or repairs) the database objects (as
// tables, types, etc...) on the data warehouse.
func (warehouse *Snowflake) initRepairDatabaseObjects(ctx context.Context, profileColumns []warehouses.Column, repair bool) error {
	queries := []string{
		createDestinationProfilesTable,
		createEventsTable,
		createOperationsTable,
		createProfileSchemaVersionTable,
		identitiesSQLSchema(profileColumns),
		profilesSQLSchema("meergo_profiles_0", profileColumns),
	}
	if !repair { // TODO(Gianluca): is this necessary in Snowflake?
		queries = append(queries, profilesViewSQLSchema(profileColumns, "meergo_profiles_0"))
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

// profilesSQLSchema returns the SQL schema (in the form "CREATE TABLE ...") of
// the profiles table with the given name, which includes, in addition to the
// columns used internally by the driver, the given profile columns.
func profilesSQLSchema(name string, profileColumns []warehouses.Column) string {
	var b strings.Builder
	b.WriteString(`CREATE TABLE IF NOT EXISTS `)
	b.WriteString(quoteIdent(name))
	b.WriteString(` (
		"__MPID__" VARCHAR(36),
		"__IDENTITIES__" ARRAY,
		"__LAST_CHANGE_TIME__" TIMESTAMP NOT NULL`)
	for _, c := range profileColumns {
		b.WriteString(",\n")
		b.WriteString(quoteIdent(c.Name))
		b.WriteByte(' ')
		b.WriteString(typeToSnowflakeType(c.Type))
	}
	b.WriteByte(')')
	return b.String()
}

// profilesViewSQLSchema returns the SQL schema (in the form "CREATE ... VIEW
// ...") of the profiles view which is based on the profiles table with the
// given name.
func profilesViewSQLSchema(profileColumns []warehouses.Column, fromProfilesTable string) string {
	var b strings.Builder
	b.WriteString(`CREATE OR REPLACE VIEW "PROFILES" AS
		SELECT
			"__MPID__",
			"__LAST_CHANGE_TIME__"`)
	for _, c := range profileColumns {
		b.WriteString(",\n")
		b.WriteString(quoteIdent(c.Name))
	}
	b.WriteString(` FROM `)
	b.WriteString(quoteIdent(fromProfilesTable))
	return b.String()
}
