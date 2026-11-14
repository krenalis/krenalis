// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
	"github.com/krenalis/krenalis/warehouses/postgresql"
)

type simulatedOwnershipSettings struct {
	value json.Value
}

func (settings *simulatedOwnershipSettings) Load(_ context.Context, destination any) error {
	return json.Unmarshal(settings.value, destination)
}

func TestSimulatedDestinationClaimSurvivesResourceDelete(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	warehouseSettings := &krenalistester.DBSettings{}
	err := json.Unmarshal(krenalistester.PostgresWarehouseSettings(), warehouseSettings)
	if err != nil {
		t.Fatalf("expected PostgreSQL warehouse settings, got %v", err)
	}
	adminPool, err := krenalistester.ConnectionPool(t.Context(), warehouseSettings)
	if err != nil {
		t.Fatalf("expected PostgreSQL connection pool, got %v", err)
	}
	database := "test_simulated_claim_delete_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	_, err = adminPool.Exec(t.Context(), "CREATE DATABASE "+database)
	adminPool.Close()
	if err != nil {
		t.Fatalf("expected isolated warehouse database, got %v", err)
	}
	cleanupSettings := *warehouseSettings
	defer func() {
		pool, err := krenalistester.ConnectionPool(context.Background(), &cleanupSettings)
		if err != nil {
			t.Errorf("expected PostgreSQL cleanup connection, got %v", err)
			return
		}
		defer pool.Close()
		_, err = pool.Exec(context.Background(), "DROP DATABASE "+database+" WITH (FORCE)")
		if err != nil {
			t.Errorf("expected isolated warehouse cleanup, got %v", err)
		}
	}()
	warehouseSettings.Database = database
	profileSchema := simulatedApplicationProfileSchema()
	var workspace struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/workspaces", http.Header{"Krenalis-Workspace": nil}, map[string]any{
		"name": "Simulated claim deletion", "environment": "development", "profileSchema": krenalistester.ProfileSchema{Schema: profileSchema},
		"warehouse": map[string]any{"platform": "PostgreSQL", "mode": "Normal", "settings": warehouseSettings},
	}, &workspace)
	k.SetWorkspaceID(workspace.ID)

	var accountIDs, connectionIDs []string
	createAccount := func(name string) string {
		var account struct {
			ID string `json:"id"`
		}
		k.Call("POST", "/v1/simulated/systems", nil, map[string]any{
			"name": name, "userCount": 0, "countries": map[string]int{"IT": 100},
		}, &account)
		accountIDs = append(accountIDs, account.ID)
		return account.ID
	}
	sourceAccount := createAccount("Claim deletion source")
	warehousePool, err := krenalistester.ConnectionPool(t.Context(), warehouseSettings)
	if err != nil {
		t.Fatalf("expected simulated-system warehouse connection, got %v", err)
	}
	defer warehousePool.Close()
	data, err := json.Marshal(map[string]any{
		"customer_id": "source-customer-1", "first_name": "Ada", "last_name": "Rossi",
		"email": "ada@example.test", "gender": "female", "birth_date": "1990-01-02", "current_age": 36,
		"phone": "+393331234567", "site_url": "https://profiles.example.test/ada",
		"address": map[string]any{
			"street": "Via Roma 1", "municipality": "Roma", "state_prov": "RM",
			"postal_code": "00100", "country": "IT",
		},
		"photo": "https://krenalis.test/photos/catalog/256/face-000001.webp",
	})
	if err != nil {
		t.Fatalf("expected source customer document, got %v", err)
	}
	_, err = warehousePool.Exec(t.Context(), `INSERT INTO krenalis_simulated_system_records
		(simulated_system, external_id, updated_at, data) VALUES ($1, $2, $3, $4::jsonb)`,
		sourceAccount, "source-customer-1", time.Now().UTC().Truncate(time.Microsecond), string(data))
	if err != nil {
		t.Fatalf("expected PostgreSQL source customer, got %v", err)
	}
	sourceConnection := k.CreateConnection(krenalistester.ConnectionToCreate{
		Name: "Claim deletion source", Role: krenalistester.Source, Connector: "simulated-application",
		Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": sourceAccount}),
	})
	connectionIDs = append(connectionIDs, sourceConnection)
	sourceSchemas := simulatedApplicationPipelineSchemas(t, k, sourceConnection)
	sourcePipeline := k.CreatePipeline(sourceConnection, "User", krenalistester.PipelineToSet{
		Name: "Import claim deletion source", Enabled: true,
		InSchema: sourceSchemas.In, OutSchema: sourceSchemas.Out,
		Transformation: &krenalistester.Transformation{Mapping: simulatedApplicationSourceMapping()},
	})
	sourceRun := k.StartPipelineRun(sourcePipeline)
	k.WaitForRunsCompletion(sourceRun)

	mode := krenalistester.Server
	createConnection := func(name, account string) string {
		connection := k.CreateConnection(krenalistester.ConnectionToCreate{
			Name: name, Role: krenalistester.Destination, Connector: "simulated-application",
			SendingMode: &mode,
			Settings:    krenalistester.JSONEncodeSettings(map[string]any{"account": account}),
		})
		connectionIDs = append(connectionIDs, connection)
		return connection
	}
	createPipeline := func(connection, name string) string {
		schemas := simulatedApplicationPipelineSchemas(t, k, connection)
		mapping := simulatedApplicationSourceMapping()
		delete(mapping, "customer_id")
		delete(mapping, "first_name")
		inSchema := types.Prune(profileSchema, func(path string) bool { return path != "customer_id" })
		return k.CreatePipeline(connection, "User", krenalistester.PipelineToSet{
			Name: name, Enabled: true, InSchema: inSchema, OutSchema: schemas.Out,
			ExportMode:     krenalistester.CreateOrUpdate,
			Matching:       krenalistester.Matching{In: "first_name", Out: "first_name"},
			Transformation: &krenalistester.Transformation{Mapping: mapping},
		})
	}

	for _, resource := range []string{"pipeline", "connection", "expiry", "failure", "orphan-delete", "workspace"} {
		account := createAccount("Claim deletion " + resource)
		connectionA := createConnection("First "+resource+" destination", account)
		pipelineA := createPipeline(connectionA, "First "+resource+" pipeline")
		pipelineB := createPipeline(connectionA, "Second "+resource+" pipeline")
		var independentAccount, independentPipeline string
		if resource == "pipeline" {
			independentAccount = createAccount("Independent destination account")
			independentConnection := createConnection("Independent destination", independentAccount)
			independentPipeline = createPipeline(independentConnection, "Independent destination pipeline")
		}

		lock, err := warehousePool.Begin(t.Context())
		if err != nil {
			t.Fatalf("expected PostgreSQL table-lock transaction, got %v", err)
		}
		defer lock.Rollback(t.Context())
		_, err = lock.Exec(t.Context(), "LOCK TABLE krenalis_simulated_system_records IN ACCESS EXCLUSIVE MODE")
		if err != nil {
			t.Fatalf("expected simulated-system table lock, got %v", err)
		}
		if resource == "failure" {
			_, err = lock.Exec(t.Context(), `CREATE FUNCTION reject_simulated_destination() RETURNS trigger
				LANGUAGE plpgsql AS $body$ BEGIN RAISE EXCEPTION 'cooperative writer failure'; END $body$;
				CREATE TRIGGER reject_simulated_destination BEFORE INSERT ON krenalis_simulated_system_records
				FOR EACH ROW EXECUTE FUNCTION reject_simulated_destination()`)
			if err != nil {
				t.Fatalf("expected deterministic destination failure gate, got %v", err)
			}
		}
		runA := k.StartPipelineRun(pipelineA)
		waitForSimulatedRunClaim(t, k, runA, account)
		var runB string
		if resource != "failure" {
			runB = k.StartPipelineRun(pipelineB)
			waitForSimulatedRunPending(t, k, runB)
		}
		var independentRun string
		if independentPipeline != "" {
			independentRun = k.StartPipelineRun(independentPipeline)
			waitForSimulatedRunClaim(t, k, independentRun, independentAccount)
		}

		if resource == "pipeline" {
			pendingPipeline := createPipeline(connectionA, "Pending destination pipeline")
			pendingRun := k.StartPipelineRun(pendingPipeline)
			waitForSimulatedRunPending(t, k, pendingRun)
			k.Call("DELETE", "/v1/pipelines/"+pendingPipeline, nil, nil, nil)
		}

		path := "/v1/pipelines/" + pipelineA
		switch resource {
		case "connection", "orphan-delete":
			path = "/v1/connections/" + connectionA
		case "workspace":
			path = "/v1/workspaces/current"
		}
		if resource == "orphan-delete" {
			testSimulatedOrphanClaimDeletion(t, k, warehousePool, account, connectionA, runA)
		} else if resource == "expiry" {
			// Freeze only heartbeat refreshes, then expire A while its warehouse
			// request is blocked. Admission and claim release still work normally.
			k.ExecQueryTestDatabase(t.Context(), `CREATE FUNCTION freeze_simulated_heartbeat() RETURNS trigger
				LANGUAGE plpgsql AS $body$ BEGIN
				IF NEW.claimed_run IS NOT DISTINCT FROM OLD.claimed_run AND NEW.claim_ping_time > OLD.claim_ping_time
				THEN NEW.claim_ping_time = OLD.claim_ping_time; END IF; RETURN NEW; END $body$;
				CREATE TRIGGER freeze_simulated_heartbeat BEFORE UPDATE ON simulated_systems
				FOR EACH ROW EXECUTE FUNCTION freeze_simulated_heartbeat()`)
			k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_systems SET claim_ping_time = '2000-01-01' WHERE id = $1", account)
			waitForSimulatedRecoveredRunClaim(t, k, runB, account)
			var generation int64
			err = warehousePool.QueryRow(t.Context(), "SELECT generation FROM krenalis_simulated_system_writers WHERE simulated_system = $1",
				account).Scan(&generation)
			if err != nil {
				t.Fatalf("expected authoritative warehouse fence, got %v", err)
			}
			if generation < 3 {
				t.Fatalf("expected A's warehouse authority revoked before B admission, got generation %d", generation)
			}
			k.ExecQueryTestDatabase(t.Context(), `DROP TRIGGER freeze_simulated_heartbeat ON simulated_systems;
				DROP FUNCTION freeze_simulated_heartbeat()`)
		} else if resource == "failure" {
			// The warehouse trigger is installed when the held transaction commits.
		} else if resource == "workspace" {
			assertHTTPError(t, k.TryCall("DELETE", path, nil, nil, nil),
				http.StatusUnprocessableEntity, "SimulatedSystemInUse")
			waitForSimulatedRunClaim(t, k, runA, account)
			waitForSimulatedRunPending(t, k, runB)
		} else {
			k.Call("DELETE", path, nil, nil, nil)
		}

		err = lock.Commit(t.Context())
		if err != nil {
			t.Fatalf("expected simulated-system table lock release, got %v", err)
		}
		if resource == "orphan-delete" {
			assertSimulatedSystemDeleted(t, k, warehousePool, warehouseSettings, account)
		} else if resource == "workspace" {
			k.WaitForRunsCompletion(runA, runB)
		} else if resource == "failure" {
			waitForFailedSimulatedRun(t, k, runA)
			_, err = warehousePool.Exec(t.Context(), `DROP TRIGGER reject_simulated_destination ON krenalis_simulated_system_records;
				DROP FUNCTION reject_simulated_destination()`)
			if err != nil {
				t.Fatalf("expected failure gate removal, got %v", err)
			}
			runB = k.StartPipelineRun(pipelineB)
			k.WaitForRunsCompletion(runB)
		} else if resource == "expiry" {
			waitForFailedSimulatedRun(t, k, runA)
			k.WaitForRunsCompletion(runB)
			var count int
			err = warehousePool.QueryRow(t.Context(), "SELECT COUNT(*) FROM krenalis_simulated_system_records WHERE simulated_system = $1",
				account).Scan(&count)
			if err != nil {
				t.Fatalf("expected recovered owner records, got %v", err)
			}
			if count != 1 {
				t.Fatalf("expected only B's permitted create after recovery, got %d records", count)
			}
		} else if independentRun != "" {
			k.WaitForRunsCompletion(runB, independentRun)
		}
		if resource == "workspace" {
			for _, connection := range connectionIDs {
				var remaining int
				k.QueryRowTestDatabase(t.Context(), &remaining,
					"SELECT COUNT(*) FROM connections WHERE id = $1", connection)
				if remaining > 0 {
					k.DeleteConnection(connection)
				}
			}
			for _, account := range accountIDs {
				var remaining int
				k.QueryRowTestDatabase(t.Context(), &remaining,
					"SELECT COUNT(*) FROM simulated_systems WHERE id = $1", account)
				if remaining > 0 {
					k.Call("DELETE", "/v1/simulated/systems/"+account, nil, nil, nil)
				}
			}
			cleanupDeadline := time.Now().Add(30 * time.Second)
			var remaining int
			for {
				k.QueryRowTestDatabase(t.Context(), &remaining,
					"SELECT COUNT(*) FROM simulated_systems WHERE workspace = $1", workspace.ID)
				if remaining == 0 || time.Now().After(cleanupDeadline) {
					break
				}
				time.Sleep(25 * time.Millisecond)
			}
			if remaining != 0 {
				t.Fatalf("expected simulated system cleanup to finish, got %d remaining", remaining)
			}
		}
		if resource == "workspace" {
			k.Call("DELETE", path, nil, nil, nil)
		}

		if resource == "connection" {
			var remaining int
			k.QueryRowTestDatabase(t.Context(), &remaining,
				"SELECT COUNT(*) FROM simulated_systems WHERE id = $1", account)
			if remaining != 1 {
				t.Fatalf("expected connection deletion to retain simulated system %s, got %d rows", account, remaining)
			}
			testSimulatedDestinationAdmissionDeleteRace(t, k, createAccount,
				createConnection, createPipeline)
		}
		if resource == "workspace" {
			var remaining int
			k.QueryRowTestDatabase(t.Context(), &remaining,
				"SELECT COUNT(*) FROM workspaces WHERE id = $1", workspace.ID)
			if remaining != 0 {
				t.Fatalf("expected workspace deletion after runs ended, got %d workspace rows", remaining)
			}
		}
	}
}

