// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
)

// Captured with the checkpoint type and serializer from bce4ff9588f5.
const legacyPhotoCheckpoint = `{"person":0,"representation":0,"catalogSHA256":"53550790e00f869a63ab5a82d52a5d0cae9308edc5dbfcdc13b3986dae82a884","worldModelVersion":"world-v1","marketDataVersion":"it-2026a","personSpecVersion":"person-v1","sourceVersion":"source-v1","namespaceVersion":1,"referenceDate":"2026-01-01","photoBaseURL":"https://krenalis.test/"}`

// TestSimulatedSystemExternalPhotoOrigins checks emitted URLs without fetching
// them.
func TestSimulatedSystemExternalPhotoOrigins(t *testing.T) {

	fixture, err := os.ReadFile("../tools/fakedata/testdata/external-photos-v1.json")
	if err != nil {
		t.Fatalf("expected external photo fixture, got %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(fixture)
	}))
	defer server.Close()
	source, err := fakedata.NewFaceCatalogSource([]string{server.URL})
	if err != nil {
		t.Fatalf("expected catalog source, got %v", err)
	}
	catalog, err := source.Load(t.Context())
	if err != nil {
		t.Fatalf("expected catalog, got %v", err)
	}
	bases := []string{
		"https://assets.krenalis.com/", "https://example.test/assets",
		"https://example.test/assets/", "http://internal-assets:8080/",
	}
	for _, base := range bases {

		account := simulatedSystemPlanFixture(t, catalog, "account", 1, "0")
		account.GenerationCheckpoint, err = initialSimulatedSystemCheckpoint(catalog, base)
		if err != nil {
			t.Fatalf("expected configured checkpoint, got %v", err)
		}
		plan, _, err := newSimulatedSystemPlan("workspace", account, catalog)
		if err != nil {
			t.Fatalf("expected configured plan, got %v", err)
		}
		record, err := plan.record(0, 0)
		if err != nil {
			t.Fatalf("expected generated record without image fetch, got %v", err)
		}
		var document simulatedSystemDocument
		err = json.Unmarshal(record.Data, &document)
		if err != nil {
			t.Fatalf("expected generated photo document, got %v", err)
		}
		truth, err := plan.world.Person(plan.personIndex(0))
		if err != nil {
			t.Fatalf("expected canonical photo ID, got %v", err)
		}
		want := strings.TrimSuffix(base, "/") + "/synthetic-world/photos/v1/256/" + truth.Person.PhotoID + ".webp"
		if document.Photo == nil || *document.Photo != want {
			t.Fatalf("expected URL %q, got %v", want, document.Photo)
		}

	}

}

