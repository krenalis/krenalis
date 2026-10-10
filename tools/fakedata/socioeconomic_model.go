// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"context"
	"crypto/sha256"
	"fmt"
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

// SocioeconomicModel is an opaque immutable Italian socioeconomic model.
// Acquire it through SocioeconomicModelSource.Load and share it across worlds.
// Its zero value is invalid; callers must not overwrite an acquired model.
type SocioeconomicModel struct {
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
	digest                [sha256.Size]byte
	fingerprints          [15][sha256.Size]byte
	initialized           bool
}

// SocioeconomicModelSource loads and retains a successful it-2026a model.
// It is safe for concurrent use. Models remain valid after the source is gone.
type SocioeconomicModelSource struct {
	resolver *externalassets.Resolver
	mu       sync.Mutex
	model    *SocioeconomicModel // protected by mu
	loading  chan struct{}       // protected by mu
}

// NewSocioeconomicModelSource creates a reusable source for the fixed it-2026a
// model using ordered External Assets origins.
func NewSocioeconomicModelSource(origins []string) (*SocioeconomicModelSource, error) {
	resolver, err := externalassets.NewResolver(origins)
	if err != nil {
		return nil, err
	}
	return &SocioeconomicModelSource{resolver: resolver}, nil
}

// Load acquires and retains an immutable model; failed loads remain retryable.
// It returns ctx.Err() when cancellation interrupts waiting for another load.
func (s *SocioeconomicModelSource) Load(ctx context.Context) (*SocioeconomicModel, error) {

	if s == nil || s.resolver == nil {
		return nil, fmt.Errorf("invalid socioeconomic model source")
	}

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

		var model *SocioeconomicModel
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
