// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestLayer1FaceCatalogChecksums protects the distinct frozen source and
// semantic selected-fixture snapshot identities.
func TestLayer1FaceCatalogChecksums(t *testing.T) {
	catalog := mustLayer1Catalog(t)
	sourceChecksum := catalog.SourceCatalogChecksum()
	if got := hex.EncodeToString(sourceChecksum[:]); got != "1644a688a2a8729acf1335adf634842100d9cdaafb1394e098fc16b74a3c7c59" {
		t.Fatalf(
			"expected source catalog checksum 1644a688a2a8729acf1335adf634842100d9cdaafb1394e098fc16b74a3c7c59, got %s",
			got,
		)
	}
	checksum := catalog.Checksum()
	if got := hex.EncodeToString(checksum[:]); got != "11bfa8eff1873ca3b52840be0136ebefc9603558a2cc5f5b254b0df447491870" {
		t.Fatalf(
			"expected semantic catalog checksum 11bfa8eff1873ca3b52840be0136ebefc9603558a2cc5f5b254b0df447491870, got %s",
			got,
		)
	}
	if !bytes.Equal(catalog.dependencies[1].value, checksum[:]) {
		t.Fatalf("expected semantic checksum photo dependency %x, got %x", checksum, catalog.dependencies[1].value)
	}
	if bytes.Equal(catalog.dependencies[1].value, sourceChecksum[:]) {
		t.Fatalf("expected photo dependency distinct from source checksum, got %x", catalog.dependencies[1].value)
	}
}

// TestLayer1FaceCatalogMetadataStability proves one load cannot mix metadata
// captured from different same-count generations.
func TestLayer1FaceCatalogMetadataStability(t *testing.T) {

	for _, test := range []struct {
		name            string
		step            string
		occurrence      int
		selectedFixture bool
	}{
		{name: "after catalog capture", step: "catalog.json", occurrence: 1},
		{name: "after specs capture", step: "specs.json", occurrence: 1, selectedFixture: true},
		{name: "after manifest capture", step: "manifest.json", occurrence: 1},
		{name: "after final asset", step: "asset", occurrence: 2 * len(supportedPhotoSizes), selectedFixture: true},
	} {
		t.Run(test.name, func(t *testing.T) {

			directory, replacement := makeFaceCatalogMetadataStabilityFixtures(t)
			occurrence := 0
			mutated := false
			faceCatalogLoadStepForTest = func(step string) {
				if mutated || step != test.step {
					return
				}
				occurrence++
				if occurrence != test.occurrence {
					return
				}
				publishFaceCatalogTestMetadata(t, replacement, directory)
				mutated = true
			}
			t.Cleanup(func() {
				faceCatalogLoadStepForTest = nil
			})

			var err error
			if test.selectedFixture {
				_, err = loadSelectedFaceCatalogFixture(
					t.Context(), directory, []string{"face-000001", "face-000002"},
				)
			} else {
				_, err = LoadFaceCatalog(t.Context(), directory)
			}
			faceCatalogLoadStepForTest = nil
			if err != nil {
				if !errors.Is(err, ErrInvalidFaceCatalog) ||
					!strings.Contains(err.Error(), "face catalog metadata changed during load: catalog.json") {
					t.Fatalf("expected changed catalog metadata error, got %v", err)
				}
			}
			if err == nil {
				t.Fatal("expected changed catalog metadata error, got nil")
			}
			if !mutated {
				t.Fatal("expected deterministic metadata mutation hook, got no call")
			}

		})
	}

	t.Run("whitespace-only specs change", func(t *testing.T) {

		directory := makeCompleteFaceCatalogFixture(t, []string{"face-000013", "face-000037"})
		mutated := false
		faceCatalogLoadStepForTest = func(step string) {
			if mutated || step != "manifest.json" {
				return
			}
			path := filepath.Join(directory, "specs.json")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("expected captured specs metadata, got %v", err)
			}
			err = os.WriteFile(path, append(data, '\n'), 0o644)
			if err != nil {
				t.Fatalf("expected whitespace-only specs rewrite, got %v", err)
			}
			mutated = true
		}
		t.Cleanup(func() {
			faceCatalogLoadStepForTest = nil
		})

		_, err := LoadFaceCatalog(t.Context(), directory)
		faceCatalogLoadStepForTest = nil
		if err != nil {
			if !errors.Is(err, ErrInvalidFaceCatalog) ||
				!strings.Contains(err.Error(), "face catalog metadata changed during load: specs.json") {
				t.Fatalf("expected byte-changed specs metadata error, got %v", err)
			}
		}
		if err == nil {
			t.Fatal("expected byte-changed specs metadata error, got nil")
		}
		if !mutated {
			t.Fatal("expected deterministic whitespace mutation hook, got no call")
		}

	})

	t.Run("canceled before final verification", func(t *testing.T) {

		directory := makeCompleteFaceCatalogFixture(t, []string{"face-000013", "face-000037"})
		ctx, cancel := context.WithCancel(t.Context())
		t.Cleanup(cancel)
		assets := 0
		faceCatalogLoadStepForTest = func(step string) {
			if step != "asset" {
				return
			}
			assets++
			if assets == 2*len(supportedPhotoSizes) {
				cancel()
			}
		}
		t.Cleanup(func() {
			faceCatalogLoadStepForTest = nil
		})

		_, err := LoadFaceCatalog(ctx, directory)
		faceCatalogLoadStepForTest = nil
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("expected canceled final metadata verification, got %v", err)
			}
		}
		if err == nil {
			t.Fatal("expected canceled final metadata verification, got nil")
		}
		if assets != 2*len(supportedPhotoSizes) {
			t.Fatalf("expected cancellation after %d assets, got %d", 2*len(supportedPhotoSizes), assets)
		}

	})

}

