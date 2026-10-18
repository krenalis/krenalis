// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"math"
	"math/big"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
)

const usaGeographyV2Directory = "../../synthetic-world-usa-l1-us-2026a-GEOGRAPHY-v2-CANDIDATE/"

// TestUSAGeographyV2Admission checks coverage, masses and owned postal slices.
func TestUSAGeographyV2Admission(t *testing.T) {

	document := readUSAGeographyV2Document(t)
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("expected reordered transport JSON, got %v", err)
	}
	indented, err := json.Indent(encoded, "", "  ")
	if err != nil {
		t.Fatalf("expected indented transport JSON, got %v", err)
	}
	data, err := decodeUSAGeographyV2Candidate(t.Context(), bytes.NewReader(indented))
	if err != nil {
		t.Fatalf("expected admitted v2 geography, got %v", err)
	}
	expectedDivisions := [9]uint64{
		15431980, 42610074, 47693655, 22069011, 70198706, 20069188, 43566089, 26346149, 53800005,
	}
	if len(data.locations) != 127 || data.total != 341784857 || data.divisionMasses != expectedDivisions ||
		fmt.Sprintf("%x", data.identity) != usaGeographyV2AssetSHA256 {
		t.Fatalf("expected identified 127-locality v2 with reconciled masses, got %d/%d/%v/%x",
			len(data.locations), data.total, data.divisionMasses, data.identity)
	}
	states := map[string]uint64{}
	zips := map[string]string{}
	var divisions [9]uint64
	hasLeadingZero := false
	for _, place := range data.locations {
		states[place.state.USPS] += place.weight
		divisions[place.state.DivisionOrdinal] += place.weight
		for _, zip := range place.zips {
			if owner, exists := zips[zip]; exists {
				t.Fatalf("expected unique ZIP owner, got %s/%s", owner, place.id)
			}
			zips[zip] = place.id
			if zip[0] == '0' {
				hasLeadingZero = true
			}
		}
	}
	if len(states) != 51 || len(zips) != 5917 || !hasLeadingZero || divisions != expectedDivisions {
		t.Fatalf("expected 51 states, 5917 ZIPs, leading zeros and nine Division masses, got %d/%d/%v/%v",
			len(states), len(zips), hasLeadingZero, divisions)
	}
	for _, entry := range document.States {
		if states[entry.State.USPS] != entry.ResidentPopulation2025 {
			t.Fatalf("expected state mass %d, got %d", entry.ResidentPopulation2025, states[entry.State.USPS])
		}
	}

	// Both admission and resolution own their postal slices.
	document.Places[0].ZIPs[0] = "99999"
	first, err := data.resolve(document.Places[0].ID)
	if err != nil {
		t.Fatalf("expected resolved locality, got %v", err)
	}
	if first.zips[0] == "99999" {
		t.Fatal("expected owned input ZIPs, got aliased slice")
	}
	first.zips[0] = "99999"
	second, err := data.resolve(first.id)
	if err != nil {
		t.Fatalf("expected resolved locality, got %v", err)
	}
	if second.zips[0] == "99999" {
		t.Fatal("expected owned resolved ZIPs, got aliased slice")
	}

}

