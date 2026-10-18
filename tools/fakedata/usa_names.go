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
)

const (
	usaNamesMass            = uint64(1<<32 - 1)
	usaNamesPolicySHA256    = "e4648c8b52fb7f7f201acdcd4da110565ea675b3557ced44cc3fc70f298e1324"
	usaSurnameAssetSHA256   = "bde17c82f426aec71bc65dd634ac394a5cc854c985ff128b2e79a7a4d0b8998e"
	usaSurnameSourceSHA256  = "2e773c7edd934bb340b09be46e5d991b74a65cd63f547c94f80b6e233db462b3"
	usaSurnameSelectionSize = 5000
)

// usaNameRow owns a scalar copy of an admitted international NameEntry.
type usaNameRow struct {
	display   string
	asciiAtom string
	weight    uint64
}

// usaSurnameCandidate is internal and candidate-only; it creates no USA World.
type usaSurnameCandidate struct {
	entries  []usaNameRow
	weights  []uint64
	total    uint64
	identity [sha256.Size]byte
}

func (d *usaSurnameCandidate) selectSurname(rng *splitMix64) (usaNameRow, error) {

	if d == nil || rng == nil || len(d.entries) != usaSurnameSelectionSize ||
		len(d.weights) != len(d.entries) || d.total != usaNamesMass {
		return usaNameRow{}, fmt.Errorf("uninitialized USA surname candidate or random stream")
	}

	position, err := rng.WeightedIndex(d.weights)
	if err != nil {
		return usaNameRow{}, err
	}

	return d.entries[position], nil
}

// usaSurnameDocument reuses NameEntry with a separate USA collection bound;
// the international Italian consumer and its 600-entry limit are unchanged.
type usaSurnameDocument struct {
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
	Entries            []internationalName `json:"entries" max:"5000"`
}

// decodeUSASurnameCandidate reads the closed partial asset; callers arrange
// interruption of blocking reads, as for the international JSON decoder.
func decodeUSASurnameCandidate(ctx context.Context, reader io.Reader) (*usaSurnameCandidate, error) {
	var document usaSurnameDocument
	err := decodeInternationalJSON(ctx, reader, &document)
	if err != nil {
		return nil, err
	}
	return prepareUSASurnameCandidate(&document)
}

// prepareUSASurnameCandidate admits only the exact qualified candidate asset.
// Its technical admission does not adopt the owner's 5,000-surname proposal.
func prepareUSASurnameCandidate(d *usaSurnameDocument) (*usaSurnameCandidate, error) {

	// Validate the standard source, editorial policy and exact coverage metadata.
	if d == nil || d.Schema != "sw-us-surnames-candidate-v1" || d.Status != "CANDIDATE" ||
		d.Approval != "NOT OWNER-APPROVED" || d.Country != "US" || d.AnchorDate != usaSocioeconomicAnchorDate ||
		d.Vintage != "2020" || d.SourceVariant != "standard-complete" ||
		d.SourceFile != "Names2020_LastNames_RaceHispanic.xlsx" || d.SourceSHA256 != usaSurnameSourceSHA256 ||
		d.FrequencyColumn != "C:FREQUENCY (COUNT)" || d.PolicySHA256 != usaNamesPolicySHA256 ||
		d.WeightKind != "editorial-top5000-census-count-hamilton-positive-reserving-v1" ||
		d.SelectionLimit != usaSurnameSelectionSize || d.SourceValidNames != 156621 ||
		d.SourceNamedMass != 262640594 || d.SourceOtherMass != 36257637 || d.SelectedSourceMass != 174828610 ||
		d.OmittedNamedMass != 87811984 || d.Total != usaNamesMass {
		return nil, fmt.Errorf("unsupported USA surname candidate metadata or provenance")
	}
	err := validateUSANominalEntries(d.Entries, usaSurnameSelectionSize, usaNamesMass)
	if err != nil {
		return nil, err
	}

	canonical, err := internationalCanonical(d)
	if err != nil {
		return nil, err
	}
	identity := sha256.Sum256(canonical)
	if fmt.Sprintf("%x", identity) != usaSurnameAssetSHA256 {
		return nil, fmt.Errorf("USA surname candidate digest mismatch")
	}

	prepared := &usaSurnameCandidate{
		entries: make([]usaNameRow, len(d.Entries)), weights: make([]uint64, len(d.Entries)),
		total: d.Total, identity: identity,
	}
	for i, entry := range d.Entries {
		prepared.entries[i] = usaNameRow{display: entry.Display, asciiAtom: *entry.ASCIIAtom, weight: entry.Weight}
		prepared.weights[i] = entry.Weight
	}

	return prepared, nil
}

// validateUSANominalEntries checks bounded NameEntry values without rewriting
// display/atoms; the verified Unicode-16 repertoire covers all observed rows.
func validateUSANominalEntries(entries []internationalName, count int, expectedTotal uint64) error {

	if count <= 0 || count > usaSurnameSelectionSize || len(entries) != count || expectedTotal == 0 {
		return fmt.Errorf("invalid USA nominal cardinality or total")
	}
	previous := ""
	var total uint64
	for i, entry := range entries {
		if len(entry.Display) > 80 {
			return fmt.Errorf("USA nominal display exceeds byte limit at position %d", i)
		}
		key, err := italianDisplayKey(entry.Display)
		if err != nil {
			return fmt.Errorf("USA nominal display at position %d: %w", i, err)
		}
		if key <= previous || entry.ASCIIAtom == nil || len(*entry.ASCIIAtom) > 80 ||
			!internationalASCIIAtom(*entry.ASCIIAtom) || entry.Weight == 0 || entry.Weight > math.MaxUint64-total {
			return fmt.Errorf("invalid USA nominal key, atom or weight at position %d", i)
		}
		previous = key
		total += entry.Weight
	}
	if total != expectedTotal {
		return fmt.Errorf("USA nominal total does not reconcile")
	}

	return nil
}