// TestLayer1FaceCatalogRecordCounts verifies defensive limits and count
// consistency before record-sized structures or candidate IDs are built.
func TestLayer1FaceCatalogRecordCounts(t *testing.T) {

	for _, test := range []struct {
		name     string
		declared int
		specs    int
		manifest int
		want     string
	}{
		{"valid maximum", maxFaceCatalogRecords, maxFaceCatalogRecords, maxFaceCatalogRecords, ""},
		{"declared greater than actual", 3, 2, 3, "catalog count is 3 but specs contains 2 records"},
		{"declared less than actual", 2, 3, 2, "catalog count is 2 but specs contains 3 records"},
		{"manifest mismatch", 3, 3, 2, "catalog count is 3 but manifest contains 2 assets"},
		{
			"declared above maximum", maxFaceCatalogRecords + 1, maxFaceCatalogRecords, maxFaceCatalogRecords,
			"catalog record count 100001 exceeds limit 100000",
		},
		{
			"specs above maximum", maxFaceCatalogRecords, maxFaceCatalogRecords + 1, maxFaceCatalogRecords,
			"specs record count 100001 exceeds limit 100000",
		},
		{
			"manifest above maximum", maxFaceCatalogRecords, maxFaceCatalogRecords, maxFaceCatalogRecords + 1,
			"manifest record count 100001 exceeds limit 100000",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateFaceCatalogRecordCounts(test.declared, test.specs, test.manifest)
			if test.want == "" {
				if err != nil {
					t.Fatalf("expected valid record counts, got %v", err)
				}
				return
			}
			if err != nil {
				if !strings.Contains(err.Error(), test.want) {
					t.Fatalf("expected record-count error containing %q, got %v", test.want, err)
				}
			}
			if err == nil {
				t.Fatalf("expected record-count error containing %q, got nil", test.want)
			}
		})
	}

	t.Run("million-record declaration", func(t *testing.T) {
		directory := t.TempDir()
		var metadata fakefacegenCatalog
		err := readFakefacegenJSON(filepath.Join(fakefacegenFixtureDirectory, "catalog.json"), &metadata)
		if err != nil {
			t.Fatalf("expected source catalog metadata, got %v", err)
		}
		metadata.Count = 1_000_000
		writeFaceCatalogTestJSON(t, filepath.Join(directory, "catalog.json"), metadata)
		writeFaceCatalogTestJSON(t, filepath.Join(directory, "specs.json"), make([]fakefacegenSpec, 3))
		writeFaceCatalogTestJSON(t, filepath.Join(directory, "manifest.json"), fakefacegenManifest{
			Assets: make([]fakefacegenManifestAsset, 3),
		})

		faceCatalogMetadataCapturedForTest = func() {
			t.Fatal("expected record-count rejection before candidate enumeration, got enumeration")
		}
		t.Cleanup(func() {
			faceCatalogMetadataCapturedForTest = nil
		})
		_, err = LoadFaceCatalog(t.Context(), directory)
		faceCatalogMetadataCapturedForTest = nil
		if err != nil {
			if !errors.Is(err, ErrInvalidFaceCatalog) ||
				!strings.Contains(err.Error(), "catalog record count 1000000 exceeds limit 100000") {
				t.Fatalf("expected bounded million-record declaration error, got %v", err)
			}
		}
		if err == nil {
			t.Fatal("expected bounded million-record declaration error, got nil")
		}
	})

	t.Run("actual metadata above maximum", func(t *testing.T) {
		data := make([]byte, 0, 3*(maxFaceCatalogRecords+1)+2)
		data = append(data, '[')
		for index := 0; index <= maxFaceCatalogRecords; index++ {
			if index != 0 {
				data = append(data, ',')
			}
			data = append(data, '{', '}')
		}
		data = append(data, ']')

		_, err := countFaceCatalogSpecs("specs.json", data)
		if err != nil {
			if !strings.Contains(err.Error(), "specs contains more than 100000 records") {
				t.Fatalf("expected bounded actual-record error, got %v", err)
			}
		}
		if err == nil {
			t.Fatal("expected bounded actual-record error, got nil")
		}
	})

	t.Run("selected IDs above maximum", func(t *testing.T) {
		_, err := loadSelectedFaceCatalogFixture(t.Context(), t.TempDir(), make([]string, maxFaceCatalogRecords+1))
		if err != nil {
			if !errors.Is(err, ErrInvalidFaceCatalog) ||
				!strings.Contains(err.Error(), "invalid selected photo count 100001") {
				t.Fatalf("expected bounded selected-ID error, got %v", err)
			}
		}
		if err == nil {
			t.Fatal("expected bounded selected-ID error, got nil")
		}
	})

}

