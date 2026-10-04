// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"sync"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

type delayedOwnershipWarehouse struct {
	warehouses.Warehouse
	fencer     simulatedSystemWarehouse
	generation int64
	entered    chan struct{}
	resume     chan struct{}
}

func (warehouse *delayedOwnershipWarehouse) DeleteSimulatedSystemRecords(ctx context.Context, system string, generation int64) error {
	return warehouse.fencer.DeleteSimulatedSystemRecords(ctx, system, generation)
}

func (warehouse *delayedOwnershipWarehouse) FenceSimulatedSystem(ctx context.Context, system string, generation int64) error {
	return warehouse.fencer.FenceSimulatedSystem(ctx, system, generation)
}

func (warehouse *delayedOwnershipWarehouse) MergeSimulatedSystemRecords(ctx context.Context, system string, generation int64, table warehouses.Table, rows [][]any) error {
	if generation == warehouse.generation {
		close(warehouse.entered)
		<-warehouse.resume
	}
	return warehouse.fencer.MergeSimulatedSystemRecords(ctx, system, generation, table, rows)
}

type ownershipWarehouse struct {
	*simulatedRecordsWriteWarehouse // records protected by mu
	mu                              sync.Mutex
	generation                      int64 // protected by mu
}

func (warehouse *ownershipWarehouse) DeleteSimulatedSystemRecords(_ context.Context, _ string, generation int64) error {
	warehouse.mu.Lock()
	defer warehouse.mu.Unlock()
	if warehouse.generation != generation {
		return errors.New("writer revoked")
	}
	clear(warehouse.records)
	return nil
}

func (warehouse *ownershipWarehouse) FenceSimulatedSystem(_ context.Context, _ string, generation int64) error {
	warehouse.mu.Lock()
	defer warehouse.mu.Unlock()
	warehouse.generation = max(warehouse.generation, generation)
	return nil
}

func (warehouse *ownershipWarehouse) MergeSimulatedSystemRecords(ctx context.Context, system string, generation int64, table warehouses.Table, rows [][]any) error {
	warehouse.mu.Lock()
	defer warehouse.mu.Unlock()
	if warehouse.generation != generation {
		return errors.New("writer revoked")
	}
	return warehouse.simulatedRecordsWriteWarehouse.MergeSimulatedSystemRecords(ctx, system, generation, table, rows)
}

func (warehouse *ownershipWarehouse) Query(ctx context.Context, query warehouses.RowQuery, next bool) (warehouses.Rows, int, error) {
	warehouse.mu.Lock()
	defer warehouse.mu.Unlock()
	return warehouse.simulatedRecordsWriteWarehouse.Query(ctx, query, next)
}

func TestSimulatedSystemWriterRecovery(t *testing.T) {

	for _, scenario := range []string{"expiry", "replacement", "deletion", "generation"} {
		t.Run(scenario, func(t *testing.T) {
			warehouse := &ownershipWarehouse{
				simulatedRecordsWriteWarehouse: &simulatedRecordsWriteWarehouse{records: map[string]simulatedRecordsWriteRow{}},
			}
			testSimulatedSystemWriterRecovery(t, warehouse, scenario)
		})
	}

}

func TestSimulatedSystemWriterRecoveryPostgreSQL(t *testing.T) {

	settings := simulatedRecordsWarehouseSettings(t, "PostgreSQL")
	warehouse := warehouses.Registered("PostgreSQL").New(newTestSettingsLoader(settings), nil)
	t.Cleanup(func() { warehouse.Close() })
	err := warehouse.Initialize(t.Context(), warehouses.Development, nil)
	if err != nil {
		t.Fatalf("expected PostgreSQL initialization, got %v", err)
	}
	for _, scenario := range []string{"expiry", "replacement", "deletion", "generation"} {
		t.Run(scenario, func(t *testing.T) {
			testSimulatedSystemWriterRecovery(t, warehouse, scenario)
		})
	}

}

func TestWriteSimulatedSystemRecordsRequiresAdmission(t *testing.T) {

	store := &Store{mc: newModeCoordinator(state.Normal)}
	for _, generation := range []int64{-1, 0, 1} {
		ctx := t.Context()
		if generation >= 0 {
			ctx = simulatedWriterContext(t, "system", generation)
		}
		_, err := store.WriteSimulatedSystemRecords(ctx, "system", []SimulatedSystemRecordWrite{{ID: "record"}}, nil)
		if err != nil {
			continue
		}
		t.Fatalf("expected admission error for generation %d, got nil", generation)
	}

}

