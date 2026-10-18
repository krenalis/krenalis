// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"crypto/sha256"
	"math"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
)

const usaCandidateDirectory = "../../synthetic-world-usa-l1-us-2026a-CANDIDATE/assets/"

func TestUSAAgeCandidate(t *testing.T) {

	document := readUSAAgeDocument(t)
	data, err := prepareUSAAgeCandidate(&document)
	if err != nil {
		t.Fatalf("expected qualified USA ages, got %v", err)
	}
	if len(data.ages) != 55 || data.ages[0].Age != 21 || data.ages[54].Age != 75 || data.total != 228029859 {
		t.Fatalf("expected 55 ages 21-75 with total 228029859, got %d/%d", len(data.ages), data.total)
	}
	var total uint64
	for _, age := range data.ages {
		if age.Weight == 0 || age.Weight > math.MaxUint64-total {
			t.Fatalf("expected positive nonoverflowing mass, got %d", age.Weight)
		}
		total += age.Weight
	}
	if total != data.total {
		t.Fatalf("expected total %d, got %d", data.total, total)
	}
	canonical, err := internationalCanonical(document)
	if err != nil {
		t.Fatalf("expected canonical ages, got %v", err)
	}
	if data.identity != sha256.Sum256(canonical) {
		t.Fatalf("expected canonical age identity, got %x", data.identity)
	}
	document.Ages[0].Weight++
	if data.ages[0].Weight == document.Ages[0].Weight {
		t.Fatal("expected owned USA age masses, got input alias")
	}

}

func TestUSAAgeCandidateInvalid(t *testing.T) {

	cases := []struct {
		name   string
		change func(*usaAgeDocument)
	}{
		{"schema", func(d *usaAgeDocument) { d.Schema = "sw-base-data-v1" }},
		{"published", func(d *usaAgeDocument) { d.Status = "RELEASE" }},
		{"country", func(d *usaAgeDocument) { d.Country = "IT" }},
		{"date", func(d *usaAgeDocument) { d.AnchorDate = "2025-07-01" }},
		{"vintage", func(d *usaAgeDocument) { d.Vintage = "2024" }},
		{"universe", func(d *usaAgeDocument) { d.Universe = "resident" }},
		{"sex", func(d *usaAgeDocument) { d.Sex = "male" }},
		{"measure", func(d *usaAgeDocument) { d.Measure = "observed-population" }},
		{"source", func(d *usaAgeDocument) { d.SourceFile = "italian-age_weights.csv" }},
		{"source-hash", func(d *usaAgeDocument) { d.SourceSHA256 = ageWeightsSHA256 }},
		{"missing", func(d *usaAgeDocument) { d.Ages = d.Ages[:54] }},
		{"excess", func(d *usaAgeDocument) { d.Ages = append(d.Ages, d.Ages[0]) }},
		{"duplicate", func(d *usaAgeDocument) { d.Ages[1].Age = 21 }},
		{"aggregate", func(d *usaAgeDocument) { d.Ages[0].Age = 999 }},
		{"order", func(d *usaAgeDocument) { d.Ages[0], d.Ages[1] = d.Ages[1], d.Ages[0] }},
		{"zero", func(d *usaAgeDocument) { d.Ages[0].Weight = 0 }},
		{"overflow", func(d *usaAgeDocument) { d.Ages[0].Weight = math.MaxUint64 }},
		{"total", func(d *usaAgeDocument) { d.Total++ }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			document := readUSAAgeDocument(t)
			test.change(&document)
			encoded, err := json.Marshal(document)
			if err != nil {
				t.Fatalf("expected encoded invalid fixture, got %v", err)
			}
			_, err = decodeUSAAgeCandidate(t.Context(), bytes.NewReader(encoded))
			if err != nil {
				return
			}
			t.Fatal("expected rejected USA ages, got nil")
		})
	}

	_, err := prepareUSAAgeCandidate(nil)
	if err != nil {
		return
	}
	t.Fatal("expected rejected nil ages, got nil")

}