// assertSimulatedSystemDeleted verifies cleanup and an uncanceled stale request
// after both the operational run and system metadata have disappeared.
func assertSimulatedSystemDeleted(t *testing.T, k *krenalistester.Krenalis, pool *pgxpool.Pool, settings *krenalistester.DBSettings, system string) {

	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		var count int
		k.QueryRowTestDatabase(t.Context(), &count, "SELECT COUNT(*) FROM simulated_systems WHERE id = $1", system)
		if count == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected successful system cleanup, got %d metadata rows", count)
		}
		time.Sleep(25 * time.Millisecond)
	}
	encoded, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("expected warehouse settings, got %v", err)
	}
	warehouse := warehouses.Registered("PostgreSQL").New(&simulatedOwnershipSettings{encoded}, nil).(*postgresql.PostgreSQL)
	t.Cleanup(func() { warehouse.Close() })
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
	err = warehouse.MergeSimulatedSystemRecords(t.Context(), system, 2, table,
		[][]any{{system, "late-owner", time.Now().UTC(), json.Value(`{"owner":"stale"}`)}})
	if err != nil {
		var count int
		if err := pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM krenalis_simulated_system_records WHERE simulated_system = $1", system).Scan(&count); err != nil {
			t.Fatalf("expected post-cleanup record count, got %v", err)
		}
		if count != 0 {
			t.Fatalf("expected no stale resurrection after metadata deletion, got %d rows", count)
		}
		return
	}

	t.Fatal("expected old owner rejected after successful cleanup, got nil")
}

