// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd_test

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	stdjson "encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/test/syntheticworldtest"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestSimulatedSystemSocioeconomicDestinationRoundTrip preserves generated facts
// through an admitted Destination partial update and a subsequent Source read.
func TestSimulatedSystemSocioeconomicDestinationRoundTrip(t *testing.T) {
	testSimulatedSystemDestinationRoundTrip(t, "generation-v5")
}

// TestSimulatedSystemHistoricalAvailable exercises persisted legacy records
// through Source, Destination, rename and deletion with the origin unavailable.
func TestSimulatedSystemHistoricalAvailable(t *testing.T) {
	for _, policy := range []string{"generation-v3", "generation-v4"} {
		t.Run(policy, func(t *testing.T) {
			testSimulatedSystemDestinationRoundTrip(t, policy)
		})
	}
}

func testSimulatedSystemDestinationRoundTrip(t *testing.T, storedPolicy string) {

	t.Helper()

	if testing.Short() {
		t.Skip()
	}
	assets := syntheticworldtest.Assets(t)
	origin := httptest.NewServer(syntheticworldtest.Handler(t, assets, ""))
	defer origin.Close()
	k := krenalistester.NewKrenalisInstance(t)
	k.SetExternalAssetsURLs([]string{origin.URL})
	k.Start()
	defer k.Stop()

	settings := &krenalistester.DBSettings{}
	err := json.Unmarshal(krenalistester.PostgresWarehouseSettings(), settings)
	if err != nil {
		t.Fatalf("expected PostgreSQL settings, got %v", err)
	}
	admin, err := krenalistester.ConnectionPool(t.Context(), settings)
	if err != nil {
		t.Fatalf("expected warehouse admin pool, got %v", err)
	}
	defer admin.Close()
	_, err = admin.Exec(t.Context(), "CREATE DATABASE test_simulated_socioeconomic_round_trip")
	if err != nil {
		t.Fatalf("expected isolated warehouse, got %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := admin.Exec(ctx, "DROP DATABASE test_simulated_socioeconomic_round_trip WITH (FORCE)")
		if err != nil {
			t.Errorf("expected isolated warehouse cleanup, got %v", err)
		}
	}()
	settings.Database = "test_simulated_socioeconomic_round_trip"
	profileSchema := types.Object([]types.Property{
		{Name: "site_url", Type: types.String().AsURL(), ReadOptional: true},
	})
	var workspace struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/workspaces", http.Header{"Krenalis-Workspace": nil}, map[string]any{
		"name": "Generated socioeconomic round trip", "environment": "development", "profileSchema": krenalistester.ProfileSchema{Schema: profileSchema},
		"warehouse": map[string]any{"platform": "PostgreSQL", "mode": "Normal", "settings": settings},
	}, &workspace)
	k.SetWorkspaceID(workspace.ID)
	pool, err := krenalistester.ConnectionPool(t.Context(), settings)
	if err != nil {
		t.Fatalf("expected warehouse pool, got %v", err)
	}
	defer pool.Close()
	var system struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/simulated/systems", nil, map[string]any{
		"name": "Generated customer", "userCount": 1, "countries": map[string]int{"IT": 1},
	}, &system)
	waitForSimulatedSystemStatus(t, k, system.ID, "Available")
	var policy string
	k.QueryRowTestDatabase(t.Context(), &policy,
		"SELECT generation_policy_version FROM simulated_systems WHERE id = $1", system.ID)
	if policy != "generation-v5" {
		t.Fatalf("expected generation-v5, got %s", policy)
	}
	rows := querySimulatedSystemRecords(t, pool, system.ID)
	if len(rows) != 1 {
		t.Fatalf("expected one generated record, got %d", len(rows))
	}

	// Independently reconstruct the workspace World and the first generated Person.
	sourceData := syntheticworldtest.Load(t, assets)
	model, err := sourceData.Model()
	if err != nil {
		t.Fatalf("expected prepared socioeconomic model, got %v", err)
	}
	worldKey := sha256.Sum256([]byte("krenalis/simulated-world/v1/" + workspace.ID))
	namespace, err := fakedata.NewIdentityNamespace("w"+hex.EncodeToString(worldKey[:10]), 1)
	if err != nil {
		t.Fatalf("expected workspace namespace, got %v", err)
	}
	base, err := fakedata.NewItalianWorldV1(fakedata.ItalianWorldConfig{
		IdentityNamespace: namespace, WorldSeed: binary.BigEndian.Uint64(worldKey[10:18]),
		ReferenceDate: "2026-01-01",
	}, sourceData)
	if err != nil {
		t.Fatalf("expected workspace World, got %v", err)
	}
	world, err := fakedata.NewSocioeconomicWorld(base, model)
	if err != nil {
		t.Fatalf("expected socioeconomic World, got %v", err)
	}
	systemKey := sha256.Sum256([]byte("krenalis/simulated-system/v1/" + system.ID))
	index := fakedata.PersonIndex(binary.BigEndian.Uint64(systemKey[:8])%uint64(fakedata.Layer1MaxPersonIndex) + 1)
	facts, err := world.Facts(index)
	if err != nil {
		t.Fatalf("expected canonical socioeconomic facts, got %v", err)
	}
	want := map[string]any{
		"education_level": string(facts.Education), "primary_activity": string(facts.PrimaryActivity),
		"annual_income_eur": stdjson.Number(strconv.FormatUint(uint64(facts.Income), 10)),
	}
	switch facts.PrimaryActivity {
	case fakedata.PrimaryActivityEmployee:
		company, err := world.Company(facts.Employee.Employer)
		if err != nil {
			t.Fatalf("expected canonical employer, got %v", err)
		}
		want["job_family"] = string(facts.Employee.JobFamily)
		want["industry"] = string(company.Industry)
		want["job_level"] = string(facts.Employee.JobLevel)
	case fakedata.PrimaryActivitySelfEmployed:
		want["job_family"] = string(facts.SelfEmployed.JobFamily)
		want["industry"] = string(facts.SelfEmployed.Industry)
	case fakedata.PrimaryActivityUnemployed, fakedata.PrimaryActivityStudent,
		fakedata.PrimaryActivityRetired, fakedata.PrimaryActivityInactive:
		// Non-working activities omit all three occupational properties.
	default:
		t.Fatalf("expected a supported activity, got %s", facts.PrimaryActivity)
	}

	if storedPolicy != "generation-v5" {

		if storedPolicy == "generation-v3" {
			_, err = pool.Exec(t.Context(), `UPDATE krenalis_simulated_system_records SET data = data -
				ARRAY['education_level','primary_activity','annual_income_eur','job_family','industry','job_level']
				WHERE simulated_system = $1`, system.ID)
			if err != nil {
				t.Fatalf("expected historical record without L2 properties, got %v", err)
			}
			want = map[string]any{}
		}
		k.ExecQueryTestDatabase(t.Context(), `UPDATE simulated_systems SET generation_policy_version = $1,
			generation_checkpoint = '{"person":1,"representation":0,"personSpecVersion":"person-v1"}'::jsonb
			WHERE id = $2`, storedPolicy, system.ID)
		rows = querySimulatedSystemRecords(t, pool, system.ID)
		origin.Close()
		k.RestartServer(t.Context())
		waitForSimulatedSystemStatus(t, k, system.ID, "Available")
		if !reflect.DeepEqual(rows, querySimulatedSystemRecords(t, pool, system.ID)) {
			t.Fatal("expected historical Available records unchanged after restart, got rewritten rows")
		}

	}

	source := k.CreateConnection(krenalistester.ConnectionToCreate{
		Name: "Generated Source", Role: krenalistester.Source, Connector: "simulated-application",
		Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": system.ID}),
	})
	var schemas struct {
		In  types.Type `json:"in"`
		Out types.Type `json:"out"`
	}
	k.Call("GET", "/v1/connections/"+source+"/pipelines/schemas/User", nil, nil, &schemas)
	schemaJSON, err := schemas.In.MarshalJSON()
	if err != nil {
		t.Fatalf("expected full Source schema JSON, got %v", err)
	}
	readSource := func() map[string]any {

		t.Helper()
		var response struct {
			Users []map[string]any `json:"users"`
		}
		// The HTTP harness uses json.Number, retaining exact integer tokens, including zero.
		k.Call("GET", "/v1/connections/"+source+"/users?schema="+url.QueryEscape(string(schemaJSON)), nil, nil, &response)
		if len(response.Users) != 1 {
			t.Fatalf("expected one Source record, got %d", len(response.Users))
		}
		for _, name := range []string{"education_level", "primary_activity", "annual_income_eur",
			"job_family", "industry", "job_level"} {
			got, present := response.Users[0][name]
			expected, applicable := want[name]
			if present != applicable || got != expected {
				t.Fatalf("expected canonical %s=%v present=%t, got %v present=%t", name, expected, applicable, got, present)
			}
		}

		return response.Users[0]
	}
	before := readSource()
	id, ok := before["customer_id"].(string)
	if !ok || id == "" {
		t.Fatalf("expected observed customer ID, got %v", before["customer_id"])
	}
	if _, ok := rows[id]; !ok {
		t.Fatalf("expected Source identity %q in generated warehouse records, got %v", id, rows)
	}
	input := types.Prune(schemas.In, func(path string) bool { return path == "site_url" })
	importPipeline := k.CreatePipeline(source, "User", krenalistester.PipelineToSet{
		Name: "Import generated identity", Enabled: true, InSchema: input, OutSchema: profileSchema,
		Transformation: &krenalistester.Transformation{Mapping: map[string]string{"site_url": "site_url"}},
	})
	k.WaitForRunsCompletion(k.StartPipelineRun(importPipeline))
	// Import completion can precede the automatic identity resolution.
	profileDeadline := time.Now().Add(20 * time.Second)
	for {
		profiles, total := k.Profiles([]string{"site_url"}, "", false, 0, 10)
		if total == 1 && len(profiles) == 1 && profiles[0].Attributes["site_url"] == before["site_url"] {
			break
		}
		if time.Now().After(profileDeadline) {
			t.Fatalf("expected one imported generated identity, got total=%d profiles=%v", total, profiles)
		}
		time.Sleep(50 * time.Millisecond)
	}

	mode := krenalistester.Server
	destination := k.CreateConnection(krenalistester.ConnectionToCreate{
		Name: "Generated Destination", Role: krenalistester.Destination, Connector: "simulated-application",
		SendingMode: &mode, Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": system.ID}),
	})
	destinationSchemas := simulatedApplicationPipelineSchemas(t, k, destination)
	output := types.Prune(destinationSchemas.Out, func(path string) bool {
		return path == "site_url" || path == "first_name"
	})
	updatePipeline := k.CreatePipeline(destination, "User", krenalistester.PipelineToSet{
		Name: "Update generated customer", Enabled: true, InSchema: profileSchema, OutSchema: output,
		ExportMode: krenalistester.UpdateOnly, Matching: krenalistester.Matching{In: "site_url", Out: "site_url"},
		Transformation: &krenalistester.Transformation{Mapping: map[string]string{
			"first_name": "site_url ' Updated'",
		}},
	})
	origin.Close()
	updateRun := k.StartPipelineRun(updatePipeline)
	k.WaitForRunsCompletion(updateRun)
	var claimed string
	k.QueryRowTestDatabase(t.Context(), &claimed, "SELECT simulated_system FROM pipelines_runs WHERE id = $1", updateRun)
	if claimed != system.ID {
		t.Fatalf("expected admitted Destination account %s, got %s", system.ID, claimed)
	}
	after := readSource()
	wantUpdated := maps.Clone(before)
	wantUpdated["first_name"] = before["site_url"].(string) + " Updated"
	if after["first_name"] == before["first_name"] || !reflect.DeepEqual(after, wantUpdated) {
		t.Fatalf("expected only first_name changed to %v, got before=%v after=%v", wantUpdated["first_name"], before, after)
	}
	afterRows := querySimulatedSystemRecords(t, pool, system.ID)
	if _, ok := afterRows[id]; !ok || len(afterRows) != 1 {
		t.Fatalf("expected unchanged persisted observed identity %s, got %v", id, afterRows)
	}
	t.Logf("verified %s round trip: PersonIndex=%d activity=%s income=%d",
		storedPolicy, index, facts.PrimaryActivity, facts.Income)

	if storedPolicy != "generation-v5" {

		k.Call("PUT", "/v1/simulated/systems/"+system.ID, nil, map[string]any{"name": "Historical renamed"}, nil)
		var name string
		k.QueryRowTestDatabase(t.Context(), &name, "SELECT name FROM simulated_systems WHERE id = $1", system.ID)
		if name != "Historical renamed" {
			t.Fatalf("expected renamed historical system, got %q", name)
		}
		k.DeleteConnection(source)
		k.DeleteConnection(destination)
		k.ExecQueryTestDatabase(t.Context(), `CREATE FUNCTION reject_legacy_delete() RETURNS trigger
			LANGUAGE plpgsql AS $body$ BEGIN RAISE EXCEPTION 'deletion blocked by test'; END $body$;
			CREATE TRIGGER reject_legacy_delete BEFORE DELETE ON simulated_systems
			FOR EACH ROW EXECUTE FUNCTION reject_legacy_delete()`)
		k.Call("DELETE", "/v1/simulated/systems/"+system.ID, nil, nil, nil)
		waitForSimulatedSystemStatus(t, k, system.ID, "Deleting")
		deadline := time.Now().Add(20 * time.Second)
		for len(querySimulatedSystemRecords(t, pool, system.ID)) != 0 {
			if time.Now().After(deadline) {
				t.Fatal("expected warehouse cleanup before blocked metadata deletion, got timeout")
			}
			time.Sleep(50 * time.Millisecond)
		}
		k.RestartServer(t.Context())
		k.ExecQueryTestDatabase(t.Context(), `DROP TRIGGER reject_legacy_delete ON simulated_systems;
			DROP FUNCTION reject_legacy_delete()`)
		deadline = time.Now().Add(20 * time.Second)
		for {
			var exists bool
			k.QueryRowTestDatabase(t.Context(), &exists,
				"SELECT EXISTS (SELECT FROM simulated_systems WHERE id = $1)", system.ID)
			if !exists {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("expected resumed legacy deletion without assets, got timeout")
			}
			time.Sleep(50 * time.Millisecond)
		}

	}

}

