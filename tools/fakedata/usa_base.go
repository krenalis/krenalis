// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
)

const (
	usaBaseRelease = "us-2026a-layer1-v2-candidate"
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
