// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
	"fmt"
)

// usaFirstNamesV2Candidate admits the ten adopted BASE-v2 Method B groups.
type usaFirstNamesV2Candidate struct {
	groups   [10]*usaNominalIndex
	identity [sha256.Size]byte
}

// selectFirstName consumes only the name draw for an already selected gender
// and cohort; invalid group arguments leave the random stream unchanged.
func (d *usaFirstNamesV2Candidate) selectFirstName(rng *splitMix64, gender Gender, cohort string) (usaNameRow, error) {

	if d == nil || rng == nil {
		return usaNameRow{}, fmt.Errorf("uninitialized USA first-name candidate or random stream")
	}
	position := 0
	switch gender {
	case GenderFemale:
	case GenderMale:
		position = 5
	default:
		return usaNameRow{}, fmt.Errorf("invalid USA operational gender")
	}
	switch cohort {
	case "1950-1964":
	case "1965-1979":
		position++
	case "1980-1989":
		position += 2
	case "1990-1999":
		position += 3
	case "2000-2005":
		position += 4
	default:
		return usaNameRow{}, fmt.Errorf("invalid USA birth cohort")
	}
	group := d.groups[position]
	if group == nil || len(group.entries) != 1000 || group.total != usaNamesMass {
		return usaNameRow{}, fmt.Errorf("uninitialized USA first-name group")
	}

	return group.selectName(rng)
}
