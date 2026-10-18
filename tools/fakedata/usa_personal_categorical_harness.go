// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"fmt"
	"slices"
)

// This internal sampler serves diagnostics and tests; public constructors
// cannot receive its partial categorical record as a completed Person or World.
type usaPersonalCategoricalHarness struct {
	base      *usaLayer1Harness
	model     *usaPersonalCategoricalCandidate
	companies *usaCompanyEmployerHarness
}

func newUSAPersonalCategoricalHarness(base *usaLayer1Harness, model *usaPersonalCategoricalCandidate, companies *usaCompanyEmployerHarness) (*usaPersonalCategoricalHarness, error) {

	if base == nil || base.base == nil || base.base.digest == ([32]byte{}) ||
		model == nil || model.identity == ([32]byte{}) || companies == nil || companies.model == nil ||
		companies.model.identity == ([32]byte{}) || companies.sampler.base == nil ||
		companies.sampler.base.streams != base.streams || companies.sampler.base.namespace != base.namespace ||
		companies.model.numeric.territories != socioeconomicUSADivisions ||
		len(companies.sampler.companyCounts) != 65536 || len(companies.sampler.companyIndustries) != 65536 {
		return nil, fmt.Errorf("admitted USA Layer 1, six personal nodes and Employer roster are required")
	}
	for _, index := range companies.sampler.employers {
		if len(index.companies) != 65536 || len(index.cumulative) != 65536 || index.cumulative[65535] == 0 {
			return nil, fmt.Errorf("complete USA Employer condition coverage is required")
		}
	}

	return &usaPersonalCategoricalHarness{base: base, model: model, companies: companies}, nil
}

// category uses the admitted USA path registry and the frozen single bounded
// draw/strict cumulative mapping. All parents are explicit ordered ordinals.
func (w *usaPersonalCategoricalHarness) category(record *usaPersonalCategoricalRecord, modelNode int, parents ...socioeconomicParent) (int, error) {

	if modelNode < 0 || modelNode >= len(usaPersonalCategoricalNodes) {
		return 0, fmt.Errorf("unknown USA personal categorical node")
	}
	if (modelNode == 2 || modelNode == 3) && record.activity != PrimaryActivityEmployee ||
		(modelNode == 4 || modelNode == 5) && record.activity != PrimaryActivitySelfEmployed {
		return 0, fmt.Errorf("USA occupational node is not applicable")
	}
	names := [6][]string{
		{"age-bucket", "region"}, {"age-bucket", "education", "region"}, {"education", "region"},
		{"age-bucket", "education", "job-family"}, {"education", "region"}, {"job-family"},
	}
	sizes := [6][]int{{6, 9}, {8, 7, 9}, {7, 9}, {6, 7, 10}, {7, 9}, {10}}
	if len(parents) != len(names[modelNode]) {
		return 0, fmt.Errorf("invalid USA categorical parent count")
	}
	model, err := BytesComponent("model", w.model.fingerprints[modelNode][:])
	if err != nil {
		return 0, err
	}
	components := []Component{model}
	row := 0
	for i, parent := range parents {
		if parent.name != names[modelNode][i] || parent.value < 0 || parent.value >= sizes[modelNode][i] {
			return 0, fmt.Errorf("invalid USA categorical parent name or ordinal")
		}
		row = row*sizes[modelNode][i] + parent.value
		component, err := Uint64Component(parent.name, uint64(parent.value))
		if err != nil {
			return 0, err
		}
		components = append(components, component)
	}
	table := w.model.tables[modelNode]
	if table.width <= 0 || row >= len(table.cumulative)/table.width {
		return 0, fmt.Errorf("missing USA categorical row")
	}
	cumulative := table.cumulative[row*table.width : (row+1)*table.width]
	total := cumulative[len(cumulative)-1]
	if total == 0 {
		return 0, fmt.Errorf("USA categorical row has zero total")
	}
	node := usaPersonalCategoricalNodes[modelNode]
	rng, err := w.base.streams.entityStream("person", uint64(record.coordinate),
		usaSocioeconomicPaths[node], components...)
	if err != nil {
		return 0, err
	}
	before := rng.state
	draw, err := rng.Uint64n(total)
	if err != nil {
		return 0, err
	}
	position := socioeconomicPosition(cumulative, draw)
	if position == len(cumulative) {
		return 0, fmt.Errorf("USA categorical draw has no interval")
	}
	record.traces = append(record.traces, usaPersonalCategoricalTrace{
		node: node, row: row, parents: slices.Clone(parents), position: position,
		before: before, after: rng.state, draw: draw, total: total,
	})

	return position, nil
}

func (w *usaPersonalCategoricalHarness) companyByID(id string) (Company, error) {
	index, err := entityIndexFromID(w.base.namespace, "company", id)
	if err != nil {
		return Company{}, err
	}
	return w.companies.company(CompanyIndex(index))
}

