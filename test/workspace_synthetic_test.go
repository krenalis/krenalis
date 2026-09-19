// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	_ "github.com/krenalis/krenalis/connectors/s3"
	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

func syntheticCatalogFixture(t *testing.T) string {
	t.Helper()
	root := filepath.Join("..", "tools", "fakedata", "testdata", "fakefacegen-v1-selected")
	dir := t.TempDir()
	var catalog, manifest map[string]any
	var specs []map[string]any
	for _, item := range []struct {
		name string
		out  any
	}{{"catalog.json", &catalog}, {"specs.json", &specs}, {"manifest.json", &manifest}} {
		data, err := os.ReadFile(filepath.Join(root, item.name))
		if err != nil {
			t.Fatalf("expected successful operation, got %v", err)
		}
		err = json.Unmarshal(data, item.out)
		if err != nil {
			t.Fatalf("expected successful operation, got %v", err)
		}
	}
	catalog["count"] = 2
	newSpecs := make([]map[string]any, 0, 2)
	newAssets := make([]map[string]any, 0, 2)
	for i, sourceID := range []string{"face-000013", "face-000037"} {
		id := fmt.Sprintf("face-%06d", i+1)
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
				t.Fatalf("expected successful operation, got %v", err)
			}
			path := filepath.Join(dir, folder)
			err = os.MkdirAll(path, 0o755)
			if err != nil {
				t.Fatalf("expected successful operation, got %v", err)
			}
			err = os.WriteFile(filepath.Join(path, id+".webp"), data, 0o644)
			if err != nil {
				t.Fatalf("expected successful operation, got %v", err)
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
			t.Fatalf("expected successful operation, got %v", err)
		}
		err = os.WriteFile(filepath.Join(dir, item.name), data, 0o644)
		if err != nil {
			t.Fatalf("expected successful operation, got %v", err)
		}
	}
	return dir
}

