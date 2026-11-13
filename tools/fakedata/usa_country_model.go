// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

type usaCountryData struct {
	baseDocument  usaBaseDocument
	base          *usaBaseCandidate
	phones        *usaNPACandidate
	socioeconomic *usaSocioeconomicData
}

// usaCountrySourcePin binds acquisition/provenance, independently of M_all_US.
type usaCountrySourcePin struct {
	Path   string `json:"path"`
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
}
