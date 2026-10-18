// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/bits"
	"reflect"
	"slices"
	"sort"
	"testing"
	"time"
)

const usaPersonalCategoricalReplaySHA256 = "170e2b5e514fccea030fb7a53f0ac03da63a7d646a0fd1c7d1553770368da04a"

type usaPersonalCategoricalOracle struct {
	input      [6]internationalCategorical
	employer   [40][]uint64
	conditions [40]usaPersonalEmployerObservation
}

func newUSAPersonalCategoricalOracle(t *testing.T, w *usaPersonalCategoricalHarness) *usaPersonalCategoricalOracle {

	t.Helper()
	oracle := &usaPersonalCategoricalOracle{input: readUSAPersonalCategoricalInput(t, readUSAPersonalCategoricalAssets(t))}
	var factors struct {
		Industry [][]uint64 `json:"industryFactors"`
		Size     [][]uint64 `json:"sizeFactors"`
	}
	err := json.Unmarshal(readUSACompanyEmployerAssets(t)[5], &factors)
	if err != nil {
		t.Fatalf("expected source Employer factors, got %v", err)
	}
	for condition := range 40 {

		family, level := condition/4, condition%4
		oracle.employer[condition] = make([]uint64, 65536)
		out := &oracle.conditions[condition]
		out.Condition = condition
		for offset, count := range w.companies.sampler.companyCounts {

			industry := int(w.companies.sampler.companyIndustries[offset])
			band := slices.Index(socioeconomicSizeBands[:], socioeconomicExpectedBand(count))
			high, weight := bits.Mul64(min(uint64(count), 65535), factors.Industry[family][industry])
			if high != 0 {
				t.Fatal("expected bounded source Industry product, got overflow")
			}
			high, weight = bits.Mul64(weight, factors.Size[level][band-1])
			if high != 0 || weight == 0 {
				t.Fatal("expected positive bounded source Employer product, got overflow or zero")
			}
			var carry uint64
			out.Total, carry = bits.Add64(out.Total, weight, 0)
			if carry != 0 {
				t.Fatal("expected bounded source Employer total, got overflow")
			}
			oracle.employer[condition][offset] = out.Total
			out.IndustryWeights[industry] += weight
			out.SizeBandWeights[band] += weight
			actual := &w.companies.sampler.employers[condition]
			if actual.companies[offset] != CompanyIndex(offset+1) || actual.cumulative[offset] != out.Total {
				t.Fatalf("expected source-derived Employer interval at %d/%d, got mismatch", condition, offset)
			}

		}

	}

	return oracle
}