// testSimulatedSystemWriterRecovery pauses a request with a valid, uncanceled
// context after datastore reads, then recovers authority in a separate Store.
func testSimulatedSystemWriterRecovery(t *testing.T, warehouse warehouses.Warehouse, scenario string) {

	t.Helper()
	system := "system-" + scenario
	generation := int64(2)
	if scenario == "generation" {
		generation = 0
	}
	oldWarehouse := &delayedOwnershipWarehouse{
		Warehouse: warehouse, fencer: warehouse.(simulatedSystemWarehouse), generation: generation,
		entered: make(chan struct{}), resume: make(chan struct{}),
	}
	t.Cleanup(func() {
		select {
		case <-oldWarehouse.resume:
		default:
			close(oldWarehouse.resume)
		}
	})
	oldStore := &Store{mc: newModeCoordinator(state.Normal)}
	oldStore.wh.Store(warehouses.Warehouse(oldWarehouse))
	newStore := &Store{mc: newModeCoordinator(state.Normal)}
	newStore.wh.Store(warehouse)
	err := newStore.FenceSimulatedSystem(t.Context(), system, generation)
	if err != nil {
		t.Fatalf("expected writer admission, got %v", err)
	}

	finished := make(chan error, 1)
	go func() {
		if scenario == "generation" {
			finished <- oldStore.MergeSimulatedSystemRecords(t.Context(), system,
				[]SimulatedSystemRecord{{ID: "old", Data: json.Value(`{"owner":"old"}`)}})
			return
		}
		_, err := oldStore.WriteSimulatedSystemRecords(simulatedWriterContext(t, system, generation), system,
			[]SimulatedSystemRecordWrite{{ID: "old", Data: json.Value(`{"owner":"old"}`)}}, nil)
		finished <- err
	}()
	<-oldWarehouse.entered

	// A failed/finished run uses the same sealing boundary as timeout recovery.
	err = newStore.FenceSimulatedSystem(t.Context(), system, generation+1)
	if err != nil {
		t.Fatalf("expected warehouse ownership revocation, got %v", err)
	}
	switch scenario {
	case "replacement":
		err = newStore.FenceSimulatedSystem(t.Context(), system, 4)
		if err != nil {
			t.Fatalf("expected replacement admission, got %v", err)
		}
		_, err = newStore.WriteSimulatedSystemRecords(simulatedWriterContext(t, system, 4), system,
			[]SimulatedSystemRecordWrite{{ID: "new", Data: json.Value(`{"owner":"new"}`)}}, nil)
		if err != nil {
			t.Fatalf("expected authoritative replacement write, got %v", err)
		}
	case "deletion", "generation":
		err = newStore.DeleteSimulatedSystemRecords(simulatedWriterContext(t, system, generation+1), system)
		if err != nil {
			t.Fatalf("expected successful fenced cleanup, got %v", err)
		}
	}

	close(oldWarehouse.resume)
	err = <-finished
	if err != nil {
		records, readErr := newStore.SimulatedSystemRecordsByID(t.Context(), system, []string{"old", "new"})
		if readErr != nil {
			t.Fatalf("expected final warehouse records, got %v", readErr)
		}
		for id := range records {
			if id != "new" {
				t.Fatalf("expected only permitted replacement records, got %+v", records)
			}
		}
		if scenario == "replacement" {
			if len(records) != 1 {
				t.Fatalf("expected one replacement record, got %+v", records)
			}
			// A delayed grant and release must not roll the replacement fence back.
			err = newStore.FenceSimulatedSystem(t.Context(), system, generation)
			if err != nil {
				t.Fatalf("expected monotonic old grant, got %v", err)
			}
			_, err = newStore.WriteSimulatedSystemRecords(simulatedWriterContext(t, system, 4), system, nil,
				[]SimulatedSystemRecordWrite{{ID: "new", Data: json.Value(`{"owner":"new","updated":true}`)}})
			if err != nil {
				t.Fatalf("expected replacement authority retained, got %v", err)
			}
		}
		return
	}

	t.Fatal("expected resumed stale mutation to fail, got nil")
}