// TestUSAGeographyV2ClosedJSON checks the bounded, required-field v2 format.
func TestUSAGeographyV2ClosedJSON(t *testing.T) {

	data, err := os.ReadFile(usaGeographyV2Directory + "assets/geography.json")
	if err != nil {
		t.Fatalf("expected original v2 JSON, got %v", err)
	}
	for _, test := range []struct{ name, old, replacement string }{
		{"unknown", `"schema":`, `"extra":0,"schema":`},
		{"duplicate", `"schema":`, `"\u0073chema":"sw-us-geography-candidate-v2","schema":`},
		{"missing policy", `"policySHA256":"` + usaGeographyV2PolicySHA256 + `",`, ``},
		{"missing nullable population", `"municipalPopulation2025":null,`, ``},
		{"missing observed population", `"municipalPopulation2025":195300,`, ``},
		{"missing score", `"allocationScore":195300,`, ``},
		{"missing score kind", `"scoreKind":"census-municipal-resident-population",`, ``},
		{"null score", `"allocationScore":195300`, `"allocationScore":null`},
		{"negative", `"allocationScore":195300`, `"allocationScore":-1`},
		{"negative zero", `"allocationScore":195300`, `"allocationScore":-0`},
		{"fraction", `"allocationScore":195300`, `"allocationScore":195300.5`},
		{"exponent", `"allocationScore":195300`, `"allocationScore":195300e0`},
		{"quoted", `"allocationScore":195300`, `"allocationScore":"195300"`},
		{"boolean", `"allocationScore":195300`, `"allocationScore":true`},
		{"uint64 overflow", `"allocationScore":195300`, `"allocationScore":18446744073709551616`},
		{"population overflow", `"municipalPopulation2025":195300`, `"municipalPopulation2025":18446744073709551616`},
		{"ordinal overflow", `"divisionOrdinal":5`, `"divisionOrdinal":4294967296`},
		{"unknown municipal attribute", `"municipalPopulation2025":195300`,
			`"municipalPopulation2025":195300,"population":195300`},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := bytes.Replace(data, []byte(test.old), []byte(test.replacement), 1)
			if bytes.Equal(input, data) {
				t.Fatal("expected fixture substitution, got unchanged bytes")
			}
			_, err := decodeUSAGeographyV2Candidate(t.Context(), bytes.NewReader(input))
			if err != nil {
				return
			}
			t.Fatal("expected rejected v2 JSON, got nil")
		})
	}
	inputs := []string{string(data) + "{}", string(data) + strings.Repeat(" ", maxInternationalDocumentBytes)}
	for _, input := range inputs {
		_, err := decodeUSAGeographyV2Candidate(t.Context(), strings.NewReader(input))
		if err != nil {
			continue
		}
		t.Fatal("expected rejected multiple or oversized documents, got nil")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = decodeUSAGeographyV2Candidate(ctx, bytes.NewReader(data))
	if err != nil {
		return
	}
	t.Fatal("expected canceled admission rejected, got nil")

}

// TestUSAGeographyV2Hamilton independently verifies every state's allocation.
func TestUSAGeographyV2Hamilton(t *testing.T) {

	document := readUSAGeographyV2Document(t)
	// Use arbitrary-precision rational quotas, independently of Go admission's
	// Mul64/Div64 path and the producer's integer divmod implementation.
	for _, entry := range document.States {

		var indices []int
		scoreTotal := &big.Int{}
		for index, place := range document.Places {
			if place.StateUSPS == entry.State.USPS {
				indices = append(indices, index)
				scoreTotal.Add(scoreTotal, (&big.Int{}).SetUint64(place.AllocationScore))
			}
		}
		weights := map[int]uint64{}
		fractions := map[int]*big.Rat{}
		var allocated uint64
		for _, index := range indices {
			numerator := (&big.Int{}).Mul((&big.Int{}).SetUint64(entry.ResidentPopulation2025),
				(&big.Int{}).SetUint64(document.Places[index].AllocationScore))
			quota := (&big.Rat{}).SetFrac(numerator, scoreTotal)
			floor := (&big.Int{}).Quo(quota.Num(), quota.Denom())
			weights[index] = floor.Uint64()
			allocated += weights[index]
			fractions[index] = (&big.Rat{}).Sub(quota, (&big.Rat{}).SetInt(floor))
		}
		slices.SortFunc(indices, func(a, b int) int {
			if order := fractions[b].Cmp(fractions[a]); order != 0 {
				return order
			}
			return strings.Compare(document.Places[a].Name, document.Places[b].Name)
		})
		for rank, index := range indices {
			if uint64(rank) < entry.ResidentPopulation2025-allocated {
				weights[index]++
			}
			if weights[index] != document.Places[index].Weight {
				t.Fatalf("expected independent Hamilton weight %d for %s, got %d",
					weights[index], document.Places[index].ID, document.Places[index].Weight)
			}
		}

	}

}