// check verifies each actual categorical value, parent, interval and RNG state
// against the original integer masses and the existing stream factory.
func (o *usaPersonalCategoricalOracle) check(t *testing.T, w *usaPersonalCategoricalHarness, r usaPersonalCategoricalRecord) {

	t.Helper()
	root := r.layer1
	var locality *usaLocation
	for i := range w.base.base.geography.locations {
		if w.base.base.geography.locations[i].id == root.localityID {
			locality = &w.base.base.geography.locations[i]
			break
		}
	}
	if locality == nil || root.person.Address.Country != "US" || root.person.Address.StateProv != locality.state.USPS ||
		root.division != locality.state.Division || root.divisionOrdinal != locality.state.DivisionOrdinal ||
		root.person.Address.Municipality != locality.name || !slices.Contains(locality.zips, root.person.Address.PostalCode) {
		t.Fatalf("expected one coherent selected USA locality/ZIP/state/Division, got %+v", root)
	}
	birth, err := time.Parse(time.DateOnly, root.person.BirthDate)
	if err != nil {
		t.Fatalf("expected valid birth date, got %v", err)
	}
	cohort, err := cohortForYear(birth.Year())
	if err != nil {
		t.Fatalf("expected applicable birth cohort, got %v", err)
	}
	if root.anchorAge != root.person.CurrentAge || root.cohort != cohort || root.person.PhotoID != "" {
		t.Fatal("expected adopted anchor age, cohort and unlinked photo, got different Layer 1 facts")
	}
	education := slices.Index(socioeconomicEducationLevels[:], r.education)
	activity := slices.Index(socioeconomicActivities[:], r.activity)
	if education < 0 || education == 4 || activity < 0 {
		t.Fatal("expected supported frozen Education/Activity, got invalid ordinal or structural zero")
	}
	var age [3]int
	for consumer, bounds := range [3][]int{
		{21, 25, 35, 45, 55, 65}, {21, 25, 35, 45, 55, 60, 65, 70}, {21, 25, 35, 45, 55, 65},
	} {
		for bucket, lower := range bounds {
			if root.anchorAge >= lower {
				age[consumer] = bucket
			}
		}
	}
	division := int(root.divisionOrdinal)
	nodes := []int{socioeconomicEducation, socioeconomicPrimaryActivity}
	positions := []int{education, activity}
	family, level := -1, -1
	switch r.activity {

	case PrimaryActivityEmployee:

		if r.employee == nil || r.selfEmployed != nil || r.company == nil {
			t.Fatal("expected Employee and relational Company only, got invalid branch presence")
		}
		family = slices.Index(socioeconomicJobFamilies[:], r.employee.JobFamily)
		level = slices.Index(socioeconomicJobLevels[:], r.employee.JobLevel)
		if family < 0 || level < 0 || r.employee.Employer == 0 || r.employee.Employer > 65536 {
			t.Fatal("expected frozen working ordinals and operational Employer, got invalid Employee")
		}
		company, err := w.companyByID(r.company.ID)
		if err != nil {
			t.Fatalf("expected selected operational CompanyID lookup, got %v", err)
		}
		industry := w.companies.sampler.companyIndustries[r.employee.Employer-1]
		if company != *r.company || company.Industry != socioeconomicIndustries[industry] ||
			company.EmployeeCount != w.companies.sampler.companyCounts[r.employee.Employer-1] ||
			company.SizeBand != socioeconomicExpectedBand(company.EmployeeCount) {
			t.Fatal("expected actual Employer Industry/count/derived SizeBand, got different Company facts")
		}
		decoded, err := entityIndexFromID(w.base.namespace, "company", company.ID)
		if err != nil {
			t.Fatalf("expected reversible CompanyID, got %v", err)
		}
		if decoded != uint64(r.employee.Employer) {
			t.Fatalf("expected Company coordinate %d, got %d", r.employee.Employer, decoded)
		}
		nodes = append(nodes, socioeconomicEmployeeJobFamily, socioeconomicEmployeeJobLevel, socioeconomicEmployer)
		positions = append(positions, family, level, int(r.employee.Employer-1))

	case PrimaryActivitySelfEmployed:

		if r.employee != nil || r.company != nil || r.selfEmployed == nil {
			t.Fatal("expected SelfEmployed without Employer/JobLevel/Company, got invalid branch presence")
		}
		family = slices.Index(socioeconomicJobFamilies[:], r.selfEmployed.JobFamily)
		industry := slices.Index(socioeconomicIndustries[:], r.selfEmployed.Industry)
		if family < 0 || industry < 0 {
			t.Fatal("expected frozen SelfEmployed ordinals, got invalid value")
		}
		nodes = append(nodes, socioeconomicSelfEmployedJobFamily, socioeconomicSelfEmployedIndustry)
		positions = append(positions, family, industry)

	default:
		if r.employee != nil || r.selfEmployed != nil || r.company != nil {
			t.Fatal("expected no nonworking professional facts, got working payload")
		}

	}
	if len(r.traces) != len(nodes) {
		t.Fatalf("expected exactly applicable node evaluations %v, got %d", nodes, len(r.traces))
	}
	for step, node := range nodes {

		trace := r.traces[step]
		var parents []socioeconomicParent
		var cumulative []uint64
		row := 0
		modelNode := slices.Index(usaPersonalCategoricalNodes[:], node)
		fingerprint := w.companies.model.fingerprints[socioeconomicEmployer]
		if modelNode >= 0 {
			fingerprint = w.model.fingerprints[modelNode]
		}
		switch node {
		case socioeconomicEducation:
			row = age[0]*9 + division
			parents = []socioeconomicParent{{"age-bucket", age[0]}, {"region", division}}
		case socioeconomicPrimaryActivity:
			row = (age[1]*7+education)*9 + division
			parents = []socioeconomicParent{{"age-bucket", age[1]}, {"education", education}, {"region", division}}
		case socioeconomicEmployeeJobFamily, socioeconomicSelfEmployedJobFamily:
			row = education*9 + division
			parents = []socioeconomicParent{{"education", education}, {"region", division}}
		case socioeconomicEmployeeJobLevel:
			row = (age[2]*7+education)*10 + family
			parents = []socioeconomicParent{{"age-bucket", age[2]}, {"education", education}, {"job-family", family}}
		case socioeconomicSelfEmployedIndustry:
			row = family
			parents = []socioeconomicParent{{"job-family", family}}
		case socioeconomicEmployer:
			row = family*4 + level
			parents = []socioeconomicParent{{"job-family", family}, {"job-level", level}}
			cumulative = o.employer[row]
		}
		if modelNode >= 0 {
			width := w.model.tables[modelNode].width
			var total uint64
			for _, weight := range o.input[modelNode].Weights[row*width : (row+1)*width] {
				total += uint64(weight)
				cumulative = append(cumulative, total)
			}
		}
		components := []Component{mustBytesComponent(t, "model", fingerprint[:])}
		for _, parent := range parents {
			components = append(components, mustUint64Component(t, parent.name, uint64(parent.value)))
		}
		rng, err := w.base.streams.entityStream("person", uint64(r.coordinate), usaSocioeconomicPaths[node], components...)
		if err != nil {
			t.Fatalf("expected direct factory stream, got %v", err)
		}
		if trace.node != node || trace.row != row || !reflect.DeepEqual(trace.parents, parents) || trace.before != rng.state {
			t.Fatalf("expected exact DAG node, parents, row and initial RNG at %d/%d, got %+v", r.coordinate, node, trace)
		}
		total := cumulative[len(cumulative)-1]
		draw, err := rng.Uint64n(total)
		if err != nil {
			t.Fatalf("expected bounded reference draw, got %v", err)
		}
		position := sort.Search(len(cumulative), func(i int) bool { return draw < cumulative[i] })
		if trace.total != total || trace.draw != draw || trace.after != rng.state ||
			trace.position != position || positions[step] != position {
			t.Fatalf("expected exact value, interval and final RNG at %d/%d, got %+v", r.coordinate, node, trace)
		}
		start := uint64(0)
		if position > 0 {
			start = cumulative[position-1]
		}
		if draw < start || draw >= cumulative[position] || cumulative[position] == start {
			t.Fatal("expected positive selected half-open interval, got invalid interval")
		}

	}

}

