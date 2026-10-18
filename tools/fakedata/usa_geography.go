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
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	usaGeographyWeightKind   = "editorial-state-resident-population-equal-locality-v1"
	usaPostalSourceSHA256    = "ad61fa51feebedac67abdfd835c41dfad781532d54d2031220caa9d70983488f"
	usaStatePopulationSHA256 = "49975e1b89ccd2b145e634d9e9e517ecd44c334fda8a362d0b31d4b1e1159134"
)

// usaGeographyCandidate owns admitted localities and their effective masses.
// It deliberately has no public World, registry or approval entry point.
type usaGeographyCandidate struct {
	locations      []usaLocation
	weights        []uint64
	byID           map[string]int
	divisionMasses [9]uint64
	total          uint64
	identity       [sha256.Size]byte
}

func (d *usaGeographyCandidate) resolve(id string) (usaLocation, error) {

	if d == nil || len(d.locations) == 0 {
		return usaLocation{}, fmt.Errorf("uninitialized USA geography candidate")
	}
	position, ok := d.byID[id]
	if !ok {
		return usaLocation{}, fmt.Errorf("unknown USA locality")
	}
	location := d.locations[position]
	location.zips = slices.Clone(location.zips)

	return location, nil
}

// selectLocation makes one proportional locality draw; its state supplies the
// L2 Division. Future address/Person composition must retain this same value.
// The caller owns the deterministic RNG; no Italian mixture stream is used.
func (d *usaGeographyCandidate) selectLocation(rng *splitMix64) (usaLocation, error) {

	if d == nil || len(d.locations) == 0 || rng == nil {
		return usaLocation{}, fmt.Errorf("initialized USA geography and RNG are required")
	}
	position, err := rng.WeightedIndex(d.weights)
	if err != nil {
		return usaLocation{}, err
	}

	return d.resolve(d.locations[position].id)
}

type usaGeographyDocument struct {
	Schema                      string              `json:"schema"`
	Status                      string              `json:"status"`
	Country                     string              `json:"country"`
	AnchorDate                  string              `json:"anchorDate"`
	TerritorySystem             string              `json:"territorySystem"`
	WeightKind                  string              `json:"weightKind"`
	Approval                    string              `json:"approval"`
	PostalEvidenceDate          string              `json:"postalEvidenceDate"`
	PostalSourceSHA256          string              `json:"postalSourceSHA256"`
	StatePopulationDate         string              `json:"statePopulationDate"`
	StatePopulationSourceSHA256 string              `json:"statePopulationSourceSHA256"`
	States                      []usaGeographyState `json:"states" max:"51"`
	Places                      []usaPlaceCandidate `json:"places" max:"512"`
	DivisionMasses              []uint64            `json:"divisionMasses" max:"9"`
	Total                       uint64              `json:"total"`
}

type usaGeographyState struct {
	State                  usaState `json:"state"`
	ResidentPopulation2025 uint64   `json:"residentPopulation2025"`
}

// usaLocation retains postal locality identity independently of municipality
// boundaries; its weight is editorial and must never be called population.
type usaLocation struct {
	id     string
	name   string
	state  usaState
	weight uint64
	zips   []string
}

type usaPlaceCandidate struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	StateUSPS       string   `json:"stateUSPS"`
	StateFIPS       string   `json:"stateFIPS"`
	Division        string   `json:"division"`
	DivisionOrdinal uint32   `json:"divisionOrdinal"`
	Weight          uint64   `json:"weight"`
	ZIPs            []string `json:"zips" max:"1024"`
}

// decodeUSAGeographyCandidate reads the bounded closed candidate format, not
// base.json; callers must arrange interruption of any blocking reads.
func decodeUSAGeographyCandidate(ctx context.Context, reader io.Reader) (*usaGeographyCandidate, error) {

	var document usaGeographyDocument
	err := decodeInternationalJSON(ctx, reader, &document)
	if err != nil {
		return nil, err
	}

	return prepareUSAGeographyCandidate(&document)
}