// TestUSAGeographyV2HamiltonTies checks the exact postal-name tie breaker.
func TestUSAGeographyV2HamiltonTies(t *testing.T) {

	document := readUSAGeographyV2Document(t)
	var indices []int
	var mass uint64
	for _, entry := range document.States {
		if entry.State.USPS == "CA" {
			mass = entry.ResidentPopulation2025
		}
	}
	for index := range document.Places {
		place := &document.Places[index]
		if place.StateUSPS == "CA" {
			indices = append(indices, index)
			*place.MunicipalPopulation2025 = 250000
			place.AllocationScore = 250000
		}
	}
	if len(indices) != 15 || mass%uint64(len(indices)) == 0 {
		t.Fatalf("expected 15 California localities with a nonzero remainder, got %d/%d", len(indices), mass)
	}
	for rank, index := range indices {
		document.Places[index].Weight = mass / uint64(len(indices))
		if uint64(rank) < mass%uint64(len(indices)) {
			document.Places[index].Weight++
		}
	}
	_, err := prepareUSAGeographyV2Candidate(&document)
	if err != nil {
		if !strings.Contains(err.Error(), "owner-adopted asset") {
			t.Fatalf("expected postal-name ties to pass Hamilton before identity rejection, got %v", err)
		}
		first, last := indices[0], indices[len(indices)-1]
		document.Places[first].Weight--
		document.Places[last].Weight++
		_, err = prepareUSAGeographyV2Candidate(&document)
		if err != nil {
			if !strings.Contains(err.Error(), "Hamilton allocation") {
				t.Fatalf("expected Hamilton tie rejection, got %v", err)
			}
			return
		}
		t.Fatal("expected reversed Hamilton tie rejected, got nil")
	}
	t.Fatal("expected unapproved tied-score snapshot rejected, got nil")

}

// TestUSAGeographyV2Identity checks the adopted evidence and declared files.
func TestUSAGeographyV2Identity(t *testing.T) {

	for _, entry := range []struct{ path, digest string }{
		{usaGeographyV2Directory + "manifest.json", "e614ca38170cc4856b1d568d503948469810d88e2b6a1ceae280b5c469a78e3a"},
		{usaGeographyV2Directory + "locality-policy.json", usaGeographyV2PolicySHA256},
		{usaGeographyV2Directory + "assets/geography.json", usaGeographyV2AssetSHA256},
		{usaGeographyV2Directory + "sources.json", "8d61d05261df4ca1f2acf02003ccf0c7c3f225ea517549ea765fbd9ebb0dfdfa"},
		{"../../SYNTHETIC_WORLD_USA_LAYER1_GEOGRAPHY_DEMO_PLAUSIBILITY_CORRECTIVE_v2_REPORT.md",
			"22df56423e57cd26a2c35f801bfc1776427259203d4d252c0f797a1f9a3a4c81"},
	} {
		data, err := os.ReadFile(entry.path)
		if err != nil {
			t.Fatalf("expected adopted evidence, got %v", err)
		}
		if digest := fmt.Sprintf("%x", sha256.Sum256(data)); digest != entry.digest {
			t.Fatalf("expected digest %s, got %s", entry.digest, digest)
		}
	}
	data, err := os.ReadFile(usaGeographyV2Directory + "manifest.json")
	if err != nil {
		t.Fatalf("expected candidate manifest, got %v", err)
	}
	var manifest struct {
		ContentSHA256 string `json:"contentSHA256"`
		Files         map[string]struct {
			SHA256 string `json:"sha256"`
			Size   uint64 `json:"size"`
		} `json:"files"`
	}
	err = json.Unmarshal(data, &manifest)
	if err != nil {
		t.Fatalf("expected decoded manifest, got %v", err)
	}
	canonical, err := internationalCanonical(manifest.Files)
	if err != nil {
		t.Fatalf("expected canonical file map, got %v", err)
	}
	expected := "116edb73bc75c5b01a9ffde41b99cc654e064b6437910bd72a44a458f1f6c18a"
	if len(manifest.Files) != 13 || manifest.ContentSHA256 != expected ||
		fmt.Sprintf("%x", sha256.Sum256(canonical)) != expected {
		t.Fatalf("expected 13 declared files and adopted content digest, got %d/%s",
			len(manifest.Files), manifest.ContentSHA256)
	}
	// Qualify only declared paths; Finder metadata is outside this content set.
	for name, entry := range manifest.Files {
		data, err := os.ReadFile(usaGeographyV2Directory + name)
		if err != nil {
			t.Fatalf("expected declared file %s, got %v", name, err)
		}
		if uint64(len(data)) != entry.Size || fmt.Sprintf("%x", sha256.Sum256(data)) != entry.SHA256 {
			t.Fatalf("expected declared size/digest for %s, got %d/%x", name, len(data), sha256.Sum256(data))
		}
	}

}

