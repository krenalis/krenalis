// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const usaReleaseDirectory = "synthetic-world/countries/US/us-2026a/"
const usaReleasePath = usaReleaseDirectory + "release.json"

// prepareUSAReleaseBase indexes only the exact approved BASE-v2 document.
// Production evidence is not a runtime dependency; this byte lock replaces its
// repeated source comparison without changing sampling inputs or algorithms.
func prepareUSAReleaseBase(ctx context.Context, encoded []byte) (usaBaseDocument, *usaBaseCandidate, error) {

	if len(encoded) != 3583463 ||
		fmt.Sprintf("%x", sha256.Sum256(encoded)) != "d8fd7c76142a86917ed4acec065951f8c5e9b44865e1bf6eed7f09f847eb5b80" {
		return usaBaseDocument{}, nil, fmt.Errorf("altered USA release base")
	}
	var d usaBaseDocument
	err := decodeInternationalJSON(ctx, bytes.NewReader(encoded), &d)
	if err != nil {
		return usaBaseDocument{}, nil, err
	}

	// Build the same runtime indexes from their already embedded scalar values.
	base := &usaBaseCandidate{
		ages: d.Ages, firstNames: &usaFirstNamesV2Candidate{}, lastNames: &usaSurname50KCandidate{},
		geography: &usaGeographyCandidate{byID: map[string]int{}},
	}
	for _, age := range d.Ages {
		base.ageWeights = append(base.ageWeights, age.Weight)
	}
	for i, group := range d.FirstNames {
		base.firstNames.groups[i], err = newUSANominalIndex(group.Entries, 1000, usaNamesMass)
		if err != nil {
			return usaBaseDocument{}, nil, err
		}
	}
	base.lastNames.names, err = newUSANominalIndex(d.LastNames, 50000, usaNamesMass)
	if err != nil {
		return usaBaseDocument{}, nil, err
	}
	for i, place := range d.Geography.Places {

		var state usaState
		for _, candidate := range usaStates() {
			if candidate.USPS == *place.Administrative[0].Code {
				state = candidate
				break
			}
		}
		if state.USPS == "" || state.Division != place.Territory {
			return usaBaseDocument{}, nil, fmt.Errorf("incompatible USA release state")
		}
		g := base.geography
		g.byID[place.ID] = i
		g.locations = append(g.locations, usaLocation{
			id: place.ID, name: place.Name, state: state, weight: place.Population, zips: d.Geography.PostalCodes[place.ID],
		})
		g.weights = append(g.weights, place.Population)
		g.total += place.Population
		g.divisionMasses[state.DivisionOrdinal] += place.Population

	}
	for _, street := range d.StreetNames.Entries {
		base.streets = append(base.streets, street.Text)
	}

	// Preserve the content fingerprints and the frozen BASE-v2 stream inputs.
	common := internationalBase{
		Schema: d.Schema, Release: d.Release, Profile: d.Profile, Country: d.Country, AnchorDate: d.AnchorDate,
		Genders: d.Genders, Ages: d.Ages, Cohorts: d.Cohorts, LastNames: d.LastNames, StreetNames: d.StreetNames,
		Geography: internationalGeography{
			PlaceCodeSystem: d.Geography.PlaceCodeSystem, TerritorySystem: d.Geography.TerritorySystem,
			PostalCodeSystem: d.Geography.PostalCodeSystem, Places: d.Geography.Places, PostalCodes: d.Geography.PostalCodes,
		},
	}
	for _, group := range d.FirstNames {
		common.FirstNames = append(common.FirstNames, internationalFirstNames{
			Gender: group.Gender, Cohort: group.Cohort, Entries: group.Entries,
		})
	}
	base.fingerprints, err = internationalBaseFingerprints(&common)
	if err != nil {
		return usaBaseDocument{}, nil, err
	}
	base.digest, err = internationalDigest("fakedata/base-data/json-v1", d)
	if err != nil {
		return usaBaseDocument{}, nil, err
	}
	for i, name := range []string{
		ageDatasetComponentName, firstNameDatasetComponentName, lastNameDatasetComponentName,
		placesDatasetComponentName, postalDatasetComponentName, streetNameDatasetComponentName,
	} {
		fingerprint, err := hex.DecodeString(usaBaseSamplingFingerprints[i])
		if err != nil {
			return usaBaseDocument{}, nil, err
		}
		base.dependencies[i], err = BytesComponent(name, fingerprint)
		if err != nil {
			return usaBaseDocument{}, nil, err
		}
	}

	return d, base, nil
}