func testSimulatedDestinationAdmissionDeleteRace(t *testing.T, k *krenalistester.Krenalis, createAccount func(string) string, createConnection func(string, string) string, createPipeline func(string, string) string) {

	t.Helper()
	account := createAccount("Admission deletion race")
	connection := createConnection("Admission deletion race", account)
	pipeline := createPipeline(connection, "Admission deletion race")
	settings := k.TestDatabaseSettings()
	operationalPool, err := krenalistester.ConnectionPool(t.Context(), &settings)
	if err != nil {
		t.Fatalf("expected operational PostgreSQL pool, got %v", err)
	}
	defer operationalPool.Close()

	connectionLock, err := operationalPool.Begin(t.Context())
	if err != nil {
		t.Fatalf("expected connection lock transaction, got %v", err)
	}
	defer connectionLock.Rollback(context.Background())
	_, err = connectionLock.Exec(t.Context(), "SELECT id FROM connections WHERE id = $1 FOR UPDATE", connection)
	if err != nil {
		t.Fatalf("expected connection row lock, got %v", err)
	}

	run := k.StartPipelineRun(pipeline)
	waitForSimulatedRunPending(t, k, run)
	waitForOperationalLockWait(t, operationalPool, "SELECT c.connector, c.role, c.simulated_system FROM pipelines_runs%")
	k.Call("DELETE", "/v1/pipelines/"+pipeline, nil, nil, nil)
	var remaining int
	err = operationalPool.QueryRow(t.Context(), "SELECT COUNT(*) FROM pipelines_runs WHERE id = $1", run).Scan(&remaining)
	if err != nil {
		t.Fatalf("expected run-row lookup, got %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected deleted pipeline to remove pending run, got %d rows", remaining)
	}

	err = connectionLock.Commit(t.Context())
	if err != nil {
		t.Fatalf("expected connection row lock release, got %v", err)
	}
}