type usaPersonalEmployerObservation struct {
	Condition        int        `json:"condition"`
	Total            uint64     `json:"total"`
	IndustryWeights  [14]uint64 `json:"industryWeights"`
	SizeBandWeights  [5]uint64  `json:"sizeBandWeights"`
	Selected         int        `json:"selected"`
	ObservedIndustry [14]int    `json:"observedIndustry"`
	ObservedSizeBand [5]int     `json:"observedSizeBand"`
}

// TestUSAPersonalCategoricalJoinReplay generates actual coordinates rather than
// drawing a second root from D0. Every comparison is keyed by raw coordinate.
func TestUSAPersonalCategoricalJoinReplay(t *testing.T) {

	a := readUSAPersonalCategoricalHarness(t)
	oracle := newUSAPersonalCategoricalOracle(t, a)
	var branch [6]int
	var education [7]int
	var division [9]int
	var rootCounts [55][9]int
	var employeeFamily, selfEmployedFamily [10]int
	var jobLevel [4]int
	var employeeIndustry, selfEmployedIndustry [14]int
	var sizeBand [5]int
	var observed [6][][]int
	for i, table := range a.model.tables {
		observed[i] = make([][]int, len(table.cumulative)/table.width)
		for row := range observed[i] {
			observed[i][row] = make([]int, table.width)
		}
	}
	companies := map[CompanyIndex]struct{}{}
	first := map[PersonIndex][32]byte{}
	examples := map[PrimaryActivity]json.RawMessage{}
	canonical := sha256.New()
	canonicalBytes := 0
	for coordinate := PersonIndex(1); coordinate <= 20000; coordinate++ {

		r, err := a.person(coordinate)
		if err != nil {
			t.Fatalf("expected valid USA coordinate %d, got %v", coordinate, err)
		}
		oracle.check(t, a, r)
		encoded := usaPersonalCategoricalTuple(t, r)
		first[coordinate] = sha256.Sum256(encoded)
		_, _ = canonical.Write(encoded)
		canonicalBytes += len(encoded)
		branch[slices.Index(socioeconomicActivities[:], r.activity)]++
		education[slices.Index(socioeconomicEducationLevels[:], r.education)]++
		division[r.layer1.divisionOrdinal]++
		rootCounts[r.layer1.anchorAge-21][r.layer1.divisionOrdinal]++
		for _, trace := range r.traces {
			if node := slices.Index(usaPersonalCategoricalNodes[:], trace.node); node >= 0 {
				observed[node][trace.row][trace.position]++
			}
		}
		if _, ok := examples[r.activity]; !ok {
			examples[r.activity] = encoded
		}
		if r.employee != nil {
			family := slices.Index(socioeconomicJobFamilies[:], r.employee.JobFamily)
			level := slices.Index(socioeconomicJobLevels[:], r.employee.JobLevel)
			industry := slices.Index(socioeconomicIndustries[:], r.company.Industry)
			band := slices.Index(socioeconomicSizeBands[:], r.company.SizeBand)
			employeeFamily[family]++
			jobLevel[level]++
			employeeIndustry[industry]++
			sizeBand[band]++
			companies[r.employee.Employer] = struct{}{}
			condition := &oracle.conditions[family*4+level]
			condition.Selected++
			condition.ObservedIndustry[industry]++
			condition.ObservedSizeBand[band]++
		}
		if r.selfEmployed != nil {
			selfEmployedFamily[slices.Index(socioeconomicJobFamilies[:], r.selfEmployed.JobFamily)]++
			selfEmployedIndustry[slices.Index(socioeconomicIndustries[:], r.selfEmployed.Industry)]++
		}

	}
	for _, count := range branch {
		if count == 0 {
			t.Fatal("expected all six real branches covered, got missing branch")
		}
	}
	for _, condition := range oracle.conditions {
		if condition.Selected == 0 {
			t.Fatalf("expected all 40 Employer conditions covered, got no Employee in condition %d", condition.Condition)
		}
	}
	for i, count := range education {
		if (count == 0) != (i == 4) {
			t.Fatalf("expected six supported Education and intentional E=4 zero, got %v", education)
		}
	}
	for _, coverage := range [][]int{employeeFamily[:], selfEmployedFamily[:], jobLevel[:], employeeIndustry[:],
		selfEmployedIndustry[:], sizeBand[1:], division[:]} {
		if slices.Contains(coverage, 0) {
			t.Fatalf("expected complete applicable category coverage, got %v", coverage)
		}
	}

	// B reacquires all actual inputs and rebuilds all 65,536 Companies/indexes.
	b := readUSAPersonalCategoricalHarness(t)
	for coordinate := PersonIndex(20000); coordinate > 0; coordinate-- {
		r, err := b.person(coordinate)
		if err != nil {
			t.Fatalf("expected descending USA replay, got %v", err)
		}
		if got := sha256.Sum256(usaPersonalCategoricalTuple(t, r)); got != first[coordinate] {
			t.Fatalf("expected per-coordinate A/B equality at %d, got %x/%x", coordinate, first[coordinate], got)
		}
	}

	var sparse []PersonIndex
	sparseHash := sha256.New()
	for i := 1; i <= 1000; i++ {
		coordinate := PersonIndex(20001 + uint64(i)*(uint64(Layer1MaxPersonIndex)-20001)/1000)
		sparse = append(sparse, coordinate)
		r, err := a.person(coordinate)
		if err != nil {
			t.Fatalf("expected sparse coordinate, got %v", err)
		}
		oracle.check(t, a, r)
		encoded := usaPersonalCategoricalTuple(t, r)
		first[coordinate] = sha256.Sum256(encoded)
		_, _ = sparseHash.Write(encoded)
	}
	for i := range 1000 {
		coordinate := sparse[(i*337)%1000]
		r, err := b.person(coordinate)
		if err != nil {
			t.Fatalf("expected permuted sparse replay, got %v", err)
		}
		if sha256.Sum256(usaPersonalCategoricalTuple(t, r)) != first[coordinate] {
			t.Fatalf("expected sparse per-coordinate replay at %d, got mismatch", coordinate)
		}
	}
	boundaries := []PersonIndex{1, 20000, Layer1MaxPersonIndex - 1, Layer1MaxPersonIndex}
	for _, coordinate := range boundaries {
		left, err := a.person(coordinate)
		if err != nil {
			t.Fatalf("expected boundary coordinate, got %v", err)
		}
		right, err := b.person(coordinate)
		if err != nil {
			t.Fatalf("expected replayed boundary coordinate, got %v", err)
		}
		oracle.check(t, a, left)
		if !reflect.DeepEqual(left, right) {
			t.Fatalf("expected complete boundary equality at %d, got mismatch", coordinate)
		}
	}

	// Regenerate, hash and compare the complete roster in opposite orders.
	rosterHash := sha256.New()
	var workforce uint64
	for coordinate := CompanyIndex(1); coordinate <= 65536; coordinate++ {
		company := usaCompanyEmployerAt(t, a.companies, coordinate)
		reverse := usaCompanyEmployerAt(t, b.companies, 65537-coordinate)
		other := 65537 - coordinate
		if reverse.EmployeeCount != a.companies.sampler.companyCounts[other-1] ||
			reverse.Industry != socioeconomicIndustries[a.companies.sampler.companyIndustries[other-1]] {
			t.Fatal("expected complete opposite-order Company replay, got mismatch")
		}
		if company.SizeBand != socioeconomicExpectedBand(company.EmployeeCount) {
			t.Fatal("expected deterministic Company SizeBand, got mismatch")
		}
		_, _ = fmt.Fprintf(rosterHash, "%d\t%s\t%s\t%d\t%s\n",
			coordinate, company.ID, company.Industry, company.EmployeeCount, company.SizeBand)
		workforce += uint64(company.EmployeeCount)
	}
	if got := fmt.Sprintf("%x", rosterHash.Sum(nil)); got != usaCompanyRosterSHA256 {
		t.Fatalf("expected historical roster %s, got %s", usaCompanyRosterSHA256, got)
	}
	digest := fmt.Sprintf("%x", canonical.Sum(nil))
	if digest != usaPersonalCategoricalReplaySHA256 {
		t.Fatalf("expected technical replay %s, got %s", usaPersonalCategoricalReplaySHA256, digest)
	}
	t.Logf("prefix=20000 sha256=%s branches=%v companies=65536 distinct-employers=%d", digest, branch, len(companies))
	writeUSAPersonalCategoricalEvidence(t, "join-replay.json", map[string]any{
		"status": "PASS", "seed": 2026, "namespace": a.base.namespace, "generation": 1,
		"prefixCoordinates": 20000, "prefixSHA256": digest, "prefixCanonicalBytes": canonicalBytes,
		"prefixOrders":      []string{"ascending-A", "descending-B-independent-acquisition"},
		"sparseCoordinates": 1000, "sparseSHA256": fmt.Sprintf("%x", sparseHash.Sum(nil)),
		"sparseOrders": []string{"ascending", "permutation-337-mod-1000"}, "boundaries": boundaries,
		"distinctValidCoordinates": 21001, "personCallsInReplayTest": 42008,
		"branchOrder": socioeconomicActivities, "branchCounts": branch,
		"educationOrder": socioeconomicEducationLevels, "educationCounts": education,
		"divisionCounts": division, "employeeFamilyCounts": employeeFamily, "selfEmployedFamilyCounts": selfEmployedFamily,
		"rootCounts": rootCounts, "layer1AgeWeights": a.base.base.ageWeights,
		"layer1DivisionWeights": a.base.base.geography.divisionMasses,
		"employeeLevelCounts":   jobLevel, "employeeIndustryCounts": employeeIndustry,
		"selfEmployedIndustryCounts": selfEmployedIndustry, "employeeSizeBandCounts": sizeBand,
		"employerSelections": branch[0], "distinctSelectedCompanies": len(companies),
		"companyCount": 65536, "companyRosterSHA256": usaCompanyRosterSHA256, "companyTotalWorkforce": workforce,
		"incomePresent": false, "photoConnected": false, "rejectedValidCoordinates": 0, "examples": examples,
	})
	writeUSAPersonalCategoricalEvidence(t, "conditional-observations.json", map[string]any{
		"status": "PASS", "personalNodeOrder": usaPersonalCategoricalNodes, "observationsByRow": observed,
		"employerConditions": oracle.conditions, "applicablePersonSampleOnly": true,
	})

}
