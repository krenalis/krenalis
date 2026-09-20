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
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestSyntheticABIdentityResolution verifies the complete ordinary FileSystem
// import path for the two Synthetic World source outputs.
func TestSyntheticABIdentityResolution(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	photoDirectory := syntheticCatalogFixture(t)
	canonicalizeSyntheticCatalogFixture(t, photoDirectory)
	catalog, err := fakedata.LoadFaceCatalog(t.Context(), photoDirectory)
	if err != nil {
		t.Fatalf("expected complete face catalog, got %v", err)
	}

	namespace, err := fakedata.NewIdentityNamespace("workspace-test", 1)
	if err != nil {
		t.Fatalf("expected identity namespace, got %v", err)
	}
	world, err := fakedata.NewWorld(fakedata.WorldConfig{
		IdentityNamespace: namespace,
		WorldSeed:         726381,
		ReferenceDate:     "2026-01-01",
		FaceCatalog:       catalog,
	})
	if err != nil {
		t.Fatalf("expected source world base, got %v", err)
	}
	sourceWorld, err := fakedata.NewSourceWorld(world, []fakedata.CountryShare{{
		Code: "IT", Version: fakedata.MarketDataVersion, Weight: 1,
	}})
	if err != nil {
		t.Fatalf("expected source world, got %v", err)
	}

	sources := []syntheticSourceFixture{
		{name: "customers-a.csv", sourceID: "a", config: fakedata.SourceInstanceConfig{
			ID: "a", Version: "demo-v1", CoverageNumerator: 1, CoverageDenominator: 1,
			DuplicateNumerator: 1, DuplicateDenominator: 1,
		}},
		{name: "customers-b.csv", sourceID: "b", config: fakedata.SourceInstanceConfig{
			ID: "b", Version: "demo-v1", CoverageNumerator: 1, CoverageDenominator: 1,
			DuplicateNumerator: 0, DuplicateDenominator: 1,
		}},
	}
	storageDirectory := t.TempDir()
	for index := range sources {
		sources[index].instance, err = fakedata.NewSourceInstance(sourceWorld, sources[index].config)
		if err != nil {
			t.Fatalf("expected source instance %s, got %v", sources[index].sourceID, err)
		}
		sources[index].records = writeSyntheticSourceCSV(t, filepath.Join(storageDirectory, sources[index].name),
			sources[index].instance, catalog, 20)
	}

	k := krenalistester.NewKrenalisInstance(t)
	k.PopulateProfileSchema(false)
	k.SetFileSystemRoot(storageDirectory)
	k.SetSyntheticPhotosDir(photoDirectory)
	k.Start()
	defer k.Stop()

	k.UpdateIdentityResolutionSettings(false, []string{"email"})
	for index := range sources {
		sources[index].connection = k.CreateSourceFileSystem()
	}

	properties := k.Workspace().ProfileSchema.Properties().Slice()
	properties = append(properties, []types.Property{
		{Name: "first_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
		{Name: "last_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
		{Name: "phone", Type: types.String().WithMaxLength(300), ReadOptional: true},
		{Name: "photo_url", Type: types.String().AsURL(), ReadOptional: true},
		{Name: "country", Type: types.String().AsCountry(types.ISO3166Alpha2), ReadOptional: true},
	}...)
	profileSchema := types.Object(properties)
	assignedRoles := krenalistester.ProfileRoleAssignments{
		FirstName: "first_name", LastName: "last_name", Country: "country", Photo: "photo_url",
	}
	primarySources := map[string]string{}
	for _, property := range profileSchema.Properties().Slice() {
		primarySources[property.Name] = sources[1].connection
	}
	k.AlterProfileSchemaWithAssignedRolesAndWait(profileSchema, assignedRoles, primarySources, nil)

	for index := range sources {
		sources[index].pipeline = k.CreatePipeline(sources[index].connection, "User", krenalistester.PipelineToSet{
			Name:         "Import " + sources[index].name,
			Enabled:      true,
			Path:         sources[index].name,
			Format:       "csv",
			UserIDColumn: "source_record_id",
			InSchema: types.Object([]types.Property{
				{Name: "source_record_id", Type: types.String()},
				{Name: "first_name", Type: types.String()},
				{Name: "last_name", Type: types.String()},
				{Name: "email", Type: types.String()},
				{Name: "phone", Type: types.String()},
				{Name: "photo_url", Type: types.String()},
				{Name: "country", Type: types.String()},
			}),
			OutSchema: profileSchema,
			Transformation: &krenalistester.Transformation{Mapping: map[string]string{
				"first_name": "first_name",
				"last_name":  "last_name",
				"email":      "if(eq(email, ''), null, email)",
				"phone":      "phone",
				"photo_url":  "photo_url",
				"country":    "if(eq(country, ''), null, country)",
			}},
			FormatSettings: krenalistester.JSONEncodeSettings(map[string]any{
				"separator": ",", "hasColumnNames": true,
			}),
		})
	}

	runs := []string{k.StartPipelineRun(sources[0].pipeline), k.StartPipelineRun(sources[1].pipeline)}
	k.WaitForRunsCompletion(runs...)
	for _, runID := range runs {
		run := k.PipelineRun(runID)
		if run.Error != "" || run.Failed != [7]int{} {
			t.Fatalf("expected successful pipeline run, got %+v", run)
		}
	}

	expected, expectedGroups := expectedSyntheticIdentities(sources)
	for _, source := range sources {
		identities, total := k.ConnectionIdentities(source.connection, 0, 100)
		if total != len(source.records) || len(identities) != len(source.records) {
			t.Fatalf("expected %d identities for %s, got total %d and %d rows", len(source.records), source.name, total, len(identities))
		}
		for _, identity := range identities {
			key := source.connection + "\x00" + identity.UserID
			if _, ok := expected[key]; !ok {
				t.Fatalf("expected identity from generated source records, got %q", key)
			}
			if identity.Pipeline != source.pipeline {
				t.Fatalf("expected pipeline %s for %q, got %s", source.pipeline, key, identity.Pipeline)
			}
		}
	}

	k.RunIdentityResolutionAndWait()
	profiles, total := k.Profiles([]string{"first_name", "last_name", "email", "phone", "photo_url", "country"}, "", false, 0, 100)
	if total != len(expectedGroups) || len(profiles) != total {
		t.Fatalf("expected %d profiles from observed email partition, got total %d and %d rows", len(expectedGroups), total, len(profiles))
	}

	actualGroups := map[string]map[string]struct{}{}
	profileRecords := map[string][]fakedata.SourceRecord{}
	identityProfile := map[string]string{}
	for _, profile := range profiles {
		identities, identityTotal := k.Identities(profile.KPID, 0, 100)
		if identityTotal != len(identities) || identityTotal == 0 {
			t.Fatalf("expected complete non-empty identities for profile %s, got total %d and %d rows", profile.KPID, identityTotal, len(identities))
		}
		group := map[string]struct{}{}
		for _, identity := range identities {
			key := identity.Connection + "\x00" + identity.UserID
			item, ok := expected[key]
			if !ok {
				t.Fatalf("expected profile identity in generated source records, got %q", key)
			}
			if _, ok := identityProfile[key]; ok {
				t.Fatalf("expected identity %q in one profile, got duplicate", key)
			}
			identityProfile[key] = profile.KPID.String()
			group[key] = struct{}{}
			profileRecords[profile.KPID.String()] = append(profileRecords[profile.KPID.String()], item)
		}
		actualGroups[profile.KPID.String()] = group
	}
	if len(identityProfile) != len(expected) {
		t.Fatalf("expected all %d identities in profiles, got %d", len(expected), len(identityProfile))
	}
	for name, want := range expectedGroups {
		matched := false
		for _, got := range actualGroups {
			if sameSyntheticIdentitySet(want, got) {
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("expected profile partition for %q, got no matching profile", name)
		}
	}

	for profileID, records := range profileRecords {
		profile := syntheticProfileByID(t, profiles, profileID)
		for _, field := range []string{"first_name", "last_name", "email", "phone", "photo_url", "country"} {
			values := map[string]struct{}{}
			for _, record := range records {
				if value := observedSyntheticValue(t, record, field, catalog); value != "" {
					values[value] = struct{}{}
				}
			}
			actual, exists := profile.Attributes[field]
			if len(values) == 0 {
				if exists && actual != nil && actual != "" {
					t.Fatalf("expected absent %s for profile %s, got %v", field, profileID, actual)
				}
				continue
			}
			value, ok := actual.(string)
			if !exists || !ok {
				t.Fatalf("expected observed %s for profile %s, got %v", field, profileID, actual)
			}
			if _, ok := values[value]; !ok {
				t.Fatalf("expected observed %s value for profile %s, got %q", field, profileID, value)
			}
		}
	}

	verifySyntheticPhotoServed(t, k, sources, catalog)
}

type syntheticSourceFixture struct {
	name       string
	sourceID   string
	config     fakedata.SourceInstanceConfig
	instance   *fakedata.SourceInstance
	records    []fakedata.SourceRecord
	connection string
	pipeline   string
}

func canonicalizeSyntheticCatalogFixture(t *testing.T, directory string) {
	t.Helper()
	for _, name := range []string{"catalog.json", "specs.json", "manifest.json"} {
		path := filepath.Join(directory, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected catalog metadata %s, got %v", name, err)
		}
		data, err = json.Canonicalize(data)
		if err != nil {
			t.Fatalf("expected canonical %s metadata, got %v", name, err)
		}
		err = os.WriteFile(path, data, 0o644)
		if err != nil {
			t.Fatalf("expected canonicalized %s metadata, got %v", name, err)
		}
	}
}

func expectedSyntheticIdentities(sources []syntheticSourceFixture) (map[string]fakedata.SourceRecord, map[string]map[string]struct{}) {
	expected := map[string]fakedata.SourceRecord{}
	groups := map[string]map[string]struct{}{}
	for _, source := range sources {
		for _, record := range source.records {
			key := source.connection + "\x00" + record.ID
			expected[key] = record
			group := key
			if record.Email != nil {
				group = "email\x00" + *record.Email
			}
			if groups[group] == nil {
				groups[group] = map[string]struct{}{}
			}
			groups[group][key] = struct{}{}
		}
	}
	return expected, groups
}

func observedSyntheticValue(t *testing.T, record fakedata.SourceRecord, field string, catalog *fakedata.FaceCatalog) string {
	t.Helper()
	switch field {
	case "first_name":
		return syntheticString(record.FirstName)
	case "last_name":
		return syntheticString(record.LastName)
	case "email":
		return syntheticString(record.Email)
	case "phone":
		return syntheticString(record.Phone)
	case "photo_url":
		if record.PhotoID == nil {
			return ""
		}
		asset, err := catalog.Asset(*record.PhotoID, fakedata.PhotoSize256)
		if err != nil {
			t.Fatalf("expected observed photo asset, got %v", err)
		}
		return asset.Path
	case "country":
		return syntheticString(record.Country)
	}
	t.Fatalf("expected known source field, got %q", field)
	return ""
}

func sameSyntheticIdentitySet(left, right map[string]struct{}) bool {
	if len(left) != len(right) {
		return false
	}
	for key := range left {
		if _, ok := right[key]; !ok {
			return false
		}
	}
	return true
}

func syntheticCatalogFixture(t *testing.T) string {
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
		err = os.WriteFile(filepath.Join(directory, item.name), data, 0o644)
		if err != nil {
			t.Fatalf("expected face fixture metadata %s, got %v", item.name, err)
		}
	}
	return directory
}

func syntheticProfileByID(t *testing.T, profiles []krenalistester.Profile, id string) krenalistester.Profile {
	t.Helper()
	for _, profile := range profiles {
		if profile.KPID.String() == id {
			return profile
		}
	}
	t.Fatalf("expected profile %s, got none", id)
	return krenalistester.Profile{}
}

func syntheticString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func verifySyntheticPhotoServed(t *testing.T, k *krenalistester.Krenalis, sources []syntheticSourceFixture, catalog *fakedata.FaceCatalog) {
	t.Helper()
	for _, source := range sources {
		for _, record := range source.records {
			if record.PhotoID == nil {
				continue
			}
			asset, err := catalog.Asset(*record.PhotoID, fakedata.PhotoSize256)
			if err != nil {
				t.Fatalf("expected photo asset, got %v", err)
			}
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+k.Addr()+asset.Path, nil)
			if err != nil {
				t.Fatalf("expected photo request, got %v", err)
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("expected served photo, got %v", err)
			}
			body, readErr := io.ReadAll(res.Body)
			closeErr := res.Body.Close()
			if readErr != nil {
				t.Fatalf("expected photo body, got %v", readErr)
			}
			if closeErr != nil {
				t.Fatalf("expected closed photo body, got %v", closeErr)
			}
			if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "image/webp" {
				t.Fatalf("expected served WebP photo, got status %d and type %q", res.StatusCode, res.Header.Get("Content-Type"))
			}
			digest := sha256.Sum256(body)
			if got := hex.EncodeToString(digest[:]); got != asset.ContentSHA256 {
				t.Fatalf("expected photo checksum %s, got %s", asset.ContentSHA256, got)
			}
			return
		}
	}
	t.Fatal("expected at least one observed photo")
}

func writeSyntheticSourceCSV(t *testing.T, path string, instance *fakedata.SourceInstance, catalog *fakedata.FaceCatalog, personCount int) []fakedata.SourceRecord {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("expected CSV destination %s, got %v", path, err)
	}
	writer, err := fakedata.NewSourceCSVWriter(file, catalog)
	if err != nil {
		_ = file.Close()
		t.Fatalf("expected source CSV writer, got %v", err)
	}
	records := []fakedata.SourceRecord{}
	for index := fakedata.PersonIndex(1); index <= fakedata.PersonIndex(personCount); index++ {
		generated, err := instance.Records(index)
		if err != nil {
			_ = file.Close()
			t.Fatalf("expected source records for person %d, got %v", index, err)
		}
		for _, record := range generated {
			err = writer.Write(t.Context(), record)
			if err != nil {
				_ = file.Close()
				t.Fatalf("expected CSV record %s, got %v", record.ID, err)
			}
			records = append(records, record)
		}
	}
	err = writer.Flush(t.Context())
	if err != nil {
		_ = file.Close()
		t.Fatalf("expected flushed source CSV, got %v", err)
	}
	err = file.Close()
	if err != nil {
		t.Fatalf("expected closed source CSV, got %v", err)
	}
	return records
}
