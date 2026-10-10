// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

// TestSocioeconomicBranchNonEvaluation poisons inapplicable tables and indexes.
// Touching an inactive branch would fail instead of being silently discarded.
func TestSocioeconomicBranchNonEvaluation(t *testing.T) {
	for branch, activity := range socioeconomicActivities {
		t.Run(string(activity), func(t *testing.T) {
			base := mustLayer1World(t, PopulationAnchorDate, layer1TestSeed)
			model := socioeconomicTestModel(t, func(document map[string]any) {
				setSocioeconomicTestCategory(document, "primaryActivity", 6, func(int) int { return branch })
			})
			world := mustSocioeconomicWorld(t, base, model)
			before := socioeconomicFactsAt(t, world, 42)
			for other := range model.income {
				if other != branch {
					model.income[other] = socioeconomicIncome{}
				}
			}
			if branch != 0 {
				model.employeeJobFamily = socioeconomicWeights{}
				model.employeeJobLevel = socioeconomicWeights{}
				world.employers = [40]socioeconomicEmployerIndex{}
				world.companyCounts = nil
				world.companyIndustries = nil
			}
			if branch != 1 {
				model.selfEmployedJobFamily = socioeconomicWeights{}
				model.selfEmployedIndustry = socioeconomicWeights{}
			}
			// These are internal test tripwires, not supported caller mutations.
			data := *base.data
			data.firstNames, data.firstNameWeights = nil, nil
			data.lastNames, data.lastNameWeights = nil, nil
			data.postalCodes, data.streetNames, data.streetNameWeights = nil, nil, nil
			base.data = &data
			base.faceCatalog = nil
			if after := socioeconomicFactsAt(t, world, 42); !reflect.DeepEqual(after, before) {
				t.Fatalf("expected unchanged Facts without inactive or presentation inputs, got %+v/%+v", before, after)
			}
		})
	}
}