func TestUSACandidateClosedJSON(t *testing.T) {

	for _, filename := range []string{"ages.json", "geography.json"} {
		data, err := os.ReadFile(usaCandidateDirectory + filename)
		if err != nil {
			t.Fatalf("expected candidate asset, got %v", err)
		}
		cases := []string{
			strings.Replace(string(data), "{", "{\"unknown\":1,", 1),
			strings.Replace(string(data), "{", "{\"country\":\"US\",", 1),
			strings.Replace(string(data), "\"total\":", "\"total\":null,\"discarded\":", 1),
			strings.Replace(string(data), "\"weight\":", "\"weight\":1.5,\"discarded\":", 1),
			string(data) + "{}",
		}
		for _, input := range cases {
			if filename == "ages.json" {
				_, err = decodeUSAAgeCandidate(t.Context(), strings.NewReader(input))
			}
			if filename == "geography.json" {
				_, err = decodeUSAGeographyCandidate(t.Context(), strings.NewReader(input))
			}
			if err != nil {
				continue
			}
			t.Fatal("expected rejected nonconforming candidate JSON, got nil")
		}
	}

}

func TestUSAGeographyCandidate(t *testing.T) {

	document := readUSAGeographyDocument(t)
	data, err := prepareUSAGeographyCandidate(&document)
	if err != nil {
		t.Fatalf("expected qualified geography candidate, got %v", err)
	}
	expectedMasses := [9]uint64{
		15431980, 42610074, 47693655, 22069011, 70198706, 20069188, 43566089, 26346149, 53800005,
	}
	if len(data.locations) != 78 || data.total != 341784857 || data.divisionMasses != expectedMasses {
		t.Fatalf("expected 78 localities and reconciled masses, got %d/%d/%v",
			len(data.locations), data.total, data.divisionMasses)
	}
	counts, masses := map[string]int{}, map[string]uint64{}
	zipCount, hasLeadingZero := 0, false
	for _, location := range data.locations {
		counts[location.state.USPS]++
		masses[location.state.USPS] += location.weight
		zipCount += len(location.zips)
		for _, zip := range location.zips {
			if zip[0] == '0' {
				hasLeadingZero = true
			}
		}
	}
	if len(counts) != 51 || zipCount != 4314 || !hasLeadingZero {
		t.Fatalf("expected 51 states, 4314 ZIPs and leading zeros, got %d/%d/%v", len(counts), zipCount, hasLeadingZero)
	}
	for _, entry := range document.States {
		if masses[entry.State.USPS] != entry.ResidentPopulation2025 {
			t.Fatalf("expected state mass %d, got %d", entry.ResidentPopulation2025, masses[entry.State.USPS])
		}
	}

	// Input and resolved ZIP slices cannot mutate the admitted geography.
	first := document.Places[0]
	document.Places[0].ZIPs[0] = "99999"
	location, err := data.resolve(first.ID)
	if err != nil {
		t.Fatalf("expected resolved locality, got %v", err)
	}
	if location.zips[0] == "99999" {
		t.Fatal("expected owned postal associations, got input alias")
	}
	location.zips[0] = "99999"
	resolved, err := data.resolve(first.ID)
	if err != nil {
		t.Fatalf("expected resolved locality, got %v", err)
	}
	if resolved.zips[0] == "99999" {
		t.Fatal("expected immutable admitted ZIPs, got resolved alias")
	}

}

