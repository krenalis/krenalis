// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
	"fmt"
)

// D2-A bounds retained numerical data at 8 MiB.
const maxSocioeconomicRetainedBytes = 8 << 20

const (
	socioeconomicItalyRegions socioeconomicTerritoryDomain = 20
	socioeconomicUSADivisions socioeconomicTerritoryDomain = 9
)

// SocioeconomicModel is an opaque immutable Italian socioeconomic model.
// Acquire it through ItalianWorldData.Model and share it across worlds.
// Its zero value is invalid; callers must not overwrite an acquired model.
type SocioeconomicModel struct {
	socioeconomicNumericModel
	digest       [sha256.Size]byte
	fingerprints [15][sha256.Size]byte
	initialized  bool
}

// socioeconomicNumericModel has no national identity or monetary consumer.
// Only the Italian compiler wraps it in an acquired SocioeconomicModel.
type socioeconomicNumericModel struct {
	territories           socioeconomicTerritoryDomain
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

// territoryRow flattens an outer row and its innermost territorial axis.
func (m *socioeconomicNumericModel) territoryRow(prefix, territory int) (int, error) {
	if m.territories != socioeconomicItalyRegions && m.territories != socioeconomicUSADivisions {
		return 0, fmt.Errorf("invalid socioeconomic territory domain")
	}
	if territory < 0 || territory >= int(m.territories) {
		return 0, fmt.Errorf("invalid socioeconomic territory ordinal")
	}
	if prefix < 0 || prefix > (int(^uint(0)>>1)-territory)/int(m.territories) {
		return 0, fmt.Errorf("socioeconomic territorial row overflow")
	}
	return prefix*int(m.territories) + territory, nil
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

// socioeconomicTerritoryDomain admits only Italian regions and USA divisions.
type socioeconomicTerritoryDomain uint8

// Each row restarts its cumulative sum; zero-weight categories keep their slot.
// Canonical weights remain recoverable as adjacent cumulative differences.
type socioeconomicWeights struct {
	width      int
	cumulative []uint64
}