// testSimulatedOrphanClaimDeletion holds the warehouse fence while resource
// deletion cascades the run away, proving authority survives independently.
func testSimulatedOrphanClaimDeletion(t *testing.T, k *krenalistester.Krenalis, pool *pgxpool.Pool, system, connection, run string) {

	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		var installed bool
		if err := pool.QueryRow(t.Context(), "SELECT EXISTS (SELECT FROM krenalis_simulated_system_writers WHERE simulated_system = $1 AND generation = 2)", system).Scan(&installed); err != nil {
			t.Fatalf("expected installed writer authority, got %v", err)
		}
		if installed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("expected installed warehouse generation 2, got none")
		}
		time.Sleep(25 * time.Millisecond)
	}
	fenceLock, err := pool.Begin(t.Context())
	if err != nil {
		t.Fatalf("expected warehouse fence lock, got %v", err)
	}
	defer fenceLock.Rollback(t.Context())
	var generation int64
	if err := fenceLock.QueryRow(t.Context(), "SELECT generation FROM krenalis_simulated_system_writers WHERE simulated_system = $1 FOR UPDATE", system).Scan(&generation); err != nil {
		t.Fatalf("expected retained writer generation, got %v", err)
	}

	k.DeleteConnection(connection)
	var runCount int
	k.QueryRowTestDatabase(t.Context(), &runCount, "SELECT COUNT(*) FROM pipelines_runs WHERE id = $1", run)
	if runCount != 0 {
		t.Fatalf("expected run-row cascade, got %d rows", runCount)
	}
	var retained bool
	k.QueryRowTestDatabase(t.Context(), &retained,
		"SELECT claimed_run = $1 AND writer_generation = 2 FROM simulated_systems WHERE id = $2", run, system)
	if !retained {
		t.Fatal("expected writer authority retained until the warehouse seal succeeds, got released claim")
	}
	k.Call("DELETE", "/v1/simulated/systems/"+system, nil, nil, nil)
	k.Call("DELETE", "/v1/simulated/systems/"+system, nil, nil, nil)
	var deleting bool
	k.QueryRowTestDatabase(t.Context(), &deleting,
		"SELECT status = 'Deleting' AND claimed_run = $1 FROM simulated_systems WHERE id = $2", run, system)
	if !deleting {
		t.Fatal("expected retryable Deleting metadata with retained authority, got premature cleanup")
	}
	k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_systems SET claim_ping_time = '2000-01-01' WHERE id = $1", system)

	err = fenceLock.Commit(t.Context())
	if err != nil {
		t.Fatalf("expected warehouse seal to become possible, got %v", err)
	}

}

