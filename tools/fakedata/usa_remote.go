// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"sync"

	"github.com/krenalis/krenalis/tools/externalassets"
)

// USAWorldSource acquires the published USA release from ordered origins.
// Successful data is immutable and retained; failed loads can be retried.
type USAWorldSource struct {
	resolver *externalassets.Resolver
	mu       sync.Mutex
	prepared *USAWorldData // protected by mu
	loading  chan struct{} // protected by mu
}

// NewUSAWorldSource validates origins without acquiring any assets.
func NewUSAWorldSource(origins []string) (*USAWorldSource, error) {
	resolver, err := externalassets.NewResolver(origins)
	if err != nil {
		return nil, err
	}
	return &USAWorldSource{resolver: resolver}, nil
}

// Load returns the complete published USA World data, retrying failed loads.
// Cancellation returns ctx.Err(); waiting callers do not cancel other loads.
func (s *USAWorldSource) Load(ctx context.Context) (*USAWorldData, error) {

	if s == nil || s.resolver == nil || ctx == nil {
		return nil, fmt.Errorf("USA World source and context are required")
	}
	for {

		s.mu.Lock()
		if s.prepared != nil {
			data := s.prepared
			s.mu.Unlock()
			return data, nil
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

		data, err := loadRemoteUSAWorldData(ctx, s.resolver)
		s.mu.Lock()
		if err == nil {
			s.prepared = data
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

		return data, nil
	}

}

func loadRemoteUSAWorldData(ctx context.Context, resolver *externalassets.Resolver) (*USAWorldData, error) {

	var release *CountryReleaseData
	err := resolver.Resolve(ctx, usaReleasePath, 4096, func(reader io.Reader) error {

		candidate, err := decodeCountryReleaseData(ctx, reader)
		if err != nil {
			return err
		}
		d := &candidate.document
		if d.Country != "US" || d.Release != usaBaseRelease ||
			d.Base.Path != usaReleaseDirectory+"base.json" ||
			d.Socioeconomic.Path != usaReleaseDirectory+"socioeconomic.json" ||
			d.Photos.Path != "synthetic-world/photos/v1-json1/catalog.json" {
			return fmt.Errorf("unsupported USA release references")
		}
		release = candidate

		return nil
	})
	if err != nil {
		return nil, err
	}
	d := &release.document
	pins := []usaCountrySourcePin{
		{d.Base.Path, 3583463, "d8fd7c76142a86917ed4acec065951f8c5e9b44865e1bf6eed7f09f847eb5b80"},
		{usaReleaseDirectory + "npas.json", 22449, usaNPAAssetSHA256},
		{d.Socioeconomic.Path, usaSourceUSDModelBytes, usaSourceUSDModelSHA256},
		{d.Photos.Path, 322180, "657932d81a4d0d100ff3b473911ccc7dc0af2f869711b79e2d96f10d1287cde2"},
	}
	assets := map[string][]byte{}
	for _, pin := range pins {

		var candidate []byte
		err = resolver.Resolve(ctx, pin.Path, int64(pin.Bytes), func(reader io.Reader) error {
			encoded, err := io.ReadAll(reader)
			if err != nil {
				return err
			}
			if len(encoded) != pin.Bytes || fmt.Sprintf("%x", sha256.Sum256(encoded)) != pin.SHA256 {
				return fmt.Errorf("altered USA release asset")
			}
			candidate = encoded
			return nil
		})
		if err != nil {
			return nil, err
		}
		assets[pin.Path] = candidate

	}

	// Reuse the portable release preparers and the exact existing World binding.
	document, base, err := prepareUSAReleaseBase(ctx, assets[d.Base.Path])
	if err != nil {
		return nil, err
	}
	phones, err := decodeUSANPACandidate(ctx, bytes.NewReader(assets[usaReleaseDirectory+"npas.json"]))
	if err != nil {
		return nil, err
	}
	model, err := decodeUSASocioeconomicData(ctx, bytes.NewReader(assets[d.Socioeconomic.Path]))
	if err != nil {
		return nil, err
	}
	modelDigest, err := model.logicalDigest()
	if err != nil {
		return nil, err
	}
	photos, err := decodePhotoCatalogData(ctx, bytes.NewReader(assets[d.Photos.Path]))
	if err != nil {
		return nil, err
	}
	if d.Base.LogicalSHA256 != fmt.Sprintf("%x", base.digest) ||
		d.Socioeconomic.LogicalSHA256 != fmt.Sprintf("%x", modelDigest) ||
		d.Photos.LogicalSHA256 != fmt.Sprintf("%x", photos.digest) {
		return nil, fmt.Errorf("USA release logical digest mismatch")
	}
	catalog, err := prepareItalianPhotoCatalog(photos, document.Genders)
	if err != nil {
		return nil, err
	}
	digest, err := internationalDigest("fakedata/usa-internal-demo/v1", struct {
		Base   [32]byte
		Phones [32]byte
		Model  [32]byte
		Photos [32]byte
	}{base.digest, phones.identity, model.model.digest, photos.digest})
	if err != nil {
		return nil, err
	}
	country := &usaCountryData{baseDocument: document, base: base, phones: phones, socioeconomic: model}

	return &USAWorldData{country: country, catalog: catalog, digest: digest, resolver: resolver}, nil
}
