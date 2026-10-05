// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// TestExternalPhotoCatalog checks that runtime metadata contains no image
// bytes.
func TestExternalPhotoCatalog(t *testing.T) {

	data := externalPhotoFixture(t)
	catalog, err := decodeExternalPhotoCatalog(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("expected typed catalog, got %v", err)
	}
	if catalog.Release() != "v1" || len(catalog.faces) != 4 || len(catalog.paths) != 0 {
		t.Fatalf("expected v1 metadata without served images, got %#v", catalog)
	}
	asset, err := catalog.Asset("face-000013", PhotoSize256)
	if err != nil {
		t.Fatalf("expected rendition metadata, got %v", err)
	}
	if asset.Path != "synthetic-world/photos/v1/256/face-000013.webp" ||
		asset.ContentLength != 0 || asset.ContentSHA256 != "" {
		t.Fatalf("expected relative rendition without byte metadata, got %#v", asset)
	}
	for _, record := range catalog.assets {
		if record.data != nil {
			t.Fatalf("expected metadata only, got %d image bytes", len(record.data))
		}
	}

}

// TestExternalPhotoCatalogCompatibility preserves frozen local selection
// inputs.
func TestExternalPhotoCatalogCompatibility(t *testing.T) {

	directory := makeCompleteFaceCatalogFixture(t, []string{
		"face-000013", "face-000013", "face-000037", "face-000037", "face-000013", "face-000037",
	})
	var specs []fakefacegenSpec
	err := readFakefacegenJSON(filepath.Join(directory, "specs.json"), &specs)
	if err != nil {
		t.Fatalf("expected fixture specs, got %v", err)
	}
	for i, age := range []int{21, 75, 35, 45, 21, 75} {
		specs[i].Age = age
	}
	writeFaceCatalogTestJSON(t, filepath.Join(directory, "specs.json"), specs)
	mutateFakefacegenManifest(t, directory, "face-000005", func(asset *fakefacegenManifestAsset) {
		asset.Status = fakefacegenStatusFailed
	})
	mutateFakefacegenManifest(t, directory, "face-000006", func(asset *fakefacegenManifestAsset) {
		asset.Status = fakefacegenStatusMissing
	})
	local, err := LoadFaceCatalog(t.Context(), directory)
	if err != nil {
		t.Fatalf("expected local catalog, got %v", err)
	}
	seed := local.Checksum()
	document := externalPhotoCatalog{Format: 1, Release: "v1", SelectionSeed: hex.EncodeToString(seed[:])}
	for _, face := range local.faces {
		if face.excluded {
			continue
		}
		document.Faces = append(document.Faces, externalPhotoFace{
			ID: face.id, Gender: face.gender, RequestedAge: face.requestedAge,
		})
	}
	slices.Reverse(document.Faces) // Distribution order must not change tie selection.
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("expected metadata JSON, got %v", err)
	}
	external, err := decodeExternalPhotoCatalog(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("expected equivalent external catalog, got %v", err)
	}
	a := newLayer1WorldWithCatalog(t, local)
	b := newLayer1WorldWithCatalog(t, external)
	if a.SnapshotID() != b.SnapshotID() || a.WorldID() != b.WorldID() {
		t.Fatalf("expected unchanged world and snapshot identity, got %s and %s", a.SnapshotID(), b.SnapshotID())
	}
	for _, test := range []struct {
		gender     Gender
		age        int
		candidates []string
	}{
		{GenderFemale, 21, []string{"face-000001"}},
		{GenderFemale, 47, []string{"face-000001"}},
		{GenderFemale, 48, []string{"face-000001", "face-000002"}},
		{GenderFemale, 49, []string{"face-000002"}},
		{GenderFemale, 75, []string{"face-000002"}},
		{GenderMale, 21, []string{"face-000003"}},
		{GenderMale, 40, []string{"face-000003", "face-000004"}},
		{GenderMale, 75, []string{"face-000004"}},
	} {

		seen := map[string]bool{}
		for index := PersonIndex(1); index <= 64; index++ {
			before, err := local.selectPhoto(a.streams, index, test.gender, test.age)
			if err != nil {
				t.Fatalf("expected local selection, got %v", err)
			}
			after, err := external.selectPhoto(b.streams, index, test.gender, test.age)
			if err != nil {
				t.Fatalf("expected external selection, got %v", err)
			}
			if before != after || !slices.Contains(test.candidates, after) {
				t.Fatalf("expected equivalent selection in %v for %s/%d, got %q/%q",
					test.candidates, test.gender, test.age, before, after)
			}
			seen[after] = true
		}
		if len(seen) != len(test.candidates) {
			t.Fatalf("expected every tie candidate %v selected, got %v", test.candidates, seen)
		}

	}

	for index := PersonIndex(1); index <= 64; index++ {
		before, err := a.Person(index)
		if err != nil {
			t.Fatalf("expected local person, got %v", err)
		}
		after, err := b.Person(index)
		if err != nil {
			t.Fatalf("expected external person, got %v", err)
		}
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("expected unchanged Person and PhotoID at %d, got %#v and %#v", index, before, after)
		}
	}

}

