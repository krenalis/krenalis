// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/initdb"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/kms"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type simulatedWorkerWarehouse struct {
	warehouses.Warehouse
	blocked  atomic.Bool
	fences   atomic.Int64
	deletes  atomic.Int64
	contexts chan context.Context
}

// Close releases the test warehouse.
func (w *simulatedWorkerWarehouse) Close() error {
	return nil
}

// Counts reports an empty test warehouse.
func (w *simulatedWorkerWarehouse) Counts(context.Context, string, []warehouses.Expr) ([]int, error) {
	return []int{0}, nil
}

// DeleteSimulatedSystemRecords records completed cleanup.
func (w *simulatedWorkerWarehouse) DeleteSimulatedSystemRecords(context.Context, string, int64) error {
	w.deletes.Add(1)
	return nil
}

// FenceSimulatedSystem simulates a temporary failure during finalization.
func (w *simulatedWorkerWarehouse) FenceSimulatedSystem(ctx context.Context, _ string, _ int64) error {
	w.contexts <- ctx
	w.fences.Add(1)
	if w.blocked.Load() {
		return errors.New("test fence unavailable")
	}
	return nil
}

// MergeSimulatedSystemRecords is unused by the failure and cleanup scenarios.
func (w *simulatedWorkerWarehouse) MergeSimulatedSystemRecords(context.Context, string, int64, warehouses.Table, [][]any) error {
	panic("core: unexpected generation in worker finalization test")
}

var simulatedWorkerWarehouseNumber atomic.Int64

