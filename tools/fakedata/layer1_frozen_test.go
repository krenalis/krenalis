// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sync"
	"testing"
)

// Historical CSV fingerprints are retained only for the unchanged test vectors.
const (
	ageWeightsSHA256  = "5abdc04bb817e5fc300ce71920c1dedf269551d3cc0065148264d26a39c84555"
	firstNamesSHA256  = "ed249d8095d3133d90a662bb3dc4d592941d9aa1de1d476a8000b392761ed8db"
	lastNamesSHA256   = "d44f1859847b5c94066d20c0846fb16859827802bdf878d59e17dc26a046f96b"
	placesSHA256      = "27539c5025412c2f03b199cbd6fdea7b9bda8fb57f6d8665883c70eeaa7b05b6"
	postalCodesSHA256 = "0fed3e2822c13a54c4255d2b83fe0fd63aea27b432d63a682b28f64f4981426c"
	streetNamesSHA256 = "13a757c70266ad96f0c1f9d76ff32597995090009e860eec86a1dded5b5095ec"
)

var frozenData struct {
	sync.Once
	data *italyData
	err  error
}

// frozenLayer1Data prepares the published JSON and restores only the
// historical dependency identities used by the unchanged frozen vectors.
func frozenLayer1Data(t *testing.T) (*italyData, error) {

	t.Helper()
	frozenData.Do(func() {

		base, err := readItalianBaseFixture(t)
		if err != nil {
			frozenData.err = err
			return
		}
		model, err := DecodeItalianSocioeconomicData(t.Context(),
			bytes.NewReader(internationalTestJSON(t, internationalSocioeconomicFixture(t))))
		if err != nil {
			frozenData.err = err
			return
		}
		photos := mustInternationalPhotos(t, internationalPhotoFixture())
		prepared, err := PrepareItalianWorldData(base, model, photos)
		if err != nil {
			frozenData.err = err
			return
		}
		data := *prepared.data
		for _, item := range []struct {
			component *Component
			name      string
			digest    string
		}{
			{&data.ageDependency, ageDatasetComponentName, ageWeightsSHA256},
			{&data.firstNameDependency, firstNameDatasetComponentName, firstNamesSHA256},
			{&data.lastNameDependency, lastNameDatasetComponentName, lastNamesSHA256},
			{&data.placesDependency, placesDatasetComponentName, placesSHA256},
			{&data.postalDependency, postalDatasetComponentName, postalCodesSHA256},
			{&data.streetNameDependency, streetNameDatasetComponentName, streetNamesSHA256},
		} {
			value, err := hex.DecodeString(item.digest)
			if err != nil {
				frozenData.err = err
				return
			}
			*item.component, err = BytesComponent(item.name, value)
			if err != nil {
				frozenData.err = err
				return
			}
		}
		frozenData.data = &data

	})

	return frozenData.data, frozenData.err
}

// newFrozenWorld configures the shared engine solely for historical vectors.
func newFrozenWorld(t *testing.T, config worldConfig) (*World, error) {

	t.Helper()
	reference, err := parseReferenceDate(config.ReferenceDate)
	if err != nil {
		return nil, err
	}
	data, err := frozenLayer1Data(t)
	if err != nil {
		return nil, err
	}
	err = config.FaceCatalog.validate()
	if err != nil {
		return nil, err
	}
	for _, gender := range []Gender{GenderFemale, GenderMale} {
		if !config.FaceCatalog.hasUsableGender(gender) {
			return nil, fmt.Errorf("%w: gender %s", ErrNoCompatiblePhoto, gender)
		}
	}

	return newWorldWithData(config, reference, data, PersonSpecVersion, MarketDataVersion)
}
