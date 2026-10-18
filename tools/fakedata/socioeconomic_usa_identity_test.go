// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

// TestUSASocioeconomicIdentityMismatch rejects divergent compiled meanings or
// numbers before publishing any numerical digest or local fingerprint.
func TestUSASocioeconomicIdentityMismatch(t *testing.T) {

	for _, test := range []struct {
		name   string
		mutate func(*usaSocioeconomicModel)
	}{
		{"country", func(m *usaSocioeconomicModel) { m.interpretation.Country = "IT" }},
		{"profile", func(m *usaSocioeconomicModel) { m.interpretation.Profile = "unknown" }},
		{"anchor", func(m *usaSocioeconomicModel) { m.interpretation.AnchorDate = "2026-01-02" }},
		{"age domain", func(m *usaSocioeconomicModel) { m.interpretation.AgeDomain.Maximum = 76 }},
		{"taxonomy", func(m *usaSocioeconomicModel) { m.interpretation.Dimensions.Education.SemanticID = "unknown" }},
		{"unit", func(m *usaSocioeconomicModel) { m.interpretation.IncomeUnit.Currency = "EUR" }},
		{"territories", func(m *usaSocioeconomicModel) { m.numeric.territories = 20 }},
		{"company count", func(m *usaSocioeconomicModel) { m.numeric.companyCount++ }},
		{"partition", func(m *usaSocioeconomicModel) { m.numeric.ages[0][3] = 1 }},
		{"categorical", func(m *usaSocioeconomicModel) { m.numeric.education.cumulative[0]++ }},
		{"count support", func(m *usaSocioeconomicModel) { m.numeric.employeeCount[0].values[1]++ }},
		{"employer", func(m *usaSocioeconomicModel) { m.numeric.employer.cap++ }},
		{"income support", func(m *usaSocioeconomicModel) { m.numeric.income[0].laws[0].values[1]++ }},
		{"income map", func(m *usaSocioeconomicModel) { m.numeric.income[0].tupleMap[1] = 1 }},
		{"protocol", func(m *usaSocioeconomicModel) { m.protocol = "socioeconomic-v1" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := usaSocioeconomicFixture()
			data := mustUSASocioeconomicData(t, document)
			m := &usaSocioeconomicModel{
				numeric: data.model.numeric, interpretation: data.model.interpretation, protocol: data.model.protocol,
			}
			test.mutate(m)
			err := m.prepareIdentity(usaSocioeconomicProjectionForTest(t, document))
			if err != nil {
				if m.digest != ([32]byte{}) || m.fingerprints != ([15][32]byte{}) || m.initialized {
					t.Fatal("expected no published identity after mismatch, got initialized or nonzero state")
				}
				return
			}
			t.Fatal("expected inconsistent compiled USA interpretation rejected, got success")
		})
	}
	t.Run("projection digest", func(t *testing.T) {
		document := usaSocioeconomicFixture()
		data := mustUSASocioeconomicData(t, document)
		projection := usaSocioeconomicProjectionForTest(t, document)
		changed := bytes.Replace(projection, []byte(`"companyCount":65536`), []byte(`"companyCount":65537`), 1)
		if sha256.Sum256(projection) == sha256.Sum256(changed) {
			t.Fatal("expected mismatched numerical projection digest, got equality")
		}
		err := data.model.prepareIdentity(changed)
		if err != nil {
			return
		}
		t.Fatal("expected mismatched numerical projection rejected, got success")
	})

}

// TestUSASocioeconomicInterpretationIdentity checks every H.2 meaning even when
// that semantic revision is outside the admitted v1 national binding.
func TestUSASocioeconomicInterpretationIdentity(t *testing.T) {

	baseline := mustUSASocioeconomicData(t, usaSocioeconomicFixture())
	for _, test := range []struct {
		name   string
		mutate func(*internationalSocioeconomic)
	}{
		{"country", func(d *internationalSocioeconomic) { d.Country = "CA" }},
		{"profile", func(d *internationalSocioeconomic) { d.Profile = "us-2026b-socioeconomic-v1" }},
		{"anchor", func(d *internationalSocioeconomic) { d.AnchorDate = "2026-01-02" }},
		{"age minimum", func(d *internationalSocioeconomic) { d.AgeDomain.Minimum-- }},
		{"age maximum", func(d *internationalSocioeconomic) { d.AgeDomain.Maximum++ }},
		{"territory meaning", func(d *internationalSocioeconomic) { d.Dimensions.Territory.SemanticID = "changed" }},
		{"territory code", func(d *internationalSocioeconomic) { d.Dimensions.Territory.Categories[0].Code = "changed" }},
		{"education meaning", func(d *internationalSocioeconomic) { d.Dimensions.Education.SemanticID = "changed" }},
		{"industry meaning", func(d *internationalSocioeconomic) { d.Dimensions.Industry.SemanticID = "changed" }},
		{"family meaning", func(d *internationalSocioeconomic) { d.Dimensions.JobFamily.SemanticID = "changed" }},
		{"level meaning", func(d *internationalSocioeconomic) { d.Dimensions.JobLevel.SemanticID = "changed" }},
		{"activity meaning", func(d *internationalSocioeconomic) { d.Dimensions.Activity.SemanticID = "changed" }},
		{"branch meaning", func(d *internationalSocioeconomic) { d.Dimensions.Activity.Categories[0].Branch = "student" }},
		{"size meaning", func(d *internationalSocioeconomic) { d.Dimensions.SizeBand.SemanticID = "changed" }},
		{"size threshold", func(d *internationalSocioeconomic) {
			d.Dimensions.SizeBand.Categories[2].MinimumEmployeeCount++
		}},
		{"company count", func(d *internationalSocioeconomic) { d.CompanyCount++ }},
		{"currency", func(d *internationalSocioeconomic) { d.IncomeUnit.Currency = "EUR" }},
		{"precision", func(d *internationalSocioeconomic) { d.IncomeUnit.DecimalPlaces++ }},
		{"period", func(d *internationalSocioeconomic) { d.IncomeUnit.Period = "month" }},
		{"basis", func(d *internationalSocioeconomic) { d.IncomeUnit.Basis = "net" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := usaSocioeconomicFixture()
			test.mutate(&document)
			if sha256.Sum256(usaSocioeconomicProjectionForTest(t, document)) == baseline.model.digest {
				t.Fatal("expected changed semantic projection identity, got equality")
			}
			_, err := decodeUSASocioeconomicData(t.Context(), bytes.NewReader(internationalTestJSON(t, document)))
			if err != nil {
				return
			}
			t.Fatal("expected unauthorized semantic binding rejected, got success")
		})
	}
	for partition := range 8 {
		t.Run(fmt.Sprintf("partition %d", partition), func(t *testing.T) {
			document := usaSocioeconomicFixture()
			document.AgePartitions.ordered()[partition].LowerBounds[1]++
			if sha256.Sum256(usaSocioeconomicProjectionForTest(t, document)) == baseline.model.digest {
				t.Fatal("expected changed owned age partition identity, got equality")
			}
		})
	}

}