// TestExternalPhotoCatalogFallback checks fallback through the production
// loader.
func TestExternalPhotoCatalogFallback(t *testing.T) {

	fixture := externalPhotoFixture(t)
	for _, failure := range []string{"HTTP", "JSON", "semantic"} {
		t.Run(failure, func(t *testing.T) {

			var firstCalls, secondCalls atomic.Int32
			first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				firstCalls.Add(1)
				switch failure {
				case "HTTP":
					w.WriteHeader(http.StatusServiceUnavailable)
				case "JSON":
					_, _ = io.WriteString(w, "{")
				case "semantic":
					_, _ = io.WriteString(w, strings.Replace(string(fixture), `"requestedAge": 35`, `"requestedAge": 76`, 1))
				}
			}))
			defer first.Close()
			second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				secondCalls.Add(1)
				_, _ = w.Write(fixture)
			}))
			defer second.Close()
			source, err := NewFaceCatalogSource([]string{first.URL, second.URL})
			if err != nil {
				t.Fatalf("expected source, got %v", err)
			}
			catalog, err := source.Load(t.Context())
			if err != nil {
				t.Fatalf("expected valid fallback catalog, got %v", err)
			}
			if catalog.Release() != "v1" || firstCalls.Load() != 1 || secondCalls.Load() != 1 {
				t.Fatalf("expected second origin to win once, got release %q and calls %d/%d",
					catalog.Release(), firstCalls.Load(), secondCalls.Load())
			}

		})
	}

}

// TestExternalPhotoCatalogInvalid checks representation and selection
// constraints.
func TestExternalPhotoCatalogInvalid(t *testing.T) {

	valid := string(externalPhotoFixture(t))
	cases := []string{
		"{", "null", valid + "{}",
		strings.Replace(valid, `"format": 1`, `"format": 2`, 1),
		strings.Replace(valid, `"release": "v1"`, `"release": "v2"`, 1),
		strings.Replace(valid, `"release": "v1"`, `"release": "v1", "release": "v1"`, 1),
		strings.Replace(valid, `"format": 1`, `"unknown": 1`, 1),
		strings.Replace(valid, strings.Repeat("1", 64), "invalid", 1),
		strings.Replace(valid, strings.Repeat("1", 64), strings.Repeat("0", 64), 1),
		strings.Replace(valid, `"faces": [`, `"faces": null, "unused": [`, 1),
		strings.Replace(valid, `"face-000014"`, `"face-000013"`, 1),
		strings.Replace(valid, `"face-000014"`, `"../escape"`, 1),
		strings.Replace(valid, `"requestedAge": 35`, `"requestedAge": 20`, 1),
		strings.Replace(valid, `"requestedAge": 35`, `"requestedAge": 76`, 1),
		strings.ReplaceAll(valid, `"gender": "male"`, `"gender": "female"`),
		strings.Replace(valid, `"gender": "female"`, `"gender": "other"`, 1),
	}
	for i, data := range cases {
		_, err := decodeExternalPhotoCatalog(strings.NewReader(data))
		if err != nil {
			continue
		}
		t.Fatalf("expected invalid catalog case %d to fail, got success", i)
	}

}

