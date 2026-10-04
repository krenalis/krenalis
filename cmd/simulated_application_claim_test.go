// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

func TestSimulatedApplicationUsesAdmittedAccount(t *testing.T) {
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
	database := "test_simulated_claim_" + strconv.FormatInt(time.Now().UnixNano(), 10)
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
	profileSchema = types.Object(append(profileSchema.Properties().Slice(), types.Property{
		Name: "photo_url", Type: types.String().AsURL(), ReadOptional: true,
	}))
	var workspace struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/workspaces", http.Header{"Krenalis-Workspace": nil}, map[string]any{
		"name": "Simulated Application PostgreSQL", "environment": "development",
		"profileSchema": profileSchema,
		"warehouse":     map[string]any{"platform": "PostgreSQL", "mode": "Normal", "settings": warehouseSettings},
	}, &workspace)
	k.SetWorkspaceID(workspace.ID)

	createAccount := func(name string) string {
		t.Helper()
		var account struct {
			ID string `json:"id"`
		}
		k.Call("POST", "/v1/simulated/systems", nil, map[string]any{
			"name": name, "userCount": 0, "countries": map[string]int{"IT": 100},
		}, &account)
		return account.ID
	}
	sourceAccount := createAccount("Source account")
	staleAccount := createAccount("Stale destination account")
	claimedAccount := createAccount("Claimed destination account")

	warehousePool, err := krenalistester.ConnectionPool(t.Context(), warehouseSettings)
	if err != nil {
		t.Fatalf("expected simulated-system warehouse connection, got %v", err)
	}
	defer warehousePool.Close()
	data, err := json.Marshal(map[string]any{
		"customer_id": "source-customer-1",
		"first_name":  "Ada",
		"last_name":   "Rossi",
		"email":       "ada@example.test",
		"gender":      "female",
		"birth_date":  "1990-01-02",
		"current_age": 36,
		"phone":       "+393331234567",
		"site_url":    "https://profiles.example.test/ada",
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
		Name: "Simulated Application source", Role: krenalistester.Source, Connector: "simulated-application",
		Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": sourceAccount}),
	})
	sourceSchemas := simulatedApplicationPipelineSchemas(t, k, sourceConnection)
	sourceMapping := simulatedApplicationSourceMapping()
	sourceMapping["photo_url"] = "photo"
	sourcePipeline := k.CreatePipeline(sourceConnection, "User", krenalistester.PipelineToSet{
		Name: "Import simulated source customer", Enabled: true,
		InSchema: sourceSchemas.In, OutSchema: sourceSchemas.Out,
		Transformation: &krenalistester.Transformation{Mapping: sourceMapping},
	})
	sourceRun := k.StartPipelineRun(sourcePipeline)
	k.WaitForRunsCompletion(sourceRun)
	profiles, total := k.Profiles([]string{"email", "photo_url"}, "", false, 0, 10)
	if total != 1 || len(profiles) != 1 || profiles[0].Attributes["email"] != "ada@example.test" {
		t.Fatalf("expected one imported source profile, got total=%d profiles=%+v", total, profiles)
	}
	if profiles[0].Attributes["photo_url"] != "https://krenalis.test/photos/catalog/256/face-000001.webp" {
		t.Fatalf("expected Source photo mapped to profile photo_url, got %v", profiles[0].Attributes["photo_url"])
	}

	mode := krenalistester.Server
	createConnection := func(name, account string) string {
		t.Helper()
		return k.CreateConnection(krenalistester.ConnectionToCreate{
			Name: name, Role: krenalistester.Destination, Connector: "simulated-application",
			SendingMode: &mode,
			Settings:    krenalistester.JSONEncodeSettings(map[string]any{"account": account}),
		})
	}
	staleConnection := createConnection("Stale state destination", staleAccount)
	destinationSchemas := simulatedApplicationPipelineSchemas(t, k, staleConnection)
	createMapping := simulatedApplicationSourceMapping()
	delete(createMapping, "customer_id")
	delete(createMapping, "first_name")
	createInSchema := types.Prune(profileSchema, func(path string) bool {
		return path != "customer_id" && path != "photo_url"
	})
	createPipeline := k.CreatePipeline(staleConnection, "User", krenalistester.PipelineToSet{
		Name: "Create simulated destination customer", Enabled: true,
		InSchema: createInSchema, OutSchema: destinationSchemas.Out,
		ExportMode:     krenalistester.CreateOrUpdate,
		Matching:       krenalistester.Matching{In: "first_name", Out: "first_name"},
		Transformation: &krenalistester.Transformation{Mapping: createMapping},
	})
	claimInSchema := types.Prune(profileSchema, func(path string) bool {
		return path == "last_name" || path == "address.street"
	})
	claimOutSchema := types.Prune(destinationSchemas.Out, func(path string) bool {
		return path == "last_name" || path == "address.street"
	})
	claimPipeline := k.CreatePipeline(staleConnection, "User", krenalistester.PipelineToSet{
		Name: "Contend for simulated destination account", Enabled: true,
		InSchema: claimInSchema, OutSchema: claimOutSchema, ExportMode: krenalistester.UpdateOnly,
		Matching:       krenalistester.Matching{In: "last_name", Out: "last_name"},
		Transformation: &krenalistester.Transformation{Mapping: map[string]string{"address.street": "address.street"}},
	})

	// The local connection state still holds staleAccount when admission reads the
	// authoritative claimedAccount binding from PostgreSQL.
	k.ExecQueryTestDatabase(t.Context(), "UPDATE connections SET simulated_system = $1 WHERE id = $2",
		claimedAccount, staleConnection)
	lock, err := warehousePool.Begin(t.Context())
	if err != nil {
		t.Fatalf("expected PostgreSQL table-lock transaction, got %v", err)
	}
	defer lock.Rollback(context.Background())
	_, err = lock.Exec(t.Context(), "LOCK TABLE krenalis_simulated_system_records IN ACCESS EXCLUSIVE MODE")
	if err != nil {
		t.Fatalf("expected simulated-system record table lock, got %v", err)
	}

	firstRun := k.StartPipelineRun(createPipeline)
	waitForSimulatedRunClaim(t, k, firstRun, claimedAccount)
	claimContenderRun := k.StartPipelineRun(claimPipeline)
	waitForSimulatedRunPending(t, k, claimContenderRun)
	assertHTTPError(t, k.TryCall("PUT", "/v1/connections/"+staleConnection, nil, map[string]any{
		"name": "Stale state destination", "simulatedSystem": staleAccount, "sendingMode": "Server",
	}, nil), http.StatusBadRequest, "simulatedSystem cannot be updated")

	err = lock.Commit(t.Context())
	if err != nil {
		t.Fatalf("expected destination table lock release, got %v", err)
	}
	k.WaitForRunsCompletionAllowFailed(firstRun)
	firstRunResult := k.PipelineRun(firstRun)
	if firstRunResult.Failed != [7]int{} {
		k.RestartServer(t.Context())
		var message string
		k.QueryRowTestDatabase(t.Context(), &message,
			"SELECT message FROM pipelines_errors WHERE pipeline = $1 ORDER BY timeslot DESC LIMIT 1", createPipeline)
		t.Fatalf("expected Destination create to succeed, got run=%+v error=%q", firstRunResult, message)
	}
	k.WaitForRunsCompletion(claimContenderRun)
	var destinationCount int
	err = warehousePool.QueryRow(t.Context(), "SELECT COUNT(*) FROM krenalis_simulated_system_records WHERE simulated_system = $1",
		claimedAccount).Scan(&destinationCount)
	if err != nil {
		t.Fatalf("expected claimed-account record count, got %v", err)
	}
	if destinationCount != 1 {
		t.Fatalf("expected one created destination record, got %d", destinationCount)
	}
	var firstRecordID string
	err = warehousePool.QueryRow(t.Context(), "SELECT external_id FROM krenalis_simulated_system_records WHERE simulated_system = $1",
		claimedAccount).Scan(&firstRecordID)
	if err != nil {
		t.Fatalf("expected generated destination external ID, got %v", err)
	}
	firstRecordID, firstUpdatedAt, firstDocument := simulatedDestinationRecord(t, warehousePool, claimedAccount, firstRecordID)
	if firstRecordID == "" || strings.Contains(firstRecordID, ":") {
		t.Fatalf("expected generated external ID without a colon, got %q", firstRecordID)
	}
	if firstDocument["customer_id"] != firstRecordID {
		t.Fatalf("expected customer_id and external_id to agree, got %v and %q", firstDocument["customer_id"], firstRecordID)
	}
	var staleAccountCount int
	err = warehousePool.QueryRow(t.Context(), "SELECT COUNT(*) FROM krenalis_simulated_system_records WHERE simulated_system = $1",
		staleAccount).Scan(&staleAccountCount)
	if err != nil {
		t.Fatalf("expected stale-account record count, got %v", err)
	}
	if staleAccountCount != 0 {
		t.Fatalf("expected no write to stale account %q, got %d rows", staleAccount, staleAccountCount)
	}
	firstAddress, ok := firstDocument["address"].(map[string]any)
	if !ok || firstAddress["street"] != "Via Roma 1" {
		t.Fatalf("expected unchanged create address before update, got %v", firstDocument["address"])
	}
	assertUTCmicrosecond(t, firstUpdatedAt)

	updateSchema := types.Prune(destinationSchemas.Out, func(path string) bool {
		return path == "first_name" || path == "address.street"
	})
	updateInSchema := types.Prune(profileSchema, func(path string) bool {
		return path == "first_name" || path == "address.street"
	})
	updatePipeline := k.CreatePipeline(staleConnection, "User", krenalistester.PipelineToSet{
		Name: "Partially update simulated destination customer", Enabled: true,
		InSchema: updateInSchema, OutSchema: updateSchema, ExportMode: krenalistester.UpdateOnly,
		Matching: krenalistester.Matching{In: "first_name", Out: "first_name"},
		Transformation: &krenalistester.Transformation{Mapping: map[string]string{
			"address.street": "address.street ', Suite 2'",
		}},
	})
	updateRun := k.StartPipelineRun(updatePipeline)
	k.WaitForRunsCompletion(updateRun)
	secondRecordID, secondUpdatedAt, secondDocument := simulatedDestinationRecord(t, warehousePool, claimedAccount, firstRecordID)
	if secondRecordID != firstRecordID {
		t.Fatalf("expected update to preserve destination external ID %q, got %q", firstRecordID, secondRecordID)
	}
	if !secondUpdatedAt.After(firstUpdatedAt) {
		t.Fatalf("expected destination update timestamp after %v, got %v", firstUpdatedAt, secondUpdatedAt)
	}
	address, ok := secondDocument["address"].(map[string]any)
	if !ok || address["street"] != "Via Roma 1, Suite 2" || address["municipality"] != "Roma" ||
		address["state_prov"] != "RM" || address["postal_code"] != "00100" || address["country"] != "IT" {
		t.Fatalf("expected nested partial update to preserve the complete address, got %v", secondDocument["address"])
	}
	assertUTCmicrosecond(t, secondUpdatedAt)
	t.Logf("destination snapshots: account=%s external_id=%s create_street=%q T1=%s update_street=%q T2=%s",
		claimedAccount, firstRecordID, firstAddress["street"], firstUpdatedAt, address["street"], secondUpdatedAt)

	var claimedID string
	k.QueryRowTestDatabase(t.Context(), &claimedID, "SELECT simulated_system FROM pipelines_runs WHERE id = $1", firstRun)
	if claimedID != claimedAccount {
		t.Fatalf("expected first run claim %q, got %q", claimedAccount, claimedID)
	}
	k.QueryRowTestDatabase(t.Context(), &claimedID, "SELECT simulated_system FROM pipelines_runs WHERE id = $1", claimContenderRun)
	if claimedID != claimedAccount {
		t.Fatalf("expected pending run to acquire released claim %q, got %q", claimedAccount, claimedID)
	}

	assertHTTPError(t, k.TryCall("PUT", "/v1/connections/"+staleConnection, nil, map[string]any{
		"name": "Stale state destination", "simulatedSystem": staleAccount, "sendingMode": "Server",
	}, nil), http.StatusBadRequest, "simulatedSystem cannot be updated")

	assertHTTPError(t, k.TryCall("PUT", "/v1/connections/"+sourceConnection, nil, map[string]any{
		"name": "Simulated Application source", "simulatedSystem": claimedAccount,
	}, nil), http.StatusBadRequest, "simulatedSystem cannot be updated")
	claimedSource := k.CreateConnection(krenalistester.ConnectionToCreate{
		Name: "Claimed account source", Role: krenalistester.Source, Connector: "simulated-application",
		Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": claimedAccount}),
	})
	schemaJSON, err := sourceSchemas.In.MarshalJSON()
	if err != nil {
		t.Fatalf("expected Source schema serialization, got %v", err)
	}
	var sourceResponse struct {
		Users  []map[string]any `json:"users"`
		Cursor string           `json:"cursor"`
	}
	path := fmt.Sprintf("/v1/connections/%s/users?schema=%s", claimedSource, url.QueryEscape(string(schemaJSON)))
	k.Call("GET", path, nil, nil, &sourceResponse)
	if len(sourceResponse.Users) != 1 || sourceResponse.Users[0]["customer_id"] != firstRecordID ||
		sourceResponse.Users[0]["first_name"] != "Ada" {
		t.Fatalf("expected Source to return the Destination-created and updated record, got %+v", sourceResponse.Users)
	}
	if sourceResponse.Users[0]["address"].(map[string]any)["street"] != "Via Roma 1, Suite 2" {
		t.Fatalf("expected Source to expose the destination patch, got %+v", sourceResponse.Users[0]["address"])
	}

	k.RestartServer(t.Context())
	k.SetWorkspaceID(workspace.ID)
	thirdRun := k.StartPipelineRun(updatePipeline)
	k.WaitForRunsCompletion(thirdRun)
	thirdRecordID, thirdUpdatedAt, _ := simulatedDestinationRecord(t, warehousePool, claimedAccount, firstRecordID)
	if thirdRecordID != firstRecordID || !thirdUpdatedAt.After(secondUpdatedAt) {
		t.Fatalf("expected restarted Destination runtime to advance persisted timestamp after %v, got id=%q timestamp=%v",
			secondUpdatedAt, thirdRecordID, thirdUpdatedAt)
	}
	assertUTCmicrosecond(t, thirdUpdatedAt)
	k.QueryRowTestDatabase(t.Context(), &claimedID, "SELECT simulated_system FROM pipelines_runs WHERE id = $1", thirdRun)
	if claimedID != claimedAccount {
		t.Fatalf("expected post-reload run to claim configured account %q, got %q", claimedAccount, claimedID)
	}
}

