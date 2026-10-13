// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ConvertExternalPhotoCatalogData converts the complete old external v1
// catalog offline. Release v1 is copied into selection.version; this chooses a
// new discriminant instead of preserving fakefacegen-v1. Image integrity is
// omitted because the old external catalog does not declare it.
// It returns ErrNoCompatiblePhoto for missing Italian gender coverage.
// It returns context errors between reads and tools/json SyntaxError for
// malformed tokens. Callers must interrupt blocking reads themselves.
func ConvertExternalPhotoCatalogData(ctx context.Context, reader io.Reader) (*PhotoCatalogData, error) {

	var old struct {
		Format        uint32 `json:"format"`
		Release       string `json:"release"`
		SelectionSeed string `json:"selectionSeed"`
		Faces         []struct {
			ID           string `json:"id"`
			Gender       string `json:"gender"`
			RequestedAge uint32 `json:"requestedAge"`
		} `json:"faces" max:"100000"`
	}
	err := decodeInternationalJSON(ctx, reader, &old)
	if err != nil {
		return nil, err
	}
	if old.Format != 1 || old.Release != "v1" {
		return nil, fmt.Errorf("unsupported original external photo catalog")
	}
	document := internationalPhotos{
		Schema: "sw-photo-catalog-v1", Release: "v1-json1",
		GenderTaxonomy: internationalTaxonomyFor("sw-l1-gender-v1", []Gender{GenderFemale, GenderMale}),
		Selection:      internationalSelection{Version: old.Release, Seed: old.SelectionSeed},
	}
	for _, face := range old.Faces {
		if face.RequestedAge < 21 || face.RequestedAge > 75 {
			return nil, fmt.Errorf("invalid original photo age")
		}
		document.Faces = append(document.Faces, internationalFace{
			ID: face.ID, Gender: face.Gender, RequestedAge: face.RequestedAge,
			Renditions: []internationalRendition{{
				Width: 256, Height: 256, Path: "synthetic-world/photos/" + old.Release + "/256/" + face.ID + ".webp",
			}},
		})
	}
	data, err := preparePhotoCatalogData(document)
	if err != nil {
		return nil, err
	}
	_, err = prepareItalianPhotoCatalog(data, document.GenderTaxonomy)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// ConvertLocalPhotoCatalogData verifies a complete fakefacegen snapshot offline
// and converts its usable faces and five verified renditions to v1-json1.
// It uses selectionSeed as selection.seed and release v1 as selection.version.
// The seed must encode a nonzero 32-byte value as 64 lowercase hex digits;
// it is independent of the snapshot checksum and verified rendition bytes.
// This does not attest equivalence with any remotely published catalog.
// It returns ErrNoCompatiblePhoto for missing Italian gender coverage.
// It returns context errors and file-system errors from snapshot loading.
func ConvertLocalPhotoCatalogData(ctx context.Context, directory, selectionSeed string) (*PhotoCatalogData, error) {

	if ctx == nil || directory == "" {
		return nil, fmt.Errorf("context and complete photo snapshot directory are required")
	}
	if !internationalSHA256(selectionSeed) || selectionSeed == strings.Repeat("0", 64) {
		return nil, fmt.Errorf("selection seed must be 64 lowercase hex digits encoding a nonzero 32-byte value")
	}
	snapshot, err := captureFaceCatalogMetadata(ctx, directory)
	if err != nil {
		return nil, err
	}
	faces, err := loadFaceCatalogFaces(ctx, directory, snapshot, nil, faceCatalogSnapshotModeComplete)
	if err != nil {
		return nil, err
	}
	err = verifyFaceCatalogMetadataStable(ctx, directory, snapshot)
	if err != nil {
		return nil, err
	}
	const release = "v1"
	document := internationalPhotos{
		Schema: "sw-photo-catalog-v1", Release: "v1-json1",
		GenderTaxonomy: internationalTaxonomyFor("sw-l1-gender-v1", []Gender{GenderFemale, GenderMale}),
		Selection:      internationalSelection{Version: release, Seed: selectionSeed},
	}
	for _, face := range faces {

		if face.Excluded || face.Status != fakefacegenStatusGenerated {
			continue
		}
		converted := internationalFace{ID: face.ID, Gender: string(face.Gender), RequestedAge: uint32(face.RequestedAge)}
		for _, asset := range face.Assets {
			converted.Renditions = append(converted.Renditions, internationalRendition{
				Width: asset.Width, Height: asset.Height,
				Path:      "synthetic-world/photos/" + release + "/" + strconv.Itoa(int(asset.Size)) + "/" + face.ID + ".webp",
				Integrity: &internationalIntegrity{SHA256: fmt.Sprintf("%x", asset.SHA256), Bytes: uint64(asset.ByteSize)},
			})
		}
		document.Faces = append(document.Faces, converted)

	}
	data, err := preparePhotoCatalogData(document)
	if err != nil {
		return nil, err
	}
	_, err = prepareItalianPhotoCatalog(data, document.GenderTaxonomy)
	if err != nil {
		return nil, err
	}

	return data, nil
}