// TestUSAGeographyV2Invalid checks recipe, provenance and identity failures.
func TestUSAGeographyV2Invalid(t *testing.T) {

	original := readUSAGeographyV2Document(t)
	honolulu := slices.IndexFunc(original.Places, func(p usaPlaceV2Candidate) bool { return p.Name == "Honolulu" })
	if honolulu < 0 {
		t.Fatal("expected Honolulu fixture, got missing locality")
	}
	cases := []struct {
		name   string
		change func(*usaGeographyV2Document)
		cause  string
	}{
		{"schema", func(d *usaGeographyV2Document) { d.Schema = "sw-us-geography-candidate-v1" }, "metadata"},
		{"status", func(d *usaGeographyV2Document) { d.Status = "OWNER-APPROVED" }, "metadata"},
		{"approval", func(d *usaGeographyV2Document) { d.Approval = "OWNER-APPROVED" }, "metadata"},
		{"country", func(d *usaGeographyV2Document) { d.Country = "IT" }, "metadata"},
		{"anchor", func(d *usaGeographyV2Document) { d.AnchorDate = "2026-07-01" }, "metadata"},
		{"weight kind", func(d *usaGeographyV2Document) { d.WeightKind = usaGeographyWeightKind }, "metadata"},
		{"policy", func(d *usaGeographyV2Document) { d.PolicySHA256 = strings.Repeat("0", 64) }, "metadata"},
		{"postal date", func(d *usaGeographyV2Document) { d.PostalEvidenceDate = "2026-01-01" }, "metadata"},
		{"postal source", func(d *usaGeographyV2Document) { d.PostalSourceSHA256 = strings.Repeat("0", 64) }, "metadata"},
		{"state date", func(d *usaGeographyV2Document) { d.StatePopulationDate = "2026-01-01" }, "metadata"},
		{"state source", func(d *usaGeographyV2Document) { d.StatePopulationSourceSHA256 = usaAgeSourceSHA256 }, "metadata"},
		{"municipal date", func(d *usaGeographyV2Document) { d.MunicipalPopulationDate = "2024-07-01" }, "metadata"},
		{"municipal source", func(d *usaGeographyV2Document) {
			d.MunicipalPopulationSourceSHA256 = usaStatePopulationSHA256
		}, "metadata"},
		{"territory system", func(d *usaGeographyV2Document) { d.TerritorySystem = "sw-it-regions-v1" }, "territory"},
		{"AGE denominator", func(d *usaGeographyV2Document) { d.Total = 228029859 }, "total"},
		{"total overflow", func(d *usaGeographyV2Document) { d.Total = math.MaxUint64 }, "total"},
		{"missing state", func(d *usaGeographyV2Document) { d.States = d.States[:50] }, "coverage"},
		{"duplicate state", func(d *usaGeographyV2Document) { d.States[1] = d.States[0] }, "state identity"},
		{"state order", func(d *usaGeographyV2Document) {
			d.States[0], d.States[1] = d.States[1], d.States[0]
		}, "state identity"},
		{"state FIPS", func(d *usaGeographyV2Document) { d.States[0].State.FIPS = "02" }, "state identity"},
		{"state GNIS", func(d *usaGeographyV2Document) { d.States[0].State.GNIS = "00000000" }, "state identity"},
		{"state Division", func(d *usaGeographyV2Document) { d.States[0].State.Division = "pacific" }, "state identity"},
		{"state ordinal", func(d *usaGeographyV2Document) { d.States[0].State.DivisionOrdinal = 9 }, "state identity"},
		{"state zero", func(d *usaGeographyV2Document) { d.States[0].ResidentPopulation2025 = 0 }, "state identity or mass"},
		{"state overflow", func(d *usaGeographyV2Document) {
			d.States[0].ResidentPopulation2025 = math.MaxUint64
		}, "state identity or mass"},
		{"state total", func(d *usaGeographyV2Document) { d.States[0].ResidentPopulation2025-- }, "reference total"},
		{"missing locality", func(d *usaGeographyV2Document) { d.Places = d.Places[1:] }, "coverage"},
		{"duplicate locality", func(d *usaGeographyV2Document) { d.Places[1] = d.Places[0] }, "locality identity"},
		{"locality order", func(d *usaGeographyV2Document) {
			d.Places[0], d.Places[1] = d.Places[1], d.Places[0]
		}, "locality identity"},
		{"unknown state", func(d *usaGeographyV2Document) { d.Places[0].StateUSPS = "ZZ" }, "state identifiers"},
		{"incompatible FIPS", func(d *usaGeographyV2Document) { d.Places[0].StateFIPS = "02" }, "state identifiers"},
		{"locality Division", func(d *usaGeographyV2Document) { d.Places[0].Division = "pacific" }, "locality identity"},
		{"locality ordinal", func(d *usaGeographyV2Document) {
			d.Places[0].DivisionOrdinal = math.MaxUint32
		}, "locality identity"},
		{"ID collision", func(d *usaGeographyV2Document) { d.Places[1].ID = d.Places[0].ID }, "locality identity"},
		{"name whitespace", func(d *usaGeographyV2Document) { d.Places[0].Name = " Montgomery" }, "locality identity"},
		{"long name", func(d *usaGeographyV2Document) { d.Places[0].Name = strings.Repeat("x", 181) }, "locality identity"},
		{"invalid UTF8", func(d *usaGeographyV2Document) { d.Places[0].Name = "\xff" }, "locality identity"},
		{"municipal missing", func(d *usaGeographyV2Document) {
			d.Places[0].MunicipalPopulation2025 = nil
		}, "municipal observation"},
		{"municipal zero", func(d *usaGeographyV2Document) {
			*d.Places[0].MunicipalPopulation2025 = 0
		}, "municipal observation"},
		{"municipal overflow", func(d *usaGeographyV2Document) {
			*d.Places[0].MunicipalPopulation2025 = math.MaxUint64
		}, "municipal observation"},
		{"municipal exceeds state", func(d *usaGeographyV2Document) {
			*d.Places[0].MunicipalPopulation2025 = d.States[0].ResidentPopulation2025 + 1
		}, "municipal observation"},
		{"municipal contradiction", func(d *usaGeographyV2Document) {
			*d.Places[0].MunicipalPopulation2025++
		}, "municipal observation"},
		{"score zero", func(d *usaGeographyV2Document) { d.Places[0].AllocationScore = 0 }, "municipal observation"},
		{"score overflow", func(d *usaGeographyV2Document) {
			d.Places[0].AllocationScore = math.MaxUint64
		}, "municipal observation"},
		{"observed as editorial", func(d *usaGeographyV2Document) {
			d.Places[0].ScoreKind = "editorial-source-coverage-floor"
		}, "municipal observation"},
		{"unauthorized fallback", func(d *usaGeographyV2Document) {
			d.Places[0].MunicipalPopulation2025 = nil
			d.Places[0].AllocationScore = 20000
			d.Places[0].ScoreKind = "editorial-source-coverage-floor"
		}, "municipal observation"},
		{"editorial as observed", func(d *usaGeographyV2Document) {
			d.Places[honolulu].ScoreKind = "census-municipal-resident-population"
		}, "editorial fallback"},
		{"CDP as municipal", func(d *usaGeographyV2Document) {
			population := uint64(341868)
			d.Places[honolulu].MunicipalPopulation2025 = &population
			d.Places[honolulu].AllocationScore = population
			d.Places[honolulu].ScoreKind = "census-municipal-resident-population"
		}, "editorial fallback"},
		{"editorial population", func(d *usaGeographyV2Document) {
			d.Places[honolulu].MunicipalPopulation2025 = &d.Places[honolulu].AllocationScore
		}, "editorial fallback"},
		{"fallback score", func(d *usaGeographyV2Document) { d.Places[honolulu].AllocationScore++ }, "editorial fallback"},
		{"zero weight", func(d *usaGeographyV2Document) { d.Places[0].Weight = 0 }, "nonpositive or overflowing"},
		{"overflowing weights", func(d *usaGeographyV2Document) {
			d.Places[1].Weight = math.MaxUint64
		}, "nonpositive or overflowing"},
		{"sum preserving wrong weights", func(d *usaGeographyV2Document) {
			d.Places[1].Weight++
			d.Places[2].Weight--
		}, "Hamilton allocation"},
		{"Division mass", func(d *usaGeographyV2Document) { d.DivisionMasses[0]++ }, "Division mass"},
		{"missing Division", func(d *usaGeographyV2Document) { d.DivisionMasses = d.DivisionMasses[:8] }, "coverage"},
		{"ZIP absent", func(d *usaGeographyV2Document) { d.Places[0].ZIPs = nil }, "postal associations"},
		{"excess ZIPs", func(d *usaGeographyV2Document) { d.Places[0].ZIPs = make([]string, 1025) }, "postal associations"},
		{"ZIP format", func(d *usaGeographyV2Document) { d.Places[0].ZIPs[0] = "1234" }, "ZIP"},
		{"ZIP nonASCII", func(d *usaGeographyV2Document) { d.Places[0].ZIPs[0] = "０１２３４" }, "ZIP"},
		{"ZIP excluded zero", func(d *usaGeographyV2Document) { d.Places[0].ZIPs[0] = "00000" }, "ZIP"},
		{"ZIP excluded 10000", func(d *usaGeographyV2Document) { d.Places[0].ZIPs[0] = "10000" }, "ZIP"},
		{"ZIP duplicate", func(d *usaGeographyV2Document) { d.Places[0].ZIPs[1] = d.Places[0].ZIPs[0] }, "ZIP"},
		{"ZIP order", func(d *usaGeographyV2Document) {
			d.Places[0].ZIPs[0], d.Places[0].ZIPs[1] = d.Places[0].ZIPs[1], d.Places[0].ZIPs[0]
		}, "ZIP"},
		{"ZIP collision", func(d *usaGeographyV2Document) {
			d.Places[1].ZIPs = []string{d.Places[0].ZIPs[0]}
		}, "contradictory"},
		{"ZIP missing row", func(d *usaGeographyV2Document) { d.Places[0].ZIPs = d.Places[0].ZIPs[1:] }, "coverage"},
		{"unqualified observation", func(d *usaGeographyV2Document) {
			*d.Places[0].MunicipalPopulation2025++
			d.Places[0].AllocationScore++
		}, "owner-adopted asset"},
		{"unqualified subthreshold municipality", func(d *usaGeographyV2Document) {
			*d.Places[1].MunicipalPopulation2025 = 1
			d.Places[1].AllocationScore = 1
			d.Places[1].Weight = 23
			d.Places[2].Weight = 737247
		}, "owner-adopted asset"},
		{"unqualified ZIP mapping", func(d *usaGeographyV2Document) { d.Places[0].ZIPs[0] = "36099" }, "owner-adopted asset"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			document := readUSAGeographyV2Document(t)
			test.change(&document)
			_, err := prepareUSAGeographyV2Candidate(&document)
			if err != nil {
				if !strings.Contains(err.Error(), test.cause) {
					t.Fatalf("expected rejection containing %q, got %v", test.cause, err)
				}
				return
			}
			t.Fatal("expected rejected v2 geography, got nil")
		})
	}
	_, err := prepareUSAGeographyV2Candidate(nil)
	if err != nil {
		return
	}
	t.Fatal("expected rejected nil v2 document, got nil")

}

