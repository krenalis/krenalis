// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/webp"
)

// TestUSAPhotoCorpusQualification checks every existing 256 rendition, not
// selected samples. This is a technical audit, not a grant of image rights.
func TestUSAPhotoCorpusQualification(t *testing.T) {

	encoded, err := os.ReadFile("../../synthetic-world-staging-ii-20261013/photos.json")
	if err != nil {
		t.Fatalf("expected existing complete photo metadata, got %v", err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(encoded)) != "b81061ea6754ecb68a8e5b5fcc25ef10321d738500557dc98ae7b99c14fa0db4" {
		t.Fatal("expected preserved original photo metadata, got different identity")
	}
	photos, err := decodePhotoCatalogData(t.Context(), bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("expected existing international photo catalog, got %v", err)
	}
	var inventory struct {
		Status              string `json:"status"`
		Rights              string `json:"rights"`
		SourceCatalogSHA256 string `json:"sourceCatalogSHA256"`
		Count               uint32 `json:"count"`
		Renditions          []struct {
			ID           string `json:"id"`
			Gender       string `json:"gender"`
			RequestedAge uint32 `json:"requestedAge"`
			Path         string `json:"path"`
			Bytes        uint64 `json:"bytes"`
			SHA256       string `json:"sha256"`
		} `json:"renditions" max:"1250"`
	}
	inventoryBytes := readUSACompletionFile(t, "evidence/photos-256-inventory.json")
	err = decodeInternationalJSON(t.Context(), bytes.NewReader(inventoryBytes), &inventory)
	if err != nil {
		t.Fatalf("expected bounded audit inventory, got %v", err)
	}
	if len(photos.document.Faces) != 1250 || inventory.Count != 1250 || len(inventory.Renditions) != 1250 ||
		!strings.HasPrefix(inventory.Rights, "OPEN:") {
		t.Fatal("expected complete technical audit with open rights condition, got different inventory")
	}
	byID := map[string]internationalFace{}
	for _, face := range photos.document.Faces {
		byID[face.ID] = face
	}
	genders := map[string]int{}
	ageCoverage := map[string]map[uint32]int{"female": {}, "male": {}}
	var totalBytes uint64
	minimum, maximum := uint64(1<<20), uint64(0)
	for _, entry := range inventory.Renditions {

		face, exists := byID[entry.ID]
		if !exists || face.Gender != entry.Gender || face.RequestedAge != entry.RequestedAge ||
			entry.Path != "tools/fakefacegen/output/faces-v1/derived/256/"+entry.ID+".webp" ||
			entry.Bytes == 0 || entry.Bytes > 1<<20 || entry.RequestedAge < 21 || entry.RequestedAge > 75 ||
			ageCoverage[entry.Gender] == nil {
			t.Fatal("expected matched, bounded photo inventory, got incompatible entry")
		}
		// Bound encoded bytes and dimensions before decoding pixel storage.
		data, err := readBoundedPhotoFile(t.Context(), filepath.Join("../..", entry.Path), 1<<20)
		if err != nil {
			t.Fatalf("expected existing 256 rendition, got %v", err)
		}
		if uint64(len(data)) != entry.Bytes || fmt.Sprintf("%x", sha256.Sum256(data)) != entry.SHA256 {
			t.Fatalf("expected declared photo integrity for %s, got mismatch", entry.ID)
		}
		matched := false
		for _, rendition := range face.Renditions {
			if rendition.Width == 256 && rendition.Height == 256 && rendition.Integrity != nil &&
				rendition.Integrity.Bytes == entry.Bytes && rendition.Integrity.SHA256 == entry.SHA256 {
				matched = true
			}
		}
		if !matched {
			t.Fatalf("expected original 256 integrity declaration for %s, got missing rendition", entry.ID)
		}
		config, err := webp.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("expected decodable WebP config for %s, got %v", entry.ID, err)
		}
		if config.Width != 256 || config.Height != 256 {
			t.Fatalf("expected 256x256 dimensions, got %dx%d", config.Width, config.Height)
		}
		decoded, err := webp.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("expected full WebP decode for %s, got %v", entry.ID, err)
		}
		if decoded.Bounds().Dx() != 256 || decoded.Bounds().Dy() != 256 {
			t.Fatalf("expected 256x256 decoded pixels, got %v", decoded.Bounds())
		}
		genders[entry.Gender]++
		ageCoverage[entry.Gender][entry.RequestedAge]++
		totalBytes += entry.Bytes
		minimum, maximum = min(minimum, entry.Bytes), max(maximum, entry.Bytes)

	}
	if totalBytes != 13030260 || genders["female"] != 652 || genders["male"] != 598 ||
		len(ageCoverage["female"]) != 55 || len(ageCoverage["male"]) != 53 {
		t.Fatalf("expected 13,030,260 bytes and qualified age coverage 55/53, got %d/%d/%d", totalBytes,
			len(ageCoverage["female"]), len(ageCoverage["male"]))
	}
	// The existing selector uses operational gender and nearest requested age.
	gendersTaxonomy := internationalTaxonomyFor("sw-l1-gender-v1", []Gender{GenderFemale, GenderMale})
	catalog, err := prepareItalianPhotoCatalog(photos, gendersTaxonomy)
	if err != nil {
		t.Fatalf("expected existing shared selector compatible with USA genders, got %v", err)
	}
	w := readUSACompletionHarness(t)
	maximumDistance := 0
	for _, gender := range []Gender{GenderFemale, GenderMale} {
		for age := 21; age <= 75; age++ {
			nearest := 55
			for requested := range ageCoverage[string(gender)] {
				distance := int(requested) - age
				if distance < 0 {
					distance = -distance
				}
				nearest = min(nearest, distance)
			}
			maximumDistance = max(maximumDistance, nearest)
			for _, index := range []PersonIndex{1, 42, 65536, Layer1MaxPersonIndex} {
				id, err := catalog.selectPhoto(w.streams, index, gender, age)
				if err != nil {
					t.Fatalf("expected compatible photo selection, got %v", err)
				}
				face := byID[id]
				distance := int(face.RequestedAge) - age
				if distance < 0 {
					distance = -distance
				}
				if face.Gender != string(gender) || distance != nearest {
					t.Fatalf("expected nearest requested age and gender, got %s/%d", face.Gender, face.RequestedAge)
				}
			}
		}
	}
	if maximumDistance != 1 {
		t.Fatalf("expected maximum nearest-age distance one, got %d", maximumDistance)
	}
	t.Logf("photos=%d genders=%v female=21..75 male=21..74 except 72; "+
		"nearest gap=%d bytes=%d min=%d max=%d D_photos=%x F_photo=%x",
		inventory.Count, genders, maximumDistance, totalBytes, minimum, maximum, photos.digest, photos.fingerprint)

}

