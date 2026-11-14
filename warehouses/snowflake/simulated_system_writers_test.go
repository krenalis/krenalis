// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

func TestSimulatedSystemWriterFencing(t *testing.T) {

	warehouse, _ := newTestSnowflakeWarehouse(t)
	err := warehouse.Initialize(t.Context(), warehouses.Development, nil)
	if err != nil {
		t.Fatalf("expected development initialization, got %v", err)
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
	for _, test := range []struct {
		name       string
		generation int64
	}{
		{name: "generation", generation: 0},
		{name: "destination", generation: 2},
	} {
		t.Run(test.name, func(t *testing.T) {

			system := test.name
			rows := [][]any{{system, "record", time.Now().UTC().Truncate(time.Microsecond), json.Value(`{"owner":"old"}`)}}
			err := warehouse.FenceSimulatedSystem(t.Context(), system, test.generation)
			if err != nil {
				t.Fatalf("expected initial authority, got %v", err)
			}
			err = warehouse.MergeSimulatedSystemRecords(t.Context(), system, test.generation, table, rows)
			if err != nil {
				t.Fatalf("expected admitted write, got %v", err)
			}
			err = warehouse.FenceSimulatedSystem(t.Context(), system, test.generation+1)
			if err != nil {
				t.Fatalf("expected revocation, got %v", err)
			}
			err = warehouse.MergeSimulatedSystemRecords(t.Context(), system, test.generation, table, rows)
			assertSimulatedSystemWriterRejected(t, err)

			err = warehouse.FenceSimulatedSystem(t.Context(), system, test.generation+2)
			if err != nil {
				t.Fatalf("expected replacement authority, got %v", err)
			}
			err = warehouse.FenceSimulatedSystem(t.Context(), system, test.generation)
			if err != nil {
				t.Fatalf("expected idempotent delayed old grant, got %v", err)
			}
			rows[0][3] = json.Value(`{"owner":"replacement"}`)
			err = warehouse.MergeSimulatedSystemRecords(t.Context(), system, test.generation+2, table, rows)
			if err != nil {
				t.Fatalf("expected replacement write after delayed grant, got %v", err)
			}
			where := warehouses.NewBaseExpr(table.Columns[0], warehouses.OpIs, system)
			stored, _, err := warehouse.Query(t.Context(), warehouses.RowQuery{
				Table: table.Name, Columns: []warehouses.Column{table.Columns[3]}, Where: where, Limit: 1,
			}, false)
			if err != nil {
				t.Fatalf("expected persisted replacement record, got %v", err)
			}
			if !stored.Next() {
				t.Fatal("expected persisted replacement row, got none")
			}
			value := make([]any, 1)
			if err := stored.Scan(value...); err != nil {
				t.Fatalf("expected replacement document, got %v", err)
			}
			err = stored.Close()
			if err != nil {
				t.Fatalf("expected closed result, got %v", err)
			}
			var document struct {
				Owner string `json:"owner"`
			}
			err = json.Unmarshal(value[0].(json.Value), &document)
			if err != nil {
				t.Fatalf("expected decoded replacement document, got %v", err)
			}
			if document.Owner != "replacement" {
				t.Fatalf("expected replacement data, got %v", document)
			}

			err = warehouse.DeleteSimulatedSystemRecords(t.Context(), system, test.generation+1)
			assertSimulatedSystemWriterRejected(t, err)
			err = warehouse.FenceSimulatedSystem(t.Context(), system, test.generation+3)
			if err != nil {
				t.Fatalf("expected cleanup authority, got %v", err)
			}
			err = warehouse.DeleteSimulatedSystemRecords(t.Context(), system, test.generation+3)
			if err != nil {
				t.Fatalf("expected fenced cleanup, got %v", err)
			}
			err = warehouse.MergeSimulatedSystemRecords(t.Context(), system, test.generation+2, table, rows)
			if err != nil {
				counts, countErr := warehouse.Counts(t.Context(), table.Name, []warehouses.Expr{where})
				if countErr != nil {
					t.Fatalf("expected post-cleanup count, got %v", countErr)
				}
				if len(counts) != 1 || counts[0] != 0 {
					t.Fatalf("expected one zero count after cleanup, got %v", counts)
				}
				return
			}

			t.Fatal("expected stale post-cleanup write error, got nil")
		})
	}

}

func TestSimulatedSystemWriterSingletonInitialization(t *testing.T) {
	if !strings.Contains(createSimulatedSystemWritersTable, "CREATE TABLE IF NOT EXISTS") ||
		!strings.Contains(createSimulatedSystemWritersTable, "SELECT ''::VARCHAR") {
		t.Fatalf("expected atomic persistent singleton creation, got %s", createSimulatedSystemWritersTable)
	}
}

func TestSimulatedSystemWriterTransaction(t *testing.T) {

	for _, permitted := range []bool{true, false} {
		db, queries := newCheckReadOnlyTestDB(t, []checkReadOnlyQuery{
			{match: "EXECUTE IMMEDIATE", cols: []string{"RESULT"}, rows: [][]driver.Value{{permitted}}},
		})
		t.Cleanup(func() { db.Close() })
		conn, err := db.Conn(t.Context())
		if err != nil {
			t.Fatalf("expected warehouse session, got %v", err)
		}
		err = mergeSimulatedSystemWriter(t.Context(), conn, simulatedSystemWriter{"system'\\test", 2},
			"MERGE INTO KRENALIS_SIMULATED_SYSTEM_RECORDS USING STAGED_RECORDS ON TRUE WHEN MATCHED THEN DELETE")
		conn.Close()
		if err != nil {
			if permitted {
				t.Fatalf("expected permitted mutation, got %v", err)
			}
		}
		if !permitted && err == nil {
			t.Fatal("expected revoked mutation error, got nil")
		}
		if len(*queries) != 1 {
			t.Fatalf("expected one complete server request, got %v", *queries)
		}
		query := (*queries)[0]
		parts := []string{"BEGIN TRANSACTION;", "UPDATE KRENALIS_SIMULATED_SYSTEM_WRITERS", "SQLROWCOUNT <> 1",
			"SELECT MAX(GENERATION)", "writer_generation <> 2", "MERGE INTO KRENALIS_SIMULATED_SYSTEM_RECORDS", "COMMIT;"}
		position := 0
		for _, part := range parts {
			index := strings.Index(query[position:], part)
			if index < 0 {
				t.Fatalf("expected transaction to contain %q in order, got %s", part, query)
			}
			position += index + len(part)
		}
		for _, forbidden := range []string{"CREATE TEMPORARY", "PUT file://", "COPY INTO"} {
			if strings.Contains(query, forbidden) {
				t.Fatalf("expected staging outside guarded transaction, got %s", query)
			}
		}
		if !strings.Contains(query, "EXCEPTION WHEN OTHER THEN ROLLBACK; RAISE;") {
			t.Fatalf("expected errors to roll back guarded mutation, got %s", query)
		}
	}

}

func assertSimulatedSystemWriterRejected(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		return
	}
	t.Fatal("expected revoked authority error, got nil")
}