// TestUSAGeographyV2MunicipalEvidence distinguishes observed and editorial data.
func TestUSAGeographyV2MunicipalEvidence(t *testing.T) {

	document := readUSAGeographyV2Document(t)
	data, err := os.ReadFile(usaGeographyV2Directory + "assets/evidence/locality-weights-v1-v2.csv")
	if err != nil {
		t.Fatalf("expected municipal evidence, got %v", err)
	}
	rows, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		t.Fatalf("expected decoded municipal evidence, got %v", err)
	}
	if len(rows) != 128 || len(rows[0]) != 14 {
		t.Fatalf("expected 127 evidence rows with 14 columns, got %d/%d", len(rows)-1, len(rows[0]))
	}
	var policy struct {
		Capitals      map[string]string `json:"capitals"`
		PostalAliases map[string]string `json:"postalAliases"`
	}
	data, err = os.ReadFile(usaGeographyV2Directory + "locality-policy.json")
	if err != nil {
		t.Fatalf("expected adopted policy, got %v", err)
	}
	err = json.Unmarshal(data, &policy)
	if err != nil {
		t.Fatalf("expected decoded policy, got %v", err)
	}
	if len(policy.Capitals) != 51 || len(policy.PostalAliases) != 13 {
		t.Fatalf("expected 51 capitals and 13 explicit correspondences, got %d/%d",
			len(policy.Capitals), len(policy.PostalAliases))
	}
	fallbacks := map[string]string{"HI": "Honolulu", "ME": "Augusta", "SD": "Pierre", "VT": "Montpelier"}
	observed, editorial := 0, 0
	usedAliases := map[string]bool{}
	capitals := map[string]bool{}
	for i, place := range document.Places {

		row := rows[i+1]
		if row[0] != place.ID || row[1] != place.StateUSPS || row[2] != place.Name || row[4] != place.ScoreKind ||
			row[6] != strconv.FormatUint(place.AllocationScore, 10) || row[11] != strconv.FormatUint(place.Weight, 10) {
			t.Fatalf("expected matching locality evidence, got %v", row)
		}
		if policy.Capitals[place.StateUSPS] == place.Name {
			capitals[place.StateUSPS] = true
		}
		if place.MunicipalPopulation2025 == nil {
			if fallbacks[place.StateUSPS] != place.Name || place.AllocationScore != 20000 ||
				place.ScoreKind != "editorial-source-coverage-floor" || row[5] != "" ||
				row[7] != "" || row[8] != "" || row[9] != "" {
				t.Fatalf("expected explicit editorial score without municipal evidence, got %v", row)
			}
			editorial++
			continue
		}
		if place.ScoreKind != "census-municipal-resident-population" ||
			*place.MunicipalPopulation2025 != place.AllocationScore ||
			row[5] != strconv.FormatUint(*place.MunicipalPopulation2025, 10) || row[7] == "" || row[9] == "" ||
			strings.Contains(row[8], " CDP,") ||
			*place.MunicipalPopulation2025 < 250000 && policy.Capitals[place.StateUSPS] != place.Name {
			t.Fatalf("expected qualified municipal population and inclusion, got %v", row)
		}
		if alias, exists := policy.PostalAliases[row[8]]; exists {
			if alias != place.Name {
				t.Fatalf("expected explicit correspondence %s, got %s", alias, place.Name)
			}
			usedAliases[row[8]] = true
		}
		observed++

	}
	if observed != 123 || editorial != 4 || len(usedAliases) != 13 || len(capitals) != 51 {
		t.Fatalf("expected 123 observations, four editorial scores, 13 correspondences and 51 capitals, got %d/%d/%d/%d",
			observed, editorial, len(usedAliases), len(capitals))
	}

}