// prepareUSAGeographyCandidate validates the proposed editorial recipe and
// computes Division masses from admitted locality weights before any World.
func prepareUSAGeographyCandidate(d *usaGeographyDocument) (*usaGeographyCandidate, error) {

	// Validate the candidate boundary and the two distinct source periods.
	if d == nil || d.Schema != "sw-us-geography-candidate-v1" || d.Status != "CANDIDATE" || d.Country != "US" ||
		d.AnchorDate != usaSocioeconomicAnchorDate || d.WeightKind != usaGeographyWeightKind ||
		d.Approval != "owner-decision-required" || d.PostalEvidenceDate != "2015-06-15" ||
		d.PostalSourceSHA256 != usaPostalSourceSHA256 || d.StatePopulationDate != "2025-07-01" ||
		d.StatePopulationSourceSHA256 != usaStatePopulationSHA256 {
		return nil, fmt.Errorf("unsupported USA geography candidate metadata")
	}
	taxonomy := usaInternationalDimensions().Territory
	if d.TerritorySystem != taxonomy.SemanticID || len(d.DivisionMasses) != 9 || len(d.States) != 51 ||
		len(d.Places) < 51 || len(d.Places) > 512 {
		return nil, fmt.Errorf("invalid USA geography coverage or territory binding")
	}

	// Validate exact state identity and the observed reference masses.
	states := usaStates()
	statePopulation := map[string]uint64{}
	var stateTotal uint64
	for i, entry := range d.States {
		state := entry.State
		if state != states[i] || state.DivisionOrdinal >= 9 ||
			state.Division != taxonomy.Categories[state.DivisionOrdinal].Code ||
			entry.ResidentPopulation2025 == 0 || entry.ResidentPopulation2025 > math.MaxUint64-stateTotal {
			return nil, fmt.Errorf("invalid USA state identity or mass at position %d", i)
		}
		statePopulation[state.USPS] = entry.ResidentPopulation2025
		stateTotal += entry.ResidentPopulation2025
	}

	prepared := &usaGeographyCandidate{byID: map[string]int{}}
	stateCounts, stateMasses := map[string]uint64{}, map[string]uint64{}
	zipOwners := map[string]string{}
	previous := ""
	for i, place := range d.Places {

		// Resolve state first; the locality may not choose a different Division.
		state, err := resolveUSAState(place.StateUSPS, place.StateFIPS)
		if err != nil {
			return nil, err
		}
		key := state.FIPS + "\x00" + place.Name
		if !internationalText(place.Name) || !utf8.ValidString(place.Name) || len(place.Name) > 180 ||
			strings.TrimSpace(place.Name) != place.Name ||
			place.ID != usaLocalityID(state.FIPS, place.Name) || key <= previous ||
			place.Division != state.Division || place.DivisionOrdinal != state.DivisionOrdinal {
			return nil, fmt.Errorf("invalid USA locality identity, order or Division at position %d", i)
		}
		if _, exists := prepared.byID[place.ID]; exists {
			return nil, fmt.Errorf("duplicate USA locality ID")
		}
		if place.Weight == 0 || place.Weight > math.MaxUint64-prepared.total {
			return nil, fmt.Errorf("nonpositive or overflowing USA locality mass")
		}

		if len(place.ZIPs) == 0 || len(place.ZIPs) > 1024 {
			return nil, fmt.Errorf("missing or excessive USA postal associations")
		}
		for j, zip := range place.ZIPs {
			if !validFixedDigits(zip, 5) || zip == "00000" || zip == "10000" || j > 0 && zip <= place.ZIPs[j-1] {
				return nil, fmt.Errorf("invalid or unordered USA ZIP")
			}
			if _, exists := zipOwners[zip]; exists {
				return nil, fmt.Errorf("contradictory USA postal association")
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
		stateCounts[state.USPS]++
		stateMasses[state.USPS] += place.Weight
		previous = key

	}

	// Reconcile actual locality masses, then enforce the stated candidate recipe.
	if len(stateCounts) != 51 || prepared.total != d.Total || prepared.total != stateTotal {
		return nil, fmt.Errorf("USA state coverage or total mass does not reconcile")
	}
	for ordinal, mass := range prepared.divisionMasses {
		if mass == 0 || mass != d.DivisionMasses[ordinal] {
			return nil, fmt.Errorf("USA Division mass does not reconcile")
		}
	}
	statePosition := map[string]uint64{}
	for _, place := range d.Places {
		population, count := statePopulation[place.StateUSPS], stateCounts[place.StateUSPS]
		expected := population / count
		if statePosition[place.StateUSPS] < population%count {
			expected++
		}
		if place.Weight != expected || stateMasses[place.StateUSPS] != population {
			return nil, fmt.Errorf("USA locality weight differs from the declared editorial recipe")
		}
		statePosition[place.StateUSPS]++
	}
	canonical, err := internationalCanonical(d)
	if err != nil {
		return nil, err
	}
	prepared.identity = sha256.Sum256(canonical)

	return prepared, nil
}

func usaLocalityID(fips, name string) string {
	digest := sha256.Sum256([]byte(name))
	return fmt.Sprintf("us-postal-%s-%x", fips, digest[:8])
}
