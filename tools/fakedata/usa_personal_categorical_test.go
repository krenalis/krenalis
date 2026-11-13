// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

// TestUSAPersonalCategoricalAgePartitions checks every age, including all
// consumer boundaries, independently of the derived birth-date observation.
func TestUSAPersonalCategoricalAgePartitions(t *testing.T) {

	m := readUSAPersonalCategoricalModel(t)
	bounds := [3][]int{{21, 25, 35, 45, 55, 65}, {21, 25, 35, 45, 55, 60, 65, 70}, {21, 25, 35, 45, 55, 65}}
	for consumer, lower := range bounds {
		for age := 21; age <= 75; age++ {
			want := 0
			for i, boundary := range lower {
				if age >= boundary {
					want = i
				}
			}
			if got := int(m.ages[consumer][age-21]); got != want {
				t.Fatalf("expected consumer %d age %d bucket %d, got %d", consumer, age, want, got)
			}
		}
	}
	m.ages[0][0]++
	if m.ages[2][0] != 0 {
		t.Fatal("expected separately owned consumer maps, got mutable alias")
	}

}

// TestUSAPersonalCategoricalPublicBoundaries rejects partial Worlds and absent
// Income through the actual existing constructors and runtime methods.
func TestUSAPersonalCategoricalPublicBoundaries(t *testing.T) {
	w := readUSAPersonalCategoricalHarness(t)
	_, err := newUSAPersonalCategoricalHarness(w.base, w.model, nil)
	usaCompanyEmployerRejected(t, err, nil, "USA model without Employer")
	missing := *w.companies
	missing.sampler.employers[39] = socioeconomicEmployerIndex{}
	_, err = newUSAPersonalCategoricalHarness(w.base, w.model, &missing)
	usaCompanyEmployerRejected(t, err, nil, "missing Employer condition")
	_, err = NewSocioeconomicWorld(w.companies.sampler.base, w.companies.sampler.model)
	usaCompanyEmployerRejected(t, err, ErrInvalidSocioeconomicWorld, "public partial World")
	_, err = w.companies.sampler.Facts(1)
	usaCompanyEmployerRejected(t, err, ErrInvalidSocioeconomicWorld, "public partial Facts")
	_, err = w.companies.sampler.Person(1)
	usaCompanyEmployerRejected(t, err, ErrInvalidSocioeconomicWorld, "public partial Person")
	_, err = w.companies.sampler.income(1, 0, 0)
	usaCompanyEmployerRejected(t, err, nil, "absent Income used as a valid zero law")
	unknown, err := entityIDForIndex(w.base.namespace, "company", 65537)
	if err != nil {
		t.Fatalf("expected representable nonmember ID, got %v", err)
	}
	_, err = w.companyByID(unknown)
	usaCompanyEmployerRejected(t, err, ErrCompanyIndexOutOfRange, "unknown operational CompanyID")
	for _, index := range []PersonIndex{0, Layer1MaxPersonIndex + 1, PersonIndex(math.MaxUint64)} {
		record, err := w.person(index)
		if !reflect.DeepEqual(record, usaPersonalCategoricalRecord{}) {
			t.Fatal("expected no partial record on invalid coordinate, got record")
		}
		usaCompanyEmployerRejected(t, err, nil, "invalid Person coordinate")
	}
	for _, typ := range []reflect.Type{
		reflect.TypeFor[usaPersonalCategoricalCandidate](), reflect.TypeFor[usaPersonalCategoricalRecord](),
	} {
		for i := range typ.NumField() {
			if strings.Contains(strings.ToLower(typ.Field(i).Name), "income") {
				t.Fatal("expected no Income placeholder field, got Income field")
			}
		}
	}
}

func readUSAPersonalCategoricalHarness(t *testing.T) *usaPersonalCategoricalHarness {
	t.Helper()
	return readUSAIncomeHarness(t).categorical
}

func readUSAPersonalCategoricalModel(t *testing.T) *usaPersonalCategoricalCandidate {
	t.Helper()
	return readUSAPersonalCategoricalHarness(t).model
}
