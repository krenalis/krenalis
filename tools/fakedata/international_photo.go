// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"context"
	"fmt"
	"io"
	"math"
	"slices"
	"strings"
)

const internationalPhotoCatalogMode = "photo-json-v1"

// PhotoCatalogData is an immutable, normalized sw-photo-catalog-v1 document.
// It declares image metadata; decoding does not verify or download WebP bytes.
// Its zero value is invalid.
type PhotoCatalogData struct {
	document    internationalPhotos
	digest      [32]byte
	fingerprint [32]byte
	initialized bool
}

// DecodePhotoCatalogData validates an international v1 photo catalog.
// It returns context errors between reads and tools/json SyntaxError for
// malformed tokens. Callers must interrupt blocking reads themselves.
func DecodePhotoCatalogData(ctx context.Context, reader io.Reader) (*PhotoCatalogData, error) {
	return decodePhotoCatalogData(ctx, reader)
}

// CanonicalJSON returns the normalized catalog's C1 serialization.
func (d *PhotoCatalogData) CanonicalJSON() ([]byte, error) {
	if d == nil || !d.initialized {
		return nil, fmt.Errorf("uninitialized photo catalog document")
	}
	return internationalCanonical(d.document)
}

// LogicalDigest returns D_photos, including release and rendition metadata.
func (d *PhotoCatalogData) LogicalDigest() ([32]byte, error) {
	if d == nil || !d.initialized {
		return [32]byte{}, fmt.Errorf("uninitialized photo catalog document")
	}
	return d.digest, nil
}

// SelectionFingerprint returns F_photo, excluding all rendition metadata.
func (d *PhotoCatalogData) SelectionFingerprint() ([32]byte, error) {
	if d == nil || !d.initialized {
		return [32]byte{}, fmt.Errorf("uninitialized photo catalog document")
	}
	return d.fingerprint, nil
}

type internationalPhotos struct {
	Schema         string                 `json:"schema"`
	Release        string                 `json:"release"`
	GenderTaxonomy internationalTaxonomy  `json:"genderTaxonomy"`
	Selection      internationalSelection `json:"selection"`
	Faces          []internationalFace    `json:"faces" max:"100000"`
}

type internationalSelection struct {
	Version string `json:"version"`
	Seed    string `json:"seed"`
}

type internationalFace struct {
	ID           string                   `json:"id"`
	Gender       string                   `json:"gender"`
	RequestedAge uint32                   `json:"requestedAge"`
	Renditions   []internationalRendition `json:"renditions" max:"16"` // local allocation bound
}

type internationalRendition struct {
	Width     uint32                  `json:"width"`
	Height    uint32                  `json:"height"`
	Path      string                  `json:"path"`
	Integrity *internationalIntegrity `json:"integrity,omitempty"`
}

type internationalIntegrity struct {
	SHA256 string `json:"sha256"`
	Bytes  uint64 `json:"bytes"`
}

func decodePhotoCatalogData(ctx context.Context, reader io.Reader) (*PhotoCatalogData, error) {
	var document internationalPhotos
	err := decodeInternationalJSON(ctx, reader, &document)
	if err != nil {
		return nil, err
	}
	return preparePhotoCatalogData(document)
}

func internationalAssetPath(value string) bool {
	if value == "" || len(value) > 4096 {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
		for _, c := range segment {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
				c == '.' || c == '_' || c == '-') {
				return false
			}
		}
	}
	return true
}

// prepareItalianPhotoCatalog checks Italian coverage and prepares selection
// and the existing square-size asset API. Other renditions stay in the full
// document and its digest; no image bytes are acquired or verified here.
func prepareItalianPhotoCatalog(photos *PhotoCatalogData, genders internationalTaxonomy) (*FaceCatalog, error) {

	err := validateInternationalTaxonomy(photos.document.GenderTaxonomy, genders)
	if err != nil {
		return nil, err
	}
	version, err := StringComponent(faceCatalogVersionComponentName, photos.document.Selection.Version)
	if err != nil {
		return nil, err
	}
	selection, err := BytesComponent(faceCatalogSHA256ComponentName, photos.fingerprint[:])
	if err != nil {
		return nil, err
	}
	catalog := &FaceCatalog{
		version: photos.document.Selection.Version, release: photos.document.Release,
		checksum: photos.digest, dependencies: [2]Component{version, selection}, mode: internationalPhotoCatalogMode,
		assets: map[photoAssetKey]photoAssetRecord{},
	}
	for _, face := range photos.document.Faces {

		// The selector subtracts AnchorAge in an int; bound before narrowing.
		if uint64(face.RequestedAge) > uint64(math.MaxInt) {
			return nil, fmt.Errorf("photo age exceeds runtime representation")
		}
		catalog.faces = append(catalog.faces, photoFace{
			id: face.ID, gender: Gender(face.Gender), requestedAge: int(face.RequestedAge), status: fakefacegenStatusGenerated,
		})
		has256 := false
		for _, rendition := range face.Renditions {

			if rendition.Width == 256 && rendition.Height == 256 {
				has256 = true
			}
			if rendition.Width != rendition.Height || rendition.Width > math.MaxUint16 ||
				!slices.Contains(supportedPhotoSizes[:], PhotoSize(rendition.Width)) {
				continue
			}
			asset := PhotoAsset{
				PhotoID: face.ID, Size: PhotoSize(rendition.Width), Width: rendition.Width, Height: rendition.Height,
				CatalogVersion: photos.document.Release, CatalogSHA256: fmt.Sprintf("%x", photos.digest), Path: rendition.Path,
			}
			if integrity := rendition.Integrity; integrity != nil {
				if integrity.Bytes > math.MaxInt64 {
					return nil, fmt.Errorf("photo byte count exceeds runtime representation")
				}
				asset.ContentSHA256, asset.ContentLength = integrity.SHA256, int64(integrity.Bytes)
			}
			catalog.assets[photoAssetKey{photoID: face.ID, size: asset.Size}] = photoAssetRecord{metadata: asset}

		}
		if !has256 {
			return nil, fmt.Errorf("Italian photo candidate lacks a 256x256 rendition")
		}

	}
	for _, gender := range genders.Categories {
		if !catalog.hasUsableGender(Gender(gender.Code)) {
			return nil, fmt.Errorf("%w: gender %s", ErrNoCompatiblePhoto, gender.Code)
		}
	}

	return catalog, nil
}

