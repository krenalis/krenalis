// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

func TestUSAWorldSource(t *testing.T) {

	assets := usaRemoteAssetFixture(t)
	var requests atomic.Int32
	var altered atomic.Bool
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		path := strings.TrimPrefix(r.URL.Path, "/")
		encoded, exists := assets[path]
		if !exists {
			http.NotFound(w, r)
			return
		}
		if altered.Load() && (strings.HasSuffix(path, "base.json") || strings.HasSuffix(path, ".webp")) {
			encoded = bytes.Clone(encoded)
			encoded[0] ^= 1
		}
		_, _ = w.Write(encoded)
	}))
	defer origin.Close()
	source, err := NewUSAWorldSource([]string{origin.URL})
	if err != nil {
		t.Fatalf("expected configured remote source, got %v", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("expected no acquisition before Load, got %d requests", requests.Load())
	}
	altered.Store(true)
	data, err := source.Load(t.Context())
	if err != nil {
		if data != nil || source.prepared != nil {
			t.Fatal("expected no partial remote data, got prepared data")
		}
	}
	if err == nil {
		t.Fatal("expected corrupt remote base rejection, got admission")
	}
	altered.Store(false)
	data, err = source.Load(t.Context())
	if err != nil {
		t.Fatalf("expected remote retry, got %v", err)
	}
	if data.resolver == nil ||
		fmt.Sprintf("%x", data.digest) != "6cb56637d9d41f59e1f764f568dbb6a1f2873aa7e12503bf8a184f126cecd8f1" {
		t.Fatal("expected repository-independent published binding, got different data")
	}
	loadedRequests := requests.Load()
	reloaded, err := source.Load(t.Context())
	if err != nil {
		t.Fatalf("expected cached remote data, got %v", err)
	}
	if reloaded != data || requests.Load() != loadedRequests {
		t.Fatal("expected immutable cached success, got repeated acquisition")
	}

	// Compare exact observations with the portable published release baseline.
	world, err := NewUSAWorld(data, IdentityNamespace{Name: "usa-layer1", Generation: 1}, 2026,
		SourceInstanceConfig{ID: "demo-test", Version: USASourceVersion, CoverageNumerator: 1,
			CoverageDenominator: 1, DuplicateNumerator: 1, DuplicateDenominator: 1})
	if err != nil {
		t.Fatalf("expected remote USA World, got %v", err)
	}
	digest := sha256.New()
	for index := PersonIndex(1); index <= 2048; index++ {
		records, err := world.Records(index)
		if err != nil {
			t.Fatalf("expected USA generation, got %v", err)
		}
		for _, record := range records {
			if record.Country != "US" || record.Currency != "USD" || record.AnnualIncomeUSD == nil ||
				record.Observation.PhotoID == nil {
				t.Fatal("expected photographed USA/USD observation, got incomplete data")
			}
			encoded, err := world.EncodeDocument(record)
			if err != nil {
				t.Fatalf("expected unchanged document encoding, got %v", err)
			}
			digest.Write(encoded)
		}
	}
	if got := fmt.Sprintf("%x", digest.Sum(nil)); got != "22f9ae10278ade70f132cf3c5f9c3db3143b65a002982777f67a6cb22cecaf7c" {
		t.Fatalf("expected unchanged published observations, got %s", got)
	}
	_, image, err := data.Photo(t.Context(), "face-000001")
	if err != nil {
		t.Fatalf("expected remote WebP, got %v", err)
	}
	if !bytes.Equal(image, assets["synthetic-world/photos/v1/256/face-000001.webp"]) {
		t.Fatal("expected original remote image bytes, got mismatch")
	}
	altered.Store(true)
	_, image, err = data.Photo(t.Context(), "face-000001")
	if err != nil {
		if image != nil {
			t.Fatal("expected no corrupt image exposure, got bytes")
		}
		return
	}

	t.Fatal("expected corrupt remote photo rejection, got admission")
}

func publishedUSATestData(t *testing.T) *USAWorldData {
	t.Helper()
	origin := httptest.NewServer(http.FileServer(http.Dir("testdata/published")))
	t.Cleanup(origin.Close)
	source, err := NewUSAWorldSource([]string{origin.URL + "/"})
	if err != nil {
		t.Fatalf("expected configured USA source, got %v", err)
	}
	data, err := source.Load(t.Context())
	if err != nil {
		t.Fatalf("expected prepared USA data, got %v", err)
	}
	return data
}

func readUSACountryData(t *testing.T) *usaCountryData {
	t.Helper()
	return publishedUSATestData(t).country
}

func usaRemoteAssetFixture(t *testing.T) map[string][]byte {
	t.Helper()
	assets := map[string][]byte{}
	for _, path := range []string{
		usaReleasePath, usaReleaseDirectory + "base.json", usaReleaseDirectory + "npas.json",
		usaReleaseDirectory + "socioeconomic.json", "synthetic-world/photos/v1-json1/catalog.json",
		"synthetic-world/photos/v1/256/face-000001.webp",
	} {
		encoded, err := os.ReadFile("testdata/published/" + path)
		if err != nil {
			t.Fatalf("expected approved fixture %s, got %v", path, err)
		}
		assets[path] = encoded
	}
	return assets
}
