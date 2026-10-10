// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"math"
	"slices"
	"testing"
)

// TestSocioeconomicCompanyUniverse checks both Company nodes against explicit
// protocol addresses and all forty indexes against independent raw weights.
func TestSocioeconomicCompanyUniverse(t *testing.T) {

	base := mustLayer1World(t, PopulationAnchorDate, layer1TestSeed)
	m := socioeconomicTestModel(t, nil)
	w := mustSocioeconomicWorld(t, base, m)
	for index := CompanyIndex(1); uint64(index) <= m.companyCount; index++ {
		company := socioeconomicCompanyAt(t, w, index)
		industry := socioeconomicReferenceSelection(t, base, "company", uint64(index),
			"socioeconomic/v1/company/industry", m.fingerprints[0],
			socioeconomicRawWeights(m.companyIndustry.cumulative))
		countLaw := m.employeeCount[industry]
		position := socioeconomicReferenceSelection(t, base, "company", uint64(index),
			"socioeconomic/v1/company/employee-count", m.fingerprints[1], socioeconomicRawWeights(countLaw.cumulative),
			mustUint64Component(t, "industry", uint64(industry)))
		if company.Industry != socioeconomicIndustries[industry] || company.EmployeeCount != countLaw.values[position] {
			t.Fatalf("expected Company Industry/count %s/%d, got %+v",
				socioeconomicIndustries[industry], countLaw.values[position], company)
		}
		if company.SizeBand != socioeconomicExpectedBand(company.EmployeeCount) {
			t.Fatalf("expected count-derived band %s, got %s",
				socioeconomicExpectedBand(company.EmployeeCount), company.SizeBand)
		}
		decoded, err := CompanyIndexFromID(base.namespace, company.ID)
		if err != nil {
			t.Fatalf("expected Company identity inverse, got %v", err)
		}
		if decoded != index {
			t.Fatalf("expected Company coordinate %d, got %d", index, decoded)
		}
	}

	for family := range 10 {
		for level := range 4 {
			index := &w.employers[4*family+level]
			weights := socioeconomicReferenceEmployerWeights(t, w, family, level)
			var total uint64
			position := 0
			for offset, weight := range weights {
				if weight == 0 {
					continue
				}
				total += weight
				if position >= len(index.companies) || index.companies[position] != CompanyIndex(offset+1) ||
					index.cumulative[position] != total {
					t.Fatalf("expected condition %d/%d interval %d ending at %d, got %+v",
						family, level, offset+1, total, index)
				}
				// Both ends of every positive interval use strict-greater lookup.
				for _, draw := range []uint64{total - weight, total - 1} {
					if got := socioeconomicPosition(index.cumulative, draw); got != position {
						t.Fatalf("expected interval %d for draw %d, got %d", position, draw, got)
					}
				}
				position++
			}
			if position != len(index.companies) || len(index.cumulative) != position || total == 0 {
				t.Fatalf("expected complete positive Employer index, got %d/%d entries and total %d",
					position, len(index.companies), total)
			}
		}
	}

	for _, test := range []struct {
		count uint32
		band  CompanySizeBand
	}{
		{0, CompanySizeBandZero}, {1, CompanySizeBandMicro}, {9, CompanySizeBandMicro},
		{10, CompanySizeBandSmall}, {49, CompanySizeBandSmall}, {50, CompanySizeBandMedium},
		{249, CompanySizeBandMedium}, {250, CompanySizeBandLarge}, {math.MaxUint32, CompanySizeBandLarge},
	} {
		if got := socioeconomicSizeBands[socioeconomicSizeBand(test.count)]; got != test.band {
			t.Fatalf("expected band %s at count %d, got %s", test.band, test.count, got)
		}
	}

}

// TestSocioeconomicEmployerRoutes checks every public Facts F/J route against
// the foundation's independent raw-weight sampler.
func TestSocioeconomicEmployerRoutes(t *testing.T) {
	base := mustLayer1World(t, PopulationAnchorDate, layer1TestSeed)
	for family := range 10 {
		for level := range 4 {
			model := socioeconomicTestModel(t, func(document map[string]any) {
				setSocioeconomicTestCategory(document, "primaryActivity", 6, func(int) int { return 0 })
				setSocioeconomicTestCategory(document, "employeeJobFamily", 10, func(int) int { return family })
				setSocioeconomicTestCategory(document, "employeeJobLevel", 4, func(int) int { return level })
			})
			world := mustSocioeconomicWorld(t, base, model)
			weights := socioeconomicReferenceEmployerWeights(t, world, family, level)
			for _, coordinate := range []PersonIndex{1, 42, 1009, Layer1MaxPersonIndex} {
				facts := socioeconomicFactsAt(t, world, coordinate)
				position := socioeconomicReferenceSelection(t, base, "person", uint64(coordinate),
					"socioeconomic/v1/person/employee/employer", model.fingerprints[6], weights,
					mustUint64Component(t, "job-family", uint64(family)),
					mustUint64Component(t, "job-level", uint64(level)))
				if facts.Employee == nil || facts.Employee.Employer != CompanyIndex(position+1) ||
					facts.Employee.JobFamily != socioeconomicJobFamilies[family] ||
					facts.Employee.JobLevel != socioeconomicJobLevels[level] {
					t.Fatalf("expected route %d/%d and Employer %d, got %+v", family, level, position+1, facts.Employee)
				}
			}
		}
	}
}

func socioeconomicExpectedBand(count uint32) CompanySizeBand {
	if count == 0 {
		return CompanySizeBandZero
	}
	if count <= 9 {
		return CompanySizeBandMicro
	}
	if count <= 49 {
		return CompanySizeBandSmall
	}
	if count <= 249 {
		return CompanySizeBandMedium
	}
	return CompanySizeBandLarge
}

func socioeconomicRawWeights(cumulative []uint64) []uint64 {
	weights := make([]uint64, len(cumulative))
	var previous uint64
	for i, value := range cumulative {
		weights[i] = value - previous
		previous = value
	}
	return weights
}

func socioeconomicReferenceEmployerWeights(t *testing.T, world *SocioeconomicWorld, family, level int) []uint64 {
	t.Helper()
	weights := make([]uint64, world.model.companyCount)
	for offset := range weights {
		company := socioeconomicCompanyAt(t, world, CompanyIndex(offset+1))
		if company.EmployeeCount == 0 {
			continue
		}
		industry := slices.Index(socioeconomicIndustries[:], company.Industry)
		band := slices.Index(socioeconomicSizeBands[:], company.SizeBand)
		// Fixture factors deliberately have a common divisor; these exact raw
		// products would expose an incorrect Employer GCD normalization.
		weights[offset] = min(uint64(company.EmployeeCount), 17) *
			uint64(2*(family+1)*(industry+1)) * uint64(3*(level+1)*band)
	}
	return weights
}

func socioeconomicReferenceSelection(t *testing.T, base *World, kind string, index uint64, path string, fingerprint [32]byte, weights []uint64, parents ...Component) int {
	t.Helper()
	components := append([]Component{mustBytesComponent(t, "model", fingerprint[:])}, parents...)
	rng, err := base.streams.entityStream(kind, index, path, components...)
	if err != nil {
		t.Fatalf("expected reference stream, got %v", err)
	}
	position, err := rng.WeightedIndex(weights)
	if err != nil {
		t.Fatalf("expected reference weighted selection, got %v", err)
	}
	return position
}
