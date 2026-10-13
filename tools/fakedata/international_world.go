// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"fmt"
)

// Versions of the explicit Italian international JSON entry point.
const (
	ItalianJSONPersonSpecVersion = "person-v2"
	ItalianJSONLayer1Protocol    = "layer1-json-v1"
)

// ItalianWorldConfig supplies coordinates for a prepared Italian JSON World.
type ItalianWorldConfig struct {
	IdentityNamespace IdentityNamespace
	WorldSeed         uint64
	ReferenceDate     string
}

// ItalianWorldData owns a complete, immutable, compatible set of external data.
// Its zero value is invalid. Worlds retain it independently of the source.
// Callers must not overwrite prepared data or its decoded input documents.
type ItalianWorldData struct {
	base          *ItalianBaseData
	socioeconomic *ItalianSocioeconomicData
	photos        *PhotoCatalogData
	data          *italyData
	catalog       *FaceCatalog
	packageDigest [32]byte
}

// PrepareItalianWorldData checks composition and builds reusable runtime data.
// Inputs must be complete decoded v1 documents. It performs bounded local work
// without acquisition, CSV conversion, or numerical model recompilation.
// It returns ErrNoCompatiblePhoto when a generable gender has no candidate.
func PrepareItalianWorldData(base *ItalianBaseData, socioeconomic *ItalianSocioeconomicData, photos *PhotoCatalogData) (*ItalianWorldData, error) {
	return prepareItalianWorldData(base, socioeconomic, photos, nil)
}

// BaseRelease returns the base document release used in the World identity.
func (d *ItalianWorldData) BaseRelease() (string, error) {
	if d == nil || d.data == nil {
		return "", fmt.Errorf("uninitialized Italian World data")
	}
	return d.base.document.Release, nil
}

// Model returns the already compiled immutable model of this combination.
func (d *ItalianWorldData) Model() (*SocioeconomicModel, error) {
	if d == nil || d.data == nil {
		return nil, fmt.Errorf("uninitialized Italian World data")
	}
	return d.socioeconomic.model, nil
}

// PackageDigest identifies the country and three complete logical documents.
func (d *ItalianWorldData) PackageDigest() ([32]byte, error) {
	if d == nil || d.data == nil {
		return [32]byte{}, fmt.Errorf("uninitialized Italian World data")
	}
	return d.packageDigest, nil
}

// PhotoCatalog returns the prepared selection and declared rendition metadata.
func (d *ItalianWorldData) PhotoCatalog() (*FaceCatalog, error) {
	if d == nil || d.data == nil {
		return nil, fmt.Errorf("uninitialized Italian World data")
	}
	return d.catalog, nil
}

// NewItalianWorldV1 constructs a person-v2 World from complete prepared data.
// It performs no acquisition or validation of already prepared documents.
// Invalid coordinates return ErrInvalidIdentityNamespace or
// ErrInvalidReferenceDate.
func NewItalianWorldV1(config ItalianWorldConfig, data *ItalianWorldData) (*World, error) {

	if data == nil || data.data == nil || data.catalog == nil || data.socioeconomic == nil {
		return nil, fmt.Errorf("complete prepared Italian World data is required")
	}
	reference, err := parseReferenceDate(config.ReferenceDate)
	if err != nil {
		return nil, err
	}
	world, err := newWorldWithData(worldConfig{
		IdentityNamespace: config.IdentityNamespace, WorldSeed: config.WorldSeed,
		ReferenceDate: config.ReferenceDate, FaceCatalog: data.catalog,
	}, reference, data.data, ItalianJSONPersonSpecVersion, data.base.document.Release)
	if err != nil {
		return nil, err
	}
	world.italian = data

	return world, nil
}