func assertUTCmicrosecond(t *testing.T, value time.Time) {
	t.Helper()
	if value.Location() != time.UTC || value.Nanosecond()%1000 != 0 {
		t.Fatalf("expected UTC microsecond timestamp, got %v", value)
	}
}

func simulatedApplicationPipelineSchemas(t *testing.T, k *krenalistester.Krenalis, connection string) struct {
	In  types.Type `json:"in"`
	Out types.Type `json:"out"`
} {
	t.Helper()
	var schemas struct {
		In  types.Type `json:"in"`
		Out types.Type `json:"out"`
	}
	k.Call("GET", fmt.Sprintf("/v1/connections/%s/pipelines/schemas/User", connection), nil, nil, &schemas)
	if !schemas.In.Valid() || !schemas.Out.Valid() {
		t.Fatalf("expected valid Simulated Application pipeline schemas, got in=%v out=%v", schemas.In, schemas.Out)
	}
	return schemas
}

func simulatedApplicationProfileSchema() types.Type {
	return types.Object([]types.Property{
		{Name: "customer_id", Type: types.String(), ReadOptional: true},
		{Name: "first_name", Type: types.String(), ReadOptional: true},
		{Name: "last_name", Type: types.String(), ReadOptional: true},
		{Name: "email", Type: types.String().AsEmail(), ReadOptional: true},
		{Name: "gender", Type: types.String(), ReadOptional: true},
		{Name: "birth_date", Type: types.Date(), ReadOptional: true},
		{Name: "current_age", Type: types.Int(8), ReadOptional: true},
		{Name: "phone", Type: types.String().AsPhone(), ReadOptional: true},
		{Name: "site_url", Type: types.String().AsURL(), ReadOptional: true},
		{Name: "address", Type: types.Object([]types.Property{
			{Name: "street", Type: types.String(), ReadOptional: true},
			{Name: "municipality", Type: types.String(), ReadOptional: true},
			{Name: "state_prov", Type: types.String(), ReadOptional: true},
			{Name: "postal_code", Type: types.String(), ReadOptional: true},
			{Name: "country", Type: types.String().AsCountry(types.ISO3166Alpha2), ReadOptional: true},
		}), ReadOptional: true},
		{Name: "photo", Type: types.String().AsURL(), ReadOptional: true},
	})
}

