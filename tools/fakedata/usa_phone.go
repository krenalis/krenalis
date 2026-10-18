// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"time"
)

const (
	usaNPAAssetSHA256   = "29fe0902e5423be1feb998cf2b6aed7fd5dfd9006f44a64993ea3afa8239ae1f"
	usaNPASourceSHA256  = "73694f25eb3f71a6b562e4cc4daee948dc12f582203158663eabe1a7119482f0"
	usaNPAHistorySHA256 = "23c6be02b9f6f2bccb41a5f5660b5ec2ffe8144f7f21b017fedba94745e09cb8"
)

// usaNPACandidate owns the temporally qualified, numerically ordered rule.
type usaNPACandidate struct {
	npas     []string
	identity [sha256.Size]byte
}

// phone cycles evenly over the reserved fiction interval; the operational
// coordinate domain is checked before subtraction or modulo, without RNG.
func (d *usaNPACandidate) phone(index PersonIndex) (string, error) {

	if index == 0 {
		return "", ErrInvalidIndex
	}
	if index > Layer1MaxPersonIndex {
		return "", ErrLayer1IndexOutOfRange
	}
	if d == nil || len(d.npas) != 368 || d.identity == ([sha256.Size]byte{}) {
		return "", fmt.Errorf("qualified USA NPA candidate is required")
	}
	count := uint64(len(d.npas))
	r := (uint64(index) - 1) % (100 * count)

	return fmt.Sprintf("+1%s555%04d", d.npas[r%count], 100+r/count), nil
}

type usaNPADocument struct {
	Schema                   string        `json:"schema"`
	Status                   string        `json:"status"`
	Country                  string        `json:"country"`
	AnchorDate               string        `json:"anchorDate"`
	SourceDate               string        `json:"sourceDate"`
	SourceSHA256             string        `json:"sourceSHA256"`
	HistoricalAsOf           string        `json:"historicalAsOf"`
	HistoricalEvidenceSHA256 string        `json:"historicalEvidenceSHA256"`
	ExcludedByPolicy         []string      `json:"excludedByPolicy" max:"1"`
	Entries                  []usaNPAEntry `json:"entries" max:"400"`
}

type usaNPAEntry struct {
	NPA           string `json:"npa"`
	StateUSPS     string `json:"stateUSPS"`
	InServiceDate string `json:"inServiceDate"`
}

// decodeUSANPACandidate admits the exact temporally qualified NPA list;
// callers arrange interruption of any blocking read.
func decodeUSANPACandidate(ctx context.Context, reader io.Reader) (*usaNPACandidate, error) {

	var d usaNPADocument
	err := decodeInternationalJSON(ctx, reader, &d)
	if err != nil {
		return nil, err
	}
	if d.Schema != "sw-us-npa-candidate-v1" || d.Status != "CANDIDATE" || d.Country != "US" ||
		d.AnchorDate != "2026-01-01" || d.SourceDate != "2026-10-16" || d.SourceSHA256 != usaNPASourceSHA256 ||
		d.HistoricalAsOf != "2025-12-31" || d.HistoricalEvidenceSHA256 != usaNPAHistorySHA256 ||
		len(d.ExcludedByPolicy) != 1 || d.ExcludedByPolicy[0] != "679" || len(d.Entries) != 368 {
		return nil, fmt.Errorf("unsupported USA NPA source, period or model policy")
	}
	prepared := &usaNPACandidate{}
	previous := ""
	for _, entry := range d.Entries {

		if !validFixedDigits(entry.NPA, 3) || entry.NPA[0] < '2' || entry.NPA <= previous || entry.NPA == "679" {
			return nil, fmt.Errorf("invalid or unordered USA geographic NPA")
		}
		found := false
		for _, state := range usaStates() {
			if state.USPS == entry.StateUSPS {
				found = true
				break
			}
		}
		date, err := time.Parse(time.DateOnly, entry.InServiceDate)
		if err != nil {
			return nil, fmt.Errorf("invalid USA NPA service date")
		}
		if !found || date.Year() < 1947 || entry.InServiceDate > d.AnchorDate {
			return nil, fmt.Errorf("USA NPA outside admitted jurisdiction or service period")
		}
		prepared.npas = append(prepared.npas, entry.NPA)
		previous = entry.NPA

	}
	canonical, err := internationalCanonical(d)
	if err != nil {
		return nil, err
	}
	prepared.identity = sha256.Sum256(canonical)
	if fmt.Sprintf("%x", prepared.identity) != usaNPAAssetSHA256 {
		return nil, fmt.Errorf("USA NPA candidate digest mismatch")
	}

	return prepared, nil
}