func TestUSAGeographyCandidateInvalid(t *testing.T) {

	cases := []struct {
		name   string
		change func(*usaGeographyDocument)
	}{
		{"country", func(d *usaGeographyDocument) { d.Country = "IT" }},
		{"status", func(d *usaGeographyDocument) { d.Status = "APPROVED" }},
		{"approval", func(d *usaGeographyDocument) { d.Approval = "approved" }},
		{"schema", func(d *usaGeographyDocument) { d.Schema = "sw-base-data-v1" }},
		{"anchor", func(d *usaGeographyDocument) { d.AnchorDate = "2026-07-01" }},
		{"taxonomy", func(d *usaGeographyDocument) { d.TerritorySystem = "sw-it-regions-v1" }},
		{"weight-kind", func(d *usaGeographyDocument) { d.WeightKind = "official-population" }},
		{"postal-date", func(d *usaGeographyDocument) { d.PostalEvidenceDate = "2026-01-01" }},
		{"postal-hash", func(d *usaGeographyDocument) { d.PostalSourceSHA256 = postalCodesSHA256 }},
		{"population-date", func(d *usaGeographyDocument) { d.StatePopulationDate = "2026-01-01" }},
		{"population-hash", func(d *usaGeographyDocument) { d.StatePopulationSourceSHA256 = placesSHA256 }},
		{"missing-state", func(d *usaGeographyDocument) { d.States = d.States[:50] }},
		{"duplicate-state", func(d *usaGeographyDocument) { d.States[1] = d.States[0] }},
		{"state-order", func(d *usaGeographyDocument) { d.States[0], d.States[1] = d.States[1], d.States[0] }},
		{"state-identity", func(d *usaGeographyDocument) { d.States[0].State.GNIS = "00000000" }},
		{"zero-state-mass", func(d *usaGeographyDocument) { d.States[0].ResidentPopulation2025 = 0 }},
		{"state-overflow", func(d *usaGeographyDocument) { d.States[0].ResidentPopulation2025 = math.MaxUint64 }},
		{"missing-locality", func(d *usaGeographyDocument) { d.Places = d.Places[1:] }},
		{"duplicate-locality", func(d *usaGeographyDocument) { d.Places[1] = d.Places[0] }},
		{"locality-order", func(d *usaGeographyDocument) { d.Places[0], d.Places[1] = d.Places[1], d.Places[0] }},
		{"unknown-state", func(d *usaGeographyDocument) { d.Places[0].StateUSPS = "ZZ" }},
		{"incompatible-state", func(d *usaGeographyDocument) { d.Places[0].StateFIPS = "02" }},
		{"division", func(d *usaGeographyDocument) { d.Places[0].Division = "pacific" }},
		{"ordinal", func(d *usaGeographyDocument) { d.Places[0].DivisionOrdinal = 8 }},
		{"id", func(d *usaGeographyDocument) { d.Places[0].ID = "fake" }},
		{"name", func(d *usaGeographyDocument) { d.Places[0].Name = " Montgomery" }},
		{"long-name", func(d *usaGeographyDocument) { d.Places[0].Name = strings.Repeat("x", 181) }},
		{"zero-weight", func(d *usaGeographyDocument) { d.Places[0].Weight = 0 }},
		{"weight-overflow", func(d *usaGeographyDocument) { d.Places[0].Weight = math.MaxUint64 }},
		{"editorial-recipe", func(d *usaGeographyDocument) { d.Places[2].Weight++; d.Places[3].Weight-- }},
		{"total", func(d *usaGeographyDocument) { d.Total++ }},
		{"division-mass", func(d *usaGeographyDocument) { d.DivisionMasses[0]++ }},
		{"zero-division-mass", func(d *usaGeographyDocument) { d.DivisionMasses[0] = 0 }},
		{"missing-division", func(d *usaGeographyDocument) { d.DivisionMasses = d.DivisionMasses[:8] }},
		{"missing-ZIP", func(d *usaGeographyDocument) { d.Places[0].ZIPs = nil }},
		{"short-ZIP", func(d *usaGeographyDocument) { d.Places[0].ZIPs[0] = "0123" }},
		{"long-ZIP", func(d *usaGeographyDocument) { d.Places[0].ZIPs[0] = "012345" }},
		{"nondigit-ZIP", func(d *usaGeographyDocument) { d.Places[0].ZIPs[0] = "０１２３４" }},
		{"zero-ZIP", func(d *usaGeographyDocument) { d.Places[0].ZIPs[0] = "00000" }},
		{"unqualified-ZIP", func(d *usaGeographyDocument) { d.Places[0].ZIPs[0] = "10000" }},
		{"duplicate-ZIP", func(d *usaGeographyDocument) { d.Places[0].ZIPs[1] = d.Places[0].ZIPs[0] }},
		{"ZIP-order", func(d *usaGeographyDocument) {
			d.Places[0].ZIPs[0], d.Places[0].ZIPs[1] = d.Places[0].ZIPs[1], d.Places[0].ZIPs[0]
		}},
		{"contradictory-ZIP", func(d *usaGeographyDocument) { d.Places[1].ZIPs = []string{d.Places[0].ZIPs[0]} }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			document := readUSAGeographyDocument(t)
			test.change(&document)
			_, err := prepareUSAGeographyCandidate(&document)
			if err != nil {
				return
			}
			t.Fatal("expected rejected USA geography, got nil")
		})
	}

	_, err := prepareUSAGeographyCandidate(nil)
	if err != nil {
		return
	}
	t.Fatal("expected rejected nil geography, got nil")

}

