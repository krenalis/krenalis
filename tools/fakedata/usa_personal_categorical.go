// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
)

var usaPersonalCategoricalNodes = [...]int{
	socioeconomicEducation, socioeconomicPrimaryActivity, socioeconomicEmployeeJobFamily,
	socioeconomicEmployeeJobLevel, socioeconomicSelfEmployedJobFamily, socioeconomicSelfEmployedIndustry,
}

// usaPersonalCategoricalCandidate holds six tables, three independent age maps
// and an admission guard, without Income or a public World representation.
type usaPersonalCategoricalCandidate struct {
	ages         [3][55]uint8
	tables       [6]socioeconomicWeights
	fingerprints [6][sha256.Size]byte
	identity     [sha256.Size]byte
}
