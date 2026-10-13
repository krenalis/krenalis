// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"context"
	"fmt"
	"io"
	"math"
	"slices"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// ItalianBaseData is an immutable validated Italian international v1 document.
// Its zero value is invalid; prepare it with PrepareItalianWorldData.
type ItalianBaseData struct {
	document    internationalBase
	initialized bool
}

// DecodeItalianBaseData decodes and validates the Italian base v1 profile.
// It returns a context error when cancellation is observed between reads;
// callers must arrange interruption of any blocking read themselves.
// Malformed JSON tokens return a tools/json SyntaxError.
func DecodeItalianBaseData(ctx context.Context, reader io.Reader) (*ItalianBaseData, error) {
	return decodeItalianBaseData(ctx, reader)
}

// CanonicalJSON returns the document's C1 serialization with its final LF.
func (d *ItalianBaseData) CanonicalJSON() ([]byte, error) {
	if d == nil || !d.initialized {
		return nil, fmt.Errorf("uninitialized Italian base document")
	}
	return internationalCanonical(d.document)
}

// Fingerprints returns the closed F.4 projections in the order ages, first
// names, last names, places, postal codes, street names.
func (d *ItalianBaseData) Fingerprints() ([6][32]byte, error) {
	return d.fingerprints()
}

// LogicalDigest returns the international document's KFD1/C1 digest.
func (d *ItalianBaseData) LogicalDigest() ([32]byte, error) {
	if d == nil || !d.initialized {
		return [32]byte{}, fmt.Errorf("uninitialized Italian base document")
	}
	return internationalDigest("fakedata/base-data/json-v1", d.document)
}

func (d *ItalianBaseData) fingerprints() ([6][32]byte, error) {

	if d == nil || !d.initialized {
		return [6][32]byte{}, fmt.Errorf("uninitialized Italian base document")
	}

	base := &d.document
	genders := internationalTaxonomyFor(base.Genders.SemanticID, []string{"female", "male"})
	projections := []map[string]any{
		{"anchorDate": base.AnchorDate, "ages": base.Ages},
		{"genders": genders, "cohorts": base.Cohorts, "firstNames": base.FirstNames},
		{"lastNames": base.LastNames},
		{"placeCodeSystem": base.Geography.PlaceCodeSystem, "territorySystem": base.Geography.TerritorySystem,
			"places": base.Geography.Places},
		{"placeCodeSystem": base.Geography.PlaceCodeSystem, "postalCodeSystem": base.Geography.PostalCodeSystem,
			"postalCodes": base.Geography.PostalCodes},
		{"streetNames": base.StreetNames},
	}

	country, err := StringComponent("country", base.Country)
	if err != nil {
		return [6][32]byte{}, err
	}
	profile, err := StringComponent("profile", base.Profile)
	if err != nil {
		return [6][32]byte{}, err
	}

	var fingerprints [6][32]byte
	for i, family := range []string{"ages", "first-names", "last-names", "places", "postal-codes", "street-names"} {
		fingerprints[i], err = internationalDigest("fakedata/layer1-json/v1/"+family, projections[i], country, profile)
		if err != nil {
			return [6][32]byte{}, err
		}
	}

	return fingerprints, nil
}

func decodeItalianBaseData(ctx context.Context, reader io.Reader) (*ItalianBaseData, error) {
	var document internationalBase
	err := decodeInternationalJSON(ctx, reader, &document)
	if err != nil {
		return nil, err
	}
	err = validateItalianBase(&document)
	if err != nil {
		return nil, err
	}
	return &ItalianBaseData{document: document, initialized: true}, nil
}

