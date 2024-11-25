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
	var objects []string
	for rows.Next() {
		var createdOn, databaseName, schemaName any
		var name, kind string
		err := rows.Scan(&createdOn, &name, &kind, &databaseName, &schemaName)
		if err != nil {
			return snowflake(err)
		}
		typ := strings.ToLower(kind)
		objects = append(objects, fmt.Sprintf("%s '%s'", typ, name))
	}
	if err := rows.Err(); err != nil {
		return snowflake(err)
	}
	if objects != nil {
		reason := fmt.Sprintf("expected an empty database, got: %s", strings.Join(objects, ", "))
		return meergo.NewWarehouseNonInitializableError(reason)
	}
	return nil
}

// Initialize initializes the database objects on the data warehouse in order to
// make it work with Meergo.
func (warehouse *Snowflake) Initialize(ctx context.Context) error {
	return warehouse.initRepair(ctx, false)
}

// Repair repairs the database objects on the data warehouse needed by Meergo.
func (warehouse *Snowflake) Repair(ctx context.Context) error {
	return warehouse.initRepair(ctx, true)
}

// initRepair initializes (or repairs) the database objects (as tables, types,
// etc...) on the data warehouse.
func (warehouse *Snowflake) initRepair(ctx context.Context, repair bool) error {
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
	db := warehouse.openDB()
	for _, query := range queries {
		_, err := db.ExecContext(ctx, query)
		if err != nil {
			return snowflake(err)
		}
	}
	return nil
}