// TestUSAGeographyV2SingleSelection checks deterministic locality composition.
func TestUSAGeographyV2SingleSelection(t *testing.T) {

	document := readUSAGeographyV2Document(t)
	data, err := prepareUSAGeographyV2Candidate(&document)
	if err != nil {
		t.Fatalf("expected admitted v2 geography, got %v", err)
	}
	rng, replay, expected := splitMix64{state: 2026}, splitMix64{state: 2026}, splitMix64{state: 2026}
	for range 2000 {
		position, err := expected.WeightedIndex(data.weights)
		if err != nil {
			t.Fatalf("expected proportional locality draw, got %v", err)
		}
		first, err := data.selectLocation(&rng)
		if err != nil {
			t.Fatalf("expected selected locality, got %v", err)
		}
		second, err := data.selectLocation(&replay)
		if err != nil {
			t.Fatalf("expected replayed locality, got %v", err)
		}
		place := document.Places[position]
		if first.id != place.ID || first.id != second.id || first.name != place.Name ||
			first.state.USPS != place.StateUSPS || first.state.FIPS != place.StateFIPS ||
			first.state.Division != place.Division || first.state.DivisionOrdinal != place.DivisionOrdinal ||
			!slices.Equal(first.zips, place.ZIPs) || rng.state != expected.state || rng.state != replay.state {
			t.Fatal("expected city, state, ZIP set and L2 Division from one deterministic draw, got independent geography")
		}
	}

}