func TestUSAGeographySingleSelection(t *testing.T) {

	document := readUSAGeographyDocument(t)
	data, err := prepareUSAGeographyCandidate(&document)
	if err != nil {
		t.Fatalf("expected prepared geography, got %v", err)
	}
	rng, replay, expected := splitMix64{state: 2026}, splitMix64{state: 2026}, splitMix64{state: 2026}
	for range 1000 {
		position, err := expected.WeightedIndex(data.weights)
		if err != nil {
			t.Fatalf("expected weighted locality draw, got %v", err)
		}
		first, err := data.selectLocation(&rng)
		if err != nil {
			t.Fatalf("expected USA locality, got %v", err)
		}
		second, err := data.selectLocation(&replay)
		if err != nil {
			t.Fatalf("expected repeated USA locality, got %v", err)
		}
		place := document.Places[position]
		if first.id != place.ID || first.id != second.id || first.state.USPS != place.StateUSPS ||
			first.state.DivisionOrdinal != place.DivisionOrdinal || rng.state != expected.state {
			t.Fatal("expected one proportional locality draw with its own L2 Division, got independent geography")
		}
	}
	_, err = data.resolve("unknown")
	if err != nil {
		_, err = data.selectLocation(nil)
		if err != nil {
			return
		}
		t.Fatal("expected rejected nil RNG, got nil")
	}
	t.Fatal("expected rejected unknown locality, got nil")

}