// TestSocioeconomicDAGAddresses checks runtime draws at varied real Layer 1
// inputs using literal protocol dependencies and the existing raw-weight
// sampler.
func TestSocioeconomicDAGAddresses(t *testing.T) {
	base := mustLayer1World(t, "2030-12-31", layer1TestSeed)
	m := socioeconomicTestModel(t, nil)
	w := mustSocioeconomicWorld(t, base, m)
	seen := map[PrimaryActivity]bool{}
	for index := PersonIndex(1); index <= 400; index++ {
		anchorAge, region := socioeconomicSemanticInputs(t, base, index)
		facts := socioeconomicFactsAt(t, w, index)
		seen[facts.PrimaryActivity] = true
		bucket := func(domain int) int { return int(m.ages[domain][anchorAge-21]) }
		u := func(name string, value int) Component { return mustUint64Component(t, name, uint64(value)) }
		category := func(node int, path string, table socioeconomicWeights, row int, parents ...Component) int {
			weights := socioeconomicRawWeights(table.cumulative[row*table.width : (row+1)*table.width])
			return socioeconomicReferenceSelection(
				t, base, "person", uint64(index), path, m.fingerprints[node], weights, parents...)
		}
		education := category(2, "socioeconomic/v1/person/education", m.education, bucket(0)*20+region,
			u("age-bucket", bucket(0)), u("region", region))
		activity := category(3, "socioeconomic/v1/person/primary-activity", m.primaryActivity,
			(bucket(1)*7+education)*20+region,
			u("age-bucket", bucket(1)), u("education", education), u("region", region))
		if facts.Education != socioeconomicEducationLevels[education] ||
			facts.PrimaryActivity != socioeconomicActivities[activity] {
			t.Fatalf("expected reference Education/activity %d/%d, got %+v", education, activity, facts)
		}
		var tuple int
		var parents []Component
		switch activity {
		case 0:
			family := category(4, "socioeconomic/v1/person/employee/job-family",
				m.employeeJobFamily, education*20+region,
				u("education", education), u("region", region))
			level := category(5, "socioeconomic/v1/person/employee/job-level",
				m.employeeJobLevel, (bucket(2)*7+education)*10+family,
				u("age-bucket", bucket(2)), u("education", education), u("job-family", family))
			weights := socioeconomicReferenceEmployerWeights(t, w, family, level)
			employer := socioeconomicReferenceSelection(t, base, "person", uint64(index),
				"socioeconomic/v1/person/employee/employer", m.fingerprints[6], weights,
				u("job-family", family), u("job-level", level))
			if facts.Employee == nil || facts.Employee.JobFamily != socioeconomicJobFamilies[family] ||
				facts.Employee.JobLevel != socioeconomicJobLevels[level] ||
				facts.Employee.Employer != CompanyIndex(employer+1) {
				t.Fatalf("expected reference employee %d/%d/%d, got %+v", family, level, employer+1, facts.Employee)
			}
			company := socioeconomicCompanyAt(t, w, CompanyIndex(employer+1))
			industry := slices.Index(socioeconomicIndustries[:], company.Industry)
			band := slices.Index(socioeconomicSizeBands[:], company.SizeBand)
			tuple = socioeconomicTestTuple(
				[]int{region, family, level, industry, band}, []int{20, 10, 4, 14, 5})
			parents = []Component{
				u("region", region), u("job-family", family), u("job-level", level),
				u("industry", industry), u("size-band", band),
			}
		case 1:
			family := category(7, "socioeconomic/v1/person/self-employed/job-family",
				m.selfEmployedJobFamily, education*20+region,
				u("education", education), u("region", region))
			industry := category(8, "socioeconomic/v1/person/self-employed/industry",
				m.selfEmployedIndustry, family, u("job-family", family))
			if facts.SelfEmployed == nil || facts.SelfEmployed.JobFamily != socioeconomicJobFamilies[family] ||
				facts.SelfEmployed.Industry != socioeconomicIndustries[industry] {
				t.Fatalf("expected reference self-employment %d/%d, got %+v", family, industry, facts.SelfEmployed)
			}
			tuple = socioeconomicTestTuple([]int{bucket(3), region, family, industry}, []int{2, 20, 10, 14})
			parents = []Component{
				u("age-bucket", bucket(3)), u("region", region), u("job-family", family), u("industry", industry),
			}
		default:
			tuple = socioeconomicTestTuple([]int{bucket(activity + 2), education, region}, []int{2, 7, 20})
			parents = []Component{u("age-bucket", bucket(activity+2)), u("education", education), u("region", region)}
		}
		income := m.income[activity]
		law := income.laws[income.tupleMap[tuple]]
		position := socioeconomicReferenceSelection(t, base, "person", uint64(index),
			"socioeconomic/v1/person/income/"+string(facts.PrimaryActivity),
			m.fingerprints[9+activity], socioeconomicRawWeights(law.cumulative), parents...)
		if facts.Income != AnnualIncomeEUR(law.values[position]) {
			t.Fatalf("expected reference Income %d, got %d", law.values[position], facts.Income)
		}
	}
	if len(seen) != 6 {
		t.Fatalf("expected all six branches exercised, got %v", seen)
	}
}

