// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"context"
	"fmt"
	"io"
)

// ItalianReleasePath is the fixed descriptor of the Italian JSON v1 policy.
const ItalianReleasePath = "synthetic-world/countries/IT/it-2026a-json1/release.json"

// CountryReleaseData is an immutable sw-country-release-v1 descriptor.
// Its zero value is invalid; references contain logical, not physical digests.
type CountryReleaseData struct {
	document    internationalRelease
	initialized bool
}

// DecodeCountryReleaseData validates a closed international v1 descriptor.
// It returns context errors between reads and tools/json SyntaxError for
// malformed tokens. Callers must interrupt blocking reads themselves.
func DecodeCountryReleaseData(ctx context.Context, reader io.Reader) (*CountryReleaseData, error) {
	return decodeCountryReleaseData(ctx, reader)
}

// CanonicalJSON returns the descriptor's C1 serialization.
func (d *CountryReleaseData) CanonicalJSON() ([]byte, error) {
	if d == nil || !d.initialized {
		return nil, fmt.Errorf("uninitialized country release document")
	}
	return internationalCanonical(d.document)
}

// LogicalDigest returns the descriptor's KFD1/C1 digest.
func (d *CountryReleaseData) LogicalDigest() ([32]byte, error) {
	if d == nil || !d.initialized {
		return [32]byte{}, fmt.Errorf("uninitialized country release document")
	}
	return internationalDigest("fakedata/country-release/json-v1", d.document)
}

type internationalRelease struct {
	Schema        string                      `json:"schema"`
	Release       string                      `json:"release"`
	Country       string                      `json:"country"`
	Base          internationalAssetReference `json:"base"`
	Socioeconomic internationalAssetReference `json:"socioeconomic"`
	Photos        internationalAssetReference `json:"photos"`
}

type internationalAssetReference struct {
	Path          string `json:"path"`
	LogicalSHA256 string `json:"logicalSha256"`
}

func decodeCountryReleaseData(ctx context.Context, reader io.Reader) (*CountryReleaseData, error) {

	var document internationalRelease
	err := decodeInternationalJSON(ctx, reader, &document)
	if err != nil {
		return nil, err
	}
	if document.Schema != "sw-country-release-v1" || !internationalToken(document.Release) ||
		len(document.Country) != 2 || document.Country[0] < 'A' || document.Country[0] > 'Z' ||
		document.Country[1] < 'A' || document.Country[1] > 'Z' {
		return nil, fmt.Errorf("invalid country release metadata")
	}
	for _, reference := range []internationalAssetReference{document.Base, document.Socioeconomic, document.Photos} {
		if !internationalAssetPath(reference.Path) || !internationalSHA256(reference.LogicalSHA256) {
			return nil, fmt.Errorf("invalid country release reference")
		}
	}

	return &CountryReleaseData{document: document, initialized: true}, nil
}

func internationalSHA256(value string) bool {
	return len(value) == 64 && lowerHex(value)
}