func TestUSAStatesDivisions(t *testing.T) {

	// Spec-USA predates this change and is the source of the existing L2 binding.
	spec, err := os.ReadFile("../../SYNTHETIC_WORLD_USA_NUMERIC_PRODUCTION_SPEC_v1_CANDIDATE.md")
	if err != nil {
		t.Fatalf("expected existing Spec-USA, got %v", err)
	}
	rows := regexp.MustCompile("(?m)^\\| ([0-8]) \\| `([a-z-]+)` \\| ([A-Z, ]+) \\|$").FindAllSubmatch(spec, -1)
	if len(rows) != 9 {
		t.Fatalf("expected nine Spec-USA Division rows, got %d", len(rows))
	}
	expected := map[string]uint32{}
	for _, row := range rows {
		ordinal := uint32(row[1][0] - '0')
		if usaInternationalDimensions().Territory.Categories[ordinal].Code != string(row[2]) {
			t.Fatal("expected unchanged L2 Division code, got incompatible binding")
		}
		for _, state := range strings.Split(string(row[3]), ", ") {
			if _, exists := expected[state]; exists {
				t.Fatalf("expected unique Spec-USA state, got %s", state)
			}
			expected[state] = ordinal
		}
	}
	states := usaStates()
	if len(expected) != 51 {
		t.Fatalf("expected 51 states/DC, got %d", len(expected))
	}
	assetBytes, err := os.ReadFile(usaCandidateDirectory + "states-divisions.json")
	if err != nil {
		t.Fatalf("expected standalone state asset, got %v", err)
	}
	var asset struct {
		Schema          string     `json:"schema"`
		Status          string     `json:"status"`
		Country         string     `json:"country"`
		TerritorySystem string     `json:"territorySystem"`
		SourceSHA256    string     `json:"sourceSHA256"`
		States          []usaState `json:"states" max:"51"`
		Divisions       []struct {
			Ordinal    uint32 `json:"ordinal"`
			CensusCode string `json:"censusCode"`
			Code       string `json:"code"`
		} `json:"divisions" max:"9"`
	}
	err = decodeInternationalJSON(t.Context(), bytes.NewReader(assetBytes), &asset)
	if err != nil {
		t.Fatalf("expected decoded standalone state asset, got %v", err)
	}
	if !slices.Equal(asset.States, states[:]) || len(asset.Divisions) != 9 ||
		asset.TerritorySystem != usaInternationalDimensions().Territory.SemanticID {
		t.Fatal("expected exact qualified state asset and existing taxonomy, got incompatible binding")
	}
	for i, division := range asset.Divisions {
		if division.Ordinal != uint32(i) || division.CensusCode != string(rune('1'+i)) ||
			division.Code != usaInternationalDimensions().Territory.Categories[i].Code {
			t.Fatal("expected exact Census codes 1-9 and runtime ordinals 0-8, got incompatible Division")
		}
	}
	for _, state := range states {
		ordinal, ok := expected[state.USPS]
		if !ok || state.DivisionOrdinal != ordinal || !validFixedDigits(state.FIPS, 2) ||
			!validFixedDigits(state.GNIS, 8) {
			t.Fatalf("expected exact Spec-USA state binding, got %+v", state)
		}
		resolved, err := resolveUSAState(state.USPS, state.FIPS)
		if err != nil {
			t.Fatalf("expected resolved state, got %v", err)
		}
		if resolved != state {
			t.Fatalf("expected state %+v, got %+v", state, resolved)
		}
	}
	for _, pair := range [][2]string{{"PR", "72"}, {"GU", "66"}, {"ZZ", "99"}, {"AK", "01"}, {"al", "01"}} {
		_, err := resolveUSAState(pair[0], pair[1])
		if err != nil {
			continue
		}
		t.Fatalf("expected rejected state identifiers, got %v", pair)
	}
	if usaLocalityID("17", "Springfield") == usaLocalityID("25", "Springfield") {
		t.Fatal("expected state-qualified homonymous locality IDs, got identical IDs")
	}

}

func readUSAAgeDocument(t *testing.T) usaAgeDocument {

	t.Helper()
	data, err := os.ReadFile(usaCandidateDirectory + "ages.json")
	if err != nil {
		t.Fatalf("expected AGE asset, got %v", err)
	}
	var document usaAgeDocument
	err = decodeInternationalJSON(t.Context(), bytes.NewReader(data), &document)
	if err != nil {
		t.Fatalf("expected decoded AGE asset, got %v", err)
	}
	prepared, err := decodeUSAAgeCandidate(t.Context(), bytes.NewReader(data))
	if err != nil {
		t.Fatalf("expected admitted AGE asset, got %v", err)
	}
	if !slices.Equal(prepared.ages, document.Ages) {
		t.Fatal("expected preserved original age masses, got transformed weights")
	}

	return document
}

func readUSAGeographyDocument(t *testing.T) usaGeographyDocument {

	t.Helper()
	data, err := os.ReadFile(usaCandidateDirectory + "geography.json")
	if err != nil {
		t.Fatalf("expected geography asset, got %v", err)
	}
	var document usaGeographyDocument
	err = decodeInternationalJSON(t.Context(), bytes.NewReader(data), &document)
	if err != nil {
		t.Fatalf("expected decoded geography asset, got %v", err)
	}
	_, err = decodeUSAGeographyCandidate(t.Context(), bytes.NewReader(data))
	if err != nil {
		t.Fatalf("expected admitted geography asset, got %v", err)
	}

	return document
}
