// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package snowflake

import (
	"database/sql"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

func TestInitializeAndRepairSimulatedSystemRecordsTable(t *testing.T) {
	warehouse, database := newTestSnowflakeWarehouse(t)

	err := warehouse.Initialize(t.Context(), warehouses.Production, nil)
	if err != nil {
		t.Fatalf("expected warehouse initialization, got %v", err)
	}
	assertSimulatedSystemRecordsTableDoesNotExist(t, database)
	err = warehouse.Repair(t.Context(), warehouses.Development, nil)
	if err != nil {
		t.Fatalf("expected development warehouse repair, got %v", err)
	}
	assertSimulatedSystemRecordsTable(t, database)

	mustExecSQL(t, database, `DROP TABLE "KRENALIS_SIMULATED_SYSTEM_RECORDS"`)
	err = warehouse.Repair(t.Context(), warehouses.Development, nil)
	if err != nil {
		t.Fatalf("expected simulated system records repair, got %v", err)
	}
	assertSimulatedSystemRecordsTable(t, database)
}

func TestMergeSimulatedSystemRecordsReplay(t *testing.T) {
	warehouse, database := newTestSnowflakeWarehouse(t)
	err := warehouse.Initialize(t.Context(), warehouses.Development, nil)
	if err != nil {
		t.Fatalf("expected warehouse initialization, got %v", err)
	}
	table := warehouses.Table{
		Name: "krenalis_simulated_system_records",
		Columns: []warehouses.Column{
			{Name: "simulated_system", Type: types.String()},
			{Name: "external_id", Type: types.String()},
			{Name: "updated_at", Type: types.DateTime()},
			{Name: "data", Type: types.JSON()},
		},
		Keys: []string{"simulated_system", "external_id"},
	}
	firstUpdatedAt := time.Date(2024, 10, 3, 15, 38, 36, 920638000, time.UTC)
	secondUpdatedAt := firstUpdatedAt.Add(time.Microsecond)
	rows := [][]any{{"account-a", "record-1", firstUpdatedAt, json.Value(`{"first_name":"Ada"}`)},
		{"account-a", "record-2", secondUpdatedAt, json.Value(`{"first_name":"Ada"}`)}}
	err = warehouse.Merge(t.Context(), table, rows, nil)
	if err != nil {
		t.Fatalf("expected initial merge, got %v", err)
	}
	thirdUpdatedAt := secondUpdatedAt.Add(time.Hour)
	replayRows := [][]any{{"account-a", "record-1", thirdUpdatedAt, json.Value(`{"first_name":"Grace"}`)},
		{"account-a", "record-2", thirdUpdatedAt.Add(time.Microsecond), json.Value(`{"first_name":"Grace"}`)},
		{"account-a", "record-3", thirdUpdatedAt.Add(2 * time.Microsecond), json.Value(`{"first_name":"Grace"}`)}}
	for range 2 {
		err = warehouse.Merge(t.Context(), table, replayRows, nil)
		if err != nil {
			t.Fatalf("expected replay merge, got %v", err)
		}
	}
	var count int
	err = database.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM "KRENALIS_SIMULATED_SYSTEM_RECORDS"
		WHERE "SIMULATED_SYSTEM" = 'account-a'`).Scan(&count)
	if err != nil {
		t.Fatalf("expected replayed account count, got %v", err)
	}
	if count != 3 {
		t.Fatalf("expected three records after partial replay, got %d", count)
	}
	var kind, name string
	err = database.QueryRowContext(t.Context(), `SELECT TYPEOF("DATA"), "DATA":first_name::STRING
		FROM "KRENALIS_SIMULATED_SYSTEM_RECORDS"
		WHERE "SIMULATED_SYSTEM" = 'account-a' LIMIT 1`).Scan(&kind, &name)
	if err != nil {
		t.Fatalf("expected JSON object type, got %v", err)
	}
	if kind != "OBJECT" {
		t.Fatalf("expected JSON object, got %s", kind)
	}
	if name != "Grace" {
		t.Fatalf("expected replayed first name Grace, got %s", name)
	}
	rowsResult, err := database.QueryContext(t.Context(), `SELECT "EXTERNAL_ID", "UPDATED_AT"
		FROM "KRENALIS_SIMULATED_SYSTEM_RECORDS"
		WHERE "SIMULATED_SYSTEM" = 'account-a' ORDER BY "UPDATED_AT"`)
	if err != nil {
		t.Fatalf("expected simulated system timestamps query, got %v", err)
	}
	defer rowsResult.Close()
	var externalIDs []string
	var updatedAt []time.Time
	for rowsResult.Next() {
		var id string
		var value time.Time
		if err := rowsResult.Scan(&id, &value); err != nil {
			t.Fatalf("expected simulated system timestamp scan, got %v", err)
		}
		externalIDs = append(externalIDs, id)
		updatedAt = append(updatedAt, value)
	}
	if err := rowsResult.Err(); err != nil {
		t.Fatalf("expected simulated system timestamp rows, got %v", err)
	}
	if len(externalIDs) != 3 || externalIDs[0] != "record-1" || externalIDs[1] != "record-2" || externalIDs[2] != "record-3" {
		t.Fatalf("expected timestamps ordered by updated_at, got %v", externalIDs)
	}
	if updatedAt[0].Location() != time.UTC || updatedAt[1].Location() != time.UTC || updatedAt[2].Location() != time.UTC ||
		!updatedAt[0].Equal(thirdUpdatedAt) || !updatedAt[1].Equal(thirdUpdatedAt.Add(time.Microsecond)) ||
		!updatedAt[2].Equal(thirdUpdatedAt.Add(2*time.Microsecond)) {
		t.Fatalf("expected replayed timestamps, got %v", updatedAt)
	}
}

func assertSimulatedSystemRecordsTable(t *testing.T, database *sql.DB) {
	t.Helper()
	var columns []string
	rows, err := database.QueryContext(t.Context(), `SELECT "COLUMN_NAME"
		FROM "INFORMATION_SCHEMA"."COLUMNS"
		WHERE "TABLE_SCHEMA" = CURRENT_SCHEMA() AND "TABLE_NAME" = 'KRENALIS_SIMULATED_SYSTEM_RECORDS'
		ORDER BY "ORDINAL_POSITION"`)
	if err != nil {
		t.Fatalf("expected simulated system records columns query, got %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			t.Fatalf("expected simulated system records column scan, got %v", err)
		}
		columns = append(columns, strings.ToLower(column))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("expected simulated system records columns, got %v", err)
	}
	if want := []string{"simulated_system", "external_id", "updated_at", "data"}; !slices.Equal(columns, want) {
		t.Fatalf("expected simulated system records columns %v, got %v", want, columns)
	}
	var dataType, nullable string
	var precision sql.NullInt64
	err = database.QueryRowContext(t.Context(), `SELECT "DATA_TYPE", "DATETIME_PRECISION", "IS_NULLABLE"
		FROM "INFORMATION_SCHEMA"."COLUMNS"
		WHERE "TABLE_SCHEMA" = CURRENT_SCHEMA() AND "TABLE_NAME" = 'KRENALIS_SIMULATED_SYSTEM_RECORDS'
			AND "COLUMN_NAME" = 'UPDATED_AT'`).Scan(&dataType, &precision, &nullable)
	if err != nil {
		t.Fatalf("expected simulated system records updated_at query, got %v", err)
	}
	if dataType != "TIMESTAMP_NTZ" || !precision.Valid || precision.Int64 != 6 || nullable != "NO" {
		t.Fatalf("expected updated_at timestamp_ntz(6), got type=%q precision=%v nullable=%q",
			dataType, precision, nullable)
	}
}

func assertSimulatedSystemRecordsTableDoesNotExist(t *testing.T, database *sql.DB) {
	t.Helper()
	var exists bool
	err := database.QueryRowContext(t.Context(), `SELECT COUNT(*) > 0
		FROM "INFORMATION_SCHEMA"."TABLES"
		WHERE "TABLE_SCHEMA" = CURRENT_SCHEMA() AND "TABLE_NAME" = 'KRENALIS_SIMULATED_SYSTEM_RECORDS'`).Scan(&exists)
	if err != nil {
		t.Fatalf("expected simulated system records table query, got %v", err)
	}
	if exists {
		t.Fatal("expected simulated system records table to be absent, got present")
	}
}
