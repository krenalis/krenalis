// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"math/bits"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	usaGeographyV2WeightKind     = "editorial-state-resident-population-municipal-proportional-v2"
	usaGeographyV2PolicySHA256   = "73b13806767a5dcecb847c2e7597dbc90b375d3692d2adda24192b2370e74a14"
	usaGeographyV2AssetSHA256    = "5d48985af972b1a629e728925d54ebcb50fae3a2164eabc29017a6fab374bd15"
	usaMunicipalPopulationSHA256 = "c696964db3fcbc42db6fc5ef1285c76476fd84024964f5caab6d7f3ec7bf1b9a"
	usaGeographyV2Total          = 341784857
)

// usaGeographyV2Document preserves the source's CANDIDATE metadata; the owner's
// separate Demo Plausible adoption is recorded in the evidence register.
type usaGeographyV2Document struct {
	Schema                          string                `json:"schema"`
	Status                          string                `json:"status"`
	Country                         string                `json:"country"`
	AnchorDate                      string                `json:"anchorDate"`
	TerritorySystem                 string                `json:"territorySystem"`
	WeightKind                      string                `json:"weightKind"`
	Approval                        string                `json:"approval"`
	PolicySHA256                    string                `json:"policySHA256"`
	PostalEvidenceDate              string                `json:"postalEvidenceDate"`
	PostalSourceSHA256              string                `json:"postalSourceSHA256"`
	StatePopulationDate             string                `json:"statePopulationDate"`
	StatePopulationSourceSHA256     string                `json:"statePopulationSourceSHA256"`
	MunicipalPopulationDate         string                `json:"municipalPopulationDate"`
	MunicipalPopulationSourceSHA256 string                `json:"municipalPopulationSourceSHA256"`
	States                          []usaGeographyState   `json:"states" max:"51"`
	Places                          []usaPlaceV2Candidate `json:"places" max:"127"`
	DivisionMasses                  []uint64              `json:"divisionMasses" max:"9"`
	Total                           uint64                `json:"total"`
}

type usaPlaceV2Candidate struct {
	ID                      string   `json:"id"`
	Name                    string   `json:"name"`
	StateUSPS               string   `json:"stateUSPS"`
	StateFIPS               string   `json:"stateFIPS"`
	Division                string   `json:"division"`
	DivisionOrdinal         uint32   `json:"divisionOrdinal"`
	Weight                  uint64   `json:"weight"`
	ZIPs                    []string `json:"zips" max:"1024"`
	MunicipalPopulation2025 *uint64  `json:"municipalPopulation2025" nullable:"true"`
	AllocationScore         uint64   `json:"allocationScore"`
	ScoreKind               string   `json:"scoreKind"`
}

// decodeUSAGeographyV2Candidate admits the closed v2 document through its own
// typed boundary; callers must arrange interruption of blocking reads.
func decodeUSAGeographyV2Candidate(ctx context.Context, reader io.Reader) (*usaGeographyCandidate, error) {

	var document usaGeographyV2Document
	err := decodeInternationalJSON(ctx, reader, &document)
	if err != nil {
		return nil, err
	}

	return prepareUSAGeographyV2Candidate(&document)
}

