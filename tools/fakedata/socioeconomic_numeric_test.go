// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// TestSocioeconomicNumericInvalid exercises the internal, decoder-free boundary.
func TestSocioeconomicNumericInvalid(t *testing.T) {

	for _, test := range []struct {
		name   string
		domain socioeconomicTerritoryDomain
		input  int
		change func(*socioeconomicModelJSON)
		want   string
	}{
		{"absent domain", 0, 9, nil, "territory domain"},
		{"unauthorized domain", 10, 9, nil, "territory domain"},
		{"maximum domain", 255, 9, nil, "territory domain"},
		{"nine declared twenty", socioeconomicItalyRegions, 9, nil, "education: expected"},
		{"twenty declared nine", socioeconomicUSADivisions, 20, nil, "education: expected"},
		{"company zero", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.companyCount = 0
		}, "positive"},
		{"company overflow", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.companyCount = math.MaxUint64
		}, "companyCount exceeds"},
		{"age cardinality", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.ages[0].lowerBounds = append(d.ages[0].lowerBounds, 27)
		}, "cardinality"},
		{"age coverage", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.ages[0].compiled = nil
		}, "coverage"},
		{"age order", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.ages[0].lowerBounds[1] = 21
		}, "increasing"},
		{"age domain", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.ages[0].lowerBounds[5] = math.MaxUint32
		}, "lower bounds"},
		{"age mapping", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.ages[0].compiled[54] = 0
		}, "inconsistent bucket"},
		{"missing weights", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.education = nil
		}, "education: expected"},
		{"truncated row", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.primaryActivity = d.primaryActivity[:len(d.primaryActivity)-1]
		}, "primaryActivity: expected"},
		{"oversized row", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.employeeJobFamily = append(d.employeeJobFamily, 1)
		}, "employeeJobFamily: expected"},
		{"zero row", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			clear(d.education[:7])
		}, "zero total"},
		{"noncanonical row", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.education[0], d.education[1] = 2, 4
		}, "GCD"},
		{"missing count law", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.employeeCount = d.employeeCount[:13]
		}, "expected 14 laws"},
		{"count points", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.employeeCount[0].values = make([]uint32, 129)
		}, "128 points"},
		{"no workforce", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.employeeCount[0] = socioeconomicLawJSON{values: []uint32{0}, weights: []uint32{1}}
		}, "no positive-workforce"},
		{"cap zero", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.employer.cap = 0
		}, "positive"},
		{"factor zero", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.employer.industryFactors[0] = 0
		}, "positive"},
		{"factor range", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.employer.sizeFactors[0] = math.MaxUint16
		}, "sizeFactors"},
		{"missing inapplicable sentinel", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.income[0].tupleMap[0] = 0
		}, "sentinel 65535"},
		{"misplaced sentinel", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.income[0].tupleMap[1] = math.MaxUint16
		}, "reference"},
		{"reference narrowing overflow", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.income[0].tupleMap[1] = math.MaxUint32
		}, "reference"},
		{"first occurrence", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.income[0].tupleMap[1] = 1
		}, "noncanonical"},
		{"duplicate law", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.income[0].laws[1] = d.income[0].laws[0]
		}, "duplicate canonical laws"},
		{"unreferenced law", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {
			d.income[0].laws = append(d.income[0].laws,
				socioeconomicLawJSON{values: []uint32{0, 99}, weights: []uint32{1, 1}})
		}, "unreferenced"},
		{"retained budget", socioeconomicUSADivisions, 9, func(d *socioeconomicModelJSON) {

			values := make([]uint32, 512)
			for i := range values {
				values[i] = uint32(i)
			}
			law := socioeconomicLawJSON{values: values, weights: slices.Repeat([]uint32{1}, 512)}
			for branch, maximum := range [...]int{512, 256, 64, 64, 64, 64} {
				d.income[branch].laws = slices.Repeat([]socioeconomicLawJSON{law}, maximum)
			}

		}, "retained model charge"},
	} {
		t.Run(test.name, func(t *testing.T) {

			input := socioeconomicNumericFixture(test.input)
			if test.change != nil {
				test.change(input)
			}
			model, err := compileSocioeconomicNumericModel(input, test.domain)
			if err != nil {
				if model != nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("expected nil model and %q rejection, got %v/%v", test.want, model, err)
				}
				return
			}

			t.Fatal("expected invalid numerical input rejected, got success")

		})
	}

	t.Run("missing input", func(t *testing.T) {
		_, err := compileSocioeconomicNumericModel(nil, socioeconomicUSADivisions)
		if err != nil {
			return
		}
		t.Fatal("expected missing input rejected, got success")
	})

	for branch := range 6 {
		for _, test := range []struct {
			name   string
			change func(*socioeconomicIncomeJSON)
			want   string
		}{
			{"truncated map", func(i *socioeconomicIncomeJSON) {
				i.tupleMap = i.tupleMap[:len(i.tupleMap)-1]
			}, "tuple slots"},
			{"oversized map", func(i *socioeconomicIncomeJSON) {
				i.tupleMap = append(i.tupleMap, 0)
			}, "tuple slots"},
			{"missing laws", func(i *socioeconomicIncomeJSON) {
				i.laws = nil
			}, "nonempty laws"},
			{"law count", func(i *socioeconomicIncomeJSON) {
				maximum := [...]int{512, 256, 64, 64, 64, 64}
				i.laws = slices.Repeat(i.laws[:1], maximum[branch]+1)
			}, "law count"},
			{"law ID", func(i *socioeconomicIncomeJSON) {
				i.tupleMap[1] = uint32(len(i.laws))
			}, "reference"},
			{"null in applicable slot", func(i *socioeconomicIncomeJSON) {
				i.tupleMap[1] = math.MaxUint16
			}, "reference"},
			{"law support limit", func(i *socioeconomicIncomeJSON) {
				i.laws[0].values = make([]uint32, 513)
			}, "512 points"},
			{"law weight limit", func(i *socioeconomicIncomeJSON) {
				i.laws[0].weights = make([]uint32, 513)
			}, "512 points"},
			{"empty law", func(i *socioeconomicIncomeJSON) {
				i.laws[0].values, i.laws[0].weights = nil, nil
			}, "equal positive length"},
			{"missing law weight", func(i *socioeconomicIncomeJSON) {
				i.laws[0].weights = i.laws[0].weights[:1]
			}, "equal positive length"},
			{"zero law weight", func(i *socioeconomicIncomeJSON) {
				i.laws[0].weights[0] = 0
			}, "zero weight"},
			{"law GCD", func(i *socioeconomicIncomeJSON) {
				i.laws[0].weights = []uint32{2, 4}
			}, "GCD"},
			{"support order", func(i *socioeconomicIncomeJSON) {
				i.laws[0].values = []uint32{1, 0}
			}, "increasing"},
		} {
			t.Run(fmt.Sprintf("%s/%s", socioeconomicBranchNames[branch], test.name), func(t *testing.T) {
				input := socioeconomicNumericFixture(9)
				test.change(&input.income[branch])
				_, err := compileSocioeconomicNumericModel(input, socioeconomicUSADivisions)
				if err != nil {
					if !strings.Contains(err.Error(), test.want) {
						t.Fatalf("expected %q rejection, got %v", test.want, err)
					}
					return
				}
				t.Fatal("expected invalid Income rejected, got success")
			})
		}
	}

}

