// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package postgresql

import (
	"testing"

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestInitializeAndRepairSimulatedAccountRecordsTable(t *testing.T) {
	warehouse, pool := newTestPostgreSQLWarehouse(t)

	err := warehouse.Initialize(t.Context(), nil)
	if err != nil {
		t.Fatalf("expected warehouse initialization, got %v", err)
	}
	assertSimulatedAccountRecordsTable(t, pool)

	mustExecSQL(t, pool, "DROP TABLE krenalis_simulated_account_records")
	err = warehouse.Repair(t.Context(), nil)
	if err != nil {
		t.Fatalf("expected warehouse repair, got %v", err)
	}
	assertSimulatedAccountRecordsTable(t, pool)
}

func TestMergeSimulatedAccountRecordsReplay(t *testing.T) {
	warehouse, pool := newTestPostgreSQLWarehouse(t)
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
	err = pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM krenalis_simulated_account_records"+
		" WHERE simulated_account_id = 'account-a'").Scan(&count)
	if err != nil {
		t.Fatalf("expected replayed account count, got %v", err)
	}
	if count != 2 {
		t.Fatalf("expected two records after replay, got %d", count)
	}
	var kind string
	err = pool.QueryRow(t.Context(), "SELECT jsonb_typeof(data) FROM krenalis_simulated_account_records"+
		" WHERE simulated_account_id = 'account-a' LIMIT 1").Scan(&kind)
	if err != nil {
		t.Fatalf("expected JSON object type, got %v", err)
	}
	if kind != "object" {
		t.Fatalf("expected JSON object, got %s", kind)
	}
}

func assertSimulatedAccountRecordsTable(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	var exists bool
	err := pool.QueryRow(t.Context(), "SELECT to_regclass('krenalis_simulated_account_records') IS NOT NULL").Scan(&exists)
	if err != nil {
		t.Fatalf("expected simulated account records table query, got %v", err)
	}
	if !exists {
		t.Fatal("expected simulated account records table to exist")
	}
}