// prepareItalianWorldData joins validated documents and transfers their values
// into the existing samplers. Acquisition may supply an already prepared
// Italian catalog; local preparation builds it once when catalog is nil.
func prepareItalianWorldData(base *ItalianBaseData, socioeconomic *ItalianSocioeconomicData, photos *PhotoCatalogData, catalog *FaceCatalog) (*ItalianWorldData, error) {

	if base == nil || !base.initialized || socioeconomic == nil || socioeconomic.model == nil ||
		photos == nil || !photos.initialized {
		return nil, fmt.Errorf("complete decoded Italian documents are required")
	}
	b, m := &base.document, &socioeconomic.document
	if b.Country != "IT" || m.Country != "IT" || b.AnchorDate != m.AnchorDate ||
		b.Geography.TerritorySystem != m.Dimensions.Territory.SemanticID {
		return nil, fmt.Errorf("incompatible Italian country, anchor or territory system")
	}
	for _, place := range b.Geography.Places {
		matches := 0
		for _, territory := range m.Dimensions.Territory.Categories {
			if place.Territory == territory.Code {
				matches++
			}
		}
		if matches != 1 {
			return nil, fmt.Errorf("unresolved Italian place territory")
		}
	}

	// Check the birth interval used by the existing Italian birth algorithm.
	for _, age := range b.Ages {
		if age.Age < m.AgeDomain.Minimum || age.Age > m.AgeDomain.Maximum {
			return nil, fmt.Errorf("generable age outside socioeconomic domain")
		}
		for year := uint32(2026) - age.Age - 1; year <= uint32(2026)-age.Age; year++ {
			matches := 0
			for _, cohort := range b.Cohorts {
				if year >= cohort.FromYear && year <= cohort.ToYear {
					matches++
				}
			}
			if matches != 1 {
				return nil, fmt.Errorf("generable birth year lacks a unique cohort")
			}
		}
	}
	err := validateInternationalTaxonomy(photos.document.GenderTaxonomy, b.Genders)
	if err != nil {
		return nil, err
	}
	if catalog == nil {
		catalog, err = prepareItalianPhotoCatalog(photos, b.Genders)
		if err != nil {
			return nil, err
		}
	}

	// Transfer validated values directly into the existing sampling structures.
	data := &italyData{
		firstNames: map[firstNameGroup][]firstNameRow{}, firstNameWeights: map[firstNameGroup][]uint64{},
		postalCodes: b.Geography.PostalCodes,
	}
	for _, age := range b.Ages {
		data.ages = append(data.ages, ageRow{age: int(age.Age), weight: age.Weight})
		data.ageWeights = append(data.ageWeights, age.Weight)
	}
	for _, group := range b.FirstNames {
		key := firstNameGroup{gender: Gender(group.Gender), cohort: group.Cohort}
		for _, name := range group.Entries {
			data.firstNames[key] = append(data.firstNames[key], firstNameRow{
				display: name.Display, asciiAtom: *name.ASCIIAtom, weight: name.Weight,
			})
			data.firstNameWeights[key] = append(data.firstNameWeights[key], name.Weight)
		}
	}
	for _, name := range b.LastNames {
		data.lastNames = append(data.lastNames, nameRow{
			display: name.Display, asciiAtom: *name.ASCIIAtom, weight: name.Weight,
		})
		data.lastNameWeights = append(data.lastNameWeights, name.Weight)
	}
	for _, place := range b.Geography.Places {
		data.places = append(data.places, placeRow{
			istatCode: place.ID, municipality: place.Name, stateProv: *place.Administrative[0].Code,
			region: place.Territory, population: place.Population,
		})
	}
	data.placeWeightBuckets, data.placeWeightBucketTotals = buildPlaceWeightBuckets(data.places)
	for _, street := range b.StreetNames.Entries {
		data.streetNames = append(data.streetNames, nameRow{display: street.Text, weight: street.Weight})
		data.streetNameWeights = append(data.streetNameWeights, street.Weight)
	}
	fingerprints, err := base.fingerprints()
	if err != nil {
		return nil, err
	}
	dependencies := []*Component{
		&data.ageDependency, &data.firstNameDependency, &data.lastNameDependency,
		&data.placesDependency, &data.postalDependency, &data.streetNameDependency,
	}
	for i, name := range []string{
		ageDatasetComponentName, firstNameDatasetComponentName, lastNameDatasetComponentName,
		placesDatasetComponentName, postalDatasetComponentName, streetNameDatasetComponentName,
	} {
		*dependencies[i], err = BytesComponent(name, fingerprints[i][:])
		if err != nil {
			return nil, err
		}
	}

	baseDigest, err := internationalDigest("fakedata/base-data/json-v1", b)
	if err != nil {
		return nil, err
	}
	modelDigest, err := internationalDigest("fakedata/socioeconomic-data/json-v1", m)
	if err != nil {
		return nil, err
	}
	country, err := StringComponent("country", b.Country)
	if err != nil {
		return nil, err
	}
	components := []Component{country}
	for _, entry := range []struct {
		name   string
		digest [32]byte
	}{{"base", baseDigest}, {"socioeconomic", modelDigest}, {"photos", photos.digest}} {
		component, err := BytesComponent(entry.name, entry.digest[:])
		if err != nil {
			return nil, err
		}
		components = append(components, component)
	}
	packageDigest, err := canonicalDigest("fakedata/country-package/v1", components)
	if err != nil {
		return nil, err
	}

	return &ItalianWorldData{
		base: base, socioeconomic: socioeconomic, photos: photos, data: data, catalog: catalog,
		packageDigest: packageDigest,
	}, nil
}