// TestLayer1SelectedFaceCatalogIdentity proves selected-fixture identity is a
// canonical set identity throughout World integration and photo selection.
func TestLayer1SelectedFaceCatalogIdentity(t *testing.T) {

	directory := makeCompleteFaceCatalogFixture(t, []string{"face-000037", "face-000013", "face-000037"})
	baseline, err := loadSelectedFaceCatalogFixture(
		t.Context(), directory, []string{"face-000001", "face-000002"},
	)
	if err != nil {
		t.Fatalf("expected baseline selected fixture, got %v", err)
	}
	expanded, err := loadSelectedFaceCatalogFixture(
		t.Context(), directory, []string{"face-000001", "face-000002", "face-000003"},
	)
	if err != nil {
		t.Fatalf("expected expanded selected fixture, got %v", err)
	}
	reordered, err := loadSelectedFaceCatalogFixture(
		t.Context(), directory, []string{"face-000003", "face-000002", "face-000001"},
	)
	if err != nil {
		t.Fatalf("expected reordered selected fixture, got %v", err)
	}
	if baseline.SourceCatalogChecksum() != expanded.SourceCatalogChecksum() {
		t.Fatalf(
			"expected equal source catalog checksums, got %x and %x",
			baseline.SourceCatalogChecksum(), expanded.SourceCatalogChecksum(),
		)
	}
	if baseline.Checksum() == expanded.Checksum() {
		t.Fatalf("expected selected-set-sensitive semantic checksum, got %x", expanded.Checksum())
	}
	if expanded.Checksum() != reordered.Checksum() {
		t.Fatalf("expected selection-order-independent checksum %x, got %x", expanded.Checksum(), reordered.Checksum())
	}
	if len(reordered.faces) != 3 || reordered.faces[0].id != "face-000001" || reordered.faces[1].id != "face-000002" ||
		reordered.faces[2].id != "face-000003" {
		t.Fatalf("expected canonical PhotoID order, got %#v", reordered.faces)
	}

	_, err = loadSelectedFaceCatalogFixture(
		t.Context(), directory, []string{"face-000001", "face-000001"},
	)
	if err != nil {
		if !errors.Is(err, ErrInvalidFaceCatalog) || !strings.Contains(err.Error(), "appears more than once") {
			t.Fatalf("expected duplicate selected-ID error, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected duplicate selected-ID error, got nil")
	}

	manifestOrderDirectory := makeCompleteFaceCatalogFixture(
		t, []string{"face-000037", "face-000013", "face-000037"},
	)
	manifestOrderBaseline, err := loadSelectedFaceCatalogFixture(
		t.Context(), manifestOrderDirectory, []string{"face-000001", "face-000002", "face-000003"},
	)
	if err != nil {
		t.Fatalf("expected manifest-order baseline, got %v", err)
	}
	manifestPath := filepath.Join(manifestOrderDirectory, "manifest.json")
	var manifest fakefacegenManifest
	err = readFakefacegenJSON(manifestPath, &manifest)
	if err != nil {
		t.Fatalf("expected manifest-order metadata, got %v", err)
	}
	for left, right := 0, len(manifest.Assets)-1; left < right; left, right = left+1, right-1 {
		manifest.Assets[left], manifest.Assets[right] = manifest.Assets[right], manifest.Assets[left]
	}
	writeFaceCatalogTestJSON(t, manifestPath, manifest)
	manifestOrderChanged, err := loadSelectedFaceCatalogFixture(
		t.Context(), manifestOrderDirectory, []string{"face-000001", "face-000002", "face-000003"},
	)
	if err != nil {
		t.Fatalf("expected reordered manifest fixture, got %v", err)
	}
	if manifestOrderBaseline.Checksum() != manifestOrderChanged.Checksum() {
		t.Fatalf(
			"expected manifest-order-independent checksum %x, got %x",
			manifestOrderBaseline.Checksum(), manifestOrderChanged.Checksum(),
		)
	}

	baselineWorld := newLayer1WorldWithCatalog(t, baseline)
	expandedWorld := newLayer1WorldWithCatalog(t, expanded)
	if baselineWorld.WorldID() != expandedWorld.WorldID() {
		t.Fatalf("expected catalog-independent WorldID %s, got %s", baselineWorld.WorldID(), expandedWorld.WorldID())
	}
	if baselineWorld.SnapshotID() == expandedWorld.SnapshotID() {
		t.Fatalf("expected selected-set-sensitive SnapshotID, got %s", expandedWorld.SnapshotID())
	}
	baselinePhotoStream, err := baselineWorld.streams.entityStream(
		personEntityKind, 1, streamPhotoSelect, baseline.dependencies[:]...,
	)
	if err != nil {
		t.Fatalf("expected baseline photo stream, got %v", err)
	}
	expandedPhotoStream, err := expandedWorld.streams.entityStream(
		personEntityKind, 1, streamPhotoSelect, expanded.dependencies[:]...,
	)
	if err != nil {
		t.Fatalf("expected expanded photo stream, got %v", err)
	}
	if baselinePhotoStream.state == expandedPhotoStream.state {
		t.Fatalf("expected selected-set-sensitive photo stream, got %x", expandedPhotoStream.state)
	}

	photoChanged := false
	for index := PersonIndex(1); index <= 1_000; index++ {
		before, err := baselineWorld.Person(index)
		if err != nil {
			t.Fatalf("expected baseline Person %d, got %v", index, err)
		}
		after, err := expandedWorld.Person(index)
		if err != nil {
			t.Fatalf("expected expanded-catalog Person %d, got %v", index, err)
		}
		if personWithoutPhoto(before) != personWithoutPhoto(after) {
			t.Fatalf("expected unchanged non-photo Person %d, got %#v and %#v", index, before, after)
		}
		if before.PhotoID != after.PhotoID {
			photoChanged = true
			break
		}
	}
	if !photoChanged {
		t.Fatal("expected a selected-set-dependent PhotoID, got none")
	}

}

// TestLayer1FaceCatalogAssetIdentity proves verified asset bytes feed semantic
// catalog identity, SnapshotID, public paths, and only the photo stream.
func TestLayer1FaceCatalogAssetIdentity(t *testing.T) {

	directory := copyFakefacegenFixture(t)
	baseline, err := loadSelectedFaceCatalogFixture(
		t.Context(), directory, []string{"face-000013", "face-000037"},
	)
	if err != nil {
		t.Fatalf("expected baseline selected fixture, got %v", err)
	}
	beforeAsset, err := baseline.Asset("face-000013", PhotoSize64)
	if err != nil {
		t.Fatalf("expected baseline photo asset, got %v", err)
	}

	copyFaceCatalogTestAsset(t, directory, "face-000013", "face-000037", PhotoSize64)
	changed, err := loadSelectedFaceCatalogFixture(
		t.Context(), directory, []string{"face-000013", "face-000037"},
	)
	if err != nil {
		t.Fatalf("expected changed selected fixture, got %v", err)
	}
	afterAsset, err := changed.Asset("face-000013", PhotoSize64)
	if err != nil {
		t.Fatalf("expected changed photo asset, got %v", err)
	}
	if baseline.SourceCatalogChecksum() != changed.SourceCatalogChecksum() {
		t.Fatalf(
			"expected equal source catalog checksums, got %x and %x",
			baseline.SourceCatalogChecksum(), changed.SourceCatalogChecksum(),
		)
	}
	if baseline.Checksum() == changed.Checksum() {
		t.Fatalf("expected asset-sensitive semantic checksum, got %x", changed.Checksum())
	}
	if beforeAsset.ContentSHA256 == afterAsset.ContentSHA256 {
		t.Fatalf("expected changed asset checksum, got %s", afterAsset.ContentSHA256)
	}
	if beforeAsset.Path == afterAsset.Path {
		t.Fatalf("expected semantic-checksum-sensitive public path, got %q", afterAsset.Path)
	}
	changedChecksum := changed.Checksum()
	if !strings.Contains(afterAsset.Path, hex.EncodeToString(changedChecksum[:])) {
		t.Fatalf("expected semantic checksum in public path, got %q", afterAsset.Path)
	}

	baselineWorld := newLayer1WorldWithCatalog(t, baseline)
	changedWorld := newLayer1WorldWithCatalog(t, changed)
	if baselineWorld.WorldID() != changedWorld.WorldID() {
		t.Fatalf("expected asset-independent WorldID %s, got %s", baselineWorld.WorldID(), changedWorld.WorldID())
	}
	if baselineWorld.SnapshotID() == changedWorld.SnapshotID() {
		t.Fatalf("expected asset-sensitive SnapshotID, got %s", changedWorld.SnapshotID())
	}
	before, err := baselineWorld.Person(1_537_291)
	if err != nil {
		t.Fatalf("expected baseline Person, got %v", err)
	}
	after, err := changedWorld.Person(1_537_291)
	if err != nil {
		t.Fatalf("expected changed-asset Person, got %v", err)
	}
	if before.PhotoID != after.PhotoID {
		t.Fatalf("expected unchanged PhotoID %q, got %q", before.PhotoID, after.PhotoID)
	}
	if personWithoutPhoto(before) != personWithoutPhoto(after) {
		t.Fatalf("expected unchanged non-photo Person, got %#v and %#v", before, after)
	}

}

// TestLayer1FaceCatalogLoadModes verifies complete production validation and
// the deliberately incomplete selected-fixture exception.
func TestLayer1FaceCatalogLoadModes(t *testing.T) {

	selected, err := loadSelectedFaceCatalogFixture(
		t.Context(), fakefacegenFixtureDirectory, []string{"face-000013", "face-000037"},
	)
	if err != nil {
		t.Fatalf("expected authentic selected fixture, got %v", err)
	}
	if selected.mode != faceCatalogSnapshotModeSelectedFixture {
		t.Fatalf("expected selected-fixture mode, got %q", selected.mode)
	}
	_, err = LoadFaceCatalog(t.Context(), fakefacegenFixtureDirectory)
	if err != nil {
		if !errors.Is(err, ErrInvalidFaceCatalog) {
			t.Fatalf("expected incomplete production catalog error, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected incomplete production catalog error, got nil")
	}

	completeDirectory := makeCompleteFaceCatalogFixture(t, []string{"face-000037", "face-000013", "face-000037"})
	complete, err := LoadFaceCatalog(t.Context(), completeDirectory)
	if err != nil {
		t.Fatalf("expected complete production catalog, got %v", err)
	}
	if complete.mode != faceCatalogSnapshotModeComplete || len(complete.faces) != 3 {
		t.Fatalf("expected complete three-face catalog, got mode %q and %d faces", complete.mode, len(complete.faces))
	}
	selectedComplete, err := loadSelectedFaceCatalogFixture(
		t.Context(), completeDirectory, []string{"face-000001", "face-000002", "face-000003"},
	)
	if err != nil {
		t.Fatalf("expected complete data in selected-fixture mode, got %v", err)
	}
	if complete.SourceCatalogChecksum() != selectedComplete.SourceCatalogChecksum() {
		t.Fatalf(
			"expected equal source checksums across modes, got %x and %x",
			complete.SourceCatalogChecksum(), selectedComplete.SourceCatalogChecksum(),
		)
	}
	if complete.Checksum() == selectedComplete.Checksum() {
		t.Fatalf("expected snapshot-mode-sensitive semantic checksum, got %x", complete.Checksum())
	}

	for _, test := range []struct {
		name string
		size PhotoSize
	}{
		{"missing derivative", PhotoSize64},
		{"missing master", PhotoSize1024},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := makeCompleteFaceCatalogFixture(t, []string{"face-000037", "face-000013", "face-000037"})
			path := faceCatalogTestAssetPath(directory, "face-000003", test.size)
			err := os.Remove(path)
			if err != nil {
				t.Fatalf("expected generated asset removal, got %v", err)
			}
			_, err = LoadFaceCatalog(t.Context(), directory)
			if err != nil {
				if !errors.Is(err, ErrInvalidFaceCatalog) {
					t.Fatalf("expected incomplete generated-record error, got %v", err)
				}
			}
			if err == nil {
				t.Fatal("expected incomplete generated-record error, got nil")
			}
		})
	}

	t.Run("corrupt generated asset", func(t *testing.T) {
		directory := makeCompleteFaceCatalogFixture(t, []string{"face-000037", "face-000013", "face-000037"})
		path := faceCatalogTestAssetPath(directory, "face-000003", PhotoSize64)
		err := os.WriteFile(path, []byte("not a WebP image"), 0o644)
		if err != nil {
			t.Fatalf("expected corrupt generated asset write, got %v", err)
		}
		_, err = LoadFaceCatalog(t.Context(), directory)
		if err != nil {
			if !errors.Is(err, ErrInvalidFaceCatalog) || !strings.Contains(err.Error(), "not image/webp") {
				t.Fatalf("expected corrupt generated-record error, got %v", err)
			}
		}
		if err == nil {
			t.Fatal("expected corrupt generated-record error, got nil")
		}
	})

	t.Run("unknown status", func(t *testing.T) {
		directory := makeCompleteFaceCatalogFixture(t, []string{"face-000037", "face-000013", "face-000037"})
		mutateFakefacegenManifest(t, directory, "face-000003", func(asset *fakefacegenManifestAsset) {
			asset.Status = "unknown"
		})
		_, err := LoadFaceCatalog(t.Context(), directory)
		if err != nil {
			if !errors.Is(err, ErrInvalidFaceCatalog) || !strings.Contains(err.Error(), "unsupported status") {
				t.Fatalf("expected unknown-status production error, got %v", err)
			}
		}
		if err == nil {
			t.Fatal("expected unknown-status production error, got nil")
		}
	})

	statusCatalogs := map[fakefacegenSourceStatus]*FaceCatalog{}
	for _, status := range []fakefacegenSourceStatus{fakefacegenStatusFailed, fakefacegenStatusMissing} {
		t.Run(string(status), func(t *testing.T) {
			directory := makeCompleteFaceCatalogFixture(t, []string{"face-000037", "face-000013", "face-000037"})
			mutateFakefacegenManifest(t, directory, "face-000003", func(asset *fakefacegenManifestAsset) {
				asset.Status = status
			})
			for _, size := range supportedPhotoSizes {
				path := faceCatalogTestAssetPath(directory, "face-000003", size)
				err := os.Remove(path)
				if err != nil {
					t.Fatalf("expected excluded asset removal, got %v", err)
				}
			}
			catalog, err := LoadFaceCatalog(t.Context(), directory)
			if err != nil {
				t.Fatalf("expected %s record without assets, got %v", status, err)
			}
			if catalog.faces[2].status != status || !catalog.faces[2].excluded {
				t.Fatalf("expected excluded %s record, got %#v", status, catalog.faces[2])
			}
			_, err = catalog.Asset("face-000003", PhotoSize64)
			if err != nil {
				if !errors.Is(err, ErrInvalidPhotoAsset) {
					t.Fatalf("expected unavailable %s asset error, got %v", status, err)
				}
			}
			if err == nil {
				t.Fatalf("expected unavailable %s asset error, got nil", status)
			}
			selectedCatalog, err := loadSelectedFaceCatalogFixture(
				t.Context(), directory, []string{"face-000001", "face-000002", "face-000003"},
			)
			if err != nil {
				t.Fatalf("expected selected-fixture %s record without assets, got %v", status, err)
			}
			if selectedCatalog.faces[2].status != status || !selectedCatalog.faces[2].excluded {
				t.Fatalf("expected selected-fixture excluded %s record, got %#v", status, selectedCatalog.faces[2])
			}
			statusCatalogs[status] = catalog
		})
	}
	if statusCatalogs[fakefacegenStatusFailed].Checksum() == statusCatalogs[fakefacegenStatusMissing].Checksum() {
		t.Fatalf("expected status-sensitive semantic checksum, got %x", statusCatalogs[fakefacegenStatusMissing].Checksum())
	}

}