// TestSimulatedSystemSocioeconomicLifecycle exercises cmd/Core ownership,
// cancellation, acquisition/construction failures and historical preparation.
func TestSimulatedSystemSocioeconomicLifecycle(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	assets := syntheticworldtest.Assets(t)
	var zeroAssets map[string][]byte
	var mode, calls, requests atomic.Int32
	started, canceled := make(chan struct{}, 1), make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		requests.Add(1)
		current := mode.Load()
		if r.URL.Path != "/socioeconomic.json" {
			documents := assets
			if current == 3 {
				documents = zeroAssets
			}
			syntheticworldtest.Handler(t, documents, "").ServeHTTP(w, r)
			return
		}
		calls.Add(1)
		switch current {
		case 0:
			started <- struct{}{}
			<-r.Context().Done()
			canceled <- struct{}{}
		case 1:
			_, _ = w.Write(assets["socioeconomic.json"])
		case 2:
			_, _ = w.Write([]byte(`{"schema":"invalid"}`))
		case 3:
			_, _ = w.Write(zeroAssets["socioeconomic.json"])
		}

	}))
	defer server.Close()
	k := krenalistester.NewKrenalisInstance(t)
	k.SetExternalAssetsURLs([]string{server.URL})
	k.Start()
	defer k.Stop()
	if requests.Load() != 0 {
		t.Fatalf("expected no acquisition at startup, got %d requests", requests.Load())
	}

	settings := &krenalistester.DBSettings{}
	err := json.Unmarshal(krenalistester.PostgresWarehouseSettings(), settings)
	if err != nil {
		t.Fatalf("expected PostgreSQL settings, got %v", err)
	}
	admin, err := krenalistester.ConnectionPool(t.Context(), settings)
	if err != nil {
		t.Fatalf("expected warehouse admin pool, got %v", err)
	}
	defer admin.Close()
	_, err = admin.Exec(t.Context(), "CREATE DATABASE test_simulated_socioeconomic")
	if err != nil {
		t.Fatalf("expected isolated warehouse, got %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := admin.Exec(ctx, "DROP DATABASE test_simulated_socioeconomic WITH (FORCE)")
		if err != nil {
			t.Errorf("expected isolated warehouse cleanup, got %v", err)
		}
	}()
	settings.Database = "test_simulated_socioeconomic"
	var workspace struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/workspaces", http.Header{"Krenalis-Workspace": nil}, map[string]any{
		"name": "socioeconomic", "environment": "development",
		"profileSchema": krenalistester.ProfileSchema{Schema: types.Object([]types.Property{
			{Name: "email", Type: types.String().AsEmail(), ReadOptional: true},
		})},
		"warehouse": map[string]any{"platform": "PostgreSQL", "mode": "Normal", "settings": settings},
	}, &workspace)
	k.SetWorkspaceID(workspace.ID)
	pool, err := krenalistester.ConnectionPool(t.Context(), settings)
	if err != nil {
		t.Fatalf("expected isolated warehouse pool, got %v", err)
	}
	defer pool.Close()
	create := func(name string, count int) string {
		t.Helper()
		var system struct {
			ID string `json:"id"`
		}
		k.Call("POST", "/v1/simulated/systems", nil, map[string]any{
			"name": name, "userCount": count, "countries": map[string]int{"IT": 1},
		}, &system)
		return system.ID
	}
	assertNoProgress := func(id string) {

		t.Helper()
		var count int
		k.QueryRowTestDatabase(t.Context(), &count,
			"SELECT generated_record_count FROM simulated_systems WHERE id = $1", id)
		var hasCheckpoint bool
		k.QueryRowTestDatabase(t.Context(), &hasCheckpoint,
			"SELECT generation_checkpoint IS NOT NULL FROM simulated_systems WHERE id = $1", id)
		rows := querySimulatedSystemRecords(t, pool, id)
		if count != 0 || len(rows) != 0 || hasCheckpoint {
			t.Fatalf("expected no records, progress or checkpoint, got count=%d rows=%d checkpoint=%t",
				count, len(rows), hasCheckpoint)
		}

	}

	empty := create("empty", 0)
	waitForSimulatedSystemStatus(t, k, empty, "Available")
	var policy string
	k.QueryRowTestDatabase(t.Context(), &policy,
		"SELECT generation_policy_version FROM simulated_systems WHERE id = $1", empty)
	if policy != "empty-v1" || requests.Load() != 0 {
		t.Fatalf("expected empty-v1 without acquisition, got %s and %d requests", policy, requests.Load())
	}
	interrupted := create("interrupted acquisition", 3)
	select {
	case <-started:
	case <-time.After(20 * time.Second):
		t.Fatal("expected worker model request, got timeout")
	}
	assertNoProgress(interrupted)
	mode.Store(1)
	k.RestartServer(t.Context())
	select {
	case <-canceled:
	case <-time.After(20 * time.Second):
		t.Fatal("expected shutdown to cancel model acquisition, got timeout")
	}
	waitForSimulatedSystemStatus(t, k, interrupted, "Available")
	waitForSimulatedSystemStatus(t, k, create("reused model", 2), "Available")
	if calls.Load() != 2 {
		t.Fatalf("expected canceled request and one shared success after restart, got %d requests", calls.Load())
	}

	mode.Store(2)
	k.RestartServer(t.Context())
	if calls.Load() != 2 {
		t.Fatalf("expected Available systems and startup without acquisition, got %d requests", calls.Load())
	}
	source := k.CreateConnection(krenalistester.ConnectionToCreate{
		Name: "Available system source", Role: krenalistester.Source, Connector: "simulated-application",
		Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": interrupted}),
	})
	schemas := k.PipelineSchemas(source, core.TargetUser, "")
	schemaJSON, err := json.Marshal(schemas["in"])
	if err != nil {
		t.Fatalf("expected Source schema JSON, got %v", err)
	}
	var response struct {
		Users []map[string]any `json:"users"`
	}
	k.Call("GET", "/v1/connections/"+source+"/users?schema="+url.QueryEscape(string(schemaJSON)), nil, nil, &response)
	if len(response.Users) != 3 || calls.Load() != 2 {
		t.Fatalf("expected ordinary Source reads without model acquisition, got users=%d requests=%d",
			len(response.Users), calls.Load())
	}
	for _, user := range response.Users {
		for _, name := range []string{"education_level", "primary_activity", "annual_income_eur"} {
			if _, ok := user[name]; !ok {
				t.Fatalf("expected persisted Source property %s, got %v", name, user)
			}
		}
	}
	failed := create("invalid model", 2)
	waitForSimulatedSystemStatus(t, k, failed, "Failed")
	assertNoProgress(failed)
	mode.Store(1)
	waitForSimulatedSystemStatus(t, k, create("retry in a new job", 1), "Available")
	if calls.Load() != 4 {
		t.Fatalf("expected failed acquisition followed by a reusable success, got %d requests", calls.Load())
	}
	waitForSimulatedSystemStatus(t, k, failed, "Failed")
	assertNoProgress(failed)

	// Retired preparations fail without acquiring anything, including on a
	// restart after persisting failure intent and sealing the warehouse.
	for _, retired := range []string{"generation-v3", "generation-v4"} {

		legacy := create("historical preparing", 0)
		const historical = `{"person":1,"representation":0,"catalogSHA256":"historical","worldModelVersion":"world-v1","marketDataVersion":"it-2026a","personSpecVersion":"person-v1","sourceVersion":"source-v1","namespaceVersion":1,"referenceDate":"2026-01-01","photoBaseURL":"https://unavailable.test/"}`
		_, err = pool.Exec(t.Context(), `INSERT INTO krenalis_simulated_system_records
			(simulated_system, external_id, updated_at, data)
			SELECT $1, external_id, updated_at, data FROM krenalis_simulated_system_records
			WHERE simulated_system = $2 LIMIT 1`, legacy, interrupted)
		if err != nil {
			t.Fatalf("expected historical partial record, got %v", err)
		}
		before := querySimulatedSystemRecords(t, pool, legacy)
		k.ExecQueryTestDatabase(t.Context(), `CREATE SEQUENCE legacy_failure_attempts;
			CREATE FUNCTION reject_legacy_failure() RETURNS trigger LANGUAGE plpgsql AS $body$
			BEGIN IF NEW.status = 'Failed' THEN PERFORM nextval('legacy_failure_attempts');
			RAISE EXCEPTION 'failure transition blocked by test'; END IF; RETURN NEW; END $body$;
			CREATE TRIGGER reject_legacy_failure BEFORE UPDATE ON simulated_systems
			FOR EACH ROW EXECUTE FUNCTION reject_legacy_failure()`)
		k.ExecQueryTestDatabase(t.Context(), `UPDATE simulated_systems SET status = 'Preparing', user_count = 3,
			generated_record_count = 1, generation_policy_version = $1, generation_checkpoint = $2::jsonb WHERE id = $3`,
			retired, historical, legacy)
		mode.Store(2)
		beforeRequests := requests.Load()
		k.RestartServer(t.Context())
		deadline := time.Now().Add(20 * time.Second)
		for {
			var attempted bool
			k.QueryRowTestDatabase(t.Context(), &attempted, "SELECT is_called FROM legacy_failure_attempts")
			if attempted {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("expected attempted legacy failure transition, got timeout")
			}
			time.Sleep(50 * time.Millisecond)
		}
		var intent string
		k.QueryRowTestDatabase(t.Context(), &intent,
			"SELECT generation_error FROM simulated_systems WHERE id = $1", legacy)
		wantError := "legacy generation policy " + retired + " is no longer supported"
		if intent != wantError {
			t.Fatalf("expected durable failure intent %q, got %q", wantError, intent)
		}
		var sealed int64
		err = pool.QueryRow(t.Context(), "SELECT generation FROM krenalis_simulated_system_writers WHERE simulated_system = $1", legacy).Scan(&sealed)
		if err != nil {
			t.Fatalf("expected legacy writer fence, got %v", err)
		}
		if sealed != 1 {
			t.Fatalf("expected sealed generation 1, got %d", sealed)
		}
		k.RestartServer(t.Context())
		k.ExecQueryTestDatabase(t.Context(), `DROP TRIGGER reject_legacy_failure ON simulated_systems;
			DROP FUNCTION reject_legacy_failure(); DROP SEQUENCE legacy_failure_attempts`)
		waitForSimulatedSystemStatus(t, k, legacy, "Failed")
		var preserved bool
		k.QueryRowTestDatabase(t.Context(), &preserved, `SELECT generation_policy_version = $1
			AND generation_checkpoint = $2::jsonb AND generated_record_count = 1 AND generation_error = $3
			FROM simulated_systems WHERE id = $4`, retired, historical, wantError, legacy)
		if !preserved || requests.Load() != beforeRequests || !reflect.DeepEqual(before, querySimulatedSystemRecords(t, pool, legacy)) {
			t.Fatal("expected untouched historical policy, checkpoint and records without acquisition, got changed state")
		}
		k.Call("DELETE", "/v1/simulated/systems/"+legacy, nil, nil, nil)
		deadline = time.Now().Add(20 * time.Second)
		for {
			var exists bool
			k.QueryRowTestDatabase(t.Context(), &exists, "SELECT EXISTS (SELECT FROM simulated_systems WHERE id = $1)", legacy)
			if !exists {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("expected deleted Failed legacy system, got timeout")
			}
			time.Sleep(50 * time.Millisecond)
		}
		if len(querySimulatedSystemRecords(t, pool, legacy)) != 0 || requests.Load() != beforeRequests {
			t.Fatal("expected cleanup without acquisition, got records or new requests")
		}

	}

	zeroAssets = maps.Clone(assets)
	zeroModel := simulatedSystemConstructionFailureModel(t, workspace.ID, assets)
	syntheticworldtest.SetModel(t, zeroAssets, zeroModel)
	mode.Store(3)
	construction := create("invalid realized employer universe", 1)
	waitForSimulatedSystemStatus(t, k, construction, "Failed")
	assertNoProgress(construction)
	var message string
	k.QueryRowTestDatabase(t.Context(), &message,
		"SELECT generation_error FROM simulated_systems WHERE id = $1", construction)
	if message != "generation configuration is unavailable or changed" {
		t.Fatalf("expected construction failure after successful acquisition, got %s", message)
	}
	var sealed int64
	if err := pool.QueryRow(t.Context(),
		"SELECT generation FROM krenalis_simulated_system_writers WHERE simulated_system = $1",
		construction).Scan(&sealed); err != nil {
		t.Fatalf("expected construction failure writer fence, got %v", err)
	}
	if sealed != 1 {
		t.Fatalf("expected failed generation sealed at 1, got %d", sealed)
	}
	mode.Store(1)
	retained := create("retained acquired model", 1)
	waitForSimulatedSystemStatus(t, k, retained, "Failed")
	assertNoProgress(retained)
	if calls.Load() != 5 {
		t.Fatalf("expected successful acquisition retained despite construction failure, got %d requests", calls.Load())
	}
	k.RestartServer(t.Context())
	waitForSimulatedSystemStatus(t, k, create("fresh process", 1), "Available")
	if calls.Load() != 6 {
		t.Fatalf("expected fresh-process acquisition for the new job, got %d requests", calls.Load())
	}

}

