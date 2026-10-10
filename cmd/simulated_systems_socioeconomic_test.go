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
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestSimulatedSystemSocioeconomicDestinationRoundTrip preserves generated facts
// through an admitted Destination partial update and a subsequent Source read.
func TestSimulatedSystemSocioeconomicDestinationRoundTrip(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	fixture := simulatedSystemModelFixture(t)
	photos, err := os.ReadFile("../tools/fakedata/testdata/external-photos-v1.json")
	if err != nil {
		t.Fatalf("expected photo fixture, got %v", err)
	}
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/synthetic-world/it-2026a/model.json":
			_, _ = w.Write(fixture)
		case "/synthetic-world/photos/v1/catalog.json":
			_, _ = w.Write(photos)
		default:
			t.Errorf("expected model or photo catalog request, got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer origin.Close()
	k := krenalistester.NewKrenalisInstance(t)
	k.SetExternalAssetsURLs([]string{origin.URL})
	k.Start()
	defer k.Stop()

	settings := &krenalistester.DBSettings{}
	err = json.Unmarshal(krenalistester.PostgresWarehouseSettings(), settings)
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
		"name": "Generated socioeconomic round trip", "environment": "development", "profileSchema": profileSchema,
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
	if policy != "generation-v4" {
		t.Fatalf("expected generation-v4, got %s", policy)
	}
	rows := querySimulatedSystemRecords(t, pool, system.ID)
	if len(rows) != 1 {
		t.Fatalf("expected one generated record, got %d", len(rows))
	}

	// Independently reconstruct the workspace World and the first generated Person.
	photoSource, err := fakedata.NewFaceCatalogSource([]string{origin.URL})
	if err != nil {
		t.Fatalf("expected photo source, got %v", err)
	}
	catalog, err := photoSource.Load(t.Context())
	if err != nil {
		t.Fatalf("expected photo catalog, got %v", err)
	}
	modelSource, err := fakedata.NewSocioeconomicModelSource([]string{origin.URL})
	if err != nil {
		t.Fatalf("expected model source, got %v", err)
	}
	model, err := modelSource.Load(t.Context())
	if err != nil {
		t.Fatalf("expected model, got %v", err)
	}
	worldKey := sha256.Sum256([]byte("krenalis/simulated-world/v1/" + workspace.ID))
	namespace, err := fakedata.NewIdentityNamespace("w"+hex.EncodeToString(worldKey[:10]), 1)
	if err != nil {
		t.Fatalf("expected workspace namespace, got %v", err)
	}
	base, err := fakedata.NewWorld(fakedata.WorldConfig{
		IdentityNamespace: namespace, WorldSeed: binary.BigEndian.Uint64(worldKey[10:18]),
		ReferenceDate: "2026-01-01", FaceCatalog: catalog,
	})
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
	profiles, total := k.Profiles([]string{"site_url"}, "", false, 0, 10)
	if total != 1 || len(profiles) != 1 || profiles[0].Attributes["site_url"] != before["site_url"] {
		t.Fatalf("expected one imported generated identity, got total=%d profiles=%v", total, profiles)
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
	t.Logf("verified generated v4 round trip: PersonIndex=%d activity=%s income=%d",
		index, facts.PrimaryActivity, facts.Income)

}

// TestSimulatedSystemSocioeconomicLifecycle exercises cmd/Core ownership,
// cancellation, acquisition/construction failures and historical preparation.
func TestSimulatedSystemSocioeconomicLifecycle(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	model := simulatedSystemModelFixture(t)
	photos, err := os.ReadFile("../tools/fakedata/testdata/external-photos-v1.json")
	if err != nil {
		t.Fatalf("expected photo fixture, got %v", err)
	}
	var zeroModel []byte
	var mode, calls atomic.Int32
	started, canceled := make(chan struct{}, 1), make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/synthetic-world/photos/v1/catalog.json" {
			_, _ = w.Write(photos)
			return
		}
		if r.URL.Path != "/synthetic-world/it-2026a/model.json" {
			t.Errorf("expected photo catalog or fixed model path, got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		calls.Add(1)
		switch mode.Load() {
		case 0:
			started <- struct{}{}
			<-r.Context().Done()
			canceled <- struct{}{}
		case 1:
			_, _ = w.Write(model)
		case 2:
			_, _ = w.Write([]byte(`{"schema":"invalid"}`))
		case 3:
			_, _ = w.Write(zeroModel)
		}

	}))
	defer server.Close()
	k := krenalistester.NewKrenalisInstance(t)
	k.SetExternalAssetsURLs([]string{server.URL})
	k.Start()
	defer k.Stop()
	if calls.Load() != 0 {
		t.Fatalf("expected no model acquisition at startup, got %d requests", calls.Load())
	}

	settings := &krenalistester.DBSettings{}
	err = json.Unmarshal(krenalistester.PostgresWarehouseSettings(), settings)
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
		"profileSchema": types.Object([]types.Property{
			{Name: "email", Type: types.String().AsEmail(), ReadOptional: true},
		}),
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
		rows := querySimulatedSystemRecords(t, pool, id)
		if count != 0 || len(rows) != 0 {
			t.Fatalf("expected no model-dependent records or progress, got count=%d rows=%d", count, len(rows))
		}

	}

	empty := create("empty", 0)
	waitForSimulatedSystemStatus(t, k, empty, "Available")
	var policy string
	k.QueryRowTestDatabase(t.Context(), &policy,
		"SELECT generation_policy_version FROM simulated_systems WHERE id = $1", empty)
	if policy != "empty-v1" || calls.Load() != 0 {
		t.Fatalf("expected empty-v1 without model acquisition, got %s and %d requests", policy, calls.Load())
	}
	interrupted := create("interrupted acquisition", 3)
	select {
	case <-started:
	case <-time.After(20 * time.Second):
		t.Fatal("expected worker model request, got timeout")
	}
	assertNoProgress(interrupted)
	var checkpoint string
	k.QueryRowTestDatabase(t.Context(), &checkpoint,
		"SELECT generation_checkpoint::text FROM simulated_systems WHERE id = $1", interrupted)
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

	// Recreate persisted historical Preparing metadata using the unchanged
	// initial checkpoint format, then restart with no usable model origin.
	legacy := create("historical preparing", 0)
	k.ExecQueryTestDatabase(t.Context(), `UPDATE simulated_systems SET status = 'Preparing', user_count = 3,
		generation_policy_version = 'generation-v3', generation_checkpoint = $1::jsonb WHERE id = $2`,
		checkpoint, legacy)
	mode.Store(2)
	k.RestartServer(t.Context())
	waitForSimulatedSystemStatus(t, k, legacy, "Available")
	if calls.Load() != 4 {
		t.Fatalf("expected historical preparation without model acquisition, got %d requests", calls.Load())
	}
	rows := querySimulatedSystemRecords(t, pool, legacy)
	if len(rows) != 3 {
		t.Fatalf("expected three historical records, got %d", len(rows))
	}
	for _, row := range rows {

		fields := map[string]json.Value{}
		err := json.Unmarshal([]byte(row.Data), &fields)
		if err != nil {
			t.Fatalf("expected historical document JSON, got %v", err)
		}
		for _, name := range []string{"education_level", "primary_activity", "annual_income_eur",
			"job_family", "industry", "job_level"} {
			if _, ok := fields[name]; ok {
				t.Fatalf("expected historical %s absent, got %s", name, row.Data)
			}
		}

	}

	zeroModel = simulatedSystemConstructionFailureModel(t, workspace.ID, server.URL, model)
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

