// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"math"
	"os"
	"strings"
	"testing"
)

// TestInternationalBaseInvalid exercises semantic constraints on the real base.
func TestInternationalBaseInvalid(t *testing.T) {
	base, err := ConvertItalianBaseData(t.Context(), os.DirFS("data/it-2026a"))
	if err != nil {
		t.Fatalf("expected base conversion, got %v", err)
	}
	encoded, err := base.CanonicalJSON()
	if err != nil {
		t.Fatalf("expected base JSON, got %v", err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*internationalBase)
	}{
		{"schema", func(d *internationalBase) {
			d.Schema = "sw-base-data-v2"
		}},
		{"profile", func(d *internationalBase) {
			d.Profile = "other"
		}},
		{"country", func(d *internationalBase) {
			d.Country = "US"
		}},
		{"anchor", func(d *internationalBase) {
			d.AnchorDate = "2026-02-30"
		}},
		{"release token", func(d *internationalBase) {
			d.Release = " bad"
		}},
		{"gender identity", func(d *internationalBase) {
			d.Genders.SemanticID = "unknown"
		}},
		{"gender order", func(d *internationalBase) {
			d.Genders.Categories[0], d.Genders.Categories[1] = d.Genders.Categories[1], d.Genders.Categories[0]
		}},
		{"age coverage", func(d *internationalBase) {
			d.Ages = d.Ages[1:]
		}},
		{"age order", func(d *internationalBase) {
			d.Ages[0], d.Ages[1] = d.Ages[1], d.Ages[0]
		}},
		{"age zero", func(d *internationalBase) {
			d.Ages[0].Weight = 0
		}},
		{"age overflow", func(d *internationalBase) {
			d.Ages[0].Weight = math.MaxUint64
		}},
		{"cohort gap", func(d *internationalBase) {
			d.Cohorts[0].FromYear = 1951
		}},
		{"cohort overlap", func(d *internationalBase) {
			d.Cohorts[0].ToYear = 1965
		}},
		{"group order", func(d *internationalBase) {
			d.FirstNames[0], d.FirstNames[1] = d.FirstNames[1], d.FirstNames[0]
		}},
		{"group reference", func(d *internationalBase) {
			d.FirstNames[0].Cohort = "unknown"
		}},
		{"missing atom", func(d *internationalBase) {
			d.FirstNames[0].Entries[0].ASCIIAtom = nil
		}},
		{"atom grammar", func(d *internationalBase) {
			s := "a--b"
			d.FirstNames[0].Entries[0].ASCIIAtom = &s
		}},
		{"name duplicate", func(d *internationalBase) {
			d.FirstNames[0].Entries[1] = d.FirstNames[0].Entries[0]
		}},
		{"name quota", func(d *internationalBase) {
			d.FirstNames[0].Entries[0].Weight = 3
		}},
		{"name decomposed", func(d *internationalBase) {
			d.FirstNames[0].Entries[0].Display = "A\u0301"
		}},
		{"surname order", func(d *internationalBase) {
			d.LastNames[0], d.LastNames[1] = d.LastNames[1], d.LastNames[0]
		}},
		{"surname missing", func(d *internationalBase) {
			d.LastNames = d.LastNames[:599]
		}},
		{"place system", func(d *internationalBase) {
			d.Geography.PlaceCodeSystem = "unknown"
		}},
		{"territory system", func(d *internationalBase) {
			d.Geography.TerritorySystem = "unknown"
		}},
		{"place missing", func(d *internationalBase) {
			d.Geography.Places = d.Geography.Places[1:]
		}},
		{"place order", func(d *internationalBase) {
			d.Geography.Places[0], d.Geography.Places[1] = d.Geography.Places[1], d.Geography.Places[0]
		}},
		{"place code", func(d *internationalBase) {
			d.Geography.Places[0].ID = "1001"
		}},
		{"place duplicate", func(d *internationalBase) {
			d.Geography.Places[1].ID = d.Geography.Places[0].ID
		}},
		{"territory reference", func(d *internationalBase) {
			d.Geography.Places[0].Territory = "unknown"
		}},
		{"population zero", func(d *internationalBase) {
			d.Geography.Places[0].Population = 0
		}},
		{"population overflow", func(d *internationalBase) {
			d.Geography.Places[0].Population = math.MaxUint64
		}},
		{"province missing", func(d *internationalBase) {
			d.Geography.Places[0].Administrative = nil
		}},
		{"province code", func(d *internationalBase) {
			s := "To"
			d.Geography.Places[0].Administrative[0].Code = &s
		}},
		{"province duplicate", func(d *internationalBase) {
			p := &d.Geography.Places[0]
			p.Administrative = append(p.Administrative, p.Administrative[0])
		}},
		{"province inconsistent", func(d *internationalBase) {
			a, b := "A", "B"
			d.Geography.Places[0].Administrative[0].Name = &a
			d.Geography.Places[1].Administrative[0].Name = &b
		}},
		{"postal missing", func(d *internationalBase) {
			delete(d.Geography.PostalCodes, d.Geography.Places[0].ID)
		}},
		{"postal unknown", func(d *internationalBase) {
			d.Geography.PostalCodes["unknown"] = []string{"12345"}
		}},
		{"postal empty", func(d *internationalBase) {
			d.Geography.PostalCodes[d.Geography.Places[0].ID] = []string{}
		}},
		{"postal order", func(d *internationalBase) {
			d.Geography.PostalCodes[d.Geography.Places[0].ID] = []string{"10011", "00001"}
		}},
		{"postal duplicate", func(d *internationalBase) {
			d.Geography.PostalCodes[d.Geography.Places[0].ID] = []string{"10011", "10011"}
		}},
		{"postal width", func(d *internationalBase) {
			d.Geography.PostalCodes[d.Geography.Places[0].ID] = []string{"1234"}
		}},
		{"street kind", func(d *internationalBase) {
			d.StreetNames.Kind = "full-name"
		}},
		{"street prefix", func(d *internationalBase) {
			d.StreetNames.Entries[0].Text = "Via Roma"
		}},
		{"street order", func(d *internationalBase) {
			d.StreetNames.Entries[0], d.StreetNames.Entries[1] = d.StreetNames.Entries[1], d.StreetNames.Entries[0]
		}},
		{"street quota", func(d *internationalBase) {
			d.StreetNames.Entries[0].Weight = 3
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var document internationalBase
			err := decodeInternationalJSON(t.Context(), bytes.NewReader(encoded), &document)
			if err != nil {
				t.Fatalf("expected original base, got %v", err)
			}
			test.mutate(&document)
			input, err := internationalCanonical(document)
			if err != nil {
				t.Fatalf("expected mutated JSON, got %v", err)
			}
			_, err = DecodeItalianBaseData(t.Context(), bytes.NewReader(input))
			if err != nil {
				return
			}
			t.Fatal("expected invalid Italian base rejected, got success")
		})
	}
}