// TestSocioeconomicNumericItalyBoundary rejects nine territories on both IT paths.
func TestSocioeconomicNumericItalyBoundary(t *testing.T) {

	input := socioeconomicNumericFixture(9)
	document := map[string]any{"schema": "SW-L2-D2B2-logical-v0", "companyCount": input.companyCount}
	ages := map[string]any{}
	for i, age := range input.ages {
		ages[socioeconomicAgeNames[i]] = map[string]any{"lowerBounds": age.lowerBounds, "compiled": age.compiled}
	}
	document["ages"] = ages
	for name, weights := range map[string][]uint32{
		"companyIndustry": input.companyIndustry, "education": input.education,
		"primaryActivity": input.primaryActivity, "employeeJobFamily": input.employeeJobFamily,
		"employeeJobLevel": input.employeeJobLevel, "selfEmployedJobFamily": input.selfEmployedJobFamily,
		"selfEmployedIndustry": input.selfEmployedIndustry,
	} {
		document[name] = map[string]any{"weights": weights}
	}
	counts := make([]any, len(input.employeeCount))
	for i, law := range input.employeeCount {
		counts[i] = map[string]any{"values": law.values, "weights": law.weights}
	}
	document["employeeCount"] = map[string]any{"laws": counts}
	document["employer"] = map[string]any{
		"cap": input.employer.cap, "industryFactors": input.employer.industryFactors,
		"sizeFactors": input.employer.sizeFactors,
	}
	income := map[string]any{}
	for branch, entry := range input.income {
		laws := make([]any, len(entry.laws))
		for i, law := range entry.laws {
			laws[i] = map[string]any{"values": law.values, "weights": law.weights}
		}
		income[socioeconomicBranchNames[branch]] = map[string]any{"tupleMap": entry.tupleMap, "laws": laws}
	}
	document["income"] = income

	// Deliberately mislabel this negative transport fixture; never publish it.
	t.Run("logical-v0 decoder", func(t *testing.T) {
		_, err := decodeSocioeconomicModel(strings.NewReader(string(internationalTestJSON(t, document))))
		if err != nil {
			if !strings.Contains(err.Error(), "education: expected 840 weights, got 378") {
				t.Fatalf("expected Italian dimension rejection, got %v", err)
			}
			return
		}
		t.Fatal("expected nine territories rejected by Italian decoder, got success")
	})
	t.Run("international decoder", func(t *testing.T) {
		document := internationalSocioeconomicFixture(t)
		document.Dimensions.Territory.Categories = document.Dimensions.Territory.Categories[:9]
		_, err := DecodeItalianSocioeconomicData(t.Context(), strings.NewReader(string(internationalTestJSON(t, document))))
		if err != nil {
			if !strings.Contains(err.Error(), "taxonomy binding") {
				t.Fatalf("expected Italian taxonomy rejection, got %v", err)
			}
			return
		}
		t.Fatal("expected nine territories rejected by Italian decoder, got success")
	})

}

