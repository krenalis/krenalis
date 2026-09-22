// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package snowflake

import (
	"database/sql"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

func TestInitializeAndRepairSimulatedAccountRecordsTable(t *testing.T) {
	warehouse, database := newTestSnowflakeWarehouse(t)

	err := warehouse.Initialize(t.Context(), nil)
	if err != nil {
		t.Fatalf("expected warehouse initialization, got %v", err)
	}
	assertSimulatedAccountRecordsTable(t, database)

	mustExecSQL(t, database, `DROP TABLE "KRENALIS_SIMULATED_ACCOUNT_RECORDS"`)
	err = warehouse.Repair(t.Context(), nil)
	if err != nil {
		t.Fatalf("expected warehouse repair, got %v", err)
	}
	assertSimulatedAccountRecordsTable(t, database)
}

func TestMergeSimulatedAccountRecordsReplay(t *testing.T) {
	warehouse, database := newTestSnowflakeWarehouse(t)
	err := warehouse.Initialize(t.Context(), nil)
	if err != nil {
		t.Fatalf("expected warehouse initialization, got %v", err)
	}
	table := warehouses.Table{
		Name: "krenalis_simulated_account_records",
		Columns: []warehouses.Column{
			{Name: "simulated_account_id", Type: types.String()},
			{Name: "external_id", Type: types.String()},
			{Name: "data", Type: types.JSON()},
		},
		Keys: []string{"simulated_account_id", "external_id"},
	}
	rows := [][]any{{"account-a", "record-1", json.Value(`{"first_name":"Ada"}`)},
		{"account-a", "record-2", json.Value(`{"first_name":"Ada"}`)}}
	for range 2 {
		err = warehouse.Merge(t.Context(), table, rows, nil)
		if err != nil {
			t.Fatalf("expected idempotent merge, got %v", err)
		}
	}
	var count int
	err = database.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM "KRENALIS_SIMULATED_ACCOUNT_RECORDS"
		WHERE "SIMULATED_ACCOUNT_ID" = 'account-a'`).Scan(&count)
	if err != nil {
		t.Fatalf("expected replayed account count, got %v", err)
	}
	if count != 2 {
		t.Fatalf("expected two records after replay, got %d", count)
	}
	var kind string
	err = database.QueryRowContext(t.Context(), `SELECT TYPEOF("DATA") FROM "KRENALIS_SIMULATED_ACCOUNT_RECORDS"
		WHERE "SIMULATED_ACCOUNT_ID" = 'account-a' LIMIT 1`).Scan(&kind)
	if err != nil {
		t.Fatalf("expected JSON object type, got %v", err)
	}
	if kind != "OBJECT" {
		t.Fatalf("expected JSON object, got %s", kind)
	}
}

func assertSimulatedAccountRecordsTable(t *testing.T, database *sql.DB) {
	t.Helper()
	var exists bool
	err := database.QueryRowContext(t.Context(), `SELECT COUNT(*) > 0
		FROM "INFORMATION_SCHEMA"."TABLES"
		WHERE "TABLE_SCHEMA" = CURRENT_SCHEMA() AND "TABLE_NAME" = 'KRENALIS_SIMULATED_ACCOUNT_RECORDS'`).Scan(&exists)
	if err != nil {
		t.Fatalf("expected simulated account records table query, got %v", err)
	}
	if !exists {
		t.Fatal("expected simulated account records table to exist")
	}
}
