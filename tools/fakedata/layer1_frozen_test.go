// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"testing"
)

var frozenData struct {
	sync.Once
	data *italyData
	err  error
}

// frozenLayer1Data projects the offline CSV conversion through the JSON
// preparation, then restores only the historical dependency identities for
// frozen vectors. No historical loader or generator is linked into runtime.
func frozenLayer1Data(t *testing.T) (*italyData, error) {

	t.Helper()
	frozenData.Do(func() {

		base, err := ConvertItalianBaseData(t.Context(), os.DirFS("data/it-2026a"))
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