func TestSyntheticAndNormalWorkspaces(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	photoDir := syntheticCatalogFixture(t)
	catalog, err := fakedata.LoadFaceCatalog(t.Context(), photoDir)
	if err != nil {
		t.Fatalf("expected successful operation, got %v", err)
	}
	storageDir, err := filepath.Abs("testdata/storage")
	if err != nil {
		t.Fatalf("expected successful operation, got %v", err)
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.SetFileSystemRoot(storageDir)
	k.SetSyntheticPhotosDir(photoDir)
	config := core.SyntheticConfig{
		Namespace: "workspace-test", Generation: 1, Seed: 726381, ReferenceDate: "2026-01-01", PersonCount: 20,
		PhotoOrigin: "http://" + k.Addr(), SourceAID: "a", SourceAVersion: "demo-v1",
		SourceACoverageNumerator: 1, SourceACoverageDenominator: 1,
		SourceADuplicateNumerator: 1, SourceADuplicateDenominator: 1,
		SourceBID: "b", SourceBVersion: "demo-v1", SourceBCoverageNumerator: 1,
		SourceBCoverageDenominator: 1, SourceBDuplicateNumerator: 0, SourceBDuplicateDenominator: 1,
	}
	k.SetSyntheticConfig(&config)
	k.Start()
	defer k.Stop()

	normalID := k.WorkspaceID()
	var normal struct {
		Name          string             `json:"name"`
		Synthetic     bool               `json:"synthetic"`
		UIPreferences core.UIPreferences `json:"uiPreferences"`
	}
	k.Call("GET", "/v1/workspaces/current", nil, nil, &normal)
	if normal.Synthetic {
		t.Fatalf("expected bootstrap workspace to be normal, got synthetic")
	}
	err = k.TryCall("PUT", "/v1/workspaces/current", nil, map[string]any{
		"name": "normal", "synthetic": true, "uiPreferences": normal.UIPreferences,
	}, nil)
	assertSyntheticBadRequest(t, err)
	k.Call("PUT", "/v1/workspaces/current", nil, map[string]any{
		"name": "normal-renamed", "uiPreferences": normal.UIPreferences,
	}, nil)
	k.Call("GET", "/v1/workspaces/current", nil, nil, &normal)
	if normal.Name != "normal-renamed" || normal.Synthetic {
		t.Fatalf("expected renamed normal workspace, got %+v", normal)
	}
	var catalogResponse struct {
		Connectors []struct {
			Code          string    `json:"code"`
			AsSource      *struct{} `json:"asSource"`
			AsDestination *struct{} `json:"asDestination"`
		} `json:"connectors"`
	}
	k.Call("GET", "/v1/connectors", nil, nil, &catalogResponse)
	if len(catalogResponse.Connectors) <= 2 {
		t.Fatalf("expected normal catalog, got %d entries", len(catalogResponse.Connectors))
	}
	fs := k.CreateSourceFileSystem()
	normalRows, _ := k.File(fs, "users.csv", "csv", "", krenalistester.NoCompression,
		krenalistester.JSONEncodeSettings(map[string]any{"separator": ",", "hasColumnNames": true}), 100)
	if len(normalRows) != 2 {
		t.Fatalf("expected 2 real FileSystem rows, got %d", len(normalRows))
	}

	settings := &krenalistester.DBSettings{}
	err = json.Unmarshal(krenalistester.PostgresWarehouseSettings(), settings)
	if err != nil {
		t.Fatalf("expected successful operation, got %v", err)
	}
	pool, err := krenalistester.ConnectionPool(t.Context(), settings)
	if err != nil {
		t.Fatalf("expected successful operation, got %v", err)
	}
	_, err = pool.Exec(t.Context(), "CREATE DATABASE test_synthetic_workspace")
	pool.Close()
	if err != nil {
		t.Fatalf("expected successful operation, got %v", err)
	}
	settings.Database = "test_synthetic_workspace"
	profileSchema := types.Object([]types.Property{
		{Name: "first_name", Type: types.String().WithMaxLength(100), ReadOptional: true},
		{Name: "email", Type: types.String().WithMaxLength(254), ReadOptional: true},
	})
	request := map[string]any{"name": "synthetic", "synthetic": true, "profileSchema": profileSchema,
		"warehouse":     map[string]any{"platform": "PostgreSQL", "mode": "Normal", "settings": settings},
		"uiPreferences": map[string]any{"profile": map[string]string{"image": "", "firstName": "first_name", "lastName": "", "extra": "email"}}}
	headers := http.Header{"Krenalis-Workspace": nil}
	k.Call("POST", "/v1/workspaces/test", headers, request, nil)
	request["synthetic"] = false
	k.Call("POST", "/v1/workspaces/test", headers, request, nil)
	for _, invalid := range []any{nil, "true", 1} {
		request["synthetic"] = invalid
		for _, endpoint := range []string{"/v1/workspaces/test", "/v1/workspaces"} {
			err := k.TryCall("POST", endpoint, headers, request, nil)
			assertSyntheticBadRequest(t, err)
		}
	}
	request["synthetic"] = true
	var created struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/workspaces", headers, request, &created)
	k.SetWorkspaceID(created.ID)
	var syntheticWorkspace struct {
		Name          string             `json:"name"`
		Synthetic     bool               `json:"synthetic"`
		UIPreferences core.UIPreferences `json:"uiPreferences"`
	}
	k.Call("GET", "/v1/workspaces/current", nil, nil, &syntheticWorkspace)
	if !syntheticWorkspace.Synthetic {
		t.Fatalf("expected synthetic workspace, got normal")
	}
	for _, value := range []bool{true, false} {
		err := k.TryCall("PUT", "/v1/workspaces/current", nil, map[string]any{
			"name": "synthetic", "synthetic": value, "uiPreferences": request["uiPreferences"],
		}, nil)
		assertSyntheticBadRequest(t, err)
	}
	k.Call("PUT", "/v1/workspaces/current", nil, map[string]any{
		"name": "synthetic-renamed", "uiPreferences": map[string]any{"profile": map[string]string{
			"image": "", "firstName": "first_name", "lastName": "", "extra": "",
		}},
	}, nil)
	k.Call("GET", "/v1/workspaces/current", nil, nil, &syntheticWorkspace)
	if syntheticWorkspace.Name != "synthetic-renamed" || !syntheticWorkspace.Synthetic || syntheticWorkspace.UIPreferences.Profile.Extra != "" {
		t.Fatalf("expected renamed Synthetic workspace, got %+v", syntheticWorkspace)
	}
	var list struct {
		Workspaces []struct {
			ID        string `json:"id"`
			Synthetic bool   `json:"synthetic"`
		} `json:"workspaces"`
	}
	k.Call("GET", "/v1/workspaces", headers, nil, &list)
	if len(list.Workspaces) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(list.Workspaces))
	}
	modes := map[string]bool{}
	for _, workspace := range list.Workspaces {
		modes[workspace.ID] = workspace.Synthetic
	}
	if normalMode, ok := modes[normalID]; !ok || normalMode {
		t.Fatalf("expected normal %s in workspace list, got %v", normalID, modes)
	}
	if syntheticMode, ok := modes[created.ID]; !ok || !syntheticMode {
		t.Fatalf("expected normal %s and Synthetic %s, got %v", normalID, created.ID, modes)
	}
	k.Call("GET", "/v1/connectors", nil, nil, &catalogResponse)
	if len(catalogResponse.Connectors) != 2 {
		t.Fatalf("expected 2 Synthetic connectors, got %d", len(catalogResponse.Connectors))
	}
	seen := map[string]bool{}
	for _, connector := range catalogResponse.Connectors {
		seen[connector.Code] = true
		if connector.AsSource == nil || connector.AsDestination != nil {
			t.Fatalf("expected source-only Synthetic connector, got %+v", connector)
		}
	}
	if !seen["s3"] || !seen["csv"] {
		t.Fatalf("expected S3 and CSV catalog, got %v", seen)
	}
	for _, item := range []struct {
		code     string
		settings any
	}{
		{"sftp", map[string]any{"host": "127.0.0.1", "port": 22, "username": "demo", "password": "demo"}},
		{"postgresql", settings},
		{"hubspot", map[string]any{"accessToken": "demo-token"}},
	} {
		err := k.TryCall("POST", "/v1/connections", nil, map[string]any{
			"name": "unsupported", "role": "Source", "connector": item.code, "settings": item.settings,
		}, nil)
		assertSyntheticBadRequest(t, err)
		if !strings.Contains(err.Error(), "Synthetic workspace supports only S3 and CSV sources") {
			t.Fatalf("expected Synthetic connector rejection, got %v", err)
		}
	}
	err = k.TryCall("POST", "/v1/connections", nil, map[string]any{
		"name": "destination", "role": "Destination", "connector": "s3",
		"settings": map[string]any{"accessKeyID": "AAAAAAAAAAAAAAAAAAAA",
			"secretAccessKey": "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB", "region": "us-east-1", "bucket": "demo"},
	}, nil)
	assertSyntheticBadRequest(t, err)
	if !strings.Contains(err.Error(), "Synthetic workspace supports only S3 and CSV sources") {
		t.Fatalf("expected Synthetic destination rejection, got %v", err)
	}
	err = k.TryCall("POST", "/v1/ui", nil, map[string]any{"connector": "sftp", "role": "Source"}, nil)
	assertSyntheticBadRequest(t, err)
	err = k.TryCall("GET", "/v1/connections/auth-url?connector=hubspot&role=Source&redirectURI=http%3A%2F%2Flocalhost", nil, nil, nil)
	assertSyntheticBadRequest(t, err)
	err = k.TryCall("GET", "/v1/connections/auth-token?connector=hubspot&authCode=code&redirectURI=http%3A%2F%2Flocalhost", nil, nil, nil)
	assertSyntheticBadRequest(t, err)

	s3 := k.CreateConnection(krenalistester.ConnectionToCreate{Name: "demo", Role: krenalistester.Source,
		Connector: "s3", Settings: krenalistester.JSONEncodeSettings(map[string]any{
			"accessKeyID": "AAAAAAAAAAAAAAAAAAAA", "secretAccessKey": "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB",
			"region": "us-east-1", "bucket": "demo"})})
	if path := k.AbsolutePath(s3, "customers-a.csv"); path != "s3://demo/customers-a.csv" {
		t.Fatalf("expected S3 path, got %q", path)
	}
	err = k.TryCall("GET", "/v1/connections/"+s3+"/files?path=customers-b.csv&format=json&limit=100", nil, nil, nil)
	assertSyntheticBadRequest(t, err)
	base, err := fakedata.NewWorld(fakedata.WorldConfig{IdentityNamespace: mustNamespace(t), WorldSeed: config.Seed,
		ReferenceDate: config.ReferenceDate, FaceCatalog: catalog})
	if err != nil {
		t.Fatalf("expected successful operation, got %v", err)
	}
	world, err := fakedata.NewSourceWorld(base, []fakedata.CountryShare{{Code: "IT", Version: fakedata.MarketDataVersion, Weight: 1}})
	if err != nil {
		t.Fatalf("expected successful operation, got %v", err)
	}
	for _, source := range []struct {
		name, id         string
		duplicate, count int
	}{{"customers-a.csv", "a", 1, 40}, {"customers-b.csv", "b", 0, 20}} {
		instance, err := fakedata.NewSourceInstance(world, fakedata.SourceInstanceConfig{ID: source.id, Version: "demo-v1",
			CoverageNumerator: 1, CoverageDenominator: 1, DuplicateNumerator: uint64(source.duplicate), DuplicateDenominator: 1})
		if err != nil {
			t.Fatalf("expected successful operation, got %v", err)
		}
		expected := map[string]fakedata.SourceRecord{}
		for index := fakedata.PersonIndex(1); index <= 20; index++ {
			records, err := instance.Records(index)
			if err != nil {
				t.Fatalf("expected successful operation, got %v", err)
			}
			for _, record := range records {
				expected[record.ID] = record
			}
		}
		rows, _ := k.File(s3, source.name, "csv", "", krenalistester.NoCompression,
			krenalistester.JSONEncodeSettings(map[string]any{"separator": ",", "hasColumnNames": true}), 100)
		if len(rows) != source.count {
			t.Fatalf("expected %d %s rows, got %d", source.count, source.name, len(rows))
		}
		for _, row := range rows {
			id, ok := row["source_record_id"].(string)
			if !ok {
				t.Fatalf("expected source_record_id string, got %T", row["source_record_id"])
			}
			record, ok := expected[id]
			if !ok {
				t.Fatalf("expected known source record, got %q", id)
			}
			delete(expected, id)
			if _, ok := row["person_id"]; ok {
				t.Fatalf("expected hidden PersonID, got %v", row)
			}
			for _, field := range []struct {
				key   string
				value *string
			}{{"first_name", record.FirstName}, {"last_name", record.LastName}, {"email", record.Email}, {"phone", record.Phone}} {
				want := ""
				if field.value != nil {
					want = *field.value
				}
				got, _ := row[field.key].(string)
				if got != want {
					t.Fatalf("expected %s %q, got %q", field.key, want, got)
				}
			}
			if record.PhotoID != nil {
				asset, err := catalog.Asset(*record.PhotoID, fakedata.PhotoSize256)
				if err != nil {
					t.Fatalf("expected successful operation, got %v", err)
				}
				url := config.PhotoOrigin + asset.Path
				if row["photo_url"] != url {
					t.Fatalf("expected photo URL %q, got %v", url, row["photo_url"])
				}
			}
		}
		if len(expected) != 0 {
			t.Fatalf("expected all %s records, got %d missing", source.name, len(expected))
		}
	}
	rows, _ := k.File(s3, "customers-b.csv", "csv", "", krenalistester.NoCompression,
		krenalistester.JSONEncodeSettings(map[string]any{"separator": ",", "hasColumnNames": true}), 100)
	photoHashes := map[string]string{}
	for _, row := range rows {
		url, _ := row["photo_url"].(string)
		if url == "" {
			continue
		}
		parts := strings.Split(url, "/")
		asset, err := catalog.Asset(parts[len(parts)-1][:len(parts[len(parts)-1])-5], fakedata.PhotoSize256)
		if err != nil {
			t.Fatalf("expected successful operation, got %v", err)
		}
		photoHashes[url] = asset.ContentSHA256
	}
	if len(photoHashes) != 2 {
		t.Fatalf("expected 2 photo URLs, got %d", len(photoHashes))
	}
	for url, want := range photoHashes {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		if err != nil {
			t.Fatalf("expected successful operation, got %v", err)
		}
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("expected successful operation, got %v", err)
		}
		data, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatalf("expected successful operation, got %v", err)
		}
		got := sha256.Sum256(data)
		if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "image/webp" || hex.EncodeToString(got[:]) != want {
			t.Fatalf("expected verified WebP %s, got status %d type %s hash %x", want, response.StatusCode, response.Header.Get("Content-Type"), got)
		}
	}
	inSchema := types.Object([]types.Property{
		{Name: "source_record_id", Type: types.String()},
		{Name: "first_name", Type: types.String()},
		{Name: "email", Type: types.String()},
	})
	pipelineSettings := krenalistester.PipelineToSet{Name: "Synthetic B", Enabled: true,
		Path: "customers-b.csv", Format: "csv", UserIDColumn: "source_record_id", InSchema: inSchema,
		OutSchema: profileSchema, Transformation: &krenalistester.Transformation{Mapping: map[string]string{
			"first_name": "first_name", "email": "email"}}, FormatSettings: krenalistester.JSONEncodeSettings(map[string]any{
			"separator": ",", "hasColumnNames": true})}
	unsupported := pipelineSettings
	unsupported.Format = "json"
	_, err = k.TryCreatePipeline(s3, "User", unsupported)
	assertSyntheticBadRequest(t, err)
	if !strings.Contains(err.Error(), "Synthetic workspace supports only CSV source format") {
		t.Fatalf("expected unsupported format rejection, got %v", err)
	}
	missing := pipelineSettings
	missing.Format = ""
	_, err = k.TryCreatePipeline(s3, "User", missing)
	assertSyntheticBadRequest(t, err)
	if !strings.Contains(err.Error(), "Synthetic workspace requires CSV source format") {
		t.Fatalf("expected missing format rejection, got %v", err)
	}
	pipeline := k.CreatePipeline(s3, "User", pipelineSettings)
	for _, format := range []string{"json", ""} {
		changed := pipelineSettings
		changed.Format = format
		err := k.TryUpdatePipeline(pipeline, changed)
		assertSyntheticBadRequest(t, err)
		want := "Synthetic workspace supports only CSV source format"
		if format == "" {
			want = "Synthetic workspace requires CSV source format"
		}
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q rejection, got %v", want, err)
		}
	}
	run := k.StartPipelineRun(pipeline)
	k.WaitForRunsCompletion(run)
	identities, total := k.ConnectionIdentities(s3, 0, 100)
	if total != 20 {
		t.Fatalf("expected 20 pipeline identities, got %d", total)
	}
	ids := map[string]bool{}
	for _, row := range rows {
		ids[row["source_record_id"].(string)] = true
	}
	for _, identity := range identities {
		if !ids[identity.UserID] {
			t.Fatalf("expected source record ID, got %q", identity.UserID)
		}
	}
	k.SetWorkspaceID(normalID)
	k.Call("GET", "/v1/connectors", nil, nil, &catalogResponse)
	if len(catalogResponse.Connectors) <= 2 {
		t.Fatalf("expected restored normal catalog, got %d entries", len(catalogResponse.Connectors))
	}
	normalRows, _ = k.File(fs, "users.csv", "csv", "", krenalistester.NoCompression,
		krenalistester.JSONEncodeSettings(map[string]any{"separator": ",", "hasColumnNames": true}), 100)
	if len(normalRows) != 2 {
		t.Fatalf("expected 2 real FileSystem rows after Synthetic use, got %d", len(normalRows))
	}

	var organizationID string
	k.QueryRowTestDatabase(t.Context(), &organizationID, "SELECT organization FROM workspaces WHERE id = $1", created.ID)
	loaded := k.OpenCoreWithoutSynthetic(t.Context())
	defer loaded.Close(t.Context())
	organization, err := loaded.Organization(organizationID)
	if err != nil {
		t.Fatalf("expected persisted organization, got %v", err)
	}
	workspace, err := organization.Workspace(created.ID)
	if err != nil {
		t.Fatalf("expected persisted workspace, got %v", err)
	}
	if !workspace.Synthetic {
		t.Fatalf("expected reloaded Synthetic workspace, got normal")
	}
	loadedCatalog := loaded.Connectors(workspace)
	if len(loadedCatalog) != 2 {
		t.Fatalf("expected 2 reloaded Synthetic connectors, got %d", len(loadedCatalog))
	}
	for _, connector := range loadedCatalog {
		if connector.Code != "s3" && connector.Code != "csv" || connector.AsSource == nil || connector.AsDestination != nil {
			t.Fatalf("expected reloaded S3/CSV Source catalog, got %+v", connector)
		}
	}
	connection, err := workspace.Connection(t.Context(), s3)
	if err != nil {
		t.Fatalf("expected reloaded S3 connection, got %v", err)
	}
	_, _, _, err = connection.File(t.Context(), "customers-b.csv", "csv", "", core.NoCompression,
		krenalistester.JSONEncodeSettings(map[string]any{"separator": ",", "hasColumnNames": true}), 100)
	if err == nil {
		t.Fatal("expected unavailable Synthetic scenario, got successful S3 read")
	}
	if !strings.Contains(err.Error(), "Synthetic scenario is not available") {
		t.Fatalf("expected unavailable Synthetic scenario without S3 fallback, got %v", err)
	}

}

func TestSyntheticWorkspaceRequiresScenario(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()
	for _, endpoint := range []string{"/v1/workspaces/test", "/v1/workspaces"} {
		err := k.TryCall("POST", endpoint, http.Header{"Krenalis-Workspace": nil}, map[string]any{
			"synthetic": true,
		}, nil)
		assertSyntheticBadRequest(t, err)
		if !strings.Contains(err.Error(), "Synthetic scenario is not available") {
			t.Fatalf("expected unavailable scenario before warehouse access, got %v", err)
		}
	}
}

func mustNamespace(t *testing.T) fakedata.IdentityNamespace {
	t.Helper()
	namespace, err := fakedata.NewIdentityNamespace("workspace-test", 1)
	if err != nil {
		t.Fatalf("expected successful operation, got %v", err)
	}
	return namespace
}

func assertSyntheticBadRequest(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected HTTP 400, got success")
	}
	status, ok := err.(*krenalistester.StatusCodeError)
	if !ok {
		t.Fatalf("expected HTTP status error, got %T", err)
	}
	if status.Response.Code != http.StatusBadRequest {
		t.Fatalf("expected HTTP 400, got %d: %s", status.Response.Code, status.Response.Text)
	}
}
