// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/image/webp"
)

// TestUSAReleaseInstallation exercises only the portable release directory,
// including full image decoding, all branches, reloading and reverse replay.
func TestUSAReleaseInstallation(t *testing.T) {

	root := os.Getenv("KRENALIS_TEST_USA_RELEASE_DIR")
	if root == "" {
		t.Skip("portable release installation requires KRENALIS_TEST_USA_RELEASE_DIR")
	}
	data, err := LoadUSAWorldData(t.Context(), root)
	if err != nil {
		t.Fatalf("expected portable USA data, got %v", err)
	}
	if !data.packaged || fmt.Sprintf("%x", data.country.socioeconomic.model.digest) !=
		"f7c8d650c6f3963afbb34a95780c38c1293f29f72df505bbe07d5760e1d48a54" {
		t.Fatal("expected portable current approved USD model, got mismatch")
	}
	config := SourceInstanceConfig{ID: "demo-test", Version: USASourceVersion,
		CoverageNumerator: 1, CoverageDenominator: 1, DuplicateNumerator: 1, DuplicateDenominator: 1}
	world, err := NewUSAWorld(data, IdentityNamespace{Name: "usa-layer1", Generation: 1}, 2026, config)
	if err != nil {
		t.Fatalf("expected portable USA World, got %v", err)
	}
	for _, name := range []string{"ages.json", "first-names.json", "surnames.json", "geography.json"} {
		_, err := os.Stat(filepath.Join(root, usaReleaseDirectory, name))
		if err != nil {
			if !os.IsNotExist(err) {
				t.Fatalf("expected absent production evidence, got %v", err)
			}
		}
		if err == nil {
			t.Fatalf("expected no production evidence in minimal installation, got %s", name)
		}
	}
	observationsDigest := sha256.New()
	activities := map[PrimaryActivity]int{}
	records := map[PersonIndex][]USASourceObservation{}
	for index := PersonIndex(1); index <= 2048; index++ {

		observations, err := world.Records(index)
		if err != nil {
			t.Fatalf("expected USA observations at %d, got %v", index, err)
		}
		for _, observation := range observations {
			if observation.Country != "US" || observation.Currency != "USD" || observation.AnnualIncomeUSD == nil ||
				observation.Observation.PhotoID == nil {
				t.Fatalf("expected photographed USD observation, got %+v", observation)
			}
			encoded, err := world.EncodeDocument(observation)
			if err != nil {
				t.Fatalf("expected encoded USA document, got %v", err)
			}
			observationsDigest.Write(encoded)
			decoded, err := world.DecodeDocument(t.Context(), encoded)
			if err != nil {
				t.Fatalf("expected decoded USA document, got %v", err)
			}
			if !reflect.DeepEqual(decoded, observation) || bytes.Contains(encoded, []byte("annualIncomeEUR")) {
				t.Fatal("expected unchanged USD round trip, got mismatch")
			}
		}
		activities[observations[0].PrimaryActivity]++
		records[index] = observations

	}
	if len(activities) != 6 {
		t.Fatalf("expected all six socioeconomic branches, got %v", activities)
	}
	for _, test := range []struct {
		index PersonIndex
		first string
		last  string
	}{{374054, "Susan", "Dang"}, {18076, "Keith", "Smith"}} {
		observations, err := world.Records(test.index)
		if err != nil {
			t.Fatalf("expected natural names, got %v", err)
		}
		if observations[0].Observation.FirstName == nil || observations[0].Observation.LastName == nil ||
			*observations[0].Observation.FirstName != test.first || *observations[0].Observation.LastName != test.last {
			t.Fatalf("expected %s %s, got %+v", test.first, test.last, observations[0].Observation)
		}
	}
	reloaded, err := LoadUSAWorldData(t.Context(), root)
	if err != nil {
		t.Fatalf("expected independent reload, got %v", err)
	}
	replay, err := NewUSAWorld(reloaded, IdentityNamespace{Name: "usa-layer1", Generation: 1}, 2026, config)
	if err != nil {
		t.Fatalf("expected reloaded World, got %v", err)
	}
	for index := PersonIndex(2048); index > 0; index-- {
		actual, err := replay.Records(index)
		if err != nil {
			t.Fatalf("expected reverse replay, got %v", err)
		}
		if !reflect.DeepEqual(actual, records[index]) {
			t.Fatalf("expected byte-equivalent replay at %d, got mismatch", index)
		}
	}
	photos, err := os.ReadFile(filepath.Join(root, "synthetic-world/photos/v1-json1/catalog.json"))
	if err != nil {
		t.Fatalf("expected packaged catalog, got %v", err)
	}
	var catalog internationalPhotos
	err = json.Unmarshal(photos, &catalog)
	if err != nil {
		t.Fatalf("expected photo catalog, got %v", err)
	}
	count := 0
	for _, face := range catalog.Faces {
		for _, rendition := range face.Renditions {
			encoded, err := os.ReadFile(filepath.Join(root, rendition.Path))
			if err != nil {
				t.Fatalf("expected packaged rendition, got %v", err)
			}
			decoded, err := webp.Decode(bytes.NewReader(encoded))
			if err != nil {
				t.Fatalf("expected fully readable WebP %s, got %v", rendition.Path, err)
			}
			if decoded.Bounds().Dx() != int(rendition.Width) || decoded.Bounds().Dy() != int(rendition.Height) {
				t.Fatal("expected declared image dimensions, got mismatch")
			}
			count++
		}
	}
	if count != 1250 {
		t.Fatalf("expected 1250 fully decoded shared images, got %d", count)
	}
	asset, _, err := data.Photo(t.Context(), *records[1][0].Observation.PhotoID)
	if err != nil {
		t.Fatalf("expected application photo from package, got %v", err)
	}
	if !strings.HasPrefix(asset.Path, "synthetic-world/photos/v1/256/") {
		t.Fatalf("expected stable published photo path, got %s", asset.Path)
	}
	if got := fmt.Sprintf("%x", observationsDigest.Sum(nil)); got != "22f9ae10278ade70f132cf3c5f9c3db3143b65a002982777f67a6cb22cecaf7c" {
		t.Fatalf("expected unchanged observations from the prior release, got %s", got)
	}
	if got := fmt.Sprintf("%x", data.digest); got != "6cb56637d9d41f59e1f764f568dbb6a1f2873aa7e12503bf8a184f126cecd8f1" {
		t.Fatalf("expected unchanged release binding, got %s", got)
	}
	t.Logf("observationsSHA256=%x", observationsDigest.Sum(nil))
	t.Logf("baseWorldID=%s socioeconomicWorldID=%s modelID=%x dataDigest=%x branches=%v images=%d",
		world.adapter.world.baseID, world.adapter.world.id, data.country.socioeconomic.model.digest,
		data.digest, activities, count)

}

