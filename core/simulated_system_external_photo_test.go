// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/krenalis/krenalis/test/syntheticworldtest"
	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
)

// Captured with the checkpoint type and serializer from bce4ff9588f5.
const legacyPhotoCheckpoint = `{"person":0,"representation":0,"catalogSHA256":"53550790e00f869a63ab5a82d52a5d0cae9308edc5dbfcdc13b3986dae82a884","worldModelVersion":"world-v1","marketDataVersion":"it-2026a","personSpecVersion":"person-v1","sourceVersion":"source-v1","namespaceVersion":1,"referenceDate":"2026-01-01","photoBaseURL":"https://krenalis.test/"}`

// TestSimulatedSystemExternalPhotoOrigins checks emitted URLs without fetching
// them.
func TestSimulatedSystemExternalPhotoOrigins(t *testing.T) {

	data := simulatedSystemTestData(t)
	bases := []string{
		"https://assets.krenalis.com/", "https://example.test/assets",
		"https://example.test/assets/", "http://internal-assets:8080/",
	}
	for _, base := range bases {

		account := simulatedSystemPlanFixture(t, data, "account", 1, "0")
		checkpoint, err := initialSimulatedSystemCheckpoint(data, base)
		account.GenerationCheckpoint = checkpoint
		if err != nil {
			t.Fatalf("expected configured checkpoint, got %v", err)
		}
		plan, _, err := newSimulatedSystemPlan("workspace", account, data)
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
		want := strings.TrimSuffix(base, "/") + "/fixture/256/" + truth.Person.PhotoID + ".webp"
		if document.Photo == nil || *document.Photo != want {
			t.Fatalf("expected URL %q, got %v", want, document.Photo)
		}

	}

}

// TestSimulatedSystemExternalPhotos checks fallback and external checkpoint
// replay.
func TestSimulatedSystemExternalPhotos(t *testing.T) {

	assets := syntheticworldtest.Assets(t)
	var firstCalls, secondCalls atomic.Int32
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		firstCalls.Add(1)
		_, _ = w.Write([]byte(`{"schema":"invalid"}`))
	}))
	defer first.Close()
	handler := syntheticworldtest.Handler(t, assets, "/mirror")
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secondCalls.Add(1)
		handler.ServeHTTP(w, r)
	}))
	defer second.Close()
	origins := []string{first.URL + "/primary", second.URL + "/mirror/"}
	source, err := fakedata.NewItalianWorldSource(origins)
	if err != nil {
		t.Fatalf("expected Italian source, got %v", err)
	}
	data, err := source.Load(t.Context())
	if err != nil {
		t.Fatalf("expected fallback combination, got %v", err)
	}
	account := simulatedSystemPlanFixture(t, data, "external-account", 20, "50")
	account.GenerationCheckpoint, err = initialSimulatedSystemCheckpoint(data, origins[0])
	if err != nil {
		t.Fatalf("expected external checkpoint, got %v", err)
	}
	plan, checkpoint, err := newSimulatedSystemPlan("workspace-a", account, data)
	if err != nil {
		t.Fatalf("expected external generation plan, got %v", err)
	}
	if checkpoint.CatalogRelease != "artificial-photos" || checkpoint.PhotoBaseURL != origins[0] {
		t.Fatalf("expected international release and first-origin checkpoint, got %#v", checkpoint)
	}
	baseline, oracle := simulatedSystemPlanRecords(t, plan, checkpoint, 4)
	for _, record := range baseline {
		var document simulatedSystemDocument
		err = json.Unmarshal(record.Data, &document)
		if err != nil {
			t.Fatalf("expected generated document, got %v", err)
		}
		if document.Photo == nil || !strings.HasPrefix(*document.Photo, origins[0]+"/fixture/256/") ||
			!strings.HasSuffix(*document.Photo, ".webp") {
			t.Fatalf("expected absolute first-origin photo URL, got %v", document.Photo)
		}
	}
	for range 7 {
		checkpoint = plan.advance(checkpoint)
	}
	account.GeneratedRecordCount = 7
	account.GenerationCheckpoint, err = json.Marshal(checkpoint)
	if err != nil {
		t.Fatalf("expected progress checkpoint, got %v", err)
	}

	// A restarted process reloads the same release, retaining exact replay.
	restarted, err := fakedata.NewItalianWorldSource(origins)
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
	if firstCalls.Load() != 8 || secondCalls.Load() != 8 {
		t.Fatalf("expected eight JSON fetches per origin and no image fetches, got %d/%d",
			firstCalls.Load(), secondCalls.Load())
	}

	account.GenerationCheckpoint = json.Value(legacyPhotoCheckpoint)
	_, _, err = newSimulatedSystemPlan("workspace-a", account, data)
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
		_, _, err = newSimulatedSystemPlan("workspace-a", account, data)
		if err != nil {
			continue
		}
		t.Fatalf("expected release mismatch %q to fail, got success", release)
	}

}

// TestSimulatedSystemLegacyPhotoCheckpoint rejects historical policy and JSON
// without rewriting the serialized cursor or its persisted count.
func TestSimulatedSystemLegacyPhotoCheckpoint(t *testing.T) {

	data := simulatedSystemTestData(t)
	for _, policy := range []string{"generation-v3", "generation-v4"} {
		account := simulatedSystemPlanFixture(t, data, "legacy-account", 20, "50")
		account.GenerationPolicyVersion = policy
		account.GenerationCheckpoint = json.Value(legacyPhotoCheckpoint)
		account.GeneratedRecordCount = 4
		_, _, err := newSimulatedSystemPlan("workspace-a", account, nil)
		if err != nil {
			if string(account.GenerationCheckpoint) != legacyPhotoCheckpoint || account.GeneratedRecordCount != 4 ||
				account.GenerationPolicyVersion != policy {
				t.Fatal("expected untouched legacy metadata, got rewritten checkpoint, count or policy")
			}
			continue
		}
		t.Fatalf("expected retired %s rejected before preparation, got success", policy)
	}

}