// TestSocioeconomicNumericSupportBounds accepts the exact finite-law limits.
func TestSocioeconomicNumericSupportBounds(t *testing.T) {

	input := socioeconomicNumericFixture(9)
	for _, test := range []struct {
		law    *socioeconomicLawJSON
		points int
	}{
		{&input.employeeCount[0], 128},
		{&input.income[0].laws[0], 512},
	} {
		test.law.values = make([]uint32, test.points)
		test.law.weights = slices.Repeat([]uint32{math.MaxUint32}, test.points)
		test.law.weights[0] = 1
		for i := range test.law.values {
			test.law.values[i] = uint32(i)
		}
		test.law.values[test.points-1] = math.MaxUint32
	}
	model, err := compileSocioeconomicNumericModel(input, socioeconomicUSADivisions)
	if err != nil {
		t.Fatalf("expected exact support limits accepted, got %v", err)
	}
	for _, law := range []socioeconomicLaw{model.employeeCount[0], model.income[0].laws[0]} {
		points := len(law.values)
		want := 1 + uint64(points-1)*math.MaxUint32
		if law.values[points-1] != math.MaxUint32 || law.cumulative[points-1] != want {
			t.Fatalf("expected maximum support and exact total %d, got %d/%d",
				want, law.values[points-1], law.cumulative[points-1])
		}
	}

}