// TestUSAReleaseRejection checks that invalid portable input cannot fall back
// to historical production evidence or expose partially prepared data.
func TestUSAReleaseRejection(t *testing.T) {

	root := os.Getenv("KRENALIS_TEST_USA_RELEASE_DIR")
	if root == "" {
		t.Skip("portable release rejection requires KRENALIS_TEST_USA_RELEASE_DIR")
	}
	for _, test := range []struct {
		name string
		path string
	}{{"descriptor", usaReleasePath}, {"base", usaReleaseDirectory + "base.json"},
		{"model", usaReleaseDirectory + "socioeconomic.json"},
		{"phones", usaReleaseDirectory + "npas.json"},
		{"image", "synthetic-world/photos/v1/256/face-000001.webp"}} {

		t.Run(test.name, func(t *testing.T) {

			isolated := t.TempDir()
			for _, name := range []string{"release.json", "base.json", "socioeconomic.json",
				"npas.json", "../../../photos/v1-json1/catalog.json"} {
				path := filepath.Clean(filepath.Join(usaReleaseDirectory, name))
				encoded, err := os.ReadFile(filepath.Join(root, path))
				if err != nil {
					t.Fatalf("expected isolated input, got %v", err)
				}
				err = os.MkdirAll(filepath.Dir(filepath.Join(isolated, path)), 0700)
				if err != nil {
					t.Fatalf("expected isolated directory, got %v", err)
				}
				err = os.WriteFile(filepath.Join(isolated, path), encoded, 0600)
				if err != nil {
					t.Fatalf("expected isolated copy, got %v", err)
				}
			}
			err := os.MkdirAll(filepath.Dir(filepath.Join(isolated, test.path)), 0700)
			if err != nil {
				t.Fatalf("expected isolated corruption directory, got %v", err)
			}
			err = os.WriteFile(filepath.Join(isolated, test.path), []byte("{}"), 0600)
			if err != nil {
				t.Fatalf("expected corrupted input, got %v", err)
			}
			data, err := LoadUSAWorldData(t.Context(), isolated)
			if err != nil {
				if data != nil {
					t.Fatal("expected no partial data on failure, got prepared data")
				}
				return
			}

			t.Fatal("expected corrupted portable input rejection, got success")

		})

	}

}
