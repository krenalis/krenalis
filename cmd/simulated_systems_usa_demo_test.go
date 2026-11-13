// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestSimulatedSystemUSAGeneration checks private M2 persistence, crash replay,
// Source/Destination, Identity Resolution and preservation of Italian records.
// KRENALIS_TEST_USA_DEMO_BROWSER=true also verifies the Admin in Chromium.
func TestSimulatedSystemUSAGeneration(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("expected repository directory, got %v", err)
	}
	k := krenalistester.NewKrenalisInstance(t)
	assetRoot := filepath.Join(root, "tools/fakedata/testdata/published")
	origin := httptest.NewServer(http.FileServer(http.Dir(assetRoot)))
	defer origin.Close()
	k.SetExternalAssetsURLs([]string{origin.URL})
	k.Start()
	defer k.Stop()

	settings := &krenalistester.DBSettings{}
	err = json.Unmarshal(krenalistester.PostgresWarehouseSettings(), settings)
	if err != nil {
		t.Fatalf("expected warehouse settings, got %v", err)
	}
	admin, err := krenalistester.ConnectionPool(t.Context(), settings)
	if err != nil {
		t.Fatalf("expected warehouse administration pool, got %v", err)
	}
	database := "test_usa_demo_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	_, err = admin.Exec(t.Context(), "CREATE DATABASE "+database)
	if err != nil {
		admin.Close()
		t.Fatalf("expected isolated demo database, got %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := admin.Exec(ctx, "DROP DATABASE "+database+" WITH (FORCE)")
		if err != nil {
			t.Errorf("expected isolated demo database cleanup, got %v", err)
		}
		admin.Close()
	}()
	settings.Database = database
	pool, err := krenalistester.ConnectionPool(t.Context(), settings)
	if err != nil {
		t.Fatalf("expected isolated warehouse pool, got %v", err)
	}
	defer pool.Close()
	profileSchema := types.Object([]types.Property{
		{Name: "site_url", Type: types.String().AsURL(), ReadOptional: true},
		{Name: "first_name", Type: types.String(), ReadOptional: true},
		{Name: "last_name", Type: types.String(), ReadOptional: true},
		{Name: "email", Type: types.String().AsEmail(), ReadOptional: true},
		{Name: "phone", Type: types.String().AsPhone(), ReadOptional: true},
		{Name: "gender", Type: types.String(), ReadOptional: true},
		{Name: "birth_date", Type: types.Date(), ReadOptional: true},
		{Name: "current_age", Type: types.Int(8), ReadOptional: true},
		{Name: "address", Type: types.Object([]types.Property{
			{Name: "street", Type: types.String(), ReadOptional: true},
			{Name: "municipality", Type: types.String(), ReadOptional: true},
			{Name: "state_prov", Type: types.String(), ReadOptional: true},
			{Name: "postal_code", Type: types.String(), ReadOptional: true},
			{Name: "country", Type: types.String().AsCountry(types.ISO3166Alpha2), ReadOptional: true},
		}), ReadOptional: true},
		{Name: "photo", Type: types.String().AsURL(), ReadOptional: true},
		{Name: "education_level", Type: types.String(), ReadOptional: true},
		{Name: "primary_activity", Type: types.String(), ReadOptional: true},
		{Name: "annual_income_usd", Type: types.Int(32).Unsigned(), ReadOptional: true},
		{Name: "job_family", Type: types.String(), ReadOptional: true},
		{Name: "industry", Type: types.String(), ReadOptional: true},
		{Name: "job_level", Type: types.String(), ReadOptional: true},
	})
	var workspace struct{ ID string }
	k.Call("POST", "/v1/workspaces", http.Header{"Krenalis-Workspace": nil}, map[string]any{
		"name": "USA internal demo", "environment": "development",
		"profileSchema": profileSchema,
		"warehouse":     map[string]any{"platform": "PostgreSQL", "mode": "Normal", "settings": settings},
	}, &workspace)
	k.SetWorkspaceID(workspace.ID)
	var italy struct{ ID string }
	k.Call("POST", "/v1/simulated/systems", nil, map[string]any{
		"name": "Persisted Italy V5", "userCount": 3, "countries": map[string]int{"IT": 100},
	}, &italy)
	waitForSimulatedSystemStatus(t, k, italy.ID, "Available")
	italianRows := querySimulatedSystemRecords(t, pool, italy.ID)
	var italianCheckpoint string
	k.QueryRowTestDatabase(t.Context(), &italianCheckpoint,
		"SELECT generation_checkpoint::text FROM simulated_systems WHERE id = $1", italy.ID)
	k.ExecQueryTestDatabase(t.Context(), `CREATE SEQUENCE usa_m2_checkpoint_attempts;
		CREATE FUNCTION reject_usa_m2_checkpoint() RETURNS trigger LANGUAGE plpgsql AS $body$
		BEGIN IF NEW.generation_policy_version = 'generation-us-v2' AND NEW.generated_record_count > 256 THEN
		PERFORM nextval('usa_m2_checkpoint_attempts'); RAISE EXCEPTION 'USA checkpoint blocked by test';
		END IF; RETURN NEW; END $body$;
		CREATE TRIGGER reject_usa_m2_checkpoint BEFORE UPDATE ON simulated_systems
		FOR EACH ROW EXECUTE FUNCTION reject_usa_m2_checkpoint()`)
	var account struct{ ID string }
	k.Call("POST", "/v1/simulated/systems", nil, map[string]any{
		"name": "USA photographed people", "userCount": 513, "duplicateRecordPercent": 50,
		"countries": map[string]int{"US": 100},
	}, &account)
	deadline := time.Now().Add(45 * time.Second)
	var attempts int64
	for {
		var attempted bool
		k.QueryRowTestDatabase(t.Context(), &attempted, "SELECT is_called FROM usa_m2_checkpoint_attempts")
		if attempted {
			k.QueryRowTestDatabase(t.Context(), &attempts, "SELECT last_value FROM usa_m2_checkpoint_attempts")
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("expected blocked USA checkpoint after warehouse write, got timeout")
		}
		time.Sleep(50 * time.Millisecond)
	}
	var durableCount int
	k.QueryRowTestDatabase(t.Context(), &durableCount,
		"SELECT generated_record_count FROM simulated_systems WHERE id = $1", account.ID)
	beforeReplay := querySimulatedSystemRecords(t, pool, account.ID)
	if durableCount != 256 || len(beforeReplay) != 512 {
		t.Fatalf("expected durable count 256 and 512 written records, got %d and %d", durableCount, len(beforeReplay))
	}
	k.RestartServer(t.Context())
	deadline = time.Now().Add(45 * time.Second)
	for {
		var current int64
		k.QueryRowTestDatabase(t.Context(), &current, "SELECT last_value FROM usa_m2_checkpoint_attempts")
		if current > attempts {
			break
		}
		var status string
		k.QueryRowTestDatabase(t.Context(), &status, "SELECT status::text FROM simulated_systems WHERE id = $1", account.ID)
		if status == "Failed" {
			var message string
			k.QueryRowTestDatabase(t.Context(), &message, "SELECT generation_error FROM simulated_systems WHERE id = $1", account.ID)
			t.Fatalf("expected USA replay after process restart, got Failed: %s", message)
		}
		if time.Now().After(deadline) {
			t.Fatal("expected USA replay after process restart, got timeout")
		}
		time.Sleep(50 * time.Millisecond)
	}
	k.ExecQueryTestDatabase(t.Context(), `DROP TRIGGER reject_usa_m2_checkpoint ON simulated_systems;
		DROP FUNCTION reject_usa_m2_checkpoint(); DROP SEQUENCE usa_m2_checkpoint_attempts`)
	deadline = time.Now().Add(45 * time.Second)
	var detail struct {
		Status               string
		GeneratedRecordCount int
	}
	for {
		k.Call("GET", "/v1/simulated/systems/"+account.ID, nil, nil, &detail)
		if detail.Status == "Available" || detail.Status == "Failed" || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if detail.Status != "Available" || detail.GeneratedRecordCount != 513 {
		t.Fatalf("expected 513 available USA records, got status=%s count=%d", detail.Status, detail.GeneratedRecordCount)
	}
	afterReplay := querySimulatedSystemRecords(t, pool, account.ID)
	if len(afterReplay) != 513 {
		t.Fatalf("expected 513 distinct persisted USA records, got %d", len(afterReplay))
	}
	for id, before := range beforeReplay {
		if afterReplay[id].Data != before.Data {
			t.Fatalf("expected unchanged USA record %s after crash replay, got changed payload", id)
		}
	}
	var preview struct {
		Records []core.USADemoPreview
	}
	k.Call("GET", "/v1/simulated/systems/"+account.ID+"/usa-preview", nil, nil, &preview)
	if len(preview.Records) != 8 {
		t.Fatalf("expected eight stored demo records, got %d", len(preview.Records))
	}
	for _, record := range preview.Records {
		if strings.IndexFunc(record.Name, unicode.IsLower) < 0 {
			t.Fatalf("expected natural name from stored backend data, got %q", record.Name)
		}
	}
	t.Logf("persisted BASE-v2 preview: %s; %s", preview.Records[0].Name, preview.Records[1].Name)
	photo := preview.Records[0].Photo
	if !strings.HasPrefix(photo, "http://"+k.Addr()+"/v1/simulated/usa/photos/"+workspace.ID+"/face-") {
		t.Fatalf("expected local authenticated photo URL, got %q", photo)
	}
	request, err := http.NewRequestWithContext(t.Context(), "GET", photo, nil)
	if err != nil {
		t.Fatalf("expected unauthenticated photo request, got %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("expected unauthenticated photo response, got %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated HTTP 401, got %d", response.StatusCode)
	}
	client := &http.Client{}
	login, err := http.NewRequestWithContext(t.Context(), "POST", "http://"+k.Addr()+"/v1/members/login",
		strings.NewReader(`{"email":"acme@krenalis.com","password":"krenalis-password"}`))
	if err != nil {
		t.Fatalf("expected Admin login request, got %v", err)
	}
	login.Header.Set("Content-Type", "application/json")
	loggedIn, err := client.Do(login)
	if err != nil {
		t.Fatalf("expected Admin login response, got %v", err)
	}
	loggedIn.Body.Close()
	if loggedIn.StatusCode != http.StatusOK {
		t.Fatalf("expected Admin login HTTP 200, got %d", loggedIn.StatusCode)
	}
	request, err = http.NewRequestWithContext(t.Context(), "GET", photo, nil)
	if err != nil {
		t.Fatalf("expected authenticated photo request, got %v", err)
	}
	for _, cookie := range loggedIn.Cookies() {
		request.AddCookie(cookie)
	}
	response, err = client.Do(request)
	if err != nil {
		t.Fatalf("expected authenticated photo response, got %v", err)
	}
	defer response.Body.Close()
	image, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		t.Fatalf("expected photo bytes, got %v", err)
	}
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "image/webp" ||
		response.Header.Get("Cache-Control") != "private, no-store" {
		t.Fatalf("expected private authenticated WebP, got %d %v", response.StatusCode, response.Header)
	}
	id := strings.TrimSuffix(filepath.Base(photo), ".webp")
	imagePath := filepath.Join(assetRoot, "synthetic-world/photos/v1/256", id+".webp")
	original, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatalf("expected original rendition, got %v", err)
	}
	if !bytes.Equal(image, original) {
		t.Fatalf("expected original rendition SHA256=%x, got %x", sha256.Sum256(original), sha256.Sum256(image))
	}
	request.Header.Set("Authorization", "Bearer not-an-admin-session")
	response, err = client.Do(request)
	if err != nil {
		t.Fatalf("expected API-key photo response, got %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected API-key HTTP 401, got %d", response.StatusCode)
	}
	var checkpointJSON string
	k.QueryRowTestDatabase(t.Context(), &checkpointJSON,
		"SELECT generation_checkpoint::text FROM simulated_systems WHERE id = $1", account.ID)
	var checkpoint struct {
		MarketDataVersion string `json:"marketDataVersion"`
		USA               struct {
			Country  string `json:"country"`
			Currency string `json:"currency"`
		} `json:"usa"`
	}
	err = json.Unmarshal([]byte(checkpointJSON), &checkpoint)
	if err != nil {
		t.Fatalf("expected persisted checkpoint JSON, got %v", err)
	}
	if checkpoint.MarketDataVersion != "us-2026a-layer1-v2-candidate" ||
		checkpoint.USA.Country != "US" || checkpoint.USA.Currency != "USD" {
		t.Fatalf("expected persisted country-aware BASE-v2 checkpoint, got %+v", checkpoint)
	}
	var policy string
	k.QueryRowTestDatabase(t.Context(), &policy,
		"SELECT generation_policy_version FROM simulated_systems WHERE id = $1", account.ID)
	if policy != "generation-us-v2" {
		t.Fatalf("expected country-aware USA M2 policy, got %q", policy)
	}
	source := k.CreateConnection(krenalistester.ConnectionToCreate{
		Name: "USA demo Source", Role: krenalistester.Source, Connector: "simulated-application",
		Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": account.ID}),
	})
	var schemas struct{ In types.Type }
	k.Call("GET", "/v1/connections/"+source+"/pipelines/schemas/User", nil, nil, &schemas)
	if _, exists := schemas.In.Properties().ByName("annual_income_usd"); !exists {
		t.Fatal("expected USD Source schema, got missing income field")
	}
	if _, exists := schemas.In.Properties().ByName("annual_income_eur"); exists {
		t.Fatal("expected USD-only Source schema, got EUR field")
	}
	for _, name := range []string{"first_name", "last_name", "phone"} {
		property, exists := schemas.In.Properties().ByName(name)
		if !exists || !property.ReadOptional {
			t.Fatalf("expected approved optional USA Source property %s, got %+v", name, property)
		}
	}
	schemaJSON, err := schemas.In.MarshalJSON()
	if err != nil {
		t.Fatalf("expected USA Source schema JSON, got %v", err)
	}
	var observations struct{ Users []map[string]any }
	sourceIDs := make([]string, 0, len(afterReplay))
	for id := range afterReplay {
		sourceIDs = append(sourceIDs, id)
	}
	slices.Sort(sourceIDs)
	readSource := func() []map[string]any {

		t.Helper()
		var records []map[string]any
		for first := 0; first < len(sourceIDs); first += 50 {
			filter, err := json.Marshal(krenalistester.Filter{Operator: krenalistester.OpAnd,
				Rules: []krenalistester.FilterRule{&krenalistester.FilterCondition{
					Property: "customer_id", Operator: krenalistester.OpIsOneOf,
					Values: sourceIDs[first:min(first+50, len(sourceIDs))],
				}}})
			if err != nil {
				t.Fatalf("expected bounded Source filter, got %v", err)
			}
			var response struct{ Users []map[string]any }
			k.Call("GET", "/v1/connections/"+source+"/users?schema="+url.QueryEscape(string(schemaJSON))+
				"&filter="+url.QueryEscape(string(filter)), nil, nil, &response)
			records = append(records, response.Users...)
		}

		return records
	}
	observations.Users = readSource()
	if len(observations.Users) != 513 {
		t.Fatalf("expected 513 readable USA Source records, got %d", len(observations.Users))
	}
	for _, record := range observations.Users {
		for _, field := range []string{"first_name", "last_name"} {
			value, exists := record[field]
			if !exists {
				continue
			}
			name, ok := value.(string)
			if !ok || strings.IndexFunc(name, unicode.IsLower) < 0 {
				t.Fatalf("expected natural %s in persisted Source JSON, got %v", field, record[field])
			}
		}
		if record["annual_income_usd"] == nil || record["photo_id"] == nil || record["photo"] == nil ||
			record["annual_income_eur"] != nil || record["currency"] != "USD" || record["country"] != "US" {
			t.Fatalf("expected photographed USD facts through the Source connector, got %v", record)
		}
	}
	k.UpdateIdentityResolutionSettings(false, []string{"site_url"})
	mapping := map[string]string{
		"site_url": "site_url", "photo": "photo", "education_level": "education_level",
		"primary_activity": "primary_activity", "annual_income_usd": "annual_income_usd",
		"first_name": "first_name", "last_name": "last_name", "phone": "phone", "email": "email",
		"gender": "gender", "birth_date": "birth_date", "current_age": "current_age",
		"address.street": "address.street", "address.municipality": "address.municipality",
		"address.state_prov": "address.state_prov", "address.postal_code": "address.postal_code",
		"address.country": "address.country",
	}
	input := types.Prune(schemas.In, func(path string) bool { _, ok := mapping[path]; return ok })
	importOutput := types.Prune(profileSchema, func(path string) bool { _, ok := mapping[path]; return ok })
	importPipeline := k.CreatePipeline(source, "User", krenalistester.PipelineToSet{
		Name: "USA USD import", Enabled: true, InSchema: input, OutSchema: importOutput,
		Transformation: &krenalistester.Transformation{Mapping: mapping},
	})
	k.WaitForRunsCompletion(k.StartPipelineRun(importPipeline))
	k.RunIdentityResolutionAndWait()
	profileProperties := []string{"site_url", "photo", "education_level", "primary_activity", "annual_income_usd"}
	profiles, total := k.Profiles(profileProperties, "site_url", false, 0, 1000)
	if total != 257 || len(profiles) != 257 {
		t.Fatalf("expected 257 resolved USA identities from 513 observations, got %d and %d", total, len(profiles))
	}
	bySite := map[string]map[string]any{}
	counts := map[string]int{}
	for _, record := range observations.Users {
		site := record["site_url"].(string)
		bySite[site] = record
		counts[site]++
	}
	for _, profile := range profiles {
		record := bySite[profile.Attributes["site_url"].(string)]
		for _, property := range profileProperties {
			if profile.Attributes[property] != record[property] {
				t.Fatalf("expected resolved USA %s=%v, got %v", property, record[property], profile.Attributes[property])
			}
		}
	}
	var singleton map[string]any
	for site, count := range counts {
		if count == 1 {
			singleton = bySite[site]
			break
		}
	}
	if singleton == nil {
		t.Fatal("expected one observation without a duplicate, got none")
	}
	mode := krenalistester.Server
	destination := k.CreateConnection(krenalistester.ConnectionToCreate{
		Name: "USA Destination", Role: krenalistester.Destination, Connector: "simulated-application",
		SendingMode: &mode, Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": account.ID}),
	})
	destinationSchemas := simulatedApplicationPipelineSchemas(t, k, destination)
	output := types.Prune(destinationSchemas.Out, func(path string) bool {
		return path == "site_url" || path == "first_name"
	})
	updatePipeline := k.CreatePipeline(destination, "User", krenalistester.PipelineToSet{
		Name: "USA observation update", Enabled: true,
		InSchema: types.Prune(profileSchema, func(path string) bool { return path == "site_url" }), OutSchema: output,
		ExportMode: krenalistester.UpdateOnly, Matching: krenalistester.Matching{In: "site_url", Out: "site_url"},
		Filter: &krenalistester.Filter{Operator: krenalistester.OpAnd, Rules: []krenalistester.FilterRule{
			&krenalistester.FilterCondition{Property: "site_url", Operator: krenalistester.OpIs,
				Values: []string{singleton["site_url"].(string)}},
		}},
		Transformation: &krenalistester.Transformation{Mapping: map[string]string{
			"first_name": "site_url ' Updated'",
		}},
	})
	beforeDestination := querySimulatedSystemRecords(t, pool, account.ID)
	updateRun := k.StartPipelineRun(updatePipeline)
	k.WaitForRunsCompletion(updateRun)
	var claimed string
	k.QueryRowTestDatabase(t.Context(), &claimed,
		"SELECT simulated_system FROM pipelines_runs WHERE id = $1", updateRun)
	if claimed != account.ID {
		t.Fatalf("expected admitted USA Destination %s, got %s", account.ID, claimed)
	}
	var updated struct{ Users []map[string]any }
	updated.Users = readSource()
	if len(updated.Users) != 513 {
		t.Fatalf("expected unchanged USA record count after Destination, got %d", len(updated.Users))
	}
	for _, record := range updated.Users {
		expected := bySite[record["site_url"].(string)]
		if record["customer_id"] == singleton["customer_id"] {
			expected = maps.Clone(singleton)
			expected["first_name"] = singleton["site_url"].(string) + " Updated"
			if !reflect.DeepEqual(record, expected) {
				t.Fatalf("expected only first_name changed with all USD facts preserved, got %v", record)
			}
		}
	}
	afterDestination := querySimulatedSystemRecords(t, pool, account.ID)
	for id, before := range beforeDestination {
		if id != singleton["customer_id"] {
			if afterDestination[id] != before {
				t.Fatalf("expected unchanged untargeted USA record %s, got mutation", id)
			}
			continue
		}
		var a, b struct {
			Source json.Value `json:"source"`
		}
		err = json.Unmarshal([]byte(before.Data), &a)
		if err != nil {
			t.Fatalf("expected original candidate document, got %v", err)
		}
		err = json.Unmarshal([]byte(afterDestination[id].Data), &b)
		if err != nil {
			t.Fatalf("expected updated candidate document, got %v", err)
		}
		if !bytes.Equal(a.Source, b.Source) {
			t.Fatal("expected unchanged canonical USD candidate after Destination, got mutation")
		}
	}
	// Refresh the preview after the observation update before comparing restarts.
	k.Call("GET", "/v1/simulated/systems/"+account.ID+"/usa-preview", nil, nil, &preview)
	k.RestartServer(t.Context())
	var reloaded struct{ Records []core.USADemoPreview }
	k.Call("GET", "/v1/simulated/systems/"+account.ID+"/usa-preview", nil, nil, &reloaded)
	a, err := json.Marshal(preview)
	if err != nil {
		t.Fatalf("expected original preview JSON, got %v", err)
	}
	b, err := json.Marshal(reloaded)
	if err != nil {
		t.Fatalf("expected resumed preview JSON, got %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("expected identical stored USA preview after restart, got drift")
	}
	reloadedProfiles, reloadedTotal := k.Profiles(profileProperties, "site_url", false, 0, 1000)
	if reloadedTotal != total || !reflect.DeepEqual(profiles, reloadedProfiles) ||
		!reflect.DeepEqual(afterDestination, querySimulatedSystemRecords(t, pool, account.ID)) {
		t.Fatal("expected preserved USA profiles, identities and records after restart, got mutation")
	}
	var currentItalianCheckpoint string
	k.QueryRowTestDatabase(t.Context(), &currentItalianCheckpoint,
		"SELECT generation_checkpoint::text FROM simulated_systems WHERE id = $1", italy.ID)
	if currentItalianCheckpoint != italianCheckpoint ||
		!reflect.DeepEqual(italianRows, querySimulatedSystemRecords(t, pool, italy.ID)) {
		t.Fatal("expected unchanged persisted Italian V5 rows and checkpoint, got mutation")
	}
	testContext := t.Context()
	t.Run("empty USA", func(t *testing.T) {

		var empty struct{ ID string }
		k.Call("POST", "/v1/simulated/systems", nil, map[string]any{
			"name": "Empty USA Destination", "userCount": 0, "countries": map[string]int{"US": 100},
		}, &empty)
		waitForSimulatedSystemStatus(t, k, empty.ID, "Available")
		var bound bool
		k.QueryRowTestDatabase(t.Context(), &bound, `SELECT generation_policy_version = 'generation-us-v2'
			AND generation_checkpoint->'usa'->>'country' = 'US' AND generated_record_count = 0
			FROM simulated_systems WHERE id = $1`, empty.ID)
		if !bound || len(querySimulatedSystemRecords(t, pool, empty.ID)) != 0 {
			t.Fatal("expected an empty country-bound USA system, got missing binding or generated records")
		}
		createProperties := []string{"first_name", "last_name", "email", "gender", "birth_date", "current_age",
			"phone", "site_url", "address", "photo"}
		createProfiles, _ := k.Profiles(createProperties, "site_url", false, 0, 1000)
		var createProfile map[string]any
		for _, profile := range createProfiles {
			if profile.Attributes["first_name"] != nil && profile.Attributes["last_name"] != nil &&
				profile.Attributes["phone"] != nil {
				createProfile = profile.Attributes
				break
			}
		}
		if createProfile == nil {
			t.Fatal("expected one profile with required Destination contacts, got none")
		}
		connection := k.CreateConnection(krenalistester.ConnectionToCreate{
			Name: "USA create Destination", Role: krenalistester.Destination, Connector: "simulated-application",
			SendingMode: &mode, Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": empty.ID}),
		})
		createSchemas := simulatedApplicationPipelineSchemas(t, k, connection)
		createMapping := map[string]string{}
		for _, property := range createProperties {
			if property != "address" && property != "site_url" {
				createMapping[property] = property
			}
		}
		for _, leaf := range []string{"street", "municipality", "state_prov", "postal_code", "country"} {
			createMapping["address."+leaf] = "address." + leaf
		}
		createPipeline := k.CreatePipeline(connection, "User", krenalistester.PipelineToSet{
			Name: "USA create observation", Enabled: true,
			InSchema: types.Prune(profileSchema, func(path string) bool {
				return slices.Contains(createProperties, strings.Split(path, ".")[0])
			}), OutSchema: createSchemas.Out, ExportMode: krenalistester.CreateOrUpdate,
			Matching: krenalistester.Matching{In: "site_url", Out: "site_url"},
			Filter: &krenalistester.Filter{Operator: krenalistester.OpAnd, Rules: []krenalistester.FilterRule{
				&krenalistester.FilterCondition{Property: "site_url", Operator: krenalistester.OpIs,
					Values: []string{createProfile["site_url"].(string)}},
			}},
			Transformation: &krenalistester.Transformation{Mapping: createMapping},
		})
		k.WaitForRunsCompletion(k.StartPipelineRun(createPipeline))
		createdRows := querySimulatedSystemRecords(t, pool, empty.ID)
		if len(createdRows) != 1 {
			t.Fatalf("expected one admitted USA Destination create, got %d rows", len(createdRows))
		}
		createdSource := k.CreateConnection(krenalistester.ConnectionToCreate{
			Name: "USA created Source", Role: krenalistester.Source, Connector: "simulated-application",
			Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": empty.ID}),
		})
		var created struct{ Users []map[string]any }
		k.Call("GET", "/v1/connections/"+createdSource+"/users?schema="+url.QueryEscape(string(schemaJSON)),
			nil, nil, &created)
		if len(created.Users) != 1 {
			t.Fatalf("expected readable Destination-created USA observation, got %d records", len(created.Users))
		}
		observation := created.Users[0]
		if observation["country"] != "US" || observation["currency"] != "USD" ||
			observation["site_url"] != createProfile["site_url"] || observation["annual_income_usd"] != nil ||
			observation["annual_income_eur"] != nil || observation["photo_id"] != nil {
			t.Fatalf("expected US Destination observation without invented facts, got %v", observation)
		}
		var generated struct{ Records []core.USADemoPreview }
		k.Call("GET", "/v1/simulated/systems/"+empty.ID+"/usa-preview", nil, nil, &generated)
		if len(generated.Records) != 0 {
			t.Fatalf("expected no generated preview for Destination-only observations, got %v", generated.Records)
		}
		k.RestartServer(testContext)
		if !reflect.DeepEqual(createdRows, querySimulatedSystemRecords(t, pool, empty.ID)) {
			t.Fatal("expected unchanged Destination-created USA observation after restart, got mutation")
		}

	})
	t.Run("mixed countries", func(t *testing.T) {
		err := k.TryCall("POST", "/v1/simulated/systems", nil, map[string]any{
			"name": "mixed", "userCount": 1, "countries": map[string]int{"US": 50, "IT": 50},
		}, nil)
		if err != nil {
			response, ok := errors.AsType[*krenalistester.StatusCodeError](err)
			if !ok || response.Response.Code != http.StatusBadRequest {
				t.Fatalf("expected mixed-country HTTP error, got %v", err)
			}
			return
		}
		t.Fatal("expected mixed-country rejection, got success")
	})
	if os.Getenv("KRENALIS_TEST_USA_DEMO_BROWSER") == "true" {
		command := exec.CommandContext(t.Context(), "node", "--experimental-strip-types",
			filepath.Join(root, "admin/tests/usaDemo.browser.mjs"), "http://"+k.Addr(), workspace.ID)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("expected photographed USA demo in the browser, got %v\n%s", err, output)
		}
		t.Logf("%s", output)
	}

}
