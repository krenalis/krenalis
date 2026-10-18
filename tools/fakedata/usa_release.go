// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"

	"golang.org/x/image/webp"
)

const usaReleaseDirectory = "synthetic-world/countries/US/us-2026a/"
const usaReleasePath = usaReleaseDirectory + "release.json"

// loadUSAReleaseData loads the current byte-pinned USA assets from a portable
// directory. It preserves the private M2 contract and does not admit a public
// USD schema or invent a USA SnapshotID. Production evidence stays offline.
func loadUSAReleaseData(ctx context.Context, root string) (*USAWorldData, error) {

	encoded, err := readBoundedPhotoFile(ctx, filepath.Join(root, usaReleasePath), 4096)
	if err != nil {
		return nil, err
	}
	release, err := decodeCountryReleaseData(ctx, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	d := &release.document
	if d.Country != "US" || d.Release != usaBaseRelease ||
		d.Base.Path != usaReleaseDirectory+"base.json" ||
		d.Socioeconomic.Path != usaReleaseDirectory+"socioeconomic.json" ||
		d.Photos.Path != "synthetic-world/photos/v1-json1/catalog.json" {
		return nil, fmt.Errorf("unsupported USA release references")
	}

	// Admit only the exact approved runtime documents, without production evidence.
	pins := []usaCountrySourcePin{
		{"base.json", 3583463, "d8fd7c76142a86917ed4acec065951f8c5e9b44865e1bf6eed7f09f847eb5b80"},
		{"npas.json", 22449, usaNPAAssetSHA256},
		{"socioeconomic.json", usaSourceUSDModelBytes, usaSourceUSDModelSHA256},
	}
	assets := map[string][]byte{}
	for _, pin := range pins {
		data, err := readBoundedPhotoFile(ctx, filepath.Join(root, usaReleaseDirectory, pin.Path), pin.Bytes)
		if err != nil {
			return nil, err
		}
		if len(data) != pin.Bytes || fmt.Sprintf("%x", sha256.Sum256(data)) != pin.SHA256 {
			return nil, fmt.Errorf("altered USA release asset: %s", pin.Path)
		}
		assets[pin.Path] = data
	}
	document, base, err := prepareUSAReleaseBase(ctx, assets["base.json"])
	if err != nil {
		return nil, err
	}
	phones, err := decodeUSANPACandidate(ctx, bytes.NewReader(assets["npas.json"]))
	if err != nil {
		return nil, err
	}
	model, err := decodeUSASocioeconomicData(ctx, bytes.NewReader(assets["socioeconomic.json"]))
	if err != nil {
		return nil, err
	}
	modelDigest, err := model.logicalDigest()
	if err != nil {
		return nil, err
	}
	encoded, err = readBoundedPhotoFile(ctx, filepath.Join(root, d.Photos.Path), 322180)
	if err != nil {
		return nil, err
	}
	if len(encoded) != 322180 || fmt.Sprintf("%x", sha256.Sum256(encoded)) != "657932d81a4d0d100ff3b473911ccc7dc0af2f869711b79e2d96f10d1287cde2" {
		return nil, fmt.Errorf("altered USA release photo catalog")
	}
	photos, err := decodePhotoCatalogData(ctx, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	if d.Base.LogicalSHA256 != fmt.Sprintf("%x", base.digest) ||
		d.Socioeconomic.LogicalSHA256 != fmt.Sprintf("%x", modelDigest) ||
		d.Photos.LogicalSHA256 != fmt.Sprintf("%x", photos.digest) {
		return nil, fmt.Errorf("USA release logical digest mismatch")
	}

	// Verify every original rendition before exposing reusable application data.
	for _, face := range photos.document.Faces {
		for _, rendition := range face.Renditions {
			if rendition.Integrity == nil || rendition.Integrity.Bytes == 0 || rendition.Integrity.Bytes > 8<<20 {
				return nil, fmt.Errorf("missing or excessive USA rendition integrity")
			}
			data, err := readBoundedPhotoFile(ctx, filepath.Join(root, rendition.Path), int(rendition.Integrity.Bytes))
			if err != nil {
				return nil, err
			}
			if uint64(len(data)) != rendition.Integrity.Bytes ||
				fmt.Sprintf("%x", sha256.Sum256(data)) != rendition.Integrity.SHA256 {
				return nil, fmt.Errorf("altered USA release rendition")
			}
			config, err := webp.DecodeConfig(bytes.NewReader(data))
			if err != nil {
				return nil, err
			}
			if config.Width != int(rendition.Width) || config.Height != int(rendition.Height) {
				return nil, fmt.Errorf("incompatible USA release rendition dimensions")
			}
		}
	}
	catalog, err := prepareItalianPhotoCatalog(photos, document.Genders)
	if err != nil {
		return nil, err
	}
	digest, err := internationalDigest("fakedata/usa-internal-demo/v1", struct {
		Base   [32]byte
		Phones [32]byte
		Model  [32]byte
		Photos [32]byte
	}{base.digest, phones.identity, model.model.digest, photos.digest})
	if err != nil {
		return nil, err
	}
	country := &usaCountryData{baseDocument: document, base: base, phones: phones, socioeconomic: model}

	return &USAWorldData{country: country, catalog: catalog, root: root, digest: digest, packaged: true}, nil
}

// prepareUSAReleaseBase indexes only the exact approved BASE-v2 document.
// Production evidence is not a runtime dependency; this byte lock replaces its
// repeated source comparison without changing sampling inputs or algorithms.
func prepareUSAReleaseBase(ctx context.Context, encoded []byte) (usaBaseDocument, *usaBaseCandidate, error) {

	if len(encoded) != 3583463 ||
		fmt.Sprintf("%x", sha256.Sum256(encoded)) != "d8fd7c76142a86917ed4acec065951f8c5e9b44865e1bf6eed7f09f847eb5b80" {
		return usaBaseDocument{}, nil, fmt.Errorf("altered USA release base")
	}
	var d usaBaseDocument
	err := decodeInternationalJSON(ctx, bytes.NewReader(encoded), &d)
	if err != nil {
		return usaBaseDocument{}, nil, err
	}

	// Build the same runtime indexes from their already embedded scalar values.
	base := &usaBaseCandidate{
		ages: d.Ages, firstNames: &usaFirstNamesV2Candidate{}, lastNames: &usaSurname50KCandidate{},
		geography: &usaGeographyCandidate{byID: map[string]int{}},
	}
	for _, age := range d.Ages {
		base.ageWeights = append(base.ageWeights, age.Weight)
	}
	for i, group := range d.FirstNames {
		base.firstNames.groups[i], err = newUSANominalIndex(group.Entries, 1000, usaNamesMass)
		if err != nil {
			return usaBaseDocument{}, nil, err
		}
	}
	base.lastNames.names, err = newUSANominalIndex(d.LastNames, 50000, usaNamesMass)
	if err != nil {
		return usaBaseDocument{}, nil, err
	}
	for i, place := range d.Geography.Places {

		var state usaState
		for _, candidate := range usaStates() {
			if candidate.USPS == *place.Administrative[0].Code {
				state = candidate
				break
			}
		}
		if state.USPS == "" || state.Division != place.Territory {
			return usaBaseDocument{}, nil, fmt.Errorf("incompatible USA release state")
		}
		g := base.geography
		g.byID[place.ID] = i
		g.locations = append(g.locations, usaLocation{
			id: place.ID, name: place.Name, state: state, weight: place.Population, zips: d.Geography.PostalCodes[place.ID],
		})
		g.weights = append(g.weights, place.Population)
		g.total += place.Population
		g.divisionMasses[state.DivisionOrdinal] += place.Population

	}
	for _, street := range d.StreetNames.Entries {
		base.streets = append(base.streets, street.Text)
	}

	// Preserve the content fingerprints and the frozen BASE-v2 stream inputs.
	common := internationalBase{
		Schema: d.Schema, Release: d.Release, Profile: d.Profile, Country: d.Country, AnchorDate: d.AnchorDate,
		Genders: d.Genders, Ages: d.Ages, Cohorts: d.Cohorts, LastNames: d.LastNames, StreetNames: d.StreetNames,
		Geography: internationalGeography{
			PlaceCodeSystem: d.Geography.PlaceCodeSystem, TerritorySystem: d.Geography.TerritorySystem,
			PostalCodeSystem: d.Geography.PostalCodeSystem, Places: d.Geography.Places, PostalCodes: d.Geography.PostalCodes,
		},
	}
	for _, group := range d.FirstNames {
		common.FirstNames = append(common.FirstNames, internationalFirstNames{
			Gender: group.Gender, Cohort: group.Cohort, Entries: group.Entries,
		})
	}
	base.fingerprints, err = internationalBaseFingerprints(&common)
	if err != nil {
		return usaBaseDocument{}, nil, err
	}
	base.digest, err = internationalDigest("fakedata/base-data/json-v1", d)
	if err != nil {
		return usaBaseDocument{}, nil, err
	}
	for i, name := range []string{
		ageDatasetComponentName, firstNameDatasetComponentName, lastNameDatasetComponentName,
		placesDatasetComponentName, postalDatasetComponentName, streetNameDatasetComponentName,
	} {
		fingerprint, err := hex.DecodeString(usaBaseSamplingFingerprints[i])
		if err != nil {
			return usaBaseDocument{}, nil, err
		}
		base.dependencies[i], err = BytesComponent(name, fingerprint)
		if err != nil {
			return usaBaseDocument{}, nil, err
		}
	}

	return d, base, nil
}
