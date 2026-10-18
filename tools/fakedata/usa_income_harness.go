// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"fmt"
	"slices"
)

// This internal sampler returns no public Person, Facts or World.
type usaIncomeHarness struct {
	categorical *usaPersonalCategoricalHarness
	income      *usaIncomeCandidate
}

func (w *usaIncomeHarness) person(index PersonIndex) (usaIncomeRecord, error) {

	if w == nil || w.categorical == nil || w.income == nil || w.income.identity == ([32]byte{}) {
		return usaIncomeRecord{}, fmt.Errorf("admitted internal USA Income harness is required")
	}
	record, err := w.categorical.person(index)
	if err != nil {
		return usaIncomeRecord{}, err
	}
	income, trace, err := w.sampleIncome(record)
	if err != nil {
		return usaIncomeRecord{}, err
	}

	return usaIncomeRecord{categorical: record, income: income, trace: trace}, nil
}

// sampleIncome resolves every authorized axis and map slot before the single
// frozen bounded POINT draw. Law IDs and support positions are only addresses.
func (w *usaIncomeHarness) sampleIncome(record usaPersonalCategoricalRecord) (annualIncomeUSD, usaIncomeTrace, error) {

	if w == nil || w.categorical == nil || w.categorical.base == nil || w.income == nil ||
		w.income.identity == ([32]byte{}) || record.coordinate == 0 || record.coordinate > Layer1MaxPersonIndex {
		return 0, usaIncomeTrace{}, fmt.Errorf("valid admitted Income state and Person coordinate are required")
	}
	branch := slices.Index(socioeconomicActivities[:], record.activity)
	age, division := record.layer1.anchorAge, int(record.layer1.divisionOrdinal)
	if branch < 0 || age < 21 || age > 75 || division < 0 || division >= 9 {
		return 0, usaIncomeTrace{}, fmt.Errorf("invalid Income activity, anchor age or Division")
	}
	var parents []socioeconomicParent
	var sizes []int
	switch branch {

	case 0:

		if record.employee == nil || record.selfEmployed != nil || record.company == nil {
			return 0, usaIncomeTrace{}, fmt.Errorf("employee Income requires the selected occupational branch")
		}
		company, err := w.categorical.companies.company(record.employee.Employer)
		if err != nil {
			return 0, usaIncomeTrace{}, err
		}
		family := slices.Index(socioeconomicJobFamilies[:], record.employee.JobFamily)
		level := slices.Index(socioeconomicJobLevels[:], record.employee.JobLevel)
		industry := slices.Index(socioeconomicIndustries[:], company.Industry)
		band := slices.Index(socioeconomicSizeBands[:], company.SizeBand)
		if band <= 0 {
			return 0, usaIncomeTrace{}, fmt.Errorf("employee Income cannot address an inapplicable SizeBand")
		}
		parents = []socioeconomicParent{
			{"region", division}, {"job-family", family}, {"job-level", level}, {"industry", industry}, {"size-band", band},
		}
		sizes = []int{9, 10, 4, 14, 5}

	case 1:

		if record.selfEmployed == nil || record.employee != nil || record.company != nil {
			return 0, usaIncomeTrace{}, fmt.Errorf("self-employed Income requires only its occupational branch")
		}
		family := slices.Index(socioeconomicJobFamilies[:], record.selfEmployed.JobFamily)
		industry := slices.Index(socioeconomicIndustries[:], record.selfEmployed.Industry)
		bucket := int(w.income.numeric.ages[3][age-21])
		parents = []socioeconomicParent{
			{"age-bucket", bucket}, {"region", division}, {"job-family", family}, {"industry", industry},
		}
		sizes = []int{6, 9, 10, 14}

	default:

		if record.employee != nil || record.selfEmployed != nil || record.company != nil {
			return 0, usaIncomeTrace{}, fmt.Errorf("nonworking Income cannot receive occupational payload")
		}
		education := slices.Index(socioeconomicEducationLevels[:], record.education)
		bucket := int(w.income.numeric.ages[branch+2][age-21])
		parents = []socioeconomicParent{{"age-bucket", bucket}, {"education", education}, {"region", division}}
		sizes = []int{6, 7, 9}

	}

	model, err := BytesComponent("model", w.income.fingerprints[branch][:])
	if err != nil {
		return 0, usaIncomeTrace{}, err
	}
	components := []Component{model}
	tuple := 0
	for axis, parent := range parents {
		if parent.value < 0 || parent.value >= sizes[axis] {
			return 0, usaIncomeTrace{}, fmt.Errorf("invalid Income parent ordinal")
		}
		tuple = tuple*sizes[axis] + parent.value
		component, err := Uint64Component(parent.name, uint64(parent.value))
		if err != nil {
			return 0, usaIncomeTrace{}, err
		}
		components = append(components, component)
	}
	pool := &w.income.numeric.income[branch]
	if tuple >= len(pool.tupleMap) {
		return 0, usaIncomeTrace{}, fmt.Errorf("missing USA Income map slot")
	}
	lawID := int(pool.tupleMap[tuple])
	if lawID == 65535 || lawID >= len(pool.laws) {
		return 0, usaIncomeTrace{}, fmt.Errorf("inapplicable or invalid USA Income law reference")
	}
	law := pool.laws[lawID]
	if len(law.values) != 385 || len(law.cumulative) != 385 || law.cumulative[384] == 0 {
		return 0, usaIncomeTrace{}, fmt.Errorf("invalid compiled USA Income law")
	}
	node := socioeconomicIncomeEmployee + branch
	rng, err := w.categorical.base.streams.entityStream("person", uint64(record.coordinate),
		usaSocioeconomicPaths[node], components...)
	if err != nil {
		return 0, usaIncomeTrace{}, err
	}
	before := rng.state
	total := law.cumulative[len(law.cumulative)-1]
	draw, err := rng.Uint64n(total)
	if err != nil {
		return 0, usaIncomeTrace{}, err
	}
	position := socioeconomicPosition(law.cumulative, draw)
	if position >= len(law.values) {
		return 0, usaIncomeTrace{}, fmt.Errorf("USA Income draw has no interval")
	}
	trace := usaIncomeTrace{branch: branch, tuple: tuple, law: lawID, parents: parents,
		position: position, before: before, after: rng.state, draw: draw, total: total}

	return annualIncomeUSD(law.values[position]), trace, nil
}

type usaIncomeRecord struct {
	categorical usaPersonalCategoricalRecord
	income      annualIncomeUSD
	trace       usaIncomeTrace
}

type usaIncomeTrace struct {
	branch   int
	tuple    int
	law      int
	parents  []socioeconomicParent
	position int
	before   uint64
	after    uint64
	draw     uint64
	total    uint64
}