// TestUSAGeographyV2V1Preservation checks unchanged v1 identity and associations.
func TestUSAGeographyV2V1Preservation(t *testing.T) {

	v1Document, v2Document := readUSAGeographyDocument(t), readUSAGeographyV2Document(t)
	v1, err := prepareUSAGeographyCandidate(&v1Document)
	if err != nil {
		t.Fatalf("expected original v1 admission, got %v", err)
	}
	v2, err := prepareUSAGeographyV2Candidate(&v2Document)
	if err != nil {
		t.Fatalf("expected v2 admission, got %v", err)
	}
	if fmt.Sprintf("%x", v1.identity) != "ba1db6bb788430d078025429b8a18830013aeda1b7cee872ac05e6aeef0fc220" ||
		v1.identity == v2.identity || v1.total != v2.total || v1.divisionMasses != v2.divisionMasses ||
		len(v1.locations) != 78 || len(v2.locations)-len(v1.locations) != 49 {
		t.Fatal("expected unchanged v1 identity and masses with distinct expanded v2, got incompatible versions")
	}
	for _, old := range v1.locations {
		location, err := v2.resolve(old.id)
		if err != nil {
			t.Fatalf("expected retained v1 locality, got %v", err)
		}
		if location.name != old.name || location.state != old.state || !slices.Equal(location.zips, old.zips) {
			t.Fatalf("expected preserved v1 identity and ZIPs for %s, got %v", old.id, location)
		}
	}
	if usaLocalityID("17", "Springfield") == usaLocalityID("25", "Springfield") {
		t.Fatal("expected distinct state-qualified homonyms, got identical IDs")
	}

}