// TestLayer1SelectedFixtureValidatesEveryAsset verifies all required assets of
// each selected generated record remain mandatory in fixture mode.
func TestLayer1SelectedFixtureValidatesEveryAsset(t *testing.T) {
	for _, size := range supportedPhotoSizes {
		t.Run(strconv.Itoa(int(size)), func(t *testing.T) {
			directory := copyFakefacegenFixture(t)
			path := faceCatalogTestAssetPath(directory, "face-000013", size)
			err := os.Remove(path)
			if err != nil {
				t.Fatalf("expected selected asset removal, got %v", err)
			}
			_, err = loadSelectedFaceCatalogFixture(
				t.Context(), directory, []string{"face-000013", "face-000037"},
			)
			if err != nil {
				if !errors.Is(err, ErrInvalidFaceCatalog) {
					t.Fatalf("expected selected generated-asset error, got %v", err)
				}
			}
			if err == nil {
				t.Fatal("expected selected generated-asset error, got nil")
			}
		})
	}
}

// TestLayer1FaceCatalogContext verifies filesystem loading observes caller
// cancellation before beginning its metadata reads.
func TestLayer1FaceCatalogContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := loadSelectedFaceCatalogFixture(
		ctx, fakefacegenFixtureDirectory, []string{"face-000013", "face-000037"},
	)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected canceled fixture load, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected canceled fixture load, got nil")
	}
}