// TestSimulatedSystemExternalPhotos checks fallback and external checkpoint
// replay.
func TestSimulatedSystemExternalPhotos(t *testing.T) {

	fixture, err := os.ReadFile("../tools/fakedata/testdata/external-photos-v1.json")
	if err != nil {
		t.Fatalf("expected external photo fixture, got %v", err)
	}
	var firstCalls, secondCalls atomic.Int32
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		firstCalls.Add(1)
		if r.URL.Path != "/primary/synthetic-world/photos/v1/catalog.json" {
			t.Errorf("expected catalog request only, got %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"format":1}`)) // A 200 with unusable metadata must fall through.
	}))
	defer first.Close()
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secondCalls.Add(1)
		if r.URL.Path != "/mirror/synthetic-world/photos/v1/catalog.json" {
			t.Errorf("expected catalog request only, got %s", r.URL.Path)
		}
		_, _ = w.Write(fixture)
	}))
	defer second.Close()
	origins := []string{first.URL + "/primary", second.URL + "/mirror/"}
	source, err := fakedata.NewFaceCatalogSource(origins)
	if err != nil {
		t.Fatalf("expected catalog source, got %v", err)
	}
	core := &Core{simulatedPhotoSource: source}
	catalog, err := core.loadSimulatedCatalog(t.Context())
	if err != nil {
		t.Fatalf("expected fallback catalog, got %v", err)
	}
	account := simulatedSystemPlanFixture(t, catalog, "external-account", 20, "50")
	account.GenerationCheckpoint, err = initialSimulatedSystemCheckpoint(catalog, origins[0])
	if err != nil {
		t.Fatalf("expected external checkpoint, got %v", err)
	}
	plan, checkpoint, err := newSimulatedSystemPlan("workspace-a", account, catalog)
	if err != nil {
		t.Fatalf("expected external generation plan, got %v", err)
	}
	if checkpoint.CatalogRelease != "v1" || checkpoint.PhotoBaseURL != origins[0] {
		t.Fatalf("expected v1 and first-origin checkpoint, got %#v", checkpoint)
	}
	baseline, oracle := simulatedSystemPlanRecords(t, plan, checkpoint, 4)
	for _, record := range baseline {
		var document simulatedSystemDocument
		err = json.Unmarshal(record.Data, &document)
		if err != nil {
			t.Fatalf("expected generated document, got %v", err)
		}
		if document.Photo == nil || !strings.HasPrefix(*document.Photo, origins[0]+"/synthetic-world/photos/v1/256/") ||
			!strings.HasSuffix(*document.Photo, ".webp") {
			t.Fatalf("expected absolute first-origin photo URL, got %v", document.Photo)
		}
	}
	for i := 0; i < 7; i++ {
		checkpoint = plan.advance(checkpoint)
	}
	account.GeneratedRecordCount = 7
	account.GenerationCheckpoint, err = json.Marshal(checkpoint)
	if err != nil {
		t.Fatalf("expected progress checkpoint, got %v", err)
	}

	// A restarted process reloads the same release, retaining exact replay.
	restarted, err := fakedata.NewFaceCatalogSource(origins)
	if err != nil {
		t.Fatalf("expected restarted source, got %v", err)
	}
	reloaded, err := restarted.Load(t.Context())
	if err != nil {
		t.Fatalf("expected same release after restart, got %v", err)
	}
	resumed, next, err := newSimulatedSystemPlan("workspace-a", account, reloaded)
	if err != nil {
		t.Fatalf("expected resumed plan, got %v", err)
	}
	records, resumedOracle := simulatedSystemPlanRecords(t, resumed, next, 3)
	if !reflect.DeepEqual(records, baseline[7:]) || !reflect.DeepEqual(resumedOracle, oracle[7:]) {
		t.Fatal("expected identical resumed suffix, got different records")
	}
	if firstCalls.Load() != 2 || secondCalls.Load() != 2 {
		t.Fatalf("expected two catalog fetches per origin and no image fetches, got %d/%d",
			firstCalls.Load(), secondCalls.Load())
	}

	account.GenerationCheckpoint = json.Value(legacyPhotoCheckpoint)
	_, _, err = newSimulatedSystemPlan("workspace-a", account, catalog)
	if err == nil {
		t.Fatal("expected old serialized checkpoint to reject an external catalog, got success")
	}

	for _, release := range []string{"", "v2"} {
		changed := checkpoint
		changed.CatalogRelease = release
		account.GenerationCheckpoint, err = json.Marshal(changed)
		if err != nil {
			t.Fatalf("expected changed checkpoint, got %v", err)
		}
		_, _, err = newSimulatedSystemPlan("workspace-a", account, catalog)
		if err != nil {
			continue
		}
		t.Fatalf("expected release mismatch %q to fail, got success", release)
	}

}

// TestSimulatedSystemLegacyPhotoCheckpoint replays committed checkpoint JSON.
func TestSimulatedSystemLegacyPhotoCheckpoint(t *testing.T) {

	catalog := simulatedSystemTestCatalog(t)
	account := simulatedSystemPlanFixture(t, catalog, "legacy-account", 20, "50")
	account.GenerationCheckpoint = json.Value(legacyPhotoCheckpoint)
	plan, checkpoint, err := newSimulatedSystemPlan("workspace-a", account, catalog)
	if err != nil {
		t.Fatalf("expected committed legacy checkpoint to load, got %v", err)
	}
	if checkpoint.CatalogRelease != "" {
		t.Fatalf("expected local checkpoint, got release %q", checkpoint.CatalogRelease)
	}
	records, oracle := simulatedSystemPlanRecords(t, plan, checkpoint, 4)
	for _, record := range records {
		var document simulatedSystemDocument
		err = json.Unmarshal(record.Data, &document)
		if err != nil {
			t.Fatalf("expected legacy document, got %v", err)
		}
		if document.Photo == nil || !strings.HasPrefix(*document.Photo,
			"https://krenalis.test/photos/53550790e00f869a63ab5a82d52a5d0cae9308edc5dbfcdc13b3986dae82a884/256/") {
			t.Fatalf("expected legacy photo namespace, got %v", document.Photo)
		}
	}

	// Resume an old-format persisted coordinate with a freshly loaded catalog.
	account.GenerationCheckpoint = json.Value(strings.Replace(legacyPhotoCheckpoint, `"person":0`, `"person":2`, 1))
	account.GeneratedRecordCount = 4
	resumed, next, err := newSimulatedSystemPlan("workspace-a", account, simulatedSystemTestCatalog(t))
	if err != nil {
		t.Fatalf("expected legacy restart, got %v", err)
	}
	suffix, suffixOracle := simulatedSystemPlanRecords(t, resumed, next, 3)
	if !reflect.DeepEqual(suffix, records[4:]) || !reflect.DeepEqual(suffixOracle, oracle[4:]) {
		t.Fatal("expected identical legacy replay including photo URLs, got different records")
	}

}
