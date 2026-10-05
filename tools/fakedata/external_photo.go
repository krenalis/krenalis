// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"sync"

	"github.com/krenalis/krenalis/tools/externalassets"
)

const (
	externalPhotoCatalogPath = "synthetic-world/photos/v1/catalog.json"
	externalPhotoCatalogMode = "external"
)

// At most 100,000 records of roughly 200 bytes (including a 128-byte ID),
// with headroom for JSON whitespace. Image bytes are never included.
const maxExternalPhotoCatalogBytes = 32 << 20

// FaceCatalogSource loads and retains a successful external photo catalog.
type FaceCatalogSource struct {
	resolver *externalassets.Resolver
	mu       sync.Mutex
	catalog  *FaceCatalog  // protected by mu
	loading  chan struct{} // protected by mu
}

// NewFaceCatalogSource creates a lazy source using the ordered asset origins.
func NewFaceCatalogSource(origins []string) (*FaceCatalogSource, error) {
	resolver, err := externalassets.NewResolver(origins)
	if err != nil {
		return nil, err
	}
	return &FaceCatalogSource{resolver: resolver}, nil
}

// Load returns the immutable process-lifetime catalog, retrying failed loads.
// It returns ErrInvalidFaceCatalog for a nil or uninitialized source.
// It returns ctx.Err() if waiting for another load is canceled.
func (s *FaceCatalogSource) Load(ctx context.Context) (*FaceCatalog, error) {

	if s == nil || s.resolver == nil {
		return nil, ErrInvalidFaceCatalog
	}

	for {

		s.mu.Lock()
		if s.catalog != nil {
			catalog := s.catalog
			s.mu.Unlock()
			return catalog, nil
		}
		if s.loading != nil {
			loading := s.loading
			s.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-loading:
				continue
			}
		}
		s.loading = make(chan struct{})
		s.mu.Unlock()

		var catalog *FaceCatalog
		consume := func(reader io.Reader) error {
			var err error
			catalog, err = decodeExternalPhotoCatalog(reader)
			return err
		}
		err := s.resolver.Resolve(ctx, externalPhotoCatalogPath, maxExternalPhotoCatalogBytes, consume)
		s.mu.Lock()
		if err == nil {
			s.catalog = catalog
		}
		close(s.loading)
		s.loading = nil
		s.mu.Unlock()
		if err != nil {
			return nil, err
		}

		return catalog, nil
	}

}

type externalPhotoCatalog struct {
	Format        int                 `json:"format"`
	Release       string              `json:"release"`
	SelectionSeed string              `json:"selectionSeed"`
	Faces         []externalPhotoFace `json:"faces"`
}

type externalPhotoFace struct {
	ID           string `json:"id"`
	Gender       Gender `json:"gender"`
	RequestedAge int    `json:"requestedAge"`
}

// decodeExternalPhotoCatalog bounds the record count before growing the slice.
func decodeExternalPhotoCatalog(reader io.Reader) (*FaceCatalog, error) {

	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if token != json.Delim('{') {
		return nil, fmt.Errorf("photo catalog must be an object")
	}
	var document externalPhotoCatalog
	seen := map[string]bool{}
	for decoder.More() {

		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		field, ok := token.(string)
		if !ok || seen[field] {
			return nil, fmt.Errorf("invalid or duplicate photo catalog field")
		}
		seen[field] = true
		switch field {
		case "format":
			err = decoder.Decode(&document.Format)
		case "release":
			err = decoder.Decode(&document.Release)
		case "selectionSeed":
			err = decoder.Decode(&document.SelectionSeed)
		case "faces":
			token, err = decoder.Token()
			if err != nil {
				return nil, err
			}
			if token != json.Delim('[') {
				return nil, fmt.Errorf("photo catalog faces must be an array")
			}
			for decoder.More() {
				if len(document.Faces) == maxFaceCatalogRecords {
					return nil, fmt.Errorf("photo catalog exceeds %d records", maxFaceCatalogRecords)
				}
				var face externalPhotoFace
				err = decoder.Decode(&face)
				if err != nil {
					return nil, err
				}
				document.Faces = append(document.Faces, face)
			}
			_, err = decoder.Token()
		default:
			return nil, fmt.Errorf("unknown photo catalog field")
		}
		if err != nil {
			return nil, err
		}

	}
	_, err = decoder.Token()
	if err != nil {
		return nil, err
	}
	_, err = decoder.Token()
	if err != nil {
		if err != io.EOF {
			return nil, err
		}
	}
	if err == nil {
		return nil, fmt.Errorf("photo catalog contains multiple JSON values")
	}

	// Validate the representation and frozen selection input.
	if document.Format != 1 || document.Release != "v1" || len(document.Faces) == 0 ||
		len(document.SelectionSeed) != 2*sha256.Size {
		return nil, fmt.Errorf("invalid photo catalog format, release, selection seed, or record count")
	}
	seed, err := hex.DecodeString(document.SelectionSeed)
	if err != nil {
		return nil, fmt.Errorf("invalid photo catalog selection seed")
	}
	var checksum [sha256.Size]byte
	copy(checksum[:], seed)
	if checksum == ([sha256.Size]byte{}) {
		return nil, fmt.Errorf("photo catalog selection seed is zero")
	}

	// This is the existing Layer 1 RNG input, not a hash of downloaded content.
	// Publishers migrating a local snapshot preserve its Checksum as this seed.
	version, err := StringComponent(faceCatalogVersionComponentName, fakefacegenV1SourceVersion)
	if err != nil {
		return nil, err
	}
	identity, err := BytesComponent(faceCatalogSHA256ComponentName, checksum[:])
	if err != nil {
		return nil, err
	}
	catalog := &FaceCatalog{
		version: fakefacegenV1SourceVersion, release: document.Release, checksum: checksum,
		mode: externalPhotoCatalogMode, dependencies: [2]Component{version, identity},
		assets: map[photoAssetKey]photoAssetRecord{},
	}
	ids := map[string]bool{}
	for _, face := range document.Faces {

		if len(face.ID) == 0 || len(face.ID) > 128 || ids[face.ID] {
			return nil, fmt.Errorf("invalid or duplicate photo ID")
		}
		for _, ch := range face.ID {
			if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_') {
				return nil, fmt.Errorf("invalid photo ID characters")
			}
		}
		ids[face.ID] = true
		catalog.faces = append(catalog.faces, photoFace{
			id: face.ID, gender: face.Gender, requestedAge: face.RequestedAge, status: fakefacegenStatusGenerated,
		})
		catalog.assets[photoAssetKey{photoID: face.ID, size: PhotoSize256}] = photoAssetRecord{metadata: PhotoAsset{
			PhotoID: face.ID, Size: PhotoSize256, Width: 256, Height: 256, CatalogVersion: document.Release,
			Path: "synthetic-world/photos/" + document.Release + "/256/" + face.ID + ".webp",
		}}

	}
	sort.Slice(catalog.faces, func(i, j int) bool { return catalog.faces[i].id < catalog.faces[j].id })
	err = catalog.validate()
	if err != nil {
		return nil, err
	}
	for _, gender := range [...]Gender{GenderFemale, GenderMale} {
		if !catalog.hasUsableGender(gender) {
			return nil, fmt.Errorf("%w: gender %s", ErrNoCompatiblePhoto, gender)
		}
	}

	return catalog, nil
}
