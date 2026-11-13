// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
	"fmt"
)

// usaSurname50KCandidate is the separate owner-adopted Demo Plausible binding.
// It admits only the natural-name catalog adopted with BASE-v2.
type usaSurname50KCandidate struct {
	names    *usaNominalIndex
	identity [sha256.Size]byte
}

func (d *usaSurname50KCandidate) selectSurname(rng *splitMix64) (usaNameRow, error) {
	if d == nil || d.names == nil || len(d.names.entries) != 50000 || d.names.total != usaNamesMass {
		return usaNameRow{}, fmt.Errorf("uninitialized USA 50K surname candidate")
	}
	return d.names.selectName(rng)
}
