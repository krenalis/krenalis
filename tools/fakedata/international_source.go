// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"sync"

	"github.com/krenalis/krenalis/tools/externalassets"
)

// ItalianWorldSource acquires the fixed Italian JSON v1 release on demand.
// Configure it with the ordered KRENALIS_EXTERNAL_ASSETS_URLS origins. It owns
// immutable successes and coordinates concurrent loads without caching errors.
// Prepared data remains usable independently of this owner's lifetime.
type ItalianWorldSource struct {
	resolver  *externalassets.Resolver
	photoBase *url.URL
	mu        sync.Mutex
	documents italianWorldDocuments // protected by mu
	prepared  *ItalianWorldData     // protected by mu
	loading   chan struct{}         // protected by mu
}

// NewItalianWorldSource configures a lazy owner without acquiring any assets.
// Origin validation is delegated to the existing External Assets resolver.
func NewItalianWorldSource(origins []string) (*ItalianWorldSource, error) {

	resolver, err := externalassets.NewResolver(origins)
	if err != nil {
		return nil, err
	}
	photoBase, err := url.Parse(origins[0])
	if err != nil {
		return nil, err
	}

	return &ItalianWorldSource{resolver: resolver, photoBase: photoBase}, nil
}

// Load returns a complete prepared combination, retrying unsuccessful loads.
// Cancellation returns ctx.Err(); waiting callers do not cancel other loads.
func (s *ItalianWorldSource) Load(ctx context.Context) (*ItalianWorldData, error) {

	if s == nil || s.resolver == nil || ctx == nil {
		return nil, fmt.Errorf("invalid Italian World source or context")
	}

	for {

		s.mu.Lock()
		if s.prepared != nil {
			prepared := s.prepared
			s.mu.Unlock()
			return prepared, nil
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
		documents := s.documents
		s.mu.Unlock()

		prepared, err := s.acquire(ctx, &documents)
		s.mu.Lock()
		// Each retained document has completed Resolve, including its trailer.
		// An incompatible combination does not discard autonomous successes.
		s.documents = documents
		if err == nil {
			s.prepared = prepared
		}
		close(s.loading)
		s.loading = nil
		s.mu.Unlock()
		if err != nil {
			if contextErr := ctx.Err(); contextErr != nil {
				return nil, contextErr
			}
			return nil, err
		}

		return prepared, nil
	}

}

// PhotoURL joins a declared rendition path to the first configured origin.
// It performs no image request or fallback and returns ErrInvalidPhotoAsset
// when the rendition is unavailable, or ErrInvalidFaceCatalog for nil catalogs.
func (s *ItalianWorldSource) PhotoURL(catalog *FaceCatalog, photoID string, size PhotoSize) (string, error) {

	if s == nil || s.photoBase == nil {
		return "", fmt.Errorf("uninitialized Italian World source")
	}
	asset, err := catalog.asset(photoID, size)
	if err != nil {
		return "", err
	}
	if catalog.mode != internationalPhotoCatalogMode {
		return "", fmt.Errorf("Italian JSON photo catalog is required")
	}

	return s.photoBase.JoinPath(asset.Path).String(), nil
}

func (s *ItalianWorldSource) acquire(ctx context.Context, d *italianWorldDocuments) (*ItalianWorldData, error) {

	if d.release == nil {

		var candidate *CountryReleaseData
		err := s.resolver.Resolve(ctx, ItalianReleasePath, 64<<10, func(reader io.Reader) error {
			var err error
			candidate, err = decodeCountryReleaseData(ctx, reader)
			if err != nil {
				return err
			}
			if candidate.document.Country != "IT" {
				return fmt.Errorf("unsupported release country")
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		d.release = candidate

	}
	release := &d.release.document
	if d.base == nil {

		var candidate *ItalianBaseData
		err := s.resolver.Resolve(ctx, release.Base.Path, maxInternationalDocumentBytes, func(reader io.Reader) error {
			var err error
			candidate, err = decodeItalianBaseData(ctx, reader)
			if err != nil {
				return err
			}
			digest, err := internationalDigest("fakedata/base-data/json-v1", candidate.document)
			if err != nil {
				return err
			}
			if fmt.Sprintf("%x", digest) != release.Base.LogicalSHA256 {
				return fmt.Errorf("base logical digest mismatch")
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		d.base = candidate

	}
	if d.socioeconomic == nil {

		var candidate *ItalianSocioeconomicData
		reference := release.Socioeconomic
		err := s.resolver.Resolve(ctx, reference.Path, maxInternationalDocumentBytes, func(reader io.Reader) error {
			var err error
			candidate, err = decodeItalianSocioeconomicData(ctx, reader)
			if err != nil {
				return err
			}
			digest, err := internationalDigest("fakedata/socioeconomic-data/json-v1", candidate.document)
			if err != nil {
				return err
			}
			if fmt.Sprintf("%x", digest) != reference.LogicalSHA256 {
				return fmt.Errorf("socioeconomic logical digest mismatch")
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		d.socioeconomic = candidate

	}
	if d.photos == nil {

		var candidate *PhotoCatalogData
		var catalog *FaceCatalog
		err := s.resolver.Resolve(ctx, release.Photos.Path, maxInternationalDocumentBytes, func(reader io.Reader) error {
			var err error
			candidate, err = decodePhotoCatalogData(ctx, reader)
			if err != nil {
				return err
			}
			if fmt.Sprintf("%x", candidate.digest) != release.Photos.LogicalSHA256 {
				return fmt.Errorf("photo logical digest mismatch")
			}
			genders := internationalTaxonomyFor("sw-l1-gender-v1", []Gender{GenderFemale, GenderMale})
			catalog, err = prepareItalianPhotoCatalog(candidate, genders)
			return err
		})
		if err != nil {
			return nil, err
		}
		d.photos, d.catalog = candidate, catalog

	}

	return prepareItalianWorldData(d.base, d.socioeconomic, d.photos, d.catalog)
}

// The successful descriptor is pinned for this owner's lifetime and origin
// scope. Each typed slot belongs exclusively to its role, path and expected
// digest in that descriptor; another descriptor requires another owner.
type italianWorldDocuments struct {
	release       *CountryReleaseData
	base          *ItalianBaseData
	socioeconomic *ItalianSocioeconomicData
	photos        *PhotoCatalogData
	catalog       *FaceCatalog
}