func internationalASCIIAtom(value string) bool {
	if value == "" || value[0] == '-' || value[len(value)-1] == '-' || strings.Contains(value, "--") {
		return false
	}
	for _, c := range value {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return true
}

// italianDisplayKey validates, but never replaces, the original display.
// Unicode-16 repertoire and normalization/folding conformance are checked by
// international_unicode_test.go against the pinned UCD-derived fixture.
func italianDisplayKey(value string) (string, error) {

	if !internationalText(value) || strings.TrimSpace(value) != value || strings.Contains(value, "  ") {
		return "", fmt.Errorf("invalid Italian display form")
	}
	for _, c := range value {
		if !strings.ContainsRune(italianUnicode16Repertoire, c) {
			return "", fmt.Errorf("Italian display outside Unicode-16 repertoire")
		}
	}
	if italianNFC(value) != value {
		return "", fmt.Errorf("Italian display is not NFC")
	}

	base, hasLetter := false, false
	for _, scalar := range value {
		for _, c := range norm.NFD.String(string(scalar)) {
			switch {
			case c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z':
				base = true
				hasLetter = true
			case c >= 0x300 && c <= 0x36f && c != 0x34f:
				if !base {
					return "", fmt.Errorf("orphan Italian display mark")
				}
			case c == ' ' || c == '\'' || c == '-':
				base = false
			default:
				return "", fmt.Errorf("unsupported Italian display scalar")
			}
		}
	}
	if !hasLetter {
		return "", fmt.Errorf("Italian display has no letters")
	}

	return italianNFC(cases.Fold().String(value)), nil
}

func italianInternationalCohorts() []internationalCohort {
	return []internationalCohort{
		{"1950-1964", 1950, 1964}, {"1965-1979", 1965, 1979}, {"1980-1989", 1980, 1989},
		{"1990-1999", 1990, 1999}, {"2000-2005", 2000, 2005},
	}
}

// italianNFC normalizes only the bounded D1-B repertoire and its fold closure.
// x/text inserts CGJ after 30 nonstarters in whole-string normalization; D1-B
// requires ordinary NFC. Reuse its verified scalar decomposition, CCC and pair
// composition tables while preserving arbitrarily long admitted mark sequences.
func italianNFC(value string) string {

	var decomposed []rune
	var classes []uint8
	for _, scalar := range value {
		for _, c := range norm.NFD.String(string(scalar)) {
			class := norm.NFD.PropertiesString(string(c)).CCC()
			decomposed = append(decomposed, c)
			classes = append(classes, class)
			for i := len(classes) - 1; i > 0 && class != 0 && classes[i-1] > class; i-- {
				decomposed[i-1], decomposed[i] = decomposed[i], decomposed[i-1]
				classes[i-1], classes[i] = classes[i], classes[i-1]
			}
		}
	}

	composed := make([]rune, 0, len(decomposed))
	starter := -1
	var previous uint8
	for i, c := range decomposed {
		class := classes[i]
		if starter >= 0 && (previous == 0 || previous < class) {
			pair := []rune(norm.NFC.String(string(composed[starter]) + string(c)))
			if len(pair) == 1 {
				composed[starter] = pair[0]
				continue
			}
		}
		composed = append(composed, c)
		if class == 0 {
			starter = len(composed) - 1
		}
		previous = class
	}

	return string(composed)
}

func validateItalianBase(d *internationalBase) error {

	if d.Schema != "sw-base-data-v1" || d.Profile != "it-2026a-base-v1" || d.Country != "IT" ||
		d.AnchorDate != PopulationAnchorDate || !internationalToken(d.Release) {
		return fmt.Errorf("unsupported Italian base metadata")
	}
	genders := internationalTaxonomyFor("sw-l1-gender-v1", []string{"female", "male"})
	err := validateInternationalTaxonomy(d.Genders, genders)
	if err != nil {
		return err
	}
	if !slices.Equal(d.Cohorts, italianInternationalCohorts()) || len(d.Ages) != 55 {
		return fmt.Errorf("invalid Italian age or cohort coverage")
	}
	var ageTotal uint64
	for i, age := range d.Ages {
		if age.Age != uint32(i+21) || age.Weight == 0 || math.MaxUint64-ageTotal < age.Weight {
			return fmt.Errorf("invalid age or weight at position %d", i)
		}
		ageTotal += age.Weight
	}

	// Validate complete, ordered gender × cohort groups and all nominal entries.
	if len(d.FirstNames) != 10 {
		return fmt.Errorf("expected ten Italian first-name groups")
	}
	for i, group := range d.FirstNames {
		if group.Gender != d.Genders.Categories[i/5].Code || group.Cohort != d.Cohorts[i%5].ID {
			return fmt.Errorf("invalid first-name group order at position %d", i)
		}
		err = validateItalianNames(group.Entries, 80, 20, 40, 20)
		if err != nil {
			return fmt.Errorf("first-name group %d: %w", i, err)
		}
	}
	err = validateItalianNames(d.LastNames, 600, 240, 240, 120)
	if err != nil {
		return fmt.Errorf("last names: %w", err)
	}

	// Validate light geography and the exact postal coverage, without hierarchies.
	geography := &d.Geography
	if geography.PlaceCodeSystem != "it-istat-municipality-it-2026a" ||
		geography.TerritorySystem != "sw-it-2026a-region-v1" ||
		geography.PostalCodeSystem != "it-postal-observations-it-2026a" ||
		len(geography.Places) != 7896 || len(geography.PostalCodes) != 7896 {
		return fmt.Errorf("invalid Italian geography systems or cardinality")
	}
	var population uint64
	previous := ""
	regions := map[string]bool{}
	provinceNames := map[string]string{}
	for i, place := range geography.Places {

		if !validFixedDigits(place.ID, 6) || place.ID <= previous || !internationalText(place.Name) ||
			!validDisplayName(place.Name) || place.Population == 0 || math.MaxUint64-population < place.Population ||
			!slices.Contains(socioeconomicRegions[:], place.Territory) || len(place.Administrative) != 1 {
			return fmt.Errorf("invalid Italian place at position %d", i)
		}
		previous = place.ID
		population += place.Population
		regions[place.Territory] = true
		province := place.Administrative[0]
		if province.System != "it-provincial-uts-it-2026a" || province.Code == nil || !validStateProv(*province.Code) {
			return fmt.Errorf("invalid provincial attribute at place %d", i)
		}
		if province.Name != nil {
			if !internationalText(*province.Name) {
				return fmt.Errorf("invalid provincial name")
			}
			if name, exists := provinceNames[*province.Code]; exists && name != *province.Name {
				return fmt.Errorf("inconsistent provincial names")
			}
			provinceNames[*province.Code] = *province.Name
		}

		codes := geography.PostalCodes[place.ID]
		if len(codes) == 0 || len(codes) > 8 {
			return fmt.Errorf("invalid postal coverage at place %d", i)
		}
		for j, code := range codes {
			if !validFixedDigits(code, 5) || j > 0 && code <= codes[j-1] {
				return fmt.Errorf("invalid postal order or code at place %d", i)
			}
		}

	}
	if len(regions) != len(socioeconomicRegions) {
		return fmt.Errorf("incomplete Italian territory coverage")
	}

	// Validate the street lexicon, D1-B ordering and exact weight quotas.
	if d.StreetNames.Kind != "fragment" || len(d.StreetNames.Entries) != 400 {
		return fmt.Errorf("invalid Italian street lexicon")
	}
	previous = ""
	weights := make([]uint64, len(d.StreetNames.Entries))
	for i, street := range d.StreetNames.Entries {
		key, err := italianDisplayKey(street.Text)
		if err != nil {
			return fmt.Errorf("street %d: %w", i, err)
		}
		first, _, _ := strings.Cut(key, " ")
		if key <= previous || slices.Contains([]string{
			"via", "viale", "piazza", "corso", "largo", "vicolo", "strada", "piazzale", "piazzetta",
			"lungomare", "contrada", "località",
		}, first) {
			return fmt.Errorf("invalid street order or type prefix at position %d", i)
		}
		previous = key
		weights[i] = street.Weight
	}
	if !validTierQuota(weights, 160, 160, 80) {
		return fmt.Errorf("invalid Italian street weight quotas")
	}

	return nil
}

func validateItalianNames(entries []internationalName, count, ones, twos, fours int) error {
	if len(entries) != count {
		return fmt.Errorf("expected %d name entries", count)
	}
	weights := make([]uint64, len(entries))
	previous := ""
	for i, entry := range entries {
		key, err := italianDisplayKey(entry.Display)
		if err != nil {
			return fmt.Errorf("entry %d: %w", i, err)
		}
		if key <= previous || entry.ASCIIAtom == nil || !internationalASCIIAtom(*entry.ASCIIAtom) {
			return fmt.Errorf("invalid name order, duplicate key or ASCII atom at position %d", i)
		}
		previous = key
		weights[i] = entry.Weight
	}
	if !validTierQuota(weights, ones, twos, fours) {
		return fmt.Errorf("invalid Italian name weight quotas")
	}
	return nil
}
