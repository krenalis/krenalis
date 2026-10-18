// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"slices"
	"strings"
)

const (
	usaBaseProfile       = "us-2026a-base-v1"
	usaBaseRelease       = "us-2026a-layer1-v2-candidate"
	usaStreetAssetSHA256 = "4f5b26c8c460265d49115067882df193a938bd25e460d00b2b99b6fd3f5784e9"
)

// BASE-v2 fixes these sampling dependencies independently of content identity.
// They are the six stream inputs in the byte-pinned sampling contract.
var usaBaseSamplingFingerprints = [6]string{
	"1e5f4ae29e8c9ac7696bce9eba891f0f55046c0797665c7ad266190db31ee158",
	"6af2ea8f9ed1f3c9e183a1f07c836a5d8465ce318465269cb8ccb487773c014a",
	"1ed45af003d8e90a7901549ab3e793da179476eb942019ad854946d4d0877945",
	"5eba151a28c148e96d61163271b7e8696eecdcb380ba7adcd4bef4cdf929bbb5",
	"f6a289e771a4f4a4d0bfbf9be7096de7e7212676fa3f4a9d676a6ed6b4381914",
	"02bc1a386edbe518bb2bbcc682b7d08b885fddaf3b292d1d0bccb1fc9cf5980a",
}

// usaBaseCandidate owns prepared national inputs; it is not a public World or
// a release. Photography and the NPA rule retain separate identities.
type usaBaseCandidate struct {
	ages         []internationalAgeWeight
	ageWeights   []uint64
	firstNames   *usaFirstNamesV2Candidate
	lastNames    *usaSurname50KCandidate
	geography    *usaGeographyCandidate
	streets      []string
	digest       [sha256.Size]byte
	fingerprints [6][sha256.Size]byte
	dependencies [6]Component
}

// usaBaseDocument is sw-base-data-v1 with USA-only allocation bounds; the
// Italian transport's 600 surnames and 80 names per group stay unchanged.
type usaBaseDocument struct {
	Schema      string                   `json:"schema"`
	Release     string                   `json:"release"`
	Profile     string                   `json:"profile"`
	Country     string                   `json:"country"`
	AnchorDate  string                   `json:"anchorDate"`
	Genders     internationalTaxonomy    `json:"genders"`
	Ages        []internationalAgeWeight `json:"ages" max:"55"`
	Cohorts     []internationalCohort    `json:"cohorts" max:"5"`
	FirstNames  []usaBaseFirstNames      `json:"firstNames" max:"10"`
	LastNames   []internationalName      `json:"lastNames" max:"50000"`
	Geography   usaBaseGeography         `json:"geography"`
	StreetNames internationalStreets     `json:"streetNames"`
}

type usaBaseEvidence struct {
	ages       *usaAgeCandidate
	firstNames *usaFirstNamesV2Candidate
	lastNames  *usaSurname50KCandidate
	geography  *usaGeographyCandidate
}

type usaBaseFirstNames struct {
	Gender  string              `json:"gender"`
	Cohort  string              `json:"cohort"`
	Entries []internationalName `json:"entries" max:"1000"`
}

type usaBaseGeography struct {
	PlaceCodeSystem  string               `json:"placeCodeSystem"`
	TerritorySystem  string               `json:"territorySystem"`
	PostalCodeSystem string               `json:"postalCodeSystem"`
	Places           []internationalPlace `json:"places" max:"127"`
	PostalCodes      map[string][]string  `json:"postalCodes" max:"127" maxvalue:"1024"`
}

// decodeUSABaseCandidate admits the international schema against separately
// qualified national evidence; callers interrupt any blocking reads.
func decodeUSABaseCandidate(ctx context.Context, reader io.Reader, evidence usaBaseEvidence) (*usaBaseCandidate, error) {

	var document usaBaseDocument
	err := decodeInternationalJSON(ctx, reader, &document)
	if err != nil {
		return nil, err
	}

	return prepareUSABaseCandidate(&document, evidence)
}

