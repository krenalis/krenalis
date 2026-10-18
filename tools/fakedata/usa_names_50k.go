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
	usaSurname50KAssetSHA256  = "2f44ec588fc1cee43b2a222e36e68ac2222fc16ca8adf0f8c16175d7fe800385"
	usaSurname50KPolicySHA256 = "bf59d184d913f5684b626bf744f39cbf2063446b437392da72c099a95e543583"
)

// usaSurname50KCandidate is the separate owner-adopted Demo Plausible binding.
// It admits only the natural-name catalog adopted with BASE-v2.
type usaSurname50KCandidate struct {
	names    *usaNominalIndex
	identity [sha256.Size]byte
}

func (d *usaSurname50KCandidate) selectSurname(rng *splitMix64) (usaNameRow, error) {
	if d == nil || d.names == nil || len(d.names.entries) != 50000 || d.names.total != usaNamesMass {
		return usaNameRow{}, fmt.Errorf("uninitialized USA 50K surname candidate")
	}
	return d.names.selectName(rng)
}

// usaSurname50KDocument keeps NameEntry and the common decoder unchanged.
type usaSurname50KDocument struct {
	Schema             string              `json:"schema"`
	Status             string              `json:"status"`
	Approval           string              `json:"approval"`
	Country            string              `json:"country"`
	AnchorDate         string              `json:"anchorDate"`
	Vintage            string              `json:"vintage"`
	SourceVariant      string              `json:"sourceVariant"`
	SourceFile         string              `json:"sourceFile"`
	SourceSHA256       string              `json:"sourceSHA256"`
	FrequencyColumn    string              `json:"frequencyColumn"`
	PolicySHA256       string              `json:"policySHA256"`
	WeightKind         string              `json:"weightKind"`
	SelectionLimit     uint32              `json:"selectionLimit"`
	SourceValidNames   uint32              `json:"sourceValidNames"`
	SourceNamedMass    uint64              `json:"sourceNamedMass"`
	SourceOtherMass    uint64              `json:"sourceOtherMass"`
	SelectedSourceMass uint64              `json:"selectedSourceMass"`
	OmittedNamedMass   uint64              `json:"omittedNamedMass"`
	Total              uint64              `json:"total"`
	Entries            []internationalName `json:"entries" max:"50000"`
}

// decodeUSASurname50KCandidate admits only the produced 50K asset; callers
// arrange interruption of blocking reads, as for the common decoder.
func decodeUSASurname50KCandidate(ctx context.Context, reader io.Reader) (*usaSurname50KCandidate, error) {

	var document usaSurname50KDocument
	err := decodeInternationalJSON(ctx, reader, &document)
	if err != nil {
		return nil, err
	}

	return prepareUSASurname50KCandidate(&document)
}

// prepareUSASurname50KCandidate verifies provenance, entries and canonical
// identity before returning the separate immutable sampling binding.
func prepareUSASurname50KCandidate(d *usaSurname50KDocument) (*usaSurname50KCandidate, error) {

	// Validate the adopted policy and the qualified complete source metadata.
	if d == nil || d.Schema != "sw-us-surnames-candidate-v2" || d.Status != "CANDIDATE" ||
		d.Approval != "CANDIDATE: CAPITALIZATION NOT OWNER-APPROVED" ||
		d.Country != "US" || d.AnchorDate != usaSocioeconomicAnchorDate ||
		d.Vintage != "2020" || d.SourceVariant != "standard-complete" ||
		d.SourceFile != "Names2020_LastNames_RaceHispanic.xlsx" || d.SourceSHA256 != usaSurnameSourceSHA256 ||
		d.FrequencyColumn != "C:FREQUENCY (COUNT)" || d.PolicySHA256 != usaSurname50KPolicySHA256 ||
		d.WeightKind != "editorial-top50000-census-count-hamilton-positive-reserving-v2" ||
		d.SelectionLimit != 50000 || d.SourceValidNames != 156621 || d.SourceNamedMass != 262640594 ||
		d.SourceOtherMass != 36257637 || d.SelectedSourceMass != 239992615 || d.OmittedNamedMass != 22647979 ||
		d.Total != usaNamesMass {
		return nil, fmt.Errorf("unsupported USA 50K surname metadata or provenance")
	}
	names, err := newUSANominalIndex(d.Entries, 50000, usaNamesMass)
	if err != nil {
		return nil, err
	}

	canonical, err := internationalCanonical(d)
	if err != nil {
		return nil, err
	}
	identity := sha256.Sum256(canonical)
	if fmt.Sprintf("%x", identity) != usaSurname50KAssetSHA256 {
		return nil, fmt.Errorf("USA 50K surname digest mismatch")
	}

	return &usaSurname50KCandidate{names: names, identity: identity}, nil
}