// TestLayer1FaceCatalogConfiguration verifies catalog identity changes only
// snapshot/photo configuration and never Person identity.
func TestLayer1FaceCatalogConfiguration(t *testing.T) {

	baselineCatalog := mustLayer1Catalog(t)
	directory := copyFakefacegenFixture(t)
	path := filepath.Join(directory, "catalog.json")
	var metadata fakefacegenCatalog
	err := readFakefacegenJSON(path, &metadata)
	if err != nil {
		t.Fatalf("expected copied catalog metadata, got %v", err)
	}
	metadata.Seed++
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatalf("expected changed catalog encoding, got %v", err)
	}
	err = os.WriteFile(path, append(data, '\n'), 0o644)
	if err != nil {
		t.Fatalf("expected changed catalog write, got %v", err)
	}
	changedCatalog, err := loadSelectedFaceCatalogFixture(
		t.Context(), directory, []string{"face-000013", "face-000037"},
	)
	if err != nil {
		t.Fatalf("expected changed FaceCatalog, got %v", err)
	}
	if changedCatalog.Checksum() == baselineCatalog.Checksum() {
		t.Fatalf("expected changed catalog checksum, got %x", changedCatalog.Checksum())
	}

	namespace, err := NewIdentityNamespace("krenalis-demo", 1)
	if err != nil {
		t.Fatalf("expected identity namespace, got %v", err)
	}
	baseline := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	changed, err := NewWorld(WorldConfig{
		IdentityNamespace: namespace,
		WorldSeed:         layer1TestSeed,
		ReferenceDate:     layer1TestReferenceDate,
		FaceCatalog:       changedCatalog,
	})
	if err != nil {
		t.Fatalf("expected changed-catalog World, got %v", err)
	}
	if baseline.WorldID() != changed.WorldID() {
		t.Fatalf("expected catalog-independent WorldID %s, got %s", baseline.WorldID(), changed.WorldID())
	}
	if baseline.SnapshotID() == changed.SnapshotID() {
		t.Fatalf("expected catalog-dependent SnapshotID, got %s", changed.SnapshotID())
	}
	baselineChecksum := baselineCatalog.Checksum()
	baselinePhotoStream, err := baseline.streams.entityStream(
		personEntityKind,
		42,
		streamPhotoSelect,
		mustStringComponent(t, faceCatalogVersionComponentName, baselineCatalog.Version()),
		mustBytesComponent(t, faceCatalogSHA256ComponentName, baselineChecksum[:]),
	)
	if err != nil {
		t.Fatalf("expected baseline photo stream, got %v", err)
	}
	changedChecksum := changedCatalog.Checksum()
	changedPhotoStream, err := changed.streams.entityStream(
		personEntityKind,
		42,
		streamPhotoSelect,
		mustStringComponent(t, faceCatalogVersionComponentName, changedCatalog.Version()),
		mustBytesComponent(t, faceCatalogSHA256ComponentName, changedChecksum[:]),
	)
	if err != nil {
		t.Fatalf("expected changed photo stream, got %v", err)
	}
	if baselinePhotoStream.state == changedPhotoStream.state {
		t.Fatalf("expected catalog-dependent photo stream, got %x", changedPhotoStream.state)
	}
	before, err := baseline.Person(42)
	if err != nil {
		t.Fatalf("expected baseline Person, got %v", err)
	}
	after, err := changed.Person(42)
	if err != nil {
		t.Fatalf("expected changed-catalog Person, got %v", err)
	}
	if before.ID != after.ID {
		t.Fatalf("expected catalog-independent Person ID %q, got %q", before.ID, after.ID)
	}
	before.PhotoID = ""
	after.PhotoID = ""
	if before != after {
		t.Fatalf("expected catalog isolation outside PhotoID, got %#v and %#v", before, after)
	}

}

