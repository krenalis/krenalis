// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/krenalis/krenalis/connectors/s3"
	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

func TestSyntheticABIdentityResolution(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	photoDir := syntheticCatalogFixture(t)
	canonicalizeSyntheticCatalogFixture(t, photoDir)
	catalogBytes, err := os.ReadFile(filepath.Join(photoDir, "catalog.json"))
	if err != nil {
		t.Fatalf("expected successful operation, got %v", err)
	}
	catalogSHA256 := sha256.Sum256(catalogBytes)
	catalogSHA256Text := hex.EncodeToString(catalogSHA256[:])
	catalog, err := fakedata.LoadFaceCatalog(t.Context(), photoDir)
	if err != nil {
		t.Fatalf("expected successful operation, got %v", err)
	}
	k := krenalistester.NewKrenalisInstance(t)
	config := core.SyntheticConfig{
		Namespace: "workspace-test", Generation: 1, Seed: 726381, ReferenceDate: "2026-01-01", PersonCount: 20,
		PhotoOrigin: "http://" + k.Addr(), SourceAID: "a", SourceAVersion: "demo-v1",
		SourceACoverageNumerator: 1, SourceACoverageDenominator: 1,
		SourceADuplicateNumerator: 1, SourceADuplicateDenominator: 1,
		SourceBID: "b", SourceBVersion: "demo-v1", SourceBCoverageNumerator: 1,
		SourceBCoverageDenominator: 1, SourceBDuplicateNumerator: 0, SourceBDuplicateDenominator: 1,
	}
	k.SetSyntheticPhotosDir(photoDir)
	k.SetSyntheticConfig(&config)
	k.Start()
	defer k.Stop()

	settings := &krenalistester.DBSettings{}
	err = json.Unmarshal(krenalistester.PostgresWarehouseSettings(), settings)
	if err != nil {
		t.Fatalf("expected successful operation, got %v", err)
	}
	pool, err := krenalistester.ConnectionPool(t.Context(), settings)
	if err != nil {
		t.Fatalf("expected successful operation, got %v", err)
	}
	_, err = pool.Exec(t.Context(), "CREATE DATABASE test_synthetic_ab_profiles")
	pool.Close()
	if err != nil {
		t.Fatalf("expected successful operation, got %v", err)
	}
	settings.Database = "test_synthetic_ab_profiles"

	profileSchema := types.Object([]types.Property{
		{Name: "first_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
		{Name: "last_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
		{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
		{Name: "phone", Type: types.String().WithMaxLength(300), ReadOptional: true},
		{Name: "photo_url", Type: types.String().WithMaxLength(2048), ReadOptional: true},
	})
	request := map[string]any{
		"name": "synthetic-ab", "synthetic": true, "profileSchema": profileSchema,
		"warehouse": map[string]any{
			"platform": "PostgreSQL", "mode": "Normal", "settings": settings,
		},
		"uiPreferences": map[string]any{"profile": map[string]string{
			"image": "photo_url", "firstName": "first_name", "lastName": "last_name", "extra": "email",
		}},
	}
	var workspace struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/workspaces", http.Header{"Krenalis-Workspace": nil}, request, &workspace)
	k.SetWorkspaceID(workspace.ID)

	k.UpdateIdentityResolutionSettings(false, []string{"email"})
	settingsValue := krenalistester.JSONEncodeSettings(map[string]any{
		"accessKeyID": "AAAAAAAAAAAAAAAAAAAA", "secretAccessKey": "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB",
		"region": "us-east-1", "bucket": "demo",
	})
	connectionA := k.CreateConnection(krenalistester.ConnectionToCreate{Name: "Synthetic A", Role: krenalistester.Source,
		Connector: "s3", Settings: settingsValue})
	connectionB := k.CreateConnection(krenalistester.ConnectionToCreate{Name: "Synthetic B", Role: krenalistester.Source,
		Connector: "s3", Settings: settingsValue})

	k.AlterProfileSchemaAndWait(profileSchema, map[string]string{
		"first_name": connectionB, "last_name": connectionB, "email": connectionB,
		"phone": connectionB, "photo_url": connectionB,
	}, nil)

	namespace, err := fakedata.NewIdentityNamespace(config.Namespace, config.Generation)
	if err != nil {
		t.Fatalf("expected successful operation, got %v", err)
	}
	base, err := fakedata.NewWorld(fakedata.WorldConfig{IdentityNamespace: namespace, WorldSeed: config.Seed,
		ReferenceDate: config.ReferenceDate, FaceCatalog: catalog})
	if err != nil {
		t.Fatalf("expected successful operation, got %v", err)
	}
	sourceWorld, err := fakedata.NewSourceWorld(base, []fakedata.CountryShare{{Code: "IT", Version: fakedata.MarketDataVersion, Weight: 1}})
	if err != nil {
		t.Fatalf("expected successful operation, got %v", err)
	}
	sources := []struct {
		connection string
		pipeline   string
		name       string
		sourceID   string
		instance   *fakedata.SourceInstance
	}{
		{connection: connectionA, name: "customers-a.csv", sourceID: "a"},
		{connection: connectionB, name: "customers-b.csv", sourceID: "b"},
	}
	instances := []fakedata.SourceInstanceConfig{
		{ID: "a", Version: "demo-v1", CoverageNumerator: 1, CoverageDenominator: 1,
			DuplicateNumerator: 1, DuplicateDenominator: 1},
		{ID: "b", Version: "demo-v1", CoverageNumerator: 1, CoverageDenominator: 1,
			DuplicateNumerator: 0, DuplicateDenominator: 1},
	}
	for i := range sources {
		sources[i].instance, err = fakedata.NewSourceInstance(sourceWorld, instances[i])
		if err != nil {
			t.Fatalf("expected successful operation, got %v", err)
		}
	}

	type expectedIdentity struct {
		record fakedata.SourceRecord
	}
	expected := map[string]expectedIdentity{}
	expectedGroups := map[string]map[string]struct{}{}
	oracleBySourceRecord := map[string]string{}
	for i, source := range sources {
		for index := fakedata.PersonIndex(1); index <= config.PersonCount; index++ {
			records, err := source.instance.Records(index)
			if err != nil {
				t.Fatalf("expected successful operation, got %v", err)
			}
			oracleRecords, err := sourceWorld.Oracle(source.instance, index)
			if err != nil {
				t.Fatalf("expected successful operation, got %v", err)
			}
			if len(records) != len(oracleRecords) {
				t.Fatalf("expected Oracle coverage for %s person %d, got %d records and %d mappings", instances[i].ID, index, len(records), len(oracleRecords))
			}
			for _, record := range records {
				key := source.connection + "\x00" + record.ID
				if _, ok := expected[key]; ok {
					t.Fatalf("expected unique identity key, got %q twice", key)
				}
				expected[key] = expectedIdentity{record: record}
				group := key
				if record.Email != nil {
					group = "email\x00" + *record.Email
				}
				if expectedGroups[group] == nil {
					expectedGroups[group] = map[string]struct{}{}
				}
				expectedGroups[group][key] = struct{}{}
			}
			for _, oracleRecord := range oracleRecords {
				if oracleRecord.SourceInstanceID != source.sourceID {
					t.Fatalf("expected Oracle source %q, got %q", source.sourceID, oracleRecord.SourceInstanceID)
				}
				if oracleRecord.SourceRecordID == "" {
					t.Fatal("expected non-empty Oracle source record ID")
				}
				if oracleRecord.PersonID == "" {
					t.Fatal("expected non-empty Oracle person ID")
				}
				key := oracleRecord.SourceInstanceID + "\x00" + oracleRecord.SourceRecordID
				if _, ok := oracleBySourceRecord[key]; ok {
					t.Fatalf("expected unique Oracle key, got %q twice", key)
				}
				oracleBySourceRecord[key] = oracleRecord.PersonID
			}
		}
	}
	connectionBySourceID := map[string]string{"a": connectionA, "b": connectionB}
	oracle := map[string]string{}
	for sourceRecordKey, personID := range oracleBySourceRecord {
		sourceID, recordID, ok := strings.Cut(sourceRecordKey, "\x00")
		if !ok {
			t.Fatalf("expected Oracle key with source and record ID, got %q", sourceRecordKey)
		}
		connection, ok := connectionBySourceID[sourceID]
		if !ok {
			t.Fatalf("expected connection for Oracle source %q", sourceID)
		}
		key := connection + "\x00" + recordID
		if _, ok := oracle[key]; ok {
			t.Fatalf("expected unique Oracle connection key, got %q twice", key)
		}
		oracle[key] = personID
	}
	if len(expected) != 60 || len(oracle) != len(expected) {
		t.Fatalf("expected 60 identities and complete Oracle, got %d identities and %d mappings", len(expected), len(oracle))
	}
	for key := range expected {
		if _, ok := oracle[key]; !ok {
			t.Fatalf("expected Oracle mapping for observed key %q", key)
		}
	}
	for key := range oracle {
		if _, ok := expected[key]; !ok {
			t.Fatalf("expected Oracle key to be observed, got %q", key)
		}
	}
	personIDs := map[string]struct{}{}
	for _, personID := range oracle {
		personIDs[personID] = struct{}{}
	}
	if len(personIDs) != 20 {
		t.Fatalf("expected 20 Oracle persons, got %d", len(personIDs))
	}

	makePipeline := func(source struct {
		connection string
		pipeline   string
		name       string
		sourceID   string
		instance   *fakedata.SourceInstance
	}) string {
		input := types.Object([]types.Property{
			{Name: "source_record_id", Type: types.String()}, {Name: "first_name", Type: types.String()},
			{Name: "last_name", Type: types.String()}, {Name: "email", Type: types.String()},
			{Name: "phone", Type: types.String()}, {Name: "photo_url", Type: types.String()},
		})
		return k.CreatePipeline(source.connection, "User", krenalistester.PipelineToSet{
			Name: "Import " + source.name, Enabled: true, Path: source.name, Format: "csv",
			UserIDColumn: "source_record_id", InSchema: input, OutSchema: profileSchema,
			Transformation: &krenalistester.Transformation{Mapping: map[string]string{
				"first_name": "first_name", "last_name": "last_name",
				"email": "if(eq(email, ''), null, email)", "phone": "phone", "photo_url": "photo_url",
			}}, FormatSettings: krenalistester.JSONEncodeSettings(map[string]any{
				"separator": ",", "hasColumnNames": true,
			}),
		})
	}
	for i := range sources {
		sources[i].pipeline = makePipeline(sources[i])
	}
	runs := []string{k.StartPipelineRun(sources[0].pipeline), k.StartPipelineRun(sources[1].pipeline)}
	k.WaitForRunsCompletion(runs...)
	for _, runID := range runs {
		run := k.PipelineRun(runID)
		if run.Error != "" || run.Failed != [7]int{} {
			t.Fatalf("expected successful pipeline run, got %+v", run)
		}
	}

	for _, source := range sources {
		identities, total := k.ConnectionIdentities(source.connection, 0, 100)
		want := 40
		if source.name == "customers-b.csv" {
			want = 20
		}
		if total != want || len(identities) != want {
			t.Fatalf("expected %d identities for %s, got total %d and %d rows", want, source.name, total, len(identities))
		}
		for _, identity := range identities {
			if identity.Connection != source.connection || identity.Pipeline != source.pipeline {
				t.Fatalf("expected identity from connection %s and pipeline %s, got %+v", source.connection, source.pipeline, identity)
			}
			key := source.connection + "\x00" + identity.UserID
			if _, ok := expected[key]; !ok {
				t.Fatalf("expected observed source record %q", key)
			}
		}
	}

	k.RunIdentityResolutionAndWait()
	profiles, _, total := k.Profiles([]string{"first_name", "last_name", "email", "phone", "photo_url"}, "", false, 0, 100)
	if total != len(expectedGroups) || len(profiles) != total {
		t.Fatalf("expected %d profiles from observed email partition, got total %d and %d rows", len(expectedGroups), total, len(profiles))
	}
	actualGroups := map[string]map[string]struct{}{}
	identityProfile := map[string]string{}
	profileRecords := map[string][]expectedIdentity{}
	for _, profile := range profiles {
		identities, identityTotal := k.Identities(profile.KPID, 0, 100)
		if identityTotal != len(identities) || identityTotal == 0 {
			t.Fatalf("expected complete identities for profile %s, got total %d and %d rows", profile.KPID, identityTotal, len(identities))
		}
		group := map[string]struct{}{}
		for _, identity := range identities {
			key := identity.Connection + "\x00" + identity.UserID
			value, ok := expected[key]
			if !ok {
				t.Fatalf("expected profile identity in Oracle coverage, got %q", key)
			}
			if _, ok := identityProfile[key]; ok {
				t.Fatalf("expected identity %q in one profile, got duplicate", key)
			}
			identityProfile[key] = profile.KPID.String()
			group[key] = struct{}{}
			profileRecords[profile.KPID.String()] = append(profileRecords[profile.KPID.String()], value)
		}
		actualGroups[profile.KPID.String()] = group
	}
	if len(identityProfile) != len(expected) {
		t.Fatalf("expected all 60 identities in profiles, got %d", len(identityProfile))
	}
	for groupName, want := range expectedGroups {
		found := false
		for _, got := range actualGroups {
			if sameIdentitySet(want, got) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected profile partition for %q, got no matching profile", groupName)
		}
	}

	correct, incorrect, missed := 0, 0, 0
	keys := make([]string, 0, len(expected))
	for key := range expected {
		keys = append(keys, key)
	}
	for i, left := range keys {
		for _, right := range keys[i+1:] {
			sameProfile := identityProfile[left] == identityProfile[right]
			samePerson := oracle[left] == oracle[right]
			switch {
			case sameProfile && samePerson:
				correct++
			case sameProfile:
				incorrect++
			case samePerson:
				missed++
			}
		}
	}
	t.Logf("identity resolution pairs: correct=%d incorrect=%d missed=%d; Oracle persons=%d identities=%d profiles=%d", correct, incorrect, missed, len(personIDs), len(expected), total)

	for profileID, records := range profileRecords {
		profile := findProfile(profiles, profileID)
		for _, field := range []string{"first_name", "last_name", "email", "phone", "photo_url"} {
			values := map[string]struct{}{}
			for _, item := range records {
				value := observedValue(item.record, field, catalog, config.PhotoOrigin)
				if value != "" {
					values[value] = struct{}{}
				}
			}
			actual, present := profile.Attributes[field]
			if len(values) == 0 {
				if present && actual != nil && actual != "" {
					t.Fatalf("expected absent %s for profile %s, got %v", field, profileID, actual)
				}
				continue
			}
			value, ok := actual.(string)
			if !present || !ok {
				t.Fatalf("expected observed %s for profile %s, got %v", field, profileID, actual)
			}
			if _, ok := values[value]; !ok {
				t.Fatalf("expected observed %s value for profile %s, got %q", field, profileID, value)
			}
		}
		allAbsentEmail := true
		for _, item := range records {
			if item.record.Email != nil {
				allAbsentEmail = false
				break
			}
		}
		if allAbsentEmail {
			if _, ok := profile.Attributes["email"]; ok {
				t.Fatalf("expected no reconstructed email for profile %s, got %v", profileID, profile.Attributes["email"])
			}
		}
	}

	t.Logf("synthetic snapshot: catalog_sha256=%s catalog_checksum=%x snapshot_id=%s workspace_id=%s expected_profiles=%d api_profiles=%d", catalogSHA256Text, catalog.Checksum(), base.SnapshotID(), workspace.ID, len(expectedGroups), total)
}

func canonicalizeSyntheticCatalogFixture(t *testing.T, directory string) {
	t.Helper()
	for _, name := range []string{"catalog.json", "specs.json", "manifest.json"} {
		path := filepath.Join(directory, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected successful operation, got %v", err)
		}
		data, err = json.Canonicalize(data)
		if err != nil {
			t.Fatalf("expected successful operation, got %v", err)
		}
		err = os.WriteFile(path, data, 0o644)
		if err != nil {
			t.Fatalf("expected successful operation, got %v", err)
		}
	}
}

func findProfile(profiles []krenalistester.Profile, id string) krenalistester.Profile {
	for _, profile := range profiles {
		if profile.KPID.String() == id {
			return profile
		}
	}
	panic(fmt.Sprintf("profile %s not found", id))
}

func observedValue(record fakedata.SourceRecord, field string, catalog *fakedata.FaceCatalog, origin string) string {
	switch field {
	case "first_name":
		return sourceString(record.FirstName)
	case "last_name":
		return sourceString(record.LastName)
	case "email":
		return sourceString(record.Email)
	case "phone":
		return sourceString(record.Phone)
	case "photo_url":
		if record.PhotoID == nil {
			return ""
		}
		asset, err := catalog.Asset(*record.PhotoID, fakedata.PhotoSize256)
		if err != nil {
			panic(err)
		}
		return strings.TrimSuffix(origin, "/") + asset.Path
	default:
		panic(fmt.Sprintf("unknown observed field %q", field))
	}
}

func sameIdentitySet(left, right map[string]struct{}) bool {
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

func sourceString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
