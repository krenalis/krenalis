// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestConvertLocalPhotoCatalogDataInvalidSeed rejects invalid seeds before I/O.
func TestConvertLocalPhotoCatalogDataInvalidSeed(t *testing.T) {

	directory := filepath.Join(t.TempDir(), "missing-snapshot")

	for _, test := range []struct {
		name string
		seed string
	}{
		{"empty", ""},
		{"short", strings.Repeat("1", 63)},
		{"long", strings.Repeat("1", 65)},
		{"nonhex", strings.Repeat("g", 64)},
		{"zero", strings.Repeat("0", 64)},
		{"uppercase", strings.Repeat("A", 64)},
		{"mixed case", "A" + strings.Repeat("b", 63)},
		{"prefix", "0x" + strings.Repeat("1", 62)},
		{"leading space", " " + strings.Repeat("1", 64)},
		{"trailing newline", strings.Repeat("1", 64) + "\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := ConvertLocalPhotoCatalogData(t.Context(), directory, test.seed)
			if err != nil {
				if !strings.Contains(err.Error(), "selection seed must be") {
					t.Fatalf("expected seed rejection before snapshot loading, got %v", err)
				}
				return
			}
			t.Fatal("expected invalid selection seed rejected, got success")
		})
	}

}

// TestConvertLocalPhotoCatalogDataRenditionIsolation reconverts real WebP bytes
// in an isolated snapshot and checks selection through the international World.
func TestConvertLocalPhotoCatalogDataRenditionIsolation(t *testing.T) {

	// Two candidates per gender exercise seed-dependent ties in the selector.
	directory := makeCompleteFaceCatalogFixture(t, []string{
		"face-000013", "face-000013", "face-000037", "face-000037",
	})
	seed := strings.Repeat("1", 64)
	baseline := mustConvertLocalPhotoCatalogData(t, directory, seed)
	originalSnapshot, err := LoadFaceCatalog(t.Context(), directory)
	if err != nil {
		t.Fatalf("expected valid original snapshot, got %v", err)
	}

	copyFaceCatalogTestAsset(t, directory, "face-000001", "face-000037", PhotoSize256)
	changed := mustConvertLocalPhotoCatalogData(t, directory, seed)
	changedSnapshot, err := LoadFaceCatalog(t.Context(), directory)
	if err != nil {
		t.Fatalf("expected valid modified snapshot, got %v", err)
	}
	if originalSnapshot.Checksum() == changedSnapshot.Checksum() {
		t.Fatalf("expected rendition-dependent snapshot checksum, got unchanged %x", changedSnapshot.Checksum())
	}
	if baseline.document.Selection.Seed != seed || changed.document.Selection != baseline.document.Selection {
		t.Fatalf("expected explicit selection seed %s retained, got %#v", seed, changed.document.Selection)
	}
	if baseline.fingerprint != changed.fingerprint {
		t.Fatalf("expected unchanged F_photo %x, got %x", baseline.fingerprint, changed.fingerprint)
	}
	if baseline.digest == changed.digest {
		t.Fatalf("expected changed D_photos after rendition replacement, got unchanged %x", changed.digest)
	}

	if len(baseline.document.Faces) != 4 || len(changed.document.Faces) != 4 {
		t.Fatal("expected four candidates before and after conversion, got different counts")
	}
	for i, face := range baseline.document.Faces {

		other := changed.document.Faces[i]
		if face.ID != other.ID || face.Gender != other.Gender || face.RequestedAge != other.RequestedAge ||
			len(face.Renditions) != 5 || len(other.Renditions) != 5 {
			t.Fatal("expected unchanged candidates and five renditions each, got different metadata")
		}

		for j, before := range face.Renditions {
			after := other.Renditions[j]
			if face.ID == "face-000001" && before.Width == 256 {
				if before.Integrity == nil || after.Integrity == nil ||
					before.Integrity.SHA256 == after.Integrity.SHA256 ||
					before.Integrity.Bytes == after.Integrity.Bytes {
					t.Fatal("expected changed WebP hash and byte count, got unchanged or absent integrity")
				}
				before.Integrity = after.Integrity
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("expected only the replaced rendition's integrity to change, got other metadata changes")
			}
		}

	}

	assets := loadInternationalTestAssets(t, internationalAssetFixture(t))
	beforeData, err := PrepareItalianWorldData(assets.base, assets.socioeconomic, baseline)
	if err != nil {
		t.Fatalf("expected prepared original photos, got %v", err)
	}
	afterData, err := PrepareItalianWorldData(assets.base, assets.socioeconomic, changed)
	if err != nil {
		t.Fatalf("expected prepared modified photos, got %v", err)
	}
	beforeWorld, afterWorld := mustItalianJSONWorld(t, beforeData), mustItalianJSONWorld(t, afterData)
	selected := map[string]bool{}
	for index := PersonIndex(1); index <= 64; index++ {
		before, err := beforeWorld.Person(index)
		if err != nil {
			t.Fatalf("expected original Person, got %v", err)
		}
		after, err := afterWorld.Person(index)
		if err != nil {
			t.Fatalf("expected Person after rendition replacement, got %v", err)
		}
		if before.PhotoID != after.PhotoID {
			t.Fatalf("expected unchanged PhotoID %s at index %d, got %s", before.PhotoID, index, after.PhotoID)
		}
		selected[before.PhotoID] = true
	}
	if len(selected) != 4 {
		t.Fatalf("expected selection coverage of all four tied candidates, got %v", selected)
	}

}

// TestConvertLocalPhotoCatalogDataSelectionSeed changes selection explicitly.
func TestConvertLocalPhotoCatalogDataSelectionSeed(t *testing.T) {

	directory := makeCompleteFaceCatalogFixture(t, []string{"face-000013", "face-000037"})
	baseline := mustConvertLocalPhotoCatalogData(t, directory, strings.Repeat("1", 64))
	seed := strings.Repeat("0", 63) + "1"
	changed := mustConvertLocalPhotoCatalogData(t, directory, seed)

	if changed.document.Selection.Seed != seed {
		t.Fatalf("expected explicitly changed selection seed %s, got %s", seed, changed.document.Selection.Seed)
	}
	if baseline.fingerprint == changed.fingerprint {
		t.Fatalf("expected seed-dependent F_photo, got unchanged %x", changed.fingerprint)
	}
	if !reflect.DeepEqual(baseline.document.Faces, changed.document.Faces) {
		t.Fatal("expected unchanged candidates and renditions after seed change, got different faces")
	}

}

func mustConvertLocalPhotoCatalogData(t *testing.T, directory, selectionSeed string) *PhotoCatalogData {

	t.Helper()
	converted, err := ConvertLocalPhotoCatalogData(t.Context(), directory, selectionSeed)
	if err != nil {
		t.Fatalf("expected converted local snapshot, got %v", err)
	}

	canonical, err := converted.CanonicalJSON()
	if err != nil {
		t.Fatalf("expected canonical converted photos, got %v", err)
	}
	decoded, err := DecodePhotoCatalogData(t.Context(), bytes.NewReader(canonical))
	if err != nil {
		t.Fatalf("expected decoded converted photos, got %v", err)
	}
	if !reflect.DeepEqual(converted, decoded) {
		t.Fatal("expected converted catalog preserved through canonical JSON, got different data")
	}

	return decoded
}