// TestLayer1FaceCatalogErrors verifies exclusions, unknown statuses, and hard
// generated-asset failures at the public catalog boundary.
func TestLayer1FaceCatalogErrors(t *testing.T) {

	t.Run("no female photo", func(t *testing.T) {
		directory := copyFakefacegenFixture(t)
		mutateFakefacegenManifest(t, directory, "face-000013", func(asset *fakefacegenManifestAsset) {
			asset.Status = fakefacegenStatusFailed
		})
		catalog, err := loadSelectedFaceCatalogFixture(
			t.Context(), directory, []string{"face-000013", "face-000037"},
		)
		if err != nil {
			t.Fatalf("expected failed photo exclusion, got %v", err)
		}
		_, err = catalog.Asset("face-000013", PhotoSize64)
		if err != nil {
			if !errors.Is(err, ErrInvalidPhotoAsset) {
				t.Fatalf("expected excluded photo asset error, got %v", err)
			}
		}
		if err == nil {
			t.Fatal("expected excluded photo asset error, got nil")
		}
		namespace, err := NewIdentityNamespace("krenalis-demo", 1)
		if err != nil {
			t.Fatalf("expected identity namespace, got %v", err)
		}
		_, err = NewWorld(WorldConfig{
			IdentityNamespace: namespace,
			WorldSeed:         layer1TestSeed,
			ReferenceDate:     layer1TestReferenceDate,
			FaceCatalog:       catalog,
		})
		if err != nil {
			if !errors.Is(err, ErrNoCompatiblePhoto) {
				t.Fatalf("expected no compatible photo, got %v", err)
			}
		}
		if err == nil {
			t.Fatal("expected no compatible photo, got nil")
		}
	})

	t.Run("unknown status", func(t *testing.T) {
		directory := copyFakefacegenFixture(t)
		mutateFakefacegenManifest(t, directory, "face-000013", func(asset *fakefacegenManifestAsset) {
			asset.Status = "unknown"
		})
		_, err := loadSelectedFaceCatalogFixture(t.Context(), directory, []string{"face-000013"})
		if err != nil {
			if !errors.Is(err, ErrInvalidFaceCatalog) || !strings.Contains(err.Error(), "unsupported status") {
				t.Fatalf("expected unknown-status catalog error, got %v", err)
			}
		}
		if err == nil {
			t.Fatal("expected unknown-status catalog error, got nil")
		}
	})

	t.Run("corrupt generated asset", func(t *testing.T) {
		directory := copyFakefacegenFixture(t)
		path := filepath.Join(directory, "derived", "64", "face-000013.webp")
		err := os.WriteFile(path, []byte("not a WebP image"), 0o644)
		if err != nil {
			t.Fatalf("expected corrupt asset write, got %v", err)
		}
		_, err = loadSelectedFaceCatalogFixture(t.Context(), directory, []string{"face-000013"})
		if err != nil {
			if !errors.Is(err, ErrInvalidFaceCatalog) || !strings.Contains(err.Error(), "not image/webp") {
				t.Fatalf("expected corrupt generated-asset error, got %v", err)
			}
		}
		if err == nil {
			t.Fatal("expected corrupt generated-asset error, got nil")
		}
	})

}

// TestLayer1PhotoHTTP verifies normalized metadata and exact GET/HEAD/404/405
// behavior over immutable verified bytes.
func TestLayer1PhotoHTTP(t *testing.T) {

	catalog := mustLayer1Catalog(t)
	world := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	checksum := catalog.Checksum()
	if catalog.Version() != fakefacegenV1SourceVersion {
		t.Fatalf("expected catalog version %q, got %q", fakefacegenV1SourceVersion, catalog.Version())
	}

	paths := map[string]struct{}{}
	for _, photoID := range []string{"face-000013", "face-000037"} {
		for _, size := range supportedPhotoSizes {
			asset, err := catalog.Asset(photoID, size)
			if err != nil {
				t.Fatalf("expected asset %s/%d, got %v", photoID, size, err)
			}
			if asset.PhotoID != photoID || asset.Size != size || asset.Width != uint32(size) || asset.Height != uint32(size) {
				t.Fatalf("expected normalized asset identity/dimensions %s/%d, got %#v", photoID, size, asset)
			}
			if asset.CatalogVersion != catalog.Version() || asset.CatalogSHA256 != hex.EncodeToString(checksum[:]) {
				t.Fatalf("expected normalized catalog identity, got %#v", asset)
			}
			if strings.Contains(asset.Path, fakefacegenFixtureDirectory) || strings.Contains(asset.Path, "masters/") ||
				strings.Contains(asset.Path, "derived/") {
				t.Fatalf("expected public path without filesystem locator, got %q", asset.Path)
			}
			if _, exists := paths[asset.Path]; exists {
				t.Fatalf("expected content-addressed path for %s/%d, got duplicate %q", photoID, size, asset.Path)
			}
			paths[asset.Path] = struct{}{}
		}
	}

	asset, err := world.PhotoAsset("face-000013", PhotoSize64)
	if err != nil {
		t.Fatalf("expected public photo asset, got %v", err)
	}
	for _, request := range []struct {
		photoID string
		size    PhotoSize
	}{
		{"face-999999", PhotoSize64},
		{"face-000013", PhotoSize(63)},
	} {
		_, err := catalog.Asset(request.photoID, request.size)
		if err != nil {
			if !errors.Is(err, ErrInvalidPhotoAsset) {
				t.Fatalf("expected invalid photo asset for %s/%d, got %v", request.photoID, request.size, err)
			}
		}
		if err == nil {
			t.Fatalf("expected invalid photo asset for %s/%d, got nil", request.photoID, request.size)
		}
	}
	record := catalog.assets[photoAssetKey{photoID: asset.PhotoID, size: asset.Size}]
	handler := world.PhotoHandler()

	getRequest := httptest.NewRequest(http.MethodGet, asset.Path, nil)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected GET status 200, got %d", getResponse.Code)
	}
	if getResponse.Header().Get("Content-Type") != "image/webp" {
		t.Fatalf("expected GET Content-Type image/webp, got %q", getResponse.Header().Get("Content-Type"))
	}
	if getResponse.Header().Get("Content-Length") != strconv.FormatInt(asset.ContentLength, 10) {
		t.Fatalf("expected GET Content-Length %d, got %q", asset.ContentLength, getResponse.Header().Get("Content-Length"))
	}
	if !bytes.Equal(getResponse.Body.Bytes(), record.data) {
		t.Fatalf("expected GET body with %d verified bytes, got %d", len(record.data), getResponse.Body.Len())
	}

	headRequest := httptest.NewRequest(http.MethodHead, asset.Path, nil)
	headResponse := httptest.NewRecorder()
	handler.ServeHTTP(headResponse, headRequest)
	if headResponse.Code != http.StatusOK {
		t.Fatalf("expected HEAD status 200, got %d", headResponse.Code)
	}
	if headResponse.Header().Get("Content-Type") != "image/webp" ||
		headResponse.Header().Get("Content-Length") != strconv.FormatInt(asset.ContentLength, 10) {
		t.Fatalf("expected HEAD asset headers, got %v", headResponse.Header())
	}
	if headResponse.Body.Len() != 0 {
		t.Fatalf("expected empty HEAD body, got %d bytes", headResponse.Body.Len())
	}

	for _, path := range []string{
		"/unknown", strings.Replace(asset.Path, asset.CatalogSHA256, strings.Repeat("0", 64), 1),
		strings.Replace(asset.Path, asset.PhotoID, "face-999999", 1), strings.Replace(asset.Path, "/64/", "/63/", 1),
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for %q, got %d", path, response.Code)
		}
	}

	postRequest := httptest.NewRequest(http.MethodPost, asset.Path, nil)
	postResponse := httptest.NewRecorder()
	handler.ServeHTTP(postResponse, postRequest)
	if postResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected POST status 405, got %d", postResponse.Code)
	}
	if postResponse.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("expected Allow GET, HEAD, got %q", postResponse.Header().Get("Allow"))
	}

}