// TestSocioeconomicNumericTerritories verifies every technical grid cell.
func TestSocioeconomicNumericTerritories(t *testing.T) {

	for _, domain := range []socioeconomicTerritoryDomain{socioeconomicUSADivisions, socioeconomicItalyRegions} {
		t.Run(fmt.Sprint(domain), func(t *testing.T) {

			input := socioeconomicNumericFixture(int(domain))
			model, err := compileSocioeconomicNumericModel(input, domain)
			if err != nil {
				t.Fatalf("expected compiled technical model, got %v", err)
			}
			if model.territories != domain {
				t.Fatalf("expected territory domain %d, got %d", domain, model.territories)
			}
			for i, age := range model.ages {
				for offset, bucket := range age {
					if uint32(bucket) != input.ages[i].compiled[offset] {
						t.Fatalf("expected age bucket %d at %d/%d, got %d",
							input.ages[i].compiled[offset], i, offset, bucket)
					}
				}
			}
			if _, hasIdentity := any(model).(interface{ canonicalModel() ([]byte, error) }); hasIdentity {
				t.Fatal("expected numerical model without Italian serialization, got canonicalModel")
			}
			if _, hasIdentity := any(model).(interface{ prepareIdentity() error }); hasIdentity {
				t.Fatal("expected numerical model without national identity, got prepareIdentity")
			}

			tables := [...]socioeconomicWeights{
				model.companyIndustry, model.education, model.primaryActivity, model.employeeJobFamily,
				model.employeeJobLevel, model.selfEmployedJobFamily, model.selfEmployedIndustry,
			}
			weights := [...][]uint32{
				input.companyIndustry, input.education, input.primaryActivity, input.employeeJobFamily,
				input.employeeJobLevel, input.selfEmployedJobFamily, input.selfEmployedIndustry,
			}
			var rows, weightCount int
			for i, table := range tables {

				width := [...]int{14, 7, 6, 10, 4, 10, 14}
				if table.width != width[i] || len(table.cumulative) != len(weights[i]) {
					t.Fatalf("expected table %d width/length %d/%d, got %d/%d",
						i, width[i], len(weights[i]), table.width, len(table.cumulative))
				}
				rows += len(table.cumulative) / table.width
				weightCount += len(table.cumulative)
				var previous uint64
				for slot, cumulative := range table.cumulative {
					if slot%table.width == 0 {
						previous = 0
					}
					if cumulative-previous != uint64(weights[i][slot]) {
						t.Fatalf("expected exact weight %d at %d/%d, got %d",
							weights[i][slot], i, slot, cumulative-previous)
					}
					previous = cumulative
				}

			}

			for _, table := range []socioeconomicWeights{
				model.education, model.primaryActivity, model.employeeJobFamily, model.selfEmployedJobFamily,
			} {
				for prefix := range len(table.cumulative) / table.width / int(domain) {
					for territory := range int(domain) {
						row, err := model.territoryRow(prefix, territory)
						if err != nil {
							t.Fatalf("expected valid territorial row, got %v", err)
						}
						if row != prefix*int(domain)+territory || table.cumulative[row*table.width+1] != uint64(row+2) {
							t.Fatalf("expected distinct row %d/%d, got %d", prefix, territory, row)
						}
					}
				}
			}

			var slots, nulls int
			for branch, income := range model.income {

				slots += len(income.tupleMap)
				// Enumerate axes independently, with the last axis innermost.
				dimensions := []int{6, 7, int(domain)}
				if branch == 0 {
					dimensions = []int{int(domain), 10, 4, 14, 5}
				}
				if branch == 1 {
					dimensions = []int{6, int(domain), 10, 14}
				}
				count := 1
				for _, dimension := range dimensions {
					count *= dimension
				}
				if len(income.tupleMap) != count {
					t.Fatalf("expected branch %d length %d, got %d", branch, count, len(income.tupleMap))
				}
				for tuple, id := range income.tupleMap {

					ordinals := make([]int, len(dimensions))
					remainder := tuple
					for axis := len(dimensions) - 1; axis >= 0; axis-- {
						ordinals[axis] = remainder % dimensions[axis]
						remainder /= dimensions[axis]
					}
					territory := ordinals[len(ordinals)-1]
					if branch == 0 {
						territory = ordinals[0]
					}
					if branch == 1 {
						territory = ordinals[1]
					}
					if branch == 0 && ordinals[4] == 0 {
						nulls++
						if id != math.MaxUint16 {
							t.Fatalf("expected inapplicable sentinel at %d, got %d", tuple, id)
						}
						continue
					}
					if id != uint16(territory) {
						t.Fatalf("expected branch %d tuple %d law %d, got %d", branch, tuple, territory, id)
					}
					law := income.laws[id]
					if !slices.Equal(law.values, []uint32{0, uint32(branch*100 + territory + 1)}) ||
						!slices.Equal(law.cumulative, []uint64{1, 1 << 32}) {
						t.Fatalf("expected exact technical zero/positive law at %d/%d, got %+v", branch, tuple, law)
					}

				}

			}
			want := [4]int{1115, 6496, 34272, 5040}
			if domain == socioeconomicItalyRegions {
				want = [4]int{1951, 12194, 76160, 11200}
				italian, err := compileSocioeconomicModel(input)
				if err != nil {
					t.Fatalf("expected Italian wrapper, got %v", err)
				}
				if !reflect.DeepEqual(italian.socioeconomicNumericModel, *model) || !italian.initialized {
					t.Fatal("expected identical Italian numerical state and acquired model, got divergence")
				}
			}
			if got := [4]int{rows, weightCount, slots, nulls}; got != want {
				t.Fatalf("expected rows/weights/slots/nulls %v, got %v", want, got)
			}
			repeated, err := compileSocioeconomicNumericModel(input, domain)
			if err != nil {
				t.Fatalf("expected repeat compilation, got %v", err)
			}
			if !reflect.DeepEqual(model, repeated) {
				t.Fatal("expected deterministic complete numerical state, got divergence")
			}
			t.Logf("territories=%d rows=%d weights=%d slots=%d applicable=%d inapplicable=%d",
				domain, rows, weightCount, slots, slots-nulls, nulls)

		})
	}

}