func simulatedApplicationSourceMapping() map[string]string {
	return map[string]string{
		"customer_id": "customer_id", "first_name": "first_name", "last_name": "last_name", "email": "email",
		"gender": "gender", "birth_date": "birth_date", "current_age": "current_age", "phone": "phone",
		"site_url": "site_url", "address.street": "address.street", "address.municipality": "address.municipality",
		"address.state_prov": "address.state_prov", "address.postal_code": "address.postal_code",
		"address.country": "address.country", "photo": "photo",
	}
}

func simulatedDestinationRecord(t *testing.T, pool *pgxpool.Pool, account, externalID string) (string, time.Time, map[string]any) {
	t.Helper()
	var id string
	var updatedAt time.Time
	var data string
	err := pool.QueryRow(t.Context(), `SELECT external_id, updated_at, data::text
		FROM krenalis_simulated_system_records WHERE simulated_system = $1 AND external_id = $2`, account, externalID).Scan(&id, &updatedAt, &data)
	if err != nil {
		t.Fatalf("expected simulated destination record for %q, got %v", account, err)
	}
	var document map[string]any
	err = json.Unmarshal([]byte(data), &document)
	if err != nil {
		t.Fatalf("expected persisted destination JSON, got %v", err)
	}
	return id, updatedAt, document
}

func waitForSimulatedRunClaim(t *testing.T, k *krenalistester.Krenalis, run, account string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		var claimed bool
		k.QueryRowTestDatabase(t.Context(), &claimed, `SELECT EXISTS (
			SELECT FROM simulated_systems AS a
			INNER JOIN pipelines_runs AS r ON r.id = a.claimed_run
			WHERE a.id = $1 AND r.id = $2 AND r.simulated_system = $1
				AND a.claim_ping_time IS NOT NULL AND r.node IS NOT NULL AND r.end_time IS NULL)`, account, run)
		if claimed {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("expected live run %q to claim simulated system %q", run, account)
}

func waitForSimulatedRunPending(t *testing.T, k *krenalistester.Krenalis, run string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var pending bool
		k.QueryRowTestDatabase(t.Context(), &pending, `SELECT EXISTS (
			SELECT FROM pipelines_runs WHERE id = $1 AND node IS NULL AND end_time IS NULL)`, run)
		if pending {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("expected run %q to remain pending behind the live account claim", run)
}