// TestSimulatedSystemWorkerNotifications uses two replicated states to check
// follower requests, retry without new events, and takeover of pending work.
func TestSimulatedSystemWorkerNotifications(t *testing.T) {

	if testing.Short() {
		t.Skip("requires PostgreSQL")
	}
	container, err := postgres.Run(t.Context(), testimages.PostgreSQL,
		postgres.WithDatabase("krenalis"), postgres.WithUsername("krenalis"), postgres.WithPassword("krenalis"),
		postgres.BasicWaitStrategies())
	t.Cleanup(func() {
		err := testcontainers.TerminateContainer(container)
		if err != nil {
			t.Errorf("expected container termination, got %v", err)
		}
	})
	if err != nil {
		t.Fatalf("expected PostgreSQL container, got %v", err)
	}
	host, err := container.Host(t.Context())
	if err != nil {
		t.Fatalf("expected container host, got %v", err)
	}
	port, err := container.MappedPort(t.Context(), "5432/tcp")
	if err != nil {
		t.Fatalf("expected container port, got %v", err)
	}
	options := db.Options{
		Host: host, Port: int(port.Num()), Username: "krenalis", Password: "krenalis", Database: "krenalis",
	}
	database, err := db.Open(&options)
	if err != nil {
		t.Fatalf("expected database pool, got %v", err)
	}
	t.Cleanup(database.Close)
	manager, err := kms.New(t.Context(), "key:"+base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatalf("expected KMS, got %v", err)
	}
	err = initdb.InitIfEmpty(t.Context(), database, manager, false)
	if err != nil {
		t.Fatalf("expected initialized schema, got %v", err)
	}
	remote := &simulatedWorkerWarehouse{contexts: make(chan context.Context, 16)}
	remote.blocked.Store(true)
	platform := fmt.Sprintf("SimulatedWorkerTest%d", simulatedWorkerWarehouseNumber.Add(1))
	warehouses.Register(warehouses.Platform{Name: platform}, func(warehouses.SettingsLoader, warehouses.DialWith) *simulatedWorkerWarehouse {
		return remote
	})
	const workspace = "111111111111"
	schema, err := types.Object([]types.Property{
		{Name: "email", Type: types.String(), ReadOptional: true},
	}).MarshalJSON()
	if err != nil {
		t.Fatalf("expected profile schema, got %v", err)
	}
	_, err = database.Exec(t.Context(), `INSERT INTO workspaces
		(id, organization, name, environment, warehouse_name, warehouse_mode, warehouse_settings,
		 kms_encrypted_warehouse_settings_key, kms_encrypted_warehouse_mcp_settings_key, profile_schema)
		SELECT $1, id, 'test', 'development', $2, 'Normal', '', '', '', $3
		FROM organizations`, workspace, platform, schema)
	if err != nil {
		t.Fatalf("expected workspace, got %v", err)
	}
	_, err = database.Exec(t.Context(), `CREATE ROLE worker_reader LOGIN PASSWORD 'worker_reader';
		GRANT USAGE ON SCHEMA public TO worker_reader`)
	if err != nil {
		t.Fatalf("expected restricted reader, got %v", err)
	}
	readerOptions := options
	readerOptions.Username = "worker_reader"
	readerOptions.Password = "worker_reader"
	reader, err := db.Open(&readerOptions)
	if err != nil {
		t.Fatalf("expected restricted pool, got %v", err)
	}
	t.Cleanup(reader.Close)
	origin := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(origin.Close)
	source, err := fakedata.NewItalianWorldSource([]string{origin.URL})
	if err != nil {
		t.Fatalf("expected test asset source, got %v", err)
	}

	type pass struct {
		node  int
		retry bool
	}
	passes := make(chan pass, 32)
	var nodes [2]*Core
	for i := range nodes {
		st, err := state.New(t.Context(), database, manager, nil, false)
		if err != nil {
			t.Fatalf("expected replicated State, got %v", err)
		}
		ds, err := datastore.New(st)
		if err != nil {
			t.Fatalf("expected datastore, got %v", err)
		}
		node := &Core{db: reader, state: st, datastore: ds, simulatedWorldSource: source}
		node.close.ctx, node.close.cancelCtx = context.WithCancelCause(t.Context())
		node.simulatedWorker = newSimulatedSystemWorker(node)
		node.simulatedWorker.coordinator.process = func(ctx context.Context) bool {
			retry := node.simulatedWorker.process(ctx)
			passes <- pass{node: i, retry: retry}
			return retry
		}
		st.Freeze()
		st.AddListener(node.simulatedWorker.onCreateSimulatedSystem)
		st.AddListener(node.simulatedWorker.onMarkSimulatedSystemDeleting)
		st.AddListener(func(state.ElectLeader) { node.simulatedWorker.leaderChanged() })
		node.simulatedWorker.Start()
		st.Unfreeze()
		nodes[i] = node
		t.Cleanup(func() {
			node.close.cancelCtx(nil)
			node.simulatedWorker.Close()
			ds.Close()
			if !node.closed.Load() {
				st.Close(context.Background())
			}
		})
	}
	wait := func(condition func() bool) {
		t.Helper()
		deadline := time.Now().Add(20 * time.Second)
		for !condition() {
			if time.Now().After(deadline) {
				t.Fatal("expected worker condition, got timeout")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	wait(func() bool { return nodes[0].state.IsLeader() != nodes[1].state.IsLeader() })
	leader := 0
	if nodes[1].state.IsLeader() {
		leader = 1
	}
	follower := 1 - leader
	waitPass := func(retry bool) {
		t.Helper()
		timer := time.NewTimer(15 * time.Second)
		defer timer.Stop()
		for {
			select {
			case actual := <-passes:
				if actual.node != leader {
					t.Fatalf("expected work on leader %d, got node %d", leader, actual.node)
				}
				if actual.retry == retry {
					return
				}
			case <-timer.C:
				t.Fatalf("expected leader pass with retry=%t, got timeout", retry)
			}
		}
	}
	ws, ok := nodes[follower].state.Workspace(workspace)
	if !ok {
		t.Fatal("expected follower workspace, got none")
	}
	store, ok := nodes[follower].datastore.Store(workspace)
	if !ok {
		t.Fatal("expected follower store, got none")
	}
	facade := &Workspace{core: nodes[follower], workspace: ws, store: store}
	create := func() string {
		t.Helper()
		id, err := facade.CreateSimulatedSystem(t.Context(), "from follower",
			json.Value("1"), nil, json.Value(`{"IT":100}`))
		if err != nil {
			t.Fatalf("expected follower creation, got %v", err)
		}
		return id
	}

	id := create()
	waitPass(true)
	_, err = database.Exec(t.Context(), "GRANT SELECT ON simulated_systems, connections TO worker_reader")
	if err != nil {
		t.Fatalf("expected reader recovery without a notification, got %v", err)
	}
	wait(func() bool { return remote.fences.Load() > 0 })
	waitPass(true)
	account, ok := nodes[leader].state.SimulatedSystem(id)
	if !ok || account.Status != state.SimulatedSystemPreparing || account.GenerationError == "" {
		t.Fatal("expected Preparing with persisted failure intent, got different state")
	}
	operationCtx := <-remote.contexts
	select {
	case <-operationCtx.Done():
	default:
		t.Fatal("expected completed attempt context to be canceled, got retained operation context")
	}
	if !nodes[leader].state.IsLeader() || nodes[leader].close.ctx.Err() != nil {
		t.Fatal("expected active leadership after the attempt, got stopped leader")
	}
	remote.blocked.Store(false)
	waitPass(false)
	wait(func() bool {
		account, ok := nodes[follower].state.SimulatedSystem(id)
		return ok && account.Status == state.SimulatedSystemFailed
	})

	_, err = database.Exec(t.Context(), "UPDATE simulated_systems SET claimed_run = '222222222222',"+
		" writer_generation = 2 WHERE id = $1", id)
	if err != nil {
		t.Fatalf("expected active writer claim, got %v", err)
	}
	err = facade.DeleteSimulatedSystem(t.Context(), id)
	if err != nil {
		t.Fatalf("expected follower deletion request, got %v", err)
	}
	waitPass(true)
	_, err = database.Exec(t.Context(), "UPDATE simulated_systems SET claimed_run = NULL WHERE id = $1", id)
	if err != nil {
		t.Fatalf("expected claim release without a notification, got %v", err)
	}
	waitPass(false)
	if _, ok := nodes[leader].state.SimulatedSystem(id); ok || remote.deletes.Load() != 1 {
		t.Fatal("expected completed cleanup, got retained system or missing delete")
	}

	remote.blocked.Store(true)
	pending := create()
	waitPass(true)
	nodes[leader].close.cancelCtx(nil)
	nodes[leader].simulatedWorker.Close()
	nodes[leader].state.Close(context.Background())
	nodes[leader].closed.Store(true)
	leader = follower
	remote.blocked.Store(false)
	wait(func() bool { return nodes[leader].state.IsLeader() })
	waitPass(false)
	account, ok = nodes[leader].state.SimulatedSystem(pending)
	if !ok || account.Status != state.SimulatedSystemFailed || account.GenerationError == "" {
		t.Fatal("expected takeover to finish pending failure, got different state")
	}

	entered, release := make(chan struct{}), make(chan struct{})
	releaseDispatch := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseDispatch)
	node := nodes[leader]
	node.state.Freeze()
	node.state.AddListener(func(n state.UpdateSimulatedSystemGeneration) {
		if n.Workspace == workspace && n.Status == state.SimulatedSystemPreparing && n.GenerationError != "" {
			close(entered)
			<-release
		}
	})
	node.state.Unfreeze()
	committed := create()
	select {
	case <-entered:
	case <-time.After(15 * time.Second):
		t.Fatal("expected suspended application of failure intent, got timeout")
	}
	var persisted bool
	var committedVersion int
	err = database.QueryRow(t.Context(), `SELECT status = 'Preparing' AND generation_error <> '',
		(SELECT MAX(version) FROM notifications WHERE name = 'UpdateSimulatedSystemGeneration'
		 AND payload::jsonb->>'ID' = $1) FROM simulated_systems WHERE id = $1`, committed).
		Scan(&persisted, &committedVersion)
	if err != nil {
		t.Fatalf("expected committed failure intent, got %v", err)
	}
	if !persisted || node.state.Version() >= committedVersion {
		t.Fatal("expected committed intent awaiting local application, got different state")
	}
	select {
	case actual := <-passes:
		t.Fatalf("expected worker waiting for application, got completed pass %+v", actual)
	default:
	}
	closed := make(chan struct{})
	go func() {
		node.simulatedWorker.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("expected worker shutdown during the version wait, got timeout")
	}
	if node.state.Version() >= committedVersion || node.close.ctx.Err() != nil {
		t.Fatal("expected suspended application and active Core after worker shutdown, got different state")
	}
	releaseDispatch()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	err = node.state.WaitVersion(ctx, committedVersion)
	if err != nil {
		t.Fatalf("expected released application, got %v", err)
	}
	node.state.Freeze()
	node.simulatedWorker = newSimulatedSystemWorker(node)
	node.state.AddListener(node.simulatedWorker.onCreateSimulatedSystem)
	node.state.AddListener(node.simulatedWorker.onMarkSimulatedSystemDeleting)
	node.simulatedWorker.Start()
	node.state.Unfreeze()
	wait(func() bool {
		account, ok := node.state.SimulatedSystem(committed)
		return ok && account.Status == state.SimulatedSystemFailed && account.GenerationError != ""
	})
	var finalStatus string
	err = database.QueryRow(t.Context(), "SELECT status::text FROM simulated_systems WHERE id = $1", committed).
		Scan(&finalStatus)
	if err != nil {
		t.Fatalf("expected recovered persisted status, got %v", err)
	}
	if finalStatus != "Failed" {
		t.Fatalf("expected recovered Failed status, got %s", finalStatus)
	}

}