// TestInternationalSocioeconomicInvalid verifies profile restrictions and the
// old decoder's narrowing/cardinality preconditions before compilation.
func TestInternationalSocioeconomicInvalid(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*internationalSocioeconomic)
	}{
		{"schema", func(d *internationalSocioeconomic) {
			d.Schema = "SW-L2-D2B2-logical-v0"
		}},
		{"country", func(d *internationalSocioeconomic) {
			d.Country = "US"
		}},
		{"source absent", func(d *internationalSocioeconomic) {
			d.SourceModel = nil
		}},
		{"source hash", func(d *internationalSocioeconomic) {
			d.SourceModel.ByteSHA256 = strings.Repeat("0", 64)
		}},
		{"age domain", func(d *internationalSocioeconomic) {
			d.AgeDomain.Maximum = math.MaxUint32
		}},
		{"company count", func(d *internationalSocioeconomic) {
			d.CompanyCount = 104856
		}},
		{"company zero", func(d *internationalSocioeconomic) {
			d.CompanyCount = 0
		}},
		{"currency", func(d *internationalSocioeconomic) {
			d.IncomeUnit.Currency = "USD"
		}},
		{"precision", func(d *internationalSocioeconomic) {
			d.IncomeUnit.DecimalPlaces = 2
		}},
		{"basis", func(d *internationalSocioeconomic) {
			d.IncomeUnit.Basis = "net"
		}},
		{"taxonomy id", func(d *internationalSocioeconomic) {
			d.Dimensions.Education.SemanticID = "unknown"
		}},
		{"taxonomy order", func(d *internationalSocioeconomic) {
			c := d.Dimensions.Industry.Categories
			c[0], c[1] = c[1], c[0]
		}},
		{"category duplicate", func(d *internationalSocioeconomic) {
			d.Dimensions.Territory.Categories[0] = d.Dimensions.Territory.Categories[1]
		}},
		{"branch binding", func(d *internationalSocioeconomic) {
			d.Dimensions.Activity.Categories[0].Branch = "student"
		}},
		{"size threshold", func(d *internationalSocioeconomic) {
			d.Dimensions.SizeBand.Categories[1].MinimumEmployeeCount = 2
		}},
		{"partition order", func(d *internationalSocioeconomic) {
			d.AgePartitions.Education.LowerBounds[1] = 21
		}},
		{"partition width", func(d *internationalSocioeconomic) {
			d.AgePartitions.StudentIncome.LowerBounds[5] = 256
		}},
		{"partition count", func(d *internationalSocioeconomic) {
			d.AgePartitions.EmployeeLevel.LowerBounds = append(d.AgePartitions.EmployeeLevel.LowerBounds, 70)
		}},
		{"axes order", func(d *internationalSocioeconomic) {
			a := d.Models.Education.Axes
			a[0], a[1] = a[1], a[0]
		}},
		{"outcome", func(d *internationalSocioeconomic) {
			d.Models.Education.Outcome = "activity"
		}},
		{"categorical missing", func(d *internationalSocioeconomic) {
			d.Models.Education.Weights = d.Models.Education.Weights[1:]
		}},
		{"categorical gcd", func(d *internationalSocioeconomic) {
			for i := range 7 {
				d.Models.Education.Weights[i] = 2
			}
		}},
		{"categorical zero", func(d *internationalSocioeconomic) {
			for i := range 7 {
				d.Models.Education.Weights[i] = 0
			}
		}},
		{"count axes", func(d *internationalSocioeconomic) {
			d.Models.EmployeeCount.Axes = []string{}
		}},
		{"count missing", func(d *internationalSocioeconomic) {
			d.Models.EmployeeCount.Laws = d.Models.EmployeeCount.Laws[1:]
		}},
		{"count points", func(d *internationalSocioeconomic) {
			d.Models.EmployeeCount.Laws[0].Values = make([]uint32, 129)
		}},
		{"count duplicate support", func(d *internationalSocioeconomic) {
			d.Models.EmployeeCount.Laws[0].Values = []uint32{1, 1}
		}},
		{"count zero workforce", func(d *internationalSocioeconomic) {
			for i := range d.Models.EmployeeCount.Laws {
				d.Models.EmployeeCount.Laws[i] = internationalLaw{[]uint32{0}, []uint32{1}}
			}
		}},
		{"cap narrowing", func(d *internationalSocioeconomic) {
			d.Models.Employer.WorkforceCap = 65536 + 4096
		}},
		{"cap profile", func(d *internationalSocioeconomic) {
			d.Models.Employer.WorkforceCap = 4095
		}},
		{"factor narrowing", func(d *internationalSocioeconomic) {
			d.Models.Employer.IndustryFactors.Values[0] = 65537
		}},
		{"factor maximum", func(d *internationalSocioeconomic) {
			d.Models.Employer.IndustryFactors.Values[0] = 4096
		}},
		{"factor zero", func(d *internationalSocioeconomic) {
			d.Models.Employer.SizeFactors.Values[0] = 0
		}},
		{"factor count", func(d *internationalSocioeconomic) {
			d.Models.Employer.SizeFactors.Values = d.Models.Employer.SizeFactors.Values[1:]
		}},
		{"factor axes", func(d *internationalSocioeconomic) {
			d.Models.Employer.SizeFactors.Axes[1] = "sizeBand"
		}},
		{"income axes", func(d *internationalSocioeconomic) {
			d.Models.Income.Employee.Axes[0] = "age.education"
		}},
		{"income map count", func(d *internationalSocioeconomic) {
			d.Models.Income.Employee.TupleMap = d.Models.Income.Employee.TupleMap[1:]
		}},
		{"income null positive", func(d *internationalSocioeconomic) {
			d.Models.Income.Employee.TupleMap[1] = nil
		}},
		{"income null elsewhere", func(d *internationalSocioeconomic) {
			d.Models.Income.Student.TupleMap[0] = nil
		}},
		{"income number inapplicable", func(d *internationalSocioeconomic) {
			id := uint32(65535)
			d.Models.Income.Employee.TupleMap[0] = &id
		}},
		{"income sentinel reachable", func(d *internationalSocioeconomic) {
			id := uint32(65535)
			d.Models.Income.Employee.TupleMap[1] = &id
		}},
		{"income reference", func(d *internationalSocioeconomic) {
			id := uint32(math.MaxUint32)
			d.Models.Income.Employee.TupleMap[1] = &id
		}},
		{"income first occurrence", func(d *internationalSocioeconomic) {
			id := uint32(1)
			d.Models.Income.Student.TupleMap[0] = &id
			law := internationalLaw{[]uint32{2}, []uint32{1}}
			d.Models.Income.Student.Laws = append(d.Models.Income.Student.Laws, law)
		}},
		{"income duplicate law", func(d *internationalSocioeconomic) {
			d.Models.Income.Employee.Laws = append(d.Models.Income.Employee.Laws, d.Models.Income.Employee.Laws[0])
		}},
		{"income unreferenced", func(d *internationalSocioeconomic) {
			law := internationalLaw{[]uint32{2}, []uint32{1}}
			d.Models.Income.Student.Laws = append(d.Models.Income.Student.Laws, law)
		}},
		{"income empty", func(d *internationalSocioeconomic) {
			d.Models.Income.Student.Laws = nil
		}},
		{"income law count", func(d *internationalSocioeconomic) {
			d.Models.Income.Student.Laws = make([]internationalLaw, 65)
		}},
		{"income self law count", func(d *internationalSocioeconomic) {
			d.Models.Income.SelfEmployed.Laws = make([]internationalLaw, 257)
		}},
		{"income employee law count", func(d *internationalSocioeconomic) {
			d.Models.Income.Employee.Laws = make([]internationalLaw, 513)
		}},
		{"income points", func(d *internationalSocioeconomic) {
			d.Models.Income.Student.Laws[0].Values = make([]uint32, 513)
		}},
		{"income supports order", func(d *internationalSocioeconomic) {
			d.Models.Income.Student.Laws[0].Values = []uint32{1, 0, math.MaxUint32}
		}},
		{"income support weight count", func(d *internationalSocioeconomic) {
			d.Models.Income.Student.Laws[0].Weights = []uint32{1}
		}},
		{"income weight zero", func(d *internationalSocioeconomic) {
			d.Models.Income.Student.Laws[0].Weights[0] = 0
		}},
		{"income weight gcd", func(d *internationalSocioeconomic) {
			d.Models.Income.Student.Laws[0].Weights = []uint32{2, 2, 2}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := internationalSocioeconomicFixture(t)
			test.mutate(&document)
			encoded, err := internationalCanonical(document)
			if err != nil {
				t.Fatalf("expected mutated model JSON, got %v", err)
			}
			_, err = DecodeItalianSocioeconomicData(t.Context(), bytes.NewReader(encoded))
			if err != nil {
				return
			}
			t.Fatal("expected invalid socioeconomic document rejected, got success")
		})
	}
}