// employer calls the adopted real consumer. Its trace replays that same
// fresh stream for test observation, without advancing any retained RNG.
func (w *usaPersonalCategoricalHarness) employer(record *usaPersonalCategoricalRecord, family, level int) error {

	if record.activity != PrimaryActivityEmployee || family < 0 || family >= 10 || level < 0 || level >= 4 {
		return fmt.Errorf("USA Employer node is not applicable or has invalid parents")
	}
	facts, err := w.companies.employee(record.coordinate, socioeconomicJobFamilies[family], socioeconomicJobLevels[level])
	if err != nil {
		return err
	}
	company, err := w.companies.company(facts.Employer)
	if err != nil {
		return err
	}
	parents := []socioeconomicParent{{"job-family", family}, {"job-level", level}}
	rng, err := w.companies.sampler.stream(uint64(record.coordinate), socioeconomicEmployer, parents...)
	if err != nil {
		return err
	}
	condition := family*4 + level
	index := &w.companies.sampler.employers[condition]
	total := index.cumulative[len(index.cumulative)-1]
	before := rng.state
	draw, err := rng.Uint64n(total)
	if err != nil {
		return err
	}
	position := socioeconomicPosition(index.cumulative, draw)
	if position >= len(index.companies) || index.companies[position] != facts.Employer {
		return fmt.Errorf("USA Employer replay differs from the adopted consumer")
	}
	record.employee, record.company = &facts, &company
	record.traces = append(record.traces, usaPersonalCategoricalTrace{
		node: socioeconomicEmployer, row: condition, parents: parents, position: position,
		before: before, after: rng.state, draw: draw, total: total,
	})

	return nil
}

func (w *usaPersonalCategoricalHarness) person(index PersonIndex) (usaPersonalCategoricalRecord, error) {

	if w == nil || w.base == nil || w.model == nil || w.model.identity == ([32]byte{}) || w.companies == nil {
		return usaPersonalCategoricalRecord{}, fmt.Errorf("initialized USA personal categorical harness is required")
	}
	root, err := w.base.person(index)
	if err != nil {
		return usaPersonalCategoricalRecord{}, err
	}
	if root.anchorAge < 21 || root.anchorAge > 75 || root.divisionOrdinal >= 9 {
		return usaPersonalCategoricalRecord{}, fmt.Errorf("invalid USA Layer 1 age or Division")
	}
	record := usaPersonalCategoricalRecord{
		coordinate: index, layer1: root,
		replay: [4][32]byte{w.base.base.digest, w.model.identity, w.companies.model.identity, w.base.streams.root},
	}
	ageOffset, division := root.anchorAge-21, int(root.divisionOrdinal)
	education, err := w.category(&record, 0,
		socioeconomicParent{"age-bucket", int(w.model.ages[0][ageOffset])}, socioeconomicParent{"region", division})
	if err != nil {
		return usaPersonalCategoricalRecord{}, err
	}
	record.education = socioeconomicEducationLevels[education]
	activity, err := w.category(&record, 1,
		socioeconomicParent{"age-bucket", int(w.model.ages[1][ageOffset])},
		socioeconomicParent{"education", education}, socioeconomicParent{"region", division})
	if err != nil {
		return usaPersonalCategoricalRecord{}, err
	}
	record.activity = socioeconomicActivities[activity]

	switch record.activity {

	case PrimaryActivityEmployee:

		family, err := w.category(&record, 2,
			socioeconomicParent{"education", education}, socioeconomicParent{"region", division})
		if err != nil {
			return usaPersonalCategoricalRecord{}, err
		}
		level, err := w.category(&record, 3,
			socioeconomicParent{"age-bucket", int(w.model.ages[2][ageOffset])},
			socioeconomicParent{"education", education}, socioeconomicParent{"job-family", family})
		if err != nil {
			return usaPersonalCategoricalRecord{}, err
		}
		err = w.employer(&record, family, level)
		if err != nil {
			return usaPersonalCategoricalRecord{}, err
		}

	case PrimaryActivitySelfEmployed:

		family, err := w.category(&record, 4,
			socioeconomicParent{"education", education}, socioeconomicParent{"region", division})
		if err != nil {
			return usaPersonalCategoricalRecord{}, err
		}
		industry, err := w.category(&record, 5, socioeconomicParent{"job-family", family})
		if err != nil {
			return usaPersonalCategoricalRecord{}, err
		}
		record.selfEmployed = &SelfEmployedFacts{
			JobFamily: socioeconomicJobFamilies[family], Industry: socioeconomicIndustries[industry],
		}

	case PrimaryActivityUnemployed, PrimaryActivityStudent, PrimaryActivityRetired, PrimaryActivityInactive:
	default:
		return usaPersonalCategoricalRecord{}, fmt.Errorf("invalid USA PrimaryActivity")

	}

	return record, nil
}

type usaPersonalCategoricalRecord struct {
	coordinate   PersonIndex
	layer1       usaLayer1Person
	education    EducationLevel
	activity     PrimaryActivity
	employee     *EmployeeFacts
	selfEmployed *SelfEmployedFacts
	company      *Company
	replay       [4][32]byte
	traces       []usaPersonalCategoricalTrace
}

type usaPersonalCategoricalTrace struct {
	node     int
	row      int
	parents  []socioeconomicParent
	position int
	before   uint64
	after    uint64
	draw     uint64
	total    uint64
}
