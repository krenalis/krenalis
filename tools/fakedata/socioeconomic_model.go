// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import "crypto/sha256"

// D2-A bounds encoded numerical input, decoded work, and retained data
// separately at 8 MiB. The canonical JSON currently occupies 685,312 bytes.
const (
	maxSocioeconomicModelBytes    = 8 << 20
	maxSocioeconomicRetainedBytes = 8 << 20
)

// SocioeconomicModel is an opaque immutable Italian socioeconomic model.
// Acquire it through ItalianWorldData.Model and share it across worlds.
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