// TestUSAPhotoIdentitySeparation checks rendition-independent selection and
// prevents photographic dependencies from entering the USA base or contacts.
func TestUSAPhotoIdentitySeparation(t *testing.T) {

	encoded, err := os.ReadFile("../../synthetic-world-staging-ii-20261013/photos.json")
	if err != nil {
		t.Fatalf("expected existing photo catalog, got %v", err)
	}
	first, err := decodePhotoCatalogData(t.Context(), bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("expected decoded photo catalog, got %v", err)
	}
	second, err := decodePhotoCatalogData(t.Context(), bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("expected independently decoded photo catalog, got %v", err)
	}
	for i := range second.document.Faces {
		for j := range second.document.Faces[i].Renditions {
			second.document.Faces[i].Renditions[j].Path = "replacement/" + second.document.Faces[i].Renditions[j].Path
		}
	}
	changed, err := preparePhotoCatalogData(second.document)
	if err != nil {
		t.Fatalf("expected alternate rendition metadata, got %v", err)
	}
	if first.fingerprint != changed.fingerprint || first.digest == changed.digest {
		t.Fatal("expected rendition-independent selection and separate catalog identity, got conflation")
	}
	w := readUSACompletionHarness(t)
	before, err := w.person(42)
	if err != nil {
		t.Fatalf("expected original USA person, got %v", err)
	}
	firstCatalog, err := prepareItalianPhotoCatalog(first, first.document.GenderTaxonomy)
	if err != nil {
		t.Fatalf("expected original selector, got %v", err)
	}
	changedCatalog, err := prepareItalianPhotoCatalog(changed, changed.document.GenderTaxonomy)
	if err != nil {
		t.Fatalf("expected replacement selector, got %v", err)
	}
	for _, index := range []PersonIndex{1, 42, Layer1MaxPersonIndex} {
		a, err := firstCatalog.selectPhoto(w.streams, index, GenderFemale, 35)
		if err != nil {
			t.Fatalf("expected original selection, got %v", err)
		}
		b, err := changedCatalog.selectPhoto(w.streams, index, GenderFemale, 35)
		if err != nil {
			t.Fatalf("expected rendition-independent selection, got %v", err)
		}
		if a != b {
			t.Fatalf("expected same PhotoID after rendition change, got %s/%s", a, b)
		}
	}
	second.document.Selection.Seed = strings.Repeat("1", 64)
	changed, err = preparePhotoCatalogData(second.document)
	if err != nil {
		t.Fatalf("expected alternate selection identity, got %v", err)
	}
	if first.fingerprint == changed.fingerprint {
		t.Fatal("expected selection-seed-sensitive F_photo, got unchanged identity")
	}
	after, err := w.person(42)
	if err != nil {
		t.Fatalf("expected unchanged USA person, got %v", err)
	}
	if before != after {
		t.Fatal("expected photography separate from base, contacts and future L2 Division, got interference")
	}

}