func waitForFailedSimulatedRun(t *testing.T, k *krenalistester.Krenalis, run string) {

	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		var ended bool
		k.QueryRowTestDatabase(t.Context(), &ended,
			"SELECT end_time IS NOT NULL AND error <> '' FROM pipelines_runs WHERE id = $1", run)
		if ended {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("expected cooperative failed run %s to finish, got no failure", run)
}
func waitForOperationalLockWait(t *testing.T, pool *pgxpool.Pool, queryPattern string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		var waiting bool
		err := pool.QueryRow(t.Context(), `SELECT EXISTS (SELECT FROM pg_stat_activity
			WHERE wait_event_type = 'Lock' AND query LIKE $1 AND pid <> pg_backend_pid())`,
			queryPattern).Scan(&waiting)
		if err != nil {
			t.Fatalf("expected PostgreSQL lock-wait observation, got %v", err)
		}
		if waiting {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("expected PostgreSQL lock wait for query %q", queryPattern)
}

func waitForSimulatedRecoveredRunClaim(t *testing.T, k *krenalistester.Krenalis, run, system string) {

	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		var claimed bool
		k.QueryRowTestDatabase(t.Context(), &claimed,
			"SELECT COALESCE(claimed_run = $1, FALSE) FROM simulated_systems WHERE id = $2", run, system)
		if claimed {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("expected recovered run %s to claim %s, got no admission", run, system)
}