// TestSocioeconomicNumericTerritoryBounds rejects aliases and integer overflow.
func TestSocioeconomicNumericTerritoryBounds(t *testing.T) {

	for _, domain := range []socioeconomicTerritoryDomain{0, 9, 10, 20, 255} {
		model := &socioeconomicNumericModel{territories: domain}
		for _, ordinals := range [][2]int{{0, -1}, {0, int(domain)}, {0, 20}, {-1, 0}, {math.MaxInt, 0}} {
			t.Run(fmt.Sprintf("%d/%v", domain, ordinals), func(t *testing.T) {
				_, err := model.territoryRow(ordinals[0], ordinals[1])
				if err != nil {
					return
				}
				t.Fatal("expected invalid domain, ordinal or overflow rejected, got success")
			})
		}
	}

}

// socioeconomicNumericFixture uses artificial ordinals and masses, with no
// statistical source, country package, Income currency or runtime approval.
func socioeconomicNumericFixture(territories int) *socioeconomicModelJSON {

	input := &socioeconomicModelJSON{companyCount: 1}
	for i := range input.ages {

		count := 6
		if i == 1 {
			count = 8
		}
		age := &input.ages[i]
		age.lowerBounds = make([]uint32, count)
		for bucket := range count {
			age.lowerBounds[bucket] = uint32(21 + bucket)
		}
		age.compiled = make([]uint32, 55)
		for offset := range age.compiled {
			age.compiled[offset] = uint32(min(offset, count-1))
		}

	}
	for _, table := range []struct {
		out   *[]uint32
		rows  int
		width int
	}{
		{&input.companyIndustry, 1, 14}, {&input.education, 6 * territories, 7},
		{&input.primaryActivity, 8 * 7 * territories, 6}, {&input.employeeJobFamily, 7 * territories, 10},
		{&input.employeeJobLevel, 6 * 7 * 10, 4}, {&input.selfEmployedJobFamily, 7 * territories, 10},
		{&input.selfEmployedIndustry, 10, 14},
	} {
		*table.out = make([]uint32, table.rows*table.width)
		for row := range table.rows {
			(*table.out)[row*table.width] = 1
			(*table.out)[row*table.width+1] = uint32(row + 1)
		}
	}
	// Only Industry 0 has positive Company mass; all other law slots still exist.
	clear(input.companyIndustry[1:])
	input.employeeCount = make([]socioeconomicLawJSON, 14)
	for i := range input.employeeCount {
		input.employeeCount[i] = socioeconomicLawJSON{values: []uint32{0, 1}, weights: []uint32{1, 1}}
	}
	input.employer.cap = 1
	for i := range input.employer.industryFactors {
		input.employer.industryFactors[i] = 1
	}
	for i := range input.employer.sizeFactors {
		input.employer.sizeFactors[i] = 1
	}
	for branch := range input.income {

		slots := 6 * 7 * territories
		if branch == 0 {
			slots = territories * 10 * 4 * 14 * 5
		}
		if branch == 1 {
			slots = 6 * territories * 10 * 14
		}
		income := &input.income[branch]
		income.tupleMap = make([]uint32, slots)
		for slot := range income.tupleMap {
			territory := slot % territories
			if branch == 0 {
				territory = slot / (10 * 4 * 14 * 5)
			}
			if branch == 1 {
				territory = (slot / (10 * 14)) % territories
			}
			income.tupleMap[slot] = uint32(territory)
			if branch == 0 && slot%5 == 0 {
				income.tupleMap[slot] = math.MaxUint16
			}
		}
		income.laws = make([]socioeconomicLawJSON, territories)
		for territory := range income.laws {
			income.laws[territory] = socioeconomicLawJSON{
				values: []uint32{0, uint32(branch*100 + territory + 1)}, weights: []uint32{1, math.MaxUint32},
			}
		}

	}

	return input
}
