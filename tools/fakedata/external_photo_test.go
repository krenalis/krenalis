// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
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

func externalPhotoFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/external-photos-v1.json")
	if err != nil {
		t.Fatalf("expected external catalog fixture, got %v", err)
	}
	return data
}
