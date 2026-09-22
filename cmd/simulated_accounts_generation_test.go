// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

func TestSimulatedAccountGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.SetSyntheticPhotosDir(simulatedAccountCatalogFixture(t))
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

	profileSchema := types.Object([]types.Property{{
		Name: "email", Type: types.String().AsEmail().WithMaxLength(254), ReadOptional: true,
	}})
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
		{"count above limit", func(body map[string]any) { body["userCount"] = 1_000_001 }, "userCount must be between"},
		{"unsupported country", func(body map[string]any) { body["countries"] = map[string]int{"FR": 40} }, "not supported"},
		{"empty countries", func(body map[string]any) { body["countries"] = map[string]int{} }, "countries must contain IT"},
		{"percent above limit", func(body map[string]any) { body["duplicateRecordPercent"] = 50.01 }, "duplicateRecordPercent must be between"},
		{"fractional precision", func(body map[string]any) { body["duplicateRecordPercent"] = 1.001 }, "duplicateRecordPercent is not valid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := map[string]any{}
			for key, value := range valid {
				body[key] = value
			}
			test.change(body)
			assertEnvironmentBadRequest(t, k.TryCall("POST", "/v1/simulated-accounts", nil, body, nil), test.want)
		})
	}
	var before int
	k.QueryRowTestDatabase(t.Context(), &before, "SELECT COUNT(*) FROM simulated_accounts WHERE workspace = $1", workspace.ID)
	if before != 0 {
		t.Fatalf("expected no metadata after invalid requests, got %d", before)
	}

	var created struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/simulated-accounts", nil, valid, &created)
	deadline := time.Now().Add(45 * time.Second)
	var detail struct {
		Status               string         `json:"status"`
		UserCount            int            `json:"userCount"`
		GeneratedRecordCount int            `json:"generatedRecordCount"`
		GenerationError      string         `json:"generationError"`
		Countries            map[string]int `json:"countries"`
	}
	for {
		k.Call("GET", "/v1/simulated-accounts/"+created.ID, nil, nil, &detail)
		if detail.Status == "Ready" || detail.Status == "Failed" || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if detail.Status != "Ready" || detail.UserCount != 513 || detail.GeneratedRecordCount != 513 ||
		detail.GenerationError != "" || detail.Countries["IT"] != 40 {
		t.Fatalf("expected completed account with 513 records, got %+v", detail)
	}
	var ordinary map[string]json.Value
	k.Call("GET", "/v1/simulated-accounts/"+created.ID, nil, nil, &ordinary)
	for _, field := range []string{"oracle", "personID", "generationCheckpoint"} {
		if _, ok := ordinary[field]; ok {
			t.Fatalf("expected no %s in ordinary API payload, got %s", field, ordinary[field])
		}
	}
	var progressEvents int
	k.QueryRowTestDatabase(t.Context(), &progressEvents,
		"SELECT COUNT(*) FROM notifications WHERE name = 'UpdateSimulatedAccountGeneration'")
	if progressEvents < 4 {
		t.Fatalf("expected three batch checkpoints and Ready notification, got %d events", progressEvents)
	}

	warehousePool, err := krenalistester.ConnectionPool(t.Context(), settings)
	if err != nil {
		t.Fatalf("expected generated warehouse pool, got %v", err)
	}
	defer warehousePool.Close()
	var total, unique int
	err = warehousePool.QueryRow(t.Context(), "SELECT COUNT(*), COUNT(DISTINCT external_id)"+
		" FROM krenalis_simulated_account_records WHERE simulated_account_id = $1", created.ID).Scan(&total, &unique)
	if err != nil {
		t.Fatalf("expected warehouse record counts, got %v", err)
	}
	if total != 513 || unique != 513 {
		t.Fatalf("expected 513 distinct warehouse records, got total=%d unique=%d", total, unique)
	}
	var data string
	err = warehousePool.QueryRow(t.Context(), "SELECT data::text FROM krenalis_simulated_account_records"+
		" WHERE simulated_account_id = $1 LIMIT 1", created.ID).Scan(&data)
	if err != nil {
		t.Fatalf("expected ordinary warehouse record, got %v", err)
	}
	if !strings.Contains(data, "source_record_id") || !strings.Contains(data, "photo_url") ||
		strings.Contains(data, "person_id") || strings.Contains(data, "oracle") || strings.Contains(data, "role") {
		t.Fatalf("expected ordinary customer document without Oracle or role, got %s", data)
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
		BEFORE INSERT ON krenalis_simulated_account_records FOR EACH ROW
		EXECUTE FUNCTION simulated_generation_reject_insert()`)
	if err != nil {
		t.Fatalf("expected warehouse write gate, got %v", err)
	}
	var beforeWrite struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/simulated-accounts", nil, map[string]any{
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
		"SELECT generated_record_count FROM simulated_accounts WHERE id = $1", beforeWrite.ID)
	err = warehousePool.QueryRow(t.Context(), "SELECT COUNT(*) FROM krenalis_simulated_account_records"+
		" WHERE simulated_account_id = $1", beforeWrite.ID).Scan(&total)
	if err != nil {
		t.Fatalf("expected blocked warehouse count, got %v", err)
	}
	if beforeWriteCount != 0 || total != 0 {
		t.Fatalf("expected no progress before a successful write, got count=%d rows=%d", beforeWriteCount, total)
	}
	_, err = warehousePool.Exec(t.Context(), "DROP TRIGGER simulated_generation_reject_insert ON krenalis_simulated_account_records")
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
			"SELECT status::text FROM simulated_accounts WHERE id = $1", beforeWrite.ID)
		if beforeWriteStatus == "Ready" || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if beforeWriteStatus != "Ready" {
		t.Fatalf("expected account Ready after warehouse write recovery, got %s", beforeWriteStatus)
	}

	k.ExecQueryTestDatabase(t.Context(), "CREATE TABLE simulated_generation_test_phase (phase text NOT NULL)")
	k.ExecQueryTestDatabase(t.Context(), "INSERT INTO simulated_generation_test_phase VALUES ('checkpoint')")
	k.ExecQueryTestDatabase(t.Context(), "CREATE SEQUENCE simulated_generation_checkpoint_attempts")
	k.ExecQueryTestDatabase(t.Context(), "CREATE SEQUENCE simulated_generation_ready_attempts")
	k.ExecQueryTestDatabase(t.Context(), `CREATE FUNCTION simulated_generation_gate() RETURNS trigger AS $$
		DECLARE current_phase text;
		BEGIN
			SELECT phase INTO current_phase FROM simulated_generation_test_phase;
			IF current_phase = 'checkpoint' AND NEW.generated_record_count > OLD.generated_record_count THEN
				PERFORM nextval('simulated_generation_checkpoint_attempts');
				RAISE EXCEPTION 'checkpoint blocked by test';
			END IF;
			IF current_phase = 'ready' AND NEW.status = 'Ready' THEN
				PERFORM nextval('simulated_generation_ready_attempts');
				RAISE EXCEPTION 'Ready blocked by test';
			END IF;
			RETURN NEW;
		END; $$ LANGUAGE plpgsql`)
	k.ExecQueryTestDatabase(t.Context(), `CREATE TRIGGER simulated_generation_gate
		BEFORE UPDATE ON simulated_accounts FOR EACH ROW EXECUTE FUNCTION simulated_generation_gate()`)
	var replay struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/simulated-accounts", nil, map[string]any{
		"name": "replay", "userCount": 3,
		"duplicateRecordPercent": 50, "countries": map[string]int{"IT": 1},
	}, &replay)
	deadline = time.Now().Add(25 * time.Second)
	var replayRows, savedCount int
	for {
		err = warehousePool.QueryRow(t.Context(), "SELECT COUNT(*) FROM krenalis_simulated_account_records"+
			" WHERE simulated_account_id = $1", replay.ID).Scan(&replayRows)
		if err != nil {
			t.Fatalf("expected replay warehouse count, got %v", err)
		}
		k.QueryRowTestDatabase(t.Context(), &savedCount,
			"SELECT generated_record_count FROM simulated_accounts WHERE id = $1", replay.ID)
		if replayRows == 3 || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if replayRows != 3 || savedCount != 0 {
		t.Fatalf("expected warehouse write before blocked checkpoint, got rows=%d saved=%d", replayRows, savedCount)
	}
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
		"SELECT generated_record_count FROM simulated_accounts WHERE id = $1", replay.ID)
	if savedCount != 0 {
		t.Fatalf("expected blocked checkpoint to remain zero, got %d", savedCount)
	}
	k.RestartServer(t.Context())
	var restarted struct {
		Status               string `json:"status"`
		GeneratedRecordCount int    `json:"generatedRecordCount"`
	}
	k.Call("GET", "/v1/simulated-accounts/"+replay.ID, nil, nil, &restarted)
	if restarted.Status != "Preparing" || restarted.GeneratedRecordCount != 0 {
		t.Fatalf("expected persisted Preparing checkpoint after restart, got %+v", restarted)
	}
	k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_generation_test_phase SET phase = 'ready'")
	deadline = time.Now().Add(20 * time.Second)
	var replayStatus string
	for {
		k.QueryRowTestDatabase(t.Context(), &savedCount,
			"SELECT generated_record_count FROM simulated_accounts WHERE id = $1", replay.ID)
		k.QueryRowTestDatabase(t.Context(), &replayStatus,
			"SELECT status::text FROM simulated_accounts WHERE id = $1", replay.ID)
		if savedCount == 3 || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if savedCount != 3 || replayStatus != "Preparing" {
		t.Fatalf("expected final checkpoint before Ready, got count=%d status=%s", savedCount, replayStatus)
	}
	deadline = time.Now().Add(20 * time.Second)
	attempted = false
	for {
		k.QueryRowTestDatabase(t.Context(), &attempted, "SELECT is_called FROM simulated_generation_ready_attempts")
		if attempted || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !attempted {
		t.Fatal("expected a blocked Ready transition, got none")
	}
	k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_generation_test_phase SET phase = 'none'")
	deadline = time.Now().Add(20 * time.Second)
	for {
		k.QueryRowTestDatabase(t.Context(), &replayStatus,
			"SELECT status::text FROM simulated_accounts WHERE id = $1", replay.ID)
		if replayStatus == "Ready" || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if replayStatus != "Ready" {
		t.Fatalf("expected replayed account Ready, got %s", replayStatus)
	}
	err = warehousePool.QueryRow(t.Context(), "SELECT COUNT(*) FROM krenalis_simulated_account_records"+
		" WHERE simulated_account_id = $1", replay.ID).Scan(&replayRows)
	if err != nil {
		t.Fatalf("expected replayed warehouse count, got %v", err)
	}
	if replayRows != 3 {
		t.Fatalf("expected exactly three rows after replay, got %d", replayRows)
	}
	k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_generation_test_phase SET phase = 'checkpoint'")
	var failed struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/simulated-accounts", nil, map[string]any{
		"name": "unavailable configuration", "userCount": 1,
		"countries": map[string]int{"IT": 1},
	}, &failed)
	deadline = time.Now().Add(20 * time.Second)
	var failedRows int
	for {
		err = warehousePool.QueryRow(t.Context(), "SELECT COUNT(*) FROM krenalis_simulated_account_records"+
			" WHERE simulated_account_id = $1", failed.ID).Scan(&failedRows)
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
	k.ExecQueryTestDatabase(t.Context(), `UPDATE simulated_accounts
		SET generation_checkpoint = jsonb_set(generation_checkpoint, '{catalogSHA256}', '"missing"'::jsonb)
		WHERE id = $1`, failed.ID)
	k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_generation_test_phase SET phase = 'none'")
	deadline = time.Now().Add(20 * time.Second)
	for {
		k.QueryRowTestDatabase(t.Context(), &replayStatus,
			"SELECT status::text FROM simulated_accounts WHERE id = $1", failed.ID)
		if replayStatus == "Failed" || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if replayStatus != "Failed" {
		t.Fatalf("expected unavailable account Failed, got %s", replayStatus)
	}
	var failedDetail struct {
		Status          string `json:"status"`
		GenerationError string `json:"generationError"`
	}
	k.Call("GET", "/v1/simulated-accounts/"+failed.ID, nil, nil, &failedDetail)
	if failedDetail.Status != "Failed" || failedDetail.GenerationError == "" || len(failedDetail.GenerationError) > 256 {
		t.Fatalf("expected bounded failed status, got %+v", failedDetail)
	}
	var afterFailed struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/simulated-accounts", nil, map[string]any{
		"name": "after failure", "userCount": 1,
		"countries": map[string]int{"IT": 1},
	}, &afterFailed)
	deadline = time.Now().Add(20 * time.Second)
	for {
		k.QueryRowTestDatabase(t.Context(), &replayStatus,
			"SELECT status::text FROM simulated_accounts WHERE id = $1", afterFailed.ID)
		if replayStatus == "Ready" || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if replayStatus != "Ready" {
		t.Fatalf("expected next account Ready after failure, got %s", replayStatus)
	}
	k.ExecQueryTestDatabase(t.Context(), "DROP TRIGGER simulated_generation_gate ON simulated_accounts")
	k.ExecQueryTestDatabase(t.Context(), "DROP FUNCTION simulated_generation_gate()")
	k.ExecQueryTestDatabase(t.Context(), "DROP TABLE simulated_generation_test_phase")
	k.ExecQueryTestDatabase(t.Context(), "DROP SEQUENCE simulated_generation_checkpoint_attempts")
	k.ExecQueryTestDatabase(t.Context(), "DROP SEQUENCE simulated_generation_ready_attempts")

	_, err = warehousePool.Exec(t.Context(), `UPDATE krenalis_simulated_account_records
		SET data = '{"terminal_marker":true}'::jsonb WHERE simulated_account_id = $1
		AND external_id = (SELECT MIN(external_id) FROM krenalis_simulated_account_records
			WHERE simulated_account_id = $1)`, created.ID)
	if err != nil {
		t.Fatalf("expected terminal record marker, got %v", err)
	}
	time.Sleep(6 * time.Second)
	var markerCount int
	err = warehousePool.QueryRow(t.Context(), `SELECT COUNT(*) FROM krenalis_simulated_account_records
		WHERE simulated_account_id = $1 AND data = '{"terminal_marker":true}'::jsonb`, created.ID).Scan(&markerCount)
	if err != nil {
		t.Fatalf("expected terminal marker query, got %v", err)
	}
	if markerCount != 1 {
		t.Fatalf("expected terminal record not to be regenerated, got %d markers", markerCount)
	}
	_, err = warehousePool.Exec(t.Context(), "DROP TABLE krenalis_simulated_account_records")
	if err != nil {
		t.Fatalf("expected isolated warehouse table removal, got %v", err)
	}
	k.Call("GET", "/v1/simulated-accounts/"+created.ID, nil, nil, &detail)
	if detail.Status != "Ready" || detail.GeneratedRecordCount != 513 {
		t.Fatalf("expected operational progress without warehouse count, got %+v", detail)
	}
}

func simulatedAccountCatalogFixture(t *testing.T) string {
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