// TestLayer1PhotoSelection verifies gender, anchor-age nearest matching,
// deterministic tie selection, ReferenceDate invariance, and PhotoID reuse.
func TestLayer1PhotoSelection(t *testing.T) {

	world := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	reversed, err := loadSelectedFaceCatalogFixture(
		t.Context(), fakefacegenFixtureDirectory, []string{"face-000037", "face-000013"},
	)
	if err != nil {
		t.Fatalf("expected reversed-load catalog, got %v", err)
	}
	if reversed.faces[0].id != "face-000013" || reversed.faces[1].id != "face-000037" {
		t.Fatalf("expected stable catalog ID order, got %#v", reversed.faces)
	}
	for _, test := range []struct {
		gender Gender
		age    int
		want   string
	}{
		{GenderFemale, 75, "face-000013"},
		{GenderMale, 21, "face-000037"},
	} {
		photoID, err := world.faceCatalog.selectPhoto(world.streams, 42, test.gender, test.age)
		if err != nil {
			t.Fatalf("expected gender-compatible photo for %s/%d, got %v", test.gender, test.age, err)
		}
		if photoID != test.want {
			t.Fatalf("expected PhotoID %q for %s/%d, got %q", test.want, test.gender, test.age, photoID)
		}
	}

	catalog := addTestPhotoFace(t, world.faceCatalog, "face-test-060", GenderMale, 60, "face-000037")
	closest, err := catalog.selectPhoto(world.streams, 42, GenderMale, 59)
	if err != nil {
		t.Fatalf("expected nearest-age photo, got %v", err)
	}
	if closest != "face-test-060" {
		t.Fatalf("expected nearest-age PhotoID face-test-060, got %q", closest)
	}
	catalog = addTestPhotoFace(t, catalog, "face-test-062", GenderMale, 62, "face-000037")

	seenTies := map[string]struct{}{}
	for index := PersonIndex(1); index <= 1_000; index++ {
		first, err := catalog.selectPhoto(world.streams, index, GenderMale, 61)
		if err != nil {
			t.Fatalf("expected tie-selected photo for %d, got %v", index, err)
		}
		second, err := catalog.selectPhoto(world.streams, index, GenderMale, 61)
		if err != nil {
			t.Fatalf("expected repeated tie-selected photo for %d, got %v", index, err)
		}
		if first != second {
			t.Fatalf("expected deterministic tie selection %q, got %q", first, second)
		}
		seenTies[first] = struct{}{}
	}
	if _, exists := seenTies["face-test-060"]; !exists {
		t.Fatalf("expected first tied PhotoID reachable, got %v", seenTies)
	}
	if _, exists := seenTies["face-test-062"]; !exists {
		t.Fatalf("expected second tied PhotoID reachable, got %v", seenTies)
	}

	firstPerson, err := world.Person(1)
	if err != nil {
		t.Fatalf("expected first reusable-photo Person, got %v", err)
	}
	secondPerson, err := world.Person(2)
	if err != nil {
		t.Fatalf("expected second reusable-photo Person, got %v", err)
	}
	if firstPerson.PhotoID != secondPerson.PhotoID {
		t.Fatalf("expected reusable PhotoID for fixture males, got %q/%q", firstPerson.PhotoID, secondPerson.PhotoID)
	}
	later := mustLayer1World(t, "2030-12-31", layer1TestSeed)
	laterPerson, err := later.Person(1)
	if err != nil {
		t.Fatalf("expected later-reference Person, got %v", err)
	}
	if laterPerson.PhotoID != firstPerson.PhotoID {
		t.Fatalf("expected ReferenceDate-independent PhotoID %q, got %q", firstPerson.PhotoID, laterPerson.PhotoID)
	}

}

func addTestPhotoFace(t *testing.T, source *FaceCatalog, id string, gender Gender, age int, assetSourceID string) *FaceCatalog {
	t.Helper()
	clone := *source
	clone.faces = append([]photoFace(nil), source.faces...)
	clone.assets = make(map[photoAssetKey]photoAssetRecord, len(source.assets)+len(supportedPhotoSizes))
	for key, record := range source.assets {
		clone.assets[key] = record
	}
	clone.paths = make(map[string]photoAssetRecord, len(source.paths)+len(supportedPhotoSizes))
	for path, record := range source.paths {
		clone.paths[path] = record
	}
	clone.faces = append(clone.faces, photoFace{
		id: id, gender: gender, requestedAge: age, status: fakefacegenStatusGenerated,
	})
	for _, size := range supportedPhotoSizes {
		record, exists := source.assets[photoAssetKey{photoID: assetSourceID, size: size}]
		if !exists {
			t.Fatalf("expected source asset %s/%d, got none", assetSourceID, size)
		}
		record.metadata.PhotoID = id
		record.metadata.Path = publicPhotoPath(clone.checksum, size, id)
		clone.assets[photoAssetKey{photoID: id, size: size}] = record
		clone.paths[record.metadata.Path] = record
	}
	return &clone
}

func copyFaceCatalogTestAsset(t *testing.T, directory, destinationID, sourceID string, size PhotoSize) {
	t.Helper()
	source := faceCatalogTestAssetPath(fakefacegenFixtureDirectory, sourceID, size)
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("expected source photo asset %s/%d, got %v", sourceID, size, err)
	}
	destination := faceCatalogTestAssetPath(directory, destinationID, size)
	err = os.MkdirAll(filepath.Dir(destination), 0o755)
	if err != nil {
		t.Fatalf("expected destination photo directory, got %v", err)
	}
	err = os.WriteFile(destination, data, 0o644)
	if err != nil {
		t.Fatalf("expected copied photo asset %s/%d, got %v", destinationID, size, err)
	}
}

func copyFaceCatalogTestFile(t *testing.T, source, destination string) {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("expected source test file %q, got %v", source, err)
	}
	err = os.WriteFile(destination, data, 0o644)
	if err != nil {
		t.Fatalf("expected copied test file %q, got %v", destination, err)
	}
}

func faceCatalogTestAssetPath(directory, photoID string, size PhotoSize) string {
	if size == PhotoSize1024 {
		return filepath.Join(directory, "masters", photoID+".webp")
	}
	return filepath.Join(directory, "derived", strconv.Itoa(int(size)), photoID+".webp")
}