// TestSocioeconomicPersonBranches makes categorical rows and Income laws
// distinguishable by their semantic coordinates, including every own age map.
func TestSocioeconomicPersonBranches(t *testing.T) {
	base := mustLayer1World(t, "2030-12-31", layer1TestSeed)
	for branch, activity := range socioeconomicActivities {
		t.Run(string(activity), func(t *testing.T) {
			model := socioeconomicTestModel(t, func(document map[string]any) {
				setSocioeconomicTestCategory(document, "education", 7, func(row int) int {
					return (row/20 + row%20) % 7
				})
				setSocioeconomicTestCategory(document, "primaryActivity", 6, func(int) int { return branch })
				setSocioeconomicTestCategory(document, "employeeJobFamily", 10, func(row int) int {
					return (row/20 + row%20) % 10
				})
				setSocioeconomicTestCategory(document, "employeeJobLevel", 4, func(row int) int {
					return (row/70 + row/10%7 + row%10) % 4
				})
				setSocioeconomicTestCategory(document, "selfEmployedJobFamily", 10, func(row int) int {
					return (2*(row/20) + row%20) % 10
				})
				setSocioeconomicTestCategory(document, "selfEmployedIndustry", 14, func(row int) int {
					return row * 3 % 14
				})
				income := document["income"].(map[string]any)[socioeconomicBranchNames[branch]].(map[string]any)
				laws := income["laws"].([]any)
				for law := range laws {
					value := uint32((branch+1)*10000 + law*100)
					if law == 0 {
						value = 0
					}
					laws[law] = map[string]any{"values": []uint32{value}, "weights": []uint32{1}}
				}
			})
			world := mustSocioeconomicWorld(t, base, model)
			var sawZero, sawPositive bool
			for coordinate := PersonIndex(1); coordinate <= 256; coordinate++ {
				age, region := socioeconomicSemanticInputs(t, base, coordinate)
				facts := socioeconomicFactsAt(t, world, coordinate)
				education := (socioeconomicTestAgeBucket(age, 35) + region) % 7
				if facts.Education != socioeconomicEducationLevels[education] || facts.PrimaryActivity != activity {
					t.Fatalf("expected Education/activity %s/%s, got %+v",
						socioeconomicEducationLevels[education], activity, facts)
				}
				var tuple int
				switch branch {
				case 0:
					family := (education + region) % 10
					level := (socioeconomicTestAgeBucket(age, 45) + education + family) % 4
					if facts.Employee == nil || facts.SelfEmployed != nil ||
						facts.Employee.JobFamily != socioeconomicJobFamilies[family] ||
						facts.Employee.JobLevel != socioeconomicJobLevels[level] {
						t.Fatalf("expected employee occupation %d/%d, got %+v", family, level, facts)
					}
					company := socioeconomicCompanyAt(t, world, facts.Employee.Employer)
					if company.EmployeeCount == 0 {
						t.Fatalf("expected positive-workforce Employer, got %+v", company)
					}
					industry := slices.Index(socioeconomicIndustries[:], company.Industry)
					band := slices.Index(socioeconomicSizeBands[:], socioeconomicExpectedBand(company.EmployeeCount))
					tuple = socioeconomicTestTuple(
						[]int{region, family, level, industry, band}, []int{20, 10, 4, 14, 5})
				case 1:
					family := (2*education + region) % 10
					industry := family * 3 % 14
					if facts.SelfEmployed == nil || facts.Employee != nil ||
						facts.SelfEmployed.JobFamily != socioeconomicJobFamilies[family] ||
						facts.SelfEmployed.Industry != socioeconomicIndustries[industry] {
						t.Fatalf("expected self-employment %d/%d, got %+v", family, industry, facts)
					}
					tuple = socioeconomicTestTuple(
						[]int{socioeconomicTestAgeBucket(age, 30), region, family, industry}, []int{2, 20, 10, 14})
				default:
					if facts.Employee != nil || facts.SelfEmployed != nil {
						t.Fatalf("expected no occupation in %s, got %+v", activity, facts)
					}
					threshold := []int{50, 32, 60, 55}[branch-2]
					tuple = socioeconomicTestTuple(
						[]int{socioeconomicTestAgeBucket(age, threshold), education, region}, []int{2, 7, 20})
				}
				law := (tuple/3 + tuple/17 + tuple/127) % 5
				expected := AnnualIncomeEUR((branch+1)*10000 + law*100)
				if law == 0 {
					expected = 0
				}
				if facts.Income != expected {
					t.Fatalf("expected branch %d tuple %d Income %d, got %d", branch, tuple, expected, facts.Income)
				}
				sawZero = sawZero || facts.Income == 0
				sawPositive = sawPositive || facts.Income > 0
			}
			if !sawZero || !sawPositive {
				t.Fatalf("expected ordinary zero and positive Income in %s, got %t/%t", activity, sawZero, sawPositive)
			}
		})
	}
}

func socioeconomicSemanticInputs(t *testing.T, base *World, index PersonIndex) (int, int) {
	t.Helper()
	person, err := base.Person(index)
	if err != nil {
		t.Fatalf("expected Layer 1 reference Person, got %v", err)
	}
	birth, err := time.Parse(time.DateOnly, person.BirthDate)
	if err != nil {
		t.Fatalf("expected reference birth date, got %v", err)
	}
	age := ageAt(birth, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	for _, place := range base.data.places {
		if place.municipality == person.Address.Municipality && place.stateProv == person.Address.StateProv {
			region := slices.Index(socioeconomicRegions[:], place.region)
			if region < 0 {
				t.Fatalf("expected canonical Region, got %q", place.region)
			}
			return age, region
		}
	}
	t.Fatalf("expected Layer 1 municipality, got %+v", person.Address)
	return 0, 0
}

func socioeconomicTestAgeBucket(age, boundary int) int {
	if age >= boundary {
		return 1
	}
	return 0
}

func socioeconomicTestTuple(values, dimensions []int) int {
	index := 0
	for i, value := range values {
		index = index*dimensions[i] + value
	}
	return index
}