// TestUSAGeographyV2VersionBoundary rejects both directions of version spoofing.
func TestUSAGeographyV2VersionBoundary(t *testing.T) {

	for _, test := range []struct {
		name, path, oldSchema, newSchema string
		decodeV2                         bool
	}{
		{"v1 as v2", usaCandidateDirectory + "geography.json", "", "", true},
		{"relabeled v1 as v2", usaCandidateDirectory + "geography.json",
			"sw-us-geography-candidate-v1", "sw-us-geography-candidate-v2", true},
		{"v2 as v1", usaGeographyV2Directory + "assets/geography.json", "", "", false},
		{"relabeled v2 as v1", usaGeographyV2Directory + "assets/geography.json",
			"sw-us-geography-candidate-v2", "sw-us-geography-candidate-v1", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			data, err := os.ReadFile(test.path)
			if err != nil {
				t.Fatalf("expected candidate document, got %v", err)
			}
			if test.oldSchema != "" {
				data = bytes.Replace(data, []byte(test.oldSchema), []byte(test.newSchema), 1)
			}
			if test.decodeV2 {
				_, err = decodeUSAGeographyV2Candidate(t.Context(), bytes.NewReader(data))
			}
			if !test.decodeV2 {
				_, err = decodeUSAGeographyCandidate(t.Context(), bytes.NewReader(data))
			}
			if err != nil {
				return
			}
			t.Fatal("expected incompatible version rejected, got nil")
		})
	}

}

func readUSAGeographyV2Document(t *testing.T) usaGeographyV2Document {

	t.Helper()
	data, err := os.ReadFile(usaGeographyV2Directory + "assets/geography.json")
	if err != nil {
		t.Fatalf("expected v2 geography asset, got %v", err)
	}
	var document usaGeographyV2Document
	err = decodeInternationalJSON(t.Context(), bytes.NewReader(data), &document)
	if err != nil {
		t.Fatalf("expected decoded v2 geography, got %v", err)
	}

	return document
}
