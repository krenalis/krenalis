// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"context"
	"io"
	"sync"

	"github.com/krenalis/krenalis/tools/externalassets"
)

// D2-A bounds encoded numerical input, decoded work, and retained data
// separately at 8 MiB. The canonical JSON currently occupies 685,312 bytes.
const (
	socioeconomicModelPath        = "synthetic-world/it-2026a/model.json"
	maxSocioeconomicModelBytes    = 8 << 20
	maxSocioeconomicRetainedBytes = 8 << 20
)

// socioeconomicModel contains the immutable Italian numerical model at
// PopulationAnchorDate (2026-01-01), ready for subsequent runtime construction.
// It contains no generated people, companies, or random state.
type socioeconomicModel struct {
	companyCount          uint64
	ages                  [8][55]uint8
	companyIndustry       socioeconomicWeights
	employeeCount         []socioeconomicLaw
	education             socioeconomicWeights
	primaryActivity       socioeconomicWeights
	employeeJobFamily     socioeconomicWeights
	employeeJobLevel      socioeconomicWeights
	selfEmployedJobFamily socioeconomicWeights
	selfEmployedIndustry  socioeconomicWeights
	employer              socioeconomicEmployerParameters
	income                [6]socioeconomicIncome
}

// socioeconomicModelSource loads and retains a successful it-2026a model.
// Keep one source for the process lifetime, as with FaceCatalogSource.
type socioeconomicModelSource struct {
	resolver *externalassets.Resolver
	mu       sync.Mutex
	model    *socioeconomicModel // protected by mu
	loading  chan struct{}       // protected by mu
}

// newSocioeconomicModelSource creates a lazy source using ordered asset origins
// from the application's KRENALIS_EXTERNAL_ASSETS_URLS configuration.
func newSocioeconomicModelSource(origins []string) (*socioeconomicModelSource, error) {
	resolver, err := externalassets.NewResolver(origins)
	if err != nil {
		return nil, err
	}
	return &socioeconomicModelSource{resolver: resolver}, nil
}

// load returns the retained immutable model; failed attempts remain retryable.
// It returns ctx.Err() if waiting for another load is canceled.
func (s *socioeconomicModelSource) load(ctx context.Context) (*socioeconomicModel, error) {
	for {

		s.mu.Lock()
		if s.model != nil {
			model := s.model
			s.mu.Unlock()
			return model, nil
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

		var model *socioeconomicModel
		consume := func(reader io.Reader) error {
			model = nil
			var err error
			model, err = decodeSocioeconomicModel(reader)
			return err
		}
		err := s.resolver.Resolve(ctx, socioeconomicModelPath, maxSocioeconomicModelBytes, consume)
		s.mu.Lock()
		if err == nil {
			s.model = model
		}
		close(s.loading)
		s.loading = nil
		s.mu.Unlock()
		if err != nil {
			return nil, err
		}

		return model, nil
	}
}

type socioeconomicEmployerParameters struct {
	cap             uint16
	industryFactors [140]uint16
	sizeFactors     [16]uint16
}

// Income branches use e, s, u, t, r, i order. Tuple dimensions and category
// ordinals are those of D2-A sections 5 and 16, with the last axis innermost.
type socioeconomicIncome struct {
	tupleMap []uint16
	laws     []socioeconomicLaw
}

type socioeconomicLaw struct {
	values     []uint32
	cumulative []uint64
}

// Each row restarts its cumulative sum; zero-weight categories keep their slot.
// Canonical weights remain recoverable as adjacent cumulative differences.
type socioeconomicWeights struct {
	width      int
	cumulative []uint64
}