// TestExternalPhotoCatalogLimit checks the catalog decoded-byte boundary.
func TestExternalPhotoCatalogLimit(t *testing.T) {

	fixture := externalPhotoFixture(t)
	for _, extra := range []int{0, 1} {

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(fixture)
			_, _ = io.WriteString(w, strings.Repeat(" ", maxExternalPhotoCatalogBytes-len(fixture)+extra))
		}))
		source, err := NewFaceCatalogSource([]string{server.URL})
		if err != nil {
			t.Fatalf("expected source, got %v", err)
		}
		_, err = source.Load(t.Context())
		server.Close()
		if err != nil {
			if extra == 0 || !strings.Contains(err.Error(), "exceeds") {
				t.Fatalf("expected oversize error only, got %v", err)
			}
			continue
		}
		if extra != 0 {
			t.Fatal("expected decoded catalog limit, got success")
		}

	}

}

// TestExternalPhotoCatalogRecordLimit bounds cardinality before compilation.
func TestExternalPhotoCatalogRecordLimit(t *testing.T) {

	data := `{"format":1,"release":"v1","selectionSeed":"` + strings.Repeat("1", 64) + `","faces":[` +
		strings.Repeat(`{},`, maxFaceCatalogRecords) + `{}` + `]}`
	_, err := decodeExternalPhotoCatalog(strings.NewReader(data))
	if err != nil {
		if !strings.Contains(err.Error(), "exceeds 100000 records") {
			t.Fatalf("expected record-count bound before compilation, got %v", err)
		}
		return
	}
	t.Fatal("expected oversized record count to fail, got success")

}

// TestExternalPhotoCatalogSourceLifetime checks retry and concurrent cache
// reuse.
func TestExternalPhotoCatalogSourceLifetime(t *testing.T) {

	fixture := externalPhotoFixture(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/prefix/synthetic-world/photos/v1/catalog.json" {
			t.Errorf("expected prefixed catalog request only, got %s", r.URL.Path)
			w.WriteHeader(404)
			return
		}
		if calls.Add(1) == 1 {
			_, _ = io.WriteString(w, "invalid")
			return
		}
		_, _ = w.Write(fixture)
	}))
	defer server.Close()
	source, err := NewFaceCatalogSource([]string{server.URL + "/prefix"})
	if err != nil {
		t.Fatalf("expected source, got %v", err)
	}
	catalog, err := source.Load(t.Context())
	if err != nil {
		if catalog != nil {
			t.Fatalf("expected failed load without cached catalog, got %#v", catalog)
		}
	}
	if err == nil {
		t.Fatal("expected first invalid load to fail, got success")
	}
	var wg sync.WaitGroup
	results := make(chan *FaceCatalog, 20)
	for i := 0; i < cap(results); i++ {
		wg.Go(func() {
			catalog, err := source.Load(t.Context())
			if err != nil {
				t.Errorf("expected retried successful load, got %v", err)
			}
			results <- catalog
		})
	}
	wg.Wait()
	close(results)
	for current := range results {
		if catalog == nil {
			catalog = current
		}
		if current == nil || current != catalog {
			t.Fatalf("expected shared immutable catalog, got %p and %p", catalog, current)
		}
	}
	_, err = source.Load(t.Context())
	if err != nil {
		t.Fatalf("expected cached catalog, got %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected one failed and one successful fetch, got %d", calls.Load())
	}

}

// TestExternalPhotoCatalogWaitingCancellation checks cancellable load waiters.
func TestExternalPhotoCatalogWaitingCancellation(t *testing.T) {

	fixture := externalPhotoFixture(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		_, _ = w.Write(fixture)
	}))
	defer server.Close()
	source, err := NewFaceCatalogSource([]string{server.URL})
	if err != nil {
		t.Fatalf("expected source, got %v", err)
	}
	finished := make(chan error, 1)
	go func() {
		_, err := source.Load(t.Context())
		finished <- err
	}()
	<-entered
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = source.Load(ctx)
	close(release)
	if err != nil {
		if err != context.Canceled {
			t.Errorf("expected waiting cancellation, got %v", err)
		}
	}
	if err == nil {
		t.Error("expected waiting cancellation, got success")
	}
	err = <-finished
	if err != nil {
		t.Fatalf("expected original load to succeed, got %v", err)
	}

}

func externalPhotoFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/external-photos-v1.json")
	if err != nil {
		t.Fatalf("expected external catalog fixture, got %v", err)
	}
	return data
}
