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
)

const usaAgeSourceSHA256 = "23dd05d6bc3bd7b76c69df1552f8499dd5ae6b9be9a65719318bab4286571d0b"

// usaAgeCandidate owns only qualified USA age masses, with no World entry.
type usaAgeCandidate struct {
	ages     []internationalAgeWeight
	total    uint64
	identity [sha256.Size]byte
}

type usaAgeDocument struct {
	Schema       string                   `json:"schema"`
	Status       string                   `json:"status"`
	Country      string                   `json:"country"`
	AnchorDate   string                   `json:"anchorDate"`
	Vintage      string                   `json:"vintage"`
	Universe     string                   `json:"universe"`
	Sex          string                   `json:"sex"`
	Measure      string                   `json:"measure"`
	SourceFile   string                   `json:"sourceFile"`
	SourceSHA256 string                   `json:"sourceSHA256"`
	Ages         []internationalAgeWeight `json:"ages" max:"55"`
	Total        uint64                   `json:"total"`
}

// decodeUSAAgeCandidate reads a closed candidate asset; callers must arrange
// interruption of blocking reads, as for the international document decoder.
func decodeUSAAgeCandidate(ctx context.Context, reader io.Reader) (*usaAgeCandidate, error) {

	var document usaAgeDocument
	err := decodeInternationalJSON(ctx, reader, &document)
	if err != nil {
		return nil, err
	}

	return prepareUSAAgeCandidate(&document)
}

func prepareUSAAgeCandidate(d *usaAgeDocument) (*usaAgeCandidate, error) {

	// Validate the exact qualified Census period, universe and quantity.
	if d == nil || d.Schema != "sw-us-age-candidate-v1" || d.Status != "CANDIDATE" || d.Country != "US" ||
		d.AnchorDate != usaSocioeconomicAnchorDate || d.Vintage != "2025" ||
		d.Universe != "civilian-noninstitutionalized" || d.Sex != "both" ||
		d.Measure != "short-term-projected-population" || d.SourceFile != "nc-est2025-alldata-n-file13.csv" ||
		d.SourceSHA256 != usaAgeSourceSHA256 {
		return nil, fmt.Errorf("unsupported USA age source, period or universe")
	}
	if len(d.Ages) != 55 {
		return nil, fmt.Errorf("expected 55 USA ages")
	}

	var total uint64
	for i, age := range d.Ages {
		if age.Age != uint32(i+21) || age.Weight == 0 || age.Weight > math.MaxUint64-total {
			return nil, fmt.Errorf("invalid USA age or weight at position %d", i)
		}
		total += age.Weight
	}
	if d.Total != total {
		return nil, fmt.Errorf("USA age total does not reconcile")
	}
	canonical, err := internationalCanonical(d)
	if err != nil {
		return nil, err
	}

	return &usaAgeCandidate{ages: slices.Clone(d.Ages), total: total, identity: sha256.Sum256(canonical)}, nil
}