// simulatedSystemConstructionFailureModel selects and verifies a valid model
// whose one-Company universe has no realized employer for this workspace.
func simulatedSystemConstructionFailureModel(t *testing.T, workspaceID, origin string, fixture []byte) []byte {

	t.Helper()
	photos, err := fakedata.NewFaceCatalogSource([]string{origin})
	if err != nil {
		t.Fatalf("expected photo source, got %v", err)
	}
	catalog, err := photos.Load(t.Context())
	if err != nil {
		t.Fatalf("expected acquired photo catalog, got %v", err)
	}
	key := sha256.Sum256([]byte("krenalis/simulated-world/v1/" + workspaceID))
	namespace, err := fakedata.NewIdentityNamespace("w"+hex.EncodeToString(key[:10]), 1)
	if err != nil {
		t.Fatalf("expected workspace namespace, got %v", err)
	}
	base, err := fakedata.NewWorld(fakedata.WorldConfig{
		IdentityNamespace: namespace, WorldSeed: binary.BigEndian.Uint64(key[10:18]),
		ReferenceDate: "2026-01-01", FaceCatalog: catalog,
	})
	if err != nil {
		t.Fatalf("expected workspace base World, got %v", err)
	}
	fields := map[string]json.Value{}
	err = json.Unmarshal(fixture, &fields)
	if err != nil {
		t.Fatalf("expected artificial model JSON, got %v", err)
	}
	fields["companyCount"] = json.Value(`1`)
	for positive := 1; positive <= 8; positive++ {

		// Positive support keeps acquisition valid; construction must still
		// reject an all-zero realization. Verify against the actual seed so
		// this test does not depend on a probabilistic fixture assumption.
		law, err := json.Marshal(map[string][]uint32{"values": {0, uint32(positive)}, "weights": {1<<32 - 1, 1}})
		if err != nil {
			t.Fatalf("expected count law JSON, got %v", err)
		}
		laws := strings.TrimSuffix(strings.Repeat(string(law)+",", 14), ",")
		fields["employeeCount"] = json.Value(`{"laws":[` + laws + `]}`)
		candidate, err := json.Marshal(fields)
		if err != nil {
			t.Fatalf("expected candidate model JSON, got %v", err)
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(candidate)
		}))
		source, err := fakedata.NewSocioeconomicModelSource([]string{server.URL})
		if err != nil {
			server.Close()
			t.Fatalf("expected model source, got %v", err)
		}
		model, err := source.Load(t.Context())
		server.Close()
		if err != nil {
			t.Fatalf("expected acquisition-valid construction fixture, got %v", err)
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

func simulatedSystemModelFixture(t *testing.T) []byte {
	t.Helper()
	model, err := os.ReadFile("../core/testdata/socioeconomic-model.json")
	if err != nil {
		t.Fatalf("expected artificial socioeconomic model fixture, got %v", err)
	}
	return model
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
