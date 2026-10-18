// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

// TestUSABaseV2Preservation compares the current loader with hashes captured
// before adoption: 12297 photographed people and three full Company rosters.
// Uppercasing names only in the hash preserves the selected name identity.
func TestUSABaseV2Preservation(t *testing.T) {

	data, err := LoadUSAWorldData(t.Context(), "../..")
	if err != nil {
		t.Fatalf("expected loaded demo, got %v", err)
	}
	result := map[string][2]string{}
	for _, seed := range []uint64{0, 2026, ^uint64(0)} {

		world, err := newUSACountryWorld(data.country, IdentityNamespace{Name: "usa-layer1", Generation: 1}, seed)
		if err != nil {
			t.Fatalf("expected World, got %v", err)
		}
		world.catalog = data.catalog
		people, companies := sha256.New(), sha256.New()
		coordinates := []PersonIndex{65536, 999983, Layer1MaxPersonIndex}
		for i := PersonIndex(1); i <= 4096; i++ {
			coordinates = append(coordinates, i)
		}

		for _, index := range coordinates {

			person, err := world.person(index)
			if err != nil {
				t.Fatalf("expected person, got %v", err)
			}
			person.Person.FirstName = strings.ToUpper(person.Person.FirstName)
			person.Person.LastName = strings.ToUpper(person.Person.LastName)
			encoded, err := internationalCanonical(person)
			if err != nil {
				t.Fatalf("expected person JSON, got %v", err)
			}
			people.Write(encoded)

		}

		for index := CompanyIndex(1); index <= 65536; index++ {

			company, err := world.sampler.categorical.companies.company(index)
			if err != nil {
				t.Fatalf("expected company, got %v", err)
			}
			encoded, err := internationalCanonical(company)
			if err != nil {
				t.Fatalf("expected company JSON, got %v", err)
			}
			companies.Write(encoded)

		}

		result[fmt.Sprint(seed)] = [2]string{fmt.Sprintf("%x", people.Sum(nil)), fmt.Sprintf("%x", companies.Sum(nil))}

	}

	encoded, err := os.ReadFile("testdata/usa-base-v2-preservation.json")
	if err != nil {
		t.Fatalf("expected prior USA facts, got %v", err)
	}
	var expected map[string][2]string
	err = json.Unmarshal(encoded, &expected)
	if err != nil {
		t.Fatalf("expected prior USA hashes, got %v", err)
	}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected unchanged USA facts, selections and photos %v, got %v", expected, result)
	}

	t.Log("12297 photographed profiles and 196608 Companies match the baseline before adoption")

}
