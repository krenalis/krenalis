// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd_test

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/jackc/pgx/v5/pgxpool"
)

type simulatedSystemRecordState struct {
	UpdatedAt time.Time
	Data      string
}

func TestSimulatedSystemGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.SetSyntheticPhotosDir(simulatedSystemCatalogFixture(t))
	model := simulatedSystemModelFixture(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/synthetic-world/it-2026a/model.json" {
			t.Errorf("expected fixed model request, got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(model)
	}))
	defer server.Close()
	k.SetExternalAssetsURLs([]string{server.URL})
	testSimulatedSystemGeneration(t, k, "http://"+k.Addr()+"/photos/")
}

func testSimulatedSystemGeneration(t *testing.T, k *krenalistester.Krenalis, photoPrefix string) {

	t.Helper()
	k.Start()
	defer k.Stop()

	settings := &krenalistester.DBSettings{}
	err := json.Unmarshal(krenalistester.PostgresWarehouseSettings(), settings)
	if err != nil {
		t.Fatalf("expected PostgreSQL warehouse settings, got %v", err)
	}
	database := "test_simulated_generation_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	pool, err := krenalistester.ConnectionPool(t.Context(), settings)
	if err != nil {
		t.Fatalf("expected PostgreSQL connection pool, got %v", err)
	}
	_, err = pool.Exec(t.Context(), "CREATE DATABASE "+database)
	pool.Close()
	if err != nil {
		t.Fatalf("expected isolated warehouse database, got %v", err)
	}
	cleanupSettings := *settings
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		pool, err := krenalistester.ConnectionPool(ctx, &cleanupSettings)
		if err != nil {
			t.Errorf("expected cleanup warehouse pool, got %v", err)
			return
		}
		defer pool.Close()
		_, err = pool.Exec(ctx, "DROP DATABASE "+database+" WITH (FORCE)")
		if err != nil {
			t.Errorf("expected isolated warehouse cleanup, got %v", err)
		}
	}()
	settings.Database = database

	profileSchema := types.Object([]types.Property{
		{Name: "email", Type: types.String().AsEmail().WithMaxLength(254), ReadOptional: true},
		{Name: "photo_url", Type: types.String().AsURL(), ReadOptional: true},
		{Name: "education_level", Type: types.String(), ReadOptional: true},
		{Name: "primary_activity", Type: types.String(), ReadOptional: true},
		{Name: "annual_income_eur", Type: types.Int(32).Unsigned(), ReadOptional: true},
		{Name: "job_family", Type: types.String(), ReadOptional: true},
		{Name: "industry", Type: types.String(), ReadOptional: true},
		{Name: "job_level", Type: types.String(), ReadOptional: true},
	})
	var workspace struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/workspaces", http.Header{"Krenalis-Workspace": nil}, map[string]any{
		"name": "generation", "environment": "development", "profileSchema": profileSchema,
		"warehouse": map[string]any{"platform": "PostgreSQL", "mode": "Normal", "settings": settings},
	}, &workspace)
	k.SetWorkspaceID(workspace.ID)

	valid := map[string]any{
		"name": "initial customers", "userCount": 513,
		"duplicateRecordPercent": 25.50, "countries": map[string]int{"IT": 40},
	}
	for _, test := range []struct {
		name   string
		change func(map[string]any)
		want   string
	}{
		{"count above limit", func(body map[string]any) { body["userCount"] = 10_000_001 }, "userCount must be between"},
		{"unsupported country", func(body map[string]any) { body["countries"] = map[string]int{"FR": 40} }, "not supported"},
		{"empty countries", func(body map[string]any) { body["countries"] = map[string]int{} }, "countries must contain IT"},
		{"percent above limit", func(body map[string]any) { body["duplicateRecordPercent"] = 50.01 }, "duplicateRecordPercent must be between"},
		{"fractional precision", func(body map[string]any) { body["duplicateRecordPercent"] = 1.001 }, "duplicateRecordPercent is not valid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := maps.Clone(valid)
			test.change(body)
			assertEnvironmentBadRequest(t, k.TryCall("POST", "/v1/simulated/systems", nil, body, nil), test.want)
		})
	}
	var before int
	k.QueryRowTestDatabase(t.Context(), &before, "SELECT COUNT(*) FROM simulated_systems WHERE workspace = $1", workspace.ID)
	if before != 0 {
		t.Fatalf("expected no metadata after invalid requests, got %d", before)
	}

	var created struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/simulated/systems", nil, valid, &created)
	deadline := time.Now().Add(45 * time.Second)
	var detail struct {
		Status               string         `json:"status"`
		UserCount            int            `json:"userCount"`
		GeneratedRecordCount int            `json:"generatedRecordCount"`
		Countries            map[string]int `json:"countries"`
	}
	for {
		k.Call("GET", "/v1/simulated/systems/"+created.ID, nil, nil, &detail)
		if detail.Status == "Available" || detail.Status == "Failed" || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if detail.Status != "Available" || detail.UserCount != 513 || detail.GeneratedRecordCount != 513 || detail.Countries["IT"] != 40 {
		t.Fatalf("expected completed account with 513 records, got %+v", detail)
	}
	var policy string
	k.QueryRowTestDatabase(t.Context(), &policy,
		"SELECT generation_policy_version FROM simulated_systems WHERE id = $1", created.ID)
	if policy != "generation-v4" {
		t.Fatalf("expected successor generation-v4 policy, got %s", policy)
	}
	var ordinary map[string]json.Value
	k.Call("GET", "/v1/simulated/systems/"+created.ID, nil, nil, &ordinary)
	for _, field := range []string{"oracle", "personID", "generationCheckpoint", "generationPolicyVersion", "generationError"} {
		if _, ok := ordinary[field]; ok {
			t.Fatalf("expected no %s in ordinary API payload, got %s", field, ordinary[field])
		}
	}
	var progressEvents int
	k.QueryRowTestDatabase(t.Context(), &progressEvents,
		"SELECT COUNT(*) FROM notifications WHERE name = 'UpdateSimulatedSystemGeneration'")
	if progressEvents < 4 {
		t.Fatalf("expected three batch checkpoints and Available notification, got %d events", progressEvents)
	}

	warehousePool, err := krenalistester.ConnectionPool(t.Context(), settings)
	if err != nil {
		t.Fatalf("expected generated warehouse pool, got %v", err)
	}
	defer warehousePool.Close()
	var total, unique int
	err = warehousePool.QueryRow(t.Context(), "SELECT COUNT(*), COUNT(DISTINCT external_id)"+
		" FROM krenalis_simulated_system_records WHERE simulated_system = $1", created.ID).Scan(&total, &unique)
	if err != nil {
		t.Fatalf("expected warehouse record counts, got %v", err)
	}
	if total != 513 || unique != 513 {
		t.Fatalf("expected 513 distinct warehouse records, got total=%d unique=%d", total, unique)
	}
	var timestamped int
	var earliestUpdatedAt time.Time
	err = warehousePool.QueryRow(t.Context(), "SELECT COUNT(updated_at), MIN(updated_at)"+
		" FROM krenalis_simulated_system_records WHERE simulated_system = $1", created.ID).Scan(&timestamped, &earliestUpdatedAt)
	if err != nil {
		t.Fatalf("expected generated record timestamps, got %v", err)
	}
	if timestamped != total || earliestUpdatedAt.Location() != time.UTC || earliestUpdatedAt.Nanosecond()%1000 != 0 {
		t.Fatalf("expected UTC microsecond generated timestamps, got count=%d timestamp=%v", timestamped, earliestUpdatedAt)
	}
	var externalID, data string
	err = warehousePool.QueryRow(t.Context(), "SELECT external_id, data::text FROM krenalis_simulated_system_records"+
		" WHERE simulated_system = $1 LIMIT 1", created.ID).Scan(&externalID, &data)
	if err != nil {
		t.Fatalf("expected ordinary warehouse record, got %v", err)
	}
	var document map[string]json.Value
	err = json.Unmarshal([]byte(data), &document)
	if err != nil {
		t.Fatalf("expected generated customer document JSON, got %v", err)
	}
	for _, field := range []string{"customer_id", "gender", "birth_date", "current_age", "site_url", "address", "photo",
		"education_level", "primary_activity", "annual_income_eur"} {
		if _, ok := document[field]; !ok {
			t.Fatalf("expected generated customer document field %q, got %s", field, data)
		}
	}
	var customerID string
	err = json.Unmarshal(document["customer_id"], &customerID)
	if err != nil || customerID != externalID {
		t.Fatalf("expected customer_id to equal external_id %q, got %q", externalID, customerID)
	}
	var photo string
	err = json.Unmarshal(document["photo"], &photo)
	if err != nil || !strings.HasPrefix(photo, photoPrefix) {
		t.Fatalf("expected absolute configured photo URL, got %q", photo)
	}
	for _, field := range []string{"source_record_id", "photo_id", "photo_url", "country", "person_id", "oracle",
		"role", "employer", "employer_id", "company_id", "company_index", "employee_count", "size_band"} {
		if _, ok := document[field]; ok {
			t.Fatalf("expected generated customer document field %q to be absent, got %s", field, data)
		}
	}
	simulatedConnection := k.CreateConnection(krenalistester.ConnectionToCreate{
		Name: "Simulated Application source", Role: krenalistester.Source, Connector: "simulated-application",
		Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": created.ID}),
	})
	var sourceSchemas struct {
		In types.Type `json:"in"`
	}
	k.Call("GET", "/v1/connections/"+simulatedConnection+"/pipelines/schemas/User", nil, nil, &sourceSchemas)
	if !sourceSchemas.In.Valid() {
		t.Fatalf("expected Simulated Application source schema, got %v", sourceSchemas.In)
	}
	simulatedSourceSchema := types.Prune(sourceSchemas.In, func(path string) bool {
		switch path {
		case "email", "photo", "education_level", "primary_activity", "annual_income_eur",
			"job_family", "industry", "job_level":
			return true
		default:
			return false
		}
	})
	simulatedOutputSchema := profileSchema
	pipeline := k.CreatePipeline(simulatedConnection, "User", krenalistester.PipelineToSet{
		Name: "Read Simulated Application users", Enabled: true, InSchema: simulatedSourceSchema,
		OutSchema: simulatedOutputSchema,
		Transformation: &krenalistester.Transformation{Mapping: map[string]string{
			"email": "email", "photo_url": "photo", "education_level": "education_level",
			"primary_activity": "primary_activity", "annual_income_eur": "annual_income_eur",
			"job_family": "job_family", "industry": "industry", "job_level": "job_level",
		}},
	})
	runID := k.StartPipelineRun(pipeline)
	k.WaitForRunsCompletion(runID)

	run := k.PipelineRun(runID)
	if run.Error != "" || run.Failed != [7]int{} {
		t.Fatalf("expected successful Source readback and mapping, got %+v", run)
	}
	k.RunIdentityResolutionAndWait()
	properties := []string{"photo_url", "education_level", "primary_activity", "annual_income_eur"}
	profiles, profileCount := k.Profiles(properties, "", false, 0, 10)
	if profileCount == 0 || len(profiles) == 0 {
		t.Fatalf("expected generated profiles, got %d", profileCount)
	}
	for _, profile := range profiles {
		mappedPhoto, ok := profile.Attributes["photo_url"].(string)
		if !ok {
			t.Fatalf("expected mapped photo URL, got %v", profile.Attributes["photo_url"])
		}
		if !strings.HasPrefix(mappedPhoto, photoPrefix) {
			t.Fatalf("expected photo mapped to photo_url under %q, got %q", photoPrefix, mappedPhoto)
		}
		for _, name := range []string{"education_level", "primary_activity", "annual_income_eur"} {
			if _, ok := profile.Attributes[name]; !ok {
				t.Fatalf("expected generated %s imported through pipeline, got %v", name, profile.Attributes)
			}
		}
	}

	_, err = warehousePool.Exec(t.Context(), "CREATE SEQUENCE simulated_generation_attempts")
	if err != nil {
		t.Fatalf("expected warehouse write attempt sequence, got %v", err)
	}
	_, err = warehousePool.Exec(t.Context(), `CREATE FUNCTION simulated_generation_reject_insert() RETURNS trigger AS $$
		BEGIN
			PERFORM nextval('simulated_generation_attempts');
			RAISE EXCEPTION 'warehouse write blocked by test';
		END; $$ LANGUAGE plpgsql`)
	if err != nil {
		t.Fatalf("expected warehouse write gate function, got %v", err)
	}
	_, err = warehousePool.Exec(t.Context(), `CREATE TRIGGER simulated_generation_reject_insert
		BEFORE INSERT ON krenalis_simulated_system_records FOR EACH ROW
		EXECUTE FUNCTION simulated_generation_reject_insert()`)
	if err != nil {
		t.Fatalf("expected warehouse write gate, got %v", err)
	}
	var beforeWrite struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/simulated/systems", nil, map[string]any{
		"name": "before write", "userCount": 1,
		"countries": map[string]int{"IT": 1},
	}, &beforeWrite)
	deadline = time.Now().Add(20 * time.Second)
	var attempted bool
	for {
		err = warehousePool.QueryRow(t.Context(), "SELECT is_called FROM simulated_generation_attempts").Scan(&attempted)
		if err != nil {
			t.Fatalf("expected warehouse write attempt state, got %v", err)
		}
		if attempted || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !attempted {
		t.Fatal("expected a blocked warehouse write attempt, got none")
	}
	var beforeWriteCount int
	k.QueryRowTestDatabase(t.Context(), &beforeWriteCount,
		"SELECT generated_record_count FROM simulated_systems WHERE id = $1", beforeWrite.ID)
	err = warehousePool.QueryRow(t.Context(), "SELECT COUNT(*) FROM krenalis_simulated_system_records"+
		" WHERE simulated_system = $1", beforeWrite.ID).Scan(&total)
	if err != nil {
		t.Fatalf("expected blocked warehouse count, got %v", err)
	}
	if beforeWriteCount != 0 || total != 0 {
		t.Fatalf("expected no progress before a successful write, got count=%d rows=%d", beforeWriteCount, total)
	}
	_, err = warehousePool.Exec(t.Context(), "DROP TRIGGER simulated_generation_reject_insert ON krenalis_simulated_system_records")
	if err != nil {
		t.Fatalf("expected warehouse write gate removal, got %v", err)
	}
	_, err = warehousePool.Exec(t.Context(), "DROP FUNCTION simulated_generation_reject_insert()")
	if err != nil {
		t.Fatalf("expected warehouse write gate function removal, got %v", err)
	}
	_, err = warehousePool.Exec(t.Context(), "DROP SEQUENCE simulated_generation_attempts")
	if err != nil {
		t.Fatalf("expected warehouse write attempt sequence removal, got %v", err)
	}
	deadline = time.Now().Add(20 * time.Second)
	var beforeWriteStatus string
	for {
		k.QueryRowTestDatabase(t.Context(), &beforeWriteStatus,
			"SELECT status::text FROM simulated_systems WHERE id = $1", beforeWrite.ID)
		if beforeWriteStatus == "Available" || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if beforeWriteStatus != "Available" {
		t.Fatalf("expected account Available after warehouse write recovery, got %s", beforeWriteStatus)
	}

	k.ExecQueryTestDatabase(t.Context(), "CREATE TABLE simulated_generation_test_phase (phase text NOT NULL)")
	k.ExecQueryTestDatabase(t.Context(), "INSERT INTO simulated_generation_test_phase VALUES ('checkpoint')")
	k.ExecQueryTestDatabase(t.Context(), "CREATE SEQUENCE simulated_generation_checkpoint_attempts")
	k.ExecQueryTestDatabase(t.Context(), "CREATE SEQUENCE simulated_generation_available_attempts")
	k.ExecQueryTestDatabase(t.Context(), `CREATE FUNCTION simulated_generation_gate() RETURNS trigger AS $$
		DECLARE current_phase text;
		BEGIN
			SELECT phase INTO current_phase FROM simulated_generation_test_phase;
			IF current_phase = 'checkpoint' AND NEW.generated_record_count > OLD.generated_record_count THEN
				PERFORM nextval('simulated_generation_checkpoint_attempts');
				RAISE EXCEPTION 'checkpoint blocked by test';
			END IF;
			IF current_phase = 'available' AND NEW.status = 'Available' THEN
				PERFORM nextval('simulated_generation_available_attempts');
				RAISE EXCEPTION 'Available blocked by test';
			END IF;
			RETURN NEW;
		END; $$ LANGUAGE plpgsql`)
	k.ExecQueryTestDatabase(t.Context(), `CREATE TRIGGER simulated_generation_gate
		BEFORE UPDATE ON simulated_systems FOR EACH ROW EXECUTE FUNCTION simulated_generation_gate()`)
	var replay struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/simulated/systems", nil, map[string]any{
		"name": "replay", "userCount": 3,
		"duplicateRecordPercent": 50, "countries": map[string]int{"IT": 1},
	}, &replay)
	deadline = time.Now().Add(25 * time.Second)
	var replayRows, savedCount int
	for {
		err = warehousePool.QueryRow(t.Context(), "SELECT COUNT(*) FROM krenalis_simulated_system_records"+
			" WHERE simulated_system = $1", replay.ID).Scan(&replayRows)
		if err != nil {
			t.Fatalf("expected replay warehouse count, got %v", err)
		}
		k.QueryRowTestDatabase(t.Context(), &savedCount,
			"SELECT generated_record_count FROM simulated_systems WHERE id = $1", replay.ID)
		if replayRows == 3 || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if replayRows != 3 || savedCount != 0 {
		t.Fatalf("expected warehouse write before blocked checkpoint, got rows=%d saved=%d", replayRows, savedCount)
	}
	beforeReplayRecords := querySimulatedSystemRecords(t, warehousePool, replay.ID)
	deadline = time.Now().Add(20 * time.Second)
	attempted = false
	for {
		k.QueryRowTestDatabase(t.Context(), &attempted, "SELECT is_called FROM simulated_generation_checkpoint_attempts")
		if attempted || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !attempted {
		t.Fatal("expected a blocked checkpoint attempt, got none")
	}
	k.QueryRowTestDatabase(t.Context(), &savedCount,
		"SELECT generated_record_count FROM simulated_systems WHERE id = $1", replay.ID)
	if savedCount != 0 {
		t.Fatalf("expected blocked checkpoint to remain zero, got %d", savedCount)
	}
	k.RestartServer(t.Context())
	var restarted struct {
		Status string `json:"status"`
	}
	k.Call("GET", "/v1/simulated/systems/"+replay.ID, nil, nil, &restarted)
	k.QueryRowTestDatabase(t.Context(), &savedCount,
		"SELECT generated_record_count FROM simulated_systems WHERE id = $1", replay.ID)
	if restarted.Status != "Preparing" || savedCount != 0 {
		t.Fatalf("expected persisted Preparing checkpoint after restart, got %+v", restarted)
	}
	k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_generation_test_phase SET phase = 'available'")
	deadline = time.Now().Add(20 * time.Second)
	var replayStatus string
	for {
		k.QueryRowTestDatabase(t.Context(), &savedCount,
			"SELECT generated_record_count FROM simulated_systems WHERE id = $1", replay.ID)
		k.QueryRowTestDatabase(t.Context(), &replayStatus,
			"SELECT status::text FROM simulated_systems WHERE id = $1", replay.ID)
		if savedCount == 3 || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if savedCount != 3 || replayStatus != "Preparing" {
		t.Fatalf("expected final checkpoint before Available, got count=%d status=%s", savedCount, replayStatus)
	}
	deadline = time.Now().Add(20 * time.Second)
	attempted = false
	for {
		k.QueryRowTestDatabase(t.Context(), &attempted, "SELECT is_called FROM simulated_generation_available_attempts")
		if attempted || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !attempted {
		t.Fatal("expected a blocked Available transition, got none")
	}
	k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_generation_test_phase SET phase = 'none'")
	deadline = time.Now().Add(20 * time.Second)
	for {
		k.QueryRowTestDatabase(t.Context(), &replayStatus,
			"SELECT status::text FROM simulated_systems WHERE id = $1", replay.ID)
		if replayStatus == "Available" || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if replayStatus != "Available" {
		t.Fatalf("expected replayed account Available, got %s", replayStatus)
	}
	err = warehousePool.QueryRow(t.Context(), "SELECT COUNT(*) FROM krenalis_simulated_system_records"+
		" WHERE simulated_system = $1", replay.ID).Scan(&replayRows)
	if err != nil {
		t.Fatalf("expected replayed warehouse count, got %v", err)
	}
	if replayRows != 3 {
		t.Fatalf("expected exactly three rows after replay, got %d", replayRows)
	}
	afterReplayRecords := querySimulatedSystemRecords(t, warehousePool, replay.ID)
	if len(beforeReplayRecords) != 3 || len(afterReplayRecords) != 3 {
		t.Fatalf("expected three replay records, got before=%d after=%d", len(beforeReplayRecords), len(afterReplayRecords))
	}
	for id, before := range beforeReplayRecords {
		after, ok := afterReplayRecords[id]
		if !ok || after.Data != before.Data {
			t.Fatalf("expected replay record %s payload to remain unchanged, got before=%+v after=%+v", id, before, after)
		}
	}
	k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_generation_test_phase SET phase = 'checkpoint'")
	var failed struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/simulated/systems", nil, map[string]any{
		"name": "unavailable configuration", "userCount": 1,
		"countries": map[string]int{"IT": 1},
	}, &failed)
	deadline = time.Now().Add(20 * time.Second)
	var failedRows int
	for {
		err = warehousePool.QueryRow(t.Context(), "SELECT COUNT(*) FROM krenalis_simulated_system_records"+
			" WHERE simulated_system = $1", failed.ID).Scan(&failedRows)
		if err != nil {
			t.Fatalf("expected failed account warehouse count, got %v", err)
		}
		if failedRows == 1 || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if failedRows != 1 {
		t.Fatalf("expected one written record before configuration failure, got %d", failedRows)
	}
	var originalCheckpoint string
	k.QueryRowTestDatabase(t.Context(), &originalCheckpoint,
		"SELECT generation_checkpoint::text FROM simulated_systems WHERE id = $1", failed.ID)
	k.ExecQueryTestDatabase(t.Context(), `CREATE SEQUENCE simulated_failure_terminal_attempts;
		CREATE FUNCTION reject_simulated_failure_terminal() RETURNS trigger LANGUAGE plpgsql AS $body$
		BEGIN IF NEW.status = 'Failed' THEN PERFORM nextval('simulated_failure_terminal_attempts');
		RAISE EXCEPTION 'terminal metadata transition blocked'; END IF; RETURN NEW; END $body$;
		CREATE TRIGGER reject_simulated_failure_terminal BEFORE UPDATE ON simulated_systems
		FOR EACH ROW EXECUTE FUNCTION reject_simulated_failure_terminal()`)
	k.ExecQueryTestDatabase(t.Context(), `UPDATE simulated_systems
		SET generation_checkpoint = jsonb_set(generation_checkpoint, '{catalogSHA256}', '"missing"'::jsonb)
		WHERE id = $1`, failed.ID)
	k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_generation_test_phase SET phase = 'none'")
	deadline = time.Now().Add(20 * time.Second)
	for {
		var attempted bool
		k.QueryRowTestDatabase(t.Context(), &attempted, "SELECT is_called FROM simulated_failure_terminal_attempts")
		if attempted {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("expected a sealed generation with blocked terminal metadata, got no attempt")
		}
		time.Sleep(25 * time.Millisecond)
	}
	var pendingFailure bool
	k.QueryRowTestDatabase(t.Context(), &pendingFailure,
		"SELECT status = 'Preparing' AND generation_error <> '' FROM simulated_systems WHERE id = $1", failed.ID)
	if !pendingFailure {
		t.Fatal("expected durable failure intent before revocation, got none")
	}
	var sealedGeneration int64
	if err := warehousePool.QueryRow(t.Context(),
		"SELECT generation FROM krenalis_simulated_system_writers WHERE simulated_system = $1", failed.ID).Scan(&sealedGeneration); err != nil {
		t.Fatalf("expected sealed Preparing authority, got %v", err)
	}
	if sealedGeneration != 1 {
		t.Fatalf("expected generation zero revoked, got fence %d", sealedGeneration)
	}

	// Restore the configuration: a restarted worker must finish the persisted
	// failure rather than attempt generation under the irrevocably sealed token.
	k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_systems SET generation_checkpoint = $1::jsonb WHERE id = $2",
		originalCheckpoint, failed.ID)
	k.RestartServer(t.Context())
	k.ExecQueryTestDatabase(t.Context(), `DROP TRIGGER reject_simulated_failure_terminal ON simulated_systems;
		DROP FUNCTION reject_simulated_failure_terminal(); DROP SEQUENCE simulated_failure_terminal_attempts`)
	deadline = time.Now().Add(20 * time.Second)
	for {
		k.QueryRowTestDatabase(t.Context(), &replayStatus,
			"SELECT status::text FROM simulated_systems WHERE id = $1", failed.ID)
		if replayStatus == "Failed" || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if replayStatus != "Failed" {
		t.Fatalf("expected unavailable account Failed, got %s", replayStatus)
	}
	var failedDetail struct {
		Status string `json:"status"`
	}
	k.Call("GET", "/v1/simulated/systems/"+failed.ID, nil, nil, &failedDetail)
	var generationError string
	k.QueryRowTestDatabase(t.Context(), &generationError,
		"SELECT generation_error FROM simulated_systems WHERE id = $1", failed.ID)
	if failedDetail.Status != "Failed" || generationError == "" || len(generationError) > 256 {
		t.Fatalf("expected bounded failed status, got %+v", failedDetail)
	}
	var afterFailed struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/simulated/systems", nil, map[string]any{
		"name": "after failure", "userCount": 1,
		"countries": map[string]int{"IT": 1},
	}, &afterFailed)
	deadline = time.Now().Add(20 * time.Second)
	for {
		k.QueryRowTestDatabase(t.Context(), &replayStatus,
			"SELECT status::text FROM simulated_systems WHERE id = $1", afterFailed.ID)
		if replayStatus == "Available" || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if replayStatus != "Available" {
		t.Fatalf("expected next account Available after failure, got %s", replayStatus)
	}
	k.ExecQueryTestDatabase(t.Context(), "DROP TRIGGER simulated_generation_gate ON simulated_systems")
	k.ExecQueryTestDatabase(t.Context(), "DROP FUNCTION simulated_generation_gate()")
	k.ExecQueryTestDatabase(t.Context(), "DROP TABLE simulated_generation_test_phase")
	k.ExecQueryTestDatabase(t.Context(), "DROP SEQUENCE simulated_generation_checkpoint_attempts")
	k.ExecQueryTestDatabase(t.Context(), "DROP SEQUENCE simulated_generation_available_attempts")

	_, err = warehousePool.Exec(t.Context(), `UPDATE krenalis_simulated_system_records
		SET data = '{"terminal_marker":true}'::jsonb WHERE simulated_system = $1
		AND external_id = (SELECT MIN(external_id) FROM krenalis_simulated_system_records
			WHERE simulated_system = $1)`, created.ID)
	if err != nil {
		t.Fatalf("expected terminal record marker, got %v", err)
	}
	time.Sleep(6 * time.Second)
	var markerCount int
	err = warehousePool.QueryRow(t.Context(), `SELECT COUNT(*) FROM krenalis_simulated_system_records
		WHERE simulated_system = $1 AND data = '{"terminal_marker":true}'::jsonb`, created.ID).Scan(&markerCount)
	if err != nil {
		t.Fatalf("expected terminal marker query, got %v", err)
	}
	if markerCount != 1 {
		t.Fatalf("expected terminal record not to be regenerated, got %d markers", markerCount)
	}
	_, err = warehousePool.Exec(t.Context(), "DROP TABLE krenalis_simulated_system_records")
	if err != nil {
		t.Fatalf("expected isolated warehouse table removal, got %v", err)
	}
	k.Call("GET", "/v1/simulated/systems/"+created.ID, nil, nil, &detail)
	if detail.Status != "Available" {
		t.Fatalf("expected operational progress without warehouse count, got %+v", detail)
	}
}

func querySimulatedSystemRecords(t *testing.T, pool *pgxpool.Pool, account string) map[string]simulatedSystemRecordState {
	t.Helper()
	rows, err := pool.Query(t.Context(), "SELECT external_id, updated_at, data::text FROM krenalis_simulated_system_records"+
		" WHERE simulated_system = $1 ORDER BY external_id", account)
	if err != nil {
		t.Fatalf("expected simulated system timestamps query, got %v", err)
	}
	defer rows.Close()
	result := map[string]simulatedSystemRecordState{}
	for rows.Next() {
		var id string
		var updatedAt time.Time
		var data string
		if err := rows.Scan(&id, &updatedAt, &data); err != nil {
			t.Fatalf("expected simulated system record scan, got %v", err)
		}
		result[id] = simulatedSystemRecordState{UpdatedAt: updatedAt, Data: data}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("expected simulated system record rows, got %v", err)
	}
	return result
}

func simulatedSystemCatalogFixture(t *testing.T) string {
	t.Helper()
	root := filepath.Join("..", "tools", "fakedata", "testdata", "fakefacegen-v1-selected")
	directory := t.TempDir()
	var catalog, manifest map[string]any
	var specs []map[string]any
	for _, item := range []struct {
		name string
		out  any
	}{{"catalog.json", &catalog}, {"specs.json", &specs}, {"manifest.json", &manifest}} {
		data, err := os.ReadFile(filepath.Join(root, item.name))
		if err != nil {
			t.Fatalf("expected face catalog fixture %s, got %v", item.name, err)
		}
		err = json.Unmarshal(data, item.out)
		if err != nil {
			t.Fatalf("expected valid face catalog fixture %s, got %v", item.name, err)
		}
	}
	catalog["count"] = 2
	newSpecs := make([]map[string]any, 0, 2)
	newAssets := make([]map[string]any, 0, 2)
	for index, sourceID := range []string{"face-000013", "face-000037"} {
		id := fmt.Sprintf("face-%06d", index+1)
		for _, spec := range specs {
			if spec["id"] == sourceID {
				spec["id"] = id
				newSpecs = append(newSpecs, spec)
				break
			}
		}
		for _, item := range manifest["assets"].([]any) {
			asset := item.(map[string]any)
			if asset["id"] == sourceID {
				asset["id"] = id
				asset["spec_id"] = id
				asset["file"] = "masters/" + id + ".webp"
				newAssets = append(newAssets, asset)
				break
			}
		}
		for _, size := range []int{64, 128, 256, 512, 1024} {
			folder := filepath.Join("derived", strconv.Itoa(size))
			if size == 1024 {
				folder = "masters"
			}
			data, err := os.ReadFile(filepath.Join(root, folder, sourceID+".webp"))
			if err != nil {
				t.Fatalf("expected face fixture %s/%s, got %v", folder, sourceID, err)
			}
			path := filepath.Join(directory, folder)
			err = os.MkdirAll(path, 0o755)
			if err != nil {
				t.Fatalf("expected face fixture directory %s, got %v", path, err)
			}
			err = os.WriteFile(filepath.Join(path, id+".webp"), data, 0o644)
			if err != nil {
				t.Fatalf("expected face fixture asset %s, got %v", id, err)
			}
		}
	}
	manifest["assets"] = newAssets
	for _, item := range []struct {
		name  string
		value any
	}{{"catalog.json", catalog}, {"specs.json", newSpecs}, {"manifest.json", manifest}} {
		data, err := json.Marshal(item.value)
		if err != nil {
			t.Fatalf("expected face fixture JSON %s, got %v", item.name, err)
		}
		data, err = json.Canonicalize(data)
		if err != nil {
			t.Fatalf("expected canonical face fixture JSON %s, got %v", item.name, err)
		}
		err = os.WriteFile(filepath.Join(directory, item.name), data, 0o644)
		if err != nil {
			t.Fatalf("expected face fixture metadata %s, got %v", item.name, err)
		}
	}
	return directory
}