func internationalSocioeconomicFixture(t *testing.T) internationalSocioeconomic {
	t.Helper()
	d := internationalSocioeconomic{
		Schema: "sw-socioeconomic-data-v1", Release: "artificial-test", Profile: "it-2026a-socioeconomic-v1",
		Country: "IT", AnchorDate: PopulationAnchorDate,
		SourceModel: &internationalSourceModel{"SW-L2-D2B2-logical-v0", italianSourceModelSHA256},
		AgeDomain:   internationalAgeDomain{21, 75}, Dimensions: italianInternationalDimensions(), CompanyCount: 65536,
		IncomeUnit: internationalIncomeUnit{"EUR", 0, "year", "gross-personal-monetary-annualized"},
	}
	bounds := italianInternationalPartitionBounds()
	for i, partition := range d.AgePartitions.ordered() {
		partition.LowerBounds = bounds[i]
	}
	axes := [][]string{
		{}, {"age.education", "territory"}, {"age.activity", "education", "territory"},
		{"education", "territory"}, {"age.employeeLevel", "education", "jobFamily"},
		{"education", "territory"}, {"jobFamily"},
	}
	outcomes := []string{"industry", "education", "activity", "jobFamily", "jobLevel", "jobFamily", "industry"}
	counts := []int{14, 840, 6720, 1400, 1680, 1400, 140}
	for i, categorical := range d.Models.categorical() {
		*categorical = internationalCategorical{Axes: axes[i], Outcome: outcomes[i], Weights: make([]uint32, counts[i])}
		for j := range categorical.Weights {
			categorical.Weights[j] = 1
		}
	}
	d.Models.EmployeeCount.Axes = []string{"industry"}
	for range 14 {
		law := internationalLaw{[]uint32{0, 1}, []uint32{1, 1}}
		d.Models.EmployeeCount.Laws = append(d.Models.EmployeeCount.Laws, law)
	}
	d.Models.Employer.WorkforceCap = 4096
	d.Models.Employer.IndustryFactors = internationalFactorMatrix{
		[]string{"jobFamily", "industry"}, make([]uint32, 140),
	}
	d.Models.Employer.SizeFactors = internationalFactorMatrix{
		[]string{"jobLevel", "positiveSizeBand"}, make([]uint32, 16),
	}
	for _, factors := range [][]uint32{d.Models.Employer.IndustryFactors.Values, d.Models.Employer.SizeFactors.Values} {
		for i := range factors {
			factors[i] = 1
		}
	}
	incomeAxes := [][]string{
		{"territory", "jobFamily", "jobLevel", "industry", "sizeBand"},
		{"age.selfEmployedIncome", "territory", "jobFamily", "industry"},
		{"age.unemployedIncome", "education", "territory"}, {"age.studentIncome", "education", "territory"},
		{"age.retiredIncome", "education", "territory"}, {"age.inactiveIncome", "education", "territory"},
	}
	for i, income := range d.Models.Income.ordered() {
		counts := []int{56000, 16800, 840, 840, 840, 840}
		income.Axes = incomeAxes[i]
		income.Laws = []internationalLaw{{[]uint32{0, 1, math.MaxUint32}, []uint32{1, 1, 1}}}
		income.TupleMap = make([]*uint32, counts[i])
		for j := range income.TupleMap {
			if i == 0 && j%5 == 0 {
				continue
			}
			id := uint32(0)
			income.TupleMap[j] = &id
		}
	}
	return d
}