// TestUSASocioeconomicLocalIdentity proves each numerical revision owns just
// its local fingerprint; no global digest propagates to unrelated nodes.
func TestUSASocioeconomicLocalIdentity(t *testing.T) {

	baseline := mustUSASocioeconomicData(t, usaSocioeconomicFixture())
	for node, path := range usaSocioeconomicPaths {
		t.Run(path, func(t *testing.T) {
			document := usaSocioeconomicFixture()
			switch node {
			case socioeconomicCompanyEmployeeCount:
				document.Models.EmployeeCount.Laws[0].Weights[0]++
			case socioeconomicEmployer:
				document.Models.Employer.WorkforceCap++
			case socioeconomicIncomeEmployee, socioeconomicIncomeSelfEmployed, socioeconomicIncomeUnemployed,
				socioeconomicIncomeStudent, socioeconomicIncomeRetired, socioeconomicIncomeInactive:
				document.Models.Income.ordered()[node-socioeconomicIncomeEmployee].Laws[0].Weights[0]++
			default:
				index := map[int]int{0: 0, 2: 1, 3: 2, 4: 3, 5: 4, 7: 5, 8: 6}[node]
				document.Models.categorical()[index].Weights[1]++
			}
			changed := mustUSASocioeconomicData(t, document)
			if changed.model.digest == baseline.model.digest {
				t.Fatal("expected changed whole numerical identity, got equality")
			}
			for other := range usaSocioeconomicPaths {
				isEqual := changed.model.fingerprints[other] == baseline.model.fingerprints[other]
				if isEqual != (other != node) {
					t.Fatalf("expected only local fingerprint %d changed, got equality=%v at %d", node, isEqual, other)
				}
			}
		})
	}

}

// TestUSASocioeconomicUSD rejects invalid support addresses and unpublished
// states without interpreting missing or inapplicable Income as zero USD.
func TestUSASocioeconomicUSD(t *testing.T) {

	data := mustUSASocioeconomicData(t, usaSocioeconomicFixture())
	for _, test := range []struct {
		name               string
		model              *usaSocioeconomicModel
		branch, law, point int
	}{
		{"nil", nil, 0, 0, 0}, {"zero", &usaSocioeconomicModel{}, 0, 0, 0},
		{"negative branch", data.model, -1, 0, 0}, {"branch bound", data.model, 6, 0, 0},
		{"negative law", data.model, 0, -1, 0}, {"law bound", data.model, 0, 9, 0},
		{"inapplicable sentinel", data.model, 0, 65535, 0},
		{"negative point", data.model, 0, 0, -1}, {"point bound", data.model, 0, 0, 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			value, err := test.model.incomeSupport(test.branch, test.law, test.point)
			if err != nil {
				if value != 0 {
					t.Fatalf("expected zero failure result, got %d", value)
				}
				return
			}
			t.Fatal("expected invalid USD support address rejected, got success")
		})
	}
	t.Run("oversized document", func(t *testing.T) {
		input := internationalTestJSON(t, usaSocioeconomicFixture())
		input = append(input, strings.Repeat(" ", maxInternationalDocumentBytes)...)
		_, err := decodeUSASocioeconomicData(t.Context(), bytes.NewReader(input))
		if err != nil {
			return
		}
		t.Fatal("expected oversized USA document rejected, got success")
	})
	t.Run("nil document", func(t *testing.T) {
		_, err := prepareUSASocioeconomic(nil)
		if err != nil {
			return
		}
		t.Fatal("expected nil national document rejected, got success")
	})

}