// simulatedSystemConstructionFailureModel verifies an acquisition-valid model
// whose complete Company universe has no realized employer for this workspace.
func simulatedSystemConstructionFailureModel(t *testing.T, workspaceID string, assets map[string][]byte) []byte {

	t.Helper()
	key := sha256.Sum256([]byte("krenalis/simulated-world/v1/" + workspaceID))
	namespace, err := fakedata.NewIdentityNamespace("w"+hex.EncodeToString(key[:10]), 1)
	if err != nil {
		t.Fatalf("expected workspace namespace, got %v", err)
	}
	var document map[string]json.Value
	err = json.Unmarshal(assets["socioeconomic.json"], &document)
	if err != nil {
		t.Fatalf("expected artificial international model, got %v", err)
	}
	var models map[string]json.Value
	err = json.Unmarshal(document["models"], &models)
	if err != nil {
		t.Fatalf("expected artificial laws, got %v", err)
	}
	for positive := 1; positive <= 8; positive++ {

		law, err := json.Marshal(map[string][]uint32{"values": {0, uint32(positive)}, "weights": {1<<32 - 1, 1}})
		if err != nil {
			t.Fatalf("expected count law JSON, got %v", err)
		}
		laws := strings.TrimSuffix(strings.Repeat(string(law)+",", 14), ",")
		models["employeeCount"] = json.Value(`{"axes":["industry"],"laws":[` + laws + `]}`)
		document["models"], err = json.Marshal(models)
		if err != nil {
			t.Fatalf("expected candidate laws, got %v", err)
		}
		candidate, err := json.Marshal(document)
		if err != nil {
			t.Fatalf("expected candidate model, got %v", err)
		}
		combination := maps.Clone(assets)
		syntheticworldtest.SetModel(t, combination, candidate)
		data := syntheticworldtest.Load(t, combination)
		base, err := fakedata.NewItalianWorldV1(fakedata.ItalianWorldConfig{
			IdentityNamespace: namespace, WorldSeed: binary.BigEndian.Uint64(key[10:18]), ReferenceDate: "2026-01-01",
		}, data)
		if err != nil {
			t.Fatalf("expected prepared Italian World, got %v", err)
		}
		model, err := data.Model()
		if err != nil {
			t.Fatalf("expected prepared model, got %v", err)
		}
		world, err := fakedata.NewSocioeconomicWorld(base, model)
		if err != nil {
			if !errors.Is(err, fakedata.ErrInvalidSocioeconomicWorld) || world != nil {
				t.Fatalf("expected no partial World and invalid-world error, got %v and %v", world, err)
			}
			return candidate
		}

	}
	t.Fatal("expected a verified construction-failure fixture, got only successful Worlds")

	return nil
}

func waitForSimulatedSystemStatus(t *testing.T, k *krenalistester.Krenalis, id, expected string) {

	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {

		var status string
		k.QueryRowTestDatabase(t.Context(), &status, "SELECT status::text FROM simulated_systems WHERE id = $1", id)
		if status == expected {
			return
		}
		if status == "Failed" || time.Now().After(deadline) {
			t.Fatalf("expected system %s status %s, got %s", id, expected, status)
		}
		time.Sleep(50 * time.Millisecond)

	}

}
