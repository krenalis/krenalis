// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
)

const (
	usaFirstNamesV2AssetSHA256    = "643795bf573521ce6f2902656d85cb11bdc3fcc9a9ef667d22ded9ca5281941e"
	usaFirstNamesV2PolicySHA256   = "bf59d184d913f5684b626bf744f39cbf2063446b437392da72c099a95e543583"
	usaFirstNamesSourceSHA256     = "b763374b9b0ea4a9496f8563721312e8572e5a45136d34244db9ffba666c3326"
	usaFirstNamesSSAAccessSHA256  = "6cbb2e067eacb98f523e1a1dc3383936acaa572de1b6d07f4fd490817e40d02e"
	usaFirstNamesAgeAssetSHA256   = "d840be8206ee31c25d10dc8241d21cc09d199aa45ea5983836be50bb8c3fd398"
	usaFirstNamesBirthYearsSHA256 = "cc8c8b899285f509a0e5ca54fc6194be3e6524979958b651b4ce9763d91280d7"
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

type usaFirstNamesV2Document struct {
	Schema           string                `json:"schema"`
	Status           string                `json:"status"`
	Approval         string                `json:"approval"`
	Country          string                `json:"country"`
	AnchorDate       string                `json:"anchorDate"`
	Vintage          string                `json:"vintage"`
	SourceVariant    string                `json:"sourceVariant"`
	SourceFile       string                `json:"sourceFile"`
	SourceSHA256     string                `json:"sourceSHA256"`
	FrequencyColumns string                `json:"frequencyColumns"`
	PolicySHA256     string                `json:"policySHA256"`
	SSAAccessSHA256  string                `json:"ssaAccessSHA256"`
	AgeAssetSHA256   string                `json:"ageAssetSHA256"`
	BirthYearsSHA256 string                `json:"birthYearsSHA256"`
	Method           string                `json:"method"`
	CohortMeaning    string                `json:"cohortMeaning"`
	WeightKind       string                `json:"weightKind"`
	SourceValidNames uint32                `json:"sourceValidNames"`
	Groups           []usaFirstNameV2Group `json:"groups" max:"10"`
}

type usaFirstNameV2Group struct {
	Gender              Gender              `json:"gender"`
	Cohort              string              `json:"cohort"`
	SelectionLimit      uint32              `json:"selectionLimit"`
	SourcePositiveNames uint32              `json:"sourcePositiveNames"`
	SourceNamedMass     uint64              `json:"sourceNamedMass"`
	SourceOtherMass     uint64              `json:"sourceOtherMass"`
	SelectedSourceMass  uint64              `json:"selectedSourceMass"`
	OmittedNamedMass    uint64              `json:"omittedNamedMass"`
	Total               uint64              `json:"total"`
	Entries             []internationalName `json:"entries" max:"1000"`
}

// decodeUSAFirstNamesV2Candidate reads the closed ten-group candidate; callers
// arrange interruption of blocking reads, as for the common decoder.
func decodeUSAFirstNamesV2Candidate(ctx context.Context, reader io.Reader) (*usaFirstNamesV2Candidate, error) {

	var document usaFirstNamesV2Document
	err := decodeInternationalJSON(ctx, reader, &document)
	if err != nil {
		return nil, err
	}

	return prepareUSAFirstNamesV2Candidate(&document)
}

// prepareUSAFirstNamesV2Candidate verifies the ten ordered Method B groups
// and their exact canonical identity before exposing internal sampling.
func prepareUSAFirstNamesV2Candidate(d *usaFirstNamesV2Document) (*usaFirstNamesV2Candidate, error) {

	// Validate source meanings, fallback identity and the preserved AGE evidence.
	if d == nil || d.Schema != "sw-us-first-names-candidate-v2" || d.Status != "CANDIDATE" ||
		d.Approval != "CANDIDATE: CAPITALIZATION NOT OWNER-APPROVED" ||
		d.Country != "US" || d.AnchorDate != usaSocioeconomicAnchorDate ||
		d.Vintage != "2020" || d.SourceVariant != "standard-complete" || d.SourceFile != "Names2020_FirstNames_Sex.xlsx" ||
		d.SourceSHA256 != usaFirstNamesSourceSHA256 || d.FrequencyColumns != "F:MALE;G:FEMALE;C:TOTAL" ||
		d.PolicySHA256 != usaFirstNamesV2PolicySHA256 || d.SSAAccessSHA256 != usaFirstNamesSSAAccessSHA256 ||
		d.AgeAssetSHA256 != usaFirstNamesAgeAssetSHA256 || d.BirthYearsSHA256 != usaFirstNamesBirthYearsSHA256 ||
		d.Method != "B:CENSUS_ONLY" ||
		d.CohortMeaning != "same sex distribution in all five cohorts; no observed cohort fit" ||
		d.WeightKind != "editorial-top1000-census-sex-count-hamilton-positive-reserving-v2" ||
		d.SourceValidNames != 53615 || len(d.Groups) != 10 {
		return nil, fmt.Errorf("unsupported USA first-name v2 metadata or provenance")
	}
	cohorts := [...]string{"1950-1964", "1965-1979", "1980-1989", "1990-1999", "2000-2005"}
	prepared := &usaFirstNamesV2Candidate{}
	for i, group := range d.Groups {

		gender, positiveNames, namedMass, otherMass := GenderFemale, uint32(51474), uint64(143190891), uint64(11025786)
		if i >= 5 {
			gender, positiveNames, namedMass, otherMass = GenderMale, 47088, 140045939, 7768920
		}
		if group.Gender != gender || group.Cohort != cohorts[i%5] || group.SelectionLimit != 1000 ||
			group.SourcePositiveNames != positiveNames || group.SourceNamedMass != namedMass ||
			group.SourceOtherMass != otherMass || group.SelectedSourceMass == 0 || group.SelectedSourceMass > namedMass ||
			group.OmittedNamedMass != namedMass-group.SelectedSourceMass || group.Total != usaNamesMass {
			return nil, fmt.Errorf("invalid USA first-name group identity or mass at position %d", i)
		}
		names, err := newUSANominalIndex(group.Entries, 1000, usaNamesMass)
		if err != nil {
			return nil, err
		}
		prepared.groups[i] = names

	}

	canonical, err := internationalCanonical(d)
	if err != nil {
		return nil, err
	}
	prepared.identity = sha256.Sum256(canonical)
	if fmt.Sprintf("%x", prepared.identity) != usaFirstNamesV2AssetSHA256 {
		return nil, fmt.Errorf("USA first-name v2 digest mismatch")
	}

	return prepared, nil
}