// prepareUSAGeographyV2Candidate checks Hamilton allocation and the exact
// owner-adopted asset before reusing the single-locality selector from v1.
func prepareUSAGeographyV2Candidate(d *usaGeographyV2Document) (*usaGeographyCandidate, error) {

	// Validate the identified policy, source periods and candidate metadata.
	if d == nil || d.Schema != "sw-us-geography-candidate-v2" || d.Status != "CANDIDATE" || d.Country != "US" ||
		d.AnchorDate != usaSocioeconomicAnchorDate || d.WeightKind != usaGeographyV2WeightKind ||
		d.Approval != "NOT OWNER-APPROVED" || d.PolicySHA256 != usaGeographyV2PolicySHA256 ||
		d.PostalEvidenceDate != "2015-06-15" || d.PostalSourceSHA256 != usaPostalSourceSHA256 ||
		d.StatePopulationDate != "2025-07-01" || d.StatePopulationSourceSHA256 != usaStatePopulationSHA256 ||
		d.MunicipalPopulationDate != "2025-07-01" || d.MunicipalPopulationSourceSHA256 != usaMunicipalPopulationSHA256 {
		return nil, fmt.Errorf("unsupported USA geography v2 metadata or provenance")
	}
	taxonomy := usaInternationalDimensions().Territory
	if d.TerritorySystem != taxonomy.SemanticID || len(d.DivisionMasses) != 9 || len(d.States) != 51 ||
		len(d.Places) != 127 || d.Total != usaGeographyV2Total {
		return nil, fmt.Errorf("invalid USA geography v2 coverage, total or territory binding")
	}

	// Validate state identity and bound reference masses before arithmetic.
	states := usaStates()
	statePopulation := map[string]uint64{}
	var stateTotal uint64
	for i, entry := range d.States {
		state := entry.State
		if state != states[i] || state.DivisionOrdinal >= 9 ||
			state.Division != taxonomy.Categories[state.DivisionOrdinal].Code ||
			entry.ResidentPopulation2025 == 0 || entry.ResidentPopulation2025 > d.Total-stateTotal {
			return nil, fmt.Errorf("invalid USA v2 state identity or mass at position %d", i)
		}
		statePopulation[state.USPS] = entry.ResidentPopulation2025
		stateTotal += entry.ResidentPopulation2025
	}
	if stateTotal != d.Total {
		return nil, fmt.Errorf("USA v2 state reference total does not reconcile")
	}

	prepared := &usaGeographyCandidate{byID: map[string]int{}}
	statePlaces := map[string][]int{}
	stateMasses, stateScores := map[string]uint64{}, map[string]uint64{}
	zipOwners := map[string]string{}
	previous := ""
	fallbackCount := 0
	for i, place := range d.Places {

		// Bind every locality to its state's existing L2 Division.
		state, err := resolveUSAState(place.StateUSPS, place.StateFIPS)
		if err != nil {
			return nil, err
		}
		key := state.FIPS + "\x00" + place.Name
		if !internationalText(place.Name) || !utf8.ValidString(place.Name) || len(place.Name) > 180 ||
			strings.TrimSpace(place.Name) != place.Name || place.ID != usaLocalityID(state.FIPS, place.Name) ||
			key <= previous || place.Division != state.Division || place.DivisionOrdinal != state.DivisionOrdinal {
			return nil, fmt.Errorf("invalid USA v2 locality identity, order or Division at position %d", i)
		}
		if _, exists := prepared.byID[place.ID]; exists {
			return nil, fmt.Errorf("duplicate USA v2 locality ID")
		}

		// Distinguish qualified municipal observations from four editorial scores.
		switch state.USPS + "\x00" + place.Name {
		case "HI\x00Honolulu", "ME\x00Augusta", "SD\x00Pierre", "VT\x00Montpelier":
			if place.MunicipalPopulation2025 != nil || place.AllocationScore != 20000 ||
				place.ScoreKind != "editorial-source-coverage-floor" {
				return nil, fmt.Errorf("invalid USA v2 editorial fallback")
			}
			fallbackCount++
		default:
			population := place.MunicipalPopulation2025
			if population == nil || *population == 0 || *population > statePopulation[state.USPS] ||
				place.AllocationScore != *population || place.ScoreKind != "census-municipal-resident-population" {
				return nil, fmt.Errorf("invalid USA v2 municipal observation or allocation score")
			}
		}
		if place.AllocationScore > math.MaxUint64-stateScores[state.USPS] || place.Weight == 0 ||
			place.Weight > math.MaxUint64-prepared.total {
			return nil, fmt.Errorf("nonpositive or overflowing USA v2 score or mass")
		}

		// Validate bounded, unique historical ZIP associations.
		if len(place.ZIPs) == 0 || len(place.ZIPs) > 1024 {
			return nil, fmt.Errorf("missing or excessive USA v2 postal associations")
		}
		for j, zip := range place.ZIPs {
			if !validFixedDigits(zip, 5) || zip == "00000" || zip == "10000" || j > 0 && zip <= place.ZIPs[j-1] {
				return nil, fmt.Errorf("invalid or unordered USA v2 ZIP")
			}
			if _, exists := zipOwners[zip]; exists {
				return nil, fmt.Errorf("contradictory USA v2 postal association")
			}
			zipOwners[zip] = place.ID
		}

		prepared.byID[place.ID] = i
		prepared.locations = append(prepared.locations, usaLocation{
			id: place.ID, name: place.Name, state: state, weight: place.Weight, zips: slices.Clone(place.ZIPs),
		})
		prepared.weights = append(prepared.weights, place.Weight)
		prepared.total += place.Weight
		prepared.divisionMasses[state.DivisionOrdinal] += place.Weight
		statePlaces[state.USPS] = append(statePlaces[state.USPS], i)
		stateMasses[state.USPS] += place.Weight
		stateScores[state.USPS] += place.AllocationScore
		previous = key

	}

	// Reconcile masses derived from admitted localities, independently of AGE.
	if len(statePlaces) != 51 || len(zipOwners) != 5917 || fallbackCount != 4 || prepared.total != d.Total {
		return nil, fmt.Errorf("USA v2 locality coverage or total mass does not reconcile")
	}
	for ordinal, mass := range prepared.divisionMasses {
		if mass == 0 || mass != d.DivisionMasses[ordinal] {
			return nil, fmt.Errorf("USA v2 Division mass does not reconcile")
		}
	}

	// Recompute Hamilton with 128-bit integer products and exact postal ties.
	floors, remainders := make([]uint64, len(d.Places)), make([]uint64, len(d.Places))
	for _, entry := range d.States {

		usps, mass := entry.State.USPS, entry.ResidentPopulation2025
		indices, scoreTotal := statePlaces[usps], stateScores[usps]
		if stateMasses[usps] != mass || scoreTotal == 0 {
			return nil, fmt.Errorf("USA v2 state mass or scores do not reconcile")
		}
		var allocated uint64
		for _, index := range indices {
			hi, lo := bits.Mul64(mass, d.Places[index].AllocationScore)
			if hi >= scoreTotal {
				return nil, fmt.Errorf("overflowing USA v2 Hamilton quotient")
			}
			floors[index], remainders[index] = bits.Div64(hi, lo, scoreTotal)
			if floors[index] > mass-allocated {
				return nil, fmt.Errorf("overflowing USA v2 Hamilton allocation")
			}
			allocated += floors[index]
		}
		remaining := mass - allocated
		if remaining >= uint64(len(indices)) {
			return nil, fmt.Errorf("invalid USA v2 Hamilton remainder count")
		}
		slices.SortFunc(indices, func(a, b int) int {
			if remainders[a] > remainders[b] {
				return -1
			}
			if remainders[a] < remainders[b] {
				return 1
			}
			return strings.Compare(d.Places[a].Name, d.Places[b].Name)
		})
		for rank, index := range indices {
			expected := floors[index]
			if uint64(rank) < remaining {
				expected++
			}
			if expected == 0 || d.Places[index].Weight != expected {
				return nil, fmt.Errorf("USA v2 locality weight differs from Hamilton allocation")
			}
		}

	}

	// Pin membership, observed values, aliases, state masses and ZIP mappings to
	// the approved asset; a safe but different snapshot needs separate adoption.
	canonical, err := internationalCanonical(d)
	if err != nil {
		return nil, err
	}
	prepared.identity = sha256.Sum256(canonical)
	if fmt.Sprintf("%x", prepared.identity) != usaGeographyV2AssetSHA256 {
		return nil, fmt.Errorf("USA v2 geography differs from the owner-adopted asset")
	}

	return prepared, nil
}