// prepareUSABaseCandidate binds every source-derived value to admitted USA
// inputs, preserving the source displays and weights without Italian limits.
func prepareUSABaseCandidate(d *usaBaseDocument, evidence usaBaseEvidence) (*usaBaseCandidate, error) {

	// Validate the national binding and the identities of the qualified inputs.
	if d == nil || d.Schema != "sw-base-data-v1" || d.Release != usaBaseRelease || d.Profile != usaBaseProfile ||
		d.Country != "US" || d.AnchorDate != usaSocioeconomicAnchorDate {
		return nil, fmt.Errorf("unsupported USA candidate base metadata")
	}
	a, f, l, g := evidence.ages, evidence.firstNames, evidence.lastNames, evidence.geography
	if a == nil || f == nil || l == nil || g == nil ||
		fmt.Sprintf("%x", a.identity) != usaFirstNamesAgeAssetSHA256 ||
		fmt.Sprintf("%x", f.identity) != usaFirstNamesV2AssetSHA256 ||
		fmt.Sprintf("%x", l.identity) != usaSurname50KAssetSHA256 ||
		fmt.Sprintf("%x", g.identity) != usaGeographyV2AssetSHA256 {
		return nil, fmt.Errorf("qualified USA age, adopted names and geography v2 are required")
	}
	if len(a.ages) != 55 || l.names == nil || len(l.names.entries) != 50000 || len(g.locations) != 127 {
		return nil, fmt.Errorf("incomplete prepared USA evidence")
	}
	err := validateInternationalTaxonomy(d.Genders,
		internationalTaxonomyFor("sw-l1-gender-v1", []Gender{GenderFemale, GenderMale}))
	if err != nil {
		return nil, err
	}
	// The five birth-year cohorts are shared; only the national data differ.
	if !slices.Equal(d.Ages, a.ages) || !slices.Equal(d.Cohorts, italianInternationalCohorts()) ||
		len(d.FirstNames) != 10 || len(d.LastNames) != 50000 {
		return nil, fmt.Errorf("USA age, cohort or nominal coverage differs from qualified evidence")
	}
	for i, group := range d.FirstNames {

		if group.Gender != d.Genders.Categories[i/5].Code || group.Cohort != d.Cohorts[i%5].ID ||
			f.groups[i] == nil || len(group.Entries) != 1000 || len(f.groups[i].entries) != 1000 {
			return nil, fmt.Errorf("invalid USA base first-name group at position %d", i)
		}
		for j, entry := range group.Entries {
			row := f.groups[i].entries[j]
			if entry.ASCIIAtom == nil || entry.Display != row.display || *entry.ASCIIAtom != row.asciiAtom ||
				entry.Weight != row.weight {
				return nil, fmt.Errorf("USA base first name differs from adopted evidence")
			}
		}

	}
	for i, entry := range d.LastNames {
		row := l.names.entries[i]
		if entry.ASCIIAtom == nil || entry.Display != row.display || *entry.ASCIIAtom != row.asciiAtom ||
			entry.Weight != row.weight {
			return nil, fmt.Errorf("USA base surname differs from adopted evidence")
		}
	}

	// Bind city, state, postal associations and L2 Division to one locality.
	geography := &d.Geography
	if geography.PlaceCodeSystem != "us-postal-locality-us-2026a-v2" ||
		geography.TerritorySystem != "sw-us-2026a-census-division-v1" ||
		geography.PostalCodeSystem != "us-geonames-postal-observations-2015-v2" ||
		len(geography.Places) != 127 || len(geography.PostalCodes) != 127 {
		return nil, fmt.Errorf("invalid USA base geography systems or coverage")
	}
	for i, place := range geography.Places {

		location := g.locations[i]
		if place.ID != location.id || place.Name != location.name || place.Population != location.weight ||
			place.Territory != location.state.Division || len(place.Administrative) != 1 {
			return nil, fmt.Errorf("USA base locality differs from adopted geography")
		}
		state := place.Administrative[0]
		if state.System != "us-usps-state-v1" || state.Code == nil || *state.Code != location.state.USPS ||
			state.Name == nil || *state.Name != location.state.Name ||
			!slices.Equal(geography.PostalCodes[place.ID], location.zips) {
			return nil, fmt.Errorf("USA base state or ZIP associations differ from adopted geography")
		}

	}

	// Validate the complete editorial names, then pin the candidate content.
	streets := d.StreetNames
	if streets.Kind != "full-name" || len(streets.Entries) != 256 {
		return nil, fmt.Errorf("invalid USA full-name street coverage")
	}
	prepared := &usaBaseCandidate{
		ages: slices.Clone(a.ages), firstNames: f, lastNames: l, geography: g,
		streets: make([]string, len(streets.Entries)),
	}
	previous := ""
	suffixes := []string{"avenue", "circle", "court", "drive", "lane", "road", "street", "way"}
	for i, street := range streets.Entries {

		if len(street.Text) > 80 || street.Weight != 1 {
			return nil, fmt.Errorf("invalid USA street display bound or weight")
		}
		key, err := italianDisplayKey(street.Text)
		if err != nil {
			return nil, err
		}
		words := strings.Split(key, " ")
		if key <= previous || len(words) < 2 || !slices.Contains(suffixes, words[len(words)-1]) {
			return nil, fmt.Errorf("invalid USA full-name street order or suffix")
		}
		for _, word := range words[:len(words)-1] {
			if slices.Contains(suffixes, word) {
				return nil, fmt.Errorf("repeated USA street suffix")
			}
		}
		prepared.streets[i], previous = street.Text, key

	}
	canonical, err := internationalCanonical(streets)
	if err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(canonical)) != usaStreetAssetSHA256 {
		return nil, fmt.Errorf("USA street lexicon differs from candidate content")
	}
	for _, age := range a.ages {
		prepared.ageWeights = append(prepared.ageWeights, age.Weight)
	}

	// Reuse the international F.4 projections without widening Italian decoding.
	common := internationalBase{
		Schema: d.Schema, Release: d.Release, Profile: d.Profile, Country: d.Country, AnchorDate: d.AnchorDate,
		Genders: d.Genders, Ages: d.Ages, Cohorts: d.Cohorts, LastNames: d.LastNames, StreetNames: d.StreetNames,
		Geography: internationalGeography{
			PlaceCodeSystem: geography.PlaceCodeSystem, TerritorySystem: geography.TerritorySystem,
			PostalCodeSystem: geography.PostalCodeSystem, Places: geography.Places, PostalCodes: geography.PostalCodes,
		},
	}
	for _, group := range d.FirstNames {
		common.FirstNames = append(common.FirstNames, internationalFirstNames{
			Gender: group.Gender, Cohort: group.Cohort, Entries: group.Entries,
		})
	}
	prepared.fingerprints, err = internationalBaseFingerprints(&common)
	if err != nil {
		return nil, err
	}
	prepared.digest, err = internationalDigest("fakedata/base-data/json-v1", d)
	if err != nil {
		return nil, err
	}
	for i, name := range []string{
		ageDatasetComponentName, firstNameDatasetComponentName, lastNameDatasetComponentName,
		placesDatasetComponentName, postalDatasetComponentName, streetNameDatasetComponentName,
	} {
		fingerprint, err := hex.DecodeString(usaBaseSamplingFingerprints[i])
		if err != nil {
			return nil, err
		}
		prepared.dependencies[i], err = BytesComponent(name, fingerprint)
		if err != nil {
			return nil, err
		}
	}

	return prepared, nil
}
