// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"fmt"
	"sort"
)

const (
	socioeconomicCompanyIndustry = iota
	socioeconomicCompanyEmployeeCount
	socioeconomicEducation
	socioeconomicPrimaryActivity
	socioeconomicEmployeeJobFamily
	socioeconomicEmployeeJobLevel
	socioeconomicEmployer
	socioeconomicSelfEmployedJobFamily
	socioeconomicSelfEmployedIndustry
	socioeconomicIncomeEmployee
	socioeconomicIncomeSelfEmployed
	socioeconomicIncomeUnemployed
	socioeconomicIncomeStudent
	socioeconomicIncomeRetired
	socioeconomicIncomeInactive
)

var socioeconomicPaths = [...]string{
	"socioeconomic/v1/company/industry",
	"socioeconomic/v1/company/employee-count",
	"socioeconomic/v1/person/education",
	"socioeconomic/v1/person/primary-activity",
	"socioeconomic/v1/person/employee/job-family",
	"socioeconomic/v1/person/employee/job-level",
	"socioeconomic/v1/person/employee/employer",
	"socioeconomic/v1/person/self-employed/job-family",
	"socioeconomic/v1/person/self-employed/industry",
	"socioeconomic/v1/person/income/employee",
	"socioeconomic/v1/person/income/self-employed",
	"socioeconomic/v1/person/income/unemployed",
	"socioeconomic/v1/person/income/student",
	"socioeconomic/v1/person/income/retired",
	"socioeconomic/v1/person/income/inactive",
}

// A nonnegative entry identifies the node's own age map in the compiled model.
var socioeconomicNodeAges = [...]int{-1, -1, 0, 1, -1, 2, -1, -1, -1, -1, 3, 4, 5, 6, 7}

// category samples one row, preserving the ordinal slots of zero weights.
func (w *SocioeconomicWorld) category(index uint64, node int, table socioeconomicWeights, row int, parents ...socioeconomicParent) (int, error) {
	if table.width <= 0 || row < 0 || row >= len(table.cumulative)/table.width {
		return 0, fmt.Errorf("invalid socioeconomic categorical row")
	}
	start := row * table.width
	return w.sample(index, node, table.cumulative[start:start+table.width], parents...)
}

// income resolves one branch-local tuple and samples its final EUR support.
func (w *SocioeconomicWorld) income(index uint64, branch, tuple int, parents ...socioeconomicParent) (AnnualIncomeEUR, error) {

	if branch < 0 || branch >= len(w.model.income) {
		return 0, fmt.Errorf("invalid socioeconomic Income branch")
	}
	income := &w.model.income[branch]
	if tuple < 0 || tuple >= len(income.tupleMap) {
		return 0, fmt.Errorf("invalid socioeconomic Income tuple")
	}
	law := int(income.tupleMap[tuple])
	if law >= len(income.laws) {
		return 0, fmt.Errorf("invalid socioeconomic Income law")
	}

	value, err := w.point(index, socioeconomicIncomeEmployee+branch, income.laws[law], parents...)
	if err != nil {
		return 0, err
	}

	return AnnualIncomeEUR(value), nil
}

// point samples one value from a compiled finite law.
func (w *SocioeconomicWorld) point(index uint64, node int, law socioeconomicLaw, parents ...socioeconomicParent) (uint32, error) {
	if len(law.values) != len(law.cumulative) {
		return 0, fmt.Errorf("invalid socioeconomic finite law")
	}
	position, err := w.sample(index, node, law.cumulative, parents...)
	if err != nil {
		return 0, err
	}
	return law.values[position], nil
}

// sample makes one bounded draw and selects its strict cumulative interval.
func (w *SocioeconomicWorld) sample(index uint64, node int, cumulative []uint64, parents ...socioeconomicParent) (int, error) {

	if len(cumulative) == 0 || cumulative[len(cumulative)-1] == 0 {
		return 0, fmt.Errorf("socioeconomic selection has no positive total")
	}

	rng, err := w.stream(index, node, parents...)
	if err != nil {
		return 0, err
	}
	draw, err := rng.Uint64n(cumulative[len(cumulative)-1])
	if err != nil {
		return 0, err
	}
	position := socioeconomicPosition(cumulative, draw)
	if position == len(cumulative) {
		return 0, fmt.Errorf("socioeconomic selection has no interval")
	}

	return position, nil
}

// stream uses the base factory, raw coordinate and node-local dependencies.
func (w *SocioeconomicWorld) stream(index uint64, node int, parents ...socioeconomicParent) (splitMix64, error) {

	if node < 0 || node >= len(socioeconomicPaths) {
		return splitMix64{}, fmt.Errorf("invalid socioeconomic node")
	}
	model, err := BytesComponent("model", w.model.fingerprints[node][:])
	if err != nil {
		return splitMix64{}, err
	}
	components := make([]Component, 1, 1+len(parents))
	components[0] = model
	for _, parent := range parents {
		if parent.value < 0 {
			return splitMix64{}, fmt.Errorf("invalid socioeconomic ordinal")
		}
		component, err := Uint64Component(parent.name, uint64(parent.value))
		if err != nil {
			return splitMix64{}, err
		}
		components = append(components, component)
	}

	kind := "person"
	if node <= socioeconomicCompanyEmployeeCount {
		kind = "company"
	}

	return w.base.streams.entityStream(kind, index, socioeconomicPaths[node], components...)
}

// socioeconomicParent holds one typed ordinal dependency.
type socioeconomicParent struct {
	name  string
	value int
}

func socioeconomicPosition(cumulative []uint64, draw uint64) int {
	return sort.Search(len(cumulative), func(i int) bool { return cumulative[i] > draw })
}
