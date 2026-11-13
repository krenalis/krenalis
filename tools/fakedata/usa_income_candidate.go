// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
)

// usaIncomeCandidate is an internal six-branch sampler view whose identity
// guards admission; it never contributes aggregate identity to a stream.
type usaIncomeCandidate struct {
	numeric      socioeconomicNumericModel
	fingerprints [6][sha256.Size]byte
	identity     [sha256.Size]byte
}