// preparePhotoCatalogData validates and normalizes owned transport values,
// then derives the separate document and selection identities.
func preparePhotoCatalogData(document internationalPhotos) (*PhotoCatalogData, error) {

	err := validateInternationalPhotos(&document)
	if err != nil {
		return nil, err
	}

	// Only these two collections have insignificant transport order.
	slices.SortFunc(document.Faces, func(a, b internationalFace) int { return strings.Compare(a.ID, b.ID) })
	for i := range document.Faces {
		slices.SortFunc(document.Faces[i].Renditions, func(a, b internationalRendition) int {
			if a.Width < b.Width || a.Width == b.Width && a.Height < b.Height {
				return -1
			}
			if a.Width == b.Width && a.Height == b.Height {
				return 0
			}
			return 1
		})
	}
	digest, err := internationalDigest("fakedata/photo-catalog/json-v1", document)
	if err != nil {
		return nil, err
	}

	// Q preserves taxonomy order and excludes only explanatory definitions.
	projection := struct {
		GenderTaxonomy internationalTaxonomy  `json:"genderTaxonomy"`
		Selection      internationalSelection `json:"selection"`
		Faces          []map[string]any       `json:"faces"`
	}{GenderTaxonomy: internationalTaxonomy{SemanticID: document.GenderTaxonomy.SemanticID}, Selection: document.Selection}
	for _, category := range document.GenderTaxonomy.Categories {
		projection.GenderTaxonomy.Categories = append(projection.GenderTaxonomy.Categories,
			internationalCategory{Code: category.Code})
	}
	for _, face := range document.Faces {
		projection.Faces = append(projection.Faces, map[string]any{
			"id": face.ID, "gender": face.Gender, "requestedAge": face.RequestedAge,
		})
	}
	canonical, err := internationalCanonical(projection)
	if err != nil {
		return nil, err
	}
	selection, err := BytesComponent("selection", canonical)
	if err != nil {
		return nil, err
	}
	fingerprint, err := canonicalDigest("fakedata/photo-selection/json-v1", []Component{selection})
	if err != nil {
		return nil, err
	}

	return &PhotoCatalogData{document: document, digest: digest, fingerprint: fingerprint, initialized: true}, nil
}

func validateInternationalPhotos(d *internationalPhotos) error {

	if d.Schema != "sw-photo-catalog-v1" || !internationalToken(d.Release) ||
		!internationalToken(d.Selection.Version) || !internationalSHA256(d.Selection.Seed) ||
		d.Selection.Seed == strings.Repeat("0", 64) || len(d.Faces) == 0 {
		return fmt.Errorf("invalid photo catalog metadata or selection")
	}
	genders := map[string]bool{}
	if !internationalToken(d.GenderTaxonomy.SemanticID) || len(d.GenderTaxonomy.Categories) == 0 {
		return fmt.Errorf("invalid photo gender taxonomy")
	}
	for _, category := range d.GenderTaxonomy.Categories {
		if !internationalText(category.Code) || genders[category.Code] ||
			category.Definition != nil && !internationalText(*category.Definition) {
			return fmt.Errorf("invalid or duplicate photo gender category")
		}
		genders[category.Code] = true
	}

	// Validate candidates and repeated path declarations before normalization.
	ids := map[string]bool{}
	paths := map[string]internationalRendition{}
	for i, face := range d.Faces {

		if len(face.ID) == 0 || len(face.ID) > 128 || ids[face.ID] || !genders[face.Gender] ||
			len(face.Renditions) == 0 {
			return fmt.Errorf("invalid photo candidate at position %d", i)
		}
		for _, c := range face.ID {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
				return fmt.Errorf("invalid photo ID at position %d", i)
			}
		}
		ids[face.ID] = true
		dimensions := map[[2]uint32]bool{}
		for _, rendition := range face.Renditions {

			pair := [2]uint32{rendition.Width, rendition.Height}
			if rendition.Width == 0 || rendition.Height == 0 || dimensions[pair] ||
				!internationalAssetPath(rendition.Path) || !strings.HasSuffix(rendition.Path, ".webp") {
				return fmt.Errorf("invalid photo rendition at candidate %d", i)
			}
			dimensions[pair] = true
			if integrity := rendition.Integrity; integrity != nil {
				if !internationalSHA256(integrity.SHA256) || integrity.Bytes == 0 {
					return fmt.Errorf("invalid photo integrity at candidate %d", i)
				}
			}

			if previous, exists := paths[rendition.Path]; exists {
				if previous.Width != rendition.Width || previous.Height != rendition.Height ||
					previous.Integrity != nil && rendition.Integrity != nil && *previous.Integrity != *rendition.Integrity {
					return fmt.Errorf("inconsistent repeated photo path at candidate %d", i)
				}
				if rendition.Integrity == nil {
					continue
				}
			}
			paths[rendition.Path] = rendition

		}

	}

	return nil
}