func makeCompleteFaceCatalogFixture(t *testing.T, sourceIDs []string) string {

	t.Helper()
	directory := t.TempDir()

	var metadata fakefacegenCatalog
	err := readFakefacegenJSON(filepath.Join(fakefacegenFixtureDirectory, "catalog.json"), &metadata)
	if err != nil {
		t.Fatalf("expected source catalog metadata, got %v", err)
	}
	metadata.Count = len(sourceIDs)

	var sourceSpecs []fakefacegenSpec
	err = readFakefacegenJSON(filepath.Join(fakefacegenFixtureDirectory, "specs.json"), &sourceSpecs)
	if err != nil {
		t.Fatalf("expected source face specs, got %v", err)
	}
	var sourceManifest fakefacegenManifest
	err = readFakefacegenJSON(filepath.Join(fakefacegenFixtureDirectory, "manifest.json"), &sourceManifest)
	if err != nil {
		t.Fatalf("expected source face manifest, got %v", err)
	}

	specsByID := make(map[string]fakefacegenSpec, len(sourceSpecs))
	for _, spec := range sourceSpecs {
		specsByID[spec.ID] = spec
	}
	manifestByID := make(map[string]fakefacegenManifestAsset, len(sourceManifest.Assets))
	for _, asset := range sourceManifest.Assets {
		manifestByID[asset.ID] = asset
	}

	specs := make([]fakefacegenSpec, len(sourceIDs))
	manifest := fakefacegenManifest{Assets: make([]fakefacegenManifestAsset, len(sourceIDs))}
	for index, sourceID := range sourceIDs {

		destinationID := fmt.Sprintf("face-%06d", index+1)
		spec, exists := specsByID[sourceID]
		if !exists {
			t.Fatalf("expected source spec %q, got none", sourceID)
		}
		spec.ID = destinationID
		specs[index] = spec

		asset, exists := manifestByID[sourceID]
		if !exists {
			t.Fatalf("expected source manifest asset %q, got none", sourceID)
		}
		asset.ID = destinationID
		asset.SpecID = destinationID
		asset.File = filepath.ToSlash(filepath.Join("masters", destinationID+".webp"))
		manifest.Assets[index] = asset
		for _, size := range supportedPhotoSizes {
			copyFaceCatalogTestAsset(t, directory, destinationID, sourceID, size)
		}

	}

	writeFaceCatalogTestJSON(t, filepath.Join(directory, "catalog.json"), metadata)
	writeFaceCatalogTestJSON(t, filepath.Join(directory, "specs.json"), specs)
	writeFaceCatalogTestJSON(t, filepath.Join(directory, "manifest.json"), manifest)

	return directory
}

func makeFaceCatalogMetadataStabilityFixtures(t *testing.T) (string, string) {

	t.Helper()
	directory := makeCompleteFaceCatalogFixture(t, []string{"face-000013", "face-000037"})
	replacement := makeCompleteFaceCatalogFixture(t, []string{"face-000013", "face-000037"})

	specsPath := filepath.Join(directory, "specs.json")
	var specs []fakefacegenSpec
	err := readFakefacegenJSON(specsPath, &specs)
	if err != nil {
		t.Fatalf("expected generation A specs metadata, got %v", err)
	}
	specs[0].Sex = string(fakefacegenGenderMale)
	specs[0].Age = 33
	writeFaceCatalogTestJSON(t, specsPath, specs)

	specsPath = filepath.Join(replacement, "specs.json")
	err = readFakefacegenJSON(specsPath, &specs)
	if err != nil {
		t.Fatalf("expected generation B specs metadata, got %v", err)
	}
	specs[0].Sex = string(fakefacegenGenderFemale)
	specs[0].Age = 44
	writeFaceCatalogTestJSON(t, specsPath, specs)

	catalogPath := filepath.Join(replacement, "catalog.json")
	var metadata fakefacegenCatalog
	err = readFakefacegenJSON(catalogPath, &metadata)
	if err != nil {
		t.Fatalf("expected replacement catalog metadata, got %v", err)
	}
	metadata.Seed = 726_382
	writeFaceCatalogTestJSON(t, catalogPath, metadata)
	mutateFakefacegenManifest(t, replacement, "face-000001", func(asset *fakefacegenManifestAsset) {
		asset.Status = fakefacegenStatusFailed
	})

	baseline, err := LoadFaceCatalog(t.Context(), directory)
	if err != nil {
		t.Fatalf("expected valid generation A, got %v", err)
	}
	changed, err := LoadFaceCatalog(t.Context(), replacement)
	if err != nil {
		t.Fatalf("expected valid generation B, got %v", err)
	}
	if len(baseline.faces) != 2 || baseline.faces[0].gender != GenderMale || baseline.faces[0].requestedAge != 33 ||
		baseline.faces[0].status != fakefacegenStatusGenerated {
		t.Fatalf("expected generation A male/33/generated first face, got %#v", baseline.faces)
	}
	if len(changed.faces) != 2 || changed.faces[0].gender != GenderFemale || changed.faces[0].requestedAge != 44 ||
		changed.faces[0].status != fakefacegenStatusFailed {
		t.Fatalf("expected generation B female/44/failed first face, got %#v", changed.faces)
	}
	if baseline.SourceCatalogChecksum() == changed.SourceCatalogChecksum() || baseline.Checksum() == changed.Checksum() {
		t.Fatalf(
			"expected semantically distinct generations, got source/semantic checksums %x/%x and %x/%x",
			baseline.SourceCatalogChecksum(), baseline.Checksum(), changed.SourceCatalogChecksum(), changed.Checksum(),
		)
	}

	return directory, replacement
}

func newLayer1WorldWithCatalog(t *testing.T, catalog *FaceCatalog) *World {
	t.Helper()
	namespace, err := NewIdentityNamespace("krenalis-demo", 1)
	if err != nil {
		t.Fatalf("expected identity namespace, got %v", err)
	}
	world, err := NewWorld(WorldConfig{
		IdentityNamespace: namespace,
		WorldSeed:         layer1TestSeed,
		ReferenceDate:     layer1TestReferenceDate,
		FaceCatalog:       catalog,
	})
	if err != nil {
		t.Fatalf("expected catalog World, got %v", err)
	}
	return world
}

func personWithoutPhoto(person Person) Person {
	person.PhotoID = ""
	return person
}

func publishFaceCatalogTestMetadata(t *testing.T, source, destination string) {
	t.Helper()
	for _, name := range []string{"catalog.json", "specs.json", "manifest.json"} {
		copyFaceCatalogTestFile(t, filepath.Join(source, name), filepath.Join(destination, name))
	}
}

func writeFaceCatalogTestJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("expected test JSON encoding, got %v", err)
	}
	err = os.WriteFile(path, append(data, '\n'), 0o644)
	if err != nil {
		t.Fatalf("expected test JSON write, got %v", err)
	}
}
