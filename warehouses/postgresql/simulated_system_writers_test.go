// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

func TestSimulatedSystemWriterFencing(t *testing.T) {

	warehouse, _ := newTestPostgreSQLWarehouse(t)
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
				count, countErr := warehouse.Count(t.Context(), table.Name, nil, where)
				if countErr != nil {
					t.Fatalf("expected post-cleanup count, got %v", countErr)
				}
				if count != 0 {
					t.Fatalf("expected no stale resurrection, got %d records", count)
				}
				return
			}

			t.Fatal("expected stale post-cleanup write error, got nil")
		})
	}

}

func TestSimulatedSystemWriterWaitsForInFlightWrite(t *testing.T) {

	warehouse, pool := newTestPostgreSQLWarehouse(t)
	err := warehouse.Initialize(t.Context(), warehouses.Development, nil)
	if err != nil {
		t.Fatalf("expected initialized warehouse, got %v", err)
	}
	err = warehouse.FenceSimulatedSystem(t.Context(), "in-flight", 2)
	if err != nil {
		t.Fatalf("expected writer admission, got %v", err)
	}
	conn, err := pool.Acquire(t.Context())
	if err != nil {
		t.Fatalf("expected old owner session, got %v", err)
	}
	blocker, err := pool.Begin(t.Context())
	if err != nil {
		conn.Release()
		t.Fatalf("expected deterministic mutation gate, got %v", err)
	}
	_, err = blocker.Exec(t.Context(), "SELECT pg_advisory_xact_lock(15)")
	if err != nil {
		blocker.Rollback(t.Context())
		conn.Release()
		t.Fatalf("expected held mutation gate, got %v", err)
	}
	ownerDone := make(chan struct{})
	var ownerErr error
	go func() {
		ownerErr = mergeSimulatedSystemWriter(t.Context(), conn, simulatedSystemWriter{"in-flight", 2},
			`SELECT pg_advisory_xact_lock(15); INSERT INTO krenalis_simulated_system_records
			(simulated_system, external_id, updated_at, data)
			VALUES ('in-flight', 'old', CURRENT_TIMESTAMP, '{"owner":"old"}')`)
		close(ownerDone)
	}()
	defer func() {
		blocker.Rollback(t.Context())
		<-ownerDone
		conn.Release()
	}()
	waitForSimulatedWriterLock(t, pool, "SELECT pg_advisory_xact_lock(15);%")

	recoveryDone := make(chan error, 1)
	go func() { recoveryDone <- warehouse.FenceSimulatedSystem(t.Context(), "in-flight", 3) }()
	waitForSimulatedWriterLock(t, pool, "INSERT INTO krenalis_simulated_system_writers%")
	var generation int64
	if err := pool.QueryRow(t.Context(), "SELECT generation FROM krenalis_simulated_system_writers WHERE simulated_system = 'in-flight'").Scan(&generation); err != nil {
		t.Fatalf("expected persistent current authority, got %v", err)
	}
	if generation != 2 {
		t.Fatalf("expected revocation blocked behind unfinished mutation, got generation %d", generation)
	}

	err = blocker.Commit(t.Context())
	if err != nil {
		t.Fatalf("expected mutation gate release, got %v", err)
	}
	<-ownerDone
	if ownerErr != nil {
		t.Fatalf("expected old mutation to commit before revocation, got %v", ownerErr)
	}
	err = <-recoveryDone
	if err != nil {
		t.Fatalf("expected completed revocation after old mutation, got %v", err)
	}
	err = mergeSimulatedSystemWriter(t.Context(), conn, simulatedSystemWriter{"in-flight", 2},
		"DELETE FROM krenalis_simulated_system_records")
	assertSimulatedSystemWriterRejected(t, err)

}

func assertSimulatedSystemWriterRejected(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		return
	}
	t.Fatal("expected revoked authority error, got nil")
}

// waitForSimulatedWriterLock observes server lock state rather than elapsed time.
func waitForSimulatedWriterLock(t *testing.T, pool *pgxpool.Pool, pattern string) {

	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var waiting bool
		if err := pool.QueryRow(t.Context(), `SELECT EXISTS (SELECT FROM pg_stat_activity
			WHERE wait_event_type = 'Lock' AND query LIKE $1 AND pid <> pg_backend_pid())`, pattern).Scan(&waiting); err != nil {
			t.Fatalf("expected observable warehouse lock wait, got %v", err)
		}
		if waiting {
			return
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatalf("expected blocked warehouse statement %s, got none", pattern)
}
